package geom

import "testing"

func TestCubicBezier_CubicToQuadratics(t *testing.T) {
	quadratic := QuadraticBezierSegment{From: NewPoint(1.0, 2.0), Ctrl: NewPoint(10.0, 5.0), To: NewPoint(0.0, 1.0)}

	count := 0
	cubic0 := quadratic.ToCubic()
	if n := cubic0.NumQuadratics(0.0001); n != 1 {
		t.Fatalf("num_quadratics %d, want 1", n)
	}
	cubic0.ForEachQuadraticBezier(0.0001, func(c *QuadraticBezierSegment) {
		if count != 0 {
			t.Fatalf("count %d", count)
		}
		if !c.From.ApproxEq(quadratic.From) || !c.Ctrl.ApproxEq(quadratic.Ctrl) || !c.To.ApproxEq(quadratic.To) {
			t.Fatalf("quadratic %+v, want %+v", *c, quadratic)
		}
		count++
	})

	cubic := CubicBezierSegment{From: NewPoint(1.0, 1.0), Ctrl1: NewPoint(10.0, 2.0), Ctrl2: NewPoint(1.0, 3.0), To: NewPoint(10.0, 4.0)}

	prev := cubic.From
	count = 0
	cubic.ForEachQuadraticBezier(0.01, func(c *QuadraticBezierSegment) {
		if !c.From.ApproxEq(prev) {
			t.Fatalf("from %v, prev %v", c.From, prev)
		}
		prev = c.To
		count++
	})
	if !prev.ApproxEq(cubic.To) {
		t.Fatalf("end %v, want %v", prev, cubic.To)
	}
	if count >= 10 || count <= 4 {
		t.Fatalf("count %d", count)
	}
}
