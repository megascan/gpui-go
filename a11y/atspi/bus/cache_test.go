package bus

import (
	"slices"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/atspi"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
)

type noOpActionHandler struct{}

func (noOpActionHandler) DoAction(a11y.ActionRequest) {}

type noOpCallback struct{}

func (noOpCallback) RegisterInterfaces(*atspi.Adapter, consumer.NodeID, atspi.InterfaceSet)   {}
func (noOpCallback) UnregisterInterfaces(*atspi.Adapter, consumer.NodeID, atspi.InterfaceSet) {}
func (noOpCallback) EmitEvent(*atspi.Adapter, atspi.Event)                                    {}

func withChildren(role a11y.Role, children []a11y.NodeID) a11y.Node {
	node := a11y.NewNode(role)
	node.SetChildren(children)
	return node
}

const testBusName = ":1.0"

func rootFor(t *testing.T, update a11y.TreeUpdate) (*atspi.Adapter, atspi.PlatformRoot) {
	appContext := atspi.NewAppContext(gpuiutil.None[string]())
	adapter := atspi.NewAdapter(appContext, noOpCallback{}, update, false, atspi.WindowBounds{}, noOpActionHandler{})
	t.Cleanup(adapter.Release)
	return adapter, adapter.PlatformRoot()
}

func cache(t *testing.T, update a11y.TreeUpdate) (*atspi.Adapter, *cacheInterface) {
	adapter, root := rootFor(t, update)
	desktop := &desktopCell{}
	desktop.trySet(desktopRef())
	return adapter, newCacheInterface(testBusName, root, desktop)
}

func windowWithButton() a11y.TreeUpdate {
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1})},
			{ID: 1, Node: a11y.NewNode(a11y.RoleButton)},
		},
		Tree:   gpuiutil.Some(a11y.NewTree(0)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	}
}

func rootRef() ownedObjectAddress {
	return objectRef(testBusName, objectIDRoot())
}

func desktopRef() ownedObjectAddress {
	return ownedObjectAddress{BusName: ":1.1", Path: "/org/a11y/atspi/accessible/root"}
}

func TestCache_GetItemsPrependsApplicationRoot(t *testing.T) {
	_, iface := cache(t, windowWithButton())
	items, err := iface.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected application root, window, and button, got %d items", len(items))
	}
	app := items[0]
	if app.Object != rootRef() || app.App != rootRef() || app.Parent != desktopRef() {
		t.Fatalf("app refs = %+v", app)
	}
	if app.Index != -1 || app.Children != 1 || app.Role != atspi.RoleApplication {
		t.Fatalf("app = %+v", app)
	}
	if !slices.Contains(app.Ifaces, atspi.InterfaceApplication.Name()) {
		t.Fatalf("app.Ifaces = %v", app.Ifaces)
	}
}

func TestCache_TopWindowParentIsApplicationRoot(t *testing.T) {
	_, iface := cache(t, windowWithButton())
	items, err := iface.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected application root, window, and button, got %d items", len(items))
	}
	window := items[1]
	if window.Parent != rootRef() || window.Index != 0 {
		t.Fatalf("window = %+v", window)
	}
	if !strings.HasPrefix(string(window.Object.Path), "/org/a11y/atspi/accessible/") {
		t.Fatalf("window path = %s", window.Object.Path)
	}
	if window.Object.BusName != testBusName {
		t.Fatalf("window bus name = %s", window.Object.BusName)
	}
}

func TestCache_FilteredChildExcludedFromItemsAndCounts(t *testing.T) {
	hidden := a11y.NewNode(a11y.RoleButton)
	hidden.SetHidden()
	update := a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1, 2})},
			{ID: 1, Node: a11y.NewNode(a11y.RoleButton)},
			{ID: 2, Node: hidden},
		},
		Tree:   gpuiutil.Some(a11y.NewTree(0)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	}
	_, iface := cache(t, update)
	items, err := iface.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected application root, window, and visible button only, got %d items", len(items))
	}
	if items[0].Children != 1 || items[1].Children != 1 {
		t.Fatalf("children = %d, %d", items[0].Children, items[1].Children)
	}
}

func TestCache_ObjectRefEncodesBusNameAndNodePath(t *testing.T) {
	_, root := rootFor(t, windowWithButton())
	child, ok, err := root.ChildIDAtIndex(0)
	if err != nil || !ok {
		t.Fatalf("ChildIDAtIndex = %v, %v", ok, err)
	}
	expectedPath := objectIDNode(child.AdapterID, child.ID).path()
	reference := objectRef(testBusName, objectIDNode(child.AdapterID, child.ID))
	if reference.BusName != testBusName || reference.Path != expectedPath {
		t.Fatalf("reference = %+v, want path %s", reference, expectedPath)
	}
}

func TestCache_MultiAdapterWindowsHaveDistinctIndices(t *testing.T) {
	appContext := atspi.NewAppContext(gpuiutil.None[string]())
	a0 := atspi.NewAdapter(appContext, noOpCallback{}, windowWithButton(), false, atspi.WindowBounds{}, noOpActionHandler{})
	defer a0.Release()
	a1 := atspi.NewAdapter(appContext, noOpCallback{}, windowWithButton(), false, atspi.WindowBounds{}, noOpActionHandler{})
	defer a1.Release()
	iface := newCacheInterface(testBusName, atspi.NewPlatformRoot(appContext), &desktopCell{})
	items, err := iface.items()
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Children != 2 {
		t.Fatalf("app children = %d", items[0].Children)
	}
	var indices []int32
	for _, item := range items {
		if item.Parent == rootRef() {
			indices = append(indices, item.Index)
		}
	}
	slices.Sort(indices)
	if !slices.Equal(indices, []int32{0, 1}) {
		t.Fatalf("window indices = %v", indices)
	}
}
