package consumer

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

type testHandler struct {
	added   func(node *Node)
	updated func(oldNode, newNode *Node)
	focus   func(oldNode, newNode *Node)
	removed func(node *Node)
}

func (h *testHandler) NodeAdded(node *Node) {
	if h.added != nil {
		h.added(node)
	}
}

func (h *testHandler) NodeUpdated(oldNode, newNode *Node) {
	if h.updated != nil {
		h.updated(oldNode, newNode)
	}
}

func (h *testHandler) FocusMoved(oldNode, newNode *Node) {
	if h.focus != nil {
		h.focus(oldNode, newNode)
	}
}

func (h *testHandler) NodeRemoved(node *Node) {
	if h.removed != nil {
		h.removed(node)
	}
}

var noOpHandler = &testHandler{}

func strictHandler(msg string) *testHandler {
	unexpected := func() { panic(msg) }
	return &testHandler{
		added:   func(*Node) { unexpected() },
		updated: func(*Node, *Node) { unexpected() },
		focus:   func(*Node, *Node) { unexpected() },
		removed: func(*Node) { unexpected() },
	}
}

func nodeID(n uint64) NodeID {
	return newNodeID(a11y.NodeID(n), treeIndex(0))
}

func treeIDFromU128(n uint64) a11y.TreeID {
	var u a11y.Uuid
	binary.BigEndian.PutUint64(u[8:], n)
	return a11y.TreeID(u)
}

func node(role a11y.Role, children ...a11y.NodeID) a11y.Node {
	n := a11y.NewNode(role)
	if children != nil {
		n.SetChildren(children)
	}
	return n
}

func graftNode(treeID a11y.TreeID) a11y.Node {
	n := a11y.NewNode(a11y.RoleGenericContainer)
	n.SetTreeID(treeID)
	return n
}

func ent(id a11y.NodeID, n a11y.Node) a11y.NodeEntry {
	return a11y.NodeEntry{ID: id, Node: n}
}

func rootUpdate(focus a11y.NodeID, nodes ...a11y.NodeEntry) a11y.TreeUpdate {
	return a11y.TreeUpdate{Nodes: nodes, Tree: gpuiutil.Some(a11y.NewTree(0)), TreeID: a11y.TreeIDRoot(), Focus: focus}
}

func treeUpdate(treeID a11y.TreeID, tree gpuiutil.Option[a11y.Tree], focus a11y.NodeID, nodes ...a11y.NodeEntry) a11y.TreeUpdate {
	return a11y.TreeUpdate{Nodes: nodes, Tree: tree, TreeID: treeID, Focus: focus}
}

func someTree(root a11y.NodeID) gpuiutil.Option[a11y.Tree] {
	return gpuiutil.Some(a11y.NewTree(root))
}

func noTree() gpuiutil.Option[a11y.Tree] {
	return gpuiutil.None[a11y.Tree]()
}

func expectPanic(t *testing.T, expected string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", expected)
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, expected) {
			t.Fatalf("panic %q does not contain %q", msg, expected)
		}
	}()
	f()
}

func must[T any](v T, ok bool) T {
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return v
}

func eq[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func check(t *testing.T, cond bool) {
	t.Helper()
	if !cond {
		t.Fatal("assertion failed")
	}
}

func optID(n *Node) gpuiutil.Option[NodeID] {
	return focusIDOf(n)
}

func someID(id NodeID) gpuiutil.Option[NodeID] {
	return gpuiutil.Some(id)
}

func parentOpt(n Node) gpuiutil.Option[NodeID] {
	return gpuiutil.OptionFrom(n.ParentID())
}

func subtreeID() a11y.TreeID {
	return treeIDFromU128(1)
}

func subtreeNodeID(id uint64) NodeID {
	return newNodeID(a11y.NodeID(id), treeIndex(1))
}

func nestedSubtreeID() a11y.TreeID {
	return treeIDFromU128(2)
}

func nestedSubtreeNodeID(n uint64) NodeID {
	return newNodeID(a11y.NodeID(n), treeIndex(2))
}

func graftTree(isHostFocused bool) *Tree {
	return NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, graftNode(subtreeID())),
	), isHostFocused)
}

type recordingHandler struct {
	testHandler
	addedNodes   []NodeID
	updatedNodes []NodeID
	removedNodes []NodeID
}

func newRecordingHandler() *recordingHandler {
	h := &recordingHandler{}
	h.added = func(n *Node) { h.addedNodes = append(h.addedNodes, n.ID()) }
	h.updated = func(_, n *Node) { h.updatedNodes = append(h.updatedNodes, n.ID()) }
	h.removed = func(n *Node) { h.removedNodes = append(h.removedNodes, n.ID()) }
	return h
}

func TestTree_TreeIndexMapAssignsSequentialIndices(t *testing.T) {
	var m treeIndexMap
	id1 := a11y.TreeIDRoot()
	id2 := treeIDFromU128(1)
	id3 := treeIDFromU128(2)
	index1 := m.getOrCreateIndex(id1)
	index2 := m.getOrCreateIndex(id2)
	index3 := m.getOrCreateIndex(id3)
	eq(t, treeIndex(0), index1)
	eq(t, treeIndex(1), index2)
	eq(t, treeIndex(2), index3)
}

