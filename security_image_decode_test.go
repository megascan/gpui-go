package gpui

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"testing"
)

func securityPNGChunk(kind string, data []byte) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, uint32(len(data)))
	b.WriteString(kind)
	b.Write(data)
	crc := crc32.NewIEEE()
	crc.Write([]byte(kind))
	crc.Write(data)
	_ = binary.Write(&b, binary.BigEndian, crc.Sum32())
	return b.Bytes()
}

func securityPNGWithDims(width, height uint32) []byte {
	var idat bytes.Buffer
	w := zlib.NewWriter(&idat)
	_, _ = w.Write([]byte{0, 0, 0, 0, 0})
	_ = w.Close()
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], width)
	binary.BigEndian.PutUint32(ihdr[4:], height)
	ihdr[8], ihdr[9] = 8, 6
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	b.Write(securityPNGChunk("IHDR", ihdr))
	b.Write(securityPNGChunk("IDAT", idat.Bytes()))
	b.Write(securityPNGChunk("IEND", nil))
	return b.Bytes()
}

func securityJPEGWithDims(t *testing.T, width, height uint16) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	i := bytes.Index(data, []byte{0xff, 0xc0})
	if i < 0 {
		t.Fatal("no SOF0 marker")
	}
	binary.BigEndian.PutUint16(data[i+5:], height)
	binary.BigEndian.PutUint16(data[i+7:], width)
	return data
}

func securityExpectLimit(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, errImageLimitsExceeded) {
		t.Fatalf("%s: want errImageLimitsExceeded, got %v", what, err)
	}
}

func TestSecurity_SEC001_RasterDecodeBudget(t *testing.T) {
	_, err := decodeStaticImage(securityJPEGWithDims(t, 65535, 65535), ImageFormatJpeg)
	securityExpectLimit(t, "JPEG 65535x65535", err)
	_, err = decodeStaticImage(securityPNGWithDims(30000, 30000), ImageFormatPng)
	securityExpectLimit(t, "PNG 30000x30000", err)
	_, err = decodeStaticImage([]byte("P5\n65535 65535\n255\n\x00\x00\x00\x00"), ImageFormatPnm)
	securityExpectLimit(t, "PGM 65535x65535", err)

	p := securityPNGWithDims(30000, 30000)
	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0})
	entry := make([]byte, 16)
	binary.LittleEndian.PutUint16(entry[4:], 1)
	binary.LittleEndian.PutUint16(entry[6:], 32)
	binary.LittleEndian.PutUint32(entry[8:], uint32(len(p)))
	binary.LittleEndian.PutUint32(entry[12:], 22)
	ico.Write(entry)
	ico.Write(p)
	_, err = decodeStaticImage(ico.Bytes(), ImageFormatIco)
	securityExpectLimit(t, "ICO embedding PNG 30000x30000", err)

	frames, err := decodeStaticImage(securityPNGWithDims(1, 1), ImageFormatPng)
	if err != nil || len(frames) != 1 {
		t.Fatalf("1x1 PNG: got %d frames", len(frames))
	}
}

func TestSecurity_SEC001_ProgressiveJpegDetection(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	i := bytes.Index(data, []byte{0xff, 0xc0})
	if jpegIsProgressive(data) {
		t.Fatal("baseline JPEG reported as progressive")
	}
	data[i+1] = 0xc2
	padded := append(append(bytes.Clone(data[:i]), 0x00, 0xff, 0x00, 0xff), data[i:]...)
	if _, err := jpeg.DecodeConfig(bytes.NewReader(padded)); err != nil {
		t.Fatalf("padded progressive JPEG is not accepted by image/jpeg: %v", err)
	}
	if !jpegIsProgressive(padded) {
		t.Fatal("progressive JPEG with extraneous bytes before SOF2 not detected")
	}
	binary.BigEndian.PutUint16(padded[i+4+5:], 12000)
	binary.BigEndian.PutUint16(padded[i+4+7:], 12000)
	_, err := decodeStaticImage(padded, ImageFormatJpeg)
	securityExpectLimit(t, "progressive JPEG 12000x12000", err)
	baseline := bytes.Clone(padded)
	baseline[i+4+1] = 0xc0
	decoder, err := newImageDecoder(baseline, ImageFormatJpeg)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := decoder.reserve(newImageDecodeBudget()); err != nil {
		t.Fatalf("baseline JPEG 12000x12000 rejected: %v", err)
	}
}

