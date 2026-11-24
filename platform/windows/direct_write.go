//go:build windows

package windows

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/dwrite"
	"github.com/megascan/gpui-go/internal/win/win32"
)

type fontInfo struct {
	fontFamilyH    []uint16
	fontFace       *dwrite.IDWriteFontFace3
	features       *dwrite.IDWriteTypography
	fallbacks      *dwrite.IDWriteFontFallback
	fontCollection *dwrite.IDWriteFontCollection1
}

func (f *fontInfo) release() {
	f.fontFace.Release()
	f.features.Release()
	if f.fallbacks != nil {
		f.fallbacks.Release()
	}
	f.fontCollection.Release()
}

type directWriteTextSystem struct {
	gpui.PlatformTextSystemDefaults
	components directWriteComponents
	mu         sync.RWMutex
	state      directWriteState
}

type directWriteComponents struct {
	locale                  []uint16
	factory                 *dwrite.IDWriteFactory5
	inMemoryLoader          *dwrite.IDWriteInMemoryFontFileLoader
	builder                 *dwrite.IDWriteFontSetBuilder1
	textRenderer            textRendererWrapper
	systemUIFontName        gpui.SharedString
	systemSubpixelRendering bool
}

// Close unregisters the in-memory font loader and releases the components, as upstream's drop.
func (c *directWriteComponents) Close() {
	if c.factory == nil {
		return
	}
	_ = c.factory.UnregisterFontFileLoader(c.inMemoryLoader)
	c.textRenderer.inner.Release()
	c.builder.Release()
	c.inMemoryLoader.Release()
	c.factory.Release()
	c.factory = nil
}

type directWriteState struct {
	systemFontCollection *dwrite.IDWriteFontCollection1
	customFontCollection *dwrite.IDWriteFontCollection1
	fonts                []fontInfo
	fontToFontID         map[gpui.Font]gpui.FontID
	fontInfoCache        map[uintptr]gpui.FontID
	layoutLineScratch    []uint16
}

var _ gpui.PlatformTextSystem = (*directWriteTextSystem)(nil)

func newDirectWriteTextSystem() (*directWriteTextSystem, error) {
	factory, err := dwrite.DWriteCreateFactory(dwrite.DWRITE_FACTORY_TYPE_SHARED)
	if err != nil {
		return nil, err
	}
	inMemoryLoader, err := factory.CreateInMemoryFontFileLoader()
	if err != nil {
		factory.Release()
		return nil, err
	}
	if err := factory.RegisterFontFileLoader(inMemoryLoader); err != nil {
		inMemoryLoader.Release()
		factory.Release()
		return nil, err
	}
	builder, err := factory.CreateFontSetBuilder()
	if err != nil {
		_ = factory.UnregisterFontFileLoader(inMemoryLoader)
		inMemoryLoader.Release()
		factory.Release()
		return nil, err
	}
	locale := make([]uint16, localeNameMaxLength)
	win32.GetUserDefaultLocaleName(locale)
	textRenderer := newTextRendererWrapper(locale)

	systemSubpixelRendering := getSystemSubpixelRendering()
	systemUIFontName := getSystemUIFontName()
	components := directWriteComponents{
		locale:                  locale,
		factory:                 factory,
		inMemoryLoader:          inMemoryLoader,
		builder:                 builder,
		textRenderer:            textRenderer,
		systemUIFontName:        systemUIFontName,
		systemSubpixelRendering: systemSubpixelRendering,
	}

	systemFontCollection, err := components.factory.GetSystemFontCollection(false, true)
	if err != nil {
		components.Close()
		return nil, err
	}
	if systemFontCollection == nil {
		components.Close()
		return nil, errors.New("Failed to get system font collection")
	}
	customFontSet, err := components.builder.CreateFontSet()
	if err != nil {
		systemFontCollection.Release()
		components.Close()
		return nil, err
	}
	customFontCollection, err := components.factory.CreateFontCollectionFromFontSet(customFontSet)
	customFontSet.Release()
	if err != nil {
		systemFontCollection.Release()
		components.Close()
		return nil, err
	}

	return &directWriteTextSystem{
		components: components,
		state: directWriteState{
			systemFontCollection: systemFontCollection,
			customFontCollection: customFontCollection,
			fontToFontID:         map[gpui.Font]gpui.FontID{},
			fontInfoCache:        map[uintptr]gpui.FontID{},
		},
	}, nil
}

// Close releases every DirectWrite object the text system holds.
func (t *directWriteTextSystem) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.state.fonts {
		t.state.fonts[i].release()
	}
	t.state.fonts = nil
	t.state.systemFontCollection.Release()
	t.state.customFontCollection.Release()
	t.state.systemFontCollection = nil
	t.state.customFontCollection = nil
	t.components.Close()
}

func (t *directWriteTextSystem) AddFonts(fonts [][]byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state.addFonts(&t.components, fonts)
}

func (t *directWriteTextSystem) AllFontNames() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state.allFontNames(&t.components)
}

func (t *directWriteTextSystem) FontID(font gpui.Font) (gpui.FontID, error) {
	t.mu.RLock()
	fontID, ok := t.state.fontToFontID[font]
	t.mu.RUnlock()
	if ok {
		return fontID, nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if fontID, ok := t.state.fontToFontID[font]; ok {
		return fontID, nil
	}
	fontID, ok = t.state.selectAndCacheFont(&t.components, font)
	if !ok {
		return 0, fmt.Errorf("Failed to select font: %+v", font)
	}
	return fontID, nil
}

func (t *directWriteTextSystem) FontMetrics(fontID gpui.FontID) gpui.FontMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state.fontMetrics(fontID)
}

func (t *directWriteTextSystem) TypographicBounds(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Bounds[float32], error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state.getTypographicBounds(fontID, glyphID)
}

func (t *directWriteTextSystem) Advance(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Size[float32], error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state.getAdvance(fontID, glyphID)
}

func (t *directWriteTextSystem) GlyphForChar(fontID gpui.FontID, ch rune) (gpui.GlyphID, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state.glyphForChar(fontID, ch)
}

func (t *directWriteTextSystem) GlyphRasterBounds(params gpui.RenderGlyphParams) (gpui.Bounds[gpui.DevicePixels], error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state.rasterBounds(&t.components, params)
}

func (t *directWriteTextSystem) RasterizeGlyph(params gpui.RenderGlyphParams, rasterBounds gpui.Bounds[gpui.DevicePixels]) (gpui.Size[gpui.DevicePixels], []byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state.rasterizeGlyph(&t.components, params, rasterBounds)
}

func (t *directWriteTextSystem) LayoutLine(text string, fontSize gpui.Pixels, runs []gpui.FontRun) gpui.LineLayout {
	t.mu.Lock()
	defer t.mu.Unlock()
	layout, err := t.state.layoutLine(&t.components, text, fontSize, runs)
	if err != nil {
		gpuiutil.LogErr(err)
		return gpui.LineLayout{FontSize: fontSize}
	}
	return layout
}

func (t *directWriteTextSystem) RecommendedRenderingMode(fontID gpui.FontID, fontSize gpui.Pixels) gpui.TextRenderingMode {
	if t.components.systemSubpixelRendering {
		return gpui.TextRenderingModeSubpixel
	}
	return gpui.TextRenderingModeGrayscale
}

func fontFaceKey(face *dwrite.IDWriteFontFace3) uintptr {
	unknown, err := face.QueryInterface(&com.IID_IUnknown)
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	key := uintptr(unsafe.Pointer(unknown))
	unknown.Release()
	return key
}

