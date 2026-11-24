package taffy

import (
	"math"

	"github.com/megascan/gpui-go/gpuiutil"
)

// ComputeOofLayout lays out the out-of-flow candidates in output for which nodeID is the containing block,
// records them as its hoisted children and leaves the unclaimed rest in output. It is a no-op when
// output.OofPositioningArea is None.
func ComputeOofLayout[S CheapCloneStr](tree LayoutContainingBlock[S], nodeID NodeID, output *LayoutOutput) {
	area, ok := output.OofPositioningArea.Get()
	if !ok {
		return
	}

	style := tree.GetCoreContainerStyle(nodeID)
	direction := style.Direction()
	claims := style.IsContainingBlock()

	candidates := output.OofCandidates.Take()
	result := ComputeOofLayoutForArea(tree, nodeID, candidates.AsSlice(), area, direction, claims)
	tree.ClearHoistedChildren(nodeID)
	tree.AddHoistedChildren(nodeID, result.Hoisted)
	output.OofCandidates = result.Unclaimed
	output.ScrollableOverflowRect = output.ScrollableOverflowRect.Union(result.ScrollableOverflowRect)
}

// ResolveStaticOffset resolves the final static offset of an out-of-flow box from its static position, given the
// box's final size and resolved margins.
func ResolveStaticOffset(staticPosition Point[AxisStaticPosition], finalSize Size[float32], resolvedMargin Rect[float32]) Point[float32] {
	return Point[float32]{
		X: resolveStaticOffsetAxis(
			staticPosition.X,
			finalSize.Width,
			Line[float32]{Start: resolvedMargin.Left, End: resolvedMargin.Right},
		),
		Y: resolveStaticOffsetAxis(
			staticPosition.Y,
			finalSize.Height,
			Line[float32]{Start: resolvedMargin.Top, End: resolvedMargin.Bottom},
		),
	}
}

func resolveStaticOffsetAxis(sp AxisStaticPosition, size float32, margin Line[float32]) float32 {
	marginBoxSize := size + margin.Start + margin.End
	overflows := sp.Align.Safety == AlignmentSafetySafe && marginBoxSize > sp.Area.End-sp.Area.Start
	keyword := sp.Align.Keyword
	if overflows {
		keyword = sp.Align.Fallback
	}
	return alignInLine(sp.Area, marginBoxSize, keyword) + margin.Start
}

func alignInLine(container Line[float32], size float32, edge AxisStaticEdge) float32 {
	switch edge {
	case AxisStaticEdgeStart:
		return container.Start
	case AxisStaticEdgeEnd:
		return container.End - size
	default:
		return (container.Start + container.End - size) / 2.0
	}
}

type oofAlignment uint8

const (
	oofAlignmentNormal oofAlignment = iota
	oofAlignmentStretch
	oofAlignmentStart
	oofAlignmentCenter
	oofAlignmentEnd
)

func oofAlignmentFromResolved(alignment AlignSelf) oofAlignment {
	if gpuiutil.DebugAssertions && (alignment.keyword == AlignItemsKeywordSelfStart || alignment.keyword == AlignItemsKeywordSelfEnd) {
		panic("self-start/self-end must be resolved before computing out-of-flow alignment")
	}
	switch alignment.keyword {
	case AlignItemsKeywordNormal:
		return oofAlignmentNormal
	case AlignItemsKeywordStretch:
		return oofAlignmentStretch
	case AlignItemsKeywordCenter:
		return oofAlignmentCenter
	case AlignItemsKeywordEnd, AlignItemsKeywordFlexEnd, AlignItemsKeywordSelfEnd:
		return oofAlignmentEnd
	default:
		return oofAlignmentStart
	}
}

func (a oofAlignment) physicalEdge(reversed bool) AxisStaticEdge {
	switch {
	case a == oofAlignmentCenter:
		return AxisStaticEdgeCenter
	case (a == oofAlignmentEnd) != reversed:
		return AxisStaticEdgeEnd
	default:
		return AxisStaticEdgeStart
	}
}

type oofAxis struct {
	cbSize         float32
	inset          Line[Option[float32]]
	margin         Line[Option[float32]]
	staticPosition AxisStaticPosition
	alignment      oofAlignment
	safety         AlignmentSafety
	isInline       bool
	reversed       bool
	isScrollAxis   bool
}

