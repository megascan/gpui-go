//go:build windows

package windows

import (
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
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
	check := a11y.NewNode(a11y.RoleCheckBox)
	check.SetLabel("Check")
	check.SetToggled(a11y.ToggledTrue)
	check.AddAction(a11y.ActionClick)
	tree := a11y.NewTree(0)
	tree.ToolkitName = gpuiutil.Some("gpui")
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: window},
			{ID: 1, Node: button},
			{ID: 2, Node: check},
		},
		Tree:   gpuiutil.Some(tree),
		TreeID: a11y.TreeIDRoot(),
		Focus:  1,
	}
}

func testContext(handler *recordingHandler) *Context {
	return newContext(WindowHandle{}, consumer.NewTree(testUpdate(), true), handler, false)
}

func nodeByLocalID(t *testing.T, context *Context, local a11y.NodeID) consumer.Node {
	t.Helper()
	tree, unlock := context.readTree()
	defer unlock()
	n, ok := tree.State().NodeByTreeLocalID(local, a11y.TreeIDRoot())
	if !ok {
		t.Fatalf("node %d missing", local)
	}
	return n
}

func TestNode_PlatformNodeVtablesBuild(t *testing.T) {
	vtables := platformNodeVtables()
	if len(vtables) != platformNodeWindow+1 {
		t.Fatalf("got %d vtables", len(vtables))
	}
}

func TestNode_ControlTypeCoversEveryRole(t *testing.T) {
	for role := a11y.Role(0); role <= a11y.RoleTerminal; role++ {
		n := a11y.NewNode(role)
		tree := consumer.NewTree(a11y.TreeUpdate{
			Nodes:  []a11y.NodeEntry{{ID: 0, Node: n}},
			Tree:   gpuiutil.Some(a11y.NewTree(0)),
			TreeID: a11y.TreeIDRoot(),
			Focus:  0,
		}, false)
		root := tree.State().Root()
		w := nodeWrapper{node: &root}
		got := w.controlType()
		if role == a11y.RoleButton && got != i32Value(uiaButtonControlTypeID) {
			t.Fatalf("Button mapped to %d", got)
		}
	}
}

func TestNode_PropertiesAndPatterns(t *testing.T) {
	handler := &recordingHandler{}
	context := testContext(handler)
	defer context.Release()
	button := nodeByLocalID(t, context, 1)
	var buf []uint16
	w := nodeWrapper{node: &button, stringBuffer: &buf}
	if got := w.name(); got != (optString{value: "OK", ok: true}) {
		t.Fatalf("name %+v", got)
	}
	if got := w.ariaRole(); got != (optString{value: "button", ok: true}) {
		t.Fatalf("aria role %+v", got)
	}
	if !w.isInvokePatternSupported() {
		t.Fatal("button should support Invoke")
	}
	check := nodeByLocalID(t, context, 2)
	cw := nodeWrapper{node: &check, stringBuffer: &buf}
	if !cw.isTogglePatternSupported() || cw.toggleState() != toggleStateOn {
		t.Fatal("checkbox toggle state")
	}
}

func TestNode_InvokeSendsClick(t *testing.T) {
	handler := &recordingHandler{}
	context := testContext(handler)
	defer context.Release()
	id := nodeByLocalID(t, context, 1).ID()
	object := context.getOrCreatePlatformNode(id)
	defer object.Release()
	p := platformNodeOf(object.Interface(platformNodeSimple))
	if p == nil {
		t.Fatal("no platform node behind the interface")
	}
	if err := p.invoke(); err != nil {
		t.Fatal(err)
	}
	if len(handler.requests) != 1 || handler.requests[0].Action != a11y.ActionClick || handler.requests[0].TargetNode != 1 {
		t.Fatalf("requests %+v", handler.requests)
	}
	context.isPlaceholder.Store(true)
	if err := p.invoke(); err != elementNotEnabled() {
		t.Fatalf("placeholder invoke returned %v", err)
	}
}

func TestNode_ReleasedContextIsNotAvailable(t *testing.T) {
	context := testContext(&recordingHandler{})
	id := nodeByLocalID(t, context, 1).ID()
	object := context.getOrCreatePlatformNode(id)
	defer object.Release()
	p := platformNodeOf(object.Interface(platformNodeSimple))
	context.Release()
	if err := p.invoke(); err != elementNotAvailable() {
		t.Fatalf("got %v", err)
	}
}

func TestNode_RuntimeIDFromNodeID(t *testing.T) {
	context := testContext(&recordingHandler{})
	defer context.Release()
	id := nodeByLocalID(t, context, 1).ID()
	hi, lo := id.ToU128()
	got := runtimeIDFromNodeID(id)
	want := [runtimeIDSize]int32{uiaAppendRuntimeID, int32(uint32(hi >> 32)), int32(uint32(hi)), int32(uint32(lo >> 32)), int32(uint32(lo))}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNode_PlatformNodeReferenceCounting(t *testing.T) {
	update := testUpdate()
	update.Nodes[1].Node.SetControls([]a11y.NodeID{2})
	context := newContext(WindowHandle{}, consumer.NewTree(update, true), &recordingHandler{}, false)
	defer context.Release()
	buttonID := nodeByLocalID(t, context, 1).ID()
	checkID := nodeByLocalID(t, context, 2).ID()

	a := context.getOrCreatePlatformNode(buttonID)
	b := context.getOrCreatePlatformNode(buttonID)
	if a != b || a.RefCount() != 3 {
		t.Fatalf("refcount %d after two lookups", a.RefCount())
	}
	a.Release()
	b.Release()
	if a.RefCount() != 1 {
		t.Fatalf("refcount %d held by the map", a.RefCount())
	}

	var queue []queuedEvent
	enqueuePropertyChange(&queue, a.Interface(platformNodeSimple), uiaNamePropertyID, variantEmpty(), variantFromI32(1))
	if a.RefCount() != 2 {
		t.Fatalf("refcount %d with a queued event", a.RefCount())
	}
	for _, e := range queue {
		e.release()
	}
	if a.RefCount() != 1 {
		t.Fatalf("refcount %d after the event is released", a.RefCount())
	}

	p := platformNodeOf(a.Interface(platformNodeSimple))
	v, err := p.getPropertyValue(uiaControllerForPropertyID)
	if err != nil {
		t.Fatal(err)
	}
	check := context.getOrCreatePlatformNode(checkID)
	if check.RefCount() != 3 {
		t.Fatalf("controlled refcount %d while the VARIANT holds it", check.RefCount())
	}
	com.VariantClear(&v)
	check.Release()
	if check.RefCount() != 1 {
		t.Fatalf("controlled refcount %d after VariantClear", check.RefCount())
	}

	iface := a.Interface(platformNodeSimple)
	context.removePlatformNode(buttonID)
	if com.ObjectOf(iface) != nil {
		t.Fatal("removed platform node is still alive")
	}
	context.Release()
	if com.ObjectOf(check.Interface(platformNodeSimple)) != nil {
		t.Fatal("Release left a platform node alive")
	}
}
