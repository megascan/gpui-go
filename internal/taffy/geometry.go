package taffy

import (
	"fmt"
	"reflect"
)

// AbsoluteAxis is the simple absolute horizontal and vertical axis.
type AbsoluteAxis uint8

const (
	AbsoluteAxisHorizontal AbsoluteAxis = iota
	AbsoluteAxisVertical
)

// OtherAxis returns the other variant of the enum.
func (a AbsoluteAxis) OtherAxis() AbsoluteAxis {
	switch a {
	case AbsoluteAxisHorizontal:
		return AbsoluteAxisVertical
	default:
		return AbsoluteAxisHorizontal
	}
}

// GetAbs gets either the width or height depending on the AbsoluteAxis passed in.
func (s Size[T]) GetAbs(axis AbsoluteAxis) T {
	switch axis {
	case AbsoluteAxisHorizontal:
		return s.Width
	default:
		return s.Height
	}
}

// Transpose swaps the width and height components.
func (s Size[T]) Transpose() Size[T] {
	return Size[T]{Width: s.Height, Height: s.Width}
}

// GridAxisSum gets either the horizontal or the vertical sum depending on the AbsoluteAxis passed in.
func (r Rect[T]) GridAxisSum(axis AbsoluteAxis) T {
	switch axis {
	case AbsoluteAxisHorizontal:
		return add(r.Left, r.Right)
	default:
		return add(r.Top, r.Bottom)
	}
}

// AbstractAxis is the CSS abstract axis.
type AbstractAxis uint8

const (
	AbstractAxisInline AbstractAxis = iota
	AbstractAxisBlock
)

// Other returns the other variant of the enum.
func (a AbstractAxis) Other() AbstractAxis {
	switch a {
	case AbstractAxisInline:
		return AbstractAxisBlock
	default:
		return AbstractAxisInline
	}
}

// AsAbsNaive converts an AbstractAxis into an AbsoluteAxis, assuming that the inline axis is horizontal.
func (a AbstractAxis) AsAbsNaive() AbsoluteAxis {
	switch a {
	case AbstractAxisInline:
		return AbsoluteAxisHorizontal
	default:
		return AbsoluteAxisVertical
	}
}

type inBothAbsAxis[T any] struct {
	horizontal T
	vertical   T
}

func (b inBothAbsAxis[T]) get(axis AbsoluteAxis) T {
	switch axis {
	case AbsoluteAxisHorizontal:
		return b.horizontal
	default:
		return b.vertical
	}
}

// Rect is an axis-aligned UI rectangle.
type Rect[T any] struct {
	Left   T
	Right  T
	Top    T
	Bottom T
}

// Add adds the sides of two rects.
func (r Rect[T]) Add(rhs Rect[T]) Rect[T] {
	return Rect[T]{
		Left:   add(r.Left, rhs.Left),
		Right:  add(r.Right, rhs.Right),
		Top:    add(r.Top, rhs.Top),
		Bottom: add(r.Bottom, rhs.Bottom),
	}
}

// Transpose swaps the horizontal and vertical sides (left with top, right with bottom).
func (r Rect[T]) Transpose() Rect[T] {
	return Rect[T]{Left: r.Top, Right: r.Bottom, Top: r.Left, Bottom: r.Right}
}

func (r Rect[T]) zipSize[U, R any](size Size[U], f func(T, U) R) Rect[R] {
	return Rect[R]{
		Left:   f(r.Left, size.Width),
		Right:  f(r.Right, size.Width),
		Top:    f(r.Top, size.Height),
		Bottom: f(r.Bottom, size.Height),
	}
}

// Map applies f to all four sides of the rect.
func (r Rect[T]) Map[R any](f func(T) R) Rect[R] {
	return Rect[R]{Left: f(r.Left), Right: f(r.Right), Top: f(r.Top), Bottom: f(r.Bottom)}
}

// HorizontalComponents returns a Line holding the left and right sides of the rect.
func (r Rect[T]) HorizontalComponents() Line[T] {
	return Line[T]{Start: r.Left, End: r.Right}
}

// VerticalComponents returns a Line holding the top and bottom sides of the rect.
func (r Rect[T]) VerticalComponents() Line[T] {
	return Line[T]{Start: r.Top, End: r.Bottom}
}

// HorizontalAxisSum returns the sum of the left and right sides.
func (r Rect[T]) HorizontalAxisSum() T {
	return add(r.Left, r.Right)
}

// VerticalAxisSum returns the sum of the top and bottom sides.
func (r Rect[T]) VerticalAxisSum() T {
	return add(r.Top, r.Bottom)
}