func (a *oofAxis) startEdge() AxisStaticEdge {
	if a.reversed {
		return AxisStaticEdgeEnd
	}
	return AxisStaticEdgeStart
}

func (a *oofAxis) endEdge() AxisStaticEdge {
	if a.reversed {
		return AxisStaticEdgeStart
	}
	return AxisStaticEdgeEnd
}

func (a *oofAxis) insetModifiedContainingBlock() Line[float32] {
	var imcb Line[float32]
	var weakerEdge AxisStaticEdge
	start, sok := a.inset.Start.Get()
	end, eok := a.inset.End.Get()
	switch {
	case sok && eok:
		imcb, weakerEdge = Line[float32]{Start: start, End: a.cbSize - end}, a.endEdge()
	case sok:
		imcb, weakerEdge = Line[float32]{Start: start, End: a.cbSize}, AxisStaticEdgeEnd
	case eok:
		imcb, weakerEdge = Line[float32]{Start: 0.0, End: a.cbSize - end}, AxisStaticEdgeStart
	default:
		sp := a.staticPosition
		switch sp.Align.Keyword {
		case AxisStaticEdgeStart:
			imcb = Line[float32]{Start: sp.Area.Start, End: a.cbSize}
		case AxisStaticEdgeEnd:
			imcb = Line[float32]{Start: 0.0, End: sp.Area.End}
		default:
			center := (sp.Area.Start + sp.Area.End) / 2.0
			startDistance := center
			endDistance := a.cbSize - center
			if abs(startDistance) <= abs(endDistance) {
				imcb = Line[float32]{Start: 0.0, End: 2.0 * startDistance}
			} else {
				imcb = Line[float32]{Start: a.cbSize - float32(2.0*endDistance), End: a.cbSize}
			}
		}
		weakerEdge = a.endEdge()
	}

	if imcb.End < imcb.Start {
		if weakerEdge == AxisStaticEdgeStart {
			imcb.Start = imcb.End
		} else {
			imcb.End = imcb.Start
		}
	}
	return imcb
}

func (a *oofAxis) insetModifiedContainingBlockSize() float32 {
	imcb := a.insetModifiedContainingBlock()
	return imcb.End - imcb.Start
}

func (a *oofAxis) stretchFitSize() float32 {
	nonAutoMarginSum := a.margin.Start.UnwrapOr(0.0) + a.margin.End.UnwrapOr(0.0)
	return f32Max(a.insetModifiedContainingBlockSize()-nonAutoMarginSum, 0.0)
}

func (a *oofAxis) isStretchSized(isReplaced, isTable bool) bool {
	noAutoInset := a.inset.Start.IsSome() && a.inset.End.IsSome()
	if !noAutoInset {
		return false
	}
	switch a.alignment {
	case oofAlignmentStretch:
		return true
	case oofAlignmentNormal:
		return !isReplaced && !isTable
	default:
		return false
	}
}

func (a *oofAxis) resolveMargins(size float32) Line[float32] {
	bothInsets := a.inset.Start.IsSome() && a.inset.End.IsSome()
	imcb := a.insetModifiedContainingBlock()
	freeSpace := imcb.End - imcb.Start - size
	start, sok := a.margin.Start.Get()
	end, eok := a.margin.End.Get()
	switch {
	case sok && eok:
		return Line[float32]{Start: start, End: end}
	case !bothInsets:
		return Line[float32]{Start: a.margin.Start.UnwrapOr(0.0), End: a.margin.End.UnwrapOr(0.0)}
	case sok:
		return Line[float32]{Start: start, End: freeSpace - start}
	case eok:
		return Line[float32]{Start: freeSpace - end, End: end}
	default:
		if freeSpace < 0.0 && a.isInline {
			if a.reversed {
				return Line[float32]{Start: freeSpace, End: 0.0}
			}
			return Line[float32]{Start: 0.0, End: freeSpace}
		}
		return Line[float32]{Start: freeSpace / 2.0, End: freeSpace / 2.0}
	}
}

