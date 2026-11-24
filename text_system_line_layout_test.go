package gpui

import (
	"slices"
	"sync/atomic"
	"testing"
	"unsafe"
)

func TestLineLayout_ShapeLineSharesItsTextWithTheLayoutCache(t *testing.T) {
	textSystem := NewWindowTextSystem(NewTextSystem(NewNoopTextSystem()))
	text := SharedString("let value = compute_something(argument_one, argument_two);")
	run := DefaultTextRun()
	run.Len = len(text)
	run.Font = NewFont(".ZedMono")
	runs := []TextRun{run}

	textSystem.ShapeLine(text, Px(14.), runs, None[Pixels]())
	var key lineLayoutCacheKey
	found := false
	for k := range textSystem.lineLayoutCache.state.currentFrame.lines {
		key, found = k, true
		break
	}
	if !found {
		t.Fatalf("no cached line")
	}
	if unsafe.StringData(key.text) != unsafe.StringData(string(text)) {
		t.Fatalf("cache key does not share the text")
	}
}

func glyphAt(x float32, index int) ShapedGlyph {
	return ShapedGlyph{
		ID:       GlyphID(0),
		Position: NewPoint(Px(x), Px(0.)),
		Index:    index,
		IsEmoji:  false,
	}
}

func makeLayout(glyphs []ShapedGlyph) LineLayout {
	return LineLayout{
		FontSize: Px(16.),
		Width:    Px(100.),
		Ascent:   Px(12.),
		Descent:  Px(4.),
		Runs:     []ShapedRun{{FontID: FontID(0), Glyphs: glyphs}},
		Len:      0,
	}
}

func glyphXPositions(layout *LineLayout) []float32 {
	var out []float32
	for _, g := range layout.Runs[0].Glyphs {
		out = append(out, float32(g.Position.X))
	}
	return out
}

func TestLineLayout_WrapPreservesShapedClusters(t *testing.T) {
	layout := makeLayout([]ShapedGlyph{glyphAt(0., 0), glyphAt(5., 0), glyphAt(10., 3), glyphAt(15., 3)})
	layout.Width = Px(20.)
	layout.Len = len("你好")
	got := layout.computeWrapBoundaries("你好", Px(15.), None[int](), &wrapScratch{})
	if want := []WrapBoundary{{RunIx: 0, GlyphIx: 2}}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	layout = makeLayout([]ShapedGlyph{glyphAt(0., 0), glyphAt(10., 0)})
	layout.Width = Px(20.)
	layout.Len = 1
	if got := layout.computeWrapBoundaries("a", Px(10.), None[int](), &wrapScratch{}); len(got) != 0 {
		t.Fatalf("got %v, want none", got)
	}
}

func TestLineLayout_WrapRightToLeftText(t *testing.T) {
	for _, tc := range []struct {
		text               string
		indices            []int
		expectedBoundaries []int
	}{
		{"אבג דהו", []int{11, 9, 7, 6, 4, 2, 0}, []int{4}},
		{"a אבג b", []int{0, 1, 6, 4, 2, 8, 9}, []int{2, 6}},
	} {
		var glyphs []ShapedGlyph
		for position, index := range tc.indices {
			glyphs = append(glyphs, glyphAt(float32(position)*10., index))
		}
		layout := makeLayout(glyphs)
		layout.Width = Px(float32(len(tc.indices)) * 10.)
		layout.Len = len(tc.text)
		var want []WrapBoundary
		for _, glyphIx := range tc.expectedBoundaries {
			want = append(want, WrapBoundary{RunIx: 0, GlyphIx: glyphIx})
		}
		if got := layout.computeWrapBoundaries(tc.text, Px(40.), None[int](), &wrapScratch{}); !slices.Equal(got, want) {
			t.Fatalf("%q: got %v, want %v", tc.text, got, want)
		}
	}
}

