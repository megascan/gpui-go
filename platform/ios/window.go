//go:build ios

package ios

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"structs"
	"sync"
	"sync/atomic"
	"weak"

	pobjc "github.com/ebitengine/purego/objc"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/mtl"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/internal/apple/uikit"
	"github.com/megascan/gpui-go/platform/apple"
	"github.com/megascan/gpui-go/scheduler"
)

var keyboardDismissDistance = gpui.Px(24.)

type keyboardDismissTouch struct {
	id            gpui.TouchID
	startPosition gpui.Point[gpui.Pixels]
}

var statusBarStyle atomic.Int32

const (
	uiStatusBarStyleLightContent objc.NSInteger = 1
	uiStatusBarStyleDarkContent  objc.NSInteger = 3

	uiViewAutoresizingFlexibleWidth  objc.NSUInteger = 1 << 1
	uiViewAutoresizingFlexibleHeight objc.NSUInteger = 1 << 4

	uiScrollTypeMaskAll objc.NSInteger = 3

	uiUserInterfaceLayoutDirectionRightToLeft objc.NSInteger = 1

	uiRectCornerTopLeft     objc.NSUInteger = 1 << 0
	uiRectCornerTopRight    objc.NSUInteger = 1 << 1
	uiRectCornerBottomLeft  objc.NSUInteger = 1 << 2
	uiRectCornerBottomRight objc.NSUInteger = 1 << 3
)

type windowReference struct {
	window weak.Pointer[IosWindowState]
}

func (r *windowReference) set(window *IosWindowState) {
	if window == nil {
		r.window = weak.Pointer[IosWindowState]{}
		return
	}
	r.window = weak.Make(window)
}

func (r *windowReference) withWindow(callback func(window *IosWindowState)) bool {
	_, ok := windowReferenceWithR(r, func(window *IosWindowState) struct{} {
		callback(window)
		return struct{}{}
	})
	return ok
}

func windowReferenceWithR[R any](r *windowReference, callback func(window *IosWindowState) R) (R, bool) {
	windowReferenceQueries++
	window := r.window.Value()
	if window == nil {
		var zero R
		return zero, false
	}
	window.dispatchDepth++
	defer window.endDispatch()
	return callback(window), true
}

var windowReferenceQueries int

var (
	nativeRefsMu sync.Mutex
	nativeRefs   = map[objc.ID]*windowReference{}
)

func nativeWindowReference(this objc.ID) *windowReference {
	nativeRefsMu.Lock()
	defer nativeRefsMu.Unlock()
	reference, ok := nativeRefs[this]
	if !ok {
		reference = &windowReference{}
		nativeRefs[this] = reference
	}
	return reference
}

func dropNativeWindowReference(this objc.ID) {
	nativeRefsMu.Lock()
	delete(nativeRefs, this)
	nativeRefsMu.Unlock()
}

func nativeDealloc(this objc.ID, _ objc.SEL) {
	dropNativeWindowReference(this)
	objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
}

const (
	viewControllerClassName = "GPUIViewController"
	metalViewClassName      = "GPUIMetalView"
)

var (
	viewControllerClassOnce sync.Once
	viewControllerClassVal  objc.Class
	metalViewClassOnce      sync.Once
	metalViewClassVal       objc.Class
)

func viewControllerClass() objc.Class {
	viewControllerClassOnce.Do(func() {
		methods := []objc.MethodDef{
			{Cmd: objc.Sel("dealloc"), Fn: nativeDealloc},
			{Cmd: objc.Sel("preferredStatusBarStyle"), Fn: viewControllerPreferredStatusBarStyle},
			{Cmd: objc.Sel("viewDidLayoutSubviews"), Fn: viewControllerViewDidLayoutSubviews},
		}
		class, err := objc.RegisterClass(viewControllerClassName, objc.GetClass("UIViewController"), nil, nil, methods)
		if err != nil {
			panic(fmt.Sprintf("failed to register %s: %v", viewControllerClassName, err))
		}
		viewControllerClassVal = class
	})
	return viewControllerClassVal
}

func viewControllerPreferredStatusBarStyle(_ objc.ID, _ objc.SEL) objc.NSInteger {
	if statusBarStyle.Load() == 1 {
		return uiStatusBarStyleLightContent
	}
	return uiStatusBarStyleDarkContent
}

func viewControllerViewDidLayoutSubviews(this objc.ID, _ objc.SEL) {
	objc.SendSuper[objc.ID](this, objc.Sel("viewDidLayoutSubviews"))
	nativeWindowReference(this).withWindow((*IosWindowState).handleLayoutChange)
}

func newViewController() objc.ID {
	return objc.Send[objc.ID](objc.Alloc(viewControllerClass()), objc.Sel("init"))
}

// SetStatusBarStyle sets the iOS status bar content style (light or dark text and icons) and
// asks every window's root view controller to re-query preferredStatusBarStyle.
func SetStatusBarStyle(style StatusBarContentStyle) {
	value := int32(0)
	if style == StatusBarContentStyleLight {
		value = 1
	}
	statusBarStyle.Store(value)

	if application := currentApplication(); application != nil {
		application.withWindows(func(window applicationWindow) {
			if state, ok := window.(*IosWindowState); ok {
				objc.SendVoid(state.viewController.ID(), objc.Sel("setNeedsStatusBarAppearanceUpdate"))
			}
		})
	}
}

