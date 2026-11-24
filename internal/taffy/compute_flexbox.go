package taffy

import (
	"math"
	"unsafe"
)

type flexItem struct {
	node NodeID

	order uint32

	size                  Size[Option[float32]]
	sizeStyle             Size[Dimension]
	minSize               Size[Option[float32]]
	maxSize               Size[Option[float32]]
	aspectRatio           Option[float32]
	alignSelf             AlignSelf
	overflow              Point[Overflow]
	contain               Contain
	scrollbarWidth        float32
	flexShrink            float32
	flexGrow              float32
	flexBasisIsDefinite   bool
	resolvedMinimumMainSz float32

	relativeInset Size[float32]
	margin        Rect[float32]
	marginIsAuto  Rect[bool]
	padding       Rect[float32]
	border        Rect[float32]

	flexBasis      float32
	innerFlexBasis float32
	violation      float32
	frozen         bool

	contentFlexFraction float32

	hypotheticalInnerSize Size[float32]
	hypotheticalOuterSize Size[float32]
	targetSize            Size[float32]
	outerTargetSize       Size[float32]

	baseline float32

	offsetMain  float32
	offsetCross float32

	oofCandidates OofCandidates
}

func (f *flexItem) isScrollContainer() bool {
	return f.overflow.X.IsScrollContainer() || f.overflow.Y.IsScrollContainer()
}

func (f *flexItem) participatesInBaselineAlignment(dir FlexDirection) bool {
	return f.alignSelf == AlignItemsBaseline &&
		!f.marginIsAuto.crossStart(dir) &&
		!f.marginIsAuto.crossEnd(dir)
}

type flexLine struct {
	items       []flexItem
	crossSize   float32
	offsetCross float32
}

type algoConstants struct {
	dir             FlexDirection
	layoutDirection Direction
	isRow           bool
	isColumn        bool
	isWrap          bool
	isWrapReverse   bool

	minSize           Size[Option[float32]]
	maxSize           Size[Option[float32]]
	border            Rect[float32]
	contentBoxInset   Rect[float32]
	scrollbarGutter   Point[float32]
	isScrollContainer bool
	gap               Size[float32]
	alignItems        AlignItems
	alignContent      AlignContent
	justifyContent    JustifyContent

	nodeOuterSize                     Size[Option[float32]]
	nodeInnerSize                     Size[Option[float32]]
	knownMainSizeIsDefinite           bool
	hasDefiniteMainSize               bool
	hasDefiniteCrossSize              bool
	crossAxisAvailableSpaceIsDefinite bool
	containerSize                     Size[float32]
	innerContainerSize                Size[float32]
}

func (c *algoConstants) dividedCrossSpace(crossAvailableSpace float32) float32 {
	return crossAvailableSpace
}

func treeCalc(tree LayoutPartialTree) func(unsafe.Pointer, float32) float32 {
	return func(val unsafe.Pointer, basis float32) float32 { return calc(tree, val, basis) }
}

var sizeAvailableSpaceMaxContent = Size[AvailableSpace]{Width: AvailableSpaceMaxContent, Height: AvailableSpaceMaxContent}

var sizeAvailableSpaceMinContent = Size[AvailableSpace]{Width: AvailableSpaceMinContent, Height: AvailableSpaceMinContent}

func someF32(v float32) Option[float32] { return some(v) }

func f32IsNormal(v float32) bool {
	bits := math.Float32bits(v) & 0x7f80_0000
	return bits != 0 && bits != 0x7f80_0000
}

func f32TotalCmp(a, b float32) int {
	ka := int32(math.Float32bits(a))
	kb := int32(math.Float32bits(b))
	ka ^= int32(uint32(ka>>31) >> 1)
	kb ^= int32(uint32(kb>>31) >> 1)
	switch {
	case ka < kb:
		return -1
	case ka > kb:
		return 1
	}
	return 0
}

// ComputeFlexboxLayout computes the layout of a box according to the flexbox algorithm.
func ComputeFlexboxLayout(tree LayoutFlexboxContainer, node NodeID, inputs LayoutInput) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	runMode := inputs.RunMode
	style := tree.GetFlexboxContainerStyle(node)
	calcFn := treeCalc(tree)

	contain := style.Contain()
	aspectRatio := style.AspectRatio()
	padding := RectResolveOrZeroOptionF32(style.Padding(), parentSize.Width, calcFn)
	border := RectResolveOrZeroOptionF32(style.Border(), parentSize.Width, calcFn)
	paddingBorderSum := padding.SumAxes().Add(border.SumAxes())
	boxSizingAdjustment := SizeF32Zero
	if style.BoxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSum
	}

	minSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MinSize(), parentSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	maxSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MaxSize(), parentSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	clampedStyleSize := SizeOptionF32None
	if inputs.SizingMode == SizingModeInherentSize {
		clampedStyleSize = SizeOptionF32MaybeClamp(
			SizeOptionF32MaybeAddF32(
				SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.Size(), parentSize, calcFn), aspectRatio),
				boxSizingAdjustment,
			),
			minSize,
			maxSize,
		)
	}

	minMaxDefiniteSize := minSize.ZipMap(maxSize, func(minV, maxV Option[float32]) Option[float32] {
		mn, mnok := minV.Get()
		mx, mxok := maxV.Get()
		if mnok && mxok && mx <= mn {
			return some(mn)
		}
		return none[float32]()
	})

	styledBasedKnownDimensions := SizeOptionOr(
		knownDimensions,
		SizeOptionF32MaybeMaxF32(SizeOptionOr(minMaxDefiniteSize, clampedStyleSize), paddingBorderSum),
	)

	if runMode == RunModeComputeSize {
		width, wok := styledBasedKnownDimensions.Width.Get()
		height, hok := styledBasedKnownDimensions.Height.Get()
		if wok && hok {
			return LayoutOutputFromOuterSize(Size[float32]{Width: width, Height: height})
		}

		if inputs.Axis == RequestedAxisHorizontal {
			if wok {
				return LayoutOutputFromOuterSize(Size[float32]{Width: width, Height: 0.0})
			}
		}
	}

	if runMode == RunModeComputeSize {
		width, wok := styledBasedKnownDimensions.Width.Get()
		height, hok := styledBasedKnownDimensions.Height.Get()
		if wok && hok {
			return LayoutOutputFromOuterSize(Size[float32]{Width: width, Height: height})
		}
	}

	knownDimensionsAreDefinite := inputs.KnownDimensionsAreDefinite.ZipMap(knownDimensions, func(isDefinite bool, knownDimension Option[float32]) bool {
		return isDefinite || knownDimension.IsNone()
	})

	preliminaryInputs := inputs
	preliminaryInputs.KnownDimensions = styledBasedKnownDimensions
	preliminaryInputs.KnownDimensionsAreDefinite = knownDimensionsAreDefinite
	output := computePreliminary(tree, node, preliminaryInputs)

	if contain.SuppressesBaseline() {
		output.Baselines = BaselinesNone
	}

	return output
}

func computePreliminary(tree LayoutFlexboxContainer, node NodeID, inputs LayoutInput) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	runMode := inputs.RunMode
	calcFn := treeCalc(tree)

	constants := computeConstants(
		tree,
		tree.GetFlexboxContainerStyle(node),
		knownDimensions,
		inputs.KnownDimensionsAreDefinite,
		parentSize,
		availableSpace,
	)

	flexItems := generateAnonymousFlexItems(tree, node, &constants)

	availableSpace = determineAvailableSpace(knownDimensions, availableSpace, &constants)

	determineFlexBaseSize(tree, &constants, availableSpace, flexItems)

	flexLines := collectFlexLines(&constants, availableSpace, flexItems)

	if innerMainSize, ok := constants.nodeInnerSize.main(constants.dir).Get(); ok {
		outerMainSize := innerMainSize + constants.contentBoxInset.mainAxisSum(constants.dir)
		constants.innerContainerSize.setMain(constants.dir, innerMainSize)
		constants.containerSize.setMain(constants.dir, outerMainSize)
	} else {
		determineContainerMainSize(tree, availableSpace, flexLines, &constants)
		constants.nodeInnerSize.setMain(constants.dir, some(constants.innerContainerSize.main(constants.dir)))
		constants.nodeOuterSize.setMain(constants.dir, some(constants.containerSize.main(constants.dir)))

		style := tree.GetFlexboxContainerStyle(node)
		innerContainerSize := constants.innerContainerSize.main(constants.dir)
		newGap := MaybeResolveF32(style.Gap().main(constants.dir), innerContainerSize, calcFn).UnwrapOr(0.0)
		constants.gap.setMain(constants.dir, newGap)
	}

	if runMode == RunModeComputeSize {
		var mainAxisIsRequestedAxis bool
		switch inputs.Axis {
		case RequestedAxisHorizontal:
			mainAxisIsRequestedAxis = constants.dir.isRow()
		case RequestedAxisVertical:
			mainAxisIsRequestedAxis = constants.dir.isColumn()
		case RequestedAxisBoth:
			mainAxisIsRequestedAxis = false
		}
		if mainAxisIsRequestedAxis {
			size := SizeF32Zero.withMain(constants.dir, constants.containerSize.main(constants.dir))
			return LayoutOutputFromOuterSize(size)
		}
	}

	for i := range flexLines {
		resolveFlexibleLengths(&flexLines[i], &constants)
	}

	for i := range flexLines {
		determineHypotheticalCrossSize(tree, &flexLines[i], &constants, availableSpace)
	}

	calculateChildrenBaseLines(tree, knownDimensions, availableSpace, flexLines, &constants)

	calculateCrossSize(flexLines, knownDimensions, &constants)

	handleAlignContentStretch(flexLines, knownDimensions, &constants)

	determineUsedCrossSize(tree, flexLines, &constants)

	distributeRemainingFreeSpace(flexLines, &constants)

	resolveCrossAxisAutoMargins(flexLines, &constants)

	totalLineCrossSize := determineContainerCrossSize(flexLines, knownDimensions, &constants)

	if runMode == RunModeComputeSize {
		return LayoutOutputFromOuterSize(constants.containerSize)
	}

	alignFlexLinesPerAlignContent(flexLines, &constants, totalLineCrossSize)

	inflowOverflowRect := finalLayoutPass(tree, flexLines, &constants)

	candidates := NewOofCandidates()
	collectOofCandidates(tree, node, &constants, flexLines, &candidates)

	gutterLeft, gutterRight := float32(0.0), constants.scrollbarGutter.X
	if constants.layoutDirection.IsRtl() {
		gutterLeft, gutterRight = constants.scrollbarGutter.X, 0.0
	}
	absolutePositionInset := constants.border.Add(Rect[float32]{
		Left:   gutterLeft,
		Right:  gutterRight,
		Top:    0.0,
		Bottom: constants.scrollbarGutter.Y,
	})
	absolutePositionArea := constants.containerSize.Sub(absolutePositionInset.SumAxes())
	absolutePositionOffset := Point[float32]{X: absolutePositionInset.Left, Y: absolutePositionInset.Top}

	length := tree.ChildCount(node)
	for order := 0; order < length; order++ {
		child := tree.GetChildID(node, order)
		if tree.GetFlexboxChildStyle(child).BoxGenerationMode() == BoxGenerationModeNone {
			layout := LayoutWithOrder(uint32(order))
			tree.SetUnroundedLayout(child, &layout)
			performChildLayout(
				tree,
				child,
				SizeOptionF32None,
				SizeOptionF32None,
				sizeAvailableSpaceMaxContent,
				SizingModeContentSize,
				LineBoolFalse,
			)
		}
	}

	var firstLine *flexLine
	if len(flexLines) > 0 {
		if constants.isWrapReverse {
			firstLine = &flexLines[len(flexLines)-1]
		} else {
			firstLine = &flexLines[0]
		}
	}
	firstVerticalBaseline := none[float32]()
	if firstLine != nil {
		items := firstLine.items
		if constants.isColumn {
			if len(items) > 0 {
				if constants.dir.isReverse() {
					firstVerticalBaseline = some(items[len(items)-1].baseline)
				} else {
					firstVerticalBaseline = some(items[0].baseline)
				}
			}
		} else {
			found := false
			for i := range items {
				if items[i].participatesInBaselineAlignment(constants.dir) {
					firstVerticalBaseline = some(items[i].baseline)
					found = true
					break
				}
			}
			if !found && len(items) > 0 {
				firstVerticalBaseline = some(items[0].baseline)
			}
		}
	}

	output := LayoutOutputFromSizesAndBaselines(
		constants.containerSize,
		inflowOverflowRect,
		BaselinesFromFirst(firstVerticalBaseline),
	)
	output.OofCandidates = candidates
	output.OofPositioningArea = some(OofPositioningArea{Size: absolutePositionArea, Offset: absolutePositionOffset})
	return output
}

