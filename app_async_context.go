package gpui

import (
	"errors"
	"reflect"
	"weak"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// AsyncApp is an async-friendly version of App that can be held across await points. App.Spawn
// provides one, and App.ToAsync creates one.
//
// It holds the App weakly. Methods panic if the app has been released, which does not happen
// for foreground tasks spawned with Spawn. A foreground task's body runs as a coroutine that
// the main thread resumes, so every method reaches the App on the main thread through the
// AppCell borrow guard. It embeds the task's Scope, so it is the Awaiter of task.Await(cx).
type AsyncApp struct {
	*scheduler.Scope
	app                weak.Pointer[AppCell]
	bgExecutor         BackgroundExecutor
	foregroundExecutor ForegroundExecutor
}

var (
	errAppReleased = errors.New("app was released")
	errAppQuitting = errors.New("app is quitting")
)

func (cx *AsyncApp) appCell() *AppCell {
	app := cx.app.Value()
	if app == nil {
		panic("app was released before async operation completed")
	}
	return app
}

func (cx *AsyncApp) upgradeApp() (*AppCell, bool) {
	app := cx.app.Value()
	return app, app != nil
}

func (cx *AsyncApp) clone() *AsyncApp {
	child := *cx
	child.Scope = nil
	return &child
}

func (cx *AsyncApp) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	app := cx.appCell().BorrowMut()
	defer app.Release()
	return app.newEntity(entityType, build)
}

func (cx *AsyncApp) reserveEntity(entityType reflect.Type) AnyEntity {
	app := cx.appCell().BorrowMut()
	defer app.Release()
	return app.reserveEntity(entityType)
}

func (cx *AsyncApp) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	app := cx.appCell().BorrowMut()
	defer app.Release()
	return app.insertEntity(slot, build)
}

func (cx *AsyncApp) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	app := cx.appCell().BorrowMut()
	defer app.Release()
	app.updateEntity(handle, update)
}

func (cx *AsyncApp) asMut(handle AnyEntity) *App {
	panic("Cannot as_mut with an async context. Try calling update() first")
}

func (cx *AsyncApp) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	app := cx.appCell().Borrow()
	defer app.Release()
	app.Get().readEntity(handle, read)
}

func (cx *AsyncApp) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	cell, ok := cx.upgradeApp()
	if !ok {
		return errAppReleased
	}
	lock, err := cell.tryBorrowMut(gpuiutil.CallerLocation(1))
	if err != nil {
		return err
	}
	defer lock.Release()
	if lock.quitting {
		return errAppQuitting
	}
	return lock.updateWindow(window, update)
}

func (cx *AsyncApp) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	cell, ok := cx.upgradeApp()
	if !ok {
		return false
	}
	lock, err := cell.tryBorrowMut(gpuiutil.CallerLocation(1))
	if err != nil {
		return false
	}
	defer lock.Release()
	if lock.quitting {
		return false
	}
	return lock.withWindow(entityID, f)
}

func (cx *AsyncApp) readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error {
	cell, ok := cx.upgradeApp()
	if !ok {
		return errAppReleased
	}
	lock := cell.Borrow()
	defer lock.Release()
	if lock.Get().quitting {
		return errAppQuitting
	}
	return lock.Get().readWindow(window, read)
}

func (cx *AsyncApp) backgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

func (cx *AsyncApp) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	lock := cx.appCell().BorrowMut()
	defer lock.Release()
	lock.update(func(cx *App) { cx.readGlobal(globalType, read) })
}

// New creates a new entity.
func (cx *AsyncApp) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// ReserveEntity reserves a slot for an entity to be inserted later with InsertEntity.
func (cx *AsyncApp) ReserveEntity[T any]() Reservation[T] {
	return AppContextReserveEntity[T](cx)
}

// InsertEntity inserts an entity into a slot obtained from ReserveEntity.
func (cx *AsyncApp) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

