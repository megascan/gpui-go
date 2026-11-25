package taffy

import (
	"io"
	"sort"
	"strings"
)

type gridContainerConstants struct {
	direction             Direction
	contain               Contain
	containingBlockClaims ContainingBlockClaims
	padding               Rect[float32]
	border                Rect[float32]
	paddingBorderSize     Size[float32]
	minSize               Size[Option[float32]]
	maxSize               Size[Option[float32]]
	preferredSize         Size[Option[float32]]
	scrollbarGutter       Point[float32]
	isScrollContainer     bool
	contentBoxInset       Rect[float32]
	alignContent          AlignContent
	justifyContent        AlignContent
	alignItems            AlignItems
	justifyItems          AlignItems
	availableGridSpace    Size[AvailableSpace]
	outerNodeSize         Size[Option[float32]]
	innerMinSize          Size[Option[float32]]
	innerMaxSize          Size[Option[float32]]
	innerNodeSize         Size[Option[float32]]
}

func computeContainerConstants[S CheapCloneStr](tree LayoutGridContainer[S], style GridContainerStyle[S], inputs LayoutInput) gridContainerConstants {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	calcFn := treeCalc(tree)

	direction := style.Direction()
	contain := style.Contain()
	containingBlockClaims := style.IsContainingBlock()

	aspectRatio := style.AspectRatio()
	padding := RectResolveOrZeroOptionF32(style.Padding(), parentSize.Width, calcFn)
	border := RectResolveOrZeroOptionF32(style.Border(), parentSize.Width, calcFn)
	paddingBorder := padding.Add(border)
	paddingBorderSize := paddingBorder.SumAxes()
	boxSizingAdjustment := SizeF32Zero
	if style.BoxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	minSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MinSize(), parentSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	maxSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MaxSize(), parentSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	preferredSize := SizeOptionF32None
	if inputs.SizingMode == SizingModeInherentSize {
		preferredSize = SizeOptionF32MaybeAddF32(
			SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.Size(), parentSize, calcFn), style.AspectRatio()),
			boxSizingAdjustment,
		)
	}

	scrollbarGutter := style.Overflow().Transpose().Map(func(overflow Overflow) float32 {
		if overflow == OverflowScroll {
			return style.ScrollbarWidth()
		}
		return 0.0
	})
	overflow := style.Overflow()
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()
	contentBoxInset := paddingBorder
	contentBoxInset.Bottom += scrollbarGutter.Y

	switch direction {
	case DirectionLtr:
		contentBoxInset.Right += scrollbarGutter.X
	case DirectionRtl:
		contentBoxInset.Left += scrollbarGutter.X
	}

	alignContent := style.GridAlignContent(AbstractAxisBlock)
	justifyContent := style.GridAlignContent(AbstractAxisInline)
	alignItems := style.AlignItems()
	justifyItems := style.JustifyItems()
	styled := SizeOptionOr(knownDimensions, preferredSize)
	constrainedAvailableSpace := availableSpace
	if w, ok := styled.Width.Get(); ok {
		constrainedAvailableSpace.Width = AvailableSpaceDefinite(w)
	}
	if h, ok := styled.Height.Get(); ok {
		constrainedAvailableSpace.Height = AvailableSpaceDefinite(h)
	}
	constrainedAvailableSpace = SizeAvailableSpaceMaybeMaxF32(
		SizeAvailableSpaceMaybeClamp(constrainedAvailableSpace, minSize, maxSize),
		paddingBorderSize,
	)

	availableGridSpace := Size[AvailableSpace]{
		Width: constrainedAvailableSpace.Width.MapDefiniteValue(func(space float32) float32 {
			return space - contentBoxInset.HorizontalAxisSum()
		}),
		Height: constrainedAvailableSpace.Height.MapDefiniteValue(func(space float32) float32 {
			return space - contentBoxInset.VerticalAxisSum()
		}),
	}

	outerNodeSize := SizeOptionF32MaybeMaxF32(
		SizeOptionF32MaybeClamp(SizeOptionOr(knownDimensions, preferredSize), minSize, maxSize),
		paddingBorderSize,
	)

	innerMinSize := SizeOptionF32MaybeSubF32(minSize, contentBoxInset.SumAxes())
	innerMaxSize := SizeOptionF32MaybeSubF32(maxSize, contentBoxInset.SumAxes())
	innerNodeSize := Size[Option[float32]]{
		Width:  OptionF32MaybeSubF32(outerNodeSize.Width, contentBoxInset.HorizontalAxisSum()),
		Height: OptionF32MaybeSubF32(outerNodeSize.Height, contentBoxInset.VerticalAxisSum()),
	}

	return gridContainerConstants{
		direction:             direction,
		contain:               contain,
		containingBlockClaims: containingBlockClaims,
		padding:               padding,
		border:                border,
		paddingBorderSize:     paddingBorderSize,
		minSize:               minSize,
		maxSize:               maxSize,
		preferredSize:         preferredSize,
		scrollbarGutter:       scrollbarGutter,
		isScrollContainer:     isScrollContainer,
		contentBoxInset:       contentBoxInset,
		alignContent:          alignContent,
		justifyContent:        justifyContent,
		alignItems:            alignItems,
		justifyItems:          justifyItems,
		availableGridSpace:    availableGridSpace,
		outerNodeSize:         outerNodeSize,
		innerMinSize:          innerMinSize,
		innerMaxSize:          innerMaxSize,
		innerNodeSize:         innerNodeSize,
	}
}

func sumBaseSizes(tracks []gridTrack) float32 {
	var sum float32
	for i := range tracks {
		sum += tracks[i].baseSize
	}
	return sum
}

func baseSizeEstimate(track *gridTrack, _ Option[float32], _ LayoutPartialTree) Option[float32] {
	return some(track.baseSize)
}