func resolveNormalAlignment(alignment AlignItems) AlignItems {
	if alignment.keyword == AlignItemsKeywordNormal {
		return AlignItemsStretch
	}
	return alignment
}

func computeConstants(
	tree LayoutFlexboxContainer,
	style FlexboxContainerStyle,
	knownDimensions Size[Option[float32]],
	knownDimensionsAreDefinite Size[bool],
	parentSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
) algoConstants {
	calcFn := treeCalc(tree)
	dir := style.FlexDirection()
	isRow := dir.isRow()
	isColumn := dir.isColumn()
	flexWrap := style.FlexWrap()
	isWrap := flexWrap.isMultiLine()
	isWrapReverse := flexWrap.isReverse()

	aspectRatio := style.AspectRatio()
	padding := RectResolveOrZeroOptionF32(style.Padding(), parentSize.Width, calcFn)
	border := RectResolveOrZeroOptionF32(style.Border(), parentSize.Width, calcFn)
	paddingBorderSum := padding.SumAxes().Add(border.SumAxes())
	boxSizingAdjustment := SizeF32Zero
	if style.BoxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSum
	}

	alignItems := resolveNormalAlignment(style.AlignItems())
	alignContent := style.AlignContent()
	if alignContent.keyword == AlignContentKeywordNormal {
		alignContent = AlignContentStretch
	}
	justifyContent := style.JustifyContent()
	if justifyContent.keyword == AlignContentKeywordNormal {
		justifyContent = AlignContentFlexStart
	}
	layoutDirection := style.Direction()

	scrollbarGutter := style.Overflow().Transpose().Map(func(overflow Overflow) float32 {
		if overflow == OverflowScroll {
			return style.ScrollbarWidth()
		}
		return 0.0
	})
	overflow := style.Overflow()
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()
	contentBoxInset := padding.Add(border)
	contentBoxInset.Bottom += scrollbarGutter.Y

	switch layoutDirection {
	case DirectionLtr:
		contentBoxInset.Right += scrollbarGutter.X
	case DirectionRtl:
		contentBoxInset.Left += scrollbarGutter.X
	}

	nodeOuterSize := knownDimensions
	nodeInnerSize := SizeOptionF32MaybeSubF32(nodeOuterSize, contentBoxInset.SumAxes())
	knownMainSizeIsDefinite := knownDimensionsAreDefinite.main(dir)
	hasDefiniteMainSize := knownMainSizeIsDefinite && knownDimensions.main(dir).IsSome()
	hasDefiniteCrossSize := knownDimensionsAreDefinite.cross(dir) && knownDimensions.cross(dir).IsSome()
	crossAxisAvailableSpaceIsDefinite := hasDefiniteCrossSize || availableSpace.cross(dir).IsDefinite()
	gap := SizeResolveOrZero(style.Gap(), SizeOptionOr(nodeInnerSize, Size[Option[float32]]{Width: some[float32](0.0), Height: some[float32](0.0)}), calcFn)

	return algoConstants{
		dir:             dir,
		layoutDirection: layoutDirection,
		isRow:           isRow,
		isColumn:        isColumn,
		isWrap:          isWrap,
		isWrapReverse:   isWrapReverse,
		minSize: SizeOptionF32MaybeAddF32(
			SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MinSize(), parentSize, calcFn), aspectRatio),
			boxSizingAdjustment,
		),
		maxSize: SizeOptionF32MaybeAddF32(
			SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MaxSize(), parentSize, calcFn), aspectRatio),
			boxSizingAdjustment,
		),
		border:                            border,
		gap:                               gap,
		contentBoxInset:                   contentBoxInset,
		scrollbarGutter:                   scrollbarGutter,
		isScrollContainer:                 isScrollContainer,
		alignItems:                        alignItems,
		alignContent:                      alignContent,
		justifyContent:                    justifyContent,
		nodeOuterSize:                     nodeOuterSize,
		nodeInnerSize:                     nodeInnerSize,
		knownMainSizeIsDefinite:           knownMainSizeIsDefinite,
		hasDefiniteMainSize:               hasDefiniteMainSize,
		hasDefiniteCrossSize:              hasDefiniteCrossSize,
		crossAxisAvailableSpaceIsDefinite: crossAxisAvailableSpaceIsDefinite,
		containerSize:                     SizeF32Zero,
		innerContainerSize:                SizeF32Zero,
	}
}

func resolveRelativeInset(inset Rect[Option[float32]], constants *algoConstants) Size[float32] {
	direction := constants.dir
	isRtlRow := direction.isRow() && constants.layoutDirection.IsRtl()
	isRtlColumn := direction.isColumn() && constants.layoutDirection.IsRtl()
	negate := func(o Option[float32]) Option[float32] {
		if v, ok := o.Get(); ok {
			return some(-v)
		}
		return o
	}
	var mainV float32
	if isRtlRow {
		mainV = inset.mainEnd(direction).Or(negate(inset.mainStart(direction))).UnwrapOr(0.0)
	} else {
		mainV = inset.mainStart(direction).Or(negate(inset.mainEnd(direction))).UnwrapOr(0.0)
	}
	var crossV float32
	if isRtlColumn {
		crossV = negate(inset.crossEnd(direction)).Or(inset.crossStart(direction)).UnwrapOr(0.0)
	} else {
		crossV = inset.crossStart(direction).Or(negate(inset.crossEnd(direction))).UnwrapOr(0.0)
	}
	out := SizeF32Zero
	out.setMain(direction, mainV)
	out.setCross(direction, crossV)
	return out
}

func generateAnonymousFlexItems(tree LayoutFlexboxContainer, node NodeID, constants *algoConstants) []flexItem {
	calcFn := treeCalc(tree)
	percentResolutionSize := constants.nodeInnerSize
	if !constants.knownMainSizeIsDefinite {
		percentResolutionSize = constants.nodeInnerSize.withMain(constants.dir, none[float32]())
	}

	var items []flexItem
	index := 0
	for child := range tree.ChildIds(node) {
		childStyle := tree.GetFlexboxChildStyle(child)
		idx := index
		index++
		if childStyle.Position().IsOutOfFlow() {
			continue
		}
		if childStyle.BoxGenerationMode() == BoxGenerationModeNone {
			continue
		}
		aspectRatio := childStyle.AspectRatio()
		padding := RectResolveOrZeroOptionF32(childStyle.Padding(), constants.nodeInnerSize.Width, calcFn)
		border := RectResolveOrZeroOptionF32(childStyle.Border(), constants.nodeInnerSize.Width, calcFn)
		pbSum := padding.Add(border).SumAxes()
		boxSizingAdjustment := SizeF32Zero
		if childStyle.BoxSizing() == BoxSizingContentBox {
			boxSizingAdjustment = pbSum
		}
		relativeInset := SizeF32Zero
		if childStyle.Position() == PositionRelative {
			inset := childStyle.Inset().zipSize(constants.nodeInnerSize, func(p LengthPercentageAuto, s Option[float32]) Option[float32] {
				return p.MaybeResolve(s, calcFn)
			})
			relativeInset = resolveRelativeInset(inset, constants)
		}
		items = append(items, flexItem{
			node:  child,
			order: uint32(idx),
			size: SizeOptionF32MaybeAddF32(
				SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(childStyle.Size(), percentResolutionSize, calcFn), aspectRatio),
				boxSizingAdjustment,
			),
			sizeStyle:     childStyle.Size(),
			minSize:       SizeOptionF32MaybeAddF32(SizeMaybeResolve(childStyle.MinSize(), percentResolutionSize, calcFn), boxSizingAdjustment),
			maxSize:       SizeOptionF32MaybeAddF32(SizeMaybeResolve(childStyle.MaxSize(), percentResolutionSize, calcFn), boxSizingAdjustment),
			aspectRatio:   aspectRatio,
			relativeInset: relativeInset,
			margin:        RectResolveOrZeroOptionF32(childStyle.Margin(), constants.nodeInnerSize.Width, calcFn),
			marginIsAuto:  childStyle.Margin().Map(LengthPercentageAuto.IsAuto),
			padding:       RectResolveOrZeroOptionF32(childStyle.Padding(), constants.nodeInnerSize.Width, calcFn),
			border:        RectResolveOrZeroOptionF32(childStyle.Border(), constants.nodeInnerSize.Width, calcFn),
			alignSelf: resolveNormalAlignment(childStyle.AlignSelf().UnwrapOr(constants.alignItems)).
				ResolveSelfRelative(childStyle.Direction(), constants.layoutDirection, constants.isColumn),
			overflow:       childStyle.Overflow(),
			contain:        childStyle.Contain(),
			scrollbarWidth: childStyle.ScrollbarWidth(),
			flexGrow:       childStyle.FlexGrow(),
			flexShrink:     childStyle.FlexShrink(),
			oofCandidates:  OofCandidatesNone,
		})
	}
	return items
}

