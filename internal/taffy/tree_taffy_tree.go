package taffy

import (
	"fmt"
	"iter"
	"slices"
	"unsafe"
)

type keyData struct {
	idx     uint32
	version uint32
}

func keyDataFromFFI(value uint64) keyData {
	return keyData{idx: uint32(value & 0xffff_ffff), version: uint32(value>>32) | 1}
}

func (k keyData) asFFI() uint64 {
	return uint64(k.version)<<32 | uint64(k.idx)
}

func nodeIDFromKey(key keyData) NodeID {
	return NodeIDFromU64(key.asFFI())
}

func keyFromNodeID(id NodeID) keyData {
	return keyDataFromFFI(U64FromNodeID(id))
}

type slotMapSlot[V any] struct {
	value    V
	nextFree uint32
	version  uint32
}

type slotMap[V any] struct {
	slots    []slotMapSlot[V]
	freeHead uint32
	numElems uint32
}

func newSlotMapWithCapacity[V any](capacity int) slotMap[V] {
	slots := make([]slotMapSlot[V], 1, capacity+1)
	return slotMap[V]{slots: slots, freeHead: 1}
}

func (m *slotMap[V]) len() int {
	return int(m.numElems)
}

func (m *slotMap[V]) capacity() int {
	return cap(m.slots) - 1
}

func (m *slotMap[V]) insert(value V) keyData {
	if int(m.freeHead) < len(m.slots) {
		slot := &m.slots[m.freeHead]
		occupiedVersion := slot.version | 1
		kd := keyData{idx: m.freeHead, version: occupiedVersion}
		m.freeHead = slot.nextFree
		slot.value = value
		slot.version = occupiedVersion
		m.numElems++
		return kd
	}
	if len(m.slots) >= 0xffff_ffff {
		panic("SlotMap is full")
	}
	kd := keyData{idx: uint32(len(m.slots)), version: 1}
	m.slots = append(m.slots, slotMapSlot[V]{value: value, version: 1})
	m.freeHead = kd.idx + 1
	m.numElems++
	return kd
}

func (m *slotMap[V]) containsKey(key keyData) bool {
	return int(key.idx) < len(m.slots) && m.slots[key.idx].version == key.version
}

func (m *slotMap[V]) get(key keyData) (*V, bool) {
	if !m.containsKey(key) {
		return nil, false
	}
	return &m.slots[key.idx].value, true
}

func (m *slotMap[V]) index(key keyData) *V {
	if v, ok := m.get(key); ok {
		return v
	}
	panic("invalid SlotMap key used")
}

func (m *slotMap[V]) removeFromSlot(idx uint32) V {
	slot := &m.slots[idx]
	value := slot.value
	var zero V
	slot.value = zero
	slot.nextFree = m.freeHead
	m.freeHead = idx
	m.numElems--
	slot.version++
	return value
}

func (m *slotMap[V]) remove(key keyData) (V, bool) {
	if m.containsKey(key) {
		return m.removeFromSlot(key.idx), true
	}
	var zero V
	return zero, false
}

func (m *slotMap[V]) clear() {
	for idx := 1; idx < len(m.slots) && m.numElems > 0; idx++ {
		if m.slots[idx].version%2 == 1 {
			m.removeFromSlot(uint32(idx))
		}
	}
}

type sparseSecondarySlot[V any] struct {
	version uint32
	value   *V
}

type sparseSecondaryMap[V any] struct {
	slots map[uint32]*sparseSecondarySlot[V]
}

func newSparseSecondaryMapWithCapacity[V any](capacity int) sparseSecondaryMap[V] {
	return sparseSecondaryMap[V]{slots: make(map[uint32]*sparseSecondarySlot[V], capacity)}
}

func isOlderVersion(a, b uint32) bool {
	return a-b >= 1<<31
}

func (m *sparseSecondaryMap[V]) insert(key keyData, value V) {
	if key.idx == 0xffff_ffff {
		return
	}
	if slot, ok := m.slots[key.idx]; ok {
		if slot.version == key.version {
			*slot.value = value
			return
		}
		if isOlderVersion(key.version, slot.version) {
			return
		}
		m.slots[key.idx] = &sparseSecondarySlot[V]{version: key.version, value: &value}
		return
	}
	m.slots[key.idx] = &sparseSecondarySlot[V]{version: key.version, value: &value}
}

func (m *sparseSecondaryMap[V]) get(key keyData) *V {
	if slot, ok := m.slots[key.idx]; ok && slot.version == key.version {
		return slot.value
	}
	return nil
}

func (m *sparseSecondaryMap[V]) remove(key keyData) {
	if slot, ok := m.slots[key.idx]; ok && slot.version == key.version {
		delete(m.slots, key.idx)
	}
}

