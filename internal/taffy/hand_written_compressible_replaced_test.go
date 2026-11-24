package taffy

import "testing"

var compressibleReplacedInput = testNodeContextAspectRatio(150.0, 20.0/150.0)

func compressibleReplacedLayoutChild(container, childStyle Style[defaultCheapStr]) Layout {
	taffy := newTestTree()
	child := hwUnwrap(taffy.NewLeafWithContext(childStyle, compressibleReplacedInput))
	root := hwUnwrap(taffy.NewWithChildren(container, []NodeID{child}))
	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), testMeasureFunction))
	return hwLayout(taffy, child)
}

func compressibleReplacedBlockContainer() Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Size = hwSizeDim(hwDim(400.0), DimensionAuto())
	return s
}

func compressibleReplacedPositionedContainer() Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Position = PositionRelative
	s.Size = hwSizeDim(hwDim(400.0), hwDim(300.0))
	return s
}

func compressibleReplacedAllInsets(value float32) Rect[LengthPercentageAuto] {
	return Rect[LengthPercentageAuto]{Left: hwLPA(value), Right: hwLPA(value), Top: hwLPA(value), Bottom: hwLPA(value)}
}

func TestHandWrittenCompressibleReplaced_BlockLevelAutoWidthUsesIntrinsicSize(t *testing.T) {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.ItemIsCompressibleReplaced = true
	layout := compressibleReplacedLayoutChild(compressibleReplacedBlockContainer(), s)
	hwAssertEq(t, layout.Size, Size[float32]{Width: 150.0, Height: 20.0})
}

func TestHandWrittenCompressibleReplaced_BlockLevelJustifySelfStretchStretches(t *testing.T) {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.ItemIsCompressibleReplaced = true
	s.JustifySelf = some(AlignItemsStretch)
	layout := compressibleReplacedLayoutChild(compressibleReplacedBlockContainer(), s)
	hwAssertEq(t, layout.Size.Width, 400.0)
}

func TestHandWrittenCompressibleReplaced_AbsoluteAutoSizeIsStretchedBetweenInsets(t *testing.T) {
	s := defaultStyle()
	s.Position = PositionAbsolute
	s.ItemIsCompressibleReplaced = true
	s.Inset = compressibleReplacedAllInsets(10.0)
	layout := compressibleReplacedLayoutChild(compressibleReplacedPositionedContainer(), s)
	hwAssertEq(t, layout.Size, Size[float32]{Width: 380.0, Height: 280.0})
	hwAssertEq(t, layout.Location, Point[float32]{X: 10.0, Y: 10.0})
}

func TestHandWrittenCompressibleReplaced_ReplacedImpliesCompressibleReplaced(t *testing.T) {
	style := defaultStyle()
	style.ItemIsReplaced = true
	if !style.Ref().IsCompressibleReplaced() {
		t.Fatal("assertion failed: CoreStyle::is_compressible_replaced(&style)")
	}
	if !style.Ref().IsReplaced() {
		t.Fatal("assertion failed: CoreStyle::is_replaced(&style)")
	}
	style = defaultStyle()
	style.ItemIsCompressibleReplaced = true
	if !style.Ref().IsCompressibleReplaced() {
		t.Fatal("assertion failed: CoreStyle::is_compressible_replaced(&style)")
	}
	if style.Ref().IsReplaced() {
		t.Fatal("assertion failed: !CoreStyle::is_replaced(&style)")
	}
}
