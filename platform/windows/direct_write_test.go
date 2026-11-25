//go:build windows

package windows

import (
	"math"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/dwrite"
)

func absPx(p gpui.Pixels) gpui.Pixels {
	if p < 0 {
		return -p
	}
	return p
}

func newTestDirectWriteTextSystem(t *testing.T) *directWriteTextSystem {
	t.Helper()
	platformTextSystem, err := newDirectWriteTextSystem()
	if err != nil {
		t.Fatalf("DirectWriteTextSystem::new: %v", err)
	}
	t.Cleanup(platformTextSystem.Close)
	return platformTextSystem
}

func TestDirectWrite_TabsPreserveNativeAdvancesAndCursor(t *testing.T) {
	platformTextSystem := newTestDirectWriteTextSystem(t)
	textSystem := gpui.NewWindowTextSystem(gpui.NewTextSystem(platformTextSystem))
	for _, family := range []string{"Consolas", "Segoe UI"} {
		font := gpui.NewFont(gpui.SharedString(family))
		fontID, err := platformTextSystem.FontID(font)
		if err != nil {
			t.Fatal(err)
		}
		for _, fontSize := range []gpui.Pixels{gpui.Px(12.0), gpui.Px(16.0)} {
			platformTextSystem.mu.RLock()
			info := &platformTextSystem.state.fonts[fontID]
			format, err := platformTextSystem.components.factory.CreateTextFormat(
				info.fontFamilyH,
				info.fontCollection,
				info.fontFace.GetWeight(),
				info.fontFace.GetStyle(),
				dwrite.DWRITE_FONT_STRETCH_NORMAL,
				float32(fontSize),
				platformTextSystem.components.locale,
			)
			platformTextSystem.mu.RUnlock()
			if err != nil {
				t.Fatal(err)
			}
			tabInterval := gpui.Px(format.GetIncrementalTabStop())
			for _, text := range []string{
				"ab",
				"\t",
				"\t\t",
				"\t\tab",
				"a\tb",
				"abcd\te",
				"ab\t",
				"ab\t\t",
				"a\tb\tc",
				"é\t𐐀\tz",
				"a \tb  ",
				"אב",
				"aאבb",
				"a\tאב\tb",
			} {
				var nativeMetrics dwrite.DWRITE_TEXT_METRICS
				layout, err := platformTextSystem.components.factory.CreateTextLayout(utf16.Encode([]rune(text)), format, float32(math.Inf(1)), float32(math.Inf(1)))
				if err != nil {
					t.Fatal(err)
				}
				err = layout.GetMetrics(&nativeMetrics)
				layout.Release()
				if err != nil {
					t.Fatal(err)
				}
				for _, splitRuns := range []bool{false, true} {
					var runs []gpui.TextRun
					if splitRuns {
						for _, character := range text {
							runs = append(runs, gpui.TextRun{Len: utf8.RuneLen(character), Font: font})
						}
					} else {
						runs = []gpui.TextRun{{Len: len(text), Font: font}}
					}
					shaped := textSystem.ShapeLine(gpui.SharedString(text), fontSize, runs, gpui.None[gpui.Pixels]())
					cursor := shaped.Cursor()
					expectedX := gpui.Px(0.0)
					for index, character := range text {
						end := index + utf8.RuneLen(character)
						var expectedEnd gpui.Pixels
						if character == '\t' {
							expectedEnd = tabInterval * (gpui.Pixels(math.Floor(float64(expectedX/tabInterval))) + 1.0)
						} else {
							characterRun := gpui.TextRun{Len: utf8.RuneLen(character), Font: font}
							expectedEnd = expectedX + textSystem.ShapeLine(gpui.SharedString(string(character)), fontSize, []gpui.TextRun{characterRun}, gpui.None[gpui.Pixels]()).Width()
						}
						if got := shaped.XForIndex(index); absPx(got-expectedX) >= gpui.Px(0.001) {
							t.Fatalf("%s, %v, %q, %v, index=%d: expected %v, got %v", family, fontSize, text, splitRuns, index, expectedX, got)
						}
						if got, ok := shaped.IndexForX(expectedX + gpui.Px(0.01)); !ok || got != index {
							t.Fatalf("%s, %q, index=%d: index_for_x = %d, %v", family, text, index, got, ok)
						}
						piece := cursor.TakeUntil(end)
						if string(piece.Text) != string(character) {
							t.Fatalf("%s, %q, index=%d: piece %q", family, text, index, piece.Text)
						}
						if absPx(piece.Width()-(expectedEnd-expectedX)) >= gpui.Px(0.001) {
							t.Fatalf("%s, %q, index=%d: piece width %v", family, text, index, piece.Width())
						}
						expectedX = expectedEnd
					}
					if absPx(shaped.Width()-expectedX) >= gpui.Px(0.001) {
						t.Fatalf("%s, %q: width %v, expected %v", family, text, shaped.Width(), expectedX)
					}
					if absPx(shaped.Width()-gpui.Px(nativeMetrics.WidthIncludingTrailingWhitespace)) >= gpui.Px(0.001) {
						t.Fatalf("%s, %q: width %v, native %v", family, text, shaped.Width(), nativeMetrics.WidthIncludingTrailingWhitespace)
					}
					if shaped.XForIndex(len(text)) != shaped.Width() {
						t.Fatalf("%s, %q: x_for_index(len) %v != width %v", family, text, shaped.XForIndex(len(text)), shaped.Width())
					}
				}
			}
			format.Release()
		}
	}
}

