//go:build windows

package windows

import (
	"fmt"
	"log/slog"
	"math"
	"slices"
	"unicode"
	"unicode/utf16"
	"unsafe"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/scheduler"
)

// Window messages gpui posts to its own windows.
const (
	WM_GPUI_CURSOR_STYLE_CHANGED           uint32 = win32.WM_USER + 1
	WM_GPUI_CLOSE_ONE_WINDOW               uint32 = win32.WM_USER + 2
	WM_GPUI_TASK_DISPATCHED_ON_MAIN_THREAD uint32 = win32.WM_USER + 3
	WM_GPUI_DOCK_MENU_ACTION               uint32 = win32.WM_USER + 4
	WM_GPUI_FORCE_UPDATE_WINDOW            uint32 = win32.WM_USER + 5
	WM_GPUI_KEYBOARD_LAYOUT_CHANGED        uint32 = win32.WM_USER + 6
	WM_GPUI_GPU_DEVICE_LOST                uint32 = win32.WM_USER + 7
	WM_GPUI_DISPLAYS_CHANGED               uint32 = win32.WM_USER + 10
)

const sizeMoveLoopTimerID uintptr = 1

const (
	hoverDefault uint32 = 0xFFFFFFFF
	attrInput    byte   = 0x00
)

// DrawCoordinator coordinates window draws on the UI thread; it is shared by every window so a
// re-entrant paint request for one window while another draws is deferred.
type DrawCoordinator struct {
	drawing bool
}

// NewDrawCoordinator returns a coordinator with no draw in progress.
func NewDrawCoordinator() *DrawCoordinator {
	return &DrawCoordinator{}
}

func (c *DrawCoordinator) tryBeginDraw() (*drawWindowGuard, bool) {
	if c.drawing {
		return nil, false
	}
	c.drawing = true
	return &drawWindowGuard{coordinator: c}, true
}

type drawWindowGuard struct {
	coordinator *DrawCoordinator
}

func (g *drawWindowGuard) Release() {
	g.coordinator.drawing = false
}

type msgResult struct {
	value uintptr
	ok    bool
}

func handled(n int) msgResult {
	return msgResult{value: uintptr(n), ok: true}
}

var unhandled = msgResult{}

func lparamPtr[T any](l win32.LPARAM) *T {
	return *(**T)(unsafe.Pointer(&l))
}

func logLastError(ok bool, what string) {
	if !ok {
		gpuiutil.LogErr(lastErrorFor(what, xwin.GetLastError()))
	}
}

func lastErrorFor(what string, err error) error {
	if err == nil {
		return fmt.Errorf("%s failed", what)
	}
	return fmt.Errorf("%s: %w", what, err)
}

func logIfErr(err error) {
	if err != nil {
		gpuiutil.LogErr(err)
	}
}

func screenToClient(handle HWND, point *win32.POINT) {
	logLastError(win32.ScreenToClient(handle, point), "ScreenToClient")
}

func (w *WindowsWindowInner) handleMsg(handle HWND, msg uint32, wparam win32.WPARAM, lparam win32.LPARAM) win32.LRESULT {
	var result msgResult
	switch msg {
	case win32.WM_MOUSEACTIVATE:
		result = handled(win32.MA_ACTIVATE)
	case win32.WM_ACTIVATE:
		result = w.handleActivateMsg(wparam)
	case win32.WM_CREATE:
		result = w.handleCreateMsg(handle)
	case win32.WM_MOVE:
		result = w.handleMoveMsg(handle, lparam)
	case win32.WM_SIZE:
		result = w.handleSizeMsg(wparam, lparam)
	case win32.WM_GETMINMAXINFO:
		result = w.handleGetMinMaxInfoMsg(lparam)
	case win32.WM_ENTERSIZEMOVE, win32.WM_ENTERMENULOOP:
		result = w.handleSizeMoveLoop(handle)
	case win32.WM_EXITSIZEMOVE, win32.WM_EXITMENULOOP:
		result = w.handleSizeMoveLoopExit(handle)
	case win32.WM_TIMER:
		result = w.handleTimerMsg(handle, wparam)
	case win32.WM_NCCALCSIZE:
		result = w.handleCalcClientSize(handle, wparam, lparam)
	case win32.WM_DPICHANGED:
		result = w.handleDpiChangedMsg(handle, wparam, lparam)
	case win32.WM_DISPLAYCHANGE:
		result = w.handleDisplayChangeMsg(handle)
	case win32.WM_NCHITTEST:
		result = w.handleHitTestMsg(handle, lparam)
	case win32.WM_PAINT:
		result = w.handlePaintMsg(handle)
	case win32.WM_CLOSE:
		result = w.handleCloseMsg()
	case win32.WM_DESTROY:
		result = w.handleDestroyMsg(handle)
	case win32.WM_MOUSEMOVE:
		result = w.handleMouseMoveMsg(handle, lparam, wparam)
	case win32.WM_MOUSELEAVE, win32.WM_NCMOUSELEAVE:
		result = w.handleMouseLeaveMsg()
	case win32.WM_NCMOUSEMOVE:
		result = w.handleNcMouseMoveMsg(handle, lparam)
	case win32.WM_NCLBUTTONDBLCLK, win32.WM_NCLBUTTONDOWN:
		result = w.handleNcMouseDownMsg(handle, gpui.MouseButtonLeft, wparam, lparam)
	case win32.WM_NCRBUTTONDOWN:
		result = w.handleNcMouseDownMsg(handle, gpui.MouseButtonRight, wparam, lparam)
	case win32.WM_NCMBUTTONDOWN:
		result = w.handleNcMouseDownMsg(handle, gpui.MouseButtonMiddle, wparam, lparam)
	case win32.WM_NCLBUTTONUP:
		result = w.handleNcMouseUpMsg(handle, gpui.MouseButtonLeft, wparam, lparam)
	case win32.WM_NCRBUTTONUP:
		result = w.handleNcMouseUpMsg(handle, gpui.MouseButtonRight, wparam, lparam)
	case win32.WM_NCMBUTTONUP:
		result = w.handleNcMouseUpMsg(handle, gpui.MouseButtonMiddle, wparam, lparam)
	case win32.WM_LBUTTONDOWN:
		result = w.handleMouseDownMsg(handle, gpui.MouseButtonLeft, lparam)
	case win32.WM_RBUTTONDOWN:
		result = w.handleMouseDownMsg(handle, gpui.MouseButtonRight, lparam)
	case win32.WM_MBUTTONDOWN:
		result = w.handleMouseDownMsg(handle, gpui.MouseButtonMiddle, lparam)
	case win32.WM_XBUTTONDOWN:
		result = w.handleXbuttonMsg(handle, wparam, lparam, (*WindowsWindowInner).handleMouseDownMsg)
	case win32.WM_LBUTTONUP:
		result = w.handleMouseUpMsg(handle, gpui.MouseButtonLeft, lparam)
	case win32.WM_RBUTTONUP:
		result = w.handleMouseUpMsg(handle, gpui.MouseButtonRight, lparam)
	case win32.WM_MBUTTONUP:
		result = w.handleMouseUpMsg(handle, gpui.MouseButtonMiddle, lparam)
	case win32.WM_XBUTTONUP:
		result = w.handleXbuttonMsg(handle, wparam, lparam, (*WindowsWindowInner).handleMouseUpMsg)
	case win32.WM_MOUSEWHEEL:
		result = w.handleMouseWheelMsg(handle, wparam, lparam)
	case win32.WM_MOUSEHWHEEL:
		result = w.handleMouseHorizontalWheelMsg(handle, wparam, lparam)
	case win32.WM_SYSKEYUP:
		result = w.handleSyskeyupMsg(wparam, lparam)
	case win32.WM_KEYUP:
		result = w.handleKeyupMsg(wparam, lparam)
	case win32.WM_CHAR:
		result = w.handleCharMsg(wparam)
	case win32.WM_IME_STARTCOMPOSITION:
		result = w.handleImePosition(handle)
	case win32.WM_IME_COMPOSITION:
		result = w.handleImeComposition(handle, lparam)
	case win32.WM_SETCURSOR:
		result = w.handleSetCursor(handle, lparam)
	case win32.WM_SETTINGCHANGE:
		result = w.handleSystemSettingsChanged(handle, wparam, lparam)
	case win32.WM_INPUTLANGCHANGE:
		result = w.handleInputLanguageChanged()
	case win32.WM_SHOWWINDOW:
		result = w.handleWindowVisibilityChanged(handle, wparam)
	case win32.DM_POINTERHITTEST:
		result = w.handleDmPointerHitTest(wparam)
	case win32.WM_GETOBJECT:
		result = w.handleWmGetobject(wparam, lparam)
	case WM_GPUI_CURSOR_STYLE_CHANGED, WM_GPUI_FORCE_UPDATE_WINDOW, WM_GPUI_GPU_DEVICE_LOST:
		result = w.handleGpuiEvents(handle, msg, wparam, lparam)
	}
	if result.ok {
		return win32.LRESULT(result.value)
	}
	return win32.DefWindowProcW(handle, msg, wparam, lparam)
}