func (s *directWriteState) selectFont(components *directWriteComponents, font gpui.Font) (gpui.FontID, bool) {
	var info fontInfo
	found := false
	for _, fontCollection := range []*dwrite.IDWriteFontCollection1{s.customFontCollection, s.systemFontCollection} {
		if info, found = makeFontFromFontCollection(font, fontCollection, components.factory, s.systemFontCollection, components.systemUIFontName); found {
			break
		}
	}
	if !found {
		return 0, false
	}
	fontID := gpui.FontID(len(s.fonts))
	fontFaceKey := fontFaceKey(info.fontFace)
	s.fonts = append(s.fonts, info)
	s.fontInfoCache[fontFaceKey] = fontID
	return fontID, true
}

func (s *directWriteState) selectAndCacheFont(components *directWriteComponents, font gpui.Font) (gpui.FontID, bool) {
	fontID, ok := s.selectFont(components, font)
	if !ok {
		collection, err := components.factory.GetSystemFontCollection(false, true)
		fontCollectionUpdated := true
		if err != nil {
			gpuiutil.LogErr(err)
			fontCollectionUpdated = false
		}
		if fontCollectionUpdated && collection != nil {
			s.systemFontCollection.Release()
			s.systemFontCollection = collection
		}
		fontID, ok = s.selectFont(components, font)
	}
	if !ok {
		return 0, false
	}
	s.fontToFontID[font] = fontID
	return fontID, true
}

func (s *directWriteState) addFonts(components *directWriteComponents, fonts [][]byte) error {
	for _, fontData := range fonts {
		fontFile, err := components.inMemoryLoader.CreateInMemoryFontFileReference(components.factory, fontData, nil)
		if err != nil {
			return err
		}
		err = components.builder.AddFontFile(fontFile)
		fontFile.Release()
		if err != nil {
			return err
		}
	}
	set, err := components.builder.CreateFontSet()
	if err != nil {
		return err
	}
	collection, err := components.factory.CreateFontCollectionFromFontSet(set)
	set.Release()
	if err != nil {
		return err
	}
	s.customFontCollection.Release()
	s.customFontCollection = collection
	return nil
}

func generateFontFallbacks(fallbacks gpui.FontFallbacks, factory *dwrite.IDWriteFactory5, systemFontCollection *dwrite.IDWriteFontCollection1) (*dwrite.IDWriteFontFallback, error) {
	fallbackList := fallbacks.FallbackList()
	if len(fallbackList) == 0 {
		return nil, nil
	}
	builder, err := factory.CreateFontFallbackBuilder()
	if err != nil {
		return nil, err
	}
	defer builder.Release()
	fontSet, err := systemFontCollection.GetFontSet()
	if err != nil {
		return nil, err
	}
	defer fontSet.Release()
	var unicodeRanges []dwrite.DWRITE_UNICODE_RANGE
	for _, familyName := range fallbackList {
		familyNameH := dwrite.PCWSTR(familyName)
		fonts, err := fontSet.GetMatchingFonts(familyNameH, dwrite.DWRITE_FONT_WEIGHT_NORMAL, dwrite.DWRITE_FONT_STRETCH_NORMAL, dwrite.DWRITE_FONT_STYLE_NORMAL)
		if err != nil {
			gpuiutil.LogErr(err)
			continue
		}
		fontFace, err := fonts.GetFontFaceReference(0)
		fonts.Release()
		if err != nil {
			continue
		}
		font, err := fontFace.CreateFontFace()
		fontFace.Release()
		if err != nil {
			return nil, err
		}
		count, _ := font.GetUnicodeRanges(nil)
		if count == 0 {
			font.Release()
			continue
		}
		unicodeRanges = unicodeRanges[:0]
		unicodeRanges = append(unicodeRanges, make([]dwrite.DWRITE_UNICODE_RANGE, count)...)
		_, err = font.GetUnicodeRanges(unicodeRanges)
		font.Release()
		if err != nil {
			gpuiutil.LogErr(err)
			continue
		}
		if err := builder.AddMapping(unicodeRanges, []*uint16{&familyNameH[0]}, nil, nil, nil, 1.0); err != nil {
			return nil, err
		}
	}
	systemFallbacks, err := factory.GetSystemFontFallback()
	if err != nil {
		return nil, err
	}
	defer systemFallbacks.Release()
	if err := builder.AddMappings(systemFallbacks); err != nil {
		return nil, err
	}
	return builder.CreateFontFallback()
}

func generateFontFeatures(factory *dwrite.IDWriteFactory5, fontFeatures gpui.FontFeatures) (*dwrite.IDWriteTypography, error) {
	directWriteFeatures, err := factory.CreateTypography()
	if err != nil {
		return nil, err
	}
	if err := applyFontFeatures(directWriteFeatures, fontFeatures); err != nil {
		directWriteFeatures.Release()
		return nil, err
	}
	return directWriteFeatures, nil
}

const systemUIFontNameAlias = ".SystemUIFont"

func makeFontFromFontCollection(font gpui.Font, collection *dwrite.IDWriteFontCollection1, factory *dwrite.IDWriteFactory5, systemFontCollection *dwrite.IDWriteFontCollection1, systemUIFontName gpui.SharedString) (fontInfo, bool) {
	family := systemUIFontName
	if font.Family != systemUIFontNameAlias {
		family = gpui.FontNameWithFallbacksShared(font.Family, systemUIFontName)
	}
	fontset, err := collection.GetFontSet()
	if err != nil {
		gpuiutil.LogErr(err)
		return fontInfo{}, false
	}
	defer fontset.Release()
	fontFamilyH := dwrite.PCWSTR(string(family))
	matching, err := fontset.GetMatchingFonts(fontFamilyH, fontWeightToDwrite(font.Weight), dwrite.DWRITE_FONT_STRETCH_NORMAL, fontStyleToDwrite(font.Style))
	if err != nil {
		gpuiutil.LogErr(err)
		return fontInfo{}, false
	}
	defer matching.Release()
	totalNumber := matching.GetFontCount()
	for index := range totalNumber {
		info, ok := func() (fontInfo, bool) {
			fontFaceRef, err := matching.GetFontFaceReference(index)
			if err != nil {
				gpuiutil.LogErr(err)
				return fontInfo{}, false
			}
			defer fontFaceRef.Release()
			fontFace, err := fontFaceRef.CreateFontFace()
			if err != nil {
				gpuiutil.LogErr(err)
				return fontInfo{}, false
			}
			directWriteFeatures, err := generateFontFeatures(factory, font.Features)
			if err != nil {
				gpuiutil.LogErr(err)
				fontFace.Release()
				return fontInfo{}, false
			}
			var fallbacks *dwrite.IDWriteFontFallback
			if fb, ok := font.Fallbacks.Get(); ok {
				fallbacks, err = generateFontFallbacks(fb, factory, systemFontCollection)
				if err != nil {
					gpuiutil.LogErr(err)
					fallbacks = nil
				}
			}
			collection.AddRef()
			return fontInfo{
				fontFamilyH:    fontFamilyH,
				fontFace:       fontFace,
				features:       directWriteFeatures,
				fallbacks:      fallbacks,
				fontCollection: collection,
			}, true
		}()
		if ok {
			return info, true
		}
	}
	return fontInfo{}, false
}

