package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// WindowContent shows the bounds a window was requested with and the bounds it got.
type WindowContent struct {
	text   gpui.SharedString
	bounds gpui.Bounds[gpui.Pixels]
	bg     gpui.Hsla
}

// Render renders the window content.
func (w *WindowContent) Render(window *gpui.Window, _ *gpui.Context[WindowContent]) gpui.IntoElement {
	windowBounds := window.Bounds()

	return gpui.NewDiv().
		Flex().
		FlexCol().
		Bg(w.bg).
		SizeFull().
		ItemsCenter().
		TextColor(gpui.Rgb(0xffffff)).
		Child(w.text).
		Child(
			gpui.NewDiv().
				Flex().
				FlexCol().
				TextSm().
				ItemsCenter().
				SizeFull().
				ChildText(fmt.Sprintf(
					"origin: %s, %s size: %s, %s",
					w.bounds.Origin.X,
					w.bounds.Origin.Y,
					w.bounds.Size.Width,
					w.bounds.Size.Height,
				)).
				ChildText(fmt.Sprintf(
					"cx.bounds() origin: %s, %s size %s, %s",
					windowBounds.Origin.X,
					windowBounds.Origin.Y,
					windowBounds.Size.Width,
					windowBounds.Size.Height,
				)),
		)
}

func buildWindowOptions(displayID gpui.DisplayID, bounds gpui.Bounds[gpui.Pixels]) gpui.WindowOptions {
	options := gpui.DefaultWindowOptions()
	options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
	options.DisplayID = gpui.Some(displayID)
	options.Titlebar = gpui.None[gpui.TitlebarOptions]()
	options.WindowBackground = gpui.WindowBackgroundAppearanceTransparent
	options.Focus = false
	options.Show = true
	options.Kind = gpui.WindowKindPopUp{}
	options.IsMovable = false
	options.AppID = gpui.None[string]()
	options.WindowMinSize = gpui.None[gpui.Size[gpui.Pixels]]()
	options.WindowDecorations = gpui.None[gpui.WindowDecorations]()
	options.TabbingIdentifier = gpui.None[string]()
	return options
}

func openContent(cx *gpui.App, screen gpui.PlatformDisplay, bounds gpui.Bounds[gpui.Pixels], label string, bg gpui.Hsla) {
	if _, err := cx.OpenWindow(buildWindowOptions(screen.ID(), bounds), func(_ *gpui.Window, cx *gpui.App) gpui.Entity[WindowContent] {
		return cx.New(func(_ *gpui.Context[WindowContent]) *WindowContent {
			return &WindowContent{
				text:   gpui.SharedString(fmt.Sprintf("%s %v", label, screen.ID())),
				bg:     bg,
				bounds: bounds,
			}
		})
	}); err != nil {
		panic(err)
	}
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		size := gpui.Size[gpui.Pixels]{
			Width:  gpui.Px(350.),
			Height: gpui.Px(75.),
		}
		marginOffset := gpui.Px(150.)

		for _, screen := range cx.Displays() {
			screenBounds := screen.Bounds()

			bounds := gpui.Bounds[gpui.Pixels]{
				Origin: screenBounds.Origin.Add(gpui.NewPoint(marginOffset, marginOffset)),
				Size:   size,
			}
			openContent(cx, screen, bounds, "Top Left", gpui.Red())

			bounds = gpui.Bounds[gpui.Pixels]{
				Origin: screenBounds.TopRight().Sub(gpui.NewPoint(size.Width+marginOffset, -marginOffset)),
				Size:   size,
			}
			openContent(cx, screen, bounds, "Top Right", gpui.Red())

			bounds = gpui.Bounds[gpui.Pixels]{
				Origin: screenBounds.BottomLeft().Sub(gpui.NewPoint(-marginOffset, size.Height+marginOffset)),
				Size:   size,
			}
			openContent(cx, screen, bounds, "Bottom Left", gpui.Blue())

			bounds = gpui.Bounds[gpui.Pixels]{
				Origin: screenBounds.BottomRight().Sub(gpui.NewPoint(size.Width+marginOffset, size.Height+marginOffset)),
				Size:   size,
			}
			openContent(cx, screen, bounds, "Bottom Right", gpui.Blue())

			bounds = gpui.Bounds[gpui.Pixels]{
				Origin: gpui.NewPoint(
					screenBounds.Center().X-size.Center().X,
					screenBounds.Top()+marginOffset,
				),
				Size: size,
			}
			openContent(cx, screen, bounds, "Top Center", gpui.Black())

			bounds = gpui.Bounds[gpui.Pixels]{
				Origin: gpui.NewPoint(
					screenBounds.Left()+marginOffset,
					screenBounds.Center().Y-size.Center().Y,
				),
				Size: size,
			}
			openContent(cx, screen, bounds, "Left Center", gpui.Black())

			bounds = gpui.Bounds[gpui.Pixels]{
				Origin: gpui.NewPoint(
					screenBounds.Center().X-size.Center().X,
					screenBounds.Center().Y-size.Center().Y,
				),
				Size: size,
			}
			openContent(cx, screen, bounds, "Center", gpui.Black())

			bounds = gpui.Bounds[gpui.Pixels]{
				Origin: gpui.NewPoint(
					screenBounds.Right()-size.Width-marginOffset,
					screenBounds.Center().Y-size.Center().Y,
				),
				Size: size,
			}
			openContent(cx, screen, bounds, "Right Center", gpui.Black())

			bounds = gpui.Bounds[gpui.Pixels]{
				Origin: gpui.NewPoint(
					screenBounds.Center().X-size.Center().X,
					screenBounds.Bottom()-size.Height-marginOffset,
				),
				Size: size,
			}
			openContent(cx, screen, bounds, "Bottom Center", gpui.Black())
		}
	})
}

func main() {
	runExample()
}
