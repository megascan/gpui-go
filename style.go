package gpui

import (
	"encoding/binary"
	"encoding/json/jsontext"
	"fmt"
	"hash"
	"iter"
	"math"
	"slices"
	"strings"
)

//go:generate go run github.com/megascan/gpui-go/internal/cmd/refinegen style.go

// DebugBelow is set as a global while an element styled with debug_below paints its subtree.
type DebugBelow struct{}

// ObjectFit is how to fit an image into the bounds of the element.
type ObjectFit uint8

const (
	// ObjectFitFill stretches the image to fill the bounds of the element.
	ObjectFitFill ObjectFit = iota
	// ObjectFitContain scales the image to fit within the bounds of the element.
	ObjectFitContain
	// ObjectFitCover scales the image to cover the bounds of the element.
	ObjectFitCover
	// ObjectFitScaleDown scales the image down to fit within the bounds of the element.
	ObjectFitScaleDown
	// ObjectFitNone keeps the image at its original size.
	ObjectFitNone
)

var objectFitNames = [...]string{"Fill", "Contain", "Cover", "ScaleDown", "None"}

// String formats the fit like its Rust Debug output.
func (o ObjectFit) String() string { return variantName(objectFitNames[:], o, "ObjectFit") }

// GetBounds returns the bounds of the image within the given bounds.
func (o ObjectFit) GetBounds(bounds Bounds[Pixels], imageSize Size[DevicePixels]) Bounds[Pixels] {
	size := Size[Pixels]{
		Width:  Pixels(float32(uint32(imageSize.Width))),
		Height: Pixels(float32(uint32(imageSize.Height))),
	}
	imageRatio := float32(size.Width) / float32(size.Height)
	boundsRatio := float32(bounds.Size.Width) / float32(bounds.Size.Height)

	centered := func(newSize Size[Pixels]) Bounds[Pixels] {
		return Bounds[Pixels]{
			Origin: NewPoint(
				bounds.Origin.X+(bounds.Size.Width-newSize.Width)/2.0,
				bounds.Origin.Y+(bounds.Size.Height-newSize.Height)/2.0,
			),
			Size: newSize,
		}
	}
	containSize := func() Size[Pixels] {
		if boundsRatio > imageRatio {
			return NewSize(
				size.Width*Pixels(float32(bounds.Size.Height)/float32(size.Height)),
				bounds.Size.Height,
			)
		}
		return NewSize(
			bounds.Size.Width,
			size.Height*Pixels(float32(bounds.Size.Width)/float32(size.Width)),
		)
	}

	switch o {
	case ObjectFitFill:
		return bounds
	case ObjectFitContain:
		return centered(containSize())
	case ObjectFitScaleDown:
		if size.Width > bounds.Size.Width || size.Height > bounds.Size.Height {
			return centered(containSize())
		}
		return centered(NewSize(size.Width, size.Height))
	case ObjectFitCover:
		var newSize Size[Pixels]
		if boundsRatio > imageRatio {
			newSize = NewSize(
				bounds.Size.Width,
				size.Height*Pixels(float32(bounds.Size.Width)/float32(size.Width)),
			)
		} else {
			newSize = NewSize(
				size.Width*Pixels(float32(bounds.Size.Height)/float32(size.Height)),
				bounds.Size.Height,
			)
		}
		return centered(newSize)
	}
	return Bounds[Pixels]{Origin: bounds.Origin, Size: size}
}

// GridTemplateMinSize is the minimum size of a column or row in a grid layout.
type GridTemplateMinSize uint8

const (
	// GridTemplateMinSizeZero lets the column or row size be 0.
	GridTemplateMinSizeZero GridTemplateMinSize = iota
	// GridTemplateMinSizeMinContent sizes the column or row by its min content.
	GridTemplateMinSizeMinContent
	// GridTemplateMinSizeMaxContent sizes the column or row by its max content.
	GridTemplateMinSizeMaxContent
)

var gridTemplateMinSizeNames = [...]string{"Zero", "MinContent", "MaxContent"}

// String formats the value like its Rust Debug output.
func (g GridTemplateMinSize) String() string {
	return variantName(gridTemplateMinSizeNames[:], g, "GridTemplateMinSize")
}

// MarshalJSONTo encodes the value as its variant name.
func (g GridTemplateMinSize) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, gridTemplateMinSizeNames[:], g)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (g *GridTemplateMinSize) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, gridTemplateMinSizeNames[:], g)
}

// GridTemplate is a simplified representation of the grid-template-* value.
//
//gpui:refineable
type GridTemplate struct {
	// Repeat is how this template directive should be repeated.
	Repeat uint16 `json:"repeat"`
	// MinSize is the minimum size in the repeat(<>, minmax(_, 1fr)) equation.
	MinSize GridTemplateMinSize `json:"min_size"`
}

