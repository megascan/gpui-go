package a11y

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/megascan/gpui-go/jsonschema"
)

// Affine is a 2D affine transform. Derived from kurbo.
type Affine struct {
	c [6]float64
}

// NewAffine constructs an affine transform from coefficients (a, b, c, d, e, f).
func NewAffine(c [6]float64) Affine {
	return Affine{c: c}
}

// AffineIdentity returns the identity transform.
func AffineIdentity() Affine {
	return AffineScale(1.0)
}

// AffineFlipY returns a transform flipped on the y-axis.
func AffineFlipY() Affine {
	return NewAffine([6]float64{1.0, 0., 0., -1.0, 0., 0.})
}

// AffineFlipX returns a transform flipped on the x-axis.
func AffineFlipX() Affine {
	return NewAffine([6]float64{-1.0, 0., 0., 1.0, 0., 0.})
}

// DefaultAffine returns the default transform, which is the identity.
func DefaultAffine() Affine {
	return AffineIdentity()
}

// AffineScale returns a transform representing uniform scaling.
func AffineScale(s float64) Affine {
	return Affine{c: [6]float64{s, 0.0, 0.0, s, 0.0, 0.0}}
}

// AffineScaleNonUniform returns a transform with separate x and y scale values.
func AffineScaleNonUniform(sx, sy float64) Affine {
	return Affine{c: [6]float64{sx, 0.0, 0.0, sy, 0.0, 0.0}}
}

// AffineTranslate returns a transform representing translation by p.
func AffineTranslate(p Vec2) Affine {
	return Affine{c: [6]float64{1.0, 0.0, 0.0, 1.0, p.X, p.Y}}
}

// AffineMapUnitSquare returns the transform that takes the unit square to rect.
func AffineMapUnitSquare(rect Rect) Affine {
	return Affine{c: [6]float64{rect.Width(), 0., 0., rect.Height(), rect.X0, rect.Y0}}
}

// AsCoeffs returns the coefficients of the transform.
func (a Affine) AsCoeffs() [6]float64 {
	return a.c
}

// Determinant returns the determinant of the transform.
func (a Affine) Determinant() float64 {
	return float64(a.c[0]*a.c[3]) - float64(a.c[1]*a.c[2])
}

// Inverse returns the inverse transform. It produces NaN values when the determinant is zero.
func (a Affine) Inverse() Affine {
	invDet := 1 / a.Determinant()
	return Affine{c: [6]float64{
		invDet * a.c[3],
		-invDet * a.c[1],
		-invDet * a.c[2],
		invDet * a.c[0],
		invDet * (float64(a.c[2]*a.c[5]) - float64(a.c[3]*a.c[4])),
		invDet * (float64(a.c[1]*a.c[4]) - float64(a.c[0]*a.c[5])),
	}}
}

// TransformRectBbox returns the minimal Rect enclosing rect after the transform. The result has non-negative width and height.
func (a Affine) TransformRectBbox(rect Rect) Rect {
	p00 := a.MulPoint(Point{X: rect.X0, Y: rect.Y0})
	p01 := a.MulPoint(Point{X: rect.X0, Y: rect.Y1})
	p10 := a.MulPoint(Point{X: rect.X1, Y: rect.Y0})
	p11 := a.MulPoint(Point{X: rect.X1, Y: rect.Y1})
	return RectFromPoints(p00, p01).Union(RectFromPoints(p10, p11))
}

