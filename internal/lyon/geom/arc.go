package geom

// Arc is lyon_geom's Arc<f32>, an elliptic arc curve segment.
type Arc struct {
	Center     Point
	Radii      Vector
	StartAngle Angle
	SweepAngle Angle
	XRotation  Angle
}

// SvgArc is lyon_geom's SvgArc<f32>, an elliptic arc in SVG end-point notation.
type SvgArc struct {
	From      Point
	To        Point
	Radii     Vector
	XRotation Angle
	Flags     ArcFlags
}

// ArcFlags is lyon_geom's ArcFlags, the SVG large-arc and sweep flags.
type ArcFlags struct {
	LargeArc bool
	Sweep    bool
}

// ArcCircle is Arc::circle.
func ArcCircle(center Point, radius float32) Arc {
	return Arc{
		Center:     center,
		Radii:      NewVector(radius, radius),
		StartAngle: AngleZero(),
		SweepAngle: AngleTwoPi(),
		XRotation:  AngleZero(),
	}
}

// ArcFromSvgArc is Arc::from_svg_arc.
func ArcFromSvgArc(arc *SvgArc) Arc {
	if arc.IsStraightLine() {
		panic("assertion failed: !arc.is_straight_line()")
	}

	rx := abs(arc.Radii.X)
	ry := abs(arc.Radii.Y)

	xr := rem(arc.XRotation.Get(), 2*pi)
	cosPhi := cos(xr)
	sinPhi := sin(xr)
	hdX := (arc.From.X - arc.To.X) / 2
	hdY := (arc.From.Y - arc.To.Y) / 2
	hsX := (arc.From.X + arc.To.X) / 2
	hsY := (arc.From.Y + arc.To.Y) / 2

	p := Point{
		float32(cosPhi*hdX) + float32(sinPhi*hdY),
		float32(-sinPhi*hdX) + float32(cosPhi*hdY),
	}

	rf := float32(float32(p.X*p.X)/float32(rx*rx)) + float32(float32(p.Y*p.Y)/float32(ry*ry))
	if rf > 1 {
		scale := sqrt(rf)
		rx *= scale
		ry *= scale
	}

	rxry := rx * ry
	rxpy := rx * p.Y
	rypx := ry * p.X
	sumOfSq := float32(rxpy*rxpy) + float32(rypx*rypx)

	signCoe := float32(1)
	if arc.Flags.LargeArc == arc.Flags.Sweep {
		signCoe = -1
	}
	coe := signCoe * sqrt(abs((float32(rxry*rxry)-sumOfSq)/sumOfSq))
	transformedCx := float32(coe*rxpy) / ry
	transformedCy := float32(-coe*rypx) / rx

	center := NewPoint(
		float32(cosPhi*transformedCx)-float32(sinPhi*transformedCy)+hsX,
		float32(sinPhi*transformedCx)+float32(cosPhi*transformedCy)+hsY,
	)

	startV := NewVector((p.X-transformedCx)/rx, (p.Y-transformedCy)/ry)
	endV := NewVector((-p.X-transformedCx)/rx, (-p.Y-transformedCy)/ry)

	twoPi := 2 * pi

	startAngle := startV.AngleFromXAxis()

	sweepAngle := rem(endV.AngleFromXAxis().Sub(startAngle).Radians, twoPi)

	if arc.Flags.Sweep && sweepAngle < 0 {
		sweepAngle += twoPi
	} else if !arc.Flags.Sweep && sweepAngle > 0 {
		sweepAngle -= twoPi
	}

	return Arc{
		Center:     center,
		Radii:      NewVector(rx, ry),
		StartAngle: startAngle,
		SweepAngle: AngleRadians(sweepAngle),
		XRotation:  arc.XRotation,
	}
}

// ToSvgArc is Arc::to_svg_arc.
func (a *Arc) ToSvgArc() SvgArc {
	from := a.Sample(0)
	to := a.Sample(1)
	flags := ArcFlags{
		Sweep:    a.SweepAngle.Get() >= 0,
		LargeArc: abs(a.SweepAngle.Get()) >= pi,
	}
	return SvgArc{From: from, To: to, Radii: a.Radii, XRotation: a.XRotation, Flags: flags}
}

// ForEachQuadraticBezier is Arc::for_each_quadratic_bezier.
func (a *Arc) ForEachQuadraticBezier(cb func(*QuadraticBezierSegment)) {
	arcToQuadraticBeziersWithT(a, func(curve *QuadraticBezierSegment, _ Range) { cb(curve) })
}

// ForEachQuadraticBezierWithT is Arc::for_each_quadratic_bezier_with_t.
func (a *Arc) ForEachQuadraticBezierWithT(cb func(*QuadraticBezierSegment, Range)) {
	arcToQuadraticBeziersWithT(a, cb)
}

// ForEachCubicBezier is Arc::for_each_cubic_bezier.
func (a *Arc) ForEachCubicBezier(cb func(*CubicBezierSegment)) {
	arcToCubicBeziers(a, cb)
}

// Sample is Arc::sample.
func (a *Arc) Sample(t float32) Point {
	angle := a.GetAngle(t)
	return a.Center.Add(sampleEllipse(a.Radii, a.XRotation, angle).ToVector())
}

// X is Arc::x.
func (a *Arc) X(t float32) float32 { return a.Sample(t).X }

// Y is Arc::y.
func (a *Arc) Y(t float32) float32 { return a.Sample(t).Y }

// SampleTangent is Arc::sample_tangent.
func (a *Arc) SampleTangent(t float32) Vector { return a.tangentAtAngle(a.GetAngle(t)) }

// GetAngle is Arc::get_angle.
func (a *Arc) GetAngle(t float32) Angle {
	return a.StartAngle.Add(AngleRadians(float32(a.SweepAngle.Get() * t)))
}