func (w *WindowsWindowInner) handleMoveMsg(handle HWND, lparam win32.LPARAM) msgResult {
	origin := logicalPoint(float32(signedLoword(lparam)), float32(signedHiword(lparam)), w.state.scaleFactor)
	w.state.origin = origin
	size := w.state.logicalSize
	centerX := origin.X.AsF32() + size.Width.AsF32()/2
	centerY := origin.Y.AsF32() + size.Height.AsF32()/2
	monitorBounds := w.state.display.Bounds()
	if centerX < monitorBounds.Left().AsF32() ||
		centerX > monitorBounds.Right().AsF32() ||
		centerY < monitorBounds.Top().AsF32() ||
		centerY > monitorBounds.Bottom().AsF32() {
		monitor := win32.MonitorFromWindow(handle, win32.MONITOR_DEFAULTTONULL)
		if monitor != 0 && w.state.display.Handle != monitor {
			display, ok := NewWindowsDisplay(displayIDForMonitor(monitor))
			if !ok {
				return unhandled
			}
			w.setDisplay(display)
		}
	}
	if callback := w.state.callbacks.moved; callback != nil {
		w.state.callbacks.moved = nil
		callback()
		w.state.callbacks.moved = callback
	}
	return handled(0)
}

func (w *WindowsWindowInner) handleGetMinMaxInfoMsg(lparam win32.LPARAM) msgResult {
	minSize, ok := w.state.minSize.Get()
	if !ok {
		return unhandled
	}
	scaleFactor := w.state.scaleFactor
	boarderOffset := &w.state.borderOffset

	minmaxInfo := lparamPtr[win32.MINMAXINFO](lparam)
	minmaxInfo.PtMinTrackSize.X = int32(minSize.Width.Scale(scaleFactor).AsF32()) + boarderOffset.widthOffset
	minmaxInfo.PtMinTrackSize.Y = int32(minSize.Height.Scale(scaleFactor).AsF32()) + boarderOffset.heightOffset
	return handled(0)
}

func (w *WindowsWindowInner) handleSizeMsg(wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	w.reportVisibility()

	if wparam == win32.SIZE_MINIMIZED {
		w.state.restoreFromMinimized = w.state.callbacks.requestFrame
		w.state.callbacks.requestFrame = nil
		return handled(0)
	}

	width := int32(max(loword(lparam), 1))
	height := int32(max(hiword(lparam), 1))
	newSize := gpui.Size[gpui.DevicePixels]{Width: gpui.DevicePixels(width), Height: gpui.DevicePixels(height)}

	scaleFactor := w.state.scaleFactor
	shouldResizeRenderer := false
	if restoreFromMinimized := w.state.restoreFromMinimized; restoreFromMinimized != nil {
		w.state.restoreFromMinimized = nil
		w.state.callbacks.requestFrame = restoreFromMinimized
	} else {
		shouldResizeRenderer = true
	}

	w.handleSizeChange(newSize, scaleFactor, shouldResizeRenderer)
	return handled(0)
}

func (w *WindowsWindowInner) handleSizeChange(deviceSize gpui.Size[gpui.DevicePixels], scaleFactor float32, shouldResizeRenderer bool) {
	newLogicalSize := deviceSize.ToPixels(scaleFactor)

	w.state.logicalSize = newLogicalSize
	if shouldResizeRenderer {
		if err := w.state.renderer.Resize(deviceSize); err != nil {
			slog.Error(fmt.Sprintf("Failed to resize renderer, invalidating devices: %v", err))
			w.state.invalidateDevices.Store(true)
		}
	}
	if callback := w.state.callbacks.resize; callback != nil {
		w.state.callbacks.resize = nil
		callback(newLogicalSize, scaleFactor)
		w.state.callbacks.resize = callback
	}
}

func (w *WindowsWindowInner) handleSizeMoveLoop(handle HWND) msgResult {
	if ret := win32.SetTimer(handle, sizeMoveLoopTimerID, win32.USER_TIMER_MINIMUM, 0); ret == 0 {
		slog.Error(fmt.Sprintf("unable to create timer: %v", xwin.GetLastError()))
	}
	return unhandled
}

func (w *WindowsWindowInner) handleSizeMoveLoopExit(handle HWND) msgResult {
	logIfErr(win32.KillTimer(handle, sizeMoveLoopTimerID))
	return unhandled
}

func (w *WindowsWindowInner) handleTimerMsg(handle HWND, wparam win32.WPARAM) msgResult {
	if uintptr(wparam) == sizeMoveLoopTimerID {
		for runnable, err := range w.mainReceiver.Clone().TryIter() {
			if err != nil {
				break
			}
			WindowsDispatcherExecuteRunnable(runnable)
		}
		return w.handlePaintMsg(handle)
	}
	return unhandled
}

func (w *WindowsWindowInner) handlePaintMsg(handle HWND) msgResult {
	return w.drawWindow(handle, false)
}

func (w *WindowsWindowInner) handleCloseMsg() msgResult {
	callback := w.state.callbacks.shouldClose
	if callback == nil {
		return unhandled
	}
	w.state.callbacks.shouldClose = nil
	shouldClose := callback()
	w.state.callbacks.shouldClose = callback
	if shouldClose {
		return unhandled
	}
	return handled(0)
}

func (w *WindowsWindowInner) handleDestroyMsg(handle HWND) msgResult {
	w.state.windowPlacement()
	callback := w.state.callbacks.close
	w.state.callbacks.close = nil
	if w.parentHwnd != 0 {
		win32.EnableWindow(w.parentHwnd, true)
		win32.SetForegroundWindow(w.parentHwnd)
	}

	if callback != nil {
		callback()
	}
	if cell := &w.state.a11y; !cell.borrowed && cell.value != nil {
		cell.value.release()
		cell.value = nil
	}
	logIfErr(win32.PostMessageW(w.platformWindowHandle, WM_GPUI_CLOSE_ONE_WINDOW, win32.WPARAM(w.validationNumber), win32.LPARAM(handle)))
	return handled(0)
}