func TestLineLayout_WrapIgnoresBreaksInsideShapedClusters(t *testing.T) {
	text := "x a-bcde"
	var glyphs []ShapedGlyph
	for _, index := range []int{0, 1, 2, 5, 6, 7} {
		glyphs = append(glyphs, glyphAt(float32(index)*10., index))
	}
	layout := makeLayout(glyphs)
	layout.Width = Px(80.)
	layout.Len = len(text)
	got := layout.computeWrapBoundaries(text, Px(60.), None[int](), &wrapScratch{})
	if want := []WrapBoundary{{RunIx: 0, GlyphIx: 2}}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func assertPositions(t *testing.T, layout *LineLayout, want []float32) {
	t.Helper()
	if got := glyphXPositions(layout); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLineLayout_ForceWidthLatinUnchanged(t *testing.T) {
	cellWidth := Px(8.)
	layout := makeLayout([]ShapedGlyph{glyphAt(0., 0), glyphAt(8., 1), glyphAt(16., 2)})
	applyForceWidthToLayout(&layout, cellWidth)
	assertPositions(t, &layout, []float32{0., 8., 16.})
}

func TestLineLayout_ForceWidthCombiningMarksNotAdvanced(t *testing.T) {
	cellWidth := Px(8.)
	layout := makeLayout([]ShapedGlyph{glyphAt(0., 0), glyphAt(0., 3)})
	applyForceWidthToLayout(&layout, cellWidth)
	assertPositions(t, &layout, []float32{0., 0.})
}

func TestLineLayout_ForceWidthBaseAfterCombiningMark(t *testing.T) {
	cellWidth := Px(8.)
	layout := makeLayout([]ShapedGlyph{glyphAt(0., 0), glyphAt(0., 3), glyphAt(8., 6)})
	applyForceWidthToLayout(&layout, cellWidth)
	assertPositions(t, &layout, []float32{0., 0., 8.})
}

func TestLineLayout_ForceWidthMultipleCombiningMarks(t *testing.T) {
	cellWidth := Px(8.)
	layout := makeLayout([]ShapedGlyph{glyphAt(0., 0), glyphAt(0., 3), glyphAt(0., 6), glyphAt(8., 9)})
	applyForceWidthToLayout(&layout, cellWidth)
	assertPositions(t, &layout, []float32{0., 0., 0., 8.})
}

func TestLineLayout_ForceWidthCorrectsDriftedBasePositions(t *testing.T) {
	cellWidth := Px(8.)
	layout := makeLayout([]ShapedGlyph{glyphAt(0.5, 0), glyphAt(10.2, 1), glyphAt(19.8, 2)})
	applyForceWidthToLayout(&layout, cellWidth)
	assertPositions(t, &layout, []float32{0.5, 8., 16.})
}

func TestLineLayout_ForceWidthCombiningMarkAfterWithinToleranceBase(t *testing.T) {
	cellWidth := Px(8.)
	layout := makeLayout([]ShapedGlyph{glyphAt(0.5, 0), glyphAt(0.5, 3)})
	applyForceWidthToLayout(&layout, cellWidth)
	assertPositions(t, &layout, []float32{0.5, 0.5})
}

func TestLineLayout_WrapIgnoresTheSignOfAZeroWrapWidthInTheCache(t *testing.T) {
	runs := []FontRun{{Len: 2, FontID: FontID(1)}}

	coldCache := NewLineLayoutCache(NewNoopTextSystem(), &atomic.Uint64{})
	cold := coldCache.layoutWrappedLine("ab", Px(0.), runs, Some(Px(0.)), None[int]())

	warmCache := NewLineLayoutCache(NewNoopTextSystem(), &atomic.Uint64{})
	warmCache.layoutWrappedLine("ab", Px(0.), runs, Some(-Px(0.)), None[int]())
	warm := warmCache.layoutWrappedLine("ab", Px(0.), runs, Some(Px(0.)), None[int]())

	if !slices.Equal(warm.WrapBoundaries, cold.WrapBoundaries) {
		t.Fatalf("warm %v != cold %v", warm.WrapBoundaries, cold.WrapBoundaries)
	}
}

func TestLineLayout_BreakIndicesMatchICU(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []int
	}{
		{"x a-bcde", []int{0, 2, 4, 8}},
		{"hello world", []int{0, 6, 11}},
	} {
		got := slices.Collect(lineWrapperBreakIndices(tc.text))
		if !slices.Equal(got, tc.want) {
			t.Fatalf("%q: got %v, want %v", tc.text, got, tc.want)
		}
	}
}
