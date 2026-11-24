//go:build darwin

package apple

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"sync"
	"unicode/utf16"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/cg"
	"github.com/megascan/gpui-go/internal/apple/ct"
)

// AppleTextSystem is the Apple text system using CoreText for font shaping.
type AppleTextSystem struct {
	gpui.PlatformTextSystemDefaults
	mu    sync.RWMutex
	state appleTextSystemState
}

type fontKey struct {
	fontFamily    gpui.SharedString
	fontFeatures  gpui.FontFeatures
	fontFallbacks gpui.Option[gpui.FontFallbacks]
}

type appleTextSystemState struct {
	memorySource            memSource
	systemSource            systemSource
	fonts                   []*fontKitFont
	fontSelections          map[gpui.Font]gpui.FontID
	fontIDsByPostscriptName map[string]gpui.FontID
	fontIDsByFontKey        map[fontKey][]gpui.FontID
	postscriptNamesByFontID map[gpui.FontID]string
}

var _ gpui.PlatformTextSystem = (*AppleTextSystem)(nil)

// NewAppleTextSystem creates a new AppleTextSystem.
func NewAppleTextSystem() *AppleTextSystem {
	if err := ct.Load(); err != nil {
		panic(err)
	}
	return &AppleTextSystem{state: appleTextSystemState{
		fontSelections:          map[gpui.Font]gpui.FontID{},
		fontIDsByPostscriptName: map[string]gpui.FontID{},
		fontIDsByFontKey:        map[fontKey][]gpui.FontID{},
		postscriptNamesByFontID: map[gpui.FontID]string{},
	}}
}

// AddFonts adds in-memory fonts.
func (s *AppleTextSystem) AddFonts(fonts [][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.addFonts(fonts)
}

// AllFontNames returns the family names of every system and in-memory font.
func (s *AppleTextSystem) AllFontNames() []string {
	names := []string{}
	collection := ct.FontCollectionCreateForAllFamilies()
	if collection == nil {
		return names
	}
	defer collection.Release()
	descriptors := collection.Ref().CreateMatchingFontDescriptors()
	if descriptors == nil {
		return names
	}
	defer descriptors.Release()
	arr := descriptors.Ref()
	for i := range arr.Count() {
		if name, ok := FamilyName(ct.FontDescriptor(arr.At(i))); ok {
			names = append(names, name)
		}
	}
	s.mu.RLock()
	names = append(names, s.state.memorySource.allFamilies()...)
	s.mu.RUnlock()
	return names
}

// FontID returns the id of the best match for the font, loading its family on first use.
func (s *AppleTextSystem) FontID(font gpui.Font) (gpui.FontID, error) {
	s.mu.RLock()
	if fontID, ok := s.state.fontSelections[font]; ok {
		s.mu.RUnlock()
		return fontID, nil
	}
	s.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	lock := &s.state
	if fontID, ok := lock.fontSelections[font]; ok {
		return fontID, nil
	}
	key := fontKey{fontFamily: font.Family, fontFeatures: font.Features, fontFallbacks: font.Fallbacks}
	candidates, ok := lock.fontIDsByFontKey[key]
	if !ok {
		fontIDs, err := lock.loadFamily(string(font.Family), font.Features, font.Fallbacks)
		if err != nil {
			return 0, err
		}
		lock.fontIDsByFontKey[key] = fontIDs
		candidates = fontIDs
	}

	candidateProperties := make([]fontKitProperties, len(candidates))
	for i, fontID := range candidates {
		candidateProperties[i] = lock.fonts[fontID].properties()
	}

	ix, err := findBestMatch(candidateProperties, fontKitProperties{
		style:   fontkitStyle(font.Style),
		weight:  fontkitWeight(font.Weight),
		stretch: stretchNormal,
	})
	if err != nil {
		return 0, err
	}

	fontID := candidates[ix]
	lock.fontSelections[font] = fontID
	return fontID, nil
}

// FontMetrics returns the font's metrics in font units.
func (s *AppleTextSystem) FontMetrics(fontID gpui.FontID) gpui.FontMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fontKitMetricsToMetrics(s.state.fonts[fontID].metrics())
}

// TypographicBounds returns the glyph's bounds in font units.
func (s *AppleTextSystem) TypographicBounds(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Bounds[float32], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, err := s.state.fonts[fontID].typographicBounds(uint32(glyphID))
	if err != nil {
		return gpui.Bounds[float32]{}, err
	}
	return boundsFromRect(r), nil
}