func (w *WindowsWindowInner) handleMouseMoveMsg(handle HWND, lparam win32.LPARAM, wparam win32.WPARAM) msgResult {
	w.startTrackingMouse(handle, win32.TME_LEAVE)
	w.restoreCursorAfterHide()

	x := float32(signedLoword(lparam))
	y := float32(signedHiword(lparam))
	scaleFactor := w.state.scaleFactor
	newPosition := logicalPoint(x, y, scaleFactor)

	var pressedButton gpui.Option[gpui.MouseButton]
	flags := uint32(loword(wparam))
	switch {
	case flags&win32.MK_LBUTTON != 0:
		pressedButton = gpui.Some(gpui.MouseButtonLeft)
	case flags&win32.MK_RBUTTON != 0:
		pressedButton = gpui.Some(gpui.MouseButtonRight)
	case flags&win32.MK_MBUTTON != 0:
		pressedButton = gpui.Some(gpui.MouseButtonMiddle)
	case flags&win32.MK_XBUTTON1 != 0:
		pressedButton = gpui.Some(gpui.MouseButtonNavigate(gpui.NavigationDirectionBack))
	case flags&win32.MK_XBUTTON2 != 0:
		pressedButton = gpui.Some(gpui.MouseButtonNavigate(gpui.NavigationDirectionForward))
	}

	if pressedButton.IsNone() && newPosition == w.state.mousePosition {
		return handled(1)
	}
	w.state.mousePosition = newPosition

	fn := w.state.callbacks.input
	if fn == nil {
		return handled(1)
	}
	w.state.callbacks.input = nil

	input := gpui.MouseMoveEvent{
		Position:      newPosition,
		PressedButton: pressedButton,
		Modifiers:     currentModifiers(),
	}
	isHandled := !fn(input).Propagate
	w.state.callbacks.input = fn
	if isHandled {
		return handled(0)
	}
	return handled(1)
}

func (w *WindowsWindowInner) handleMouseLeaveMsg() msgResult {
	w.state.hovered = false
	w.state.cursorVisible.Store(true)
	if callback := w.state.callbacks.hoveredStatusChange; callback != nil {
		w.state.callbacks.hoveredStatusChange = nil
		callback(false)
		w.state.callbacks.hoveredStatusChange = callback
	}

	return handled(0)
}

func (w *WindowsWindowInner) handleSyskeyupMsg(wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	input, ok := handleKeyEvent(wparam, lparam, &w.state, func(keystroke gpui.Keystroke, _ bool) gpui.PlatformInput {
		return gpui.KeyUpEvent{Keystroke: keystroke}
	})
	if !ok {
		return unhandled
	}
	fn := w.state.callbacks.input
	if fn == nil {
		return unhandled
	}
	w.state.callbacks.input = nil

	fn(input)
	w.state.callbacks.input = fn

	return handled(0)
}

func (w *WindowsWindowInner) handleKeydownMsg(wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	input, ok := handleKeyEvent(wparam, lparam, &w.state, func(keystroke gpui.Keystroke, preferCharacterInput bool) gpui.PlatformInput {
		return gpui.KeyDownEvent{
			Keystroke:            keystroke,
			IsHeld:               lparam&(0x1<<30) > 0,
			PreferCharacterInput: preferCharacterInput,
		}
	})
	if !ok {
		return handled(1)
	}

	fn := w.state.callbacks.input
	if fn == nil {
		return handled(1)
	}
	w.state.callbacks.input = nil

	isHandled := !fn(input).Propagate

	w.state.callbacks.input = fn

	if isHandled {
		return handled(0)
	}
	return handled(1)
}

func (w *WindowsWindowInner) handleKeyupMsg(wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	input, ok := handleKeyEvent(wparam, lparam, &w.state, func(keystroke gpui.Keystroke, _ bool) gpui.PlatformInput {
		return gpui.KeyUpEvent{Keystroke: keystroke}
	})
	if !ok {
		return handled(1)
	}

	fn := w.state.callbacks.input
	if fn == nil {
		return handled(1)
	}
	w.state.callbacks.input = nil

	isHandled := !fn(input).Propagate
	w.state.callbacks.input = fn

	if isHandled {
		return handled(0)
	}
	return handled(1)
}

func (w *WindowsWindowInner) handleCharMsg(wparam win32.WPARAM) msgResult {
	input, ok := w.parseCharMessage(wparam)
	if !ok {
		return unhandled
	}
	w.withInputHandler(func(inputHandler *gpui.PlatformInputHandler) {
		inputHandler.ReplaceTextInRange(gpui.None[gpui.Range](), input)
	})

	return handled(0)
}

func (w *WindowsWindowInner) handleMouseDownMsg(handle HWND, button gpui.MouseButton, lparam win32.LPARAM) msgResult {
	win32.SetCapture(handle)

	fn := w.state.callbacks.input
	if fn == nil {
		return handled(1)
	}
	w.state.callbacks.input = nil
	x := signedLoword(lparam)
	y := signedHiword(lparam)
	physicalPoint := gpui.Point[gpui.DevicePixels]{X: gpui.DevicePixels(int32(x)), Y: gpui.DevicePixels(int32(y))}
	clickCount := w.state.clickState.update(button, physicalPoint)
	scaleFactor := w.state.scaleFactor

	input := gpui.MouseDownEvent{
		Button:     button,
		Position:   logicalPoint(float32(x), float32(y), scaleFactor),
		Modifiers:  currentModifiers(),
		ClickCount: clickCount,
		FirstMouse: false,
	}
	isHandled := !fn(input).Propagate
	w.state.callbacks.input = fn

	if isHandled {
		return handled(0)
	}
	return handled(1)
}

func (w *WindowsWindowInner) handleMouseUpMsg(_ HWND, button gpui.MouseButton, lparam win32.LPARAM) msgResult {
	logIfErr(win32.ReleaseCapture())

	fn := w.state.callbacks.input
	if fn == nil {
		return handled(1)
	}
	w.state.callbacks.input = nil
	x := float32(signedLoword(lparam))
	y := float32(signedHiword(lparam))
	clickCount := w.state.clickState.currentCount
	scaleFactor := w.state.scaleFactor

	input := gpui.MouseUpEvent{
		Button:     button,
		Position:   logicalPoint(x, y, scaleFactor),
		Modifiers:  currentModifiers(),
		ClickCount: clickCount,
	}
	isHandled := !fn(input).Propagate
	w.state.callbacks.input = fn

	if isHandled {
		return handled(0)
	}
	return handled(1)
}

func (w *WindowsWindowInner) handleXbuttonMsg(handle HWND, wparam win32.WPARAM, lparam win32.LPARAM, handler func(*WindowsWindowInner, HWND, gpui.MouseButton, win32.LPARAM) msgResult) msgResult {
	var navDir gpui.NavigationDirection
	switch hiword(wparam) {
	case win32.XBUTTON1:
		navDir = gpui.NavigationDirectionBack
	case win32.XBUTTON2:
		navDir = gpui.NavigationDirectionForward
	default:
		return handled(1)
	}
	return handler(w, handle, gpui.MouseButtonNavigate(navDir), lparam)
}

func (w *WindowsWindowInner) handleMouseWheelMsg(handle HWND, wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	modifiers := currentModifiers()

	fn := w.state.callbacks.input
	if fn == nil {
		return handled(1)
	}
	w.state.callbacks.input = nil
	scaleFactor := w.state.scaleFactor
	var wheelScrollAmount uint32
	if modifiers.Shift {
		wheelScrollAmount = w.systemSettings.MouseWheelSettings.WheelScrollChars
	} else {
		wheelScrollAmount = w.systemSettings.MouseWheelSettings.WheelScrollLines
	}

	wheelDistance := (float32(signedHiword(wparam)) / float32(win32.WHEEL_DELTA)) * float32(wheelScrollAmount)
	cursorPoint := win32.POINT{X: int32(signedLoword(lparam)), Y: int32(signedHiword(lparam))}
	screenToClient(handle, &cursorPoint)
	var delta gpui.Point[float32]
	if modifiers.Shift {
		delta = gpui.Point[float32]{X: wheelDistance, Y: 0}
	} else {
		delta = gpui.Point[float32]{X: 0, Y: wheelDistance}
	}
	input := gpui.ScrollEvent{
		Position:   logicalPoint(float32(cursorPoint.X), float32(cursorPoint.Y), scaleFactor),
		Delta:      gpui.ScrollDeltaLines(delta),
		Modifiers:  modifiers,
		TouchPhase: gpui.TouchPhaseMoved,
		Source:     gpui.ScrollSourcePointer,
	}
	isHandled := !fn(input).Propagate
	w.state.callbacks.input = fn

	if isHandled {
		return handled(0)
	}
	return handled(1)
}

