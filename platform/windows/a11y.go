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
	return nil
}

func (s *A11yState) updateIfActive(update func() a11y.TreeUpdate) (*a11ywin.QueuedEvents, bool) {
	if s.adapter == nil {
		return nil, false
	}
	return nil, false
}

func (s *A11yState) updateWindowFocusState(activated bool) (*a11ywin.QueuedEvents, bool) {
	if s.adapter == nil {
		return nil, false
	}
	return nil, false
}

func (s *A11yState) handleWMGetObject(wparam win32.WPARAM, lparam win32.LPARAM) (func() win32.LRESULT, bool) {
	if s.adapter == nil {
		return nil, false
	}
	return nil, false
}

func raiseA11yEvents(events *a11ywin.QueuedEvents) {}