// ComputeGridLayout computes the layout of a CSS grid container and its direct children.
func ComputeGridLayout[S CheapCloneStr](tree LayoutGridContainer[S], node NodeID, inputs LayoutInput) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	runMode := inputs.RunMode
	calcFn := treeCalc(tree)

	style := tree.GetGridContainerStyle(node)
	constants := computeContainerConstants(tree, style, inputs)
	direction := constants.direction
	contain := constants.contain
	containingBlockClaims := constants.containingBlockClaims
	padding := constants.padding
	border := constants.border
	paddingBorderSize := constants.paddingBorderSize
	minSize := constants.minSize
	maxSize := constants.maxSize
	preferredSize := constants.preferredSize
	scrollbarGutter := constants.scrollbarGutter
	isScrollContainer := constants.isScrollContainer
	contentBoxInset := constants.contentBoxInset
	alignContent := constants.alignContent
	justifyContent := constants.justifyContent
	alignItems := constants.alignItems
	justifyItems := constants.justifyItems
	availableGridSpace := constants.availableGridSpace
	outerNodeSize := constants.outerNodeSize
	innerMinSize := constants.innerMinSize
	innerMaxSize := constants.innerMaxSize
	innerNodeSize := constants.innerNodeSize

	if runMode == RunModeComputeSize {
		width, wok := outerNodeSize.Width.Get()
		height, hok := outerNodeSize.Height.Get()
		if wok && hok {
			return LayoutOutputFromOuterSize(Size[float32]{Width: width, Height: height})
		}

		if inputs.Axis == RequestedAxisHorizontal && wok {
			return LayoutOutputFromOuterSize(Size[float32]{Width: width, Height: 0.0})
		}
	}

	autoFitContainerSize := SizeOptionF32MaybeSubF32(
		SizeOptionF32MaybeMaxF32(
			SizeOptionF32MaybeClamp(SizeOptionOr(SizeOptionOr(outerNodeSize, maxSize), minSize), minSize, maxSize),
			paddingBorderSize,
		),
		contentBoxInset.SumAxes(),
	)

	autoRepeatFitStrategy := SizeOptionOr(outerNodeSize, maxSize).Map(func(val Option[float32]) autoRepeatStrategy {
		if val.IsSome() {
			return autoRepeatStrategyMaxRepetitionsThatDoNotOverflow
		}
		return autoRepeatStrategyMinRepetitionsThatDoOverflow
	})

	colAutoRepetitionCount, gridTemplateColCount := computeExplicitGridSizeInAxis(
		style,
		autoFitContainerSize.Width,
		autoRepeatFitStrategy.Width,
		calcFn,
		AbsoluteAxisHorizontal,
	)
	rowAutoRepetitionCount, gridTemplateRowCount := computeExplicitGridSizeInAxis(
		style,
		autoFitContainerSize.Height,
		autoRepeatFitStrategy.Height,
		calcFn,
		AbsoluteAxisVertical,
	)

	nameResolver := newNamedLineResolver(style, colAutoRepetitionCount, rowAutoRepetitionCount)

	explicitColCount := min(max(gridTemplateColCount, nameResolver.getAreaColumnCount()), maxGridTracks)
	explicitRowCount := min(max(gridTemplateRowCount, nameResolver.getAreaRowCount()), maxGridTracks)

	nameResolver.setExplicitColumnCount(explicitColCount)
	nameResolver.setExplicitRowCount(explicitRowCount)

	detailedColumnLineNames := nameResolver.detailedLineNames(AbsoluteAxisHorizontal)
	detailedRowLineNames := nameResolver.detailedLineNames(AbsoluteAxisVertical)

	childCount := tree.ChildCount(node)
	items := make([]gridItem, 0, childCount)
	placements := make([]itemPlacement, 0, childCount)
	index := 0
	for childNode := range tree.ChildIds(node) {
		childIndex := index
		index++
		childStyle := tree.GetGridChildStyle(childNode)
		if childStyle.BoxGenerationMode() == BoxGenerationModeNone || childStyle.Position().IsOutOfFlow() {
			continue
		}
		placements = append(placements, itemPlacement{
			horizontal: lineNonNamedGridPlacementIntoOriginZero(nameResolver.resolveColumnNames(childStyle.GridColumn()), explicitColCount),
			vertical:   lineNonNamedGridPlacementIntoOriginZero(nameResolver.resolveRowNames(childStyle.GridRow()), explicitRowCount),
		})
		items = append(items, newGridItemWithStyleAndOrder(
			childNode,
			childStyle,
			alignItems,
			justifyItems,
			uint16(childIndex),
		))
	}

	estColCounts, estRowCounts := computeGridSizeEstimate(explicitColCount, explicitRowCount, placements)

	cellOccupancyMatrix := cellOccupancyMatrixWithTrackCounts(estColCounts, estRowCounts)
	placeGridItems(&cellOccupancyMatrix, items, placements, style.GridAutoFlow())

	finalColCounts := *cellOccupancyMatrix.trackCounts(AbsoluteAxisHorizontal)
	finalRowCounts := *cellOccupancyMatrix.trackCounts(AbsoluteAxisVertical)

	var columns []gridTrack
	var rows []gridTrack
	initializeGridTracks(
		&columns,
		finalColCounts,
		style,
		AbsoluteAxisHorizontal,
		colAutoRepetitionCount,
		func(columnIndex int) bool { return cellOccupancyMatrix.columnIsOccupied(columnIndex) },
	)
	initializeGridTracks(
		&rows,
		finalRowCounts,
		style,
		AbsoluteAxisVertical,
		rowAutoRepetitionCount,
		func(rowIndex int) bool { return cellOccupancyMatrix.rowIsOccupied(rowIndex) },
	)

	resolveItemTrackIndexes(items, finalColCounts, finalRowCounts)
	determineIfItemCrossesFlexibleOrIntrinsicTracks(items, columns, rows)

	hasBaselineAlignedItem := false
	for i := range items {
		if items[i].participatesInBaselineAlignment() {
			hasBaselineAlignedItem = true
			break
		}
	}

	trackSizingAlgorithm(
		tree,
		AbstractAxisInline,
		innerMinSize.get(AbstractAxisInline),
		innerMaxSize.get(AbstractAxisInline),
		justifyContent,
		alignContent,
		availableGridSpace,
		innerNodeSize,
		columns,
		rows,
		items,
		func(track *gridTrack, parentSize Option[float32], tree LayoutPartialTree) Option[float32] {
			return track.maxTrackSizingFunction.DefiniteValue(parentSize, treeCalc(tree))
		},
		hasBaselineAlignedItem,
	)
	initialColumnSum := sumBaseSizes(columns)
	if innerNodeSize.Width.IsNone() {
		innerNodeSize.Width = some(initialColumnSum)
	}

	if runMode == RunModeComputeSize && inputs.Axis == RequestedAxisHorizontal {
		width := f32Max(
			F32MaybeClamp(initialColumnSum+contentBoxInset.HorizontalAxisSum(), minSize.Width, maxSize.Width),
			paddingBorderSize.Width,
		)
		return LayoutOutputFromOuterSize(Size[float32]{Width: width, Height: 0.0})
	}

	for i := range items {
		items[i].gridAreaSizeCache = none[Size[Option[float32]]]()
	}

	trackSizingAlgorithm(
		tree,
		AbstractAxisBlock,
		innerMinSize.get(AbstractAxisBlock),
		innerMaxSize.get(AbstractAxisBlock),
		alignContent,
		justifyContent,
		availableGridSpace,
		innerNodeSize,
		rows,
		columns,
		items,
		baseSizeEstimate,
		false,
	)
	initialRowSum := sumBaseSizes(rows)

	resolvedStyleSize := SizeOptionOr(knownDimensions, preferredSize)
	containerBorderBox := Size[float32]{
		Width: f32Max(
			F32MaybeClamp(
				resolvedStyleSize.get(AbstractAxisInline).UnwrapOr(initialColumnSum+contentBoxInset.HorizontalAxisSum()),
				minSize.Width,
				maxSize.Width,
			),
			paddingBorderSize.Width,
		),
		Height: f32Max(
			F32MaybeClamp(
				resolvedStyleSize.get(AbstractAxisBlock).UnwrapOr(initialRowSum+contentBoxInset.VerticalAxisSum()),
				minSize.Height,
				maxSize.Height,
			),
			paddingBorderSize.Height,
		),
	}
	containerContentBox := Size[float32]{
		Width:  f32Max(0.0, containerBorderBox.Width-contentBoxInset.HorizontalAxisSum()),
		Height: f32Max(0.0, containerBorderBox.Height-contentBoxInset.VerticalAxisSum()),
	}

	if runMode == RunModeComputeSize {
		return LayoutOutputFromOuterSize(containerBorderBox)
	}

	innerNodeSize = containerContentBox.Map(someF32)

	var rerunColumnSizing bool
	intrinsicColumnContributionChanged := false

	hasPercentageColumn := false
	for i := range columns {
		if columns[i].usesPercentage() {
			hasPercentageColumn = true
			break
		}
	}
	hasPercentageRow := false
	for i := range rows {
		if rows[i].usesPercentage() {
			hasPercentageRow = true
			break
		}
	}
	parentWidthIndefinite := outerNodeSize.Width.IsNone()
	rerunColumnSizing = parentWidthIndefinite && hasPercentageColumn

	estimateBase := func(track *gridTrack, _ Option[float32]) Option[float32] { return some(track.baseSize) }

	if !rerunColumnSizing {
		anyChanged := false
		for i := range items {
			item := &items[i]
			if !item.crossesIntrinsicColumn {
				continue
			}
			gridAreaSize := item.gridAreaSize(AbstractAxisInline, columns, rows, innerNodeSize, estimateBase, calcFn)
			availableSpace := gridAreaSize.with(AbstractAxisInline, none[float32]())
			newMinContentContribution := item.minContentContribution(AbstractAxisInline, tree, gridAreaSize, availableSpace)

			cached, cok := item.minContentContributionCache.Width.Get()
			hasChanged := !cok || cached != newMinContentContribution

			item.gridAreaSizeCache = some(gridAreaSize)
			item.minContentContributionCache.Width = some(newMinContentContribution)
			item.maxContentContributionCache.Width = none[float32]()
			item.minimumContributionCache.Width = none[float32]()

			anyChanged = anyChanged || hasChanged
		}
		intrinsicColumnContributionChanged = anyChanged
		rerunColumnSizing = intrinsicColumnContributionChanged
	} else {
		for i := range items {
			item := &items[i]
			item.gridAreaSizeCache = none[Size[Option[float32]]]()
			item.minContentContributionCache.Width = none[float32]()
			item.maxContentContributionCache.Width = none[float32]()
			item.minimumContributionCache.Width = none[float32]()
		}
	}

	parentHeightIndefinite := outerNodeSize.Height.IsNone()
	rerunRowSizing := parentHeightIndefinite && hasPercentageRow
	intrinsicRowContributionChanged := false

	if rerunRowSizing {
		for i := range items {
			item := &items[i]
			item.gridAreaSizeCache = none[Size[Option[float32]]]()
			item.minContentContributionCache.Height = none[float32]()
			item.maxContentContributionCache.Height = none[float32]()
			item.minimumContributionCache.Height = none[float32]()
		}
	}

	if rerunColumnSizing {
		trackSizingAlgorithm(
			tree,
			AbstractAxisInline,
			innerMinSize.get(AbstractAxisInline),
			innerMaxSize.get(AbstractAxisInline),
			justifyContent,
			alignContent,
			availableGridSpace,
			innerNodeSize,
			columns,
			rows,
			items,
			baseSizeEstimate,
			hasBaselineAlignedItem,
		)

		if !rerunRowSizing {
			anyChanged := false
			for i := range items {
				item := &items[i]
				if !item.crossesIntrinsicRow {
					continue
				}
				gridAreaSize := item.gridAreaSize(AbstractAxisBlock, rows, columns, innerNodeSize, estimateBase, calcFn)
				availableSpace := gridAreaSize.with(AbstractAxisBlock, none[float32]())
				newMinContentContribution := item.minContentContribution(AbstractAxisBlock, tree, gridAreaSize, availableSpace)

				cached, cok := item.minContentContributionCache.Height.Get()
				hasChanged := !cok || cached != newMinContentContribution

				item.gridAreaSizeCache = some(gridAreaSize)
				item.minContentContributionCache.Height = some(newMinContentContribution)
				item.maxContentContributionCache.Height = none[float32]()
				item.minimumContributionCache.Height = none[float32]()

				anyChanged = anyChanged || hasChanged
			}
			intrinsicRowContributionChanged = anyChanged
			rerunRowSizing = intrinsicRowContributionChanged
		}
	}

	if rerunRowSizing {
		trackSizingAlgorithm(
			tree,
			AbstractAxisBlock,
			innerMinSize.get(AbstractAxisBlock),
			innerMaxSize.get(AbstractAxisBlock),
			alignContent,
			justifyContent,
			availableGridSpace,
			innerNodeSize,
			rows,
			columns,
			items,
			baseSizeEstimate,
			false,
		)
	}

	if (intrinsicColumnContributionChanged && !hasPercentageColumn) || (intrinsicRowContributionChanged && !hasPercentageRow) {
		finalColumnSum := sumBaseSizes(columns)
		finalRowSum := sumBaseSizes(rows)

		if intrinsicColumnContributionChanged && !hasPercentageColumn {
			containerBorderBox.Width = f32Max(
				F32MaybeClamp(
					resolvedStyleSize.get(AbstractAxisInline).UnwrapOr(finalColumnSum+contentBoxInset.HorizontalAxisSum()),
					minSize.Width,
					maxSize.Width,
				),
				paddingBorderSize.Width,
			)
			containerContentBox.Width = f32Max(0.0, containerBorderBox.Width-contentBoxInset.HorizontalAxisSum())
		}

		if intrinsicRowContributionChanged && !hasPercentageRow {
			containerBorderBox.Height = f32Max(
				F32MaybeClamp(
					resolvedStyleSize.get(AbstractAxisBlock).UnwrapOr(finalRowSum+contentBoxInset.VerticalAxisSum()),
					minSize.Height,
					maxSize.Height,
				),
				paddingBorderSize.Height,
			)
			containerContentBox.Height = f32Max(0.0, containerBorderBox.Height-contentBoxInset.VerticalAxisSum())
		}
	}

	if runMode == RunModeComputeSize {
		return LayoutOutputFromOuterSize(containerBorderBox)
	}

	inlineSizeWithoutScrollbar := f32Max(containerBorderBox.Width-paddingBorderSize.Width, 0.0)
	inlineScrollbarGutterForAlignment := f32Min(scrollbarGutter.X, inlineSizeWithoutScrollbar)
	columnPadding := Line[float32]{Start: padding.Left, End: padding.Right}
	if direction.IsRtl() {
		columnPadding.Start += inlineScrollbarGutterForAlignment
	} else {
		columnPadding.End += inlineScrollbarGutterForAlignment
	}
	alignTracks(
		containerContentBox.get(AbstractAxisInline),
		columnPadding,
		Line[float32]{Start: border.Left, End: border.Right},
		columns,
		justifyContent,
		direction.IsRtl(),
	)
	alignTracks(
		containerContentBox.get(AbstractAxisBlock),
		Line[float32]{Start: padding.Top, End: padding.Bottom},
		Line[float32]{Start: border.Top, End: border.Bottom},
		rows,
		alignContent,
		false,
	)

	itemOverflowRect := RectF32Zero

	oofCandidates := NewOofCandidates()

	containerAlignmentStyles := inBothAbsAxis[AlignItems]{horizontal: justifyItems, vertical: alignItems}

	for index := range items {
		item := &items[index]
		gridArea := Rect[float32]{
			Top:    rows[int(item.rowIndexes.Start)+1].offset,
			Bottom: rows[int(item.rowIndexes.End)].offset,
		}
		if direction.IsRtl() {
			gridArea.Left = columns[int(item.columnIndexes.End)-1].offset
			gridArea.Right = columns[int(item.columnIndexes.Start)].offset
		} else {
			gridArea.Left = columns[int(item.columnIndexes.Start)+1].offset
			gridArea.Right = columns[int(item.columnIndexes.End)].offset
		}
		overflowContribution, yPosition, height := alignAndPositionItem(
			tree,
			item.node,
			uint32(index),
			gridArea,
			Size[float32]{Width: gridArea.Right - gridArea.Left, Height: gridArea.Bottom - gridArea.Top},
			containerAlignmentStyles,
			item.baselineShim,
			direction,
			containerBorderBox.Width,
			border,
			isScrollContainer,
			&item.oofCandidates,
		)
		item.yPosition = yPosition
		item.height = height

		itemOverflowRect = itemOverflowRect.Union(overflowContribution)
	}

	order := uint32(len(items))
	nextItem := 0
	for index := range tree.ChildCount(node) {
		if nextItem < len(items) && int(items[nextItem].sourceOrder) == index {
			oofCandidates.Append(&items[nextItem].oofCandidates)
			nextItem++
			continue
		}

		child := tree.GetChildID(node, index)
		childStyle := tree.GetGridChildStyle(child)

		if childStyle.BoxGenerationMode() == BoxGenerationModeNone {
			layout := LayoutWithOrder(order)
			tree.SetUnroundedLayout(child, &layout)
			performChildLayout(
				tree,
				child,
				SizeOptionF32None,
				SizeOptionF32None,
				sizeAvailableSpaceMaxContent,
				SizingModeInherentSize,
				LineBoolFalse,
			)
			order++
			continue
		}

		if childStyle.Position().IsOutOfFlow() {
			position := childStyle.Position()
			itemDirection := childStyle.Direction()
			justifySelf := childStyle.JustifySelf().UnwrapOr(justifyItems).ResolveSelfRelative(itemDirection, direction, true)
			alignSelf := childStyle.AlignSelf().UnwrapOr(alignItems).ResolveSelfRelative(itemDirection, direction, false)

			var area Rect[float32]
			if containingBlockClaims.ForPosition(position) {
				area = resolveStaticPositionGridArea(
					childStyle,
					&nameResolver,
					finalRowCounts,
					finalColCounts,
					rows,
					columns,
					direction,
					border,
					scrollbarGutter,
					containerBorderBox,
				)
			} else {
				rtlGutter, ltrGutter := float32(0.0), scrollbarGutter.X
				if direction.IsRtl() {
					rtlGutter, ltrGutter = scrollbarGutter.X, 0.0
				}
				area = Rect[float32]{
					Left:   border.Left + padding.Left + rtlGutter,
					Right:  containerBorderBox.Width - border.Right - padding.Right - ltrGutter,
					Top:    border.Top + padding.Top,
					Bottom: containerBorderBox.Height - border.Bottom - padding.Bottom - scrollbarGutter.Y,
				}
			}

			oofCandidates.Push(OofCandidate{
				Node:     child,
				Order:    order,
				Position: position,
				StaticPosition: Point[AxisStaticPosition]{
					X: AxisStaticPositionFromAlignment(justifySelf, Line[float32]{Start: area.Left, End: area.Right}, direction.IsRtl()),
					Y: AxisStaticPositionFromAlignment(alignSelf, Line[float32]{Start: area.Top, End: area.Bottom}, false),
				},
			})

			order++
		}
	}

	absolutePositionInset := border
	if direction.IsRtl() {
		absolutePositionInset.Left += scrollbarGutter.X
	} else {
		absolutePositionInset.Right += scrollbarGutter.X
	}
	absolutePositionInset.Bottom += scrollbarGutter.Y
	absolutePositionArea := containerBorderBox.Sub(absolutePositionInset.SumAxes())
	absolutePositionOffset := Point[float32]{X: absolutePositionInset.Left, Y: absolutePositionInset.Top}
	areas := nameResolver.populateDetailedLineResolvers(&detailedRowLineNames, &detailedColumnLineNames)
	itemsInfo := make([]DetailedGridItemsInfo, len(items))
	for i := range items {
		itemsInfo[i] = detailedGridItemsInfoFromGridItem(&items[i])
	}
	tree.SetDetailedGridInfo(node, DetailedGridInfo[S]{
		Rows:    detailedGridTracksInfoFromGridTracksAndTrackCount(finalRowCounts, rows, detailedRowLineNames),
		Columns: detailedGridTracksInfoFromGridTracksAndTrackCount(finalColCounts, columns, detailedColumnLineNames),
		Items:   itemsInfo,
		areas:   areas,
	})

	oofPositioningArea := some(OofPositioningArea{Size: absolutePositionArea, Offset: absolutePositionOffset})

	if len(items) == 0 {
		overflowRect := itemOverflowRect
		if isScrollContainer {
			if direction.IsRtl() {
				overflowRect.Right += padding.Left
			} else {
				overflowRect.Right += padding.Right
			}
			overflowRect.Bottom += padding.Bottom
		}
		output := LayoutOutputFromSizes(containerBorderBox, overflowRect)
		output.OofCandidates = oofCandidates
		output.OofPositioningArea = oofPositioningArea
		return output
	}

	gridContainerBaseline := none[float32]()
	if !contain.SuppressesBaseline() {
		firstRow := items[0].rowIndexes.Start
		for i := range items {
			firstRow = min(firstRow, items[i].rowIndexes.Start)
		}

		var firstItem *gridItem
		var found *gridItem
		for i := range items {
			candidate := &items[i]
			if candidate.rowIndexes.Start != firstRow {
				continue
			}
			if firstItem == nil {
				firstItem = candidate
				if firstItem.participatesInBaselineAlignment() {
					found = firstItem
					break
				}
				continue
			}
			if candidate.participatesInBaselineAlignment() {
				found = candidate
				break
			}
		}
		if firstItem == nil {
			panic("called `Option::unwrap()` on a `None` value")
		}
		item := firstItem
		if found != nil {
			item = found
		}

		gridContainerBaseline = some(item.yPosition + item.baseline.UnwrapOr(item.height))
	}

	scrollableOverflowRect := itemOverflowRect
	if isScrollContainer {
		if direction.IsRtl() {
			scrollableOverflowRect.Right += padding.Left
		} else {
			scrollableOverflowRect.Right += padding.Right
		}
		scrollableOverflowRect.Bottom += padding.Bottom
	}

	output := LayoutOutputFromSizesAndBaselines(
		containerBorderBox,
		scrollableOverflowRect,
		BaselinesFromFirst(gridContainerBaseline),
	)
	output.OofCandidates = oofCandidates
	output.OofPositioningArea = oofPositioningArea
	return output
}

