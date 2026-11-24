package gpui

import (
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

const rootNodeID a11y.NodeID = 0

type a11yActionListener func(data a11y.ActionData, window *Window, cx *App)

type a11yActionEntry struct {
	action   a11y.Action
	listener a11yActionListener
}

type windowA11y struct {
	forceDisabled        bool
	activeFlag           *atomic.Bool
	activeThisFrame      bool
	nodes                a11yNodeBuilder
	focusIDs             map[a11y.NodeID]FocusID
	nodeBounds           map[a11y.NodeID]Bounds[Pixels]
	actionListeners      map[a11y.NodeID][]a11yActionEntry
	windowTitle          Option[SharedString]
	lastFocusWithoutNode Option[FocusID]
	debug                a11yDebug
	viewTypeNames        map[EntityID]string
}

func newWindowA11y(activeFlag *atomic.Bool, forceDisabled bool, windowTitle Option[SharedString]) windowA11y {
	return windowA11y{
		forceDisabled:   forceDisabled,
		activeFlag:      activeFlag,
		nodes:           newA11yNodeBuilder(),
		focusIDs:        make(map[a11y.NodeID]FocusID),
		nodeBounds:      make(map[a11y.NodeID]Bounds[Pixels]),
		actionListeners: make(map[a11y.NodeID][]a11yActionEntry),
		windowTitle:     windowTitle,
		viewTypeNames:   make(map[EntityID]string),
	}
}

func (a *windowA11y) noteFocusWithoutNode(focusID FocusID, reason string) {
	if last, ok := a.lastFocusWithoutNode.Get(); !ok || last != focusID {
		a.lastFocusWithoutNode = Some(focusID)
		slog.Info(fmt.Sprintf("a11y: focused element (FocusId(%d)) has no accessibility node (%s); assistive technology will announce the whole window instead. Give it both an `.id(...)` and a `.role(...)` to expose it.", uint64(focusID), reason))
	}
}

func (a *windowA11y) setWindowTitle(title SharedString) {
	a.windowTitle = Some(title)
}

func (a *windowA11y) syncActiveFlag() {
	a.activeThisFrame = a.isEnabled() && a.activeFlag.Load()
}

func (a *windowA11y) isEnabled() bool {
	return !a.forceDisabled
}

func (a *windowA11y) isActive() bool {
	return a.activeThisFrame
}

func (a *windowA11y) setFocusable(nodeID a11y.NodeID, focusID FocusID) {
	a.focusIDs[nodeID] = focusID
}

func (a *windowA11y) setFocus(nodeID a11y.NodeID) {
	if _, ok := a.focusIDs[nodeID]; !ok {
		if gpuiutil.DebugAssertions {
			panic("set_focus called for a node that was not registered with set_focusable")
		} else {
			slog.Warn(fmt.Sprintf("a11y: set_focus called for a node that was not registered with set_focusable (%v)", nodeID))
		}
	}
	if a.nodes.hasNode(nodeID) {
		a.lastFocusWithoutNode = None[FocusID]()
		a.nodes.setFocus(nodeID)
	} else if focusID, ok := a.focusIDs[nodeID]; ok {
		a.noteFocusWithoutNode(focusID, "it has an id but no role")
	}
}

func (a *windowA11y) setActiveDescendant(nodeID a11y.NodeID) {
	if a.nodes.nodeIsFocused(nodeID) {
		if gpuiutil.DebugAssertions {
			panic("set_active_descendant called on the focused node")
		} else {
			slog.Warn(fmt.Sprintf("a11y: set_active_descendant called on the focused node (%v)", nodeID))
		}
		return
	}
	if a.nodes.hasNode(nodeID) && a.nodes.focusIsAncestorOfCurrent() {
		a.nodes.setActiveDescendant(nodeID)
	}
}

func (a *windowA11y) beginFrame() {
	clear(a.focusIDs)
	clear(a.nodeBounds)
	clear(a.actionListeners)
	var title *SharedString
	if t, ok := a.windowTitle.Get(); ok {
		title = &t
	}
	a.nodes.beginFrame(title)
}

func (a *windowA11y) endFrame(frame frameDebugInfo) a11y.TreeUpdate {
	update := a.nodes.finalize()
	a.debug.capture(&update, a.nodes.focus, a.nodes.activeDescendant, a.windowTitle, frame)
	if gpuiutil.DebugAssertions {
		a.debug.captureNodeInfo(a.nodes.nodeInfo)
	}
	return update
}

func (a *windowA11y) debugTreeJSON() (string, bool) {
	return a.debug.toJSON()
}

// A11ySubtreeBuilder adds synthetic accessibility nodes below an element; see
// ElementA11ySyntheticChildren.
type A11ySubtreeBuilder struct {
	parentID a11y.NodeID
	nodes    *a11yNodeBuilder
	creator  nodeCreator
}

func newA11ySubtreeBuilder(parentID a11y.NodeID, nodes *a11yNodeBuilder) A11ySubtreeBuilder {
	return A11ySubtreeBuilder{parentID: parentID, nodes: nodes}
}

func (b A11ySubtreeBuilder) withCreator(creator nodeCreator) A11ySubtreeBuilder {
	b.creator = creator
	return b
}

// SyntheticNodeID derives a NodeID for a synthetic child from key and the parent's ID. Keys must
// be unique within one ElementA11ySyntheticChildren call.
func (b *A11ySubtreeBuilder) SyntheticNodeID[K any](key K) a11y.NodeID {
	var h FxHasher
	h.WriteU64(uint64(b.parentID))
	h.WriteU64(Hash(key))
	return a11y.NodeID(h.Finish())
}

// PushChild appends a synthetic leaf node as a child of this element's node. It returns false
// if a node with this id is already in the tree, in which case the node is discarded.
func (b *A11ySubtreeBuilder) PushChild(id a11y.NodeID, node a11y.Node) bool {
	pushed := b.nodes.pushLeaf(id, node)
	if gpuiutil.DebugAssertions && pushed {
		b.nodes.recordNodeInfo(id, nodeDebugInfo{
			synthetic:      true,
			view:           b.creator.view,
			elementID:      b.creator.elementID,
			sourceLocation: b.creator.sourceLocation,
		})
	}
	return pushed
}

// ParentNode returns the parent node for modification.
func (b *A11ySubtreeBuilder) ParentNode() *a11y.Node {
	node, ok := b.nodes.currentNodeMut()
	if !ok {
		panic("A11ySubtreeBuilder exists only while its element's node is on the stack")
	}
	return node
}

type a11yNodeBuilder struct {
	idsStack         []a11y.NodeID
	nodesStack       []a11y.Node
	allNodes         []a11y.NodeEntry
	seenIDs          map[a11y.NodeID]struct{}
	focus            Option[a11y.NodeID]
	activeDescendant Option[a11y.NodeID]
	nodeInfo         map[a11y.NodeID]nodeDebugInfo
}

func newA11yNodeBuilder() a11yNodeBuilder {
	return a11yNodeBuilder{
		seenIDs:  make(map[a11y.NodeID]struct{}),
		nodeInfo: make(map[a11y.NodeID]nodeDebugInfo),
	}
}

func (b *a11yNodeBuilder) recordNodeInfo(id a11y.NodeID, info nodeDebugInfo) {
	b.nodeInfo[id] = info
}

func (b *a11yNodeBuilder) canPush(id a11y.NodeID) bool {
	if gpuiutil.DebugAssertions && len(b.idsStack) == 0 {
		panic("node pushed before push_root")
	}
	if _, seen := b.seenIDs[id]; seen {
		if gpuiutil.DebugAssertions {
			panic(fmt.Sprintf("Duplicate a11y node id: %v. In a release build, this node would be silently discarded from the a11y tree.", id))
		}
		return false
	}
	b.seenIDs[id] = struct{}{}
	return true
}

func (b *a11yNodeBuilder) push(id a11y.NodeID, node a11y.Node) bool {
	if !b.canPush(id) {
		return false
	}
	if len(b.nodesStack) > 0 {
		b.nodesStack[len(b.nodesStack)-1].PushChild(id)
	}
	b.idsStack = append(b.idsStack, id)
	b.nodesStack = append(b.nodesStack, node)
	return true
}

func (b *a11yNodeBuilder) pushLeaf(id a11y.NodeID, node a11y.Node) bool {
	if !b.canPush(id) {
		return false
	}
	if len(b.nodesStack) > 0 {
		b.nodesStack[len(b.nodesStack)-1].PushChild(id)
	}
	b.allNodes = append(b.allNodes, a11y.NodeEntry{ID: id, Node: node})
	return true
}

func (b *a11yNodeBuilder) currentNodeMut() (*a11y.Node, bool) {
	if len(b.nodesStack) == 0 {
		return nil, false
	}
	return &b.nodesStack[len(b.nodesStack)-1], true
}

func (b *a11yNodeBuilder) pop() {
	if gpuiutil.DebugAssertions && len(b.idsStack) <= 1 {
		panic("pop would remove the root node")
	}
	b.popAny()
}

func (b *a11yNodeBuilder) popAny() {
	if len(b.idsStack) == 0 || len(b.nodesStack) == 0 {
		if len(b.idsStack) > 0 {
			b.idsStack = b.idsStack[:len(b.idsStack)-1]
		}
		if len(b.nodesStack) > 0 {
			b.nodesStack = b.nodesStack[:len(b.nodesStack)-1]
		}
		return
	}
	id := b.idsStack[len(b.idsStack)-1]
	node := b.nodesStack[len(b.nodesStack)-1]
	b.idsStack = b.idsStack[:len(b.idsStack)-1]
	b.nodesStack = b.nodesStack[:len(b.nodesStack)-1]
	b.allNodes = append(b.allNodes, a11y.NodeEntry{ID: id, Node: node})
}

func (b *a11yNodeBuilder) beginFrame(windowTitle *SharedString) {
	b.allNodes = b.allNodes[:0]
	b.idsStack = b.idsStack[:0]
	b.nodesStack = b.nodesStack[:0]
	if b.seenIDs == nil {
		b.seenIDs = make(map[a11y.NodeID]struct{})
	}
	clear(b.seenIDs)
	if b.nodeInfo == nil {
		b.nodeInfo = make(map[a11y.NodeID]nodeDebugInfo)
	}
	if gpuiutil.DebugAssertions {
		clear(b.nodeInfo)
	}
	rootNode := a11y.NewNode(a11y.RoleWindow)
	if windowTitle != nil {
		rootNode.SetLabel(string(*windowTitle))
	}
	b.idsStack = append(b.idsStack, rootNodeID)
	b.nodesStack = append(b.nodesStack, rootNode)
	b.focus = None[a11y.NodeID]()
	b.activeDescendant = None[a11y.NodeID]()
}

func (b *a11yNodeBuilder) hasNode(id a11y.NodeID) bool {
	if id == rootNodeID {
		return true
	}
	_, ok := b.seenIDs[id]
	return ok
}

func (b *a11yNodeBuilder) nodeIsFocused(id a11y.NodeID) bool {
	focus, ok := b.focus.Get()
	return ok && focus == id
}

func (b *a11yNodeBuilder) focusIsAncestorOfCurrent() bool {
	focus, ok := b.focus.Get()
	if !ok {
		return false
	}
	ancestorCount := max(len(b.idsStack)-1, 0)
	return slices.Contains(b.idsStack[:ancestorCount], focus)
}

func (b *a11yNodeBuilder) setActiveDescendant(id a11y.NodeID) {
	if existing, ok := b.activeDescendant.Get(); ok && existing != id {
		if gpuiutil.DebugAssertions {
			panic("active descendant claimed by multiple nodes in one frame")
		} else {
			slog.Warn(fmt.Sprintf("a11y: multiple nodes claimed the active descendant this frame; using last-wins (%v)", id))
		}
	}
	b.activeDescendant = Some(id)
}

func (b *a11yNodeBuilder) setFocus(id a11y.NodeID) {
	if b.focus.IsSome() {
		if gpuiutil.DebugAssertions {
			panic("set_focus called more than once in a single frame")
		} else {
			slog.Warn(fmt.Sprintf("a11y: set_focus called more than once in a single frame; using last-wins (%v)", id))
		}
	}
	b.focus = Some(id)
}

func (b *a11yNodeBuilder) finalize() a11y.TreeUpdate {
	if gpuiutil.DebugAssertions {
		if len(b.idsStack) != 1 {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", len(b.idsStack), 1))
		}
		if b.idsStack[0] != rootNodeID {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %v\n right: %v", b.idsStack[0], rootNodeID))
		}
	}
	if len(b.idsStack) != 1 {
		slog.Error(fmt.Sprintf("a11y: Stack imbalance at end of frame: expected 1 (root), got %d. Some elements may have pushed without popping.", len(b.idsStack)))
	}
	for len(b.idsStack) > 0 {
		b.popAny()
	}

	var focus a11y.NodeID
	if id, ok := b.activeDescendant.Get(); ok {
		if b.hasNode(id) {
			focus = id
		} else if gpuiutil.DebugAssertions {
			panic(fmt.Sprintf("active_descendant set to %v, which is not in the tree", id))
		} else {
			slog.Warn(fmt.Sprintf("active_descendant set to %v, which is not in the tree", id))
			focus = b.focus.UnwrapOr(rootNodeID)
		}
	} else {
		focus = b.focus.UnwrapOr(rootNodeID)
	}

	nodes := b.allNodes
	b.allNodes = nil
	update := a11y.TreeUpdate{
		Nodes:  nodes,
		Tree:   Some(a11y.NewTree(rootNodeID)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  focus,
	}
	return repairTreeUpdate(update)
}

func repairTreeUpdate(update a11y.TreeUpdate) a11y.TreeUpdate {
	nodeIDs := make(map[a11y.NodeID]struct{}, len(update.Nodes))
	for _, entry := range update.Nodes {
		nodeIDs[entry.ID] = struct{}{}
	}
	contains := func(id a11y.NodeID) bool {
		_, ok := nodeIDs[id]
		return ok
	}

	if !contains(update.Focus) {
		slog.Error(fmt.Sprintf("a11y: Focused node %v is not in the tree (%d nodes). Falling back to root. This is a bug in the a11y tree builder.", update.Focus, len(update.Nodes)))
		update.Focus = rootNodeID
	}

	for i := range update.Nodes {
		node := &update.Nodes[i].Node
		children := node.Children()
		invalidCount := 0
		for _, child := range children {
			if !contains(child) {
				invalidCount++
			}
		}
		if invalidCount > 0 {
			slog.Error(fmt.Sprintf("a11y: Node %v references %d children not present in the tree. Stripping invalid child references.", update.Nodes[i].ID, invalidCount))
			valid := make([]a11y.NodeID, 0, len(children)-invalidCount)
			for _, child := range children {
				if contains(child) {
					valid = append(valid, child)
				}
			}
			node.SetChildren(valid)
		}
	}
	return update
}
