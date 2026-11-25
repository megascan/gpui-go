package taffy

import "testing"

const (
	scrollableOverflowContainer float32 = 200.0
	scrollableOverflowChild     float32 = 50.0
)

var scrollableOverflowDisplays = []Display{DisplayBlock, DisplayFlex, DisplayGrid}

func scrollableOverflowInset(v Option[float32]) LengthPercentageAuto {
	if x, ok := v.Get(); ok {
		return hwLPA(x)
	}
	return LengthPercentageAutoAuto()
}

func scrollableOverflowContainerWithAbsoluteChild(display Display, overflow Overflow, direction Direction, insetLeft, insetRight Option[float32]) Layout {
	tree := newTestTree()
	cs := defaultStyle()
	cs.Position = PositionAbsolute
	cs.Inset = Rect[LengthPercentageAuto]{
		Left:   scrollableOverflowInset(insetLeft),
		Right:  scrollableOverflowInset(insetRight),
		Top:    hwLPA(0.0),
		Bottom: LengthPercentageAutoAuto(),
	}
	cs.Size = hwSizeDim(hwDim(scrollableOverflowChild), hwDim(scrollableOverflowChild))
	child := hwUnwrap(tree.NewLeaf(cs))
	ns := defaultStyle()
	ns.Display = display
	ns.Direction = direction
	ns.Position = PositionRelative
	ns.Size = hwSizeDim(hwDim(scrollableOverflowContainer), hwDim(scrollableOverflowContainer))
	ns.Overflow = Point[Overflow]{X: overflow, Y: overflow}
	node := hwUnwrap(tree.NewWithChildren(ns, []NodeID{child}))

	hwOK(tree.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
	return hwLayout(tree, node)
}

func TestHandWrittenScrollableOverflow_WhollyUnreachableBoxesAreExcludedForScrollContainers(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		layout := scrollableOverflowContainerWithAbsoluteChild(display, OverflowScroll, DirectionLtr, some[float32](-100.0), none[float32]())

		hwAssertEqMsg(t, layout.ScrollableOverflowRect, RectF32Zero, display.String())
		hwAssertEqMsg(t, layout.ScrollWidth(), 0.0, display.String())
	}
}

func TestHandWrittenScrollableOverflow_WhollyUnreachableBoxesContributeToNonScrollContainers(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		layout := scrollableOverflowContainerWithAbsoluteChild(display, OverflowVisible, DirectionLtr, some[float32](-100.0), none[float32]())

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Left, -100.0, display.String())
	}
}

func TestHandWrittenScrollableOverflow_PartiallyUnreachableBoxesContributeTheirWholeBorderBox(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		layout := scrollableOverflowContainerWithAbsoluteChild(display, OverflowScroll, DirectionLtr, some[float32](-25.0), none[float32]())

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Left, -25.0, display.String())
		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Right, 25.0, display.String())
		hwAssertEqMsg(t, layout.ScrollWidth(), 0.0, display.String())
	}
}

func TestHandWrittenScrollableOverflow_EndSideOverflowIsCapturedAndScrollable(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		layout := scrollableOverflowContainerWithAbsoluteChild(display, OverflowScroll, DirectionLtr, some(scrollableOverflowContainer), none[float32]())

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Right, scrollableOverflowContainer+scrollableOverflowChild, display.String())
		hwAssertEqMsg(t, layout.ScrollWidth(), scrollableOverflowChild, display.String())
	}
}

