package taffy

func advancePosition(position OriginZeroLine) OriginZeroLine {
	if position == OriginZeroLine(int16(0x7FFF)) {
		return position
	}
	return position + 1
}

func resolveIndefiniteGridSpan(position OriginZeroLine, span uint16) Line[OriginZeroLine] {
	p := int32(position)
	s := int32(span)
	line := func(value int32) OriginZeroLine {
		return OriginZeroLine(int16(min(max(value, -0x8000), 0x7FFF)))
	}
	return Line[OriginZeroLine]{Start: line(p), End: line(p + s)}
}

type itemPlacement = inBothAbsAxis[Line[originZeroGridPlacement]]

func placeGridItems(cellOccupancyMatrix *cellOccupancyMatrix, items []gridItem, placements []itemPlacement, gridAutoFlow GridAutoFlow) {
	primaryAxis := gridAutoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()

	for i := range items {
		item, placement := &items[i], placements[i]
		if !(LineOriginZeroGridPlacementIsDefinite(placement.horizontal) && LineOriginZeroGridPlacementIsDefinite(placement.vertical)) {
			continue
		}
		rowSpan, colSpan := placeDefiniteGridItem(placement, primaryAxis)
		recordGridPlacement(cellOccupancyMatrix, item, primaryAxis, rowSpan, colSpan, cellOccupancyStateDefinitelyPlaced)
	}

	for i := range items {
		item, placement := &items[i], placements[i]
		if !(LineOriginZeroGridPlacementIsDefinite(placement.get(secondaryAxis)) && !LineOriginZeroGridPlacementIsDefinite(placement.get(primaryAxis))) {
			continue
		}
		primarySpan, secondarySpan := placeDefiniteSecondaryAxisItem(cellOccupancyMatrix, placement, gridAutoFlow)
		recordGridPlacement(cellOccupancyMatrix, item, primaryAxis, primarySpan, secondarySpan, cellOccupancyStateAutoPlaced)
	}

	primaryAxis = gridAutoFlow.PrimaryAxis()
	secondaryAxis = primaryAxis.OtherAxis()
	primaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(primaryAxis).implicitStartLine()
	secondaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(secondaryAxis).implicitStartLine()
	gridStartPosition := [2]OriginZeroLine{primaryAxisGridStartLine, secondaryAxisGridStartLine}
	gridPosition := gridStartPosition
	for i := range items {
		item, placement := &items[i], placements[i]
		if LineOriginZeroGridPlacementIsDefinite(placement.get(secondaryAxis)) {
			continue
		}
		primarySpan, secondarySpan := placeIndefinitelyPositionedItem(cellOccupancyMatrix, placement, gridAutoFlow, gridPosition)
		recordGridPlacement(cellOccupancyMatrix, item, primaryAxis, primarySpan, secondarySpan, cellOccupancyStateAutoPlaced)
		if gridAutoFlow.IsDense() {
			gridPosition = gridStartPosition
		} else {
			gridPosition = [2]OriginZeroLine{primarySpan.End, secondarySpan.Start}
		}
	}
}

func placeDefiniteGridItem(placement itemPlacement, primaryAxis AbsoluteAxis) (Line[OriginZeroLine], Line[OriginZeroLine]) {
	primarySpan := LineOriginZeroGridPlacementResolveDefiniteGridLines(placement.get(primaryAxis))
	secondarySpan := LineOriginZeroGridPlacementResolveDefiniteGridLines(placement.get(primaryAxis.OtherAxis()))
	return primarySpan, secondarySpan
}

func placeDefiniteSecondaryAxisItem(cellOccupancyMatrix *cellOccupancyMatrix, placement itemPlacement, autoFlow GridAutoFlow) (Line[OriginZeroLine], Line[OriginZeroLine]) {
	primaryAxis := autoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()
	primaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(primaryAxis).implicitStartLine()

	secondaryAxisPlacement := LineOriginZeroGridPlacementResolveDefiniteGridLines(placement.get(secondaryAxis))
	var startingPosition OriginZeroLine
	if autoFlow.IsDense() {
		startingPosition = primaryAxisGridStartLine
	} else {
		startingPosition = cellOccupancyMatrix.autoPlacementCursor(primaryAxis, secondaryAxisPlacement.Start).UnwrapOr(primaryAxisGridStartLine)
	}
	primaryAxisSpan := LineGenericGridPlacementIndefiniteSpan(placement.get(primaryAxis))

	position := startingPosition
	for {
		primaryAxisPlacement := resolveIndefiniteGridSpan(position, primaryAxisSpan)
		nextPosition, ok := cellOccupancyMatrix.lineAreaCollisionJump(primaryAxis, primaryAxisPlacement, secondaryAxisPlacement).Get()
		if !ok {
			return primaryAxisPlacement, secondaryAxisPlacement
		}
		position = nextPosition
	}
}

