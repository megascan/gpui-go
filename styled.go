package gpui

//go:generate go run github.com/megascan/gpui-go/internal/cmd/stylegen -out styled_gen.go -pkg gpui

const ellipsis SharedString = "…"

// Styler is implemented by elements that embed Styled: Style returns the element's style memory.
type Styler interface {
	Style() *StyleRefinement
}

// Styled is the fluent styling mixin (upstream's Styled trait); bind it with BindStyled.
type Styled[S Styler] struct {
	self S
}

// BindStyled binds a Styled mixin to the element that embeds it.
func BindStyled[S Styler](m *Styled[S], self S) {
	m.self = self
}

// Block sets the display type of the element to block. [Docs](https://tailwindcss.com/docs/display)
func (s *Styled[S]) Block() S {
	s.self.Style().Display = Some(DisplayBlock)
	return s.self
}

// Flex sets the display type of the element to flex. [Docs](https://tailwindcss.com/docs/display)
func (s *Styled[S]) Flex() S {
	s.self.Style().Display = Some(DisplayFlex)
	return s.self
}

// Grid sets the display type of the element to grid. [Docs](https://tailwindcss.com/docs/display)
func (s *Styled[S]) Grid() S {
	s.self.Style().Display = Some(DisplayGrid)
	return s.self
}

// Hidden sets the display type of the element to none. [Docs](https://tailwindcss.com/docs/display)
func (s *Styled[S]) Hidden() S {
	s.self.Style().Display = Some(DisplayNone)
	return s.self
}

// ScrollbarWidth sets the space to be reserved for rendering the scrollbar.
//
// This only affects the layout of the element when its overflow is Overflow::Scroll.
func (s *Styled[S]) ScrollbarWidth(width IntoAbsoluteLength) S {
	s.self.Style().ScrollbarWidth = Some(width.ToAbsoluteLength())
	return s.self
}

// WhitespaceNormal sets the whitespace of the element to normal. [Docs](https://tailwindcss.com/docs/whitespace#normal)
func (s *Styled[S]) WhitespaceNormal() S {
	s.TextStyle().WhiteSpace = Some(WhiteSpaceNormal)
	return s.self
}

// WhitespaceNowrap sets the whitespace of the element to nowrap. [Docs](https://tailwindcss.com/docs/whitespace#nowrap)
func (s *Styled[S]) WhitespaceNowrap() S {
	s.TextStyle().WhiteSpace = Some(WhiteSpaceNowrap)
	return s.self
}

// TextEllipsis truncates overflowing text with an ellipsis (…) at the end if needed. [Docs](https://tailwindcss.com/docs/text-overflow#ellipsis)
func (s *Styled[S]) TextEllipsis() S {
	s.TextStyle().TextOverflow = Some(TextOverflowTruncate(ellipsis))
	return s.self
}

// TextEllipsisStart truncates overflowing text with an ellipsis (…) at the start if needed.
// Typically more adequate for file paths where the end is more important than the beginning.
// Note: This doesn't exist in Tailwind CSS.
func (s *Styled[S]) TextEllipsisStart() S {
	s.TextStyle().TextOverflow = Some(TextOverflowTruncateStart(ellipsis))
	return s.self
}

// TextEllipsisMiddle truncates overflowing text with an ellipsis (…) in the middle if needed.
// Preserves the beginning and end of the text. Useful for filenames.
// Note: This doesn't exist in Tailwind CSS.
func (s *Styled[S]) TextEllipsisMiddle() S {
	s.TextStyle().TextOverflow = Some(TextOverflowTruncateMiddle(ellipsis))
	return s.self
}

// TextOverflow sets the text overflow behavior of the element.
func (s *Styled[S]) TextOverflow(overflow TextOverflow) S {
	s.TextStyle().TextOverflow = Some(overflow)
	return s.self
}

// TextAlign sets the text alignment of the element.
func (s *Styled[S]) TextAlign(align TextAlign) S {
	s.TextStyle().TextAlign = Some(align)
	return s.self
}

