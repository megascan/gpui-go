package taffy

import "testing"

func floatsBlock(width, height float32, float Float) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Float = float
	s.Size = hwSizeDim(hwDim(width), hwDim(height))
	return s
}

func floatsRootStyle(width float32) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayBlock
	s.Size = hwSizeDim(hwDim(width), DimensionAuto())
	return s
}

func TestHandWrittenFloats_FloatNoHigherThanEarlierFloats(t *testing.T) {
	taffy := newTestTree()

	spacer := hwUnwrap(taffy.NewLeaf(floatsBlock(400.0, 43.2, FloatNone)))
	floatA := hwUnwrap(taffy.NewLeaf(floatsBlock(258.0, 42.0, FloatLeft)))
	floatB := hwUnwrap(taffy.NewLeaf(floatsBlock(298.0, 42.0, FloatLeft)))
	floatC := hwUnwrap(taffy.NewLeaf(floatsBlock(98.0, 42.0, FloatRight)))

	root := hwUnwrap(taffy.NewWithChildren(floatsRootStyle(400.0), []NodeID{spacer, floatA, floatB, floatC}))

	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	hwAssertEq(t, hwLayout(taffy, floatA).Location, Point[float32]{X: 0.0, Y: 43.0})
	hwAssertEq(t, hwLayout(taffy, floatB).Location, Point[float32]{X: 0.0, Y: 85.0})
	hwAssertEq(t, hwLayout(taffy, floatC).Location, Point[float32]{X: 302.0, Y: 85.0})
}

func TestHandWrittenFloats_ClearPastZeroWidthFloat(t *testing.T) {
	taffy := newTestTree()

	fs := defaultStyle()
	fs.Display = DisplayBlock
	fs.Float = FloatLeft
	fs.Size = hwSizeDim(DimensionAuto(), hwDim(1.0))
	floated := hwUnwrap(taffy.NewLeaf(fs))
	cs := defaultStyle()
	cs.Display = DisplayBlock
	cs.Clear = ClearLeft
	cleared := hwUnwrap(taffy.NewLeaf(cs))
	ss := defaultStyle()
	ss.Display = DisplayBlock
	margin := RectZero[LengthPercentageAuto]()
	margin.Top = hwLPA(99.0)
	ss.Margin = margin
	sibling := hwUnwrap(taffy.NewLeaf(ss))

	container := hwUnwrap(taffy.NewWithChildren(floatsRootStyle(100.0), []NodeID{floated, cleared, sibling}))
	rs := defaultStyle()
	rs.Display = DisplayBlock
	root := hwUnwrap(taffy.NewWithChildren(rs, []NodeID{container}))

	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	hwAssertEq(t, hwLayout(taffy, container).Size.Height, 100.0)
}

func TestHandWrittenFloats_OversizedFloatOverflowsContainingBlock(t *testing.T) {
	taffy := newTestTree()

	floatA := hwUnwrap(taffy.NewLeaf(floatsBlock(150.0, 50.0, FloatLeft)))
	root := hwUnwrap(taffy.NewWithChildren(floatsRootStyle(100.0), []NodeID{floatA}))

	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	hwAssertEq(t, hwLayout(taffy, floatA).Location, Point[float32]{X: 0.0, Y: 0.0})
}

func TestHandWrittenFloats_FloatBesideExistingFloatMovesDownInsteadOfOverflowing(t *testing.T) {
	taffy := newTestTree()

	left := hwUnwrap(taffy.NewLeaf(floatsBlock(60.0, 50.0, FloatLeft)))
	right := hwUnwrap(taffy.NewLeaf(floatsBlock(60.0, 50.0, FloatRight)))
	left2 := hwUnwrap(taffy.NewLeaf(floatsBlock(60.0, 50.0, FloatLeft)))

	root := hwUnwrap(taffy.NewWithChildren(floatsRootStyle(100.0), []NodeID{left, right, left2}))

	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	hwAssertEq(t, hwLayout(taffy, left).Location, Point[float32]{X: 0.0, Y: 0.0})
	hwAssertEq(t, hwLayout(taffy, right).Location, Point[float32]{X: 40.0, Y: 50.0})
	hwAssertEq(t, hwLayout(taffy, left2).Location, Point[float32]{X: 0.0, Y: 100.0})
}

func floatsAutoOrZero(isAuto bool) LengthPercentageAuto {
	if isAuto {
		return LengthPercentageAutoAuto()
	}
	return LengthPercentageAutoZero
}

