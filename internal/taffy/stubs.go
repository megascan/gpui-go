package taffy

// FlexDirection is the direction of the flexbox layout main axis.
type FlexDirection uint8

// IsRow reports whether the direction is a row direction.
func (FlexDirection) IsRow() bool {
	panic("taffy: unimplemented FlexDirection.IsRow")
}

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
