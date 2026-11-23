package geom

// QuadraticBezierSegment is lyon_geom's QuadraticBezierSegment<f32>.
type QuadraticBezierSegment struct {
	From Point
	Ctrl Point
	To   Point
}

// Sample is QuadraticBezierSegment::sample.
func (q *QuadraticBezierSegment) Sample(t float32) Point {
	t2 := t * t
	oneT := 1 - t
	oneT2 := oneT * oneT

	return q.From.Mul(oneT2).Add(q.Ctrl.ToVector().Mul(2).Mul(oneT).Mul(t)).Add(q.To.ToVector().Mul(t2))
}

// X is QuadraticBezierSegment::x.
func (q *QuadraticBezierSegment) X(t float32) float32 {
	t2 := t * t
	oneT := 1 - t
	oneT2 := oneT * oneT

	return float32(q.From.X*oneT2) + float32(float32(float32(q.Ctrl.X*2)*oneT)*t) + float32(q.To.X*t2)
}

// Y is QuadraticBezierSegment::y.
func (q *QuadraticBezierSegment) Y(t float32) float32 {
	t2 := t * t
	oneT := 1 - t
	oneT2 := oneT * oneT

	return float32(q.From.Y*oneT2) + float32(float32(float32(q.Ctrl.Y*2)*oneT)*t) + float32(q.To.Y*t2)
}

func (q *QuadraticBezierSegment) derivativeCoefficients(t float32) (float32, float32, float32) {
	return float32(2*t) - 2, float32(-4*t) + 2, 2 * t
}

// Derivative is QuadraticBezierSegment::derivative.
func (q *QuadraticBezierSegment) Derivative(t float32) Vector {
	c0, c1, c2 := q.derivativeCoefficients(t)
	return q.From.ToVector().Mul(c0).Add(q.Ctrl.ToVector().Mul(c1)).Add(q.To.ToVector().Mul(c2))
}

// Dx is QuadraticBezierSegment::dx.
func (q *QuadraticBezierSegment) Dx(t float32) float32 {
	c0, c1, c2 := q.derivativeCoefficients(t)
	return float32(q.From.X*c0) + float32(q.Ctrl.X*c1) + float32(q.To.X*c2)
}

// Dy is QuadraticBezierSegment::dy.
func (q *QuadraticBezierSegment) Dy(t float32) float32 {
	c0, c1, c2 := q.derivativeCoefficients(t)
	return float32(q.From.Y*c0) + float32(q.Ctrl.Y*c1) + float32(q.To.Y*c2)
}

// Flip is QuadraticBezierSegment::flip.
func (q *QuadraticBezierSegment) Flip() QuadraticBezierSegment {
	return QuadraticBezierSegment{From: q.To, Ctrl: q.Ctrl, To: q.From}
}

// YMaximumT is QuadraticBezierSegment::y_maximum_t.
func (q *QuadraticBezierSegment) YMaximumT() float32 {
	if t, ok := q.LocalYExtremumT(); ok {
		y := q.Y(t)
		if y > q.From.Y && y > q.To.Y {
			return t
		}
	}
	if q.From.Y > q.To.Y {
		return 0
	}
	return 1
}

// YMinimumT is QuadraticBezierSegment::y_minimum_t.
func (q *QuadraticBezierSegment) YMinimumT() float32 {
	if t, ok := q.LocalYExtremumT(); ok {
		y := q.Y(t)
		if y < q.From.Y && y < q.To.Y {
			return t
		}
	}
	if q.From.Y < q.To.Y {
		return 0
	}
	return 1
}

// LocalYExtremumT is QuadraticBezierSegment::local_y_extremum_t.
func (q *QuadraticBezierSegment) LocalYExtremumT() (float32, bool) {
	div := q.From.Y - float32(2*q.Ctrl.Y) + q.To.Y
	if div == 0 {
		return 0, false
	}
	t := (q.From.Y - q.Ctrl.Y) / div
	if t > 0 && t < 1 {
		return t, true
	}
	return 0, false
}

// XMaximumT is QuadraticBezierSegment::x_maximum_t.
func (q *QuadraticBezierSegment) XMaximumT() float32 {
	if t, ok := q.LocalXExtremumT(); ok {
		x := q.X(t)
		if x > q.From.X && x > q.To.X {
			return t
		}
	}
	if q.From.X > q.To.X {
		return 0
	}
	return 1
}

