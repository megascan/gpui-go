package gpui

import (
	"fmt"
	"iter"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-text/typesetting/segmenter"
)

// TruncateFrom determines whether to truncate text from the start or end.
type TruncateFrom uint8

const (
	// TruncateFromStart truncates text from the start.
	TruncateFromStart TruncateFrom = iota
	// TruncateFromEnd truncates text from the end.
	TruncateFromEnd
	// TruncateFromMiddle truncates text from the middle, preserving the start and end.
	TruncateFromMiddle
)

// IndentAdjustmentKind is the variant of an IndentAdjustment.
type IndentAdjustmentKind uint8

const (
	// IndentAdjustmentKindNoIndent starts continuation lines at column 0.
	IndentAdjustmentKindNoIndent IndentAdjustmentKind = iota
	// IndentAdjustmentKindSameIndent matches the original line's leading whitespace.
	IndentAdjustmentKindSameIndent
	// IndentAdjustmentKindExtraColumns adds extra columns of indent.
	IndentAdjustmentKindExtraColumns
)

// IndentAdjustment controls how soft-wrapped continuation lines are indented.
type IndentAdjustment struct {
	// Kind is the variant.
	Kind IndentAdjustmentKind
	// ExtraColumns is the number of extra columns (in space-character widths) for
	// IndentAdjustmentKindExtraColumns.
	ExtraColumns uint32
}

var (
	// IndentAdjustmentNoIndent starts continuation lines at column 0.
	IndentAdjustmentNoIndent = IndentAdjustment{Kind: IndentAdjustmentKindNoIndent}
	// IndentAdjustmentSameIndent matches the original line's leading whitespace.
	IndentAdjustmentSameIndent = IndentAdjustment{Kind: IndentAdjustmentKindSameIndent}
)

// IndentAdjustmentExtraColumns adds n extra columns of indent (in space-character widths).
func IndentAdjustmentExtraColumns(n uint32) IndentAdjustment {
	return IndentAdjustment{Kind: IndentAdjustmentKindExtraColumns, ExtraColumns: n}
}

// DefaultIndentAdjustment returns IndentAdjustmentSameIndent.
func DefaultIndentAdjustment() IndentAdjustment {
	return IndentAdjustmentSameIndent
}

// String formats the adjustment as upstream's Debug does.
func (a IndentAdjustment) String() string {
	switch a.Kind {
	case IndentAdjustmentKindNoIndent:
		return "NoIndent"
	case IndentAdjustmentKindSameIndent:
		return "SameIndent"
	default:
		return fmt.Sprintf("ExtraColumns(%d)", a.ExtraColumns)
	}
}

// LineWrapper is the GPUI line wrapper, used to wrap lines of text to a given width.
type LineWrapper struct {
	textSystem *TextSystem
	fontID     FontID
	fontSize   Pixels
	charWidths charWidthCache
	textBuffer []byte
}

type charWidthCache struct {
	ascii [128]Option[Pixels]
	other map[rune]Pixels
}

func defaultCharWidthCache() charWidthCache {
	return charWidthCache{other: map[rune]Pixels{}}
}

// LineWrapperMaxIndent is the maximum indent that can be applied to a line.
const LineWrapperMaxIndent uint32 = 256

func newLineWrapper(fontID FontID, fontSize Pixels, textSystem *TextSystem, charWidths charWidthCache) *LineWrapper {
	return &LineWrapper{textSystem: textSystem, fontID: fontID, fontSize: fontSize, charWidths: charWidths}
}

func (w *LineWrapper) intoCharWidths() charWidthCache {
	return w.charWidths
}

type peekableInts struct {
	items []int
	pos   int
}

func (p *peekableInts) nextIfEq(v int) bool {
	if p.pos < len(p.items) && p.items[p.pos] == v {
		p.pos++
		return true
	}
	return false
}

func (p *peekableInts) nextIfLess(v int) bool {
	if p.pos < len(p.items) && p.items[p.pos] < v {
		p.pos++
		return true
	}
	return false
}

func graphemeIndices(text string) []int {
	var seg segmenter.Segmenter
	seg.Init([]rune(text))
	graphemes := seg.GraphemeIterator()
	var out []int
	for graphemes.Next() {
		out = append(out, graphemes.Grapheme().OffsetInBytes)
	}
	return out
}

func minU32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func saturatingAddU32(a, b uint32) uint32 {
	if a > math.MaxUint32-b {
		return math.MaxUint32
	}
	return a + b
}

func maxPixels(a, b Pixels) Pixels {
	if a > b {
		return a
	}
	return b
}

