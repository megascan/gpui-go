package taffy

import "math"

// ComputeBlockAlignContentOffset computes the block-axis offset that align-content applies to the in-flow
// content of a block container, given the free space between the container's content box and its content.
func ComputeBlockAlignContentOffset(alignContent AlignContent, freeSpace float32) float32 {
	keyword := applyAlignmentFallback(freeSpace, 1, alignContent)
	return computeAlignmentOffset(freeSpace, 1, 0.0, keyword, false, true)
}

// BlockFormattingContext is the context for positioning block and float boxes within a block formatting context.
type BlockFormattingContext struct {
	floatContext FloatContext
}

// NewBlockFormattingContext creates a new BlockFormattingContext.
func NewBlockFormattingContext() BlockFormattingContext {
	return BlockFormattingContext{floatContext: NewFloatContext()}
}

// RootBlockContext creates the initial BlockContext for this BlockFormattingContext.
func (b *BlockFormattingContext) RootBlockContext() BlockContext {
	return BlockContext{
		bfc:                      b,
		yOffset:                  0.0,
		insets:                   [2]float32{0.0, 0.0},
		contentBoxInsets:         [2]float32{0.0, 0.0},
		floatContentContribution: float32(math.Inf(-1)),
		isRoot:                   true,
		adjoiningFloats:          [2]bool{false, false},
		topAdjoiningFloats:       none[[2]bool](),
	}
}

// BlockContext is the per-block state of a block within a block formatting context.
type BlockContext struct {
	bfc                      *BlockFormattingContext
	yOffset                  float32
	insets                   [2]float32
	contentBoxInsets         [2]float32
	floatContentContribution float32
	isRoot                   bool
	adjoiningFloats          [2]bool
	topAdjoiningFloats       Option[[2]bool]
}

// SubContext creates a sub-BlockContext for a child block node.
func (c *BlockContext) SubContext(additionalYOffset float32, insets [2]float32) BlockContext {
	insets = [2]float32{c.insets[0] + insets[0], c.insets[1] + insets[1]}
	return BlockContext{
		bfc:                      c.bfc,
		yOffset:                  c.yOffset + additionalYOffset,
		insets:                   insets,
		contentBoxInsets:         insets,
		floatContentContribution: float32(math.Inf(-1)),
		isRoot:                   false,
		adjoiningFloats:          c.adjoiningFloats,
		topAdjoiningFloats:       none[[2]bool](),
	}
}

// IsBfcRoot reports whether this block is the root block of its block formatting context.
func (c *BlockContext) IsBfcRoot() bool {
	return c.isRoot
}

// SetWidth sets the width of the overall block formatting context.
func (c *BlockContext) SetWidth(availableWidth float32) {
	c.bfc.floatContext.SetWidth(availableWidth)
}

// ApplyContentBoxInset sets the x-axis content-box insets of the BlockContext.
func (c *BlockContext) ApplyContentBoxInset(contentBoxXInsets [2]float32) {
	c.contentBoxInsets[0] = c.insets[0] + contentBoxXInsets[0]
	c.contentBoxInsets[1] = c.insets[1] + contentBoxXInsets[1]
}

// HasFloats reports whether the float context contains any floats.
func (c *BlockContext) HasFloats() bool {
	return c.bfc.floatContext.HasFloats()
}

// HasActiveFloats reports whether the float context contains any floats that extend to or below minY.
func (c *BlockContext) HasActiveFloats(minY float32) bool {
	return c.bfc.floatContext.HasActiveFloats(minY + c.yOffset)
}

// PlaceFloatedBox positions a floated box within the context.
func (c *BlockContext) PlaceFloatedBox(floatedBox Size[float32], minY float32, direction FloatDirection, clear Clear, adjoinsUnresolvedStrut bool) Point[float32] {
	if adjoinsUnresolvedStrut {
		c.adjoiningFloats[int(direction)] = true
	}
	pos := c.bfc.floatContext.PlaceFloatedBox(floatedBox, minY+c.yOffset, c.contentBoxInsets, direction, clear)
	pos.Y -= c.yOffset
	pos.X -= c.insets[0]

	c.floatContentContribution = f32Max(c.floatContentContribution, pos.Y+floatedBox.Height)

	return pos
}

// FindContentSlot searches for a space suitable for laying out non-floated content into.
func (c *BlockContext) FindContentSlot(minY float32, clear Clear, after Option[int]) ContentSlot {
	slot := c.bfc.floatContext.FindContentSlot(minY+c.yOffset, c.contentBoxInsets, clear, after)
	slot.Y -= c.yOffset
	slot.X -= c.insets[0]
	return slot
}

// FindBfcSlot searches for a space suitable for laying out a box that establishes an independent formatting context.
func (c *BlockContext) FindBfcSlot(minY float32, margins [2]float32, direction Direction, clear Clear, after Option[int]) BfcSlot {
	slot := c.bfc.floatContext.FindBfcSlot(minY+c.yOffset, c.contentBoxInsets, margins, direction, clear, after)
	slot.Y -= c.yOffset
	slot.X -= c.insets[0]
	return slot
}

// ClearedThreshold returns the bottom of the lowest relevant float for the clear property.
func (c *BlockContext) ClearedThreshold(clear Clear) Option[float32] {
	if threshold, ok := c.bfc.floatContext.ClearedThreshold(clear).Get(); ok {
		return some(threshold - c.yOffset)
	}
	return none[float32]()
}

// HasAdjoiningFloat reports whether a float adjoining the current margin-collapse strut has been placed on the
// sides relevant to clear.
func (c *BlockContext) HasAdjoiningFloat(clear Clear) bool {
	switch clear {
	case ClearLeft:
		return c.adjoiningFloats[0]
	case ClearRight:
		return c.adjoiningFloats[1]
	case ClearBoth:
		return c.adjoiningFloats[0] || c.adjoiningFloats[1]
	default:
		return false
	}
}

func (c *BlockContext) mergeAdjoiningFloats(flags [2]bool) {
	c.adjoiningFloats[0] = c.adjoiningFloats[0] || flags[0]
	c.adjoiningFloats[1] = c.adjoiningFloats[1] || flags[1]
}

func (c *BlockContext) commitStrut() {
	if c.topAdjoiningFloats.IsNone() {
		c.topAdjoiningFloats = some(c.adjoiningFloats)
	}
	c.adjoiningFloats = [2]bool{false, false}
}

func (c *BlockContext) getTopAdjoiningFloats() [2]bool {
	return c.topAdjoiningFloats.UnwrapOr(c.adjoiningFloats)
}

// AddChildFloatedContentHeightContribution updates the height that descendant floats consume with the
// contribution of a particular child (relative to this block's border-top).
func (c *BlockContext) AddChildFloatedContentHeightContribution(childContribution float32) {
	c.floatContentContribution = f32Max(c.floatContentContribution, childContribution)
}

