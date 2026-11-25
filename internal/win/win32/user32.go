//go:build windows

package win32

import (
	"unsafe"
)

var (
	procAdjustWindowRectEx                  = user32.NewProc("AdjustWindowRectEx")
	procAdjustWindowRectExForDpi            = user32.NewProc("AdjustWindowRectExForDpi")
	procClientToScreen                      = user32.NewProc("ClientToScreen")
	procCloseClipboard                      = user32.NewProc("CloseClipboard")
	procCountClipboardFormats               = user32.NewProc("CountClipboardFormats")
	procCreateWindowExW                     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW                      = user32.NewProc("DefWindowProcW")
	procDestroyWindow                       = user32.NewProc("DestroyWindow")
	procDispatchMessageW                    = user32.NewProc("DispatchMessageW")
	procEmptyClipboard                      = user32.NewProc("EmptyClipboard")
	procEnableWindow                        = user32.NewProc("EnableWindow")
	procEnumClipboardFormats                = user32.NewProc("EnumClipboardFormats")
	procEnumDisplayMonitors                 = user32.NewProc("EnumDisplayMonitors")
	procEnumDisplaySettingsW                = user32.NewProc("EnumDisplaySettingsW")
	procEnumThreadWindows                   = user32.NewProc("EnumThreadWindows")
	procFlashWindowEx                       = user32.NewProc("FlashWindowEx")
	procGetActiveWindow                     = user32.NewProc("GetActiveWindow")
	procGetAncestor                         = user32.NewProc("GetAncestor")
	procGetClassNameW                       = user32.NewProc("GetClassNameW")
	procGetClientRect                       = user32.NewProc("GetClientRect")
	procGetClipboardData                    = user32.NewProc("GetClipboardData")
	procGetClipboardFormatNameW             = user32.NewProc("GetClipboardFormatNameW")
	procGetCursorPos                        = user32.NewProc("GetCursorPos")
	procGetDoubleClickTime                  = user32.NewProc("GetDoubleClickTime")
	procGetDpiForSystem                     = user32.NewProc("GetDpiForSystem")
	procGetDpiForWindow                     = user32.NewProc("GetDpiForWindow")
	procGetFocus                            = user32.NewProc("GetFocus")
	procGetForegroundWindow                 = user32.NewProc("GetForegroundWindow")
	procGetKeyState                         = user32.NewProc("GetKeyState")
	procGetKeyboardLayout                   = user32.NewProc("GetKeyboardLayout")
	procGetKeyboardLayoutNameW              = user32.NewProc("GetKeyboardLayoutNameW")
	procGetKeyboardState                    = user32.NewProc("GetKeyboardState")
	procGetMessageW                         = user32.NewProc("GetMessageW")
	procGetMonitorInfoW                     = user32.NewProc("GetMonitorInfoW")
	procGetParent                           = user32.NewProc("GetParent")
	procGetPointerType                      = user32.NewProc("GetPointerType")
	procGetProcessWindowStation             = user32.NewProc("GetProcessWindowStation")
	procGetPropW                            = user32.NewProc("GetPropW")
	procGetSystemMetrics                    = user32.NewProc("GetSystemMetrics")
	procGetSystemMetricsForDpi              = user32.NewProc("GetSystemMetricsForDpi")
	procGetUserObjectInformationW           = user32.NewProc("GetUserObjectInformationW")
	procGetWindowLongPtrW                   = user32.NewProc("GetWindowLongPtrW")
	procGetWindowLongW                      = user32.NewProc("GetWindowLongW")
	procGetWindowPlacement                  = user32.NewProc("GetWindowPlacement")
	procGetWindowRect                       = user32.NewProc("GetWindowRect")
	procInvalidateRect                      = user32.NewProc("InvalidateRect")
	procIsClipboardFormatAvailable          = user32.NewProc("IsClipboardFormatAvailable")
	procIsIconic                            = user32.NewProc("IsIconic")
	procIsWindow                            = user32.NewProc("IsWindow")
	procIsWindowEnabled                     = user32.NewProc("IsWindowEnabled")
	procIsWindowVisible                     = user32.NewProc("IsWindowVisible")
	procIsZoomed                            = user32.NewProc("IsZoomed")
	procKillTimer                           = user32.NewProc("KillTimer")
	procLoadImageW                          = user32.NewProc("LoadImageW")
	procMapVirtualKeyW                      = user32.NewProc("MapVirtualKeyW")
	procMessageBeep                         = user32.NewProc("MessageBeep")
	procMonitorFromPoint                    = user32.NewProc("MonitorFromPoint")
	procMonitorFromWindow                   = user32.NewProc("MonitorFromWindow")
	procMsgWaitForMultipleObjects           = user32.NewProc("MsgWaitForMultipleObjects")
	procOpenClipboard                       = user32.NewProc("OpenClipboard")
	procPeekMessageW                        = user32.NewProc("PeekMessageW")
	procPostMessageW                        = user32.NewProc("PostMessageW")
	procPostQuitMessage                     = user32.NewProc("PostQuitMessage")
	procPostThreadMessageW                  = user32.NewProc("PostThreadMessageW")
	procRedrawWindow                        = user32.NewProc("RedrawWindow")
	procRegisterClassW                      = user32.NewProc("RegisterClassW")
	procRegisterClipboardFormatW            = user32.NewProc("RegisterClipboardFormatW")
	procRegisterPowerSettingNotification    = user32.NewProc("RegisterPowerSettingNotification")
	procRegisterSuspendResumeNotification   = user32.NewProc("RegisterSuspendResumeNotification")
	procReleaseCapture                      = user32.NewProc("ReleaseCapture")
	procRemovePropW                         = user32.NewProc("RemovePropW")
	procScreenToClient                      = user32.NewProc("ScreenToClient")
	procSendInput                           = user32.NewProc("SendInput")
	procSendMessageW                        = user32.NewProc("SendMessageW")
	procSetActiveWindow                     = user32.NewProc("SetActiveWindow")
	procSetCapture                          = user32.NewProc("SetCapture")
	procSetClipboardData                    = user32.NewProc("SetClipboardData")
	procSetCursor                           = user32.NewProc("SetCursor")
	procSetFocus                            = user32.NewProc("SetFocus")
	procSetForegroundWindow                 = user32.NewProc("SetForegroundWindow")
	procSetProcessDpiAwarenessContext       = user32.NewProc("SetProcessDpiAwarenessContext")
	procGetThreadDpiAwarenessContext        = user32.NewProc("GetThreadDpiAwarenessContext")
	procAreDpiAwarenessContextsEqual        = user32.NewProc("AreDpiAwarenessContextsEqual")
	procSetPropW                            = user32.NewProc("SetPropW")
	procSetTimer                            = user32.NewProc("SetTimer")
	procSetWindowLongPtrW                   = user32.NewProc("SetWindowLongPtrW")
	procSetWindowLongW                      = user32.NewProc("SetWindowLongW")
	procSetWindowPlacement                  = user32.NewProc("SetWindowPlacement")
	procSetWindowPos                        = user32.NewProc("SetWindowPos")
	procSetWindowTextW                      = user32.NewProc("SetWindowTextW")
	procShowWindow                          = user32.NewProc("ShowWindow")
	procShowWindowAsync                     = user32.NewProc("ShowWindowAsync")
	procSystemParametersInfoW               = user32.NewProc("SystemParametersInfoW")
	procToUnicode                           = user32.NewProc("ToUnicode")
	procTrackMouseEvent                     = user32.NewProc("TrackMouseEvent")
	procTranslateMessage                    = user32.NewProc("TranslateMessage")
	procUnregisterClassW                    = user32.NewProc("UnregisterClassW")
	procUnregisterPowerSettingNotification  = user32.NewProc("UnregisterPowerSettingNotification")
	procUnregisterSuspendResumeNotification = user32.NewProc("UnregisterSuspendResumeNotification")
	procUpdateWindow                        = user32.NewProc("UpdateWindow")
	procValidateRect                        = user32.NewProc("ValidateRect")
	procWaitMessage                         = user32.NewProc("WaitMessage")
	procWindowFromPoint                     = user32.NewProc("WindowFromPoint")
)

