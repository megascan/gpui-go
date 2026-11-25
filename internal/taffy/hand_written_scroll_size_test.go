package taffy

import (
	"fmt"
	"testing"
)

const (
	scrollSizeContent   float32 = 1000.0
	scrollSizeContainer float32 = 200.0
	scrollSizeBorder    float32 = 20.0
	scrollSizePadding   float32 = 20.0
)

var scrollSizeDisplays = []Display{DisplayBlock, DisplayFlex, DisplayGrid}

func scrollSizeEdgePairs(v float32) [][2]float32 {
	return [][2]float32{{v, 0.0}, {0.0, v}, {v, v}, {0.0, 0.0}}
}

func scrollSizeScrollerWithPadding(display Display, border, padding Rect[LengthPercentage]) Layout {
	tree := newTestTree()
	cs := defaultStyle()
	cs.Size = hwSizeDim(hwDim(100.0), hwDim(scrollSizeContent))
	child := hwUnwrap(tree.NewLeaf(cs))
	ns := defaultStyle()
	ns.Display = display
	ns.BoxSizing = BoxSizingBorderBox
	ns.Size = hwSizeDim(hwDim(300.0), hwDim(scrollSizeContainer))
	ns.Border = border
	ns.Padding = padding
	ns.Overflow = Point[Overflow]{X: OverflowScroll, Y: OverflowScroll}
	node := hwUnwrap(tree.NewWithChildren(ns, []NodeID{child}))

	hwOK(tree.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
	return hwLayout(tree, node)
}

func scrollSizeScroller(display Display, border Rect[LengthPercentage]) Layout {
	return scrollSizeScrollerWithPadding(display, border, scrollSizeEdge(0.0, 0.0))
}

func scrollSizeEdge(top, bottom float32) Rect[LengthPercentage] {
	return Rect[LengthPercentage]{
		Left:   LengthPercentageLength(0.0),
		Right:  LengthPercentageLength(0.0),
		Top:    LengthPercentageLength(top),
		Bottom: LengthPercentageLength(bottom),
	}
}

func TestHandWrittenScrollSize_ScrollHeightDoesNotDependOnWhichEdgeTheBorderIsOn(t *testing.T) {
	for _, display := range scrollSizeDisplays {
		top := scrollSizeScroller(display, scrollSizeEdge(scrollSizeBorder, 0.0))
		bottom := scrollSizeScroller(display, scrollSizeEdge(0.0, scrollSizeBorder))

		hwAssertEqMsg(t, top.ScrollHeight(), bottom.ScrollHeight(), fmt.Sprintf("%v: border-top gave %v and border-bottom gave %v", display, top.ScrollHeight(), bottom.ScrollHeight()))
	}
}

func TestHandWrittenScrollSize_ScrollHeightIsTheContentBeyondThePaddingBox(t *testing.T) {
	for _, display := range scrollSizeDisplays {
		for _, e := range scrollSizeEdgePairs(scrollSizeBorder) {
			top, bottom := e[0], e[1]
			layout := scrollSizeScroller(display, scrollSizeEdge(top, bottom))
			paddingBox := scrollSizeContainer - top - bottom

			hwAssertEqMsg(t, layout.ScrollHeight(), scrollSizeContent-paddingBox, fmt.Sprintf("%v with border %v/%v", display, top, bottom))
		}
	}
}

func TestHandWrittenScrollSize_ScrollableOverflowExcludesTheContainersOwnBorder(t *testing.T) {
	for _, display := range scrollSizeDisplays {
		for _, e := range scrollSizeEdgePairs(scrollSizeBorder) {
			top, bottom := e[0], e[1]
			layout := scrollSizeScroller(display, scrollSizeEdge(top, bottom))

			hwAssertEqMsg(t, layout.ScrollableOverflowRect.Bottom, scrollSizeContent, fmt.Sprintf("%v with border %v/%v", display, top, bottom))
		}
	}
}

func TestHandWrittenScrollSize_ScrollableOverflowIncludesTheContainersOwnPadding(t *testing.T) {
	for _, display := range scrollSizeDisplays {
		for _, e := range scrollSizeEdgePairs(scrollSizePadding) {
			top, bottom := e[0], e[1]
			layout := scrollSizeScrollerWithPadding(display, scrollSizeEdge(0.0, 0.0), scrollSizeEdge(top, bottom))

			hwAssertEqMsg(t, layout.ScrollableOverflowRect.Bottom, scrollSizeContent+top+bottom, fmt.Sprintf("%v with padding %v/%v", display, top, bottom))
		}
	}
}

func TestHandWrittenScrollSize_ScrollableOverflowExcludesTheOwnPaddingOfANonScrollContainer(t *testing.T) {
	for _, display := range scrollSizeDisplays {
		for _, e := range scrollSizeEdgePairs(scrollSizePadding) {
			top, bottom := e[0], e[1]
			tree := newTestTree()
			cs := defaultStyle()
			cs.Size = hwSizeDim(hwDim(100.0), hwDim(scrollSizeContent))
			child := hwUnwrap(tree.NewLeaf(cs))
			ns := defaultStyle()
			ns.Display = display
			ns.BoxSizing = BoxSizingBorderBox
			ns.Size = hwSizeDim(hwDim(300.0), hwDim(scrollSizeContainer))
			ns.Padding = scrollSizeEdge(top, bottom)
			node := hwUnwrap(tree.NewWithChildren(ns, []NodeID{child}))

			hwOK(tree.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
			layout := hwLayout(tree, node)

			hwAssertEqMsg(t, layout.ScrollableOverflowRect.Bottom, top+scrollSizeContent, fmt.Sprintf("%v with padding %v/%v", display, top, bottom))
		}
	}
}

func scrollSizeMeasuredLeaf(overflow Overflow, padding Rect[LengthPercentage]) Layout {
	tree := newTestTree()
	ns := defaultStyle()
	ns.BoxSizing = BoxSizingBorderBox
	ns.Size = hwSizeDim(hwDim(300.0), hwDim(scrollSizeContainer))
	ns.Padding = padding
	ns.Overflow = Point[Overflow]{X: overflow, Y: overflow}
	node := hwUnwrap(tree.NewLeafWithContext(ns, testNodeContextFixed(100.0, scrollSizeContent)))

	hwOK(tree.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), testMeasureFunction))
	return hwLayout(tree, node)
}

