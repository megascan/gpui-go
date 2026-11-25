// Package sumtree is a port of Zed's sum_tree crate: a persistent B+ tree
// whose nodes cache a summary of their subtree.
package sumtree

import (
	"fmt"
	"iter"
	"reflect"
	"slices"
)

// TreeBase is the minimum fan-out of a node; a node holds at most 2*TreeBase children.
// Tests in this package run with a base of 2, as upstream's cfg(test) build does.
const TreeBase = 6

var treeBase = TreeBase

const capacityPanic = "item should fit into fixed size ArrayVec"

// Item is a value stored in a SumTree, summarized by S under context C.
type Item[S any, C any] interface {
	Summary(cx C) S
}

// KeyedItem is an Item whose summary has a key identifying it.
type KeyedItem[S any, C any, K any] interface {
	Item[S, C]
	Key() K
}

// Summary describes the sum of all items in a subtree. Zero is called on the
// zero value of S; AddSummary returns the receiver with summary added.
type Summary[S any, C any] interface {
	Zero(cx C) S
	AddSummary(summary S, cx C) S
}

// ContextLessSummary is a Summary whose context is the unit struct.
type ContextLessSummary[S any] = Summary[S, struct{}]

// NoSummary is a catch-all Summary that carries no data.
type NoSummary struct{}

// Zero returns NoSummary.
func (NoSummary) Zero(struct{}) NoSummary { return NoSummary{} }

// AddSummary returns NoSummary.
func (NoSummary) AddSummary(NoSummary, struct{}) NoSummary { return NoSummary{} }

// Dimension is a quantity measured by a Summary S that can be used to seek in a
// SumTree. Every Summary is a Dimension of itself.
type Dimension[D any, S any, C any] interface {
	Zero(cx C) D
	AddSummary(summary S, cx C) D
}

// WithAddedSummary returns d with summary added.
func WithAddedSummary[D Dimension[D, S, C], S any, C any](d D, summary S, cx C) D {
	return d.AddSummary(summary, cx)
}

// FromSummary returns the dimension of summary measured from zero.
func FromSummary[D Dimension[D, S, C], S any, C any](summary S, cx C) D {
	dimension := zeroDim[D](cx)
	return dimension.AddSummary(summary, cx)
}

func zeroDim[D Dimension[D, S, C], S any, C any](cx C) D {
	var d D
	return d.Zero(cx)
}

// SeekTarget is a position a cursor over dimension D can seek to. Cmp
// compares the target with the cursor location like cmp.Compare.
type SeekTarget[S any, D any, C any] interface {
	Cmp(cursorLocation D, cx C) int
}

// OrdDimension is a dimension that is also a seek target for itself, the Go
// form of upstream's `Dimension + Ord` key bound.
type OrdDimension[K any, S any, C any] interface {
	Dimension[K, S, C]
	SeekTarget[S, K, C]
}

// Unit is the empty dimension, upstream's `()`.
type Unit[S any, C any] struct{}

// Zero returns Unit.
func (Unit[S, C]) Zero(C) Unit[S, C] { return Unit[S, C]{} }

// AddSummary returns Unit.
func (Unit[S, C]) AddSummary(S, C) Unit[S, C] { return Unit[S, C]{} }

// Dimensions measures up to three dimensions at once.
type Dimensions[S any, C any, D1 Dimension[D1, S, C], D2 Dimension[D2, S, C], D3 Dimension[D3, S, C]] struct {
	V0 D1
	V1 D2
	V2 D3
}

// Zero returns the zero of every dimension.
func (Dimensions[S, C, D1, D2, D3]) Zero(cx C) Dimensions[S, C, D1, D2, D3] {
	return Dimensions[S, C, D1, D2, D3]{zeroDim[D1](cx), zeroDim[D2](cx), zeroDim[D3](cx)}
}

// AddSummary adds summary to every dimension.
func (d Dimensions[S, C, D1, D2, D3]) AddSummary(summary S, cx C) Dimensions[S, C, D1, D2, D3] {
	d.V0 = d.V0.AddSummary(summary, cx)
	d.V1 = d.V1.AddSummary(summary, cx)
	d.V2 = d.V2.AddSummary(summary, cx)
	return d
}

// DimensionsTarget seeks a Dimensions cursor by its first dimension.
type DimensionsTarget[S any, C any, D1 Dimension[D1, S, C], D2 Dimension[D2, S, C], D3 Dimension[D3, S, C]] struct {
	Target SeekTarget[S, D1, C]
}

// Cmp compares the target with the first dimension of cursorLocation.
func (t DimensionsTarget[S, C, D1, D2, D3]) Cmp(cursorLocation Dimensions[S, C, D1, D2, D3], cx C) int {
	return t.Target.Cmp(cursorLocation.V0, cx)
}

// Bias settles ambiguities when determining positions in an ordered sequence.
type Bias uint8

const (
	// BiasLeft attaches to the character on the left.
	BiasLeft Bias = iota
	// BiasRight attaches to the character on the right.
	BiasRight
)

// Invert returns the opposite bias.
func (b Bias) Invert() Bias {
	switch b {
	case BiasLeft:
		return BiasRight
	default:
		return BiasLeft
	}
}

