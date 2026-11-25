package gotext

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/text/gotext/internal/zeno"
)

func securityTableDir(data []byte) map[string][2]int {
	out := map[string][2]int{}
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := range n {
		r := data[12+16*i:]
		out[string(r[:4])] = [2]int{int(binary.BigEndian.Uint32(r[8:])), int(binary.BigEndian.Uint32(r[12:]))}
	}
	return out
}

func securityDecreasingLoca(data []byte) []byte {
	mut := bytes.Clone(data)
	tb := securityTableDir(mut)
	head, loca := tb["head"][0], tb["loca"][0]
	if binary.BigEndian.Uint16(mut[head+50:]) == 1 {
		a := binary.BigEndian.Uint32(mut[loca+8:])
		binary.BigEndian.PutUint32(mut[loca+4:], a+64)
	} else {
		a := binary.BigEndian.Uint16(mut[loca+4:])
		binary.BigEndian.PutUint16(mut[loca+2:], a+32)
	}
	return mut
}

func securityTinyImage() image.Image {
	return image.NewNRGBA(image.Rect(0, 0, 2, 2))
}

func TestSecurity_SEC008_PanickingFaceIsUnloadable(t *testing.T) {
	broken := securityDecreasingLoca(ibmPlex)
	db := fontDatabase{}
	db.loadFontData(broken)
	if len(db.faces) != 1 {
		t.Fatalf("faces = %d, want 1", len(db.faces))
	}
	if _, ok := newCosmicFont(&db, 0); ok {
		t.Fatalf("face with decreasing loca loaded")
	}

	ts := NewTextSystemWithoutSystemFonts("")
	if err := ts.AddFonts([][]byte{broken}); err != nil {
		t.Fatalf("AddFonts: %v", err)
	}
	if _, err := ts.FontID(gpui.Font{Family: "IBM Plex Sans", Weight: gpui.FontWeightNormal}); err == nil {
		t.Fatalf("FontID of a panicking face returned no error")
	}

	fb := NewTextSystemWithoutSystemFonts("")
	if err := fb.AddFonts([][]byte{lilex, broken}); err != nil {
		t.Fatalf("AddFonts: %v", err)
	}
	id, err := fb.FontID(gpui.Font{Family: "Lilex", Weight: gpui.FontWeightNormal})
	if err != nil {
		t.Fatalf("FontID: %v", err)
	}
	text := "ab\U0010FFFD中"
	layout := fb.LayoutLine(text, 16, []gpui.FontRun{{Len: len(text), FontID: id}})
	if len(layout.Runs) == 0 {
		t.Fatalf("fallback layout produced no runs")
	}
}

func TestSecurity_SEC009_GlyphSizeNoWrap(t *testing.T) {
	if _, ok := glyphImageBytes(32768, 32768, 4); ok {
		t.Fatalf("32768x32768 RGBA accepted (w*h*4 wraps to 0 in uint32)")
	}
	if _, ok := glyphImageBytes(maxGlyphImageDim+1, 1, 1); ok {
		t.Fatalf("over-wide glyph accepted")
	}
	if n, ok := glyphImageBytes(64, 32, 4); !ok || n != 64*32*4 {
		t.Fatalf("small glyph = %d, %v", n, ok)
	}
	if _, ok := glyphDim(1e12); ok {
		t.Fatalf("huge scaled dimension accepted")
	}
	if n := zeno.FormatSubpixel.BufferSize(32768, 32768); n != 32768*32768*4 {
		t.Fatalf("zeno BufferSize wrapped: %d", n)
	}
	if _, ok := maskBufferSize(zeno.FormatAlpha, 65535, 65535); ok {
		t.Fatalf("65535x65535 mask accepted")
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, securityTinyImage()); err != nil {
		t.Fatal(err)
	}
	pngData := buf.Bytes()
	binary.BigEndian.PutUint32(pngData[16:], 32768)
	binary.BigEndian.PutUint32(pngData[20:], 32768)
	binary.BigEndian.PutUint32(pngData[29:], crc32.ChecksumIEEE(pngData[12:29]))
	data := swashBitmapData{data: pngData, ppem: 16, isPng: true}
	if _, ok := newSwashBitmap(&data, 1000, false); ok {
		t.Fatalf("CBDT PNG with IHDR 32768x32768 accepted")
	}
	if _, _, ok := decodePng(pngData, nil); ok {
		t.Fatalf("decodePng accepted 32768x32768 into an empty target")
	}
	small := swashBitmap{format: swashBitmapFormat{kind: swashBitmapFormatPng}, ppem: 1, width: 2, height: 2}
	if _, _, _, ok := small.scaledSize(1e6); ok {
		t.Fatalf("1e6x upscaled strike accepted")
	}
}

func TestSecurity_SEC010_StrikeCountBounded(t *testing.T) {
	table := make([]byte, 8+2*48)
	binary.BigEndian.PutUint32(table[4:], 0xFFFFFFFF)
	if s := newSwashBitmapStrikes(table, nil, 1000, false, false); s.len != 2 {
		t.Fatalf("CBLC strike count = %d, want 2", s.len)
	}
	if s := newSwashBitmapStrikes(table, nil, 1000, true, false); s.len != (len(table)-8)/4 {
		t.Fatalf("sbix strike count = %d", s.len)
	}

	d := make([]byte, 256)
	const strikeBase = 8
	const arrayOffset = 64
	binary.BigEndian.PutUint32(d[strikeBase:], arrayOffset)
	binary.BigEndian.PutUint32(d[strikeBase+8:], 1)
	binary.BigEndian.PutUint16(d[strikeBase+40:], 0)
	binary.BigEndian.PutUint16(d[strikeBase+42:], 10)
	d[strikeBase+45] = 16
	d[strikeBase+46] = 8
	binary.BigEndian.PutUint16(d[arrayOffset:], 0)
	binary.BigEndian.PutUint16(d[arrayOffset+2:], 10)
	binary.BigEndian.PutUint32(d[arrayOffset+4:], 8)
	binary.BigEndian.PutUint16(d[arrayOffset+8:], 4)
	binary.BigEndian.PutUint16(d[arrayOffset+10:], 5)
	binary.BigEndian.PutUint32(d[arrayOffset+16:], 1)
	done := make(chan struct{})
	go func() {
		getLocation(d, strikeBase, false, 5)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("index format 4 lookup with numGlyphs=1 still looping after 5s")
	}
}

func TestSecurity_SEC011_BitmapDecodeLength(t *testing.T) {
	for _, tc := range []struct {
		name string
		bits uint8
		size uint32
		w, h uint8
		want bool
	}{
		{"8bit_exact", 8, 4, 2, 2, true},
		{"8bit_short", 8, 3, 2, 2, false},
		{"8bit_long", 8, 5, 2, 2, false},
		{"1bit_short", 1, 1, 16, 16, false},
		{"4bit_short", 4, 1, 4, 4, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 64)
			loc := swashLocation{format: 5, size: tc.size, ppem: 16, bitDepth: tc.bits, width: tc.w, height: tc.h}
			bitmap, ok := loc.get(data, 1000, false)
			if !ok {
				t.Fatalf("location rejected")
			}
			target := resizeBytes(nil, bitmap.format.bufferSize(bitmap.width, bitmap.height))
			var got bool
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("decode panicked: %v", r)
					}
				}()
				got = bitmap.decode(target)
			}()
			if got != tc.want {
				t.Fatalf("decode = %v, want %v", got, tc.want)
			}
		})
	}
}
