//go:build windows

package windows

import (
	"github.com/megascan/gpui-go/a11y"
	a11ywin "github.com/megascan/gpui-go/a11y/windows"
	"github.com/megascan/gpui-go/internal/win/win32"
)

type a11yCell struct {
	value    *A11yState
	borrowed bool
}

func newA11yAdapter(hwnd HWND, isFocused bool, actionHandler *A11yActionHandler) *a11ywin.Adapter {
	return a11ywin.NewAdapter(hwnd, isFocused, actionHandler)
}

func (s *A11yState) updateIfActive(update func() a11y.TreeUpdate) (*a11ywin.QueuedEvents, bool) {
	return s.adapter.UpdateIfActive(update)
}

func (s *A11yState) updateWindowFocusState(activated bool) (*a11ywin.QueuedEvents, bool) {
	return s.adapter.UpdateWindowFocusState(activated)
}

func (s *A11yState) handleWMGetObject(wparam win32.WPARAM, lparam win32.LPARAM) (func() win32.LRESULT, bool) {
	return s.adapter.HandleWmGetobject(wparam, lparam, s.activationHandler)
}

func (s *A11yState) release() {
	s.adapter.Release()
}

func raiseA11yEvents(events *a11ywin.QueuedEvents) {
	events.Raise()
}
