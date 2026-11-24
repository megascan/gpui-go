package gpui

import (
	"fmt"
	"slices"
	"testing"
)

func makeShapedLine(text string, glyphs [][2]float32, width float32, decorations []DecorationRun) ShapedLine {
	shapedGlyphs := make([]ShapedGlyph, 0, len(glyphs))
	for _, g := range glyphs {
		shapedGlyphs = append(shapedGlyphs, ShapedGlyph{
			ID:       0,
			Position: NewPoint(Px(g[1]), Px(0.)),
			Index:    int(g[0]),
		})
	}
	return ShapedLine{
		LineLayout: &LineLayout{
			FontSize: Px(16.),
			Width:    Px(width),
			Ascent:   Px(12.),
			Descent:  Px(4.),
			Runs:     []ShapedRun{{FontID: 0, Glyphs: shapedGlyphs}},
			Len:      len(text),
		},
		Text:           SharedString(text),
		decorationRuns: slices.Clone(decorations),
	}
}

type underlineStroke struct {
	rng    Range
	origin Point[Pixels]
	width  Pixels
	style  UnderlineStyle
}

type lineTestCanvas struct {
	paint func(window *Window, cx *App)
}

func (c *lineTestCanvas) ElementID() (ElementID, bool) { return ElementID{}, false }

func (c *lineTestCanvas) SourceLocation() *SourceLocation { return nil }

func (c *lineTestCanvas) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	style := DefaultStyle()
	style.Size.Width = Relative(1.).ToLength()
	style.Size.Height = Relative(1.).ToLength()
	return window.RequestLayout(style, nil, cx), struct{}{}
}

func (c *lineTestCanvas) Prepaint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, _ *Window, _ *App) struct{} {
	return struct{}{}
}

func (c *lineTestCanvas) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, _ *struct{}, window *Window, cx *App) {
	window.withElementOpacity(Some[float32](0.5), func(window *Window) struct{} {
		c.paint(window, cx)
		return struct{}{}
	})
}

func (c *lineTestCanvas) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](c)
}

type underlineHandlerTestView struct {
	paint func(window *Window, cx *App)
}

func (v *underlineHandlerTestView) Render(_ *Window, _ *Context[underlineHandlerTestView]) IntoElement {
	return &lineTestCanvas{paint: v.paint}
}

func lineTestUpdateWindow(cx *TestAppContext, window AnyWindowHandle, f func(window *Window, cx *App)) {
	_, err := AppContextUpdateWindow(cx, window, func(_ AnyView, w *Window, cx *App) struct{} {
		f(w, cx)
		return struct{}{}
	})
	if err != nil {
		panic(err)
	}
}

func testUnderlineHandlerAtScales(cx *TestAppContext, paint func(window *Window, cx *App)) {
	calls := 0
	window := cx.AddWindow(func(_ *Window, _ *Context[underlineHandlerTestView]) *underlineHandlerTestView {
		return &underlineHandlerTestView{paint: func(window *Window, cx *App) {
			calls++
			paint(window, cx)
		}}
	}).AnyWindowHandle
	defer func() {
		for _, step := range []func(*Window){
			func(w *Window) { w.renderedFrame.tabStops.clear(); w.nextFrame.tabStops.clear() },
			func(w *Window) {
				if root, ok := w.root.Take().Get(); ok {
					root.Release()
				}
				w.RemoveWindow()
			},
		} {
			_, _ = AppContextUpdateWindow(cx, window, func(_ AnyView, w *Window, _ *App) struct{} {
				step(w)
				return struct{}{}
			})
		}
	}()
	for _, scale := range []float32{1., 1.25, 1.5, 2., 3.} {
		cx.SimulateWindowScaleFactorChange(window, scale)
		lineTestUpdateWindow(cx, window, func(window *Window, cx *App) { window.Draw(cx).Clear(cx) })
	}
	if calls == 0 {
		panic("the test canvas was never painted")
	}
}

