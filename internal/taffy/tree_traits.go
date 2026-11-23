package taffy

import (
	"iter"
	"unsafe"
)

// TraversePartialTree is taffy's abstraction for downward traversal of a container node and its immediate children.
type TraversePartialTree interface {
	ChildIds(parentNodeID NodeID) iter.Seq[NodeID]
	ChildCount(parentNodeID NodeID) int
	GetChildID(parentNodeID NodeID, childIndex int) NodeID
}

// TraverseTree extends TraversePartialTree with the guarantee that children can be recursed into indefinitely.
// The TraverseTree method is the marker that implementors declare.
type TraverseTree interface {
	TraversePartialTree
	TraverseTree()
}

// LayoutPartialTree is implemented by trees that can be laid out with taffy's algorithms.
type LayoutPartialTree interface {
	TraversePartialTree
	GetCoreContainerStyle(nodeID NodeID) CoreStyle
	ResolveCalcValue(val unsafe.Pointer, basis float32) float32
	SetUnroundedLayout(nodeID NodeID, layout *Layout)
	ComputeChildLayout(nodeID NodeID, inputs LayoutInput) LayoutOutput
}

// LayoutPartialTreeDefaults holds the default method bodies of LayoutPartialTree; embed it in implementations.
type LayoutPartialTreeDefaults struct{}

// ResolveCalcValue is the default calc resolver: it returns 0.
func (LayoutPartialTreeDefaults) ResolveCalcValue(val unsafe.Pointer, basis float32) float32 {
	_ = val
	_ = basis
	return 0.0
}

// LayoutContainingBlock extends LayoutPartialTree with the operations of the out-of-flow positioning pass.
type LayoutContainingBlock[S CheapCloneStr] interface {
	LayoutPartialTree
	GetOofItemStyle(nodeID NodeID) OofItemStyle[S]
	ClearHoistedChildren(nodeID NodeID)
	AddHoistedChildren(nodeID NodeID, hoisted []NodeID)
	GetDetailedLayoutInfo(nodeID NodeID) DetailedLayoutInfo[S]
}

// LayoutContainingBlockDefaults holds the default method bodies of LayoutContainingBlock; embed it in implementations.
type LayoutContainingBlockDefaults[S CheapCloneStr] struct{}

// GetDetailedLayoutInfo is the default: DetailedLayoutInfo None.
func (LayoutContainingBlockDefaults[S]) GetDetailedLayoutInfo(nodeID NodeID) DetailedLayoutInfo[S] {
	_ = nodeID
	return DetailedLayoutInfoNone[S]()
}

// CacheTree lets compute_cached_layout store and retrieve cached layout results.
type CacheTree interface {
	CacheGet(nodeID NodeID, input *LayoutInput) Option[LayoutOutput]
	CacheStore(nodeID NodeID, input *LayoutInput, layoutOutput LayoutOutput)
	CacheClear(nodeID NodeID)
}

// RoundTree is used by round_layout to snap a tree of unrounded layouts to the pixel grid.
type RoundTree interface {
	TraverseTree
	GetUnroundedLayout(nodeID NodeID) Layout
	SetFinalLayout(nodeID NodeID, layout *Layout)
	IsOutOfFlow(nodeID NodeID) bool
	HoistedChildCount(nodeID NodeID) int
	GetHoistedChildID(nodeID NodeID, index int) NodeID
}

// PrintTree is used by WriteTree and PrintTreeStdout to print a debug representation of a tree.
type PrintTree interface {
	TraverseTree
	GetDebugLabel(nodeID NodeID) string
	GetFinalLayout(nodeID NodeID) Layout
}

// LayoutFlexboxContainer extends LayoutPartialTree with the style getters of flexbox layout.
type LayoutFlexboxContainer interface {
	LayoutPartialTree
	GetFlexboxContainerStyle(nodeID NodeID) FlexboxContainerStyle
	GetFlexboxChildStyle(childNodeID NodeID) FlexboxItemStyle
}

// LayoutGridContainer extends LayoutPartialTree with the style getters of CSS grid layout.
type LayoutGridContainer[S CheapCloneStr] interface {
	LayoutPartialTree
	GetGridContainerStyle(nodeID NodeID) GridContainerStyle[S]
	GetGridChildStyle(childNodeID NodeID) GridItemStyle[S]
	SetDetailedGridInfo(nodeID NodeID, detailedGridInfo DetailedGridInfo[S])
}

// LayoutGridContainerDefaults holds the default method bodies of LayoutGridContainer; embed it in implementations.
type LayoutGridContainerDefaults[S CheapCloneStr] struct{}

// SetDetailedGridInfo is the default: it discards the info.
func (LayoutGridContainerDefaults[S]) SetDetailedGridInfo(nodeID NodeID, detailedGridInfo DetailedGridInfo[S]) {
	_ = nodeID
	_ = detailedGridInfo
}

// LayoutBlockContainer extends LayoutPartialTree with the style getters of CSS block layout.
type LayoutBlockContainer interface {
	LayoutPartialTree
	GetBlockContainerStyle(nodeID NodeID) BlockContainerStyle
	GetBlockChildStyle(childNodeID NodeID) BlockItemStyle
	ComputeBlockChildLayout(nodeID NodeID, inputs LayoutInput, blockCtx *BlockContext) LayoutOutput
}

// LayoutBlockContainerDefaultComputeBlockChildLayout is the default body of
// LayoutBlockContainer.ComputeBlockChildLayout, which implementors call: it ignores blockCtx.
func LayoutBlockContainerDefaultComputeBlockChildLayout(tree LayoutPartialTree, nodeID NodeID, inputs LayoutInput, blockCtx *BlockContext) LayoutOutput {
	_ = blockCtx
	return tree.ComputeChildLayout(nodeID, inputs)
}

func measureChildSize(
	tree LayoutPartialTree,
	nodeID NodeID,
	knownDimensions Size[Option[float32]],
	parentSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	sizingMode SizingMode,
	axis AbsoluteAxis,
	verticalMarginsAreCollapsible Line[bool],
) float32 {
	output := tree.ComputeChildLayout(nodeID, LayoutInput{
		KnownDimensions:               knownDimensions,
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          RequestedAxisFromAbsoluteAxis(axis),
		RunMode:                       RunModeComputeSize,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	})
	return output.Size.GetAbs(axis)
}

func measureChildSizeBoth(
	tree LayoutPartialTree,
	nodeID NodeID,
	knownDimensions Size[Option[float32]],
	parentSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	sizingMode SizingMode,
	verticalMarginsAreCollapsible Line[bool],
) Size[float32] {
	return tree.ComputeChildLayout(nodeID, LayoutInput{
		KnownDimensions:               knownDimensions,
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          RequestedAxisBoth,
		RunMode:                       RunModeComputeSize,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	}).Size
}

func performChildLayout(
	tree LayoutPartialTree,
	nodeID NodeID,
	knownDimensions Size[Option[float32]],
	parentSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	sizingMode SizingMode,
	verticalMarginsAreCollapsible Line[bool],
) LayoutOutput {
	return tree.ComputeChildLayout(nodeID, LayoutInput{
		KnownDimensions:               knownDimensions,
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          RequestedAxisBoth,
		RunMode:                       RunModePerformLayout,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	})
}

func calc(tree LayoutPartialTree, val unsafe.Pointer, basis float32) float32 {
	_ = tree
	_ = val
	_ = basis
	return 0.0
}
