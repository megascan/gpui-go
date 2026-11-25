package consumer

import (
	"iter"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

type nodeIDIterator interface {
	Next() (NodeID, bool)
	NextBack() (NodeID, bool)
	Len() int
}

func forward[T any](next func() (T, bool)) iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			v, ok := next()
			if !ok || !yield(v) {
				return
			}
		}
	}
}

// Nodes maps an iterator over node IDs to the nodes of a tree state.
type Nodes struct {
	state *State
	ids   nodeIDIterator
}

// Next returns the next node from the front.
func (it *Nodes) Next() (Node, bool) {
	id, ok := it.ids.Next()
	if !ok {
		return Node{}, false
	}
	return it.state.mustNode(id), true
}

// NextBack returns the next node from the back.
func (it *Nodes) NextBack() (Node, bool) {
	id, ok := it.ids.NextBack()
	if !ok {
		return Node{}, false
	}
	return it.state.mustNode(id), true
}

// Len returns the number of remaining nodes.
func (it *Nodes) Len() int {
	return it.ids.Len()
}

// All yields the remaining nodes from the front.
func (it *Nodes) All() iter.Seq[Node] {
	return forward(it.Next)
}

// Backward yields the remaining nodes from the back.
func (it *Nodes) Backward() iter.Seq[Node] {
	return forward(it.NextBack)
}

// ChildIDs iterates over child node IDs, handling both normal nodes and graft nodes.
type ChildIDs struct {
	graft      bool
	graftID    NodeID
	hasGraftID bool
	parentID   NodeID
	children   []a11y.NodeID
}

func (it *ChildIDs) takeGraft() (NodeID, bool) {
	id, ok := it.graftID, it.hasGraftID
	it.hasGraftID = false
	return id, ok
}

// Next returns the next child ID from the front.
func (it *ChildIDs) Next() (NodeID, bool) {
	if it.graft {
		return it.takeGraft()
	}
	if len(it.children) == 0 {
		return NodeID{}, false
	}
	child := it.children[0]
	it.children = it.children[1:]
	return it.parentID.withSameTree(child), true
}

// NextBack returns the next child ID from the back.
func (it *ChildIDs) NextBack() (NodeID, bool) {
	if it.graft {
		return it.takeGraft()
	}
	if len(it.children) == 0 {
		return NodeID{}, false
	}
	child := it.children[len(it.children)-1]
	it.children = it.children[:len(it.children)-1]
	return it.parentID.withSameTree(child), true
}

// Len returns the number of remaining child IDs.
func (it *ChildIDs) Len() int {
	if it.graft {
		if it.hasGraftID {
			return 1
		}
		return 0
	}
	return len(it.children)
}

// All yields the remaining child IDs from the front.
func (it *ChildIDs) All() iter.Seq[NodeID] {
	return forward(it.Next)
}

// Backward yields the remaining child IDs from the back.
func (it *ChildIDs) Backward() iter.Seq[NodeID] {
	return forward(it.NextBack)
}

// FollowingSiblings yields the IDs of the following siblings of a node.
type FollowingSiblings struct {
	backPosition  int
	done          bool
	frontPosition int
	parent        gpuiutil.Option[Node]
	nodeID        NodeID
}

func newFollowingSiblings(node Node) *FollowingSiblings {
	parent, index, ok := node.ParentAndIndex()
	backPosition, frontPosition, done := 0, 0, true
	if ok && !parent.IsGraft() {
		backPosition = len(parent.Data().Children()) - 1
		frontPosition = index + 1
		done = frontPosition > backPosition
	}
	it := &FollowingSiblings{backPosition: backPosition, done: done, frontPosition: frontPosition, nodeID: node.id}
	if ok {
		it.parent = gpuiutil.Some(parent)
	}
	return it
}

func siblingAt(parent gpuiutil.Option[Node], position int) (a11y.NodeID, bool) {
	p, ok := parent.Get()
	if !ok {
		return 0, false
	}
	children := p.Data().Children()
	if position < 0 || position >= len(children) {
		return 0, false
	}
	return children[position], true
}