// Advance returns the glyph's advance in font units.
func (s *AppleTextSystem) Advance(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Size[float32], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.advance(fontID, glyphID)
}

// GlyphForChar returns the font's glyph for the character.
func (s *AppleTextSystem) GlyphForChar(fontID gpui.FontID, ch rune) (gpui.GlyphID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.glyphForChar(fontID, ch)
}

// GlyphRasterBounds returns the device-pixel bounds a rasterised glyph covers.
func (s *AppleTextSystem) GlyphRasterBounds(params gpui.RenderGlyphParams) (gpui.Bounds[gpui.DevicePixels], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.rasterBounds(params)
}

// RasterizeGlyph rasterises the glyph into an alpha mask, or BGRA for emoji.
func (s *AppleTextSystem) RasterizeGlyph(params gpui.RenderGlyphParams, rasterBounds gpui.Bounds[gpui.DevicePixels]) (gpui.Size[gpui.DevicePixels], []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.rasterizeGlyph(params, rasterBounds)
}

// LayoutLine shapes the text with CoreText.
func (s *AppleTextSystem) LayoutLine(text string, fontSize gpui.Pixels, fontRuns []gpui.FontRun) gpui.LineLayout {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.layoutLine(text, fontSize, fontRuns)
}

// RecommendedRenderingMode returns grayscale.
func (s *AppleTextSystem) RecommendedRenderingMode(fontID gpui.FontID, fontSize gpui.Pixels) gpui.TextRenderingMode {
	return gpui.TextRenderingModeGrayscale
}

// GlyphDilationForColor returns the CoreGraphics font-smoothing dilation level for the colour on
// macOS, and 0 elsewhere.
func (s *AppleTextSystem) GlyphDilationForColor(color gpui.Hsla) uint8 {
	if runtime.GOOS != "darwin" {
		return 0
	}
	if !fontSmoothingAllowedByUser() {
		return 0
	}
	rgba := color.ToRgb()
	luminance := 0.2126*rgba.R + 0.7152*rgba.G + 0.0722*rgba.B
	level := int32(math.Floor(float64(4.0*luminance + 0.5)))
	return uint8(min(max(level, 0), 4))
}

var fontSmoothingAllowedByUser = sync.OnceValue(func() bool {
	key := cf.StringCreate("AppleFontSmoothing")
	defer key.Release()
	value := ct.PreferencesCopyAppValue(key.Ref())
	if value == nil {
		return true
	}
	defer value.Release()
	number, ok := ct.AsNumber(value.Ref())
	if !ok {
		return true
	}
	v, ok := number.Int64()
	return !ok || v != 0
})

func (st *appleTextSystemState) addFonts(fonts [][]byte) error {
	handles := make([]fontHandle, len(fonts))
	for i, bytes := range fonts {
		handles[i] = fontHandle{bytes: bytes}
	}
	return st.memorySource.addFonts(handles)
}

func (st *appleTextSystemState) loadFamily(name string, features gpui.FontFeatures, fallbacks gpui.Option[gpui.FontFallbacks]) ([]gpui.FontID, error) {
	name = gpui.FontNameWithFallbacks(name, ".AppleSystemUIFont")

	var fontIDs []gpui.FontID
	postscriptNamesSeen := map[string]struct{}{}
	family, err := st.memorySource.selectFamilyByName(name)
	if err != nil {
		family, err = st.systemSource.selectFamilyByName(name)
		if err != nil {
			return nil, err
		}
	}
	defer family.release()
	for _, handle := range family.fonts {
		font, err := handle.load()
		if err != nil {
			return nil, err
		}

		if err := ApplyFeaturesAndFallbacks(&font.native, features, fallbacks); err != nil {
			font.release()
			return nil, err
		}
		{
			_, hasMGlyph := font.glyphForChar('m')
			isSegoeFluentIcons := font.fullName() == "Segoe Fluent Icons"
			if !hasMGlyph && !isSegoeFluentIcons {
				slog.Warn(fmt.Sprintf("font '%s' has no 'm' character and was not loaded", font.fullName()))
				font.release()
				continue
			}
		}

		traits := font.allTraits()
		_, okSymbolic := traits.number(ct.FontSymbolicTrait)
		_, okWidth := traits.number(ct.FontWidthTrait)
		_, okWeight := traits.number(ct.FontWeightTrait)
		_, okSlant := traits.number(ct.FontSlantTrait)
		traits.release()
		if !(okSymbolic && okWidth && okWeight && okSlant) {
			psName, _ := font.postscriptName()
			slog.Error(fmt.Sprintf("Failed to read traits for font %q (PostScript name %q)", font.fullName(), psName))
			font.release()
			continue
		}

		postscriptName, ok := font.postscriptName()
		if !ok {
			slog.Warn(fmt.Sprintf("font %q in family %q has no PostScript name; skipping", font.fullName(), name))
			font.release()
			continue
		}
		if _, seen := postscriptNamesSeen[postscriptName]; seen {
			slog.Warn(fmt.Sprintf("skipping duplicate font %q with PostScript name %q in family %q", font.fullName(), postscriptName, name))
			font.release()
			continue
		}
		postscriptNamesSeen[postscriptName] = struct{}{}
		fontID := gpui.FontID(len(st.fonts))
		fontIDs = append(fontIDs, fontID)
		st.fontIDsByPostscriptName[postscriptName] = fontID
		st.postscriptNamesByFontID[fontID] = postscriptName
		st.fonts = append(st.fonts, font)
	}
	return fontIDs, nil
}

