package taffy

type trackCounts struct {
	negativeImplicit uint16
	explicit         uint16
	positiveImplicit uint16
}

type rangeI16 struct {
	start int16
	end   int16
}

func trackCountsFromRaw(negativeImplicit, explicit, positiveImplicit uint16) trackCounts {
	return trackCounts{negativeImplicit: negativeImplicit, explicit: explicit, positiveImplicit: positiveImplicit}
}

func (c trackCounts) len() int {
	return int(c.negativeImplicit + c.explicit + c.positiveImplicit)
}

func (c trackCounts) implicitStartLine() OriginZeroLine {
	return OriginZeroLine(-int16(c.negativeImplicit))
}

func (c trackCounts) implicitEndLine() OriginZeroLine {
	return OriginZeroLine(int16(c.explicit + c.positiveImplicit))
}

func (c trackCounts) ozLineToNextTrack(index OriginZeroLine) int16 {
	return int16(index) + int16(c.negativeImplicit)
}

func (c trackCounts) ozLineRangeToTrackRange(input Line[OriginZeroLine]) rangeI16 {
	start := c.ozLineToNextTrack(input.Start)
	end := c.ozLineToNextTrack(input.End)
	return rangeI16{start: start, end: end}
}

func (c trackCounts) trackToPrevOzLine(index uint16) OriginZeroLine {
	return OriginZeroLine(int16(index) - int16(c.negativeImplicit))
}

func (c trackCounts) trackRangeToOzLineRange(input rangeI16) Line[OriginZeroLine] {
	start := c.trackToPrevOzLine(uint16(input.start))
	end := c.trackToPrevOzLine(uint16(input.end))
	return Line[OriginZeroLine]{Start: start, End: end}
}
