package taffy

import (
	"math"
	"unsafe"
)

type gridTrackKind uint8

const (
	gridTrackKindTrack gridTrackKind = iota
	gridTrackKindGutter
)

type gridTrack struct {
	kind                       gridTrackKind
	isCollapsed                bool
	minTrackSizingFunction     MinTrackSizingFunction
	maxTrackSizingFunction     MaxTrackSizingFunction
	offset                     float32
	baseSize                   float32
	growthLimit                float32
	contentAlignmentAdjustment float32
	itemIncurredIncrease       float32
	baseSizePlannedIncrease    float32
	growthLimitPlannedIncrease float32
	infinitelyGrowable         bool
	fitContentLimit            float32
}

func newGridTrackWithKind(kind gridTrackKind, minTrackSizingFunction MinTrackSizingFunction, maxTrackSizingFunction MaxTrackSizingFunction) gridTrack {
	return gridTrack{
		kind:                       kind,
		isCollapsed:                false,
		minTrackSizingFunction:     minTrackSizingFunction,
		maxTrackSizingFunction:     maxTrackSizingFunction,
		offset:                     0.0,
		baseSize:                   0.0,
		growthLimit:                0.0,
		contentAlignmentAdjustment: 0.0,
		itemIncurredIncrease:       0.0,
		baseSizePlannedIncrease:    0.0,
		growthLimitPlannedIncrease: 0.0,
		infinitelyGrowable:         false,
		fitContentLimit:            float32(math.Inf(1)),
	}
}

func newGridTrack(minTrackSizingFunction MinTrackSizingFunction, maxTrackSizingFunction MaxTrackSizingFunction) gridTrack {
	return newGridTrackWithKind(gridTrackKindTrack, minTrackSizingFunction, maxTrackSizingFunction)
}

func gridTrackGutter(size LengthPercentage) gridTrack {
	return newGridTrackWithKind(
		gridTrackKindGutter,
		MinTrackSizingFunctionFromLengthPercentage(size),
		MaxTrackSizingFunctionFromLengthPercentage(size),
	)
}

func (t *gridTrack) collapse() {
	t.isCollapsed = true
	t.minTrackSizingFunction = MinTrackSizingFunctionZero
	t.maxTrackSizingFunction = MaxTrackSizingFunctionZero
	t.fitContentLimit = float32(math.Inf(1))
}

func (t *gridTrack) isFlexible() bool {
	return t.maxTrackSizingFunction.IsFr()
}

func (t *gridTrack) usesPercentage() bool {
	return t.minTrackSizingFunction.UsesPercentage() || t.maxTrackSizingFunction.UsesPercentage()
}

func (t *gridTrack) hasIntrinsicSizingFunction() bool {
	return t.minTrackSizingFunction.IsIntrinsic() || t.maxTrackSizingFunction.IsIntrinsic()
}

func (t *gridTrack) resolveFitContentLimit(axisInnerNodeSize Option[float32], calcResolver func(unsafe.Pointer, float32) float32) {
	t.fitContentLimit = t.maxTrackSizingFunction.FitContentLimit(axisInnerNodeSize, calcResolver).UnwrapOr(float32(math.Inf(1)))
}

func (t *gridTrack) getFitContentLimit() float32 {
	return t.fitContentLimit
}

func (t *gridTrack) fitContentLimitedGrowthLimit() float32 {
	return f32Min(t.growthLimit, t.fitContentLimit)
}

func (t *gridTrack) flexFactor() float32 {
	if t.maxTrackSizingFunction.IsFr() {
		return t.maxTrackSizingFunction.inner.Value()
	}
	return 0.0
}
