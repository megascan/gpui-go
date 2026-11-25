package taffy

import (
	"fmt"
	"reflect"
	"unsafe"
)

// GridTemplateAreas is the value of the grid-template-areas property: the named areas plus the size of the template in tracks.
type GridTemplateAreas[S CheapCloneStr] struct {
	Areas       []GridTemplateArea[S]
	RowCount    uint16
	ColumnCount uint16
}

// GridTemplateArea is one named grid area.
type GridTemplateArea[S CheapCloneStr] struct {
	Name        S
	RowStart    uint16
	RowEnd      uint16
	ColumnStart uint16
	ColumnEnd   uint16
}

// NamedGridLine is a named grid line.
type NamedGridLine[S CheapCloneStr] struct {
	Name  S
	Index uint16
}

type gridAreaAxis uint8

const (
	gridAreaAxisRow gridAreaAxis = iota
	gridAreaAxisColumn
)

type gridAreaEnd uint8

const (
	gridAreaEndStart gridAreaEnd = iota
	gridAreaEndEnd
)

// GenericRepetition is a repeat() clause in a grid-template-* definition.
type GenericRepetition[S CheapCloneStr] interface {
	Count() RepetitionCount
	Tracks() []TrackSizingFunction
	TrackCount() uint16
	LinesNames() TemplateLineNames[S]
}

// GenericRepetitionDefaultTrackCount is upstream's default GenericRepetition::track_count.
func GenericRepetitionDefaultTrackCount[S CheapCloneStr](r GenericRepetition[S]) uint16 {
	return uint16(min(len(r.Tracks()), 0xFFFF))
}

// TemplateLineNames is a nested list of line names: one set of names per grid line.
type TemplateLineNames[S CheapCloneStr] = [][]S

// GenericGridTemplateComponentKind is the variant of a GenericGridTemplateComponent.
type GenericGridTemplateComponentKind uint8

const (
	GenericGridTemplateComponentKindSingle GenericGridTemplateComponentKind = iota
	GenericGridTemplateComponentKindRepeat
)

// GenericGridTemplateComponent is a component of a grid-template-* definition with a generic repeat() type.
type GenericGridTemplateComponent[S CheapCloneStr, R GenericRepetition[S]] struct {
	kind   GenericGridTemplateComponentKind
	single TrackSizingFunction
	repeat R
}

// GenericGridTemplateComponentSingle is the Single variant: a single track sizing function.
func GenericGridTemplateComponentSingle[S CheapCloneStr, R GenericRepetition[S]](value TrackSizingFunction) GenericGridTemplateComponent[S, R] {
	return GenericGridTemplateComponent[S, R]{kind: GenericGridTemplateComponentKindSingle, single: value}
}

// GenericGridTemplateComponentRepeat is the Repeat variant: a repeat() clause.
func GenericGridTemplateComponentRepeat[S CheapCloneStr, R GenericRepetition[S]](value R) GenericGridTemplateComponent[S, R] {
	return GenericGridTemplateComponent[S, R]{kind: GenericGridTemplateComponentKindRepeat, repeat: value}
}

// Kind returns the variant.
func (c GenericGridTemplateComponent[S, R]) Kind() GenericGridTemplateComponentKind {
	return c.kind
}

// Single returns the track sizing function of the Single variant.
func (c GenericGridTemplateComponent[S, R]) Single() TrackSizingFunction {
	return c.single
}

// Repeat returns the repetition of the Repeat variant.
func (c GenericGridTemplateComponent[S, R]) Repeat() R {
	return c.repeat
}

// IsAutoRepetition reports whether the component is an auto-repeated fragment.
func (c GenericGridTemplateComponent[S, R]) IsAutoRepetition() bool {
	switch c.kind {
	case GenericGridTemplateComponentKindSingle:
		return false
	}
	switch c.repeat.Count().kind {
	case RepetitionCountKindAutoFit, RepetitionCountKindAutoFill:
		return true
	}
	return false
}

// GridContainerStyle is the set of styles required for a CSS Grid container.
type GridContainerStyle[S CheapCloneStr] interface {
	CoreStyle
	GridTemplateRows() Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]]
	GridTemplateColumns() Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]]
	GridAutoRows() []TrackSizingFunction
	GridAutoColumns() []TrackSizingFunction
	GridTemplateAreas() Option[[]GridTemplateArea[S]]
	GridTemplateAreaRowCount() uint16
	GridTemplateAreaColumnCount() uint16
	GridTemplateColumnNames() Option[TemplateLineNames[S]]
	GridTemplateRowNames() Option[TemplateLineNames[S]]
	GridAutoFlow() GridAutoFlow
	Gap() Size[LengthPercentage]
	AlignContent() AlignContent
	JustifyContent() JustifyContent
	AlignItems() AlignItems
	JustifyItems() AlignItems
	GridTemplateTracks(axis AbsoluteAxis) Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]]
	GridAlignContent(axis AbstractAxis) AlignContent
}

// GridContainerStyleDefaults provides upstream's default GridContainerStyle methods that do not call other trait methods.
type GridContainerStyleDefaults struct{}

// GridAutoFlow returns the default grid auto flow.
func (GridContainerStyleDefaults) GridAutoFlow() GridAutoFlow {
	return StyleDefault[defaultCheapStr]().GridAutoFlow
}

// Gap returns the default gap.
func (GridContainerStyleDefaults) Gap() Size[LengthPercentage] {
	return StyleDefault[defaultCheapStr]().Gap
}

// AlignContent returns the default align-content.
func (GridContainerStyleDefaults) AlignContent() AlignContent {
	return StyleDefault[defaultCheapStr]().AlignContent
}

// JustifyContent returns the default justify-content.
func (GridContainerStyleDefaults) JustifyContent() JustifyContent {
	return StyleDefault[defaultCheapStr]().JustifyContent
}

// AlignItems returns the default align-items.
func (GridContainerStyleDefaults) AlignItems() AlignItems {
	return StyleDefault[defaultCheapStr]().AlignItems
}

// JustifyItems returns the default justify-items.
func (GridContainerStyleDefaults) JustifyItems() AlignItems {
	return StyleDefault[defaultCheapStr]().JustifyItems
}

// GridContainerStyleDefaultGridTemplateAreaRowCount is upstream's default GridContainerStyle::grid_template_area_row_count.
func GridContainerStyleDefaultGridTemplateAreaRowCount[S CheapCloneStr](s GridContainerStyle[S]) uint16 {
	areas, ok := s.GridTemplateAreas().Get()
	if !ok {
		return 0
	}
	var out uint16
	for _, area := range areas {
		out = max(out, max(area.RowEnd, 1)-1)
	}
	return out
}

// GridContainerStyleDefaultGridTemplateAreaColumnCount is upstream's default GridContainerStyle::grid_template_area_column_count.
func GridContainerStyleDefaultGridTemplateAreaColumnCount[S CheapCloneStr](s GridContainerStyle[S]) uint16 {
	areas, ok := s.GridTemplateAreas().Get()
	if !ok {
		return 0
	}
	var out uint16
	for _, area := range areas {
		out = max(out, max(area.ColumnEnd, 1)-1)
	}
	return out
}

// GridContainerStyleDefaultGridTemplateTracks is upstream's default GridContainerStyle::grid_template_tracks.
func GridContainerStyleDefaultGridTemplateTracks[S CheapCloneStr](s GridContainerStyle[S], axis AbsoluteAxis) Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]] {
	switch axis {
	case AbsoluteAxisHorizontal:
		return s.GridTemplateColumns()
	}
	return s.GridTemplateRows()
}

// GridContainerStyleDefaultGridAlignContent is upstream's default GridContainerStyle::grid_align_content (normal resolves to stretch).
func GridContainerStyleDefaultGridAlignContent[S CheapCloneStr](s GridContainerStyle[S], axis AbstractAxis) AlignContent {
	var alignment AlignContent
	switch axis {
	case AbstractAxisInline:
		alignment = s.JustifyContent()
	default:
		alignment = s.AlignContent()
	}
	switch alignment.keyword {
	case AlignContentKeywordNormal:
		return AlignContentStretch
	}
	return alignment
}

// GridItemStyle is the set of styles required for a CSS Grid item.
type GridItemStyle[S CheapCloneStr] interface {
	CoreStyle
	GridRow() Line[GridPlacement[S]]
	GridColumn() Line[GridPlacement[S]]
	AlignSelf() Option[AlignSelf]
	JustifySelf() Option[AlignSelf]
	GridPlacement(axis AbsoluteAxis) Line[GridPlacement[S]]
}

