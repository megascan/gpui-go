package taffy

import (
	"math"
	"testing"
	"unsafe"
)

func calc4242(unsafe.Pointer, float32) float32 { return 42.42 }

type gtc = GridTemplateComponent[string]

func tLen(v float64) TrackSizingFunction { return Length[TrackSizingFunction](v) }

func autoFillRepeat(tracks ...TrackSizingFunction) gtc {
	return Repeat[string](RepetitionCountAutoFill, tracks)
}

func gridStyle() Style[string] {
	s := StyleDefault[string]()
	s.Display = DisplayGrid
	return s
}

func explicitSizes(t *testing.T, style *Style[string], inner Size[Option[float32]], strategy autoRepeatStrategy) (uint16, uint16, uint16, uint16) {
	t.Helper()
	autoColReps, colCount := computeExplicitGridSizeInAxis[string](style.Ref(), inner.GetAbs(AbsoluteAxisHorizontal), strategy, calc4242, AbsoluteAxisHorizontal)
	autoRowReps, rowCount := computeExplicitGridSizeInAxis[string](style.Ref(), inner.GetAbs(AbsoluteAxisVertical), strategy, calc4242, AbsoluteAxisVertical)
	return colCount, rowCount, autoColReps, autoRowReps
}

func preferredSize(style *Style[string]) Size[Option[float32]] {
	return style.Size.Map(func(d Dimension) Option[float32] { return d.IntoOption() })
}

func assertExplicit(t *testing.T, got [4]uint16, want [4]uint16) {
	t.Helper()
	if got != want {
		t.Fatalf("(col_count, row_count, auto_col_reps, auto_row_reps) = %v, want %v", got, want)
	}
}

func sz(w, h float64) Size[Dimension] {
	return Size[Dimension]{Width: Length[Dimension](w), Height: Length[Dimension](h)}
}

