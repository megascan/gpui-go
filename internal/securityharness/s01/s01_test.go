//go:build securityharness

package s01

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
	"github.com/megascan/gpui-go/scheduler"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

const childTimeout = 50 * time.Second

func decodeFile(format gpui.ImageFormat) func(string) (string, error) {
	return func(path string) (string, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("harness read: %w", err)
		}
		return decodeBytes(format, data)
	}
}

func decodeBytes(format gpui.ImageFormat, data []byte) (string, error) {
	img := gpui.ImageFromBytes(format, data)
	r, err := img.ToImageData(gpui.NewSvgRenderer(nil))
	if err != nil {
		return "", err
	}
	s := r.Size(0)
	return fmt.Sprintf("decoded frames=%d size=%dx%d", r.FrameCount(), s.Width, s.Height), nil
}

func writeFixture(t *testing.T, name string, data []byte) string {
	t.Helper()
	if bombchild.IsChild(t) {
		return ""
	}
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("fixture %s: %d bytes", name, len(data))
	return p
}

func pngChunk(kind string, data []byte) []byte {
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

func pngWithDims(width, height uint32, colorType byte, idat []byte) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], width)
	binary.BigEndian.PutUint32(ihdr[4:], height)
	ihdr[8] = 8
	ihdr[9] = colorType
	b.Write(pngChunk("IHDR", ihdr))
	b.Write(pngChunk("IDAT", idat))
	b.Write(pngChunk("IEND", nil))
	return b.Bytes()
}

func zeroGrayIDAT(width, height int) []byte {
	var b bytes.Buffer
	w, _ := zlib.NewWriterLevel(&b, zlib.BestCompression)
	row := make([]byte, width+1)
	for range height {
		_, _ = w.Write(row)
	}
	_ = w.Close()
	return b.Bytes()
}

func tinyIDAT() []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, _ = w.Write([]byte{0, 0, 0, 0, 0})
	_ = w.Close()
	return b.Bytes()
}

func TestS01_00_ControlTinyPNG(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	_ = png.Encode(&buf, img)
	path := writeFixture(t, "tiny.png", buf.Bytes())
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatPng))
	if !parent {
		return
	}
	if o.Verdict != "ok" {
		t.Fatalf("control decode under cap failed, harness is not trustworthy: %s", o)
	}
}

func TestS01_00b_ControlTouch512MiB(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		b := make([]byte, 512<<20)
		for i := 0; i < len(b); i += 4096 {
			b[i] = 1
		}
		return fmt.Sprintf("touched %d MiB", len(b)>>20), nil
	})
	if parent && o.Verdict != "ok" {
		t.Fatalf("control: 512 MiB allocation fails under the cap, OOM verdicts are not trustworthy: %s", o)
	}
}

func TestS01_01_GifHugeLogicalScreen(t *testing.T) {
	var data []byte
	if !bombchild.IsChild(t) {
		pal := color.Palette{color.Black, color.White}
		frame := image.NewPaletted(image.Rect(0, 0, 1, 1), pal)
		var buf bytes.Buffer
		err := gif.EncodeAll(&buf, &gif.GIF{
			Image:  []*image.Paletted{frame},
			Delay:  []int{0},
			Config: image.Config{ColorModel: pal, Width: 65535, Height: 65535},
		})
		if err != nil {
			t.Fatal(err)
		}
		data = buf.Bytes()
	}
	path := writeFixture(t, "screen.gif", data)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatGif))
	if parent {
		bombchild.ExpectSafe(t, o, "GIF 65535x65535 logical screen with 1x1 frame")
	}
}

