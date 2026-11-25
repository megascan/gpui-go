package gpui_test

import (
	"testing"

	"github.com/megascan/gpui-go"
)

func TestEqHash_PixelsHashAgreesWithEqualityOnSignedZero(t *testing.T) {
	positive := gpui.Px(0.0)
	negative := -gpui.Px(0.0)
	if positive != negative {
		t.Fatalf("%v != %v", positive, negative)
	}
	set := map[gpui.Pixels]struct{}{positive: {}}
	if _, ok := set[negative]; !ok {
		t.Fatal("hash of -0 differs from hash of +0")
	}
}

func TestEqHash_RenderGlyphParamsHashAgreesWithEqualityOnSignedZero(t *testing.T) {
	params := gpui.RenderGlyphParams{
		FontID:          gpui.FontID(0),
		GlyphID:         gpui.GlyphID(0),
		FontSize:        gpui.Px(0.0),
		SubpixelVariant: gpui.NewPointOf[uint8](0, 0),
	}
	negZero := float32(0)
	negZero = -negZero
	for _, c := range [][2]float32{{negZero, 0.0}, {0.0, negZero}, {negZero, negZero}} {
		negated := params
		negated.FontSize = gpui.Px(c[0])
		negated.ScaleFactor = c[1]
		if params != negated {
			t.Fatalf("%#v != %#v", params, negated)
		}
		set := map[gpui.RenderGlyphParams]struct{}{params: {}}
		if _, ok := set[negated]; !ok {
			t.Fatalf("hash of %#v differs from hash of %#v", negated, params)
		}
	}
}
