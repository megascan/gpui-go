package taffy

func computeGridSizeEstimate(explicitColCount uint16, explicitRowCount uint16, placements []itemPlacement) (trackCounts, trackCounts) {
	colMin, colMax, colMaxSpan, rowMin, rowMax, rowMaxSpan := getKnownChildPositions(placements)

	negativeImplicitInlineTracks := colMin.impliedNegativeImplicitTracks()
	explicitInlineTracks := explicitColCount
	positiveImplicitInlineTracks := colMax.impliedPositiveImplicitTracks(explicitColCount)
	negativeImplicitBlockTracks := rowMin.impliedNegativeImplicitTracks()
	explicitBlockTracks := explicitRowCount
	positiveImplicitBlockTracks := rowMax.impliedPositiveImplicitTracks(explicitRowCount)

	totInlineTracks := negativeImplicitInlineTracks + explicitInlineTracks + positiveImplicitInlineTracks
	if totInlineTracks < colMaxSpan {
		positiveImplicitInlineTracks = colMaxSpan - explicitInlineTracks - negativeImplicitInlineTracks
	}

	totBlockTracks := negativeImplicitBlockTracks + explicitBlockTracks + positiveImplicitBlockTracks
	if totBlockTracks < rowMaxSpan {
		positiveImplicitBlockTracks = rowMaxSpan - explicitBlockTracks - negativeImplicitBlockTracks
	}

	columnCounts := trackCountsFromRaw(negativeImplicitInlineTracks, explicitInlineTracks, positiveImplicitInlineTracks)
	rowCounts := trackCountsFromRaw(negativeImplicitBlockTracks, explicitBlockTracks, positiveImplicitBlockTracks)
	return columnCounts, rowCounts
}

func getKnownChildPositions(placements []itemPlacement) (OriginZeroLine, OriginZeroLine, uint16, OriginZeroLine, OriginZeroLine, uint16) {
	var colMin, colMax OriginZeroLine
	var colMaxSpan uint16
	var rowMin, rowMax OriginZeroLine
	var rowMaxSpan uint16
	for _, placement := range placements {
		childColMin, childColMax, childColSpan := childMinLineMaxLineSpan(placement.horizontal)
		childRowMin, childRowMax, childRowSpan := childMinLineMaxLineSpan(placement.vertical)

		colMin = min(colMin, childColMin)
		colMax = max(colMax, childColMax)
		colMaxSpan = max(colMaxSpan, childColSpan)
		rowMin = min(rowMin, childRowMin)
		rowMax = max(rowMax, childRowMax)
		rowMaxSpan = max(rowMaxSpan, childRowSpan)
	}
	return colMin, colMax, colMaxSpan, rowMin, rowMax, rowMaxSpan
}

func childMinLineMaxLineSpan(ozLine Line[originZeroGridPlacement]) (OriginZeroLine, OriginZeroLine, uint16) {
	const (
		auto = GenericGridPlacementKindAuto
		line = GenericGridPlacementKindLine
		span = GenericGridPlacementKindSpan
	)
	start, end := ozLine.Start, ozLine.End

	var minLine OriginZeroLine
	switch {
	case start.kind == line && end.kind == line:
		if start.line == end.line {
			minLine = start.line
		} else {
			minLine = min(start.line, end.line)
		}
	case start.kind == line && end.kind == auto:
		minLine = start.line
	case start.kind == line && end.kind == span:
		minLine = start.line
	case start.kind == auto && end.kind == line:
		minLine = end.line.SubU16(1)
	case start.kind == span && end.kind == line:
		minLine = end.line.SubU16(start.span)
	default:
		minLine = OriginZeroLine(0)
	}

	var maxLine OriginZeroLine
	switch {
	case start.kind == line && end.kind == line:
		if start.line == end.line {
			maxLine = start.line.AddU16(1)
		} else {
			maxLine = max(start.line, end.line)
		}
	case start.kind == line && end.kind == auto:
		maxLine = start.line.AddU16(1)
	case start.kind == line && end.kind == span:
		maxLine = start.line.AddU16(end.span)
	case start.kind == auto && end.kind == line:
		maxLine = end.line
	case start.kind == span && end.kind == line:
		maxLine = end.line
	default:
		maxLine = OriginZeroLine(0)
	}

	var spanCount uint16
	if start.kind != line && end.kind != line {
		spanCount = LineGenericGridPlacementIndefiniteSpan(ozLine)
	} else {
		spanCount = 1
	}

	clampedMin := OriginZeroLine(max(int16(minLine), minOzLine))
	clampedMax := OriginZeroLine(min(int16(maxLine), maxOzLine))
	return clampedMin, clampedMax, spanCount
}
