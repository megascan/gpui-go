package gotext

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func rasterParams(t *testing.T, s *CosmicTextSystem, ch rune, subpixel bool, variant uint8) gpui.RenderGlyphParams {
	t.Helper()
	fontID := mustFontID(t, s, gpui.NewFont("IBM Plex Sans"))
	glyph, ok := s.GlyphForChar(fontID, ch)
	if !ok {
		t.Fatalf("no glyph for %q", ch)
	}
	return gpui.RenderGlyphParams{
		FontID:            fontID,
		GlyphID:           glyph,
		FontSize:          16,
		SubpixelVariant:   gpui.PointOf[uint8]{X: variant},
		ScaleFactor:       2,
		SubpixelRendering: subpixel,
	}
}

func TestCosmicTextSystem_RasterizeGlyphAlphaMask(t *testing.T) {
	s := newTestTextSystem(t)
	params := rasterParams(t, s, 'm', false, 0)
	bounds, err := s.GlyphRasterBounds(params)
	if err != nil {
		t.Fatal(err)
	}
	if bounds.Size.Width < 20 || bounds.Size.Width > 30 || bounds.Size.Height < 14 || bounds.Size.Height > 20 {
		t.Fatalf("bounds %+v", bounds)
	}
	if bounds.Origin.Y >= 0 || -bounds.Origin.Y < bounds.Size.Height-1 {
		t.Fatalf("origin %+v", bounds)
	}
	if _, ok := s.state.pendingGlyphImages[params]; !ok {
		t.Fatal("bounds did not cache the image")
	}
	size, data, err := s.RasterizeGlyph(params, bounds)
	if err != nil {
		t.Fatal(err)
	}
	if size != bounds.Size || len(data) != int(size.Width*size.Height) {
		t.Fatalf("size %+v len %d", size, len(data))
	}
	if _, ok := s.state.pendingGlyphImages[params]; ok {
		t.Fatal("rasterize did not consume the cached image")
	}
	var full int
	for _, b := range data {
		if b == 255 {
			full++
		}
	}
	if full == 0 {
		t.Fatal("no opaque coverage")
	}
	_, again, err := s.RasterizeGlyph(params, bounds)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(data) {
		t.Fatal("re-render differs from cached image")
	}
}

func TestCosmicTextSystem_RasterizeGlyphSubpixelMask(t *testing.T) {
	s := newTestTextSystem(t)
	params := rasterParams(t, s, 'l', true, 0)
	bounds, err := s.GlyphRasterBounds(params)
	if err != nil {
		t.Fatal(err)
	}
	size, data, err := s.RasterizeGlyph(params, bounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != int(size.Width*size.Height)*4 {
		t.Fatalf("len %d size %+v", len(data), size)
	}
	w := int(size.Width)
	var b, r int
	differs := false
	for y := range int(size.Height) {
		for x := range w {
			p := data[(y*w+x)*4:]
			if p[3] != 0 {
				t.Fatalf("alpha channel written at %d,%d", x, y)
			}
			if p[0] != p[2] {
				differs = true
			}
			if x == 0 {
				b += int(p[0])
				r += int(p[2])
			}
		}
	}
	if !differs {
		t.Fatal("channels identical; no subpixel offsets")
	}
	if b <= r {
		t.Fatalf("leftmost column: byte 0 (the -0.3 pass after the B/R swap) %d should exceed byte 2 %d", b, r)
	}
}

func TestCosmicTextSystem_RasterizeGlyphSubpixelVariantShiftsBounds(t *testing.T) {
	s := newTestTextSystem(t)
	a, err := s.GlyphRasterBounds(rasterParams(t, s, 'l', false, 0))
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.GlyphRasterBounds(rasterParams(t, s, 'l', false, 3))
	if err != nil {
		t.Fatal(err)
	}
	if b.Origin.X < a.Origin.X || b.Origin.X+b.Size.Width < a.Origin.X+a.Size.Width {
		t.Fatalf("variant 0 %+v, variant 3 %+v", a, b)
	}
}

func TestCosmicTextSystem_RasterizeEmptyGlyph(t *testing.T) {
	s := newTestTextSystem(t)
	params := rasterParams(t, s, ' ', false, 0)
	bounds, err := s.GlyphRasterBounds(params)
	if err != nil {
		t.Fatal(err)
	}
	if !bounds.IsZero() {
		t.Fatalf("bounds %+v", bounds)
	}
	if len(s.state.pendingGlyphImages) != 0 {
		t.Fatal("empty glyph was cached")
	}
	if _, _, err := s.RasterizeGlyph(params, bounds); err == nil || err.Error() != "glyph bounds are empty" {
		t.Fatalf("err %v", err)
	}
}
