//go:build windows

package windows

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
	"weak"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/a11y"
	a11ywin "github.com/megascan/gpui-go/a11y/windows"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/scheduler"
)

// WindowsWindow is a native Win32 window. Close is its drop.
type WindowsWindow struct {
	gpui.PlatformWindowDefaults
	inner  *WindowsWindowInner
	closed bool
}

var _ gpui.PlatformWindow = (*WindowsWindow)(nil)

// WindowsWindowState is the mutable state of a window.
type WindowsWindowState struct {
	origin                  gpui.Point[gpui.Pixels]
	logicalSize             gpui.Size[gpui.Pixels]
	minSize                 gpui.Option[gpui.Size[gpui.Pixels]]
	fullscreenRestoreBounds gpui.Bounds[gpui.Pixels]
	borderOffset            WindowBorderOffset
	appearance              gpui.WindowAppearance
	backgroundAppearance    gpui.WindowBackgroundAppearance
	scaleFactor             float32
	restoreFromMinimized    func(gpui.RequestFrameOptions)

	callbacks             Callbacks
	frameSignal           *gpui.PlatformFrameSignal
	inputHandler          *gpui.PlatformInputHandler
	imeEnabled            bool
	pendingSurrogate      gpui.Option[uint16]
	lastReportedModifiers gpui.Option[gpui.Modifiers]
	lastReportedCapslock  gpui.Option[gpui.Capslock]
	hovered               bool
	lastVisibility        gpui.Option[gpui.WindowVisibility]
	directManipulation    *DirectManipulationHandler

	renderer           *DirectXRenderer
	rendererDestroyed  bool
	forceRenderPending bool

	clickState      ClickState
	currentCursor   win32.HCURSOR
	cursorVisible   *atomic.Bool
	ncButtonPressed gpui.Option[uint32]

	display           *WindowsDisplay
	invalidateDevices *atomic.Bool
	drawCoordinator   *DrawCoordinator
	fullscreen        gpui.Option[styleAndBounds]
	initialPlacement  gpui.Option[windowOpenStatus]
	hwnd              HWND
	a11y              a11yCell
	mousePosition     gpui.Point[gpui.Pixels]
	destroyed         bool
	lastPlacement     gpui.Option[win32.WINDOWPLACEMENT]
}

// WindowsWindowInner is the window procedure's view of a window.
type WindowsWindowInner struct {
	hwnd                 HWND
	dialogOwner          *DialogOwner
	dropTargetHelper     *IDropTargetHelper
	state                WindowsWindowState
	systemSettings       *WindowsSystemSettings
	handle               gpui.AnyWindowHandle
	hideTitleBar         bool
	isMovable            bool
	isResizable          bool
	isMinimizable        bool
	executor             gpui.ForegroundExecutor
	validationNumber     uintptr
	mainReceiver         *gpui.PriorityQueueReceiver[gpui.RunnableVariant]
	platformWindowHandle HWND
	parentHwnd           HWND
}

func newWindowsWindowState(
	hwnd HWND,
	directxDevices *DirectXDevices,
	windowParams *win32.CREATESTRUCTW,
	currentCursor win32.HCURSOR,
	cursorVisible *atomic.Bool,
	display *WindowsDisplay,
	minSize gpui.Option[gpui.Size[gpui.Pixels]],
	appearance gpui.WindowAppearance,
	disableDirectComposition bool,
	invalidateDevices *atomic.Bool,
	drawCoordinator *DrawCoordinator,
) (*WindowsWindowState, error) {
	monitorDpi := float32(win32.GetDpiForWindow(hwnd))
	scaleFactor := monitorDpi / float32(win32.USER_DEFAULT_SCREEN_DPI)
	origin := logicalPoint(float32(windowParams.X), float32(windowParams.Y), scaleFactor)
	physicalSize := gpui.NewSize(gpui.DevicePixels(windowParams.Cx), gpui.DevicePixels(windowParams.Cy))
	logicalSize := physicalSize.ToPixels(scaleFactor)
	fullscreenRestoreBounds := gpui.NewBounds(origin, logicalSize)
	renderer, err := NewDirectXRenderer(hwnd, directxDevices, disableDirectComposition)
	if err != nil {
		return nil, fmt.Errorf("Creating DirectX renderer: %w", err)
	}
	clickState := newClickState()

	directManipulation, err := NewDirectManipulationHandler(hwnd, scaleFactor)
	if err != nil {
		renderer.Destroy()
		return nil, fmt.Errorf("initializing Direct Manipulation: %w", err)
	}

	return &WindowsWindowState{
		origin:                  origin,
		logicalSize:             logicalSize,
		fullscreenRestoreBounds: fullscreenRestoreBounds,
		appearance:              appearance,
		backgroundAppearance:    gpui.WindowBackgroundAppearanceOpaque,
		scaleFactor:             scaleFactor,
		minSize:                 minSize,
		frameSignal:             gpui.NewPlatformFrameSignal(),
		imeEnabled:              true,
		renderer:                renderer,
		clickState:              *clickState,
		currentCursor:           currentCursor,
		cursorVisible:           cursorVisible,
		display:                 display,
		hwnd:                    hwnd,
		invalidateDevices:       invalidateDevices,
		drawCoordinator:         drawCoordinator,
		directManipulation:      directManipulation,
	}, nil
}

func (s *WindowsWindowState) isFullscreen() bool {
	return s.fullscreen.IsSome()
}

func (s *WindowsWindowState) isMaximized() bool {
	return !s.isFullscreen() && win32.IsZoomed(s.hwnd)
}

func (s *WindowsWindowState) bounds() gpui.Bounds[gpui.Pixels] {
	return gpui.NewBounds(s.origin, s.logicalSize)
}

func (s *WindowsWindowState) calculateWindowBounds() (gpui.Bounds[gpui.Pixels], bool) {
	placement := s.windowPlacement()
	return calculateClientRect(placement.RcNormalPosition, &s.borderOffset, s.scaleFactor),
		placement.ShowCmd == win32.SW_SHOWMAXIMIZED
}

func (s *WindowsWindowState) windowPlacement() win32.WINDOWPLACEMENT {
	if s.destroyed {
		return s.lastPlacement.UnwrapOrDefault()
	}
	placement := win32.WINDOWPLACEMENT{Length: uint32(unsafe.Sizeof(win32.WINDOWPLACEMENT{}))}
	if err := win32.GetWindowPlacement(s.hwnd, &placement); err != nil {
		gpuiutil.LogErr(fmt.Errorf("failed to get window placement: %w", err))
		return s.lastPlacement.UnwrapOr(placement)
	}
	s.lastPlacement = gpui.Some(placement)
	return placement
}

func (s *WindowsWindowState) windowBounds() gpui.WindowBounds {
	bounds, maximized := s.calculateWindowBounds()

	if s.isFullscreen() {
		return gpui.WindowBoundsFullscreen(s.fullscreenRestoreBounds)
	} else if maximized {
		return gpui.WindowBoundsMaximized(bounds)
	}
	return gpui.WindowBoundsWindowed(bounds)
}

func (s *WindowsWindowState) contentSize() gpui.Size[gpui.Pixels] {
	return s.logicalSize
}

func (w *WindowsWindowInner) visibility() gpui.WindowVisibility {
	isVisible := win32.IsWindowVisible(w.hwnd) && !win32.IsIconic(w.hwnd)
	if isVisible {
		return gpui.WindowVisibilityVisible
	}
	return gpui.WindowVisibilityHidden
}

