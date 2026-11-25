package taffy

import "unsafe"

// LengthPercentage is a length or a percentage.
type LengthPercentage struct {
	inner CompactLength
}

// LengthPercentageZero is 0 px.
var LengthPercentageZero = LengthPercentage{CompactLengthZero}

// Zero returns LengthPercentageZero.
func (LengthPercentage) Zero() LengthPercentage {
	return LengthPercentageZero
}

// FromLength returns a length value.
func (LengthPercentage) FromLength(value float64) LengthPercentage {
	return LengthPercentageLength(float32(value))
}

// FromPercent returns a percentage value.
func (LengthPercentage) FromPercent(value float64) LengthPercentage {
	return LengthPercentagePercent(float32(value))
}

// LengthPercentageLength is an absolute length in some abstract units.
func LengthPercentageLength(val float32) LengthPercentage {
	return LengthPercentage{CompactLengthLength(val)}
}

// LengthPercentagePercent is a percentage length relative to the size of the containing block, in the range [0.0, 1.0].
func LengthPercentagePercent(val float32) LengthPercentage {
	return LengthPercentage{CompactLengthPercent(val)}
}

// LengthPercentageFromRaw wraps a raw CompactLength without checking its tag.
func LengthPercentageFromRaw(val CompactLength) LengthPercentage {
	return LengthPercentage{val}
}

// IntoRaw returns the underlying CompactLength.
func (l LengthPercentage) IntoRaw() CompactLength {
	return l.inner
}

// Expand converts the value into an ExpandedLengthPercentage.
func (l LengthPercentage) Expand() ExpandedLengthPercentage {
	switch l.inner.Tag() {
	case CompactLengthLengthTag:
		return ExpandedLengthPercentageLength(l.inner.Value())
	case CompactLengthPercentTag:
		return ExpandedLengthPercentagePercent(l.inner.Value())
	}
	panic("internal error: entered unreachable code: LengthPercentage contains a value with an invalid tag")
}

// ExpandedLengthPercentageKind is the variant of an ExpandedLengthPercentage.
type ExpandedLengthPercentageKind uint8

const (
	ExpandedLengthPercentageKindLength ExpandedLengthPercentageKind = iota
	ExpandedLengthPercentageKindPercent
)

// ExpandedLengthPercentage is the matchable form of a LengthPercentage.
type ExpandedLengthPercentage struct {
	kind  ExpandedLengthPercentageKind
	value float32
}

// ExpandedLengthPercentageLength is the Length variant.
func ExpandedLengthPercentageLength(val float32) ExpandedLengthPercentage {
	return ExpandedLengthPercentage{kind: ExpandedLengthPercentageKindLength, value: val}
}

// ExpandedLengthPercentagePercent is the Percent variant.
func ExpandedLengthPercentagePercent(val float32) ExpandedLengthPercentage {
	return ExpandedLengthPercentage{kind: ExpandedLengthPercentageKindPercent, value: val}
}

// Kind returns the variant.
func (e ExpandedLengthPercentage) Kind() ExpandedLengthPercentageKind {
	return e.kind
}

// Value returns the f32 payload of the Length and Percent variants.
func (e ExpandedLengthPercentage) Value() float32 {
	return e.value
}

// ExpandedLengthPercentageFromLengthPercentage converts value with Expand.
func ExpandedLengthPercentageFromLengthPercentage(value LengthPercentage) ExpandedLengthPercentage {
	return value.Expand()
}

// LengthPercentageFromExpandedLengthPercentage converts an expanded value back.
func LengthPercentageFromExpandedLengthPercentage(value ExpandedLengthPercentage) LengthPercentage {
	switch value.kind {
	case ExpandedLengthPercentageKindLength:
		return LengthPercentageLength(value.value)
	default:
		return LengthPercentagePercent(value.value)
	}
}

// LengthPercentageAuto is a length, a percentage or auto.
type LengthPercentageAuto struct {
	inner CompactLength
}

// LengthPercentageAutoZero is 0 px.
var LengthPercentageAutoZero = LengthPercentageAuto{CompactLengthZero}

