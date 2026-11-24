package taffy

import (
	"math"
	"testing"
)

func testNamedResolver(explicitTrackCount uint16) namedLineResolver[string] {
	style := StyleDefault[string]()
	resolver := newNamedLineResolver[string](style.Ref(), 0, 0)
	resolver.setExplicitColumnCount(explicitTrackCount)
	return resolver
}

func resolvedStartLine(t *testing.T, resolver *namedLineResolver[string], placement GridPlacement[string]) int16 {
	t.Helper()
	resolved := resolver.resolveColumnNames(Line[GridPlacement[string]]{Start: placement, End: GridPlacementAuto[string]()})
	if resolved.Start.Kind() != GenericGridPlacementKindLine {
		t.Fatal("expected a resolved line")
	}
	return resolved.Start.Line().AsI16()
}

func resolvedEndLine(t *testing.T, resolver *namedLineResolver[string], placement GridPlacement[string]) int16 {
	t.Helper()
	resolved := resolver.resolveColumnNames(Line[GridPlacement[string]]{Start: GridPlacementAuto[string](), End: placement})
	if resolved.End.Kind() != GenericGridPlacementKindLine {
		t.Fatal("expected a resolved line")
	}
	return resolved.End.Line().AsI16()
}

func TestComputeGridTypesNamed_ExtremeMissingNamedLineIndicesDoNotOverflow(t *testing.T) {
	resolver := testNamedResolver(10_000)
	assertEq(t, resolvedStartLine(t, &resolver, GridPlacementNamedLine("missing", math.MaxInt16)), OriginZeroLine(maxOzLine).intoGridLine(10_000).AsI16())
	assertEq(t, resolvedStartLine(t, &resolver, GridPlacementNamedLine("missing", math.MinInt16)), OriginZeroLine(minOzLine).intoGridLine(10_000).AsI16())
}

func TestComputeGridTypesNamed_LargeNamedSpanDoesNotWrapNegative(t *testing.T) {
	resolver := testNamedResolver(10_000)
	resolved := resolver.resolveColumnNames(Line[GridPlacement[string]]{
		Start: GridPlacementLine[string](GridLineFromI16(1)),
		End:   GridPlacementNamedSpan("missing", math.MaxUint16),
	})
	if resolved.End.Kind() != GenericGridPlacementKindLine {
		t.Fatal("expected a resolved line")
	}
	assertEq(t, resolved.End.Line(), OriginZeroLine(maxOzLine).intoGridLine(10_000))
}

func TestComputeGridTypesNamed_AreaLinesSaturateWhenConvertedToGridLines(t *testing.T) {
	style := StyleDefault[string]()
	style.GridTemplateAreas = some(GridTemplateAreas[string]{
		Areas: []GridTemplateArea[string]{{
			Name:        "area",
			RowStart:    1,
			RowEnd:      2,
			ColumnStart: math.MaxUint16,
			ColumnEnd:   math.MaxUint16,
		}},
		RowCount:    1,
		ColumnCount: math.MaxUint16,
	})
	resolver := newNamedLineResolver[string](style.Ref(), 0, 0)
	assertEq(t, resolvedStartLine(t, &resolver, GridPlacementNamedLine("area-start", 1)), OriginZeroLine(maxOzLine).intoGridLine(0).AsI16())
	assertEq(t, resolvedStartLine(t, &resolver, GridPlacementNamedLine("area", 0)), OriginZeroLine(maxOzLine).intoGridLine(0).AsI16())
}

func areaResolver(columnLineNames [5][]string) namedLineResolver[string] {
	style := StyleDefault[string]()
	names := make([][]string, 0, len(columnLineNames))
	for _, set := range columnLineNames {
		names = append(names, append([]string{}, set...))
	}
	style.GridTemplateColumnNames = names
	style.GridTemplateAreas = some(GridTemplateAreas[string]{
		Areas: []GridTemplateArea[string]{{
			Name:        "area",
			RowStart:    1,
			RowEnd:      2,
			ColumnStart: 2,
			ColumnEnd:   4,
		}},
		RowCount:    1,
		ColumnCount: 4,
	})
	resolver := newNamedLineResolver[string](style.Ref(), 0, 0)
	resolver.setExplicitColumnCount(4)
	resolver.setExplicitRowCount(1)
	return resolver
}

