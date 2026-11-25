package main

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/go-text/typesetting/segmenter"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Backspace deletes backwards.
type Backspace struct{}

// Delete deletes forwards.
type Delete struct{}

// Left moves the cursor left.
type Left struct{}

// Right moves the cursor right.
type Right struct{}

// SelectLeft extends the selection left.
type SelectLeft struct{}

// SelectRight extends the selection right.
type SelectRight struct{}

// SelectAll selects all text.
type SelectAll struct{}

// Home moves the cursor to the start.
type Home struct{}

// End moves the cursor to the end.
type End struct{}

// ShowCharacterPalette shows the character palette.
type ShowCharacterPalette struct{}

// Paste pastes from the clipboard.
type Paste struct{}

// Cut cuts the selection.
type Cut struct{}

// Copy copies the selection.
type Copy struct{}

// Quit quits the example.
type Quit struct{}

// Name returns the action name.
func (Backspace) Name() string { return "text_input::Backspace" }

// Name returns the action name.
func (Delete) Name() string { return "text_input::Delete" }

// Name returns the action name.
func (Left) Name() string { return "text_input::Left" }

// Name returns the action name.
func (Right) Name() string { return "text_input::Right" }

// Name returns the action name.
func (SelectLeft) Name() string { return "text_input::SelectLeft" }

// Name returns the action name.
func (SelectRight) Name() string { return "text_input::SelectRight" }

// Name returns the action name.
func (SelectAll) Name() string { return "text_input::SelectAll" }

// Name returns the action name.
func (Home) Name() string { return "text_input::Home" }

// Name returns the action name.
func (End) Name() string { return "text_input::End" }

// Name returns the action name.
func (ShowCharacterPalette) Name() string { return "text_input::ShowCharacterPalette" }

// Name returns the action name.
func (Paste) Name() string { return "text_input::Paste" }

// Name returns the action name.
func (Cut) Name() string { return "text_input::Cut" }

// Name returns the action name.
func (Copy) Name() string { return "text_input::Copy" }

// Name returns the action name.
func (Quit) Name() string { return "text_input::Quit" }

// BoxedClone clones the action.
func (a Backspace) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Delete) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Left) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Right) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a SelectLeft) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a SelectRight) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a SelectAll) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Home) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a End) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a ShowCharacterPalette) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Paste) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Cut) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Copy) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Quit) BoxedClone() gpui.Action { return a }

func actionEq[A gpui.Action](other gpui.Action) bool {
	_, ok := other.(A)
	return ok
}

// PartialEq reports whether other is the same action.
func (Backspace) PartialEq(other gpui.Action) bool { return actionEq[Backspace](other) }

// PartialEq reports whether other is the same action.
func (Delete) PartialEq(other gpui.Action) bool { return actionEq[Delete](other) }

// PartialEq reports whether other is the same action.
func (Left) PartialEq(other gpui.Action) bool { return actionEq[Left](other) }

// PartialEq reports whether other is the same action.
func (Right) PartialEq(other gpui.Action) bool { return actionEq[Right](other) }

// PartialEq reports whether other is the same action.
func (SelectLeft) PartialEq(other gpui.Action) bool { return actionEq[SelectLeft](other) }

// PartialEq reports whether other is the same action.
func (SelectRight) PartialEq(other gpui.Action) bool { return actionEq[SelectRight](other) }

// PartialEq reports whether other is the same action.
func (SelectAll) PartialEq(other gpui.Action) bool { return actionEq[SelectAll](other) }

// PartialEq reports whether other is the same action.
func (Home) PartialEq(other gpui.Action) bool { return actionEq[Home](other) }

// PartialEq reports whether other is the same action.
func (End) PartialEq(other gpui.Action) bool { return actionEq[End](other) }

// PartialEq reports whether other is the same action.
func (ShowCharacterPalette) PartialEq(other gpui.Action) bool {
	return actionEq[ShowCharacterPalette](other)
}

// PartialEq reports whether other is the same action.
func (Paste) PartialEq(other gpui.Action) bool { return actionEq[Paste](other) }

// PartialEq reports whether other is the same action.
func (Cut) PartialEq(other gpui.Action) bool { return actionEq[Cut](other) }

// PartialEq reports whether other is the same action.
func (Copy) PartialEq(other gpui.Action) bool { return actionEq[Copy](other) }