func underlineTestLine(text string, decorations []DecorationRun, zeroAdvance bool, window *Window) ShapedLine {
	run := DefaultTextRun()
	run.Len = len(text)
	run.Font = NewFont(".ZedMono")
	run.Color = Black()
	line := window.TextSystem().ShapeLine(SharedString(text), Px(16.), []TextRun{run}, None[Pixels]())
	line.decorationRuns = slices.Clone(decorations)
	layout := line.LineLayout
	runs := make([]ShapedRun, len(layout.Runs))
	for i, r := range layout.Runs {
		runs[i] = ShapedRun{FontID: r.FontID, Glyphs: slices.Clone(r.Glyphs)}
	}
	advance := Px(8.)
	if zeroAdvance {
		advance = Px(0.)
	}
	width := Px(0.)
	for i := range runs {
		for j := range runs[i].Glyphs {
			runs[i].Glyphs[j].Position.X = width
			width += advance
		}
	}
	line.LineLayout = &LineLayout{
		FontSize: layout.FontSize,
		Width:    width,
		Ascent:   layout.Ascent,
		Descent:  layout.Descent,
		Runs:     runs,
		Len:      layout.Len,
	}
	return line
}

func assertUnderlinePrimitivesEq(actual, expected []Underline) {
	if len(actual) != len(expected) {
		panic(fmt.Sprintf("underline count %d, want %d", len(actual), len(expected)))
	}
	for i := range actual {
		a, e := actual[i], expected[i]
		if a.Bounds != e.Bounds || a.ContentMask != e.ContentMask || a.Color != e.Color || a.Thickness != e.Thickness || a.Wavy != e.Wavy || a.Order != e.Order || a.Pad != e.Pad {
			panic(fmt.Sprintf("underline %d = %+v, want %+v", i, a, e))
		}
	}
}

func assertStrokesEq(actual, expected []underlineStroke) {
	if !slices.Equal(actual, expected) {
		panic(fmt.Sprintf("strokes = %+v\nwant %+v", actual, expected))
	}
}

func TestLine_UnderlineHandlerMatchesDefaultPaint(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		testUnderlineHandlerAtScales(cx, func(window *Window, cx *App) {
			firstStyle := UnderlineStyle{Thickness: Px(1.), Color: Some(NewHsla(0., 1., 0.5, 1.)), Wavy: true}
			lastStyle := firstStyle
			lastStyle.Color = Some(NewHsla(0.5, 1., 0.5, 1.))
			lastStyle.Wavy = false
			fallbackStyle := firstStyle
			fallbackStyle.Color = Some(Black())
			decoration := DecorationRun{Len: 1, Color: Black(), Underline: Some(firstStyle)}
			d2 := decoration
			d2.Len = 2
			d3 := decoration
			d3.Underline = None[UnderlineStyle]()
			d4 := decoration
			noColor := firstStyle
			noColor.Color = None[Hsla]()
			d4.Underline = Some(noColor)
			d5 := decoration
			d5.Underline = Some(lastStyle)
			line := underlineTestLine("aébcde", []DecorationRun{decoration, d2, d3, d4, d5, d5}, false, window)
			if line.Width() != Px(48.) {
				panic(fmt.Sprintf("width %v", line.Width()))
			}
			type alignCase struct {
				align      TextAlign
				alignWidth Option[Pixels]
				offset     float32
			}
			for _, originX := range []float32{-3.25, 0., 4.25} {
				for _, c := range []alignCase{
					{TextAlignLeft, None[Pixels](), 0.},
					{TextAlignCenter, None[Pixels](), 0.},
					{TextAlignRight, None[Pixels](), 0.},
					{TextAlignLeft, Some(Px(96.)), 0.},
					{TextAlignCenter, Some(Px(96.)), 24.},
					{TextAlignRight, Some(Px(96.)), 48.},
				} {
					origin := NewPoint(Px(originX), Px(12.25))
					lineHeight := Px(20.)
					scene := &window.nextFrame.scene
					scene.Clear()
					if err := line.Paint(origin, lineHeight, c.align, c.alignWidth, window, cx); err != nil {
						panic(err)
					}
					original := slices.Clone(scene.Underlines)
					scene.Clear()
					if err := line.LineLayout.Paint(origin, lineHeight, c.align, c.alignWidth, line.decorationRuns, window, cx); err != nil {
						panic(err)
					}
					assertUnderlinePrimitivesEq(scene.Underlines, original)

					scene.Clear()
					var strokes []underlineStroke
					if err := line.PaintWithUnderlineHandler(origin, lineHeight, c.align, c.alignWidth, window, cx, func(rng Range, origin Point[Pixels], width Pixels, style *UnderlineStyle, window *Window) {
						strokes = append(strokes, underlineStroke{rng, origin, width, *style})
						window.PaintUnderline(origin, width, *style)
					}); err != nil {
						panic(err)
					}
					start := Px(originX + c.offset)
					y := origin.Y + UnderlineYOffset(lineHeight, line.Ascent, line.Descent)
					assertStrokesEq(strokes, []underlineStroke{
						{Range{0, 3}, NewPoint(start, y), Px(16.), firstStyle},
						{Range{4, 5}, NewPoint(start+Px(24.), y), Px(8.), fallbackStyle},
						{Range{5, 7}, NewPoint(start+Px(32.), y), Px(16.), lastStyle},
					})
					assertUnderlinePrimitivesEq(scene.Underlines, original)

					scene.Clear()
					var captured []underlineStroke
					if err := line.PaintWithUnderlineHandler(origin, lineHeight, c.align, c.alignWidth, window, cx, func(rng Range, origin Point[Pixels], width Pixels, style *UnderlineStyle, _ *Window) {
						captured = append(captured, underlineStroke{rng, origin, width, *style})
					}); err != nil {
						panic(err)
					}
					assertStrokesEq(captured, strokes)
					if len(scene.Underlines) != 0 {
						panic("underlines painted without handler")
					}
				}
			}
			for _, text := range []string{"", "abc"} {
				line := underlineTestLine(text, nil, false, window)
				calls := 0
				if err := line.PaintWithUnderlineHandler(NewPoint(Px(4.), Px(10.)), Px(20.), TextAlignLeft, None[Pixels](), window, cx, func(Range, Point[Pixels], Pixels, *UnderlineStyle, *Window) { calls++ }); err != nil {
					panic(err)
				}
				if calls != 0 {
					panic(fmt.Sprintf("calls = %d", calls))
				}
			}
		})
	})
}