// GridItemStyleDefaults provides upstream's default GridItemStyle methods that do not call other trait methods.
type GridItemStyleDefaults[S CheapCloneStr] struct{}

// GridRow returns the default (auto) row placement.
func (GridItemStyleDefaults[S]) GridRow() Line[GridPlacement[S]] {
	return LineGridPlacementDefault[S]()
}

// GridColumn returns the default (auto) column placement.
func (GridItemStyleDefaults[S]) GridColumn() Line[GridPlacement[S]] {
	return LineGridPlacementDefault[S]()
}

// AlignSelf returns the default align-self.
func (GridItemStyleDefaults[S]) AlignSelf() Option[AlignSelf] {
	return StyleDefault[S]().AlignSelf
}

// JustifySelf returns the default justify-self.
func (GridItemStyleDefaults[S]) JustifySelf() Option[AlignSelf] {
	return StyleDefault[S]().JustifySelf
}

// GridItemStyleDefaultGridPlacement is upstream's default GridItemStyle::grid_placement.
func GridItemStyleDefaultGridPlacement[S CheapCloneStr](s GridItemStyle[S], axis AbsoluteAxis) Line[GridPlacement[S]] {
	switch axis {
	case AbsoluteAxisHorizontal:
		return s.GridColumn()
	}
	return s.GridRow()
}

// GridAutoFlow controls how auto-placed items are placed and whether the dense packing algorithm is used.
type GridAutoFlow uint8

const (
	GridAutoFlowRow GridAutoFlow = iota
	GridAutoFlowColumn
	GridAutoFlowRowDense
	GridAutoFlowColumnDense
)

// GridAutoFlowDefault is GridAutoFlowRow.
const GridAutoFlowDefault = GridAutoFlowRow

// IsDense reports whether auto placement uses the dense algorithm.
func (f GridAutoFlow) IsDense() bool {
	switch f {
	case GridAutoFlowRow, GridAutoFlowColumn:
		return false
	}
	return true
}

// PrimaryAxis returns the axis auto placement fills first.
func (f GridAutoFlow) PrimaryAxis() AbsoluteAxis {
	switch f {
	case GridAutoFlowRow, GridAutoFlowRowDense:
		return AbsoluteAxisHorizontal
	}
	return AbsoluteAxisVertical
}

// GenericGridPlacementKind is the variant of a GenericGridPlacement.
type GenericGridPlacementKind uint8

const (
	GenericGridPlacementKindAuto GenericGridPlacementKind = iota
	GenericGridPlacementKindLine
	GenericGridPlacementKindSpan
)

// GenericGridPlacement is a grid line placement generic over its coordinate system; the zero value is Auto.
type GenericGridPlacement[L GridCoordinate] struct {
	kind GenericGridPlacementKind
	line L
	span uint16
}

// GenericGridPlacementAuto is the Auto variant.
func GenericGridPlacementAuto[L GridCoordinate]() GenericGridPlacement[L] {
	return GenericGridPlacement[L]{}
}

// GenericGridPlacementLine is the Line variant.
func GenericGridPlacementLine[L GridCoordinate](line L) GenericGridPlacement[L] {
	return GenericGridPlacement[L]{kind: GenericGridPlacementKindLine, line: line}
}

// GenericGridPlacementSpan is the Span variant.
func GenericGridPlacementSpan[L GridCoordinate](span uint16) GenericGridPlacement[L] {
	return GenericGridPlacement[L]{kind: GenericGridPlacementKindSpan, span: span}
}

// Kind returns the variant.
func (p GenericGridPlacement[L]) Kind() GenericGridPlacementKind {
	return p.kind
}

// Line returns the line of the Line variant.
func (p GenericGridPlacement[L]) Line() L {
	return p.line
}

// Span returns the span of the Span variant.
func (p GenericGridPlacement[L]) Span() uint16 {
	return p.span
}

type originZeroGridPlacement = GenericGridPlacement[OriginZeroLine]

type nonNamedGridPlacement = GenericGridPlacement[GridLine]

// GridPlacementKind is the variant of a GridPlacement.
type GridPlacementKind uint8

const (
	GridPlacementKindAuto GridPlacementKind = iota
	GridPlacementKindLine
	GridPlacementKindNamedLine
	GridPlacementKindSpan
	GridPlacementKindNamedSpan
)

// GridPlacement is a grid line placement used for grid-[row/column]-[start/end]; the zero value is Auto.
type GridPlacement[S CheapCloneStr] struct {
	kind  GridPlacementKind
	name  S
	value int32
}

// GridPlacementAuto is the Auto variant.
func GridPlacementAuto[S CheapCloneStr]() GridPlacement[S] {
	return GridPlacement[S]{}
}

// GridPlacementLine is the Line variant.
func GridPlacementLine[S CheapCloneStr](line GridLine) GridPlacement[S] {
	return GridPlacement[S]{kind: GridPlacementKindLine, value: int32(line.inner)}
}

// GridPlacementNamedLine is the NamedLine variant.
func GridPlacementNamedLine[S CheapCloneStr](name S, index int16) GridPlacement[S] {
	return GridPlacement[S]{kind: GridPlacementKindNamedLine, name: name, value: int32(index)}
}

// GridPlacementSpan is the Span variant.
func GridPlacementSpan[S CheapCloneStr](span uint16) GridPlacement[S] {
	return GridPlacement[S]{kind: GridPlacementKindSpan, value: int32(span)}
}

// GridPlacementNamedSpan is the NamedSpan variant.
func GridPlacementNamedSpan[S CheapCloneStr](name S, span uint16) GridPlacement[S] {
	return GridPlacement[S]{kind: GridPlacementKindNamedSpan, name: name, value: int32(span)}
}

// GridPlacementDefault is the Auto variant.
func GridPlacementDefault[S CheapCloneStr]() GridPlacement[S] {
	return GridPlacement[S]{}
}

// Kind returns the variant.
func (p GridPlacement[S]) Kind() GridPlacementKind {
	return p.kind
}

// Line returns the line of the Line variant.
func (p GridPlacement[S]) Line() GridLine {
	return GridLineFromI16(int16(p.value))
}

// Name returns the name of the NamedLine and NamedSpan variants.
func (p GridPlacement[S]) Name() S {
	return p.name
}

// Index returns the index of the NamedLine variant.
func (p GridPlacement[S]) Index() int16 {
	return int16(p.value)
}

// Span returns the span of the Span and NamedSpan variants.
func (p GridPlacement[S]) Span() uint16 {
	return uint16(p.value)
}

// Auto returns the Auto variant.
func (GridPlacement[S]) Auto() GridPlacement[S] {
	return GridPlacement[S]{}
}

// FromLineIndex returns the Line variant at index.
func (GridPlacement[S]) FromLineIndex(index int16) GridPlacement[S] {
	return GridPlacementLine[S](GridLineFromI16(index))
}

// FromSpan returns the Span variant.
func (GridPlacement[S]) FromSpan(span uint16) GridPlacement[S] {
	return GridPlacementSpan[S](span)
}

// FromLineIndex returns a Line starting at grid line index and ending auto; only defined for Line[GridPlacement[S]].
func (Line[T]) FromLineIndex(index int16) Line[T] {
	var z T
	if g, ok := any(z).(TaffyGridLine[T]); ok {
		return Line[T]{Start: g.FromLineIndex(index), End: autoOf[T]()}
	}
	panic(fmt.Sprintf("taffy: TaffyGridLine is not implemented for %s", reflect.TypeFor[Line[T]]()))
}

// FromSpan returns a Line starting with a span and ending auto; only defined for Line[GridPlacement[S]].
func (Line[T]) FromSpan(span uint16) Line[T] {
	var z T
	if g, ok := any(z).(TaffyGridSpan[T]); ok {
		return Line[T]{Start: g.FromSpan(span), End: autoOf[T]()}
	}
	panic(fmt.Sprintf("taffy: TaffyGridSpan is not implemented for %s", reflect.TypeFor[Line[T]]()))
}