func placeIndefinitelyPositionedItem(cellOccupancyMatrix *cellOccupancyMatrix, placement itemPlacement, autoFlow GridAutoFlow, gridPosition [2]OriginZeroLine) (Line[OriginZeroLine], Line[OriginZeroLine]) {
	primaryAxis := autoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()

	primaryPlacementStyle := placement.get(primaryAxis)
	secondaryPlacementStyle := placement.get(secondaryAxis)

	secondarySpanCount := LineGenericGridPlacementIndefiniteSpan(secondaryPlacementStyle)
	hasDefinitePrimaryAxisPosition := LineOriginZeroGridPlacementIsDefinite(primaryPlacementStyle)
	primaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(primaryAxis).implicitStartLine()
	primaryAxisGridEndLine := cellOccupancyMatrix.trackCounts(primaryAxis).implicitEndLine()
	secondaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(secondaryAxis).implicitStartLine()

	primaryIdx, secondaryIdx := gridPosition[0], gridPosition[1]

	if hasDefinitePrimaryAxisPosition {
		primarySpan := LineOriginZeroGridPlacementResolveDefiniteGridLines(primaryPlacementStyle)

		if autoFlow.IsDense() {
			secondaryIdx = secondaryAxisGridStartLine
		} else if primarySpan.Start < primaryIdx {
			secondaryIdx = advancePosition(secondaryIdx)
		}

		for {
			secondarySpan := resolveIndefiniteGridSpan(secondaryIdx, secondarySpanCount)
			if nextPosition, ok := cellOccupancyMatrix.lineAreaCollisionJump(secondaryAxis, secondarySpan, primarySpan).Get(); ok {
				secondaryIdx = nextPosition
				continue
			}
			return primarySpan, secondarySpan
		}
	}

	primarySpanCount := LineGenericGridPlacementIndefiniteSpan(primaryPlacementStyle)
	spansAllPrimaryTracks := int(primarySpanCount) >= cellOccupancyMatrix.trackCounts(primaryAxis).len()

	for {
		primarySpan := resolveIndefiniteGridSpan(primaryIdx, primarySpanCount)
		secondarySpan := resolveIndefiniteGridSpan(secondaryIdx, secondarySpanCount)

		primaryOutOfBounds := primarySpan.End > primaryAxisGridEndLine
		if primaryOutOfBounds {
			if primaryIdx == primaryAxisGridStartLine {
				return primarySpan, secondarySpan
			}
			secondaryIdx = advancePosition(secondaryIdx)
			primaryIdx = primaryAxisGridStartLine
			continue
		}

		if spansAllPrimaryTracks {
			nextPosition, ok := cellOccupancyMatrix.occupiedTrackJump(secondaryAxis, secondarySpan).Get()
			if ok {
				secondaryIdx = nextPosition
				primaryIdx = primaryAxisGridStartLine
				continue
			}
			return primarySpan, secondarySpan
		}

		if nextPosition, ok := cellOccupancyMatrix.lineAreaCollisionJump(primaryAxis, primarySpan, secondarySpan).Get(); ok {
			primaryIdx = nextPosition
			continue
		}

		return primarySpan, secondarySpan
	}
}

func clampSpanToLimitedGrid(span Line[OriginZeroLine]) Line[OriginZeroLine] {
	start := min(max(int16(span.Start), minOzLine), maxOzLine-1)
	end := min(max(int16(span.End), start+1), maxOzLine)
	return Line[OriginZeroLine]{Start: OriginZeroLine(start), End: OriginZeroLine(end)}
}

func recordGridPlacement(cellOccupancyMatrix *cellOccupancyMatrix, item *gridItem, primaryAxis AbsoluteAxis, primarySpan Line[OriginZeroLine], secondarySpan Line[OriginZeroLine], placementType cellOccupancyState) {
	primarySpan = clampSpanToLimitedGrid(primarySpan)
	secondarySpan = clampSpanToLimitedGrid(secondarySpan)

	cellOccupancyMatrix.markAreaAs(primaryAxis, primarySpan, secondarySpan, placementType)

	var colSpan, rowSpan Line[OriginZeroLine]
	switch primaryAxis {
	case AbsoluteAxisHorizontal:
		colSpan, rowSpan = primarySpan, secondarySpan
	default:
		colSpan, rowSpan = secondarySpan, primarySpan
	}
	item.column = colSpan
	item.row = rowSpan
}
