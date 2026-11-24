package main

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-text/typesetting/segmenter"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

// Editor is the workhorse entity. It owns the cursor, blink, focus, keyboard handling, and the
// specialized text-shaping renderer. The text itself lives in a shared Entity[string] it is
// handed at construction, so the value is readable and writable from outside while the editing
// machinery stays in here.
type Editor struct {
	gpui.EntityInputHandlerDefaults[Editor]
	value         gpui.Entity[string]
	focusHandle   gpui.FocusHandle
	cursor        int
	cursorVisible bool
	blinkTask     *gpui.Task[struct{}]
	subscriptions []gpui.Subscription
}

// NewEditor creates an editor that owns its own string internally, seeded with text.
func NewEditor(text string, window *gpui.Window, cx *gpui.Context[Editor]) *Editor {
	value := cx.New(func(*gpui.Context[string]) *string { return &text })
	return EditorOver(value, window, cx)
}

// EditorOver creates an editor over a string the caller owns, so the value is shared in and
// out. The editor takes ownership of value.
func EditorOver(value gpui.Entity[string], window *gpui.Window, cx *gpui.Context[Editor]) *Editor {
	focusHandle := cx.FocusHandle()

	focusSub := cx.OnFocus(focusHandle, window, func(this *Editor, _ *gpui.Window, cx *gpui.Context[Editor]) {
		this.startBlink(cx)
	})
	blurSub := cx.OnBlur(focusHandle, window, func(this *Editor, _ *gpui.Window, cx *gpui.Context[Editor]) {
		this.stopBlink(cx)
	})

	valueSub := cx.Observe(value, func(this *Editor, value gpui.Entity[string], cx *gpui.Context[Editor]) {
		content := *value.Read(cx.App)
		cursor := min(this.cursor, len(content))
		for cursor > 0 && !isCharBoundary(content, cursor) {
			cursor--
		}
		this.cursor = cursor
		cx.Notify()
	})

	e := &Editor{
		value:         value,
		focusHandle:   focusHandle,
		cursor:        0,
		cursorVisible: false,
		blinkTask:     scheduler.ReadyTask(struct{}{}),
		subscriptions: []gpui.Subscription{focusSub, blurSub, valueSub},
	}
	gpui.BindEntityInputHandlerDefaults(&e.EntityInputHandlerDefaults, e)
	return e
}

// Drop releases the editor's handles, task and subscriptions.
func (e *Editor) Drop(_ *gpui.App) {
	e.blinkTask.Cancel()
	for _, s := range e.subscriptions {
		s.Unsubscribe()
	}
	e.focusHandle.Release()
	e.value.Release()
}

func isCharBoundary(s string, index int) bool {
	if index == 0 || index == len(s) {
		return true
	}
	if index > len(s) {
		return false
	}
	return utf8.RuneStart(s[index])
}

// Text returns the current text. Read this from anywhere to get the value out.
func (e *Editor) Text(cx *gpui.App) string {
	return *e.value.Read(cx)
}

func (e *Editor) setBlinkTask(task *gpui.Task[struct{}]) {
	e.blinkTask.Cancel()
	e.blinkTask = task
}

func (e *Editor) startBlink(cx *gpui.Context[Editor]) {
	e.cursorVisible = true
	e.setBlinkTask(spawnBlinkTask(cx))
}

func (e *Editor) stopBlink(cx *gpui.Context[Editor]) {
	e.cursorVisible = false
	e.setBlinkTask(scheduler.ReadyTask(struct{}{}))
	cx.Notify()
}

func spawnBlinkTask(cx *gpui.Context[Editor]) *gpui.Task[struct{}] {
	return cx.Spawn(func(this gpui.WeakEntity[Editor], cx *gpui.AsyncApp) (struct{}, error) {
		for {
			if _, err := cx.BackgroundExecutor().Timer(500 * time.Millisecond).Await(cx); err != nil {
				return struct{}{}, fmt.Errorf("blink timer: %w", err)
			}
			result := this.Update(cx, func(editor *Editor, cx *gpui.Context[Editor]) {
				editor.cursorVisible = !editor.cursorVisible
				cx.Notify()
			})
			if result != nil {
				break
			}
		}
		return struct{}{}, nil
	})
}

func (e *Editor) resetBlink(cx *gpui.Context[Editor]) {
	e.cursorVisible = true
	e.setBlinkTask(spawnBlinkTask(cx))
}

func (e *Editor) left(_ *Left, _ *gpui.Window, cx *gpui.Context[Editor]) {
	content := e.Text(cx.App)
	if e.cursor > 0 {
		e.cursor = previousBoundary(content, e.cursor)
	}
	e.resetBlink(cx)
	cx.Notify()
}

