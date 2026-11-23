package taffy

import (
	"iter"
	"slices"

	"github.com/megascan/gpui-go/gpuiutil"
)

// RunMode is whether a full layout is performed or the node is merely sized.
type RunMode uint8

const (
	RunModePerformLayout RunMode = iota
	RunModeComputeSize
	RunModePerformHiddenLayout
)

// SizingMode is whether styles are taken into account when computing size.
type SizingMode uint8

const (
	SizingModeContentSize SizingMode = iota
	SizingModeInherentSize
)

// CollapsibleMarginSet is a set of margins available for collapsing in block layout.
type CollapsibleMarginSet struct {
	positive float32
	negative float32
}

// CollapsibleMarginSetZero is a margin set with no collapsible margins.
var CollapsibleMarginSetZero = CollapsibleMarginSet{positive: 0.0, negative: 0.0}

// CollapsibleMarginSetFromMargin creates a set from a single margin.
func CollapsibleMarginSetFromMargin(margin float32) CollapsibleMarginSet {
	if margin >= 0.0 {
		return CollapsibleMarginSet{positive: margin, negative: 0.0}
	}
	return CollapsibleMarginSet{positive: 0.0, negative: margin}
}

// CollapseWithMargin collapses a single margin with this set.
func (s CollapsibleMarginSet) CollapseWithMargin(margin float32) CollapsibleMarginSet {
	if margin >= 0.0 {
		s.positive = f32Max(s.positive, margin)
	} else {
		s.negative = f32Min(s.negative, margin)
	}
	return s
}

// CollapseWithSet collapses another margin set with this set.
func (s CollapsibleMarginSet) CollapseWithSet(other CollapsibleMarginSet) CollapsibleMarginSet {
	s.positive = f32Max(s.positive, other.positive)
	s.negative = f32Min(s.negative, other.negative)
	return s
}

// Resolve returns the resultant margin once all collapsible margins have been collapsed into the set.
func (s CollapsibleMarginSet) Resolve() float32 {
	return s.positive + s.negative
}

// RequestedAxis is an axis that layout algorithms can be requested to compute a size for.
type RequestedAxis uint8

const (
	RequestedAxisHorizontal RequestedAxis = iota
	RequestedAxisVertical
	RequestedAxisBoth
)

// Transpose swaps the horizontal and vertical axes (Both is unchanged).
func (a RequestedAxis) Transpose() RequestedAxis {
	switch a {
	case RequestedAxisHorizontal:
		return RequestedAxisVertical
	case RequestedAxisVertical:
		return RequestedAxisHorizontal
	default:
		return RequestedAxisBoth
	}
}

// RequestedAxisFromAbsoluteAxis converts an AbsoluteAxis into a RequestedAxis.
func RequestedAxisFromAbsoluteAxis(value AbsoluteAxis) RequestedAxis {
	switch value {
	case AbsoluteAxisHorizontal:
		return RequestedAxisHorizontal
	default:
		return RequestedAxisVertical
	}
}

// AbsoluteAxisTryFromRequestedAxis converts a RequestedAxis into an AbsoluteAxis; ok is false for Both.
func AbsoluteAxisTryFromRequestedAxis(value RequestedAxis) (AbsoluteAxis, bool) {
	switch value {
	case RequestedAxisHorizontal:
		return AbsoluteAxisHorizontal, true
	case RequestedAxisVertical:
		return AbsoluteAxisVertical, true
	default:
		return 0, false
	}
}

// LayoutInput holds the input constraints and hints for laying out a node, passed in by the parent.
type LayoutInput struct {
	RunMode                       RunMode
	SizingMode                    SizingMode
	Axis                          RequestedAxis
	KnownDimensions               Size[Option[float32]]
	KnownDimensionsAreDefinite    Size[bool]
	ParentSize                    Size[Option[float32]]
	AvailableSpace                Size[AvailableSpace]
	VerticalMarginsAreCollapsible Line[bool]
}