func TestComputeGridExplicitGrid_ExplicitGridSizingNoRepeats(t *testing.T) {
	style := parentTestNode{600, 600, 2, 4}.intoGrid()
	c, r, cr, rr := explicitSizes(t, &style, preferredSize(&style), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{2, 4, 0, 0})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillExactFit(t *testing.T) {
	style := gridStyle()
	style.Size = sz(120, 80)
	style.GridTemplateColumns = []gtc{autoFillRepeat(tLen(40))}
	style.GridTemplateRows = []gtc{autoFillRepeat(tLen(20))}
	c, r, cr, rr := explicitSizes(t, &style, preferredSize(&style), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{3, 4, 3, 4})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillNonExactFit(t *testing.T) {
	style := gridStyle()
	style.Size = sz(140, 90)
	style.GridTemplateColumns = []gtc{autoFillRepeat(tLen(40))}
	style.GridTemplateRows = []gtc{autoFillRepeat(tLen(20))}
	c, r, cr, rr := explicitSizes(t, &style, preferredSize(&style), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{3, 4, 3, 4})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillMinSizeExactFit(t *testing.T) {
	style := gridStyle()
	style.MinSize = Size[LengthPercentageAuto]{Width: Length[LengthPercentageAuto](120), Height: Length[LengthPercentageAuto](80)}
	style.GridTemplateColumns = []gtc{autoFillRepeat(tLen(40))}
	style.GridTemplateRows = []gtc{autoFillRepeat(tLen(20))}
	inner := Size[Option[float32]]{Width: some[float32](120), Height: some[float32](80)}
	c, r, cr, rr := explicitSizes(t, &style, inner, autoRepeatStrategyMinRepetitionsThatDoOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{3, 4, 3, 4})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillMinSizeNonExactFit(t *testing.T) {
	style := gridStyle()
	style.MinSize = Size[LengthPercentageAuto]{Width: Length[LengthPercentageAuto](140), Height: Length[LengthPercentageAuto](90)}
	style.GridTemplateColumns = []gtc{autoFillRepeat(tLen(40))}
	style.GridTemplateRows = []gtc{autoFillRepeat(tLen(20))}
	inner := Size[Option[float32]]{Width: some[float32](140), Height: some[float32](90)}
	c, r, cr, rr := explicitSizes(t, &style, inner, autoRepeatStrategyMinRepetitionsThatDoOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{4, 5, 4, 5})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillMultipleRepeatedTracks(t *testing.T) {
	style := gridStyle()
	style.Size = sz(140, 100)
	style.GridTemplateColumns = []gtc{autoFillRepeat(tLen(40), tLen(20))}
	style.GridTemplateRows = []gtc{autoFillRepeat(tLen(20), tLen(10))}
	c, r, cr, rr := explicitSizes(t, &style, preferredSize(&style), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{4, 6, 2, 3})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillGap(t *testing.T) {
	style := gridStyle()
	style.Size = sz(140, 100)
	style.GridTemplateColumns = []gtc{autoFillRepeat(tLen(40))}
	style.GridTemplateRows = []gtc{autoFillRepeat(tLen(20))}
	style.Gap = Length[Size[LengthPercentage]](20)
	c, r, cr, rr := explicitSizes(t, &style, preferredSize(&style), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{2, 3, 2, 3})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingNoDefinedSize(t *testing.T) {
	style := gridStyle()
	style.GridTemplateColumns = []gtc{autoFillRepeat(tLen(40), Percent[TrackSizingFunction](0.5), tLen(20))}
	style.GridTemplateRows = []gtc{autoFillRepeat(tLen(20))}
	style.Gap = Length[Size[LengthPercentage]](20)
	c, r, cr, rr := explicitSizes(t, &style, preferredSize(&style), autoRepeatStrategyMinRepetitionsThatDoOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{3, 1, 1, 1})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingMixRepeatedAndNonRepeated(t *testing.T) {
	style := gridStyle()
	style.Size = sz(140, 100)
	style.GridTemplateColumns = []gtc{Length[gtc](20), autoFillRepeat(tLen(40))}
	style.GridTemplateRows = []gtc{Length[gtc](40), autoFillRepeat(tLen(20))}
	style.Gap = Length[Size[LengthPercentage]](20)
	c, r, cr, rr := explicitSizes(t, &style, preferredSize(&style), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{3, 2, 2, 1})
}

func TestComputeGridExplicitGrid_ExplicitGridSizingMixWithPadding(t *testing.T) {
	style := gridStyle()
	style.Size = sz(120, 120)
	style.Padding = Rect[LengthPercentage]{Left: Length[LengthPercentage](10), Right: Length[LengthPercentage](10), Top: Length[LengthPercentage](20), Bottom: Length[LengthPercentage](20)}
	style.GridTemplateColumns = []gtc{autoFillRepeat(tLen(20))}
	style.GridTemplateRows = []gtc{autoFillRepeat(tLen(20))}
	inner := Size[Option[float32]]{Width: some[float32](100), Height: some[float32](80)}
	c, r, cr, rr := explicitSizes(t, &style, inner, autoRepeatStrategyMaxRepetitionsThatDoNotOverflow)
	assertExplicit(t, [4]uint16{c, r, cr, rr}, [4]uint16{5, 4, 5, 4})
}

func TestComputeGridExplicitGrid_AutoRepeatClampingPreservesTemplateSourceOrder(t *testing.T) {
	autoRepeatFirst := gridStyle()
	autoRepeatFirst.GridTemplateColumns = []gtc{autoFillRepeat(tLen(2)), Repeat[string](RepetitionCountCount(10_000), []TrackSizingFunction{tLen(37)})}
	repetitions, trackCount := computeExplicitGridSizeInAxis[string](autoRepeatFirst.Ref(), none[float32](), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow, calc4242, AbsoluteAxisHorizontal)
	assertEq(t, [2]uint16{repetitions, trackCount}, [2]uint16{1, 10_000})
	var tracks []gridTrack
	initializeGridTracks[string](&tracks, trackCountsFromRaw(0, trackCount, 0), autoRepeatFirst.Ref(), AbsoluteAxisHorizontal, repetitions, func(int) bool { return false })
	assertEq(t, tracks[1].minTrackSizingFunction, Length[MinTrackSizingFunction](2))
	assertEq(t, tracks[3].minTrackSizingFunction, Length[MinTrackSizingFunction](37))

	autoRepeatLast := gridStyle()
	autoRepeatLast.GridTemplateColumns = []gtc{Repeat[string](RepetitionCountCount(10_000), []TrackSizingFunction{tLen(37)}), autoFillRepeat(tLen(2))}
	repetitions, trackCount = computeExplicitGridSizeInAxis[string](autoRepeatLast.Ref(), none[float32](), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow, calc4242, AbsoluteAxisHorizontal)
	assertEq(t, [2]uint16{repetitions, trackCount}, [2]uint16{0, 10_000})
}

func TestComputeGridExplicitGrid_ExplicitTrackCountAccumulationSaturates(t *testing.T) {
	repeatedTracks := make([]TrackSizingFunction, 32_769)
	for i := range repeatedTracks {
		repeatedTracks[i] = tLen(1)
	}
	style := gridStyle()
	style.GridTemplateColumns = []gtc{
		Repeat[string](RepetitionCountCount(math.MaxUint16), repeatedTracks),
		Repeat[string](RepetitionCountCount(math.MaxUint16), repeatedTracks),
		autoFillRepeat(tLen(1)),
	}
	repetitions, trackCount := computeExplicitGridSizeInAxis[string](style.Ref(), none[float32](), autoRepeatStrategyMaxRepetitionsThatDoNotOverflow, calc4242, AbsoluteAxisHorizontal)
	assertEq(t, [2]uint16{repetitions, trackCount}, [2]uint16{0, 10_000})
}

func TestComputeGridExplicitGrid_InitializeGridTracks(t *testing.T) {
	minpx0 := Length[MinTrackSizingFunction](0)
	minpx20 := Length[MinTrackSizingFunction](20)
	minpx100 := Length[MinTrackSizingFunction](100)
	maxpx0 := Length[MaxTrackSizingFunction](0)
	maxpx20 := Length[MaxTrackSizingFunction](20)
	maxpx100 := Length[MaxTrackSizingFunction](100)
	minAuto := Auto[MinTrackSizingFunction]()
	maxAuto := Auto[MaxTrackSizingFunction]()

	style := gridStyle()
	style.Gap = Length[Size[LengthPercentage]](20)
	style.GridTemplateColumns = []gtc{Length[gtc](100), Minmax[gtc](Length[MinTrackSizingFunction](100), Fr[MaxTrackSizingFunction](2)), Fr[gtc](1)}
	style.GridAutoColumns = []TrackSizingFunction{Auto[TrackSizingFunction](), tLen(100)}
	counts := trackCounts{negativeImplicit: 3, explicit: uint16(len(style.GridTemplateColumns)), positiveImplicit: 3}

	var tracks []gridTrack
	initializeGridTracks[string](&tracks, counts, style.Ref(), AbsoluteAxisHorizontal, 0, func(int) bool { return false })

	type exp struct {
		kind gridTrackKind
		min  MinTrackSizingFunction
		max  MaxTrackSizingFunction
	}
	track, gutter := gridTrackKindTrack, gridTrackKindGutter
	expected := []exp{
		{gutter, minpx0, maxpx0},
		{track, minpx100, maxpx100},
		{gutter, minpx20, maxpx20},
		{track, minAuto, maxAuto},
		{gutter, minpx20, maxpx20},
		{track, minpx100, maxpx100},
		{gutter, minpx20, maxpx20},
		{track, minpx100, maxpx100},
		{gutter, minpx20, maxpx20},
		{track, minpx100, Fr[MaxTrackSizingFunction](2)},
		{gutter, minpx20, maxpx20},
		{track, minAuto, Fr[MaxTrackSizingFunction](1)},
		{gutter, minpx20, maxpx20},
		{track, minAuto, maxAuto},
		{gutter, minpx20, maxpx20},
		{track, minpx100, maxpx100},
		{gutter, minpx20, maxpx20},
		{track, minAuto, maxAuto},
		{gutter, minpx0, maxpx0},
	}
	if len(tracks) != len(expected) {
		t.Fatalf("Number of tracks doesn't match: got %d, want %d", len(tracks), len(expected))
	}
	for idx, e := range expected {
		actual := tracks[idx]
		if actual.kind != e.kind || actual.minTrackSizingFunction != e.min || actual.maxTrackSizingFunction != e.max {
			t.Fatalf("Track %d (0-based index): got (%v, %v, %v), want %v", idx, actual.kind, actual.minTrackSizingFunction, actual.maxTrackSizingFunction, e)
		}
	}
}
