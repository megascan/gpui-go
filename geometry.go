package gpui

import (
	"cmp"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/jsonschema"
)

//go:generate go run github.com/megascan/gpui-go/internal/cmd/refinegen geometry.go

// Number constrains the numeric family of geometry types (D-GEOM).
type Number interface {
	~float32 | ~float64 | ~int32 | ~int64 | ~int | ~uint32 | ~uint64 | ~uint
}

// Axis is an axis in a 2D cartesian space.
type Axis uint8

const (
	// AxisVertical is the y axis, or up and down.
	AxisVertical Axis = iota
	// AxisHorizontal is the x axis, or left and right.
	AxisHorizontal
)

// Invert swaps this axis to the opposite axis.
func (a Axis) Invert() Axis {
	switch a {
	case AxisVertical:
		return AxisHorizontal
	default:
		return AxisVertical
	}
}

// String formats the axis like its Rust Debug output.
func (a Axis) String() string {
	switch a {
	case AxisVertical:
		return "Vertical"
	case AxisHorizontal:
		return "Horizontal"
	}
	return fmt.Sprintf("Axis(%d)", uint8(a))
}

// MarshalJSONTo encodes the axis as its variant name.
func (a Axis) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(a.String()))
}

// UnmarshalJSONFrom decodes the axis from its variant name.
func (a *Axis) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '"' {
		return fmt.Errorf("invalid type: %v, expected variant identifier", tok.Kind())
	}
	switch s := tok.String(); s {
	case "Vertical":
		*a = AxisVertical
	case "Horizontal":
		*a = AxisHorizontal
	default:
		return fmt.Errorf("unknown variant `%s`, expected `Vertical` or `Horizontal`", s)
	}
	return nil
}

// Along accesses the unit U of a value S along an axis.
type Along[U any, S any] interface {
	Along(axis Axis) U
	ApplyAlong(axis Axis, f func(U) U) S
}

// Point is a location in a 2D cartesian space.
//
//gpui:refineable Debug, PartialEq, Serialize
type Point[T Number] struct {
	X T `json:"x"`
	Y T `json:"y"`
}

// NewPoint constructs a Point with the given x and y coordinates.
func NewPoint[T Number](x, y T) Point[T] {
	return Point[T]{X: x, Y: y}
}

// Map transforms the point to a Point[U] by applying f to both coordinates.
func (p Point[T]) Map[U Number](f func(T) U) Point[U] {
	return Point[U]{X: f(p.X), Y: f(p.Y)}
}

// MapOf transforms the point to a PointOf[U] by applying f to both coordinates.
func (p Point[T]) MapOf[U any](f func(T) U) PointOf[U] {
	return PointOf[U]{X: f(p.X), Y: f(p.Y)}
}

// Along returns the coordinate along the given axis.
func (p Point[T]) Along(axis Axis) T {
	switch axis {
	case AxisHorizontal:
		return p.X
	default:
		return p.Y
	}
}

// ApplyAlong applies f to the coordinate along the given axis.
func (p Point[T]) ApplyAlong(axis Axis, f func(T) T) Point[T] {
	switch axis {
	case AxisHorizontal:
		return Point[T]{X: f(p.X), Y: p.Y}
	default:
		return Point[T]{X: p.X, Y: f(p.Y)}
	}
}

// Scale scales the point by factor, producing scaled pixels.
func (p Point[T]) Scale(factor float32) Point[ScaledPixels] {
	return Point[ScaledPixels]{X: scalePixels(p.X, factor), Y: scalePixels(p.Y, factor)}
}

// Magnitude returns the Euclidean distance from the origin to this point.
func (p Point[T]) Magnitude() float64 {
	x, y := float32(p.X), float32(p.Y)
	return math.Sqrt(float64(float32(x*x) + float32(y*y)))
}

// RelativeTo returns the position of this point relative to origin.
func (p Point[T]) RelativeTo(origin Point[T]) Point[T] {
	return NewPoint(p.X-origin.X, p.Y-origin.Y)
}

// Add adds two points component-wise.
func (p Point[T]) Add(rhs Point[T]) Point[T] {
	return Point[T]{X: p.X + rhs.X, Y: p.Y + rhs.Y}
}

// AddAssign adds rhs to p component-wise.
func (p *Point[T]) AddAssign(rhs Point[T]) {
	p.X += rhs.X
	p.Y += rhs.Y
}

// Sub subtracts two points component-wise.
func (p Point[T]) Sub(rhs Point[T]) Point[T] {
	return Point[T]{X: p.X - rhs.X, Y: p.Y - rhs.Y}
}

// SubAssign subtracts rhs from p component-wise.
func (p *Point[T]) SubAssign(rhs Point[T]) {
	p.X -= rhs.X
	p.Y -= rhs.Y
}

// Neg negates both coordinates.
func (p Point[T]) Neg() Point[T] {
	return Point[T]{X: -p.X, Y: -p.Y}
}

// Mul multiplies both coordinates by rhs.
func (p Point[T]) Mul(rhs T) Point[T] {
	return Point[T]{X: p.X * rhs, Y: p.Y * rhs}
}

// MulAssign multiplies both coordinates of p by rhs.
func (p *Point[T]) MulAssign(rhs T) {
	p.X = p.X * rhs
	p.Y = p.Y * rhs
}

// Div divides both coordinates by rhs.
func (p Point[T]) Div(rhs T) Point[T] {
	return Point[T]{X: p.X / rhs, Y: p.Y / rhs}
}

// Max returns the point with the maximum of each coordinate of p and other.
func (p Point[T]) Max(other Point[T]) Point[T] {
	var out Point[T]
	if geomGT(p.X, other.X) {
		out.X = p.X
	} else {
		out.X = other.X
	}
	if geomGT(p.Y, other.Y) {
		out.Y = p.Y
	} else {
		out.Y = other.Y
	}
	return out
}

// Min returns the point with the minimum of each coordinate of p and other.
func (p Point[T]) Min(other Point[T]) Point[T] {
	var out Point[T]
	if geomLE(p.X, other.X) {
		out.X = p.X
	} else {
		out.X = other.X
	}
	if geomLE(p.Y, other.Y) {
		out.Y = p.Y
	} else {
		out.Y = other.Y
	}
	return out
}

// Clamp constrains the point to the range between min and max.
func (p Point[T]) Clamp(min, max Point[T]) Point[T] {
	return p.Max(min).Min(max)
}

// ToSize converts the point to a Size with width x and height y.
func (p Point[T]) ToSize() Size[T] {
	return Size[T]{Width: p.X, Height: p.Y}
}

// String formats the point as (x, y).
func (p Point[T]) String() string {
	return fmt.Sprintf("(%v, %v)", p.X, p.Y)
}

// GoString formats the point like its Rust Debug output.
func (p Point[T]) GoString() string {
	return fmt.Sprintf("Point { x: %v, y: %v }", p.X, p.Y)
}

// PointOf is Point for non-numeric contents (D-GEOM).
//
//gpui:refineable Debug, PartialEq, Serialize
type PointOf[T any] struct {
	X T `json:"x"`
	Y T `json:"y"`
}

// NewPointOf constructs a PointOf with the given x and y values.
func NewPointOf[T any](x, y T) PointOf[T] {
	return PointOf[T]{X: x, Y: y}
}

// Map transforms the point to a PointOf[U] by applying f to both values.
func (p PointOf[T]) Map[U any](f func(T) U) PointOf[U] {
	return PointOf[U]{X: f(p.X), Y: f(p.Y)}
}

// MapNumber transforms the point to a Point[U] by applying f to both values.
func (p PointOf[T]) MapNumber[U Number](f func(T) U) Point[U] {
	return Point[U]{X: f(p.X), Y: f(p.Y)}
}

// Along returns the value along the given axis.
func (p PointOf[T]) Along(axis Axis) T {
	switch axis {
	case AxisHorizontal:
		return p.X
	default:
		return p.Y
	}
}

// ApplyAlong applies f to the value along the given axis.
func (p PointOf[T]) ApplyAlong(axis Axis, f func(T) T) PointOf[T] {
	switch axis {
	case AxisHorizontal:
		return PointOf[T]{X: f(p.X), Y: p.Y}
	default:
		return PointOf[T]{X: p.X, Y: f(p.Y)}
	}
}

// String formats the point as (x, y).
func (p PointOf[T]) String() string {
	return fmt.Sprintf("(%v, %v)", p.X, p.Y)
}

// GoString formats the point like its Rust Debug output.
func (p PointOf[T]) GoString() string {
	return fmt.Sprintf("Point { x: %v, y: %v }", p.X, p.Y)
}

// Size is a two-dimensional size with a width and a height.
//
//gpui:refineable Debug, PartialEq, Serialize
type Size[T Number] struct {
	Width  T `json:"width"`
	Height T `json:"height"`
}

// NewSize constructs a Size with the given width and height.
func NewSize[T Number](width, height T) Size[T] {
	return Size[T]{Width: width, Height: height}
}

// Map applies f to the width and height, producing a Size[U].
func (s Size[T]) Map[U Number](f func(T) U) Size[U] {
	return Size[U]{Width: f(s.Width), Height: f(s.Height)}
}

// MapOf applies f to the width and height, producing a SizeOf[U].
func (s Size[T]) MapOf[U any](f func(T) U) SizeOf[U] {
	return SizeOf[U]{Width: f(s.Width), Height: f(s.Height)}
}

// Center returns the center point of the size.
func (s Size[T]) Center() Point[T] {
	return Point[T]{X: half(s.Width), Y: half(s.Height)}
}