// SumTree is a B+ tree whose leaves hold items and whose nodes hold the
// summary of their subtree. Copying a SumTree is cheap; nodes are shared and
// copied on write. The zero value is an empty tree under the zero context.
type SumTree[T Item[S, C], S Summary[S, C], C any] struct {
	node *Node[T, S, C]
}

// Node is one node of a SumTree: internal when it has child trees, a leaf when it has items.
type Node[T Item[S, C], S Summary[S, C], C any] struct {
	internal       bool
	height         uint8
	summary        S
	childSummaries []S
	childTrees     []SumTree[T, S, C]
	items          []T
}

func newLeaf[T Item[S, C], S Summary[S, C], C any](summary S) *Node[T, S, C] {
	return &Node[T, S, C]{summary: summary}
}

// NewSumTree returns an empty tree.
func NewSumTree[T Item[S, C], S Summary[S, C], C any](cx C) SumTree[T, S, C] {
	var s S
	return SumTree[T, S, C]{newLeaf[T](s.Zero(cx))}
}

// SumTreeFromSummary returns an empty tree with the given zero summary.
func SumTreeFromSummary[T Item[S, C], S Summary[S, C], C any](summary S) SumTree[T, S, C] {
	return SumTree[T, S, C]{newLeaf[T](summary)}
}

// SumTreeFromItem returns a tree holding item.
func SumTreeFromItem[T Item[S, C], S Summary[S, C], C any](item T, cx C) SumTree[T, S, C] {
	tree := NewSumTree[T](cx)
	tree.Push(item, cx)
	return tree
}

func leafFromItems[T Item[S, C], S Summary[S, C], C any](items []T, cx C) SumTree[T, S, C] {
	itemSummaries := make([]S, len(items))
	for i, item := range items {
		itemSummaries[i] = item.Summary(cx)
	}
	summary := itemSummaries[0]
	for _, itemSummary := range itemSummaries[1:] {
		summary = summary.AddSummary(itemSummary, cx)
	}
	return SumTree[T, S, C]{&Node[T, S, C]{summary: summary, items: items, childSummaries: itemSummaries}}
}

// SumTreeFromIter builds a tree from items in order.
func SumTreeFromIter[T Item[S, C], S Summary[S, C], C any](items []T, cx C) SumTree[T, S, C] {
	var nodes []SumTree[T, S, C]
	for start := 0; start < len(items); start += 2 * treeBase {
		end := min(start+2*treeBase, len(items))
		nodes = append(nodes, leafFromItems[T](slices.Clone(items[start:end]), cx))
	}

	var parentNodes []SumTree[T, S, C]
	var height uint8
	for len(nodes) > 1 {
		height++
		var currentParentNode *Node[T, S, C]
		for _, childNode := range nodes {
			if currentParentNode == nil {
				currentParentNode = &Node[T, S, C]{internal: true, height: height, summary: zeroDim[S](cx)}
			}
			childSummary := childNode.Summary()
			currentParentNode.summary = currentParentNode.summary.AddSummary(childSummary, cx)
			currentParentNode.childSummaries = pushCap(currentParentNode.childSummaries, childSummary)
			currentParentNode.childTrees = pushCap(currentParentNode.childTrees, childNode)
			if len(currentParentNode.childTrees) == 2*treeBase {
				parentNodes = append(parentNodes, SumTree[T, S, C]{currentParentNode})
				currentParentNode = nil
			}
		}
		if currentParentNode != nil {
			parentNodes = append(parentNodes, SumTree[T, S, C]{currentParentNode})
		}
		nodes, parentNodes = parentNodes, nil
	}

	if len(nodes) == 0 {
		return NewSumTree[T](cx)
	}
	return nodes[len(nodes)-1]
}

// SumTreeFromParIter builds a tree from items in chunks, as upstream's
// parallel constructor does; the Go port runs the chunks sequentially.
func SumTreeFromParIter[T Item[S, C], S Summary[S, C], C any](items []T, cx C) SumTree[T, S, C] {
	var nodes []SumTree[T, S, C]
	for start := 0; start < len(items); start += 2 * treeBase {
		end := min(start+2*treeBase, len(items))
		nodes = append(nodes, leafFromItems[T](slices.Clone(items[start:end]), cx))
	}

	var height uint8
	for len(nodes) > 1 {
		height++
		var parents []SumTree[T, S, C]
		for start := 0; start < len(nodes); start += 2 * treeBase {
			end := min(start+2*treeBase, len(nodes))
			childTrees := slices.Clone(nodes[start:end])
			childSummaries := make([]S, len(childTrees))
			for i, childTree := range childTrees {
				childSummaries[i] = childTree.Summary()
			}
			summary := childSummaries[0]
			for _, childSummary := range childSummaries[1:] {
				summary = summary.AddSummary(childSummary, cx)
			}
			parents = append(parents, SumTree[T, S, C]{&Node[T, S, C]{
				internal:       true,
				height:         height,
				summary:        summary,
				childSummaries: childSummaries,
				childTrees:     childTrees,
			}})
		}
		nodes = parents
	}

	if len(nodes) == 0 {
		return NewSumTree[T](cx)
	}
	return nodes[len(nodes)-1]
}

