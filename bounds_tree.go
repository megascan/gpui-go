package gpui

const boundsTreeMaxChildren = 12

type boundsTree[U Number] struct {
	nodes       []boundsTreeNode[U]
	root        Option[int]
	maxLeaf     Option[int]
	insertPath  []int
	searchStack []int
}

type boundsTreeNode[U Number] struct {
	bounds Bounds[U]
	maxKey boundsTreeKey
	kind   boundsTreeNodeKind
}

type boundsTreeKey struct {
	ordering uint32
	kind     primitiveKind
}

func (k boundsTreeKey) cmp(other boundsTreeKey) int {
	switch {
	case k.ordering < other.ordering:
		return -1
	case k.ordering > other.ordering:
		return 1
	case k.kind < other.kind:
		return -1
	case k.kind > other.kind:
		return 1
	}
	return 0
}

func (k boundsTreeKey) less(other boundsTreeKey) bool { return k.cmp(other) < 0 }

func maxBoundsTreeKey(a, b boundsTreeKey) boundsTreeKey {
	if b.cmp(a) >= 0 {
		return b
	}
	return a
}

type boundsTreeNodeKind struct {
	leaf     bool
	key      boundsTreeKey
	children boundsTreeNodeChildren
}

type boundsTreeNodeChildren struct {
	indices [boundsTreeMaxChildren]int
	len     uint8
}

func (c *boundsTreeNodeChildren) push(index int) {
	c.indices[c.len] = index
	c.len++
}

func (c *boundsTreeNodeChildren) length() int { return int(c.len) }

func (c *boundsTreeNodeChildren) asSlice() []int { return c.indices[:c.len] }

func (t *boundsTree[U]) clear() {
	t.nodes = t.nodes[:0]
	t.root = None[int]()
	t.maxLeaf = None[int]()
	t.insertPath = t.insertPath[:0]
	t.searchStack = t.searchStack[:0]
}

func (t *boundsTree[U]) insert(newBounds Bounds[U], kind primitiveKind, maxSharedKind primitiveKind) uint32 {
	var ordering uint32
	maxKey, found := t.findMaxKey(newBounds)
	switch {
	case !found:
		ordering = 1
	case maxKey.kind <= maxSharedKind:
		ordering = maxKey.ordering
	default:
		ordering = maxKey.ordering + 1
	}
	key := boundsTreeKey{ordering: ordering, kind: kind}

	newLeafIdx := t.insertLeaf(newBounds, key)

	if oldIdx, ok := t.maxLeaf.Get(); !ok {
		t.maxLeaf = Some(newLeafIdx)
	} else if t.nodes[oldIdx].maxKey.less(key) {
		t.maxLeaf = Some(newLeafIdx)
	}

	return ordering
}

func (t *boundsTree[U]) findMaxKey(query Bounds[U]) (boundsTreeKey, bool) {
	rootIdx, ok := t.root.Get()
	if !ok {
		return boundsTreeKey{}, false
	}

	if maxIdx, ok := t.maxLeaf.Get(); ok {
		maxNode := &t.nodes[maxIdx]
		if query.Intersects(maxNode.bounds) {
			return maxNode.maxKey, true
		}
	}

	t.searchStack = append(t.searchStack[:0], rootIdx)

	var maxFound boundsTreeKey
	hasMax := false

	for len(t.searchStack) > 0 {
		nodeIdx := t.searchStack[len(t.searchStack)-1]
		t.searchStack = t.searchStack[:len(t.searchStack)-1]
		node := &t.nodes[nodeIdx]

		if hasMax && node.maxKey.cmp(maxFound) <= 0 {
			continue
		}

		if !query.Intersects(node.bounds) {
			continue
		}

		if node.kind.leaf {
			if !hasMax || node.kind.key.cmp(maxFound) > 0 {
				maxFound = node.kind.key
				hasMax = true
			}
		} else {
			for _, childIdx := range node.kind.children.asSlice() {
				if !hasMax || t.nodes[childIdx].maxKey.cmp(maxFound) > 0 {
					t.searchStack = append(t.searchStack, childIdx)
				}
			}
		}
	}

	return maxFound, hasMax
}

