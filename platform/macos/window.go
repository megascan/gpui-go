//go:build darwin && !ios

package macos

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"image"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unsafe"
	"weak"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/gcd"
	"github.com/megascan/gpui-go/internal/apple/mtl"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/platform/apple"
	"github.com/megascan/gpui-go/scheduler"
)

const windowStateIvar = "windowState"

var restoresWorkspaceAtLaunchDefault sync.Once

const (
	nsNormalWindowLevel   objc.NSInteger = 0
	nsFloatingWindowLevel objc.NSInteger = 3
	nsPopUpWindowLevel    objc.NSInteger = 101
)

// UserTabbingPreference is the user's "Prefer tabs when opening documents" system setting.
type UserTabbingPreference uint8

const (
	// UserTabbingPreferenceNever never opens new windows as tabs.
	UserTabbingPreferenceNever UserTabbingPreference = iota
	// UserTabbingPreferenceAlways always opens new windows as tabs.
	UserTabbingPreferenceAlways
	// UserTabbingPreferenceInFullScreen opens new windows as tabs only in full screen.
	UserTabbingPreferenceInFullScreen
)

type rendererContext = apple.Context

func newRendererContext() rendererContext {
	return apple.NewInstanceBufferPool()
}

var (
	windowStatesMu sync.Mutex
	windowStates   = map[objc.ID]*lockedWindowState{}
)

func storeWindowState(object objc.ID, state *lockedWindowState) {
	windowStatesMu.Lock()
	defer windowStatesMu.Unlock()
	windowStates[object] = state
}

func loadWindowState(object objc.ID) (*lockedWindowState, bool) {
	windowStatesMu.Lock()
	defer windowStatesMu.Unlock()
	state, ok := windowStates[object]
	return state, ok
}

func clearWindowState(object objc.ID) (*lockedWindowState, bool) {
	windowStatesMu.Lock()
	defer windowStatesMu.Unlock()
	state, ok := windowStates[object]
	delete(windowStates, object)
	return state, ok
}

func windowStateArchiverWillEncodeObject(_ objc.ID, _ objc.SEL, _ objc.ID, object objc.ID) objc.ID {
	if object == objc.Nil {
		return object
	}
	isView := objc.Send[bool](object, objc.Sel("isKindOfClass:"), objc.GetClass("NSView"))
	isWindow := objc.Send[bool](object, objc.Sel("isKindOfClass:"), objc.GetClass("NSWindow"))
	if isView || isWindow {
		return objc.Nil
	}
	return object
}

func windowStateUnarchiverRestorationOptions(_ objc.ID, _ objc.SEL) objc.ID {
	if !isMacosVersionAtLeast(nsOperatingSystemVersion{MajorVersion: 15, MinorVersion: 0, PatchVersion: 0}) {
		return objc.Nil
	}
	class := objc.GetClass("NSWindowRestorationOptions")
	if class == 0 {
		return objc.Nil
	}
	options := objc.Send[objc.ID](objc.Alloc(class), objc.Sel("init"))
	return objc.Autorelease(options)
}

func convertMousePosition(position objc.CGPoint, windowHeight gpui.Pixels) gpui.Point[gpui.Pixels] {
	return gpui.NewPoint(
		gpui.Px(float32(position.X)),
		windowHeight-gpui.Px(float32(position.Y)),
	)
}

func setActiveWindowCursorStyle(style gpui.CursorStyle) {
	app := appkit.NSApplicationSharedApplication()
	keyWindow := app.KeyWindow().ID()
	mainWindow := app.MainWindow().ID()
	var activeWindow objc.ID
	switch {
	case keyWindow != objc.Nil && isGpuiWindow(keyWindow):
		activeWindow = keyWindow
	case mainWindow != objc.Nil && isGpuiWindow(mainWindow):
		activeWindow = mainWindow
	default:
		return
	}

	windowState := getWindowState(activeWindow)
	windowState.Lock()
	defer windowState.Unlock()
	if windowState.cursorStyle != style {
		windowState.cursorStyle = style
		objc.SendVoid(windowState.nativeWindow, objc.Sel("invalidateCursorRectsForView:"), windowState.nativeView)
	}
}

type trafficLightFrames struct {
	titlebar objc.CGRect
	close    objc.CGRect
	minimize objc.CGRect
	zoom     objc.CGRect
}

type trafficLightButtons struct {
	close    appkit.NSView
	minimize appkit.NSView
	zoom     appkit.NSView
}

const (
	nsApplicationPresentationAutoHideDock    objc.NSUInteger = 1 << 0
	nsApplicationPresentationAutoHideMenuBar objc.NSUInteger = 1 << 2
)

type simpleFullscreenState struct {
	frame     objc.CGRect
	bounds    gpui.Bounds[gpui.Pixels]
	styleMask appkit.NSWindowStyleMask
}

type simpleFullscreenPlan struct {
	enter       bool
	screenFrame objc.CGRect
	exit        simpleFullscreenState
}

type simpleFullscreenAppStateT struct {
	windowCount              int
	savedPresentationOptions objc.NSUInteger
}

var (
	simpleFullscreenAppStateMu sync.Mutex
	simpleFullscreenAppState   *simpleFullscreenAppStateT
)

func pushSimpleFullscreenPresentationOptions() {
	simpleFullscreenAppStateMu.Lock()
	defer simpleFullscreenAppStateMu.Unlock()
	if simpleFullscreenAppState != nil {
		simpleFullscreenAppState.windowCount++
		return
	}
	app := appkit.NSApplicationSharedApplication().ID()
	savedPresentationOptions := objc.Send[objc.NSUInteger](app, objc.Sel("presentationOptions"))
	objc.SendVoid(app, objc.Sel("setPresentationOptions:"), nsApplicationPresentationAutoHideDock|nsApplicationPresentationAutoHideMenuBar)
	simpleFullscreenAppState = &simpleFullscreenAppStateT{
		windowCount:              1,
		savedPresentationOptions: savedPresentationOptions,
	}
}

func popSimpleFullscreenPresentationOptions() {
	simpleFullscreenAppStateMu.Lock()
	defer simpleFullscreenAppStateMu.Unlock()
	if state := simpleFullscreenAppState; state != nil {
		if state.windowCount > 0 {
			state.windowCount--
		}
		if state.windowCount == 0 {
			app := appkit.NSApplicationSharedApplication().ID()
			objc.SendVoid(app, objc.Sel("setPresentationOptions:"), state.savedPresentationOptions)
			simpleFullscreenAppState = nil
		}
	}
}

func applySimpleFullscreenPlan(nativeWindow objc.ID, nativeView objc.ID, plan simpleFullscreenPlan) {
	window := appkit.NSWindow(nativeWindow)
	if plan.enter {
		pushSimpleFullscreenPresentationOptions()
		window.SetStyleMask(appkit.NSWindowStyleMaskBorderless)
		window.SetFrameDisplay(plan.screenFrame, true)
	} else {
		popSimpleFullscreenPresentationOptions()
		window.SetStyleMask(plan.exit.styleMask)
		window.SetFrameDisplay(plan.exit.frame, true)
	}

	window.MakeKeyAndOrderFront(objc.Nil)
	window.MakeFirstResponder(nativeView)
}

type frameSource struct {
	metal       *MetalFrames
	displayLink *WindowFrameSource
}

type macWindowState struct {
	handle                          gpui.AnyWindowHandle
	foregroundExecutor              gpui.ForegroundExecutor
	backgroundExecutor              gpui.BackgroundExecutor
	nativeWindow                    objc.ID
	nativeView                      objc.ID
	blurredView                     objc.ID
	backgroundAppearance            gpui.WindowBackgroundAppearance
	cursorStyle                     gpui.CursorStyle
	cursorVisible                   *atomic.Bool
	frameSource                     frameSource
	renderer                        *apple.Renderer
	requestFrameCallback            func(options gpui.RequestFrameOptions)
	eventCallback                   func(input gpui.PlatformInput) gpui.DispatchEventResult
	activateCallback                func(active bool)
	visibilityCallback              func(visibility gpui.WindowVisibility)
	lastVisibility                  gpui.Option[gpui.WindowVisibility]
	resizeCallback                  func(size gpui.Size[gpui.Pixels], scaleFactor float32)
	movedCallback                   func()
	displayChangedCallback          func()
	shouldCloseCallback             func() bool
	closeCallback                   func()
	appearanceChangedCallback       func()
	inputHandler                    *gpui.PlatformInputHandler
	lastKeyEquivalent               *gpui.KeyDownEvent
	lastLeftMouseDownEvent          *objc.Retained
	syntheticDragCounter            int
	trafficLightPosition            gpui.Option[gpui.Point[gpui.Pixels]]
	nativeTitlebarVisible           bool
	trafficLightFrames              *trafficLightFrames
	preFullscreenTrafficLightFrames *trafficLightFrames
	transparentTitlebar             bool
	previousModifiersChangedEvent   gpui.PlatformInput
	keystrokeForDoCommand           *gpui.Keystroke
	doCommandHandled                gpui.Option[bool]
	externalFilesDragged            bool
	firstMouse                      bool
	appOwnsTitlebarDrag             bool
	fullscreenRestoreBounds         gpui.Bounds[gpui.Pixels]
	isExitingFullscreen             bool
	simpleFullscreenState           *simpleFullscreenState
	moveTabToNewWindowCallback      func()
	mergeAllWindowsCallback         func()
	selectNextTabCallback           func()
	selectPreviousTabCallback       func()
	toggleTabBarCallback            func()
	activatedLeastOnce              bool
	closed                          *atomic.Bool
	a11yAdapter                     a11yAdapter
	sheetParent                     objc.ID
	nativeViews                     nativeViews
}

type lockedWindowState struct {
	sync.Mutex
	macWindowState
}

func (s *macWindowState) moveTrafficLight() {
	if !s.nativeTitlebarVisible {
		s.updateNativeTitlebarVisibility()
	}

	trafficLightPosition, ok := s.trafficLightPosition.Get()
	if !ok {
		return
	}
	if s.isFullscreen() && !s.isExitingFullscreen {
		s.restoreTrafficLight()
		return
	}

	if s.trafficLightFrames == nil {
		s.trafficLightFrames = s.captureTrafficLightFrames()
	}

	windowHeight := gpui.Px(float32(s.window().Frame().Size.Height))
	if s.trafficLightFrames == nil {
		return
	}
	buttons, ok := s.trafficLightButtons()
	if !ok {
		return
	}
	titlebarContainer, ok := titlebarContainer(buttons.close)
	if !ok {
		return
	}

	closeFrame := buttons.close.Frame()
	minimizeFrame := buttons.minimize.Frame()
	buttonWidth := gpui.Px(float32(closeFrame.Size.Width))
	buttonHeight := gpui.Px(float32(closeFrame.Size.Height))
	buttonPadding := gpui.Px(float32(minimizeFrame.Origin.X - closeFrame.Origin.X - closeFrame.Size.Width))
	containerHeight := buttonHeight + trafficLightPosition.Y + trafficLightPosition.Y

	titlebarFrame := titlebarContainer.Frame()
	titlebarFrame.Size.Height = containerHeight.ToF64()
	titlebarFrame.Origin.Y = (windowHeight - containerHeight).ToF64()

	minimizeX := trafficLightPosition.X + buttonWidth + buttonPadding
	zoomX := minimizeX + buttonWidth + buttonPadding

	titlebarContainer.SetFrame(titlebarFrame)
	setFrameOrigin(buttons.close, objc.MakeCGPoint(trafficLightPosition.X.ToF64(), trafficLightPosition.Y.ToF64()))
	setFrameOrigin(buttons.minimize, objc.MakeCGPoint(minimizeX.ToF64(), trafficLightPosition.Y.ToF64()))
	setFrameOrigin(buttons.zoom, objc.MakeCGPoint(zoomX.ToF64(), trafficLightPosition.Y.ToF64()))

	updateTrackingAreas(titlebarContainer)
	updateTrackingAreas(buttons.close)
	updateTrackingAreas(buttons.minimize)
	updateTrackingAreas(buttons.zoom)
}

func setFrameOrigin(view appkit.NSView, origin objc.CGPoint) {
	objc.SendVoid(view.ID(), objc.Sel("setFrameOrigin:"), origin)
}

func updateTrackingAreas(view appkit.NSView) {
	objc.SendVoid(view.ID(), objc.Sel("updateTrackingAreas"))
}

func (s *macWindowState) updateNativeTitlebarVisibility() {
	closeButton := s.window().StandardWindowButton(appkit.NSWindowCloseButton)
	if closeButton == 0 {
		return
	}
	if titlebarContainer, ok := titlebarContainer(closeButton); ok {
		titlebarContainer.SetHidden(!s.nativeTitlebarVisible)
	}
}

func (s *macWindowState) applyFullscreenTitlebarOpacity() {
	minVersion := nsOperatingSystemVersion{MajorVersion: 15, MinorVersion: 3, PatchVersion: 0}
	if isMacosVersionAtLeast(minVersion) {
		s.window().SetTitlebarAppearsTransparent(false)
	}
}

func (s *macWindowState) restoreWindowedTitlebarTransparency() {
	minVersion := nsOperatingSystemVersion{MajorVersion: 15, MinorVersion: 3, PatchVersion: 0}
	if isMacosVersionAtLeast(minVersion) && s.transparentTitlebar {
		s.window().SetTitlebarAppearsTransparent(true)
	}
}

func (s *macWindowState) captureTrafficLightFrames() *trafficLightFrames {
	buttons, ok := s.trafficLightButtons()
	if !ok {
		return nil
	}
	titlebarContainer, ok := titlebarContainer(buttons.close)
	if !ok {
		return nil
	}
	return &trafficLightFrames{
		titlebar: titlebarContainer.Frame(),
		close:    buttons.close.Frame(),
		minimize: buttons.minimize.Frame(),
		zoom:     buttons.zoom.Frame(),
	}
}

func (s *macWindowState) window() appkit.NSWindow {
	return appkit.NSWindow(s.nativeWindow)
}

func (s *macWindowState) trafficLightButtons() (trafficLightButtons, bool) {
	window := s.window()
	closeButton := window.StandardWindowButton(appkit.NSWindowCloseButton)
	if closeButton == 0 {
		return trafficLightButtons{}, false
	}
	minimize := window.StandardWindowButton(appkit.NSWindowMiniaturizeButton)
	if minimize == 0 {
		return trafficLightButtons{}, false
	}
	zoom := window.StandardWindowButton(appkit.NSWindowZoomButton)
	if zoom == 0 {
		return trafficLightButtons{}, false
	}
	return trafficLightButtons{close: closeButton, minimize: minimize, zoom: zoom}, true
}

func titlebarContainer(closeButton appkit.NSView) (appkit.NSView, bool) {
	buttonContainer := closeButton.Superview()
	if buttonContainer == 0 {
		return 0, false
	}
	container := buttonContainer.Superview()
	return container, container != 0
}

func (s *macWindowState) restoreTrafficLight() {
	frames := s.trafficLightFrames
	if frames == nil {
		return
	}
	s.trafficLightFrames = nil
	buttons, ok := s.trafficLightButtons()
	if !ok {
		return
	}
	titlebarContainer, ok := titlebarContainer(buttons.close)
	if !ok {
		return
	}

	buttons.close.SetFrame(frames.close)
	buttons.minimize.SetFrame(frames.minimize)
	buttons.zoom.SetFrame(frames.zoom)
	titlebarContainer.SetFrame(frames.titlebar)

	updateTrackingAreas(titlebarContainer)
	updateTrackingAreas(buttons.close)
	updateTrackingAreas(buttons.minimize)
	updateTrackingAreas(buttons.zoom)
}

func (s *macWindowState) startDisplayLink(windowState *lockedWindowState) {
	s.stopDisplayLink()
	if s.window().OcclusionState()&appkit.NSWindowOcclusionStateVisible == 0 {
		return
	}
	if metalFrames := s.frameSource.metal; metalFrames != nil {
		layer, ok := s.renderer.Layer()
		if !ok {
			return
		}
		mainThread, ok := objc.NewMainThreadMarker()
		if !ok {
			panic("macOS windows require the main thread")
		}
		weakState := weak.Make(windowState)
		source := apple.NewMetalFrameSource(layer.ID(), mainThread, func(update objc.ID) {
			if windowState := weakState.Value(); windowState != nil {
				metalFrame(windowState, update)
			}
		})
		if source != nil {
			metalFrames.Replace(source)
			s.renderer.SetDisplayLinkAttached(true)
		}
		return
	}

	screen := objc.Send[objc.ID](s.nativeWindow, objc.Sel("screen"))
	displayID, ok := displayIDForScreen(screen)
	if !ok {
		return
	}
	if s.frameSource.displayLink == nil {
		s.frameSource.displayLink = NewWindowFrameSource(uintptr(s.nativeView), stepCallback())
	}
	if err := s.frameSource.displayLink.Start(displayID); err != nil {
		slog.Error(err.Error())
	}
}

func (s *macWindowState) stopDisplayLink() {
	if metalFrames := s.frameSource.metal; metalFrames != nil {
		if metalFrames.IsAttached() {
			metalFrames.Replace(nil)
			s.renderer.SetDisplayLinkAttached(false)
		}
		return
	}
	if s.frameSource.displayLink != nil {
		s.frameSource.displayLink.Stop()
	}
}