func determineAvailableSpace(
	knownDimensions Size[Option[float32]],
	outerAvailableSpace Size[AvailableSpace],
	constants *algoConstants,
) Size[AvailableSpace] {
	var width AvailableSpace
	if nodeWidth, ok := knownDimensions.Width.Get(); ok {
		width = AvailableSpaceDefinite(f32Max(nodeWidth-constants.contentBoxInset.HorizontalAxisSum(), 0.0))
	} else {
		width = outerAvailableSpace.Width.MaybeSubF32(constants.contentBoxInset.HorizontalAxisSum()).MaybeMaxF32(0.0)
	}

	var height AvailableSpace
	if nodeHeight, ok := knownDimensions.Height.Get(); ok {
		height = AvailableSpaceDefinite(f32Max(nodeHeight-constants.contentBoxInset.VerticalAxisSum(), 0.0))
	} else {
		height = outerAvailableSpace.Height.MaybeSubF32(constants.contentBoxInset.VerticalAxisSum()).MaybeMaxF32(0.0)
	}

	return Size[AvailableSpace]{Width: width, Height: height}
}

func determineFlexBaseSize(
	tree LayoutFlexboxContainer,
	constants *algoConstants,
	availableSpace Size[AvailableSpace],
	flexItems []flexItem,
) {
	calcFn := treeCalc(tree)
	dir := constants.dir

	for i := range flexItems {
		child := &flexItems[i]
		childStyle := tree.GetFlexboxChildStyle(child.node)

		crossAxisParentSize := constants.nodeInnerSize.cross(dir)
		childParentSize := SizeOptionF32FromCross(dir, crossAxisParentSize)

		transferredMinSize := SizeOptionF32MaybeApplyAspectRatio(child.minSize, child.aspectRatio)
		transferredMaxSize := SizeOptionF32MaybeApplyAspectRatio(child.maxSize, child.aspectRatio)
		childMinCross := transferredMinSize.cross(dir)
		childMaxCross := transferredMaxSize.cross(dir)

		var crossAxisAvailableSpace AvailableSpace
		crossAvail := availableSpace.cross(dir)
		switch crossAvail.Kind() {
		case AvailableSpaceKindDefinite:
			val := crossAvail.Unwrap()
			crossAxisAvailableSpace = AvailableSpaceDefinite(F32MaybeClamp(
				f32Max(constants.dividedCrossSpace(crossAxisParentSize.UnwrapOr(val))-child.margin.crossAxisSum(dir), 0.0),
				childMinCross,
				childMaxCross,
			))
		case AvailableSpaceKindMinContent:
			if mn, ok := childMinCross.Get(); ok {
				crossAxisAvailableSpace = AvailableSpaceDefinite(mn)
			} else {
				crossAxisAvailableSpace = AvailableSpaceMinContent
			}
		case AvailableSpaceKindMaxContent:
			if mx, ok := childMaxCross.Get(); ok {
				crossAxisAvailableSpace = AvailableSpaceDefinite(mx)
			} else {
				crossAxisAvailableSpace = AvailableSpaceMaxContent
			}
		}

		childCrossSizeIsDefinite := child.size.cross(dir).IsSome()
		childKnownDimensions := child.size.withMain(dir, none[float32]())
		childKnownDimensions.setCross(
			dir,
			OptionF32MaybeClamp(childKnownDimensions.cross(dir), transferredMinSize.cross(dir), transferredMaxSize.cross(dir)),
		)
		if child.alignSelf == AlignItemsStretch &&
			!child.marginIsAuto.crossStart(constants.dir) &&
			!child.marginIsAuto.crossEnd(constants.dir) &&
			childKnownDimensions.cross(dir).IsNone() {
			childKnownDimensions.setCross(dir, crossAxisAvailableSpace.IntoOption())
			childCrossSizeIsDefinite = !constants.isWrap && constants.hasDefiniteCrossSize && crossAxisParentSize.IsSome()
		}

		containerWidth := constants.nodeInnerSize.Width
		boxSizingAdjustment := SizeF32Zero
		if childStyle.BoxSizing() == BoxSizingContentBox {
			padding := RectResolveOrZeroOptionF32(childStyle.Padding(), containerWidth, calcFn)
			border := RectResolveOrZeroOptionF32(childStyle.Border(), containerWidth, calcFn)
			boxSizingAdjustment = padding.Add(border).SumAxes()
		}
		boxSizingAdjustmentMain := boxSizingAdjustment.main(dir)
		percentResolutionMainSize := none[float32]()
		if constants.knownMainSizeIsDefinite {
			percentResolutionMainSize = constants.nodeInnerSize.main(dir)
		}
		flexBasisStyle := childStyle.FlexBasis()
		flexBasis := OptionF32MaybeAddF32(flexBasisStyle.MaybeResolve(percentResolutionMainSize, calcFn), boxSizingAdjustmentMain)

		child.flexBasis = func() float32 {
			mainSize := child.size.main(dir)
			mainStretchSize := OptionF32MaybeMaxF32(OptionF32MaybeSubF32(percentResolutionMainSize, child.margin.mainAxisSum(dir)), 0.0)

			keywordMainAvailableSpace := none[AvailableSpace]()
			if flexBasisStyle.IsContent() {
				keywordMainAvailableSpace = none[AvailableSpace]()
			} else if flexBasisStyle.IsSizingKeyword() {
				if res, ok := resolveSizingKeyword(flexBasisStyle, mainStretchSize, percentResolutionMainSize, calcFn).Get(); ok {
					if res.kind == sizingKeywordResolutionKindExact {
						child.flexBasisIsDefinite = true
						return res.exact
					}
					keywordMainAvailableSpace = some(res.measure)
				}
			} else {
				candidate := flexBasis
				if candidate.IsNone() && flexBasisStyle.IsAuto() {
					candidate = mainSize
				}
				if fb, ok := candidate.Get(); ok {
					child.flexBasisIsDefinite = true
					return fb
				}

				if flexBasisStyle.IsAuto() {
					if res, ok := resolveSizingKeyword(child.sizeStyle.main(dir), mainStretchSize, percentResolutionMainSize, calcFn).Get(); ok {
						if res.kind == sizingKeywordResolutionKindExact {
							child.flexBasisIsDefinite = true
							return res.exact
						}
						keywordMainAvailableSpace = some(res.measure)
					}
				}
			}

			if childCrossSizeIsDefinite {
				ratio, rok := child.aspectRatio.Get()
				crossV, cok := childKnownDimensions.cross(dir).Get()
				if rok && cok {
					child.flexBasisIsDefinite = true
					if dir.isRow() {
						return crossV * ratio
					}
					return crossV / ratio
				}
			}

			defaultMain := AvailableSpaceMaxContent
			if availableSpace.main(dir) == AvailableSpaceMinContent {
				defaultMain = AvailableSpaceMinContent
			}
			childAvailableSpace := sizeAvailableSpaceMaxContent.
				withMain(dir, keywordMainAvailableSpace.UnwrapOr(defaultMain)).
				withCross(dir, crossAxisAvailableSpace)

			return measureChildSize(
				tree,
				child.node,
				childKnownDimensions,
				childParentSize,
				childAvailableSpace,
				SizingModeContentSize,
				dir.mainAxis(),
				LineBoolFalse,
			)
		}()

		paddingBorderSum := child.padding.mainAxisSum(constants.dir) + child.border.mainAxisSum(constants.dir)
		child.flexBasis = f32Max(child.flexBasis, paddingBorderSum)

		child.innerFlexBasis = child.flexBasis - child.padding.mainAxisSum(constants.dir) - child.border.mainAxisSum(constants.dir)

		paddingBorderAxesSums := child.padding.Add(child.border).SumAxes().Map(someF32)

		styleMinMainSize := SizeOptionOr(child.minSize, SizeFromPoint(child.overflow.Map(Overflow.maybeIntoAutomaticMinSize))).main(dir)

		if v, ok := styleMinMainSize.Get(); ok {
			child.resolvedMinimumMainSz = v
		} else {
			childAvailableSpace := sizeAvailableSpaceMinContent.withCross(dir, crossAxisAvailableSpace)
			minContentMainSize := measureChildSize(
				tree,
				child.node,
				childKnownDimensions,
				childParentSize,
				childAvailableSpace,
				SizingModeContentSize,
				dir.mainAxis(),
				LineBoolFalse,
			)

			clampedMinContentSize := F32MaybeMin(F32MaybeMin(minContentMainSize, child.size.main(dir)), transferredMaxSize.main(dir))
			child.resolvedMinimumMainSz = F32MaybeMax(clampedMinContentSize, paddingBorderAxesSums.main(dir))
		}

		hypotheticalInnerMinMain := F32MaybeMax(
			F32MaybeMax(child.resolvedMinimumMainSz, transferredMinSize.main(constants.dir)),
			paddingBorderAxesSums.main(constants.dir),
		)
		hypotheticalInnerSize := F32MaybeClamp(child.flexBasis, some(hypotheticalInnerMinMain), transferredMaxSize.main(constants.dir))
		hypotheticalOuterSize := hypotheticalInnerSize + child.margin.mainAxisSum(constants.dir)

		child.hypotheticalInnerSize.setMain(constants.dir, hypotheticalInnerSize)
		child.hypotheticalOuterSize.setMain(constants.dir, hypotheticalOuterSize)
	}
}

