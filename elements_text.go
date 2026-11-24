package gpui

import (
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

// Text is an element that renders text with an optional ID, which produces a node in the
// accessibility tree. Create one with TextMacro (upstream's text!) or NewText.
type Text struct {
	id   Option[ElementID]
	text SharedString
}

// NewText creates a Text element with a specific ID.
func NewText(id ElementID, text SharedString) Text {
	return Text{id: Some(id), text: text}
}

// NewTextInaccessible creates a Text element that is inaccessible to screen readers.
func NewTextInaccessible(text SharedString) Text {
	return Text{id: None[ElementID](), text: text}
}

// WithID returns a new Text with the given id.
func (t Text) WithID(id ElementID) Text {
	t.id = Some(id)
	return t
}

// Text returns the text that this element will display.
func (t Text) Text() SharedString {
	return t.text
}

// HashTextMacroLocationUnstableDoNotUse is the trivial FNV-1a hash of the location produced by
// TextMacro. Not covered by semver guarantees.
func HashTextMacroLocationUnstableDoNotUse(s string) uint64 {
	const basis uint64 = 0xcbf29ce484222325
	const prime uint64 = 0x100000001b3

	hash := basis
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= prime
	}
	return hash
}

// TextMacro creates an accessible Text element whose ID is derived from the caller's source
// location (upstream's text! macro; use NewText for text!(id = ..., ...)).
func TextMacro(text SharedString) Text {
	loc := CallerSourceLocation(1)
	id := ""
	if loc != nil {
		id = fmt.Sprintf("%s/%d:%d", loc.File, loc.Line, loc.Column)
	}
	return NewText(ElementIDInteger(HashTextMacroLocationUnstableDoNotUse(id)), text)
}

// ElementID returns the element's ID.
func (t Text) ElementID() (ElementID, bool) {
	return t.id.Get()
}

// SourceLocation returns nil.
func (t Text) SourceLocation() *SourceLocation {
	return nil
}

// A11yRole reports Label when the text has an ID.
func (t Text) A11yRole() (a11y.Role, bool) {
	if t.id.IsSome() {
		return a11y.RoleLabel, true
	}
	return 0, false
}

// WriteA11yInfo writes the text as the node's value.
func (t Text) WriteA11yInfo(node *a11y.Node) {
	node.SetValue(string(t.text))
}

// RequestLayout lays out the text.
func (t Text) RequestLayout(id *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, TextLayout) {
	return t.text.RequestLayout(id, inspectorID, window, cx)
}

// Prepaint records the text's bounds.
func (t Text) Prepaint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *TextLayout, window *Window, cx *App) struct{} {
	return t.text.Prepaint(id, inspectorID, bounds, requestLayout, window, cx)
}

// Paint paints the text.
func (t Text) Paint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *TextLayout, prepaint *struct{}, window *Window, cx *App) {
	t.text.Paint(id, inspectorID, bounds, requestLayout, prepaint, window, cx)
}

// IntoAnyElement converts the text into an element.
func (t Text) IntoAnyElement() AnyElement {
	return NewAnyElement[TextLayout, struct{}](t)
}

// ElementID returns no ID.
func (s SharedString) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (s SharedString) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout lays out the string in the window's text style.
func (s SharedString) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, TextLayout) {
	state := NewTextLayout()
	layoutID := state.layout(s, None[[]TextRun](), window, cx)
	return layoutID, state
}

// Prepaint records the string's bounds.
func (s SharedString) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], textLayout *TextLayout, _ *Window, _ *App) struct{} {
	textLayout.prepaint(bounds, string(s))
	return struct{}{}
}

// Paint paints the string.
func (s SharedString) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], textLayout *TextLayout, _ *struct{}, window *Window, cx *App) {
	textLayout.paint(string(s), window, cx)
}

// IntoAnyElement converts the string into a text element.
func (s SharedString) IntoAnyElement() AnyElement {
	return NewAnyElement[TextLayout, struct{}](s)
}

