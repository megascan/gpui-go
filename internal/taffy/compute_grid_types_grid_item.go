package taffy

import "unsafe"

type knownDimensionsCacheEntry struct {
	gridAreaSize    Size[Option[float32]]
	knownDimensions Size[Option[float32]]
}

type gridItem struct {
	node                        NodeID
	sourceOrder                 uint16
	row                         Line[OriginZeroLine]
	column                      Line[OriginZeroLine]
	isCompressibleReplaced      bool
	isReplaced                  bool
	overflow                    Point[Overflow]
	boxSizing                   BoxSizing
	size                        Size[Dimension]
	minSize                     Size[LengthPercentageAuto]
	maxSize                     Size[LengthPercentageAuto]
	aspectRatio                 Option[float32]
	padding                     Rect[LengthPercentage]
	border                      Rect[LengthPercentage]
	margin                      Rect[LengthPercentageAuto]
	alignSelf                   AlignSelf
	justifySelf                 AlignSelf
	baseline                    Option[float32]
	baselineShim                float32
	rowIndexes                  Line[uint16]
	columnIndexes               Line[uint16]
	crossesFlexibleRow          bool
	crossesFlexibleColumn       bool
	crossesIntrinsicRow         bool
	crossesIntrinsicColumn      bool
	gridAreaSizeCache           Option[Size[Option[float32]]]
	knownDimensionsCache        Option[knownDimensionsCacheEntry]
	minContentContributionCache Size[Option[float32]]
	minimumContributionCache    Size[Option[float32]]
	maxContentContributionCache Size[Option[float32]]
	yPosition                   float32
	height                      float32
	oofCandidates               OofCandidates
}

func newGridItemWithStyleAndOrder[S CheapCloneStr](node NodeID, style GridItemStyle[S], parentAlignItems AlignItems, parentJustifyItems AlignItems, sourceOrder uint16) gridItem {
	unplaced := Line[OriginZeroLine]{Start: 0, End: 0}
	return gridItem{
		node:                        node,
		sourceOrder:                 sourceOrder,
		row:                         unplaced,
		column:                      unplaced,
		isCompressibleReplaced:      style.IsCompressibleReplaced(),
		isReplaced:                  style.IsReplaced(),
		overflow:                    style.Overflow(),
		boxSizing:                   style.BoxSizing(),
		size:                        style.Size(),
		minSize:                     style.MinSize(),
		maxSize:                     style.MaxSize(),
		aspectRatio:                 style.AspectRatio(),
		padding:                     style.Padding(),
		border:                      style.Border(),
		margin:                      style.Margin(),
		alignSelf:                   style.AlignSelf().UnwrapOr(parentAlignItems),
		justifySelf:                 style.JustifySelf().UnwrapOr(parentJustifyItems),
		baseline:                    none[float32](),
		baselineShim:                0.0,
		rowIndexes:                  Line[uint16]{Start: 0, End: 0},
		columnIndexes:               Line[uint16]{Start: 0, End: 0},
		crossesFlexibleRow:          false,
		crossesFlexibleColumn:       false,
		crossesIntrinsicRow:         false,
		crossesIntrinsicColumn:      false,
		gridAreaSizeCache:           none[Size[Option[float32]]](),
		minContentContributionCache: SizeOptionF32None,
		maxContentContributionCache: SizeOptionF32None,
		knownDimensionsCache:        none[knownDimensionsCacheEntry](),
		minimumContributionCache:    SizeOptionF32None,
		yPosition:                   0.0,
		height:                      0.0,
		oofCandidates:               OofCandidatesNone,
	}
}

func (g *gridItem) hasAutoBlockMargin() bool {
	return g.margin.Top.IsAuto() || g.margin.Bottom.IsAuto()
}

func (g *gridItem) hasCyclicBlockSizeDependency() bool {
	return g.size.Height.inner.UsesPercentage() && (g.crossesIntrinsicRow || g.crossesFlexibleRow)
}