func (t *boundsTree[U]) insertLeaf(bounds Bounds[U], key boundsTreeKey) int {
	newLeafIdx := len(t.nodes)
	t.nodes = append(t.nodes, boundsTreeNode[U]{
		bounds: bounds,
		maxKey: key,
		kind:   boundsTreeNodeKind{leaf: true, key: key},
	})

	rootIdx, ok := t.root.Get()
	if !ok {
		t.root = Some(newLeafIdx)
		return newLeafIdx
	}

	if t.nodes[rootIdx].kind.leaf {
		rootBounds := t.nodes[rootIdx].bounds
		rootKey := t.nodes[rootIdx].maxKey

		var children boundsTreeNodeChildren
		if key.cmp(rootKey) > 0 {
			children.push(rootIdx)
			children.push(newLeafIdx)
		} else {
			children.push(newLeafIdx)
			children.push(rootIdx)
		}

		newRootIdx := len(t.nodes)
		t.nodes = append(t.nodes, boundsTreeNode[U]{
			bounds: rootBounds.Union(bounds),
			maxKey: maxBoundsTreeKey(rootKey, key),
			kind:   boundsTreeNodeKind{children: children},
		})
		t.root = Some(newRootIdx)
		return newLeafIdx
	}

	t.insertPath = t.insertPath[:0]
	currentIdx := rootIdx

	for {
		current := &t.nodes[currentIdx]
		if current.kind.leaf {
			panic("internal error: entered unreachable code: Should only traverse internal nodes")
		}
		children := current.kind.children.asSlice()

		t.insertPath = append(t.insertPath, currentIdx)

		bestChildIdx := children[0]
		bestChildPos := 0
		bestCost := bounds.Union(t.nodes[bestChildIdx].bounds).HalfPerimeter()

		for pos := 1; pos < len(children); pos++ {
			childIdx := children[pos]
			cost := bounds.Union(t.nodes[childIdx].bounds).HalfPerimeter()
			if cost < bestCost {
				bestCost = cost
				bestChildIdx = childIdx
				bestChildPos = pos
			}
		}

		if t.nodes[bestChildIdx].kind.leaf {
			if current.kind.children.length() < boundsTreeMaxChildren {
				node := &t.nodes[currentIdx]
				c := &node.kind.children
				c.push(newLeafIdx)
				if key.cmp(node.maxKey) <= 0 {
					last := c.length() - 1
					c.indices[last-1], c.indices[last] = c.indices[last], c.indices[last-1]
				}

				node.bounds = node.bounds.Union(bounds)
				node.maxKey = maxBoundsTreeKey(node.maxKey, key)
				break
			}
			siblingBounds := t.nodes[bestChildIdx].bounds
			siblingKey := t.nodes[bestChildIdx].maxKey

			var newChildren boundsTreeNodeChildren
			if key.cmp(siblingKey) > 0 {
				newChildren.push(bestChildIdx)
				newChildren.push(newLeafIdx)
			} else {
				newChildren.push(newLeafIdx)
				newChildren.push(bestChildIdx)
			}

			newInternalIdx := len(t.nodes)
			newInternalMaxKey := maxBoundsTreeKey(siblingKey, key)
			t.nodes = append(t.nodes, boundsTreeNode[U]{
				bounds: siblingBounds.Union(bounds),
				maxKey: newInternalMaxKey,
				kind:   boundsTreeNodeKind{children: newChildren},
			})

			parent := &t.nodes[currentIdx]
			c := &parent.kind.children
			childrenLen := c.length()

			c.indices[bestChildPos] = newInternalIdx

			if newInternalMaxKey.cmp(parent.maxKey) > 0 {
				c.indices[bestChildPos], c.indices[childrenLen-1] = c.indices[childrenLen-1], c.indices[bestChildPos]
			}
			break
		}
		currentIdx = bestChildIdx
	}

	updatedChildIdx := None[int]()
	for i := len(t.insertPath) - 1; i >= 0; i-- {
		nodeIdx := t.insertPath[i]
		node := &t.nodes[nodeIdx]
		node.bounds = node.bounds.Union(bounds)

		if node.maxKey.less(key) {
			node.maxKey = key

			if childIdx, ok := updatedChildIdx.Get(); ok && !node.kind.leaf {
				c := &node.kind.children
				for pos, ch := range c.asSlice() {
					if ch == childIdx {
						last := c.length() - 1
						if pos != last {
							c.indices[pos], c.indices[last] = c.indices[last], c.indices[pos]
						}
						break
					}
				}
			}
		}

		updatedChildIdx = Some(nodeIdx)
	}

	return newLeafIdx
}