func newWindowsWindowInner(context *windowCreateContext, hwnd HWND, cs *win32.CREATESTRUCTW) (*WindowsWindowInner, error) {
	state, err := newWindowsWindowState(
		hwnd,
		context.directxDevices,
		cs,
		context.currentCursor,
		context.cursorVisible,
		context.display,
		context.minSize,
		context.appearance,
		context.disableDirectComposition,
		context.invalidateDevices,
		context.drawCoordinator,
	)
	if err != nil {
		return nil, err
	}

	if context.dropTargetHelper != nil {
		context.dropTargetHelper.AddRef()
	}
	return &WindowsWindowInner{
		hwnd:                 hwnd,
		dialogOwner:          NewDialogOwner(hwnd),
		dropTargetHelper:     context.dropTargetHelper,
		state:                *state,
		handle:               context.handle,
		hideTitleBar:         context.hideTitleBar,
		isMovable:            context.isMovable,
		isResizable:          context.isResizable,
		isMinimizable:        context.isMinimizable,
		executor:             context.executor,
		validationNumber:     context.validationNumber,
		mainReceiver:         context.mainReceiver.Clone(),
		platformWindowHandle: context.platformWindowHandle,
		systemSettings:       NewWindowsSystemSettings(),
		parentHwnd:           context.parentHwnd,
	}, nil
}

func (w *WindowsWindowInner) toggleFullscreen() {
	w.executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		var target styleAndBounds
		if state, ok := w.state.fullscreen.Take().Get(); ok {
			target = state
		} else {
			windowBounds, _ := w.state.calculateWindowBounds()
			w.state.fullscreenRestoreBounds = windowBounds

			style := uint32(getWindowLong(w.hwnd, win32.GWL_STYLE))
			var rc win32.RECT
			if err := win32.GetWindowRect(w.hwnd, &rc); err != nil {
				gpuiutil.LogErr(fmt.Errorf("failed to get window rect: %w", err))
			}
			w.state.fullscreen = gpui.Some(styleAndBounds{
				style: style,
				x:     rc.Left,
				y:     rc.Top,
				cx:    rc.Right - rc.Left,
				cy:    rc.Bottom - rc.Top,
			})
			style &^= win32.WS_THICKFRAME | win32.WS_SYSMENU | win32.WS_MAXIMIZEBOX | win32.WS_MINIMIZEBOX | win32.WS_CAPTION
			physicalBounds := w.state.display.PhysicalBounds()
			target = styleAndBounds{
				style: style,
				x:     int32(physicalBounds.Left()),
				y:     int32(physicalBounds.Top()),
				cx:    int32(physicalBounds.Size.Width),
				cy:    int32(physicalBounds.Size.Height),
			}
		}
		setNonRudeHwnd(w.hwnd, !w.state.isFullscreen())
		setWindowLong(w.hwnd, win32.GWL_STYLE, int(target.style))
		logIfErr(win32.SetWindowPos(w.hwnd, 0, target.x, target.y, target.cx, target.cy, win32.SWP_FRAMECHANGED|win32.SWP_NOACTIVATE|win32.SWP_NOZORDER))
		return struct{}{}, nil
	}).Detach()
}

func (w *WindowsWindowInner) setWindowPlacement() error {
	openStatus, ok := w.state.initialPlacement.Take().Get()
	if !ok {
		return nil
	}
	switch openStatus.state {
	case windowOpenStateMaximized:
		if err := win32.SetWindowPlacement(w.hwnd, &openStatus.placement); err != nil {
			return fmt.Errorf("failed to set window placement: %w", err)
		}
		if !win32.ShowWindowAsync(w.hwnd, win32.SW_MAXIMIZE) {
			return xwin.GetLastError()
		}
	case windowOpenStateFullscreen:
		if err := win32.SetWindowPlacement(w.hwnd, &openStatus.placement); err != nil {
			return fmt.Errorf("failed to set window placement: %w", err)
		}
		w.toggleFullscreen()
	case windowOpenStateWindowed:
		if err := win32.SetWindowPlacement(w.hwnd, &openStatus.placement); err != nil {
			return fmt.Errorf("failed to set window placement: %w", err)
		}
	}
	return nil
}

// Callbacks are the window's registered platform callbacks; each is taken while it runs.
type Callbacks struct {
	requestFrame         func(gpui.RequestFrameOptions)
	input                func(gpui.PlatformInput) gpui.DispatchEventResult
	activeStatusChange   func(bool)
	visibilityChange     func(gpui.WindowVisibility)
	hoveredStatusChange  func(bool)
	resize               func(gpui.Size[gpui.Pixels], float32)
	moved                func()
	displayChanged       func()
	shouldClose          func() bool
	close                func()
	hitTestWindowControl func() (gpui.WindowControlArea, bool)
	appearanceChanged    func()
}

type windowCreateContext struct {
	inner                    *WindowsWindowInner
	innerErr                 error
	innerSet                 bool
	handle                   gpui.AnyWindowHandle
	hideTitleBar             bool
	display                  *WindowsDisplay
	isMovable                bool
	isResizable              bool
	isMinimizable            bool
	minSize                  gpui.Option[gpui.Size[gpui.Pixels]]
	executor                 gpui.ForegroundExecutor
	currentCursor            win32.HCURSOR
	cursorVisible            *atomic.Bool
	dropTargetHelper         *IDropTargetHelper
	validationNumber         uintptr
	mainReceiver             *gpui.PriorityQueueReceiver[gpui.RunnableVariant]
	platformWindowHandle     HWND
	appearance               gpui.WindowAppearance
	disableDirectComposition bool
	directxDevices           *DirectXDevices
	invalidateDevices        *atomic.Bool
	drawCoordinator          *DrawCoordinator
	parentHwnd               HWND
}

var (
	createContextsMu   sync.Mutex
	createContexts     = map[uintptr]*windowCreateContext{}
	nextCreateContext  uintptr
	windowRegistryMu   sync.Mutex
	windowRegistry     = map[uintptr]weak.Pointer[WindowsWindowInner]{}
	nextWindowRegistry uintptr
)