func singleFlexLine(flexItems []flexItem) []flexLine {
	lines := newVecWithCapacity[flexLine](1)
	return append(lines, flexLine{items: flexItems, crossSize: 0.0, offsetCross: 0.0})
}

func collectFlexLines(constants *algoConstants, availableSpace Size[AvailableSpace], flexItems []flexItem) []flexLine {
	if !constants.isWrap || !constants.knownMainSizeIsDefinite {
		return singleFlexLine(flexItems)
	}

	var mainAxisAvailableSpace AvailableSpace
	if maxSize, ok := constants.maxSize.main(constants.dir).Get(); ok {
		available := availableSpace.main(constants.dir).IntoOption().UnwrapOr(maxSize)
		if !constants.hasDefiniteMainSize {
			available = f32Min(available, maxSize)
		}
		mainAxisAvailableSpace = AvailableSpaceDefinite(F32MaybeMax(available, constants.minSize.main(constants.dir)))
	} else if !constants.dir.isRow() &&
		!constants.hasDefiniteMainSize &&
		availableSpace.main(constants.dir).IsDefinite() {
		mainAxisAvailableSpace = AvailableSpaceMaxContent
	} else {
		mainAxisAvailableSpace = availableSpace.main(constants.dir)
	}

	switch mainAxisAvailableSpace.Kind() {
	case AvailableSpaceKindMaxContent:
		return singleFlexLine(flexItems)
	case AvailableSpaceKindMinContent:
		lines := newVecWithCapacity[flexLine](len(flexItems))
		for i := range flexItems {
			lines = append(lines, flexLine{items: flexItems[i : i+1 : i+1], crossSize: 0.0, offsetCross: 0.0})
		}
		return lines
	default:
		available := mainAxisAvailableSpace.Unwrap()
		lines := newVecWithCapacity[flexLine](1)
		items := flexItems
		mainAxisGap := constants.gap.main(constants.dir)

		for len(items) > 0 {
			lineLength := float32(0.0)
			index := len(items)
			for idx := range items {
				gapContribution := mainAxisGap
				if idx == 0 {
					gapContribution = 0.0
				}
				lineLength += items[idx].hypotheticalOuterSize.main(constants.dir) + gapContribution
				if lineLength > available && idx != 0 {
					index = idx
					break
				}
			}

			lines = append(lines, flexLine{items: items[:index:index], crossSize: 0.0, offsetCross: 0.0})
			items = items[index:]
		}
		return lines
	}
}

func itemKnownDimensionDefiniteness(constants *algoConstants, item *flexItem) Size[bool] {
	dir := constants.dir
	mainIsDefinite := constants.hasDefiniteMainSize || item.flexBasisIsDefinite

	hasCrossAutoMargins := item.marginIsAuto.crossStart(dir) || item.marginIsAuto.crossEnd(dir)
	crossSize := item.sizeStyle.cross(dir)
	isStretched := !hasCrossAutoMargins &&
		(crossSize.IsStretch() || (item.alignSelf == AlignItemsStretch && crossSize.IsAuto()))
	crossIsDefinite := isStretched ||
		item.size.cross(dir).IsSome() ||
		(!dir.isRow() && constants.crossAxisAvailableSpaceIsDefinite)

	return Size[bool]{Width: true, Height: true}.withMain(dir, mainIsDefinite).withCross(dir, crossIsDefinite)
}

func determineContainerMainSize(
	tree LayoutFlexboxContainer,
	availableSpace Size[AvailableSpace],
	lines []flexLine,
	constants *algoConstants,
) {
	dir := constants.dir
	mainContentBoxInset := constants.contentBoxInset.mainAxisSum(constants.dir)

	outerMainSize, known := constants.nodeOuterSize.main(constants.dir).Get()
	if !known {
		mainAvail := availableSpace.main(dir)
		switch {
		case mainAvail.Kind() == AvailableSpaceKindDefinite:
			mainAxisAvailableSpace := mainAvail.Unwrap()
			mainAxisGap := constants.gap.main(constants.dir)
			itemMainLength := func(child *flexItem) float32 {
				paddingBorderSum := child.padding.Add(child.border).mainAxisSum(constants.dir)
				return f32Max(
					F32MaybeMax(child.flexBasis, child.minSize.main(constants.dir))+child.margin.mainAxisSum(constants.dir),
					paddingBorderSum,
				)
			}
			longestLineLength := float32(0.0)
			for li := range lines {
				line := &lines[li]
				lineMainAxisGap := sumAxisGaps(mainAxisGap, len(line.items))
				totalTargetSize := float32(0.0)
				for i := range line.items {
					totalTargetSize += itemMainLength(&line.items[i])
				}
				length := totalTargetSize + lineMainAxisGap
				if li == 0 || f32TotalCmp(length, longestLineLength) >= 0 {
					longestLineLength = length
				}
			}
			size := longestLineLength + mainContentBoxInset

			if len(lines) > 1 {
				outerMainSize = f32Max(size, mainAxisAvailableSpace)
			} else {
				outerMainSize = size
			}
		case mainAvail.Kind() == AvailableSpaceKindMinContent && constants.isWrap:
			longestLineLength := float32(0.0)
			for li := range lines {
				line := &lines[li]
				lineMainAxisGap := sumAxisGaps(constants.gap.main(constants.dir), len(line.items))
				totalTargetSize := float32(0.0)
				for i := range line.items {
					child := &line.items[i]
					paddingBorderSum := child.padding.Add(child.border).mainAxisSum(constants.dir)
					totalTargetSize += f32Max(
						F32MaybeMax(child.flexBasis, child.minSize.main(constants.dir))+child.margin.mainAxisSum(constants.dir),
						paddingBorderSum,
					)
				}
				length := totalTargetSize + lineMainAxisGap
				if li == 0 || f32TotalCmp(length, longestLineLength) >= 0 {
					longestLineLength = length
				}
			}
			outerMainSize = longestLineLength + mainContentBoxInset
		default:
			mainSize := float32(0.0)

			for li := range lines {
				line := &lines[li]
				for i := range line.items {
					item := &line.items[i]
					styleMin := item.minSize.main(constants.dir)
					stylePreferred := item.size.main(constants.dir)
					styleMax := item.maxSize.main(constants.dir)

					var clampingBasis Option[float32]
					if constants.isRow {
						clampingBasis = OptionF32MaybeMax(some(item.flexBasis), stylePreferred)
					} else {
						clampingBasis = some(item.flexBasis)
					}
					flexBasisMin := none[float32]()
					if item.flexShrink == 0.0 {
						flexBasisMin = clampingBasis
					}
					flexBasisMax := none[float32]()
					if item.flexGrow == 0.0 {
						flexBasisMax = clampingBasis
					}

					minMainSize := f32Max(
						OptionF32MaybeMax(styleMin, flexBasisMin).Or(flexBasisMin).UnwrapOr(item.resolvedMinimumMainSz),
						item.resolvedMinimumMainSz,
					)
					maxMainSize := OptionF32MaybeMin(styleMax, flexBasisMax).Or(flexBasisMax).UnwrapOr(float32(math.Inf(1)))

					var contentContribution float32
					pref, hasPref := stylePreferred.Get()
					switch {
					case hasPref && (maxMainSize <= minMainSize || maxMainSize <= pref):
						contentContribution = f32Max(f32Min(pref, maxMainSize), minMainSize) + item.margin.mainAxisSum(constants.dir)
					case maxMainSize <= minMainSize:
						contentContribution = minMainSize + item.margin.mainAxisSum(constants.dir)
					case item.isScrollContainer():
						contentContribution = item.flexBasis + item.margin.mainAxisSum(constants.dir)
					case hasPref:
						itemPbMain := item.padding.mainAxisSum(constants.dir) + item.border.mainAxisSum(constants.dir)
						innerMainSize := f32Max(pref, itemPbMain)
						if constants.isRow {
							contentContribution = F32MaybeClamp(innerMainSize+item.margin.mainAxisSum(constants.dir), styleMin, styleMax)
						} else {
							contentContribution = F32MaybeClamp(f32Max(innerMainSize, item.flexBasis)+item.margin.mainAxisSum(constants.dir), styleMin, styleMax)
						}
					default:
						crossAxisParentSize := constants.nodeInnerSize.cross(dir)

						childMinCross := item.minSize.cross(dir)
						childMaxCross := item.maxSize.cross(dir)
						crossAxisAvailableSpace := availableSpace.cross(dir).
							MapDefiniteValue(func(val float32) float32 {
								return f32Max(constants.dividedCrossSpace(crossAxisParentSize.UnwrapOr(val))-item.margin.crossAxisSum(dir), 0.0)
							}).
							MaybeClamp(childMinCross, childMaxCross)

						childAvailableSpace := availableSpace.withCross(dir, crossAxisAvailableSpace)

						childKnownDimensions := item.size.withMain(dir, none[float32]())
						transferredMinSize := SizeOptionF32MaybeApplyAspectRatio(item.minSize, item.aspectRatio)
						transferredMaxSize := SizeOptionF32MaybeApplyAspectRatio(item.maxSize, item.aspectRatio)
						childKnownDimensions.setCross(
							dir,
							OptionF32MaybeClamp(childKnownDimensions.cross(dir), transferredMinSize.cross(dir), transferredMaxSize.cross(dir)),
						)
						if item.alignSelf == AlignItemsStretch && childKnownDimensions.cross(dir).IsNone() {
							childKnownDimensions.setCross(dir, crossAxisAvailableSpace.IntoOption())
						}

						measuredMainSize := measureChildSize(
							tree,
							item.node,
							childKnownDimensions,
							constants.nodeInnerSize,
							childAvailableSpace,
							SizingModeContentSize,
							dir.mainAxis(),
							LineBoolFalse,
						)

						transferredMainSize := none[float32]()
						ratio, rok := item.aspectRatio.Get()
						crossV, cok := childKnownDimensions.cross(dir).Get()
						if rok && cok {
							if constants.isRow {
								transferredMainSize = some(crossV * ratio)
							} else {
								transferredMainSize = some(crossV / ratio)
							}
						}

						innerMainSize := F32MaybeMax(measuredMainSize, transferredMainSize)

						if constants.isRow {
							contentContribution = F32MaybeClamp(innerMainSize+item.margin.mainAxisSum(constants.dir), styleMin, styleMax)
						} else {
							contentContribution = F32MaybeClamp(f32Max(innerMainSize, item.flexBasis)+item.margin.mainAxisSum(constants.dir), styleMin, styleMax)
						}
					}
					diff := contentContribution - item.flexBasis
					switch {
					case diff > 0.0:
						item.contentFlexFraction = diff / f32Max(1.0, item.flexGrow)
					case diff < 0.0:
						scaledShrinkFactor := f32Max(1.0, item.flexShrink) * item.innerFlexBasis
						item.contentFlexFraction = diff / scaledShrinkFactor
					default:
						item.contentFlexFraction = 0.0
					}
				}

				itemMainSizeSum := float32(0.0)
				for i := range line.items {
					item := &line.items[i]
					flexFraction := item.contentFlexFraction

					var flexContribution float32
					switch {
					case item.contentFlexFraction > 0.0:
						flexContribution = float32(f32Max(1.0, item.flexGrow) * flexFraction)
					case item.contentFlexFraction < 0.0:
						scaledShrinkFactor := f32Max(1.0, item.flexShrink) * item.innerFlexBasis
						if scaledShrinkFactor == 0.0 {
							flexContribution = 0.0
						} else {
							flexContribution = float32(scaledShrinkFactor * flexFraction)
						}
					default:
						flexContribution = 0.0
					}
					size := item.flexBasis + flexContribution
					item.outerTargetSize.setMain(constants.dir, size)
					item.targetSize.setMain(constants.dir, size)
					itemMainSizeSum += size
				}

				gapSum := sumAxisGaps(constants.gap.main(constants.dir), len(line.items))
				mainSize = f32Max(mainSize, itemMainSizeSum+gapSum)
			}

			outerMainSize = mainSize + mainContentBoxInset
		}
	}

	outerMainSize = f32Max(
		F32MaybeClamp(outerMainSize, constants.minSize.main(constants.dir), constants.maxSize.main(constants.dir)),
		mainContentBoxInset-constants.scrollbarGutter.main(constants.dir),
	)

	innerMainSize := f32Max(outerMainSize-mainContentBoxInset, 0.0)
	constants.containerSize.setMain(constants.dir, outerMainSize)
	constants.innerContainerSize.setMain(constants.dir, innerMainSize)
	constants.nodeInnerSize.setMain(constants.dir, some(innerMainSize))
}

