package taffy

import (
	"math"
	"testing"
)

type placementChild struct {
	index    int
	style    Style[string]
	expected expectedPlacement
}

func stylesSeq(styles []*Style[string]) func(func(*Style[string]) bool) {
	return func(yield func(*Style[string]) bool) {
		for _, s := range styles {
			if !yield(s) {
				return
			}
		}
	}
}

func placementSetup(explicitColCount, explicitRowCount uint16, indexes []int, styles []*Style[string], flow GridAutoFlow) ([]gridItem, cellOccupancyMatrix) {
	def := StyleDefault[string]()
	nameResolver := newNamedLineResolver[string](def.Ref(), 0, 0)
	nameResolver.setExplicitColumnCount(explicitColCount)
	nameResolver.setExplicitRowCount(explicitRowCount)
	items := gridItems(func(yield func(int, *Style[string]) bool) {
		for i, s := range styles {
			if !yield(indexes[i], s) {
				return
			}
		}
	})
	placements := resolveNamedLines(stylesSeq(styles), &nameResolver, explicitColCount, explicitRowCount)
	cols, rows := computeGridSizeEstimate(explicitColCount, explicitRowCount, placements)
	matrix := cellOccupancyMatrixWithTrackCounts(cols, rows)
	placeGridItems(&matrix, items, placements, flow)
	return items, matrix
}

func placementTestRunner(t *testing.T, explicitColCount, explicitRowCount uint16, children []placementChild, expectedColCounts, expectedRowCounts trackCounts, flow GridAutoFlow) {
	t.Helper()
	indexes := make([]int, len(children))
	styles := make([]*Style[string], len(children))
	for i := range children {
		indexes[i] = children[i].index
		styles[i] = &children[i].style
	}
	items, matrix := placementSetup(explicitColCount, explicitRowCount, indexes, styles, flow)

	assertEq(t, len(items), len(children))
	for idx, child := range children {
		item := items[idx]
		assertEq(t, item.node, NodeIDFromUsize(child.index))
		a, b, c, d := child.expected.intoOz()
		got := [4]OriginZeroLine{item.column.Start, item.column.End, item.row.Start, item.row.End}
		if want := [4]OriginZeroLine{a, b, c, d}; got != want {
			t.Fatalf("Item %d (0-indexed): got %v, want %v", idx, got, want)
		}
	}

	if got := *matrix.trackCounts(AbsoluteAxisVertical); got != expectedRowCounts {
		t.Fatalf("row track counts: got %v, want %v", got, expectedRowCounts)
	}
	if got := *matrix.trackCounts(AbsoluteAxisHorizontal); got != expectedColCounts {
		t.Fatalf("column track counts: got %v, want %v", got, expectedColCounts)
	}
}

func pc(index int, a, b, c, d gp, e expectedPlacement) placementChild {
	return placementChild{index: index, style: childTestNode{a, b, c, d}.intoGridChild(), expected: e}
}

func TestComputeGridPlacement_OnlyFixedPlacement(t *testing.T) {
	children := []placementChild{
		pc(1, gLine(1), gAuto(), gLine(1), gAuto(), expectedPlacement{0, 1, 0, 1}),
		pc(2, gLine(-4), gAuto(), gLine(-3), gAuto(), expectedPlacement{-1, 0, 0, 1}),
		pc(3, gLine(-3), gAuto(), gLine(-4), gAuto(), expectedPlacement{0, 1, -1, 0}),
		pc(4, gLine(3), gSpan(2), gLine(5), gAuto(), expectedPlacement{2, 4, 4, 5}),
	}
	placementTestRunner(t, 2, 2, children, trackCounts{1, 2, 2}, trackCounts{1, 2, 3}, GridAutoFlowRow)
}

func TestComputeGridPlacement_PlacementSpanningOrigin(t *testing.T) {
	children := []placementChild{
		pc(1, gLine(-1), gLine(-1), gLine(-1), gLine(-1), expectedPlacement{2, 3, 2, 3}),
		pc(2, gLine(-1), gSpan(2), gLine(-1), gSpan(2), expectedPlacement{2, 4, 2, 4}),
		pc(3, gLine(-4), gLine(-4), gLine(-4), gLine(-4), expectedPlacement{-1, 0, -1, 0}),
		pc(4, gLine(-4), gSpan(2), gLine(-4), gSpan(2), expectedPlacement{-1, 1, -1, 1}),
	}
	placementTestRunner(t, 2, 2, children, trackCounts{1, 2, 2}, trackCounts{1, 2, 2}, GridAutoFlowRow)
}

func autoChildren(expected ...expectedPlacement) []placementChild {
	out := make([]placementChild, len(expected))
	for i, e := range expected {
		out[i] = pc(i+1, gAuto(), gAuto(), gAuto(), gAuto(), e)
	}
	return out
}

func TestComputeGridPlacement_OnlyAutoPlacementRowFlow(t *testing.T) {
	children := autoChildren(
		expectedPlacement{0, 1, 0, 1},
		expectedPlacement{1, 2, 0, 1},
		expectedPlacement{0, 1, 1, 2},
		expectedPlacement{1, 2, 1, 2},
		expectedPlacement{0, 1, 2, 3},
		expectedPlacement{1, 2, 2, 3},
		expectedPlacement{0, 1, 3, 4},
		expectedPlacement{1, 2, 3, 4},
	)
	placementTestRunner(t, 2, 2, children, trackCounts{0, 2, 0}, trackCounts{0, 2, 2}, GridAutoFlowRow)
}