// WrapLine wraps a line of text to the given width with this wrapper's font and font size.
// The returned sequence is single-use, like upstream's iterator.
func (w *LineWrapper) WrapLine(fragments []LineFragment, wrapWidth Pixels, indentAdjustment IndentAdjustment) iter.Seq[Boundary] {
	width := Px(0.)
	firstNonWhitespaceIx := None[int]()
	baseIndent := None[uint32]()
	lastCandidateIx := 0
	lastCandidateWidth := Px(0.)
	lastWrapIx := 0
	index := 0
	text := w.textBuffer[:0]
	for _, fragment := range fragments {
		if fragment.isElement {
			for range fragment.lenUTF8 {
				text = append(text, 'a')
			}
		} else {
			text = append(text, fragment.text...)
		}
	}
	w.textBuffer = text
	textStr := string(text)
	breakIndices := &peekableInts{}
	for ix := range lineWrapperBreakIndices(textStr) {
		breakIndices.items = append(breakIndices.items, ix)
	}
	graphemes := &peekableInts{items: graphemeIndices(textStr)}
	widthForChar := func(c rune) Pixels {
		return widthForChar(&w.charWidths, c, w.textSystem, w.fontID, w.fontSize)
	}
	graphemeIndex := 0
	widthBeforeGrapheme := Px(0.)
	var candidates []wrapBoundaryCandidate
	for _, fragment := range fragments {
		candidates = fragment.appendWrapBoundaryCandidates(candidates)
	}
	candidatePos := 0
	next := func() (Boundary, bool) {
		for candidatePos < len(candidates) {
			candidate := candidates[candidatePos]
			candidatePos++
			ix := index
			index += candidate.lenUTF8()
			for graphemes.nextIfLess(ix) {
			}
			isGraphemeBoundary := graphemes.nextIfEq(ix)
			if isGraphemeBoundary {
				graphemeIndex = ix
				widthBeforeGrapheme = width
			}
			if breakIndices.nextIfEq(ix) && isGraphemeBoundary && firstNonWhitespaceIx.IsSome() {
				lastCandidateIx = ix
				lastCandidateWidth = width
			}
			var itemWidth Pixels
			if !candidate.isElement {
				c := candidate.character
				if c == '\n' {
					continue
				}
				if c != ' ' && firstNonWhitespaceIx.IsNone() {
					firstNonWhitespaceIx = Some(ix)
				}
				itemWidth = widthForChar(c)
			} else {
				if firstNonWhitespaceIx.IsNone() {
					firstNonWhitespaceIx = Some(ix)
				}
				itemWidth = candidate.width
			}

			width += itemWidth
			if width > wrapWidth && graphemeIndex > lastWrapIx {
				wrapAtCandidate := lastCandidateIx > lastWrapIx && width-lastCandidateWidth <= wrapWidth

				var carriedWidth Pixels
				if wrapAtCandidate {
					carriedWidth = width - lastCandidateWidth
				} else {
					carriedWidth = width - widthBeforeGrapheme
				}

				if baseIndent.IsNone() {
					if first, ok := firstNonWhitespaceIx.Get(); ok {
						baseIndent = Some(minU32(LineWrapperMaxIndent, uint32(saturatingSub(first, lastWrapIx))))
					}
				}

				var nextIndent uint32
				switch indentAdjustment.Kind {
				case IndentAdjustmentKindNoIndent:
					nextIndent = 0
				case IndentAdjustmentKindSameIndent:
					nextIndent = baseIndent.UnwrapOr(0)
				case IndentAdjustmentKindExtraColumns:
					if base, ok := baseIndent.Get(); ok {
						candidate := saturatingAddU32(base, indentAdjustment.ExtraColumns)
						candidateIndentWidth := widthForChar(' ') * Pixels(float32(candidate))
						minHeadroom := maxPixels(carriedWidth, widthForChar(' ')*2.0)
						if candidateIndentWidth+minHeadroom > wrapWidth {
							nextIndent = 0
						} else {
							nextIndent = minU32(LineWrapperMaxIndent, candidate)
						}
					}
				}

				if wrapAtCandidate {
					lastWrapIx = lastCandidateIx
					width -= lastCandidateWidth
					widthBeforeGrapheme -= lastCandidateWidth
				} else {
					lastWrapIx = graphemeIndex
					width -= widthBeforeGrapheme
					widthBeforeGrapheme = Px(0.)
				}
				lastCandidateIx = 0

				indentWidth := widthForChar(' ') * Pixels(float32(nextIndent))
				width += indentWidth
				widthBeforeGrapheme += indentWidth

				return newBoundary(lastWrapIx, nextIndent), true
			}
		}
		return Boundary{}, false
	}
	return func(yield func(Boundary) bool) {
		for {
			boundary, ok := next()
			if !ok || !yield(boundary) {
				return
			}
		}
	}
}

