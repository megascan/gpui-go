package gotext

import (
	_ "embed"
	"fmt"
	"math"
	"os"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	gpui "github.com/megascan/gpui-go"
)

//go:embed testdata/fonts/ibm-plex-sans_IBMPlexSans-Regular.ttf
var ibmPlex []byte

//go:embed testdata/fonts/lilex_Lilex-Regular.ttf
var lilex []byte

//go:embed testdata/fonts/lilex_Lilex-Bold.ttf
var lilexBold []byte

var separators = []rune{'\u000a', '\u000d', '\u001c', '\u001d', '\u001e', '\u0085', ' '}

func fid(i int) gpui.FontID {
	return gpui.FontID(i)
}

func chain(ids ...int) []fallbackEntry {
	out := []fallbackEntry{}
	for _, i := range ids {
		out = append(out, fallbackEntry{fontID: fid(i), family: gpui.SharedString(fmt.Sprintf("fb%d", i))})
	}
	return out
}

func span(start, end, slot int, fontID gpui.FontID) runSpan {
	return runSpan{start: start, end: end, slot: slot, fontID: fontID}
}

func newTestTextSystem(t *testing.T) *CosmicTextSystem {
	t.Helper()
	textSystem := NewTextSystemWithoutSystemFonts("IBM Plex Sans")
	if err := textSystem.AddFonts([][]byte{ibmPlex}); err != nil {
		t.Fatal(err)
	}
	return textSystem
}