// Style is the CSS styling that can be applied to an element via the Styled mixin.
//
//gpui:refineable Debug, PartialEq, Serialize
type Style struct {
	// Display is the layout strategy to use.
	Display Display
	// Visibility is whether the element should be painted on screen.
	Visibility Visibility
	// Overflow is how children overflowing their container affect layout.
	//gpui:refineable
	Overflow PointOf[Overflow]
	// ScrollbarWidth is how much space should be reserved for the scrollbars of Scroll and Auto nodes.
	ScrollbarWidth AbsoluteLength
	// AllowConcurrentScroll is whether both axes should be scrollable at the same time.
	AllowConcurrentScroll bool
	// RestrictScrollToAxis is whether scrolling is restricted to the input gesture's axis.
	RestrictScrollToAxis bool
	// Position is the base offset the inset is applied to.
	Position Position
	// Inset is how the position of this element is tweaked relative to the layout.
	//gpui:refineable
	Inset EdgesOf[Length]
	// Size is the initial size of the item.
	//gpui:refineable
	Size SizeOf[Length]
	// MinSize is the minimum size of the item.
	//gpui:refineable
	MinSize SizeOf[Length]
	// MaxSize is the maximum size of the item.
	//gpui:refineable
	MaxSize SizeOf[Length]
	// AspectRatio is the preferred width / height ratio of the item.
	AspectRatio Option[float32]
	// Margin is the margin on each side.
	//gpui:refineable
	Margin EdgesOf[Length]
	// Padding is the padding on each side.
	//gpui:refineable
	Padding EdgesOf[DefiniteLength]
	// BorderWidths is the border width on each side.
	//gpui:refineable
	BorderWidths EdgesOf[AbsoluteLength]
	// AlignItems is how this node's children are aligned in the cross/block axis.
	AlignItems Option[AlignItems]
	// AlignSelf is how this node is aligned in the cross/block axis; falls back to the parent's AlignItems.
	AlignSelf Option[AlignSelf]
	// AlignContent is how content within this item is aligned in the cross/block axis.
	AlignContent Option[AlignContent]
	// JustifyContent is how content within this item is aligned in the main/inline axis.
	JustifyContent Option[JustifyContent]
	// Gap is the size of the gaps between items in a flex container.
	//gpui:refineable
	Gap SizeOf[DefiniteLength]
	// FlexDirection is the direction of the main axis.
	FlexDirection FlexDirection
	// FlexWrap is whether elements wrap or stay on a single line.
	FlexWrap FlexWrap
	// FlexBasis is the initial main axis size of the item.
	FlexBasis Length
	// FlexGrow is the relative rate at which this item grows.
	FlexGrow float32
	// FlexShrink is the relative rate at which this item shrinks.
	FlexShrink float32
	// Background is the fill color of this element.
	Background Option[Fill]
	// BorderColor is the border color of this element.
	BorderColor Option[Hsla]
	// BorderStyle is the border style of this element.
	BorderStyle BorderStyle
	// CornerRadii is the radius of the corners of this element.
	//gpui:refineable
	CornerRadii CornersOf[AbsoluteLength]
	// BoxShadow is the box shadow of the element.
	BoxShadow []BoxShadow
	// Text is the text style of this element.
	//gpui:refineable
	Text TextStyleRefinement
	// MouseCursor is the cursor shown when the mouse pointer is over the element.
	MouseCursor Option[CursorStyle]
	// Opacity is the opacity of this element.
	Opacity Option[float32]
	// GridCols is the grid columns of this element.
	GridCols Option[GridTemplate]
	// GridRows is the grid rows of this element.
	GridRows Option[GridTemplate]
	// GridLocation is the grid location of this element.
	GridLocation Option[GridLocation]
	// Debug is whether to draw a red debugging outline around this element.
	Debug bool
	// DebugBelow is whether to draw a red debugging outline around this element and all of its children.
	DebugBelow bool
}

// Style returns r itself (upstream's impl Styled for StyleRefinement).
func (r *StyleRefinement) Style() *StyleRefinement {
	return r
}

// GridLocationMut returns the grid location of this element, inserting the default first.
func (r *StyleRefinement) GridLocationMut() *GridLocation {
	if r.GridLocation.IsNone() {
		r.GridLocation = Some(GridLocation{})
	}
	return r.GridLocation.AsMut()
}

// Visibility is the value of the visibility property, similar to CSS visibility.
type Visibility uint8

const (
	// VisibilityVisible draws the element as normal.
	VisibilityVisible Visibility = iota
	// VisibilityHidden does not draw the element, but it still takes up space in the layout.
	VisibilityHidden
)

var visibilityNames = [...]string{"Visible", "Hidden"}

// String formats the value like its Rust Debug output.
func (v Visibility) String() string { return variantName(visibilityNames[:], v, "Visibility") }

// MarshalJSONTo encodes the value as its variant name.
func (v Visibility) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, visibilityNames[:], v)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (v *Visibility) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, visibilityNames[:], v)
}

// BoxShadow is a value of the box-shadow property.
type BoxShadow struct {
	// Color is the color of the shadow.
	Color Hsla `json:"color"`
	// Offset is how the shadow is offset from its element.
	Offset Point[Pixels] `json:"offset"`
	// Blur is how much the shadow is blurred (upstream blur_radius).
	Blur Pixels `json:"blur_radius"`
	// Spread is how much the shadow spreads (upstream spread_radius).
	Spread Pixels `json:"spread_radius"`
	// IsInset is whether this is an inset shadow, drawn inside the element's bounds.
	IsInset bool `json:"inset"`
}

// NewBoxShadow creates a box shadow with the given offset and color, in the order of the CSS box-shadow property.
func NewBoxShadow(offsetX, offsetY Pixels, color Hsla) BoxShadow {
	return BoxShadow{
		Color:   color,
		Offset:  NewPoint(offsetX, offsetY),
		Blur:    Px(0.),
		Spread:  Px(0.),
		IsInset: false,
	}
}