func (m *sparseSecondaryMap[V]) clear() {
	clear(m.slots)
}

func (m *sparseSecondaryMap[V]) len() int {
	return len(m.slots)
}

func (m *sparseSecondaryMap[V]) getDisjointMut(keys []keyData) ([]*V, bool) {
	ptrs := make([]*V, 0, len(keys))
	i := 0
	for i < len(keys) {
		kd := keys[i]
		slot, ok := m.slots[kd.idx]
		if !ok || slot.version != kd.version {
			break
		}
		ptrs = append(ptrs, slot.value)
		slot.version ^= 1
		i++
	}
	for _, k := range keys[:i] {
		m.slots[k.idx].version ^= 1
	}
	if i == len(keys) {
		return ptrs, true
	}
	return nil, false
}

// TaffyErrorKind is the variant of a TaffyError.
type TaffyErrorKind uint8

const (
	// TaffyErrorKindChildIndexOutOfBounds: the parent node does not have a child at childIndex.
	TaffyErrorKindChildIndexOutOfBounds TaffyErrorKind = iota
	// TaffyErrorKindInvalidParentNode: the parent node was not found in the TaffyTree.
	TaffyErrorKindInvalidParentNode
	// TaffyErrorKindInvalidChildNode: the child node was not found in the TaffyTree.
	TaffyErrorKindInvalidChildNode
	// TaffyErrorKindInvalidInputNode: the supplied node was not found in the TaffyTree.
	TaffyErrorKindInvalidInputNode
)

// TaffyError is an error that occurs while trying to access or modify a node's children by index.
type TaffyError struct {
	kind       TaffyErrorKind
	node       NodeID
	childIndex int
	childCount int
}

// TaffyErrorChildIndexOutOfBounds builds the ChildIndexOutOfBounds variant.
func TaffyErrorChildIndexOutOfBounds(parent NodeID, childIndex, childCount int) TaffyError {
	return TaffyError{kind: TaffyErrorKindChildIndexOutOfBounds, node: parent, childIndex: childIndex, childCount: childCount}
}

// TaffyErrorInvalidParentNode builds the InvalidParentNode variant.
func TaffyErrorInvalidParentNode(parent NodeID) TaffyError {
	return TaffyError{kind: TaffyErrorKindInvalidParentNode, node: parent}
}

// TaffyErrorInvalidChildNode builds the InvalidChildNode variant.
func TaffyErrorInvalidChildNode(child NodeID) TaffyError {
	return TaffyError{kind: TaffyErrorKindInvalidChildNode, node: child}
}

// TaffyErrorInvalidInputNode builds the InvalidInputNode variant.
func TaffyErrorInvalidInputNode(node NodeID) TaffyError {
	return TaffyError{kind: TaffyErrorKindInvalidInputNode, node: node}
}

// Kind returns the variant of the error.
func (e TaffyError) Kind() TaffyErrorKind {
	return e.kind
}

// Node returns the parent, child or input node the error refers to.
func (e TaffyError) Node() NodeID {
	return e.node
}

// ChildIndex returns the index that was looked up (ChildIndexOutOfBounds).
func (e TaffyError) ChildIndex() int {
	return e.childIndex
}

// ChildCount returns the number of children the parent has (ChildIndexOutOfBounds).
func (e TaffyError) ChildCount() int {
	return e.childCount
}

// Error formats the error as upstream's Display impl.
func (e TaffyError) Error() string {
	switch e.kind {
	case TaffyErrorKindChildIndexOutOfBounds:
		return fmt.Sprintf("Index (is %d) should be < child_count (%d) for parent node %v", e.childIndex, e.childCount, e.node)
	case TaffyErrorKindInvalidParentNode:
		return fmt.Sprintf("Parent Node %v is not in the TaffyTree instance", e.node)
	case TaffyErrorKindInvalidChildNode:
		return fmt.Sprintf("Child Node %v is not in the TaffyTree instance", e.node)
	default:
		return fmt.Sprintf("Supplied Node %v is not in the TaffyTree instance", e.node)
	}
}

type taffyConfig struct {
	useRounding bool
}

func taffyConfigDefault() taffyConfig {
	return taffyConfig{useRounding: true}
}

type nodeData struct {
	style              Style[defaultCheapStr]
	unroundedLayout    Layout
	finalLayout        Layout
	hasContext         bool
	hoistedChildren    []NodeID
	cache              Cache
	detailedLayoutInfo DetailedLayoutInfo[defaultCheapStr]
}

