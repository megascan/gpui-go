package lyon

import (
	"slices"
	"testing"

	"github.com/megascan/gpui-go/internal/lyon/geom"
)

func TestPath_BuilderEventsAndTransform(t *testing.T) {
	b := NewBuilder()
	b.Begin(NewPoint(0, 0))
	b.LineTo(NewPoint(1, 0))
	b.QuadraticBezierTo(NewPoint(2, 0), NewPoint(2, 1))
	b.CubicBezierTo(NewPoint(3, 0), NewPoint(3, 1), NewPoint(3, 2))
	b.End(true)
	b.AddPolygon(Polygon[Point]{Points: []Point{NewPoint(5, 5), NewPoint(6, 6)}, Closed: false})
	path := b.Build()

	want := []PathEvent{
		{Kind: EventBegin, At: NewPoint(0, 0)},
		{Kind: EventLine, From: NewPoint(0, 0), To: NewPoint(1, 0)},
		{Kind: EventQuadratic, From: NewPoint(1, 0), Ctrl: NewPoint(2, 0), To: NewPoint(2, 1)},
		{Kind: EventCubic, From: NewPoint(2, 1), Ctrl1: NewPoint(3, 0), Ctrl2: NewPoint(3, 1), To: NewPoint(3, 2)},
		{Kind: EventEnd, Last: NewPoint(3, 2), First: NewPoint(0, 0), Close: true},
		{Kind: EventBegin, At: NewPoint(5, 5)},
		{Kind: EventLine, From: NewPoint(5, 5), To: NewPoint(6, 6)},
		{Kind: EventEnd, Last: NewPoint(6, 6), First: NewPoint(5, 5), Close: false},
	}
	assertEvents(t, path, want)

	round := PathFromEvents(path.Iter())
	assertEvents(t, round, want)

	tr := geom.TransformTranslation(10, 20).ThenScale(2, 2)
	moved := path.Transformed(tr)
	var wantMoved []PathEvent
	for _, e := range want {
		wantMoved = append(wantMoved, PathEventTransformed(e, tr))
	}
	assertEvents(t, moved, wantMoved)
	assertEvents(t, path, want)

	if !slices.Equal(
		[]bool{want[0].IsEdge(), want[1].IsEdge(), want[4].IsEdge(), want[7].IsEdge()},
		[]bool{false, true, true, false},
	) {
		t.Fatal("IsEdge")
	}
	if want[4].FromEndpoint() != NewPoint(3, 2) || want[4].ToEndpoint() != NewPoint(0, 0) {
		t.Fatal("FromEndpoint/ToEndpoint")
	}
}

func TestPath_ValidatorPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != "edge operation is made before begin()" {
			t.Fatalf("panic %v", r)
		}
	}()
	NewBuilder().LineTo(NewPoint(1, 1))
}