// XMinimumT is QuadraticBezierSegment::x_minimum_t.
func (q *QuadraticBezierSegment) XMinimumT() float32 {
	if t, ok := q.LocalXExtremumT(); ok {
		x := q.X(t)
		if x < q.From.X && x < q.To.X {
			return t
		}
	}
	if q.From.X < q.To.X {
		return 0
	}
	return 1
}

// LocalXExtremumT is QuadraticBezierSegment::local_x_extremum_t.
func (q *QuadraticBezierSegment) LocalXExtremumT() (float32, bool) {
	div := q.From.X - float32(2*q.Ctrl.X) + q.To.X
	if div == 0 {
		return 0, false
	}
	t := (q.From.X - q.Ctrl.X) / div
	if t > 0 && t < 1 {
		return t, true
	}
	return 0, false
}

// SplitRange is QuadraticBezierSegment::split_range.
func (q *QuadraticBezierSegment) SplitRange(tRange Range) QuadraticBezierSegment {
	t0 := tRange.Start
	t1 := tRange.End

	from := q.Sample(t0)
	to := q.Sample(t1)
	ctrl := from.Add(q.Ctrl.Sub(q.From).Lerp(q.To.Sub(q.Ctrl), t0).Mul(t1 - t0))

	return QuadraticBezierSegment{From: from, Ctrl: ctrl, To: to}
}

// Split is QuadraticBezierSegment::split.
func (q *QuadraticBezierSegment) Split(t float32) (QuadraticBezierSegment, QuadraticBezierSegment) {
	splitPoint := q.Sample(t)
	return QuadraticBezierSegment{From: q.From, Ctrl: q.From.Lerp(q.Ctrl, t), To: splitPoint},
		QuadraticBezierSegment{From: splitPoint, Ctrl: q.Ctrl.Lerp(q.To, t), To: q.To}
}

// BeforeSplit is QuadraticBezierSegment::before_split.
func (q *QuadraticBezierSegment) BeforeSplit(t float32) QuadraticBezierSegment {
	return QuadraticBezierSegment{From: q.From, Ctrl: q.From.Lerp(q.Ctrl, t), To: q.Sample(t)}
}

// AfterSplit is QuadraticBezierSegment::after_split.
func (q *QuadraticBezierSegment) AfterSplit(t float32) QuadraticBezierSegment {
	return QuadraticBezierSegment{From: q.Sample(t), Ctrl: q.Ctrl.Lerp(q.To, t), To: q.To}
}

// ToCubic is QuadraticBezierSegment::to_cubic.
func (q *QuadraticBezierSegment) ToCubic() CubicBezierSegment {
	return CubicBezierSegment{
		From:  q.From,
		Ctrl1: q.From.Add(q.Ctrl.ToVector().Mul(2)).Div(3),
		Ctrl2: q.To.Add(q.Ctrl.ToVector().Mul(2)).Div(3),
		To:    q.To,
	}
}

// IsAPoint is QuadraticBezierSegment::is_a_point.
func (q *QuadraticBezierSegment) IsAPoint(tolerance float32) bool {
	tol2 := tolerance * tolerance
	return q.From.Sub(q.To).SquareLength() <= tol2 && q.From.Sub(q.Ctrl).SquareLength() <= tol2
}

// ForEachMonotonicRange is QuadraticBezierSegment::for_each_monotonic_range.
func (q *QuadraticBezierSegment) ForEachMonotonicRange(cb func(Range)) {
	t0, ok0 := q.LocalXExtremumT()
	t1, ok1 := q.LocalYExtremumT()

	swap := ok0 && ok1 && t0 > t1
	if swap {
		t0, t1 = t1, t0
		ok0, ok1 = ok1, ok0
	}

	var start float32

	if ok0 {
		cb(Range{start, t0})
		start = t0
	}

	if ok1 {
		if t1 != start {
			cb(Range{start, t1})
			start = t1
		}
	}

	cb(Range{start, 1})
}

// ForEachMonotonic is QuadraticBezierSegment::for_each_monotonic.
func (q *QuadraticBezierSegment) ForEachMonotonic(cb func(*QuadraticBezierSegment)) {
	q.ForEachMonotonicRange(func(r Range) {
		sub := q.SplitRange(r)
		minX := Min(sub.From.X, sub.To.X)
		maxX := Max(sub.From.X, sub.To.X)
		minY := Min(sub.From.Y, sub.To.Y)
		maxY := Max(sub.From.Y, sub.To.Y)
		sub.Ctrl.X = Min(Max(sub.Ctrl.X, minX), maxX)
		sub.Ctrl.Y = Min(Max(sub.Ctrl.Y, minY), maxY)
		cb(&sub)
	})
}
