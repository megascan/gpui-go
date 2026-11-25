//go:build securityharness

package s19

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/text/gotext"
)

const (
	seed     uint64 = 0x5519
	fontPath        = "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
)

func textSystem(t *testing.T) (*gpui.TextSystem, *gotext.CosmicTextSystem, gpui.Font) {
	t.Helper()
	data, err := os.ReadFile(fontPath)
	if err != nil {
		t.Skipf("harness-error: font fixture missing: %v", err)
	}
	pts := gotext.NewTextSystemWithoutSystemFonts("Liberation Sans")
	if err := pts.AddFonts([][]byte{data}); err != nil {
		t.Fatalf("harness-error: AddFonts: %v", err)
	}
	return gpui.NewTextSystem(pts), pts, gpui.NewFont("Liberation Sans")
}

var pieces = []string{"a", "W", " ", ".", "-", "é", "é", "日", "‮", "⁦", "​", "\xff", "\xc3", "\xe6\x97", "\x00", "\t", "😀", "👩‍💻", "/", "\n"}

func randomLine(rng *rand.Rand, max int) string {
	var b strings.Builder
	for range 1 + rng.IntN(max) {
		b.WriteString(pieces[rng.IntN(len(pieces))])
	}
	return b.String()
}

func runsLen(runs []gpui.TextRun) int {
	n := 0
	for _, r := range runs {
		n += r.Len
	}
	return n
}

func TestS19_TruncateFuzz(t *testing.T) {
	ts, pts, font := textSystem(t)
	fontID := ts.ResolveFont(font)
	wrapper := ts.LineWrapper(font, gpui.Px(14))
	defer wrapper.Release()
	t.Logf("seed=%#x", seed)
	rng := rand.New(rand.NewPCG(seed, 1))
	mismatches := 0
	invalidOut := 0
	var firstMismatch, firstInvalid string
	for i := range 20000 {
		line := randomLine(rng, 40)
		width := gpui.Px(float32(rng.IntN(300)))
		affix := []string{"…", "", "...", "\xff"}[rng.IntN(4)]
		from := []gpui.TruncateFrom{gpui.TruncateFromStart, gpui.TruncateFromEnd, gpui.TruncateFromMiddle}[rng.IntN(3)]
		split := rng.IntN(len(line) + 1)
		runs := []gpui.TextRun{{Len: split, Font: font}, {Len: len(line) - split, Font: font}}
		maxLines := rng.IntN(3)
		var out gpui.SharedString
		var outRuns []gpui.TextRun
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC iteration %d TruncateLine/TruncateWrappedLine(%q, width=%v, affix=%q, from=%v, lines=%d): %v", i, line, width, affix, from, maxLines, r)
				}
			}()
			if maxLines == 0 {
				out, outRuns = wrapper.TruncateLine(gpui.SharedString(line), width, affix, runs, from)
			} else {
				out, outRuns = wrapper.TruncateWrappedLine(gpui.SharedString(line), width, maxLines, affix, runs, from)
			}
		}()
		if runsLen(outRuns) != len(out) {
			mismatches++
			if firstMismatch == "" {
				firstMismatch = fmt.Sprintf("line=%q width=%v affix=%q from=%v lines=%d -> %q runs=%d", line, width, affix, from, maxLines, out, runsLen(outRuns))
			}
		}
		if utf8.ValidString(line) && utf8.ValidString(affix) && !utf8.ValidString(string(out)) {
			invalidOut++
			if firstInvalid == "" {
				firstInvalid = fmt.Sprintf("line=%q width=%v affix=%q from=%v lines=%d -> %q", line, width, affix, from, maxLines, out)
			}
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC iteration %d LayoutLine(%q) after truncation of %q: %v", i, out, line, r)
				}
			}()
			fr := make([]gpui.FontRun, 0, len(outRuns))
			for _, r := range outRuns {
				if r.Len > 0 {
					fr = append(fr, gpui.FontRun{Len: r.Len, FontID: fontID})
				}
			}
			_ = pts.LayoutLine(string(out), gpui.Px(14), fr)
		}()
	}
	t.Logf("run-length mismatches=%d first=%s", mismatches, firstMismatch)
	t.Logf("valid-in invalid-out=%d first=%s", invalidOut, firstInvalid)
	if invalidOut > 0 {
		t.Errorf("CORRUPT: truncation of valid UTF-8 produced invalid UTF-8")
	}
}