func TestLine_UnderlineHandlerReportsZeroAdvanceGeometry(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		testUnderlineHandlerAtScales(cx, func(window *Window, cx *App) {
			firstStyle := UnderlineStyle{Thickness: Px(1.), Color: Some(Black()), Wavy: true}
			lastStyle := firstStyle
			lastStyle.Wavy = false
			decoration := DecorationRun{Len: 1, Color: Black(), Underline: Some(firstStyle)}
			d2 := decoration
			d2.Underline = Some(lastStyle)
			line := underlineTestLine("ab", []DecorationRun{decoration, d2}, true, window)
			halfWidth := cx.TextSystem().BoundingBox(line.Runs[0].FontID, line.FontSize).Size.Width / 2.
			origin := NewPoint(Px(40.25), Px(10.25))
			lineHeight := Px(20.)
			y := origin.Y + UnderlineYOffset(lineHeight, line.Ascent, line.Descent)
			for _, c := range []struct {
				align  TextAlign
				offset float32
			}{{TextAlignLeft, 0.}, {TextAlignCenter, 16.}, {TextAlignRight, 32.}} {
				scene := &window.nextFrame.scene
				scene.Clear()
				if err := line.Paint(origin, lineHeight, c.align, Some(Px(32.)), window, cx); err != nil {
					panic(err)
				}
				original := slices.Clone(scene.Underlines)
				scene.Clear()
				var strokes []underlineStroke
				if err := line.PaintWithUnderlineHandler(origin, lineHeight, c.align, Some(Px(32.)), window, cx, func(rng Range, origin Point[Pixels], width Pixels, style *UnderlineStyle, window *Window) {
					strokes = append(strokes, underlineStroke{rng, origin, width, *style})
					window.PaintUnderline(origin, width, *style)
				}); err != nil {
					panic(err)
				}
				end := origin.X + Px(c.offset)
				start := NewPoint(end-halfWidth, y)
				width := end - start.X
				assertStrokesEq(strokes, []underlineStroke{
					{Range{0, 1}, start, width, firstStyle},
					{Range{1, 2}, start, width, lastStyle},
				})
				assertUnderlinePrimitivesEq(scene.Underlines, original)
			}
		})
	})
}