// Zero returns LengthPercentageAutoZero.
func (LengthPercentageAuto) Zero() LengthPercentageAuto {
	return LengthPercentageAutoZero
}

// Auto returns LengthPercentageAutoAuto().
func (LengthPercentageAuto) Auto() LengthPercentageAuto {
	return LengthPercentageAutoAuto()
}

// FromLength returns a length value.
func (LengthPercentageAuto) FromLength(value float64) LengthPercentageAuto {
	return LengthPercentageAutoLength(float32(value))
}

// FromPercent returns a percentage value.
func (LengthPercentageAuto) FromPercent(value float64) LengthPercentageAuto {
	return LengthPercentageAutoPercent(float32(value))
}

// LengthPercentageAutoFromLengthPercentage converts a LengthPercentage.
func LengthPercentageAutoFromLengthPercentage(input LengthPercentage) LengthPercentageAuto {
	return LengthPercentageAuto{input.inner}
}

// LengthPercentageAutoLength is an absolute length in some abstract units.
func LengthPercentageAutoLength(val float32) LengthPercentageAuto {
	return LengthPercentageAuto{CompactLengthLength(val)}
}

// LengthPercentageAutoPercent is a percentage length relative to the size of the containing block, in the range [0.0, 1.0].
func LengthPercentageAutoPercent(val float32) LengthPercentageAuto {
	return LengthPercentageAuto{CompactLengthPercent(val)}
}

// LengthPercentageAutoAuto is the auto value.
func LengthPercentageAutoAuto() LengthPercentageAuto {
	return LengthPercentageAuto{CompactLengthAuto()}
}

// LengthPercentageAutoFromRaw wraps a raw CompactLength without checking its tag.
func LengthPercentageAutoFromRaw(val CompactLength) LengthPercentageAuto {
	return LengthPercentageAuto{val}
}

// IntoRaw returns the underlying CompactLength.
func (l LengthPercentageAuto) IntoRaw() CompactLength {
	return l.inner
}

// ResolveToOption returns the length, the percentage of context, or None for auto.
func (l LengthPercentageAuto) ResolveToOption(context float32, calcResolver func(unsafe.Pointer, float32) float32) Option[float32] {
	switch l.inner.Tag() {
	case CompactLengthLengthTag:
		return some(l.inner.Value())
	case CompactLengthPercentTag:
		return some(float32(context * l.inner.Value()))
	case CompactLengthAutoTag:
		return none[float32]()
	}
	panic("internal error: entered unreachable code: LengthPercentageAuto values cannot be constructed with other tags")
}

// IsAuto reports whether the value is auto.
func (l LengthPercentageAuto) IsAuto() bool {
	return l.inner.IsAuto()
}

// Expand converts the value into an ExpandedLengthPercentageAuto.
func (l LengthPercentageAuto) Expand() ExpandedLengthPercentageAuto {
	switch l.inner.Tag() {
	case CompactLengthLengthTag:
		return ExpandedLengthPercentageAutoLength(l.inner.Value())
	case CompactLengthPercentTag:
		return ExpandedLengthPercentageAutoPercent(l.inner.Value())
	case CompactLengthAutoTag:
		return ExpandedLengthPercentageAutoAuto
	}
	panic("internal error: entered unreachable code: LengthPercentageAuto contains a value with an invalid tag")
}

// ExpandedLengthPercentageAutoKind is the variant of an ExpandedLengthPercentageAuto.
type ExpandedLengthPercentageAutoKind uint8

const (
	ExpandedLengthPercentageAutoKindLength ExpandedLengthPercentageAutoKind = iota
	ExpandedLengthPercentageAutoKindPercent
	ExpandedLengthPercentageAutoKindAuto
)

// ExpandedLengthPercentageAuto is the matchable form of a LengthPercentageAuto.
type ExpandedLengthPercentageAuto struct {
	kind  ExpandedLengthPercentageAutoKind
	value float32
}