func (s *macWindowState) isMaximized() bool {
	rectToSize := func(rect objc.CGRect) gpui.Size[gpui.Pixels] {
		return gpui.NewSize(gpui.Pixels(rect.Size.Width), gpui.Pixels(rect.Size.Height))
	}

	bounds := s.bounds()
	var screenFrame objc.CGRect
	if screen := s.window().Screen(); screen != 0 {
		screenFrame = screen.VisibleFrame()
	}
	return bounds.Size == rectToSize(screenFrame)
}

func (s *macWindowState) isFullscreen() bool {
	return s.window().StyleMask().Contains(appkit.NSWindowStyleMaskFullScreen)
}

func (s *macWindowState) toggleSimpleFullscreen() (simpleFullscreenPlan, bool) {
	if s.isFullscreen() {
		return simpleFullscreenPlan{}, false
	}

	if saved := s.simpleFullscreenState; saved != nil {
		s.simpleFullscreenState = nil
		return simpleFullscreenPlan{exit: *saved}, true
	}
	screen := s.window().Screen()
	if screen == 0 {
		return simpleFullscreenPlan{}, false
	}
	screenFrame := screen.Frame()
	bounds := s.bounds()

	s.simpleFullscreenState = &simpleFullscreenState{
		frame:     s.window().Frame(),
		bounds:    bounds,
		styleMask: s.window().StyleMask(),
	}

	return simpleFullscreenPlan{enter: true, screenFrame: screenFrame}, true
}

func (s *macWindowState) bounds() gpui.Bounds[gpui.Pixels] {
	windowFrame := s.window().Frame()
	screen := s.window().Screen()
	if screen == 0 {
		return gpui.NewBounds(gpui.NewPoint(gpui.Px(0.), gpui.Px(0.)), gpui.DefaultWindowSize)
	}
	screenFrame := screen.Frame()

	windowFrame.Origin.Y = screenFrame.Size.Height - windowFrame.Origin.Y - windowFrame.Size.Height

	return gpui.NewBounds(
		gpui.NewPoint(
			gpui.Px(float32(windowFrame.Origin.X-screenFrame.Origin.X)),
			gpui.Px(float32(windowFrame.Origin.Y+screenFrame.Origin.Y)),
		),
		gpui.NewSize(
			gpui.Px(float32(windowFrame.Size.Width)),
			gpui.Px(float32(windowFrame.Size.Height)),
		),
	)
}

func (s *macWindowState) contentSize() gpui.Size[gpui.Pixels] {
	var size objc.CGSize
	if view := s.window().ContentView(); view != 0 {
		size = view.Frame().Size
	}
	return gpui.NewSize(gpui.Px(float32(size.Width)), gpui.Px(float32(size.Height)))
}

func (s *macWindowState) scaleFactor() float32 {
	return getScaleFactor(s.nativeWindow)
}

func (s *macWindowState) windowBounds() gpui.WindowBounds {
	if s.isFullscreen() {
		return gpui.WindowBoundsFullscreen(s.fullscreenRestoreBounds)
	} else if state := s.simpleFullscreenState; state != nil {
		return gpui.WindowBoundsWindowed(state.bounds)
	}
	return gpui.WindowBoundsWindowed(s.bounds())
}

// MacWindow is a macOS window. Close is its drop.
type MacWindow struct {
	gpui.PlatformWindowDefaults
	state       *lockedWindowState
	marker      objc.MainThreadMarker
	metalFrames *MetalFrames
	dropped     bool
}

var _ gpui.PlatformWindow = (*MacWindow)(nil)

func detachOnForeground(executor gpui.ForegroundExecutor, f func()) {
	executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		f()
		return struct{}{}, nil
	}).Detach()
}

// MacWindowOpen opens a window with options on the main thread.
func MacWindowOpen(handle gpui.AnyWindowHandle, options gpui.WindowParams, cursorVisible *atomic.Bool, foregroundExecutor gpui.ForegroundExecutor, backgroundExecutor gpui.BackgroundExecutor, rendererContext rendererContext, marker objc.MainThreadMarker) (gpui.PlatformWindow, error) {
	var window *MacWindow
	var err error
	objc.AutoreleasePool(func() {
		window, err = macWindowOpen(handle, options, cursorVisible, foregroundExecutor, backgroundExecutor, rendererContext, marker)
	})
	if err != nil {
		return nil, err
	}
	return window, nil
}

func macWindowOpen(handle gpui.AnyWindowHandle, options gpui.WindowParams, cursorVisible *atomic.Bool, foregroundExecutor gpui.ForegroundExecutor, backgroundExecutor gpui.BackgroundExecutor, rendererContext rendererContext, marker objc.MainThreadMarker) (*MacWindow, error) {
	renderer, err := apple.NewRenderer(rendererContext, false, options.UseDisplayP3)
	if err != nil {
		return nil, err
	}
	nsWindowClass := objc.GetClass("NSWindow")
	allowsAutomaticWindowTabbing := options.TabbingIdentifier.IsSome()
	objc.SendVoid(objc.ID(nsWindowClass), objc.Sel("setAllowsAutomaticWindowTabbing:"), allowsAutomaticWindowTabbing)

	titlebar, hasTitlebar := options.Titlebar.Get()
	var styleMask appkit.NSWindowStyleMask
	if hasTitlebar {
		styleMask = appkit.NSWindowStyleMaskClosable | appkit.NSWindowStyleMaskTitled

		if options.IsResizable {
			styleMask |= appkit.NSWindowStyleMaskResizable
		}

		if options.IsMinimizable {
			styleMask |= appkit.NSWindowStyleMaskMiniaturizable
		}

		if titlebar.AppearsTransparent {
			styleMask |= appkit.NSWindowStyleMaskFullSizeContentView
		}
	} else {
		styleMask = appkit.NSWindowStyleMaskTitled | appkit.NSWindowStyleMaskFullSizeContentView
	}

	var nativeWindow objc.ID
	switch options.Kind.(type) {
	case gpui.WindowKindPopUp, gpui.WindowKindAnchoredPopup:
		styleMask |= appkit.NSWindowStyleMaskNonactivatingPanel
		nativeWindow = objc.Alloc(panelClass())
	case gpui.WindowKindFloating, gpui.WindowKindDialog:
		nativeWindow = objc.Alloc(panelClass())
	default:
		nativeWindow = objc.Alloc(windowClass())
	}

	var display *MacDisplay
	if displayID, ok := options.DisplayID.Get(); ok {
		display, ok = MacDisplayFindByID(displayID)
		if !ok {
			display = MacDisplayPrimary()
		}
	} else {
		display = MacDisplayPrimary()
	}

	var targetScreen appkit.NSScreen
	for _, screen := range appkit.NSScreenScreens() {
		if id, ok := displayIDForScreen(screen.ID()); ok && id == display.displayID {
			targetScreen = screen
		}
	}
	if targetScreen == 0 {
		targetScreen = appkit.NSScreenMainScreen()
	}
	var screenFrame objc.CGRect
	if targetScreen != 0 {
		screenFrame = targetScreen.Frame()
	}

	bounds := options.Bounds
	windowRect := objc.MakeCGRect(
		screenFrame.Origin.X+float64(bounds.Origin.X.AsF32()),
		screenFrame.Origin.Y+float64((display.Bounds().Size.Height-bounds.Origin.Y).AsF32()),
		float64(bounds.Size.Width.AsF32()),
		float64(bounds.Size.Height.AsF32()),
	)

	nativeWindow = objc.Send[objc.ID](
		nativeWindow,
		objc.Sel("initWithContentRect:styleMask:backing:defer:screen:"),
		windowRect,
		styleMask,
		appkit.NSBackingStoreBuffered,
		false,
		targetScreen.ID(),
	)
	if nativeWindow == objc.Nil {
		panic("assertion failed: !native_window.is_null()")
	}
	filenamesType := appkit.Constant("NSFilenamesPboardType")
	draggedTypes := objc.NSArrayFromIDs([]objc.ID{filenamesType})
	objc.SendVoid(nativeWindow, objc.Sel("registerForDraggedTypes:"), draggedTypes)
	objc.Release(draggedTypes)
	objc.SendVoid(nativeWindow, objc.Sel("setReleasedWhenClosed:"), false)

	contentView := objc.Send[objc.ID](nativeWindow, objc.Sel("contentView"))
	nativeView := objc.Alloc(viewClass())
	contentBounds := objc.Send[objc.CGRect](contentView, objc.Sel("bounds"))
	nativeView = objc.Send[objc.ID](nativeView, objc.Sel("initWithFrame:"), contentBounds)
	if nativeView == objc.Nil {
		panic("assertion failed: !native_view.is_null()")
	}

	var metalFrames *MetalFrames
	if apple.MetalFrameSourceIsAvailable() {
		metalFrames = &MetalFrames{}
	}
	var trafficLightPosition gpui.Option[gpui.Point[gpui.Pixels]]
	if hasTitlebar {
		trafficLightPosition = titlebar.TrafficLightPosition
	}
	state := &lockedWindowState{macWindowState: macWindowState{
		handle:                handle,
		foregroundExecutor:    foregroundExecutor,
		backgroundExecutor:    backgroundExecutor,
		nativeWindow:          nativeWindow,
		nativeView:            nativeView,
		backgroundAppearance:  gpui.WindowBackgroundAppearanceOpaque,
		cursorStyle:           gpui.CursorStyleArrow,
		cursorVisible:         cursorVisible,
		frameSource:           frameSource{metal: metalFrames},
		renderer:              renderer,
		trafficLightPosition:  trafficLightPosition,
		nativeTitlebarVisible: true,
		transparentTitlebar:   !hasTitlebar || titlebar.AppearsTransparent,
		appOwnsTitlebarDrag:   options.AppOwnsTitlebarDrag,
		closed:                &atomic.Bool{},
	}}
	window := &MacWindow{state: state, marker: marker, metalFrames: metalFrames}
	gpui.BindPlatformWindowDefaults(&window.PlatformWindowDefaults, window)

	storeWindowState(nativeWindow, state)
	objc.SendVoid(nativeWindow, objc.Sel("setDelegate:"), nativeWindow)
	storeWindowState(nativeView, state)

	if hasTitlebar {
		if title, ok := titlebar.Title.Get(); ok {
			window.SetTitle(string(title))
		}
	}

	objc.SendVoid(nativeWindow, objc.Sel("setMovable:"), options.IsMovable)

	if windowMinSize, ok := options.WindowMinSize.Get(); ok {
		appkit.NSWindow(nativeWindow).SetContentMinSize(objc.MakeCGSize(windowMinSize.Width.ToF64(), windowMinSize.Height.ToF64()))
	}

	if !hasTitlebar || titlebar.AppearsTransparent {
		appkit.NSWindow(nativeWindow).SetTitlebarAppearsTransparent(true)
		appkit.NSWindow(nativeWindow).SetTitleVisibility(appkit.NSWindowTitleHidden)
	}

	view := appkit.NSView(nativeView)
	view.SetAutoresizingMask(appkit.NSViewWidthSizable | appkit.NSViewHeightSizable)
	objc.SendVoid(nativeView, objc.Sel("setWantsBestResolutionOpenGLSurface:"), true)

	view.SetWantsLayer(true)
	view.SetLayerContentsRedrawPolicy(appkit.NSViewLayerContentsRedrawDuringViewResize)

	nativeView = objc.Autorelease(nativeView)
	objc.SendVoid(contentView, objc.Sel("addSubview:"), nativeView)
	appkit.NSWindow(nativeWindow).MakeFirstResponder(nativeView)

	app := appkit.NSApplicationSharedApplication()
	mainWindow := app.MainWindow().ID()
	var sheetParent objc.ID

	switch options.Kind.(type) {
	case gpui.WindowKindPopUp, gpui.WindowKindAnchoredPopup:
		addMouseTrackingArea(nativeView)

		objc.SendVoid(nativeWindow, objc.Sel("setLevel:"), nsPopUpWindowLevel)
		objc.SendVoid(nativeWindow, objc.Sel("setAnimationBehavior:"), nsWindowAnimationBehaviorUtilityWindow)
		appkit.NSWindow(nativeWindow).SetCollectionBehavior(appkit.NSWindowCollectionBehaviorCanJoinAllSpaces | appkit.NSWindowCollectionBehaviorFullScreenAuxiliary)
	case gpui.WindowKindDialog:
		if mainWindow != objc.Nil {
			parent := objc.Send[objc.ID](mainWindow, objc.Sel("attachedSheet"))
			if parent == objc.Nil {
				parent = mainWindow
			}
			objc.SendVoid(parent, objc.Sel("beginSheet:completionHandler:"), nativeWindow, objc.Nil)
			sheetParent = parent
		}
	default:
		if _, floating := options.Kind.(gpui.WindowKindFloating); floating {
			objc.SendVoid(nativeWindow, objc.Sel("setLevel:"), nsFloatingWindowLevel)
			appkit.NSWindow(nativeWindow).SetAcceptsMouseMovedEvents(true)
		} else {
			objc.SendVoid(nativeWindow, objc.Sel("setLevel:"), nsNormalWindowLevel)
			appkit.NSWindow(nativeWindow).SetAcceptsMouseMovedEvents(false)
			addMouseTrackingArea(nativeView)
		}

		setTabbingIdentifier(nativeWindow, options.TabbingIdentifier)
	}

	if allowsAutomaticWindowTabbing && mainWindow != objc.Nil && mainWindow != nativeWindow {
		mainWindowIsFullscreen := appkit.NSWindow(mainWindow).StyleMask().Contains(appkit.NSWindowStyleMaskFullScreen)
		userTabbingPreference, ok := MacWindowGetUserTabbingPreference()
		if !ok {
			userTabbingPreference = UserTabbingPreferenceInFullScreen
		}
		shouldAddAsTab := userTabbingPreference == UserTabbingPreferenceAlways ||
			userTabbingPreference == UserTabbingPreferenceInFullScreen && mainWindowIsFullscreen

		if shouldAddAsTab {
			mainWindowCanTab := objc.RespondsToSelector(mainWindow, objc.Sel("addTabbedWindow:ordered:"))
			mainWindowVisible := objc.Send[bool](mainWindow, objc.Sel("isVisible"))

			if mainWindowCanTab && mainWindowVisible {
				objc.SendVoid(mainWindow, objc.Sel("addTabbedWindow:ordered:"), nativeWindow, appkit.NSWindowAbove)

				if !mainWindowIsFullscreen {
					appkit.NSWindow(nativeWindow).OrderFront(objc.Nil)
				}
			}
		}
	}

	if options.Focus && options.Show {
		appkit.NSWindow(nativeWindow).MakeKeyAndOrderFront(objc.Nil)
	} else if options.Show {
		appkit.NSWindow(nativeWindow).OrderFront(objc.Nil)
	}

	appkit.NSWindow(nativeWindow).SetFrameTopLeftPoint(windowRect.Origin)
	state.Lock()
	state.moveTrafficLight()
	state.sheetParent = sheetParent
	state.Unlock()

	return window, nil
}

const nsWindowAnimationBehaviorUtilityWindow objc.NSInteger = 4

func setTabbingIdentifier(nativeWindow objc.ID, tabbingIdentifier gpui.Option[string]) {
	if identifier, ok := tabbingIdentifier.Get(); ok {
		objc.SendVoid(nativeWindow, objc.Sel("setTabbingIdentifier:"), nsString(identifier))
	} else {
		objc.SendVoid(nativeWindow, objc.Sel("setTabbingIdentifier:"), objc.Nil)
	}
}

// MacWindowActiveWindow returns the handle of the main window when it is a GPUI window.
func MacWindowActiveWindow() (gpui.AnyWindowHandle, bool) {
	app := appkit.NSApplicationSharedApplication()
	mainWindow := app.MainWindow().ID()
	if mainWindow == objc.Nil {
		return gpui.AnyWindowHandle{}, false
	}

	if objc.Send[bool](mainWindow, objc.Sel("isKindOfClass:"), windowClass()) {
		state := getWindowState(mainWindow)
		state.Lock()
		handle := state.handle
		state.Unlock()
		return handle, true
	}
	return gpui.AnyWindowHandle{}, false
}

// MacWindowOrderedWindows returns the handles of the GPUI windows, front to back.
func MacWindowOrderedWindows() []gpui.AnyWindowHandle {
	app := appkit.NSApplicationSharedApplication()
	var windowHandles []gpui.AnyWindowHandle
	for _, window := range app.OrderedWindows() {
		if objc.Send[bool](window.ID(), objc.Sel("isKindOfClass:"), windowClass()) {
			state := getWindowState(window.ID())
			state.Lock()
			windowHandles = append(windowHandles, state.handle)
			state.Unlock()
		}
	}
	return windowHandles
}

// MacWindowGetUserTabbingPreference reads AppleWindowTabbingMode from the global user defaults.
func MacWindowGetUserTabbingPreference() (UserTabbingPreference, bool) {
	switch globalDefaultsString("AppleWindowTabbingMode") {
	case "manual":
		return UserTabbingPreferenceNever, true
	case "always":
		return UserTabbingPreferenceAlways, true
	default:
		return UserTabbingPreferenceInFullScreen, true
	}
}