// LayoutInputHidden is a LayoutInput that requests hidden layout.
var LayoutInputHidden = LayoutInput{
	RunMode:                       RunModePerformHiddenLayout,
	KnownDimensions:               SizeOptionF32None,
	KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
	ParentSize:                    SizeOptionF32None,
	AvailableSpace:                Size[AvailableSpace]{Width: AvailableSpaceMaxContent, Height: AvailableSpaceMaxContent},
	SizingMode:                    SizingModeInherentSize,
	Axis:                          RequestedAxisBoth,
	VerticalMarginsAreCollapsible: LineBoolFalse,
}

// Transpose swaps the horizontal and vertical axes of every per-axis input.
func (i LayoutInput) Transpose() LayoutInput {
	return LayoutInput{
		RunMode:                       i.RunMode,
		SizingMode:                    i.SizingMode,
		Axis:                          i.Axis.Transpose(),
		KnownDimensions:               i.KnownDimensions.Transpose(),
		KnownDimensionsAreDefinite:    i.KnownDimensionsAreDefinite.Transpose(),
		ParentSize:                    i.ParentSize.Transpose(),
		AvailableSpace:                i.AvailableSpace.Transpose(),
		VerticalMarginsAreCollapsible: LineBoolFalse,
	}
}

// Baselines holds the first and last baselines of a node in the horizontal axis.
type Baselines struct {
	First Option[float32]
	Last  Option[float32]
}

// BaselinesNone is a Baselines with neither a first nor a last baseline.
var BaselinesNone = Baselines{First: none[float32](), Last: none[float32]()}

// BaselinesFromFirst creates a Baselines from just a first baseline.
func BaselinesFromFirst(first Option[float32]) Baselines {
	return Baselines{First: first, Last: none[float32]()}
}

// AxisStaticEdge is the physical alignment of an out-of-flow box's margin box within its static-position area.
type AxisStaticEdge uint8

const (
	AxisStaticEdgeStart AxisStaticEdge = iota
	AxisStaticEdgeCenter
	AxisStaticEdgeEnd
)

// AxisStaticAlign is the alignment applied to an out-of-flow box's margin box within its static-position area in one axis.
type AxisStaticAlign struct {
	Keyword  AxisStaticEdge
	Safety   AlignmentSafety
	Fallback AxisStaticEdge
}

// AxisStaticAlignFromKeyword creates an AxisStaticAlign from a keyword with no safe fallback.
func AxisStaticAlignFromKeyword(keyword AxisStaticEdge) AxisStaticAlign {
	return AxisStaticAlign{Keyword: keyword, Safety: AlignmentSafetyDefault, Fallback: keyword}
}

// AxisStaticPosition is the static position of an out-of-flow box in a single axis.
type AxisStaticPosition struct {
	Area  Line[float32]
	Align AxisStaticAlign
}

// AxisStaticPositionFromEdge creates an AxisStaticPosition with a degenerate area at anchor and no safe fallback.
func AxisStaticPositionFromEdge(anchor float32, edge AxisStaticEdge) AxisStaticPosition {
	return AxisStaticPosition{Area: Line[float32]{Start: anchor, End: anchor}, Align: AxisStaticAlignFromKeyword(edge)}
}

// AxisStaticPositionFromAlignment creates an AxisStaticPosition that aligns the box within area by its resolved self-alignment.
func AxisStaticPositionFromAlignment(alignment AlignSelf, area Line[float32], axisIsRtl bool) AxisStaticPosition {
	if gpuiutil.DebugAssertions && (alignment.keyword == AlignItemsKeywordSelfStart || alignment.keyword == AlignItemsKeywordSelfEnd) {
		panic("self-start/self-end must be resolved before computing the static position")
	}
	edgeFor := func(keyword AlignItemsKeyword, axisIsRtl bool) AxisStaticEdge {
		startPosition := !(keyword == AlignItemsKeywordEnd || keyword == AlignItemsKeywordFlexEnd) != axisIsRtl
		switch {
		case keyword == AlignItemsKeywordCenter:
			return AxisStaticEdgeCenter
		case startPosition:
			return AxisStaticEdgeStart
		default:
			return AxisStaticEdgeEnd
		}
	}
	fallback := alignment.keyword
	if alignment.safety == AlignmentSafetySafe {
		fallback = AlignItemsKeywordStart
	}
	return AxisStaticPosition{
		Area: area,
		Align: AxisStaticAlign{
			Keyword:  edgeFor(alignment.keyword, axisIsRtl),
			Safety:   alignment.safety,
			Fallback: edgeFor(fallback, axisIsRtl),
		},
	}
}

