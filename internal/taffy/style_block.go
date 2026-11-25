package taffy

// BlockContainerStyle is the set of styles required for a Block layout container.
type BlockContainerStyle interface {
	CoreStyle
	TextAlign() TextAlign
	AlignContent() AlignContent
	JustifyItems() AlignItems
}

// BlockContainerStyleDefaults provides upstream's default BlockContainerStyle methods for embedding.
type BlockContainerStyleDefaults struct{}

// TextAlign returns the default text-align.
func (BlockContainerStyleDefaults) TextAlign() TextAlign {
	return StyleDefault[defaultCheapStr]().TextAlign
}

// AlignContent returns the default align-content.
func (BlockContainerStyleDefaults) AlignContent() AlignContent {
	return StyleDefault[defaultCheapStr]().AlignContent
}

// JustifyItems returns the default justify-items.
func (BlockContainerStyleDefaults) JustifyItems() AlignItems {
	return StyleDefault[defaultCheapStr]().JustifyItems
}

// BlockItemStyle is the set of styles required for a Block layout item (child of a Block container).
type BlockItemStyle interface {
	CoreStyle
	IsTable() bool
	AlignSelf() Option[AlignSelf]
	JustifySelf() Option[AlignSelf]
	AlignContent() AlignContent
	Float() Float
	Clear() Clear
}

// BlockItemStyleDefaults provides upstream's default BlockItemStyle methods for embedding.
type BlockItemStyleDefaults struct{}

// IsTable returns false.
func (BlockItemStyleDefaults) IsTable() bool {
	return false
}

// AlignSelf returns None.
func (BlockItemStyleDefaults) AlignSelf() Option[AlignSelf] {
	return none[AlignSelf]()
}

// JustifySelf returns None.
func (BlockItemStyleDefaults) JustifySelf() Option[AlignSelf] {
	return none[AlignSelf]()
}

// AlignContent returns AlignContentNormal.
func (BlockItemStyleDefaults) AlignContent() AlignContent {
	return AlignContentNormal
}

// Float returns FloatNone.
func (BlockItemStyleDefaults) Float() Float {
	return FloatNone
}

// Clear returns ClearNone.
func (BlockItemStyleDefaults) Clear() Clear {
	return ClearNone
}

// TextAlign is used by block layout to implement the legacy behaviour of `<center>` and `<div align="left | right | center">`.
type TextAlign uint8

const (
	TextAlignAuto TextAlign = iota
	TextAlignLegacyLeft
	TextAlignLegacyRight
	TextAlignLegacyCenter
)

// TextAlignDefault is the Default value of TextAlign.
const TextAlignDefault = TextAlignAuto
