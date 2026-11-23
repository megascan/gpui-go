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
