package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// HolyGrailExample lays out https://en.wikipedia.org/wiki/Holy_grail_(web_design) with a
// container query that collapses to one column when the container is too narrow.
type HolyGrailExample struct{}

// Render renders the view.
func (*HolyGrailExample) Render(_ *gpui.Window, _ *gpui.Context[HolyGrailExample]) gpui.IntoElement {
	return gpui.NewContainerQuery(func(containerSize gpui.Size[gpui.Pixels], _ *gpui.Window, _ *gpui.App) *gpui.Div {
		block := func(color gpui.Hsla) *gpui.Div {
			return gpui.NewDiv().
				SizeFull().
				Bg(color).
				Border1().
				BorderDashed().
				RoundedMd().
				BorderColor(gpui.White()).
				ItemsCenter()
		}

		header := block(gpui.White()).ChildText(fmt.Sprintf("Header — %s", containerSize.Width))
		tableOfContents := block(gpui.Red()).ChildText("Table of contents")
		content := block(gpui.Green()).ChildText("Content")
		ad := block(gpui.Blue()).ChildText("AD :(").TextColor(gpui.White())
		footer := block(gpui.Black()).
			TextColor(gpui.White()).
			ChildText("Footer")

		container := gpui.NewDiv().Gap1().Bg(gpui.Rgb(0x505050)).ShadowLg().SizeFull()

		if containerSize.Width < gpui.Px(400.) {
			return container.
				Flex().
				FlexCol().
				Child(header.H12().FlexNone()).
				Child(tableOfContents.H20().FlexNone()).
				Child(content.Flex1()).
				Child(ad.H20().FlexNone()).
				Child(footer.H12().FlexNone())
		}
		return container.
			Grid().
			GridCols(5).
			GridRows(5).
			Child(header.RowSpan(1).ColSpanFull()).
			Child(tableOfContents.ColSpan(1).H56()).
			Child(content.ColSpan(3).RowSpan(3)).
			Child(ad.ColSpan(1).RowSpan(3)).
			Child(footer.RowSpan(1).ColSpanFull())
	})
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(500.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[HolyGrailExample] {
			return cx.New(func(*gpui.Context[HolyGrailExample]) *HolyGrailExample { return &HolyGrailExample{} })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
