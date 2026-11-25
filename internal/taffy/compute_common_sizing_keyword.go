package taffy

import "unsafe"

type sizingKeywordResolutionKind uint8

const (
	sizingKeywordResolutionKindMeasure sizingKeywordResolutionKind = iota
	sizingKeywordResolutionKindExact
)

type sizingKeywordResolution struct {
	kind    sizingKeywordResolutionKind
	measure AvailableSpace
	exact   float32
}

func sizingKeywordResolutionMeasure(space AvailableSpace) sizingKeywordResolution {
	return sizingKeywordResolution{kind: sizingKeywordResolutionKindMeasure, measure: space}
}

func sizingKeywordResolutionExact(value float32) sizingKeywordResolution {
	return sizingKeywordResolution{kind: sizingKeywordResolutionKindExact, exact: value}
}

func resolveSizingKeyword(
	style Dimension,
	stretchSize Option[float32],
	percentResolutionBasis Option[float32],
	calcResolver func(unsafe.Pointer, float32) float32,
) Option[sizingKeywordResolution] {
	_ = calcResolver
	switch style.Tag() {
	case CompactLengthMinContentTag:
		return some(sizingKeywordResolutionMeasure(AvailableSpaceMinContent))
	case CompactLengthMaxContentTag:
		return some(sizingKeywordResolutionMeasure(AvailableSpaceMaxContent))
	case CompactLengthFitContentPxTag:
		return some(sizingKeywordResolutionMeasure(AvailableSpaceDefinite(style.Value())))
	case CompactLengthFitContentPercentTag:
		if basis, ok := percentResolutionBasis.Get(); ok {
			return some(sizingKeywordResolutionMeasure(AvailableSpaceDefinite(float32(basis * style.Value()))))
		}
		return none[sizingKeywordResolution]()
	case CompactLengthFitContentKeywordTag:
		if size, ok := stretchSize.Get(); ok {
			return some(sizingKeywordResolutionMeasure(AvailableSpaceDefinite(size)))
		}
		return none[sizingKeywordResolution]()
	case CompactLengthStretchTag:
		if size, ok := stretchSize.Get(); ok {
			return some(sizingKeywordResolutionExact(size))
		}
		return none[sizingKeywordResolution]()
	default:
		return none[sizingKeywordResolution]()
	}
}

func resolveAbsoluteSizingKeywords(
	tree LayoutPartialTree,
	node NodeID,
	knownDimensions *Size[Option[float32]],
	sizeStyle Size[Dimension],
	areaSize Size[float32],
	inset Rect[Option[float32]],
	margin Rect[Option[float32]],
	sizingMode SizingMode,
) {
	stretchSize := Size[float32]{
		Width: f32Max(
			areaSize.Width-
				inset.Left.UnwrapOr(0.0)-
				inset.Right.UnwrapOr(0.0)-
				margin.Left.UnwrapOr(0.0)-
				margin.Right.UnwrapOr(0.0),
			0.0,
		),
		Height: f32Max(
			areaSize.Height-
				inset.Top.UnwrapOr(0.0)-
				inset.Bottom.UnwrapOr(0.0)-
				margin.Top.UnwrapOr(0.0)-
				margin.Bottom.UnwrapOr(0.0),
			0.0,
		),
	}
	calcFn := func(val unsafe.Pointer, basis float32) float32 { return calc(tree, val, basis) }

	keywordWidth := none[sizingKeywordResolution]()
	if knownDimensions.Width.IsNone() {
		keywordWidth = resolveSizingKeyword(sizeStyle.Width, some(stretchSize.Width), some(areaSize.Width), calcFn)
	}
	keywordHeight := none[sizingKeywordResolution]()
	if knownDimensions.Height.IsNone() {
		keywordHeight = resolveSizingKeyword(sizeStyle.Height, some(stretchSize.Height), some(areaSize.Height), calcFn)
	}

	someArea := areaSize.Map(func(v float32) Option[float32] { return some(v) })
	kw, wok := keywordWidth.Get()
	kh, hok := keywordHeight.Get()
	if wok && hok && kw.kind == sizingKeywordResolutionKindMeasure && kh.kind == sizingKeywordResolutionKindMeasure {
		measuredSize := measureChildSizeBoth(
			tree,
			node,
			SizeOptionF32None,
			someArea,
			Size[AvailableSpace]{Width: kw.measure, Height: kh.measure},
			sizingMode,
			LineBoolFalse,
		)
		*knownDimensions = measuredSize.Map(func(v float32) Option[float32] { return some(v) })
		return
	}
	if wok {
		switch kw.kind {
		case sizingKeywordResolutionKindExact:
			knownDimensions.Width = some(kw.exact)
		case sizingKeywordResolutionKindMeasure:
			knownDimensions.Width = some(measureChildSize(
				tree,
				node,
				*knownDimensions,
				someArea,
				Size[AvailableSpace]{Width: kw.measure, Height: AvailableSpaceDefinite(stretchSize.Height)},
				sizingMode,
				AbsoluteAxisHorizontal,
				LineBoolFalse,
			))
		}
	}
	if hok {
		switch kh.kind {
		case sizingKeywordResolutionKindExact:
			knownDimensions.Height = some(kh.exact)
		case sizingKeywordResolutionKindMeasure:
			width := AvailableSpaceDefinite(stretchSize.Width)
			if w, ok := knownDimensions.Width.Get(); ok {
				width = AvailableSpaceDefinite(w)
			}
			knownDimensions.Height = some(measureChildSize(
				tree,
				node,
				*knownDimensions,
				someArea,
				Size[AvailableSpace]{Width: width, Height: kh.measure},
				sizingMode,
				AbsoluteAxisVertical,
				LineBoolFalse,
			))
		}
	}
}
