package taffy

// CheapCloneStr is the constraint on the custom identifier type used for named grid lines and areas.
type CheapCloneStr interface {
	~string
}

// CoreStyle is the core set of styles that are shared between all CSS layout nodes.
type CoreStyle interface {
	BoxGenerationMode() BoxGenerationMode
	IsBlock() bool
	IsCompressibleReplaced() bool
	IsReplaced() bool
	BoxSizing() BoxSizing
	Direction() Direction
	Overflow() Point[Overflow]
	ScrollbarWidth() float32
	Position() Position
	Inset() Rect[LengthPercentageAuto]
	IsContainingBlock() ContainingBlockClaims
	Size() Size[Dimension]
	MinSize() Size[LengthPercentageAuto]
	MaxSize() Size[LengthPercentageAuto]
	AspectRatio() Option[float32]
	Margin() Rect[LengthPercentageAuto]
	Padding() Rect[LengthPercentage]
	Border() Rect[LengthPercentage]
	Contain() Contain
}

// CoreStyleDefaults provides upstream's default CoreStyle methods for embedding, except IsContainingBlock, whose default depends on Position (see CoreStyleDefaultIsContainingBlock).
type CoreStyleDefaults struct{}

// BoxGenerationMode returns BoxGenerationModeDefault.
func (CoreStyleDefaults) BoxGenerationMode() BoxGenerationMode {
	return BoxGenerationModeDefault
}

// IsBlock returns false.
func (CoreStyleDefaults) IsBlock() bool {
	return false
}

// IsCompressibleReplaced returns false.
func (CoreStyleDefaults) IsCompressibleReplaced() bool {
	return false
}

// IsReplaced returns false.
func (CoreStyleDefaults) IsReplaced() bool {
	return false
}

// BoxSizing returns BoxSizingBorderBox.
func (CoreStyleDefaults) BoxSizing() BoxSizing {
	return BoxSizingBorderBox
}

// Direction returns DirectionLtr.
func (CoreStyleDefaults) Direction() Direction {
	return DirectionLtr
}

// Overflow returns the default overflow.
func (CoreStyleDefaults) Overflow() Point[Overflow] {
	return StyleDefault[defaultCheapStr]().Overflow
}

// ScrollbarWidth returns 0.
func (CoreStyleDefaults) ScrollbarWidth() float32 {
	return 0.0
}

// Position returns the default position.
func (CoreStyleDefaults) Position() Position {
	return StyleDefault[defaultCheapStr]().Position
}

// Inset returns the default inset.
func (CoreStyleDefaults) Inset() Rect[LengthPercentageAuto] {
	return StyleDefault[defaultCheapStr]().Inset
}

// Size returns the default size.
func (CoreStyleDefaults) Size() Size[Dimension] {
	return StyleDefault[defaultCheapStr]().Size
}

// MinSize returns the default min size.
func (CoreStyleDefaults) MinSize() Size[LengthPercentageAuto] {
	return StyleDefault[defaultCheapStr]().MinSize
}

// MaxSize returns the default max size.
func (CoreStyleDefaults) MaxSize() Size[LengthPercentageAuto] {
	return StyleDefault[defaultCheapStr]().MaxSize
}

// AspectRatio returns the default aspect ratio (None).
func (CoreStyleDefaults) AspectRatio() Option[float32] {
	return StyleDefault[defaultCheapStr]().AspectRatio
}

// Margin returns the default margin.
func (CoreStyleDefaults) Margin() Rect[LengthPercentageAuto] {
	return StyleDefault[defaultCheapStr]().Margin
}

// Padding returns the default padding.
func (CoreStyleDefaults) Padding() Rect[LengthPercentage] {
	return StyleDefault[defaultCheapStr]().Padding
}

// Border returns the default border.
func (CoreStyleDefaults) Border() Rect[LengthPercentage] {
	return StyleDefault[defaultCheapStr]().Border
}

// Contain returns ContainNone.
func (CoreStyleDefaults) Contain() Contain {
	return ContainNone
}

// CoreStyleDefaultIsContainingBlock is upstream's default CoreStyle::is_containing_block.
func CoreStyleDefaultIsContainingBlock(s CoreStyle) ContainingBlockClaims {
	return ContainingBlockClaims{Absolute: s.Position().IsPositioned(), Fixed: false}
}

