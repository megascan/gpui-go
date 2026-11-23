package sumtree

import "iter"

const stackCapacity = 16

type stackEntry[T Item[S, C], S Summary[S, C], C any, D any] struct {
	tree     SumTree[T, S, C]
	index    uint32
	position D
}

// Cursor walks a SumTree, tracking its position in dimension D.
type Cursor[T Item[S, C], S Summary[S, C], C any, D Dimension[D, S, C]] struct {
	tree     SumTree[T, S, C]
	stack    []stackEntry[T, S, C, D]
	Position D
	didSeek  bool
	atEnd    bool
	cx       C
}

type iterEntry[T Item[S, C], S Summary[S, C], C any] struct {
	tree  SumTree[T, S, C]
	index uint32
}

// Iter iterates the items of a SumTree in order.
type Iter[T Item[S, C], S Summary[S, C], C any] struct {
	tree  SumTree[T, S, C]
	stack []iterEntry[T, S, C]
}

func pushStack[E any](stack []E, e E) []E {
	if len(stack) >= stackCapacity {
		panic(capacityPanic)
	}
	return append(stack, e)
}

// NewCursor returns a cursor over tree that has not yet seeked.
func NewCursor[T Item[S, C], S Summary[S, C], C any, D Dimension[D, S, C]](tree SumTree[T, S, C], cx C) *Cursor[T, S, C, D] {
	if tree.node == nil {
		tree = NewSumTree[T](cx)
	}
	return &Cursor[T, S, C, D]{
		tree:     tree,
		Position: zeroDim[D](cx),
		atEnd:    tree.IsEmpty(),
		cx:       cx,
	}
}

// Clone returns an independent copy of the cursor.
func (c *Cursor[T, S, C, D]) Clone() *Cursor[T, S, C, D] {
	clone := *c
	clone.stack = append([]stackEntry[T, S, C, D](nil), c.stack...)
	return &clone
}

// Reset returns the cursor to its unseeked state.
func (c *Cursor[T, S, C, D]) Reset() {
	c.didSeek = false
	c.atEnd = c.tree.IsEmpty()
	c.stack = c.stack[:0]
	c.Position = zeroDim[D](c.cx)
}

// Start returns the position at the start of the current item.
func (c *Cursor[T, S, C, D]) Start() D {
	return c.Position
}

// End returns the position at the end of the current item.
func (c *Cursor[T, S, C, D]) End() D {
	if itemSummary, ok := c.ItemSummary(); ok {
		return c.Start().AddSummary(itemSummary, c.cx)
	}
	return c.Start()
}

// Item returns the current item; ok is false when the tree is empty or the cursor is at the end.
func (c *Cursor[T, S, C, D]) Item() (T, bool) {
	c.assertDidSeek()
	var none T
	if len(c.stack) == 0 {
		return none, false
	}
	entry := c.stack[len(c.stack)-1]
	node := entry.tree.node
	if node.internal {
		panic("internal error: entered unreachable code")
	}
	if int(entry.index) == len(node.items) {
		return none, false
	}
	return node.items[entry.index], true
}

// ItemSummary returns the summary of the current item.
func (c *Cursor[T, S, C, D]) ItemSummary() (S, bool) {
	c.assertDidSeek()
	var none S
	if len(c.stack) == 0 {
		return none, false
	}
	entry := c.stack[len(c.stack)-1]
	node := entry.tree.node
	if node.internal {
		panic("internal error: entered unreachable code")
	}
	if int(entry.index) == len(node.childSummaries) {
		return none, false
	}
	return node.childSummaries[entry.index], true
}

// NextItem returns the item after the current one.
func (c *Cursor[T, S, C, D]) NextItem() (T, bool) {
	c.assertDidSeek()
	var none T
	if len(c.stack) > 0 {
		entry := c.stack[len(c.stack)-1]
		items := entry.tree.node.itemsOf()
		if int(entry.index) == len(items)-1 {
			if nextLeaf, ok := c.nextLeaf(); ok {
				return nextLeaf.node.itemsOf()[0], true
			}
			return none, false
		}
		return items[entry.index+1], true
	} else if c.atEnd {
		return none, false
	}
	return c.tree.First()
}

