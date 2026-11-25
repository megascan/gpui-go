//go:build securityharness

package s03

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
	"github.com/megascan/gpui-go/text/gotext"
)

const (
	emojiFont = "/usr/share/fonts/noto/NotoColorEmoji.ttf"
	colrFont  = "/usr/share/fonts/noto/NotoZnamennyMusicalNotation-Regular.ttf"
)

type table struct{ off, length int }

func tables(data []byte) map[string]table {
	out := map[string]table{}
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := range n {
		r := data[12+16*i:]
		out[string(r[:4])] = table{int(binary.BigEndian.Uint32(r[8:])), int(binary.BigEndian.Uint32(r[12:]))}
	}
	return out
}

func readFont(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Skipf("font fixture missing: %v", err)
	}
	return b
}

type rasterStats struct {
	ok, errs, glyphs int
	maxW, maxH       int32
}

const baseFont = "/usr/share/fonts/noto/NotoSans-Regular.ttf"

var baseBytes = func() []byte {
	b, _ := os.ReadFile(baseFont)
	return b
}()

func emojiText() string { return "m😀🎉👍❤🙂🦀" }

func colrText() string {
	r := []rune{'m'}
	for c := rune(0x1CF00); c < 0x1CF50; c++ {
		r = append(r, c)
	}
	return string(r)
}

func rasterize(target []byte, text string, sizes []gpui.Pixels) (rasterStats, error) {
	var st rasterStats
	if len(baseBytes) == 0 {
		return st, fmt.Errorf("harness: base font missing")
	}
	ts := gotext.NewTextSystemWithoutSystemFonts("")
	fonts := [][]byte{baseBytes}
	if target != nil {
		fonts = append(fonts, target)
	}
	if err := ts.AddFonts(fonts); err != nil {
		return st, err
	}
	base, err := ts.FontID(gpui.Font{Family: "Noto Sans", Weight: gpui.FontWeightNormal})
	if err != nil {
		return st, err
	}
	for _, size := range sizes {
		layout := ts.LayoutLine(text, size, []gpui.FontRun{{Len: len(text), FontID: base}})
		for _, run := range layout.Runs {
			if target != nil && run.FontID == base {
				continue
			}
			for _, g := range run.Glyphs {
				st.glyphs++
				params := gpui.RenderGlyphParams{FontID: run.FontID, GlyphID: g.ID, FontSize: size, ScaleFactor: 1, IsEmoji: g.IsEmoji}
				bounds, err := ts.GlyphRasterBounds(params)
				if err != nil {
					st.errs++
					continue
				}
				sz, _, err := ts.RasterizeGlyph(params, bounds)
				if err != nil {
					st.errs++
					continue
				}
				st.ok++
				st.maxW = max(st.maxW, int32(sz.Width))
				st.maxH = max(st.maxH, int32(sz.Height))
			}
		}
	}
	return st, nil
}

func (s rasterStats) String() string {
	return fmt.Sprintf("fallbackGlyphs=%d ok=%d errs=%d max=%dx%d", s.glyphs, s.ok, s.errs, s.maxW, s.maxH)
}

var sizes = []gpui.Pixels{16, 109, 300}

func TestS03_00_ControlEmoji(t *testing.T) {
	data := readFont(t, emojiFont)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 50*time.Second, "", func(string) (string, error) {
		st, err := rasterize(data, emojiText(), sizes)
		if err != nil {
			return "", err
		}
		if st.ok == 0 {
			return "", fmt.Errorf("control rasterized nothing: %s", st)
		}
		return st.String(), nil
	})
	if parent && o.Verdict != "ok" {
		t.Fatalf("control emoji raster failed, harness invalid: %s", o)
	}
}

func patchPNGs(data []byte, w, h uint32) int {
	tb := tables(data)["CBDT"]
	region := data[tb.off : tb.off+tb.length]
	n := 0
	for i := 0; ; {
		j := bytes.Index(region[i:], []byte("\x89PNG\r\n\x1a\n"))
		if j < 0 {
			break
		}
		p := i + j
		ihdr := region[p+8:]
		if string(ihdr[4:8]) == "IHDR" {
			binary.BigEndian.PutUint32(ihdr[8:], w)
			binary.BigEndian.PutUint32(ihdr[12:], h)
			crc := crc32.ChecksumIEEE(ihdr[4:21])
			binary.BigEndian.PutUint32(ihdr[21:], crc)
			n++
		}
		i = p + 8
	}
	return n
}