// OofItemStyle is the styles of an out-of-flow box read by its containing block's out-of-flow positioning pass.
type OofItemStyle[S CheapCloneStr] interface {
	CoreStyle
	AlignSelf() Option[AlignSelf]
	JustifySelf() Option[AlignSelf]
	IsTable() bool
	GridRow() Line[GridPlacement[S]]
	GridColumn() Line[GridPlacement[S]]
}

// OofItemStyleDefaults provides upstream's default OofItemStyle methods for embedding.
type OofItemStyleDefaults[S CheapCloneStr] struct{}

// AlignSelf returns None.
func (OofItemStyleDefaults[S]) AlignSelf() Option[AlignSelf] {
	return none[AlignSelf]()
}

// JustifySelf returns None.
func (OofItemStyleDefaults[S]) JustifySelf() Option[AlignSelf] {
	return none[AlignSelf]()
}

// IsTable returns false.
func (OofItemStyleDefaults[S]) IsTable() bool {
	return false
}

// GridRow returns the default grid row placement.
func (OofItemStyleDefaults[S]) GridRow() Line[GridPlacement[S]] {
	return Line[GridPlacement[S]]{}
}

// GridColumn returns the default grid column placement.
func (OofItemStyleDefaults[S]) GridColumn() Line[GridPlacement[S]] {
	return Line[GridPlacement[S]]{}
}

// Display sets the layout used for the children of this node.
type Display uint8

const (
	DisplayBlock Display = iota
	DisplayFlowRoot
	DisplayFlex
	DisplayGrid
	DisplayNone
)

// DisplayDefault is the default Display mode.
const DisplayDefault = DisplayFlex

// String formats the display mode as upstream's Display impl.
func (d Display) String() string {
	switch d {
	case DisplayNone:
		return "NONE"
	case DisplayBlock:
		return "BLOCK"
	case DisplayFlowRoot:
		return "FLOW-ROOT"
	case DisplayFlex:
		return "FLEX"
	case DisplayGrid:
		return "GRID"
	}
	panic("internal error: entered unreachable code: invalid Display")
}

// BoxGenerationMode is an abstracted version of the CSS display property where any value other than "none" is represented by "normal".
type BoxGenerationMode uint8

const (
	BoxGenerationModeNormal BoxGenerationMode = iota
	BoxGenerationModeNone
)

// BoxGenerationModeDefault is the default of BoxGenerationMode.
const BoxGenerationModeDefault = BoxGenerationModeNormal

// Position is the positioning strategy for this item.
type Position uint8

const (
	PositionStatic Position = iota
	PositionRelative
	PositionAbsolute
	PositionFixed
	PositionSticky
)

// PositionDefault is the Default value of Position.
const PositionDefault = PositionStatic

// IsOutOfFlow reports whether this position takes the item out of normal flow (Absolute or Fixed).
func (p Position) IsOutOfFlow() bool {
	return p == PositionAbsolute || p == PositionFixed
}

// IsPositioned reports whether this position makes the item a containing block for absolutely positioned descendants.
func (p Position) IsPositioned() bool {
	return p != PositionStatic
}

// ContainingBlockClaims is which out-of-flow positions a node establishes a containing block for.
type ContainingBlockClaims struct {
	Absolute bool
	Fixed    bool
}

var (
	// ContainingBlockClaimsNone claims nothing.
	ContainingBlockClaimsNone = ContainingBlockClaims{Absolute: false, Fixed: false}
	// ContainingBlockClaimsAll claims every out-of-flow box (the initial containing block).
	ContainingBlockClaimsAll = ContainingBlockClaims{Absolute: true, Fixed: true}
)

// ForPosition reports whether the node is the containing block for a box with the given position.
func (c ContainingBlockClaims) ForPosition(position Position) bool {
	switch position {
	case PositionAbsolute:
		return c.Absolute
	case PositionFixed:
		return c.Fixed
	}
	return false
}

// BoxSizing specifies whether size styles for this node are assigned to the node's content box or border box.
type BoxSizing uint8

const (
	BoxSizingBorderBox BoxSizing = iota
	BoxSizingContentBox
)

// BoxSizingDefault is the Default value of BoxSizing.
const BoxSizingDefault = BoxSizingBorderBox

// Overflow is how children overflowing their container should affect layout.
type Overflow uint8

const (
	OverflowVisible Overflow = iota
	OverflowClip
	OverflowHidden
	OverflowScroll
)

// OverflowDefault is the Default value of Overflow.
const OverflowDefault = OverflowVisible

