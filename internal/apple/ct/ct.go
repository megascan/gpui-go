//go:build darwin

// Package ct binds the CoreText fonts, font descriptors, collections, lines and runs the Apple text
// system uses, plus the CoreFoundation types package cf does not cover, on purego without cgo.
package ct

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/cg"
)

// CoreTextPath is the install path of CoreText.framework.
const CoreTextPath = "/System/Library/Frameworks/CoreText.framework/CoreText"

// Font is a CTFontRef.
type Font cf.TypeRef

// FontDescriptor is a CTFontDescriptorRef.
type FontDescriptor cf.TypeRef

// FontCollection is a CTFontCollectionRef.
type FontCollection cf.TypeRef

// Line is a CTLineRef.
type Line cf.TypeRef

// Run is a CTRunRef.
type Run cf.TypeRef

// SymbolicTraits is CTFontSymbolicTraits.
type SymbolicTraits uint32

// FontItalicTrait is kCTFontItalicTrait.
const FontItalicTrait SymbolicTraits = 1 << 0

// FontMonoSpaceTrait is kCTFontMonoSpaceTrait.
const FontMonoSpaceTrait SymbolicTraits = 1 << 10

// FontOrientation is CTFontOrientation.
type FontOrientation uint32

// FontDefaultOrientation is kCTFontDefaultOrientation.
const FontDefaultOrientation FontOrientation = 0

// TypographicBounds are a line's width and vertical extents.
type TypographicBounds struct {
	Width   float64
	Ascent  float64
	Descent float64
	Leading float64
}

var (
	loadOnce sync.Once
	loadErr  error

	ctFontCreateWithFontDescriptor                func(d FontDescriptor, size cg.Float, matrix unsafe.Pointer) Font
	ctFontCreateCopyWithAttributes                func(f Font, size cg.Float, matrix unsafe.Pointer, d FontDescriptor) Font
	ctFontCreateWithGraphicsFont                  func(g cg.Font, size cg.Float, matrix unsafe.Pointer, d FontDescriptor) Font
	ctFontCopyGraphicsFont                        func(f Font, d unsafe.Pointer) cg.Font
	ctFontCopyPostScriptName                      func(f Font) cf.String
	ctFontCopyDisplayName                         func(f Font) cf.String
	ctFontCopyFamilyName                          func(f Font) cf.String
	ctFontGetSymbolicTraits                       func(f Font) SymbolicTraits
	ctFontCopyTraits                              func(f Font) cf.Dictionary
	ctFontGetGlyphsForCharacters                  func(f Font, chars unsafe.Pointer, glyphs unsafe.Pointer, count cf.Index) bool
	ctFontGetBoundingRectsForGlyphs               func(f Font, o FontOrientation, glyphs unsafe.Pointer, rects unsafe.Pointer, count cf.Index) cg.Rect
	ctFontGetAdvancesForGlyphs                    func(f Font, o FontOrientation, glyphs unsafe.Pointer, advances unsafe.Pointer, count cf.Index) float64
	ctFontGetUnitsPerEm                           func(f Font) uint32
	ctFontGetSize                                 func(f Font) cg.Float
	ctFontGetBoundingBox                          func(f Font) cg.Rect
	ctFontGetAscent                               func(f Font) cg.Float
	ctFontGetDescent                              func(f Font) cg.Float
	ctFontGetLeading                              func(f Font) cg.Float
	ctFontGetUnderlinePosition                    func(f Font) cg.Float
	ctFontGetUnderlineThickness                   func(f Font) cg.Float
	ctFontGetCapHeight                            func(f Font) cg.Float
	ctFontGetXHeight                              func(f Font) cg.Float
	ctFontDrawGlyphs                              func(f Font, glyphs unsafe.Pointer, positions unsafe.Pointer, count uint, c cg.Context)
	ctFontCopyDefaultCascadeListForLanguages      func(f Font, langs cf.Array) cf.Array
	ctFontDescriptorCreateWithAttributes          func(attrs cf.Dictionary) FontDescriptor
	ctFontDescriptorCopyAttribute                 func(d FontDescriptor, attr cf.String) cf.TypeRef
	ctFontCollectionCreateFromAvailableFonts      func(options cf.Dictionary) FontCollection
	ctFontCollectionCreateWithFontDescriptors     func(descs cf.Array, options cf.Dictionary) FontCollection
	ctFontCollectionCreateMatchingFontDescriptors func(c FontCollection) cf.Array
	ctFontManagerCreateFontDescriptorFromData     func(data cf.Data) FontDescriptor
	ctLineCreateWithAttributedString              func(s MutableAttributedString) Line
	ctLineGetGlyphRuns                            func(l Line) cf.Array
	ctLineGetTypographicBounds                    func(l Line, ascent, descent, leading *float64) float64
	ctRunGetAttributes                            func(r Run) cf.Dictionary
	ctRunGetGlyphCount                            func(r Run) cf.Index
	ctRunGetGlyphs                                func(r Run, rg cf.Range, buf unsafe.Pointer)
	ctRunGetPositions                             func(r Run, rg cf.Range, buf unsafe.Pointer)
	ctRunGetStringIndices                         func(r Run, rg cf.Range, buf unsafe.Pointer)
	ctFontGetTypeID                               func() cf.TypeID

	// FontAttributeName is kCTFontAttributeName.
	FontAttributeName cf.String
	// FontFeatureSettingsAttribute is kCTFontFeatureSettingsAttribute.
	FontFeatureSettingsAttribute cf.String
	// FontCascadeListAttribute is kCTFontCascadeListAttribute.
	FontCascadeListAttribute cf.String
	// FontFamilyNameAttribute is kCTFontFamilyNameAttribute.
	FontFamilyNameAttribute cf.String
	// FontTraitsAttribute is kCTFontTraitsAttribute.
	FontTraitsAttribute cf.String
	// FontURLAttribute is kCTFontURLAttribute.
	FontURLAttribute cf.String
	// FontSymbolicTrait is kCTFontSymbolicTrait.
	FontSymbolicTrait cf.String
	// FontWeightTrait is kCTFontWeightTrait.
	FontWeightTrait cf.String
	// FontWidthTrait is kCTFontWidthTrait.
	FontWidthTrait cf.String
	// FontSlantTrait is kCTFontSlantTrait.
	FontSlantTrait cf.String
	// FontOpenTypeFeatureTag is kCTFontOpenTypeFeatureTag.
	FontOpenTypeFeatureTag cf.String
	// FontOpenTypeFeatureValue is kCTFontOpenTypeFeatureValue.
	FontOpenTypeFeatureValue cf.String
	// FontCollectionRemoveDuplicatesOption is kCTFontCollectionRemoveDuplicatesOption.
	FontCollectionRemoveDuplicatesOption cf.String
)

