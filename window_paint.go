package gpui

import (
	"cmp"
	"encoding/json/jsontext"
	"fmt"
	"image"
	"log/slog"
	"maps"
	"math"
	"reflect"
	"runtime"
	"slices"
	"time"

	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// ScaledPixelsRange is a half-open range of device-space offsets, upstream's
// std::ops::Range<ScaledPixels> (D-SPRANGE).
type ScaledPixelsRange struct {
	// Start is the first offset in the range.
	Start ScaledPixels
	// End is one past the last offset in the range.
	End ScaledPixels
}

func (w *Window) markViewDirty(viewID EntityID) {
	for viewID := range w.renderedFrame.dispatchTree.viewPathReversed(viewID) {
		if _, ok := w.dirtyViews[viewID]; ok {
			break
		}
		w.dirtyViews[viewID] = struct{}{}
	}
}

func (w *Window) refreshVisibility(cx *App) {
	visibility := w.platformWindow.Visibility()
	if w.visibility != visibility {
		w.visibility = visibility
		if profilerFeatureEnabled {
			recordWindowVisibility(w.handle.WindowID(), visibility)
		}
		w.visibilityObservers.retain(func(callback *func(visibility WindowVisibility, window *Window, cx *App) bool) bool {
			return (*callback)(visibility, w, cx)
		})
	}
	if profilerFeatureEnabled && (w.invalidator.isDirty() || *w.needsPresent) {
		recordFramePending(w.handle.WindowID(), scheduler.InstantNow())
	}
}

type debugStateCachedU64 struct {
	Cached   Option[uint64] `json:"cached"`
	Platform Option[uint64] `json:"platform"`
}

type debugStateCachedF32 struct {
	Cached   float32 `json:"cached"`
	Platform float32 `json:"platform"`
}

type debugStateCachedBool struct {
	Cached   bool `json:"cached"`
	Platform bool `json:"platform"`
}

type debugStateCachedString struct {
	Cached   string `json:"cached"`
	Platform string `json:"platform"`
}

type debugStateCachedPoint struct {
	Cached   Point[Pixels] `json:"cached"`
	Platform Point[Pixels] `json:"platform"`
}

type debugStateCachedModifiers struct {
	Cached   Modifiers `json:"cached"`
	Platform Modifiers `json:"platform"`
}

type debugStateFrame struct {
	Dirty              bool `json:"dirty"`
	NeedsPresent       bool `json:"needs_present"`
	Refreshing         bool `json:"refreshing"`
	NextFrameCallbacks int  `json:"next_frame_callbacks"`
}

type debugStateWindow struct {
	ID           uint64                    `json:"id"`
	Title        string                    `json:"title"`
	Bounds       Bounds[Pixels]            `json:"bounds"`
	WindowBounds string                    `json:"window_bounds"`
	ContentSize  Size[Pixels]              `json:"content_size"`
	ViewportSize Size[Pixels]              `json:"viewport_size"`
	ScaleFactor  debugStateCachedF32       `json:"scale_factor"`
	DisplayID    debugStateCachedU64       `json:"display_id"`
	Active       debugStateCachedBool      `json:"active"`
	Hovered      debugStateCachedBool      `json:"hovered"`
	Visibility   debugStateCachedString    `json:"visibility"`
	Appearance   debugStateCachedString    `json:"appearance"`
	IsFullscreen bool                      `json:"is_fullscreen"`
	IsMaximized  bool                      `json:"is_maximized"`
	MousePos     debugStateCachedPoint     `json:"mouse_position"`
	Modifiers    debugStateCachedModifiers `json:"modifiers"`
	CursorStyle  Option[CursorStyle]       `json:"cursor_style"`
	Frame        debugStateFrame           `json:"frame"`
	GPU          Option[GPUSpecs]          `json:"gpu"`
	Platform     jsontext.Value            `json:"platform"`
}

func windowVisibilityDebugName(visibility WindowVisibility) string {
	switch visibility {
	case WindowVisibilityVisible:
		return "Visible"
	case WindowVisibilityHidden:
		return "Hidden"
	}
	return fmt.Sprintf("WindowVisibility(%d)", visibility)
}

func windowBoundsDebugString(bounds WindowBounds) string {
	switch bounds.Kind() {
	case WindowBoundsKindMaximized:
		return "Maximized(" + bounds.GetBounds().GoString() + ")"
	case WindowBoundsKindFullscreen:
		return "Fullscreen(" + bounds.GetBounds().GoString() + ")"
	}
	return "Windowed(" + bounds.GetBounds().GoString() + ")"
}

func displayIDU64(display PlatformDisplay) Option[uint64] {
	if display == nil {
		return None[uint64]()
	}
	return Some(uint64(display.ID()))
}

func (w *Window) debugState() jsontext.Value {
	platformWindow := w.platformWindow
	cachedDisplayID := None[uint64]()
	if id, ok := w.displayID.Get(); ok {
		cachedDisplayID = Some(uint64(id))
	}
	return marshalDebugState(debugStateWindow{
		ID:           w.handle.WindowID().AsU64(),
		Title:        platformWindow.GetTitle(),
		Bounds:       platformWindow.Bounds(),
		WindowBounds: windowBoundsDebugString(platformWindow.WindowBounds()),
		ContentSize:  platformWindow.ContentSize(),
		ViewportSize: w.viewportSize,
		ScaleFactor: debugStateCachedF32{
			Cached:   w.scaleFactor,
			Platform: platformWindow.ScaleFactor(),
		},
		DisplayID: debugStateCachedU64{
			Cached:   cachedDisplayID,
			Platform: displayIDU64(platformWindow.Display()),
		},
		Active: debugStateCachedBool{
			Cached:   *w.active,
			Platform: platformWindow.IsActive(),
		},
		Hovered: debugStateCachedBool{
			Cached:   *w.hovered,
			Platform: platformWindow.IsHovered(),
		},
		Visibility: debugStateCachedString{
			Cached:   windowVisibilityDebugName(w.visibility),
			Platform: windowVisibilityDebugName(platformWindow.Visibility()),
		},
		Appearance: debugStateCachedString{
			Cached:   windowAppearanceDebugName(w.appearance),
			Platform: windowAppearanceDebugName(platformWindow.Appearance()),
		},
		IsFullscreen: platformWindow.IsFullscreen(),
		IsMaximized:  platformWindow.IsMaximized(),
		MousePos: debugStateCachedPoint{
			Cached:   w.mousePosition,
			Platform: platformWindow.MousePosition(),
		},
		Modifiers: debugStateCachedModifiers{
			Cached:   w.modifiers,
			Platform: platformWindow.Modifiers(),
		},
		CursorStyle: OptionFrom(w.renderedFrame.cursorStyle(w)),
		Frame: debugStateFrame{
			Dirty:              w.invalidator.isDirty(),
			NeedsPresent:       *w.needsPresent,
			Refreshing:         w.refreshing,
			NextFrameCallbacks: len(*w.nextFrameCallbacks),
		},
		GPU:      OptionFrom(platformWindow.GPUSpecs()),
		Platform: platformWindow.DebugState(),
	})
}

// Visibility reports whether the platform is presenting this window's frames.
func (w *Window) Visibility() WindowVisibility {
	return w.visibility
}

// IsVisible reports whether frames drawn for this window will be shown. A shown window that is
// fully covered, minimized or on a sleeping display is not visible here.
func (w *Window) IsVisible() bool {
	return w.visibility.IsVisible()
}

// ObserveWindowVisibility registers a callback invoked when the window's visibility changes.
func (w *Window) ObserveWindowVisibility(callback func(visibility WindowVisibility, window *Window, cx *App)) Subscription {
	return w.visibilityObservers.insert(func(visibility WindowVisibility, window *Window, cx *App) bool {
		callback(visibility, window, cx)
		return true
	})
}

// ObserveWindowAppearance registers a callback invoked when the window appearance changes.
func (w *Window) ObserveWindowAppearance(callback func(window *Window, cx *App)) Subscription {
	return w.appearanceObservers.insert(func(window *Window, cx *App) bool {
		callback(window, cx)
		return true
	})
}

// ObserveButtonLayoutChanged registers a callback invoked when the window button layout changes.
func (w *Window) ObserveButtonLayoutChanged(callback func(window *Window, cx *App)) Subscription {
	return w.buttonLayoutObservers.insert(func(window *Window, cx *App) bool {
		callback(window, cx)
		return true
	})
}

// ReplaceRoot replaces the window's root view; *V must implement Render[V].
func (w *Window) ReplaceRoot[V any](cx *App, buildView func(window *Window, cx *Context[V]) *V) Entity[V] {
	mustRender[V]()
	view := cx.New(func(cx *Context[V]) *V { return buildView(w, cx) })
	if old, ok := w.root.Get(); ok {
		old.Release()
	}
	w.root = Some(AnyViewFromEntity(view.Clone()))
	w.Refresh()
	return view
}

// Root returns the root view of the window as an Entity[V], which the caller releases. The bool
// reports whether the window has a root; the Option is None when the root is not a V.
func (w *Window) Root[V any]() (Option[Entity[V]], bool) {
	mustRender[V]()
	view, ok := w.root.Get()
	if !ok {
		return None[Entity[V]](), false
	}
	clone := view.Clone()
	entity, ok := clone.Downcast[V]()
	if !ok {
		clone.Release()
		return None[Entity[V]](), true
	}
	return Some(entity), true
}

// WindowHandle returns a handle to the window.
func (w *Window) WindowHandle() AnyWindowHandle {
	return w.handle
}

// Refresh marks the window dirty, scheduling it to be redrawn on the next frame.
func (w *Window) Refresh() {
	if w.invalidator.notDrawing() {
		w.refreshing = true
		w.invalidator.setDirty(true)
	}
}

// RemoveWindow closes the window.
func (w *Window) RemoveWindow() {
	w.removed = true
}

// Focused returns the currently focused handle, which the caller releases.
func (w *Window) Focused(cx *App) (FocusHandle, bool) {
	id, ok := w.focus.Get()
	if !ok {
		return FocusHandle{}, false
	}
	return focusHandleForID(id, cx.focusHandles)
}

// FocusLostRestoreTarget returns, while focus-lost listeners are dispatched, the closest ancestor
// of the previously focused element that can still receive focus; the caller releases it.
func (w *Window) FocusLostRestoreTarget(cx *App) (FocusHandle, bool) {
	if len(w.focusLostPath) == 0 {
		return FocusHandle{}, false
	}
	ancestors := w.focusLostPath[:len(w.focusLostPath)-1]
	for i := len(ancestors) - 1; i >= 0; i-- {
		id := ancestors[i]
		if _, ok := w.renderedFrame.dispatchTree.focusableNodeID(id); !ok {
			continue
		}
		if handle, ok := focusHandleForID(id, cx.focusHandles); ok {
			return handle, true
		}
	}
	return FocusHandle{}, false
}

// Focus moves focus to the element associated with handle.
func (w *Window) Focus(handle FocusHandle, cx *App) {
	if !w.focusEnabled || w.focus == Some(handle.id) {
		return
	}

	w.endCompositionBeforeFocusingNativeView(handle.id, cx)
	w.focus = Some(handle.id)
	w.syncNativeViewFocus()
	w.focusGeneration++
	w.clearPendingKeystrokes(cx)

	w.Refresh()
}

// Blur removes focus from all elements within the window.
func (w *Window) Blur(cx *App) {
	w.clearPendingKeystrokes(cx)

	if !w.focusEnabled {
		return
	}

	if w.focus.IsSome() {
		w.focusGeneration++
	}
	w.focus = None[FocusID]()
	w.syncNativeViewFocus()
	w.Refresh()
}

// DisableFocus blurs the window and doesn't allow anything in it to be focused again.
func (w *Window) DisableFocus(cx *App) {
	w.Blur(cx)
	w.focusEnabled = false
}

// FocusNext moves focus to the next tab stop.
func (w *Window) FocusNext(cx *App) {
	if !w.focusEnabled {
		return
	}

	if handle, ok := w.renderedFrame.tabStops.next(w.focus); ok {
		defer handle.Release()
		w.Focus(handle, cx)
	}
}

// FocusPrev moves focus to the previous tab stop.
func (w *Window) FocusPrev(cx *App) {
	if !w.focusEnabled {
		return
	}

	if handle, ok := w.renderedFrame.tabStops.prev(w.focus); ok {
		defer handle.Release()
		w.Focus(handle, cx)
	}
}

// TextSystem returns the window's text system.
func (w *Window) TextSystem() *WindowTextSystem {
	return w.textSystem
}

// TextStyle returns the current text style, composed of every refinement pushed with
// WithTextStyle.
func (w *Window) TextStyle() TextStyle {
	style := DefaultTextStyle()
	for i := range w.textStyleStack {
		style.Refine(&w.textStyleStack[i])
	}
	return style
}

// IsMaximized reports whether the platform window is maximized.
func (w *Window) IsMaximized() bool {
	return w.platformWindow.IsMaximized()
}

// RequestDecorations requests a window decoration mode (Wayland).
func (w *Window) RequestDecorations(decorations WindowDecorations) {
	w.platformWindow.RequestDecorations(decorations)
}

// SetExclusiveZone sets the exclusive zone of a layer-shell surface (Wayland layer-shell
// windows only).
func (w *Window) SetExclusiveZone(zone Pixels) {
	w.platformWindow.SetExclusiveZone(zone)
}

// SetExclusiveEdge sets which anchored edge a layer-shell surface's exclusive zone applies to
// (Wayland layer-shell windows only).
func (w *Window) SetExclusiveEdge(edge LayerShellAnchor) {
	w.platformWindow.SetExclusiveEdge(edge)
}

// StartWindowResize starts an interactive resize if the window is resizable.
func (w *Window) StartWindowResize(edge ResizeEdge) {
	if w.isResizable {
		w.platformWindow.StartWindowResize(edge)
	}
}

// SetInputRegion sets the area that receives pointer and touch input (Wayland). None resets it
// to the whole window; an empty slice receives no input.
func (w *Window) SetInputRegion(region Option[[]Bounds[Pixels]]) {
	w.platformWindow.SetInputRegion(region)
}

// WindowBounds returns how the window should be opened after it has been closed.
func (w *Window) WindowBounds() WindowBounds {
	return w.platformWindow.WindowBounds()
}

// InnerWindowBounds returns the window bounds excluding insets (Wayland and X11).
func (w *Window) InnerWindowBounds() WindowBounds {
	return w.platformWindow.InnerWindowBounds()
}

// NativeWindowState encodes the window's native restorable state into an opaque blob.
func (w *Window) NativeWindowState() ([]byte, bool) {
	return w.platformWindow.NativeWindowState()
}

// RestoreNativeWindowState restores state produced by NativeWindowState.
func (w *Window) RestoreNativeWindowState(state []byte) {
	w.platformWindow.RestoreNativeWindowState(state)
}

// DispatchAction dispatches action on the currently focused element.
func (w *Window) DispatchAction(action Action, cx *App) {
	focusID := None[FocusID]()
	if handle, ok := w.Focused(cx); ok {
		focusID = Some(handle.id)
		handle.Release()
	}

	window := w.handle
	cx.Defer(func(cx *App) {
		logErrContext(window.Update(cx, func(_ AnyView, window *Window, cx *App) {
			nodeID := window.focusNodeIDInRenderedFrame(focusID)
			window.dispatchActionOnNode(nodeID, action, cx)
		}), "dispatching action for window %#v", window.WindowID())
	})
}

func (w *Window) dispatchKeystrokeObservers(keystroke *Keystroke, inputPreference InputPreference, action Action, contextStack []KeyContext, cx *App) {
	cx.keystrokeObservers.retain(func(callback *keystrokeObserver) bool {
		boxed := None[Action]()
		if action != nil {
			boxed = Some(action.BoxedClone())
		}
		return (*callback)(&KeystrokeEvent{
			Keystroke:       *keystroke,
			InputPreference: inputPreference,
			Action:          boxed,
			ContextStack:    slices.Clone(contextStack),
		}, w, cx)
	})
}

func (w *Window) dispatchKeystrokeInterceptors(keystroke *Keystroke, inputPreference InputPreference, contextStack []KeyContext, cx *App) {
	cx.keystrokeInterceptors.retain(func(callback *keystrokeObserver) bool {
		return (*callback)(&KeystrokeEvent{
			Keystroke:       *keystroke,
			InputPreference: inputPreference,
			Action:          None[Action](),
			ContextStack:    slices.Clone(contextStack),
		}, w, cx)
	})
}

// Defer schedules f to run at the end of the current effect cycle, allowing entities that are
// currently on the stack to be returned to the app.
func (w *Window) Defer(cx *App, f func(window *Window, cx *App)) {
	handle := w.handle
	cx.Defer(func(cx *App) {
		_ = handle.Update(cx, func(_ AnyView, window *Window, cx *App) { f(window, cx) })
	})
}

// Observe invokes onNotify, with this window, whenever observed is notified.
func (w *Window) Observe[T any](observed Entity[T], cx *App, onNotify func(entity Entity[T], window *Window, cx *App)) Subscription {
	entityID := observed.EntityID()
	weak := observed.Downgrade()
	windowHandle := w.handle
	return cx.newObserver(entityID, func(cx *App) bool {
		result, err := windowHandle.UpdateR(cx, func(_ AnyView, window *Window, cx *App) bool {
			handle, ok := weak.Upgrade()
			if !ok {
				return false
			}
			defer handle.Release()
			onNotify(handle, window, cx)
			return true
		})
		if err != nil {
			return false
		}
		return result
	})
}

// Subscribe invokes onEvent, with this window, whenever entity emits an event of type E.
func (w *Window) Subscribe[T, E any](entity Entity[T], cx *App, onEvent func(entity Entity[T], event *E, window *Window, cx *App)) Subscription {
	entityID := entity.EntityID()
	handle := entity.Downgrade()
	windowHandle := w.handle
	return cx.newSubscription(entityID, eventListener{
		eventType: reflect.TypeFor[E](),
		listener: func(event any, cx *App) bool {
			result, err := windowHandle.UpdateR(cx, func(_ AnyView, window *Window, cx *App) bool {
				entity, ok := handle.Upgrade()
				if !ok {
					return false
				}
				defer entity.Release()
				typed, ok := event.(*E)
				if !ok {
					panic("invalid event type")
				}
				onEvent(entity, typed, window, cx)
				return true
			})
			if err != nil {
				return false
			}
			return result
		},
	})
}

// ObserveRelease registers onRelease to be invoked, with this window, when entity is released.
func (w *Window) ObserveRelease[T any](entity Entity[T], cx *App, onRelease func(state *T, window *Window, cx *App)) Subscription {
	entityID := entity.EntityID()
	windowHandle := w.handle
	return cx.releaseListeners.insert(entityID, func(entity any, cx *App) {
		state, ok := entity.(*T)
		if !ok {
			panic("invalid entity type")
		}
		_ = windowHandle.Update(cx, func(_ AnyView, window *Window, cx *App) { onRelease(state, window, cx) })
	})
}

// ToAsync creates an AsyncWindowContext, which can be held across await points.
func (w *Window) ToAsync(cx *App) *AsyncWindowContext {
	return newAsyncWindowContext(cx.ToAsync(), w.handle)
}

// OnNextFrame schedules callback to run directly after the current frame is rendered.
func (w *Window) OnNextFrame(callback func(window *Window, cx *App)) {
	*w.nextFrameCallbacks = append(*w.nextFrameCallbacks, callback)
	w.platformWindow.ScheduleFrame()
	w.invalidator.wakePlatform()
}

// RequestAnimationFrame schedules a frame to be drawn on the next animation frame. Called from
// within a view, it notifies that view on the next frame.
func (w *Window) RequestAnimationFrame() {
	entity := w.CurrentView()
	w.OnNextFrame(func(_ *Window, cx *App) { cx.Notify(entity) })
}

func (w *Window) advanceSceneAnimations(cx *App) {
	elapsed := cx.BackgroundExecutor().Now().Sub(cx.syncedAnimationEpoch)
	w.renderedFrame.scene.advanceAnimations(elapsed)
}

// SimulateNextFrame runs every callback scheduled with OnNextFrame and returns how many ran.
func (w *Window) SimulateNextFrame(cx *App) int {
	callbacks := *w.nextFrameCallbacks
	*w.nextFrameCallbacks = nil
	count := len(callbacks)
	for _, callback := range callbacks {
		callback(w, cx)
	}
	return count
}

// Spawn spawns f on the main thread with an AsyncWindowContext for this window.
func (w *Window) Spawn[R any](cx *App, f func(cx *AsyncWindowContext) (R, error)) *Task[R] {
	handle := w.handle
	return cx.Spawn(func(app *AsyncApp) (R, error) {
		asyncWindowCx := newAsyncWindowContext(app, handle)
		return f(asyncWindowCx)
	})
}

// SpawnWithPriority spawns f on the main thread with the given priority and an
// AsyncWindowContext for this window.
func (w *Window) SpawnWithPriority[R any](priority Priority, cx *App, f func(cx *AsyncWindowContext) (R, error)) *Task[R] {
	handle := w.handle
	return cx.SpawnWithPriority(priority, func(app *AsyncApp) (R, error) {
		asyncWindowCx := newAsyncWindowContext(app, handle)
		return f(asyncWindowCx)
	})
}

// BoundsChanged updates the viewport size and scale factor from the platform window and
// notifies bounds observers.
func (w *Window) BoundsChanged(cx *App) {
	w.scaleFactor = w.platformWindow.ScaleFactor()
	w.viewportSize = w.platformWindow.ContentSize()
	w.lastWindowBounds = w.platformWindow.WindowBounds()
	cx.syncWindowUpdateEntry(w)
	w.mousePosition = w.platformWindow.MousePosition()

	w.Refresh()

	w.boundsObservers.retain(func(callback *anyObserver) bool { return (*callback)(w, cx) })
}

func windowDisplayChanged(handle AnyWindowHandle, cx *App) {
	movedTo, err := handle.UpdateR(cx, func(_ AnyView, window *Window, _ *App) Option[Option[DisplayID]] {
		displayID := None[DisplayID]()
		if display := window.platformWindow.Display(); display != nil {
			displayID = Some(display.ID())
		}
		moved := displayID != window.displayID
		window.displayID = displayID
		if moved {
			return Some(displayID)
		}
		return None[Option[DisplayID]]()
	})
	logErrContext(err, "handling display change for window %#v", handle.WindowID())
	if err != nil {
		return
	}
	displayID, ok := movedTo.Get()
	if !ok {
		return
	}
	if id, ok := displayID.Get(); ok && !cx.knowsDisplay(id) {
		cx.displaysChanged()
	}
	logErrContext(handle.Update(cx, func(_ AnyView, window *Window, cx *App) {
		window.notifyDisplayObservers(cx)
	}), "notifying display observers for window %#v", handle.WindowID())
}

func (w *Window) notifyDisplayObservers(cx *App) {
	w.displayObservers.retain(func(callback *anyObserver) bool { return (*callback)(w, cx) })
}

// RefreshInterval returns the time between refreshes of the window's display, as of the
// platform's last display change notification.
func (w *Window) RefreshInterval(cx *App) (time.Duration, bool) {
	id, ok := w.displayID.Get()
	if !ok {
		return 0, false
	}
	return cx.displayRefreshInterval(id)
}

// ObserveWindowDisplay registers a callback invoked when the window moves to another display or
// its display's properties change.
func (w *Window) ObserveWindowDisplay(callback func(window *Window, cx *App)) Subscription {
	return w.displayObservers.insert(func(window *Window, cx *App) bool {
		callback(window, cx)
		return true
	})
}

// Bounds returns the bounds of the window in the global coordinate space.
func (w *Window) Bounds() Bounds[Pixels] {
	return w.platformWindow.Bounds()
}

// Insets returns the regions of this window obscured by system UI and the keyboard.
func (w *Window) Insets() WindowInsets {
	return w.platformWindow.Insets()
}

// RenderToImage renders the current frame's scene into an RGBA image without presenting it.
func (w *Window) RenderToImage() (*image.RGBA, error) {
	return w.platformWindow.RenderToImage(&w.renderedFrame.scene)
}

// RenderedScene returns the most recently rendered frame's scene; it must not be modified.
func (w *Window) RenderedScene() *Scene {
	return &w.renderedFrame.scene
}

// PaintedQuads returns the quads of the most recently rendered frame's scene, in scaled pixels
// and unclipped.
func (w *Window) PaintedQuads() []Quad {
	return slices.Clone(w.renderedFrame.scene.Quads)
}

// PaintedUnderlines returns the underlines of the most recently rendered frame's scene.
func (w *Window) PaintedUnderlines() []Underline {
	return slices.Clone(w.renderedFrame.scene.Underlines)
}

// Resize sets the content size of the window.
func (w *Window) Resize(size Size[Pixels]) {
	w.platformWindow.Resize(size)
}

// IsFullscreen reports whether the window is fullscreen.
func (w *Window) IsFullscreen() bool {
	return w.platformWindow.IsFullscreen()
}

// IsSimpleFullscreen reports whether the window is in simple (borderless) fullscreen; always
// false on platforms other than macOS.
func (w *Window) IsSimpleFullscreen() bool {
	return w.platformWindow.IsSimpleFullscreen()
}

func (w *Window) appearanceChanged(cx *App) {
	w.appearance = w.platformWindow.Appearance()

	w.appearanceObservers.retain(func(callback *anyObserver) bool { return (*callback)(w, cx) })
}

func (w *Window) buttonLayoutChanged(cx *App) {
	w.buttonLayoutObservers.retain(func(callback *anyObserver) bool { return (*callback)(w, cx) })
}

// Appearance returns the appearance of the window.
func (w *Window) Appearance() WindowAppearance {
	return w.appearance
}

// ViewportSize returns the size of the drawable area within the window.
func (w *Window) ViewportSize() Size[Pixels] {
	return w.viewportSize
}

// VisualViewportBounds returns the platform's visible viewport in window-local logical pixels.
func (w *Window) VisualViewportBounds() Bounds[Pixels] {
	return w.platformWindow.VisualViewportBounds()
}

// FullyVisibleBounds returns a conservative rectangle avoiding platform-known obscured content.
func (w *Window) FullyVisibleBounds() Bounds[Pixels] {
	insets := w.platformWindow.Insets().Effective()
	viewport := w.ViewportSize()
	left := min(max(insets.Left, 0), viewport.Width)
	top := min(max(insets.Top, 0), viewport.Height)
	right := max(viewport.Width-max(insets.Right, 0), left)
	bottom := max(viewport.Height-max(insets.Bottom, 0), top)
	safeBounds := BoundsFromCorners(NewPoint(left, top), NewPoint(right, bottom))
	visible := safeBounds.Intersect(w.VisualViewportBounds())
	visible.Size.Width = max(visible.Size.Width, 0)
	visible.Size.Height = max(visible.Size.Height, 0)
	return visible
}

// RequestVirtualKeyboard requests the virtual keyboard for the focused text input.
func (w *Window) RequestVirtualKeyboard() {
	w.platformWindow.ShowSoftKeyboard()
}

// DismissVirtualKeyboard requests dismissal of the virtual keyboard without changing focus.
func (w *Window) DismissVirtualKeyboard() {
	w.platformWindow.HideSoftKeyboard()
}

// IsWindowActive reports whether the window is focused by the operating system.
func (w *Window) IsWindowActive() bool {
	return *w.active
}

// IsWindowHovered reports whether the window owns the mouse cursor; on macOS it is
// IsWindowActive.
func (w *Window) IsWindowHovered() bool {
	switch runtime.GOOS {
	case "windows", "linux", "freebsd":
		return *w.hovered
	}
	return w.IsWindowActive()
}

// ZoomWindow toggles zoom on the window.
func (w *Window) ZoomWindow() {
	w.platformWindow.Zoom()
}

// ShowWindowMenu opens the native title bar context menu (Wayland and X11).
func (w *Window) ShowWindowMenu(position Point[Pixels]) {
	w.platformWindow.ShowWindowMenu(position)
}

// StartWindowMove tells the compositor to take control of window movement (Wayland and X11).
func (w *Window) StartWindowMove() {
	w.platformWindow.StartWindowMove()
}

// SetClientInset sets the width of the invisible client-side decorations (Wayland and X11).
func (w *Window) SetClientInset(inset Pixels) {
	w.clientInset = Some(inset)
	w.platformWindow.SetClientInset(inset)
}

// SetClientCornerRadii sets the client-drawn corner radii so platform effects match the shape.
func (w *Window) SetClientCornerRadii(radii Corners[Pixels]) {
	w.platformWindow.SetClientCornerRadii(radii)
}

// ClientInset returns the value set with SetClientInset.
func (w *Window) ClientInset() (Pixels, bool) {
	return w.clientInset.Get()
}

// WindowDecorations reports whether the window controls must be rendered by the application.
func (w *Window) WindowDecorations() Decorations {
	return w.platformWindow.WindowDecorations()
}

// IsResizable reports whether the window is resizable.
func (w *Window) IsResizable() bool {
	return w.isResizable
}

// IsMinimizable reports whether the window is minimizable.
func (w *Window) IsMinimizable() bool {
	return w.isMinimizable
}

// WindowControls returns the controls supported by the platform.
func (w *Window) WindowControls() WindowControls {
	return w.platformWindow.WindowControls()
}

// SetWindowTitle updates the window's title at the platform level.
func (w *Window) SetWindowTitle(title string) {
	w.platformWindow.SetTitle(title)
	w.a11y.setWindowTitle(SharedString(title))
}

// SetTrafficLightPosition sets the position of the macOS traffic light buttons.
func (w *Window) SetTrafficLightPosition(position Point[Pixels]) {
	w.platformWindow.SetTrafficLightPosition(position)
}

// SetNativeTitlebarVisibility shows or hides the native macOS titlebar.
func (w *Window) SetNativeTitlebarVisibility(visible bool) {
	w.platformWindow.SetNativeTitlebarVisibility(visible)
}

// SetAppID sets the application identifier.
func (w *Window) SetAppID(appID string) {
	w.platformWindow.SetAppID(appID)
}

// SetBackgroundAppearance sets the window background appearance.
func (w *Window) SetBackgroundAppearance(backgroundAppearance WindowBackgroundAppearance) {
	w.platformWindow.SetBackgroundAppearance(backgroundAppearance)
}

// SetWindowEdited marks the window as edited at the platform level.
func (w *Window) SetWindowEdited(edited bool) {
	w.platformWindow.SetEdited(edited)
}

// SetDocumentPath sets the path of the file this window represents.
func (w *Window) SetDocumentPath(path Option[string]) {
	w.platformWindow.SetDocumentPath(path)
}

// Display returns the display the window is on, or nil.
func (w *Window) Display(cx *App) PlatformDisplay {
	for _, display := range cx.platform.Displays() {
		if Some(display.ID()) == w.displayID {
			return display
		}
	}
	return nil
}

// ShowCharacterPalette shows the platform character palette.
func (w *Window) ShowCharacterPalette() {
	w.platformWindow.ShowCharacterPalette()
}

// ScaleFactor returns the scale factor of the window's display.
func (w *Window) ScaleFactor() float32 {
	return w.scaleFactor
}

// SetScaleFactor overrides the display scale factor for tests.
func (w *Window) SetScaleFactor(scaleFactor float32) {
	w.scaleFactor = scaleFactor
	w.Refresh()
}

// RemSize returns the size of an em for the base font of the application.
func (w *Window) RemSize() Pixels {
	if n := len(w.remSizeOverrideStack); n != 0 {
		return w.remSizeOverrideStack[n-1]
	}
	return w.remSize
}

// SetRemSize sets the size of an em for the base font of the application.
func (w *Window) SetRemSize(remSize Pixels) {
	w.remSize = remSize
}

// WithGlobalID calls f with a globally unique identifier for elementID, valid only during f.
func (w *Window) WithGlobalID[R any](elementID ElementID, f func(globalID *GlobalElementID, window *Window) R) R {
	return w.WithID(elementID, func(window *Window) R {
		globalID := window.elementIDStack.globalID()
		return f(&globalID, window)
	})
}

// WithID calls f with elementID pushed on the element id stack.
func (w *Window) WithID[R any](elementID ElementID, f func(window *Window) R) R {
	w.elementIDStack.push(elementID)
	result := f(w)
	w.elementIDStack.pop()
	return result
}

// WithRemSize calls f with the given rem size; call it only during element drawing.
func (w *Window) WithRemSize[R any](remSize Option[Pixels], f func(window *Window) R) R {
	w.invalidator.debugAssertPaintOrPrepaint()

	if size, ok := remSize.Get(); ok {
		w.remSizeOverrideStack = append(w.remSizeOverrideStack, size)
		result := f(w)
		w.remSizeOverrideStack = w.remSizeOverrideStack[:len(w.remSizeOverrideStack)-1]
		return result
	}
	return f(w)
}

// LineHeight returns the line height of the current text style.
func (w *Window) LineHeight() Pixels {
	return w.TextStyle().LineHeightInPixels(w.RemSize())
}

// PixelSnap rounds a logical value to the nearest device pixel.
func (w *Window) PixelSnap(value Pixels) Pixels {
	return Px(roundToDevicePixel(float32(value), w.ScaleFactor()) / w.ScaleFactor())
}

// PixelSnapF64 is the float64 variant of PixelSnap.
func (w *Window) PixelSnapF64(value float64) float64 {
	scaleFactor := float64(w.ScaleFactor())
	return roundHalfTowardZeroF64(value*scaleFactor) / scaleFactor
}

// PixelSnapBounds snaps a bounds' origin and size to the nearest device pixel.
func (w *Window) PixelSnapBounds(bounds Bounds[Pixels]) Bounds[Pixels] {
	return bounds.Map(w.PixelSnap)
}

// PixelSnapPoint snaps a point's coordinates to the nearest device pixel.
func (w *Window) PixelSnapPoint(position Point[Pixels]) Point[Pixels] {
	return position.Map(w.PixelSnap)
}

func (w *Window) devicePixel(scaleFactor float32) func(c Pixels) ScaledPixels {
	return func(c Pixels) ScaledPixels {
		return ScaledPixels(roundToDevicePixel(float32(c), scaleFactor))
	}
}

// UnderlineBounds returns the snapped device-space bounds used to paint an underline.
func (w *Window) UnderlineBounds(origin Point[Pixels], width Pixels, style UnderlineStyle) Bounds[ScaledPixels] {
	scaleFactor := w.ScaleFactor()
	thickness := w.snapStroke(style.Thickness)
	height := thickness
	if style.Wavy {
		height = ScaledPixels(float32(thickness) * 3.)
	}
	return Bounds[ScaledPixels]{
		Origin: origin.Map(w.devicePixel(scaleFactor)),
		Size:   Size[ScaledPixels]{Width: w.snapStroke(width), Height: height},
	}
}

// PaintUnderlineWithExclusions paints an underline excluding absolute device-space horizontal
// spans.
func (w *Window) PaintUnderlineWithExclusions(origin Point[Pixels], width Pixels, style UnderlineStyle, exclusions []ScaledPixelsRange) {
	w.invalidator.debugAssertPaint()
	underline := w.underline(origin, width, style)
	if len(exclusions) == 0 {
		w.nextFrame.scene.InsertPrimitive(underline)
		return
	}

	bounds := underline.Bounds.Intersect(underline.ContentMask.Bounds)
	if bounds.IsEmpty() {
		return
	}

	spans := make([]ScaledPixelsRange, 0, 4)
	for _, span := range exclusions {
		start := max(span.Start, bounds.Left())
		end := min(span.End, bounds.Right())
		if start < end {
			spans = append(spans, ScaledPixelsRange{Start: start, End: end})
		}
	}
	if len(spans) == 0 {
		w.nextFrame.scene.InsertPrimitive(underline)
		return
	}
	slices.SortFunc(spans, func(a, b ScaledPixelsRange) int { return cmp.Compare(a.Start, b.Start) })

	paintSpan := func(start, end ScaledPixels) {
		span := underline
		span.ContentMask = ContentMask[ScaledPixels]{
			Bounds: BoundsFromCorners(NewPoint(start, bounds.Top()), NewPoint(end, bounds.Bottom())),
		}
		w.nextFrame.scene.InsertPrimitive(span)
	}
	start := bounds.Left()
	for _, exclusion := range spans {
		if start < exclusion.Start {
			paintSpan(start, exclusion.Start)
		}
		start = max(start, exclusion.End)
	}
	if start < bounds.Right() {
		paintSpan(start, bounds.Right())
	}
}

func (w *Window) underline(origin Point[Pixels], width Pixels, style UnderlineStyle) Underline {
	return Underline{
		Order:       0,
		Pad:         0,
		Bounds:      w.UnderlineBounds(origin, width, style),
		ContentMask: w.snappedContentMask(),
		Color:       style.Color.UnwrapOrDefault().Opacity(w.elementOpacity()),
		Thickness:   w.snapStroke(style.Thickness),
		Wavy:        PaddedBool32FromBool(style.Wavy),
	}
}

func (w *Window) snapBounds(bounds Bounds[Pixels]) Bounds[ScaledPixels] {
	scaleFactor := w.ScaleFactor()
	left := roundToDevicePixel(float32(bounds.Left()), scaleFactor)
	top := roundToDevicePixel(float32(bounds.Top()), scaleFactor)
	right := max(roundToDevicePixel(float32(bounds.Right()), scaleFactor), left)
	bottom := max(roundToDevicePixel(float32(bounds.Bottom()), scaleFactor), top)
	return BoundsFromCorners(
		NewPoint(ScaledPixels(left), ScaledPixels(top)),
		NewPoint(ScaledPixels(right), ScaledPixels(bottom)),
	)
}

func (w *Window) snapStroke(value Pixels) ScaledPixels {
	return ScaledPixels(roundStrokeToDevicePixel(float32(value), w.ScaleFactor()))
}

func (w *Window) snapBorderWidths(edges Edges[Pixels]) Edges[ScaledPixels] {
	return edges.Map(w.snapStroke)
}

func (w *Window) coverBounds(bounds Bounds[Pixels]) Bounds[ScaledPixels] {
	scaleFactor := w.ScaleFactor()
	left := floorToDevicePixel(float32(bounds.Left()), scaleFactor)
	top := floorToDevicePixel(float32(bounds.Top()), scaleFactor)
	if bounds.IsEmpty() {
		return NewBounds(NewPoint(ScaledPixels(left), ScaledPixels(top)), Size[ScaledPixels]{})
	}
	right := max(ceilToDevicePixel(float32(bounds.Right()), scaleFactor), left)
	bottom := max(ceilToDevicePixel(float32(bounds.Bottom()), scaleFactor), top)
	return BoundsFromCorners(
		NewPoint(ScaledPixels(left), ScaledPixels(top)),
		NewPoint(ScaledPixels(right), ScaledPixels(bottom)),
	)
}

func (w *Window) snappedContentMask() ContentMask[ScaledPixels] {
	return ContentMask[ScaledPixels]{Bounds: w.coverBounds(w.ContentMask().Bounds)}
}

// PreventDefault prevents the default action of the event being dispatched.
func (w *Window) PreventDefault() {
	w.defaultPrevented = true
}

// DefaultPrevented reports whether the default action of the event being dispatched was
// prevented.
func (w *Window) DefaultPrevented() bool {
	return w.defaultPrevented
}

// IsActionAvailable reports whether action has a handler along the dispatch path to the focused
// element.
func (w *Window) IsActionAvailable(action Action, cx *App) bool {
	focusID := None[FocusID]()
	if handle, ok := w.Focused(cx); ok {
		focusID = Some(handle.id)
		handle.Release()
	}
	nodeID := w.focusNodeIDInRenderedFrame(focusID)
	return w.renderedFrame.dispatchTree.isActionAvailable(action, nodeID)
}

// IsActionAvailableIn reports whether action has a handler along the dispatch path to
// focusHandle.
func (w *Window) IsActionAvailableIn(action Action, focusHandle FocusHandle) bool {
	nodeID := w.focusNodeIDInRenderedFrame(Some(focusHandle.id))
	return w.renderedFrame.dispatchTree.isActionAvailable(action, nodeID)
}

// MousePosition returns the mouse position relative to the window.
func (w *Window) MousePosition() Point[Pixels] {
	return w.mousePosition
}

// CapturePointer captures the pointer for hitboxID until the next mouse up or ReleasePointer.
func (w *Window) CapturePointer(hitboxID HitboxID) {
	w.capturedHitbox = Some(hitboxID)
}

// ReleasePointer releases any active pointer capture.
func (w *Window) ReleasePointer() {
	w.capturedHitbox = None[HitboxID]()
}

// CapturedHitbox returns the hitbox that has captured the pointer, if any.
func (w *Window) CapturedHitbox() (HitboxID, bool) {
	return w.capturedHitbox.Get()
}

// CaptureLongPress captures the current long press for entity.
func (w *Window) CaptureLongPress[T any](entity Entity[T]) {
	w.longPressCapture = Some(entity.EntityID())
}

// HasLongPressCapture reports whether entity has captured the current long press.
func (w *Window) HasLongPressCapture[T any](entity Entity[T]) bool {
	return w.longPressCapture == Some(entity.EntityID())
}

// Modifiers returns the current state of the keyboard's modifiers.
func (w *Window) Modifiers() Modifiers {
	return w.modifiers
}

// LastInputWasKeyboard reports whether the last input event was keyboard-based.
func (w *Window) LastInputWasKeyboard() bool {
	return w.lastInputModality == inputModalityKeyboard
}

// LastInputWasTouch reports whether the last input event came from a touch screen.
func (w *Window) LastInputWasTouch() bool {
	return w.lastInputModality == inputModalityTouch
}

// Capslock returns the current state of the keyboard's capslock.
func (w *Window) Capslock() Capslock {
	return w.capslock
}

// Draw produces a new frame and makes it the rendered frame; Present shows it.
func (w *Window) Draw(cx *App) ArenaClearNeeded {
	var frameDirty frameDirtyAccumulator
	if profilerFeatureEnabled {
		frameDirty = w.invalidator.takeFrameDirty()
		w.windowProfiler.BeginDraw()
	}

	arenaScope := enterElementArenaScope(cx.elementArena())
	defer arenaScope.End()

	if w.platformWindow.PrepareFrame() {
		w.Refresh()
	}
	w.invalidateEntities()
	cx.entities.clearAccessed()
	if gpuiutil.DebugAssertions && len(w.renderedEntityStack) != 0 {
		panic("assertion failed: self.rendered_entity_stack.is_empty()")
	}
	w.invalidator.setDirty(false)
	w.requestedAutoscroll = None[Bounds[Pixels]]()

	if inputHandler := w.platformWindow.TakeInputHandler(); inputHandler != nil {
		restored := false
		for i := len(w.renderedFrame.inputHandlers) - 1; i >= 0; i-- {
			if w.renderedFrame.inputHandlers[i] == nil {
				w.renderedFrame.inputHandlers[i] = inputHandler
				restored = true
				break
			}
		}
		if !restored {
			w.renderedFrame.inputHandlers = append(w.renderedFrame.inputHandlers, inputHandler)
		}
	}
	if !cx.mode.shouldSkipDrawing() {
		w.drawRoots(cx)
		if profilerFeatureEnabled {
			viewportSize := w.viewportSize
			scaleFactor := w.ScaleFactor()
			w.debugFrameOverlay.paint(&w.nextFrame.scene, viewportSize, scaleFactor)
		}
	}
	clear(w.dirtyViews)
	w.nextFrame.windowActive = *w.active

	focusedTextInput := None[FocusID]()
	for i := len(w.nextFrame.inputHandlers) - 1; i >= 0; i-- {
		inputHandler := w.nextFrame.inputHandlers[i]
		if inputHandler == nil {
			continue
		}
		w.nextFrame.inputHandlers[i] = nil
		acceptsTextInput := inputHandler.AcceptsTextInput(w, cx)
		w.platformWindow.SetInputHandler(inputHandler)
		if acceptsTextInput {
			focusedTextInput = w.focus
		}
		break
	}
	w.applyTextInputConfiguration(cx)
	previous := w.focusedTextInput
	w.focusedTextInput = focusedTextInput
	if previous.IsSome() != focusedTextInput.IsSome() {
		change := TextInputStateChangeFocusLost
		if focusedTextInput.IsSome() {
			change = TextInputStateChangeFocusGained
		}
		w.platformWindow.TextInputStateChanged(change)
	}

	w.layoutEngine.Clear()
	w.TextSystem().finishFrame()
	w.nextFrame.finish(w.renderedFrame)

	w.invalidator.setPhase(drawPhaseFocus)
	previousFocusPath := w.renderedFrame.focusPath()
	previousWindowActive := w.renderedFrame.windowActive
	w.renderedFrame, w.nextFrame = w.nextFrame, w.renderedFrame
	*w.sceneAnimationInterval = OptionFrom(w.renderedFrame.scene.animationFrameInterval())
	w.nextFrame.clear()
	currentFocusPath := w.renderedFrame.focusPath()
	currentWindowActive := w.renderedFrame.windowActive
	focusBeforeListeners := w.focus

	if !slices.Equal(previousFocusPath, currentFocusPath) || previousWindowActive != currentWindowActive {
		if len(previousFocusPath) != 0 && len(currentFocusPath) == 0 {
			w.focusLostPath = slices.Clone(previousFocusPath)
			w.focusLostListeners.retain(func(listener *anyObserver) bool { return (*listener)(w, cx) })
			w.focusLostPath = nil
			focusBeforeListeners = w.focus
		}

		var event windowFocusEvent
		if previousWindowActive {
			event.previousFocusPath = previousFocusPath
		}
		if currentWindowActive {
			event.currentFocusPath = currentFocusPath
		}
		w.focusListeners.retain(func(listener *anyWindowFocusListener) bool { return (*listener)(&event, w, cx) })
	}

	if gpuiutil.DebugAssertions && len(w.renderedEntityStack) != 0 {
		panic("assertion failed: self.rendered_entity_stack.is_empty()")
	}
	w.recordEntitiesAccessed(cx)
	w.resetCursorStyle(cx)
	w.refreshing = false
	w.invalidator.setPhase(drawPhaseNone)
	if w.focus != focusBeforeListeners {
		w.Refresh()
	}
	*w.needsPresent = true

	if profilerFeatureEnabled {
		w.windowProfiler.SetRefreshInterval(OptionFrom(w.RefreshInterval(cx)))
		drawDuration := w.windowProfiler.EndDraw(frameDirty.dirtyAt, frameDirty.invalidations)
		w.debugFrameOverlay.recordFrame(drawDuration)
	}

	return arenaScope.exit(cx.elementArena())
}

func (w *Window) recordEntitiesAccessed(cx *App) {
	entities := cx.entities.takeAccessed()
	handle := w.handle
	cx.recordEntitiesAccessed(handle, w.invalidator, entities)
	cx.entities.restoreAccessed(entities)
}

func (w *Window) invalidateEntities() {
	views := w.invalidator.takeViews()
	for entity := range views {
		w.markViewDirty(entity)
	}
	clear(views)
	w.invalidator.replaceViews(views)
}

func (w *Window) present() {
	if profilerFeatureEnabled {
		beginForegroundTurn()
		defer endForegroundTurn()
	}
	presentStart := scheduler.InstantNow()
	w.platformWindow.Draw(&w.renderedFrame.scene)
	if profilerFeatureEnabled {
		w.windowProfiler.RecordPresent(presentStart, scheduler.InstantNow(), *w.active, len(*w.nextFrameCallbacks) != 0)
	}
	*w.needsPresent = false
}

func (w *Window) presentIfNeeded() {
	if *w.needsPresent {
		w.present()
	}
}

// InputLatencySnapshot returns a snapshot of the input-latency histograms (gpui_profiler).
func (w *Window) InputLatencySnapshot() InputLatencySnapshot {
	return w.windowProfiler.InputLatencySnapshot()
}

// FrameDurationSnapshot returns a snapshot of the frame-duration histograms (gpui_profiler).
func (w *Window) FrameDurationSnapshot() FrameDurationSnapshot {
	return w.windowProfiler.FrameDurationSnapshot()
}

// DebugFrameOverlayMode returns the mode of the debug frame overlay (gpui_profiler).
func (w *Window) DebugFrameOverlayMode() DebugFrameOverlayMode {
	return w.debugFrameOverlay.mode()
}

// SetDebugFrameOverlayMode sets the mode of the debug frame overlay and schedules a redraw
// (gpui_profiler).
func (w *Window) SetDebugFrameOverlayMode(mode DebugFrameOverlayMode) {
	w.debugFrameOverlay.setMode(mode)
	w.Refresh()
}

// CycleDebugFrameOverlayMode advances the debug frame overlay through its modes
// (gpui_profiler).
func (w *Window) CycleDebugFrameOverlayMode() {
	w.SetDebugFrameOverlayMode(w.debugFrameOverlay.mode().Next())
}

// ResetDebugFrameOverlayStats clears the debug frame overlay's frame-time statistics, except the
// total frame count, and schedules a redraw (gpui_profiler).
func (w *Window) ResetDebugFrameOverlayStats() {
	w.debugFrameOverlay.resetStats()
	w.Refresh()
}

func (w *Window) drawRoots(cx *App) {
	w.invalidator.setPhase(drawPhasePrepaint)
	w.tooltipBounds.Take()

	w.a11y.syncActiveFlag()
	if w.a11y.isActive() {
		w.a11y.beginFrame()
	}

	inspectorWidth := Rems(30.0).ToPixels(w.RemSize())
	rootSize := w.viewportSize
	if w.inspector.IsSome() {
		rootSize.Width = max(rootSize.Width-inspectorWidth, Px(0.0))
	}

	scaleFactor := w.ScaleFactor()
	var requestLayoutStart, prepaintStart, paintStart scheduler.Instant
	if profilerFeatureEnabled {
		requestLayoutStart = scheduler.InstantNow()
	}
	rootElement := w.root.Unwrap().Clone().IntoAnyElement()
	rootLayoutID := rootElement.RequestLayout(w, cx)
	if profilerFeatureEnabled {
		prepaintStart = scheduler.InstantNow()
	}
	w.layoutEngine.StretchAutoSizeToFill(rootLayoutID, rootSize, scaleFactor)
	rootElement.PrepaintAsRoot(Point[Pixels]{}, SizeAvailableSpaceFromPixels(rootSize), w, cx)

	inspectorElement := w.prepaintInspector(inspectorWidth, cx)

	w.prepaintDeferredDraws(cx)

	promptElement := None[AnyElement]()
	activeDragElement := None[AnyElement]()
	tooltipElement := None[AnyElement]()
	if prompt, ok := w.prompt.Take().Get(); ok {
		element := prompt.view.anyView().IntoAnyElement()
		promptLayoutID := element.RequestLayout(w, cx)
		w.layoutEngine.StretchAutoSizeToFill(promptLayoutID, rootSize, scaleFactor)
		element.PrepaintAsRoot(Point[Pixels]{}, SizeAvailableSpaceFromPixels(rootSize), w, cx)
		promptElement = Some(element)
		w.prompt = Some(prompt)
	} else if activeDrag, ok := cx.activeDrag.Take().Get(); ok {
		element := activeDrag.View.Clone().IntoAnyElement()
		offset := w.MousePosition().Sub(activeDrag.CursorOffset)
		element.PrepaintAsRoot(offset, AvailableSpaceMinSize(), w, cx)
		activeDragElement = Some(element)
		cx.activeDrag = Some(activeDrag)
	} else {
		tooltipElement = w.prepaintTooltip(cx)
	}

	w.mouseHitTest = w.nextFrame.hitTest(w.mousePosition)

	if profilerFeatureEnabled {
		paintStart = scheduler.InstantNow()
	}
	w.invalidator.setPhase(drawPhasePaint)
	rootElement.Paint(w, cx)

	w.paintInspector(inspectorElement, cx)

	w.paintDeferredDraws(cx)

	if element := promptElement.AsMut(); element != nil {
		element.Paint(w, cx)
	} else if element := activeDragElement.AsMut(); element != nil {
		element.Paint(w, cx)
	} else if element := tooltipElement.AsMut(); element != nil {
		element.Paint(w, cx)
	}

	w.paintInspectorHitbox(cx)

	if profilerFeatureEnabled {
		w.windowProfiler.RecordDrawPhases(DrawPhases{
			RequestLayout: prepaintStart.Sub(requestLayoutStart),
			Prepaint:      paintStart.Sub(prepaintStart),
			Paint:         scheduler.InstantNow().Sub(paintStart),
		})
	}

	a11yActiveStartOfFrame := w.a11y.isActive()
	w.a11y.syncActiveFlag()
	a11yActiveEndOfFrame := w.a11y.isActive()

	shouldSendA11yUpdate := a11yActiveStartOfFrame && a11yActiveEndOfFrame

	if a11yActiveStartOfFrame {
		frameInfo := frameDebugInfo{
			viewportSize: w.viewportSize,
			scaleFactor:  w.scaleFactor,
			tabStopCount: w.nextFrame.tabStops.tabStopCount(),
		}
		treeUpdate := w.a11y.endFrame(frameInfo)

		if shouldSendA11yUpdate {
			slog.Debug(fmt.Sprintf("Sending a11y tree update: %d nodes", len(treeUpdate.Nodes)))
			w.platformWindow.A11yTreeUpdate(treeUpdate)
		}
	}
}

func (w *Window) prepaintTooltip(cx *App) Option[AnyElement] {
	for tooltipRequestIndex := len(w.nextFrame.tooltipRequests) - 1; tooltipRequestIndex >= 0; tooltipRequestIndex-- {
		tooltipRequest := w.nextFrame.tooltipRequests[tooltipRequestIndex]
		if tooltipRequest == nil {
			slog.Error("Unexpectedly absent TooltipRequest")
			continue
		}
		element := tooltipRequest.tooltip.View.Clone().IntoAnyElement()
		mousePosition := tooltipRequest.tooltip.MousePosition
		tooltipSize := element.LayoutAsRoot(AvailableSpaceMinSize(), w, cx)

		bounds := NewBounds(mousePosition.Add(NewPoint(Px(1.), Px(1.))), tooltipSize)
		windowBounds := Bounds[Pixels]{Origin: Point[Pixels]{}, Size: w.ViewportSize()}

		if bounds.Right() > windowBounds.Right() {
			newX := mousePosition.X - bounds.Size.Width - Px(1.)
			if newX >= 0 {
				bounds.Origin.X = newX
			} else {
				bounds.Origin.X = max(0, bounds.Origin.X-bounds.Right()-windowBounds.Right())
			}
		}

		if bounds.Bottom() > windowBounds.Bottom() {
			newY := mousePosition.Y - bounds.Size.Height - Px(1.)
			if newY >= 0 {
				bounds.Origin.Y = newY
			} else {
				bounds.Origin.Y = max(0, bounds.Origin.Y-bounds.Bottom()-windowBounds.Bottom())
			}
		}

		isVisible := tooltipRequest.tooltip.CheckVisibleAndUpdate(bounds, w, cx)
		if !isVisible {
			continue
		}

		w.WithAbsoluteElementOffset(bounds.Origin, func(window *Window) struct{} {
			element.Prepaint(window, cx)
			return struct{}{}
		})

		w.tooltipBounds = Some(tooltipBounds{id: tooltipRequest.id, bounds: bounds})
		return Some(element)
	}
	return None[AnyElement]()
}

func assertElementIDStackEmpty(stack *elementIDStack) {
	if n := len(stack.ids); n != 0 {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: 0", n))
	}
}

func (w *Window) prepaintDeferredDraws(cx *App) {
	assertElementIDStackEmpty(&w.elementIDStack)

	roundStart := 0
	depth := 0
	for {
		roundEnd := len(w.nextFrame.deferredDraws)
		if roundStart == roundEnd {
			break
		}
		if !(depth < 10) {
			panic("Exceeded maximum (10) deferred depth")
		}
		depth++

		traversalOrder := make([]int, 0, roundEnd-roundStart)
		for ix := roundStart; ix < roundEnd; ix++ {
			traversalOrder = append(traversalOrder, ix)
		}
		slices.SortStableFunc(traversalOrder, func(a, b int) int {
			return cmp.Compare(w.nextFrame.deferredDraws[a].priority, w.nextFrame.deferredDraws[b].priority)
		})

		for _, deferredDrawIx := range traversalOrder {
			deferredDraw := &w.nextFrame.deferredDraws[deferredDrawIx]
			w.elementIDStack = deferredDraw.elementIDStack.clone()
			w.textStyleStack = slices.Clone(deferredDraw.textStyleStack)
			element := deferredDraw.element.Take()
			parentNode := deferredDraw.parentNode
			currentView := deferredDraw.currentView
			remSize := deferredDraw.remSize
			absoluteOffset := deferredDraw.absoluteOffset
			prepaintRange := deferredDraw.prepaintRange
			w.nextFrame.dispatchTree.setActiveNode(parentNode)

			prepaintStart := w.prepaintIndex()
			if element, ok := element.Get(); ok {
				_, accessedEntities := cx.detectAccessedEntities(func(cx *App) struct{} {
					return w.withRenderedView(currentView, func(window *Window) struct{} {
						return window.WithRemSize(Some(remSize), func(window *Window) struct{} {
							return window.WithAbsoluteElementOffset(absoluteOffset, func(window *Window) struct{} {
								element.Prepaint(window, cx)
								return struct{}{}
							})
						})
					})
				})
				w.nextFrame.deferredDraws[deferredDrawIx].accessedEntities = accessedEntities
				w.nextFrame.deferredDraws[deferredDrawIx].element = Some(element)
			} else {
				cx.entities.extendAccessed(slices.Collect(maps.Keys(w.nextFrame.deferredDraws[deferredDrawIx].accessedEntities)))
				w.reusePrepaint(prepaintRange.start, prepaintRange.end)
			}
			prepaintEnd := w.prepaintIndex()
			w.nextFrame.deferredDraws[deferredDrawIx].prepaintRange = prepaintStateRange{start: prepaintStart, end: prepaintEnd}
		}

		w.elementIDStack.clear()
		w.textStyleStack = w.textStyleStack[:0]
		roundStart = roundEnd
	}
}

func (w *Window) paintDeferredDraws(cx *App) {
	assertElementIDStackEmpty(&w.elementIDStack)

	if len(w.nextFrame.deferredDraws) == 0 {
		return
	}

	traversalOrder := w.deferredDrawTraversalOrder()
	deferredDraws := w.nextFrame.deferredDraws
	w.nextFrame.deferredDraws = nil
	for _, deferredDrawIx := range traversalOrder {
		deferredDraw := &deferredDraws[deferredDrawIx]
		w.elementIDStack = deferredDraw.elementIDStack.clone()
		w.nextFrame.dispatchTree.setActiveNode(deferredDraw.parentNode)

		paintStart := w.paintIndex()
		contentMask := deferredDraw.contentMask
		if element := deferredDraw.element.AsMut(); element != nil {
			_, accessedEntities := cx.detectAccessedEntities(func(cx *App) struct{} {
				return w.withRenderedView(deferredDraw.currentView, func(window *Window) struct{} {
					return window.WithContentMask(contentMask, func(window *Window) struct{} {
						return window.WithRemSize(Some(deferredDraw.remSize), func(window *Window) struct{} {
							element.Paint(window, cx)
							return struct{}{}
						})
					})
				})
			})
			if deferredDraw.accessedEntities == nil {
				deferredDraw.accessedEntities = collections.HashSet[EntityID]{}
			}
			maps.Copy(deferredDraw.accessedEntities, accessedEntities)
		} else {
			w.reusePaint(deferredDraw.paintRange.start, deferredDraw.paintRange.end)
		}
		paintEnd := w.paintIndex()
		deferredDraw.paintRange = paintRange{start: paintStart, end: paintEnd}
	}
	w.nextFrame.deferredDraws = deferredDraws
	w.elementIDStack.clear()
}

func (w *Window) deferredDrawTraversalOrder() []int {
	deferredCount := len(w.nextFrame.deferredDraws)
	sortedIndices := make([]int, deferredCount)
	for ix := range sortedIndices {
		sortedIndices[ix] = ix
	}
	slices.SortStableFunc(sortedIndices, func(a, b int) int {
		return cmp.Compare(w.nextFrame.deferredDraws[a].priority, w.nextFrame.deferredDraws[b].priority)
	})
	return sortedIndices
}

func (w *Window) prepaintIndex() prepaintStateIndex {
	return prepaintStateIndex{
		hitboxesIndex:              len(w.nextFrame.hitboxes),
		tooltipsIndex:              len(w.nextFrame.tooltipRequests),
		deferredDrawsIndex:         len(w.nextFrame.deferredDraws),
		dispatchTreeIndex:          w.nextFrame.dispatchTree.len(),
		accessedElementStatesIndex: len(w.nextFrame.accessedElementStates),
		lineLayoutIndex:            w.textSystem.layoutIndex(),
	}
}

func (w *Window) reusePrepaint(start, end prepaintStateIndex) {
	w.nextFrame.hitboxes = append(w.nextFrame.hitboxes, w.renderedFrame.hitboxes[start.hitboxesIndex:end.hitboxesIndex]...)
	for i := start.tooltipsIndex; i < end.tooltipsIndex; i++ {
		w.nextFrame.tooltipRequests = append(w.nextFrame.tooltipRequests, w.renderedFrame.tooltipRequests[i])
		w.renderedFrame.tooltipRequests[i] = nil
	}
	w.nextFrame.accessedElementStates = append(w.nextFrame.accessedElementStates,
		w.renderedFrame.accessedElementStates[start.accessedElementStatesIndex:end.accessedElementStatesIndex]...)
	w.textSystem.reuseLayouts(start.lineLayoutIndex, end.lineLayoutIndex)

	reusedSubtree := w.nextFrame.dispatchTree.reuseSubtree(
		Range{Start: start.dispatchTreeIndex, End: end.dispatchTreeIndex},
		w.renderedFrame.dispatchTree,
		w.focus,
	)

	if reusedSubtree.containsFocus() {
		w.nextFrame.focus = w.focus
	}

	for _, draw := range w.renderedFrame.deferredDraws[start.deferredDrawsIndex:end.deferredDrawsIndex] {
		w.nextFrame.deferredDraws = append(w.nextFrame.deferredDraws, deferredDraw{
			currentView:      draw.currentView,
			parentNode:       reusedSubtree.refreshNodeID(draw.parentNode),
			elementIDStack:   draw.elementIDStack.clone(),
			textStyleStack:   slices.Clone(draw.textStyleStack),
			contentMask:      draw.contentMask,
			remSize:          draw.remSize,
			priority:         draw.priority,
			element:          None[AnyElement](),
			absoluteOffset:   draw.absoluteOffset,
			prepaintRange:    draw.prepaintRange,
			paintRange:       draw.paintRange,
			accessedEntities: maps.Clone(draw.accessedEntities),
		})
	}
}

func (w *Window) paintIndex() paintIndex {
	return paintIndex{
		sceneIndex:                 w.nextFrame.scene.Len(),
		debugBoundsIndex:           len(w.nextFrame.debugBoundsRecords),
		mouseListenersIndex:        len(w.nextFrame.mouseListeners),
		inputHandlersIndex:         len(w.nextFrame.inputHandlers),
		cursorStylesIndex:          len(w.nextFrame.cursorStyles),
		accessedElementStatesIndex: len(w.nextFrame.accessedElementStates),
		tabHandleIndex:             w.nextFrame.tabStops.paintIndex(),
		lineLayoutIndex:            w.textSystem.layoutIndex(),
	}
}

func (w *Window) reusePaint(start, end paintIndex) {
	for _, record := range w.renderedFrame.debugBoundsRecords[start.debugBoundsIndex:end.debugBoundsIndex] {
		w.nextFrame.recordDebugBounds(record.selector, record.bounds)
	}
	w.nextFrame.cursorStyles = append(w.nextFrame.cursorStyles, w.renderedFrame.cursorStyles[start.cursorStylesIndex:end.cursorStylesIndex]...)
	for i := start.inputHandlersIndex; i < end.inputHandlersIndex; i++ {
		w.nextFrame.inputHandlers = append(w.nextFrame.inputHandlers, w.renderedFrame.inputHandlers[i])
		w.renderedFrame.inputHandlers[i] = nil
	}
	for i := start.mouseListenersIndex; i < end.mouseListenersIndex; i++ {
		w.nextFrame.mouseListeners = append(w.nextFrame.mouseListeners, w.renderedFrame.mouseListeners[i])
		w.renderedFrame.mouseListeners[i] = nil
	}
	w.nextFrame.accessedElementStates = append(w.nextFrame.accessedElementStates,
		w.renderedFrame.accessedElementStates[start.accessedElementStatesIndex:end.accessedElementStatesIndex]...)
	w.nextFrame.tabStops.replay(w.renderedFrame.tabStops.insertionHistory[start.tabHandleIndex:end.tabHandleIndex])

	w.textSystem.reuseLayouts(start.lineLayoutIndex, end.lineLayoutIndex)
	w.nextFrame.scene.Replay(Range{Start: start.sceneIndex, End: end.sceneIndex}, &w.renderedFrame.scene)
}

// WithTextStyle pushes a text style onto the stack and calls f with it active; call it only
// during element drawing.
func (w *Window) WithTextStyle[R any](style Option[TextStyleRefinement], f func(window *Window) R) R {
	w.invalidator.debugAssertPaintOrPrepaint()
	if style, ok := style.Get(); ok {
		w.textStyleStack = append(w.textStyleStack, style)
		result := f(w)
		w.textStyleStack = w.textStyleStack[:len(w.textStyleStack)-1]
		return result
	}
	return f(w)
}

// SetCursorStyle requests a cursor style while hitbox is hovered; call it only during paint.
func (w *Window) SetCursorStyle(style CursorStyle, hitbox Hitbox) {
	w.invalidator.debugAssertPaint()
	w.nextFrame.cursorStyles = append(w.nextFrame.cursorStyles, cursorStyleRequest{
		hitboxID: Some(hitbox.ID),
		style:    style,
	})
}

// SetWindowCursorStyle requests a cursor style for the whole window, taking precedence over
// SetCursorStyle; call it only during paint.
func (w *Window) SetWindowCursorStyle(style CursorStyle) {
	w.invalidator.debugAssertPaint()
	w.nextFrame.cursorStyles = append(w.nextFrame.cursorStyles, cursorStyleRequest{
		hitboxID: None[HitboxID](),
		style:    style,
	})
}

func (w *Window) drop() {
	if w.closeA11yChannels != nil {
		w.closeA11yChannels()
		w.closeA11yChannels = nil
	}
	if root, ok := w.root.Take().Get(); ok {
		root.Release()
	}
	if prompt, ok := w.prompt.Take().Get(); ok {
		prompt.release()
	}
	w.renderedFrame.clear()
	w.nextFrame.clear()
	if inputHandler := w.platformWindow.TakeInputHandler(); inputHandler != nil {
		inputHandler.release()
	}
	w.platformWindow.Close()
}

// SetTooltip sets a tooltip to be rendered for the upcoming frame; call it only during prepaint.
func (w *Window) SetTooltip(tooltip AnyTooltip) TooltipID {
	w.invalidator.debugAssertPrepaint()
	id := w.nextTooltipID
	w.nextTooltipID++
	w.nextFrame.tooltipRequests = append(w.nextFrame.tooltipRequests, &tooltipRequest{id: id, tooltip: tooltip})
	return id
}

// WithContentMask calls f with mask intersected with the current content mask; call it only
// during element drawing.
func (w *Window) WithContentMask[R any](mask Option[ContentMask[Pixels]], f func(window *Window) R) R {
	w.invalidator.debugAssertPaintOrPrepaint()
	if mask, ok := mask.Get(); ok {
		mask = mask.Intersect(w.ContentMask())
		w.contentMaskStack = append(w.contentMaskStack, mask)
		result := f(w)
		w.contentMaskStack = w.contentMaskStack[:len(w.contentMaskStack)-1]
		return result
	}
	return f(w)
}

// WithElementOffset calls f with the element offset moved by offset, used to implement
// scrolling; call it only during prepaint.
func (w *Window) WithElementOffset[R any](offset Point[Pixels], f func(window *Window) R) R {
	w.invalidator.debugAssertPrepaint()

	if offset.IsZero() {
		return f(w)
	}

	absOffset := w.ElementOffset().Add(offset)
	return w.WithAbsoluteElementOffset(absOffset, f)
}

// WithAbsoluteElementOffset calls f with the element offset set to offset; call it only during
// prepaint.
func (w *Window) WithAbsoluteElementOffset[R any](offset Point[Pixels], f func(window *Window) R) R {
	w.invalidator.debugAssertPrepaint()
	w.elementOffsetStack = append(w.elementOffsetStack, offset)
	result := f(w)
	w.elementOffsetStack = w.elementOffsetStack[:len(w.elementOffsetStack)-1]
	return result
}

func (w *Window) withElementOpacity[R any](opacity Option[float32], f func(window *Window) R) R {
	w.invalidator.debugAssertPaintOrPrepaint()

	value, ok := opacity.Get()
	if !ok {
		return f(w)
	}

	previousOpacity := w.currentElementOpacity
	w.currentElementOpacity = previousOpacity * value
	result := f(w)
	w.currentElementOpacity = previousOpacity
	return result
}

// Transact prepaints in a retryable manner: when f fails, the side effects of its prepaints are
// discarded. Call it only during prepaint.
func (w *Window) Transact[T any](f func(window *Window) (T, error)) (T, error) {
	w.invalidator.debugAssertPrepaint()
	index := w.prepaintIndex()
	result, err := f(w)
	if err != nil {
		w.nextFrame.hitboxes = w.nextFrame.hitboxes[:index.hitboxesIndex]
		releaseTooltipRequests(w.nextFrame.tooltipRequests[index.tooltipsIndex:])
		w.nextFrame.tooltipRequests = w.nextFrame.tooltipRequests[:index.tooltipsIndex]
		clear(w.nextFrame.deferredDraws[index.deferredDrawsIndex:])
		w.nextFrame.deferredDraws = w.nextFrame.deferredDraws[:index.deferredDrawsIndex]
		w.nextFrame.dispatchTree.truncate(index.dispatchTreeIndex)
		clear(w.nextFrame.accessedElementStates[index.accessedElementStatesIndex:])
		w.nextFrame.accessedElementStates = w.nextFrame.accessedElementStates[:index.accessedElementStatesIndex]
		w.textSystem.truncateLayouts(index.lineLayoutIndex)
	}
	return result, err
}

// RequestAutoscroll asks containing elements to scroll bounds into view; call it only during
// prepaint.
func (w *Window) RequestAutoscroll(bounds Bounds[Pixels]) {
	w.invalidator.debugAssertPrepaint()
	w.requestedAutoscroll = Some(bounds)
}

// TakeAutoscroll takes the bounds requested with RequestAutoscroll.
func (w *Window) TakeAutoscroll() (Bounds[Pixels], bool) {
	w.invalidator.debugAssertPrepaint()
	return w.requestedAutoscroll.Take().Get()
}

// UseAsset loads an asset asynchronously and returns false until it has loaded; the current
// view is redrawn once it has.
func (w *Window) UseAsset[A Asset[S, O], S, O any](source S, cx *App) (O, bool) {
	return assetEntry[A](cx, source).useBy(w.CurrentView())
}

// GetAsset loads an asset asynchronously and returns false until it has loaded; the view is not
// redrawn when it has.
func (w *Window) GetAsset[A Asset[S, O], S, O any](source S, cx *App) (O, bool) {
	return cx.FetchAsset[A](source)
}

// ElementOffset returns the current element offset; call it only during prepaint.
func (w *Window) ElementOffset() Point[Pixels] {
	w.invalidator.debugAssertPrepaint()
	if n := len(w.elementOffsetStack); n != 0 {
		return w.elementOffsetStack[n-1]
	}
	return Point[Pixels]{}
}

func (w *Window) elementOpacity() float32 {
	w.invalidator.debugAssertPaintOrPrepaint()
	return w.currentElementOpacity
}

// ContentMask returns the current content mask; call it only during element drawing.
func (w *Window) ContentMask() ContentMask[Pixels] {
	w.invalidator.debugAssertPaintOrPrepaint()
	if n := len(w.contentMaskStack); n != 0 {
		return w.contentMaskStack[n-1]
	}
	return ContentMask[Pixels]{Bounds: Bounds[Pixels]{Origin: Point[Pixels]{}, Size: w.viewportSize}}
}

// WithElementNamespace calls f in a new namespace, in which element ids must be unique.
func (w *Window) WithElementNamespace[R any](elementID ElementID, f func(window *Window) R) R {
	w.elementIDStack.push(elementID)
	result := f(w)
	w.elementIDStack.pop()
	return result
}

// UseKeyedState returns state that lives as long as this element is rendered in consecutive
// frames; the caller releases the returned handle.
func (w *Window) UseKeyedState[S any](key ElementID, cx *App, init func(window *Window, cx *Context[S]) *S) Entity[S] {
	currentView := w.CurrentView()
	return w.WithGlobalID[Entity[S]](key, func(globalID *GlobalElementID, window *Window) Entity[S] {
		return window.WithElementState(globalID, func(state Option[Entity[S]], window *Window) (Entity[S], Entity[S]) {
			if state, ok := state.Get(); ok {
				return state.Clone(), state
			}
			newState := cx.New(func(cx *Context[S]) *S { return init(window, cx) })
			observeKeyedState(newState, currentView, cx)
			return newState.Clone(), newState
		})
	})
}

// UseState is UseKeyedState keyed by the caller's source location.
func (w *Window) UseState[S any](cx *App, init func(window *Window, cx *Context[S]) *S) Entity[S] {
	return w.UseKeyedState(ElementIDCodeLocation(CallerSourceLocation(1)), cx, init)
}

// WithElementState gives f the element state of type S stored under id in the rendered frame,
// and stores the state f returns for the next frame. Call it only during element drawing.
func (w *Window) WithElementState[S, R any](globalID *GlobalElementID, f func(state Option[S], window *Window) (R, S)) R {
	w.invalidator.debugAssertPaintOrPrepaint()

	key, box, ok := w.takeElementState(globalID, reflect.TypeFor[S]())

	if ok {
		stateBox, isS := box.inner.(*Option[S])
		if !isS {
			requested := typeNameOf(reflect.TypeFor[S]())
			if gpuiutil.DebugAssertions {
				panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: invalid element state type for id, requested %q, actual: %q", requested, box.typeName))
			}
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: invalid element state type for id, requested %q", requested))
		}

		state := stateBox.Take().Expect("reentrant call to with_element_state for the same state type and element id")
		result, state := f(Some(state), w)
		stateBox.Replace(state)
		w.insertElementState(key, newElementStateBox(stateBox, box.typeName))
		return result
	}
	result, state := f(None[S](), w)
	stateBox := Some(state)
	w.insertElementState(key, newElementStateBox(&stateBox, typeNameOf(reflect.TypeFor[S]())))
	return result
}

// WithOptionalElementState is WithElementState for an optional id: a nil globalID gives f None
// and f must return None.
func (w *Window) WithOptionalElementState[S, R any](globalID *GlobalElementID, f func(state Option[Option[S]], window *Window) (R, Option[S])) R {
	w.invalidator.debugAssertPaintOrPrepaint()

	if globalID != nil {
		return w.WithElementState(globalID, func(state Option[S], window *Window) (R, S) {
			result, state2 := f(Some(state), window)
			return result, state2.Expect("you must return some state when you pass some element id")
		})
	}
	result, state := f(None[Option[S]](), w)
	if gpuiutil.DebugAssertions && state.IsSome() {
		panic("you must not return an element state when passing None for the global id")
	}
	return result
}

// WithTabGroup calls f within a tab group at index.
func (w *Window) WithTabGroup[R any](index Option[int], f func(window *Window) R) R {
	if index, ok := index.Get(); ok {
		w.nextFrame.tabStops.beginGroup(index)
		result := f(w)
		w.nextFrame.tabStops.endGroup()
		return result
	}
	return f(w)
}

// DeferDraw defers drawing element on top of the current tree; higher priorities draw on top. A
// contentMask clips it during prepaint and paint. Call it only during prepaint.
func (w *Window) DeferDraw(element AnyElement, absoluteOffset Point[Pixels], priority int, contentMask Option[ContentMask[Pixels]]) {
	w.invalidator.debugAssertPrepaint()
	parentNode, ok := w.nextFrame.dispatchTree.activeNodeID()
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	w.nextFrame.deferredDraws = append(w.nextFrame.deferredDraws, deferredDraw{
		currentView:      w.CurrentView(),
		parentNode:       parentNode,
		elementIDStack:   w.elementIDStack.clone(),
		textStyleStack:   slices.Clone(w.textStyleStack),
		contentMask:      contentMask,
		remSize:          w.RemSize(),
		priority:         priority,
		element:          Some(element),
		absoluteOffset:   absoluteOffset,
		prepaintRange:    prepaintStateRange{},
		paintRange:       paintRange{},
		accessedEntities: collections.HashSet[EntityID]{},
	})
}

// PaintLayer creates a new painting layer drawn above this one; call it only during paint.
func (w *Window) PaintLayer[R any](f func(window *Window) R) R {
	w.invalidator.debugAssertPaint()

	w.nextFrame.scene.PushLayer()
	result := f(w)
	w.nextFrame.scene.PopLayer()
	return result
}

// PaintDropShadows paints the drop (non-inset) shadows of shadows; call it only during paint.
func (w *Window) PaintDropShadows(bounds Bounds[Pixels], cornerRadii Corners[Pixels], shadows []BoxShadow) {
	w.invalidator.debugAssertPaint()

	scaleFactor := w.ScaleFactor()
	contentMask := w.snappedContentMask()
	opacity := w.elementOpacity()
	for _, shadow := range shadows {
		if shadow.IsInset {
			continue
		}
		shapeBounds := w.coverBounds(bounds.Add(shadow.Offset).Dilate(shadow.Spread))
		if shapeBounds.IsEmpty() {
			continue
		}
		blurRadius := shadow.Blur.Scale(scaleFactor)
		w.nextFrame.scene.InsertPrimitive(Shadow{
			Order:            0,
			BlurRadius:       blurRadius,
			Bounds:           shapeBounds.Dilate(blurRadius * 3.),
			ContentMask:      contentMask,
			CornerRadii:      Corners[ScaledPixels]{},
			Color:            shadow.Color.Opacity(opacity),
			ShapeBounds:      shapeBounds,
			ShapeCornerRadii: cornerRadii.Scale(scaleFactor),
			Inset:            PaddedBool32FromBool(false),
			Pad:              0,
		})
	}
}

// PaintInsetShadows paints the inset shadows of shadows, after the element's background; call it
// only during paint.
func (w *Window) PaintInsetShadows(bounds Bounds[Pixels], cornerRadii Corners[Pixels], shadows []BoxShadow) {
	w.invalidator.debugAssertPaint()

	scaleFactor := w.ScaleFactor()
	contentMask := w.snappedContentMask()
	opacity := w.elementOpacity()
	elementBounds := w.coverBounds(bounds)
	elementCornerRadii := cornerRadii.Scale(scaleFactor)
	for _, shadow := range shadows {
		if !shadow.IsInset {
			continue
		}
		hole := bounds.Add(shadow.Offset).Dilate(-shadow.Spread)
		var zero Pixels
		holeCornerRadii := Corners[Pixels]{
			TopLeft:     max(cornerRadii.TopLeft-shadow.Spread, zero),
			TopRight:    max(cornerRadii.TopRight-shadow.Spread, zero),
			BottomRight: max(cornerRadii.BottomRight-shadow.Spread, zero),
			BottomLeft:  max(cornerRadii.BottomLeft-shadow.Spread, zero),
		}
		w.nextFrame.scene.InsertPrimitive(Shadow{
			Order:            0,
			BlurRadius:       shadow.Blur.Scale(scaleFactor),
			Bounds:           elementBounds,
			ContentMask:      contentMask,
			CornerRadii:      elementCornerRadii,
			Color:            shadow.Color.Opacity(opacity),
			ShapeBounds:      w.coverBounds(hole),
			ShapeCornerRadii: holeCornerRadii.Scale(scaleFactor),
			Inset:            PaddedBool32FromBool(true),
			Pad:              0,
		})
	}
}

func windowLargestBorderInterior(quad *Quad) Bounds[ScaledPixels] {
	size := quad.Bounds.Size
	width, height := float32(size.Width), float32(size.Height)
	diagonal := float32(math.Sqrt(float64(float32(width*width) + float32(height*height))))
	if diagonal == 0. {
		return Bounds[ScaledPixels]{}
	}
	cos := height / diagonal
	sin := width / diagonal

	cornerInset := func(radius, widthX, widthY ScaledPixels) Point[ScaledPixels] {
		return NewPoint(
			max(widthX, radius-ScaledPixels(float32((radius-widthX)*ScaledPixels(cos)))),
			max(widthY, radius-ScaledPixels(float32((radius-widthY)*ScaledPixels(sin)))),
		)
	}

	radii := &quad.CornerRadii
	widths := &quad.BorderWidths
	topLeft := cornerInset(radii.TopLeft, widths.Left, widths.Top)
	topRight := cornerInset(radii.TopRight, widths.Right, widths.Top)
	bottomRight := cornerInset(radii.BottomRight, widths.Right, widths.Bottom)
	bottomLeft := cornerInset(radii.BottomLeft, widths.Left, widths.Bottom)

	antialiasInset := NewPoint(ScaledPixels(1.0), ScaledPixels(1.0))
	return BoundsFromCorners(
		quad.Bounds.Origin.
			Add(NewPoint(max(topLeft.X, bottomLeft.X), max(topLeft.Y, topRight.Y))).
			Add(antialiasInset),
		quad.Bounds.BottomRight().
			Sub(NewPoint(max(topRight.X, bottomRight.X), max(bottomLeft.Y, bottomRight.Y))).
			Sub(antialiasInset),
	)
}

// PaintQuad paints a quad into the scene for the next frame; call it only during paint. The
// corner radii may exceed the bounds; use Corners.ClampRadiiForQuadSize to fit them.
func (w *Window) PaintQuad(quad PaintQuad) {
	w.invalidator.debugAssertPaint()

	opacity := w.elementOpacity()
	snappedBounds := w.snapBounds(quad.Bounds)
	snappedBorderWidths := w.snapBorderWidths(quad.BorderWidths)
	w.insertQuad(Quad{
		Order:        0,
		Bounds:       snappedBounds,
		ContentMask:  w.snappedContentMask(),
		Background:   quad.Background.Opacity(opacity),
		BorderColor:  quad.BorderColor.Opacity(opacity),
		CornerRadii:  quad.CornerRadii.Scale(w.ScaleFactor()),
		BorderWidths: snappedBorderWidths,
		BorderStyle:  quad.BorderStyle,
	})
}

func (w *Window) insertQuad(quad Quad) {
	hasBackground := !quad.Background.IsTransparent()
	hasBorder := quad.BorderWidths != (Edges[ScaledPixels]{}) && !quad.BorderColor.IsTransparent()
	if !hasBackground && !hasBorder {
		return
	}

	isRectangle := quad.BorderWidths == (Edges[ScaledPixels]{}) && quad.CornerRadii == (Corners[ScaledPixels]{})
	if hasBackground && isRectangle {
		w.nextFrame.scene.InsertPrimitive(quad)
		return
	}

	plainBorder := quad.CornerRadii == (Corners[ScaledPixels]{}) &&
		quad.BorderStyle == BorderStyleSolid &&
		(!hasBackground || quad.BorderColor.IsOpaque())

	outerBounds := quad.Bounds
	var innerBounds Bounds[ScaledPixels]
	if plainBorder {
		innerBounds = quad.Bounds.Extend(quad.BorderWidths.Map(func(width ScaledPixels) ScaledPixels { return -width }))
	} else {
		innerBounds = windowLargestBorderInterior(&quad)
	}

	stripQuad := quad
	if plainBorder {
		stripQuad.Background = SolidBackground(quad.BorderColor)
		stripQuad.BorderWidths = Edges[ScaledPixels]{}
	}

	if innerBounds.IsEmpty() {
		w.nextFrame.scene.InsertPrimitive(stripQuad)
		return
	}

	if hasBackground {
		interior := quad
		interior.ContentMask = ContentMask[ScaledPixels]{Bounds: quad.ContentMask.Bounds.Intersect(innerBounds)}
		interior.BorderWidths = Edges[ScaledPixels]{}
		interior.CornerRadii = Corners[ScaledPixels]{}
		w.nextFrame.scene.InsertPrimitive(interior)
	}

	strips := [4]Bounds[ScaledPixels]{
		BoundsFromCorners(outerBounds.Origin, NewPoint(outerBounds.Right(), innerBounds.Top())),
		BoundsFromCorners(NewPoint(outerBounds.Left(), innerBounds.Bottom()), outerBounds.BottomRight()),
		BoundsFromCorners(NewPoint(outerBounds.Left(), innerBounds.Top()), innerBounds.BottomLeft()),
		BoundsFromCorners(innerBounds.TopRight(), NewPoint(outerBounds.Right(), innerBounds.Bottom())),
	}

	for _, strip := range strips {
		stripped := stripQuad
		stripped.ContentMask = ContentMask[ScaledPixels]{Bounds: quad.ContentMask.Bounds.Intersect(strip)}
		w.nextFrame.scene.InsertPrimitive(stripped)
	}
}

// PaintPath paints path into the scene for the next frame; call it only during paint.
func (w *Window) PaintPath(path Path, color Background) {
	w.invalidator.debugAssertPaint()

	scaleFactor := w.ScaleFactor()
	contentMask := w.ContentMask()
	opacity := w.elementOpacity()
	w.nextFrame.scene.InsertPrimitive(NewPaintedPath(path, scaleFactor, contentMask, color.Opacity(opacity)))
}

// PaintUnderline paints an underline into the scene for the next frame; call it only during
// paint.
func (w *Window) PaintUnderline(origin Point[Pixels], width Pixels, style UnderlineStyle) {
	w.invalidator.debugAssertPaint()
	underline := w.underline(origin, width, style)
	w.nextFrame.scene.InsertPrimitive(underline)
}

// PaintStrikethrough paints a strikethrough into the scene for the next frame; call it only
// during paint.
func (w *Window) PaintStrikethrough(origin Point[Pixels], width Pixels, style StrikethroughStyle) {
	w.invalidator.debugAssertPaint()

	scaleFactor := w.ScaleFactor()
	height := style.Thickness
	bounds := Bounds[ScaledPixels]{
		Origin: origin.Map(w.devicePixel(scaleFactor)),
		Size:   Size[ScaledPixels]{Width: w.snapStroke(width), Height: w.snapStroke(height)},
	}
	opacity := w.elementOpacity()

	w.nextFrame.scene.InsertPrimitive(Underline{
		Order:       0,
		Pad:         0,
		Bounds:      bounds,
		ContentMask: w.snappedContentMask(),
		Thickness:   w.snapStroke(style.Thickness),
		Color:       style.Color.UnwrapOrDefault().Opacity(opacity),
		Wavy:        PaddedBool32FromBool(false),
	})
}

func fract32(v float32) float32 {
	return v - float32(math.Trunc(float64(v)))
}

func devicePixelsToScaled(d DevicePixels) ScaledPixels {
	return ScaledPixels(float32(d))
}

// PaintGlyph paints a monochrome (non-emoji) glyph whose baseline is at origin; call it only
// during paint. Prefer ShapedLine.Paint or WrappedLine.Paint.
func (w *Window) PaintGlyph(origin Point[Pixels], fontID FontID, glyphID GlyphID, fontSize Pixels, color Hsla) error {
	w.invalidator.debugAssertPaint()

	elementOpacity := w.elementOpacity()
	scaleFactor := w.ScaleFactor()
	glyphOrigin := origin.Scale(scaleFactor)

	quantizedOrigin := NewPoint(
		roundHalfTowardZero(float32(glyphOrigin.X)*float32(SubpixelVariantsX))/float32(SubpixelVariantsX),
		roundHalfTowardZero(float32(glyphOrigin.Y)*float32(SubpixelVariantsY))/float32(SubpixelVariantsY),
	)
	subpixelVariant := PointOf[uint8]{
		X: saturatingF32ToU8(fract32(quantizedOrigin.X) * float32(SubpixelVariantsX)),
		Y: saturatingF32ToU8(fract32(quantizedOrigin.Y) * float32(SubpixelVariantsY)),
	}
	integerOrigin := quantizedOrigin.Map(func(c float32) ScaledPixels { return ScaledPixels(float32(math.Trunc(float64(c)))) })
	subpixelRendering := w.shouldUseSubpixelRendering(fontID, fontSize)
	dilation := w.TextSystem().glyphDilationForColor(color)
	params := RenderGlyphParams{
		FontID:            fontID,
		GlyphID:           glyphID,
		FontSize:          fontSize,
		SubpixelVariant:   subpixelVariant,
		ScaleFactor:       scaleFactor,
		IsEmoji:           false,
		SubpixelRendering: subpixelRendering,
		Dilation:          dilation,
	}

	rasterBounds, err := w.TextSystem().getRasterBounds(params)
	if err != nil {
		return err
	}
	if !rasterBounds.IsZero() {
		tile, ok, err := w.spriteAtlas.GetOrInsertWith(AtlasKeyFromGlyph(params), func() (Size[DevicePixels], []byte, bool, error) {
			size, bytes, err := w.TextSystem().rasterizeGlyph(params)
			if err != nil {
				return Size[DevicePixels]{}, nil, false, err
			}
			return size, bytes, true, nil
		})
		if err != nil {
			return err
		}
		if !ok {
			panic("Callback above only errors or returns Some")
		}
		bounds := Bounds[ScaledPixels]{
			Origin: integerOrigin.Add(rasterBounds.Origin.Map(devicePixelsToScaled)),
			Size:   tile.Bounds.Size.Map(devicePixelsToScaled),
		}
		contentMask := w.snappedContentMask()

		if subpixelRendering {
			w.nextFrame.scene.InsertPrimitive(SubpixelSprite{
				Order:          0,
				Pad:            0,
				Bounds:         bounds,
				ContentMask:    contentMask,
				Color:          color.Opacity(elementOpacity),
				Tile:           tile,
				Transformation: TransformationMatrixUnit(),
			})
		} else {
			w.nextFrame.scene.InsertPrimitive(MonochromeSprite{
				Order:          0,
				Animation:      SpriteAnimationIDNone,
				Bounds:         bounds,
				ContentMask:    contentMask,
				Color:          color.Opacity(elementOpacity),
				Tile:           tile,
				Transformation: TransformationMatrixUnit(),
			})
		}
	}
	return nil
}

func (w *Window) shouldUseSubpixelRendering(fontID FontID, fontSize Pixels) bool {
	if !w.platformWindow.IsSubpixelRenderingSupported() {
		return false
	}

	if w.platformWindow.BackgroundAppearance() != WindowBackgroundAppearanceOpaque {
		return false
	}

	mode := *w.textRenderingMode
	if mode == TextRenderingModePlatformDefault {
		mode = w.TextSystem().recommendedRenderingMode(fontID, fontSize)
	}

	return mode == TextRenderingModeSubpixel
}

// PaintEmoji paints an emoji glyph whose baseline is at origin; call it only during paint.
// Prefer ShapedLine.Paint or WrappedLine.Paint.
func (w *Window) PaintEmoji(origin Point[Pixels], fontID FontID, glyphID GlyphID, fontSize Pixels) error {
	w.invalidator.debugAssertPaint()

	scaleFactor := w.ScaleFactor()
	glyphOrigin := origin.Scale(scaleFactor)
	integerOrigin := glyphOrigin.Map(func(c ScaledPixels) ScaledPixels { return ScaledPixels(roundHalfTowardZero(float32(c))) })
	params := RenderGlyphParams{
		FontID:            fontID,
		GlyphID:           glyphID,
		FontSize:          fontSize,
		SubpixelVariant:   PointOf[uint8]{},
		ScaleFactor:       scaleFactor,
		IsEmoji:           true,
		SubpixelRendering: false,
		Dilation:          0,
	}

	rasterBounds, err := w.TextSystem().getRasterBounds(params)
	if err != nil {
		return err
	}
	if !rasterBounds.IsZero() {
		tile, ok, err := w.spriteAtlas.GetOrInsertWith(AtlasKeyFromGlyph(params), func() (Size[DevicePixels], []byte, bool, error) {
			size, bytes, err := w.TextSystem().rasterizeGlyph(params)
			if err != nil {
				return Size[DevicePixels]{}, nil, false, err
			}
			return size, bytes, true, nil
		})
		if err != nil {
			return err
		}
		if !ok {
			panic("Callback above only errors or returns Some")
		}

		bounds := Bounds[ScaledPixels]{
			Origin: integerOrigin.Add(rasterBounds.Origin.Map(devicePixelsToScaled)),
			Size:   tile.Bounds.Size.Map(devicePixelsToScaled),
		}
		contentMask := w.snappedContentMask()
		opacity := w.elementOpacity()

		w.nextFrame.scene.InsertPrimitive(PolychromeSprite{
			Order:       0,
			Pad:         0,
			Grayscale:   PaddedBool32FromBool(false),
			Bounds:      bounds,
			CornerRadii: Corners[ScaledPixels]{},
			ContentMask: contentMask,
			Tile:        tile,
			Opacity:     opacity,
		})
	}
	return nil
}

// PaintSvg paints a monochrome SVG into the scene for the next frame; data, when not nil, is the
// SVG source. Call it only during paint.
func (w *Window) PaintSvg(bounds Bounds[Pixels], path SharedString, data []byte, transformation TransformationMatrix, color Hsla, cx *App) error {
	return w.paintAnimatedSvg(bounds, path, data, transformation, None[spriteAnimation](), color, cx)
}

func (w *Window) paintAnimatedSvg(bounds Bounds[Pixels], path SharedString, data []byte, transformation TransformationMatrix, animation Option[spriteAnimation], color Hsla, cx *App) error {
	w.invalidator.debugAssertPaint()

	elementOpacity := w.elementOpacity()
	snapped := w.snapBounds(bounds)

	params := RenderSvgParams{
		Path: path,
		Size: snapped.Size.Map(func(pixels ScaledPixels) DevicePixels {
			return DevicePixels(saturatingF32ToI32(float32(math.Ceil(float64(float32(pixels) * SmoothSvgScaleFactor)))))
		}),
	}

	tile, ok, err := w.spriteAtlas.GetOrInsertWith(AtlasKeyFromSvg(params), func() (Size[DevicePixels], []byte, bool, error) {
		return cx.svgRenderer.renderAlphaMask(params, data)
	})
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	contentMask := w.snappedContentMask()
	svgBounds := Bounds[ScaledPixels]{
		Origin: snapped.Center().Sub(NewPoint(
			ScaledPixels(float32(tile.Bounds.Size.Width)/SmoothSvgScaleFactor/2.),
			ScaledPixels(float32(tile.Bounds.Size.Height)/SmoothSvgScaleFactor/2.),
		)),
		Size: tile.Bounds.Size.Map(func(value DevicePixels) ScaledPixels {
			return ScaledPixels(float32(value) / SmoothSvgScaleFactor)
		}),
	}
	finalBounds := svgBounds.
		MapOrigin(func(value ScaledPixels) ScaledPixels { return ScaledPixels(roundHalfTowardZero(float32(value))) }).
		MapSize(ScaledPixels.Ceil)

	sprite := MonochromeSprite{
		Order:          0,
		Animation:      SpriteAnimationIDNone,
		Bounds:         finalBounds,
		ContentMask:    contentMask,
		Color:          color.Opacity(elementOpacity),
		Tile:           tile,
		Transformation: transformation,
	}
	if animation, ok := animation.Get(); ok {
		w.nextFrame.scene.insertAnimatedSprite(sprite, animation)
	} else {
		w.nextFrame.scene.InsertPrimitive(sprite)
	}

	return nil
}

func roundF32ToI32(v float32) int32 {
	return saturatingF32ToI32(float32(math.Round(float64(v))))
}

// PaintImage paints frame frameIndex of data into bounds, positioned and scaled by imageBounds;
// the visible region is bounds intersected with imageBounds, with cornerRadii applied to bounds.
// Call it only during paint.
func (w *Window) PaintImage(bounds Bounds[Pixels], imageBounds Bounds[Pixels], cornerRadii Corners[Pixels], data *RenderImage, frameIndex int, grayscale bool) error {
	w.invalidator.debugAssertPaint()

	visibleBounds := bounds.Intersect(imageBounds)
	if visibleBounds.Size.Width <= 0 || visibleBounds.Size.Height <= 0 {
		return nil
	}
	if imageBounds.Size.Width <= 0 || imageBounds.Size.Height <= 0 {
		return nil
	}

	params := RenderImageParams{
		ImageID:    data.ID,
		FrameIndex: frameIndex,
	}

	tile, ok, err := w.spriteAtlas.GetOrInsertWith(AtlasKeyFromImage(params), func() (Size[DevicePixels], []byte, bool, error) {
		bytes, ok := data.AsBytes(frameIndex)
		if !ok {
			panic("It's the caller's job to pass a valid frame index")
		}
		return data.Size(frameIndex), bytes, true, nil
	})
	if err != nil {
		return err
	}
	if !ok {
		panic("Callback above only returns Some")
	}

	visibleBoundsSnapped := w.snapBounds(visibleBounds)

	subTile := tile
	if visibleBounds != imageBounds {
		xOffsetRatio := float32(visibleBounds.Origin.X-imageBounds.Origin.X) / float32(imageBounds.Size.Width)
		yOffsetRatio := float32(visibleBounds.Origin.Y-imageBounds.Origin.Y) / float32(imageBounds.Size.Height)
		widthRatio := float32(visibleBounds.Size.Width) / float32(imageBounds.Size.Width)
		heightRatio := float32(visibleBounds.Size.Height) / float32(imageBounds.Size.Height)

		tileOriginX := int32(tile.Bounds.Origin.X)
		tileOriginY := int32(tile.Bounds.Origin.Y)
		tileWidth := int32(tile.Bounds.Size.Width)
		tileHeight := int32(tile.Bounds.Size.Height)

		subOriginX := tileOriginX + roundF32ToI32(float32(xOffsetRatio*float32(tileWidth)))
		subOriginY := tileOriginY + roundF32ToI32(float32(yOffsetRatio*float32(tileHeight)))
		subWidth := roundF32ToI32(float32(widthRatio * float32(tileWidth)))
		subHeight := roundF32ToI32(float32(heightRatio * float32(tileHeight)))

		maxX := tileOriginX + tileWidth
		maxY := tileOriginY + tileHeight

		clampedOriginX := min(max(subOriginX, tileOriginX), maxX)
		clampedOriginY := min(max(subOriginY, tileOriginY), maxY)
		clampedWidth := max(min(subWidth, maxX-clampedOriginX), 0)
		clampedHeight := max(min(subHeight, maxY-clampedOriginY), 0)

		subTile.Bounds = Bounds[DevicePixels]{
			Origin: NewPoint(DevicePixels(clampedOriginX), DevicePixels(clampedOriginY)),
			Size:   Size[DevicePixels]{Width: DevicePixels(clampedWidth), Height: DevicePixels(clampedHeight)},
		}
	}

	contentMask := w.snappedContentMask()
	scaledCornerRadii := cornerRadii.ClampRadiiForQuadSize(visibleBounds.Size).Scale(w.ScaleFactor())
	opacity := w.elementOpacity()

	w.nextFrame.scene.InsertPrimitive(PolychromeSprite{
		Order:       0,
		Pad:         0,
		Grayscale:   PaddedBool32FromBool(grayscale),
		Bounds:      visibleBoundsSnapped,
		ContentMask: contentMask,
		CornerRadii: scaledCornerRadii,
		Tile:        subTile,
		Opacity:     opacity,
	})
	return nil
}

// PaintSurface paints a platform video surface (a CVPixelBufferRef on macOS and iOS) into the
// scene for the next frame; call it only during paint.
func (w *Window) PaintSurface(bounds Bounds[Pixels], imageBuffer uintptr) {
	w.invalidator.debugAssertPaint()

	snapped := w.snapBounds(bounds)
	contentMask := w.snappedContentMask()
	w.nextFrame.scene.InsertPrimitive(PaintSurface{
		Order:       0,
		Bounds:      snapped,
		ContentMask: contentMask,
		ImageBuffer: imageBuffer,
	})
}

// DropImage removes every frame of data from the sprite atlas.
func (w *Window) DropImage(data *RenderImage) error {
	for frameIndex := range data.FrameCount() {
		params := RenderImageParams{
			ImageID:    data.ID,
			FrameIndex: frameIndex,
		}

		w.spriteAtlas.Remove(AtlasKeyFromImage(params))
	}

	return nil
}

// HasImageAtlasEntry reports whether every frame of data is present in the sprite atlas.
func (w *Window) HasImageAtlasEntry(data *RenderImage) bool {
	if data.FrameCount() == 0 {
		return false
	}
	for frameIndex := range data.FrameCount() {
		if !w.spriteAtlas.Contains(AtlasKeyFromImage(RenderImageParams{ImageID: data.ID, FrameIndex: frameIndex})) {
			return false
		}
	}
	return true
}
