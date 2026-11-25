package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Scrollable is the scrollable example view.
type Scrollable struct{}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

// Render renders the view.
func (*Scrollable) Render(_ *gpui.Window, _ *gpui.Context[Scrollable]) gpui.IntoElement {
	return gpui.NewDiv().
		SizeFull().
		ID("vertical").
		P4().
		OverflowScroll().
		Bg(gpui.White()).
		ChildText("Example for test 2 way scroll in nested layout").
		Child(
			gpui.NewDiv().
				H(gpui.Px(5000.)).
				Border1().
				BorderColor(gpui.Blue()).
				Bg(gpui.Blue().Opacity(0.05)).
				P4().
				Child(
					gpui.NewDiv().
						Mb5().
						WFull().
						ID("horizontal").
						OverflowScroll().
						Child(
							gpui.NewDiv().
								W(gpui.Px(2000.)).
								H(gpui.Px(150.)).
								Bg(gpui.Green().Opacity(0.1)).
								Hover(func(this *gpui.StyleRefinement) *gpui.StyleRefinement {
									return styled(this).Bg(gpui.Green().Opacity(0.2))
								}).
								Border1().
								BorderColor(gpui.Green()).
								P4().
								ChildText("Scroll Horizontal"),
						),
				).
				ChildText("Scroll Vertical"),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(500.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Scrollable] {
			return cx.New(func(*gpui.Context[Scrollable]) *Scrollable { return &Scrollable{} })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
