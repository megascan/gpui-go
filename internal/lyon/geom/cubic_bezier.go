package geom

// CubicBezierSegment is lyon_geom's CubicBezierSegment<f32>.
type CubicBezierSegment struct {
	From  Point
	Ctrl1 Point
	Ctrl2 Point
	To    Point
}

// Sample is CubicBezierSegment::sample.
func (c *CubicBezierSegment) Sample(t float32) Point {
	t2 := t * t
	t3 := t2 * t
	oneT := 1 - t
	oneT2 := oneT * oneT
	oneT3 := oneT2 * oneT

	return c.From.Mul(oneT3).
		Add(c.Ctrl1.ToVector().Mul(3).Mul(oneT2).Mul(t)).
		Add(c.Ctrl2.ToVector().Mul(3).Mul(oneT).Mul(t2)).
		Add(c.To.ToVector().Mul(t3))
}

// X is CubicBezierSegment::x.
func (c *CubicBezierSegment) X(t float32) float32 {
	t2 := t * t
	t3 := t2 * t
	oneT := 1 - t
	oneT2 := oneT * oneT
	oneT3 := oneT2 * oneT

	return float32(c.From.X*oneT3) +
		float32(float32(float32(c.Ctrl1.X*3)*oneT2)*t) +
		float32(float32(float32(c.Ctrl2.X*3)*oneT)*t2) +
		float32(c.To.X*t3)
}

// Y is CubicBezierSegment::y.
func (c *CubicBezierSegment) Y(t float32) float32 {
	t2 := t * t
	t3 := t2 * t
	oneT := 1 - t
	oneT2 := oneT * oneT
	oneT3 := oneT2 * oneT

	return float32(c.From.Y*oneT3) +
		float32(float32(float32(c.Ctrl1.Y*3)*oneT2)*t) +
		float32(float32(float32(c.Ctrl2.Y*3)*oneT)*t2) +
		float32(c.To.Y*t3)
}

// SplitRange is CubicBezierSegment::split_range.
func (c *CubicBezierSegment) SplitRange(tRange Range) CubicBezierSegment {
	t0, t1 := tRange.Start, tRange.End
	from := c.Sample(t0)
	to := c.Sample(t1)

	d := QuadraticBezierSegment{
		From: c.Ctrl1.Sub(c.From).ToPoint(),
		Ctrl: c.Ctrl2.Sub(c.Ctrl1).ToPoint(),
		To:   c.To.Sub(c.Ctrl2).ToPoint(),
	}

	dt := t1 - t0
	ctrl1 := from.Add(d.Sample(t0).ToVector().Mul(dt))
	ctrl2 := to.SubVector(d.Sample(t1).ToVector().Mul(dt))

	return CubicBezierSegment{From: from, Ctrl1: ctrl1, Ctrl2: ctrl2, To: to}
}

// Split is CubicBezierSegment::split.
func (c *CubicBezierSegment) Split(t float32) (CubicBezierSegment, CubicBezierSegment) {
	ctrl1a := c.From.Add(c.Ctrl1.Sub(c.From).Mul(t))
	ctrl2a := c.Ctrl1.Add(c.Ctrl2.Sub(c.Ctrl1).Mul(t))
	ctrl1aa := ctrl1a.Add(ctrl2a.Sub(ctrl1a).Mul(t))
	ctrl3a := c.Ctrl2.Add(c.To.Sub(c.Ctrl2).Mul(t))
	ctrl2aa := ctrl2a.Add(ctrl3a.Sub(ctrl2a).Mul(t))
	ctrl1aaa := ctrl1aa.Add(ctrl2aa.Sub(ctrl1aa).Mul(t))

	return CubicBezierSegment{From: c.From, Ctrl1: ctrl1a, Ctrl2: ctrl1aa, To: ctrl1aaa},
		CubicBezierSegment{From: ctrl1aaa, Ctrl1: ctrl2aa, Ctrl2: ctrl3a, To: c.To}
}