// ExpandedLengthPercentageAutoLength is the Length variant.
func ExpandedLengthPercentageAutoLength(val float32) ExpandedLengthPercentageAuto {
	return ExpandedLengthPercentageAuto{kind: ExpandedLengthPercentageAutoKindLength, value: val}
}

// ExpandedLengthPercentageAutoPercent is the Percent variant.
func ExpandedLengthPercentageAutoPercent(val float32) ExpandedLengthPercentageAuto {
	return ExpandedLengthPercentageAuto{kind: ExpandedLengthPercentageAutoKindPercent, value: val}
}

// ExpandedLengthPercentageAutoAuto is the Auto variant.
var ExpandedLengthPercentageAutoAuto = ExpandedLengthPercentageAuto{kind: ExpandedLengthPercentageAutoKindAuto}

// Kind returns the variant.
func (e ExpandedLengthPercentageAuto) Kind() ExpandedLengthPercentageAutoKind {
	return e.kind
}

// Value returns the f32 payload of the Length and Percent variants.
func (e ExpandedLengthPercentageAuto) Value() float32 {
	return e.value
}

// ExpandedLengthPercentageAutoFromLengthPercentageAuto converts value with Expand.
func ExpandedLengthPercentageAutoFromLengthPercentageAuto(value LengthPercentageAuto) ExpandedLengthPercentageAuto {
	return value.Expand()
}

// LengthPercentageAutoFromExpandedLengthPercentageAuto converts an expanded value back.
func LengthPercentageAutoFromExpandedLengthPercentageAuto(value ExpandedLengthPercentageAuto) LengthPercentageAuto {
	switch value.kind {
	case ExpandedLengthPercentageAutoKindLength:
		return LengthPercentageAutoLength(value.value)
	case ExpandedLengthPercentageAutoKindPercent:
		return LengthPercentageAutoPercent(value.value)
	default:
		return LengthPercentageAutoAuto()
	}
}

// Dimension is a unit of linear measurement.
type Dimension struct {
	inner CompactLength
}

// DimensionZero is 0 px.
var DimensionZero = Dimension{CompactLengthZero}

// Zero returns DimensionZero.
func (Dimension) Zero() Dimension {
	return DimensionZero
}

// Auto returns DimensionAuto().
func (Dimension) Auto() Dimension {
	return DimensionAuto()
}

// FromLength returns a length value.
func (Dimension) FromLength(value float64) Dimension {
	return DimensionLength(float32(value))
}

// FromPercent returns a percentage value.
func (Dimension) FromPercent(value float64) Dimension {
	return DimensionPercent(float32(value))
}

// DimensionFromLengthPercentage converts a LengthPercentage.
func DimensionFromLengthPercentage(input LengthPercentage) Dimension {
	return Dimension{input.inner}
}

// DimensionFromLengthPercentageAuto converts a LengthPercentageAuto.
func DimensionFromLengthPercentageAuto(input LengthPercentageAuto) Dimension {
	return Dimension{input.inner}
}

// DimensionLength is an absolute length in some abstract units.
func DimensionLength(val float32) Dimension {
	return Dimension{CompactLengthLength(val)}
}

// DimensionPercent is a percentage length relative to the size of the containing block, in the range [0.0, 1.0].
func DimensionPercent(val float32) Dimension {
	return Dimension{CompactLengthPercent(val)}
}

// DimensionAuto is the auto value.
func DimensionAuto() Dimension {
	return Dimension{CompactLengthAuto()}
}

// DimensionMinContent is the min-content size.
func DimensionMinContent() Dimension {
	return Dimension{CompactLengthMinContent()}
}

// DimensionMaxContent is the max-content size.
func DimensionMaxContent() Dimension {
	return Dimension{CompactLengthMaxContent()}
}

// DimensionFitContent is the plain fit-content keyword.
func DimensionFitContent() Dimension {
	return Dimension{CompactLengthFitContentKeyword()}
}

// DimensionFitContentPx is fit-content with a length limit.
func DimensionFitContentPx(limit float32) Dimension {
	return Dimension{CompactLengthFitContentPx(limit)}
}

// DimensionStretch is the stretch-fit size.
func DimensionStretch() Dimension {
	return Dimension{CompactLengthStretch()}
}

