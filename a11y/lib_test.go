package a11y

import (
	"encoding/json/v2"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/jsonschema"
)

var testTextDecoration = TextDecoration{
	Style: TextDecorationStyleDotted,
	Color: Color{Red: 0, Green: 0, Blue: 0, Alpha: 255},
}

func assertEmpty[T any](t *testing.T, v []T) {
	t.Helper()
	if len(v) != 0 {
		t.Fatalf("expected empty, got %v", v)
	}
}

func TestLib_U64ShouldBeConvertibleToNodeID(t *testing.T) {
	if NodeID(uint64(0)) != NodeID(0) || NodeID(uint64(1)) != NodeID(1) {
		t.Fatal("conversion failed")
	}
}

func TestLib_NodeIDShouldBeConvertibleToU64(t *testing.T) {
	if NodeIDContent(NodeID(0)) != 0 || NodeIDContent(NodeID(1)) != 1 {
		t.Fatal("conversion failed")
	}
}

func TestLib_NodeIDShouldHaveDebugRepr(t *testing.T) {
	if got := NodeID(0).String(); got != "#0" {
		t.Fatal(got)
	}
	if got := NodeID(1).String(); got != "#1" {
		t.Fatal(got)
	}
}

func TestLib_ActionNShouldReturnTheCorrespondingVariant(t *testing.T) {
	want := []Action{
		ActionClick, ActionFocus, ActionBlur, ActionCollapse, ActionExpand, ActionCustomAction,
		ActionDecrement, ActionIncrement, ActionHideTooltip, ActionShowTooltip, ActionReplaceSelectedText,
		ActionScrollDown, ActionScrollLeft, ActionScrollRight, ActionScrollUp, ActionScrollIntoView,
		ActionScrollToPoint, ActionSetScrollOffset, ActionSetTextSelection,
		ActionSetSequentialFocusNavigationStartingPoint, ActionSetValue, ActionShowContextMenu,
	}
	for i, w := range want {
		if got, ok := actionN(uint8(i)); !ok || got != w {
			t.Fatalf("actionN(%d) = %v, %v", i, got, ok)
		}
	}
	if _, ok := actionN(22); ok {
		t.Fatal("actionN(22) should be None")
	}
}