// NewWindowsWindow creates a window; a native popup is rejected with gpui.PopupNotSupportedError.
func NewWindowsWindow(handle gpui.AnyWindowHandle, params gpui.WindowParams, creationInfo WindowCreationInfo) (*WindowsWindow, error) {
	if _, ok := params.Kind.(gpui.WindowKindAnchoredPopup); ok {
		return nil, gpui.PopupNotSupportedError{}
	}

	registerWindowClass(creationInfo.icon)
	_, isDialog := params.Kind.(gpui.WindowKindDialog)
	_, isPopUp := params.Kind.(gpui.WindowKindPopUp)
	var parentHwnd HWND
	if isDialog {
		parentWindow := win32.GetActiveWindow()
		if parentWindow != 0 {
			win32.EnableWindow(parentWindow, false)
			parentHwnd = parentWindow
		}
	}
	hideTitleBar := true
	windowName := ""
	if titlebar, ok := params.Titlebar.Get(); ok {
		hideTitleBar = titlebar.AppearsTransparent
		if title, ok := titlebar.Title.Get(); ok {
			windowName = string(title)
		}
	}

	var dwexstyle, dwstyle uint32
	if isPopUp {
		dwexstyle, dwstyle = win32.WS_EX_TOOLWINDOW|win32.WS_EX_TOPMOST, 0
	} else {
		dwstyle = win32.WS_SYSMENU

		if params.IsResizable {
			dwstyle |= win32.WS_THICKFRAME | win32.WS_MAXIMIZEBOX
		}

		if params.IsMinimizable {
			dwstyle |= win32.WS_MINIMIZEBOX
		}
		if isDialog {
			dwstyle |= win32.WS_POPUP | win32.WS_CAPTION
			dwexstyle = win32.WS_EX_DLGMODALFRAME
		} else {
			dwexstyle = win32.WS_EX_APPWINDOW
		}
	}
	if !creationInfo.disableDirectComposition {
		dwexstyle |= win32.WS_EX_NOREDIRECTIONBITMAP
	}

	hinstance := getModuleHandle()
	var display *WindowsDisplay
	ok := false
	if displayID, has := params.DisplayID.Get(); has {
		display, ok = NewWindowsDisplay(displayID)
	}
	if !ok {
		display, ok = WindowsDisplayPrimaryMonitor()
	}
	if !ok {
		return nil, errors.New("failed to find any monitor")
	}
	appearance, err := systemAppearance()
	if err != nil {
		appearance = gpui.WindowAppearanceLight
	}
	context := &windowCreateContext{
		handle:                   handle,
		hideTitleBar:             hideTitleBar,
		display:                  display,
		isMovable:                params.IsMovable,
		isResizable:              params.IsResizable,
		isMinimizable:            params.IsMinimizable,
		minSize:                  params.WindowMinSize,
		executor:                 creationInfo.executor,
		currentCursor:            creationInfo.currentCursor,
		cursorVisible:            creationInfo.cursorVisible,
		dropTargetHelper:         creationInfo.dropTargetHelper,
		validationNumber:         creationInfo.validationNumber,
		mainReceiver:             creationInfo.mainReceiver,
		platformWindowHandle:     creationInfo.platformWindowHandle,
		appearance:               appearance,
		disableDirectComposition: creationInfo.disableDirectComposition,
		directxDevices:           creationInfo.directxDevices,
		invalidateDevices:        creationInfo.invalidateDevices,
		drawCoordinator:          creationInfo.drawCoordinator,
		parentHwnd:               parentHwnd,
	}
	createContextsMu.Lock()
	nextCreateContext++
	contextID := nextCreateContext
	createContexts[contextID] = context
	createContextsMu.Unlock()
	hwnd, creationErr := win32.CreateWindowExW(
		dwexstyle,
		windowClassName,
		wideZ(windowName),
		dwstyle,
		win32.CW_USEDEFAULT,
		win32.CW_USEDEFAULT,
		win32.CW_USEDEFAULT,
		win32.CW_USEDEFAULT,
		parentHwnd,
		0,
		hinstance,
		contextID,
	)
	createContextsMu.Lock()
	delete(createContexts, contextID)
	createContextsMu.Unlock()

	if context.innerSet && context.innerErr != nil {
		return nil, context.innerErr
	}
	if creationErr != nil {
		return nil, creationErr
	}
	this := context.inner

	if err := registerDragDrop(this); err != nil {
		return nil, err
	}
	setNonRudeHwnd(hwnd, true)
	configureDwmDarkMode(hwnd, appearance)
	if err := this.state.borderOffset.update(hwnd); err != nil {
		return nil, err
	}
	placement, err := retrieveWindowPlacement(hwnd, display, params.Bounds, &this.state.borderOffset)
	if err != nil {
		return nil, err
	}
	if params.Show {
		if !params.Focus {
			placement.ShowCmd = win32.SW_SHOWNOACTIVATE
		}
		if err := win32.SetWindowPlacement(hwnd, &placement); err != nil {
			return nil, err
		}
	} else {
		this.state.initialPlacement = gpui.Some(windowOpenStatus{
			placement: placement,
			state:     windowOpenStateWindowed,
		})
	}

	window := &WindowsWindow{inner: this}
	gpui.BindPlatformWindowDefaults(&window.PlatformWindowDefaults, window)
	return window, nil
}

// RawWindowHandle returns the HWND and its module instance.
func (w *WindowsWindow) RawWindowHandle() gpui.RawWindowHandle {
	return gpui.RawWindowHandle{
		Kind:           gpui.RawWindowHandleWin32,
		Win32HWND:      uintptr(w.inner.hwnd),
		Win32HInstance: uintptr(getModuleHandle()),
	}
}

// Close hides the window and destroys it once its dialogs are idle; the renderer is destroyed
// before the HWND. It is idempotent.
func (w *WindowsWindow) Close() {
	if w.closed {
		return
	}
	w.closed = true
	inner := w.inner
	inner.dialogOwner.close()
	if !inner.state.destroyed {
		logLastError(win32.ShowWindowAsync(inner.hwnd, win32.SW_HIDE), "ShowWindowAsync")
	}
	inner.state.callbacks.visibilityChange = nil
	inner.executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		scheduler.Await(s, inner.dialogOwner.whenIdle())
		handle := inner.hwnd
		destroyed := inner.state.destroyed
		if !destroyed {
			logIfErr(win32.RevokeDragDrop(handle))
		}
		inner.state.directManipulation.Close()
		inner.state.rendererDestroyed = true
		inner.state.renderer.Destroy()
		if !destroyed {
			logIfErr(win32.DestroyWindow(handle))
		}
		if inner.dropTargetHelper != nil {
			inner.dropTargetHelper.Release()
		}
		inner.mainReceiver.Close()
		return struct{}{}, nil
	}).Detach()
}

// Bounds returns the window's logical bounds.
func (w *WindowsWindow) Bounds() gpui.Bounds[gpui.Pixels] {
	return w.inner.state.bounds()
}

// IsMaximized reports whether the window is maximized and not fullscreen.
func (w *WindowsWindow) IsMaximized() bool {
	return w.inner.state.isMaximized()
}

// WindowBounds returns the bounds to save and restore the window with.
func (w *WindowsWindow) WindowBounds() gpui.WindowBounds {
	return w.inner.state.windowBounds()
}

// ContentSize returns the logical size of the drawable area.
func (w *WindowsWindow) ContentSize() gpui.Size[gpui.Pixels] {
	return w.inner.state.contentSize()
}

// Resize resizes the client area to size.
func (w *WindowsWindow) Resize(size gpui.Size[gpui.Pixels]) {
	hwnd := w.inner.hwnd
	bounds := gpui.NewBounds(w.Bounds().Origin, size).ToDevicePixels(w.ScaleFactor())
	rect := calculateWindowRect(bounds, &w.inner.state.borderOffset)

	w.inner.executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		if err := win32.SetWindowPos(hwnd, 0, int32(bounds.Origin.X), int32(bounds.Origin.Y), rect.Right-rect.Left, rect.Bottom-rect.Top, win32.SWP_NOMOVE); err != nil {
			gpuiutil.LogErr(fmt.Errorf("unable to set window content size: %w", err))
		}
		return struct{}{}, nil
	}).Detach()
}

// ScaleFactor returns the window's DPI scale.
func (w *WindowsWindow) ScaleFactor() float32 {
	return w.inner.state.scaleFactor
}

// Appearance returns the window's appearance.
func (w *WindowsWindow) Appearance() gpui.WindowAppearance {
	return w.inner.state.appearance
}

// Display returns the display the window is on.
func (w *WindowsWindow) Display() gpui.PlatformDisplay {
	return w.inner.state.display
}

// MousePosition returns the cursor position in window coordinates.
func (w *WindowsWindow) MousePosition() gpui.Point[gpui.Pixels] {
	scaleFactor := w.ScaleFactor()
	var point win32.POINT
	if err := win32.GetCursorPos(&point); err != nil {
		gpuiutil.LogErr(fmt.Errorf("unable to get cursor position: %w", err))
	}
	if w.inner.state.destroyed {
		return w.inner.state.mousePosition
	}
	screenToClient(w.inner.hwnd, &point)
	return logicalPoint(float32(point.X), float32(point.Y), scaleFactor)
}

