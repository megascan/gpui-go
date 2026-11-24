//go:build securityharness

package s01

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
	"github.com/megascan/gpui-go/scheduler"
)

type bitWriter struct {
	buf   []byte
	acc   uint64
	nbits uint
}

func (w *bitWriter) put(v uint64, n uint) {
	w.acc |= v << w.nbits
	w.nbits += n
	for w.nbits >= 8 {
		w.buf = append(w.buf, byte(w.acc))
		w.acc >>= 8
		w.nbits -= 8
	}
}

func (w *bitWriter) bytes() []byte {
	out := append([]byte(nil), w.buf...)
	if w.nbits > 0 {
		out = append(out, byte(w.acc))
	}
	return out
}

func vp8lSolid(width, height uint32) []byte {
	var w bitWriter
	w.put(0x2f, 8)
	w.put(uint64(width-1), 14)
	w.put(uint64(height-1), 14)
	w.put(1, 1)
	w.put(0, 3)
	w.put(0, 1)
	w.put(0, 1)
	w.put(0, 1)
	for range 5 {
		w.put(1, 1)
		w.put(0, 1)
		w.put(0, 1)
		w.put(0, 1)
	}
	return w.bytes()
}

func riffChunk(fourCC string, data []byte) []byte {
	out := []byte(fourCC)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data)))
	out = append(out, data...)
	if len(data)&1 == 1 {
		out = append(out, 0)
	}
	return out
}

func put24(b []byte, v uint32) {
	b[0], b[1], b[2] = byte(v), byte(v>>8), byte(v>>16)
}