func TestTree_TreeIndexMapReturnsSameIndexForSameId(t *testing.T) {
	var m treeIndexMap
	id := a11y.TreeIDRoot()
	eq(t, m.getOrCreateIndex(id), m.getOrCreateIndex(id))
}

func TestTree_TreeIndexMapGetIdReturnsCorrectId(t *testing.T) {
	var m treeIndexMap
	id1 := a11y.TreeIDRoot()
	id2 := treeIDFromU128(1)
	index1 := m.getOrCreateIndex(id1)
	index2 := m.getOrCreateIndex(id2)
	eq(t, gpuiutil.Some(id1), gpuiutil.OptionFrom(m.getID(index1)))
	eq(t, gpuiutil.Some(id2), gpuiutil.OptionFrom(m.getID(index2)))
}

func TestTree_TreeIndexMapGetIdReturnsNoneForUnknownIndex(t *testing.T) {
	var m treeIndexMap
	_, ok := m.getID(treeIndex(0))
	check(t, !ok)
	_, ok = m.getID(treeIndex(999))
	check(t, !ok)
}

func TestTree_TreeIndexMapGetIndexReturnsExistingIndex(t *testing.T) {
	var m treeIndexMap
	id1 := a11y.TreeIDRoot()
	id2 := treeIDFromU128(1)
	created1 := m.getOrCreateIndex(id1)
	created2 := m.getOrCreateIndex(id2)
	eq(t, gpuiutil.Some(created1), gpuiutil.OptionFrom(m.getIndex(id1)))
	eq(t, gpuiutil.Some(created2), gpuiutil.OptionFrom(m.getIndex(id2)))
}

func TestTree_TreeIndexMapGetIndexReturnsNoneForUnknownId(t *testing.T) {
	var m treeIndexMap
	_, ok := m.getIndex(a11y.TreeIDRoot())
	check(t, !ok)
	_, ok = m.getIndex(treeIDFromU128(42))
	check(t, !ok)
}

func TestTree_TreeIndexMapGetIndexDoesNotCreateMapping(t *testing.T) {
	var m treeIndexMap
	unknown := treeIDFromU128(7)
	_, ok := m.getIndex(unknown)
	check(t, !ok)
	first := m.getOrCreateIndex(a11y.TreeIDRoot())
	eq(t, treeIndex(0), first)
	_, ok = m.getIndex(unknown)
	check(t, !ok)
}

func TestTree_InitTreeWithRootNode(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow))), false)
	eq(t, nodeID(0), tree.State().Root().ID())
	eq(t, a11y.RoleWindow, tree.State().Root().Role())
	_, ok := tree.State().Root().Parent()
	check(t, !ok)
}

func TestTree_InitTreeWithNonRootTreeIdPanics(t *testing.T) {
	expectPanic(t, "Cannot initialize with a subtree. TreeUpdate::tree_id must be TreeId::ROOT.", func() {
		update := rootUpdate(0, ent(0, node(a11y.RoleWindow)))
		update.TreeID = treeIDFromU128(1)
		NewTree(update, false)
	})
}

func TestTree_RootNodeHasChildren(t *testing.T) {
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1, 2)),
		ent(1, node(a11y.RoleButton)),
		ent(2, node(a11y.RoleButton)),
	), false)
	state := tree.State()
	eq(t, nodeID(0), must(must(state.NodeByID(nodeID(1))).Parent()).ID())
	eq(t, nodeID(0), must(must(state.NodeByID(nodeID(2))).Parent()).ID())
	eq(t, 2, state.Root().Children().Len())
}

func TestTree_AddChildToRootNode(t *testing.T) {
	rootNode := a11y.NewNode(a11y.RoleWindow)
	tree := NewTree(rootUpdate(0, ent(0, rootNode.Clone())), false)
	eq(t, 0, tree.State().Root().Children().Len())
	second := rootNode
	second.PushChild(1)
	secondUpdate := treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
		ent(0, second),
		ent(1, node(a11y.RoleRootWebArea)),
	)
	gotNewChildNode, gotUpdatedRootNode := false, false
	h := strictHandler("expected only new child node and updated root node")
	h.added = func(n *Node) {
		if n.ID() == nodeID(1) {
			gotNewChildNode = true
			return
		}
		panic("expected only new child node and updated root node")
	}
	h.updated = func(oldNode, newNode *Node) {
		if newNode.ID() == nodeID(0) && len(oldNode.Data().Children()) == 0 &&
			slices.Equal(newNode.Data().Children(), []a11y.NodeID{1}) {
			gotUpdatedRootNode = true
			return
		}
		panic("expected only new child node and updated root node")
	}
	tree.UpdateAndProcessChanges(secondUpdate, h)
	check(t, gotNewChildNode)
	check(t, gotUpdatedRootNode)
	state := tree.State()
	eq(t, 1, state.Root().Children().Len())
	eq(t, nodeID(1), state.Root().Children().collect()[0].ID())
	eq(t, nodeID(0), must(must(state.NodeByID(nodeID(1))).Parent()).ID())
}