func (st *appleTextSystemState) advance(fontID gpui.FontID, glyphID gpui.GlyphID) (gpui.Size[float32], error) {
	x, y, err := st.fonts[fontID].advance(uint32(glyphID))
	if err != nil {
		return gpui.Size[float32]{}, err
	}
	return gpui.NewSize(x, y), nil
}

func (st *appleTextSystemState) glyphForChar(fontID gpui.FontID, ch rune) (gpui.GlyphID, bool) {
	id, ok := st.fonts[fontID].glyphForChar(ch)
	return gpui.GlyphID(id), ok
}

func (st *appleTextSystemState) idForNativeFont(requestedFont ct.Font) gpui.FontID {
	postscriptName := requestedFont.PostScriptName()
	if fontID, ok := st.fontIDsByPostscriptName[postscriptName]; ok {
		return fontID
	}
	fontID := gpui.FontID(len(st.fonts))
	st.fontIDsByPostscriptName[postscriptName] = fontID
	st.postscriptNamesByFontID[fontID] = postscriptName
	cgFont := requestedFont.CopyGraphicsFont()
	st.fonts = append(st.fonts, fontKitFromCoreGraphicsFont(cgFont))
	cgFont.Release()
	return fontID
}

func (st *appleTextSystemState) isEmoji(fontID gpui.FontID) bool {
	postscriptName, ok := st.postscriptNamesByFontID[fontID]
	return ok && (postscriptName == "AppleColorEmoji" || postscriptName == ".AppleColorEmojiUI")
}

func (st *appleTextSystemState) rasterBounds(params gpui.RenderGlyphParams) (gpui.Bounds[gpui.DevicePixels], error) {
	font := st.fonts[params.FontID]
	r, err := font.rasterBounds(uint32(params.GlyphID), float32(params.FontSize), params.ScaleFactor)
	if err != nil {
		return gpui.Bounds[gpui.DevicePixels]{}, err
	}
	bounds := boundsFromRectI(r)
	return bounds.Dilate(gpui.DevicePixels(1)), nil
}

