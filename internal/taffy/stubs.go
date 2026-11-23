package taffy

// TaffyZero is implemented by types with a zero value (upstream's TaffyZero::ZERO).
type TaffyZero[T any] interface {
	Zero() T
}

// TaffyAuto is implemented by types with an auto value (upstream's TaffyAuto::AUTO).
type TaffyAuto[T any] interface {
	Auto() T
}

// TaffyMinContent is implemented by types with a min-content value (upstream's TaffyMinContent::MIN_CONTENT).
type TaffyMinContent[T any] interface {
	MinContent() T
}

// TaffyMaxContent is implemented by types with a max-content value (upstream's TaffyMaxContent::MAX_CONTENT).
type TaffyMaxContent[T any] interface {
	MaxContent() T
}

// TaffyFitContent is implemented by types that can be a fit-content value.
type TaffyFitContent[T any] interface {
	FitContent(argument LengthPercentage) T
}

// FromLength is implemented by types that can be built from a length.
type FromLength[T any] interface {
	FromLength(value float64) T
}

// FromPercent is implemented by types that can be built from a percentage.
type FromPercent[T any] interface {
	FromPercent(value float64) T
}

// FromFr is implemented by types that can be built from an fr value.
type FromFr[T any] interface {
	FromFr(value float64) T
}

// GridTemplateComponent is a grid template track: a single track or a repeat() (owner: M0-taffy-style-grid).
type GridTemplateComponent[S CheapCloneStr] struct {
	kind   uint8
	single TrackSizingFunction
	repeat *GridTemplateRepetition[S]
}

// GridTemplateRepetition is a repeat() clause (owner: M0-taffy-style-grid).
type GridTemplateRepetition[S CheapCloneStr] struct {
	Tracks    []TrackSizingFunction
	LineNames [][]S
}

// MinTrackSizingFunction is the minimum track sizing function (owner: M0-taffy-style-grid).
type MinTrackSizingFunction struct {
	inner CompactLength
}

// MaxTrackSizingFunction is the maximum track sizing function (owner: M0-taffy-style-grid).
type MaxTrackSizingFunction struct {
	inner CompactLength
}

// TrackSizingFunction is a track's min and max sizing functions (owner: M0-taffy-style-grid).
type TrackSizingFunction = MinMax[MinTrackSizingFunction, MaxTrackSizingFunction]

// GridAutoFlow controls how auto-placed items are placed (owner: M0-taffy-style-grid).
type GridAutoFlow uint8

const (
	GridAutoFlowRow GridAutoFlow = iota
	GridAutoFlowColumn
	GridAutoFlowRowDense
	GridAutoFlowColumnDense
)

// GridTemplateAreas is the grid-template-areas value (owner: M0-taffy-style-grid).
type GridTemplateAreas[S CheapCloneStr] struct {
	Areas       []GridTemplateArea[S]
	RowCount    uint16
	ColumnCount uint16
}

// GridTemplateArea is one named grid area (owner: M0-taffy-style-grid).
type GridTemplateArea[S CheapCloneStr] struct {
	Name        S
	RowStart    uint16
	RowEnd      uint16
	ColumnStart uint16
	ColumnEnd   uint16
}

// GridPlacement is a grid line placement; the zero value is Auto (owner: M0-taffy-style-grid).
type GridPlacement[S CheapCloneStr] struct {
	kind  uint8
	name  S
	value int32
}
