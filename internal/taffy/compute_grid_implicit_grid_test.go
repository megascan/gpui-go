package taffy

import (
	"testing"
)

type gp = GridPlacement[string]

func gLine(i int16) gp  { return NewLine[gp](i) }
func gSpan(s uint16) gp { return Span[gp](s) }
func gAuto() gp         { return GridPlacementAuto[string]() }

func TestComputeGridImplicitGrid_ChildMinMaxLineAuto(t *testing.T) {
	minCol, maxCol, span := childMinLineMaxLineSpan(LineGridPlacementIntoOriginZeroIgnoringNamed(Line[gp]{Start: gLine(5), End: gSpan(6)}, 6))
	assertEq(t, minCol, OriginZeroLine(4))
	assertEq(t, maxCol, OriginZeroLine(10))
	assertEq(t, span, uint16(1))
}

func TestComputeGridImplicitGrid_ChildMinMaxLineNegativeTrack(t *testing.T) {
	minCol, maxCol, span := childMinLineMaxLineSpan(LineGridPlacementIntoOriginZeroIgnoringNamed(Line[gp]{Start: gLine(-5), End: gSpan(3)}, 6))
	assertEq(t, minCol, OriginZeroLine(2))
	assertEq(t, maxCol, OriginZeroLine(5))
	assertEq(t, span, uint16(1))
}

func implicitGridEstimate(explicitColCount, explicitRowCount uint16, childStyles []Style[string]) (trackCounts, trackCounts) {
	def := StyleDefault[string]()
	nameResolver := newNamedLineResolver[string](def.Ref(), 0, 0)
	nameResolver.setExplicitColumnCount(explicitColCount)
	nameResolver.setExplicitRowCount(explicitRowCount)
	placements := resolveNamedLines(func(yield func(*Style[string]) bool) {
		for i := range childStyles {
			if !yield(&childStyles[i]) {
				return
			}
		}
	}, &nameResolver, explicitColCount, explicitRowCount)
	return computeGridSizeEstimate(explicitColCount, explicitRowCount, placements)
}

func TestComputeGridImplicitGrid_ExplicitGridSizingWithChildren(t *testing.T) {
	var explicitColCount, explicitRowCount uint16 = 6, 8
	childStyles := []Style[string]{
		childTestNode{gLine(1), gSpan(2), gLine(2), gAuto()}.intoGridChild(),
		childTestNode{gLine(-4), gAuto(), gLine(-2), gAuto()}.intoGridChild(),
	}
	inline, block := implicitGridEstimate(explicitColCount, explicitRowCount, childStyles)
	assertEq(t, inline, trackCounts{0, explicitColCount, 0})
	assertEq(t, block, trackCounts{0, explicitRowCount, 0})
}

func TestComputeGridImplicitGrid_NegativeImplicitGridSizing(t *testing.T) {
	var explicitColCount, explicitRowCount uint16 = 4, 4
	childStyles := []Style[string]{
		childTestNode{gLine(-6), gSpan(2), gLine(-8), gAuto()}.intoGridChild(),
		childTestNode{gLine(4), gAuto(), gLine(3), gAuto()}.intoGridChild(),
	}
	inline, block := implicitGridEstimate(explicitColCount, explicitRowCount, childStyles)
	assertEq(t, inline, trackCounts{1, explicitColCount, 0})
	assertEq(t, block, trackCounts{3, explicitRowCount, 0})
}