func metalViewClass() objc.Class {
	metalViewClassOnce.Do(func() {
		touches := func(this objc.ID, _ objc.SEL, touches objc.ID, event objc.ID) {
			metalViewHandleTouches(this, touches, event)
		}
		methods := []objc.MethodDef{
			{Cmd: objc.Sel("dealloc"), Fn: nativeDealloc},
			{Cmd: objc.Sel("traitCollectionDidChange:"), Fn: metalViewTraitCollectionDidChange},
			{Cmd: objc.Sel("touchesBegan:withEvent:"), Fn: touches},
			{Cmd: objc.Sel("touchesMoved:withEvent:"), Fn: touches},
			{Cmd: objc.Sel("touchesEnded:withEvent:"), Fn: touches},
			{Cmd: objc.Sel("touchesCancelled:withEvent:"), Fn: touches},
			{Cmd: objc.Sel("handleScroll:"), Fn: metalViewHandleScroll},
			{Cmd: objc.Sel("handleBackGesture:"), Fn: metalViewHandleBackGesture},
			{Cmd: objc.Sel("gestureRecognizerShouldBegin:"), Fn: metalViewGestureRecognizerShouldBegin},
		}
		class, err := objc.RegisterClass(metalViewClassName, objc.GetClass("UIView"), []string{"NSObject", "UIGestureRecognizerDelegate"}, nil, methods)
		if err != nil {
			panic(fmt.Sprintf("failed to register %s: %v", metalViewClassName, err))
		}
		meta := pobjc.ID(class).Class()
		if !meta.AddMethod(objc.Sel("layerClass"), pobjc.NewIMP(metalViewLayerClass), "#@:") {
			panic(fmt.Sprintf("failed to add layerClass to %s", metalViewClassName))
		}
		metalViewClassVal = class
	})
	return metalViewClassVal
}

func metalViewLayerClass(_ objc.ID, _ objc.SEL) objc.Class {
	return objc.GetClass("CAMetalLayer")
}

func metalViewTraitCollectionDidChange(this objc.ID, _ objc.SEL, previous objc.ID) {
	objc.SendSuper[objc.ID](this, objc.Sel("traitCollectionDidChange:"), previous)
	traits := objc.Send[objc.ID](this, objc.Sel("traitCollection"))
	if objc.Send[bool](traits, objc.Sel("hasDifferentColorAppearanceComparedToTraitCollection:"), previous) {
		nativeWindowReference(this).withWindow((*IosWindowState).notifyAppearanceChanged)
	}
}

func metalViewHandleScroll(this objc.ID, _ objc.SEL, recognizer objc.ID) {
	nativeWindowReference(this).withWindow(func(window *IosWindowState) {
		window.handleScroll(uikit.UIPanGestureRecognizer{UIGestureRecognizer: uikit.UIGestureRecognizer(recognizer)})
	})
}

func metalViewHandleBackGesture(this objc.ID, _ objc.SEL, recognizer objc.ID) {
	nativeWindowReference(this).withWindow(func(window *IosWindowState) {
		window.handleBackGesture(uikit.UIScreenEdgePanGestureRecognizer{
			UIPanGestureRecognizer: uikit.UIPanGestureRecognizer{UIGestureRecognizer: uikit.UIGestureRecognizer(recognizer)},
		})
	})
}

func metalViewGestureRecognizerShouldBegin(this objc.ID, _ objc.SEL, recognizer objc.ID) bool {
	claimed, ok := windowReferenceWithR(nativeWindowReference(this), func(window *IosWindowState) bool {
		return window.beginBackGesture(uikit.UIGestureRecognizer(recognizer))
	})
	return ok && claimed
}

func metalViewHandleTouches(this objc.ID, touches objc.ID, event objc.ID) {
	for _, touch := range uikit.NSSetAllObjects(touches) {
		nativeWindowReference(this).withWindow(func(window *IosWindowState) {
			window.handleTouch(uikit.UITouch(touch), event)
		})
	}
}

func newMetalView(frame objc.CGRect) objc.ID {
	return objc.Send[objc.ID](objc.Alloc(metalViewClass()), objc.Sel("initWithFrame:"), frame)
}

// IosWindow is a UIKit window as a gpui.PlatformWindow. Close is its drop.
type IosWindow struct {
	gpui.PlatformWindowDefaults
	state  *IosWindowState
	closed bool
}

var _ gpui.PlatformWindow = (*IosWindow)(nil)