func (a *oofAxis) position(size float32, margin Line[float32]) float32 {
	imcb := a.insetModifiedContainingBlock()
	sok := a.inset.Start.IsSome()
	eok := a.inset.End.IsSome()
	switch {
	case !sok && !eok:
		sp := a.staticPosition
		marginBoxSize := size + margin.Start + margin.End
		var offset float32
		if sp.Align.Safety == AlignmentSafetySafe && marginBoxSize > imcb.End-imcb.Start {
			offset = alignInLine(imcb, marginBoxSize, a.startEdge())
		} else {
			offset = alignInLine(sp.Area, marginBoxSize, sp.Align.Keyword)
		}
		return offset + margin.Start
	case sok && !eok:
		return imcb.Start + margin.Start
	case !sok && eok:
		return imcb.End - size - margin.End
	default:
		if a.margin.Start.IsNone() || a.margin.End.IsNone() {
			return imcb.Start + margin.Start
		}

		marginBoxSize := size + margin.Start + margin.End
		imcbSize := imcb.End - imcb.Start
		edge := a.alignment.physicalEdge(a.reversed)
		var marginBoxStart float32
		switch a.safety {
		case AlignmentSafetyUnsafe:
			marginBoxStart = alignInLine(imcb, marginBoxSize, edge)
		case AlignmentSafetySafe:
			if marginBoxSize > imcbSize {
				edge = a.startEdge()
			}
			marginBoxStart = alignInLine(imcb, marginBoxSize, edge)
		default:
			if marginBoxSize <= imcbSize || a.alignment == oofAlignmentNormal {
				marginBoxStart = alignInLine(imcb, marginBoxSize, edge)
			} else {
				limit := Line[float32]{Start: f32Min(imcb.Start, 0.0), End: f32Max(imcb.End, a.cbSize)}
				if a.isScrollAxis {
					if a.reversed {
						limit.Start = float32(math.Inf(-1))
					} else {
						limit.End = float32(math.Inf(1))
					}
				}
				if marginBoxSize <= limit.End-limit.Start {
					start := alignInLine(imcb, marginBoxSize, edge)
					marginBoxStart = f32Max(f32Min(start, limit.End-marginBoxSize), limit.Start)
				} else {
					marginBoxStart = alignInLine(limit, marginBoxSize, a.startEdge())
				}
			}
		}
		return marginBoxStart + margin.Start
	}
}

// OofLayoutResult is the result of laying out out-of-flow candidates against an explicit positioning area.
type OofLayoutResult struct {
	Hoisted                []NodeID
	Unclaimed              OofCandidates
	ScrollableOverflowRect Rect[float32]
}

// ComputeOofLayoutForArea lays out out-of-flow candidates against an explicit positioning area. Unlike
// ComputeOofLayout it does not update the geometry owner's hoisted children or merge overflow into a LayoutOutput.
func ComputeOofLayoutForArea[S CheapCloneStr](
	tree LayoutContainingBlock[S],
	geometryOwner NodeID,
	candidates []OofCandidate,
	area OofPositioningArea,
	direction Direction,
	claims ContainingBlockClaims,
) OofLayoutResult {
	overflow := tree.GetCoreContainerStyle(geometryOwner).Overflow()

	var hoisted []NodeID
	unclaimed := NewOofCandidates()
	scrollableOverflowRect := performOofLayout(
		tree,
		geometryOwner,
		candidates,
		area.Size,
		area.Offset,
		direction,
		claims,
		overflow,
		&hoisted,
		&unclaimed,
	)

	return OofLayoutResult{Hoisted: hoisted, Unclaimed: unclaimed, ScrollableOverflowRect: scrollableOverflowRect}
}

type oofBoxLayout struct {
	layout   Layout
	surfaced OofCandidates
	overflow Point[Overflow]
	contain  Contain
}