// BlurRadius returns b with the given blur radius.
func (b BoxShadow) BlurRadius(blurRadius Pixels) BoxShadow {
	b.Blur = blurRadius
	return b
}

// SpreadRadius returns b with the given spread radius.
func (b BoxShadow) SpreadRadius(spreadRadius Pixels) BoxShadow {
	b.Spread = spreadRadius
	return b
}

// Inset returns b marked as an inset shadow.
func (b BoxShadow) Inset() BoxShadow {
	b.IsInset = true
	return b
}

// WhiteSpace is how to handle whitespace in text.
type WhiteSpace uint8

const (
	// WhiteSpaceNormal wraps lines when text overflows the width of the element.
	WhiteSpaceNormal WhiteSpace = iota
	// WhiteSpaceNowrap does not wrap lines; text overflows the width of the element.
	WhiteSpaceNowrap
)

var whiteSpaceNames = [...]string{"Normal", "Nowrap"}

// String formats the value like its Rust Debug output.
func (w WhiteSpace) String() string { return variantName(whiteSpaceNames[:], w, "WhiteSpace") }

// MarshalJSONTo encodes the value as its variant name.
func (w WhiteSpace) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, whiteSpaceNames[:], w)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (w *WhiteSpace) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, whiteSpaceNames[:], w)
}

type textOverflowKind uint8

const (
	textOverflowTruncate textOverflowKind = iota
	textOverflowTruncateStart
	textOverflowTruncateMiddle
)

var textOverflowNames = [...]string{"Truncate", "TruncateStart", "TruncateMiddle"}

// TextOverflow is how to truncate text that overflows the width of the element; it is comparable.
type TextOverflow struct {
	kind   textOverflowKind
	marker SharedString
}

// TextOverflowTruncate truncates the text at the end and shows marker there (e.g. "very long te…").
func TextOverflowTruncate(marker SharedString) TextOverflow {
	return TextOverflow{kind: textOverflowTruncate, marker: marker}
}

// TextOverflowTruncateStart truncates the text at the start and shows marker there (e.g. "…ong text here").
func TextOverflowTruncateStart(marker SharedString) TextOverflow {
	return TextOverflow{kind: textOverflowTruncateStart, marker: marker}
}

// TextOverflowTruncateMiddle truncates the text in the middle, preserving its start and end (e.g. "long fi…name.rs").
func TextOverflowTruncateMiddle(marker SharedString) TextOverflow {
	return TextOverflow{kind: textOverflowTruncateMiddle, marker: marker}
}

// AsTruncate returns the marker when t is the Truncate variant.
func (t TextOverflow) AsTruncate() (SharedString, bool) {
	return t.marker, t.kind == textOverflowTruncate
}

// AsTruncateStart returns the marker when t is the TruncateStart variant.
func (t TextOverflow) AsTruncateStart() (SharedString, bool) {
	return t.marker, t.kind == textOverflowTruncateStart
}

// AsTruncateMiddle returns the marker when t is the TruncateMiddle variant.
func (t TextOverflow) AsTruncateMiddle() (SharedString, bool) {
	return t.marker, t.kind == textOverflowTruncateMiddle
}

// String formats the value like its Rust Debug output.
func (t TextOverflow) String() string {
	return fmt.Sprintf("%s(%q)", textOverflowNames[t.kind], string(t.marker))
}

// MarshalJSONTo encodes the value in serde's externally tagged form.
func (t TextOverflow) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(textOverflowNames[t.kind])); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(string(t.marker))); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSONFrom decodes the value from serde's externally tagged form.
func (t *TextOverflow) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '{' {
		return fmt.Errorf("invalid type: %v, expected enum TextOverflow", tok.Kind())
	}
	var kind textOverflowKind
	if err := unmarshalVariant(dec, textOverflowNames[:], &kind); err != nil {
		return err
	}
	tok, err = dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '"' {
		return fmt.Errorf("invalid type: %v, expected a string", tok.Kind())
	}
	marker := SharedString(tok.String())
	if tok, err = dec.ReadToken(); err != nil {
		return err
	}
	if tok.Kind() != '}' {
		return fmt.Errorf("invalid type: %v, expected end of enum TextOverflow", tok.Kind())
	}
	*t = TextOverflow{kind: kind, marker: marker}
	return nil
}

// TextAlign is how to align text within the element.
type TextAlign uint8

const (
	// TextAlignLeft aligns the text to the left of the element.
	TextAlignLeft TextAlign = iota
	// TextAlignCenter centers the text within the element.
	TextAlignCenter
	// TextAlignRight aligns the text to the right of the element.
	TextAlignRight
)

var textAlignNames = [...]string{"Left", "Center", "Right"}

// String formats the value like its Rust Debug output.
func (t TextAlign) String() string { return variantName(textAlignNames[:], t, "TextAlign") }

// MarshalJSONTo encodes the value as its variant name.
func (t TextAlign) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, textAlignNames[:], t)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (t *TextAlign) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, textAlignNames[:], t)
}

