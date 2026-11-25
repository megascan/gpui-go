package taffy

import "math"

// AvailableSpaceKind is the variant of an AvailableSpace.
type AvailableSpaceKind uint8

const (
	AvailableSpaceKindDefinite AvailableSpaceKind = iota
	AvailableSpaceKindMinContent
	AvailableSpaceKindMaxContent
)

// AvailableSpace is the amount of space available to a node in a given axis.
type AvailableSpace struct {
	kind  AvailableSpaceKind
	value float32
}

// AvailableSpaceDefinite is the Definite variant: the specified number of pixels.
func AvailableSpaceDefinite(value float32) AvailableSpace {
	return AvailableSpace{kind: AvailableSpaceKindDefinite, value: value}
}

var (
	// AvailableSpaceMinContent is the MinContent variant: layout under a min-content constraint.
	AvailableSpaceMinContent = AvailableSpace{kind: AvailableSpaceKindMinContent}
	// AvailableSpaceMaxContent is the MaxContent variant: layout under a max-content constraint.
	AvailableSpaceMaxContent = AvailableSpace{kind: AvailableSpaceKindMaxContent}
	// AvailableSpaceZero is Definite(0.0).
	AvailableSpaceZero = AvailableSpaceDefinite(0.0)
)

// Kind returns the variant.
func (a AvailableSpace) Kind() AvailableSpaceKind {
	return a.kind
}

// Zero returns AvailableSpaceZero.
func (AvailableSpace) Zero() AvailableSpace {
	return AvailableSpaceZero
}

// MaxContent returns AvailableSpaceMaxContent.
func (AvailableSpace) MaxContent() AvailableSpace {
	return AvailableSpaceMaxContent
}

// MinContent returns AvailableSpaceMinContent.
func (AvailableSpace) MinContent() AvailableSpace {
	return AvailableSpaceMinContent
}

// FromLength returns a definite value.
func (AvailableSpace) FromLength(value float64) AvailableSpace {
	return AvailableSpaceDefinite(float32(value))
}

// IsDefinite reports whether the value is definite.
func (a AvailableSpace) IsDefinite() bool {
	return a.kind == AvailableSpaceKindDefinite
}

// IntoOption returns Some for definite values and None for constraints.
func (a AvailableSpace) IntoOption() Option[float32] {
	if a.kind == AvailableSpaceKindDefinite {
		return some(a.value)
	}
	return none[float32]()
}

// UnwrapOr returns the definite value or def.
func (a AvailableSpace) UnwrapOr(def float32) float32 {
	return a.IntoOption().UnwrapOr(def)
}

// Unwrap returns the definite value and panics if the value is not definite.
func (a AvailableSpace) Unwrap() float32 {
	return a.IntoOption().Unwrap()
}

// Or returns a if it is definite and def otherwise.
func (a AvailableSpace) Or(def AvailableSpace) AvailableSpace {
	if a.kind == AvailableSpaceKindDefinite {
		return a
	}
	return def
}

// OrElse returns a if it is definite and the result of defaultCb otherwise.
func (a AvailableSpace) OrElse(defaultCb func() AvailableSpace) AvailableSpace {
	if a.kind == AvailableSpaceKindDefinite {
		return a
	}
	return defaultCb()
}

// UnwrapOrElse returns the definite value or the result of defaultCb.
func (a AvailableSpace) UnwrapOrElse(defaultCb func() float32) float32 {
	if v, ok := a.IntoOption().Get(); ok {
		return v
	}
	return defaultCb()
}

// MaybeSet returns Definite(value) if value is Some and a otherwise.
func (a AvailableSpace) MaybeSet(value Option[float32]) AvailableSpace {
	if v, ok := value.Get(); ok {
		return AvailableSpaceDefinite(v)
	}
	return a
}

// MapDefiniteValue applies mapFunction to a definite value.
func (a AvailableSpace) MapDefiniteValue(mapFunction func(float32) float32) AvailableSpace {
	if a.kind == AvailableSpaceKindDefinite {
		return AvailableSpaceDefinite(mapFunction(a.value))
	}
	return a
}

// ComputeFreeSpace computes the free space given usedSpace.
func (a AvailableSpace) ComputeFreeSpace(usedSpace float32) float32 {
	switch a.kind {
	case AvailableSpaceKindMaxContent:
		return float32(math.Inf(1))
	case AvailableSpaceKindMinContent:
		return 0.0
	default:
		return a.value - usedSpace
	}
}

// IsRoughlyEqual compares two values, treating definite values within f32::EPSILON of each other as equal.
func (a AvailableSpace) IsRoughlyEqual(other AvailableSpace) bool {
	switch {
	case a.kind == AvailableSpaceKindDefinite && other.kind == AvailableSpaceKindDefinite:
		return abs(a.value-other.value) < f32Epsilon
	case a.kind == AvailableSpaceKindMinContent && other.kind == AvailableSpaceKindMinContent:
		return true
	case a.kind == AvailableSpaceKindMaxContent && other.kind == AvailableSpaceKindMaxContent:
		return true
	}
	return false
}

const f32Epsilon float32 = 1.1920929e-07

// AvailableSpaceFromF32 returns Definite(value).
func AvailableSpaceFromF32(value float32) AvailableSpace {
	return AvailableSpaceDefinite(value)
}

// AvailableSpaceFromOptionF32 returns Definite for Some and MaxContent for None.
func AvailableSpaceFromOptionF32(option Option[float32]) AvailableSpace {
	if v, ok := option.Get(); ok {
		return AvailableSpaceDefinite(v)
	}
	return AvailableSpaceMaxContent
}

// SizeAvailableSpaceIntoOptions converts a Size[AvailableSpace] into a Size[Option[float32]].
func SizeAvailableSpaceIntoOptions(s Size[AvailableSpace]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: s.Width.IntoOption(), Height: s.Height.IntoOption()}
}

// SizeAvailableSpaceMaybeSet applies AvailableSpace.MaybeSet to each component.
func SizeAvailableSpaceMaybeSet(s Size[AvailableSpace], value Size[Option[float32]]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: s.Width.MaybeSet(value.Width), Height: s.Height.MaybeSet(value.Height)}
}
