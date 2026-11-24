package main

import (
	"math"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/internal/lyon"
	"github.com/megascan/gpui-go/platform"
)

type pathColor struct {
	path  gpui.Path
	color gpui.Background
}

type quadColor struct {
	bounds gpui.Bounds[gpui.Pixels]
	color  gpui.Background
}

// PaintingViewer is the painting example view.
type PaintingViewer struct {
	defaultLines    []pathColor
	backgroundQuads []quadColor
	lines           [][]gpui.Point[gpui.Pixels]
	start           gpui.Point[gpui.Pixels]
	dashed          bool
	painting        bool
}

func point(x, y gpui.Pixels) gpui.Point[gpui.Pixels] {
	return gpui.NewPoint(x, y)
}

func px(v float32) gpui.Pixels {
	return gpui.Px(v)
}

func mustBuild(builder *gpui.PathBuilder) gpui.Path {
	path, err := builder.Build()
	if err != nil {
		panic(err)
	}
	return path
}

func pathBuilderFromLyon(path lyon.Path) *gpui.PathBuilder {
	builder := gpui.PathBuilderFill()
	for event := range path.Iter() {
		switch event.Kind {
		case lyon.EventBegin:
			builder.MoveTo(point(px(event.At.X), px(event.At.Y)))
		case lyon.EventLine:
			builder.LineTo(point(px(event.To.X), px(event.To.Y)))
		case lyon.EventQuadratic:
			builder.CurveTo(point(px(event.To.X), px(event.To.Y)), point(px(event.Ctrl.X), px(event.Ctrl.Y)))
		case lyon.EventCubic:
			builder.CubicBezierTo(
				point(px(event.To.X), px(event.To.Y)),
				point(px(event.Ctrl1.X), px(event.Ctrl1.Y)),
				point(px(event.Ctrl2.X), px(event.Ctrl2.Y)),
			)
		case lyon.EventEnd:
			if event.Close {
				builder.Close()
			}
		}
	}
	return builder
}