func (w *WindowsWindowInner) handleMouseHorizontalWheelMsg(handle HWND, wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	fn := w.state.callbacks.input
	if fn == nil {
		return handled(1)
	}
	w.state.callbacks.input = nil
	scaleFactor := w.state.scaleFactor
	wheelScrollChars := w.systemSettings.MouseWheelSettings.WheelScrollChars

	wheelDistance := (float32(-signedHiword(wparam)) / float32(win32.WHEEL_DELTA)) * float32(wheelScrollChars)
	cursorPoint := win32.POINT{X: int32(signedLoword(lparam)), Y: int32(signedHiword(lparam))}
	screenToClient(handle, &cursorPoint)
	event := gpui.ScrollEvent{
		Position:   logicalPoint(float32(cursorPoint.X), float32(cursorPoint.Y), scaleFactor),
		Delta:      gpui.ScrollDeltaLines(gpui.Point[float32]{X: wheelDistance, Y: 0}),
		Modifiers:  currentModifiers(),
		TouchPhase: gpui.TouchPhaseMoved,
		Source:     gpui.ScrollSourcePointer,
	}
	isHandled := !fn(event).Propagate
	w.state.callbacks.input = fn

	if isHandled {
		return handled(0)
	}
	return handled(1)
}

func (w *WindowsWindowInner) retrieveCaretPosition() (win32.POINT, bool) {
	return withInputHandlerAndScaleFactor(w, func(inputHandler *gpui.PlatformInputHandler, scaleFactor float32) (win32.POINT, bool) {
		caretRange, ok := inputHandler.SelectedTextRange(false)
		if !ok {
			return win32.POINT{}, false
		}
		caretPosition, ok := inputHandler.BoundsForRange(caretRange.Range)
		if !ok {
			return win32.POINT{}, false
		}
		return win32.POINT{
			X: int32(caretPosition.Origin.X.AsF32() * scaleFactor),
			Y: int32(caretPosition.Origin.Y.AsF32()*scaleFactor) + (int32(caretPosition.Size.Height.AsF32()*scaleFactor) / 2),
		}, true
	})
}

func (w *WindowsWindowInner) handleImePosition(handle HWND) msgResult {
	if caretPosition, ok := w.retrieveCaretPosition(); ok {
		w.updateImePosition(handle, caretPosition)
	}
	return handled(0)
}

func (w *WindowsWindowInner) updateImePosition(handle HWND, caretPosition win32.POINT) {
	ctx, ok := getImeContext(handle)
	if !ok {
		return
	}
	defer ctx.Release()
	logLastError(win32.ImmSetCompositionWindow(ctx.himc, &win32.COMPOSITIONFORM{
		DwStyle:      win32.CFS_POINT,
		PtCurrentPos: caretPosition,
	}), "ImmSetCompositionWindow")

	logLastError(win32.ImmSetCandidateWindow(ctx.himc, &win32.CANDIDATEFORM{
		DwStyle:      win32.CFS_CANDIDATEPOS,
		PtCurrentPos: caretPosition,
	}), "ImmSetCandidateWindow")
}

func (w *WindowsWindowInner) updateImeEnabled(handle HWND) {
	imeEnabled, _ := withInputHandlerR(w, func(inputHandler *gpui.PlatformInputHandler) bool {
		return inputHandler.QueryAcceptsTextInput()
	})
	if imeEnabled == w.state.imeEnabled {
		return
	}
	w.state.imeEnabled = imeEnabled
	if imeEnabled {
		logLastError(win32.ImmAssociateContextEx(handle, 0, win32.IACE_DEFAULT), "ImmAssociateContextEx")
	} else {
		if win32.GetFocus() == handle {
			if ctx, ok := getImeContext(handle); ok {
				logLastError(win32.ImmNotifyIME(ctx.himc, win32.NI_COMPOSITIONSTR, win32.CPS_COMPLETE, 0), "ImmNotifyIME")
				ctx.Release()
			}
		}
		logLastError(win32.ImmAssociateContextEx(handle, 0, 0), "ImmAssociateContextEx")
	}
}

func (w *WindowsWindowInner) handleImeComposition(handle HWND, lparam win32.LPARAM) msgResult {
	ctx, ok := getImeContext(handle)
	if !ok {
		return unhandled
	}
	defer ctx.Release()
	return w.handleImeCompositionInner(ctx.himc, lparam)
}

func (w *WindowsWindowInner) handleImeCompositionInner(ctx win32.HIMC, lparamValue win32.LPARAM) msgResult {
	lparam := uint32(lparamValue)
	if lparam == 0 {
		if !w.withInputHandler(func(inputHandler *gpui.PlatformInputHandler) {
			inputHandler.ReplaceTextInRange(gpui.None[gpui.Range](), "")
		}) {
			return unhandled
		}
		return handled(0)
	}
	if lparam&win32.GCS_RESULTSTR > 0 {
		compResult, ok := parseImeCompositionString(ctx, win32.GCS_RESULTSTR)
		if !ok {
			return unhandled
		}
		if !w.withInputHandler(func(inputHandler *gpui.PlatformInputHandler) {
			inputHandler.ReplaceTextInRange(gpui.None[gpui.Range](), stringFromUTF16Lossy(compResult))
		}) {
			return unhandled
		}
	}
	if lparam&win32.GCS_COMPSTR > 0 {
		compString, ok := parseImeCompositionString(ctx, win32.GCS_COMPSTR)
		if !ok {
			return unhandled
		}
		var caretPos gpui.Option[gpui.Range]
		if len(compString) != 0 && lparam&win32.GCS_CURSORPOS > 0 {
			cursorPos := retrieveCompositionCursorPosition(ctx)
			pos := len(compString)
			if shouldUseImeCursorPosition(ctx, cursorPos) {
				pos = cursorPos
			}
			caretPos = gpui.Some(gpui.Range{Start: pos, End: pos})
		}
		if !w.withInputHandler(func(inputHandler *gpui.PlatformInputHandler) {
			inputHandler.ReplaceAndMarkTextInRange(gpui.None[gpui.Range](), stringFromUTF16Lossy(compString), caretPos)
		}) {
			return unhandled
		}
	}
	if lparam&(win32.GCS_RESULTSTR|win32.GCS_COMPSTR) > 0 {
		return handled(0)
	}

	return unhandled
}

func (w *WindowsWindowInner) handleCalcClientSize(handle HWND, wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	if !w.hideTitleBar || w.state.isFullscreen() || wparam == 0 {
		return unhandled
	}

	params := lparamPtr[win32.NCCALCSIZE_PARAMS](lparam)
	savedTop := params.Rgrc[0].Top
	result := win32.DefWindowProcW(handle, win32.WM_NCCALCSIZE, wparam, lparam)
	params.Rgrc[0].Top = savedTop
	if w.state.isMaximized() {
		dpi := win32.GetDpiForWindow(handle)
		params.Rgrc[0].Top += getFrameThicknessx(dpi)
	}
	return msgResult{value: uintptr(result), ok: true}
}