// IntoOriginZeroPlacementIgnoringNamed converts the placement to origin-zero coordinates; named lines and spans become Auto.
func (p GridPlacement[S]) IntoOriginZeroPlacementIgnoringNamed(explicitTrackCount uint16) GenericGridPlacement[OriginZeroLine] {
	switch p.kind {
	case GridPlacementKindAuto:
		return GenericGridPlacementAuto[OriginZeroLine]()
	case GridPlacementKindSpan:
		return GenericGridPlacementSpan[OriginZeroLine](min(max(p.Span(), 1), maxGridTracks))
	case GridPlacementKindLine:
		line := p.Line()
		switch line.AsI16() {
		case 0:
			return GenericGridPlacementAuto[OriginZeroLine]()
		}
		return GenericGridPlacementLine(line.intoOriginZeroLine(explicitTrackCount))
	}
	return GenericGridPlacementAuto[OriginZeroLine]()
}

// LineGridPlacementIntoOriginZeroIgnoringNamed converts both ends of l to origin-zero coordinates, ignoring named lines.
func LineGridPlacementIntoOriginZeroIgnoringNamed[S CheapCloneStr](l Line[GridPlacement[S]], explicitTrackCount uint16) Line[GenericGridPlacement[OriginZeroLine]] {
	return Line[GenericGridPlacement[OriginZeroLine]]{
		Start: l.Start.IntoOriginZeroPlacementIgnoringNamed(explicitTrackCount),
		End:   l.End.IntoOriginZeroPlacementIgnoringNamed(explicitTrackCount),
	}
}

func nonNamedGridPlacementIntoOriginZeroPlacement(p nonNamedGridPlacement, explicitTrackCount uint16) originZeroGridPlacement {
	switch p.kind {
	case GenericGridPlacementKindAuto:
		return GenericGridPlacementAuto[OriginZeroLine]()
	case GenericGridPlacementKindSpan:
		return GenericGridPlacementSpan[OriginZeroLine](min(max(p.span, 1), maxGridTracks))
	}
	switch p.line.AsI16() {
	case 0:
		return GenericGridPlacementAuto[OriginZeroLine]()
	}
	return GenericGridPlacementLine(p.line.intoOriginZeroLine(explicitTrackCount))
}

// LineGenericGridPlacementIndefiniteSpan returns the span of an indefinitely positioned item; it panics when both ends are lines.
func LineGenericGridPlacementIndefiniteSpan[L GridCoordinate](l Line[GenericGridPlacement[L]]) uint16 {
	const (
		auto = GenericGridPlacementKindAuto
		line = GenericGridPlacementKindLine
		span = GenericGridPlacementKindSpan
	)
	switch {
	case l.Start.kind == line && l.End.kind == auto:
		return 1
	case l.Start.kind == auto && l.End.kind == line:
		return 1
	case l.Start.kind == auto && l.End.kind == auto:
		return 1
	case l.Start.kind == line && l.End.kind == span:
		return l.End.span
	case l.Start.kind == span && l.End.kind == line:
		return l.Start.span
	case l.Start.kind == span && l.End.kind == auto:
		return l.Start.span
	case l.Start.kind == auto && l.End.kind == span:
		return l.End.span
	case l.Start.kind == span && l.End.kind == span:
		return l.Start.span
	}
	panic("indefinite_span should only be called on indefinite grid tracks")
}

// LineGridPlacementIsDefinite reports whether either end is a non-zero line or a named line.
func LineGridPlacementIsDefinite[S CheapCloneStr](l Line[GridPlacement[S]]) bool {
	switch {
	case l.Start.kind == GridPlacementKindLine && l.Start.Line().AsI16() != 0:
		return true
	case l.End.kind == GridPlacementKindLine && l.End.Line().AsI16() != 0:
		return true
	case l.Start.kind == GridPlacementKindNamedLine:
		return true
	case l.End.kind == GridPlacementKindNamedLine:
		return true
	}
	return false
}

func lineNonNamedGridPlacementIsDefinite(l Line[nonNamedGridPlacement]) bool {
	switch {
	case l.Start.kind == GenericGridPlacementKindLine && l.Start.line.AsI16() != 0:
		return true
	case l.End.kind == GenericGridPlacementKindLine && l.End.line.AsI16() != 0:
		return true
	}
	return false
}

func lineNonNamedGridPlacementIntoOriginZero(l Line[nonNamedGridPlacement], explicitTrackCount uint16) Line[originZeroGridPlacement] {
	return Line[originZeroGridPlacement]{
		Start: nonNamedGridPlacementIntoOriginZeroPlacement(l.Start, explicitTrackCount),
		End:   nonNamedGridPlacementIntoOriginZeroPlacement(l.End, explicitTrackCount),
	}
}

// LineOriginZeroGridPlacementIsDefinite reports whether either end is a line.
func LineOriginZeroGridPlacementIsDefinite(l Line[GenericGridPlacement[OriginZeroLine]]) bool {
	return l.Start.kind == GenericGridPlacementKindLine || l.End.kind == GenericGridPlacementKindLine
}

// LineOriginZeroGridPlacementResolveDefiniteGridLines resolves a definite placement to start and end lines.
func LineOriginZeroGridPlacementResolveDefiniteGridLines(l Line[GenericGridPlacement[OriginZeroLine]]) Line[OriginZeroLine] {
	const (
		auto = GenericGridPlacementKindAuto
		line = GenericGridPlacementKindLine
		span = GenericGridPlacementKindSpan
	)
	switch {
	case l.Start.kind == line && l.End.kind == line:
		line1, line2 := l.Start.line, l.End.line
		if line1 == line2 {
			return Line[OriginZeroLine]{Start: line1, End: line1 + 1}
		}
		return Line[OriginZeroLine]{Start: min(line1, line2), End: max(line1, line2)}
	case l.Start.kind == line && l.End.kind == span:
		return Line[OriginZeroLine]{Start: l.Start.line, End: l.Start.line + OriginZeroLine(l.End.span)}
	case l.Start.kind == line && l.End.kind == auto:
		return Line[OriginZeroLine]{Start: l.Start.line, End: l.Start.line + 1}
	case l.Start.kind == span && l.End.kind == line:
		return Line[OriginZeroLine]{Start: l.End.line - OriginZeroLine(l.Start.span), End: l.End.line}
	case l.Start.kind == auto && l.End.kind == line:
		return Line[OriginZeroLine]{Start: l.End.line - 1, End: l.End.line}
	}
	panic("resolve_definite_grid_tracks should only be called on definite grid tracks")
}

// LineOriginZeroGridPlacementResolveAbsolutelyPositionedGridTracks resolves the placement of an absolutely positioned item; unresolvable ends are None.
func LineOriginZeroGridPlacementResolveAbsolutelyPositionedGridTracks(l Line[GenericGridPlacement[OriginZeroLine]]) Line[Option[OriginZeroLine]] {
	const (
		auto = GenericGridPlacementKindAuto
		line = GenericGridPlacementKindLine
		span = GenericGridPlacementKindSpan
	)
	switch {
	case l.Start.kind == line && l.End.kind == line:
		track1, track2 := l.Start.line, l.End.line
		if track1 == track2 {
			return Line[Option[OriginZeroLine]]{Start: some(track1), End: some(track1 + 1)}
		}
		return Line[Option[OriginZeroLine]]{Start: some(min(track1, track2)), End: some(max(track1, track2))}
	case l.Start.kind == line && l.End.kind == span:
		return Line[Option[OriginZeroLine]]{Start: some(l.Start.line), End: some(l.Start.line + OriginZeroLine(l.End.span))}
	case l.Start.kind == line && l.End.kind == auto:
		return Line[Option[OriginZeroLine]]{Start: some(l.Start.line), End: none[OriginZeroLine]()}
	case l.Start.kind == span && l.End.kind == line:
		return Line[Option[OriginZeroLine]]{Start: some(l.End.line - OriginZeroLine(l.Start.span)), End: some(l.End.line)}
	case l.Start.kind == auto && l.End.kind == line:
		return Line[Option[OriginZeroLine]]{Start: none[OriginZeroLine](), End: some(l.End.line)}
	}
	return Line[Option[OriginZeroLine]]{Start: none[OriginZeroLine](), End: none[OriginZeroLine]()}
}

