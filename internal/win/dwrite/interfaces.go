//go:build windows

package dwrite

import (
	"math"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/megascan/gpui-go/internal/win/com"
)

type fnKey struct {
	addr uintptr
	typ  reflect.Type
}

var floatFns sync.Map

var emptyText uint16

func floatFn[F any](this unsafe.Pointer, slot int) F {
	key := fnKey{addr: com.Method(this, slot), typ: reflect.TypeFor[F]()}
	if v, ok := floatFns.Load(key); ok {
		return v.(F)
	}
	var f F
	purego.RegisterFunc(&f, key.addr)
	v, _ := floatFns.LoadOrStore(key, f)
	return v.(F)
}

func hr(r uintptr) error {
	return com.HRESULTFromUintptr(r).Err()
}

func hr32(r int32) error {
	return com.HRESULT(r).Err()
}

func slicePtr[T any](s []T) unsafe.Pointer {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Pointer(&s[0])
}

const (
	factorySlots  = com.IUnknownSlots + 21
	factory1Slots = factorySlots + 2
	factory2Slots = factory1Slots + 5
	factory3Slots = factory2Slots + 9
	factory4Slots = factory3Slots + 3

	fontFaceSlots  = com.IUnknownSlots + 15
	fontFace1Slots = fontFaceSlots + 12
	fontFace2Slots = fontFace1Slots + 5

	textFormatSlots = com.IUnknownSlots + 25
)

// IDWriteFactory5 is the DirectWrite factory.
type IDWriteFactory5 struct{ com.IUnknown }