// TextLeft sets the text alignment to left.
func (s *Styled[S]) TextLeft() S {
	return s.TextAlign(TextAlignLeft)
}

// TextCenter sets the text alignment to center.
func (s *Styled[S]) TextCenter() S {
	return s.TextAlign(TextAlignCenter)
}

// TextRight sets the text alignment to right.
func (s *Styled[S]) TextRight() S {
	return s.TextAlign(TextAlignRight)
}

// Truncate prevents text from wrapping and truncates overflowing text with an ellipsis (…) if needed. [Docs](https://tailwindcss.com/docs/text-overflow#truncate)
func (s *Styled[S]) Truncate() S {
	s.OverflowHidden()
	s.WhitespaceNowrap()
	return s.TextEllipsis()
}

// LineClamp sets the number of lines to show before truncating the text. [Docs](https://tailwindcss.com/docs/line-clamp)
func (s *Styled[S]) LineClamp(lines int) S {
	textStyle := s.TextStyle()
	textStyle.LineClamp = Some(lines)
	return s.OverflowHidden()
}

// FlexCol sets the flex direction of the element to column. [Docs](https://tailwindcss.com/docs/flex-direction#column)
func (s *Styled[S]) FlexCol() S {
	s.self.Style().FlexDirection = Some(FlexDirectionColumn)
	return s.self
}

// FlexColReverse sets the flex direction of the element to column-reverse. [Docs](https://tailwindcss.com/docs/flex-direction#column-reverse)
func (s *Styled[S]) FlexColReverse() S {
	s.self.Style().FlexDirection = Some(FlexDirectionColumnReverse)
	return s.self
}

// FlexRow sets the flex direction of the element to row. [Docs](https://tailwindcss.com/docs/flex-direction#row)
func (s *Styled[S]) FlexRow() S {
	s.self.Style().FlexDirection = Some(FlexDirectionRow)
	return s.self
}

// FlexRowReverse sets the flex direction of the element to row-reverse. [Docs](https://tailwindcss.com/docs/flex-direction#row-reverse)
func (s *Styled[S]) FlexRowReverse() S {
	s.self.Style().FlexDirection = Some(FlexDirectionRowReverse)
	return s.self
}

// Flex1 lets a flex item grow and shrink as needed, ignoring its initial size. [Docs](https://tailwindcss.com/docs/flex#flex-1)
func (s *Styled[S]) Flex1() S {
	s.self.Style().FlexGrow = Some[float32](1.)
	s.self.Style().FlexShrink = Some[float32](1.)
	s.self.Style().FlexBasis = Some(Relative(0.).ToLength())
	return s.self
}

// FlexAuto lets a flex item grow and shrink, taking into account its initial size. [Docs](https://tailwindcss.com/docs/flex#auto)
func (s *Styled[S]) FlexAuto() S {
	s.self.Style().FlexGrow = Some[float32](1.)
	s.self.Style().FlexShrink = Some[float32](1.)
	s.self.Style().FlexBasis = Some(LengthAuto())
	return s.self
}

// FlexInitial lets a flex item shrink but not grow, taking into account its initial size. [Docs](https://tailwindcss.com/docs/flex#initial)
func (s *Styled[S]) FlexInitial() S {
	s.self.Style().FlexGrow = Some[float32](0.)
	s.self.Style().FlexShrink = Some[float32](1.)
	s.self.Style().FlexBasis = Some(LengthAuto())
	return s.self
}

// FlexNone prevents a flex item from growing or shrinking. [Docs](https://tailwindcss.com/docs/flex#none)
func (s *Styled[S]) FlexNone() S {
	s.self.Style().FlexGrow = Some[float32](0.)
	s.self.Style().FlexShrink = Some[float32](0.)
	s.self.Style().FlexBasis = Some(LengthAuto())
	return s.self
}