func (g *gridItem) participatesInBaselineAlignment() bool {
	return g.alignSelf.keyword == AlignItemsKeywordBaseline &&
		!g.hasAutoBlockMargin() &&
		!g.hasCyclicBlockSizeDependency()
}

func (g *gridItem) placement(axis AbstractAxis) Line[OriginZeroLine] {
	switch axis {
	case AbstractAxisBlock:
		return g.row
	default:
		return g.column
	}
}

func (g *gridItem) placementIndexes(axis AbstractAxis) Line[uint16] {
	switch axis {
	case AbstractAxisBlock:
		return g.rowIndexes
	default:
		return g.columnIndexes
	}
}

func (g *gridItem) trackRangeExcludingLines(axis AbstractAxis) (int, int) {
	indexes := g.placementIndexes(axis)
	return int(indexes.Start) + 1, int(indexes.End)
}

func (g *gridItem) spansTrackMatching(axis AbstractAxis, axisTracks []gridTrack, predicate func(*gridTrack) bool) bool {
	start, end := g.trackRangeExcludingLines(axis)
	tracks := axisTracks[start:end]
	for i := range tracks {
		if predicate(&tracks[i]) {
			return true
		}
	}
	return false
}

func (g *gridItem) span(axis AbstractAxis) uint16 {
	switch axis {
	case AbstractAxisBlock:
		return lineOriginZeroLineSpan(g.row)
	default:
		return lineOriginZeroLineSpan(g.column)
	}
}

func (g *gridItem) crossesFlexibleTrack(axis AbstractAxis) bool {
	switch axis {
	case AbstractAxisInline:
		return g.crossesFlexibleColumn
	default:
		return g.crossesFlexibleRow
	}
}

func (g *gridItem) crossesIntrinsicTrack(axis AbstractAxis) bool {
	switch axis {
	case AbstractAxisInline:
		return g.crossesIntrinsicColumn
	default:
		return g.crossesIntrinsicRow
	}
}

func (g *gridItem) spannedTrackLimit(axis AbstractAxis, axisTracks []gridTrack, axisParentSize Option[float32], resolveCalcValue func(unsafe.Pointer, float32) float32) Option[float32] {
	start, end := g.trackRangeExcludingLines(axis)
	spannedTracks := axisTracks[start:end]
	tracksAllFixed := true
	for i := range spannedTracks {
		if spannedTracks[i].maxTrackSizingFunction.DefiniteLimit(axisParentSize, resolveCalcValue).IsNone() {
			tracksAllFixed = false
			break
		}
	}
	if tracksAllFixed {
		var limit float32
		for i := range spannedTracks {
			limit += spannedTracks[i].maxTrackSizingFunction.DefiniteLimit(axisParentSize, resolveCalcValue).Unwrap()
		}
		return some(limit)
	}
	return none[float32]()
}

func (g *gridItem) spannedFixedTrackLimit(axis AbstractAxis, axisTracks []gridTrack, axisParentSize Option[float32], resolveCalcValue func(unsafe.Pointer, float32) float32) Option[float32] {
	start, end := g.trackRangeExcludingLines(axis)
	spannedTracks := axisTracks[start:end]
	tracksAllFixed := true
	for i := range spannedTracks {
		if spannedTracks[i].maxTrackSizingFunction.DefiniteValue(axisParentSize, resolveCalcValue).IsNone() {
			tracksAllFixed = false
			break
		}
	}
	if tracksAllFixed {
		var limit float32
		for i := range spannedTracks {
			limit += spannedTracks[i].maxTrackSizingFunction.DefiniteValue(axisParentSize, resolveCalcValue).Unwrap()
		}
		return some(limit)
	}
	return none[float32]()
}

func (g *gridItem) knownDimensionsCached(tree LayoutPartialTree, gridAreaSize Size[Option[float32]]) Size[Option[float32]] {
	if entry, ok := g.knownDimensionsCache.Get(); ok {
		if entry.gridAreaSize == gridAreaSize {
			return entry.knownDimensions
		}
	}
	knownDimensions := g.knownDimensions(tree, gridAreaSize)
	g.knownDimensionsCache = some(knownDimensionsCacheEntry{gridAreaSize: gridAreaSize, knownDimensions: knownDimensions})
	return knownDimensions
}

