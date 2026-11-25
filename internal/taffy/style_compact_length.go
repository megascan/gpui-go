package taffy

import (
	"math"
	"unsafe"
)

// CompactLength is a length style value: an 8-bit tag plus an f32 payload, compared bitwise as upstream's tagged pointer is.
type CompactLength struct {
	tag  uint8
	bits uint32
}

func compactLengthFromVal(val float32, tag int) CompactLength {
	return CompactLength{tag: uint8(tag), bits: math.Float32bits(val)}
}

func compactLengthFromTag(tag int) CompactLength {
	return CompactLength{tag: uint8(tag)}
}

const (
	CompactLengthNonPointerBit    = 0b0001
	CompactLengthKeywordBit       = 0b0010
	CompactLengthPercentageBit    = 0b0100
	CompactLengthSizingKeywordBit = 0b1000

	CompactLengthLengthTag            = 0b0000_0001
	CompactLengthPercentTag           = 0b0000_0101
	CompactLengthAutoTag              = 0b0000_0011
	CompactLengthFrTag                = 0b0001_0011
	CompactLengthContentTag           = 0b0010_0011
	CompactLengthMinContentTag        = 0b0000_1011
	CompactLengthMaxContentTag        = 0b0001_1011
	CompactLengthFitContentPxTag      = 0b0010_1011
	CompactLengthFitContentKeywordTag = 0b0011_1011
	CompactLengthStretchTag           = 0b0100_1011
	CompactLengthFitContentPercentTag = 0b0000_1111
)

// CompactLengthLength is an absolute length in some abstract units.
func CompactLengthLength(val float32) CompactLength {
	return compactLengthFromVal(val, CompactLengthLengthTag)
}

// CompactLengthPercent is a percentage length relative to the size of the containing block, in the range [0.0, 1.0].
func CompactLengthPercent(val float32) CompactLength {
	return compactLengthFromVal(val, CompactLengthPercentTag)
}

// CompactLengthAuto is the auto value.
func CompactLengthAuto() CompactLength {
	return compactLengthFromTag(CompactLengthAutoTag)
}

// CompactLengthFr is a fraction of the total available grid space.
func CompactLengthFr(val float32) CompactLength {
	return compactLengthFromVal(val, CompactLengthFrTag)
}

// CompactLengthMinContent is the min-content size.
func CompactLengthMinContent() CompactLength {
	return compactLengthFromTag(CompactLengthMinContentTag)
}

// CompactLengthMaxContent is the max-content size.
func CompactLengthMaxContent() CompactLength {
	return compactLengthFromTag(CompactLengthMaxContentTag)
}

// CompactLengthFitContentPx is fit-content with a length limit.
func CompactLengthFitContentPx(limit float32) CompactLength {
	return compactLengthFromVal(limit, CompactLengthFitContentPxTag)
}

// CompactLengthFitContentPercent is fit-content with a percentage limit.
func CompactLengthFitContentPercent(limit float32) CompactLength {
	return compactLengthFromVal(limit, CompactLengthFitContentPercentTag)
}

// CompactLengthFitContentKeyword is the plain fit-content keyword.
func CompactLengthFitContentKeyword() CompactLength {
	return compactLengthFromTag(CompactLengthFitContentKeywordTag)
}

// CompactLengthStretch is the stretch-fit size.
func CompactLengthStretch() CompactLength {
	return compactLengthFromTag(CompactLengthStretchTag)
}

// CompactLengthContent is the content keyword, valid only for flex-basis.
func CompactLengthContent() CompactLength {
	return compactLengthFromTag(CompactLengthContentTag)
}

// Tag returns the primary tag.
func (c CompactLength) Tag() int {
	return int(c.tag)
}

// Value returns the numeric value associated with the CompactLength.
func (c CompactLength) Value() float32 {
	return math.Float32frombits(c.bits)
}

// IsFitContentCalc reports whether the value is a fit-content() value with a calc() limit.
func (c CompactLength) IsFitContentCalc() bool {
	return c.Tag()&(CompactLengthNonPointerBit|CompactLengthKeywordBit) == CompactLengthKeywordBit
}

// HasCalc reports whether the value holds a calc() pointer.
func (c CompactLength) HasCalc() bool {
	return c.Tag()&CompactLengthNonPointerBit == 0
}

// IsZero reports whether the value is 0 px.
func (c CompactLength) IsZero() bool {
	return c == CompactLengthZero
}

// IsLengthOrPercentage reports whether the value is a length or percentage value.
func (c CompactLength) IsLengthOrPercentage() bool {
	return c.Tag()&(CompactLengthNonPointerBit|CompactLengthKeywordBit) == CompactLengthNonPointerBit
}

// IsAuto reports whether the value is auto.
func (c CompactLength) IsAuto() bool {
	return c.Tag() == CompactLengthAutoTag
}

