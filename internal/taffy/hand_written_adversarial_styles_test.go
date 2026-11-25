package taffy

import (
	"math"
	"testing"
)

func adversarialStylesDefinite(size float32) Size[AvailableSpace] {
	return hwDefinite(size, size)
}

func TestHandWrittenAdversarialStyles_NonFiniteFlexFactorTerminates(t *testing.T) {
	for _, flexFactor := range []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
		tree := newTestTree()
		cs := defaultStyle()
		cs.Size = hwSizeDim(Length[Dimension](10.0), Length[Dimension](10.0))
		child := hwUnwrap(tree.NewLeaf(cs))
		rs := defaultStyle()
		rs.Display = DisplayGrid
		rs.GridTemplateColumns = []GridTemplateComponent[defaultCheapStr]{GridTemplateComponentSingle[defaultCheapStr](TrackSizingFunction{
			Min: MinTrackSizingFunctionAuto(),
			Max: MaxTrackSizingFunctionFr(flexFactor),
		})}
		root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{child}))
		hwOK(tree.ComputeLayout(root, adversarialStylesDefinite(100.0)))
	}
}

func TestHandWrittenAdversarialStyles_NonFiniteGridContainerSizeTerminates(t *testing.T) {
	for _, width := range []float32{float32(math.NaN()), float32(math.Inf(1))} {
		tree := newTestTree()
		child := hwUnwrap(tree.NewLeaf(defaultStyle()))
		rs := defaultStyle()
		rs.Display = DisplayGrid
		rs.Size = hwSizeDim(DimensionLength(width), Length[Dimension](100.0))
		rs.GridTemplateColumns = []GridTemplateComponent[defaultCheapStr]{Repeat[defaultCheapStr](uint16(2), []TrackSizingFunction{Fr[TrackSizingFunction](1.0)})}
		root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{child}))
		hwOK(tree.ComputeLayout(root, adversarialStylesDefinite(100.0)))
	}
}

func TestHandWrittenAdversarialStyles_RepetitionWithoutLineNamesResolvesLines(t *testing.T) {
	tree := newTestTree()
	cs := defaultStyle()
	cs.Size = hwSizeDim(Length[Dimension](5.0), Length[Dimension](5.0))
	cs.GridColumn = Line[GridPlacement[defaultCheapStr]]{Start: GridPlacementNamedLine[defaultCheapStr]("c", 1), End: GridPlacementAuto[defaultCheapStr]()}
	child := hwUnwrap(tree.NewLeaf(cs))
	rs := defaultStyle()
	rs.Display = DisplayGrid
	rs.Size = hwSizeDim(Length[Dimension](100.0), Length[Dimension](100.0))
	rs.GridTemplateColumns = []GridTemplateComponent[defaultCheapStr]{
		Repeat[defaultCheapStr](uint16(2), []TrackSizingFunction{Length[TrackSizingFunction](10.0)}),
		Length[GridTemplateComponent[defaultCheapStr]](10.0),
	}
	rs.GridTemplateColumnNames = [][]string{{}, {}, {"c"}}
	root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{child}))

	hwOK(tree.ComputeLayout(root, adversarialStylesDefinite(100.0)))

	hwAssertEq(t, hwLayout(tree, child).Location.X, 30.0)
}

func TestHandWrittenAdversarialStyles_ZeroSpanAndNamedSpanTerminates(t *testing.T) {
	for _, direction := range []Direction{DirectionLtr, DirectionRtl} {
		tree := newTestTree()
		cs := defaultStyle()
		cs.GridColumn = Line[GridPlacement[defaultCheapStr]]{
			Start: GridPlacementNamedSpan[defaultCheapStr]("does-not-exist", 1),
			End:   GridPlacement[defaultCheapStr]{}.FromSpan(0),
		}
		child := hwUnwrap(tree.NewLeaf(cs))
		rs := defaultStyle()
		rs.Display = DisplayGrid
		rs.Direction = direction
		root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{child}))

		hwOK(tree.ComputeLayout(root, adversarialStylesDefinite(100.0)))
	}
}

func TestHandWrittenAdversarialStyles_RepetitionWithWrongLineNameSetCountPanics(t *testing.T) {
	hwShouldPanic(t, "line name sets", func() {
		tree := newTestTree()
		child := hwUnwrap(tree.NewLeaf(defaultStyle()))
		rs := defaultStyle()
		rs.Display = DisplayGrid
		rs.GridTemplateColumns = []GridTemplateComponent[defaultCheapStr]{GridTemplateComponentRepeat(GridTemplateRepetition[defaultCheapStr]{
			Count:     RepetitionCountCount(2),
			Tracks:    []TrackSizingFunction{Length[TrackSizingFunction](10.0)},
			LineNames: [][]string{{"a"}},
		})}
		rs.GridTemplateColumnNames = [][]string{{}, {}}
		root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{child}))

		hwOK(tree.ComputeLayout(root, adversarialStylesDefinite(100.0)))
	})
}

func TestHandWrittenAdversarialStyles_SmallItemInMaxSizeOccupiedGridTerminates(t *testing.T) {
	tree := newTestTree()
	gp := GridPlacement[defaultCheapStr]{}
	bs := defaultStyle()
	bs.GridRow = Line[GridPlacement[defaultCheapStr]]{Start: gp.FromSpan(10000), End: gp.Auto()}
	bs.GridColumn = Line[GridPlacement[defaultCheapStr]]{Start: gp.FromSpan(10000), End: gp.Auto()}
	big := hwUnwrap(tree.NewLeaf(bs))
	ss := defaultStyle()
	ss.GridRow = Line[GridPlacement[defaultCheapStr]]{Start: gp.FromSpan(10000), End: gp.Auto()}
	ss.GridColumn = Line[GridPlacement[defaultCheapStr]]{Start: gp.FromSpan(1), End: gp.Auto()}
	small := hwUnwrap(tree.NewLeaf(ss))
	rs := defaultStyle()
	rs.Display = DisplayGrid
	rs.Size = hwSizeDim(Length[Dimension](100.0), Length[Dimension](100.0))
	root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{big, small}))

	hwOK(tree.ComputeLayout(root, adversarialStylesDefinite(100.0)))
}

func TestHandWrittenAdversarialStyles_ExtremePlacementsAndSpansTerminate(t *testing.T) {
	tree := newTestTree()
	gp := GridPlacement[defaultCheapStr]{}
	placements := []Line[GridPlacement[defaultCheapStr]]{
		{Start: gp.Auto(), End: gp.FromSpan(2000)},
		{Start: gp.FromLineIndex(-2000), End: gp.FromLineIndex(2000)},
	}
	children := make([]NodeID, 0, len(placements))
	for _, placement := range placements {
		s := defaultStyle()
		s.GridRow = placement
		s.GridColumn = placement
		children = append(children, hwUnwrap(tree.NewLeaf(s)))
	}
	rs := defaultStyle()
	rs.Display = DisplayGrid
	rs.Size = hwSizeDim(Length[Dimension](100.0), Length[Dimension](100.0))
	root := hwUnwrap(tree.NewWithChildren(rs, children))

	hwOK(tree.ComputeLayout(root, adversarialStylesDefinite(100.0)))
}