func TestHandWrittenScrollableOverflow_AbsoluteOnlyContainersIncludeScrollContainerEndPadding(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		tree := newTestTree()
		as := defaultStyle()
		as.Position = PositionAbsolute
		as.Inset = Rect[LengthPercentageAuto]{Left: hwLPA(0.0), Right: LengthPercentageAutoAuto(), Top: hwLPA(0.0), Bottom: LengthPercentageAutoAuto()}
		as.Size = hwSizeDim(hwDim(5.0), hwDim(5.0))
		abs := hwUnwrap(tree.NewLeaf(as))
		ns := defaultStyle()
		ns.Display = display
		ns.Position = PositionRelative
		ns.Overflow = Point[Overflow]{X: OverflowScroll, Y: OverflowScroll}
		ns.Padding = Rect[LengthPercentage]{Left: LengthPercentageLength(0.0), Right: LengthPercentageLength(20.0), Top: LengthPercentageLength(0.0), Bottom: LengthPercentageLength(30.0)}
		ns.Size = hwSizeDim(hwDim(scrollableOverflowContainer), hwDim(scrollableOverflowContainer))
		node := hwUnwrap(tree.NewWithChildren(ns, []NodeID{abs}))

		hwOK(tree.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
		layout := hwLayout(tree, node)

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Right, 20.0, display.String())
		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Bottom, 30.0, display.String())
	}
}

func scrollableOverflowNestedOverflowLayout(display Display, childOverflow Point[Overflow]) Layout {
	tree := newTestTree()
	gs := defaultStyle()
	gs.Position = PositionAbsolute
	gs.Inset = Rect[LengthPercentageAuto]{Left: hwLPA(0.0), Right: LengthPercentageAutoAuto(), Top: hwLPA(scrollableOverflowContainer), Bottom: LengthPercentageAutoAuto()}
	gs.Size = hwSizeDim(hwDim(scrollableOverflowChild), hwDim(scrollableOverflowChild))
	grandchild := hwUnwrap(tree.NewLeaf(gs))
	cs := defaultStyle()
	cs.Display = display
	cs.Position = PositionRelative
	cs.Size = hwSizeDim(hwDim(scrollableOverflowContainer), hwDim(scrollableOverflowContainer))
	cs.Overflow = childOverflow
	child := hwUnwrap(tree.NewWithChildren(cs, []NodeID{grandchild}))
	ns := defaultStyle()
	ns.Display = display
	ns.Size = hwSizeDim(hwDim(scrollableOverflowContainer), hwDim(scrollableOverflowContainer))
	node := hwUnwrap(tree.NewWithChildren(ns, []NodeID{child}))

	hwOK(tree.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
	return hwLayout(tree, node)
}

func TestHandWrittenScrollableOverflow_ScrollContainerInOneAxisClipsBothAxes(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		layout := scrollableOverflowNestedOverflowLayout(display, Point[Overflow]{X: OverflowHidden, Y: OverflowVisible})

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Bottom, scrollableOverflowContainer, display.String())
	}
}

func TestHandWrittenScrollableOverflow_ClipInOneAxisDoesNotClipTheOther(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		layout := scrollableOverflowNestedOverflowLayout(display, Point[Overflow]{X: OverflowClip, Y: OverflowVisible})

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Bottom, scrollableOverflowContainer+scrollableOverflowChild, display.String())
	}
}

func TestHandWrittenScrollableOverflow_WhollyUnreachableBoxesAreExcludedForRtlScrollContainers(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		layout := scrollableOverflowContainerWithAbsoluteChild(display, OverflowScroll, DirectionRtl, none[float32](), some[float32](-100.0))

		hwAssertEqMsg(t, layout.ScrollableOverflowRect, RectF32Zero, display.String())
		hwAssertEqMsg(t, layout.ScrollWidth(), 0.0, display.String())
	}
}

func TestHandWrittenScrollableOverflow_EndSideOverflowIsCapturedForRtlScrollContainers(t *testing.T) {
	for _, display := range scrollableOverflowDisplays {
		layout := scrollableOverflowContainerWithAbsoluteChild(display, OverflowScroll, DirectionRtl, some(-scrollableOverflowChild), none[float32]())

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Right, scrollableOverflowContainer+scrollableOverflowChild, display.String())
		hwAssertEqMsg(t, layout.ScrollWidth(), scrollableOverflowChild, display.String())
	}
}