func (st *appleTextSystemState) rasterizeGlyph(params gpui.RenderGlyphParams, glyphBounds gpui.Bounds[gpui.DevicePixels]) (gpui.Size[gpui.DevicePixels], []byte, error) {
	if glyphBounds.Size.Width == 0 || glyphBounds.Size.Height == 0 {
		return gpui.Size[gpui.DevicePixels]{}, nil, errors.New("glyph bounds are empty")
	}
	bitmapSize := glyphBounds.Size
	if params.SubpixelVariant.X > 0 {
		bitmapSize.Width += 1
	}
	if params.SubpixelVariant.Y > 0 {
		bitmapSize.Height += 1
	}

	w, h := uint(bitmapSize.Width), uint(bitmapSize.Height)
	var bytes []byte
	var cx *cg.BitmapContext
	if params.IsEmoji {
		bytes = make([]byte, w*4*h)
		cs := cg.ColorSpaceCreateDeviceRGB()
		cx = cg.CreateBitmapContext(bytes, w, h, 8, w*4, cs, cg.ImageAlphaPremultipliedLast)
		cs.Release()
	} else {
		bytes = make([]byte, w*h)
		cs := cg.ColorSpaceCreateDeviceGray()
		cx = cg.CreateBitmapContext(bytes, w, h, 8, w, cs, cg.ImageAlphaOnly)
		cs.Release()
	}
	if cx == nil {
		panic("gpui: CGBitmapContextCreate failed")
	}

	cx.TranslateCTM(cg.Float(-glyphBounds.Origin.X), cg.Float(glyphBounds.Origin.Y+glyphBounds.Size.Height))
	cx.ScaleCTM(cg.Float(params.ScaleFactor), cg.Float(params.ScaleFactor))

	shiftX := float32(params.SubpixelVariant.X) / float32(gpui.SubpixelVariantsX)
	shiftY := float32(params.SubpixelVariant.Y) / float32(gpui.SubpixelVariantsX)
	cx.SetTextDrawingMode(cg.TextFill)
	cx.SetAllowsAntialiasing(true)
	cx.SetShouldAntialias(true)
	cx.SetAllowsFontSubpixelPositioning(true)
	cx.SetShouldSubpixelPositionFonts(true)
	cx.SetAllowsFontSubpixelQuantization(false)
	cx.SetShouldSubpixelQuantizeFonts(false)

	if runtime.GOOS == "darwin" && params.Dilation > 0 {
		luminance := float64(params.Dilation) * 0.25
		cx.SetShouldSmoothFonts(true)
		cx.SetGrayFillColor(luminance, 1.0)
	} else {
		cx.SetGrayFillColor(0.0, 1.0)
	}
	sized := st.fonts[params.FontID].nativeFont().CloneWithFontSize(float64(float32(params.FontSize)))
	sized.Ref().DrawGlyphs(
		[]cg.Glyph{cg.Glyph(params.GlyphID)},
		[]cg.Point{{X: cg.Float(shiftX / params.ScaleFactor), Y: cg.Float(shiftY / params.ScaleFactor)}},
		cx.Context,
	)
	sized.Release()
	cx.Release()

	if params.IsEmoji {
		for i := 0; i+4 <= len(bytes); i += 4 {
			gpui.SwapRgbaPaToBgra(bytes[i : i+4])
		}
	}

	return bitmapSize, bytes, nil
}

func (st *appleTextSystemState) layoutLine(text string, fontSize gpui.Pixels, fontRuns []gpui.FontRun) gpui.LineLayout {
	str := ct.AttributedStringCreateMutable()
	defer str.Release()
	s := str.Ref()
	var maxAscent, maxDescent float32

	{
		rest := text
		breakLigature := false
		for _, run := range fontRuns {
			textRun := rest[:run.Len]
			rest = rest[run.Len:]

			utf16Start := s.Len()
			runString := cf.StringCreate(textRun)
			s.ReplaceString(cf.Range{Location: utf16Start, Length: 0}, runString.Ref())
			runString.Release()
			utf16End := s.Len()

			length := utf16End - utf16Start
			cfRange := cf.Range{Location: utf16Start, Length: length}
			font := st.fonts[run.FontID]

			fontMetrics := font.metrics()
			fontScale := float32(fontSize) / float32(fontMetrics.unitsPerEm)
			maxAscent = max(maxAscent, fontMetrics.ascent*fontScale)
			maxDescent = max(maxDescent, -fontMetrics.descent*fontScale)

			runSize := fontSize
			if breakLigature {
				runSize = gpui.Px(math.Nextafter32(float32(fontSize), float32(math.Inf(1))))
			}
			sized := font.nativeFont().CloneWithFontSize(float64(float32(runSize)))
			s.SetAttribute(cfRange, ct.FontAttributeName, sized.TypeRef())
			sized.Release()
			if length > 0 {
				breakLigature = !breakLigature
			}
		}
	}

	line := ct.LineCreateWithAttributedString(s)
	defer line.Release()
	glyphRuns := line.Ref().GlyphRuns()
	runs := make([]gpui.ShapedRun, 0, len(glyphRuns))
	ixConverter := newStringIndexConverter(text)
	for _, run := range glyphRuns {
		attributes := run.Attributes()
		if attributes == 0 {
			panic("called `Option::unwrap()` on a `None` value")
		}
		fontRef, _ := attributes.Get(cf.TypeRef(ct.FontAttributeName))
		font, ok := ct.AsFont(fontRef)
		if !ok {
			panic("called `Option::unwrap()` on a `None` value")
		}
		fontID := st.idForNativeFont(font)

		if len(runs) == 0 || runs[len(runs)-1].FontID != fontID {
			runs = append(runs, gpui.ShapedRun{FontID: fontID, Glyphs: make([]gpui.ShapedGlyph, 0, run.GlyphCount())})
		}
		last := &runs[len(runs)-1]
		glyphs := run.Glyphs()
		positions := run.Positions()
		indices := run.StringIndices()
		for i := range min(len(glyphs), len(positions), len(indices)) {
			if indices[i] < 0 {
				panic("called `Result::unwrap()` on an `Err` value: TryFromIntError(())")
			}
			glyphUTF16Ix := int(indices[i])
			if ixConverter.utf16Ix > glyphUTF16Ix {
				ixConverter = newStringIndexConverter(text)
			}
			ixConverter.advanceToUTF16Ix(glyphUTF16Ix)
			last.Glyphs = append(last.Glyphs, gpui.ShapedGlyph{
				ID:       gpui.GlyphID(glyphs[i]),
				Position: gpui.NewPoint(gpui.Px(float32(positions[i].X)), gpui.Px(float32(positions[i].Y))),
				Index:    ixConverter.utf8Ix,
				IsEmoji:  st.isEmoji(fontID),
			})
		}
	}
	typographicBounds := line.Ref().TypographicBounds()
	return gpui.LineLayout{
		Runs:     runs,
		FontSize: fontSize,
		Width:    gpui.Px(float32(typographicBounds.Width)),
		Ascent:   gpui.Px(maxAscent),
		Descent:  gpui.Px(maxDescent),
		Len:      len(text),
	}
}

