//go:build js && wasm

package main

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"sync"
	"syscall/js"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

//go:embed fonts/Lilex-Regular.ttf
var LilexRegular []byte

type Reports struct {
	mu      sync.Mutex
	missing []gpui.MissingGlyph
}

func (r *Reports) Report(missing []gpui.MissingGlyph) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.missing = append(r.missing, missing...)
}

func (r *Reports) take() []gpui.MissingGlyph {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.missing)
}

func (r *Reports) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.missing = nil
}

func testMissingGlyphNotifications() error {
	textSystem := platform.CurrentPlatform(false).TextSystem()
	if err := textSystem.AddFonts([][]byte{LilexRegular}); err != nil {
		return err
	}
	fontID, err := textSystem.FontID(gpui.NewFont("Lilex"))
	if err != nil {
		return err
	}
	sink := &Reports{}
	textSystem.SetMissingGlyphSink(sink)

	text := "A界😀👨‍👩‍👧‍👦が"
	runs := []gpui.FontRun{{Len: len(text), FontID: fontID}}
	layout := textSystem.LayoutLine(text, gpui.Px(24), runs)
	if !slices.ContainsFunc(layout.Runs, func(run gpui.ShapedRun) bool {
		return slices.ContainsFunc(run.Glyphs, func(glyph gpui.ShapedGlyph) bool { return glyph.IsEmoji })
	}) {
		return fmt.Errorf("expected an emoji glyph")
	}
	missing := sink.take()
	var graphemes []string
	for _, report := range missing {
		graphemes = append(graphemes, report.Grapheme())
	}
	if !slices.Equal(graphemes, []string{"界", "が"}) {
		return fmt.Errorf("missing graphemes = %q", graphemes)
	}
	for _, report := range missing {
		if report.FontClass() != gpui.FallbackFontClassMonospace {
			return fmt.Errorf("missing glyph %q has font class %v", report.Grapheme(), report.FontClass())
		}
	}

	for _, graphemeSet := range [][]string{
		{"🕸", "🕸", "🕸"},
		{"🕸", "🕸️", "🕸︎", "🕸"},
	} {
		sink.clear()
		text := strings.Join(graphemeSet, "")
		layout := textSystem.LayoutLine(text, gpui.Px(24), []gpui.FontRun{{Len: len(text), FontID: fontID}})
		type resolved struct {
			fontID gpui.FontID
			glyph  gpui.ShapedGlyph
		}
		var glyphs []resolved
		for _, run := range layout.Runs {
			for _, glyph := range run.Glyphs {
				glyphs = append(glyphs, resolved{fontID: run.FontID, glyph: glyph})
			}
		}
		if len(glyphs) != len(graphemeSet) {
			return fmt.Errorf("glyph count = %d, want %d", len(glyphs), len(graphemeSet))
		}
		sourceIndex := 0
		for i, grapheme := range graphemeSet {
			if glyphs[i].fontID == fontID {
				return fmt.Errorf("missing Lilex glyph must fall back")
			}
			if glyphs[i].glyph.Index != sourceIndex {
				return fmt.Errorf("glyph index = %d, want %d", glyphs[i].glyph.Index, sourceIndex)
			}
			if glyphs[i].glyph.IsEmoji != strings.HasSuffix(grapheme, "️") {
				return fmt.Errorf("glyph %q emoji flag mismatch", grapheme)
			}
			sourceIndex += len(grapheme)
		}
		if sourceIndex != len(text) {
			return fmt.Errorf("source length = %d, want %d", sourceIndex, len(text))
		}
		if len(sink.take()) != 0 {
			return fmt.Errorf("every spider-web glyph must resolve, not just the first cached raster")
		}
	}

	sink.clear()
	bundled := "Hello ©"
	layout = textSystem.LayoutLine(bundled, gpui.Px(24), []gpui.FontRun{{Len: len(bundled), FontID: fontID}})
	for _, run := range layout.Runs {
		if run.FontID != fontID {
			return fmt.Errorf("retain existing bundled text glyphs")
		}
		for _, glyph := range run.Glyphs {
			if glyph.IsEmoji {
				return fmt.Errorf("retain existing bundled text glyphs: emoji flag set")
			}
		}
	}
	if len(sink.take()) != 0 {
		return fmt.Errorf("bundled text must not report missing glyphs")
	}

	sink.clear()
	textSystem.SetMissingGlyphSink(nil)
	textSystem.LayoutLine(text, gpui.Px(24), runs)
	if len(sink.take()) != 0 {
		return fmt.Errorf("a cleared sink must not receive reports")
	}
	return nil
}

func registerFontFallbackTest() {
	exports := js.Global().Get("Object").New()
	exports.Set("test_missing_glyph_notifications", js.FuncOf(func(js.Value, []js.Value) any {
		return testResult(testMissingGlyphNotifications())
	}))
	js.Global().Set("gpuiWebTests", exports)
}

func testResult(err error) js.Value {
	result := js.Global().Get("Object").New()
	if err != nil {
		result.Set("error", err.Error())
	}
	return result
}