// FloatedContentHeightContribution returns the height that descendant floats consume.
func (c *BlockContext) FloatedContentHeightContribution() float32 {
	return c.floatContentContribution
}

type blockItem struct {
	nodeID                 NodeID
	order                  uint32
	isTable                bool
	isCompressibleReplaced bool
	isInSameBfc            bool
	float                  Float
	clear                  Clear
	sizeStyle              Size[Dimension]
	size                   Size[Option[float32]]
	minSize                Size[Option[float32]]
	maxSize                Size[Option[float32]]
	overflow               Point[Overflow]
	contain                Contain
	scrollbarWidth         float32
	position               Position
	direction              Direction
	alignSelf              Option[AlignSelf]
	justifySelf            Option[AlignSelf]
	inset                  Rect[LengthPercentageAuto]
	margin                 Rect[LengthPercentageAuto]
	padding                Rect[float32]
	border                 Rect[float32]
	paddingBorderSum       Size[float32]
	computedSize           Size[float32]
	staticPosition         Point[float32]
	canBeCollapsedThrough  bool
	finalLayout            Option[Layout]
	oofCandidates          OofCandidates
}

// ComputeBlockLayout computes the layout of a block container and its direct children. blockCtx is nil when no
// block formatting context is inherited.
func ComputeBlockLayout(tree LayoutBlockContainer, nodeID NodeID, inputs LayoutInput, blockCtx *BlockContext) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	runMode := inputs.RunMode
	style := tree.GetBlockContainerStyle(nodeID)
	calcFn := treeCalc(tree)

	overflow := style.Overflow()
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()
	contain := style.Contain()
	establishesNewBfc := isScrollContainer ||
		style.AlignContent().keyword != AlignContentKeywordNormal ||
		contain.EstablishesIndependentFormattingContext()
	aspectRatio := style.AspectRatio()
	padding := RectResolveOrZeroOptionF32(style.Padding(), parentSize.Width, calcFn)
	border := RectResolveOrZeroOptionF32(style.Border(), parentSize.Width, calcFn)
	paddingBorderSize := padding.Add(border).SumAxes()
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

	styledBasedKnownDimensions := SizeOptionF32MaybeMaxF32(
		SizeOptionOr(SizeOptionOr(knownDimensions, minMaxDefiniteSize), clampedStyleSize),
		paddingBorderSize,
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

	innerInputs := inputs
	innerInputs.KnownDimensions = styledBasedKnownDimensions
	var output LayoutOutput
	if blockCtx != nil && !establishesNewBfc {
		output = computeInner(tree, nodeID, innerInputs, blockCtx)
	} else {
		rootBfc := NewBlockFormattingContext()
		rootCtx := rootBfc.RootBlockContext()
		output = computeInner(tree, nodeID, innerInputs, &rootCtx)
	}

	if contain.SuppressesBaseline() {
		output.Baselines = BaselinesNone
	}

	return output
}