func newNodeData(style Style[defaultCheapStr]) nodeData {
	return nodeData{
		style:              style,
		cache:              NewCache(),
		unroundedLayout:    NewLayout(),
		finalLayout:        NewLayout(),
		hasContext:         false,
		hoistedChildren:    newConstChildrenVec[NodeID](),
		detailedLayoutInfo: DetailedLayoutInfoNone[defaultCheapStr](),
	}
}

func (n *nodeData) markDirty() ClearState {
	return n.cache.Clear()
}

// TaffyTree is an entire tree of UI nodes, the entry point to taffy's high-level API.
type TaffyTree[NodeContext any] struct {
	nodes           slotMap[nodeData]
	nodeContextData sparseSecondaryMap[NodeContext]
	children        slotMap[[]NodeID]
	parents         slotMap[Option[NodeID]]
	config          taffyConfig
}

// TaffyTreeChildIter iterates over the children of a node (upstream's slice iterator wrapper).
type TaffyTreeChildIter = iter.Seq[NodeID]

func childIter(children []NodeID) TaffyTreeChildIter {
	return func(yield func(NodeID) bool) {
		for _, child := range children {
			if !yield(child) {
				return
			}
		}
	}
}

// ChildIds iterates over the children of parentNodeID.
func (t *TaffyTree[NodeContext]) ChildIds(parentNodeID NodeID) iter.Seq[NodeID] {
	return childIter(*t.children.index(keyFromNodeID(parentNodeID)))
}

// ChildCount returns the number of children of parentNodeID.
func (t *TaffyTree[NodeContext]) ChildCount(parentNodeID NodeID) int {
	return len(*t.children.index(keyFromNodeID(parentNodeID)))
}

// GetChildID returns the child of parentNodeID at index id.
func (t *TaffyTree[NodeContext]) GetChildID(parentNodeID NodeID, id int) NodeID {
	return (*t.children.index(keyFromNodeID(parentNodeID)))[id]
}

// TraverseTree marks TaffyTree as a TraverseTree.
func (t *TaffyTree[NodeContext]) TraverseTree() {}

// CacheGet returns the cached layout of nodeID for input.
func (t *TaffyTree[NodeContext]) CacheGet(nodeID NodeID, input *LayoutInput) Option[LayoutOutput] {
	return t.nodes.index(keyFromNodeID(nodeID)).cache.Get(input)
}

// CacheStore stores layoutOutput in the cache of nodeID for input.
func (t *TaffyTree[NodeContext]) CacheStore(nodeID NodeID, input *LayoutInput, layoutOutput LayoutOutput) {
	t.nodes.index(keyFromNodeID(nodeID)).cache.Store(input, layoutOutput)
}

// CacheClear clears the cache of nodeID.
func (t *TaffyTree[NodeContext]) CacheClear(nodeID NodeID) {
	t.nodes.index(keyFromNodeID(nodeID)).cache.Clear()
}

// GetDebugLabel returns the label PrintTree shows for nodeID.
func (t *TaffyTree[NodeContext]) GetDebugLabel(nodeID NodeID) string {
	node := t.nodes.index(keyFromNodeID(nodeID))
	display := node.style.Display
	numChildren := t.ChildCount(nodeID)

	switch {
	case display == DisplayNone:
		return "NONE"
	case numChildren == 0:
		return "LEAF"
	case display == DisplayBlock:
		return "BLOCK"
	case display == DisplayFlowRoot:
		return "FLOW-ROOT"
	case display == DisplayFlex:
		switch node.style.FlexDirection {
		case FlexDirectionRow, FlexDirectionRowReverse:
			return "FLEX ROW"
		default:
			return "FLEX COL"
		}
	default:
		return "GRID"
	}
}

// GetFinalLayout returns the final layout of nodeID: rounded when rounding is enabled, else unrounded.
func (t *TaffyTree[NodeContext]) GetFinalLayout(nodeID NodeID) Layout {
	if t.config.useRounding {
		return t.nodes.index(keyFromNodeID(nodeID)).finalLayout
	}
	return t.nodes.index(keyFromNodeID(nodeID)).unroundedLayout
}

// MeasureFunction computes the layout of a leaf node of a TaffyTree; nodeContext is nil when the node has no context.
type MeasureFunction[NodeContext any] func(inputs LayoutInput, nodeID NodeID, nodeContext *NodeContext, style *Style[defaultCheapStr]) LayoutOutput

type taffyView[NodeContext any] struct {
	taffy           *TaffyTree[NodeContext]
	measureFunction MeasureFunction[NodeContext]
}

func (v *taffyView[NodeContext]) node(nodeID NodeID) *nodeData {
	return v.taffy.nodes.index(keyFromNodeID(nodeID))
}