// NewPaintingViewer creates the view with its default paths.
func NewPaintingViewer(_ *gpui.Window, _ *gpui.Context[PaintingViewer]) *PaintingViewer {
	var lines []pathColor

	backgroundQuads := []quadColor{
		{gpui.Bounds[gpui.Pixels]{Origin: point(px(70.), px(70.)), Size: gpui.NewSize(px(40.), px(40.))}, gpui.Black().ToBackground()},
		{gpui.Bounds[gpui.Pixels]{Origin: point(px(170.), px(70.)), Size: gpui.NewSize(px(40.), px(40.))}, gpui.Black().ToBackground()},
		{gpui.Bounds[gpui.Pixels]{Origin: point(px(270.), px(70.)), Size: gpui.NewSize(px(40.), px(40.))}, gpui.Black().ToBackground()},
		{gpui.Bounds[gpui.Pixels]{Origin: point(px(370.), px(70.)), Size: gpui.NewSize(px(40.), px(40.))}, gpui.Black().ToBackground()},
		{gpui.Bounds[gpui.Pixels]{Origin: point(px(450.), px(50.)), Size: gpui.NewSize(px(80.), px(80.))}, gpui.Black().ToBackground()},
	}

	square := func(left, right float32, color gpui.Rgba) {
		builder := gpui.PathBuilderFill()
		builder.MoveTo(point(px(left), px(50.)))
		builder.LineTo(point(px(right), px(50.)))
		builder.LineTo(point(px(right), px(130.)))
		builder.LineTo(point(px(left), px(130.)))
		builder.Close()
		lines = append(lines, pathColor{mustBuild(builder), color.ToBackground()})
	}
	square(50., 130., gpui.Rgb(0xFF0000).Alpha(0.5))
	square(150., 230., gpui.Rgb(0x0000FF).Alpha(0.5))
	square(250., 330., gpui.Rgb(0x00FF00).Alpha(0.5))
	square(350., 430., gpui.Rgb(0x000000).Alpha(0.5))

	circle := func(center gpui.Point[gpui.Pixels]) {
		builder := gpui.PathBuilderFill()
		radius := px(30.)
		builder.MoveTo(point(center.X+radius, center.Y))
		builder.ArcTo(point(radius, radius), px(0.), false, false, point(center.X-radius, center.Y))
		builder.ArcTo(point(radius, radius), px(0.), false, false, point(center.X+radius, center.Y))
		builder.Close()
		lines = append(lines, pathColor{mustBuild(builder), gpui.Rgb(0xFF0000).Alpha(0.5).ToBackground()})
	}
	circle(point(px(530.), px(85.)))
	circle(point(px(570.), px(85.)))

	logo := lyon.PathSvgBuilder()
	buildLogoPath(logo)
	builder := pathBuilderFromLyon(logo.Build())
	builder.Translate(point(px(10.), px(200.)))
	builder.Scale(0.9)
	lines = append(lines, pathColor{mustBuild(builder), gpui.Black().ToBackground()})

	builder = gpui.PathBuilderFill()
	builder.AddPolygon([]gpui.Point[gpui.Pixels]{
		point(px(150.), px(300.)),
		point(px(200.), px(225.)),
		point(px(200.), px(275.)),
		point(px(250.), px(200.)),
	}, false)
	lines = append(lines, pathColor{mustBuild(builder), gpui.Rgb(0x1d4ed8).ToBackground()})

	builder = gpui.PathBuilderFill()
	builder.MoveTo(point(px(350.), px(200.)))
	builder.LineTo(point(px(370.), px(260.)))
	builder.LineTo(point(px(430.), px(260.)))
	builder.LineTo(point(px(380.), px(300.)))
	builder.LineTo(point(px(400.), px(360.)))
	builder.LineTo(point(px(350.), px(320.)))
	builder.LineTo(point(px(300.), px(360.)))
	builder.LineTo(point(px(320.), px(300.)))
	builder.LineTo(point(px(270.), px(260.)))
	builder.LineTo(point(px(330.), px(260.)))
	builder.LineTo(point(px(350.), px(200.)))
	lines = append(lines, pathColor{
		mustBuild(builder),
		gpui.LinearGradient(
			180.,
			gpui.NewLinearColorStop(gpui.Rgb(0xFACC15), 0.7),
			gpui.NewLinearColorStop(gpui.Rgb(0xD56D0C), 1.),
		).ColorSpace(gpui.ColorSpaceOklab),
	})

	squareBounds := gpui.Bounds[gpui.Pixels]{
		Origin: point(px(450.), px(200.)),
		Size:   gpui.NewSize(px(200.), px(80.)),
	}
	height := squareBounds.Size.Height
	horizontalOffset := height
	verticalOffset := px(30.)
	builder = gpui.PathBuilderFill()
	builder.MoveTo(squareBounds.BottomLeft())
	builder.CurveTo(
		squareBounds.Origin.Add(point(horizontalOffset, verticalOffset)),
		squareBounds.Origin.Add(point(px(0.0), verticalOffset)),
	)
	builder.LineTo(squareBounds.TopRight().Add(point(-horizontalOffset, verticalOffset)))
	builder.CurveTo(
		squareBounds.BottomRight(),
		squareBounds.TopRight().Add(point(px(0.0), verticalOffset)),
	)
	builder.LineTo(squareBounds.BottomLeft())
	lines = append(lines, pathColor{
		mustBuild(builder),
		gpui.LinearGradient(
			180.,
			gpui.NewLinearColorStop(gpui.Blue(), 0.4),
			gpui.NewLinearColorStop(gpui.Red(), 1.),
		),
	})

	center := point(px(96.), px(96.))
	pieCenter := point(px(775.), px(255.))
	segments := []struct {
		start, end gpui.Point[gpui.Pixels]
		color      gpui.Rgba
	}{
		{point(px(871.), px(255.)), point(px(747.), px(163.)), gpui.Rgb(0x1374e9)},
		{point(px(747.), px(163.)), point(px(679.), px(263.)), gpui.Rgb(0xe13527)},
		{point(px(679.), px(263.)), point(px(754.), px(349.)), gpui.Rgb(0x0751ce)},
		{point(px(754.), px(349.)), point(px(854.), px(310.)), gpui.Rgb(0x209742)},
		{point(px(854.), px(310.)), point(px(871.), px(255.)), gpui.Rgb(0xfbc10a)},
	}

	for _, segment := range segments {
		builder := gpui.PathBuilderFill()
		builder.MoveTo(segment.start)
		builder.ArcTo(center, px(0.), false, false, segment.end)
		builder.LineTo(pieCenter)
		builder.Close()
		lines = append(lines, pathColor{mustBuild(builder), segment.color.ToBackground()})
	}

	options := gpui.DefaultStrokeOptions()
	options.LineWidth = 1.
	options.LineJoin = gpui.LineJoinBevel
	builder = gpui.PathBuilderStroke(px(1.)).WithStyle(gpui.PathStyleStroke{StrokeOptions: options})
	builder.MoveTo(point(px(40.), px(420.)))
	for i := 1; i < 50; i++ {
		builder.LineTo(point(
			px(40.0+float32(i)*10.0),
			px(420.0+float32(math.Sin(float64(float32(i)*10.0)))*40.0),
		))
	}
	lines = append(lines, pathColor{mustBuild(builder), gpui.Green().ToBackground()})

	return &PaintingViewer{
		defaultLines:    lines,
		backgroundQuads: backgroundQuads,
		lines:           nil,
		start:           point(px(0.), px(0.)),
		dashed:          false,
		painting:        false,
	}
}

