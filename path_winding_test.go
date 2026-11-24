package gpui

import (
	"math"
	"testing"
)

func TestPathWinding_RectangleIsNineTiles(t *testing.T) {
	path := pathWindingTestSquare(t, 100., 40., 900., 600.)
	if got := len(path.bin(1.25).tiles()); got != 9 {
		t.Fatalf("got %d tiles, want 9", got)
	}
}

func TestPathWinding_HugePathPaintsOnlyVisibleTiles(t *testing.T) {
	path := pathWindingTestSquare(t, -20000., -20000., 20000., 20000.)
	tileCount := len(path.bin(1.5).tiles())
	if !(tileCount < 10_000) {
		t.Fatalf("%d tiles", tileCount)
	}

	mask := Bounds[Pixels]{
		Origin: NewPoint(Px(300.), Px(200.)),
		Size:   NewSize(Px(120.), Px(90.)),
	}
	var scene Scene
	scene.InsertPrimitive(NewPaintedPath(
		path,
		1.5,
		ContentMask[Pixels]{Bounds: mask},
		Black().ToBackground(),
	))
	scene.Finish()
	painted := 0
	for _, path := range scene.Paths {
		painted += path.TileCount()
	}
	if !(painted > 0 && painted < 20) {
		t.Fatalf("%d tiles", painted)
	}
}

func TestPathWinding_OverflowingPathsAreSkipped(t *testing.T) {
	radius := float32(math.MaxFloat32) / 4.0
	path := pathWindingTestSquare(t, -radius, -radius, radius, radius)
	if len(path.bin(1.0).tiles()) == 0 {
		t.Fatal("expected tiles at scale 1")
	}
	if len(path.bin(4.0).tiles()) != 0 {
		t.Fatal("expected no tiles at scale 4")
	}
	radius = float32(math.MaxFloat32) * 0.75
	path = pathWindingTestSquare(t, -radius, -radius, radius, radius)
	if len(path.bin(1.0).tiles()) != 0 {
		t.Fatal("expected no tiles")
	}
}

func TestPathWinding_NonFinitePointFailsPath(t *testing.T) {
	var o outline
	o.moveTo(NewPoint[float32](0., 0.))
	o.lineTo(NewPoint[float32](10., 0.))
	o.lineTo(NewPoint[float32](10., 10.))
	o.moveTo(NewPoint[float32](20., 0.))
	o.lineTo(NewPoint(float32(math.NaN()), 5.))
	o.lineTo(NewPoint[float32](30., 10.))
	if _, err := o.intoCurves(); err == nil {
		t.Fatal("expected an error")
	}
}

func TestPathWinding_EndpointsBalance(t *testing.T) {
	curvedFill := PathBuilderFill()
	curvedFill.MoveTo(NewPoint(Px(450.3), Px(280.7)))
	curvedFill.CurveTo(NewPoint(Px(530.3), Px(230.7)), NewPoint(Px(450.3), Px(230.7)))
	curvedFill.ArcTo(
		NewPoint(Px(30.), Px(30.)),
		Px(0.),
		false,
		false,
		NewPoint(Px(590.3), Px(230.7)),
	)
	curvedFill.Close()

	stroke := PathBuilderStroke(Px(2.))
	stroke.MoveTo(NewPoint(Px(80.3), Px(500.7)))
	stroke.CubicBezierTo(
		NewPoint(Px(900.), Px(250.)),
		NewPoint(Px(300.), Px(50.)),
		NewPoint(Px(700.), Px(800.)),
	)

	dashed := PathBuilderStroke(Px(3.)).DashArray([]Pixels{Px(9.), Px(4.)})
	dashed.MoveTo(NewPoint(Px(50.3), Px(10.7)))
	dashed.ArcTo(
		NewPoint(Px(40.), Px(40.)),
		Px(0.),
		false,
		true,
		NewPoint(Px(90.3), Px(50.7)),
	)

	for _, tc := range []struct {
		name    string
		builder *PathBuilder
	}{
		{"curved fill", curvedFill},
		{"stroke", stroke},
		{"dashed stroke", dashed},
	} {
		curves, _, err := tc.builder.outline()
		if err != nil {
			t.Fatal(err)
		}
		type key struct{ x, y uint32 }
		balance := map[key]int32{}
		for i := range curves {
			for _, endpoint := range curves[i].endpoints() {
				balance[key{math.Float32bits(endpoint.point.X), math.Float32bits(endpoint.point.Y)}] += endpoint.step
			}
		}
		if len(balance) == 0 {
			t.Fatalf("%s produced no curves", tc.name)
		}
		for k, sum := range balance {
			if sum != 0 {
				t.Fatalf("%s: endpoints at (%v, %v) do not balance", tc.name, math.Float32frombits(k.x), math.Float32frombits(k.y))
			}
		}
	}
}

func pathWindingTestSquare(t *testing.T, left, top, right, bottom float32) Path {
	t.Helper()
	builder := PathBuilderFill()
	corners := [][2]float32{{left, top}, {right, top}, {right, bottom}, {left, bottom}}
	points := make([]Point[Pixels], len(corners))
	for i, c := range corners {
		points[i] = NewPoint(Px(c[0]), Px(c[1]))
	}
	builder.AddPolygon(points, true)
	path, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	return path
}
