package taffy

import (
	"math"
	"slices"
	"unsafe"
)

type itemBatcher struct {
	axis          AbstractAxis
	indexOffset   int
	currentSpan   uint16
	currentIsFlex bool
}

func newItemBatcher(axis AbstractAxis) itemBatcher {
	return itemBatcher{indexOffset: 0, axis: axis, currentSpan: 1, currentIsFlex: false}
}

func (b *itemBatcher) next(items []*gridItem) ([]*gridItem, bool, bool) {
	if b.currentIsFlex || b.indexOffset >= len(items) {
		return nil, false, false
	}

	item := items[b.indexOffset]
	b.currentSpan = item.span(b.axis)
	b.currentIsFlex = item.crossesFlexibleTrack(b.axis)

	var nextIndexOffset int
	if b.currentIsFlex {
		nextIndexOffset = len(items)
	} else {
		nextIndexOffset = len(items)
		for i, item := range items {
			if item.crossesFlexibleTrack(b.axis) || item.span(b.axis) > b.currentSpan {
				nextIndexOffset = i
				break
			}
		}
	}

	batch := items[b.indexOffset:nextIndexOffset]
	b.indexOffset = nextIndexOffset
	return batch, b.currentIsFlex, true
}

type trackSizeEstimateFunc func(track *gridTrack, parentSize Option[float32], tree LayoutPartialTree) Option[float32]

type intrinsicSizeMeasurer struct {
	tree                 LayoutPartialTree
	otherAxisTracks      []gridTrack
	getTrackSizeEstimate trackSizeEstimateFunc
	axis                 AbstractAxis
	innerNodeSize        Size[Option[float32]]
}

func (m *intrinsicSizeMeasurer) gridAreaSize(item *gridItem, axisTracks []gridTrack) Size[Option[float32]] {
	return item.gridAreaSizeCached(
		m.axis,
		axisTracks,
		m.otherAxisTracks,
		m.innerNodeSize,
		func(track *gridTrack, basis Option[float32]) Option[float32] {
			return m.getTrackSizeEstimate(track, basis, m.tree)
		},
		treeCalc(m.tree),
	)
}

func (m *intrinsicSizeMeasurer) marginsAxisSumsWithBaselineShims(item *gridItem, percentageBasis Option[float32]) Size[float32] {
	return item.marginsAxisSumsWithBaselineShims(percentageBasis, m.tree)
}

func (m *intrinsicSizeMeasurer) calc(val unsafe.Pointer, basis float32) float32 {
	return calc(m.tree, val, basis)
}

func (m *intrinsicSizeMeasurer) minContentContribution(item *gridItem, axisTracks []gridTrack) float32 {
	gridAreaSize := m.gridAreaSize(item, axisTracks)
	availableSpace := gridAreaSize.with(m.axis, none[float32]())
	marginAxisSums := m.marginsAxisSumsWithBaselineShims(item, availableSpace.Width)
	contribution := item.minContentContributionCached(m.axis, m.tree, gridAreaSize, availableSpace)
	return contribution + marginAxisSums.get(m.axis)
}

func (m *intrinsicSizeMeasurer) maxContentContribution(item *gridItem, axisTracks []gridTrack) float32 {
	gridAreaSize := m.gridAreaSize(item, axisTracks)
	availableSpace := gridAreaSize.with(m.axis, none[float32]())
	marginAxisSums := m.marginsAxisSumsWithBaselineShims(item, availableSpace.Width)
	contribution := item.maxContentContributionCached(m.axis, m.tree, gridAreaSize, availableSpace)
	return contribution + marginAxisSums.get(m.axis)
}

func (m *intrinsicSizeMeasurer) minimumContribution(item *gridItem, axisTracks []gridTrack) float32 {
	gridAreaSize := m.gridAreaSize(item, axisTracks)
	availableSpace := gridAreaSize.with(m.axis, none[float32]())
	marginAxisSums := m.marginsAxisSumsWithBaselineShims(item, availableSpace.Width)
	contribution := item.minimumContributionCached(m.tree, m.axis, axisTracks, gridAreaSize, m.innerNodeSize)
	return contribution + marginAxisSums.get(m.axis)
}

func computeAlignmentGutterAdjustment(
	alignment AlignContent,
	axisInnerNodeSize Option[float32],
	getTrackSizeEstimate func(*gridTrack, Option[float32]) Option[float32],
	tracks []gridTrack,
) float32 {
	if len(tracks) <= 1 {
		return 0.0
	}

	var outerGutterWeight int
	switch alignment.Keyword() {
	case AlignContentKeywordStart, AlignContentKeywordFlexStart, AlignContentKeywordEnd, AlignContentKeywordFlexEnd, AlignContentKeywordCenter:
		outerGutterWeight = 1
	case AlignContentKeywordNormal, AlignContentKeywordStretch:
		outerGutterWeight = 0
	case AlignContentKeywordSpaceBetween:
		outerGutterWeight = 0
	case AlignContentKeywordSpaceAround:
		outerGutterWeight = 1
	case AlignContentKeywordSpaceEvenly:
		outerGutterWeight = 1
	}

	var innerGutterWeight int
	switch alignment.Keyword() {
	case AlignContentKeywordFlexStart, AlignContentKeywordStart, AlignContentKeywordFlexEnd, AlignContentKeywordEnd,
		AlignContentKeywordCenter, AlignContentKeywordNormal, AlignContentKeywordStretch:
		innerGutterWeight = 0
	case AlignContentKeywordSpaceBetween:
		innerGutterWeight = 1
	case AlignContentKeywordSpaceAround:
		innerGutterWeight = 2
	case AlignContentKeywordSpaceEvenly:
		innerGutterWeight = 1
	}

	if innerGutterWeight == 0 {
		return 0.0
	}

	if size, ok := axisInnerNodeSize.Get(); ok {
		var freeSpace float32
		var trackSizeSum float32
		allSome := true
		for i := range tracks {
			v, ok := getTrackSizeEstimate(&tracks[i], some(size)).Get()
			if !ok {
				allSome = false
				break
			}
			trackSizeSum += v
		}
		if allSome {
			freeSpace = f32Max(0.0, size-trackSizeSum)
		}

		weightedTrackCount := ((len(tracks)-3)/2)*innerGutterWeight + 2*outerGutterWeight

		return float32((freeSpace / float32(weightedTrackCount)) * float32(innerGutterWeight))
	}

	return 0.0
}

