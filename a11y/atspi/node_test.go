package atspi

import (
	"runtime"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
)

type recordingHandler struct {
	requests []a11y.ActionRequest
}

func (h *recordingHandler) DoAction(request a11y.ActionRequest) {
	h.requests = append(h.requests, request)
}

func testUpdate() a11y.TreeUpdate {
	window := a11y.NewNode(a11y.RoleWindow)
	window.SetLabel("Window")
	window.SetChildren([]a11y.NodeID{1, 2})
	button := a11y.NewNode(a11y.RoleButton)
	button.SetLabel("OK")
	button.SetBounds(a11y.Rect{X0: 10, Y0: 20, X1: 110, Y1: 50})
	button.AddAction(a11y.ActionClick)
	input := a11y.NewNode(a11y.RoleTextInput)
	input.SetBounds(a11y.Rect{X0: 0, Y0: 60, X1: 100, Y1: 80})
	input.SetChildren([]a11y.NodeID{3})
	input.SetTextDirection(a11y.TextDirectionLeftToRight)
	input.SetTextSelection(a11y.TextSelection{
		Anchor: a11y.TextPosition{Node: 3, CharacterIndex: 1},
		Focus:  a11y.TextPosition{Node: 3, CharacterIndex: 3},
	})
	run := a11y.NewNode(a11y.RoleTextRun)
	run.SetBounds(a11y.Rect{X0: 0, Y0: 60, X1: 50, Y1: 80})
	run.SetValue("hello")
	run.SetCharacterLengths([]uint8{1, 1, 1, 1, 1})
	run.SetCharacterPositions([]float32{0, 10, 20, 30, 40})
	run.SetCharacterWidths([]float32{10, 10, 10, 10, 10})
	run.SetWordStarts([]uint8{0})
	tree := a11y.NewTree(0)
	tree.ToolkitName = gpuiutil.Some("gpui")
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: window},
			{ID: 1, Node: button},
			{ID: 2, Node: input},
			{ID: 3, Node: run},
		},
		Tree:   gpuiutil.Some(tree),
		TreeID: a11y.TreeIDRoot(),
		Focus:  1,
	}
}

func testContext(t *testing.T) (*AppContext, *Context, *recordingHandler) {
	t.Helper()
	appContext := NewAppContext(gpuiutil.Some("app"))
	handler := &recordingHandler{}
	bounds := NewWindowBounds(a11y.Rect{X0: 95, Y0: 75, X1: 505, Y1: 405}, a11y.Rect{X0: 100, Y0: 100, X1: 500, Y1: 400})
	context := newContext(appContext, consumer.NewTree(testUpdate(), true), handler, bounds)
	appContext.pushAdapter(7, context)
	return appContext, context, handler
}

func nodeID(context *Context, local a11y.NodeID) consumer.NodeID {
	tree, done := context.readTree()
	defer done()
	n, _ := tree.State().NodeByTreeLocalID(local, a11y.TreeIDRoot())
	return n.ID()
}

func TestNode_RoleMapsEveryAccessKitRole(t *testing.T) {
	for role := a11y.Role(0); role <= a11y.RoleTerminal; role++ {
		n := a11y.NewNode(role)
		tree := consumer.NewTree(a11y.TreeUpdate{
			Nodes:  []a11y.NodeEntry{{ID: 0, Node: n}},
			Tree:   gpuiutil.Some(a11y.NewTree(0)),
			TreeID: a11y.TreeIDRoot(),
			Focus:  0,
		}, false)
		root := tree.State().Root()
		got := nodeWrapper{&root}.role()
		if role == a11y.RoleWindow && got != RoleFrame {
			t.Fatalf("Window mapped to %v", got)
		}
	}
}

