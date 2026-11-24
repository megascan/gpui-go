//go:build darwin && !ios

package macos

import (
	"cmp"
	"slices"
	"sync"
	"sync/atomic"
	"weak"

	"github.com/ebitengine/purego"
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

type nativeViews struct {
	views                 []attachedView
	focused               gpui.Option[gpui.NativeViewID]
	clicked               gpui.Option[gpui.NativeViewID]
	declined              gpui.Option[gpui.NativeViewID]
	reconciling           bool
	buttonsPressedInView  []int
	keysTakenBeforeView   []uint16
	pointerOverView       bool
	drewSincePointerCheck bool
	routingKeyEquivalent  bool
	deliveringKeyDown     bool
	passedUpKeyEquivalent *objc.Retained
	stacking              []gpui.NativeViewID
	background            gpui.Option[gpui.Rgba]
	hitTestCallback       func(position gpui.Point[gpui.Pixels]) (gpui.NativeViewID, bool)
	focusedCallback       func(id gpui.NativeViewID)
	keyDownBeforeCallback func(event gpui.KeyDownEvent) bool
}

type attachedView struct {
	id        gpui.NativeViewID
	view      *objc.Retained
	container *objc.Retained
	placement gpui.Option[placement]
}

type placement struct {
	container [4]float64
	view      [4]float64
}

func newPlacement(bounds gpui.Bounds[gpui.ScaledPixels], contentMask gpui.Bounds[gpui.ScaledPixels], scaleFactor float64, gpuiFrame [4]float64) gpui.Option[placement] {
	visible := bounds.Intersect(contentMask)
	if visible.IsEmpty() {
		return gpui.None[placement]()
	}
	toFrame := func(bounds gpui.Bounds[gpui.ScaledPixels]) [4]float64 {
		gpuiX, gpuiY, gpuiHeight := gpuiFrame[0], gpuiFrame[1], gpuiFrame[3]
		toPoints := func(value gpui.ScaledPixels) float64 { return float64(value) / scaleFactor }
		height := toPoints(bounds.Size.Height)
		return [4]float64{
			gpuiX + toPoints(bounds.Origin.X),
			gpuiY + gpuiHeight - toPoints(bounds.Origin.Y) - height,
			toPoints(bounds.Size.Width),
			height,
		}
	}
	container := toFrame(visible)
	frame := toFrame(bounds)
	return gpui.Some(placement{
		container: container,
		view:      [4]float64{frame[0] - container[0], frame[1] - container[1], frame[2], frame[3]},
	})
}

func (n *nativeViews) shown(id gpui.NativeViewID) (*attachedView, bool) {
	for i := range n.views {
		if n.views[i].id == id && n.views[i].placement.IsSome() {
			return &n.views[i], true
		}
	}
	return nil, false
}

func (n *nativeViews) focusedView() gpui.Option[focusedNativeView] {
	id, ok := n.focused.Get()
	if !ok {
		return gpui.None[focusedNativeView]()
	}
	_, shown := n.shown(id)
	return gpui.Some(focusedNativeView{id: id, shown: shown, declined: n.declined == gpui.Some(id)})
}

func (n *nativeViews) isShowingAny() bool {
	return slices.ContainsFunc(n.views, func(attached attachedView) bool { return attached.placement.IsSome() })
}

func (n *nativeViews) noteButtonPressedInView(button int) {
	if !slices.Contains(n.buttonsPressedInView, button) {
		n.buttonsPressedInView = append(n.buttonsPressedInView, button)
	}
}

func (n *nativeViews) takeButtonPressedInView(button int) bool {
	count := len(n.buttonsPressedInView)
	n.buttonsPressedInView = slices.DeleteFunc(n.buttonsPressedInView, func(pressed int) bool { return pressed == button })
	return len(n.buttonsPressedInView) != count
}

func (n *nativeViews) holder(responder objc.ID, nativeWindow objc.ID, gpuiView objc.ID) inputHolder {
	if responder == objc.Nil || responder == nativeWindow {
		return inputHolder{kind: inputHolderWindow}
	} else if responder == gpuiView {
		return inputHolder{kind: inputHolderGpuiView}
	} else if id, ok := n.containing(responder); ok {
		return inputHolder{kind: inputHolderNativeView, id: id}
	}
	return inputHolder{kind: inputHolderOther}
}

func (n *nativeViews) containing(responder objc.ID) (gpui.NativeViewID, bool) {
	if responder == objc.Nil {
		return 0, false
	}
	if !objc.Send[bool](responder, objc.Sel("isKindOfClass:"), objc.GetClass("NSView")) {
		return 0, false
	}
	for _, attached := range n.views {
		if objc.Send[bool](responder, objc.Sel("isDescendantOf:"), attached.view.ID()) {
			return attached.id, true
		}
	}
	return 0, false
}

type nativeViewAttachment struct {
	windowState weak.Pointer[lockedWindowState]
	id          gpui.NativeViewID
	closed      atomic.Bool
}

func (a *nativeViewAttachment) Close() {
	if a.closed.Swap(true) {
		return
	}
	windowState := a.windowState.Value()
	if windowState == nil {
		return
	}
	windowState.Lock()
	index := slices.IndexFunc(windowState.nativeViews.views, func(attached attachedView) bool { return attached.id == a.id })
	if index < 0 {
		windowState.Unlock()
		return
	}
	if windowState.nativeViews.focused == gpui.Some(a.id) {
		windowState.nativeViews.focused = gpui.None[gpui.NativeViewID]()
	}
	attached := windowState.nativeViews.views[index]
	windowState.nativeViews.views = slices.Delete(windowState.nativeViews.views, index, index+1)
	nativeWindow, gpuiView := windowState.nativeWindow, windowState.nativeView
	windowState.Unlock()
	defer attached.view.Release()
	defer attached.container.Release()
	giveUpFirstResponder(nativeWindow, gpuiView, attached.view.ID())
	appkit.NSView(attached.container.ID()).RemoveFromSuperview()
	appkit.NSView(attached.view.ID()).RemoveFromSuperview()
}

const nsWindowBelow objc.NSInteger = -1

func nativeViewsAttach(windowState *lockedWindowState, id gpui.NativeViewID, view uintptr) gpui.NativeViewAttachment {
	if view == 0 {
		return nil
	}
	retainedView := objc.RetainObject(objc.ID(view))
	windowState.Lock()
	gpuiView := windowState.nativeView
	windowState.Unlock()
	container := objc.Own(objc.SendClass[objc.ID](containerClass(), objc.Sel("new")))
	objc.SendVoid(container.ID(), objc.Sel("setClipsToBounds:"), true)
	appkit.NSView(container.ID()).SetHidden(true)
	appkit.NSView(container.ID()).AddSubview(appkit.NSView(retainedView.ID()))
	contentView := objc.Send[objc.ID](gpuiView, objc.Sel("superview"))
	objc.SendVoid(contentView, objc.Sel("addSubview:positioned:relativeTo:"), container.ID(), nsWindowBelow, gpuiView)
	windowState.Lock()
	windowState.nativeViews.views = append(windowState.nativeViews.views, attachedView{
		id:        id,
		view:      retainedView,
		container: container,
	})
	windowState.Unlock()
	return &nativeViewAttachment{windowState: weak.Make(windowState), id: id}
}

var containerClass = sync.OnceValue(func() objc.Class {
	const name = "GPUINativeViewContainer"
	class, err := objc.RegisterClass(name, objc.GetClass("NSView"), nil, nil, []objc.MethodDef{
		{Cmd: objc.Sel("performKeyEquivalent:"), Fn: containerPerformKeyEquivalent},
	})
	if err != nil {
		if existing := objc.GetClass(name); existing != 0 {
			return existing
		}
		return objc.GetClass("NSView")
	}
	return class
})

func containerPerformKeyEquivalent(_ objc.ID, _ objc.SEL, _ objc.ID) bool {
	return false
}

func nativeViewsUpdateBackground(state *macWindowState, scene *gpui.Scene) {
	if state.backgroundAppearance != gpui.WindowBackgroundAppearanceOpaque {
		return
	}
	background := gpui.None[gpui.Rgba]()
	for _, nativeView := range slices.Backward(scene.NativeViews) {
		if rgba := nativeView.Background.ToRgb(); rgba.A > 0 {
			background = gpui.Some(rgba)
			break
		}
	}
	if state.nativeViews.background == background {
		return
	}
	state.nativeViews.background = background
	objc.SendVoid(state.nativeWindow, objc.Sel("setBackgroundColor:"), backgroundColor(background))
}

func backgroundColor(color gpui.Option[gpui.Rgba]) objc.ID {
	c := color.UnwrapOrDefault()
	return objc.SendClass[objc.ID](objc.GetClass("NSColor"), objc.Sel("colorWithSRGBRed:green:blue:alpha:"), float64(c.R), float64(c.G), float64(c.B), 1.0)
}

type nativeViewPlacement struct {
	id        gpui.NativeViewID
	placement gpui.Option[placement]
}

type nativeViewLayout struct {
	placements []nativeViewPlacement
	stacking   gpui.Option[[]gpui.NativeViewID]
}

func nativeViewsPlanLayout(state *macWindowState, scene *gpui.Scene) nativeViewLayout {
	if len(state.nativeViews.views) == 0 {
		return nativeViewLayout{}
	}
	scaleFactor := float64(state.scaleFactor())
	frame := objc.Send[objc.CGRect](state.nativeView, objc.Sel("frame"))
	gpuiFrame := [4]float64{frame.Origin.X, frame.Origin.Y, frame.Size.Width, frame.Size.Height}

	var placements []nativeViewPlacement
	for _, attached := range state.nativeViews.views {
		p := gpui.None[placement]()
		if index := slices.IndexFunc(scene.NativeViews, func(nativeView gpui.PaintNativeView) bool { return nativeView.ID == attached.id }); index >= 0 {
			nativeView := scene.NativeViews[index]
			p = newPlacement(nativeView.Bounds, nativeView.ContentMask.Bounds, scaleFactor, gpuiFrame)
		}
		if attached.placement != p {
			placements = append(placements, nativeViewPlacement{id: attached.id, placement: p})
		}
	}

	var stacking []gpui.NativeViewID
	for _, nativeView := range scene.NativeViews {
		if !slices.Contains(stacking, nativeView.ID) {
			stacking = append(stacking, nativeView.ID)
		}
	}
	layout := nativeViewLayout{placements: placements}
	if !slices.Equal(stacking, state.nativeViews.stacking) {
		layout.stacking = gpui.Some(stacking)
	}
	return layout
}

type appliedPlacement struct {
	view      *objc.Retained
	container *objc.Retained
	placement gpui.Option[placement]
}

func nativeViewsApplyLayout(windowState *lockedWindowState, layout nativeViewLayout) {
	if len(layout.placements) == 0 && layout.stacking.IsNone() {
		return
	}
	windowState.Lock()
	var placements []appliedPlacement
	for _, planned := range layout.placements {
		index := slices.IndexFunc(windowState.nativeViews.views, func(attached attachedView) bool { return attached.id == planned.id })
		if index < 0 {
			continue
		}
		attached := &windowState.nativeViews.views[index]
		attached.placement = planned.placement
		placements = append(placements, appliedPlacement{
			view:      objc.RetainObject(attached.view.ID()),
			container: objc.RetainObject(attached.container.ID()),
			placement: planned.placement,
		})
	}
	var order *stacking
	if planned, ok := layout.stacking.Get(); ok {
		var frontToBack []objc.ID
		for _, id := range slices.Backward(planned) {
			if index := slices.IndexFunc(windowState.nativeViews.views, func(view attachedView) bool { return view.id == id }); index >= 0 {
				frontToBack = append(frontToBack, windowState.nativeViews.views[index].container.ID())
			}
		}
		attached := make([]objc.ID, 0, len(windowState.nativeViews.views))
		for _, view := range windowState.nativeViews.views {
			attached = append(attached, view.container.ID())
		}
		windowState.nativeViews.stacking = planned
		order = &stacking{
			gpuiView:    windowState.nativeView,
			frontToBack: frontToBack,
			attached:    attached,
		}
	}
	nativeWindow, gpuiView := windowState.nativeWindow, windowState.nativeView
	windowState.Unlock()

	toRect := func(frame [4]float64) objc.CGRect {
		return objc.CGRect{Origin: objc.CGPoint{X: frame[0], Y: frame[1]}, Size: objc.CGSize{Width: frame[2], Height: frame[3]}}
	}
	for _, applied := range placements {
		if p, ok := applied.placement.Get(); ok {
			appkit.NSView(applied.container.ID()).SetFrame(toRect(p.container))
			appkit.NSView(applied.view.ID()).SetFrame(toRect(p.view))
			appkit.NSView(applied.container.ID()).SetHidden(false)
		} else {
			giveUpFirstResponder(nativeWindow, gpuiView, applied.view.ID())
			appkit.NSView(applied.container.ID()).SetHidden(true)
		}
		applied.view.Release()
		applied.container.Release()
	}
	if order != nil && len(order.frontToBack) > 1 {
		contentView := objc.Send[objc.ID](gpuiView, objc.Sel("superview"))
		if contentView != objc.Nil {
			order.current = objc.NSArrayToIDs(objc.Send[objc.ID](contentView, objc.Sel("subviews")))
			context := registerStacking(order)
			objc.SendVoid(contentView, objc.Sel("sortSubviewsUsingFunction:context:"), compareStackingCallback(), context)
			unregisterStacking(context)
		}
	}
	reconcileFirstResponder(windowState)
}

type stacking struct {
	gpuiView    objc.ID
	frontToBack []objc.ID
	attached    []objc.ID
	current     []objc.ID
}

type stackingRank struct {
	tier  uint8
	index int
}

func (s *stacking) rank(view objc.ID) stackingRank {
	place := func(view objc.ID) (int, bool) {
		index := slices.Index(s.current, view)
		return index, index >= 0
	}
	if view == s.gpuiView {
		return stackingRank{3, 0}
	} else if index := slices.Index(s.frontToBack, view); index >= 0 {
		return stackingRank{2, len(s.frontToBack) - index}
	} else if slices.Contains(s.attached, view) {
		return stackingRank{1, 0}
	}
	viewPlace, viewOK := place(view)
	gpuiPlace, gpuiOK := place(s.gpuiView)
	if viewOK && (!gpuiOK || viewPlace > gpuiPlace) {
		return stackingRank{4, viewPlace}
	}
	if !viewOK {
		viewPlace = 0
	}
	return stackingRank{0, viewPlace}
}

type nsComparisonResult = objc.NSInteger

const (
	nsOrderedAscending  nsComparisonResult = -1
	nsOrderedSame       nsComparisonResult = 0
	nsOrderedDescending nsComparisonResult = 1
)

func compareStacking(first objc.ID, second objc.ID, s *stacking) nsComparisonResult {
	a, b := s.rank(first), s.rank(second)
	switch c := cmp.Or(cmp.Compare(a.tier, b.tier), cmp.Compare(a.index, b.index)); {
	case c < 0:
		return nsOrderedAscending
	case c == 0:
		return nsOrderedSame
	default:
		return nsOrderedDescending
	}
}

var (
	stackingsMu   sync.Mutex
	stackings     = map[uintptr]*stacking{}
	nextStackings uintptr
)

func registerStacking(s *stacking) uintptr {
	stackingsMu.Lock()
	defer stackingsMu.Unlock()
	nextStackings++
	stackings[nextStackings] = s
	return nextStackings
}

func unregisterStacking(context uintptr) {
	stackingsMu.Lock()
	defer stackingsMu.Unlock()
	delete(stackings, context)
}

var compareStackingCallback = sync.OnceValue(func() uintptr {
	return purego.NewCallback(func(first objc.ID, second objc.ID, context uintptr) nsComparisonResult {
		stackingsMu.Lock()
		s := stackings[context]
		stackingsMu.Unlock()
		return compareStacking(first, second, s)
	})
})

type inputHolderKind uint8

const (
	inputHolderWindow inputHolderKind = iota
	inputHolderGpuiView
	inputHolderNativeView
	inputHolderOther
)

type inputHolder struct {
	kind inputHolderKind
	id   gpui.NativeViewID
}

type focusedNativeView struct {
	id       gpui.NativeViewID
	shown    bool
	declined bool
}

type inputTargetKind uint8

const (
	inputTargetGpuiView inputTargetKind = iota
	inputTargetNativeView
)

type inputTarget struct {
	kind inputTargetKind
	id   gpui.NativeViewID
}

func inputTargetForFocus(focused gpui.Option[focusedNativeView], current inputHolder) gpui.Option[inputTarget] {
	target := inputTarget{kind: inputTargetGpuiView}
	holder := inputHolder{kind: inputHolderGpuiView}
	if f, ok := focused.Get(); ok && f.shown && !f.declined {
		target = inputTarget{kind: inputTargetNativeView, id: f.id}
		holder = inputHolder{kind: inputHolderNativeView, id: f.id}
	}
	if current.kind != inputHolderOther && current != holder {
		return gpui.Some(target)
	}
	return gpui.None[inputTarget]()
}

func mayTakeInput(requested inputHolder, current inputHolder, focused gpui.Option[focusedNativeView], clicked gpui.Option[gpui.NativeViewID], voiceOverReachesIt bool) bool {
	switch requested.kind {
	case inputHolderGpuiView:
		f, ok := focused.Get()
		return !(ok && f.shown && current == inputHolder{kind: inputHolderNativeView, id: f.id})
	case inputHolderNativeView:
		f, ok := focused.Get()
		return (ok && f.id == requested.id) || clicked == gpui.Some(requested.id) || voiceOverReachesIt
	default:
		return true
	}
}

func nativeViewsSetFocused(windowState *lockedWindowState, id gpui.Option[gpui.NativeViewID]) {
	windowState.Lock()
	views := &windowState.nativeViews
	if views.focused != id {
		views.declined = gpui.None[gpui.NativeViewID]()
	}
	views.focused = id
	windowState.Unlock()
	reconcileFirstResponder(windowState)
}

func reconcileFirstResponder(windowState *lockedWindowState) {
	windowState.Lock()
	wasReconciling := windowState.nativeViews.reconciling
	windowState.nativeViews.reconciling = true
	windowState.Unlock()
	if wasReconciling {
		return
	}
	for range 2 {
		if !requestFirstResponderForFocus(windowState) {
			break
		}
	}
	windowState.Lock()
	windowState.nativeViews.reconciling = false
	windowState.Unlock()
}

func requestFirstResponderForFocus(windowState *lockedWindowState) bool {
	windowState.Lock()
	nativeWindow := windowState.nativeWindow
	gpuiView := windowState.nativeView
	current := objc.Send[objc.ID](nativeWindow, objc.Sel("firstResponder"))
	views := &windowState.nativeViews
	holder := views.holder(current, nativeWindow, gpuiView)
	target, ok := inputTargetForFocus(views.focusedView(), holder).Get()
	if !ok {
		windowState.Unlock()
		return false
	}
	var retained *objc.Retained
	if target.kind == inputTargetNativeView {
		attached, shown := views.shown(target.id)
		if !shown {
			windowState.Unlock()
			return false
		}
		retained = objc.RetainObject(attached.view.ID())
	}
	windowState.Unlock()
	if retained != nil {
		defer retained.Release()
		objc.Send[bool](nativeWindow, objc.Sel("makeFirstResponder:"), retained.ID())
	} else {
		objc.Send[bool](nativeWindow, objc.Sel("makeFirstResponder:"), gpuiView)
	}
	return true
}

func giveUpFirstResponder(nativeWindow objc.ID, gpuiView objc.ID, view objc.ID) {
	current := objc.Send[objc.ID](nativeWindow, objc.Sel("firstResponder"))
	isView := current != objc.Nil && objc.Send[bool](current, objc.Sel("isKindOfClass:"), objc.GetClass("NSView"))
	inside := isView && objc.Send[bool](current, objc.Sel("isDescendantOf:"), view)
	if inside {
		objc.Send[bool](nativeWindow, objc.Sel("makeFirstResponder:"), gpuiView)
	}
}

func nativeViewsAddWindowMethods() []objc.MethodDef {
	return []objc.MethodDef{
		{Cmd: objc.Sel("makeFirstResponder:"), Fn: makeFirstResponder},
		{Cmd: objc.Sel("sendEvent:"), Fn: sendEvent},
		{Cmd: objc.Sel("performKeyEquivalent:"), Fn: windowPerformKeyEquivalent},
		{Cmd: objc.Sel("keyDown:"), Fn: windowKeyDown},
	}
}

func nativeViewsAddViewMethods() []objc.MethodDef {
	return []objc.MethodDef{
		{Cmd: objc.Sel("hitTest:"), Fn: hitTest},
		{Cmd: objc.Sel("becomeFirstResponder"), Fn: becomeFirstResponder},
		{Cmd: objc.Sel("resignFirstResponder"), Fn: resignFirstResponder},
	}
}

func hitTestNativeView(windowState *lockedWindowState, positionInWindow objc.CGPoint) (gpui.NativeViewID, bool) {
	if !windowState.TryLock() {
		return 0, false
	}
	if !windowState.nativeViews.isShowingAny() {
		windowState.Unlock()
		return 0, false
	}
	position := convertMousePosition(positionInWindow, windowState.contentSize().Height)
	callback := windowState.nativeViews.hitTestCallback
	if callback == nil {
		windowState.Unlock()
		return 0, false
	}
	windowState.nativeViews.hitTestCallback = nil
	windowState.Unlock()
	id, ok := callback(position)
	windowState.Lock()
	defer windowState.Unlock()
	windowState.nativeViews.hitTestCallback = callback
	if !ok {
		return 0, false
	}
	if _, shown := windowState.nativeViews.shown(id); !shown {
		return 0, false
	}
	return id, true
}

func hitTest(this objc.ID, _ objc.SEL, point objc.CGPoint) objc.ID {
	superview := objc.Send[objc.ID](this, objc.Sel("superview"))
	if windowState, ok := windowStateIfSet(this); ok && superview != objc.Nil {
		positionInWindow := objc.Send[objc.CGPoint](superview, objc.Sel("convertPoint:toView:"), point, objc.Nil)
		if _, hit := hitTestNativeView(windowState, positionInWindow); hit {
			return objc.Nil
		}
	}
	return objc.SendSuper[objc.ID](this, objc.Sel("hitTest:"), point)
}

func sendEvent(this objc.ID, _ objc.SEL, event objc.ID) {
	windowState, ok := windowStateIfSet(this)
	if !ok {
		objc.SendSuper[objc.ID](this, objc.Sel("sendEvent:"), event)
		return
	}
	nsEvent := appkit.NSEvent(event)
	if keyDown, ok := keyDownForNativeView(windowState, this, event); ok {
		keyCode := nsEvent.KeyCode()
		if offerKeyDownBeforeNativeView(windowState, keyDown) {
			noteKeyTakenBeforeView(windowState, keyCode, true)
			return
		}
		noteKeyTakenBeforeView(windowState, keyCode, false)
		windowState.Lock()
		windowState.nativeViews.deliveringKeyDown = true
		windowState.Unlock()
		objc.SendSuper[objc.ID](this, objc.Sel("sendEvent:"), event)
		windowState.Lock()
		windowState.nativeViews.deliveringKeyDown = false
		windowState.Unlock()
		return
	}

	eventType := nsEvent.Type()
	if eventType == appkit.NSEventTypeKeyUp {
		keyCode := nsEvent.KeyCode()
		wasTaken := false
		if windowState.TryLock() {
			taken := windowState.nativeViews.keysTakenBeforeView
			count := len(taken)
			windowState.nativeViews.keysTakenBeforeView = slices.DeleteFunc(taken, func(key uint16) bool { return key == keyCode })
			wasTaken = len(windowState.nativeViews.keysTakenBeforeView) != count
			windowState.Unlock()
		}
		if wasTaken {
			if keyUp := platformInputFromNative(event, gpui.None[gpui.Pixels]()); keyUp != nil {
				passInputToGpui(windowState, keyUp)
			}
			return
		}
	}
	if eventType == appkit.NSEventTypeFlagsChanged {
		objc.SendSuper[objc.ID](this, objc.Sel("sendEvent:"), event)
		if modifiersChanged := modifiersChangedBesideView(windowState, this, event); modifiersChanged != nil {
			passInputToGpui(windowState, modifiersChanged)
		}
		return
	}
	if eventType == appkit.NSEventTypeMouseMoved {
		updatePointerOverView(windowState, nsEvent.LocationInWindow())
	}
	clicked := gpui.None[gpui.NativeViewID]()
	switch eventType {
	case appkit.NSEventTypeLeftMouseDown, appkit.NSEventTypeRightMouseDown, appkit.NSEventTypeOtherMouseDown:
		position := nsEvent.LocationInWindow()
		if _, hit := hitTestNativeView(windowState, position); hit {
			clicked = gpui.OptionFrom(nativeViewUnderPress(windowState, this, position))
		}
	}
	if clicked.IsNone() {
		isRelease := eventType == appkit.NSEventTypeLeftMouseUp || eventType == appkit.NSEventTypeRightMouseUp || eventType == appkit.NSEventTypeOtherMouseUp
		var release gpui.PlatformInput
		if isRelease {
			button := nsEvent.ButtonNumber()
			if windowState.TryLock() {
				owed := windowState.nativeViews.takeButtonPressedInView(button)
				windowHeight := windowState.contentSize().Height
				windowState.Unlock()
				if owed {
					release = platformInputFromNative(event, gpui.Some(windowHeight))
				}
			}
		}
		objc.SendSuper[objc.ID](this, objc.Sel("sendEvent:"), event)
		if release != nil {
			passInputToGpui(windowState, release)
		}
		return
	}
	button := nsEvent.ButtonNumber()
	windowState.Lock()
	windowState.nativeViews.clicked = clicked
	windowState.nativeViews.noteButtonPressedInView(button)
	windowHeight := windowState.contentSize().Height
	windowState.Unlock()
	if mouseDown := platformInputFromNative(event, gpui.Some(windowHeight)); mouseDown != nil {
		passInputToGpui(windowState, mouseDown)
	}
	objc.SendSuper[objc.ID](this, objc.Sel("sendEvent:"), event)
	windowState.Lock()
	windowState.nativeViews.clicked = gpui.None[gpui.NativeViewID]()
	windowState.Unlock()
}

func nativeViewUnderPress(windowState *lockedWindowState, window objc.ID, positionInWindow objc.CGPoint) (gpui.NativeViewID, bool) {
	contentView := objc.Send[objc.ID](window, objc.Sel("contentView"))
	if contentView == objc.Nil {
		return 0, false
	}
	frameView := objc.Send[objc.ID](contentView, objc.Sel("superview"))
	rootView := frameView
	if frameView == objc.Nil {
		rootView = contentView
	}
	recipient := objc.Send[objc.ID](rootView, objc.Sel("hitTest:"), positionInWindow)
	if !windowState.TryLock() {
		return 0, false
	}
	defer windowState.Unlock()
	return windowState.nativeViews.containing(recipient)
}

func updatePointerOverView(windowState *lockedWindowState, positionInWindow objc.CGPoint) {
	_, overView := hitTestNativeView(windowState, positionInWindow)
	if !windowState.TryLock() {
		return
	}
	if windowState.nativeViews.pointerOverView == overView {
		windowState.Unlock()
		return
	}
	windowState.nativeViews.pointerOverView = overView
	nativeWindow, gpuiView := windowState.nativeWindow, windowState.nativeView
	windowState.Unlock()
	objc.SendVoid(nativeWindow, objc.Sel("invalidateCursorRectsForView:"), gpuiView)
}

func nativeViewsSceneDrawn(state *macWindowState, scene *gpui.Scene) {
	views := &state.nativeViews
	views.drewSincePointerCheck = views.pointerOverView || len(scene.NativeViews) != 0
}

func checkPointerAfterFrame(windowState *lockedWindowState) {
	if !windowState.TryLock() {
		return
	}
	drew := windowState.nativeViews.drewSincePointerCheck
	windowState.nativeViews.drewSincePointerCheck = false
	nativeWindow := windowState.nativeWindow
	windowState.Unlock()
	if !drew {
		return
	}
	position := objc.Send[objc.CGPoint](nativeWindow, objc.Sel("mouseLocationOutsideOfEventStream"))
	updatePointerOverView(windowState, position)
}

func nativeViewSetsCursor(state *macWindowState) bool {
	return state.nativeViews.pointerOverView && state.nativeViews.isShowingAny()
}

func makeFirstResponder(this objc.ID, _ objc.SEL, responder objc.ID) bool {
	windowState, ok := windowStateIfSet(this)
	if !ok {
		return objc.SendSuper[bool](this, objc.Sel("makeFirstResponder:"), responder)
	}
	if !windowState.TryLock() {
		return objc.SendSuper[bool](this, objc.Sel("makeFirstResponder:"), responder)
	}
	if len(windowState.nativeViews.views) == 0 {
		windowState.Unlock()
		return objc.SendSuper[bool](this, objc.Sel("makeFirstResponder:"), responder)
	}
	views := &windowState.nativeViews
	gpuiView := windowState.nativeView
	nativeWindow := windowState.nativeWindow
	current := views.holder(objc.Send[objc.ID](this, objc.Sel("firstResponder")), nativeWindow, gpuiView)
	requested := views.holder(responder, nativeWindow, gpuiView)
	focused := views.focusedView()
	clicked := views.clicked
	requestedIsShown := false
	if requested.kind == inputHolderNativeView {
		_, requestedIsShown = views.shown(requested.id)
	}
	windowState.Unlock()

	workspace := objc.SendClass[objc.ID](objc.GetClass("NSWorkspace"), objc.Sel("sharedWorkspace"))
	voiceOverEnabled := objc.Send[bool](workspace, objc.Sel("isVoiceOverEnabled"))
	voiceOverReachesIt := voiceOverEnabled && requestedIsShown
	if !mayTakeInput(requested, current, focused, clicked, voiceOverReachesIt) {
		return false
	}

	accepted := objc.SendSuper[bool](this, objc.Sel("makeFirstResponder:"), responder)
	if !accepted {
		return accepted
	}

	if requested.kind == inputHolderNativeView {
		windowState.Lock()
		firstResponder := objc.Send[objc.ID](this, objc.Sel("firstResponder"))
		views := &windowState.nativeViews
		containing, containsOK := views.containing(firstResponder)
		tookInput := containsOK && containing == requested.id
		if !tookInput {
			views.declined = gpui.Some(requested.id)
		} else if views.declined == gpui.Some(requested.id) {
			views.declined = gpui.None[gpui.NativeViewID]()
		}
		windowState.Unlock()
		f, isFocused := focused.Get()
		if tookInput && (!isFocused || f.id != requested.id) {
			windowState.Lock()
			callback := windowState.nativeViews.focusedCallback
			windowState.nativeViews.focusedCallback = nil
			windowState.Unlock()
			if callback != nil {
				callback(requested.id)
				windowState.Lock()
				windowState.nativeViews.focusedCallback = callback
				windowState.Unlock()
			}
		}
	}
	reconcileFirstResponder(windowState)
	return accepted
}

func keyDownForNativeView(windowState *lockedWindowState, window objc.ID, event objc.ID) (gpui.KeyDownEvent, bool) {
	if appkit.NSEvent(event).Type() != appkit.NSEventTypeKeyDown {
		return gpui.KeyDownEvent{}, false
	}
	if !windowState.TryLock() {
		return gpui.KeyDownEvent{}, false
	}
	if len(windowState.nativeViews.views) == 0 {
		windowState.Unlock()
		return gpui.KeyDownEvent{}, false
	}
	if id, ok := windowState.nativeViews.focused.Get(); ok {
		if _, shown := windowState.nativeViews.shown(id); shown {
			windowState.Unlock()
			reconcileFirstResponder(windowState)
			if !windowState.TryLock() {
				return gpui.KeyDownEvent{}, false
			}
		}
	}
	firstResponder := objc.Send[objc.ID](window, objc.Sel("firstResponder"))
	if _, ok := windowState.nativeViews.containing(firstResponder); !ok {
		windowState.Unlock()
		return gpui.KeyDownEvent{}, false
	}
	windowHeight := windowState.contentSize().Height
	windowState.Unlock()
	keyDown, ok := platformInputFromNative(event, gpui.Some(windowHeight)).(gpui.KeyDownEvent)
	return keyDown, ok
}

func noteKeyTakenBeforeView(windowState *lockedWindowState, keyCode uint16, taken bool) {
	windowState.Lock()
	defer windowState.Unlock()
	keys := slices.DeleteFunc(windowState.nativeViews.keysTakenBeforeView, func(key uint16) bool { return key == keyCode })
	if taken {
		keys = append(keys, keyCode)
	}
	windowState.nativeViews.keysTakenBeforeView = keys
}

func modifiersChangedBesideView(windowState *lockedWindowState, window objc.ID, event objc.ID) gpui.PlatformInput {
	if !windowState.TryLock() {
		return nil
	}
	defer windowState.Unlock()
	if len(windowState.nativeViews.views) == 0 {
		return nil
	}
	firstResponder := objc.Send[objc.ID](window, objc.Sel("firstResponder"))
	if _, ok := windowState.nativeViews.containing(firstResponder); !ok {
		return nil
	}
	input := platformInputFromNative(event, gpui.None[gpui.Pixels]())
	current, ok := input.(gpui.ModifiersChangedEvent)
	if !ok {
		return nil
	}
	if previous, ok := windowState.previousModifiersChangedEvent.(gpui.ModifiersChangedEvent); ok &&
		previous.Modifiers == current.Modifiers && previous.Capslock == current.Capslock {
		return nil
	}
	windowState.previousModifiersChangedEvent = input
	return input
}

func offerKeyDownBeforeNativeView(windowState *lockedWindowState, keyDown gpui.KeyDownEvent) bool {
	windowState.Lock()
	callback := windowState.nativeViews.keyDownBeforeCallback
	windowState.nativeViews.keyDownBeforeCallback = nil
	windowState.Unlock()
	if callback == nil {
		return false
	}
	handled := callback(keyDown)
	windowState.Lock()
	windowState.nativeViews.keyDownBeforeCallback = callback
	windowState.Unlock()
	return handled
}

func passKeyDownToGpui(windowState *lockedWindowState, keyDown gpui.KeyDownEvent) bool {
	return passInputToGpui(windowState, keyDown)
}

func passInputToGpui(windowState *lockedWindowState, input gpui.PlatformInput) bool {
	windowState.Lock()
	callback := windowState.eventCallback
	windowState.eventCallback = nil
	windowState.Unlock()
	if callback == nil {
		return false
	}
	handled := !callback(input).Propagate
	windowState.Lock()
	windowState.eventCallback = callback
	windowState.Unlock()
	return handled
}

func windowPerformKeyEquivalent(this objc.ID, _ objc.SEL, event objc.ID) bool {
	windowState, ok := windowStateIfSet(this)
	if !ok {
		return objc.SendSuper[bool](this, objc.Sel("performKeyEquivalent:"), event)
	}
	keyDown, ok := keyDownForNativeView(windowState, this, event)
	if !ok {
		return objc.SendSuper[bool](this, objc.Sel("performKeyEquivalent:"), event)
	}
	windowState.Lock()
	deliveringKeyDown := windowState.nativeViews.deliveringKeyDown
	windowState.Unlock()
	if !deliveringKeyDown && offerKeyDownBeforeNativeView(windowState, keyDown) {
		noteKeyTakenBeforeView(windowState, appkit.NSEvent(event).KeyCode(), true)
		return true
	}

	view := nativeViewWithKeyboardInput(windowState, this)
	windowState.Lock()
	windowState.nativeViews.routingKeyEquivalent = true
	windowState.Unlock()
	performed := false
	if view != nil {
		performed = objc.Send[bool](view.ID(), objc.Sel("performKeyEquivalent:"), event)
		view.Release()
	}
	if !performed {
		performed = objc.SendSuper[bool](this, objc.Sel("performKeyEquivalent:"), event)
	}
	windowState.Lock()
	windowState.nativeViews.routingKeyEquivalent = false
	windowState.Unlock()
	if performed || deliveringKeyDown {
		return performed
	}

	windowState.Lock()
	previous := windowState.nativeViews.passedUpKeyEquivalent
	windowState.nativeViews.passedUpKeyEquivalent = nil
	if event != objc.Nil {
		windowState.nativeViews.passedUpKeyEquivalent = objc.RetainObject(event)
	}
	windowState.Unlock()
	if previous != nil {
		previous.Release()
	}
	return passKeyDownToGpui(windowState, keyDown)
}

func nativeViewWithKeyboardInput(windowState *lockedWindowState, window objc.ID) *objc.Retained {
	if !windowState.TryLock() {
		return nil
	}
	defer windowState.Unlock()
	firstResponder := objc.Send[objc.ID](window, objc.Sel("firstResponder"))
	id, ok := windowState.nativeViews.containing(firstResponder)
	if !ok {
		return nil
	}
	index := slices.IndexFunc(windowState.nativeViews.views, func(view attachedView) bool { return view.id == id })
	if index < 0 {
		return nil
	}
	return objc.RetainObject(windowState.nativeViews.views[index].view.ID())
}

func windowKeyDown(this objc.ID, _ objc.SEL, event objc.ID) {
	windowState, ok := windowStateIfSet(this)
	if !ok {
		objc.SendSuper[objc.ID](this, objc.Sel("keyDown:"), event)
		return
	}
	keyDown, ok := keyDownForNativeView(windowState, this, event)
	if !ok {
		objc.SendSuper[objc.ID](this, objc.Sel("keyDown:"), event)
		return
	}
	windowState.Lock()
	passedUp := windowState.nativeViews.passedUpKeyEquivalent
	windowState.nativeViews.passedUpKeyEquivalent = nil
	windowState.Unlock()
	alreadyPassedUp := false
	if passedUp != nil {
		alreadyPassedUp = passedUp.ID() == event
		passedUp.Release()
	}
	if !alreadyPassedUp {
		passKeyDownToGpui(windowState, keyDown)
	}
}

func isRoutingKeyEquivalent(windowState *lockedWindowState) bool {
	if !windowState.TryLock() {
		return false
	}
	defer windowState.Unlock()
	return windowState.nativeViews.routingKeyEquivalent
}

func gpuiViewIsFocused(state *macWindowState) bool {
	if !state.window().IsKeyWindow() {
		return false
	}
	if len(state.nativeViews.views) == 0 {
		return true
	}
	firstResponder := objc.Send[objc.ID](state.nativeWindow, objc.Sel("firstResponder"))
	return firstResponder == state.nativeView
}

func updateAccessibilityFocus(windowState *lockedWindowState, isFirstResponder bool) {
	if !windowState.TryLock() {
		return
	}
	isFocused := isFirstResponder && windowState.window().IsKeyWindow()
	var events a11yQueuedEvents
	hasEvents := false
	if adapter := windowState.a11yAdapter; adapter != nil {
		events, hasEvents = adapter.UpdateViewFocusState(isFocused)
	}
	windowState.Unlock()
	if hasEvents {
		events.Raise()
	}
}

func becomeFirstResponder(this objc.ID, _ objc.SEL) bool {
	accepted := objc.SendSuper[bool](this, objc.Sel("becomeFirstResponder"))
	windowState, ok := windowStateIfSet(this)
	if !ok {
		return accepted
	}
	if !windowState.TryLock() {
		return accepted
	}
	if !accepted || len(windowState.nativeViews.views) == 0 {
		windowState.Unlock()
		return accepted
	}
	executor := windowState.foregroundExecutor
	windowState.Unlock()
	updateAccessibilityFocus(windowState, true)
	detachOnForeground(executor, func() {
		event := gpui.ModifiersChangedEvent{Modifiers: currentModifiers(), Capslock: currentCapslock()}
		windowState.Lock()
		if previous, ok := windowState.previousModifiersChangedEvent.(gpui.ModifiersChangedEvent); ok &&
			previous.Modifiers == event.Modifiers && previous.Capslock == event.Capslock {
			windowState.Unlock()
			return
		}
		windowState.previousModifiersChangedEvent = event
		callback := windowState.eventCallback
		if callback == nil {
			windowState.Unlock()
			return
		}
		windowState.eventCallback = nil
		windowState.Unlock()
		callback(event)
		windowState.Lock()
		windowState.eventCallback = callback
		windowState.Unlock()
	})
	return accepted
}

func resignFirstResponder(this objc.ID, _ objc.SEL) bool {
	accepted := objc.SendSuper[bool](this, objc.Sel("resignFirstResponder"))
	windowState, ok := windowStateIfSet(this)
	if !ok {
		return accepted
	}
	hasNativeViews := false
	if windowState.TryLock() {
		hasNativeViews = len(windowState.nativeViews.views) != 0
		windowState.Unlock()
	}
	if accepted && hasNativeViews {
		withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) struct{} {
			inputHandler.UnmarkText()
			return struct{}{}
		})
		inputContext := objc.Send[objc.ID](this, objc.Sel("inputContext"))
		objc.SendVoid(inputContext, objc.Sel("discardMarkedText"))
		updateAccessibilityFocus(windowState, false)
	}
	return accepted
}
