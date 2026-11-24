//go:build windows

package windows

import (
	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/win32"
)

// HWND is a window handle.
type HWND = windows.HWND

// WPARAM is a window message's first parameter.
type WPARAM = win32.WPARAM

// LPARAM is a window message's second parameter.
type LPARAM = win32.LPARAM

// LRESULT is a window procedure's result.
type LRESULT = win32.LRESULT