// ReadWindow reads the root view of a window.
func (cx *AsyncApp) ReadWindow[V, R any](window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	return AppContextReadWindow(cx, window, read)
}

// BackgroundSpawn spawns a future on a background thread.
func (cx *AsyncApp) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return cx.bgExecutor.Spawn(future)
}

// Refresh schedules all windows in the application to be redrawn.
func (cx *AsyncApp) Refresh() {
	lock := cx.appCell().BorrowMut()
	defer lock.Release()
	lock.update(func(cx *App) { cx.RefreshWindows() })
}

// BackgroundExecutor returns an executor which can be used to spawn futures in the background.
func (cx *AsyncApp) BackgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

// IsTest reports whether this app runs on the deterministic test scheduler.
func (cx *AsyncApp) IsTest() bool {
	return cx.bgExecutor.IsTest()
}

// ForegroundExecutor returns an executor which can be used to spawn futures in the foreground.
func (cx *AsyncApp) ForegroundExecutor() ForegroundExecutor {
	return cx.foregroundExecutor
}

// Update invokes f in the context of the app, then flushes any effects produced during its
// invocation.
func (cx *AsyncApp) Update(f func(cx *App)) {
	lock := cx.appCell().BorrowMut()
	defer lock.Release()
	lock.update(f)
}

// UpdateR invokes f in the context of the app, flushes its effects and returns its result.
func (cx *AsyncApp) UpdateR[R any](f func(cx *App) R) R {
	lock := cx.appCell().BorrowMut()
	defer lock.Release()
	return lock.updateR(f)
}

// Subscribe arranges for onEvent to be invoked whenever the given entity emits an event of type
// E.
func (cx *AsyncApp) Subscribe[T, E any](entity Entity[T], onEvent func(entity Entity[T], event *E, cx *App)) Subscription {
	lock := cx.appCell().BorrowMut()
	defer lock.Release()
	return lock.Subscribe(entity, onEvent)
}

// OpenWindow opens a window with the given options based on the root view returned by
// buildRootView.
func (cx *AsyncApp) OpenWindow[V any](options WindowOptions, buildRootView func(window *Window, cx *App) Entity[V]) (WindowHandle[V], error) {
	lock := cx.appCell().BorrowMut()
	defer lock.Release()
	if lock.quitting {
		return WindowHandle[V]{}, errAppQuitting
	}
	return lock.OpenWindow(options, buildRootView)
}

// Spawn schedules a future to be polled in the foreground. f gets its own AsyncApp, whose Scope
// is the new task's.
func (cx *AsyncApp) Spawn[R any](f func(cx *AsyncApp) (R, error)) *Task[R] {
	child := cx.clone()
	return cx.foregroundExecutor.Spawn(func(s *scheduler.Scope) (R, error) {
		child.Scope = s
		return f(child)
	})
}

// HasGlobal reports whether a global of type G has been assigned.
func (cx *AsyncApp) HasGlobal[G any]() bool {
	app := cx.appCell().BorrowMut()
	defer app.Release()
	return app.HasGlobal[G]()
}

// ReadGlobal reads the global of type G, passing it to read. It panics if no global of that
// type has been assigned.
func (cx *AsyncApp) ReadGlobal[G, R any](read func(global *G, cx *App) R) R {
	app := cx.appCell().BorrowMut()
	defer app.Release()
	return read(app.Global[G](), app.App)
}

// TryReadGlobal reads the global of type G, passing it to read. Unlike ReadGlobal it reports
// false instead of panicking.
func (cx *AsyncApp) TryReadGlobal[G, R any](read func(global *G, cx *App) R) (R, bool) {
	var zero R
	app := cx.appCell().BorrowMut()
	defer app.Release()
	if app.quitting {
		return zero, false
	}
	global, ok := app.TryGlobal[G]()
	if !ok {
		return zero, false
	}
	return read(global, app.App), true
}

