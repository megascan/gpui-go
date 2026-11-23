package atspi

import (
	"math"

	"github.com/megascan/gpui-go/a11y"
)

// Rect is an AT-SPI rectangle in integer coordinates.
type Rect struct {
	X      int32
	Y      int32
	Width  int32
	Height int32
}

// RectInvalid returns the rectangle AT-SPI uses for "no extents".
func RectInvalid() Rect {
	return Rect{X: -1, Y: -1, Width: -1, Height: -1}
}

// RectFromAccessKit converts an accesskit rectangle, truncating as Rust's `as i32` does.
func RectFromAccessKit(value a11y.Rect) Rect {
	return Rect{
		X:      f64AsI32(value.X0),
		Y:      f64AsI32(value.Y0),
		Width:  f64AsI32(value.Width()),
		Height: f64AsI32(value.Height()),
	}
}

func f64AsI32(v float64) int32 {
	switch {
	case math.IsNaN(v):
		return 0
	case v >= math.MaxInt32:
		return math.MaxInt32
	case v <= math.MinInt32:
		return math.MinInt32
	default:
		return int32(v)
	}
}