// StyledText renders text with runs of different styles.
type StyledText struct {
	text                       SharedString
	runs                       Option[[]TextRun]
	delayedHighlights          Option[[]textHighlight]
	delayedFontFamilyOverrides Option[[]fontFamilyOverride]
	layout                     TextLayout
}

type textHighlight struct {
	rng   Range
	style HighlightStyle
}

type fontFamilyOverride struct {
	rng    Range
	family SharedString
}

// NewStyledText creates a styled text element from the given string.
func NewStyledText(text SharedString) *StyledText {
	return &StyledText{
		text:   text,
		layout: NewTextLayout(),
	}
}

// Layout returns the layout for this element, to map indices to pixels and back.
func (s *StyledText) Layout() TextLayout {
	return s.layout
}

// WithDefaultHighlights sets the styling attributes of the text from defaultStyle and the
// ranges of text that have had their style customized.
func (s *StyledText) WithDefaultHighlights(defaultStyle *TextStyle, highlights iter.Seq2[Range, HighlightStyle]) *StyledText {
	if gpuiutil.DebugAssertions && s.delayedHighlights.IsSome() {
		panic("Can't use `with_default_highlights` and `with_highlights`")
	}
	runs := styledTextComputeRuns(string(s.text), defaultStyle, collectHighlights(highlights))
	return s.WithRuns(runs)
}

// WithHighlights sets the ranges of text whose style is customized on top of the inherited
// text style.
func (s *StyledText) WithHighlights(highlights iter.Seq2[Range, HighlightStyle]) *StyledText {
	if gpuiutil.DebugAssertions && s.runs.IsSome() {
		panic("Can't use `with_highlights` and `with_default_highlights`")
	}
	collected := collectHighlights(highlights)
	if gpuiutil.DebugAssertions {
		for _, h := range collected {
			debugAssertCharBoundary(string(s.text), h.rng.Start)
			debugAssertCharBoundary(string(s.text), h.rng.End)
		}
	}
	s.delayedHighlights = Some(collected)
	return s
}

func debugAssertCharBoundary(text string, index int) {
	if !textIsCharBoundary(text, index) {
		panic("assertion failed: text.is_char_boundary(index)")
	}
}

func collectHighlights(highlights iter.Seq2[Range, HighlightStyle]) []textHighlight {
	var out []textHighlight
	if highlights == nil {
		return out
	}
	for rng, style := range highlights {
		out = append(out, textHighlight{rng: rng, style: style})
	}
	return out
}

func styledTextComputeRuns(text string, defaultStyle *TextStyle, highlights []textHighlight) []TextRun {
	var runs []TextRun
	ix := 0
	for _, h := range highlights {
		if ix < h.rng.Start {
			if gpuiutil.DebugAssertions {
				debugAssertCharBoundary(text, h.rng.Start)
			}
			runs = append(runs, defaultStyle.ToRun(h.rng.Start-ix))
		}
		if gpuiutil.DebugAssertions {
			debugAssertCharBoundary(text, h.rng.End)
		}
		runs = append(runs, defaultStyle.Highlight(h.style).ToRun(h.rng.End-h.rng.Start))
		ix = h.rng.End
	}
	if ix < len(text) {
		runs = append(runs, defaultStyle.ToRun(len(text)-ix))
	}
	return runs
}

// WithFontFamilyOverrides overrides the font family for byte ranges of the text, resolved at
// layout time on top of the inherited text style. The overrides must be sorted by range start,
// non-overlapping and on character boundaries.
func (s *StyledText) WithFontFamilyOverrides(overrides iter.Seq2[Range, SharedString]) *StyledText {
	var collected []fontFamilyOverride
	if overrides != nil {
		for rng, family := range overrides {
			if gpuiutil.DebugAssertions {
				debugAssertCharBoundary(string(s.text), rng.Start)
				debugAssertCharBoundary(string(s.text), rng.End)
			}
			collected = append(collected, fontFamilyOverride{rng: rng, family: family})
		}
	}
	s.delayedFontFamilyOverrides = Some(collected)
	return s
}