func globalDefaultsString(name string) string {
	defaults := objc.SendClass[objc.ID](objc.GetClass("NSUserDefaults"), objc.Sel("standardUserDefaults"))
	domain := nsString("NSGlobalDomain")
	key := nsString(name)

	dict := objc.Send[objc.ID](defaults, objc.Sel("persistentDomainForName:"), domain)
	var value objc.ID
	if dict != objc.Nil {
		value = objc.Send[objc.ID](dict, objc.Sel("objectForKey:"), key)
	}
	if value == objc.Nil {
		return ""
	}
	return nsStringToStr(value)
}

// Close releases the window: it drops the accessibility adapter, destroys the renderer, detaches
// the frame source and the delegate, and closes the native window on the next main-loop turn. It
// is idempotent.
func (w *MacWindow) Close() {
	if w.dropped {
		return
	}
	w.dropped = true
	this := w.state
	this.Lock()
	defer this.Unlock()
	if adapter := this.a11yAdapter; adapter != nil {
		this.a11yAdapter = nil
		adapter.Close()
	}
	this.renderer.Destroy()
	window := this.nativeWindow
	sheetParent := this.sheetParent
	this.sheetParent = objc.Nil
	if this.frameSource.metal != nil {
		this.frameSource.metal.Replace(nil)
	} else if frameSource := this.frameSource.displayLink; frameSource != nil {
		this.frameSource.displayLink = nil
		frameSource.Close()
	}
	objc.SendVoid(this.nativeWindow, objc.Sel("setDelegate:"), objc.Nil)
	this.inputHandler = nil
	this.visibilityCallback = nil
	detachOnForeground(this.foregroundExecutor, func() {
		if sheetParent != objc.Nil {
			objc.SendVoid(sheetParent, objc.Sel("endSheet:"), window)
		}
		objc.SendVoid(window, objc.Sel("close"))
		objc.Autorelease(window)
	})
}

func ifWindowNotClosed(closed *atomic.Bool, f func()) {
	if !closed.Load() {
		f()
	}
}

// RawWindowHandle returns the AppKit handle of the window's view.
func (w *MacWindow) RawWindowHandle() gpui.RawWindowHandle {
	w.state.Lock()
	defer w.state.Unlock()
	return gpui.RawWindowHandle{Kind: gpui.RawWindowHandleAppKit, AppKitView: uintptr(w.state.nativeView)}
}

// Bounds returns the window bounds in screen coordinates with a top-left origin.
func (w *MacWindow) Bounds() gpui.Bounds[gpui.Pixels] {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.bounds()
}

// WindowBounds returns the bounds to restore the window with.
func (w *MacWindow) WindowBounds() gpui.WindowBounds {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.windowBounds()
}

// IsMaximized reports whether the window fills its screen's visible frame.
func (w *MacWindow) IsMaximized() bool {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.isMaximized()
}

// ContentSize returns the size of the window's content view.
func (w *MacWindow) ContentSize() gpui.Size[gpui.Pixels] {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.contentSize()
}

// Resize sets the content size on the next main-loop turn.
func (w *MacWindow) Resize(size gpui.Size[gpui.Pixels]) {
	w.state.Lock()
	defer w.state.Unlock()
	window := w.state.nativeWindow
	closed := w.state.closed
	detachOnForeground(w.state.foregroundExecutor, func() {
		ifWindowNotClosed(closed, func() {
			appkit.NSWindow(window).SetContentSize(objc.MakeCGSize(float64(size.Width.AsF32()), float64(size.Height.AsF32())))
		})
	})
}

// MergeAllWindows merges all windows into this window's tab group.
func (w *MacWindow) MergeAllWindows() {
	w.state.Lock()
	nativeWindow := w.state.nativeWindow
	w.state.Unlock()
	gcd.MainQueue().Async(func() {
		objc.SendVoid(nativeWindow, objc.Sel("mergeAllWindows:"), objc.Nil)
	})
}

// MoveTabToNewWindow moves this tab into a window of its own.
func (w *MacWindow) MoveTabToNewWindow() {
	w.state.Lock()
	nativeWindow := w.state.nativeWindow
	w.state.Unlock()
	gcd.MainQueue().Async(func() {
		objc.SendVoid(nativeWindow, objc.Sel("moveTabToNewWindow:"), objc.Nil)
		appkit.NSWindow(nativeWindow).MakeKeyAndOrderFront(objc.Nil)
	})
}

// ToggleWindowTabOverview toggles the tab overview.
func (w *MacWindow) ToggleWindowTabOverview() {
	w.state.Lock()
	nativeWindow := w.state.nativeWindow
	w.state.Unlock()
	objc.SendVoid(nativeWindow, objc.Sel("toggleTabOverview:"), objc.Nil)
}

// SetTabbingIdentifier sets the window's tabbing identifier and enables automatic tabbing when set.
func (w *MacWindow) SetTabbingIdentifier(tabbingIdentifier gpui.Option[string]) {
	w.state.Lock()
	nativeWindow := w.state.nativeWindow
	w.state.Unlock()
	objc.SendVoid(objc.ID(objc.GetClass("NSWindow")), objc.Sel("setAllowsAutomaticWindowTabbing:"), tabbingIdentifier.IsSome())
	setTabbingIdentifier(nativeWindow, tabbingIdentifier)
}

// NativeWindowState archives the window's restorable state, or reports false in full screen.
func (w *MacWindow) NativeWindowState() ([]byte, bool) {
	w.state.Lock()
	if w.state.isFullscreen() || w.state.simpleFullscreenState != nil {
		w.state.Unlock()
		return nil, false
	}
	nativeWindow := w.state.nativeWindow
	w.state.Unlock()

	archiver := objc.Alloc(objc.GetClass("NSKeyedArchiver"))
	archiver = objc.Send[objc.ID](archiver, objc.Sel("initRequiringSecureCoding:"), true)
	if archiver == objc.Nil {
		slog.Warn("failed to create an archiver for the native window state")
		return nil, false
	}
	delegate := objc.SendClass[objc.ID](windowStateArchiverDelegateClass(), objc.Sel("new"))
	objc.SendVoid(archiver, objc.Sel("setDelegate:"), delegate)
	objc.SendVoid(nativeWindow, objc.Sel("encodeRestorableStateWithCoder:"), archiver)
	objc.SendVoid(archiver, objc.Sel("finishEncoding"))
	objc.SendVoid(archiver, objc.Sel("setDelegate:"), objc.Nil)

	data := objc.Send[objc.ID](archiver, objc.Sel("encodedData"))
	var p unsafe.Pointer
	if data != objc.Nil {
		p = objc.Send[unsafe.Pointer](data, objc.Sel("bytes"))
	}
	var state []byte
	ok := false
	if p == nil {
		slog.Warn("the archiver produced no data for the native window state")
	} else {
		length := objc.Send[objc.NSUInteger](data, objc.Sel("length"))
		state = bytes.Clone(unsafe.Slice((*byte)(p), length))
		ok = true
	}

	objc.Release(delegate)
	objc.Release(archiver)
	return state, ok
}

// RestoreNativeWindowState restores state produced by NativeWindowState.
func (w *MacWindow) RestoreNativeWindowState(state []byte) {
	if len(state) == 0 {
		return
	}
	w.state.Lock()
	nativeWindow := w.state.nativeWindow
	w.state.Unlock()
	data := objc.SendClass[objc.ID](objc.GetClass("NSData"), objc.Sel("dataWithBytes:length:"), unsafe.Pointer(&state[0]), objc.NSUInteger(len(state)))
	if data == objc.Nil {
		slog.Warn(fmt.Sprintf("failed to wrap %d bytes of native window state", len(state)))
		return
	}

	if !isMacosVersionAtLeast(nsOperatingSystemVersion{MajorVersion: 15, MinorVersion: 0, PatchVersion: 0}) {
		restoresWorkspaceAtLaunchDefault.Do(func() {
			defaults := objc.SendClass[objc.ID](objc.GetClass("NSUserDefaults"), objc.Sel("standardUserDefaults"))
			key := nsString("NSWindowRestoresWorkspaceAtLaunch")
			yesValue := objc.SendClass[objc.ID](objc.GetClass("NSNumber"), objc.Sel("numberWithBool:"), true)
			dict := objc.SendClass[objc.ID](objc.GetClass("NSDictionary"), objc.Sel("dictionaryWithObject:forKey:"), yesValue, key)
			objc.SendVoid(defaults, objc.Sel("registerDefaults:"), dict)
		})
	}

	unarchiver := objc.Alloc(windowStateUnarchiverClass())
	var errID objc.ID
	unarchiver = objc.Send[objc.ID](unarchiver, objc.Sel("initForReadingFromData:error:"), data, unsafe.Pointer(&errID))
	if unarchiver == objc.Nil {
		slog.Warn("failed to unarchive the native window state: " + nsErrorDescription(errID))
		return
	}
	objc.SendVoid(nativeWindow, objc.Sel("restoreStateWithCoder:"), unarchiver)
	if errID := objc.Send[objc.ID](unarchiver, objc.Sel("error")); errID != objc.Nil {
		slog.Warn("failed to restore the native window state: " + nsErrorDescription(errID))
	}
	objc.Release(unarchiver)
}

// SetNativeTitlebarVisibility shows or hides the native titlebar container.
func (w *MacWindow) SetNativeTitlebarVisibility(visible bool) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.nativeTitlebarVisible = visible
	w.state.updateNativeTitlebarVisibility()
}

// SetTrafficLightPosition moves the traffic light buttons.
func (w *MacWindow) SetTrafficLightPosition(position gpui.Point[gpui.Pixels]) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.trafficLightPosition = gpui.Some(position)
	w.state.moveTrafficLight()
}

// ScaleFactor returns the backing scale factor of the window's screen.
func (w *MacWindow) ScaleFactor() float32 {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.scaleFactor()
}

// Appearance returns the window's effective appearance.
func (w *MacWindow) Appearance() gpui.WindowAppearance {
	w.state.Lock()
	nativeWindow := w.state.nativeWindow
	w.state.Unlock()
	return windowAppearanceFromNative(appkit.NSWindow(nativeWindow).EffectiveAppearance())
}

// Display returns the display the window is on.
func (w *MacWindow) Display() gpui.PlatformDisplay {
	w.state.Lock()
	screen := objc.Send[objc.ID](w.state.nativeWindow, objc.Sel("screen"))
	w.state.Unlock()
	displayID, ok := displayIDForScreen(screen)
	if !ok {
		return nil
	}
	return newMacDisplay(displayID)
}

// MousePosition returns the mouse position in window coordinates.
func (w *MacWindow) MousePosition() gpui.Point[gpui.Pixels] {
	w.state.Lock()
	position := w.state.window().MouseLocationOutsideOfEventStream()
	w.state.Unlock()
	return convertMousePosition(position, w.ContentSize().Height)
}

// Modifiers returns the current modifier keys.
func (w *MacWindow) Modifiers() gpui.Modifiers {
	return currentModifiers()
}

// Capslock returns the current capslock state.
func (w *MacWindow) Capslock() gpui.Capslock {
	return currentCapslock()
}

// SetInputHandler sets the input handler that receives text input.
func (w *MacWindow) SetInputHandler(inputHandler *gpui.PlatformInputHandler) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.inputHandler = inputHandler
}

// TakeInputHandler removes and returns the input handler.
func (w *MacWindow) TakeInputHandler() *gpui.PlatformInputHandler {
	w.state.Lock()
	defer w.state.Unlock()
	inputHandler := w.state.inputHandler
	w.state.inputHandler = nil
	return inputHandler
}

// Prompt shows an NSAlert sheet and returns a receiver for the index of the chosen answer.
func (w *MacWindow) Prompt(level gpui.PromptLevel, msg string, detail gpui.Option[string], answers []gpui.PromptButton) *scheduler.OneshotReceiver[int] {
	initialFocusIx := -1
	for ix := len(answers) - 1; ix >= 0; ix-- {
		if !answers[ix].IsCancel() {
			if ix > 0 {
				initialFocusIx = ix
			}
			break
		}
	}

	alert := objc.AllocInit(objc.GetClass("NSAlert"))
	var style objc.NSUInteger
	switch level {
	case gpui.PromptLevelCritical:
		style = nsAlertStyleCritical
	case gpui.PromptLevelWarning:
		style = nsAlertStyleWarning
	default:
		style = nsAlertStyleInformational
	}
	objc.SendVoid(alert, objc.Sel("setAlertStyle:"), style)
	message := objc.NSStringFromGo(msg)
	objc.SendVoid(alert, objc.Sel("setMessageText:"), message)
	objc.Release(message)

	if detail, ok := detail.Get(); ok {
		detailText := objc.NSStringFromGo(detail)
		objc.SendVoid(alert, objc.Sel("setInformativeText:"), detailText)
		objc.Release(detailText)
	}

	var initialFocusButton objc.ID
	for ix, answer := range answers {
		title := objc.NSStringFromGo(string(answer.Label()))
		button := objc.Send[objc.ID](alert, objc.Sel("addButtonWithTitle:"), title)
		objc.Release(title)
		objc.SendVoid(button, objc.Sel("setTag:"), objc.NSInteger(ix))

		if answer.IsCancel() {
			key := objc.NSStringFromGo(string(rune(escapeKey)))
			objc.SendVoid(button, objc.Sel("setKeyEquivalent:"), key)
			objc.Release(key)
		} else if ix == initialFocusIx {
			initialFocusButton = button
		}
	}

	if initialFocusButton != objc.Nil {
		alertWindow := objc.Send[objc.ID](alert, objc.Sel("window"))
		objc.SendVoid(alertWindow, objc.Sel("setInitialFirstResponder:"), initialFocusButton)
	}

	doneTx, doneRx := scheduler.Oneshot[int]()
	var sent atomic.Bool

	block := objc.NewBlock(func(_ objc.Block, answer objc.NSInteger) {
		if !sent.Swap(true) {
			if answer < 0 {
				panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: TryFromIntError(()) for %d", answer))
			}
			_ = doneTx.Send(int(answer))
		}
	})

	w.state.Lock()
	nativeWindow := w.state.nativeWindow
	closed := w.state.closed
	executor := w.state.foregroundExecutor
	w.state.Unlock()
	detachOnForeground(executor, func() {
		if !closed.Load() {
			objc.SendVoid(alert, objc.Sel("beginSheetModalForWindow:completionHandler:"), nativeWindow, block)
		}
		objc.Release(alert)
		block.Release()
	})

	return doneRx
}

const (
	nsAlertStyleWarning       objc.NSUInteger = 0
	nsAlertStyleInformational objc.NSUInteger = 1
	nsAlertStyleCritical      objc.NSUInteger = 2
)

// Activate makes the window key and orders it front on the next main-loop turn.
func (w *MacWindow) Activate() {
	w.state.Lock()
	window := w.state.nativeWindow
	view := w.state.nativeView
	closed := w.state.closed
	executor := w.state.foregroundExecutor
	w.state.Unlock()
	detachOnForeground(executor, func() {
		if !closed.Load() {
			nsWindow := appkit.NSWindow(window)
			if !nsWindow.IsVisible() {
				appkit.NSView(view).SetNeedsDisplay(true)
			}
			nsWindow.MakeKeyAndOrderFront(objc.Nil)
		}
	})
}

// RequestAttention bounces the dock icon unless the window is active.
func (w *MacWindow) RequestAttention() {
	if w.IsActive() {
		return
	}

	w.state.Lock()
	executor := w.state.foregroundExecutor
	w.state.Unlock()
	detachOnForeground(executor, func() {
		appkit.NSApplicationSharedApplication().RequestUserAttention(appkit.NSInformationalRequest)
	})
}

// IsActive reports whether the window is the key window.
func (w *MacWindow) IsActive() bool {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.window().IsKeyWindow()
}

// Visibility reports whether the window is visible on screen.
func (w *MacWindow) Visibility() gpui.WindowVisibility {
	w.state.Lock()
	defer w.state.Unlock()
	return visibility(&w.state.macWindowState)
}

// IsHovered is unused on macOS; see Window.IsWindowHovered.
func (w *MacWindow) IsHovered() bool {
	return false
}

// SetTitle sets the window title and its Window menu entry.
func (w *MacWindow) SetTitle(title string) {
	app := appkit.NSApplicationSharedApplication().ID()
	w.state.Lock()
	window := w.state.nativeWindow
	w.state.Unlock()
	nsTitle := nsString(title)
	objc.SendVoid(app, objc.Sel("changeWindowsItem:title:filename:"), window, nsTitle, false)
	objc.SendVoid(window, objc.Sel("setTitle:"), nsTitle)
	w.state.Lock()
	w.state.moveTrafficLight()
	w.state.Unlock()
}

// GetTitle returns the window title.
func (w *MacWindow) GetTitle() string {
	w.state.Lock()
	title := objc.Send[objc.ID](w.state.nativeWindow, objc.Sel("title"))
	w.state.Unlock()
	if title == objc.Nil {
		return ""
	}
	return nsStringToStr(title)
}