func (s *directWriteState) layoutLine(components *directWriteComponents, text string, fontSize gpui.Pixels, fontRuns []gpui.FontRun) (gpui.LineLayout, error) {
	if !strings.ContainsFunc(text, isLineSeparator) {
		return s.layoutLineWithoutSeparators(components, text, fontSize, fontRuns)
	}

	layout := gpui.LineLayout{FontSize: fontSize, Len: len(text)}
	remainingRuns := fontRuns
	runOffset := 0
	textOffset := 0
	segmentRuns := make([]gpui.FontRun, 0, 4)
	for _, segment := range splitInclusive(text, isLineSeparator) {
		segmentRuns = segmentRuns[:0]
		remainingBytes := len(segment)
		for remainingBytes > 0 {
			if len(remainingRuns) == 0 {
				break
			}
			run := remainingRuns[0]
			n := min(remainingBytes, run.Len-runOffset)
			if n > 0 {
				segmentRuns = append(segmentRuns, gpui.FontRun{Len: n, FontID: run.FontID})
			}
			remainingBytes -= n
			runOffset += n
			if runOffset == run.Len {
				remainingRuns = remainingRuns[1:]
				runOffset = 0
			}
		}

		segmentLayout, err := s.layoutLineWithoutSeparators(components, segment, fontSize, segmentRuns)
		if err != nil {
			return gpui.LineLayout{}, err
		}
		for _, run := range segmentLayout.Runs {
			for i := range run.Glyphs {
				run.Glyphs[i].Index += textOffset
				run.Glyphs[i].Position.X += layout.Width
			}
			if n := len(layout.Runs); n > 0 && layout.Runs[n-1].FontID == run.FontID {
				layout.Runs[n-1].Glyphs = append(layout.Runs[n-1].Glyphs, run.Glyphs...)
			} else {
				layout.Runs = append(layout.Runs, run)
			}
		}
		layout.Width += segmentLayout.Width
		layout.Ascent = max(layout.Ascent, segmentLayout.Ascent)
		layout.Descent = max(layout.Descent, segmentLayout.Descent)
		textOffset += len(segment)
	}
	return layout, nil
}

func splitInclusive(text string, pred func(rune) bool) []string {
	var segments []string
	start := 0
	for i, r := range text {
		if pred(r) {
			end := i + utf8.RuneLen(r)
			segments = append(segments, text[start:end])
			start = end
		}
	}
	if start < len(text) {
		segments = append(segments, text[start:])
	}
	return segments
}

func utf16Len(s string) uint32 {
	n := uint32(0)
	for _, r := range s {
		n += uint32(utf16.RuneLen(r))
	}
	return n
}

var rendererContexts win32.Handles[*rendererContext]

func (s *directWriteState) layoutLineWithoutSeparators(components *directWriteComponents, text string, fontSize gpui.Pixels, fontRuns []gpui.FontRun) (gpui.LineLayout, error) {
	if len(fontRuns) == 0 {
		return gpui.LineLayout{FontSize: fontSize}, nil
	}
	s.layoutLineScratch = s.layoutLineScratch[:0]
	for _, r := range text {
		s.layoutLineScratch = utf16.AppendRune(s.layoutLineScratch, r)
	}
	textWide := s.layoutLineScratch

	utf8Offset := 0
	utf16Offset := uint32(0)
	textLayout, err := func() (*dwrite.IDWriteTextLayout, error) {
		firstRun := fontRuns[0]
		fontInfo := &s.fonts[firstRun.FontID]
		collection := fontInfo.fontCollection
		baseFormat, err := components.factory.CreateTextFormat(fontInfo.fontFamilyH, collection, fontInfo.fontFace.GetWeight(), fontInfo.fontFace.GetStyle(), dwrite.DWRITE_FONT_STRETCH_NORMAL, float32(fontSize), components.locale)
		if err != nil {
			return nil, err
		}
		format, err := baseFormat.CastTextFormat1()
		baseFormat.Release()
		if err != nil {
			return nil, err
		}
		defer format.Release()
		if fontInfo.fallbacks != nil {
			if err := format.SetFontFallback(fontInfo.fallbacks); err != nil {
				return nil, err
			}
		}

		layout, err := components.factory.CreateTextLayout(textWide, &format.IDWriteTextFormat, float32(math.Inf(1)), float32(math.Inf(1)))
		if err != nil {
			return nil, err
		}
		currentText := text[utf8Offset : utf8Offset+firstRun.Len]
		utf8Offset += firstRun.Len
		currentTextUTF16Length := utf16Len(currentText)
		textRange := dwrite.DWRITE_TEXT_RANGE{StartPosition: utf16Offset, Length: currentTextUTF16Length}
		if err := layout.SetTypography(fontInfo.features, textRange); err != nil {
			layout.Release()
			return nil, err
		}
		utf16Offset += currentTextUTF16Length
		return layout, nil
	}()
	if err != nil {
		return gpui.LineLayout{}, err
	}
	defer textLayout.Release()

	var firstMetrics [4]dwrite.DWRITE_LINE_METRICS
	if _, err := textLayout.GetLineMetrics(firstMetrics[:]); err != nil {
		return gpui.LineLayout{}, err
	}
	ascent := gpui.Px(firstMetrics[0].Baseline)
	descent := gpui.Px(firstMetrics[0].Height - firstMetrics[0].Baseline)

	breakLigatures := true
	for _, run := range fontRuns[1:] {
		fontInfo := &s.fonts[run.FontID]
		currentText := text[utf8Offset : utf8Offset+run.Len]
		utf8Offset += run.Len
		currentTextUTF16Length := utf16Len(currentText)

		collection := fontInfo.fontCollection
		textRange := dwrite.DWRITE_TEXT_RANGE{StartPosition: utf16Offset, Length: currentTextUTF16Length}
		utf16Offset += currentTextUTF16Length
		if err := textLayout.SetFontCollection(collection, textRange); err != nil {
			return gpui.LineLayout{}, err
		}
		if err := textLayout.SetFontFamilyName(fontInfo.fontFamilyH, textRange); err != nil {
			return gpui.LineLayout{}, err
		}
		size := float32(fontSize)
		if breakLigatures {
			size = math.Nextafter32(size, float32(math.Inf(1)))
		}
		if err := textLayout.SetFontSize(size, textRange); err != nil {
			return gpui.LineLayout{}, err
		}
		if err := textLayout.SetFontStyle(fontInfo.fontFace.GetStyle(), textRange); err != nil {
			return gpui.LineLayout{}, err
		}
		if err := textLayout.SetFontWeight(fontInfo.fontFace.GetWeight(), textRange); err != nil {
			return gpui.LineLayout{}, err
		}
		if err := textLayout.SetTypography(fontInfo.features, textRange); err != nil {
			return gpui.LineLayout{}, err
		}

		breakLigatures = !breakLigatures
	}

	var runs []gpui.ShapedRun
	rendererContext := &rendererContext{
		textSystem:     s,
		components:     components,
		indexConverter: newStringIndexConverter(text),
		runs:           &runs,
		layout:         textLayout,
	}
	key := rendererContexts.Insert(rendererContext)
	err = textLayout.Draw(key, components.textRenderer.inner, 0.0, 0.0)
	rendererContexts.Take(key)
	if err != nil {
		return gpui.LineLayout{}, err
	}
	width := gpui.Px(rendererContext.width)

	return gpui.LineLayout{
		FontSize: fontSize,
		Width:    width,
		Ascent:   ascent,
		Descent:  descent,
		Runs:     runs,
		Len:      len(text),
	}, nil
}

func (s *directWriteState) fontMetrics(fontID gpui.FontID) gpui.FontMetrics {
	fontInfo := &s.fonts[fontID]
	var metrics dwrite.DWRITE_FONT_METRICS1
	fontInfo.fontFace.GetMetrics(&metrics)

	return gpui.FontMetrics{
		UnitsPerEm:         uint32(metrics.Base.DesignUnitsPerEm),
		Ascent:             float32(metrics.Base.Ascent),
		Descent:            -float32(metrics.Base.Descent),
		LineGap:            float32(metrics.Base.LineGap),
		UnderlinePosition:  float32(metrics.Base.UnderlinePosition),
		UnderlineThickness: float32(metrics.Base.UnderlineThickness),
		CapHeight:          float32(metrics.Base.CapHeight),
		XHeight:            float32(metrics.Base.XHeight),
		BoundingBox: gpui.Bounds[float32]{
			Origin: gpui.Point[float32]{
				X: float32(metrics.GlyphBoxLeft),
				Y: float32(metrics.GlyphBoxBottom),
			},
			Size: gpui.Size[float32]{
				Width:  float32(metrics.GlyphBoxRight - metrics.GlyphBoxLeft),
				Height: float32(metrics.GlyphBoxTop - metrics.GlyphBoxBottom),
			},
		},
	}
}

