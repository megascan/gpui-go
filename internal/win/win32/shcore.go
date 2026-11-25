//go:build windows

package win32

import "unsafe"

var (
	procGetDpiForMonitor       = shcore.NewProc("GetDpiForMonitor")
	procSetProcessDpiAwareness = shcore.NewProc("SetProcessDpiAwareness")
)

// GetDpiForMonitor returns a monitor's horizontal and vertical DPI of the given type.
func GetDpiForMonitor(monitor HMONITOR, dpiType int32, dpiX, dpiY *uint32) error {
	r, _, _ := procGetDpiForMonitor.Call(uintptr(monitor), uintptr(dpiType), uintptr(unsafe.Pointer(dpiX)), uintptr(unsafe.Pointer(dpiY)))
	return hresult(r)
}

// SetProcessDpiAwareness sets the process's DPI awareness (Windows 8.1 API).
func SetProcessDpiAwareness(value int32) error {
	r, _, _ := procSetProcessDpiAwareness.Call(uintptr(value))
	return hresult(r)
}
