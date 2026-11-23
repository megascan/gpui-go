package taffy

import "testing"

func TestTreeLayout_OofCandidatesAppendAndClone(t *testing.T) {
	var a, b OofCandidates
	b.Push(OofCandidate{Node: NewNodeID(1)})
	a.Append(&b)
	if len(a.AsSlice()) != 1 || b.IsEmpty() == false || b.inner != nil {
		t.Fatal("append into empty list must move the storage")
	}
	b.Push(OofCandidate{Node: NewNodeID(2)})
	a.Append(&b)
	if len(a.AsSlice()) != 2 || !b.IsEmpty() || b.inner == nil {
		t.Fatal("append into a list must leave other as an empty allocated list")
	}
	if b.Equal(OofCandidatesNone) {
		t.Fatal("an empty allocated list differs from None")
	}
	c := a.Clone()
	c.Translate(Point[float32]{X: 1, Y: 2})
	if a.AsSlice()[0].StaticPosition.X.Area.Start != 0 || c.AsSlice()[0].StaticPosition.Y.Area.End != 2 {
		t.Fatal("clone must not share storage")
	}
}

func TestTreeLayout_AxisStaticPositionFromAlignment(t *testing.T) {
	area := Line[float32]{Start: 0, End: 10}
	p := AxisStaticPositionFromAlignment(NewAlignItems(AlignItemsKeywordEnd, AlignmentSafetySafe), area, false)
	if p.Align != (AxisStaticAlign{Keyword: AxisStaticEdgeEnd, Safety: AlignmentSafetySafe, Fallback: AxisStaticEdgeStart}) {
		t.Fatalf("got %+v", p.Align)
	}
	p = AxisStaticPositionFromAlignment(AlignItemsStart, area, true)
	if p.Align.Keyword != AxisStaticEdgeEnd {
		t.Fatalf("rtl start must map to end, got %+v", p.Align)
	}
}

func TestTreeLayout_ScrollWidth(t *testing.T) {
	l := NewLayout()
	l.Size = Size[float32]{Width: 100, Height: 50}
	l.ScrollableOverflowRect.Right = 150
	l.ScrollbarSize.Width = 10
	if got := l.ScrollWidth(); got != 60 {
		t.Fatalf("ScrollWidth = %v", got)
	}
	if got := l.ScrollHeight(); got != 0 {
		t.Fatalf("ScrollHeight = %v", got)
	}
}

func TestTreeLayout_TransposeRoundTrip(t *testing.T) {
	in := LayoutInputHidden
	in.Axis = RequestedAxisHorizontal
	in.KnownDimensions = Size[Option[float32]]{Width: some[float32](3)}
	if got := in.Transpose().Transpose(); got != in {
		t.Fatalf("got %+v", got)
	}
}