// IsScrollContainer reports whether the overflow mode contains its contents (Hidden, Scroll).
func (o Overflow) IsScrollContainer() bool {
	switch o {
	case OverflowVisible, OverflowClip:
		return false
	}
	return true
}

func (o Overflow) maybeIntoAutomaticMinSize() Option[float32] {
	if o.IsScrollContainer() {
		return some[float32](0.0)
	}
	return none[float32]()
}

// Contain is the layout-affecting parts of the CSS contain property.
type Contain uint8

const (
	// ContainNone is no containment (the default).
	ContainNone Contain = 0
	// ContainLayout is layout containment.
	ContainLayout Contain = 1 << 0
	// ContainPaint is paint containment.
	ContainPaint Contain = 1 << 1
	// ContainContent is the containment implied by `contain: content`.
	ContainContent Contain = ContainLayout | ContainPaint
	// ContainDefault is the default containment (no containment).
	ContainDefault Contain = ContainNone
)

// Contains reports whether c contains all of the containment types in other.
func (c Contain) Contains(other Contain) bool {
	return c&other == other
}

// Intersects reports whether c contains any of the containment types in other.
func (c Contain) Intersects(other Contain) bool {
	return c&other != 0
}

// Union returns the union of the containment types in c and other.
func (c Contain) Union(other Contain) Contain {
	return c | other
}

// EstablishesIndependentFormattingContext reports whether this containment makes the box establish an independent formatting context.
func (c Contain) EstablishesIndependentFormattingContext() bool {
	return c.Intersects(ContainLayout.Union(ContainPaint))
}

// SuppressesBaseline reports whether this containment suppresses the box's baseline for baseline-alignment purposes.
func (c Contain) SuppressesBaseline() bool {
	return c.Contains(ContainLayout)
}

// ContainsScrollableOverflow reports whether this containment keeps the box's overflowing content out of an ancestor's scrollable overflow region.
func (c Contain) ContainsScrollableOverflow() bool {
	return c.Intersects(ContainLayout.Union(ContainPaint))
}

// Direction sets the direction of text, table and grid columns, and horizontal overflow.
type Direction uint8

const (
	DirectionLtr Direction = iota
	DirectionRtl
)

// DirectionDefault is the Default value of Direction.
const DirectionDefault = DirectionLtr

// IsRtl reports whether the direction is right-to-left.
func (d Direction) IsRtl() bool {
	return d == DirectionRtl
}

// Style is a typed representation of the CSS style information for a single node.
type Style[S CheapCloneStr] struct {
	Display                    Display
	ItemIsTable                bool
	ItemIsReplaced             bool
	ItemIsCompressibleReplaced bool
	BoxSizing                  BoxSizing
	Direction                  Direction
	Overflow                   Point[Overflow]
	ScrollbarWidth             float32
	Contain                    Contain
	Float                      Float
	Clear                      Clear
	Position                   Position
	Inset                      Rect[LengthPercentageAuto]
	Size                       Size[Dimension]
	MinSize                    Size[LengthPercentageAuto]
	MaxSize                    Size[LengthPercentageAuto]
	AspectRatio                Option[float32]
	Margin                     Rect[LengthPercentageAuto]
	Padding                    Rect[LengthPercentage]
	Border                     Rect[LengthPercentage]
	AlignItems                 AlignItems
	AlignSelf                  Option[AlignSelf]
	JustifyItems               AlignItems
	JustifySelf                Option[AlignSelf]
	AlignContent               AlignContent
	JustifyContent             JustifyContent
	Gap                        Size[LengthPercentage]
	TextAlign                  TextAlign
	FlexDirection              FlexDirection
	FlexWrap                   FlexWrap
	FlexBasis                  Dimension
	FlexGrow                   float32
	FlexShrink                 float32
	GridTemplateRows           []GridTemplateComponent[S]
	GridTemplateColumns        []GridTemplateComponent[S]
	GridAutoRows               []TrackSizingFunction
	GridAutoColumns            []TrackSizingFunction
	GridAutoFlow               GridAutoFlow
	GridTemplateAreas          Option[GridTemplateAreas[S]]
	GridTemplateColumnNames    [][]S
	GridTemplateRowNames       [][]S
	GridRow                    Line[GridPlacement[S]]
	GridColumn                 Line[GridPlacement[S]]
}