// IosWindowState is the shared state of an IosWindow that UIKit callbacks reach weakly.
type IosWindowState struct {
	platform                  weak.Pointer[iosPlatformState]
	keyboardObservers         []*objc.Retained
	window                    *objc.Retained
	viewController            *objc.Retained
	view                      *objc.Retained
	textInputView             *objc.Retained
	bounds                    gpui.Bounds[gpui.Pixels]
	scaleFactor               float32
	inputHandler              callbackSlot[*gpui.PlatformInputHandler]
	requestFrameCallback      callbackSlot[func(options gpui.RequestFrameOptions)]
	forceNextFrame            bool
	inputCallback             callbackSlot[func(input gpui.PlatformInput) gpui.DispatchEventResult]
	activeStatusCallback      callbackSlot[func(active bool)]
	visibility                gpui.WindowVisibility
	visibilityCallback        callbackSlot[func(visibility gpui.WindowVisibility)]
	hoverStatusCallback       callbackSlot[func(hovered bool)]
	resizeCallback            callbackSlot[func(size gpui.Size[gpui.Pixels], scaleFactor float32)]
	displayID                 gpui.DisplayID
	displayChangedCallback    callbackSlot[func()]
	movedCallback             callbackSlot[func()]
	shouldCloseCallback       callbackSlot[func() bool]
	hitTestCallback           callbackSlot[func() (gpui.WindowControlArea, bool)]
	closeCallback             callbackSlot[func()]
	appearanceChangedCallback callbackSlot[func()]
	insetsChangedCallback     callbackSlot[func(insets gpui.WindowInsets)]
	keyboardDismissTouch      gpui.Option[keyboardDismissTouch]
	backGestureStart          gpui.Option[gpui.Point[gpui.Pixels]]
	keyboardHeight            float32
	mousePosition             gpui.Point[gpui.Pixels]
	modifiers                 gpui.Modifiers
	rendererMu                sync.Mutex
	renderer                  *apple.MetalRenderer
	dispatchDepth             int
	dropped                   bool
	nativeReleased            bool
}

func newIosWindow(handle gpui.AnyWindowHandle, options gpui.WindowParams, platform *iosPlatformState) (*IosWindow, error) {
	screen := IosDisplayMain()
	screenBounds := screen.Bounds()
	scaleFactor := screen.Scale()
	screen.Release()

	if _, ok := objc.NewMainThreadMarker(); !ok {
		panic("UIKit requires the main thread")
	}
	var window objc.ID
	var screenBoundsCG objc.CGRect
	var scale objc.CGFloat
	if scene, ok := platform.application.windowScene(); ok {
		sceneScreen := objc.Send[objc.ID](scene.ID(), objc.Sel("screen"))
		screenBoundsCG = uikit.UIScreen(sceneScreen).Bounds()
		window = objc.Send[objc.ID](objc.Alloc(objc.GetClass("UIWindow")), objc.Sel("initWithWindowScene:"), scene.ID())
		objc.SendVoid(window, objc.Sel("setFrame:"), screenBoundsCG)
		scale = uikit.UIScreen(sceneScreen).Scale()
	} else {
		mainScreen := uikit.UIScreenMainScreen()
		screenBoundsCG = mainScreen.Bounds()
		window = objc.Send[objc.ID](objc.Alloc(objc.GetClass("UIWindow")), objc.Sel("initWithFrame:"), screenBoundsCG)
		scale = mainScreen.Scale()
	}

	metalFrame := objc.CGRect{Size: screenBoundsCG.Size}
	view := newMetalView(metalFrame)

	layer := objc.Send[objc.ID](view, objc.Sel("layer"))
	objc.SendVoid(layer, objc.Sel("setContentsScale:"), scale)
	if !uikit.IsKindOfClass(layer, "CAMetalLayer") {
		objc.Release(view)
		objc.Release(window)
		return nil, errors.New("UIKit view must have a CAMetalLayer")
	}
	renderer, err := apple.MetalRendererFromLayer(apple.NewInstanceBufferPool(), mtl.MetalLayer(layer), false)
	if err != nil {
		objc.Release(view)
		objc.Release(window)
		return nil, err
	}
	pixelW := int32(screenBoundsCG.Size.Width * scale)
	pixelH := int32(screenBoundsCG.Size.Height * scale)
	renderer.UpdateDrawableSize(gpui.NewSize(gpui.DevicePixels(pixelW), gpui.DevicePixels(pixelH)))

	objc.SendVoid(view, objc.Sel("setAutoresizingMask:"), uiViewAutoresizingFlexibleWidth|uiViewAutoresizingFlexibleHeight)

	objc.SendVoid(view, objc.Sel("setUserInteractionEnabled:"), true)
	objc.SendVoid(view, objc.Sel("setMultipleTouchEnabled:"), true)

	if reportsCornerRadii(view) {
		radius := objc.SendClass[objc.ID](objc.GetClass("UICornerRadius"), objc.Sel("containerConcentricRadius"))
		configuration := objc.SendClass[objc.ID](objc.GetClass("UICornerConfiguration"), objc.Sel("configurationWithRadius:"), radius)
		objc.SendVoid(view, objc.Sel("setCornerConfiguration:"), configuration)
	}

	scrollRecognizer := objc.Send[objc.ID](objc.Alloc(objc.GetClass("UIPanGestureRecognizer")), objc.Sel("initWithTarget:action:"), view, objc.Sel("handleScroll:"))
	objc.SendVoid(scrollRecognizer, objc.Sel("setAllowedScrollTypesMask:"), uiScrollTypeMaskAll)
	noTouchTypes := objc.NSArrayFromIDs(nil)
	objc.SendVoid(scrollRecognizer, objc.Sel("setAllowedTouchTypes:"), noTouchTypes)
	objc.Release(noTouchTypes)
	objc.SendVoid(scrollRecognizer, objc.Sel("setCancelsTouchesInView:"), false)
	objc.SendVoid(view, objc.Sel("addGestureRecognizer:"), scrollRecognizer)
	objc.Release(scrollRecognizer)

	backRecognizer := objc.Send[objc.ID](objc.Alloc(objc.GetClass("UIScreenEdgePanGestureRecognizer")), objc.Sel("initWithTarget:action:"), view, objc.Sel("handleBackGesture:"))
	edges := uikit.UIRectEdgeLeft
	if objc.Send[objc.NSInteger](view, objc.Sel("effectiveUserInterfaceLayoutDirection")) == uiUserInterfaceLayoutDirectionRightToLeft {
		edges = uikit.UIRectEdgeRight
	}
	objc.SendVoid(backRecognizer, objc.Sel("setEdges:"), edges)
	objc.SendVoid(backRecognizer, objc.Sel("setDelegate:"), view)
	objc.SendVoid(view, objc.Sel("addGestureRecognizer:"), backRecognizer)
	objc.Release(backRecognizer)

	viewController := newViewController()
	objc.SendVoid(viewController, objc.Sel("setView:"), view)
	objc.SendVoid(window, objc.Sel("setRootViewController:"), viewController)

	objc.SendVoid(window, objc.Sel("makeKeyAndVisible"))

	textInputFrame := objc.CGRect{Size: objc.CGSize{Width: 1.0, Height: 1.0}}
	textInputView := newTextInputView(textInputFrame)
	objc.SendVoid(textInputView, objc.Sel("setAlpha:"), objc.CGFloat(0.01))
	objc.SendVoid(textInputView, objc.Sel("setUserInteractionEnabled:"), true)
	objc.SendVoid(view, objc.Sel("addSubview:"), textInputView)

	windowScreen := IosDisplayFromScreen(uikit.UIScreen(objc.Send[objc.ID](window, objc.Sel("screen"))))
	displayID := windowScreen.ID()
	windowScreen.Release()

	state := &IosWindowState{
		platform:       weak.Make(platform),
		displayID:      displayID,
		window:         objc.Own(window),
		viewController: objc.Own(viewController),
		view:           objc.Own(view),
		textInputView:  objc.Own(textInputView),
		bounds:         screenBounds,
		scaleFactor:    scaleFactor,
		forceNextFrame: true,
		visibility:     gpui.WindowVisibilityVisible,
		renderer:       renderer,
	}
	w := &IosWindow{state: state}
	gpui.BindPlatformWindowDefaults(&w.PlatformWindowDefaults, w)
	return w, nil
}