// SetAppID does nothing on macOS.
func (w *MacWindow) SetAppID(appID string) {}

// SetBackgroundAppearance sets the window's opacity and adds or removes the blur view.
func (w *MacWindow) SetBackgroundAppearance(backgroundAppearance gpui.WindowBackgroundAppearance) {
	this := w.state
	this.Lock()
	defer this.Unlock()
	this.backgroundAppearance = backgroundAppearance

	opaque := backgroundAppearance == gpui.WindowBackgroundAppearanceOpaque
	this.renderer.UpdateTransparency(!opaque)

	objc.SendVoid(this.nativeWindow, objc.Sel("setOpaque:"), opaque)
	var backgroundColor appkit.NSColor
	if opaque {
		backgroundColor = appkit.NSColorWithSRGB(0, 0, 0, 1)
	} else {
		backgroundColor = appkit.NSColorWithSRGB(0, 0, 0, 0.0001)
	}
	this.window().SetBackgroundColor(backgroundColor)
	this.nativeViews.background = gpui.None[gpui.Rgba]()

	if backgroundAppearance != gpui.WindowBackgroundAppearanceBlurred {
		if blurView := this.blurredView; blurView != objc.Nil {
			objc.SendVoid(blurView, objc.Sel("removeFromSuperview"))
			this.blurredView = objc.Nil
		}
	} else if this.blurredView == objc.Nil {
		contentView := objc.Send[objc.ID](this.nativeWindow, objc.Sel("contentView"))
		frame := objc.Send[objc.CGRect](contentView, objc.Sel("bounds"))
		blurView := objc.Alloc(blurredViewClass())
		blurView = objc.Send[objc.ID](blurView, objc.Sel("initWithFrame:"), frame)
		appkit.NSView(blurView).SetAutoresizingMask(appkit.NSViewWidthSizable | appkit.NSViewHeightSizable)

		objc.SendVoid(contentView, objc.Sel("addSubview:positioned:relativeTo:"), blurView, appkit.NSWindowBelow, objc.Nil)
		this.blurredView = objc.Autorelease(blurView)
	}
}

// BackgroundAppearance returns the window's background appearance.
func (w *MacWindow) BackgroundAppearance() gpui.WindowBackgroundAppearance {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.backgroundAppearance
}

// IsSubpixelRenderingSupported reports false: the Metal renderer draws grayscale text.
func (w *MacWindow) IsSubpixelRenderingSupported() bool {
	return false
}

// SetEdited sets the window's document-edited dot.
func (w *MacWindow) SetEdited(edited bool) {
	w.state.Lock()
	window := w.state.nativeWindow
	w.state.Unlock()
	objc.SendVoid(window, objc.Sel("setDocumentEdited:"), edited)

	w.state.Lock()
	w.state.moveTrafficLight()
	w.state.Unlock()
}

// SetDocumentPath sets the window's represented file.
func (w *MacWindow) SetDocumentPath(path gpui.Option[string]) {
	w.state.Lock()
	window := w.state.nativeWindow
	w.state.Unlock()
	filename := nsString("")
	if p, ok := path.Get(); ok {
		filename = nsString(p)
	}
	objc.SendVoid(window, objc.Sel("setRepresentedFilename:"), filename)

	w.state.Lock()
	w.state.moveTrafficLight()
	w.state.Unlock()
}

// ShowCharacterPalette opens the character palette on the next main-loop turn.
func (w *MacWindow) ShowCharacterPalette() {
	w.state.Lock()
	defer w.state.Unlock()
	window := w.state.nativeWindow
	detachOnForeground(w.state.foregroundExecutor, func() {
		app := appkit.NSApplicationSharedApplication().ID()
		objc.SendVoid(app, objc.Sel("orderFrontCharacterPalette:"), window)
	})
}

// Minimize miniaturizes the window.
func (w *MacWindow) Minimize() {
	w.state.Lock()
	window := w.state.nativeWindow
	w.state.Unlock()
	appkit.NSWindow(window).Miniaturize(objc.Nil)
}

// Zoom zooms the window on the next main-loop turn.
func (w *MacWindow) Zoom() {
	w.state.Lock()
	defer w.state.Unlock()
	window := w.state.nativeWindow
	closed := w.state.closed
	detachOnForeground(w.state.foregroundExecutor, func() {
		ifWindowNotClosed(closed, func() {
			appkit.NSWindow(window).Zoom(objc.Nil)
		})
	})
}

// ToggleFullscreen toggles native full screen on the next main-loop turn.
func (w *MacWindow) ToggleFullscreen() {
	w.state.Lock()
	defer w.state.Unlock()
	window := w.state.nativeWindow
	closed := w.state.closed
	detachOnForeground(w.state.foregroundExecutor, func() {
		ifWindowNotClosed(closed, func() {
			appkit.NSWindow(window).ToggleFullScreen(objc.Nil)
		})
	})
}

// ToggleSimpleFullscreen toggles borderless full screen on the next main-loop turn.
func (w *MacWindow) ToggleSimpleFullscreen() {
	state := w.state
	state.Lock()
	foregroundExecutor := state.foregroundExecutor
	closed := state.closed
	state.Unlock()
	detachOnForeground(foregroundExecutor, func() {
		ifWindowNotClosed(closed, func() {
			state.Lock()
			nativeWindow := state.nativeWindow
			nativeView := state.nativeView
			plan, ok := state.toggleSimpleFullscreen()
			state.Unlock()
			if ok {
				applySimpleFullscreenPlan(nativeWindow, nativeView, plan)
			}
		})
	})
}

// IsSimpleFullscreen reports whether the window is in simple full screen.
func (w *MacWindow) IsSimpleFullscreen() bool {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.simpleFullscreenState != nil
}

// IsFullscreen reports whether the window is in native full screen.
func (w *MacWindow) IsFullscreen() bool {
	w.state.Lock()
	defer w.state.Unlock()
	return appkit.NSWindow(w.state.nativeWindow).StyleMask().Contains(appkit.NSWindowStyleMaskFullScreen)
}

// OnRequestFrame sets the frame callback.
func (w *MacWindow) OnRequestFrame(callback func(options gpui.RequestFrameOptions)) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.requestFrameCallback = callback
}

// OnInput sets the input callback.
func (w *MacWindow) OnInput(callback func(input gpui.PlatformInput) gpui.DispatchEventResult) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.eventCallback = callback
}

// OnActiveStatusChange sets the activation callback.
func (w *MacWindow) OnActiveStatusChange(callback func(active bool)) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.activateCallback = callback
}

// OnVisibilityChange sets the visibility callback, recording the current visibility.
func (w *MacWindow) OnVisibilityChange(callback func(visibility gpui.WindowVisibility)) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.lastVisibility = gpui.Some(visibility(&w.state.macWindowState))
	w.state.visibilityCallback = callback
}

// OnHoverStatusChange does nothing on macOS.
func (w *MacWindow) OnHoverStatusChange(callback func(hovered bool)) {}

// OnResize sets the resize callback.
func (w *MacWindow) OnResize(callback func(size gpui.Size[gpui.Pixels], scaleFactor float32)) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.resizeCallback = callback
}

// OnDisplayChanged sets the display-change callback.
func (w *MacWindow) OnDisplayChanged(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.displayChangedCallback = callback
}

// OnMoved sets the move callback.
func (w *MacWindow) OnMoved(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.movedCallback = callback
}

// OnShouldClose sets the should-close callback.
func (w *MacWindow) OnShouldClose(callback func() bool) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.shouldCloseCallback = callback
}

// OnClose sets the close callback.
func (w *MacWindow) OnClose(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.closeCallback = callback
}

// OnHitTestWindowControl does nothing on macOS.
func (w *MacWindow) OnHitTestWindowControl(callback func() (gpui.WindowControlArea, bool)) {}

// OnAppearanceChanged sets the appearance callback.
func (w *MacWindow) OnAppearanceChanged(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.appearanceChangedCallback = callback
}

// TabbedWindows returns the GPUI windows of this window's tab group.
func (w *MacWindow) TabbedWindows() ([]gpui.SystemWindowTab, bool) {
	w.state.Lock()
	windows := objc.Send[objc.ID](w.state.nativeWindow, objc.Sel("tabbedWindows"))
	w.state.Unlock()
	if windows == objc.Nil {
		return nil, false
	}

	var result []gpui.SystemWindowTab
	for _, window := range objc.NSArrayToIDs(windows) {
		if objc.Send[bool](window, objc.Sel("isKindOfClass:"), windowClass()) {
			state := getWindowState(window)
			state.Lock()
			handle := state.handle
			state.Unlock()
			title := objc.Send[objc.ID](window, objc.Sel("title"))
			result = append(result, gpui.NewSystemWindowTab(gpui.NewSharedString(nsStringToStr(title)), handle))
		}
	}

	return result, true
}

// TabBarVisible reports whether the window's tab bar is visible.
func (w *MacWindow) TabBarVisible() bool {
	w.state.Lock()
	tabGroup := objc.Send[objc.ID](w.state.nativeWindow, objc.Sel("tabGroup"))
	w.state.Unlock()
	if tabGroup == objc.Nil {
		return false
	}
	return objc.Send[bool](tabGroup, objc.Sel("isTabBarVisible"))
}

// OnMoveTabToNewWindow sets the move-tab callback.
func (w *MacWindow) OnMoveTabToNewWindow(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.moveTabToNewWindowCallback = callback
}

// OnMergeAllWindows sets the merge-windows callback.
func (w *MacWindow) OnMergeAllWindows(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.mergeAllWindowsCallback = callback
}

// OnSelectNextTab sets the next-tab callback.
func (w *MacWindow) OnSelectNextTab(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.selectNextTabCallback = callback
}

// OnSelectPreviousTab sets the previous-tab callback.
func (w *MacWindow) OnSelectPreviousTab(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.selectPreviousTabCallback = callback
}

// OnToggleTabBar sets the toggle-tab-bar callback.
func (w *MacWindow) OnToggleTabBar(callback func()) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.toggleTabBarCallback = callback
}

// Draw renders scene and lays out the attached native views.
func (w *MacWindow) Draw(scene *gpui.Scene) {
	this := w.state
	this.Lock()
	nativeViewLayout := nativeViewsPlanLayout(&this.macWindowState, scene)
	nativeViewsUpdateBackground(&this.macWindowState, scene)
	nativeViewsSceneDrawn(&this.macWindowState, scene)
	this.renderer.Draw(scene)
	this.Unlock()
	nativeViewsApplyLayout(w.state, nativeViewLayout)
}

// AttachNativeView adds view, an NSView without a superview, beneath the window's content.
func (w *MacWindow) AttachNativeView(id gpui.NativeViewID, view uintptr) gpui.NativeViewAttachment {
	return nativeViewsAttach(w.state, id, view)
}

// SetFocusedNativeView gives keyboard input to the native view id, or to GPUI when none.
func (w *MacWindow) SetFocusedNativeView(id gpui.Option[gpui.NativeViewID]) {
	nativeViewsSetFocused(w.state, id)
}

// OnHitTestNativeView sets the native-view hit test callback.
func (w *MacWindow) OnHitTestNativeView(callback func(position gpui.Point[gpui.Pixels]) (gpui.NativeViewID, bool)) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.nativeViews.hitTestCallback = callback
}

// OnNativeViewFocused sets the native-view focus callback.
func (w *MacWindow) OnNativeViewFocused(callback func(id gpui.NativeViewID)) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.nativeViews.focusedCallback = callback
}

// OnKeyDownBeforeNativeView sets the callback offered key-downs before a native view.
func (w *MacWindow) OnKeyDownBeforeNativeView(callback func(event gpui.KeyDownEvent) bool) {
	w.state.Lock()
	defer w.state.Unlock()
	w.state.nativeViews.keyDownBeforeCallback = callback
}

// ScheduleFrame resumes a paused Metal display link.
func (w *MacWindow) ScheduleFrame() {
	if w.metalFrames != nil {
		w.metalFrames.Wake()
	}
}

// FrameWaker returns a function that wakes the Metal display link, or nil under a CVDisplayLink.
func (w *MacWindow) FrameWaker() func() {
	if w.metalFrames == nil {
		return nil
	}
	metalFrames := weak.Make(w.metalFrames)
	return func() {
		if metalFrames := metalFrames.Value(); metalFrames != nil {
			metalFrames.Wake()
		}
	}
}

// SpriteAtlas returns the renderer's sprite atlas.
func (w *MacWindow) SpriteAtlas() gpui.PlatformAtlas {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.renderer.SpriteAtlas()
}

// GPUSpecs reports no GPU specs on macOS.
func (w *MacWindow) GPUSpecs() (gpui.GPUSpecs, bool) {
	return gpui.GPUSpecs{}, false
}

type macWindowDebugCursor struct {
	CachedStyle   gpui.CursorStyle `json:"cached_style"`
	CachedVisible bool             `json:"cached_visible"`
}

type macWindowDebugLayer struct {
	ContentsScale float64    `json:"contents_scale"`
	DrawableSize  [2]float64 `json:"drawable_size"`
}

type macWindowDebugRenderer struct {
	DisplayLinkAttached     bool                 `json:"display_link_attached"`
	Layer                   *macWindowDebugLayer `json:"layer"`
	PresentsWithTransaction bool                 `json:"presents_with_transaction"`
}

type macWindowDebugState struct {
	ActivatedAtLeastOnce          bool                   `json:"activated_at_least_once"`
	BackingScaleFactor            float64                `json:"backing_scale_factor"`
	Cursor                        macWindowDebugCursor   `json:"cursor"`
	Frame                         [4]float64             `json:"frame"`
	FrameSource                   jsontext.Value         `json:"frame_source"`
	IsKeyWindow                   bool                   `json:"is_key_window"`
	IsMainWindow                  bool                   `json:"is_main_window"`
	IsMiniaturized                bool                   `json:"is_miniaturized"`
	IsOnActiveSpace               bool                   `json:"is_on_active_space"`
	IsVisible                     bool                   `json:"is_visible"`
	LastReportedVisibility        *string                `json:"last_reported_visibility"`
	OcclusionStateVisible         bool                   `json:"occlusion_state_visible"`
	Renderer                      macWindowDebugRenderer `json:"renderer"`
	RequestFrameCallbackInstalled bool                   `json:"request_frame_callback_installed"`
	ScreenDisplayID               *uint32                `json:"screen_display_id"`
	WindowNumber                  int                    `json:"window_number"`
}

func windowVisibilityDebug(v gpui.WindowVisibility) string {
	if v == gpui.WindowVisibilityVisible {
		return "Visible"
	}
	return "Hidden"
}

// DebugState returns the window's AppKit, frame source, renderer and cursor state.
func (w *MacWindow) DebugState() jsontext.Value {
	state := w.state
	state.Lock()
	defer state.Unlock()
	window := state.window()
	screen := objc.Send[objc.ID](state.nativeWindow, objc.Sel("screen"))
	frame := window.Frame()
	var layer *macWindowDebugLayer
	if metalLayer, ok := state.renderer.Layer(); ok {
		drawableSize := metalLayer.DrawableSize()
		layer = &macWindowDebugLayer{
			ContentsScale: objc.Send[objc.CGFloat](metalLayer.ID(), objc.Sel("contentsScale")),
			DrawableSize:  [2]float64{drawableSize.Width, drawableSize.Height},
		}
	}
	var frameSourceState jsontext.Value
	switch {
	case state.frameSource.metal != nil:
		frameSourceState = state.frameSource.metal.DebugState()
	case state.frameSource.displayLink != nil:
		frameSourceState = state.frameSource.displayLink.DebugState()
	default:
		frameSourceState = mustMarshalDebugState(struct {
			Attached bool   `json:"attached"`
			Kind     string `json:"kind"`
		}{false, "CVDisplayLink"})
	}
	var screenDisplayID *uint32
	if id, ok := displayIDForScreen(screen); ok {
		screenDisplayID = &id
	}
	var lastReportedVisibility *string
	if v, ok := state.lastVisibility.Get(); ok {
		s := windowVisibilityDebug(v)
		lastReportedVisibility = &s
	}
	debugState := macWindowDebugState{
		ActivatedAtLeastOnce: state.activatedLeastOnce,
		BackingScaleFactor:   window.BackingScaleFactor(),
		Cursor: macWindowDebugCursor{
			CachedStyle:   state.cursorStyle,
			CachedVisible: state.cursorVisible.Load(),
		},
		Frame:                  [4]float64{frame.Origin.X, frame.Origin.Y, frame.Size.Width, frame.Size.Height},
		FrameSource:            frameSourceState,
		IsKeyWindow:            window.IsKeyWindow(),
		IsMainWindow:           objc.Send[bool](state.nativeWindow, objc.Sel("isMainWindow")),
		IsMiniaturized:         window.IsMiniaturized(),
		IsOnActiveSpace:        objc.Send[bool](state.nativeWindow, objc.Sel("isOnActiveSpace")),
		IsVisible:              window.IsVisible(),
		LastReportedVisibility: lastReportedVisibility,
		OcclusionStateVisible:  window.OcclusionState()&appkit.NSWindowOcclusionStateVisible != 0,
		Renderer: macWindowDebugRenderer{
			DisplayLinkAttached:     state.renderer.IsDisplayLinkAttached(),
			Layer:                   layer,
			PresentsWithTransaction: state.renderer.PresentsWithTransaction(),
		},
		RequestFrameCallbackInstalled: state.requestFrameCallback != nil,
		ScreenDisplayID:               screenDisplayID,
		WindowNumber:                  window.WindowNumber(),
	}
	value, err := json.Marshal(debugState)
	if err != nil {
		panic(err)
	}
	return jsontext.Value(value)
}

