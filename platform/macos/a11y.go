//go:build darwin && !ios

package macos

import (
	"github.com/megascan/gpui-go/a11y"
	a11ymacos "github.com/megascan/gpui-go/a11y/macos"
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

type subclassingA11yAdapter struct {
	inner *a11ymacos.SubclassingAdapter
}

func newA11yAdapter(window objc.ID, activationHandler *a11yActivationHandler, actionHandler *a11yActionHandler) a11yAdapter {
	return &subclassingA11yAdapter{
		inner: a11ymacos.SubclassingAdapterForWindow(window, activationHandler, actionHandler),
	}
}

func (a *subclassingA11yAdapter) UpdateIfActive(updateFactory func() a11y.TreeUpdate) (a11yQueuedEvents, bool) {
	events, ok := a.inner.UpdateIfActive(updateFactory)
	if !ok {
		return nil, false
	}
	return events, true
}

func (a *subclassingA11yAdapter) UpdateViewFocusState(isFocused bool) (a11yQueuedEvents, bool) {
	events, ok := a.inner.UpdateViewFocusState(isFocused)
	if !ok {
		return nil, false
	}
	return events, true
}

func (a *subclassingA11yAdapter) Close() {
	a.inner.Release()
}
