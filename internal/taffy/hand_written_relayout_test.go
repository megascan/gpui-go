package taffy

import (
	"slices"
	"testing"
)

func relayoutSized(display Display, w, h float32) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = display
	s.Size = hwSizeDim(hwDim(w), hwDim(h))
	return s
}

func relayoutAssertZeroLayout(t *testing.T, layout Layout) {
	t.Helper()
	hwAssertEq(t, layout.Location.X, 0.0)
	hwAssertEq(t, layout.Location.Y, 0.0)
	hwAssertEq(t, layout.Size.Width, 0.0)
	hwAssertEq(t, layout.Size.Height, 0.0)
}

func relayoutAssertHundredLayout(t *testing.T, layout Layout) {
	t.Helper()
	hwAssertEq(t, layout.Location.X, 0.0)
	hwAssertEq(t, layout.Location.Y, 0.0)
	hwAssertEq(t, layout.Size.Width, 100.0)
	hwAssertEq(t, layout.Size.Height, 100.0)
}

func TestHandWrittenRelayout_Relayout(t *testing.T) {
	taffy := newTestTree()
	s1 := defaultStyle()
	s1.Size = hwSizeDim(hwDim(8.0), hwDim(80.0))
	node1 := hwUnwrap(taffy.NewLeaf(s1))
	s0 := defaultStyle()
	s0.AlignSelf = some(AlignItemsCenter)
	s0.Size = hwSizeDim(DimensionAuto(), DimensionAuto())
	node0 := hwUnwrap(taffy.NewWithChildren(s0, []NodeID{node1}))
	s := defaultStyle()
	s.Size = hwSizeDim(DimensionPercent(1), DimensionPercent(1))
	node := hwUnwrap(taffy.NewWithChildren(s, []NodeID{node0}))
	hwOK(taffy.ComputeLayout(node, hwDefinite(100, 100)))
	initial := hwLayout(taffy, node).Location
	initial0 := hwLayout(taffy, node0).Location
	initial1 := hwLayout(taffy, node1).Location
	for range 9 {
		hwOK(taffy.ComputeLayout(node, hwDefinite(100, 100)))
		hwAssertEq(t, hwLayout(taffy, node).Location, initial)
		hwAssertEq(t, hwLayout(taffy, node0).Location, initial0)
		hwAssertEq(t, hwLayout(taffy, node1).Location, initial1)
	}
}

func TestHandWrittenRelayout_ToggleRootDisplayNone(t *testing.T) {
	hiddenStyle := relayoutSized(DisplayNone, 100.0, 100.0)
	flexStyle := relayoutSized(DisplayFlex, 100.0, 100.0)

	taffy := newTestTree()
	node := hwUnwrap(taffy.NewLeaf(hiddenStyle))

	hwOK(taffy.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
	relayoutAssertZeroLayout(t, hwLayout(taffy, node))

	hwOK(taffy.SetStyle(node, flexStyle))
	hwOK(taffy.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
	relayoutAssertHundredLayout(t, hwLayout(taffy, node))

	hwOK(taffy.SetStyle(node, hiddenStyle))
	hwOK(taffy.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
	relayoutAssertZeroLayout(t, hwLayout(taffy, node))
}

func TestHandWrittenRelayout_ToggleRootDisplayNoneWithChildren(t *testing.T) {
	taffy := newTestTree()

	cs := defaultStyle()
	cs.Size = hwSizeDim(hwDim(800.0), hwDim(100.0))
	child := hwUnwrap(taffy.NewLeaf(cs))

	parent := hwUnwrap(taffy.NewWithChildren(cs, []NodeID{child}))

	root := hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{parent}))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 800.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)

	ns := defaultStyle()
	ns.Display = DisplayNone
	hwOK(taffy.SetStyle(root, ns))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 0.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 0.0)

	hwOK(taffy.SetStyle(root, defaultStyle()))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(taffy, parent).Size.Width, 800.0)
	hwAssertEq(t, hwLayout(taffy, parent).Size.Height, 100.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 800.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)
}