// UpdateIMEPosition invalidates the input context's character coordinates on the next main-loop turn.
func (w *MacWindow) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {
	w.state.Lock()
	executor := w.state.foregroundExecutor
	w.state.Unlock()
	detachOnForeground(executor, func() {
		inputContext := objc.SendClass[objc.ID](objc.GetClass("NSTextInputContext"), objc.Sel("currentInputContext"))
		if inputContext == objc.Nil {
			return
		}
		objc.SendVoid(inputContext, objc.Sel("invalidateCharacterCoordinates"))
	})
}

// TitlebarDoubleClick performs the system's AppleActionOnDoubleClick action.
func (w *MacWindow) TitlebarDoubleClick(isResizable bool, isMinimizable bool) {
	w.state.Lock()
	defer w.state.Unlock()
	if w.state.simpleFullscreenState != nil {
		return
	}
	window := w.state.nativeWindow
	closed := w.state.closed
	detachOnForeground(w.state.foregroundExecutor, func() {
		ifWindowNotClosed(closed, func() {
			switch globalDefaultsString("AppleActionOnDoubleClick") {
			case "None":
			case "Minimize":
				if isMinimizable {
					objc.SendVoid(window, objc.Sel("miniaturize:"), objc.Nil)
				}
			case "Maximize":
				if isResizable {
					objc.SendVoid(window, objc.Sel("zoom:"), objc.Nil)
				}
			case "Fill":
				if isResizable {
					if objc.RespondsToSelector(window, objc.Sel("_zoomFill:")) {
						objc.SendVoid(window, objc.Sel("_zoomFill:"), objc.Nil)
					} else {
						objc.SendVoid(window, objc.Sel("zoom:"), objc.Nil)
					}
				}
			default:
				if isResizable {
					objc.SendVoid(window, objc.Sel("zoom:"), objc.Nil)
				}
			}
		})
	})
}

// StartWindowMove drags the window with the current event.
func (w *MacWindow) StartWindowMove() {
	w.state.Lock()
	defer w.state.Unlock()
	if w.state.simpleFullscreenState != nil {
		return
	}
	window := w.state.nativeWindow

	event := appkit.NSApplicationSharedApplication().CurrentEvent()
	objc.SendVoid(window, objc.Sel("performWindowDragWithEvent:"), event.ID())
}

// CanStartExternalDrag reports true: macOS supports outbound file drags.
func (w *MacWindow) CanStartExternalDrag() bool {
	return true
}

var (
	uniformTypeIdentifiersOnce sync.Once
	uniformTypeIdentifiers     uintptr
	uniformTypeIdentifiersErr  error
)

const uniformTypeIdentifiersPath = "/System/Library/Frameworks/UniformTypeIdentifiers.framework/UniformTypeIdentifiers"

func utTypeConstant(name string) objc.ID {
	uniformTypeIdentifiersOnce.Do(func() {
		uniformTypeIdentifiers, uniformTypeIdentifiersErr = purego.Dlopen(uniformTypeIdentifiersPath, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	})
	if uniformTypeIdentifiersErr != nil {
		panic(fmt.Errorf("load UniformTypeIdentifiers: %w", uniformTypeIdentifiersErr))
	}
	id, err := objc.Constant(uniformTypeIdentifiers, name)
	if err != nil {
		panic(err)
	}
	return id
}

// StartExternalDrag starts an AppKit dragging session for the payload's files.
func (w *MacWindow) StartExternalDrag(payload gpui.ExternalDragPayload) bool {
	files, isFiles := payload.(gpui.ExternalDragPayloadFiles)
	if !isFiles || len(files.Entries()) == 0 {
		slog.Warn("start_external_drag declined: no paths")
		return false
	}
	paths := files.Entries()

	w.state.Lock()
	nativeView := w.state.nativeView
	nativeWindow := w.state.nativeWindow
	var lastLeftMouseDownEvent objc.ID
	if w.state.lastLeftMouseDownEvent != nil {
		lastLeftMouseDownEvent = objc.Retain(w.state.lastLeftMouseDownEvent.ID())
	}
	w.state.Unlock()

	if lastLeftMouseDownEvent == objc.Nil {
		slog.Warn("start_external_drag declined: no retained left mouse down event")
		return false
	}
	defer objc.Release(lastLeftMouseDownEvent)

	event := lastLeftMouseDownEvent
	draggingItems := objc.SendClass[objc.ID](objc.GetClass("NSMutableArray"), objc.Sel("array"))
	location := objc.Send[objc.CGPoint](event, objc.Sel("locationInWindow"))
	frame := objc.MakeCGRect(location.X-16., location.Y-16., 32., 32.)

	for _, entry := range paths {
		if bytes.IndexByte([]byte(entry.Path), 0) >= 0 {
			slog.Warn("start_external_drag skipped path containing an interior nul byte")
			continue
		}
		pathBytes := append([]byte(entry.Path), 0)
		url := objc.SendClass[objc.ID](objc.GetClass("NSURL"), objc.Sel("fileURLWithFileSystemRepresentation:isDirectory:relativeToURL:"), unsafe.Pointer(&pathBytes[0]), entry.IsDir, objc.Nil)

		item := objc.Send[objc.ID](objc.Alloc(objc.GetClass("NSDraggingItem")), objc.Sel("initWithPasteboardWriter:"), url)

		isDirectory := entry.IsDir
		var extension string
		hasExtension := false
		if ext := filepath.Ext(entry.Path); len(ext) > 1 && filepath.Base(entry.Path) != ext {
			extension = ext[1:]
			hasExtension = true
		}
		provider := objc.NewBlock(func(_ objc.Block) objc.ID {
			component := objc.SendClass[objc.ID](objc.GetClass("NSDraggingImageComponent"), objc.Sel("draggingImageComponentWithKey:"), appkit.Constant("NSDraggingImageComponentIconKey"))
			workspace := objc.SendClass[objc.ID](objc.GetClass("NSWorkspace"), objc.Sel("sharedWorkspace"))
			var contentType objc.ID
			if isDirectory {
				contentType = utTypeConstant("UTTypeFolder")
			} else {
				if hasExtension {
					contentType = objc.SendClass[objc.ID](objc.GetClass("UTType"), objc.Sel("typeWithFilenameExtension:"), nsString(extension))
				}
				if contentType == objc.Nil {
					contentType = utTypeConstant("UTTypeData")
				}
			}
			icon := objc.Send[objc.ID](workspace, objc.Sel("iconForContentType:"), contentType)
			objc.SendVoid(component, objc.Sel("setContents:"), icon)
			objc.SendVoid(component, objc.Sel("setFrame:"), objc.MakeCGRect(0., 0., 32., 32.))
			return objc.Autorelease(objc.NSArrayFromIDs([]objc.ID{component}))
		})
		objc.SendVoid(item, objc.Sel("setDraggingFrame:"), frame)
		objc.SendVoid(item, objc.Sel("setImageComponentsProvider:"), provider)
		provider.Release()
		objc.SendVoid(draggingItems, objc.Sel("addObject:"), item)
		objc.Release(item)
	}

	count := objc.Send[objc.NSUInteger](draggingItems, objc.Sel("count"))
	if count == 0 {
		slog.Warn("start_external_drag declined: no dragging items")
		return false
	}

	session := objc.Send[objc.ID](nativeView, objc.Sel("beginDraggingSessionWithItems:event:source:"), draggingItems, event, nativeWindow)

	started := session != objc.Nil
	if started {
		w.state.Lock()
		w.state.syntheticDragCounter++
		w.state.Unlock()
	}
	slog.Debug(fmt.Sprintf("start_external_drag completed: started=%t, item_count=%d", started, count))
	return started
}

// PlaySystemBell plays the system beep.
func (w *MacWindow) PlaySystemBell() {
	appkit.NSBeep()
}

// RenderToImage renders scene offscreen at the window's drawable size.
func (w *MacWindow) RenderToImage(scene *gpui.Scene) (*image.RGBA, error) {
	w.state.Lock()
	defer w.state.Unlock()
	return w.state.renderer.RenderToImage(scene)
}

// A11yInit attaches the accessibility adapter to the window.
func (w *MacWindow) A11yInit(callbacks gpui.A11yCallbacks) {
	w.state.Lock()
	defer w.state.Unlock()

	activationHandler := &a11yActivationHandler{callback: callbacks.Activation}
	actionHandler := &a11yActionHandler{callback: callbacks.Action}

	w.state.a11yAdapter = newA11yAdapter(w.state.nativeWindow, activationHandler, actionHandler)
}

// A11yTreeUpdate applies treeUpdate when the adapter is active and raises its events.
func (w *MacWindow) A11yTreeUpdate(treeUpdate a11y.TreeUpdate) {
	w.state.Lock()
	var events a11yQueuedEvents
	ok := false
	if adapter := w.state.a11yAdapter; adapter != nil {
		events, ok = adapter.UpdateIfActive(func() a11y.TreeUpdate { return treeUpdate })
	}
	w.state.Unlock()
	if ok {
		events.Raise()
	}
}

// A11yUpdateWindowBounds does nothing: NSAccessibility tracks window bounds itself.
func (w *MacWindow) A11yUpdateWindowBounds() {}

type a11yActivationHandler struct {
	callback func() (a11y.TreeUpdate, bool)
}

func (h *a11yActivationHandler) RequestInitialTree() (a11y.TreeUpdate, bool) {
	return h.callback()
}

type a11yActionHandler struct {
	callback func(request a11y.ActionRequest)
}

func (h *a11yActionHandler) DoAction(request a11y.ActionRequest) {
	h.callback(request)
}

func getScaleFactor(nativeWindow objc.ID) float32 {
	screen := objc.Send[objc.ID](nativeWindow, objc.Sel("screen"))
	if screen == objc.Nil {
		return 2.0
	}
	factor := float32(appkit.NSScreen(screen).BackingScaleFactor())

	if factor == 0.0 {
		return 2.
	}
	return factor
}

func currentModifiers() gpui.Modifiers {
	modifiers := appkit.NSEventModifierFlagsNow()

	return gpui.Modifiers{
		Control:  modifiers.Contains(appkit.NSEventModifierFlagControl),
		Alt:      modifiers.Contains(appkit.NSEventModifierFlagOption),
		Shift:    modifiers.Contains(appkit.NSEventModifierFlagShift),
		Platform: modifiers.Contains(appkit.NSEventModifierFlagCommand),
		Function: modifiers.Contains(appkit.NSEventModifierFlagFunction),
	}
}

func currentCapslock() gpui.Capslock {
	modifiers := appkit.NSEventModifierFlagsNow()

	return gpui.Capslock{
		On: modifiers.Contains(appkit.NSEventModifierFlagCapsLock),
	}
}

type macClasses struct {
	window                      objc.Class
	panel                       objc.Class
	view                        objc.Class
	blurredView                 objc.Class
	windowStateArchiverDelegate objc.Class
	windowStateUnarchiver       objc.Class
}

var classes = sync.OnceValue(buildWindowClasses)

func windowClass() objc.Class { return classes().window }

func panelClass() objc.Class { return classes().panel }

func viewClass() objc.Class { return classes().view }

func blurredViewClass() objc.Class { return classes().blurredView }

func windowStateArchiverDelegateClass() objc.Class { return classes().windowStateArchiverDelegate }

func windowStateUnarchiverClass() objc.Class { return classes().windowStateUnarchiver }

func mustRegisterClass(name string, superclass string, protocols []string, methods []objc.MethodDef) objc.Class {
	class, err := objc.RegisterClass(name, objc.GetClass(superclass), protocols, nil, methods)
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
	return class
}

func buildWindowClasses() macClasses {
	var c macClasses
	c.window = buildWindowClass("GPUIWindow", "NSWindow")
	c.panel = buildWindowClass("GPUIPanel", "NSPanel")
	viewEvent := func(sel string) objc.MethodDef {
		return objc.MethodDef{Cmd: objc.Sel(sel), Fn: handleViewEvent}
	}
	viewMethods := []objc.MethodDef{
		{Cmd: objc.Sel("dealloc"), Fn: deallocView},
		{Cmd: objc.Sel("performKeyEquivalent:"), Fn: handleKeyEquivalent},
		{Cmd: objc.Sel("keyDown:"), Fn: handleKeyDown},
		{Cmd: objc.Sel("keyUp:"), Fn: handleKeyUp},
		viewEvent("mouseDown:"),
		viewEvent("mouseUp:"),
		viewEvent("rightMouseDown:"),
		viewEvent("rightMouseUp:"),
		viewEvent("otherMouseDown:"),
		viewEvent("otherMouseUp:"),
		viewEvent("mouseMoved:"),
		{Cmd: objc.Sel("resetCursorRects"), Fn: resetCursorRects},
		viewEvent("pressureChangeWithEvent:"),
		viewEvent("mouseExited:"),
		viewEvent("magnifyWithEvent:"),
		viewEvent("mouseDragged:"),
		viewEvent("rightMouseDragged:"),
		viewEvent("otherMouseDragged:"),
		viewEvent("scrollWheel:"),
		viewEvent("swipeWithEvent:"),
		viewEvent("flagsChanged:"),
		{Cmd: objc.Sel("makeBackingLayer"), Fn: makeBackingLayer},
		{Cmd: objc.Sel("viewDidChangeBackingProperties"), Fn: viewDidChangeBackingProperties},
		{Cmd: objc.Sel("setFrameSize:"), Fn: setFrameSize},
		{Cmd: objc.Sel("displayLayer:"), Fn: displayLayer},
		{Cmd: objc.Sel("validAttributesForMarkedText"), Fn: validAttributesForMarkedText},
		{Cmd: objc.Sel("hasMarkedText"), Fn: hasMarkedText},
		{Cmd: objc.Sel("markedRange"), Fn: markedRange},
		{Cmd: objc.Sel("selectedRange"), Fn: selectedRange},
		{Cmd: objc.Sel("firstRectForCharacterRange:actualRange:"), Fn: firstRectForCharacterRange},
		{Cmd: objc.Sel("insertText:replacementRange:"), Fn: insertText},
		{Cmd: objc.Sel("setMarkedText:selectedRange:replacementRange:"), Fn: setMarkedText},
		{Cmd: objc.Sel("unmarkText"), Fn: unmarkText},
		{Cmd: objc.Sel("attributedSubstringForProposedRange:actualRange:"), Fn: attributedSubstringForProposedRange},
		{Cmd: objc.Sel("viewDidChangeEffectiveAppearance"), Fn: viewDidChangeEffectiveAppearance},
		{Cmd: objc.Sel("doCommandBySelector:"), Fn: doCommandBySelector},
		{Cmd: objc.Sel("acceptsFirstMouse:"), Fn: acceptsFirstMouse},
		{Cmd: objc.Sel("_opaqueRectForWindowMoveWhenInTitlebar"), Fn: opaqueRectForWindowMoveWhenInTitlebar},
		{Cmd: objc.Sel("characterIndexForPoint:"), Fn: characterIndexForPoint},
	}
	viewMethods = append(viewMethods, nativeViewsAddViewMethods()...)
	c.view = mustRegisterClass("GPUIView", "NSView", []string{"CALayerDelegate", "NSTextInputClient"}, viewMethods)
	c.blurredView = mustRegisterClass("BlurredView", "NSVisualEffectView", nil, []objc.MethodDef{
		{Cmd: objc.Sel("initWithFrame:"), Fn: blurredViewInitWithFrame},
		{Cmd: objc.Sel("updateLayer"), Fn: blurredViewUpdateLayer},
	})
	c.windowStateArchiverDelegate = mustRegisterClass("GPUIWindowStateArchiverDelegate", "NSObject", nil, []objc.MethodDef{
		{Cmd: objc.Sel("archiver:willEncodeObject:"), Fn: windowStateArchiverWillEncodeObject},
	})
	c.windowStateUnarchiver = mustRegisterClass("GPUIWindowStateKeyedUnarchiver", "NSKeyedUnarchiver", nil, []objc.MethodDef{
		{Cmd: objc.Sel("_windowRestorationOptions"), Fn: windowStateUnarchiverRestorationOptions},
	})
	return c
}

func buildWindowClass(name string, superclass string) objc.Class {
	methods := []objc.MethodDef{
		{Cmd: objc.Sel("dealloc"), Fn: deallocWindow},
	}
	methods = append(methods, nativeViewsAddWindowMethods()...)
	methods = append(methods,
		objc.MethodDef{Cmd: objc.Sel("canBecomeMainWindow"), Fn: yes},
		objc.MethodDef{Cmd: objc.Sel("canBecomeKeyWindow"), Fn: yes},
		objc.MethodDef{Cmd: objc.Sel("windowDidResize:"), Fn: windowDidResize},
		objc.MethodDef{Cmd: objc.Sel("windowDidChangeOcclusionState:"), Fn: windowDidChangeOcclusionState},
		objc.MethodDef{Cmd: objc.Sel("windowWillEnterFullScreen:"), Fn: windowWillEnterFullscreen},
		objc.MethodDef{Cmd: objc.Sel("windowDidFailToEnterFullScreen:"), Fn: windowDidFailToEnterFullscreen},
		objc.MethodDef{Cmd: objc.Sel("windowWillExitFullScreen:"), Fn: windowWillExitFullscreen},
		objc.MethodDef{Cmd: objc.Sel("windowDidFailToExitFullScreen:"), Fn: windowDidFailToExitFullscreen},
		objc.MethodDef{Cmd: objc.Sel("windowDidExitFullScreen:"), Fn: windowDidExitFullscreen},
		objc.MethodDef{Cmd: objc.Sel("windowDidMove:"), Fn: windowDidMove},
		objc.MethodDef{Cmd: objc.Sel("windowDidChangeScreen:"), Fn: windowDidChangeScreen},
		objc.MethodDef{Cmd: objc.Sel("windowDidBecomeKey:"), Fn: windowDidChangeKeyStatus},
		objc.MethodDef{Cmd: objc.Sel("windowDidResignKey:"), Fn: windowDidChangeKeyStatus},
		objc.MethodDef{Cmd: objc.Sel("windowShouldClose:"), Fn: windowShouldClose},
		objc.MethodDef{Cmd: objc.Sel("close"), Fn: closeWindow},
		objc.MethodDef{Cmd: objc.Sel("draggingEntered:"), Fn: draggingEntered},
		objc.MethodDef{Cmd: objc.Sel("draggingUpdated:"), Fn: draggingUpdated},
		objc.MethodDef{Cmd: objc.Sel("draggingExited:"), Fn: draggingExited},
		objc.MethodDef{Cmd: objc.Sel("performDragOperation:"), Fn: performDragOperation},
		objc.MethodDef{Cmd: objc.Sel("concludeDragOperation:"), Fn: concludeDragOperation},
		objc.MethodDef{Cmd: objc.Sel("draggingSession:sourceOperationMaskForDraggingContext:"), Fn: draggingSessionSourceOperationMask},
		objc.MethodDef{Cmd: objc.Sel("draggingSession:endedAtPoint:operation:"), Fn: draggingSessionEnded},
		objc.MethodDef{Cmd: objc.Sel("addTitlebarAccessoryViewController:"), Fn: addTitlebarAccessoryViewController},
		objc.MethodDef{Cmd: objc.Sel("moveTabToNewWindow:"), Fn: moveTabToNewWindow},
		objc.MethodDef{Cmd: objc.Sel("mergeAllWindows:"), Fn: mergeAllWindows},
		objc.MethodDef{Cmd: objc.Sel("selectNextTab:"), Fn: selectNextTab},
		objc.MethodDef{Cmd: objc.Sel("selectPreviousTab:"), Fn: selectPreviousTab},
		objc.MethodDef{Cmd: objc.Sel("toggleTabBar:"), Fn: toggleTabBar},
	)
	return mustRegisterClass(name, superclass, []string{"NSDraggingSource"}, methods)
}

func isGpuiWindow(window objc.ID) bool {
	return objc.Send[bool](window, objc.Sel("isKindOfClass:"), windowClass()) ||
		objc.Send[bool](window, objc.Sel("isKindOfClass:"), panelClass())
}

func getWindowState(object objc.ID) *lockedWindowState {
	state, ok := loadWindowState(object)
	if !ok {
		panic("GPUI window state ivar")
	}
	return state
}

func windowStateIfSet(object objc.ID) (*lockedWindowState, bool) {
	return loadWindowState(object)
}

func dropWindowState(object objc.ID) {
	clearWindowState(object)
}

func yes(_ objc.ID, _ objc.SEL) bool {
	return true
}

func deallocWindow(this objc.ID, _ objc.SEL) {
	dropWindowState(this)
	objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
}

func deallocView(this objc.ID, _ objc.SEL) {
	dropWindowState(this)
	objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
}

const (
	nsTrackingMouseEnteredAndExited objc.NSUInteger = 0x01
	nsTrackingMouseMoved            objc.NSUInteger = 0x02
	nsTrackingActiveAlways          objc.NSUInteger = 0x80
	nsTrackingInVisibleRect         objc.NSUInteger = 0x200
)

func addMouseTrackingArea(nativeView objc.ID) {
	options := nsTrackingMouseEnteredAndExited |
		nsTrackingMouseMoved |
		nsTrackingActiveAlways |
		nsTrackingInVisibleRect
	trackingArea := objc.Send[objc.ID](
		objc.Alloc(objc.GetClass("NSTrackingArea")),
		objc.Sel("initWithRect:options:owner:userInfo:"),
		objc.CGRect{},
		options,
		nativeView,
		objc.Nil,
	)
	objc.SendVoid(nativeView, objc.Sel("addTrackingArea:"), trackingArea)
	objc.Release(trackingArea)
}

func nativeCursor(style gpui.CursorStyle) objc.ID {
	cursor := objc.GetClass("NSCursor")
	send := func(sel string) objc.ID { return objc.SendClass[objc.ID](cursor, objc.Sel(sel)) }
	switch style {
	case gpui.CursorStyleArrow:
		return send("arrowCursor")
	case gpui.CursorStyleIBeam:
		return send("IBeamCursor")
	case gpui.CursorStyleCrosshair:
		return send("crosshairCursor")
	case gpui.CursorStyleClosedHand:
		return send("closedHandCursor")
	case gpui.CursorStyleOpenHand:
		return send("openHandCursor")
	case gpui.CursorStylePointingHand:
		return send("pointingHandCursor")
	case gpui.CursorStyleResizeLeftRight:
		return send("resizeLeftRightCursor")
	case gpui.CursorStyleResizeUpDown:
		return send("resizeUpDownCursor")
	case gpui.CursorStyleResizeLeft:
		return send("resizeLeftCursor")
	case gpui.CursorStyleResizeRight:
		return send("resizeRightCursor")
	case gpui.CursorStyleResizeColumn:
		return send("resizeLeftRightCursor")
	case gpui.CursorStyleResizeRow:
		return send("resizeUpDownCursor")
	case gpui.CursorStyleResizeUp:
		return send("resizeUpCursor")
	case gpui.CursorStyleResizeDown:
		return send("resizeDownCursor")
	case gpui.CursorStyleResizeUpLeftDownRight:
		return send("_windowResizeNorthWestSouthEastCursor")
	case gpui.CursorStyleResizeUpRightDownLeft:
		return send("_windowResizeNorthEastSouthWestCursor")
	case gpui.CursorStyleIBeamCursorForVerticalLayout:
		return send("IBeamCursorForVerticalLayout")
	case gpui.CursorStyleOperationNotAllowed:
		return send("operationNotAllowedCursor")
	case gpui.CursorStyleDragLink:
		return send("dragLinkCursor")
	case gpui.CursorStyleDragCopy:
		return send("dragCopyCursor")
	case gpui.CursorStyleContextualMenu:
		return send("contextualMenuCursor")
	}
	return send("arrowCursor")
}

func resetCursorRects(this objc.ID, _ objc.SEL) {
	objc.SendSuper[objc.ID](this, objc.Sel("resetCursorRects"))

	windowState := getWindowState(this)
	windowState.Lock()
	if nativeViewSetsCursor(&windowState.macWindowState) {
		windowState.Unlock()
		return
	}
	cursorStyle := windowState.cursorStyle
	windowState.Unlock()

	cursor := nativeCursor(cursorStyle)

	bounds := objc.Send[objc.CGRect](this, objc.Sel("bounds"))
	objc.SendVoid(this, objc.Sel("addCursorRect:cursor:"), bounds, cursor)
}

func handleKeyEquivalent(this objc.ID, _ objc.SEL, nativeEvent objc.ID) bool {
	if isRoutingKeyEquivalent(getWindowState(this)) {
		return false
	}
	return handleKeyEvent(this, nativeEvent, true)
}

func handleKeyDown(this objc.ID, _ objc.SEL, nativeEvent objc.ID) {
	handleKeyEvent(this, nativeEvent, false)
}

func handleKeyUp(this objc.ID, _ objc.SEL, nativeEvent objc.ID) {
	handleKeyEvent(this, nativeEvent, false)
}

func isIMEInputSourceActive() bool {
	mustLoadCarbon()
	source := tisCopyCurrentKeyboardInputSource()
	if source == objc.Nil {
		return false
	}

	sourceType := tisGetInputSourceProperty(source, kTISPropertyInputSourceType)
	isInputMode := sourceType != objc.Nil &&
		cf.Equal(cf.TypeRef(sourceType), kTISTypeKeyboardInputMode)

	isASCII := tisGetInputSourceProperty(source, kTISPropertyInputSourceIsASCIICapable)
	isASCIICapable := isASCII != objc.Nil && cf.Boolean(isASCII).Value()

	cf.Release(cf.TypeRef(source))

	return isInputMode && !isASCIICapable
}

func inputContextHandleEvent(this objc.ID, nativeEvent objc.ID) bool {
	inputContext := objc.Send[objc.ID](this, objc.Sel("inputContext"))
	if inputContext == objc.Nil {
		return false
	}
	return objc.Send[bool](inputContext, objc.Sel("handleEvent:"), nativeEvent)
}

func handleKeyEvent(this objc.ID, nativeEvent objc.ID, keyEquivalent bool) bool {
	windowState := getWindowState(this)
	windowState.Lock()

	windowHeight := windowState.contentSize().Height
	event := platformInputFromNative(nativeEvent, gpui.Some(windowHeight))

	if event == nil {
		windowState.Unlock()
		return false
	}

	runCallback := func(event gpui.PlatformInput) bool {
		windowState.Lock()
		callback := windowState.eventCallback
		windowState.eventCallback = nil
		windowState.Unlock()
		handled := false
		if callback != nil {
			handled = !callback(event).Propagate
		}
		windowState.Lock()
		windowState.eventCallback = callback
		windowState.Unlock()
		return handled
	}

	switch keyDownEvent := event.(type) {
	case gpui.KeyDownEvent:
		if keyEquivalent {
			ev := keyDownEvent
			windowState.lastKeyEquivalent = &ev
		} else {
			last := windowState.lastKeyEquivalent
			windowState.lastKeyEquivalent = nil
			if last != nil && *last == keyDownEvent {
				windowState.Unlock()
				return false
			}
		}

		windowState.Unlock()

		markedRange, hasHandler := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) gpui.Option[gpui.Range] {
			return gpui.OptionFrom(inputHandler.MarkedTextRange())
		})
		isComposing := hasHandler && markedRange.IsSome()

		keystroke := keyDownEvent.Keystroke
		keyChar, hasKeyChar := keystroke.KeyChar.Get()
		isIMEPrintableKey := !isComposing &&
			hasKeyChar && !strings.ContainsFunc(keyChar, unicode.IsControl) &&
			!keystroke.Modifiers.Control &&
			!keystroke.Modifiers.Function &&
			!keystroke.Modifiers.Platform &&
			isIMEInputSourceActive()
		if isIMEPrintableKey {
			prefers, ok := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) bool {
				return inputHandler.QueryPrefersIMEForPrintableKeys()
			})
			isIMEPrintableKey = ok && prefers
		}

		if isComposing ||
			isIMEPrintableKey ||
			(!hasKeyChar &&
				!keystroke.Modifiers.Control &&
				!keystroke.Modifiers.Function &&
				!keystroke.Modifiers.Platform) {
			windowState.Lock()
			ks := keystroke
			windowState.keystrokeForDoCommand = &ks
			windowState.doCommandHandled = gpui.None[bool]()
			windowState.Unlock()

			handled := inputContextHandleEvent(this, nativeEvent)
			windowState.Lock()
			windowState.keystrokeForDoCommand = nil
			doCommandHandled := windowState.doCommandHandled
			windowState.doCommandHandled = gpui.None[bool]()
			windowState.Unlock()
			if commandHandled, ok := doCommandHandled.Get(); ok {
				return commandHandled
			} else if handled {
				return true
			}

			return runCallback(keyDownEvent)
		}

		if runCallback(keyDownEvent) {
			return true
		}

		if keyDownEvent.IsHeld && hasKeyChar {
			handled, ok := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) bool {
				if !inputHandler.ApplePressAndHoldEnabled() {
					inputHandler.ReplaceTextInRange(gpui.None[gpui.Range](), keyChar)
					return true
				}
				return false
			})
			if ok && handled {
				return true
			}
		}

		if keyEquivalent && keystroke.Modifiers != gpui.ModifiersFunction() {
			return false
		}

		return inputContextHandleEvent(this, nativeEvent)

	case gpui.KeyUpEvent:
		windowState.Unlock()
		return runCallback(event)

	default:
		windowState.Unlock()
		return false
	}
}