// IsFinite reports whether every coefficient is finite.
func (a Affine) IsFinite() bool {
	for _, v := range a.c {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// IsNan reports whether any coefficient is NaN.
func (a Affine) IsNan() bool {
	for _, v := range a.c {
		if math.IsNaN(v) {
			return true
		}
	}
	return false
}

// MulPoint applies the transform to p.
func (a Affine) MulPoint(p Point) Point {
	return Point{
		X: float64(a.c[0]*p.X) + float64(a.c[2]*p.Y) + a.c[4],
		Y: float64(a.c[1]*p.X) + float64(a.c[3]*p.Y) + a.c[5],
	}
}

// Mul composes two transforms: a * o applies o first, then a.
func (a Affine) Mul(o Affine) Affine {
	return Affine{c: [6]float64{
		float64(a.c[0]*o.c[0]) + float64(a.c[2]*o.c[1]),
		float64(a.c[1]*o.c[0]) + float64(a.c[3]*o.c[1]),
		float64(a.c[0]*o.c[2]) + float64(a.c[2]*o.c[3]),
		float64(a.c[1]*o.c[2]) + float64(a.c[3]*o.c[3]),
		float64(a.c[0]*o.c[4]) + float64(a.c[2]*o.c[5]) + a.c[4],
		float64(a.c[1]*o.c[4]) + float64(a.c[3]*o.c[5]) + a.c[5],
	}}
}

// MulAssign sets a to a * o.
func (a *Affine) MulAssign(o Affine) {
	*a = a.Mul(o)
}

// Float64MulAffine scales every coefficient of a by s.
func Float64MulAffine(s float64, a Affine) Affine {
	return Affine{c: [6]float64{
		s * a.c[0],
		s * a.c[1],
		s * a.c[2],
		s * a.c[3],
		s * a.c[4],
		s * a.c[5],
	}}
}

// String formats the coefficients like Rust's derived Debug output.
func (a Affine) String() string {
	parts := make([]string, len(a.c))
	for i, v := range a.c {
		parts[i] = debugF64(v)
	}
	return "Affine([" + strings.Join(parts, ", ") + "])"
}

// Point is a 2D point. Derived from kurbo.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PointZero returns the point (0, 0).
func PointZero() Point {
	return Point{X: 0., Y: 0.}
}

// PointOrigin returns the point at the origin; (0, 0).
func PointOrigin() Point {
	return Point{X: 0., Y: 0.}
}

// NewPoint creates a Point with the provided x and y coordinates.
func NewPoint(x, y float64) Point {
	return Point{X: x, Y: y}
}

// ToVec2 converts the point into a Vec2.
func (p Point) ToVec2() Vec2 {
	return NewVec2(p.X, p.Y)
}

// Add translates the point by v.
func (p Point) Add(v Vec2) Point {
	return NewPoint(p.X+v.X, p.Y+v.Y)
}

// AddAssign translates the point in place by v.
func (p *Point) AddAssign(v Vec2) {
	*p = NewPoint(p.X+v.X, p.Y+v.Y)
}

// Sub translates the point by -v.
func (p Point) Sub(v Vec2) Point {
	return NewPoint(p.X-v.X, p.Y-v.Y)
}

// SubAssign translates the point in place by -v.
func (p *Point) SubAssign(v Vec2) {
	*p = NewPoint(p.X-v.X, p.Y-v.Y)
}

// SubPoint returns the vector from o to p.
func (p Point) SubPoint(o Point) Vec2 {
	return NewVec2(p.X-o.X, p.Y-o.Y)
}

// String formats the point like Rust's Debug output, "(x, y)".
func (p Point) String() string {
	return "(" + debugF64(p.X) + ", " + debugF64(p.Y) + ")"
}

// Rect is a rectangle. Derived from kurbo.
type Rect struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
}

// RectZero returns the empty rectangle at the origin.
func RectZero() Rect {
	return NewRect(0., 0., 0., 0.)
}

// NewRect creates a rectangle from minimum and maximum coordinates.
func NewRect(x0, y0, x1, y1 float64) Rect {
	return Rect{X0: x0, Y0: y0, X1: x1, Y1: y1}
}

// RectFromPoints creates a rectangle from two points. The result has non-negative width and height.
func RectFromPoints(p0, p1 Point) Rect {
	return NewRect(p0.X, p0.Y, p1.X, p1.Y).Abs()
}

// RectFromOriginSize creates a rectangle from an origin and a size. The result has non-negative width and height.
func RectFromOriginSize(origin Point, size Size) Rect {
	return RectFromPoints(origin, origin.Add(size.ToVec2()))
}

// WithOrigin returns a rectangle with the same size as r and a new origin.
func (r Rect) WithOrigin(origin Point) Rect {
	return RectFromOriginSize(origin, r.Size())
}

// WithSize returns a rectangle with the same origin as r and a new size.
func (r Rect) WithSize(size Size) Rect {
	return RectFromOriginSize(r.Origin(), size)
}

// Width returns the width of the rectangle. Negative widths are not forbidden.
func (r Rect) Width() float64 {
	return r.X1 - r.X0
}

// Height returns the height of the rectangle. Negative heights are not forbidden.
func (r Rect) Height() float64 {
	return r.Y1 - r.Y0
}

// MinX returns the minimum x-coordinate of the rectangle.
func (r Rect) MinX() float64 {
	return rustMin(r.X0, r.X1)
}

// MaxX returns the maximum x-coordinate of the rectangle.
func (r Rect) MaxX() float64 {
	return rustMax(r.X0, r.X1)
}

// MinY returns the minimum y-coordinate of the rectangle.
func (r Rect) MinY() float64 {
	return rustMin(r.Y0, r.Y1)
}

// MaxY returns the maximum y-coordinate of the rectangle.
func (r Rect) MaxY() float64 {
	return rustMax(r.Y0, r.Y1)
}

// Origin returns the top-left corner in a y-down space.
func (r Rect) Origin() Point {
	return NewPoint(r.X0, r.Y0)
}

// Size returns the size of the rectangle.
func (r Rect) Size() Size {
	return NewSize(r.Width(), r.Height())
}

// Abs returns the rectangle with non-negative width and height and the same extents.
func (r Rect) Abs() Rect {
	return NewRect(rustMin(r.X0, r.X1), rustMin(r.Y0, r.Y1), rustMax(r.X0, r.X1), rustMax(r.Y0, r.Y1))
}

// Area returns the area of the rectangle.
func (r Rect) Area() float64 {
	return r.Width() * r.Height()
}

// IsEmpty reports whether the rectangle has zero area. A rectangle with negative area is not empty.
func (r Rect) IsEmpty() bool {
	return r.Area() == 0.0
}

// Contains reports whether p lies within r.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X0 && p.X < r.X1 && p.Y >= r.Y0 && p.Y < r.Y1
}

