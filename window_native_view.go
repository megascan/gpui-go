package gpui

import (
	"errors"
	"reflect"
	"slices"
	"weak"
)

// NativeViewID identifies a NativeView within its window.
type NativeViewID uint64

// NativeViewIDForTest makes an ID for a test that needs one without a window to attach a view to.
func NativeViewIDForTest(id uint64) NativeViewID {
	return NativeViewID(id)
}

// NativeView is a view owned by the platform's UI toolkit, shown inside a GPUI window. Only
// macOS shows native views so far, where the view is an NSView; NewNativeView fails on every
// other platform.
//
// The view is composited beneath the window's GPUI content and shows through wherever this
// handle is used as an element. It stays attached to the window it was created in until the
// last clone of the handle is released.
type NativeView struct {
	handle *nativeViewHandle
}

type nativeViewHandle struct {
	state    *nativeViewState
	released bool
}

type nativeViewState struct {
	id              NativeViewID
	window          AnyWindowHandle
	focusHandle     FocusHandle
	background      Hsla
	shownHitbox     Option[HitboxID]
	reservedActions map[reflect.Type]struct{}
	attachment      NativeViewAttachment
	refs            int
	dropped         bool
}

// NewNativeView attaches a platform view to the window. It fails on platforms that cannot
// composite native views.
//
// view must point to a live view of the kind the window's platform embeds, an NSView on macOS,
// that is not in another view hierarchy. GPUI retains the view until the last clone of the
// returned handle is released, and decides its frame, visibility and order until then.
func NewNativeView(view uintptr, window *Window, cx *App) (NativeView, error) {
	id := NativeViewID(window.nextNativeViewID)
	attachment := window.platformWindow.AttachNativeView(id, view)
	if attachment == nil {
		return NativeView{}, errors.New("this platform cannot show native views")
	}
	window.nextNativeViewID++
	state := &nativeViewState{
		id:              id,
		window:          window.handle,
		focusHandle:     cx.FocusHandle(),
		background:      TransparentBlack(),
		reservedActions: make(map[reflect.Type]struct{}),
		attachment:      attachment,
		refs:            1,
	}
	window.nativeViews = append(window.nativeViews, weak.Make(state))
	return NativeView{handle: &nativeViewHandle{state: state}}, nil
}

func (v NativeView) state() *nativeViewState {
	if v.handle == nil || v.handle.released {
		panic("gpui: use of a released NativeView")
	}
	return v.handle.state
}

// Clone returns a new handle to the same view.
func (v NativeView) Clone() NativeView {
	state := v.state()
	state.refs++
	return NativeView{handle: &nativeViewHandle{state: state}}
}

// Release drops this handle. The view is detached from its window when the last handle is
// released. Release is idempotent.
func (v NativeView) Release() {
	if v.handle == nil || v.handle.released {
		return
	}
	v.handle.released = true
	state := v.handle.state
	state.refs--
	if state.refs == 0 {
		state.dropped = true
		state.attachment.Close()
		state.focusHandle.Release()
	}
}

// SetBackground sets the color shown where the view has not caught up with its element.
func (v NativeView) SetBackground(intoColor IntoHsla, cx *App) {
	color := intoColor.ToHsla()
	state := v.state()
	previous := state.background
	state.background = color
	if previous == color {
		return
	}
	window := state.window
	cx.Defer(func(cx *App) {
		_ = window.Update(cx, func(_ AnyView, window *Window, _ *App) { window.Refresh() })
	})
}

// ReserveAction makes key bindings for A apply before the view sees the key.
func (v NativeView) ReserveAction[A Action]() {
	v.state().reservedActions[reflect.TypeFor[A]()] = struct{}{}
}

// FocusHandle returns the view's focus handle.
func (v NativeView) FocusHandle(_ *App) FocusHandle {
	return v.state().focusHandle.Clone()
}

// IntoAnyElement shows the view as a div filling its parent.
func (v NativeView) IntoAnyElement() AnyElement {
	return v.IntoElement().IntoAnyElement()
}

// IntoElement returns the div that shows the view.
func (v NativeView) IntoElement() *Div {
	state := v.state()
	return NewDiv().
		TrackFocus(state.focusHandle).
		SizeFull().
		Child(&nativeViewElement{view: v})
}

type nativeViewElement struct {
	view NativeView
}

func (e *nativeViewElement) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

func (e *nativeViewElement) SourceLocation() *SourceLocation {
	return nil
}

func (e *nativeViewElement) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	style := DefaultStyle()
	style.Size.Width = Relative(1.).ToLength()
	style.Size.Height = Relative(1.).ToLength()
	return window.RequestLayout(style, nil, cx), struct{}{}
}