func lineAsStartEdge(tracks []gridTrack, index int) float32 {
	if index+1 < len(tracks) {
		return tracks[index+1].offset
	}
	return tracks[index].offset
}

func lineAsEndEdge(tracks []gridTrack, index int) float32 {
	if index == 0 {
		if len(tracks) > 1 {
			return tracks[1].offset
		}
		return tracks[0].offset
	}
	return tracks[index].offset
}

func rtlLineAsStartEdge(tracks []gridTrack, index int) float32 {
	if len(tracks) > index+1 {
		return tracks[index].offset
	} else if index == 0 {
		return tracks[0].offset
	}
	return tracks[index-1].offset
}

func rtlLineAsEndEdge(tracks []gridTrack, index int) float32 {
	if index == 0 {
		return tracks[0].offset
	}
	return tracks[index-1].offset
}

func resolveStaticPositionGridArea[S CheapCloneStr](
	childStyle GridItemStyle[S],
	nameResolver *namedLineResolver[S],
	finalRowCounts trackCounts,
	finalColCounts trackCounts,
	rows []gridTrack,
	columns []gridTrack,
	direction Direction,
	border Rect[float32],
	scrollbarGutter Point[float32],
	containerBorderBox Size[float32],
) Rect[float32] {
	toIndexes := func(line Line[Option[OriginZeroLine]], counts trackCounts) Line[Option[int]] {
		return line.Map(func(maybeGridLine Option[OriginZeroLine]) Option[int] {
			l, ok := maybeGridLine.Get()
			if !ok {
				return none[int]()
			}
			return l.tryIntoTrackVecIndex(counts)
		})
	}
	maybeColIndexes := toIndexes(
		LineOriginZeroGridPlacementResolveAbsolutelyPositionedGridTracks(
			lineNonNamedGridPlacementIntoOriginZero(nameResolver.resolveColumnNames(childStyle.GridColumn()), finalColCounts.explicit),
		),
		finalColCounts,
	)
	maybeRowIndexes := toIndexes(
		LineOriginZeroGridPlacementResolveAbsolutelyPositionedGridTracks(
			lineNonNamedGridPlacementIntoOriginZero(nameResolver.resolveRowNames(childStyle.GridRow()), finalRowCounts.explicit),
		),
		finalRowCounts,
	)

	var gridAreaLeft, gridAreaRight float32
	if direction.IsRtl() {
		if index, ok := maybeColIndexes.End.Get(); ok {
			gridAreaLeft = rtlLineAsEndEdge(columns, index)
		} else {
			gridAreaLeft = border.Left + scrollbarGutter.X
		}
		if index, ok := maybeColIndexes.Start.Get(); ok {
			gridAreaRight = rtlLineAsStartEdge(columns, index)
		} else {
			gridAreaRight = containerBorderBox.Width - border.Right
		}
	} else {
		if index, ok := maybeColIndexes.Start.Get(); ok {
			gridAreaLeft = lineAsStartEdge(columns, index)
		} else {
			gridAreaLeft = border.Left
		}
		if index, ok := maybeColIndexes.End.Get(); ok {
			gridAreaRight = lineAsEndEdge(columns, index)
		} else {
			gridAreaRight = containerBorderBox.Width - border.Right - scrollbarGutter.X
		}
	}

	top := border.Top
	if index, ok := maybeRowIndexes.Start.Get(); ok {
		top = lineAsStartEdge(rows, index)
	}
	bottom := containerBorderBox.Height - border.Bottom - scrollbarGutter.Y
	if index, ok := maybeRowIndexes.End.Get(); ok {
		bottom = lineAsEndEdge(rows, index)
	}

	return Rect[float32]{Top: top, Bottom: bottom, Left: gridAreaLeft, Right: gridAreaRight}
}

