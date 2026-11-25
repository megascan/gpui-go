// Command counter is the app built in docs/guide/getting-started.md.
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

func (c *Counter) add(delta int, cx *gpui.Context[Counter]) {
	c.count += delta
	cx.Notify()
}

func button(id string, label string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		Size10().
		Flex().
		ItemsCenter().
		JustifyCenter().
		RoundedMd().
		Bg(gpui.Rgb(0x313244)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		CursorPointer().
		ChildText(label)
}

// Render renders the view.
func (c *Counter) Render(_ *gpui.Window, cx *gpui.Context[Counter]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap4().
		SizeFull().
		JustifyCenter().
		ItemsCenter().
		Bg(gpui.Rgb(0x1e1e2e)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		Child(
			gpui.NewDiv().
				Text3xl().
				ChildText(fmt.Sprintf("%d", c.count)),
		).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				Child(
					button("decrement", "-").
						OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Counter]) {
							this.add(-1, cx)
						})),
				).
				Child(
					button("increment", "+").
						OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Counter]) {
							this.add(1, cx)
						})),
				),
		)
}

func main() {
	platform.Application().Run(func(cx *gpui.App) {
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(320), gpui.Px(240)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		options.Titlebar = gpui.Some(gpui.TitlebarOptions{Title: gpui.Some(gpui.SharedString("Counter"))})
		_, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Counter] {
			return cx.New(func(_ *gpui.Context[Counter]) *Counter {
				return &Counter{}
			})
		})
		if err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}