func resolveItemTrackIndexes(items []gridItem, columnCounts trackCounts, rowCounts trackCounts) {
	for i := range items {
		item := &items[i]
		item.columnIndexes = Line[uint16]{
			Start: uint16(item.column.Start.intoTrackVecIndex(columnCounts)),
			End:   uint16(item.column.End.intoTrackVecIndex(columnCounts)),
		}
		item.rowIndexes = Line[uint16]{
			Start: uint16(item.row.Start.intoTrackVecIndex(rowCounts)),
			End:   uint16(item.row.End.intoTrackVecIndex(rowCounts)),
		}
	}
}

func anyTrackInRange(tracks []gridTrack, start, end int, predicate func(*gridTrack) bool) bool {
	for i := start; i < end; i++ {
		if predicate(&tracks[i]) {
			return true
		}
	}
	return false
}

func determineIfItemCrossesFlexibleOrIntrinsicTracks(items []gridItem, columns []gridTrack, rows []gridTrack) {
	for i := range items {
		item := &items[i]
		cs, ce := item.trackRangeExcludingLines(AbstractAxisInline)
		rs, re := item.trackRangeExcludingLines(AbstractAxisBlock)
		item.crossesFlexibleColumn = anyTrackInRange(columns, cs, ce, (*gridTrack).isFlexible)
		item.crossesIntrinsicColumn = anyTrackInRange(columns, cs, ce, (*gridTrack).hasIntrinsicSizingFunction)
		item.crossesFlexibleRow = anyTrackInRange(rows, rs, re, (*gridTrack).isFlexible)
		item.crossesIntrinsicRow = anyTrackInRange(rows, rs, re, (*gridTrack).hasIntrinsicSizingFunction)
	}
}

func trackSizingAlgorithm(
	tree LayoutPartialTree,
	axis AbstractAxis,
	axisMinSize Option[float32],
	axisMaxSize Option[float32],
	axisAlignment AlignContent,
	otherAxisAlignment AlignContent,
	availableGridSpace Size[AvailableSpace],
	innerNodeSize Size[Option[float32]],
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	items []gridItem,
	getTrackSizeEstimate trackSizeEstimateFunc,
	hasBaselineAlignedItem bool,
) {
	calcFn := treeCalc(tree)
	percentageBasis := innerNodeSize.get(axis).Or(axisMinSize)
	initializeTrackSizes(tree, axisTracks, percentageBasis)

	axisInnerNodeSize := innerNodeSize.get(axis)
	for i := range axisTracks {
		axisTracks[i].resolveFitContentLimit(axisInnerNodeSize, calcFn)
	}

	if hasBaselineAlignedItem {
		resolveItemBaselines(tree, axis, items, innerNodeSize)
	}

	allFinal := true
	for i := range axisTracks {
		track := &axisTracks[i]
		if !(track.baseSize == track.growthLimit && track.minTrackSizingFunction.DefiniteValue(percentageBasis, calcFn).IsSome()) {
			allFinal = false
			break
		}
	}
	if allFinal {
		return
	}

	gutterAlignmentAdjustment := computeAlignmentGutterAdjustment(
		otherAxisAlignment,
		innerNodeSize.get(axis.Other()),
		func(track *gridTrack, basis Option[float32]) Option[float32] {
			return getTrackSizeEstimate(track, basis, tree)
		},
		otherAxisTracks,
	)
	if len(otherAxisTracks) > 3 {
		for i := 2; i < len(otherAxisTracks); i += 2 {
			otherAxisTracks[i].contentAlignmentAdjustment = gutterAlignmentAdjustment
		}
	}

	resolveIntrinsicTrackSizes(
		tree,
		axis,
		axisTracks,
		otherAxisTracks,
		items,
		availableGridSpace.get(axis),
		innerNodeSize,
		getTrackSizeEstimate,
	)

	maximiseTracks(axisTracks, availableGridSpace.get(axis))

	var axisAvailableSpaceForExpansion AvailableSpace
	if availableSpace, ok := innerNodeSize.get(axis).Get(); ok {
		axisAvailableSpaceForExpansion = AvailableSpaceDefinite(availableSpace)
	} else {
		space := availableGridSpace.get(axis)
		if space.kind == AvailableSpaceKindMinContent && axis == AbstractAxisInline {
			axisAvailableSpaceForExpansion = AvailableSpaceMinContent
		} else {
			axisAvailableSpaceForExpansion = AvailableSpaceMaxContent
		}
	}

	expandFlexibleTracks(
		tree,
		axis,
		axisTracks,
		otherAxisTracks,
		items,
		axisMinSize,
		axisMaxSize,
		axisAvailableSpaceForExpansion,
		innerNodeSize,
		getTrackSizeEstimate,
	)

	if axisAlignment == AlignContentStretch {
		stretchAutoTracks(axisTracks, axisMinSize, axisAvailableSpaceForExpansion)
	}
}

