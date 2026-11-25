package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// UniformListExample is the uniform_list example view.
type UniformListExample struct{}

// Render renders the view.
func (*UniformListExample) Render(_ *gpui.Window, cx *gpui.Context[UniformListExample]) gpui.IntoElement {
	return gpui.NewDiv().SizeFull().Bg(gpui.Rgb(0xffffff)).Child(
		gpui.NewUniformList(
			"entries",
			50,
			cx.Processor(func(_ *UniformListExample, visibleRange gpui.Range, _ *gpui.Window, _ *gpui.Context[UniformListExample]) []*gpui.Stateful[*gpui.Div] {
				var items []*gpui.Stateful[*gpui.Div]
				for ix := visibleRange.Start; ix < visibleRange.End; ix++ {
					item := ix + 1
					items = append(items,
						gpui.NewDiv().
							ID(ix).
							Px2().
							CursorPointer().
							OnClick(func(*gpui.ClickEvent, *gpui.Window, *gpui.App) {
								fmt.Printf("clicked Item %d\n", item)
							}).
							ChildText(fmt.Sprintf("Item %d", item)),
					)
				}
				return items
			}),
		).HFull(),
	)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(300.0), gpui.Px(300.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[UniformListExample] {
			return cx.New(func(*gpui.Context[UniformListExample]) *UniformListExample { return &UniformListExample{} })
		}); err != nil {
			panic(err)
		}
	})
}

func main() {
	runExample()
}
