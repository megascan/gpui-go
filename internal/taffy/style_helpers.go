package taffy

import (
	"fmt"
	"reflect"
)

// RepetitionCountInput is the set of types Repeat accepts as a repetition kind (upstream's `TryInto<RepetitionCount>`).
type RepetitionCountInput interface {
	uint16 | string | RepetitionCount
}

// Repeat creates a repeat() track template component; it panics when kind is a string other than "auto-fit" or "auto-fill".
func Repeat[S CheapCloneStr, I RepetitionCountInput](repetitionKind I, tracks []TrackSizingFunction) GridTemplateComponent[S] {
	var count RepetitionCount
	switch v := any(repetitionKind).(type) {
	case uint16:
		count = RepetitionCountFromU16(v)
	case RepetitionCount:
		count = v
	case string:
		c, err := RepetitionCountTryFromStr(v)
		if err != nil {
			panic("called `Result::unwrap()` on an `Err` value: InvalidStringRepetitionValue")
		}
		count = c
	}
	return GridTemplateComponentRepeat(GridTemplateRepetition[S]{
		Count:     count,
		Tracks:    tracks,
		LineNames: nil,
	})
}

// EvenlySizedTracks returns a grid template of count evenly sized flexible tracks.
func EvenlySizedTracks[S CheapCloneStr](count uint16) []GridTemplateComponent[S] {
	repeatedTracks := newVecWithCapacity[TrackSizingFunction](1)
	repeatedTracks = append(repeatedTracks, Flex[TrackSizingFunction](1.0))
	tracks := newVecWithCapacity[GridTemplateComponent[S]](1)
	tracks = append(tracks, Repeat[S](count, repeatedTracks))
	return tracks
}

// NewLine returns a grid line placement at index (upstream's `line`).
func NewLine[T TaffyGridLine[T]](index int16) T {
	var z T
	return z.FromLineIndex(index)
}

// TaffyGridLine is implemented by types that can be built from a grid line index.
type TaffyGridLine[T any] interface {
	FromLineIndex(index int16) T
}

// Span returns a grid span placement of span tracks.
func Span[T TaffyGridSpan[T]](span uint16) T {
	var z T
	return z.FromSpan(span)
}

// TaffyGridSpan is implemented by types that can be built from a grid span.
type TaffyGridSpan[T any] interface {
	FromSpan(span uint16) T
}

// Minmax returns a track sizing function (or a value built from one) with the given min and max.
func Minmax[O any](min MinTrackSizingFunction, max MaxTrackSizingFunction) O {
	return fromMinMax[O](TrackSizingFunction{Min: min, Max: max})
}

// Flex returns a flexible track sizing function minmax(0, <flexFraction>fr) (or a value built from one).
func Flex[O any](flexFraction float64) O {
	return fromMinMax[O](TrackSizingFunction{Min: Zero[MinTrackSizingFunction](), Max: Fr[MaxTrackSizingFunction](flexFraction)})
}

type fromMinMaxer[O any] interface {
	fromMinMax(input TrackSizingFunction) O
}

func fromMinMax[O any](input TrackSizingFunction) O {
	var out O
	if p, ok := any(&out).(*TrackSizingFunction); ok {
		*p = input
		return out
	}
	if f, ok := any(out).(fromMinMaxer[O]); ok {
		return f.fromMinMax(input)
	}
	panic(fmt.Sprintf("taffy: %s does not implement From<MinMax<MinTrackSizingFunction, MaxTrackSizingFunction>>", reflect.TypeFor[O]()))
}

// Zero returns the zero value of T (upstream's `TaffyZero::ZERO`).
func Zero[T TaffyZero[T]]() T {
	var z T
	return z.Zero()
}

// TaffyZero is implemented by types with a zero value (upstream's TaffyZero::ZERO).
type TaffyZero[T any] interface {
	Zero() T
}

func zeroOf[T any]() T {
	var z T
	switch p := any(&z).(type) {
	case *float32:
		*p = 0.0
		return z
	case *Option[float32]:
		*p = some[float32](0.0)
		return z
	}
	if v, ok := any(z).(TaffyZero[T]); ok {
		return v.Zero()
	}
	panic(fmt.Sprintf("taffy: TaffyZero is not implemented for %s", reflect.TypeFor[T]()))
}

// Zero returns a Point with both components zero.
func (Point[T]) Zero() Point[T] {
	return PointZero[T]()
}

// PointZero returns a Point with both components zero.
func PointZero[T any]() Point[T] {
	return Point[T]{X: zeroOf[T](), Y: zeroOf[T]()}
}

// Zero returns a Line with both ends zero.
func (Line[T]) Zero() Line[T] {
	return LineZero[T]()
}

