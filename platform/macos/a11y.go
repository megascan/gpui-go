//go:build darwin && !ios

package macos

import (
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

type a11yQueuedEvents interface {
	Raise()
}

type a11yAdapter interface {
	UpdateIfActive(updateFactory func() a11y.TreeUpdate) (a11yQueuedEvents, bool)
	UpdateViewFocusState(isFocused bool) (a11yQueuedEvents, bool)
	Close()
}

func newA11yAdapter(window objc.ID, activationHandler *a11yActivationHandler, actionHandler *a11yActionHandler) a11yAdapter {
	return nil
}