// Scale scales the size by factor, producing scaled pixels.
func (s Size[T]) Scale(factor float32) Size[ScaledPixels] {
	return Size[ScaledPixels]{Width: scalePixels(s.Width, factor), Height: scalePixels(s.Height, factor)}
}

// Along returns the dimension along the given axis.
func (s Size[T]) Along(axis Axis) T {
	switch axis {
	case AxisHorizontal:
		return s.Width
	default:
		return s.Height
	}
}

// ApplyAlong applies f to the dimension along the given axis.
func (s Size[T]) ApplyAlong(axis Axis, f func(T) T) Size[T] {
	switch axis {
	case AxisHorizontal:
		return Size[T]{Width: f(s.Width), Height: s.Height}
	default:
		return Size[T]{Width: s.Width, Height: f(s.Height)}
	}
}

// Max returns the size with the maximum width and height of s and other.
func (s Size[T]) Max(other Size[T]) Size[T] {
	var out Size[T]
	if geomGE(s.Width, other.Width) {
		out.Width = s.Width
	} else {
		out.Width = other.Width
	}
	if geomGE(s.Height, other.Height) {
		out.Height = s.Height
	} else {
		out.Height = other.Height
	}
	return out
}

// Min returns the size with the minimum width and height of s and other.
func (s Size[T]) Min(other Size[T]) Size[T] {
	var out Size[T]
	if geomGE(s.Width, other.Width) {
		out.Width = other.Width
	} else {
		out.Width = s.Width
	}
	if geomGE(s.Height, other.Height) {
		out.Height = other.Height
	} else {
		out.Height = s.Height
	}
	return out
}

// Add adds two sizes component-wise.
func (s Size[T]) Add(rhs Size[T]) Size[T] {
	return Size[T]{Width: s.Width + rhs.Width, Height: s.Height + rhs.Height}
}

// Sub subtracts two sizes component-wise.
func (s Size[T]) Sub(rhs Size[T]) Size[T] {
	return Size[T]{Width: s.Width - rhs.Width, Height: s.Height - rhs.Height}
}

// Neg negates the width and height.
func (s Size[T]) Neg() Size[T] {
	return Size[T]{Width: -s.Width, Height: -s.Height}
}

// Mul multiplies the width and height by rhs.
func (s Size[T]) Mul(rhs T) Size[T] {
	return Size[T]{Width: s.Width * rhs, Height: s.Height * rhs}
}

// MulAssign multiplies the width and height of s by rhs.
func (s *Size[T]) MulAssign(rhs T) {
	s.Width = s.Width * rhs
	s.Height = s.Height * rhs
}

// Div divides the width and height by rhs.
func (s Size[T]) Div(rhs T) Size[T] {
	return Size[T]{Width: s.Width / rhs, Height: s.Height / rhs}
}

// ToPoint converts the size to a Point with x width and y height.
func (s Size[T]) ToPoint() Point[T] {
	return Point[T]{X: s.Width, Y: s.Height}
}

// ToPixels converts the size from physical to logical pixels.
func (s Size[T]) ToPixels(scaleFactor float32) Size[Pixels] {
	return NewSize(Pixels(float32(s.Width)/scaleFactor), Pixels(float32(s.Height)/scaleFactor))
}

// ToDevicePixels converts the size from logical to physical pixels.
func (s Size[T]) ToDevicePixels(scaleFactor float32) Size[DevicePixels] {
	return NewSize(roundToDevicePixels(s.Width, scaleFactor), roundToDevicePixels(s.Height, scaleFactor))
}

// String formats the size as width × height.
func (s Size[T]) String() string {
	return fmt.Sprintf("%v × %v", s.Width, s.Height)
}

// GoString formats the size like its Rust Debug output.
func (s Size[T]) GoString() string {
	return fmt.Sprintf("Size { %v × %v }", s.Width, s.Height)
}

// SizeOf is Size for non-numeric contents (D-GEOM).
//
//gpui:refineable Debug, PartialEq, Serialize
type SizeOf[T any] struct {
	Width  T `json:"width"`
	Height T `json:"height"`
}

// NewSizeOf constructs a SizeOf with the given width and height.
func NewSizeOf[T any](width, height T) SizeOf[T] {
	return SizeOf[T]{Width: width, Height: height}
}

// Map applies f to the width and height, producing a SizeOf[U].
func (s SizeOf[T]) Map[U any](f func(T) U) SizeOf[U] {
	return SizeOf[U]{Width: f(s.Width), Height: f(s.Height)}
}

// MapNumber applies f to the width and height, producing a Size[U].
func (s SizeOf[T]) MapNumber[U Number](f func(T) U) Size[U] {
	return Size[U]{Width: f(s.Width), Height: f(s.Height)}
}

// Along returns the dimension along the given axis.
func (s SizeOf[T]) Along(axis Axis) T {
	switch axis {
	case AxisHorizontal:
		return s.Width
	default:
		return s.Height
	}
}

// ApplyAlong applies f to the dimension along the given axis.
func (s SizeOf[T]) ApplyAlong(axis Axis, f func(T) T) SizeOf[T] {
	switch axis {
	case AxisHorizontal:
		return SizeOf[T]{Width: f(s.Width), Height: s.Height}
	default:
		return SizeOf[T]{Width: s.Width, Height: f(s.Height)}
	}
}

// ToPoint converts the size to a PointOf with x width and y height.
func (s SizeOf[T]) ToPoint() PointOf[T] {
	return PointOf[T]{X: s.Width, Y: s.Height}
}

// String formats the size as width × height.
func (s SizeOf[T]) String() string {
	return fmt.Sprintf("%v × %v", s.Width, s.Height)
}

// GoString formats the size like its Rust Debug output.
func (s SizeOf[T]) GoString() string {
	return fmt.Sprintf("Size { %v × %v }", s.Width, s.Height)
}

// Bounds is a rectangular area with an origin point and a size.
//
//gpui:refineable Debug
type Bounds[T Number] struct {
	Origin Point[T] `json:"origin"`
	Size   Size[T]  `json:"size"`
}

// NewBounds creates a Bounds with the given origin and size.
func NewBounds[T Number](origin Point[T], size Size[T]) Bounds[T] {
	return Bounds[T]{Origin: origin, Size: size}
}

// BoundsCentered generates bounds of the given size centered on the display, or on the primary display when displayID is None or unknown.
func BoundsCentered(displayID Option[DisplayID], size Size[Pixels], cx *App) Bounds[Pixels] {
	var display PlatformDisplay
	if id, ok := displayID.Get(); ok {
		display = cx.FindDisplay(id)
	}
	if display == nil {
		display = cx.PrimaryDisplay()
	}
	if display != nil {
		visibleBounds := display.VisibleBounds()
		return BoundsCenteredAt(visibleBounds.Center(), size.Min(visibleBounds.Size))
	}
	return Bounds[Pixels]{Origin: NewPoint[Pixels](0, 0), Size: size}
}

// BoundsMaximized generates maximized bounds for the display, or for the primary display when displayID is None or unknown.
func BoundsMaximized(displayID Option[DisplayID], cx *App) Bounds[Pixels] {
	var display PlatformDisplay
	if id, ok := displayID.Get(); ok {
		display = cx.FindDisplay(id)
	}
	if display == nil {
		display = cx.PrimaryDisplay()
	}
	if display != nil {
		return display.Bounds()
	}
	return Bounds[Pixels]{Origin: NewPoint[Pixels](0, 0), Size: NewSize[Pixels](1024, 768)}
}

// BoundsFromCorners constructs bounds from the top left and bottom right corners.
func BoundsFromCorners[T Number](topLeft, bottomRight Point[T]) Bounds[T] {
	origin := Point[T]{X: topLeft.X, Y: topLeft.Y}
	size := Size[T]{Width: bottomRight.X - topLeft.X, Height: bottomRight.Y - topLeft.Y}
	return Bounds[T]{Origin: origin, Size: size}
}

// BoundsFromAnchorAndSize constructs bounds of the given size whose corner is placed at origin.
func BoundsFromAnchorAndSize[T Number](corner Anchor, origin Point[T], size Size[T]) Bounds[T] {
	switch corner {
	case AnchorTopLeft:
	case AnchorTopRight:
		origin = Point[T]{X: origin.X - size.Width, Y: origin.Y}
	case AnchorBottomLeft:
		origin = Point[T]{X: origin.X, Y: origin.Y - size.Height}
	case AnchorBottomRight:
		origin = Point[T]{X: origin.X - size.Width, Y: origin.Y - size.Height}
	case AnchorTopCenter:
		origin = Point[T]{X: origin.X - half(size.Width), Y: origin.Y}
	case AnchorBottomCenter:
		origin = Point[T]{X: origin.X - half(size.Width), Y: origin.Y - size.Height}
	case AnchorLeftCenter:
		origin = Point[T]{X: origin.X, Y: origin.Y - half(size.Height)}
	case AnchorRightCenter:
		origin = Point[T]{X: origin.X - size.Width, Y: origin.Y - half(size.Height)}
	}
	return Bounds[T]{Origin: origin, Size: size}
}

// BoundsCenteredAt creates bounds of the given size centered at center.
func BoundsCenteredAt[T Number](center Point[T], size Size[T]) Bounds[T] {
	origin := Point[T]{X: center.X - half(size.Width), Y: center.Y - half(size.Height)}
	return NewBounds(origin, size)
}

// TopCenter returns the top center point of the bounds.
func (b Bounds[T]) TopCenter() Point[T] {
	return Point[T]{X: b.Origin.X + half(b.Size.Width), Y: b.Origin.Y}
}

// BottomCenter returns the bottom center point of the bounds.
func (b Bounds[T]) BottomCenter() Point[T] {
	return Point[T]{X: b.Origin.X + half(b.Size.Width), Y: b.Origin.Y + b.Size.Height}
}