func TestHandWrittenScrollSize_LeafContentSizeIncludesTheContainersOwnPadding(t *testing.T) {
	for _, e := range scrollSizeEdgePairs(scrollSizePadding) {
		top, bottom := e[0], e[1]
		layout := scrollSizeMeasuredLeaf(OverflowScroll, scrollSizeEdge(top, bottom))

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Bottom, top+scrollSizeContent+bottom, fmt.Sprintf("Leaf with padding %v/%v", top, bottom))
	}
}

func TestHandWrittenScrollSize_LeafContentSizeExcludesTheOwnPaddingOfANonScrollContainer(t *testing.T) {
	for _, e := range scrollSizeEdgePairs(scrollSizePadding) {
		top, bottom := e[0], e[1]
		layout := scrollSizeMeasuredLeaf(OverflowVisible, scrollSizeEdge(top, bottom))

		hwAssertEqMsg(t, layout.ScrollableOverflowRect.Bottom, top+scrollSizeContent, fmt.Sprintf("Leaf with padding %v/%v", top, bottom))
	}
}

func TestHandWrittenScrollSize_ScrollHeightAccountsForTheContainersOwnPadding(t *testing.T) {
	for _, display := range scrollSizeDisplays {
		for _, e := range scrollSizeEdgePairs(scrollSizePadding) {
			top, bottom := e[0], e[1]
			layout := scrollSizeScrollerWithPadding(display, scrollSizeEdge(0.0, 0.0), scrollSizeEdge(top, bottom))

			expected := (top + scrollSizeContent + bottom) - scrollSizeContainer
			hwAssertEqMsg(t, layout.ScrollHeight(), expected, fmt.Sprintf("%v with padding %v/%v", display, top, bottom))
		}
	}
}

func TestHandWrittenScrollSize_GridScrollableOverflowIncludesItemPositionsInOverflowingTracks(t *testing.T) {
	tree := newTestTree()
	child := func() Style[defaultCheapStr] {
		s := defaultStyle()
		s.Size = hwSizeDim(hwDim(100.0), hwDim(500.0))
		return s
	}
	c1 := hwUnwrap(tree.NewLeaf(child()))
	c2 := hwUnwrap(tree.NewLeaf(child()))
	ns := defaultStyle()
	ns.Display = DisplayGrid
	ns.Size = hwSizeDim(hwDim(300.0), hwDim(scrollSizeContainer))
	ns.Overflow = Point[Overflow]{X: OverflowScroll, Y: OverflowScroll}
	node := hwUnwrap(tree.NewWithChildren(ns, []NodeID{c1, c2}))

	hwOK(tree.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
	layout := hwLayout(tree, node)

	hwAssertEq(t, layout.ScrollableOverflowRect.Bottom, 1000.0)
	hwAssertEq(t, layout.ScrollHeight(), 1000.0-scrollSizeContainer)
}

func TestHandWrittenScrollSize_ScrollHeightAgreesWithBordersAndPaddingCombined(t *testing.T) {
	for _, display := range scrollSizeDisplays {
		layout := scrollSizeScrollerWithPadding(display, scrollSizeEdge(scrollSizeBorder, scrollSizeBorder), scrollSizeEdge(scrollSizePadding, scrollSizePadding))

		paddingBox := scrollSizeContainer - scrollSizeBorder - scrollSizeBorder
		expected := (scrollSizePadding + scrollSizeContent + scrollSizePadding) - paddingBox
		hwAssertEqMsg(t, layout.ScrollHeight(), expected, display.String())
	}
}