// PartialEq reports whether other is the same action.
func (Quit) PartialEq(other gpui.Action) bool { return actionEq[Quit](other) }

func init() {
	for _, a := range []gpui.Action{
		Backspace{}, Delete{}, Left{}, Right{}, SelectLeft{}, SelectRight{}, SelectAll{},
		Home{}, End{}, ShowCharacterPalette{}, Paste{}, Cut{}, Copy{}, Quit{},
	} {
		gpui.RegisterAction(gpui.ActionData{
			Name:  a.Name(),
			Type:  reflect.TypeOf(a),
			Build: func(jsontext.Value) (gpui.Action, error) { return a.BoxedClone(), nil },
		})
	}
}

// TextInput is a single-line text input.
type TextInput struct {
	gpui.EntityInputHandlerDefaults[TextInput]
	focusHandle       gpui.FocusHandle
	content           gpui.SharedString
	placeholder       gpui.SharedString
	selectedRange     gpui.Range
	selectionReversed bool
	markedRange       gpui.Option[gpui.Range]
	lastLayout        gpui.Option[gpui.ShapedLine]
	lastBounds        gpui.Option[gpui.Bounds[gpui.Pixels]]
	isSelecting       bool
}

func newTextInput(input *TextInput) *TextInput {
	gpui.BindEntityInputHandlerDefaults(&input.EntityInputHandlerDefaults, input)
	return input
}

// Drop releases the input's focus handle.
func (t *TextInput) Drop(_ *gpui.App) {
	t.focusHandle.Release()
}

func rangeIsEmpty(r gpui.Range) bool {
	return r.Start >= r.End
}

func (t *TextInput) left(_ *Left, window *gpui.Window, cx *gpui.Context[TextInput]) {
	if rangeIsEmpty(t.selectedRange) {
		t.moveTo(t.previousBoundary(t.cursorOffset()), window, cx)
	} else {
		t.moveTo(t.selectedRange.Start, window, cx)
	}
}

func (t *TextInput) right(_ *Right, window *gpui.Window, cx *gpui.Context[TextInput]) {
	if rangeIsEmpty(t.selectedRange) {
		t.moveTo(t.nextBoundary(t.selectedRange.End), window, cx)
	} else {
		t.moveTo(t.selectedRange.End, window, cx)
	}
}

func (t *TextInput) selectLeft(_ *SelectLeft, window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.selectTo(t.previousBoundary(t.cursorOffset()), window, cx)
}

func (t *TextInput) selectRight(_ *SelectRight, window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.selectTo(t.nextBoundary(t.cursorOffset()), window, cx)
}

func (t *TextInput) selectAll(_ *SelectAll, window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.moveTo(0, window, cx)
	t.selectTo(len(t.content), window, cx)
}

func (t *TextInput) home(_ *Home, window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.moveTo(0, window, cx)
}

func (t *TextInput) end(_ *End, window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.moveTo(len(t.content), window, cx)
}

func (t *TextInput) backspace(_ *Backspace, window *gpui.Window, cx *gpui.Context[TextInput]) {
	if rangeIsEmpty(t.selectedRange) {
		prev := t.previousBoundary(t.cursorOffset())
		if t.cursorOffset() == prev {
			window.PlaySystemBell()
			return
		}
		t.selectTo(prev, window, cx)
	}
	t.ReplaceTextInRange(gpui.None[gpui.Range](), "", window, cx)
}

func (t *TextInput) delete(_ *Delete, window *gpui.Window, cx *gpui.Context[TextInput]) {
	if rangeIsEmpty(t.selectedRange) {
		next := t.nextBoundary(t.cursorOffset())
		if t.cursorOffset() == next {
			window.PlaySystemBell()
			return
		}
		t.selectTo(next, window, cx)
	}
	t.ReplaceTextInRange(gpui.None[gpui.Range](), "", window, cx)
}

func (t *TextInput) onMouseDown(event *gpui.MouseDownEvent, window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.isSelecting = true
	window.RequestVirtualKeyboard()

	if event.Modifiers.Shift {
		t.selectTo(t.indexForMousePosition(event.Position), window, cx)
	} else {
		t.moveTo(t.indexForMousePosition(event.Position), window, cx)
	}
}

func (t *TextInput) onMouseUp(_ *gpui.MouseUpEvent, _ *gpui.Window, _ *gpui.Context[TextInput]) {
	t.isSelecting = false
}

