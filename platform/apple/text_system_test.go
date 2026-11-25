//go:build darwin

package apple

import (
	_ "embed"
	"slices"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/ct"
)

//go:embed testdata/fonts/lilex_Lilex-Regular.ttf
var lilexRegular []byte

func mustFontID(t *testing.T, fonts *AppleTextSystem, font gpui.Font) gpui.FontID {
	t.Helper()
	id, err := fonts.FontID(font)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestTextSystem_ReloadFamilyWithFeaturesAndFallbacks(t *testing.T) {
	fonts := NewAppleTextSystem()
	font := gpui.NewFont("Helvetica")
	originalID := mustFontID(t, fonts, font)
	font.Features = gpui.FontFeaturesDisableLigatures()
	font.Fallbacks = gpui.Some(gpui.FontFallbacksFromFonts([]string{"Times"}))
	configuredID := mustFontID(t, fonts, font)
	if originalID == configuredID {
		t.Fatalf("configured id %d equals original id", configuredID)
	}
	if got := mustFontID(t, fonts, font); got != configuredID {
		t.Fatalf("font id = %d, want %d", got, configuredID)
	}
	if _, ok := fonts.GlyphForChar(configuredID, 'm'); !ok {
		t.Fatal("no glyph for 'm'")
	}
	layout := fonts.LayoutLine("office", gpui.Px(16.), []gpui.FontRun{{FontID: configuredID, Len: 6}})
	if layout.Len != 6 {
		t.Fatalf("len = %d", layout.Len)
	}
	if len(layout.Runs) == 0 {
		t.Fatal("no runs")
	}
}

func TestTextSystem_LayoutLineSetsALoneRunAtItsExactSize(t *testing.T) {
	fonts := NewAppleTextSystem()
	fontID := mustFontID(t, fonts, gpui.NewFont(".SystemUIFont"))
	text := "Hamburgefonstiv 0123 Wafer"
	bom := "\ufeff"
	withBOM := bom + text
	for _, size := range []float32{11., 17., 22.} {
		fonts.mu.RLock()
		exact := fonts.state.fonts[fontID].nativeFont().CloneWithFontSize(float64(size))
		fonts.mu.RUnlock()
		str := ct.AttributedStringCreateMutable()
		cfText := cf.StringCreate(text)
		str.Ref().ReplaceString(cf.Range{}, cfText.Ref())
		cfText.Release()
		str.Ref().SetAttribute(cf.Range{Location: 0, Length: str.Ref().Len()}, ct.FontAttributeName, exact.TypeRef())
		line := ct.LineCreateWithAttributedString(str.Ref())
		width := float32(line.Ref().TypographicBounds().Width)
		line.Release()
		str.Release()
		exact.Release()

		layout := fonts.LayoutLine(text, gpui.Px(size), []gpui.FontRun{{FontID: fontID, Len: len(text)}})
		if float32(layout.Width) != width {
			t.Errorf("%v px: width %v, want %v", size, layout.Width, width)
		}

		layout = fonts.LayoutLine(withBOM, gpui.Px(size), []gpui.FontRun{
			{FontID: fontID, Len: len(bom)},
			{FontID: fontID, Len: len(text)},
		})
		if float32(layout.Width) != width {
			t.Errorf("%v px after a BOM run: width %v, want %v", size, layout.Width, width)
		}
	}
}

type runSpec struct {
	fontID gpui.FontID
	len    int
}

func TestTextSystem_LayoutLineBreaksLigaturesBetweenRuns(t *testing.T) {
	fonts := NewAppleTextSystem()
	if err := fonts.AddFonts([][]byte{lilexRegular}); err != nil {
		t.Fatal(err)
	}
	glyphs := func(text string, specs ...runSpec) []gpui.GlyphID {
		runs := make([]gpui.FontRun, len(specs))
		for i, s := range specs {
			runs[i] = gpui.FontRun{FontID: s.fontID, Len: s.len}
		}
		var out []gpui.GlyphID
		for _, run := range fonts.LayoutLine(text, gpui.Px(16.), runs).Runs {
			for _, glyph := range run.Glyphs {
				out = append(out, glyph.ID)
			}
		}
		return out
	}
	repeat := func(g gpui.GlyphID, n int) []gpui.GlyphID {
		return slices.Repeat([]gpui.GlyphID{g}, n)
	}

	lilex := mustFontID(t, fonts, gpui.NewFont("Lilex"))
	gt := glyphs(">", runSpec{lilex, 1})[0]
	if slices.Equal(glyphs(">>>", runSpec{lilex, 3}), repeat(gt, 3)) {
		t.Error(">>> as one run")
	}
	if got := glyphs(">>>", runSpec{lilex, 1}, runSpec{lilex, 1}, runSpec{lilex, 1}); !slices.Equal(got, repeat(gt, 3)) {
		t.Errorf("three runs: %v", got)
	}
	split := append(glyphs(">>", runSpec{lilex, 2}), gt)
	if got := glyphs(">>>", runSpec{lilex, 2}, runSpec{lilex, 1}); !slices.Equal(got, split) {
		t.Errorf("2+1: %v, want %v", got, split)
	}
	split = append([]gpui.GlyphID{gt}, glyphs(">>", runSpec{lilex, 2})...)
	if got := glyphs(">>>", runSpec{lilex, 1}, runSpec{lilex, 2}); !slices.Equal(got, split) {
		t.Errorf("1+2: %v, want %v", got, split)
	}

	zedMono := mustFontID(t, fonts, gpui.NewFont(".ZedMono"))
	noFallbacksFont := gpui.NewFont("Lilex")
	noFallbacksFont.Fallbacks = gpui.Some(gpui.FontFallbacksFromFonts([]string{}))
	noFallbacks := mustFontID(t, fonts, noFallbacksFont)
	for _, other := range []gpui.FontID{zedMono, noFallbacks} {
		if lilex == other {
			t.Errorf("font id %d equals lilex", other)
		}
		if got := glyphs(">>>", runSpec{lilex, 1}, runSpec{other, 1}, runSpec{lilex, 1}); !slices.Equal(got, repeat(gt, 3)) {
			t.Errorf("with %d: %v", other, got)
		}
	}

	times := mustFontID(t, fonts, gpui.NewFont("Times"))
	helvetica := mustFontID(t, fonts, gpui.NewFont("Helvetica"))
	system := mustFontID(t, fonts, gpui.NewFont(".SystemUIFont"))
	split = append(glyphs("f", runSpec{times, 1}), glyphs("i", runSpec{times, 1})...)
	if slices.Equal(glyphs("fi", runSpec{times, 2}), split) {
		t.Error("fi as one run")
	}
	if got := glyphs("fi", runSpec{times, 1}, runSpec{helvetica, 0}, runSpec{system, 0}, runSpec{times, 1}); !slices.Equal(got, split) {
		t.Errorf("fi with empty runs: %v, want %v", got, split)
	}

	if got := glyphs("لا", runSpec{helvetica, 4}); len(got) != 1 {
		t.Errorf("لا as one run: %d glyphs", len(got))
	}
	if got := glyphs("لا", runSpec{helvetica, 2}, runSpec{times, 2}); len(got) != 2 {
		t.Errorf("لا split: %d glyphs", len(got))
	}
}

func TestTextSystem_LayoutLineBomChar(t *testing.T) {
	fonts := NewAppleTextSystem()
	fontID := mustFontID(t, fonts, gpui.NewFont("Helvetica"))
	line := "\ufeff"
	style := gpui.FontRun{FontID: fontID, Len: len(line)}

	layout := fonts.LayoutLine(line, gpui.Px(16.), []gpui.FontRun{style})
	if layout.Len != len(line) {
		t.Fatalf("len = %d", layout.Len)
	}
	if len(layout.Runs) != 0 {
		t.Fatalf("runs = %d, want 0", len(layout.Runs))
	}

	line = "a\ufeffb"
	style.Len = len(line)
	layout = fonts.LayoutLine(line, gpui.Px(16.), []gpui.FontRun{style})
	checkAB := func(layout gpui.LineLayout, line string) {
		t.Helper()
		if layout.Len != len(line) {
			t.Fatalf("len = %d", layout.Len)
		}
		if len(layout.Runs) != 1 {
			t.Fatalf("runs = %d, want 1", len(layout.Runs))
		}
		if len(layout.Runs[0].Glyphs) != 2 {
			t.Fatalf("glyphs = %d, want 2", len(layout.Runs[0].Glyphs))
		}
		if id := layout.Runs[0].Glyphs[0].ID; id != 68 {
			t.Errorf("glyph 0 = %d, want 68", id)
		}
		if id := layout.Runs[0].Glyphs[1].ID; id != 69 {
			t.Errorf("glyph 1 = %d, want 69", id)
		}
	}
	checkAB(layout, line)

	line = "\ufeffab"
	layout = fonts.LayoutLine(line, gpui.Px(16.), []gpui.FontRun{
		{Len: len("\ufeff"), FontID: fontID},
		{Len: len("ab"), FontID: fontID},
	})
	checkAB(layout, line)
}

func checkGlyphIndices(t *testing.T, layout gpui.LineLayout, text string) {
	t.Helper()
	for _, run := range layout.Runs {
		for _, glyph := range run.Glyphs {
			if glyph.Index >= len(text) {
				t.Errorf("Glyph index %d is out of bounds for text length %d", glyph.Index, len(text))
			}
		}
	}
}

func TestTextSystem_LayoutLineZwnjInsertion(t *testing.T) {
	fonts := NewAppleTextSystem()
	fontID := mustFontID(t, fonts, gpui.NewFont("Helvetica"))

	text := "hello world"
	layout := fonts.LayoutLine(text, gpui.Px(16.), []gpui.FontRun{{FontID: fontID, Len: 5}, {FontID: fontID, Len: 6}})
	if layout.Len != len(text) {
		t.Fatalf("len = %d", layout.Len)
	}
	checkGlyphIndices(t, layout, text)

	fontID2, err := fonts.FontID(gpui.NewFont("Times"))
	if err != nil {
		fontID2 = fontID
	}
	layout2 := fonts.LayoutLine(text, gpui.Px(16.), []gpui.FontRun{{FontID: fontID, Len: 5}, {FontID: fontID2, Len: 6}})
	if layout2.Len != len(text) {
		t.Fatalf("len = %d", layout2.Len)
	}
	checkGlyphIndices(t, layout2, text)
}

func TestTextSystem_LayoutLineZwnjEdgeCases(t *testing.T) {
	fonts := NewAppleTextSystem()
	fontID := mustFontID(t, fonts, gpui.NewFont("Helvetica"))

	text := "hello"
	layout := fonts.LayoutLine(text, gpui.Px(16.), []gpui.FontRun{{FontID: fontID, Len: 5}})
	if layout.Len != len(text) {
		t.Fatalf("len = %d", layout.Len)
	}

	text = "abc"
	layout = fonts.LayoutLine(text, gpui.Px(16.), []gpui.FontRun{{FontID: fontID, Len: 1}, {FontID: fontID, Len: 1}, {FontID: fontID, Len: 1}})
	if layout.Len != len(text) {
		t.Fatalf("len = %d", layout.Len)
	}
	checkGlyphIndices(t, layout, text)

	layout = fonts.LayoutLine("", gpui.Px(16.), []gpui.FontRun{})
	if layout.Len != 0 {
		t.Fatalf("len = %d", layout.Len)
	}
	if len(layout.Runs) != 0 {
		t.Fatalf("runs = %d", len(layout.Runs))
	}
}

func TestFontKit_CoreTextToCSSFontWeight(t *testing.T) {
	for in, want := range map[float32]float32{-0.7: 100, 0.0: 400, 0.4: 700, 0.8: 900, 0.1: 450} {
		if got := coreTextToCSSFontWeight(in); got != want {
			t.Errorf("weight(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestFontKit_CoreTextToCSSFontStretch(t *testing.T) {
	for in, want := range map[float32]float32{0.0: 1.0, -1.0: 0.5, 1.0: 2.0, 0.85: 1.7} {
		if got := coreTextWidthToCSSStretchiness(in); got != want {
			t.Errorf("stretch(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestFontKit_FindBestMatch(t *testing.T) {
	candidates := []fontKitProperties{
		{style: fontKitStyleNormal, weight: 400, stretch: 1},
		{style: fontKitStyleItalic, weight: 400, stretch: 1},
		{style: fontKitStyleNormal, weight: 700, stretch: 1},
	}
	for _, c := range []struct {
		query fontKitProperties
		want  int
	}{
		{fontKitProperties{style: fontKitStyleNormal, weight: 400, stretch: 1}, 0},
		{fontKitProperties{style: fontKitStyleItalic, weight: 700, stretch: 1}, 1},
		{fontKitProperties{style: fontKitStyleNormal, weight: 600, stretch: 1}, 2},
		{fontKitProperties{style: fontKitStyleOblique, weight: 400, stretch: 1}, 1},
	} {
		got, err := findBestMatch(candidates, c.query)
		if err != nil || got != c.want {
			t.Errorf("findBestMatch(%+v) = %d, %v; want %d", c.query, got, err, c.want)
		}
	}
	if _, err := findBestMatch(nil, candidates[0]); err == nil {
		t.Error("empty candidates matched")
	}
}