// Modifiers returns the current keyboard modifiers.
func (w *WindowsWindow) Modifiers() gpui.Modifiers {
	return currentModifiers()
}

// Capslock returns the current caps lock state.
func (w *WindowsWindow) Capslock() gpui.Capslock {
	return currentCapslock()
}

// SetInputHandler installs the text input handler.
func (w *WindowsWindow) SetInputHandler(inputHandler *gpui.PlatformInputHandler) {
	w.inner.state.inputHandler = inputHandler
}

// TakeInputHandler removes and returns the text input handler.
func (w *WindowsWindow) TakeInputHandler() *gpui.PlatformInputHandler {
	h := w.inner.state.inputHandler
	w.inner.state.inputHandler = nil
	return h
}

// Prompt shows a task dialog and resolves to the index of the clicked answer.
func (w *WindowsWindow) Prompt(level gpui.PromptLevel, msg string, detail gpui.Option[string], answers []gpui.PromptButton) *scheduler.OneshotReceiver[int] {
	doneTx, doneRx := scheduler.Oneshot[int]()
	answers = slices.Clone(answers)
	dialog := showDialog(w.inner.dialogOwner, w.inner.executor, func(handle HWND) (gpui.Option[int], error) {
		return showTaskDialog(handle, level, msg, detail, answers)
	})
	w.inner.executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		defer doneTx.Close()
		var clicked gpui.Option[int]
		var dialogErr error
		won := scheduler.Select(s,
			scheduler.Case[scheduler.Result[scheduler.Result[gpui.Option[int]]]](dialog, func(r scheduler.Result[scheduler.Result[gpui.Option[int]]]) {
				if r.Err != nil {
					dialogErr = fmt.Errorf("native dialog thread stopped: %w", r.Err)
					return
				}
				clicked, dialogErr = r.Value.Value, r.Value.Err
			}),
			scheduler.Case(scheduler.PollFn(func(*scheduler.Context) (struct{}, bool) {
				return struct{}{}, doneTx.IsCanceled()
			}), nil),
		)
		if won != 0 {
			return struct{}{}, nil
		}
		if dialogErr != nil {
			gpuiutil.LogErr(dialogErr)
			return struct{}{}, nil
		}
		if index, ok := clicked.Get(); ok {
			_ = doneTx.Send(index)
		}
		return struct{}{}, nil
	}).Detach()

	return doneRx
}

const (
	taskDialogConfigSize = 160
	taskDialogButtonSize = 12
)

var (
	promptTitleInfo     = wideZ("Info")
	promptTitleWarning  = wideZ("Warning")
	promptTitleCritical = wideZ("Critical")
)

func putPtr(b []byte, off int, p unsafe.Pointer) {
	binary.LittleEndian.PutUint64(b[off:], uint64(uintptr(p)))
}

func showTaskDialog(handle HWND, level gpui.PromptLevel, msg string, detail gpui.Option[string], answers []gpui.PromptButton) (gpui.Option[int], error) {
	config := make([]byte, taskDialogConfigSize)
	binary.LittleEndian.PutUint32(config[0:], taskDialogConfigSize)
	binary.LittleEndian.PutUint64(config[4:], uint64(handle))
	binary.LittleEndian.PutUint32(config[20:], win32.TDF_ALLOW_DIALOG_CANCELLATION)
	var title *uint16
	var mainIcon uintptr
	switch level {
	case gpui.PromptLevelInfo:
		title = promptTitleInfo
		mainIcon = win32.TD_INFORMATION_ICON
	case gpui.PromptLevelWarning:
		title = promptTitleWarning
		mainIcon = win32.TD_WARNING_ICON
	case gpui.PromptLevelCritical:
		title = promptTitleCritical
		mainIcon = win32.TD_ERROR_ICON
	}
	putPtr(config, 28, unsafe.Pointer(title))
	binary.LittleEndian.PutUint64(config[36:], uint64(mainIcon))
	instruction := wideZ(msg)
	putPtr(config, 44, unsafe.Pointer(instruction))
	var hintsEncoded *uint16
	if hints, ok := detail.Get(); ok {
		hintsEncoded = wideZ(hints)
		putPtr(config, 52, unsafe.Pointer(hintsEncoded))
	}
	buttonIDMap := make([]int32, 0, len(answers))
	buttons := make([]byte, taskDialogButtonSize*max(len(answers), 1))
	btnEncoded := make([]*uint16, 0, len(answers))
	for index, btn := range answers {
		encoded := wideZ(string(btn.Label()))
		var buttonID int32
		switch btn.Kind() {
		case gpui.PromptButtonKindOk:
			buttonID = win32.IDOK
		case gpui.PromptButtonKindCancel:
			buttonID = win32.IDCANCEL
		default:
			buttonID = -int32(index) - 1
		}
		buttonIDMap = append(buttonIDMap, buttonID)
		binary.LittleEndian.PutUint32(buttons[index*taskDialogButtonSize:], uint32(buttonID))
		putPtr(buttons, index*taskDialogButtonSize+4, unsafe.Pointer(encoded))
		btnEncoded = append(btnEncoded, encoded)
	}
	binary.LittleEndian.PutUint32(config[60:], uint32(len(answers)))
	if len(answers) > 0 {
		putPtr(config, 64, unsafe.Pointer(&buttons[0]))
	}

	binary.LittleEndian.PutUint64(config[140:], uint64(taskDialogCallbackPtr()))
	hasCancel := uint64(0)
	if slices.Contains(buttonIDMap, win32.IDCANCEL) {
		hasCancel = 1
	}
	binary.LittleEndian.PutUint64(config[148:], hasCancel)
	var res int32
	err := win32.TaskDialogIndirect(unsafe.Pointer(&config[0]), &res, nil, nil)
	runtime.KeepAlive(instruction)
	runtime.KeepAlive(hintsEncoded)
	runtime.KeepAlive(btnEncoded)
	runtime.KeepAlive(buttons)
	runtime.KeepAlive(title)
	if err != nil {
		return gpui.None[int](), fmt.Errorf("unable to create task dialog: %w", err)
	}
	if i := slices.Index(buttonIDMap, res); i >= 0 {
		return gpui.Some(i), nil
	}
	return gpui.None[int](), nil
}

// Activate restores, focuses and brings the window to the foreground.
func (w *WindowsWindow) Activate() {
	hwnd := w.inner.hwnd
	this := w.inner
	w.inner.executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		logIfErr(this.setWindowPlacement())

		if win32.IsIconic(hwnd) {
			logLastError(win32.ShowWindowAsync(hwnd, win32.SW_RESTORE), "ShowWindowAsync")
		}

		_, _ = win32.SetActiveWindow(hwnd)
		_, _ = win32.SetFocus(hwnd)

		inputs := make([]win32.INPUT, 2)
		inputs[0].Type = win32.INPUT_KEYBOARD
		inputs[0].Ki().WVk = win32.VK_MENU
		inputs[1].Type = win32.INPUT_KEYBOARD
		inputs[1].Ki().WVk = win32.VK_MENU
		inputs[1].Ki().DwFlags = win32.KEYEVENTF_KEYUP
		win32.SendInput(inputs)

		win32.SetForegroundWindow(hwnd)
		return struct{}{}, nil
	}).Detach()
}

// RequestAttention flashes the window when it is not active.
func (w *WindowsWindow) RequestAttention() {
	if w.IsActive() {
		return
	}

	hwnd := w.inner.hwnd
	w.inner.executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		info := win32.FLASHWINFO{
			CbSize:    uint32(unsafe.Sizeof(win32.FLASHWINFO{})),
			Hwnd:      hwnd,
			DwFlags:   win32.FLASHW_ALL,
			UCount:    1,
			DwTimeout: 0,
		}
		logLastError(win32.FlashWindowEx(&info), "FlashWindowEx")
		return struct{}{}, nil
	}).Detach()
}