// OofCandidate records an out-of-flow box not yet laid out because its containing block is an ancestor of its parent.
type OofCandidate struct {
	Node           NodeID
	Order          uint32
	Position       Position
	StaticPosition Point[AxisStaticPosition]
}

// OofCandidates is a lazily allocated list of OofCandidate; the zero value is empty. Copies share storage: use Clone.
type OofCandidates struct {
	inner *[]OofCandidate
}

// OofCandidatesNone is an empty list of candidates.
var OofCandidatesNone = OofCandidates{inner: nil}

// NewOofCandidates creates a new empty list of candidates.
func NewOofCandidates() OofCandidates {
	return OofCandidatesNone
}

// IsEmpty reports whether the list contains no candidates.
func (c OofCandidates) IsEmpty() bool {
	return len(c.AsSlice()) == 0
}

// AsSlice returns the candidates as a slice.
func (c OofCandidates) AsSlice() []OofCandidate {
	if c.inner == nil {
		return nil
	}
	return *c.inner
}

// AsMutSlice returns the candidates as a mutable slice.
func (c *OofCandidates) AsMutSlice() []OofCandidate {
	return c.AsSlice()
}

// Push appends a candidate to the list.
func (c *OofCandidates) Push(candidate OofCandidate) {
	if c.inner == nil {
		c.inner = new([]OofCandidate)
	}
	*c.inner = append(*c.inner, candidate)
}

// Append moves all candidates from other into c, leaving other empty.
func (c *OofCandidates) Append(other *OofCandidates) {
	if other.IsEmpty() {
		return
	}
	if c.inner == nil {
		c.inner = other.inner
		other.inner = nil
		return
	}
	*c.inner = append(*c.inner, *other.inner...)
	*other.inner = nil
}

// Take returns the candidates, leaving c empty.
func (c *OofCandidates) Take() OofCandidates {
	taken := *c
	*c = OofCandidates{}
	return taken
}

// Clear removes all candidates from the list.
func (c *OofCandidates) Clear() {
	c.inner = nil
}

// Iter iterates over the candidates.
func (c OofCandidates) Iter() iter.Seq[OofCandidate] {
	return slices.Values(c.AsSlice())
}

// Translate offsets the static-position areas of every candidate by offset.
func (c *OofCandidates) Translate(offset Point[float32]) {
	candidates := c.AsMutSlice()
	for i := range candidates {
		candidate := &candidates[i]
		candidate.StaticPosition.X.Area.Start += offset.X
		candidate.StaticPosition.X.Area.End += offset.X
		candidate.StaticPosition.Y.Area.Start += offset.Y
		candidate.StaticPosition.Y.Area.End += offset.Y
	}
}

// Clone returns a deep copy of the list (upstream's Clone).
func (c OofCandidates) Clone() OofCandidates {
	if c.inner == nil {
		return OofCandidates{}
	}
	cloned := slices.Clone(*c.inner)
	if cloned == nil {
		cloned = []OofCandidate{}
	}
	return OofCandidates{inner: &cloned}
}

// Equal reports whether c and other are equal (upstream's PartialEq).
func (c OofCandidates) Equal(other OofCandidates) bool {
	if (c.inner == nil) != (other.inner == nil) {
		return false
	}
	return c.inner == nil || slices.Equal(*c.inner, *other.inner)
}

// OofPositioningArea is the area of a container against which the insets of its out-of-flow boxes are resolved.
type OofPositioningArea struct {
	Size   Size[float32]
	Offset Point[float32]
}