// Next returns the next following sibling ID from the front.
func (it *FollowingSiblings) Next() (NodeID, bool) {
	if it.done {
		return NodeID{}, false
	}
	it.done = it.frontPosition == it.backPosition
	child, ok := siblingAt(it.parent, it.frontPosition)
	if !ok {
		return NodeID{}, false
	}
	it.frontPosition++
	return it.nodeID.withSameTree(child), true
}

// NextBack returns the next following sibling ID from the back.
func (it *FollowingSiblings) NextBack() (NodeID, bool) {
	if it.done {
		return NodeID{}, false
	}
	it.done = it.backPosition == it.frontPosition
	child, ok := siblingAt(it.parent, it.backPosition)
	if !ok {
		return NodeID{}, false
	}
	it.backPosition--
	return it.nodeID.withSameTree(child), true
}

// Len returns the number of remaining sibling IDs.
func (it *FollowingSiblings) Len() int {
	if it.done {
		return 0
	}
	return it.backPosition + 1 - it.frontPosition
}

// All yields the remaining sibling IDs from the front.
func (it *FollowingSiblings) All() iter.Seq[NodeID] {
	return forward(it.Next)
}

// Backward yields the remaining sibling IDs from the back.
func (it *FollowingSiblings) Backward() iter.Seq[NodeID] {
	return forward(it.NextBack)
}

// PrecedingSiblings yields the IDs of the preceding siblings of a node.
type PrecedingSiblings struct {
	backPosition  int
	done          bool
	frontPosition int
	parent        gpuiutil.Option[Node]
	nodeID        NodeID
}

func newPrecedingSiblings(node Node) *PrecedingSiblings {
	parent, index, ok := node.ParentAndIndex()
	backPosition, frontPosition, done := 0, 0, true
	if ok && !parent.IsGraft() {
		frontPosition = max(index-1, 0)
		done = index == 0
	}
	it := &PrecedingSiblings{backPosition: backPosition, done: done, frontPosition: frontPosition, nodeID: node.id}
	if ok {
		it.parent = gpuiutil.Some(parent)
	}
	return it
}

// Next returns the next preceding sibling ID from the front.
func (it *PrecedingSiblings) Next() (NodeID, bool) {
	if it.done {
		return NodeID{}, false
	}
	it.done = it.frontPosition == it.backPosition
	child, ok := siblingAt(it.parent, it.frontPosition)
	if !ok {
		return NodeID{}, false
	}
	if !it.done {
		it.frontPosition--
	}
	return it.nodeID.withSameTree(child), true
}

// NextBack returns the next preceding sibling ID from the back.
func (it *PrecedingSiblings) NextBack() (NodeID, bool) {
	if it.done {
		return NodeID{}, false
	}
	it.done = it.backPosition == it.frontPosition
	child, ok := siblingAt(it.parent, it.backPosition)
	if !ok {
		return NodeID{}, false
	}
	it.backPosition++
	return it.nodeID.withSameTree(child), true
}

// Len returns the number of remaining sibling IDs.
func (it *PrecedingSiblings) Len() int {
	if it.done {
		return 0
	}
	return it.frontPosition + 1 - it.backPosition
}

// All yields the remaining sibling IDs from the front.
func (it *PrecedingSiblings) All() iter.Seq[NodeID] {
	return forward(it.Next)
}

// Backward yields the remaining sibling IDs from the back.
func (it *PrecedingSiblings) Backward() iter.Seq[NodeID] {
	return forward(it.NextBack)
}