func TestNode_PlatformNodeAccessible(t *testing.T) {
	appContext, context, _ := testContext(t)
	defer runtime.KeepAlive(appContext)
	root := newPlatformNode(context, 7, nodeID(context, 0))
	if name, err := root.Name(); err != nil || name != "Window" {
		t.Fatalf("name = %q, %v", name, err)
	}
	if role, _ := root.Role(); role != RoleFrame {
		t.Fatalf("role = %v", role)
	}
	if n, _ := root.ChildCount(); n != 2 {
		t.Fatalf("child count = %d", n)
	}
	if i, _ := root.IndexInParent(); i != 0 {
		t.Fatalf("index in parent = %d", i)
	}
	if parent, _ := root.Parent(); !parent.IsRoot() {
		t.Fatal("root parent is not the application")
	}
	if tk, _ := root.ToolkitName(); tk != "gpui" {
		t.Fatalf("toolkit = %q", tk)
	}
	state := root.State()
	if !state.Contains(StateActive) || !state.Contains(StateEnabled|StateSensitive) {
		t.Fatalf("root state = %x", state.Bits())
	}
	button := root.Relative(nodeID(context, 1))
	if role, _ := button.Role(); role != RoleButton {
		t.Fatalf("button role = %v", role)
	}
	if !button.State().Contains(StateFocused | StateFocusable | StateVisible | StateShowing) {
		t.Fatalf("button state = %x", button.State().Bits())
	}
	ifaces, _ := button.Interfaces()
	if !ifaces.Contains(InterfaceAccessible|InterfaceAction|InterfaceComponent) || ifaces.Contains(InterfaceText) {
		t.Fatalf("button interfaces = %x", ifaces.Bits())
	}
	if i, _ := button.IndexInParent(); i != 0 {
		t.Fatalf("button index = %d", i)
	}
	cache, err := button.CacheNode()
	if err != nil || cache.Name != "OK" || cache.ChildCount != 0 || cache.Role != RoleButton {
		t.Fatalf("cache node = %+v, %v", cache, err)
	}
	ids, _ := root.MapChildren(func(id consumer.NodeID) consumer.NodeID { return id })
	if len(ids) != 2 || ids[0] != button.ID() {
		t.Fatalf("children = %v", ids)
	}
}

func TestNode_PlatformNodeComponentAndAction(t *testing.T) {
	appContext, context, handler := testContext(t)
	defer runtime.KeepAlive(appContext)
	button := newPlatformNode(context, 7, nodeID(context, 1))
	if r, _ := button.Extents(CoordTypeWindow); r != (Rect{X: 10, Y: 20, Width: 100, Height: 30}) {
		t.Fatalf("window extents = %+v", r)
	}
	if r, _ := button.Extents(CoordTypeScreen); r != (Rect{X: 110, Y: 120, Width: 100, Height: 30}) {
		t.Fatalf("screen extents = %+v", r)
	}
	if ok, _ := button.Contains(115, 125, CoordTypeScreen); !ok {
		t.Fatal("contains failed")
	}
	root := button.Relative(nodeID(context, 0))
	if id, ok, _ := root.AccessibleAtPoint(115, 125, CoordTypeScreen); !ok || id != button.ID() {
		t.Fatalf("accessible at point = %v, %v", id, ok)
	}
	if n, _ := button.NActions(); n != 1 {
		t.Fatalf("n actions = %d", n)
	}
	if name, _ := button.ActionName(0); name != "click" {
		t.Fatalf("action name = %q", name)
	}
	if ok, err := button.DoAction(0); !ok || err != nil {
		t.Fatalf("do action = %v, %v", ok, err)
	}
	if ok, _ := button.DoAction(1); ok {
		t.Fatal("do action 1 succeeded")
	}
	if len(handler.requests) != 1 || handler.requests[0].Action != a11y.ActionClick || handler.requests[0].TargetNode != 1 {
		t.Fatalf("requests = %+v", handler.requests)
	}
	if layer, _ := button.Layer(); layer != LayerWidget {
		t.Fatalf("layer = %v", layer)
	}
}

