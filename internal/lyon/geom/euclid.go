package geom

import "math"

// Point is euclid's default::Point2D<f32>.
type Point struct {
	X float32
	Y float32
}

// Vector is euclid's default::Vector2D<f32>.
type Vector struct {
	X float32
	Y float32
}

// Angle is euclid's Angle<f32>.
type Angle struct {
	Radians float32
}

// Transform is euclid's default::Transform2D<f32>, a row-major 3x2 matrix.
type Transform struct {
	M11, M12 float32
	M21, M22 float32
	M31, M32 float32
}

// Rotation is euclid's default::Rotation2D<f32>.
type Rotation struct {
	Angle float32
}

// Range is core::ops::Range<f32>.
type Range struct {
	Start float32
	End   float32
}

// ApproxEpsilon is euclid's ApproxEq<f32>::approx_epsilon.
const ApproxEpsilon float32 = 1.0e-6

// ApproxEqF32 is euclid's ApproxEq::approx_eq for f32.
func ApproxEqF32(a, b float32) bool {
	return ApproxEqEpsF32(a, b, ApproxEpsilon)
}

// ApproxEqEpsF32 is euclid's ApproxEq::approx_eq_eps for f32.
func ApproxEqEpsF32(a, b, eps float32) bool {
	return abs(a-b) < eps
}

// Add returns p + v.
func (p Point) Add(v Vector) Point { return Point{p.X + v.X, p.Y + v.Y} }

// Sub returns p - o as a vector.
func (p Point) Sub(o Point) Vector { return Vector{p.X - o.X, p.Y - o.Y} }

// SubVector returns p - v.
func (p Point) SubVector(v Vector) Point { return Point{p.X - v.X, p.Y - v.Y} }

// Mul returns p * s.
func (p Point) Mul(s float32) Point { return Point{float32(p.X * s), float32(p.Y * s)} }

// Div returns p / s.
func (p Point) Div(s float32) Point { return Point{float32(p.X / s), float32(p.Y / s)} }

// ToVector is euclid's Point2D::to_vector.
func (p Point) ToVector() Vector { return Vector(p) }

// Lerp is euclid's Point2D::lerp.
func (p Point) Lerp(other Point, t float32) Point {
	oneT := 1 - t
	return Point{
		float32(oneT*p.X) + float32(t*other.X),
		float32(oneT*p.Y) + float32(t*other.Y),
	}
}

// ApproxEq is euclid's ApproxEq::approx_eq for Point2D.
func (p Point) ApproxEq(other Point) bool {
	return p.ApproxEqEps(other, Point{ApproxEpsilon, ApproxEpsilon})
}

// ApproxEqEps is euclid's ApproxEq::approx_eq_eps for Point2D.
func (p Point) ApproxEqEps(other Point, eps Point) bool {
	return ApproxEqEpsF32(p.X, other.X, eps.X) && ApproxEqEpsF32(p.Y, other.Y, eps.Y)
}

// Add returns v + o.
func (v Vector) Add(o Vector) Vector { return Vector{v.X + o.X, v.Y + o.Y} }

// Sub returns v - o.
func (v Vector) Sub(o Vector) Vector { return Vector{v.X - o.X, v.Y - o.Y} }

// Mul returns v * s.
func (v Vector) Mul(s float32) Vector { return Vector{float32(v.X * s), float32(v.Y * s)} }

// Div returns v / s.
func (v Vector) Div(s float32) Vector { return Vector{float32(v.X / s), float32(v.Y / s)} }

// Neg returns -v.
func (v Vector) Neg() Vector { return Vector{-v.X, -v.Y} }

// ToPoint is euclid's Vector2D::to_point.
func (v Vector) ToPoint() Point { return Point(v) }

// SquareLength is euclid's Vector2D::square_length.
func (v Vector) SquareLength() float32 { return float32(v.X*v.X) + float32(v.Y*v.Y) }

// Length is euclid's Vector2D::length.
func (v Vector) Length() float32 { return sqrt(v.SquareLength()) }

// Lerp is euclid's Vector2D::lerp.
func (v Vector) Lerp(other Vector, t float32) Vector {
	oneT := 1 - t
	return v.Mul(oneT).Add(other.Mul(t))
}

// AngleFromXAxis is euclid's Vector2D::angle_from_x_axis.
func (v Vector) AngleFromXAxis() Angle {
	return AngleRadians(FastAtan2(v.Y, v.X))
}

// AngleRadians is euclid's Angle::radians.
func AngleRadians(radians float32) Angle { return Angle{radians} }