func nextFilteredSibling(node gpuiutil.Option[Node], filter func(*Node) FilterResult) gpuiutil.Option[Node] {
	next := node
	considerChildren := false
	for {
		current, ok := next.Get()
		if !ok {
			return next
		}
		if considerChildren {
			if child, ok := current.Children().Next(); ok {
				result := filter(&child)
				next = gpuiutil.Some(child)
				if result == FilterResultInclude {
					return next
				}
				considerChildren = result == FilterResultExcludeNode
				continue
			}
		}
		if sibling, ok := current.FollowingSiblings().Next(); ok {
			result := filter(&sibling)
			next = gpuiutil.Some(sibling)
			if result == FilterResultInclude {
				return next
			}
			if result == FilterResultExcludeNode {
				considerChildren = true
			}
			continue
		}
		parent, ok := current.Parent()
		if !ok {
			return gpuiutil.None[Node]()
		}
		next = gpuiutil.Some(parent)
		if filter(&parent) != FilterResultExcludeNode {
			return gpuiutil.None[Node]()
		}
		considerChildren = false
	}
}

func previousFilteredSibling(node gpuiutil.Option[Node], filter func(*Node) FilterResult) gpuiutil.Option[Node] {
	previous := node
	considerChildren := false
	for {
		current, ok := previous.Get()
		if !ok {
			return previous
		}
		if considerChildren {
			if child, ok := current.Children().NextBack(); ok {
				result := filter(&child)
				previous = gpuiutil.Some(child)
				if result == FilterResultInclude {
					return previous
				}
				considerChildren = result == FilterResultExcludeNode
				continue
			}
		}
		if sibling, ok := current.PrecedingSiblings().Next(); ok {
			result := filter(&sibling)
			previous = gpuiutil.Some(sibling)
			if result == FilterResultInclude {
				return previous
			}
			if result == FilterResultExcludeNode {
				considerChildren = true
			}
			continue
		}
		parent, ok := current.Parent()
		if !ok {
			return gpuiutil.None[Node]()
		}
		previous = gpuiutil.Some(parent)
		if filter(&parent) != FilterResultExcludeNode {
			return gpuiutil.None[Node]()
		}
		considerChildren = false
	}
}

func optionOf(n Node, ok bool) gpuiutil.Option[Node] {
	if !ok {
		return gpuiutil.None[Node]()
	}
	return gpuiutil.Some(n)
}

type filteredRange struct {
	filter func(*Node) FilterResult
	back   gpuiutil.Option[Node]
	done   bool
	front  gpuiutil.Option[Node]
}

func newFilteredRange(filter func(*Node) FilterResult, front, back gpuiutil.Option[Node]) filteredRange {
	return filteredRange{filter: filter, back: back, done: back.IsNone() || front.IsNone(), front: front}
}

func (r *filteredRange) updateDone() {
	f, fok := r.front.Get()
	b, bok := r.back.Get()
	r.done = !fok || !bok || f.ID() == b.ID()
}

func (r *filteredRange) step(end *gpuiutil.Option[Node], advance func(gpuiutil.Option[Node], func(*Node) FilterResult) gpuiutil.Option[Node]) (Node, bool) {
	if r.done {
		return Node{}, false
	}
	r.updateDone()
	current := *end
	*end = advance(*end, r.filter)
	return current.Get()
}

// FollowingFilteredSiblings yields the following siblings of a node according to a filter.
type FollowingFilteredSiblings struct{ r filteredRange }

func newFollowingFilteredSiblings(node Node, filter func(*Node) FilterResult) *FollowingFilteredSiblings {
	front := nextFilteredSibling(gpuiutil.Some(node), filter)
	back := gpuiutil.None[Node]()
	if parent, ok := node.FilteredParent(filter); ok {
		back = optionOf(parent.lastFilteredChild(filter))
	}
	return &FollowingFilteredSiblings{r: newFilteredRange(filter, front, back)}
}

// Next returns the next sibling from the front.
func (it *FollowingFilteredSiblings) Next() (Node, bool) {
	return it.r.step(&it.r.front, nextFilteredSibling)
}

// NextBack returns the next sibling from the back.
func (it *FollowingFilteredSiblings) NextBack() (Node, bool) {
	return it.r.step(&it.r.back, previousFilteredSibling)
}