func relayoutToggleChild(t *testing.T, display Display) {
	t.Helper()
	hiddenStyle := relayoutSized(DisplayNone, 100.0, 100.0)
	shownStyle := relayoutSized(display, 100.0, 100.0)

	taffy := newTestTree()
	node := hwUnwrap(taffy.NewLeaf(hiddenStyle))
	root := hwUnwrap(taffy.NewWithChildren(shownStyle, []NodeID{node}))

	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	relayoutAssertZeroLayout(t, hwLayout(taffy, node))

	hwOK(taffy.SetStyle(node, shownStyle))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	relayoutAssertHundredLayout(t, hwLayout(taffy, node))

	hwOK(taffy.SetStyle(node, hiddenStyle))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	relayoutAssertZeroLayout(t, hwLayout(taffy, node))
}

func relayoutToggleContainer(t *testing.T, display Display) {
	t.Helper()
	hiddenStyle := relayoutSized(DisplayNone, 100.0, 100.0)
	shownStyle := relayoutSized(display, 100.0, 100.0)

	taffy := newTestTree()
	node := hwUnwrap(taffy.NewLeaf(hiddenStyle))
	root := hwUnwrap(taffy.NewWithChildren(hiddenStyle, []NodeID{node}))

	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	relayoutAssertZeroLayout(t, hwLayout(taffy, root))

	hwOK(taffy.SetStyle(root, shownStyle))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	relayoutAssertHundredLayout(t, hwLayout(taffy, root))

	hwOK(taffy.SetStyle(root, hiddenStyle))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	relayoutAssertZeroLayout(t, hwLayout(taffy, root))
}

func TestHandWrittenRelayout_ToggleFlexChildDisplayNone(t *testing.T) {
	relayoutToggleChild(t, DisplayFlex)
}

func TestHandWrittenRelayout_ToggleFlexContainerDisplayNone(t *testing.T) {
	relayoutToggleContainer(t, DisplayFlex)
}

func TestHandWrittenRelayout_ToggleGridChildDisplayNone(t *testing.T) {
	relayoutToggleChild(t, DisplayGrid)
}

func TestHandWrittenRelayout_ToggleGridContainerDisplayNone(t *testing.T) {
	relayoutToggleContainer(t, DisplayGrid)
}

func TestHandWrittenRelayout_RelayoutIsStableWithRounding(t *testing.T) {
	taffy := newTestTree()
	taffy.EnableRounding()

	is := defaultStyle()
	is.MinSize = Size[LengthPercentageAuto]{Width: hwLPA(300.), Height: LengthPercentageAutoAuto()}
	inner := hwUnwrap(taffy.NewLeaf(is))
	ws := defaultStyle()
	ws.Size = hwSizeDim(hwDim(150.), DimensionAuto())
	ws.JustifyContent = AlignContentEnd
	wrapper := hwUnwrap(taffy.NewWithChildren(ws, []NodeID{inner}))
	os := defaultStyle()
	os.Size = hwSizeDim(DimensionPercent(1.), DimensionAuto())
	os.Position = PositionRelative
	os.Inset = Rect[LengthPercentageAuto]{Left: hwLPA(1.5), Right: LengthPercentageAutoAuto(), Top: LengthPercentageAutoAuto(), Bottom: LengthPercentageAutoAuto()}
	outer := hwUnwrap(taffy.NewWithChildren(os, []NodeID{wrapper}))
	rs := defaultStyle()
	rs.Size = hwSizeDim(hwDim(1920.), hwDim(1080.))
	root := hwUnwrap(taffy.NewWithChildren(rs, []NodeID{outer}))

	_ = taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]())
	taffy.PrintTree(root)

	initialRootLayout := hwLayout(taffy, root)
	hwAssertEq(t, initialRootLayout.Location.X, 0.0)
	hwAssertEq(t, initialRootLayout.Location.Y, 0.0)
	hwAssertEq(t, initialRootLayout.Size.Width, 1920.0)
	hwAssertEq(t, initialRootLayout.Size.Height, 1080.0)

	initialOuterLayout := hwLayout(taffy, outer)
	hwAssertEq(t, initialOuterLayout.Location.X, 2.0)
	hwAssertEq(t, initialOuterLayout.Location.Y, 0.0)
	hwAssertEq(t, initialOuterLayout.Size.Width, 1920.0)
	hwAssertEq(t, initialOuterLayout.Size.Height, 1080.0)

	initialWrapperLayout := hwLayout(taffy, wrapper)
	hwAssertEq(t, initialWrapperLayout.Location.X, 0.0)
	hwAssertEq(t, initialWrapperLayout.Location.Y, 0.0)
	hwAssertEq(t, initialWrapperLayout.Size.Width, 150.0)
	hwAssertEq(t, initialWrapperLayout.Size.Height, 1080.0)

	initialInnerLayout := hwLayout(taffy, inner)
	hwAssertEq(t, initialInnerLayout.Location.X, -150.0)
	hwAssertEq(t, initialInnerLayout.Location.Y, 0.0)
	hwAssertEq(t, initialInnerLayout.Size.Width, 300.0)
	hwAssertEq(t, initialInnerLayout.Size.Height, 1080.0)

	for range 5 {
		_ = taffy.MarkDirty(root)
		_ = taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]())
		taffy.PrintTree(root)

		rootLayout := hwLayout(taffy, root)
		hwAssertEq(t, initialRootLayout.Location.X, rootLayout.Location.X)
		hwAssertEq(t, initialRootLayout.Location.Y, rootLayout.Location.Y)
		hwAssertEq(t, initialRootLayout.Size.Width, rootLayout.Size.Width)
		hwAssertEq(t, initialRootLayout.Size.Height, rootLayout.Size.Height)
		outerLayout := hwLayout(taffy, outer)
		hwAssertEq(t, initialOuterLayout.Location.X, outerLayout.Location.X)
		hwAssertEq(t, initialOuterLayout.Location.Y, outerLayout.Location.Y)
		hwAssertEq(t, initialOuterLayout.Size.Width, outerLayout.Size.Width)
		hwAssertEq(t, initialOuterLayout.Size.Height, outerLayout.Size.Height)
		wrapperLayout := hwLayout(taffy, wrapper)
		hwAssertEq(t, initialWrapperLayout.Location.X, wrapperLayout.Location.X)
		hwAssertEq(t, initialWrapperLayout.Location.X, wrapperLayout.Location.Y)
		hwAssertEq(t, initialWrapperLayout.Size.Width, wrapperLayout.Size.Width)
		hwAssertEq(t, initialWrapperLayout.Size.Height, wrapperLayout.Size.Height)
		innerLayout := hwLayout(taffy, inner)
		hwAssertEq(t, initialInnerLayout.Location.X, innerLayout.Location.X)
		hwAssertEq(t, initialInnerLayout.Location.Y, innerLayout.Location.Y)
		hwAssertEq(t, initialInnerLayout.Size.Width, innerLayout.Size.Width)
		hwAssertEq(t, initialInnerLayout.Size.Height, innerLayout.Size.Height)
	}
}