func (p *PaintingViewer) clear(cx *gpui.Context[PaintingViewer]) {
	p.lines = p.lines[:0]
	cx.Notify()
}

func (p *PaintingViewer) stressScribble(cx *gpui.Context[PaintingViewer]) {
	p.lines = p.lines[:0]
	var state uint32 = 0x9E3779B9
	random := func() float32 {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		return float32(state) / float32(math.MaxUint32)
	}
	for range 30 {
		x := 40.0 + random()*860.0
		y := 80.0 + random()*440.0
		angle := random() * (2 * math.Pi)
		points := make([]gpui.Point[gpui.Pixels], 0, 121)
		points = append(points, point(px(x), px(y)))
		for range 120 {
			angle += (random() - 0.5) * 1.2
			x += float32(math.Cos(float64(angle))) * 12.0
			y += float32(math.Sin(float64(angle))) * 12.0
			if !(x >= 40.0 && x <= 900.0) {
				angle = math.Pi - angle
				x = min(max(x, 40.0), 900.0)
			}
			if !(y >= 80.0 && y <= 520.0) {
				angle = -angle
				y = min(max(y, 80.0), 520.0)
			}
			points = append(points, point(px(x), px(y)))
		}
		p.lines = append(p.lines, points)
	}
	cx.Notify()
}

func (p *PaintingViewer) stressGrid(cx *gpui.Context[PaintingViewer]) {
	p.lines = p.lines[:0]
	for i := range 55 {
		y := 80.0 + float32(i)*8.0
		p.lines = append(p.lines, []gpui.Point[gpui.Pixels]{point(px(40.0), px(y)), point(px(900.0), px(y))})
	}
	for i := range 108 {
		x := 40.0 + float32(i)*8.0
		p.lines = append(p.lines, []gpui.Point[gpui.Pixels]{point(px(x), px(80.0)), point(px(x), px(520.0))})
	}
	cx.Notify()
}

func button(text string, cx *gpui.Context[PaintingViewer], onClick func(this *PaintingViewer, cx *gpui.Context[PaintingViewer])) gpui.IntoElement {
	return gpui.NewDiv().
		ID(text).
		ChildText(text).
		Bg(gpui.Black()).
		TextColor(gpui.White()).
		Active(func(this *gpui.StyleRefinement) *gpui.StyleRefinement {
			var s gpui.Styled[*gpui.StyleRefinement]
			gpui.BindStyled(&s, this)
			return s.Opacity(0.8)
		}).
		Flex().
		Px3().
		Py1().
		OnClick(cx.Listener(func(this *PaintingViewer, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[PaintingViewer]) {
			onClick(this, cx)
		}))
}

