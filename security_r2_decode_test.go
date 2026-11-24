package gpui

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"testing"
	"time"
)

type securityBitWriter struct {
	buf   []byte
	acc   uint64
	nbits uint
}

func (w *securityBitWriter) put(v uint64, n uint) {
	w.acc |= v << w.nbits
	w.nbits += n
	for w.nbits >= 8 {
		w.buf = append(w.buf, byte(w.acc))
		w.acc >>= 8
		w.nbits -= 8
	}
}

func securityVP8LSolid(width, height uint32) []byte {
	var w securityBitWriter
	w.put(0x2f, 8)
	w.put(uint64(width-1), 14)
	w.put(uint64(height-1), 14)
	w.put(1, 1)
	w.put(0, 3)
	w.put(0, 3)
	for range 5 {
		w.put(1, 1)
		w.put(0, 3)
	}
	if w.nbits > 0 {
		w.buf = append(w.buf, byte(w.acc))
	}
	return w.buf
}

func securityAnimatedWebp(canvas, frame uint32, bitstream []byte, frames int) []byte {
	vp8x := make([]byte, 10)
	vp8x[0] = 0x12
	vp8x[4], vp8x[5], vp8x[6] = byte(canvas-1), byte((canvas-1)>>8), byte((canvas-1)>>16)
	vp8x[7], vp8x[8], vp8x[9] = byte(canvas-1), byte((canvas-1)>>8), byte((canvas-1)>>16)
	chunks := [][]byte{webpChunkBytes("VP8X", vp8x), webpChunkBytes("ANIM", make([]byte, 6))}
	for range frames {
		hdr := make([]byte, 16)
		hdr[6], hdr[7], hdr[8] = byte(frame-1), byte((frame-1)>>8), byte((frame-1)>>16)
		hdr[9], hdr[10], hdr[11] = byte(frame-1), byte((frame-1)>>8), byte((frame-1)>>16)
		hdr[12] = 100
		chunks = append(chunks, webpChunkBytes("ANMF", append(hdr, webpChunkBytes("VP8L", bitstream)...)))
	}
	return webpFile(chunks...)
}

func TestSecurity_R2_1_WebpFrameBitstreamDims(t *testing.T) {
	frames, err := decodeWebpImage(securityAnimatedWebp(4, 4, securityVP8LSolid(4, 4), 2))
	if err != nil || len(frames) != 2 {
		t.Fatalf("4x4 animated WebP control: %d frames, %v", len(frames), err)
	}
	start := time.Now()
	_, err = decodeWebpImage(securityAnimatedWebp(1, 1, securityVP8LSolid(16384, 16384), 4))
	if err == nil {
		t.Fatal("1x1 ANMF frames carrying a 16384x16384 VP8L bitstream decoded")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("mismatched frame bitstream was decoded before being refused (%s)", elapsed)
	}
}

func securityTiffDeflateGray(width, height uint32, inflated int) []byte {
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	_, _ = zw.Write(make([]byte, inflated))
	_ = zw.Close()
	type entry struct {
		tag, typ uint16
		val      uint32
	}
	entries := []entry{{256, 4, width}, {257, 4, height}, {258, 3, 8}, {259, 3, 8}, {262, 3, 1}, {273, 4, 0}, {277, 3, 1}, {278, 4, height}, {279, 4, uint32(z.Len())}}
	dataOff := uint32(8 + 2 + len(entries)*12 + 4)
	var b bytes.Buffer
	b.WriteString("II")
	_ = binary.Write(&b, binary.LittleEndian, uint16(42))
	_ = binary.Write(&b, binary.LittleEndian, uint32(8))
	_ = binary.Write(&b, binary.LittleEndian, uint16(len(entries)))
	for _, e := range entries {
		if e.tag == 273 {
			e.val = dataOff
		}
		_ = binary.Write(&b, binary.LittleEndian, e.tag)
		_ = binary.Write(&b, binary.LittleEndian, e.typ)
		_ = binary.Write(&b, binary.LittleEndian, uint32(1))
		if e.typ == 3 {
			_ = binary.Write(&b, binary.LittleEndian, uint16(e.val))
			_ = binary.Write(&b, binary.LittleEndian, uint16(0))
		} else {
			_ = binary.Write(&b, binary.LittleEndian, e.val)
		}
	}
	_ = binary.Write(&b, binary.LittleEndian, uint32(0))
	b.Write(z.Bytes())
	return b.Bytes()
}

func TestSecurity_R2_5_TiffStripInflation(t *testing.T) {
	frames, err := decodeStaticImage(securityTiffDeflateGray(64, 64, 64*64), ImageFormatTiff)
	if err != nil || len(frames) != 1 || frames[0].Buffer.Rect.Dx() != 64 {
		t.Fatalf("64x64 deflate TIFF control: %v", err)
	}
	_, err = decodeStaticImage(securityTiffDeflateGray(64, 64, 64*64*8), ImageFormatTiff)
	securityExpectLimit(t, "TIFF strip inflating to 8 bytes per pixel", err)
}

func TestSecurity_R2_6_ProcessWideDecodeMemory(t *testing.T) {
	s := &imageDecodeSemaphore{size: 100}
	first := s.acquire(60)
	acquired := make(chan func())
	go func() { acquired <- s.acquire(60) }()
	select {
	case <-acquired:
		t.Fatal("second decode ran while the first held the in-flight budget")
	case <-time.After(50 * time.Millisecond):
	}
	first()
	first()
	select {
	case release := <-acquired:
		release()
	case <-time.After(5 * time.Second):
		t.Fatal("waiting decode was not woken by release")
	}
	oversized := s.acquire(1000)
	go func() { acquired <- s.acquire(1) }()
	select {
	case <-acquired:
		t.Fatal("decode ran beside one that takes the whole in-flight budget")
	case <-time.After(50 * time.Millisecond):
	}
	oversized()
	(<-acquired)()
	if s.used != 0 || len(s.waiters) != 0 {
		t.Fatalf("semaphore not drained: used=%d waiters=%d", s.used, len(s.waiters))
	}
}