func (c *Cursor[T, S, C, D]) nextLeaf() (SumTree[T, S, C], bool) {
	for i := len(c.stack) - 2; i >= 0; i-- {
		entry := c.stack[i]
		childTrees := entry.tree.node.childTreesOf()
		if int(entry.index) < len(childTrees)-1 {
			return childTrees[entry.index+1].leftmostLeaf(), true
		}
	}
	return SumTree[T, S, C]{}, false
}

// PrevItem returns the item before the current one.
func (c *Cursor[T, S, C, D]) PrevItem() (T, bool) {
	c.assertDidSeek()
	var none T
	if len(c.stack) > 0 {
		entry := c.stack[len(c.stack)-1]
		if entry.index == 0 {
			if prevLeaf, ok := c.prevLeaf(); ok {
				items := prevLeaf.node.itemsOf()
				return items[len(items)-1], true
			}
			return none, false
		}
		return entry.tree.node.itemsOf()[entry.index-1], true
	} else if c.atEnd {
		return c.tree.Last()
	}
	return none, false
}

func (c *Cursor[T, S, C, D]) prevLeaf() (SumTree[T, S, C], bool) {
	for i := len(c.stack) - 2; i >= 0; i-- {
		entry := c.stack[i]
		if entry.index != 0 {
			childTrees := entry.tree.node.childTreesOf()
			return childTrees[entry.index-1].rightmostLeaf(), true
		}
	}
	return SumTree[T, S, C]{}, false
}

// Prev moves to the previous item.
func (c *Cursor[T, S, C, D]) Prev() {
	c.SearchBackward(func(S) bool { return true })
}

// SearchBackward moves to the previous item whose subtree summaries satisfy filterNode.
func (c *Cursor[T, S, C, D]) SearchBackward(filterNode func(S) bool) {
	if !c.didSeek {
		c.didSeek = true
		c.atEnd = true
	}

	if c.atEnd {
		c.Position = zeroDim[D](c.cx)
		c.atEnd = c.tree.IsEmpty()
		if !c.tree.IsEmpty() {
			c.stack = pushStack(c.stack, stackEntry[T, S, C, D]{
				tree:     c.tree,
				index:    uint32(len(c.tree.node.childSummaries)),
				position: FromSummary[D](c.tree.Summary(), c.cx),
			})
		}
	}

	descending := false
	for len(c.stack) > 0 {
		if len(c.stack) >= 2 {
			c.Position = c.stack[len(c.stack)-2].position
		} else {
			c.Position = zeroDim[D](c.cx)
		}

		entry := &c.stack[len(c.stack)-1]
		if !descending {
			if entry.index == 0 {
				c.stack = c.stack[:len(c.stack)-1]
				continue
			}
			entry.index--
		}

		node := entry.tree.node
		for _, summary := range node.childSummaries[:entry.index] {
			c.Position = c.Position.AddSummary(summary, c.cx)
		}
		entry.position = c.Position

		descending = filterNode(node.childSummaries[entry.index])
		if node.internal {
			if descending {
				tree := node.childTrees[entry.index]
				c.stack = pushStack(c.stack, stackEntry[T, S, C, D]{
					position: zeroDim[D](c.cx),
					tree:     tree,
					index:    uint32(len(tree.node.childSummaries)) - 1,
				})
			}
		} else if descending {
			break
		}
	}
}

// Next moves to the next item.
func (c *Cursor[T, S, C, D]) Next() {
	c.SearchForward(func(S) bool { return true })
}

