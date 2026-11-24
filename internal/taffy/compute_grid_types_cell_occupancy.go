package taffy

import (
	"fmt"
	"slices"
	"strings"
)

type cellOccupancyState uint8

const (
	cellOccupancyStateUnoccupied cellOccupancyState = iota
	cellOccupancyStateDefinitelyPlaced
	cellOccupancyStateAutoPlaced
)

type occupiedInterval struct {
	rng   rangeI16
	state cellOccupancyState
}

func (i occupiedInterval) overlaps(r rangeI16) bool {
	return i.rng.start < r.end && i.rng.end > r.start
}

type trackIntervals struct {
	intervals           []occupiedInterval
	autoPlacementCursor Option[OriginZeroLine]
}

func (t *trackIntervals) isEmpty() bool {
	return len(t.intervals) == 0
}

func (t *trackIntervals) stateAt(coordinate int16) cellOccupancyState {
	for _, interval := range t.intervals {
		if interval.rng.start <= coordinate && coordinate < interval.rng.end {
			return interval.state
		}
	}
	return cellOccupancyStateUnoccupied
}

func (t *trackIntervals) paint(r rangeI16, state cellOccupancyState) {
	if r.start >= r.end {
		return
	}

	if len(t.intervals) == 0 {
		t.intervals = append(t.intervals, occupiedInterval{rng: r, state: state})
		return
	}
	last := &t.intervals[len(t.intervals)-1]
	if last.rng.end <= r.start {
		if last.state == state && last.rng.end == r.start {
			last.rng.end = r.end
		} else {
			t.intervals = append(t.intervals, occupiedInterval{rng: r, state: state})
		}
		return
	}

	result := make([]occupiedInterval, 0, max(2, len(t.intervals)+2))

	for _, interval := range t.intervals {
		if interval.rng.end <= r.start {
			result = append(result, interval)
		} else if interval.rng.start < r.start {
			result = append(result, occupiedInterval{rng: rangeI16{start: interval.rng.start, end: r.start}, state: interval.state})
		}
	}

	if n := len(result); n > 0 && result[n-1].state == state && result[n-1].rng.end == r.start {
		result[n-1].rng.end = r.end
	} else {
		result = append(result, occupiedInterval{rng: r, state: state})
	}

	for _, interval := range t.intervals {
		var trimmed occupiedInterval
		if interval.rng.start >= r.end {
			trimmed = interval
		} else if interval.rng.end > r.end {
			trimmed = occupiedInterval{rng: rangeI16{start: r.end, end: interval.rng.end}, state: interval.state}
		} else {
			continue
		}
		if n := len(result); n > 0 && result[n-1].state == trimmed.state && result[n-1].rng.end == trimmed.rng.start {
			result[n-1].rng.end = trimmed.rng.end
		} else {
			result = append(result, trimmed)
		}
	}

	t.intervals = result
}

func (t *trackIntervals) collisionExtent(r rangeI16) Option[int16] {
	for i := len(t.intervals) - 1; i >= 0; i-- {
		interval := t.intervals[i]
		if !(interval.rng.end > r.start) {
			break
		}
		if interval.overlaps(r) {
			return some(interval.rng.end - 1)
		}
	}
	return none[int16]()
}

type cellOccupancyMatrix struct {
	columns         trackCounts
	rows            trackCounts
	rowIntervals    []trackIntervals
	columnIntervals []trackIntervals
}

func (m *cellOccupancyMatrix) String() string {
	var f strings.Builder
	fmt.Fprintf(&f, "Rows: neg_implicit=%d explicit=%d pos_implicit=%d\n", m.rows.negativeImplicit, m.rows.explicit, m.rows.positiveImplicit)
	fmt.Fprintf(&f, "Cols: neg_implicit=%d explicit=%d pos_implicit=%d\n", m.columns.negativeImplicit, m.columns.explicit, m.columns.positiveImplicit)
	if m.rows.len() > 100 || m.columns.len() > 100 {
		f.WriteString("State: (not printed: more than 100 tracks)\n")
		return f.String()
	}
	f.WriteString("State:\n")

	for i := range m.rowIntervals {
		row := &m.rowIntervals[i]
		for columnIndex := range m.columns.len() {
			coordinate := m.columns.trackToPrevOzLine(uint16(columnIndex))
			var letter byte
			switch row.stateAt(int16(coordinate)) {
			case cellOccupancyStateUnoccupied:
				letter = '_'
			case cellOccupancyStateDefinitelyPlaced:
				letter = 'D'
			case cellOccupancyStateAutoPlaced:
				letter = 'A'
			}
			f.WriteByte(letter)
		}
		f.WriteByte('\n')
	}

	return f.String()
}

