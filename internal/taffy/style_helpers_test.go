package taffy

import (
	"reflect"
	"testing"
)

var styleHelpersTestVec []TrackSizingFunction

func TestStyleHelpers_RepeatFnTests_RepeatU16(t *testing.T) {
	got := Repeat[string](uint16(123), styleHelpersTestVec)
	want := GridTemplateComponentRepeat(GridTemplateRepetition[string]{Count: RepetitionCountCount(123), Tracks: styleHelpersTestVec, LineNames: nil})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStyleHelpers_RepeatFnTests_RepeatAutoFitStr(t *testing.T) {
	got := Repeat[string]("auto-fit", styleHelpersTestVec)
	want := GridTemplateComponentRepeat(GridTemplateRepetition[string]{Count: RepetitionCountAutoFit, Tracks: styleHelpersTestVec, LineNames: nil})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStyleHelpers_RepeatFnTests_RepeatAutoFillStr(t *testing.T) {
	got := Repeat[string]("auto-fill", styleHelpersTestVec)
	want := GridTemplateComponentRepeat(GridTemplateRepetition[string]{Count: RepetitionCountAutoFill, Tracks: styleHelpersTestVec, LineNames: nil})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStyleHelpers_Constructors(t *testing.T) {
	if Zero[Size[Dimension]]() != (Size[Dimension]{Width: DimensionZero, Height: DimensionZero}) {
		t.Error("zero size")
	}
	if PointZero[float32]() != (Point[float32]{}) || SizeLength[Option[float32]](3) != NewSizeOptionF32(3, 3) {
		t.Error("f32 helpers")
	}
	if Auto[Rect[LengthPercentageAuto]]().Left != LengthPercentageAutoAuto() {
		t.Error("auto rect")
	}
	if Percent[Line[Dimension]](0.5).End != DimensionPercent(0.5) {
		t.Error("percent line")
	}
	if MinContent[TrackSizingFunction]() != TrackSizingFunctionMinContent || MaxContent[GridTemplateComponent[string]]().Single() != TrackSizingFunctionMaxContent {
		t.Error("min/max content")
	}
	if FitContent[TrackSizingFunction](LengthPercentageLength(10)) != (TrackSizingFunction{Min: MinTrackSizingFunctionAuto(), Max: MaxTrackSizingFunctionFitContentPx(10)}) {
		t.Error("fit content")
	}
	if Flex[TrackSizingFunction](2) != (TrackSizingFunction{Min: MinTrackSizingFunctionZero, Max: MaxTrackSizingFunctionFr(2)}) {
		t.Error("flex")
	}
	if Minmax[GridTemplateComponent[string]](MinTrackSizingFunctionAuto(), MaxTrackSizingFunctionAuto()).Single() != TrackSizingFunctionAuto {
		t.Error("minmax")
	}
	l := NewLine[Line[GridPlacement[string]]](2)
	if l.Start != GridPlacementLine[string](GridLineFromI16(2)) || l.End != GridPlacementAuto[string]() {
		t.Error("line")
	}
	if Span[GridPlacement[string]](3) != GridPlacementSpan[string](3) {
		t.Error("span")
	}
	tracks := EvenlySizedTracks[string](4)
	if len(tracks) != 1 || tracks[0].Repeat().Count != RepetitionCountCount(4) || tracks[0].Repeat().Tracks[0] != Flex[TrackSizingFunction](1) {
		t.Errorf("evenly sized tracks %v", tracks)
	}
	func() {
		defer func() {
			if r := recover(); r != "called `Result::unwrap()` on an `Err` value: InvalidStringRepetitionValue" {
				t.Errorf("panic %v", r)
			}
		}()
		Repeat[string]("bogus", nil)
	}()
}
