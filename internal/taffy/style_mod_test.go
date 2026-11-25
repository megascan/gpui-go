package taffy

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestStyleMod_DefaultsMatch(t *testing.T) {
	lpaAuto := LengthPercentageAutoAuto()
	lpaZero := LengthPercentageAutoZero
	lpZero := LengthPercentageZero
	oldDefaults := Style[defaultCheapStr]{
		Display:                    DisplayDefault,
		ItemIsTable:                false,
		ItemIsReplaced:             false,
		ItemIsCompressibleReplaced: false,
		BoxSizing:                  BoxSizingDefault,
		Float:                      FloatDefault,
		Clear:                      ClearDefault,
		Direction:                  DirectionDefault,
		Overflow:                   Point[Overflow]{X: OverflowDefault, Y: OverflowDefault},
		ScrollbarWidth:             0.0,
		Contain:                    ContainDefault,
		Position:                   PositionDefault,
		FlexDirection:              FlexDirectionDefault,
		FlexWrap:                   FlexWrapDefault,
		AlignItems:                 AlignItemsDefault,
		AlignSelf:                  Option[AlignSelf]{},
		JustifyItems:               AlignItemsDefault,
		JustifySelf:                Option[AlignSelf]{},
		AlignContent:               AlignContentDefault,
		JustifyContent:             AlignContentDefault,
		Inset:                      Rect[LengthPercentageAuto]{Left: lpaAuto, Right: lpaAuto, Top: lpaAuto, Bottom: lpaAuto},
		Margin:                     Rect[LengthPercentageAuto]{Left: lpaZero, Right: lpaZero, Top: lpaZero, Bottom: lpaZero},
		Padding:                    Rect[LengthPercentage]{Left: lpZero, Right: lpZero, Top: lpZero, Bottom: lpZero},
		Border:                     Rect[LengthPercentage]{Left: lpZero, Right: lpZero, Top: lpZero, Bottom: lpZero},
		Gap:                        Size[LengthPercentage]{Width: lpZero, Height: lpZero},
		TextAlign:                  TextAlignDefault,
		FlexGrow:                   0.0,
		FlexShrink:                 1.0,
		FlexBasis:                  DimensionAuto(),
		Size:                       Size[Dimension]{Width: DimensionAuto(), Height: DimensionAuto()},
		MinSize:                    Size[LengthPercentageAuto]{Width: lpaAuto, Height: lpaAuto},
		MaxSize:                    Size[LengthPercentageAuto]{Width: lpaAuto, Height: lpaAuto},
		AspectRatio:                Option[float32]{},
		GridAutoFlow:               GridAutoFlowRow,
		GridRow:                    Line[GridPlacement[defaultCheapStr]]{},
		GridColumn:                 Line[GridPlacement[defaultCheapStr]]{},
	}
	if !reflect.DeepEqual(StyleDefault[defaultCheapStr](), oldDefaults) {
		t.Errorf("StyleDefault() = %+v, want %+v", StyleDefault[defaultCheapStr](), oldDefaults)
	}
}

func TestStyleMod_StyleSizes(t *testing.T) {
	cases := []struct {
		name     string
		got      uintptr
		expected uintptr
	}{
		{"Display", unsafe.Sizeof(Display(0)), 1},
		{"BoxSizing", unsafe.Sizeof(BoxSizing(0)), 1},
		{"Position", unsafe.Sizeof(Position(0)), 1},
		{"Overflow", unsafe.Sizeof(Overflow(0)), 1},
		{"f32", unsafe.Sizeof(float32(0)), 4},
		{"LengthPercentage", unsafe.Sizeof(LengthPercentage{}), 8},
		{"LengthPercentageAuto", unsafe.Sizeof(LengthPercentageAuto{}), 8},
		{"Dimension", unsafe.Sizeof(Dimension{}), 8},
		{"Size<LengthPercentage>", unsafe.Sizeof(Size[LengthPercentage]{}), 16},
		{"Size<LengthPercentageAuto>", unsafe.Sizeof(Size[LengthPercentageAuto]{}), 16},
		{"Size<Dimension>", unsafe.Sizeof(Size[Dimension]{}), 16},
		{"Rect<LengthPercentage>", unsafe.Sizeof(Rect[LengthPercentage]{}), 32},
		{"Rect<LengthPercentageAuto>", unsafe.Sizeof(Rect[LengthPercentageAuto]{}), 32},
		{"Rect<Dimension>", unsafe.Sizeof(Rect[Dimension]{}), 32},
		{"AlignContentKeyword", unsafe.Sizeof(AlignContentKeyword(0)), 1},
		{"AlignItemsKeyword", unsafe.Sizeof(AlignItemsKeyword(0)), 1},
		{"AlignmentSafety", unsafe.Sizeof(AlignmentSafety(0)), 1},
		{"AlignContent", unsafe.Sizeof(AlignContent{}), 2},
		{"AlignItems", unsafe.Sizeof(AlignItems{}), 2},
		{"FlexDirection", unsafe.Sizeof(FlexDirection(0)), 1},
		{"FlexWrap", unsafe.Sizeof(FlexWrap(0)), 1},
		{"GridAutoFlow", unsafe.Sizeof(GridAutoFlow(0)), 1},
		{"MinTrackSizingFunction", unsafe.Sizeof(MinTrackSizingFunction{}), 8},
		{"MaxTrackSizingFunction", unsafe.Sizeof(MaxTrackSizingFunction{}), 8},
		{"TrackSizingFunction", unsafe.Sizeof(TrackSizingFunction{}), 16},
		{"Vec<TrackSizingFunction>", unsafe.Sizeof([]TrackSizingFunction{}), 24},
		{"Vec<GridTemplateComponent<S>>", unsafe.Sizeof([]GridTemplateComponent[string]{}), 24},
	}
	for _, c := range cases {
		if c.got != c.expected {
			t.Errorf("Expected %s for be %d byte(s) but it was %d byte(s)", c.name, c.expected, c.got)
		}
	}
}

