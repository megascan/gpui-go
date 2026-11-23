package taffy

import "testing"

func TestStyleDimension_ExpandTests_LengthPercentageRoundTrips(t *testing.T) {
	cases := []LengthPercentage{LengthPercentageLength(12.0), LengthPercentagePercent(0.5), LengthPercentageZero}
	for _, value := range cases {
		if got := LengthPercentageFromExpandedLengthPercentage(value.Expand()); got != value {
			t.Errorf("round trip of %v = %v", value, got)
		}
		if ExpandedLengthPercentageFromLengthPercentage(value) != value.Expand() {
			t.Errorf("From of %v differs from Expand", value)
		}
	}
	if got := LengthPercentageLength(3.0).Expand(); got != ExpandedLengthPercentageLength(3.0) {
		t.Errorf("got %v", got)
	}
	if got := LengthPercentagePercent(0.25).Expand(); got != ExpandedLengthPercentagePercent(0.25) {
		t.Errorf("got %v", got)
	}
}

func TestStyleDimension_ExpandTests_LengthPercentageAutoRoundTrips(t *testing.T) {
	cases := []LengthPercentageAuto{LengthPercentageAutoLength(12.0), LengthPercentageAutoPercent(0.5), LengthPercentageAutoAuto()}
	for _, value := range cases {
		if got := LengthPercentageAutoFromExpandedLengthPercentageAuto(value.Expand()); got != value {
			t.Errorf("round trip of %v = %v", value, got)
		}
	}
	if got := LengthPercentageAutoAuto().Expand(); got != ExpandedLengthPercentageAutoAuto {
		t.Errorf("got %v", got)
	}
}

func TestStyleDimension_ExpandTests_DimensionRoundTripsAllKeywords(t *testing.T) {
	cases := []Dimension{
		DimensionLength(12.0),
		DimensionPercent(0.5),
		DimensionAuto(),
		DimensionMinContent(),
		DimensionMaxContent(),
		DimensionFitContentPx(30.0),
		DimensionFitContentPercent(0.75),
		DimensionFitContent(),
		DimensionStretch(),
		DimensionContent(),
	}
	for _, value := range cases {
		if got := DimensionFromExpandedDimension(value.Expand()); got != value {
			t.Errorf("round trip of %v = %v", value, got)
		}
	}
	if got := DimensionFitContentPx(30.0).Expand(); got != ExpandedDimensionFitContentPx(30.0) {
		t.Errorf("got %v", got)
	}
	if got := DimensionContent().Expand(); got != ExpandedDimensionContent {
		t.Errorf("got %v", got)
	}
}

func TestStyleDimension_ResolveToOption(t *testing.T) {
	if got := LengthPercentageAutoLength(4).ResolveToOption(10, nil); got != some[float32](4) {
		t.Errorf("got %v", got)
	}
	if got := LengthPercentageAutoPercent(0.5).ResolveToOption(10, nil); got != some[float32](5) {
		t.Errorf("got %v", got)
	}
	if got := LengthPercentageAutoAuto().ResolveToOption(10, nil); got.IsSome() {
		t.Errorf("got %v", got)
	}
	if got := DimensionLength(7).IntoOption(); got != some[float32](7) {
		t.Errorf("got %v", got)
	}
	if got := DimensionPercent(0.7).IntoOption(); got.IsSome() {
		t.Errorf("got %v", got)
	}
	if !DimensionStretch().IsStretch() || !DimensionStretch().IsSizingKeyword() || DimensionAuto().IsSizingKeyword() {
		t.Error("keyword predicates")
	}
	r := RectDimensionFromLength(1, 2, 3, 4)
	if r.Left != DimensionLength(1) || r.Bottom != DimensionLength(4) {
		t.Errorf("got %v", r)
	}
	s := SizeDimensionFromPercent(0.1, 0.2)
	if s.Width != DimensionPercent(0.1) || s.Height != DimensionPercent(0.2) {
		t.Errorf("got %v", s)
	}
}
