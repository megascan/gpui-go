package taffy

import (
	"fmt"
	"testing"
)

var initialContainingBlockViewport = hwDefinite(800.0, 600.0)

func initialContainingBlockRootStyle() Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Margin = Rect[LengthPercentageAuto]{Left: hwLPA(10.0), Right: hwLPA(10.0), Top: hwLPA(20.0), Bottom: hwLPA(20.0)}
	s.Border = Rect[LengthPercentage]{Left: LengthPercentageLength(5.0), Right: LengthPercentageLength(5.0), Top: LengthPercentageLength(5.0), Bottom: LengthPercentageLength(5.0)}
	return s
}

func initialContainingBlockOofStyle(position Position, inset Rect[LengthPercentageAuto], size Size[Dimension]) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Position = position
	s.Inset = inset
	s.Size = size
	return s
}

func TestHandWrittenInitialContainingBlock_RootIsPlacedAtItsMargin(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	root := hwUnwrap(tree.NewLeaf(initialContainingBlockRootStyle()))
	hwOK(tree.ComputeLayout(root, initialContainingBlockViewport))

	layout := hwLayout(tree, root)
	hwAssertEq(t, layout.Location, Point[float32]{X: 10.0, Y: 20.0})
	hwAssertEq(t, layout.Size.Width, 780.0)
}

func TestHandWrittenInitialContainingBlock_InsetZeroFillsViewport(t *testing.T) {
	for _, position := range []Position{PositionAbsolute, PositionFixed} {
		tree := NewTaffyTree[struct{}]()
		oof := hwUnwrap(tree.NewLeaf(initialContainingBlockOofStyle(position, Length[Rect[LengthPercentageAuto]](0.0), SizeAuto[Dimension]())))
		root := hwUnwrap(tree.NewWithChildren(initialContainingBlockRootStyle(), []NodeID{oof}))
		hwOK(tree.ComputeLayout(root, initialContainingBlockViewport))

		layout := hwLayout(tree, oof)
		msg := fmt.Sprintf("%v", position)
		hwAssertEqMsg(t, layout.Location, Point[float32]{X: -10.0, Y: -20.0}, msg)
		hwAssertEqMsg(t, layout.Size, Size[float32]{Width: 800.0, Height: 600.0}, msg)
	}
}

func TestHandWrittenInitialContainingBlock_BottomRightInsetsResolveAgainstViewport(t *testing.T) {
	for _, position := range []Position{PositionAbsolute, PositionFixed} {
		tree := NewTaffyTree[struct{}]()
		oof := hwUnwrap(tree.NewLeaf(initialContainingBlockOofStyle(
			position,
			Rect[LengthPercentageAuto]{Left: LengthPercentageAutoAuto(), Top: LengthPercentageAutoAuto(), Right: hwLPA(0.0), Bottom: hwLPA(0.0)},
			hwSizeDim(hwDim(50.0), hwDim(50.0)),
		)))
		root := hwUnwrap(tree.NewWithChildren(initialContainingBlockRootStyle(), []NodeID{oof}))
		hwOK(tree.ComputeLayout(root, initialContainingBlockViewport))

		layout := hwLayout(tree, oof)
		hwAssertEqMsg(t, layout.Location, Point[float32]{X: 800.0 - 50.0 - 10.0, Y: 600.0 - 50.0 - 20.0}, fmt.Sprintf("%v", position))
	}
}

func TestHandWrittenInitialContainingBlock_StaticPositionIsUnaffected(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	oof := hwUnwrap(tree.NewLeaf(initialContainingBlockOofStyle(PositionAbsolute, RectAuto[LengthPercentageAuto](), hwSizeDim(hwDim(50.0), hwDim(50.0)))))
	root := hwUnwrap(tree.NewWithChildren(initialContainingBlockRootStyle(), []NodeID{oof}))
	hwOK(tree.ComputeLayout(root, initialContainingBlockViewport))

	hwAssertEq(t, hwLayout(tree, oof).Location, Point[float32]{X: 5.0, Y: 5.0})
}

func TestHandWrittenInitialContainingBlock_FixedRootIsPositionedByItsInsets(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Position = PositionFixed
	s.Inset = Rect[LengthPercentageAuto]{Left: LengthPercentageAutoAuto(), Top: LengthPercentageAutoPercent(0.1), Right: hwLPA(10.0), Bottom: LengthPercentageAutoAuto()}
	s.Size = hwSizeDim(hwDim(100.0), hwDim(50.0))
	root := hwUnwrap(tree.NewLeaf(s))
	hwOK(tree.ComputeLayout(root, initialContainingBlockViewport))

	layout := hwLayout(tree, root)
	hwAssertEq(t, layout.Location, Point[float32]{X: 800.0 - 100.0 - 10.0, Y: 60.0})
	hwAssertEq(t, layout.Size, Size[float32]{Width: 100.0, Height: 50.0})
}