// SearchForward moves to the next item whose subtree summaries satisfy filterNode.
func (c *Cursor[T, S, C, D]) SearchForward(filterNode func(S) bool) {
	descend := false

	if len(c.stack) == 0 {
		if !c.atEnd {
			c.stack = pushStack(c.stack, stackEntry[T, S, C, D]{
				tree:     c.tree,
				index:    0,
				position: zeroDim[D](c.cx),
			})
			descend = true
		}
		c.didSeek = true
	}

	for len(c.stack) > 0 {
		entry := &c.stack[len(c.stack)-1]
		node := entry.tree.node
		var newSubtree SumTree[T, S, C]
		hasSubtree := false
		if node.internal {
			if !descend {
				entry.index++
				entry.position = c.Position
			}

			for int(entry.index) < len(node.childSummaries) {
				nextSummary := node.childSummaries[entry.index]
				if filterNode(nextSummary) {
					break
				}
				entry.index++
				entry.position = entry.position.AddSummary(nextSummary, c.cx)
				c.Position = c.Position.AddSummary(nextSummary, c.cx)
			}

			if int(entry.index) < len(node.childTrees) {
				newSubtree, hasSubtree = node.childTrees[entry.index], true
			}
		} else {
			if !descend {
				itemSummary := node.childSummaries[entry.index]
				entry.index++
				entry.position = entry.position.AddSummary(itemSummary, c.cx)
				c.Position = c.Position.AddSummary(itemSummary, c.cx)
			}

			for int(entry.index) < len(node.childSummaries) {
				nextItemSummary := node.childSummaries[entry.index]
				if filterNode(nextItemSummary) {
					return
				}
				entry.index++
				entry.position = entry.position.AddSummary(nextItemSummary, c.cx)
				c.Position = c.Position.AddSummary(nextItemSummary, c.cx)
			}
		}

		if hasSubtree {
			descend = true
			c.stack = pushStack(c.stack, stackEntry[T, S, C, D]{
				tree:     newSubtree,
				index:    0,
				position: c.Position,
			})
		} else {
			descend = false
			c.stack = c.stack[:len(c.stack)-1]
		}
	}

	c.atEnd = len(c.stack) == 0
}

func (c *Cursor[T, S, C, D]) assertDidSeek() {
	if !c.didSeek {
		panic("Must call `seek`, `next` or `prev` before calling this method")
	}
}

// DidSeek reports whether the cursor has been positioned.
func (c *Cursor[T, S, C, D]) DidSeek() bool {
	return c.didSeek
}

// Seek resets the cursor and moves it to target; it reports whether target was found exactly.
func (c *Cursor[T, S, C, D]) Seek(pos SeekTarget[S, D, C], bias Bias) bool {
	c.Reset()
	return c.seekInternal(pos, bias, noAggregate[T, S, C]{})
}

// SeekForward moves the cursor forward to target; it reports whether target was found exactly.
func (c *Cursor[T, S, C, D]) SeekForward(pos SeekTarget[S, D, C], bias Bias) bool {
	return c.seekInternal(pos, bias, noAggregate[T, S, C]{})
}

// Slice advances the cursor to end and returns the traversed items as a tree.
func (c *Cursor[T, S, C, D]) Slice(end SeekTarget[S, D, C], bias Bias) SumTree[T, S, C] {
	slice := &sliceSeekAggregate[T, S, C]{
		tree:        NewSumTree[T](c.cx),
		leafSummary: zeroDim[S](c.cx),
	}
	c.seekInternal(end, bias, slice)
	return slice.tree
}

// Suffix advances the cursor to the end and returns the traversed items as a tree.
func (c *Cursor[T, S, C, D]) Suffix() SumTree[T, S, C] {
	return c.Slice(endTarget[S, D, C]{}, BiasRight)
}

// Summary advances the cursor to end and returns the traversed extent in dimension Output.
func (c *Cursor[T, S, C, D]) Summary[Output Dimension[Output, S, C]](end SeekTarget[S, D, C], bias Bias) Output {
	summary := &summarySeekAggregate[T, S, C, Output]{zeroDim[Output](c.cx)}
	c.seekInternal(end, bias, summary)
	return summary.value
}

