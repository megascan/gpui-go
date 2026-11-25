package gpui

import (
	"encoding/json/jsontext"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type testActionStruct struct{}

func (testActionStruct) Name() string                  { return "test::Struct" }
func (a testActionStruct) BoxedClone() Action          { return a }
func (a testActionStruct) PartialEq(other Action) bool { _, ok := other.(testActionStruct); return ok }

func TestAction_Registry(t *testing.T) {
	r := newActionRegistry()
	a, err := r.buildAction("zed::NoAction", None[jsontext.Value]())
	if err != nil || !IsNoAction(a) {
		t.Fatalf("NoAction: %v %v", a, err)
	}
	a, err = r.buildAction("zed::Unbind", Some(jsontext.Value(`"editor::NewLine"`)))
	if err != nil || !IsUnbind(a) || a.(Unbind) != "editor::NewLine" {
		t.Fatalf("Unbind: %v %v", a, err)
	}
	if !a.PartialEq(Unbind("editor::NewLine")) || a.PartialEq(NoAction{}) {
		t.Fatal("PartialEq")
	}
	_, err = r.buildAction("zed::Unbind", None[jsontext.Value]())
	var be *ActionBuildErrorBuildError
	if !errors.As(err, &be) || !strings.HasPrefix(err.Error(), "Error while building action \"zed::Unbind\": ") {
		t.Fatalf("build error: %v", err)
	}
	_, err = r.buildAction("missing", None[jsontext.Value]())
	if err == nil || err.Error() != `Didn't find an action named "missing"` {
		t.Fatalf("not found: %v", err)
	}
	if a, err := r.buildActionType(reflect.TypeFor[NoAction]()); err != nil || !IsNoAction(a) {
		t.Fatalf("by type: %v %v", a, err)
	}
	if _, err := r.buildActionType(reflect.TypeFor[testActionStruct]()); err == nil || !strings.HasPrefix(err.Error(), "no action type registered for ") {
		t.Fatalf("by type missing: %v", err)
	}
	if s, ok := r.actionSchemaByName("zed::Unbind"); !ok || s == nil {
		t.Fatal("Unbind schema")
	}
	if s, ok := r.actionSchemaByName("zed::NoAction"); !ok || s != nil {
		t.Fatal("NoAction schema")
	}
	if r.documentationMap()["zed::NoAction"] == "" {
		t.Fatal("documentation")
	}
}

func TestAction_InsertActionPanics(t *testing.T) {
	expectPanic := func(want string, f func()) {
		t.Helper()
		defer func() {
			if got := recover(); got != want {
				t.Errorf("panic %v, want %v", got, want)
			}
		}()
		f()
	}
	r := &actionRegistry{byName: map[string]registeredAction{}, namesByType: map[reflect.Type]string{},
		deprecatedAliases: map[string]string{}, deprecationMessages: map[string]string{}, documentation: map[string]string{}}
	data := ActionData{Name: "test::Struct", Type: reflect.TypeFor[testActionStruct](), DeprecatedAliases: []string{"test::Old"}, DeprecationMessage: "gone"}
	r.insertAction(data)
	if r.deprecatedAliasesMap()["test::Old"] != "test::Struct" || r.deprecationMessagesMap()["test::Struct"] != "gone" {
		t.Fatal("aliases")
	}
	if got := r.allActionNames(); len(got) != 2 || got[0] != "test::Old" || got[1] != "test::Struct" {
		t.Fatalf("names %v", got)
	}
	expectPanic("Action with name `test::Struct` already registered (might be registered in `#[action(deprecated_aliases = [...])]`.", func() {
		r.insertAction(data)
	})
	expectPanic("Action with name `test::Old` already registered. `test::Old` is specified in `#[action(deprecated_aliases = [...])]` for action `test::New`.", func() {
		r.insertAction(ActionData{Name: "test::New", DeprecatedAliases: []string{"test::Old"}})
	})
	found := false
	for a := range GenerateListOfAllRegisteredActions() {
		if a.Name == "zed::Unbind" {
			found = true
		}
	}
	if !found {
		t.Fatal("inventory")
	}
}

func TestAction_BuildErrorUnwrap(t *testing.T) {
	inner := errors.New("inner")
	err := error(&ActionBuildErrorBuildError{Name: "x", Err: inner})
	if !errors.Is(err, inner) {
		t.Fatal("build error does not unwrap to its source")
	}
}