type coreStyleOnly struct {
	CoreStyleDefaults
	position Position
}

func (c coreStyleOnly) Position() Position {
	return c.position
}

func (c coreStyleOnly) IsContainingBlock() ContainingBlockClaims {
	return CoreStyleDefaultIsContainingBlock(c)
}

func TestStyleMod_TraitDefaults(t *testing.T) {
	var s CoreStyle = coreStyleOnly{}
	if s.BoxGenerationMode() != BoxGenerationModeNormal || s.IsBlock() || s.IsReplaced() || s.IsCompressibleReplaced() || s.BoxSizing() != BoxSizingBorderBox || s.Direction() != DirectionLtr || s.ScrollbarWidth() != 0 || s.Contain() != ContainNone {
		t.Error("CoreStyleDefaults scalars")
	}
	def := StyleDefault[string]()
	if s.Overflow() != def.Overflow || s.Inset() != def.Inset || s.Size() != def.Size || s.MinSize() != def.MinSize || s.MaxSize() != def.MaxSize || s.AspectRatio() != def.AspectRatio || s.Margin() != def.Margin || s.Padding() != def.Padding || s.Border() != def.Border {
		t.Error("CoreStyleDefaults from StyleDefault")
	}
	if s.IsContainingBlock() != ContainingBlockClaimsNone {
		t.Error("static is not a containing block")
	}
	if (coreStyleOnly{position: PositionRelative}).IsContainingBlock() != (ContainingBlockClaims{Absolute: true}) {
		t.Error("relative claims absolute")
	}
	var o OofItemStyleDefaults[string]
	if o.AlignSelf().IsSome() || o.JustifySelf().IsSome() || o.IsTable() || o.GridRow() != def.GridRow || o.GridColumn() != def.GridColumn {
		t.Error("OofItemStyleDefaults")
	}
}

func TestStyleMod_StyleRef(t *testing.T) {
	s := StyleDefault[string]()
	s.Display = DisplayNone
	s.ItemIsReplaced = true
	s.Position = PositionFixed
	s.AlignSelf = some(AlignItemsCenter)
	s.FlexGrow = 2
	s.Float = FloatLeft
	r := s.Ref()
	if r.Style() != &s || r.BoxGenerationMode() != BoxGenerationModeNone || !r.IsCompressibleReplaced() || !r.IsReplaced() || r.IsBlock() {
		t.Error("StyleRef core")
	}
	if r.IsContainingBlock() != (ContainingBlockClaims{Absolute: true, Fixed: false}) {
		t.Error("StyleRef IsContainingBlock")
	}
	if r.AlignSelf() != some(AlignItemsCenter) || r.FlexGrow() != 2 || r.Float() != FloatLeft || r.FlexShrink() != 1 {
		t.Error("StyleRef fields")
	}
	s.Display = DisplayBlock
	if !r.IsBlock() || r.BoxGenerationMode() != BoxGenerationModeNormal {
		t.Error("StyleRef reads through the pointer")
	}
}

func TestStyleMod_Enums(t *testing.T) {
	if DisplayDefault != DisplayFlex || DisplayFlex.String() != "FLEX" || DisplayFlowRoot.String() != "FLOW-ROOT" || DisplayNone.String() != "NONE" || DisplayBlock.String() != "BLOCK" || DisplayGrid.String() != "GRID" {
		t.Error("Display")
	}
	if !PositionAbsolute.IsOutOfFlow() || !PositionFixed.IsOutOfFlow() || PositionSticky.IsOutOfFlow() || PositionStatic.IsPositioned() || !PositionSticky.IsPositioned() {
		t.Error("Position")
	}
	if !ContainingBlockClaimsAll.ForPosition(PositionFixed) || ContainingBlockClaimsAll.ForPosition(PositionRelative) || ContainingBlockClaimsNone.ForPosition(PositionAbsolute) {
		t.Error("ForPosition")
	}
	if OverflowClip.IsScrollContainer() || !OverflowHidden.IsScrollContainer() || OverflowVisible.maybeIntoAutomaticMinSize().IsSome() || OverflowScroll.maybeIntoAutomaticMinSize() != some[float32](0) {
		t.Error("Overflow")
	}
	if ContainContent != ContainLayout.Union(ContainPaint) || !ContainContent.Contains(ContainLayout) || ContainPaint.Contains(ContainLayout) || !ContainContent.Intersects(ContainPaint) {
		t.Error("Contain bits")
	}
	if !ContainPaint.EstablishesIndependentFormattingContext() || ContainNone.EstablishesIndependentFormattingContext() || ContainPaint.SuppressesBaseline() || !ContainLayout.SuppressesBaseline() || !ContainPaint.ContainsScrollableOverflow() {
		t.Error("Contain predicates")
	}
	if !DirectionRtl.IsRtl() || DirectionLtr.IsRtl() {
		t.Error("Direction")
	}
}
