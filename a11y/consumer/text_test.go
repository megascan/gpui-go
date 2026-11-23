package consumer

import (
	"slices"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

var (
	testTextDecoration1 = a11y.TextDecoration{Style: a11y.TextDecorationStyleSolid, Color: a11y.Color{Red: 0, Green: 0, Blue: 0, Alpha: 255}}
	testTextDecoration2 = a11y.TextDecoration{Style: a11y.TextDecorationStyleDotted, Color: a11y.Color{Red: 255, Green: 0, Blue: 0, Alpha: 255}}
	testTextDecoration3 = a11y.TextDecoration{Style: a11y.TextDecorationStyleDashed, Color: a11y.Color{Red: 0, Green: 255, Blue: 0, Alpha: 255}}
	testTextDecoration4 = a11y.TextDecoration{Style: a11y.TextDecorationStyleDouble, Color: a11y.Color{Red: 0, Green: 0, Blue: 255, Alpha: 255}}
)

func ones(n int) []uint8 {
	s := make([]uint8, n)
	for i := range s {
		s[i] = 1
	}
	return s
}

func widths(n int, last float32) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = 7.58557
	}
	if n > 0 && last >= 0 {
		s[n-1] = last
	}
	return s
}

var standardPositions = []float32{
	0.0, 7.3333335, 14.666667, 22.0, 29.333334, 36.666668, 44.0, 51.333332,
	58.666668, 66.0, 73.333336, 80.666664, 88.0, 95.333336, 102.666664, 110.0,
	117.333336, 124.666664, 132.0, 139.33333, 146.66667, 154.0, 161.33333,
	168.66667, 176.0, 183.33333, 190.66667, 198.0, 205.33333, 212.66667, 220.0,
	227.33333, 234.66667, 242.0, 249.33333, 256.66666, 264.0, 271.33334,
}

func mainMultilineTree(selection gpuiutil.Option[a11y.TextSelection]) *Tree {
	window := node(a11y.RoleWindow, 1)
	window.SetTransform(a11y.AffineScale(1.5))

	input := node(a11y.RoleMultilineTextInput, 2, 3, 4, 5, 6, 7, 8, 9)
	input.SetBounds(a11y.Rect{X0: 8.0, Y0: 31.666664123535156, X1: 296.0, Y1: 123.66666412353516})
	input.AddAction(a11y.ActionFocus)
	input.SetTextDirection(a11y.TextDirectionLeftToRight)
	if s, ok := selection.Get(); ok {
		input.SetTextSelection(s)
	}

	n2 := node(a11y.RoleTextRun)
	n2.SetBounds(a11y.Rect{X0: 12.0, Y0: 33.666664123535156, X1: 290.9189147949219, Y1: 48.33333206176758})
	n2.SetValue("This paragraph is long enough to wrap ")
	l2 := ones(38)
	l2[17] = 2
	n2.SetCharacterLengths(l2)
	n2.SetCharacterPositions(standardPositions)
	n2.SetCharacterWidths(widths(38, -1))
	n2.SetWordStarts([]uint8{5, 15, 18, 23, 30, 33})

	n3 := node(a11y.RoleTextRun)
	n3.SetBounds(a11y.Rect{X0: 12.0, Y0: 48.33333206176758, X1: 34.252257, Y1: 63.0})
	n3.SetValue("to ")
	n3.SetCharacterLengths([]uint8{1, 1, 1})
	n3.SetCharacterPositions([]float32{0.0, 7.3333435, 14.666687})
	n3.SetCharacterWidths([]float32{7.58557, 7.58557, 7.58557})
	n3.SetWordStarts([]uint8{0})
	n3.SetNextOnLine(4)

	n4 := node(a11y.RoleTextRun)
	n4.SetBounds(a11y.Rect{X0: 34.0, Y0: 48.33333206176758, X1: 85.58557, Y1: 63.0})
	n4.SetValue("another")
	n4.SetCharacterLengths(ones(7))
	n4.SetCharacterPositions([]float32{0.0, 7.333344, 14.666687, 22.0, 29.333344, 36.666687, 44.0})
	n4.SetCharacterWidths(widths(7, -1))
	n4.SetWordStarts([]uint8{0})
	n4.SetUnderline(testTextDecoration1)
	n4.SetPreviousOnLine(3)
	n4.SetNextOnLine(5)

	n5 := node(a11y.RoleTextRun)
	n5.SetBounds(a11y.Rect{X0: 85.33334, Y0: 48.33333206176758, X1: 129.5855712890625, Y1: 63.0})
	n5.SetValue(" line.\n")
	n5.SetCharacterLengths(ones(7))
	n5.SetCharacterPositions([]float32{0.0, 7.333344, 14.666687, 22.0, 29.333344, 36.666687, 44.25226})
	n5.SetCharacterWidths(widths(7, 0.0))
	n5.SetWordStarts([]uint8{1})
	n5.SetPreviousOnLine(4)

	n6 := node(a11y.RoleTextRun)
	n6.SetBounds(a11y.Rect{X0: 12.0, Y0: 63.0, X1: 144.25222778320313, Y1: 77.66666412353516})
	n6.SetValue("Another paragraph.\n")
	n6.SetCharacterLengths(ones(19))
	p6 := slices.Clone(standardPositions[:19])
	p6[18] = 132.25223
	n6.SetCharacterPositions(p6)
	n6.SetCharacterWidths(widths(19, 0.0))
	n6.SetWordStarts([]uint8{8})

	n7 := node(a11y.RoleTextRun)
	n7.SetBounds(a11y.Rect{X0: 12.0, Y0: 77.66666412353516, X1: 12.0, Y1: 92.33332824707031})
	n7.SetValue("\n")
	n7.SetCharacterLengths([]uint8{1})
	n7.SetCharacterPositions([]float32{0.0})
	n7.SetCharacterWidths([]float32{0.0})

	n8 := node(a11y.RoleTextRun)
	n8.SetBounds(a11y.Rect{X0: 12.0, Y0: 92.33332824707031, X1: 158.9188995361328, Y1: 107.0})
	n8.SetValue("Last non-blank line\U0001f44d\U0001f3fb\n")
	l8 := ones(21)
	l8[19] = 8
	n8.SetCharacterLengths(l8)
	p8 := slices.Clone(standardPositions[:21])
	p8[20] = 146.9189
	n8.SetCharacterPositions(p8)
	n8.SetCharacterWidths(widths(21, 0.0))
	n8.SetWordStarts([]uint8{5, 9, 15})

	n9 := node(a11y.RoleTextRun)
	n9.SetBounds(a11y.Rect{X0: 12.0, Y0: 107.0, X1: 12.0, Y1: 121.66666412353516})
	n9.SetValue("")
	n9.SetCharacterLengths([]uint8{})
	n9.SetCharacterPositions([]float32{})
	n9.SetCharacterWidths([]float32{})

	return NewTree(rootUpdate(1,
		ent(0, window), ent(1, input), ent(2, n2), ent(3, n3), ent(4, n4),
		ent(5, n5), ent(6, n6), ent(7, n7), ent(8, n8), ent(9, n9),
	), true)
}

