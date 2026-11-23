package taffy

import (
	"testing"
	"unsafe"
)

func TestStyleAlignment_AlignTypesWithinSizeBudget(t *testing.T) {
	if s := unsafe.Sizeof(AlignItems{}); s > 2 {
		t.Errorf("AlignItems grew to %d", s)
	}
	if s := unsafe.Sizeof(AlignContent{}); s > 2 {
		t.Errorf("AlignContent grew to %d", s)
	}
	if unsafe.Sizeof(Option[AlignItems]{}) > 3 {
		t.Error("Option[AlignItems] grew")
	}
}

func TestStyleAlignment_AlignItemsIsSafe(t *testing.T) {
	for _, a := range []AlignItems{AlignItemsSafeStart, AlignItemsSafeEnd, AlignItemsSafeFlexStart, AlignItemsSafeFlexEnd, AlignItemsSafeCenter, AlignItemsSafeSelfStart, AlignItemsSafeSelfEnd} {
		if !a.IsSafe() {
			t.Errorf("%v should be safe", a)
		}
	}
	for _, a := range []AlignItems{AlignItemsStart, AlignItemsEnd, AlignItemsFlexStart, AlignItemsFlexEnd, AlignItemsCenter, AlignItemsBaseline, AlignItemsStretch, AlignItemsSelfStart, AlignItemsSelfEnd, AlignItemsNormal} {
		if a.IsSafe() {
			t.Errorf("%v should not be safe", a)
		}
	}
}

func TestStyleAlignment_AlignItemsKeywordStripsSafe(t *testing.T) {
	cases := []struct {
		a AlignItems
		k AlignItemsKeyword
	}{
		{AlignItemsSafeStart, AlignItemsKeywordStart},
		{AlignItemsSafeEnd, AlignItemsKeywordEnd},
		{AlignItemsSafeFlexStart, AlignItemsKeywordFlexStart},
		{AlignItemsSafeFlexEnd, AlignItemsKeywordFlexEnd},
		{AlignItemsSafeCenter, AlignItemsKeywordCenter},
		{AlignItemsSafeSelfStart, AlignItemsKeywordSelfStart},
		{AlignItemsSafeSelfEnd, AlignItemsKeywordSelfEnd},
	}
	for _, c := range cases {
		if c.a.Keyword() != c.k {
			t.Errorf("%v.Keyword() = %v, want %v", c.a, c.a.Keyword(), c.k)
		}
	}
}

func TestStyleAlignment_AlignItemsKeywordPassthrough(t *testing.T) {
	cases := []struct {
		a AlignItems
		k AlignItemsKeyword
	}{
		{AlignItemsStart, AlignItemsKeywordStart},
		{AlignItemsStretch, AlignItemsKeywordStretch},
		{AlignItemsBaseline, AlignItemsKeywordBaseline},
		{AlignItemsFlexStart, AlignItemsKeywordFlexStart},
		{AlignItemsNormal, AlignItemsKeywordNormal},
	}
	for _, c := range cases {
		if c.a.Keyword() != c.k {
			t.Errorf("%v.Keyword() = %v, want %v", c.a, c.a.Keyword(), c.k)
		}
	}
}

func TestStyleAlignment_AlignItemsDefaultIsNormal(t *testing.T) {
	if AlignItemsDefault != AlignItemsNormal || (AlignItems{}) != AlignItemsNormal {
		t.Error("AlignItems default")
	}
	if AlignItemsKeywordDefault != AlignItemsKeywordNormal {
		t.Error("AlignItemsKeyword default")
	}
	var s AlignmentSafety
	if s != AlignmentSafetyDefault {
		t.Error("AlignmentSafety default")
	}
}

func TestStyleAlignment_ResolveSelfRelativeInlineAxis(t *testing.T) {
	ltr, rtl := DirectionLtr, DirectionRtl
	cases := []struct {
		got, want AlignItems
	}{
		{AlignItemsSelfStart.ResolveSelfRelative(ltr, ltr, true), AlignItemsStart},
		{AlignItemsSelfEnd.ResolveSelfRelative(ltr, ltr, true), AlignItemsEnd},
		{AlignItemsSelfStart.ResolveSelfRelative(rtl, rtl, true), AlignItemsStart},
		{AlignItemsSelfEnd.ResolveSelfRelative(rtl, rtl, true), AlignItemsEnd},
		{AlignItemsSelfStart.ResolveSelfRelative(ltr, rtl, true), AlignItemsEnd},
		{AlignItemsSelfEnd.ResolveSelfRelative(ltr, rtl, true), AlignItemsStart},
		{AlignItemsSelfStart.ResolveSelfRelative(rtl, ltr, true), AlignItemsEnd},
		{AlignItemsSelfEnd.ResolveSelfRelative(rtl, ltr, true), AlignItemsStart},
		{AlignItemsSafeSelfStart.ResolveSelfRelative(ltr, rtl, true), AlignItemsSafeEnd},
		{AlignItemsStart.ResolveSelfRelative(ltr, rtl, true), AlignItemsStart},
		{AlignItemsFlexEnd.ResolveSelfRelative(ltr, rtl, true), AlignItemsFlexEnd},
		{AlignItemsNormal.ResolveSelfRelative(ltr, rtl, true), AlignItemsNormal},
	}
	for i, c := range cases {
		if c.got != c.want {
			t.Errorf("case %d: got %v, want %v", i, c.got, c.want)
		}
	}
}