// SumAxes returns HorizontalAxisSum and VerticalAxisSum as a Size.
func (r Rect[T]) SumAxes() Size[T] {
	return Size[T]{Width: r.HorizontalAxisSum(), Height: r.VerticalAxisSum()}
}

func (r Rect[T]) mainAxisSum(direction FlexDirection) T {
	if direction.IsRow() {
		return r.HorizontalAxisSum()
	}
	return r.VerticalAxisSum()
}

func (r Rect[T]) crossAxisSum(direction FlexDirection) T {
	if direction.IsRow() {
		return r.VerticalAxisSum()
	}
	return r.HorizontalAxisSum()
}

func (r Rect[T]) mainStart(direction FlexDirection) T {
	if direction.IsRow() {
		return r.Left
	}
	return r.Top
}

func (r Rect[T]) mainEnd(direction FlexDirection) T {
	if direction.IsRow() {
		return r.Right
	}
	return r.Bottom
}

func (r Rect[T]) crossStart(direction FlexDirection) T {
	if direction.IsRow() {
		return r.Top
	}
	return r.Left
}

func (r Rect[T]) crossEnd(direction FlexDirection) T {
	if direction.IsRow() {
		return r.Bottom
	}
	return r.Right
}

// RectOptionF32None is a Rect with None on all sides.
var RectOptionF32None = Rect[Option[float32]]{}

// RectF32Zero is a Rect with 0.0 on all sides.
var RectF32Zero = Rect[float32]{}

// NewRectF32 creates a new Rect.
func NewRectF32(start, end, top, bottom float32) Rect[float32] {
	return Rect[float32]{Left: start, Right: end, Top: top, Bottom: bottom}
}

// Union returns the smallest rectangle that contains both r and other, reading the sides as edge coordinates.
// It is defined for Rect[float32] only.
func (r Rect[T]) Union(other Rect[T]) Rect[T] {
	a, b := asF32Rect(&r, "Union"), asF32Rect(&other, "Union")
	out := Rect[float32]{
		Left:   f32Min(a.Left, b.Left),
		Right:  f32Max(a.Right, b.Right),
		Top:    f32Min(a.Top, b.Top),
		Bottom: f32Max(a.Bottom, b.Bottom),
	}
	return *any(&out).(*Rect[T])
}

// Line is an abstract line: anything with a start and an end.
type Line[T any] struct {
	Start T
	End   T
}

// Map applies f to both the start and the end.
func (l Line[T]) Map[R any](f func(T) R) Line[R] {
	return Line[R]{Start: f(l.Start), End: f(l.End)}
}

var (
	// LineBoolTrue is a Line[bool] with both start and end set to true.
	LineBoolTrue = Line[bool]{Start: true, End: true}
	// LineBoolFalse is a Line[bool] with both start and end set to false.
	LineBoolFalse = Line[bool]{Start: false, End: false}
)

// Sum adds the start and end values together.
func (l Line[T]) Sum() T {
	return add(l.Start, l.End)
}

// Size is the width and height of a Rect.
type Size[T any] struct {
	Width  T
	Height T
}

// Add adds two sizes component-wise.
func (s Size[T]) Add(rhs Size[T]) Size[T] {
	return Size[T]{Width: add(s.Width, rhs.Width), Height: add(s.Height, rhs.Height)}
}

// Sub subtracts rhs from s component-wise.
func (s Size[T]) Sub(rhs Size[T]) Size[T] {
	return Size[T]{Width: sub(s.Width, rhs.Width), Height: sub(s.Height, rhs.Height)}
}

// Map applies f to both the width and height.
func (s Size[T]) Map[R any](f func(T) R) Size[R] {
	return Size[R]{Width: f(s.Width), Height: f(s.Height)}
}

// MapWidth applies f to the width.
func (s Size[T]) MapWidth(f func(T) T) Size[T] {
	return Size[T]{Width: f(s.Width), Height: s.Height}
}

// MapHeight applies f to the height.
func (s Size[T]) MapHeight(f func(T) T) Size[T] {
	return Size[T]{Width: s.Width, Height: f(s.Height)}
}

// ZipMap applies f to the width and height of s and other.
func (s Size[T]) ZipMap[Other, Ret any](other Size[Other], f func(T, Other) Ret) Size[Ret] {
	return Size[Ret]{Width: f(s.Width, other.Width), Height: f(s.Height, other.Height)}
}

func (s *Size[T]) setMain(direction FlexDirection, value T) {
	if direction.IsRow() {
		s.Width = value
	} else {
		s.Height = value
	}
}

func (s *Size[T]) setCross(direction FlexDirection, value T) {
	if direction.IsRow() {
		s.Height = value
	} else {
		s.Width = value
	}
}

func (s Size[T]) withMain(direction FlexDirection, value T) Size[T] {
	n := s
	if direction.IsRow() {
		n.Width = value
	} else {
		n.Height = value
	}
	return n
}