// DetailedGridInfo is the information from the computation of a grid container's layout.
type DetailedGridInfo[S CheapCloneStr] struct {
	Rows    DetailedGridTracksInfo[S]
	Columns DetailedGridTracksInfo[S]
	Items   []DetailedGridItemsInfo
	areas   gridAreasMap[S]
}

func (d *DetailedGridTracksInfo[S]) resolveAbsoluteGridAxis(
	placement Line[GridPlacement[S]],
	areas gridAreasMap[S],
	axis gridAreaAxis,
	paddingStart float32,
	paddingEnd float32,
	isReversed bool,
) Line[float32] {
	counts := trackCounts{
		negativeImplicit: d.NegativeImplicitTracks,
		explicit:         d.ExplicitTracks,
		positiveImplicit: d.PositiveImplicitTracks,
	}
	minLine := -int16(counts.negativeImplicit)
	maxLine := int16(counts.explicit + counts.positiveImplicit)
	resolved := LineOriginZeroGridPlacementResolveAbsolutelyPositionedGridTracks(
		lineNonNamedGridPlacementIntoOriginZero(d.LineNames.resolveLineNames(placement, areas, axis, d.ExplicitTracks), d.ExplicitTracks),
	)
	lines := resolved.Map(func(line Option[OriginZeroLine]) Option[int] {
		l, ok := line.Get()
		if !ok || !(int16(l) >= minLine && int16(l) <= maxLine) {
			return none[int]()
		}
		index, ok := l.tryIntoTrackVecIndex(counts).Get()
		if !ok {
			return none[int]()
		}
		return some(index / 2)
	})

	var startPosition float32
	if line, ok := lines.Start.Get(); ok {
		switch {
		case line < len(d.Positions):
			startPosition = d.absoluteTrackEdge(line, true, isReversed)
		case len(d.Positions) > 0:
			track := d.Positions[len(d.Positions)-1]
			if isReversed {
				startPosition = track.Start
			} else {
				startPosition = track.End
			}
		default:
			ok = false
			if v, eok := d.EmptyAxisLine.Get(); eok {
				startPosition, ok = v, true
			}
		}
		if !ok {
			startPosition = paddingStart
			if isReversed {
				startPosition = paddingEnd
			}
		}
	} else if isReversed {
		startPosition = paddingEnd
	} else {
		startPosition = paddingStart
	}

	var endPosition float32
	if line, ok := lines.End.Get(); ok {
		switch {
		case line >= 1 && line-1 < len(d.Positions):
			endPosition = d.absoluteTrackEdge(line-1, false, isReversed)
		case len(d.Positions) > 0:
			track := d.Positions[0]
			if isReversed {
				endPosition = track.End
			} else {
				endPosition = track.Start
			}
		default:
			ok = false
			if v, eok := d.EmptyAxisLine.Get(); eok {
				endPosition, ok = v, true
			}
		}
		if !ok {
			endPosition = paddingEnd
			if isReversed {
				endPosition = paddingStart
			}
		}
	} else if isReversed {
		endPosition = paddingStart
	} else {
		endPosition = paddingEnd
	}

	if isReversed {
		return Line[float32]{Start: f32Min(startPosition, endPosition), End: startPosition}
	}
	return Line[float32]{Start: startPosition, End: f32Max(startPosition, endPosition)}
}