func caretSelection(id a11y.NodeID, index int) gpuiutil.Option[a11y.TextSelection] {
	pos := a11y.TextPosition{Node: id, CharacterIndex: index}
	return gpuiutil.Some(a11y.TextSelection{Anchor: pos, Focus: pos})
}

func multilineEndSelection() gpuiutil.Option[a11y.TextSelection] { return caretSelection(9, 0) }

func multilinePastEndSelection() gpuiutil.Option[a11y.TextSelection] { return caretSelection(9, 3) }

func multilineWrappedLineEndSelection() gpuiutil.Option[a11y.TextSelection] {
	return caretSelection(2, 38)
}

func multilineFirstLineMiddleSelection() gpuiutil.Option[a11y.TextSelection] {
	return caretSelection(2, 5)
}

func multilineSecondLineMiddleSelection() gpuiutil.Option[a11y.TextSelection] {
	return caretSelection(4, 3)
}

func rects(t *testing.T, want, got []a11y.Rect) {
	t.Helper()
	if !slices.Equal(want, got) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func less(a, b Position) bool {
	c, ok := a.PartialCmp(b)
	return ok && c < 0
}

var (
	rectLine0      = a11y.Rect{X0: 18.0, Y0: 50.499996185302734, X1: 436.3783721923828, Y1: 72.49999809265137}
	rectTo         = a11y.Rect{X0: 18.0, Y0: 72.49999809265137, X1: 51.3783855, Y1: 94.5}
	rectAnother    = a11y.Rect{X0: 51.0, Y0: 72.49999809265137, X1: 128.378355, Y1: 94.5}
	rectLine       = a11y.Rect{X0: 128.00001, Y0: 72.49999809265137, X1: 194.37835693359375, Y1: 94.5}
	rectParagraph2 = a11y.Rect{X0: 18.0, Y0: 94.5, X1: 216.3783416748047, Y1: 116.49999618530273}
	rectBlank      = a11y.Rect{X0: 18.0, Y0: 116.49999618530273, X1: 18.0, Y1: 138.49999237060547}
	rectLast       = a11y.Rect{X0: 18.0, Y0: 138.49999237060547, X1: 238.37834930419922, Y1: 160.5}
)

func TestText_SupportsTextRanges(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	check(t, !byID(tree, 0).SupportsTextRanges())
	check(t, byID(tree, 1).SupportsTextRanges())
}

func TestText_MultilineDocumentRange(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	node := byID(tree, 1)
	rng := node.DocumentRange()
	start := rng.Start()
	check(t, start.IsWordStart())
	check(t, start.IsLineStart())
	check(t, !start.IsLineEnd())
	check(t, start.IsParagraphStart())
	check(t, start.IsDocumentStart())
	check(t, !start.IsDocumentEnd())
	end := rng.End()
	check(t, less(start, end))
	check(t, end.IsWordStart())
	check(t, end.IsLineStart())
	check(t, end.IsLineEnd())
	check(t, end.IsParagraphStart())
	check(t, !end.IsDocumentStart())
	check(t, end.IsDocumentEnd())
	eq(t, "This paragraph is long enough to wrap to another line.\nAnother paragraph.\n\nLast non-blank line\U0001f44d\U0001f3fb\n", rng.Text())
	rects(t, []a11y.Rect{rectLine0, rectTo, rectAnother, rectLine, rectParagraph2, rectBlank, rectLast}, rng.BoundingBoxes())
}

func TestText_MultilineDocumentRangeToFirstFormatChange(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	rng := byID(tree, 1).DocumentRange()
	rng.SetEnd(rng.Start().ForwardToFormatEnd())
	eq(t, "This paragraph is long enough to wrap to ", rng.Text())
	rects(t, []a11y.Rect{rectLine0, rectTo}, rng.BoundingBoxes())
}

func TestText_MultilineDocumentRangeFromLastFormatChange(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	rng := byID(tree, 1).DocumentRange()
	rng.SetStart(rng.End().BackwardToFormatStart())
	eq(t, " line.\nAnother paragraph.\n\nLast non-blank line\U0001f44d\U0001f3fb\n", rng.Text())
	rects(t, []a11y.Rect{rectLine, rectParagraph2, rectBlank, rectLast}, rng.BoundingBoxes())
}

func TestText_MultilineEndDegenerateRange(t *testing.T) {
	tree := mainMultilineTree(multilineEndSelection())
	rng := must(byID(tree, 1).TextSelection())
	check(t, rng.IsDegenerate())
	pos := rng.Start()
	check(t, pos.IsWordStart())
	check(t, pos.IsLineStart())
	check(t, pos.IsLineEnd())
	check(t, pos.IsParagraphStart())
	check(t, !pos.IsDocumentStart())
	check(t, pos.IsDocumentEnd())
	eq(t, "", rng.Text())
	rects(t, []a11y.Rect{{X0: 18.0, Y0: 160.5, X1: 18.0, Y1: 182.49999618530273}}, rng.BoundingBoxes())
}

func TestText_MultilineWrappedLineEndRange(t *testing.T) {
	tree := mainMultilineTree(multilineWrappedLineEndSelection())
	rng := must(byID(tree, 1).TextSelection())
	check(t, rng.IsDegenerate())
	pos := rng.Start()
	check(t, !pos.IsWordStart())
	check(t, !pos.IsLineStart())
	check(t, pos.IsLineEnd())
	check(t, !pos.IsParagraphStart())
	check(t, !pos.IsDocumentStart())
	check(t, !pos.IsDocumentEnd())
	eq(t, "", rng.Text())
	rects(t, []a11y.Rect{{X0: 436.3783721923828, Y0: 50.499996185302734, X1: 436.3783721923828, Y1: 72.49999809265137}}, rng.BoundingBoxes())
	charEndPos := pos.ForwardToCharacterEnd()
	lineStartRange := rng
	lineStartRange.SetEnd(charEndPos)
	check(t, !lineStartRange.IsDegenerate())
	check(t, lineStartRange.Start().IsLineStart())
	eq(t, "t", lineStartRange.Text())
	rects(t, []a11y.Rect{{X0: 18.0, Y0: 72.49999809265137, X1: 29.378354787826538, Y1: 94.5}}, lineStartRange.BoundingBoxes())
	prevCharPos := pos.BackwardToCharacterStart()
	prevCharRange := rng
	prevCharRange.SetStart(prevCharPos)
	check(t, !prevCharRange.IsDegenerate())
	check(t, prevCharRange.End().IsLineEnd())
	eq(t, " ", prevCharRange.Text())
	rects(t, []a11y.Rect{{X0: 425.00001525878906, Y0: 50.499996185302734, X1: 436.3783721923828, Y1: 72.49999809265137}}, prevCharRange.BoundingBoxes())
	check(t, prevCharPos.ForwardToCharacterEnd().IsLineEnd())
	check(t, prevCharPos.ForwardToWordEnd().IsLineEnd())
	check(t, prevCharPos.ForwardToLineEnd().IsLineEnd())
	check(t, prevCharPos.ForwardToCharacterStart().IsLineStart())
	check(t, prevCharPos.ForwardToWordStart().IsLineStart())
	check(t, prevCharPos.ForwardToLineStart().IsLineStart())
}

func TestText_MultilineFindLineEndsFromMiddle(t *testing.T) {
	tree := mainMultilineTree(multilineSecondLineMiddleSelection())
	rng := must(byID(tree, 1).TextSelection())
	check(t, rng.IsDegenerate())
	pos := rng.Start()
	check(t, !pos.IsLineStart())
	check(t, !pos.IsLineEnd())
	check(t, !pos.IsDocumentStart())
	check(t, !pos.IsDocumentEnd())
	lineStart := pos.BackwardToLineStart()
	rng.SetStart(lineStart)
	lineEnd := lineStart.ForwardToLineEnd()
	rng.SetEnd(lineEnd)
	check(t, !rng.IsDegenerate())
	check(t, rng.Start().IsLineStart())
	check(t, rng.End().IsLineEnd())
	eq(t, "to another line.\n", rng.Text())
	rects(t, []a11y.Rect{rectTo, rectAnother, rectLine}, rng.BoundingBoxes())
	check(t, lineStart.ForwardToLineStart().IsLineStart())
}

func TestText_MultilineFindWrappedLineEndsFromMiddle(t *testing.T) {
	tree := mainMultilineTree(multilineFirstLineMiddleSelection())
	rng := must(byID(tree, 1).TextSelection())
	check(t, rng.IsDegenerate())
	pos := rng.Start()
	check(t, !pos.IsLineStart())
	check(t, !pos.IsLineEnd())
	check(t, !pos.IsDocumentStart())
	check(t, !pos.IsDocumentEnd())
	lineStart := pos.BackwardToLineStart()
	rng.SetStart(lineStart)
	lineEnd := lineStart.ForwardToLineEnd()
	rng.SetEnd(lineEnd)
	check(t, !rng.IsDegenerate())
	check(t, rng.Start().IsLineStart())
	check(t, rng.End().IsLineEnd())
	eq(t, "This paragraph is long enough to wrap ", rng.Text())
	rects(t, []a11y.Rect{rectLine0}, rng.BoundingBoxes())
	check(t, lineStart.ForwardToLineStart().IsLineStart())
}

func TestText_MultilineFindParagraphEndsFromMiddle(t *testing.T) {
	tree := mainMultilineTree(multilineSecondLineMiddleSelection())
	rng := must(byID(tree, 1).TextSelection())
	check(t, rng.IsDegenerate())
	pos := rng.Start()
	check(t, !pos.IsParagraphStart())
	check(t, !pos.IsDocumentStart())
	check(t, !pos.IsDocumentEnd())
	paragraphStart := pos.BackwardToParagraphStart()
	rng.SetStart(paragraphStart)
	paragraphEnd := paragraphStart.ForwardToParagraphEnd()
	rng.SetEnd(paragraphEnd)
	check(t, !rng.IsDegenerate())
	check(t, rng.Start().IsParagraphStart())
	check(t, rng.End().IsParagraphEnd())
	eq(t, "This paragraph is long enough to wrap to another line.\n", rng.Text())
	rects(t, []a11y.Rect{rectLine0, rectTo, rectAnother, rectLine}, rng.BoundingBoxes())
	check(t, paragraphStart.ForwardToParagraphStart().IsParagraphStart())
}

func TestText_MultilineFindFormatEndsFromMiddle(t *testing.T) {
	tree := mainMultilineTree(multilineSecondLineMiddleSelection())
	rng := must(byID(tree, 1).TextSelection())
	check(t, rng.IsDegenerate())
	pos := rng.Start()
	check(t, !pos.IsFormatStart())
	check(t, !pos.IsDocumentStart())
	check(t, !pos.IsDocumentEnd())
	formatStart := pos.BackwardToFormatStart()
	rng.SetStart(formatStart)
	formatEnd := pos.ForwardToFormatEnd()
	rng.SetEnd(formatEnd)
	check(t, !rng.IsDegenerate())
	eq(t, "another", rng.Text())
	rects(t, []a11y.Rect{rectAnother}, rng.BoundingBoxes())
}

func TestText_MultilineFindWordEndsFromMiddle(t *testing.T) {
	tree := mainMultilineTree(multilineSecondLineMiddleSelection())
	rng := must(byID(tree, 1).TextSelection())
	check(t, rng.IsDegenerate())
	pos := rng.Start()
	check(t, !pos.IsWordStart())
	check(t, !pos.IsDocumentStart())
	check(t, !pos.IsDocumentEnd())
	wordStart := pos.BackwardToWordStart()
	rng.SetStart(wordStart)
	wordEnd := wordStart.ForwardToWordEnd()
	wordEnd2 := pos.ForwardToWordEnd()
	check(t, wordEnd.Equal(wordEnd2))
	wordStart2 := wordEnd.BackwardToWordStart()
	check(t, wordStart.Equal(wordStart2))
	rng.SetEnd(wordEnd)
	check(t, !rng.IsDegenerate())
	eq(t, "another ", rng.Text())
	rects(t, []a11y.Rect{rectAnother, {X0: 128.00001, Y0: 72.49999809265137, X1: 139.37836478782654, Y1: 94.5}}, rng.BoundingBoxes())
}

func TestText_TextPositionAtPoint(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	node := byID(tree, 1)

	check(t, node.TextPositionAtPoint(a11y.NewPoint(8.0, 31.666664123535156)).IsDocumentStart())
	check(t, node.TextPositionAtPoint(a11y.NewPoint(12.0, 33.666664123535156)).IsDocumentStart())
	check(t, node.TextPositionAtPoint(a11y.NewPoint(16.0, 40.0)).IsDocumentStart())

	for _, x := range []float64{144.0, 150.0} {
		pos := node.TextPositionAtPoint(a11y.NewPoint(x, 40.0))
		check(t, !pos.IsDocumentStart())
		check(t, !pos.IsDocumentEnd())
		check(t, !pos.IsLineEnd())
		rng := pos.ToDegenerateRange()
		rng.SetEnd(pos.ForwardToCharacterEnd())
		eq(t, "l", rng.Text())
	}

	{
		pos := node.TextPositionAtPoint(a11y.NewPoint(291.0, 40.0))
		check(t, !pos.IsDocumentStart())
		check(t, !pos.IsDocumentEnd())
		check(t, pos.IsLineEnd())
		rng := pos.ToDegenerateRange()
		rng.SetStart(pos.BackwardToWordStart())
		eq(t, "wrap ", rng.Text())
	}

	{
		pos := node.TextPositionAtPoint(a11y.NewPoint(12.0, 50.0))
		check(t, !pos.IsDocumentStart())
		check(t, pos.IsLineStart())
		check(t, !pos.IsParagraphStart())
		rng := pos.ToDegenerateRange()
		rng.SetEnd(pos.ForwardToWordEnd())
		eq(t, "to ", rng.Text())
	}

	{
		pos := node.TextPositionAtPoint(a11y.NewPoint(130.0, 50.0))
		check(t, !pos.IsDocumentStart())
		check(t, !pos.IsDocumentEnd())
		check(t, pos.IsLineEnd())
		rng := pos.ToDegenerateRange()
		rng.SetStart(pos.BackwardToWordStart())
		eq(t, "line.\n", rng.Text())
	}

	{
		pos := node.TextPositionAtPoint(a11y.NewPoint(12.0, 80.0))
		check(t, !pos.IsDocumentStart())
		check(t, !pos.IsDocumentEnd())
		check(t, pos.IsLineEnd())
		rng := pos.ToDegenerateRange()
		rng.SetStart(pos.BackwardToLineStart())
		eq(t, "\n", rng.Text())
	}

	check(t, node.TextPositionAtPoint(a11y.NewPoint(12.0, 120.0)).IsDocumentEnd())
	check(t, node.TextPositionAtPoint(a11y.NewPoint(250.0, 122.0)).IsDocumentEnd())
}

func TestText_ToGlobalUsvIndex(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	node := byID(tree, 1)
	rng := node.DocumentRange()
	eq(t, 0, rng.Start().ToGlobalUsvIndex())
	eq(t, 97, rng.End().ToGlobalUsvIndex())
	eq(t, 38, rng.Start().ForwardToLineEnd().ToGlobalUsvIndex())
	pos := rng.Start().ForwardToLineStart()
	eq(t, 38, pos.ToGlobalUsvIndex())
	pos = pos.ForwardToCharacterStart()
	eq(t, 39, pos.ToGlobalUsvIndex())
	pos = pos.ForwardToLineStart()
	eq(t, 55, pos.ToGlobalUsvIndex())
}

func TestText_ToGlobalUTF16Index(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	node := byID(tree, 1)
	rng := node.DocumentRange()
	eq(t, 0, rng.Start().ToGlobalUTF16Index())
	eq(t, 99, rng.End().ToGlobalUTF16Index())
	eq(t, 38, rng.Start().ForwardToLineEnd().ToGlobalUTF16Index())
	pos := rng.Start().ForwardToLineStart()
	eq(t, 38, pos.ToGlobalUTF16Index())
	pos = pos.ForwardToCharacterStart()
	eq(t, 39, pos.ToGlobalUTF16Index())
	pos = pos.ForwardToLineStart()
	eq(t, 55, pos.ToGlobalUTF16Index())
}

func TestText_ToLineIndex(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	node := byID(tree, 1)
	rng := node.DocumentRange()
	eq(t, 0, rng.Start().ToLineIndex())
	eq(t, 5, rng.End().ToLineIndex())
	eq(t, 0, rng.Start().ForwardToLineEnd().ToLineIndex())
	pos := rng.Start().ForwardToLineStart()
	eq(t, 1, pos.ToLineIndex())
	pos = pos.ForwardToCharacterStart()
	eq(t, 1, pos.ToLineIndex())
	eq(t, 1, pos.ForwardToLineEnd().ToLineIndex())
	pos = pos.ForwardToLineStart()
	eq(t, 2, pos.ToLineIndex())
}

func TestText_LineRangeFromIndex(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	node := byID(tree, 1)
	want := []string{
		"This paragraph is long enough to wrap ",
		"to another line.\n",
		"Another paragraph.\n",
		"\n",
		"Last non-blank line\U0001f44d\U0001f3fb\n",
		"",
	}
	for i, text := range want {
		eq(t, text, must(node.LineRangeFromIndex(i)).Text())
	}
	_, ok := node.LineRangeFromIndex(6)
	check(t, !ok)
}

func charAfter(pos Position) string {
	rng := pos.ToDegenerateRange()
	rng.SetEnd(pos.ForwardToCharacterEnd())
	return rng.Text()
}

func TestText_TextPositionFromGlobalUsvIndex(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	node := byID(tree, 1)
	at := func(i int) Position { return must(node.TextPositionFromGlobalUsvIndex(i)) }

	check(t, at(0).IsDocumentStart())
	eq(t, " ", charAfter(at(17)))
	eq(t, "l", charAfter(at(18)))
	eq(t, " ", charAfter(at(37)))
	{
		pos := at(38)
		check(t, !pos.IsParagraphStart())
		check(t, pos.IsLineStart())
		eq(t, "t", charAfter(pos))
	}
	eq(t, "\n", charAfter(at(54)))
	{
		pos := at(55)
		check(t, pos.IsParagraphStart())
		check(t, pos.IsLineStart())
		eq(t, "A", charAfter(pos))
	}
	for i := 94; i <= 95; i++ {
		eq(t, "\U0001f44d\U0001f3fb", charAfter(at(i)))
	}
	eq(t, "\n", charAfter(at(96)))
	check(t, at(97).IsDocumentEnd())
	_, ok := node.TextPositionFromGlobalUsvIndex(98)
	check(t, !ok)
}

func TestText_TextPositionFromGlobalUTF16Index(t *testing.T) {
	tree := mainMultilineTree(gpuiutil.None[a11y.TextSelection]())
	node := byID(tree, 1)
	at := func(i int) Position { return must(node.TextPositionFromGlobalUTF16Index(i)) }

	check(t, at(0).IsDocumentStart())
	eq(t, " ", charAfter(at(17)))
	eq(t, "l", charAfter(at(18)))
	eq(t, " ", charAfter(at(37)))
	{
		pos := at(38)
		check(t, !pos.IsParagraphStart())
		check(t, pos.IsLineStart())
		eq(t, "t", charAfter(pos))
	}
	eq(t, "\n", charAfter(at(54)))
	{
		pos := at(55)
		check(t, pos.IsParagraphStart())
		check(t, pos.IsLineStart())
		eq(t, "A", charAfter(pos))
	}
	for i := 94; i <= 97; i++ {
		eq(t, "\U0001f44d\U0001f3fb", charAfter(at(i)))
	}
	eq(t, "\n", charAfter(at(98)))
	check(t, at(99).IsDocumentEnd())
	_, ok := node.TextPositionFromGlobalUTF16Index(100)
	check(t, !ok)
}

func TestText_MultilineSelectionClamping(t *testing.T) {
	tree := mainMultilineTree(multilinePastEndSelection())
	_ = must(byID(tree, 1).TextSelection())
}

func TestText_RangePropertyValueMap(t *testing.T) {
	inc := func(x int) int { return x + 1 }
	eq(t, RangePropertyValueSingle(gpuiutil.Some(1)), RangePropertyValueMap(RangePropertyValueSingle(gpuiutil.Some(0)), inc))
	eq(t, RangePropertyValueSingle(gpuiutil.None[int]()), RangePropertyValueMap(RangePropertyValueSingle(gpuiutil.None[int]()), inc))
	eq(t, RangePropertyValueMixed[gpuiutil.Option[int]](), RangePropertyValueMap(RangePropertyValueMixed[gpuiutil.Option[int]](), inc))
}

type inheritedPropertyCase[T comparable] struct {
	set      func(*a11y.Node, T)
	node     func(Node) (T, bool)
	position func(Position) (T, bool)
	rng      func(Range) RangePropertyValue[gpuiutil.Option[T]]
	value1   T
	value2   T
}

func textRun(value string, lengths int) a11y.Node {
	n := node(a11y.RoleTextRun)
	n.SetValue(value)
	n.SetCharacterLengths(ones(lengths))
	return n
}

func inheritedTree(parent a11y.Node, runs ...a11y.Node) *Tree {
	entries := []a11y.NodeEntry{ent(0, parent)}
	for i, run := range runs {
		entries = append(entries, ent(a11y.NodeID(i+1), run))
	}
	return NewTree(rootUpdate(0, entries...), false)
}

func someOf[T any](v T) gpuiutil.Option[T] { return gpuiutil.Some(v) }

func (c inheritedPropertyCase[T]) directlySet(t *testing.T) {
	run := textRun("text", 4)
	c.set(&run, c.value1)
	n := byID(inheritedTree(node(a11y.RoleTextInput, 1), run), 0)
	eq(t, someOf(c.value1), gpuiutil.OptionFrom(c.position(n.DocumentStart())))
	eq(t, RangePropertyValueSingle(someOf(c.value1)), c.rng(n.DocumentRange()))
}

func (c inheritedPropertyCase[T]) setOnParent(t *testing.T) {
	parent := node(a11y.RoleTextInput, 1)
	c.set(&parent, c.value1)
	n := byID(inheritedTree(parent, textRun("text", 4)), 0)
	eq(t, someOf(c.value1), gpuiutil.OptionFrom(c.position(n.DocumentStart())))
	eq(t, RangePropertyValueSingle(someOf(c.value1)), c.rng(n.DocumentRange()))
}

func (c inheritedPropertyCase[T]) onlyChildOverridesParent(t *testing.T) {
	parent := node(a11y.RoleTextInput, 1)
	c.set(&parent, c.value1)
	run := textRun("text", 4)
	c.set(&run, c.value2)
	n := byID(inheritedTree(parent, run), 0)
	eq(t, someOf(c.value1), gpuiutil.OptionFrom(c.node(n)))
	eq(t, someOf(c.value2), gpuiutil.OptionFrom(c.position(n.DocumentStart())))
	eq(t, RangePropertyValueSingle(someOf(c.value2)), c.rng(n.DocumentRange()))
}

func (c inheritedPropertyCase[T]) unset(t *testing.T) {
	n := byID(inheritedTree(node(a11y.RoleTextInput, 1), textRun("text", 4)), 0)
	eq(t, gpuiutil.None[T](), gpuiutil.OptionFrom(c.position(n.DocumentStart())))
	eq(t, RangePropertyValueSingle(gpuiutil.None[T]()), c.rng(n.DocumentRange()))
}

func (c inheritedPropertyCase[T]) mixedSomeAndNone(t *testing.T) {
	run1 := textRun("text 1\n", 7)
	c.set(&run1, c.value1)
	n := byID(inheritedTree(node(a11y.RoleTextInput, 1, 2), run1, textRun("text 2", 6)), 0)
	eq(t, RangePropertyValueMixed[gpuiutil.Option[T]](), c.rng(n.DocumentRange()))
}

func (c inheritedPropertyCase[T]) mixedOneChildOverridesParent(t *testing.T) {
	parent := node(a11y.RoleTextInput, 1, 2)
	c.set(&parent, c.value1)
	run1 := textRun("text 1\n", 7)
	c.set(&run1, c.value2)
	n := byID(inheritedTree(parent, run1, textRun("text 2", 6)), 0)
	eq(t, someOf(c.value1), gpuiutil.OptionFrom(c.node(n)))
	start := n.DocumentStart()
	eq(t, someOf(c.value2), gpuiutil.OptionFrom(c.position(start)))
	eq(t, RangePropertyValueSingle(someOf(c.value2)), c.rng(start.ToDegenerateRange()))
	end := n.DocumentEnd()
	eq(t, someOf(c.value1), gpuiutil.OptionFrom(c.position(end)))
	eq(t, RangePropertyValueSingle(someOf(c.value1)), c.rng(end.ToDegenerateRange()))
	eq(t, RangePropertyValueMixed[gpuiutil.Option[T]](), c.rng(n.DocumentRange()))
}

var (
	textDirectionCase   = inheritedPropertyCase[a11y.TextDirection]{(*a11y.Node).SetTextDirection, Node.TextDirection, Position.TextDirection, Range.TextDirection, a11y.TextDirectionLeftToRight, a11y.TextDirectionRightToLeft}
	fontFamilyCase      = inheritedPropertyCase[string]{(*a11y.Node).SetFontFamily, Node.FontFamily, Position.FontFamily, Range.FontFamily, "Noto", "Inconsolata"}
	languageCase        = inheritedPropertyCase[string]{(*a11y.Node).SetLanguage, Node.Language, Position.Language, Range.Language, "en", "fr"}
	fontSizeCase        = inheritedPropertyCase[float32]{(*a11y.Node).SetFontSize, Node.FontSize, Position.FontSize, Range.FontSize, 12.0, 24.0}
	fontWeightCase      = inheritedPropertyCase[float32]{(*a11y.Node).SetFontWeight, Node.FontWeight, Position.FontWeight, Range.FontWeight, 400.0, 700.0}
	backgroundColorCase = inheritedPropertyCase[a11y.Color]{(*a11y.Node).SetBackgroundColor, Node.BackgroundColor, Position.BackgroundColor, Range.BackgroundColor, a11y.Color{Red: 255, Green: 255, Blue: 255, Alpha: 255}, a11y.Color{Red: 255, Green: 0, Blue: 0, Alpha: 255}}
	foregroundColorCase = inheritedPropertyCase[a11y.Color]{(*a11y.Node).SetForegroundColor, Node.ForegroundColor, Position.ForegroundColor, Range.ForegroundColor, a11y.Color{Red: 0, Green: 0, Blue: 0, Alpha: 255}, a11y.Color{Red: 0, Green: 0, Blue: 255, Alpha: 255}}
	overlineCase        = inheritedPropertyCase[a11y.TextDecoration]{(*a11y.Node).SetOverline, Node.Overline, Position.Overline, Range.Overline, testTextDecoration1, testTextDecoration2}
	strikethroughCase   = inheritedPropertyCase[a11y.TextDecoration]{(*a11y.Node).SetStrikethrough, Node.Strikethrough, Position.Strikethrough, Range.Strikethrough, testTextDecoration2, testTextDecoration3}
	underlineCase       = inheritedPropertyCase[a11y.TextDecoration]{(*a11y.Node).SetUnderline, Node.Underline, Position.Underline, Range.Underline, testTextDecoration3, testTextDecoration4}
	textAlignCase       = inheritedPropertyCase[a11y.TextAlign]{(*a11y.Node).SetTextAlign, Node.TextAlign, Position.TextAlign, Range.TextAlign, a11y.TextAlignLeft, a11y.TextAlignJustify}
	verticalOffsetCase  = inheritedPropertyCase[a11y.VerticalOffset]{(*a11y.Node).SetVerticalOffset, Node.VerticalOffset, Position.VerticalOffset, Range.VerticalOffset, a11y.VerticalOffsetSubscript, a11y.VerticalOffsetSuperscript}
)

func TestText_TextDirection_DirectlySet(t *testing.T) { textDirectionCase.directlySet(t) }

func TestText_TextDirection_SetOnParent(t *testing.T) { textDirectionCase.setOnParent(t) }

func TestText_TextDirection_OnlyChildOverridesParent(t *testing.T) {
	textDirectionCase.onlyChildOverridesParent(t)
}

func TestText_TextDirection_Unset(t *testing.T) { textDirectionCase.unset(t) }

func TestText_TextDirection_MixedSomeAndNone(t *testing.T) { textDirectionCase.mixedSomeAndNone(t) }

func TestText_TextDirection_MixedOneChildOverridesParent(t *testing.T) {
	textDirectionCase.mixedOneChildOverridesParent(t)
}

func TestText_FontFamily_DirectlySet(t *testing.T) { fontFamilyCase.directlySet(t) }

func TestText_FontFamily_SetOnParent(t *testing.T) { fontFamilyCase.setOnParent(t) }

func TestText_FontFamily_OnlyChildOverridesParent(t *testing.T) {
	fontFamilyCase.onlyChildOverridesParent(t)
}

func TestText_FontFamily_Unset(t *testing.T) { fontFamilyCase.unset(t) }

func TestText_FontFamily_MixedSomeAndNone(t *testing.T) { fontFamilyCase.mixedSomeAndNone(t) }

func TestText_FontFamily_MixedOneChildOverridesParent(t *testing.T) {
	fontFamilyCase.mixedOneChildOverridesParent(t)
}

func TestText_Language_DirectlySet(t *testing.T) { languageCase.directlySet(t) }

func TestText_Language_SetOnParent(t *testing.T) { languageCase.setOnParent(t) }

func TestText_Language_OnlyChildOverridesParent(t *testing.T) {
	languageCase.onlyChildOverridesParent(t)
}

func TestText_Language_Unset(t *testing.T) { languageCase.unset(t) }

func TestText_Language_MixedSomeAndNone(t *testing.T) { languageCase.mixedSomeAndNone(t) }

func TestText_Language_MixedOneChildOverridesParent(t *testing.T) {
	languageCase.mixedOneChildOverridesParent(t)
}

func TestText_FontSize_DirectlySet(t *testing.T) { fontSizeCase.directlySet(t) }

func TestText_FontSize_SetOnParent(t *testing.T) { fontSizeCase.setOnParent(t) }

func TestText_FontSize_OnlyChildOverridesParent(t *testing.T) {
	fontSizeCase.onlyChildOverridesParent(t)
}

func TestText_FontSize_Unset(t *testing.T) { fontSizeCase.unset(t) }

func TestText_FontSize_MixedSomeAndNone(t *testing.T) { fontSizeCase.mixedSomeAndNone(t) }

func TestText_FontSize_MixedOneChildOverridesParent(t *testing.T) {
	fontSizeCase.mixedOneChildOverridesParent(t)
}

func TestText_FontWeight_DirectlySet(t *testing.T) { fontWeightCase.directlySet(t) }

func TestText_FontWeight_SetOnParent(t *testing.T) { fontWeightCase.setOnParent(t) }

func TestText_FontWeight_OnlyChildOverridesParent(t *testing.T) {
	fontWeightCase.onlyChildOverridesParent(t)
}

func TestText_FontWeight_Unset(t *testing.T) { fontWeightCase.unset(t) }

func TestText_FontWeight_MixedSomeAndNone(t *testing.T) { fontWeightCase.mixedSomeAndNone(t) }

func TestText_FontWeight_MixedOneChildOverridesParent(t *testing.T) {
	fontWeightCase.mixedOneChildOverridesParent(t)
}

func TestText_BackgroundColor_DirectlySet(t *testing.T) { backgroundColorCase.directlySet(t) }

func TestText_BackgroundColor_SetOnParent(t *testing.T) { backgroundColorCase.setOnParent(t) }

func TestText_BackgroundColor_OnlyChildOverridesParent(t *testing.T) {
	backgroundColorCase.onlyChildOverridesParent(t)
}

func TestText_BackgroundColor_Unset(t *testing.T) { backgroundColorCase.unset(t) }

func TestText_BackgroundColor_MixedSomeAndNone(t *testing.T) { backgroundColorCase.mixedSomeAndNone(t) }

func TestText_BackgroundColor_MixedOneChildOverridesParent(t *testing.T) {
	backgroundColorCase.mixedOneChildOverridesParent(t)
}

func TestText_ForegroundColor_DirectlySet(t *testing.T) { foregroundColorCase.directlySet(t) }

func TestText_ForegroundColor_SetOnParent(t *testing.T) { foregroundColorCase.setOnParent(t) }

func TestText_ForegroundColor_OnlyChildOverridesParent(t *testing.T) {
	foregroundColorCase.onlyChildOverridesParent(t)
}

func TestText_ForegroundColor_Unset(t *testing.T) { foregroundColorCase.unset(t) }

func TestText_ForegroundColor_MixedSomeAndNone(t *testing.T) { foregroundColorCase.mixedSomeAndNone(t) }

func TestText_ForegroundColor_MixedOneChildOverridesParent(t *testing.T) {
	foregroundColorCase.mixedOneChildOverridesParent(t)
}

func TestText_Overline_DirectlySet(t *testing.T) { overlineCase.directlySet(t) }

func TestText_Overline_SetOnParent(t *testing.T) { overlineCase.setOnParent(t) }

func TestText_Overline_OnlyChildOverridesParent(t *testing.T) {
	overlineCase.onlyChildOverridesParent(t)
}

func TestText_Overline_Unset(t *testing.T) { overlineCase.unset(t) }

func TestText_Overline_MixedSomeAndNone(t *testing.T) { overlineCase.mixedSomeAndNone(t) }

func TestText_Overline_MixedOneChildOverridesParent(t *testing.T) {
	overlineCase.mixedOneChildOverridesParent(t)
}

func TestText_Strikethrough_DirectlySet(t *testing.T) { strikethroughCase.directlySet(t) }

func TestText_Strikethrough_SetOnParent(t *testing.T) { strikethroughCase.setOnParent(t) }

func TestText_Strikethrough_OnlyChildOverridesParent(t *testing.T) {
	strikethroughCase.onlyChildOverridesParent(t)
}

func TestText_Strikethrough_Unset(t *testing.T) { strikethroughCase.unset(t) }

func TestText_Strikethrough_MixedSomeAndNone(t *testing.T) { strikethroughCase.mixedSomeAndNone(t) }

func TestText_Strikethrough_MixedOneChildOverridesParent(t *testing.T) {
	strikethroughCase.mixedOneChildOverridesParent(t)
}

func TestText_Underline_DirectlySet(t *testing.T) { underlineCase.directlySet(t) }

func TestText_Underline_SetOnParent(t *testing.T) { underlineCase.setOnParent(t) }

func TestText_Underline_OnlyChildOverridesParent(t *testing.T) {
	underlineCase.onlyChildOverridesParent(t)
}

func TestText_Underline_Unset(t *testing.T) { underlineCase.unset(t) }

func TestText_Underline_MixedSomeAndNone(t *testing.T) { underlineCase.mixedSomeAndNone(t) }

func TestText_Underline_MixedOneChildOverridesParent(t *testing.T) {
	underlineCase.mixedOneChildOverridesParent(t)
}

func TestText_TextAlign_DirectlySet(t *testing.T) { textAlignCase.directlySet(t) }

func TestText_TextAlign_SetOnParent(t *testing.T) { textAlignCase.setOnParent(t) }

func TestText_TextAlign_OnlyChildOverridesParent(t *testing.T) {
	textAlignCase.onlyChildOverridesParent(t)
}

func TestText_TextAlign_Unset(t *testing.T) { textAlignCase.unset(t) }

func TestText_TextAlign_MixedSomeAndNone(t *testing.T) { textAlignCase.mixedSomeAndNone(t) }

func TestText_TextAlign_MixedOneChildOverridesParent(t *testing.T) {
	textAlignCase.mixedOneChildOverridesParent(t)
}

func TestText_VerticalOffset_DirectlySet(t *testing.T) { verticalOffsetCase.directlySet(t) }

func TestText_VerticalOffset_SetOnParent(t *testing.T) { verticalOffsetCase.setOnParent(t) }

func TestText_VerticalOffset_OnlyChildOverridesParent(t *testing.T) {
	verticalOffsetCase.onlyChildOverridesParent(t)
}

func TestText_VerticalOffset_Unset(t *testing.T) { verticalOffsetCase.unset(t) }

func TestText_VerticalOffset_MixedSomeAndNone(t *testing.T) { verticalOffsetCase.mixedSomeAndNone(t) }

func TestText_VerticalOffset_MixedOneChildOverridesParent(t *testing.T) {
	verticalOffsetCase.mixedOneChildOverridesParent(t)
}

func TestText_IsItalic_DirectlySet(t *testing.T) {
	run := textRun("text", 4)
	run.SetItalic()
	n := byID(inheritedTree(node(a11y.RoleTextInput, 1), run), 0)
	check(t, n.DocumentStart().IsItalic())
	eq(t, RangePropertyValueSingle(true), n.DocumentRange().IsItalic())
}

func TestText_IsItalic_SetOnParent(t *testing.T) {
	parent := node(a11y.RoleTextInput, 1)
	parent.SetItalic()
	n := byID(inheritedTree(parent, textRun("text", 4)), 0)
	check(t, n.DocumentStart().IsItalic())
	eq(t, RangePropertyValueSingle(true), n.DocumentRange().IsItalic())
}

func TestText_IsItalic_Unset(t *testing.T) {
	n := byID(inheritedTree(node(a11y.RoleTextInput, 1), textRun("text", 4)), 0)
	check(t, !n.DocumentStart().IsItalic())
	eq(t, RangePropertyValueSingle(false), n.DocumentRange().IsItalic())
}

func TestText_IsItalic_Mixed(t *testing.T) {
	run1 := textRun("text 1\n", 7)
	run1.SetItalic()
	n := byID(inheritedTree(node(a11y.RoleTextInput, 1, 2), run1, textRun("text 2", 6)), 0)
	eq(t, RangePropertyValueMixed[bool](), n.DocumentRange().IsItalic())
}
