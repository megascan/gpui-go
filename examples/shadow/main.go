package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Shadow is the shadow example view.
type Shadow struct{}

func shadowBase() *gpui.Div {
	return gpui.NewDiv().
		Size16().
		Bg(gpui.Rgb(0xffffff)).
		RoundedFull().
		Border1().
		BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 0.1))
}

func shadowSquare() *gpui.Div {
	return gpui.NewDiv().
		Size16().
		Bg(gpui.Rgb(0xffffff)).
		Border1().
		BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 0.1))
}

func shadowRoundedSmall() *gpui.Div {
	return gpui.NewDiv().
		Size16().
		Bg(gpui.Rgb(0xffffff)).
		Rounded(gpui.Px(4.)).
		Border1().
		BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 0.1))
}

func shadowRoundedMedium() *gpui.Div {
	return gpui.NewDiv().
		Size16().
		Bg(gpui.Rgb(0xffffff)).
		Rounded(gpui.Px(8.)).
		Border1().
		BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 0.1))
}

func shadowRoundedLarge() *gpui.Div {
	return gpui.NewDiv().
		Size16().
		Bg(gpui.Rgb(0xffffff)).
		Rounded(gpui.Px(12.)).
		Border1().
		BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 0.1))
}

func example(label gpui.SharedString, example gpui.IntoElement) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		JustifyCenter().
		ItemsCenter().
		W(gpui.Relative(1. / 6.)).
		BorderR1().
		BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
		Child(
			gpui.NewDiv().
				Flex().
				ItemsCenter().
				JustifyCenter().
				Flex1().
				Py12().
				Child(example),
		).
		Child(
			gpui.NewDiv().
				WFull().
				BorderT1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				P1().
				Flex().
				ItemsCenter().
				Child(label),
		)
}