func (d *DetailedGridTracksInfo[S]) absoluteTrackEdge(index int, isStart bool, isReversed bool) float32 {
	rangeIndex := sort.Search(len(d.CollapsedTracks), func(i int) bool { return !(d.CollapsedTracks[i].End <= index) })
	track := d.Positions[index]
	if rangeIndex < len(d.CollapsedTracks) && d.CollapsedTracks[rangeIndex].Contains(index) {
		r := d.CollapsedTracks[rangeIndex]
		adjacent := -1
		if isStart {
			adjacent = r.End
		} else if r.Start >= 1 {
			adjacent = r.Start - 1
		}
		if adjacent >= 0 && adjacent < len(d.Positions) {
			track = d.Positions[adjacent]
		}
	}
	if isStart != isReversed {
		return track.Start
	}
	return track.End
}

// WriteGridTemplateRows writes the used row track sizes and line names to out in the resolved value format of grid-template-rows, dividing sizes by zoom.
func (d *DetailedGridInfo[S]) WriteGridTemplateRows(out io.Writer, zoom float32) error {
	return d.Rows.WriteTrackList(out, zoom)
}

// WriteGridTemplateColumns writes the used column track sizes and line names to out in the resolved value format of grid-template-columns, dividing sizes by zoom.
func (d *DetailedGridInfo[S]) WriteGridTemplateColumns(out io.Writer, zoom float32) error {
	return d.Columns.WriteTrackList(out, zoom)
}