// IsActive reports whether the window is the active window.
func (w *WindowsWindow) IsActive() bool {
	return w.inner.hwnd == win32.GetActiveWindow()
}

// Visibility reports whether the window is shown and not minimized.
func (w *WindowsWindow) Visibility() gpui.WindowVisibility {
	return w.inner.visibility()
}

// IsHovered reports whether the cursor is over the window.
func (w *WindowsWindow) IsHovered() bool {
	return w.inner.state.hovered
}

// BackgroundAppearance returns the window's background appearance.
func (w *WindowsWindow) BackgroundAppearance() gpui.WindowBackgroundAppearance {
	return w.inner.state.backgroundAppearance
}

// IsSubpixelRenderingSupported is always true on Windows.
func (w *WindowsWindow) IsSubpixelRenderingSupported() bool {
	return true
}

// SetTitle sets the window title.
func (w *WindowsWindow) SetTitle(title string) {
	if err := win32.SetWindowTextW(w.inner.hwnd, wideZ(title)); err != nil {
		slog.Error(fmt.Sprintf("Set title failed: %v", err))
	}
}

// SetBackgroundAppearance sets the backdrop through DWM or the composition attribute.
func (w *WindowsWindow) SetBackgroundAppearance(backgroundAppearance gpui.WindowBackgroundAppearance) {
	w.inner.state.backgroundAppearance = backgroundAppearance
	hwnd := w.inner.hwnd

	switch backgroundAppearance {
	case gpui.WindowBackgroundAppearanceOpaque:
		setWindowCompositionAttribute(hwnd, gpui.None[color](), 0)
	case gpui.WindowBackgroundAppearanceTransparent:
		setWindowCompositionAttribute(hwnd, gpui.None[color](), 2)
	case gpui.WindowBackgroundAppearanceBlurred:
		setWindowCompositionAttribute(hwnd, gpui.Some(color{}), 4)
	case gpui.WindowBackgroundAppearanceMicaBackdrop:
		dwmSetWindowCompositionAttribute(hwnd, 2)
	case gpui.WindowBackgroundAppearanceMicaAltBackdrop:
		dwmSetWindowCompositionAttribute(hwnd, 4)
	}
}

// Minimize minimizes the window.
func (w *WindowsWindow) Minimize() {
	logLastError(win32.ShowWindowAsync(w.inner.hwnd, win32.SW_MINIMIZE), "ShowWindowAsync")
}

// Zoom maximizes the window, or opens it maximized if it is not shown yet.
func (w *WindowsWindow) Zoom() {
	if win32.IsWindowVisible(w.inner.hwnd) {
		logLastError(win32.ShowWindowAsync(w.inner.hwnd, win32.SW_MAXIMIZE), "ShowWindowAsync")
	} else if status, ok := w.inner.state.initialPlacement.Take().Get(); ok {
		status.state = windowOpenStateMaximized
		w.inner.state.initialPlacement = gpui.Some(status)
	}
}

// ToggleFullscreen toggles fullscreen, or opens the window fullscreen if it is not shown yet.
func (w *WindowsWindow) ToggleFullscreen() {
	if win32.IsWindowVisible(w.inner.hwnd) {
		w.inner.toggleFullscreen()
	} else if status, ok := w.inner.state.initialPlacement.Take().Get(); ok {
		status.state = windowOpenStateFullscreen
		w.inner.state.initialPlacement = gpui.Some(status)
	}
}

// IsFullscreen reports whether the window is fullscreen.
func (w *WindowsWindow) IsFullscreen() bool {
	return w.inner.state.isFullscreen()
}

// OnRequestFrame registers the frame callback.
func (w *WindowsWindow) OnRequestFrame(callback func(options gpui.RequestFrameOptions)) {
	w.inner.state.callbacks.requestFrame = callback
}

// OnInput registers the input callback.
func (w *WindowsWindow) OnInput(callback func(input gpui.PlatformInput) gpui.DispatchEventResult) {
	w.inner.state.callbacks.input = callback
}

// OnActiveStatusChange registers the activation callback.
func (w *WindowsWindow) OnActiveStatusChange(callback func(active bool)) {
	w.inner.state.callbacks.activeStatusChange = callback
}

// OnVisibilityChange registers the visibility callback and records the current visibility.
func (w *WindowsWindow) OnVisibilityChange(callback func(visibility gpui.WindowVisibility)) {
	w.inner.state.lastVisibility = gpui.Some(w.inner.visibility())
	w.inner.state.callbacks.visibilityChange = callback
}

// OnHoverStatusChange registers the hover callback.
func (w *WindowsWindow) OnHoverStatusChange(callback func(hovered bool)) {
	w.inner.state.callbacks.hoveredStatusChange = callback
}

// OnResize registers the resize callback.
func (w *WindowsWindow) OnResize(callback func(size gpui.Size[gpui.Pixels], scaleFactor float32)) {
	w.inner.state.callbacks.resize = callback
}

// OnDisplayChanged registers the display change callback.
func (w *WindowsWindow) OnDisplayChanged(callback func()) {
	w.inner.state.callbacks.displayChanged = callback
}

// OnMoved registers the move callback.
func (w *WindowsWindow) OnMoved(callback func()) {
	w.inner.state.callbacks.moved = callback
}

// OnShouldClose registers the close request callback.
func (w *WindowsWindow) OnShouldClose(callback func() bool) {
	w.inner.state.callbacks.shouldClose = callback
}

// OnClose registers the close callback.
func (w *WindowsWindow) OnClose(callback func()) {
	w.inner.state.callbacks.close = callback
}

// OnHitTestWindowControl registers the window control hit test callback.
func (w *WindowsWindow) OnHitTestWindowControl(callback func() (gpui.WindowControlArea, bool)) {
	w.inner.state.callbacks.hitTestWindowControl = callback
}

// OnAppearanceChanged registers the appearance callback.
func (w *WindowsWindow) OnAppearanceChanged(callback func()) {
	w.inner.state.callbacks.appearanceChanged = callback
}

// Draw renders scene with the window's background appearance.
func (w *WindowsWindow) Draw(scene *gpui.Scene) {
	if w.inner.state.rendererDestroyed {
		return
	}
	logIfErr(w.inner.state.renderer.DrawWithAppearance(scene, w.inner.state.backgroundAppearance))
}

// RenderToImage renders scene into an image.
func (w *WindowsWindow) RenderToImage(scene *gpui.Scene) (*image.RGBA, error) {
	return w.inner.state.renderer.RenderToImage(scene, w.inner.state.backgroundAppearance)
}

// SpriteAtlas returns the renderer's atlas.
func (w *WindowsWindow) SpriteAtlas() gpui.PlatformAtlas {
	return w.inner.state.renderer.SpriteAtlas()
}

// GetRawHandle returns the HWND.
func (w *WindowsWindow) GetRawHandle() uintptr {
	return uintptr(w.inner.hwnd)
}

// GPUSpecs returns the renderer's GPU description.
func (w *WindowsWindow) GPUSpecs() (gpui.GPUSpecs, bool) {
	return w.inner.state.renderer.GPUSpecs()
}

// UpdateIMEPosition moves the IME candidate window next to bounds.
func (w *WindowsWindow) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {
	scaleFactor := w.inner.state.scaleFactor
	caretPosition := win32.POINT{
		X: int32(bounds.Origin.X.AsF32() * scaleFactor),
		Y: int32(bounds.Origin.Y.AsF32()*scaleFactor) + int32(bounds.Size.Height.AsF32()*scaleFactor)/2,
	}

	w.inner.updateImePosition(w.inner.hwnd, caretPosition)
}