// Render renders the view.
func (p *PaintingViewer) Render(_ *gpui.Window, cx *gpui.Context[PaintingViewer]) gpui.IntoElement {
	defaultLines := append([]pathColor(nil), p.defaultLines...)
	backgroundQuads := append([]quadColor(nil), p.backgroundQuads...)
	lines := make([][]gpui.Point[gpui.Pixels], len(p.lines))
	for i, l := range p.lines {
		lines[i] = append([]gpui.Point[gpui.Pixels](nil), l...)
	}
	dashed := p.dashed

	dashedLabel := "Dashed"
	if dashed {
		dashedLabel = "Solid"
	}

	return gpui.NewDiv().
		Bg(gpui.White()).
		SizeFull().
		P4().
		Flex().
		FlexCol().
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				JustifyBetween().
				ItemsCenter().
				ChildText("Mouse down any point and drag to draw lines (Hold on shift key to draw straight lines)").
				Child(
					gpui.NewDiv().
						Flex().
						GapX2().
						Child(button(dashedLabel, cx, func(this *PaintingViewer, _ *gpui.Context[PaintingViewer]) {
							this.dashed = !dashed
						})).
						Child(button("Scribble", cx, func(this *PaintingViewer, cx *gpui.Context[PaintingViewer]) { this.stressScribble(cx) })).
						Child(button("Grid", cx, func(this *PaintingViewer, cx *gpui.Context[PaintingViewer]) { this.stressGrid(cx) })).
						Child(button("Clear", cx, func(this *PaintingViewer, cx *gpui.Context[PaintingViewer]) { this.clear(cx) })),
				),
		).
		Child(
			gpui.NewDiv().
				SizeFull().
				Child(
					gpui.NewCanvas(
						func(_ gpui.Bounds[gpui.Pixels], _ *gpui.Window, _ *gpui.App) struct{} { return struct{}{} },
						func(_ gpui.Bounds[gpui.Pixels], _ struct{}, window *gpui.Window, _ *gpui.App) {
							for _, q := range backgroundQuads {
								window.PaintQuad(gpui.NewQuad(
									q.bounds,
									gpui.CornersAll(px(0.)),
									q.color,
									gpui.EdgesAll(px(0.)),
									gpui.TransparentBlack(),
									gpui.BorderStyleSolid,
								))
							}

							for _, l := range defaultLines {
								window.PaintPath(l.path, l.color)
							}

							for _, points := range lines {
								if len(points) < 2 {
									continue
								}

								builder := gpui.PathBuilderStroke(px(1.))
								if dashed {
									builder = builder.DashArray([]gpui.Pixels{px(4.), px(2.)})
								}
								for i, pt := range points {
									if i == 0 {
										builder.MoveTo(pt)
									} else {
										builder.LineTo(pt)
									}
								}

								if path, err := builder.Build(); err == nil {
									window.PaintPath(path, gpui.Black().ToBackground())
								}
							}
						},
					).
						SizeFull(),
				).
				OnMouseDown(
					gpui.MouseButtonLeft,
					cx.Listener(func(this *PaintingViewer, ev *gpui.MouseDownEvent, _ *gpui.Window, _ *gpui.Context[PaintingViewer]) {
						this.painting = true
						this.start = ev.Position
						path := []gpui.Point[gpui.Pixels]{ev.Position}
						this.lines = append(this.lines, path)
					}),
				).
				OnMouseMove(cx.Listener(func(this *PaintingViewer, ev *gpui.MouseMoveEvent, _ *gpui.Window, cx *gpui.Context[PaintingViewer]) {
					if !this.painting {
						return
					}

					isShifted := ev.Modifiers.Shift
					pos := ev.Position
					if isShifted {
						dx := pos.X - this.start.X
						dy := pos.Y - this.start.Y
						if dx.Abs() > dy.Abs() {
							pos.Y = this.start.Y
						} else {
							pos.X = this.start.X
						}
					}

					if n := len(this.lines); n > 0 {
						this.lines[n-1] = append(this.lines[n-1], pos)
					}

					cx.Notify()
				})).
				OnMouseUp(
					gpui.MouseButtonLeft,
					cx.Listener(func(this *PaintingViewer, _ *gpui.MouseUpEvent, _ *gpui.Window, _ *gpui.Context[PaintingViewer]) {
						this.painting = false
					}),
				),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		options := gpui.DefaultWindowOptions()
		options.Focus = true
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[PaintingViewer] {
			return cx.New(func(cx *gpui.Context[PaintingViewer]) *PaintingViewer { return NewPaintingViewer(window, cx) })
		}); err != nil {
			panic(err)
		}
		cx.OnWindowClosed(func(cx *gpui.App, _ gpui.WindowID) {
			cx.Quit()
		}).Detach()
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