func (w *LineWrapper) charsWidth(s string) Pixels {
	total := Px(0.0)
	for _, c := range s {
		total += w.widthForChar(c)
	}
	return total
}

// ShouldTruncateLine determines if a line should be truncated based on its width and
// returns the truncation index in line.
func (w *LineWrapper) ShouldTruncateLine(line string, truncateWidth Pixels, truncationAffix string, truncateFrom TruncateFrom) (int, bool) {
	width := Px(0.)
	suffixWidth := w.charsWidth(truncationAffix)
	truncateIx := 0

	step := func(ix int, c rune) bool {
		if width+suffixWidth < truncateWidth {
			truncateIx = ix
		}
		width += w.widthForChar(c)
		return width.Floor() > truncateWidth
	}

	switch truncateFrom {
	case TruncateFromStart:
		for ix := len(line); ix > 0; {
			c, size := utf8.DecodeLastRuneInString(line[:ix])
			ix -= size
			if step(ix, c) {
				return truncateIx, true
			}
		}
	case TruncateFromEnd:
		for ix, c := range line {
			if step(ix, c) {
				return truncateIx, true
			}
		}
	case TruncateFromMiddle:
	}

	return 0, false
}

func (w *LineWrapper) shouldTruncateLineMiddle(line string, truncateWidth Pixels, truncationAffix string) (int, int, bool) {
	suffixWidth := w.charsWidth(truncationAffix)
	totalWidth := w.charsWidth(line)

	if totalWidth <= truncateWidth {
		return 0, 0, false
	}

	contentBudget := truncateWidth - suffixWidth
	if contentBudget <= Px(0.) {
		return 0, len(line), true
	}

	frontBudget := contentBudget * (2.0 / 3.0)
	backBudget := contentBudget - frontBudget

	frontWidth := Px(0.)
	frontEndIx := 0
	for ix, c := range line {
		charWidth := w.widthForChar(c)
		if frontWidth+charWidth > frontBudget {
			break
		}
		frontWidth += charWidth
		frontEndIx = ix + utf8.RuneLen(c)
	}

	backWidth := Px(0.)
	backStartIx := len(line)
	for ix := len(line); ix > 0; {
		c, size := utf8.DecodeLastRuneInString(line[:ix])
		ix -= size
		charWidth := w.widthForChar(c)
		if backWidth+charWidth > backBudget {
			break
		}
		backWidth += charWidth
		backStartIx = ix
	}

	if frontEndIx >= backStartIx {
		return 0, len(line), true
	}

	return frontEndIx, backStartIx, true
}

func ceilCharBoundary(s string, index int) int {
	if index > len(s) {
		return len(s)
	}
	for index < len(s) && !utf8.RuneStart(s[index]) {
		index++
	}
	return index
}

func isASCIIPunctuation(c rune) bool {
	return c < utf8.RuneSelf && (unicode.IsPunct(c) || unicode.IsSymbol(c))
}

func trimEndWhitespaceAndPunctuation(s string) string {
	return strings.TrimRightFunc(s, func(c rune) bool {
		return unicode.IsSpace(c) || isASCIIPunctuation(c)
	})
}

// TruncateLine truncates a line of text to the given width with this wrapper's font and font
// size. The returned runs are the input slice when nothing was truncated, a new slice otherwise.
func (w *LineWrapper) TruncateLine(line SharedString, truncateWidth Pixels, truncationAffix string, runs []TextRun, truncateFrom TruncateFrom) (SharedString, []TextRun) {
	if truncateFrom == TruncateFromMiddle {
		if frontEndIx, backStartIx, ok := w.shouldTruncateLineMiddle(string(line), truncateWidth, truncationAffix); ok {
			result := SharedString(string(line[:frontEndIx]) + truncationAffix + string(line[backStartIx:]))
			newRuns := append([]TextRun(nil), runs...)
			updateRunsAfterMiddleTruncation(truncationAffix, &newRuns, frontEndIx, backStartIx)
			return result, newRuns
		}
		return line, runs
	}

	if truncateIx, ok := w.ShouldTruncateLine(string(line), truncateWidth, truncationAffix, truncateFrom); ok {
		var result SharedString
		switch truncateFrom {
		case TruncateFromStart:
			result = SharedString(truncationAffix + string(line[ceilCharBoundary(string(line), truncateIx+1):]))
		case TruncateFromEnd:
			result = SharedString(trimEndWhitespaceAndPunctuation(string(line[:truncateIx])) + truncationAffix)
		default:
			panic("internal error: entered unreachable code: Middle truncation is handled above")
		}
		newRuns := append([]TextRun(nil), runs...)
		updateRunsAfterTruncation(string(result), truncationAffix, &newRuns, truncateFrom)
		return result, newRuns
	}
	return line, runs
}