func withGlyphRun[R any](s *directWriteState, params gpui.RenderGlyphParams, f func(*dwrite.DWRITE_GLYPH_RUN, dwrite.Vector2) R) R {
	font := &s.fonts[params.FontID]
	glyphID := [1]uint16{uint16(params.GlyphID)}
	advance := [1]float32{0.0}
	offset := [1]dwrite.DWRITE_GLYPH_OFFSET{}
	glyphRun := dwrite.DWRITE_GLYPH_RUN{
		FontFace:      font.fontFace.AsFontFace(),
		FontEmSize:    float32(params.FontSize) * params.ScaleFactor,
		GlyphCount:    1,
		GlyphIndices:  &glyphID[0],
		GlyphAdvances: &advance[0],
		GlyphOffsets:  &offset[0],
		IsSideways:    0,
		BidiLevel:     0,
	}
	baselineOrigin := dwrite.Vector2{
		X: float32(params.SubpixelVariant.X) / float32(gpui.SubpixelVariantsX),
		Y: float32(params.SubpixelVariant.Y) / float32(gpui.SubpixelVariantsY),
	}
	return f(&glyphRun, baselineOrigin)
}

func (s *directWriteState) renderingMode(params gpui.RenderGlyphParams) (int32, int32, error) {
	font := &s.fonts[params.FontID]
	renderingMode, gridFitMode, err := font.fontFace.GetRecommendedRenderingMode(
		float32(params.FontSize)*params.ScaleFactor,
		96.0,
		96.0,
		nil,
		false,
		dwrite.DWRITE_OUTLINE_THRESHOLD_ANTIALIASED,
		dwrite.DWRITE_MEASURING_MODE_NATURAL,
		nil,
	)
	if err != nil {
		return 0, 0, err
	}
	if renderingMode == dwrite.DWRITE_RENDERING_MODE1_OUTLINE {
		renderingMode = dwrite.DWRITE_RENDERING_MODE1_NATURAL_SYMMETRIC
	}
	return renderingMode, gridFitMode, nil
}

func (s *directWriteState) glyphAnalysis(components *directWriteComponents, params gpui.RenderGlyphParams) (*dwrite.IDWriteGlyphRunAnalysis, error) {
	renderingMode, gridFitMode, err := s.renderingMode(params)
	if err != nil {
		return nil, err
	}
	antialiasMode := dwrite.DWRITE_TEXT_ANTIALIAS_MODE_GRAYSCALE
	if params.SubpixelRendering {
		antialiasMode = dwrite.DWRITE_TEXT_ANTIALIAS_MODE_CLEARTYPE
	}
	type result struct {
		analysis *dwrite.IDWriteGlyphRunAnalysis
		err      error
	}
	r := withGlyphRun(s, params, func(glyphRun *dwrite.DWRITE_GLYPH_RUN, origin dwrite.Vector2) result {
		analysis, err := components.factory.CreateGlyphRunAnalysis(
			glyphRun,
			nil,
			renderingMode,
			dwrite.DWRITE_MEASURING_MODE_NATURAL,
			gridFitMode,
			antialiasMode,
			origin.X,
			origin.Y,
		)
		return result{analysis, err}
	})
	return r.analysis, r.err
}

func releaseColorLayers(layers []colorLayer) {
	for _, layer := range layers {
		layer.analysis.Release()
	}
}

func (s *directWriteState) colorLayers(components *directWriteComponents, params gpui.RenderGlyphParams) ([]colorLayer, error) {
	renderingMode, gridFitMode, err := s.renderingMode(params)
	if err != nil {
		return nil, err
	}
	type result struct {
		layers []colorLayer
		err    error
	}
	r := withGlyphRun(s, params, func(glyphRun *dwrite.DWRITE_GLYPH_RUN, origin dwrite.Vector2) result {
		colorRuns, err := components.factory.TranslateColorGlyphRun(
			origin,
			glyphRun,
			nil,
			dwrite.DWRITE_GLYPH_IMAGE_FORMATS_COLR,
			dwrite.DWRITE_MEASURING_MODE_NATURAL,
			nil,
			0,
		)
		if err != nil {
			if dwrite.IsHRESULT(err, dwrite.DWRITE_E_NOCOLOR) {
				return result{}
			}
			return result{err: err}
		}
		defer colorRuns.Release()
		var layers []colorLayer
		for {
			hasRun, err := colorRuns.MoveNext()
			if err != nil {
				releaseColorLayers(layers)
				return result{err: err}
			}
			if !hasRun {
				break
			}
			colorRun, err := colorRuns.GetCurrentRun()
			if err != nil {
				releaseColorLayers(layers)
				return result{err: err}
			}
			imageFormat := colorRun.GlyphImageFormat &^ dwrite.DWRITE_GLYPH_IMAGE_FORMATS_TRUETYPE
			if imageFormat != dwrite.DWRITE_GLYPH_IMAGE_FORMATS_COLR {
				continue
			}
			analysis, err := components.factory.CreateGlyphRunAnalysis(
				&colorRun.Base.GlyphRun,
				nil,
				renderingMode,
				dwrite.DWRITE_MEASURING_MODE_NATURAL,
				gridFitMode,
				dwrite.DWRITE_TEXT_ANTIALIAS_MODE_GRAYSCALE,
				colorRun.Base.BaselineOriginX,
				colorRun.Base.BaselineOriginY,
			)
			if err != nil {
				releaseColorLayers(layers)
				return result{err: err}
			}
			color := colorRun.Base.RunColor
			layers = append(layers, colorLayer{
				color: gpui.Rgba{
					R: color.R,
					G: color.G,
					B: color.B,
					A: color.A,
				},
				analysis: analysis,
			})
		}
		return result{layers: layers}
	})
	return r.layers, r.err
}

func (s *directWriteState) rasterBounds(components *directWriteComponents, params gpui.RenderGlyphParams) (gpui.Bounds[gpui.DevicePixels], error) {
	var colorLayers []colorLayer
	if params.IsEmoji {
		layers, err := s.colorLayers(components, params)
		if err != nil {
			gpuiutil.LogErr(fmt.Errorf("Failed to measure color glyph layers: %w", err))
		} else {
			colorLayers = layers
		}
	}
	var analyses []*dwrite.IDWriteGlyphRunAnalysis
	if len(colorLayers) == 0 {
		analysis, err := s.glyphAnalysis(components, params)
		if err != nil {
			return gpui.Bounds[gpui.DevicePixels]{}, err
		}
		analyses = []*dwrite.IDWriteGlyphRunAnalysis{analysis}
	} else {
		for _, layer := range colorLayers {
			analyses = append(analyses, layer.analysis)
		}
	}
	defer func() {
		for _, analysis := range analyses {
			analysis.Release()
		}
	}()
	textureType := dwrite.DWRITE_TEXTURE_ALIASED_1x1
	if params.SubpixelRendering {
		textureType = dwrite.DWRITE_TEXTURE_CLEARTYPE_3x1
	}

	var union gpui.Bounds[gpui.DevicePixels]
	haveUnion := false
	for _, analysis := range analyses {
		rect, err := analysis.GetAlphaTextureBounds(textureType)
		if err != nil {
			return gpui.Bounds[gpui.DevicePixels]{}, err
		}
		bounds := gpui.BoundsFromCorners(
			gpui.NewPoint(gpui.DevicePixels(rect.Left), gpui.DevicePixels(rect.Top)),
			gpui.NewPoint(gpui.DevicePixels(rect.Right), gpui.DevicePixels(rect.Bottom)),
		)
		if !bounds.IsEmpty() {
			if haveUnion {
				union = union.Union(bounds)
			} else {
				union = bounds
				haveUnion = true
			}
		}
	}
	return union, nil
}