func TestStyleAlignment_ResolveSelfRelativeBlockAxis(t *testing.T) {
	if AlignItemsSelfStart.ResolveSelfRelative(DirectionLtr, DirectionRtl, false) != AlignItemsStart {
		t.Error("self-start block axis")
	}
	if AlignItemsSelfEnd.ResolveSelfRelative(DirectionRtl, DirectionLtr, false) != AlignItemsEnd {
		t.Error("self-end block axis")
	}
}

func TestStyleAlignment_AlignContentIsSafe(t *testing.T) {
	if !AlignContentSafeStart.IsSafe() || !AlignContentSafeCenter.IsSafe() {
		t.Error("safe values")
	}
	if AlignContentSpaceBetween.IsSafe() || AlignContentStretch.IsSafe() || AlignContentNormal.IsSafe() {
		t.Error("unsafe values")
	}
}

func TestStyleAlignment_AlignContentKeywordStripsSafe(t *testing.T) {
	if AlignContentSafeStart.Keyword() != AlignContentKeywordStart ||
		AlignContentSafeFlexEnd.Keyword() != AlignContentKeywordFlexEnd ||
		AlignContentSafeCenter.Keyword() != AlignContentKeywordCenter ||
		AlignContentSpaceBetween.Keyword() != AlignContentKeywordSpaceBetween ||
		AlignContentNormal.Keyword() != AlignContentKeywordNormal {
		t.Error("Keyword")
	}
	var jc JustifyContent = AlignContentNormal
	if jc != AlignContentNormal {
		t.Error("JustifyContent alias")
	}
}

func TestStyleAlignment_AlignContentKeywordReversedSwapsStartEnd(t *testing.T) {
	cases := [][2]AlignContentKeyword{
		{AlignContentKeywordStart, AlignContentKeywordEnd},
		{AlignContentKeywordEnd, AlignContentKeywordStart},
		{AlignContentKeywordFlexStart, AlignContentKeywordFlexEnd},
		{AlignContentKeywordFlexEnd, AlignContentKeywordFlexStart},
		{AlignContentKeywordStretch, AlignContentKeywordEnd},
		{AlignContentKeywordNormal, AlignContentKeywordNormal},
		{AlignContentKeywordCenter, AlignContentKeywordCenter},
		{AlignContentKeywordSpaceBetween, AlignContentKeywordSpaceBetween},
		{AlignContentKeywordSpaceEvenly, AlignContentKeywordSpaceEvenly},
		{AlignContentKeywordSpaceAround, AlignContentKeywordSpaceAround},
	}
	for _, c := range cases {
		if c[0].reversed() != c[1] {
			t.Errorf("%v.reversed() = %v, want %v", c[0], c[0].reversed(), c[1])
		}
	}
}

func TestStyleAlignment_AlignContentDefaultIsNormal(t *testing.T) {
	if AlignContentDefault != AlignContentNormal || (AlignContent{}) != AlignContentNormal {
		t.Error("AlignContent default")
	}
	var jc JustifyContent
	if jc != AlignContentNormal {
		t.Error("JustifyContent default")
	}
}

func TestStyleAlignment_Constructors(t *testing.T) {
	a := NewAlignItems(AlignItemsKeywordCenter, AlignmentSafetyUnsafe)
	if a.Keyword() != AlignItemsKeywordCenter || a.Safety() != AlignmentSafetyUnsafe || a != AlignItemsCenter.WithSafety(AlignmentSafetyUnsafe) {
		t.Error("NewAlignItems")
	}
	c := NewAlignContent(AlignContentKeywordEnd, AlignmentSafetySafe)
	if c != AlignContentSafeEnd || c.Safety() != AlignmentSafetySafe || AlignContentEnd.WithSafety(AlignmentSafetySafe) != c {
		t.Error("NewAlignContent")
	}
	if !AlignItemsStretch.IsStretchOrNormal() || !AlignItemsNormal.IsStretchOrNormal() || AlignItemsStart.IsStretchOrNormal() {
		t.Error("IsStretchOrNormal")
	}
}
