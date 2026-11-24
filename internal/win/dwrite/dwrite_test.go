//go:build windows

package dwrite

import (
	"testing"
	"unsafe"
)

func TestDwrite_StructLayout(t *testing.T) {
	var run DWRITE_COLOR_GLYPH_RUN1
	cases := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"DWRITE_GLYPH_RUN", unsafe.Sizeof(DWRITE_GLYPH_RUN{}), 48},
		{"DWRITE_GLYPH_RUN_DESCRIPTION", unsafe.Sizeof(DWRITE_GLYPH_RUN_DESCRIPTION{}), 40},
		{"DWRITE_COLOR_GLYPH_RUN", unsafe.Sizeof(DWRITE_COLOR_GLYPH_RUN{}), 88},
		{"DWRITE_COLOR_GLYPH_RUN1", unsafe.Sizeof(DWRITE_COLOR_GLYPH_RUN1{}), 96},
		{"DWRITE_COLOR_GLYPH_RUN1.GlyphImageFormat", unsafe.Offsetof(run.GlyphImageFormat), 88},
		{"DWRITE_COLOR_GLYPH_RUN.RunColor", unsafe.Offsetof(run.Base.RunColor), 64},
		{"DWRITE_FONT_METRICS", unsafe.Sizeof(DWRITE_FONT_METRICS{}), 20},
		{"DWRITE_FONT_METRICS1", unsafe.Sizeof(DWRITE_FONT_METRICS1{}), 48},
		{"DWRITE_GLYPH_METRICS", unsafe.Sizeof(DWRITE_GLYPH_METRICS{}), 28},
		{"DWRITE_LINE_METRICS", unsafe.Sizeof(DWRITE_LINE_METRICS{}), 24},
		{"DWRITE_TEXT_METRICS", unsafe.Sizeof(DWRITE_TEXT_METRICS{}), 36},
		{"DWRITE_HIT_TEST_METRICS", unsafe.Sizeof(DWRITE_HIT_TEST_METRICS{}), 36},
		{"DWRITE_MATRIX", unsafe.Sizeof(DWRITE_MATRIX{}), 24},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestDwrite_CreateFactory(t *testing.T) {
	factory, err := DWriteCreateFactory(DWRITE_FACTORY_TYPE_SHARED)
	if err != nil {
		t.Skipf("DirectWrite unavailable: %v", err)
	}
	defer factory.Release()
	collection, err := factory.GetSystemFontCollection(false, true)
	if err != nil || collection == nil {
		t.Fatalf("system font collection: %v", err)
	}
	defer collection.Release()
	if collection.GetFontFamilyCount() == 0 {
		t.Fatal("no system font families")
	}
}