func computeInner(tree LayoutBlockContainer, nodeID NodeID, inputs LayoutInput, blockCtx *BlockContext) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	runMode := inputs.RunMode
	verticalMarginsAreCollapsible := inputs.VerticalMarginsAreCollapsible
	calcFn := treeCalc(tree)

	style := tree.GetBlockContainerStyle(nodeID)
	rawPadding := style.Padding()
	rawBorder := style.Border()
	rawMargin := style.Margin()
	aspectRatio := style.AspectRatio()
	padding := RectResolveOrZeroOptionF32(rawPadding, parentSize.Width, calcFn)
	border := RectResolveOrZeroOptionF32(rawBorder, parentSize.Width, calcFn)
	direction := style.Direction()

	offsets := style.Overflow().Transpose().Map(func(overflow Overflow) float32 {
		if overflow == OverflowScroll {
			return style.ScrollbarWidth()
		}
		return 0.0
	})
	var scrollbarGutter Rect[float32]
	switch direction {
	case DirectionRtl:
		scrollbarGutter = Rect[float32]{Top: 0.0, Left: offsets.X, Right: 0.0, Bottom: offsets.Y}
	default:
		scrollbarGutter = Rect[float32]{Top: 0.0, Left: 0.0, Right: offsets.X, Bottom: offsets.Y}
	}
	paddingBorder := padding.Add(border)
	paddingBorderSize := paddingBorder.SumAxes()
	contentBoxInset := paddingBorder.Add(scrollbarGutter)

	blockCtx.ApplyContentBoxInset([2]float32{contentBoxInset.Left, contentBoxInset.Right})

	boxSizingAdjustment := SizeF32Zero
	if style.BoxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}
	size := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.Size(), parentSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	minSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MinSize(), parentSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	maxSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MaxSize(), parentSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)

	derived := SizeOptionF32MaybeClamp(SizeOptionF32MaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
	knownDimensions = Size[Option[float32]]{
		Width:  knownDimensions.Width.Or(derived.Width),
		Height: knownDimensions.Height.Or(derived.Height),
	}
	percentageBasisDimensions := Size[Option[float32]]{
		Width:  knownDimensions.Width,
		Height: knownDimensions.Height,
	}
	if !inputs.KnownDimensionsAreDefinite.Height {
		percentageBasisDimensions.Height = none[float32]()
	}
	containerContentBoxSize := SizeOptionF32MaybeSubF32(percentageBasisDimensions, contentBoxInset.SumAxes())

	overflow := style.Overflow()
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()
	establishesNewBfc := isScrollContainer ||
		style.AlignContent().keyword != AlignContentKeywordNormal ||
		style.Contain().EstablishesIndependentFormattingContext()

	childrenMarginsAdjoinOwnEdges := Line[bool]{
		Start: verticalMarginsAreCollapsible.Start &&
			!establishesNewBfc &&
			!style.Position().IsOutOfFlow() &&
			padding.Top == 0.0 &&
			border.Top == 0.0,
		End: verticalMarginsAreCollapsible.End &&
			!establishesNewBfc &&
			!style.Position().IsOutOfFlow() &&
			padding.Bottom == 0.0 &&
			border.Bottom == 0.0,
	}
	ownMarginsCollapseWithChildren := Line[bool]{
		Start: childrenMarginsAdjoinOwnEdges.Start,
		End:   childrenMarginsAdjoinOwnEdges.End && size.Height.IsNone(),
	}
	hasStylesPreventingBeingCollapsedThrough := !style.IsBlock() ||
		blockCtx.IsBfcRoot() ||
		establishesNewBfc ||
		style.Position().IsOutOfFlow() ||
		padding.Top > 0.0 ||
		padding.Bottom > 0.0 ||
		border.Top > 0.0 ||
		border.Bottom > 0.0

	textAlign := style.TextAlign()
	alignContent := style.AlignContent()
	justifyItems := style.JustifyItems()

	items := generateItemList(tree, nodeID, containerContentBoxSize)

	var containerOuterWidth float32
	if w, ok := knownDimensions.Width.Get(); ok {
		containerOuterWidth = w
	} else {
		availableWidth := availableSpace.Width.MaybeSubF32(contentBoxInset.HorizontalAxisSum())
		intrinsicWidth := determineContentBasedContainerWidth(tree, items, availableWidth) +
			contentBoxInset.HorizontalAxisSum()
		containerOuterWidth = F32MaybeMax(F32MaybeClamp(intrinsicWidth, minSize.Width, maxSize.Width), some(paddingBorderSize.Width))
	}

	if containerOuterHeight, ok := knownDimensions.Height.Get(); ok && runMode == RunModeComputeSize {
		return LayoutOutputFromOuterSize(Size[float32]{Width: containerOuterWidth, Height: containerOuterHeight})
	}

	if runMode == RunModeComputeSize && inputs.Axis == RequestedAxisHorizontal {
		return LayoutOutputFromOuterSize(Size[float32]{Width: containerOuterWidth, Height: 0.0})
	}

	containerPercentageResolutionHeight := percentageBasisDimensions.Height
	if containerPercentageResolutionHeight.IsNone() && inputs.SizingMode == SizingModeInherentSize {
		containerPercentageResolutionHeight = OptionF32MaybeMax(size.Height, minSize.Height)
	}

	percentageResolutionWidth := parentSize.Width.UnwrapOr(containerOuterWidth)
	resolvedPadding := RectResolveOrZeroOptionF32(rawPadding, some(percentageResolutionWidth), calcFn)
	resolvedBorder := RectResolveOrZeroOptionF32(rawBorder, some(percentageResolutionWidth), calcFn)
	resolvedContentBoxInset := resolvedPadding.Add(resolvedBorder).Add(scrollbarGutter)
	inflowOverflowRect, intrinsicOuterHeight, firstChildTopMarginSet, lastChildBottomMarginSet, firstBaseline := performFinalLayoutOnInFlowChildren(
		tree,
		runMode,
		items,
		containerOuterWidth,
		containerPercentageResolutionHeight,
		contentBoxInset,
		resolvedContentBoxInset,
		resolvedBorder,
		textAlign,
		justifyItems,
		direction,
		ownMarginsCollapseWithChildren,
		childrenMarginsAdjoinOwnEdges,
		isScrollContainer,
		blockCtx,
	)

	if blockCtx.IsBfcRoot() || establishesNewBfc {
		intrinsicOuterHeight = f32Max(intrinsicOuterHeight, blockCtx.FloatedContentHeightContribution())
	}

	containerOuterHeight := F32MaybeMax(
		knownDimensions.Height.UnwrapOr(F32MaybeClamp(intrinsicOuterHeight, minSize.Height, maxSize.Height)),
		some(paddingBorderSize.Height),
	)
	finalOuterSize := Size[float32]{Width: containerOuterWidth, Height: containerOuterHeight}

	heightConstrainedByMinHeight := false
	if h, ok := minSize.Height.Get(); ok && h > 0.0 && h >= containerOuterHeight {
		heightConstrainedByMinHeight = true
	}
	ownBottomMarginCollapsesWithChildren := ownMarginsCollapseWithChildren.End && !heightConstrainedByMinHeight

	if alignContent.keyword != AlignContentKeywordNormal {
		containerInnerHeight := containerOuterHeight - resolvedContentBoxInset.VerticalAxisSum()
		inflowContentHeight := intrinsicOuterHeight - resolvedContentBoxInset.VerticalAxisSum()
		freeSpace := containerInnerHeight - inflowContentHeight
		groupOffset := ComputeBlockAlignContentOffset(alignContent, freeSpace)
		if b, ok := firstBaseline.Get(); ok {
			firstBaseline = some(b + groupOffset)
		}
		for i := range items {
			item := &items[i]
			if layout, ok := item.finalLayout.Get(); ok {
				layout.Location.Y += groupOffset
				item.finalLayout = some(layout)
			}
			if item.position.IsOutOfFlow() {
				item.staticPosition.Y += groupOffset
			}
		}
		anyInFlow := false
		for i := range items {
			if items[i].finalLayout.IsSome() {
				anyInFlow = true
				break
			}
		}
		if anyInFlow {
			inflowOverflowRect = RectF32Zero
			for i := range items {
				item := &items[i]
				if layout, ok := item.finalLayout.Get(); ok {
					var contributionLocation Point[float32]
					if direction.IsRtl() {
						contributionLocation = Point[float32]{
							X: containerOuterWidth - (layout.Location.X + layout.Size.Width) - resolvedBorder.Right,
							Y: layout.Location.Y - resolvedBorder.Top,
						}
					} else {
						contributionLocation = Point[float32]{
							X: layout.Location.X - resolvedBorder.Left,
							Y: layout.Location.Y - resolvedBorder.Top,
						}
					}
					inflowOverflowRect = inflowOverflowRect.Union(computeScrollableOverflowContribution(
						contributionLocation,
						layout.Size,
						layout.ScrollableOverflowRect,
						item.overflow,
						item.contain,
						isScrollContainer,
					))
				}
			}
		}
	}

	allInFlowChildrenCanBeCollapsedThrough := true
	for i := range items {
		item := &items[i]
		if item.float.IsFloated() {
			continue
		}
		if !(item.position.IsOutOfFlow() || item.canBeCollapsedThrough) {
			allInFlowChildrenCanBeCollapsedThrough = false
			break
		}
	}
	canBeCollapsedThrough := !hasStylesPreventingBeingCollapsedThrough &&
		containerOuterHeight == 0.0 &&
		allInFlowChildrenCanBeCollapsedThrough

	var topMargin, bottomMargin CollapsibleMarginSet
	if ownMarginsCollapseWithChildren.Start {
		topMargin = firstChildTopMarginSet
	} else {
		marginTop := rawMargin.Top.ResolveOrZero(parentSize.Width, calcFn)
		topMargin = CollapsibleMarginSetFromMargin(marginTop)
	}
	if ownBottomMarginCollapsesWithChildren {
		bottomMargin = lastChildBottomMarginSet
	} else {
		marginBottom := rawMargin.Bottom.ResolveOrZero(parentSize.Width, calcFn)
		bottomMargin = CollapsibleMarginSetFromMargin(marginBottom)
	}
	output := LayoutOutput{
		Size:                      finalOuterSize,
		ScrollableOverflowRect:    RectF32Zero,
		Baselines:                 BaselinesFromFirst(firstBaseline),
		TopMargin:                 topMargin,
		BottomMargin:              bottomMargin,
		MarginsCanCollapseThrough: canBeCollapsedThrough,
		OofCandidates:             OofCandidatesNone,
		OofPositioningArea:        none[OofPositioningArea](),
	}

	if runMode == RunModeComputeSize {
		return output
	}

	for i := range items {
		if layout, ok := items[i].finalLayout.Get(); ok {
			tree.SetUnroundedLayout(items[i].nodeID, &layout)
		}
	}

	candidates := NewOofCandidates()
	for i := range items {
		item := &items[i]
		if item.position.IsOutOfFlow() {
			inlineArea := Line[float32]{
				Start: resolvedContentBoxInset.Left,
				End:   containerOuterWidth - resolvedContentBoxInset.Right,
			}
			blockArea := Line[float32]{Start: item.staticPosition.Y, End: item.staticPosition.Y}
			justifySelf := item.justifySelf.UnwrapOr(justifyItems).ResolveSelfRelative(item.direction, direction, true)
			alignSelf := item.alignSelf.UnwrapOr(AlignItemsNormal).ResolveSelfRelative(item.direction, direction, false)
			candidates.Push(OofCandidate{
				Node:     item.nodeID,
				Order:    item.order,
				Position: item.position,
				StaticPosition: Point[AxisStaticPosition]{
					X: AxisStaticPositionFromAlignment(justifySelf, inlineArea, direction.IsRtl()),
					Y: AxisStaticPositionFromAlignment(alignSelf, blockArea, false),
				},
			})
		} else if !item.oofCandidates.IsEmpty() {
			if layout, ok := item.finalLayout.Get(); ok {
				item.oofCandidates.Translate(layout.Location)
			}
			candidates.Append(&item.oofCandidates)
		}
	}

	absolutePositionInset := resolvedBorder.Add(scrollbarGutter)
	absolutePositionArea := finalOuterSize.Sub(absolutePositionInset.SumAxes())
	absolutePositionOffset := Point[float32]{X: absolutePositionInset.Left, Y: absolutePositionInset.Top}
	output.OofCandidates = candidates
	output.OofPositioningArea = some(OofPositioningArea{Size: absolutePositionArea, Offset: absolutePositionOffset})

	if isScrollContainer {
		if direction.IsRtl() {
			inflowOverflowRect.Right += resolvedPadding.Left
		} else {
			inflowOverflowRect.Right += resolvedPadding.Right
		}
		inflowOverflowRect.Bottom += resolvedPadding.Bottom
	}
	output.ScrollableOverflowRect = inflowOverflowRect

	length := tree.ChildCount(nodeID)
	for order := 0; order < length; order++ {
		child := tree.GetChildID(nodeID, order)
		childStyle := tree.GetBlockChildStyle(child)
		if childStyle.BoxGenerationMode() == BoxGenerationModeNone {
			layout := LayoutWithOrder(uint32(order))
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
		}
	}

	return output
}