// LineOriginZeroGridPlacementResolveIndefiniteGridTracks resolves an indefinite placement starting at start.
func LineOriginZeroGridPlacementResolveIndefiniteGridTracks(l Line[GenericGridPlacement[OriginZeroLine]], start OriginZeroLine) Line[OriginZeroLine] {
	const (
		auto = GenericGridPlacementKindAuto
		span = GenericGridPlacementKindSpan
	)
	switch {
	case l.Start.kind == auto && l.End.kind == auto:
		return Line[OriginZeroLine]{Start: start, End: start + 1}
	case l.Start.kind == span && l.End.kind == auto:
		return Line[OriginZeroLine]{Start: start, End: start + OriginZeroLine(l.Start.span)}
	case l.Start.kind == auto && l.End.kind == span:
		return Line[OriginZeroLine]{Start: start, End: start + OriginZeroLine(l.End.span)}
	case l.Start.kind == span && l.End.kind == span:
		return Line[OriginZeroLine]{Start: start, End: start + OriginZeroLine(l.Start.span)}
	}
	panic("resolve_indefinite_grid_tracks should only be called on indefinite grid tracks")
}

// LineGridPlacementDefault returns a Line with both ends Auto.
func LineGridPlacementDefault[S CheapCloneStr]() Line[GridPlacement[S]] {
	return Line[GridPlacement[S]]{Start: GridPlacementAuto[S](), End: GridPlacementAuto[S]()}
}

// MaxTrackSizingFunction is the maximum track sizing function of a grid track.
type MaxTrackSizingFunction struct {
	inner CompactLength
}

// MaxTrackSizingFunctionZero is a zero-length max track sizing function.
var MaxTrackSizingFunctionZero = MaxTrackSizingFunction{CompactLengthZero}

// Zero returns MaxTrackSizingFunctionZero.
func (MaxTrackSizingFunction) Zero() MaxTrackSizingFunction {
	return MaxTrackSizingFunctionZero
}

// Auto returns MaxTrackSizingFunctionAuto().
func (MaxTrackSizingFunction) Auto() MaxTrackSizingFunction {
	return MaxTrackSizingFunctionAuto()
}

// MinContent returns MaxTrackSizingFunctionMinContent().
func (MaxTrackSizingFunction) MinContent() MaxTrackSizingFunction {
	return MaxTrackSizingFunctionMinContent()
}

// MaxContent returns MaxTrackSizingFunctionMaxContent().
func (MaxTrackSizingFunction) MaxContent() MaxTrackSizingFunction {
	return MaxTrackSizingFunctionMaxContent()
}

// FromLength returns a length value.
func (MaxTrackSizingFunction) FromLength(value float64) MaxTrackSizingFunction {
	return MaxTrackSizingFunctionLength(float32(value))
}

// FromPercent returns a percentage value.
func (MaxTrackSizingFunction) FromPercent(value float64) MaxTrackSizingFunction {
	return MaxTrackSizingFunctionPercent(float32(value))
}

// FitContent returns fit-content(argument).
func (MaxTrackSizingFunction) FitContent(argument LengthPercentage) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLength{}.FitContent(argument)}
}

// FromFr returns an fr value.
func (MaxTrackSizingFunction) FromFr(value float64) MaxTrackSizingFunction {
	return MaxTrackSizingFunctionFr(float32(value))
}

// MaxTrackSizingFunctionFromLengthPercentage converts a LengthPercentage.
func MaxTrackSizingFunctionFromLengthPercentage(input LengthPercentage) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{input.inner}
}

// MaxTrackSizingFunctionFromLengthPercentageAuto converts a LengthPercentageAuto.
func MaxTrackSizingFunctionFromLengthPercentageAuto(input LengthPercentageAuto) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{input.inner}
}

// MaxTrackSizingFunctionFromDimension converts a Dimension; values that are not valid max track sizing functions become auto.
func MaxTrackSizingFunctionFromDimension(input Dimension) MaxTrackSizingFunction {
	switch input.inner.Tag() {
	case CompactLengthFitContentKeywordTag, CompactLengthStretchTag, CompactLengthContentTag:
		return MaxTrackSizingFunctionAuto()
	}
	return MaxTrackSizingFunction{input.inner}
}

// MaxTrackSizingFunctionFromMinTrackSizingFunction converts a MinTrackSizingFunction.
func MaxTrackSizingFunctionFromMinTrackSizingFunction(input MinTrackSizingFunction) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{input.inner}
}

// MaxTrackSizingFunctionLength is an absolute length in points.
func MaxTrackSizingFunctionLength(val float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLengthLength(val)}
}

// MaxTrackSizingFunctionPercent is a percentage of the container's size (1.0 is 100%).
func MaxTrackSizingFunctionPercent(val float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLengthPercent(val)}
}

// MaxTrackSizingFunctionAuto is the auto track sizing function.
func MaxTrackSizingFunctionAuto() MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLengthAuto()}
}

// MaxTrackSizingFunctionMinContent is the min-content track sizing function.
func MaxTrackSizingFunctionMinContent() MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLengthMinContent()}
}

// MaxTrackSizingFunctionMaxContent is the max-content track sizing function.
func MaxTrackSizingFunctionMaxContent() MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLengthMaxContent()}
}

// MaxTrackSizingFunctionFitContentPx is fit-content(<limit>px).
func MaxTrackSizingFunctionFitContentPx(limit float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLengthFitContentPx(limit)}
}

// MaxTrackSizingFunctionFitContentPercent is fit-content(<limit>%).
func MaxTrackSizingFunctionFitContentPercent(limit float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLengthFitContentPercent(limit)}
}

// MaxTrackSizingFunctionFr is a fraction of the remaining free space.
func MaxTrackSizingFunctionFr(val float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{CompactLengthFr(val)}
}

// MaxTrackSizingFunctionFromRaw creates a value from a raw CompactLength without validating it.
func MaxTrackSizingFunctionFromRaw(val CompactLength) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{val}
}

// IntoRaw returns the underlying CompactLength.
func (m MaxTrackSizingFunction) IntoRaw() CompactLength {
	return m.inner
}

// IsIntrinsic reports whether the value is min-content, max-content, fit-content or auto.
func (m MaxTrackSizingFunction) IsIntrinsic() bool {
	return m.inner.IsIntrinsic()
}

// IsMaxContentAlike reports whether the value is max-content, fit-content or auto.
func (m MaxTrackSizingFunction) IsMaxContentAlike() bool {
	return m.inner.IsMaxContentAlike()
}

// IsFr reports whether the value is an fr value.
func (m MaxTrackSizingFunction) IsFr() bool {
	return m.inner.IsFr()
}

// IsAuto reports whether the value is auto.
func (m MaxTrackSizingFunction) IsAuto() bool {
	return m.inner.IsAuto()
}

// IsMinContent reports whether the value is min-content.
func (m MaxTrackSizingFunction) IsMinContent() bool {
	return m.inner.IsMinContent()
}

// IsMaxContent reports whether the value is max-content.
func (m MaxTrackSizingFunction) IsMaxContent() bool {
	return m.inner.IsMaxContent()
}

// IsFitContent reports whether the value is fit-content(...).
func (m MaxTrackSizingFunction) IsFitContent() bool {
	return m.inner.IsFitContent()
}

// IsMaxOrFitContent reports whether the value is max-content or fit-content(...).
func (m MaxTrackSizingFunction) IsMaxOrFitContent() bool {
	return m.inner.IsMaxOrFitContent()
}

// HasDefiniteValue reports whether the value resolves to a definite size given parentSize.
func (m MaxTrackSizingFunction) HasDefiniteValue(parentSize Option[float32]) bool {
	switch m.inner.Tag() {
	case CompactLengthLengthTag:
		return true
	case CompactLengthPercentTag:
		return parentSize.IsSome()
	}
	return false
}

// DefiniteValue returns the definite size of a length or percentage, or None.
func (m MaxTrackSizingFunction) DefiniteValue(parentSize Option[float32], calcResolver func(unsafe.Pointer, float32) float32) Option[float32] {
	switch m.inner.Tag() {
	case CompactLengthLengthTag:
		return some(m.inner.Value())
	case CompactLengthPercentTag:
		if size, ok := parentSize.Get(); ok {
			return some(float32(m.inner.Value() * size))
		}
		return none[float32]()
	}
	return none[float32]()
}

// FitContentLimit returns the limit of a fit-content value, or None.
func (m MaxTrackSizingFunction) FitContentLimit(parentSize Option[float32], calcResolver func(unsafe.Pointer, float32) float32) Option[float32] {
	switch m.inner.Tag() {
	case CompactLengthFitContentPxTag:
		return some(m.inner.Value())
	case CompactLengthFitContentPercentTag:
		if size, ok := parentSize.Get(); ok {
			return some(float32(m.inner.Value() * size))
		}
		return none[float32]()
	}
	return none[float32]()
}