func resolveFlexibleLengths(line *flexLine, constants *algoConstants) {
	totalMainAxisGap := sumAxisGaps(constants.gap.main(constants.dir), len(line.items))

	totalHypotheticalOuterMainSize := float32(0.0)
	for i := range line.items {
		totalHypotheticalOuterMainSize += line.items[i].hypotheticalOuterSize.main(constants.dir)
	}
	usedFlexFactor := totalMainAxisGap + totalHypotheticalOuterMainSize
	growing := usedFlexFactor < constants.nodeInnerSize.main(constants.dir).UnwrapOr(0.0)
	shrinking := usedFlexFactor > constants.nodeInnerSize.main(constants.dir).UnwrapOr(0.0)
	exactlySized := !growing && !shrinking

	for i := range line.items {
		child := &line.items[i]
		innerTargetSize := child.hypotheticalInnerSize.main(constants.dir)
		child.targetSize.setMain(constants.dir, innerTargetSize)

		if exactlySized ||
			(child.flexGrow == 0.0 && child.flexShrink == 0.0) ||
			(growing && child.flexBasis > child.hypotheticalInnerSize.main(constants.dir)) ||
			(shrinking && child.flexBasis < child.hypotheticalInnerSize.main(constants.dir)) {
			child.frozen = true
			outerTargetSize := innerTargetSize + child.margin.mainAxisSum(constants.dir)
			child.outerTargetSize.setMain(constants.dir, outerTargetSize)
		}
	}

	if exactlySized {
		return
	}

	usedSpaceOf := func() float32 {
		sum := float32(0.0)
		for i := range line.items {
			child := &line.items[i]
			if child.frozen {
				sum += child.outerTargetSize.main(constants.dir)
			} else {
				sum += child.flexBasis + child.margin.mainAxisSum(constants.dir)
			}
		}
		return totalMainAxisGap + sum
	}

	usedSpace := usedSpaceOf()

	initialFreeSpace := OptionF32MaybeSubF32(constants.nodeInnerSize.main(constants.dir), usedSpace).UnwrapOr(0.0)

	for {
		allFrozen := true
		for i := range line.items {
			if !line.items[i].frozen {
				allFrozen = false
				break
			}
		}
		if allFrozen {
			break
		}

		usedSpace := usedSpaceOf()

		var unfrozen []*flexItem
		for i := range line.items {
			if !line.items[i].frozen {
				unfrozen = append(unfrozen, &line.items[i])
			}
		}

		sumFlexGrow, sumFlexShrink := float32(0.0), float32(0.0)
		for _, item := range unfrozen {
			sumFlexGrow += item.flexGrow
			sumFlexShrink += item.flexShrink
		}

		var freeSpace float32
		switch {
		case growing && sumFlexGrow < 1.0:
			freeSpace = F32MaybeMin(
				float32(initialFreeSpace*sumFlexGrow)-totalMainAxisGap,
				OptionF32MaybeSubF32(constants.nodeInnerSize.main(constants.dir), usedSpace),
			)
		case shrinking && sumFlexShrink < 1.0:
			freeSpace = F32MaybeMax(
				float32(initialFreeSpace*sumFlexShrink)-totalMainAxisGap,
				OptionF32MaybeSubF32(constants.nodeInnerSize.main(constants.dir), usedSpace),
			)
		default:
			freeSpace = OptionF32MaybeSubF32(constants.nodeInnerSize.main(constants.dir), usedSpace).
				UnwrapOr(usedFlexFactor - usedSpace)
		}

		if f32IsNormal(freeSpace) {
			if growing && sumFlexGrow > 0.0 {
				for _, child := range unfrozen {
					child.targetSize.setMain(constants.dir, child.flexBasis+float32(freeSpace*(child.flexGrow/sumFlexGrow)))
				}
			} else if shrinking && sumFlexShrink > 0.0 {
				sumScaledShrinkFactor := float32(0.0)
				for _, child := range unfrozen {
					sumScaledShrinkFactor += float32(child.innerFlexBasis * child.flexShrink)
				}

				if sumScaledShrinkFactor > 0.0 {
					for _, child := range unfrozen {
						scaledShrinkFactor := child.innerFlexBasis * child.flexShrink
						child.targetSize.setMain(
							constants.dir,
							child.flexBasis+float32(freeSpace*(scaledShrinkFactor/sumScaledShrinkFactor)),
						)
					}
				}
			}
		}

		totalViolation := float32(0.0)
		for _, child := range unfrozen {
			resolvedMinMain := some(child.resolvedMinimumMainSz)
			maxMain := child.maxSize.main(constants.dir)
			clamped := f32Max(F32MaybeClamp(child.targetSize.main(constants.dir), resolvedMinMain, maxMain), 0.0)
			child.violation = clamped - child.targetSize.main(constants.dir)
			child.targetSize.setMain(constants.dir, clamped)
			child.outerTargetSize.setMain(
				constants.dir,
				child.targetSize.main(constants.dir)+child.margin.mainAxisSum(constants.dir),
			)

			totalViolation += child.violation
		}

		for _, child := range unfrozen {
			switch {
			case totalViolation > 0.0:
				child.frozen = child.violation > 0.0
			case totalViolation < 0.0:
				child.frozen = child.violation < 0.0
			default:
				child.frozen = true
			}
		}
	}
}

