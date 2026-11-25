package refineable

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/megascan/gpui-go/gpuiutil"
)

// Refineable is implemented by *S for a type S refined by partial updates R.
type Refineable[S any, R any] interface {
	Refine(refinement *R)
	Refined(refinement R) S
	IsSupersetOf(refinement *R) bool
	Subtract(refinement *R) R
}

// FromCascade creates an S by merging every refinement of cascade atop the zero value.
func FromCascade[S interface{ Refined(R) S }, R any](cascade *Cascade[S, R]) S {
	var s S
	return s.Refined(cascade.Merged())
}

// IsEmpty is implemented by refinements.
type IsEmpty interface {
	IsEmpty() bool
}

// Cascade is a sequence of optional refinements merged in priority order; slot 0 is the base.
// The zero value holds an empty base refinement.
type Cascade[S any, R any] struct {
	slots []gpuiutil.Option[R]
}

// CascadeSlot identifies a slot of a Cascade.
type CascadeSlot struct {
	index int
}

// DefaultCascade returns a cascade holding only an empty base refinement.
func DefaultCascade[S any, R any]() Cascade[S, R] {
	var base R
	return Cascade[S, R]{slots: []gpuiutil.Option[R]{gpuiutil.Some(base)}}
}

func (c *Cascade[S, R]) ensureBase() {
	if len(c.slots) == 0 {
		var base R
		c.slots = append(c.slots, gpuiutil.Some(base))
	}
}

// Reserve adds an empty slot and returns its handle.
func (c *Cascade[S, R]) Reserve() CascadeSlot {
	c.ensureBase()
	c.slots = append(c.slots, gpuiutil.None[R]())
	return CascadeSlot{index: len(c.slots) - 1}
}

// Base returns the base refinement (slot 0).
func (c *Cascade[S, R]) Base() *R {
	c.ensureBase()
	base := c.slots[0].AsMut()
	if base == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return base
}

// Set stores refinement in slot; None removes the slot from merging.
func (c *Cascade[S, R]) Set(slot CascadeSlot, refinement gpuiutil.Option[R]) {
	c.ensureBase()
	c.slots[slot.index] = refinement
}

// Merged merges the refinements in slot order, later slots taking precedence.
func (c *Cascade[S, R]) Merged() R {
	c.ensureBase()
	merged := c.slots[0].Unwrap()
	target, ok := any(&merged).(interface{ Refine(*R) })
	if !ok {
		panic(fmt.Sprintf("refineable: %v does not implement Refine(*%v)", reflect.TypeFor[*R](), reflect.TypeFor[R]()))
	}
	for _, slot := range c.slots[1:] {
		if refinement, ok := slot.Get(); ok {
			target.Refine(&refinement)
		}
	}
	return merged
}

// Eq reports whether a equals b: through an Eq(T) bool method when T has one, == when T is
// comparable, and reflect.DeepEqual otherwise.
func Eq[T any](a, b T) bool {
	if e, ok := any(a).(interface{ Eq(T) bool }); ok {
		return e.Eq(b)
	}
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Interface && t.Comparable() {
		return any(a) == any(b)
	}
	return reflect.DeepEqual(a, b)
}

// OptionEq reports whether two options are both None or both Some with Eq values.
func OptionEq[T any](a, b gpuiutil.Option[T]) bool {
	av, aok := a.Get()
	bv, bok := b.Get()
	if aok != bok {
		return false
	}
	return !aok || Eq(av, bv)
}

// DebugStruct formats a refinement the way Rust's debug_struct does.
type DebugStruct struct {
	b      strings.Builder
	fields int
}

// NewDebugStruct starts formatting a struct called name.
func NewDebugStruct(name string) *DebugStruct {
	d := &DebugStruct{}
	d.b.WriteString(name)
	return d
}

// Field appends one field.
func (d *DebugStruct) Field(name string, value any) {
	if d.fields == 0 {
		d.b.WriteString(" { ")
	} else {
		d.b.WriteString(", ")
	}
	fmt.Fprintf(&d.b, "%s: %v", name, value)
	d.fields++
}

// Finish closes the struct.
func (d *DebugStruct) Finish() string {
	if d.fields > 0 {
		d.b.WriteString(" }")
	}
	return d.b.String()
}

// FinishNonExhaustive closes the struct and marks fields as omitted.
func (d *DebugStruct) FinishNonExhaustive() string {
	if d.fields > 0 {
		d.b.WriteString(", .. }")
	} else {
		d.b.WriteString(" { .. }")
	}
	return d.b.String()
}
