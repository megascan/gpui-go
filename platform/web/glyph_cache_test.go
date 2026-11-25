//go:build js && wasm

package web

import (
	"math"
	"strings"
	"testing"
	"unsafe"

	"github.com/megascan/gpui-go"
)

func TestGlyphCache_WarmLookupBorrowsTextAndPreservesStorage(t *testing.T) {
	var cache glyphCache
	fontID := gpui.FontID(1)
	glyphID, err := cache.getOrInsert(fontID, "\U0001f469‍\U0001f4bb", true)
	if err != nil {
		t.Fatal(err)
	}
	glyphsPointer := unsafe.SliceData(cache.glyphs)
	glyphsCapacity := cap(cache.glyphs)
	glyph, ok := cache.glyph(fontID, glyphID)
	if !ok {
		t.Fatal("missing glyph")
	}
	textPointer := unsafe.StringData(glyph.text)
	surroundingText := strings.Clone("\U0001f469‍\U0001f4bb suffix")
	borrowedText, ok := strings.CutSuffix(surroundingText, " suffix")
	if !ok {
		t.Fatal("missing suffix")
	}

	for range 100 {
		if id, ok := cache.idFor(fontID, borrowedText, true); !ok || id != glyphID {
			t.Fatalf("idFor = %v, %v", id, ok)
		}
		id, err := cache.getOrInsert(fontID, borrowedText, true)
		if err != nil || id != glyphID {
			t.Fatalf("getOrInsert = %v, %v", id, err)
		}
	}

	if len(cache.glyphs) != 1 {
		t.Fatalf("len = %d", len(cache.glyphs))
	}
	if unsafe.SliceData(cache.glyphs) != glyphsPointer || cap(cache.glyphs) != glyphsCapacity {
		t.Fatal("glyph storage moved")
	}
	glyph, ok = cache.glyph(fontID, glyphID)
	if !ok {
		t.Fatal("missing glyph")
	}
	if unsafe.StringData(glyph.text) != textPointer {
		t.Fatal("glyph text moved")
	}
}

func TestGlyphCache_FontAndPresentationHaveDistinctIDs(t *testing.T) {
	var cache glyphCache
	firstFont := gpui.FontID(1)
	secondFont := gpui.FontID(2)
	must := func(id gpui.GlyphID, err error) gpui.GlyphID {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	text := must(cache.getOrInsert(firstFont, "❤", false))
	color := must(cache.getOrInsert(firstFont, "❤", true))
	otherFont := must(cache.getOrInsert(secondFont, "❤", false))

	if text == color || text == otherFont || color == otherFont {
		t.Fatalf("ids not distinct: %v %v %v", text, color, otherFont)
	}
	check := func(fontID gpui.FontID, color bool, want gpui.GlyphID) {
		t.Helper()
		if id, ok := cache.idFor(fontID, "❤", color); !ok || id != want {
			t.Fatalf("idFor(%v, %v) = %v, %v", fontID, color, id, ok)
		}
	}
	check(firstFont, false, text)
	check(firstFont, true, color)
	check(secondFont, false, otherFont)
	if _, ok := cache.glyph(secondFont, text); ok {
		t.Fatal("glyph for wrong font")
	}
	if _, ok := cache.glyph(firstFont, gpui.GlyphID(math.MaxUint32)); ok {
		t.Fatal("glyph out of range")
	}
	if g, ok := cache.glyph(firstFont, text); !ok || g.color {
		t.Fatal("text glyph color")
	}
	if g, ok := cache.glyph(firstFont, color); !ok || !g.color {
		t.Fatal("color glyph color")
	}
}

func TestGlyphCache_InsertionRechecksAPreviousMiss(t *testing.T) {
	var cache glyphCache
	fontID := gpui.FontID(1)
	if _, ok := cache.idFor(fontID, "中", false); ok {
		t.Fatal("unexpected hit")
	}
	inserted, err := cache.getOrInsert(fontID, "中", false)
	if err != nil {
		t.Fatal(err)
	}
	again, err := cache.getOrInsert(fontID, "中", false)
	if err != nil || again != inserted {
		t.Fatalf("again = %v, %v", again, err)
	}
	if len(cache.glyphs) != 1 {
		t.Fatalf("len = %d", len(cache.glyphs))
	}
}