// DefiniteLimit returns the fit-content limit, or else the definite value.
func (m MaxTrackSizingFunction) DefiniteLimit(parentSize Option[float32], calcResolver func(unsafe.Pointer, float32) float32) Option[float32] {
	switch m.inner.Tag() {
	case CompactLengthFitContentPxTag:
		return some(m.inner.Value())
	case CompactLengthFitContentPercentTag:
		if size, ok := parentSize.Get(); ok {
			return some(float32(m.inner.Value() * size))
		}
		return none[float32]()
	}
	return m.DefiniteValue(parentSize, calcResolver)
}

// ResolvedPercentageSize resolves percentage values against parentSize; other values return None.
func (m MaxTrackSizingFunction) ResolvedPercentageSize(parentSize float32, calcResolver func(unsafe.Pointer, float32) float32) Option[float32] {
	return m.inner.ResolvedPercentageSize(parentSize, calcResolver)
}

// UsesPercentage reports whether the value depends on the size of the parent.
func (m MaxTrackSizingFunction) UsesPercentage() bool {
	return m.inner.UsesPercentage()
}

// Expand converts the value into an ExpandedMaxTrackSizingFunction.
func (m MaxTrackSizingFunction) Expand() ExpandedMaxTrackSizingFunction {
	switch m.inner.Tag() {
	case CompactLengthLengthTag:
		return ExpandedMaxTrackSizingFunctionLength(m.inner.Value())
	case CompactLengthPercentTag:
		return ExpandedMaxTrackSizingFunctionPercent(m.inner.Value())
	case CompactLengthAutoTag:
		return ExpandedMaxTrackSizingFunctionAuto
	case CompactLengthMinContentTag:
		return ExpandedMaxTrackSizingFunctionMinContent
	case CompactLengthMaxContentTag:
		return ExpandedMaxTrackSizingFunctionMaxContent
	case CompactLengthFitContentPxTag:
		return ExpandedMaxTrackSizingFunctionFitContentPx(m.inner.Value())
	case CompactLengthFitContentPercentTag:
		return ExpandedMaxTrackSizingFunctionFitContentPercent(m.inner.Value())
	case CompactLengthFrTag:
		return ExpandedMaxTrackSizingFunctionFr(m.inner.Value())
	}
	panic("internal error: entered unreachable code: MaxTrackSizingFunction contains a value with an invalid tag")
}

// ExpandedMaxTrackSizingFunctionKind is the variant of an ExpandedMaxTrackSizingFunction.
type ExpandedMaxTrackSizingFunctionKind uint8

const (
	ExpandedMaxTrackSizingFunctionKindLength ExpandedMaxTrackSizingFunctionKind = iota
	ExpandedMaxTrackSizingFunctionKindPercent
	ExpandedMaxTrackSizingFunctionKindAuto
	ExpandedMaxTrackSizingFunctionKindMinContent
	ExpandedMaxTrackSizingFunctionKindMaxContent
	ExpandedMaxTrackSizingFunctionKindFitContentPx
	ExpandedMaxTrackSizingFunctionKindFitContentPercent
	ExpandedMaxTrackSizingFunctionKindFr
)

// ExpandedMaxTrackSizingFunction is the matchable form of a MaxTrackSizingFunction.
type ExpandedMaxTrackSizingFunction struct {
	kind  ExpandedMaxTrackSizingFunctionKind
	value float32
}

// ExpandedMaxTrackSizingFunctionLength is the Length variant.
func ExpandedMaxTrackSizingFunctionLength(val float32) ExpandedMaxTrackSizingFunction {
	return ExpandedMaxTrackSizingFunction{kind: ExpandedMaxTrackSizingFunctionKindLength, value: val}
}

// ExpandedMaxTrackSizingFunctionPercent is the Percent variant.
func ExpandedMaxTrackSizingFunctionPercent(val float32) ExpandedMaxTrackSizingFunction {
	return ExpandedMaxTrackSizingFunction{kind: ExpandedMaxTrackSizingFunctionKindPercent, value: val}
}

var (
	// ExpandedMaxTrackSizingFunctionAuto is the Auto variant.
	ExpandedMaxTrackSizingFunctionAuto = ExpandedMaxTrackSizingFunction{kind: ExpandedMaxTrackSizingFunctionKindAuto}
	// ExpandedMaxTrackSizingFunctionMinContent is the MinContent variant.
	ExpandedMaxTrackSizingFunctionMinContent = ExpandedMaxTrackSizingFunction{kind: ExpandedMaxTrackSizingFunctionKindMinContent}
	// ExpandedMaxTrackSizingFunctionMaxContent is the MaxContent variant.
	ExpandedMaxTrackSizingFunctionMaxContent = ExpandedMaxTrackSizingFunction{kind: ExpandedMaxTrackSizingFunctionKindMaxContent}
)

// ExpandedMaxTrackSizingFunctionFitContentPx is the FitContentPx variant.
func ExpandedMaxTrackSizingFunctionFitContentPx(val float32) ExpandedMaxTrackSizingFunction {
	return ExpandedMaxTrackSizingFunction{kind: ExpandedMaxTrackSizingFunctionKindFitContentPx, value: val}
}

// ExpandedMaxTrackSizingFunctionFitContentPercent is the FitContentPercent variant.
func ExpandedMaxTrackSizingFunctionFitContentPercent(val float32) ExpandedMaxTrackSizingFunction {
	return ExpandedMaxTrackSizingFunction{kind: ExpandedMaxTrackSizingFunctionKindFitContentPercent, value: val}
}

// ExpandedMaxTrackSizingFunctionFr is the Fr variant.
func ExpandedMaxTrackSizingFunctionFr(val float32) ExpandedMaxTrackSizingFunction {
	return ExpandedMaxTrackSizingFunction{kind: ExpandedMaxTrackSizingFunctionKindFr, value: val}
}

// Kind returns the variant.
func (e ExpandedMaxTrackSizingFunction) Kind() ExpandedMaxTrackSizingFunctionKind {
	return e.kind
}

// Value returns the payload of the variants that carry one.
func (e ExpandedMaxTrackSizingFunction) Value() float32 {
	return e.value
}

// ExpandedMaxTrackSizingFunctionFromMaxTrackSizingFunction converts value with Expand.
func ExpandedMaxTrackSizingFunctionFromMaxTrackSizingFunction(value MaxTrackSizingFunction) ExpandedMaxTrackSizingFunction {
	return value.Expand()
}

// MaxTrackSizingFunctionFromExpandedMaxTrackSizingFunction converts an expanded value back.
func MaxTrackSizingFunctionFromExpandedMaxTrackSizingFunction(value ExpandedMaxTrackSizingFunction) MaxTrackSizingFunction {
	switch value.kind {
	case ExpandedMaxTrackSizingFunctionKindLength:
		return MaxTrackSizingFunctionLength(value.value)
	case ExpandedMaxTrackSizingFunctionKindPercent:
		return MaxTrackSizingFunctionPercent(value.value)
	case ExpandedMaxTrackSizingFunctionKindAuto:
		return MaxTrackSizingFunctionAuto()
	case ExpandedMaxTrackSizingFunctionKindMinContent:
		return MaxTrackSizingFunctionMinContent()
	case ExpandedMaxTrackSizingFunctionKindMaxContent:
		return MaxTrackSizingFunctionMaxContent()
	case ExpandedMaxTrackSizingFunctionKindFitContentPx:
		return MaxTrackSizingFunctionFitContentPx(value.value)
	case ExpandedMaxTrackSizingFunctionKindFitContentPercent:
		return MaxTrackSizingFunctionFitContentPercent(value.value)
	}
	return MaxTrackSizingFunctionFr(value.value)
}

// MinTrackSizingFunction is the minimum track sizing function of a grid track.
type MinTrackSizingFunction struct {
	inner CompactLength
}

// MinTrackSizingFunctionZero is a zero-length min track sizing function.
var MinTrackSizingFunctionZero = MinTrackSizingFunction{CompactLengthZero}

// Zero returns MinTrackSizingFunctionZero.
func (MinTrackSizingFunction) Zero() MinTrackSizingFunction {
	return MinTrackSizingFunctionZero
}

// Auto returns MinTrackSizingFunctionAuto().
func (MinTrackSizingFunction) Auto() MinTrackSizingFunction {
	return MinTrackSizingFunctionAuto()
}

// MinContent returns MinTrackSizingFunctionMinContent().
func (MinTrackSizingFunction) MinContent() MinTrackSizingFunction {
	return MinTrackSizingFunctionMinContent()
}