func layoutOofBox[S CheapCloneStr, Tree LayoutContainingBlock[S]](
	tree Tree,
	candidate OofCandidate,
	areaSize Size[float32],
	areaOffset Point[float32],
	direction Direction,
	containerOverflow Point[Overflow],
	gridOwner Option[NodeID],
	beforeLayout func(tree Tree, inputs *LayoutInput),
) oofBoxLayout {
	calcFn := treeCalc(tree)
	childStyle := tree.GetOofItemStyle(candidate.Node)

	if owner, ok := gridOwner.Get(); ok {
		info := tree.GetDetailedLayoutInfo(owner)
		if gridInfo, ok := info.Grid(); ok {
			areaRect := Rect[float32]{
				Left:   areaOffset.X,
				Right:  areaOffset.X + areaSize.Width,
				Top:    areaOffset.Y,
				Bottom: areaOffset.Y + areaSize.Height,
			}
			gridArea := gridInfo.resolveAbsoluteGridArea(
				childStyle.GridRow(),
				childStyle.GridColumn(),
				direction,
				areaRect,
			)
			areaSize = Size[float32]{Width: gridArea.Right - gridArea.Left, Height: gridArea.Bottom - gridArea.Top}
			areaOffset = Point[float32]{X: gridArea.Left, Y: gridArea.Top}
		}
	}
	areaWidth := areaSize.Width
	areaHeight := areaSize.Height

	aspectRatio := childStyle.AspectRatio()
	margin := childStyle.Margin().Map(func(m LengthPercentageAuto) Option[float32] {
		return m.ResolveToOption(areaWidth, calcFn)
	})
	padding := RectResolveOrZeroOptionF32(childStyle.Padding(), some(areaWidth), calcFn)
	border := RectResolveOrZeroOptionF32(childStyle.Border(), some(areaWidth), calcFn)
	paddingBorderSum := padding.Add(border).SumAxes()
	boxSizingAdjustment := SizeF32Zero
	if childStyle.BoxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSum
	}

	inset := childStyle.Inset()
	left := MaybeResolveF32(inset.Left, areaWidth, calcFn)
	right := MaybeResolveF32(inset.Right, areaWidth, calcFn)
	top := MaybeResolveF32(inset.Top, areaHeight, calcFn)
	bottom := MaybeResolveF32(inset.Bottom, areaHeight, calcFn)

	sizeStyle := childStyle.Size()
	styleSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolveF32(sizeStyle, areaSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	minSize := SizeOptionF32MaybeMaxF32(
		SizeOptionOr(
			SizeOptionF32MaybeAddF32(
				SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolveF32(childStyle.MinSize(), areaSize, calcFn), aspectRatio),
				boxSizingAdjustment,
			),
			paddingBorderSum.Map(someF32),
		),
		paddingBorderSum,
	)
	maxSize := SizeOptionF32MaybeAddF32(
		SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolveF32(childStyle.MaxSize(), areaSize, calcFn), aspectRatio),
		boxSizingAdjustment,
	)
	knownDimensions := SizeOptionF32MaybeClamp(styleSize, minSize, maxSize)

	isReplaced := childStyle.IsReplaced()
	isTable := childStyle.IsTable()
	itemDirection := childStyle.Direction()
	justifySelf := childStyle.JustifySelf().UnwrapOr(AlignItemsNormal).ResolveSelfRelative(itemDirection, direction, true)
	alignSelf := childStyle.AlignSelf().UnwrapOr(AlignItemsNormal).ResolveSelfRelative(itemDirection, direction, false)
	overflow := childStyle.Overflow()
	scrollbarWidth := childStyle.ScrollbarWidth()
	contain := childStyle.Contain()

	staticPosition := Point[AxisStaticPosition]{
		X: AxisStaticPosition{
			Area:  candidate.StaticPosition.X.Area.Map(func(v float32) float32 { return v - areaOffset.X }),
			Align: candidate.StaticPosition.X.Align,
		},
		Y: AxisStaticPosition{
			Area:  candidate.StaticPosition.Y.Area.Map(func(v float32) float32 { return v - areaOffset.Y }),
			Align: candidate.StaticPosition.Y.Align,
		},
	}

	horizontal := oofAxis{
		cbSize:         areaWidth,
		inset:          Line[Option[float32]]{Start: left, End: right},
		margin:         Line[Option[float32]]{Start: margin.Left, End: margin.Right},
		staticPosition: staticPosition.X,
		alignment:      oofAlignmentFromResolved(justifySelf),
		safety:         justifySelf.safety,
		isInline:       true,
		reversed:       direction.IsRtl(),
		isScrollAxis:   containerOverflow.X.IsScrollContainer(),
	}
	vertical := oofAxis{
		cbSize:         areaHeight,
		inset:          Line[Option[float32]]{Start: top, End: bottom},
		margin:         Line[Option[float32]]{Start: margin.Top, End: margin.Bottom},
		staticPosition: staticPosition.Y,
		alignment:      oofAlignmentFromResolved(alignSelf),
		safety:         alignSelf.safety,
		isInline:       false,
		reversed:       false,
		isScrollAxis:   containerOverflow.Y.IsScrollContainer(),
	}
	stretchFitSize := Size[float32]{Width: horizontal.stretchFitSize(), Height: vertical.stretchFitSize()}
	availableSpaceF32 := stretchFitSize
	if isTable {
		nonAutoMargin := margin.Map(func(m Option[float32]) float32 { return m.UnwrapOr(0.0) })
		availableSpaceF32 = availableSpaceF32.F32Min(Size[float32]{
			Width:  f32Max(areaWidth-nonAutoMargin.HorizontalAxisSum(), 0.0),
			Height: f32Max(areaHeight-nonAutoMargin.VerticalAxisSum(), 0.0),
		})
	}
	availableSpace := SizeF32MaybeClamp(availableSpaceF32, minSize, maxSize).Map(AvailableSpaceDefinite)

	if sizeStyle.Width.IsSizingKeyword() || sizeStyle.Height.IsSizingKeyword() {
		resolveAbsoluteSizingKeywords(
			tree,
			candidate.Node,
			&knownDimensions,
			sizeStyle,
			areaSize,
			Rect[Option[float32]]{Left: left, Right: right, Top: top, Bottom: bottom},
			margin,
			SizingModeContentSize,
		)
		knownDimensions = SizeOptionF32MaybeClamp(SizeOptionF32MaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
	}

	stretchWidth := knownDimensions.Width.IsNone() && horizontal.isStretchSized(isReplaced, isTable)
	stretchHeight := knownDimensions.Height.IsNone() && vertical.isStretchSized(isReplaced, isTable)
	bothExplicitStretch := horizontal.alignment == oofAlignmentStretch && vertical.alignment == oofAlignmentStretch
	stretchAspectRatio := aspectRatio
	if stretchWidth && stretchHeight && bothExplicitStretch {
		stretchAspectRatio = none[float32]()
	}
	if stretchWidth {
		if isTable {
			minSize.Width = some(f32Max(minSize.Width.UnwrapOr(0.0), stretchFitSize.Width))
		} else {
			knownDimensions.Width = some(stretchFitSize.Width)
			knownDimensions = SizeOptionF32MaybeClamp(SizeOptionF32MaybeApplyAspectRatio(knownDimensions, stretchAspectRatio), minSize, maxSize)
		}
	}
	if knownDimensions.Height.IsNone() && stretchHeight {
		if isTable {
			minSize.Height = some(f32Max(minSize.Height.UnwrapOr(0.0), stretchFitSize.Height))
		} else {
			knownDimensions.Height = some(stretchFitSize.Height)
			knownDimensions = SizeOptionF32MaybeClamp(SizeOptionF32MaybeApplyAspectRatio(knownDimensions, stretchAspectRatio), minSize, maxSize)
		}
	}

	var finalSize Size[float32]
	width, wok := knownDimensions.Width.Get()
	height, hok := knownDimensions.Height.Get()
	if wok && hok {
		finalSize = Size[float32]{Width: width, Height: height}
	} else {
		measuredSize := measureChildSizeBoth(
			tree,
			candidate.Node,
			knownDimensions,
			areaSize.Map(someF32),
			availableSpace,
			SizingModeContentSize,
			LineBoolFalse,
		)
		finalSize = SizeOptionUnwrapOr(knownDimensions, measuredSize)
	}
	finalSize = SizeF32MaybeClamp(finalSize, minSize, maxSize)

	layoutInput := LayoutInput{
		KnownDimensions:               finalSize.Map(someF32),
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    areaSize.Map(someF32),
		AvailableSpace:                availableSpace,
		SizingMode:                    SizingModeContentSize,
		Axis:                          RequestedAxisBoth,
		RunMode:                       RunModePerformLayout,
		VerticalMarginsAreCollapsible: LineBoolFalse,
	}
	beforeLayout(tree, &layoutInput)
	layoutOutput := tree.ComputeChildLayout(candidate.Node, layoutInput)

	horizontalMargin := horizontal.resolveMargins(finalSize.Width)
	verticalMargin := vertical.resolveMargins(finalSize.Height)
	resolvedMargin := Rect[float32]{
		Left:   horizontalMargin.Start,
		Right:  horizontalMargin.End,
		Top:    verticalMargin.Start,
		Bottom: verticalMargin.End,
	}

	location := Point[float32]{
		X: horizontal.position(finalSize.Width, horizontalMargin) + areaOffset.X,
		Y: vertical.position(finalSize.Height, verticalMargin) + areaOffset.Y,
	}

	scrollbarSize := SizeF32Zero
	if overflow.Y == OverflowScroll {
		scrollbarSize.Width = scrollbarWidth
	}
	if overflow.X == OverflowScroll {
		scrollbarSize.Height = scrollbarWidth
	}

	layout := Layout{
		Order:                  candidate.Order,
		Size:                   finalSize,
		ScrollableOverflowRect: layoutOutput.ScrollableOverflowRect,
		ScrollbarSize:          scrollbarSize,
		Location:               location,
		Padding:                padding,
		Border:                 border,
		Margin:                 resolvedMargin,
	}
	tree.SetUnroundedLayout(candidate.Node, &layout)

	return oofBoxLayout{
		layout:   layout,
		surfaced: layoutOutput.OofCandidates.Take(),
		overflow: overflow,
		contain:  contain,
	}
}