func TestTree_RemoveChildFromRootNode(t *testing.T) {
	rootNode := a11y.NewNode(a11y.RoleWindow)
	first := rootNode.Clone()
	first.PushChild(1)
	tree := NewTree(rootUpdate(0, ent(0, first), ent(1, node(a11y.RoleRootWebArea))), false)
	eq(t, 1, tree.State().Root().Children().Len())
	secondUpdate := treeUpdate(a11y.TreeIDRoot(), noTree(), 0, ent(0, rootNode))
	gotUpdatedRootNode, gotRemovedChildNode := false, false
	const msg = "expected only removed child node and updated root node"
	h := strictHandler(msg)
	h.updated = func(oldNode, newNode *Node) {
		if newNode.ID() == nodeID(0) && slices.Equal(oldNode.Data().Children(), []a11y.NodeID{1}) &&
			len(newNode.Data().Children()) == 0 {
			gotUpdatedRootNode = true
			return
		}
		panic(msg)
	}
	h.removed = func(n *Node) {
		if n.ID() == nodeID(1) {
			gotRemovedChildNode = true
			return
		}
		panic(msg)
	}
	tree.UpdateAndProcessChanges(secondUpdate, h)
	check(t, gotUpdatedRootNode)
	check(t, gotRemovedChildNode)
	eq(t, 0, tree.State().Root().Children().Len())
	_, ok := tree.State().NodeByID(nodeID(1))
	check(t, !ok)
}

func TestTree_MoveFocusBetweenSiblings(t *testing.T) {
	tree := NewTree(rootUpdate(1,
		ent(0, node(a11y.RoleWindow, 1, 2)),
		ent(1, node(a11y.RoleButton)),
		ent(2, node(a11y.RoleButton)),
	), true)
	check(t, must(tree.State().NodeByID(nodeID(1))).IsFocused())
	secondUpdate := treeUpdate(a11y.TreeIDRoot(), noTree(), 2)
	gotOld, gotNew, gotFocus := false, false, false
	const msg = "expected only focus change"
	h := strictHandler(msg)
	h.updated = func(oldNode, newNode *Node) {
		if oldNode.ID() == nodeID(1) && newNode.ID() == nodeID(1) && oldNode.IsFocused() && !newNode.IsFocused() {
			gotOld = true
			return
		}
		if oldNode.ID() == nodeID(2) && newNode.ID() == nodeID(2) && !oldNode.IsFocused() && newNode.IsFocused() {
			gotNew = true
			return
		}
		panic(msg)
	}
	h.focus = func(oldNode, newNode *Node) {
		if oldNode != nil && newNode != nil && oldNode.ID() == nodeID(1) && newNode.ID() == nodeID(2) {
			gotFocus = true
			return
		}
		panic(msg)
	}
	tree.UpdateAndProcessChanges(secondUpdate, h)
	check(t, gotOld)
	check(t, gotNew)
	check(t, gotFocus)
	check(t, must(tree.State().NodeByID(nodeID(2))).IsFocused())
	check(t, !must(tree.State().NodeByID(nodeID(1))).IsFocused())
}

func TestTree_UpdateNode(t *testing.T) {
	childNode := a11y.NewNode(a11y.RoleButton)
	first := childNode.Clone()
	first.SetLabel("foo")
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow, 1)), ent(1, first)), false)
	eq(t, gpuiutil.Some("foo"), gpuiutil.OptionFrom(must(tree.State().NodeByID(nodeID(1))).Label()))
	second := childNode
	second.SetLabel("bar")
	secondUpdate := treeUpdate(a11y.TreeIDRoot(), noTree(), 0, ent(1, second))
	gotUpdatedChildNode := false
	const msg = "expected only updated child node"
	h := strictHandler(msg)
	h.updated = func(oldNode, newNode *Node) {
		if newNode.ID() == nodeID(1) && gpuiutil.OptionFrom(oldNode.Label()) == gpuiutil.Some("foo") &&
			gpuiutil.OptionFrom(newNode.Label()) == gpuiutil.Some("bar") {
			gotUpdatedChildNode = true
			return
		}
		panic(msg)
	}
	tree.UpdateAndProcessChanges(secondUpdate, h)
	check(t, gotUpdatedChildNode)
	eq(t, gpuiutil.Some("bar"), gpuiutil.OptionFrom(must(tree.State().NodeByID(nodeID(1))).Label()))
}

func TestTree_NoChangeUpdate(t *testing.T) {
	makeUpdate := func() a11y.TreeUpdate {
		button := a11y.NewNode(a11y.RoleButton)
		button.SetLabel("foo")
		return rootUpdate(0, ent(0, node(a11y.RoleWindow, 1)), ent(1, button))
	}
	tree := NewTree(makeUpdate(), false)
	tree.UpdateAndProcessChanges(makeUpdate(), strictHandler("expected no changes"))
}

