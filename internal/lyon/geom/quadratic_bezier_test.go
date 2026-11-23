package geom

import "testing"

func TestQuadraticBezier_YMaximumTForSimpleSegment(t *testing.T) {
	a := QuadraticBezierSegment{From: NewPoint(0.0, 0.0), Ctrl: NewPoint(1.0, 1.0), To: NewPoint(2.0, 0.0)}
	if got := a.YMaximumT(); got != 0.5 {
		t.Fatalf("y_maximum_t %v", got)
	}
}

func TestQuadraticBezier_LocalYExtremumForSimpleSegment(t *testing.T) {
	a := QuadraticBezierSegment{From: NewPoint(0.0, 0.0), Ctrl: NewPoint(1.0, 1.0), To: NewPoint(2.0, 0.0)}
	got, ok := a.LocalYExtremumT()
	if !ok {
		panic("explicit panic")
	}
	if got != 0.5 {
		t.Fatalf("local_y_extremum_t %v", got)
	}
}

func TestQuadraticBezier_YMinimumTForSimpleSegment(t *testing.T) {
	a := QuadraticBezierSegment{From: NewPoint(0.0, 0.0), Ctrl: NewPoint(1.0, -1.0), To: NewPoint(2.0, 0.0)}
	if got := a.YMinimumT(); got != 0.5 {
		t.Fatalf("y_minimum_t %v", got)
	}
}

func TestQuadraticBezier_XMaximumTForSimpleSegment(t *testing.T) {
	a := QuadraticBezierSegment{From: NewPoint(0.0, 0.0), Ctrl: NewPoint(1.0, 1.0), To: NewPoint(0.0, 2.0)}
	if got := a.XMaximumT(); got != 0.5 {
		t.Fatalf("x_maximum_t %v", got)
	}
}

func TestQuadraticBezier_LocalXExtremumForSimpleSegment(t *testing.T) {
	a := QuadraticBezierSegment{From: NewPoint(0.0, 0.0), Ctrl: NewPoint(1.0, 1.0), To: NewPoint(0.0, 2.0)}
	got, ok := a.LocalXExtremumT()
	if !ok {
		panic("explicit panic")
	}
	if got != 0.5 {
		t.Fatalf("local_x_extremum_t %v", got)
	}
}

func TestQuadraticBezier_XMinimumTForSimpleSegment(t *testing.T) {
	a := QuadraticBezierSegment{From: NewPoint(2.0, 0.0), Ctrl: NewPoint(1.0, 1.0), To: NewPoint(2.0, 2.0)}
	if got := a.XMinimumT(); got != 0.5 {
		t.Fatalf("x_minimum_t %v", got)
	}
}

func TestQuadraticBezier_Derivatives(t *testing.T) {
	c1 := QuadraticBezierSegment{From: NewPoint(1.0, 1.0), Ctrl: NewPoint(2.0, 1.0), To: NewPoint(2.0, 2.0)}
	if c1.Dy(0.0) != 0.0 {
		t.Fatal("dy(0)")
	}
	if c1.Dx(1.0) != 0.0 {
		t.Fatal("dx(1)")
	}
	if c1.Dy(0.5) != c1.Dx(0.5) {
		t.Fatal("dy(0.5) != dx(0.5)")
	}
}

func TestQuadraticBezier_ForEachMonotonic(t *testing.T) {
	q := QuadraticBezierSegment{From: NewPoint(0, 0), Ctrl: NewPoint(4, 4), To: NewPoint(1, 0)}
	var pieces []QuadraticBezierSegment
	q.ForEachMonotonic(func(s *QuadraticBezierSegment) { pieces = append(pieces, *s) })
	if len(pieces) != 3 {
		t.Fatalf("pieces %d, want 3", len(pieces))
	}
	if pieces[0].From != q.From || pieces[2].To != q.To {
		t.Fatalf("endpoints %+v", pieces)
	}
	for i := 1; i < len(pieces); i++ {
		if pieces[i].From != pieces[i-1].To {
			t.Fatalf("gap at %d: %+v", i, pieces)
		}
	}
}