// AdjustWindowRectEx grows rect from a client rectangle to the window rectangle.
func AdjustWindowRectEx(rect *RECT, style uint32, menu bool, exStyle uint32) error {
	return boolResult(procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(rect)), uintptr(style), uintptr(Bool(menu)), uintptr(exStyle)))
}

// AdjustWindowRectExForDpi is AdjustWindowRectEx at the given DPI.
func AdjustWindowRectExForDpi(rect *RECT, style uint32, menu bool, exStyle uint32, dpi uint32) error {
	return boolResult(procAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(rect)), uintptr(style), uintptr(Bool(menu)), uintptr(exStyle), uintptr(dpi)))
}

// ClientToScreen converts a client point to screen coordinates.
func ClientToScreen(hwnd HWND, point *POINT) bool {
	r, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(point)))
	return r != 0
}

// CloseClipboard closes the clipboard.
func CloseClipboard() error {
	return boolResult(procCloseClipboard.Call())
}

// CountClipboardFormats returns the number of formats on the clipboard.
func CountClipboardFormats() int32 {
	r, _, _ := procCountClipboardFormats.Call()
	return int32(r)
}

// CreateWindowExW creates a window.
func CreateWindowExW(exStyle uint32, className *uint16, windowName *uint16, style uint32, x, y, width, height int32, parent HWND, menu HMENU, instance HINSTANCE, param uintptr) (HWND, error) {
	r, _, err := procCreateWindowExW.Call(uintptr(exStyle), uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(windowName)), uintptr(style), uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(parent), uintptr(menu), uintptr(instance), param)
	if r == 0 {
		return 0, lastError(err)
	}
	return HWND(r), nil
}