// EndAngle is Arc::end_angle.
func (a *Arc) EndAngle() Angle { return a.StartAngle.Add(a.SweepAngle) }

// From is Arc::from.
func (a *Arc) From() Point { return a.Sample(0) }

// To is Arc::to.
func (a *Arc) To() Point { return a.Sample(1) }

// Flip is Arc::flip.
func (a *Arc) Flip() Arc {
	arc := *a
	arc.StartAngle = arc.StartAngle.Add(a.SweepAngle)
	arc.SweepAngle = AngleRadians(-a.SweepAngle.Get())
	return arc
}

func (a *Arc) tangentAtAngle(angle Angle) Vector {
	r := angle.Get()
	return NewRotation(a.XRotation).TransformVector(NewVector(
		float32(-a.Radii.X*sin(r)),
		float32(a.Radii.Y*cos(r)),
	))
}

// ToArc is SvgArc::to_arc.
func (s *SvgArc) ToArc() Arc { return ArcFromSvgArc(s) }

// IsStraightLine is SvgArc::is_straight_line.
func (s *SvgArc) IsStraightLine() bool {
	return abs(s.Radii.X) <= Epsilon || abs(s.Radii.Y) <= Epsilon || s.From == s.To
}

// ForEachQuadraticBezier is SvgArc::for_each_quadratic_bezier.
func (s *SvgArc) ForEachQuadraticBezier(cb func(*QuadraticBezierSegment)) {
	if s.IsStraightLine() {
		cb(&QuadraticBezierSegment{From: s.From, Ctrl: s.From, To: s.To})
		return
	}
	arc := ArcFromSvgArc(s)
	arc.ForEachQuadraticBezier(cb)
}

// ForEachQuadraticBezierWithT is SvgArc::for_each_quadratic_bezier_with_t.
func (s *SvgArc) ForEachQuadraticBezierWithT(cb func(*QuadraticBezierSegment, Range)) {
	if s.IsStraightLine() {
		cb(&QuadraticBezierSegment{From: s.From, Ctrl: s.From, To: s.To}, Range{0, 1})
		return
	}
	arc := ArcFromSvgArc(s)
	arc.ForEachQuadraticBezierWithT(cb)
}

// ForEachCubicBezier is SvgArc::for_each_cubic_bezier.
func (s *SvgArc) ForEachCubicBezier(cb func(*CubicBezierSegment)) {
	if s.IsStraightLine() {
		cb(&CubicBezierSegment{From: s.From, Ctrl1: s.From, Ctrl2: s.To, To: s.To})
		return
	}
	arc := ArcFromSvgArc(s)
	arc.ForEachCubicBezier(cb)
}

func arcToQuadraticBeziersWithT(arc *Arc, callback func(*QuadraticBezierSegment, Range)) {
	sign := signum(arc.SweepAngle.Get())
	sweepAngle := Min(abs(arc.SweepAngle.Get()), pi*2)

	nSteps := ceil(sweepAngle / fracPi4)
	step := AngleRadians(float32(sweepAngle/nSteps) * sign)

	var t0 float32
	dt := 1 / nSteps

	alpha := tan(step.Radians * 0.5)

	n := asI32(nSteps)
	for i := int32(0); i < n; i++ {
		a1 := arc.StartAngle.Add(step.Mul(float32(i)))
		a2 := arc.StartAngle.Add(step.Mul(float32(i + 1)))

		v1 := sampleEllipse(arc.Radii, arc.XRotation, a1).ToVector()
		v2 := sampleEllipse(arc.Radii, arc.XRotation, a2).ToVector()
		from := arc.Center.Add(v1)
		to := arc.Center.Add(v2)

		ctrl := from.Add(arc.tangentAtAngle(a1).Mul(alpha))

		t1 := t0 + dt
		if i+1 == n {
			t1 = 1
		}

		callback(&QuadraticBezierSegment{From: from, Ctrl: ctrl, To: to}, Range{t0, t1})
		t0 = t1
	}
}

func arcToCubicBeziers(arc *Arc, callback func(*CubicBezierSegment)) {
	sign := signum(arc.SweepAngle.Get())
	sweepAngle := Min(abs(arc.SweepAngle.Get()), pi*2)

	nSteps := ceil(sweepAngle / fracPi2)
	step := AngleRadians(float32(sweepAngle/nSteps) * sign)

	n := asI32(nSteps)
	for i := int32(0); i < n; i++ {
		a1 := arc.StartAngle.Add(step.Mul(float32(i)))
		a2 := arc.StartAngle.Add(step.Mul(float32(i + 1)))

		v1 := sampleEllipse(arc.Radii, arc.XRotation, a1).ToVector()
		v2 := sampleEllipse(arc.Radii, arc.XRotation, a2).ToVector()
		from := arc.Center.Add(v1)
		to := arc.Center.Add(v2)

		deltaA := a2.Sub(a1)
		tanDa := tan(deltaA.Get() * 0.5)
		alphaSqrt := sqrt(4 + float32(float32(3*tanDa)*tanDa))
		alpha := float32(sin(deltaA.Get())*(alphaSqrt-1)) / 3
		ctrl1 := from.Add(arc.tangentAtAngle(a1).Mul(alpha))
		ctrl2 := to.SubVector(arc.tangentAtAngle(a2).Mul(alpha))

		callback(&CubicBezierSegment{From: from, Ctrl1: ctrl1, Ctrl2: ctrl2, To: to})
	}
}

func sampleEllipse(radii Vector, xRotation Angle, angle Angle) Point {
	return NewRotation(xRotation).TransformPoint(NewPoint(
		float32(radii.X*cos(angle.Get())),
		float32(radii.Y*sin(angle.Get())),
	))
}