func TestLine_UnderlineHandlerMatchesWrappedPaint(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		testUnderlineHandlerAtScales(cx, func(window *Window, cx *App) {
			style := UnderlineStyle{Thickness: Px(1.), Color: Some(Black()), Wavy: true}
			for _, zeroAdvance := range []bool{false, true} {
				line := underlineTestLine("abcd", []DecorationRun{{Len: 4, Color: Black(), Underline: Some(style)}}, zeroAdvance, window)
				halfWidth := cx.TextSystem().BoundingBox(line.Runs[0].FontID, line.FontSize).Size.Width / 2.
				origin := NewPoint(Px(40.25), Px(10.25))
				lineHeight := Px(20.)
				y := origin.Y + UnderlineYOffset(lineHeight, line.Ascent, line.Descent)
				wrapped := WrappedLine{
					WrappedLineLayout: &WrappedLineLayout{
						UnwrappedLayout: line.LineLayout,
						WrapBoundaries:  []WrapBoundary{{RunIx: 0, GlyphIx: 2}},
						WrapWidth:       Some(Px(16.)),
					},
					Text:           line.Text,
					decorationRuns: line.decorationRuns,
				}
				scene := &window.nextFrame.scene
				scene.Clear()
				if err := wrapped.Paint(origin, lineHeight, TextAlignLeft, None[Bounds[Pixels]](), window, cx); err != nil {
					panic(err)
				}
				original := slices.Clone(scene.Underlines)
				scene.Clear()
				var strokes []underlineStroke
				if err := paintLine(origin, wrapped.UnwrappedLayout, lineHeight, TextAlignLeft, Some(Px(16.)), wrapped.decorationRuns, wrapped.WrapBoundaries, window, cx, func(rng Range, origin Point[Pixels], width Pixels, style *UnderlineStyle, window *Window) {
					strokes = append(strokes, underlineStroke{rng, origin, width, *style})
					window.PaintUnderline(origin, width, *style)
				}); err != nil {
					panic(err)
				}
				var start, width Pixels
				if zeroAdvance {
					start = origin.X - halfWidth
					width = origin.X - start
				} else {
					start, width = origin.X, Px(16.)
				}
				assertStrokesEq(strokes, []underlineStroke{
					{Range{0, 4}, NewPoint(start, y), width, style},
					{Range{0, 4}, NewPoint(start, y+lineHeight), width, style},
				})
				assertUnderlinePrimitivesEq(scene.Underlines, original)
			}
		})
	})
}

var sixGlyphs = [][2]float32{{0, 0.}, {1, 10.}, {2, 20.}, {3, 30.}, {4, 40.}, {5, 50.}}

func TestLine_SplitAtInvariants(t *testing.T) {
	line := makeShapedLine("abcdef", sixGlyphs, 60., nil)

	for i := 0; i <= 6; i++ {
		left, right := line.SplitAt(i)
		if left.Width()+right.Width() != line.Width() {
			t.Fatalf("widths must sum at split=%d", i)
		}
		if left.Len()+right.Len() != line.Len() {
			t.Fatalf("lengths must sum at split=%d", i)
		}
		if string(left.Text)+string(right.Text) != "abcdef" {
			t.Fatalf("text must concatenate at split=%d", i)
		}
		if left.FontSize != line.FontSize {
			t.Fatalf("font_size at split=%d", i)
		}
		if right.Ascent != line.Ascent {
			t.Fatalf("ascent at split=%d", i)
		}
		if right.Descent != line.Descent {
			t.Fatalf("descent at split=%d", i)
		}
	}

	left, right := line.SplitAt(0)
	if len(left.Runs) != 0 || len(right.Runs[0].Glyphs) != 6 {
		t.Fatal("split at 0")
	}

	left, right = line.SplitAt(6)
	if len(left.Runs[0].Glyphs) != 6 || len(right.Runs) != 0 {
		t.Fatal("split at end")
	}
}