func (t SumTree[T, S, C]) root() *Node[T, S, C] {
	if t.node == nil {
		var cx C
		return newLeaf[T](zeroDim[S](cx))
	}
	return t.node
}

func (t *SumTree[T, S, C]) makeMut() *Node[T, S, C] {
	n := *t.root()
	n.childSummaries = slices.Clone(n.childSummaries)
	n.childTrees = slices.Clone(n.childTrees)
	n.items = slices.Clone(n.items)
	t.node = &n
	return &n
}

// Items returns all items in order.
func (t SumTree[T, S, C]) Items(cx C) []T {
	var items []T
	cursor := NewCursor[T, S, C, Unit[S, C]](t, cx)
	cursor.Next()
	for {
		item, ok := cursor.Item()
		if !ok {
			break
		}
		items = append(items, item)
		cursor.Next()
	}
	return items
}

// Iter returns the items in order.
func (t SumTree[T, S, C]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		it := newIter(t)
		for {
			item, ok := it.Next()
			if !ok || !yield(item) {
				return
			}
		}
	}
}

// FindExact returns the start and end of the item that exactly matches target, and the item.
func (t SumTree[T, S, C]) FindExact[D Dimension[D, S, C]](cx C, target SeekTarget[S, D, C], bias Bias) (D, D, T, bool) {
	return findImpl[T, S, C, D](t, cx, target, bias, true)
}

// Find is a faster Cursor + Seek + Item.
func (t SumTree[T, S, C]) Find[D Dimension[D, S, C]](cx C, target SeekTarget[S, D, C], bias Bias) (D, D, T, bool) {
	return findImpl[T, S, C, D](t, cx, target, bias, false)
}

func findImpl[T Item[S, C], S Summary[S, C], C any, D Dimension[D, S, C]](t SumTree[T, S, C], cx C, target SeekTarget[S, D, C], bias Bias, exact bool) (D, D, T, bool) {
	var none T
	treeEnd := zeroDim[D](cx).AddSummary(t.Summary(), cx)
	comparison := target.Cmp(treeEnd, cx)
	if comparison > 0 || (comparison == 0 && bias == BiasRight) {
		return treeEnd, treeEnd, none, false
	}

	pos := zeroDim[D](cx)
	if item, end, ok := findIterate(cx, target, bias, &pos, t, exact); ok {
		return pos, end, item, true
	}
	return pos, pos, none, false
}

func findIterate[T Item[S, C], S Summary[S, C], C any, D Dimension[D, S, C]](cx C, target SeekTarget[S, D, C], bias Bias, position *D, this SumTree[T, S, C], exact bool) (T, D, bool) {
iterate:
	for {
		node := this.root()
		if node.internal {
			for i, childTree := range node.childTrees {
				childEnd := (*position).AddSummary(node.childSummaries[i], cx)
				comparison := target.Cmp(childEnd, cx)
				if comparison < 0 || (comparison == 0 && bias == BiasLeft) {
					this = childTree
					continue iterate
				}
				*position = childEnd
			}
		} else {
			for i, item := range node.items {
				childEnd := (*position).AddSummary(node.childSummaries[i], cx)
				comparison := target.Cmp(childEnd, cx)
				var entryFound bool
				if exact {
					entryFound = comparison == 0
				} else {
					entryFound = comparison < 0 || (comparison == 0 && bias == BiasLeft)
				}
				if entryFound {
					return item, childEnd, true
				}
				*position = childEnd
			}
		}
		var none T
		var zero D
		return none, zero, false
	}
}

// FindWithPrev is Find that also returns the item before the found one.
func (t SumTree[T, S, C]) FindWithPrev[D Dimension[D, S, C]](cx C, target SeekTarget[S, D, C], bias Bias) (start D, end D, prev T, hasPrev bool, item T, ok bool) {
	treeEnd := zeroDim[D](cx).AddSummary(t.Summary(), cx)
	comparison := target.Cmp(treeEnd, cx)
	if comparison > 0 || (comparison == 0 && bias == BiasRight) {
		return treeEnd, treeEnd, prev, false, item, false
	}

	pos := zeroDim[D](cx)
	if p, hp, it, e, found := findWithPrevIterate(cx, target, bias, &pos, t); found {
		return pos, e, p, hp, it, true
	}
	return pos, pos, prev, false, item, false
}

func findWithPrevIterate[T Item[S, C], S Summary[S, C], C any, D Dimension[D, S, C]](cx C, target SeekTarget[S, D, C], bias Bias, position *D, this SumTree[T, S, C]) (T, bool, T, D, bool) {
	var prev T
	hasPrev := false
iterate:
	for {
		node := this.root()
		if node.internal {
			for i, childTree := range node.childTrees {
				childEnd := (*position).AddSummary(node.childSummaries[i], cx)
				comparison := target.Cmp(childEnd, cx)
				if comparison < 0 || (comparison == 0 && bias == BiasLeft) {
					this = childTree
					continue iterate
				}
				prev, hasPrev = childTree.Last()
				*position = childEnd
			}
		} else {
			for i, item := range node.items {
				childEnd := (*position).AddSummary(node.childSummaries[i], cx)
				comparison := target.Cmp(childEnd, cx)
				if comparison < 0 || (comparison == 0 && bias == BiasLeft) {
					return prev, hasPrev, item, childEnd, true
				}
				prev, hasPrev = item, true
				*position = childEnd
			}
		}
		var none T
		var zero D
		return none, false, none, zero, false
	}
}