func relayoutAndFresh(before, after Style[defaultCheapStr], build func(*TaffyTree[TestNodeContext], Style[defaultCheapStr]) []NodeID) ([]Layout, []Layout) {
	layouts := func(tree *TaffyTree[TestNodeContext], ids []NodeID) []Layout {
		out := make([]Layout, 0, len(ids))
		for _, id := range ids {
			out = append(out, *tree.UnroundedLayout(id))
		}
		return out
	}

	tree := newTestTree()
	ids := build(tree, before)
	root := ids[len(ids)-1]
	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	hwOK(tree.SetStyle(ids[0], after))
	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	fresh := newTestTree()
	freshIDs := build(fresh, after)
	hwOK(fresh.ComputeLayout(freshIDs[len(freshIDs)-1], SizeMaxContent[AvailableSpace]()))

	return layouts(tree, ids), layouts(fresh, freshIDs)
}

func relayoutAssertLayoutsEq(t *testing.T, left, right []Layout) {
	t.Helper()
	if !slices.Equal(left, right) {
		t.Fatalf("assertion `left == right` failed\n  left: %+v\n right: %+v", left, right)
	}
}

func relayoutBlockParentAs(t *testing.T, container Style[defaultCheapStr]) {
	t.Helper()
	size := hwSizeDim(hwDim(100.0), hwDim(200.0))
	block := defaultStyle()
	block.Display = DisplayBlock
	block.Size = size
	container.Size = size

	relaid, fresh := relayoutAndFresh(block, container, func(tree *TaffyTree[TestNodeContext], rootStyle Style[defaultCheapStr]) []NodeID {
		is := defaultStyle()
		is.Display = DisplayBlock
		is.Size = hwSizeDim(DimensionAuto(), hwDim(10.0))
		is.Margin = RectZero[LengthPercentageAuto]()
		is.Margin.Top = hwLPA(20.0)
		inner := hwUnwrap(tree.NewLeaf(is))
		ms := defaultStyle()
		ms.Display = DisplayBlock
		ms.Size = hwSizeDim(DimensionAuto(), hwDim(50.0))
		middle := hwUnwrap(tree.NewWithChildren(ms, []NodeID{inner}))
		root := hwUnwrap(tree.NewWithChildren(rootStyle, []NodeID{middle}))
		return []NodeID{root, middle, inner, root}
	})

	hwAssertEqMsg(t, fresh[1].Location.Y, 0.0, "the item is at the top of its container")
	hwAssertEqMsg(t, fresh[2].Location.Y, 20.0, "the item contains its child's margin")
	relayoutAssertLayoutsEq(t, relaid, fresh)
}

