package gpui

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"
	"time"
)

func securityTiffManyStrips(width, height, rows uint32, inflated, strips int) []byte {
	var z bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&z, zlib.BestCompression)
	chunk := make([]byte, 1<<20)
	for left := inflated; left > 0; left -= len(chunk) {
		_, _ = zw.Write(chunk[:min(left, len(chunk))])
	}
	_ = zw.Close()
	type entry struct {
		tag, typ uint16
		count    uint32
		val      uint32
	}
	entries := []entry{{256, 4, 1, width}, {257, 4, 1, height}, {258, 3, 1, 8}, {259, 3, 1, 8}, {262, 3, 1, 1}, {273, 4, uint32(strips), 0}, {277, 3, 1, 1}, {278, 4, 1, rows}, {279, 4, uint32(strips), 0}}
	offsetsAt := uint32(8 + 2 + len(entries)*12 + 4)
	countsAt := offsetsAt + uint32(4*strips)
	dataAt := countsAt + uint32(4*strips)
	var b bytes.Buffer
	b.WriteString("II")
	_ = binary.Write(&b, binary.LittleEndian, uint16(42))
	_ = binary.Write(&b, binary.LittleEndian, uint32(8))
	_ = binary.Write(&b, binary.LittleEndian, uint16(len(entries)))
	for _, e := range entries {
		switch e.tag {
		case 273:
			e.val = offsetsAt
		case 279:
			e.val = countsAt
		}
		_ = binary.Write(&b, binary.LittleEndian, e.tag)
		_ = binary.Write(&b, binary.LittleEndian, e.typ)
		_ = binary.Write(&b, binary.LittleEndian, e.count)
		if e.typ == 3 {
			_ = binary.Write(&b, binary.LittleEndian, uint16(e.val))
			_ = binary.Write(&b, binary.LittleEndian, uint16(0))
		} else {
			_ = binary.Write(&b, binary.LittleEndian, e.val)
		}
	}
	_ = binary.Write(&b, binary.LittleEndian, uint32(0))
	for range strips {
		_ = binary.Write(&b, binary.LittleEndian, dataAt)
	}
	for range strips {
		_ = binary.Write(&b, binary.LittleEndian, uint32(z.Len()))
	}
	b.Write(z.Bytes())
	return b.Bytes()
}

func TestSecurity_FV1_TiffManyStripsBounded(t *testing.T) {
	const side, rows = 16384, 16383
	data := securityTiffManyStrips(side, side, rows, side*rows, 4096)
	start := time.Now()
	_, err := decodeStaticImage(data, ImageFormatTiff)
	elapsed := time.Since(start)
	securityExpectLimit(t, "16384x16384 TIFF with 4096 strips on one 256 MiB zlib run", err)
	if limit := securityFV1Budget(); elapsed > limit {
		t.Fatalf("strip guard inflated unused or duplicate strips: refused after %s, want under %s", elapsed, limit)
	}
	short := securityTiffManyStrips(side, side, 8192, 64, 1)
	if _, err := decodeStaticImage(short, ImageFormatTiff); !errors.Is(err, errImageLimitsExceeded) {
		t.Fatalf("TIFF declaring 1 of 2 strips: want errImageLimitsExceeded, got %v", err)
	}
	frames, err := decodeStaticImage(securityTiffManyStrips(64, 64, 16, 64*16, 4), ImageFormatTiff)
	if err != nil || len(frames) != 1 || frames[0].Buffer.Rect.Dy() != 64 {
		t.Fatalf("64x64 four-strip TIFF control: %v", err)
	}
}

func securitySvgzBomb(inflated int) []byte {
	const chunk = 1 << 20
	prefix := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><!--`)
	suffix := []byte(`--></svg>`)
	pad := bytes.Repeat([]byte{' '}, chunk)
	var deflated bytes.Buffer
	fw, _ := flate.NewWriter(&deflated, flate.BestCompression)
	_, _ = fw.Write(prefix)
	_, _ = fw.Write(pad)
	_ = fw.Flush()
	head := bytes.Clone(deflated.Bytes())
	deflated.Reset()
	_, _ = fw.Write(pad)
	_ = fw.Flush()
	body := bytes.Clone(deflated.Bytes())
	deflated.Reset()
	_, _ = fw.Write(suffix)
	_ = fw.Close()
	tail := deflated.Bytes()
	chunks := max((inflated-len(prefix)-len(suffix))/chunk, 2)
	crc := crc32.Update(0, crc32.IEEETable, prefix)
	for range chunks {
		crc = crc32.Update(crc, crc32.IEEETable, pad)
	}
	crc = crc32.Update(crc, crc32.IEEETable, suffix)
	size := uint32(len(prefix) + chunks*chunk + len(suffix))
	out := bytes.NewBuffer([]byte{0x1f, 0x8b, 8, 0, 0, 0, 0, 0, 0, 0xff})
	out.Write(head)
	for range chunks - 1 {
		out.Write(body)
	}
	out.Write(tail)
	_ = binary.Write(out, binary.LittleEndian, crc)
	_ = binary.Write(out, binary.LittleEndian, size)
	return out.Bytes()
}

func TestSecurity_FV2_SvgzBombRefused(t *testing.T) {
	r := NewSvgRenderer(nil)
	control := securitySvgzBomb(4 << 20)
	if _, err := r.ParseSvg(control); err != nil {
		t.Fatalf("4 MiB svgz control: %v", err)
	}
	bomb := securitySvgzBomb(2 << 30)
	if len(bomb) > 4<<20 {
		t.Fatalf("bomb fixture is %d bytes", len(bomb))
	}
	if _, err := r.ParseSvg(bomb); !errors.Is(err, errSvgzTooLarge) {
		t.Fatalf("2 GiB svgz: want errSvgzTooLarge, got %v", err)
	}
	nested := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><image width="8" height="8" href="data:image/svg+xml;base64,` + base64.StdEncoding.EncodeToString(bomb) + `"/></svg>`)
	if _, err := r.ParseSvg(nested); err != nil {
		t.Fatalf("SVG nesting a 2 GiB svgz: %v", err)
	}
}