func (s *directWriteState) glyphForChar(fontID gpui.FontID, ch rune) (gpui.GlyphID, bool) {
	fontInfo := &s.fonts[fontID]
	codepoints := [1]uint32{uint32(ch)}
	var glyphIndices [1]uint16
	if err := fontInfo.fontFace.GetGlyphIndices(codepoints[:], glyphIndices[:]); err != nil {
		gpuiutil.LogErr(err)
		return 0, false
	}
	return gpui.GlyphID(glyphIndices[0]), true
}

func (s *directWriteState) rasterizeGlyph(components *directWriteComponents, params gpui.RenderGlyphParams, glyphBounds gpui.Bounds[gpui.DevicePixels]) (gpui.Size[gpui.DevicePixels], []byte, error) {
	if glyphBounds.Size.Width == 0 || glyphBounds.Size.Height == 0 {
		return gpui.Size[gpui.DevicePixels]{}, nil, errors.New("glyph bounds are empty")
	}

	var bitmapData []byte
	if params.IsEmoji {
		if color, err := s.rasterizeColor(components, params, glyphBounds); err == nil {
			bitmapData = color
		} else {
			monochrome, err := s.rasterizeMonochrome(components, params, glyphBounds)
			if err != nil {
				return gpui.Size[gpui.DevicePixels]{}, nil, err
			}
			bitmapData = make([]byte, 0, len(monochrome)*4)
			for _, pixel := range monochrome {
				bitmapData = append(bitmapData, 0, 0, 0, pixel)
			}
		}
	} else {
		monochrome, err := s.rasterizeMonochrome(components, params, glyphBounds)
		if err != nil {
			return gpui.Size[gpui.DevicePixels]{}, nil, err
		}
		bitmapData = monochrome
	}

	return glyphBounds.Size, bitmapData, nil
}

func (s *directWriteState) rasterizeMonochrome(components *directWriteComponents, params gpui.RenderGlyphParams, glyphBounds gpui.Bounds[gpui.DevicePixels]) ([]byte, error) {
	glyphAnalysis, err := s.glyphAnalysis(components, params)
	if err != nil {
		return nil, err
	}
	defer glyphAnalysis.Release()
	rect := dwrite.RECT{
		Left:   int32(glyphBounds.Origin.X),
		Top:    int32(glyphBounds.Origin.Y),
		Right:  int32(glyphBounds.Size.Width + glyphBounds.Origin.X),
		Bottom: int32(glyphBounds.Size.Height + glyphBounds.Origin.Y),
	}
	if !params.SubpixelRendering {
		bitmapData := make([]byte, int(glyphBounds.Size.Width)*int(glyphBounds.Size.Height))
		if err := glyphAnalysis.CreateAlphaTexture(dwrite.DWRITE_TEXTURE_ALIASED_1x1, &rect, bitmapData); err != nil {
			return nil, err
		}
		return bitmapData, nil
	}

	width := int(glyphBounds.Size.Width)
	height := int(glyphBounds.Size.Height)
	pixelCount := width * height

	bitmapData := make([]byte, pixelCount*4)

	if err := glyphAnalysis.CreateAlphaTexture(dwrite.DWRITE_TEXTURE_CLEARTYPE_3x1, &rect, bitmapData[:pixelCount*3]); err != nil {
		return nil, err
	}

	for pixelIx := pixelCount - 1; pixelIx >= 0; pixelIx-- {
		src := pixelIx * 3
		dst := pixelIx * 4
		bitmapData[dst], bitmapData[dst+1], bitmapData[dst+2], bitmapData[dst+3] = bitmapData[src], bitmapData[src+1], bitmapData[src+2], 0
	}

	return bitmapData, nil
}

func (s *directWriteState) rasterizeColor(components *directWriteComponents, params gpui.RenderGlyphParams, glyphBounds gpui.Bounds[gpui.DevicePixels]) ([]byte, error) {
	layers, err := s.colorLayers(components, params)
	if err != nil {
		return nil, err
	}
	defer releaseColorLayers(layers)
	if len(layers) == 0 {
		return nil, errors.New("the glyph has no color layers")
	}
	bitmap := newColorGlyphBitmap(glyphBounds.Size)
	fontInfo := directXRendererGetFontInfo()
	var coverage []byte
	for _, layer := range layers {
		layerBounds, err := layer.analysis.GetAlphaTextureBounds(dwrite.DWRITE_TEXTURE_ALIASED_1x1)
		if err != nil {
			return nil, err
		}
		layerSize := gpui.NewSize(layerBounds.Right-layerBounds.Left, layerBounds.Bottom-layerBounds.Top)
		if layerSize.Width <= 0 || layerSize.Height <= 0 {
			continue
		}
		coverage = append(coverage[:0], make([]byte, int(layerSize.Width*layerSize.Height))...)
		if err := layer.analysis.CreateAlphaTexture(dwrite.DWRITE_TEXTURE_ALIASED_1x1, &layerBounds, coverage); err != nil {
			return nil, err
		}
		bitmap.composite(
			layer.color,
			gpui.NewBounds(
				gpui.NewPoint(
					layerBounds.Left-int32(glyphBounds.Origin.X),
					layerBounds.Top-int32(glyphBounds.Origin.Y),
				),
				layerSize,
			),
			coverage,
			fontInfo,
		)
	}
	return bitmap.intoStraightBgra(), nil
}

func (s *directWriteState) getTypographicBounds(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Bounds[float32], error) {
	font := s.fonts[fontID].fontFace
	glyphIndices := [1]uint16{uint16(glyphID)}
	var metrics [1]dwrite.DWRITE_GLYPH_METRICS
	if err := font.GetDesignGlyphMetrics(glyphIndices[:], metrics[:], false); err != nil {
		return gpui.Bounds[float32]{}, err
	}

	m := &metrics[0]
	advanceWidth := int32(m.AdvanceWidth)
	advanceHeight := int32(m.AdvanceHeight)
	leftSideBearing := m.LeftSideBearing
	rightSideBearing := m.RightSideBearing
	topSideBearing := m.TopSideBearing
	bottomSideBearing := m.BottomSideBearing
	verticalOriginY := m.VerticalOriginY

	yOffset := verticalOriginY + bottomSideBearing - advanceHeight
	width := advanceWidth - (leftSideBearing + rightSideBearing)
	height := advanceHeight - (topSideBearing + bottomSideBearing)

	return gpui.Bounds[float32]{
		Origin: gpui.Point[float32]{
			X: float32(leftSideBearing),
			Y: float32(yOffset),
		},
		Size: gpui.Size[float32]{
			Width:  float32(width),
			Height: float32(height),
		},
	}, nil
}

func (s *directWriteState) getAdvance(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Size[float32], error) {
	font := s.fonts[fontID].fontFace
	glyphIndices := [1]uint16{uint16(glyphID)}
	var metrics [1]dwrite.DWRITE_GLYPH_METRICS
	if err := font.GetDesignGlyphMetrics(glyphIndices[:], metrics[:], false); err != nil {
		return gpui.Size[float32]{}, err
	}

	return gpui.Size[float32]{
		Width:  float32(metrics[0].AdvanceWidth),
		Height: 0.0,
	}, nil
}

