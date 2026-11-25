package taffy

import (
	"unsafe"

	"github.com/megascan/gpui-go/gpuiutil"
)

// RootLayoutTree is the tree bound of ComputeRootLayout (upstream `impl LayoutContainingBlock + CacheTree`).
type RootLayoutTree[S CheapCloneStr] interface {
	LayoutContainingBlock[S]
	CacheTree
}

// HiddenLayoutTree is the tree bound of ComputeHiddenLayout (upstream `impl LayoutPartialTree + CacheTree`).
type HiddenLayoutTree interface {
	LayoutPartialTree
	CacheTree
}

// ComputeRootLayout computes layout for the root node in the tree against the initial containing block of size availableSpace.
func ComputeRootLayout[S CheapCloneStr](tree RootLayoutTree[S], root NodeID, availableSpace Size[AvailableSpace]) {
	style := tree.GetCoreContainerStyle(root)
	position := style.Position()
	direction := style.Direction()

	icbSize := SizeAvailableSpaceIntoOptions(availableSpace)
	oofRootArea := none[Size[float32]]()
	if position.IsOutOfFlow() {
		width, wok := icbSize.Width.Get()
		height, hok := icbSize.Height.Get()
		if wok && hok {
			oofRootArea = some(Size[float32]{Width: width, Height: height})
		}
	}

	rootIsCached := false

	var layout Layout
	var candidates OofCandidates
	if areaSize, ok := oofRootArea.Get(); ok {
		start, end := float32(0.0), float32(0.0)
		if direction.IsRtl() {
			start, end = areaSize.Width, areaSize.Width
		}
		edge := AxisStaticEdgeStart
		if direction.IsRtl() {
			edge = AxisStaticEdgeEnd
		}
		candidate := OofCandidate{
			Node:     root,
			Order:    0,
			Position: position,
			StaticPosition: Point[AxisStaticPosition]{
				X: AxisStaticPosition{
					Area:  Line[float32]{Start: start, End: end},
					Align: AxisStaticAlignFromKeyword(edge),
				},
				Y: AxisStaticPositionFromEdge(0.0, AxisStaticEdgeStart),
			},
		}
		boxLayout := layoutOofBox[S](
			tree,
			candidate,
			areaSize,
			PointF32Zero,
			direction,
			Point[Overflow]{X: OverflowVisible, Y: OverflowVisible},
			none[NodeID](),
			func(tree RootLayoutTree[S], inputs *LayoutInput) {
				rootIsCached = tree.CacheGet(root, inputs).IsSome()
			},
		)
		layout, candidates = boxLayout.layout, boxLayout.surfaced
	} else {
		layout, candidates = computeInFlowRootLayout[S](tree, root, availableSpace, &rootIsCached)
	}

	if !candidates.IsEmpty() {
		overflow := tree.GetCoreContainerStyle(root).Overflow()

		areaInset := layout.Border.Add(Rect[float32]{
			Left:   0.0,
			Right:  layout.ScrollbarSize.Width,
			Top:    0.0,
			Bottom: layout.ScrollbarSize.Height,
		})
		rootPaddingBoxSize := layout.Size.Sub(Size[float32]{Width: areaInset.HorizontalAxisSum(), Height: areaInset.VerticalAxisSum()})
		var areaWidth, areaX float32
		if width, ok := icbSize.Width.Get(); ok {
			areaWidth, areaX = f32Max(width-layout.ScrollbarSize.Width, 0.0), -layout.Location.X
		} else {
			areaWidth, areaX = rootPaddingBoxSize.Width, areaInset.Left
		}
		var areaHeight, areaY float32
		if height, ok := icbSize.Height.Get(); ok {
			areaHeight, areaY = f32Max(height-layout.ScrollbarSize.Height, 0.0), -layout.Location.Y
		} else {
			areaHeight, areaY = rootPaddingBoxSize.Height, areaInset.Top
		}
		areaSize := Size[float32]{Width: areaWidth, Height: areaHeight}
		areaOffset := Point[float32]{X: areaX, Y: areaY}

		var hoisted []NodeID
		unclaimed := NewOofCandidates()
		performOofLayout[S](
			tree,
			root,
			candidates.AsSlice(),
			areaSize,
			areaOffset,
			direction,
			ContainingBlockClaimsAll,
			overflow,
			&hoisted,
			&unclaimed,
		)
		if gpuiutil.DebugAssertions && !unclaimed.IsEmpty() {
			panic("the root positioning pass must claim all remaining candidates")
		}
		if !rootIsCached {
			tree.AddHoistedChildren(root, hoisted)
		}
	}
}