func (w *IosWindow) register() {
	if platform := w.state.platform.Value(); platform != nil {
		platform.application.registerWindow(w.state)
	}

	nativeWindowReference(w.state.viewController.ID()).set(w.state)
	nativeWindowReference(w.state.view.ID()).set(w.state)
	textInputSetWindow(w.state.textInputView.ID(), w.state)

	w.state.registerKeyboardObservers()
}

func (s *IosWindowState) attachToScene(scene uikit.UIWindowScene) {
	objc.SendVoid(s.window.ID(), objc.Sel("setWindowScene:"), scene.ID())
	objc.SendVoid(s.window.ID(), objc.Sel("makeKeyAndVisible"))
	s.handleLayoutChange()
}

func (s *IosWindowState) registerKeyboardObservers() {
	center := objc.SendClass[objc.ID](objc.GetClass("NSNotificationCenter"), objc.Sel("defaultCenter"))

	window := weak.Make(s)
	frameChangeBlock := objc.NewBlock(func(_ objc.Block, notification objc.ID) {
		userInfo := objc.Send[objc.ID](notification, objc.Sel("userInfo"))
		if userInfo == objc.Nil {
			return
		}
		frameValue := objc.NSDictionaryObjectForKey(userInfo, uikit.Constant("UIKeyboardFrameEndUserInfoKey"))
		if frameValue == objc.Nil {
			return
		}
		if !uikit.IsKindOfClass(frameValue, "NSValue") {
			return
		}
		frame := objc.Send[objc.CGRect](frameValue, objc.Sel("CGRectValue"))
		if window := window.Value(); window != nil {
			window.setKeyboardHeight(float32(frame.Size.Height))
		}
	})

	hideBlock := objc.NewBlock(func(_ objc.Block, _ objc.ID) {
		if window := window.Value(); window != nil {
			window.setKeyboardHeight(0.)
		}
	})

	frameObserver := objc.Send[objc.ID](center, objc.Sel("addObserverForName:object:queue:usingBlock:"), uikit.Constant("UIKeyboardWillChangeFrameNotification"), objc.Nil, objc.Nil, frameChangeBlock)
	hideObserver := objc.Send[objc.ID](center, objc.Sel("addObserverForName:object:queue:usingBlock:"), uikit.Constant("UIKeyboardWillHideNotification"), objc.Nil, objc.Nil, hideBlock)
	frameChangeBlock.Release()
	hideBlock.Release()
	s.keyboardObservers = append(s.keyboardObservers, objc.RetainObject(frameObserver), objc.RetainObject(hideObserver))
}

func (s *IosWindowState) handleTouch(touch uikit.UITouch, _ objc.ID) {
	id := TouchID(touch)
	s.handleTouchSample(touch, id)
}