// FlexBasis sets the initial size of flex items for this element. [Docs](https://tailwindcss.com/docs/flex-basis)
func (s *Styled[S]) FlexBasis(basis IntoLength) S {
	s.self.Style().FlexBasis = Some(basis.ToLength())
	return s.self
}

// FlexGrow sets the flex item's grow factor. [Docs](https://tailwindcss.com/docs/flex-grow)
func (s *Styled[S]) FlexGrow(grow float32) S {
	s.self.Style().FlexGrow = Some(grow)
	return s.self
}

// FlexGrow0 disables flex item growth (flex-grow: 0). [Docs](https://tailwindcss.com/docs/flex-grow#dont-grow)
func (s *Styled[S]) FlexGrow0() S {
	s.self.Style().FlexGrow = Some[float32](0.)
	return s.self
}

// FlexGrow1 enables flex item growth (flex-grow: 1). [Docs](https://tailwindcss.com/docs/flex-grow#grow-1)
func (s *Styled[S]) FlexGrow1() S {
	s.self.Style().FlexGrow = Some[float32](1.)
	return s.self
}

// FlexShrink sets the flex item's shrink factor. [Docs](https://tailwindcss.com/docs/flex-shrink)
func (s *Styled[S]) FlexShrink(shrink float32) S {
	s.self.Style().FlexShrink = Some(shrink)
	return s.self
}

// FlexShrink0 disables flex item shrinking (flex-shrink: 0). [Docs](https://tailwindcss.com/docs/flex-shrink#dont-shrink)
func (s *Styled[S]) FlexShrink0() S {
	s.self.Style().FlexShrink = Some[float32](0.)
	return s.self
}

// FlexShrink1 enables flex item shrinking (flex-shrink: 1). [Docs](https://tailwindcss.com/docs/flex-shrink#shrink-1)
func (s *Styled[S]) FlexShrink1() S {
	s.self.Style().FlexShrink = Some[float32](1.)
	return s.self
}

// FlexWrap lets flex items wrap. [Docs](https://tailwindcss.com/docs/flex-wrap#wrap-normally)
func (s *Styled[S]) FlexWrap() S {
	s.self.Style().FlexWrap = Some(FlexWrapWrap)
	return s.self
}

// FlexWrapReverse wraps flex items in the reverse direction. [Docs](https://tailwindcss.com/docs/flex-wrap#wrap-reversed)
func (s *Styled[S]) FlexWrapReverse() S {
	s.self.Style().FlexWrap = Some(FlexWrapWrapReverse)
	return s.self
}

// FlexNowrap prevents flex items from wrapping, causing inflexible items to overflow the container if necessary. [Docs](https://tailwindcss.com/docs/flex-wrap#dont-wrap)
func (s *Styled[S]) FlexNowrap() S {
	s.self.Style().FlexWrap = Some(FlexWrapNoWrap)
	return s.self
}

// ItemsStart aligns flex items to the start of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-items#start)
func (s *Styled[S]) ItemsStart() S {
	s.self.Style().AlignItems = Some(AlignItemsFlexStart)
	return s.self
}

// ItemsEnd aligns flex items to the end of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-items#end)
func (s *Styled[S]) ItemsEnd() S {
	s.self.Style().AlignItems = Some(AlignItemsFlexEnd)
	return s.self
}

// ItemsCenter aligns flex items along the center of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-items#center)
func (s *Styled[S]) ItemsCenter() S {
	s.self.Style().AlignItems = Some(AlignItemsCenter)
	return s.self
}

// ItemsBaseline aligns flex items along the baseline of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-items#baseline)
func (s *Styled[S]) ItemsBaseline() S {
	s.self.Style().AlignItems = Some(AlignItemsBaseline)
	return s.self
}

// ItemsStretch stretches flex items to fill the available space along the container's cross axis. [Docs](https://tailwindcss.com/docs/align-items#stretch)
func (s *Styled[S]) ItemsStretch() S {
	s.self.Style().AlignItems = Some(AlignItemsStretch)
	return s.self
}

