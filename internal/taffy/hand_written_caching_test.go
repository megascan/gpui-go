package taffy

import "testing"

type gpl = GridPlacement[defaultCheapStr]

var cachingNodeContext = testNodeContextFixed(50.0, 50.0)

const cachingText = "HH​HH​HH​HH"

func cachingGridStyle() Style[defaultCheapStr] {
	s := defaultStyle()
	s.Display = DisplayGrid
	return s
}

func TestHandWrittenCaching_MeasureCountFlexbox(t *testing.T) {
	taffy := newTestTree()

	leaf := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), cachingNodeContext))

	node := hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{leaf}))
	for range 100 {
		node = hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{node}))
	}

	hwOK(taffy.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), testMeasureFunction))

	hwAssertEq(t, taffy.GetNodeContextMut(leaf).Count, 7)
}

func TestHandWrittenCaching_MeasureCountGrid(t *testing.T) {
	taffy := newTestTree()

	leaf := hwUnwrap(taffy.NewLeafWithContext(cachingGridStyle(), cachingNodeContext))

	node := hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{leaf}))
	for range 100 {
		node = hwUnwrap(taffy.NewWithChildren(defaultStyle(), []NodeID{node}))
	}

	hwOK(taffy.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), testMeasureFunction))
	hwAssertEq(t, taffy.GetNodeContextMut(leaf).Count, 7)
}

func TestHandWrittenCaching_MeasureCountNestedGrid(t *testing.T) {
	taffy := newTestTree()

	text := testNodeContextAhemText(cachingText, WritingModeHorizontal)
	leaf := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), text))
	inner := hwUnwrap(taffy.NewWithChildren(cachingGridStyle(), []NodeID{leaf}))
	middle := hwUnwrap(taffy.NewWithChildren(cachingGridStyle(), []NodeID{inner}))
	os := cachingGridStyle()
	os.Size = hwSizeDim(hwDim(50.0), DimensionAuto())
	outer := hwUnwrap(taffy.NewWithChildren(os, []NodeID{middle}))

	hwOK(taffy.ComputeLayoutWithMeasure(outer, SizeMaxContent[AvailableSpace](), testMeasureFunction))
	hwAssertEq(t, hwLayout(taffy, leaf).Size, Size[float32]{Width: 50.0, Height: 20.0})
	hwAssertEq(t, taffy.GetNodeContextMut(leaf).Count, 6)
}

func TestHandWrittenCaching_MeasureCountGridItemWithKnownWidth(t *testing.T) {
	taffy := newTestTree()

	text := testNodeContextAhemText(cachingText, WritingModeHorizontal)
	ls := defaultStyle()
	ls.Size = hwSizeDim(hwDim(40.0), DimensionAuto())
	leaf := hwUnwrap(taffy.NewLeafWithContext(ls, text))
	grid := hwUnwrap(taffy.NewWithChildren(cachingGridStyle(), []NodeID{leaf}))

	hwOK(taffy.ComputeLayoutWithMeasure(grid, SizeMaxContent[AvailableSpace](), testMeasureFunction))
	hwAssertEq(t, hwLayout(taffy, grid).Size, Size[float32]{Width: 40.0, Height: 20.0})
	hwAssertEq(t, hwLayout(taffy, leaf).Size, Size[float32]{Width: 40.0, Height: 20.0})
	hwAssertEq(t, taffy.GetNodeContextMut(leaf).Count, 3)
}

func TestHandWrittenCaching_MeasureCountGridItemInFixedTrack(t *testing.T) {
	taffy := newTestTree()

	text := testNodeContextAhemText(cachingText, WritingModeHorizontal)
	ls := defaultStyle()
	ls.JustifySelf = some(AlignItemsStart)
	ls.GridColumn = NewLine[Line[gpl]](1)
	leaf := hwUnwrap(taffy.NewLeafWithContext(ls, text))
	ss := defaultStyle()
	ss.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
	ss.GridColumn = NewLine[Line[gpl]](2)
	sibling := hwUnwrap(taffy.NewLeaf(ss))
	gs := cachingGridStyle()
	gs.GridTemplateColumns = []gtc{Length[gtc](100.0), Auto[gtc]()}
	grid := hwUnwrap(taffy.NewWithChildren(gs, []NodeID{leaf, sibling}))

	hwOK(taffy.ComputeLayoutWithMeasure(grid, SizeMaxContent[AvailableSpace](), testMeasureFunction))
	hwAssertEq(t, hwLayout(taffy, grid).Size, Size[float32]{Width: 150.0, Height: 50.0})
	hwAssertEq(t, hwLayout(taffy, leaf).Size, Size[float32]{Width: 80.0, Height: 50.0})
	hwAssertEq(t, taffy.GetNodeContextMut(leaf).Count, 3)
}