func performOofLayout[S CheapCloneStr](
	tree LayoutContainingBlock[S],
	nodeID NodeID,
	candidates []OofCandidate,
	areaSize Size[float32],
	areaOffset Point[float32],
	direction Direction,
	claims ContainingBlockClaims,
	containerOverflow Point[Overflow],
	hoisted *[]NodeID,
	unclaimed *OofCandidates,
) Rect[float32] {
	absoluteOverflowRect := RectF32Zero

	if len(candidates) == 0 {
		return absoluteOverflowRect
	}

	isScrollContainer := containerOverflow.X.IsScrollContainer() || containerOverflow.Y.IsScrollContainer()

	var worklist []OofCandidate
	for _, candidate := range candidates {
		if claims.ForPosition(candidate.Position) {
			worklist = append(worklist, candidate)
		} else {
			unclaimed.Push(candidate)
		}
	}

	for index := 0; index < len(worklist); {
		candidate := worklist[index]
		index++
		*hoisted = append(*hoisted, candidate.Node)

		boxLayout := layoutOofBox[S](
			tree,
			candidate,
			areaSize,
			areaOffset,
			direction,
			containerOverflow,
			some(nodeID),
			func(LayoutContainingBlock[S], *LayoutInput) {},
		)
		location := boxLayout.layout.Location
		finalSize := boxLayout.layout.Size

		surfaced := boxLayout.surfaced
		if !surfaced.IsEmpty() {
			surfaced.Translate(location)
			for surfacedCandidate := range surfaced.Iter() {
				if claims.ForPosition(surfacedCandidate.Position) {
					worklist = append(worklist, surfacedCandidate)
				} else {
					unclaimed.Push(surfacedCandidate)
				}
			}
		}

		relativeLocation := Point[float32]{X: location.X - areaOffset.X, Y: location.Y - areaOffset.Y}
		if direction.IsRtl() {
			relativeLocation = Point[float32]{
				X: areaSize.Width - (location.X - areaOffset.X) - finalSize.Width,
				Y: location.Y - areaOffset.Y,
			}
		}
		absoluteOverflowRect = absoluteOverflowRect.Union(computeScrollableOverflowContribution(
			relativeLocation,
			finalSize,
			boxLayout.layout.ScrollableOverflowRect,
			boxLayout.overflow,
			boxLayout.contain,
			isScrollContainer,
		))
	}

	return absoluteOverflowRect
}