func (v *taffyView[NodeContext]) computeChildLayout(nodeID NodeID, inputs LayoutInput, blockCtx *BlockContext) LayoutOutput {
	if inputs.RunMode == RunModePerformHiddenLayout {
		return ComputeHiddenLayout(v, nodeID)
	}

	return ComputeCachedLayout(v, nodeID, inputs, func(tree *taffyView[NodeContext], nodeID NodeID, inputs LayoutInput) LayoutOutput {
		displayMode := tree.node(nodeID).style.Display
		hasChildren := tree.ChildCount(nodeID) > 0

		var output LayoutOutput
		switch {
		case displayMode == DisplayNone:
			output = ComputeHiddenLayout(tree, nodeID)
		case displayMode == DisplayBlock && hasChildren:
			output = ComputeBlockLayout(tree, nodeID, inputs, blockCtx)
		case displayMode == DisplayFlowRoot && hasChildren:
			output = ComputeBlockLayout(tree, nodeID, inputs, nil)
		case displayMode == DisplayFlex && hasChildren:
			output = ComputeFlexboxLayout(tree, nodeID, inputs)
		case displayMode == DisplayGrid && hasChildren:
			output = ComputeGridLayout[defaultCheapStr](tree, nodeID, inputs)
		default:
			nodeKey := keyFromNodeID(nodeID)
			node := tree.taffy.nodes.index(nodeKey)
			var nodeContext *NodeContext
			if node.hasContext {
				nodeContext = tree.taffy.nodeContextData.get(nodeKey)
			}
			output = tree.measureFunction(inputs, nodeID, nodeContext, &node.style)
		}

		if inputs.RunMode == RunModePerformLayout {
			ComputeOofLayout[defaultCheapStr](tree, nodeID, &output)
		}

		return output
	})
}

func (v *taffyView[NodeContext]) ChildIds(parentNodeID NodeID) iter.Seq[NodeID] {
	return v.taffy.ChildIds(parentNodeID)
}

func (v *taffyView[NodeContext]) ChildCount(parentNodeID NodeID) int {
	return v.taffy.ChildCount(parentNodeID)
}

func (v *taffyView[NodeContext]) GetChildID(parentNodeID NodeID, childIndex int) NodeID {
	return v.taffy.GetChildID(parentNodeID, childIndex)
}

func (v *taffyView[NodeContext]) TraverseTree() {}

func (v *taffyView[NodeContext]) GetCoreContainerStyle(nodeID NodeID) CoreStyle {
	return v.node(nodeID).style.Ref()
}

func (v *taffyView[NodeContext]) SetUnroundedLayout(nodeID NodeID, layout *Layout) {
	v.node(nodeID).unroundedLayout = *layout
}

func (v *taffyView[NodeContext]) ResolveCalcValue(val unsafe.Pointer, basis float32) float32 {
	_ = val
	_ = basis
	return 0.0
}

func (v *taffyView[NodeContext]) ComputeChildLayout(nodeID NodeID, inputs LayoutInput) LayoutOutput {
	return v.computeChildLayout(nodeID, inputs, nil)
}

func (v *taffyView[NodeContext]) GetOofItemStyle(nodeID NodeID) OofItemStyle[defaultCheapStr] {
	return v.node(nodeID).style.Ref()
}

func (v *taffyView[NodeContext]) ClearHoistedChildren(nodeID NodeID) {
	node := v.node(nodeID)
	node.hoistedChildren = node.hoistedChildren[:0]
}

func (v *taffyView[NodeContext]) AddHoistedChildren(nodeID NodeID, hoisted []NodeID) {
	node := v.node(nodeID)
	node.hoistedChildren = append(node.hoistedChildren, hoisted...)
}

func (v *taffyView[NodeContext]) GetDetailedLayoutInfo(nodeID NodeID) DetailedLayoutInfo[defaultCheapStr] {
	return v.node(nodeID).detailedLayoutInfo
}

func (v *taffyView[NodeContext]) CacheGet(nodeID NodeID, input *LayoutInput) Option[LayoutOutput] {
	return v.node(nodeID).cache.Get(input)
}

func (v *taffyView[NodeContext]) CacheStore(nodeID NodeID, input *LayoutInput, layoutOutput LayoutOutput) {
	v.node(nodeID).cache.Store(input, layoutOutput)
}

func (v *taffyView[NodeContext]) CacheClear(nodeID NodeID) {
	v.node(nodeID).cache.Clear()
}

func (v *taffyView[NodeContext]) GetBlockContainerStyle(nodeID NodeID) BlockContainerStyle {
	return v.node(nodeID).style.Ref()
}

