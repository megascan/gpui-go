package taffy

import "testing"

func oofTestSp(area Line[float32], keyword AxisStaticEdge, safety AlignmentSafety, fallback AxisStaticEdge) AxisStaticPosition {
	return AxisStaticPosition{Area: area, Align: AxisStaticAlign{Keyword: keyword, Safety: safety, Fallback: fallback}}
}

func oofTestResolveX(spX AxisStaticPosition, width float32, margin Rect[float32]) float32 {
	position := Point[AxisStaticPosition]{X: spX, Y: AxisStaticPositionFromEdge(0.0, AxisStaticEdgeStart)}
	return ResolveStaticOffset(position, Size[float32]{Width: width, Height: 10.0}, margin).X
}

var (
	oofTestArea     = Line[float32]{Start: 10.0, End: 110.0}
	oofTestNoMargin = Rect[float32]{Left: 0.0, Right: 0.0, Top: 0.0, Bottom: 0.0}
	oofTestMargin   = Rect[float32]{Left: 5.0, Right: 15.0, Top: 0.0, Bottom: 0.0}
)

func oofAssertEq(t *testing.T, left, right float32) {
	t.Helper()
	if left != right {
		t.Fatalf("assertion `left == right` failed\n  left: %v\n right: %v", left, right)
	}
}

func TestComputeOof_StartAlignment(t *testing.T) {
	spX := oofTestSp(oofTestArea, AxisStaticEdgeStart, AlignmentSafetyUnsafe, AxisStaticEdgeStart)
	oofAssertEq(t, oofTestResolveX(spX, 20.0, oofTestNoMargin), 10.0)
	oofAssertEq(t, oofTestResolveX(spX, 20.0, oofTestMargin), 15.0)
}

func TestComputeOof_EndAlignment(t *testing.T) {
	spX := oofTestSp(oofTestArea, AxisStaticEdgeEnd, AlignmentSafetyUnsafe, AxisStaticEdgeEnd)
	oofAssertEq(t, oofTestResolveX(spX, 20.0, oofTestNoMargin), 90.0)
	oofAssertEq(t, oofTestResolveX(spX, 20.0, oofTestMargin), 75.0)
}

func TestComputeOof_CenterAlignment(t *testing.T) {
	spX := oofTestSp(oofTestArea, AxisStaticEdgeCenter, AlignmentSafetyUnsafe, AxisStaticEdgeCenter)
	oofAssertEq(t, oofTestResolveX(spX, 20.0, oofTestNoMargin), 50.0)
	oofAssertEq(t, oofTestResolveX(spX, 20.0, oofTestMargin), 45.0)
}

func TestComputeOof_SafeAlignmentFallsBackOnlyOnOverflow(t *testing.T) {
	spX := oofTestSp(oofTestArea, AxisStaticEdgeEnd, AlignmentSafetySafe, AxisStaticEdgeStart)
	oofAssertEq(t, oofTestResolveX(spX, 20.0, oofTestNoMargin), 90.0)
	oofAssertEq(t, oofTestResolveX(spX, 100.0, oofTestNoMargin), 10.0)
	oofAssertEq(t, oofTestResolveX(spX, 120.0, oofTestNoMargin), 10.0)
	oofAssertEq(t, oofTestResolveX(spX, 90.0, oofTestMargin), 15.0)
}

func TestComputeOof_UnsafeAlignmentNeverFallsBack(t *testing.T) {
	spX := oofTestSp(oofTestArea, AxisStaticEdgeEnd, AlignmentSafetyUnsafe, AxisStaticEdgeStart)
	oofAssertEq(t, oofTestResolveX(spX, 120.0, oofTestNoMargin), -10.0)
}

func TestComputeOof_DegenerateArea(t *testing.T) {
	anchor := AxisStaticPositionFromEdge(40.0, AxisStaticEdgeStart)
	oofAssertEq(t, oofTestResolveX(anchor, 20.0, oofTestNoMargin), 40.0)
	anchorEnd := AxisStaticPositionFromEdge(40.0, AxisStaticEdgeEnd)
	oofAssertEq(t, oofTestResolveX(anchorEnd, 20.0, oofTestNoMargin), 20.0)
}