// Cursor returns a cursor over the tree measuring dimension D.
func (t SumTree[T, S, C]) Cursor[D Dimension[D, S, C]](cx C) *Cursor[T, S, C, D] {
	return NewCursor[T, S, C, D](t, cx)
}

// Filter returns a cursor that only visits subtrees whose summary satisfies filterNode.
func (t SumTree[T, S, C]) Filter[U Dimension[U, S, C]](cx C, filterNode func(S) bool) *FilterCursor[T, S, C, U] {
	return NewFilterCursor[T, S, C, U](t, cx, filterNode)
}

// First returns the first item.
func (t SumTree[T, S, C]) First() (T, bool) {
	items := t.leftmostLeaf().root().itemsOf()
	if len(items) == 0 {
		var none T
		return none, false
	}
	return items[0], true
}

// Last returns the last item.
func (t SumTree[T, S, C]) Last() (T, bool) {
	items := t.rightmostLeaf().root().itemsOf()
	if len(items) == 0 {
		var none T
		return none, false
	}
	return items[len(items)-1], true
}

// LastSummary returns the summary of the last item.
func (t SumTree[T, S, C]) LastSummary() (S, bool) {
	summaries := t.rightmostLeaf().root().childSummaries
	if len(summaries) == 0 {
		var none S
		return none, false
	}
	return summaries[len(summaries)-1], true
}

// UpdateLast applies f to the last item and refreshes the summaries above it.
func (t *SumTree[T, S, C]) UpdateLast(f func(*T), cx C) {
	t.updateLastRecursive(f, cx)
}

func (t *SumTree[T, S, C]) updateLastRecursive(f func(*T), cx C) (S, bool) {
	n := t.makeMut()
	if n.internal {
		last := len(n.childTrees) - 1
		lastSummary, ok := n.childTrees[last].updateLastRecursive(f, cx)
		if !ok {
			panic("called `Option::unwrap()` on a `None` value")
		}
		n.childSummaries[last] = lastSummary
		n.summary = sum(n.childSummaries, cx)
		return n.summary, true
	}
	if len(n.items) == 0 || len(n.childSummaries) == 0 {
		var none S
		return none, false
	}
	last := len(n.items) - 1
	f(&n.items[last])
	n.childSummaries[len(n.childSummaries)-1] = n.items[last].Summary(cx)
	n.summary = sum(n.childSummaries, cx)
	return n.summary, true
}

// UpdateFirst applies f to the first item and refreshes the summaries above it.
func (t *SumTree[T, S, C]) UpdateFirst(f func(*T), cx C) {
	t.updateFirstRecursive(f, cx)
}

func (t *SumTree[T, S, C]) updateFirstRecursive(f func(*T), cx C) (S, bool) {
	n := t.makeMut()
	if n.internal {
		firstSummary, ok := n.childTrees[0].updateFirstRecursive(f, cx)
		if !ok {
			panic("called `Option::unwrap()` on a `None` value")
		}
		n.childSummaries[0] = firstSummary
		n.summary = sum(n.childSummaries, cx)
		return n.summary, true
	}
	if len(n.items) == 0 || len(n.childSummaries) == 0 {
		var none S
		return none, false
	}
	f(&n.items[0])
	n.childSummaries[0] = n.items[0].Summary(cx)
	n.summary = sum(n.childSummaries, cx)
	return n.summary, true
}

// Extent returns the dimension D of the whole tree.
func (t SumTree[T, S, C]) Extent[D Dimension[D, S, C]](cx C) D {
	extent := zeroDim[D](cx)
	return extent.AddSummary(t.root().summary, cx)
}

// Summary returns the summary of the whole tree.
func (t SumTree[T, S, C]) Summary() S {
	return t.root().summary
}

// IsEmpty reports whether the tree holds no items.
func (t SumTree[T, S, C]) IsEmpty() bool {
	n := t.root()
	if n.internal {
		return false
	}
	return len(n.items) == 0
}

// Extend appends items.
func (t *SumTree[T, S, C]) Extend(items []T, cx C) {
	t.Append(SumTreeFromIter[T](items, cx), cx)
}

// ParExtend appends items through SumTreeFromParIter.
func (t *SumTree[T, S, C]) ParExtend(items []T, cx C) {
	t.Append(SumTreeFromParIter[T](items, cx), cx)
}

// Push appends one item.
func (t *SumTree[T, S, C]) Push(item T, cx C) {
	summary := item.Summary(cx)
	t.Append(SumTree[T, S, C]{&Node[T, S, C]{
		summary:        summary,
		items:          []T{item},
		childSummaries: []S{summary},
	}}, cx)
}