func (w *WindowsWindowInner) handleActivateMsg(wparam win32.WPARAM) msgResult {
	activated := loword(wparam) > 0

	if cell := &w.state.a11y; !cell.borrowed && cell.value != nil {
		cell.borrowed = true
		events, ok := cell.value.updateWindowFocusState(activated)
		cell.borrowed = false
		if ok {
			raiseA11yEvents(events)
		}
	}

	if !activated {
		w.state.cursorVisible.Store(true)
	}

	if activated {
		w.state.lastReportedModifiers = gpui.None[gpui.Modifiers]()
		w.state.lastReportedCapslock = gpui.None[gpui.Capslock]()

		if fn := w.state.callbacks.input; fn != nil {
			w.state.callbacks.input = nil
			input := gpui.ModifiersChangedEvent{
				Modifiers: currentModifiers(),
				Capslock:  currentCapslock(),
			}
			fn(input)
			w.state.callbacks.input = fn
		}
	}

	w.executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		if fn := w.state.callbacks.activeStatusChange; fn != nil {
			w.state.callbacks.activeStatusChange = nil
			fn(activated)
			w.state.callbacks.activeStatusChange = fn
		}
		return struct{}{}, nil
	}).Detach()

	return unhandled
}

func (w *WindowsWindowInner) handleWmGetobject(wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	cell := &w.state.a11y
	if cell.borrowed {
		panic("already borrowed: BorrowMutError")
	}
	if cell.value == nil {
		return unhandled
	}
	cell.borrowed = true
	result, ok := cell.value.handleWMGetObject(wparam, lparam)
	cell.borrowed = false
	if !ok {
		return unhandled
	}
	return msgResult{value: uintptr(result()), ok: true}
}

func (w *WindowsWindowInner) handleCreateMsg(handle HWND) msgResult {
	if w.hideTitleBar {
		notifyFrameChanged(handle)
		return handled(0)
	}
	return unhandled
}

func (w *WindowsWindowInner) handleDpiChangedMsg(handle HWND, wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	newDpi := float32(loword(wparam))

	isMaximized := w.state.isMaximized()
	newScaleFactor := newDpi / float32(win32.USER_DEFAULT_SCREEN_DPI)
	w.state.scaleFactor = newScaleFactor
	logIfErr(w.state.borderOffset.update(handle))

	w.state.directManipulation.setScaleFactor(newScaleFactor)

	if isMaximized {
		monitor := win32.MonitorFromWindow(handle, win32.MONITOR_DEFAULTTONEAREST)
		var monitorInfo win32.MONITORINFO
		monitorInfo.CbSize = uint32(unsafe.Sizeof(monitorInfo))
		if win32.GetMonitorInfoW(monitor, &monitorInfo) == nil {
			workArea := monitorInfo.RcWork
			width := workArea.Right - workArea.Left
			height := workArea.Bottom - workArea.Top

			if err := win32.SetWindowPos(handle, 0, workArea.Left, workArea.Top, width, height, win32.SWP_NOZORDER|win32.SWP_NOACTIVATE|win32.SWP_FRAMECHANGED); err != nil {
				gpuiutil.LogErr(fmt.Errorf("unable to set maximized window position after dpi has changed: %w", err))
			}

			deviceSize := gpui.Size[gpui.DevicePixels]{Width: gpui.DevicePixels(width), Height: gpui.DevicePixels(height)}
			w.handleSizeChange(deviceSize, newScaleFactor, true)
		}
	} else {
		rect := lparamPtr[win32.RECT](lparam)
		width := rect.Right - rect.Left
		height := rect.Bottom - rect.Top
		if err := win32.SetWindowPos(handle, 0, rect.Left, rect.Top, width, height, win32.SWP_NOZORDER|win32.SWP_NOACTIVATE); err != nil {
			gpuiutil.LogErr(fmt.Errorf("unable to set window position after dpi has changed: %w", err))
		}
	}

	return handled(0)
}

func (w *WindowsWindowInner) handleDisplayChangeMsg(handle HWND) msgResult {
	newMonitor := win32.MonitorFromWindow(handle, win32.MONITOR_DEFAULTTONULL)
	if newMonitor == 0 {
		slog.Error("No monitor detected!")
		return unhandled
	}
	newDisplay, ok := NewWindowsDisplay(displayIDForMonitor(newMonitor))
	if !ok {
		return unhandled
	}
	w.setDisplay(newDisplay)
	return handled(0)
}

func (w *WindowsWindowInner) setDisplay(display *WindowsDisplay) {
	w.state.display = display
	w.executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		if callback := w.state.callbacks.displayChanged; callback != nil {
			w.state.callbacks.displayChanged = nil
			callback()
			w.state.callbacks.displayChanged = callback
		}
		return struct{}{}, nil
	}).Detach()
}

func (w *WindowsWindowInner) handleHitTestMsg(handle HWND, lparam win32.LPARAM) msgResult {
	if w.state.isFullscreen() {
		return unhandled
	}

	dragArea := unhandled
	if callback := w.state.callbacks.hitTestWindowControl; callback != nil {
		w.state.callbacks.hitTestWindowControl = nil
		area, ok := callback()
		w.state.callbacks.hitTestWindowControl = callback
		if ok {
			switch {
			case area == gpui.WindowControlAreaDrag && w.isMovable:
				dragArea = handled(win32.HTCAPTION)
			case area == gpui.WindowControlAreaDrag:
				dragArea = unhandled
			case area == gpui.WindowControlAreaClose:
				dragArea = handled(win32.HTCLOSE)
			case area == gpui.WindowControlAreaMax && w.isResizable:
				dragArea = handled(win32.HTMAXBUTTON)
			case area == gpui.WindowControlAreaMax && w.isMovable:
				dragArea = handled(win32.HTCAPTION)
			case area == gpui.WindowControlAreaMax:
				dragArea = handled(win32.HTNOWHERE)
			case area == gpui.WindowControlAreaMin && w.isMinimizable:
				dragArea = handled(win32.HTMINBUTTON)
			case area == gpui.WindowControlAreaMin && w.isMovable:
				dragArea = handled(win32.HTCAPTION)
			case area == gpui.WindowControlAreaMin:
				dragArea = handled(win32.HTNOWHERE)
			}
		}
	}

	if !w.hideTitleBar {
		return dragArea
	}

	dpi := win32.GetDpiForWindow(handle)
	frameY := getFrameThicknessx(dpi)
	frameX := getFrameThicknessy(dpi)
	cursorPoint := win32.POINT{X: int32(signedLoword(lparam)), Y: int32(signedHiword(lparam))}

	screenToClient(handle, &cursorPoint)
	if w.isResizable && !w.state.isMaximized() && 0 <= cursorPoint.Y && cursorPoint.Y <= frameY {
		if cursorPoint.X <= 0 {
			return handled(win32.HTTOPLEFT)
		}
		var rect win32.RECT
		logIfErr(win32.GetWindowRect(handle, &rect))
		right := rect.Right - rect.Left - 1
		if right-2*frameX <= cursorPoint.X {
			return handled(win32.HTTOPRIGHT)
		}
		return handled(win32.HTTOP)
	}

	return dragArea
}

func (w *WindowsWindowInner) handleNcMouseMoveMsg(handle HWND, lparam win32.LPARAM) msgResult {
	w.startTrackingMouse(handle, win32.TME_LEAVE|win32.TME_NONCLIENT)
	w.restoreCursorAfterHide()

	fn := w.state.callbacks.input
	if fn == nil {
		return unhandled
	}
	w.state.callbacks.input = nil
	scaleFactor := w.state.scaleFactor

	cursorPoint := win32.POINT{X: int32(signedLoword(lparam)), Y: int32(signedHiword(lparam))}
	screenToClient(handle, &cursorPoint)
	input := gpui.MouseMoveEvent{
		Position:      logicalPoint(float32(cursorPoint.X), float32(cursorPoint.Y), scaleFactor),
		PressedButton: gpui.None[gpui.MouseButton](),
		Modifiers:     currentModifiers(),
	}
	isHandled := !fn(input).Propagate
	w.state.callbacks.input = fn

	if isHandled {
		return handled(0)
	}
	return unhandled
}

