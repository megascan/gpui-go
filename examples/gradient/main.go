package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// GradientViewer is the gradient example view.
type GradientViewer struct {
	colorSpace gpui.ColorSpace
}

// NewGradientViewer creates a GradientViewer with the default color space.
func NewGradientViewer() *GradientViewer {
	return &GradientViewer{colorSpace: gpui.ColorSpaceSrgb}
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func gradient(angle float32, from gpui.Hsla, fromPct float32, to gpui.Hsla, toPct float32, colorSpace gpui.ColorSpace) gpui.Background {
	return gpui.LinearGradient(
		angle,
		gpui.NewLinearColorStop(from, fromPct),
		gpui.NewLinearColorStop(to, toPct),
	).ColorSpace(colorSpace)
}

func swatch(bg gpui.Background) *gpui.Div {
	return gpui.NewDiv().Flex1().RoundedXl().Bg(bg)
}

// Render renders the view.
func (v *GradientViewer) Render(_ *gpui.Window, cx *gpui.Context[GradientViewer]) gpui.IntoElement {
	colorSpace := v.colorSpace

	return gpui.NewDiv().
		Bg(gpui.White()).
		SizeFull().
		P4().
		Flex().
		FlexCol().
		Gap3().
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				JustifyBetween().
				ItemsCenter().
				ChildText("Gradient Examples").
				Child(
					gpui.NewDiv().Flex().Gap2().ItemsCenter().Child(
						gpui.NewDiv().
							ID("method").
							Flex().
							Px3().
							Py1().
							TextSm().
							Bg(gpui.Black()).
							TextColor(gpui.White()).
							ChildText(colorSpace.String()).
							Active(func(this *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(this).Opacity(0.8) }).
							OnClick(cx.Listener(func(this *GradientViewer, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[GradientViewer]) {
								switch this.colorSpace {
								case gpui.ColorSpaceOklab:
									this.colorSpace = gpui.ColorSpaceSrgb
								case gpui.ColorSpaceSrgb:
									this.colorSpace = gpui.ColorSpaceOklab
								}
								cx.Notify()
							})),
					),
				),
		).
		Child(
			gpui.NewDiv().
				Flex().
				Flex1().
				Gap3().
				Child(
					gpui.NewDiv().
						SizeFull().
						RoundedXl().
						Flex().
						ItemsCenter().
						JustifyCenter().
						Bg(gpui.Red()).
						TextColor(gpui.White()).
						ChildText("Solid Color"),
				).
				Child(
					gpui.NewDiv().
						SizeFull().
						RoundedXl().
						Flex().
						ItemsCenter().
						JustifyCenter().
						Bg(gpui.Blue()).
						TextColor(gpui.White()).
						ChildText("Solid Color"),
				),
		).
		Child(
			gpui.NewDiv().
				Flex().
				Flex1().
				Gap3().
				H24().
				TextColor(gpui.White()).
				Child(swatch(gradient(45., gpui.Red(), 0., gpui.Blue(), 1., colorSpace))).
				Child(swatch(gradient(135., gpui.Red(), 0., gpui.Green(), 1., colorSpace))).
				Child(swatch(gradient(225., gpui.Green(), 0., gpui.Blue(), 1., colorSpace))).
				Child(swatch(gradient(315., gpui.Green(), 0., gpui.Yellow(), 1., colorSpace))),
		).
		Child(
			gpui.NewDiv().
				Flex().
				Flex1().
				Gap3().
				H24().
				TextColor(gpui.White()).
				Child(swatch(gradient(0., gpui.Red(), 0., gpui.White(), 1., colorSpace))).
				Child(swatch(gradient(90., gpui.Blue(), 0., gpui.White(), 1., colorSpace))).
				Child(swatch(gradient(180., gpui.Green(), 0., gpui.White(), 1., colorSpace))).
				Child(swatch(gradient(360., gpui.Yellow(), 0., gpui.White(), 1., colorSpace))),
		).
		Child(swatch(gradient(0., gpui.Green(), 0.05, gpui.Yellow(), 0.95, colorSpace))).
		Child(swatch(gradient(90., gpui.Blue(), 0.05, gpui.Red(), 0.95, colorSpace))).
		Child(
			gpui.NewDiv().
				Flex().
				Flex1().
				Gap3().
				Child(
					gpui.NewDiv().Flex().Flex1().Gap3().Child(
						swatch(gradient(90., gpui.Blue(), 0.5, gpui.Red(), 0.5, colorSpace)),
					),
				).
				Child(swatch(gradient(180., gpui.Green(), 0., gpui.Blue(), 0.5, colorSpace))),
		).
		Child(gpui.NewDiv().H24().Child(gpui.NewCanvas(
			func(gpui.Bounds[gpui.Pixels], *gpui.Window, *gpui.App) struct{} { return struct{}{} },
			func(bounds gpui.Bounds[gpui.Pixels], _ struct{}, window *gpui.Window, _ *gpui.App) {
				size := gpui.NewSize(bounds.Size.Width*0.8, gpui.Px(80.))
				squareBounds := gpui.Bounds[gpui.Pixels]{
					Origin: gpui.NewPoint(
						bounds.Size.Width.Half()-size.Width.Half(),
						bounds.Origin.Y,
					),
					Size: size,
				}
				height := squareBounds.Size.Height
				horizontalOffset := height
				verticalOffset := gpui.Px(30.)
				builder := gpui.PathBuilderFill()
				builder.MoveTo(squareBounds.BottomLeft())
				builder.LineTo(squareBounds.Origin.Add(gpui.NewPoint(horizontalOffset, verticalOffset)))
				builder.LineTo(squareBounds.TopRight().Add(gpui.NewPoint(-horizontalOffset, verticalOffset)))

				builder.LineTo(squareBounds.BottomRight())
				builder.LineTo(squareBounds.BottomLeft())
				path, err := builder.Build()
				if err != nil {
					panic(err)
				}
				window.PaintPath(path, gradient(180., gpui.Red(), 0., gpui.Blue(), 1., colorSpace))
			},
		)))
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		options := gpui.DefaultWindowOptions()
		options.Focus = true
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[GradientViewer] {
			return cx.New(func(*gpui.Context[GradientViewer]) *GradientViewer { return NewGradientViewer() })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
