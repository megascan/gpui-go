//go:build linux

package wayland

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/atspi/bus"
)

type a11yAdapter interface {
	UpdateWindowFocusState(isFocused bool)
	UpdateIfActive(updateFactory func() a11y.TreeUpdate)
	Release()
}

type trivialActivationHandler struct {
	callback func() (a11y.TreeUpdate, bool)
}

func (h trivialActivationHandler) RequestInitialTree() (a11y.TreeUpdate, bool) {
	return h.callback()
}

type trivialActionHandler func(request a11y.ActionRequest)

func (h trivialActionHandler) DoAction(request a11y.ActionRequest) {
	h(request)
}

type trivialDeactivationHandler struct {
	callback func()
}

func (h trivialDeactivationHandler) DeactivateAccessibility() {
	h.callback()
}

func newA11yAdapter(callbacks gpui.A11yCallbacks) a11yAdapter {
	activationHandler := trivialActivationHandler{callback: callbacks.Activation}
	actionHandler := trivialActionHandler(callbacks.Action)
	deactivationHandler := trivialDeactivationHandler{callback: callbacks.Deactivation}
	return bus.NewAdapter(activationHandler, actionHandler, deactivationHandler)
}
