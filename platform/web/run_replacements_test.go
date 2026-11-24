//go:build js && wasm

package web

import (
	"slices"
	"testing"
	"unsafe"

	"github.com/megascan/gpui-go"
)

func sourceGlyph(index int, x float32, id uint32, color bool) gpui.ShapedGlyph {
	return gpui.ShapedGlyph{
		ID:       gpui.GlyphID(id),
		Position: gpui.NewPoint(gpui.Px(x), gpui.Px(2.0)),
		Index:    index,
		IsEmoji:  color,
	}
}

func TestRunReplacements_CandidatesCaptureBoundsWithoutAFlattenedVector(t *testing.T) {
	text := "中a文 "
	layout := testLayout(2, 0, 2)
	layout.Width = gpui.Px(50.0)
	layout.Runs[0].Glyphs = []gpui.ShapedGlyph{sourceGlyph(0, 0.0, 0, false), sourceGlyph(3, 10.0, 1, false)}
	layout.Runs[2].Glyphs = []gpui.ShapedGlyph{sourceGlyph(4, 25.0, 0, false), sourceGlyph(7, 40.0, 3, false)}
	candidates := collectCandidates(text, []gpui.FontRun{{Len: len(text), FontID: 7}}, &layout)
	if len(candidates) != 2 {
		t.Fatalf("len = %d", len(candidates))
	}
	first := candidates[0]
	if first.source != (gpui.Range{Start: 0, End: 3}) || first.fontID != 7 {
		t.Fatalf("first = %+v", first)
	}
	want := glyphSpan{
		glyphRange:  gpui.Range{Start: 0, End: 1},
		position:    gpui.NewPoint(gpui.Px(0.0), gpui.Px(2.0)),
		endX:        gpui.Px(10.0),
		contiguous:  true,
		missing:     true,
		nativeColor: false,
	}
	if first.glyphs == nil || *first.glyphs != want {
		t.Fatalf("first glyphs = %+v", first.glyphs)
	}
	last := candidates[len(candidates)-1]
	if last.source != (gpui.Range{Start: 4, End: 7}) {
		t.Fatalf("last source = %+v", last.source)
	}
	span := last.glyphs
	if span == nil {
		t.Fatal("missing glyph span")
	}
	if span.glyphRange != (gpui.Range{Start: 2, End: 3}) || span.position.X != gpui.Px(25.0) || span.endX != gpui.Px(40.0) {
		t.Fatalf("last span = %+v", span)
	}
}