func generateItemList(tree LayoutBlockContainer, node NodeID, nodeInnerSize Size[Option[float32]]) []blockItem {
	calcFn := treeCalc(tree)
	var items []blockItem
	order := 0
	for childNodeID := range tree.ChildIds(node) {
		childStyle := tree.GetBlockChildStyle(childNodeID)
		if childStyle.BoxGenerationMode() == BoxGenerationModeNone {
			continue
		}
		aspectRatio := childStyle.AspectRatio()
		padding := RectResolveOrZeroOptionF32(childStyle.Padding(), nodeInnerSize.Width, calcFn)
		border := RectResolveOrZeroOptionF32(childStyle.Border(), nodeInnerSize.Width, calcFn)
		pbSum := padding.Add(border).SumAxes()
		boxSizingAdjustment := SizeF32Zero
		if childStyle.BoxSizing() == BoxSizingContentBox {
			boxSizingAdjustment = pbSum
		}

		position := childStyle.Position()
		overflow := childStyle.Overflow()

		float := childStyle.Float()
		isNotFloated := float == FloatNone

		isBlock := childStyle.IsBlock()
		isTable := childStyle.IsTable()
		isCompressibleReplaced := childStyle.IsCompressibleReplaced()
		isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()
		contain := childStyle.Contain()

		isInSameBfc := isBlock &&
			!isTable &&
			!position.IsOutOfFlow() &&
			isNotFloated &&
			!isScrollContainer &&
			!contain.EstablishesIndependentFormattingContext() &&
			childStyle.AlignContent().keyword == AlignContentKeywordNormal

		items = append(items, blockItem{
			nodeID:                 childNodeID,
			order:                  uint32(order),
			isTable:                isTable,
			isCompressibleReplaced: isCompressibleReplaced,
			isInSameBfc:            isInSameBfc,
			float:                  float,
			clear:                  childStyle.Clear(),
			sizeStyle:              childStyle.Size(),
			size: SizeOptionF32MaybeAddF32(
				SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(childStyle.Size(), nodeInnerSize, calcFn), aspectRatio),
				boxSizingAdjustment,
			),
			minSize: SizeOptionF32MaybeAddF32(
				SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(childStyle.MinSize(), nodeInnerSize, calcFn), aspectRatio),
				boxSizingAdjustment,
			),
			maxSize: SizeOptionF32MaybeAddF32(
				SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(childStyle.MaxSize(), nodeInnerSize, calcFn), aspectRatio),
				boxSizingAdjustment,
			),
			overflow:              overflow,
			contain:               contain,
			scrollbarWidth:        childStyle.ScrollbarWidth(),
			position:              position,
			direction:             childStyle.Direction(),
			alignSelf:             childStyle.AlignSelf(),
			justifySelf:           childStyle.JustifySelf(),
			inset:                 childStyle.Inset(),
			margin:                childStyle.Margin(),
			padding:               padding,
			border:                border,
			paddingBorderSum:      pbSum,
			computedSize:          SizeF32Zero,
			staticPosition:        PointF32Zero,
			canBeCollapsedThrough: false,
			finalLayout:           none[Layout](),
			oofCandidates:         NewOofCandidates(),
		})
		order++
	}
	return items
}

func resolveStretchHeight(heightStyle Dimension, containerInnerHeight Option[float32], itemMargin Rect[float32], marginsTreatedAsZero Line[bool]) Option[float32] {
	if !heightStyle.IsStretch() {
		return none[float32]()
	}
	height, ok := containerInnerHeight.Get()
	if !ok {
		return none[float32]()
	}
	if !marginsTreatedAsZero.Start {
		height -= itemMargin.Top
	}
	if !marginsTreatedAsZero.End {
		height -= itemMargin.Bottom
	}
	return some(height)
}