func TestTree_MoveNode(t *testing.T) {
	gotUpdatedRoot, gotUpdatedChild, gotRemovedContainer := false, false, false
	const msg = "expected only updated root and removed container"
	h := strictHandler(msg)
	h.updated = func(oldNode, newNode *Node) {
		if newNode.ID() == nodeID(0) && slices.Equal(oldNode.ChildIDs().collect(), []NodeID{nodeID(1)}) &&
			slices.Equal(newNode.ChildIDs().collect(), []NodeID{nodeID(2)}) {
			gotUpdatedRoot = true
			return
		}
		if newNode.ID() == nodeID(2) && parentOpt(*oldNode) == someID(nodeID(1)) && parentOpt(*newNode) == someID(nodeID(0)) {
			gotUpdatedChild = true
			return
		}
		panic(msg)
	}
	h.removed = func(n *Node) {
		if n.ID() == nodeID(1) {
			gotRemovedContainer = true
			return
		}
		panic(msg)
	}
	root := node(a11y.RoleWindow, 1)
	container := node(a11y.RoleGenericContainer, 2)
	tree := NewTree(rootUpdate(0,
		ent(0, root.Clone()),
		ent(1, container),
		ent(2, node(a11y.RoleButton)),
	), false)
	root.SetChildren([]a11y.NodeID{2})
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0, ent(0, root)), h)
	check(t, gotUpdatedRoot)
	check(t, gotUpdatedChild)
	check(t, gotRemovedContainer)
	check(t, slices.Equal(must(tree.State().NodeByID(nodeID(0))).ChildIDs().collect(), []NodeID{nodeID(2)}))
	_, ok := tree.State().NodeByID(nodeID(1))
	check(t, !ok)
	eq(t, someID(nodeID(0)), parentOpt(must(tree.State().NodeByID(nodeID(2)))))
}

func TestTree_GraftNodeTracksSubtree(t *testing.T) {
	tree := graftTree(false)
	eq(t, nodeID(1), tree.State().graftParents[subtreeID()])
}

func TestTree_DuplicateGraftParentPanics(t *testing.T) {
	expectPanic(t, "already has a graft parent", func() {
		NewTree(rootUpdate(0,
			ent(0, node(a11y.RoleWindow, 1, 2)),
			ent(1, graftNode(subtreeID())),
			ent(2, graftNode(subtreeID())),
		), false)
	})
}

func TestTree_ReparentSubtreeByRemovingOldGraft(t *testing.T) {
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1, 2)),
		ent(1, graftNode(subtreeID())),
		ent(2, node(a11y.RoleGenericContainer)),
	), false)
	eq(t, nodeID(1), tree.State().graftParents[subtreeID()])
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
		ent(0, node(a11y.RoleWindow, 2)),
		ent(2, graftNode(subtreeID())),
	), noOpHandler)
	eq(t, nodeID(2), tree.State().graftParents[subtreeID()])
}

func TestTree_ReparentSubtreeByClearingOldGraftTreeId(t *testing.T) {
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1, 2)),
		ent(1, graftNode(subtreeID())),
		ent(2, node(a11y.RoleGenericContainer)),
	), false)
	eq(t, nodeID(1), tree.State().graftParents[subtreeID()])
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
		ent(1, node(a11y.RoleGenericContainer)),
		ent(2, graftNode(subtreeID())),
	), noOpHandler)
	eq(t, nodeID(2), tree.State().graftParents[subtreeID()])
}

func TestTree_DuplicateGraftParentOnUpdatePanics(t *testing.T) {
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1, 2)),
		ent(1, graftNode(subtreeID())),
		ent(2, node(a11y.RoleGenericContainer)),
	), false)
	expectPanic(t, "already has a graft parent", func() {
		tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
			ent(2, graftNode(subtreeID())),
		), noOpHandler)
	})
}

func TestTree_GraftRootTreePanics(t *testing.T) {
	expectPanic(t, "Cannot graft the root tree", func() {
		NewTree(rootUpdate(0,
			ent(0, node(a11y.RoleWindow, 1)),
			ent(1, graftNode(a11y.TreeIDRoot())),
		), false)
	})
}

func TestTree_GraftRootTreeOnUpdatePanics(t *testing.T) {
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, node(a11y.RoleGenericContainer)),
	), false)
	expectPanic(t, "Cannot graft the root tree", func() {
		tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
			ent(1, graftNode(a11y.TreeIDRoot())),
		), noOpHandler)
	})
}

func TestTree_NodeByTreeLocalIdFindsRootTreeNode(t *testing.T) {
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, node(a11y.RoleButton)),
	), false)
	root := must(tree.State().NodeByTreeLocalID(0, a11y.TreeIDRoot()))
	eq(t, nodeID(0), root.ID())
	eq(t, a11y.RoleWindow, root.Role())
	child := must(tree.State().NodeByTreeLocalID(1, a11y.TreeIDRoot()))
	eq(t, nodeID(1), child.ID())
	eq(t, a11y.RoleButton, child.Role())
}

