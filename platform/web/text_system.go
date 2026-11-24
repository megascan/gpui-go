//go:build js && wasm

package web

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/bits"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/text/gotext"
)

const (
	canvasFontBit   gpui.FontID = 1 << (bits.UintSize - 1)
	maxMeasurements             = 4096
)

type webTextSystem struct {
	native             *gotext.CosmicTextSystem
	mu                 sync.RWMutex
	state              webTextSystemState
	canvasFontFallback CanvasFontFallback
}

type measurementKey struct {
	fontID   gpui.FontID
	glyphID  gpui.GlyphID
	fontSize gpui.Pixels
}

type webTextSystemState struct {
	missingGlyphSink gpui.MissingGlyphSink
	descriptors      map[gpui.FontID]gpui.Font
	canvasFonts      []*canvasFont
	canvasFontIDs    map[gpui.FontID]gpui.FontID
	glyphs           glyphCache
	measurements     map[measurementKey]canvasTextMetrics
}

type canvasFont struct {
	nativeID   gpui.FontID
	descriptor gpui.Font
	monospace  bool
}

func (s *webTextSystemState) cachedFallback(nativeID gpui.FontID, text string, color bool, fontSize gpui.Pixels) (gpui.FontID, gpui.GlyphID, canvasTextMetrics, bool) {
	fontID, ok := s.canvasFontIDs[nativeID]
	if !ok {
		return 0, 0, canvasTextMetrics{}, false
	}
	glyphID, ok := s.glyphs.idFor(fontID, text, color)
	if !ok {
		return 0, 0, canvasTextMetrics{}, false
	}
	metrics, ok := s.measurements[measurementKey{fontID, glyphID, fontSize}]
	if !ok {
		return 0, 0, canvasTextMetrics{}, false
	}
	return fontID, glyphID, metrics, true
}

func newWebTextSystem(systemFontFallback string, canvasFontFallback CanvasFontFallback) gpui.PlatformTextSystem {
	return &webTextSystem{
		native: gotext.NewTextSystemWithoutSystemFonts(systemFontFallback),
		state: webTextSystemState{
			descriptors:   map[gpui.FontID]gpui.Font{},
			canvasFontIDs: map[gpui.FontID]gpui.FontID{},
			measurements:  map[measurementKey]canvasTextMetrics{},
		},
		canvasFontFallback: canvasFontFallback,
	}
}