// DefWindowProcW is the default window procedure.
func DefWindowProcW(hwnd HWND, msg uint32, wparam WPARAM, lparam LPARAM) LRESULT {
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), uintptr(wparam), uintptr(lparam))
	return LRESULT(r)
}

// DestroyWindow destroys a window.
func DestroyWindow(hwnd HWND) error {
	return boolResult(procDestroyWindow.Call(uintptr(hwnd)))
}

// DispatchMessageW dispatches a message to its window procedure.
func DispatchMessageW(msg *MSG) LRESULT {
	r, _, _ := procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
	return LRESULT(r)
}

// EmptyClipboard empties the clipboard and takes ownership of it.
func EmptyClipboard() error {
	return boolResult(procEmptyClipboard.Call())
}

// EnableWindow enables or disables input to a window and reports whether it was disabled.
func EnableWindow(hwnd HWND, enable bool) bool {
	r, _, _ := procEnableWindow.Call(uintptr(hwnd), uintptr(Bool(enable)))
	return r != 0
}

// EnumClipboardFormats returns the clipboard format after format, or 0 at the end.
func EnumClipboardFormats(format uint32) uint32 {
	r, _, _ := procEnumClipboardFormats.Call(uintptr(format))
	return uint32(r)
}

// EnumDisplayMonitors calls callback, a windows.NewCallback MONITORENUMPROC, for each monitor.
func EnumDisplayMonitors(hdc HDC, clip *RECT, callback uintptr, data LPARAM) error {
	return boolResult(procEnumDisplayMonitors.Call(uintptr(hdc), uintptr(unsafe.Pointer(clip)), callback, uintptr(data)))
}

// EnumDisplaySettingsW reads a display device's graphics mode.
func EnumDisplaySettingsW(deviceName *uint16, modeNum uint32, mode *DEVMODEW) bool {
	r, _, _ := procEnumDisplaySettingsW.Call(uintptr(unsafe.Pointer(deviceName)), uintptr(modeNum), uintptr(unsafe.Pointer(mode)))
	return r != 0
}

// EnumThreadWindows calls callback, a windows.NewCallback WNDENUMPROC, for each window of a thread.
func EnumThreadWindows(threadID uint32, callback uintptr, data LPARAM) bool {
	r, _, _ := procEnumThreadWindows.Call(uintptr(threadID), callback, uintptr(data))
	return r != 0
}

// FlashWindowEx flashes a window and reports whether it was active before.
func FlashWindowEx(info *FLASHWINFO) bool {
	r, _, _ := procFlashWindowEx.Call(uintptr(unsafe.Pointer(info)))
	return r != 0
}

// GetActiveWindow returns the active window of the calling thread.
func GetActiveWindow() HWND {
	r, _, _ := procGetActiveWindow.Call()
	return HWND(r)
}

// GetAncestor returns an ancestor of a window.
func GetAncestor(hwnd HWND, flags uint32) HWND {
	r, _, _ := procGetAncestor.Call(uintptr(hwnd), uintptr(flags))
	return HWND(r)
}