type binding struct {
	fptr any
	name string
}

func bind(h uintptr, lib string, bs []binding) error {
	for _, b := range bs {
		addr, err := purego.Dlsym(h, b.name)
		if err != nil {
			return fmt.Errorf("ct: %s: %w", lib, err)
		}
		purego.RegisterFunc(b.fptr, addr)
	}
	return nil
}

// Load opens CoreText and CoreFoundation and resolves their symbols. It is safe to call repeatedly.
func Load() error {
	loadOnce.Do(func() {
		if err := cf.Load(); err != nil {
			loadErr = err
			return
		}
		if err := cg.Load(); err != nil {
			loadErr = err
			return
		}
		cfh, err := purego.Dlopen(cf.CoreFoundationPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("ct: open %s: %w", cf.CoreFoundationPath, err)
			return
		}
		if err := bind(cfh, "CoreFoundation", []binding{
			{&cfNumberCreate, "CFNumberCreate"},
			{&cfNumberGetValue, "CFNumberGetValue"},
			{&cfArrayCreateMutable, "CFArrayCreateMutable"},
			{&cfArrayAppendValue, "CFArrayAppendValue"},
			{&cfAttributedStringCreateMutable, "CFAttributedStringCreateMutable"},
			{&cfAttributedStringReplaceString, "CFAttributedStringReplaceString"},
			{&cfAttributedStringGetLength, "CFAttributedStringGetLength"},
			{&cfAttributedStringSetAttribute, "CFAttributedStringSetAttribute"},
			{&cfLocaleCopyPreferredLanguages, "CFLocaleCopyPreferredLanguages"},
			{&cfPreferencesCopyAppValue, "CFPreferencesCopyAppValue"},
			{&cfURLCopyFileSystemPath, "CFURLCopyFileSystemPath"},
			{&cfURLGetTypeID, "CFURLGetTypeID"},
		}); err != nil {
			loadErr = err
			return
		}
		if typeArrayCallBacksAddr, err = purego.Dlsym(cfh, "kCFTypeArrayCallBacks"); err != nil {
			loadErr = fmt.Errorf("ct: %w", err)
			return
		}
		if preferencesCurrentApplication, err = cf.Constant(cfh, "kCFPreferencesCurrentApplication"); err != nil {
			loadErr = err
			return
		}
		h, err := purego.Dlopen(CoreTextPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("ct: open %s: %w", CoreTextPath, err)
			return
		}
		if err := bind(h, "CoreText", []binding{
			{&ctFontCreateWithFontDescriptor, "CTFontCreateWithFontDescriptor"},
			{&ctFontCreateCopyWithAttributes, "CTFontCreateCopyWithAttributes"},
			{&ctFontCreateWithGraphicsFont, "CTFontCreateWithGraphicsFont"},
			{&ctFontCopyGraphicsFont, "CTFontCopyGraphicsFont"},
			{&ctFontCopyPostScriptName, "CTFontCopyPostScriptName"},
			{&ctFontCopyDisplayName, "CTFontCopyDisplayName"},
			{&ctFontCopyFamilyName, "CTFontCopyFamilyName"},
			{&ctFontGetSymbolicTraits, "CTFontGetSymbolicTraits"},
			{&ctFontCopyTraits, "CTFontCopyTraits"},
			{&ctFontGetGlyphsForCharacters, "CTFontGetGlyphsForCharacters"},
			{&ctFontGetBoundingRectsForGlyphs, "CTFontGetBoundingRectsForGlyphs"},
			{&ctFontGetAdvancesForGlyphs, "CTFontGetAdvancesForGlyphs"},
			{&ctFontGetUnitsPerEm, "CTFontGetUnitsPerEm"},
			{&ctFontGetSize, "CTFontGetSize"},
			{&ctFontGetBoundingBox, "CTFontGetBoundingBox"},
			{&ctFontGetAscent, "CTFontGetAscent"},
			{&ctFontGetDescent, "CTFontGetDescent"},
			{&ctFontGetLeading, "CTFontGetLeading"},
			{&ctFontGetUnderlinePosition, "CTFontGetUnderlinePosition"},
			{&ctFontGetUnderlineThickness, "CTFontGetUnderlineThickness"},
			{&ctFontGetCapHeight, "CTFontGetCapHeight"},
			{&ctFontGetXHeight, "CTFontGetXHeight"},
			{&ctFontDrawGlyphs, "CTFontDrawGlyphs"},
			{&ctFontCopyDefaultCascadeListForLanguages, "CTFontCopyDefaultCascadeListForLanguages"},
			{&ctFontDescriptorCreateWithAttributes, "CTFontDescriptorCreateWithAttributes"},
			{&ctFontDescriptorCopyAttribute, "CTFontDescriptorCopyAttribute"},
			{&ctFontCollectionCreateFromAvailableFonts, "CTFontCollectionCreateFromAvailableFonts"},
			{&ctFontCollectionCreateWithFontDescriptors, "CTFontCollectionCreateWithFontDescriptors"},
			{&ctFontCollectionCreateMatchingFontDescriptors, "CTFontCollectionCreateMatchingFontDescriptors"},
			{&ctFontManagerCreateFontDescriptorFromData, "CTFontManagerCreateFontDescriptorFromData"},
			{&ctLineCreateWithAttributedString, "CTLineCreateWithAttributedString"},
			{&ctLineGetGlyphRuns, "CTLineGetGlyphRuns"},
			{&ctLineGetTypographicBounds, "CTLineGetTypographicBounds"},
			{&ctRunGetAttributes, "CTRunGetAttributes"},
			{&ctRunGetGlyphCount, "CTRunGetGlyphCount"},
			{&ctRunGetGlyphs, "CTRunGetGlyphs"},
			{&ctRunGetPositions, "CTRunGetPositions"},
			{&ctRunGetStringIndices, "CTRunGetStringIndices"},
			{&ctFontGetTypeID, "CTFontGetTypeID"},
		}); err != nil {
			loadErr = err
			return
		}
		for _, s := range []struct {
			dst  *cf.String
			name string
		}{
			{&FontAttributeName, "kCTFontAttributeName"},
			{&FontFeatureSettingsAttribute, "kCTFontFeatureSettingsAttribute"},
			{&FontCascadeListAttribute, "kCTFontCascadeListAttribute"},
			{&FontFamilyNameAttribute, "kCTFontFamilyNameAttribute"},
			{&FontTraitsAttribute, "kCTFontTraitsAttribute"},
			{&FontURLAttribute, "kCTFontURLAttribute"},
			{&FontSymbolicTrait, "kCTFontSymbolicTrait"},
			{&FontWeightTrait, "kCTFontWeightTrait"},
			{&FontWidthTrait, "kCTFontWidthTrait"},
			{&FontSlantTrait, "kCTFontSlantTrait"},
			{&FontOpenTypeFeatureTag, "kCTFontOpenTypeFeatureTag"},
			{&FontOpenTypeFeatureValue, "kCTFontOpenTypeFeatureValue"},
			{&FontCollectionRemoveDuplicatesOption, "kCTFontCollectionRemoveDuplicatesOption"},
		} {
			ref, err := cf.Constant(h, s.name)
			if err != nil {
				loadErr = err
				return
			}
			*s.dst = cf.String(ref)
		}
	})
	return loadErr
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// FontCreateWithFontDescriptor returns a new font for the descriptor at size points.
func FontCreateWithFontDescriptor(d FontDescriptor, size float64) *cf.Owned[Font] {
	mustLoad()
	return cf.Own(ctFontCreateWithFontDescriptor(d, size, nil))
}

// FontCreateWithGraphicsFont returns a new font for the CGFont at size points.
func FontCreateWithGraphicsFont(g cg.Font, size float64) *cf.Owned[Font] {
	mustLoad()
	return cf.Own(ctFontCreateWithGraphicsFont(g, size, nil, 0))
}

// CopyWithAttributes returns a copy of the font at size points (0 keeps the size) with the
// descriptor's attributes applied.
func (f Font) CopyWithAttributes(size float64, d FontDescriptor) *cf.Owned[Font] {
	mustLoad()
	return cf.Own(ctFontCreateCopyWithAttributes(f, size, nil, d))
}

// CloneWithFontSize returns a copy of the font at size points.
func (f Font) CloneWithFontSize(size float64) *cf.Owned[Font] {
	return f.CopyWithAttributes(size, 0)
}

// CopyGraphicsFont returns the font's CGFont; Release it.
func (f Font) CopyGraphicsFont() cg.Font {
	mustLoad()
	return ctFontCopyGraphicsFont(f, nil)
}

func ownedString(s cf.String) string {
	o := cf.Own(s)
	if o == nil {
		return ""
	}
	defer o.Release()
	return o.Ref().GoString()
}

// PostScriptName returns the font's PostScript name.
func (f Font) PostScriptName() string {
	mustLoad()
	return ownedString(ctFontCopyPostScriptName(f))
}

// DisplayName returns the font's display (full) name.
func (f Font) DisplayName() string {
	mustLoad()
	return ownedString(ctFontCopyDisplayName(f))
}

// FamilyName returns the font's family name.
func (f Font) FamilyName() string {
	mustLoad()
	return ownedString(ctFontCopyFamilyName(f))
}

// SymbolicTraits returns the font's symbolic traits.
func (f Font) SymbolicTraits() SymbolicTraits {
	mustLoad()
	return ctFontGetSymbolicTraits(f)
}

// CopyTraits returns the font's traits dictionary.
func (f Font) CopyTraits() *cf.Owned[cf.Dictionary] {
	mustLoad()
	return cf.Own(ctFontCopyTraits(f))
}

// GlyphsForCharacters maps UTF-16 code units to glyphs; it reports whether every unit mapped.
func (f Font) GlyphsForCharacters(chars []uint16, glyphs []cg.Glyph) bool {
	mustLoad()
	if len(glyphs) < len(chars) {
		panic("ct: GlyphsForCharacters needs a glyph per character")
	}
	if len(chars) == 0 {
		return true
	}
	return ctFontGetGlyphsForCharacters(f, unsafe.Pointer(&chars[0]), unsafe.Pointer(&glyphs[0]), len(chars))
}

// BoundingRectForGlyph returns the glyph's bounding rectangle at the font's size.
func (f Font) BoundingRectForGlyph(o FontOrientation, glyph cg.Glyph) cg.Rect {
	mustLoad()
	return ctFontGetBoundingRectsForGlyphs(f, o, unsafe.Pointer(&glyph), nil, 1)
}

// AdvanceForGlyph returns the glyph's advance at the font's size.
func (f Font) AdvanceForGlyph(o FontOrientation, glyph cg.Glyph) cg.Size {
	mustLoad()
	var advance cg.Size
	ctFontGetAdvancesForGlyphs(f, o, unsafe.Pointer(&glyph), unsafe.Pointer(&advance), 1)
	return advance
}

// UnitsPerEm returns the font's units per em.
func (f Font) UnitsPerEm() uint32 { mustLoad(); return ctFontGetUnitsPerEm(f) }

// PtSize returns the font's point size.
func (f Font) PtSize() float64 { mustLoad(); return ctFontGetSize(f) }

// BoundingBox returns the font's bounding box at its size.
func (f Font) BoundingBox() cg.Rect { mustLoad(); return ctFontGetBoundingBox(f) }

// Ascent returns the font's ascent at its size.
func (f Font) Ascent() float64 { mustLoad(); return ctFontGetAscent(f) }

// Descent returns the font's descent at its size.
func (f Font) Descent() float64 { mustLoad(); return ctFontGetDescent(f) }

// Leading returns the font's leading at its size.
func (f Font) Leading() float64 { mustLoad(); return ctFontGetLeading(f) }

// UnderlinePosition returns the font's underline position at its size.
func (f Font) UnderlinePosition() float64 { mustLoad(); return ctFontGetUnderlinePosition(f) }

// UnderlineThickness returns the font's underline thickness at its size.
func (f Font) UnderlineThickness() float64 { mustLoad(); return ctFontGetUnderlineThickness(f) }

// CapHeight returns the font's cap height at its size.
func (f Font) CapHeight() float64 { mustLoad(); return ctFontGetCapHeight(f) }

// XHeight returns the font's x-height at its size.
func (f Font) XHeight() float64 { mustLoad(); return ctFontGetXHeight(f) }

// DrawGlyphs draws the glyphs at the positions into the context.
func (f Font) DrawGlyphs(glyphs []cg.Glyph, positions []cg.Point, c cg.Context) {
	mustLoad()
	if len(positions) < len(glyphs) {
		panic("ct: DrawGlyphs needs a position per glyph")
	}
	if len(glyphs) == 0 {
		return
	}
	ctFontDrawGlyphs(f, unsafe.Pointer(&glyphs[0]), unsafe.Pointer(&positions[0]), uint(len(glyphs)), c)
}

// CascadeListForLanguages returns the system's default fallback descriptors for the font.
func (f Font) CascadeListForLanguages(langs cf.Array) *cf.Owned[cf.Array] {
	mustLoad()
	return cf.Own(ctFontCopyDefaultCascadeListForLanguages(f, langs))
}

// FontDescriptorCreateWithAttributes returns a new descriptor with the attributes.
func FontDescriptorCreateWithAttributes(attrs cf.Dictionary) *cf.Owned[FontDescriptor] {
	mustLoad()
	return cf.Own(ctFontDescriptorCreateWithAttributes(attrs))
}

// CopyAttribute returns the descriptor's attribute, or nil.
func (d FontDescriptor) CopyAttribute(attr cf.String) *cf.Owned[cf.TypeRef] {
	mustLoad()
	return cf.Own(ctFontDescriptorCopyAttribute(d, attr))
}

// FontPath returns the path of the descriptor's font file.
func (d FontDescriptor) FontPath() (string, bool) {
	v := d.CopyAttribute(FontURLAttribute)
	if v == nil {
		return "", false
	}
	defer v.Release()
	return URLFileSystemPath(v.Ref())
}

func removeDuplicatesOptions() *cf.Owned[cf.Dictionary] {
	one := NumberCreateInt64(1)
	defer one.Release()
	return cf.DictionaryCreate([]cf.TypeRef{cf.TypeRef(FontCollectionRemoveDuplicatesOption)}, []cf.TypeRef{one.TypeRef()})
}

// FontCollectionCreateForAllFamilies returns a collection of every available font.
func FontCollectionCreateForAllFamilies() *cf.Owned[FontCollection] {
	mustLoad()
	opts := removeDuplicatesOptions()
	defer opts.Release()
	return cf.Own(ctFontCollectionCreateFromAvailableFonts(opts.Ref()))
}

// FontCollectionCreateWithFontDescriptors returns a collection matching the descriptors.
func FontCollectionCreateWithFontDescriptors(descs cf.Array) *cf.Owned[FontCollection] {
	mustLoad()
	opts := removeDuplicatesOptions()
	defer opts.Release()
	return cf.Own(ctFontCollectionCreateWithFontDescriptors(descs, opts.Ref()))
}

// CreateMatchingFontDescriptors returns the collection's descriptors (Create rule), or nil.
func (c FontCollection) CreateMatchingFontDescriptors() *cf.Owned[cf.Array] {
	mustLoad()
	return cf.Own(ctFontCollectionCreateMatchingFontDescriptors(c))
}

// FontManagerCreateFontDescriptorFromData returns a descriptor for the font data, or nil.
func FontManagerCreateFontDescriptorFromData(data cf.Data) *cf.Owned[FontDescriptor] {
	mustLoad()
	return cf.Own(ctFontManagerCreateFontDescriptorFromData(data))
}

// LineCreateWithAttributedString returns a line set from the string.
func LineCreateWithAttributedString(s MutableAttributedString) *cf.Owned[Line] {
	mustLoad()
	return cf.Own(ctLineCreateWithAttributedString(s))
}

// GlyphRuns returns the line's runs, borrowed from the line.
func (l Line) GlyphRuns() []Run {
	mustLoad()
	a := ctLineGetGlyphRuns(l)
	n := a.Count()
	runs := make([]Run, n)
	for i := range n {
		runs[i] = Run(a.At(i))
	}
	return runs
}

// TypographicBounds returns the line's typographic bounds.
func (l Line) TypographicBounds() TypographicBounds {
	mustLoad()
	var b TypographicBounds
	b.Width = ctLineGetTypographicBounds(l, &b.Ascent, &b.Descent, &b.Leading)
	return b
}

// Attributes returns the run's attributes, borrowed from the run.
func (r Run) Attributes() cf.Dictionary {
	mustLoad()
	return ctRunGetAttributes(r)
}

// GlyphCount returns the number of glyphs in the run.
func (r Run) GlyphCount() int {
	mustLoad()
	return ctRunGetGlyphCount(r)
}

// Glyphs returns a copy of the run's glyphs.
func (r Run) Glyphs() []cg.Glyph {
	n := r.GlyphCount()
	out := make([]cg.Glyph, n)
	if n > 0 {
		ctRunGetGlyphs(r, cf.Range{}, unsafe.Pointer(&out[0]))
	}
	return out
}

// Positions returns a copy of the run's glyph positions.
func (r Run) Positions() []cg.Point {
	n := r.GlyphCount()
	out := make([]cg.Point, n)
	if n > 0 {
		ctRunGetPositions(r, cf.Range{}, unsafe.Pointer(&out[0]))
	}
	return out
}

// StringIndices returns a copy of the run's UTF-16 string indices.
func (r Run) StringIndices() []cf.Index {
	n := r.GlyphCount()
	out := make([]cf.Index, n)
	if n > 0 {
		ctRunGetStringIndices(r, cf.Range{}, unsafe.Pointer(&out[0]))
	}
	return out
}

// AsFont downcasts ref to a CTFont.
func AsFont(ref cf.TypeRef) (Font, bool) {
	mustLoad()
	if ref == 0 || cf.GetTypeID(ref) != ctFontGetTypeID() {
		return 0, false
	}
	return Font(ref), true
}