func (s *IosWindowState) handleTouchSample(touch uikit.UITouch, id gpui.TouchID) {
	position := TouchLocationInView(touch, uikit.UIView(s.view.ID()))
	s.mousePosition = position

	force := gpui.None[float32]()
	if value, ok := TouchForce(touch); ok {
		force = gpui.Some(value)
	}
	event := gpui.TouchEvent{
		ID:                id,
		Phase:             TouchPhase(touch),
		Position:          position,
		PredictedPosition: gpui.None[gpui.Point[gpui.Pixels]](),
		Force:             force,
	}
	s.handleKeyboardDismissTouch(event)
	s.inputCallback.with(func(callback *func(input gpui.PlatformInput) gpui.DispatchEventResult) {
		(*callback)(event)
	})
}

func (s *IosWindowState) handleScroll(recognizer uikit.UIPanGestureRecognizer) {
	event, ok := ScrollEvent(recognizer, uikit.UIView(s.view.ID()))
	if !ok {
		return
	}
	s.mousePosition = event.Position
	s.modifiers = event.Modifiers
	s.inputCallback.with(func(callback *func(input gpui.PlatformInput) gpui.DispatchEventResult) {
		(*callback)(event)
	})
}

func (s *IosWindowState) beginBackGesture(recognizer uikit.UIGestureRecognizer) bool {
	location := recognizer.LocationInView(uikit.UIView(s.view.ID()))
	startPosition := gpui.NewPoint(gpui.Px(float32(location.X)), gpui.Px(float32(location.Y)))
	result, ok := callbackSlotWithR(&s.inputCallback, func(callback *func(input gpui.PlatformInput) gpui.DispatchEventResult) gpui.DispatchEventResult {
		return (*callback)(gpui.BackGestureEvent{
			Phase:         gpui.BackGesturePhaseStarted,
			StartPosition: startPosition,
			Progress:      0.,
		})
	})
	claimed := ok && result.DefaultPrevented
	if claimed {
		s.backGestureStart = gpui.Some(startPosition)
	} else {
		s.backGestureStart = gpui.None[gpui.Point[gpui.Pixels]]()
	}
	return claimed
}

func (s *IosWindowState) handleBackGesture(recognizer uikit.UIScreenEdgePanGestureRecognizer) {
	startPosition, ok := s.backGestureStart.Get()
	if !ok {
		return
	}
	event, ok := BackEvent(recognizer, uikit.UIView(s.view.ID()), startPosition)
	if !ok {
		return
	}
	if event.Phase == gpui.BackGesturePhaseCommitted || event.Phase == gpui.BackGesturePhaseCancelled {
		s.backGestureStart = gpui.None[gpui.Point[gpui.Pixels]]()
	}
	s.inputCallback.with(func(callback *func(input gpui.PlatformInput) gpui.DispatchEventResult) {
		(*callback)(event)
	})
}

func (s *IosWindowState) handleKeyboardDismissTouch(event gpui.TouchEvent) {
	if s.keyboardHeight <= 0. {
		s.keyboardDismissTouch = gpui.None[keyboardDismissTouch]()
		return
	}

	switch event.Phase {
	case gpui.TouchPhaseStarted:
		if s.keyboardDismissTouch.IsNone() {
			s.keyboardDismissTouch = gpui.Some(keyboardDismissTouch{id: event.ID, startPosition: event.Position})
		}
	case gpui.TouchPhaseMoved:
		touch, ok := s.keyboardDismissTouch.Get()
		if !ok || touch.id != event.ID {
			return
		}
		delta := event.Position.Sub(touch.startPosition)
		if delta.Y >= keyboardDismissDistance && delta.Y.Abs() > delta.X.Abs()*1.25 {
			s.keyboardDismissTouch = gpui.None[keyboardDismissTouch]()
			s.hideKeyboard()
		}
	case gpui.TouchPhaseEnded, gpui.TouchPhaseCancelled:
		if touch, ok := s.keyboardDismissTouch.Get(); ok && touch.id == event.ID {
			s.keyboardDismissTouch = gpui.None[keyboardDismissTouch]()
		}
	}
}

func (s *IosWindowState) requestFrame() {
	s.requestFrameCallback.with(func(callback *func(options gpui.RequestFrameOptions)) {
		forceRender := s.forceNextFrame
		s.forceNextFrame = false
		(*callback)(gpui.RequestFrameOptions{ForceRender: forceRender})
	})
}

func (s *IosWindowState) safeAreaInsets() (top, bottom, left, right float32) {
	insets := objc.Send[uiEdgeInsets](s.view.ID(), objc.Sel("safeAreaInsets"))
	return float32(insets.Top), float32(insets.Bottom), float32(insets.Left), float32(insets.Right)
}

type uiEdgeInsets struct {
	_                        structs.HostLayout
	Top, Left, Bottom, Right objc.CGFloat
}

func (s *IosWindowState) currentInsets() gpui.WindowInsets {
	top, bottom, left, right := s.safeAreaInsets()
	return gpui.WindowInsets{
		SafeArea: gpui.Edges[gpui.Pixels]{
			Top:    gpui.Px(top),
			Right:  gpui.Px(right),
			Bottom: gpui.Px(bottom),
			Left:   gpui.Px(left),
		},
		IME: gpui.Edges[gpui.Pixels]{
			Bottom: gpui.Px(s.keyboardHeight),
		},
		CornerRadii: s.displayCornerRadii(),
	}
}