// IsContent reports whether the value is the content keyword.
func (c CompactLength) IsContent() bool {
	return c.Tag() == CompactLengthContentTag
}

// IsMinContent reports whether the value is min-content.
func (c CompactLength) IsMinContent() bool {
	return c.Tag() == CompactLengthMinContentTag
}

// IsMaxContent reports whether the value is max-content.
func (c CompactLength) IsMaxContent() bool {
	return c.Tag() == CompactLengthMaxContentTag
}

// IsFitContent reports whether the value is a fit-content(...) value.
func (c CompactLength) IsFitContent() bool {
	switch c.Tag() {
	case CompactLengthFitContentPxTag, CompactLengthFitContentPercentTag:
		return true
	}
	return c.IsFitContentCalc()
}

// IsSizingKeyword reports whether the value is min-content, max-content, fit-content, fit-content(...) or stretch.
func (c CompactLength) IsSizingKeyword() bool {
	const mask = CompactLengthNonPointerBit | CompactLengthKeywordBit | CompactLengthSizingKeywordBit
	return c.Tag()&mask == mask || c.IsFitContentCalc()
}

// IsMaxOrFitContent reports whether the value is max-content or a fit-content(...) value.
func (c CompactLength) IsMaxOrFitContent() bool {
	switch c.Tag() {
	case CompactLengthMaxContentTag, CompactLengthFitContentPxTag, CompactLengthFitContentPercentTag:
		return true
	}
	return c.IsFitContentCalc()
}

// IsMaxContentAlike reports whether the max track sizing function is max-content, fit-content or auto.
func (c CompactLength) IsMaxContentAlike() bool {
	switch c.Tag() {
	case CompactLengthAutoTag, CompactLengthMaxContentTag, CompactLengthFitContentPxTag, CompactLengthFitContentPercentTag:
		return true
	}
	return c.IsFitContentCalc()
}

// IsMinOrMaxContent reports whether the min track sizing function is min-content or max-content.
func (c CompactLength) IsMinOrMaxContent() bool {
	switch c.Tag() {
	case CompactLengthMinContentTag, CompactLengthMaxContentTag:
		return true
	}
	return false
}

// IsIntrinsic reports whether the value is auto, min-content, max-content or fit-content(...).
func (c CompactLength) IsIntrinsic() bool {
	switch c.Tag() {
	case CompactLengthAutoTag, CompactLengthMinContentTag, CompactLengthMaxContentTag, CompactLengthFitContentPxTag, CompactLengthFitContentPercentTag:
		return true
	}
	return c.IsFitContentCalc()
}

// IsFr reports whether the value is an fr value.
func (c CompactLength) IsFr() bool {
	return c.Tag() == CompactLengthFrTag
}

// UsesPercentage reports whether the value depends on the size of the parent node.
func (c CompactLength) UsesPercentage() bool {
	return c.Tag()&CompactLengthPercentageBit != 0
}

// ResolvedPercentageSize resolves percentage values against parentSize; other values return None.
func (c CompactLength) ResolvedPercentageSize(parentSize float32, calcResolver func(unsafe.Pointer, float32) float32) Option[float32] {
	switch c.Tag() {
	case CompactLengthPercentTag:
		return some(float32(c.Value() * parentSize))
	}
	return none[float32]()
}

// CompactLengthZero is 0 px.
var CompactLengthZero = CompactLengthLength(0.0)

// Zero returns CompactLengthZero.
func (CompactLength) Zero() CompactLength {
	return CompactLengthZero
}

// Auto returns CompactLengthAuto().
func (CompactLength) Auto() CompactLength {
	return CompactLengthAuto()
}

// MinContent returns CompactLengthMinContent().
func (CompactLength) MinContent() CompactLength {
	return CompactLengthMinContent()
}

// MaxContent returns CompactLengthMaxContent().
func (CompactLength) MaxContent() CompactLength {
	return CompactLengthMaxContent()
}

// FromLength returns a length value.
func (CompactLength) FromLength(value float64) CompactLength {
	return CompactLengthLength(float32(value))
}

// FromPercent returns a percentage value.
func (CompactLength) FromPercent(value float64) CompactLength {
	return CompactLengthPercent(float32(value))
}

// FromFr returns an fr value.
func (CompactLength) FromFr(value float64) CompactLength {
	return CompactLengthFr(float32(value))
}

// FitContent returns fit-content with the limit lp.
func (CompactLength) FitContent(lp LengthPercentage) CompactLength {
	value := lp.inner.Value()
	switch lp.inner.Tag() {
	case CompactLengthLengthTag:
		return CompactLengthFitContentPx(value)
	case CompactLengthPercentTag:
		return CompactLengthFitContentPercent(value)
	}
	panic("internal error: entered unreachable code")
}
