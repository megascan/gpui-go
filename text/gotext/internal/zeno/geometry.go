// Package zeno is a port of the fill path of zeno 0.3.3, the path rasterizer swash uses.
package zeno

import (
	"math"
)

// Vector is a two-dimensional vector.
type Vector struct {
	X, Y float32
}

// Point is a point in two-dimensional space.
type Point = Vector

// VectorZero is the zero vector.
var VectorZero = Vector{}

// NewVector creates a vector.
func NewVector(x, y float32) Vector {
	return Vector{X: x, Y: y}
}

// Add returns the sum of two vectors.
func (v Vector) Add(o Vector) Vector {
	return Vector{X: v.X + o.X, Y: v.Y + o.Y}
}

// Sub returns the difference of two vectors.
func (v Vector) Sub(o Vector) Vector {
	return Vector{X: v.X - o.X, Y: v.Y - o.Y}
}

// Ceil returns the vector with each component rounded up.
func (v Vector) Ceil() Vector {
	return Vector{X: float32(math.Ceil(float64(v.X))), Y: float32(math.Ceil(float64(v.Y)))}
}

// Floor returns the vector with each component rounded down.
func (v Vector) Floor() Vector {
	return Vector{X: float32(math.Floor(float64(v.X))), Y: float32(math.Floor(float64(v.Y)))}
}

// Transform is a two-dimensional affine transformation.
type Transform struct {
	XX, XY, YX, YY, X, Y float32
}

// TransformIdentity is the identity matrix.
var TransformIdentity = Transform{XX: 1, YY: 1}

// NewTransform creates a transform.
func NewTransform(xx, xy, yx, yy, x, y float32) Transform {
	return Transform{XX: xx, XY: xy, YX: yx, YY: yy, X: x, Y: y}
}

// TransformTranslation creates a translation transform.
func TransformTranslation(x, y float32) Transform {
	return NewTransform(1, 0, 0, 1, x, y)
}

// TransformPoint returns the result of applying the transform to a point.
func (t Transform) TransformPoint(p Point) Point {
	return Vector{
		X: (p.X*t.XX + p.Y*t.YX) + t.X,
		Y: (p.X*t.XY + p.Y*t.YY) + t.Y,
	}
}

// TransformVector returns the result of applying the transform to a vector.
func (t Transform) TransformVector(v Vector) Vector {
	return Vector{
		X: v.X*t.XX + v.Y*t.YX,
		Y: v.X*t.XY + v.Y*t.YY,
	}
}

// Origin is the origin of the coordinate system for rendering.
type Origin uint8

const (
	// OriginTopLeft puts (0, 0) at the top left of the image.
	OriginTopLeft Origin = iota
	// OriginBottomLeft puts (0, 0) at the bottom left of the image.
	OriginBottomLeft
)

// Placement is the position and size of a rendered mask.
type Placement struct {
	Left   int32
	Top    int32
	Width  uint32
	Height uint32
}

// Bounds is an axis-aligned bounding box.
type Bounds struct {
	Min, Max Point
}

// NewBounds creates a bounding box from minimum and maximum points.
func NewBounds(min, max Point) Bounds {
	return Bounds{Min: min, Max: max}
}

// BoundsFromPoints creates a bounding box from a sequence of points.
func BoundsFromPoints(points []Point) Bounds {
	b := NewBoundsBuilder()
	for _, p := range points {
		b.add(p)
	}
	return b.Build()
}

// IsEmpty reports whether the bounding box is empty.
func (b Bounds) IsEmpty() bool {
	return b.Min.X >= b.Max.X || b.Min.Y >= b.Max.Y
}

// Width returns the width of the bounding box.
func (b Bounds) Width() float32 {
	return b.Max.X - b.Min.X
}

// Height returns the height of the bounding box.
func (b Bounds) Height() float32 {
	return b.Max.Y - b.Min.Y
}

// Contains reports whether the box contains the point.
func (b Bounds) Contains(p Point) bool {
	return p.X > b.Min.X && p.X < b.Max.X && p.Y > b.Min.Y && p.Y < b.Max.Y
}

// BoundsBuilder accumulates the bounding box of path points.
type BoundsBuilder struct {
	count   int
	start   Point
	current Point
	min     Point
	max     Point
}

// NewBoundsBuilder creates an empty bounds builder.
func NewBoundsBuilder() *BoundsBuilder {
	return &BoundsBuilder{
		min: Point{X: math.MaxFloat32, Y: math.MaxFloat32},
		max: Point{X: -math.MaxFloat32, Y: -math.MaxFloat32},
	}
}

func (b *BoundsBuilder) add(p Point) {
	if p.X < b.min.X {
		b.min.X = p.X
	}
	if p.X > b.max.X {
		b.max.X = p.X
	}
	if p.Y < b.min.Y {
		b.min.Y = p.Y
	}
	if p.Y > b.max.Y {
		b.max.Y = p.Y
	}
	b.count++
}

// Build returns the accumulated bounds.
func (b *BoundsBuilder) Build() Bounds {
	if b.count != 0 {
		return Bounds{Min: b.min, Max: b.max}
	}
	return Bounds{}
}