// GridTemplateRows serializes the used row track sizes and line names in the resolved value format of grid-template-rows.
func (d *DetailedGridInfo[S]) GridTemplateRows(zoom float32) string {
	return d.Rows.ToTrackListString(zoom)
}

// GridTemplateColumns serializes the used column track sizes and line names in the resolved value format of grid-template-columns.
func (d *DetailedGridInfo[S]) GridTemplateColumns(zoom float32) string {
	return d.Columns.ToTrackListString(zoom)
}

// ResolveAbsoluteGridArea resolves the physical grid area of an absolutely positioned box from its grid placement, relative to the container's border box.
func (d *DetailedGridInfo[S]) ResolveAbsoluteGridArea(gridRow, gridColumn Line[GridPlacement[S]], direction Direction, paddingBox Rect[float32]) Rect[float32] {
	columns := d.Columns.resolveAbsoluteGridAxis(gridColumn, d.areas, gridAreaAxisColumn, paddingBox.Left, paddingBox.Right, direction.IsRtl())
	rows := d.Rows.resolveAbsoluteGridAxis(gridRow, d.areas, gridAreaAxisRow, paddingBox.Top, paddingBox.Bottom, false)
	return Rect[float32]{Left: columns.Start, Right: columns.End, Top: rows.Start, Bottom: rows.End}
}