// ToQuadratic is CubicBezierSegment::to_quadratic.
func (c *CubicBezierSegment) ToQuadratic() QuadraticBezierSegment {
	c1 := c.Ctrl1.Mul(3).Sub(c.From).Mul(0.5)
	c2 := c.Ctrl2.Mul(3).Sub(c.To).Mul(0.5)
	return QuadraticBezierSegment{From: c.From, Ctrl: c1.Add(c2).Mul(0.5).ToPoint(), To: c.To}
}

// ToQuadraticError is CubicBezierSegment::to_quadratic_error.
func (c *CubicBezierSegment) ToQuadraticError() float32 {
	return float32(sqrt(3)/36) * c.To.Sub(c.Ctrl2.Mul(3)).Add(c.Ctrl1.Mul(3).Sub(c.From)).Length()
}

// NumQuadratics is CubicBezierSegment::num_quadratics.
func (c *CubicBezierSegment) NumQuadratics(tolerance float32) uint32 {
	return c.numQuadraticsImpl(tolerance)
}

var numQuadraticsLUT = [16]float32{
	1.0, 64.0, 729.0, 4096.0, 15625.0, 46656.0, 117649.0, 262144.0, 531441.0, 1000000.0,
	1771561.0, 2985984.0, 4826809.0, 7529536.0, 11390625.0, 16777216.0,
}

func (c *CubicBezierSegment) numQuadraticsImpl(tolerance float32) uint32 {
	x := c.From.X - float32(3*c.Ctrl1.X) + float32(3*c.Ctrl2.X) - c.To.X
	y := c.From.Y - float32(3*c.Ctrl1.Y) + float32(3*c.Ctrl2.Y) - c.To.Y

	err := (float32(x*x) + float32(y*y)) / float32(float32(432.0*tolerance)*tolerance)

	if err <= 16777216.0 {
		for i, v := range numQuadraticsLUT {
			if err <= v {
				return uint32(i + 1)
			}
		}
	}

	r := Max(ceil(float32(powf(err, 1.0/6.0))), 1)
	return asU32(r, 1)
}

// ForEachQuadraticBezier is CubicBezierSegment::for_each_quadratic_bezier.
func (c *CubicBezierSegment) ForEachQuadraticBezier(tolerance float32, cb func(*QuadraticBezierSegment)) {
	c.ForEachQuadraticBezierWithT(tolerance, func(quad *QuadraticBezierSegment, _ Range) { cb(quad) })
}

// ForEachQuadraticBezierWithT is CubicBezierSegment::for_each_quadratic_bezier_with_t.
func (c *CubicBezierSegment) ForEachQuadraticBezierWithT(tolerance float32, cb func(*QuadraticBezierSegment, Range)) {
	n := max(c.NumQuadratics(tolerance), 1)
	if n == 1 {
		q := c.ToQuadratic()
		cb(&q, Range{0, 1})
		return
	}

	from := c.From.ToVector()
	ctrl1 := c.Ctrl1.ToVector()
	ctrl2 := c.Ctrl2.ToVector()
	to := c.To.ToVector()
	a := to.Sub(from).Add(ctrl1.Sub(ctrl2).Mul(3))
	b := from.Sub(ctrl1.Mul(2)).Add(ctrl2).Mul(3)
	cc := ctrl1.Sub(from).Mul(3)

	h := 1 / float32(n)
	quarterH := float32(h * 0.25)
	var t0 float32
	p0 := c.From
	d0 := cc
	for i := uint32(1); i < n; i++ {
		t1 := float32(h * float32(i))
		p1 := c.From.Add(cc.Add(b.Add(a.Mul(t1)).Mul(t1)).Mul(t1))
		d1 := cc.Add(b.Mul(2).Add(a.Mul(float32(3 * t1))).Mul(t1))

		ctrl := p0.Lerp(p1, 0.5).Add(d0.Sub(d1).Mul(quarterH))
		cb(&QuadraticBezierSegment{From: p0, Ctrl: ctrl, To: p1}, Range{t0, t1})

		t0 = t1
		p0 = p1
		d0 = d1
	}

	d1 := to.Sub(ctrl2).Mul(3)
	ctrl := p0.Lerp(c.To, 0.5).Add(d0.Sub(d1).Mul(quarterH))
	cb(&QuadraticBezierSegment{From: p0, Ctrl: ctrl, To: c.To}, Range{t0, 1})
}