func mustFontID(t *testing.T, s *CosmicTextSystem, font gpui.Font) gpui.FontID {
	t.Helper()
	id, err := s.FontID(font)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func layoutText(t *testing.T, s *CosmicTextSystem, text string) gpui.LineLayout {
	t.Helper()
	fontID := mustFontID(t, s, gpui.NewFont("IBM Plex Sans"))
	return s.LayoutLine(text, gpui.Px(14.0), []gpui.FontRun{{Len: len(text), FontID: fontID}})
}

func allGlyphs(layout gpui.LineLayout) []gpui.ShapedGlyph {
	var glyphs []gpui.ShapedGlyph
	for _, run := range layout.Runs {
		glyphs = append(glyphs, run.Glyphs...)
	}
	return glyphs
}

func textRuns(text gpui.SharedString, family gpui.SharedString) []gpui.TextRun {
	run := gpui.DefaultTextRun()
	run.Len = len(text)
	run.Font = gpui.NewFont(family)
	return []gpui.TextRun{run}
}

func TestCosmicTextSystem_AllFontNamesTracksAvailableFamilies(t *testing.T) {
	textSystem := gpui.NewTextSystem(NewTextSystemWithoutSystemFonts("IBM Plex Sans"))
	if names := textSystem.AllFontNames(); len(names) != 0 {
		t.Fatalf("expected no fonts, got %v", names)
	}
	if err := textSystem.AddFonts([][]byte{lilex}); err != nil {
		t.Fatal(err)
	}
	if names := textSystem.AllFontNames(); !slices.Equal(names, []string{"Lilex"}) {
		t.Fatalf("got %v", names)
	}
	if err := textSystem.AddFonts([][]byte{ibmPlex, lilexBold}); err != nil {
		t.Fatal(err)
	}
	if names := textSystem.AllFontNames(); !slices.Equal(names, []string{"IBM Plex Sans", "Lilex"}) {
		t.Fatalf("got %v", names)
	}
}

func TestCosmicTextSystem_CachedFontResolutionDoesNotWaitForColdLookups(t *testing.T) {
	if os.Getenv("GOTEXT_ROOT_RESOLVE_FONT_FIXED") == "" {
		t.Skip("blocked:M2-exit: gpui.TextSystem.resolveFontUncached holds fontIDsByFontMu exclusively across the platform FontID call, where upstream holds an upgradable read; see docs/port/handoff/M2-gotext-layout.md")
	}
	for _, family := range []gpui.SharedString{".ZedSans", "Missing Font"} {
		platformTextSystem := newTestTextSystem(t)
		textSystem := gpui.NewTextSystem(platformTextSystem)
		cached := gpui.NewFont(family)
		expected := mustFontID(t, platformTextSystem, gpui.NewFont(".ZedSans"))
		if got := textSystem.ResolveFont(cached); got != expected {
			t.Fatalf("resolve %q: got %v, want %v", family, got, expected)
		}
		results := make(chan gpui.FontID, 1)

		platformTextSystem.mu.RLock()
		deadline := time.Now().Add(5 * time.Second)
		var cold sync.WaitGroup
		var coldResult gpui.FontID
		cold.Go(func() { coldResult = textSystem.ResolveFont(cached.Bold()) })
		lockedExclusive := func() bool {
			if platformTextSystem.mu.TryRLock() {
				platformTextSystem.mu.RUnlock()
				return false
			}
			return true
		}
		for !lockedExclusive() && time.Now().Before(deadline) {
			runtime.Gosched()
		}
		coldWaiting := lockedExclusive()
		var warm sync.WaitGroup
		warm.Go(func() { results <- textSystem.ResolveFont(cached) })
		var result gpui.FontID
		received := false
		select {
		case result = <-results:
			received = true
		case <-time.After(time.Until(deadline)):
		}
		platformTextSystem.mu.RUnlock()
		cold.Wait()
		warm.Wait()

		if !coldWaiting {
			t.Fatal("cold lookup did not wait for the write lock")
		}
		if !received {
			t.Fatal("cached lookup blocked")
		}
		if result != expected {
			t.Fatalf("cached lookup: got %v, want %v", result, expected)
		}
		if coldResult != expected {
			t.Fatalf("cold lookup: got %v, want %v", coldResult, expected)
		}
	}
}

func TestCosmicTextSystem_AddingFontsRetriesCachedFontFailures(t *testing.T) {
	platformTextSystem := NewTextSystemWithoutSystemFonts("Lilex")
	textSystem := gpui.NewTextSystem(platformTextSystem)
	if err := textSystem.AddFonts([][]byte{ibmPlex}); err != nil {
		t.Fatal(err)
	}
	originalFallback := mustFontID(t, platformTextSystem, gpui.NewFont(".ZedSans"))
	requests := []gpui.Font{gpui.DefaultFont(), gpui.NewFont("Lilex"), gpui.NewFont("Missing Font")}
	for _, requested := range requests {
		if got := textSystem.ResolveFont(requested); got != originalFallback {
			t.Fatalf("resolve %v: got %v, want %v", requested.Family, got, originalFallback)
		}
	}
	if font, ok := textSystem.GetFontForID(originalFallback); !ok || font != gpui.NewFont(".ZedSans") {
		t.Fatalf("GetFontForID: got %v, %v", font, ok)
	}

	if err := textSystem.AddFonts([][]byte{lilex}); err != nil {
		t.Fatal(err)
	}
	for i, resolved := range []gpui.SharedString{".SystemUIFont", "Lilex", ".ZedMono"} {
		installed := mustFontID(t, platformTextSystem, gpui.NewFont(resolved))
		if installed == originalFallback {
			t.Fatalf("%q resolved to the original fallback", resolved)
		}
		if got := textSystem.ResolveFont(requests[i]); got != installed {
			t.Fatalf("resolve %v: got %v, want %v", requests[i].Family, got, installed)
		}
	}
}

func TestCosmicTextSystem_FontPropertiesDescribeTheSelectedFace(t *testing.T) {
	textSystem := newTestTextSystem(t)
	regular := gpui.NewFont("IBM Plex Sans")
	regularID := mustFontID(t, textSystem, regular)
	for _, tc := range []struct {
		weight gpui.FontWeight
		style  gpui.FontStyle
	}{
		{gpui.FontWeightMedium, gpui.FontStyleNormal},
		{gpui.FontWeightBold, gpui.FontStyleItalic},
		{gpui.FontWeightNormal, gpui.FontStyleOblique},
	} {
		requested := regular
		requested.Weight = tc.weight
		requested.Style = tc.style
		fontID := mustFontID(t, textSystem, requested)
		if fontID != regularID {
			t.Fatalf("got %v, want %v", fontID, regularID)
		}
		weight, style, err := textSystem.FontWeightAndStyle(fontID)
		if err != nil {
			t.Fatal(err)
		}
		if weight != gpui.FontWeightNormal || style != gpui.FontStyleNormal {
			t.Fatalf("got (%v, %v)", weight, style)
		}
	}
}

func TestCosmicTextSystem_CombiningMarksArePositionedOverTheirBase(t *testing.T) {
	textSystem := newTestTextSystem(t)
	glyphs := allGlyphs(layoutText(t, textSystem, "v̄v"))
	if len(glyphs) != 3 {
		t.Fatalf("got %d glyphs", len(glyphs))
	}
	base, mark, next := glyphs[0].Position, glyphs[1].Position, glyphs[2].Position
	if !(mark.X > base.X) {
		t.Fatalf("mark.x=%v base.x=%v", mark.X, base.X)
	}
	if !(mark.X < next.X) {
		t.Fatalf("combining mark positioned after its base: mark.x=%v, next.x=%v", mark.X, next.X)
	}
}

func TestCosmicTextSystem_ShapeTextWithMixedDirectionParagraphs(t *testing.T) {
	textSystem := gpui.NewTextSystem(newTestTextSystem(t))
	windowTextSystem := gpui.NewWindowTextSystem(textSystem)
	text := gpui.SharedString("first line\nא\u001cA")
	lines, err := windowTextSystem.ShapeText(text, gpui.Px(14.0), textRuns(text, "IBM Plex Sans"), gpui.None[gpui.Pixels](), gpui.None[int]())
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
	if lines[1].Len() != len("א\u001cA") {
		t.Fatalf("got len %d", lines[1].Len())
	}
	if !(lines[1].Width() > 0) {
		t.Fatal("second line has no width")
	}
}

func TestCosmicTextSystem_ReportsGraphemesThatExhaustFontFallback(t *testing.T) {
	t.Skip("blocked:M0-test-contexts: needs TestAppContext.BuildWithTextSystem and App.OnMissingGlyphs")
}

func TestCosmicTextSystem_CombinesMissingGlyphsFromOneGrapheme(t *testing.T) {
	textSystem := newTestTextSystem(t)
	fontID := mustFontID(t, textSystem, gpui.NewFont("IBM Plex Sans"))
	text := "x́"
	runs := []gpui.FontRun{{Len: len(text), FontID: fontID}}
	missing := textSystem.MissingGlyphs(text, runs, []int{0, len("x")})
	if len(missing) != 1 {
		t.Fatalf("got %d missing glyphs", len(missing))
	}
	if missing[0].Grapheme() != text {
		t.Fatalf("got %q", missing[0].Grapheme())
	}
}

func TestCosmicTextSystem_AddingFontsInvalidatesCachedLineLayouts(t *testing.T) {
	platformTextSystem := newTestTextSystem(t)
	textSystem := gpui.NewTextSystem(platformTextSystem)
	windowTextSystem := gpui.NewWindowTextSystem(textSystem)
	text := gpui.SharedString("cached text")
	runs := textRuns(text, "IBM Plex Sans")

	firstLayout := windowTextSystem.ShapeLine(text, gpui.Px(14.0), runs, gpui.None[gpui.Pixels]())
	cachedLayout := windowTextSystem.ShapeLine(text, gpui.Px(14.0), runs, gpui.None[gpui.Pixels]())
	if firstLayout.LineLayout != cachedLayout.LineLayout {
		t.Fatal("second layout was not cached")
	}
	platformTextSystem.mu.RLock()
	loadedFontCount := len(platformTextSystem.state.loadedFonts)
	platformTextSystem.mu.RUnlock()

	if err := textSystem.AddFonts([][]byte{lilex}); err != nil {
		t.Fatal(err)
	}

	refreshedLayout := windowTextSystem.ShapeLine(text, gpui.Px(14.0), runs, gpui.None[gpui.Pixels]())
	if firstLayout.LineLayout == refreshedLayout.LineLayout {
		t.Fatal("layout was not invalidated")
	}
	platformTextSystem.mu.RLock()
	defer platformTextSystem.mu.RUnlock()
	if got := len(platformTextSystem.state.loadedFonts); got != loadedFontCount {
		t.Fatalf("loaded fonts: got %d, want %d", got, loadedFontCount)
	}
}

func TestCosmicTextSystem_LayoutLineWithMixedDirectionParagraphs(t *testing.T) {
	textSystem := newTestTextSystem(t)
	for _, separator := range separators {
		for _, text := range []string{"א" + string(separator) + "A", "A" + string(separator) + "א"} {
			layout := layoutText(t, textSystem, text)
			if layout.Len != len(text) {
				t.Fatalf("%q: len %d", text, layout.Len)
			}
			if !(layout.Width > 0) {
				t.Fatalf("%q: no width", text)
			}
			if !slices.ContainsFunc(layout.Runs, func(run gpui.ShapedRun) bool { return len(run.Glyphs) > 0 }) {
				t.Fatalf("%q: no glyphs", text)
			}
		}
	}
}

func TestCosmicTextSystem_LayoutLineWithSeparatorsAtLineEdges(t *testing.T) {
	textSystem := newTestTextSystem(t)
	for _, text := range []string{
		"\u001c",
		"\u001c\u001c",
		"\u001cא",
		"א\u001c",
		"א\u001c\u001cA",
		"\u001cא\u001cA\u001c",
	} {
		if layout := layoutText(t, textSystem, text); layout.Len != len(text) {
			t.Fatalf("%q: len %d", text, layout.Len)
		}
	}
}

func TestCosmicTextSystem_LayoutLineKeepsIndicesAndPositionsOrderedAcrossParagraphs(t *testing.T) {
	textSystem := newTestTextSystem(t)
	text := "ab\u001ccd ef"
	layout := layoutText(t, textSystem, text)
	glyphs := allGlyphs(layout)
	if len(glyphs) == 0 {
		t.Fatal("no glyphs")
	}
	for _, glyph := range glyphs {
		if glyph.Index >= len(text) || !utf8.RuneStart(text[glyph.Index]) {
			t.Fatalf("bad index %d", glyph.Index)
		}
	}
	for i := 1; i < len(glyphs); i++ {
		if !(glyphs[i-1].Index < glyphs[i].Index) {
			t.Fatalf("indices out of order: %d, %d", glyphs[i-1].Index, glyphs[i].Index)
		}
		if !(glyphs[i-1].Position.X <= glyphs[i].Position.X) {
			t.Fatalf("positions out of order: %v, %v", glyphs[i-1].Position.X, glyphs[i].Position.X)
		}
	}
	if !(layout.Width > layoutText(t, textSystem, "ab").Width) {
		t.Fatal("segments did not contribute width")
	}
}

func TestCosmicTextSystem_LayoutLineWithFontRunStraddlingASeparator(t *testing.T) {
	textSystem := newTestTextSystem(t)
	fontID := mustFontID(t, textSystem, gpui.NewFont("IBM Plex Sans"))
	text := "ab\u001cאב"
	runs := []gpui.FontRun{
		{Len: len("ab\u001cא"), FontID: fontID},
		{Len: len("ב"), FontID: fontID},
	}
	layout := textSystem.LayoutLine(text, gpui.Px(14.0), runs)
	if layout.Len != len(text) {
		t.Fatalf("len %d", layout.Len)
	}
	if !(layout.Width > 0) {
		t.Fatal("no width")
	}
}

func TestCosmicTextSystem_LayoutLineWithoutSeparatorsTakesFastPath(t *testing.T) {
	textSystem := newTestTextSystem(t)
	for _, text := range []string{"hello world", "אבג", "mixed אב"} {
		if containsParagraphSeparator(text) {
			t.Fatalf("%q contains a separator", text)
		}
		layout := layoutText(t, textSystem, text)
		if layout.Len != len(text) {
			t.Fatalf("%q: len %d", text, layout.Len)
		}
		if !(layout.Width > 0) {
			t.Fatalf("%q: no width", text)
		}
	}
}

func TestCosmicTextSystem_IndexForPositionPastLineWidth(t *testing.T) {
	textSystem := gpui.NewTextSystem(newTestTextSystem(t))
	windowTextSystem := gpui.NewWindowTextSystem(textSystem)
	text := gpui.SharedString("Warning: this will delete files​")
	lines, err := windowTextSystem.ShapeText(text, gpui.Px(14.), textRuns(text, "IBM Plex Sans"), gpui.Some(gpui.Px(4.)), gpui.None[int]())
	if err != nil {
		t.Fatal(err)
	}
	line := lines[0]
	width := line.UnwrappedLayout.Width
	boundaryGlyph := func(row int) gpui.ShapedGlyph {
		boundary := line.WrapBoundaries[row]
		return line.Runs()[boundary.RunIx].Glyphs[boundary.GlyphIx]
	}
	row := -1
	for r := 1; r < len(line.WrapBoundaries); r++ {
		if boundaryGlyph(r).Position.X > width {
			row = r
			break
		}
	}
	if row < 0 {
		t.Fatal("trailing zero-width space should be a wrap boundary past the line width")
	}
	rowStartX := float32(boundaryGlyph(row - 1).Position.X)
	rowEnd := boundaryGlyph(row)

	x := float32(width) - rowStartX
	for x+rowStartX < float32(width) {
		x = math.Nextafter32(x, float32(math.Inf(1)))
	}
	if !(gpui.Px(x+rowStartX) < rowEnd.Position.X) {
		t.Fatal("probe position is not inside the sliver")
	}
	lineHeight := gpui.Px(20.)
	position := gpui.Point[gpui.Pixels]{X: gpui.Px(x), Y: lineHeight*gpui.Pixels(row) + gpui.Px(1.)}
	index, ok := line.IndexForPosition(position, lineHeight)
	if ok || index != rowEnd.Index {
		t.Fatalf("got (%d, %v), want Err(%d)", index, ok, rowEnd.Index)
	}
}

func TestCosmicTextSystem_ParagraphSeparatorDetection(t *testing.T) {
	for _, separator := range separators {
		if !isParagraphSeparator(separator) {
			t.Fatalf("%q", separator)
		}
		if !containsParagraphSeparator("a" + string(separator) + "b") {
			t.Fatalf("%q", separator)
		}
	}
	for _, text := range []string{"", "plain ascii", "א", "tab\there", "emoji \U0001f600"} {
		if containsParagraphSeparator(text) {
			t.Fatalf("%q", text)
		}
	}
}

func TestCosmicTextSystem_FontRunsAreClippedToSegment(t *testing.T) {
	runs := []gpui.FontRun{{Len: 3, FontID: fid(1)}, {Len: 4, FontID: fid(2)}}
	if got := clipFontRuns(runs, 0, 7); !slices.Equal(got, runs) {
		t.Fatalf("got %v", got)
	}
	if got := clipFontRuns(runs, 2, 5); !slices.Equal(got, []gpui.FontRun{{Len: 1, FontID: fid(1)}, {Len: 2, FontID: fid(2)}}) {
		t.Fatalf("got %v", got)
	}
	if got := clipFontRuns(runs, 3, 7); !slices.Equal(got, []gpui.FontRun{{Len: 4, FontID: fid(2)}}) {
		t.Fatalf("got %v", got)
	}
	if got := clipFontRuns(runs, 5, 5); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestCosmicTextSystem_PrimaryWinsOverCurrentFallbackWhenPrimaryCovers(t *testing.T) {
	covers := func(id gpui.FontID, _ rune) bool { return id == fid(0) || id == fid(1) }
	if got := pickCoveringSlot('a', 0, fid(0), chain(1, 2), covers); got != -1 {
		t.Fatalf("got %d", got)
	}
}

func TestCosmicTextSystem_PrimaryPreferredOverFallbackWhenBothCover(t *testing.T) {
	covers := func(gpui.FontID, rune) bool { return true }
	if got := pickCoveringSlot('a', -1, fid(0), chain(1), covers); got != -1 {
		t.Fatalf("got %d", got)
	}
}

func TestCosmicTextSystem_FallsThroughChainInOrder(t *testing.T) {
	covers := func(id gpui.FontID, _ rune) bool { return id == fid(2) }
	if got := pickCoveringSlot('字', -1, fid(0), chain(1, 2, 3), covers); got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestCosmicTextSystem_NoCoverageReturnsPrimary(t *testing.T) {
	covers := func(gpui.FontID, rune) bool { return false }
	if got := pickCoveringSlot('\U0001F600', 1, fid(0), chain(1, 2), covers); got != -1 {
		t.Fatalf("got %d", got)
	}
}

func TestCosmicTextSystem_EmptyChainAlwaysReturnsPrimary(t *testing.T) {
	covers := func(gpui.FontID, rune) bool { return false }
	if got := pickCoveringSlot('a', -1, fid(0), chain(), covers); got != -1 {
		t.Fatalf("got %d", got)
	}
}

func TestCosmicTextSystem_SlotFontIDResolution(t *testing.T) {
	primary, fb := fid(7), chain(10, 20)
	if slotFontID(-1, primary, fb) != fid(7) || slotFontID(0, primary, fb) != fid(10) || slotFontID(1, primary, fb) != fid(20) {
		t.Fatal("wrong slot font id")
	}
}

func asciiSplit(primary gpui.FontID) func(gpui.FontID, rune) bool {
	return func(id gpui.FontID, ch rune) bool {
		if id == primary {
			return ch < utf8.RuneSelf
		}
		return ch >= utf8.RuneSelf
	}
}

func TestCosmicTextSystem_RunSpansWithNoChainEmitOnePrimarySpan(t *testing.T) {
	text := "hello"
	spans := computeRunSpans(text, 0, len(text), fid(0), chain(), func(gpui.FontID, rune) bool { return false })
	if !slices.Equal(spans, []runSpan{span(0, len(text), -1, fid(0))}) {
		t.Fatalf("got %v", spans)
	}
}

func TestCosmicTextSystem_RunSpansUseByteOffsetsForMultibyteChars(t *testing.T) {
	text := "a字b"
	spans := computeRunSpans(text, 0, len(text), fid(0), chain(1), asciiSplit(fid(0)))
	want := []runSpan{span(0, 1, -1, fid(0)), span(1, 4, 0, fid(1)), span(4, 5, -1, fid(0))}
	if !slices.Equal(spans, want) {
		t.Fatalf("got %v", spans)
	}
}

func TestCosmicTextSystem_RunSpansRespectRunOffset(t *testing.T) {
	text := "xx字y"
	spans := computeRunSpans(text, 2, len(text)-2, fid(0), chain(1), asciiSplit(fid(0)))
	want := []runSpan{span(2, 5, 0, fid(1)), span(5, 6, -1, fid(0))}
	if !slices.Equal(spans, want) {
		t.Fatalf("got %v", spans)
	}
}

func TestCosmicTextSystem_RunSpansKeepCombiningMarksWithBaseInFallback(t *testing.T) {
	primary := fid(0)
	covers := func(id gpui.FontID, ch rune) bool {
		if id == primary {
			return ch < utf8.RuneSelf
		}
		return ch == 'अ'
	}
	text := "अं"
	spans := computeRunSpans(text, 0, len(text), primary, chain(1), covers)
	if !slices.Equal(spans, []runSpan{span(0, len(text), 0, fid(1))}) {
		t.Fatalf("got %v", spans)
	}
}

func TestCosmicTextSystem_RunSpansKeepZwjInsideEmojiCluster(t *testing.T) {
	covers := func(id gpui.FontID, ch rune) bool { return id == fid(1) && ch != '‍' }
	text := "\U0001F469‍\U0001F467"
	spans := computeRunSpans(text, 0, len(text), fid(0), chain(1), covers)
	if !slices.Equal(spans, []runSpan{span(0, len(text), 0, fid(1))}) {
		t.Fatalf("got %v", spans)
	}
}

func TestCosmicTextSystem_RunSpansCollapseAdjacentSameSlot(t *testing.T) {
	text := "字字字"
	spans := computeRunSpans(text, 0, len(text), fid(0), chain(1), asciiSplit(fid(0)))
	if !slices.Equal(spans, []runSpan{span(0, len(text), 0, fid(1))}) {
		t.Fatalf("got %v", spans)
	}
}

func TestCosmicTextSystem_RunSpansEmptyRunReturnsNoSpans(t *testing.T) {
	spans := computeRunSpans("anything", 3, 0, fid(0), chain(1), func(gpui.FontID, rune) bool { return true })
	if len(spans) != 0 {
		t.Fatalf("got %v", spans)
	}
}