func applyFontFamilyOverrides(runs []TextRun, overrides []fontFamilyOverride) {
	byteOffset := 0
	overrideIdx := 0
	for i := range runs {
		run := &runs[i]
		runEnd := byteOffset + run.Len
		for overrideIdx < len(overrides) && overrides[overrideIdx].rng.End <= byteOffset {
			overrideIdx++
		}
		if overrideIdx < len(overrides) {
			o := overrides[overrideIdx]
			if byteOffset >= o.rng.Start && runEnd <= o.rng.End {
				run.Font.Family = o.family
			}
		}
		byteOffset = runEnd
	}
}

// WithRuns sets the text runs for this piece of text.
func (s *StyledText) WithRuns(runs []TextRun) *StyledText {
	text := string(s.text)
	for _, run := range runs {
		if run.Len > len(text) || !textIsCharBoundary(text, run.Len) {
			if gpuiutil.DebugAssertions {
				panic(fmt.Sprintf("invalid text run. Text: '%s', run: %+v", text, run))
			}
			panic("invalid text run")
		}
		text = text[run.Len:]
	}
	if text != "" {
		panic("invalid text run")
	}
	s.runs = Some(runs)
	return s
}

// ElementID returns no ID.
func (s *StyledText) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (s *StyledText) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout computes the runs and lays out the text.
func (s *StyledText) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	fontFamilyOverrides := s.delayedFontFamilyOverrides.Take()
	runs := s.runs.Take()
	if runs.IsNone() {
		if delayed, ok := s.delayedHighlights.Take().Get(); ok {
			style := window.TextStyle()
			runs = Some(styledTextComputeRuns(string(s.text), &style, delayed))
		}
	}

	if overrides, ok := fontFamilyOverrides.Get(); ok {
		if runs.IsNone() {
			runs = Some([]TextRun{window.TextStyle().ToRun(len(s.text))})
		}
		applyFontFamilyOverrides(runs.Unwrap(), overrides)
	}

	layoutID := s.layout.layout(s.text, runs, window, cx)
	return layoutID, struct{}{}
}

// Prepaint records the text's bounds.
func (s *StyledText) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, _ *Window, _ *App) struct{} {
	s.layout.prepaint(bounds, string(s.text))
	return struct{}{}
}

// Paint paints the text.
func (s *StyledText) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, _ *struct{}, window *Window, cx *App) {
	s.layout.paint(string(s.text), window, cx)
}

// IntoAnyElement converts the styled text into an element.
func (s *StyledText) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](s)
}

// TextLayout is the layout of a text element, used to map indices to pixels and back. Copies
// share the same layout.
type TextLayout struct {
	cell *textLayoutCell
}

type textLayoutCell struct {
	inner *textLayoutInner
}

type textPaintPass uint8

const (
	textPaintPassAll textPaintPass = iota
	textPaintPassBackground
	textPaintPassForeground
)

type textLayoutInner struct {
	len           int
	lines         []WrappedLine
	lineHeight    Pixels
	wrapWidth     Option[Pixels]
	truncateWidth Option[Pixels]
	size          Option[Size[Pixels]]
	bounds        Option[Bounds[Pixels]]
}

// NewTextLayout creates an empty text layout, upstream's TextLayout::default().
func NewTextLayout() TextLayout {
	return TextLayout{cell: &textLayoutCell{}}
}

func (l TextLayout) state() *textLayoutInner {
	if l.cell == nil {
		return nil
	}
	return l.cell.inner
}

func (l TextLayout) mustState() *textLayoutInner {
	inner := l.state()
	if inner == nil {
		panic("measurement has not been performed")
	}
	return inner
}

