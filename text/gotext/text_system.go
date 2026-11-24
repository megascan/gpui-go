package gotext

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"unicode/utf8"

	"github.com/go-text/typesetting/segmenter"
	gpui "github.com/megascan/gpui-go"
)

// CosmicTextSystem is the Linux and web PlatformTextSystem, built on go-text/typesetting.
type CosmicTextSystem struct {
	gpui.PlatformTextSystemDefaults
	mu    sync.RWMutex
	state cosmicTextSystemState
}

var _ gpui.PlatformTextSystem = (*CosmicTextSystem)(nil)

type fontKey struct {
	family    gpui.SharedString
	features  gpui.FontFeatures
	fallbacks gpui.Option[gpui.FontFallbacks]
}

func newFontKey(family gpui.SharedString, features gpui.FontFeatures, fallbacks gpui.Option[gpui.FontFallbacks]) fontKey {
	return fontKey{family: family, features: features, fallbacks: fallbacks}
}

type loadedFontKey struct {
	databaseID faceID
	font       fontKey
}

type fallbackEntry struct {
	fontID gpui.FontID
	family gpui.SharedString
}

type cosmicTextSystemState struct {
	fontSystem           *fontSystem
	loadedFonts          []loadedFont
	loadedFontIDsByKey   map[loadedFontKey]gpui.FontID
	fontIDsByFamilyCache map[fontKey][]gpui.FontID
	systemFontFallback   string
	missingGlyphSink     gpui.MissingGlyphSink
}

type loadedFont struct {
	font              *cosmicFont
	features          []fontFeature
	isKnownEmojiFont  bool
	userFallbackChain []fallbackEntry
}

type fontMatchProperties struct {
	primaryFamilyName gpui.SharedString
	stretch           uint16
	style             faceStyle
	weight            uint16
	features          []fontFeature
	fallbackChain     []fallbackEntry
}

func (p *fontMatchProperties) attributes(fontID gpui.FontID, familyName gpui.SharedString) attrs {
	return attrs{
		metadata: int(fontID),
		family:   string(familyName),
		stretch:  p.stretch,
		style:    p.style,
		weight:   p.weight,
		features: p.features,
	}
}

// FontWeightAndStyle returns the selected face's weight and style, which may differ from the request.
func (s *CosmicTextSystem) FontWeightAndStyle(fontID gpui.FontID) (gpui.FontWeight, gpui.FontStyle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if int(fontID) >= len(s.state.loadedFonts) {
		return 0, 0, errors.New("invalid font ID")
	}
	font := &s.state.loadedFonts[fontID]
	face, ok := s.state.fontSystem.db.face(font.font.id)
	if !ok {
		return 0, 0, errors.New("font face not found")
	}
	var style gpui.FontStyle
	switch face.style {
	case faceStyleNormal:
		style = gpui.FontStyleNormal
	case faceStyleItalic:
		style = gpui.FontStyleItalic
	case faceStyleOblique:
		style = gpui.FontStyleOblique
	}
	return gpui.FontWeight(float32(face.weight)), style, nil
}

// MissingGlyphs builds reports for unresolved source indices after an outer text system has
// applied its own fallback.
func (s *CosmicTextSystem) MissingGlyphs(text string, fontRuns []gpui.FontRun, missingTextIndices []int) []gpui.MissingGlyph {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.missingGlyphs(text, fontRuns, missingTextIndices)
}

func newCosmicTextSystem(fs *fontSystem, systemFontFallback string) *CosmicTextSystem {
	return &CosmicTextSystem{state: cosmicTextSystemState{
		fontSystem:           fs,
		loadedFontIDsByKey:   map[loadedFontKey]gpui.FontID{},
		fontIDsByFamilyCache: map[fontKey][]gpui.FontID{},
		systemFontFallback:   systemFontFallback,
	}}
}

// NewTextSystem creates a text system that loads the system fonts, mapping `.SystemUIFont`
// to systemFontFallback.
func NewTextSystem(systemFontFallback string) *CosmicTextSystem {
	fs := newFontSystem(systemLocale())
	fs.db.loadSystemFonts()
	return newCosmicTextSystem(fs, systemFontFallback)
}