// MaxContent returns MinTrackSizingFunctionMaxContent().
func (MinTrackSizingFunction) MaxContent() MinTrackSizingFunction {
	return MinTrackSizingFunctionMaxContent()
}

// FromLength returns a length value.
func (MinTrackSizingFunction) FromLength(value float64) MinTrackSizingFunction {
	return MinTrackSizingFunctionLength(float32(value))
}

// FromPercent returns a percentage value.
func (MinTrackSizingFunction) FromPercent(value float64) MinTrackSizingFunction {
	return MinTrackSizingFunctionPercent(float32(value))
}

// MinTrackSizingFunctionFromLengthPercentage converts a LengthPercentage.
func MinTrackSizingFunctionFromLengthPercentage(input LengthPercentage) MinTrackSizingFunction {
	return MinTrackSizingFunction{input.inner}
}

// MinTrackSizingFunctionFromLengthPercentageAuto converts a LengthPercentageAuto.
func MinTrackSizingFunctionFromLengthPercentageAuto(input LengthPercentageAuto) MinTrackSizingFunction {
	return MinTrackSizingFunction{input.inner}
}

// MinTrackSizingFunctionFromDimension converts a Dimension; values that are not valid min track sizing functions become auto.
func MinTrackSizingFunctionFromDimension(input Dimension) MinTrackSizingFunction {
	switch input.inner.Tag() {
	case CompactLengthFitContentPxTag, CompactLengthFitContentPercentTag, CompactLengthFitContentKeywordTag,
		CompactLengthStretchTag, CompactLengthContentTag:
		return MinTrackSizingFunctionAuto()
	}
	if input.inner.IsFitContentCalc() {
		return MinTrackSizingFunctionAuto()
	}
	return MinTrackSizingFunction{input.inner}
}

// MinTrackSizingFunctionFromMaxTrackSizingFunction converts a MaxTrackSizingFunction; fr and fit-content values become auto.
func MinTrackSizingFunctionFromMaxTrackSizingFunction(input MaxTrackSizingFunction) MinTrackSizingFunction {
	if input.IsFr() || input.IsFitContent() {
		return MinTrackSizingFunctionAuto()
	}
	return MinTrackSizingFunction{input.inner}
}

// MinTrackSizingFunctionLength is an absolute length in points.
func MinTrackSizingFunctionLength(val float32) MinTrackSizingFunction {
	return MinTrackSizingFunction{CompactLengthLength(val)}
}

// MinTrackSizingFunctionPercent is a percentage of the container's size (1.0 is 100%).
func MinTrackSizingFunctionPercent(val float32) MinTrackSizingFunction {
	return MinTrackSizingFunction{CompactLengthPercent(val)}
}

// MinTrackSizingFunctionAuto is the auto track sizing function.
func MinTrackSizingFunctionAuto() MinTrackSizingFunction {
	return MinTrackSizingFunction{CompactLengthAuto()}
}

// MinTrackSizingFunctionMinContent is the min-content track sizing function.
func MinTrackSizingFunctionMinContent() MinTrackSizingFunction {
	return MinTrackSizingFunction{CompactLengthMinContent()}
}

// MinTrackSizingFunctionMaxContent is the max-content track sizing function.
func MinTrackSizingFunctionMaxContent() MinTrackSizingFunction {
	return MinTrackSizingFunction{CompactLengthMaxContent()}
}

// MinTrackSizingFunctionFromRaw creates a value from a raw CompactLength without validating it.
func MinTrackSizingFunctionFromRaw(val CompactLength) MinTrackSizingFunction {
	return MinTrackSizingFunction{val}
}

// IntoRaw returns the underlying CompactLength.
func (m MinTrackSizingFunction) IntoRaw() CompactLength {
	return m.inner
}

// IsIntrinsic reports whether the value is min-content, max-content or auto.
func (m MinTrackSizingFunction) IsIntrinsic() bool {
	return m.inner.IsIntrinsic()
}

// IsMinOrMaxContent reports whether the value is min-content or max-content.
func (m MinTrackSizingFunction) IsMinOrMaxContent() bool {
	return m.inner.IsMinOrMaxContent()
}

// IsFr reports whether the value is an fr value.
func (m MinTrackSizingFunction) IsFr() bool {
	return m.inner.IsFr()
}

// IsAuto reports whether the value is auto.
func (m MinTrackSizingFunction) IsAuto() bool {
	return m.inner.IsAuto()
}

// BehavesAsAuto reports whether the value is auto, or a percentage with an indefinite parent size.
func (m MinTrackSizingFunction) BehavesAsAuto(parentSize Option[float32]) bool {
	return m.inner.IsAuto() || (parentSize.IsNone() && m.inner.UsesPercentage())
}

// IsMinContent reports whether the value is min-content.
func (m MinTrackSizingFunction) IsMinContent() bool {
	return m.inner.IsMinContent()
}

// IsMaxContent reports whether the value is max-content.
func (m MinTrackSizingFunction) IsMaxContent() bool {
	return m.inner.IsMaxContent()
}

// DefiniteValue returns the definite size of a length or percentage, or None.
func (m MinTrackSizingFunction) DefiniteValue(parentSize Option[float32], calcResolver func(unsafe.Pointer, float32) float32) Option[float32] {
	switch m.inner.Tag() {
	case CompactLengthLengthTag:
		return some(m.inner.Value())
	case CompactLengthPercentTag:
		if size, ok := parentSize.Get(); ok {
			return some(float32(m.inner.Value() * size))
		}
		return none[float32]()
	}
	return none[float32]()
}

// ResolvedPercentageSize resolves percentage values against parentSize; other values return None.
func (m MinTrackSizingFunction) ResolvedPercentageSize(parentSize float32, calcResolver func(unsafe.Pointer, float32) float32) Option[float32] {
	return m.inner.ResolvedPercentageSize(parentSize, calcResolver)
}

// UsesPercentage reports whether the value depends on the size of the parent.
func (m MinTrackSizingFunction) UsesPercentage() bool {
	return m.inner.UsesPercentage()
}

// Expand converts the value into an ExpandedMinTrackSizingFunction.
func (m MinTrackSizingFunction) Expand() ExpandedMinTrackSizingFunction {
	switch m.inner.Tag() {
	case CompactLengthLengthTag:
		return ExpandedMinTrackSizingFunctionLength(m.inner.Value())
	case CompactLengthPercentTag:
		return ExpandedMinTrackSizingFunctionPercent(m.inner.Value())
	case CompactLengthAutoTag:
		return ExpandedMinTrackSizingFunctionAuto
	case CompactLengthMinContentTag:
		return ExpandedMinTrackSizingFunctionMinContent
	case CompactLengthMaxContentTag:
		return ExpandedMinTrackSizingFunctionMaxContent
	}
	panic("internal error: entered unreachable code: MinTrackSizingFunction contains a value with an invalid tag")
}

// ExpandedMinTrackSizingFunctionKind is the variant of an ExpandedMinTrackSizingFunction.
type ExpandedMinTrackSizingFunctionKind uint8

const (
	ExpandedMinTrackSizingFunctionKindLength ExpandedMinTrackSizingFunctionKind = iota
	ExpandedMinTrackSizingFunctionKindPercent
	ExpandedMinTrackSizingFunctionKindAuto
	ExpandedMinTrackSizingFunctionKindMinContent
	ExpandedMinTrackSizingFunctionKindMaxContent
)

// ExpandedMinTrackSizingFunction is the matchable form of a MinTrackSizingFunction.
type ExpandedMinTrackSizingFunction struct {
	kind  ExpandedMinTrackSizingFunctionKind
	value float32
}

// ExpandedMinTrackSizingFunctionLength is the Length variant.
func ExpandedMinTrackSizingFunctionLength(val float32) ExpandedMinTrackSizingFunction {
	return ExpandedMinTrackSizingFunction{kind: ExpandedMinTrackSizingFunctionKindLength, value: val}
}

// ExpandedMinTrackSizingFunctionPercent is the Percent variant.
func ExpandedMinTrackSizingFunctionPercent(val float32) ExpandedMinTrackSizingFunction {
	return ExpandedMinTrackSizingFunction{kind: ExpandedMinTrackSizingFunctionKindPercent, value: val}
}