func (e *Editor) right(_ *Right, _ *gpui.Window, cx *gpui.Context[Editor]) {
	content := e.Text(cx.App)
	if e.cursor < len(content) {
		e.cursor = nextBoundary(content, e.cursor)
	}
	e.resetBlink(cx)
	cx.Notify()
}

func (e *Editor) home(_ *Home, _ *gpui.Window, cx *gpui.Context[Editor]) {
	e.cursor = 0
	e.resetBlink(cx)
	cx.Notify()
}

func (e *Editor) end(_ *End, _ *gpui.Window, cx *gpui.Context[Editor]) {
	e.cursor = len(e.Text(cx.App))
	e.resetBlink(cx)
	cx.Notify()
}

func (e *Editor) backspace(_ *Backspace, _ *gpui.Window, cx *gpui.Context[Editor]) {
	content := e.Text(cx.App)
	if e.cursor > 0 {
		prev := previousBoundary(content, e.cursor)
		cursor := e.cursor
		e.value.Update(cx, func(s *string, cx *gpui.Context[string]) {
			*s = (*s)[:prev] + (*s)[cursor:]
			cx.Notify()
		})
		e.cursor = prev
	}
	e.resetBlink(cx)
	cx.Notify()
}

func (e *Editor) delete(_ *Delete, _ *gpui.Window, cx *gpui.Context[Editor]) {
	content := e.Text(cx.App)
	if e.cursor < len(content) {
		next := nextBoundary(content, e.cursor)
		cursor := e.cursor
		e.value.Update(cx, func(s *string, cx *gpui.Context[string]) {
			*s = (*s)[:cursor] + (*s)[next:]
			cx.Notify()
		})
	}
	e.resetBlink(cx)
	cx.Notify()
}

