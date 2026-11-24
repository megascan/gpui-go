package taffy

import (
	"slices"
	"testing"
)

func oofHoistingLeafStyle(width, height float32) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Size = hwSizeDim(hwDim(width), hwDim(height))
	return s
}

func oofHoistingInset(left, top float32) Rect[LengthPercentageAuto] {
	return Rect[LengthPercentageAuto]{Left: hwLPA(left), Top: hwLPA(top), Right: LengthPercentageAutoAuto(), Bottom: LengthPercentageAutoAuto()}
}

func oofHoistingAssertIDs(t *testing.T, left, right []NodeID, msg string) {
	t.Helper()
	if !slices.Equal(left, right) {
		t.Fatalf("assertion `left == right` failed: %s\n  left: %v\n right: %v", msg, left, right)
	}
}

func TestHandWrittenOofHoisting_AbsoluteSkipsStaticParent(t *testing.T) {
	for _, display := range []Display{DisplayBlock, DisplayFlex, DisplayGrid} {
		tree := NewTaffyTree[struct{}]()
		as := defaultStyle()
		as.Position = PositionAbsolute
		as.Inset = oofHoistingInset(10.0, 20.0)
		as.Size = hwSizeDim(hwDim(30.0), hwDim(30.0))
		abs := hwUnwrap(tree.NewLeaf(as))
		ps := defaultStyle()
		ps.Display = display
		ps.Margin = oofHoistingInset(40.0, 40.0)
		ps.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
		staticParent := hwUnwrap(tree.NewWithChildren(ps, []NodeID{abs}))
		cs := defaultStyle()
		cs.Display = display
		cs.Position = PositionRelative
		cs.Size = hwSizeDim(hwDim(200.0), hwDim(200.0))
		cb := hwUnwrap(tree.NewWithChildren(cs, []NodeID{staticParent}))

		hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))

		layout := hwLayout(tree, abs)
		hwAssertEqMsg(t, layout.Location, Point[float32]{X: 10.0, Y: 20.0}, display.String())
		hwAssertEqMsg(t, layout.Size, Size[float32]{Width: 30.0, Height: 30.0}, display.String())
	}
}

func TestHandWrittenOofHoisting_HoistedChildrenAreInDocumentOrder(t *testing.T) {
	for _, display := range []Display{DisplayBlock, DisplayFlex, DisplayGrid} {
		tree := NewTaffyTree[struct{}]()
		absStyle := defaultStyle()
		absStyle.Position = PositionAbsolute
		absStyle.Size = hwSizeDim(hwDim(10.0), hwDim(10.0))
		inflowStyle := defaultStyle()
		inflowStyle.Display = display
		inflowStyle.Size = hwSizeDim(hwDim(20.0), hwDim(20.0))

		directA := hwUnwrap(tree.NewLeaf(absStyle))
		nestedB := hwUnwrap(tree.NewLeaf(absStyle))
		staticB := hwUnwrap(tree.NewWithChildren(inflowStyle, []NodeID{nestedB}))
		nestedC := hwUnwrap(tree.NewLeaf(absStyle))
		staticC := hwUnwrap(tree.NewWithChildren(inflowStyle, []NodeID{nestedC}))
		directD := hwUnwrap(tree.NewLeaf(absStyle))
		nestedE := hwUnwrap(tree.NewLeaf(absStyle))
		staticE := hwUnwrap(tree.NewWithChildren(inflowStyle, []NodeID{nestedE}))
		cs := defaultStyle()
		cs.Display = display
		cs.Position = PositionRelative
		cs.FlexWrap = FlexWrapWrapReverse
		cs.FlexDirection = FlexDirectionRowReverse
		cs.Size = hwSizeDim(hwDim(50.0), hwDim(200.0))
		cb := hwUnwrap(tree.NewWithChildren(cs, []NodeID{directA, staticB, staticC, directD, staticE}))

		hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))

		oofHoistingAssertIDs(t, hwUnwrap(tree.HoistedChildren(cb)), []NodeID{directA, nestedB, nestedC, directD, nestedE}, display.String())
	}
}