// GetClassNameW copies a window's class name into buf and returns its length.
func GetClassNameW(hwnd HWND, buf []uint16) (int32, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	r, _, err := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return 0, lastError(err)
	}
	return int32(r), nil
}

// GetClientRect returns a window's client rectangle.
func GetClientRect(hwnd HWND, rect *RECT) error {
	return boolResult(procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect))))
}

// GetClipboardData returns the clipboard data in format.
func GetClipboardData(format uint32) (HANDLE, error) {
	r, _, err := procGetClipboardData.Call(uintptr(format))
	if r == 0 {
		return 0, lastError(err)
	}
	return HANDLE(r), nil
}

// GetClipboardFormatNameW copies a registered format's name into buf and returns its length.
func GetClipboardFormatNameW(format uint32, buf []uint16) int32 {
	if len(buf) == 0 {
		return 0
	}
	r, _, _ := procGetClipboardFormatNameW.Call(uintptr(format), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return int32(r)
}

// GetCursorPos returns the cursor position in screen coordinates.
func GetCursorPos(point *POINT) error {
	return boolResult(procGetCursorPos.Call(uintptr(unsafe.Pointer(point))))
}

// GetDoubleClickTime returns the double-click time in milliseconds.
func GetDoubleClickTime() uint32 {
	r, _, _ := procGetDoubleClickTime.Call()
	return uint32(r)
}

// GetDpiForSystem returns the system DPI.
func GetDpiForSystem() uint32 {
	r, _, _ := procGetDpiForSystem.Call()
	return uint32(r)
}

// GetDpiForWindow returns a window's DPI.
func GetDpiForWindow(hwnd HWND) uint32 {
	r, _, _ := procGetDpiForWindow.Call(uintptr(hwnd))
	return uint32(r)
}

// GetFocus returns the window with keyboard focus on the calling thread.
func GetFocus() HWND {
	r, _, _ := procGetFocus.Call()
	return HWND(r)
}

// GetForegroundWindow returns the foreground window.
func GetForegroundWindow() HWND {
	r, _, _ := procGetForegroundWindow.Call()
	return HWND(r)
}

// GetKeyState returns the state of a virtual key at the current message.
func GetKeyState(vk int32) int16 {
	r, _, _ := procGetKeyState.Call(uintptr(vk))
	return int16(r)
}

// GetKeyboardLayout returns a thread's active input locale.
func GetKeyboardLayout(threadID uint32) HKL {
	r, _, _ := procGetKeyboardLayout.Call(uintptr(threadID))
	return HKL(r)
}

// GetKeyboardLayoutNameW copies the active layout's KLID into buf (KL_NAMELENGTH units).
func GetKeyboardLayoutNameW(buf *[KL_NAMELENGTH]uint16) error {
	return boolResult(procGetKeyboardLayoutNameW.Call(uintptr(unsafe.Pointer(&buf[0]))))
}

// GetKeyboardState copies the state of all 256 virtual keys.
func GetKeyboardState(state *[256]byte) error {
	return boolResult(procGetKeyboardState.Call(uintptr(unsafe.Pointer(&state[0]))))
}

// GetMessageW waits for and retrieves a message; it returns 0 for WM_QUIT and -1 on error.
func GetMessageW(msg *MSG, hwnd HWND, filterMin, filterMax uint32) int32 {
	r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(msg)), uintptr(hwnd), uintptr(filterMin), uintptr(filterMax))
	return int32(r)
}

// GetMonitorInfoW fills info, a MONITORINFO or the MONITORINFO header of a MONITORINFOEXW.
func GetMonitorInfoW(monitor HMONITOR, info *MONITORINFO) error {
	return boolResult(procGetMonitorInfoW.Call(uintptr(monitor), uintptr(unsafe.Pointer(info))))
}

// GetParent returns a window's parent or owner.
func GetParent(hwnd HWND) HWND {
	r, _, _ := procGetParent.Call(uintptr(hwnd))
	return HWND(r)
}

// GetPointerType returns the input type of a pointer.
func GetPointerType(pointerID uint32, pointerType *int32) error {
	return boolResult(procGetPointerType.Call(uintptr(pointerID), uintptr(unsafe.Pointer(pointerType))))
}

// GetProcessWindowStation returns the calling process's window station.
func GetProcessWindowStation() (HWINSTA, error) {
	r, _, err := procGetProcessWindowStation.Call()
	if r == 0 {
		return 0, lastError(err)
	}
	return HWINSTA(r), nil
}