// AngleDegrees is euclid's Angle::degrees.
func AngleDegrees(deg float32) Angle { return Angle{ToRadians(deg)} }

// AngleZero is euclid's Angle::zero.
func AngleZero() Angle { return Angle{} }

// AnglePi is euclid's Angle::pi.
func AnglePi() Angle { return Angle{pi} }

// AngleTwoPi is euclid's Angle::two_pi.
func AngleTwoPi() Angle { return Angle{pi + pi} }

// Get is euclid's Angle::get.
func (a Angle) Get() float32 { return a.Radians }

// ToDegrees is euclid's Angle::to_degrees.
func (a Angle) ToDegrees() float32 { return ToDegrees(a.Radians) }

// Add returns a + o.
func (a Angle) Add(o Angle) Angle { return Angle{a.Radians + o.Radians} }

// Sub returns a - o.
func (a Angle) Sub(o Angle) Angle { return Angle{a.Radians - o.Radians} }

// Mul returns a * s.
func (a Angle) Mul(s float32) Angle { return Angle{float32(a.Radians * s)} }

// Positive is euclid's Angle::positive.
func (a Angle) Positive() Angle {
	twoPi := pi + pi
	r := rem(a.Radians, twoPi)
	if r < 0 {
		r = r + twoPi
	}
	return Angle{r}
}

// Signed is euclid's Angle::signed.
func (a Angle) Signed() Angle {
	return AnglePi().Sub(AnglePi().Sub(a).Positive())
}

// FastAtan2 is euclid's Trig::fast_atan2 for f32.
func FastAtan2(y, x float32) float32 {
	xAbs := abs(x)
	yAbs := abs(y)
	a := float32(Min(xAbs, yAbs) / Max(xAbs, yAbs))
	s := float32(a * a)
	r1 := float32(-0.046_496_474_9*s) + 0.159_314_22
	r2 := float32(r1*s) - 0.327_622_764
	result := float32(float32(r2*s)*a) + a
	if yAbs > xAbs {
		result = fracPi2 - result
	}
	if x < 0.0 {
		result = pi - result
	}
	if y < 0.0 {
		result = -result
	}
	return result
}

// ToRadians is f32::to_radians.
func ToRadians(deg float32) float32 { return float32(deg * radsPerDeg) }

// ToDegrees is f32::to_degrees.
func ToDegrees(rad float32) float32 { return float32(rad * degsPerRad) }

// NewTransform is euclid's Transform2D::new.
func NewTransform(m11, m12, m21, m22, m31, m32 float32) Transform {
	return Transform{m11, m12, m21, m22, m31, m32}
}

// TransformIdentity is euclid's Transform2D::identity.
func TransformIdentity() Transform { return TransformTranslation(0, 0) }

// TransformTranslation is euclid's Transform2D::translation.
func TransformTranslation(x, y float32) Transform {
	return NewTransform(1, 0, 0, 1, x, y)
}

// TransformScale is euclid's Transform2D::scale.
func TransformScale(x, y float32) Transform {
	return NewTransform(x, 0, 0, y, 0, 0)
}

// TransformRotation is euclid's Transform2D::rotation.
func TransformRotation(theta Angle) Transform {
	var zero float32
	c := cos(theta.Get())
	s := sin(theta.Get())
	return NewTransform(c, s, zero-s, c, zero, zero)
}

// IsIdentity is euclid's Transform2D::is_identity.
func (t Transform) IsIdentity() bool { return t == TransformIdentity() }

// Then is euclid's Transform2D::then.
func (t Transform) Then(mat Transform) Transform {
	return NewTransform(
		float32(t.M11*mat.M11)+float32(t.M12*mat.M21),
		float32(t.M11*mat.M12)+float32(t.M12*mat.M22),
		float32(t.M21*mat.M11)+float32(t.M22*mat.M21),
		float32(t.M21*mat.M12)+float32(t.M22*mat.M22),
		float32(t.M31*mat.M11)+float32(t.M32*mat.M21)+mat.M31,
		float32(t.M31*mat.M12)+float32(t.M32*mat.M22)+mat.M32,
	)
}

// ThenTranslate is euclid's Transform2D::then_translate.
func (t Transform) ThenTranslate(v Vector) Transform {
	return t.Then(TransformTranslation(v.X, v.Y))
}

// PreTranslate is euclid's Transform2D::pre_translate.
func (t Transform) PreTranslate(v Vector) Transform {
	return TransformTranslation(v.X, v.Y).Then(t)
}

