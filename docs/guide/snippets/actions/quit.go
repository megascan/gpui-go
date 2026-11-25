package actions

import (
	"encoding/json/jsontext"
	"reflect"

	"github.com/megascan/gpui-go"
)

// Quit quits the application.
type Quit struct{}

// Name returns the action name.
func (Quit) Name() string { return "app::Quit" }

// BoxedClone clones the action.
func (Quit) BoxedClone() gpui.Action { return Quit{} }

// PartialEq reports whether other is a Quit.
func (Quit) PartialEq(other gpui.Action) bool {
	_, ok := other.(Quit)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "app::Quit",
		Type:  reflect.TypeFor[Quit](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Quit{}, nil },
	})
}
