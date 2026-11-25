package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// PatternExample is the pattern example view.
type PatternExample struct{}

func slash(w, h float32, color gpui.Hsla, width, interval float32) *gpui.Div {
	return gpui.NewDiv().W(gpui.Px(w)).H(gpui.Px(h)).Bg(gpui.PatternSlash(color, width, interval))
}

// Render renders the view.
func (*PatternExample) Render(_ *gpui.Window, _ *gpui.Context[PatternExample]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap3().
		Bg(gpui.Rgb(0xffffff)).
		Size(gpui.Px(600.0)).
		JustifyCenter().
		ItemsCenter().
		ShadowLg().
		TextXl().
		TextColor(gpui.Rgb(0x000000)).
		ChildText("Pattern Example").
		Child(
			gpui.NewDiv().
				Flex().
				FlexCol().
				Border1().
				BorderColor(gpui.Blue()).
				Child(slash(54.0, 18.0, gpui.Red(), 18.0/4.0, 18.0/4.0)).
				Child(slash(54.0, 18.0, gpui.Red(), 18.0/4.0, 18.0/4.0)).
				Child(slash(54.0, 18.0, gpui.Red(), 18.0/4.0, 18.0/4.0)).
				Child(slash(54.0, 18.0, gpui.Red(), 18.0/4.0, 18.0/2.0)),
		).
		Child(
			gpui.NewDiv().
				Flex().
				FlexCol().
				Border1().
				BorderColor(gpui.Blue()).
				Bg(gpui.Green().Opacity(0.16)).
				ChildText("Elements the same height should align").
				Child(slash(256.0, 56.0, gpui.Red(), 56.0/6.0, 56.0/6.0)).
				Child(slash(256.0, 56.0, gpui.Green(), 56.0/6.0, 56.0/6.0)).
				Child(slash(256.0, 56.0, gpui.Blue(), 56.0/6.0, 56.0/6.0)).
				Child(slash(256.0, 26.0, gpui.Yellow(), 56.0/6.0, 56.0/6.0)),
		).
		Child(
			gpui.NewDiv().
				Border1().
				BorderColor(gpui.Blue()).
				W(gpui.Px(240.0)).
				H(gpui.Px(40.0)).
				Bg(gpui.Red()),
		).
		Child(
			gpui.NewDiv().
				Border1().
				BorderColor(gpui.Blue()).
				W(gpui.Px(240.0)).
				H(gpui.Px(40.0)).
				Bg(gpui.LinearGradient(
					45.,
					gpui.NewLinearColorStop(gpui.Red(), 0.),
					gpui.NewLinearColorStop(gpui.Blue(), 1.),
				)),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(600.0), gpui.Px(600.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[PatternExample] {
			return cx.New(func(*gpui.Context[PatternExample]) *PatternExample { return &PatternExample{} })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
