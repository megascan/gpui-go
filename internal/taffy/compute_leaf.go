package taffy

import "unsafe"

// ComputeLeafLayout computes the size of a leaf node (a node with no children).
// A definite inputs.AvailableSpace is the space available to the node's border box: the caller has already subtracted the node's margins.
func ComputeLeafLayout(
	inputs LayoutInput,
	style CoreStyle,
	resolveCalcValue func(unsafe.Pointer, float32) float32,
	measureFunction func(knownDimensions Size[Option[float32]], availableSpace Size[AvailableSpace]) Size[float32],
) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	sizingMode := inputs.SizingMode
	runMode := inputs.RunMode

	padding := RectResolveOrZeroOptionF32(style.Padding(), parentSize.Width, resolveCalcValue)
	border := RectResolveOrZeroOptionF32(style.Border(), parentSize.Width, resolveCalcValue)
	paddingBorder := padding.Add(border)
	pbSum := paddingBorder.SumAxes()
	boxSizingAdjustment := SizeF32Zero
	if style.BoxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = pbSum
	}

	var nodeSize, nodeMinSize, nodeMaxSize Size[Option[float32]]
	var aspectRatio Option[float32]
	switch sizingMode {
	case SizingModeContentSize:
		nodeSize = knownDimensions
		nodeMinSize = SizeOptionF32None
		nodeMaxSize = SizeOptionF32None
		aspectRatio = none[float32]()
	case SizingModeInherentSize:
		aspectRatio = style.AspectRatio()
		styleSize := SizeOptionF32MaybeAddF32(
			SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.Size(), parentSize, resolveCalcValue), aspectRatio),
			boxSizingAdjustment,
		)
		styleMinSize := SizeOptionF32MaybeAddF32(
			SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MinSize(), parentSize, resolveCalcValue), aspectRatio),
			boxSizingAdjustment,
		)
		styleMaxSize := SizeOptionF32MaybeAddF32(SizeMaybeResolve(style.MaxSize(), parentSize, resolveCalcValue), boxSizingAdjustment)

		nodeSize = SizeOptionOr(knownDimensions, styleSize)
		nodeMinSize = styleMinSize
		nodeMaxSize = styleMaxSize
	}

	scrollbarGutter := style.Overflow().Transpose().Map(func(overflow Overflow) float32 {
		if overflow == OverflowScroll {
			return style.ScrollbarWidth()
		}
		return 0.0
	})
	contentBoxInset := paddingBorder
	contentBoxInset.Right += scrollbarGutter.X
	contentBoxInset.Bottom += scrollbarGutter.Y

	optPositive := func(o Option[float32]) bool {
		v, ok := o.Get()
		return ok && v > 0.0
	}
	hasStylesPreventingBeingCollapsedThrough := !style.IsBlock() ||
		style.Overflow().X.IsScrollContainer() ||
		style.Overflow().Y.IsScrollContainer() ||
		style.Position().IsOutOfFlow() ||
		style.Contain().EstablishesIndependentFormattingContext() ||
		padding.Top > 0.0 ||
		padding.Bottom > 0.0 ||
		border.Top > 0.0 ||
		border.Bottom > 0.0 ||
		optPositive(nodeSize.Height) ||
		optPositive(nodeMinSize.Height)

	someF32 := func(v float32) Option[float32] { return some(v) }

	if runMode == RunModeComputeSize && hasStylesPreventingBeingCollapsedThrough {
		width, wok := nodeSize.Width.Get()
		height, hok := nodeSize.Height.Get()
		if wok && hok {
			size := SizeF32MaybeMax(
				SizeF32MaybeClamp(Size[float32]{Width: width, Height: height}, nodeMinSize, nodeMaxSize),
				paddingBorder.SumAxes().Map(someF32),
			)
			return LayoutOutput{
				Size:                      size,
				ScrollableOverflowRect:    RectF32Zero,
				Baselines:                 BaselinesNone,
				TopMargin:                 CollapsibleMarginSetZero,
				BottomMargin:              CollapsibleMarginSetZero,
				MarginsCanCollapseThrough: false,
				OofCandidates:             OofCandidatesNone,
				OofPositioningArea:        none[OofPositioningArea](),
			}
		}
	}

	availableSpaceFor := func(known Option[float32], avail AvailableSpace, styled Option[float32], min, max Option[float32], inset float32) AvailableSpace {
		base := avail
		if v, ok := known.Get(); ok {
			base = AvailableSpaceFromF32(v)
		}
		return base.MaybeSet(known).MaybeSet(styled).MapDefiniteValue(func(size float32) float32 {
			return F32MaybeClamp(size, min, max) - inset
		})
	}
	availableSpace = Size[AvailableSpace]{
		Width:  availableSpaceFor(knownDimensions.Width, availableSpace.Width, nodeSize.Width, nodeMinSize.Width, nodeMaxSize.Width, contentBoxInset.HorizontalAxisSum()),
		Height: availableSpaceFor(knownDimensions.Height, availableSpace.Height, nodeSize.Height, nodeMinSize.Height, nodeMaxSize.Height, contentBoxInset.VerticalAxisSum()),
	}

	var measureKnown Size[Option[float32]]
	switch runMode {
	case RunModeComputeSize:
		measureKnown = knownDimensions
	case RunModePerformLayout:
		measureKnown = SizeOptionF32None
	default:
		panic("internal error: entered unreachable code")
	}
	measuredSize := measureFunction(measureKnown, availableSpace)
	clampedSize := SizeF32MaybeClamp(
		SizeOptionUnwrapOr(SizeOptionOr(knownDimensions, nodeSize), measuredSize.Add(contentBoxInset.SumAxes())),
		nodeMinSize,
		nodeMaxSize,
	)
	aspectHeight := float32(0.0)
	if ratio, ok := aspectRatio.Get(); ok {
		aspectHeight = clampedSize.Width / ratio
	}
	size := Size[float32]{
		Width:  clampedSize.Width,
		Height: f32Max(clampedSize.Height, aspectHeight),
	}
	size = SizeF32MaybeMax(size, paddingBorder.SumAxes().Map(someF32))

	isScrollContainer := style.Overflow().X.IsScrollContainer() || style.Overflow().Y.IsScrollContainer()
	isRtl := style.Direction().IsRtl()
	startPadding, endPadding := padding.Left, padding.Right
	if isRtl {
		startPadding, endPadding = padding.Right, padding.Left
	}
	endPaddingX, endPaddingY := float32(0.0), float32(0.0)
	if isScrollContainer {
		endPaddingX, endPaddingY = endPadding, padding.Bottom
	}
	scrollableOverflowRect := Rect[float32]{
		Left:   0.0,
		Right:  startPadding + measuredSize.Width + endPaddingX,
		Top:    0.0,
		Bottom: padding.Top + measuredSize.Height + endPaddingY,
	}

	return LayoutOutput{
		Size:                   size,
		ScrollableOverflowRect: scrollableOverflowRect,
		Baselines:              BaselinesNone,
		TopMargin:              CollapsibleMarginSetZero,
		BottomMargin:           CollapsibleMarginSetZero,
		MarginsCanCollapseThrough: !hasStylesPreventingBeingCollapsedThrough &&
			size.Height == 0.0 &&
			measuredSize.Height == 0.0,
		OofCandidates:      OofCandidatesNone,
		OofPositioningArea: none[OofPositioningArea](),
	}
}
