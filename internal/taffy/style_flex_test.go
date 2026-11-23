package taffy

import "testing"

func TestStyleFlex_FlexDirection_FlexDirectionIsRow(t *testing.T) {
	if !FlexDirectionRow.isRow() || !FlexDirectionRowReverse.isRow() || FlexDirectionColumn.isRow() || FlexDirectionColumnReverse.isRow() {
		t.Error("isRow")
	}
}

func TestStyleFlex_FlexDirection_FlexDirectionIsColumn(t *testing.T) {
	if FlexDirectionRow.isColumn() || FlexDirectionRowReverse.isColumn() || !FlexDirectionColumn.isColumn() || !FlexDirectionColumnReverse.isColumn() {
		t.Error("isColumn")
	}
}

func TestStyleFlex_FlexDirection_FlexDirectionIsReverse(t *testing.T) {
	if FlexDirectionRow.isReverse() || !FlexDirectionRowReverse.isReverse() || FlexDirectionColumn.isReverse() || !FlexDirectionColumnReverse.isReverse() {
		t.Error("isReverse")
	}
}

func TestStyleFlex_AxesAndWrap(t *testing.T) {
	if FlexDirectionRowReverse.mainAxis() != AbsoluteAxisHorizontal || FlexDirectionRowReverse.crossAxis() != AbsoluteAxisVertical {
		t.Error("row axes")
	}
	if FlexDirectionColumn.mainAxis() != AbsoluteAxisVertical || FlexDirectionColumn.crossAxis() != AbsoluteAxisHorizontal {
		t.Error("column axes")
	}
	if FlexWrapNoWrap.isMultiLine() || !FlexWrapWrap.isMultiLine() || !FlexWrapWrapReverse.isMultiLine() {
		t.Error("isMultiLine")
	}
	if FlexWrapWrap.isReverse() || !FlexWrapWrapReverse.isReverse() {
		t.Error("isReverse")
	}
	var d FlexboxContainerStyleDefaults
	if d.FlexDirection() != FlexDirectionRow || d.FlexWrap() != FlexWrapNoWrap || d.AlignItems() != AlignItemsNormal || d.JustifyContent() != AlignContentNormal || d.AlignContent() != AlignContentNormal || d.Gap().Width != LengthPercentageZero {
		t.Error("FlexboxContainerStyleDefaults")
	}
	var i FlexboxItemStyleDefaults
	if i.FlexBasis() != DimensionAuto() || i.FlexGrow() != 0 || i.FlexShrink() != 1 || i.AlignSelf().IsSome() {
		t.Error("FlexboxItemStyleDefaults")
	}
}