func (w *WindowsWindowInner) handleNcMouseDownMsg(handle HWND, button gpui.MouseButton, wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	if fn := w.state.callbacks.input; fn != nil {
		w.state.callbacks.input = nil
		scaleFactor := w.state.scaleFactor
		cursorPoint := win32.POINT{X: int32(signedLoword(lparam)), Y: int32(signedHiword(lparam))}
		screenToClient(handle, &cursorPoint)
		physicalPoint := gpui.Point[gpui.DevicePixels]{X: gpui.DevicePixels(cursorPoint.X), Y: gpui.DevicePixels(cursorPoint.Y)}
		clickCount := w.state.clickState.update(button, physicalPoint)

		input := gpui.MouseDownEvent{
			Button:     button,
			Position:   logicalPoint(float32(cursorPoint.X), float32(cursorPoint.Y), scaleFactor),
			Modifiers:  currentModifiers(),
			ClickCount: clickCount,
			FirstMouse: false,
		}
		isHandled := !fn(input).Propagate
		w.state.callbacks.input = fn

		if isHandled {
			return handled(0)
		}
	}

	if button == gpui.MouseButtonLeft {
		switch uint32(wparam) {
		case win32.HTMINBUTTON:
			w.state.ncButtonPressed = gpui.Some[uint32](win32.HTMINBUTTON)
		case win32.HTMAXBUTTON:
			w.state.ncButtonPressed = gpui.Some[uint32](win32.HTMAXBUTTON)
		case win32.HTCLOSE:
			w.state.ncButtonPressed = gpui.Some[uint32](win32.HTCLOSE)
		default:
			return unhandled
		}
		return handled(0)
	}
	return unhandled
}

func (w *WindowsWindowInner) handleNcMouseUpMsg(handle HWND, button gpui.MouseButton, wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	if fn := w.state.callbacks.input; fn != nil {
		w.state.callbacks.input = nil
		scaleFactor := w.state.scaleFactor

		cursorPoint := win32.POINT{X: int32(signedLoword(lparam)), Y: int32(signedHiword(lparam))}
		screenToClient(handle, &cursorPoint)
		input := gpui.MouseUpEvent{
			Button:     button,
			Position:   logicalPoint(float32(cursorPoint.X), float32(cursorPoint.Y), scaleFactor),
			Modifiers:  currentModifiers(),
			ClickCount: 1,
		}
		isHandled := !fn(input).Propagate
		w.state.callbacks.input = fn

		if isHandled {
			return handled(0)
		}
	}

	lastPressed, ok := w.state.ncButtonPressed.Take().Get()
	if button == gpui.MouseButtonLeft && ok {
		isHandled := false
		hit := uint32(wparam)
		switch {
		case hit == win32.HTMINBUTTON && lastPressed == win32.HTMINBUTTON && w.isMinimizable:
			win32.ShowWindowAsync(handle, win32.SW_MINIMIZE)
			isHandled = true
		case hit == win32.HTMINBUTTON && lastPressed == win32.HTMINBUTTON:
			isHandled = true
		case hit == win32.HTMAXBUTTON && lastPressed == win32.HTMAXBUTTON && w.isResizable:
			if w.state.isMaximized() {
				win32.ShowWindowAsync(handle, win32.SW_NORMAL)
			} else {
				win32.ShowWindowAsync(handle, win32.SW_MAXIMIZE)
			}
			isHandled = true
		case hit == win32.HTMAXBUTTON && lastPressed == win32.HTMAXBUTTON:
			isHandled = true
		case hit == win32.HTCLOSE && lastPressed == win32.HTCLOSE:
			logIfErr(win32.PostMessageW(handle, win32.WM_CLOSE, 0, 0))
			isHandled = true
		}
		if isHandled {
			return handled(0)
		}
	}

	return unhandled
}

func (w *WindowsWindowInner) handleCursorChanged(lparam win32.LPARAM) msgResult {
	hadCursor := w.state.currentCursor != 0

	w.state.currentCursor = win32.HCURSOR(lparam)

	if hadCursor != (w.state.currentCursor != 0) {
		win32.SetCursor(w.state.currentCursor)
	}

	return handled(0)
}

func (w *WindowsWindowInner) handleSetCursor(handle HWND, lparam win32.LPARAM) msgResult {
	if !win32.IsWindowEnabled(handle) {
		return unhandled
	}
	switch uint32(loword(lparam)) {
	case win32.HTLEFT, win32.HTRIGHT, win32.HTTOP, win32.HTTOPLEFT, win32.HTTOPRIGHT,
		win32.HTBOTTOM, win32.HTBOTTOMLEFT, win32.HTBOTTOMRIGHT:
		return unhandled
	}
	var cursor win32.HCURSOR
	if w.state.cursorVisible.Load() {
		cursor = w.state.currentCursor
	}
	win32.SetCursor(cursor)
	return handled(0)
}

func (w *WindowsWindowInner) handleSystemSettingsChanged(handle HWND, wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	if wparam != 0 {
		w.state.clickState.systemUpdate(uintptr(wparam))
		logIfErr(w.state.borderOffset.update(handle))

		w.systemSettings.Update(uintptr(wparam))
	} else if !w.handleSystemThemeChanged(handle, lparam).ok {
		return unhandled
	}

	return handled(0)
}

func (w *WindowsWindowInner) handleSystemThemeChanged(handle HWND, lparam win32.LPARAM) msgResult {
	parameter := lparamPtr[uint16](lparam)
	if parameter != nil && *parameter != 0 {
		parameterString := xwin.UTF16PtrToString(parameter)
		slog.Info(fmt.Sprintf("System settings changed: %s", parameterString))
		if parameterString == "ImmersiveColorSet" {
			newAppearance, err := systemAppearance()
			if err != nil {
				gpuiutil.LogErr(fmt.Errorf("unable to get system appearance when handling ImmersiveColorSet: %w", err))
				return unhandled
			}

			if newAppearance != w.state.appearance {
				w.state.appearance = newAppearance
				callback := w.state.callbacks.appearanceChanged
				if callback == nil {
					return unhandled
				}
				w.state.callbacks.appearanceChanged = nil

				callback()
				w.state.callbacks.appearanceChanged = callback
				configureDwmDarkMode(handle, newAppearance)
			}
		}
	}
	return handled(0)
}

func (w *WindowsWindowInner) handleInputLanguageChanged() msgResult {
	logIfErr(win32.PostMessageW(w.platformWindowHandle, WM_GPUI_KEYBOARD_LAYOUT_CHANGED, win32.WPARAM(w.validationNumber), 0))
	return handled(0)
}

func (w *WindowsWindowInner) handleWindowVisibilityChanged(handle HWND, wparam win32.WPARAM) msgResult {
	w.reportVisibility()
	if wparam == 1 {
		w.drawWindow(handle, false)
	}
	return unhandled
}

func (w *WindowsWindowInner) reportVisibility() {
	if w.state.lastVisibility.IsNone() {
		return
	}
	w.executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		visibility := w.visibility()
		if w.state.lastVisibility == gpui.Some(visibility) {
			return struct{}{}, nil
		}
		w.state.lastVisibility = gpui.Some(visibility)
		if callback := w.state.callbacks.visibilityChange; callback != nil {
			w.state.callbacks.visibilityChange = nil
			callback(visibility)
			w.state.callbacks.visibilityChange = callback
		}
		return struct{}{}, nil
	}).Detach()
}

func (w *WindowsWindowInner) handleGpuiEvents(handle HWND, message uint32, wparam win32.WPARAM, lparam win32.LPARAM) msgResult {
	if uintptr(wparam) != w.validationNumber {
		return unhandled
	}
	switch message {
	case WM_GPUI_CURSOR_STYLE_CHANGED:
		return w.handleCursorChanged(lparam)
	case WM_GPUI_FORCE_UPDATE_WINDOW:
		return w.drawWindow(handle, true)
	case WM_GPUI_GPU_DEVICE_LOST:
		return w.handleDeviceLost(lparam)
	}
	panic("internal error: entered unreachable code")
}

