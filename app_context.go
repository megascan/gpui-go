package gpui

import (
	"fmt"
	"reflect"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// Context is the app context, with specialized behavior for the given entity. It embeds *App,
// upstream's Deref<Target = App>.
type Context[T any] struct {
	*App
	entityState WeakEntity[T]
}

func newContext[T any](app *App, entityState WeakEntity[T]) *Context[T] {
	return &Context[T]{App: app, entityState: entityState}
}

// EntityID returns the entity id of the entity backing this context.
func (cx *Context[T]) EntityID() EntityID {
	return cx.entityState.entityID
}

// Entity returns a handle to the entity belonging to this context. The caller releases it.
func (cx *Context[T]) Entity() Entity[T] {
	entity, ok := cx.WeakEntity().Upgrade()
	if !ok {
		panic("The entity must be alive if we have a entity context")
	}
	return entity
}

// WeakEntity returns a weak handle to the entity belonging to this context.
func (cx *Context[T]) WeakEntity() WeakEntity[T] {
	return cx.entityState
}

// Observe arranges for onNotify to be called whenever Notify is called with the given entity.
func (cx *Context[T]) Observe[W any](entity Entity[W], onNotify func(this *T, entity Entity[W], cx *Context[T])) Subscription {
	observer := cx.WeakEntity()
	return observeInternal(cx.App, entity, func(entity Entity[W], cx *App) bool {
		return invokeObserver(observer, entity, cx, onNotify)
	})
}

// ObserveSelf observes changes to this context's own entity.
func (cx *Context[T]) ObserveSelf(onEvent func(this *T, cx *Context[T])) Subscription {
	this := cx.Entity()
	defer this.Release()
	return cx.App.Observe(this, func(this Entity[T], cx *App) {
		this.Update(cx, func(this *T, cx *Context[T]) { onEvent(this, cx) })
	})
}

// Subscribe subscribes to events of type E emitted by another entity.
func (cx *Context[T]) Subscribe[T2, E any](entity Entity[T2], onEvent func(this *T, entity Entity[T2], event *E, cx *Context[T])) Subscription {
	subscriber := cx.WeakEntity()
	return subscribeInternal(cx.App, entity, func(entity Entity[T2], event *E, cx *App) bool {
		return invokeSubscriber(subscriber, entity, event, cx, onEvent)
	})
}

// SubscribeSelf subscribes to events of type E emitted by this context's own entity.
func (cx *Context[T]) SubscribeSelf[E any](onEvent func(this *T, event *E, cx *Context[T])) Subscription {
	this := cx.Entity()
	defer this.Release()
	return cx.App.Subscribe(this, func(this Entity[T], event *E, cx *App) {
		this.Update(cx, func(this *T, cx *Context[T]) { onEvent(this, event, cx) })
	})
}

// OnRelease registers a callback to be invoked when GPUI releases this entity.
func (cx *Context[T]) OnRelease(onRelease func(this *T, cx *App)) Subscription {
	return cx.App.releaseListeners.insert(cx.entityState.entityID, func(this any, cx *App) {
		state, ok := this.(*T)
		if !ok {
			panic("invalid entity type")
		}
		onRelease(state, cx)
	})
}

// ObserveRelease registers a callback to be run on the release of another entity.
func (cx *Context[T]) ObserveRelease[T2 any](entity Entity[T2], onRelease func(this *T, entity *T2, cx *Context[T])) Subscription {
	entityID := entity.EntityID()
	this := cx.WeakEntity()
	return cx.App.releaseListeners.insert(entityID, func(entity any, cx *App) {
		state, ok := entity.(*T2)
		if !ok {
			panic("invalid entity type")
		}
		if this, ok := this.Upgrade(); ok {
			defer this.Release()
			this.Update(cx, func(this *T, cx *Context[T]) { onRelease(this, state, cx) })
		}
	})
}

// ObserveGlobal registers a callback for updates to the global of type G.
func (cx *Context[T]) ObserveGlobal[G any](f func(this *T, cx *Context[T])) Subscription {
	handle := cx.WeakEntity()
	subscription, activate := cx.App.globalObservers.insertPending(reflect.TypeFor[G](), func(cx *App) bool {
		return handle.Update(cx, func(view *T, cx *Context[T]) { f(view, cx) }) == nil
	})
	cx.App.Defer(func(*App) { activate() })
	return subscription
}

// OnAppRestart registers a callback to be invoked when the application is about to restart.
func (cx *Context[T]) OnAppRestart(onRestart func(this *T, cx *App)) Subscription {
	handle := cx.WeakEntity()
	return cx.App.OnAppRestart(func(cx *App) {
		_ = handle.Update(cx, func(entity *T, cx *Context[T]) { onRestart(entity, cx.App) })
	})
}

// OnAppQuit arranges for onQuit to be invoked whenever the application is quit. The returned
// future is polled for up to ShutdownTimeout until the app fully quits.
func (cx *Context[T]) OnAppQuit(onQuit func(this *T, cx *Context[T]) scheduler.Future[struct{}]) Subscription {
	handle := cx.WeakEntity()
	return cx.App.OnAppQuit(func(cx *App) scheduler.Future[struct{}] {
		future, err := handle.UpdateR(cx, func(entity *T, cx *Context[T]) scheduler.Future[struct{}] {
			return onQuit(entity, cx)
		})
		return scheduler.Async(func(s *scheduler.Scope) struct{} {
			if err == nil && future != nil {
				scheduler.Await(s, future)
			}
			return struct{}{}
		})
	})
}

// Notify tells GPUI that this entity has changed and observers of it should be notified.
func (cx *Context[T]) Notify() {
	cx.App.Notify(cx.entityState.entityID)
}

// Spawn spawns the future returned by f. f is given a weak handle to the entity owned by this
// context and an AsyncApp that can be held across await points. The returned task must be held
// or detached.
func (cx *Context[T]) Spawn[R any](f func(this WeakEntity[T], cx *AsyncApp) (R, error)) *Task[R] {
	this := cx.WeakEntity()
	return cx.App.Spawn(func(cx *AsyncApp) (R, error) { return f(this, cx) })
}

// Listener converts a listener on this entity into a window event listener of the form
// func(event, window, cx).
func (cx *Context[T]) Listener[E any](listener func(this *T, event *E, window *Window, cx *Context[T])) func(event *E, window *Window, cx *App) {
	entity := cx.Entity()
	view := entity.Downgrade()
	entity.Release()
	return func(event *E, window *Window, cx *App) {
		invokeListener(view, window, cx, func(view *T, window *Window, cx *Context[T]) {
			listener(view, event, window, cx)
		})
	}
}

// Processor converts a function on this entity into a window callback that produces a value.
// The callback holds a strong handle to the entity, as upstream's closure does.
func (cx *Context[T]) Processor[E, R any](f func(this *T, event E, window *Window, cx *Context[T]) R) func(event E, window *Window, cx *App) R {
	view := cx.Entity()
	return func(e E, window *Window, cx *App) R {
		return view.UpdateR(cx, func(view *T, cx *Context[T]) R { return f(view, e, window, cx) })
	}
}

// OnDrop returns a Deferred that runs f with this entity when it is run, as upstream's value
// does when it is dropped.
func (cx *Context[T]) OnDrop(f func(this *T, cx *Context[T])) gpuiutil.Deferred {
	this := cx.WeakEntity()
	acx := cx.ToAsync()
	return gpuiutil.Defer(func() {
		_ = this.Update(acx, f)
	})
}

// FocusView focuses the given view in the given window. *W must implement Focusable; it panics
// naming the type otherwise (D-COND).
func (cx *Context[T]) FocusView[W any](view Entity[W], window *Window) {
	focusable, ok := any(view.Read(cx.App)).(Focusable)
	if !ok {
		panic(fmt.Sprintf("gpui: %s does not implement Focusable", reflect.TypeFor[*W]()))
	}
	handle := focusable.FocusHandle(cx.App)
	defer handle.Release()
	window.Focus(handle, cx.App)
}

// OnNextFrame sets a callback to be run on the next frame.
func (cx *Context[T]) OnNextFrame(window *Window, f func(this *T, window *Window, cx *Context[T])) {
	view := cx.Entity()
	window.OnNextFrame(func(window *Window, cx *App) {
		defer view.Release()
		view.Update(cx, func(view *T, cx *Context[T]) { f(view, window, cx) })
	})
}

// DeferIn schedules f to be run at the end of the current effect cycle, allowing entities that
// are currently on the stack to be returned to the app.
func (cx *Context[T]) DeferIn(window *Window, f func(this *T, window *Window, cx *Context[T])) {
	view := cx.WeakEntity()
	entityID := cx.EntityID()
	cx.App.ensureWindow(entityID, window.handle.id)
	cx.App.Defer(func(cx *App) {
		cx.withWindow(entityID, func(window *Window, cx *App) {
			_ = view.Update(cx, func(view *T, cx *Context[T]) { f(view, window, cx) })
		})
	})
}

// ObserveIn observes another entity for changes to its state, as tracked by Notify, with
// access to this entity's window.
func (cx *Context[T]) ObserveIn[V2 any](observed Entity[V2], window *Window, onNotify func(this *T, observed Entity[V2], window *Window, cx *Context[T])) Subscription {
	observedID := observed.EntityID()
	observedWeak := observed.Downgrade()
	observer := cx.WeakEntity()
	observerID := cx.EntityID()
	cx.App.ensureWindow(observerID, window.handle.id)
	return cx.App.newObserver(observedID, func(cx *App) bool {
		return invokeObserverIn(observer, observedWeak, cx, onNotify)
	})
}

// SubscribeIn subscribes to events of type E emitted by another entity, with access to this
// entity's window.
func (cx *Context[T]) SubscribeIn[Emitter, E any](emitter Entity[Emitter], window *Window, onEvent func(this *T, emitter Entity[Emitter], event *E, window *Window, cx *Context[T])) Subscription {
	emitterWeak := emitter.Downgrade()
	subscriber := cx.WeakEntity()
	subscriberID := cx.EntityID()
	cx.App.ensureWindow(subscriberID, window.handle.id)
	return cx.App.newSubscription(emitterWeak.EntityID(), eventListener{
		eventType: reflect.TypeFor[E](),
		listener: func(event any, cx *App) bool {
			return invokeSubscriberIn(subscriber, emitterWeak, event, cx, onEvent)
		},
	})
}

// OnReleaseIn registers a callback to be invoked when the view is released. The window handle
// may be invalid if the window was closed before the view was released.
func (cx *Context[T]) OnReleaseIn(window *Window, onRelease func(this *T, window *Window, cx *App)) Subscription {
	entity := cx.Entity()
	defer entity.Release()
	return cx.App.ObserveReleaseIn(entity, window, onRelease)
}

// ObserveReleaseIn registers a callback to be invoked when the given entity is released.
func (cx *Context[T]) ObserveReleaseIn[T2 any](observed Entity[T2], window *Window, onRelease func(this *T, observed *T2, window *Window, cx *Context[T])) Subscription {
	observer := cx.WeakEntity()
	return cx.App.ObserveReleaseIn(observed, window, func(observed *T2, window *Window, cx *App) {
		_ = observer.Update(cx, func(observer *T, cx *Context[T]) { onRelease(observer, observed, window, cx) })
	})
}

func observeWindowWith[T any](view WeakEntity[T], callback func(this *T, window *Window, cx *Context[T])) anyObserver {
	return func(window *Window, cx *App) bool {
		return view.Update(cx, func(view *T, cx *Context[T]) { callback(view, window, cx) }) == nil
	}
}

// ObserveWindowBounds registers a callback to be invoked when the window is resized.
func (cx *Context[T]) ObserveWindowBounds(window *Window, callback func(this *T, window *Window, cx *Context[T])) Subscription {
	return window.boundsObservers.insert(observeWindowWith(cx.WeakEntity(), callback))
}

// ObserveWindowActivation registers a callback to be invoked when the window is activated or
// deactivated.
func (cx *Context[T]) ObserveWindowActivation(window *Window, callback func(this *T, window *Window, cx *Context[T])) Subscription {
	return window.activationObservers.insert(observeWindowWith(cx.WeakEntity(), callback))
}

// ObserveWindowVisibility registers a callback to be invoked when the window's visibility
// changes.
func (cx *Context[T]) ObserveWindowVisibility(window *Window, callback func(this *T, visibility WindowVisibility, window *Window, cx *Context[T])) Subscription {
	view := cx.WeakEntity()
	return window.visibilityObservers.insert(func(visibility WindowVisibility, window *Window, cx *App) bool {
		return view.Update(cx, func(view *T, cx *Context[T]) { callback(view, visibility, window, cx) }) == nil
	})
}

// ObserveWindowAppearance registers a callback to be invoked when the window appearance
// changes.
func (cx *Context[T]) ObserveWindowAppearance(window *Window, callback func(this *T, window *Window, cx *Context[T])) Subscription {
	return window.appearanceObservers.insert(observeWindowWith(cx.WeakEntity(), callback))
}

// ObserveWindowDisplay registers a callback invoked when the window moves to another display
// or its display's refresh interval changes.
func (cx *Context[T]) ObserveWindowDisplay(window *Window, callback func(this *T, window *Window, cx *Context[T])) Subscription {
	return window.displayObservers.insert(observeWindowWith(cx.WeakEntity(), callback))
}

// ObserveButtonLayoutChanged registers a callback to be invoked when the window button layout
// changes.
func (cx *Context[T]) ObserveButtonLayoutChanged(window *Window, callback func(this *T, window *Window, cx *Context[T])) Subscription {
	return window.buttonLayoutObservers.insert(observeWindowWith(cx.WeakEntity(), callback))
}

// ObserveKeystrokes registers a callback for App.ObserveKeystrokes that updates this entity.
func (cx *Context[T]) ObserveKeystrokes(f func(this *T, event *KeystrokeEvent, window *Window, cx *Context[T])) Subscription {
	view := cx.WeakEntity()
	return cx.App.keystrokeObservers.insert(func(event *KeystrokeEvent, window *Window, cx *App) bool {
		view, ok := view.Upgrade()
		if !ok {
			return false
		}
		defer view.Release()
		view.Update(cx, func(view *T, cx *Context[T]) { f(view, event, window, cx) })
		return true
	})
}

// ObservePendingInput registers a callback to be invoked when the window's pending input
// changes.
func (cx *Context[T]) ObservePendingInput(window *Window, callback func(this *T, window *Window, cx *Context[T])) Subscription {
	return window.pendingInputObservers.insert(observeWindowWith(cx.WeakEntity(), callback))
}

func (cx *Context[T]) focusListener(window *Window, listener func(view *T, event *windowFocusEvent, window *Window, cx *Context[T])) Subscription {
	view := cx.WeakEntity()
	subscription, activate := window.newFocusListener(func(event *windowFocusEvent, window *Window, cx *App) bool {
		return view.Update(cx, func(view *T, cx *Context[T]) { listener(view, event, window, cx) }) == nil
	})
	cx.App.Defer(func(*App) { activate() })
	return subscription
}

func lastFocusID(path []FocusID) (FocusID, bool) {
	if len(path) == 0 {
		return 0, false
	}
	return path[len(path)-1], true
}

func lastFocusIs(path []FocusID, id FocusID) bool {
	last, ok := lastFocusID(path)
	return ok && last == id
}

// OnFocus registers a listener to be called when the given focus handle receives focus. It
// persists until the subscription is dropped.
func (cx *Context[T]) OnFocus(handle FocusHandle, window *Window, listener func(this *T, window *Window, cx *Context[T])) Subscription {
	focusID := handle.id
	return cx.focusListener(window, func(view *T, event *windowFocusEvent, window *Window, cx *Context[T]) {
		if !lastFocusIs(event.previousFocusPath, focusID) && lastFocusIs(event.currentFocusPath, focusID) {
			listener(view, window, cx)
		}
	})
}

// OnFocusIn registers a listener to be called when the given focus handle or one of its
// descendants receives focus. It does not fire if the handle or a descendant was previously
// focused.
func (cx *Context[T]) OnFocusIn(handle FocusHandle, window *Window, listener func(this *T, window *Window, cx *Context[T])) Subscription {
	focusID := handle.id
	return cx.focusListener(window, func(view *T, event *windowFocusEvent, window *Window, cx *Context[T]) {
		if event.isFocusIn(focusID) {
			listener(view, window, cx)
		}
	})
}

// OnBlur registers a listener to be called when the given focus handle loses focus.
func (cx *Context[T]) OnBlur(handle FocusHandle, window *Window, listener func(this *T, window *Window, cx *Context[T])) Subscription {
	focusID := handle.id
	return cx.focusListener(window, func(view *T, event *windowFocusEvent, window *Window, cx *Context[T]) {
		if lastFocusIs(event.previousFocusPath, focusID) && !lastFocusIs(event.currentFocusPath, focusID) {
			listener(view, window, cx)
		}
	})
}

// OnFocusLost registers a listener to be called when nothing in the window has focus, which
// typically happens when the focused node is removed from the tree.
func (cx *Context[T]) OnFocusLost(window *Window, listener func(this *T, window *Window, cx *Context[T])) Subscription {
	view := cx.WeakEntity()
	subscription, activate := window.focusLostListeners.insertPending(func(window *Window, cx *App) bool {
		return view.Update(cx, func(view *T, cx *Context[T]) { listener(view, window, cx) }) == nil
	})
	cx.App.Defer(func(*App) { activate() })
	return subscription
}

// OnFocusOut registers a listener to be called when the given focus handle or one of its
// descendants loses focus.
func (cx *Context[T]) OnFocusOut(handle FocusHandle, window *Window, listener func(this *T, event FocusOutEvent, window *Window, cx *Context[T])) Subscription {
	focusID := handle.id
	return cx.focusListener(window, func(view *T, event *windowFocusEvent, window *Window, cx *Context[T]) {
		if blurredID, ok := lastFocusID(event.previousFocusPath); ok && event.isFocusOut(focusID) {
			listener(view, FocusOutEvent{Blurred: newWeakFocusHandle(blurredID, cx.App.focusHandles)}, window, cx)
		}
	})
}

// SpawnIn schedules a future to be run asynchronously on the main thread. f is given a weak
// handle to this entity and an AsyncWindowContext that can be held across await points.
func (cx *Context[T]) SpawnIn[R any](window *Window, f func(this WeakEntity[T], cx *AsyncWindowContext) (R, error)) *Task[R] {
	view := cx.WeakEntity()
	return window.Spawn(cx.App, func(cx *AsyncWindowContext) (R, error) { return f(view, cx) })
}

// SpawnInWithPriority schedules a future to be run asynchronously on the main thread with the
// given priority.
func (cx *Context[T]) SpawnInWithPriority[R any](priority Priority, window *Window, f func(this WeakEntity[T], cx *AsyncWindowContext) (R, error)) *Task[R] {
	view := cx.WeakEntity()
	return window.SpawnWithPriority(priority, cx.App, func(cx *AsyncWindowContext) (R, error) { return f(view, cx) })
}

// ObserveGlobalIn registers a callback to be invoked when the global of type G changes, with
// access to the given window.
func (cx *Context[T]) ObserveGlobalIn[G any](window *Window, f func(this *T, window *Window, cx *Context[T])) Subscription {
	windowHandle := window.handle
	view := cx.WeakEntity()
	subscription, activate := cx.App.globalObservers.insertPending(reflect.TypeFor[G](), func(cx *App) bool {
		if !view.IsUpgradable() {
			return false
		}
		entityAlive := false
		if err := windowHandle.Update(cx, func(_ AnyView, window *Window, cx *App) {
			entityAlive = view.Update(cx, func(view *T, cx *Context[T]) { f(view, window, cx) }) == nil
		}); err != nil {
			return true
		}
		return entityAlive
	})
	cx.App.Defer(func(*App) { activate() })
	return subscription
}

// OnAction registers a callback to be invoked when the given action type is dispatched to the
// window.
func (cx *Context[T]) OnAction(actionType reflect.Type, window *Window, listener func(this *T, action any, phase DispatchPhase, window *Window, cx *Context[T])) {
	handle := cx.WeakEntity()
	window.OnAction(actionType, func(action any, phase DispatchPhase, window *Window, cx *App) {
		_ = handle.Update(cx, func(view *T, cx *Context[T]) { listener(view, action, phase, window, cx) })
	})
}

// FocusSelf moves focus to the current view. *T must implement Focusable; it panics naming the
// type otherwise (D-COND).
func (cx *Context[T]) FocusSelf(window *Window) {
	view := cx.Entity()
	window.Defer(cx.App, func(window *Window, cx *App) {
		defer view.Release()
		focusable, ok := any(view.Read(cx)).(Focusable)
		if !ok {
			panic(fmt.Sprintf("gpui: %s does not implement Focusable", reflect.TypeFor[*T]()))
		}
		handle := focusable.FocusHandle(cx)
		defer handle.Release()
		handle.Focus(window, cx)
	})
}

// Emit emits an event of type E, which other entities subscribed with Subscribe handle. It is
// not checked at compile time that T emits E (D-EMIT).
func (cx *Context[T]) Emit[E any](event E) {
	cx.App.pushEffect(effectEmit{
		emitter:   cx.entityState.entityID,
		eventType: reflect.TypeFor[E](),
		event:     &event,
	})
}

func invokeObserver[T, W any](observer WeakEntity[T], observed Entity[W], cx *App, onNotify func(this *T, entity Entity[W], cx *Context[T])) bool {
	strong, ok := observer.Upgrade()
	if !ok {
		return false
	}
	defer strong.Release()
	strong.Update(cx, func(observer *T, cx *Context[T]) { onNotify(observer, observed, cx) })
	return true
}

func invokeSubscriber[T, Emitter, E any](subscriber WeakEntity[T], emitter Entity[Emitter], event *E, cx *App, onEvent func(this *T, entity Entity[Emitter], event *E, cx *Context[T])) bool {
	strong, ok := subscriber.Upgrade()
	if !ok {
		return false
	}
	defer strong.Release()
	strong.Update(cx, func(subscriber *T, cx *Context[T]) { onEvent(subscriber, emitter, event, cx) })
	return true
}

func invokeObserverIn[T, W any](observer WeakEntity[T], observed WeakEntity[W], cx *App, onNotify func(this *T, observed Entity[W], window *Window, cx *Context[T])) bool {
	strongObserver, ok := observer.Upgrade()
	if !ok {
		return false
	}
	defer strongObserver.Release()
	strongObserved, ok := observed.Upgrade()
	if !ok {
		return false
	}
	defer strongObserved.Release()
	cx.withWindow(strongObserver.EntityID(), func(window *Window, cx *App) {
		strongObserver.Update(cx, func(observer *T, cx *Context[T]) {
			onNotify(observer, strongObserved, window, cx)
		})
	})
	return true
}

func invokeSubscriberIn[T, Emitter, E any](subscriber WeakEntity[T], emitter WeakEntity[Emitter], event any, cx *App, onEvent func(this *T, emitter Entity[Emitter], event *E, window *Window, cx *Context[T])) bool {
	strongSubscriber, ok := subscriber.Upgrade()
	if !ok {
		return false
	}
	defer strongSubscriber.Release()
	strongEmitter, ok := emitter.Upgrade()
	if !ok {
		return false
	}
	defer strongEmitter.Release()
	typed, ok := event.(*E)
	if !ok {
		panic("invalid event type")
	}
	cx.withWindow(strongSubscriber.EntityID(), func(window *Window, cx *App) {
		strongSubscriber.Update(cx, func(subscriber *T, cx *Context[T]) {
			onEvent(subscriber, strongEmitter, typed, window, cx)
		})
	})
	return true
}

func invokeListener[T any](view WeakEntity[T], window *Window, cx *App, listener func(view *T, window *Window, cx *Context[T])) {
	_ = view.Update(cx, func(view *T, cx *Context[T]) { listener(view, window, cx) })
}