// ThenScale is euclid's Transform2D::then_scale.
func (t Transform) ThenScale(x, y float32) Transform {
	return t.Then(TransformScale(x, y))
}

// PreScale is euclid's Transform2D::pre_scale.
func (t Transform) PreScale(x, y float32) Transform {
	return NewTransform(
		float32(t.M11*x), float32(t.M12*x),
		float32(t.M21*y), float32(t.M22*y),
		t.M31, t.M32,
	)
}

// ThenRotate is euclid's Transform2D::then_rotate.
func (t Transform) ThenRotate(theta Angle) Transform {
	return t.Then(TransformRotation(theta))
}

// PreRotate is euclid's Transform2D::pre_rotate.
func (t Transform) PreRotate(theta Angle) Transform {
	return TransformRotation(theta).Then(t)
}

// TransformPoint is euclid's Transform2D::transform_point.
func (t Transform) TransformPoint(p Point) Point {
	return Point{
		float32(p.X*t.M11) + float32(p.Y*t.M21) + t.M31,
		float32(p.X*t.M12) + float32(p.Y*t.M22) + t.M32,
	}
}

// TransformVector is euclid's Transform2D::transform_vector.
func (t Transform) TransformVector(v Vector) Vector {
	return Vector{
		float32(v.X*t.M11) + float32(v.Y*t.M21),
		float32(v.X*t.M12) + float32(v.Y*t.M22),
	}
}

// Determinant is euclid's Transform2D::determinant.
func (t Transform) Determinant() float32 {
	return float32(t.M11*t.M22) - float32(t.M12*t.M21)
}

// Inverse is euclid's Transform2D::inverse.
func (t Transform) Inverse() (Transform, bool) {
	det := t.Determinant()
	var zero float32
	if det == zero {
		return Transform{}, false
	}
	invDet := 1 / det
	return NewTransform(
		float32(invDet*t.M22),
		float32(invDet*(zero-t.M12)),
		float32(invDet*(zero-t.M21)),
		float32(invDet*t.M11),
		float32(invDet*(float32(t.M21*t.M32)-float32(t.M22*t.M31))),
		float32(invDet*(float32(t.M31*t.M12)-float32(t.M11*t.M32))),
	), true
}

// NewRotation is euclid's Rotation2D::new.
func NewRotation(angle Angle) Rotation { return Rotation{angle.Radians} }

// TransformPoint is euclid's Rotation2D::transform_point.
func (r Rotation) TransformPoint(p Point) Point {
	s, c := sin(r.Angle), cos(r.Angle)
	return Point{float32(p.X*c) - float32(p.Y*s), float32(p.Y*c) + float32(p.X*s)}
}

// TransformVector is euclid's Rotation2D::transform_vector.
func (r Rotation) TransformVector(v Vector) Vector {
	return r.TransformPoint(v.ToPoint()).ToVector()
}

var (
	pi         = float32(math.Pi)
	fracPi2    = float32(math.Pi / 2)
	fracPi4    = float32(math.Pi / 4)
	radsPerDeg = pi / 180.0
	degsPerRad = float32(57.2957795130823208767981548141051703)
)

// Min is f32::min: when one operand is NaN the other is returned.
func Min(a, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if b < a {
		return b
	}
	return a
}

// Max is f32::max: when one operand is NaN the other is returned.
func Max(a, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if b > a {
		return b
	}
	return a
}

func abs(x float32) float32 { return math.Float32frombits(math.Float32bits(x) &^ (1 << 31)) }

func sqrt(x float32) float32 { return float32(math.Sqrt(float64(x))) }

func sin(x float32) float32 { return float32(math.Sin(float64(x))) }

func cos(x float32) float32 { return float32(math.Cos(float64(x))) }

func tan(x float32) float32 { return float32(math.Tan(float64(x))) }

func ceil(x float32) float32 { return float32(math.Ceil(float64(x))) }

func rem(x, y float32) float32 { return float32(math.Mod(float64(x), float64(y))) }

func signum(x float32) float32 {
	if x != x {
		return x
	}
	if math.Signbit(float64(x)) {
		return -1
	}
	return 1
}

func asI32(x float32) int32 {
	switch {
	case x != x:
		return 0
	case x >= 2147483647:
		return math.MaxInt32
	case x <= -2147483648:
		return math.MinInt32
	}
	return int32(x)
}

func powf(x, y float32) float32 { return float32(math.Pow(float64(x), float64(y))) }

func asU32(x float32, fallback uint32) uint32 {
	if x != x || x < 0 || x > math.MaxUint32 {
		return fallback
	}
	return uint32(x)
}