func (l TextLayout) layout(text SharedString, runs Option[[]TextRun], window *Window, _ *App) LayoutID {
	textStyle := window.TextStyle()
	fontSize := textStyle.FontSize.ToPixels(window.RemSize())
	lineHeight := window.PixelSnap(textStyle.LineHeight.ToPixels(fontSize.ToAbsoluteLength(), window.RemSize()))

	textRuns, ok := runs.Get()
	if !ok {
		textRuns = []TextRun{textStyle.ToRun(len(text))}
	}
	elementState := l
	return window.RequestMeasuredLayout(DefaultStyle(), func(knownDimensions SizeOf[Option[Pixels]], availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels] {
		wrapWidth := None[Pixels]()
		if textStyle.WhiteSpace == WhiteSpaceNormal {
			wrapWidth = knownDimensions.Width
			if wrapWidth.IsNone() {
				if x, ok := availableSpace.Width.AsDefinite(); ok {
					wrapWidth = Some(x)
				}
			}
		}

		truncateWidth := None[Pixels]()
		truncationAffix := SharedString("")
		truncateFrom := TruncateFromEnd
		if textOverflow, ok := textStyle.TextOverflow.Get(); ok {
			truncateWidth = knownDimensions.Width
			if truncateWidth.IsNone() {
				if x, ok := availableSpace.Width.AsDefinite(); ok {
					if maxLines, ok := textStyle.LineClamp.Get(); ok {
						truncateWidth = Some(x * Pixels(maxLines))
					} else {
						truncateWidth = Some(x)
					}
				}
			}

			if s, ok := textOverflow.AsTruncate(); ok {
				truncationAffix, truncateFrom = s, TruncateFromEnd
			} else if s, ok := textOverflow.AsTruncateStart(); ok {
				truncationAffix, truncateFrom = s, TruncateFromStart
			} else if s, ok := textOverflow.AsTruncateMiddle(); ok {
				truncationAffix, truncateFrom = s, TruncateFromMiddle
			}
		}

		if cached := elementState.state(); cached != nil {
			if size, ok := cached.size.Get(); ok &&
				(wrapWidth.IsNone() || wrapWidth == cached.wrapWidth) &&
				truncateWidth.IsNone() &&
				cached.truncateWidth.IsNone() {
				return size
			}
		}

		shapedText, shapedRuns := text, textRuns
		if tw, ok := truncateWidth.Get(); ok {
			lineWrapper := cx.TextSystem().LineWrapper(textStyle.Font(), fontSize)
			maxLines, hasClamp := textStyle.LineClamp.Get()
			ww, hasWrap := wrapWidth.Get()
			if hasClamp && hasWrap {
				shapedText, shapedRuns = lineWrapper.TruncateWrappedLine(text, ww, maxLines, string(truncationAffix), textRuns, truncateFrom)
			} else if unclipped, err := window.TextSystem().ShapeText(text, fontSize, textRuns, None[Pixels](), None[int]()); err == nil && allLinesFit(unclipped, lineHeight, tw) {
				shapedText, shapedRuns = text, textRuns
			} else {
				if err != nil {
					gpuiutil.LogErr(err)
				}
				shapedText, shapedRuns = lineWrapper.TruncateLine(text, tw, string(truncationAffix), textRuns, truncateFrom)
			}
			lineWrapper.Release()
		}
		length := len(shapedText)

		lines, err := window.TextSystem().ShapeText(shapedText, fontSize, shapedRuns, wrapWidth, textStyle.LineClamp)
		if err != nil {
			gpuiutil.LogErr(err)
			elementState.cell.inner = &textLayoutInner{
				len:           0,
				lineHeight:    lineHeight,
				wrapWidth:     wrapWidth,
				truncateWidth: truncateWidth,
				size:          Some(Size[Pixels]{}),
				bounds:        None[Bounds[Pixels]](),
			}
			return Size[Pixels]{}
		}

		var size Size[Pixels]
		for _, line := range lines {
			lineSize := line.Size(lineHeight)
			size.Height += lineSize.Height
			size.Width = max(size.Width, lineSize.Width).Ceil()
		}

		elementState.cell.inner = &textLayoutInner{
			lines:         lines,
			len:           length,
			lineHeight:    lineHeight,
			wrapWidth:     wrapWidth,
			truncateWidth: truncateWidth,
			size:          Some(size),
			bounds:        None[Bounds[Pixels]](),
		}

		return size
	})
}