func (t *TextInput) onMouseMove(event *gpui.MouseMoveEvent, window *gpui.Window, cx *gpui.Context[TextInput]) {
	if t.isSelecting {
		t.selectTo(t.indexForMousePosition(event.Position), window, cx)
	}
}

func (t *TextInput) showCharacterPalette(_ *ShowCharacterPalette, window *gpui.Window, _ *gpui.Context[TextInput]) {
	window.ShowCharacterPalette()
}

func (t *TextInput) paste(_ *Paste, window *gpui.Window, cx *gpui.Context[TextInput]) {
	if item, ok := cx.ReadFromClipboard(); ok {
		if text, ok := item.Text(); ok {
			t.ReplaceTextInRange(gpui.None[gpui.Range](), strings.ReplaceAll(text, "\n", " "), window, cx)
		}
	}
}

func (t *TextInput) copy(_ *Copy, _ *gpui.Window, cx *gpui.Context[TextInput]) {
	if !rangeIsEmpty(t.selectedRange) {
		cx.WriteToClipboard(gpui.NewClipboardItemString(string(t.content[t.selectedRange.Start:t.selectedRange.End])))
	}
}

func (t *TextInput) cut(_ *Cut, window *gpui.Window, cx *gpui.Context[TextInput]) {
	if !rangeIsEmpty(t.selectedRange) {
		cx.WriteToClipboard(gpui.NewClipboardItemString(string(t.content[t.selectedRange.Start:t.selectedRange.End])))
		t.ReplaceTextInRange(gpui.None[gpui.Range](), "", window, cx)
	}
}

func (t *TextInput) moveTo(offset int, window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.selectedRange = gpui.Range{Start: offset, End: offset}
	t.selectionReversed = false
	window.NotifyTextSelectionChanged(t.focusHandle)
	cx.Notify()
}

func (t *TextInput) cursorOffset() int {
	if t.selectionReversed {
		return t.selectedRange.Start
	}
	return t.selectedRange.End
}

func (t *TextInput) indexForMousePosition(position gpui.Point[gpui.Pixels]) int {
	if len(t.content) == 0 {
		return 0
	}

	bounds, okBounds := t.lastBounds.Get()
	line, okLine := t.lastLayout.Get()
	if !okBounds || !okLine {
		return 0
	}
	if position.Y < bounds.Top() {
		return 0
	}
	if position.Y > bounds.Bottom() {
		return len(t.content)
	}
	return line.ClosestIndexForX(position.X - bounds.Left())
}

func (t *TextInput) selectTo(offset int, window *gpui.Window, cx *gpui.Context[TextInput]) {
	if t.selectionReversed {
		t.selectedRange.Start = offset
	} else {
		t.selectedRange.End = offset
	}
	if t.selectedRange.End < t.selectedRange.Start {
		t.selectionReversed = !t.selectionReversed
		t.selectedRange = gpui.Range{Start: t.selectedRange.End, End: t.selectedRange.Start}
	}
	window.NotifyTextSelectionChanged(t.focusHandle)
	cx.Notify()
}