// ReadDefaultGlobal reads the global of type G, assigning its zero value first if it has not
// been assigned.
func (cx *AsyncApp) ReadDefaultGlobal[G, R any](read func(global *G, cx *App) R) R {
	app := cx.appCell().BorrowMut()
	defer app.Release()
	app.update(func(cx *App) { cx.DefaultGlobal[G]() })
	return read(app.Global[G](), app.App)
}

// UpdateGlobal updates the global of type G.
func (cx *AsyncApp) UpdateGlobal[G any](update func(global *G, cx *App)) {
	cx.UpdateGlobalR(func(global *G, cx *App) struct{} {
		update(global, cx)
		return struct{}{}
	})
}

// UpdateGlobalR updates the global of type G and returns the callback's result.
func (cx *AsyncApp) UpdateGlobalR[G, R any](update func(global *G, cx *App) R) R {
	app := cx.appCell().BorrowMut()
	defer app.Release()
	return app.updateR(func(cx *App) R { return UpdateGlobalR(cx, update) })
}

// AsyncAppOnDrop returns a Deferred that runs f with the entity when it is run, as upstream's
// value does when it is dropped.
func AsyncAppOnDrop[T any](cx *AsyncApp, entity WeakEntity[T], f func(entity *T, cx *Context[T])) gpuiutil.Deferred {
	acx := cx.clone()
	return gpuiutil.Defer(func() {
		_ = entity.Update(acx, f)
	})
}

// AsyncWindowContext is a cloneable, owned handle to the application context, composed with
// the window associated with the current task. It embeds AsyncApp, upstream's Deref.
type AsyncWindowContext struct {
	*AsyncApp
	window AnyWindowHandle
}

func newAsyncWindowContext(app *AsyncApp, window AnyWindowHandle) *AsyncWindowContext {
	return &AsyncWindowContext{AsyncApp: app, window: window}
}

// WindowHandle returns the handle of the window this context is associated with.
func (cx *AsyncWindowContext) WindowHandle() AnyWindowHandle {
	return cx.window
}

// Update is a convenience method for App.UpdateWindow.
func (cx *AsyncWindowContext) Update(update func(window *Window, cx *App)) error {
	return cx.AsyncApp.updateWindow(cx.window, func(_ AnyView, window *Window, cx *App) { update(window, cx) })
}

// UpdateR is Update returning the callback's result.
func (cx *AsyncWindowContext) UpdateR[R any](update func(window *Window, cx *App) R) (R, error) {
	var result R
	err := cx.AsyncApp.updateWindow(cx.window, func(_ AnyView, window *Window, cx *App) { result = update(window, cx) })
	return result, err
}

// UpdateRoot is a convenience method for App.UpdateWindow that also passes the root view.
func (cx *AsyncWindowContext) UpdateRoot(update func(root AnyView, window *Window, cx *App)) error {
	return cx.AsyncApp.updateWindow(cx.window, update)
}

// UpdateRootR is UpdateRoot returning the callback's result.
func (cx *AsyncWindowContext) UpdateRootR[R any](update func(root AnyView, window *Window, cx *App) R) (R, error) {
	var result R
	err := cx.AsyncApp.updateWindow(cx.window, func(root AnyView, window *Window, cx *App) { result = update(root, window, cx) })
	return result, err
}

// OnNextFrame is a convenience method for Window.OnNextFrame.
func (cx *AsyncWindowContext) OnNextFrame(f func(window *Window, cx *App)) {
	_ = cx.AsyncApp.updateWindow(cx.window, func(_ AnyView, window *Window, _ *App) { window.OnNextFrame(f) })
}

// ReadGlobal is a convenience method for App.Global with the window.
func (cx *AsyncWindowContext) ReadGlobal[G, R any](read func(global *G, window *Window, cx *App) R) (R, error) {
	return cx.UpdateR(func(window *Window, cx *App) R { return read(cx.Global[G](), window, cx) })
}