// Append appends every item of other.
func (t *SumTree[T, S, C]) Append(other SumTree[T, S, C], cx C) {
	if t.IsEmpty() {
		*t = other
	} else if otherNode := other.root(); !otherNode.isLeaf() || len(otherNode.items) != 0 {
		if t.root().height < otherNode.height {
			if tree, ok := appendLarge(*t, &other, cx); ok {
				*t = fromChildTrees(tree, other, cx)
			} else {
				*t = other
			}
		} else if splitTree, ok := t.pushTreeRecursive(other, cx); ok {
			*t = fromChildTrees(*t, splitTree, cx)
		}
	}
}

func (t *SumTree[T, S, C]) pushTreeRecursive(other SumTree[T, S, C], cx C) (SumTree[T, S, C], bool) {
	n := t.makeMut()
	otherNode := other.root()
	if n.internal {
		n.summary = n.summary.AddSummary(otherNode.summary, cx)

		heightDelta := n.height - otherNode.height
		var summariesToAppend []S
		var treesToAppend []SumTree[T, S, C]
		if heightDelta == 0 {
			summariesToAppend = extendCap(summariesToAppend, otherNode.childSummaries)
			treesToAppend = extendCap(treesToAppend, otherNode.childTreesOf())
		} else if heightDelta == 1 && !otherNode.isUnderflowing() {
			summariesToAppend = pushCap(summariesToAppend, otherNode.summary)
			treesToAppend = pushCap(treesToAppend, other)
		} else {
			last := len(n.childTrees) - 1
			treeToAppend, ok := n.childTrees[last].pushTreeRecursive(other, cx)
			n.childSummaries[len(n.childSummaries)-1] = n.childTrees[last].root().summary
			if ok {
				summariesToAppend = pushCap(summariesToAppend, treeToAppend.root().summary)
				treesToAppend = pushCap(treesToAppend, treeToAppend)
			}
		}

		childCount := len(n.childTrees) + len(treesToAppend)
		if childCount > 2*treeBase {
			midpoint := (childCount + childCount%2) / 2
			allSummaries := slices.Concat(n.childSummaries, summariesToAppend)
			allTrees := slices.Concat(n.childTrees, treesToAppend)
			leftSummaries := slices.Clone(allSummaries[:midpoint])
			rightSummaries := slices.Clone(allSummaries[midpoint:])
			leftTrees := slices.Clone(allTrees[:midpoint])
			rightTrees := slices.Clone(allTrees[midpoint:])
			checkCap(len(rightSummaries))
			n.summary = sum(leftSummaries, cx)
			n.childSummaries = leftSummaries
			n.childTrees = leftTrees

			return SumTree[T, S, C]{&Node[T, S, C]{
				internal:       true,
				height:         n.height,
				summary:        sum(rightSummaries, cx),
				childSummaries: rightSummaries,
				childTrees:     rightTrees,
			}}, true
		}
		n.childSummaries = extendCap(n.childSummaries, summariesToAppend)
		n.childTrees = extendCap(n.childTrees, treesToAppend)
		return SumTree[T, S, C]{}, false
	}

	childCount := len(n.items) + len(otherNode.itemsOf())
	if childCount > 2*treeBase {
		midpoint := (childCount + childCount%2) / 2
		allItems := slices.Concat(n.items, otherNode.itemsOf())
		allSummaries := slices.Concat(n.childSummaries, otherNode.childSummaries)
		leftItems := slices.Clone(allItems[:midpoint])
		rightItems := slices.Clone(allItems[midpoint:])
		leftSummaries := slices.Clone(allSummaries[:midpoint])
		rightSummaries := slices.Clone(allSummaries[midpoint:])
		checkCap(len(rightSummaries))
		n.items = leftItems
		n.childSummaries = leftSummaries
		n.summary = sum(n.childSummaries, cx)
		return SumTree[T, S, C]{&Node[T, S, C]{
			items:          rightItems,
			summary:        sum(rightSummaries, cx),
			childSummaries: rightSummaries,
		}}, true
	}
	n.summary = n.summary.AddSummary(otherNode.summary, cx)
	n.items = extendCap(n.items, otherNode.itemsOf())
	n.childSummaries = extendCap(n.childSummaries, otherNode.childSummaries)
	return SumTree[T, S, C]{}, false
}

func appendLarge[T Item[S, C], S Summary[S, C], C any](small SumTree[T, S, C], large *SumTree[T, S, C], cx C) (SumTree[T, S, C], bool) {
	if small.root().height == large.root().height {
		if !small.root().isUnderflowing() {
			return small, true
		}
		return mergeIntoRight(small, large, cx)
	}
	n := large.makeMut()
	if !n.internal {
		panic("internal error: entered unreachable code")
	}
	fullSummary := small.Summary()
	fullSummary = fullSummary.AddSummary(n.summary, cx)
	n.summary = fullSummary

	res, ok := appendLarge(small, &n.childTrees[0], cx)
	n.childSummaries[0] = n.childTrees[0].Summary()
	if !ok {
		return SumTree[T, S, C]{}, false
	}
	tree := res
	if len(n.childTrees) < 2*treeBase {
		n.childSummaries = insertCap(n.childSummaries, 0, tree.Summary())
		n.childTrees = insertCap(n.childTrees, 0, tree)
		return SumTree[T, S, C]{}, false
	}
	newChildSummaries := append([]S{tree.Summary()}, n.childSummaries[:treeBase]...)
	n.childSummaries = slices.Clone(n.childSummaries[treeBase:])
	newChildTrees := append([]SumTree[T, S, C]{tree}, n.childTrees[:treeBase]...)
	n.childTrees = slices.Clone(n.childTrees[treeBase:])
	newTree := SumTree[T, S, C]{&Node[T, S, C]{
		internal:       true,
		height:         n.height,
		summary:        sum(newChildSummaries, cx),
		childSummaries: newChildSummaries,
		childTrees:     newChildTrees,
	}}
	n.summary = sum(n.childSummaries, cx)
	return newTree, true
}

