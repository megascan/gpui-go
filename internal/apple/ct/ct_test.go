//go:build darwin

package ct

import (
	"testing"
	"unsafe"

	"github.com/megascan/gpui-go/internal/apple/cf"
)

func TestCt_Layout(t *testing.T) {
	if s := unsafe.Sizeof(cf.Index(0)); s != 8 {
		t.Fatalf("sizeof(CFIndex) = %d, want 8", s)
	}
	if FontItalicTrait != 1 || FontDefaultOrientation != 0 {
		t.Fatal("CoreText constants wrong")
	}
}

func TestCt_Symbols(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	for name, s := range map[string]cf.String{
		"kCTFontAttributeName":                    FontAttributeName,
		"kCTFontFeatureSettingsAttribute":         FontFeatureSettingsAttribute,
		"kCTFontCascadeListAttribute":             FontCascadeListAttribute,
		"kCTFontFamilyNameAttribute":              FontFamilyNameAttribute,
		"kCTFontTraitsAttribute":                  FontTraitsAttribute,
		"kCTFontURLAttribute":                     FontURLAttribute,
		"kCTFontSymbolicTrait":                    FontSymbolicTrait,
		"kCTFontWeightTrait":                      FontWeightTrait,
		"kCTFontWidthTrait":                       FontWidthTrait,
		"kCTFontSlantTrait":                       FontSlantTrait,
		"kCTFontOpenTypeFeatureTag":               FontOpenTypeFeatureTag,
		"kCTFontOpenTypeFeatureValue":             FontOpenTypeFeatureValue,
		"kCTFontCollectionRemoveDuplicatesOption": FontCollectionRemoveDuplicatesOption,
	} {
		if _, ok := cf.AsString(cf.TypeRef(s)); !ok {
			t.Errorf("%s is not a CFString", name)
		}
	}
}

func TestCt_NumbersAndAttributedString(t *testing.T) {
	n := NumberCreateInt32(-7)
	defer n.Release()
	if v, ok := n.Ref().Int64(); !ok || v != -7 {
		t.Fatalf("Int64 = %d, %v", v, ok)
	}
	f := NumberCreateFloat64(0.25)
	defer f.Release()
	if v, ok := f.Ref().Float64(); !ok || v != 0.25 {
		t.Fatalf("Float64 = %v, %v", v, ok)
	}
	s := AttributedStringCreateMutable()
	defer s.Release()
	text := cf.StringCreate("aé\U0001F600")
	defer text.Release()
	s.Ref().ReplaceString(cf.Range{}, text.Ref())
	if got := s.Ref().Len(); got != 4 {
		t.Fatalf("UTF-16 length = %d, want 4", got)
	}
}

func TestCt_FontAndLine(t *testing.T) {
	name := cf.StringCreate("Helvetica")
	defer name.Release()
	attrs := cf.DictionaryCreate([]cf.TypeRef{cf.TypeRef(FontFamilyNameAttribute)}, []cf.TypeRef{name.TypeRef()})
	defer attrs.Release()
	desc := FontDescriptorCreateWithAttributes(attrs.Ref())
	defer desc.Release()
	font := FontCreateWithFontDescriptor(desc.Ref(), 16)
	defer font.Release()
	if font.Ref().PtSize() != 16 || font.Ref().UnitsPerEm() == 0 {
		t.Fatal("font metrics wrong")
	}
	glyphs := make([]uint16, 2)
	if !font.Ref().GlyphsForCharacters([]uint16{'m', 0}, glyphs) && glyphs[0] == 0 {
		t.Fatal("no glyph for m")
	}
	s := AttributedStringCreateMutable()
	defer s.Release()
	text := cf.StringCreate("mm")
	defer text.Release()
	s.Ref().ReplaceString(cf.Range{}, text.Ref())
	s.Ref().SetAttribute(cf.Range{Location: 0, Length: 2}, FontAttributeName, font.TypeRef())
	line := LineCreateWithAttributedString(s.Ref())
	defer line.Release()
	runs := line.Ref().GlyphRuns()
	if len(runs) != 1 || runs[0].GlyphCount() != 2 || len(runs[0].StringIndices()) != 2 {
		t.Fatalf("runs = %d", len(runs))
	}
	if line.Ref().TypographicBounds().Width <= 0 {
		t.Fatal("zero line width")
	}
}