func webpAnmfWithBitstream(canvasW, canvasH, frameW, frameH uint32, bitstream []byte, frames int) []byte {
	vp8x := make([]byte, 10)
	vp8x[0] = 0x12
	put24(vp8x[4:], canvasW-1)
	put24(vp8x[7:], canvasH-1)
	anim := make([]byte, 6)
	var body bytes.Buffer
	body.WriteString("WEBP")
	body.Write(riffChunk("VP8X", vp8x))
	body.Write(riffChunk("ANIM", anim))
	for range frames {
		hdr := make([]byte, 16)
		put24(hdr[6:], frameW-1)
		put24(hdr[9:], frameH-1)
		put24(hdr[12:], 100)
		body.Write(riffChunk("ANMF", append(hdr, riffChunk("VP8L", bitstream)...)))
	}
	var out bytes.Buffer
	out.WriteString("RIFF")
	_ = binary.Write(&out, binary.LittleEndian, uint32(body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

func decodeWithStats(format gpui.ImageFormat) func(string) (string, error) {
	return func(path string) (string, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		start := time.Now()
		res, derr := decodeBytes(format, data)
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		stats := fmt.Sprintf("input=%dB TotalAlloc=%dMiB Sys=%dMiB elapsed=%s", len(data), ms.TotalAlloc>>20, ms.Sys>>20, time.Since(start).Round(time.Millisecond))
		if derr != nil {
			return "", fmt.Errorf("%v; %s", derr, stats)
		}
		return res + "; " + stats, nil
	}
}

func totalAllocMiB(o bombchild.Outcome) int {
	i := strings.Index(o.Result, "TotalAlloc=")
	if i < 0 {
		return -1
	}
	var n int
	_, _ = fmt.Sscanf(o.Result[i+len("TotalAlloc="):], "%dMiB", &n)
	return n
}

func TestR2_S01_00_WebpAnmfControl(t *testing.T) {
	path := writeFixture(t, "anmf-ok.webp", webpAnmfWithBitstream(4, 4, 4, 4, vp8lSolid(4, 4), 2))
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeWithStats(gpui.ImageFormatWebp))
	if parent && o.Verdict != "ok" {
		t.Fatalf("control: 4x4 animated WebP with VP8L frames does not decode, the bitstream builder is wrong: %s", o)
	}
}

func TestR2_S01_01_WebpAnmfSubBitstreamDims(t *testing.T) {
	const frames = 4
	data := webpAnmfWithBitstream(1, 1, 1, 1, vp8lSolid(16384, 16384), frames)
	path := writeFixture(t, "anmf-bitstream.webp", data)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeWithStats(gpui.ImageFormatWebp))
	if !parent {
		return
	}
	bombchild.ExpectSafe(t, o, "animated WebP")
	alloc := totalAllocMiB(o)
	t.Logf("%d-byte animated WebP, 1x1 canvas, %d ANMF frames declared 1x1 whose VP8L bitstream says 16384x16384: TotalAlloc=%d MiB, elapsed=%s", len(data), frames, alloc, o.Elapsed.Round(time.Millisecond))
	if alloc < 0 || alloc >= 512 || o.Elapsed > 5*time.Second {
		t.Errorf("BUDGET BYPASS: decoding a %d-byte WebP allocated %d MiB (decode budget is 1024 MiB total, 512 MiB per buffer) in %s; frame bitstream dims are never checked against the ANMF frame before webp.Decode", len(data), alloc, o.Elapsed.Round(time.Millisecond))
	}
}

func gifFrames(width, height, frames int) []byte {
	pal := color.Palette{color.Black, color.White, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}}
	imgs := make([]*image.Paletted, 4)
	for k := range imgs {
		p := image.NewPaletted(image.Rect(0, 0, width, height), pal)
		for y := 0; y < height; y += 8 {
			for x := 0; x < width; x++ {
				p.Pix[y*width+x] = uint8((x/16 + y/16 + k) % 4)
			}
		}
		imgs[k] = p
	}
	g := &gif.GIF{Config: image.Config{Width: width, Height: height, ColorModel: pal}}
	for i := range frames {
		g.Image = append(g.Image, imgs[i%4])
		g.Delay = append(g.Delay, 3)
		g.Disposal = append(g.Disposal, gif.DisposalNone)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func legitGif(t *testing.T, width, height, frames int) {
	var path string
	if !bombchild.IsChild(t) {
		path = writeFixture(t, "legit.gif", gifFrames(width, height, frames))
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeWithStats(gpui.ImageFormatGif))
	if !parent {
		return
	}
	t.Logf("%dx%d GIF with %d frames (%.1fs at 30 fps): %s", width, height, frames, float64(frames)/30, o)
	if strings.Contains(o.Result, "too large to decode within the memory limit") {
		t.Errorf("BEHAVIOUR-REGRESSION: an ordinary %dx%d animated GIF with %d frames is refused by the decode budget", width, height, frames)
	} else if o.Verdict != "ok" {
		t.Errorf("legit GIF did not decode: %s", o)
	}
}

func TestR2_S01_02_LegitGif640x480x900(t *testing.T) { legitGif(t, 640, 480, 900) }

func TestR2_S01_03_LegitGif1920x1080x150(t *testing.T) { legitGif(t, 1920, 1080, 150) }

func TestR2_S01_04_LegitProgressiveJpeg61MP(t *testing.T) {
	if _, err := exec.LookPath("magick"); err != nil {
		t.Skip("harness-error: magick not available")
	}
	var path string
	if !bombchild.IsChild(t) {
		dir := t.TempDir()
		path = filepath.Join(dir, "a7r4.jpg")
		cmd := exec.Command("nice", "-n", "19", "magick", "-size", "9504x6336", "plasma:fractal", "-seed", "7", "-interlace", "JPEG", "-quality", "85", path)
		cmd.Env = append(os.Environ(), "MAGICK_THREAD_LIMIT=2")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("harness-error: magick: %v %s", err, out)
		}
		st, _ := os.Stat(path)
		t.Logf("fixture: genuine progressive JPEG 9504x6336 (61 MP, a Sony A7R IV frame size), %d bytes", st.Size())
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeWithStats(gpui.ImageFormatJpeg))
	if !parent {
		return
	}
	t.Logf("%s", o)
	if strings.Contains(o.Result, "too large to decode within the memory limit") {
		t.Errorf("BEHAVIOUR-REGRESSION: a genuine 61 MP progressive JPEG is refused by the decode budget")
	}
}

func TestR2_S01_05_ConcurrentNearCapDecodes(t *testing.T) {
	const side = 11000
	const workers = 8
	var path string
	if !bombchild.IsChild(t) {
		data := pngWithDims(side, side, 0, zeroGrayIDAT(side, side))
		path = writeFixture(t, "near-cap.png", data)
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, func(p string) (string, error) {
		data, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		ex := gpui.NewBackgroundExecutor(gpui.NewThreadedDispatcher())
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errs []error
		ok := 0
		start := time.Now()
		for range workers {
			wg.Add(1)
			ex.Spawn(func(s *scheduler.Scope) (struct{}, error) {
				defer wg.Done()
				_, err := decodeBytes(gpui.ImageFormatPng, data)
				mu.Lock()
				if err != nil {
					errs = append(errs, err)
				} else {
					ok++
				}
				mu.Unlock()
				return struct{}{}, nil
			}).Detach()
		}
		wg.Wait()
		mu.Lock()
		defer mu.Unlock()
		if ok == 0 && len(errs) > 0 {
			return "", errors.Join(errs...)
		}
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		return fmt.Sprintf("%d concurrent decodes of a %d-byte %dx%d PNG ok=%d in %s, Sys=%dMiB", workers, len(data), side, side, ok, time.Since(start).Round(time.Millisecond), ms.Sys>>20), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, fmt.Sprintf("%d concurrent background decodes of one %dx%d PNG that individually fits the budget", workers, side, side))
	}
}

func TestR2_S01_01b_WebpAnmfSubBitstreamOneFrame(t *testing.T) {
	data := webpAnmfWithBitstream(1, 1, 1, 1, vp8lSolid(16384, 16384), 1)
	path := writeFixture(t, "anmf-bitstream-1.webp", data)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeWithStats(gpui.ImageFormatWebp))
	if !parent {
		return
	}
	alloc := totalAllocMiB(o)
	t.Logf("%d-byte animated WebP, 1x1 canvas, one 1x1 ANMF frame with a 16384x16384 VP8L bitstream: %s", len(data), o)
	if !o.Safe() || alloc >= 512 || o.Elapsed > 5*time.Second {
		t.Errorf("BUDGET BYPASS: one %d-byte frame cost TotalAlloc=%d MiB and %s (budget 1024 MiB total)", len(data), alloc, o.Elapsed.Round(time.Millisecond))
	}
}