func (s Size[T]) withCross(direction FlexDirection, value T) Size[T] {
	n := s
	if direction.IsRow() {
		n.Height = value
	} else {
		n.Width = value
	}
	return n
}

func (s Size[T]) mapMain(direction FlexDirection, mapper func(T) T) Size[T] {
	n := s
	if direction.IsRow() {
		n.Width = mapper(n.Width)
	} else {
		n.Height = mapper(n.Height)
	}
	return n
}

func (s Size[T]) mapCross(direction FlexDirection, mapper func(T) T) Size[T] {
	n := s
	if direction.IsRow() {
		n.Height = mapper(n.Height)
	} else {
		n.Width = mapper(n.Width)
	}
	return n
}

func (s Size[T]) main(direction FlexDirection) T {
	if direction.IsRow() {
		return s.Width
	}
	return s.Height
}

func (s Size[T]) cross(direction FlexDirection) T {
	if direction.IsRow() {
		return s.Height
	}
	return s.Width
}

func (s Size[T]) get(axis AbstractAxis) T {
	switch axis {
	case AbstractAxisInline:
		return s.Width
	default:
		return s.Height
	}
}

func (s *Size[T]) set(axis AbstractAxis, value T) {
	switch axis {
	case AbstractAxisInline:
		s.Width = value
	default:
		s.Height = value
	}
}

func (s Size[T]) with(axis AbstractAxis, value T) Size[T] {
	switch axis {
	case AbstractAxisInline:
		s.Width = value
	default:
		s.Height = value
	}
	return s
}

// SizeF32Zero is a Size with zero width and height.
var SizeF32Zero = Size[float32]{}

// F32Max applies f32_max to each component separately. It is defined for Size[float32] only.
func (s Size[T]) F32Max(rhs Size[T]) Size[T] {
	a, b := asF32Size(&s, "F32Max"), asF32Size(&rhs, "F32Max")
	out := Size[float32]{Width: f32Max(a.Width, b.Width), Height: f32Max(a.Height, b.Height)}
	return *any(&out).(*Size[T])
}

// F32Min applies f32_min to each component separately. It is defined for Size[float32] only.
func (s Size[T]) F32Min(rhs Size[T]) Size[T] {
	a, b := asF32Size(&s, "F32Min"), asF32Size(&rhs, "F32Min")
	out := Size[float32]{Width: f32Min(a.Width, b.Width), Height: f32Min(a.Height, b.Height)}
	return *any(&out).(*Size[T])
}

// HasNonZeroArea reports whether both width and height are greater than 0. It is defined for Size[float32] only.
func (s Size[T]) HasNonZeroArea() bool {
	a := asF32Size(&s, "HasNonZeroArea")
	return a.Width > 0.0 && a.Height > 0.0
}

// SizeOptionF32None is a Size with None width and height.
var SizeOptionF32None = Size[Option[float32]]{}

// NewSizeOptionF32 returns a Size with Some(width) and Some(height).
func NewSizeOptionF32(width, height float32) Size[Option[float32]] {
	return Size[Option[float32]]{Width: some(width), Height: some(height)}
}

// SizeOptionF32FromCross creates a Size with either the width or height set based on direction.
func SizeOptionF32FromCross(direction FlexDirection, value Option[float32]) Size[Option[float32]] {
	n := SizeOptionF32None
	if direction.IsRow() {
		n.Height = value
	} else {
		n.Width = value
	}
	return n
}

// SizeOptionF32MaybeApplyAspectRatio fills in a missing width or height from the other one and aspect_ratio.
func SizeOptionF32MaybeApplyAspectRatio(s Size[Option[float32]], aspectRatio Option[float32]) Size[Option[float32]] {
	ratio, ok := aspectRatio.Get()
	if !ok {
		return s
	}
	width, wok := s.Width.Get()
	height, hok := s.Height.Get()
	switch {
	case wok && !hok:
		return Size[Option[float32]]{Width: some(width), Height: some(float32(width / ratio))}
	case !wok && hok:
		return Size[Option[float32]]{Width: some(float32(height * ratio)), Height: some(height)}
	}
	return s
}

// SizeOptionUnwrapOr performs Option.UnwrapOr on each component separately.
func SizeOptionUnwrapOr[T any](s Size[Option[T]], alt Size[T]) Size[T] {
	return Size[T]{Width: s.Width.UnwrapOr(alt.Width), Height: s.Height.UnwrapOr(alt.Height)}
}

// SizeOptionOr performs Option.Or on each component separately.
func SizeOptionOr[T any](s Size[Option[T]], alt Size[Option[T]]) Size[Option[T]] {
	return Size[Option[T]]{Width: s.Width.Or(alt.Width), Height: s.Height.Or(alt.Height)}
}

