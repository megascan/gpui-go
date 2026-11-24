//go:build windows

package win32

import "unsafe"

var (
	procDwmExtendFrameIntoClientArea = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
	procDwmFlush                     = dwmapi.NewProc("DwmFlush")
	procDwmGetCompositionTimingInfo  = dwmapi.NewProc("DwmGetCompositionTimingInfo")
	procDwmGetWindowAttribute        = dwmapi.NewProc("DwmGetWindowAttribute")
	procDwmSetWindowAttribute        = dwmapi.NewProc("DwmSetWindowAttribute")
)

// DwmExtendFrameIntoClientArea extends the window frame into the client area.
func DwmExtendFrameIntoClientArea(hwnd HWND, margins *MARGINS) error {
	r, _, _ := procDwmExtendFrameIntoClientArea.Call(uintptr(hwnd), uintptr(unsafe.Pointer(margins)))
	return hresult(r)
}

// DwmFlush waits for the next DWM composition pass.
func DwmFlush() error {
	r, _, _ := procDwmFlush.Call()
	return hresult(r)
}

// DwmGetCompositionTimingInfo reads composition timing for hwnd, or for the desktop when it is 0.
func DwmGetCompositionTimingInfo(hwnd HWND, info *DWM_TIMING_INFO) error {
	r, _, _ := procDwmGetCompositionTimingInfo.Call(uintptr(hwnd), uintptr(unsafe.Pointer(info)))
	return hresult(r)
}

// DwmGetWindowAttribute reads a DWM window attribute.
func DwmGetWindowAttribute(hwnd HWND, attribute uint32, value unsafe.Pointer, size uint32) error {
	r, _, _ := procDwmGetWindowAttribute.Call(uintptr(hwnd), uintptr(attribute), uintptr(value), uintptr(size))
	return hresult(r)
}

// DwmSetWindowAttribute writes a DWM window attribute.
func DwmSetWindowAttribute(hwnd HWND, attribute uint32, value unsafe.Pointer, size uint32) error {
	r, _, _ := procDwmSetWindowAttribute.Call(uintptr(hwnd), uintptr(attribute), uintptr(value), uintptr(size))
	return hresult(r)
}