// SelfStart sets how this specific element is aligned along the container's cross axis. [Docs](https://tailwindcss.com/docs/align-self#start)
func (s *Styled[S]) SelfStart() S {
	s.self.Style().AlignSelf = Some(AlignItemsStart)
	return s.self
}

// SelfEnd aligns this element against the end of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-self#end)
func (s *Styled[S]) SelfEnd() S {
	s.self.Style().AlignSelf = Some(AlignItemsEnd)
	return s.self
}

// SelfFlexStart aligns this element against the start of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-self#start)
func (s *Styled[S]) SelfFlexStart() S {
	s.self.Style().AlignSelf = Some(AlignItemsFlexStart)
	return s.self
}

// SelfFlexEnd aligns this element against the end of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-self#end)
func (s *Styled[S]) SelfFlexEnd() S {
	s.self.Style().AlignSelf = Some(AlignItemsFlexEnd)
	return s.self
}

// SelfCenter aligns this element along the center of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-self#center)
func (s *Styled[S]) SelfCenter() S {
	s.self.Style().AlignSelf = Some(AlignItemsCenter)
	return s.self
}

// SelfBaseline aligns this element along the baseline of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-self#baseline)
func (s *Styled[S]) SelfBaseline() S {
	s.self.Style().AlignSelf = Some(AlignItemsBaseline)
	return s.self
}

// SelfStretch stretches this element to fill the available space along the container's cross axis. [Docs](https://tailwindcss.com/docs/align-self#stretch)
func (s *Styled[S]) SelfStretch() S {
	s.self.Style().AlignSelf = Some(AlignItemsStretch)
	return s.self
}

// JustifyStart justifies flex items against the start of the container's main axis. [Docs](https://tailwindcss.com/docs/justify-content#start)
func (s *Styled[S]) JustifyStart() S {
	s.self.Style().JustifyContent = Some(AlignContentStart)
	return s.self
}

// JustifyEnd justifies flex items against the end of the container's main axis. [Docs](https://tailwindcss.com/docs/justify-content#end)
func (s *Styled[S]) JustifyEnd() S {
	s.self.Style().JustifyContent = Some(AlignContentEnd)
	return s.self
}

// JustifyCenter justifies flex items along the center of the container's main axis. [Docs](https://tailwindcss.com/docs/justify-content#center)
func (s *Styled[S]) JustifyCenter() S {
	s.self.Style().JustifyContent = Some(AlignContentCenter)
	return s.self
}

// JustifyBetween justifies flex items along the container's main axis
// such that there is an equal amount of space between each item. [Docs](https://tailwindcss.com/docs/justify-content#space-between)
func (s *Styled[S]) JustifyBetween() S {
	s.self.Style().JustifyContent = Some(AlignContentSpaceBetween)
	return s.self
}

// JustifyAround justifies items along the container's main axis such
// that there is an equal amount of space on each side of each item. [Docs](https://tailwindcss.com/docs/justify-content#space-around)
func (s *Styled[S]) JustifyAround() S {
	s.self.Style().JustifyContent = Some(AlignContentSpaceAround)
	return s.self
}

// JustifyEvenly justifies items along the container's main axis such
// that there is an equal amount of space around each item, but also
// accounting for the doubling of space you would normally see between
// each item when using justify-around. [Docs](https://tailwindcss.com/docs/justify-content#space-evenly)
func (s *Styled[S]) JustifyEvenly() S {
	s.self.Style().JustifyContent = Some(AlignContentSpaceEvenly)
	return s.self
}

// ContentNormal packs content items in their default position as if no align-content value was set. [Docs](https://tailwindcss.com/docs/align-content#normal)
func (s *Styled[S]) ContentNormal() S {
	s.self.Style().AlignContent = None[AlignContent]()
	return s.self
}

// ContentCenter packs content items in the center of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-content#center)
func (s *Styled[S]) ContentCenter() S {
	s.self.Style().AlignContent = Some(AlignContentCenter)
	return s.self
}