func determineContentBasedContainerWidth(tree LayoutPartialTree, items []blockItem, availableWidth AvailableSpace) float32 {
	calcFn := treeCalc(tree)
	availableSpace := Size[AvailableSpace]{Width: availableWidth, Height: AvailableSpaceMinContent}

	var maxChildWidth float32 = 0.0
	floatContribution := NewFloatIntrinsicWidthCalculator(availableWidth)
	for i := range items {
		item := &items[i]
		if item.position.IsOutOfFlow() {
			continue
		}
		knownDimensions := SizeOptionF32MaybeClamp(item.size, item.minSize, item.maxSize)

		itemXMarginSum := RectResolveOrZeroOptionF32(item.margin, availableSpace.Width.IntoOption(), calcFn).HorizontalAxisSum()
		width, ok := knownDimensions.Width.Get()
		if !ok {
			var itemAvailableWidth AvailableSpace
			if resolution, rok := resolveSizingKeyword(item.sizeStyle.Width, none[float32](), none[float32](), calcFn).Get(); rok {
				if resolution.kind == sizingKeywordResolutionKindMeasure {
					itemAvailableWidth = resolution.measure
				} else {
					itemAvailableWidth = AvailableSpaceDefinite(resolution.exact)
				}
			} else {
				itemAvailableWidth = availableSpace.Width.MaybeSubF32(itemXMarginSum)
			}
			collapsible := LineBoolFalse
			if item.isInSameBfc {
				collapsible = LineBoolTrue
			}
			width = measureChildSize(
				tree,
				item.nodeID,
				knownDimensions,
				SizeOptionF32None,
				Size[AvailableSpace]{Width: itemAvailableWidth, Height: availableSpace.Height},
				SizingModeInherentSize,
				AbsoluteAxisHorizontal,
				collapsible,
			)
		}

		width = f32Max(width, item.paddingBorderSum.Width) + itemXMarginSum

		if direction, dok := item.float.FloatDirection().Get(); dok {
			floatContribution.AddFloat(width, direction, item.clear)
			continue
		}

		maxChildWidth = f32Max(maxChildWidth, width)
	}

	maxChildWidth = f32Max(maxChildWidth, floatContribution.Result())

	return maxChildWidth
}