func TestSecurity_SEC002_PnmDimensionOverflow(t *testing.T) {
	for _, data := range []string{
		"P7\nWIDTH 4294967295\nHEIGHT 4294967295\nDEPTH 1\nMAXVAL 255\nTUPLTYPE GRAYSCALE\nENDHDR\n\x00",
		"P7\nWIDTH 2147483648\nHEIGHT 2147483648\nDEPTH 4\nMAXVAL 255\nTUPLTYPE RGB_ALPHA\nENDHDR\n\x00",
	} {
		_, err := decodeStaticImage([]byte(data), ImageFormatPnm)
		securityExpectLimit(t, "PAM header", err)
		decoder, err := newPnmDecoder([]byte(data))
		if err != nil {
			t.Fatal(err)
		}
		_, err = decoder.decode()
		securityExpectLimit(t, "PAM decode", err)
	}
	img, err := decodeStaticImage([]byte("P7\nWIDTH 2\nHEIGHT 1\nDEPTH 1\nMAXVAL 255\nTUPLTYPE GRAYSCALE\nENDHDR\n\x10\x20"), ImageFormatPnm)
	if err != nil || len(img) != 1 || img[0].Buffer.Rect.Dx() != 2 {
		t.Fatalf("valid PAM: %v %v", img, err)
	}
}

func TestSecurity_SEC003_AnimatedFrameBudget(t *testing.T) {
	pal := color.Palette{color.Black, color.White}
	var screen bytes.Buffer
	if err := gif.EncodeAll(&screen, &gif.GIF{
		Image:  []*image.Paletted{image.NewPaletted(image.Rect(0, 0, 1, 1), pal)},
		Delay:  []int{0},
		Config: image.Config{ColorModel: pal, Width: 65535, Height: 65535},
	}); err != nil {
		t.Fatal(err)
	}
	_, err := decodeGifImage(screen.Bytes())
	securityExpectLimit(t, "GIF 65535x65535 screen", err)

	many := &gif.GIF{Config: image.Config{ColorModel: pal, Width: 6000, Height: 6000}}
	for range 32 {
		many.Image = append(many.Image, image.NewPaletted(image.Rect(0, 0, 1, 1), pal))
		many.Delay = append(many.Delay, 1)
	}
	var frames bytes.Buffer
	if err := gif.EncodeAll(&frames, many); err != nil {
		t.Fatal(err)
	}
	if got := len(gifFrameSizes(frames.Bytes())); got != 32 {
		t.Fatalf("gifFrameSizes: got %d frames, want 32", got)
	}
	_, err = decodeGifImage(frames.Bytes())
	securityExpectLimit(t, "GIF 6000x6000 with 32 frames", err)

	small := &gif.GIF{Config: image.Config{ColorModel: pal, Width: 4, Height: 4}}
	for range 3 {
		small.Image = append(small.Image, image.NewPaletted(image.Rect(0, 0, 4, 4), pal))
		small.Delay = append(small.Delay, 1)
	}
	var ok bytes.Buffer
	if err := gif.EncodeAll(&ok, small); err != nil {
		t.Fatal(err)
	}
	if decoded, err := decodeGifImage(ok.Bytes()); err != nil || len(decoded) != 3 {
		t.Fatalf("small GIF: %d frames, %v", len(decoded), err)
	}

	webp := &webpDecoder{width: 11000, height: 11000, animated: true, frameData: make([]webpChunk, 10)}
	_, err = decodeAnimatedFrames(webp.frames(), "WebP")
	securityExpectLimit(t, "WebP 11000x11000 with 10 frames", err)
}

func TestSecurity_SEC031_ImageIDKeyed(t *testing.T) {
	data := securityPNGWithDims(1, 1)
	a := ImageFromBytes(ImageFormatPng, data)
	if a.ID == Hash(data) {
		t.Fatalf("Image.ID is the unkeyed FxHash of the bytes")
	}
	if b := ImageFromBytes(ImageFormatPng, bytes.Clone(data)); b.ID != a.ID {
		t.Fatalf("equal bytes give different IDs: %#x %#x", a.ID, b.ID)
	}
	if c := ImageFromBytes(ImageFormatGif, data); c.ID == a.ID {
		t.Fatalf("format is not part of Image.ID")
	}
	if d := ImageFromBytes(ImageFormatPng, append(bytes.Clone(data), 0)); d.ID == a.ID {
		t.Fatalf("distinct bytes share Image.ID")
	}
}
