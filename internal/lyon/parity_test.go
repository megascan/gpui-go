package lyon

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/internal/lyon/geom"
)

const rustGolden = `
B 412a6d2a,41a63a58
L 411239d2,41ad5614
Q 3e547ce0,41db3096 bfab543c,421d64dd
Q c0389c1e,424d3170 4062ed82,4281ccf6
Q 411f9dcc,429d0135 41af67da,42b11e7b
Q 42078061,42c53bc0 423b1d52,42ca9296
Q 426eba44,42cfe96d 4289379e,42c46f96
Q 42897a3f,42cc044e 42844dc2,42d18f62
Q 427e428b,42d71a76 426f1a98,42d75da3
Q 425ff2a2,42d7a0d0 4254dbb0,42d274c0
Q 4249c4c1,42cd48b1 42493d3f,42c5b4ba
C 40c947eb,4082b6a0 4306a660,429c1b08 41b88766,42a89222
C c2b1090b,42b5093d 402a6783,415cd40d 40c17ea2,3f93c7c2
Q 40c17ea2,3f93c7c2 4117572a,41caf4a8
E true
B 43302580,43909c4e
L 431907e1,43941edd
Q 4313c12e,439edc1f 43053830,43a04bc1
Q 42ed5e5f,43a1bb63 42cd614f,4399181e
Q 42ad643d,439074d9 429ba691,4382614b
Q 4289e8de,43689b77 428fefb9,4350af46
Q 4295f694,4338c316 42b0860a,4331e831
Q 42cb1580,432b0d4d 42ebecd6,4338f16c
E false
M 3f800000,3f800000 40b60000,3fd00000 40b60000,40100000
M 40b60000,40100000 40b60000,40180000 40b00000,40200000
M 40b00000,40200000 40aa0000,40280000 40a9ffff,40300000
M 40a9ffff,40300000 40a9ffff,40580000 41200000,40800000
`

func fmtPoint(p Point) string {
	return fmt.Sprintf("%08x,%08x", math.Float32bits(p.X), math.Float32bits(p.Y))
}

func TestParity_RustGoldenEvents(t *testing.T) {
	b := NewBuilder().WithSvg()
	b.MoveTo(NewPoint(10.0, 10.0))
	b.ArcTo(NewVector(30.0, 20.0), geom.AngleDegrees(15.0), ArcFlags{LargeArc: true, Sweep: false}, NewPoint(70.0, 33.3))
	b.RelativeArcTo(NewVector(5.0, 5.0), geom.AngleDegrees(0.0), ArcFlags{LargeArc: false, Sweep: true}, NewVector(-10.0, 7.0))
	b.CubicBezierTo(NewPoint(1.5, 2.25), NewPoint(100.125, -3.0), NewPoint(40.0, 41.0))
	b.SmoothCubicBezierTo(NewPoint(3.0, 9.0), NewPoint(0.3, 0.7))
	b.SmoothQuadraticBezierTo(NewPoint(11.0, 13.0))
	b.Close()
	b.MoveTo(NewPoint(200.0, 100.0))
	b.Arc(NewPoint(150.0, 100.0), NewVector(50.0, 25.0), geom.AngleRadians(4.5), geom.AngleRadians(0.7))
	tr := geom.TransformTranslation(3.5, -2.25).ThenScale(1.5, 1.5).ThenRotate(geom.AngleRadians(geom.ToRadians(33)))
	path := b.Build().Transformed(tr)

	var out strings.Builder
	for e := range path.Iter() {
		switch e.Kind {
		case EventBegin:
			fmt.Fprintf(&out, "B %s\n", fmtPoint(e.At))
		case EventLine:
			fmt.Fprintf(&out, "L %s\n", fmtPoint(e.To))
		case EventQuadratic:
			fmt.Fprintf(&out, "Q %s %s\n", fmtPoint(e.Ctrl), fmtPoint(e.To))
		case EventCubic:
			fmt.Fprintf(&out, "C %s %s %s\n", fmtPoint(e.Ctrl1), fmtPoint(e.Ctrl2), fmtPoint(e.To))
		case EventEnd:
			fmt.Fprintf(&out, "E %v\n", e.Close)
		}
	}
	c := geom.CubicBezierSegment{From: NewPoint(1.0, 1.0), Ctrl1: NewPoint(10.0, 2.0), Ctrl2: NewPoint(1.0, 3.0), To: NewPoint(10.0, 4.0)}
	c.ForEachQuadraticBezier(0.25, func(q *geom.QuadraticBezierSegment) {
		q.ForEachMonotonic(func(m *geom.QuadraticBezierSegment) {
			fmt.Fprintf(&out, "M %s %s %s\n", fmtPoint(m.From), fmtPoint(m.Ctrl), fmtPoint(m.To))
		})
	})

	want := strings.Split(strings.TrimSpace(rustGolden), "\n")
	got := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d:\n%s", len(got), len(want), out.String())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d: got %s, want %s", i, got[i], want[i])
		}
	}
}