func TestNode_PlatformNodeText(t *testing.T) {
	appContext, context, handler := testContext(t)
	defer runtime.KeepAlive(appContext)
	input := newPlatformNode(context, 7, nodeID(context, 2))
	if n, err := input.CharacterCount(); err != nil || n != 5 {
		t.Fatalf("character count = %d, %v", n, err)
	}
	if text, _ := input.Text(1, -1); text != "ello" {
		t.Fatalf("text = %q", text)
	}
	if c, _ := input.CaretOffset(); c != 3 {
		t.Fatalf("caret = %d", c)
	}
	if s, e, _ := input.Selection(0); s != 1 || e != 3 {
		t.Fatalf("selection = %d..%d", s, e)
	}
	if n, _ := input.NSelections(); n != 1 {
		t.Fatalf("n selections = %d", n)
	}
	if text, s, e, _ := input.StringAtOffset(2, GranularityChar); text != "l" || s != 2 || e != 3 {
		t.Fatalf("char at offset = %q %d %d", text, s, e)
	}
	if _, _, _, err := input.StringAtOffset(0, GranularitySentence); err != ErrorUnsupportedTextGranularity {
		t.Fatalf("sentence err = %v", err)
	}
	if _, err := input.Text(9, 10); err != ErrorIndexOutOfRange {
		t.Fatalf("out of range err = %v", err)
	}
	if r, _ := input.CharacterExtents(1, CoordTypeWindow); r != (Rect{X: 10, Y: 60, Width: 10, Height: 20}) {
		t.Fatalf("character extents = %+v", r)
	}
	if ok, _ := input.SetCaretOffset(4); !ok {
		t.Fatal("set caret failed")
	}
	last := handler.requests[len(handler.requests)-1]
	if last.Action != a11y.ActionSetTextSelection {
		t.Fatalf("last request = %+v", last)
	}
	if state := input.State(); !state.Contains(StateEditable | StateSelectableText | StateSingleLine) {
		t.Fatalf("input state = %x", state.Bits())
	}
	button := input.Relative(nodeID(context, 1))
	if _, err := button.CharacterCount(); err != ErrorUnsupportedInterface {
		t.Fatalf("button text err = %v", err)
	}
}

func TestNode_PlatformRoot(t *testing.T) {
	appContext, context, _ := testContext(t)
	root := NewPlatformRoot(appContext)
	if name, _ := root.Name(); name != "app" {
		t.Fatalf("name = %q", name)
	}
	if n, _ := root.ChildCount(); n != 1 {
		t.Fatalf("child count = %d", n)
	}
	if id, _ := root.ID(); id != -1 {
		t.Fatalf("id = %d", id)
	}
	if err := root.SetID(5); err != nil {
		t.Fatal(err)
	}
	if id, _ := root.ID(); id != 5 {
		t.Fatalf("id = %d", id)
	}
	child, ok, _ := root.ChildAtIndex(0)
	if !ok || child.AdapterID() != 7 || child.ID() != nodeID(context, 0) {
		t.Fatalf("child = %+v %v", child, ok)
	}
	nodes, err := root.MapDescendantCacheNodes(func(c CacheNode) CacheNode { return c })
	if err != nil || len(nodes) != 3 {
		t.Fatalf("descendants = %d, %v", len(nodes), err)
	}
	if !root.Equal(NewPlatformRoot(appContext)) {
		t.Fatal("roots differ")
	}
}

func TestNode_DefunctAfterRelease(t *testing.T) {
	appContext, context, _ := testContext(t)
	defer runtime.KeepAlive(appContext)
	node := newPlatformNode(context, 7, nodeID(context, 1))
	context.released.Store(true)
	if _, err := node.Name(); err != ErrorDefunct {
		t.Fatalf("err = %v", err)
	}
	if !node.State().Contains(StateDefunct) {
		t.Fatal("state lacks defunct")
	}
}

func TestNode_StateSetIterOrder(t *testing.T) {
	s := NewStateSet(StateReadOnly, StateActive, StateFocused)
	var got []State
	for st := range s.Iter() {
		got = append(got, st)
	}
	if len(got) != 3 || got[0] != StateActive || got[1] != StateFocused || got[2] != StateReadOnly {
		t.Fatalf("order = %v", got)
	}
	if StateReadOnly != 1<<43 || InterfaceValue != 1<<19 || RolePushButtonMenu != 129 {
		t.Fatal("discriminants differ from atspi-common")
	}
	if StateFromStr("read-only") != StateReadOnly || RoleDocumentWeb.Name() != "document web" {
		t.Fatal("names differ from atspi-common")
	}
}

func TestRect_FromAccessKit(t *testing.T) {
	r := RectFromAccessKit(a11y.Rect{X0: 1.9, Y0: -2.5, X1: 1e20, Y1: 10})
	if r != (Rect{X: 1, Y: -2, Width: 2147483647, Height: 12}) {
		t.Fatalf("rect = %+v", r)
	}
}

func TestTextAttributes_Size(t *testing.T) {
	if formatF32(16) != "16" || formatF32(12.5) != "12.5" {
		t.Fatal("f32 display differs")
	}
}