func mergeIntoRight[T Item[S, C], S Summary[S, C], C any](small SumTree[T, S, C], large *SumTree[T, S, C], cx C) (SumTree[T, S, C], bool) {
	smallNode := small.root()
	n := large.makeMut()
	switch {
	case smallNode.internal && n.internal:
		totalChildCount := len(n.childTrees) + len(smallNode.childTrees)
		if totalChildCount <= 2*treeBase {
			n.childTrees = slices.Concat(smallNode.childTrees, n.childTrees)
			n.childSummaries = slices.Concat(smallNode.childSummaries, n.childSummaries)
			fullSummary := smallNode.summary
			fullSummary = fullSummary.AddSummary(n.summary, cx)
			n.summary = fullSummary
			return SumTree[T, S, C]{}, false
		}
		midpoint := (totalChildCount + 1) / 2
		allTrees := slices.Concat(smallNode.childTrees, n.childTrees)
		leftTrees := slices.Clone(allTrees[:midpoint])
		n.childTrees = slices.Clone(allTrees[midpoint:])
		allSummaries := slices.Concat(smallNode.childSummaries, n.childSummaries)
		leftSummaries := slices.Clone(allSummaries[:midpoint])
		n.childSummaries = slices.Clone(allSummaries[midpoint:])
		n.summary = sum(n.childSummaries, cx)
		return SumTree[T, S, C]{&Node[T, S, C]{
			internal:       true,
			height:         n.height,
			summary:        sum(leftSummaries, cx),
			childSummaries: leftSummaries,
			childTrees:     leftTrees,
		}}, true
	case !smallNode.internal && !n.internal:
		totalChildCount := len(smallNode.items) + len(n.items)
		if totalChildCount <= 2*treeBase {
			n.items = slices.Concat(smallNode.items, n.items)
			n.childSummaries = slices.Concat(smallNode.childSummaries, n.childSummaries)
			fullSummary := smallNode.summary
			fullSummary = fullSummary.AddSummary(n.summary, cx)
			n.summary = fullSummary
			return SumTree[T, S, C]{}, false
		}
		midpoint := (totalChildCount + 1) / 2
		allItems := slices.Concat(smallNode.items, n.items)
		leftItems := slices.Clone(allItems[:midpoint])
		n.items = slices.Clone(allItems[midpoint:])
		allSummaries := slices.Concat(smallNode.childSummaries, n.childSummaries)
		leftSummaries := slices.Clone(allSummaries[:midpoint])
		n.childSummaries = slices.Clone(allSummaries[midpoint:])
		n.summary = sum(n.childSummaries, cx)
		return SumTree[T, S, C]{&Node[T, S, C]{
			items:          leftItems,
			summary:        sum(leftSummaries, cx),
			childSummaries: leftSummaries,
		}}, true
	default:
		panic("internal error: entered unreachable code")
	}
}

func fromChildTrees[T Item[S, C], S Summary[S, C], C any](left, right SumTree[T, S, C], cx C) SumTree[T, S, C] {
	height := left.root().height + 1
	childSummaries := []S{left.root().summary, right.root().summary}
	childTrees := []SumTree[T, S, C]{left, right}
	return SumTree[T, S, C]{&Node[T, S, C]{
		internal:       true,
		height:         height,
		summary:        sum(childSummaries, cx),
		childSummaries: childSummaries,
		childTrees:     childTrees,
	}}
}

func (t SumTree[T, S, C]) leftmostLeaf() SumTree[T, S, C] {
	n := t.root()
	if !n.internal {
		return t
	}
	return n.childTrees[0].leftmostLeaf()
}

func (t SumTree[T, S, C]) rightmostLeaf() SumTree[T, S, C] {
	n := t.root()
	if !n.internal {
		return t
	}
	return n.childTrees[len(n.childTrees)-1].rightmostLeaf()
}

func keyOf[K any, T any](item T) K {
	keyed, ok := any(item).(interface{ Key() K })
	if !ok {
		panic(fmt.Sprintf("sumtree: %s is not a KeyedItem with key %s", reflect.TypeFor[T](), reflect.TypeFor[K]()))
	}
	return keyed.Key()
}

// InsertOrReplace inserts item, replacing and returning the item with the same key.
func (t *SumTree[T, S, C]) InsertOrReplace[K OrdDimension[K, S, C]](item T, cx C) (T, bool) {
	removed := insertSorted[T, S, C, K](t, []T{item}, cx)
	if len(removed) == 0 {
		var none T
		return none, false
	}
	return removed[len(removed)-1], true
}