func (s *directWriteState) allFontNames(components *directWriteComponents) []string {
	result := getFontNamesFromCollection(s.systemFontCollection, components.locale)
	result = append(result, getFontNamesFromCollection(s.customFontCollection, components.locale)...)
	return result
}

type colorLayer struct {
	color    gpui.Rgba
	analysis *dwrite.IDWriteGlyphRunAnalysis
}

type colorGlyphBitmap struct {
	width  int
	height int
	blue   []float32
	green  []float32
	red    []float32
	alpha  []float32
}

func newColorGlyphBitmap(size gpui.Size[gpui.DevicePixels]) colorGlyphBitmap {
	width := int(max(size.Width, 0))
	height := int(max(size.Height, 0))
	n := width * height
	return colorGlyphBitmap{
		width:  width,
		height: height,
		blue:   make([]float32, n),
		green:  make([]float32, n),
		red:    make([]float32, n),
		alpha:  make([]float32, n),
	}
}

func clampI32(v, lo, hi int32) int32 {
	if lo > hi {
		panic("assertion failed: min <= max")
	}
	return min(max(v, lo), hi)
}

func clampF32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (b *colorGlyphBitmap) composite(color gpui.Rgba, bounds gpui.Bounds[int32], coverage []byte, fontInfo *directXFontInfo) {
	red, green, blue, colorAlpha := color.R, color.G, color.B, color.A
	brightness := colorBrightness([3]float32{red, green, blue})
	contrast := lightOnDarkContrast(fontInfo.grayscaleEnhancedContrast, brightness)

	width := int32(b.width)
	height := int32(b.height)
	left := clampI32(bounds.Origin.X, 0, width)
	right := clampI32(bounds.Origin.X+bounds.Size.Width, left, width)
	top := clampI32(bounds.Origin.Y, 0, height)
	bottom := clampI32(bounds.Origin.Y+bounds.Size.Height, top, height)
	n := int(right - left)
	skippedColumns := int(left - bounds.Origin.X)
	for y := top; y < bottom; y++ {
		coverageStart := int(y-bounds.Origin.Y)*int(bounds.Size.Width) + skippedColumns
		pixelStart := int(y)*b.width + int(left)
		if coverageStart < 0 || coverageStart+n > len(coverage) || pixelStart+n > len(b.alpha) {
			slog.Error("color glyph layer coverage is smaller than its bounds")
			return
		}
		coverageRow := coverage[coverageStart : coverageStart+n]
		blueRow := b.blue[pixelStart : pixelStart+n]
		greenRow := b.green[pixelStart : pixelStart+n]
		redRow := b.red[pixelStart : pixelStart+n]
		alphaRow := b.alpha[pixelStart : pixelStart+n]
		for index := range n {
			corrected := applyAlphaCorrection(
				enhanceContrast(float32(coverageRow[index])/255.0, contrast),
				brightness,
				fontInfo.gammaRatios,
			)
			alpha := clampF32(corrected*colorAlpha, 0.0, 1.0)
			remaining := 1.0 - alpha
			blueRow[index] = blue*alpha + blueRow[index]*remaining
			greenRow[index] = green*alpha + greenRow[index]*remaining
			redRow[index] = red*alpha + redRow[index]*remaining
			alphaRow[index] = alpha + alphaRow[index]*remaining
		}
	}
}

func (b *colorGlyphBitmap) intoStraightBgra() []byte {
	n := len(b.alpha)
	bytes := make([]byte, n*4)
	for index := range n {
		pixel := bytes[index*4 : index*4+4]
		pixelAlpha := clampF32(b.alpha[index], 0.0, 1.0)
		unpremultiply := float32(0.0)
		if pixelAlpha > 0.0 {
			unpremultiply = 1.0 / pixelAlpha
		}
		pixel[0] = toUnorm8(b.blue[index] * unpremultiply)
		pixel[1] = toUnorm8(b.green[index] * unpremultiply)
		pixel[2] = toUnorm8(b.red[index] * unpremultiply)
		pixel[3] = toUnorm8(pixelAlpha)
	}
	return bytes
}

func toUnorm8(value float32) uint8 {
	return uint8(clampF32(value, 0.0, 1.0)*255.0 + 0.5)
}

func colorBrightness(rgb [3]float32) float32 {
	return rgb[0]*0.30 + rgb[1]*0.59 + rgb[2]*0.11
}

func lightOnDarkContrast(enhancedContrast float32, brightness float32) float32 {
	return enhancedContrast * clampF32(4.0*(0.75-brightness), 0.0, 1.0)
}

func enhanceContrast(alpha float32, contrast float32) float32 {
	return alpha * (contrast + 1.0) / (alpha*contrast + 1.0)
}

func applyAlphaCorrection(alpha float32, brightness float32, gammaRatios [4]float32) float32 {
	brightnessAdjustment := gammaRatios[0]*brightness + gammaRatios[1]
	correction := brightnessAdjustment*alpha + (gammaRatios[2]*brightness + gammaRatios[3])
	return alpha + alpha*(1.0-alpha)*correction
}

type textRendererWrapper struct {
	inner *dwrite.IDWriteTextRenderer
}

func newTextRendererWrapper(localeStr []uint16) textRendererWrapper {
	inner := newTextRenderer(localeStr)
	return textRendererWrapper{inner: dwrite.NewTextRenderer(inner)}
}

type textRenderer struct {
	locale []uint16
}

func newTextRenderer(locale []uint16) *textRenderer {
	return &textRenderer{locale: locale}
}

type rendererContext struct {
	textSystem     *directWriteState
	components     *directWriteComponents
	indexConverter stringIndexConverter
	runs           *[]gpui.ShapedRun
	layout         *dwrite.IDWriteTextLayout
	width          float32
}

type clusterAnalyzer struct {
	utf16Idx   int
	glyphIdx   int
	glyphCount int
	clusterMap []uint16
}

func newClusterAnalyzer(clusterMap []uint16, glyphCount int) *clusterAnalyzer {
	return &clusterAnalyzer{
		utf16Idx:   0,
		glyphIdx:   0,
		glyphCount: glyphCount,
		clusterMap: clusterMap,
	}
}

func (c *clusterAnalyzer) next() (utf16Len int, glyphCount int, ok bool) {
	if c.utf16Idx >= len(c.clusterMap) {
		return 0, 0, false
	}
	startUTF16Idx := c.utf16Idx
	currentGlyph := int(c.clusterMap[startUTF16Idx])

	endUTF16Idx := startUTF16Idx + 1
	for endUTF16Idx < len(c.clusterMap) && int(c.clusterMap[endUTF16Idx]) == currentGlyph {
		endUTF16Idx++
	}

	utf16Len = endUTF16Idx - startUTF16Idx

	nextGlyph := c.glyphCount
	if endUTF16Idx < len(c.clusterMap) {
		nextGlyph = int(c.clusterMap[endUTF16Idx])
	}

	glyphCount = nextGlyph - currentGlyph

	c.utf16Idx = endUTF16Idx
	c.glyphIdx = nextGlyph

	return utf16Len, glyphCount, true
}

func (r *textRenderer) IsPixelSnappingDisabled(clientDrawingContext uintptr) (bool, error) {
	return false, nil
}

func (r *textRenderer) GetCurrentTransform(clientDrawingContext uintptr) (dwrite.DWRITE_MATRIX, error) {
	return dwrite.DWRITE_MATRIX{
		M11: 1.0,
		M12: 0.0,
		M21: 0.0,
		M22: 1.0,
		Dx:  0.0,
		Dy:  0.0,
	}, nil
}

func (r *textRenderer) GetPixelsPerDip(clientDrawingContext uintptr) (float32, error) {
	return 1.0, nil
}