func (w *WindowsWindowInner) handleDeviceLost(lparam win32.LPARAM) msgResult {
	if lparam == 0 {
		return unhandled
	}
	devices := lparamPtr[DirectXDevices](lparam)
	if err := w.state.renderer.HandleDeviceLost(devices); err != nil {
		panic(fmt.Sprintf("Device lost: %v", err))
	}
	w.state.forceRenderPending = true
	return handled(0)
}

func (w *WindowsWindowInner) handleDmPointerHitTest(wparam win32.WPARAM) msgResult {
	w.state.directManipulation.onPointerHitTest(wparam)
	return unhandled
}

func (w *WindowsWindowInner) drawWindow(handle HWND, forceRender bool) msgResult {
	guard, ok := w.state.drawCoordinator.tryBeginDraw()
	if !ok {
		slog.Debug(fmt.Sprintf("deferring re-entrant draw of window %#x", uintptr(handle)))
		if forceRender {
			w.state.forceRenderPending = true
		}
		logLastError(win32.ValidateRect(handle, nil), "ValidateRect")
		return handled(0)
	}
	defer guard.Release()
	requestFrame := w.state.callbacks.requestFrame
	if requestFrame == nil {
		return unhandled
	}
	w.state.callbacks.requestFrame = nil

	w.state.directManipulation.update()

	events := w.state.directManipulation.drainEvents()
	if len(events) != 0 {
		if fn := w.state.callbacks.input; fn != nil {
			w.state.callbacks.input = nil
			for _, event := range events {
				fn(event)
			}
			w.state.callbacks.input = fn
		}
	}

	pending := w.state.forceRenderPending
	w.state.forceRenderPending = false
	forceRender = forceRender || pending
	if forceRender {
		w.state.renderer.MarkDrawable()
	}
	signalAt := gpui.None[scheduler.Instant]()
	signalSource := gpui.FrameRequestSourceNativeCallback
	if at, source, ok := w.state.frameSignal.Take(); ok {
		signalAt = gpui.Some(at)
		signalSource = source
	}
	requestFrame(gpui.RequestFrameOptions{
		RequirePresentation: false,
		ForceRender:         forceRender,
		SignalAt:            signalAt,
		SignalSource:        signalSource,
	})

	w.state.callbacks.requestFrame = requestFrame
	w.updateImeEnabled(handle)
	logLastError(win32.ValidateRect(handle, nil), "ValidateRect")

	return handled(0)
}

func (w *WindowsWindowInner) parseCharMessage(wparam win32.WPARAM) (string, bool) {
	codePoint := loword(wparam)

	switch {
	case codePoint >= 0xD800 && codePoint <= 0xDBFF:
		w.state.pendingSurrogate = gpui.Some(codePoint)
		return "", false
	case codePoint >= 0xDC00 && codePoint <= 0xDFFF:
		if highSurrogate, ok := w.state.pendingSurrogate.Take().Get(); ok {
			return strictUTF16([]uint16{highSurrogate, codePoint})
		}
		slog.Warn(fmt.Sprintf("Received low surrogate without a preceding high surrogate: %x", codePoint))
		return "", false
	default:
		w.state.pendingSurrogate = gpui.None[uint16]()
		c := rune(codePoint)
		if unicode.IsControl(c) {
			return "", false
		}
		return string(c), true
	}
}

func (w *WindowsWindowInner) restoreCursorAfterHide() {
	if !w.state.cursorVisible.Swap(true) {
		win32.SetCursor(w.state.currentCursor)
	}
}

func (w *WindowsWindowInner) startTrackingMouse(handle HWND, flags uint32) {
	if !w.state.hovered {
		w.state.hovered = true
		event := win32.TRACKMOUSEEVENT{
			DwFlags:     flags,
			HwndTrack:   handle,
			DwHoverTime: hoverDefault,
		}
		event.CbSize = uint32(unsafe.Sizeof(event))
		logIfErr(win32.TrackMouseEvent(&event))
		if callback := w.state.callbacks.hoveredStatusChange; callback != nil {
			w.state.callbacks.hoveredStatusChange = nil
			callback(true)
			w.state.callbacks.hoveredStatusChange = callback
		}
	}
}

func (w *WindowsWindowInner) withInputHandler(f func(*gpui.PlatformInputHandler)) bool {
	_, ok := withInputHandlerR(w, func(h *gpui.PlatformInputHandler) struct{} {
		f(h)
		return struct{}{}
	})
	return ok
}

func withInputHandlerR[R any](w *WindowsWindowInner, f func(*gpui.PlatformInputHandler) R) (R, bool) {
	inputHandler := w.state.inputHandler
	if inputHandler == nil {
		var zero R
		return zero, false
	}
	w.state.inputHandler = nil
	result := f(inputHandler)
	w.state.inputHandler = inputHandler
	return result, true
}

func withInputHandlerAndScaleFactor[R any](w *WindowsWindowInner, f func(*gpui.PlatformInputHandler, float32) (R, bool)) (R, bool) {
	inputHandler := w.state.inputHandler
	if inputHandler == nil {
		var zero R
		return zero, false
	}
	w.state.inputHandler = nil
	scaleFactor := w.state.scaleFactor

	result, ok := f(inputHandler, scaleFactor)
	w.state.inputHandler = inputHandler
	return result, ok
}

type imeContext struct {
	hwnd HWND
	himc win32.HIMC
}

func getImeContext(hwnd HWND) (*imeContext, bool) {
	himc := win32.ImmGetContext(hwnd)
	if himc == 0 {
		return nil, false
	}
	return &imeContext{hwnd: hwnd, himc: himc}, true
}

func (c *imeContext) Release() {
	logLastError(win32.ImmReleaseContext(c.hwnd, c.himc), "ImmReleaseContext")
}

func handleKeyEvent(wparam win32.WPARAM, lparam win32.LPARAM, state *WindowsWindowState, f func(gpui.Keystroke, bool) gpui.PlatformInput) (gpui.PlatformInput, bool) {
	virtualKey := VIRTUAL_KEY(loword(wparam))
	modifiers := currentModifiers()

	switch virtualKey {
	case win32.VK_SHIFT, win32.VK_CONTROL, win32.VK_MENU, win32.VK_LMENU, win32.VK_RMENU, win32.VK_LWIN, win32.VK_RWIN:
		if prev, ok := state.lastReportedModifiers.Get(); ok && prev == modifiers {
			return nil, false
		}
		state.lastReportedModifiers = gpui.Some(modifiers)
		return gpui.ModifiersChangedEvent{
			Modifiers: modifiers,
			Capslock:  currentCapslock(),
		}, true
	case win32.VK_PACKET:
		return nil, false
	case win32.VK_CAPITAL:
		capslock := currentCapslock()
		if prev, ok := state.lastReportedCapslock.Get(); ok && prev == capslock {
			return nil, false
		}
		state.lastReportedCapslock = gpui.Some(capslock)
		return gpui.ModifiersChangedEvent{
			Modifiers: modifiers,
			Capslock:  capslock,
		}, true
	default:
		keystroke, preferCharacterInput, ok := parseNormalKey(virtualKey, lparam, modifiers)
		if !ok {
			return nil, false
		}
		return f(keystroke, preferCharacterInput), true
	}
}