// All yields the remaining siblings from the front.
func (it *FollowingFilteredSiblings) All() iter.Seq[Node] {
	return forward(it.Next)
}

// Backward yields the remaining siblings from the back.
func (it *FollowingFilteredSiblings) Backward() iter.Seq[Node] {
	return forward(it.NextBack)
}

// PrecedingFilteredSiblings yields the preceding siblings of a node according to a filter.
type PrecedingFilteredSiblings struct{ r filteredRange }

func newPrecedingFilteredSiblings(node Node, filter func(*Node) FilterResult) *PrecedingFilteredSiblings {
	front := previousFilteredSibling(gpuiutil.Some(node), filter)
	back := gpuiutil.None[Node]()
	if parent, ok := node.FilteredParent(filter); ok {
		back = optionOf(parent.firstFilteredChild(filter))
	}
	return &PrecedingFilteredSiblings{r: newFilteredRange(filter, front, back)}
}

// Next returns the next sibling from the front.
func (it *PrecedingFilteredSiblings) Next() (Node, bool) {
	return it.r.step(&it.r.front, previousFilteredSibling)
}

// NextBack returns the next sibling from the back.
func (it *PrecedingFilteredSiblings) NextBack() (Node, bool) {
	return it.r.step(&it.r.back, nextFilteredSibling)
}

// All yields the remaining siblings from the front.
func (it *PrecedingFilteredSiblings) All() iter.Seq[Node] {
	return forward(it.Next)
}

// Backward yields the remaining siblings from the back.
func (it *PrecedingFilteredSiblings) Backward() iter.Seq[Node] {
	return forward(it.NextBack)
}

// FilteredChildren yields the children of a node according to a filter.
type FilteredChildren struct{ r filteredRange }

func newFilteredChildren(node Node, filter func(*Node) FilterResult) *FilteredChildren {
	front := optionOf(node.firstFilteredChild(filter))
	back := optionOf(node.lastFilteredChild(filter))
	return &FilteredChildren{r: newFilteredRange(filter, front, back)}
}

// Next returns the next child from the front.
func (it *FilteredChildren) Next() (Node, bool) {
	return it.r.step(&it.r.front, nextFilteredSibling)
}

// NextBack returns the next child from the back.
func (it *FilteredChildren) NextBack() (Node, bool) {
	return it.r.step(&it.r.back, previousFilteredSibling)
}

// All yields the remaining children from the front.
func (it *FilteredChildren) All() iter.Seq[Node] {
	return forward(it.Next)
}

// Backward yields the remaining children from the back.
func (it *FilteredChildren) Backward() iter.Seq[Node] {
	return forward(it.NextBack)
}

// LabelledBy yields the nodes that label a node, either explicit ones or label descendants.
type LabelledBy struct {
	fromDescendants *FilteredChildren
	ids             []a11y.NodeID
	treeState       *State
	nodeID          NodeID
}

// Next returns the next labelling node from the front.
func (it *LabelledBy) Next() (Node, bool) {
	if it.fromDescendants != nil {
		return it.fromDescendants.Next()
	}
	if len(it.ids) == 0 {
		return Node{}, false
	}
	id := it.ids[0]
	it.ids = it.ids[1:]
	return it.treeState.mustNode(it.nodeID.withSameTree(id)), true
}

// NextBack returns the next labelling node from the back.
func (it *LabelledBy) NextBack() (Node, bool) {
	if it.fromDescendants != nil {
		return it.fromDescendants.NextBack()
	}
	if len(it.ids) == 0 {
		return Node{}, false
	}
	id := it.ids[len(it.ids)-1]
	it.ids = it.ids[:len(it.ids)-1]
	return it.treeState.mustNode(it.nodeID.withSameTree(id)), true
}

// All yields the remaining labelling nodes from the front.
func (it *LabelledBy) All() iter.Seq[Node] {
	return forward(it.Next)
}

// Backward yields the remaining labelling nodes from the back.
func (it *LabelledBy) Backward() iter.Seq[Node] {
	return forward(it.NextBack)
}
