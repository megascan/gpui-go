package taffy

func alignTracks(
	gridContainerContentBoxSize float32,
	padding Line[float32],
	border Line[float32],
	tracks []gridTrack,
	trackAlignmentStyle AlignContent,
	axisIsReversed bool,
) {
	var usedSize float32
	for i := range tracks {
		usedSize += tracks[i].baseSize
	}
	freeSpace := gridContainerContentBoxSize - usedSize
	origin := padding.Start + border.Start

	numTracks := 0
	for i := 1; i < len(tracks); i += 2 {
		if !tracks[i].isCollapsed {
			numTracks++
		}
	}

	const gap float32 = 0.0
	layoutIsReversed := false
	trackAlignment := applyAlignmentFallback(freeSpace, numTracks, trackAlignmentStyle)
	if axisIsReversed {
		trackAlignment = trackAlignment.reversed()
	}

	emptyGridOffset := float32(0.0)
	if numTracks == 0 {
		emptyGridOffset = computeAlignmentOffset(freeSpace, numTracks, gap, trackAlignment, layoutIsReversed, true)
	}

	totalOffset := origin + emptyGridOffset
	seenNonCollapsedTrack := false
	positionTrack := func(i int, track *gridTrack) {
		isGutter := i%2 == 0
		isNonCollapsedTrack := !isGutter && !track.isCollapsed

		isFirst := isNonCollapsedTrack && !seenNonCollapsedTrack

		offset := float32(0.0)
		if isNonCollapsedTrack {
			offset = computeAlignmentOffset(freeSpace, numTracks, gap, trackAlignment, layoutIsReversed, isFirst)
		}

		track.offset = totalOffset + offset
		totalOffset = totalOffset + offset + track.baseSize
		if isNonCollapsedTrack {
			seenNonCollapsedTrack = true
		}
	}
	if axisIsReversed {
		for i := range tracks {
			positionTrack(i, &tracks[len(tracks)-1-i])
		}
	} else {
		for i := range tracks {
			positionTrack(i, &tracks[i])
		}
	}
}