func TestLib_EmptyActionMaskShouldBeConvertedToEmptyVec(t *testing.T) {
	node := NewNode(RoleUnknown)
	if got := actionMaskToActionVec(node.actions); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestLib_ActionMaskShouldBeConvertibleToVec(t *testing.T) {
	cases := [][]Action{
		{ActionClick},
		{ActionShowContextMenu},
		{ActionClick, ActionShowContextMenu},
		{ActionFocus, ActionBlur, ActionCollapse},
	}
	for _, c := range cases {
		node := NewNode(RoleUnknown)
		for _, a := range c {
			node.AddAction(a)
		}
		if got := actionMaskToActionVec(node.actions); !slices.Equal(got, c) {
			t.Fatalf("got %v want %v", got, c)
		}
	}
}

func TestLib_NewNodeShouldHaveUserProvidedRole(t *testing.T) {
	node := NewNode(RoleButton)
	if node.Role() != RoleButton {
		t.Fatal(node.Role())
	}
}

func TestLib_NodeRoleSetterShouldUpdateTheRole(t *testing.T) {
	node := NewNode(RoleButton)
	node.SetRole(RoleCheckBox)
	if node.Role() != RoleCheckBox {
		t.Fatal(node.Role())
	}
}

func TestLib_NewNodeShouldNotSupportAnyaction(t *testing.T) {
	node := NewNode(RoleUnknown)
	for i := range len(actionNames) {
		a := Action(i)
		if node.SupportsAction(a) || node.ChildSupportsAction(a) {
			t.Fatalf("%v supported", a)
		}
	}
}

func TestLib_NodeAddActionShouldAddTheAction(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddAction(ActionFocus)
	if !node.SupportsAction(ActionFocus) {
		t.Fatal("focus")
	}
	node.AddAction(ActionBlur)
	if !node.SupportsAction(ActionBlur) {
		t.Fatal("blur")
	}
}

func TestLib_NodeAddChildActionShouldAddTheAction(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddChildAction(ActionFocus)
	if !node.ChildSupportsAction(ActionFocus) {
		t.Fatal("focus")
	}
	node.AddChildAction(ActionBlur)
	if !node.ChildSupportsAction(ActionBlur) {
		t.Fatal("blur")
	}
}

func TestLib_NodeAddActionShouldDoNothingIfTheActionIsAlreadySupported(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddAction(ActionFocus)
	node.AddAction(ActionFocus)
	if !node.SupportsAction(ActionFocus) {
		t.Fatal("focus")
	}
}

func TestLib_NodeAddChildActionShouldDoNothingIfTheActionIsAlreadySupported(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddChildAction(ActionFocus)
	node.AddChildAction(ActionFocus)
	if !node.ChildSupportsAction(ActionFocus) {
		t.Fatal("focus")
	}
}

func TestLib_NodeRemoveActionShouldRemoveTheAction(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddAction(ActionBlur)
	node.RemoveAction(ActionBlur)
	if node.SupportsAction(ActionBlur) {
		t.Fatal("blur")
	}
}

func TestLib_NodeRemoveChildActionShouldRemoveTheAction(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddChildAction(ActionBlur)
	node.RemoveChildAction(ActionBlur)
	if node.ChildSupportsAction(ActionBlur) {
		t.Fatal("blur")
	}
}

func TestLib_NodeClearActionsShouldRemoveAllActions(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddAction(ActionFocus)
	node.AddAction(ActionBlur)
	node.ClearActions()
	if node.SupportsAction(ActionFocus) || node.SupportsAction(ActionBlur) {
		t.Fatal("not cleared")
	}
}

func TestLib_NodeClearChildActionsShouldRemoveAllActions(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddChildAction(ActionFocus)
	node.AddChildAction(ActionBlur)
	node.ClearChildActions()
	if node.ChildSupportsAction(ActionFocus) || node.ChildSupportsAction(ActionBlur) {
		t.Fatal("not cleared")
	}
}

func TestLib_NodeShouldHaveDebugRepr(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.AddAction(ActionClick)
	node.AddAction(ActionFocus)
	node.AddChildAction(ActionScrollIntoView)
	node.SetHidden()
	node.SetMultiselectable()
	node.SetChildren([]NodeID{0, 1})
	node.SetActiveDescendant(2)
	node.PushCustomAction(CustomAction{ID: 0, Description: "test action"})
	want := `Node { role: Unknown, actions: [Click, Focus], child_actions: [ScrollIntoView], is_hidden: true, is_multiselectable: true, children: [#0, #1], active_descendant: #2, custom_actions: [CustomAction { id: 0, description: "test action" }] }`
	if got := node.String(); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestLib_NewTreeShouldHaveRootID(t *testing.T) {
	tree := NewTree(1)
	if tree.Root != 1 || tree.ToolkitName.IsSome() || tree.ToolkitVersion.IsSome() {
		t.Fatal(tree)
	}
}

func TestLib_ZeroNodeEqualsDefault(t *testing.T) {
	var zero Node
	def := NewNode(RoleUnknown)
	if !zero.Equal(&def) {
		t.Fatal("zero node differs from Node::default()")
	}
	if _, ok := zero.Label(); ok {
		t.Fatal("zero node has a label")
	}
}

func TestLib_NodeEqualityFollowsPartialEq(t *testing.T) {
	a := NewNode(RoleButton)
	a.SetLabel("x")
	a.SetNumericValue(1)
	b := NewNode(RoleButton)
	b.SetLabel("x")
	b.SetNumericValue(1)
	if !a.Equal(&b) {
		t.Fatal("equal nodes differ")
	}
	c := NewNode(RoleButton)
	c.SetNumericValue(1)
	c.SetLabel("x")
	if a.Equal(&c) {
		t.Fatal("insertion order is part of PartialEq")
	}
	d := a.Clone()
	d.ClearLabel()
	e := NewNode(RoleButton)
	e.SetNumericValue(1)
	if d.Equal(&e) {
		t.Fatal("a cleared slot is part of PartialEq")
	}
	if !a.Equal(&b) || a.Equal(&d) {
		t.Fatal("Clone aliased the original")
	}
	n := NewNode(RoleSlider)
	n.SetNumericValue(math.NaN())
	m := n.Clone()
	if n.Equal(&m) {
		t.Fatal("NaN must not equal NaN")
	}
}

func TestLib_PusherOnClearedPropertyDoesNothing(t *testing.T) {
	node := NewNode(RoleUnknown)
	node.SetChildren([]NodeID{0})
	node.ClearChildren()
	node.PushChild(1)
	assertEmpty(t, node.Children())
}

func TestLib_SetterCopiesInput(t *testing.T) {
	ids := []NodeID{1, 2}
	node := NewNode(RoleUnknown)
	node.SetChildren(ids)
	ids[0] = 9
	if got := node.Children(); got[0] != 1 {
		t.Fatal("setter aliased input")
	}
}

func fullNode() Node {
	n := NewNode(RoleTextInput)
	n.AddAction(ActionFocus)
	n.AddChildAction(ActionClick)
	n.SetHidden()
	n.SetChildren([]NodeID{1, 2})
	n.SetActiveDescendant(3)
	n.SetLabel("label \"q\"")
	n.SetURL("https://example.invalid")
	n.SetScrollX(1.5)
	n.SetFontSize(12.5)
	n.SetRowCount(4)
	n.SetColorValue(Color{Red: 1, Green: 2, Blue: 3, Alpha: 4})
	n.SetUnderline(testTextDecoration)
	n.SetCharacterLengths([]uint8{1, 2, 255})
	n.SetCharacterPositions([]float32{0, 1.25})
	n.SetExpanded(true)
	n.SetToggled(ToggledMixed)
	n.SetTextDirection(TextDirectionRightToLeft)
	n.SetTransform(AffineScale(2))
	n.SetBounds(NewRect(0, 1, 2, 3))
	n.SetTextSelection(TextSelection{Anchor: TextPosition{Node: 5, CharacterIndex: 1}, Focus: TextPosition{Node: 5, CharacterIndex: 3}})
	n.PushCustomAction(CustomAction{ID: 7, Description: "seven"})
	n.SetTreeID(TreeID(Uuid{0x12, 0x34, 15: 0xff}))
	return n
}

func TestLib_NodeJSONRoundTrip(t *testing.T) {
	n := fullNode()
	data, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		`"role":"textInput"`, `"actions":2`, `"childActions":1`, `"flags":1`,
		`"children":[1,2]`, `"activeDescendant":3`, `"url":"https://example.invalid"`,
		`"characterLengths":[1,2,255]`, `"toggled":"mixed"`, `"textDirection":"rightToLeft"`,
		`"transform":[2,0,0,2,0,0]`, `"bounds":{"x0":0,"y0":1,"x1":2,"y1":3}`,
		`"underline":{"style":"dotted","color":{"red":0,"green":0,"blue":0,"alpha":255}}`,
		`"textSelection":{"anchor":{"node":5,"characterIndex":1},"focus":{"node":5,"characterIndex":3}}`,
		`"customActions":[{"id":7,"description":"seven"}]`, `"treeId":"12340000-0000-0000-0000-0000000000ff"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in %s", want, s)
		}
	}
	if strings.Index(s, `"children"`) > strings.Index(s, `"label"`) {
		t.Error("properties are not in PropertyId order")
	}
	var back Node
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !n.Equal(&back) {
		t.Fatalf("round trip differs:\n%s\n%s", n.String(), back.String())
	}
}

func TestLib_NodeJSONSkipsClearedAndRejectsUnknown(t *testing.T) {
	n := NewNode(RoleButton)
	n.SetLabel("x")
	n.ClearLabel()
	data, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != `{"role":"button","actions":0,"childActions":0,"flags":0,"properties":{}}` {
		t.Fatal(got)
	}
	var back Node
	if err := json.Unmarshal([]byte(`{"role":"button","actions":0,"childActions":0,"flags":0,"properties":{},"extra":1}`), &back); err == nil {
		t.Fatal("unknown field accepted")
	}
	if err := json.Unmarshal([]byte(`{"role":"button","actions":0,"childActions":0,"flags":0,"properties":{"bogus":1}}`), &back); err == nil {
		t.Fatal("unknown property accepted")
	}
	if err := json.Unmarshal([]byte(`{"role":"button"}`), &back); err == nil {
		t.Fatal("missing fields accepted")
	}
	if err := json.Unmarshal([]byte(`{"role":"button","actions":0,"childActions":0,"flags":0,"properties":{"unset":[1],"label":"y"}}`), &back); err != nil {
		t.Fatal(err)
	}
	if v, _ := back.Label(); v != "y" {
		t.Fatal(v)
	}
}

func TestLib_TreeUpdateJSONRoundTrip(t *testing.T) {
	tree := NewTree(1)
	tree.ToolkitName = gpuiutil.Some("gpui")
	u := TreeUpdate{
		Nodes:  []NodeEntry{{ID: 1, Node: fullNode()}, {ID: 2, Node: NewNode(RoleWindow)}},
		Tree:   gpuiutil.Some(tree),
		TreeID: TreeIDRoot(),
		Focus:  1,
	}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{`"nodes":[[1,{`, `"tree":{"root":1,"toolkitName":"gpui","toolkitVersion":null}`, `"treeId":"00000000-0000-0000-0000-000000000000"`, `"focus":1`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in %s", want, s)
		}
	}
	var back TreeUpdate
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !u.Equal(&back) {
		t.Fatal("round trip differs")
	}
	c := u.Clone()
	c.Nodes[0].Node.SetLabel("changed")
	if !u.Equal(&back) || u.Equal(&c) {
		t.Fatal("Clone aliased the original")
	}
}

func TestLib_ActionRequestJSONRoundTrip(t *testing.T) {
	cases := []ActionRequest{
		{Action: ActionClick, TargetTree: TreeIDRoot(), TargetNode: 3},
		{Action: ActionCustomAction, TargetNode: 1, Data: ActionDataCustomAction(5)},
		{Action: ActionSetValue, Data: ActionDataValue("v")},
		{Action: ActionSetValue, Data: ActionDataNumericValue(2.5)},
		{Action: ActionScrollDown, Data: ActionDataScrollUnit(ScrollUnitPage)},
		{Action: ActionScrollIntoView, Data: ActionDataScrollHint(ScrollHintBottomEdge)},
		{Action: ActionScrollToPoint, Data: ActionDataScrollToPoint(NewPoint(1, 2))},
		{Action: ActionSetScrollOffset, Data: ActionDataSetScrollOffset(NewPoint(3, 4))},
		{Action: ActionSetTextSelection, Data: ActionDataSetTextSelection(TextSelection{Focus: TextPosition{Node: 1, CharacterIndex: 2}})},
	}
	for _, c := range cases {
		data, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		var back ActionRequest
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("%s: %v", data, err)
		}
		if back != c {
			t.Fatalf("%s: got %+v want %+v", data, back, c)
		}
	}
	data, _ := json.Marshal(cases[1])
	if got := string(data); got != `{"action":"customAction","targetTree":"00000000-0000-0000-0000-000000000000","targetNode":1,"data":{"customAction":5}}` {
		t.Fatal(got)
	}
	data, _ = json.Marshal(cases[4])
	if !strings.Contains(string(data), `"data":{"scrollUnit":"page"}`) {
		t.Fatal(string(data))
	}
}

func TestLib_EnumsUseUpstreamValuesAndNames(t *testing.T) {
	if RoleUnknown != 0 || RoleTerminal != Role(len(roleNames)-1) || RoleTextRun != 1 || RoleWindow.String() != "Window" {
		t.Fatal("Role values")
	}
	if ActionShowContextMenu != 21 || ToggledMixed != 2 || AriaCurrentTime != 6 || ScrollHintRightEdge != 5 {
		t.Fatal("enum values")
	}
	if len(propertyIDJSONNames) != int(propertyIDUnset)+1 || propertyIDTreeId != propertyIDUnset-1 {
		t.Fatal("PropertyId layout")
	}
	for _, c := range []struct {
		v    interface{ MarshalText() ([]byte, error) }
		want string
	}{{RoleUrlInput, "urlInput"}, {RoleSvgRoot, "svgRoot"}, {ActionSetSequentialFocusNavigationStartingPoint, "setSequentialFocusNavigationStartingPoint"}, {HasPopupListbox, "listbox"}} {
		got, _ := c.v.MarshalText()
		if string(got) != c.want {
			t.Fatalf("got %s want %s", got, c.want)
		}
	}
	var r Role
	if err := r.UnmarshalText([]byte("nope")); err == nil {
		t.Fatal("unknown variant accepted")
	}
}

func TestLib_UuidParsesAllForms(t *testing.T) {
	want := Uuid{0x67, 0xe5, 0x50, 0x44, 0x10, 0xb1, 0x42, 0x6f, 0x92, 0x47, 0xbb, 0x68, 0x0e, 0x5f, 0xe0, 0xc8}
	for _, s := range []string{"67e55044-10b1-426f-9247-bb680e5fe0c8", "67e5504410b1426f9247bb680e5fe0c8", "{67e55044-10b1-426f-9247-bb680e5fe0c8}", "urn:uuid:67e55044-10b1-426f-9247-bb680e5fe0c8"} {
		var u Uuid
		if err := u.UnmarshalText([]byte(s)); err != nil || u != want {
			t.Fatalf("%s: %v %v", s, u, err)
		}
	}
	if want.String() != "67e55044-10b1-426f-9247-bb680e5fe0c8" {
		t.Fatal(want.String())
	}
}

func TestLib_DebugFormatsValues(t *testing.T) {
	n := NewNode(RoleUnknown)
	n.SetLabel("a\"b\n")
	n.SetFontSize(1)
	n.SetCharacterPositions([]float32{1, 2.5})
	n.SetTreeID(TreeIDRoot())
	want := `Node { role: Unknown, label: "a\"b\n", font_size: 1.0, character_positions: [1.0, 2.5], tree_id: TreeId(00000000-0000-0000-0000-000000000000) }`
	if got := n.String(); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestLib_PropertiesJSONSchemaListsUpstreamProperties(t *testing.T) {
	g := jsonschema.NewGenerator()
	s := g.Reflect(reflect.TypeFor[Node]())
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	props := g.SchemaFor(reflect.TypeFor[properties]())
	def, ok := s.Defs[strings.TrimPrefix(props.Ref, "#/$defs/")]
	if !ok {
		t.Fatalf("no properties def in %s", data)
	}
	if len(def.Properties) != int(propertyIDUnset)-1 {
		t.Fatalf("got %d properties", len(def.Properties))
	}
	if _, ok := def.Properties["treeId"]; ok {
		t.Fatal("upstream's schema omits treeId")
	}
	if got := def.Properties["characterLengths"].Type; len(got) != 1 || got[0] != "array" {
		t.Fatalf("characterLengths schema %v", got)
	}
	for _, want := range []string{`"textRun"`, `"additionalProperties":false`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in %s", want, data)
		}
	}
}

func TestLib_NodeCopyDoesNotAlias(t *testing.T) {
	n := NewNode(RoleButton)
	n.SetLabel("a")
	n.PushChild(1)
	entry := NodeEntry{ID: 1, Node: n}
	n.SetLabel("b")
	n.PushChild(2)
	got, _ := entry.Node.Label()
	if got != "a" || len(entry.Node.Children()) != 1 {
		t.Fatalf("copy changed: %q %v", got, entry.Node.Children())
	}
	entry.Node.PushChild(3)
	if c := n.Children(); len(c) != 2 || c[1] != 2 {
		t.Fatalf("original changed: %v", c)
	}
	n.Children()[0] = 9
	if n.Children()[0] != 1 {
		t.Fatal("getter returned internal slice")
	}
}

func TestLib_StrictUnmarshalRejectsMissingFields(t *testing.T) {
	var u TreeUpdate
	if err := json.Unmarshal([]byte(`{"treeId":"00000000-0000-0000-0000-000000000000"}`), &u); err == nil {
		t.Fatal("expected missing field error")
	}
	var p TextPosition
	if err := json.Unmarshal([]byte(`{}`), &p); err == nil {
		t.Fatal("expected missing field error")
	}
	var tr Tree
	if err := json.Unmarshal([]byte(`{"root":1}`), &tr); err != nil {
		t.Fatal(err)
	}
}

func TestLib_DuplicateFieldsRejected(t *testing.T) {
	var n Node
	if err := json.Unmarshal([]byte(`{"role":"button","role":"button","actions":0,"childActions":0,"flags":0,"properties":{}}`), &n); err == nil {
		t.Fatal("expected duplicate field error")
	}
}

func TestLib_DebugStrEscapes(t *testing.T) {
	if got := debugStr("a b "); got != "\"a b \"" {
		t.Fatal(got)
	}
	if got := debugStr("́á​"); got != `"\u{301}a`+"́"+`\u{200b}"` {
		t.Fatal(got)
	}
}