// ItemGridArea returns the location and size of the grid area of the item at itemIndex, relative to the container's border box, or false if the index is out of bounds.
func (d *DetailedGridInfo[S]) ItemGridArea(itemIndex int) (Point[float32], Size[float32], bool) {
	if itemIndex < 0 || itemIndex >= len(d.Items) {
		return Point[float32]{}, Size[float32]{}, false
	}
	item := d.Items[itemIndex]
	startCol := d.Columns.Positions[int(item.ColumnStart)-1]
	endCol := d.Columns.Positions[int(item.ColumnEnd)-2]
	left := f32Min(startCol.Start, endCol.Start)
	right := f32Max(startCol.End, endCol.End)
	top := d.Rows.Positions[int(item.RowStart)-1].Start
	bottom := d.Rows.Positions[int(item.RowEnd)-2].End
	return Point[float32]{X: left, Y: top}, Size[float32]{Width: right - left, Height: bottom - top}, true
}

// TrackRange is a half-open range of track indices (upstream `Range<usize>`).
type TrackRange struct {
	Start int
	End   int
}

// Contains reports whether index lies in the range.
func (r TrackRange) Contains(index int) bool {
	return r.Start <= index && index < r.End
}

// DetailedGridTracksInfo is the information from the computation of a grid's tracks in one axis.
type DetailedGridTracksInfo[S CheapCloneStr] struct {
	NegativeImplicitTracks uint16
	ExplicitTracks         uint16
	PositiveImplicitTracks uint16
	Positions              []Line[float32]
	CollapsedTracks        []TrackRange
	EmptyAxisLine          Option[float32]
	LineNames              GridLineNames[S]
}