func (g *gridItem) isStretchAligned(alignment AlignSelf) bool {
	switch alignment.keyword {
	case AlignItemsKeywordStretch:
		return true
	case AlignItemsKeywordNormal:
		return !g.isReplaced
	default:
		return false
	}
}

func (g *gridItem) knownDimensions(tree LayoutPartialTree, gridAreaSize Size[Option[float32]]) Size[Option[float32]] {
	calcFn := treeCalc(tree)
	margins := g.marginsAxisSumsWithBaselineShims(gridAreaSize.Width, tree)

	aspectRatio := g.aspectRatio
	padding := RectResolveOrZeroOptionF32(g.padding, gridAreaSize.Width, calcFn)
	border := RectResolveOrZeroOptionF32(g.border, gridAreaSize.Width, calcFn)
	paddingBorderSize := padding.Add(border).SumAxes()
	boxSizingAdjustment := SizeF32Zero
	if g.boxSizing == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}
	inherentSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve[Dimension, Option[float32], Option[float32]](g.size, gridAreaSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	minSize := SizeOptionF32MaybeMaxF32(
		SizeOptionOr(
			SizeOptionF32MaybeAddF32(
				SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve[LengthPercentageAuto, Option[float32], Option[float32]](g.minSize, gridAreaSize, calcFn), aspectRatio),
				boxSizingAdjustment,
			),
			Size[Option[float32]]{Width: some(paddingBorderSize.Width), Height: some(paddingBorderSize.Height)},
		),
		paddingBorderSize,
	)
	maxSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve[LengthPercentageAuto, Option[float32], Option[float32]](g.maxSize, gridAreaSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)

	gridAreaMinusItemMarginsSize := SizeOptionF32MaybeMaxF32(SizeOptionF32MaybeSubF32(gridAreaSize, margins), SizeF32Zero)

	width := inherentSize.Width
	if width.IsNone() {
		width = func() Option[float32] {
			if g.size.Width.IsSizingKeyword() {
				if res, ok := resolveSizingKeyword(g.size.Width, gridAreaMinusItemMarginsSize.Width, gridAreaSize.Width, calcFn).Get(); ok && res.kind == sizingKeywordResolutionKindExact {
					return some(res.exact)
				}
				return none[float32]()
			}
			if !g.margin.Left.IsAuto() && !g.margin.Right.IsAuto() && g.isStretchAligned(g.justifySelf) {
				return gridAreaMinusItemMarginsSize.Width
			}
			return none[float32]()
		}()
	}
	s := SizeOptionF32MaybeApplyAspectRatio(Size[Option[float32]]{Width: width, Height: inherentSize.Height}, aspectRatio)
	width, height := s.Width, s.Height

	if height.IsNone() {
		height = func() Option[float32] {
			if g.size.Height.IsSizingKeyword() {
				if res, ok := resolveSizingKeyword(g.size.Height, gridAreaMinusItemMarginsSize.Height, gridAreaSize.Height, calcFn).Get(); ok && res.kind == sizingKeywordResolutionKindExact {
					return some(res.exact)
				}
				return none[float32]()
			}
			if !g.margin.Top.IsAuto() && !g.margin.Bottom.IsAuto() && g.isStretchAligned(g.alignSelf) {
				return gridAreaMinusItemMarginsSize.Height
			}
			return none[float32]()
		}()
	}
	s = SizeOptionF32MaybeApplyAspectRatio(Size[Option[float32]]{Width: width, Height: height}, aspectRatio)

	return SizeOptionF32MaybeClamp(s, minSize, maxSize)
}

func sumOptionF32(values []Option[float32]) Option[float32] {
	var total float32
	for _, v := range values {
		f, ok := v.Get()
		if !ok {
			return none[float32]()
		}
		total += f
	}
	return some(total)
}