type intrinsicContributionType uint8

const (
	intrinsicContributionTypeMinimum intrinsicContributionType = iota
	intrinsicContributionTypeMaximum
)

func flushPlannedBaseSizeIncreases(tracks []gridTrack) {
	for i := range tracks {
		track := &tracks[i]
		track.baseSize += track.baseSizePlannedIncrease
		track.baseSizePlannedIncrease = 0.0
	}
}

func flushPlannedGrowthLimitIncreases(tracks []gridTrack, setInfinitelyGrowable bool) {
	inf := float32(math.Inf(1))
	for i := range tracks {
		track := &tracks[i]
		if track.growthLimitPlannedIncrease > 0.0 {
			if track.growthLimit == inf {
				track.growthLimit = track.baseSize + track.growthLimitPlannedIncrease
			} else {
				track.growthLimit = track.growthLimit + track.growthLimitPlannedIncrease
			}
			track.infinitelyGrowable = setInfinitelyGrowable
		} else {
			track.infinitelyGrowable = false
		}
		track.growthLimitPlannedIncrease = 0.0
	}
}

func initializeTrackSizes(tree LayoutPartialTree, axisTracks []gridTrack, axisInnerNodeSize Option[float32]) {
	calcFn := treeCalc(tree)
	for i := range axisTracks {
		track := &axisTracks[i]
		track.baseSize = track.minTrackSizingFunction.DefiniteValue(axisInnerNodeSize, calcFn).UnwrapOr(0.0)
		track.growthLimit = track.maxTrackSizingFunction.DefiniteValue(axisInnerNodeSize, calcFn).UnwrapOr(float32(math.Inf(1)))
		if track.growthLimit < track.baseSize {
			track.growthLimit = track.baseSize
		}
	}
}

func resolveItemBaselines(tree LayoutPartialTree, axis AbstractAxis, allItems []gridItem, innerNodeSize Size[Option[float32]]) {
	otherAxis := axis.Other()
	items := make([]*gridItem, len(allItems))
	for i := range allItems {
		items[i] = &allItems[i]
	}
	slices.SortStableFunc(items, func(a, b *gridItem) int {
		return int(a.placement(otherAxis).Start) - int(b.placement(otherAxis).Start)
	})

	remainingItems := items
	for len(remainingItems) > 0 {
		currentRow := remainingItems[0].placement(otherAxis).Start

		nextRowFirstItem := -1
		for i, item := range remainingItems {
			if item.placement(otherAxis).Start != currentRow {
				nextRowFirstItem = i
				break
			}
		}

		var rowItems []*gridItem
		if nextRowFirstItem >= 0 {
			rowItems = remainingItems[:nextRowFirstItem]
			remainingItems = remainingItems[nextRowFirstItem:]
		} else {
			rowItems = remainingItems
			remainingItems = nil
		}

		rowBaselineItemCount := 0
		for _, item := range rowItems {
			if item.participatesInBaselineAlignment() {
				rowBaselineItemCount++
			}
		}
		if rowBaselineItemCount <= 1 {
			continue
		}

		for _, item := range rowItems {
			if !item.participatesInBaselineAlignment() {
				continue
			}

			measuredSizeAndBaselines := performChildLayout(
				tree,
				item.node,
				SizeOptionF32None,
				innerNodeSize,
				sizeAvailableSpaceMinContent,
				SizingModeInherentSize,
				LineBoolFalse,
			)

			baselineOpt := measuredSizeAndBaselines.Baselines.First
			height := measuredSizeAndBaselines.Size.Height

			var baseline float32
			if item.overflow.Y.IsScrollContainer() {
				baseline = f32Max(f32Min(baselineOpt.UnwrapOr(height), height), 0.0)
			} else {
				baseline = baselineOpt.UnwrapOr(height)
			}

			item.baseline = some(baseline + item.margin.Top.ResolveOrZero(innerNodeSize.Width, treeCalc(tree)))
		}

		first := true
		var rowMaxBaseline float32
		for _, item := range rowItems {
			if !item.participatesInBaselineAlignment() {
				continue
			}
			b := item.baseline.UnwrapOr(0.0)
			if first || f32TotalCmp(b, rowMaxBaseline) >= 0 {
				rowMaxBaseline = b
				first = false
			}
		}
		if first {
			panic("called `Option::unwrap()` on a `None` value")
		}

		for _, item := range rowItems {
			if item.participatesInBaselineAlignment() {
				item.baselineShim = rowMaxBaseline - item.baseline.UnwrapOr(0.0)
				item.knownDimensionsCache = none[knownDimensionsCacheEntry]()
			}
		}
	}
}