func (t *webTextSystem) canvasFont(fontID gpui.FontID) (*canvasFont, bool) {
	if fontID&canvasFontBit == 0 {
		return nil, false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	index := uint64(fontID &^ canvasFontBit)
	if index >= uint64(len(t.state.canvasFonts)) {
		return nil, false
	}
	return t.state.canvasFonts[index], true
}

func (t *webTextSystem) nativeFontID(fontID gpui.FontID) gpui.FontID {
	if font, ok := t.canvasFont(fontID); ok {
		return font.nativeID
	}
	return fontID
}

func (t *webTextSystem) canvasFontID(nativeID gpui.FontID) (gpui.FontID, error) {
	t.mu.RLock()
	if fontID, ok := t.state.canvasFontIDs[nativeID]; ok {
		t.mu.RUnlock()
		return fontID, nil
	}
	descriptor, ok := t.state.descriptors[nativeID]
	t.mu.RUnlock()
	if !ok {
		return 0, errors.New("missing source font descriptor for Canvas fallback")
	}
	monospace := false
	narrow, narrowOK := t.native.GlyphForChar(nativeID, 'i')
	wide, wideOK := t.native.GlyphForChar(nativeID, 'm')
	if narrowOK && wideOK {
		narrowAdvance, err := t.native.Advance(nativeID, narrow)
		if err != nil {
			return 0, err
		}
		wideAdvance, err := t.native.Advance(nativeID, wide)
		if err != nil {
			return 0, err
		}
		monospace = narrowAdvance.Width == wideAdvance.Width
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if fontID, ok := t.state.canvasFontIDs[nativeID]; ok {
		return fontID, nil
	}
	if gpui.FontID(len(t.state.canvasFonts)) >= canvasFontBit {
		return 0, errors.New("Canvas font ID space exhausted")
	}
	fontID := canvasFontBit | gpui.FontID(len(t.state.canvasFonts))
	t.state.canvasFonts = append(t.state.canvasFonts, &canvasFont{
		nativeID:   nativeID,
		descriptor: descriptor,
		monospace:  monospace,
	})
	t.state.canvasFontIDs[nativeID] = fontID
	return fontID, nil
}

func (t *webTextSystem) registerGlyph(fontID gpui.FontID, text string, color bool) (gpui.GlyphID, error) {
	t.mu.RLock()
	glyphID, ok := t.state.glyphs.idFor(fontID, text, color)
	t.mu.RUnlock()
	if ok {
		return glyphID, nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state.glyphs.getOrInsert(fontID, text, color)
}

func (t *webTextSystem) canvasGlyph(fontID gpui.FontID, glyphID gpui.GlyphID) (canvasGlyph, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	glyph, ok := t.state.glyphs.glyph(fontID, glyphID)
	if !ok {
		return canvasGlyph{}, errors.New("invalid Canvas glyph ID")
	}
	return *glyph, nil
}

func (t *webTextSystem) measure(fontID gpui.FontID, glyphID gpui.GlyphID, fontSize gpui.Pixels) (canvasTextMetrics, error) {
	key := measurementKey{fontID, glyphID, fontSize}
	t.mu.RLock()
	metrics, ok := t.state.measurements[key]
	t.mu.RUnlock()
	if ok {
		return metrics, nil
	}
	font, ok := t.canvasFont(fontID)
	if !ok {
		return canvasTextMetrics{}, errors.New("invalid Canvas font ID")
	}
	glyph, err := t.canvasGlyph(fontID, glyphID)
	if err != nil {
		return canvasTextMetrics{}, err
	}
	cssFont, err := font.cssFont(fontSize, glyph.color)
	if err != nil {
		return canvasTextMetrics{}, err
	}
	metrics, err = canvasMeasure(glyph.text, cssFont)
	if err != nil {
		return canvasTextMetrics{}, err
	}
	if !(metrics.advance >= 0.0) {
		return canvasTextMetrics{}, errors.New("Canvas returned a negative advance")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.state.measurements) >= maxMeasurements {
		clear(t.state.measurements)
	}
	t.state.measurements[key] = metrics
	return metrics, nil
}

func (t *webTextSystem) fallbackGlyph(nativeID gpui.FontID, text string, color bool, fontSize gpui.Pixels) (gpui.FontID, gpui.GlyphID, canvasTextMetrics, error) {
	t.mu.RLock()
	fontID, glyphID, metrics, ok := t.state.cachedFallback(nativeID, text, color, fontSize)
	t.mu.RUnlock()
	if ok {
		return fontID, glyphID, metrics, nil
	}
	fontID, err := t.canvasFontID(nativeID)
	if err != nil {
		return 0, 0, canvasTextMetrics{}, err
	}
	glyphID, err = t.registerGlyph(fontID, text, color)
	if err != nil {
		return 0, 0, canvasTextMetrics{}, err
	}
	metrics, err = t.measure(fontID, glyphID, fontSize)
	if err != nil {
		return 0, 0, canvasTextMetrics{}, err
	}
	return fontID, glyphID, metrics, nil
}

func (t *webTextSystem) applyFallback(text string, fontRuns []gpui.FontRun, layout *gpui.LineLayout) {
	fontSize := float64(layout.FontSize)
	if t.canvasFontFallback == CanvasFontFallbackDisabled ||
		isASCII(text) ||
		math.IsNaN(fontSize) || math.IsInf(fontSize, 0) ||
		layout.FontSize <= 0 {
		return
	}
	var replacements []replacement
	for _, candidate := range collectCandidates(text, fontRuns, layout) {
		if !t.canvasFontFallback.allows(candidate.isEmoji) {
			continue
		}
		glyphs := candidate.glyphs
		if glyphs == nil {
			continue
		}
		if !glyphs.contiguous || !(glyphs.missing || (candidate.color && !glyphs.nativeColor)) {
			continue
		}
		oldWidth := glyphs.endX - glyphs.position.X
		if oldWidth < 0 {
			continue
		}
		repl, err := func() (replacement, error) {
			if candidate.source.Start > candidate.source.End || candidate.source.End > len(text) ||
				!utf8.RuneStart(byteAt(text, candidate.source.Start)) || !utf8.RuneStart(byteAt(text, candidate.source.End)) {
				return replacement{}, errors.New("invalid Canvas fallback source range")
			}
			source := text[candidate.source.Start:candidate.source.End]
			fontID, glyphID, metrics, err := t.fallbackGlyph(candidate.fontID, source, candidate.color, layout.FontSize)
			if err != nil {
				return replacement{}, err
			}
			return replacement{
				glyphRange: glyphs.glyphRange,
				fontID:     fontID,
				glyph: gpui.ShapedGlyph{
					ID:       glyphID,
					Position: glyphs.position,
					Index:    candidate.source.Start,
					IsEmoji:  candidate.color,
				},
				widthDelta: gpui.Px(metrics.advance) - oldWidth,
			}, nil
		}()
		if err != nil {
			slog.Warn(fmt.Sprintf("Canvas font fallback failed: %v", err))
			continue
		}
		replacements = append(replacements, repl)
	}
	if len(replacements) == 0 {
		return
	}
	slices.SortFunc(replacements, func(a, b replacement) int { return a.glyphRange.Start - b.glyphRange.Start })
	applyReplacements(layout, replacements)
}

func byteAt(text string, index int) byte {
	if index >= len(text) {
		return 0
	}
	return text[index]
}

// SetMissingGlyphSink stores the sink locally so the native system cannot report glyphs Canvas
// may still replace.
func (t *webTextSystem) SetMissingGlyphSink(sink gpui.MissingGlyphSink) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.missingGlyphSink = sink
}

// AddFonts loads the font data into the native font database.
func (t *webTextSystem) AddFonts(fonts [][]byte) error {
	if err := t.native.AddFonts(fonts); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	clear(t.state.measurements)
	return nil
}

// AllFontNames returns the loaded font family names.
func (t *webTextSystem) AllFontNames() []string {
	return t.native.AllFontNames()
}

// FontID resolves descriptor to a native font id and records its descriptor for Canvas fallback.
func (t *webTextSystem) FontID(font gpui.Font) (gpui.FontID, error) {
	fontID, err := t.native.FontID(font)
	if err != nil {
		return 0, err
	}
	if fontID&canvasFontBit != 0 {
		return 0, errors.New("native font ID space exhausted")
	}
	descriptor := font
	weight, style, err := t.native.FontWeightAndStyle(fontID)
	if err != nil {
		return 0, err
	}
	descriptor.Weight, descriptor.Style = weight, style
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.state.descriptors[fontID]; !ok {
		t.state.descriptors[fontID] = descriptor
	}
	return fontID, nil
}

// PrewarmFonts prewarms the native fonts behind fontIDs.
func (t *webTextSystem) PrewarmFonts(fontIDs []gpui.FontID) {
	if !slices.ContainsFunc(fontIDs, func(id gpui.FontID) bool { return id&canvasFontBit != 0 }) {
		t.native.PrewarmFonts(fontIDs)
		return
	}
	nativeIDs := make([]gpui.FontID, len(fontIDs))
	for i, fontID := range fontIDs {
		nativeIDs[i] = t.nativeFontID(fontID)
	}
	t.native.PrewarmFonts(nativeIDs)
}

// FontMetrics returns the metrics of the native font behind fontID.
func (t *webTextSystem) FontMetrics(fontID gpui.FontID) gpui.FontMetrics {
	return t.native.FontMetrics(t.nativeFontID(fontID))
}

// TypographicBounds returns a glyph's bounds in font units.
func (t *webTextSystem) TypographicBounds(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Bounds[float32], error) {
	font, ok := t.canvasFont(fontID)
	if !ok {
		return t.native.TypographicBounds(fontID, glyphID)
	}
	fontSize := gpui.Px(float32(t.native.FontMetrics(font.nativeID).UnitsPerEm))
	metrics, err := t.measure(fontID, glyphID, fontSize)
	if err != nil {
		return gpui.Bounds[float32]{}, err
	}
	return gpui.Bounds[float32]{
		Origin: gpui.NewPoint(metrics.left, -metrics.ascent),
		Size:   gpui.NewSize(metrics.right-metrics.left, metrics.ascent+metrics.descent),
	}, nil
}

// Advance returns a glyph's advance in font units.
func (t *webTextSystem) Advance(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Size[float32], error) {
	font, ok := t.canvasFont(fontID)
	if !ok {
		return t.native.Advance(fontID, glyphID)
	}
	fontSize := gpui.Px(float32(t.native.FontMetrics(font.nativeID).UnitsPerEm))
	metrics, err := t.measure(fontID, glyphID, fontSize)
	if err != nil {
		return gpui.Size[float32]{}, err
	}
	return gpui.NewSize(metrics.advance, 0), nil
}

// GlyphForChar returns the glyph for ch, registering a Canvas glyph for Canvas fonts.
func (t *webTextSystem) GlyphForChar(fontID gpui.FontID, ch rune) (gpui.GlyphID, bool) {
	if _, ok := t.canvasFont(fontID); !ok {
		return t.native.GlyphForChar(fontID, ch)
	}
	text := string(ch)
	fallback, ok := ClassifyCanvasFallback(text)
	color := ok && fallback.EmojiPresentation
	glyphID, err := t.registerGlyph(fontID, text, color)
	if err != nil {
		slog.Warn(fmt.Sprintf("registering Canvas glyph failed: %v", err))
		return 0, false
	}
	return glyphID, true
}

// GlyphRasterBounds returns the device-pixel bounds a glyph rasterizes into.
func (t *webTextSystem) GlyphRasterBounds(params gpui.RenderGlyphParams) (gpui.Bounds[gpui.DevicePixels], error) {
	if _, ok := t.canvasFont(params.FontID); !ok {
		return t.native.GlyphRasterBounds(params)
	}
	metrics, err := t.measure(params.FontID, params.GlyphID, params.FontSize*gpui.Pixels(params.ScaleFactor))
	if err != nil {
		return gpui.Bounds[gpui.DevicePixels]{}, err
	}
	if metrics.right <= metrics.left || metrics.ascent+metrics.descent <= 0 {
		return gpui.Bounds[gpui.DevicePixels]{}, nil
	}
	offsetX, offsetY := subpixelOffset(params)
	left := math.Floor(float64(metrics.left)+float64(offsetX)) - 1
	top := math.Floor(-float64(metrics.ascent)+float64(offsetY)) - 1
	right := math.Ceil(float64(metrics.right)+float64(offsetX)) + 1
	bottom := math.Ceil(float64(metrics.descent)+float64(offsetY)) + 1
	for _, v := range []float64{left, top, right, bottom} {
		if !(v >= math.MinInt32 && v <= math.MaxInt32) {
			return gpui.Bounds[gpui.DevicePixels]{}, errors.New("Canvas text bounds exceed device coordinates")
		}
	}
	width := int64(int32(right)) - int64(int32(left))
	if width < math.MinInt32 || width > math.MaxInt32 {
		return gpui.Bounds[gpui.DevicePixels]{}, errors.New("Canvas text width overflow")
	}
	height := int64(int32(bottom)) - int64(int32(top))
	if height < math.MinInt32 || height > math.MaxInt32 {
		return gpui.Bounds[gpui.DevicePixels]{}, errors.New("Canvas text height overflow")
	}
	return gpui.Bounds[gpui.DevicePixels]{
		Origin: gpui.NewPoint(gpui.DevicePixels(int32(left)), gpui.DevicePixels(int32(top))),
		Size:   gpui.NewSize(gpui.DevicePixels(width), gpui.DevicePixels(height)),
	}, nil
}

// RasterizeGlyph rasterizes a glyph into bounds.
func (t *webTextSystem) RasterizeGlyph(params gpui.RenderGlyphParams, bounds gpui.Bounds[gpui.DevicePixels]) (gpui.Size[gpui.DevicePixels], []byte, error) {
	font, ok := t.canvasFont(params.FontID)
	if !ok {
		return t.native.RasterizeGlyph(params, bounds)
	}
	glyph, err := t.canvasGlyph(params.FontID, params.GlyphID)
	if err != nil {
		return gpui.Size[gpui.DevicePixels]{}, nil, err
	}
	cssFont, err := font.cssFont(params.FontSize*gpui.Pixels(params.ScaleFactor), glyph.color)
	if err != nil {
		return gpui.Size[gpui.DevicePixels]{}, nil, err
	}
	offsetX, offsetY := subpixelOffset(params)
	pixels, err := canvasRasterize(glyph.text, cssFont, bounds, [2]float32{offsetX, offsetY}, glyph.color)
	if err != nil {
		return gpui.Size[gpui.DevicePixels]{}, nil, err
	}
	if params.SubpixelRendering && !glyph.color {
		expanded := make([]byte, 0, len(pixels)*4)
		for _, alpha := range pixels {
			expanded = append(expanded, alpha, alpha, alpha, alpha)
		}
		pixels = expanded
	}
	return bounds.Size, pixels, nil
}

// LayoutLine shapes text natively, then replaces eligible graphemes with Canvas glyphs.
func (t *webTextSystem) LayoutLine(text string, fontSize gpui.Pixels, runs []gpui.FontRun) gpui.LineLayout {
	nativeRuns := runs
	if slices.ContainsFunc(runs, func(run gpui.FontRun) bool { return run.FontID&canvasFontBit != 0 }) {
		nativeRuns = make([]gpui.FontRun, len(runs))
		for i, run := range runs {
			nativeRuns[i] = gpui.FontRun{Len: run.Len, FontID: t.nativeFontID(run.FontID)}
		}
	}
	layout := t.native.LayoutLine(text, fontSize, nativeRuns)
	t.applyFallback(text, nativeRuns, &layout)
	t.mu.RLock()
	sink := t.state.missingGlyphSink
	t.mu.RUnlock()
	if sink != nil {
		var missingIndices []int
		for _, run := range layout.Runs {
			if run.FontID&canvasFontBit != 0 {
				continue
			}
			for _, glyph := range run.Glyphs {
				if glyph.ID == 0 {
					missingIndices = append(missingIndices, glyph.Index)
				}
			}
		}
		sink.Report(t.native.MissingGlyphs(text, nativeRuns, missingIndices))
	}
	return layout
}

// RecommendedRenderingMode returns grayscale for Canvas fonts and the native choice otherwise.
func (t *webTextSystem) RecommendedRenderingMode(fontID gpui.FontID, fontSize gpui.Pixels) gpui.TextRenderingMode {
	if _, ok := t.canvasFont(fontID); ok {
		return gpui.TextRenderingModeGrayscale
	}
	return t.native.RecommendedRenderingMode(fontID, fontSize)
}

// GlyphDilationForColor returns the native dilation for color.
func (t *webTextSystem) GlyphDilationForColor(color gpui.Hsla) uint8 {
	return t.native.GlyphDilationForColor(color)
}

func (f *canvasFont) cssFont(fontSize gpui.Pixels, color bool) (string, error) {
	size := float32(fontSize)
	if !isFiniteF32(size) || !(size > 0) {
		return "", errors.New("invalid Canvas font size")
	}
	var style string
	switch f.descriptor.Style {
	case gpui.FontStyleItalic:
		style = "italic"
	case gpui.FontStyleOblique:
		style = "oblique"
	default:
		style = "normal"
	}
	var families []string
	family := string(f.descriptor.Family)
	if !color && !strings.HasPrefix(family, ".") {
		families = append(families, quoteFamily(family))
	}
	if fallbacks, ok := f.descriptor.Fallbacks.Get(); ok {
		for _, name := range fallbacks.FallbackList() {
			families = append(families, quoteFamily(name))
		}
	}
	switch {
	case color:
		families = append(families, "emoji")
	case f.monospace:
		families = append(families, "monospace")
	default:
		families = append(families, "sans-serif")
	}
	return style + " " + formatF32(float32(f.descriptor.Weight)) + " " + formatF32(size) + "px " + strings.Join(families, ", "), nil
}

func quoteFamily(family string) string {
	var quoted strings.Builder
	quoted.WriteByte('"')
	for _, character := range family {
		switch {
		case character == '\\' || character == '"':
			quoted.WriteByte('\\')
			quoted.WriteRune(character)
		case unicode.IsControl(character):
			quoted.WriteByte(' ')
		default:
			quoted.WriteRune(character)
		}
	}
	quoted.WriteByte('"')
	return quoted.String()
}

func subpixelOffset(params gpui.RenderGlyphParams) (float32, float32) {
	return float32(params.SubpixelVariant.X) / float32(gpui.SubpixelVariantsX),
		float32(params.SubpixelVariant.Y) / float32(gpui.SubpixelVariantsY)
}
