//go:build linux

package x11

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/a11y"
)

type a11yAdapter interface {
	UpdateWindowFocusState(isFocused bool)
	UpdateIfActive(updateFactory func() a11y.TreeUpdate)
	SetRootWindowBounds(outer, inner a11y.Rect)
}

func newA11yAdapter(callbacks gpui.A11yCallbacks) a11yAdapter {
	return nil
}
