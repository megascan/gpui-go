package taffy

import (
	"math"

	"github.com/megascan/gpui-go/gpuiutil"
)

// Option is the optional value used for upstream's Option<T>.
type Option[T any] = gpuiutil.Option[T]

type defaultCheapStr = string

func newVecWithCapacity[A any](capacity int) []A {
	return make([]A, 0, capacity)
}

func newConstChildrenVec[A any]() []A {
	return nil
}

func round(value float32) float32 {
	return floor(value + 0.5)
}

func ceil(value float32) float32 {
	return float32(math.Ceil(float64(value)))
}

func floor(value float32) float32 {
	return float32(math.Floor(float64(value)))
}

func abs(value float32) float32 {
	return math.Float32frombits(math.Float32bits(value) &^ (1 << 31))
}

func f32Max(a, b float32) float32 {
	switch {
	case a != a:
		return b
	case b != b:
		return a
	case a > b:
		return a
	}
	return b
}

func f32Min(a, b float32) float32 {
	switch {
	case a != a:
		return b
	case b != b:
		return a
	case a < b:
		return a
	}
	return b
}

func some[T any](v T) Option[T] {
	return gpuiutil.Some(v)
}

func none[T any]() Option[T] {
	return gpuiutil.None[T]()
}