func (r *textRenderer) DrawGlyphRun(clientDrawingContext uintptr, measuringMode int32, glyphRun *dwrite.DWRITE_GLYPH_RUN, glyphRunDescription *dwrite.DWRITE_GLYPH_RUN_DESCRIPTION, clientDrawingEffect *com.IUnknown) error {
	glyphCount := int(glyphRun.GlyphCount)
	desc := glyphRunDescription
	var text []uint16
	hasText := false
	if glyphCount == 0 {
		t, err := sliceFromNullable(desc.String, int(desc.StringLength), "DirectWrite returned a null run string")
		if err != nil {
			return err
		}
		text = t
		hasText = true
	}
	context, ok := rendererContexts.Get(clientDrawingContext)
	if !ok {
		return dwrite.NewError(com.E_INVALIDARG, "unknown renderer context")
	}
	if glyphRun.FontFace == nil {
		return nil
	}
	fontFace, err := com.Cast[dwrite.IDWriteFontFace3](&glyphRun.FontFace.IUnknown, &dwrite.IID_IDWriteFontFace3)
	if err != nil {
		return dwrite.NewError(dwrite.DWRITE_E_UNSUPPORTEDOPERATION, "Failed to cast font face")
	}
	defer fontFace.Release()

	fontFaceKey := fontFaceKey(fontFace)
	fontID, ok := context.textSystem.fontInfoCache[fontFaceKey]
	if !ok {
		font, ok := fontFaceToFont(fontFace, r.locale)
		if !ok {
			return dwrite.NewError(dwrite.DWRITE_E_NOFONT, "Failed to create font")
		}
		if id, ok := context.textSystem.fontToFontID[font]; ok {
			fontID = id
		} else {
			id, ok := context.textSystem.selectAndCacheFont(context.components, font)
			if !ok {
				return dwrite.NewError(dwrite.DWRITE_E_NOFONT, "Failed to create font")
			}
			fontID = id
		}
		context.textSystem.fontInfoCache[fontFaceKey] = fontID
	}

	if hasText {
		var glyphs []gpui.ShapedGlyph
		for offset, character := range text {
			if character != uint16('\t') {
				continue
			}
			position := desc.TextPosition + uint32(offset)
			var x, y float32
			var metrics dwrite.DWRITE_HIT_TEST_METRICS
			var space [1]uint16
			if err := context.layout.HitTestTextPosition(position, false, &x, &y, &metrics); err != nil {
				return err
			}
			if err := fontFace.GetGlyphIndices([]uint32{uint32(' ')}, space[:]); err != nil {
				return err
			}
			context.indexConverter.advanceToUTF16Ix(int(position))
			glyphs = append(glyphs, gpui.ShapedGlyph{
				ID:       gpui.GlyphID(uint32(space[0])),
				Position: gpui.NewPoint(gpui.Px(context.width), gpui.Px(0.0)),
				Index:    context.indexConverter.utf8Ix,
				IsEmoji:  false,
			})
			context.width += metrics.Width
		}
		if len(glyphs) > 0 {
			*context.runs = append(*context.runs, gpui.ShapedRun{FontID: fontID, Glyphs: glyphs})
		}
	} else {
		colorFont := fontFace.IsColorFont()

		glyphIDs, err := sliceFromNullable(glyphRun.GlyphIndices, glyphCount, "DirectWrite returned a null glyph indices array")
		if err != nil {
			return err
		}
		glyphAdvances, err := sliceFromNullable(glyphRun.GlyphAdvances, glyphCount, "DirectWrite returned a null glyph advances array")
		if err != nil {
			return err
		}
		glyphOffsets, err := sliceFromNullable(glyphRun.GlyphOffsets, glyphCount, "DirectWrite returned a null glyph offsets array")
		if err != nil {
			return err
		}
		clusterMap, err := sliceFromNullable(desc.ClusterMap, int(desc.StringLength), "DirectWrite returned a null cluster map")
		if err != nil {
			return err
		}

		clusterAnalyzer := newClusterAnalyzer(clusterMap, glyphCount)
		utf16Idx := int(desc.TextPosition)
		glyphIdx := 0
		glyphs := make([]gpui.ShapedGlyph, 0, glyphCount)
		for {
			clusterUTF16Len, clusterGlyphCount, ok := clusterAnalyzer.next()
			if !ok {
				break
			}
			context.indexConverter.advanceToUTF16Ix(utf16Idx)
			utf16Idx += clusterUTF16Len
			for clusterGlyphIdx, glyphID := range glyphIDs[glyphIdx : glyphIdx+clusterGlyphCount] {
				id := gpui.GlyphID(uint32(glyphID))
				isEmoji := colorFont && isColorGlyph(fontFace, id, context.components.factory)
				thisGlyphIdx := glyphIdx + clusterGlyphIdx
				glyphs = append(glyphs, gpui.ShapedGlyph{
					ID: id,
					Position: gpui.NewPoint(
						gpui.Px(context.width+glyphOffsets[thisGlyphIdx].AdvanceOffset),
						gpui.Px(-glyphOffsets[thisGlyphIdx].AscenderOffset),
					),
					Index:   context.indexConverter.utf8Ix,
					IsEmoji: isEmoji,
				})
				context.width += glyphAdvances[thisGlyphIdx]
			}
			glyphIdx += clusterGlyphCount
		}
		*context.runs = append(*context.runs, gpui.ShapedRun{FontID: fontID, Glyphs: glyphs})
	}
	return nil
}

func (r *textRenderer) DrawUnderline(clientDrawingContext uintptr) error {
	return dwrite.NewError(com.E_NOTIMPL, "DrawUnderline unimplemented")
}

func (r *textRenderer) DrawStrikethrough(clientDrawingContext uintptr) error {
	return dwrite.NewError(com.E_NOTIMPL, "DrawStrikethrough unimplemented")
}

func (r *textRenderer) DrawInlineObject(clientDrawingContext uintptr) error {
	return dwrite.NewError(com.E_NOTIMPL, "DrawInlineObject unimplemented")
}

func sliceFromNullable[T any](ptr *T, n int, nullErrorMessage string) ([]T, error) {
	if ptr == nil {
		if n != 0 {
			return nil, dwrite.NewError(com.E_INVALIDARG, nullErrorMessage)
		}
		return []T{}, nil
	}
	return unsafe.Slice(ptr, n), nil
}

type stringIndexConverter struct {
	text    string
	utf8Ix  int
	utf16Ix int
}

func newStringIndexConverter(text string) stringIndexConverter {
	return stringIndexConverter{text: text}
}

func (c *stringIndexConverter) advanceToUTF8Ix(utf8Target int) {
	for ix, ch := range c.text[c.utf8Ix:] {
		if c.utf8Ix+ix >= utf8Target {
			c.utf8Ix += ix
			return
		}
		c.utf16Ix += utf16.RuneLen(ch)
	}
	c.utf8Ix = len(c.text)
}

func (c *stringIndexConverter) advanceToUTF16Ix(utf16Target int) {
	for ix, ch := range c.text[c.utf8Ix:] {
		if c.utf16Ix >= utf16Target {
			c.utf8Ix += ix
			return
		}
		c.utf16Ix += utf16.RuneLen(ch)
	}
	c.utf8Ix = len(c.text)
}

func fontStyleToDwrite(style gpui.FontStyle) int32 {
	switch style {
	case gpui.FontStyleNormal:
		return dwrite.DWRITE_FONT_STYLE_NORMAL
	case gpui.FontStyleItalic:
		return dwrite.DWRITE_FONT_STYLE_ITALIC
	case gpui.FontStyleOblique:
		return dwrite.DWRITE_FONT_STYLE_OBLIQUE
	}
	panic("internal error: entered unreachable code")
}

func fontStyleFromDwrite(value int32) gpui.FontStyle {
	switch value {
	case 0:
		return gpui.FontStyleNormal
	case 1:
		return gpui.FontStyleItalic
	case 2:
		return gpui.FontStyleOblique
	}
	panic("internal error: entered unreachable code")
}