// LayoutOutput is the result of laying out a single node, returned up to the parent node.
type LayoutOutput struct {
	Size                      Size[float32]
	ScrollableOverflowRect    Rect[float32]
	Baselines                 Baselines
	TopMargin                 CollapsibleMarginSet
	BottomMargin              CollapsibleMarginSet
	MarginsCanCollapseThrough bool
	OofCandidates             OofCandidates
	OofPositioningArea        Option[OofPositioningArea]
}

// LayoutOutputHidden is an all-zero LayoutOutput for hidden nodes.
var LayoutOutputHidden = LayoutOutput{
	Size:                      SizeF32Zero,
	ScrollableOverflowRect:    RectF32Zero,
	Baselines:                 BaselinesNone,
	TopMargin:                 CollapsibleMarginSetZero,
	BottomMargin:              CollapsibleMarginSetZero,
	MarginsCanCollapseThrough: false,
	OofCandidates:             OofCandidatesNone,
	OofPositioningArea:        none[OofPositioningArea](),
}

// LayoutOutputDefault is a blank layout output.
var LayoutOutputDefault = LayoutOutputHidden

// LayoutOutputFromSizesAndBaselines creates a LayoutOutput from the size, scrollable overflow rectangle and baselines.
func LayoutOutputFromSizesAndBaselines(size Size[float32], scrollableOverflowRect Rect[float32], baselines Baselines) LayoutOutput {
	return LayoutOutput{
		Size:                      size,
		ScrollableOverflowRect:    scrollableOverflowRect,
		Baselines:                 baselines,
		TopMargin:                 CollapsibleMarginSetZero,
		BottomMargin:              CollapsibleMarginSetZero,
		MarginsCanCollapseThrough: false,
		OofCandidates:             OofCandidatesNone,
		OofPositioningArea:        none[OofPositioningArea](),
	}
}

// LayoutOutputFromSizes creates a LayoutOutput from the container size and scrollable overflow rectangle.
func LayoutOutputFromSizes(size Size[float32], scrollableOverflowRect Rect[float32]) LayoutOutput {
	return LayoutOutputFromSizesAndBaselines(size, scrollableOverflowRect, BaselinesNone)
}

// Transpose swaps the horizontal and vertical axes of the output (the inverse of LayoutInput.Transpose).
func (o LayoutOutput) Transpose() LayoutOutput {
	area := none[OofPositioningArea]()
	if a, ok := o.OofPositioningArea.Get(); ok {
		area = some(OofPositioningArea{Size: a.Size.Transpose(), Offset: a.Offset.Transpose()})
	}
	return LayoutOutput{
		Size:                      o.Size.Transpose(),
		ScrollableOverflowRect:    o.ScrollableOverflowRect.Transpose(),
		Baselines:                 BaselinesNone,
		TopMargin:                 CollapsibleMarginSetZero,
		BottomMargin:              CollapsibleMarginSetZero,
		MarginsCanCollapseThrough: false,
		OofCandidates:             o.OofCandidates,
		OofPositioningArea:        area,
	}
}

// LayoutOutputFromOuterSize creates a LayoutOutput from just the container's size.
func LayoutOutputFromOuterSize(size Size[float32]) LayoutOutput {
	return LayoutOutputFromSizes(size, RectF32Zero)
}

// Clone returns a deep copy of the output (upstream's Clone).
func (o LayoutOutput) Clone() LayoutOutput {
	o.OofCandidates = o.OofCandidates.Clone()
	return o
}

// Equal reports whether o and other are equal (upstream's PartialEq).
func (o LayoutOutput) Equal(other LayoutOutput) bool {
	return o.Size == other.Size &&
		o.ScrollableOverflowRect == other.ScrollableOverflowRect &&
		o.Baselines == other.Baselines &&
		o.TopMargin == other.TopMargin &&
		o.BottomMargin == other.BottomMargin &&
		o.MarginsCanCollapseThrough == other.MarginsCanCollapseThrough &&
		o.OofCandidates.Equal(other.OofCandidates) &&
		o.OofPositioningArea == other.OofPositioningArea
}