func TestS19_MiddleTruncationInvalidByte(t *testing.T) {
	ts, _, font := textSystem(t)
	wrapper := ts.LineWrapper(font, gpui.Px(14))
	defer wrapper.Release()
	for _, line := range []string{"ab\xffé" + strings.Repeat("x", 60) + "end", "\xff日本" + strings.Repeat("W", 50), "a\xff\xff\xffbcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"} {
		for w := 10; w < 200; w += 3 {
			runs := []gpui.TextRun{{Len: len(line), Font: font}}
			out, outRuns := wrapper.TruncateLine(gpui.SharedString(line), gpui.Px(float32(w)), "…", runs, gpui.TruncateFromMiddle)
			front, _, _ := strings.Cut(string(out), "…")
			if !strings.HasPrefix(line, front) {
				t.Errorf("MISMATCH width=%d: front %q is not a prefix of %q", w, front, line)
			}
			if runsLen(outRuns) != len(out) {
				t.Errorf("RUNS width=%d: runs cover %d bytes, text %d (%q)", w, runsLen(outRuns), len(out), out)
			}
			if bad := strings.Count(string(out), "�"); bad > 0 || (utf8.ValidString(string(out)) != utf8.ValidString(line) && !strings.Contains(line, "\xff")) {
				t.Logf("width=%d out=%q", w, out)
			}
		}
	}
}

func TestS19_BidiOverrideReordersGlyphs(t *testing.T) {
	_, pts, _ := textSystem(t)
	fontID, err := pts.FontID(gpui.NewFont("Liberation Sans"))
	if err != nil {
		t.Fatalf("harness-error: %v", err)
	}
	text := "invoice_‮gpj.exe"
	layout := pts.LayoutLine(text, gpui.Px(14), []gpui.FontRun{{Len: len(text), FontID: fontID}})
	visual := make([]byte, 0, len(text))
	type gx struct {
		x float32
		i int
	}
	var gs []gx
	for _, r := range layout.Runs {
		for _, g := range r.Glyphs {
			gs = append(gs, gx{float32(g.Position.X), g.Index})
		}
	}
	for i := range gs {
		for j := i + 1; j < len(gs); j++ {
			if gs[j].x < gs[i].x {
				gs[i], gs[j] = gs[j], gs[i]
			}
		}
	}
	for _, g := range gs {
		r, _ := utf8.DecodeRuneInString(text[g.i:])
		if r != '‮' {
			visual = utf8.AppendRune(visual, r)
		}
	}
	t.Logf("logical %q renders left-to-right as %q", text, visual)
	if strings.HasSuffix(string(visual), "exe.jpg") {
		t.Logf("INFO: U+202E is honoured, so an attacker-named file %q displays as %q (Trojan-Source style spoof; no stripping by default)", text, visual)
	}
}

func TestS19_WrappedTruncateInvalidUTF8Panic(t *testing.T) {
	ts, _, font := textSystem(t)
	wrapper := ts.LineWrapper(font, gpui.Px(14))
	defer wrapper.Release()
	candidates := []string{"a\nW\xe6\x97", "report.pdf\n\xe6\x97", "\u200b\n\u202eW-😀\t日\xe6\x97", "日\xe6\x97", "abc \xe6\x97", "file\xe6\x97", "a\xff", "a b\xc3", "W\xe6"}
	panics := 0
	for _, line := range candidates {
		for lines := 1; lines <= 3; lines++ {
			for w := 1; w <= 120; w++ {
				for _, from := range []gpui.TruncateFrom{gpui.TruncateFromStart, gpui.TruncateFromEnd, gpui.TruncateFromMiddle} {
					func() {
						defer func() {
							if r := recover(); r != nil {
								panics++
								if panics <= 6 {
									t.Errorf("PANIC TruncateWrappedLine(%q, wrapWidth=%dpx, maxLines=%d, affix=\"\", from=%v): %v", line, w, lines, from, r)
								}
							}
						}()
						runs := []gpui.TextRun{{Len: len(line), Font: font}}
						wrapper.TruncateWrappedLine(gpui.SharedString(line), gpui.Px(float32(w)), lines, "", runs, from)
						wrapper.TruncateWrappedLine(gpui.SharedString(line), gpui.Px(float32(w)), lines, "…", runs, from)
					}()
				}
			}
		}
	}
	t.Logf("total panics=%d", panics)
}