// LeftCenter returns the left center point of the bounds.
func (b Bounds[T]) LeftCenter() Point[T] {
	return Point[T]{X: b.Origin.X, Y: b.Origin.Y + half(b.Size.Height)}
}

// RightCenter returns the right center point of the bounds.
func (b Bounds[T]) RightCenter() Point[T] {
	return Point[T]{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y + half(b.Size.Height)}
}

// Intersects reports whether b and other overlap.
func (b Bounds[T]) Intersects(other Bounds[T]) bool {
	myLowerRight := b.BottomRight()
	theirLowerRight := other.BottomRight()
	return geomLT(b.Origin.X, theirLowerRight.X) &&
		geomGT(myLowerRight.X, other.Origin.X) &&
		geomLT(b.Origin.Y, theirLowerRight.Y) &&
		geomGT(myLowerRight.Y, other.Origin.Y)
}

// Center returns the center point of the bounds.
func (b Bounds[T]) Center() Point[T] {
	return Point[T]{X: b.Origin.X + half(b.Size.Width), Y: b.Origin.Y + half(b.Size.Height)}
}

// HalfPerimeter returns the sum of the width and the height.
func (b Bounds[T]) HalfPerimeter() T {
	return b.Size.Width + b.Size.Height
}

// Dilate expands the bounds by amount in all directions.
func (b Bounds[T]) Dilate(amount T) Bounds[T] {
	doubleAmount := amount + amount
	return Bounds[T]{
		Origin: b.Origin.Sub(NewPoint(amount, amount)),
		Size:   b.Size.Add(NewSize(doubleAmount, doubleAmount)),
	}
}

// Extend extends the bounds by a different amount in each direction.
func (b Bounds[T]) Extend(amount Edges[T]) Bounds[T] {
	return Bounds[T]{
		Origin: b.Origin.Sub(NewPoint(amount.Left, amount.Top)),
		Size:   b.Size.Add(NewSize(amount.Left+amount.Right, amount.Top+amount.Bottom)),
	}
}

// Inset shrinks the bounds by amount; it is Dilate with the amount negated.
func (b Bounds[T]) Inset(amount T) Bounds[T] {
	return b.Dilate(-amount)
}

// Intersect returns the overlapping region of b and other, with a zero size when they do not intersect.
func (b Bounds[T]) Intersect(other Bounds[T]) Bounds[T] {
	upperLeft := b.Origin.Max(other.Origin)
	bottomRight := b.BottomRight().Min(other.BottomRight()).Max(upperLeft)
	return BoundsFromCorners(upperLeft, bottomRight)
}

// Union returns the smallest bounds containing both b and other.
func (b Bounds[T]) Union(other Bounds[T]) Bounds[T] {
	topLeft := b.Origin.Min(other.Origin)
	bottomRight := b.BottomRight().Max(other.BottomRight())
	return BoundsFromCorners(topLeft, bottomRight)
}

// Join returns the smallest bounds covering the area of both; an empty bounds covers nothing.
func (b Bounds[T]) Join(other Bounds[T]) Bounds[T] {
	if b.IsEmpty() {
		return other
	} else if other.IsEmpty() {
		return b
	}
	return b.Union(other)
}

// SpaceWithin computes the space available within outer bounds.
func (b Bounds[T]) SpaceWithin(outer Bounds[T]) Edges[T] {
	return Edges[T]{
		Top:    b.Top() - outer.Top(),
		Right:  outer.Right() - b.Right(),
		Bottom: outer.Bottom() - b.Bottom(),
		Left:   b.Left() - outer.Left(),
	}
}

// Mul multiplies the origin and size by rhs.
func (b Bounds[T]) Mul(rhs T) Bounds[T] {
	return Bounds[T]{Origin: b.Origin.Mul(rhs), Size: b.Size.Mul(rhs)}
}

// MulAssign multiplies the origin and size of b by rhs.
func (b *Bounds[T]) MulAssign(rhs T) {
	b.Origin.MulAssign(rhs)
	b.Size.MulAssign(rhs)
}

// Div divides the origin and size by rhs.
func (b Bounds[T]) Div(rhs T) Bounds[T] {
	return Bounds[T]{Origin: b.Origin.Div(rhs), Size: b.Size.Div(rhs)}
}

// Add translates the bounds by rhs.
func (b Bounds[T]) Add(rhs Point[T]) Bounds[T] {
	return Bounds[T]{Origin: b.Origin.Add(rhs), Size: b.Size}
}

// Sub translates the bounds by -rhs.
func (b Bounds[T]) Sub(rhs Point[T]) Bounds[T] {
	return Bounds[T]{Origin: b.Origin.Sub(rhs), Size: b.Size}
}

// Top returns the y coordinate of the top edge.
func (b Bounds[T]) Top() T {
	return b.Origin.Y
}

// Bottom returns the y coordinate of the bottom edge.
func (b Bounds[T]) Bottom() T {
	return b.Origin.Y + b.Size.Height
}

// Left returns the x coordinate of the left edge.
func (b Bounds[T]) Left() T {
	return b.Origin.X
}

// Right returns the x coordinate of the right edge.
func (b Bounds[T]) Right() T {
	return b.Origin.X + b.Size.Width
}

// TopRight returns the top right corner point.
func (b Bounds[T]) TopRight() Point[T] {
	return Point[T]{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y}
}

// BottomRight returns the bottom right corner point.
func (b Bounds[T]) BottomRight() Point[T] {
	return Point[T]{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y + b.Size.Height}
}

// BottomLeft returns the bottom left corner point.
func (b Bounds[T]) BottomLeft() Point[T] {
	return Point[T]{X: b.Origin.X, Y: b.Origin.Y + b.Size.Height}
}

// Corner returns the point of the bounds identified by corner.
func (b Bounds[T]) Corner(corner Anchor) Point[T] {
	switch corner {
	case AnchorTopRight:
		return b.TopRight()
	case AnchorBottomLeft:
		return b.BottomLeft()
	case AnchorBottomRight:
		return b.BottomRight()
	case AnchorTopCenter:
		return b.TopCenter()
	case AnchorBottomCenter:
		return b.BottomCenter()
	case AnchorLeftCenter:
		return b.LeftCenter()
	case AnchorRightCenter:
		return b.RightCenter()
	default:
		return b.Origin
	}
}

// Contains reports whether point lies within the bounds; the right and bottom edges are exclusive.
func (b Bounds[T]) Contains(point Point[T]) bool {
	return geomGE(point.X, b.Origin.X) &&
		geomLT(point.X, b.Origin.X+b.Size.Width) &&
		geomGE(point.Y, b.Origin.Y) &&
		geomLT(point.Y, b.Origin.Y+b.Size.Height)
}

// IsContainedWithin reports whether the origin and bottom right corner of b lie within other.
func (b Bounds[T]) IsContainedWithin(other Bounds[T]) bool {
	return other.Contains(b.Origin) && other.Contains(b.BottomRight())
}

// Map applies f to the origin and size, producing a Bounds[U].
func (b Bounds[T]) Map[U Number](f func(T) U) Bounds[U] {
	return Bounds[U]{Origin: b.Origin.Map[U](f), Size: b.Size.Map[U](f)}
}

// MapOrigin applies f to the origin.
func (b Bounds[T]) MapOrigin(f func(T) T) Bounds[T] {
	return Bounds[T]{Origin: b.Origin.Map[T](f), Size: b.Size}
}

// MapSize applies f to the size.
func (b Bounds[T]) MapSize(f func(T) T) Bounds[T] {
	return Bounds[T]{Origin: b.Origin, Size: b.Size.Map[T](f)}
}

// Localize converts point to the coordinate space of the bounds; ok is false when the bounds do not contain it.
func (b Bounds[T]) Localize(point Point[T]) (Point[T], bool) {
	if !b.Contains(point) {
		return Point[T]{}, false
	}
	return point.RelativeTo(b.Origin), true
}

// IsEmpty reports whether the width or the height is less than or equal to zero.
func (b Bounds[T]) IsEmpty() bool {
	var zero T
	return geomLE(b.Size.Width, zero) || geomLE(b.Size.Height, zero)
}

// String formats the bounds as origin - bottom right (size width × height).
func (b Bounds[T]) String() string {
	return fmt.Sprintf("%v - %v (size %v)", b.Origin, b.BottomRight(), b.Size)
}

// GoString formats the bounds like its Rust Debug output.
func (b Bounds[T]) GoString() string {
	return fmt.Sprintf("Bounds { origin: %#v, size: %#v }", b.Origin, b.Size)
}

// Scale scales the bounds by factor, producing scaled pixels.
func (b Bounds[T]) Scale(factor float32) Bounds[ScaledPixels] {
	return Bounds[ScaledPixels]{Origin: b.Origin.Scale(factor), Size: b.Size.Scale(factor)}
}

// ToDevicePixels converts the bounds from logical to physical pixels.
func (b Bounds[T]) ToDevicePixels(factor float32) Bounds[DevicePixels] {
	return Bounds[DevicePixels]{
		Origin: NewPoint(roundToDevicePixels(b.Origin.X, factor), roundToDevicePixels(b.Origin.Y, factor)),
		Size:   b.Size.ToDevicePixels(factor),
	}
}

// ToPixels converts the bounds from physical to logical pixels.
func (b Bounds[T]) ToPixels(scaleFactor float32) Bounds[Pixels] {
	return Bounds[Pixels]{
		Origin: NewPoint(Pixels(float32(b.Origin.X)/scaleFactor), Pixels(float32(b.Origin.Y)/scaleFactor)),
		Size:   b.Size.ToPixels(scaleFactor),
	}
}

