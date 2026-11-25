package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

// Example is the active_state_bug example view.
type Example struct{}

// Render renders the view.
func (*Example) Render(_ *gpui.Window, _ *gpui.Context[Example]) gpui.IntoElement {
	bg := gpui.NewHsla(215./360., 0.12, 0.15, 1.)
	text := gpui.NewHsla(221./360., 0.11, 0.86, 1.)
	hover := gpui.NewHsla(225./360., 0.118, 0.267, 1.)
	active := gpui.NewHsla(220./360., 0.118, 0.20, 1.)

	return gpui.NewDiv().Bg(bg).SizeFull().P1().Child(
		gpui.NewDiv().
			ID("button").
			Px2().
			Py0p5().
			RoundedMd().
			TextSm().
			TextColor(text).
			Hover(func(s *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(s).Bg(hover) }).
			Active(func(s *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(s).Bg(active) }).
			OnClick(func(*gpui.ClickEvent, *gpui.Window, *gpui.App) {}).
			ChildText("Click me"),
	)
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func main() {
	platform.Application().Run(func(cx *gpui.App) {
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.BoundsCentered(
			gpui.None[gpui.DisplayID](),
			gpui.NewSize(gpui.Px(200.), gpui.Px(60.)),
			cx,
		)))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Example] {
			return cx.New(func(*gpui.Context[Example]) *Example { return &Example{} })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}