func TestS03_01_CbdtPngStrikeHugeIHDR(t *testing.T) {
	data := readFont(t, emojiFont)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 50*time.Second, "", func(string) (string, error) {
		mut := bytes.Clone(data)
		n := patchPNGs(mut, 65535, 65535)
		st, err := rasterize(mut, emojiText(), sizes)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("patched=%d %s", n, st), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "CBDT strike PNGs with IHDR 65535x65535 (CRC fixed)")
	}
}

func TestS03_02_CbdtPngStrikeWrapIHDR(t *testing.T) {
	data := readFont(t, emojiFont)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 50*time.Second, "", func(string) (string, error) {
		mut := bytes.Clone(data)
		n := patchPNGs(mut, 32768, 32768)
		st, err := rasterize(mut, emojiText(), sizes)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("patched=%d %s", n, st), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "CBDT strike PNGs with IHDR 32768x32768 (w*h*4 wraps to 0 in uint32)")
	}
}

func fuzzTables(seed uint64, data []byte, tags []string, iters int, text string, window int) (string, error) {
	rng := rand.New(rand.NewPCG(seed, seed*31+7))
	tb := tables(data)
	var regions []table
	for _, tag := range tags {
		if r, ok := tb[tag]; ok {
			if window > 0 && r.length > window {
				r.length = window
			}
			regions = append(regions, r)
		}
	}
	if len(regions) == 0 {
		return "", fmt.Errorf("no target tables %v", tags)
	}
	total := rasterStats{}
	addErrs := 0
	deadline := time.Now().Add(40 * time.Second)
	done := 0
	for i := range iters {
		if time.Now().After(deadline) {
			break
		}
		mut := bytes.Clone(data)
		r := regions[rng.IntN(len(regions))]
		for range 1 + rng.IntN(8) {
			p := r.off + rng.IntN(r.length)
			switch rng.IntN(4) {
			case 0:
				mut[p] = byte(rng.IntN(256))
			case 1:
				mut[p] = []byte{0, 0xff, 0x7f, 0x80, 1, 0xfe}[rng.IntN(6)]
			case 2:
				mut[p] ^= 1 << rng.IntN(8)
			default:
				if p+4 <= len(mut) {
					binary.BigEndian.PutUint32(mut[p:], []uint32{0, 0xffffffff, 0x7fffffff, 0x80000000, 0x10000}[rng.IntN(5)])
				}
			}
		}
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					fmt.Printf("PANIC seed=%d iter=%d tags=%v\n", seed, i, tags)
					panic(rec)
				}
			}()
			st, err := rasterize(mut, text, sizes)
			if err != nil {
				addErrs++
				return
			}
			total.ok += st.ok
			total.glyphs += st.glyphs
			total.errs += st.errs
		}()
		done = i + 1
	}
	return fmt.Sprintf("iters=%d addErrs=%d raster %s", done, addErrs, total), nil
}

func TestS03_03_FuzzCblcCbdt(t *testing.T) {
	data := readFont(t, emojiFont)
	seed := uint64(0x5e0301)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 55*time.Second, "", func(string) (string, error) {
		return fuzzTables(seed, data, []string{"CBLC", "CBDT"}, 400, emojiText(), 1<<16)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "seeded mutation of CBLC and first 64 KiB of CBDT")
	}
}

func TestS03_04_FuzzColrCpal(t *testing.T) {
	data := readFont(t, colrFont)
	seed := uint64(0x5e0302)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 55*time.Second, "", func(string) (string, error) {
		return fuzzTables(seed, data, []string{"COLR", "CPAL"}, 3000, colrText(), 0)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "seeded mutation of COLR/CPAL")
		if strings.Contains(o.Result, "fallbackGlyphs=0 ") {
			t.Skipf("COVERAGE GAP: no COLR glyph was shaped (host COLR font lacks 'm' for FontID and is never chosen by fallback); scale_color.go/palette.go not exercised: %s", o.Result)
		}
	}
}

