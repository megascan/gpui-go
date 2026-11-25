package taffy

import "testing"

func positionVariantsLeafWithInset(position Position, display Display) (*TaffyTree[struct{}], NodeID) {
	taffy := NewTaffyTree[struct{}]()
	cs := defaultStyle()
	cs.Position = position
	cs.Size = hwSizeDim(hwDim(10.0), hwDim(10.0))
	cs.Inset = Rect[LengthPercentageAuto]{Left: hwLPA(5.0), Right: LengthPercentageAutoAuto(), Top: hwLPA(7.0), Bottom: LengthPercentageAutoAuto()}
	child := hwUnwrap(taffy.NewLeaf(cs))
	rs := defaultStyle()
	rs.Display = display
	rs.Size = hwSizeDim(hwDim(100.0), hwDim(100.0))
	root := hwUnwrap(taffy.NewWithChildren(rs, []NodeID{child}))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	return taffy, child
}

func TestHandWrittenPositionPositionVariants_StaticAndStickyIgnoreInset(t *testing.T) {
	for _, display := range []Display{DisplayBlock, DisplayFlex, DisplayGrid} {
		for _, position := range []Position{PositionStatic, PositionSticky} {
			taffy, child := positionVariantsLeafWithInset(position, display)
			layout := hwLayout(taffy, child)
			hwAssertEqMsg(t, layout.Location, Point[float32]{X: 0.0, Y: 0.0}, positionVariantsMsg(position, display))
		}
		taffy, child := positionVariantsLeafWithInset(PositionRelative, display)
		hwAssertEqMsg(t, hwLayout(taffy, child).Location, Point[float32]{X: 5.0, Y: 7.0}, "Relative in "+display.String())
	}
}

func positionVariantsMsg(position Position, display Display) string {
	names := map[Position]string{PositionStatic: "Static", PositionSticky: "Sticky"}
	return names[position] + " in " + display.String()
}

func TestHandWrittenPositionPositionVariants_StickyIsPositionedButInFlow(t *testing.T) {
	if !PositionSticky.IsPositioned() {
		t.Fatal("assertion failed: Position::Sticky.is_positioned()")
	}
	if PositionSticky.IsOutOfFlow() {
		t.Fatal("assertion failed: !Position::Sticky.is_out_of_flow()")
	}
	if PositionStatic.IsPositioned() {
		t.Fatal("assertion failed: !Position::Static.is_positioned()")
	}
}