func TestLine_SplitAtGlyphRebasing(t *testing.T) {
	glyph := func(x float32, index int) ShapedGlyph {
		return ShapedGlyph{ID: 0, Position: NewPoint(Px(x), Px(0.)), Index: index}
	}
	line := ShapedLine{
		LineLayout: &LineLayout{
			FontSize: Px(16.),
			Width:    Px(60.),
			Ascent:   Px(12.),
			Descent:  Px(4.),
			Runs: []ShapedRun{
				{FontID: 0, Glyphs: []ShapedGlyph{glyph(0., 0), glyph(10., 1), glyph(20., 2)}},
				{FontID: 1, Glyphs: []ShapedGlyph{glyph(30., 3), glyph(40., 4), glyph(50., 5)}},
			},
			Len: 6,
		},
		Text: "abcdef",
	}

	first, remainder := line.SplitAt(2)
	if first.Text != "ab" || len(first.Runs) != 1 || first.Runs[0].FontID != 0 {
		t.Fatalf("first = %+v", first)
	}

	if remainder.Text != "cdef" || len(remainder.Runs) != 2 {
		t.Fatalf("remainder = %+v", remainder)
	}
	if remainder.Runs[0].FontID != 0 || len(remainder.Runs[0].Glyphs) != 1 || remainder.Runs[0].Glyphs[0].Index != 0 || remainder.Runs[0].Glyphs[0].Position.X != Px(0.) {
		t.Fatalf("remainder run 0 = %+v", remainder.Runs[0])
	}
	if remainder.Runs[1].FontID != 1 || remainder.Runs[1].Glyphs[0].Index != 1 || remainder.Runs[1].Glyphs[0].Position.X != Px(10.) {
		t.Fatalf("remainder run 1 = %+v", remainder.Runs[1])
	}

	second, finalPart := remainder.SplitAt(2)
	if second.Text != "cd" || finalPart.Text != "ef" {
		t.Fatalf("second %q final %q", second.Text, finalPart.Text)
	}
	if finalPart.Runs[0].Glyphs[0].Index != 0 || finalPart.Runs[0].Glyphs[0].Position.X != Px(0.) {
		t.Fatalf("final = %+v", finalPart.Runs[0])
	}

	if first.Width()+second.Width()+finalPart.Width() != line.Width() {
		t.Fatal("widths must sum")
	}
}

func TestLine_SplitAtDecorations(t *testing.T) {
	red := Hsla{H: 0.0, S: 1.0, L: 0.5, A: 1.0}
	green := Hsla{H: 0.3, S: 1.0, L: 0.5, A: 1.0}
	blue := Hsla{H: 0.6, S: 1.0, L: 0.5, A: 1.0}

	line := makeShapedLine("abcdef", sixGlyphs, 60., []DecorationRun{
		{Len: 2, Color: red},
		{Len: 3, Color: green},
		{Len: 1, Color: blue},
	})

	left, right := line.SplitAt(3)

	if len(left.decorationRuns) != 2 || left.decorationRuns[0].Len != 2 || left.decorationRuns[0].Color != red || left.decorationRuns[1].Len != 1 || left.decorationRuns[1].Color != green {
		t.Fatalf("left = %+v", left.decorationRuns)
	}

	if len(right.decorationRuns) != 2 || right.decorationRuns[0].Len != 2 || right.decorationRuns[0].Color != green || right.decorationRuns[1].Len != 1 || right.decorationRuns[1].Color != blue {
		t.Fatalf("right = %+v", right.decorationRuns)
	}
}