// GetPropW returns a window property.
func GetPropW(hwnd HWND, name *uint16) HANDLE {
	r, _, _ := procGetPropW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(name)))
	return HANDLE(r)
}

// GetSystemMetrics returns a system metric.
func GetSystemMetrics(index int32) int32 {
	r, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int32(r)
}

// GetSystemMetricsForDpi returns a system metric at the given DPI.
func GetSystemMetricsForDpi(index int32, dpi uint32) int32 {
	r, _, _ := procGetSystemMetricsForDpi.Call(uintptr(index), uintptr(dpi))
	return int32(r)
}

// GetUserObjectInformationW reads information about a window station or desktop.
func GetUserObjectInformationW(object HANDLE, index int32, info unsafe.Pointer, length uint32, needed *uint32) error {
	return boolResult(procGetUserObjectInformationW.Call(uintptr(object), uintptr(index), uintptr(info), uintptr(length), uintptr(unsafe.Pointer(needed))))
}

// GetWindowLongPtrW reads a window's pointer-sized value at index.
func GetWindowLongPtrW(hwnd HWND, index int32) uintptr {
	r, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(index))
	return r
}

// GetWindowLongW reads a window's 32-bit value at index.
func GetWindowLongW(hwnd HWND, index int32) int32 {
	r, _, _ := procGetWindowLongW.Call(uintptr(hwnd), uintptr(index))
	return int32(r)
}

// GetWindowPlacement reads a window's placement.
func GetWindowPlacement(hwnd HWND, placement *WINDOWPLACEMENT) error {
	return boolResult(procGetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(placement))))
}

// GetWindowRect returns a window's rectangle in screen coordinates.
func GetWindowRect(hwnd HWND, rect *RECT) error {
	return boolResult(procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect))))
}

// InvalidateRect adds rect (nil for the whole client area) to a window's update region.
func InvalidateRect(hwnd HWND, rect *RECT, erase bool) bool {
	r, _, _ := procInvalidateRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)), uintptr(Bool(erase)))
	return r != 0
}

// IsClipboardFormatAvailable reports whether the clipboard holds format.
func IsClipboardFormatAvailable(format uint32) error {
	return boolResult(procIsClipboardFormatAvailable.Call(uintptr(format)))
}

// IsIconic reports whether a window is minimized.
func IsIconic(hwnd HWND) bool {
	r, _, _ := procIsIconic.Call(uintptr(hwnd))
	return r != 0
}

// IsWindow reports whether hwnd identifies a window.
func IsWindow(hwnd HWND) bool {
	r, _, _ := procIsWindow.Call(uintptr(hwnd))
	return r != 0
}

// IsWindowEnabled reports whether a window accepts input.
func IsWindowEnabled(hwnd HWND) bool {
	r, _, _ := procIsWindowEnabled.Call(uintptr(hwnd))
	return r != 0
}

// IsWindowVisible reports whether a window is visible.
func IsWindowVisible(hwnd HWND) bool {
	r, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return r != 0
}

// IsZoomed reports whether a window is maximized.
func IsZoomed(hwnd HWND) bool {
	r, _, _ := procIsZoomed.Call(uintptr(hwnd))
	return r != 0
}

// KillTimer destroys a timer.
func KillTimer(hwnd HWND, id uintptr) error {
	return boolResult(procKillTimer.Call(uintptr(hwnd), id))
}

// LoadImageW loads an icon, cursor or bitmap; name may be a MakeIntResource id.
func LoadImageW(instance HINSTANCE, name *uint16, imageType uint32, cx, cy int32, flags uint32) (HANDLE, error) {
	r, _, err := procLoadImageW.Call(uintptr(instance), uintptr(unsafe.Pointer(name)), uintptr(imageType), uintptr(cx), uintptr(cy), uintptr(flags))
	if r == 0 {
		return 0, lastError(err)
	}
	return HANDLE(r), nil
}

// MapVirtualKeyW translates between virtual keys, scan codes and characters.
func MapVirtualKeyW(code uint32, mapType uint32) uint32 {
	r, _, _ := procMapVirtualKeyW.Call(uintptr(code), uintptr(mapType))
	return uint32(r)
}