func (g *gridItem) gridAreaSize(
	axis AbstractAxis,
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	availableSpace Size[Option[float32]],
	getTrackSizeEstimate func(*gridTrack, Option[float32]) Option[float32],
	resolveCalcValue func(unsafe.Pointer, float32) float32,
) Size[Option[float32]] {
	size := SizeOptionF32None

	start, end := g.trackRangeExcludingLines(axis)
	axisSum := func() Option[float32] {
		var total float32
		for i := start; i < end; i++ {
			track := &axisTracks[i]
			minSize, ok := track.minTrackSizingFunction.DefiniteValue(availableSpace.get(axis), resolveCalcValue).Get()
			if !ok {
				return none[float32]()
			}
			maxSize, ok := track.maxTrackSizingFunction.DefiniteValue(availableSpace.get(axis), resolveCalcValue).Get()
			if !ok {
				return none[float32]()
			}
			if minSize != maxSize {
				return none[float32]()
			}
			total += track.baseSize
		}
		return some(total)
	}()
	size.set(axis, axisSum)

	otherStart, otherEnd := g.trackRangeExcludingLines(axis.Other())
	otherSum := func() Option[float32] {
		var total float32
		for i := otherStart; i < otherEnd; i++ {
			track := &otherAxisTracks[i]
			estimate, ok := getTrackSizeEstimate(track, availableSpace.get(axis.Other())).Get()
			if !ok {
				return none[float32]()
			}
			total += estimate + track.contentAlignmentAdjustment
		}
		return some(total)
	}()
	size.set(axis.Other(), otherSum)

	return size
}

func (g *gridItem) gridAreaSizeCached(
	axis AbstractAxis,
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	availableSpace Size[Option[float32]],
	getTrackSizeEstimate func(*gridTrack, Option[float32]) Option[float32],
	resolveCalcValue func(unsafe.Pointer, float32) float32,
) Size[Option[float32]] {
	if cached, ok := g.gridAreaSizeCache.Get(); ok {
		return cached
	}
	gridAreaSize := g.gridAreaSize(axis, axisTracks, otherAxisTracks, availableSpace, getTrackSizeEstimate, resolveCalcValue)
	g.gridAreaSizeCache = some(gridAreaSize)
	return gridAreaSize
}

func (g *gridItem) marginsAxisSumsWithBaselineShims(innerNodeWidth Option[float32], tree LayoutPartialTree) Size[float32] {
	calcFn := treeCalc(tree)
	return Rect[float32]{
		Left:   g.margin.Left.ResolveOrZero(some[float32](0.0), calcFn),
		Right:  g.margin.Right.ResolveOrZero(some[float32](0.0), calcFn),
		Top:    g.margin.Top.ResolveOrZero(innerNodeWidth, calcFn) + g.baselineShim,
		Bottom: g.margin.Bottom.ResolveOrZero(innerNodeWidth, calcFn),
	}.SumAxes()
}

func (g *gridItem) minContentContribution(axis AbstractAxis, tree LayoutPartialTree, gridAreaSize Size[Option[float32]], availableSpace Size[Option[float32]]) float32 {
	knownDimensions := g.knownDimensionsCached(tree, gridAreaSize)
	if size, ok := knownDimensions.get(axis).Get(); ok {
		return size
	}
	return measureChildSize(
		tree,
		g.node,
		knownDimensions,
		gridAreaSize,
		g.keywordAdjustedAvailableSpace(
			gridAreaSize,
			Size[Option[float32]].Map(g.availableSpaceMinusMargins(gridAreaSize, availableSpace, tree), func(opt Option[float32]) AvailableSpace {
				if size, ok := opt.Get(); ok {
					return AvailableSpaceDefinite(size)
				}
				return AvailableSpaceMinContent
			}),
			tree,
		),
		SizingModeInherentSize,
		axis.AsAbsNaive(),
		LineBoolFalse,
	)
}

func (g *gridItem) minContentContributionCached(axis AbstractAxis, tree LayoutPartialTree, gridAreaSize Size[Option[float32]], availableSpace Size[Option[float32]]) float32 {
	if size, ok := g.minContentContributionCache.get(axis).Get(); ok {
		return size
	}
	size := g.minContentContribution(axis, tree, gridAreaSize, availableSpace)
	g.minContentContributionCache.set(axis, some(size))
	return size
}