func (e *Editor) insertNewline(cx *gpui.Context[Editor]) {
	cursor := e.cursor
	e.value.Update(cx, func(s *string, cx *gpui.Context[string]) {
		*s = (*s)[:cursor] + "\n" + (*s)[cursor:]
		cx.Notify()
	})
	e.cursor++
	e.resetBlink(cx)
	cx.Notify()
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

func previousBoundary(content string, offset int) int {
	for _, idx := range slices.Backward(graphemeIndices(content)) {
		if idx < offset {
			return idx
		}
	}
	return 0
}

func nextBoundary(content string, offset int) int {
	for _, idx := range graphemeIndices(content) {
		if idx > offset {
			return idx
		}
	}
	return len(content)
}

func lenUTF16(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

func offsetFromUTF16(content string, offset int) int {
	utf8Offset := 0
	utf16Count := 0
	for _, ch := range content {
		if utf16Count >= offset {
			break
		}
		utf16Count += lenUTF16(ch)
		utf8Offset += utf8.RuneLen(ch)
	}
	return utf8Offset
}

func offsetToUTF16(content string, offset int) int {
	utf16Offset := 0
	utf8Count := 0
	for _, ch := range content {
		if utf8Count >= offset {
			break
		}
		utf8Count += utf8.RuneLen(ch)
		utf16Offset += lenUTF16(ch)
	}
	return utf16Offset
}

func rangeToUTF16(content string, r gpui.Range) gpui.Range {
	return gpui.Range{Start: offsetToUTF16(content, r.Start), End: offsetToUTF16(content, r.End)}
}

func rangeFromUTF16(content string, rangeUTF16 gpui.Range) gpui.Range {
	return gpui.Range{Start: offsetFromUTF16(content, rangeUTF16.Start), End: offsetFromUTF16(content, rangeUTF16.End)}
}

// FocusHandle returns the editor's focus handle.
func (e *Editor) FocusHandle(_ *gpui.App) gpui.FocusHandle {
	return e.focusHandle.Clone()
}

// TextForRange returns the text in the UTF-16 range.
func (e *Editor) TextForRange(rangeUTF16 gpui.Range, actualRange *gpui.Option[gpui.Range], _ *gpui.Window, cx *gpui.Context[Editor]) (string, bool) {
	content := e.Text(cx.App)
	r := rangeFromUTF16(content, rangeUTF16)
	*actualRange = gpui.Some(rangeToUTF16(content, r))
	return content[r.Start:r.End], true
}

// SelectedTextRange returns the cursor as an empty UTF-16 selection.
func (e *Editor) SelectedTextRange(_ bool, _ *gpui.Window, cx *gpui.Context[Editor]) (gpui.UTF16Selection, bool) {
	content := e.Text(cx.App)
	utf16Cursor := offsetToUTF16(content, e.cursor)
	return gpui.UTF16Selection{
		Range:    gpui.Range{Start: utf16Cursor, End: utf16Cursor},
		Reversed: false,
	}, true
}

// MarkedTextRange reports that there is no marked text.
func (e *Editor) MarkedTextRange(_ *gpui.Window, _ *gpui.Context[Editor]) (gpui.Range, bool) {
	return gpui.Range{}, false
}

// UnmarkText does nothing.
func (e *Editor) UnmarkText(_ *gpui.Window, _ *gpui.Context[Editor]) {}

// ReplaceTextInRange replaces the text in the UTF-16 range, or inserts at the cursor.
func (e *Editor) ReplaceTextInRange(rangeUTF16 gpui.Option[gpui.Range], newText string, _ *gpui.Window, cx *gpui.Context[Editor]) {
	content := e.Text(cx.App)
	r := gpui.Range{Start: e.cursor, End: e.cursor}
	if ru, ok := rangeUTF16.Get(); ok {
		r = rangeFromUTF16(content, ru)
	}

	newContent := content[:r.Start] + newText + content[r.End:]
	e.cursor = r.Start + len(newText)
	e.value.Update(cx, func(s *string, cx *gpui.Context[string]) {
		*s = newContent
		cx.Notify()
	})
	e.resetBlink(cx)
	cx.Notify()
}

// ReplaceAndMarkTextInRange replaces the text in the range; the editor does not mark text.
func (e *Editor) ReplaceAndMarkTextInRange(rangeUTF16 gpui.Option[gpui.Range], newText string, _ gpui.Option[gpui.Range], window *gpui.Window, cx *gpui.Context[Editor]) {
	e.ReplaceTextInRange(rangeUTF16, newText, window, cx)
}

// BoundsForRange reports no bounds.
func (e *Editor) BoundsForRange(_ gpui.Range, _ gpui.Bounds[gpui.Pixels], _ *gpui.Window, _ *gpui.Context[Editor]) (gpui.Bounds[gpui.Pixels], bool) {
	return gpui.Bounds[gpui.Pixels]{}, false
}

// CharacterIndexForPoint reports no index.
func (e *Editor) CharacterIndexForPoint(_ gpui.Point[gpui.Pixels], _ *gpui.Window, _ *gpui.Context[Editor]) (int, bool) {
	return 0, false
}

// Render renders the editor's text.
func (e *Editor) Render(_ *gpui.Window, cx *gpui.Context[Editor]) gpui.IntoElement {
	return &EditorText{editor: cx.Entity()}
}

// EditorText is the specialized renderer: it shapes the text and paints the cursor.
type EditorText struct {
	editor gpui.Entity[Editor]
}

// EditorTextPrepaint is EditorText's prepaint state.
type EditorTextPrepaint struct {
	lines  []gpui.ShapedLine
	cursor gpui.Option[gpui.PaintQuad]
}

// ElementID reports that the element has no id.
func (*EditorText) ElementID() (gpui.ElementID, bool) {
	return gpui.ElementID{}, false
}

// SourceLocation returns nil.
func (*EditorText) SourceLocation() *gpui.SourceLocation {
	return nil
}

// RequestLayout requests a full-width layout one line tall per text line.
func (t *EditorText) RequestLayout(_ *gpui.GlobalElementID, _ *gpui.InspectorElementID, window *gpui.Window, cx *gpui.App) (gpui.LayoutID, struct{}) {
	editor := t.editor.Read(cx)
	content := *editor.value.Read(cx)
	lineCount := max(strings.Count(content, "\n")+1, 1)
	lineHeight := window.LineHeight()
	style := gpui.DefaultStyle()
	style.Size.Width = gpui.Relative(1.).ToLength()
	style.Size.Height = (lineHeight * gpui.Pixels(lineCount)).ToLength()
	return window.RequestLayout(style, nil, cx), struct{}{}
}

// Prepaint shapes the lines and computes the cursor quad.
func (t *EditorText) Prepaint(_ *gpui.GlobalElementID, _ *gpui.InspectorElementID, bounds gpui.Bounds[gpui.Pixels], _ *struct{}, window *gpui.Window, cx *gpui.App) EditorTextPrepaint {
	editor := t.editor.Read(cx)
	content := *editor.value.Read(cx)
	cursorOffset := editor.cursor
	cursorVisible := editor.cursorVisible
	isFocused := editor.focusHandle.IsFocused(window)

	style := window.TextStyle()
	textColor := style.Color
	fontSize := style.FontSize.ToPixels(window.RemSize())
	lineHeight := window.LineHeight()

	isPlaceholder := len(content) == 0

	var lines []gpui.ShapedLine
	if isPlaceholder {
		placeholder := gpui.SharedString("Type here...")
		run := gpui.TextRun{
			Len:             len(placeholder),
			Font:            style.Font(),
			Color:           gpui.NewHsla(0., 0., 0.5, 0.5),
			BackgroundColor: gpui.None[gpui.Hsla](),
			Underline:       gpui.None[gpui.UnderlineStyle](),
			Strikethrough:   gpui.None[gpui.StrikethroughStyle](),
		}
		lines = []gpui.ShapedLine{
			window.TextSystem().ShapeLine(placeholder, fontSize, []gpui.TextRun{run}, gpui.None[gpui.Pixels]()),
		}
	} else {
		for lineStr := range strings.SplitSeq(content, "\n") {
			text := gpui.SharedString(lineStr)
			run := gpui.TextRun{
				Len:             len(text),
				Font:            style.Font(),
				Color:           textColor,
				BackgroundColor: gpui.None[gpui.Hsla](),
				Underline:       gpui.None[gpui.UnderlineStyle](),
				Strikethrough:   gpui.None[gpui.StrikethroughStyle](),
			}
			lines = append(lines, window.TextSystem().ShapeLine(text, fontSize, []gpui.TextRun{run}, gpui.None[gpui.Pixels]()))
		}
	}

	cursor := gpui.None[gpui.PaintQuad]()
	switch {
	case isFocused && cursorVisible && !isPlaceholder:
		cursorLine, offsetInLine := cursorLineAndOffset(content, cursorOffset)
		cursorLine = min(cursorLine, max(len(lines)-1, 0))
		cursorX := lines[cursorLine].XForIndex(offsetInLine)
		cursor = gpui.Some(gpui.NewFill(
			gpui.NewBounds(
				gpui.NewPoint(bounds.Left()+cursorX, bounds.Top()+lineHeight*gpui.Pixels(cursorLine)),
				gpui.NewSize(gpui.Px(1.5), lineHeight),
			),
			textColor.ToBackground(),
		))
	case isFocused && cursorVisible && isPlaceholder:
		cursor = gpui.Some(gpui.NewFill(
			gpui.NewBounds(
				gpui.NewPoint(bounds.Left(), bounds.Top()),
				gpui.NewSize(gpui.Px(1.5), lineHeight),
			),
			textColor.ToBackground(),
		))
	}

	return EditorTextPrepaint{lines: lines, cursor: cursor}
}

// Paint registers the input handler and paints the lines and cursor.
func (t *EditorText) Paint(_ *gpui.GlobalElementID, _ *gpui.InspectorElementID, bounds gpui.Bounds[gpui.Pixels], _ *struct{}, prepaint *EditorTextPrepaint, window *gpui.Window, cx *gpui.App) {
	defer t.editor.Release()
	focusHandle := t.editor.Read(cx).focusHandle.Clone()
	defer focusHandle.Release()
	inputHandler := gpui.NewElementInputHandler(bounds, t.editor.Clone())
	window.HandleInput(focusHandle, inputHandler, cx)
	if !focusHandle.IsFocused(window) {
		inputHandler.Release()
	}

	lineHeight := window.LineHeight()
	for i, line := range prepaint.lines {
		origin := gpui.NewPoint(bounds.Left(), bounds.Top()+lineHeight*gpui.Pixels(i))
		if err := line.Paint(origin, lineHeight, gpui.TextAlignLeft, gpui.None[gpui.Pixels](), window, cx); err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
	}

	if cursor, ok := prepaint.cursor.Get(); ok {
		prepaint.cursor = gpui.None[gpui.PaintQuad]()
		window.PaintQuad(cursor)
	}
}

// IntoAnyElement converts the element into an AnyElement.
func (t *EditorText) IntoAnyElement() gpui.AnyElement {
	return gpui.NewAnyElement[struct{}, EditorTextPrepaint](t)
}

func cursorLineAndOffset(content string, cursor int) (int, int) {
	lineIndex := 0
	lineStart := 0
	for i, ch := range content {
		if i >= cursor {
			break
		}
		if ch == '\n' {
			lineIndex++
			lineStart = i + 1
		}
	}
	return lineIndex, cursor - lineStart
}

func editorAction[A any](editor gpui.WeakEntity[Editor], f func(e *Editor, a *A, window *gpui.Window, cx *gpui.Context[Editor])) func(a A, window *gpui.Window, cx *gpui.App) {
	return func(a A, window *gpui.Window, cx *gpui.App) {
		_ = editor.Update(cx, func(e *Editor, cx *gpui.Context[Editor]) { f(e, &a, window, cx) })
	}
}

func standardActions(editor gpui.WeakEntity[Editor]) func(element *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
	return func(element *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
		return element.
			OnAction(editorAction(editor, (*Editor).left)).
			OnAction(editorAction(editor, (*Editor).right)).
			OnAction(editorAction(editor, (*Editor).home)).
			OnAction(editorAction(editor, (*Editor).end)).
			OnAction(editorAction(editor, (*Editor).backspace)).
			OnAction(editorAction(editor, (*Editor).delete))
	}
}
