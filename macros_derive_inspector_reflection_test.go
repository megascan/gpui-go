package gpui_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/megascan/gpui-go"
)

type transform interface {
	Double() inspectorNumber
	Triple() inspectorNumber
	Increment() inspectorNumber
	Quadruple() inspectorNumber
	Add(other *inspectorNumber) inspectorNumber
	GetValue() int32
	AddOne() inspectorNumber
}

var transformDocs = map[string]string{
	"Double":    "Doubles the value",
	"Triple":    "Triples the value",
	"Increment": "Increments the value by one\n\nThis method has a default implementation",
	"Quadruple": "Quadruples the value by doubling twice",
	"AddOne":    "Adds one to the value",
}

type inspectorNumber struct{ value int32 }

func (n inspectorNumber) Double() inspectorNumber    { return inspectorNumber{n.value * 2} }
func (n inspectorNumber) Triple() inspectorNumber    { return inspectorNumber{n.value * 3} }
func (n inspectorNumber) Increment() inspectorNumber { return n.AddOne() }
func (n inspectorNumber) Quadruple() inspectorNumber { return n.Double().Double() }
func (n inspectorNumber) Add(other *inspectorNumber) inspectorNumber {
	return inspectorNumber{n.value + other.value}
}
func (n *inspectorNumber) SetValue(value int32)      { n.value = value }
func (n inspectorNumber) GetValue() int32            { return n.value }
func (n inspectorNumber) AddOne() inspectorNumber    { return inspectorNumber{n.value + 1} }
func (n inspectorNumber) Unrelated() inspectorNumber { return n }

func findTransform(name string) (gpui.FunctionReflection[inspectorNumber], bool) {
	return gpui.InspectorReflectionFindMethod[inspectorNumber](reflect.TypeFor[transform](), transformDocs, name)
}

func TestDeriveInspectorReflection_DeriveInspectorReflection(t *testing.T) {
	methods := gpui.InspectorReflectionMethods[inspectorNumber](reflect.TypeFor[transform](), transformDocs)

	if len(methods) != 5 {
		t.Fatalf("len(methods) = %d, want 5", len(methods))
	}
	var methodNames []string
	for _, m := range methods {
		methodNames = append(methodNames, m.Name)
	}
	for _, name := range []string{"Double", "Triple", "Increment", "Quadruple", "AddOne"} {
		if !slices.Contains(methodNames, name) {
			t.Fatalf("methods %v do not contain %s", methodNames, name)
		}
	}

	num := inspectorNumber{5}
	invoke := func(name string, value inspectorNumber) inspectorNumber {
		m, ok := findTransform(name)
		if !ok {
			t.Fatalf("method %s not found", name)
		}
		return m.Invoke(value)
	}
	if got := invoke("Double", num); got != (inspectorNumber{10}) {
		t.Fatalf("Double = %v", got)
	}
	if got := invoke("Triple", num); got != (inspectorNumber{15}) {
		t.Fatalf("Triple = %v", got)
	}
	if got := invoke("Increment", num); got != (inspectorNumber{6}) {
		t.Fatalf("Increment = %v", got)
	}
	if got := invoke("Quadruple", num); got != (inspectorNumber{20}) {
		t.Fatalf("Quadruple = %v", got)
	}

	if _, ok := findTransform("nonexistent"); ok {
		t.Fatal("found nonexistent method")
	}

	result := gpui.None[inspectorNumber]()
	if m, ok := findTransform("Double"); ok {
		n := m.Invoke(inspectorNumber{10})
		if m, ok := findTransform("Increment"); ok {
			n = m.Invoke(n)
			if m, ok := findTransform("Triple"); ok {
				result = gpui.Some(m.Invoke(n))
			}
		}
	}
	if got, ok := result.Get(); !ok || got != (inspectorNumber{63}) {
		t.Fatalf("chained result = %v, %v", got, ok)
	}

	for name, doc := range transformDocs {
		m, _ := findTransform(name)
		if got, ok := m.Documentation.Get(); !ok || got != doc {
			t.Fatalf("%s documentation = %q", name, got)
		}
	}
}

func TestDeriveInspectorReflection_StyledMethods(t *testing.T) {
	methods := gpui.StyledReflectionMethods[*gpui.Div]()
	if len(methods) < 100 {
		t.Fatalf("got %d Styled methods", len(methods))
	}
	flex, ok := gpui.StyledReflectionFindMethod[*gpui.Div]("Flex")
	if !ok {
		t.Fatal("Flex not found")
	}
	doc, ok := flex.Documentation.Get()
	if !ok || doc == "" {
		t.Fatal("Flex has no documentation")
	}
	div := gpui.NewDiv()
	if got := flex.Invoke(div); got != div {
		t.Fatal("Flex did not return the element")
	}
	if display, ok := div.Style().Display.Get(); !ok || display != gpui.DisplayFlex {
		t.Fatalf("display = %v, %v", display, ok)
	}
	if _, ok := gpui.StyledReflectionFindMethod[*gpui.Div]("OnClick"); ok {
		t.Fatal("OnClick is not a Styled method")
	}
}
