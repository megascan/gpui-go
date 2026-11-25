// Package etagere is a port of the BucketedAtlasAllocator of etagere 0.2.15.
package etagere

// Point is an integer point, the Go form of euclid's Point2D<i32>.
type Point struct {
	X, Y int32
}

// Size is an integer size, the Go form of euclid's Size2D<i32>.
type Size struct {
	Width, Height int32
}

// Rectangle is an integer box from Min to Max, the Go form of euclid's Box2D<i32>.
type Rectangle struct {
	Min, Max Point
}

// Point2 returns the point (x, y).
func Point2(x, y int32) Point {
	return Point{X: x, Y: y}
}

// Size2 returns the size (width, height).
func Size2(width, height int32) Size {
	return Size{Width: width, Height: height}
}

// IsEmpty reports whether the width or the height is not strictly positive.
func (s Size) IsEmpty() bool {
	return !(s.Width > 0 && s.Height > 0)
}

// Area returns width times height.
func (s Size) Area() int32 {
	return s.Width * s.Height
}

// Size returns the extent of the rectangle, Max minus Min.
func (r Rectangle) Size() Size {
	return Size2(r.Max.X-r.Min.X, r.Max.Y-r.Min.Y)
}

// Area returns the area of the rectangle.
func (r Rectangle) Area() int32 {
	return r.Size().Area()
}

// AllocatorOptions tweaks the behaviour of the atlas allocator.
type AllocatorOptions struct {
	Alignment       Size
	VerticalShelves bool
	NumColumns      int32
}

// DefaultOptions is the upstream DEFAULT_OPTIONS value: no alignment, horizontal shelves, one column.
var DefaultOptions = AllocatorOptions{
	Alignment:  Size2(1, 1),
	NumColumns: 1,
}

// DefaultAllocatorOptions returns DefaultOptions.
func DefaultAllocatorOptions() AllocatorOptions {
	return DefaultOptions
}

// Allocation is the AllocID and rectangle resulting from an allocation.
type Allocation struct {
	ID        AllocID
	Rectangle Rectangle
}

// AllocID refers to an allocated rectangle.
type AllocID uint32

// Serialize returns the raw 32-bit encoding of the id.
func (id AllocID) Serialize() uint32 {
	return uint32(id)
}

// AllocIDDeserialize rebuilds an AllocID from its raw 32-bit encoding.
func AllocIDDeserialize(bytes uint32) AllocID {
	return AllocID(bytes)
}