// SizeOptionBothAxisDefined reports whether both components are Some.
func SizeOptionBothAxisDefined[T any](s Size[Option[T]]) bool {
	return s.Width.IsSome() && s.Height.IsSome()
}

// SizeDimensionFromLengths returns a Size[Dimension] of length values.
func SizeDimensionFromLengths(width, height float32) Size[Dimension] {
	return Size[Dimension]{Width: Dimension{CompactLengthLength(width)}, Height: Dimension{CompactLengthLength(height)}}
}

// SizeDimensionFromPercent returns a Size[Dimension] of percentage values.
func SizeDimensionFromPercent(width, height float32) Size[Dimension] {
	return Size[Dimension]{Width: Dimension{CompactLengthPercent(width)}, Height: Dimension{CompactLengthPercent(height)}}
}

// Point is a 2-dimensional coordinate.
type Point[T any] struct {
	X T
	Y T
}

// PointF32Zero is the origin.
var PointF32Zero = Point[float32]{}

// PointOptionF32None is a Point with values (None, None).
var PointOptionF32None = Point[Option[float32]]{}

// Add adds two points component-wise.
func (p Point[T]) Add(rhs Point[T]) Point[T] {
	return Point[T]{X: add(p.X, rhs.X), Y: add(p.Y, rhs.Y)}
}

// Map applies f to both x and y.
func (p Point[T]) Map[R any](f func(T) R) Point[R] {
	return Point[R]{X: f(p.X), Y: f(p.Y)}
}

// Get gets the component of the specified layout axis.
func (p Point[T]) Get(axis AbstractAxis) T {
	switch axis {
	case AbstractAxisInline:
		return p.X
	default:
		return p.Y
	}
}

// Transpose swaps the x and y components.
func (p Point[T]) Transpose() Point[T] {
	return Point[T]{X: p.Y, Y: p.X}
}

// Set sets the component of the specified layout axis.
func (p *Point[T]) Set(axis AbstractAxis, value T) {
	switch axis {
	case AbstractAxisInline:
		p.X = value
	default:
		p.Y = value
	}
}

func (p Point[T]) main(direction FlexDirection) T {
	if direction.IsRow() {
		return p.X
	}
	return p.Y
}

func (p Point[T]) cross(direction FlexDirection) T {
	if direction.IsRow() {
		return p.Y
	}
	return p.X
}

// SizeFromPoint converts a Point into a Size.
func SizeFromPoint[T any](value Point[T]) Size[T] {
	return Size[T]{Width: value.X, Height: value.Y}
}

// MinMax holds a min value and a max value.
type MinMax[Min, Max any] struct {
	Min Min
	Max Max
}

func asF32Rect[T any](r *Rect[T], op string) *Rect[float32] {
	if p, ok := any(r).(*Rect[float32]); ok {
		return p
	}
	panic(fmt.Sprintf("taffy: Rect.%s is only defined for Rect[float32], not %s", op, reflect.TypeFor[Rect[T]]()))
}

func asF32Size[T any](s *Size[T], op string) *Size[float32] {
	if p, ok := any(s).(*Size[float32]); ok {
		return p
	}
	panic(fmt.Sprintf("taffy: Size.%s is only defined for Size[float32], not %s", op, reflect.TypeFor[Size[T]]()))
}

func add[T any](a, b T) T {
	switch pa := any(&a).(type) {
	case *float32:
		*pa += *any(&b).(*float32)
	case *float64:
		*pa += *any(&b).(*float64)
	case *int:
		*pa += *any(&b).(*int)
	case *int16:
		*pa += *any(&b).(*int16)
	case *int32:
		*pa += *any(&b).(*int32)
	case *uint16:
		*pa += *any(&b).(*uint16)
	case *uint32:
		*pa += *any(&b).(*uint32)
	default:
		panic(fmt.Sprintf("taffy: Add is not defined for %s", reflect.TypeFor[T]()))
	}
	return a
}

func sub[T any](a, b T) T {
	switch pa := any(&a).(type) {
	case *float32:
		*pa -= *any(&b).(*float32)
	case *float64:
		*pa -= *any(&b).(*float64)
	case *int:
		*pa -= *any(&b).(*int)
	case *int16:
		*pa -= *any(&b).(*int16)
	case *int32:
		*pa -= *any(&b).(*int32)
	case *uint16:
		*pa -= *any(&b).(*uint16)
	case *uint32:
		*pa -= *any(&b).(*uint32)
	default:
		panic(fmt.Sprintf("taffy: Sub is not defined for %s", reflect.TypeFor[T]()))
	}
	return a
}