func (v *taffyView[NodeContext]) GetBlockChildStyle(childNodeID NodeID) BlockItemStyle {
	return v.node(childNodeID).style.Ref()
}

func (v *taffyView[NodeContext]) ComputeBlockChildLayout(nodeID NodeID, inputs LayoutInput, blockCtx *BlockContext) LayoutOutput {
	return v.computeChildLayout(nodeID, inputs, blockCtx)
}

func (v *taffyView[NodeContext]) GetFlexboxContainerStyle(nodeID NodeID) FlexboxContainerStyle {
	return v.node(nodeID).style.Ref()
}

func (v *taffyView[NodeContext]) GetFlexboxChildStyle(childNodeID NodeID) FlexboxItemStyle {
	return v.node(childNodeID).style.Ref()
}

func (v *taffyView[NodeContext]) GetGridContainerStyle(nodeID NodeID) GridContainerStyle[defaultCheapStr] {
	return v.node(nodeID).style.Ref()
}

func (v *taffyView[NodeContext]) GetGridChildStyle(childNodeID NodeID) GridItemStyle[defaultCheapStr] {
	return v.node(childNodeID).style.Ref()
}

func (v *taffyView[NodeContext]) SetDetailedGridInfo(nodeID NodeID, detailedGridInfo DetailedGridInfo[defaultCheapStr]) {
	v.node(nodeID).detailedLayoutInfo = DetailedLayoutInfoGrid(&detailedGridInfo)
}

func (v *taffyView[NodeContext]) GetUnroundedLayout(nodeID NodeID) Layout {
	return v.node(nodeID).unroundedLayout
}

func (v *taffyView[NodeContext]) SetFinalLayout(nodeID NodeID, layout *Layout) {
	v.node(nodeID).finalLayout = *layout
}

func (v *taffyView[NodeContext]) IsOutOfFlow(nodeID NodeID) bool {
	node := v.node(nodeID)
	return node.style.Position.IsOutOfFlow() && node.style.Display != DisplayNone
}

func (v *taffyView[NodeContext]) HoistedChildCount(nodeID NodeID) int {
	return len(v.node(nodeID).hoistedChildren)
}

func (v *taffyView[NodeContext]) GetHoistedChildID(nodeID NodeID, index int) NodeID {
	return v.node(nodeID).hoistedChildren[index]
}

var (
	_ RootLayoutTree[defaultCheapStr]      = (*taffyView[struct{}])(nil)
	_ LayoutBlockContainer                 = (*taffyView[struct{}])(nil)
	_ LayoutFlexboxContainer               = (*taffyView[struct{}])(nil)
	_ LayoutGridContainer[defaultCheapStr] = (*taffyView[struct{}])(nil)
	_ RoundTree                            = (*taffyView[struct{}])(nil)
	_ PrintTree                            = (*TaffyTree[struct{}])(nil)
	_ CacheTree                            = (*TaffyTree[struct{}])(nil)
)

// NewTaffyTree creates a new TaffyTree with the default capacity of 16 nodes (also upstream's Default).
func NewTaffyTree[NodeContext any]() *TaffyTree[NodeContext] {
	return TaffyTreeWithCapacity[NodeContext](16)
}

// TaffyTreeWithCapacity creates a new TaffyTree that can store capacity nodes before reallocation.
func TaffyTreeWithCapacity[NodeContext any](capacity int) *TaffyTree[NodeContext] {
	return &TaffyTree[NodeContext]{
		nodes:           newSlotMapWithCapacity[nodeData](capacity),
		children:        newSlotMapWithCapacity[[]NodeID](capacity),
		parents:         newSlotMapWithCapacity[Option[NodeID]](capacity),
		nodeContextData: newSparseSecondaryMapWithCapacity[NodeContext](capacity),
		config:          taffyConfigDefault(),
	}
}

// EnableRounding enables rounding of layout values. Rounding is enabled by default.
func (t *TaffyTree[NodeContext]) EnableRounding() {
	t.config.useRounding = true
}

// DisableRounding disables rounding of layout values. Rounding is enabled by default.
func (t *TaffyTree[NodeContext]) DisableRounding() {
	t.config.useRounding = false
}

// NewLeaf creates and adds a new unattached leaf node to the tree and returns its id.
func (t *TaffyTree[NodeContext]) NewLeaf(layout Style[defaultCheapStr]) (NodeID, error) {
	id := t.nodes.insert(newNodeData(layout))
	_ = t.children.insert(newVecWithCapacity[NodeID](0))
	_ = t.parents.insert(none[NodeID]())

	return nodeIDFromKey(id), nil
}

