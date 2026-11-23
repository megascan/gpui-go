package gpui

import "github.com/megascan/gpui-go/gpuiutil"

type Option[T any] = gpuiutil.Option[T]

type Pixels float32

type Rems float32

func Px(v float32) Pixels { return Pixels(v) }

type AbsoluteLength struct{}

type DefiniteLength struct{}

func (d DefiniteLength) Neg() DefiniteLength { return d }

type Length struct{}

func Relative(f float32) DefiniteLength { return DefiniteLength{} }

func Auto() Length { return Length{} }

type IntoLength interface{ ToLength() Length }

type IntoDefiniteLength interface{ ToDefiniteLength() DefiniteLength }

type IntoAbsoluteLength interface{ ToAbsoluteLength() AbsoluteLength }

func (Pixels) ToLength() Length                 { return Length{} }
func (Pixels) ToDefiniteLength() DefiniteLength { return DefiniteLength{} }
func (Pixels) ToAbsoluteLength() AbsoluteLength { return AbsoluteLength{} }
func (Rems) ToLength() Length                   { return Length{} }
func (Rems) ToDefiniteLength() DefiniteLength   { return DefiniteLength{} }
func (Rems) ToAbsoluteLength() AbsoluteLength   { return AbsoluteLength{} }
func (DefiniteLength) ToLength() Length         { return Length{} }
func (DefiniteLength) ToDefiniteLength() DefiniteLength {
	return DefiniteLength{}
}
func (Length) ToLength() Length { return Length{} }

type Hsla struct{ H, S, L, A float32 }

func NewHsla(h, s, l, a float32) Hsla { return Hsla{h, s, l, a} }

type IntoHsla interface{ ToHsla() Hsla }

type BoxShadow struct{}

func NewBoxShadow(x, y Pixels, color Hsla) BoxShadow { return BoxShadow{} }
func (b BoxShadow) BlurRadius(r Pixels) BoxShadow    { return b }
func (b BoxShadow) SpreadRadius(r Pixels) BoxShadow  { return b }

type Visibility uint8

const (
	VisibilityVisible Visibility = iota
	VisibilityHidden
)

type Position uint8

const (
	PositionRelative Position = iota
	PositionAbsolute
)

type Overflow uint8

const OverflowHidden Overflow = 1

type CursorStyle uint8

const (
	CursorStyleArrow CursorStyle = iota
	CursorStyleIBeam
	CursorStyleCrosshair
	CursorStyleClosedHand
	CursorStyleOpenHand
	CursorStylePointingHand
	CursorStyleResizeLeft
	CursorStyleResizeRight
	CursorStyleResizeLeftRight
	CursorStyleResizeUp
	CursorStyleResizeDown
	CursorStyleResizeUpDown
	CursorStyleResizeUpLeftDownRight
	CursorStyleResizeUpRightDownLeft
	CursorStyleResizeColumn
	CursorStyleResizeRow
	CursorStyleIBeamCursorForVerticalLayout
	CursorStyleOperationNotAllowed
	CursorStyleDragLink
	CursorStyleDragCopy
	CursorStyleContextualMenu
)

type EdgesRefinement[T any] struct{ Top, Right, Bottom, Left Option[T] }

type SizeRefinement[T any] struct{ Width, Height Option[T] }

type CornersRefinement[T any] struct{ TopLeft, TopRight, BottomRight, BottomLeft Option[T] }

type PointRefinement[T any] struct{ X, Y Option[T] }

type StyleRefinement struct {
	Visibility   Option[Visibility]
	Overflow     PointRefinement[Overflow]
	Position     Option[Position]
	Inset        EdgesRefinement[Length]
	Size         SizeRefinement[Length]
	MinSize      SizeRefinement[Length]
	MaxSize      SizeRefinement[Length]
	Margin       EdgesRefinement[Length]
	Padding      EdgesRefinement[DefiniteLength]
	BorderWidths EdgesRefinement[AbsoluteLength]
	Gap          SizeRefinement[DefiniteLength]
	BorderColor  Option[Hsla]
	CornerRadii  CornersRefinement[AbsoluteLength]
	BoxShadow    Option[[]BoxShadow]
	MouseCursor  Option[CursorStyle]
}

type Styler interface{ Style() *StyleRefinement }

type Styled[S Styler] struct{ self S }