func TestDirectWrite_UnicodeLineSeparatorsPreserveShapedLineCursor(t *testing.T) {
	checkUnicodeSeparatorCursor(t, ' ')
}

func TestDirectWrite_UnicodeParagraphSeparatorsPreserveShapedLineCursor(t *testing.T) {
	checkUnicodeSeparatorCursor(t, ' ')
}

func checkUnicodeSeparatorCursor(t *testing.T, separator rune) {
	platformTextSystem := newTestDirectWriteTextSystem(t)
	font := gpui.NewFont("Segoe UI")
	if _, err := platformTextSystem.FontID(font); err != nil {
		t.Fatal(err)
	}
	textSystem := gpui.NewWindowTextSystem(gpui.NewTextSystem(platformTextSystem))

	sep := string(separator)
	separatedLetters := strings.Join([]string{"a", "b", "c", "d", "e", "f"}, sep)
	for _, text := range []string{
		separatedLetters,
		sep + "ab" + sep + sep + "cd" + sep,
		sep + sep + sep + sep + sep,
		"é" + sep + "𐐀" + sep + "z",
	} {
		for _, multipleFonts := range []bool{false, true} {
			runs := []gpui.TextRun{{Len: len(text), Font: font}}
			if multipleFonts {
				_, firstLen := utf8.DecodeRuneInString(text)
				runs[0].Len = firstLen
				boldFont := font
				boldFont.Weight = gpui.FontWeightBold
				runs = append(runs, gpui.TextRun{Len: len(text) - firstLen, Font: boldFont})
			}
			shaped := textSystem.ShapeLine(gpui.SharedString(text), gpui.Px(14.0), runs, gpui.None[gpui.Pixels]())
			referenceRuns := append([]gpui.TextRun(nil), runs...)
			offset := 0
			for i := range referenceRuns {
				end := offset + referenceRuns[i].Len
				n := 0
				for _, character := range text[offset:end] {
					if character != separator {
						n += utf8.RuneLen(character)
					}
				}
				referenceRuns[i].Len = n
				offset = end
			}
			referenceText := strings.ReplaceAll(text, sep, "")
			reference := textSystem.ShapeLine(gpui.SharedString(referenceText), gpui.Px(14.0), referenceRuns, gpui.None[gpui.Pixels]())
			if absPx(shaped.Width()-reference.Width()) >= gpui.Px(0.001) {
				t.Fatalf("%q, multiple_fonts=%v: width %v, reference %v", text, multipleFonts, shaped.Width(), reference.Width())
			}

			type fontGlyph struct {
				font  gpui.FontID
				glyph gpui.ShapedGlyph
			}
			var visibleGlyphs []fontGlyph
			for _, run := range shaped.Runs {
				for _, glyph := range run.Glyphs {
					if !strings.HasPrefix(text[glyph.Index:], sep) {
						visibleGlyphs = append(visibleGlyphs, fontGlyph{run.FontID, glyph})
					}
				}
			}
			var referenceGlyphs []fontGlyph
			for _, run := range reference.Runs {
				for _, glyph := range run.Glyphs {
					referenceGlyphs = append(referenceGlyphs, fontGlyph{run.FontID, glyph})
				}
			}
			if len(visibleGlyphs) != len(referenceGlyphs) {
				t.Fatalf("%q: %d glyphs, reference %d", text, len(visibleGlyphs), len(referenceGlyphs))
			}
			for i, actual := range visibleGlyphs {
				expected := referenceGlyphs[i]
				if actual.font != expected.font {
					t.Fatalf("%q: font %v, expected %v", text, actual.font, expected.font)
				}
				if actual.glyph.ID != expected.glyph.ID {
					t.Fatalf("%q: glyph %v, expected %v", text, actual.glyph.ID, expected.glyph.ID)
				}
				if absPx(actual.glyph.Position.X-expected.glyph.Position.X) >= gpui.Px(0.001) {
					t.Fatalf("%q: x %v, expected %v", text, actual.glyph.Position.X, expected.glyph.Position.X)
				}
				if absPx(actual.glyph.Position.Y-expected.glyph.Position.Y) >= gpui.Px(0.001) {
					t.Fatalf("%q: y %v, expected %v", text, actual.glyph.Position.Y, expected.glyph.Position.Y)
				}
				referenceIndex := 0
				for _, character := range text[:actual.glyph.Index] {
					if character != separator {
						referenceIndex += utf8.RuneLen(character)
					}
				}
				if referenceIndex != expected.glyph.Index {
					t.Fatalf("%q: index %d, expected %d", text, referenceIndex, expected.glyph.Index)
				}
			}

			cursor := shaped.Cursor()
			for index, character := range text {
				piece := cursor.TakeUntil(index + utf8.RuneLen(character))
				if string(piece.Text) != string(character) {
					t.Fatalf("%q: piece %q, expected %q", text, piece.Text, string(character))
				}
				if piece.Len() != utf8.RuneLen(character) {
					t.Fatalf("%q: piece len %d", text, piece.Len())
				}
				if character == separator && piece.Width() != gpui.Px(0.0) {
					t.Fatalf("%q: separator width %v", text, piece.Width())
				}
			}
		}
	}
}