func TestTree_NodeByTreeLocalIdFindsSubtreeNode(t *testing.T) {
	tree := graftTree(false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleParagraph)),
	), noOpHandler)
	subRoot := must(tree.State().NodeByTreeLocalID(0, subtreeID()))
	eq(t, subtreeNodeID(0), subRoot.ID())
	eq(t, a11y.RoleDocument, subRoot.Role())
	subChild := must(tree.State().NodeByTreeLocalID(1, subtreeID()))
	eq(t, subtreeNodeID(1), subChild.ID())
	eq(t, a11y.RoleParagraph, subChild.Role())
	graft := must(tree.State().NodeByTreeLocalID(1, a11y.TreeIDRoot()))
	eq(t, nodeID(1), graft.ID())
	eq(t, a11y.RoleGenericContainer, graft.Role())
}

func TestTree_NodeByTreeLocalIdReturnsNoneForUnknownTreeId(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow))), false)
	_, ok := tree.State().NodeByTreeLocalID(0, subtreeID())
	check(t, !ok)
}

func TestTree_NodeByTreeLocalIdReturnsNoneForUnknownLocalId(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow))), false)
	_, ok := tree.State().NodeByTreeLocalID(999, a11y.TreeIDRoot())
	check(t, !ok)
}

func TestTree_SubtreeRootParentIsGraftWhenGraftExistsFirst(t *testing.T) {
	tree := graftTree(false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument)),
	), noOpHandler)
	subtreeRoot := must(tree.State().NodeByID(subtreeNodeID(0)))
	eq(t, someID(nodeID(1)), parentOpt(subtreeRoot))
	graft := must(tree.State().NodeByID(nodeID(1)))
	children := graft.ChildIDs().collect()
	eq(t, 1, len(children))
	eq(t, subtreeNodeID(0), children[0])
}

func TestTree_SubtreePushWithoutGraftPanics(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow))), false)
	expectPanic(t, "no graft node exists for this tree", func() {
		tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
			ent(0, node(a11y.RoleDocument)),
		), noOpHandler)
	})
}

func TestTree_SubtreeUpdateWithoutTreeDataPanics(t *testing.T) {
	tree := graftTree(false)
	expectPanic(t, "subtree does not exist", func() {
		tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), noTree(), 0,
			ent(0, node(a11y.RoleDocument)),
		), noOpHandler)
	})
}

func TestTree_SubtreeNodesRemovedWhenGraftRemoved(t *testing.T) {
	tree := graftTree(false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, graftNode(nestedSubtreeID())),
	), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(nestedSubtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleParagraph)),
	), noOpHandler)
	s := tree.State()
	check(t, s.HasNode(subtreeNodeID(0)))
	check(t, s.HasNode(subtreeNodeID(1)))
	check(t, s.HasNode(nestedSubtreeNodeID(0)))
	check(t, s.HasNode(nestedSubtreeNodeID(1)))
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
		ent(0, node(a11y.RoleWindow, []a11y.NodeID{}...)),
	), noOpHandler)
	s = tree.State()
	check(t, !s.HasNode(subtreeNodeID(0)))
	check(t, !s.HasNode(subtreeNodeID(1)))
	check(t, !s.HasNode(nestedSubtreeNodeID(0)))
	check(t, !s.HasNode(nestedSubtreeNodeID(1)))
	_, ok := s.subtrees[subtreeID()]
	check(t, !ok)
	_, ok = s.subtrees[nestedSubtreeID()]
	check(t, !ok)
}

func TestTree_SubtreeNodesRemovedWhenGraftTreeIdCleared(t *testing.T) {
	tree := graftTree(false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleParagraph)),
	), noOpHandler)
	check(t, tree.State().HasNode(subtreeNodeID(0)))
	check(t, tree.State().HasNode(subtreeNodeID(1)))
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
		ent(1, node(a11y.RoleGenericContainer)),
	), noOpHandler)
	check(t, !tree.State().HasNode(subtreeNodeID(0)))
	check(t, !tree.State().HasNode(subtreeNodeID(1)))
	_, ok := tree.State().subtrees[subtreeID()]
	check(t, !ok)
}

func TestTree_GraftNodeHasNoChildrenWhenSubtreeNotPushed(t *testing.T) {
	tree := graftTree(false)
	graft := must(tree.State().NodeByID(nodeID(1)))
	eq(t, 0, len(graft.ChildIDs().collect()))
	eq(t, 0, graft.Children().Len())
}

func TestTree_GraftNodeWithChildrenPanics(t *testing.T) {
	expectPanic(t, "has both tree_id", func() {
		g := graftNode(subtreeID())
		g.SetChildren([]a11y.NodeID{2})
		NewTree(rootUpdate(0,
			ent(0, node(a11y.RoleWindow, 1)),
			ent(1, g),
			ent(2, node(a11y.RoleButton)),
		), false)
	})
}

func TestTree_NodeAddedCalledWhenSubtreePushed(t *testing.T) {
	tree := graftTree(false)
	h := newRecordingHandler()
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1, 2)),
		ent(1, node(a11y.RoleParagraph)),
		ent(2, node(a11y.RoleButton)),
	), h)
	eq(t, 3, len(h.addedNodes))
	check(t, slices.Contains(h.addedNodes, subtreeNodeID(0)))
	check(t, slices.Contains(h.addedNodes, subtreeNodeID(1)))
	check(t, slices.Contains(h.addedNodes, subtreeNodeID(2)))
}