// Edges holds a value for each edge of a box, such as padding or margin.
//
//gpui:refineable Debug, PartialEq, Serialize
type Edges[T Number] struct {
	Top    T `json:"top"`
	Right  T `json:"right"`
	Bottom T `json:"bottom"`
	Left   T `json:"left"`
}

// EdgesAll constructs Edges with every side set to value.
func EdgesAll[T Number](value T) Edges[T] {
	return Edges[T]{Top: value, Right: value, Bottom: value, Left: value}
}

// Add adds two edges component-wise.
func (e Edges[T]) Add(rhs Edges[T]) Edges[T] {
	return Edges[T]{Top: e.Top + rhs.Top, Right: e.Right + rhs.Right, Bottom: e.Bottom + rhs.Bottom, Left: e.Left + rhs.Left}
}

// Mul multiplies two edges component-wise.
func (e Edges[T]) Mul(rhs Edges[T]) Edges[T] {
	return Edges[T]{Top: e.Top * rhs.Top, Right: e.Right * rhs.Right, Bottom: e.Bottom * rhs.Bottom, Left: e.Left * rhs.Left}
}

// MulAssign multiplies every side of e by rhs.
func (e *Edges[T]) MulAssign(rhs T) {
	e.Top = e.Top * rhs
	e.Right = e.Right * rhs
	e.Bottom = e.Bottom * rhs
	e.Left = e.Left * rhs
}

// Map applies f to each side, producing an Edges[U].
func (e Edges[T]) Map[U Number](f func(T) U) Edges[U] {
	return Edges[U]{Top: f(e.Top), Right: f(e.Right), Bottom: f(e.Bottom), Left: f(e.Left)}
}

// MapOf applies f to each side, producing an EdgesOf[U].
func (e Edges[T]) MapOf[U any](f func(T) U) EdgesOf[U] {
	return EdgesOf[U]{Top: f(e.Top), Right: f(e.Right), Bottom: f(e.Bottom), Left: f(e.Left)}
}

// Any reports whether predicate holds for any side.
func (e Edges[T]) Any(predicate func(T) bool) bool {
	return predicate(e.Top) || predicate(e.Right) || predicate(e.Bottom) || predicate(e.Left)
}

// Scale scales every side by factor, producing scaled pixels.
func (e Edges[T]) Scale(factor float32) Edges[ScaledPixels] {
	return Edges[ScaledPixels]{
		Top:    scalePixels(e.Top, factor),
		Right:  scalePixels(e.Right, factor),
		Bottom: scalePixels(e.Bottom, factor),
		Left:   scalePixels(e.Left, factor),
	}
}

// Max returns the maximum value of any side.
func (e Edges[T]) Max() T {
	return geomMax(geomMax(geomMax(e.Top, e.Right), e.Bottom), e.Left)
}

// Along returns the sum of the two sides along the given axis.
func (e Edges[T]) Along(axis Axis) T {
	switch axis {
	case AxisHorizontal:
		return e.Left + e.Right
	default:
		return e.Top + e.Bottom
	}
}

// ApplyAlong applies f to both sides along the given axis.
func (e Edges[T]) ApplyAlong(axis Axis, f func(T) T) Edges[T] {
	switch axis {
	case AxisHorizontal:
		return Edges[T]{Top: e.Top, Bottom: e.Bottom, Right: f(e.Right), Left: f(e.Left)}
	default:
		return Edges[T]{Top: f(e.Top), Bottom: f(e.Bottom), Right: e.Right, Left: e.Left}
	}
}

// String formats the edges like their Rust Debug output.
func (e Edges[T]) String() string {
	return fmt.Sprintf("Edges { top: %v, right: %v, bottom: %v, left: %v }", e.Top, e.Right, e.Bottom, e.Left)
}

// EdgesOf is Edges for non-numeric contents (D-GEOM).
//
//gpui:refineable Debug, PartialEq, Serialize
type EdgesOf[T any] struct {
	Top    T `json:"top"`
	Right  T `json:"right"`
	Bottom T `json:"bottom"`
	Left   T `json:"left"`
}

// EdgesOfAll constructs EdgesOf with every side set to value.
func EdgesOfAll[T any](value T) EdgesOf[T] {
	return EdgesOf[T]{Top: value, Right: value, Bottom: value, Left: value}
}

// Map applies f to each side, producing an EdgesOf[U].
func (e EdgesOf[T]) Map[U any](f func(T) U) EdgesOf[U] {
	return EdgesOf[U]{Top: f(e.Top), Right: f(e.Right), Bottom: f(e.Bottom), Left: f(e.Left)}
}

// MapNumber applies f to each side, producing an Edges[U].
func (e EdgesOf[T]) MapNumber[U Number](f func(T) U) Edges[U] {
	return Edges[U]{Top: f(e.Top), Right: f(e.Right), Bottom: f(e.Bottom), Left: f(e.Left)}
}

// Any reports whether predicate holds for any side.
func (e EdgesOf[T]) Any(predicate func(T) bool) bool {
	return predicate(e.Top) || predicate(e.Right) || predicate(e.Bottom) || predicate(e.Left)
}

// String formats the edges like their Rust Debug output.
func (e EdgesOf[T]) String() string {
	return fmt.Sprintf("Edges { top: %v, right: %v, bottom: %v, left: %v }", e.Top, e.Right, e.Bottom, e.Left)
}

// Anchor identifies a reference point on a 2D box, used to anchor positioned elements.
type Anchor uint8

const (
	// AnchorTopLeft is the top left corner.
	AnchorTopLeft Anchor = iota
	// AnchorTopRight is the top right corner.
	AnchorTopRight
	// AnchorBottomLeft is the bottom left corner.
	AnchorBottomLeft
	// AnchorBottomRight is the bottom right corner.
	AnchorBottomRight
	// AnchorTopCenter is the top center position.
	AnchorTopCenter
	// AnchorBottomCenter is the bottom center position.
	AnchorBottomCenter
	// AnchorLeftCenter is the left center position.
	AnchorLeftCenter
	// AnchorRightCenter is the right center position.
	AnchorRightCenter
)

// Opposite returns the directly opposite anchor.
func (a Anchor) Opposite() Anchor {
	switch a {
	case AnchorTopLeft:
		return AnchorBottomRight
	case AnchorTopRight:
		return AnchorBottomLeft
	case AnchorBottomLeft:
		return AnchorTopRight
	case AnchorBottomRight:
		return AnchorTopLeft
	case AnchorTopCenter:
		return AnchorBottomCenter
	case AnchorBottomCenter:
		return AnchorTopCenter
	case AnchorLeftCenter:
		return AnchorRightCenter
	default:
		return AnchorLeftCenter
	}
}

// OtherSideAlong returns the anchor across from this one, moving along axis.
func (a Anchor) OtherSideAlong(axis Axis) Anchor {
	switch axis {
	case AxisVertical:
		switch a {
		case AnchorTopLeft:
			return AnchorBottomLeft
		case AnchorTopRight:
			return AnchorBottomRight
		case AnchorBottomLeft:
			return AnchorTopLeft
		case AnchorBottomRight:
			return AnchorTopRight
		case AnchorTopCenter:
			return AnchorBottomCenter
		case AnchorBottomCenter:
			return AnchorTopCenter
		case AnchorLeftCenter:
			return AnchorLeftCenter
		default:
			return AnchorRightCenter
		}
	default:
		switch a {
		case AnchorTopLeft:
			return AnchorTopRight
		case AnchorTopRight:
			return AnchorTopLeft
		case AnchorBottomLeft:
			return AnchorBottomRight
		case AnchorBottomRight:
			return AnchorBottomLeft
		case AnchorTopCenter:
			return AnchorTopCenter
		case AnchorBottomCenter:
			return AnchorBottomCenter
		case AnchorLeftCenter:
			return AnchorRightCenter
		default:
			return AnchorLeftCenter
		}
	}
}

// IsCenter reports whether the anchor is center-positioned.
func (a Anchor) IsCenter() bool {
	switch a {
	case AnchorTopCenter, AnchorBottomCenter, AnchorLeftCenter, AnchorRightCenter:
		return true
	}
	return false
}

// IsBottom reports whether the anchor is bottom-positioned.
func (a Anchor) IsBottom() bool {
	switch a {
	case AnchorBottomCenter, AnchorBottomLeft, AnchorBottomRight:
		return true
	}
	return false
}

// String formats the anchor like its Rust Debug output.
func (a Anchor) String() string {
	switch a {
	case AnchorTopLeft:
		return "TopLeft"
	case AnchorTopRight:
		return "TopRight"
	case AnchorBottomLeft:
		return "BottomLeft"
	case AnchorBottomRight:
		return "BottomRight"
	case AnchorTopCenter:
		return "TopCenter"
	case AnchorBottomCenter:
		return "BottomCenter"
	case AnchorLeftCenter:
		return "LeftCenter"
	case AnchorRightCenter:
		return "RightCenter"
	}
	return fmt.Sprintf("Anchor(%d)", uint8(a))
}

// Corners holds a value for each corner of a box, such as corner radii.
//
//gpui:refineable Debug, PartialEq, Serialize
type Corners[T Number] struct {
	TopLeft     T `json:"top_left"`
	TopRight    T `json:"top_right"`
	BottomRight T `json:"bottom_right"`
	BottomLeft  T `json:"bottom_left"`
}

// CornersAll constructs Corners with every corner set to value.
func CornersAll[T Number](value T) Corners[T] {
	return Corners[T]{TopLeft: value, TopRight: value, BottomRight: value, BottomLeft: value}
}