// CreateRenderingParams creates rendering params for the primary monitor.
func (f *IDWriteFactory5) CreateRenderingParams() (*IDWriteRenderingParams, error) {
	var out *IDWriteRenderingParams
	err := com.CallHR(f.Ptr(), com.IUnknownSlots+7, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// RegisterFontFileLoader registers a font file loader.
func (f *IDWriteFactory5) RegisterFontFileLoader(loader *IDWriteInMemoryFontFileLoader) error {
	return com.CallHR(f.Ptr(), com.IUnknownSlots+10, uintptr(loader.Ptr()))
}

// UnregisterFontFileLoader unregisters a font file loader.
func (f *IDWriteFactory5) UnregisterFontFileLoader(loader *IDWriteInMemoryFontFileLoader) error {
	return com.CallHR(f.Ptr(), com.IUnknownSlots+11, uintptr(loader.Ptr()))
}

// CreateTextFormat creates a text format.
func (f *IDWriteFactory5) CreateTextFormat(familyName []uint16, collection *IDWriteFontCollection1, weight, style, stretch int32, fontSize float32, locale []uint16) (*IDWriteTextFormat, error) {
	var out *IDWriteTextFormat
	call := floatFn[func(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer, int32, int32, int32, float32, unsafe.Pointer, unsafe.Pointer) int32](f.Ptr(), com.IUnknownSlots+12)
	r := call(f.Ptr(), slicePtr(familyName), unsafe.Pointer(collection), weight, style, stretch, fontSize, slicePtr(locale), unsafe.Pointer(&out))
	runtime.KeepAlive(familyName)
	runtime.KeepAlive(locale)
	return out, hr32(r)
}

// CreateTypography creates an empty typography.
func (f *IDWriteFactory5) CreateTypography() (*IDWriteTypography, error) {
	var out *IDWriteTypography
	err := com.CallHR(f.Ptr(), com.IUnknownSlots+13, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateTextLayout lays out text in a format.
func (f *IDWriteFactory5) CreateTextLayout(text []uint16, format *IDWriteTextFormat, maxWidth, maxHeight float32) (*IDWriteTextLayout, error) {
	var out *IDWriteTextLayout
	call := floatFn[func(unsafe.Pointer, unsafe.Pointer, uint32, unsafe.Pointer, float32, float32, unsafe.Pointer) int32](f.Ptr(), com.IUnknownSlots+15)
	textPtr := slicePtr(text)
	if textPtr == nil {
		textPtr = unsafe.Pointer(&emptyText)
	}
	r := call(f.Ptr(), textPtr, uint32(len(text)), format.Ptr(), maxWidth, maxHeight, unsafe.Pointer(&out))
	runtime.KeepAlive(text)
	return out, hr32(r)
}

// GetSystemFontFallback returns the system font fallback.
func (f *IDWriteFactory5) GetSystemFontFallback() (*IDWriteFontFallback, error) {
	var out *IDWriteFontFallback
	err := com.CallHR(f.Ptr(), factory1Slots, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateFontFallbackBuilder creates a font fallback builder.
func (f *IDWriteFactory5) CreateFontFallbackBuilder() (*IDWriteFontFallbackBuilder, error) {
	var out *IDWriteFontFallbackBuilder
	err := com.CallHR(f.Ptr(), factory1Slots+1, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateGlyphRunAnalysis analyses a glyph run for rasterisation (IDWriteFactory3).
func (f *IDWriteFactory5) CreateGlyphRunAnalysis(glyphRun *DWRITE_GLYPH_RUN, transform *DWRITE_MATRIX, renderingMode, measuringMode, gridFitMode, antialiasMode int32, baselineOriginX, baselineOriginY float32) (*IDWriteGlyphRunAnalysis, error) {
	var out *IDWriteGlyphRunAnalysis
	call := floatFn[func(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer, int32, int32, int32, int32, float32, float32, unsafe.Pointer) int32](f.Ptr(), factory2Slots)
	r := call(f.Ptr(), unsafe.Pointer(glyphRun), unsafe.Pointer(transform), renderingMode, measuringMode, gridFitMode, antialiasMode, baselineOriginX, baselineOriginY, unsafe.Pointer(&out))
	return out, hr32(r)
}

// CreateFontCollectionFromFontSet creates a font collection from a font set.
func (f *IDWriteFactory5) CreateFontCollectionFromFontSet(set *IDWriteFontSet) (*IDWriteFontCollection1, error) {
	var out *IDWriteFontCollection1
	err := com.CallHR(f.Ptr(), factory2Slots+6, uintptr(set.Ptr()), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// GetSystemFontCollection returns the system font collection (IDWriteFactory3); it may be nil.
func (f *IDWriteFactory5) GetSystemFontCollection(includeDownloadableFonts bool, checkForUpdates bool) (*IDWriteFontCollection1, error) {
	var out *IDWriteFontCollection1
	err := com.CallHR(f.Ptr(), factory2Slots+7, uintptr(BoolOf(includeDownloadableFonts)), uintptr(unsafe.Pointer(&out)), uintptr(BoolOf(checkForUpdates)))
	return out, err
}

// TranslateColorGlyphRun splits a colour glyph run into its layers (IDWriteFactory4).
func (f *IDWriteFactory5) TranslateColorGlyphRun(baselineOrigin Vector2, glyphRun *DWRITE_GLYPH_RUN, glyphRunDescription *DWRITE_GLYPH_RUN_DESCRIPTION, desiredGlyphImageFormats, measuringMode int32, worldAndDpiTransform *DWRITE_MATRIX, colorPaletteIndex uint32) (*IDWriteColorGlyphRunEnumerator1, error) {
	var out *IDWriteColorGlyphRunEnumerator1
	var r int32
	if runtime.GOARCH == "arm64" {
		call := floatFn[func(unsafe.Pointer, float32, float32, unsafe.Pointer, unsafe.Pointer, int32, int32, unsafe.Pointer, uint32, unsafe.Pointer) int32](f.Ptr(), factory3Slots)
		r = call(f.Ptr(), baselineOrigin.X, baselineOrigin.Y, unsafe.Pointer(glyphRun), unsafe.Pointer(glyphRunDescription), desiredGlyphImageFormats, measuringMode, unsafe.Pointer(worldAndDpiTransform), colorPaletteIndex, unsafe.Pointer(&out))
	} else {
		origin := uint64(math.Float32bits(baselineOrigin.X)) | uint64(math.Float32bits(baselineOrigin.Y))<<32
		call := floatFn[func(unsafe.Pointer, uint64, unsafe.Pointer, unsafe.Pointer, int32, int32, unsafe.Pointer, uint32, unsafe.Pointer) int32](f.Ptr(), factory3Slots)
		r = call(f.Ptr(), origin, unsafe.Pointer(glyphRun), unsafe.Pointer(glyphRunDescription), desiredGlyphImageFormats, measuringMode, unsafe.Pointer(worldAndDpiTransform), colorPaletteIndex, unsafe.Pointer(&out))
	}
	return out, hr32(r)
}

// CreateFontSetBuilder creates a font set builder (IDWriteFactory5).
func (f *IDWriteFactory5) CreateFontSetBuilder() (*IDWriteFontSetBuilder1, error) {
	var out *IDWriteFontSetBuilder1
	err := com.CallHR(f.Ptr(), factory4Slots, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateInMemoryFontFileLoader creates an in-memory font file loader.
func (f *IDWriteFactory5) CreateInMemoryFontFileLoader() (*IDWriteInMemoryFontFileLoader, error) {
	var out *IDWriteInMemoryFontFileLoader
	err := com.CallHR(f.Ptr(), factory4Slots+1, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDWriteFontFile is a font file reference.
type IDWriteFontFile struct{ com.IUnknown }

// IDWriteFontFallback maps characters to fallback fonts.
type IDWriteFontFallback struct{ com.IUnknown }

// IDWriteInMemoryFontFileLoader loads fonts from memory.
type IDWriteInMemoryFontFileLoader struct{ com.IUnknown }

// CreateInMemoryFontFileReference creates a font file from data; with a nil owner the data is copied.
func (l *IDWriteInMemoryFontFileLoader) CreateInMemoryFontFileReference(factory *IDWriteFactory5, data []byte, owner *com.IUnknown) (*IDWriteFontFile, error) {
	var out *IDWriteFontFile
	err := com.CallHR(l.Ptr(), com.IUnknownSlots+1, uintptr(factory.Ptr()), uintptr(slicePtr(data)), uintptr(len(data)), uintptr(unsafe.Pointer(owner)), uintptr(unsafe.Pointer(&out)))
	runtime.KeepAlive(data)
	return out, err
}

// IDWriteFontSetBuilder1 builds font sets.
type IDWriteFontSetBuilder1 struct{ com.IUnknown }

// CreateFontSet creates a font set from the added fonts.
func (b *IDWriteFontSetBuilder1) CreateFontSet() (*IDWriteFontSet, error) {
	var out *IDWriteFontSet
	err := com.CallHR(b.Ptr(), com.IUnknownSlots+3, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// AddFontFile adds every font in a file.
func (b *IDWriteFontSetBuilder1) AddFontFile(file *IDWriteFontFile) error {
	return com.CallHR(b.Ptr(), com.IUnknownSlots+4, uintptr(file.Ptr()))
}

// IDWriteFontCollection1 is a font collection backed by a font set.
type IDWriteFontCollection1 struct{ com.IUnknown }

// GetFontFamilyCount returns the number of families.
func (c *IDWriteFontCollection1) GetFontFamilyCount() uint32 {
	return uint32(com.Call(c.Ptr(), com.IUnknownSlots))
}

// GetFontSet returns the collection's font set.
func (c *IDWriteFontCollection1) GetFontSet() (*IDWriteFontSet, error) {
	var out *IDWriteFontSet
	err := com.CallHR(c.Ptr(), com.IUnknownSlots+4, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// GetFontFamily returns a family by index (IDWriteFontCollection1).
func (c *IDWriteFontCollection1) GetFontFamily(index uint32) (*IDWriteFontFamily, error) {
	var out *IDWriteFontFamily
	err := com.CallHR(c.Ptr(), com.IUnknownSlots+5, uintptr(index), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDWriteFontFamily is a font family.
type IDWriteFontFamily struct{ com.IUnknown }

// GetFamilyNames returns the family's localized names.
func (f *IDWriteFontFamily) GetFamilyNames() (*IDWriteLocalizedStrings, error) {
	var out *IDWriteLocalizedStrings
	err := com.CallHR(f.Ptr(), com.IUnknownSlots+3, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDWriteFontSet is a set of font face references.
type IDWriteFontSet struct{ com.IUnknown }

// GetFontCount returns the number of fonts.
func (s *IDWriteFontSet) GetFontCount() uint32 {
	return uint32(com.Call(s.Ptr(), com.IUnknownSlots))
}

// GetFontFaceReference returns a font by index.
func (s *IDWriteFontSet) GetFontFaceReference(index uint32) (*IDWriteFontFaceReference, error) {
	var out *IDWriteFontFaceReference
	err := com.CallHR(s.Ptr(), com.IUnknownSlots+1, uintptr(index), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// GetMatchingFonts returns the fonts matching a family, weight, stretch and style.
func (s *IDWriteFontSet) GetMatchingFonts(familyName []uint16, weight, stretch, style int32) (*IDWriteFontSet, error) {
	var out *IDWriteFontSet
	err := com.CallHR(s.Ptr(), com.IUnknownSlots+8, uintptr(slicePtr(familyName)), uintptr(weight), uintptr(stretch), uintptr(style), uintptr(unsafe.Pointer(&out)))
	runtime.KeepAlive(familyName)
	return out, err
}

// IDWriteFontFaceReference references a font face.
type IDWriteFontFaceReference struct{ com.IUnknown }

// CreateFontFace creates the referenced font face.
func (r *IDWriteFontFaceReference) CreateFontFace() (*IDWriteFontFace3, error) {
	var out *IDWriteFontFace3
	err := com.CallHR(r.Ptr(), com.IUnknownSlots, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDWriteFontFace is the base font face interface.
type IDWriteFontFace struct{ com.IUnknown }

// IDWriteFontFace3 is a font face.
type IDWriteFontFace3 struct{ com.IUnknown }

// AsFontFace returns f's IDWriteFontFace view.
func (f *IDWriteFontFace3) AsFontFace() *IDWriteFontFace {
	return com.As[IDWriteFontFace](f)
}

// GetDesignGlyphMetrics returns glyph metrics in design units.
func (f *IDWriteFontFace3) GetDesignGlyphMetrics(glyphIndices []uint16, metrics []DWRITE_GLYPH_METRICS, isSideways bool) error {
	err := com.CallHR(f.Ptr(), com.IUnknownSlots+7, uintptr(slicePtr(glyphIndices)), uintptr(len(glyphIndices)), uintptr(slicePtr(metrics)), uintptr(BoolOf(isSideways)))
	runtime.KeepAlive(glyphIndices)
	runtime.KeepAlive(metrics)
	return err
}

// GetGlyphIndices maps code points to glyph indices.
func (f *IDWriteFontFace3) GetGlyphIndices(codePoints []uint32, glyphIndices []uint16) error {
	err := com.CallHR(f.Ptr(), com.IUnknownSlots+8, uintptr(slicePtr(codePoints)), uintptr(len(codePoints)), uintptr(slicePtr(glyphIndices)))
	runtime.KeepAlive(codePoints)
	runtime.KeepAlive(glyphIndices)
	return err
}

// GetMetrics returns the face's metrics (IDWriteFontFace1).
func (f *IDWriteFontFace3) GetMetrics(metrics *DWRITE_FONT_METRICS1) {
	com.Call(f.Ptr(), fontFaceSlots, uintptr(unsafe.Pointer(metrics)))
}

// GetUnicodeRanges fills ranges and returns the face's total range count; the error is
// E_NOT_SUFFICIENT_BUFFER when ranges is too short.
func (f *IDWriteFontFace3) GetUnicodeRanges(ranges []DWRITE_UNICODE_RANGE) (uint32, error) {
	var count uint32
	err := com.CallHR(f.Ptr(), fontFaceSlots+3, uintptr(len(ranges)), uintptr(slicePtr(ranges)), uintptr(unsafe.Pointer(&count)))
	runtime.KeepAlive(ranges)
	return count, err
}

// IsColorFont reports whether the face has colour glyphs (IDWriteFontFace2).
func (f *IDWriteFontFace3) IsColorFont() bool {
	return BOOL(int32(uint32(com.Call(f.Ptr(), fontFace1Slots)))).AsBool()
}

// GetWeight returns the face's weight.
func (f *IDWriteFontFace3) GetWeight() int32 {
	return int32(uint32(com.Call(f.Ptr(), fontFace2Slots+2)))
}

// GetStyle returns the face's style.
func (f *IDWriteFontFace3) GetStyle() int32 {
	return int32(uint32(com.Call(f.Ptr(), fontFace2Slots+4)))
}

// GetFamilyNames returns the face's localized family names.
func (f *IDWriteFontFace3) GetFamilyNames() (*IDWriteLocalizedStrings, error) {
	var out *IDWriteLocalizedStrings
	err := com.CallHR(f.Ptr(), fontFace2Slots+5, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// GetRecommendedRenderingMode returns the rendering and grid fit modes for a size (IDWriteFontFace3).
func (f *IDWriteFontFace3) GetRecommendedRenderingMode(fontEmSize, dpiX, dpiY float32, transform *DWRITE_MATRIX, isSideways bool, outlineThreshold, measuringMode int32, renderingParams *IDWriteRenderingParams) (renderingMode int32, gridFitMode int32, err error) {
	call := floatFn[func(unsafe.Pointer, float32, float32, float32, unsafe.Pointer, int32, int32, int32, unsafe.Pointer, unsafe.Pointer, unsafe.Pointer) int32](f.Ptr(), fontFace2Slots+9)
	r := call(f.Ptr(), fontEmSize, dpiX, dpiY, unsafe.Pointer(transform), int32(BoolOf(isSideways)), outlineThreshold, measuringMode, unsafe.Pointer(renderingParams), unsafe.Pointer(&renderingMode), unsafe.Pointer(&gridFitMode))
	return renderingMode, gridFitMode, hr32(r)
}

// IDWriteFontFallbackBuilder builds font fallbacks.
type IDWriteFontFallbackBuilder struct{ com.IUnknown }

// AddMapping maps ranges to target families.
func (b *IDWriteFontFallbackBuilder) AddMapping(ranges []DWRITE_UNICODE_RANGE, targetFamilyNames []*uint16, collection *IDWriteFontCollection1, localeName *uint16, baseFamilyName *uint16, scale float32) error {
	call := floatFn[func(unsafe.Pointer, unsafe.Pointer, uint32, unsafe.Pointer, uint32, unsafe.Pointer, unsafe.Pointer, unsafe.Pointer, float32) int32](b.Ptr(), com.IUnknownSlots)
	r := call(b.Ptr(), slicePtr(ranges), uint32(len(ranges)), slicePtr(targetFamilyNames), uint32(len(targetFamilyNames)), unsafe.Pointer(collection), unsafe.Pointer(localeName), unsafe.Pointer(baseFamilyName), scale)
	runtime.KeepAlive(ranges)
	runtime.KeepAlive(targetFamilyNames)
	return hr32(r)
}

// AddMappings appends every mapping of a fallback.
func (b *IDWriteFontFallbackBuilder) AddMappings(fallback *IDWriteFontFallback) error {
	return com.CallHR(b.Ptr(), com.IUnknownSlots+1, uintptr(fallback.Ptr()))
}

// CreateFontFallback creates the fallback.
func (b *IDWriteFontFallbackBuilder) CreateFontFallback() (*IDWriteFontFallback, error) {
	var out *IDWriteFontFallback
	err := com.CallHR(b.Ptr(), com.IUnknownSlots+2, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDWriteTypography is a list of font features.
type IDWriteTypography struct{ com.IUnknown }

// AddFontFeature adds a feature.
func (t *IDWriteTypography) AddFontFeature(feature DWRITE_FONT_FEATURE) error {
	return com.CallHR(t.Ptr(), com.IUnknownSlots, uintptr(uint64(feature.NameTag)|uint64(feature.Parameter)<<32))
}

// IDWriteTextFormat is a text format.
type IDWriteTextFormat struct{ com.IUnknown }

// GetIncrementalTabStop returns the distance between tab stops.
func (t *IDWriteTextFormat) GetIncrementalTabStop() float32 {
	call := floatFn[func(unsafe.Pointer) float32](t.Ptr(), com.IUnknownSlots+13)
	return call(t.Ptr())
}

// CastTextFormat1 queries t for IDWriteTextFormat1.
func (t *IDWriteTextFormat) CastTextFormat1() (*IDWriteTextFormat1, error) {
	return com.Cast[IDWriteTextFormat1](&t.IUnknown, &IID_IDWriteTextFormat1)
}

// IDWriteTextFormat1 is a text format with fallback support.
type IDWriteTextFormat1 struct{ IDWriteTextFormat }

// SetFontFallback sets the format's fallback.
func (t *IDWriteTextFormat1) SetFontFallback(fallback *IDWriteFontFallback) error {
	return com.CallHR(t.Ptr(), textFormatSlots+6, uintptr(fallback.Ptr()))
}

// IDWriteTextLayout is a laid-out text block.
type IDWriteTextLayout struct{ com.IUnknown }

// SetFontCollection sets the font collection of a range.
func (l *IDWriteTextLayout) SetFontCollection(collection *IDWriteFontCollection1, textRange DWRITE_TEXT_RANGE) error {
	return com.CallHR(l.Ptr(), textFormatSlots+2, uintptr(collection.Ptr()), textRange.arg())
}

// SetFontFamilyName sets the family of a range.
func (l *IDWriteTextLayout) SetFontFamilyName(familyName []uint16, textRange DWRITE_TEXT_RANGE) error {
	err := com.CallHR(l.Ptr(), textFormatSlots+3, uintptr(slicePtr(familyName)), textRange.arg())
	runtime.KeepAlive(familyName)
	return err
}

// SetFontWeight sets the weight of a range.
func (l *IDWriteTextLayout) SetFontWeight(weight int32, textRange DWRITE_TEXT_RANGE) error {
	return com.CallHR(l.Ptr(), textFormatSlots+4, uintptr(weight), textRange.arg())
}

// SetFontStyle sets the style of a range.
func (l *IDWriteTextLayout) SetFontStyle(style int32, textRange DWRITE_TEXT_RANGE) error {
	return com.CallHR(l.Ptr(), textFormatSlots+5, uintptr(style), textRange.arg())
}

// SetFontSize sets the size of a range.
func (l *IDWriteTextLayout) SetFontSize(fontSize float32, textRange DWRITE_TEXT_RANGE) error {
	call := floatFn[func(unsafe.Pointer, float32, uint64) int32](l.Ptr(), textFormatSlots+7)
	return hr32(call(l.Ptr(), fontSize, uint64(textRange.arg())))
}

// SetTypography sets the features of a range.
func (l *IDWriteTextLayout) SetTypography(typography *IDWriteTypography, textRange DWRITE_TEXT_RANGE) error {
	return com.CallHR(l.Ptr(), textFormatSlots+12, uintptr(typography.Ptr()), textRange.arg())
}

// Draw draws the layout through a renderer; clientDrawingContext is passed back to it unchanged.
func (l *IDWriteTextLayout) Draw(clientDrawingContext uintptr, renderer *IDWriteTextRenderer, originX, originY float32) error {
	call := floatFn[func(unsafe.Pointer, uintptr, unsafe.Pointer, float32, float32) int32](l.Ptr(), textFormatSlots+30)
	return hr32(call(l.Ptr(), clientDrawingContext, renderer.Ptr(), originX, originY))
}

// GetLineMetrics fills metrics and returns the line count.
func (l *IDWriteTextLayout) GetLineMetrics(metrics []DWRITE_LINE_METRICS) (uint32, error) {
	var count uint32
	err := com.CallHR(l.Ptr(), textFormatSlots+31, uintptr(slicePtr(metrics)), uintptr(len(metrics)), uintptr(unsafe.Pointer(&count)))
	runtime.KeepAlive(metrics)
	return count, err
}

// GetMetrics returns the layout's metrics.
func (l *IDWriteTextLayout) GetMetrics(metrics *DWRITE_TEXT_METRICS) error {
	return com.CallHR(l.Ptr(), textFormatSlots+32, uintptr(unsafe.Pointer(metrics)))
}

// HitTestTextPosition returns the caret position of a text position.
func (l *IDWriteTextLayout) HitTestTextPosition(textPosition uint32, isTrailingHit bool, pointX, pointY *float32, metrics *DWRITE_HIT_TEST_METRICS) error {
	return com.CallHR(l.Ptr(), textFormatSlots+37, uintptr(textPosition), uintptr(BoolOf(isTrailingHit)), uintptr(unsafe.Pointer(pointX)), uintptr(unsafe.Pointer(pointY)), uintptr(unsafe.Pointer(metrics)))
}

// IDWriteGlyphRunAnalysis rasterises a glyph run.
type IDWriteGlyphRunAnalysis struct{ com.IUnknown }

// GetAlphaTextureBounds returns the bounds of the alpha texture.
func (a *IDWriteGlyphRunAnalysis) GetAlphaTextureBounds(textureType int32) (RECT, error) {
	var rect RECT
	err := com.CallHR(a.Ptr(), com.IUnknownSlots, uintptr(textureType), uintptr(unsafe.Pointer(&rect)))
	return rect, err
}

// CreateAlphaTexture renders the alpha texture of bounds into buffer.
func (a *IDWriteGlyphRunAnalysis) CreateAlphaTexture(textureType int32, bounds *RECT, buffer []byte) error {
	err := com.CallHR(a.Ptr(), com.IUnknownSlots+1, uintptr(textureType), uintptr(unsafe.Pointer(bounds)), uintptr(slicePtr(buffer)), uintptr(len(buffer)))
	runtime.KeepAlive(buffer)
	return err
}

// IDWriteColorGlyphRunEnumerator1 enumerates colour glyph layers.
type IDWriteColorGlyphRunEnumerator1 struct{ com.IUnknown }

// MoveNext advances to the next run.
func (e *IDWriteColorGlyphRunEnumerator1) MoveNext() (bool, error) {
	var hasRun BOOL
	err := com.CallHR(e.Ptr(), com.IUnknownSlots, uintptr(unsafe.Pointer(&hasRun)))
	return hasRun.AsBool(), err
}

// GetCurrentRun returns the current run, valid until the next MoveNext (IDWriteColorGlyphRunEnumerator1).
func (e *IDWriteColorGlyphRunEnumerator1) GetCurrentRun() (*DWRITE_COLOR_GLYPH_RUN1, error) {
	var out *DWRITE_COLOR_GLYPH_RUN1
	err := com.CallHR(e.Ptr(), com.IUnknownSlots+2, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDWriteLocalizedStrings is a list of strings by locale.
type IDWriteLocalizedStrings struct{ com.IUnknown }

// FindLocaleName returns the index of a locale's string.
func (s *IDWriteLocalizedStrings) FindLocaleName(localeName []uint16) (uint32, bool, error) {
	var index uint32
	var exists BOOL
	err := com.CallHR(s.Ptr(), com.IUnknownSlots+1, uintptr(slicePtr(localeName)), uintptr(unsafe.Pointer(&index)), uintptr(unsafe.Pointer(&exists)))
	runtime.KeepAlive(localeName)
	return index, exists.AsBool(), err
}

// GetStringLength returns a string's length without its NUL.
func (s *IDWriteLocalizedStrings) GetStringLength(index uint32) (uint32, error) {
	var length uint32
	err := com.CallHR(s.Ptr(), com.IUnknownSlots+4, uintptr(index), uintptr(unsafe.Pointer(&length)))
	return length, err
}

// GetString copies a string, NUL-terminated, into buffer.
func (s *IDWriteLocalizedStrings) GetString(index uint32, buffer []uint16) error {
	err := com.CallHR(s.Ptr(), com.IUnknownSlots+5, uintptr(index), uintptr(slicePtr(buffer)), uintptr(len(buffer)))
	runtime.KeepAlive(buffer)
	return err
}

// IDWriteRenderingParams are rendering parameters.
type IDWriteRenderingParams struct{ com.IUnknown }

// GetGamma returns the gamma.
func (p *IDWriteRenderingParams) GetGamma() float32 {
	return floatFn[func(unsafe.Pointer) float32](p.Ptr(), com.IUnknownSlots)(p.Ptr())
}

// GetEnhancedContrast returns the ClearType enhanced contrast.
func (p *IDWriteRenderingParams) GetEnhancedContrast() float32 {
	return floatFn[func(unsafe.Pointer) float32](p.Ptr(), com.IUnknownSlots+1)(p.Ptr())
}

// GetPixelGeometry returns the pixel geometry.
func (p *IDWriteRenderingParams) GetPixelGeometry() int32 {
	return int32(uint32(com.Call(p.Ptr(), com.IUnknownSlots+3)))
}

// CastRenderingParams1 queries p for IDWriteRenderingParams1.
func (p *IDWriteRenderingParams) CastRenderingParams1() (*IDWriteRenderingParams1, error) {
	return com.Cast[IDWriteRenderingParams1](&p.IUnknown, &IID_IDWriteRenderingParams1)
}

// IDWriteRenderingParams1 adds the grayscale enhanced contrast.
type IDWriteRenderingParams1 struct{ IDWriteRenderingParams }

// GetGrayscaleEnhancedContrast returns the grayscale enhanced contrast.
func (p *IDWriteRenderingParams1) GetGrayscaleEnhancedContrast() float32 {
	return floatFn[func(unsafe.Pointer) float32](p.Ptr(), com.IUnknownSlots+5)(p.Ptr())
}