func determineHypotheticalCrossSize(
	tree LayoutFlexboxContainer,
	line *flexLine,
	constants *algoConstants,
	availableSpace Size[AvailableSpace],
) {
	calcFn := treeCalc(tree)
	for i := range line.items {
		child := &line.items[i]
		paddingBorderSum := child.padding.Add(child.border).crossAxisSum(constants.dir)

		childAvailableMain := AvailableSpaceDefinite(f32Max(constants.containerSize.main(constants.dir)-child.margin.mainAxisSum(constants.dir), 0.0))

		transferredMinCross := SizeOptionF32MaybeApplyAspectRatio(child.minSize, child.aspectRatio).cross(constants.dir)
		transferredMaxCross := SizeOptionF32MaybeApplyAspectRatio(child.maxSize, child.aspectRatio).cross(constants.dir)

		childCross := OptionF32MaybeMaxF32(
			OptionF32MaybeClamp(child.size.cross(constants.dir), transferredMinCross, transferredMaxCross),
			paddingBorderSum,
		)

		childAvailableCross := availableSpace.cross(constants.dir).
			MapDefiniteValue(func(val float32) float32 {
				return f32Max(constants.dividedCrossSpace(val)-child.margin.crossAxisSum(constants.dir), 0.0)
			}).
			MaybeClamp(transferredMinCross, transferredMaxCross).
			MaybeMaxF32(paddingBorderSum)

		crossStretchSize := constants.nodeInnerSize.cross(constants.dir)
		if v, ok := crossStretchSize.Get(); ok {
			crossStretchSize = some(constants.dividedCrossSpace(v))
		}
		crossStretchSize = OptionF32MaybeMaxF32(OptionF32MaybeSubF32(crossStretchSize, child.margin.crossAxisSum(constants.dir)), 0.0)
		if res, ok := resolveSizingKeyword(
			child.sizeStyle.cross(constants.dir),
			crossStretchSize,
			constants.nodeInnerSize.cross(constants.dir),
			calcFn,
		).Get(); ok && res.kind == sizingKeywordResolutionKindMeasure {
			childAvailableCross = res.measure
		}

		childInnerCross, ok := childCross.Get()
		if !ok {
			knownDimensions := Size[Option[float32]]{Width: childCross, Height: some(child.targetSize.Height)}
			availSpace := Size[AvailableSpace]{Width: childAvailableCross, Height: childAvailableMain}
			if constants.isRow {
				knownDimensions = Size[Option[float32]]{Width: some(child.targetSize.Width), Height: childCross}
				availSpace = Size[AvailableSpace]{Width: childAvailableMain, Height: childAvailableCross}
			}
			childInnerCross = f32Max(
				F32MaybeClamp(
					tree.ComputeChildLayout(child.node, LayoutInput{
						RunMode:                       RunModeComputeSize,
						SizingMode:                    SizingModeContentSize,
						Axis:                          RequestedAxisFromAbsoluteAxis(constants.dir.crossAxis()),
						KnownDimensions:               knownDimensions,
						KnownDimensionsAreDefinite:    itemKnownDimensionDefiniteness(constants, child),
						ParentSize:                    constants.nodeInnerSize,
						AvailableSpace:                availSpace,
						VerticalMarginsAreCollapsible: LineBoolFalse,
					}).Size.GetAbs(constants.dir.crossAxis()),
					transferredMinCross,
					transferredMaxCross,
				),
				paddingBorderSum,
			)
		}
		childOuterCross := childInnerCross + child.margin.crossAxisSum(constants.dir)

		child.hypotheticalInnerSize.setCross(constants.dir, childInnerCross)
		child.hypotheticalOuterSize.setCross(constants.dir, childOuterCross)
	}
}

func calculateChildrenBaseLines(
	tree LayoutFlexboxContainer,
	nodeSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	flexLines []flexLine,
	constants *algoConstants,
) {
	if !constants.isRow {
		return
	}

	for li := range flexLines {
		line := &flexLines[li]
		lineBaselineChildCount := 0
		for i := range line.items {
			if line.items[i].participatesInBaselineAlignment(constants.dir) {
				lineBaselineChildCount++
			}
		}
		if lineBaselineChildCount <= 1 {
			continue
		}

		for i := range line.items {
			child := &line.items[i]
			if !child.participatesInBaselineAlignment(constants.dir) {
				continue
			}

			knownDimensions := Size[Option[float32]]{
				Width:  some(child.hypotheticalInnerSize.Width),
				Height: some(child.targetSize.Height),
			}
			availSpace := Size[AvailableSpace]{
				Width:  availableSpace.Width.MaybeSet(nodeSize.Width),
				Height: AvailableSpaceDefinite(constants.containerSize.Height),
			}
			if constants.isRow {
				knownDimensions = Size[Option[float32]]{
					Width:  some(child.targetSize.Width),
					Height: some(child.hypotheticalInnerSize.Height),
				}
				availSpace = Size[AvailableSpace]{
					Width:  AvailableSpaceDefinite(constants.containerSize.Width),
					Height: availableSpace.Height.MaybeSet(nodeSize.Height),
				}
			}

			measuredSizeAndBaselines := tree.ComputeChildLayout(child.node, LayoutInput{
				RunMode:                       RunModePerformLayout,
				SizingMode:                    SizingModeContentSize,
				Axis:                          RequestedAxisBoth,
				KnownDimensions:               knownDimensions,
				KnownDimensionsAreDefinite:    itemKnownDimensionDefiniteness(constants, child),
				ParentSize:                    constants.nodeInnerSize,
				AvailableSpace:                SizeAvailableSpaceMaybeMaxF32(SizeAvailableSpaceMaybeSubF32(availSpace, child.margin.SumAxes()), SizeF32Zero),
				VerticalMarginsAreCollapsible: LineBoolFalse,
			})

			baselineOpt := measuredSizeAndBaselines.Baselines.First
			height := measuredSizeAndBaselines.Size.Height

			var baseline float32
			if child.overflow.Y.IsScrollContainer() {
				baseline = f32Max(f32Min(baselineOpt.UnwrapOr(height), height), 0.0)
			} else {
				baseline = baselineOpt.UnwrapOr(height)
			}

			child.baseline = baseline + child.margin.Top
		}
	}
}

func calculateCrossSize(flexLines []flexLine, nodeSize Size[Option[float32]], constants *algoConstants) {
	if !constants.isWrap && nodeSize.cross(constants.dir).IsSome() {
		crossAxisPaddingBorder := constants.contentBoxInset.crossAxisSum(constants.dir)
		crossMinSize := constants.minSize.cross(constants.dir)
		crossMaxSize := constants.maxSize.cross(constants.dir)
		flexLines[0].crossSize = OptionF32MaybeMaxF32(
			OptionF32MaybeSubF32(
				OptionF32MaybeClamp(nodeSize.cross(constants.dir), crossMinSize, crossMaxSize),
				crossAxisPaddingBorder,
			),
			0.0,
		).UnwrapOr(0.0)
	} else {
		for li := range flexLines {
			line := &flexLines[li]
			maxBaseline := float32(0.0)
			for i := range line.items {
				maxBaseline = f32Max(maxBaseline, line.items[i].baseline)
			}
			crossSize := float32(0.0)
			for i := range line.items {
				child := &line.items[i]
				var v float32
				if child.participatesInBaselineAlignment(constants.dir) {
					v = maxBaseline - child.baseline + child.hypotheticalOuterSize.cross(constants.dir)
				} else {
					v = child.hypotheticalOuterSize.cross(constants.dir)
				}
				crossSize = f32Max(crossSize, v)
			}
			line.crossSize = crossSize
		}

		if !constants.isWrap {
			crossAxisPaddingBorder := constants.contentBoxInset.crossAxisSum(constants.dir)
			crossMinSize := constants.minSize.cross(constants.dir)
			crossMaxSize := constants.maxSize.cross(constants.dir)
			flexLines[0].crossSize = F32MaybeClamp(
				flexLines[0].crossSize,
				OptionF32MaybeSubF32(crossMinSize, crossAxisPaddingBorder),
				OptionF32MaybeSubF32(crossMaxSize, crossAxisPaddingBorder),
			)
		}
	}
}

func handleAlignContentStretch(flexLines []flexLine, nodeSize Size[Option[float32]], constants *algoConstants) {
	if constants.alignContent == AlignContentStretch {
		crossAxisPaddingBorder := constants.contentBoxInset.crossAxisSum(constants.dir)
		crossMinSize := constants.minSize.cross(constants.dir)
		crossMaxSize := constants.maxSize.cross(constants.dir)
		containerMinInnerCross := OptionF32MaybeMaxF32(
			OptionF32MaybeSubF32(
				OptionF32MaybeClamp(nodeSize.cross(constants.dir).Or(crossMinSize), crossMinSize, crossMaxSize),
				crossAxisPaddingBorder,
			),
			0.0,
		).UnwrapOr(0.0)

		totalCrossAxisGap := sumAxisGaps(constants.gap.cross(constants.dir), len(flexLines))
		linesTotalCross := float32(0.0)
		for i := range flexLines {
			linesTotalCross += flexLines[i].crossSize
		}
		linesTotalCross += totalCrossAxisGap

		if linesTotalCross < containerMinInnerCross {
			remaining := containerMinInnerCross - linesTotalCross
			addition := remaining / float32(len(flexLines))
			for i := range flexLines {
				flexLines[i].crossSize += addition
			}
		}
	}
}

func determineUsedCrossSize(tree LayoutFlexboxContainer, flexLines []flexLine, constants *algoConstants) {
	calcFn := treeCalc(tree)
	for li := range flexLines {
		line := &flexLines[li]
		lineCrossSize := line.crossSize

		for i := range line.items {
			child := &line.items[i]
			childStyle := tree.GetFlexboxChildStyle(child.node)
			crossIsStretch := child.sizeStyle.cross(constants.dir).IsStretch()
			var targetCross float32
			if !child.marginIsAuto.crossStart(constants.dir) &&
				!child.marginIsAuto.crossEnd(constants.dir) &&
				(crossIsStretch ||
					(child.alignSelf == AlignItemsStretch &&
						childStyle.Size().cross(constants.dir).IsAuto())) {
				padding := RectResolveOrZeroOptionF32(childStyle.Padding(), constants.nodeInnerSize.Width, calcFn)
				border := RectResolveOrZeroOptionF32(childStyle.Border(), constants.nodeInnerSize.Width, calcFn)
				pbSum := padding.Add(border).SumAxes()
				boxSizingAdjustment := SizeF32Zero
				if childStyle.BoxSizing() == BoxSizingContentBox {
					boxSizingAdjustment = pbSum
				}

				maxSizeIgnoringAspectRatio := SizeOptionF32MaybeAddF32(
					SizeMaybeResolve(childStyle.MaxSize(), constants.nodeInnerSize, calcFn),
					boxSizingAdjustment,
				)

				targetCross = F32MaybeClamp(
					f32Max(lineCrossSize-child.margin.crossAxisSum(constants.dir), 0.0),
					child.minSize.cross(constants.dir),
					maxSizeIgnoringAspectRatio.cross(constants.dir),
				)
			} else {
				targetCross = child.hypotheticalInnerSize.cross(constants.dir)
			}
			child.targetSize.setCross(constants.dir, targetCross)

			child.outerTargetSize.setCross(
				constants.dir,
				child.targetSize.cross(constants.dir)+child.margin.crossAxisSum(constants.dir),
			)
		}
	}
}