// Corner returns the requested corner value; center anchors average the two adjacent corners.
func (c Corners[T]) Corner(corner Anchor) T {
	switch corner {
	case AnchorTopLeft:
		return c.TopLeft
	case AnchorTopRight:
		return c.TopRight
	case AnchorBottomLeft:
		return c.BottomLeft
	case AnchorBottomRight:
		return c.BottomRight
	case AnchorTopCenter:
		return half(c.TopLeft + c.TopRight)
	case AnchorBottomCenter:
		return half(c.BottomLeft + c.BottomRight)
	case AnchorLeftCenter:
		return half(c.TopLeft + c.BottomLeft)
	default:
		return half(c.TopRight + c.BottomRight)
	}
}

// Scale scales every corner by factor, producing scaled pixels.
func (c Corners[T]) Scale(factor float32) Corners[ScaledPixels] {
	return Corners[ScaledPixels]{
		TopLeft:     scalePixels(c.TopLeft, factor),
		TopRight:    scalePixels(c.TopRight, factor),
		BottomRight: scalePixels(c.BottomRight, factor),
		BottomLeft:  scalePixels(c.BottomLeft, factor),
	}
}

// Max returns the maximum value of any corner.
func (c Corners[T]) Max() T {
	return geomOrdMax(geomOrdMax(geomOrdMax(c.TopLeft, c.TopRight), c.BottomRight), c.BottomLeft)
}

// ClampRadiiForQuadSize clamps corner radii to at most half the shortest side of a quad.
func (c Corners[T]) ClampRadiiForQuadSize(size Size[T]) Corners[T] {
	maxValue := geomOrdMin(size.Width, size.Height) / 2
	return Corners[T]{
		TopLeft:     geomOrdMin(c.TopLeft, maxValue),
		TopRight:    geomOrdMin(c.TopRight, maxValue),
		BottomRight: geomOrdMin(c.BottomRight, maxValue),
		BottomLeft:  geomOrdMin(c.BottomLeft, maxValue),
	}
}

// Map applies f to each corner, producing a Corners[U].
func (c Corners[T]) Map[U Number](f func(T) U) Corners[U] {
	return Corners[U]{TopLeft: f(c.TopLeft), TopRight: f(c.TopRight), BottomRight: f(c.BottomRight), BottomLeft: f(c.BottomLeft)}
}

// MapOf applies f to each corner, producing a CornersOf[U].
func (c Corners[T]) MapOf[U any](f func(T) U) CornersOf[U] {
	return CornersOf[U]{TopLeft: f(c.TopLeft), TopRight: f(c.TopRight), BottomRight: f(c.BottomRight), BottomLeft: f(c.BottomLeft)}
}

// Mul multiplies two corners component-wise.
func (c Corners[T]) Mul(rhs Corners[T]) Corners[T] {
	return Corners[T]{
		TopLeft:     c.TopLeft * rhs.TopLeft,
		TopRight:    c.TopRight * rhs.TopRight,
		BottomRight: c.BottomRight * rhs.BottomRight,
		BottomLeft:  c.BottomLeft * rhs.BottomLeft,
	}
}

// MulAssign multiplies every corner of c by rhs.
func (c *Corners[T]) MulAssign(rhs T) {
	c.TopLeft = c.TopLeft * rhs
	c.TopRight = c.TopRight * rhs
	c.BottomRight = c.BottomRight * rhs
	c.BottomLeft = c.BottomLeft * rhs
}

// IsZero reports whether every corner is zero.
func (c Corners[T]) IsZero() bool {
	return c.TopLeft == 0 && c.TopRight == 0 && c.BottomRight == 0 && c.BottomLeft == 0
}

// String formats the corners like their Rust Debug output.
func (c Corners[T]) String() string {
	return fmt.Sprintf("Corners { top_left: %v, top_right: %v, bottom_right: %v, bottom_left: %v }", c.TopLeft, c.TopRight, c.BottomRight, c.BottomLeft)
}

// CornersOf is Corners for non-numeric contents (D-GEOM).
//
//gpui:refineable Debug, PartialEq, Serialize
type CornersOf[T any] struct {
	TopLeft     T `json:"top_left"`
	TopRight    T `json:"top_right"`
	BottomRight T `json:"bottom_right"`
	BottomLeft  T `json:"bottom_left"`
}

// CornersOfAll constructs CornersOf with every corner set to value.
func CornersOfAll[T any](value T) CornersOf[T] {
	return CornersOf[T]{TopLeft: value, TopRight: value, BottomRight: value, BottomLeft: value}
}

// Map applies f to each corner, producing a CornersOf[U].
func (c CornersOf[T]) Map[U any](f func(T) U) CornersOf[U] {
	return CornersOf[U]{TopLeft: f(c.TopLeft), TopRight: f(c.TopRight), BottomRight: f(c.BottomRight), BottomLeft: f(c.BottomLeft)}
}

// MapNumber applies f to each corner, producing a Corners[U].
func (c CornersOf[T]) MapNumber[U Number](f func(T) U) Corners[U] {
	return Corners[U]{TopLeft: f(c.TopLeft), TopRight: f(c.TopRight), BottomRight: f(c.BottomRight), BottomLeft: f(c.BottomLeft)}
}

// String formats the corners like their Rust Debug output.
func (c CornersOf[T]) String() string {
	return fmt.Sprintf("Corners { top_left: %v, top_right: %v, bottom_right: %v, bottom_left: %v }", c.TopLeft, c.TopRight, c.BottomRight, c.BottomLeft)
}

// CornersAbsoluteLengthToPixels converts each corner to pixels using rem size remSize.
func CornersAbsoluteLengthToPixels(c CornersOf[AbsoluteLength], remSize Pixels) Corners[Pixels] {
	return Corners[Pixels]{
		TopLeft:     c.TopLeft.ToPixels(remSize),
		TopRight:    c.TopRight.ToPixels(remSize),
		BottomRight: c.BottomRight.ToPixels(remSize),
		BottomLeft:  c.BottomLeft.ToPixels(remSize),
	}
}

// SizeDefiniteLengthFromPixels converts a pixel size to definite lengths.
func SizeDefiniteLengthFromPixels(size Size[Pixels]) SizeOf[DefiniteLength] {
	return SizeOf[DefiniteLength]{Width: size.Width.ToDefiniteLength(), Height: size.Height.ToDefiniteLength()}
}

// SizeAbsoluteLengthFromPixels converts a pixel size to absolute lengths.
func SizeAbsoluteLengthFromPixels(size Size[Pixels]) SizeOf[AbsoluteLength] {
	return SizeOf[AbsoluteLength]{Width: size.Width.ToAbsoluteLength(), Height: size.Height.ToAbsoluteLength()}
}

// SizeLengthFull returns a size whose width and height are 100% of the parent.
func SizeLengthFull() SizeOf[Length] {
	return SizeOf[Length]{Width: Relative(1.).ToLength(), Height: Relative(1.).ToLength()}
}

// SizeLengthAuto returns a size whose width and height are auto.
func SizeLengthAuto() SizeOf[Length] {
	return SizeOf[Length]{Width: LengthAuto(), Height: LengthAuto()}
}

// EdgesLengthAuto returns edges that are all auto.
func EdgesLengthAuto() EdgesOf[Length] {
	return EdgesOf[Length]{Top: LengthAuto(), Right: LengthAuto(), Bottom: LengthAuto(), Left: LengthAuto()}
}

// EdgesLengthZero returns edges that are all zero pixels.
func EdgesLengthZero() EdgesOf[Length] {
	return EdgesOf[Length]{Top: Px(0.).ToLength(), Right: Px(0.).ToLength(), Bottom: Px(0.).ToLength(), Left: Px(0.).ToLength()}
}

// EdgesDefiniteLengthZero returns edges that are all zero pixels.
func EdgesDefiniteLengthZero() EdgesOf[DefiniteLength] {
	return EdgesOf[DefiniteLength]{Top: Px(0.).ToDefiniteLength(), Right: Px(0.).ToDefiniteLength(), Bottom: Px(0.).ToDefiniteLength(), Left: Px(0.).ToDefiniteLength()}
}

// EdgesDefiniteLengthToPixels converts each edge to pixels relative to parentSize and remSize.
func EdgesDefiniteLengthToPixels(e EdgesOf[DefiniteLength], parentSize SizeOf[AbsoluteLength], remSize Pixels) Edges[Pixels] {
	return Edges[Pixels]{
		Top:    e.Top.ToPixels(parentSize.Height, remSize),
		Right:  e.Right.ToPixels(parentSize.Width, remSize),
		Bottom: e.Bottom.ToPixels(parentSize.Height, remSize),
		Left:   e.Left.ToPixels(parentSize.Width, remSize),
	}
}

// EdgesAbsoluteLengthZero returns edges that are all zero pixels.
func EdgesAbsoluteLengthZero() EdgesOf[AbsoluteLength] {
	return EdgesOf[AbsoluteLength]{Top: Px(0.).ToAbsoluteLength(), Right: Px(0.).ToAbsoluteLength(), Bottom: Px(0.).ToAbsoluteLength(), Left: Px(0.).ToAbsoluteLength()}
}

// EdgesAbsoluteLengthToPixels converts each edge to pixels using rem size remSize.
func EdgesAbsoluteLengthToPixels(e EdgesOf[AbsoluteLength], remSize Pixels) Edges[Pixels] {
	return Edges[Pixels]{
		Top:    e.Top.ToPixels(remSize),
		Right:  e.Right.ToPixels(remSize),
		Bottom: e.Bottom.ToPixels(remSize),
		Left:   e.Left.ToPixels(remSize),
	}
}

// Radians is an angle in radians.
type Radians float32

// String formats the angle like its Rust Debug output.
func (r Radians) String() string {
	return "Radians(" + rustDebugF32(float32(r)) + ")"
}