func TestTree_NodeRemovedCalledWhenGraftRemoved(t *testing.T) {
	tree := graftTree(false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleParagraph)),
	), noOpHandler)
	check(t, tree.State().HasNode(subtreeNodeID(0)))
	check(t, tree.State().HasNode(subtreeNodeID(1)))
	h := newRecordingHandler()
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
		ent(0, node(a11y.RoleWindow, []a11y.NodeID{}...)),
	), h)
	check(t, slices.Contains(h.removedNodes, nodeID(1)))
	check(t, slices.Contains(h.removedNodes, subtreeNodeID(0)))
	check(t, slices.Contains(h.removedNodes, subtreeNodeID(1)))
	eq(t, 3, len(h.removedNodes))
}

func TestTree_NodeUpdatedCalledWhenSubtreeReparented(t *testing.T) {
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1, 2)),
		ent(1, graftNode(subtreeID())),
		ent(2, node(a11y.RoleGenericContainer)),
	), false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument)),
	), noOpHandler)
	eq(t, nodeID(1), must(must(tree.State().NodeByID(subtreeNodeID(0))).Parent()).ID())
	h := newRecordingHandler()
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 0,
		ent(1, node(a11y.RoleGenericContainer)),
		ent(2, graftNode(subtreeID())),
	), h)
	check(t, slices.Contains(h.updatedNodes, subtreeNodeID(0)))
	eq(t, nodeID(2), must(must(tree.State().NodeByID(subtreeNodeID(0))).Parent()).ID())
}

type focusRecorder struct {
	testHandler
	moves [][2]gpuiutil.Option[NodeID]
}

func newFocusRecorder() *focusRecorder {
	h := &focusRecorder{}
	h.focus = func(oldNode, newNode *Node) {
		h.moves = append(h.moves, [2]gpuiutil.Option[NodeID]{optID(oldNode), optID(newNode)})
	}
	return h
}

func TestTree_FocusMovedCalledWhenFocusMovesToSubtree(t *testing.T) {
	tree := graftTree(true)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleButton)),
	), noOpHandler)
	h := newFocusRecorder()
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 1), h)
	eq(t, 1, len(h.moves))
	eq(t, someID(nodeID(0)), h.moves[0][0])
	eq(t, someID(subtreeNodeID(0)), h.moves[0][1])
}

func TestTree_FocusMovedCalledWhenSubtreeFocusChanges(t *testing.T) {
	tree := graftTree(true)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleButton)),
	), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 1), noOpHandler)
	h := newFocusRecorder()
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), noTree(), 1), h)
	eq(t, 1, len(h.moves))
	eq(t, someID(subtreeNodeID(0)), h.moves[0][0])
	eq(t, someID(subtreeNodeID(1)), h.moves[0][1])
}

func TestTree_NestedSubtreeFocusFollowsGraftChain(t *testing.T) {
	tree := graftTree(true)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, graftNode(nestedSubtreeID())),
	), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(nestedSubtreeID(), someTree(0), 1,
		ent(0, node(a11y.RoleGroup, 1)),
		ent(1, node(a11y.RoleButton)),
	), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 1), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), noTree(), 1), noOpHandler)
	eq(t, someID(nestedSubtreeNodeID(1)), gpuiutil.OptionFrom(tree.State().FocusID()))
}

func TestTree_NestedSubtreeFocusUpdateChangesEffectiveFocus(t *testing.T) {
	tree := graftTree(true)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 1,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, graftNode(nestedSubtreeID())),
	), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(nestedSubtreeID(), someTree(0), 1,
		ent(0, node(a11y.RoleGroup, 1, 2)),
		ent(1, node(a11y.RoleButton)),
		ent(2, node(a11y.RoleButton)),
	), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 1), noOpHandler)
	eq(t, someID(nestedSubtreeNodeID(1)), gpuiutil.OptionFrom(tree.State().FocusID()))
	tree.UpdateAndProcessChanges(treeUpdate(nestedSubtreeID(), noTree(), 2), noOpHandler)
	eq(t, someID(nestedSubtreeNodeID(2)), gpuiutil.OptionFrom(tree.State().FocusID()))
}

func TestTree_RemovingNestedSubtreeWhileIntermediateFocusOnGraftPanics(t *testing.T) {
	expectPanic(t, "Graft nodes cannot be focused without their subtree", func() {
		tree := NewTree(rootUpdate(1,
			ent(0, node(a11y.RoleWindow, 1)),
			ent(1, graftNode(subtreeID())),
		), true)
		tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 1,
			ent(0, node(a11y.RoleDocument, 1)),
			ent(1, graftNode(nestedSubtreeID())),
		), noOpHandler)
		tree.UpdateAndProcessChanges(treeUpdate(nestedSubtreeID(), someTree(0), 0,
			ent(0, node(a11y.RoleButton)),
		), noOpHandler)
		tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), noTree(), 1,
			ent(1, node(a11y.RoleGenericContainer)),
		), noOpHandler)
	})
}