func resolveIntrinsicTrackSizes(
	tree LayoutPartialTree,
	axis AbstractAxis,
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	allItems []gridItem,
	axisAvailableGridSpace AvailableSpace,
	innerNodeSize Size[Option[float32]],
	getTrackSizeEstimate trackSizeEstimateFunc,
) {
	partitionedItems := frontBackVecBuilderWithCapacity[*gridItem](len(allItems))
	needsSort := false
	for i := range allItems {
		item := &allItems[i]
		if item.crossesFlexibleTrack(axis) {
			partitionedItems.pushBack(item)
		} else {
			needsSort = needsSort || item.span(axis) > 1
			partitionedItems.pushFront(item)
		}
	}
	items, nonFlexItemCount := partitionedItems.intoVec()
	if needsSort {
		slices.SortStableFunc(items[:nonFlexItemCount], func(a, b *gridItem) int {
			return int(a.span(axis)) - int(b.span(axis))
		})
	}

	axisInnerNodeSize := innerNodeSize.get(axis)
	itemSizer := &intrinsicSizeMeasurer{tree: tree, otherAxisTracks: otherAxisTracks, axis: axis, innerNodeSize: innerNodeSize, getTrackSizeEstimate: getTrackSizeEstimate}
	sizerCalc := itemSizer.calc
	inf := float32(math.Inf(1))
	isMinOrMaxContentKind := axisAvailableGridSpace.kind == AvailableSpaceKindMinContent || axisAvailableGridSpace.kind == AvailableSpaceKindMaxContent

	batcher := newItemBatcher(axis)
	for {
		batch, isFlex, ok := batcher.next(items)
		if !ok {
			break
		}
		batchSpan := lineOriginZeroLineSpan(batch[0].placement(axis))
		if !isFlex && batchSpan == 1 {
			for _, item := range batch {
				trackIndex := int(item.placementIndexes(axis).Start) + 1
				track := &axisTracks[trackIndex]

				var newBaseSize float32
				switch {
				case track.minTrackSizingFunction.inner.Tag() == CompactLengthMinContentTag:
					newBaseSize = f32Max(track.baseSize, itemSizer.minContentContribution(item, axisTracks))
				case track.minTrackSizingFunction.inner.Tag() == CompactLengthMaxContentTag:
					newBaseSize = f32Max(track.baseSize, itemSizer.maxContentContribution(item, axisTracks))
				case track.minTrackSizingFunction.BehavesAsAuto(axisInnerNodeSize):
					var space float32
					if isMinOrMaxContentKind && !item.overflow.Get(axis).IsScrollContainer() {
						axisMinimumSize := itemSizer.minimumContribution(item, axisTracks)
						axisMinContentSize := itemSizer.minContentContribution(item, axisTracks)
						limit := track.maxTrackSizingFunction.DefiniteLimit(axisInnerNodeSize, sizerCalc)
						space = f32Max(F32MaybeMin(axisMinContentSize, limit), axisMinimumSize)
					} else {
						space = itemSizer.minimumContribution(item, axisTracks)
					}
					newBaseSize = f32Max(track.baseSize, space)
				default:
					newBaseSize = track.baseSize
				}
				axisTracks[trackIndex].baseSize = newBaseSize

				track = &axisTracks[trackIndex]
				maxTrackSizingFunction := track.maxTrackSizingFunction
				fitContentLimit := track.getFitContentLimit()
				minContentContribution := none[float32]()
				if track.hasIntrinsicSizingFunction() {
					minContentContribution = some(itemSizer.minContentContribution(item, axisTracks))
				}
				var plannedIncrease Option[float32]
				switch {
				case maxTrackSizingFunction.IsFitContent():
					maxContentContribution := f32Min(itemSizer.maxContentContribution(item, axisTracks), fitContentLimit)
					if mcc, ok := minContentContribution.Get(); ok && !item.overflow.Get(axis).IsScrollContainer() {
						plannedIncrease = some(f32Max(mcc, maxContentContribution))
					} else {
						plannedIncrease = some(maxContentContribution)
					}
				case maxTrackSizingFunction.IsMaxContentAlike() || maxTrackSizingFunction.UsesPercentage() && axisInnerNodeSize.IsNone():
					plannedIncrease = some(itemSizer.maxContentContribution(item, axisTracks))
				case maxTrackSizingFunction.IsIntrinsic():
					plannedIncrease = minContentContribution
				default:
					plannedIncrease = none[float32]()
				}
				if pi, ok := plannedIncrease.Get(); ok {
					track := &axisTracks[trackIndex]
					track.growthLimitPlannedIncrease = f32Max(track.growthLimitPlannedIncrease, pi)
				}
			}

			for i := range axisTracks {
				track := &axisTracks[i]
				if track.growthLimitPlannedIncrease > 0.0 {
					if track.growthLimit == inf {
						track.growthLimit = track.growthLimitPlannedIncrease
					} else {
						track.growthLimit = f32Max(track.growthLimit, track.growthLimitPlannedIncrease)
					}
				}
				track.infinitelyGrowable = false
				track.growthLimitPlannedIncrease = 0.0
				if track.growthLimit < track.baseSize {
					track.growthLimit = track.baseSize
				}
			}

			continue
		}

		growthLimit := func(track *gridTrack) float32 { return track.growthLimit }
		fitContentLimited := (*gridTrack).fitContentLimitedGrowthLimit

		for _, item := range batch {
			if !item.crossesIntrinsicTrack(axis) {
				continue
			}
			var space float32
			if isMinOrMaxContentKind && !item.overflow.Get(axis).IsScrollContainer() {
				axisMinimumSize := itemSizer.minimumContribution(item, axisTracks)
				axisMinContentSize := itemSizer.minContentContribution(item, axisTracks)
				limit := item.spannedTrackLimit(axis, axisTracks, axisInnerNodeSize, sizerCalc)
				limitedMinContent := f32Max(F32MaybeMin(axisMinContentSize, limit), axisMinimumSize)

				if isFlex {
					s, e := item.trackRangeExcludingLines(axis)
					spannedTracks := axisTracks[s:e]
					var inflexibleSizes float32
					for i := range spannedTracks {
						if !spannedTracks[i].isFlexible() {
							inflexibleSizes += spannedTracks[i].baseSize
						}
					}
					scale := f32Min(crossedFlexFactorSum(spannedTracks), 1.0)
					excess := f32Max(limitedMinContent-inflexibleSizes, 0.0)
					space = f32Max(axisMinimumSize, inflexibleSizes+float32(excess*scale))
				} else {
					space = limitedMinContent
				}
			} else {
				space = itemSizer.minimumContribution(item, axisTracks)
			}
			s, e := item.trackRangeExcludingLines(axis)
			tracks := axisTracks[s:e]
			if space > 0.0 {
				hasIntrinsicMinTrackSizingFunction := func(track *gridTrack) bool {
					return track.minTrackSizingFunction.DefiniteValue(axisInnerNodeSize, sizerCalc).IsNone()
				}
				if item.overflow.Get(axis).IsScrollContainer() {
					distributeItemSpaceToBaseSize(isFlex, space, tracks, hasIntrinsicMinTrackSizingFunction, fitContentLimited, intrinsicContributionTypeMinimum)
				} else {
					distributeItemSpaceToBaseSize(isFlex, space, tracks, hasIntrinsicMinTrackSizingFunction, growthLimit, intrinsicContributionTypeMinimum)
				}
			}
		}
		flushPlannedBaseSizeIncreases(axisTracks)

		hasMinOrMaxContentMinTrackSizingFunction := func(track *gridTrack) bool {
			return track.minTrackSizingFunction.IsMinOrMaxContent()
		}
		for _, item := range batch {
			if !item.spansTrackMatching(axis, axisTracks, hasMinOrMaxContentMinTrackSizingFunction) {
				continue
			}
			space := itemSizer.minContentContribution(item, axisTracks)
			s, e := item.trackRangeExcludingLines(axis)
			tracks := axisTracks[s:e]
			if space > 0.0 {
				if item.overflow.Get(axis).IsScrollContainer() {
					distributeItemSpaceToBaseSize(isFlex, space, tracks, hasMinOrMaxContentMinTrackSizingFunction, fitContentLimited, intrinsicContributionTypeMinimum)
				} else {
					distributeItemSpaceToBaseSize(isFlex, space, tracks, hasMinOrMaxContentMinTrackSizingFunction, growthLimit, intrinsicContributionTypeMinimum)
				}
			}
		}
		flushPlannedBaseSizeIncreases(axisTracks)

		hasMaxContentMinTrackSizingFunction := func(track *gridTrack) bool {
			return track.minTrackSizingFunction.IsMaxContent()
		}

		if axisAvailableGridSpace == AvailableSpaceMaxContent {
			hasAutoMinTrackSizingFunction := func(track *gridTrack) bool {
				return track.minTrackSizingFunction.IsAuto() && !track.maxTrackSizingFunction.IsMinContent()
			}

			for _, item := range batch {
				if !item.spansTrackMatching(axis, axisTracks, func(track *gridTrack) bool {
					return hasAutoMinTrackSizingFunction(track) || hasMaxContentMinTrackSizingFunction(track)
				}) {
					continue
				}
				axisMaxContentSize := itemSizer.maxContentContribution(item, axisTracks)
				limit := item.spannedTrackLimit(axis, axisTracks, axisInnerNodeSize, sizerCalc)
				space := F32MaybeMin(axisMaxContentSize, limit)

				if isFlex {
					s, e := item.trackRangeExcludingLines(axis)
					spannedTracks := axisTracks[s:e]
					var inflexibleSizes float32
					for i := range spannedTracks {
						if !spannedTracks[i].isFlexible() {
							inflexibleSizes += spannedTracks[i].baseSize
						}
					}
					scale := f32Min(crossedFlexFactorSum(spannedTracks), 1.0)
					space = inflexibleSizes + float32(f32Max(space-inflexibleSizes, 0.0)*scale)
				}
				s, e := item.trackRangeExcludingLines(axis)
				tracks := axisTracks[s:e]
				if space > 0.0 {
					anyMaxContent := false
					for i := range tracks {
						if hasMaxContentMinTrackSizingFunction(&tracks[i]) {
							anyMaxContent = true
							break
						}
					}
					if anyMaxContent {
						distributeItemSpaceToBaseSize(isFlex, space, tracks, hasMaxContentMinTrackSizingFunction, func(*gridTrack) float32 { return inf }, intrinsicContributionTypeMaximum)
					} else {
						distributeItemSpaceToBaseSize(isFlex, space, tracks, hasAutoMinTrackSizingFunction, fitContentLimited, intrinsicContributionTypeMaximum)
					}
				}
			}
			flushPlannedBaseSizeIncreases(axisTracks)
		}

		for _, item := range batch {
			if !item.spansTrackMatching(axis, axisTracks, hasMaxContentMinTrackSizingFunction) {
				continue
			}
			space := itemSizer.maxContentContribution(item, axisTracks)
			s, e := item.trackRangeExcludingLines(axis)
			tracks := axisTracks[s:e]
			if space > 0.0 {
				distributeItemSpaceToBaseSize(isFlex, space, tracks, hasMaxContentMinTrackSizingFunction, growthLimit, intrinsicContributionTypeMaximum)
			}
		}
		flushPlannedBaseSizeIncreases(axisTracks)

		for i := range axisTracks {
			if axisTracks[i].growthLimit < axisTracks[i].baseSize {
				axisTracks[i].growthLimit = axisTracks[i].baseSize
			}
		}

		if !isFlex {
			hasIntrinsicMaxTrackSizingFunction := func(track *gridTrack) bool {
				return !track.maxTrackSizingFunction.HasDefiniteValue(axisInnerNodeSize)
			}
			for _, item := range batch {
				if !item.spansTrackMatching(axis, axisTracks, hasIntrinsicMaxTrackSizingFunction) {
					continue
				}
				space := itemSizer.minContentContribution(item, axisTracks)
				s, e := item.trackRangeExcludingLines(axis)
				tracks := axisTracks[s:e]
				if space > 0.0 {
					distributeItemSpaceToGrowthLimit(space, tracks, hasIntrinsicMaxTrackSizingFunction)
				}
			}
			flushPlannedGrowthLimitIncreases(axisTracks, true)

			hasMaxContentMaxTrackSizingFunction := func(track *gridTrack) bool {
				return track.maxTrackSizingFunction.IsMaxContentAlike() ||
					(track.maxTrackSizingFunction.UsesPercentage() && axisInnerNodeSize.IsNone())
			}
			for _, item := range batch {
				if !item.spansTrackMatching(axis, axisTracks, hasMaxContentMaxTrackSizingFunction) {
					continue
				}
				space := itemSizer.maxContentContribution(item, axisTracks)
				s, e := item.trackRangeExcludingLines(axis)
				tracks := axisTracks[s:e]
				if space > 0.0 {
					distributeItemSpaceToGrowthLimit(space, tracks, hasMaxContentMaxTrackSizingFunction)
				}
			}
			flushPlannedGrowthLimitIncreases(axisTracks, false)
		}
	}

	for i := range axisTracks {
		if axisTracks[i].growthLimit == inf {
			axisTracks[i].growthLimit = axisTracks[i].baseSize
		}
	}
}

