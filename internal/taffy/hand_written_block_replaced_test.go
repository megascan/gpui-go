package taffy

import "testing"

var blockReplacedBadge = testNodeContextAspectRatio(142.0, 20.0/142.0)

func blockReplacedBlockContainer(width float32) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Size = hwSizeDim(hwDim(width), DimensionAuto())
	return s
}

func blockReplacedLayout(taffy *TaffyTree[TestNodeContext], root NodeID) {
	hwOK(taffy.ComputeLayoutWithMeasure(root, SizeMaxContent[AvailableSpace](), testMeasureFunction))
}

func blockReplacedStyle() Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.ItemIsReplaced = true
	return s
}

func TestHandWrittenBlockReplaced_AutoWidthUsesIntrinsicSize(t *testing.T) {
	taffy := newTestTree()
	child := hwUnwrap(taffy.NewLeafWithContext(blockReplacedStyle(), blockReplacedBadge))
	root := hwUnwrap(taffy.NewWithChildren(blockReplacedBlockContainer(600.0), []NodeID{child}))
	blockReplacedLayout(taffy, root)

	childLayout := hwLayout(taffy, child)
	hwAssertEq(t, childLayout.Size.Width, 142.0)
	hwAssertEq(t, childLayout.Size.Height, 20.0)
}

func TestHandWrittenBlockReplaced_NonReplacedChildIsStretchSized(t *testing.T) {
	taffy := newTestTree()
	s := defaultStyle()
	s.Display = DisplayBlock
	child := hwUnwrap(taffy.NewLeafWithContext(s, blockReplacedBadge))
	root := hwUnwrap(taffy.NewWithChildren(blockReplacedBlockContainer(600.0), []NodeID{child}))
	blockReplacedLayout(taffy, root)

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 600.0)
}

func TestHandWrittenBlockReplaced_ExplicitWidthIsUsed(t *testing.T) {
	taffy := newTestTree()
	s := blockReplacedStyle()
	s.Size = hwSizeDim(hwDim(300.0), hwDim(50.0))
	child := hwUnwrap(taffy.NewLeafWithContext(s, blockReplacedBadge))
	root := hwUnwrap(taffy.NewWithChildren(blockReplacedBlockContainer(600.0), []NodeID{child}))
	blockReplacedLayout(taffy, root)

	childLayout := hwLayout(taffy, child)
	hwAssertEq(t, childLayout.Size.Width, 300.0)
	hwAssertEq(t, childLayout.Size.Height, 50.0)
}

func TestHandWrittenBlockReplaced_MaxWidthClampsIntrinsicSize(t *testing.T) {
	taffy := newTestTree()
	s := blockReplacedStyle()
	s.MaxSize = Size[LengthPercentageAuto]{Width: LengthPercentageAutoPercent(1.0), Height: LengthPercentageAutoAuto()}
	child := hwUnwrap(taffy.NewLeafWithContext(s, blockReplacedBadge))
	root := hwUnwrap(taffy.NewWithChildren(blockReplacedBlockContainer(100.0), []NodeID{child}))
	blockReplacedLayout(taffy, root)

	hwAssertEq(t, hwLayout(taffy, child).Size.Width, 100.0)
}

func TestHandWrittenBlockReplaced_AutoMarginsCenterReplacedChild(t *testing.T) {
	taffy := newTestTree()
	s := blockReplacedStyle()
	s.Margin = Rect[LengthPercentageAuto]{Left: LengthPercentageAutoAuto(), Right: LengthPercentageAutoAuto(), Top: LengthPercentageAutoZero, Bottom: LengthPercentageAutoZero}
	child := hwUnwrap(taffy.NewLeafWithContext(s, blockReplacedBadge))
	root := hwUnwrap(taffy.NewWithChildren(blockReplacedBlockContainer(600.0), []NodeID{child}))
	blockReplacedLayout(taffy, root)

	childLayout := hwLayout(taffy, child)
	hwAssertEq(t, childLayout.Size.Width, 142.0)
	hwAssertEq(t, childLayout.Location.X, (600.0-142.0)/2.0)
}
