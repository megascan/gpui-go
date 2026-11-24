package main

import (
	"github.com/megascan/gpui-go"
)

type source struct {
	value    gpui.Entity[string]
	editor   gpui.Entity[Editor]
	isEditor bool
}

func (s source) entityID() gpui.EntityID {
	if s.isEditor {
		return s.editor.EntityID()
	}
	return s.value.EntityID()
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func focusShadow(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
	return this.Shadow([]gpui.BoxShadow{{
		Color:   gpui.NewHsla(220./360., 0.8, 0.5, 0.3),
		Offset:  gpui.NewPoint(gpui.Px(0.), gpui.Px(0.)),
		Blur:    gpui.Px(4.),
		Spread:  gpui.Px(1.),
		IsInset: false,
	}})
}

// Input is a single-line text input, the shaping layer over Editor. Construct it from a string
// (NewInput: the editor is allocated internally) or from an editor you own (InputEditor: you
// can read and drive its cursor).
type Input struct {
	source source
	width  gpui.Option[gpui.Pixels]
	color  gpui.Option[gpui.Hsla]
}

// NewInput creates an input backed by a bare string; the editor is allocated internally.
func NewInput(value gpui.Entity[string]) *Input {
	return &Input{
		source: source{value: value},
		width:  gpui.None[gpui.Pixels](),
		color:  gpui.None[gpui.Hsla](),
	}
}

// InputEditor creates an input backed by an editor you own.
func InputEditor(editor gpui.Entity[Editor]) *Input {
	return &Input{
		source: source{editor: editor, isEditor: true},
		width:  gpui.None[gpui.Pixels](),
		color:  gpui.None[gpui.Hsla](),
	}
}

// Width sets the input's width.
func (i *Input) Width(width gpui.Pixels) *Input {
	i.width = gpui.Some(width)
	return i
}

// Color sets the input's text color.
func (i *Input) Color(color gpui.Hsla) *Input {
	i.color = gpui.Some(color)
	return i
}

// EntityID returns the identity of the backing string or editor.
func (i *Input) EntityID() (gpui.EntityID, bool) {
	return i.source.entityID(), true
}

// Render renders the input.
func (i *Input) Render(window *gpui.Window, cx *gpui.App) gpui.IntoElement {
	editor := i.source.editor
	if !i.source.isEditor {
		value := i.source.value
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
	textColor := i.color.UnwrapOr(gpui.NewHsla(0., 0., 0.1, 1.))
	boxWidth := i.width.UnwrapOr(gpui.Px(300.))

	border := gpui.NewHsla(0., 0., 0.75, 1.)
	if isFocused {
		border = gpui.NewHsla(220./360., 0.8, 0.5, 1.)
	}

	var cacheStyle gpui.StyleRefinement
	return gpui.NewDiv().
		ID("input").
		KeyContext("TextInput").
		TrackFocus(focusHandle).
		Cursor(gpui.CursorStyleIBeam).
		Map(standardActions(editor.Downgrade())).
		W(boxWidth).
		H(gpui.Px(36.)).
		Px(gpui.Px(8.)).
		Bg(gpui.White()).
		Border1().
		BorderColor(border).
		When(isFocused, focusShadow).
		Rounded(gpui.Px(4.)).
		OverflowHidden().
		Flex().
		ItemsCenter().
		LineHeight(gpui.Px(20.)).
		TextSize(gpui.Px(14.)).
		TextColor(textColor).
		Child(editor.Cached(*styled(&cacheStyle).SizeFull()))
}

// IntoAnyElement converts the input into a view element.
func (i *Input) IntoAnyElement() gpui.AnyElement {
	return gpui.NewViewElement(i).IntoAnyElement()
}
