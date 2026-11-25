// Command step1 is the first step of docs/guide/getting-started.md.
package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

// Counter is the root view.
type Counter struct {
	count int
}

// Render renders the view.
func (c *Counter) Render(_ *gpui.Window, _ *gpui.Context[Counter]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		SizeFull().
		JustifyCenter().
		ItemsCenter().
		Bg(gpui.Rgb(0x1e1e2e)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		Text3xl().
		ChildText(fmt.Sprintf("%d", c.count))
}

func main() {
	platform.Application().Run(func(cx *gpui.App) {
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(320), gpui.Px(240)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		_, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Counter] {
			return cx.New(func(_ *gpui.Context[Counter]) *Counter {
				return &Counter{}
			})
		})
		if err != nil {
			panic(err)
		}
	})
}