func TestDirectWrite_ClusterMap(t *testing.T) {
	type next struct {
		utf16Len, glyphCount int
		ok                   bool
	}
	step := func(a *clusterAnalyzer) next {
		u, g, ok := a.next()
		return next{u, g, ok}
	}
	check := func(clusterMap []uint16, glyphCount int, want ...next) {
		t.Helper()
		analyzer := newClusterAnalyzer(clusterMap, glyphCount)
		for i, w := range append(want, next{}) {
			if got := step(analyzer); got != w {
				t.Fatalf("%v: step %d: got %+v, want %+v", clusterMap, i, got, w)
			}
		}
	}
	check([]uint16{0}, 1, next{1, 1, true})
	check([]uint16{0, 1, 2}, 3, next{1, 1, true}, next{1, 1, true}, next{1, 1, true})
	check([]uint16{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 4, 4, 4, 4}, 5, next{11, 4, true}, next{5, 1, true})
	check([]uint16{0, 0, 0, 0, 0}, 1, next{5, 1, true})
}

func TestDirectWrite_StringIndexConverter(t *testing.T) {
	text := "é𐐀z"
	c := newStringIndexConverter(text)
	c.advanceToUTF16Ix(1)
	if c.utf8Ix != 2 || c.utf16Ix != 1 {
		t.Fatalf("after utf16 1: %d %d", c.utf8Ix, c.utf16Ix)
	}
	c.advanceToUTF16Ix(3)
	if c.utf8Ix != 6 || c.utf16Ix != 3 {
		t.Fatalf("after utf16 3: %d %d", c.utf8Ix, c.utf16Ix)
	}
	c.advanceToUTF16Ix(10)
	if c.utf8Ix != len(text) {
		t.Fatalf("past end: %d", c.utf8Ix)
	}
	d := newStringIndexConverter(text)
	d.advanceToUTF8Ix(2)
	if d.utf8Ix != 2 || d.utf16Ix != 1 {
		t.Fatalf("after utf8 2: %d %d", d.utf8Ix, d.utf16Ix)
	}
}

func TestDirectWrite_LogfontLayout(t *testing.T) {
	var info logfontw
	if unsafe.Sizeof(info) != 92 || unsafe.Offsetof(info.lfFaceName) != 28 {
		t.Fatalf("LOGFONTW size %d, lfFaceName at %d", unsafe.Sizeof(info), unsafe.Offsetof(info.lfFaceName))
	}
}

func TestDirectWrite_SplitInclusive(t *testing.T) {
	got := splitInclusive(" ab  cd", isLineSeparator)
	want := []string{" ", "ab ", " ", "cd"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := splitInclusive("a ", isLineSeparator); len(got) != 1 || got[0] != "a " {
		t.Fatalf("trailing separator: %q", got)
	}
}