// UpdateGlobal updates the global of type G with the window.
func (cx *AsyncWindowContext) UpdateGlobal[G any](update func(global *G, window *Window, cx *App)) error {
	_, err := cx.UpdateGlobalR(func(global *G, window *Window, cx *App) struct{} {
		update(global, window, cx)
		return struct{}{}
	})
	return err
}

// UpdateGlobalR updates the global of type G with the window and returns the callback's
// result.
func (cx *AsyncWindowContext) UpdateGlobalR[G, R any](update func(global *G, window *Window, cx *App) R) (R, error) {
	return cx.UpdateR(func(window *Window, cx *App) R {
		return UpdateGlobalR(cx, func(global *G, cx *App) R { return update(global, window, cx) })
	})
}

// Spawn schedules a future to be executed on the main thread. f gets its own
// AsyncWindowContext, whose Scope is the new task's.
func (cx *AsyncWindowContext) Spawn[R any](f func(cx *AsyncWindowContext) (R, error)) *Task[R] {
	child := &AsyncWindowContext{AsyncApp: cx.AsyncApp.clone(), window: cx.window}
	return cx.foregroundExecutor.Spawn(func(s *scheduler.Scope) (R, error) {
		child.Scope = s
		return f(child)
	})
}

// Prompt presents a platform dialog with message and a button for each answer. The returned
// receiver gets the index of the clicked button; it is closed if the window is gone.
func (cx *AsyncWindowContext) Prompt(level PromptLevel, message string, detail Option[string], answers []PromptButton) *scheduler.OneshotReceiver[int] {
	rx, err := cx.UpdateR(func(window *Window, cx *App) *scheduler.OneshotReceiver[int] {
		return window.Prompt(level, message, detail, answers, cx)
	})
	if err != nil {
		tx, rx := scheduler.Oneshot[int]()
		tx.Close()
		return rx
	}
	return rx
}

func (cx *AsyncWindowContext) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	var entity AnyEntity
	called := false
	err := cx.AsyncApp.updateWindow(cx.window, func(_ AnyView, _ *Window, app *App) {
		called = true
		entity = app.newEntity(entityType, build)
	})
	if err == nil {
		return entity
	}
	if called {
		panic("update_window returned Err without invoking the closure")
	}
	return cx.AsyncApp.newEntity(entityType, build)
}

func (cx *AsyncWindowContext) reserveEntity(entityType reflect.Type) AnyEntity {
	return cx.AsyncApp.reserveEntity(entityType)
}

func (cx *AsyncWindowContext) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	var entity AnyEntity
	called := false
	err := cx.AsyncApp.updateWindow(cx.window, func(_ AnyView, _ *Window, app *App) {
		called = true
		entity = app.insertEntity(slot, build)
	})
	if err == nil {
		return entity
	}
	if called {
		panic("update_window returned Err without invoking the closure")
	}
	return cx.AsyncApp.insertEntity(slot, build)
}

func (cx *AsyncWindowContext) asMut(handle AnyEntity) *App {
	panic("Cannot use as_mut() from an async context, call `update`")
}

func (cx *AsyncWindowContext) updateWindowContext(update func(window *Window, cx *App)) error {
	return cx.AsyncApp.updateWindow(cx.window, func(_ AnyView, window *Window, cx *App) { update(window, cx) })
}

// New creates a new entity, inside the window's update when the window is available.
func (cx *AsyncWindowContext) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// ReserveEntity reserves a slot for an entity to be inserted later with InsertEntity.
func (cx *AsyncWindowContext) ReserveEntity[T any]() Reservation[T] {
	return AppContextReserveEntity[T](cx)
}

// InsertEntity inserts an entity into a slot obtained from ReserveEntity.
func (cx *AsyncWindowContext) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

var (
	_ AppContext    = (*AsyncWindowContext)(nil)
	_ VisualContext = (*AsyncWindowContext)(nil)
)