func crossedFlexFactorSum(tracks []gridTrack) float32 {
	var sum float32
	for i := range tracks {
		if tracks[i].isFlexible() {
			sum += tracks[i].flexFactor()
		}
	}
	return sum
}

func distributeItemSpaceToBaseSize(
	isFlex bool,
	space float32,
	tracks []gridTrack,
	trackIsAffected func(*gridTrack) bool,
	trackLimit func(*gridTrack) float32,
	intrinsicContributionType intrinsicContributionType,
) {
	one := func(*gridTrack) float32 { return 1.0 }
	if isFlex {
		filter := func(track *gridTrack) bool { return track.isFlexible() && trackIsAffected(track) }

		var flexFactorSum float32
		for i := range tracks {
			if filter(&tracks[i]) {
				flexFactorSum += tracks[i].flexFactor()
			}
		}
		if flexFactorSum > 0.0 {
			distributeItemSpaceToBaseSizeInner(space, tracks, filter, (*gridTrack).flexFactor, trackLimit, intrinsicContributionType)
		} else {
			distributeItemSpaceToBaseSizeInner(space, tracks, filter, one, trackLimit, intrinsicContributionType)
		}
	} else {
		distributeItemSpaceToBaseSizeInner(space, tracks, trackIsAffected, one, trackLimit, intrinsicContributionType)
	}
}