// LineZero returns a Line with both ends zero.
func LineZero[T any]() Line[T] {
	return Line[T]{Start: zeroOf[T](), End: zeroOf[T]()}
}

// Zero returns a Size with both dimensions zero.
func (Size[T]) Zero() Size[T] {
	return SizeZero[T]()
}

// SizeZero returns a Size with both dimensions zero.
func SizeZero[T any]() Size[T] {
	return Size[T]{Width: zeroOf[T](), Height: zeroOf[T]()}
}

// Zero returns a Rect with all sides zero.
func (Rect[T]) Zero() Rect[T] {
	return RectZero[T]()
}

// RectZero returns a Rect with all sides zero.
func RectZero[T any]() Rect[T] {
	return Rect[T]{Left: zeroOf[T](), Right: zeroOf[T](), Top: zeroOf[T](), Bottom: zeroOf[T]()}
}

// Auto returns the auto value of T (upstream's `TaffyAuto::AUTO`).
func Auto[T TaffyAuto[T]]() T {
	var z T
	return z.Auto()
}

// TaffyAuto is implemented by types with an auto value (upstream's TaffyAuto::AUTO).
type TaffyAuto[T any] interface {
	Auto() T
}

func autoOf[T any]() T {
	var z T
	if v, ok := any(z).(TaffyAuto[T]); ok {
		return v.Auto()
	}
	panic(fmt.Sprintf("taffy: TaffyAuto is not implemented for %s", reflect.TypeFor[T]()))
}

// Auto returns a Point with both components auto.
func (Point[T]) Auto() Point[T] {
	return PointAuto[T]()
}

// PointAuto returns a Point with both components auto.
func PointAuto[T any]() Point[T] {
	return Point[T]{X: autoOf[T](), Y: autoOf[T]()}
}

// Auto returns a Line with both ends auto.
func (Line[T]) Auto() Line[T] {
	return LineAuto[T]()
}

// LineAuto returns a Line with both ends auto.
func LineAuto[T any]() Line[T] {
	return Line[T]{Start: autoOf[T](), End: autoOf[T]()}
}

// Auto returns a Size with both dimensions auto.
func (Size[T]) Auto() Size[T] {
	return SizeAuto[T]()
}

// SizeAuto returns a Size with both dimensions auto.
func SizeAuto[T any]() Size[T] {
	return Size[T]{Width: autoOf[T](), Height: autoOf[T]()}
}

// Auto returns a Rect with all sides auto.
func (Rect[T]) Auto() Rect[T] {
	return RectAuto[T]()
}

// RectAuto returns a Rect with all sides auto.
func RectAuto[T any]() Rect[T] {
	return Rect[T]{Left: autoOf[T](), Right: autoOf[T](), Top: autoOf[T](), Bottom: autoOf[T]()}
}

// MinContent returns the min-content value of T (upstream's `TaffyMinContent::MIN_CONTENT`).
func MinContent[T TaffyMinContent[T]]() T {
	var z T
	return z.MinContent()
}

// TaffyMinContent is implemented by types with a min-content value (upstream's TaffyMinContent::MIN_CONTENT).
type TaffyMinContent[T any] interface {
	MinContent() T
}

func minContentOf[T any]() T {
	var z T
	if v, ok := any(z).(TaffyMinContent[T]); ok {
		return v.MinContent()
	}
	panic(fmt.Sprintf("taffy: TaffyMinContent is not implemented for %s", reflect.TypeFor[T]()))
}

// MinContent returns a Point with both components min-content.
func (Point[T]) MinContent() Point[T] {
	return PointMinContent[T]()
}

// PointMinContent returns a Point with both components min-content.
func PointMinContent[T any]() Point[T] {
	return Point[T]{X: minContentOf[T](), Y: minContentOf[T]()}
}

// MinContent returns a Line with both ends min-content.
func (Line[T]) MinContent() Line[T] {
	return LineMinContent[T]()
}

// LineMinContent returns a Line with both ends min-content.
func LineMinContent[T any]() Line[T] {
	return Line[T]{Start: minContentOf[T](), End: minContentOf[T]()}
}

// MinContent returns a Size with both dimensions min-content.
func (Size[T]) MinContent() Size[T] {
	return SizeMinContent[T]()
}

// SizeMinContent returns a Size with both dimensions min-content.
func SizeMinContent[T any]() Size[T] {
	return Size[T]{Width: minContentOf[T](), Height: minContentOf[T]()}
}

// MinContent returns a Rect with all sides min-content.
func (Rect[T]) MinContent() Rect[T] {
	return RectMinContent[T]()
}

// RectMinContent returns a Rect with all sides min-content.
func RectMinContent[T any]() Rect[T] {
	return Rect[T]{Left: minContentOf[T](), Right: minContentOf[T](), Top: minContentOf[T](), Bottom: minContentOf[T]()}
}

