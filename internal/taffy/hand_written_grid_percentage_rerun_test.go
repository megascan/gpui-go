package taffy

import "testing"

func gridPercentageRerunLayoutFitContentPercentColumn(availableWidth AvailableSpace) [2]float32 {
	tree := newTestTree()
	item := hwUnwrap(tree.NewLeafWithContext(defaultStyle(), testNodeContextAhemText("HH​HH​HH", WritingModeHorizontal)))
	rs := defaultStyle()
	rs.Display = DisplayGrid
	rs.GridTemplateColumns = []gtc{FitContent[gtc](Percent[LengthPercentage](0.5))}
	rs.GridTemplateRows = []gtc{Length[gtc](40.0)}
	root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{item}))
	hwOK(tree.ComputeLayoutWithMeasure(root, Size[AvailableSpace]{Width: availableWidth, Height: AvailableSpaceMaxContent}, testMeasureFunction))
	return [2]float32{hwLayout(tree, root).Size.Width, hwLayout(tree, item).Size.Width}
}

func TestHandWrittenGridPercentageRerun_FitContentPercentageIndefiniteContainerMaxContentAvailableSpace(t *testing.T) {
	hwAssertEq(t, gridPercentageRerunLayoutFitContentPercentColumn(AvailableSpaceMaxContent), [2]float32{60.0, 30.0})
}

func TestHandWrittenGridPercentageRerun_FitContentPercentageIndefiniteContainerDefiniteAvailableSpace(t *testing.T) {
	hwAssertEq(t, gridPercentageRerunLayoutFitContentPercentColumn(AvailableSpaceDefinite(200.0)), [2]float32{60.0, 30.0})
}

func gridPercentageRerunLayoutPercentRowShrinkwrap(availableHeight AvailableSpace) [4]float32 {
	tree := NewTaffyTree[struct{}]()
	item := func(height float32) NodeID {
		s := defaultStyle()
		s.Size = hwSizeDim(hwDim(20.0), hwDim(height))
		return hwUnwrap(tree.NewLeaf(s))
	}
	a := item(20.0)
	b := item(60.0)
	c := item(20.0)
	rs := defaultStyle()
	rs.Display = DisplayGrid
	rs.GridTemplateColumns = []gtc{Auto[gtc]()}
	rs.GridTemplateRows = []gtc{Auto[gtc](), Percent[gtc](0.2), Auto[gtc]()}
	root := hwUnwrap(tree.NewWithChildren(rs, []NodeID{a, b, c}))
	hwOK(tree.ComputeLayout(root, Size[AvailableSpace]{Width: AvailableSpaceDefinite(200.0), Height: availableHeight}))
	total := hwLayout(tree, root).Size.Height
	y := func(node NodeID) float32 { return hwLayout(tree, node).Location.Y }
	return [4]float32{total, y(b), y(c) - y(b), total - y(c)}
}

func TestHandWrittenGridPercentageRerun_PercentageRowsIndefiniteContainerMaxContentAvailableSpace(t *testing.T) {
	hwAssertEq(t, gridPercentageRerunLayoutPercentRowShrinkwrap(AvailableSpaceMaxContent), [4]float32{100.0, 40.0, 20.0, 40.0})
}

func TestHandWrittenGridPercentageRerun_PercentageRowsIndefiniteContainerDefiniteAvailableSpace(t *testing.T) {
	hwAssertEq(t, gridPercentageRerunLayoutPercentRowShrinkwrap(AvailableSpaceDefinite(500.0)), [4]float32{100.0, 40.0, 20.0, 40.0})
}