func parseImmutable(vkey VIRTUAL_KEY) (string, bool) {
	switch vkey {
	case win32.VK_SPACE:
		return "space", true
	case win32.VK_BACK:
		return "backspace", true
	case win32.VK_RETURN:
		return "enter", true
	case win32.VK_TAB:
		return "tab", true
	case win32.VK_UP:
		return "up", true
	case win32.VK_DOWN:
		return "down", true
	case win32.VK_RIGHT:
		return "right", true
	case win32.VK_LEFT:
		return "left", true
	case win32.VK_HOME:
		return "home", true
	case win32.VK_END:
		return "end", true
	case win32.VK_PRIOR:
		return "pageup", true
	case win32.VK_NEXT:
		return "pagedown", true
	case win32.VK_BROWSER_BACK:
		return "back", true
	case win32.VK_BROWSER_FORWARD:
		return "forward", true
	case win32.VK_ESCAPE:
		return "escape", true
	case win32.VK_INSERT:
		return "insert", true
	case win32.VK_DELETE:
		return "delete", true
	case win32.VK_APPS:
		return "menu", true
	}
	if vkey >= win32.VK_F1 && vkey <= win32.VK_F24 {
		return fmt.Sprintf("f%d", vkey-win32.VK_F1+1), true
	}
	return "", false
}

func parseNormalKey(vkey VIRTUAL_KEY, lparam win32.LPARAM, modifiers gpui.Modifiers) (gpui.Keystroke, bool, bool) {
	keyChar, preferCharacterInput := processKey(vkey, hiword(lparam))

	key, ok := parseImmutable(vkey)
	if !ok {
		scanCode := hiword(lparam) & 0xFF
		key, ok = getKeystrokeKey(vkey, uint32(scanCode), &modifiers)
		if !ok {
			return gpui.Keystroke{}, false, false
		}
	}

	return gpui.Keystroke{
		Modifiers: modifiers,
		Key:       key,
		KeyChar:   keyChar,
	}, preferCharacterInput, true
}

func processKey(vkey VIRTUAL_KEY, scanCode uint16) (gpui.Option[string], bool) {
	var keyboardState [256]byte
	if err := win32.GetKeyboardState(&keyboardState); err != nil {
		return gpui.None[string](), false
	}

	var bufferC [8]uint16
	resultC := win32.ToUnicode(uint32(vkey), uint32(scanCode), &keyboardState, bufferC[:], 0x4)

	if resultC == 0 {
		return gpui.None[string](), false
	}

	c := bufferC[:absInt32(resultC)]
	keyChar := gpui.None[string]()
	if s, ok := strictUTF16(c); ok && s != "" && !unicode.IsControl([]rune(s)[0]) {
		keyChar = gpui.Some(s)
	}

	if resultC < 0 {
		return keyChar, true
	}

	if keyChar.IsNone() {
		return gpui.None[string](), false
	}

	ctrlDown := keyboardState[win32.VK_CONTROL]&0x80 != 0
	altDown := keyboardState[win32.VK_MENU]&0x80 != 0
	winDown := keyboardState[win32.VK_LWIN]&0x80 != 0 || keyboardState[win32.VK_RWIN]&0x80 != 0

	hasModifiers := ctrlDown || altDown || winDown
	if !hasModifiers {
		return keyChar, false
	}

	stateNoModifiers := keyboardState
	stateNoModifiers[win32.VK_CONTROL] = 0
	stateNoModifiers[win32.VK_LCONTROL] = 0
	stateNoModifiers[win32.VK_RCONTROL] = 0
	stateNoModifiers[win32.VK_MENU] = 0
	stateNoModifiers[win32.VK_LMENU] = 0
	stateNoModifiers[win32.VK_RMENU] = 0
	stateNoModifiers[win32.VK_LWIN] = 0
	stateNoModifiers[win32.VK_RWIN] = 0

	var bufferCNoModifiers [8]uint16
	resultCNoModifiers := win32.ToUnicode(uint32(vkey), uint32(scanCode), &stateNoModifiers, bufferCNoModifiers[:], 0x4)

	cNoModifiers := bufferCNoModifiers[:absInt32(resultCNoModifiers)]
	return keyChar, resultC != resultCNoModifiers || !slices.Equal(c, cNoModifiers)
}

func absInt32(v int32) int {
	if v < 0 {
		return int(-v)
	}
	return int(v)
}

func stringFromUTF16Lossy(s []uint16) string {
	return string(utf16.Decode(s))
}

func parseImeCompositionString(ctx win32.HIMC, compType uint32) ([]uint16, bool) {
	stringLen := win32.ImmGetCompositionStringW(ctx, compType, nil, 0)
	if stringLen < 0 {
		return nil, false
	}
	buffer := make([]uint16, (int(stringLen)+2+1)/2)
	win32.ImmGetCompositionStringW(ctx, compType, unsafe.Pointer(&buffer[0]), uint32(stringLen))
	wstring := make([]uint16, int(stringLen)/2)
	copy(wstring, buffer)
	return wstring, true
}

func retrieveCompositionCursorPosition(ctx win32.HIMC) int {
	return usizeFromInt32(win32.ImmGetCompositionStringW(ctx, win32.GCS_CURSORPOS, nil, 0))
}

func usizeFromInt32(v int32) int {
	if v < 0 {
		return math.MaxInt
	}
	return int(v)
}

func shouldUseImeCursorPosition(ctx win32.HIMC, cursorPos int) bool {
	attrsSize := usizeFromInt32(win32.ImmGetCompositionStringW(ctx, win32.GCS_COMPATTR, nil, 0))
	if attrsSize == 0 || attrsSize == math.MaxInt {
		return false
	}

	attrs := make([]byte, attrsSize)
	result := win32.ImmGetCompositionStringW(ctx, win32.GCS_COMPATTR, unsafe.Pointer(&attrs[0]), uint32(attrsSize))
	if result <= 0 {
		return false
	}

	atCursorIsInput := cursorPos < len(attrs) && attrs[cursorPos] == attrInput
	beforeCursorIsInput := cursorPos > 0 && (cursorPos-1) < len(attrs) && attrs[cursorPos-1] == attrInput

	return atCursorIsInput || beforeCursorIsInput
}

func isVirtualKeyPressed(vkey VIRTUAL_KEY) bool {
	return win32.GetKeyState(int32(vkey)) < 0
}

func currentModifiers() gpui.Modifiers {
	return gpui.Modifiers{
		Control:  isVirtualKeyPressed(win32.VK_CONTROL),
		Alt:      isVirtualKeyPressed(win32.VK_MENU),
		Shift:    isVirtualKeyPressed(win32.VK_SHIFT),
		Platform: isVirtualKeyPressed(win32.VK_LWIN) || isVirtualKeyPressed(win32.VK_RWIN),
		Function: false,
	}
}

func currentCapslock() gpui.Capslock {
	on := win32.GetKeyState(win32.VK_CAPITAL)&1 > 0
	return gpui.Capslock{On: on}
}

func getFrameThicknessx(dpi uint32) int32 {
	resizeFrameThickness := win32.GetSystemMetricsForDpi(win32.SM_CXSIZEFRAME, dpi)
	paddingThickness := win32.GetSystemMetricsForDpi(win32.SM_CXPADDEDBORDER, dpi)
	return resizeFrameThickness + paddingThickness
}

func getFrameThicknessy(dpi uint32) int32 {
	resizeFrameThickness := win32.GetSystemMetricsForDpi(win32.SM_CYSIZEFRAME, dpi)
	paddingThickness := win32.GetSystemMetricsForDpi(win32.SM_CXPADDEDBORDER, dpi)
	return resizeFrameThickness + paddingThickness
}

func notifyFrameChanged(handle HWND) {
	logIfErr(win32.SetWindowPos(handle, 0, 0, 0, 0, 0,
		win32.SWP_FRAMECHANGED|
			win32.SWP_NOACTIVATE|
			win32.SWP_NOCOPYBITS|
			win32.SWP_NOMOVE|
			win32.SWP_NOOWNERZORDER|
			win32.SWP_NOREPOSITION|
			win32.SWP_NOSENDCHANGING|
			win32.SWP_NOSIZE|
			win32.SWP_NOZORDER))
}