func TestTree_NestedSubtreeRootLookupForFocusOnlyUpdate(t *testing.T) {
	tree := graftTree(true)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1, 2)),
		ent(1, graftNode(nestedSubtreeID())),
		ent(2, node(a11y.RoleButton)),
	), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(nestedSubtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleGroup, 1)),
		ent(1, node(a11y.RoleButton)),
	), noOpHandler)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), noTree(), 2), noOpHandler)
	eq(t, subtreeNodeID(2), tree.State().subtrees[subtreeID()].focus)
}

func TestTree_SubtreeRootChangeUpdatesGraftAndParent(t *testing.T) {
	tree := graftTree(false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleParagraph)),
	), noOpHandler)
	h := newRecordingHandler()
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(2), 2,
		ent(2, node(a11y.RoleArticle, 3)),
		ent(3, node(a11y.RoleButton)),
	), h)
	s := tree.State()
	children := must(s.NodeByID(nodeID(1))).ChildIDs().collect()
	eq(t, 1, len(children))
	eq(t, subtreeNodeID(2), children[0])
	newSubtreeRoot := must(s.NodeByID(subtreeNodeID(2)))
	eq(t, someID(nodeID(1)), parentOpt(newSubtreeRoot))
	eq(t, a11y.RoleArticle, newSubtreeRoot.Role())
	check(t, !s.HasNode(subtreeNodeID(0)))
	check(t, !s.HasNode(subtreeNodeID(1)))
	check(t, s.HasNode(subtreeNodeID(2)))
	check(t, s.HasNode(subtreeNodeID(3)))
	check(t, slices.Contains(h.removedNodes, subtreeNodeID(0)))
	check(t, slices.Contains(h.removedNodes, subtreeNodeID(1)))
	check(t, slices.Contains(h.addedNodes, subtreeNodeID(2)))
	check(t, slices.Contains(h.addedNodes, subtreeNodeID(3)))
}

func TestTree_SubtreeRootChangeToExistingChild(t *testing.T) {
	tree := graftTree(false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleArticle, 2)),
		ent(2, node(a11y.RoleParagraph)),
	), noOpHandler)
	s := tree.State()
	eq(t, someID(subtreeNodeID(0)), gpuiutil.Some(must(s.NodeByID(nodeID(1))).ChildIDs().collect()[0]))
	oldRoot := must(s.NodeByID(subtreeNodeID(0)))
	eq(t, a11y.RoleDocument, oldRoot.Role())
	eq(t, someID(nodeID(1)), parentOpt(oldRoot))
	child := must(s.NodeByID(subtreeNodeID(1)))
	eq(t, a11y.RoleArticle, child.Role())
	eq(t, someID(subtreeNodeID(0)), parentOpt(child))
	eq(t, someID(subtreeNodeID(1)), parentOpt(must(s.NodeByID(subtreeNodeID(2)))))
	h := newRecordingHandler()
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(1), 1,
		ent(1, node(a11y.RoleArticle, 2)),
		ent(2, node(a11y.RoleParagraph)),
	), h)
	s = tree.State()
	children := must(s.NodeByID(nodeID(1))).ChildIDs().collect()
	eq(t, 1, len(children))
	eq(t, subtreeNodeID(1), children[0])
	newRoot := must(s.NodeByID(subtreeNodeID(1)))
	eq(t, someID(nodeID(1)), parentOpt(newRoot))
	eq(t, a11y.RoleArticle, newRoot.Role())
	check(t, !s.HasNode(subtreeNodeID(0)))
	eq(t, someID(subtreeNodeID(1)), parentOpt(must(s.NodeByID(subtreeNodeID(2)))))
	check(t, slices.Contains(h.removedNodes, subtreeNodeID(0)))
	check(t, slices.Contains(h.updatedNodes, subtreeNodeID(1)))
	check(t, !slices.Contains(h.addedNodes, subtreeNodeID(1)))
	check(t, !slices.Contains(h.addedNodes, subtreeNodeID(2)))
}

func TestTree_SubtreeRootChangeToNewParentOfOldRoot(t *testing.T) {
	tree := graftTree(false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleParagraph)),
	), noOpHandler)
	h := newRecordingHandler()
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(2), 2,
		ent(2, node(a11y.RoleArticle, 0)),
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleParagraph)),
	), h)
	s := tree.State()
	children := must(s.NodeByID(nodeID(1))).ChildIDs().collect()
	eq(t, 1, len(children))
	eq(t, subtreeNodeID(2), children[0])
	newRoot := must(s.NodeByID(subtreeNodeID(2)))
	eq(t, someID(nodeID(1)), parentOpt(newRoot))
	eq(t, a11y.RoleArticle, newRoot.Role())
	oldRoot := must(s.NodeByID(subtreeNodeID(0)))
	eq(t, someID(subtreeNodeID(2)), parentOpt(oldRoot))
	eq(t, a11y.RoleDocument, oldRoot.Role())
	eq(t, someID(subtreeNodeID(0)), parentOpt(must(s.NodeByID(subtreeNodeID(1)))))
	check(t, slices.Contains(h.addedNodes, subtreeNodeID(2)))
	check(t, slices.Contains(h.updatedNodes, subtreeNodeID(0)))
	check(t, !slices.Contains(h.removedNodes, subtreeNodeID(0)))
	check(t, !slices.Contains(h.removedNodes, subtreeNodeID(1)))
}