// MessageBeep plays a system sound.
func MessageBeep(beepType uint32) error {
	return boolResult(procMessageBeep.Call(uintptr(beepType)))
}

// MonitorFromPoint returns the monitor containing pt.
func MonitorFromPoint(pt POINT, flags uint32) HMONITOR {
	r, _, _ := procMonitorFromPoint.Call(PackPoint(pt), uintptr(flags))
	return HMONITOR(r)
}

// MonitorFromPointLastError is MonitorFromPoint plus the thread's last error captured by the same call.
func MonitorFromPointLastError(pt POINT, flags uint32) (HMONITOR, error) {
	r, _, lastErr := procMonitorFromPoint.Call(PackPoint(pt), uintptr(flags))
	return HMONITOR(r), lastErr
}

// MonitorFromWindow returns the monitor with the largest intersection with a window.
func MonitorFromWindow(hwnd HWND, flags uint32) HMONITOR {
	r, _, _ := procMonitorFromWindow.Call(uintptr(hwnd), uintptr(flags))
	return HMONITOR(r)
}

// MsgWaitForMultipleObjects waits for handles or input; it returns WAIT_FAILED on error.
func MsgWaitForMultipleObjects(handles []HANDLE, waitAll bool, milliseconds uint32, wakeMask uint32) uint32 {
	var first *HANDLE
	if len(handles) > 0 {
		first = &handles[0]
	}
	r, _, _ := procMsgWaitForMultipleObjects.Call(uintptr(len(handles)), uintptr(unsafe.Pointer(first)), uintptr(Bool(waitAll)), uintptr(milliseconds), uintptr(wakeMask))
	return uint32(r)
}

// OpenClipboard opens the clipboard for owner.
func OpenClipboard(owner HWND) error {
	return boolResult(procOpenClipboard.Call(uintptr(owner)))
}

// PeekMessageW retrieves a message if one is available.
func PeekMessageW(msg *MSG, hwnd HWND, filterMin, filterMax uint32, removeMsg uint32) bool {
	r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(msg)), uintptr(hwnd), uintptr(filterMin), uintptr(filterMax), uintptr(removeMsg))
	return r != 0
}

// PostMessageW posts a message to a window's queue.
func PostMessageW(hwnd HWND, msg uint32, wparam WPARAM, lparam LPARAM) error {
	return boolResult(procPostMessageW.Call(uintptr(hwnd), uintptr(msg), uintptr(wparam), uintptr(lparam)))
}

// PostQuitMessage posts WM_QUIT.
func PostQuitMessage(exitCode int32) {
	procPostQuitMessage.Call(uintptr(exitCode))
}

// PostThreadMessageW posts a message to a thread's queue.
func PostThreadMessageW(threadID uint32, msg uint32, wparam WPARAM, lparam LPARAM) error {
	return boolResult(procPostThreadMessageW.Call(uintptr(threadID), uintptr(msg), uintptr(wparam), uintptr(lparam)))
}

// RedrawWindow updates a window's client area.
func RedrawWindow(hwnd HWND, update *RECT, region uintptr, flags uint32) bool {
	r, _, _ := procRedrawWindow.Call(uintptr(hwnd), uintptr(unsafe.Pointer(update)), region, uintptr(flags))
	return r != 0
}

// RegisterClassW registers a window class.
func RegisterClassW(class *WNDCLASSW) (ATOM, error) {
	r, _, err := procRegisterClassW.Call(uintptr(unsafe.Pointer(class)))
	if r == 0 {
		return 0, lastError(err)
	}
	return ATOM(r), nil
}

// RegisterClipboardFormatW registers a clipboard format by name.
func RegisterClipboardFormatW(name *uint16) (uint32, error) {
	r, _, err := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, lastError(err)
	}
	return uint32(r), nil
}

// RegisterPowerSettingNotification registers for power setting events.
func RegisterPowerSettingNotification(recipient HANDLE, setting *GUID, flags uint32) (HPOWERNOTIFY, error) {
	r, _, err := procRegisterPowerSettingNotification.Call(uintptr(recipient), uintptr(unsafe.Pointer(setting)), uintptr(flags))
	if r == 0 {
		return 0, lastError(err)
	}
	return HPOWERNOTIFY(r), nil
}