func allLinesFit(lines []WrappedLine, lineHeight, truncateWidth Pixels) bool {
	for _, line := range lines {
		if line.Size(lineHeight).Width > truncateWidth {
			return false
		}
	}
	return true
}

func (l TextLayout) prepaint(bounds Bounds[Pixels], text string) {
	inner := l.state()
	if inner == nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: measurement has not been performed on %s", text))
	}
	inner.bounds = Some(bounds)
}

func (l TextLayout) paint(text string, window *Window, cx *App) {
	if err := l.paintLines(textPaintPassAll, window, cx); err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: failed to paint %s: %v", text, err))
	}
}

// PaintBackground paints the backgrounds of the text runs without the glyphs, so callers can
// paint content between the backgrounds and the text; follow it with PaintForeground.
func (l TextLayout) PaintBackground(window *Window, cx *App) error {
	return l.paintLines(textPaintPassBackground, window, cx)
}

// PaintForeground paints the glyphs and decorations of the text without the run backgrounds.
func (l TextLayout) PaintForeground(window *Window, cx *App) error {
	return l.paintLines(textPaintPassForeground, window, cx)
}

func (l TextLayout) paintLines(pass textPaintPass, window *Window, cx *App) error {
	inner := l.state()
	if inner == nil {
		return errors.New("measurement has not been performed")
	}
	bounds, ok := inner.bounds.Get()
	if !ok {
		return errors.New("prepaint has not been performed")
	}

	lineHeight := inner.lineHeight
	lineOrigin := bounds.Origin
	textStyle := window.TextStyle()
	for _, line := range inner.lines {
		if pass == textPaintPassAll || pass == textPaintPassBackground {
			if err := line.PaintBackground(lineOrigin, lineHeight, textStyle.TextAlign, Some(bounds), window, cx); err != nil {
				gpuiutil.LogErr(err)
			}
		}
		if pass == textPaintPassAll || pass == textPaintPassForeground {
			if err := line.Paint(lineOrigin, lineHeight, textStyle.TextAlign, Some(bounds), window, cx); err != nil {
				gpuiutil.LogErr(err)
			}
		}
		lineOrigin.Y += line.Size(lineHeight).Height
	}
	return nil
}

// IndexForPosition returns the byte index into the input at the pixel position; ok is false
// (upstream's Err) when the position is not over a character, with the closest index.
func (l TextLayout) IndexForPosition(position Point[Pixels]) (int, bool) {
	inner := l.state()
	if inner == nil {
		panic("measurement has not been performed")
	}
	bounds := inner.bounds.Expect("prepaint has not been performed")

	if position.Y < bounds.Top() {
		return 0, false
	}

	lineHeight := inner.lineHeight
	lineOrigin := bounds.Origin
	lineStartIx := 0
	for _, line := range inner.lines {
		lineBottom := lineOrigin.Y + line.Size(lineHeight).Height
		if position.Y > lineBottom {
			lineOrigin.Y = lineBottom
			lineStartIx += line.Len() + 1
		} else {
			positionWithinLine := position.Sub(lineOrigin)
			indexWithinLine, ok := line.IndexForPosition(positionWithinLine, lineHeight)
			return lineStartIx + indexWithinLine, ok
		}
	}

	return saturatingSub(lineStartIx, 1), false
}

