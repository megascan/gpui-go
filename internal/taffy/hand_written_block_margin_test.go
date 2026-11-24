package taffy

import "testing"

func blockMarginBlock(marginTop, marginBottom float32, height Dimension) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Size = hwSizeDim(DimensionAuto(), height)
	s.Margin = Rect[LengthPercentageAuto]{Left: LengthPercentageAutoZero, Right: LengthPercentageAutoZero, Top: hwLPA(marginTop), Bottom: hwLPA(marginBottom)}
	return s
}

func TestHandWrittenBlockMargin_LayoutMarginExcludesCollapsedMargins(t *testing.T) {
	taffy := newTestTree()
	inner := hwUnwrap(taffy.NewLeaf(blockMarginBlock(13.0, 20.0, hwDim(10.0))))
	outer := hwUnwrap(taffy.NewWithChildren(blockMarginBlock(8.0, 5.0, DimensionAuto()), []NodeID{inner}))
	sibling := hwUnwrap(taffy.NewLeaf(blockMarginBlock(30.0, 0.0, hwDim(10.0))))
	rs := defaultStyle()
	rs.Display = DisplayFlowRoot
	rs.Size = hwSizeDim(hwDim(100.0), DimensionAuto())
	root := hwUnwrap(taffy.NewWithChildren(rs, []NodeID{outer, sibling}))

	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	outerLayout := hwLayout(taffy, outer)
	hwAssertEq(t, outerLayout.Location.Y, 13.0)
	hwAssertEq(t, outerLayout.Size.Height, 10.0)
	hwAssertEq(t, outerLayout.Margin.Top, 8.0)
	hwAssertEq(t, outerLayout.Margin.Bottom, 5.0)

	innerLayout := hwLayout(taffy, inner)
	hwAssertEq(t, innerLayout.Location.Y, 0.0)
	hwAssertEq(t, innerLayout.Margin.Top, 13.0)
	hwAssertEq(t, innerLayout.Margin.Bottom, 20.0)

	siblingLayout := hwLayout(taffy, sibling)
	hwAssertEq(t, siblingLayout.Location.Y, 53.0)
	hwAssertEq(t, siblingLayout.Margin.Top, 30.0)
}