func distributeRemainingFreeSpace(flexLines []flexLine, constants *algoConstants) {
	for li := range flexLines {
		line := &flexLines[li]
		totalMainAxisGap := sumAxisGaps(constants.gap.main(constants.dir), len(line.items))
		usedSpace := float32(0.0)
		for i := range line.items {
			usedSpace += line.items[i].outerTargetSize.main(constants.dir)
		}
		usedSpace = totalMainAxisGap + usedSpace
		freeSpace := constants.innerContainerSize.main(constants.dir) - usedSpace
		numAutoMargins := 0

		for i := range line.items {
			child := &line.items[i]
			if child.marginIsAuto.mainStart(constants.dir) {
				numAutoMargins++
			}
			if child.marginIsAuto.mainEnd(constants.dir) {
				numAutoMargins++
			}
		}

		if freeSpace > 0.0 && numAutoMargins > 0 {
			margin := freeSpace / float32(numAutoMargins)

			for i := range line.items {
				child := &line.items[i]
				if child.marginIsAuto.mainStart(constants.dir) {
					if constants.isRow {
						child.margin.Left = margin
					} else {
						child.margin.Top = margin
					}
				}
				if child.marginIsAuto.mainEnd(constants.dir) {
					if constants.isRow {
						child.margin.Right = margin
					} else {
						child.margin.Bottom = margin
					}
				}
			}

			freeSpace = 0.0
		}

		numItems := len(line.items)
		layoutReverse := constants.dir.isReverse()
		gap := constants.gap.main(constants.dir)
		justifyContentMode := applyAlignmentFallback(freeSpace, numItems, constants.justifyContent)

		justifyItem := func(i int, child *flexItem) {
			child.offsetMain = computeAlignmentOffset(freeSpace, numItems, gap, justifyContentMode, layoutReverse, i == 0)
		}

		if layoutReverse {
			for i := range numItems {
				justifyItem(i, &line.items[numItems-1-i])
			}
		} else {
			for i := range numItems {
				justifyItem(i, &line.items[i])
			}
		}
	}
}

func resolveCrossAxisAutoMargins(flexLines []flexLine, constants *algoConstants) {
	for li := range flexLines {
		line := &flexLines[li]
		lineCrossSize := line.crossSize
		maxBaseline := float32(0.0)
		for i := range line.items {
			maxBaseline = f32Max(maxBaseline, line.items[i].baseline)
		}
		maxBaselineToBottomDistance := float32(0.0)
		for i := range line.items {
			child := &line.items[i]
			if child.participatesInBaselineAlignment(constants.dir) {
				maxBaselineToBottomDistance = f32Max(maxBaselineToBottomDistance, child.outerTargetSize.cross(constants.dir)-child.baseline)
			}
		}

		for i := range line.items {
			child := &line.items[i]
			freeSpace := lineCrossSize - child.outerTargetSize.cross(constants.dir)

			switch {
			case child.marginIsAuto.crossStart(constants.dir) && child.marginIsAuto.crossEnd(constants.dir):
				if constants.isRow {
					child.margin.Top = freeSpace / 2.0
					child.margin.Bottom = freeSpace / 2.0
				} else {
					child.margin.Left = freeSpace / 2.0
					child.margin.Right = freeSpace / 2.0
				}
			case child.marginIsAuto.crossStart(constants.dir):
				if constants.isRow {
					child.margin.Top = freeSpace
				} else {
					child.margin.Left = freeSpace
				}
			case child.marginIsAuto.crossEnd(constants.dir):
				if constants.isRow {
					child.margin.Bottom = freeSpace
				} else {
					child.margin.Right = freeSpace
				}
			default:
				child.offsetCross = alignFlexItemsAlongCrossAxis(
					child,
					freeSpace,
					maxBaseline,
					maxBaselineToBottomDistance,
					constants,
				)
			}
		}
	}
}

func alignFlexItemsAlongCrossAxis(
	child *flexItem,
	freeSpace float32,
	maxBaseline float32,
	maxBaselineToBottomDistance float32,
	constants *algoConstants,
) float32 {
	crossAxisShouldReverse := constants.isColumn && constants.layoutDirection == DirectionRtl

	alignKeyword := child.alignSelf.keyword
	if child.alignSelf.IsSafe() && freeSpace < 0.0 {
		alignKeyword = AlignItemsKeywordStart
	}

	switch alignKeyword {
	case AlignItemsKeywordStart:
		if crossAxisShouldReverse {
			return freeSpace
		}
		return 0.0
	case AlignItemsKeywordFlexStart:
		if constants.isWrapReverse != crossAxisShouldReverse {
			return freeSpace
		}
		return 0.0
	case AlignItemsKeywordEnd:
		if crossAxisShouldReverse {
			return 0.0
		}
		return freeSpace
	case AlignItemsKeywordFlexEnd:
		if constants.isWrapReverse != crossAxisShouldReverse {
			return 0.0
		}
		return freeSpace
	case AlignItemsKeywordCenter:
		return freeSpace / 2.0
	case AlignItemsKeywordBaseline:
		if constants.isRow {
			if constants.isWrapReverse {
				lineCrossSize := freeSpace + child.outerTargetSize.cross(constants.dir)
				return lineCrossSize - maxBaselineToBottomDistance - child.baseline
			}
			return maxBaseline - child.baseline
		}
		baselineColumnShouldReverse := crossAxisShouldReverse && !constants.isWrap
		if constants.isWrapReverse != baselineColumnShouldReverse {
			return freeSpace
		}
		return 0.0
	case AlignItemsKeywordStretch:
		if constants.isWrapReverse != crossAxisShouldReverse {
			return freeSpace
		}
		return 0.0
	default:
		panic("internal error: entered unreachable code")
	}
}

func determineContainerCrossSize(flexLines []flexLine, nodeSize Size[Option[float32]], constants *algoConstants) float32 {
	totalCrossAxisGap := sumAxisGaps(constants.gap.cross(constants.dir), len(flexLines))
	totalLineCrossSize := float32(0.0)
	for i := range flexLines {
		totalLineCrossSize += flexLines[i].crossSize
	}

	paddingBorderSum := constants.contentBoxInset.crossAxisSum(constants.dir)
	crossScrollbarGutter := constants.scrollbarGutter.cross(constants.dir)
	minCrossSize := constants.minSize.cross(constants.dir)
	maxCrossSize := constants.maxSize.cross(constants.dir)
	outerContainerSize := f32Max(
		F32MaybeClamp(
			nodeSize.cross(constants.dir).UnwrapOr(totalLineCrossSize+totalCrossAxisGap+paddingBorderSum),
			minCrossSize,
			maxCrossSize,
		),
		paddingBorderSum-crossScrollbarGutter,
	)
	innerContainerSize := f32Max(outerContainerSize-paddingBorderSum, 0.0)

	constants.containerSize.setCross(constants.dir, outerContainerSize)
	constants.innerContainerSize.setCross(constants.dir, innerContainerSize)

	return totalLineCrossSize
}

func alignFlexLinesPerAlignContent(flexLines []flexLine, constants *algoConstants, totalCrossSize float32) {
	numLines := len(flexLines)
	gap := constants.gap.cross(constants.dir)
	totalCrossAxisGap := sumAxisGaps(gap, numLines)
	freeSpace := constants.innerContainerSize.cross(constants.dir) - totalCrossSize - totalCrossAxisGap

	alignContentMode := applyAlignmentFallback(freeSpace, numLines, constants.alignContent)

	alignLine := func(i int, line *flexLine) {
		line.offsetCross = computeAlignmentOffset(freeSpace, numLines, gap, alignContentMode, constants.isWrapReverse, i == 0)
	}

	if constants.isWrapReverse {
		for i := range numLines {
			alignLine(i, &flexLines[numLines-1-i])
		}
	} else {
		for i := range numLines {
			alignLine(i, &flexLines[i])
		}
	}
}

func calculateFlexItem(
	tree LayoutFlexboxContainer,
	item *flexItem,
	totalOffsetMain *float32,
	totalOffsetCross float32,
	lineOffsetCross float32,
	totalOverflowRect *Rect[float32],
	border Rect[float32],
	constants *algoConstants,
) {
	containerSize := constants.containerSize
	nodeInnerSize := constants.nodeInnerSize
	direction := constants.dir
	layoutDirection := constants.layoutDirection
	definiteness := itemKnownDimensionDefiniteness(constants, item)
	layoutOutput := tree.ComputeChildLayout(item.node, LayoutInput{
		RunMode:                    RunModePerformLayout,
		SizingMode:                 SizingModeContentSize,
		Axis:                       RequestedAxisBoth,
		KnownDimensions:            item.targetSize.Map(someF32),
		KnownDimensionsAreDefinite: definiteness,
		ParentSize:                 nodeInnerSize,
		AvailableSpace: containerSize.Sub(item.margin.SumAxes()).Map(func(s float32) AvailableSpace {
			return AvailableSpaceDefinite(f32Max(s, 0.0))
		}),
		VerticalMarginsAreCollapsible: LineBoolFalse,
	})
	size := layoutOutput.Size
	scrollableOverflowRect := layoutOutput.ScrollableOverflowRect

	isRtlRow := direction.isRow() && layoutDirection.IsRtl()
	isRtlColumn := direction.isColumn() && layoutDirection.IsRtl()
	mainRelativeInset := item.relativeInset.main(direction)
	crossRelativeInset := item.relativeInset.cross(direction)
	effectiveLineOffsetCross := lineOffsetCross
	if isRtlColumn {
		effectiveLineOffsetCross = 0.0
	}

	var offsetMain float32
	if isRtlRow {
		offsetMain = *totalOffsetMain - item.offsetMain - item.margin.mainEnd(direction) - mainRelativeInset - size.Width
	} else {
		offsetMain = *totalOffsetMain + item.offsetMain + item.margin.mainStart(direction) + mainRelativeInset
	}

	offsetCross := totalOffsetCross +
		item.offsetCross +
		effectiveLineOffsetCross +
		item.margin.crossStart(direction) +
		crossRelativeInset

	if direction.isRow() {
		baselineOffsetCross := totalOffsetCross + item.offsetCross + effectiveLineOffsetCross + item.margin.crossStart(direction)
		innerBaseline := layoutOutput.Baselines.First.UnwrapOr(size.Height)
		if item.overflow.Y.IsScrollContainer() {
			innerBaseline = f32Max(f32Min(innerBaseline, size.Height), 0.0)
		}
		item.baseline = baselineOffsetCross + innerBaseline
	} else {
		baselineOffsetMain := *totalOffsetMain + item.offsetMain + item.margin.mainStart(direction)
		innerBaseline := layoutOutput.Baselines.First.UnwrapOr(size.Height)
		item.baseline = baselineOffsetMain + innerBaseline
	}

	location := Point[float32]{X: offsetCross, Y: offsetMain}
	if direction.isRow() {
		location = Point[float32]{X: offsetMain, Y: offsetCross}
	}
	scrollbarSize := SizeF32Zero
	if item.overflow.Y == OverflowScroll {
		scrollbarSize.Width = item.scrollbarWidth
	}
	if item.overflow.X == OverflowScroll {
		scrollbarSize.Height = item.scrollbarWidth
	}

	tree.SetUnroundedLayout(item.node, &Layout{
		Order:                  item.order,
		Size:                   size,
		ScrollableOverflowRect: scrollableOverflowRect,
		ScrollbarSize:          scrollbarSize,
		Location:               location,
		Padding:                item.padding,
		Border:                 item.border,
		Margin:                 item.margin,
	})

	item.oofCandidates = layoutOutput.OofCandidates.Take()
	if !item.oofCandidates.IsEmpty() {
		item.oofCandidates.Translate(location)
	}

	if isRtlRow {
		*totalOffsetMain -= item.offsetMain + item.margin.mainAxisSum(direction) + size.main(direction)
	} else {
		*totalOffsetMain += item.offsetMain + item.margin.mainAxisSum(direction) + size.main(direction)
	}

	contributionLocation := Point[float32]{X: location.X - border.Left, Y: location.Y - border.Top}
	if layoutDirection.IsRtl() {
		contributionLocation = Point[float32]{X: containerSize.Width - (location.X + size.Width) - border.Right, Y: location.Y - border.Top}
	}
	*totalOverflowRect = totalOverflowRect.Union(computeScrollableOverflowContribution(
		contributionLocation,
		size,
		scrollableOverflowRect,
		item.overflow,
		item.contain,
		constants.isScrollContainer,
	))
}