// ContentStart packs content items against the start of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-content#start)
func (s *Styled[S]) ContentStart() S {
	s.self.Style().AlignContent = Some(AlignContentFlexStart)
	return s.self
}

// ContentEnd packs content items against the end of the container's cross axis. [Docs](https://tailwindcss.com/docs/align-content#end)
func (s *Styled[S]) ContentEnd() S {
	s.self.Style().AlignContent = Some(AlignContentFlexEnd)
	return s.self
}

// ContentBetween packs content items along the container's cross axis
// such that there is an equal amount of space between each item. [Docs](https://tailwindcss.com/docs/align-content#space-between)
func (s *Styled[S]) ContentBetween() S {
	s.self.Style().AlignContent = Some(AlignContentSpaceBetween)
	return s.self
}

// ContentAround packs content items along the container's cross axis
// such that there is an equal amount of space on each side of each item. [Docs](https://tailwindcss.com/docs/align-content#space-around)
func (s *Styled[S]) ContentAround() S {
	s.self.Style().AlignContent = Some(AlignContentSpaceAround)
	return s.self
}

// ContentEvenly packs content items along the container's cross axis
// such that there is an equal amount of space between each item. [Docs](https://tailwindcss.com/docs/align-content#space-evenly)
func (s *Styled[S]) ContentEvenly() S {
	s.self.Style().AlignContent = Some(AlignContentSpaceEvenly)
	return s.self
}

// ContentStretch lets content items fill the available space along the container's cross axis. [Docs](https://tailwindcss.com/docs/align-content#stretch)
func (s *Styled[S]) ContentStretch() S {
	s.self.Style().AlignContent = Some(AlignContentStretch)
	return s.self
}

// AspectRatio sets the aspect ratio of the element. [Docs](https://tailwindcss.com/docs/aspect-ratio)
func (s *Styled[S]) AspectRatio(ratio float32) S {
	s.self.Style().AspectRatio = Some(ratio)
	return s.self
}

// AspectSquare sets the aspect ratio of the element to 1/1 – equal width and height. [Docs](https://tailwindcss.com/docs/aspect-ratio)
func (s *Styled[S]) AspectSquare() S {
	s.self.Style().AspectRatio = Some[float32](1.0)
	return s.self
}

// Bg sets the background color of the element.
func (s *Styled[S]) Bg(fill IntoFill) S {
	s.self.Style().Background = Some(fill.ToFill())
	return s.self
}

// BorderDashed sets the border style of the element to dashed.
func (s *Styled[S]) BorderDashed() S {
	s.self.Style().BorderStyle = Some(BorderStyleDashed)
	return s.self
}

// TextStyle returns the text style that has been configured on this element.
func (s *Styled[S]) TextStyle() *TextStyleRefinement {
	style := s.self.Style()
	return &style.Text
}

// TextColor sets the text color of this element.
//
// This value cascades to its child elements.
func (s *Styled[S]) TextColor(color IntoHsla) S {
	s.TextStyle().Color = Some(color.ToHsla())
	return s.self
}

// FontWeight sets the font weight of this element.
//
// This value cascades to its child elements.
func (s *Styled[S]) FontWeight(weight FontWeight) S {
	s.TextStyle().FontWeight = Some(weight)
	return s.self
}

// TextBg sets the background color of this element's text.
//
// This value cascades to its child elements.
func (s *Styled[S]) TextBg(bg IntoHsla) S {
	s.TextStyle().BackgroundColor = Some(bg.ToHsla())
	return s.self
}

// TextSize sets the text size of this element.
//
// This value cascades to its child elements.
func (s *Styled[S]) TextSize(size IntoAbsoluteLength) S {
	s.TextStyle().FontSize = Some(size.ToAbsoluteLength())
	return s.self
}