func TestHandWrittenOofHoisting_StaticPositionIsContainingBlockRelative(t *testing.T) {
	for _, display := range []Display{DisplayBlock, DisplayFlex, DisplayGrid} {
		tree := NewTaffyTree[struct{}]()
		as := defaultStyle()
		as.Position = PositionAbsolute
		as.Size = hwSizeDim(hwDim(30.0), hwDim(30.0))
		abs := hwUnwrap(tree.NewLeaf(as))
		sibling := hwUnwrap(tree.NewLeaf(oofHoistingLeafStyle(50.0, 50.0)))
		ps := defaultStyle()
		ps.Display = display
		ps.Margin = oofHoistingInset(40.0, 40.0)
		ps.Size = hwSizeDim(hwDim(100.0), hwDim(100.0))
		staticParent := hwUnwrap(tree.NewWithChildren(ps, []NodeID{sibling, abs}))
		cs := defaultStyle()
		cs.Display = display
		cs.Position = PositionRelative
		cs.Size = hwSizeDim(hwDim(200.0), hwDim(200.0))
		cb := hwUnwrap(tree.NewWithChildren(cs, []NodeID{staticParent}))

		hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))

		layout := hwLayout(tree, abs)
		var expected Point[float32]
		switch display {
		case DisplayBlock:
			expected = Point[float32]{X: 40.0, Y: 90.0}
		case DisplayFlex:
			expected = Point[float32]{X: 40.0, Y: 40.0}
		case DisplayGrid:
			expected = Point[float32]{X: 40.0, Y: 40.0}
		default:
			panic("internal error: entered unreachable code")
		}
		hwAssertEqMsg(t, layout.Location, expected, display.String())
	}
}

func TestHandWrittenOofHoisting_FixedHoistsToRoot(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	fs := defaultStyle()
	fs.Position = PositionFixed
	fs.Inset = oofHoistingInset(5.0, 5.0)
	fs.Size = hwSizeDim(hwDim(10.0), hwDim(10.0))
	fixed := hwUnwrap(tree.NewLeaf(fs))
	ps := defaultStyle()
	ps.Position = PositionRelative
	ps.Margin = oofHoistingInset(50.0, 50.0)
	ps.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
	positionedParent := hwUnwrap(tree.NewWithChildren(ps, []NodeID{fixed}))
	root := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(200.0, 200.0), []NodeID{positionedParent}))

	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	hwAssertEq(t, hwLayout(tree, fixed).Location, Point[float32]{X: 5.0, Y: 5.0})
}

func TestHandWrittenOofHoisting_AbsoluteWithinAbsolute(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	is := defaultStyle()
	is.Position = PositionAbsolute
	is.Inset = Rect[LengthPercentageAuto]{Right: hwLPA(0.0), Bottom: hwLPA(0.0), Left: LengthPercentageAutoAuto(), Top: LengthPercentageAutoAuto()}
	is.Size = hwSizeDim(hwDim(10.0), hwDim(10.0))
	inner := hwUnwrap(tree.NewLeaf(is))
	os := defaultStyle()
	os.Position = PositionAbsolute
	os.Inset = oofHoistingInset(20.0, 20.0)
	os.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
	outer := hwUnwrap(tree.NewWithChildren(os, []NodeID{inner}))
	root := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(200.0, 200.0), []NodeID{outer}))

	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	hwAssertEq(t, hwLayout(tree, outer).Location, Point[float32]{X: 20.0, Y: 20.0})
	hwAssertEq(t, hwLayout(tree, inner).Location, Point[float32]{X: 40.0, Y: 40.0})
}