func performFinalLayoutOnInFlowChildren(
	tree LayoutBlockContainer,
	runMode RunMode,
	items []blockItem,
	containerOuterWidth float32,
	containerPercentageResolutionHeight Option[float32],
	contentBoxInset Rect[float32],
	resolvedContentBoxInset Rect[float32],
	resolvedBorder Rect[float32],
	textAlign TextAlign,
	justifyItems AlignItems,
	direction Direction,
	ownMarginsCollapseWithChildren Line[bool],
	childrenMarginsAdjoinOwnEdges Line[bool],
	isScrollContainer bool,
	blockCtx *BlockContext,
) (Rect[float32], float32, CollapsibleMarginSet, CollapsibleMarginSet, Option[float32]) {
	calcFn := treeCalc(tree)
	negInf := float32(math.Inf(-1))

	containerInnerWidth := f32Max(containerOuterWidth-resolvedContentBoxInset.HorizontalAxisSum(), 0.0)
	containerPercentageResolutionHeight = OptionF32MaybeSubF32(containerPercentageResolutionHeight, resolvedContentBoxInset.VerticalAxisSum())
	parentSize := Size[Option[float32]]{Width: some(containerInnerWidth), Height: containerPercentageResolutionHeight}
	availableSpace := Size[AvailableSpace]{Width: AvailableSpaceDefinite(containerInnerWidth), Height: AvailableSpaceMaxContent}

	if blockCtx.IsBfcRoot() {
		blockCtx.SetWidth(containerOuterWidth)
		blockCtx.ApplyContentBoxInset([2]float32{resolvedContentBoxInset.Left, resolvedContentBoxInset.Right})
	}

	if !ownMarginsCollapseWithChildren.Start {
		blockCtx.commitStrut()
	}

	inflowOverflowRect := RectF32Zero
	committedYOffset := resolvedContentBoxInset.Top
	yOffsetForAbsolute := resolvedContentBoxInset.Top
	firstChildTopMarginSet := CollapsibleMarginSetZero
	activeCollapsibleMarginSet := CollapsibleMarginSetZero
	isCollapsingWithFirstMarginSet := true
	firstBaseline := none[float32]()
	activeMarginSetHasClearance := false

	hasActiveFloats := blockCtx.HasActiveFloats(committedYOffset)

	for i := range items {
		item := &items[i]
		if item.position.IsOutOfFlow() {
			var x float32
			switch direction {
			case DirectionRtl:
				x = containerOuterWidth - resolvedContentBoxInset.Right
			default:
				x = resolvedContentBoxInset.Left
			}
			item.staticPosition = Point[float32]{X: x, Y: yOffsetForAbsolute}
			continue
		}

		itemMargin := item.margin.Map(func(margin LengthPercentageAuto) Option[float32] {
			return margin.ResolveToOption(containerInnerWidth, calcFn)
		})
		itemNonAutoMargin := itemMargin.Map(func(m Option[float32]) float32 { return m.UnwrapOr(0.0) })
		itemNonAutoXMarginSum := itemNonAutoMargin.HorizontalAxisSum()

		scrollbarSize := Size[float32]{}
		if item.overflow.Y == OverflowScroll {
			scrollbarSize.Width = item.scrollbarWidth
		}
		if item.overflow.X == OverflowScroll {
			scrollbarSize.Height = item.scrollbarWidth
		}

		if floatDirection, ok := item.float.FloatDirection().Get(); ok {
			hasActiveFloats = true

			availableWidth := f32Max(containerInnerWidth-itemNonAutoXMarginSum, 0.0)
			itemKnownWidth := none[float32]()
			itemAvailableWidth := AvailableSpaceDefinite(availableWidth)
			if resolution, rok := resolveSizingKeyword(item.sizeStyle.Width, some(availableWidth), some(containerInnerWidth), calcFn).Get(); rok {
				if resolution.kind == sizingKeywordResolutionKindMeasure {
					itemAvailableWidth = resolution.measure
				} else {
					itemKnownWidth = some(resolution.exact)
					itemAvailableWidth = AvailableSpaceDefinite(resolution.exact)
				}
			}
			itemKnownHeight := resolveStretchHeight(
				item.sizeStyle.Height,
				containerPercentageResolutionHeight,
				itemNonAutoMargin,
				LineBoolFalse,
			)
			itemLayout := performChildLayout(
				tree,
				item.nodeID,
				Size[Option[float32]]{Width: itemKnownWidth, Height: itemKnownHeight},
				parentSize,
				Size[AvailableSpace]{Width: itemAvailableWidth, Height: AvailableSpaceMaxContent},
				SizingModeInherentSize,
				LineBoolFalse,
			)
			item.oofCandidates = itemLayout.OofCandidates.Take()
			marginBox := itemLayout.Size.Add(itemNonAutoMargin.SumAxes())

			adjoinsUnresolvedStrut := isCollapsingWithFirstMarginSet && ownMarginsCollapseWithChildren.Start
			var yOffsetForFloat float32
			if adjoinsUnresolvedStrut {
				yOffsetForFloat = committedYOffset
			} else {
				yOffsetForFloat = committedYOffset + activeCollapsibleMarginSet.Resolve()
			}

			location := blockCtx.PlaceFloatedBox(
				marginBox,
				yOffsetForFloat,
				floatDirection,
				item.clear,
				adjoinsUnresolvedStrut,
			)

			location.Y += itemNonAutoMargin.Top
			location.X += itemNonAutoMargin.Left

			item.finalLayout = some(Layout{
				Order:                  item.order,
				Size:                   itemLayout.Size,
				ScrollableOverflowRect: itemLayout.ScrollableOverflowRect,
				ScrollbarSize:          scrollbarSize,
				Location:               location,
				Padding:                item.padding,
				Border:                 item.border,
				Margin:                 itemNonAutoMargin,
			})

			var contributionLocation Point[float32]
			if direction.IsRtl() {
				contributionLocation = Point[float32]{
					X: containerOuterWidth - (location.X + itemLayout.Size.Width) - resolvedBorder.Right,
					Y: location.Y - resolvedBorder.Top,
				}
			} else {
				contributionLocation = Point[float32]{X: location.X - resolvedBorder.Left, Y: location.Y - resolvedBorder.Top}
			}
			inflowOverflowRect = inflowOverflowRect.Union(computeScrollableOverflowContribution(
				contributionLocation,
				itemLayout.Size,
				itemLayout.ScrollableOverflowRect,
				item.overflow,
				item.contain,
				isScrollContainer,
			))

			continue
		}

		var yMarginOffset float32 = 0.0
		itemAvoidsFloats := false
		itemPushedBelowFloat := false

		var stretchWidth, floatAvoidingWidth float32
		var floatAvoidingPosition Point[float32]
		if item.isInSameBfc {
			stretchWidth = f32Max(containerInnerWidth-itemNonAutoXMarginSum, 0.0)
			floatAvoidingPosition = Point[float32]{X: 0.0, Y: 0.0}
			floatAvoidingWidth = 0.0
		} else {
			if !isCollapsingWithFirstMarginSet || !ownMarginsCollapseWithChildren.Start {
				yMarginOffset = activeCollapsibleMarginSet.CollapseWithMargin(itemNonAutoMargin.Top).Resolve()
			}
			minY := committedYOffset + yMarginOffset

			if hasActiveFloats || blockCtx.HasActiveFloats(minY) {
				xMargins := [2]float32{itemNonAutoMargin.Left, itemNonAutoMargin.Right}
				minAutoWidth := -itemNonAutoXMarginSum

				slotSegment := none[int]()
				var slot BfcSlot
				for {
					slot = blockCtx.FindBfcSlot(minY, xMargins, direction, item.clear, slotSegment)
					segmentID, sok := slot.SegmentID.Get()
					if !sok {
						break
					}
					width := F32MaybeClamp(
						item.size.Width.UnwrapOr(f32Max(f32Max(slot.StretchWidth, minAutoWidth), 0.0)),
						item.minSize.Width,
						item.maxSize.Width,
					)
					if width <= slot.BorderWidth+fitTolerance {
						break
					}
					slotSegment = some(segmentID)
				}

				if slot.Y > minY {
					itemPushedBelowFloat = true
				}

				hasActiveFloats = slot.SegmentID.IsSome()
				itemAvoidsFloats = true
				stretchWidth = f32Max(f32Max(slot.StretchWidth, minAutoWidth), 0.0)
				floatAvoidingPosition = Point[float32]{X: slot.X, Y: slot.Y}
				floatAvoidingWidth = slot.BorderWidth
			} else if !hasActiveFloats {
				stretchWidth = f32Max(containerInnerWidth-itemNonAutoXMarginSum, 0.0)
				floatAvoidingPosition = Point[float32]{X: resolvedContentBoxInset.Left, Y: minY}
				floatAvoidingWidth = containerInnerWidth
			} else {
				panic("internal error: entered unreachable code: One of the above cases will always be hit")
			}
		}

		justifySelf := item.justifySelf.UnwrapOr(justifyItems).ResolveSelfRelative(item.direction, direction, true)
		isStretch := justifySelf.keyword == AlignItemsKeywordStretch
		nonStretchJustifySelf := none[AlignItems]()
		switch justifySelf.keyword {
		case AlignItemsKeywordNormal, AlignItemsKeywordStretch:
		default:
			nonStretchJustifySelf = some(justifySelf)
		}

		var knownDimensions Size[Option[float32]]
		if (item.isTable || item.isCompressibleReplaced) && !isStretch {
			knownDimensions = SizeOptionF32None
		} else {
			widthStyle := item.sizeStyle.Width
			if nonStretchJustifySelf.IsSome() && item.sizeStyle.Width.IsAuto() {
				widthStyle = DimensionFitContent()
			}

			keywordWidth := none[float32]()
			if resolution, rok := resolveSizingKeyword(widthStyle, some(stretchWidth), some(containerInnerWidth), calcFn).Get(); rok {
				if resolution.kind == sizingKeywordResolutionKindExact {
					keywordWidth = some(resolution.exact)
				} else {
					collapsible := LineBoolFalse
					if item.isInSameBfc {
						collapsible = LineBoolTrue
					}
					keywordWidth = some(measureChildSize(
						tree,
						item.nodeID,
						SizeOptionF32None,
						parentSize,
						Size[AvailableSpace]{Width: resolution.measure, Height: AvailableSpaceMaxContent},
						SizingModeInherentSize,
						AbsoluteAxisHorizontal,
						collapsible,
					))
				}
			}

			keywordHeight := resolveStretchHeight(
				item.sizeStyle.Height,
				containerPercentageResolutionHeight,
				itemNonAutoMargin,
				childrenMarginsAdjoinOwnEdges,
			)

			knownDimensions = SizeOptionF32MaybeClamp(
				Size[Option[float32]]{
					Width:  some(F32MaybeClamp(item.size.Width.Or(keywordWidth).UnwrapOr(stretchWidth), item.minSize.Width, item.maxSize.Width)),
					Height: item.size.Height.Or(keywordHeight),
				},
				item.minSize,
				item.maxSize,
			)
		}

		collapsible := LineBoolFalse
		if item.isInSameBfc {
			collapsible = LineBoolTrue
		}
		inputs := LayoutInput{
			RunMode:                       runMode,
			SizingMode:                    SizingModeInherentSize,
			Axis:                          RequestedAxisBoth,
			KnownDimensions:               knownDimensions,
			KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
			ParentSize:                    parentSize,
			AvailableSpace:                Size[AvailableSpace]{Width: AvailableSpaceDefinite(stretchWidth), Height: availableSpace.Height},
			VerticalMarginsAreCollapsible: collapsible,
		}

		clearThreshold := blockCtx.ClearedThreshold(item.clear)
		clearPos := clearThreshold.UnwrapOr(negInf)

		childFloatContribution := negInf

		var itemLayout LayoutOutput
		if item.isInSameBfc {
			width := f32Max(knownDimensions.Width.UnwrapOr(stretchWidth), item.paddingBorderSum.Width)

			insetLeft := itemNonAutoMargin.Left + contentBoxInset.Left
			insetRight := containerOuterWidth - width - insetLeft
			insets := [2]float32{insetLeft, insetRight}

			estimatedY := committedYOffset +
				activeCollapsibleMarginSet.CollapseWithMargin(itemNonAutoMargin.Top).Resolve()
			childBlockCtx := blockCtx.SubContext(f32Max(estimatedY, clearPos), insets)
			itemLayout = tree.ComputeBlockChildLayout(item.nodeID, inputs, &childBlockCtx)

			childFloatContribution = childBlockCtx.FloatedContentHeightContribution()
			childTopAdjoiningFloats := childBlockCtx.getTopAdjoiningFloats()
			blockCtx.mergeAdjoiningFloats(childTopAdjoiningFloats)
		} else {
			itemLayout = tree.ComputeChildLayout(item.nodeID, inputs)
		}
		item.oofCandidates = itemLayout.OofCandidates.Take()
		finalSize := itemLayout.Size

		topMarginSet := itemLayout.TopMargin.CollapseWithMargin(itemMargin.Top.UnwrapOr(0.0))
		bottomMarginSet := itemLayout.BottomMargin.CollapseWithMargin(itemMargin.Bottom.UnwrapOr(0.0))

		freeXSpace := f32Max(0.0, stretchWidth-finalSize.Width)
		var xAxisAutoMarginSize float32
		{
			var autoMarginCount uint8
			if itemMargin.Left.IsNone() {
				autoMarginCount++
			}
			if itemMargin.Right.IsNone() {
				autoMarginCount++
			}
			if autoMarginCount > 0 {
				xAxisAutoMarginSize = freeXSpace / float32(autoMarginCount)
			} else {
				xAxisAutoMarginSize = 0.0
			}
		}
		resolvedMargin := Rect[float32]{
			Left:   itemMargin.Left.UnwrapOr(xAxisAutoMarginSize),
			Right:  itemMargin.Right.UnwrapOr(xAxisAutoMarginSize),
			Top:    itemMargin.Top.UnwrapOr(0.0),
			Bottom: itemMargin.Bottom.UnwrapOr(0.0),
		}

		insetPercentageBasis := Size[Option[float32]]{Width: some(containerInnerWidth), Height: containerPercentageResolutionHeight}
		inset := item.inset.zipSize(insetPercentageBasis, func(p LengthPercentageAuto, s Option[float32]) Option[float32] {
			return p.MaybeResolve(s, calcFn)
		})
		negate := func(o Option[float32]) Option[float32] {
			if v, ok := o.Get(); ok {
				return some(-v)
			}
			return o
		}
		insetOffset := PointF32Zero
		if item.position == PositionRelative {
			if direction.IsRtl() {
				insetOffset.X = negate(inset.Right).Or(inset.Left).UnwrapOr(0.0)
			} else {
				insetOffset.X = inset.Left.Or(negate(inset.Right)).UnwrapOr(0.0)
			}
			insetOffset.Y = inset.Top.Or(negate(inset.Bottom)).UnwrapOr(0.0)
		}

		if item.isInSameBfc && (!isCollapsingWithFirstMarginSet || !ownMarginsCollapseWithChildren.Start) {
			yMarginOffset = activeCollapsibleMarginSet.CollapseWithSet(topMarginSet).Resolve()
		}

		hasClearance := false
		if item.isInSameBfc {
			if threshold, ok := clearThreshold.Get(); ok {
				hypotheticalY := committedYOffset + activeCollapsibleMarginSet.CollapseWithSet(topMarginSet).Resolve()
				forcedClearance := blockCtx.HasAdjoiningFloat(item.clear)
				if forcedClearance || hypotheticalY < threshold {
					hasClearance = true
					var escapedMargin float32
					if isCollapsingWithFirstMarginSet && ownMarginsCollapseWithChildren.Start {
						escapedMargin = activeCollapsibleMarginSet.Resolve()
					} else {
						escapedMargin = 0.0
					}
					yMarginOffset = threshold - committedYOffset - escapedMargin
				}
			}
		}

		item.computedSize = itemLayout.Size
		item.canBeCollapsedThrough = itemLayout.MarginsCanCollapseThrough && !hasClearance
		if item.isInSameBfc {
			unclearedY := committedYOffset + activeCollapsibleMarginSet.Resolve()
			var x float32
			switch direction {
			case DirectionRtl:
				x = containerOuterWidth - resolvedContentBoxInset.Right - finalSize.Width
			default:
				x = resolvedContentBoxInset.Left
			}
			item.staticPosition = Point[float32]{X: x, Y: f32Max(unclearedY, clearPos)}
		} else {
			var x float32
			switch direction {
			case DirectionRtl:
				x = floatAvoidingPosition.X + floatAvoidingWidth - finalSize.Width
			default:
				x = floatAvoidingPosition.X
			}
			item.staticPosition = Point[float32]{X: x, Y: floatAvoidingPosition.Y}
		}
		var location Point[float32]
		if item.isInSameBfc {
			switch direction {
			case DirectionRtl:
				location.X = containerOuterWidth -
					resolvedContentBoxInset.Right -
					finalSize.Width -
					resolvedMargin.Right +
					insetOffset.X
			default:
				location.X = resolvedContentBoxInset.Left + insetOffset.X + resolvedMargin.Left
			}
			location.Y = committedYOffset + yMarginOffset + insetOffset.Y
		} else {
			var extraMarginLeft, extraMarginRight float32
			if itemAvoidsFloats {
				extraMarginLeft = resolvedMargin.Left - itemNonAutoMargin.Left
				extraMarginRight = resolvedMargin.Right - itemNonAutoMargin.Right
			} else {
				extraMarginLeft, extraMarginRight = resolvedMargin.Left, resolvedMargin.Right
			}

			switch direction {
			case DirectionRtl:
				location.X = floatAvoidingPosition.X + floatAvoidingWidth - finalSize.Width - extraMarginRight +
					insetOffset.X
			default:
				location.X = floatAvoidingPosition.X + extraMarginLeft + insetOffset.X
			}
			location.Y = floatAvoidingPosition.Y + insetOffset.Y
		}

		if item.isInSameBfc {
			blockCtx.AddChildFloatedContentHeightContribution(
				location.Y - insetOffset.Y + childFloatContribution,
			)
		}

		hasAutoXMargin := itemMargin.Left.IsNone() || itemMargin.Right.IsNone()
		itemOuterWidth := itemLayout.Size.Width + resolvedMargin.HorizontalAxisSum()
		if js, ok := nonStretchJustifySelf.Get(); ok && !hasAutoXMargin {
			var containerStart, freeXSpace float32
			if itemAvoidsFloats {
				containerStart, freeXSpace = floatAvoidingPosition.X-resolvedMargin.Left, floatAvoidingWidth-itemLayout.Size.Width
			} else {
				containerStart, freeXSpace = resolvedContentBoxInset.Left, containerInnerWidth-itemOuterWidth
			}

			keyword := js.keyword
			if js.safety == AlignmentSafetySafe && freeXSpace < 0.0 {
				keyword = AlignItemsKeywordStart
			}
			var offset float32
			switch keyword {
			case AlignItemsKeywordCenter:
				offset = freeXSpace / 2.0
			case AlignItemsKeywordEnd, AlignItemsKeywordFlexEnd, AlignItemsKeywordSelfEnd:
				if !direction.IsRtl() {
					offset = freeXSpace
				} else {
					offset = 0.0
				}
			default:
				if direction.IsRtl() {
					offset = freeXSpace
				} else {
					offset = 0.0
				}
			}
			location.X = containerStart + resolvedMargin.Left + insetOffset.X + offset
		} else if nonStretchJustifySelf.IsNone() && itemOuterWidth < containerInnerWidth {
			freeXSpace := containerInnerWidth - itemOuterWidth
			switch textAlign {
			case TextAlignAuto:
			case TextAlignLegacyLeft:
				if direction == DirectionRtl {
					location.X -= freeXSpace
				}
			case TextAlignLegacyRight:
				if direction == DirectionLtr {
					location.X += freeXSpace
				}
			case TextAlignLegacyCenter:
				if direction == DirectionLtr {
					location.X += freeXSpace / 2.0
				} else {
					location.X -= freeXSpace / 2.0
				}
			}
		}

		if firstBaseline.IsNone() {
			var childBaseline Option[float32]
			if item.overflow.Y.IsScrollContainer() {
				childBaseline = some(f32Max(
					f32Min(itemLayout.Baselines.First.UnwrapOr(itemLayout.Size.Height), itemLayout.Size.Height),
					0.0,
				))
			} else {
				childBaseline = itemLayout.Baselines.First
			}
			if baseline, ok := childBaseline.Get(); ok {
				firstBaseline = some(location.Y + baseline)
			} else {
				firstBaseline = none[float32]()
			}
		}

		usedMargin := resolvedMargin
		if itemAvoidsFloats {
			if itemMargin.Left.IsNone() {
				usedMargin.Left += floatAvoidingPosition.X - resolvedContentBoxInset.Left
			}
			if itemMargin.Right.IsNone() {
				usedMargin.Right += containerOuterWidth -
					resolvedContentBoxInset.Right -
					floatAvoidingPosition.X -
					floatAvoidingWidth
			}
		}

		item.finalLayout = some(Layout{
			Order:                  item.order,
			Size:                   itemLayout.Size,
			ScrollableOverflowRect: itemLayout.ScrollableOverflowRect,
			ScrollbarSize:          scrollbarSize,
			Location:               location,
			Padding:                item.padding,
			Border:                 item.border,
			Margin:                 usedMargin,
		})

		var contributionLocation Point[float32]
		if direction.IsRtl() {
			contributionLocation = Point[float32]{
				X: containerOuterWidth - (location.X + finalSize.Width) - resolvedBorder.Right,
				Y: location.Y - resolvedBorder.Top,
			}
		} else {
			contributionLocation = Point[float32]{X: location.X - resolvedBorder.Left, Y: location.Y - resolvedBorder.Top}
		}
		inflowOverflowRect = inflowOverflowRect.Union(computeScrollableOverflowContribution(
			contributionLocation,
			finalSize,
			itemLayout.ScrollableOverflowRect,
			item.overflow,
			item.contain,
			isScrollContainer,
		))

		if isCollapsingWithFirstMarginSet && itemPushedBelowFloat {
			isCollapsingWithFirstMarginSet = false
		}
		if isCollapsingWithFirstMarginSet && hasClearance {
			isCollapsingWithFirstMarginSet = false
		} else if isCollapsingWithFirstMarginSet {
			if item.canBeCollapsedThrough {
				firstChildTopMarginSet = firstChildTopMarginSet.
					CollapseWithSet(topMarginSet).
					CollapseWithSet(bottomMarginSet)
			} else {
				firstChildTopMarginSet = firstChildTopMarginSet.CollapseWithSet(topMarginSet)
				isCollapsingWithFirstMarginSet = false
			}
		}

		if item.canBeCollapsedThrough {
			activeCollapsibleMarginSet = activeCollapsibleMarginSet.
				CollapseWithSet(topMarginSet).
				CollapseWithSet(bottomMarginSet)
			yOffsetForAbsolute = committedYOffset + itemLayout.Size.Height + yMarginOffset
		} else {
			committedYOffset = location.Y - insetOffset.Y + itemLayout.Size.Height
			if hasClearance && itemLayout.MarginsCanCollapseThrough {
				committedYOffset -= topMarginSet.Resolve()
				activeCollapsibleMarginSet = topMarginSet.CollapseWithSet(bottomMarginSet)
				activeMarginSetHasClearance = true
			} else {
				activeCollapsibleMarginSet = bottomMarginSet
				activeMarginSetHasClearance = false
			}
			yOffsetForAbsolute = committedYOffset + activeCollapsibleMarginSet.Resolve()
			blockCtx.commitStrut()
		}
	}

	lastChildBottomMarginSet := activeCollapsibleMarginSet
	if activeMarginSetHasClearance {
		lastChildBottomMarginSet = CollapsibleMarginSetZero
	}
	var bottomYMarginOffset float32
	switch {
	case activeMarginSetHasClearance:
		bottomYMarginOffset = activeCollapsibleMarginSet.Resolve()
	case ownMarginsCollapseWithChildren.End:
		bottomYMarginOffset = 0.0
	default:
		bottomYMarginOffset = lastChildBottomMarginSet.Resolve()
	}

	committedYOffset += resolvedContentBoxInset.Bottom + bottomYMarginOffset
	contentHeight := f32Max(0.0, committedYOffset)
	return inflowOverflowRect, contentHeight, firstChildTopMarginSet, lastChildBottomMarginSet, firstBaseline
}
