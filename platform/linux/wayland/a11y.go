//go:build linux

package wayland

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/a11y"
)

type a11yAdapter interface {
	UpdateIfActive(updateFactory func() a11y.TreeUpdate)
	UpdateWindowFocusState(isFocused bool)
}

func newA11yAdapter(callbacks gpui.A11yCallbacks) a11yAdapter {
	return nil
}