// DimensionContent is the content keyword, valid only for flex-basis.
func DimensionContent() Dimension {
	return Dimension{CompactLengthContent()}
}

// DimensionFitContentPercent is fit-content with a percentage limit.
func DimensionFitContentPercent(limit float32) Dimension {
	return Dimension{CompactLengthFitContentPercent(limit)}
}

// IsFitContentCalc reports whether the value is a fit-content() value with a calc() limit.
func (d Dimension) IsFitContentCalc() bool {
	return d.inner.IsFitContentCalc()
}

// DimensionFromRaw wraps a raw CompactLength without checking its tag.
func DimensionFromRaw(val CompactLength) Dimension {
	return Dimension{val}
}

// IntoRaw returns the underlying CompactLength.
func (d Dimension) IntoRaw() CompactLength {
	return d.inner
}

// IntoOption returns Some for length values and None otherwise.
func (d Dimension) IntoOption() Option[float32] {
	switch d.inner.Tag() {
	case CompactLengthLengthTag:
		return some(d.inner.Value())
	}
	return none[float32]()
}

// IsAuto reports whether the value is auto.
func (d Dimension) IsAuto() bool {
	return d.inner.IsAuto()
}

// IsSizingKeyword reports whether the value is a CSS sizing keyword.
func (d Dimension) IsSizingKeyword() bool {
	return d.inner.IsSizingKeyword()
}

// IsStretch reports whether the value is stretch.
func (d Dimension) IsStretch() bool {
	return d.inner.Tag() == CompactLengthStretchTag
}

// IsContent reports whether the value is the content keyword.
func (d Dimension) IsContent() bool {
	return d.inner.IsContent()
}

// Tag returns the underlying tag.
func (d Dimension) Tag() int {
	return d.inner.Tag()
}

// Value returns the underlying numeric value.
func (d Dimension) Value() float32 {
	return d.inner.Value()
}

// Expand converts the value into an ExpandedDimension.
func (d Dimension) Expand() ExpandedDimension {
	switch d.inner.Tag() {
	case CompactLengthLengthTag:
		return ExpandedDimensionLength(d.inner.Value())
	case CompactLengthPercentTag:
		return ExpandedDimensionPercent(d.inner.Value())
	case CompactLengthAutoTag:
		return ExpandedDimensionAuto
	case CompactLengthMinContentTag:
		return ExpandedDimensionMinContent
	case CompactLengthMaxContentTag:
		return ExpandedDimensionMaxContent
	case CompactLengthFitContentPxTag:
		return ExpandedDimensionFitContentPx(d.inner.Value())
	case CompactLengthFitContentPercentTag:
		return ExpandedDimensionFitContentPercent(d.inner.Value())
	case CompactLengthFitContentKeywordTag:
		return ExpandedDimensionFitContent
	case CompactLengthStretchTag:
		return ExpandedDimensionStretch
	case CompactLengthContentTag:
		return ExpandedDimensionContent
	}
	panic("internal error: entered unreachable code: Dimension contains a value with an invalid tag")
}

// ExpandedDimensionKind is the variant of an ExpandedDimension.
type ExpandedDimensionKind uint8

const (
	ExpandedDimensionKindLength ExpandedDimensionKind = iota
	ExpandedDimensionKindPercent
	ExpandedDimensionKindAuto
	ExpandedDimensionKindMinContent
	ExpandedDimensionKindMaxContent
	ExpandedDimensionKindFitContentPx
	ExpandedDimensionKindFitContentPercent
	ExpandedDimensionKindFitContent
	ExpandedDimensionKindStretch
	ExpandedDimensionKindContent
)

// ExpandedDimension is the matchable form of a Dimension.
type ExpandedDimension struct {
	kind  ExpandedDimensionKind
	value float32
}

// ExpandedDimensionLength is the Length variant.
func ExpandedDimensionLength(val float32) ExpandedDimension {
	return ExpandedDimension{kind: ExpandedDimensionKindLength, value: val}
}