// RegisterSuspendResumeNotification registers for suspend and resume events.
func RegisterSuspendResumeNotification(recipient HANDLE, flags uint32) (HPOWERNOTIFY, error) {
	r, _, err := procRegisterSuspendResumeNotification.Call(uintptr(recipient), uintptr(flags))
	if r == 0 {
		return 0, lastError(err)
	}
	return HPOWERNOTIFY(r), nil
}

// ReleaseCapture releases the mouse capture.
func ReleaseCapture() error {
	return boolResult(procReleaseCapture.Call())
}

// RemovePropW removes a window property and returns its value.
func RemovePropW(hwnd HWND, name *uint16) (HANDLE, error) {
	r, _, err := procRemovePropW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, lastError(err)
	}
	return HANDLE(r), nil
}

// ScreenToClient converts a screen point to client coordinates.
func ScreenToClient(hwnd HWND, point *POINT) bool {
	r, _, _ := procScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(point)))
	return r != 0
}

// SendInput synthesizes input events and returns how many were inserted.
func SendInput(inputs []INPUT) uint32 {
	if len(inputs) == 0 {
		return 0
	}
	r, _, _ := procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(INPUT{}))
	return uint32(r)
}

// SendMessageW sends a message and waits for its result.
func SendMessageW(hwnd HWND, msg uint32, wparam WPARAM, lparam LPARAM) LRESULT {
	r, _, _ := procSendMessageW.Call(uintptr(hwnd), uintptr(msg), uintptr(wparam), uintptr(lparam))
	return LRESULT(r)
}

// SetActiveWindow activates a window of the calling thread.
func SetActiveWindow(hwnd HWND) (HWND, error) {
	r, _, err := procSetActiveWindow.Call(uintptr(hwnd))
	if r == 0 {
		return 0, lastError(err)
	}
	return HWND(r), nil
}

// SetCapture captures the mouse for a window.
func SetCapture(hwnd HWND) HWND {
	r, _, _ := procSetCapture.Call(uintptr(hwnd))
	return HWND(r)
}

// SetClipboardData places data on the clipboard; the system owns it on success.
func SetClipboardData(format uint32, data HANDLE) (HANDLE, error) {
	r, _, err := procSetClipboardData.Call(uintptr(format), uintptr(data))
	if r == 0 {
		return 0, lastError(err)
	}
	return HANDLE(r), nil
}

// SetCursor sets the cursor shape and returns the previous one.
func SetCursor(cursor HCURSOR) HCURSOR {
	r, _, _ := procSetCursor.Call(uintptr(cursor))
	return HCURSOR(r)
}

// SetFocus gives a window keyboard focus.
func SetFocus(hwnd HWND) (HWND, error) {
	r, _, err := procSetFocus.Call(uintptr(hwnd))
	if r == 0 {
		return 0, lastError(err)
	}
	return HWND(r), nil
}

// SetForegroundWindow brings a window's thread to the foreground.
func SetForegroundWindow(hwnd HWND) bool {
	r, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	return r != 0
}

// SetProcessDpiAwarenessContext sets the process's DPI awareness.
func SetProcessDpiAwarenessContext(context uintptr) error {
	return boolResult(procSetProcessDpiAwarenessContext.Call(context))
}

// GetThreadDpiAwarenessContext returns the calling thread's DPI awareness context.
func GetThreadDpiAwarenessContext() uintptr {
	r, _, _ := procGetThreadDpiAwarenessContext.Call()
	return r
}

// AreDpiAwarenessContextsEqual reports whether two DPI awareness contexts are equal.
func AreDpiAwarenessContextsEqual(a, b uintptr) bool {
	r, _, _ := procAreDpiAwarenessContextsEqual.Call(a, b)
	return r != 0
}

// SetPropW sets a window property.
func SetPropW(hwnd HWND, name *uint16, data HANDLE) error {
	return boolResult(procSetPropW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(name)), uintptr(data)))
}

// SetTimer creates or replaces a timer; callback is 0 or a windows.NewCallback TIMERPROC.
func SetTimer(hwnd HWND, id uintptr, elapse uint32, callback uintptr) uintptr {
	r, _, _ := procSetTimer.Call(uintptr(hwnd), id, uintptr(elapse), callback)
	return r
}

// SetWindowLongPtrW writes a window's pointer-sized value at index and returns the old one.
func SetWindowLongPtrW(hwnd HWND, index int32, value uintptr) uintptr {
	r, _, _ := procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(index), value)
	return r
}