func handleViewEvent(this objc.ID, _ objc.SEL, nativeEvent objc.ID) {
	windowState := getWindowState(this)
	weakWindowState := weak.Make(windowState)
	windowState.Lock()
	locked := true
	defer func() {
		if locked {
			windowState.Unlock()
		}
	}()
	windowHeight := windowState.contentSize().Height
	nativeEventType := appkit.NSEvent(nativeEvent).Type()
	switch nativeEventType {
	case appkit.NSEventTypeLeftMouseDown:
		if windowState.lastLeftMouseDownEvent != nil {
			windowState.lastLeftMouseDownEvent.Release()
		}
		windowState.lastLeftMouseDownEvent = objc.RetainObject(nativeEvent)
	case appkit.NSEventTypeLeftMouseUp:
		if windowState.lastLeftMouseDownEvent != nil {
			windowState.lastLeftMouseDownEvent.Release()
		}
		windowState.lastLeftMouseDownEvent = nil
	}
	event := platformInputFromNative(nativeEvent, gpui.Some(windowHeight))
	if event == nil {
		return
	}

	switch event.(type) {
	case gpui.MouseMoveEvent, gpui.MouseDownEvent, gpui.MouseUpEvent, gpui.MousePressureEvent,
		gpui.MouseExitEvent, gpui.ScrollEvent, gpui.PinchEvent:
		windowState.cursorVisible.Store(true)
	}

	switch e := event.(type) {
	case gpui.MouseDownEvent:
		if e.Button == gpui.MouseButtonLeft && e.Modifiers.Control {
			e.Button = gpui.MouseButtonRight
			e.Modifiers.Control = false
			e.ClickCount = 1
			event = e
		} else if e.Button == gpui.MouseButtonLeft && windowState.firstMouse {
			e.FirstMouse = true
			event = e
			windowState.firstMouse = false
		}
	case gpui.MouseUpEvent:
		if e.Button == gpui.MouseButtonLeft && e.Modifiers.Control {
			e.Button = gpui.MouseButtonRight
			e.Modifiers.Control = false
			e.ClickCount = 1
			event = e
		}
	}

	switch e := event.(type) {
	case gpui.MouseDownEvent:
		windowState.Unlock()
		locked = false
		inputContextHandleEvent(this, nativeEvent)
		windowState.Lock()
		locked = true
	case gpui.MouseMoveEvent:
		if e.PressedButton.IsSome() && !windowState.externalFilesDragged {
			windowState.syntheticDragCounter++
			executor := windowState.foregroundExecutor
			dragID := windowState.syntheticDragCounter
			backgroundExecutor := windowState.backgroundExecutor
			executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
				syntheticDrag(s, weakWindowState, dragID, e, backgroundExecutor)
				return struct{}{}, nil
			}).Detach()
		}
	case gpui.MouseUpEvent:
		windowState.syntheticDragCounter++
	case gpui.ModifiersChangedEvent:
		if prev, ok := windowState.previousModifiersChangedEvent.(gpui.ModifiersChangedEvent); ok &&
			prev.Modifiers == e.Modifiers && prev.Capslock == e.Capslock {
			return
		}
		windowState.previousModifiersChangedEvent = event
	}

	if callback := windowState.eventCallback; callback != nil {
		windowState.eventCallback = nil
		windowState.Unlock()
		locked = false
		callback(event)
		windowState.Lock()
		locked = true
		windowState.eventCallback = callback
	}
}

func visibility(state *macWindowState) gpui.WindowVisibility {
	isVisible := state.window().OcclusionState()&appkit.NSWindowOcclusionStateVisible != 0
	if isVisible {
		return gpui.WindowVisibilityVisible
	}
	return gpui.WindowVisibilityHidden
}

func reportVisibility(windowState *lockedWindowState) {
	windowState.Lock()
	if windowState.lastVisibility.IsNone() {
		windowState.Unlock()
		return
	}
	executor := windowState.foregroundExecutor
	windowState.Unlock()

	detachOnForeground(executor, func() {
		windowState.Lock()
		vis := visibility(&windowState.macWindowState)
		if windowState.lastVisibility == gpui.Some(vis) {
			windowState.Unlock()
			return
		}
		windowState.lastVisibility = gpui.Some(vis)
		callback := windowState.visibilityCallback
		if callback == nil {
			windowState.Unlock()
			return
		}
		windowState.visibilityCallback = nil
		windowState.Unlock()
		callback(vis)
		windowState.Lock()
		windowState.visibilityCallback = callback
		windowState.Unlock()
	})
}

