package taffy

import "testing"

func borderAndPaddingArrToRect[T any](items [4]T) Rect[T] {
	return Rect[T]{Left: items[0], Right: items[1], Top: items[2], Bottom: items[3]}
}

func borderAndPaddingSingleSide(i int) Rect[LengthPercentage] {
	var lengths [4]LengthPercentage
	for j := range lengths {
		lengths[j] = LengthPercentageZero
	}
	lengths[i] = LengthPercentage{}.FromLength(10.)
	return borderAndPaddingArrToRect(lengths)
}

func TestHandWrittenBorderAndPadding_BorderOnASingleAxisDoesntIncreaseSize(t *testing.T) {
	t.Skip("#[ignore]")
	for i := range 4 {
		taffy := newTestTree()
		s := defaultStyle()
		s.Border = borderAndPaddingSingleSide(i)
		node := hwUnwrap(taffy.NewLeaf(s))

		hwOK(taffy.ComputeLayout(node, hwDefinite(100.0, 100.0)))

		layout := hwLayout(taffy, node)
		hwAssertEq(t, layout.Size.Width*layout.Size.Height, 0.)
	}
}

func TestHandWrittenBorderAndPadding_PaddingOnASingleAxisDoesntIncreaseSize(t *testing.T) {
	t.Skip("#[ignore]")
	for i := range 4 {
		taffy := newTestTree()
		s := defaultStyle()
		s.Padding = borderAndPaddingSingleSide(i)
		node := hwUnwrap(taffy.NewLeaf(s))

		hwOK(taffy.ComputeLayout(node, hwDefinite(100.0, 100.0)))

		layout := hwLayout(taffy, node)
		hwAssertEq(t, layout.Size.Width*layout.Size.Height, 0.)
	}
}

func TestHandWrittenBorderAndPadding_BorderAndPaddingOnASingleAxisDoesntIncreaseSize(t *testing.T) {
	t.Skip("#[ignore]")
	for i := range 4 {
		taffy := newTestTree()
		rect := borderAndPaddingSingleSide(i)
		s := defaultStyle()
		s.Border = rect
		s.Padding = rect
		node := hwUnwrap(taffy.NewLeaf(s))

		hwOK(taffy.ComputeLayout(node, hwDefinite(100.0, 100.0)))
		layout := hwLayout(taffy, node)
		hwAssertEq(t, layout.Size.Width*layout.Size.Height, 0.)
	}
}

func TestHandWrittenBorderAndPadding_VerticalBorderAndPaddingPercentageValuesUseAvailableSpaceCorrectly(t *testing.T) {
	t.Skip("#[ignore]")
	taffy := newTestTree()

	s := defaultStyle()
	padding := RectZero[LengthPercentage]()
	padding.Left = LengthPercentage{}.FromPercent(1.0)
	padding.Top = LengthPercentage{}.FromPercent(1.0)
	s.Padding = padding
	node := hwUnwrap(taffy.NewLeaf(s))

	hwOK(taffy.ComputeLayout(node, hwDefinite(200.0, 100.0)))

	layout := hwLayout(taffy, node)
	hwAssertEq(t, layout.Size.Width, 200.0)
	hwAssertEq(t, layout.Size.Height, 200.0)
}
