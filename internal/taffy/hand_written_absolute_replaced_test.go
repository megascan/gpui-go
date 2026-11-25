package taffy

import "testing"

var absoluteReplacedImage = testNodeContextAspectRatio(100.0, 0.5)

func absoluteReplacedContainer() Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Position = PositionRelative
	s.Size = hwSizeDim(hwDim(400.0), hwDim(300.0))
	return s
}

func absoluteReplacedAllInsets(value float32) Rect[LengthPercentageAuto] {
	return Rect[LengthPercentageAuto]{Left: hwLPA(value), Right: hwLPA(value), Top: hwLPA(value), Bottom: hwLPA(value)}
}

func absoluteReplacedLayoutChild(childStyle Style[defaultCheapStr]) Layout {
	taffy := newTestTree()
	child := hwUnwrap(taffy.NewLeafWithContext(childStyle, absoluteReplacedImage))
	root := hwUnwrap(taffy.NewWithChildren(absoluteReplacedContainer(), []NodeID{child}))
	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), testMeasureFunction))
	return hwLayout(taffy, child)
}

func TestHandWrittenAbsoluteReplaced_AutoSizeUsesIntrinsicSizeDespiteInsets(t *testing.T) {
	s := defaultStyle()
	s.Position = PositionAbsolute
	s.ItemIsReplaced = true
	s.Inset = absoluteReplacedAllInsets(10.0)
	layout := absoluteReplacedLayoutChild(s)
	hwAssertEq(t, layout.Size, Size[float32]{Width: 100.0, Height: 50.0})
	hwAssertEq(t, layout.Location, Point[float32]{X: 10.0, Y: 10.0})
}

func TestHandWrittenAbsoluteReplaced_AutoHeightUsesAspectRatioDespiteInsets(t *testing.T) {
	s := defaultStyle()
	s.Position = PositionAbsolute
	s.ItemIsReplaced = true
	s.Inset = absoluteReplacedAllInsets(10.0)
	s.Size = hwSizeDim(hwDim(200.0), DimensionAuto())
	layout := absoluteReplacedLayoutChild(s)
	hwAssertEq(t, layout.Size, Size[float32]{Width: 200.0, Height: 100.0})
}

func TestHandWrittenAbsoluteReplaced_NonReplacedAutoSizeIsStretchedBetweenInsets(t *testing.T) {
	s := defaultStyle()
	s.Position = PositionAbsolute
	s.Inset = absoluteReplacedAllInsets(10.0)
	layout := absoluteReplacedLayoutChild(s)
	hwAssertEq(t, layout.Size, Size[float32]{Width: 380.0, Height: 280.0})
}