// StyleDefault returns the default layout (upstream's Style::DEFAULT and Default impl).
func StyleDefault[S CheapCloneStr]() Style[S] {
	lpaAuto := LengthPercentageAutoAuto()
	lpaZero := LengthPercentageAutoZero
	lpZero := LengthPercentageZero
	dimAuto := DimensionAuto()
	return Style[S]{
		Display:                    DisplayDefault,
		ItemIsTable:                false,
		ItemIsReplaced:             false,
		ItemIsCompressibleReplaced: false,
		BoxSizing:                  BoxSizingBorderBox,
		Direction:                  DirectionLtr,
		Overflow:                   Point[Overflow]{X: OverflowVisible, Y: OverflowVisible},
		ScrollbarWidth:             0.0,
		Contain:                    ContainNone,
		Float:                      FloatNone,
		Clear:                      ClearNone,
		Position:                   PositionStatic,
		Inset:                      Rect[LengthPercentageAuto]{Left: lpaAuto, Right: lpaAuto, Top: lpaAuto, Bottom: lpaAuto},
		Margin:                     Rect[LengthPercentageAuto]{Left: lpaZero, Right: lpaZero, Top: lpaZero, Bottom: lpaZero},
		Padding:                    Rect[LengthPercentage]{Left: lpZero, Right: lpZero, Top: lpZero, Bottom: lpZero},
		Border:                     Rect[LengthPercentage]{Left: lpZero, Right: lpZero, Top: lpZero, Bottom: lpZero},
		Size:                       Size[Dimension]{Width: dimAuto, Height: dimAuto},
		MinSize:                    Size[LengthPercentageAuto]{Width: lpaAuto, Height: lpaAuto},
		MaxSize:                    Size[LengthPercentageAuto]{Width: lpaAuto, Height: lpaAuto},
		AspectRatio:                none[float32](),
		Gap:                        Size[LengthPercentage]{Width: lpZero, Height: lpZero},
		AlignItems:                 AlignItemsNormal,
		AlignSelf:                  none[AlignSelf](),
		JustifyItems:               AlignItemsNormal,
		JustifySelf:                none[AlignSelf](),
		AlignContent:               AlignContentNormal,
		JustifyContent:             AlignContentNormal,
		TextAlign:                  TextAlignAuto,
		FlexDirection:              FlexDirectionRow,
		FlexWrap:                   FlexWrapNoWrap,
		FlexGrow:                   0.0,
		FlexShrink:                 1.0,
		FlexBasis:                  dimAuto,
		GridTemplateRows:           nil,
		GridTemplateColumns:        nil,
		GridTemplateAreas:          none[GridTemplateAreas[S]](),
		GridTemplateColumnNames:    nil,
		GridTemplateRowNames:       nil,
		GridAutoRows:               nil,
		GridAutoColumns:            nil,
		GridAutoFlow:               GridAutoFlowRow,
		GridRow:                    Line[GridPlacement[S]]{},
		GridColumn:                 Line[GridPlacement[S]]{},
	}
}

// Ref returns the style trait view of s (upstream's `&Style`).
func (s *Style[S]) Ref() StyleRef[S] {
	return StyleRef[S]{style: s}
}

// StyleRef is a reference to a Style implementing the style traits (CoreStyle, OofItemStyle, BlockContainerStyle, BlockItemStyle, FlexboxContainerStyle, FlexboxItemStyle, GridContainerStyle, GridItemStyle).
type StyleRef[S CheapCloneStr] struct {
	style *Style[S]
}

// Style returns the referenced style.
func (r StyleRef[S]) Style() *Style[S] {
	return r.style
}

// BoxGenerationMode returns None for DisplayNone, Normal otherwise.
func (r StyleRef[S]) BoxGenerationMode() BoxGenerationMode {
	switch r.style.Display {
	case DisplayNone:
		return BoxGenerationModeNone
	}
	return BoxGenerationModeNormal
}

// IsBlock reports whether the display is DisplayBlock.
func (r StyleRef[S]) IsBlock() bool {
	return r.style.Display == DisplayBlock
}

// IsCompressibleReplaced reports whether the item is replaced or compressible replaced.
func (r StyleRef[S]) IsCompressibleReplaced() bool {
	return r.style.ItemIsReplaced || r.style.ItemIsCompressibleReplaced
}

// IsReplaced returns ItemIsReplaced.
func (r StyleRef[S]) IsReplaced() bool {
	return r.style.ItemIsReplaced
}