// SetWindowLongW writes a window's 32-bit value at index and returns the old one.
func SetWindowLongW(hwnd HWND, index int32, value int32) int32 {
	r, _, _ := procSetWindowLongW.Call(uintptr(hwnd), uintptr(index), uintptr(value))
	return int32(r)
}

// SetWindowPlacement sets a window's placement.
func SetWindowPlacement(hwnd HWND, placement *WINDOWPLACEMENT) error {
	return boolResult(procSetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(placement))))
}

// SetWindowPos changes a window's size, position and Z order.
func SetWindowPos(hwnd HWND, insertAfter HWND, x, y, cx, cy int32, flags uint32) error {
	return boolResult(procSetWindowPos.Call(uintptr(hwnd), uintptr(insertAfter), uintptr(x), uintptr(y), uintptr(cx), uintptr(cy), uintptr(flags)))
}

// SetWindowTextW sets a window's title.
func SetWindowTextW(hwnd HWND, text *uint16) error {
	return boolResult(procSetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(text))))
}

// ShowWindow sets a window's show state and reports whether it was visible.
func ShowWindow(hwnd HWND, cmd int32) bool {
	r, _, _ := procShowWindow.Call(uintptr(hwnd), uintptr(cmd))
	return r != 0
}

// ShowWindowAsync sets the show state of a window of another thread.
func ShowWindowAsync(hwnd HWND, cmd int32) bool {
	r, _, _ := procShowWindowAsync.Call(uintptr(hwnd), uintptr(cmd))
	return r != 0
}

// SystemParametersInfoW reads or writes a system-wide parameter.
func SystemParametersInfoW(action uint32, uiParam uint32, pvParam unsafe.Pointer, winIni uint32) error {
	return boolResult(procSystemParametersInfoW.Call(uintptr(action), uintptr(uiParam), uintptr(pvParam), uintptr(winIni)))
}

// ToUnicode translates a virtual key and keyboard state to UTF-16 characters.
func ToUnicode(virtKey uint32, scanCode uint32, keyState *[256]byte, buf []uint16, flags uint32) int32 {
	var first *uint16
	if len(buf) > 0 {
		first = &buf[0]
	}
	r, _, _ := procToUnicode.Call(uintptr(virtKey), uintptr(scanCode), uintptr(unsafe.Pointer(keyState)), uintptr(unsafe.Pointer(first)), uintptr(len(buf)), uintptr(flags))
	return int32(r)
}

// TrackMouseEvent requests mouse leave or hover notifications.
func TrackMouseEvent(event *TRACKMOUSEEVENT) error {
	return boolResult(procTrackMouseEvent.Call(uintptr(unsafe.Pointer(event))))
}

// TranslateMessage generates character messages from virtual-key messages.
func TranslateMessage(msg *MSG) bool {
	r, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
	return r != 0
}

// UnregisterClassW unregisters a window class.
func UnregisterClassW(className *uint16, instance HINSTANCE) error {
	return boolResult(procUnregisterClassW.Call(uintptr(unsafe.Pointer(className)), uintptr(instance)))
}

// UnregisterPowerSettingNotification unregisters a power setting notification.
func UnregisterPowerSettingNotification(handle HPOWERNOTIFY) error {
	return boolResult(procUnregisterPowerSettingNotification.Call(uintptr(handle)))
}

// UnregisterSuspendResumeNotification unregisters a suspend and resume notification.
func UnregisterSuspendResumeNotification(handle HPOWERNOTIFY) error {
	return boolResult(procUnregisterSuspendResumeNotification.Call(uintptr(handle)))
}

// UpdateWindow sends WM_PAINT if the update region is not empty.
func UpdateWindow(hwnd HWND) bool {
	r, _, _ := procUpdateWindow.Call(uintptr(hwnd))
	return r != 0
}

// ValidateRect removes rect (nil for the whole client area) from a window's update region.
func ValidateRect(hwnd HWND, rect *RECT) bool {
	r, _, _ := procValidateRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
	return r != 0
}

// WaitMessage waits until the thread's queue has a new message.
func WaitMessage() error {
	return boolResult(procWaitMessage.Call())
}

// WindowFromPoint returns the window at a screen point.
func WindowFromPoint(pt POINT) HWND {
	r, _, _ := procWindowFromPoint.Call(PackPoint(pt))
	return HWND(r)
}
