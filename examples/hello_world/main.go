package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

// HelloWorld is the hello_world example view.
type HelloWorld struct {
	text gpui.SharedString
}

func swatch(bg, border gpui.Hsla) *gpui.Div {
	return gpui.NewDiv().
		Size8().
		Bg(bg).
		Border1().
		BorderDashed().
		RoundedMd().
		BorderColor(border)
}

// Render renders the view.
func (h *HelloWorld) Render(_ *gpui.Window, _ *gpui.Context[HelloWorld]) gpui.IntoElement {
	return gpui.NewDiv().
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
		ChildText(fmt.Sprintf("Hello, %s!", h.text)).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				Child(swatch(gpui.Red(), gpui.White())).
				Child(swatch(gpui.Green(), gpui.White())).
				Child(swatch(gpui.Blue(), gpui.White())).
				Child(swatch(gpui.Yellow(), gpui.White())).
				Child(swatch(gpui.Black(), gpui.White()).RoundedMd()).
				Child(swatch(gpui.White(), gpui.Black())),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(500.0)), cx)
		if _, err := cx.OpenWindow(gpui.WindowOptions{
			WindowBounds: gpui.Some(gpui.WindowBoundsWindowed(bounds)),
		}, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[HelloWorld] {
			return cx.New(func(_ *gpui.Context[HelloWorld]) *HelloWorld {
				return &HelloWorld{text: "World"}
			})
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