func TestComputeGridPlacement_OnlyAutoPlacementColumnFlow(t *testing.T) {
	children := autoChildren(
		expectedPlacement{0, 1, 0, 1},
		expectedPlacement{0, 1, 1, 2},
		expectedPlacement{1, 2, 0, 1},
		expectedPlacement{1, 2, 1, 2},
		expectedPlacement{2, 3, 0, 1},
		expectedPlacement{2, 3, 1, 2},
		expectedPlacement{3, 4, 0, 1},
		expectedPlacement{3, 4, 1, 2},
	)
	placementTestRunner(t, 2, 2, children, trackCounts{0, 2, 2}, trackCounts{0, 2, 0}, GridAutoFlowColumn)
}

func TestComputeGridPlacement_OversizedItem(t *testing.T) {
	children := []placementChild{
		pc(1, gSpan(5), gAuto(), gAuto(), gAuto(), expectedPlacement{0, 5, 0, 1}),
	}
	placementTestRunner(t, 2, 2, children, trackCounts{0, 2, 3}, trackCounts{0, 2, 0}, GridAutoFlowRow)
}

func TestComputeGridPlacement_FixedInSecondaryAxis(t *testing.T) {
	children := []placementChild{
		pc(1, gSpan(2), gAuto(), gLine(1), gAuto(), expectedPlacement{0, 2, 0, 1}),
		pc(2, gAuto(), gAuto(), gLine(2), gAuto(), expectedPlacement{0, 1, 1, 2}),
		pc(3, gAuto(), gAuto(), gLine(1), gAuto(), expectedPlacement{2, 3, 0, 1}),
		pc(4, gAuto(), gAuto(), gLine(4), gAuto(), expectedPlacement{0, 1, 3, 4}),
	}
	placementTestRunner(t, 2, 2, children, trackCounts{0, 2, 1}, trackCounts{0, 2, 2}, GridAutoFlowRow)
}

func TestComputeGridPlacement_DefiniteInSecondaryAxisWithFullyDefiniteNegative(t *testing.T) {
	children := []placementChild{
		pc(2, gAuto(), gAuto(), gLine(2), gAuto(), expectedPlacement{0, 1, 1, 2}),
		pc(1, gLine(-4), gAuto(), gLine(2), gAuto(), expectedPlacement{-1, 0, 1, 2}),
		pc(3, gAuto(), gAuto(), gLine(1), gAuto(), expectedPlacement{-1, 0, 0, 1}),
	}
	placementTestRunner(t, 2, 2, children, trackCounts{1, 2, 0}, trackCounts{0, 2, 0}, GridAutoFlowRow)
}

func TestComputeGridPlacement_DensePackingAlgorithm(t *testing.T) {
	children := []placementChild{
		pc(1, gLine(2), gAuto(), gLine(1), gAuto(), expectedPlacement{1, 2, 0, 1}),
		pc(2, gSpan(2), gAuto(), gAuto(), gAuto(), expectedPlacement{2, 4, 0, 1}),
		pc(3, gAuto(), gAuto(), gAuto(), gAuto(), expectedPlacement{0, 1, 0, 1}),
	}
	placementTestRunner(t, 4, 4, children, trackCounts{0, 4, 0}, trackCounts{0, 4, 0}, GridAutoFlowRowDense)
}

func TestComputeGridPlacement_SparsePackingAlgorithm(t *testing.T) {
	children := []placementChild{
		pc(1, gAuto(), gSpan(3), gAuto(), gAuto(), expectedPlacement{0, 3, 0, 1}),
		pc(2, gAuto(), gSpan(3), gAuto(), gAuto(), expectedPlacement{0, 3, 1, 2}),
		pc(3, gAuto(), gSpan(1), gAuto(), gAuto(), expectedPlacement{3, 4, 1, 2}),
	}
	placementTestRunner(t, 4, 4, children, trackCounts{0, 4, 0}, trackCounts{0, 4, 0}, GridAutoFlowRow)
}

func TestComputeGridPlacement_AutoPlacementInNegativeTracks(t *testing.T) {
	children := []placementChild{
		pc(1, gLine(-5), gAuto(), gLine(1), gAuto(), expectedPlacement{-2, -1, 0, 1}),
		pc(2, gAuto(), gAuto(), gLine(2), gAuto(), expectedPlacement{-2, -1, 1, 2}),
		pc(3, gAuto(), gAuto(), gAuto(), gAuto(), expectedPlacement{-1, 0, 0, 1}),
	}
	placementTestRunner(t, 2, 2, children, trackCounts{2, 2, 0}, trackCounts{0, 2, 0}, GridAutoFlowRowDense)
}

func TestComputeGridPlacement_OverlargePlacementIsClamped(t *testing.T) {
	style := childTestNode{gLine(-19_005), gAuto(), gAuto(), gAuto()}.intoGridChild()
	items, _ := placementSetup(9_000, 0, []int{0}, []*Style[string]{&style}, GridAutoFlowRow)
	assertEq(t, items[0].column, Line[OriginZeroLine]{Start: -10_000, End: -9_999})
}

func TestComputeGridPlacement_AutoPlacementCursorSaturatesAtIntegerBounds(t *testing.T) {
	assertEq(t, advancePosition(OriginZeroLine(math.MaxInt16)), OriginZeroLine(math.MaxInt16))
}

func TestComputeGridPlacement_IndefiniteSpansSaturateAtIntegerBounds(t *testing.T) {
	assertEq(t, resolveIndefiniteGridSpan(OriginZeroLine(math.MaxInt16), 1), Line[OriginZeroLine]{Start: math.MaxInt16, End: math.MaxInt16})
}