// TextXs sets the text size to 'extra small'. [Docs](https://tailwindcss.com/docs/font-size#setting-the-font-size)
func (s *Styled[S]) TextXs() S {
	s.TextStyle().FontSize = Some(Rems(0.75).ToAbsoluteLength())
	return s.self
}

// TextSm sets the text size to 'small'. [Docs](https://tailwindcss.com/docs/font-size#setting-the-font-size)
func (s *Styled[S]) TextSm() S {
	s.TextStyle().FontSize = Some(Rems(0.875).ToAbsoluteLength())
	return s.self
}

// TextBase sets the text size to 'base'. [Docs](https://tailwindcss.com/docs/font-size#setting-the-font-size)
func (s *Styled[S]) TextBase() S {
	s.TextStyle().FontSize = Some(Rems(1.0).ToAbsoluteLength())
	return s.self
}

// TextLg sets the text size to 'large'. [Docs](https://tailwindcss.com/docs/font-size#setting-the-font-size)
func (s *Styled[S]) TextLg() S {
	s.TextStyle().FontSize = Some(Rems(1.125).ToAbsoluteLength())
	return s.self
}

// TextXl sets the text size to 'extra large'. [Docs](https://tailwindcss.com/docs/font-size#setting-the-font-size)
func (s *Styled[S]) TextXl() S {
	s.TextStyle().FontSize = Some(Rems(1.25).ToAbsoluteLength())
	return s.self
}

// Text2xl sets the text size to 'extra extra large'. [Docs](https://tailwindcss.com/docs/font-size#setting-the-font-size)
func (s *Styled[S]) Text2xl() S {
	s.TextStyle().FontSize = Some(Rems(1.5).ToAbsoluteLength())
	return s.self
}

// Text3xl sets the text size to 'extra extra extra large'. [Docs](https://tailwindcss.com/docs/font-size#setting-the-font-size)
func (s *Styled[S]) Text3xl() S {
	s.TextStyle().FontSize = Some(Rems(1.875).ToAbsoluteLength())
	return s.self
}

// Italic sets the font style of the element to italic. [Docs](https://tailwindcss.com/docs/font-style#italicizing-text)
func (s *Styled[S]) Italic() S {
	s.TextStyle().FontStyle = Some(FontStyleItalic)
	return s.self
}

// NotItalic sets the font style of the element to normal (not italic). [Docs](https://tailwindcss.com/docs/font-style#displaying-text-normally)
func (s *Styled[S]) NotItalic() S {
	s.TextStyle().FontStyle = Some(FontStyleNormal)
	return s.self
}

// Underline sets the text decoration to underline. [Docs](https://tailwindcss.com/docs/text-decoration-line#underling-text)
func (s *Styled[S]) Underline() S {
	style := s.TextStyle()
	style.Underline = Some(UnderlineStyle{Thickness: Px(1.)})
	return s.self
}

// LineThrough sets the decoration of the text to have a line through it. [Docs](https://tailwindcss.com/docs/text-decoration-line#adding-a-line-through-text)
func (s *Styled[S]) LineThrough() S {
	style := s.TextStyle()
	style.Strikethrough = Some(StrikethroughStyle{Thickness: Px(1.)})
	return s.self
}

// TextDecorationNone removes the text decoration on this element.
//
// This value cascades to its child elements.
func (s *Styled[S]) TextDecorationNone() S {
	s.TextStyle().Underline = None[UnderlineStyle]()
	return s.self
}

func (s *Styled[S]) underlineOrInsertDefault() *UnderlineStyle {
	style := s.TextStyle()
	if style.Underline.IsNone() {
		style.Underline = Some(UnderlineStyle{})
	}
	return style.Underline.AsMut()
}

// TextDecorationColor sets the color for the underline on this element.
func (s *Styled[S]) TextDecorationColor(color IntoHsla) S {
	underline := s.underlineOrInsertDefault()
	underline.Color = Some(color.ToHsla())
	return s.self
}