// MaxContent returns the max-content value of T (upstream's `TaffyMaxContent::MAX_CONTENT`).
func MaxContent[T TaffyMaxContent[T]]() T {
	var z T
	return z.MaxContent()
}

// TaffyMaxContent is implemented by types with a max-content value (upstream's TaffyMaxContent::MAX_CONTENT).
type TaffyMaxContent[T any] interface {
	MaxContent() T
}

func maxContentOf[T any]() T {
	var z T
	if v, ok := any(z).(TaffyMaxContent[T]); ok {
		return v.MaxContent()
	}
	panic(fmt.Sprintf("taffy: TaffyMaxContent is not implemented for %s", reflect.TypeFor[T]()))
}

// MaxContent returns a Point with both components max-content.
func (Point[T]) MaxContent() Point[T] {
	return PointMaxContent[T]()
}

// PointMaxContent returns a Point with both components max-content.
func PointMaxContent[T any]() Point[T] {
	return Point[T]{X: maxContentOf[T](), Y: maxContentOf[T]()}
}

// MaxContent returns a Line with both ends max-content.
func (Line[T]) MaxContent() Line[T] {
	return LineMaxContent[T]()
}

// LineMaxContent returns a Line with both ends max-content.
func LineMaxContent[T any]() Line[T] {
	return Line[T]{Start: maxContentOf[T](), End: maxContentOf[T]()}
}

// MaxContent returns a Size with both dimensions max-content.
func (Size[T]) MaxContent() Size[T] {
	return SizeMaxContent[T]()
}

// SizeMaxContent returns a Size with both dimensions max-content.
func SizeMaxContent[T any]() Size[T] {
	return Size[T]{Width: maxContentOf[T](), Height: maxContentOf[T]()}
}

// MaxContent returns a Rect with all sides max-content.
func (Rect[T]) MaxContent() Rect[T] {
	return RectMaxContent[T]()
}

// RectMaxContent returns a Rect with all sides max-content.
func RectMaxContent[T any]() Rect[T] {
	return Rect[T]{Left: maxContentOf[T](), Right: maxContentOf[T](), Top: maxContentOf[T](), Bottom: maxContentOf[T]()}
}

// FitContent returns a fit-content value of T with the given limit.
func FitContent[T TaffyFitContent[T]](argument LengthPercentage) T {
	var z T
	return z.FitContent(argument)
}

// TaffyFitContent is implemented by types that can be a fit-content value.
type TaffyFitContent[T any] interface {
	FitContent(argument LengthPercentage) T
}

func fitContentOf[T any](argument LengthPercentage) T {
	var z T
	if v, ok := any(z).(TaffyFitContent[T]); ok {
		return v.FitContent(argument)
	}
	panic(fmt.Sprintf("taffy: TaffyFitContent is not implemented for %s", reflect.TypeFor[T]()))
}

// FitContent returns a Point with both components fit-content(argument).
func (Point[T]) FitContent(argument LengthPercentage) Point[T] {
	return PointFitContent[T](argument)
}

// PointFitContent returns a Point with both components fit-content(argument).
func PointFitContent[T any](argument LengthPercentage) Point[T] {
	return Point[T]{X: fitContentOf[T](argument), Y: fitContentOf[T](argument)}
}

// FitContent returns a Line with both ends fit-content(argument).
func (Line[T]) FitContent(argument LengthPercentage) Line[T] {
	return LineFitContent[T](argument)
}

// LineFitContent returns a Line with both ends fit-content(argument).
func LineFitContent[T any](argument LengthPercentage) Line[T] {
	return Line[T]{Start: fitContentOf[T](argument), End: fitContentOf[T](argument)}
}

// FitContent returns a Size with both dimensions fit-content(argument).
func (Size[T]) FitContent(argument LengthPercentage) Size[T] {
	return SizeFitContent[T](argument)
}

// SizeFitContent returns a Size with both dimensions fit-content(argument).
func SizeFitContent[T any](argument LengthPercentage) Size[T] {
	return Size[T]{Width: fitContentOf[T](argument), Height: fitContentOf[T](argument)}
}

// FitContent returns a Rect with all sides fit-content(argument).
func (Rect[T]) FitContent(argument LengthPercentage) Rect[T] {
	return RectFitContent[T](argument)
}

// RectFitContent returns a Rect with all sides fit-content(argument).
func RectFitContent[T any](argument LengthPercentage) Rect[T] {
	return Rect[T]{
		Left:   fitContentOf[T](argument),
		Right:  fitContentOf[T](argument),
		Top:    fitContentOf[T](argument),
		Bottom: fitContentOf[T](argument),
	}
}

