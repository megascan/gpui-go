package consumer

import (
	"fmt"
	"strings"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

type treeIndex uint32

type treeIndexMap struct {
	idToIndex map[a11y.TreeID]treeIndex
	indexToID map[treeIndex]a11y.TreeID
	next      uint32
}

func (m *treeIndexMap) getOrCreateIndex(id a11y.TreeID) treeIndex {
	if index, ok := m.idToIndex[id]; ok {
		return index
	}
	if m.idToIndex == nil {
		m.idToIndex = map[a11y.TreeID]treeIndex{}
		m.indexToID = map[treeIndex]a11y.TreeID{}
	}
	index := treeIndex(m.next)
	m.next++
	m.indexToID[index] = id
	m.idToIndex[id] = index
	return index
}

func (m *treeIndexMap) getIndex(id a11y.TreeID) (treeIndex, bool) {
	index, ok := m.idToIndex[id]
	return index, ok
}

func (m *treeIndexMap) getID(index treeIndex) (a11y.TreeID, bool) {
	id, ok := m.indexToID[index]
	return id, ok
}

func (m *treeIndexMap) clone() treeIndexMap {
	c := treeIndexMap{next: m.next}
	if m.idToIndex != nil {
		c.idToIndex = make(map[a11y.TreeID]treeIndex, len(m.idToIndex))
		for k, v := range m.idToIndex {
			c.idToIndex[k] = v
		}
		c.indexToID = make(map[treeIndex]a11y.TreeID, len(m.indexToID))
		for k, v := range m.indexToID {
			c.indexToID[k] = v
		}
	}
	return c
}

type subtreeState struct {
	root  NodeID
	focus NodeID
}

// State is a snapshot of an accessibility tree.
type State struct {
	nodes         map[NodeID]*nodeState
	data          a11y.Tree
	root          NodeID
	focus         NodeID
	isHostFocused bool
	subtrees      map[a11y.TreeID]subtreeState
	graftParents  map[a11y.TreeID]NodeID
	treeIndexMap  treeIndexMap
}

type set[K comparable] map[K]struct{}

func (s set[K]) insert(k K) bool {
	if _, ok := s[k]; ok {
		return false
	}
	s[k] = struct{}{}
	return true
}

func (s set[K]) contains(k K) bool {
	_, ok := s[k]
	return ok
}

type internalChanges struct {
	addedNodeIDs   set[NodeID]
	updatedNodeIDs set[NodeID]
	removedNodeIDs set[NodeID]
}

func newInternalChanges() *internalChanges {
	return &internalChanges{addedNodeIDs: set[NodeID]{}, updatedNodeIDs: set[NodeID]{}, removedNodeIDs: set[NodeID]{}}
}

func (s *State) clone() State {
	c := *s
	c.nodes = make(map[NodeID]*nodeState, len(s.nodes))
	for k, v := range s.nodes {
		c.nodes[k] = v.clone()
	}
	c.subtrees = cloneMap(s.subtrees)
	c.graftParents = cloneMap(s.graftParents)
	c.treeIndexMap = s.treeIndexMap.clone()
	return c
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	c := make(map[K]V, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func (s *State) validateGlobal() {
	if _, ok := s.nodes[s.root]; !ok {
		panic(fmt.Sprintf("Root ID %v is not in the node list", s.data.Root))
	}
	if _, ok := s.nodes[s.focus]; !ok {
		local, _ := s.focus.toComponents()
		panic(fmt.Sprintf("Focused ID %v is not in the node list", local))
	}
}

func (s *State) computeEffectiveFocus() NodeID {
	rootSubtree, ok := s.subtrees[a11y.TreeIDRoot()]
	if !ok {
		return s.focus
	}
	currentFocus := rootSubtree.focus
	for {
		ns, ok := s.nodes[currentFocus]
		if !ok {
			break
		}
		subtreeID, ok := ns.data.TreeID()
		if !ok {
			break
		}
		subtree, ok := s.subtrees[subtreeID]
		if !ok {
			local, _ := currentFocus.toComponents()
			panic(fmt.Sprintf("Focus is on graft node %v but subtree %v does not exist. "+
				"Graft nodes cannot be focused without their subtree.", local, subtreeID))
		}
		currentFocus = subtree.focus
	}
	return currentFocus
}

func recordGraft(pendingGrafts map[a11y.TreeID]NodeID, subtreeID a11y.TreeID, graftNodeID NodeID) {
	if subtreeID == a11y.TreeIDRoot() {
		panic("Cannot graft the root tree")
	}
	if existingGraft, ok := pendingGrafts[subtreeID]; ok {
		e, _ := existingGraft.toComponents()
		g, _ := graftNodeID.toComponents()
		panic(fmt.Sprintf("Subtree %v already has a graft parent %v, cannot assign to %v", subtreeID, e, g))
	}
	pendingGrafts[subtreeID] = graftNodeID
}

func addNode(nodes map[NodeID]*nodeState, pendingGrafts map[a11y.TreeID]NodeID, changes *internalChanges, pi gpuiutil.Option[parentAndIndex], id NodeID, data a11y.Node) {
	if subtreeID, ok := data.TreeID(); ok {
		if len(data.Children()) != 0 {
			local, _ := id.toComponents()
			panic(fmt.Sprintf("Node %v has both tree_id and children. "+
				"A graft node's only child comes from its subtree.", local))
		}
		recordGraft(pendingGrafts, subtreeID, id)
	}
	nodes[id] = &nodeState{parentAndIndex: pi, data: data.Clone()}
	if changes != nil {
		changes.addedNodeIDs.insert(id)
	}
}

func traverseUnreachable(nodes map[NodeID]*nodeState, graftsToRemove set[a11y.TreeID], changes *internalChanges, seenChildIDs set[NodeID], newTreeRoot gpuiutil.Option[NodeID], root NodeID) {
	stack := []NodeID{root}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if changes != nil {
			changes.removedNodeIDs.insert(id)
		}
		node, ok := nodes[id]
		if !ok {
			panic("called `Option::unwrap()` on a `None` value")
		}
		delete(nodes, id)
		if subtreeID, ok := node.data.TreeID(); ok {
			graftsToRemove.insert(subtreeID)
		}
		_, ti := id.toComponents()
		for _, childID := range node.data.Children() {
			childNodeID := newNodeID(childID, ti)
			if !seenChildIDs.contains(childNodeID) && newTreeRoot != gpuiutil.Some(childNodeID) {
				stack = append(stack, childNodeID)
			}
		}
	}
}

func traverseSubtree(nodes map[NodeID]*nodeState, subtreesToRemove *[]a11y.TreeID, subtreesQueued set[a11y.TreeID], changes *internalChanges, root NodeID) {
	stack := []NodeID{root}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node, ok := nodes[id]
		if !ok {
			continue
		}
		delete(nodes, id)
		if changes != nil {
			changes.removedNodeIDs.insert(id)
		}
		if nestedSubtreeID, ok := node.data.TreeID(); ok {
			if subtreesQueued.insert(nestedSubtreeID) {
				*subtreesToRemove = append(*subtreesToRemove, nestedSubtreeID)
			}
		}
		_, ti := id.toComponents()
		for _, childID := range node.data.Children() {
			stack = append(stack, newNodeID(childID, ti))
		}
	}
}

func (s *State) update(update a11y.TreeUpdate, isHostFocused bool, changes *internalChanges) {
	ti := s.treeIndexMap.getOrCreateIndex(update.TreeID)
	mapID := func(id a11y.NodeID) NodeID { return newNodeID(id, ti) }

	unreachable := set[NodeID]{}
	seenChildIDs := set[NodeID]{}

	treeID := update.TreeID
	if treeID != a11y.TreeIDRoot() {
		_, subtreeExists := s.subtrees[treeID]
		if _, hasGraft := s.graftParents[treeID]; update.Tree.IsSome() && !hasGraft {
			panic(fmt.Sprintf("Cannot push subtree %v: no graft node exists for this tree. "+
				"Push the graft node (with tree_id property set) before pushing the subtree.", treeID))
		}
		if !subtreeExists && update.Tree.IsNone() {
			panic(fmt.Sprintf("Cannot update subtree %v: subtree does not exist. "+
				"The first update for a subtree must include tree data.", treeID))
		}
	}

	var newTreeRoot gpuiutil.Option[NodeID]
	if tree, ok := update.Tree.Get(); ok {
		newRoot := mapID(tree.Root)
		if treeID == a11y.TreeIDRoot() {
			if tree.Root != s.data.Root {
				unreachable.insert(s.root)
			}
			s.root = newRoot
			s.data = tree
		} else if subtree, ok := s.subtrees[treeID]; ok {
			if subtree.root != newRoot {
				unreachable.insert(subtree.root)
			}
		}
		newTreeRoot = gpuiutil.Some(newRoot)
	}

	var root a11y.NodeID
	if r, ok := newTreeRoot.Get(); ok {
		root, _ = r.toComponents()
	} else if subtree, ok := s.subtrees[treeID]; ok {
		root, _ = subtree.root.toComponents()
	} else {
		root = s.data.Root
	}

	pendingNodes := map[NodeID]a11y.Node{}
	pendingChildren := map[NodeID]parentAndIndex{}
	pendingGrafts := map[a11y.TreeID]NodeID{}
	graftsToRemove := set[a11y.TreeID]{}

	for _, entry := range update.Nodes {
		localNodeID, nodeData := entry.ID, entry.Node
		nodeID := mapID(localNodeID)
		delete(unreachable, nodeID)

		for childIndex, childID := range nodeData.Children() {
			mappedChildID := mapID(childID)
			if !seenChildIDs.insert(mappedChildID) {
				panic(fmt.Sprintf("TreeUpdate includes duplicate child %v", childID))
			}
			delete(unreachable, mappedChildID)
			pi := parentAndIndex{nodeID, childIndex}
			if childState, ok := s.nodes[mappedChildID]; ok {
				if childState.parentAndIndex != gpuiutil.Some(pi) {
					childState.parentAndIndex = gpuiutil.Some(pi)
					if changes != nil {
						changes.updatedNodeIDs.insert(mappedChildID)
					}
				}
			} else if childData, ok := pendingNodes[mappedChildID]; ok {
				delete(pendingNodes, mappedChildID)
				addNode(s.nodes, pendingGrafts, changes, gpuiutil.Some(pi), mappedChildID, childData)
			} else {
				pendingChildren[mappedChildID] = pi
			}
		}

		if ns, ok := s.nodes[nodeID]; ok {
			if localNodeID == root {
				ns.parentAndIndex = gpuiutil.None[parentAndIndex]()
			}
			for _, childID := range ns.data.Children() {
				mappedExistingChildID := mapID(childID)
				if !seenChildIDs.contains(mappedExistingChildID) {
					unreachable.insert(mappedExistingChildID)
				}
			}
			if !ns.data.Equal(&nodeData) {
				newTreeID, newOK := nodeData.TreeID()
				if newOK && len(nodeData.Children()) != 0 {
					local, _ := nodeID.toComponents()
					panic(fmt.Sprintf("Node %v has both tree_id and children. "+
						"A graft node's only child comes from its subtree.", local))
				}
				oldTreeID, oldOK := ns.data.TreeID()
				if gpuiutil.OptionFrom(oldTreeID, oldOK) != gpuiutil.OptionFrom(newTreeID, newOK) {
					if oldOK {
						graftsToRemove.insert(oldTreeID)
					}
					if newOK {
						recordGraft(pendingGrafts, newTreeID, nodeID)
					}
				}
				ns.data = nodeData.Clone()
				if changes != nil {
					changes.updatedNodeIDs.insert(nodeID)
				}
			}
		} else if pi, ok := pendingChildren[nodeID]; ok {
			delete(pendingChildren, nodeID)
			addNode(s.nodes, pendingGrafts, changes, gpuiutil.Some(pi), nodeID, nodeData)
		} else if localNodeID == root {
			addNode(s.nodes, pendingGrafts, changes, gpuiutil.None[parentAndIndex](), nodeID, nodeData)
		} else {
			pendingNodes[nodeID] = nodeData
		}
	}

	if len(pendingNodes) != 0 {
		panic(fmt.Sprintf("TreeUpdate includes %d nodes which are neither in the current tree nor a child of another node from the update: %s",
			len(pendingNodes), shortNodeList(pendingNodes)))
	}
	if len(pendingChildren) != 0 {
		panic(fmt.Sprintf("TreeUpdate's nodes include %d children ids which are neither in the current tree nor the ID of another node from the update: %s",
			len(pendingChildren), shortNodeList(pendingChildren)))
	}

	treeFocus := mapID(update.Focus)
	if newRoot, ok := newTreeRoot.Get(); ok {
		s.subtrees[treeID] = subtreeState{root: newRoot, focus: treeFocus}
	} else if subtree, ok := s.subtrees[treeID]; ok {
		subtree.focus = treeFocus
		s.subtrees[treeID] = subtree
	} else if treeID == a11y.TreeIDRoot() {
		s.subtrees[treeID] = subtreeState{root: s.root, focus: treeFocus}
	}

	s.isHostFocused = isHostFocused

	for id := range unreachable {
		traverseUnreachable(s.nodes, graftsToRemove, changes, seenChildIDs, newTreeRoot, id)
	}

	subtreesQueued := graftsToRemove
	subtreesToRemove := make([]a11y.TreeID, 0, len(subtreesQueued))
	for id := range subtreesQueued {
		subtreesToRemove = append(subtreesToRemove, id)
	}
	for i := 0; i < len(subtreesToRemove); {
		subtreeID := subtreesToRemove[i]
		i++
		if _, ok := s.graftParents[subtreeID]; !ok {
			continue
		}
		delete(s.graftParents, subtreeID)
		if _, ok := pendingGrafts[subtreeID]; ok {
			continue
		}
		if subtree, ok := s.subtrees[subtreeID]; ok {
			delete(s.subtrees, subtreeID)
			traverseSubtree(s.nodes, &subtreesToRemove, subtreesQueued, changes, subtree.root)
		}
	}

	for subtreeID, nodeID := range pendingGrafts {
		if existingGraft, ok := s.graftParents[subtreeID]; ok {
			e, _ := existingGraft.toComponents()
			g, _ := nodeID.toComponents()
			panic(fmt.Sprintf("Subtree %v already has a graft parent %v, cannot assign to %v", subtreeID, e, g))
		}
		s.graftParents[subtreeID] = nodeID
		if subtree, ok := s.subtrees[subtreeID]; ok {
			subtreeRootID := subtree.root
			if rootState, ok := s.nodes[subtreeRootID]; ok {
				rootState.parentAndIndex = gpuiutil.Some(parentAndIndex{nodeID, 0})
				if changes != nil && !changes.addedNodeIDs.contains(subtreeRootID) {
					changes.updatedNodeIDs.insert(subtreeRootID)
				}
			}
		}
	}

	if newRootID, ok := newTreeRoot.Get(); ok {
		if graftNodeID, ok := s.graftParents[treeID]; ok {
			if rootState, ok := s.nodes[newRootID]; ok {
				rootState.parentAndIndex = gpuiutil.Some(parentAndIndex{graftNodeID, 0})
				if changes != nil && !changes.addedNodeIDs.contains(newRootID) {
					changes.updatedNodeIDs.insert(newRootID)
				}
			}
		}
	}

	s.focus = s.computeEffectiveFocus()

	s.validateGlobal()
}

func (s *State) updateHostFocusState(isHostFocused bool, changes *internalChanges) {
	focus, _ := s.focus.toComponents()
	update := a11y.TreeUpdate{
		Nodes:  nil,
		Tree:   gpuiutil.None[a11y.Tree](),
		TreeID: a11y.TreeIDRoot(),
		Focus:  focus,
	}
	s.update(update, isHostFocused, changes)
}

// HasNode reports whether the tree contains a node with the given ID.
func (s *State) HasNode(id NodeID) bool {
	_, ok := s.nodes[id]
	return ok
}

// NodeByID returns the node with the given ID.
func (s *State) NodeByID(id NodeID) (Node, bool) {
	ns, ok := s.nodes[id]
	if !ok {
		return Node{}, false
	}
	return Node{TreeState: s, id: id, state: ns}, true
}

// NodeByTreeLocalID returns the node with the given local ID in the tree with the given ID.
func (s *State) NodeByTreeLocalID(localNodeID a11y.NodeID, treeID a11y.TreeID) (Node, bool) {
	ti, ok := s.treeIndexMap.getIndex(treeID)
	if !ok {
		return Node{}, false
	}
	return s.NodeByID(newNodeID(localNodeID, ti))
}

// RootID returns the ID of the root node.
func (s *State) RootID() NodeID {
	return s.root
}

// Root returns the root node.
func (s *State) Root() Node {
	return s.mustNode(s.RootID())
}

func (s *State) mustNode(id NodeID) Node {
	n, ok := s.NodeByID(id)
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return n
}

// SubtreeRoot returns the root NodeID of the subtree with the given TreeID, if it exists.
func (s *State) SubtreeRoot(treeID a11y.TreeID) (NodeID, bool) {
	st, ok := s.subtrees[treeID]
	return st.root, ok
}

// IsHostFocused reports whether the host window has focus.
func (s *State) IsHostFocused() bool {
	return s.isHostFocused
}

// FocusIDInTree returns the ID of the focused node, regardless of host focus.
func (s *State) FocusIDInTree() NodeID {
	return s.focus
}

// FocusInTree returns the focused node, regardless of host focus.
func (s *State) FocusInTree() Node {
	return s.mustNode(s.FocusIDInTree())
}

// FocusID returns the ID of the focused node when the host is focused.
func (s *State) FocusID() (NodeID, bool) {
	return s.focus, s.isHostFocused
}

// Focus returns the focused node, or its active descendant, when the host is focused.
func (s *State) Focus() (Node, bool) {
	id, ok := s.FocusID()
	if !ok {
		return Node{}, false
	}
	focused := s.mustNode(id)
	if ad, ok := focused.ActiveDescendant(); ok {
		return ad, true
	}
	return focused, true
}

// ActiveDialog returns the nearest dialog containing the focused node.
func (s *State) ActiveDialog() (Node, bool) {
	candidate, ok := s.Focus()
	for ok {
		if candidate.IsDialog() {
			return candidate, true
		}
		candidate, ok = candidate.Parent()
	}
	return Node{}, false
}

// ToolkitName returns the toolkit name, or "AccessKit" when unset.
func (s *State) ToolkitName() string {
	return s.data.ToolkitName.UnwrapOr("AccessKit")
}

// ToolkitVersion returns the toolkit version, only when the toolkit name is set too.
func (s *State) ToolkitVersion() (string, bool) {
	v, ok := s.data.ToolkitVersion.Get()
	if !ok || s.data.ToolkitName.IsNone() {
		return "", false
	}
	return v, true
}

// LocateNode returns the local ID and tree ID of the node with the given ID.
func (s *State) LocateNode(nodeID NodeID) (a11y.NodeID, a11y.TreeID, bool) {
	if !s.HasNode(nodeID) {
		return 0, a11y.TreeID{}, false
	}
	localID, ti := nodeID.toComponents()
	treeID, ok := s.treeIndexMap.getID(ti)
	if !ok {
		return 0, a11y.TreeID{}, false
	}
	return localID, treeID, true
}

// ChangeHandler receives the changes made by a tree update. A nil *Node means None.
type ChangeHandler interface {
	NodeAdded(node *Node)
	NodeUpdated(oldNode, newNode *Node)
	FocusMoved(oldNode, newNode *Node)
	NodeRemoved(node *Node)
}

// Tree keeps the current state and the state being built by the next update.
type Tree struct {
	state     State
	nextState State
}

// NewTree builds a tree from its initial update, which must carry root tree data.
func NewTree(initialState a11y.TreeUpdate, isHostFocused bool) *Tree {
	tree, ok := initialState.Tree.Get()
	initialState.Tree = gpuiutil.None[a11y.Tree]()
	if !ok {
		panic("Tried to initialize the accessibility tree without a root tree. TreeUpdate::tree must be Some.")
	}
	if initialState.TreeID != a11y.TreeIDRoot() {
		panic("Cannot initialize with a subtree. TreeUpdate::tree_id must be TreeId::ROOT.")
	}
	var tim treeIndexMap
	ti := tim.getOrCreateIndex(initialState.TreeID)
	state := State{
		nodes:         map[NodeID]*nodeState{},
		root:          newNodeID(tree.Root, ti),
		data:          tree,
		focus:         newNodeID(initialState.Focus, ti),
		isHostFocused: isHostFocused,
		subtrees:      map[a11y.TreeID]subtreeState{},
		graftParents:  map[a11y.TreeID]NodeID{},
		treeIndexMap:  tim,
	}
	state.update(initialState, isHostFocused, nil)
	t := &Tree{nextState: state.clone()}
	t.state = state
	return t
}

// UpdateAndProcessChanges applies update and reports the changes to handler.
func (t *Tree) UpdateAndProcessChanges(update a11y.TreeUpdate, handler ChangeHandler) {
	changes := newInternalChanges()
	t.nextState.update(update, t.state.isHostFocused, changes)
	t.processChanges(changes, handler)
}

// UpdateHostFocusStateAndProcessChanges sets host focus and reports the changes to handler.
func (t *Tree) UpdateHostFocusStateAndProcessChanges(isHostFocused bool, handler ChangeHandler) {
	changes := newInternalChanges()
	t.nextState.updateHostFocusState(isHostFocused, changes)
	t.processChanges(changes, handler)
}

func optionalNode(n Node, ok bool) *Node {
	if !ok {
		return nil
	}
	return &n
}

func (t *Tree) processChanges(changes *internalChanges, handler ChangeHandler) {
	for id := range changes.addedNodeIDs {
		node := t.nextState.mustNode(id)
		handler.NodeAdded(&node)
	}
	for id := range changes.updatedNodeIDs {
		oldNode := t.state.mustNode(id)
		newNode := t.nextState.mustNode(id)
		handler.NodeUpdated(&oldNode, &newNode)
	}
	oldFocus := optionalNode(t.state.Focus())
	newFocus := optionalNode(t.nextState.Focus())
	if focusIDOf(oldFocus) != focusIDOf(newFocus) {
		if oldFocus != nil {
			id := oldFocus.ID()
			if !changes.updatedNodeIDs.contains(id) && !changes.removedNodeIDs.contains(id) {
				if n, ok := t.nextState.NodeByID(id); ok {
					handler.NodeUpdated(oldFocus, &n)
				}
			}
		}
		if newFocus != nil {
			id := newFocus.ID()
			if !changes.addedNodeIDs.contains(id) && !changes.updatedNodeIDs.contains(id) {
				if n, ok := t.state.NodeByID(id); ok {
					handler.NodeUpdated(&n, newFocus)
				}
			}
		}
		handler.FocusMoved(oldFocus, newFocus)
	}
	for id := range changes.removedNodeIDs {
		node := t.state.mustNode(id)
		handler.NodeRemoved(&node)
	}
	for id := range changes.addedNodeIDs {
		t.state.nodes[id] = t.nextState.nodes[id].clone()
	}
	for id := range changes.updatedNodeIDs {
		next := t.nextState.nodes[id]
		cur := t.state.nodes[id]
		cur.parentAndIndex = next.parentAndIndex
		cur.data = next.data.Clone()
	}
	for id := range changes.removedNodeIDs {
		delete(t.state.nodes, id)
	}
	if t.state.data != t.nextState.data {
		t.state.data = t.nextState.data
	}
	t.state.root = t.nextState.root
	t.state.focus = t.nextState.focus
	t.state.isHostFocused = t.nextState.isHostFocused
	t.state.subtrees = cloneMap(t.nextState.subtrees)
	t.state.graftParents = cloneMap(t.nextState.graftParents)
	t.state.treeIndexMap = t.nextState.treeIndexMap.clone()
}

func focusIDOf(n *Node) gpuiutil.Option[NodeID] {
	if n == nil {
		return gpuiutil.None[NodeID]()
	}
	return gpuiutil.Some(n.ID())
}

// State returns the current state of the tree.
func (t *Tree) State() *State {
	return &t.state
}

func shortNodeList[T any](m map[NodeID]T) string {
	var b strings.Builder
	b.WriteString("[")
	i := 0
	for id := range m {
		if i == 10 {
			b.WriteString(" ...")
			break
		}
		if i != 0 {
			b.WriteString(", ")
		}
		local, _ := id.toComponents()
		fmt.Fprintf(&b, "%v", local)
		i++
	}
	b.WriteString("]")
	return b.String()
}