// PositionForIndex returns the pixel position for the given byte index.
func (l TextLayout) PositionForIndex(index int) (Point[Pixels], bool) {
	inner := l.state()
	if inner == nil {
		panic("measurement has not been performed")
	}
	bounds := inner.bounds.Expect("prepaint has not been performed")
	lineHeight := inner.lineHeight

	lineOrigin := bounds.Origin
	lineStartIx := 0

	for _, line := range inner.lines {
		lineEndIx := lineStartIx + line.Len()
		if index < lineStartIx {
			break
		} else if index > lineEndIx {
			lineOrigin.Y += line.Size(lineHeight).Height
			lineStartIx = lineEndIx + 1
			continue
		}
		ixWithinLine := index - lineStartIx
		position, ok := line.PositionForIndex(ixWithinLine, lineHeight)
		if !ok {
			return Point[Pixels]{}, false
		}
		return lineOrigin.Add(position), true
	}

	return Point[Pixels]{}, false
}

// LineLayoutForIndex returns the layout of the line containing the given byte index.
func (l TextLayout) LineLayoutForIndex(index int) (*WrappedLineLayout, bool) {
	inner := l.state()
	if inner == nil {
		panic("measurement has not been performed")
	}
	lineStartIx := 0

	for _, line := range inner.lines {
		lineEndIx := lineStartIx + line.Len()
		if index < lineStartIx {
			break
		} else if index > lineEndIx {
			lineStartIx = lineEndIx + 1
			continue
		}
		return line.WrappedLineLayout, true
	}

	return nil, false
}

// LineLayouts returns all line layouts in source order.
func (l TextLayout) LineLayouts() []*WrappedLineLayout {
	inner := l.state()
	if inner == nil {
		panic("measurement has not been performed")
	}
	layouts := make([]*WrappedLineLayout, 0, len(inner.lines))
	for _, line := range inner.lines {
		layouts = append(layouts, line.WrappedLineLayout)
	}
	return layouts
}

// Bounds returns the bounds of this layout.
func (l TextLayout) Bounds() Bounds[Pixels] {
	return l.mustUnwrap().bounds.Unwrap()
}

// LineHeight returns the line height of this layout.
func (l TextLayout) LineHeight() Pixels {
	return l.mustUnwrap().lineHeight
}

// Len returns the UTF-8 length of the underlying text.
func (l TextLayout) Len() int {
	return l.mustUnwrap().len
}

func (l TextLayout) mustUnwrap() *textLayoutInner {
	inner := l.state()
	if inner == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return inner
}

// Text returns the text of this layout, lines joined with newlines.
func (l TextLayout) Text() string {
	inner := l.mustUnwrap()
	parts := make([]string, 0, len(inner.lines))
	for _, line := range inner.lines {
		parts = append(parts, string(line.Text))
	}
	return strings.Join(parts, "\n")
}

// WrappedText returns the text of this layout with soft wraps as newlines.
func (l TextLayout) WrappedText() string {
	var accumulator strings.Builder

	for _, wrapped := range l.mustUnwrap().lines {
		seen := 0
		for _, boundary := range wrapped.WrapBoundaries {
			index := wrapped.UnwrappedLayout.Runs[boundary.RunIx].Glyphs[boundary.GlyphIx].Index

			accumulator.WriteString(string(wrapped.Text[seen:index]))
			accumulator.WriteByte('\n')
			seen = index
		}
		accumulator.WriteString(string(wrapped.Text[seen:]))
		accumulator.WriteByte('\n')
	}
	s := accumulator.String()
	if s != "" {
		s = s[:len(s)-1]
	}
	return s
}

// InteractiveText is a text element that can be interacted with.
type InteractiveText struct {
	elementID       ElementID
	text            *StyledText
	clickListener   func(ranges []Range, event interactiveTextClickEvent, window *Window, cx *App)
	hoverListener   func(index Option[int], event MouseMoveEvent, window *Window, cx *App)
	tooltipBuilder  func(index int, window *Window, cx *App) (AnyView, bool)
	tooltipID       Option[TooltipID]
	clickableRanges []Range
}

type interactiveTextClickEvent struct {
	mouseDownIndex int
	mouseUpIndex   int
}

// InteractiveTextState is the per-element state of an InteractiveText.
type InteractiveTextState struct {
	mouseDownIndex         *Option[int]
	hoveredIndex           *Option[int]
	activeTooltip          *activeTooltipCell
	longPressTooltipActive *bool
}

