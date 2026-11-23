package taffy

import "unsafe"

// MaybeResolve resolves a possibly context-dependent value against a context into Out, which may be None.
type MaybeResolve[In, Out any] interface {
	MaybeResolve(context In, calc func(unsafe.Pointer, float32) float32) Out
}

// ResolveOrZero resolves a possibly context-dependent value against a context, falling back to zero.
type ResolveOrZero[In, Out any] interface {
	ResolveOrZero(context In, calc func(unsafe.Pointer, float32) float32) Out
}

func maybeResolvePercent(context Option[float32], value float32) Option[float32] {
	if dim, ok := context.Get(); ok {
		return some(float32(dim * value))
	}
	return none[float32]()
}

// MaybeResolve converts the LengthPercentage into an absolute length; None for a percentage without context.
func (l LengthPercentage) MaybeResolve(context Option[float32], calc func(unsafe.Pointer, float32) float32) Option[float32] {
	switch l.inner.Tag() {
	case CompactLengthLengthTag:
		return some(l.inner.Value())
	case CompactLengthPercentTag:
		return maybeResolvePercent(context, l.inner.Value())
	}
	panic("internal error: entered unreachable code")
}

// MaybeResolve converts the LengthPercentageAuto into an absolute length; None for auto or a percentage without context.
func (l LengthPercentageAuto) MaybeResolve(context Option[float32], calc func(unsafe.Pointer, float32) float32) Option[float32] {
	switch l.inner.Tag() {
	case CompactLengthAutoTag:
		return none[float32]()
	case CompactLengthLengthTag:
		return some(l.inner.Value())
	case CompactLengthPercentTag:
		return maybeResolvePercent(context, l.inner.Value())
	}
	panic("internal error: entered unreachable code")
}

// MaybeResolve converts the Dimension into an absolute length; None for auto, content, sizing keywords or a
// percentage without context.
func (d Dimension) MaybeResolve(context Option[float32], calc func(unsafe.Pointer, float32) float32) Option[float32] {
	switch d.inner.Tag() {
	case CompactLengthAutoTag:
		return none[float32]()
	case CompactLengthContentTag:
		return none[float32]()
	case CompactLengthLengthTag:
		return some(d.inner.Value())
	case CompactLengthPercentTag:
		return maybeResolvePercent(context, d.inner.Value())
	}
	if d.inner.IsSizingKeyword() {
		return none[float32]()
	}
	panic("internal error: entered unreachable code")
}

// MaybeResolveF32 is the blanket MaybeResolve<f32, Option<f32>> impl: it resolves value against Some(context).
func MaybeResolveF32[T MaybeResolve[Option[float32], Option[float32]]](value T, context float32, calc func(unsafe.Pointer, float32) float32) Option[float32] {
	return value.MaybeResolve(some(context), calc)
}

// SizeMaybeResolve resolves each dimension of s against the matching dimension of context.
func SizeMaybeResolve[T MaybeResolve[In, Out], In, Out any](s Size[T], context Size[In], calc func(unsafe.Pointer, float32) float32) Size[Out] {
	return Size[Out]{
		Width:  s.Width.MaybeResolve(context.Width, calc),
		Height: s.Height.MaybeResolve(context.Height, calc),
	}
}

// SizeMaybeResolveF32 resolves each dimension of s against the matching f32 dimension of context.
func SizeMaybeResolveF32[T MaybeResolve[Option[float32], Option[float32]]](s Size[T], context Size[float32], calc func(unsafe.Pointer, float32) float32) Size[Option[float32]] {
	return Size[Option[float32]]{
		Width:  MaybeResolveF32(s.Width, context.Width, calc),
		Height: MaybeResolveF32(s.Height, context.Height, calc),
	}
}

// ResolveOrZero resolves the LengthPercentage, returning 0 when it cannot be resolved.
func (l LengthPercentage) ResolveOrZero(context Option[float32], calc func(unsafe.Pointer, float32) float32) float32 {
	return l.MaybeResolve(context, calc).UnwrapOr(0.0)
}

// ResolveOrZero resolves the LengthPercentageAuto, returning 0 when it cannot be resolved.
func (l LengthPercentageAuto) ResolveOrZero(context Option[float32], calc func(unsafe.Pointer, float32) float32) float32 {
	return l.MaybeResolve(context, calc).UnwrapOr(0.0)
}

// ResolveOrZero resolves the Dimension, returning 0 when it cannot be resolved.
func (d Dimension) ResolveOrZero(context Option[float32], calc func(unsafe.Pointer, float32) float32) float32 {
	return d.MaybeResolve(context, calc).UnwrapOr(0.0)
}

// SizeResolveOrZero resolves each dimension of s against the matching dimension of context.
func SizeResolveOrZero[T ResolveOrZero[In, Out], In, Out any](s Size[T], context Size[In], calc func(unsafe.Pointer, float32) float32) Size[Out] {
	return Size[Out]{
		Width:  s.Width.ResolveOrZero(context.Width, calc),
		Height: s.Height.ResolveOrZero(context.Height, calc),
	}
}

// RectResolveOrZero resolves the horizontal sides of r against context.Width and the vertical sides against context.Height.
func RectResolveOrZero[T ResolveOrZero[In, Out], In, Out any](r Rect[T], context Size[In], calc func(unsafe.Pointer, float32) float32) Rect[Out] {
	return Rect[Out]{
		Left:   r.Left.ResolveOrZero(context.Width, calc),
		Right:  r.Right.ResolveOrZero(context.Width, calc),
		Top:    r.Top.ResolveOrZero(context.Height, calc),
		Bottom: r.Bottom.ResolveOrZero(context.Height, calc),
	}
}

// RectResolveOrZeroOptionF32 resolves every side of r against the same Option<f32> context.
func RectResolveOrZeroOptionF32[T ResolveOrZero[Option[float32], Out], Out any](r Rect[T], context Option[float32], calc func(unsafe.Pointer, float32) float32) Rect[Out] {
	return Rect[Out]{
		Left:   r.Left.ResolveOrZero(context, calc),
		Right:  r.Right.ResolveOrZero(context, calc),
		Top:    r.Top.ResolveOrZero(context, calc),
		Bottom: r.Bottom.ResolveOrZero(context, calc),
	}
}

// RectResolveOrZeroF32 resolves every side of r against the same f32 context.
func RectResolveOrZeroF32[T ResolveOrZero[float32, Out], Out any](r Rect[T], context float32, calc func(unsafe.Pointer, float32) float32) Rect[Out] {
	return Rect[Out]{
		Left:   r.Left.ResolveOrZero(context, calc),
		Right:  r.Right.ResolveOrZero(context, calc),
		Top:    r.Top.ResolveOrZero(context, calc),
		Bottom: r.Bottom.ResolveOrZero(context, calc),
	}
}