func (s *IosWindowState) displayCornerRadii() gpui.Corners[gpui.Pixels] {
	if !reportsCornerRadii(s.view.ID()) {
		return gpui.Corners[gpui.Pixels]{}
	}
	radius := func(corner objc.NSUInteger) gpui.Pixels {
		return gpui.Px(float32(objc.Send[objc.CGFloat](s.view.ID(), objc.Sel("effectiveRadiusForCorner:"), corner)))
	}
	return gpui.Corners[gpui.Pixels]{
		TopLeft:     radius(uiRectCornerTopLeft),
		TopRight:    radius(uiRectCornerTopRight),
		BottomRight: radius(uiRectCornerBottomRight),
		BottomLeft:  radius(uiRectCornerBottomLeft),
	}
}

func (s *IosWindowState) notifyInsetsChanged() {
	s.insetsChangedCallback.with(func(callback *func(insets gpui.WindowInsets)) {
		(*callback)(s.currentInsets())
	})
}

func (s *IosWindowState) notifyAppearanceChanged() {
	s.appearanceChangedCallback.with(func(callback *func()) { (*callback)() })
}

func (s *IosWindowState) setKeyboardHeight(height float32) {
	height = max(height, 0.)
	if float32(math.Abs(float64(s.keyboardHeight-height))) <= 0.5 {
		return
	}
	s.keyboardHeight = height
	if height <= 0. {
		s.keyboardDismissTouch = gpui.None[keyboardDismissTouch]()
	}
	s.notifyInsetsChanged()
}

func (s *IosWindowState) showKeyboard() {
	textInputSetKeyboardVisible(s.textInputView.ID(), true)
}

func (s *IosWindowState) hideKeyboard() {
	textInputSetKeyboardVisible(s.textInputView.ID(), false)
}

func iosWindowStateWithInputHandlerR[R any](s *IosWindowState, callback func(handler *gpui.PlatformInputHandler) R) (R, bool) {
	return callbackSlotWithR(&s.inputHandler, func(handler **gpui.PlatformInputHandler) R {
		return callback(*handler)
	})
}

func (s *IosWindowState) notifyActiveStatusChange(isActive bool) {
	slog.Info(fmt.Sprintf("GPUI iOS: Window active status changed to: %t", isActive))

	s.activeStatusCallback.with(func(callback *func(active bool)) { (*callback)(isActive) })
}

func (s *IosWindowState) notifyVisibilityChange(visibility gpui.WindowVisibility) {
	previous := s.visibility
	s.visibility = visibility
	if previous != visibility {
		s.visibilityCallback.with(func(callback *func(visibility gpui.WindowVisibility)) {
			(*callback)(visibility)
		})
	}
}

func (s *IosWindowState) handleLayoutChange() {
	viewBounds := uikit.UIView(s.view.ID()).Bounds()
	display := IosDisplayFromScreen(uikit.UIScreen(objc.Send[objc.ID](s.window.ID(), objc.Sel("screen"))))
	defer display.Release()
	previousDisplayID := s.displayID
	s.displayID = display.ID()
	if previousDisplayID != display.ID() {
		s.displayChangedCallback.with(func(callback *func()) { (*callback)() })
	}
	scale := float64(display.Scale())

	width := float32(viewBounds.Size.Width)
	height := float32(viewBounds.Size.Height)
	newScale := float32(scale)

	oldBounds := s.bounds
	oldScale := s.scaleFactor

	newSize := gpui.NewSize(gpui.Px(width), gpui.Px(height))
	s.notifyInsetsChanged()

	if oldBounds.Size == newSize && math.Abs(float64(oldScale-newScale)) < 0.01 {
		return
	}

	slog.Info(fmt.Sprintf("GPUI iOS: Layout changed — %v @%.1fx → %v @%.1fx", oldBounds.Size, oldScale, newSize, newScale))

	s.bounds = gpui.Bounds[gpui.Pixels]{Size: newSize}
	s.scaleFactor = newScale
	objc.SendVoid(objc.Send[objc.ID](s.view.ID(), objc.Sel("layer")), objc.Sel("setContentsScale:"), scale)

	pixelWidth := int32(width * newScale)
	pixelHeight := int32(height * newScale)
	s.rendererMu.Lock()
	s.renderer.UpdateDrawableSize(gpui.NewSize(gpui.DevicePixels(pixelWidth), gpui.DevicePixels(pixelHeight)))
	s.rendererMu.Unlock()

	s.resizeCallback.with(func(callback *func(size gpui.Size[gpui.Pixels], scaleFactor float32)) {
		(*callback)(newSize, newScale)
	})
}

func (s *IosWindowState) endDispatch() {
	s.dispatchDepth--
	s.releaseNativeIfDropped()
}

func (s *IosWindowState) releaseNativeIfDropped() {
	if !s.dropped || s.dispatchDepth > 0 || s.nativeReleased {
		return
	}
	s.nativeReleased = true
	s.rendererMu.Lock()
	s.renderer.Destroy()
	s.rendererMu.Unlock()
	s.textInputView.Release()
	s.view.Release()
	s.viewController.Release()
	s.window.Release()
}

