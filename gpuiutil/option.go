package gpuiutil

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// Option is an optional value: Some(v) or None. The zero value is None.
type Option[T any] struct {
	value T
	some  bool
}

// Some returns an Option holding v.
func Some[T any](v T) Option[T] {
	return Option[T]{value: v, some: true}
}

// None returns an empty Option.
func None[T any]() Option[T] {
	return Option[T]{}
}

// OptionFrom returns Some(v) when ok is true and None otherwise.
func OptionFrom[T any](v T, ok bool) Option[T] {
	if !ok {
		return Option[T]{}
	}
	return Option[T]{value: v, some: true}
}

// Get returns the held value and whether there is one.
func (o Option[T]) Get() (T, bool) {
	return o.value, o.some
}

// AsMut returns a pointer to the held value, or nil when o is None.
func (o *Option[T]) AsMut() *T {
	if !o.some {
		return nil
	}
	return &o.value
}

// IsSome reports whether o holds a value.
func (o Option[T]) IsSome() bool {
	return o.some
}

// IsNone reports whether o is empty.
func (o Option[T]) IsNone() bool {
	return !o.some
}

// UnwrapOr returns the held value or def.
func (o Option[T]) UnwrapOr(def T) T {
	if o.some {
		return o.value
	}
	return def
}

// UnwrapOrDefault returns the held value or the zero value of T.
func (o Option[T]) UnwrapOrDefault() T {
	return o.value
}

// Unwrap returns the held value and panics when o is None.
func (o Option[T]) Unwrap() T {
	if !o.some {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return o.value
}

// Expect returns the held value and panics with msg when o is None.
func (o Option[T]) Expect(msg string) T {
	if !o.some {
		panic(msg)
	}
	return o.value
}

// Or returns o when it is Some and other otherwise.
func (o Option[T]) Or(other Option[T]) Option[T] {
	if o.some {
		return o
	}
	return other
}

// Take returns the current value of *o and leaves None in its place.
func (o *Option[T]) Take() Option[T] {
	v := *o
	*o = Option[T]{}
	return v
}

// Replace stores Some(v) in *o and returns the previous value.
func (o *Option[T]) Replace(v T) Option[T] {
	prev := *o
	*o = Some(v)
	return prev
}

// IsZero reports whether o is None, so `omitzero` skips empty options.
func (o Option[T]) IsZero() bool {
	return !o.some
}

// String formats o as Some(v) or None.
func (o Option[T]) String() string {
	if !o.some {
		return "None"
	}
	return fmt.Sprintf("Some(%v)", o.value)
}

// MarshalJSONTo encodes None as null and Some(v) as v.
func (o Option[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if !o.some {
		return enc.WriteToken(jsontext.Null)
	}
	return json.MarshalEncode(enc, o.value)
}

// UnmarshalJSONFrom decodes null as None and any other value as Some.
func (o *Option[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == 'n' {
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		*o = Option[T]{}
		return nil
	}
	var v T
	if err := json.UnmarshalDecode(dec, &v); err != nil {
		return err
	}
	*o = Some(v)
	return nil
}