func TestS01_02_GifFrameCountAmplification(t *testing.T) {
	var data []byte
	if !bombchild.IsChild(t) {
		pal := color.Palette{color.Black, color.White}
		g := &gif.GIF{Config: image.Config{ColorModel: pal, Width: 6000, Height: 6000}}
		for range 32 {
			g.Image = append(g.Image, image.NewPaletted(image.Rect(0, 0, 1, 1), pal))
			g.Delay = append(g.Delay, 1)
		}
		var buf bytes.Buffer
		if err := gif.EncodeAll(&buf, g); err != nil {
			t.Fatal(err)
		}
		data = buf.Bytes()
	}
	path := writeFixture(t, "frames.gif", data)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatGif))
	if parent {
		bombchild.ExpectSafe(t, o, "GIF 6000x6000 screen with 32 1x1 frames (each frame a full canvas)")
	}
}

func TestS01_03_PnmHeaderDims(t *testing.T) {
	path := writeFixture(t, "big.pgm", []byte("P5\n65535 65535\n255\n\x00\x00\x00\x00"))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatPnm))
	if parent {
		bombchild.ExpectSafe(t, o, "PNM P5 65535x65535 with 4 data bytes")
	}
}

func TestS01_04_PnmP7NegativeWrap(t *testing.T) {
	path := writeFixture(t, "wrap.pam", []byte("P7\nWIDTH 4294967295\nHEIGHT 4294967295\nDEPTH 1\nMAXVAL 255\nTUPLTYPE GRAYSCALE\nENDHDR\n\x00"))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatPnm))
	if parent {
		bombchild.ExpectSafe(t, o, "PAM 4294967295x4294967295 (int overflow in w*h*components)")
	}
}

func TestS01_05_PnmP7ZeroWrap(t *testing.T) {
	path := writeFixture(t, "zero.pam", []byte("P7\nWIDTH 2147483648\nHEIGHT 2147483648\nDEPTH 4\nMAXVAL 255\nTUPLTYPE RGB_ALPHA\nENDHDR\n\x00"))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatPnm))
	if parent {
		bombchild.ExpectSafe(t, o, "PAM 2^31x2^31x4 (w*h*components wraps to 0)")
	}
}

func TestS01_06_PngHeaderDims(t *testing.T) {
	path := writeFixture(t, "header.png", pngWithDims(30000, 30000, 6, tinyIDAT()))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatPng))
	if parent {
		bombchild.ExpectSafe(t, o, "PNG IHDR 30000x30000 RGBA with 13-byte IDAT")
	}
}

func TestS01_07_PngZlibBomb(t *testing.T) {
	var data []byte
	if !bombchild.IsChild(t) {
		data = pngWithDims(30000, 30000, 0, zeroGrayIDAT(30000, 30000))
	}
	path := writeFixture(t, "bomb.png", data)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatPng))
	if parent {
		bombchild.ExpectSafe(t, o, fmt.Sprintf("valid %d-byte gray PNG 30000x30000 (900 MB gray + 3.6 GB RGBA)", len(data)))
	}
}

func jpegWithDims(w, h uint16) []byte {
	data := bytes.Clone(exifOrientationRotate180JPG)
	i := bytes.Index(data, []byte{0xff, 0xc0, 0x00, 0x0b, 0x08})
	binary.BigEndian.PutUint16(data[i+5:], h)
	binary.BigEndian.PutUint16(data[i+7:], w)
	return data
}

func TestS01_08_JpegSOFDims(t *testing.T) {
	path := writeFixture(t, "big.jpg", jpegWithDims(65535, 65535))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatJpeg))
	if parent {
		bombchild.ExpectSafe(t, o, "JPEG SOF0 65535x65535 in a 410-byte file")
	}
}

func TestS01_09_BmpHeaderDims(t *testing.T) {
	var buf bytes.Buffer
	_ = bmp.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	data := buf.Bytes()
	binary.LittleEndian.PutUint32(data[18:], 30000)
	binary.LittleEndian.PutUint32(data[22:], 30000)
	path := writeFixture(t, "big.bmp", data)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatBmp))
	if parent {
		bombchild.ExpectSafe(t, o, "BMP 30000x30000 header with 4 pixel bytes")
	}
}