// NewLeafWithContext creates and adds a new unattached leaf node with the supplied context and returns its id.
func (t *TaffyTree[NodeContext]) NewLeafWithContext(layout Style[defaultCheapStr], context NodeContext) (NodeID, error) {
	data := newNodeData(layout)
	data.hasContext = true

	id := t.nodes.insert(data)
	t.nodeContextData.insert(id, context)

	_ = t.children.insert(newVecWithCapacity[NodeID](0))
	_ = t.parents.insert(none[NodeID]())

	return nodeIDFromKey(id), nil
}

// NewWithChildren creates and adds a new node with the given children.
func (t *TaffyTree[NodeContext]) NewWithChildren(layout Style[defaultCheapStr], children []NodeID) (NodeID, error) {
	id := nodeIDFromKey(t.nodes.insert(newNodeData(layout)))

	for _, child := range children {
		*t.parents.index(keyFromNodeID(child)) = some(id)
	}

	_ = t.children.insert(slices.Clone(children))
	_ = t.parents.insert(none[NodeID]())

	return id, nil
}

// Clear drops all nodes in the tree.
func (t *TaffyTree[NodeContext]) Clear() {
	t.nodes.clear()
	t.children.clear()
	t.parents.clear()
	t.nodeContextData.clear()
}

// Remove removes a specific node from the tree and returns its id.
func (t *TaffyTree[NodeContext]) Remove(node NodeID) (NodeID, error) {
	key := keyFromNodeID(node)
	if parent, ok := t.parents.index(key).Get(); ok {
		if children, ok := t.children.get(keyFromNodeID(parent)); ok {
			*children = slices.DeleteFunc(*children, func(f NodeID) bool { return f == node })
		}
		if err := t.MarkDirty(parent); err != nil {
			return NodeID{}, err
		}
	}

	if children, ok := t.children.get(key); ok {
		for _, child := range *children {
			*t.parents.index(keyFromNodeID(child)) = none[NodeID]()
		}
	}

	_, _ = t.children.remove(key)
	_, _ = t.parents.remove(key)
	_, _ = t.nodes.remove(key)
	t.nodeContextData.remove(key)

	return node, nil
}

// SetNodeContext sets the context data of node; a None measure removes it.
func (t *TaffyTree[NodeContext]) SetNodeContext(node NodeID, measure Option[NodeContext]) error {
	key := keyFromNodeID(node)
	if m, ok := measure.Get(); ok {
		t.nodes.index(key).hasContext = true
		t.nodeContextData.insert(key, m)
	} else {
		t.nodes.index(key).hasContext = false
		t.nodeContextData.remove(key)
	}

	return t.MarkDirty(node)
}

// GetNodeContext returns the context data of node, or nil.
func (t *TaffyTree[NodeContext]) GetNodeContext(node NodeID) *NodeContext {
	return t.nodeContextData.get(keyFromNodeID(node))
}

// GetNodeContextMut returns a mutable reference to the context data of node, or nil.
func (t *TaffyTree[NodeContext]) GetNodeContextMut(node NodeID) *NodeContext {
	return t.nodeContextData.get(keyFromNodeID(node))
}

// GetDisjointNodeContextMut returns references to the context data of the nodes; all keys must be valid and disjoint, otherwise ok is false.
func (t *TaffyTree[NodeContext]) GetDisjointNodeContextMut(keys ...NodeID) ([]*NodeContext, bool) {
	kds := make([]keyData, len(keys))
	for i, k := range keys {
		kds[i] = keyFromNodeID(k)
	}
	return t.nodeContextData.getDisjointMut(kds)
}

// AddChild adds child under parent.
func (t *TaffyTree[NodeContext]) AddChild(parent, child NodeID) error {
	parentKey := keyFromNodeID(parent)
	childKey := keyFromNodeID(child)
	*t.parents.index(childKey) = some(parent)
	children := t.children.index(parentKey)
	*children = append(*children, child)
	return t.MarkDirty(parent)
}

// InsertChildAtIndex inserts child at childIndex under parent, shifting the children after it to the right.
func (t *TaffyTree[NodeContext]) InsertChildAtIndex(parent NodeID, childIndex int, child NodeID) error {
	parentKey := keyFromNodeID(parent)

	childCount := len(*t.children.index(parentKey))
	if childIndex > childCount {
		return TaffyErrorChildIndexOutOfBounds(parent, childIndex, childCount)
	}

	*t.parents.index(keyFromNodeID(child)) = some(parent)
	children := t.children.index(parentKey)
	*children = slices.Insert(*children, childIndex, child)
	return t.MarkDirty(parent)
}