func (e *nativeViewElement) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, window *Window, _ *App) Option[Hitbox] {
	state := e.view.handle.state
	if state.window != window.handle {
		return None[Hitbox]()
	}
	isVisible := !bounds.Intersect(window.ContentMask().Bounds).IsEmpty()
	isShownElsewhere := false
	if shown, ok := state.shownHitbox.Get(); ok {
		isShownElsewhere = slices.ContainsFunc(window.nextFrame.hitboxes, func(hitbox Hitbox) bool { return hitbox.ID == shown })
	}
	showsView := isVisible && !isShownElsewhere
	behavior := HitboxBehaviorNormal
	if showsView {
		behavior = HitboxBehaviorBlockMouse
	}
	hitbox := window.InsertHitbox(bounds, behavior)
	if showsView {
		state.shownHitbox = Some(hitbox.ID)
	}
	return Some(hitbox)
}

func (e *nativeViewElement) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, prepaint *Option[Hitbox], window *Window, _ *App) {
	hitbox, ok := prepaint.Get()
	if !ok {
		return
	}
	state := e.view.handle.state
	bounds := window.snapBounds(hitbox.Bounds)
	contentMask := window.snappedContentMask()
	window.nextFrame.scene.InsertPrimitive(PaintNativeView{
		Order:       0,
		ID:          state.id,
		HitboxID:    hitbox.ID,
		Bounds:      bounds,
		ContentMask: contentMask,
		Background:  state.background,
	})
}

func (e *nativeViewElement) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, Option[Hitbox]](e)
}

func liveNativeView(view weak.Pointer[nativeViewState]) *nativeViewState {
	state := view.Value()
	if state == nil || state.dropped {
		return nil
	}
	return state
}

func (w *Window) syncNativeViewFocus() {
	w.nativeViews = slices.DeleteFunc(w.nativeViews, func(view weak.Pointer[nativeViewState]) bool { return liveNativeView(view) == nil })
	if len(w.nativeViews) == 0 {
		return
	}
	focused := None[NativeViewID]()
	if focus, ok := w.focus.Get(); ok {
		if id, ok := w.nativeViewForFocus(focus); ok {
			focused = Some(id)
		}
	}
	w.platformWindow.SetFocusedNativeView(focused)
}

func (w *Window) endCompositionBeforeFocusingNativeView(focus FocusID, cx *App) {
	if _, ok := w.nativeViewForFocus(focus); !ok {
		return
	}
	w.Defer(cx, func(window *Window, cx *App) {
		if inputHandler := window.platformWindow.TakeInputHandler(); inputHandler != nil {
			inputHandler.unmarkTextDuringUpdate(window, cx)
			window.platformWindow.SetInputHandler(inputHandler)
		}
	})
}

func (w *Window) nativeViewForFocus(focus FocusID) (NativeViewID, bool) {
	for _, view := range w.nativeViews {
		if state := liveNativeView(view); state != nil && state.focusHandle.id == focus {
			return state.id, true
		}
	}
	return 0, false
}

func (w *Window) hitTestNativeView(position Point[Pixels]) (NativeViewID, bool) {
	hitTest := w.renderedFrame.hitTest(position)
	if len(hitTest.ids) == 0 {
		return 0, false
	}
	topmost := hitTest.ids[0]
	for _, nativeView := range w.renderedFrame.scene.NativeViews {
		if nativeView.HitboxID == topmost {
			return nativeView.ID, true
		}
	}
	return 0, false
}

func (w *Window) dispatchKeyDownBeforeNativeView(event KeyDownEvent, cx *App) bool {
	var view *nativeViewState
	focus, focused := w.focus.Get()
	if focused {
		for _, weakView := range w.nativeViews {
			if state := liveNativeView(weakView); state != nil && state.focusHandle.id == focus {
				view = state
				break
			}
		}
	}
	if view == nil {
		return false
	}
	nodeID := w.focusNodeIDInRenderedFrame(w.focus)
	dispatchTree := w.renderedFrame.dispatchTree
	dispatchPath := dispatchTree.dispatchPath(nodeID)
	pending, isPending := w.PendingInputKeystrokes()
	continuesPending := false
	if isPending {
		input := append(slices.Clone(pending), event.Keystroke)
		bindings, stillPending, _ := dispatchTree.bindingsForInput(input, dispatchPath)
		continuesPending = stillPending || len(bindings) != 0
	}
	isReserved := false
	if !continuesPending {
		input := []Keystroke{event.Keystroke}
		bindings, _, contextStack := dispatchTree.bindingsForInput(input, dispatchPath)
		reserved := func(binding KeyBinding) bool {
			_, ok := view.reservedActions[reflect.TypeOf(binding.action)]
			return ok
		}
		isReserved = (len(bindings) > 0 && reserved(bindings[0])) ||
			slices.ContainsFunc(dispatchTree.possibleNextBindingsForInput(input, contextStack), reserved)
	}
	if isPending && !continuesPending {
		w.flushPendingInput(cx)
	}
	if !continuesPending && !isReserved {
		return false
	}
	w.DispatchEvent(event, cx)
	return true
}

func (w *Window) focusNativeView(id NativeViewID, cx *App) {
	for _, weakView := range w.nativeViews {
		if state := liveNativeView(weakView); state != nil && state.id == id {
			w.Focus(state.focusHandle, cx)
			return
		}
	}
}