func alignAndPositionItem[S CheapCloneStr](
	tree LayoutGridContainer[S],
	node NodeID,
	order uint32,
	gridArea Rect[float32],
	containingBlockSize Size[float32],
	containerAlignmentStyles inBothAbsAxis[AlignItems],
	baselineShim float32,
	direction Direction,
	containerBorderBoxWidth float32,
	containerBorder Rect[float32],
	containerIsScrollContainer bool,
	bubbledCandidates *OofCandidates,
) (Rect[float32], float32, float32) {
	calcFn := treeCalc(tree)
	gridAreaSize := Size[float32]{Width: gridArea.Right - gridArea.Left, Height: gridArea.Bottom - gridArea.Top}

	style := tree.GetGridChildStyle(node)

	overflow := style.Overflow()
	contain := style.Contain()
	scrollbarWidth := style.ScrollbarWidth()
	aspectRatio := style.AspectRatio()
	isReplaced := style.IsReplaced()
	itemDirection := style.Direction()
	justifySelf := style.JustifySelf().UnwrapOr(containerAlignmentStyles.horizontal).ResolveSelfRelative(itemDirection, direction, true)
	alignSelf := style.AlignSelf().UnwrapOr(containerAlignmentStyles.vertical).ResolveSelfRelative(itemDirection, direction, false)

	position := style.Position()
	insetHorizontal := style.Inset().HorizontalComponents().Map(func(size LengthPercentageAuto) Option[float32] {
		return size.ResolveToOption(containingBlockSize.Width, calcFn)
	})
	insetVertical := style.Inset().VerticalComponents().Map(func(size LengthPercentageAuto) Option[float32] {
		return size.ResolveToOption(containingBlockSize.Height, calcFn)
	})
	padding := style.Padding().Map(func(p LengthPercentage) float32 {
		return p.ResolveOrZero(some(containingBlockSize.Width), calcFn)
	})
	border := style.Border().Map(func(p LengthPercentage) float32 {
		return p.ResolveOrZero(some(containingBlockSize.Width), calcFn)
	})
	paddingBorderSize := padding.Add(border).SumAxes()

	boxSizingAdjustment := SizeF32Zero
	if style.BoxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	containingBlockOpt := containingBlockSize.Map(someF32)
	sizeStyle := style.Size()
	inherentSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(sizeStyle, containingBlockOpt, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	minSize := SizeOptionF32MaybeApplyAspectRatio(
		SizeOptionF32MaybeMaxF32(
			SizeOptionOr(
				SizeOptionF32MaybeAddF32(SizeMaybeResolve(style.MinSize(), containingBlockOpt, calcFn), boxSizingAdjustment),
				paddingBorderSize.Map(someF32),
			),
			paddingBorderSize,
		),
		aspectRatio,
	)
	maxSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.MaxSize(), containingBlockOpt, calcFn), aspectRatio),
		boxSizingAdjustment,
	)

	alignmentStyles := inBothAbsAxis[AlignSelf]{horizontal: justifySelf, vertical: alignSelf}
	if justifySelf.keyword == AlignItemsKeywordNormal {
		if inherentSize.Width.IsSome() || sizeStyle.Width.IsSizingKeyword() || isReplaced {
			alignmentStyles.horizontal = AlignItemsStart
		} else {
			alignmentStyles.horizontal = AlignItemsStretch
		}
	}
	if alignSelf.keyword == AlignItemsKeywordNormal {
		if inherentSize.Height.IsSome() || sizeStyle.Height.IsSizingKeyword() || aspectRatio.IsSome() || isReplaced {
			alignmentStyles.vertical = AlignItemsStart
		} else {
			alignmentStyles.vertical = AlignItemsStretch
		}
	}

	margin := style.Margin().Map(func(m LengthPercentageAuto) Option[float32] {
		return m.ResolveToOption(containingBlockSize.Width, calcFn)
	})

	gridAreaMinusItemMarginsSize := Size[float32]{
		Width:  f32Max(F32MaybeSub(F32MaybeSub(gridAreaSize.Width, margin.Left), margin.Right), 0.0),
		Height: f32Max(F32MaybeSub(F32MaybeSub(gridAreaSize.Height, margin.Top), margin.Bottom)-baselineShim, 0.0),
	}

	keywordWidth := none[Option[sizingKeywordResolution]]()
	if inherentSize.Width.IsNone() {
		keywordWidth = some(resolveSizingKeyword(sizeStyle.Width, some(gridAreaMinusItemMarginsSize.Width), some(containingBlockSize.Width), calcFn))
	}
	keywordHeight := none[Option[sizingKeywordResolution]]()
	if inherentSize.Height.IsNone() {
		keywordHeight = some(resolveSizingKeyword(sizeStyle.Height, some(gridAreaMinusItemMarginsSize.Height), some(containingBlockSize.Height), calcFn))
	}

	measureResolution := func(k Option[Option[sizingKeywordResolution]]) (sizingKeywordResolution, bool) {
		inner, ok := k.Get()
		if !ok {
			return sizingKeywordResolution{}, false
		}
		return inner.Get()
	}

	keywordMeasuredSize := SizeOptionF32None
	if kw, ok := measureResolution(keywordWidth); ok && kw.kind == sizingKeywordResolutionKindMeasure {
		if kh, ok := measureResolution(keywordHeight); ok && kh.kind == sizingKeywordResolutionKindMeasure && !position.IsOutOfFlow() {
			keywordMeasuredSize = measureChildSizeBoth(
				tree,
				node,
				SizeOptionF32None,
				containingBlockOpt,
				Size[AvailableSpace]{Width: kw.measure, Height: kh.measure},
				SizingModeInherentSize,
				LineBoolFalse,
			).Map(someF32)
		}
	}

	width := inherentSize.Width
	if width.IsNone() {
		width = func() Option[float32] {
			if position.IsOutOfFlow() {
				left, lok := insetHorizontal.Start.Get()
				right, rok := insetHorizontal.End.Get()
				if lok && rok {
					return some(f32Max(gridAreaMinusItemMarginsSize.Width-left-right, 0.0))
				}
			}

			if resolution, ok := measureResolution(keywordWidth); ok {
				switch resolution.kind {
				case sizingKeywordResolutionKindExact:
					return some(resolution.exact)
				default:
					if w, ok := keywordMeasuredSize.Width.Get(); ok {
						return some(w)
					}
					return some(measureChildSize(
						tree,
						node,
						SizeOptionF32None,
						containingBlockOpt,
						Size[AvailableSpace]{
							Width:  resolution.measure,
							Height: AvailableSpaceDefinite(gridAreaMinusItemMarginsSize.Height),
						},
						SizingModeInherentSize,
						AbsoluteAxisHorizontal,
						LineBoolFalse,
					))
				}
			}

			if margin.Left.IsSome() && margin.Right.IsSome() && alignmentStyles.horizontal == AlignItemsStretch && !position.IsOutOfFlow() {
				return some(gridAreaMinusItemMarginsSize.Width)
			}

			return none[float32]()
		}()
	}

	sized := SizeOptionF32MaybeApplyAspectRatio(Size[Option[float32]]{Width: width, Height: inherentSize.Height}, aspectRatio)
	width, height := sized.Width, sized.Height

	if height.IsNone() {
		height = func() Option[float32] {
			if position.IsOutOfFlow() {
				top, tok := insetVertical.Start.Get()
				bottom, bok := insetVertical.End.Get()
				if tok && bok {
					return some(f32Max(gridAreaMinusItemMarginsSize.Height-top-bottom, 0.0))
				}
			}

			if resolution, ok := measureResolution(keywordHeight); ok {
				switch resolution.kind {
				case sizingKeywordResolutionKindExact:
					return some(resolution.exact)
				default:
					if h, ok := keywordMeasuredSize.Height.Get(); ok {
						return some(h)
					}
					availableWidth := AvailableSpaceDefinite(gridAreaMinusItemMarginsSize.Width)
					if w, ok := width.Get(); ok {
						availableWidth = AvailableSpaceDefinite(w)
					}
					return some(measureChildSize(
						tree,
						node,
						Size[Option[float32]]{Width: width, Height: none[float32]()},
						containingBlockOpt,
						Size[AvailableSpace]{Width: availableWidth, Height: resolution.measure},
						SizingModeInherentSize,
						AbsoluteAxisVertical,
						LineBoolFalse,
					))
				}
			}

			if margin.Top.IsSome() && margin.Bottom.IsSome() && alignmentStyles.vertical == AlignItemsStretch && !position.IsOutOfFlow() {
				return some(gridAreaMinusItemMarginsSize.Height)
			}

			return none[float32]()
		}()
	}
	sized = SizeOptionF32MaybeApplyAspectRatio(Size[Option[float32]]{Width: width, Height: height}, aspectRatio)
	sized = SizeOptionF32MaybeClamp(sized, minSize, maxSize)
	width, height = sized.Width, sized.Height

	size := Size[Option[float32]]{Width: width, Height: height}
	if position.IsOutOfFlow() && (width.IsNone() || height.IsNone()) {
		size = measureChildSizeBoth(
			tree,
			node,
			Size[Option[float32]]{Width: width, Height: height},
			containingBlockOpt,
			gridAreaMinusItemMarginsSize.Map(AvailableSpaceDefinite),
			SizingModeInherentSize,
			LineBoolFalse,
		).Map(someF32)
	}

	layoutOutput := performChildLayout(
		tree,
		node,
		size,
		containingBlockOpt,
		gridAreaMinusItemMarginsSize.Map(AvailableSpaceDefinite),
		SizingModeInherentSize,
		LineBoolFalse,
	)

	finalSize := SizeF32MaybeClamp(SizeOptionUnwrapOr(size, layoutOutput.Size), minSize, maxSize)
	finalWidth, finalHeight := finalSize.Width, finalSize.Height

	x, xMargin := alignItemWithinArea(
		Line[float32]{Start: gridArea.Left, End: gridArea.Right},
		alignmentStyles.horizontal,
		finalWidth,
		position,
		insetHorizontal,
		margin.HorizontalComponents(),
		0.0,
		direction,
	)
	y, yMargin := alignItemWithinArea(
		Line[float32]{Start: gridArea.Top, End: gridArea.Bottom},
		alignmentStyles.vertical,
		finalHeight,
		position,
		insetVertical,
		margin.VerticalComponents(),
		baselineShim,
		DirectionLtr,
	)

	scrollbarSize := Size[float32]{}
	if overflow.Y == OverflowScroll {
		scrollbarSize.Width = scrollbarWidth
	}
	if overflow.X == OverflowScroll {
		scrollbarSize.Height = scrollbarWidth
	}

	resolvedMargin := Rect[float32]{Left: xMargin.Start, Right: xMargin.End, Top: yMargin.Start, Bottom: yMargin.End}

	itemCandidates := layoutOutput.OofCandidates.Take()
	if !itemCandidates.IsEmpty() {
		itemCandidates.Translate(Point[float32]{X: x, Y: y})
		bubbledCandidates.Append(&itemCandidates)
	}

	tree.SetUnroundedLayout(node, &Layout{
		Order:                  order,
		Location:               Point[float32]{X: x, Y: y},
		Size:                   Size[float32]{Width: finalWidth, Height: finalHeight},
		ScrollableOverflowRect: layoutOutput.ScrollableOverflowRect,
		ScrollbarSize:          scrollbarSize,
		Padding:                padding,
		Border:                 border,
		Margin:                 resolvedMargin,
	})

	var contributionLocation Point[float32]
	if direction.IsRtl() {
		contributionLocation = Point[float32]{X: containerBorderBoxWidth - (x + finalWidth) - containerBorder.Right, Y: y - containerBorder.Top}
	} else {
		contributionLocation = Point[float32]{X: x - containerBorder.Left, Y: y - containerBorder.Top}
	}
	contribution := computeScrollableOverflowContribution(
		contributionLocation,
		Size[float32]{Width: finalWidth, Height: finalHeight},
		layoutOutput.ScrollableOverflowRect,
		overflow,
		contain,
		containerIsScrollContainer,
	)

	return contribution, y, finalHeight
}

