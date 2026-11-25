package gpui

import "github.com/megascan/gpui-go/gpuiutil"

// Option is an optional value, upstream's Option<T> in struct fields (D-OPT).
type Option[T any] = gpuiutil.Option[T]

// Some returns an Option holding v.
func Some[T any](v T) Option[T] {
	return gpuiutil.Some(v)
}

// None returns an empty Option.
func None[T any]() Option[T] {
	return gpuiutil.None[T]()
}

// OptionFrom returns Some(v) when ok is true and None otherwise.
func OptionFrom[T any](v T, ok bool) Option[T] {
	return gpuiutil.OptionFrom(v, ok)
}