// PlaySystemBell plays the default beep.
func (w *WindowsWindow) PlaySystemBell() {
	_ = win32.MessageBeep(win32.MB_OK)
}

// A11yInit creates the window's UI Automation adapter.
func (w *WindowsWindow) A11yInit(callbacks gpui.A11yCallbacks) {
	actionHandler := &A11yActionHandler{callback: callbacks.Action}
	isFocused := win32.GetForegroundWindow() == w.inner.hwnd

	adapter := newA11yAdapter(w.inner.hwnd, isFocused, actionHandler)

	activationHandler := &A11yActivationHandler{callback: callbacks.Activation}

	cell := &w.inner.state.a11y
	if cell.borrowed {
		panic("already borrowed: BorrowMutError")
	}
	cell.value = &A11yState{adapter: adapter, activationHandler: activationHandler}
}

// A11yTreeUpdate applies a tree update and raises its events after the borrow is released.
func (w *WindowsWindow) A11yTreeUpdate(treeUpdate a11y.TreeUpdate) {
	cell := &w.inner.state.a11y
	if cell.borrowed {
		panic("already borrowed: BorrowMutError")
	}
	var events *a11ywin.QueuedEvents
	var ok bool
	if cell.value != nil {
		cell.borrowed = true
		events, ok = cell.value.updateIfActive(func() a11y.TreeUpdate { return treeUpdate })
		cell.borrowed = false
	}
	if ok {
		raiseA11yEvents(events)
	}
}

// A11yUpdateWindowBounds does nothing: UI Automation tracks window bounds itself.
func (w *WindowsWindow) A11yUpdateWindowBounds() {}

// A11yState is a window's accessibility adapter.
type A11yState struct {
	adapter           *a11ywin.Adapter
	activationHandler *A11yActivationHandler
}

// A11yActivationHandler supplies the initial tree when a client activates the adapter.
type A11yActivationHandler struct {
	callback func() (a11y.TreeUpdate, bool)
}

// RequestInitialTree returns the initial tree.
func (h *A11yActivationHandler) RequestInitialTree() (a11y.TreeUpdate, bool) {
	return h.callback()
}

// A11yActionHandler forwards action requests to the window.
type A11yActionHandler struct {
	callback func(a11y.ActionRequest)
}

// DoAction forwards request.
func (h *A11yActionHandler) DoAction(request a11y.ActionRequest) {
	h.callback(request)
}

// IDataObject is the OLE data object interface.
type IDataObject struct {
	com.IUnknown
}

func (d *IDataObject) getData(format *win32.FORMATETC) (win32.STGMEDIUM, error) {
	var medium win32.STGMEDIUM
	err := com.CallHR(d.Ptr(), com.IUnknownSlots+0, uintptr(unsafe.Pointer(format)), uintptr(unsafe.Pointer(&medium)))
	return medium, err
}

func (d *IDataObject) queryGetData(format *win32.FORMATETC) com.HRESULT {
	return com.HRESULTFromUintptr(com.Call(d.Ptr(), com.IUnknownSlots+2, uintptr(unsafe.Pointer(format))))
}

// IDropTargetHelper is the shell's drag image helper (CLSID_DragDropHelper).
type IDropTargetHelper struct {
	com.IUnknown
}

func (h *IDropTargetHelper) dragEnter(hwnd HWND, dataObject *IDataObject, pt *win32.POINT, effect uint32) error {
	return com.CallHR(h.Ptr(), com.IUnknownSlots+0, uintptr(hwnd), uintptr(unsafe.Pointer(dataObject)), uintptr(unsafe.Pointer(pt)), uintptr(effect))
}

func (h *IDropTargetHelper) dragLeave() error {
	return com.CallHR(h.Ptr(), com.IUnknownSlots+1)
}

func (h *IDropTargetHelper) dragOver(pt *win32.POINT, effect uint32) error {
	return com.CallHR(h.Ptr(), com.IUnknownSlots+2, uintptr(unsafe.Pointer(pt)), uintptr(effect))
}

func (h *IDropTargetHelper) drop(dataObject *IDataObject, pt *win32.POINT, effect uint32) error {
	return com.CallHR(h.Ptr(), com.IUnknownSlots+3, uintptr(unsafe.Pointer(dataObject)), uintptr(unsafe.Pointer(pt)), uintptr(effect))
}

var iidIDropTarget = com.MustGUID("{00000122-0000-0000-C000-000000000046}")

type windowsDragDropHandler struct {
	inner *WindowsWindowInner
}

func (h *windowsDragDropHandler) handleDragDrop(input gpui.PlatformInput) {
	if fn := h.inner.state.callbacks.input; fn != nil {
		h.inner.state.callbacks.input = nil
		fn(input)
		h.inner.state.callbacks.input = fn
	}
}

func pointFromPOINTL(pt uintptr) win32.POINT {
	return win32.POINT{X: int32(uint32(pt)), Y: int32(uint32(uint64(pt) >> 32))}
}

func (h *windowsDragDropHandler) dragEnter(idataObj *IDataObject, pt win32.POINT, pdweffect *uint32) com.HRESULT {
	if idataObj == nil {
		return com.E_POINTER
	}
	config := win32.FORMATETC{
		CfFormat: win32.CF_HDROP,
		Ptd:      0,
		DwAspect: win32.DVASPECT_CONTENT,
		Lindex:   -1,
		Tymed:    win32.TYMED_HGLOBAL,
	}
	cursorPosition := pt
	if idataObj.queryGetData(&config) == com.S_OK {
		*pdweffect = win32.DROPEFFECT_COPY
		idata, err := idataObj.getData(&config)
		if err != nil {
			gpuiutil.LogErr(err)
			return com.S_OK
		}
		if idata.Data == 0 {
			return com.S_OK
		}
		hdrop := win32.HDROP(idata.Data)
		var paths []string
		withFileNames(hdrop, func(fileName string) {
			paths = append(paths, fileName)
		})
		win32.ReleaseStgMedium(&idata)
		cursorPosition := cursorPosition
		screenToClient(h.inner.hwnd, &cursorPosition)
		scaleFactor := h.inner.state.scaleFactor
		input := gpui.FileDropEventEntered{
			Position: logicalPoint(float32(cursorPosition.X), float32(cursorPosition.Y), scaleFactor),
			Paths:    gpui.ExternalPaths(paths),
		}
		h.handleDragDrop(input)
	} else {
		*pdweffect = win32.DROPEFFECT_NONE
	}
	if h.inner.dropTargetHelper != nil {
		logIfErr(h.inner.dropTargetHelper.dragEnter(h.inner.hwnd, idataObj, &cursorPosition, *pdweffect))
	}
	return com.S_OK
}

func (h *windowsDragDropHandler) dragOver(pt win32.POINT, pdweffect *uint32) com.HRESULT {
	cursorPosition := pt
	*pdweffect = win32.DROPEFFECT_COPY
	if h.inner.dropTargetHelper != nil {
		logIfErr(h.inner.dropTargetHelper.dragOver(&cursorPosition, *pdweffect))
	}
	screenToClient(h.inner.hwnd, &cursorPosition)
	scaleFactor := h.inner.state.scaleFactor
	input := gpui.FileDropEventPending{
		Position: logicalPoint(float32(cursorPosition.X), float32(cursorPosition.Y), scaleFactor),
	}
	h.handleDragDrop(input)

	return com.S_OK
}