// TruncateWrappedLine truncates text to fit within a given number of wrapped lines.
//
// Unlike TruncateLine, which treats the text as a flat width budget (width * maxLines), it
// accounts for word-boundary wrapping. For maxLines == 1 it delegates to TruncateLine.
func (w *LineWrapper) TruncateWrappedLine(text SharedString, wrapWidth Pixels, maxLines int, truncationAffix string, runs []TextRun, truncateFrom TruncateFrom) (SharedString, []TextRun) {
	if maxLines <= 1 || truncateFrom == TruncateFromStart {
		return w.TruncateLine(text, wrapWidth*Pixels(float32(maxLines)), truncationAffix, runs, truncateFrom)
	}
	if truncateFrom == TruncateFromMiddle {
		return w.TruncateLine(text, wrapWidth, truncationAffix, runs, truncateFrom)
	}

	affixWidth := w.charsWidth(truncationAffix)
	str := string(text)

	width := Px(0.)
	line := 0
	firstNonWhitespaceIx := None[int]()
	lastCandidateIx := 0
	lastCandidateWidth := Px(0.)
	lastWrapIx := 0
	indent := None[uint32]()
	truncateIx := 0

	truncateAt := func() (SharedString, []TextRun) {
		truncated := trimEndWhitespaceAndPunctuation(str[:truncateIx])
		result := SharedString(truncated + truncationAffix)
		newRuns := append([]TextRun(nil), runs...)
		updateRunsAfterTruncation(string(result), truncationAffix, &newRuns, TruncateFromEnd)
		return result, newRuns
	}

	breakIndices := &peekableInts{}
	for ix := range lineWrapperBreakIndices(str) {
		breakIndices.items = append(breakIndices.items, ix)
	}
	for ix, c := range str {
		isBreak := breakIndices.nextIfEq(ix)
		if c == '\n' {
			if line >= maxLines-1 && strings.TrimSpace(str[ix+1:]) != "" {
				return truncateAt()
			}

			line++
			width = Px(0.)
			firstNonWhitespaceIx = None[int]()
			lastCandidateIx = 0
			lastCandidateWidth = Px(0.)
			lastWrapIx = ix + 1
			indent = None[uint32]()
			truncateIx = ix + 1
			continue
		}

		charWidth := w.widthForChar(c)

		if isBreak && firstNonWhitespaceIx.IsSome() {
			lastCandidateIx = ix
			lastCandidateWidth = width
		}

		if c != ' ' && firstNonWhitespaceIx.IsNone() {
			firstNonWhitespaceIx = Some(ix)
		}

		width += charWidth

		if line < maxLines-1 {
			if width > wrapWidth && ix > lastWrapIx {
				if firstNW, ok := firstNonWhitespaceIx.Get(); indent.IsNone() && ok {
					if firstNW < lastWrapIx {
						panic("attempt to subtract with overflow")
					}
					indent = Some(minU32(LineWrapperMaxIndent, uint32(firstNW-lastWrapIx)))
				}

				if lastCandidateIx > lastWrapIx {
					lastWrapIx = lastCandidateIx
					width -= lastCandidateWidth
					lastCandidateIx = 0
				} else {
					lastWrapIx = ix
					width = charWidth
				}

				if ind, ok := indent.Get(); ok {
					width += w.widthForChar(' ') * Pixels(float32(ind))
				}

				line++
				truncateIx = lastWrapIx
			}
		} else {
			if width+affixWidth <= wrapWidth {
				truncateIx = ix + utf8.RuneLen(c)
			}

			if width > wrapWidth {
				return truncateAt()
			}
		}
	}

	return text, runs
}

func lineWrapperBreakIndices(text string) iter.Seq[int] {
	return func(yield func(int) bool) {
		if !yield(0) {
			return
		}
		var seg segmenter.Segmenter
		seg.Init([]rune(text))
		lines := seg.LineIterator()
		for lines.Next() {
			line := lines.Line()
			offset := line.OffsetInBytes + line.LengthInBytes
			if offset == 0 {
				continue
			}
			if !yield(offset) {
				return
			}
		}
	}
}

func (w *LineWrapper) widthForChar(c rune) Pixels {
	return widthForChar(&w.charWidths, c, w.textSystem, w.fontID, w.fontSize)
}

