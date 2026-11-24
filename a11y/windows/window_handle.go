//go:build windows

package windows

import "golang.org/x/sys/windows"

// WindowHandle is an HWND that may be shared between threads.
type WindowHandle struct {
	HWND windows.HWND
}

// WindowHandleFromHWND wraps hwnd.
func WindowHandleFromHWND(hwnd windows.HWND) WindowHandle {
	return WindowHandle{HWND: hwnd}
}