func securityJpegSOF(marker byte, width, height uint16, comps [][3]byte, app []byte) []byte {
	out := []byte{0xff, 0xd8}
	out = append(out, app...)
	sof := []byte{0xff, marker, 0, byte(8 + 3*len(comps)), 8}
	sof = binary.BigEndian.AppendUint16(sof, height)
	sof = binary.BigEndian.AppendUint16(sof, width)
	sof = append(sof, byte(len(comps)))
	binary.BigEndian.PutUint16(sof[2:], uint16(8+3*len(comps)))
	for _, c := range comps {
		sof = append(sof, c[0], c[1]<<4|c[2], 0)
	}
	out = append(out, sof...)
	return append(out, 0xff, 0xda)
}

func securitySum(sizes []uint64) uint64 {
	var total uint64
	for _, s := range sizes {
		total += s
	}
	return total
}

func TestSecurity_BR2_LegitimateContentFitsBudget(t *testing.T) {
	for _, c := range []struct{ w, h, frames int }{{640, 480, 900}, {1920, 1080, 150}} {
		sizes := make([][2]uint64, c.frames)
		for i := range sizes {
			sizes[i] = [2]uint64{uint64(c.w), uint64(c.h)}
		}
		if _, err := reserveGifFrames(sizes, c.w, c.h); err != nil {
			t.Errorf("%dx%d GIF with %d frames refused: %v", c.w, c.h, c.frames, err)
		}
	}
	sizes := make([][2]uint64, 32)
	for i := range sizes {
		sizes[i] = [2]uint64{1, 1}
	}
	if _, err := reserveGifFrames(sizes, 6000, 6000); !errors.Is(err, errImageLimitsExceeded) {
		t.Errorf("6000x6000 GIF with 32 frames: want limit error, got %v", err)
	}

	ycc420 := [][3]byte{{1, 2, 2}, {2, 1, 1}, {3, 1, 1}}
	data := securityJpegSOF(0xc2, 9504, 6336, ycc420, nil)
	config := image.Config{Width: 9504, Height: 6336, ColorModel: color.YCbCrModel}
	decoder := &stdImageDecoder{data: data, config: config}
	var ok bool
	if decoder.bufferBytes, ok = jpegDecodeBytes(data, config); !ok {
		t.Fatal("SOF2 not parsed")
	}
	budget := newImageDecodeBudget()
	if _, _, err := decoder.reserve(budget); err != nil {
		t.Fatalf("61 MP progressive JPEG refused: %v", err)
	}
	if err := budget.reserve(9504, 6336, 4); err != nil {
		t.Fatalf("61 MP progressive JPEG RGBA output refused: %v", err)
	}

	pixels := uint64(1000 * 1000)
	cmyk, ok := jpegDecodeBytes(securityJpegSOF(0xc0, 1000, 1000, [][3]byte{{1, 1, 1}, {2, 1, 1}, {3, 1, 1}, {4, 1, 1}}, nil), image.Config{Width: 1000, Height: 1000})
	if !ok || securitySum(cmyk) < 8*pixels {
		t.Errorf("CMYK JPEG estimate %d bytes, want at least 8 B/px (4 planes + CMYK image)", securitySum(cmyk))
	}
	adobe := []byte{0xff, 0xee, 0, 14, 'A', 'd', 'o', 'b', 'e', 0, 100, 0, 0, 0, 0, 0}
	rgb, ok := jpegDecodeBytes(securityJpegSOF(0xc0, 1000, 1000, [][3]byte{{1, 1, 1}, {2, 1, 1}, {3, 1, 1}}, adobe), image.Config{Width: 1000, Height: 1000})
	if !ok || securitySum(rgb) < 7*pixels {
		t.Errorf("Adobe RGB JPEG estimate %d bytes, want at least 7 B/px (3 planes + RGBA image)", securitySum(rgb))
	}

	lossless := &webpDecoder{width: 1000, height: 1000, lossless: true}
	budget = newImageDecodeBudget()
	if _, _, err := lossless.reserve(budget); err != nil || budget.reserved != 8*pixels {
		t.Errorf("lossless WebP reserved %d bytes (%v), want 8 B/px", budget.reserved, err)
	}

	var buf bytes.Buffer
	pal := color.Palette{color.Black, color.White}
	g := &gif.GIF{Config: image.Config{ColorModel: pal, Width: 8, Height: 8}}
	for range 3 {
		p := image.NewPaletted(image.Rect(2, 2, 6, 6), pal)
		p.Pix[0] = 1
		g.Image = append(g.Image, p)
		g.Delay = append(g.Delay, 1)
	}
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatal(err)
	}
	frames, err := decodeGifImage(buf.Bytes())
	if err != nil || len(frames) != 3 {
		t.Fatalf("offset GIF: %d frames, %v", len(frames), err)
	}
	if px := frames[0].Buffer.Pix[frames[0].Buffer.PixOffset(2, 2):][:4]; px[0] != 255 || px[3] != 255 {
		t.Fatalf("offset GIF frame pixel at (2,2) = %v, want white", px)
	}
}
