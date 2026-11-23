package consumer

import (
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

// NodeID identifies a node across the root tree and its grafted subtrees.
type NodeID struct {
	treeIndex treeIndex
	localID   a11y.NodeID
}

func newNodeID(localID a11y.NodeID, ti treeIndex) NodeID {
	return NodeID{treeIndex: ti, localID: localID}
}

func (id NodeID) withSameTree(localID a11y.NodeID) NodeID {
	return NodeID{treeIndex: id.treeIndex, localID: localID}
}

func (id NodeID) toComponents() (a11y.NodeID, treeIndex) {
	return id.localID, id.treeIndex
}

type parentAndIndex struct {
	parent NodeID
	index  int
}

type nodeState struct {
	parentAndIndex gpuiutil.Option[parentAndIndex]
	data           a11y.Node
}

func (n *nodeState) clone() *nodeState {
	return &nodeState{parentAndIndex: n.parentAndIndex, data: n.data.Clone()}
}

// Node is a view of one node in a State.
type Node struct {
	TreeState *State
	id        NodeID
	state     *nodeState
}

// Data returns the node's data.
func (n Node) Data() *a11y.Node {
	return &n.state.data
}

// IsFocused reports whether the node has focus, taking active descendants into account.
func (n Node) IsFocused() bool {
	focusID, ok := n.TreeState.FocusID()
	if !ok {
		return false
	}
	if focusID == n.ID() {
		node, ok := n.TreeState.NodeByID(focusID)
		if !ok {
			return true
		}
		_, dominated := node.ActiveDescendant()
		return !dominated
	}
	focused, ok := n.TreeState.NodeByID(focusID)
	if !ok {
		return false
	}
	ad, ok := focused.ActiveDescendant()
	return ok && ad.ID() == n.ID()
}

// IsGraft reports whether the node is a graft node.
func (n Node) IsGraft() bool {
	_, ok := n.state.data.TreeID()
	return ok
}

// ParentID returns the ID of the node's parent.
func (n Node) ParentID() (NodeID, bool) {
	pi, ok := n.state.parentAndIndex.Get()
	return pi.parent, ok
}

// Parent returns the node's parent.
func (n Node) Parent() (Node, bool) {
	id, ok := n.ParentID()
	if !ok {
		return Node{}, false
	}
	return n.TreeState.mustNode(id), true
}

func (n Node) graftChildID() (NodeID, bool) {
	treeID, ok := n.state.data.TreeID()
	if !ok {
		return NodeID{}, false
	}
	return n.TreeState.SubtreeRoot(treeID)
}

// ChildIDs returns the IDs of the node's children.
func (n Node) ChildIDs() []NodeID {
	if n.IsGraft() {
		if id, ok := n.graftChildID(); ok {
			return []NodeID{id}
		}
		return nil
	}
	children := n.state.data.Children()
	ids := make([]NodeID, len(children))
	for i, c := range children {
		ids[i] = n.id.withSameTree(c)
	}
	return ids
}

// Children returns the node's children.
func (n Node) Children() []Node {
	ids := n.ChildIDs()
	nodes := make([]Node, len(ids))
	for i, id := range ids {
		nodes[i] = n.TreeState.mustNode(id)
	}
	return nodes
}

// ID returns the node's ID.
func (n Node) ID() NodeID {
	return n.id
}

// Role returns the node's role.
func (n Node) Role() a11y.Role {
	return n.Data().Role()
}

// IsDialog reports whether the node is a dialog or alert dialog.
func (n Node) IsDialog() bool {
	switch n.Role() {
	case a11y.RoleAlertDialog, a11y.RoleDialog:
		return true
	}
	return false
}

// Label returns the node's label.
func (n Node) Label() (string, bool) {
	if label, ok := n.Data().Label(); ok {
		return label, true
	}
	return n.labelFromLabelledBy()
}

func (n Node) labelFromLabelledBy() (string, bool) {
	if len(n.Data().LabelledBy()) == 0 {
		switch n.Role() {
		case a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleDefaultButton, a11y.RoleLink,
			a11y.RoleMenuItem, a11y.RoleMenuItemCheckBox, a11y.RoleMenuItemRadio, a11y.RoleRadioButton:
		default:
			return "", false
		}
	}
	panic("gpui: unimplemented consumer.Node.labelFromLabelledBy")
}

// ActiveDescendant returns the node's active descendant.
func (n Node) ActiveDescendant() (Node, bool) {
	id, ok := n.state.data.ActiveDescendant()
	if !ok {
		return Node{}, false
	}
	return n.TreeState.NodeByID(n.id.withSameTree(id))
}