func calculateLayoutLine(
	tree LayoutFlexboxContainer,
	line *flexLine,
	totalOffsetCross *float32,
	overflowRect *Rect[float32],
	border Rect[float32],
	constants *algoConstants,
) {
	containerSize := constants.containerSize
	paddingBorder := constants.contentBoxInset
	direction := constants.dir
	layoutDirection := constants.layoutDirection
	var totalOffsetMain float32
	if layoutDirection.IsRtl() && direction.isRow() {
		totalOffsetMain = containerSize.Width - paddingBorder.mainEnd(direction)
	} else {
		totalOffsetMain = paddingBorder.mainStart(direction)
	}
	lineOffsetCross := line.offsetCross

	isRtlColumn := layoutDirection.IsRtl() && direction.isColumn()
	if isRtlColumn {
		*totalOffsetCross -= lineOffsetCross + line.crossSize
	}

	if direction.isReverse() {
		for i := len(line.items) - 1; i >= 0; i-- {
			calculateFlexItem(tree, &line.items[i], &totalOffsetMain, *totalOffsetCross, lineOffsetCross, overflowRect, border, constants)
		}
	} else {
		for i := range line.items {
			calculateFlexItem(tree, &line.items[i], &totalOffsetMain, *totalOffsetCross, lineOffsetCross, overflowRect, border, constants)
		}
	}

	if !isRtlColumn {
		*totalOffsetCross += lineOffsetCross + line.crossSize
	}
}

func finalLayoutPass(tree LayoutFlexboxContainer, flexLines []flexLine, constants *algoConstants) Rect[float32] {
	var totalOffsetCross float32
	if constants.isColumn && constants.layoutDirection.IsRtl() {
		totalOffsetCross = constants.containerSize.Width - constants.contentBoxInset.crossEnd(constants.dir)
	} else {
		totalOffsetCross = constants.contentBoxInset.crossStart(constants.dir)
	}

	overflowRect := RectF32Zero

	if constants.isWrapReverse {
		for i := len(flexLines) - 1; i >= 0; i-- {
			calculateLayoutLine(tree, &flexLines[i], &totalOffsetCross, &overflowRect, constants.border, constants)
		}
	} else {
		for i := range flexLines {
			calculateLayoutLine(tree, &flexLines[i], &totalOffsetCross, &overflowRect, constants.border, constants)
		}
	}

	if constants.isScrollContainer {
		if constants.layoutDirection.IsRtl() {
			overflowRect.Right += constants.contentBoxInset.Left - constants.border.Left - constants.scrollbarGutter.X
		} else {
			overflowRect.Right += constants.contentBoxInset.Right - constants.border.Right - constants.scrollbarGutter.X
		}
		overflowRect.Bottom += constants.contentBoxInset.Bottom - constants.border.Bottom - constants.scrollbarGutter.Y
	}

	return overflowRect
}

func collectOofCandidates(
	tree LayoutFlexboxContainer,
	node NodeID,
	constants *algoConstants,
	flexLines []flexLine,
	candidates *OofCandidates,
) {
	var items []*flexItem
	for li := range flexLines {
		for i := range flexLines[li].items {
			item := &flexLines[li].items[i]
			if !item.oofCandidates.IsEmpty() {
				items = append(items, item)
			}
		}
	}
	next := 0

	dir := constants.dir
	containerSize := constants.containerSize
	contentBoxInset := constants.contentBoxInset

	mainAxisIsHorizontal := constants.isRow
	crossAxisIsHorizontal := !constants.isRow
	mainIsRtl := mainAxisIsHorizontal && constants.layoutDirection.IsRtl()
	crossIsRtl := crossAxisIsHorizontal && constants.layoutDirection.IsRtl()
	mainAxisFlexStartReversed := dir.isReverse() != mainIsRtl
	crossAxisFlexStartReversed := constants.isWrapReverse != crossIsRtl

	mainArea := Line[float32]{
		Start: contentBoxInset.mainStart(dir),
		End:   containerSize.main(dir) - contentBoxInset.mainEnd(dir),
	}
	crossArea := Line[float32]{
		Start: contentBoxInset.crossStart(dir),
		End:   containerSize.cross(dir) - contentBoxInset.crossEnd(dir),
	}

	childCount := tree.ChildCount(node)
	for order := 0; order < childCount; order++ {
		child := tree.GetChildID(node, order)
		childStyle := tree.GetFlexboxChildStyle(child)
		position := childStyle.Position()

		if childStyle.BoxGenerationMode() == BoxGenerationModeNone {
			continue
		}
		if !position.IsOutOfFlow() {
			if next < len(items) && items[next].order == uint32(order) {
				candidates.Append(&items[next].oofCandidates)
				next++
			}
			continue
		}

		alignSelf := resolveNormalAlignment(childStyle.AlignSelf().UnwrapOr(constants.alignItems)).
			ResolveSelfRelative(childStyle.Direction(), constants.layoutDirection, constants.isColumn)

		mainKeyword := constants.justifyContent.Keyword()
		mainStartPosition := true
		switch mainKeyword {
		case AlignContentKeywordStart:
			mainStartPosition = !mainIsRtl
		case AlignContentKeywordEnd:
			mainStartPosition = mainIsRtl
		}
		var mainEdge AxisStaticEdge
		switch mainKeyword {
		case AlignContentKeywordSpaceBetween, AlignContentKeywordNormal, AlignContentKeywordStretch, AlignContentKeywordFlexStart:
			if mainAxisFlexStartReversed {
				mainEdge = AxisStaticEdgeEnd
			} else {
				mainEdge = AxisStaticEdgeStart
			}
		case AlignContentKeywordFlexEnd:
			if mainAxisFlexStartReversed {
				mainEdge = AxisStaticEdgeStart
			} else {
				mainEdge = AxisStaticEdgeEnd
			}
		case AlignContentKeywordStart, AlignContentKeywordEnd:
			if mainStartPosition {
				mainEdge = AxisStaticEdgeStart
			} else {
				mainEdge = AxisStaticEdgeEnd
			}
		case AlignContentKeywordSpaceEvenly, AlignContentKeywordSpaceAround, AlignContentKeywordCenter:
			mainEdge = AxisStaticEdgeCenter
		}

		crossEdgeFor := func(keyword AlignItemsKeyword) AxisStaticEdge {
			startPosition := true
			switch keyword {
			case AlignItemsKeywordStart, AlignItemsKeywordBaseline:
				startPosition = !crossIsRtl
			case AlignItemsKeywordEnd:
				startPosition = crossIsRtl
			}
			switch keyword {
			case AlignItemsKeywordStart, AlignItemsKeywordEnd, AlignItemsKeywordBaseline:
				if startPosition {
					return AxisStaticEdgeStart
				}
				return AxisStaticEdgeEnd
			case AlignItemsKeywordStretch, AlignItemsKeywordFlexStart:
				if crossAxisFlexStartReversed {
					return AxisStaticEdgeEnd
				}
				return AxisStaticEdgeStart
			case AlignItemsKeywordFlexEnd:
				if crossAxisFlexStartReversed {
					return AxisStaticEdgeStart
				}
				return AxisStaticEdgeEnd
			case AlignItemsKeywordCenter:
				return AxisStaticEdgeCenter
			default:
				panic("internal error: entered unreachable code")
			}
		}

		crossPos := AxisStaticPosition{
			Area: crossArea,
			Align: AxisStaticAlign{
				Keyword:  crossEdgeFor(alignSelf.keyword),
				Safety:   alignSelf.safety,
				Fallback: crossEdgeFor(resolveSelfAlignmentSafety(alignSelf, true)),
			},
		}
		mainPos := AxisStaticPosition{Area: mainArea, Align: AxisStaticAlignFromKeyword(mainEdge)}

		staticPosition := Point[AxisStaticPosition]{X: crossPos, Y: mainPos}
		if constants.isRow {
			staticPosition = Point[AxisStaticPosition]{X: mainPos, Y: crossPos}
		}
		candidates.Push(OofCandidate{Node: child, Order: uint32(order), Position: position, StaticPosition: staticPosition})
	}
}

func sumAxisGaps(gap float32, numItems int) float32 {
	if numItems <= 1 {
		return 0.0
	}
	return float32(gap * float32(numItems-1))
}