// Length returns a length value of T.
func Length[T FromLength[T]](value float64) T {
	var z T
	return z.FromLength(value)
}

// FromLength is implemented by types that can be built from a length.
type FromLength[T any] interface {
	FromLength(value float64) T
}

func lengthOf[T any](value float64) T {
	var z T
	switch p := any(&z).(type) {
	case *float32:
		*p = float32(value)
		return z
	case *Option[float32]:
		*p = some(float32(value))
		return z
	}
	if v, ok := any(z).(FromLength[T]); ok {
		return v.FromLength(value)
	}
	panic(fmt.Sprintf("taffy: FromLength is not implemented for %s", reflect.TypeFor[T]()))
}

// FromLength returns a Point with both components the length value.
func (Point[T]) FromLength(value float64) Point[T] {
	return PointLength[T](value)
}

// PointLength returns a Point with both components the length value.
func PointLength[T any](value float64) Point[T] {
	return Point[T]{X: lengthOf[T](value), Y: lengthOf[T](value)}
}

// FromLength returns a Line with both ends the length value.
func (Line[T]) FromLength(value float64) Line[T] {
	return LineLength[T](value)
}

// LineLength returns a Line with both ends the length value.
func LineLength[T any](value float64) Line[T] {
	return Line[T]{Start: lengthOf[T](value), End: lengthOf[T](value)}
}

// FromLength returns a Size with both dimensions the length value.
func (Size[T]) FromLength(value float64) Size[T] {
	return SizeLength[T](value)
}

// SizeLength returns a Size with both dimensions the length value.
func SizeLength[T any](value float64) Size[T] {
	return Size[T]{Width: lengthOf[T](value), Height: lengthOf[T](value)}
}

// FromLength returns a Rect with all sides the length value.
func (Rect[T]) FromLength(value float64) Rect[T] {
	return RectLength[T](value)
}

// RectLength returns a Rect with all sides the length value.
func RectLength[T any](value float64) Rect[T] {
	return Rect[T]{Left: lengthOf[T](value), Right: lengthOf[T](value), Top: lengthOf[T](value), Bottom: lengthOf[T](value)}
}

// Percent returns a percentage value of T (1.0 is 100%).
func Percent[T FromPercent[T]](percent float64) T {
	var z T
	return z.FromPercent(percent)
}

// FromPercent is implemented by types that can be built from a percentage.
type FromPercent[T any] interface {
	FromPercent(value float64) T
}

func percentOf[T any](percent float64) T {
	var z T
	switch p := any(&z).(type) {
	case *float32:
		*p = float32(percent)
		return z
	case *Option[float32]:
		*p = some(float32(percent))
		return z
	}
	if v, ok := any(z).(FromPercent[T]); ok {
		return v.FromPercent(percent)
	}
	panic(fmt.Sprintf("taffy: FromPercent is not implemented for %s", reflect.TypeFor[T]()))
}

// FromPercent returns a Point with both components the percentage.
func (Point[T]) FromPercent(percent float64) Point[T] {
	return PointPercent[T](percent)
}

// PointPercent returns a Point with both components the percentage.
func PointPercent[T any](percentValue float64) Point[T] {
	return Point[T]{X: percentOf[T](percentValue), Y: percentOf[T](percentValue)}
}

// FromPercent returns a Line with both ends the percentage.
func (Line[T]) FromPercent(percent float64) Line[T] {
	return LinePercent[T](percent)
}

// LinePercent returns a Line with both ends the percentage.
func LinePercent[T any](percentValue float64) Line[T] {
	return Line[T]{Start: percentOf[T](percentValue), End: percentOf[T](percentValue)}
}

// FromPercent returns a Size with both dimensions the percentage.
func (Size[T]) FromPercent(percent float64) Size[T] {
	return SizePercent[T](percent)
}

// SizePercent returns a Size with both dimensions the percentage.
func SizePercent[T any](percentValue float64) Size[T] {
	return Size[T]{Width: percentOf[T](percentValue), Height: percentOf[T](percentValue)}
}

// FromPercent returns a Rect with all sides the percentage.
func (Rect[T]) FromPercent(percent float64) Rect[T] {
	return RectPercent[T](percent)
}

// RectPercent returns a Rect with all sides the percentage.
func RectPercent[T any](percentValue float64) Rect[T] {
	return Rect[T]{
		Left:   percentOf[T](percentValue),
		Right:  percentOf[T](percentValue),
		Top:    percentOf[T](percentValue),
		Bottom: percentOf[T](percentValue),
	}
}

// Fr returns an fr (flexible) value of T.
func Fr[T FromFr[T]](flex float64) T {
	var z T
	return z.FromFr(flex)
}

// FromFr is implemented by types that can be built from an fr value.
type FromFr[T any] interface {
	FromFr(value float64) T
}
