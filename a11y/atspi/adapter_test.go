package atspi

import (
	"sync"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
)

type cacheOp struct {
	removed bool
	id      consumer.NodeID
}

type capturingCallback struct {
	mu  *sync.Mutex
	ops *[]cacheOp
}

func (capturingCallback) RegisterInterfaces(*Adapter, consumer.NodeID, InterfaceSet)   {}
func (capturingCallback) UnregisterInterfaces(*Adapter, consumer.NodeID, InterfaceSet) {}
func (c capturingCallback) EmitEvent(_ *Adapter, event Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cache, ok := event.(EventCache); ok {
		switch e := cache.Event.(type) {
		case CacheEventAdded:
			*c.ops = append(*c.ops, cacheOp{id: e.ID})
		case CacheEventRemoved:
			*c.ops = append(*c.ops, cacheOp{removed: true, id: e.ID})
		}
	}
}

type noOpActionHandler struct{}

func (noOpActionHandler) DoAction(a11y.ActionRequest) {}

func withChildren(role a11y.Role, children []a11y.NodeID) a11y.Node {
	node := a11y.NewNode(role)
	node.SetChildren(children)
	return node
}

type opLog struct {
	mu  sync.Mutex
	ops []cacheOp
}

func (l *opLog) get() []cacheOp {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]cacheOp(nil), l.ops...)
}

func (l *opLog) clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ops = nil
}

func build(t *testing.T, initial a11y.TreeUpdate) (*Adapter, *opLog) {
	log := &opLog{}
	appContext := NewAppContext(gpuiutil.None[string]())
	adapter := NewAdapter(appContext, capturingCallback{mu: &log.mu, ops: &log.ops}, initial, false, WindowBounds{}, noOpActionHandler{})
	t.Cleanup(adapter.Release)
	return adapter, log
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

func update(nodes []a11y.NodeEntry) a11y.TreeUpdate {
	return a11y.TreeUpdate{Nodes: nodes, TreeID: a11y.TreeIDRoot(), Focus: 0}
}

func allOps(ops []cacheOp, removed bool) bool {
	for _, op := range ops {
		if op.removed != removed {
			return false
		}
	}
	return true
}

func TestAdapter_NoCacheEventsOnConstruction(t *testing.T) {
	_, log := build(t, initialTree())
	if ops := log.get(); len(ops) != 0 {
		t.Fatalf("ops = %v", ops)
	}
}

func TestAdapter_AddNodeEmitsOneAdded(t *testing.T) {
	adapter, log := build(t, initialTree())
	log.clear()
	adapter.Update(update([]a11y.NodeEntry{
		{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1, 2})},
		{ID: 2, Node: a11y.NewNode(a11y.RoleButton)},
	}))
	ops := log.get()
	if len(ops) != 1 || ops[0].removed {
		t.Fatalf("ops = %v", ops)
	}
}

func TestAdapter_RemoveNodeEmitsRemovedForSameID(t *testing.T) {
	adapter, log := build(t, initialTree())
	adapter.Update(update([]a11y.NodeEntry{
		{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1, 2})},
		{ID: 2, Node: a11y.NewNode(a11y.RoleButton)},
	}))
	ops := log.get()
	if len(ops) != 1 || ops[0].removed {
		t.Fatalf("expected exactly one Added, got %v", ops)
	}
	addedID := ops[0].id
	log.clear()
	adapter.Update(update([]a11y.NodeEntry{
		{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1})},
	}))
	if ops := log.get(); len(ops) != 1 || ops[0] != (cacheOp{removed: true, id: addedID}) {
		t.Fatalf("ops = %v", ops)
	}
}

func TestAdapter_SubtreeAddEmitsAddedPerNode(t *testing.T) {
	adapter, log := build(t, initialTree())
	log.clear()
	adapter.Update(update([]a11y.NodeEntry{
		{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1, 2})},
		{ID: 2, Node: withChildren(a11y.RoleGroup, []a11y.NodeID{3, 4})},
		{ID: 3, Node: a11y.NewNode(a11y.RoleButton)},
		{ID: 4, Node: a11y.NewNode(a11y.RoleButton)},
	}))
	if ops := log.get(); len(ops) != 3 || !allOps(ops, false) {
		t.Fatalf("ops = %v", ops)
	}
}

func TestAdapter_SubtreeRemoveEmitsRemovedPerNode(t *testing.T) {
	adapter, log := build(t, initialTree())
	adapter.Update(update([]a11y.NodeEntry{
		{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1, 2})},
		{ID: 2, Node: withChildren(a11y.RoleGroup, []a11y.NodeID{3, 4})},
		{ID: 3, Node: a11y.NewNode(a11y.RoleButton)},
		{ID: 4, Node: a11y.NewNode(a11y.RoleButton)},
	}))
	log.clear()
	adapter.Update(update([]a11y.NodeEntry{
		{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1})},
	}))
	if ops := log.get(); len(ops) != 3 || !allOps(ops, true) {
		t.Fatalf("ops = %v", ops)
	}
}

