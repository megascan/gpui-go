//go:build windows

package windows

import "github.com/megascan/gpui-go/internal/win/win32"

// SafeCursor is an HCURSOR that may be shared between threads.
type SafeCursor struct {
	raw win32.HCURSOR
}

// SafeCursorFrom wraps a cursor handle.
func SafeCursorFrom(value win32.HCURSOR) SafeCursor {
	return SafeCursor{raw: value}
}

// Deref returns the cursor handle.
func (c SafeCursor) Deref() win32.HCURSOR {
	return c.raw
}

// SafeHwnd is an HWND that may be shared between threads.
type SafeHwnd struct {
	raw HWND
}

// SafeHwndFrom wraps a window handle.
func SafeHwndFrom(value HWND) SafeHwnd {
	return SafeHwnd{raw: value}
}

// AsRaw returns the window handle.
func (h SafeHwnd) AsRaw() HWND {
	return h.raw
}

// Deref returns the window handle.
func (h SafeHwnd) Deref() HWND {
	return h.raw
}