func cellOccupancyMatrixWithTrackCounts(columns, rows trackCounts) cellOccupancyMatrix {
	rowIntervals := make([]trackIntervals, rows.len())
	columnIntervals := make([]trackIntervals, columns.len())
	return cellOccupancyMatrix{rows: rows, columns: columns, rowIntervals: rowIntervals, columnIntervals: columnIntervals}
}

func (m *cellOccupancyMatrix) trackLists(trackAxis AbsoluteAxis) []trackIntervals {
	switch trackAxis {
	case AbsoluteAxisHorizontal:
		return m.columnIntervals
	default:
		return m.rowIntervals
	}
}

func (m *cellOccupancyMatrix) expandToFitRange(rowSpan, colSpan Line[OriginZeroLine]) {
	reqNegativeRows := max(-int16(m.rows.negativeImplicit)-int16(rowSpan.Start), 0)
	reqPositiveRows := max(int16(rowSpan.End)-int16(m.rows.implicitEndLine()), 0)
	reqNegativeCols := max(-int16(m.columns.negativeImplicit)-int16(colSpan.Start), 0)
	reqPositiveCols := max(int16(colSpan.End)-int16(m.columns.implicitEndLine()), 0)

	if reqNegativeRows > 0 {
		m.rowIntervals = slices.Insert(m.rowIntervals, 0, make([]trackIntervals, int(reqNegativeRows))...)
	}
	if reqPositiveRows > 0 {
		m.rowIntervals = append(m.rowIntervals, make([]trackIntervals, int(reqPositiveRows))...)
	}
	if reqNegativeCols > 0 {
		m.columnIntervals = slices.Insert(m.columnIntervals, 0, make([]trackIntervals, int(reqNegativeCols))...)
	}
	if reqPositiveCols > 0 {
		m.columnIntervals = append(m.columnIntervals, make([]trackIntervals, int(reqPositiveCols))...)
	}

	m.rows.negativeImplicit += uint16(reqNegativeRows)
	m.rows.positiveImplicit += uint16(reqPositiveRows)
	m.columns.negativeImplicit += uint16(reqNegativeCols)
	m.columns.positiveImplicit += uint16(reqPositiveCols)
}

func (m *cellOccupancyMatrix) markAreaAs(primaryAxis AbsoluteAxis, primarySpan, secondarySpan Line[OriginZeroLine], value cellOccupancyState) {
	var rowSpan, columnSpan Line[OriginZeroLine]
	switch primaryAxis {
	case AbsoluteAxisHorizontal:
		rowSpan, columnSpan = secondarySpan, primarySpan
	default:
		rowSpan, columnSpan = primarySpan, secondarySpan
	}

	m.expandToFitRange(rowSpan, columnSpan)

	if value == cellOccupancyStateAutoPlaced && primarySpan.Start < primarySpan.End && secondarySpan.Start < secondarySpan.End {
		m.advanceAutoPlacementCursor(primaryAxis, secondarySpan.Start, primarySpan.End)
	}

	rowRange := m.rows.ozLineRangeToTrackRange(rowSpan)
	colRange := m.columns.ozLineRangeToTrackRange(columnSpan)
	for rowIndex := rowRange.start; rowIndex < rowRange.end; rowIndex++ {
		m.rowIntervals[int(rowIndex)].paint(rangeI16{start: int16(columnSpan.Start), end: int16(columnSpan.End)}, value)
	}
	for columnIndex := colRange.start; columnIndex < colRange.end; columnIndex++ {
		m.columnIntervals[int(columnIndex)].paint(rangeI16{start: int16(rowSpan.Start), end: int16(rowSpan.End)}, value)
	}
}