var (
	// ExpandedMinTrackSizingFunctionAuto is the Auto variant.
	ExpandedMinTrackSizingFunctionAuto = ExpandedMinTrackSizingFunction{kind: ExpandedMinTrackSizingFunctionKindAuto}
	// ExpandedMinTrackSizingFunctionMinContent is the MinContent variant.
	ExpandedMinTrackSizingFunctionMinContent = ExpandedMinTrackSizingFunction{kind: ExpandedMinTrackSizingFunctionKindMinContent}
	// ExpandedMinTrackSizingFunctionMaxContent is the MaxContent variant.
	ExpandedMinTrackSizingFunctionMaxContent = ExpandedMinTrackSizingFunction{kind: ExpandedMinTrackSizingFunctionKindMaxContent}
)

// Kind returns the variant.
func (e ExpandedMinTrackSizingFunction) Kind() ExpandedMinTrackSizingFunctionKind {
	return e.kind
}

// Value returns the payload of the variants that carry one.
func (e ExpandedMinTrackSizingFunction) Value() float32 {
	return e.value
}

// ExpandedMinTrackSizingFunctionFromMinTrackSizingFunction converts value with Expand.
func ExpandedMinTrackSizingFunctionFromMinTrackSizingFunction(value MinTrackSizingFunction) ExpandedMinTrackSizingFunction {
	return value.Expand()
}

// MinTrackSizingFunctionFromExpandedMinTrackSizingFunction converts an expanded value back.
func MinTrackSizingFunctionFromExpandedMinTrackSizingFunction(value ExpandedMinTrackSizingFunction) MinTrackSizingFunction {
	switch value.kind {
	case ExpandedMinTrackSizingFunctionKindLength:
		return MinTrackSizingFunctionLength(value.value)
	case ExpandedMinTrackSizingFunctionKindPercent:
		return MinTrackSizingFunctionPercent(value.value)
	case ExpandedMinTrackSizingFunctionKindAuto:
		return MinTrackSizingFunctionAuto()
	case ExpandedMinTrackSizingFunctionKindMinContent:
		return MinTrackSizingFunctionMinContent()
	}
	return MinTrackSizingFunctionMaxContent()
}

// TrackSizingFunction is the sizing function of a grid track: a min and a max track sizing function.
type TrackSizingFunction = MinMax[MinTrackSizingFunction, MaxTrackSizingFunction]

// TrackSizingFunctionMinSizingFunction returns the min track sizing function of t.
func TrackSizingFunctionMinSizingFunction(t TrackSizingFunction) MinTrackSizingFunction {
	return t.Min
}

// TrackSizingFunctionMaxSizingFunction returns the max track sizing function of t.
func TrackSizingFunctionMaxSizingFunction(t TrackSizingFunction) MaxTrackSizingFunction {
	return t.Max
}

// TrackSizingFunctionHasFixedComponent reports whether min or max is a length or percentage.
func TrackSizingFunctionHasFixedComponent(t TrackSizingFunction) bool {
	return t.Min.inner.IsLengthOrPercentage() || t.Max.inner.IsLengthOrPercentage()
}

var (
	// TrackSizingFunctionAuto is minmax(auto, auto).
	TrackSizingFunctionAuto = TrackSizingFunction{Min: MinTrackSizingFunctionAuto(), Max: MaxTrackSizingFunctionAuto()}
	// TrackSizingFunctionMinContent is minmax(min-content, min-content).
	TrackSizingFunctionMinContent = TrackSizingFunction{Min: MinTrackSizingFunctionMinContent(), Max: MaxTrackSizingFunctionMinContent()}
	// TrackSizingFunctionMaxContent is minmax(max-content, max-content).
	TrackSizingFunctionMaxContent = TrackSizingFunction{Min: MinTrackSizingFunctionMaxContent(), Max: MaxTrackSizingFunctionMaxContent()}
	// TrackSizingFunctionZero is minmax(0, 0).
	TrackSizingFunctionZero = TrackSizingFunction{Min: MinTrackSizingFunctionZero, Max: MaxTrackSizingFunctionZero}
)

// TrackSizingFunctionFitContent is minmax(auto, fit-content(argument)).
func TrackSizingFunctionFitContent(argument LengthPercentage) TrackSizingFunction {
	return TrackSizingFunction{Min: MinTrackSizingFunctionAuto(), Max: MaxTrackSizingFunction{}.FitContent(argument)}
}

// TrackSizingFunctionFromLengthPercentage converts a LengthPercentage into both min and max.
func TrackSizingFunctionFromLengthPercentage(input LengthPercentage) TrackSizingFunction {
	return TrackSizingFunction{Min: MinTrackSizingFunctionFromLengthPercentage(input), Max: MaxTrackSizingFunctionFromLengthPercentage(input)}
}

// TrackSizingFunctionFromLengthPercentageAuto converts a LengthPercentageAuto into both min and max.
func TrackSizingFunctionFromLengthPercentageAuto(input LengthPercentageAuto) TrackSizingFunction {
	return TrackSizingFunction{Min: MinTrackSizingFunctionFromLengthPercentageAuto(input), Max: MaxTrackSizingFunctionFromLengthPercentageAuto(input)}
}

// TrackSizingFunctionFromDimension converts a Dimension into both min and max.
func TrackSizingFunctionFromDimension(input Dimension) TrackSizingFunction {
	return TrackSizingFunction{Min: MinTrackSizingFunctionFromDimension(input), Max: MaxTrackSizingFunctionFromDimension(input)}
}

func asTrackSizingFunction[Min, Max any](m MinMax[Min, Max], op string) {
	if _, ok := any(m).(TrackSizingFunction); !ok {
		panic(fmt.Sprintf("taffy: MinMax.%s is only defined for TrackSizingFunction, not %s", op, reflect.TypeFor[MinMax[Min, Max]]()))
	}
}

// Auto returns TrackSizingFunctionAuto; only defined for TrackSizingFunction.
func (m MinMax[Min, Max]) Auto() MinMax[Min, Max] {
	asTrackSizingFunction(m, "Auto")
	return any(TrackSizingFunctionAuto).(MinMax[Min, Max])
}

// MinContent returns TrackSizingFunctionMinContent; only defined for TrackSizingFunction.
func (m MinMax[Min, Max]) MinContent() MinMax[Min, Max] {
	asTrackSizingFunction(m, "MinContent")
	return any(TrackSizingFunctionMinContent).(MinMax[Min, Max])
}

// MaxContent returns TrackSizingFunctionMaxContent; only defined for TrackSizingFunction.
func (m MinMax[Min, Max]) MaxContent() MinMax[Min, Max] {
	asTrackSizingFunction(m, "MaxContent")
	return any(TrackSizingFunctionMaxContent).(MinMax[Min, Max])
}

// Zero returns TrackSizingFunctionZero; only defined for TrackSizingFunction.
func (m MinMax[Min, Max]) Zero() MinMax[Min, Max] {
	asTrackSizingFunction(m, "Zero")
	return any(TrackSizingFunctionZero).(MinMax[Min, Max])
}

// FitContent returns TrackSizingFunctionFitContent(argument); only defined for TrackSizingFunction.
func (m MinMax[Min, Max]) FitContent(argument LengthPercentage) MinMax[Min, Max] {
	asTrackSizingFunction(m, "FitContent")
	return any(TrackSizingFunctionFitContent(argument)).(MinMax[Min, Max])
}

// FromLength returns a track with min and max the length value; only defined for TrackSizingFunction.
func (m MinMax[Min, Max]) FromLength(value float64) MinMax[Min, Max] {
	asTrackSizingFunction(m, "FromLength")
	return any(TrackSizingFunction{Min: MinTrackSizingFunction{}.FromLength(value), Max: MaxTrackSizingFunction{}.FromLength(value)}).(MinMax[Min, Max])
}

// FromPercent returns a track with min and max the percentage; only defined for TrackSizingFunction.
func (m MinMax[Min, Max]) FromPercent(percent float64) MinMax[Min, Max] {
	asTrackSizingFunction(m, "FromPercent")
	return any(TrackSizingFunction{Min: MinTrackSizingFunction{}.FromPercent(percent), Max: MaxTrackSizingFunction{}.FromPercent(percent)}).(MinMax[Min, Max])
}

// FromFr returns minmax(auto, <flex>fr); only defined for TrackSizingFunction.
func (m MinMax[Min, Max]) FromFr(flex float64) MinMax[Min, Max] {
	asTrackSizingFunction(m, "FromFr")
	return any(TrackSizingFunction{Min: MinTrackSizingFunctionAuto(), Max: MaxTrackSizingFunction{}.FromFr(flex)}).(MinMax[Min, Max])
}