// Union returns the smallest rectangle enclosing r and other.
func (r Rect) Union(other Rect) Rect {
	return NewRect(
		rustMin(r.X0, other.X0),
		rustMin(r.Y0, other.Y0),
		rustMax(r.X1, other.X1),
		rustMax(r.Y1, other.Y1),
	)
}

// UnionPt returns the smallest rectangle enclosing r and pt, including the perimeter of zero-area rectangles.
func (r Rect) UnionPt(pt Point) Rect {
	return NewRect(
		rustMin(r.X0, pt.X),
		rustMin(r.Y0, pt.Y),
		rustMax(r.X1, pt.X),
		rustMax(r.Y1, pt.Y),
	)
}

// Intersect returns the intersection of r and other. The result always has non-negative width and height.
func (r Rect) Intersect(other Rect) Rect {
	x0 := rustMax(r.X0, other.X0)
	y0 := rustMax(r.Y0, other.Y0)
	x1 := rustMin(r.X1, other.X1)
	y1 := rustMin(r.Y1, other.Y1)
	return NewRect(x0, y0, rustMax(x1, x0), rustMax(y1, y0))
}

// Add translates the rectangle by v.
func (r Rect) Add(v Vec2) Rect {
	return NewRect(r.X0+v.X, r.Y0+v.Y, r.X1+v.X, r.Y1+v.Y)
}

// Sub translates the rectangle by -v.
func (r Rect) Sub(v Vec2) Rect {
	return NewRect(r.X0-v.X, r.Y0-v.Y, r.X1-v.X, r.Y1-v.Y)
}

// String formats the rectangle like Rust's default Debug output. Rust's alternate form is not provided.
func (r Rect) String() string {
	return "Rect { x0: " + debugF64(r.X0) + ", y0: " + debugF64(r.Y0) +
		", x1: " + debugF64(r.X1) + ", y1: " + debugF64(r.Y1) + " }"
}

// Size is a 2D size. Derived from kurbo.
type Size struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// SizeZero returns a size with zero width and height.
func SizeZero() Size {
	return NewSize(0., 0.)
}

// NewSize creates a Size with the provided width and height.
func NewSize(width, height float64) Size {
	return Size{Width: width, Height: height}
}

// ToVec2 converts the size into a Vec2, with width mapped to x and height mapped to y.
func (s Size) ToVec2() Vec2 {
	return NewVec2(s.Width, s.Height)
}

// String formats the size like Rust's Debug output, "WxH" with W and H suffixes.
func (s Size) String() string {
	return debugF64(s.Width) + "W×" + debugF64(s.Height) + "H"
}

// MulAssign scales the size in place by f.
func (s *Size) MulAssign(f float64) {
	*s = Size{Width: s.Width * f, Height: s.Height * f}
}

// Mul returns the size scaled by f.
func (s Size) Mul(f float64) Size {
	return Size{Width: s.Width * f, Height: s.Height * f}
}

// Float64MulSize returns the size scaled by f.
func Float64MulSize(f float64, s Size) Size {
	return s.Mul(f)
}

// DivAssign divides the size in place by f.
func (s *Size) DivAssign(f float64) {
	*s = Size{Width: s.Width / f, Height: s.Height / f}
}

// Div returns the size divided by f.
func (s Size) Div(f float64) Size {
	return Size{Width: s.Width / f, Height: s.Height / f}
}

// Add returns the component-wise sum of s and o.
func (s Size) Add(o Size) Size {
	return Size{Width: s.Width + o.Width, Height: s.Height + o.Height}
}