func TestTree_SubtreeUpdateWithoutTreePreservesRoot(t *testing.T) {
	tree := graftTree(false)
	original := a11y.NewNode(a11y.RoleParagraph)
	original.SetLabel("original")
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, original),
	), noOpHandler)
	h := newRecordingHandler()
	modified := a11y.NewNode(a11y.RoleParagraph)
	modified.SetLabel("modified")
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID(), noTree(), 0, ent(1, modified)), h)
	s := tree.State()
	subtreeRoot := must(s.NodeByID(subtreeNodeID(0)))
	eq(t, a11y.RoleDocument, subtreeRoot.Role())
	eq(t, someID(nodeID(1)), parentOpt(subtreeRoot))
	eq(t, subtreeNodeID(0), must(s.NodeByID(nodeID(1))).ChildIDs().collect()[0])
	eq(t, gpuiutil.Some("modified"), gpuiutil.OptionFrom(must(s.NodeByID(subtreeNodeID(1))).Label()))
	check(t, len(h.removedNodes) == 0)
	check(t, len(h.addedNodes) == 0)
	check(t, slices.Contains(h.updatedNodes, subtreeNodeID(1)))
	check(t, !slices.Contains(h.updatedNodes, subtreeNodeID(0)))
}

func TestTree_FocusReturnsFocusedNode(t *testing.T) {
	tree := NewTree(rootUpdate(1,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, node(a11y.RoleButton)),
	), true)
	eq(t, nodeID(1), must(tree.State().Focus()).ID())
}

func listBox(active a11y.NodeID, children ...a11y.NodeID) a11y.Node {
	n := node(a11y.RoleListBox, children...)
	n.SetActiveDescendant(active)
	return n
}

func TestTree_FocusReturnsActiveDescendant(t *testing.T) {
	tree := NewTree(rootUpdate(1,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, listBox(2, 2)),
		ent(2, node(a11y.RoleListBoxOption)),
	), true)
	eq(t, nodeID(2), must(tree.State().Focus()).ID())
}

func TestTree_FocusMovedWhenActiveDescendantChanges(t *testing.T) {
	tree := NewTree(rootUpdate(1,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, listBox(2, 2, 3)),
		ent(2, node(a11y.RoleListBoxOption)),
		ent(3, node(a11y.RoleListBoxOption)),
	), true)
	h := newFocusRecorder()
	tree.UpdateAndProcessChanges(treeUpdate(a11y.TreeIDRoot(), noTree(), 1, ent(1, listBox(3, 2, 3))), h)
	check(t, len(h.moves) > 0)
	last := h.moves[len(h.moves)-1]
	eq(t, someID(nodeID(2)), last[0])
	eq(t, someID(nodeID(3)), last[1])
}

func treeWithToolkitInfo(name, version gpuiutil.Option[string]) *Tree {
	treeData := a11y.NewTree(0)
	treeData.ToolkitName = name
	treeData.ToolkitVersion = version
	return NewTree(a11y.TreeUpdate{
		Nodes:  []a11y.NodeEntry{ent(0, node(a11y.RoleWindow))},
		Tree:   gpuiutil.Some(treeData),
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	}, false)
}

func TestTree_ToolkitInfoIsReportedWhenBothNameAndVersionAreSet(t *testing.T) {
	state := treeWithToolkitInfo(gpuiutil.Some("egui"), gpuiutil.Some("0.30.0")).State()
	eq(t, "egui", state.ToolkitName())
	eq(t, gpuiutil.Some("0.30.0"), gpuiutil.OptionFrom(state.ToolkitVersion()))
}

func TestTree_ToolkitVersionIsNoneWhenOnlyNameIsSet(t *testing.T) {
	state := treeWithToolkitInfo(gpuiutil.Some("egui"), gpuiutil.None[string]()).State()
	eq(t, "egui", state.ToolkitName())
	eq(t, gpuiutil.None[string](), gpuiutil.OptionFrom(state.ToolkitVersion()))
}

func TestTree_ToolkitNameFallsBackToAccesskitWhenUnset(t *testing.T) {
	state := treeWithToolkitInfo(gpuiutil.None[string](), gpuiutil.None[string]()).State()
	eq(t, "AccessKit", state.ToolkitName())
	eq(t, gpuiutil.None[string](), gpuiutil.OptionFrom(state.ToolkitVersion()))
}

func TestTree_ToolkitVersionIsSuppressedWhenNameIsUnset(t *testing.T) {
	state := treeWithToolkitInfo(gpuiutil.None[string](), gpuiutil.Some("0.30.0")).State()
	eq(t, "AccessKit", state.ToolkitName())
	eq(t, gpuiutil.None[string](), gpuiutil.OptionFrom(state.ToolkitVersion()))
}