func tiffWithDims(w, h uint32) []byte {
	var buf bytes.Buffer
	_ = tiff.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 1, 1)), nil)
	data := buf.Bytes()
	off := binary.LittleEndian.Uint32(data[4:])
	n := int(binary.LittleEndian.Uint16(data[off:]))
	for i := range n {
		e := int(off) + 2 + i*12
		tag := binary.LittleEndian.Uint16(data[e:])
		typ := binary.LittleEndian.Uint16(data[e+2:])
		var v uint32
		switch tag {
		case 256:
			v = w
		case 257:
			v = h
		default:
			continue
		}
		if typ == 3 {
			binary.LittleEndian.PutUint16(data[e+8:], uint16(v))
		} else {
			binary.LittleEndian.PutUint32(data[e+8:], v)
		}
	}
	return data
}

func TestS01_10_TiffHeaderDims(t *testing.T) {
	path := writeFixture(t, "big.tiff", tiffWithDims(30000, 30000))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatTiff))
	if parent {
		bombchild.ExpectSafe(t, o, "TIFF 30000x30000 tags with 4 pixel bytes")
	}
}

func riffChunks(data []byte) [][]byte {
	var out [][]byte
	body := data[12:]
	for len(body) >= 8 {
		size := int(binary.LittleEndian.Uint32(body[4:8]))
		end := 8 + size + size&1
		if end > len(body) {
			end = len(body)
		}
		out = append(out, body[:end])
		body = body[end:]
	}
	return out
}

func webpAnimated(width, height uint32, frames int) []byte {
	chunks := riffChunks(animatedWebp)
	var anim, anmf []byte
	for _, c := range chunks {
		switch string(c[:4]) {
		case "ANIM":
			anim = c
		case "ANMF":
			if anmf == nil {
				anmf = c
			}
		}
	}
	vp8x := make([]byte, 18)
	copy(vp8x, "VP8X")
	binary.LittleEndian.PutUint32(vp8x[4:], 10)
	vp8x[8] = 0x02
	vp8x[12], vp8x[13], vp8x[14] = byte(width-1), byte((width-1)>>8), byte((width-1)>>16)
	vp8x[15], vp8x[16], vp8x[17] = byte(height-1), byte((height-1)>>8), byte((height-1)>>16)
	var body bytes.Buffer
	body.WriteString("WEBP")
	body.Write(vp8x)
	body.Write(anim)
	for range frames {
		body.Write(anmf)
	}
	var out bytes.Buffer
	out.WriteString("RIFF")
	_ = binary.Write(&out, binary.LittleEndian, uint32(body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

func TestS01_11_WebpControlFixture(t *testing.T) {
	path := writeFixture(t, "control.webp", webpAnimated(1, 1, 2))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatWebp))
	if parent && o.Verdict != "ok" {
		t.Fatalf("rebuilt animated WebP control does not decode, frame-count test is invalid: %s", o)
	}
}

func TestS01_12_WebpAnmfFrameCount(t *testing.T) {
	path := writeFixture(t, "frames.webp", webpAnimated(11000, 11000, 10))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatWebp))
	if parent {
		bombchild.ExpectSafe(t, o, "WebP VP8X 11000x11000 (under 512 MiB canvas cap) with 10 1x1 ANMF frames")
	}
}

func TestS01_13_WebpCanvasCapHonoured(t *testing.T) {
	path := writeFixture(t, "cap.webp", webpAnimated(16384, 16384, 1))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatWebp))
	if parent {
		bombchild.ExpectSafe(t, o, "WebP VP8X 16384x16384 animated")
	}
}

func TestS01_14_IcoEmbeddedPngBomb(t *testing.T) {
	var data []byte
	if !bombchild.IsChild(t) {
		p := pngWithDims(30000, 30000, 6, tinyIDAT())
		var b bytes.Buffer
		b.Write([]byte{0, 0, 1, 0, 1, 0})
		entry := make([]byte, 16)
		binary.LittleEndian.PutUint16(entry[4:], 1)
		binary.LittleEndian.PutUint16(entry[6:], 32)
		binary.LittleEndian.PutUint32(entry[8:], uint32(len(p)))
		binary.LittleEndian.PutUint32(entry[12:], 22)
		b.Write(entry)
		b.Write(p)
		data = b.Bytes()
	}
	path := writeFixture(t, "bomb.ico", data)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatIco))
	if parent {
		bombchild.ExpectSafe(t, o, "ICO entry 0x0 (=256) embedding PNG IHDR 30000x30000")
	}
}