func computeInFlowRootLayout[S CheapCloneStr](
	tree RootLayoutTree[S],
	root NodeID,
	availableSpace Size[AvailableSpace],
	rootIsCached *bool,
) (Layout, OofCandidates) {
	knownDimensions := SizeOptionF32None
	calcFn := func(val unsafe.Pointer, basis float32) float32 { return calc(tree, val, basis) }

	var rootAvailableSpace Size[AvailableSpace]
	{
		style := tree.GetCoreContainerStyle(root)
		margin := RectResolveOrZeroOptionF32(style.Margin(), availableSpace.Width.IntoOption(), calcFn)
		rootAvailableSpace = SizeAvailableSpaceMaybeMaxF32(SizeAvailableSpaceMaybeSubF32(availableSpace, margin.SumAxes()), SizeF32Zero)
	}

	{
		parentSize := SizeAvailableSpaceIntoOptions(availableSpace)
		style := tree.GetCoreContainerStyle(root)

		if style.IsBlock() {
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
			clampedStyleSize := SizeOptionF32MaybeClamp(
				SizeOptionF32MaybeAddF32(
					SizeOptionF32MaybeApplyAspectRatio(SizeMaybeResolve(style.Size(), parentSize, calcFn), aspectRatio),
					boxSizingAdjustment,
				),
				minSize,
				maxSize,
			)

			minMaxDefiniteSize := minSize.ZipMap(maxSize, func(minV, maxV Option[float32]) Option[float32] {
				mn, mnok := minV.Get()
				mx, mxok := maxV.Get()
				if mnok && mxok && mx <= mn {
					return some(mn)
				}
				return none[float32]()
			})

			availableSpaceBasedSize := Size[Option[float32]]{Width: rootAvailableSpace.Width.IntoOption(), Height: none[float32]()}

			styledBasedKnownDimensions := SizeOptionF32MaybeMaxF32(
				SizeOptionOr(SizeOptionOr(SizeOptionOr(knownDimensions, minMaxDefiniteSize), clampedStyleSize), availableSpaceBasedSize),
				paddingBorderSize,
			)

			knownDimensions = styledBasedKnownDimensions
		}
	}

	inputs := LayoutInput{
		KnownDimensions:               knownDimensions,
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    SizeAvailableSpaceIntoOptions(availableSpace),
		AvailableSpace:                rootAvailableSpace,
		SizingMode:                    SizingModeInherentSize,
		Axis:                          RequestedAxisBoth,
		RunMode:                       RunModePerformLayout,
		VerticalMarginsAreCollapsible: LineBoolFalse,
	}
	*rootIsCached = tree.CacheGet(root, &inputs).IsSome()

	output := tree.ComputeChildLayout(root, inputs)
	style := tree.GetCoreContainerStyle(root)
	padding := RectResolveOrZeroOptionF32(style.Padding(), availableSpace.Width.IntoOption(), calcFn)
	border := RectResolveOrZeroOptionF32(style.Border(), availableSpace.Width.IntoOption(), calcFn)
	margin := RectResolveOrZeroOptionF32(style.Margin(), availableSpace.Width.IntoOption(), calcFn)
	var scrollbarSize Size[float32]
	if style.Overflow().Y == OverflowScroll {
		scrollbarSize.Width = style.ScrollbarWidth()
	}
	if style.Overflow().X == OverflowScroll {
		scrollbarSize.Height = style.ScrollbarWidth()
	}
	isRtl := style.Direction().IsRtl()

	location := Point[float32]{X: margin.Left, Y: margin.Top}
	if availableWidth, ok := availableSpace.Width.IntoOption().Get(); isRtl && ok {
		location.X = availableWidth - output.Size.Width - margin.Right
	}

	if style.Position() == PositionRelative {
		icbSize := SizeAvailableSpaceIntoOptions(availableSpace)
		inset := style.Inset()
		left := inset.Left.MaybeResolve(icbSize.Width, calcFn)
		right := inset.Right.MaybeResolve(icbSize.Width, calcFn)
		top := inset.Top.MaybeResolve(icbSize.Height, calcFn)
		bottom := inset.Bottom.MaybeResolve(icbSize.Height, calcFn)
		negate := func(o Option[float32]) Option[float32] {
			if v, ok := o.Get(); ok {
				return some(-v)
			}
			return none[float32]()
		}
		if isRtl {
			location.X += negate(right).Or(left).UnwrapOr(0.0)
		} else {
			location.X += left.Or(negate(right)).UnwrapOr(0.0)
		}
		location.Y += top.Or(negate(bottom)).UnwrapOr(0.0)
	}

	layout := Layout{
		Order:                  0,
		Location:               location,
		Size:                   output.Size,
		ScrollableOverflowRect: output.ScrollableOverflowRect,
		ScrollbarSize:          scrollbarSize,
		Padding:                padding,
		Border:                 border,
		Margin:                 margin,
	}
	tree.SetUnroundedLayout(root, &layout)

	return layout, output.OofCandidates.Take()
}