// ExpandedDimensionPercent is the Percent variant.
func ExpandedDimensionPercent(val float32) ExpandedDimension {
	return ExpandedDimension{kind: ExpandedDimensionKindPercent, value: val}
}

// ExpandedDimensionFitContentPx is the FitContentPx variant.
func ExpandedDimensionFitContentPx(val float32) ExpandedDimension {
	return ExpandedDimension{kind: ExpandedDimensionKindFitContentPx, value: val}
}

// ExpandedDimensionFitContentPercent is the FitContentPercent variant.
func ExpandedDimensionFitContentPercent(val float32) ExpandedDimension {
	return ExpandedDimension{kind: ExpandedDimensionKindFitContentPercent, value: val}
}

var (
	// ExpandedDimensionAuto is the Auto variant.
	ExpandedDimensionAuto = ExpandedDimension{kind: ExpandedDimensionKindAuto}
	// ExpandedDimensionMinContent is the MinContent variant.
	ExpandedDimensionMinContent = ExpandedDimension{kind: ExpandedDimensionKindMinContent}
	// ExpandedDimensionMaxContent is the MaxContent variant.
	ExpandedDimensionMaxContent = ExpandedDimension{kind: ExpandedDimensionKindMaxContent}
	// ExpandedDimensionFitContent is the FitContent variant.
	ExpandedDimensionFitContent = ExpandedDimension{kind: ExpandedDimensionKindFitContent}
	// ExpandedDimensionStretch is the Stretch variant.
	ExpandedDimensionStretch = ExpandedDimension{kind: ExpandedDimensionKindStretch}
	// ExpandedDimensionContent is the Content variant.
	ExpandedDimensionContent = ExpandedDimension{kind: ExpandedDimensionKindContent}
)

// Kind returns the variant.
func (e ExpandedDimension) Kind() ExpandedDimensionKind {
	return e.kind
}

// Value returns the f32 payload of the Length, Percent, FitContentPx and FitContentPercent variants.
func (e ExpandedDimension) Value() float32 {
	return e.value
}

// ExpandedDimensionFromDimension converts value with Expand.
func ExpandedDimensionFromDimension(value Dimension) ExpandedDimension {
	return value.Expand()
}

// DimensionFromExpandedDimension converts an expanded value back.
func DimensionFromExpandedDimension(value ExpandedDimension) Dimension {
	switch value.kind {
	case ExpandedDimensionKindLength:
		return DimensionLength(value.value)
	case ExpandedDimensionKindPercent:
		return DimensionPercent(value.value)
	case ExpandedDimensionKindAuto:
		return DimensionAuto()
	case ExpandedDimensionKindMinContent:
		return DimensionMinContent()
	case ExpandedDimensionKindMaxContent:
		return DimensionMaxContent()
	case ExpandedDimensionKindFitContentPx:
		return DimensionFitContentPx(value.value)
	case ExpandedDimensionKindFitContentPercent:
		return DimensionFitContentPercent(value.value)
	case ExpandedDimensionKindFitContent:
		return DimensionFitContent()
	case ExpandedDimensionKindStretch:
		return DimensionStretch()
	default:
		return DimensionContent()
	}
}

// RectDimensionFromLength returns a Rect[Dimension] of length values.
func RectDimensionFromLength(start, end, top, bottom float32) Rect[Dimension] {
	return Rect[Dimension]{
		Left:   Dimension{CompactLengthLength(start)},
		Right:  Dimension{CompactLengthLength(end)},
		Top:    Dimension{CompactLengthLength(top)},
		Bottom: Dimension{CompactLengthLength(bottom)},
	}
}

// RectDimensionFromPercent returns a Rect[Dimension] of percentage values.
func RectDimensionFromPercent(start, end, top, bottom float32) Rect[Dimension] {
	return Rect[Dimension]{
		Left:   Dimension{CompactLengthPercent(start)},
		Right:  Dimension{CompactLengthPercent(end)},
		Top:    Dimension{CompactLengthPercent(top)},
		Bottom: Dimension{CompactLengthPercent(bottom)},
	}
}
