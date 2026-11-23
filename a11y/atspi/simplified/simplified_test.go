package simplified

import (
	"slices"
	"sync"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/atspi"
	"github.com/megascan/gpui-go/gpuiutil"
)

type noOpActionHandler struct{}

func (noOpActionHandler) DoAction(a11y.ActionRequest) {}

type cacheKindCallback struct {
	mu    sync.Mutex
	kinds []string
}

func (*cacheKindCallback) RegisterInterfaces(*atspi.Adapter, atspi.NodeID, atspi.InterfaceSet)   {}
func (*cacheKindCallback) UnregisterInterfaces(*atspi.Adapter, atspi.NodeID, atspi.InterfaceSet) {}
func (c *cacheKindCallback) EmitEvent(adapter *atspi.Adapter, event atspi.Event) {
	if _, ok := event.(atspi.EventCache); ok {
		simplified := NewEvent(adapter, event)
		c.mu.Lock()
		c.kinds = append(c.kinds, simplified.Kind)
		c.mu.Unlock()
	}
}

type noOpCallback struct{}

func (noOpCallback) RegisterInterfaces(*atspi.Adapter, atspi.NodeID, atspi.InterfaceSet)   {}
func (noOpCallback) UnregisterInterfaces(*atspi.Adapter, atspi.NodeID, atspi.InterfaceSet) {}
func (noOpCallback) EmitEvent(*atspi.Adapter, atspi.Event)                                 {}

func withChildren(role a11y.Role, children []a11y.NodeID) a11y.Node {
	node := a11y.NewNode(role)
	node.SetChildren(children)
	return node
}

func initialTree() a11y.TreeUpdate {
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

func TestSimplified_CacheEventsMapToKindStrings(t *testing.T) {
	callback := &cacheKindCallback{}
	appContext := atspi.NewAppContext(gpuiutil.None[string]())
	adapter := atspi.NewAdapter(appContext, callback, initialTree(), false, atspi.WindowBounds{}, noOpActionHandler{})
	defer adapter.Release()
	adapter.Update(a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1, 2})},
			{ID: 2, Node: a11y.NewNode(a11y.RoleButton)},
		},
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	})
	adapter.Update(a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1})},
		},
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	})
	if !slices.Equal(callback.kinds, []string{"cache:add", "cache:remove"}) {
		t.Fatalf("kinds = %v", callback.kinds)
	}
}

func buildCache() (*atspi.Adapter, *Cache) {
	appContext := atspi.NewAppContext(gpuiutil.None[string]())
	adapter := atspi.NewAdapter(appContext, noOpCallback{}, initialTree(), false, atspi.WindowBounds{}, noOpActionHandler{})
	cache := NewCache(adapter.PlatformRoot())
	return adapter, cache
}

func TestSimplified_ItemsPrependsApplicationRoot(t *testing.T) {
	adapter, cache := buildCache()
	defer adapter.Release()
	items, err := cache.Items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("items = %d", len(items))
	}
	if !items[0].IsRoot() {
		t.Fatal("first item is not the application root")
	}
}