func floatsU8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func TestHandWrittenFloats_AutoMarginsIncludeFloatAvoidanceOffsets(t *testing.T) {
	for _, direction := range []Direction{DirectionLtr, DirectionRtl} {
		for _, widths := range [][2]float32{{0.0, 20.0}, {20.0, 0.0}, {20.0, 20.0}, {0.0, 0.0}} {
			leftFloatWidth, rightFloatWidth := widths[0], widths[1]
			taffy := newTestTree()
			leftFloat := hwUnwrap(taffy.NewLeaf(floatsBlock(leftFloatWidth, 40.0, FloatLeft)))
			rightFloat := hwUnwrap(taffy.NewLeaf(floatsBlock(rightFloatWidth, 40.0, FloatRight)))
			autoSides := [][2]bool{{true, true}, {true, false}, {false, true}}
			boxes := make([]NodeID, 0, len(autoSides))
			for _, sides := range autoSides {
				s := defaultStyle()
				s.Display = DisplayFlowRoot
				s.Direction = direction
				s.Size = hwSizeDim(hwDim(40.0), hwDim(10.0))
				s.Margin = Rect[LengthPercentageAuto]{
					Left:   floatsAutoOrZero(sides[0]),
					Right:  floatsAutoOrZero(sides[1]),
					Top:    LengthPercentageAutoAuto(),
					Bottom: LengthPercentageAutoAuto(),
				}
				boxes = append(boxes, hwUnwrap(taffy.NewLeaf(s)))
			}
			children := append([]NodeID{leftFloat, rightFloat}, boxes...)
			rs := defaultStyle()
			rs.Display = DisplayFlowRoot
			rs.Direction = direction
			rs.Size = hwSizeDim(hwDim(100.0), DimensionAuto())
			rs.Padding = Rect[LengthPercentage]{Left: LengthPercentageLength(5.0), Right: LengthPercentageLength(5.0), Top: LengthPercentageLength(5.0), Bottom: LengthPercentageLength(5.0)}
			root := hwUnwrap(taffy.NewWithChildren(rs, children))

			hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

			for index, node := range boxes {
				leftAuto, rightAuto := autoSides[index][0], autoSides[index][1]
				layout := hwLayout(taffy, node)
				freeSpace := float32(90.0) - leftFloatWidth - rightFloatWidth - 40.0
				alignmentMargin := freeSpace / float32(floatsU8(leftAuto)+floatsU8(rightAuto))
				var leftGap float32
				if leftAuto {
					leftGap = alignmentMargin
				}
				hwAssertEq(t, layout.Location, Point[float32]{X: 5.0 + leftFloatWidth + leftGap, Y: 5.0 + float32(index)*10.0})
				hwAssertEq(t, layout.Size, Size[float32]{Width: 40.0, Height: 10.0})
				var wantLeft, wantRight float32
				if leftAuto {
					wantLeft = leftFloatWidth + alignmentMargin
				}
				if rightAuto {
					wantRight = rightFloatWidth + alignmentMargin
				}
				hwAssertEq(t, layout.Margin.Left, wantLeft)
				hwAssertEq(t, layout.Margin.Right, wantRight)
				hwAssertEq(t, layout.Margin.Top, 0.0)
				hwAssertEq(t, layout.Margin.Bottom, 0.0)
			}
		}
	}
}

func TestHandWrittenFloats_FloatAvoidingAutoMarginsInNestedBlockIgnoreRelativeInsetsAndClearance(t *testing.T) {
	for _, direction := range []Direction{DirectionLtr, DirectionRtl} {
		taffy := newTestTree()
		leftFloat := hwUnwrap(taffy.NewLeaf(floatsBlock(40.0, 40.0, FloatLeft)))
		rightFloat := hwUnwrap(taffy.NewLeaf(floatsBlock(60.0, 40.0, FloatRight)))
		boxStyle := defaultStyle()
		boxStyle.Display = DisplayFlowRoot
		boxStyle.Direction = direction
		boxStyle.Size = hwSizeDim(hwDim(40.0), hwDim(10.0))
		boxStyle.Margin = Rect[LengthPercentageAuto]{Left: LengthPercentageAutoAuto(), Right: LengthPercentageAutoAuto(), Top: LengthPercentageAutoZero, Bottom: LengthPercentageAutoZero}
		rel := boxStyle
		rel.Position = PositionRelative
		inset := RectAuto[LengthPercentageAuto]()
		inset.Left = hwLPA(7.0)
		rel.Inset = inset
		relativeBox := hwUnwrap(taffy.NewLeaf(rel))
		clr := boxStyle
		clr.Clear = ClearBoth
		clearedBox := hwUnwrap(taffy.NewLeaf(clr))
		ns := defaultStyle()
		ns.Display = DisplayBlock
		ns.Direction = direction
		nm := RectZero[LengthPercentageAuto]()
		nm.Left, nm.Right = hwLPA(20.0), hwLPA(30.0)
		ns.Margin = nm
		np := RectZero[LengthPercentage]()
		np.Left, np.Right = LengthPercentageLength(5.0), LengthPercentageLength(7.0)
		ns.Padding = np
		nb := RectZero[LengthPercentage]()
		nb.Left, nb.Right = LengthPercentageLength(3.0), LengthPercentageLength(5.0)
		ns.Border = nb
		nested := hwUnwrap(taffy.NewWithChildren(ns, []NodeID{relativeBox, clearedBox}))
		rs := defaultStyle()
		rs.Display = DisplayFlowRoot
		rs.Direction = direction
		rs.Size = hwSizeDim(hwDim(200.0), DimensionAuto())
		rs.Padding = Rect[LengthPercentage]{Left: LengthPercentageLength(10.0), Right: LengthPercentageLength(10.0), Top: LengthPercentageLength(10.0), Bottom: LengthPercentageLength(10.0)}
		root := hwUnwrap(taffy.NewWithChildren(rs, []NodeID{leftFloat, rightFloat, nested}))

		hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

		relativeLayout := hwLayout(taffy, relativeBox)
		hwAssertEq(t, relativeLayout.Location, Point[float32]{X: 47.0, Y: 0.0})
		hwAssertEq(t, relativeLayout.Margin.Left, 32.0)
		hwAssertEq(t, relativeLayout.Margin.Right, 38.0)
		clearedLayout := hwLayout(taffy, clearedBox)
		hwAssertEq(t, clearedLayout.Location, Point[float32]{X: 43.0, Y: 40.0})
		hwAssertEq(t, clearedLayout.Margin.Left, 35.0)
		hwAssertEq(t, clearedLayout.Margin.Right, 35.0)
		nestedLayout := hwLayout(taffy, nested)
		hwAssertEq(t, nestedLayout.Location, Point[float32]{X: 30.0, Y: 10.0})
		hwAssertEq(t, nestedLayout.Margin.Left, 20.0)
		hwAssertEq(t, nestedLayout.Margin.Right, 30.0)
	}
}