func distributeItemSpaceToBaseSizeInner(
	space float32,
	tracks []gridTrack,
	trackIsAffected func(*gridTrack) bool,
	trackDistributionProportion func(*gridTrack) float32,
	trackLimit func(*gridTrack) float32,
	intrinsicContributionType intrinsicContributionType,
) {
	anyAffected := false
	for i := range tracks {
		if trackIsAffected(&tracks[i]) {
			anyAffected = true
			break
		}
	}
	if space == 0.0 || !anyAffected {
		return
	}

	getBaseSize := func(track *gridTrack) float32 { return track.baseSize }

	var trackSizes float32
	for i := range tracks {
		trackSizes += tracks[i].baseSize
	}
	extraSpace := f32Max(0.0, space-trackSizes)

	const threshold float32 = 0.000001

	extraSpace = distributeSpaceUpToLimits(extraSpace, tracks, trackIsAffected, trackDistributionProportion, getBaseSize, trackLimit)

	if extraSpace > threshold {
		var filter func(*gridTrack) bool
		switch intrinsicContributionType {
		case intrinsicContributionTypeMinimum:
			filter = func(track *gridTrack) bool { return track.maxTrackSizingFunction.IsIntrinsic() }
		case intrinsicContributionTypeMaximum:
			filter = func(track *gridTrack) bool { return track.maxTrackSizingFunction.IsMaxOrFitContent() }
		}

		numberOfTracks := 0
		for i := range tracks {
			if trackIsAffected(&tracks[i]) && filter(&tracks[i]) {
				numberOfTracks++
			}
		}
		if numberOfTracks == 0 {
			filter = func(*gridTrack) bool { return true }
		}

		distributeSpaceUpToLimits(
			extraSpace,
			tracks,
			func(track *gridTrack) bool { return trackIsAffected(track) && filter(track) },
			trackDistributionProportion,
			getBaseSize,
			(*gridTrack).getFitContentLimit,
		)
	}

	for i := range tracks {
		track := &tracks[i]
		if track.itemIncurredIncrease > track.baseSizePlannedIncrease {
			track.baseSizePlannedIncrease = track.itemIncurredIncrease
		}
		track.itemIncurredIncrease = 0.0
	}
}