// TextStyle is the properties that can be used to style text.
//
//gpui:refineable Debug, PartialEq, Serialize
type TextStyle struct {
	// Color is the color of the text.
	Color Hsla
	// FontFamily is the font family to use.
	FontFamily SharedString
	// FontFeatures are the font features to use.
	FontFeatures FontFeatures
	// FontFallbacks are the fallback fonts to use.
	FontFallbacks Option[FontFallbacks]
	// FontSize is the font size, in pixels or rems.
	FontSize AbsoluteLength
	// LineHeight is the line height, in pixels or fractions.
	LineHeight DefiniteLength
	// FontWeight is the font weight, e.g. bold.
	FontWeight FontWeight
	// FontStyle is the font style, e.g. italic.
	FontStyle FontStyle
	// BackgroundColor is the background color of the text.
	BackgroundColor Option[Hsla]
	// Underline is the underline style of the text.
	Underline Option[UnderlineStyle]
	// Strikethrough is the strikethrough style of the text.
	Strikethrough Option[StrikethroughStyle]
	// WhiteSpace is how to handle whitespace in the text.
	WhiteSpace WhiteSpace
	// TextOverflow is how the text is truncated when it overflows the width of the element.
	TextOverflow Option[TextOverflow]
	// TextAlign is how the text is aligned within the element.
	TextAlign TextAlign
	// LineClamp is the number of lines to display before truncating the text.
	LineClamp Option[int]
}

// DefaultTextStyle returns the default text style.
func DefaultTextStyle() TextStyle {
	return TextStyle{
		Color:           Black(),
		FontFamily:      ".SystemUIFont",
		FontFeatures:    FontFeatures{},
		FontFallbacks:   None[FontFallbacks](),
		FontSize:        Rems(1.).ToAbsoluteLength(),
		LineHeight:      Phi(),
		FontWeight:      DefaultFontWeight(),
		FontStyle:       FontStyleNormal,
		BackgroundColor: None[Hsla](),
		Underline:       None[UnderlineStyle](),
		Strikethrough:   None[StrikethroughStyle](),
		WhiteSpace:      WhiteSpaceNormal,
		TextOverflow:    None[TextOverflow](),
		TextAlign:       TextAlignLeft,
		LineClamp:       None[int](),
	}
}

// Highlight returns a copy of t with the given highlighting applied.
func (t TextStyle) Highlight(style IntoHighlightStyle) TextStyle {
	s := style.ToHighlightStyle()
	if weight, ok := s.FontWeight.Get(); ok {
		t.FontWeight = weight
	}
	if fontStyle, ok := s.FontStyle.Get(); ok {
		t.FontStyle = fontStyle
	}
	if color, ok := s.Color.Get(); ok {
		t.Color = t.Color.Blend(color)
	}
	if factor, ok := s.FadeOut.Get(); ok {
		t.Color.FadeOut(factor)
	}
	if backgroundColor, ok := s.BackgroundColor.Get(); ok {
		t.BackgroundColor = Some(backgroundColor)
	}
	if underline, ok := s.Underline.Get(); ok {
		t.Underline = Some(underline)
	}
	if strikethrough, ok := s.Strikethrough.Get(); ok {
		t.Strikethrough = Some(strikethrough)
	}
	return t
}

// Font returns the font configured for this text style.
func (t TextStyle) Font() Font {
	return Font{
		Family:    t.FontFamily,
		Features:  t.FontFeatures,
		Fallbacks: t.FontFallbacks,
		Weight:    t.FontWeight,
		Style:     t.FontStyle,
	}
}

// LineHeightInPixels returns the rounded line height in pixels.
func (t TextStyle) LineHeightInPixels(remSize Pixels) Pixels {
	return t.LineHeight.ToPixels(t.FontSize, remSize).Round()
}

// ToRun converts this text style into a TextRun for the given length of text.
func (t TextStyle) ToRun(length int) TextRun {
	return TextRun{
		Len: length,
		Font: Font{
			Family:    t.FontFamily,
			Features:  t.FontFeatures,
			Fallbacks: t.FontFallbacks,
			Weight:    t.FontWeight,
			Style:     t.FontStyle,
		},
		Color:           t.Color,
		BackgroundColor: t.BackgroundColor,
		Underline:       t.Underline,
		Strikethrough:   t.Strikethrough,
	}
}

// HighlightStyle is a highlight to apply, like a TextStyle but for a single font with uniformly sized and spaced text.
type HighlightStyle struct {
	// Color is the color of the text.
	Color Option[Hsla]
	// FontWeight is the font weight, e.g. bold.
	FontWeight Option[FontWeight]
	// FontStyle is the font style, e.g. italic.
	FontStyle Option[FontStyle]
	// BackgroundColor is the background color of the text.
	BackgroundColor Option[Hsla]
	// Underline is the underline style of the text.
	Underline Option[UnderlineStyle]
	// Strikethrough is the strikethrough style of the text.
	Strikethrough Option[StrikethroughStyle]
	// FadeOut makes the text less vibrant, similar to the CSS opacity property.
	FadeOut Option[float32]
}

// IntoHighlightStyle is implemented by every value convertible to a HighlightStyle (D-INTO).
type IntoHighlightStyle interface {
	ToHighlightStyle() HighlightStyle
}

// Hash writes h to state in the order of upstream's Hash impl.
func (h HighlightStyle) Hash(state hash.Hash) {
	fmt.Fprint(state, h.Color, h.FontWeight, h.FontStyle, h.BackgroundColor, h.Underline, h.Strikethrough)
	var fade uint32
	if f, ok := h.FadeOut.Get(); ok {
		fade = math.Float32bits(f)
	}
	_, _ = state.Write(binary.BigEndian.AppendUint32(nil, fade))
}