func TestAdapter_FilterTransitionIntoTreeEmitsAdded(t *testing.T) {
	hidden := a11y.NewNode(a11y.RoleButton)
	hidden.SetHidden()
	adapter, log := build(t, a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1, 2})},
			{ID: 1, Node: a11y.NewNode(a11y.RoleButton)},
			{ID: 2, Node: hidden},
		},
		Tree:   gpuiutil.Some(a11y.NewTree(0)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	})
	log.clear()
	adapter.Update(update([]a11y.NodeEntry{{ID: 2, Node: a11y.NewNode(a11y.RoleButton)}}))
	if ops := log.get(); len(ops) != 1 || ops[0].removed {
		t.Fatalf("ops = %v", ops)
	}
}

func TestAdapter_FilterTransitionOutOfTreeEmitsRemoved(t *testing.T) {
	adapter, log := build(t, initialTree())
	log.clear()
	hidden := a11y.NewNode(a11y.RoleButton)
	hidden.SetHidden()
	adapter.Update(update([]a11y.NodeEntry{{ID: 1, Node: hidden}}))
	if ops := log.get(); len(ops) != 1 || !ops[0].removed {
		t.Fatalf("ops = %v", ops)
	}
}

type eventLog struct {
	events []Event
}

func (*eventLog) RegisterInterfaces(*Adapter, consumer.NodeID, InterfaceSet)   {}
func (*eventLog) UnregisterInterfaces(*Adapter, consumer.NodeID, InterfaceSet) {}
func (l *eventLog) EmitEvent(_ *Adapter, event Event)                          { l.events = append(l.events, event) }

func textTree(value string) a11y.TreeUpdate {
	input := withChildren(a11y.RoleTextInput, []a11y.NodeID{2})
	run := a11y.NewNode(a11y.RoleTextRun)
	run.SetValue(value)
	lengths := make([]uint8, len([]rune(value)))
	for i := range lengths {
		lengths[i] = uint8(len(string([]rune(value)[i])))
	}
	run.SetCharacterLengths(lengths)
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: withChildren(a11y.RoleWindow, []a11y.NodeID{1})},
			{ID: 1, Node: input},
			{ID: 2, Node: run},
		},
		Tree:   gpuiutil.Some(a11y.NewTree(0)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	}
}

func TestAdapter_TextChangeEmitsRemovedAndInserted(t *testing.T) {
	log := &eventLog{}
	adapter := NewAdapter(NewAppContext(gpuiutil.None[string]()), log, textTree("héllo"), true, WindowBounds{}, noOpActionHandler{})
	defer adapter.Release()
	log.events = nil
	next := textTree("héyo")
	next.Tree = gpuiutil.None[a11y.Tree]()
	adapter.Update(next)
	var got []ObjectEvent
	for _, e := range log.events {
		if o, ok := e.(EventObject); ok {
			switch o.Event.(type) {
			case ObjectEventTextRemoved, ObjectEventTextInserted:
				got = append(got, o.Event)
			}
		}
	}
	want := []ObjectEvent{
		ObjectEventTextRemoved{StartIndex: 2, Length: 2, Content: "ll"},
		ObjectEventTextInserted{StartIndex: 2, Length: 1, Content: "y"},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("events = %+v", got)
	}
}

func TestAdapter_ReleaseRemovesAdapter(t *testing.T) {
	log := &eventLog{}
	appContext := NewAppContext(gpuiutil.None[string]())
	adapter := NewAdapter(appContext, log, initialTree(), false, WindowBounds{}, noOpActionHandler{})
	root := NewPlatformRoot(appContext)
	if n, _ := root.ChildCount(); n != 1 {
		t.Fatalf("child count = %d", n)
	}
	node := adapter.PlatformNode(adapter.RootID())
	adapter.Release()
	adapter.Release()
	if n, _ := root.ChildCount(); n != 0 {
		t.Fatalf("child count after release = %d", n)
	}
	if _, err := node.Name(); err != ErrorDefunct {
		t.Fatalf("err = %v", err)
	}
	last, ok := log.events[len(log.events)-1].(EventObject)
	if _, removed := last.Event.(ObjectEventChildRemoved); !ok || !last.Target.IsRoot() || !removed {
		t.Fatalf("last event = %+v", log.events[len(log.events)-1])
	}
}
