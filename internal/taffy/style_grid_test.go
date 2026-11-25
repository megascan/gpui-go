package taffy

import "testing"

func TestStyleGrid_ExpandTests_MaxTrackSizingFunctionRoundTrips(t *testing.T) {
	cases := []MaxTrackSizingFunction{
		MaxTrackSizingFunctionLength(12.0),
		MaxTrackSizingFunctionPercent(0.5),
		MaxTrackSizingFunctionAuto(),
		MaxTrackSizingFunctionMinContent(),
		MaxTrackSizingFunctionMaxContent(),
		MaxTrackSizingFunctionFitContentPx(30.0),
		MaxTrackSizingFunctionFitContentPercent(0.75),
		MaxTrackSizingFunctionFr(2.0),
	}
	for _, value := range cases {
		if got := MaxTrackSizingFunctionFromExpandedMaxTrackSizingFunction(value.Expand()); got != value {
			t.Errorf("round trip of %v gave %v", value, got)
		}
		if ExpandedMaxTrackSizingFunctionFromMaxTrackSizingFunction(value) != value.Expand() {
			t.Errorf("From differs from Expand for %v", value)
		}
	}
	if MaxTrackSizingFunctionFr(2.0).Expand() != ExpandedMaxTrackSizingFunctionFr(2.0) {
		t.Error("fr(2.0).expand() != Fr(2.0)")
	}
	if MaxTrackSizingFunctionFitContentPx(30.0).Expand() != ExpandedMaxTrackSizingFunctionFitContentPx(30.0) {
		t.Error("fit_content_px(30.0).expand() != FitContentPx(30.0)")
	}
}

func TestStyleGrid_ExpandTests_MinTrackSizingFunctionRoundTrips(t *testing.T) {
	cases := []MinTrackSizingFunction{
		MinTrackSizingFunctionLength(12.0),
		MinTrackSizingFunctionPercent(0.5),
		MinTrackSizingFunctionAuto(),
		MinTrackSizingFunctionMinContent(),
		MinTrackSizingFunctionMaxContent(),
	}
	for _, value := range cases {
		if got := MinTrackSizingFunctionFromExpandedMinTrackSizingFunction(value.Expand()); got != value {
			t.Errorf("round trip of %v gave %v", value, got)
		}
		if ExpandedMinTrackSizingFunctionFromMinTrackSizingFunction(value) != value.Expand() {
			t.Errorf("From differs from Expand for %v", value)
		}
	}
	if MinTrackSizingFunctionMaxContent().Expand() != ExpandedMinTrackSizingFunctionMaxContent {
		t.Error("max_content().expand() != MaxContent")
	}
}

func TestStyleGrid_PlacementResolution(t *testing.T) {
	oz := func(l int16) GenericGridPlacement[OriginZeroLine] { return GenericGridPlacementLine(OriginZeroLine(l)) }
	sp := GenericGridPlacementSpan[OriginZeroLine]
	auto := GenericGridPlacementAuto[OriginZeroLine]()
	type L = Line[GenericGridPlacement[OriginZeroLine]]
	if got := LineOriginZeroGridPlacementResolveDefiniteGridLines(L{Start: oz(3), End: oz(1)}); got != (Line[OriginZeroLine]{Start: 1, End: 3}) {
		t.Errorf("got %v", got)
	}
	if got := LineOriginZeroGridPlacementResolveDefiniteGridLines(L{Start: oz(2), End: oz(2)}); got != (Line[OriginZeroLine]{Start: 2, End: 3}) {
		t.Errorf("got %v", got)
	}
	if got := LineOriginZeroGridPlacementResolveDefiniteGridLines(L{Start: sp(2), End: oz(5)}); got != (Line[OriginZeroLine]{Start: 3, End: 5}) {
		t.Errorf("got %v", got)
	}
	if got := LineOriginZeroGridPlacementResolveIndefiniteGridTracks(L{Start: auto, End: sp(4)}, 1); got != (Line[OriginZeroLine]{Start: 1, End: 5}) {
		t.Errorf("got %v", got)
	}
	abs := LineOriginZeroGridPlacementResolveAbsolutelyPositionedGridTracks(L{Start: oz(1), End: auto})
	if abs.Start != some(OriginZeroLine(1)) || abs.End.IsSome() {
		t.Errorf("got %v", abs)
	}
	if LineGenericGridPlacementIndefiniteSpan(L{Start: sp(3), End: auto}) != 3 {
		t.Error("indefinite span")
	}
	if !LineOriginZeroGridPlacementIsDefinite(L{Start: auto, End: oz(0)}) {
		t.Error("is_definite")
	}
	func() {
		defer func() {
			if r := recover(); r != "indefinite_span should only be called on indefinite grid tracks" {
				t.Errorf("panic %v", r)
			}
		}()
		LineGenericGridPlacementIndefiniteSpan(L{Start: oz(1), End: oz(2)})
	}()
	named := Line[GridPlacement[string]]{Start: GridPlacementNamedLine("a", 0), End: GridPlacementAuto[string]()}
	if !LineGridPlacementIsDefinite(named) || LineGridPlacementIsDefinite(LineGridPlacementDefault[string]()) {
		t.Error("GridPlacement is_definite")
	}
	if got := GridPlacementSpan[string](0).IntoOriginZeroPlacementIgnoringNamed(3); got != sp(1) {
		t.Errorf("zero span clamps to 1, got %v", got)
	}
	if got := GridPlacementNamedSpan("a", 2).IntoOriginZeroPlacementIgnoringNamed(3); got != auto {
		t.Errorf("named span is auto, got %v", got)
	}
}

