//go:build darwin

package objc

import "structs"

// CGFloat is Core Graphics' scalar, a double on every 64-bit Apple target.
type CGFloat = float64

// NSInteger is Foundation's signed word.
type NSInteger = int

// NSUInteger is Foundation's unsigned word.
type NSUInteger = uint

// CGPoint is a point in a two-dimensional coordinate system.
type CGPoint struct {
	_ structs.HostLayout
	X CGFloat
	Y CGFloat
}

// CGSize is a width and height.
type CGSize struct {
	_      structs.HostLayout
	Width  CGFloat
	Height CGFloat
}

// CGRect is an origin and a size.
type CGRect struct {
	_      structs.HostLayout
	Origin CGPoint
	Size   CGSize
}

// NSRange is a location and length in a sequence.
type NSRange struct {
	_        structs.HostLayout
	Location NSUInteger
	Length   NSUInteger
}

// NSNotFound is the location of a range that matched nothing.
const NSNotFound NSInteger = int(^uint(0) >> 1)

// MakeCGPoint returns the point (x, y).
func MakeCGPoint(x, y CGFloat) CGPoint {
	return CGPoint{X: x, Y: y}
}

// MakeCGSize returns the size width x height.
func MakeCGSize(width, height CGFloat) CGSize {
	return CGSize{Width: width, Height: height}
}

// MakeCGRect returns the rectangle at (x, y) with the given size.
func MakeCGRect(x, y, width, height CGFloat) CGRect {
	return CGRect{Origin: MakeCGPoint(x, y), Size: MakeCGSize(width, height)}
}

// MakeNSRange returns the range [location, location+length).
func MakeNSRange(location, length NSUInteger) NSRange {
	return NSRange{Location: location, Length: length}
}