// BoxSizing returns the BoxSizing field.
func (r StyleRef[S]) BoxSizing() BoxSizing {
	return r.style.BoxSizing
}

// Direction returns the Direction field.
func (r StyleRef[S]) Direction() Direction {
	return r.style.Direction
}

// Overflow returns the Overflow field.
func (r StyleRef[S]) Overflow() Point[Overflow] {
	return r.style.Overflow
}

// ScrollbarWidth returns the ScrollbarWidth field.
func (r StyleRef[S]) ScrollbarWidth() float32 {
	return r.style.ScrollbarWidth
}

// Position returns the Position field.
func (r StyleRef[S]) Position() Position {
	return r.style.Position
}

// Inset returns the Inset field.
func (r StyleRef[S]) Inset() Rect[LengthPercentageAuto] {
	return r.style.Inset
}

// IsContainingBlock is upstream's default CoreStyle::is_containing_block.
func (r StyleRef[S]) IsContainingBlock() ContainingBlockClaims {
	return CoreStyleDefaultIsContainingBlock(r)
}

// Size returns the Size field.
func (r StyleRef[S]) Size() Size[Dimension] {
	return r.style.Size
}

// MinSize returns the MinSize field.
func (r StyleRef[S]) MinSize() Size[LengthPercentageAuto] {
	return r.style.MinSize
}

// MaxSize returns the MaxSize field.
func (r StyleRef[S]) MaxSize() Size[LengthPercentageAuto] {
	return r.style.MaxSize
}

// AspectRatio returns the AspectRatio field.
func (r StyleRef[S]) AspectRatio() Option[float32] {
	return r.style.AspectRatio
}

// Margin returns the Margin field.
func (r StyleRef[S]) Margin() Rect[LengthPercentageAuto] {
	return r.style.Margin
}

// Padding returns the Padding field.
func (r StyleRef[S]) Padding() Rect[LengthPercentage] {
	return r.style.Padding
}

// Border returns the Border field.
func (r StyleRef[S]) Border() Rect[LengthPercentage] {
	return r.style.Border
}

// Contain returns the Contain field.
func (r StyleRef[S]) Contain() Contain {
	return r.style.Contain
}

// AlignSelf returns the AlignSelf field.
func (r StyleRef[S]) AlignSelf() Option[AlignSelf] {
	return r.style.AlignSelf
}

// JustifySelf returns the JustifySelf field.
func (r StyleRef[S]) JustifySelf() Option[AlignSelf] {
	return r.style.JustifySelf
}

// IsTable returns the ItemIsTable field.
func (r StyleRef[S]) IsTable() bool {
	return r.style.ItemIsTable
}

// GridRow returns the GridRow field.
func (r StyleRef[S]) GridRow() Line[GridPlacement[S]] {
	return r.style.GridRow
}

// GridColumn returns the GridColumn field.
func (r StyleRef[S]) GridColumn() Line[GridPlacement[S]] {
	return r.style.GridColumn
}

// TextAlign returns the TextAlign field.
func (r StyleRef[S]) TextAlign() TextAlign {
	return r.style.TextAlign
}

// AlignContent returns the AlignContent field.
func (r StyleRef[S]) AlignContent() AlignContent {
	return r.style.AlignContent
}

// JustifyItems returns the JustifyItems field.
func (r StyleRef[S]) JustifyItems() AlignItems {
	return r.style.JustifyItems
}

// Float returns the Float field.
func (r StyleRef[S]) Float() Float {
	return r.style.Float
}

// Clear returns the Clear field.
func (r StyleRef[S]) Clear() Clear {
	return r.style.Clear
}

// FlexDirection returns the FlexDirection field.
func (r StyleRef[S]) FlexDirection() FlexDirection {
	return r.style.FlexDirection
}

// FlexWrap returns the FlexWrap field.
func (r StyleRef[S]) FlexWrap() FlexWrap {
	return r.style.FlexWrap
}

// Gap returns the Gap field.
func (r StyleRef[S]) Gap() Size[LengthPercentage] {
	return r.style.Gap
}

// AlignItems returns the AlignItems field.
func (r StyleRef[S]) AlignItems() AlignItems {
	return r.style.AlignItems
}

// JustifyContent returns the JustifyContent field.
func (r StyleRef[S]) JustifyContent() JustifyContent {
	return r.style.JustifyContent
}