func reportsCornerRadii(view objc.ID) bool {
	return objc.RespondsToSelector(view, objc.Sel("effectiveRadiusForCorner:"))
}

// Close unregisters the window, removes its keyboard observers and detaches its native views
// from the window state. It is the window's drop and is idempotent; native objects are released
// once no UIKit callback into the window is still running.
func (w *IosWindow) Close() {
	if w.closed {
		return
	}
	w.closed = true
	s := w.state
	if platform := s.platform.Value(); platform != nil {
		platform.application.unregisterWindow(s)
	}
	observers := s.keyboardObservers
	s.keyboardObservers = nil
	center := objc.SendClass[objc.ID](objc.GetClass("NSNotificationCenter"), objc.Sel("defaultCenter"))
	for _, observer := range observers {
		objc.SendVoid(center, objc.Sel("removeObserver:"), observer.ID())
		observer.Release()
	}

	nativeWindowReference(s.viewController.ID()).set(nil)
	nativeWindowReference(s.view.ID()).set(nil)
	textInputSetWindow(s.textInputView.ID(), nil)
	objc.SendVoid(s.textInputView.ID(), objc.Sel("removeFromSuperview"))

	s.dropped = true
	s.releaseNativeIfDropped()
}

// RawWindowHandle returns the Metal-backed UIView.
func (w *IosWindow) RawWindowHandle() gpui.RawWindowHandle {
	return gpui.RawWindowHandle{Kind: gpui.RawWindowHandleUIKit, UIKitView: uintptr(w.state.view.ID())}
}

// Bounds returns the window bounds in logical pixels.
func (w *IosWindow) Bounds() gpui.Bounds[gpui.Pixels] { return w.state.bounds }

// IsMaximized reports true: iOS windows are always maximized.
func (w *IosWindow) IsMaximized() bool { return true }

// WindowBounds returns the bounds as fullscreen.
func (w *IosWindow) WindowBounds() gpui.WindowBounds {
	return gpui.WindowBoundsFullscreen(w.state.bounds)
}

// ContentSize returns the window size.
func (w *IosWindow) ContentSize() gpui.Size[gpui.Pixels] { return w.state.bounds.Size }

// Resize does nothing: iOS windows cannot be resized programmatically.
func (w *IosWindow) Resize(size gpui.Size[gpui.Pixels]) {}

// ScaleFactor returns the display scale.
func (w *IosWindow) ScaleFactor() float32 { return w.state.scaleFactor }

// Appearance returns the view's interface style.
func (w *IosWindow) Appearance() gpui.WindowAppearance {
	traits := objc.Send[objc.ID](w.state.view.ID(), objc.Sel("traitCollection"))
	if uikit.UIUserInterfaceStyle(objc.Send[objc.NSInteger](traits, objc.Sel("userInterfaceStyle"))) == uikit.UIUserInterfaceStyleDark {
		return gpui.WindowAppearanceDark
	}
	return gpui.WindowAppearanceLight
}

// Display returns the window's screen.
func (w *IosWindow) Display() gpui.PlatformDisplay {
	return IosDisplayFromScreen(uikit.UIScreen(objc.Send[objc.ID](w.state.window.ID(), objc.Sel("screen"))))
}

// OnDisplayChanged registers the callback run when the window moves to another screen.
func (w *IosWindow) OnDisplayChanged(callback func()) { w.state.displayChangedCallback.set(callback) }

// MousePosition returns the last touch position.
func (w *IosWindow) MousePosition() gpui.Point[gpui.Pixels] { return w.state.mousePosition }

// Modifiers returns the last modifiers.
func (w *IosWindow) Modifiers() gpui.Modifiers { return w.state.modifiers }

// Capslock reports capslock off.
func (w *IosWindow) Capslock() gpui.Capslock { return gpui.Capslock{On: false} }

// SetInputHandler installs the focused input handler and refreshes the keyboard.
func (w *IosWindow) SetInputHandler(inputHandler *gpui.PlatformInputHandler) {
	field := gpui.None[gpui.FocusID]()
	if id, ok := inputHandler.FocusID(); ok {
		field = gpui.Some(id)
	}
	textInputSetInputField(w.state.textInputView.ID(), field)
	w.state.inputHandler.set(inputHandler)
	textInputRefreshKeyboard(w.state.textInputView.ID())
}

// TakeInputHandler removes the input handler and refreshes the keyboard.
func (w *IosWindow) TakeInputHandler() *gpui.PlatformInputHandler {
	handler, _ := w.state.inputHandler.take()
	textInputRefreshKeyboard(w.state.textInputView.ID())
	return handler
}

// Prompt returns nil: native prompts are not implemented on iOS.
func (w *IosWindow) Prompt(level gpui.PromptLevel, msg string, detail gpui.Option[string], answers []gpui.PromptButton) *scheduler.OneshotReceiver[int] {
	return nil
}

// Activate makes the window key and visible.
func (w *IosWindow) Activate() {
	objc.SendVoid(w.state.window.ID(), objc.Sel("makeKeyAndVisible"))
}

// IsActive reports whether the window is the key window.
func (w *IosWindow) IsActive() bool {
	return uikit.UIWindow(w.state.window.ID()).IsKeyWindow()
}