// ComputeCachedLayout returns the cached layout of node for inputs, or computes it with computeUncached and stores it in the cache.
func ComputeCachedLayout[Tree CacheTree](
	tree Tree,
	node NodeID,
	inputs LayoutInput,
	computeUncached func(tree Tree, node NodeID, inputs LayoutInput) LayoutOutput,
) LayoutOutput {
	cacheEntry := tree.CacheGet(node, &inputs)
	if cachedSizeAndBaselines, ok := cacheEntry.Get(); ok {
		return cachedSizeAndBaselines
	}

	computedSizeAndBaselines := computeUncached(tree, node, inputs)

	tree.CacheStore(node, &inputs, computedSizeAndBaselines.Clone())

	return computedSizeAndBaselines
}

// RoundLayout rounds the calculated layout of nodeID and its descendants to exact pixel values, reading the unrounded layout and writing the final layout.
func RoundLayout(tree RoundTree, nodeID NodeID) {
	roundLayoutInner(tree, nodeID, 0.0, 0.0)
}

func roundLayoutInner(tree RoundTree, nodeID NodeID, cumulativeX, cumulativeY float32) {
	unroundedLayout := tree.GetUnroundedLayout(nodeID)
	layout := unroundedLayout

	parentX := cumulativeX
	parentY := cumulativeY
	cumulativeX = cumulativeX + unroundedLayout.Location.X
	cumulativeY = cumulativeY + unroundedLayout.Location.Y

	layout.Location.X = round(cumulativeX) - round(parentX)
	layout.Location.Y = round(cumulativeY) - round(parentY)
	layout.Size.Width = round(cumulativeX+unroundedLayout.Size.Width) - round(cumulativeX)
	layout.Size.Height = round(cumulativeY+unroundedLayout.Size.Height) - round(cumulativeY)
	layout.ScrollbarSize.Width = round(unroundedLayout.ScrollbarSize.Width)
	layout.ScrollbarSize.Height = round(unroundedLayout.ScrollbarSize.Height)
	layout.Border.Left = round(cumulativeX+unroundedLayout.Border.Left) - round(cumulativeX)
	layout.Border.Right = round(cumulativeX+unroundedLayout.Size.Width) -
		round(cumulativeX+unroundedLayout.Size.Width-unroundedLayout.Border.Right)
	layout.Border.Top = round(cumulativeY+unroundedLayout.Border.Top) - round(cumulativeY)
	layout.Border.Bottom = round(cumulativeY+unroundedLayout.Size.Height) -
		round(cumulativeY+unroundedLayout.Size.Height-unroundedLayout.Border.Bottom)
	layout.Padding.Left = round(cumulativeX+unroundedLayout.Padding.Left) - round(cumulativeX)
	layout.Padding.Right = round(cumulativeX+unroundedLayout.Size.Width) -
		round(cumulativeX+unroundedLayout.Size.Width-unroundedLayout.Padding.Right)
	layout.Padding.Top = round(cumulativeY+unroundedLayout.Padding.Top) - round(cumulativeY)
	layout.Padding.Bottom = round(cumulativeY+unroundedLayout.Size.Height) -
		round(cumulativeY+unroundedLayout.Size.Height-unroundedLayout.Padding.Bottom)

	roundScrollableOverflowRect(&layout, unroundedLayout.ScrollableOverflowRect, cumulativeX, cumulativeY)

	tree.SetFinalLayout(nodeID, &layout)

	childCount := tree.ChildCount(nodeID)
	for index := range childCount {
		child := tree.GetChildID(nodeID, index)
		if !tree.IsOutOfFlow(child) {
			roundLayoutInner(tree, child, cumulativeX, cumulativeY)
		}
	}

	hoistedCount := tree.HoistedChildCount(nodeID)
	for index := range hoistedCount {
		child := tree.GetHoistedChildID(nodeID, index)
		roundLayoutInner(tree, child, cumulativeX, cumulativeY)
	}
}

func roundScrollableOverflowRect(layout *Layout, unroundedRect Rect[float32], cumulativeX, cumulativeY float32) {
	layout.ScrollableOverflowRect.Left = round(cumulativeX+unroundedRect.Left) - round(cumulativeX)
	layout.ScrollableOverflowRect.Right = round(cumulativeX+unroundedRect.Right) - round(cumulativeX)
	layout.ScrollableOverflowRect.Top = round(cumulativeY+unroundedRect.Top) - round(cumulativeY)
	layout.ScrollableOverflowRect.Bottom = round(cumulativeY+unroundedRect.Bottom) - round(cumulativeY)
}

// ComputeHiddenLayout lays out node and its children recursively as hidden: each hidden node has zero size and is placed at the origin.
func ComputeHiddenLayout(tree HiddenLayoutTree, node NodeID) LayoutOutput {
	tree.CacheClear(node)
	hidden := LayoutWithOrder(0)
	tree.SetUnroundedLayout(node, &hidden)

	for index := range tree.ChildCount(node) {
		childID := tree.GetChildID(node, index)
		tree.ComputeChildLayout(childID, LayoutInputHidden)
	}

	return LayoutOutputHidden
}