func TestS03_05_FuzzOutlineTables(t *testing.T) {
	data := readFont(t, colrFont)
	seed := uint64(0x5e0303)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 55*time.Second, "", func(string) (string, error) {
		return fuzzTables(seed, data, []string{"glyf", "loca", "head", "maxp", "hhea", "hmtx", "cmap", "GSUB", "GPOS", "GDEF", "name", "OS/2"}, 3000, colrText(), 0)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "seeded mutation of outline/layout tables")
	}
}

func TestS03_06_HugeGlyphSize(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 50*time.Second, "", func(string) (string, error) {
		st, err := rasterize(nil, "mW@", []gpui.Pixels{20000, 1e9})
		if err != nil {
			return "", err
		}
		return st.String(), nil
	})
	if parent {
		t.Logf("INFO log-only (font size is app-authored): %s", o)
	}
}

func TestS03_07_HugeLayoutString(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 50*time.Second, "", func(string) (string, error) {
		ts := gotext.NewTextSystemWithoutSystemFonts("")
		if err := ts.AddFonts([][]byte{baseBytes}); err != nil {
			return "", err
		}
		id, err := ts.FontID(gpui.Font{Family: "Noto Sans", Weight: gpui.FontWeightNormal})
		if err != nil {
			return "", err
		}
		rng := rand.New(rand.NewPCG(1, 2))
		var b bytes.Buffer
		for b.Len() < 4<<20 {
			b.WriteRune([]rune{'a', '\u0301', '\u202e', '\u200d', '\u05e9', '\U0001F600', '\u0000', '\ufffd', ' '}[rng.IntN(9)])
		}
		s := b.String()
		start := time.Now()
		l := ts.LayoutLine(s, 14, []gpui.FontRun{{Len: len(s), FontID: id}})
		return fmt.Sprintf("glyph runs=%d width=%v in %s", len(l.Runs), l.Width, time.Since(start).Round(time.Millisecond)), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "LayoutLine of a 4 MiB mixed bidi/combining string")
		if o.Elapsed > 30*time.Second {
			t.Errorf("UNSAFE CPU: layout took %s", o.Elapsed)
		}
	}
}

func decreasingLoca(data []byte) []byte {
	mut := bytes.Clone(data)
	tb := tables(mut)
	head, loca := tb["head"], tb["loca"]
	long := binary.BigEndian.Uint16(mut[head.off+50:]) == 1
	if long {
		a := binary.BigEndian.Uint32(mut[loca.off+8:])
		binary.BigEndian.PutUint32(mut[loca.off+4:], a+64)
	} else {
		a := binary.BigEndian.Uint16(mut[loca.off+4:])
		binary.BigEndian.PutUint16(mut[loca.off+2:], a+32)
	}
	return mut
}

func TestS03_08_DecreasingLocaPanics(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 30*time.Second, "", func(string) (string, error) {
		mut := decreasingLoca(baseBytes)
		ts := gotext.NewTextSystemWithoutSystemFonts("")
		if err := ts.AddFonts([][]byte{mut}); err != nil {
			return "", err
		}
		names := ts.AllFontNames()
		id, err := ts.FontID(gpui.Font{Family: gpui.SharedString(names[0]), Weight: gpui.FontWeightNormal})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("font id %d", id), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "NotoSans with loca[1] > loca[2] (2-byte patch) -> AddFonts + FontID")
	}
}

func TestS03_09_DecreasingLocaViaFallback(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 30*time.Second, "", func(string) (string, error) {
		target, err := os.ReadFile(colrFont)
		if err != nil {
			return "", err
		}
		st, err := rasterize(decreasingLoca(target), colrText(), []gpui.Pixels{16})
		if err != nil {
			return "", err
		}
		return st.String(), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "app font Noto Sans + a second font with decreasing loca reached only through LayoutLine fallback")
	}
}