// HasOpaqueBackground reports whether the background is a non-transparent color.
func (s *Style) HasOpaqueBackground() bool {
	fill, ok := s.Background.Get()
	if !ok {
		return false
	}
	color, ok := fill.Color()
	return ok && !color.IsTransparent()
}

// TextStyle returns the text style in this element style, or nil when none is set.
func (s *Style) TextStyle() *TextStyleRefinement {
	if s.Text.IsSome() {
		return &s.Text
	}
	return nil
}

// OverflowMask returns the content mask for this style within bounds, or false when the element does not hide its overflow.
func (s *Style) OverflowMask(bounds Bounds[Pixels], remSize Pixels) (ContentMask[Pixels], bool) {
	if s.Overflow.X == OverflowVisible && s.Overflow.Y == OverflowVisible {
		return ContentMask[Pixels]{}, false
	}
	min := bounds.Origin
	max := bounds.BottomRight()

	if color, ok := s.BorderColor.Get(); ok && !color.IsTransparent() {
		min.X += s.BorderWidths.Left.ToPixels(remSize)
		max.X -= s.BorderWidths.Right.ToPixels(remSize)
		min.Y += s.BorderWidths.Top.ToPixels(remSize)
		max.Y -= s.BorderWidths.Bottom.ToPixels(remSize)
	}

	var masked Bounds[Pixels]
	switch xVisible, yVisible := s.Overflow.X == OverflowVisible, s.Overflow.Y == OverflowVisible; {
	case xVisible && yVisible:
		return ContentMask[Pixels]{}, false
	case xVisible:
		masked = BoundsFromCorners(NewPoint(min.X, bounds.Origin.Y), NewPoint(max.X, bounds.BottomRight().Y))
	case yVisible:
		masked = BoundsFromCorners(NewPoint(bounds.Origin.X, min.Y), NewPoint(bounds.BottomRight().X, max.Y))
	default:
		masked = BoundsFromCorners(min, max)
	}
	return ContentMask[Pixels]{Bounds: masked}, true
}

// Paint paints the background of an element styled with this style, calls continuation, then paints the border.
func (s *Style) Paint(bounds Bounds[Pixels], window *Window, cx *App, continuation func(window *Window, cx *App)) {
	if s.DebugBelow {
		cx.SetGlobal(DebugBelow{})
	}

	if s.Debug || cx.HasGlobal[DebugBelow]() {
		window.PaintQuad(Outline(bounds, Red(), BorderStyleSolid))
	}

	remSize := window.RemSize()
	cornerRadii := CornersAbsoluteLengthToPixels(s.CornerRadii, remSize).ClampRadiiForQuadSize(bounds.Size)

	window.PaintDropShadows(bounds, cornerRadii, s.BoxShadow)

	var backgroundColor Option[Background]
	if fill, ok := s.Background.Get(); ok {
		if color, ok := fill.Color(); ok {
			backgroundColor = Some(color)
		}
	}
	if color, ok := backgroundColor.Get(); ok && !color.IsTransparent() {
		var borderColor Hsla
		switch color.tag {
		case BackgroundTagSolid, BackgroundTagPatternSlash, BackgroundTagCheckerboard:
			borderColor = color.solid
		case BackgroundTagLinearGradient:
			borderColor = color.colors[0].Color
		}
		borderColor.A = 0.
		window.PaintQuad(NewQuad(
			bounds,
			cornerRadii,
			backgroundColor.UnwrapOrDefault(),
			Edges[Pixels]{},
			borderColor,
			s.BorderStyle,
		))
	}

	window.PaintInsetShadows(bounds, cornerRadii, s.BoxShadow)

	continuation(window, cx)

	if s.isBorderVisible() {
		borderWidths := EdgesAbsoluteLengthToPixels(s.BorderWidths, remSize)
		background := s.BorderColor.UnwrapOrDefault()
		background.A = 0.
		window.PaintQuad(NewQuad(
			bounds,
			cornerRadii,
			background.ToBackground(),
			borderWidths,
			s.BorderColor.UnwrapOrDefault(),
			s.BorderStyle,
		))
	}

	if s.DebugBelow {
		cx.RemoveGlobal[DebugBelow]()
	}
}

func (s *Style) isBorderVisible() bool {
	color, ok := s.BorderColor.Get()
	return ok && !color.IsTransparent() &&
		s.BorderWidths.Any(func(length AbsoluteLength) bool { return !length.IsZero() })
}