func (c *Cursor[T, S, C, D]) seekInternal(target SeekTarget[S, D, C], bias Bias, aggregate seekAggregate[T, S, C]) bool {
	if target.Cmp(c.Position, c.cx) < 0 {
		panic("cannot seek backward")
	}

	if !c.didSeek {
		c.didSeek = true
		c.stack = pushStack(c.stack, stackEntry[T, S, C, D]{
			tree:     c.tree,
			index:    0,
			position: zeroDim[D](c.cx),
		})
	}

	ascending := false
outer:
	for len(c.stack) > 0 {
		entry := &c.stack[len(c.stack)-1]
		node := entry.tree.node
		if node.internal {
			if ascending {
				entry.index++
				entry.position = c.Position
			}

			for i := int(entry.index); i < len(node.childTrees); i++ {
				childTree := node.childTrees[i]
				childSummary := node.childSummaries[i]
				childEnd := c.Position.AddSummary(childSummary, c.cx)

				comparison := target.Cmp(childEnd, c.cx)
				if comparison > 0 || (comparison == 0 && bias == BiasRight) {
					c.Position = childEnd
					aggregate.pushTree(childTree, childSummary, c.cx)
					entry.index++
					entry.position = c.Position
				} else {
					c.stack = pushStack(c.stack, stackEntry[T, S, C, D]{
						tree:     childTree,
						index:    0,
						position: c.Position,
					})
					ascending = false
					continue outer
				}
			}
		} else {
			aggregate.beginLeaf()

			for i := int(entry.index); i < len(node.items); i++ {
				item := node.items[i]
				itemSummary := node.childSummaries[i]
				childEnd := c.Position.AddSummary(itemSummary, c.cx)

				comparison := target.Cmp(childEnd, c.cx)
				if comparison > 0 || (comparison == 0 && bias == BiasRight) {
					c.Position = childEnd
					aggregate.pushItem(item, itemSummary, c.cx)
					entry.index++
				} else {
					aggregate.endLeaf(c.cx)
					break outer
				}
			}

			aggregate.endLeaf(c.cx)
		}

		c.stack = c.stack[:len(c.stack)-1]
		ascending = true
	}

	c.atEnd = len(c.stack) == 0

	end := c.Position
	if bias == BiasLeft {
		if summary, ok := c.ItemSummary(); ok {
			end = end.AddSummary(summary, c.cx)
		}
	}

	return target.Cmp(end, c.cx) == 0
}

// All iterates the remaining items, advancing the cursor, as upstream's Iterator impl.
func (c *Cursor[T, S, C, D]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			if !c.didSeek {
				c.Next()
			}
			item, ok := c.Item()
			if !ok {
				return
			}
			c.Next()
			if !yield(item) {
				return
			}
		}
	}
}

func newIter[T Item[S, C], S Summary[S, C], C any](tree SumTree[T, S, C]) *Iter[T, S, C] {
	if tree.node == nil {
		tree = SumTree[T, S, C]{tree.root()}
	}
	return &Iter[T, S, C]{tree: tree}
}

// Next returns the next item.
func (it *Iter[T, S, C]) Next() (T, bool) {
	descend := false

	if len(it.stack) == 0 {
		it.stack = pushStack(it.stack, iterEntry[T, S, C]{tree: it.tree, index: 0})
		descend = true
	}

	for len(it.stack) > 0 {
		entry := &it.stack[len(it.stack)-1]
		node := entry.tree.node
		var newSubtree SumTree[T, S, C]
		hasSubtree := false
		if node.internal {
			if !descend {
				entry.index++
			}
			if int(entry.index) < len(node.childTrees) {
				newSubtree, hasSubtree = node.childTrees[entry.index], true
			}
		} else {
			if !descend {
				entry.index++
			}
			if int(entry.index) < len(node.items) {
				return node.items[entry.index], true
			}
		}

		if hasSubtree {
			descend = true
			it.stack = pushStack(it.stack, iterEntry[T, S, C]{tree: newSubtree, index: 0})
		} else {
			descend = false
			it.stack = it.stack[:len(it.stack)-1]
		}
	}

	var none T
	return none, false
}

// Last returns the last item of the tree, consuming the iterator.
func (it *Iter[T, S, C]) Last() (T, bool) {
	it.stack = nil
	return it.tree.rightmostLeaf().Last()
}

// SizeHint returns a lower bound on the number of remaining items.
func (it *Iter[T, S, C]) SizeHint() int {
	if len(it.stack) > 0 {
		top := it.stack[len(it.stack)-1]
		return len(top.tree.node.childSummaries) - int(top.index)
	}
	return len(it.tree.root().childSummaries)
}

// FilterCursor is a Cursor that skips subtrees whose summary fails a filter.
type FilterCursor[T Item[S, C], S Summary[S, C], C any, D Dimension[D, S, C]] struct {
	cursor     *Cursor[T, S, C, D]
	filterNode func(S) bool
}