func (g *gridItem) maxContentContribution(axis AbstractAxis, tree LayoutPartialTree, gridAreaSize Size[Option[float32]], availableSpace Size[Option[float32]]) float32 {
	knownDimensions := g.knownDimensionsCached(tree, gridAreaSize)
	if size, ok := knownDimensions.get(axis).Get(); ok {
		return size
	}
	return measureChildSize(
		tree,
		g.node,
		knownDimensions,
		gridAreaSize,
		g.keywordAdjustedAvailableSpace(
			gridAreaSize,
			Size[Option[float32]].Map(g.availableSpaceMinusMargins(gridAreaSize, availableSpace, tree), func(opt Option[float32]) AvailableSpace {
				if size, ok := opt.Get(); ok {
					return AvailableSpaceDefinite(size)
				}
				return AvailableSpaceMaxContent
			}),
			tree,
		),
		SizingModeInherentSize,
		axis.AsAbsNaive(),
		LineBoolFalse,
	)
}

func (g *gridItem) availableSpaceMinusMargins(gridAreaSize Size[Option[float32]], availableSpace Size[Option[float32]], tree LayoutPartialTree) Size[Option[float32]] {
	if availableSpace.Width.IsNone() && availableSpace.Height.IsNone() {
		return availableSpace
	}
	margin := RectResolveOrZeroOptionF32(g.margin, gridAreaSize.Width, treeCalc(tree))
	out := Size[Option[float32]]{Width: none[float32](), Height: none[float32]()}
	if width, ok := availableSpace.Width.Get(); ok {
		out.Width = some(f32Max(width-margin.HorizontalAxisSum(), 0.0))
	}
	if height, ok := availableSpace.Height.Get(); ok {
		out.Height = some(f32Max(height-margin.VerticalAxisSum()-g.baselineShim, 0.0))
	}
	return out
}

func (g *gridItem) keywordAdjustedAvailableSpace(gridAreaSize Size[Option[float32]], availableSpace Size[AvailableSpace], tree LayoutPartialTree) Size[AvailableSpace] {
	if !g.size.Width.IsSizingKeyword() && !g.size.Height.IsSizingKeyword() {
		return availableSpace
	}
	calcFn := treeCalc(tree)
	margins := g.marginsAxisSumsWithBaselineShims(gridAreaSize.Width, tree)
	adjusted := availableSpace
	for _, axis := range [...]AbstractAxis{AbstractAxisInline, AbstractAxisBlock} {
		sizeStyle := g.size.get(axis)
		if !sizeStyle.IsSizingKeyword() {
			continue
		}
		stretchSize := OptionF32MaybeMaxF32(OptionF32MaybeSubF32(gridAreaSize.get(axis), margins.get(axis)), 0.0)
		if res, ok := resolveSizingKeyword(sizeStyle, stretchSize, gridAreaSize.get(axis), calcFn).Get(); ok && res.kind == sizingKeywordResolutionKindMeasure {
			adjusted.set(axis, res.measure)
		}
	}
	return adjusted
}

func (g *gridItem) maxContentContributionCached(axis AbstractAxis, tree LayoutPartialTree, gridAreaSize Size[Option[float32]], availableSpace Size[Option[float32]]) float32 {
	if size, ok := g.maxContentContributionCache.get(axis).Get(); ok {
		return size
	}
	size := g.maxContentContribution(axis, tree, gridAreaSize, availableSpace)
	g.maxContentContributionCache.set(axis, some(size))
	return size
}