func namedLine(name string, index int16) GridPlacement[string] {
	return GridPlacementNamedLine(name, index)
}

func TestComputeGridTypesNamed_BareIdentResolvesToAreaEdge(t *testing.T) {
	resolver := areaResolver([5][]string{{}, {}, {}, {}, {}})
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area", 0)), 2)
	assertEq(t, resolvedEndLine(t, &resolver, namedLine("area", 0)), 4)

	rows := resolver.resolveRowNames(Line[GridPlacement[string]]{Start: namedLine("area", 0), End: namedLine("area", 0)})
	assertEq(t, rows.Start, GenericGridPlacementLine(GridLineFromI16(1)))
	assertEq(t, rows.End, GenericGridPlacementLine(GridLineFromI16(2)))
}

func TestComputeGridTypesNamed_BareIdentResolvesToEdgeOfImplicitlyNamedArea(t *testing.T) {
	resolver := areaResolver([5][]string{{}, {"foo-start"}, {"foo-start"}, {"foo-end"}, {"foo-end"}})
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("foo", 0)), 2)
	assertEq(t, resolvedEndLine(t, &resolver, namedLine("foo", 0)), 4)
}

func TestComputeGridTypesNamed_AreaEdgesAreLinesNamedWithStartAndEndSuffixes(t *testing.T) {
	resolver := areaResolver([5][]string{{}, {}, {}, {}, {}})
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 0)), 2)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 1)), 2)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-end", 1)), 4)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-end", -1)), 4)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 2)), 6)
}

func TestComputeGridTypesNamed_AreaEdgesAreMergedWithExplicitlyNamedLines(t *testing.T) {
	resolver := areaResolver([5][]string{{"area-start"}, {}, {"area-start"}, {"area-end"}, {"area-end"}})
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 1)), 1)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 2)), 2)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 3)), 3)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", -1)), 3)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 4)), 6)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-end", 1)), 4)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-end", 2)), 5)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-end", 3)), 6)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area", 0)), 1)
	assertEq(t, resolvedEndLine(t, &resolver, namedLine("area", 0)), 4)

	span := resolver.resolveColumnNames(Line[GridPlacement[string]]{
		Start: GridPlacementLine[string](GridLineFromI16(1)),
		End:   GridPlacementNamedSpan("area-start", 2),
	})
	assertEq(t, span.End, GenericGridPlacementLine(GridLineFromI16(3)))
}

func TestComputeGridTypesNamed_AreaEdgesAreOnlyMatchedByBareIdents(t *testing.T) {
	resolver := areaResolver([5][]string{{}, {}, {}, {}, {}})
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area", 1)), 6)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area", 2)), 7)

	span := resolver.resolveColumnNames(Line[GridPlacement[string]]{
		Start: GridPlacementLine[string](GridLineFromI16(1)),
		End:   GridPlacementNamedSpan("area", 0),
	})
	assertEq(t, span.End, GenericGridPlacementLine(GridLineFromI16(6)))
}

func TestComputeGridTypesNamed_AreaEdgeIsTheFirstLineWithTheEdgeName(t *testing.T) {
	resolver := areaResolver([5][]string{{}, {}, {"area-start"}, {}, {"area-end"}})
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area", 0)), 2)
	assertEq(t, resolvedEndLine(t, &resolver, namedLine("area", 0)), 4)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 1)), 2)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 2)), 3)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-end", 2)), 5)

	resolver = areaResolver([5][]string{{}, {"area-start"}, {}, {"area-end"}, {}})
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area", 0)), 2)
	assertEq(t, resolvedEndLine(t, &resolver, namedLine("area", 0)), 4)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 1)), 2)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-start", 2)), 6)
	assertEq(t, resolvedStartLine(t, &resolver, namedLine("area-end", 2)), 6)
}