// DefaultStyle returns the default style.
func DefaultStyle() Style {
	return Style{
		Display:               DisplayBlock,
		Visibility:            VisibilityVisible,
		Overflow:              PointOf[Overflow]{X: OverflowVisible, Y: OverflowVisible},
		AllowConcurrentScroll: false,
		RestrictScrollToAxis:  false,
		ScrollbarWidth:        AbsoluteLength{},
		Position:              PositionRelative,
		Inset:                 EdgesLengthAuto(),
		Margin:                EdgesLengthZero(),
		Padding:               EdgesDefiniteLengthZero(),
		BorderWidths:          EdgesAbsoluteLengthZero(),
		Size:                  SizeLengthAuto(),
		MinSize:               SizeLengthAuto(),
		MaxSize:               SizeLengthAuto(),
		AspectRatio:           None[float32](),
		Gap:                   SizeOf[DefiniteLength]{},
		AlignItems:            None[AlignItems](),
		AlignSelf:             None[AlignSelf](),
		AlignContent:          None[AlignContent](),
		JustifyContent:        None[JustifyContent](),
		FlexDirection:         FlexDirectionRow,
		FlexWrap:              FlexWrapNoWrap,
		FlexGrow:              0.0,
		FlexShrink:            1.0,
		FlexBasis:             LengthAuto(),
		Background:            None[Fill](),
		BorderColor:           None[Hsla](),
		BorderStyle:           BorderStyleSolid,
		CornerRadii:           CornersOf[AbsoluteLength]{},
		BoxShadow:             nil,
		Text:                  TextStyleRefinement{},
		MouseCursor:           None[CursorStyle](),
		Opacity:               None[float32](),
		GridRows:              None[GridTemplate](),
		GridCols:              None[GridTemplate](),
		GridLocation:          None[GridLocation](),
		Debug:                 false,
		DebugBelow:            false,
	}
}

// UnderlineStyle is the properties that can be applied to an underline.
//
//gpui:refineable
type UnderlineStyle struct {
	// Thickness is the thickness of the underline.
	Thickness Pixels `json:"thickness"`
	// Color is the color of the underline.
	Color Option[Hsla] `json:"color"`
	// Wavy is whether the underline is wavy, like in a spell checker.
	Wavy bool `json:"wavy"`
}

// StrikethroughStyle is the properties that can be applied to a strikethrough.
//
//gpui:refineable
type StrikethroughStyle struct {
	// Thickness is the thickness of the strikethrough.
	Thickness Pixels `json:"thickness"`
	// Color is the color of the strikethrough.
	Color Option[Hsla] `json:"color"`
}

// Fill is the kind of fill that can be applied to a shape; its only variant is FillColor.
type Fill interface {
	// Color returns the fill as a solid color, if it is one.
	Color() (Background, bool)
	IntoFill
	isFill()
}

// FillColor is a solid color fill.
type FillColor Background

func (FillColor) isFill() {}

// Color returns the fill's background.
func (f FillColor) Color() (Background, bool) {
	return Background(f), true
}

// ToFill returns f.
func (f FillColor) ToFill() Fill {
	return f
}

// String formats the fill like its Rust Debug output.
func (f FillColor) String() string {
	return "Color(" + Background(f).String() + ")"
}

// DefaultFill returns a fill of the default background.
func DefaultFill() Fill {
	return FillColor(Background{})
}

// IntoFill is implemented by every value convertible to a Fill (D-INTO).
type IntoFill interface {
	ToFill() Fill
}

// ToFill converts the color to a solid fill.
func (c Hsla) ToFill() Fill {
	return FillColor(c.ToBackground())
}

// ToFill converts the color to a solid fill.
func (c Rgba) ToFill() Fill {
	return FillColor(c.ToBackground())
}

// ToFill wraps the background in a fill.
func (b Background) ToFill() Fill {
	return FillColor(b)
}

// ToHighlightStyle converts the text style into a highlight style.
func (t TextStyle) ToHighlightStyle() HighlightStyle {
	return HighlightStyle{
		Color:           Some(t.Color),
		FontWeight:      Some(t.FontWeight),
		FontStyle:       Some(t.FontStyle),
		BackgroundColor: t.BackgroundColor,
		Underline:       t.Underline,
		Strikethrough:   t.Strikethrough,
		FadeOut:         None[float32](),
	}
}

// HighlightStyleColor creates a highlight style with just a color.
func HighlightStyleColor(color Hsla) HighlightStyle {
	return HighlightStyle{Color: Some(color)}
}

// Highlight blends other into h. Non-continuous properties, like FontWeight and FontStyle, are overwritten.
func (h HighlightStyle) Highlight(other HighlightStyle) HighlightStyle {
	color := h.Color
	if otherColor, ok := other.Color.Get(); ok {
		if c, ok := h.Color.Get(); ok {
			color = Some(c.Blend(otherColor))
		} else {
			color = Some(otherColor)
		}
	}
	fadeOut := h.FadeOut
	if sourceFade, ok := other.FadeOut.Get(); ok {
		if destFade, ok := h.FadeOut.Get(); ok {
			fadeOut = Some(clampF32(destFade*(1.+sourceFade), 0., 1.))
		} else {
			fadeOut = Some(sourceFade)
		}
	}
	return HighlightStyle{
		Color:           color,
		FontWeight:      other.FontWeight.Or(h.FontWeight),
		FontStyle:       other.FontStyle.Or(h.FontStyle),
		BackgroundColor: other.BackgroundColor.Or(h.BackgroundColor),
		Underline:       other.Underline.Or(h.Underline),
		Strikethrough:   other.Strikethrough.Or(h.Strikethrough),
		FadeOut:         fadeOut,
	}
}

// ToHighlightStyle returns h.
func (h HighlightStyle) ToHighlightStyle() HighlightStyle {
	return h
}

// ToHighlightStyle creates a highlight style with just this color.
func (c Hsla) ToHighlightStyle() HighlightStyle {
	return HighlightStyle{Color: Some(c)}
}

// ToHighlightStyle creates a highlight style with just this font weight.
func (w FontWeight) ToHighlightStyle() HighlightStyle {
	return HighlightStyle{FontWeight: Some(w)}
}