func TestRunReplacements_CandidatesKeepGraphemesAcrossSameFontBoundaries(t *testing.T) {
	text := "が"
	layout := testLayout(1, 1)
	layout.Runs[0].Glyphs = []gpui.ShapedGlyph{sourceGlyph(0, 0.0, 0, false)}
	layout.Runs[1].Glyphs = []gpui.ShapedGlyph{sourceGlyph(3, 10.0, 0, false)}
	fontRuns := []gpui.FontRun{{Len: 3, FontID: 7}, {Len: 3, FontID: 7}}
	candidates := collectCandidates(text, fontRuns, &layout)
	if len(candidates) != 1 {
		t.Fatalf("len = %d", len(candidates))
	}
	span := candidates[0].glyphs
	if span == nil {
		t.Fatal("missing glyph span")
	}
	if span.glyphRange != (gpui.Range{Start: 0, End: 2}) || span.position.X != 0 || span.endX != layout.Width || !span.contiguous {
		t.Fatalf("span = %+v", span)
	}

	fontRuns[1].FontID = 8
	if got := collectCandidates(text, fontRuns, &layout); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestRunReplacements_CandidatesHandleNonmonotonicAndDisjointSourceIndices(t *testing.T) {
	layout := testLayout(3)
	layout.Width = gpui.Px(50.0)
	layout.Runs[0].Glyphs = []gpui.ShapedGlyph{
		sourceGlyph(3, 0.0, 0, false),
		sourceGlyph(0, 20.0, 0, false),
		sourceGlyph(3, 30.0, 0, false),
	}
	candidates := collectCandidates("中文", []gpui.FontRun{{Len: 6, FontID: 7}}, &layout)
	if len(candidates) == 0 || candidates[0].glyphs == nil {
		t.Fatal("missing first span")
	}
	first := candidates[0].glyphs
	if first.glyphRange != (gpui.Range{Start: 1, End: 2}) || first.position.X != gpui.Px(20.0) || first.endX != gpui.Px(30.0) || !first.contiguous {
		t.Fatalf("first = %+v", first)
	}
	last := candidates[len(candidates)-1].glyphs
	if last == nil {
		t.Fatal("missing last span")
	}
	if last.glyphRange != (gpui.Range{Start: 0, End: 3}) || last.position.X != 0 || last.endX != layout.Width || last.contiguous {
		t.Fatalf("last = %+v", last)
	}
}

func TestRunReplacements_CandidatesPreserveNativeCoverageAndColor(t *testing.T) {
	layout := testLayout(1)
	fontRuns := []gpui.FontRun{{Len: len("\U0001f600"), FontID: 7}}
	for _, tc := range []struct {
		id    uint32
		color bool
	}{{0, false}, {1, false}, {0, true}, {1, true}} {
		layout.Runs[0].Glyphs = []gpui.ShapedGlyph{sourceGlyph(0, 0.0, tc.id, tc.color)}
		candidates := collectCandidates("\U0001f600", fontRuns, &layout)
		if len(candidates) == 0 {
			t.Fatal("missing candidate")
		}
		c := candidates[0]
		if !c.color {
			t.Fatal("candidate not color")
		}
		if c.glyphs == nil {
			t.Fatal("missing span")
		}
		if c.glyphs.missing != (tc.id == 0) || c.glyphs.nativeColor != tc.color {
			t.Fatalf("span = %+v for %+v", c.glyphs, tc)
		}
	}
	layout.Runs[0].Glyphs = layout.Runs[0].Glyphs[:0]
	candidates := collectCandidates("\U0001f600", fontRuns, &layout)
	if len(candidates) == 0 {
		t.Fatal("missing candidate")
	}
	if candidates[0].glyphs != nil {
		t.Fatal("unexpected span")
	}
}

func testGlyph(index int) gpui.ShapedGlyph {
	return gpui.ShapedGlyph{
		ID:       gpui.GlyphID(index),
		Position: gpui.NewPoint(gpui.Px(float32(index)*10.0), gpui.Px(2.0)),
		Index:    index * 2,
		IsEmoji:  false,
	}
}

func testLayout(runLengths ...int) gpui.LineLayout {
	index := 0
	runs := make([]gpui.ShapedRun, 0, len(runLengths))
	for font, length := range runLengths {
		glyphs := make([]gpui.ShapedGlyph, 0, length)
		for i := index; i < index+length; i++ {
			glyphs = append(glyphs, testGlyph(i))
		}
		index += length
		runs = append(runs, gpui.ShapedRun{FontID: gpui.FontID(font), Glyphs: glyphs})
	}
	return gpui.LineLayout{
		Runs:     runs,
		Width:    gpui.Px(float32(index) * 10.0),
		FontSize: gpui.Px(14.0),
		Ascent:   gpui.Px(11.0),
		Descent:  gpui.Px(3.0),
		Len:      index * 2,
	}
}

func testReplacement(start, end int, fontID int, delta float32) replacement {
	glyph := testGlyph(start)
	glyph.ID = 999
	glyph.Position.X += gpui.Px(1.0)
	glyph.Position.Y = gpui.Px(7.0)
	glyph.IsEmoji = true
	return replacement{
		glyphRange: gpui.Range{Start: start, End: end},
		fontID:     gpui.FontID(fontID),
		glyph:      glyph,
		widthDelta: gpui.Px(delta),
	}
}

func referenceReplacements(layout *gpui.LineLayout, replacements []replacement) {
	original := layout.Runs
	layout.Runs = nil
	next := 0
	shift := gpui.Pixels(0)
	consumedEnd := 0
	index := 0
	for _, run := range original {
		for _, glyph := range run.Glyphs {
			i := index
			index++
			if i < consumedEnd {
				continue
			}
			fontID := run.FontID
			if next < len(replacements) && replacements[next].glyphRange.Start == i {
				repl := replacements[next]
				next++
				glyph = repl.glyph
				glyph.Position.X += shift
				shift += repl.widthDelta
				consumedEnd = repl.glyphRange.End
				fontID = repl.fontID
			} else {
				glyph.Position.X += shift
			}
			if n := len(layout.Runs); n > 0 && layout.Runs[n-1].FontID == fontID {
				layout.Runs[n-1].Glyphs = append(layout.Runs[n-1].Glyphs, glyph)
			} else {
				layout.Runs = append(layout.Runs, gpui.ShapedRun{FontID: fontID, Glyphs: []gpui.ShapedGlyph{glyph}})
			}
		}
	}
	layout.Width += shift
}

type flatGlyphRecord struct {
	fontID   gpui.FontID
	id       gpui.GlyphID
	position gpui.Point[gpui.Pixels]
	index    int
	isEmoji  bool
}

func assertLayoutsEqual(t *testing.T, actual, expected *gpui.LineLayout) {
	t.Helper()
	flatten := func(layout *gpui.LineLayout) []flatGlyphRecord {
		var out []flatGlyphRecord
		for _, run := range layout.Runs {
			for _, g := range run.Glyphs {
				out = append(out, flatGlyphRecord{run.FontID, g.ID, g.Position, g.Index, g.IsEmoji})
			}
		}
		return out
	}
	if a, e := flatten(actual), flatten(expected); !slices.Equal(a, e) {
		t.Fatalf("glyphs = %+v, want %+v", a, e)
	}
	if actual.Width != expected.Width || actual.FontSize != expected.FontSize || actual.Ascent != expected.Ascent ||
		actual.Descent != expected.Descent || actual.Len != expected.Len {
		t.Fatalf("metrics = %+v, want %+v", actual, expected)
	}
}

func TestRunReplacements_NoReplacementsPreservesEvenEmptyRuns(t *testing.T) {
	actual := testLayout(0, 3, 0)
	pointer := unsafe.SliceData(actual.Runs)
	glyphPointer := unsafe.SliceData(actual.Runs[1].Glyphs)
	applyReplacements(&actual, nil)
	if len(actual.Runs) != 3 || unsafe.SliceData(actual.Runs) != pointer || unsafe.SliceData(actual.Runs[1].Glyphs) != glyphPointer {
		t.Fatal("storage changed")
	}
	expected := testLayout(0, 3, 0)
	assertLayoutsEqual(t, &actual, &expected)
	empty := testLayout()
	applyReplacements(&empty, nil)
	twoEmpty := testLayout(0, 0)
	applyReplacements(&twoEmpty, nil)
}

func TestRunReplacements_WholeRunReusesBothAllocations(t *testing.T) {
	actual := testLayout(8)
	actual.Runs = slices.Grow(actual.Runs, 4)
	actual.Runs[0].Glyphs = slices.Grow(actual.Runs[0].Glyphs, 10)
	runsPointer, runsCap := unsafe.SliceData(actual.Runs), cap(actual.Runs)
	glyphsPointer, glyphsCap := unsafe.SliceData(actual.Runs[0].Glyphs), cap(actual.Runs[0].Glyphs)
	applyReplacements(&actual, []replacement{testReplacement(0, 8, 9, -20.0)})
	if unsafe.SliceData(actual.Runs) != runsPointer || cap(actual.Runs) != runsCap {
		t.Fatal("runs storage changed")
	}
	if unsafe.SliceData(actual.Runs[0].Glyphs) != glyphsPointer || cap(actual.Runs[0].Glyphs) != glyphsCap {
		t.Fatal("glyph storage changed")
	}
	if actual.Runs[0].FontID != 9 || len(actual.Runs[0].Glyphs) != 1 {
		t.Fatalf("run = %+v", actual.Runs[0])
	}
}

func TestRunReplacements_SplitsReuseOuterPrefixUnaffectedAndEmptiedBuffers(t *testing.T) {
	actual := testLayout(4, 4, 3)
	actual.Runs = slices.Grow(actual.Runs, 8)
	outerPointer, outerCap := unsafe.SliceData(actual.Runs), cap(actual.Runs)
	prefix := unsafe.SliceData(actual.Runs[0].Glyphs)
	emptiedPointer, emptiedCap := unsafe.SliceData(actual.Runs[1].Glyphs), cap(actual.Runs[1].Glyphs)
	unaffected := unsafe.SliceData(actual.Runs[2].Glyphs)
	applyReplacements(&actual, []replacement{testReplacement(2, 8, 9, -3.0)})
	if unsafe.SliceData(actual.Runs) != outerPointer || cap(actual.Runs) != outerCap {
		t.Fatal("outer storage changed")
	}
	if unsafe.SliceData(actual.Runs[0].Glyphs) != prefix {
		t.Fatal("prefix storage changed")
	}
	if unsafe.SliceData(actual.Runs[1].Glyphs) != emptiedPointer || cap(actual.Runs[1].Glyphs) != emptiedCap {
		t.Fatal("emptied buffer not reused")
	}
	if unsafe.SliceData(actual.Runs[2].Glyphs) != unaffected {
		t.Fatal("unaffected storage changed")
	}
	expected := testLayout(4, 4, 3)
	referenceReplacements(&expected, []replacement{testReplacement(2, 8, 9, -3.0)})
	assertLayoutsEqual(t, &actual, &expected)
}

func TestRunReplacements_MultipleSplitsAndShifts(t *testing.T) {
	actual := testLayout(7, 2, 3)
	actual.Runs = slices.Grow(actual.Runs, 12)
	outer := unsafe.SliceData(actual.Runs)
	expected := testLayout(7, 2, 3)
	replacements := func() []replacement {
		return []replacement{
			testReplacement(1, 2, 9, 3.0),
			testReplacement(3, 5, 8, -7.0),
			testReplacement(6, 10, 9, 2.0),
		}
	}
	applyReplacements(&actual, replacements())
	referenceReplacements(&expected, replacements())
	assertLayoutsEqual(t, &actual, &expected)
	if unsafe.SliceData(actual.Runs) != outer {
		t.Fatal("outer storage changed")
	}
}

func TestRunReplacements_ExhaustiveSmallStreamsMatchReference(t *testing.T) {
	for first := 0; first <= 4; first++ {
		for second := 0; second <= 4-first; second++ {
			lengths := []int{first, 0, second, 4 - first - second}
			for choices := 0; choices < 256; choices++ {
				replacements := func() []replacement {
					var out []replacement
					for index := range 4 {
						switch (choices >> (2 * index)) & 3 {
						case 0:
						case 1:
							out = append(out, testReplacement(index, index+1, 0, 2.0))
						case 2:
							out = append(out, testReplacement(index, index+1, 9, -3.0))
						default:
							if n := len(out); n > 0 && out[n-1].glyphRange.End == index {
								out[n-1].glyphRange.End++
							}
						}
					}
					return out
				}
				actual := testLayout(lengths...)
				expected := testLayout(lengths...)
				applyReplacements(&actual, replacements())
				referenceReplacements(&expected, replacements())
				assertLayoutsEqual(t, &actual, &expected)
			}
		}
	}
}