func distributeItemSpaceToGrowthLimit(space float32, tracks []gridTrack, trackIsAffected func(*gridTrack) bool) {
	count := 0
	for i := range tracks {
		if trackIsAffected(&tracks[i]) {
			count++
		}
	}
	if space == 0.0 || count == 0 {
		return
	}

	inf := float32(math.Inf(1))
	growthLimitOrBaseSize := func(track *gridTrack) float32 {
		if track.growthLimit == inf {
			return track.baseSize
		}
		return track.growthLimit
	}

	var trackSizes float32
	for i := range tracks {
		trackSizes += growthLimitOrBaseSize(&tracks[i])
	}
	extraSpace := f32Max(0.0, space-trackSizes)

	one := func(*gridTrack) float32 { return 1.0 }
	extraSpace = distributeSpaceUpToLimits(
		extraSpace,
		tracks,
		func(track *gridTrack) bool {
			return trackIsAffected(track) && (track.infinitelyGrowable || track.growthLimit == inf)
		},
		one,
		growthLimitOrBaseSize,
		(*gridTrack).getFitContentLimit,
	)

	if extraSpace > 0.0 {
		distributeSpaceUpToLimits(extraSpace, tracks, trackIsAffected, one, growthLimitOrBaseSize, (*gridTrack).getFitContentLimit)
	}

	for i := range tracks {
		track := &tracks[i]
		if track.itemIncurredIncrease > track.growthLimitPlannedIncrease {
			track.growthLimitPlannedIncrease = track.itemIncurredIncrease
		}
		track.itemIncurredIncrease = 0.0
	}
}

func maximiseTracks(axisTracks []gridTrack, axisAvailableGridSpace AvailableSpace) {
	var usedSpace float32
	for i := range axisTracks {
		usedSpace += axisTracks[i].baseSize
	}
	freeSpace := axisAvailableGridSpace.ComputeFreeSpace(usedSpace)
	if freeSpace == float32(math.Inf(1)) {
		for i := range axisTracks {
			axisTracks[i].baseSize = axisTracks[i].growthLimit
		}
	} else if freeSpace > 0.0 {
		distributeSpaceUpToLimits(
			freeSpace,
			axisTracks,
			func(*gridTrack) bool { return true },
			func(*gridTrack) float32 { return 1.0 },
			func(track *gridTrack) float32 { return track.baseSize },
			(*gridTrack).fitContentLimitedGrowthLimit,
		)
		for i := range axisTracks {
			track := &axisTracks[i]
			track.baseSize += track.itemIncurredIncrease
			track.itemIncurredIncrease = 0.0
		}
	}
}

func expandFlexibleTracks(
	tree LayoutPartialTree,
	axis AbstractAxis,
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	items []gridItem,
	axisMinSize Option[float32],
	axisMaxSize Option[float32],
	axisAvailableSpaceForExpansion AvailableSpace,
	innerNodeSize Size[Option[float32]],
	getTrackSizeEstimate trackSizeEstimateFunc,
) {
	itemSizer := &intrinsicSizeMeasurer{tree: tree, otherAxisTracks: otherAxisTracks, axis: axis, innerNodeSize: innerNodeSize, getTrackSizeEstimate: getTrackSizeEstimate}

	var flexFraction float32
	switch axisAvailableSpaceForExpansion.kind {
	case AvailableSpaceKindDefinite:
		availableSpace := axisAvailableSpaceForExpansion.value
		var usedSpace float32
		for i := range axisTracks {
			usedSpace += axisTracks[i].baseSize
		}
		freeSpace := availableSpace - usedSpace
		if freeSpace <= 0.0 {
			flexFraction = 0.0
		} else {
			flexFraction = findSizeOfFr(axisTracks, availableSpace)
		}
	case AvailableSpaceKindMinContent:
		flexFraction = 0.0
	case AvailableSpaceKindMaxContent:
		trackMax := float32(0.0)
		haveTrack := false
		for i := range axisTracks {
			track := &axisTracks[i]
			if !track.maxTrackSizingFunction.IsFr() {
				continue
			}
			flexFactor := track.flexFactor()
			var v float32
			if flexFactor > 1.0 {
				v = track.baseSize / flexFactor
			} else {
				v = track.baseSize
			}
			if !haveTrack || f32TotalCmp(v, trackMax) >= 0 {
				trackMax = v
				haveTrack = true
			}
		}
		itemMax := float32(0.0)
		haveItem := false
		for i := range items {
			item := &items[i]
			if !item.crossesFlexibleTrack(axis) {
				continue
			}
			maxContentContribution := itemSizer.maxContentContribution(item, axisTracks)
			s, e := item.trackRangeExcludingLines(axis)
			v := findSizeOfFr(axisTracks[s:e], maxContentContribution)
			if !haveItem || f32TotalCmp(v, itemMax) >= 0 {
				itemMax = v
				haveItem = true
			}
		}
		flexFraction = f32Max(trackMax, itemMax)

		var hypotheticalGridSize float32
		for i := range axisTracks {
			track := &axisTracks[i]
			if track.maxTrackSizingFunction.IsFr() {
				trackFlexFactor := track.maxTrackSizingFunction.inner.Value()
				hypotheticalGridSize += f32Max(track.baseSize, float32(trackFlexFactor*flexFraction))
			} else {
				hypotheticalGridSize += track.baseSize
			}
		}
		minSize := axisMinSize.UnwrapOr(0.0)
		maxSize := axisMaxSize.UnwrapOr(float32(math.Inf(1)))
		if hypotheticalGridSize < minSize {
			flexFraction = findSizeOfFr(axisTracks, minSize)
		} else if hypotheticalGridSize > maxSize {
			flexFraction = findSizeOfFr(axisTracks, maxSize)
		}
	}

	for i := range axisTracks {
		track := &axisTracks[i]
		if !track.maxTrackSizingFunction.IsFr() {
			continue
		}
		trackFlexFactor := track.maxTrackSizingFunction.inner.Value()
		track.baseSize = f32Max(track.baseSize, float32(trackFlexFactor*flexFraction))
	}
}

