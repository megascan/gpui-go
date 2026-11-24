// Command view_example composes a text input from the View primitives.
//
// A text input is deceptively complicated, and View makes it easy to compose one. Three pieces,
// each shown in its own section:
//
//   - Editor: the workhorse entity: cursor, blink, focus, keyboard, and a specialized text
//     renderer. All the hard parts live here.
//   - string: the data plane. editor.Text(cx) / value.Read(cx) get it out.
//   - Input / TextArea: the shaping layer. Each takes a string (and grows the editor
//     internally) or an Editor (so you can read the cursor).
//
// Run: go run ./examples/view_example
package main

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"

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

// Home moves the cursor to the start.
type Home struct{}

// End moves the cursor to the end.
type End struct{}

// Enter inserts a newline in a text area.
type Enter struct{}

// Quit quits the example.
type Quit struct{}

// Name returns the action name.
func (Backspace) Name() string { return "view_example::Backspace" }

// Name returns the action name.
func (Delete) Name() string { return "view_example::Delete" }

// Name returns the action name.
func (Left) Name() string { return "view_example::Left" }

// Name returns the action name.
func (Right) Name() string { return "view_example::Right" }

// Name returns the action name.
func (Home) Name() string { return "view_example::Home" }

// Name returns the action name.
func (End) Name() string { return "view_example::End" }

// Name returns the action name.
func (Enter) Name() string { return "view_example::Enter" }

// Name returns the action name.
func (Quit) Name() string { return "view_example::Quit" }

// BoxedClone clones the action.
func (a Backspace) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Delete) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Left) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Right) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Home) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a End) BoxedClone() gpui.Action { return a }

// BoxedClone clones the action.
func (a Enter) BoxedClone() gpui.Action { return a }

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
func (Home) PartialEq(other gpui.Action) bool { return actionEq[Home](other) }

// PartialEq reports whether other is the same action.
func (End) PartialEq(other gpui.Action) bool { return actionEq[End](other) }

// PartialEq reports whether other is the same action.
func (Enter) PartialEq(other gpui.Action) bool { return actionEq[Enter](other) }

// PartialEq reports whether other is the same action.
func (Quit) PartialEq(other gpui.Action) bool { return actionEq[Quit](other) }

func init() {
	for _, a := range []gpui.Action{Backspace{}, Delete{}, Left{}, Right{}, Home{}, End{}, Enter{}, Quit{}} {
		gpui.RegisterAction(gpui.ActionData{
			Name:  a.Name(),
			Type:  reflect.TypeOf(a),
			Build: func(jsontext.Value) (gpui.Action, error) { return a.BoxedClone(), nil },
		})
	}
}

// CursorReadout is a tiny stateless view that reads an editor's cursor and is composed beside
// the thing editing it: two views over one entity, zero wiring.
type CursorReadout struct {
	editor gpui.Entity[Editor]
}

// NewCursorReadout creates a readout of editor's cursor; it takes ownership of editor.
func NewCursorReadout(editor gpui.Entity[Editor]) CursorReadout {
	return CursorReadout{editor: editor}
}

// Render renders the cursor position.
func (c CursorReadout) Render(_ *gpui.Window, cx *gpui.App) gpui.IntoElement {
	defer c.editor.Release()
	cursor := c.editor.Read(cx).cursor
	return gpui.NewDiv().
		TextSm().
		TextColor(gpui.NewHsla(0., 0., 0.45, 1.)).
		Child(gpui.SharedString(fmt.Sprintf("cursor @ %d", cursor)))
}

// IntoAnyElement converts the readout into an element.
func (c CursorReadout) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(c)
}

// ViewExample is the view_example root view.
type ViewExample struct{}

// NewViewExample creates the view.
func NewViewExample() *ViewExample {
	return &ViewExample{}
}

func newString(s string) func(*gpui.Window, *gpui.Context[string]) *string {
	return func(*gpui.Window, *gpui.Context[string]) *string { return &s }
}

// Render renders the view.
func (v *ViewExample) Render(window *gpui.Window, cx *gpui.Context[ViewExample]) gpui.IntoElement {
	name := window.UseState(cx.App, newString(""))
	email := window.UseState(cx.App, newString("me@example.com"))
	bio := window.UseState(cx.App, newString(""))
	notes := window.UseState(cx.App, func(window *gpui.Window, cx *gpui.Context[Editor]) *Editor {
		return NewEditor("multi\nline", window, cx)
	})
	owned := window.UseState(cx.App, func(window *gpui.Window, cx *gpui.Context[Editor]) *Editor {
		return NewEditor("editable", window, cx)
	})

	return gpui.NewDiv().
		Flex().
		FlexCol().
		SizeFull().
		Bg(gpui.Rgb(0xf0f0f0)).
		P(gpui.Px(24.)).
		Gap(gpui.Px(24.)).
		Child(
			section("Inputs — from a String (cursor stays internal)").
				Child(NewInput(name).Width(gpui.Px(320.))).
				Child(
					NewInput(email).
						Width(gpui.Px(320.)).
						Color(gpui.NewHsla(0., 0., 0.3, 1.)),
				),
		).
		Child(
			section("Input — from an Editor (read its cursor beside it)").Child(
				gpui.NewDiv().
					Flex().
					ItemsCenter().
					Gap(gpui.Px(12.)).
					Child(InputEditor(owned.Clone()).Width(gpui.Px(320.))).
					Child(NewCursorReadout(owned)),
			),
		).
		Child(
			section("Text areas — from a String, or from an Editor").
				Child(NewTextArea(bio, 3)).
				Child(
					gpui.NewDiv().
						Flex().
						ItemsStart().
						Gap(gpui.Px(12.)).
						Child(TextAreaEditor(notes.Clone(), 3).Color(gpui.NewHsla(
							250./360.,
							0.7,
							0.4,
							1.,
						))).
						Child(NewCursorReadout(notes)),
				),
		)
}

func section(title string) *gpui.Div {
	return gpui.NewDiv().Flex().FlexCol().Gap(gpui.Px(8.)).Child(
		gpui.NewDiv().
			TextSm().
			TextColor(gpui.NewHsla(0., 0., 0.3, 1.)).
			Child(gpui.SharedString(title)),
	)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(560.0), gpui.Px(480.0)), cx)
		none := gpui.None[string]()
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("backspace", Backspace{}, none),
			gpui.NewKeyBinding("delete", Delete{}, none),
			gpui.NewKeyBinding("left", Left{}, none),
			gpui.NewKeyBinding("right", Right{}, none),
			gpui.NewKeyBinding("home", Home{}, none),
			gpui.NewKeyBinding("end", End{}, none),
			gpui.NewKeyBinding("enter", Enter{}, none),
			gpui.NewKeyBinding("cmd-q", Quit{}, none),
		})

		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[ViewExample] {
			return cx.New(func(*gpui.Context[ViewExample]) *ViewExample { return NewViewExample() })
		}); err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}

		cx.OnAction(func(_ Quit, cx *gpui.App) { cx.Quit() })
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