func lenUTF16(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

func (t *TextInput) offsetFromUTF16(offset int) int {
	utf8Offset := 0
	utf16Count := 0

	for _, ch := range string(t.content) {
		if utf16Count >= offset {
			break
		}
		utf16Count += lenUTF16(ch)
		utf8Offset += utf8.RuneLen(ch)
	}

	return utf8Offset
}

func (t *TextInput) offsetToUTF16(offset int) int {
	utf16Offset := 0
	utf8Count := 0

	for _, ch := range string(t.content) {
		if utf8Count >= offset {
			break
		}
		utf8Count += utf8.RuneLen(ch)
		utf16Offset += lenUTF16(ch)
	}

	return utf16Offset
}

func (t *TextInput) rangeToUTF16(r gpui.Range) gpui.Range {
	return gpui.Range{Start: t.offsetToUTF16(r.Start), End: t.offsetToUTF16(r.End)}
}

func (t *TextInput) rangeFromUTF16(rangeUTF16 gpui.Range) gpui.Range {
	return gpui.Range{Start: t.offsetFromUTF16(rangeUTF16.Start), End: t.offsetFromUTF16(rangeUTF16.End)}
}

func graphemeIndices(text string) []int {
	var seg segmenter.Segmenter
	if err := seg.InitWithString(text); err != nil {
		return nil
	}
	it := seg.GraphemeIterator()
	var out []int
	for it.Next() {
		out = append(out, it.Grapheme().OffsetInBytes)
	}
	return out
}

func (t *TextInput) previousBoundary(offset int) int {
	indices := graphemeIndices(string(t.content))
	for _, idx := range slices.Backward(indices) {
		if idx < offset {
			return idx
		}
	}
	return 0
}

func (t *TextInput) nextBoundary(offset int) int {
	for _, idx := range graphemeIndices(string(t.content)) {
		if idx > offset {
			return idx
		}
	}
	return len(t.content)
}

func (t *TextInput) reset(window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.content = ""
	t.selectedRange = gpui.Range{}
	t.selectionReversed = false
	t.markedRange = gpui.None[gpui.Range]()
	t.lastLayout = gpui.None[gpui.ShapedLine]()
	t.lastBounds = gpui.None[gpui.Bounds[gpui.Pixels]]()
	t.isSelecting = false
	window.NotifyTextInputChanged(t.focusHandle)
	cx.Notify()
}

// TextForRange returns the text in the UTF-16 range.
func (t *TextInput) TextForRange(rangeUTF16 gpui.Range, actualRange *gpui.Option[gpui.Range], _ *gpui.Window, _ *gpui.Context[TextInput]) (string, bool) {
	r := t.rangeFromUTF16(rangeUTF16)
	*actualRange = gpui.Some(t.rangeToUTF16(r))
	return string(t.content[r.Start:r.End]), true
}

// SelectedTextRange returns the selection in UTF-16 offsets.
func (t *TextInput) SelectedTextRange(_ bool, _ *gpui.Window, _ *gpui.Context[TextInput]) (gpui.UTF16Selection, bool) {
	return gpui.UTF16Selection{
		Range:    t.rangeToUTF16(t.selectedRange),
		Reversed: t.selectionReversed,
	}, true
}

// SetSelectedTextRange sets the selection from UTF-16 offsets.
func (t *TextInput) SetSelectedTextRange(rangeUTF16 gpui.Range, window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.selectedRange = t.rangeFromUTF16(rangeUTF16)
	t.selectionReversed = false
	window.NotifyTextSelectionChanged(t.focusHandle)
	cx.Notify()
}

// TextLengthUTF16 returns the content length in UTF-16 code units.
func (t *TextInput) TextLengthUTF16(_ *gpui.Window, _ *gpui.Context[TextInput]) (int, bool) {
	return len(utf16.Encode([]rune(string(t.content)))), true
}

// MarkedTextRange returns the marked range in UTF-16 offsets.
func (t *TextInput) MarkedTextRange(_ *gpui.Window, _ *gpui.Context[TextInput]) (gpui.Range, bool) {
	r, ok := t.markedRange.Get()
	if !ok {
		return gpui.Range{}, false
	}
	return t.rangeToUTF16(r), true
}

// UnmarkText clears the marked range.
func (t *TextInput) UnmarkText(window *gpui.Window, cx *gpui.Context[TextInput]) {
	t.markedRange = gpui.None[gpui.Range]()
	window.NotifyTextInputChanged(t.focusHandle)
	cx.Notify()
}

func (t *TextInput) resolveRange(rangeUTF16 gpui.Option[gpui.Range]) gpui.Range {
	if r, ok := rangeUTF16.Get(); ok {
		return t.rangeFromUTF16(r)
	}
	if r, ok := t.markedRange.Get(); ok {
		return r
	}
	return t.selectedRange
}

// ReplaceTextInRange replaces the text in the range with newText.
func (t *TextInput) ReplaceTextInRange(rangeUTF16 gpui.Option[gpui.Range], newText string, window *gpui.Window, cx *gpui.Context[TextInput]) {
	r := t.resolveRange(rangeUTF16)

	t.content = gpui.SharedString(string(t.content[:r.Start]) + newText + string(t.content[r.End:]))
	t.selectedRange = gpui.Range{Start: r.Start + len(newText), End: r.Start + len(newText)}
	t.selectionReversed = false
	t.markedRange = gpui.None[gpui.Range]()
	window.NotifyTextInputChanged(t.focusHandle)
	cx.Notify()
}

// ReplaceAndMarkTextInRange replaces the text in the range and marks it.
func (t *TextInput) ReplaceAndMarkTextInRange(rangeUTF16 gpui.Option[gpui.Range], newText string, newSelectedRangeUTF16 gpui.Option[gpui.Range], window *gpui.Window, cx *gpui.Context[TextInput]) {
	r := t.resolveRange(rangeUTF16)

	t.content = gpui.SharedString(string(t.content[:r.Start]) + newText + string(t.content[r.End:]))
	if newText != "" {
		t.markedRange = gpui.Some(gpui.Range{Start: r.Start, End: r.Start + len(newText)})
	} else {
		t.markedRange = gpui.None[gpui.Range]()
	}
	insertionStartUTF16 := t.offsetToUTF16(r.Start)
	insertedLengthUTF16 := len(utf16.Encode([]rune(newText)))
	if selected, ok := newSelectedRangeUTF16.Get(); ok {
		t.selectedRange = t.rangeFromUTF16(gpui.Range{
			Start: insertionStartUTF16 + min(selected.Start, insertedLengthUTF16),
			End:   insertionStartUTF16 + min(selected.End, insertedLengthUTF16),
		})
	} else {
		t.selectedRange = gpui.Range{Start: r.Start + len(newText), End: r.Start + len(newText)}
	}
	t.selectionReversed = false

	window.NotifyTextInputChanged(t.focusHandle)
	cx.Notify()
}

// BoundsForRange returns the bounds of the UTF-16 range.
func (t *TextInput) BoundsForRange(rangeUTF16 gpui.Range, bounds gpui.Bounds[gpui.Pixels], _ *gpui.Window, _ *gpui.Context[TextInput]) (gpui.Bounds[gpui.Pixels], bool) {
	lastLayout, ok := t.lastLayout.Get()
	if !ok {
		return gpui.Bounds[gpui.Pixels]{}, false
	}
	if lastLayout.Text != t.content {
		return gpui.Bounds[gpui.Pixels]{}, false
	}
	r := t.rangeFromUTF16(rangeUTF16)
	return gpui.BoundsFromCorners(
		gpui.NewPoint(bounds.Left()+lastLayout.XForIndex(r.Start), bounds.Top()),
		gpui.NewPoint(bounds.Left()+lastLayout.XForIndex(r.End), bounds.Bottom()),
	), true
}

// SelectionBoundsForRange returns the selection rectangles of the UTF-16 range.
func (t *TextInput) SelectionBoundsForRange(rangeUTF16 gpui.Range, bounds gpui.Bounds[gpui.Pixels], window *gpui.Window, cx *gpui.Context[TextInput]) []gpui.Bounds[gpui.Pixels] {
	if b, ok := t.BoundsForRange(rangeUTF16, bounds, window, cx); ok {
		return []gpui.Bounds[gpui.Pixels]{b}
	}
	return nil
}

// CharacterIndexForPoint returns the UTF-16 index closest to point.
func (t *TextInput) CharacterIndexForPoint(point gpui.Point[gpui.Pixels], _ *gpui.Window, _ *gpui.Context[TextInput]) (int, bool) {
	lastBounds, ok := t.lastBounds.Get()
	if !ok {
		return 0, false
	}
	linePoint, ok := lastBounds.Localize(point)
	if !ok {
		return 0, false
	}
	lastLayout, ok := t.lastLayout.Get()
	if !ok {
		return 0, false
	}

	if lastLayout.Text != t.content {
		return 0, false
	}
	utf8Index := lastLayout.ClosestIndexForX(linePoint.X)
	return t.offsetToUTF16(utf8Index), true
}

// TextElement paints a TextInput's line, selection and cursor.
type TextElement struct {
	input gpui.Entity[TextInput]
}

// PrepaintState is TextElement's prepaint state.
type PrepaintState struct {
	line      gpui.Option[gpui.ShapedLine]
	cursor    gpui.Option[gpui.PaintQuad]
	selection gpui.Option[gpui.PaintQuad]
}

// ElementID reports that the element has no id.
func (*TextElement) ElementID() (gpui.ElementID, bool) {
	return gpui.ElementID{}, false
}

// SourceLocation returns nil.
func (*TextElement) SourceLocation() *gpui.SourceLocation {
	return nil
}

// RequestLayout requests a full-width, one-line layout.
func (e *TextElement) RequestLayout(_ *gpui.GlobalElementID, _ *gpui.InspectorElementID, window *gpui.Window, cx *gpui.App) (gpui.LayoutID, struct{}) {
	style := gpui.DefaultStyle()
	style.Size.Width = gpui.Relative(1.).ToLength()
	style.Size.Height = window.LineHeight().ToLength()
	return window.RequestLayout(style, nil, cx), struct{}{}
}

// Prepaint shapes the line and computes the cursor and selection quads.
func (e *TextElement) Prepaint(_ *gpui.GlobalElementID, _ *gpui.InspectorElementID, bounds gpui.Bounds[gpui.Pixels], _ *struct{}, window *gpui.Window, cx *gpui.App) PrepaintState {
	input := e.input.Read(cx)
	content := input.content
	selectedRange := input.selectedRange
	cursor := input.cursorOffset()
	style := window.TextStyle()

	displayText, textColor := content, style.Color
	if len(content) == 0 {
		displayText, textColor = input.placeholder, gpui.NewHsla(0., 0., 0., 0.2)
	}

	run := gpui.TextRun{
		Len:             len(displayText),
		Font:            style.Font(),
		Color:           textColor,
		BackgroundColor: gpui.None[gpui.Hsla](),
		Underline:       gpui.None[gpui.UnderlineStyle](),
		Strikethrough:   gpui.None[gpui.StrikethroughStyle](),
	}
	var runs []gpui.TextRun
	if markedRange, ok := input.markedRange.Get(); ok {
		before := run
		before.Len = markedRange.Start
		marked := run
		marked.Len = markedRange.End - markedRange.Start
		marked.Underline = gpui.Some(gpui.UnderlineStyle{
			Color:     gpui.Some(run.Color),
			Thickness: gpui.Px(1.0),
			Wavy:      false,
		})
		after := run
		after.Len = len(displayText) - markedRange.End
		for _, r := range []gpui.TextRun{before, marked, after} {
			if r.Len > 0 {
				runs = append(runs, r)
			}
		}
	} else {
		runs = []gpui.TextRun{run}
	}

	fontSize := style.FontSize.ToPixels(window.RemSize())
	line := window.TextSystem().ShapeLine(displayText, fontSize, runs, gpui.None[gpui.Pixels]())

	cursorPos := line.XForIndex(cursor)
	var selection, cursorQuad gpui.Option[gpui.PaintQuad]
	if rangeIsEmpty(selectedRange) {
		cursorQuad = gpui.Some(gpui.NewFill(
			gpui.NewBounds(
				gpui.NewPoint(bounds.Left()+cursorPos, bounds.Top()),
				gpui.NewSize(gpui.Px(2.), bounds.Bottom()-bounds.Top()),
			),
			gpui.Blue().ToBackground(),
		))
	} else {
		selection = gpui.Some(gpui.NewFill(
			gpui.BoundsFromCorners(
				gpui.NewPoint(bounds.Left()+line.XForIndex(selectedRange.Start), bounds.Top()),
				gpui.NewPoint(bounds.Left()+line.XForIndex(selectedRange.End), bounds.Bottom()),
			),
			gpui.NewRgba(0x3311ff30).ToBackground(),
		))
	}
	return PrepaintState{
		line:      gpui.Some(line),
		cursor:    cursorQuad,
		selection: selection,
	}
}

// Paint paints the line, selection and cursor and registers the input handler.
func (e *TextElement) Paint(_ *gpui.GlobalElementID, _ *gpui.InspectorElementID, bounds gpui.Bounds[gpui.Pixels], _ *struct{}, prepaint *PrepaintState, window *gpui.Window, cx *gpui.App) {
	defer e.input.Release()
	focusHandle := e.input.Read(cx).focusHandle.Clone()
	defer focusHandle.Release()
	inputHandler := gpui.NewElementInputHandler(bounds, e.input.Clone())
	window.HandleInput(focusHandle, inputHandler, cx)
	if selection, ok := prepaint.selection.Get(); ok {
		prepaint.selection = gpui.None[gpui.PaintQuad]()
		window.PaintQuad(selection)
	}
	line, ok := prepaint.line.Get()
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	prepaint.line = gpui.None[gpui.ShapedLine]()
	if err := line.Paint(bounds.Origin, window.LineHeight(), gpui.TextAlignLeft, gpui.None[gpui.Pixels](), window, cx); err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}

	if focusHandle.IsFocused(window) {
		if cursor, ok := prepaint.cursor.Get(); ok {
			prepaint.cursor = gpui.None[gpui.PaintQuad]()
			window.PaintQuad(cursor)
		}
	}

	e.input.Update(cx, func(input *TextInput, _ *gpui.Context[TextInput]) {
		input.lastLayout = gpui.Some(line)
		input.lastBounds = gpui.Some(bounds)
	})
}

