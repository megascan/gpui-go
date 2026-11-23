package taffy

const maxGridTracks uint16 = 10_000

// GridLine is a grid line in CSS grid line coordinates: 1-based, negative indices count from the end (owner: M0-taffy-grid-types).
type GridLine struct {
	inner int16
}

// GridLineFromI16 converts an i16 line index.
func GridLineFromI16(value int16) GridLine {
	return GridLine{inner: value}
}

// AsI16 returns the line index.
func (l GridLine) AsI16() int16 {
	return l.inner
}

func (l GridLine) intoOriginZeroLine(explicitTrackCount uint16) OriginZeroLine {
	panic("taffy: unimplemented GridLine.intoOriginZeroLine")
}

// OriginZeroLine is a grid line in origin-zero coordinates (owner: M0-taffy-grid-types).
type OriginZeroLine int16

// GridCoordinate is the set of grid line coordinate types (owner: M0-taffy-grid-types).
type GridCoordinate interface {
	GridLine | OriginZeroLine
}

// DetailedGridInfo is the detailed information of a grid container's layout (owner: M0-taffy-grid-sizing).
type DetailedGridInfo[S CheapCloneStr] struct{}

// BlockContext is the per-container state of a block formatting context (owner: M0-taffy-block).
type BlockContext struct{}

// ComputeFlexboxLayout computes the layout of a flexbox container and its direct children (owner: M0-taffy-flexbox).
func ComputeFlexboxLayout(tree LayoutFlexboxContainer, node NodeID, inputs LayoutInput) LayoutOutput {
	panic("taffy: unimplemented ComputeFlexboxLayout")
}

// ComputeBlockLayout computes the layout of a block container and its direct children (owner: M0-taffy-block).
func ComputeBlockLayout(tree LayoutBlockContainer, nodeID NodeID, inputs LayoutInput, blockCtx *BlockContext) LayoutOutput {
	panic("taffy: unimplemented ComputeBlockLayout")
}

// ComputeGridLayout computes the layout of a CSS grid container and its direct children (owner: M0-taffy-grid-sizing).
func ComputeGridLayout[S CheapCloneStr](tree LayoutGridContainer[S], node NodeID, inputs LayoutInput) LayoutOutput {
	panic("taffy: unimplemented ComputeGridLayout")
}

// OofLayoutResult is the result of laying out out-of-flow candidates against an explicit positioning area (owner: M0-taffy-block).
type OofLayoutResult struct {
	Hoisted                []NodeID
	Unclaimed              OofCandidates
	ScrollableOverflowRect Rect[float32]
}

// ComputeOofLayoutForArea lays out out-of-flow candidates against an explicit positioning area (owner: M0-taffy-block).
func ComputeOofLayoutForArea[S CheapCloneStr](
	tree LayoutContainingBlock[S],
	geometryOwner NodeID,
	candidates []OofCandidate,
	area OofPositioningArea,
	direction Direction,
	claims ContainingBlockClaims,
) OofLayoutResult {
	panic("taffy: unimplemented ComputeOofLayoutForArea")
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
	panic("taffy: unimplemented layoutOofBox")
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
	panic("taffy: unimplemented performOofLayout")
}