// Percentage is a fraction of a whole, between 0 and 1.
type Percentage float32

// NewPercentage returns a percentage; it panics in debug builds when value is outside 0..=1.
func NewPercentage(value float32) Percentage {
	if gpuiutil.DebugAssertions && !(value >= 0 && value <= 1) {
		panic("Percentage must be between 0 and 1")
	}
	return Percentage(value)
}

// ToRadians converts a fraction of a full circle to radians.
func (p Percentage) ToRadians() Radians {
	return Radians(float32(float32(p)*math.Pi) * 2.0)
}

// String formats the percentage like its Rust Debug output.
func (p Percentage) String() string {
	return "Percentage(" + rustDebugF32(float32(p)) + ")"
}

// Pixels is a length in logical pixels, the base unit of measurement.
type Pixels float32

const (
	// PixelsZero is zero pixels.
	PixelsZero Pixels = 0
	// PixelsMax is the largest finite pixel value.
	PixelsMax Pixels = math.MaxFloat32
	// PixelsMin is the smallest finite pixel value.
	PixelsMin Pixels = -math.MaxFloat32
)

// AsF32 returns the raw value.
func (p Pixels) AsF32() float32 {
	return float32(p)
}

// ToI32 truncates to an i32, saturating at the bounds and mapping NaN to 0.
func (p Pixels) ToI32() int32 {
	return saturatingF32ToI32(float32(p))
}

// ToU32 truncates to a u32 like Rust's `as u32`.
func (p Pixels) ToU32() uint32 {
	return saturatingF32ToU32(float32(p))
}

// ToUsize truncates to a uint like Rust's `as usize`.
func (p Pixels) ToUsize() uint {
	return uint(saturatingF64ToU64(float64(p)))
}

// Floor rounds down to the nearest whole pixel.
func (p Pixels) Floor() Pixels {
	return Pixels(math.Floor(float64(p)))
}

// Round rounds to the nearest whole pixel, half away from zero.
func (p Pixels) Round() Pixels {
	return Pixels(math.Round(float64(p)))
}

// Ceil rounds up to the nearest whole pixel.
func (p Pixels) Ceil() Pixels {
	return Pixels(math.Ceil(float64(p)))
}

// Scale scales the pixels by factor, producing scaled pixels.
func (p Pixels) Scale(factor float32) ScaledPixels {
	return ScaledPixels(float32(p) * factor)
}

// Pow raises the pixels to exponent.
func (p Pixels) Pow(exponent float32) Pixels {
	return Pixels(math.Pow(float64(p), float64(exponent)))
}

// Abs returns the absolute value.
func (p Pixels) Abs() Pixels {
	return Pixels(math.Abs(float64(p)))
}

// Signum returns 1 for positive values and +0, -1 for negative values and -0, NaN for NaN.
func (p Pixels) Signum() float32 {
	return signumF32(float32(p))
}

// ToF64 returns the value as a float64.
func (p Pixels) ToF64() float64 {
	return float64(p)
}

// Rem returns the floating-point remainder of p / rhs.
func (p Pixels) Rem(rhs Pixels) Pixels {
	return Pixels(math.Mod(float64(p), float64(rhs)))
}

// Cmp orders pixels by partial order, falling back to total order for NaN.
func (p Pixels) Cmp(other Pixels) int {
	c, _ := geomPartialCmp(p, other)
	return c
}

// Half returns half of p.
func (p Pixels) Half() Pixels {
	return p / 2
}

// IsZero reports whether p is zero.
func (p Pixels) IsZero() bool {
	return p == 0
}

// ToLength converts p to a Length.
func (p Pixels) ToLength() Length {
	return LengthDefinite(p.ToDefiniteLength())
}

// ToDefiniteLength converts p to a DefiniteLength.
func (p Pixels) ToDefiniteLength() DefiniteLength {
	return DefiniteLengthAbsolute(p.ToAbsoluteLength())
}

// ToAbsoluteLength converts p to an AbsoluteLength.
func (p Pixels) ToAbsoluteLength() AbsoluteLength {
	return AbsoluteLengthPixels(p)
}

// String formats the pixels like Rust's Display, for example `10px`.
func (p Pixels) String() string {
	return rustDisplayF32(float32(p)) + "px"
}

// ParsePixels parses a number with a `px` suffix.
func ParsePixels(value string) (Pixels, error) {
	number, ok := strings.CutSuffix(value, "px")
	if !ok {
		return 0, errors.New("expected 'px' suffix")
	}
	v, err := parseRustF32(number)
	if err != nil {
		return 0, err
	}
	return Pixels(v), nil
}

// DevicePixels is a length in physical device pixels.
type DevicePixels int32

// ToBytes returns the number of bytes covered by d pixels of bytesPerPixel each.
func (d DevicePixels) ToBytes(bytesPerPixel uint8) uint32 {
	return uint32(d) * uint32(bytesPerPixel)
}

// ToScaledPixels converts d to scaled pixels.
func (d DevicePixels) ToScaledPixels() ScaledPixels {
	return ScaledPixels(float32(d))
}

// Half returns half of d, truncated toward zero.
func (d DevicePixels) Half() DevicePixels {
	return d / 2
}

// IsZero reports whether d is zero.
func (d DevicePixels) IsZero() bool {
	return d == 0
}

// String formats the device pixels like their Rust Debug output.
func (d DevicePixels) String() string {
	return strconv.FormatInt(int64(d), 10) + " px (device)"
}

// ScaledPixels is a length in pixels scaled by the display's scale factor.
type ScaledPixels float32

// AsF32 returns the raw value.
func (s ScaledPixels) AsF32() float32 {
	return float32(s)
}

// Floor rounds down to the nearest whole pixel.
func (s ScaledPixels) Floor() ScaledPixels {
	return ScaledPixels(math.Floor(float64(s)))
}

// Round rounds to the nearest whole pixel, half away from zero.
func (s ScaledPixels) Round() ScaledPixels {
	return ScaledPixels(math.Round(float64(s)))
}

// Ceil rounds up to the nearest whole pixel.
func (s ScaledPixels) Ceil() ScaledPixels {
	return ScaledPixels(math.Ceil(float64(s)))
}

// ToDevicePixels converts s to device pixels, rounding up.
func (s ScaledPixels) ToDevicePixels() DevicePixels {
	return DevicePixels(saturatingF32ToI32(float32(math.Ceil(float64(s)))))
}

// ToU32 truncates to a u32 like Rust's `as u32`.
func (s ScaledPixels) ToU32() uint32 {
	return saturatingF32ToU32(float32(s))
}

// Rem returns the floating-point remainder of s / rhs.
func (s ScaledPixels) Rem(rhs ScaledPixels) ScaledPixels {
	return ScaledPixels(math.Mod(float64(s), float64(rhs)))
}

// Cmp orders scaled pixels by IEEE total order.
func (s ScaledPixels) Cmp(other ScaledPixels) int {
	return totalCmpF32(float32(s), float32(other))
}

// Half returns half of s.
func (s ScaledPixels) Half() ScaledPixels {
	return s / 2
}

// IsZero reports whether s is zero.
func (s ScaledPixels) IsZero() bool {
	return s == 0
}

// String formats the scaled pixels like their Rust Debug output.
func (s ScaledPixels) String() string {
	return rustDisplayF32(float32(s)) + "px (scaled)"
}

// Rems is a length in rems, relative to the window's rem size.
type Rems float32

// RemsZero is zero rems.
const RemsZero Rems = 0

// ToPixels converts r to pixels using rem size remSize.
func (r Rems) ToPixels(remSize Pixels) Pixels {
	return Pixels(float32(r) * float32(remSize))
}

// RemsFromPixels converts length to rems using the window's rem size.
func RemsFromPixels(length Pixels, window *Window) Rems {
	return Rems(float32(length) / float32(window.RemSize()))
}

// Half returns half of r.
func (r Rems) Half() Rems {
	return r / 2
}

// IsZero reports whether r is zero.
func (r Rems) IsZero() bool {
	return r == 0
}

// ToLength converts r to a Length.
func (r Rems) ToLength() Length {
	return LengthDefinite(r.ToDefiniteLength())
}

// ToDefiniteLength converts r to a DefiniteLength.
func (r Rems) ToDefiniteLength() DefiniteLength {
	return DefiniteLengthAbsolute(r.ToAbsoluteLength())
}

// ToAbsoluteLength converts r to an AbsoluteLength.
func (r Rems) ToAbsoluteLength() AbsoluteLength {
	return AbsoluteLengthRems(r)
}

// String formats the rems like Rust's Display, for example `1.5rem`.
func (r Rems) String() string {
	return rustDisplayF32(float32(r)) + "rem"
}

// ParseRems parses a number with a `rem` suffix.
func ParseRems(value string) (Rems, error) {
	number, ok := strings.CutSuffix(value, "rem")
	if !ok {
		return 0, errors.New("expected 'rem' suffix")
	}
	v, err := parseRustF32(number)
	if err != nil {
		return 0, err
	}
	return Rems(v), nil
}

// IntoLength is implemented by every value convertible to a Length (D-INTO).
type IntoLength interface {
	ToLength() Length
}

// IntoDefiniteLength is implemented by every value convertible to a DefiniteLength (D-INTO).
type IntoDefiniteLength interface {
	ToDefiniteLength() DefiniteLength
}

// IntoAbsoluteLength is implemented by every value convertible to an AbsoluteLength (D-INTO).
type IntoAbsoluteLength interface {
	ToAbsoluteLength() AbsoluteLength
}

type absoluteLengthKind uint8

const (
	absoluteLengthPixels absoluteLengthKind = iota
	absoluteLengthRems
)