// AddAssign adds o to s in place.
func (s *Size) AddAssign(o Size) {
	*s = s.Add(o)
}

// Sub returns the component-wise difference of s and o.
func (s Size) Sub(o Size) Size {
	return Size{Width: s.Width - o.Width, Height: s.Height - o.Height}
}

// SubAssign subtracts o from s in place.
func (s *Size) SubAssign(o Size) {
	*s = s.Sub(o)
}

// Vec2 is a 2D vector. Derived from kurbo.
//
// It is intended primarily for a vector in the mathematical sense, but it can be interpreted as a translation, and converted to and from a point and size.
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Vec2Zero returns the vector (0, 0).
func Vec2Zero() Vec2 {
	return NewVec2(0., 0.)
}

// NewVec2 creates a new vector.
func NewVec2(x, y float64) Vec2 {
	return Vec2{X: x, Y: y}
}

// ToPoint converts the vector into a Point.
func (v Vec2) ToPoint() Point {
	return NewPoint(v.X, v.Y)
}

// ToSize converts the vector into a Size.
func (v Vec2) ToSize() Size {
	return NewSize(v.X, v.Y)
}

// Add returns the sum of v and o.
func (v Vec2) Add(o Vec2) Vec2 {
	return Vec2{X: v.X + o.X, Y: v.Y + o.Y}
}

// AddAssign adds o to v in place.
func (v *Vec2) AddAssign(o Vec2) {
	*v = Vec2{X: v.X + o.X, Y: v.Y + o.Y}
}

// Sub returns the difference of v and o.
func (v Vec2) Sub(o Vec2) Vec2 {
	return Vec2{X: v.X - o.X, Y: v.Y - o.Y}
}

// SubAssign subtracts o from v in place.
func (v *Vec2) SubAssign(o Vec2) {
	*v = Vec2{X: v.X - o.X, Y: v.Y - o.Y}
}

// Mul returns v scaled by f.
func (v Vec2) Mul(f float64) Vec2 {
	return Vec2{X: v.X * f, Y: v.Y * f}
}

// MulAssign scales v in place by f.
func (v *Vec2) MulAssign(f float64) {
	*v = Vec2{X: v.X * f, Y: v.Y * f}
}

// Float64MulVec2 returns v scaled by f.
func Float64MulVec2(f float64, v Vec2) Vec2 {
	return v.Mul(f)
}

// Div returns v divided by f. It multiplies by the reciprocal, as upstream does, so rounding differs from plain division.
func (v Vec2) Div(f float64) Vec2 {
	return v.Mul(1 / f)
}

// DivAssign divides v in place by f, multiplying by the reciprocal as upstream does.
func (v *Vec2) DivAssign(f float64) {
	v.MulAssign(1 / f)
}

// Neg returns the negated vector.
func (v Vec2) Neg() Vec2 {
	return Vec2{X: -v.X, Y: -v.Y}
}

// String formats the vector like Rust's derived Debug output.
func (v Vec2) String() string {
	return "Vec2 { x: " + debugF64(v.X) + ", y: " + debugF64(v.Y) + " }"
}

func rustMin(a, b float64) float64 {
	switch {
	case math.IsNaN(a):
		return b
	case math.IsNaN(b):
		return a
	case a < b:
		return a
	}
	return b
}

func rustMax(a, b float64) float64 {
	switch {
	case math.IsNaN(a):
		return b
	case math.IsNaN(b):
		return a
	case a > b:
		return a
	}
	return b
}

func debugF64(v float64) string {
	return debugFloat(v, 64)
}

func debugF32(v float32) string {
	return debugFloat(float64(v), 32)
}

func debugFloat(v float64, bits int) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	}
	if a := math.Abs(v); a != 0 && (a < 1e-4 || a >= 1e16) {
		mantissa, exp, _ := strings.Cut(strconv.FormatFloat(v, 'e', -1, bits), "e")
		sign := ""
		if exp[0] == '-' {
			sign = "-"
		}
		digits := strings.TrimLeft(strings.TrimLeft(exp, "+-"), "0")
		return mantissa + "e" + sign + digits
	}
	s := strconv.FormatFloat(v, 'f', -1, bits)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// MarshalJSONTo encodes a as the six-element coefficient array serde uses for the newtype.
func (a Affine) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, a.c)
}

// UnmarshalJSONFrom decodes a six-element coefficient array.
func (a *Affine) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return json.UnmarshalDecode(dec, &a.c)
}

// JSONSchema describes a as its six-element coefficient array.
func (a *Affine) JSONSchema(g *jsonschema.Generator) *jsonschema.Schema {
	return g.SchemaFor(reflect.TypeFor[[6]float64]())
}