// ToHighlightStyle creates a highlight style with just this font style.
func (s FontStyle) ToHighlightStyle() HighlightStyle {
	return HighlightStyle{FontStyle: Some(s)}
}

// ToHighlightStyle creates a highlight style with just this color.
func (c Rgba) ToHighlightStyle() HighlightStyle {
	return HighlightStyle{Color: Some(c.ToHsla())}
}

// CombineHighlights combines and merges the highlights and ranges in a and b.
func CombineHighlights(a, b iter.Seq2[Range, HighlightStyle]) iter.Seq2[Range, HighlightStyle] {
	type endpoint struct {
		position    int
		highlightID int
		isStart     bool
	}
	var endpoints []endpoint
	var highlights []HighlightStyle
	for _, seq := range []iter.Seq2[Range, HighlightStyle]{a, b} {
		if seq == nil {
			continue
		}
		for r, highlight := range seq {
			if r.Start < r.End {
				highlightID := len(highlights)
				endpoints = append(endpoints, endpoint{r.Start, highlightID, true})
				endpoints = append(endpoints, endpoint{r.End, highlightID, false})
				highlights = append(highlights, highlight)
			}
		}
	}
	slices.SortFunc(endpoints, func(x, y endpoint) int { return x.position - y.position })

	return func(yield func(Range, HighlightStyle) bool) {
		var activeStyles []int
		ix := 0
		for _, e := range endpoints {
			prevIndex := ix
			ix = e.position
			if ix > prevIndex && len(activeStyles) > 0 {
				currentStyle := HighlightStyle{}
				for _, highlightID := range activeStyles {
					currentStyle = currentStyle.Highlight(highlights[highlightID])
				}
				if !yield(Range{Start: prevIndex, End: ix}, currentStyle) {
					return
				}
			}
			if e.isStart {
				if i, found := slices.BinarySearch(activeStyles, e.highlightID); !found {
					activeStyles = slices.Insert(activeStyles, i, e.highlightID)
				}
			} else if i, found := slices.BinarySearch(activeStyles, e.highlightID); found {
				activeStyles = slices.Delete(activeStyles, i, i+1)
			}
		}
	}
}

// AlignItems controls how child nodes are aligned: the cross axis for flexbox, the block axis for grid.
type AlignItems uint8

const (
	// AlignItemsStart packs items toward the start of the axis.
	AlignItemsStart AlignItems = iota
	// AlignItemsEnd packs items toward the end of the axis.
	AlignItemsEnd
	// AlignItemsFlexStart packs items towards the flex-relative start of the axis.
	AlignItemsFlexStart
	// AlignItemsFlexEnd packs items towards the flex-relative end of the axis.
	AlignItemsFlexEnd
	// AlignItemsCenter packs items along the center of the cross axis.
	AlignItemsCenter
	// AlignItemsBaseline aligns items such that their baselines align.
	AlignItemsBaseline
	// AlignItemsStretch stretches items to fill the container.
	AlignItemsStretch
)

var alignItemsNames = [...]string{"Start", "End", "FlexStart", "FlexEnd", "Center", "Baseline", "Stretch"}

// String formats the value like its Rust Debug output.
func (a AlignItems) String() string { return variantName(alignItemsNames[:], a, "AlignItems") }

// MarshalJSONTo encodes the value as its variant name.
func (a AlignItems) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, alignItemsNames[:], a)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (a *AlignItems) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, alignItemsNames[:], a)
}

// JustifyItems controls how child nodes are aligned in the inline axis of a grid.
type JustifyItems = AlignItems

// AlignSelf overrides the parent's AlignItems for one node.
type AlignSelf = AlignItems

// JustifySelf overrides the parent's JustifyItems for one node.
type JustifySelf = AlignItems

// AlignContent sets the distribution of space between and around content items: the cross axis for flexbox, the block axis for grid.
type AlignContent uint8

const (
	// AlignContentStart packs items toward the start of the axis.
	AlignContentStart AlignContent = iota
	// AlignContentEnd packs items toward the end of the axis.
	AlignContentEnd
	// AlignContentFlexStart packs items towards the flex-relative start of the axis.
	AlignContentFlexStart
	// AlignContentFlexEnd packs items towards the flex-relative end of the axis.
	AlignContentFlexEnd
	// AlignContentCenter centers items around the middle of the axis.
	AlignContentCenter
	// AlignContentStretch stretches items to fill the container.
	AlignContentStretch
	// AlignContentSpaceBetween aligns the first and last items flush with the edges and distributes the gap evenly.
	AlignContentSpaceBetween
	// AlignContentSpaceEvenly makes the outer gaps the same as the gaps between items.
	AlignContentSpaceEvenly
	// AlignContentSpaceAround makes the outer gaps half the gaps between items.
	AlignContentSpaceAround
)

var alignContentNames = [...]string{"Start", "End", "FlexStart", "FlexEnd", "Center", "Stretch", "SpaceBetween", "SpaceEvenly", "SpaceAround"}

// String formats the value like its Rust Debug output.
func (a AlignContent) String() string { return variantName(alignContentNames[:], a, "AlignContent") }

// MarshalJSONTo encodes the value as its variant name.
func (a AlignContent) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, alignContentNames[:], a)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (a *AlignContent) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, alignContentNames[:], a)
}

// JustifyContent sets the distribution of space between and around content items: the main axis for flexbox, the inline axis for grid.
type JustifyContent = AlignContent

