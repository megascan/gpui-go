package taffy

import "testing"

func safeAlignmentLeaf(w, h float32) Style[defaultCheapStr] {
	s := defaultStyle()
	s.Size = hwSizeDim(hwDim(w), hwDim(h))
	return s
}

func safeAlignmentTracks(values ...float32) []GridTemplateComponent[defaultCheapStr] {
	out := make([]GridTemplateComponent[defaultCheapStr], 0, len(values))
	for _, v := range values {
		out = append(out, GridTemplateComponentSingle[defaultCheapStr](Length[TrackSizingFunction](float64(v))))
	}
	return out
}

func safeAlignmentGridRoot() Style[defaultCheapStr] {
	s := safeAlignmentLeaf(100.0, 100.0)
	s.Display = DisplayGrid
	s.GridTemplateRows = safeAlignmentTracks(100.0)
	s.GridTemplateColumns = safeAlignmentTracks(100.0)
	return s
}

func safeAlignmentLayout(rootStyle Style[defaultCheapStr], children ...Style[defaultCheapStr]) (*TaffyTree[TestNodeContext], []NodeID) {
	taffy := newTestTree()
	ids := make([]NodeID, 0, len(children))
	for _, c := range children {
		ids = append(ids, hwUnwrap(taffy.NewLeaf(c)))
	}
	root := hwUnwrap(taffy.NewWithChildren(rootStyle, ids))
	hwOK(taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))
	return taffy, ids
}

func TestHandWrittenSafeAlignment_GridSafeAlignSelfFallsBackToStartOnOverflow(t *testing.T) {
	cs := safeAlignmentLeaf(50.0, 150.0)
	cs.AlignSelf = some(AlignItemsSafeEnd)
	taffy, ids := safeAlignmentLayout(safeAlignmentGridRoot(), cs)
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.Y, 0.0)
}

func TestHandWrittenSafeAlignment_GridSafeAlignSelfBehavesAsEndWhenNoOverflow(t *testing.T) {
	cs := safeAlignmentLeaf(50.0, 40.0)
	cs.AlignSelf = some(AlignItemsSafeEnd)
	taffy, ids := safeAlignmentLayout(safeAlignmentGridRoot(), cs)
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.Y, 60.0)
}

func TestHandWrittenSafeAlignment_GridUnsafeAlignSelfKeepsOverflowingPosition(t *testing.T) {
	cs := safeAlignmentLeaf(50.0, 150.0)
	cs.AlignSelf = some(AlignItemsEnd)
	taffy, ids := safeAlignmentLayout(safeAlignmentGridRoot(), cs)
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.Y, -50.0)
}

func TestHandWrittenSafeAlignment_GridSafeJustifySelfFallsBackToStartOnOverflow(t *testing.T) {
	cs := safeAlignmentLeaf(150.0, 50.0)
	cs.JustifySelf = some(AlignItemsSafeEnd)
	taffy, ids := safeAlignmentLayout(safeAlignmentGridRoot(), cs)
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.X, 0.0)
}

func TestHandWrittenSafeAlignment_GridSafeJustifySelfRtlFallsBackToRtlStartEdge(t *testing.T) {
	cs := safeAlignmentLeaf(150.0, 50.0)
	cs.JustifySelf = some(AlignItemsSafeEnd)
	rs := safeAlignmentGridRoot()
	rs.Direction = DirectionRtl
	taffy, ids := safeAlignmentLayout(rs, cs)
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.X, -50.0)
}

func TestHandWrittenSafeAlignment_GridSafeAlignContentOverflowFallsBackToStart(t *testing.T) {
	rs := safeAlignmentLeaf(100.0, 100.0)
	rs.Display = DisplayGrid
	rs.GridTemplateRows = safeAlignmentTracks(80.0, 80.0)
	rs.GridTemplateColumns = safeAlignmentTracks(40.0)
	rs.AlignContent = AlignContentSafeEnd
	taffy, ids := safeAlignmentLayout(rs, safeAlignmentLeaf(40.0, 80.0), safeAlignmentLeaf(40.0, 80.0))
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.Y, 0.0)
}

func TestHandWrittenSafeAlignment_FlexSafeAlignSelfFallsBackToStartOnOverflow(t *testing.T) {
	cs := safeAlignmentLeaf(50.0, 150.0)
	cs.AlignSelf = some(AlignItemsSafeEnd)
	rs := safeAlignmentLeaf(100.0, 100.0)
	rs.Display = DisplayFlex
	taffy, ids := safeAlignmentLayout(rs, cs)
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.Y, 0.0)
}

func TestHandWrittenSafeAlignment_FlexUnsafeAlignSelfKeepsOverflowingPosition(t *testing.T) {
	cs := safeAlignmentLeaf(50.0, 150.0)
	cs.AlignSelf = some(AlignItemsEnd)
	rs := safeAlignmentLeaf(100.0, 100.0)
	rs.Display = DisplayFlex
	taffy, ids := safeAlignmentLayout(rs, cs)
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.Y, -50.0)
}

func TestHandWrittenSafeAlignment_FlexSafeJustifyContentFallsBackToStartOnOverflow(t *testing.T) {
	rs := safeAlignmentLeaf(100.0, 100.0)
	rs.Display = DisplayFlex
	rs.JustifyContent = AlignContentSafeEnd
	taffy, ids := safeAlignmentLayout(rs, safeAlignmentLeaf(80.0, 50.0), safeAlignmentLeaf(80.0, 50.0))
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.X, 0.0)
}

func TestHandWrittenSafeAlignment_FlexSafeAlignContentFallsBackToStartOnMultilineOverflow(t *testing.T) {
	rs := safeAlignmentLeaf(100.0, 100.0)
	rs.Display = DisplayFlex
	rs.FlexWrap = FlexWrapWrap
	rs.AlignContent = AlignContentSafeEnd
	taffy, ids := safeAlignmentLayout(rs, safeAlignmentLeaf(60.0, 80.0), safeAlignmentLeaf(60.0, 80.0))
	hwAssertEq(t, hwLayout(taffy, ids[0]).Location.Y, 0.0)
}