// IsHovered reports false: hover does not apply on iOS.
func (w *IosWindow) IsHovered() bool { return false }

// SetTitle does nothing: iOS windows have no titles.
func (w *IosWindow) SetTitle(title string) {}

// BackgroundAppearance reports opaque.
func (w *IosWindow) BackgroundAppearance() gpui.WindowBackgroundAppearance {
	return gpui.WindowBackgroundAppearanceOpaque
}

// SetBackgroundAppearance does nothing on iOS.
func (w *IosWindow) SetBackgroundAppearance(backgroundAppearance gpui.WindowBackgroundAppearance) {}

// Minimize does nothing: iOS apps cannot be minimized.
func (w *IosWindow) Minimize() {}

// Zoom does nothing: iOS apps cannot be zoomed.
func (w *IosWindow) Zoom() {}

// ToggleFullscreen does nothing: iOS apps are always fullscreen.
func (w *IosWindow) ToggleFullscreen() {}

// IsFullscreen reports true.
func (w *IosWindow) IsFullscreen() bool { return true }

// OnRequestFrame registers the frame callback driven by the display link.
func (w *IosWindow) OnRequestFrame(callback func(options gpui.RequestFrameOptions)) {
	w.state.requestFrameCallback.set(callback)
}

// OnInput registers the input callback.
func (w *IosWindow) OnInput(callback func(input gpui.PlatformInput) gpui.DispatchEventResult) {
	w.state.inputCallback.set(callback)
}

// OnActiveStatusChange registers the active-status callback.
func (w *IosWindow) OnActiveStatusChange(callback func(active bool)) {
	w.state.activeStatusCallback.set(callback)
}

// Visibility returns the window visibility.
func (w *IosWindow) Visibility() gpui.WindowVisibility { return w.state.visibility }

// OnVisibilityChange registers the visibility callback.
func (w *IosWindow) OnVisibilityChange(callback func(visibility gpui.WindowVisibility)) {
	w.state.visibilityCallback.set(callback)
}

// OnHoverStatusChange registers the hover callback, which iOS never calls.
func (w *IosWindow) OnHoverStatusChange(callback func(hovered bool)) {
	w.state.hoverStatusCallback.set(callback)
}

// OnResize registers the resize callback.
func (w *IosWindow) OnResize(callback func(size gpui.Size[gpui.Pixels], scaleFactor float32)) {
	w.state.resizeCallback.set(callback)
}

// OnMoved registers the move callback, which iOS never calls.
func (w *IosWindow) OnMoved(callback func()) { w.state.movedCallback.set(callback) }

// OnShouldClose registers the should-close callback.
func (w *IosWindow) OnShouldClose(callback func() bool) { w.state.shouldCloseCallback.set(callback) }

// OnHitTestWindowControl registers the hit-test callback.
func (w *IosWindow) OnHitTestWindowControl(callback func() (gpui.WindowControlArea, bool)) {
	w.state.hitTestCallback.set(callback)
}

// OnClose registers the close callback.
func (w *IosWindow) OnClose(callback func()) { w.state.closeCallback.set(callback) }

// OnAppearanceChanged registers the appearance callback.
func (w *IosWindow) OnAppearanceChanged(callback func()) {
	w.state.appearanceChangedCallback.set(callback)
}

// Draw renders scene with Metal.
func (w *IosWindow) Draw(scene *gpui.Scene) {
	w.state.rendererMu.Lock()
	defer w.state.rendererMu.Unlock()
	w.state.renderer.Draw(scene)
}

// SpriteAtlas returns the renderer's atlas.
func (w *IosWindow) SpriteAtlas() gpui.PlatformAtlas {
	w.state.rendererMu.Lock()
	defer w.state.rendererMu.Unlock()
	return w.state.renderer.SpriteAtlas()
}

// IsSubpixelRenderingSupported reports false.
func (w *IosWindow) IsSubpixelRenderingSupported() bool { return false }

// GPUSpecs reports none.
func (w *IosWindow) GPUSpecs() (gpui.GPUSpecs, bool) { return gpui.GPUSpecs{}, false }

// UpdateIMEPosition does nothing: iOS positions the IME itself.
func (w *IosWindow) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {}

// Insets returns the safe-area, keyboard and corner insets.
func (w *IosWindow) Insets() gpui.WindowInsets { return w.state.currentInsets() }

// OnInsetsChanged registers the insets callback.
func (w *IosWindow) OnInsetsChanged(callback func(insets gpui.WindowInsets)) {
	w.state.insetsChangedCallback.set(callback)
}

// ShowSoftKeyboard requests the software keyboard.
func (w *IosWindow) ShowSoftKeyboard() { w.state.showKeyboard() }

// HideSoftKeyboard dismisses the software keyboard.
func (w *IosWindow) HideSoftKeyboard() { w.state.hideKeyboard() }

// TextInputStateChanged forwards a text input state change to the native text input view.
func (w *IosWindow) TextInputStateChanged(change gpui.TextInputStateChange) {
	textInputStateChanged(w.state.textInputView.ID(), change)
}

// SetTextInputConfiguration applies the focused field's keyboard configuration.
func (w *IosWindow) SetTextInputConfiguration(configuration gpui.TextInputConfiguration) {
	textInputSetConfiguration(w.state.textInputView.ID(), configuration)
}