func TestHandWrittenOofHoisting_FixedWithinAbsoluteHoistsToRoot(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	fs := defaultStyle()
	fs.Position = PositionFixed
	fs.Inset = oofHoistingInset(1.0, 2.0)
	fs.Size = hwSizeDim(hwDim(10.0), hwDim(10.0))
	fixed := hwUnwrap(tree.NewLeaf(fs))
	as := defaultStyle()
	as.Position = PositionAbsolute
	as.Inset = oofHoistingInset(20.0, 20.0)
	as.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
	abs := hwUnwrap(tree.NewWithChildren(as, []NodeID{fixed}))
	cs := defaultStyle()
	cs.Position = PositionRelative
	cs.Margin = oofHoistingInset(30.0, 30.0)
	cs.Size = hwSizeDim(hwDim(100.0), hwDim(100.0))
	cb := hwUnwrap(tree.NewWithChildren(cs, []NodeID{abs}))
	root := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(200.0, 200.0), []NodeID{cb}))

	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 20.0, Y: 20.0})
	hwAssertEq(t, hwLayout(tree, fixed).Location, Point[float32]{X: 1.0, Y: 2.0})
}

func TestHandWrittenOofHoisting_HoistingSurvivesCacheHits(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	as := defaultStyle()
	as.Position = PositionAbsolute
	as.Inset = oofHoistingInset(10.0, 20.0)
	as.Size = hwSizeDim(hwDim(30.0), hwDim(30.0))
	abs := hwUnwrap(tree.NewLeaf(as))
	staticAncestor := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(50.0, 50.0), []NodeID{abs}))
	for range 3 {
		staticAncestor = hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(50.0, 50.0), []NodeID{staticAncestor}))
	}
	cs := defaultStyle()
	cs.Position = PositionRelative
	cs.Size = hwSizeDim(hwDim(200.0), hwDim(200.0))
	cb := hwUnwrap(tree.NewWithChildren(cs, []NodeID{staticAncestor}))

	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 10.0, Y: 20.0})

	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 10.0, Y: 20.0})

	hwOK(tree.MarkDirty(cb))
	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 10.0, Y: 20.0})

	newStyle := *hwUnwrap(tree.Style(abs))
	newStyle.Inset.Left = hwLPA(15.0)
	hwOK(tree.SetStyle(abs, newStyle))
	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 15.0, Y: 20.0})
}

func TestHandWrittenOofHoisting_CachedRootDoesNotDuplicateHoistedChildren(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	fs := defaultStyle()
	fs.Position = PositionFixed
	fs.Size = hwSizeDim(hwDim(30.0), hwDim(30.0))
	fixed := hwUnwrap(tree.NewLeaf(fs))
	inner := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(50.0, 50.0), []NodeID{fixed}))
	root := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(200.0, 200.0), []NodeID{inner}))

	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	oofHoistingAssertIDs(t, hwUnwrap(tree.HoistedChildren(root)), []NodeID{fixed}, "")

	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	oofHoistingAssertIDs(t, hwUnwrap(tree.HoistedChildren(root)), []NodeID{fixed}, "")

	hwOK(tree.MarkDirty(root))
	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	oofHoistingAssertIDs(t, hwUnwrap(tree.HoistedChildren(root)), []NodeID{fixed}, "")
}

func TestHandWrittenOofHoisting_RoundingUsesContainingBlockOffsets(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	tree.EnableRounding()
	as := defaultStyle()
	as.Position = PositionAbsolute
	as.Inset = oofHoistingInset(10.6, 10.6)
	as.Size = hwSizeDim(hwDim(10.0), hwDim(10.0))
	abs := hwUnwrap(tree.NewLeaf(as))
	staticParent := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(50.0, 50.0), []NodeID{abs}))
	cs := defaultStyle()
	cs.Position = PositionRelative
	cs.Margin = oofHoistingInset(20.3, 20.3)
	cs.Size = hwSizeDim(hwDim(100.0), hwDim(100.0))
	cb := hwUnwrap(tree.NewWithChildren(cs, []NodeID{staticParent}))
	root := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(200.0, 200.0), []NodeID{cb}))

	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	hwAssertEq(t, hwLayout(tree, cb).Location, Point[float32]{X: 20.0, Y: 20.0})
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 11.0, Y: 11.0})
}

