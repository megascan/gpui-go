package taffy

import "fmt"

const maxGridTracks uint16 = 10_000

const minOzLine int16 = -int16(maxGridTracks)

const maxOzLine int16 = int16(maxGridTracks)

// GridLine is a grid line position in "CSS Grid Line" coordinates: line 1 is the start edge of the explicit grid, -1 its end edge, 0 is invalid.
type GridLine struct {
	inner int16
}

// GridLineFromI16 converts an i16 line index (upstream `impl From<i16> for GridLine`).
func GridLineFromI16(value int16) GridLine {
	return GridLine{inner: value}
}

// AsI16 returns the underlying i16.
func (l GridLine) AsI16() int16 {
	return l.inner
}

func (l GridLine) String() string {
	return fmt.Sprintf("GridLine(%d)", l.inner)
}

func (l GridLine) intoOriginZeroLine(explicitTrackCount uint16) OriginZeroLine {
	explicitLineCount := int32(explicitTrackCount) + 1
	var ozLine int32
	switch {
	case l.inner > 0:
		ozLine = int32(l.inner) - 1
	case l.inner < 0:
		ozLine = int32(l.inner) + explicitLineCount
	default:
		panic("Grid line of zero is invalid")
	}
	return originZeroLineClamped(ozLine)
}

// OriginZeroLine is a grid line position in "OriginZero" coordinates: line 0 is the start edge of the explicit grid.
type OriginZeroLine int16

// Add is upstream's `impl Add<OriginZeroLine> for OriginZeroLine`.
func (l OriginZeroLine) Add(rhs OriginZeroLine) OriginZeroLine {
	return l + rhs
}

// Sub is upstream's `impl Sub<OriginZeroLine> for OriginZeroLine`.
func (l OriginZeroLine) Sub(rhs OriginZeroLine) OriginZeroLine {
	return l - rhs
}

// AddU16 is upstream's `impl Add<u16> for OriginZeroLine`.
func (l OriginZeroLine) AddU16(rhs uint16) OriginZeroLine {
	return l + OriginZeroLine(int16(rhs))
}

// AddAssignU16 is upstream's `impl AddAssign<u16> for OriginZeroLine`.
func (l *OriginZeroLine) AddAssignU16(rhs uint16) {
	*l += OriginZeroLine(int16(rhs))
}

// SubU16 is upstream's `impl Sub<u16> for OriginZeroLine`.
func (l OriginZeroLine) SubU16(rhs uint16) OriginZeroLine {
	return l - OriginZeroLine(int16(rhs))
}

func originZeroLineClamped(line int32) OriginZeroLine {
	return OriginZeroLine(int16(min(max(line, int32(minOzLine)), int32(maxOzLine))))
}

func (l OriginZeroLine) intoGridLine(explicitTrackCount uint16) GridLine {
	explicitLineCount := explicitTrackCount + 1
	if l >= 0 {
		return GridLine{inner: int16(l) + 1}
	}
	return GridLine{inner: int16(l) - int16(explicitLineCount)}
}

func (l OriginZeroLine) intoTrackVecIndex(trackCounts trackCounts) int {
	index, ok := l.tryIntoTrackVecIndex(trackCounts).Get()
	if !ok {
		if l > 0 {
			panic("OriginZero grid line cannot be more than the number of positive grid lines")
		}
		panic("OriginZero grid line cannot be less than the number of negative grid lines")
	}
	return index
}

func (l OriginZeroLine) tryIntoTrackVecIndex(trackCounts trackCounts) Option[int] {
	if int16(l) < -int16(trackCounts.negativeImplicit) {
		return none[int]()
	}
	if int16(l) > int16(trackCounts.explicit+trackCounts.positiveImplicit) {
		return none[int]()
	}
	return some(2 * int(uint(int16(l)+int16(trackCounts.negativeImplicit))))
}

func (l OriginZeroLine) impliedNegativeImplicitTracks() uint16 {
	if l < 0 {
		return uint16(-int32(l))
	}
	return 0
}

func (l OriginZeroLine) impliedPositiveImplicitTracks(explicitTrackCount uint16) uint16 {
	if int16(l) > int16(explicitTrackCount) {
		return uint16(l) - explicitTrackCount
	}
	return 0
}

func lineOriginZeroLineSpan(l Line[OriginZeroLine]) uint16 {
	return uint16(max(int16(l.End)-int16(l.Start), 0))
}

// GridCoordinate is the set of coordinate types used to define grid lines.
type GridCoordinate interface {
	GridLine | OriginZeroLine
}