// Remove removes and returns the item with the given key.
func (t *SumTree[T, S, C]) Remove[K OrdDimension[K, S, C]](key K, cx C) (T, bool) {
	var removed T
	found := false
	cursor := NewCursor[T, S, C, K](*t, cx)
	newTree := cursor.Slice(key, BiasLeft)
	if item, ok := cursor.Item(); ok && keyOf[K](item).Cmp(key, cx) == 0 {
		removed, found = item, true
		cursor.Next()
	}
	newTree.Append(cursor.Suffix(), cx)
	*t = newTree
	return removed, found
}

// Edit applies edits and returns the items they replaced or removed.
func (t *SumTree[T, S, C]) Edit[K OrdDimension[K, S, C]](edits []Edit[T, K], cx C) []T {
	if len(edits) == 0 {
		return nil
	}

	var removed []T
	edits = slices.Clone(edits)
	slices.SortFunc(edits, func(a, b Edit[T, K]) int { return a.key().Cmp(b.key(), cx) })

	allInserts := true
	for _, edit := range edits {
		if edit.remove {
			allInserts = false
			break
		}
	}
	if allInserts {
		sorted := true
		for i := 1; i < len(edits); i++ {
			if !(edits[i-1].key().Cmp(edits[i].key(), cx) < 0) {
				sorted = false
				break
			}
		}
		if sorted {
			items := make([]T, len(edits))
			for i, edit := range edits {
				items[i] = edit.item
			}
			return insertSorted[T, S, C, K](t, items, cx)
		}
	}

	cursor := NewCursor[T, S, C, K](*t, cx)
	newTree := NewSumTree[T](cx)
	var bufferedItems []T

	cursor.Seek(zeroDim[K](cx), BiasLeft)
	for _, edit := range edits {
		newKey := edit.key()
		oldItem, hasOld := cursor.Item()

		if hasOld && keyOf[K](oldItem).Cmp(newKey, cx) < 0 {
			newTree.Extend(bufferedItems, cx)
			bufferedItems = nil
			slice := cursor.Slice(newKey, BiasLeft)
			newTree.Append(slice, cx)
			oldItem, hasOld = cursor.Item()
		}

		if hasOld && keyOf[K](oldItem).Cmp(newKey, cx) == 0 {
			removed = append(removed, oldItem)
			cursor.Next()
		}

		if !edit.remove {
			bufferedItems = append(bufferedItems, edit.item)
		}
	}

	newTree.Extend(bufferedItems, cx)
	newTree.Append(cursor.Suffix(), cx)
	*t = newTree
	return removed
}

type keyedPair[T any, K any] struct {
	key  K
	item T
}

type peekable[T any, K any] struct {
	pairs []keyedPair[T, K]
	next  int
}

func (p *peekable[T, K]) peek() (keyedPair[T, K], bool) {
	if p.next < len(p.pairs) {
		return p.pairs[p.next], true
	}
	return keyedPair[T, K]{}, false
}

func insertSorted[T Item[S, C], S Summary[S, C], C any, K OrdDimension[K, S, C]](t *SumTree[T, S, C], items []T, cx C) []T {
	var removed []T
	pairs := make([]keyedPair[T, K], len(items))
	for i, item := range items {
		pairs[i] = keyedPair[T, K]{keyOf[K](item), item}
	}
	it := &peekable[T, K]{pairs: pairs}
	siblings := insertRecursive(t, it, zeroDim[K](cx), nil, &removed, cx)
	if len(siblings) > 0 {
		height := t.root().height
		nodes := append([]SumTree[T, S, C]{*t}, siblings...)
		for len(nodes) > 1 {
			height++
			nodes = groupChildren(nodes, height, cx)
		}
		if len(nodes) > 0 {
			*t = nodes[len(nodes)-1]
		}
	}
	return removed
}