func TestHandWrittenOofHoisting_ClaimChangeRelayoutsWithHotCache(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	as := defaultStyle()
	as.Position = PositionAbsolute
	as.Inset = oofHoistingInset(10.0, 20.0)
	as.Size = hwSizeDim(hwDim(30.0), hwDim(30.0))
	abs := hwUnwrap(tree.NewLeaf(as))
	middleStyle := defaultStyle()
	middleStyle.Margin = oofHoistingInset(40.0, 40.0)
	middleStyle.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
	middle := hwUnwrap(tree.NewWithChildren(middleStyle, []NodeID{abs}))
	cs := defaultStyle()
	cs.Position = PositionRelative
	cs.Size = hwSizeDim(hwDim(200.0), hwDim(200.0))
	cb := hwUnwrap(tree.NewWithChildren(cs, []NodeID{middle}))

	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 10.0, Y: 20.0})

	positionedMiddle := middleStyle
	positionedMiddle.Position = PositionRelative
	hwOK(tree.SetStyle(middle, positionedMiddle))
	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 10.0, Y: 20.0})
	hwAssertEq(t, hwLayout(tree, middle).Location, Point[float32]{X: 40.0, Y: 40.0})

	hwOK(tree.SetStyle(middle, middleStyle))
	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 10.0, Y: 20.0})
	hwAssertEq(t, hwLayout(tree, middle).Location, Point[float32]{X: 40.0, Y: 40.0})
}

func TestHandWrittenOofHoisting_FixedClaimChangeRelayoutsWithHotCache(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	fs := defaultStyle()
	fs.Position = PositionFixed
	fs.Inset = oofHoistingInset(5.0, 5.0)
	fs.Size = hwSizeDim(hwDim(10.0), hwDim(10.0))
	fixed := hwUnwrap(tree.NewLeaf(fs))
	absStyle := defaultStyle()
	absStyle.Position = PositionAbsolute
	absStyle.Inset = oofHoistingInset(50.0, 50.0)
	absStyle.Size = hwSizeDim(hwDim(80.0), hwDim(80.0))
	abs := hwUnwrap(tree.NewWithChildren(absStyle, []NodeID{fixed}))
	root := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(200.0, 200.0), []NodeID{abs}))

	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, fixed).Location, Point[float32]{X: 5.0, Y: 5.0})
	hwAssertEq(t, hwLayout(tree, abs).Location, Point[float32]{X: 50.0, Y: 50.0})

	newFixedStyle := *hwUnwrap(tree.Style(fixed))
	newFixedStyle.Inset.Left = hwLPA(7.0)
	hwOK(tree.SetStyle(fixed, newFixedStyle))
	hwOK(tree.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, fixed).Location, Point[float32]{X: 7.0, Y: 5.0})
}

func TestHandWrittenOofHoisting_ContentChangeInsideHoistedSubtree(t *testing.T) {
	tree := NewTaffyTree[struct{}]()
	inner := hwUnwrap(tree.NewLeaf(oofHoistingLeafStyle(10.0, 10.0)))
	as := defaultStyle()
	as.Position = PositionAbsolute
	as.Inset = oofHoistingInset(10.0, 10.0)
	abs := hwUnwrap(tree.NewWithChildren(as, []NodeID{inner}))
	staticParent := hwUnwrap(tree.NewWithChildren(oofHoistingLeafStyle(50.0, 50.0), []NodeID{abs}))
	cs := defaultStyle()
	cs.Position = PositionRelative
	cs.Size = hwSizeDim(hwDim(200.0), hwDim(200.0))
	cb := hwUnwrap(tree.NewWithChildren(cs, []NodeID{staticParent}))

	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Size, Size[float32]{Width: 10.0, Height: 10.0})

	hwOK(tree.SetStyle(inner, oofHoistingLeafStyle(20.0, 30.0)))
	hwOK(tree.ComputeLayout(cb, SizeMaxContent[AvailableSpace]()))
	hwAssertEq(t, hwLayout(tree, abs).Size, Size[float32]{Width: 20.0, Height: 30.0})
}
