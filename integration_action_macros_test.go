package gpui_test

import (
	"reflect"
	"testing"

	"encoding/json/jsontext"

	"github.com/megascan/gpui-go"
)

type someAction struct{}

func (someAction) Name() string                       { return "test_only::SomeAction" }
func (a someAction) BoxedClone() gpui.Action          { return a }
func (someAction) PartialEq(other gpui.Action) bool   { _, ok := other.(someAction); return ok }
func (someActionWithDocs) Name() string               { return "test_only::SomeActionWithDocs" }
func (a someActionWithDocs) BoxedClone() gpui.Action  { return a }
func (anotherAction) Name() string                    { return "test_only::AnotherAction" }
func (a anotherAction) BoxedClone() gpui.Action       { return a }
func (registerableAction) Name() string               { panic("not implemented") }
func (registerableAction) BoxedClone() gpui.Action    { panic("not implemented") }
func (registerableAction) PartialEq(gpui.Action) bool { panic("not implemented") }

type someActionWithDocs struct{}

func (someActionWithDocs) PartialEq(other gpui.Action) bool {
	_, ok := other.(someActionWithDocs)
	return ok
}

type anotherAction struct{}

func (anotherAction) PartialEq(other gpui.Action) bool {
	_, ok := other.(anotherAction)
	return ok
}

type registerableAction struct{}

func unitActionData[A gpui.Action](name string, value A, docs string) gpui.ActionData {
	return gpui.ActionData{
		Name: name,
		Type: reflect.TypeFor[A](),
		Build: func(jsontext.Value) (gpui.Action, error) {
			return value, nil
		},
		Documentation: docs,
	}
}

func init() {
	gpui.RegisterAction(unitActionData("test_only::SomeAction", someAction{}, ""))
	gpui.RegisterAction(unitActionData("test_only::SomeActionWithDocs", someActionWithDocs{}, "Documented action"))
	gpui.RegisterAction(unitActionData("test_only::AnotherAction", anotherAction{}, ""))
	gpui.RegisterAction(gpui.ActionData{
		Name: "integration_action_macros::RegisterableAction",
		Type: reflect.TypeFor[registerableAction](),
		Build: func(jsontext.Value) (gpui.Action, error) {
			panic("not implemented")
		},
	})
}

func TestActionMacros_ActionMacros(t *testing.T) {
	var _ gpui.Action = someAction{}
	var _ gpui.Action = someActionWithDocs{}
	var _ gpui.Action = anotherAction{}
	var _ gpui.Action = registerableAction{}

	gpui.RunTest(t, func(tc *gpui.TestRun) {
		tc.Cx().Update(func(cx *gpui.App) {
			for _, want := range []gpui.Action{someAction{}, someActionWithDocs{}, anotherAction{}} {
				got, err := cx.BuildAction(want.Name(), gpui.Option[jsontext.Value]{})
				if err != nil {
					t.Fatalf("BuildAction(%q): %v", want.Name(), err)
				}
				if !got.PartialEq(want) {
					t.Fatalf("BuildAction(%q) = %#v", want.Name(), got)
				}
			}
		})
	})
}
