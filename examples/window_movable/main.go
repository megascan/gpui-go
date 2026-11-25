package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// ExampleWindow shows the movability settings of its window.
type ExampleWindow struct {
	label       gpui.SharedString
	focusHandle gpui.FocusHandle
}

// Render renders the window content.
func (e *ExampleWindow) Render(_ *gpui.Window, _ *gpui.Context[ExampleWindow]) gpui.IntoElement {
	return gpui.NewDiv().
		TrackFocus(e.focusHandle).
		Flex().
		FlexCol().
		Gap3().
		Bg(gpui.Rgb(0x2e2e2e)).
		SizeFull().
		JustifyCenter().
		ItemsCenter().
		P8().
		TextLg().
		TextColor(gpui.Rgb(0xffffff)).
		Child(e.label).
		Child(
			gpui.NewDiv().
				TextSm().
				TextColor(gpui.Rgb(0xb0b0b0)).
				ChildText("Try to drag the titlebar, and check the Window menu."),
		)
}

func openTestWindow(cx *gpui.App, bounds gpui.Bounds[gpui.Pixels], label string, isMovable, appearsTransparent, appOwnsTitlebarDrag bool) {
	text := gpui.SharedString(fmt.Sprintf(
		"%s\nis_movable: %t\nappears_transparent: %t\napp_owns_titlebar_drag: %t",
		label, isMovable, appearsTransparent, appOwnsTitlebarDrag,
	))

	options := gpui.DefaultWindowOptions()
	options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
	options.IsMovable = isMovable
	options.AppOwnsTitlebarDrag = appOwnsTitlebarDrag
	options.Titlebar = gpui.Some(gpui.TitlebarOptions{
		Title:              gpui.Some(text),
		AppearsTransparent: appearsTransparent,
	})
	if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[ExampleWindow] {
		return cx.New(func(cx *gpui.Context[ExampleWindow]) *ExampleWindow {
			focusHandle := cx.FocusHandle()
			focusHandle.Focus(window, cx.App)
			return &ExampleWindow{label: text, focusHandle: focusHandle}
		})
	}); err != nil {
		panic(err)
	}
}

type windowSpec struct {
	label               string
	isMovable           bool
	appearsTransparent  bool
	appOwnsTitlebarDrag bool
	col, row            float32
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		windowSize := gpui.NewSize(gpui.Px(420.), gpui.Px(280.0))
		base := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), windowSize, cx)

		windows := []windowSpec{
			{"Native titlebar, movable", true, false, false, 0.0, 0.0},
			{"Native titlebar, NOT movable", false, false, false, 1.0, 0.0},
			{"Custom titlebar, movable", true, true, false, 0.0, 1.0},
			{"Custom titlebar, NOT movable", false, true, false, 1.0, 1.0},
		}

		for _, w := range windows {
			bounds := base
			bounds.Origin.X += windowSize.Width * gpui.Pixels(w.col)
			bounds.Origin.Y += windowSize.Height * gpui.Pixels(w.row)
			openTestWindow(cx, bounds, w.label, w.isMovable, w.appearsTransparent, w.appOwnsTitlebarDrag)
		}
	})
}

func main() {
	runExample()
}
