//go:build windows

package windows

import "testing"

func TestWrapper_SafeHandles(t *testing.T) {
	h := SafeHwndFrom(HWND(42))
	if h.AsRaw() != 42 || h.Deref() != 42 {
		t.Fatal("SafeHwnd lost its handle")
	}
	if SafeCursorFrom(7).Deref() != 7 {
		t.Fatal("SafeCursor lost its handle")
	}
}