func TestHandWrittenRelayout_BlockChildDoesNotKeepCollapsedMarginAsFlexItem(t *testing.T) {
	s := defaultStyle()
	s.Display = DisplayFlex
	s.FlexDirection = FlexDirectionColumn
	relayoutBlockParentAs(t, s)
}

func TestHandWrittenRelayout_BlockChildDoesNotKeepCollapsedMarginAsGridItem(t *testing.T) {
	s := defaultStyle()
	s.Display = DisplayGrid
	relayoutBlockParentAs(t, s)
}

func TestHandWrittenRelayout_FlexBaseSizeIsNotAnsweredByAnInherentSizeMeasurement(t *testing.T) {
	container := func(display Display) Style[defaultCheapStr] {
		s := defaultStyle()
		s.Display = display
		s.Size = hwSizeDim(hwDim(100.0), DimensionAuto())
		return s
	}

	relaid, fresh := relayoutAndFresh(container(DisplayGrid), container(DisplayFlex), func(tree *TaffyTree[TestNodeContext], rootStyle Style[defaultCheapStr]) []NodeID {
		fs := defaultStyle()
		fs.Display = DisplayGrid
		fs.FlexGrow = 1.0
		fs.Size = hwSizeDim(DimensionAuto(), hwDim(20.0))
		fs.MinSize = Size[LengthPercentageAuto]{Width: hwLPA(40.0), Height: LengthPercentageAutoAuto()}
		first := hwUnwrap(tree.NewLeaf(fs))
		ss := defaultStyle()
		ss.Display = DisplayBlock
		ss.FlexGrow = 1.0
		ss.Size = hwSizeDim(DimensionAuto(), hwDim(20.0))
		second := hwUnwrap(tree.NewLeaf(ss))
		root := hwUnwrap(tree.NewWithChildren(rootStyle, []NodeID{first, second}))
		return []NodeID{root, first, second, root}
	})

	hwAssertEq(t, fresh[1].Size.Width, 50.0)
	hwAssertEq(t, fresh[2].Size.Width, 50.0)
	relayoutAssertLayoutsEq(t, relaid, fresh)
}

func TestHandWrittenRelayout_APercentageHeightIsNotReusedAfterItsContainingBlockLosesItsHeight(t *testing.T) {
	block := func(height Dimension) Style[defaultCheapStr] {
		s := defaultStyle()
		s.Display = DisplayBlock
		s.Size = hwSizeDim(hwDim(78.0), height)
		return s
	}
	incremental, fresh := relayoutAndFresh(block(hwDim(62.0)), block(DimensionAuto()), func(tree *TaffyTree[TestNodeContext], middleStyle Style[defaultCheapStr]) []NodeID {
		ls := defaultStyle()
		ls.Display = DisplayBlock
		ls.Size = hwSizeDim(DimensionAuto(), hwDim(19.0))
		leaf := hwUnwrap(tree.NewLeaf(ls))
		is := defaultStyle()
		is.Display = DisplayFlex
		is.Size = hwSizeDim(hwDim(31.0), DimensionPercent(0.86))
		item := hwUnwrap(tree.NewWithChildren(is, []NodeID{leaf}))
		middle := hwUnwrap(tree.NewWithChildren(middleStyle, []NodeID{item}))
		rs := defaultStyle()
		rs.Display = DisplayFlex
		rs.FlexDirection = FlexDirectionColumn
		rs.Size = hwSizeDim(hwDim(78.0), DimensionAuto())
		root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{middle}))
		return []NodeID{middle, item, leaf, root}
	})
	hwAssertEq(t, fresh[1].Size.Height, 19.0)
	relayoutAssertLayoutsEq(t, incremental, fresh)
}