// TextDecorationSolid sets the text decoration style to a solid line. [Docs](https://tailwindcss.com/docs/text-decoration-style)
func (s *Styled[S]) TextDecorationSolid() S {
	underline := s.underlineOrInsertDefault()
	underline.Wavy = false
	return s.self
}

// TextDecorationWavy sets the text decoration style to a wavy line. [Docs](https://tailwindcss.com/docs/text-decoration-style)
func (s *Styled[S]) TextDecorationWavy() S {
	underline := s.underlineOrInsertDefault()
	underline.Wavy = true
	return s.self
}

// TextDecoration0 sets the text decoration to be 0px thick. [Docs](https://tailwindcss.com/docs/text-decoration-thickness)
func (s *Styled[S]) TextDecoration0() S {
	underline := s.underlineOrInsertDefault()
	underline.Thickness = Px(0.)
	return s.self
}

// TextDecoration1 sets the text decoration to be 1px thick. [Docs](https://tailwindcss.com/docs/text-decoration-thickness)
func (s *Styled[S]) TextDecoration1() S {
	underline := s.underlineOrInsertDefault()
	underline.Thickness = Px(1.)
	return s.self
}

// TextDecoration2 sets the text decoration to be 2px thick. [Docs](https://tailwindcss.com/docs/text-decoration-thickness)
func (s *Styled[S]) TextDecoration2() S {
	underline := s.underlineOrInsertDefault()
	underline.Thickness = Px(2.)
	return s.self
}

// TextDecoration4 sets the text decoration to be 4px thick. [Docs](https://tailwindcss.com/docs/text-decoration-thickness)
func (s *Styled[S]) TextDecoration4() S {
	underline := s.underlineOrInsertDefault()
	underline.Thickness = Px(4.)
	return s.self
}

// TextDecoration8 sets the text decoration to be 8px thick. [Docs](https://tailwindcss.com/docs/text-decoration-thickness)
func (s *Styled[S]) TextDecoration8() S {
	underline := s.underlineOrInsertDefault()
	underline.Thickness = Px(8.)
	return s.self
}

// FontFamily sets the font family of this element and its children.
func (s *Styled[S]) FontFamily(familyName string) S {
	s.TextStyle().FontFamily = Some(SharedString(familyName))
	return s.self
}

// FontFeatures sets the font features of this element and its children.
func (s *Styled[S]) FontFeatures(features FontFeatures) S {
	s.TextStyle().FontFeatures = Some(features)
	return s.self
}

// Font sets the font of this element and its children.
func (s *Styled[S]) Font(font Font) S {
	textStyle := s.TextStyle()
	textStyle.FontFamily = Some(font.Family)
	textStyle.FontFeatures = Some(font.Features)
	textStyle.FontWeight = Some(font.Weight)
	textStyle.FontStyle = Some(font.Style)
	textStyle.FontFallbacks = font.Fallbacks
	return s.self
}

// LineHeight sets the line height of this element and its children.
func (s *Styled[S]) LineHeight(lineHeight IntoDefiniteLength) S {
	s.TextStyle().LineHeight = Some(lineHeight.ToDefiniteLength())
	return s.self
}

// Opacity sets the opacity of this element and its children.
func (s *Styled[S]) Opacity(opacity float32) S {
	s.self.Style().Opacity = Some(opacity)
	return s.self
}

// GridCols sets the grid columns of this element.
func (s *Styled[S]) GridCols(cols uint16) S {
	s.self.Style().GridCols = Some(GridTemplate{Repeat: cols, MinSize: GridTemplateMinSizeZero})
	return s.self
}

// GridColsMinContent sets the grid columns with min-content minimum sizing.
// Unlike GridCols, it won't shrink to width 0 in AvailableSpace::MinContent constraints.
func (s *Styled[S]) GridColsMinContent(cols uint16) S {
	s.self.Style().GridCols = Some(GridTemplate{Repeat: cols, MinSize: GridTemplateMinSizeMinContent})
	return s.self
}