func windowDidChangeOcclusionState(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	if windowState.window().OcclusionState()&appkit.NSWindowOcclusionStateVisible != 0 {
		windowState.moveTrafficLight()
		windowState.startDisplayLink(windowState)
	} else {
		windowState.stopDisplayLink()
	}
	windowState.Unlock()
	reportVisibility(windowState)
}

func windowDidResize(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	defer windowState.Unlock()
	windowState.moveTrafficLight()
}

func windowWillEnterFullscreen(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	defer windowState.Unlock()
	windowState.fullscreenRestoreBounds = windowState.bounds()
	windowState.isExitingFullscreen = false
	windowState.preFullscreenTrafficLightFrames = windowState.trafficLightFrames
	windowState.restoreTrafficLight()
	windowState.applyFullscreenTitlebarOpacity()
}

func windowDidFailToEnterFullscreen(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	defer windowState.Unlock()
	windowState.restoreWindowedTitlebarTransparency()
	windowState.trafficLightFrames = windowState.preFullscreenTrafficLightFrames
	windowState.preFullscreenTrafficLightFrames = nil
	windowState.moveTrafficLight()
}

func windowWillExitFullscreen(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	defer windowState.Unlock()
	windowState.isExitingFullscreen = true
	windowState.restoreWindowedTitlebarTransparency()
	windowState.moveTrafficLight()
}

func windowDidFailToExitFullscreen(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	defer windowState.Unlock()
	windowState.isExitingFullscreen = false
	if windowState.isFullscreen() {
		windowState.applyFullscreenTitlebarOpacity()
	}
	windowState.restoreTrafficLight()
}

func windowDidExitFullscreen(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	defer windowState.Unlock()
	windowState.isExitingFullscreen = false
	windowState.trafficLightFrames = windowState.preFullscreenTrafficLightFrames
	windowState.preFullscreenTrafficLightFrames = nil
	windowState.moveTrafficLight()
}

func isMacosVersionAtLeast(version nsOperatingSystemVersion) bool {
	processInfo := objc.SendClass[objc.ID](objc.GetClass("NSProcessInfo"), objc.Sel("processInfo"))
	return msgSendIsOperatingSystemAtLeast(processInfo, objc.Sel("isOperatingSystemAtLeastVersion:"), version)
}

func nsErrorDescription(err objc.ID) string {
	if err == objc.Nil {
		return "unknown error"
	}
	description := objc.Send[objc.ID](err, objc.Sel("localizedDescription"))
	return nsStringToStr(description)
}

func windowDidMove(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	callback := windowState.movedCallback
	if callback == nil {
		windowState.Unlock()
		return
	}
	windowState.movedCallback = nil
	windowState.Unlock()
	callback()
	windowState.Lock()
	windowState.movedCallback = callback
	windowState.Unlock()
}

func updateWindowScaleFactor(windowState *lockedWindowState) {
	windowState.Lock()
	scaleFactor := windowState.scaleFactor()
	size := windowState.contentSize()
	drawableSize := size.ToDevicePixels(scaleFactor)
	layer := objc.ID(windowState.renderer.LayerPtr())
	if layer != objc.Nil {
		objc.SendVoid(layer, objc.Sel("setContentsScale:"), float64(scaleFactor))
	}

	windowState.renderer.UpdateDrawableSize(drawableSize)

	callback := windowState.resizeCallback
	if callback == nil {
		windowState.Unlock()
		return
	}
	windowState.resizeCallback = nil
	contentSize := windowState.contentSize()
	scaleFactor = windowState.scaleFactor()
	windowState.Unlock()
	callback(contentSize, scaleFactor)
	windowState.Lock()
	windowState.resizeCallback = callback
	windowState.Unlock()
}

func windowDidChangeScreen(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	windowState.startDisplayLink(windowState)
	windowState.Unlock()
	updateWindowScaleFactor(windowState)
	reportDisplayChange(windowState)
}

func reportDisplayChange(windowState *lockedWindowState) {
	windowState.Lock()
	executor := windowState.foregroundExecutor
	windowState.Unlock()
	detachOnForeground(executor, func() {
		windowState.Lock()
		callback := windowState.displayChangedCallback
		windowState.displayChangedCallback = nil
		windowState.Unlock()
		if callback != nil {
			callback()
			windowState.Lock()
			windowState.displayChangedCallback = callback
			windowState.Unlock()
		}
	})
}

func windowDidChangeKeyStatus(this objc.ID, selector objc.SEL, _ objc.ID) {
	signalAt := gpui.OptionFrom(gpui.PlatformFrameSignalCapture(scheduler.InstantNow))
	windowState := getWindowState(this)
	windowState.Lock()
	isActive := objc.Send[bool](windowState.nativeWindow, objc.Sel("isKeyWindow"))

	windowState.cursorVisible.Store(true)

	if selector == objc.Sel("windowDidBecomeKey:") && !isActive {
		nativeWindow := windowState.nativeWindow
		windowState.Unlock()
		objc.SendVoid(nativeWindow, objc.Sel("resignKeyWindow"))
		return
	}

	executor := windowState.foregroundExecutor
	windowState.Unlock()

	windowState.Lock()
	isFocused := isActive && gpuiViewIsFocused(&windowState.macWindowState)
	var a11yEvents a11yQueuedEvents
	hasA11yEvents := false
	if adapter := windowState.a11yAdapter; adapter != nil {
		a11yEvents, hasA11yEvents = adapter.UpdateViewFocusState(isFocused)
	}
	windowState.Unlock()
	if hasA11yEvents {
		a11yEvents.Raise()
	}

	if selector == objc.Sel("windowDidBecomeKey:") && isActive {
		windowState := getWindowState(this)
		windowState.Lock()

		if windowState.activatedLeastOnce {
			if callback := windowState.requestFrameCallback; callback != nil {
				windowState.requestFrameCallback = nil
				windowState.renderer.SetPresentsWithTransaction(true)
				windowState.stopDisplayLink()
				windowState.Unlock()
				callback(gpui.RequestFrameOptions{
					SignalAt:     signalAt,
					SignalSource: gpui.FrameRequestSourceNativeCallback,
				})

				windowState.Lock()
				windowState.requestFrameCallback = callback
				windowState.renderer.SetPresentsWithTransaction(false)
				windowState.startDisplayLink(windowState)
			}
		} else {
			windowState.activatedLeastOnce = true
		}
		windowState.Unlock()
	}

	detachOnForeground(executor, func() {
		windowState.Lock()
		if isActive {
			windowState.moveTrafficLight()
		}

		callback := windowState.activateCallback
		if callback == nil {
			windowState.Unlock()
			return
		}
		windowState.activateCallback = nil
		windowState.Unlock()
		callback(isActive)
		windowState.Lock()
		windowState.activateCallback = callback
		windowState.Unlock()
	})
}

func windowShouldClose(this objc.ID, _ objc.SEL, _ objc.ID) bool {
	windowState := getWindowState(this)
	windowState.Lock()
	callback := windowState.shouldCloseCallback
	if callback == nil {
		windowState.Unlock()
		return true
	}
	windowState.shouldCloseCallback = nil
	windowState.Unlock()
	shouldClose := callback()
	windowState.Lock()
	windowState.shouldCloseCallback = callback
	windowState.Unlock()
	return shouldClose
}

func closeWindow(this objc.ID, _ objc.SEL) {
	windowState := getWindowState(this)
	windowState.Lock()
	windowState.closed.Store(true)
	closeCallback := windowState.closeCallback
	windowState.closeCallback = nil
	simpleFullscreen := windowState.simpleFullscreenState
	windowState.simpleFullscreenState = nil
	windowState.Unlock()

	if simpleFullscreen != nil {
		popSimpleFullscreenPresentationOptions()
	}

	if closeCallback != nil {
		closeCallback()
	}

	objc.SendSuper[objc.ID](this, objc.Sel("close"))
}

func makeBackingLayer(this objc.ID, _ objc.SEL) objc.ID {
	windowState := getWindowState(this)
	windowState.Lock()
	defer windowState.Unlock()
	return objc.ID(windowState.renderer.LayerPtr())
}

func viewDidChangeBackingProperties(this objc.ID, _ objc.SEL) {
	updateWindowScaleFactor(getWindowState(this))
}

func setFrameSize(this objc.ID, _ objc.SEL, size objc.CGSize) {
	convert := func(value objc.CGSize) gpui.Size[gpui.Pixels] {
		return gpui.NewSize(gpui.Px(float32(value.Width)), gpui.Px(float32(value.Height)))
	}

	windowState := getWindowState(this)
	windowState.Lock()

	newSize := convert(size)
	oldFrame := objc.Send[objc.CGRect](this, objc.Sel("frame"))
	oldSize := convert(oldFrame.Size)

	if oldSize == newSize {
		windowState.Unlock()
		return
	}

	objc.SendSuper[objc.ID](this, objc.Sel("setFrameSize:"), size)

	scaleFactor := windowState.scaleFactor()
	drawableSize := newSize.ToDevicePixels(scaleFactor)
	windowState.renderer.UpdateDrawableSize(drawableSize)

	callback := windowState.resizeCallback
	if callback == nil {
		windowState.Unlock()
		return
	}
	windowState.resizeCallback = nil
	contentSize := windowState.contentSize()
	scaleFactor = windowState.scaleFactor()
	windowState.Unlock()
	callback(contentSize, scaleFactor)
	windowState.Lock()
	windowState.resizeCallback = callback
	windowState.Unlock()
}

func displayLayer(this objc.ID, _ objc.SEL, _ objc.ID) {
	signalAt := gpui.OptionFrom(gpui.PlatformFrameSignalCapture(scheduler.InstantNow))
	windowState := getWindowState(this)
	windowState.Lock()
	callback := windowState.requestFrameCallback
	if callback == nil {
		windowState.Unlock()
		return
	}
	windowState.requestFrameCallback = nil
	windowState.renderer.SetPresentsWithTransaction(true)
	windowState.stopDisplayLink()
	windowState.Unlock()
	callback(gpui.RequestFrameOptions{
		SignalAt:     signalAt,
		SignalSource: gpui.FrameRequestSourceNativeCallback,
	})

	windowState.Lock()
	windowState.requestFrameCallback = callback
	windowState.renderer.SetPresentsWithTransaction(false)
	windowState.startDisplayLink(windowState)
	windowState.Unlock()
	checkPointerAfterFrame(windowState)
}

var stepCallback = sync.OnceValue(func() gcd.Function {
	return gcd.NewFunction(step)
})

func step(view uintptr) {
	windowState := getWindowState(objc.ID(view))
	windowState.Lock()

	callback := windowState.requestFrameCallback
	if callback == nil {
		windowState.Unlock()
		return
	}
	windowState.requestFrameCallback = nil
	signalAt := gpui.None[scheduler.Instant]()
	signalSource := gpui.FrameRequestSourceNativeCallback
	if frameSource := windowState.frameSource.displayLink; windowState.frameSource.metal == nil && frameSource != nil {
		if at, source, ok := frameSource.TakeSignal(); ok {
			signalAt = gpui.Some(at)
			signalSource = source
		}
	}
	windowState.Unlock()
	callback(gpui.RequestFrameOptions{
		SignalAt:     signalAt,
		SignalSource: signalSource,
	})
	windowState.Lock()
	windowState.requestFrameCallback = callback
	windowState.Unlock()
	checkPointerAfterFrame(windowState)
}

func metalFrame(windowState *lockedWindowState, update objc.ID) {
	signalAt := gpui.OptionFrom(gpui.PlatformFrameSignalCapture(scheduler.InstantNow))
	windowState.Lock()
	metalFrames := windowState.frameSource.metal
	if metalFrames == nil {
		windowState.Unlock()
		return
	}
	callback := windowState.requestFrameCallback
	if callback == nil {
		windowState.Unlock()
		return
	}
	windowState.requestFrameCallback = nil
	windowState.renderer.SetFrameDrawable(mtl.Drawable(objc.Send[objc.ID](update, objc.Sel("drawable"))))
	windowState.Unlock()
	metalFrames.BeginFrame()
	callback(gpui.RequestFrameOptions{
		SignalAt:     signalAt,
		SignalSource: gpui.FrameRequestSourceNativeCallback,
	})
	windowState.Lock()
	windowState.requestFrameCallback = callback
	usedDrawable := windowState.renderer.SetFrameDrawable(0) == 0
	metalFrames.EndFrame(usedDrawable)
	windowState.Unlock()
	checkPointerAfterFrame(windowState)
}

func validAttributesForMarkedText(_ objc.ID, _ objc.SEL) objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSArray"), objc.Sel("array"))
}

func hasMarkedText(this objc.ID, _ objc.SEL) bool {
	result, ok := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) gpui.Option[gpui.Range] {
		return gpui.OptionFrom(inputHandler.MarkedTextRange())
	})
	return ok && result.IsSome()
}

func markedRange(this objc.ID, _ objc.SEL) objc.NSRange {
	result, ok := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) gpui.Option[gpui.Range] {
		return gpui.OptionFrom(inputHandler.MarkedTextRange())
	})
	if r, some := result.Get(); ok && some {
		return nsRangeFromRange(r).toObjc()
	}
	return nsRangeInvalid().toObjc()
}

func selectedRange(this objc.ID, _ objc.SEL) objc.NSRange {
	result, ok := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) gpui.Option[gpui.UTF16Selection] {
		return gpui.OptionFrom(inputHandler.SelectedTextRange(false))
	})
	if selection, some := result.Get(); ok && some {
		return nsRangeFromRange(selection.Range).toObjc()
	}
	return nsRangeInvalid().toObjc()
}

func fromObjcRange(r objc.NSRange) NSRange {
	return NSRange{Location: r.Location, Length: r.Length}
}

func firstRectForCharacterRange(this objc.ID, _ objc.SEL, nsRange objc.NSRange, _ *objc.NSRange) objc.CGRect {
	frame := getFrame(this)
	result, ok := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) gpui.Option[gpui.Bounds[gpui.Pixels]] {
		r, ok := fromObjcRange(nsRange).toRange()
		if !ok {
			return gpui.None[gpui.Bounds[gpui.Pixels]]()
		}
		return gpui.OptionFrom(inputHandler.BoundsForRange(r))
	})
	bounds, some := result.Get()
	if !ok || !some {
		return objc.MakeCGRect(0, 0, 0, 0)
	}
	return objc.MakeCGRect(
		frame.Origin.X+objc.CGFloat(bounds.Origin.X.AsF32()),
		frame.Origin.Y+frame.Size.Height-objc.CGFloat(bounds.Origin.Y.AsF32())-objc.CGFloat(bounds.Size.Height.AsF32()),
		objc.CGFloat(bounds.Size.Width.AsF32()),
		objc.CGFloat(bounds.Size.Height.AsF32()),
	)
}

const nsWindowStyleMaskFullSizeContentView appkit.NSWindowStyleMask = 1 << 15

func getFrame(this objc.ID) objc.CGRect {
	state := getWindowState(this)
	state.Lock()
	defer state.Unlock()
	frame := objc.Send[objc.CGRect](state.nativeWindow, objc.Sel("frame"))
	contentLayoutRect := objc.Send[objc.CGRect](state.nativeWindow, objc.Sel("contentLayoutRect"))
	styleMask := objc.Send[appkit.NSWindowStyleMask](state.nativeWindow, objc.Sel("styleMask"))
	if !styleMask.Contains(nsWindowStyleMaskFullSizeContentView) {
		frame.Origin.Y -= frame.Size.Height - contentLayoutRect.Size.Height
	}
	return frame
}

func textFromNative(text objc.ID) string {
	if objc.Send[bool](text, objc.Sel("isKindOfClass:"), objc.GetClass("NSAttributedString")) {
		text = objc.Send[objc.ID](text, objc.Sel("string"))
	}
	return nsStringToStr(text)
}

func insertText(this objc.ID, _ objc.SEL, text objc.ID, replacementRange objc.NSRange) {
	str := textFromNative(text)
	replacement := gpui.OptionFrom(fromObjcRange(replacementRange).toRange())
	withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) struct{} {
		inputHandler.ReplaceTextInRange(replacement, str)
		return struct{}{}
	})
}

func setMarkedText(this objc.ID, _ objc.SEL, text objc.ID, selectedRange objc.NSRange, replacementRange objc.NSRange) {
	str := textFromNative(text)
	selected := gpui.OptionFrom(fromObjcRange(selectedRange).toRange())
	replacement := gpui.OptionFrom(fromObjcRange(replacementRange).toRange())
	withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) struct{} {
		inputHandler.ReplaceAndMarkTextInRange(replacement, str, selected)
		return struct{}{}
	})
}

func unmarkText(this objc.ID, _ objc.SEL) {
	withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) struct{} {
		inputHandler.UnmarkText()
		return struct{}{}
	})
}

