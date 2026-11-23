package lyon

import (
	"math"
	"slices"
	"testing"

	"github.com/megascan/gpui-go/internal/lyon/geom"
)

func assertEvents(t *testing.T, path Path, want []PathEvent) {
	t.Helper()
	got := slices.Collect(path.Iter())
	if !slices.Equal(got, want) {
		t.Fatalf("events:\n got %+v\nwant %+v", got, want)
	}
}

func TestBuilder_SvgBuilderLineToAfterClose(t *testing.T) {
	p := PathSvgBuilder()
	p.LineTo(NewPoint(1.0, 0.0))
	p.Close()
	p.LineTo(NewPoint(2.0, 0.0))

	path := p.Build()
	assertEvents(t, path, []PathEvent{
		{Kind: EventBegin, At: NewPoint(1.0, 0.0)},
		{Kind: EventEnd, Last: NewPoint(1.0, 0.0), First: NewPoint(1.0, 0.0), Close: true},
		{Kind: EventBegin, At: NewPoint(1.0, 0.0)},
		{Kind: EventLine, From: NewPoint(1.0, 0.0), To: NewPoint(2.0, 0.0)},
		{Kind: EventEnd, Last: NewPoint(2.0, 0.0), First: NewPoint(1.0, 0.0), Close: false},
	})
}

func TestBuilder_SvgBuilderRelativeCurves(t *testing.T) {
	p := PathSvgBuilder()
	p.MoveTo(NewPoint(0.0, 0.0))
	p.RelativeQuadraticBezierTo(NewVector(0., 100.), NewVector(-100., 100.))
	p.RelativeLineTo(NewVector(-50., 0.))

	path := p.Build()
	assertEvents(t, path, []PathEvent{
		{Kind: EventBegin, At: NewPoint(0.0, 0.0)},
		{Kind: EventQuadratic, From: NewPoint(0.0, 0.0), Ctrl: NewPoint(0.0, 100.0), To: NewPoint(-100., 100.)},
		{Kind: EventLine, From: NewPoint(-100.0, 100.0), To: NewPoint(-150., 100.)},
		{Kind: EventEnd, First: NewPoint(0.0, 0.0), Last: NewPoint(-150., 100.), Close: false},
	})
}

func TestBuilder_SvgBuilderArcToUpdatePosition(t *testing.T) {
	p := PathSvgBuilder()
	p.MoveTo(NewPoint(0.0, 0.0))
	if p.CurrentPosition() != NewPoint(0.0, 0.0) {
		t.Fatalf("current position %v", p.CurrentPosition())
	}
	p.ArcTo(NewVector(100., 100.), geom.AngleDegrees(0.), ArcFlags{}, NewPoint(0.0, 100.0))
	if p.CurrentPosition() == NewPoint(0.0, 0.0) {
		t.Fatalf("current position was not updated")
	}
}

func TestBuilder_Issue650(t *testing.T) {
	builder := NewBuilder().WithSvg()
	builder.Arc(NewPoint(0.0, 0.0), NewVector(50.0, 50.0), geom.AngleRadians(math.Pi), geom.AngleRadians(0.0))
	builder.Build()
}

func TestBuilder_StraightLineArc(t *testing.T) {
	p := PathSvgBuilder()
	p.MoveTo(NewPoint(100.0, 0.0))
	p.ArcTo(NewVector(100., 100.), geom.AngleDegrees(0.), ArcFlags{}, NewPoint(100.0, 0.0))
}
