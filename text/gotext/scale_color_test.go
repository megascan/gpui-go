package gotext

import (
	"encoding/binary"
	"os"
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func buildColrCpal(baseGlyph uint16, layers [][2]uint16, colors [][4]byte) ([]byte, []byte) {
	be := binary.BigEndian
	colr := make([]byte, 14+6+4*len(layers))
	be.PutUint16(colr[2:], 1)
	be.PutUint32(colr[4:], 14)
	be.PutUint32(colr[8:], 20)
	be.PutUint16(colr[12:], uint16(len(layers)))
	be.PutUint16(colr[14:], baseGlyph)
	be.PutUint16(colr[16:], 0)
	be.PutUint16(colr[18:], uint16(len(layers)))
	for i, l := range layers {
		be.PutUint16(colr[20+i*4:], l[0])
		be.PutUint16(colr[22+i*4:], l[1])
	}
	cpal := make([]byte, 14+4*len(colors))
	be.PutUint16(cpal[2:], uint16(len(colors)))
	be.PutUint16(cpal[4:], 1)
	be.PutUint16(cpal[6:], uint16(len(colors)))
	be.PutUint32(cpal[8:], 14)
	for i, c := range colors {
		copy(cpal[14+i*4:], []byte{c[2], c[1], c[0], c[3]})
	}
	return colr, cpal
}

func TestScale_ColorOutlineLayers(t *testing.T) {
	s := newTestTextSystem(t)
	fontID := mustFontID(t, s, gpui.NewFont("IBM Plex Sans"))
	base, _ := s.GlyphForChar(fontID, 'O')
	inner, _ := s.GlyphForChar(fontID, 'o')
	colr, cpal := buildColrCpal(uint16(base), [][2]uint16{{uint16(base), 0}, {uint16(inner), 0xFFFF}}, [][4]byte{{255, 0, 0, 255}})
	loaded := s.state.loadedFonts[fontID].font
	saved := loaded.color
	loaded.color = swashColorProxy{colr: colr, cpal: cpal}
	defer func() { loaded.color = saved }()

	params := gpui.RenderGlyphParams{FontID: fontID, GlyphID: base, FontSize: 32, ScaleFactor: 1, IsEmoji: true}
	image, err := s.state.renderGlyphImage(params)
	if err != nil {
		t.Fatal(err)
	}
	if image.source != swashSourceColorOutline || image.content != swashContentColor {
		t.Fatalf("source %d content %d", image.source, image.content)
	}
	w, h := int(image.placement.Width), int(image.placement.Height)
	if w == 0 || h == 0 || len(image.data) != w*h*4 {
		t.Fatalf("placement %+v len %d", image.placement, len(image.data))
	}
	var red, grey int
	for i := 0; i+4 <= len(image.data); i += 4 {
		p := image.data[i : i+4]
		if p[3] >= 250 && p[0] >= 250 && p[1] == 0 && p[2] == 0 {
			red++
		}
		if p[3] >= 250 && p[0] >= 125 && p[0] <= 128 && p[0] == p[1] && p[1] == p[2] {
			grey++
		}
	}
	if red == 0 || grey == 0 {
		t.Fatalf("red %d grey %d", red, grey)
	}

	bounds, err := s.GlyphRasterBounds(params)
	if err != nil {
		t.Fatal(err)
	}
	size, data, err := s.RasterizeGlyph(params, bounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != int(size.Width*size.Height*4) {
		t.Fatalf("len %d size %+v", len(data), size)
	}
	for i := 0; i+4 <= len(data); i += 4 {
		if data[i+3] >= 250 && data[i+2] >= 250 && data[i] == 0 {
			return
		}
	}
	t.Fatal("rasterized colour glyph was not swapped to BGRA")
}

func TestScale_NonEmojiIgnoresColorOutline(t *testing.T) {
	s := newTestTextSystem(t)
	fontID := mustFontID(t, s, gpui.NewFont("IBM Plex Sans"))
	base, _ := s.GlyphForChar(fontID, 'O')
	colr, cpal := buildColrCpal(uint16(base), [][2]uint16{{uint16(base), 0}}, [][4]byte{{255, 0, 0, 255}})
	loaded := s.state.loadedFonts[fontID].font
	saved := loaded.color
	loaded.color = swashColorProxy{colr: colr, cpal: cpal}
	defer func() { loaded.color = saved }()
	image, err := s.state.renderGlyphImage(gpui.RenderGlyphParams{FontID: fontID, GlyphID: base, FontSize: 32, ScaleFactor: 1})
	if err != nil {
		t.Fatal(err)
	}
	if image.source != swashSourceOutline || image.content != swashContentMask {
		t.Fatalf("source %d content %d", image.source, image.content)
	}
}

func TestScale_Blit(t *testing.T) {
	target := make([]byte, 3*2*4)
	swashBlit([]byte{255, 128, 0, 255}, 2, 2, 1, -1, [4]byte{10, 20, 30, 255}, target, 3, 2)
	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 9, 19, 29, 253, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := range want {
		if target[i] != want[i] {
			t.Fatalf("target %v", target)
		}
	}
}

func TestScale_ResizeMitchellKeepsOpaqueColor(t *testing.T) {
	src := make([]byte, 4*4*4)
	for i := 0; i < len(src); i += 4 {
		copy(src[i:], []byte{200, 100, 50, 255})
	}
	dst := make([]byte, 2*2*4)
	if !swashResize(src, 4, 4, 4, dst, 2, 2, swashFilterMitchell, nil) {
		t.Fatal("resize failed")
	}
	for i := 0; i < len(dst); i += 4 {
		for c, want := range []byte{200, 100, 50, 255} {
			if d := int(dst[i+c]) - int(want); d < -1 || d > 1 {
				t.Fatalf("pixel %v", dst[i:i+4])
			}
		}
	}
}

func TestScale_ColorBitmapEmoji(t *testing.T) {
	const path = "/usr/share/fonts/noto/NotoColorEmoji.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("no colour bitmap font: %v", err)
	}
	s := NewTextSystemWithoutSystemFonts("Noto Color Emoji")
	if err := s.AddFonts([][]byte{data}); err != nil {
		t.Fatal(err)
	}
	var fontID gpui.FontID
	found := false
	for _, face := range s.state.fontSystem.db.faces {
		if face.hasFamily("Noto Color Emoji") {
			if fontID, err = s.state.fontIDForCosmicID(face.id); err != nil {
				t.Fatal(err)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("emoji face not in database")
	}
	if !s.state.loadedFonts[fontID].isKnownEmojiFont {
		t.Fatal("Noto Color Emoji is not a known emoji font")
	}
	glyph, ok := s.GlyphForChar(fontID, '\U0001F600')
	if !ok {
		t.Skip("emoji glyph missing")
	}
	params := gpui.RenderGlyphParams{FontID: fontID, GlyphID: glyph, FontSize: 16, ScaleFactor: 2, IsEmoji: true}
	image, err := s.state.renderGlyphImage(params)
	if err != nil {
		t.Fatal(err)
	}
	if image.source != swashSourceColorBitmap || image.strike != swashStrikeWithBestFit || image.content != swashContentColor {
		t.Fatalf("source %d strike %d content %d", image.source, image.strike, image.content)
	}
	w, h := int(image.placement.Width), int(image.placement.Height)
	if w < 28 || w > 40 || h < 28 || h > 40 || len(image.data) != w*h*4 {
		t.Fatalf("placement %+v len %d", image.placement, len(image.data))
	}
	var opaque int
	for i := 3; i < len(image.data); i += 4 {
		if image.data[i] == 255 {
			opaque++
		}
	}
	if opaque == 0 {
		t.Fatal("no opaque pixels")
	}
}
