package taffy

import "iter"

func gridItems(children iter.Seq2[int, *Style[string]]) []gridItem {
	var items []gridItem
	for index, style := range children {
		items = append(items, newGridItemWithStyleAndOrder[string](
			NodeIDFromUsize(index),
			style.Ref(),
			AlignItemsStart,
			AlignItemsStart,
			uint16(index),
		))
	}
	return items
}

func resolveNamedLines(childStyles iter.Seq[*Style[string]], nameResolver *namedLineResolver[string], explicitColCount uint16, explicitRowCount uint16) []itemPlacement {
	var placements []itemPlacement
	for style := range childStyles {
		placements = append(placements, itemPlacement{
			horizontal: lineNonNamedGridPlacementIntoOriginZero(nameResolver.resolveColumnNames(style.GridColumn), explicitColCount),
			vertical:   lineNonNamedGridPlacementIntoOriginZero(nameResolver.resolveRowNames(style.GridRow), explicitRowCount),
		})
	}
	return placements
}

type parentTestNode struct {
	width   float32
	height  float32
	columns int32
	rows    int32
}

func (n parentTestNode) intoGrid() Style[string] {
	style := StyleDefault[string]()
	style.Display = DisplayGrid
	style.Size = Size[Dimension]{Width: Dimension{}.FromLength(float64(n.width)), Height: Dimension{}.FromLength(float64(n.height))}
	style.GridTemplateColumns = repeatFr(int(n.columns))
	style.GridTemplateRows = repeatFr(int(n.rows))
	return style
}

func repeatFr(count int) []GridTemplateComponent[string] {
	out := make([]GridTemplateComponent[string], count)
	for i := range out {
		out[i] = Fr[GridTemplateComponent[string]](1)
	}
	return out
}

type childTestNode struct {
	columnStart GridPlacement[string]
	columnEnd   GridPlacement[string]
	rowStart    GridPlacement[string]
	rowEnd      GridPlacement[string]
}

func (n childTestNode) intoGridChild() Style[string] {
	style := StyleDefault[string]()
	style.Display = DisplayGrid
	style.GridColumn = Line[GridPlacement[string]]{Start: n.columnStart, End: n.columnEnd}
	style.GridRow = Line[GridPlacement[string]]{Start: n.rowStart, End: n.rowEnd}
	return style
}

type expectedPlacement struct {
	a, b, c, d int16
}

func (p expectedPlacement) intoOz() (OriginZeroLine, OriginZeroLine, OriginZeroLine, OriginZeroLine) {
	return OriginZeroLine(p.a), OriginZeroLine(p.b), OriginZeroLine(p.c), OriginZeroLine(p.d)
}
