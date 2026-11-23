package gpui

import (
	"cmp"
	"encoding/json/jsontext"
	"fmt"
	"math"
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
	Top    T
	Right  T
	Bottom T
	Left   T
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
	Top    T
	Right  T
	Bottom T
	Left   T
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

// Pixels is a length in logical pixels.
type Pixels float32

// DevicePixels is a length in physical device pixels.
type DevicePixels int32

// ScaledPixels is a length in pixels scaled by the display's scale factor.
type ScaledPixels float32

// Rems is a length in rems, relative to the window's rem size.
type Rems float32

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