type stringIndexConverter struct {
	text    string
	utf8Ix  int
	utf16Ix int
}

func newStringIndexConverter(text string) stringIndexConverter {
	return stringIndexConverter{text: text}
}

func (c *stringIndexConverter) advanceToUTF16Ix(utf16Target int) {
	rest := c.text[c.utf8Ix:]
	for ix, ch := range rest {
		if c.utf16Ix >= utf16Target {
			c.utf8Ix += ix
			return
		}
		c.utf16Ix += utf16.RuneLen(ch)
	}
	c.utf8Ix = len(c.text)
}

func fontKitMetricsToMetrics(metrics fontKitMetrics) gpui.FontMetrics {
	return gpui.FontMetrics{
		UnitsPerEm:         metrics.unitsPerEm,
		Ascent:             metrics.ascent,
		Descent:            metrics.descent,
		LineGap:            metrics.lineGap,
		UnderlinePosition:  metrics.underlinePosition,
		UnderlineThickness: metrics.underlineThickness,
		CapHeight:          metrics.capHeight,
		XHeight:            metrics.xHeight,
		BoundingBox:        boundsFromRect(metrics.boundingBox),
	}
}

func boundsFromRect(rect rectF) gpui.Bounds[float32] {
	return gpui.Bounds[float32]{
		Origin: gpui.NewPoint(rect.minX, rect.minY),
		Size:   gpui.NewSize(rect.width(), rect.height()),
	}
}

func boundsFromRectI(rect rectI) gpui.Bounds[gpui.DevicePixels] {
	return gpui.Bounds[gpui.DevicePixels]{
		Origin: gpui.NewPoint(gpui.DevicePixels(rect.minX), gpui.DevicePixels(rect.minY)),
		Size:   gpui.NewSize(gpui.DevicePixels(rect.maxX-rect.minX), gpui.DevicePixels(rect.maxY-rect.minY)),
	}
}

func fontkitWeight(value gpui.FontWeight) float32 {
	return float32(value)
}

func fontkitStyle(style gpui.FontStyle) fontKitStyle {
	switch style {
	case gpui.FontStyleItalic:
		return fontKitStyleItalic
	case gpui.FontStyleOblique:
		return fontKitStyleOblique
	default:
		return fontKitStyleNormal
	}
}

// FamilyName returns the descriptor's family name, or false when it has none. It panics when the
// attribute is not a string, as upstream's lenient_font_attributes does.
func FamilyName(descriptor ct.FontDescriptor) (string, bool) {
	return getStringAttribute(descriptor, ct.FontFamilyNameAttribute)
}

func getStringAttribute(descriptor ct.FontDescriptor, attribute cf.String) (string, bool) {
	value := descriptor.CopyAttribute(attribute)
	if value == nil {
		return "", false
	}
	defer value.Release()
	s, ok := cf.AsString(value.Ref())
	if !ok {
		panic("assertion failed: value.instance_of::<CFString>()")
	}
	return s.GoString(), true
}