// RepetitionCountKind is the variant of a RepetitionCount.
type RepetitionCountKind uint8

const (
	RepetitionCountKindAutoFill RepetitionCountKind = iota
	RepetitionCountKindAutoFit
	RepetitionCountKindCount
)

// RepetitionCount is the number of repetitions of a repeat() clause: auto-fill, auto-fit or a count.
type RepetitionCount struct {
	kind  RepetitionCountKind
	count uint16
}

var (
	// RepetitionCountAutoFill is the AutoFill variant.
	RepetitionCountAutoFill = RepetitionCount{kind: RepetitionCountKindAutoFill}
	// RepetitionCountAutoFit is the AutoFit variant.
	RepetitionCountAutoFit = RepetitionCount{kind: RepetitionCountKindAutoFit}
)

// RepetitionCountCount is the Count variant.
func RepetitionCountCount(count uint16) RepetitionCount {
	return RepetitionCount{kind: RepetitionCountKindCount, count: count}
}

// Kind returns the variant.
func (r RepetitionCount) Kind() RepetitionCountKind {
	return r.kind
}

// Count returns the count of the Count variant.
func (r RepetitionCount) Count() uint16 {
	return r.count
}

// RepetitionCountFromU16 is the Count variant.
func RepetitionCountFromU16(value uint16) RepetitionCount {
	return RepetitionCountCount(value)
}

// InvalidStringRepetitionValue is the error for a string that is neither "auto-fit" nor "auto-fill".
type InvalidStringRepetitionValue struct{}

// Error returns upstream's message.
func (InvalidStringRepetitionValue) Error() string {
	return "&str can only be converted to GridTrackRepetition if it's value is 'auto-fit' or 'auto-fill'"
}

// RepetitionCountTryFromStr converts "auto-fit" or "auto-fill".
func RepetitionCountTryFromStr(value string) (RepetitionCount, error) {
	switch value {
	case "auto-fit":
		return RepetitionCountAutoFit, nil
	case "auto-fill":
		return RepetitionCountAutoFill, nil
	}
	return RepetitionCount{}, InvalidStringRepetitionValue{}
}

// GridTemplateRepetition is a repeat() clause of a grid template.
type GridTemplateRepetition[S CheapCloneStr] struct {
	Count     RepetitionCount
	Tracks    []TrackSizingFunction
	LineNames [][]S
}

// Ref returns the GenericRepetition view of r (upstream's `&GridTemplateRepetition`).
func (r *GridTemplateRepetition[S]) Ref() GridTemplateRepetitionRef[S] {
	return GridTemplateRepetitionRef[S]{repetition: r}
}

// GridTemplateRepetitionRef is a reference to a GridTemplateRepetition implementing GenericRepetition.
type GridTemplateRepetitionRef[S CheapCloneStr] struct {
	repetition *GridTemplateRepetition[S]
}

// Repetition returns the referenced repetition.
func (r GridTemplateRepetitionRef[S]) Repetition() *GridTemplateRepetition[S] {
	return r.repetition
}

// Count returns the repetition count.
func (r GridTemplateRepetitionRef[S]) Count() RepetitionCount {
	return r.repetition.Count
}

// TrackCount returns the number of repeated tracks, saturating at 65535.
func (r GridTemplateRepetitionRef[S]) TrackCount() uint16 {
	return uint16(min(len(r.repetition.Tracks), 0xFFFF))
}

// Tracks returns the repeated tracks.
func (r GridTemplateRepetitionRef[S]) Tracks() []TrackSizingFunction {
	return r.repetition.Tracks
}

// LinesNames returns the line name sets.
func (r GridTemplateRepetitionRef[S]) LinesNames() TemplateLineNames[S] {
	return r.repetition.LineNames
}

var _ GenericRepetition[string] = GridTemplateRepetitionRef[string]{}

// GridTemplateComponentKind is the variant of a GridTemplateComponent.
type GridTemplateComponentKind uint8

const (
	GridTemplateComponentKindSingle GridTemplateComponentKind = iota
	GridTemplateComponentKindRepeat
)

// GridTemplateComponent is a track of a grid template: a single track sizing function or a repeat() clause.
type GridTemplateComponent[S CheapCloneStr] struct {
	kind   GridTemplateComponentKind
	single TrackSizingFunction
	repeat GridTemplateRepetition[S]
}

// GridTemplateComponentSingle is the Single variant.
func GridTemplateComponentSingle[S CheapCloneStr](value TrackSizingFunction) GridTemplateComponent[S] {
	return GridTemplateComponent[S]{kind: GridTemplateComponentKindSingle, single: value}
}

// GridTemplateComponentRepeat is the Repeat variant.
func GridTemplateComponentRepeat[S CheapCloneStr](value GridTemplateRepetition[S]) GridTemplateComponent[S] {
	return GridTemplateComponent[S]{kind: GridTemplateComponentKindRepeat, repeat: value}
}

// Kind returns the variant.
func (c GridTemplateComponent[S]) Kind() GridTemplateComponentKind {
	return c.kind
}

// Single returns the track sizing function of the Single variant.
func (c GridTemplateComponent[S]) Single() TrackSizingFunction {
	return c.single
}

// Repeat returns the repetition of the Repeat variant.
func (c GridTemplateComponent[S]) Repeat() GridTemplateRepetition[S] {
	return c.repeat
}

// AsComponentRef returns the GenericGridTemplateComponent view of c.
func (c *GridTemplateComponent[S]) AsComponentRef() GenericGridTemplateComponent[S, GenericRepetition[S]] {
	switch c.kind {
	case GridTemplateComponentKindSingle:
		return GenericGridTemplateComponentSingle[S, GenericRepetition[S]](c.single)
	}
	return GenericGridTemplateComponentRepeat[S, GenericRepetition[S]](c.repeat.Ref())
}

// IsAutoRepetition reports whether c is an auto-fit or auto-fill repeat().
func (c GridTemplateComponent[S]) IsAutoRepetition() bool {
	if c.kind != GridTemplateComponentKindRepeat {
		return false
	}
	switch c.repeat.Count.kind {
	case RepetitionCountKindAutoFit, RepetitionCountKindAutoFill:
		return true
	}
	return false
}

// Auto returns a single auto track.
func (GridTemplateComponent[S]) Auto() GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](TrackSizingFunctionAuto)
}

// MinContent returns a single min-content track.
func (GridTemplateComponent[S]) MinContent() GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](TrackSizingFunctionMinContent)
}

// MaxContent returns a single max-content track.
func (GridTemplateComponent[S]) MaxContent() GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](TrackSizingFunctionMaxContent)
}

// FitContent returns a single fit-content(argument) track.
func (GridTemplateComponent[S]) FitContent(argument LengthPercentage) GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](TrackSizingFunctionFitContent(argument))
}

// Zero returns a single zero-sized track.
func (GridTemplateComponent[S]) Zero() GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](TrackSizingFunctionZero)
}

// FromLength returns a single track of the length value.
func (GridTemplateComponent[S]) FromLength(value float64) GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](TrackSizingFunction{}.FromLength(value))
}

// FromPercent returns a single track of the percentage.
func (GridTemplateComponent[S]) FromPercent(percent float64) GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](TrackSizingFunction{}.FromPercent(percent))
}

// FromFr returns a single flexible track.
func (GridTemplateComponent[S]) FromFr(flex float64) GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](TrackSizingFunction{}.FromFr(flex))
}

// GridTemplateComponentFromMinMax returns a single track with the given min and max.
func GridTemplateComponentFromMinMax[S CheapCloneStr](input TrackSizingFunction) GridTemplateComponent[S] {
	return GridTemplateComponentSingle[S](input)
}

func (GridTemplateComponent[S]) fromMinMax(input TrackSizingFunction) GridTemplateComponent[S] {
	return GridTemplateComponentFromMinMax[S](input)
}

// GridTemplateTracks is a list of tracks with the line names between them.
type GridTemplateTracks[S CheapCloneStr, Track any] struct {
	Tracks    []Track
	LineNames [][]S
}

// GridTemplateTracksDefault returns an empty track list.
func GridTemplateTracksDefault[S CheapCloneStr, Track any]() GridTemplateTracks[S, Track] {
	return GridTemplateTracks[S, Track]{Tracks: nil, LineNames: nil}
}

// GridAutoTracks is a list of implicit track sizing functions.
type GridAutoTracks []TrackSizingFunction