func (h *windowsDragDropHandler) dragLeave() com.HRESULT {
	if h.inner.dropTargetHelper != nil {
		logIfErr(h.inner.dropTargetHelper.dragLeave())
	}
	h.handleDragDrop(gpui.FileDropEventExited{})

	return com.S_OK
}

func (h *windowsDragDropHandler) drop(idataObj *IDataObject, pt win32.POINT, pdweffect *uint32) com.HRESULT {
	if idataObj == nil {
		return com.E_POINTER
	}
	cursorPosition := pt
	*pdweffect = win32.DROPEFFECT_COPY
	if h.inner.dropTargetHelper != nil {
		logIfErr(h.inner.dropTargetHelper.drop(idataObj, &cursorPosition, *pdweffect))
	}
	screenToClient(h.inner.hwnd, &cursorPosition)
	scaleFactor := h.inner.state.scaleFactor
	input := gpui.FileDropEventSubmit{
		Position: logicalPoint(float32(cursorPosition.X), float32(cursorPosition.Y), scaleFactor),
	}
	h.handleDragDrop(input)

	return com.S_OK
}

var windowsDragDropHandlerVtable = com.NewLazyVtable(func() *com.Vtable {
	return com.NewVtable([]com.GUID{iidIDropTarget},
		xwin.NewCallback(func(this *com.IUnknown, dataObj *IDataObject, _ uintptr, pt uintptr, effect *uint32) uintptr {
			impl, ok := com.ImplOf[*windowsDragDropHandler](this)
			if !ok {
				return com.E_FAIL.Uintptr()
			}
			return impl.dragEnter(dataObj, pointFromPOINTL(pt), effect).Uintptr()
		}),
		xwin.NewCallback(func(this *com.IUnknown, _ uintptr, pt uintptr, effect *uint32) uintptr {
			impl, ok := com.ImplOf[*windowsDragDropHandler](this)
			if !ok {
				return com.E_FAIL.Uintptr()
			}
			return impl.dragOver(pointFromPOINTL(pt), effect).Uintptr()
		}),
		xwin.NewCallback(func(this *com.IUnknown) uintptr {
			impl, ok := com.ImplOf[*windowsDragDropHandler](this)
			if !ok {
				return com.E_FAIL.Uintptr()
			}
			return impl.dragLeave().Uintptr()
		}),
		xwin.NewCallback(func(this *com.IUnknown, dataObj *IDataObject, _ uintptr, pt uintptr, effect *uint32) uintptr {
			impl, ok := com.ImplOf[*windowsDragDropHandler](this)
			if !ok {
				return com.E_FAIL.Uintptr()
			}
			return impl.drop(dataObj, pointFromPOINTL(pt), effect).Uintptr()
		}),
	)
})

// ClickState tracks multi-click counting.
type ClickState struct {
	button                            gpui.MouseButton
	lastClick                         time.Time
	lastPosition                      gpui.Point[gpui.DevicePixels]
	doubleClickSpatialToleranceWidth  int32
	doubleClickSpatialToleranceHeight int32
	doubleClickInterval               time.Duration
	currentCount                      int
}

func newClickState() *ClickState {
	doubleClickSpatialToleranceWidth := win32.GetSystemMetrics(win32.SM_CXDOUBLECLK)
	doubleClickSpatialToleranceHeight := win32.GetSystemMetrics(win32.SM_CYDOUBLECLK)
	doubleClickInterval := time.Duration(win32.GetDoubleClickTime()) * time.Millisecond

	return &ClickState{
		button:                            gpui.MouseButtonLeft,
		lastClick:                         time.Now(),
		doubleClickSpatialToleranceWidth:  doubleClickSpatialToleranceWidth,
		doubleClickSpatialToleranceHeight: doubleClickSpatialToleranceHeight,
		doubleClickInterval:               doubleClickInterval,
		currentCount:                      0,
	}
}

func (c *ClickState) update(button gpui.MouseButton, newPosition gpui.Point[gpui.DevicePixels]) int {
	if c.button == button && c.isDoubleClick(newPosition) {
		c.currentCount++
	} else {
		c.currentCount = 1
	}
	c.lastClick = time.Now()
	c.lastPosition = newPosition
	c.button = button

	return c.currentCount
}

func (c *ClickState) systemUpdate(wparam uintptr) {
	switch wparam {
	case 29:
		c.doubleClickSpatialToleranceWidth = win32.GetSystemMetrics(win32.SM_CXDOUBLECLK)
	case 30:
		c.doubleClickSpatialToleranceHeight = win32.GetSystemMetrics(win32.SM_CYDOUBLECLK)
	case 32:
		c.doubleClickInterval = time.Duration(win32.GetDoubleClickTime()) * time.Millisecond
	}
}

func (c *ClickState) isDoubleClick(newPosition gpui.Point[gpui.DevicePixels]) bool {
	diffX := int32(c.lastPosition.X - newPosition.X)
	diffY := int32(c.lastPosition.Y - newPosition.Y)

	return time.Since(c.lastClick) < c.doubleClickInterval &&
		absI32(diffX) <= c.doubleClickSpatialToleranceWidth &&
		absI32(diffY) <= c.doubleClickSpatialToleranceHeight
}

func absI32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

type styleAndBounds struct {
	style uint32
	x     int32
	y     int32
	cx    int32
	cy    int32
}

type windowCompositionAttribData struct {
	attrib uint32
	pvData unsafe.Pointer
	cbData uintptr
}

type accentPolicy struct {
	accentState   uint32
	accentFlags   uint32
	gradientColor uint32
	animationID   uint32
}

type color [4]uint8

// WindowBorderOffset is the size of the non-client frame.
type WindowBorderOffset struct {
	widthOffset  int32
	heightOffset int32
}

func (o *WindowBorderOffset) update(hwnd HWND) error {
	var windowRect win32.RECT
	if err := win32.GetWindowRect(hwnd, &windowRect); err != nil {
		return err
	}
	var clientRect win32.RECT
	if err := win32.GetClientRect(hwnd, &clientRect); err != nil {
		return err
	}
	o.widthOffset = (windowRect.Right - windowRect.Left) - (clientRect.Right - clientRect.Left)
	o.heightOffset = (windowRect.Bottom - windowRect.Top) - (clientRect.Bottom - clientRect.Top)
	return nil
}

type windowOpenStatus struct {
	placement win32.WINDOWPLACEMENT
	state     windowOpenState
}

type windowOpenState uint8

const (
	windowOpenStateMaximized windowOpenState = iota
	windowOpenStateFullscreen
	windowOpenStateWindowed
)

var windowClassName = wideZ("Zed::Window")

var (
	registerWindowClassOnce sync.Once
	windowProcedureCallback uintptr
)

func registerWindowClass(iconHandle win32.HICON) {
	registerWindowClassOnce.Do(func() {
		windowProcedureCallback = xwin.NewCallback(windowProcedure)
		wc := win32.WNDCLASSW{
			LpfnWndProc:   windowProcedureCallback,
			HIcon:         iconHandle,
			LpszClassName: windowClassName,
			Style:         win32.CS_HREDRAW | win32.CS_VREDRAW,
			HInstance:     getModuleHandle(),
			HbrBackground: win32.CreateSolidBrush(0x00000000),
		}
		_, _ = win32.RegisterClassW(&wc)
	})
}