// Display sets the layout used for the children of this node. The zero value is Flex, upstream's default variant.
type Display uint8

const (
	// DisplayFlex lays the children out with the flexbox algorithm.
	DisplayFlex Display = 0
	// DisplayBlock lays the children out with the block algorithm.
	DisplayBlock Display = 1
	// DisplayGrid lays the children out with the CSS grid algorithm.
	DisplayGrid Display = 2
	// DisplayNone does not lay the children out; they follow absolute positioning.
	DisplayNone Display = 3
)

var displayNames = [...]string{"Flex", "Block", "Grid", "None"}

// String formats the value like its Rust Debug output.
func (d Display) String() string { return variantName(displayNames[:], d, "Display") }

// MarshalJSONTo encodes the value as its variant name.
func (d Display) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, displayNames[:], d)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (d *Display) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, displayNames[:], d)
}

// FlexWrap controls whether flex items are forced onto one line or can wrap onto multiple lines.
type FlexWrap uint8

const (
	// FlexWrapNoWrap keeps items on a single line.
	FlexWrapNoWrap FlexWrap = iota
	// FlexWrapWrap wraps items according to the FlexDirection.
	FlexWrapWrap
	// FlexWrapWrapReverse wraps items in the opposite direction to the FlexDirection.
	FlexWrapWrapReverse
)

var flexWrapNames = [...]string{"NoWrap", "Wrap", "WrapReverse"}

// String formats the value like its Rust Debug output.
func (f FlexWrap) String() string { return variantName(flexWrapNames[:], f, "FlexWrap") }

// MarshalJSONTo encodes the value as its variant name.
func (f FlexWrap) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, flexWrapNames[:], f)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (f *FlexWrap) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, flexWrapNames[:], f)
}

// FlexDirection is the direction of the flexbox layout main axis.
type FlexDirection uint8

const (
	// FlexDirectionRow defines +x as the main axis.
	FlexDirectionRow FlexDirection = iota
	// FlexDirectionColumn defines +y as the main axis.
	FlexDirectionColumn
	// FlexDirectionRowReverse defines -x as the main axis.
	FlexDirectionRowReverse
	// FlexDirectionColumnReverse defines -y as the main axis.
	FlexDirectionColumnReverse
)

var flexDirectionNames = [...]string{"Row", "Column", "RowReverse", "ColumnReverse"}

// String formats the value like its Rust Debug output.
func (f FlexDirection) String() string { return variantName(flexDirectionNames[:], f, "FlexDirection") }

// MarshalJSONTo encodes the value as its variant name.
func (f FlexDirection) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, flexDirectionNames[:], f)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (f *FlexDirection) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, flexDirectionNames[:], f)
}

// Overflow is how children overflowing their container affect layout.
type Overflow uint8

const (
	// OverflowVisible bases the automatic minimum size on the content; overflow contributes to the parent's scroll region.
	OverflowVisible Overflow = iota
	// OverflowClip bases the automatic minimum size on the content; overflow does not contribute to the parent's scroll region.
	OverflowClip
	// OverflowHidden makes the automatic minimum size 0; overflow does not contribute to the parent's scroll region.
	OverflowHidden
	// OverflowScroll is like Hidden and also reserves space for a scrollbar.
	OverflowScroll
)

var overflowNames = [...]string{"Visible", "Clip", "Hidden", "Scroll"}

// String formats the value like its Rust Debug output.
func (o Overflow) String() string { return variantName(overflowNames[:], o, "Overflow") }

// MarshalJSONTo encodes the value as its variant name.
func (o Overflow) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, overflowNames[:], o)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (o *Overflow) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, overflowNames[:], o)
}

// Position is the positioning strategy for this item.
type Position uint8

const (
	// PositionRelative offsets the item relative to the position given by the layout algorithm.
	PositionRelative Position = iota
	// PositionAbsolute offsets the item relative to its closest positioned ancestor and takes it out of the flow.
	PositionAbsolute
)

var positionNames = [...]string{"Relative", "Absolute"}

// String formats the value like its Rust Debug output.
func (p Position) String() string { return variantName(positionNames[:], p, "Position") }

// MarshalJSONTo encodes the value as its variant name.
func (p Position) MarshalJSONTo(enc *jsontext.Encoder) error {
	return marshalVariant(enc, positionNames[:], p)
}

// UnmarshalJSONFrom decodes the value from its variant name.
func (p *Position) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return unmarshalVariant(dec, positionNames[:], p)
}

func variantName[T ~uint8](names []string, v T, typeName string) string {
	if int(v) < len(names) {
		return names[v]
	}
	return fmt.Sprintf("%s(%d)", typeName, uint8(v))
}

func marshalVariant[T ~uint8](enc *jsontext.Encoder, names []string, v T) error {
	if int(v) >= len(names) {
		return fmt.Errorf("invalid variant %d", uint8(v))
	}
	return enc.WriteToken(jsontext.String(names[v]))
}

func unmarshalVariant[T ~uint8](dec *jsontext.Decoder, names []string, v *T) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '"' {
		return fmt.Errorf("invalid type: %v, expected variant identifier", tok.Kind())
	}
	s := tok.String()
	for i, name := range names {
		if name == s {
			*v = T(i)
			return nil
		}
	}
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = "`" + name + "`"
	}
	return fmt.Errorf("unknown variant `%s`, expected one of %s", s, strings.Join(quoted, ", "))
}
