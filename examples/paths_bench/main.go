package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

const (
	defaultWindowWidth  gpui.Pixels = 1024.0
	defaultWindowHeight gpui.Pixels = 768.0
)

type pathLine struct {
	path  gpui.Path
	color gpui.Background
}

// PaintingViewer paints 2000 star paths every frame.
type PaintingViewer struct {
	defaultLines []pathLine
	painting     bool
}

// NewPaintingViewer builds the star paths.
func NewPaintingViewer(_ *gpui.Window, _ *gpui.Context[PaintingViewer]) *PaintingViewer {
	var lines []pathLine

	for range 2000 {
		builder := gpui.PathBuilderFill()
		builder.MoveTo(gpui.NewPoint(gpui.Px(350.), gpui.Px(100.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(370.), gpui.Px(160.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(430.), gpui.Px(160.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(380.), gpui.Px(200.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(400.), gpui.Px(260.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(350.), gpui.Px(220.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(300.), gpui.Px(260.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(320.), gpui.Px(200.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(270.), gpui.Px(160.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(330.), gpui.Px(160.)))
		builder.LineTo(gpui.NewPoint(gpui.Px(350.), gpui.Px(100.)))
		path, err := builder.Build()
		if err != nil {
			panic(err)
		}
		lines = append(lines, pathLine{
			path: path,
			color: gpui.LinearGradient(
				180.,
				gpui.NewLinearColorStop(gpui.Rgb(0xFACC15), 0.7),
				gpui.NewLinearColorStop(gpui.Rgb(0xD56D0C), 1.),
			).ColorSpace(gpui.ColorSpaceOklab),
		})
	}

	return &PaintingViewer{
		defaultLines: lines,
		painting:     false,
	}
}

// Render renders the view.
func (v *PaintingViewer) Render(window *gpui.Window, _ *gpui.Context[PaintingViewer]) gpui.IntoElement {
	window.RequestAnimationFrame()
	lines := v.defaultLines
	return gpui.NewDiv().SizeFull().Child(
		gpui.NewCanvas(
			func(gpui.Bounds[gpui.Pixels], *gpui.Window, *gpui.App) struct{} { return struct{}{} },
			func(_ gpui.Bounds[gpui.Pixels], _ struct{}, window *gpui.Window, _ *gpui.App) {
				for _, line := range lines {
					window.PaintPath(line.path, line.color)
				}
			},
		).SizeFull(),
	)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		options := gpui.DefaultWindowOptions()
		options.Titlebar = gpui.Some(gpui.TitlebarOptions{Title: gpui.Some[gpui.SharedString]("Vulkan")})
		options.Focus = true
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.BoundsCentered(
			gpui.None[gpui.DisplayID](),
			gpui.NewSize(defaultWindowWidth, defaultWindowHeight),
			cx,
		)))
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[PaintingViewer] {
			return cx.New(func(cx *gpui.Context[PaintingViewer]) *PaintingViewer { return NewPaintingViewer(window, cx) })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