func windowProcedure(hwnd HWND, msg uint32, wparam win32.WPARAM, lparam win32.LPARAM) uintptr {
	if msg == win32.WM_NCCREATE {
		windowParams := lparamPtr[win32.CREATESTRUCTW](lparam)
		createContextsMu.Lock()
		windowCreationContext := createContexts[windowParams.LpCreateParams]
		createContextsMu.Unlock()
		if windowCreationContext == nil {
			return uintptr(win32.DefWindowProcW(hwnd, msg, wparam, lparam))
		}
		windowState, err := newWindowsWindowInner(windowCreationContext, hwnd, windowParams)
		if err != nil {
			windowCreationContext.innerErr = err
			windowCreationContext.innerSet = true
			return 0
		}
		windowRegistryMu.Lock()
		nextWindowRegistry++
		id := nextWindowRegistry
		windowRegistry[id] = weak.Make(windowState)
		windowRegistryMu.Unlock()
		setWindowLong(hwnd, win32.GWLP_USERDATA, int(id))
		windowCreationContext.inner = windowState
		windowCreationContext.innerSet = true
		return uintptr(win32.DefWindowProcW(hwnd, msg, wparam, lparam))
	}

	id := uintptr(getWindowLong(hwnd, win32.GWLP_USERDATA))
	if id == 0 {
		return uintptr(win32.DefWindowProcW(hwnd, msg, wparam, lparam))
	}
	windowRegistryMu.Lock()
	ptr := windowRegistry[id]
	windowRegistryMu.Unlock()
	var result uintptr
	if inner := ptr.Value(); inner != nil {
		if msg == win32.WM_NCDESTROY {
			inner.dialogOwner.close()
		}
		result = uintptr(inner.handleMsg(hwnd, msg, wparam, lparam))
		if msg == win32.WM_NCDESTROY {
			inner.state.destroyed = true
		}
	} else {
		result = uintptr(win32.DefWindowProcW(hwnd, msg, wparam, lparam))
	}

	if msg == win32.WM_NCDESTROY {
		setWindowLong(hwnd, win32.GWLP_USERDATA, 0)
		windowRegistryMu.Lock()
		delete(windowRegistry, id)
		windowRegistryMu.Unlock()
	}

	return result
}

func windowFromHwnd(hwnd HWND) (*WindowsWindowInner, bool) {
	if hwnd == 0 {
		return nil, false
	}

	id := uintptr(getWindowLong(hwnd, win32.GWLP_USERDATA))
	if id == 0 {
		return nil, false
	}
	windowRegistryMu.Lock()
	ptr, ok := windowRegistry[id]
	windowRegistryMu.Unlock()
	if !ok {
		return nil, false
	}
	inner := ptr.Value()
	return inner, inner != nil
}

var moduleAnchor uint16

func getModuleHandle() win32.HMODULE {
	var hModule xwin.Handle
	err := xwin.GetModuleHandleEx(
		xwin.GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS|xwin.GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
		&moduleAnchor,
		&hModule,
	)
	if err != nil {
		panic(fmt.Sprintf("Unable to get module handle: %v", err))
	}

	return hModule
}

func registerDragDrop(window *WindowsWindowInner) error {
	windowHandle := window.hwnd
	handler := &windowsDragDropHandler{inner: window}
	dragDropHandler := com.NewObject(handler, windowsDragDropHandlerVtable.Get())
	defer dragDropHandler.Release()
	if err := win32.RegisterDragDrop(windowHandle, dragDropHandler.Interface(0)); err != nil {
		return fmt.Errorf("unable to register drag-drop event: %w", err)
	}
	return nil
}

func calculateWindowRect(bounds gpui.Bounds[gpui.DevicePixels], borderOffset *WindowBorderOffset) win32.RECT {
	rect := win32.RECT{
		Left:   int32(bounds.Left()),
		Top:    int32(bounds.Top()),
		Right:  int32(bounds.Right()),
		Bottom: int32(bounds.Bottom()),
	}
	leftOffset := borderOffset.widthOffset / 2
	topOffset := borderOffset.heightOffset / 2
	rightOffset := borderOffset.widthOffset - leftOffset
	bottomOffset := borderOffset.heightOffset - topOffset
	rect.Left -= leftOffset
	rect.Top -= topOffset
	rect.Right += rightOffset
	rect.Bottom += bottomOffset
	return rect
}

func calculateClientRect(rect win32.RECT, borderOffset *WindowBorderOffset, scaleFactor float32) gpui.Bounds[gpui.Pixels] {
	leftOffset := borderOffset.widthOffset / 2
	topOffset := borderOffset.heightOffset / 2
	rightOffset := borderOffset.widthOffset - leftOffset
	bottomOffset := borderOffset.heightOffset - topOffset
	left := rect.Left + leftOffset
	top := rect.Top + topOffset
	right := rect.Right - rightOffset
	bottom := rect.Bottom - bottomOffset
	physicalSize := gpui.NewSize(gpui.DevicePixels(right-left), gpui.DevicePixels(bottom-top))
	return gpui.NewBounds(logicalPoint(float32(left), float32(top), scaleFactor), physicalSize.ToPixels(scaleFactor))
}

func retrieveWindowPlacement(hwnd HWND, display *WindowsDisplay, initialBounds gpui.Bounds[gpui.Pixels], borderOffset *WindowBorderOffset) (win32.WINDOWPLACEMENT, error) {
	placement := win32.WINDOWPLACEMENT{Length: uint32(unsafe.Sizeof(win32.WINDOWPLACEMENT{}))}
	if err := win32.GetWindowPlacement(hwnd, &placement); err != nil {
		return placement, err
	}
	bounds := initialBounds
	if !display.CheckGivenBounds(initialBounds) {
		bounds = display.DefaultBounds()
	}
	deviceBounds := bounds.ToDevicePixels(display.ScaleFactor())
	placement.RcNormalPosition = calculateWindowRect(deviceBounds, borderOffset)
	return placement, nil
}

func dwmSetWindowCompositionAttribute(hwnd HWND, backdropType uint32) {
	version := xwin.RtlGetVersion()

	if version.BuildNumber < 22621 {
		return
	}

	_ = win32.DwmSetWindowAttribute(hwnd, win32.DWMWA_SYSTEMBACKDROP_TYPE, unsafe.Pointer(&backdropType), uint32(unsafe.Sizeof(backdropType)))
}

var procSetWindowCompositionAttribute = xwin.NewLazySystemDLL("user32.dll").NewProc("SetWindowCompositionAttribute")

func setWindowCompositionAttribute(hwnd HWND, c gpui.Option[color], state uint32) {
	version := xwin.RtlGetVersion()

	if version.BuildNumber < 17763 {
		return
	}

	if err := procSetWindowCompositionAttribute.Find(); err != nil {
		return
	}
	col := c.UnwrapOrDefault()
	isAcrylic := state == 4
	if isAcrylic && col[3] == 0 {
		col[3] = 1
	}
	accentFlags := uint32(2)
	if isAcrylic {
		accentFlags = 0
	}
	accent := accentPolicy{
		accentState:   state,
		accentFlags:   accentFlags,
		gradientColor: uint32(col[0]) | uint32(col[1])<<8 | uint32(col[2])<<16 | uint32(col[3])<<24,
		animationID:   0,
	}
	data := windowCompositionAttribData{
		attrib: 0x13,
		pvData: unsafe.Pointer(&accent),
		cbData: unsafe.Sizeof(accentPolicy{}),
	}
	_, _, _ = procSetWindowCompositionAttribute.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&data)))
}

var nonRudeHwndProp = wideZ("NonRudeHWND")

func setNonRudeHwnd(hwnd HWND, nonRude bool) {
	if nonRude {
		logIfErr(win32.SetPropW(hwnd, nonRudeHwndProp, 1))
	} else {
		_, err := win32.RemovePropW(hwnd, nonRudeHwndProp)
		logIfErr(err)
	}
}
