package taffy

import "testing"

type baselineContext struct {
	size      Size[float32]
	baselineY Option[float32]
}

func baselineMeasureFunction(_ LayoutInput, _ NodeID, context *baselineContext, _ *Style[defaultCheapStr]) LayoutOutput {
	if context == nil {
		return LayoutOutputDefault
	}
	return LayoutOutputFromSizesAndBaselines(context.size, RectF32Zero, BaselinesFromFirst(context.baselineY))
}

func baselineFlexRoot(alignItems AlignItems, width, height float32) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayFlex
	s.FlexDirection = FlexDirectionRow
	s.AlignItems = alignItems
	s.Size = hwSizeDim(hwDim(width), hwDim(height))
	return s
}

func baselineItemA() Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Size = hwSizeDim(hwDim(40.0), DimensionAuto())
	s.Padding = Rect[LengthPercentage]{Left: LengthPercentageZero, Right: LengthPercentageZero, Top: LengthPercentageZero, Bottom: LengthPercentageLength(20.0)}
	return s
}

func TestHandWrittenBaseline_FlexBaselineAlignmentUsesMeasureFunctionBaseline(t *testing.T) {
	taffy := NewTaffyTree[baselineContext]()

	childA := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), baselineContext{size: Size[float32]{Width: 50.0, Height: 50.0}, baselineY: some[float32](40.0)}))
	childB := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), baselineContext{size: Size[float32]{Width: 30.0, Height: 30.0}, baselineY: some[float32](20.0)}))
	root := hwUnwrap(taffy.NewWithChildren(baselineFlexRoot(AlignItemsBaseline, 200.0, 100.0), []NodeID{childA, childB}))

	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), baselineMeasureFunction))

	layoutA := hwLayout(taffy, childA)
	layoutB := hwLayout(taffy, childB)

	hwAssertEq(t, layoutA.Location.Y, 0.0)
	hwAssertEq(t, layoutB.Location.Y, 20.0)
	hwAssertEq(t, layoutA.Location.Y+40.0, layoutB.Location.Y+20.0)
}

func TestHandWrittenBaseline_BlockScrollContainerChildWithoutBaselineSynthesizesFromBorderBox(t *testing.T) {
	taffy := NewTaffyTree[baselineContext]()

	sc := defaultStyle()
	sc.Display = DisplayBlock
	sc.Overflow = Point[Overflow]{X: OverflowHidden, Y: OverflowHidden}
	sc.Size = hwSizeDim(hwDim(30.0), hwDim(30.0))
	scrollChild := hwUnwrap(taffy.NewLeafWithContext(sc, baselineContext{size: Size[float32]{Width: 30.0, Height: 30.0}, baselineY: none[float32]()}))
	itemA := hwUnwrap(taffy.NewWithChildren(baselineItemA(), []NodeID{scrollChild}))
	itemB := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), baselineContext{size: Size[float32]{Width: 40.0, Height: 80.0}, baselineY: some[float32](10.0)}))
	root := hwUnwrap(taffy.NewWithChildren(baselineFlexRoot(AlignItemsBaseline, 200.0, 120.0), []NodeID{itemA, itemB}))

	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), baselineMeasureFunction))

	hwAssertEq(t, hwLayout(taffy, itemA).Location.Y, 0.0)
	hwAssertEq(t, hwLayout(taffy, itemB).Location.Y, 20.0)
}

func TestHandWrittenBaseline_BlockNonScrollContainerChildWithoutBaselineContributesNothing(t *testing.T) {
	taffy := NewTaffyTree[baselineContext]()

	pc := defaultStyle()
	pc.Display = DisplayBlock
	pc.Size = hwSizeDim(hwDim(30.0), hwDim(30.0))
	plainChild := hwUnwrap(taffy.NewLeafWithContext(pc, baselineContext{size: Size[float32]{Width: 30.0, Height: 30.0}, baselineY: none[float32]()}))
	itemA := hwUnwrap(taffy.NewWithChildren(baselineItemA(), []NodeID{plainChild}))
	itemB := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), baselineContext{size: Size[float32]{Width: 40.0, Height: 80.0}, baselineY: some[float32](10.0)}))
	root := hwUnwrap(taffy.NewWithChildren(baselineFlexRoot(AlignItemsBaseline, 200.0, 120.0), []NodeID{itemA, itemB}))

	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), baselineMeasureFunction))

	hwAssertEq(t, hwLayout(taffy, itemA).Location.Y, 0.0)
	hwAssertEq(t, hwLayout(taffy, itemB).Location.Y, 40.0)
}

func TestHandWrittenBaseline_BlockScrollContainerChildBaselineIsClampedToBorderBox(t *testing.T) {
	taffy := NewTaffyTree[baselineContext]()

	cs := defaultStyle()
	cs.Display = DisplayBlock
	content := hwUnwrap(taffy.NewLeafWithContext(cs, baselineContext{size: Size[float32]{Width: 30.0, Height: 60.0}, baselineY: some[float32](48.0)}))
	sc := defaultStyle()
	sc.Display = DisplayBlock
	sc.Overflow = Point[Overflow]{X: OverflowHidden, Y: OverflowHidden}
	sc.Size = hwSizeDim(hwDim(30.0), hwDim(30.0))
	scrollChild := hwUnwrap(taffy.NewWithChildren(sc, []NodeID{content}))
	itemA := hwUnwrap(taffy.NewWithChildren(baselineItemA(), []NodeID{scrollChild}))
	itemB := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), baselineContext{size: Size[float32]{Width: 40.0, Height: 80.0}, baselineY: some[float32](10.0)}))
	root := hwUnwrap(taffy.NewWithChildren(baselineFlexRoot(AlignItemsBaseline, 200.0, 120.0), []NodeID{itemA, itemB}))

	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), baselineMeasureFunction))

	hwAssertEq(t, hwLayout(taffy, itemA).Location.Y, 0.0)
	hwAssertEq(t, hwLayout(taffy, itemB).Location.Y, 20.0)
}

func TestHandWrittenBaseline_FlexBaselineIsIgnoredWhenAlignmentIsNotBaseline(t *testing.T) {
	taffy := NewTaffyTree[baselineContext]()

	childA := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), baselineContext{size: Size[float32]{Width: 50.0, Height: 50.0}, baselineY: some[float32](40.0)}))
	childB := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), baselineContext{size: Size[float32]{Width: 30.0, Height: 30.0}, baselineY: some[float32](20.0)}))
	root := hwUnwrap(taffy.NewWithChildren(baselineFlexRoot(AlignItemsFlexStart, 200.0, 100.0), []NodeID{childA, childB}))

	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), baselineMeasureFunction))

	hwAssertEq(t, hwLayout(taffy, childA).Location.Y, 0.0)
	hwAssertEq(t, hwLayout(taffy, childB).Location.Y, 0.0)
}