// FlexBasis returns the FlexBasis field.
func (r StyleRef[S]) FlexBasis() Dimension {
	return r.style.FlexBasis
}

// FlexGrow returns the FlexGrow field.
func (r StyleRef[S]) FlexGrow() float32 {
	return r.style.FlexGrow
}

// FlexShrink returns the FlexShrink field.
func (r StyleRef[S]) FlexShrink() float32 {
	return r.style.FlexShrink
}

func gridTemplateComponentRefs[S CheapCloneStr](components []GridTemplateComponent[S]) []GenericGridTemplateComponent[S, GenericRepetition[S]] {
	out := make([]GenericGridTemplateComponent[S, GenericRepetition[S]], len(components))
	for i := range components {
		out[i] = components[i].AsComponentRef()
	}
	return out
}

// GridTemplateRows returns the GridTemplateRows field as component refs.
func (r StyleRef[S]) GridTemplateRows() Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]] {
	return some(gridTemplateComponentRefs(r.style.GridTemplateRows))
}

// GridTemplateColumns returns the GridTemplateColumns field as component refs.
func (r StyleRef[S]) GridTemplateColumns() Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]] {
	return some(gridTemplateComponentRefs(r.style.GridTemplateColumns))
}

// GridAutoRows returns the GridAutoRows field.
func (r StyleRef[S]) GridAutoRows() []TrackSizingFunction {
	return r.style.GridAutoRows
}

// GridAutoColumns returns the GridAutoColumns field.
func (r StyleRef[S]) GridAutoColumns() []TrackSizingFunction {
	return r.style.GridAutoColumns
}

// GridAutoFlow returns the GridAutoFlow field.
func (r StyleRef[S]) GridAutoFlow() GridAutoFlow {
	return r.style.GridAutoFlow
}

// GridTemplateAreas returns the areas of the GridTemplateAreas field.
func (r StyleRef[S]) GridTemplateAreas() Option[[]GridTemplateArea[S]] {
	if template, ok := r.style.GridTemplateAreas.Get(); ok {
		return some(template.Areas)
	}
	return none[[]GridTemplateArea[S]]()
}

// GridTemplateAreaRowCount returns the row count of the GridTemplateAreas field, or 0.
func (r StyleRef[S]) GridTemplateAreaRowCount() uint16 {
	if template, ok := r.style.GridTemplateAreas.Get(); ok {
		return template.RowCount
	}
	return 0
}

// GridTemplateAreaColumnCount returns the column count of the GridTemplateAreas field, or 0.
func (r StyleRef[S]) GridTemplateAreaColumnCount() uint16 {
	if template, ok := r.style.GridTemplateAreas.Get(); ok {
		return template.ColumnCount
	}
	return 0
}

// GridTemplateColumnNames returns the GridTemplateColumnNames field.
func (r StyleRef[S]) GridTemplateColumnNames() Option[TemplateLineNames[S]] {
	return some(r.style.GridTemplateColumnNames)
}

// GridTemplateRowNames returns the GridTemplateRowNames field.
func (r StyleRef[S]) GridTemplateRowNames() Option[TemplateLineNames[S]] {
	return some(r.style.GridTemplateRowNames)
}

// GridTemplateTracks is upstream's default GridContainerStyle::grid_template_tracks.
func (r StyleRef[S]) GridTemplateTracks(axis AbsoluteAxis) Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]] {
	return GridContainerStyleDefaultGridTemplateTracks[S](r, axis)
}

// GridAlignContent is upstream's default GridContainerStyle::grid_align_content.
func (r StyleRef[S]) GridAlignContent(axis AbstractAxis) AlignContent {
	return GridContainerStyleDefaultGridAlignContent[S](r, axis)
}

// GridPlacement is upstream's default GridItemStyle::grid_placement.
func (r StyleRef[S]) GridPlacement(axis AbsoluteAxis) Line[GridPlacement[S]] {
	return GridItemStyleDefaultGridPlacement[S](r, axis)
}

var (
	_ OofItemStyle[string]       = StyleRef[string]{}
	_ BlockContainerStyle        = StyleRef[string]{}
	_ BlockItemStyle             = StyleRef[string]{}
	_ FlexboxContainerStyle      = StyleRef[string]{}
	_ FlexboxItemStyle           = StyleRef[string]{}
	_ GridContainerStyle[string] = StyleRef[string]{}
	_ GridItemStyle[string]      = StyleRef[string]{}
)