// IntoAnyElement converts the element into an AnyElement.
func (e *TextElement) IntoAnyElement() gpui.AnyElement {
	return gpui.NewAnyElement[struct{}, PrepaintState](e)
}

func actionListener[A any](listener func(event *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}

// Render renders the input.
func (t *TextInput) Render(_ *gpui.Window, cx *gpui.Context[TextInput]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		KeyContext("TextInput").
		TrackFocus(t.focusHandle).
		Cursor(gpui.CursorStyleIBeam).
		OnAction(actionListener(cx.Listener((*TextInput).backspace))).
		OnAction(actionListener(cx.Listener((*TextInput).delete))).
		OnAction(actionListener(cx.Listener((*TextInput).left))).
		OnAction(actionListener(cx.Listener((*TextInput).right))).
		OnAction(actionListener(cx.Listener((*TextInput).selectLeft))).
		OnAction(actionListener(cx.Listener((*TextInput).selectRight))).
		OnAction(actionListener(cx.Listener((*TextInput).selectAll))).
		OnAction(actionListener(cx.Listener((*TextInput).home))).
		OnAction(actionListener(cx.Listener((*TextInput).end))).
		OnAction(actionListener(cx.Listener((*TextInput).showCharacterPalette))).
		OnAction(actionListener(cx.Listener((*TextInput).paste))).
		OnAction(actionListener(cx.Listener((*TextInput).cut))).
		OnAction(actionListener(cx.Listener((*TextInput).copy))).
		OnMouseDown(gpui.MouseButtonLeft, cx.Listener((*TextInput).onMouseDown)).
		OnMouseUp(gpui.MouseButtonLeft, cx.Listener((*TextInput).onMouseUp)).
		OnMouseUpOut(gpui.MouseButtonLeft, cx.Listener((*TextInput).onMouseUp)).
		OnMouseMove(cx.Listener((*TextInput).onMouseMove)).
		Bg(gpui.Rgb(0xeeeeee)).
		LineHeight(gpui.Px(30.)).
		TextSize(gpui.Px(24.)).
		Child(
			gpui.NewDiv().
				H(gpui.Px(30. + 4.*2.)).
				WFull().
				P(gpui.Px(4.)).
				Bg(gpui.White()).
				Child(&TextElement{input: cx.Entity()}),
		)
}

// FocusHandle returns the input's focus handle.
func (t *TextInput) FocusHandle(_ *gpui.App) gpui.FocusHandle {
	return t.focusHandle.Clone()
}

// InputExample is the input example view.
type InputExample struct {
	textInput        gpui.Entity[TextInput]
	recentKeystrokes []gpui.Keystroke
	focusHandle      gpui.FocusHandle
}

// FocusHandle returns the example's focus handle.
func (e *InputExample) FocusHandle(_ *gpui.App) gpui.FocusHandle {
	return e.focusHandle.Clone()
}

// Drop releases the example's handles.
func (e *InputExample) Drop(_ *gpui.App) {
	e.textInput.Release()
	e.focusHandle.Release()
}

func (e *InputExample) onResetClick(_ *gpui.MouseUpEvent, window *gpui.Window, cx *gpui.Context[InputExample]) {
	e.recentKeystrokes = e.recentKeystrokes[:0]
	e.textInput.Update(cx, func(textInput *TextInput, cx *gpui.Context[TextInput]) { textInput.reset(window, cx) })
	cx.Notify()
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

// Render renders the view.
func (e *InputExample) Render(_ *gpui.Window, cx *gpui.Context[InputExample]) gpui.IntoElement {
	keystrokes := make([]gpui.IntoElement, 0, len(e.recentKeystrokes))
	for _, ks := range slices.Backward(e.recentKeystrokes) {
		keyChar := ""
		if c, ok := ks.KeyChar.Get(); ok {
			keyChar = fmt.Sprintf("-> %q", c)
		}
		keystrokes = append(keystrokes, gpui.SharedString(fmt.Sprintf("%s %s", ks.Unparse(cx.PlatformInfo().Os), keyChar)))
	}
	return gpui.NewDiv().
		Bg(gpui.Rgb(0xaaaaaa)).
		TrackFocus(e.focusHandle).
		Flex().
		FlexCol().
		SizeFull().
		Child(
			gpui.NewDiv().
				Bg(gpui.White()).
				BorderB1().
				BorderColor(gpui.Black()).
				Flex().
				FlexRow().
				JustifyBetween().
				ChildText(fmt.Sprintf("Keyboard %s", cx.KeyboardLayout().Name())).
				Child(
					gpui.NewDiv().
						Border1().
						BorderColor(gpui.Black()).
						Px2().
						Bg(gpui.Yellow()).
						ChildText("Reset").
						Hover(func(style *gpui.StyleRefinement) *gpui.StyleRefinement {
							return styled(styled(style).Bg(gpui.Yellow().Blend(gpui.OpaqueGrey(0.5, 0.5)))).CursorPointer()
						}).
						OnMouseUp(gpui.MouseButtonLeft, cx.Listener((*InputExample).onResetClick)),
				),
		).
		Child(e.textInput.Clone()).
		Children(keystrokes...)
}

func newInput(content gpui.SharedString, placeholder gpui.SharedString, cx *gpui.Context[TextInput]) *TextInput {
	return newTextInput(&TextInput{
		focusHandle:       cx.FocusHandle(),
		content:           content,
		placeholder:       placeholder,
		selectedRange:     gpui.Range{},
		selectionReversed: false,
		markedRange:       gpui.None[gpui.Range](),
		lastLayout:        gpui.None[gpui.ShapedLine](),
		lastBounds:        gpui.None[gpui.Bounds[gpui.Pixels]](),
		isSelecting:       false,
	})
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(300.0), gpui.Px(300.0)), cx)
		none := gpui.None[string]()
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("backspace", Backspace{}, none),
			gpui.NewKeyBinding("delete", Delete{}, none),
			gpui.NewKeyBinding("left", Left{}, none),
			gpui.NewKeyBinding("right", Right{}, none),
			gpui.NewKeyBinding("shift-left", SelectLeft{}, none),
			gpui.NewKeyBinding("shift-right", SelectRight{}, none),
			gpui.NewKeyBinding("cmd-a", SelectAll{}, none),
			gpui.NewKeyBinding("cmd-v", Paste{}, none),
			gpui.NewKeyBinding("cmd-c", Copy{}, none),
			gpui.NewKeyBinding("cmd-x", Cut{}, none),
			gpui.NewKeyBinding("home", Home{}, none),
			gpui.NewKeyBinding("end", End{}, none),
			gpui.NewKeyBinding("ctrl-cmd-space", ShowCharacterPalette{}, none),
		})

		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		window, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[InputExample] {
			textInput := cx.New(func(cx *gpui.Context[TextInput]) *TextInput {
				return newInput("", "Type here...", cx)
			})
			return cx.New(func(cx *gpui.Context[InputExample]) *InputExample {
				return &InputExample{
					textInput:        textInput,
					recentKeystrokes: nil,
					focusHandle:      cx.FocusHandle(),
				}
			})
		})
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		view, err := window.UpdateR(cx, func(_ *InputExample, _ *gpui.Window, cx *gpui.Context[InputExample]) gpui.Entity[InputExample] {
			return cx.Entity()
		})
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		cx.ObserveKeystrokes(func(ev *gpui.KeystrokeEvent, _ *gpui.Window, cx *gpui.App) {
			view.Update(cx, func(view *InputExample, cx *gpui.Context[InputExample]) {
				view.recentKeystrokes = append(view.recentKeystrokes, ev.Keystroke)
				cx.Notify()
			})
		}).Detach()
		cx.OnKeyboardLayoutChange(func(cx *gpui.App) {
			_ = window.Update(cx, func(_ *InputExample, _ *gpui.Window, cx *gpui.Context[InputExample]) { cx.Notify() })
		}).Detach()

		if err := window.Update(cx, func(view *InputExample, window *gpui.Window, cx *gpui.Context[InputExample]) {
			focusHandle := view.textInput.FocusHandle(cx.App)
			window.Focus(focusHandle, cx.App)
			focusHandle.Release()
			cx.Activate(true)
		}); err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		cx.OnAction(func(_ Quit, cx *gpui.App) { cx.Quit() })
		cx.BindKeys([]gpui.KeyBinding{gpui.NewKeyBinding("cmd-q", Quit{}, none)})
	})
}

func main() {
	runExample()
}