// Render renders the view.
func (*Shadow) Render(_ *gpui.Window, _ *gpui.Context[Shadow]) gpui.IntoElement {
	return gpui.NewDiv().
		ID("shadow-example").
		OverflowYScroll().
		Bg(gpui.Rgb(0xffffff)).
		SizeFull().
		TextXs().
		Child(gpui.NewDiv().Flex().FlexCol().WFull().Children(
			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				FlexRow().
				Children(
					example(
						"Square",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Rounded 4",
						shadowRoundedSmall().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Rounded 8",
						shadowRoundedMedium().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Rounded 16",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Circle",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
				),
			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				WFull().
				Children(
					example("None", shadowBase()),

					example("2X Small", shadowBase().Shadow2xs()),

					example("Extra Small", shadowBase().ShadowXs()),

					example("Small", shadowBase().ShadowSm()),

					example("Medium", shadowBase().ShadowMd()),

					example("Large", shadowBase().ShadowLg()),
					example("Extra Large", shadowBase().ShadowXl()),
					example("2X Large", shadowBase().Shadow2xl()),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				Children(
					example(
						"Blur 0",
						shadowBase().Shadow([]gpui.BoxShadow{gpui.NewBoxShadow(
							gpui.Px(0.),
							gpui.Px(8.),
							gpui.NewHsla(0.0, 0.0, 0.0, 0.3),
						)}),
					),
					example(
						"Blur 2",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(2.)),
						}),
					),
					example(
						"Blur 4",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(4.)),
						}),
					),
					example(
						"Blur 8",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Blur 16",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(16.)),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				Children(
					example(
						"Spread 0",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Spread 2",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
						}),
					),
					example(
						"Spread 4",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(4.)),
						}),
					),
					example(
						"Spread 8",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Spread 16",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(16.)),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				Children(
					example(
						"Square Spread 0",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Square Spread 8",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Square Spread 16",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(16.)),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				Children(
					example(
						"Rounded Large Spread 0",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Rounded Large Spread 8",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Rounded Large Spread 16",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.3)).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(16.)),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				Children(
					example(
						"Left",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(-8.), gpui.Px(0.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Right",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(8.), gpui.Px(0.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Top",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(-8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Bottom",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				Children(
					example(
						"Square Left",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(-8.), gpui.Px(0.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Square Right",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(8.), gpui.Px(0.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Square Top",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(-8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Square Bottom",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				Children(
					example(
						"Rounded Large Left",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(-8.), gpui.Px(0.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Rounded Large Right",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(8.), gpui.Px(0.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Rounded Large Top",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(-8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
					example(
						"Rounded Large Bottom",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.5, 0.5, 0.3)).
								BlurRadius(gpui.Px(8.)),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				Children(
					example(
						"Circle Multiple",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(
								gpui.Px(0.),
								gpui.Px(-12.),
								gpui.NewHsla(0.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(12.),
								gpui.Px(0.),
								gpui.NewHsla(60.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(0.),
								gpui.Px(12.),
								gpui.NewHsla(120.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(-12.),
								gpui.Px(0.),
								gpui.NewHsla(240.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
						}),
					),
					example(
						"Square Multiple",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(
								gpui.Px(0.),
								gpui.Px(-12.),
								gpui.NewHsla(0.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(12.),
								gpui.Px(0.),
								gpui.NewHsla(60.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(0.),
								gpui.Px(12.),
								gpui.NewHsla(120.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(-12.),
								gpui.Px(0.),
								gpui.NewHsla(240.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
						}),
					),
					example(
						"Rounded Large Multiple",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(
								gpui.Px(0.),
								gpui.Px(-12.),
								gpui.NewHsla(0.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(12.),
								gpui.Px(0.),
								gpui.NewHsla(60.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(0.),
								gpui.Px(12.),
								gpui.NewHsla(120.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
							gpui.NewBoxShadow(
								gpui.Px(-12.),
								gpui.Px(0.),
								gpui.NewHsla(240.0/360., 1.0, 0.5, 0.3),
							).
								BlurRadius(gpui.Px(8.)).
								SpreadRadius(gpui.Px(2.)),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				WFull().
				Children(
					example(
						"Inset basic",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(0.), gpui.NewHsla(0.0, 0.0, 0.0, 0.5)).
								BlurRadius(gpui.Px(12.)).
								Inset(),
						}),
					),
					example(
						"Inset offset",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(6.), gpui.Px(6.), gpui.NewHsla(0.0, 0.0, 0.0, 0.5)).
								BlurRadius(gpui.Px(8.)).
								Inset(),
						}),
					),
					example(
						"Inset spread",
						shadowBase().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(0.), gpui.NewHsla(0.0, 0.0, 0.0, 0.5)).
								BlurRadius(gpui.Px(4.)).
								SpreadRadius(gpui.Px(8.)).
								Inset(),
						}),
					),
					example(
						"Inset rounded",
						shadowRoundedLarge().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(4.), gpui.NewHsla(0.0, 0.0, 0.0, 0.5)).
								BlurRadius(gpui.Px(10.)).
								SpreadRadius(gpui.Px(2.)).
								Inset(),
						}),
					),
					example(
						"Inset sharp",
						shadowSquare().Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(0.), gpui.NewHsla(0.0, 0.0, 0.0, 0.6)).
								SpreadRadius(gpui.Px(6.)).
								Inset(),
						}),
					),
				),

			gpui.NewDiv().
				BorderB1().
				BorderColor(gpui.NewHsla(0.0, 0.0, 0.0, 1.0)).
				Flex().
				WFull().
				Children(example(
					"Drop + Inset",
					shadowRoundedMedium().Shadow([]gpui.BoxShadow{
						gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(8.), gpui.NewHsla(0.0, 0.0, 0.0, 0.25)).
							BlurRadius(gpui.Px(12.)),
						gpui.NewBoxShadow(gpui.Px(0.), gpui.Px(2.), gpui.NewHsla(0.0, 0.0, 0.0, 0.4)).
							BlurRadius(gpui.Px(4.)).
							Inset(),
					}),
				)),
		))
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(1000.0), gpui.Px(800.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Shadow] {
			return cx.New(func(_ *gpui.Context[Shadow]) *Shadow { return &Shadow{} })
		}); err != nil {
			panic(err)
		}

		cx.Activate(true)
	})
}

func main() {
	runExample()
}