func findSizeOfFr(tracks []gridTrack, spaceToFill float32) float32 {
	if spaceToFill == 0.0 {
		return 0.0
	}

	hypotheticalFrSize := float32(math.Inf(1))
	var previousIterHypotheticalFrSize float32
	maxIterations := len(tracks) + 1
	for range maxIterations {
		var usedSpace float32
		var naiveFlexFactorSum float32
		for i := range tracks {
			track := &tracks[i]
			if track.maxTrackSizingFunction.IsFr() &&
				float32(track.maxTrackSizingFunction.inner.Value()*hypotheticalFrSize) >= track.baseSize {
				naiveFlexFactorSum += track.maxTrackSizingFunction.inner.Value()
			} else {
				usedSpace += track.baseSize
			}
		}
		leftoverSpace := spaceToFill - usedSpace
		flexFactor := f32Max(naiveFlexFactorSum, 1.0)

		previousIterHypotheticalFrSize = hypotheticalFrSize
		hypotheticalFrSize = leftoverSpace / flexFactor

		valid := true
		for i := range tracks {
			track := &tracks[i]
			if track.maxTrackSizingFunction.IsFr() {
				ff := track.maxTrackSizingFunction.inner.Value()
				if !(float32(ff*hypotheticalFrSize) >= track.baseSize || float32(ff*previousIterHypotheticalFrSize) < track.baseSize) {
					valid = false
					break
				}
			}
		}
		if valid {
			break
		}
	}

	return hypotheticalFrSize
}

func stretchAutoTracks(axisTracks []gridTrack, axisMinSize Option[float32], axisAvailableSpaceForExpansion AvailableSpace) {
	numAutoTracks := 0
	for i := range axisTracks {
		if axisTracks[i].maxTrackSizingFunction.IsAuto() {
			numAutoTracks++
		}
	}
	if numAutoTracks > 0 {
		var usedSpace float32
		for i := range axisTracks {
			usedSpace += axisTracks[i].baseSize
		}

		var freeSpace float32
		if axisAvailableSpaceForExpansion.IsDefinite() {
			freeSpace = axisAvailableSpaceForExpansion.ComputeFreeSpace(usedSpace)
		} else if size, ok := axisMinSize.Get(); ok {
			freeSpace = size - usedSpace
		} else {
			freeSpace = 0.0
		}
		if freeSpace > 0.0 {
			extraSpacePerAutoTrack := freeSpace / float32(numAutoTracks)
			for i := range axisTracks {
				if axisTracks[i].maxTrackSizingFunction.IsAuto() {
					axisTracks[i].baseSize += extraSpacePerAutoTrack
				}
			}
		}
	}
}

func distributeSpaceUpToLimits(
	spaceToDistribute float32,
	tracks []gridTrack,
	trackIsAffected func(*gridTrack) bool,
	trackDistributionProportion func(*gridTrack) float32,
	trackAffectedProperty func(*gridTrack) float32,
	trackLimit func(*gridTrack) float32,
) float32 {
	const threshold float32 = 0.01

	maxIterations := len(tracks) + 1

	for range maxIterations {
		if spaceToDistribute <= threshold {
			break
		}
		var trackDistributionProportionSum float32
		for i := range tracks {
			track := &tracks[i]
			if trackAffectedProperty(track)+track.itemIncurredIncrease < trackLimit(track) && trackIsAffected(track) {
				trackDistributionProportionSum += trackDistributionProportion(track)
			}
		}

		if trackDistributionProportionSum == 0.0 {
			break
		}

		var minIncreaseLimit float32
		found := false
		for i := range tracks {
			track := &tracks[i]
			if trackAffectedProperty(track)+track.itemIncurredIncrease < trackLimit(track) && trackIsAffected(track) {
				v := (trackLimit(track) - trackAffectedProperty(track) - track.itemIncurredIncrease) / trackDistributionProportion(track)
				if !found || f32TotalCmp(v, minIncreaseLimit) < 0 {
					minIncreaseLimit = v
					found = true
				}
			}
		}
		if !found {
			panic("called `Option::unwrap()` on a `None` value")
		}
		iterationItemIncurredIncrease := f32Min(minIncreaseLimit, spaceToDistribute/trackDistributionProportionSum)

		for i := range tracks {
			track := &tracks[i]
			if !trackIsAffected(track) {
				continue
			}
			increase := float32(iterationItemIncurredIncrease * trackDistributionProportion(track))
			if increase > 0.0 && trackAffectedProperty(track)+track.itemIncurredIncrease+increase <= trackLimit(track)+threshold {
				track.itemIncurredIncrease += increase
				spaceToDistribute -= increase
			}
		}
	}

	return spaceToDistribute
}