func insertRecursive[T Item[S, C], S Summary[S, C], C any, K OrdDimension[K, S, C]](t *SumTree[T, S, C], items *peekable[T, K], position K, bound *K, removed *[]T, cx C) []SumTree[T, S, C] {
	node := t.makeMut()
	var height uint8
	var children []SumTree[T, S, C]
	if node.internal {
		type split struct {
			index    int
			siblings []SumTree[T, S, C]
		}
		var splits []split
		childCount := len(node.childTrees)
		for index := range node.childTrees {
			pair, ok := items.peek()
			if !ok {
				break
			}
			childEnd := position.AddSummary(node.childSummaries[index], cx)
			var childBound *K
			if index+1 == childCount {
				childBound = bound
			} else {
				end := childEnd
				childBound = &end
			}
			if childBound == nil || pair.key.Cmp(*childBound, cx) <= 0 {
				siblings := insertRecursive(&node.childTrees[index], items, position, childBound, removed, cx)
				node.childSummaries[index] = node.childTrees[index].Summary()
				if len(siblings) > 0 {
					splits = append(splits, split{index, siblings})
				}
			}
			position = childEnd
		}
		if len(splits) == 0 {
			node.summary = sum(node.childSummaries, cx)
			return nil
		}

		next := 0
		for index, childTree := range node.childTrees {
			children = append(children, childTree)
			if next < len(splits) && splits[next].index == index {
				children = append(children, splits[next].siblings...)
				next++
			}
		}
		node.childTrees = nil
		height = node.height
	} else {
		oldItems := node.items
		node.items = nil
		oldIndex := 0
		var oldKey K
		hasOldKey := false
		refresh := func() {
			if oldIndex < len(oldItems) {
				oldKey, hasOldKey = keyOf[K](oldItems[oldIndex]), true
			} else {
				hasOldKey = false
			}
		}
		refresh()
		var merged []T
		for {
			pair, ok := items.peek()
			if !ok || !(bound == nil || pair.key.Cmp(*bound, cx) <= 0) {
				break
			}
			items.next++
			for hasOldKey && oldKey.Cmp(pair.key, cx) < 0 {
				merged = append(merged, oldItems[oldIndex])
				oldIndex++
				refresh()
			}
			if hasOldKey && oldKey.Cmp(pair.key, cx) == 0 {
				*removed = append(*removed, oldItems[oldIndex])
				oldIndex++
				refresh()
			}
			merged = append(merged, pair.item)
		}
		merged = append(merged, oldItems[oldIndex:]...)

		var leaves []SumTree[T, S, C]
		start := 0
		for _, size := range nodeSizes(len(merged)) {
			leafItems := slices.Clone(merged[start : start+size])
			start += size
			itemSummaries := make([]S, len(leafItems))
			for i, item := range leafItems {
				itemSummaries[i] = item.Summary(cx)
			}
			leaves = append(leaves, SumTree[T, S, C]{&Node[T, S, C]{
				summary:        sum(itemSummaries, cx),
				items:          leafItems,
				childSummaries: itemSummaries,
			}})
		}
		if len(leaves) > 0 {
			*node = *leaves[0].node
			leaves = leaves[1:]
		}
		return leaves
	}

	parents := groupChildren(children, height, cx)
	if len(parents) > 0 {
		*t = parents[0]
		parents = parents[1:]
	}
	return parents
}

func groupChildren[T Item[S, C], S Summary[S, C], C any](children []SumTree[T, S, C], height uint8, cx C) []SumTree[T, S, C] {
	var parents []SumTree[T, S, C]
	start := 0
	for _, size := range nodeSizes(len(children)) {
		childTrees := slices.Clone(children[start : start+size])
		start += size
		childSummaries := make([]S, len(childTrees))
		for i, tree := range childTrees {
			childSummaries[i] = tree.Summary()
		}
		parents = append(parents, SumTree[T, S, C]{&Node[T, S, C]{
			internal:       true,
			height:         height,
			summary:        sum(childSummaries, cx),
			childSummaries: childSummaries,
			childTrees:     childTrees,
		}})
	}
	return parents
}

// Get returns the item with the given key.
func (t SumTree[T, S, C]) Get[K OrdDimension[K, S, C]](key K, cx C) (T, bool) {
	_, _, item, ok := t.FindExact[K](cx, key, BiasLeft)
	return item, ok
}

func (n *Node[T, S, C]) isLeaf() bool { return !n.internal }

func (n *Node[T, S, C]) childTreesOf() []SumTree[T, S, C] {
	if !n.internal {
		panic("Leaf nodes have no child trees")
	}
	return n.childTrees
}

func (n *Node[T, S, C]) itemsOf() []T {
	if n.internal {
		panic("Internal nodes have no items")
	}
	return n.items
}

func (n *Node[T, S, C]) isUnderflowing() bool {
	if n.internal {
		return len(n.childTrees) < treeBase
	}
	return len(n.items) < treeBase
}

// Edit is an insertion of an item or a removal by key, applied by SumTree.Edit.
type Edit[T any, K any] struct {
	remove bool
	item   T
	k      K
}

// EditInsert returns an edit inserting item.
func EditInsert[T any, K any](item T) Edit[T, K] {
	return Edit[T, K]{item: item}
}

// EditRemove returns an edit removing the item with key.
func EditRemove[T any, K any](key K) Edit[T, K] {
	return Edit[T, K]{remove: true, k: key}
}

func (e Edit[T, K]) key() K {
	if e.remove {
		return e.k
	}
	return keyOf[K](e.item)
}

func sum[S Summary[S, C], C any](summaries []S, cx C) S {
	total := zeroDim[S](cx)
	for _, value := range summaries {
		total = total.AddSummary(value, cx)
	}
	return total
}

func nodeSizes(count int) []int {
	nodeCount := (count + 2*treeBase - 1) / (2 * treeBase)
	sizes := make([]int, nodeCount)
	for index := range sizes {
		sizes[index] = count / nodeCount
		if index < count%nodeCount {
			sizes[index]++
		}
	}
	return sizes
}

func checkCap(n int) {
	if n > 2*treeBase {
		panic(capacityPanic)
	}
}

func pushCap[E any](s []E, e E) []E {
	checkCap(len(s) + 1)
	return append(s, e)
}

func extendCap[E any](s []E, more []E) []E {
	checkCap(len(s) + len(more))
	return append(s, more...)
}

func insertCap[E any](s []E, i int, e E) []E {
	checkCap(len(s) + 1)
	return slices.Insert(s, i, e)
}