func newInteractiveTextState() InteractiveTextState {
	return InteractiveTextState{
		mouseDownIndex:         new(Option[int]),
		hoveredIndex:           new(Option[int]),
		activeTooltip:          &activeTooltipCell{},
		longPressTooltipActive: new(bool),
	}
}

// NewInteractiveText creates an InteractiveText from the given text.
func NewInteractiveText(id ElementID, text *StyledText) *InteractiveText {
	return &InteractiveText{
		elementID: id,
		text:      text,
	}
}

func rangeContains(r Range, ix int) bool {
	return r.Start <= ix && ix < r.End
}

// OnClick calls listener when the user clicks on one of the given ranges, passing the index of
// the clicked range.
func (t *InteractiveText) OnClick(ranges []Range, listener func(rangeIx int, window *Window, cx *App)) *InteractiveText {
	t.clickListener = func(ranges []Range, event interactiveTextClickEvent, window *Window, cx *App) {
		for rangeIx, r := range ranges {
			if rangeContains(r, event.mouseDownIndex) && rangeContains(r, event.mouseUpIndex) {
				listener(rangeIx, window, cx)
			}
		}
	}
	t.clickableRanges = ranges
	return t
}

// OnHover calls listener when the mouse moves over a character within the text, passing the
// index of the hovered character, or None if the mouse leaves the text.
func (t *InteractiveText) OnHover(listener func(index Option[int], event MouseMoveEvent, window *Window, cx *App)) *InteractiveText {
	t.hoverListener = listener
	return t
}

// Tooltip sets a builder for the tooltip of a given character index in the string.
func (t *InteractiveText) Tooltip(builder func(index int, window *Window, cx *App) (AnyView, bool)) *InteractiveText {
	t.tooltipBuilder = builder
	return t
}

// ElementID returns the element's ID.
func (t *InteractiveText) ElementID() (ElementID, bool) {
	return t.elementID, true
}

// SourceLocation returns nil.
func (t *InteractiveText) SourceLocation() *SourceLocation {
	return nil
}

// A11yRole reports Label.
func (t *InteractiveText) A11yRole() (a11y.Role, bool) {
	return a11y.RoleLabel, true
}

// WriteA11yInfo writes the text as the node's value.
func (t *InteractiveText) WriteA11yInfo(node *a11y.Node) {
	node.SetValue(string(t.text.text))
}

// RequestLayout lays out the inner text.
func (t *InteractiveText) RequestLayout(_ *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	return t.text.RequestLayout(nil, inspectorID, window, cx)
}

// Prepaint registers the tooltip and inserts the hitbox.
func (t *InteractiveText) Prepaint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], state *struct{}, window *Window, cx *App) Hitbox {
	return window.WithOptionalElementState(globalID, func(interactiveState Option[Option[InteractiveTextState]], window *Window) (Hitbox, Option[InteractiveTextState]) {
		var stateOut Option[InteractiveTextState]
		if outer, ok := interactiveState.Get(); ok {
			st, ok := outer.Get()
			if !ok {
				st = newInteractiveTextState()
			}
			if t.tooltipBuilder != nil {
				t.tooltipID = setTooltipOnWindow(st.activeTooltip, window)
			} else {
				*st.longPressTooltipActive = false
				st.activeTooltip.set(None[activeTooltipState]())
			}
			stateOut = Some(st)
		}

		t.text.Prepaint(nil, inspectorID, bounds, state, window, cx)
		hitbox := window.InsertHitbox(bounds, HitboxBehaviorNormal)
		return hitbox, stateOut
	})
}