func TestLine_CursorPreservesShapingMetadataAcrossRuns(t *testing.T) {
	line := ShapedLine{
		LineLayout: &LineLayout{
			FontSize: Px(16.),
			Width:    Px(50.),
			Ascent:   Px(12.),
			Descent:  Px(4.),
			Runs: []ShapedRun{
				{FontID: 3, Glyphs: []ShapedGlyph{
					{ID: 11, Position: NewPoint(Px(0.), Px(1.)), Index: 0, IsEmoji: true},
					{ID: 12, Position: NewPoint(Px(17.), Px(1.)), Index: 1},
					{ID: 13, Position: NewPoint(Px(19.), Px(-1.)), Index: 1},
				}},
				{FontID: 8, Glyphs: []ShapedGlyph{
					{ID: 21, Position: NewPoint(Px(25.), Px(1.)), Index: 5, IsEmoji: true},
					{ID: 22, Position: NewPoint(Px(41.), Px(1.)), Index: 7},
				}},
			},
			Len: 10,
		},
		Text: "a😀bcdef",
	}
	cursor := line.Cursor()
	first := cursor.TakeUntil(5)
	if first.Text != "a😀" || first.Runs[0].FontID != 3 || first.Runs[0].Glyphs[0].ID != 11 || !first.Runs[0].Glyphs[0].IsEmoji {
		t.Fatalf("first = %+v", first)
	}
	if first.Runs[0].Glyphs[1].Index != 1 || first.Runs[0].Glyphs[1].Position != NewPoint(Px(17.), Px(1.)) {
		t.Fatalf("first glyph 1 = %+v", first.Runs[0].Glyphs[1])
	}
	if len(first.Runs[0].Glyphs) != 3 || first.Runs[0].Glyphs[2].Index != 1 || first.Runs[0].Glyphs[2].Position != NewPoint(Px(19.), Px(-1.)) {
		t.Fatalf("first glyph 2 = %+v", first.Runs[0].Glyphs)
	}
	if cursor.XOffset() != Px(25.) {
		t.Fatalf("x offset %v", cursor.XOffset())
	}

	second := cursor.TakeUntil(7)
	if second.Text != "bc" || second.Runs[0].FontID != 8 || second.Runs[0].Glyphs[0].ID != 21 || second.Runs[0].Glyphs[0].Index != 0 || second.Runs[0].Glyphs[0].Position != NewPoint(Px(0.), Px(1.)) {
		t.Fatalf("second = %+v", second)
	}
	if cursor.XOffset() != Px(41.) {
		t.Fatalf("x offset %v", cursor.XOffset())
	}

	finalPart := cursor.TakeUntil(10)
	if finalPart.Text != "def" || finalPart.Runs[0].FontID != 8 || finalPart.Runs[0].Glyphs[0].ID != 22 || finalPart.Runs[0].Glyphs[0].Index != 0 || finalPart.Runs[0].Glyphs[0].Position != NewPoint(Px(0.), Px(1.)) {
		t.Fatalf("final = %+v", finalPart)
	}
}

func assertShapedLinesMatch(t *testing.T, actual, expected ShapedLine, checkDecorations bool) {
	t.Helper()
	if actual.Text != expected.Text || actual.Len() != expected.Len() || actual.Width() != expected.Width() || len(actual.Runs) != len(expected.Runs) {
		t.Fatalf("actual %+v, expected %+v", actual, expected)
	}
	for i := range actual.Runs {
		a, e := actual.Runs[i], expected.Runs[i]
		if a.FontID != e.FontID || len(a.Glyphs) != len(e.Glyphs) {
			t.Fatalf("run %d: %+v vs %+v", i, a, e)
		}
		for j := range a.Glyphs {
			if a.Glyphs[j].ID != e.Glyphs[j].ID || a.Glyphs[j].Index != e.Glyphs[j].Index || a.Glyphs[j].Position != e.Glyphs[j].Position {
				t.Fatalf("glyph %d/%d: %+v vs %+v", i, j, a.Glyphs[j], e.Glyphs[j])
			}
		}
	}
	if !checkDecorations {
		return
	}
	if len(actual.decorationRuns) != len(expected.decorationRuns) {
		t.Fatalf("decorations %+v vs %+v", actual.decorationRuns, expected.decorationRuns)
	}
	for i := range actual.decorationRuns {
		a, e := actual.decorationRuns[i], expected.decorationRuns[i]
		if a.Len != e.Len || a.Color != e.Color || a.BackgroundColor != e.BackgroundColor {
			t.Fatalf("decoration %d: %+v vs %+v", i, a, e)
		}
	}
}