// AbsoluteLength is a length in pixels or rems. The zero value is zero pixels.
type AbsoluteLength struct {
	kind  absoluteLengthKind
	value float32
}

// AbsoluteLengthPixels returns the Pixels variant.
func AbsoluteLengthPixels(pixels Pixels) AbsoluteLength {
	return AbsoluteLength{kind: absoluteLengthPixels, value: float32(pixels)}
}

// AbsoluteLengthRems returns the Rems variant.
func AbsoluteLengthRems(rems Rems) AbsoluteLength {
	return AbsoluteLength{kind: absoluteLengthRems, value: float32(rems)}
}

// AsPixels returns the length in pixels when it is the Pixels variant.
func (a AbsoluteLength) AsPixels() (Pixels, bool) {
	return Pixels(a.value), a.kind == absoluteLengthPixels
}

// AsRems returns the length in rems when it is the Rems variant.
func (a AbsoluteLength) AsRems() (Rems, bool) {
	return Rems(a.value), a.kind == absoluteLengthRems
}

// IsZero reports whether the length is zero.
func (a AbsoluteLength) IsZero() bool {
	return a.value == 0
}

// ToPixels converts the length to pixels using rem size remSize.
func (a AbsoluteLength) ToPixels(remSize Pixels) Pixels {
	if a.kind == absoluteLengthRems {
		return Rems(a.value).ToPixels(remSize)
	}
	return Pixels(a.value)
}

// ToRems converts the length to rems using rem size remSize.
func (a AbsoluteLength) ToRems(remSize Pixels) Rems {
	if a.kind == absoluteLengthPixels {
		return Rems(a.value / float32(remSize))
	}
	return Rems(a.value)
}

// Neg negates the length.
func (a AbsoluteLength) Neg() AbsoluteLength {
	return AbsoluteLength{kind: a.kind, value: -a.value}
}

// ToLength converts a to a Length.
func (a AbsoluteLength) ToLength() Length {
	return LengthDefinite(DefiniteLengthAbsolute(a))
}

// ToDefiniteLength converts a to a DefiniteLength.
func (a AbsoluteLength) ToDefiniteLength() DefiniteLength {
	return DefiniteLengthAbsolute(a)
}

// ToAbsoluteLength returns a.
func (a AbsoluteLength) ToAbsoluteLength() AbsoluteLength {
	return a
}

// String formats the length like Rust's Display, for example `10px` or `1rem`.
func (a AbsoluteLength) String() string {
	if a.kind == absoluteLengthRems {
		return Rems(a.value).String()
	}
	return Pixels(a.value).String()
}

const expectedAbsoluteLength = "number with 'px' or 'rem' suffix"

// ParseAbsoluteLength parses a number with a `px` or `rem` suffix.
func ParseAbsoluteLength(value string) (AbsoluteLength, error) {
	if pixels, err := ParsePixels(value); err == nil {
		return AbsoluteLengthPixels(pixels), nil
	}
	if rems, err := ParseRems(value); err == nil {
		return AbsoluteLengthRems(rems), nil
	}
	return AbsoluteLength{}, fmt.Errorf("invalid AbsoluteLength '%s', expected %s", value, expectedAbsoluteLength)
}

// MarshalJSONTo encodes the length as its Display string.
func (a AbsoluteLength) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(a.String()))
}

// UnmarshalJSONFrom decodes the length from a string.
func (a *AbsoluteLength) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	s, err := readJSONString(dec, expectedAbsoluteLength)
	if err != nil {
		return err
	}
	v, err := ParseAbsoluteLength(s)
	if err != nil {
		return err
	}
	*a = v
	return nil
}

// JSONSchema describes the length as a string.
func (AbsoluteLength) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{Type: jsonschema.Types{"string"}}
}

// DefiniteLength is an absolute length or a fraction of the parent. The zero value is zero pixels.
type DefiniteLength struct {
	fraction   bool
	absolute   AbsoluteLength
	fractional float32
}

// DefiniteLengthAbsolute returns the Absolute variant.
func DefiniteLengthAbsolute(length AbsoluteLength) DefiniteLength {
	return DefiniteLength{absolute: length}
}

// DefiniteLengthFraction returns the Fraction variant.
func DefiniteLengthFraction(fraction float32) DefiniteLength {
	return DefiniteLength{fraction: true, fractional: fraction}
}

// AsAbsolute returns the absolute length when it is the Absolute variant.
func (d DefiniteLength) AsAbsolute() (AbsoluteLength, bool) {
	return d.absolute, !d.fraction
}

// AsFraction returns the fraction when it is the Fraction variant.
func (d DefiniteLength) AsFraction() (float32, bool) {
	return d.fractional, d.fraction
}

// ToPixels converts the length to pixels relative to baseSize and remSize.
func (d DefiniteLength) ToPixels(baseSize AbsoluteLength, remSize Pixels) Pixels {
	if !d.fraction {
		return d.absolute.ToPixels(remSize)
	}
	if baseSize.kind == absoluteLengthRems {
		return Pixels(float32(Rems(baseSize.value).ToPixels(remSize)) * d.fractional)
	}
	return Pixels(baseSize.value * d.fractional)
}

// IsZero reports whether the length is zero.
func (d DefiniteLength) IsZero() bool {
	if d.fraction {
		return d.fractional == 0
	}
	return d.absolute.IsZero()
}

// Neg negates the length.
func (d DefiniteLength) Neg() DefiniteLength {
	if d.fraction {
		return DefiniteLengthFraction(-d.fractional)
	}
	return DefiniteLengthAbsolute(d.absolute.Neg())
}

// ToLength converts d to a Length.
func (d DefiniteLength) ToLength() Length {
	return LengthDefinite(d)
}

// ToDefiniteLength returns d.
func (d DefiniteLength) ToDefiniteLength() DefiniteLength {
	return d
}

// String formats the length like Rust's Display, for example `10px` or `50%`.
func (d DefiniteLength) String() string {
	if d.fraction {
		return strconv.FormatInt(int64(saturatingF32ToI32(d.fractional*100.0)), 10) + "%"
	}
	return d.absolute.String()
}

const expectedDefiniteLength = "expected number with 'px', 'rem', or '%' suffix"

// ParseDefiniteLength parses a number with a `px`, `rem` or `%` suffix.
func ParseDefiniteLength(value string) (DefiniteLength, error) {
	if percentage, ok := strings.CutSuffix(value, "%"); ok {
		fraction, err := parseRustF32(percentage)
		if err != nil {
			return DefiniteLength{}, fmt.Errorf("invalid DefiniteLength '%s', expected %s: %w", value, expectedDefiniteLength, err)
		}
		return DefiniteLengthFraction(fraction / 100.0), nil
	}
	if absolute, err := ParseAbsoluteLength(value); err == nil {
		return DefiniteLengthAbsolute(absolute), nil
	}
	return DefiniteLength{}, fmt.Errorf("invalid DefiniteLength '%s', expected %s", value, expectedDefiniteLength)
}

// MarshalJSONTo encodes the length as its Display string.
func (d DefiniteLength) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(d.String()))
}

// UnmarshalJSONFrom decodes the length from a string.
func (d *DefiniteLength) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	s, err := readJSONString(dec, expectedDefiniteLength)
	if err != nil {
		return err
	}
	v, err := ParseDefiniteLength(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// JSONSchema describes the length as a string.
func (DefiniteLength) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{Type: jsonschema.Types{"string"}}
}

// Length is a definite length or auto. The zero value is zero pixels.
type Length struct {
	auto     bool
	definite DefiniteLength
}

// LengthDefinite returns the Definite variant.
func LengthDefinite(length DefiniteLength) Length {
	return Length{definite: length}
}

// LengthAuto returns the Auto variant.
func LengthAuto() Length {
	return Length{auto: true}
}

// AsDefinite returns the definite length when it is the Definite variant.
func (l Length) AsDefinite() (DefiniteLength, bool) {
	return l.definite, !l.auto
}

// IsAuto reports whether l is the Auto variant.
func (l Length) IsAuto() bool {
	return l.auto
}

// IsZero reports whether l is a zero definite length.
func (l Length) IsZero() bool {
	return !l.auto && l.definite.IsZero()
}

// ToLength returns l.
func (l Length) ToLength() Length {
	return l
}

// String formats the length like Rust's Display, for example `auto` or `10px`.
func (l Length) String() string {
	if l.auto {
		return "auto"
	}
	return l.definite.String()
}

const expectedLength = "expected 'auto' or number with 'px', 'rem', or '%' suffix"

// ParseLength parses `auto` or a number with a `px`, `rem` or `%` suffix.
func ParseLength(value string) (Length, error) {
	if value == "auto" {
		return LengthAuto(), nil
	}
	if definite, err := ParseDefiniteLength(value); err == nil {
		return LengthDefinite(definite), nil
	}
	return Length{}, fmt.Errorf("invalid Length '%s', expected %s", value, expectedLength)
}

// MarshalJSONTo encodes the length as its Display string.
func (l Length) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(l.String()))
}

// UnmarshalJSONFrom decodes the length from a string.
func (l *Length) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	s, err := readJSONString(dec, expectedLength)
	if err != nil {
		return err
	}
	v, err := ParseLength(s)
	if err != nil {
		return err
	}
	*l = v
	return nil
}

// JSONSchema describes the length as a string.
func (Length) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{Type: jsonschema.Types{"string"}}
}

// Relative returns a fraction of the parent's size.
func Relative(fraction float32) DefiniteLength {
	return DefiniteLengthFraction(fraction)
}

// Phi returns the golden ratio as a relative length.
func Phi() DefiniteLength {
	return Relative(math.Phi)
}

