package taffy

import (
	"testing"
	"unsafe"
)

var (
	measureHundredHundred = testNodeContextFixed(100.0, 100.0)
	measureHundredFifty   = testNodeContextFixed(100.0, 50.0)
	measureFiftyFifty     = testNodeContextFixed(50.0, 50.0)
)

func measureSizedStyle(width, height Dimension) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Size = hwSizeDim(width, height)
	return s
}

func measureCompute(taffy *TaffyTree[TestNodeContext], node NodeID) {
	hwOK(taffy.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), testMeasureFunction))
}

func TestHandWrittenMeasure_MeasureRoot(t *testing.T) {
	taffy := newTestTree()
	node := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), measureHundredHundred))
	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, node).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, node).Size.Height, 100.0)
}

func TestHandWrittenMeasure_MeasureChild(t *testing.T) {
	taffy := newTestTree()

	child := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), measureHundredHundred))

	node := hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{child}))
	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, node).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, node).Size.Height, 100.0)

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)
}

func TestHandWrittenMeasure_MeasureChildConstraint(t *testing.T) {
	taffy := newTestTree()
	child := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), measureHundredHundred))

	node := hwUnwrap(taffy.NewWithChildren(measureSizedStyle(Length[Dimension](50.0), DimensionAuto()), []NodeID{child}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, node).Size.Width, 50.0)
	hwAssertEq(t, hwLayout(taffy, node).Size.Height, 100.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)
}

func TestHandWrittenMeasure_MeasureChildConstraintPaddingParent(t *testing.T) {
	taffy := newTestTree()
	child := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), measureHundredHundred))

	s := measureSizedStyle(Length[Dimension](50.0), DimensionAuto())
	lp := Length[LengthPercentage](10.0)
	s.Padding = Rect[LengthPercentage]{Left: lp, Right: lp, Top: lp, Bottom: lp}
	node := hwUnwrap(taffy.NewWithChildren(s, []NodeID{child}))
	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, node).Location.X, 0.0)
	hwAssertEq(t, hwLayout(taffy, node).Location.Y, 0.0)
	hwAssertEq(t, hwLayout(taffy, node).Size.Width, 50.0)
	hwAssertEq(t, hwLayout(taffy, node).Size.Height, 120.0)

	hwAssertEq(t, hwLayout(taffy, child).Location.X, 10.0)
	hwAssertEq(t, hwLayout(taffy, child).Location.Y, 10.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)
}

func TestHandWrittenMeasure_MeasureChildWithFlexGrow(t *testing.T) {
	taffy := newTestTree()
	child0 := hwUnwrap(taffy.NewLeaf(measureSizedStyle(Length[Dimension](50.0), Length[Dimension](50.0))))

	cs := defaultStyle()
	cs.FlexGrow = 1.0
	child1 := hwUnwrap(taffy.NewLeafWithContext(cs, measureFiftyFifty))

	node := hwUnwrap(taffy.NewWithChildren(measureSizedStyle(Length[Dimension](100.0), DimensionAuto()), []NodeID{child0, child1}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child1).Size.Width, 50.0)
	hwAssertEq(t, hwLayout(taffy, child1).Size.Height, 50.0)
}

func TestHandWrittenMeasure_MeasureChildWithFlexShrink(t *testing.T) {
	taffy := newTestTree()
	s0 := measureSizedStyle(Length[Dimension](50.0), Length[Dimension](50.0))
	s0.FlexShrink = 0.0
	child0 := hwUnwrap(taffy.NewLeaf(s0))

	child1 := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), measureHundredFifty))

	node := hwUnwrap(taffy.NewWithChildren(measureSizedStyle(Length[Dimension](100.0), DimensionAuto()), []NodeID{child0, child1}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child1).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child1).Size.Height, 50.0)
}

func TestHandWrittenMeasure_RemeasureChildAfterGrowing(t *testing.T) {
	taffy := newTestTree()
	child0 := hwUnwrap(taffy.NewLeaf(measureSizedStyle(Length[Dimension](50.0), Length[Dimension](50.0))))

	cs := defaultStyle()
	cs.FlexGrow = 1.0
	child1 := hwUnwrap(taffy.NewLeafWithContext(cs, testNodeContextAspectRatio(10.0, 2.0)))

	ns := measureSizedStyle(Length[Dimension](100.0), DimensionAuto())
	ns.AlignItems = AlignItemsStart
	node := hwUnwrap(taffy.NewWithChildren(ns, []NodeID{child0, child1}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child1).Size.Width, 50.0)
	hwAssertEq(t, hwLayout(taffy, child1).Size.Height, 100.0)
}

func TestHandWrittenMeasure_RemeasureChildAfterShrinking(t *testing.T) {
	taffy := newTestTree()

	s0 := measureSizedStyle(Length[Dimension](50.0), Length[Dimension](50.0))
	s0.FlexShrink = 0.0
	child0 := hwUnwrap(taffy.NewLeaf(s0))

	child1 := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), testNodeContextAspectRatio(100.0, 2.0)))

	ns := measureSizedStyle(Length[Dimension](100.0), DimensionAuto())
	ns.AlignItems = AlignItemsStart
	node := hwUnwrap(taffy.NewWithChildren(ns, []NodeID{child0, child1}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child1).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child1).Size.Height, 200.0)
}