func TestS01_15_PbmBitAmplification(t *testing.T) {
	var data []byte
	if !bombchild.IsChild(t) {
		w, h := 32768, 16384
		hdr := fmt.Sprintf("P4\n%d %d\n", w, h)
		data = append([]byte(hdr), make([]byte, (w+7)/8*h)...)
	}
	path := writeFixture(t, "big.pbm", data)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeFile(gpui.ImageFormatPnm))
	if parent {
		bombchild.ExpectSafe(t, o, fmt.Sprintf("valid %d-byte P4 32768x16384 (1 bit/pixel -> ~14 bytes/pixel)", len(data)))
	}
}

func fuzzRegion(seed uint64, base []byte, lo, hi, iters int, format gpui.ImageFormat) (string, error) {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	errs := 0
	for i := range iters {
		data := bytes.Clone(base)
		n := 1 + rng.IntN(6)
		for range n {
			p := lo + rng.IntN(hi-lo)
			switch rng.IntN(3) {
			case 0:
				data[p] = byte(rng.IntN(256))
			case 1:
				data[p] = []byte{0, 0xff, 0x7f, 0x80, 1}[rng.IntN(5)]
			default:
				data[p] ^= 1 << rng.IntN(8)
			}
		}
		if rng.IntN(4) == 0 {
			data = data[:lo+rng.IntN(len(data)-lo)]
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("PANIC seed=%d iter=%d input=%s\n", seed, i, hex.EncodeToString(data))
					panic(r)
				}
			}()
			if _, err := decodeBytes(format, data); err != nil {
				errs++
			}
		}()
	}
	return fmt.Sprintf("iters=%d errors=%d", iters, errs), nil
}

func TestS01_16_ExifFuzzJpeg(t *testing.T) {
	seed := uint64(0x5e01)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		return fuzzRegion(seed, exifOrientationRotate180JPG, 20, 58, 20000, gpui.ImageFormatJpeg)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "seeded mutation of JPEG APP1/EXIF segment")
	}
}

func TestS01_17_ExifFuzzTiff(t *testing.T) {
	seed := uint64(0x5e02)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		base := tiffWithDims(1, 1)
		return fuzzRegion(seed, base, 4, len(base), 20000, gpui.ImageFormatTiff)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "seeded mutation of TIFF header/IFD")
	}
}

func TestS01_18_ExifFuzzPngEXIf(t *testing.T) {
	seed := uint64(0x5e03)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		exif := []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x01\x01\x12\x00\x03\x00\x00\x00\x01\x00\x06\x00\x00\x00\x00\x00\x00")
		var buf bytes.Buffer
		_ = png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 3, 2)))
		raw := buf.Bytes()
		base := append(bytes.Clone(raw[:33]), pngChunk("eXIf", exif)...)
		base = append(base, raw[33:]...)
		if _, err := decodeBytes(gpui.ImageFormatPng, base); err != nil {
			return "", fmt.Errorf("harness base invalid: %w", err)
		}
		return fuzzRegion(seed, base, 33, 33+12+len(exif), 20000, gpui.ImageFormatPng)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "seeded mutation of PNG eXIf chunk")
	}
}

func TestS01_19_RandomHeadersAllFormats(t *testing.T) {
	seed := uint64(0x5e04)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		bases := map[gpui.ImageFormat][]byte{
			gpui.ImageFormatWebp: webpAnimated(3, 3, 2),
			gpui.ImageFormatPnm:  []byte("P7\nWIDTH 2\nHEIGHT 2\nDEPTH 4\nMAXVAL 255\nTUPLTYPE RGB_ALPHA\nENDHDR\n0123456789abcdef"),
			gpui.ImageFormatIco:  icoDIB(),
		}
		var parts []string
		for f, b := range bases {
			r, _ := fuzzRegion(seed, b, 0, len(b), 15000, f)
			parts = append(parts, fmt.Sprintf("%v:%s", f, r))
		}
		return strings.Join(parts, " "), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "seeded mutation of small WebP/PAM/ICO-DIB")
	}
}