// SetChildren directly sets the children of parent.
func (t *TaffyTree[NodeContext]) SetChildren(parent NodeID, children []NodeID) error {
	parentKey := keyFromNodeID(parent)

	for _, child := range *t.children.index(parentKey) {
		*t.parents.index(keyFromNodeID(child)) = none[NodeID]()
	}

	for _, child := range children {
		if previousParent, ok := t.parents.index(keyFromNodeID(child)).Get(); ok {
			if _, err := t.RemoveChild(previousParent, child); err != nil {
				panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
			}
		}
		*t.parents.index(keyFromNodeID(child)) = some(parent)
	}

	parentChildren := t.children.index(parentKey)
	*parentChildren = (*parentChildren)[:0]
	*parentChildren = append(*parentChildren, children...)

	return t.MarkDirty(parent)
}

// RemoveChild detaches child from parent; the child stays in the tree.
func (t *TaffyTree[NodeContext]) RemoveChild(parent, child NodeID) (NodeID, error) {
	index := slices.Index(*t.children.index(keyFromNodeID(parent)), child)
	if index < 0 {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return t.RemoveChildAtIndex(parent, index)
}

// RemoveChildAtIndex detaches the child at childIndex from parent; the child stays in the tree.
func (t *TaffyTree[NodeContext]) RemoveChildAtIndex(parent NodeID, childIndex int) (NodeID, error) {
	parentKey := keyFromNodeID(parent)
	childCount := len(*t.children.index(parentKey))
	if childIndex >= childCount {
		return NodeID{}, TaffyErrorChildIndexOutOfBounds(parent, childIndex, childCount)
	}

	children := t.children.index(parentKey)
	child := (*children)[childIndex]
	*children = slices.Delete(*children, childIndex, childIndex+1)
	*t.parents.index(keyFromNodeID(child)) = none[NodeID]()

	if err := t.MarkDirty(parent); err != nil {
		return NodeID{}, err
	}
	return child, nil
}

// RemoveChildrenRange detaches the children in the half-open range [start, end) from parent; it panics on an invalid range.
func (t *TaffyTree[NodeContext]) RemoveChildrenRange(parent NodeID, start, end int) error {
	parentKey := keyFromNodeID(parent)
	children := t.children.index(parentKey)
	if start > end {
		panic(fmt.Sprintf("slice index starts at %d but ends at %d", start, end))
	}
	if end > len(*children) {
		panic(fmt.Sprintf("range end index %d out of range for slice of length %d", end, len(*children)))
	}
	for _, child := range (*children)[start:end] {
		*t.parents.index(keyFromNodeID(child)) = none[NodeID]()
	}
	*children = slices.Delete(*children, start, end)

	return t.MarkDirty(parent)
}

// ReplaceChildAtIndex replaces the child at childIndex of parent with newChild and returns the old child, which stays in the tree.
func (t *TaffyTree[NodeContext]) ReplaceChildAtIndex(parent NodeID, childIndex int, newChild NodeID) (NodeID, error) {
	parentKey := keyFromNodeID(parent)

	childCount := len(*t.children.index(parentKey))
	if childIndex >= childCount {
		return NodeID{}, TaffyErrorChildIndexOutOfBounds(parent, childIndex, childCount)
	}

	*t.parents.index(keyFromNodeID(newChild)) = some(parent)
	children := t.children.index(parentKey)
	oldChild := (*children)[childIndex]
	(*children)[childIndex] = newChild
	*t.parents.index(keyFromNodeID(oldChild)) = none[NodeID]()

	if err := t.MarkDirty(parent); err != nil {
		return NodeID{}, err
	}
	return oldChild, nil
}

// ChildAtIndex returns the child of parent at childIndex.
func (t *TaffyTree[NodeContext]) ChildAtIndex(parent NodeID, childIndex int) (NodeID, error) {
	parentKey := keyFromNodeID(parent)
	children := *t.children.index(parentKey)
	childCount := len(children)
	if childIndex >= childCount {
		return NodeID{}, TaffyErrorChildIndexOutOfBounds(parent, childIndex, childCount)
	}

	return children[childIndex], nil
}

// TotalNodeCount returns the total number of nodes in the tree.
func (t *TaffyTree[NodeContext]) TotalNodeCount() int {
	return t.nodes.len()
}

// Parent returns the parent of childID, if any; it panics if childID does not exist.
func (t *TaffyTree[NodeContext]) Parent(childID NodeID) Option[NodeID] {
	return *t.parents.index(keyFromNodeID(childID))
}

// Children returns a copy of the list of children of parent.
func (t *TaffyTree[NodeContext]) Children(parent NodeID) ([]NodeID, error) {
	return slices.Clone(*t.children.index(keyFromNodeID(parent))), nil
}

// HoistedChildren returns the out-of-flow boxes whose containing block is node, as recorded by the last layout.
func (t *TaffyTree[NodeContext]) HoistedChildren(node NodeID) ([]NodeID, error) {
	return t.nodes.index(keyFromNodeID(node)).hoistedChildren, nil
}

// SetStyle sets the Style of node.
func (t *TaffyTree[NodeContext]) SetStyle(node NodeID, style Style[defaultCheapStr]) error {
	t.nodes.index(keyFromNodeID(node)).style = style
	return t.MarkDirty(node)
}

// Style returns the Style of node.
func (t *TaffyTree[NodeContext]) Style(node NodeID) (*Style[defaultCheapStr], error) {
	return &t.nodes.index(keyFromNodeID(node)).style, nil
}

// Layout returns the layout of node relative to its parent: rounded when rounding is enabled.
func (t *TaffyTree[NodeContext]) Layout(node NodeID) (*Layout, error) {
	if t.config.useRounding {
		return &t.nodes.index(keyFromNodeID(node)).finalLayout, nil
	}
	return &t.nodes.index(keyFromNodeID(node)).unroundedLayout, nil
}

// UnroundedLayout returns the layout of node with unrounded values relative to its parent.
func (t *TaffyTree[NodeContext]) UnroundedLayout(node NodeID) *Layout {
	return &t.nodes.index(keyFromNodeID(node)).unroundedLayout
}

// DetailedLayoutInfo returns the detailed layout info of nodeID (grid containers only).
func (t *TaffyTree[NodeContext]) DetailedLayoutInfo(nodeID NodeID) *DetailedLayoutInfo[defaultCheapStr] {
	return &t.nodes.index(keyFromNodeID(nodeID)).detailedLayoutInfo
}

// MarkDirty marks the layout of node and its ancestors as outdated.
func (t *TaffyTree[NodeContext]) MarkDirty(node NodeID) error {
	var markDirtyRecursive func(nodeKey keyData)
	markDirtyRecursive = func(nodeKey keyData) {
		switch t.nodes.index(nodeKey).markDirty() {
		case ClearStateAlreadyEmpty:
		case ClearStateCleared:
			if parent, ok := t.parents.get(nodeKey); ok {
				if p, ok := parent.Get(); ok {
					markDirtyRecursive(keyFromNodeID(p))
				}
			}
		}
	}

	markDirtyRecursive(keyFromNodeID(node))

	return nil
}

// Dirty reports whether the layout of node needs to be recomputed.
func (t *TaffyTree[NodeContext]) Dirty(node NodeID) (bool, error) {
	return t.nodes.index(keyFromNodeID(node)).cache.IsEmpty(), nil
}

// ComputeLayoutWithMeasure updates the stored layout of nodeID and its children, measuring leaves with measureFunction.
func (t *TaffyTree[NodeContext]) ComputeLayoutWithMeasure(nodeID NodeID, availableSpace Size[AvailableSpace], measureFunction MeasureFunction[NodeContext]) error {
	useRounding := t.config.useRounding
	taffyView := &taffyView[NodeContext]{taffy: t, measureFunction: measureFunction}
	ComputeRootLayout[defaultCheapStr](taffyView, nodeID, availableSpace)
	if useRounding {
		RoundLayout(taffyView, nodeID)
	}
	return nil
}

func zeroMeasure(inputs LayoutInput, style CoreStyle) LayoutOutput {
	return ComputeLeafLayout(
		inputs,
		style,
		func(unsafe.Pointer, float32) float32 { return 0.0 },
		func(Size[Option[float32]], Size[AvailableSpace]) Size[float32] { return SizeF32Zero },
	)
}

// ComputeLayout updates the stored layout of node and its children; leaves measure as zero.
func (t *TaffyTree[NodeContext]) ComputeLayout(node NodeID, availableSpace Size[AvailableSpace]) error {
	return t.ComputeLayoutWithMeasure(node, availableSpace, func(inputs LayoutInput, _ NodeID, _ *NodeContext, style *Style[defaultCheapStr]) LayoutOutput {
		return zeroMeasure(inputs, style.Ref())
	})
}

// PrintTree prints a debug representation of the tree's layout.
func (t *TaffyTree[NodeContext]) PrintTree(root NodeID) {
	PrintTreeStdout(t, root)
}

func (t *TaffyTree[NodeContext]) asLayoutTree() *taffyView[NodeContext] {
	return &taffyView[NodeContext]{
		taffy: t,
		measureFunction: func(inputs LayoutInput, _ NodeID, _ *NodeContext, style *Style[defaultCheapStr]) LayoutOutput {
			return zeroMeasure(inputs, style.Ref())
		},
	}
}