func TestLine_CursorPreservesExistingVisualOrderSplitting(t *testing.T) {
	line := makeShapedLine("abc", [][2]float32{{0, 0.}, {2, 10.}, {1, 20.}}, 30., nil)
	cursor := line.Cursor()
	remainder := line
	previousBoundary := 0
	for _, boundary := range []int{0, 1, 2, 3} {
		expected, rest := remainder.SplitAt(boundary - previousBoundary)
		actual := cursor.TakeUntil(boundary)
		assertShapedLinesMatch(t, actual, expected, false)
		if cursor.XOffset() != line.XForIndex(boundary) {
			t.Fatalf("x offset %v at %d", cursor.XOffset(), boundary)
		}
		remainder = rest
		previousBoundary = boundary
	}
}

func TestLine_CursorPartitionsOneDecorationAcrossThreeChunks(t *testing.T) {
	line := makeShapedLine("abcdef", sixGlyphs, 60., []DecorationRun{{Len: 6, Color: Hsla{H: 0.2, S: 0.4, L: 0.6, A: 1.0}}})
	cursor := line.Cursor()
	for _, boundary := range []int{2, 4, 6} {
		if got := cursor.TakeUntil(boundary).decorationRuns[0].Len; got != 2 {
			t.Fatalf("len %d at %d", got, boundary)
		}
	}
}

func TestLine_CursorMatchesSuccessiveSplitsAtOrderedBoundaries(t *testing.T) {
	var decorations []DecorationRun
	for _, length := range []uint32{2, 0, 3, 1} {
		decorations = append(decorations, DecorationRun{
			Len:             length,
			Color:           Hsla{H: float32(length) / 10.0, S: 0.5, L: 0.5, A: 1.0},
			BackgroundColor: Some(Black()),
		})
	}
	line := makeShapedLine("abcdef", [][2]float32{{0, 5.}, {0, 5.}, {2, 15.}, {4, 25.}, {5, 35.}}, 45., decorations)
	for first := 0; first <= line.Len(); first++ {
		for second := first; second <= line.Len(); second++ {
			cursor := line.Cursor()
			remainder := line
			previousBoundary := 0
			totalWidth := Px(0.)
			text := ""
			for _, boundary := range []int{first, second, line.Len(), line.Len()} {
				expected, rest := remainder.SplitAt(boundary - previousBoundary)
				actual := cursor.TakeUntil(boundary)
				assertShapedLinesMatch(t, actual, expected, true)
				totalWidth += actual.Width()
				text += string(actual.Text)
				remainder = rest
				previousBoundary = boundary
			}
			if totalWidth != line.Width() || text != string(line.Text) {
				t.Fatalf("width %v text %q", totalWidth, text)
			}
		}
	}
}

func TestLine_CursorEmptyChunksAndRepeatedBoundaries(t *testing.T) {
	line := makeShapedLine("ab", [][2]float32{{0, 5.}, {1, 15.}}, 20., nil)
	cursor := line.Cursor()
	for _, c := range []struct {
		boundary int
		want     SharedString
	}{{0, ""}, {0, ""}, {1, "a"}, {2, "b"}, {2, ""}} {
		if got := cursor.TakeUntil(c.boundary).Text; got != c.want {
			t.Fatalf("take_until(%d) = %q, want %q", c.boundary, got, c.want)
		}
	}
	empty := makeShapedLine("", nil, 0., nil)
	piece := empty.Cursor().TakeUntil(0)
	if piece.Text != "" || len(piece.Runs) != 0 || piece.Width() != Px(0.) {
		t.Fatalf("piece = %+v", piece)
	}
}

func linePanics(f func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	f()
	return false
}

func TestLine_CursorRejectsInvalidBoundaries(t *testing.T) {
	line := makeShapedLine("é", [][2]float32{{0, 0.}}, 10., nil)
	cursor := line.Cursor()
	if !linePanics(func() { cursor.TakeUntil(1) }) {
		t.Fatal("expected panic inside a character")
	}
	cursor = line.Cursor()
	cursor.TakeUntil(2)
	if !linePanics(func() { cursor.TakeUntil(0) }) {
		t.Fatal("expected panic moving backwards")
	}
	cursor = line.Cursor()
	if !linePanics(func() { cursor.TakeUntil(3) }) {
		t.Fatal("expected panic past the end")
	}
}