func positionsFromGridTrackLayout(gridTracks []gridTrack) ([]Line[float32], []TrackRange) {
	positions := make([]Line[float32], 0, len(gridTracks)/2)
	var collapsedTracks []TrackRange
	for i := range gridTracks {
		track := &gridTracks[i]
		if track.kind != gridTrackKindTrack {
			continue
		}
		if track.isCollapsed {
			index := len(positions)
			if n := len(collapsedTracks); n > 0 && collapsedTracks[n-1].End == index {
				collapsedTracks[n-1].End++
			} else {
				collapsedTracks = append(collapsedTracks, TrackRange{Start: index, End: index + 1})
			}
		}
		positions = append(positions, Line[float32]{Start: track.offset, End: track.offset + track.baseSize})
	}
	return positions, collapsedTracks
}

func detailedGridTracksInfoFromGridTracksAndTrackCount[S CheapCloneStr](trackCount trackCounts, gridTracks []gridTrack, lineNames GridLineNames[S]) DetailedGridTracksInfo[S] {
	positions, collapsedTracks := positionsFromGridTrackLayout(gridTracks)
	emptyAxisLine := none[float32]()
	if len(positions) == 0 && len(gridTracks) > 0 {
		emptyAxisLine = some(gridTracks[0].offset)
	}
	return DetailedGridTracksInfo[S]{
		NegativeImplicitTracks: trackCount.negativeImplicit,
		ExplicitTracks:         trackCount.explicit,
		PositiveImplicitTracks: trackCount.positiveImplicit,
		Positions:              positions,
		CollapsedTracks:        collapsedTracks,
		EmptyAxisLine:          emptyAxisLine,
		LineNames:              lineNames,
	}
}

// NamesForLine returns the names of the 0-indexed grid line lineIndex of the full grid (including implicit tracks), or an empty slice.
func (d *DetailedGridTracksInfo[S]) NamesForLine(lineIndex int) []S {
	storedIndex := lineIndex - int(d.NegativeImplicitTracks)
	if lineIndex < int(d.NegativeImplicitTracks) {
		return []S{}
	}
	return d.LineNames.Line(storedIndex)
}

// IterLineNames iterates over the name group of each grid line of the full grid in line order, yielding empty groups for unnamed lines.
func (d *DetailedGridTracksInfo[S]) IterLineNames() GridLineNamesIter[S] {
	if d.LineNames.IsEmpty() {
		return d.LineNames.Iter()
	}
	totalLineCount := len(d.Positions) + 1
	leadingEmpty := int(d.NegativeImplicitTracks)
	trailingEmpty := max(totalLineCount-(leadingEmpty+d.LineNames.LineCount()), 0)
	return d.LineNames.iterPadded(leadingEmpty, trailingEmpty)
}

func writeLineNames[S CheapCloneStr](out io.Writer, names []S) error {
	var b strings.Builder
	b.WriteByte('[')
	for i, name := range names {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteString(string(name))
	}
	b.WriteByte(']')
	_, err := io.WriteString(out, b.String())
	return err
}

// WriteTrackList writes the used track sizes and line names of this axis to out in the resolved value format of grid-template-rows / grid-template-columns, dividing sizes by zoom.
func (d *DetailedGridTracksInfo[S]) WriteTrackList(out io.Writer, zoom float32) error {
	if len(d.Positions) == 0 {
		_, err := io.WriteString(out, "none")
		return err
	}

	needsSpace := false
	for trackIndex, position := range d.Positions {
		names := d.NamesForLine(trackIndex)
		if len(names) != 0 {
			if needsSpace {
				if _, err := io.WriteString(out, " "); err != nil {
					return err
				}
			}
			if err := writeLineNames(out, names); err != nil {
				return err
			}
			needsSpace = true
		}
		if needsSpace {
			if _, err := io.WriteString(out, " "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, formatF32((position.End-position.Start)/zoom)+"px"); err != nil {
			return err
		}
		needsSpace = true
	}
	trailingNames := d.NamesForLine(len(d.Positions))
	if len(trailingNames) != 0 {
		if _, err := io.WriteString(out, " "); err != nil {
			return err
		}
		if err := writeLineNames(out, trailingNames); err != nil {
			return err
		}
	}
	return nil
}

// ToTrackListString serializes the used track sizes and line names of this axis in the resolved value format of grid-template-rows / grid-template-columns.
func (d *DetailedGridTracksInfo[S]) ToTrackListString(zoom float32) string {
	var out strings.Builder
	if err := d.WriteTrackList(&out, zoom); err != nil {
		panic("writing to a String cannot fail")
	}
	return out.String()
}

// DetailedGridItemsInfo is the grid area of an item from the placement algorithm, as 1-indexed grid line numbers.
type DetailedGridItemsInfo struct {
	RowStart    uint16
	RowEnd      uint16
	ColumnStart uint16
	ColumnEnd   uint16
}

func toOneIndexedGridLine(gridTrackIndex uint16) uint16 {
	return gridTrackIndex/2 + 1
}

func detailedGridItemsInfoFromGridItem(gridItem *gridItem) DetailedGridItemsInfo {
	return DetailedGridItemsInfo{
		RowStart:    toOneIndexedGridLine(gridItem.rowIndexes.Start),
		RowEnd:      toOneIndexedGridLine(gridItem.rowIndexes.End),
		ColumnStart: toOneIndexedGridLine(gridItem.columnIndexes.Start),
		ColumnEnd:   toOneIndexedGridLine(gridItem.columnIndexes.End),
	}
}
