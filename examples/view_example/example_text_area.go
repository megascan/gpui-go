package main

import (
	"github.com/megascan/gpui-go"
)

// TextArea is a multi-line text box: the same Editor, taller chrome, and Enter inserts a
// newline. Construct it from a string or an editor, exactly like Input.
type TextArea struct {
	source source
	rows   int
	color  gpui.Option[gpui.Hsla]
}

// NewTextArea creates a text area backed by a bare string.
func NewTextArea(value gpui.Entity[string], rows int) *TextArea {
	return &TextArea{
		source: source{value: value},
		rows:   rows,
		color:  gpui.None[gpui.Hsla](),
	}
}

// TextAreaEditor creates a text area backed by an editor you own.
func TextAreaEditor(editor gpui.Entity[Editor], rows int) *TextArea {
	return &TextArea{
		source: source{editor: editor, isEditor: true},
		rows:   rows,
		color:  gpui.None[gpui.Hsla](),
	}
}

// Color sets the text area's text color.
func (a *TextArea) Color(color gpui.Hsla) *TextArea {
	a.color = gpui.Some(color)
	return a
}

// EntityID returns the identity of the backing string or editor.
func (a *TextArea) EntityID() (gpui.EntityID, bool) {
	return a.source.entityID(), true
}

// Render renders the text area.
func (a *TextArea) Render(window *gpui.Window, cx *gpui.App) gpui.IntoElement {
	editor := a.source.editor
	if !a.source.isEditor {
		value := a.source.value
		moved := false
		editor = window.UseState(cx, func(window *gpui.Window, cx *gpui.Context[Editor]) *Editor {
			moved = true
			return EditorOver(value, window, cx)
		})
		if !moved {
			value.Release()
		}
	}

	focusHandle := editor.Read(cx).focusHandle
	isFocused := focusHandle.IsFocused(window)
	textColor := a.color.UnwrapOr(gpui.NewHsla(0., 0., 0.1, 1.))
	rowHeight := gpui.Px(20.)
	boxHeight := rowHeight*gpui.Pixels(a.rows) + gpui.Px(16.)

	border := gpui.NewHsla(0., 0., 0.75, 1.)
	if isFocused {
		border = gpui.NewHsla(220./360., 0.8, 0.5, 1.)
	}

	weakEditor := editor.Downgrade()
	var cacheStyle gpui.StyleRefinement
	return gpui.NewDiv().
		ID("text-area").
		KeyContext("TextInput").
		TrackFocus(focusHandle).
		Cursor(gpui.CursorStyleIBeam).
		Map(standardActions(weakEditor)).
		OnAction(func(_ Enter, _ *gpui.Window, cx *gpui.App) {
			_ = weakEditor.Update(cx, func(e *Editor, cx *gpui.Context[Editor]) { e.insertNewline(cx) })
		}).
		W(gpui.Px(400.)).
		H(boxHeight).
		P(gpui.Px(8.)).
		Bg(gpui.White()).
		Border1().
		BorderColor(border).
		When(isFocused, focusShadow).
		Rounded(gpui.Px(4.)).
		OverflowHidden().
		LineHeight(rowHeight).
		TextSize(gpui.Px(14.)).
		TextColor(textColor).
		Child(editor.Cached(*styled(styled(&cacheStyle).WFull()).H(rowHeight * gpui.Pixels(a.rows))))
}

// IntoAnyElement converts the text area into a view element.
func (a *TextArea) IntoAnyElement() gpui.AnyElement {
	return gpui.NewViewElement(a).IntoAnyElement()
}