// Px returns a length in pixels.
func Px(pixels float32) Pixels {
	return Pixels(pixels)
}

// Auto returns the auto length.
func Auto() Length {
	return LengthAuto()
}

// GridLocation is the row and column placement of a grid item.
type GridLocation struct {
	Row    GridPlacementRange `json:"row"`
	Column GridPlacementRange `json:"column"`
}

// String formats the location like its Rust Debug output.
func (g GridLocation) String() string {
	return fmt.Sprintf("GridLocation { row: %v, column: %v }", g.Row, g.Column)
}

// GridPlacementRange is upstream's Range<GridPlacement>, the start and end lines of a grid item.
type GridPlacementRange struct {
	Start GridPlacement `json:"start"`
	End   GridPlacement `json:"end"`
}

// String formats the range like its Rust Debug output.
func (r GridPlacementRange) String() string {
	return r.Start.String() + ".." + r.End.String()
}

type gridPlacementKind uint8

const (
	gridPlacementAuto gridPlacementKind = iota
	gridPlacementLine
	gridPlacementSpan
)

// GridPlacement places a grid item on a line, across a span, or automatically. The zero value is Auto.
type GridPlacement struct {
	kind  gridPlacementKind
	value int32
}

// GridPlacementLine returns the Line variant.
func GridPlacementLine(index int16) GridPlacement {
	return GridPlacement{kind: gridPlacementLine, value: int32(index)}
}

// GridPlacementSpan returns the Span variant.
func GridPlacementSpan(span uint16) GridPlacement {
	return GridPlacement{kind: gridPlacementSpan, value: int32(span)}
}

// GridPlacementAuto returns the Auto variant.
func GridPlacementAuto() GridPlacement {
	return GridPlacement{}
}

// AsLine returns the line index when it is the Line variant.
func (g GridPlacement) AsLine() (int16, bool) {
	return int16(g.value), g.kind == gridPlacementLine
}

// AsSpan returns the span when it is the Span variant.
func (g GridPlacement) AsSpan() (uint16, bool) {
	return uint16(g.value), g.kind == gridPlacementSpan
}

// IsAuto reports whether g is the Auto variant.
func (g GridPlacement) IsAuto() bool {
	return g.kind == gridPlacementAuto
}

// String formats the placement like its Rust Debug output.
func (g GridPlacement) String() string {
	switch g.kind {
	case gridPlacementLine:
		return "Line(" + strconv.Itoa(int(g.value)) + ")"
	case gridPlacementSpan:
		return "Span(" + strconv.Itoa(int(g.value)) + ")"
	}
	return "Auto"
}

// MarshalJSONTo encodes the placement in serde's externally tagged form.
func (g GridPlacement) MarshalJSONTo(enc *jsontext.Encoder) error {
	var name string
	switch g.kind {
	case gridPlacementLine:
		name = "Line"
	case gridPlacementSpan:
		name = "Span"
	default:
		return enc.WriteToken(jsontext.String("Auto"))
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(name)); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Int(int64(g.value))); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSONFrom decodes the placement from serde's externally tagged form.
func (g *GridPlacement) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	switch tok.Kind() {
	case '"':
		if s := tok.String(); s != "Auto" {
			return fmt.Errorf("unknown variant `%s`, expected one of `Line`, `Span`, `Auto`", s)
		}
		*g = GridPlacementAuto()
		return nil
	case '{':
	default:
		return fmt.Errorf("invalid type: %v, expected enum GridPlacement", tok.Kind())
	}
	nameTok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	name := nameTok.String()
	valueTok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if valueTok.Kind() != '0' {
		return fmt.Errorf("invalid type: %v, expected an integer", valueTok.Kind())
	}
	switch name {
	case "Line":
		v, err := strconv.ParseInt(valueTok.String(), 10, 16)
		if err != nil {
			return fmt.Errorf("invalid value for Line: %w", err)
		}
		*g = GridPlacementLine(int16(v))
	case "Span":
		v, err := strconv.ParseUint(valueTok.String(), 10, 16)
		if err != nil {
			return fmt.Errorf("invalid value for Span: %w", err)
		}
		*g = GridPlacementSpan(uint16(v))
	default:
		return fmt.Errorf("unknown variant `%s`, expected one of `Line`, `Span`, `Auto`", name)
	}
	end, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if end.Kind() != '}' {
		return errors.New("invalid length 2, expected map with a single key")
	}
	return nil
}

// Half is implemented by values that can be halved.
type Half[S any] interface {
	Half() S
}

// IsZero is implemented by values that can report whether they are zero.
type IsZero interface {
	IsZero() bool
}

// IsZero reports whether both coordinates are zero.
func (p Point[T]) IsZero() bool {
	return p.X == 0 && p.Y == 0
}

// IsZero reports whether the width or the height is zero.
func (s Size[T]) IsZero() bool {
	return s.Width == 0 || s.Height == 0
}

// IsZero reports whether the size is zero.
func (b Bounds[T]) IsZero() bool {
	return b.Size.IsZero()
}

func readJSONString(dec *jsontext.Decoder, expecting string) (string, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return "", err
	}
	if tok.Kind() != '"' {
		return "", fmt.Errorf("invalid type: %v, expected %s", tok.Kind(), expecting)
	}
	return tok.String(), nil
}

func parseRustF32(s string) (float32, error) {
	body := s
	if body != "" && (body[0] == '+' || body[0] == '-') {
		body = body[1:]
	}
	if s == "" {
		return 0, errors.New("cannot parse float from empty string")
	}
	if body == "" || strings.ContainsAny(body, "_xXpP") {
		return 0, errors.New("invalid float literal")
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return float32(v), nil
		}
		return 0, errors.New("invalid float literal")
	}
	return float32(v), nil
}

func rustDisplayF32(v float32) string {
	switch {
	case v != v:
		return "NaN"
	case math.IsInf(float64(v), 1):
		return "inf"
	case math.IsInf(float64(v), -1):
		return "-inf"
	}
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

func rustDebugF32(v float32) string {
	if v != v || math.IsInf(float64(v), 0) {
		return rustDisplayF32(v)
	}
	a := math.Abs(float64(v))
	if a != 0 && (a < 1e-4 || a >= 1e16) {
		s := strconv.FormatFloat(float64(v), 'e', -1, 32)
		mantissa, exponent, _ := strings.Cut(s, "e")
		exp, _ := strconv.Atoi(exponent)
		return mantissa + "e" + strconv.Itoa(exp)
	}
	s := strconv.FormatFloat(float64(v), 'f', -1, 32)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func signumF32(v float32) float32 {
	if v != v {
		return v
	}
	return float32(math.Copysign(1, float64(v)))
}

func saturatingF32ToU32(v float32) uint32 {
	switch {
	case v != v || v <= 0:
		return 0
	case v >= 4294967296:
		return math.MaxUint32
	}
	return uint32(v)
}

func saturatingF32ToU8(v float32) uint8 {
	switch {
	case v != v || v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return uint8(v)
}

func saturatingF64ToU64(v float64) uint64 {
	switch {
	case v != v || v <= 0:
		return 0
	case v >= 18446744073709551616:
		return math.MaxUint64
	}
	return uint64(v)
}

func geomOrdCmp[T Number](a, b T) int {
	c, _ := geomPartialCmp(a, b)
	return c
}

func geomOrdMin[T Number](a, b T) T {
	if geomOrdCmp(b, a) < 0 {
		return b
	}
	return a
}

func geomOrdMax[T Number](a, b T) T {
	if geomOrdCmp(b, a) < 0 {
		return a
	}
	return b
}

func half[T Number](v T) T {
	return v / 2
}

func scalePixels[T Number](v T, factor float32) ScaledPixels {
	return ScaledPixels(float32(v) * factor)
}

func roundToDevicePixels[T Number](v T, factor float32) DevicePixels {
	product := float32(v) * factor
	return DevicePixels(saturatingF32ToI32(float32(math.Round(float64(product)))))
}

func saturatingF32ToI32(v float32) int32 {
	switch {
	case v != v:
		return 0
	case v >= 2147483648:
		return math.MaxInt32
	case v <= -2147483648:
		return math.MinInt32
	}
	return int32(v)
}

func totalCmpF32(a, b float32) int {
	x := int32(math.Float32bits(a))
	y := int32(math.Float32bits(b))
	x ^= int32(uint32(x>>31) >> 1)
	y ^= int32(uint32(y>>31) >> 1)
	return cmp.Compare(x, y)
}

func geomPartialCmp[T Number](a, b T) (int, bool) {
	switch any((*T)(nil)).(type) {
	case *Pixels:
		switch {
		case a < b:
			return -1, true
		case a > b:
			return 1, true
		case a == b:
			return 0, true
		}
		return totalCmpF32(float32(a), float32(b)), true
	case *ScaledPixels:
		return totalCmpF32(float32(a), float32(b)), true
	}
	switch {
	case a < b:
		return -1, true
	case a > b:
		return 1, true
	case a == b:
		return 0, true
	}
	return 0, false
}

func geomGT[T Number](a, b T) bool {
	c, ok := geomPartialCmp(a, b)
	return ok && c > 0
}

func geomGE[T Number](a, b T) bool {
	c, ok := geomPartialCmp(a, b)
	return ok && c >= 0
}

func geomLT[T Number](a, b T) bool {
	c, ok := geomPartialCmp(a, b)
	return ok && c < 0
}

func geomLE[T Number](a, b T) bool {
	c, ok := geomPartialCmp(a, b)
	return ok && c <= 0
}

func geomMax[T Number](a, b T) T {
	if c, ok := geomPartialCmp(a, b); ok && c > 0 {
		return a
	}
	return b
}