func fontWeightToDwrite(weight gpui.FontWeight) int32 {
	return int32(weight)
}

func fontWeightFromDwrite(value int32) gpui.FontWeight {
	return gpui.FontWeight(float32(value))
}

func getFontNamesFromCollection(collection *dwrite.IDWriteFontCollection1, locale []uint16) []string {
	var result []string
	familyCount := collection.GetFontFamilyCount()
	for index := range familyCount {
		fontFamily, err := collection.GetFontFamily(index)
		if err != nil {
			gpuiutil.LogErr(err)
			continue
		}
		localizedFamilyName, err := fontFamily.GetFamilyNames()
		fontFamily.Release()
		if err != nil {
			gpuiutil.LogErr(err)
			continue
		}
		familyName, err := getName(localizedFamilyName, locale)
		if err != nil {
			gpuiutil.LogErr(err)
			continue
		}
		result = append(result, familyName)
	}
	return result
}

func fontFaceToFont(fontFace *dwrite.IDWriteFontFace3, locale []uint16) (gpui.Font, bool) {
	localizedFamilyName, err := fontFace.GetFamilyNames()
	if err != nil {
		gpuiutil.LogErr(err)
		return gpui.Font{}, false
	}
	familyName, err := getName(localizedFamilyName, locale)
	if err != nil {
		gpuiutil.LogErr(err)
		return gpui.Font{}, false
	}
	weight := fontFace.GetWeight()
	style := fontFace.GetStyle()
	return gpui.Font{
		Family:    gpui.SharedString(familyName),
		Features:  gpui.DefaultFontFeatures(),
		Weight:    fontWeightFromDwrite(weight),
		Style:     fontStyleFromDwrite(style),
		Fallbacks: gpui.None[gpui.FontFallbacks](),
	}, true
}

func applyFontFeatures(directWriteFeatures *dwrite.IDWriteTypography, features gpui.FontFeatures) error {
	tagValues := features.TagValueList()

	featureLiga := makeDirectWriteFeature("liga", 1)
	featureClig := makeDirectWriteFeature("clig", 1)
	featureCalt := makeDirectWriteFeature("calt", 1)

	for _, tv := range tagValues {
		tag, value := tv.Tag, tv.Value
		if tag == "liga" && value == 0 {
			featureLiga.Parameter = 0
			continue
		}
		if tag == "clig" && value == 0 {
			featureClig.Parameter = 0
			continue
		}
		if tag == "calt" && value == 0 {
			featureCalt.Parameter = 0
			continue
		}

		if err := directWriteFeatures.AddFontFeature(makeDirectWriteFeature(tag, value)); err != nil {
			return err
		}
	}
	if err := directWriteFeatures.AddFontFeature(featureLiga); err != nil {
		return err
	}
	if err := directWriteFeatures.AddFontFeature(featureClig); err != nil {
		return err
	}
	return directWriteFeatures.AddFontFeature(featureCalt)
}

func makeDirectWriteFeature(featureName string, parameter uint32) dwrite.DWRITE_FONT_FEATURE {
	tag := makeDirectWriteTag(featureName)
	return dwrite.DWRITE_FONT_FEATURE{
		NameTag:   tag,
		Parameter: parameter,
	}
}

func makeOpenTypeTag(tagName string) uint32 {
	b := []byte(tagName)
	if gpuiutil.DebugAssertions && len(b) != 4 {
		panic("assertion failed: bytes.len() == 4")
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func makeDirectWriteTag(tagName string) uint32 {
	return makeOpenTypeTag(tagName)
}

func getName(s *dwrite.IDWriteLocalizedStrings, locale []uint16) (string, error) {
	defer s.Release()
	localeNameIndex, exists, err := s.FindLocaleName(locale)
	if err != nil {
		return "", err
	}
	if !exists {
		localeNameIndex, exists, err = s.FindLocaleName(defaultLocaleName)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("No localised string for %s", string(utf16.Decode(locale)))
		}
	}

	nameLength, err := s.GetStringLength(localeNameIndex)
	if err != nil {
		return "", err
	}
	nameVec := make([]uint16, nameLength+1)
	if err := s.GetString(localeNameIndex, nameVec); err != nil {
		return "", err
	}

	return string(utf16.Decode(nameVec[:nameLength])), nil
}

const (
	localeNameMaxLength       = 85
	feFontSmoothingClearType  = 2
	logfontFaceNameCharacters = 32
)

type logfontw struct {
	lfHeight         int32
	lfWidth          int32
	lfEscapement     int32
	lfOrientation    int32
	lfWeight         int32
	lfItalic         uint8
	lfUnderline      uint8
	lfStrikeOut      uint8
	lfCharSet        uint8
	lfOutPrecision   uint8
	lfClipPrecision  uint8
	lfQuality        uint8
	lfPitchAndFamily uint8
	lfFaceName       [logfontFaceNameCharacters]uint16
}

func getSystemSubpixelRendering() bool {
	var value uint32
	err := win32.SystemParametersInfoW(win32.SPI_GETFONTSMOOTHINGTYPE, 0, unsafe.Pointer(&value), 0)
	if err != nil {
		gpuiutil.LogErr(err)
		return true
	}
	return value == feFontSmoothingClearType
}

func getSystemUIFontName() gpui.SharedString {
	var info logfontw
	var fontFamily gpui.SharedString
	if err := win32.SystemParametersInfoW(win32.SPI_GETICONTITLELOGFONT, uint32(unsafe.Sizeof(info)), unsafe.Pointer(&info), 0); err != nil {
		gpuiutil.LogErr(err)
		fontFamily = "Segoe UI"
	} else {
		fontName := string(utf16.Decode(info.lfFaceName[:]))
		fontFamily = gpui.SharedString(strings.Trim(fontName, "\x00"))
	}
	slog.Info(fmt.Sprintf("Use %s as UI font.", fontFamily))
	return fontFamily
}

func isColorGlyph(fontFace *dwrite.IDWriteFontFace3, glyphID gpui.GlyphID, factory *dwrite.IDWriteFactory5) bool {
	glyphIndex := uint16(glyphID)
	glyphAdvance := float32(0.0)
	glyphOffset := dwrite.DWRITE_GLYPH_OFFSET{AdvanceOffset: 0.0, AscenderOffset: 0.0}
	glyphRun := dwrite.DWRITE_GLYPH_RUN{
		FontFace:      fontFace.AsFontFace(),
		FontEmSize:    14.0,
		GlyphCount:    1,
		GlyphIndices:  &glyphIndex,
		GlyphAdvances: &glyphAdvance,
		GlyphOffsets:  &glyphOffset,
		IsSideways:    0,
		BidiLevel:     0,
	}
	enumerator, err := factory.TranslateColorGlyphRun(
		dwrite.Vector2{},
		&glyphRun,
		nil,
		dwrite.DWRITE_GLYPH_IMAGE_FORMATS_COLR|
			dwrite.DWRITE_GLYPH_IMAGE_FORMATS_SVG|
			dwrite.DWRITE_GLYPH_IMAGE_FORMATS_PNG|
			dwrite.DWRITE_GLYPH_IMAGE_FORMATS_JPEG|
			dwrite.DWRITE_GLYPH_IMAGE_FORMATS_PREMULTIPLIED_B8G8R8A8,
		dwrite.DWRITE_MEASURING_MODE_NATURAL,
		nil,
		0,
	)
	if err != nil {
		return false
	}
	enumerator.Release()
	return true
}

var defaultLocaleName = dwrite.PCWSTR("en-US")

func isLineSeparator(character rune) bool {
	return character == ' ' || character == ' '
}