func (g *gridItem) minimumContribution(tree LayoutPartialTree, axis AbstractAxis, axisTracks []gridTrack, gridAreaSize Size[Option[float32]], innerNodeSize Size[Option[float32]]) float32 {
	calcFn := treeCalc(tree)
	padding := RectResolveOrZeroOptionF32(g.padding, gridAreaSize.Width, calcFn)
	border := RectResolveOrZeroOptionF32(g.border, gridAreaSize.Width, calcFn)
	paddingBorderSize := padding.Add(border).SumAxes()
	boxSizingAdjustment := SizeF32Zero
	if g.boxSizing == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	resolvedSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve[Dimension, Option[float32], Option[float32]](g.size, gridAreaSize, calcFn), g.aspectRatio),
		boxSizingAdjustment,
	).get(axis)
	if size, ok := resolvedSize.Get(); ok {
		minSize := g.minSize.get(axis)
		maxSize := g.maxSize.get(axis)
		if minSize.IsAuto() && maxSize.IsAuto() {
			return size
		}
		basis := gridAreaSize.get(axis)
		maxSizeBasis := basis
		if g.isCompressibleReplaced {
			maxSizeBasis = basis.Or(some[float32](0.0))
		}
		adjustment := boxSizingAdjustment.get(axis)
		resolvedMin := minSize.MaybeResolve(basis, calcFn)
		resolvedMax := maxSize.MaybeResolve(maxSizeBasis, calcFn)
		return F32MaybeClamp(size, OptionF32MaybeAddF32(resolvedMin, adjustment), OptionF32MaybeAddF32(resolvedMax, adjustment))
	}

	if size := g.size.get(axis); size.IsSizingKeyword() && !size.IsStretch() {
		return g.minContentContributionCached(axis, tree, gridAreaSize, gridAreaSize)
	}

	zeroedGridAreaSize := Size[float32]{Width: gridAreaSize.Width.UnwrapOr(0.0), Height: gridAreaSize.Height.UnwrapOr(0.0)}
	if size, ok := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolveF32(g.minSize, zeroedGridAreaSize, calcFn), g.aspectRatio),
		boxSizingAdjustment,
	).get(axis).Get(); ok {
		return size
	}

	if size, ok := g.overflow.Get(axis).maybeIntoAutomaticMinSize().Get(); ok {
		return size
	}

	start, end := g.trackRangeExcludingLines(axis)
	itemAxisTracks := axisTracks[start:end]

	spansAutoMinTrack := false
	for i := range itemAxisTracks {
		if itemAxisTracks[i].minTrackSizingFunction.BehavesAsAuto(innerNodeSize.get(axis)) {
			spansAutoMinTrack = true
			break
		}
	}

	onlySpanOneTrack := len(itemAxisTracks) == 1
	anyFlexible := false
	for i := range itemAxisTracks {
		if itemAxisTracks[i].maxTrackSizingFunction.IsFr() {
			anyFlexible = true
			break
		}
	}
	useContentBasedMinimum := spansAutoMinTrack && (onlySpanOneTrack || !anyFlexible)

	if useContentBasedMinimum {
		minimumContribution := g.minContentContributionCached(axis, tree, gridAreaSize, gridAreaSize)

		if g.isCompressibleReplaced {
			size := g.size.get(axis).MaybeResolve(some[float32](0.0), calcFn)
			maxSize := g.maxSize.get(axis).MaybeResolve(some[float32](0.0), calcFn)
			minimumContribution = F32MaybeMin(F32MaybeMin(minimumContribution, size), maxSize)
		}

		limit := g.spannedFixedTrackLimit(axis, axisTracks, innerNodeSize.get(axis), func(val unsafe.Pointer, basis float32) float32 {
			return tree.ResolveCalcValue(val, basis)
		})
		clampedLimit := none[float32]()
		if l, ok := limit.Get(); ok {
			clampedLimit = some(f32Max(l, paddingBorderSize.get(axis)))
		}
		return F32MaybeMin(minimumContribution, clampedLimit)
	}
	return 0.0
}

func (g *gridItem) minimumContributionCached(tree LayoutPartialTree, axis AbstractAxis, axisTracks []gridTrack, gridAreaSize Size[Option[float32]], innerNodeSize Size[Option[float32]]) float32 {
	if size, ok := g.minimumContributionCache.get(axis).Get(); ok {
		return size
	}
	size := g.minimumContribution(tree, axis, axisTracks, gridAreaSize, innerNodeSize)
	g.minimumContributionCache.set(axis, some(size))
	return size
}
