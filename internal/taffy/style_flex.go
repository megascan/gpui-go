package taffy

// FlexboxContainerStyle is the set of styles required for a Flexbox container.
type FlexboxContainerStyle interface {
	CoreStyle
	FlexDirection() FlexDirection
	FlexWrap() FlexWrap
	Gap() Size[LengthPercentage]
	AlignContent() AlignContent
	AlignItems() AlignItems
	JustifyContent() JustifyContent
}

// FlexboxContainerStyleDefaults provides upstream's default FlexboxContainerStyle methods for embedding.
type FlexboxContainerStyleDefaults struct{}

// FlexDirection returns the default flex direction.
func (FlexboxContainerStyleDefaults) FlexDirection() FlexDirection {
	return StyleDefault[defaultCheapStr]().FlexDirection
}

// FlexWrap returns the default flex wrap.
func (FlexboxContainerStyleDefaults) FlexWrap() FlexWrap {
	return StyleDefault[defaultCheapStr]().FlexWrap
}

// Gap returns the default gap.
func (FlexboxContainerStyleDefaults) Gap() Size[LengthPercentage] {
	return StyleDefault[defaultCheapStr]().Gap
}

// AlignContent returns the default align-content.
func (FlexboxContainerStyleDefaults) AlignContent() AlignContent {
	return StyleDefault[defaultCheapStr]().AlignContent
}

// AlignItems returns the default align-items.
func (FlexboxContainerStyleDefaults) AlignItems() AlignItems {
	return StyleDefault[defaultCheapStr]().AlignItems
}

// JustifyContent returns the default justify-content.
func (FlexboxContainerStyleDefaults) JustifyContent() JustifyContent {
	return StyleDefault[defaultCheapStr]().JustifyContent
}

// FlexboxItemStyle is the set of styles required for a Flexbox item (child of a Flexbox container).
type FlexboxItemStyle interface {
	CoreStyle
	FlexBasis() Dimension
	FlexGrow() float32
	FlexShrink() float32
	AlignSelf() Option[AlignSelf]
}

// FlexboxItemStyleDefaults provides upstream's default FlexboxItemStyle methods for embedding.
type FlexboxItemStyleDefaults struct{}

// FlexBasis returns the default flex basis.
func (FlexboxItemStyleDefaults) FlexBasis() Dimension {
	return StyleDefault[defaultCheapStr]().FlexBasis
}

// FlexGrow returns the default flex grow factor.
func (FlexboxItemStyleDefaults) FlexGrow() float32 {
	return StyleDefault[defaultCheapStr]().FlexGrow
}

// FlexShrink returns the default flex shrink factor.
func (FlexboxItemStyleDefaults) FlexShrink() float32 {
	return StyleDefault[defaultCheapStr]().FlexShrink
}

// AlignSelf returns the default align-self (None).
func (FlexboxItemStyleDefaults) AlignSelf() Option[AlignSelf] {
	return StyleDefault[defaultCheapStr]().AlignSelf
}

// FlexWrap controls whether flex items are forced onto one line or can wrap onto multiple lines.
type FlexWrap uint8

const (
	FlexWrapNoWrap FlexWrap = iota
	FlexWrapWrap
	FlexWrapWrapReverse
)

// FlexWrapDefault is the Default value of FlexWrap.
const FlexWrapDefault = FlexWrapNoWrap

func (w FlexWrap) isMultiLine() bool {
	return w != FlexWrapNoWrap
}

func (w FlexWrap) isReverse() bool {
	return w == FlexWrapWrapReverse
}

// FlexDirection is the direction of the flexbox layout main axis.
type FlexDirection uint8

const (
	FlexDirectionRow FlexDirection = iota
	FlexDirectionColumn
	FlexDirectionRowReverse
	FlexDirectionColumnReverse
)

// FlexDirectionDefault is the Default value of FlexDirection.
const FlexDirectionDefault = FlexDirectionRow

func (d FlexDirection) isRow() bool {
	return d == FlexDirectionRow || d == FlexDirectionRowReverse
}

func (d FlexDirection) isColumn() bool {
	return d == FlexDirectionColumn || d == FlexDirectionColumnReverse
}

func (d FlexDirection) isReverse() bool {
	return d == FlexDirectionRowReverse || d == FlexDirectionColumnReverse
}

func (d FlexDirection) mainAxis() AbsoluteAxis {
	switch d {
	case FlexDirectionRow, FlexDirectionRowReverse:
		return AbsoluteAxisHorizontal
	default:
		return AbsoluteAxisVertical
	}
}

func (d FlexDirection) crossAxis() AbsoluteAxis {
	switch d {
	case FlexDirectionRow, FlexDirectionRowReverse:
		return AbsoluteAxisVertical
	default:
		return AbsoluteAxisHorizontal
	}
}