func attributedSubstringForProposedRange(this objc.ID, _ objc.SEL, proposed objc.NSRange, actualRange *objc.NSRange) objc.ID {
	result, ok := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) objc.ID {
		r, ok := fromObjcRange(proposed).toRange()
		if !ok || r.Start >= r.End {
			return objc.Nil
		}
		var adjusted gpui.Option[gpui.Range]
		selectedText, ok := inputHandler.TextForRange(r, &adjusted)
		if !ok {
			return objc.Nil
		}
		if a, some := adjusted.Get(); some && a != r && actualRange != nil {
			*actualRange = nsRangeFromRange(a).toObjc()
		}
		str := objc.Alloc(objc.GetClass("NSAttributedString"))
		return objc.Send[objc.ID](str, objc.Sel("initWithString:"), nsString(selectedText))
	})
	if !ok {
		return objc.Nil
	}
	return result
}

func doCommandBySelector(this objc.ID, _ objc.SEL, _ objc.SEL) {
	state := getWindowState(this)
	state.Lock()
	keystroke := state.keystrokeForDoCommand
	state.keystrokeForDoCommand = nil
	eventCallback := state.eventCallback
	state.eventCallback = nil
	state.Unlock()

	if keystroke != nil && eventCallback != nil {
		handled := eventCallback(gpui.KeyDownEvent{
			Keystroke:            *keystroke,
			IsHeld:               false,
			PreferCharacterInput: false,
		})
		state.Lock()
		state.doCommandHandled = gpui.Some(!handled.Propagate)
		state.Unlock()
	}

	state.Lock()
	state.eventCallback = eventCallback
	state.Unlock()
}

func viewDidChangeEffectiveAppearance(this objc.ID, _ objc.SEL) {
	state := getWindowState(this)
	state.Lock()
	callback := state.appearanceChangedCallback
	state.appearanceChangedCallback = nil
	state.Unlock()

	if callback != nil {
		callback()
		state.Lock()
		state.appearanceChangedCallback = callback
		state.Unlock()
	}

	state.Lock()
	state.moveTrafficLight()
	state.Unlock()
}

func acceptsFirstMouse(this objc.ID, _ objc.SEL, _ objc.ID) bool {
	windowState := getWindowState(this)
	windowState.Lock()
	defer windowState.Unlock()
	windowState.firstMouse = true
	return true
}

func opaqueRectForWindowMoveWhenInTitlebar(this objc.ID, _ objc.SEL) objc.CGRect {
	zeroRect := objc.MakeCGRect(0, 0, 0, 0)
	windowState := getWindowState(this)
	windowState.Lock()
	appOwnsTitlebarDrag := windowState.appOwnsTitlebarDrag
	windowState.Unlock()
	if appOwnsTitlebarDrag {
		return objc.Send[objc.CGRect](this, objc.Sel("bounds"))
	}
	return zeroRect
}

func characterIndexForPoint(this objc.ID, _ objc.SEL, position objc.CGPoint) objc.NSUInteger {
	point := screenPointToGpuiPoint(this, position)
	result, ok := withInputHandler(this, func(inputHandler *gpui.PlatformInputHandler) gpui.Option[int] {
		return gpui.OptionFrom(inputHandler.CharacterIndexForPoint(point))
	})
	if index, some := result.Get(); ok && some {
		return objc.NSUInteger(index)
	}
	return objc.NSUInteger(objc.NSNotFound)
}

func screenPointToGpuiPoint(this objc.ID, position objc.CGPoint) gpui.Point[gpui.Pixels] {
	frame := getFrame(this)
	windowX := position.X - frame.Origin.X
	windowY := frame.Size.Height - (position.Y - frame.Origin.Y)

	return gpui.NewPoint(gpui.Px(float32(windowX)), gpui.Px(float32(windowY)))
}

const (
	nsDragOperationNone objc.NSUInteger = 0
	nsDragOperationCopy objc.NSUInteger = 1
	nsDragOperationMove objc.NSUInteger = 16

	nsDraggingContextOutsideApplication objc.NSInteger = 0
	nsDraggingContextWithinApplication  objc.NSInteger = 1
)

func isDragFromThisWindow(this objc.ID, draggingInfo objc.ID) bool {
	source := objc.Send[objc.ID](draggingInfo, objc.Sel("draggingSource"))
	return source == this
}

func draggingEntered(this objc.ID, _ objc.SEL, draggingInfo objc.ID) objc.NSUInteger {
	isSourceWindow := isDragFromThisWindow(this, draggingInfo)
	windowState := getWindowState(this)
	position := dragEventPosition(windowState, draggingInfo)
	if paths, ok := externalPathsFromEvent(draggingInfo); ok &&
		sendFileDropEvent(windowState, gpui.FileDropEventEntered{Position: position, Paths: paths}) {
		if isSourceWindow {
			return nsDragOperationMove
		}
		return nsDragOperationCopy
	}
	return nsDragOperationNone
}

func draggingUpdated(this objc.ID, _ objc.SEL, draggingInfo objc.ID) objc.NSUInteger {
	isSourceWindow := isDragFromThisWindow(this, draggingInfo)
	windowState := getWindowState(this)
	position := dragEventPosition(windowState, draggingInfo)
	if sendFileDropEvent(windowState, gpui.FileDropEventPending{Position: position}) {
		if isSourceWindow {
			return nsDragOperationMove
		}
		return nsDragOperationCopy
	}
	return nsDragOperationNone
}

func draggingExited(this objc.ID, _ objc.SEL, _ objc.ID) {
	sendFileDropEvent(getWindowState(this), gpui.FileDropEventExited{})
}

func performDragOperation(this objc.ID, _ objc.SEL, draggingInfo objc.ID) bool {
	windowState := getWindowState(this)
	position := dragEventPosition(windowState, draggingInfo)
	return sendFileDropEvent(windowState, gpui.FileDropEventSubmit{Position: position})
}

func externalPathsFromEvent(draggingInfo objc.ID) (gpui.ExternalPaths, bool) {
	pasteboard := objc.Send[objc.ID](draggingInfo, objc.Sel("draggingPasteboard"))
	if pasteboard == objc.Nil {
		return nil, false
	}
	filenames := objc.Send[objc.ID](pasteboard, objc.Sel("propertyListForType:"), appkit.Constant("NSFilenamesPboardType"))
	if filenames == objc.Nil {
		return nil, false
	}
	var paths gpui.ExternalPaths
	for _, file := range objc.NSArrayToIDs(filenames) {
		paths = append(paths, nsStringToStr(file))
	}
	return paths, true
}

func concludeDragOperation(this objc.ID, _ objc.SEL, _ objc.ID) {
	sendFileDropEvent(getWindowState(this), gpui.FileDropEventExited{})
}

func draggingSessionSourceOperationMask(_ objc.ID, _ objc.SEL, _ objc.ID, context objc.NSInteger) objc.NSUInteger {
	var operation objc.NSUInteger
	switch context {
	case nsDraggingContextOutsideApplication:
		operation = nsDragOperationCopy
	case nsDraggingContextWithinApplication:
		operation = nsDragOperationCopy | nsDragOperationMove
	default:
		operation = nsDragOperationCopy | nsDragOperationMove
	}
	slog.Debug(fmt.Sprintf("dragging_session_source_operation_mask: context=%d, operation=%d", context, operation))
	return operation
}

func draggingSessionEnded(this objc.ID, _ objc.SEL, _ objc.ID, _ objc.CGPoint, operation objc.NSUInteger) {
	slog.Debug(fmt.Sprintf("dragging_session_ended operation=%d", operation))
	windowState := getWindowState(this)
	windowState.Lock()
	windowState.syntheticDragCounter++
	if windowState.lastLeftMouseDownEvent != nil {
		windowState.lastLeftMouseDownEvent.Release()
	}
	windowState.lastLeftMouseDownEvent = nil
	windowState.Unlock()
	sendFileDropEvent(windowState, gpui.FileDropEventEnded{})
}

func syntheticDragButtonIsPressed(button gpui.Option[gpui.MouseButton], pressed objc.NSUInteger) bool {
	b, ok := button.Get()
	if !ok {
		return false
	}
	var bit uint
	switch b {
	case gpui.MouseButtonLeft:
		bit = 0
	case gpui.MouseButtonRight:
		bit = 1
	case gpui.MouseButtonMiddle:
		bit = 2
	case gpui.MouseButtonNavigateBack:
		bit = 3
	case gpui.MouseButtonNavigateForward:
		bit = 4
	default:
		return false
	}
	return pressed&(1<<bit) != 0
}

func syntheticDrag(s *scheduler.Scope, weakWindowState weak.Pointer[lockedWindowState], dragID int, event gpui.MouseMoveEvent, executor gpui.BackgroundExecutor) {
	for {
		_, _ = executor.Timer(16 * time.Millisecond).Await(s)
		windowState := weakWindowState.Value()
		if windowState == nil {
			break
		}
		windowState.Lock()
		if windowState.syntheticDragCounter != dragID {
			windowState.Unlock()
			break
		}
		pressed := objc.SendClass[objc.NSUInteger](objc.GetClass("NSEvent"), objc.Sel("pressedMouseButtons"))
		if !syntheticDragButtonIsPressed(event.PressedButton, pressed) {
			windowState.syntheticDragCounter++
			windowState.Unlock()
			break
		}
		callback := windowState.eventCallback
		if callback == nil {
			windowState.Unlock()
			continue
		}
		windowState.eventCallback = nil
		windowState.Unlock()
		callback(event)
		windowState.Lock()
		windowState.eventCallback = callback
		windowState.Unlock()
	}
}

func sendFileDropEvent(windowState *lockedWindowState, fileDropEvent gpui.FileDropEvent) bool {
	externalFilesDragged := gpui.None[bool]()
	switch fileDropEvent.(type) {
	case gpui.FileDropEventEntered:
		externalFilesDragged = gpui.Some(true)
	case gpui.FileDropEventExited, gpui.FileDropEventEnded:
		externalFilesDragged = gpui.Some(false)
	}

	windowState.Lock()
	callback := windowState.eventCallback
	if callback == nil {
		windowState.Unlock()
		return false
	}
	windowState.eventCallback = nil
	windowState.Unlock()
	callback(fileDropEvent)
	windowState.Lock()
	windowState.eventCallback = callback
	if dragged, ok := externalFilesDragged.Get(); ok {
		windowState.externalFilesDragged = dragged
	}
	windowState.Unlock()
	return true
}

func dragEventPosition(windowState *lockedWindowState, draggingInfo objc.ID) gpui.Point[gpui.Pixels] {
	dragLocation := objc.Send[objc.CGPoint](draggingInfo, objc.Sel("draggingLocation"))
	windowState.Lock()
	height := windowState.contentSize().Height
	windowState.Unlock()
	return convertMousePosition(dragLocation, height)
}

func withInputHandler[R any](window objc.ID, f func(inputHandler *gpui.PlatformInputHandler) R) (R, bool) {
	windowState := getWindowState(window)
	windowState.Lock()
	inputHandler := windowState.inputHandler
	if inputHandler == nil {
		windowState.Unlock()
		var zero R
		return zero, false
	}
	windowState.inputHandler = nil
	windowState.Unlock()
	result := f(inputHandler)
	windowState.Lock()
	windowState.inputHandler = inputHandler
	windowState.Unlock()
	return result, true
}

func displayIDForScreen(screen objc.ID) (uint32, bool) {
	if screen == objc.Nil {
		return 0, false
	}

	deviceDescription := objc.Send[objc.ID](screen, objc.Sel("deviceDescription"))
	if deviceDescription == objc.Nil {
		return 0, false
	}
	screenNumberKey := nsString("NSScreenNumber")
	screenNumber := objc.Send[objc.ID](deviceDescription, objc.Sel("objectForKey:"), screenNumberKey)
	if screenNumber == objc.Nil {
		return 0, false
	}
	number := objc.Send[objc.NSUInteger](screenNumber, objc.Sel("unsignedIntegerValue"))
	return uint32(number), true
}

const (
	nsVisualEffectMaterialUnderWindowBackground objc.NSInteger = 21
	nsVisualEffectBlendingModeBehindWindow      objc.NSInteger = 0
	nsVisualEffectStateActive                   objc.NSInteger = 1
)

func blurredViewInitWithFrame(this objc.ID, _ objc.SEL, frame objc.CGRect) objc.ID {
	view := objc.SendSuper[objc.ID](this, objc.Sel("initWithFrame:"), frame)
	objc.SendVoid(view, objc.Sel("setMaterial:"), nsVisualEffectMaterialUnderWindowBackground)
	objc.SendVoid(view, objc.Sel("setBlendingMode:"), nsVisualEffectBlendingModeBehindWindow)
	objc.SendVoid(view, objc.Sel("setState:"), nsVisualEffectStateActive)
	return view
}

func blurredViewUpdateLayer(this objc.ID, _ objc.SEL) {
	objc.SendSuper[objc.ID](this, objc.Sel("updateLayer"))
	layer := objc.Send[objc.ID](this, objc.Sel("layer"))
	if layer != objc.Nil {
		removeLayerBackground(layer)
	}
}

func removeLayerBackground(layer objc.ID) {
	objc.SendVoid(layer, objc.Sel("setBackgroundColor:"), objc.Nil)

	className := objc.Send[objc.ID](layer, objc.Sel("className"))
	if objc.Send[bool](className, objc.Sel("isEqualToString:"), nsString("CAChameleonLayer")) {
		objc.SendVoid(layer, objc.Sel("setHidden:"), true)
		return
	}

	filters := objc.Send[objc.ID](layer, objc.Sel("filters"))
	if filters != objc.Nil {
		testString := nsString("Saturat")
		count := objc.Send[objc.NSUInteger](filters, objc.Sel("count"))
		for i := objc.NSUInteger(0); i < count; i++ {
			filter := objc.Send[objc.ID](filters, objc.Sel("objectAtIndex:"), i)
			description := objc.Send[objc.ID](filter, objc.Sel("description"))
			if !objc.Send[bool](description, objc.Sel("containsString:"), testString) {
				continue
			}

			allIndices := objc.MakeNSRange(0, count)
			indices := objc.SendClass[objc.ID](objc.GetClass("NSMutableIndexSet"), objc.Sel("indexSet"))
			objc.SendVoid(indices, objc.Sel("addIndexesInRange:"), allIndices)
			objc.SendVoid(indices, objc.Sel("removeIndex:"), i)
			filtered := objc.Send[objc.ID](filters, objc.Sel("objectsAtIndexes:"), indices)
			objc.SendVoid(layer, objc.Sel("setFilters:"), filtered)
			break
		}
	}

	sublayers := objc.Send[objc.ID](layer, objc.Sel("sublayers"))
	if sublayers != objc.Nil {
		count := objc.Send[objc.NSUInteger](sublayers, objc.Sel("count"))
		for i := objc.NSUInteger(0); i < count; i++ {
			sublayer := objc.Send[objc.ID](sublayers, objc.Sel("objectAtIndex:"), i)
			removeLayerBackground(sublayer)
		}
	}
}

func addTitlebarAccessoryViewController(this objc.ID, _ objc.SEL, viewController objc.ID) {
	objc.SendSuper[objc.ID](this, objc.Sel("addTitlebarAccessoryViewController:"), viewController)

	accessoryView := objc.Send[objc.ID](viewController, objc.Sel("view"))
	objc.SendVoid(accessoryView, objc.Sel("setHidden:"), true)
	frame := objc.Send[objc.CGRect](accessoryView, objc.Sel("frame"))
	frame.Size.Height = 0.0
	objc.SendVoid(accessoryView, objc.Sel("setFrame:"), frame)
}

func runWindowCallback(windowState *lockedWindowState, field *func()) {
	callback := *field
	if callback == nil {
		windowState.Unlock()
		return
	}
	*field = nil
	windowState.Unlock()
	callback()
	windowState.Lock()
	*field = callback
	windowState.Unlock()
}

func moveTabToNewWindow(this objc.ID, _ objc.SEL, _ objc.ID) {
	objc.SendSuper[objc.ID](this, objc.Sel("moveTabToNewWindow:"), objc.Nil)

	windowState := getWindowState(this)
	windowState.Lock()
	runWindowCallback(windowState, &windowState.moveTabToNewWindowCallback)
}

func mergeAllWindows(this objc.ID, _ objc.SEL, _ objc.ID) {
	objc.SendSuper[objc.ID](this, objc.Sel("mergeAllWindows:"), objc.Nil)

	windowState := getWindowState(this)
	windowState.Lock()
	runWindowCallback(windowState, &windowState.mergeAllWindowsCallback)
}

func selectNextTab(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	runWindowCallback(windowState, &windowState.selectNextTabCallback)
}

func selectPreviousTab(this objc.ID, _ objc.SEL, _ objc.ID) {
	windowState := getWindowState(this)
	windowState.Lock()
	runWindowCallback(windowState, &windowState.selectPreviousTabCallback)
}

func toggleTabBar(this objc.ID, _ objc.SEL, _ objc.ID) {
	objc.SendSuper[objc.ID](this, objc.Sel("toggleTabBar:"), objc.Nil)

	windowState := getWindowState(this)
	windowState.Lock()
	windowState.moveTrafficLight()
	runWindowCallback(windowState, &windowState.toggleTabBarCallback)
}