// NewTextSystemWithoutSystemFonts creates a text system with an empty font database.
func NewTextSystemWithoutSystemFonts(systemFontFallback string) *CosmicTextSystem {
	return newCosmicTextSystem(newFontSystem("en-US"), systemFontFallback)
}

// AddFonts loads the font data into the font database.
func (s *CosmicTextSystem) AddFonts(fonts [][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.addFonts(fonts)
}

// SetMissingGlyphSink installs a sink for unresolved grapheme clusters.
func (s *CosmicTextSystem) SetMissingGlyphSink(sink gpui.MissingGlyphSink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.missingGlyphSink = sink
}

// AllFontNames returns the first family name of every face, sorted and deduplicated.
func (s *CosmicTextSystem) AllFontNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []string{}
	for _, face := range s.state.fontSystem.db.faces {
		if len(face.families) > 0 {
			result = append(result, face.families[0])
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

// FontID resolves a font descriptor to a loaded face.
func (s *CosmicTextSystem) FontID(font gpui.Font) (gpui.FontID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := &s.state
	key := newFontKey(font.Family, font.Features, font.Fallbacks)
	candidates, ok := state.fontIDsByFamilyCache[key]
	if !ok {
		fontIDs, err := state.loadFamily(string(font.Family), font.Features, font.Fallbacks)
		if err != nil {
			return 0, err
		}
		state.fontIDsByFamilyCache[key] = fontIDs
		candidates = fontIDs
	}
	ix, err := findBestMatch(font, candidates, state)
	if err != nil {
		return 0, err
	}
	return candidates[ix], nil
}

// PrewarmFonts warms the font match caches for the fonts and their fallback chains.
func (s *CosmicTextSystem) PrewarmFonts(fontIDs []gpui.FontID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.prewarmFonts(fontIDs)
}

// FontMetrics returns the font's metrics in font units.
func (s *CosmicTextSystem) FontMetrics(fontID gpui.FontID) gpui.FontMetrics {
	s.mu.RLock()
	metrics := s.state.loadedFont(fontID).font.swash
	s.mu.RUnlock()
	return gpui.FontMetrics{
		UnitsPerEm:         uint32(metrics.unitsPerEm),
		Ascent:             metrics.ascent,
		Descent:            -metrics.descent,
		LineGap:            metrics.leading,
		UnderlinePosition:  metrics.underlineOffset,
		UnderlineThickness: metrics.strokeSize,
		CapHeight:          metrics.capHeight,
		XHeight:            metrics.xHeight,
		BoundingBox: gpui.Bounds[float32]{
			Origin: gpui.Point[float32]{X: 0, Y: 0},
			Size:   gpui.Size[float32]{Width: metrics.maxWidth, Height: metrics.ascent + metrics.descent},
		},
	}
}

// TypographicBounds returns the glyph's advance box in font units.
func (s *CosmicTextSystem) TypographicBounds(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Bounds[float32], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	glyphMetrics := s.state.loadedFont(fontID).font.glyphs
	glyph := uint16(glyphID)
	return gpui.Bounds[float32]{
		Origin: gpui.Point[float32]{X: 0, Y: 0},
		Size:   gpui.Size[float32]{Width: glyphMetrics.advanceWidth(glyph), Height: glyphMetrics.advanceHeight(glyph)},
	}, nil
}

// Advance returns the glyph's advance in font units.
func (s *CosmicTextSystem) Advance(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Size[float32], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.advance(fontID, glyphID)
}

// GlyphForChar returns the font's glyph for the character.
func (s *CosmicTextSystem) GlyphForChar(fontID gpui.FontID, ch rune) (gpui.GlyphID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.glyphForChar(fontID, ch)
}

var errRasterizationUnimplemented = errors.New("gotext: glyph rasterization is ported by M2-gotext-raster")

// GlyphRasterBounds returns the raster bounds of a glyph.
func (s *CosmicTextSystem) GlyphRasterBounds(params gpui.RenderGlyphParams) (gpui.Bounds[gpui.DevicePixels], error) {
	return gpui.Bounds[gpui.DevicePixels]{}, errRasterizationUnimplemented
}

// RasterizeGlyph rasterizes a glyph into its raster bounds.
func (s *CosmicTextSystem) RasterizeGlyph(params gpui.RenderGlyphParams, rasterBounds gpui.Bounds[gpui.DevicePixels]) (gpui.Size[gpui.DevicePixels], []byte, error) {
	return gpui.Size[gpui.DevicePixels]{}, nil, errRasterizationUnimplemented
}

// LayoutLine shapes one line of text.
func (s *CosmicTextSystem) LayoutLine(text string, fontSize gpui.Pixels, runs []gpui.FontRun) gpui.LineLayout {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.layoutLine(text, fontSize, runs)
}

// RecommendedRenderingMode returns subpixel rendering.
func (s *CosmicTextSystem) RecommendedRenderingMode(fontID gpui.FontID, fontSize gpui.Pixels) gpui.TextRenderingMode {
	return gpui.TextRenderingModeSubpixel
}

func (s *cosmicTextSystemState) loadedFont(fontID gpui.FontID) *loadedFont {
	return &s.loadedFonts[fontID]
}

func (s *cosmicTextSystemState) fontMatchProperties(fontID gpui.FontID) (fontMatchProperties, bool) {
	loaded := s.loadedFont(fontID)
	face, ok := s.fontSystem.db.face(loaded.font.id)
	if !ok {
		slog.Warn(fmt.Sprintf("font face not found in database for font_id %v", fontID))
		return fontMatchProperties{}, false
	}
	if len(face.families) == 0 {
		slog.Warn(fmt.Sprintf("font face has no family names for font_id %v", fontID))
		return fontMatchProperties{}, false
	}
	return fontMatchProperties{
		primaryFamilyName: gpui.SharedString(face.families[0]),
		stretch:           face.stretch,
		style:             face.style,
		weight:            face.weight,
		features:          loaded.features,
		fallbackChain:     loaded.userFallbackChain,
	}, true
}

func (s *cosmicTextSystemState) prewarmFonts(fontIDs []gpui.FontID) {
	for _, fontID := range fontIDs {
		properties, ok := s.fontMatchProperties(fontID)
		if !ok {
			continue
		}
		primary := properties.attributes(fontID, properties.primaryFamilyName)
		s.fontSystem.getFontMatches(&primary)
		for _, fallback := range properties.fallbackChain {
			a := properties.attributes(fallback.fontID, fallback.family)
			s.fontSystem.getFontMatches(&a)
		}
	}
}

func (s *cosmicTextSystemState) addFonts(fonts [][]byte) error {
	clear(s.fontIDsByFamilyCache)
	db := s.fontSystem.dbMut()
	for _, data := range fonts {
		db.loadFontData(data)
	}
	return nil
}

func (s *cosmicTextSystemState) loadFamily(name string, features gpui.FontFeatures, fallbacks gpui.Option[gpui.FontFallbacks]) ([]gpui.FontID, error) {
	loadedKey := newFontKey(gpui.SharedString(name), features, fallbacks)

	var userFallbackChain []fallbackEntry
	if list, ok := fallbacks.Get(); ok && len(list.FallbackList()) > 0 {
		chain := []fallbackEntry{}
		for _, fallbackName := range list.FallbackList() {
			fbKey := newFontKey(gpui.SharedString(fallbackName), features, gpui.None[gpui.FontFallbacks]())
			fbIDs, cached := s.fontIDsByFamilyCache[fbKey]
			if !cached {
				loaded, err := s.loadFamily(fallbackName, features, gpui.None[gpui.FontFallbacks]())
				if err != nil {
					return nil, err
				}
				s.fontIDsByFamilyCache[fbKey] = loaded
				fbIDs = loaded
			}
			if len(fbIDs) == 0 {
				continue
			}
			fbID := fbIDs[0]
			dbID := s.loadedFonts[fbID].font.id
			if face, ok := s.fontSystem.db.face(dbID); ok && len(face.families) > 0 {
				chain = append(chain, fallbackEntry{fontID: fbID, family: gpui.SharedString(face.families[0])})
			}
		}
		userFallbackChain = chain
	}

	name = gpui.FontNameWithFallbacks(name, s.systemFontFallback)

	type familyFace struct {
		id             faceID
		postScriptName string
	}
	var families []familyFace
	for _, face := range s.fontSystem.db.faces {
		if face.hasFamily(name) {
			families = append(families, familyFace{face.id, face.postScriptName})
		}
	}

	cosmicFeatures, err := cosmicFontFeatures(features)
	if err != nil {
		return nil, err
	}

	var loadedFontIDs []gpui.FontID
	for _, family := range families {
		key := loadedFontKey{databaseID: family.id, font: loadedKey}
		if fontID, ok := s.loadedFontIDsByKey[key]; ok {
			s.loadedFonts[fontID].userFallbackChain = userFallbackChain
			loadedFontIDs = append(loadedFontIDs, fontID)
			continue
		}

		font, ok := s.fontSystem.getFont(family.id)
		if !ok {
			return nil, errors.New("Could not load font")
		}

		allowedBadFontNames := []string{"SegoeFluentIcons", "Segoe Fluent Icons"}
		if font.charmap('m') == 0 && !slices.Contains(allowedBadFontNames, family.postScriptName) {
			continue
		}

		fontID := gpui.FontID(len(s.loadedFonts))
		loadedFontIDs = append(loadedFontIDs, fontID)
		s.loadedFonts = append(s.loadedFonts, loadedFont{
			font:              font,
			features:          cosmicFeatures,
			isKnownEmojiFont:  checkIsKnownEmojiFont(family.postScriptName),
			userFallbackChain: userFallbackChain,
		})
		s.loadedFontIDsByKey[key] = fontID
	}
	return loadedFontIDs, nil
}

func (s *cosmicTextSystemState) advance(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Size[float32], error) {
	glyphMetrics := s.loadedFont(fontID).font.glyphs
	return gpui.Size[float32]{
		Width:  glyphMetrics.advanceWidth(uint16(glyphID)),
		Height: glyphMetrics.advanceHeight(uint16(glyphID)),
	}, nil
}

func (s *cosmicTextSystemState) glyphForChar(fontID gpui.FontID, ch rune) (gpui.GlyphID, bool) {
	glyphID := s.loadedFont(fontID).font.charmap(ch)
	if glyphID == 0 {
		return 0, false
	}
	return gpui.GlyphID(glyphID), true
}

func (s *cosmicTextSystemState) fontIDForCosmicID(id faceID) (gpui.FontID, error) {
	for ix := range s.loadedFonts {
		if s.loadedFonts[ix].font.id == id {
			return gpui.FontID(ix), nil
		}
	}
	font, ok := s.fontSystem.getFont(id)
	if !ok {
		return 0, errors.New("failed to get fallback font from cosmic-text font system")
	}
	face, ok := s.fontSystem.db.face(id)
	if !ok {
		return 0, errors.New("fallback font face not found in cosmic-text database")
	}
	fontID := gpui.FontID(len(s.loadedFonts))
	s.loadedFonts = append(s.loadedFonts, loadedFont{
		font:             font,
		isKnownEmojiFont: checkIsKnownEmojiFont(face.postScriptName),
	})
	return fontID, nil
}

func (s *cosmicTextSystemState) layoutLine(text string, fontSize gpui.Pixels, fontRuns []gpui.FontRun) gpui.LineLayout {
	if containsParagraphSeparator(text) {
		return s.layoutLineWithSeparators(text, fontSize, fontRuns)
	}
	return s.layoutLineNoSeparators(text, fontSize, fontRuns)
}

func (s *cosmicTextSystemState) layoutLineWithSeparators(text string, fontSize gpui.Pixels, fontRuns []gpui.FontRun) gpui.LineLayout {
	layout := gpui.LineLayout{FontSize: fontSize, Len: len(text)}
	paragraphStart := 0
	for separatorStart, separator := range text {
		if !isParagraphSeparator(separator) {
			continue
		}
		separatorEnd := separatorStart + utf8.RuneLen(separator)
		s.shapeSegment(text, paragraphStart, separatorStart, fontSize, fontRuns, &layout)
		s.shapeSegment(text, separatorStart, separatorEnd, fontSize, fontRuns, &layout)
		paragraphStart = separatorEnd
	}
	s.shapeSegment(text, paragraphStart, len(text), fontSize, fontRuns, &layout)
	return layout
}

func (s *cosmicTextSystemState) shapeSegment(text string, start, end int, fontSize gpui.Pixels, fontRuns []gpui.FontRun, layout *gpui.LineLayout) {
	if start >= end {
		return
	}
	segmentFontRuns := clipFontRuns(fontRuns, start, end)
	segment := s.layoutLineNoSeparators(text[start:end], fontSize, segmentFontRuns)

	segmentRuns := segment.Runs
	for r := range segmentRuns {
		for g := range segmentRuns[r].Glyphs {
			glyph := &segmentRuns[r].Glyphs[g]
			glyph.Index += start
			glyph.Position.X += layout.Width
		}
	}
	for _, run := range segmentRuns {
		if n := len(layout.Runs); n > 0 && layout.Runs[n-1].FontID == run.FontID {
			layout.Runs[n-1].Glyphs = append(layout.Runs[n-1].Glyphs, run.Glyphs...)
		} else {
			layout.Runs = append(layout.Runs, run)
		}
	}
	layout.Width += segment.Width
	layout.Ascent = max(layout.Ascent, segment.Ascent)
	layout.Descent = max(layout.Descent, segment.Descent)
}

func (s *cosmicTextSystemState) layoutLineNoSeparators(text string, fontSize gpui.Pixels, fontRuns []gpui.FontRun) gpui.LineLayout {
	list := attrsList{defaults: defaultAttrs()}
	offs := 0
	for _, run := range fontRuns {
		runEnd := offs + run.Len
		properties, ok := s.fontMatchProperties(run.FontID)
		if !ok {
			offs = runEnd
			continue
		}
		primaryAttrs := properties.attributes(run.FontID, properties.primaryFamilyName)
		fallbackAttrs := make([]attrs, len(properties.fallbackChain))
		for i, fallback := range properties.fallbackChain {
			fallbackAttrs[i] = properties.attributes(fallback.fontID, fallback.family)
		}
		var spans []runSpan
		if len(properties.fallbackChain) == 0 {
			spans = []runSpan{{start: offs, end: runEnd, slot: -1, fontID: run.FontID}}
		} else {
			covers := func(id gpui.FontID, ch rune) bool { return charmapCovers(s.loadedFonts, id, ch) }
			spans = computeRunSpans(text, offs, run.Len, run.FontID, properties.fallbackChain, covers)
		}
		for _, span := range spans {
			a := primaryAttrs
			if span.slot >= 0 {
				a = fallbackAttrs[span.slot]
			}
			list.addSpan(span.start, span.end, a)
		}
		offs = runEnd
	}

	line := newShapeLine(s.fontSystem, text, &list, 4)
	layout := line.layout(float32(fontSize))

	var missingGlyphs []gpui.MissingGlyph
	if s.missingGlyphSink != nil {
		var indices []int
		for _, glyph := range layout.glyphs {
			if glyph.glyphID == 0 {
				indices = append(indices, glyph.start)
			}
		}
		missingGlyphs = s.missingGlyphs(text, fontRuns, indices)
	}

	var runs []gpui.ShapedRun
	for _, glyph := range layout.glyphs {
		fontID := gpui.FontID(glyph.metadata)
		loaded := s.loadedFont(fontID)
		if loaded.font.id != glyph.fontID {
			resolved, err := s.fontIDForCosmicID(glyph.fontID)
			if err != nil {
				slog.Warn(fmt.Sprintf("failed to resolve cosmic font id %v: %v", glyph.fontID, err))
				continue
			}
			fontID = resolved
			loaded = s.loadedFont(fontID)
		}
		isEmoji := loaded.isKnownEmojiFont
		if glyph.glyphID == 3 && isEmoji {
			continue
		}
		shapedGlyph := gpui.ShapedGlyph{
			ID: gpui.GlyphID(glyph.glyphID),
			Position: gpui.Point[gpui.Pixels]{
				X: gpui.Pixels(glyph.x + float32(glyph.xOffset*float32(fontSize))),
				Y: gpui.Pixels(glyph.y - float32(glyph.yOffset*float32(fontSize))),
			},
			Index:   glyph.start,
			IsEmoji: isEmoji,
		}
		if n := len(runs); n > 0 && runs[n-1].FontID == fontID {
			runs[n-1].Glyphs = append(runs[n-1].Glyphs, shapedGlyph)
		} else {
			runs = append(runs, gpui.ShapedRun{FontID: fontID, Glyphs: []gpui.ShapedGlyph{shapedGlyph}})
		}
	}

	if s.missingGlyphSink != nil && missingGlyphs != nil {
		s.missingGlyphSink.Report(missingGlyphs)
	}

	return gpui.LineLayout{
		FontSize: fontSize,
		Width:    gpui.Pixels(layout.w),
		Ascent:   gpui.Pixels(layout.maxAscent),
		Descent:  gpui.Pixels(layout.maxDescent),
		Runs:     runs,
		Len:      len(text),
	}
}

func (s *cosmicTextSystemState) missingGlyphs(text string, fontRuns []gpui.FontRun, missingTextIndices []int) []gpui.MissingGlyph {
	if len(missingTextIndices) == 0 {
		return []gpui.MissingGlyph{}
	}
	indices := slices.Clone(missingTextIndices)
	slices.Sort(indices)
	indices = slices.Compact(indices)

	fontRunIndex := 0
	fontRunEnd := 0
	if len(fontRuns) > 0 {
		fontRunEnd = fontRuns[0].Len
	}
	missing := []gpui.MissingGlyph{}
	missingIndex := 0
	var seg segmenter.Segmenter
	if err := seg.InitWithString(text); err != nil {
		return missing
	}
	it := seg.GraphemeIterator()
	for it.Next() {
		g := it.Grapheme()
		graphemeStart := g.OffsetInBytes
		graphemeEnd := graphemeStart + g.LengthInBytes
		for missingIndex < len(indices) && indices[missingIndex] < graphemeStart {
			missingIndex++
		}
		if missingIndex >= len(indices) {
			break
		}
		textIndex := indices[missingIndex]
		if textIndex >= graphemeEnd {
			continue
		}
		for fontRunEnd <= textIndex && fontRunIndex+1 < len(fontRuns) {
			fontRunIndex++
			fontRunEnd += fontRuns[fontRunIndex].Len
		}
		var fontID gpui.Option[gpui.FontID]
		if fontRunIndex < len(fontRuns) {
			fontID = gpui.Some(fontRuns[fontRunIndex].FontID)
		} else if len(fontRuns) > 0 {
			fontID = gpui.Some(fontRuns[len(fontRuns)-1].FontID)
		}
		fontClass := s.fallbackFontClass(fontID)
		missing = append(missing, gpui.NewMissingGlyph(gpui.SharedString(text[graphemeStart:graphemeEnd]), fontClass))
		for missingIndex < len(indices) && indices[missingIndex] < graphemeEnd {
			missingIndex++
		}
	}
	return missing
}

func (s *cosmicTextSystemState) fallbackFontClass(fontID gpui.Option[gpui.FontID]) gpui.FallbackFontClass {
	id, ok := fontID.Get()
	if !ok {
		return gpui.FallbackFontClassProportional
	}
	loaded := s.loadedFont(id)
	if face, ok := s.fontSystem.db.face(loaded.font.id); ok && face.monospaced {
		return gpui.FallbackFontClassMonospace
	}
	return gpui.FallbackFontClassProportional
}

func isParagraphSeparator(character rune) bool {
	switch character {
	case '\u000a', '\u000d', '\u001c', '\u001d', '\u001e', '\u0085', ' ':
		return true
	}
	return false
}

func containsParagraphSeparator(text string) bool {
	ascii := true
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\n', '\r', 0x1c, 0x1d, 0x1e:
			return true
		}
		if text[i] >= utf8.RuneSelf {
			ascii = false
		}
	}
	if ascii {
		return false
	}
	for _, c := range text {
		if isParagraphSeparator(c) {
			return true
		}
	}
	return false
}

func clipFontRuns(fontRuns []gpui.FontRun, start, end int) []gpui.FontRun {
	clipped := []gpui.FontRun{}
	offs := 0
	for _, run := range fontRuns {
		runStart := offs
		offs += run.Len
		if offs <= start {
			continue
		}
		if runStart >= end {
			break
		}
		s := max(runStart, start)
		e := min(offs, end)
		if s < e {
			clipped = append(clipped, gpui.FontRun{Len: e - s, FontID: run.FontID})
		}
	}
	return clipped
}

func findBestMatch(font gpui.Font, candidates []gpui.FontID, state *cosmicTextSystemState) (int, error) {
	if len(candidates) == 0 {
		return 0, errors.New("requested font family contains no font matching the other parameters")
	}
	if len(candidates) == 1 {
		return 0, nil
	}
	targetWeight := float32(font.Weight)
	targetItalic := font.Style == gpui.FontStyleItalic || font.Style == gpui.FontStyleOblique
	bestIndex := 0
	bestScore := ^uint32(0)
	for index, fontID := range candidates {
		databaseID := state.loadedFont(fontID).font.id
		face, ok := state.fontSystem.db.face(databaseID)
		if !ok {
			return 0, errors.New("font face not found in database")
		}
		isItalic := face.style == faceStyleItalic || face.style == faceStyleOblique
		var stylePenalty uint32
		if isItalic != targetItalic {
			stylePenalty = 1000
		}
		diff := int32(face.weight) - int32(targetWeight)
		if diff < 0 {
			diff = -diff
		}
		score := stylePenalty + uint32(diff)
		if score < bestScore {
			bestScore = score
			bestIndex = index
		}
	}
	return bestIndex, nil
}

type runSpan struct {
	start, end int
	slot       int
	fontID     gpui.FontID
}

func computeRunSpans(text string, runOffset, runLen int, primary gpui.FontID, fallbackChain []fallbackEntry, covers func(gpui.FontID, rune) bool) []runSpan {
	spans := []runSpan{}
	runEnd := runOffset + runLen
	if runEnd <= runOffset {
		return spans
	}
	if len(fallbackChain) == 0 {
		return append(spans, runSpan{start: runOffset, end: runEnd, slot: -1, fontID: primary})
	}
	runText := text[runOffset:runEnd]
	spanStart := runOffset
	spanSlot := -1
	spanFontID := primary
	for _, graphemeIdx := range graphemeOffsets(runText) {
		abs := runOffset + graphemeIdx
		ch, _ := utf8.DecodeRuneInString(runText[graphemeIdx:])
		nextSlot := pickCoveringSlot(ch, spanSlot, primary, fallbackChain, covers)
		if nextSlot == spanSlot {
			continue
		}
		if abs > spanStart {
			spans = append(spans, runSpan{start: spanStart, end: abs, slot: spanSlot, fontID: spanFontID})
		}
		spanStart = abs
		spanSlot = nextSlot
		spanFontID = slotFontID(nextSlot, primary, fallbackChain)
	}
	if spanStart < runEnd {
		spans = append(spans, runSpan{start: spanStart, end: runEnd, slot: spanSlot, fontID: spanFontID})
	}
	return spans
}

func slotFontID(slot int, primary gpui.FontID, fallbackChain []fallbackEntry) gpui.FontID {
	if slot < 0 {
		return primary
	}
	return fallbackChain[slot].fontID
}

func pickCoveringSlot(ch rune, current int, primary gpui.FontID, fallbackChain []fallbackEntry, covers func(gpui.FontID, rune) bool) int {
	if ch <= 0x7F {
		return -1
	}
	if covers(primary, ch) {
		return -1
	}
	if covers(slotFontID(current, primary, fallbackChain), ch) {
		return current
	}
	for i, fallback := range fallbackChain {
		if covers(fallback.fontID, ch) {
			return i
		}
	}
	return -1
}

func charmapCovers(loadedFonts []loadedFont, id gpui.FontID, ch rune) bool {
	return int(id) < len(loadedFonts) && loadedFonts[id].font.charmap(ch) != 0
}

func cosmicFontFeatures(features gpui.FontFeatures) ([]fontFeature, error) {
	result := []fontFeature{}
	for _, feature := range features.TagValueList() {
		if len(feature.Tag) != 4 {
			return nil, errors.New("Incorrect feature flag format")
		}
		tag := [4]byte([]byte(feature.Tag))
		replaced := false
		for i := range result {
			if result[i].tag == tag {
				result[i].value = feature.Value
				replaced = true
			}
		}
		if !replaced {
			result = append(result, fontFeature{tag: tag, value: feature.Value})
		}
	}
	return result, nil
}

func checkIsKnownEmojiFont(postScriptName string) bool {
	return postScriptName == "NotoColorEmoji"
}
