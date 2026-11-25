package taffy

import "testing"

func TestHandWrittenRootConstraints_RootWithPercentageSize(t *testing.T) {
	taffy := newTestTree()
	s := defaultStyle()
	s.Size = hwSizeDim(DimensionPercent(1.0), DimensionPercent(1.0))
	node := hwUnwrap(taffy.NewLeaf(s))

	hwOK(taffy.ComputeLayout(node, hwDefinite(100.0, 200.0)))
	layout := hwLayout(taffy, node)

	hwAssertEq(t, layout.Size.Width, 100.0)
	hwAssertEq(t, layout.Size.Height, 200.0)
}

func TestHandWrittenRootConstraints_RootWithNoSize(t *testing.T) {
	taffy := newTestTree()
	node := hwUnwrap(taffy.NewLeaf(defaultStyle()))

	hwOK(taffy.ComputeLayout(node, hwDefinite(100.0, 100.0)))
	layout := hwLayout(taffy, node)

	hwAssertEq(t, layout.Size.Width, 0.0)
	hwAssertEq(t, layout.Size.Height, 0.0)
}

func TestHandWrittenRootConstraints_RootWithLargerSize(t *testing.T) {
	taffy := newTestTree()
	s := defaultStyle()
	s.Size = hwSizeDim(hwDim(200.0), hwDim(200.0))
	node := hwUnwrap(taffy.NewLeaf(s))

	hwOK(taffy.ComputeLayout(node, hwDefinite(100.0, 100.0)))
	layout := hwLayout(taffy, node)

	hwAssertEq(t, layout.Size.Width, 200.0)
	hwAssertEq(t, layout.Size.Height, 200.0)
}

func TestHandWrittenRootConstraints_RootPaddingAndBorderLargerThanDefiniteSize(t *testing.T) {
	tree := TaffyTreeWithCapacity[struct{}](16)

	child := hwUnwrap(tree.NewLeaf(defaultStyle()))

	rs := defaultStyle()
	rs.Size = hwSizeDim(hwDim(10.0), hwDim(10.0))
	ten := LengthPercentageLength(10.0)
	rs.Padding = Rect[LengthPercentage]{Left: ten, Right: ten, Top: ten, Bottom: ten}
	rs.Border = Rect[LengthPercentage]{Left: ten, Right: ten, Top: ten, Bottom: ten}
	root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{child}))

	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	layout := hwLayout(tree, root)

	hwAssertEq(t, layout.Size.Width, 40.0)
	hwAssertEq(t, layout.Size.Height, 40.0)
}