func alignItemWithinArea(
	gridArea Line[float32],
	alignmentStyle AlignSelf,
	resolvedSize float32,
	position Position,
	inset Line[Option[float32]],
	margin Line[Option[float32]],
	baselineShim float32,
	direction Direction,
) (float32, Line[float32]) {
	nonAutoMargin := Line[float32]{Start: margin.Start.UnwrapOr(0.0) + baselineShim, End: margin.End.UnwrapOr(0.0)}
	gridAreaSize := f32Max(gridArea.End-gridArea.Start, 0.0)
	freeSpace := f32Max(gridAreaSize-resolvedSize-nonAutoMargin.Sum(), 0.0)

	autoMarginCount := 0
	if margin.Start.IsNone() {
		autoMarginCount++
	}
	if margin.End.IsNone() {
		autoMarginCount++
	}
	autoMarginSize := float32(0.0)
	if autoMarginCount > 0 {
		autoMarginSize = freeSpace / float32(autoMarginCount)
	}
	resolvedMargin := Line[float32]{
		Start: margin.Start.UnwrapOr(autoMarginSize) + baselineShim,
		End:   margin.End.UnwrapOr(autoMarginSize),
	}

	overflows := resolvedSize+nonAutoMargin.Sum() > gridAreaSize
	alignmentKeyword := resolveSelfAlignmentSafety(alignmentStyle, overflows)

	var alignmentBasedOffset float32
	switch alignmentKeyword {
	case AlignItemsKeywordStart, AlignItemsKeywordFlexStart, AlignItemsKeywordBaseline, AlignItemsKeywordStretch:
		if direction.IsRtl() {
			alignmentBasedOffset = gridAreaSize - resolvedSize - resolvedMargin.End
		} else {
			alignmentBasedOffset = resolvedMargin.Start
		}
	case AlignItemsKeywordEnd, AlignItemsKeywordFlexEnd:
		if direction.IsRtl() {
			alignmentBasedOffset = resolvedMargin.Start
		} else {
			alignmentBasedOffset = gridAreaSize - resolvedSize - resolvedMargin.End
		}
	case AlignItemsKeywordCenter:
		alignmentBasedOffset = (gridAreaSize - resolvedSize + resolvedMargin.Start - resolvedMargin.End) / 2.0
	default:
		panic("internal error: entered unreachable code")
	}

	offsetWithinArea := alignmentBasedOffset
	if position.IsOutOfFlow() {
		start, sok := inset.Start.Get()
		end, eok := inset.End.Get()
		switch {
		case sok && eok:
			if direction.IsRtl() {
				offsetWithinArea = gridAreaSize - end - resolvedSize - nonAutoMargin.End
			} else {
				offsetWithinArea = start + nonAutoMargin.Start
			}
		case sok:
			offsetWithinArea = start + nonAutoMargin.Start
		case eok:
			offsetWithinArea = gridAreaSize - end - resolvedSize - nonAutoMargin.End
		}
	}

	start := gridArea.Start + offsetWithinArea
	if position == PositionRelative {
		var relativeInset Option[float32]
		if direction.IsRtl() {
			if e, ok := inset.End.Get(); ok {
				relativeInset = some(-e)
			} else {
				relativeInset = inset.Start
			}
		} else {
			if s, ok := inset.Start.Get(); ok {
				relativeInset = some(s)
			} else if e, ok := inset.End.Get(); ok {
				relativeInset = some(-e)
			}
		}
		start += relativeInset.UnwrapOr(0.0)
	}

	return start, resolvedMargin
}