func tiffDeflateGray(width, height uint32, decompressed int64) []byte {
	var z bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&z, zlib.BestCompression)
	zero := make([]byte, 1<<20)
	for left := decompressed; left > 0; left -= int64(len(zero)) {
		_, _ = zw.Write(zero[:min(int64(len(zero)), left)])
	}
	_ = zw.Close()
	type entry struct {
		tag, typ uint16
		val      uint32
	}
	entries := []entry{{256, 4, width}, {257, 4, height}, {258, 3, 8}, {259, 3, 8}, {262, 3, 1}, {273, 4, 0}, {277, 3, 1}, {278, 4, height}, {279, 4, uint32(z.Len())}}
	ifdOff := uint32(8)
	dataOff := ifdOff + 2 + uint32(len(entries))*12 + 4
	var b bytes.Buffer
	b.WriteString("II")
	_ = binary.Write(&b, binary.LittleEndian, uint16(42))
	_ = binary.Write(&b, binary.LittleEndian, ifdOff)
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

func TestR2_S01_06_TiffStripBufferOutgrowsBudget(t *testing.T) {
	const side = 11000
	var path string
	if !bombchild.IsChild(t) {
		path = writeFixture(t, "strip.tiff", tiffDeflateGray(side, side, int64(side)*side*8))
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, path, decodeWithStats(gpui.ImageFormatTiff))
	if !parent {
		return
	}
	alloc := totalAllocMiB(o)
	t.Logf("%dx%d 8-bit gray TIFF whose deflate strip inflates to 8 bytes/pixel: %s", side, side, o)
	bombchild.ExpectSafe(t, o, "TIFF strip")
	if alloc > 1024 {
		t.Errorf("BUDGET UNDERCOUNT: the decode budget reserved %d MiB for this TIFF (1 B/px gray + 4 B/px RGBA) but decoding allocated %d MiB (x/image/tiff strip buffer may hold 8 B/px)", int64(side)*side*5>>20, alloc)
	}
}
