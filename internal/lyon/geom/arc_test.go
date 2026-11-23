package geom

import "testing"

func TestArc_FromSvgArc(t *testing.T) {
	flags := ArcFlags{LargeArc: false, Sweep: false}

	doTestEndpoints := func(svgArc *SvgArc) {
		eps := NewPoint(0.01, 0.01)
		arc := svgArc.ToArc()
		if !arc.From().ApproxEqEps(svgArc.From, eps) {
			t.Fatalf("unexpected arc.from: %v == %v, flags: %+v", arc.From(), svgArc.From, svgArc.Flags)
		}
		if !arc.To().ApproxEqEps(svgArc.To, eps) {
			t.Fatalf("unexpected arc.from: %v == %v, flags: %+v", arc.To(), svgArc.To, svgArc.Flags)
		}
	}

	testEndpoints := func(svgArc SvgArc) {
		for _, f := range []ArcFlags{{false, false}, {true, false}, {false, true}, {true, true}} {
			a := svgArc
			a.Flags = f
			doTestEndpoints(&a)
		}
	}

	testEndpoints(SvgArc{From: NewPoint(0.0, -10.0), To: NewPoint(10.0, 0.0), Radii: NewVector(10.0, 10.0), XRotation: AngleRadians(0.0), Flags: flags})
	testEndpoints(SvgArc{From: NewPoint(0.0, -10.0), To: NewPoint(10.0, 0.0), Radii: NewVector(100.0, 10.0), XRotation: AngleRadians(0.0), Flags: flags})
	testEndpoints(SvgArc{From: NewPoint(0.0, -10.0), To: NewPoint(10.0, 0.0), Radii: NewVector(10.0, 30.0), XRotation: AngleRadians(1.0), Flags: flags})
	testEndpoints(SvgArc{From: NewPoint(5.0, -10.0), To: NewPoint(5.0, 5.0), Radii: NewVector(10.0, 30.0), XRotation: AngleRadians(-2.0), Flags: flags})
	testEndpoints(SvgArc{From: NewPoint(0.0, 0.0), To: NewPoint(80.0, 60.0), Radii: NewVector(40.0, 40.0), XRotation: AngleRadians(0.0), Flags: flags})
}

func TestArc_ToQuadraticsAndCubics(t *testing.T) {
	doTest := func(arc *Arc, expectedQuadraticCount, expectedCubicCount uint32) {
		last := arc.To()
		{
			prev := arc.From()
			var count uint32
			arc.ForEachQuadraticBezier(func(c *QuadraticBezierSegment) {
				if !c.From.ApproxEq(prev) {
					t.Fatalf("quadratic from %v, prev %v", c.From, prev)
				}
				prev = c.To
				count++
			})
			if !prev.ApproxEq(last) {
				t.Fatalf("quadratic end %v, want %v", prev, last)
			}
			if count != expectedQuadraticCount {
				t.Fatalf("quadratic count %d, want %d", count, expectedQuadraticCount)
			}
		}
		{
			prev := arc.From()
			var count uint32
			arc.ForEachCubicBezier(func(c *CubicBezierSegment) {
				if !c.From.ApproxEq(prev) {
					t.Fatalf("cubic from %v, prev %v", c.From, prev)
				}
				prev = c.To
				count++
			})
			if !prev.ApproxEq(last) {
				t.Fatalf("cubic end %v, want %v", prev, last)
			}
			if count != expectedCubicCount {
				t.Fatalf("cubic count %d, want %d", count, expectedCubicCount)
			}
		}
	}

	doTest(&Arc{Center: NewPoint(2.0, 3.0), Radii: NewVector(10.0, 3.0), StartAngle: AngleRadians(0.1), SweepAngle: AngleRadians(3.0), XRotation: AngleRadians(0.5)}, 4, 2)
	doTest(&Arc{Center: NewPoint(4.0, 5.0), Radii: NewVector(3.0, 5.0), StartAngle: AngleRadians(2.0), SweepAngle: AngleRadians(-3.0), XRotation: AngleRadians(1.3)}, 4, 2)
	doTest(&Arc{Center: NewPoint(0.0, 0.0), Radii: NewVector(100.0, 0.01), StartAngle: AngleRadians(-1.0), SweepAngle: AngleRadians(0.1), XRotation: AngleRadians(0.3)}, 1, 1)
	doTest(&Arc{Center: NewPoint(0.0, 0.0), Radii: NewVector(1.0, 1.0), StartAngle: AngleRadians(3.0), SweepAngle: AngleRadians(-0.1), XRotation: AngleRadians(-0.3)}, 1, 1)
}