func widthForChar(charWidths *charWidthCache, c rune, textSystem *TextSystem, fontID FontID, fontSize Pixels) Pixels {
	if uint32(c) < 128 {
		if cachedWidth, ok := charWidths.ascii[c].Get(); ok {
			return cachedWidth
		}
		width := textSystem.LayoutWidth(fontID, fontSize, c)
		charWidths.ascii[c] = Some(width)
		return width
	}
	if cachedWidth, ok := charWidths.other[c]; ok {
		return cachedWidth
	}
	width := textSystem.LayoutWidth(fontID, fontSize, c)
	charWidths.other[c] = width
	return width
}

func updateRunsAfterTruncation(result, ellipsis string, runs *[]TextRun, truncateFrom TruncateFrom) {
	truncateAt := len(result) - len(ellipsis)
	rs := *runs
	switch truncateFrom {
	case TruncateFromStart:
		for runIndex := len(rs) - 1; runIndex >= 0; runIndex-- {
			run := &rs[runIndex]
			if run.Len <= truncateAt {
				truncateAt -= run.Len
			} else {
				run.Len = truncateAt + len(ellipsis)
				*runs = rs[runIndex:]
				break
			}
		}
	case TruncateFromEnd:
		for runIndex := range rs {
			run := &rs[runIndex]
			if run.Len <= truncateAt {
				truncateAt -= run.Len
			} else {
				run.Len = truncateAt + len(ellipsis)
				*runs = rs[:runIndex+1]
				break
			}
		}
	case TruncateFromMiddle:
		panic("internal error: entered unreachable code: Middle truncation calls this function with TruncateFrom::End directly")
	}
}

func updateRunsAfterMiddleTruncation(ellipsis string, runs *[]TextRun, frontEndIx, backStartIx int) {
	originalRuns := *runs
	*runs = nil
	resultRuns := make([]TextRun, 0, len(originalRuns))

	frontRemaining := frontEndIx
	frontDone := false
	for _, run := range originalRuns {
		if frontDone {
			break
		}
		if run.Len <= frontRemaining {
			resultRuns = append(resultRuns, run)
			frontRemaining -= run.Len
		} else {
			partial := run
			partial.Len = frontRemaining + len(ellipsis)
			resultRuns = append(resultRuns, partial)
			frontDone = true
		}
	}
	if !frontDone {
		if len(resultRuns) > 0 {
			resultRuns[len(resultRuns)-1].Len += len(ellipsis)
		} else if len(originalRuns) > 0 {
			affixRun := originalRuns[0]
			affixRun.Len = len(ellipsis)
			resultRuns = append(resultRuns, affixRun)
		}
	}

	bytePos := 0
	for _, run := range originalRuns {
		runEnd := bytePos + run.Len
		if runEnd > backStartIx {
			if bytePos < backStartIx {
				partial := run
				partial.Len = runEnd - backStartIx
				resultRuns = append(resultRuns, partial)
			} else {
				resultRuns = append(resultRuns, run)
			}
		}
		bytePos = runEnd
	}

	*runs = resultRuns
}

// LineFragment is a fragment of a line that can be wrapped: a text fragment or a non-text
// element with a fixed width.
type LineFragment struct {
	text      string
	isElement bool
	width     Pixels
	lenUTF8   int
}

// LineFragmentText creates a new text fragment from the given text.
func LineFragmentText(text string) LineFragment {
	return LineFragment{text: text}
}

// LineFragmentElement creates a new non-text element with the given width and UTF-8 encoded
// length.
func LineFragmentElement(width Pixels, lenUTF8 int) LineFragment {
	return LineFragment{isElement: true, width: width, lenUTF8: lenUTF8}
}

func (f LineFragment) appendWrapBoundaryCandidates(out []wrapBoundaryCandidate) []wrapBoundaryCandidate {
	if f.isElement {
		return append(out, wrapBoundaryCandidate{isElement: true, width: f.width, elementLen: f.lenUTF8})
	}
	for _, character := range f.text {
		out = append(out, wrapBoundaryCandidate{character: character})
	}
	return out
}

type wrapBoundaryCandidate struct {
	isElement  bool
	character  rune
	width      Pixels
	elementLen int
}

func (c wrapBoundaryCandidate) lenUTF8() int {
	if c.isElement {
		return c.elementLen
	}
	return utf8.RuneLen(c.character)
}

// Boundary is a boundary between two lines of text.
type Boundary struct {
	// Ix is the index of the last character in a line.
	Ix int
	// NextIndent is the indent of the next line.
	NextIndent uint32
}

func newBoundary(ix int, nextIndent uint32) Boundary {
	return Boundary{Ix: ix, NextIndent: nextIndent}
}