// GridColsMaxContent sets the grid columns with max-content maximum sizing for content-based column widths.
func (s *Styled[S]) GridColsMaxContent(cols uint16) S {
	s.self.Style().GridCols = Some(GridTemplate{Repeat: cols, MinSize: GridTemplateMinSizeMaxContent})
	return s.self
}

// GridRows sets the grid rows of this element.
func (s *Styled[S]) GridRows(rows uint16) S {
	s.self.Style().GridRows = Some(GridTemplate{Repeat: rows, MinSize: GridTemplateMinSizeZero})
	return s.self
}

// GridRowsMinContent sets the grid rows with min-content minimum sizing.
// Unlike GridRows, it won't shrink to height 0 in AvailableSpace::MinContent constraints.
func (s *Styled[S]) GridRowsMinContent(rows uint16) S {
	s.self.Style().GridRows = Some(GridTemplate{Repeat: rows, MinSize: GridTemplateMinSizeMinContent})
	return s.self
}

// GridRowsMaxContent sets the grid rows with max-content maximum sizing for content-based row heights.
func (s *Styled[S]) GridRowsMaxContent(rows uint16) S {
	s.self.Style().GridRows = Some(GridTemplate{Repeat: rows, MinSize: GridTemplateMinSizeMaxContent})
	return s.self
}

// ColStart sets the column start of this element.
func (s *Styled[S]) ColStart(start int16) S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Column.Start = GridPlacementLine(start)
	return s.self
}

// ColStartAuto sets the column start of this element to auto.
func (s *Styled[S]) ColStartAuto() S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Column.Start = GridPlacementAuto()
	return s.self
}

// ColEnd sets the column end of this element.
func (s *Styled[S]) ColEnd(end int16) S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Column.End = GridPlacementLine(end)
	return s.self
}

// ColEndAuto sets the column end of this element to auto.
func (s *Styled[S]) ColEndAuto() S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Column.End = GridPlacementAuto()
	return s.self
}

// ColSpan sets the column span of this element.
func (s *Styled[S]) ColSpan(span uint16) S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Column = GridPlacementRange{Start: GridPlacementSpan(span), End: GridPlacementSpan(span)}
	return s.self
}

// ColSpanFull sets the column span of this element to every column.
func (s *Styled[S]) ColSpanFull() S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Column = GridPlacementRange{Start: GridPlacementLine(1), End: GridPlacementLine(-1)}
	return s.self
}

// RowStart sets the row start of this element.
func (s *Styled[S]) RowStart(start int16) S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Row.Start = GridPlacementLine(start)
	return s.self
}

// RowStartAuto sets the row start of this element to auto.
func (s *Styled[S]) RowStartAuto() S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Row.Start = GridPlacementAuto()
	return s.self
}

// RowEnd sets the row end of this element.
func (s *Styled[S]) RowEnd(end int16) S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Row.End = GridPlacementLine(end)
	return s.self
}

// RowEndAuto sets the row end of this element to auto.
func (s *Styled[S]) RowEndAuto() S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Row.End = GridPlacementAuto()
	return s.self
}

// RowSpan sets the row span of this element.
func (s *Styled[S]) RowSpan(span uint16) S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Row = GridPlacementRange{Start: GridPlacementSpan(span), End: GridPlacementSpan(span)}
	return s.self
}

// RowSpanFull sets the row span of this element to every row.
func (s *Styled[S]) RowSpanFull() S {
	gridLocation := s.self.Style().GridLocationMut()
	gridLocation.Row = GridPlacementRange{Start: GridPlacementLine(1), End: GridPlacementLine(-1)}
	return s.self
}

// Debug draws a debug border around this element.
func (s *Styled[S]) Debug() S {
	s.self.Style().Debug = Some(true)
	return s.self
}

// DebugBelow draws a debug border on all conforming elements below this element.
func (s *Styled[S]) DebugBelow() S {
	s.self.Style().DebugBelow = Some(true)
	return s.self
}
