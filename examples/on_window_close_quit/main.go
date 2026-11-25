package main

import (
	"encoding/json/jsontext"
	"reflect"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// CloseWindow closes the focused window.
type CloseWindow struct{}

// Name returns the action name.
func (CloseWindow) Name() string { return "example::CloseWindow" }

// BoxedClone clones the action.
func (CloseWindow) BoxedClone() gpui.Action { return CloseWindow{} }

// PartialEq reports whether other is a CloseWindow.
func (CloseWindow) PartialEq(other gpui.Action) bool {
	_, ok := other.(CloseWindow)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "example::CloseWindow",
		Type:  reflect.TypeFor[CloseWindow](),
		Build: func(jsontext.Value) (gpui.Action, error) { return CloseWindow{}, nil },
	})
}

// ExampleWindow is a window that quits the application when the last one closes.
type ExampleWindow struct {
	focusHandle gpui.FocusHandle
}

// Render renders the window content.
func (e *ExampleWindow) Render(_ *gpui.Window, _ *gpui.Context[ExampleWindow]) gpui.IntoElement {
	return gpui.NewDiv().
		OnAction(func(_ CloseWindow, window *gpui.Window, _ *gpui.App) {
			window.RemoveWindow()
		}).
		TrackFocus(e.focusHandle).
		Flex().
		FlexCol().
		Gap3().
		Bg(gpui.Rgb(0x505050)).
		Size(gpui.Px(500.0)).
		JustifyCenter().
		ItemsCenter().
		ShadowLg().
		Border1().
		BorderColor(gpui.Rgb(0x0000ff)).
		TextXl().
		TextColor(gpui.Rgb(0xffffff)).
		ChildText("Closing this window with cmd-w or the traffic lights should quit the application!")
}

func newExampleWindow(window *gpui.Window, cx *gpui.App) gpui.Entity[ExampleWindow] {
	return cx.New(func(cx *gpui.Context[ExampleWindow]) *ExampleWindow {
		focusHandle := cx.FocusHandle()
		focusHandle.Focus(window, cx.App)
		return &ExampleWindow{focusHandle: focusHandle}
	})
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(500.0)), cx)

		cx.BindKeys([]gpui.KeyBinding{gpui.NewKeyBinding("cmd-w", CloseWindow{}, gpui.None[string]())})
		cx.OnWindowClosed(func(cx *gpui.App, _ gpui.WindowID) {
			if len(cx.Windows()) == 0 {
				cx.Quit()
			}
		}).Detach()

		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[ExampleWindow] {
			cx.Activate(false)
			return newExampleWindow(window, cx)
		}); err != nil {
			panic(err)
		}

		bounds.Origin.X += bounds.Size.Width
		options = gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, newExampleWindow); err != nil {
			panic(err)
		}
	})
}

func main() {
	runExample()
}
