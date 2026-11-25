package taffy

import (
	"slices"
	"testing"
)

func interval(start, end int16, state cellOccupancyState) occupiedInterval {
	return occupiedInterval{rng: rangeI16{start: start, end: end}, state: state}
}

func r16(start, end int16) rangeI16 {
	return rangeI16{start: start, end: end}
}

func assertIntervals(t *testing.T, got []occupiedInterval, want ...occupiedInterval) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("intervals = %v, want %v", got, want)
	}
}

func assertEq[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

const (
	autoPlaced       = cellOccupancyStateAutoPlaced
	definitelyPlaced = cellOccupancyStateDefinitelyPlaced
)

func TestComputeGridTypesCellOccupancy_TrackIntervals_PaintMergesTouchingSameStateIntervals(t *testing.T) {
	var track trackIntervals
	track.paint(r16(0, 2), autoPlaced)
	track.paint(r16(4, 6), autoPlaced)
	track.paint(r16(2, 4), autoPlaced)
	assertIntervals(t, track.intervals, interval(0, 6, autoPlaced))
}

func TestComputeGridTypesCellOccupancy_TrackIntervals_PaintDoesNotMergeDifferentStates(t *testing.T) {
	var track trackIntervals
	track.paint(r16(0, 2), autoPlaced)
	track.paint(r16(2, 4), definitelyPlaced)
	assertIntervals(t, track.intervals, interval(0, 2, autoPlaced), interval(2, 4, definitelyPlaced))
}

func TestComputeGridTypesCellOccupancy_TrackIntervals_PaintOverwritesOverlappedCells(t *testing.T) {
	var track trackIntervals
	track.paint(r16(0, 6), autoPlaced)
	track.paint(r16(2, 4), definitelyPlaced)
	assertIntervals(t, track.intervals, interval(0, 2, autoPlaced), interval(2, 4, definitelyPlaced), interval(4, 6, autoPlaced))

	track.paint(r16(-1, 7), autoPlaced)
	assertIntervals(t, track.intervals, interval(-1, 7, autoPlaced))
}

func TestComputeGridTypesCellOccupancy_TrackIntervals_PaintOverwritesMultipleIntervals(t *testing.T) {
	var track trackIntervals
	track.paint(r16(0, 2), autoPlaced)
	track.paint(r16(3, 5), definitelyPlaced)
	track.paint(r16(6, 8), autoPlaced)
	track.paint(r16(1, 7), definitelyPlaced)
	assertIntervals(t, track.intervals, interval(0, 1, autoPlaced), interval(1, 7, definitelyPlaced), interval(7, 8, autoPlaced))
}

func TestComputeGridTypesCellOccupancy_TrackIntervals_CollisionExtentFindsExtremalOccupiedCell(t *testing.T) {
	var track trackIntervals
	track.paint(r16(2, 4), autoPlaced)
	track.paint(r16(6, 8), definitelyPlaced)
	assertEq(t, track.collisionExtent(r16(0, 10)), some[int16](7))
	assertEq(t, track.collisionExtent(r16(0, 7)), some[int16](7))
	assertEq(t, track.collisionExtent(r16(0, 6)), some[int16](3))
	assertEq(t, track.collisionExtent(r16(4, 6)), none[int16]())
}

func TestComputeGridTypesCellOccupancy_TrackIntervals_StateAtDistinguishesOccupiedIntervalsAndGaps(t *testing.T) {
	var track trackIntervals
	track.paint(r16(0, 2), definitelyPlaced)
	track.paint(r16(2, 4), autoPlaced)
	track.paint(r16(6, 8), autoPlaced)
	track.paint(r16(8, 9), definitelyPlaced)
	assertEq(t, track.stateAt(0), definitelyPlaced)
	assertEq(t, track.stateAt(3), autoPlaced)
	assertEq(t, track.stateAt(5), cellOccupancyStateUnoccupied)
	assertEq(t, track.stateAt(7), autoPlaced)
	assertEq(t, track.stateAt(8), definitelyPlaced)
}

func TestComputeGridTypesCellOccupancy_TrackIntervals_DefinitelyPlacedOverwriteHidesAutoPlacedCells(t *testing.T) {
	var track trackIntervals
	track.paint(r16(0, 4), cellOccupancyStateAutoPlaced)
	track.paint(r16(2, 4), cellOccupancyStateDefinitelyPlaced)
	assertEq(t, track.stateAt(1), cellOccupancyStateAutoPlaced)
	assertEq(t, track.stateAt(2), cellOccupancyStateDefinitelyPlaced)
	assertEq(t, track.stateAt(3), cellOccupancyStateDefinitelyPlaced)
}

func ozLineSpan(start, end int16) Line[OriginZeroLine] {
	return Line[OriginZeroLine]{Start: OriginZeroLine(start), End: OriginZeroLine(end)}
}

func TestComputeGridTypesCellOccupancy_CellOccupancyMatrix_NegativeExpansionPreservesOccupancy(t *testing.T) {
	h, v := AbsoluteAxisHorizontal, AbsoluteAxisVertical
	matrix := cellOccupancyMatrixWithTrackCounts(trackCountsFromRaw(0, 2, 0), trackCountsFromRaw(0, 2, 0))
	matrix.markAreaAs(h, ozLineSpan(0, 1), ozLineSpan(0, 1), autoPlaced)
	matrix.markAreaAs(h, ozLineSpan(-2, -1), ozLineSpan(-1, 0), autoPlaced)

	assertEq(t, *matrix.trackCounts(h), trackCountsFromRaw(2, 2, 0))
	assertEq(t, *matrix.trackCounts(v), trackCountsFromRaw(1, 2, 0))

	assertEq(t, matrix.lineAreaIsUnoccupied(h, ozLineSpan(0, 1), ozLineSpan(0, 1)), false)
	assertEq(t, matrix.lineAreaIsUnoccupied(h, ozLineSpan(-2, -1), ozLineSpan(-1, 0)), false)
	assertEq(t, matrix.lineAreaIsUnoccupied(h, ozLineSpan(-1, 0), ozLineSpan(0, 1)), true)

	assertEq(t, matrix.columnIsOccupied(0), true)
	assertEq(t, matrix.columnIsOccupied(1), false)
	assertEq(t, matrix.columnIsOccupied(2), true)
	assertEq(t, matrix.rowIsOccupied(0), true)
	assertEq(t, matrix.rowIsOccupied(1), true)
	assertEq(t, matrix.rowIsOccupied(2), false)
}

func TestComputeGridTypesCellOccupancy_CellOccupancyMatrix_AutoPlacedItemOnlyAdvancesCursorInFirstSecondaryTrack(t *testing.T) {
	for _, axis := range []AbsoluteAxis{AbsoluteAxisHorizontal, AbsoluteAxisVertical} {
		matrix := cellOccupancyMatrixWithTrackCounts(trackCountsFromRaw(0, 4, 0), trackCountsFromRaw(0, 4, 0))
		matrix.markAreaAs(axis, ozLineSpan(1, 3), ozLineSpan(1, 4), autoPlaced)

		assertEq(t, matrix.autoPlacementCursor(axis, 1), some(OriginZeroLine(3)))
		assertEq(t, matrix.autoPlacementCursor(axis, 2), none[OriginZeroLine]())
		assertEq(t, matrix.autoPlacementCursor(axis, 3), none[OriginZeroLine]())

		assertEq(t, matrix.lineAreaIsUnoccupied(axis, ozLineSpan(1, 3), ozLineSpan(2, 4)), false)
	}
}

func TestComputeGridTypesCellOccupancy_CellOccupancyMatrix_CursorsSurviveIntervalMergingAndGridExpansion(t *testing.T) {
	for _, axis := range []AbsoluteAxis{AbsoluteAxisHorizontal, AbsoluteAxisVertical} {
		matrix := cellOccupancyMatrixWithTrackCounts(trackCountsFromRaw(0, 4, 0), trackCountsFromRaw(0, 4, 0))
		matrix.markAreaAs(axis, ozLineSpan(2, 4), ozLineSpan(1, 2), autoPlaced)
		matrix.markAreaAs(axis, ozLineSpan(4, 8), ozLineSpan(0, 2), autoPlaced)
		matrix.markAreaAs(axis, ozLineSpan(8, 10), ozLineSpan(1, 2), cellOccupancyStateDefinitelyPlaced)
		matrix.markAreaAs(axis, ozLineSpan(-3, -1), ozLineSpan(-2, -1), autoPlaced)
		matrix.markAreaAs(axis, ozLineSpan(12, 14), ozLineSpan(7, 8), autoPlaced)
		matrix.markAreaAs(axis, ozLineSpan(0, 1), ozLineSpan(1, 2), autoPlaced)

		assertEq(t, matrix.autoPlacementCursor(axis, 1), some(OriginZeroLine(4)))
		assertEq(t, matrix.autoPlacementCursor(axis, 0), some(OriginZeroLine(8)))
		assertEq(t, matrix.autoPlacementCursor(axis, -2), some(OriginZeroLine(-1)))
		assertEq(t, matrix.autoPlacementCursor(axis, 7), some(OriginZeroLine(14)))
		for _, track := range []OriginZeroLine{-3, -1, 2, 8} {
			assertEq(t, matrix.autoPlacementCursor(axis, track), none[OriginZeroLine]())
		}
	}
}

func TestComputeGridTypesCellOccupancy_CellOccupancyMatrix_AlternatingSpansMergeOccupancyAndSkipTheOccupiedPrefix(t *testing.T) {
	for _, axis := range []AbsoluteAxis{AbsoluteAxisHorizontal, AbsoluteAxisVertical} {
		matrix := cellOccupancyMatrixWithTrackCounts(trackCountsFromRaw(0, 0, 0), trackCountsFromRaw(0, 0, 0))
		for i := range int16(128) {
			matrix.markAreaAs(axis, ozLineSpan(i, i+1), ozLineSpan(i%2, 2), autoPlaced)
		}

		track := &matrix.trackLists(axis.OtherAxis())[1]
		assertIntervals(t, track.intervals, interval(0, 128, autoPlaced))
		if len(track.intervals) > 2 {
			t.Fatalf("intervals spilled the inline capacity of 2: %d", len(track.intervals))
		}
		assertEq(t, matrix.lineAreaCollisionJump(axis, ozLineSpan(0, 1), ozLineSpan(1, 2)), some(OriginZeroLine(128)))
	}
}

func TestComputeGridTypesCellOccupancy_CellOccupancyMatrix_CollisionJumpReturnsNextSearchPosition(t *testing.T) {
	h := AbsoluteAxisHorizontal
	matrix := cellOccupancyMatrixWithTrackCounts(trackCountsFromRaw(0, 4, 0), trackCountsFromRaw(0, 4, 0))
	matrix.markAreaAs(h, ozLineSpan(1, 3), ozLineSpan(0, 1), autoPlaced)

	assertEq(t, matrix.lineAreaCollisionJump(h, ozLineSpan(0, 2), ozLineSpan(0, 1)), some(OriginZeroLine(3)))
	assertEq(t, matrix.lineAreaCollisionJump(h, ozLineSpan(0, 4), ozLineSpan(0, 1)), some(OriginZeroLine(3)))
	assertEq(t, matrix.lineAreaCollisionJump(h, ozLineSpan(0, 4), ozLineSpan(1, 2)), none[OriginZeroLine]())
}

func TestComputeGridTypesCellOccupancy_CellOccupancyMatrix_OccupiedTrackJumpSkipsNonEmptyTracks(t *testing.T) {
	matrix := cellOccupancyMatrixWithTrackCounts(trackCountsFromRaw(0, 4, 0), trackCountsFromRaw(0, 4, 0))
	matrix.markAreaAs(AbsoluteAxisHorizontal, ozLineSpan(0, 1), ozLineSpan(1, 2), autoPlaced)

	assertEq(t, matrix.occupiedTrackJump(AbsoluteAxisVertical, ozLineSpan(0, 4)), some(OriginZeroLine(2)))
	assertEq(t, matrix.occupiedTrackJump(AbsoluteAxisVertical, ozLineSpan(2, 4)), none[OriginZeroLine]())
}