func (m *cellOccupancyMatrix) lineAreaIsUnoccupied(primaryAxis AbsoluteAxis, primarySpan, secondarySpan Line[OriginZeroLine]) bool {
	return m.lineAreaCollisionJump(primaryAxis, primarySpan, secondarySpan).IsNone()
}

func (m *cellOccupancyMatrix) lineAreaCollisionJump(primaryAxis AbsoluteAxis, primarySpan, secondarySpan Line[OriginZeroLine]) Option[OriginZeroLine] {
	trackLists := m.trackLists(primaryAxis.OtherAxis())
	secondaryCounts := m.trackCounts(primaryAxis.OtherAxis())
	secondaryRange := secondaryCounts.ozLineRangeToTrackRange(secondarySpan)

	secondaryStart := max(secondaryRange.start, 0)
	secondaryEnd := min(secondaryRange.end, int16(len(trackLists)))

	primaryRange := rangeI16{start: int16(primarySpan.Start), end: int16(primarySpan.End)}

	extent := none[int16]()
	for secondaryIndex := secondaryStart; secondaryIndex < secondaryEnd; secondaryIndex++ {
		cell, ok := trackLists[int(secondaryIndex)].collisionExtent(primaryRange).Get()
		if !ok {
			continue
		}
		if best, ok := extent.Get(); ok {
			extent = some(max(best, cell))
		} else {
			extent = some(cell)
		}
	}

	if cell, ok := extent.Get(); ok {
		return some(OriginZeroLine(cell + 1))
	}
	return none[OriginZeroLine]()
}

func (m *cellOccupancyMatrix) occupiedTrackJump(axis AbsoluteAxis, span Line[OriginZeroLine]) Option[OriginZeroLine] {
	counts := m.trackCounts(axis)
	trackLists := m.trackLists(axis)
	r := counts.ozLineRangeToTrackRange(span)
	start := max(r.start, 0)
	end := min(r.end, int16(len(trackLists)))
	for index := end - 1; index >= start; index-- {
		if !trackLists[int(index)].isEmpty() {
			line := counts.trackToPrevOzLine(uint16(index))
			return some(line.AddU16(1))
		}
	}
	return none[OriginZeroLine]()
}

func (m *cellOccupancyMatrix) rowIsOccupied(rowIndex int) bool {
	tracks := m.trackLists(AbsoluteAxisVertical)
	return rowIndex >= 0 && rowIndex < len(tracks) && !tracks[rowIndex].isEmpty()
}

func (m *cellOccupancyMatrix) columnIsOccupied(columnIndex int) bool {
	tracks := m.trackLists(AbsoluteAxisHorizontal)
	return columnIndex >= 0 && columnIndex < len(tracks) && !tracks[columnIndex].isEmpty()
}

func (m *cellOccupancyMatrix) trackCounts(trackType AbsoluteAxis) *trackCounts {
	switch trackType {
	case AbsoluteAxisHorizontal:
		return &m.columns
	default:
		return &m.rows
	}
}

func (m *cellOccupancyMatrix) autoPlacementCursor(trackType AbsoluteAxis, startAt OriginZeroLine) Option[OriginZeroLine] {
	trackCounts := m.trackCounts(trackType.OtherAxis())
	trackComputedIndex := trackCounts.ozLineToNextTrack(startAt)
	trackLists := m.trackLists(trackType.OtherAxis())
	if trackComputedIndex < 0 || trackComputedIndex >= int16(len(trackLists)) {
		return none[OriginZeroLine]()
	}
	return trackLists[int(trackComputedIndex)].autoPlacementCursor
}

func (m *cellOccupancyMatrix) advanceAutoPlacementCursor(primaryAxis AbsoluteAxis, secondaryStart, primaryEnd OriginZeroLine) {
	trackIndex := int(uint(m.trackCounts(primaryAxis.OtherAxis()).ozLineToNextTrack(secondaryStart)))
	var tracks []trackIntervals
	switch primaryAxis {
	case AbsoluteAxisHorizontal:
		tracks = m.rowIntervals
	default:
		tracks = m.columnIntervals
	}
	cursor := &tracks[trackIndex].autoPlacementCursor
	if current, ok := cursor.Get(); ok {
		*cursor = some(max(current, primaryEnd))
	} else {
		*cursor = some(primaryEnd)
	}
}