func TestStyleGrid_TrackConversions(t *testing.T) {
	if MaxTrackSizingFunctionFromDimension(DimensionFromLengthPercentage(LengthPercentageLength(4))) != MaxTrackSizingFunctionLength(4) {
		t.Error("max from dimension length")
	}
	if MinTrackSizingFunctionFromMaxTrackSizingFunction(MaxTrackSizingFunctionFr(1)) != MinTrackSizingFunctionAuto() {
		t.Error("min from fr")
	}
	if MinTrackSizingFunctionFromMaxTrackSizingFunction(MaxTrackSizingFunctionFitContentPx(5)) != MinTrackSizingFunctionAuto() {
		t.Error("min from fit-content")
	}
	if !TrackSizingFunctionHasFixedComponent(TrackSizingFunction{Min: MinTrackSizingFunctionAuto(), Max: MaxTrackSizingFunctionPercent(0.5)}) {
		t.Error("has_fixed_component")
	}
	if got := MaxTrackSizingFunctionPercent(0.5).DefiniteValue(some[float32](200), nil); got != some[float32](100) {
		t.Errorf("definite_value %v", got)
	}
	if got := MaxTrackSizingFunctionFitContentPx(7).DefiniteLimit(none[float32](), nil); got != some[float32](7) {
		t.Errorf("definite_limit %v", got)
	}
	if !MinTrackSizingFunctionPercent(0.1).BehavesAsAuto(none[float32]()) {
		t.Error("behaves_as_auto")
	}
	if GridAutoFlowColumnDense.PrimaryAxis() != AbsoluteAxisVertical || !GridAutoFlowRowDense.IsDense() || GridAutoFlowRow.IsDense() {
		t.Error("GridAutoFlow")
	}
}

func TestStyleGrid_StyleRefImplementsGridStyles(t *testing.T) {
	style := StyleDefault[string]()
	style.GridTemplateColumns = []GridTemplateComponent[string]{
		Length[GridTemplateComponent[string]](10),
		Repeat[string](RepetitionCountAutoFill, []TrackSizingFunction{Fr[TrackSizingFunction](1)}),
	}
	style.GridTemplateAreas = some(GridTemplateAreas[string]{Areas: []GridTemplateArea[string]{{Name: "a", RowEnd: 3, ColumnEnd: 2}}, RowCount: 4, ColumnCount: 5})
	ref := style.Ref()
	cols := GridContainerStyleDefaultGridTemplateTracks[string](ref, AbsoluteAxisHorizontal).Unwrap()
	if len(cols) != 2 || cols[0].IsAutoRepetition() || !cols[1].IsAutoRepetition() || cols[1].Repeat().TrackCount() != 1 {
		t.Errorf("columns %v", cols)
	}
	if ref.GridTemplateAreaRowCount() != 4 || GridContainerStyleDefaultGridTemplateAreaRowCount[string](ref) != 2 {
		t.Error("area row count")
	}
	if ref.GridAlignContent(AbstractAxisBlock) != AlignContentStretch {
		t.Error("normal resolves to stretch")
	}
}