func icoDIB() []byte {
	w, h := 4, 4
	dib := make([]byte, 40)
	binary.LittleEndian.PutUint32(dib[0:], 40)
	binary.LittleEndian.PutUint32(dib[4:], uint32(w))
	binary.LittleEndian.PutUint32(dib[8:], uint32(h*2))
	binary.LittleEndian.PutUint16(dib[12:], 1)
	binary.LittleEndian.PutUint16(dib[14:], 32)
	dib = append(dib, make([]byte, w*h*4+h*4)...)
	var b bytes.Buffer
	b.Write([]byte{0, 0, 1, 0, 1, 0})
	entry := []byte{byte(w), byte(h), 0, 0, 1, 0, 32, 0}
	entry = binary.LittleEndian.AppendUint32(entry, uint32(len(dib)))
	entry = binary.LittleEndian.AppendUint32(entry, 22)
	b.Write(entry)
	b.Write(dib)
	return b.Bytes()
}

func loaderBody(t *testing.T) func(string) (string, error) {
	return func(path string) (string, error) {
		var load func(*scheduler.Scope) scheduler.Result[*gpui.RenderImage]
		gpui.RunTest(t, func(tc *gpui.TestRun) {
			tc.Cx().Update(func(cx *gpui.App) {
				load = gpui.ImageAssetLoader{}.Load(gpui.Resource{Kind: gpui.ResourceKindPath, Path: path}, cx)
			})
		})
		res := load(nil)
		if res.Err != nil {
			return "", res.Err
		}
		return fmt.Sprintf("decoded frames=%d", res.Value.FrameCount()), nil
	}
}

func TestS01_22_AssetLoaderSniffedFormats(t *testing.T) {
	cases := map[string][]byte{
		"wrap.pam":   []byte("P7\nWIDTH 4294967295\nHEIGHT 4294967295\nDEPTH 1\nMAXVAL 255\nTUPLTYPE GRAYSCALE\nENDHDR\n\x00"),
		"screen.gif": nil,
		"bomb.png":   nil,
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			var path string
			if !bombchild.IsChild(t) {
				switch name {
				case "screen.gif":
					pal := color.Palette{color.Black, color.White}
					var buf bytes.Buffer
					_ = gif.EncodeAll(&buf, &gif.GIF{Image: []*image.Paletted{image.NewPaletted(image.Rect(0, 0, 1, 1), pal)}, Delay: []int{0}, Config: image.Config{ColorModel: pal, Width: 65535, Height: 65535}})
					data = buf.Bytes()
				case "bomb.png":
					data = pngWithDims(30000, 30000, 0, zeroGrayIDAT(30000, 30000))
				}
				path = filepath.Join(t.TempDir(), "noext")
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, loaderBody(t))
			if parent {
				bombchild.ExpectSafe(t, o, "ImageAssetLoader.Load (format sniffed from magic, as for remote img(uri) bodies) of "+name)
			}
		})
	}
}

func TestS01_20_AssetLoaderDevZero(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "/dev/zero", loaderBody(t))
	if parent {
		bombchild.ExpectSafe(t, o, "ImageAssetLoader ResourceKindPath /dev/zero")
	}
}

func TestS01_21_AssetLoaderFifo(t *testing.T) {
	var p string
	if !bombchild.IsChild(t) {
		p = filepath.Join(t.TempDir(), "fifo.png")
		if err := syscall.Mkfifo(p, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 15*time.Second, p, loaderBody(t))
	if parent {
		bombchild.ExpectSafe(t, o, "ImageAssetLoader ResourceKindPath FIFO with no writer")
	}
}