// NewFilterCursor returns a filter cursor over tree.
func NewFilterCursor[T Item[S, C], S Summary[S, C], C any, D Dimension[D, S, C]](tree SumTree[T, S, C], cx C, filterNode func(S) bool) *FilterCursor[T, S, C, D] {
	return &FilterCursor[T, S, C, D]{cursor: NewCursor[T, S, C, D](tree, cx), filterNode: filterNode}
}

// Start returns the position at the start of the current item.
func (f *FilterCursor[T, S, C, D]) Start() D { return f.cursor.Start() }

// End returns the position at the end of the current item.
func (f *FilterCursor[T, S, C, D]) End() D { return f.cursor.End() }

// Item returns the current item.
func (f *FilterCursor[T, S, C, D]) Item() (T, bool) { return f.cursor.Item() }

// ItemSummary returns the summary of the current item.
func (f *FilterCursor[T, S, C, D]) ItemSummary() (S, bool) { return f.cursor.ItemSummary() }

// Next moves to the next matching item.
func (f *FilterCursor[T, S, C, D]) Next() { f.cursor.SearchForward(f.filterNode) }

// Prev moves to the previous matching item.
func (f *FilterCursor[T, S, C, D]) Prev() { f.cursor.SearchBackward(f.filterNode) }

// All iterates the remaining matching items, advancing the cursor.
func (f *FilterCursor[T, S, C, D]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			if !f.cursor.didSeek {
				f.Next()
			}
			item, ok := f.Item()
			if !ok {
				return
			}
			f.cursor.SearchForward(f.filterNode)
			if !yield(item) {
				return
			}
		}
	}
}

type seekAggregate[T Item[S, C], S Summary[S, C], C any] interface {
	beginLeaf()
	endLeaf(cx C)
	pushItem(item T, summary S, cx C)
	pushTree(tree SumTree[T, S, C], summary S, cx C)
}

type noAggregate[T Item[S, C], S Summary[S, C], C any] struct{}

func (noAggregate[T, S, C]) beginLeaf()                      {}
func (noAggregate[T, S, C]) endLeaf(C)                       {}
func (noAggregate[T, S, C]) pushItem(T, S, C)                {}
func (noAggregate[T, S, C]) pushTree(SumTree[T, S, C], S, C) {}

type sliceSeekAggregate[T Item[S, C], S Summary[S, C], C any] struct {
	tree              SumTree[T, S, C]
	leafItems         []T
	leafItemSummaries []S
	leafSummary       S
}

func (a *sliceSeekAggregate[T, S, C]) beginLeaf() {}

func (a *sliceSeekAggregate[T, S, C]) endLeaf(cx C) {
	summary := a.leafSummary
	a.leafSummary = zeroDim[S](cx)
	items := a.leafItems
	a.leafItems = nil
	itemSummaries := a.leafItemSummaries
	a.leafItemSummaries = nil
	a.tree.Append(SumTree[T, S, C]{&Node[T, S, C]{
		summary:        summary,
		items:          items,
		childSummaries: itemSummaries,
	}}, cx)
}

func (a *sliceSeekAggregate[T, S, C]) pushItem(item T, summary S, cx C) {
	a.leafItems = pushCap(a.leafItems, item)
	a.leafItemSummaries = pushCap(a.leafItemSummaries, summary)
	a.leafSummary = a.leafSummary.AddSummary(summary, cx)
}

func (a *sliceSeekAggregate[T, S, C]) pushTree(tree SumTree[T, S, C], _ S, cx C) {
	a.tree.Append(tree, cx)
}

type summarySeekAggregate[T Item[S, C], S Summary[S, C], C any, D Dimension[D, S, C]] struct {
	value D
}

func (a *summarySeekAggregate[T, S, C, D]) beginLeaf() {}
func (a *summarySeekAggregate[T, S, C, D]) endLeaf(C)  {}

func (a *summarySeekAggregate[T, S, C, D]) pushItem(_ T, summary S, cx C) {
	a.value = a.value.AddSummary(summary, cx)
}

func (a *summarySeekAggregate[T, S, C, D]) pushTree(_ SumTree[T, S, C], summary S, cx C) {
	a.value = a.value.AddSummary(summary, cx)
}

type endTarget[S any, D any, C any] struct{}

func (endTarget[S, D, C]) Cmp(D, C) int { return 1 }