// Layout is the final result of a layout algorithm for a single node.
type Layout struct {
	Order                  uint32
	Location               Point[float32]
	Size                   Size[float32]
	ScrollableOverflowRect Rect[float32]
	ScrollbarSize          Size[float32]
	Border                 Rect[float32]
	Padding                Rect[float32]
	Margin                 Rect[float32]
}

// NewLayout creates a zero Layout with order 0 (also upstream's Default).
func NewLayout() Layout {
	return Layout{
		Order:                  0,
		Location:               PointF32Zero,
		Size:                   SizeF32Zero,
		ScrollableOverflowRect: RectF32Zero,
		ScrollbarSize:          SizeF32Zero,
		Border:                 RectF32Zero,
		Padding:                RectF32Zero,
		Margin:                 RectF32Zero,
	}
}

// LayoutWithOrder creates a zero Layout with the supplied order.
func LayoutWithOrder(order uint32) Layout {
	l := NewLayout()
	l.Order = order
	return l
}

// ContentBoxWidth returns the width of the node's content box.
func (l Layout) ContentBoxWidth() float32 {
	return l.Size.Width - l.Padding.Left - l.Padding.Right - l.Border.Left - l.Border.Right
}

// ContentBoxHeight returns the height of the node's content box.
func (l Layout) ContentBoxHeight() float32 {
	return l.Size.Height - l.Padding.Top - l.Padding.Bottom - l.Border.Top - l.Border.Bottom
}

// ContentBoxSize returns the size of the node's content box.
func (l Layout) ContentBoxSize() Size[float32] {
	return Size[float32]{Width: l.ContentBoxWidth(), Height: l.ContentBoxHeight()}
}

// ContentBoxX returns the x offset of the node's content box relative to its parent's border box.
func (l Layout) ContentBoxX() float32 {
	return l.Location.X + l.Border.Left + l.Padding.Left
}

// ContentBoxY returns the y offset of the node's content box relative to its parent's border box.
func (l Layout) ContentBoxY() float32 {
	return l.Location.Y + l.Border.Top + l.Padding.Top
}

// ScrollWidth returns the maximum horizontal scroll offset of the node.
func (l Layout) ScrollWidth() float32 {
	return f32Max(
		0.0,
		l.ScrollableOverflowRect.Right+f32Min(l.ScrollbarSize.Width, l.Size.Width)-l.Size.Width+l.Border.Left+l.Border.Right,
	)
}

// ScrollHeight returns the maximum vertical scroll offset of the node.
func (l Layout) ScrollHeight() float32 {
	return f32Max(
		0.0,
		l.ScrollableOverflowRect.Bottom+f32Min(l.ScrollbarSize.Height, l.Size.Height)-l.Size.Height+l.Border.Top+l.Border.Bottom,
	)
}

// DetailedLayoutInfoKind is the variant of a DetailedLayoutInfo.
type DetailedLayoutInfoKind uint8

const (
	DetailedLayoutInfoKindGrid DetailedLayoutInfoKind = iota
	DetailedLayoutInfoKindNone
)

// DetailedLayoutInfo is the additional information from a layout algorithm. The zero value is None.
type DetailedLayoutInfo[S CheapCloneStr] struct {
	grid *DetailedGridInfo[S]
}

// DetailedLayoutInfoGrid is the Grid variant.
func DetailedLayoutInfoGrid[S CheapCloneStr](info *DetailedGridInfo[S]) DetailedLayoutInfo[S] {
	return DetailedLayoutInfo[S]{grid: info}
}

// DetailedLayoutInfoNone is the None variant: a node that has no detailed information yet.
func DetailedLayoutInfoNone[S CheapCloneStr]() DetailedLayoutInfo[S] {
	return DetailedLayoutInfo[S]{}
}

// Kind returns the variant.
func (d DetailedLayoutInfo[S]) Kind() DetailedLayoutInfoKind {
	if d.grid != nil {
		return DetailedLayoutInfoKindGrid
	}
	return DetailedLayoutInfoKindNone
}

// Grid returns the grid info of the Grid variant.
func (d DetailedLayoutInfo[S]) Grid() (*DetailedGridInfo[S], bool) {
	return d.grid, d.grid != nil
}