func TestHandWrittenMeasure_RemeasureChildAfterStretching(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	customMeasureFunction := func(inputs LayoutInput, _ NodeID, _ *struct{}, style *Style[defaultCheapStr]) LayoutOutput {
		return ComputeLeafLayout(
			inputs,
			style.Ref(),
			func(unsafe.Pointer, float32) float32 { return 0.0 },
			func(knownDimensions Size[Option[float32]], _ Size[AvailableSpace]) Size[float32] {
				height := knownDimensions.Height.UnwrapOr(50.0)
				width := knownDimensions.Width.UnwrapOr(height)
				return Size[float32]{Width: width, Height: height}
			},
		)
	}

	child := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), struct{}{}))

	node := hwUnwrap(taffy.NewWithChildren(measureSizedStyle(Length[Dimension](100.0), Length[Dimension](100.0)), []NodeID{child}))

	hwOK(taffy.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), customMeasureFunction))

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)
}

func TestHandWrittenMeasure_WidthOverridesMeasure(t *testing.T) {
	taffy := newTestTree()
	child := hwUnwrap(taffy.NewLeafWithContext(measureSizedStyle(Length[Dimension](50.0), DimensionAuto()), measureHundredHundred))

	node := hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{child}))
	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 50.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)
}

func TestHandWrittenMeasure_HeightOverridesMeasure(t *testing.T) {
	taffy := newTestTree()
	child := hwUnwrap(taffy.NewLeafWithContext(measureSizedStyle(DimensionAuto(), Length[Dimension](50.0)), measureHundredHundred))

	node := hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{child}))
	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 50.0)
}

func TestHandWrittenMeasure_FlexBasisOverridesMeasure(t *testing.T) {
	taffy := newTestTree()
	s := defaultStyle()
	s.FlexBasis = Length[Dimension](50.0)
	s.FlexGrow = 1.0
	child0 := hwUnwrap(taffy.NewLeaf(s))

	child1 := hwUnwrap(taffy.NewLeafWithContext(s, measureHundredHundred))

	node := hwUnwrap(taffy.NewWithChildren(measureSizedStyle(Length[Dimension](200.0), Length[Dimension](100.0)), []NodeID{child0, child1}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child0).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child0).Size.Height, 100.0)
	hwAssertEq(t, hwLayout(taffy, child1).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child1).Size.Height, 100.0)
}

func TestHandWrittenMeasure_StretchOverridesMeasure(t *testing.T) {
	taffy := newTestTree()
	child := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), measureFiftyFifty))

	node := hwUnwrap(taffy.NewWithChildren(measureSizedStyle(Length[Dimension](100.0), Length[Dimension](100.0)), []NodeID{child}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 50.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)
}

func TestHandWrittenMeasure_ExplicitMinMainSizeSkipsMinContentMeasurement(t *testing.T) {
	assertNotMinContent := func(inputs LayoutInput, _ NodeID, _ *struct{}, _ *Style[defaultCheapStr]) LayoutOutput {
		if inputs.AvailableSpace.Width == AvailableSpaceMinContent {
			t.Fatalf("assertion `left != right` failed\n  left: %+v\n right: %+v", inputs.AvailableSpace.Width, AvailableSpaceMinContent)
		}
		return LayoutOutputFromOuterSize(Size[float32]{Width: 10.0, Height: 10.0})
	}

	taffy := NewTaffyTree[struct{}]()
	s := defaultStyle()
	s.MinSize = Size[LengthPercentageAuto]{Width: hwLPA(10.0), Height: LengthPercentageAutoAuto()}
	child := hwUnwrap(taffy.NewLeafWithContext(s, struct{}{}))
	root := hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{child}))

	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), assertNotMinContent))
}

func TestHandWrittenMeasure_MeasureAbsoluteChild(t *testing.T) {
	taffy := newTestTree()
	cs := defaultStyle()
	cs.Position = PositionAbsolute
	child := hwUnwrap(taffy.NewLeafWithContext(cs, measureFiftyFifty))

	node := hwUnwrap(taffy.NewWithChildren(measureSizedStyle(Length[Dimension](100.0), Length[Dimension](100.0)), []NodeID{child}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 50.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 50.0)
}

func TestHandWrittenMeasure_IgnoreInvalidMeasure(t *testing.T) {
	taffy := newTestTree()
	cs := defaultStyle()
	cs.FlexGrow = 1.0
	child := hwUnwrap(taffy.NewLeaf(cs))

	node := hwUnwrap(taffy.NewWithChildren(measureSizedStyle(Length[Dimension](100.0), Length[Dimension](100.0)), []NodeID{child}))

	measureCompute(taffy, node)

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 100.0)
	hwAssertEq(t, hwLayout(taffy, child).Size.Height, 100.0)
}