func TestHandWrittenCaching_MeasureCountFlexInGrid(t *testing.T) {
	cases := []struct {
		flexDirection FlexDirection
		expectedCount int
	}{{FlexDirectionRow, 10}, {FlexDirectionColumn, 11}}
	for _, c := range cases {
		taffy := newTestTree()

		text := testNodeContextAhemText(cachingText, WritingModeHorizontal)
		leaf := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), text))
		fs := defaultStyle()
		fs.Display = DisplayFlex
		fs.FlexDirection = c.flexDirection
		flex := hwUnwrap(taffy.NewWithChildren(fs, []NodeID{leaf}))
		gs := cachingGridStyle()
		gs.Size = hwSizeDim(hwDim(50.0), DimensionAuto())
		grid := hwUnwrap(taffy.NewWithChildren(gs, []NodeID{flex}))

		hwOK(taffy.ComputeLayoutWithMeasure(grid, SizeMaxContent[AvailableSpace](), testMeasureFunction))
		count := taffy.GetNodeContextMut(leaf).Count
		hwAssertEq(t, hwLayout(taffy, leaf).Size, Size[float32]{Width: 50.0, Height: 20.0})
		hwAssertEq(t, count, c.expectedCount)
	}
}

func TestHandWrittenCaching_GridBlockItemHeightNotPollutedByWidthMeasure(t *testing.T) {
	taffy := newTestTree()

	itemStyle := func(row int16) Style[defaultCheapStr] {
		s := defaultStyle()
		s.Display = DisplayBlock
		s.GridRow = Line[gpl]{Start: NewLine[gpl](row), End: NewLine[gpl](row + 1)}
		s.GridColumn = Line[gpl]{Start: NewLine[gpl](1), End: NewLine[gpl](2)}
		return s
	}

	makeItem := func(row int16) NodeID {
		leaf := hwUnwrap(taffy.NewLeafWithContext(defaultStyle(), testNodeContextFixed(100.0, 19.0)))
		return hwUnwrap(taffy.NewWithChildren(itemStyle(row), []NodeID{leaf}))
	}
	a := makeItem(1)
	b := makeItem(2)
	c := makeItem(3)

	cs := cachingGridStyle()
	cs.GridTemplateColumns = []gtc{Percent[gtc](0.6), Auto[gtc]()}
	cs.Size = hwSizeDim(hwDim(1000.0), DimensionAuto())
	container := hwUnwrap(taffy.NewWithChildren(cs, []NodeID{a, b, c}))

	hwOK(taffy.ComputeLayoutWithMeasure(container, hwDefinite(1000.0, 1000.0), testMeasureFunction))

	for index, node := range []NodeID{a, b, c} {
		layout := hwLayout(taffy, node)
		hwAssertEq(t, layout.Size.Height, 19.0)
		hwAssertEq(t, layout.Location.Y, 19.0*float32(index))
	}
}

func TestHandWrittenCaching_GridItemSpanningOnlyUnaffectedTracksIsNotMeasured(t *testing.T) {
	taffy := newTestTree()

	text := testNodeContextAhemText("HH HH HH HH", WritingModeHorizontal)
	ls := defaultStyle()
	ls.GridColumn = Line[gpl]{Start: NewLine[gpl](2), End: NewLine[gpl](3)}
	leaf := hwUnwrap(taffy.NewLeafWithContext(ls, text))
	gs := cachingGridStyle()
	gs.Size = hwSizeDim(hwDim(400.0), hwDim(50.0))
	gs.GridTemplateColumns = []gtc{Length[gtc](100.0), Minmax[gtc](Length[MinTrackSizingFunction](0.0), Fr[MaxTrackSizingFunction](1.0))}
	gs.GridTemplateRows = []gtc{Length[gtc](50.0)}
	grid := hwUnwrap(taffy.NewWithChildren(gs, []NodeID{leaf}))

	hwOK(taffy.ComputeLayoutWithMeasure(grid, SizeMaxContent[AvailableSpace](), testMeasureFunction))

	hwAssertEq(t, hwLayout(taffy, leaf).Size.Width, 300.0)
	hwAssertEq(t, taffy.GetNodeContextMut(leaf).Count, 1)
}