// Paint registers the mouse and tooltip handlers and paints the text.
func (t *InteractiveText) Paint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, hitbox *Hitbox, window *Window, cx *App) {
	currentView := window.CurrentView()
	textLayout := t.text.Layout()
	if globalID == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	window.WithElementState(globalID, func(interactiveState Option[InteractiveTextState], window *Window) (struct{}, InteractiveTextState) {
		st, ok := interactiveState.Get()
		if !ok {
			st = newInteractiveTextState()
		}
		if clickListener := t.clickListener; clickListener != nil {
			t.clickListener = nil
			mousePosition := window.MousePosition()
			if ix, ok := textLayout.IndexForPosition(mousePosition); ok {
				for _, r := range t.clickableRanges {
					if rangeContains(r, ix) {
						window.SetCursorStyle(CursorStylePointingHand, *hitbox)
						break
					}
				}
			}

			mouseDown := st.mouseDownIndex
			if mouseDownIndex, ok := mouseDown.Get(); ok {
				hitbox := *hitbox
				clickableRanges := t.clickableRanges
				t.clickableRanges = nil
				window.OnMouseEvent(func(event *MouseUpEvent, phase DispatchPhase, window *Window, cx *App) {
					if phase.Bubble() && hitbox.IsHovered(window) {
						if mouseUpIndex, ok := textLayout.IndexForPosition(event.Position); ok {
							clickListener(clickableRanges, interactiveTextClickEvent{mouseDownIndex: mouseDownIndex, mouseUpIndex: mouseUpIndex}, window, cx)
						}

						mouseDown.Take()
						window.Refresh()
					}
				})
			} else {
				hitbox := *hitbox
				window.OnMouseEvent(func(event *MouseDownEvent, phase DispatchPhase, window *Window, _ *App) {
					if phase.Bubble() && hitbox.IsHovered(window) {
						if mouseDownIndex, ok := textLayout.IndexForPosition(event.Position); ok {
							*mouseDown = Some(mouseDownIndex)
							window.Refresh()
						}
					}
				})
			}
		}

		{
			hoverListener := t.hoverListener
			t.hoverListener = nil
			hitbox := *hitbox
			hoveredIndex := st.hoveredIndex
			window.OnMouseEvent(func(event *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
				if phase.Bubble() && hitbox.IsHovered(window) {
					current := *hoveredIndex
					updated := None[int]()
					if ix, ok := textLayout.IndexForPosition(event.Position); ok {
						updated = Some(ix)
					}
					if current != updated {
						*hoveredIndex = updated
						if hoverListener != nil {
							hoverListener(updated, *event, window, cx)
						}
						cx.Notify(currentView)
					}
				}
			})
		}

		if tooltipBuilder := t.tooltipBuilder; tooltipBuilder != nil {
			activeTooltip := st.activeTooltip
			tooltipIsHoverable := false
			buildTooltip := func(window *Window, cx *App) (AnyView, bool, bool) {
				position, ok := textLayout.IndexForPosition(window.MousePosition())
				if !ok {
					return AnyView{}, false, false
				}
				view, ok := tooltipBuilder(position, window, cx)
				if !ok {
					return AnyView{}, false, false
				}
				return view, tooltipIsHoverable, true
			}

			sourceBounds := hitbox.Bounds
			pendingMouseDown := st.mouseDownIndex
			checkIsHoveredDuringPrepaint := func(window *Window) bool {
				_, ok := textLayout.IndexForPosition(window.MousePosition())
				return ok && sourceBounds.Contains(window.MousePosition()) && pendingMouseDown.IsNone()
			}

			hitboxCopy := *hitbox
			checkIsHovered := func(window *Window) bool {
				_, ok := textLayout.IndexForPosition(window.MousePosition())
				return ok && hitboxCopy.IsHovered(window) && pendingMouseDown.IsNone()
			}

			registerTooltipMouseHandlers(activeTooltip, t.tooltipID, buildTooltip, checkIsHovered, checkIsHoveredDuringPrepaint, st.longPressTooltipActive, None[time.Duration](), window)
		}

		var unit struct{}
		t.text.Paint(nil, inspectorID, bounds, &unit, &unit, window, cx)

		return struct{}{}, st
	})
}

// IntoAnyElement converts the interactive text into an element.
func (t *InteractiveText) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, Hitbox](t)
}
