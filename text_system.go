package gpui

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/jsonschema"
	"github.com/megascan/gpui-go/scheduler"
)

// FontID is an opaque identifier for a specific font.
type FontID uint

// FontFamilyID is an opaque identifier for a specific font family.
type FontFamilyID uint

// SubpixelVariantsX is the number of subpixel glyph variants along the X axis.
const SubpixelVariantsX uint8 = 4

// SubpixelVariantsY is the number of subpixel glyph variants along the Y axis.
const SubpixelVariantsY uint8 = 1

const underlineDescentOffsetFactor float32 = 0.618

// UnderlineYOffset returns the vertical offset used to paint an underline within a line.
func UnderlineYOffset(lineHeight, ascent, descent Pixels) Pixels {
	paddingTop := (lineHeight - ascent - descent) / 2.
	return paddingTop + ascent + descent*Pixels(underlineDescentOffsetFactor)
}

const maxReportedMissingGlyphs = 1024

// FallbackFontClass is the spacing behavior required of a fallback font.
type FallbackFontClass uint8

const (
	// FallbackFontClassProportional is a proportionally spaced fallback font.
	FallbackFontClassProportional FallbackFontClass = iota
	// FallbackFontClassMonospace is a fixed-width fallback font.
	FallbackFontClassMonospace
)

// MissingGlyph is a grapheme cluster that could not be represented by any available font.
type MissingGlyph struct {
	grapheme  SharedString
	fontClass FallbackFontClass
}

// NewMissingGlyph creates a missing-glyph report.
func NewMissingGlyph(grapheme SharedString, fontClass FallbackFontClass) MissingGlyph {
	return MissingGlyph{grapheme: grapheme, fontClass: fontClass}
}

// Grapheme returns the unresolved grapheme cluster.
func (m MissingGlyph) Grapheme() string {
	return string(m.grapheme)
}

// FontClass returns the spacing behavior required of a fallback font.
func (m MissingGlyph) FontClass() FallbackFontClass {
	return m.fontClass
}

// MissingGlyphSink accepts missing glyphs detected by a platform text system.
type MissingGlyphSink interface {
	Report(missingGlyphs []MissingGlyph)
}

type missingGlyphState struct {
	reported      map[MissingGlyph]struct{}
	reportedOrder []MissingGlyph
	generation    uint64
}

func newMissingGlyphState() missingGlyphState {
	return missingGlyphState{reported: map[MissingGlyph]struct{}{}}
}

func (s *missingGlyphState) reset(generation uint64) {
	clear(s.reported)
	s.reportedOrder = s.reportedOrder[:0]
	s.generation = generation
}

type queuedMissingGlyph struct {
	generation   uint64
	missingGlyph MissingGlyph
}

type missingGlyphReporter struct {
	generation *atomic.Uint64
	sender     *scheduler.ChanSender[queuedMissingGlyph]
}

func (r *missingGlyphReporter) Report(missingGlyphs []MissingGlyph) {
	if r.sender.IsClosed() {
		return
	}
	generation := r.generation.Load()
	seen := make(map[MissingGlyph]struct{}, len(missingGlyphs))
	for _, missingGlyph := range missingGlyphs {
		if _, dup := seen[missingGlyph]; dup {
			continue
		}
		seen[missingGlyph] = struct{}{}
		queued := queuedMissingGlyph{generation: generation, missingGlyph: missingGlyph}
		if r.sender.TrySend(queued) != nil {
			break
		}
	}
}

func (r *missingGlyphReporter) reset() {
	r.generation.Add(1)
}

// MissingGlyphReceiver receives batches of grapheme clusters that exhausted font fallback.
// Close is its drop.
type MissingGlyphReceiver struct {
	state      missingGlyphState
	generation *atomic.Uint64
	receiver   *scheduler.ChanReceiver[queuedMissingGlyph]
}

// Recv waits until at least one new missing glyph has been observed. It returns
// scheduler.ErrClosed if the reporting channel is closed.
func (r *MissingGlyphReceiver) Recv(cx scheduler.Awaiter) ([]MissingGlyph, error) {
	for {
		queued, ok := r.receiver.Recv(cx)
		if !ok {
			return nil, scheduler.ErrClosed
		}
		var missingGlyphs []MissingGlyph
		for taken := 0; taken < maxReportedMissingGlyphs; taken++ {
			if taken > 0 {
				next, ok, _ := r.receiver.TryRecv()
				if !ok {
					break
				}
				queued = next
			}
			generation := r.generation.Load()
			if r.state.generation != generation {
				r.state.reset(generation)
				missingGlyphs = missingGlyphs[:0]
			}
			if queued.generation != generation {
				continue
			}
			if _, dup := r.state.reported[queued.missingGlyph]; dup {
				continue
			}
			r.state.reported[queued.missingGlyph] = struct{}{}
			r.state.reportedOrder = append(r.state.reportedOrder, queued.missingGlyph)
			missingGlyphs = append(missingGlyphs, queued.missingGlyph)
			if len(r.state.reported) > maxReportedMissingGlyphs && len(r.state.reportedOrder) > 0 {
				expired := r.state.reportedOrder[0]
				r.state.reportedOrder = r.state.reportedOrder[1:]
				delete(r.state.reported, expired)
			}
		}
		if len(missingGlyphs) != 0 {
			return missingGlyphs, nil
		}
		yielded := false
		scheduler.Await(cx, scheduler.PollFn(func(c *scheduler.Context) (struct{}, bool) {
			if yielded {
				return struct{}{}, true
			}
			yielded = true
			c.Waker().WakeByRef()
			return struct{}{}, false
		}))
	}
}

// Close closes the reporting channel and discards queued reports. It is idempotent.
func (r *MissingGlyphReceiver) Close() {
	r.receiver.Close()
	for {
		if _, ok, _ := r.receiver.TryRecv(); !ok {
			break
		}
	}
}

type cachedFontKind uint8

const (
	cachedFontDirect cachedFontKind = iota
	cachedFontFallback
	cachedFontMissing
)

type cachedFont struct {
	kind   cachedFontKind
	fontID FontID
	err    error
}

type fontIDWithSize struct {
	fontID   FontID
	fontSize Pixels
}

// TextSystem is the GPUI text rendering sub system. It is safe for concurrent use.
type TextSystem struct {
	platformTextSystem     PlatformTextSystem
	fontIDsByFontMu        sync.RWMutex
	fontIDsUpgradeMu       sync.Mutex
	fontIDsByFont          map[Font]cachedFont
	fontMetricsMu          sync.RWMutex
	fontMetrics            map[FontID]FontMetrics
	rasterBoundsMu         sync.RWMutex
	rasterBounds           map[RenderGlyphParams]Bounds[DevicePixels]
	wrapperPoolMu          sync.Mutex
	wrapperPool            map[fontIDWithSize][]charWidthCache
	fallbackFontStack      []Font
	fontGeneration         *atomic.Uint64
	missingGlyphReporter   *missingGlyphReporter
	missingGlyphReceiverMu sync.Mutex
	missingGlyphReceiver   *MissingGlyphReceiver
}

// NewTextSystem creates a TextSystem with the given platform text system.
func NewTextSystem(platformTextSystem PlatformTextSystem) *TextSystem {
	sender, receiver := scheduler.Chan[queuedMissingGlyph](maxReportedMissingGlyphs)
	missingGlyphGeneration := &atomic.Uint64{}
	return &TextSystem{
		platformTextSystem: platformTextSystem,
		fontMetrics:        map[FontID]FontMetrics{},
		rasterBounds:       map[RenderGlyphParams]Bounds[DevicePixels]{},
		fontIDsByFont:      map[Font]cachedFont{},
		wrapperPool:        map[fontIDWithSize][]charWidthCache{},
		fallbackFontStack: []Font{
			NewFont(".ZedMono"),
			NewFont(".ZedSans"),
			NewFont("Helvetica"),
			NewFont("Segoe UI"),
			NewFont("Ubuntu"),
			NewFont("Adwaita Sans"),
			NewFont("Cantarell"),
			NewFont("Noto Sans"),
			NewFont("DejaVu Sans"),
			NewFont("Arial"),
		},
		fontGeneration: &atomic.Uint64{},
		missingGlyphReporter: &missingGlyphReporter{
			generation: missingGlyphGeneration,
			sender:     sender,
		},
		missingGlyphReceiver: &MissingGlyphReceiver{
			state:      newMissingGlyphState(),
			generation: missingGlyphGeneration,
			receiver:   receiver,
		},
	}
}

// AllFontNames returns the sorted, unique font family names available to the platform text
// system, including fonts registered with AddFonts.
func (ts *TextSystem) AllFontNames() []string {
	names := slices.Clone(ts.platformTextSystem.AllFontNames())
	slices.Sort(names)
	return slices.Compact(names)
}

// AddFonts adds font data to the text system. Cached font resolution and line layouts are
// invalidated after installation; layouts already in progress may complete against the
// previous font set.
func (ts *TextSystem) AddFonts(fonts [][]byte) error {
	if err := ts.platformTextSystem.AddFonts(fonts); err != nil {
		return err
	}
	ts.fontIDsUpgradeMu.Lock()
	ts.fontIDsByFontMu.Lock()
	clear(ts.fontIDsByFont)
	ts.fontIDsByFontMu.Unlock()
	ts.fontIDsUpgradeMu.Unlock()
	ts.missingGlyphReporter.reset()
	ts.fontGeneration.Add(1)
	return nil
}

func (ts *TextSystem) takeMissingGlyphReceiver() *MissingGlyphReceiver {
	if !ts.missingGlyphReceiverMu.TryLock() {
		return nil
	}
	defer ts.missingGlyphReceiverMu.Unlock()
	receiver := ts.missingGlyphReceiver
	ts.missingGlyphReceiver = nil
	return receiver
}

func (ts *TextSystem) enableMissingGlyphReporting() {
	ts.platformTextSystem.SetMissingGlyphSink(ts.missingGlyphReporter)
}

func (ts *TextSystem) disableMissingGlyphReporting() {
	ts.platformTextSystem.SetMissingGlyphSink(nil)
	ts.missingGlyphReporter.reset()
}

// GetFontForID returns the Font that resolved directly to the font id.
func (ts *TextSystem) GetFontForID(id FontID) (Font, bool) {
	ts.fontIDsByFontMu.RLock()
	defer ts.fontIDsByFontMu.RUnlock()
	for font, result := range ts.fontIDsByFont {
		if result.kind == cachedFontDirect && result.fontID == id {
			return font, true
		}
	}
	return Font{}, false
}

// ResolveFont resolves the font, falling back to the default font stack if it fails to
// load. It panics if neither the font nor any of the fallbacks can be resolved.
func (ts *TextSystem) ResolveFont(font Font) FontID {
	ts.fontIDsByFontMu.RLock()
	cached, ok := ts.fontIDsByFont[font]
	ts.fontIDsByFontMu.RUnlock()
	if ok && (cached.kind == cachedFontDirect || cached.kind == cachedFontFallback) {
		return cached.fontID
	}
	return ts.resolveFontUncached(font)
}

// PrewarmFonts prewarms any system font caches needed to shape text. It may be expensive, so
// callers should generally invoke it on a background executor.
func (ts *TextSystem) PrewarmFonts(fonts []Font) {
	var fontIDs []FontID
	for _, font := range fonts {
		fontID := ts.ResolveFont(font)
		if !slices.Contains(fontIDs, fontID) {
			fontIDs = append(fontIDs, fontID)
		}
	}
	ts.platformTextSystem.PrewarmFonts(fontIDs)
}

// BoundingBox returns the smallest rectangle that could enclose all glyphs of the font
// superimposed over one another, at the font size.
func (ts *TextSystem) BoundingBox(fontID FontID, fontSize Pixels) Bounds[Pixels] {
	metrics := ts.readMetrics(fontID)
	return metrics.BoundingBoxPixels(fontSize)
}

// TypographicBounds returns the typographic bounds of the character in the font and size.
func (ts *TextSystem) TypographicBounds(fontID FontID, fontSize Pixels, character rune) (Bounds[Pixels], error) {
	glyphID, ok := ts.platformTextSystem.GlyphForChar(fontID, character)
	if !ok {
		return Bounds[Pixels]{}, fmt.Errorf("glyph not found for character '%c'", character)
	}
	bounds, err := ts.platformTextSystem.TypographicBounds(fontID, glyphID)
	if err != nil {
		return Bounds[Pixels]{}, err
	}
	metrics := ts.readMetrics(fontID)
	return bounds.Div(float32(metrics.UnitsPerEm)).Mul(float32(fontSize)).Map(func(v float32) Pixels { return Px(v) }), nil
}

// Advance returns the advance of the character in the font and size.
func (ts *TextSystem) Advance(fontID FontID, fontSize Pixels, ch rune) (Size[Pixels], error) {
	glyphID, ok := ts.platformTextSystem.GlyphForChar(fontID, ch)
	if !ok {
		return Size[Pixels]{}, fmt.Errorf("glyph not found for character '%c'", ch)
	}
	advance, err := ts.platformTextSystem.Advance(fontID, glyphID)
	if err != nil {
		return Size[Pixels]{}, err
	}
	result := advance.Div(float32(ts.UnitsPerEm(fontID)))
	return Size[Pixels]{
		Width:  Pixels(result.Width * float32(fontSize)),
		Height: Pixels(result.Height * float32(fontSize)),
	}, nil
}

// LayoutWidth returns the shaped layout width of the character in the font and size.
func (ts *TextSystem) LayoutWidth(fontID FontID, fontSize Pixels, ch rune) Pixels {
	buffer := string(ch)
	return ts.platformTextSystem.LayoutLine(buffer, fontSize, []FontRun{{Len: len(buffer), FontID: fontID}}).Width
}

// EmWidth returns the width of an em, the width of the `m` character.
func (ts *TextSystem) EmWidth(fontID FontID, fontSize Pixels) (Pixels, error) {
	bounds, err := ts.TypographicBounds(fontID, fontSize, 'm')
	if err != nil {
		return 0, err
	}
	return bounds.Size.Width, nil
}

// EmAdvance returns the advance width of an em, the advance of the `m` character.
func (ts *TextSystem) EmAdvance(fontID FontID, fontSize Pixels) (Pixels, error) {
	advance, err := ts.Advance(fontID, fontSize, 'm')
	if err != nil {
		return 0, err
	}
	return advance.Width, nil
}

// ChWidth returns the width of a ch, the width of the `0` character.
func (ts *TextSystem) ChWidth(fontID FontID, fontSize Pixels) (Pixels, error) {
	bounds, err := ts.TypographicBounds(fontID, fontSize, '0')
	if err != nil {
		return 0, err
	}
	return bounds.Size.Width, nil
}

// ChAdvance returns the advance width of a ch, the advance of the `0` character.
func (ts *TextSystem) ChAdvance(fontID FontID, fontSize Pixels) (Pixels, error) {
	advance, err := ts.Advance(fontID, fontSize, '0')
	if err != nil {
		return 0, err
	}
	return advance.Width, nil
}

// UnitsPerEm returns the number of font units per em square.
func (ts *TextSystem) UnitsPerEm(fontID FontID) uint32 {
	return ts.readMetrics(fontID).UnitsPerEm
}

// CapHeight returns the height of a capital letter in the font and size.
func (ts *TextSystem) CapHeight(fontID FontID, fontSize Pixels) Pixels {
	metrics := ts.readMetrics(fontID)
	return metrics.CapHeightPixels(fontSize)
}

// XHeight returns the height of the x character in the font and size.
func (ts *TextSystem) XHeight(fontID FontID, fontSize Pixels) Pixels {
	metrics := ts.readMetrics(fontID)
	return metrics.XHeightPixels(fontSize)
}

// Ascent returns the recommended distance from the baseline for the font.
func (ts *TextSystem) Ascent(fontID FontID, fontSize Pixels) Pixels {
	metrics := ts.readMetrics(fontID)
	return metrics.AscentPixels(fontSize)
}

// Descent returns the recommended distance below the baseline for the font, in single
// spaced text.
func (ts *TextSystem) Descent(fontID FontID, fontSize Pixels) Pixels {
	metrics := ts.readMetrics(fontID)
	return metrics.DescentPixels(fontSize)
}

// BaselineOffset returns the recommended baseline offset for the font and line height.
func (ts *TextSystem) BaselineOffset(fontID FontID, fontSize, lineHeight Pixels) Pixels {
	ascent := ts.Ascent(fontID, fontSize)
	descent := ts.Descent(fontID, fontSize)
	paddingTop := (lineHeight - ascent - descent) / 2.
	return paddingTop + ascent
}

func (ts *TextSystem) readMetrics(fontID FontID) FontMetrics {
	ts.fontMetricsMu.RLock()
	metrics, ok := ts.fontMetrics[fontID]
	ts.fontMetricsMu.RUnlock()
	if ok {
		return metrics
	}
	ts.fontMetricsMu.Lock()
	defer ts.fontMetricsMu.Unlock()
	if metrics, ok := ts.fontMetrics[fontID]; ok {
		return metrics
	}
	metrics = ts.platformTextSystem.FontMetrics(fontID)
	ts.fontMetrics[fontID] = metrics
	return metrics
}

// LineWrapper returns a handle to a line wrapper for the font and font size. Release returns
// it to the pool.
func (ts *TextSystem) LineWrapper(font Font, fontSize Pixels) *LineWrapperHandle {
	ts.wrapperPoolMu.Lock()
	defer ts.wrapperPoolMu.Unlock()
	fontID := ts.ResolveFont(font)
	key := fontIDWithSize{fontID: fontID, fontSize: fontSize}
	pool := ts.wrapperPool[key]
	var charWidths charWidthCache
	if n := len(pool); n > 0 {
		charWidths = pool[n-1]
		pool = pool[:n-1]
	} else {
		charWidths = defaultCharWidthCache()
	}
	ts.wrapperPool[key] = pool
	return &LineWrapperHandle{
		LineWrapper: newLineWrapper(fontID, fontSize, ts, charWidths),
		textSystem:  ts,
	}
}

func (ts *TextSystem) getRasterBounds(params RenderGlyphParams) (Bounds[DevicePixels], error) {
	ts.rasterBoundsMu.RLock()
	bounds, ok := ts.rasterBounds[params]
	ts.rasterBoundsMu.RUnlock()
	if ok {
		return bounds, nil
	}
	ts.rasterBoundsMu.Lock()
	defer ts.rasterBoundsMu.Unlock()
	if bounds, ok := ts.rasterBounds[params]; ok {
		return bounds, nil
	}
	bounds, err := ts.platformTextSystem.GlyphRasterBounds(params)
	if err != nil {
		return Bounds[DevicePixels]{}, err
	}
	ts.rasterBounds[params] = bounds
	return bounds, nil
}

func (ts *TextSystem) rasterizeGlyph(params RenderGlyphParams) (Size[DevicePixels], []byte, error) {
	rasterBounds, err := ts.getRasterBounds(params)
	if err != nil {
		return Size[DevicePixels]{}, nil, err
	}
	return ts.platformTextSystem.RasterizeGlyph(params, rasterBounds)
}

func (ts *TextSystem) glyphDilationForColor(color Hsla) uint8 {
	return ts.platformTextSystem.GlyphDilationForColor(color)
}

func (ts *TextSystem) recommendedRenderingMode(fontID FontID, fontSize Pixels) TextRenderingMode {
	return ts.platformTextSystem.RecommendedRenderingMode(fontID, fontSize)
}

func (ts *TextSystem) resolveFontUncached(font Font) FontID {
	var fontError error
	fontID, found := func() (FontID, bool) {
		ts.fontIDsUpgradeMu.Lock()
		defer ts.fontIDsUpgradeMu.Unlock()
		insert := func(key Font, value cachedFont) {
			ts.fontIDsByFontMu.Lock()
			ts.fontIDsByFont[key] = value
			ts.fontIDsByFontMu.Unlock()
		}
		if cached, ok := ts.fontIDsByFont[font]; ok && (cached.kind == cachedFontDirect || cached.kind == cachedFontFallback) {
			return cached.fontID, true
		}
		lookupFont := func(candidate Font) (FontID, bool) {
			cached, ok := ts.fontIDsByFont[candidate]
			if !ok {
				if fontID, err := ts.platformTextSystem.FontID(candidate); err == nil {
					cached = cachedFont{kind: cachedFontDirect, fontID: fontID}
				} else {
					cached = cachedFont{kind: cachedFontMissing, err: err}
				}
				insert(candidate, cached)
			}
			switch cached.kind {
			case cachedFontDirect:
				return cached.fontID, true
			case cachedFontMissing:
				if candidate == font {
					fontError = cached.err
				}
			}
			return 0, false
		}
		if fontID, ok := lookupFont(font); ok {
			return fontID, true
		}
		for _, fallback := range ts.fallbackFontStack {
			if fontID, ok := lookupFont(fallback); ok {
				insert(font, cachedFont{kind: cachedFontFallback, fontID: fontID})
				return fontID, true
			}
		}
		return 0, false
	}()
	if fontError != nil {
		slog.Warn(fmt.Sprintf("failed to load font %+v: %v", font, fontError))
	}
	if !found {
		families := make([]string, len(ts.fallbackFontStack))
		for i, fallback := range ts.fallbackFontStack {
			families[i] = string(fallback.Family)
		}
		panic(fmt.Sprintf("failed to resolve font '%s' or any of the fallbacks: %s", font.Family, strings.Join(families, ", ")))
	}
	return fontID
}

// WindowTextSystem is the GPUI text layout subsystem of one window. It embeds the shared
// TextSystem (upstream's Deref) and is used from the window's thread only.
type WindowTextSystem struct {
	*TextSystem
	lineLayoutCache *LineLayoutCache
	fontRunsBuffer  []FontRun
}

// NewWindowTextSystem creates a WindowTextSystem over the TextSystem.
func NewWindowTextSystem(textSystem *TextSystem) *WindowTextSystem {
	return &WindowTextSystem{
		TextSystem:      textSystem,
		lineLayoutCache: NewLineLayoutCache(textSystem.platformTextSystem, textSystem.fontGeneration),
	}
}

func (w *WindowTextSystem) layoutIndex() LineLayoutIndex {
	return w.lineLayoutCache.layoutIndex()
}

func (w *WindowTextSystem) reuseLayouts(start, end LineLayoutIndex) {
	w.lineLayoutCache.reuseLayouts(start, end)
}

func (w *WindowTextSystem) truncateLayouts(index LineLayoutIndex) {
	w.lineLayoutCache.truncateLayouts(index)
}

func sameDecoration(last *DecorationRun, run *TextRun) bool {
	return last.Color == run.Color &&
		last.Underline == run.Underline &&
		last.Strikethrough == run.Strikethrough &&
		last.BackgroundColor == run.BackgroundColor
}

// ShapeLine shapes a single line of text at the font size for painting; subsets of the line
// are styled by runs. The text must not contain newlines; use ShapeText for several lines.
func (w *WindowTextSystem) ShapeLine(text SharedString, fontSize Pixels, runs []TextRun, forceWidth Option[Pixels]) ShapedLine {
	if gpuiutil.DebugAssertions && strings.Contains(string(text), "\n") {
		panic("text argument should not contain newlines")
	}
	var decorationRuns []DecorationRun
	for i := range runs {
		run := &runs[i]
		if n := len(decorationRuns); n > 0 && sameDecoration(&decorationRuns[n-1], run) {
			decorationRuns[n-1].Len += uint32(run.Len)
			continue
		}
		decorationRuns = append(decorationRuns, DecorationRun{
			Len:             uint32(run.Len),
			Color:           run.Color,
			BackgroundColor: run.BackgroundColor,
			Underline:       run.Underline,
			Strikethrough:   run.Strikethrough,
		})
	}
	layout := w.LayoutLine(text, fontSize, runs, forceWidth)
	return ShapedLine{
		LineLayout:     layout,
		Text:           text,
		decorationRuns: decorationRuns,
	}
}

// ShapeText shapes multi-line text at the font size for painting; subsets of the text are
// styled by runs. If wrapWidth is set, line breaks are adjusted to fit within it.
func (w *WindowTextSystem) ShapeText(text SharedString, fontSize Pixels, runs []TextRun, wrapWidth Option[Pixels], lineClamp Option[int]) ([]WrappedLine, error) {
	var pending []TextRun
	for _, run := range runs {
		if run.Len > 0 {
			pending = append(pending, run)
		}
	}
	runIx := 0
	fontRuns := w.fontRunsBuffer
	w.fontRunsBuffer = nil

	var lines []WrappedLine
	maxWrapLines := lineClamp
	wrappedLines := 0

	processLine := func(lineText SharedString, lineStart, lineEnd int) {
		fontRuns = fontRuns[:0]
		var decorationRuns []DecorationRun
		runStart := lineStart
		for runStart < lineEnd {
			if runIx >= len(pending) {
				slog.Warn("`TextRun`s do not cover the entire to be shaped text")
				break
			}
			run := &pending[runIx]
			runLenWithinLine := min(lineEnd-runStart, run.Len)

			decorationChanged := true
			if n := len(decorationRuns); n > 0 && sameDecoration(&decorationRuns[n-1], run) {
				decorationRuns[n-1].Len += uint32(runLenWithinLine)
				decorationChanged = false
			} else {
				decorationRuns = append(decorationRuns, DecorationRun{
					Len:             uint32(runLenWithinLine),
					Color:           run.Color,
					BackgroundColor: run.BackgroundColor,
					Underline:       run.Underline,
					Strikethrough:   run.Strikethrough,
				})
			}

			fontID := w.ResolveFont(run.Font)
			if n := len(fontRuns); n > 0 && fontID == fontRuns[n-1].FontID && !decorationChanged {
				fontRuns[n-1].Len += runLenWithinLine
			} else {
				fontRuns = append(fontRuns, FontRun{Len: runLenWithinLine, FontID: fontID})
			}

			run.Len -= runLenWithinLine
			if run.Len == 0 {
				runIx++
			}
			runStart += runLenWithinLine
		}

		maxLines := None[int]()
		if max, ok := maxWrapLines.Get(); ok {
			maxLines = Some(saturatingSub(max, wrappedLines))
		}
		layout := w.lineLayoutCache.layoutWrappedLine(string(lineText), fontSize, fontRuns, wrapWidth, maxLines)
		wrappedLines += len(layout.WrapBoundaries)

		lines = append(lines, WrappedLine{
			WrappedLineLayout: layout,
			decorationRuns:    decorationRuns,
			Text:              lineText,
		})

		if runIx < len(pending) {
			run := &pending[runIx]
			run.Len--
			if run.Len == 0 {
				runIx++
			}
		}
	}

	splitLines := strings.Split(string(text), "\n")
	if len(splitLines) >= 2 {
		lineStart := 0
		for i, lineText := range splitLines {
			if i > 0 {
				lineStart += len(splitLines[i-1]) + 1
			}
			processLine(NewSharedString(lineText), lineStart, lineStart+len(lineText))
		}
	} else {
		processLine(text, 0, len(text))
	}

	w.fontRunsBuffer = fontRuns
	return lines, nil
}

func saturatingSub(a, b int) int {
	if a < b {
		return 0
	}
	return a - b
}

func (w *WindowTextSystem) finishFrame() {
	w.lineLayoutCache.finishFrame()
}

// LayoutLine lays out the line of text at the font size; subsets of the line are styled by
// runs. Prefer ShapeLine, whose result can be painted directly.
func (w *WindowTextSystem) LayoutLine(text SharedString, fontSize Pixels, runs []TextRun, forceWidth Option[Pixels]) *LineLayout {
	fontRuns := w.fontRuns(runs)
	layout := w.lineLayoutCache.layoutLine(string(text), fontSize, fontRuns, forceWidth)
	w.fontRunsBuffer = fontRuns
	return layout
}

func (w *WindowTextSystem) fontRuns(runs []TextRun) []FontRun {
	var lastRun *TextRun
	fontRuns := w.fontRunsBuffer[:0]
	w.fontRunsBuffer = nil

	for i := range runs {
		run := &runs[i]
		decorationChanged := true
		if lastRun != nil &&
			lastRun.Color == run.Color &&
			lastRun.Underline == run.Underline &&
			lastRun.Strikethrough == run.Strikethrough {
			decorationChanged = false
		} else {
			lastRun = run
		}

		fontID := w.ResolveFont(run.Font)
		if n := len(fontRuns); n > 0 && fontID == fontRuns[n-1].FontID && !decorationChanged {
			fontRuns[n-1].Len += run.Len
		} else {
			fontRuns = append(fontRuns, FontRun{Len: run.Len, FontID: fontID})
		}
	}
	return fontRuns
}

// LayoutWidth returns the shaped layout width of the character in the font and size.
func (w *WindowTextSystem) LayoutWidth(fontID FontID, fontSize Pixels, ch rune) Pixels {
	buffer := string(ch)
	return w.lineLayoutCache.layoutLine(buffer, fontSize, []FontRun{{Len: len(buffer), FontID: fontID}}, None[Pixels]()).Width
}

// EmLayoutWidth returns the shaped layout width of an em.
func (w *WindowTextSystem) EmLayoutWidth(fontID FontID, fontSize Pixels) Pixels {
	return w.LayoutWidth(fontID, fontSize, 'm')
}

// LineWrapperHandle is a pooled line wrapper; Release returns it to the text system's pool.
// It embeds the wrapper (upstream's Deref); using it after Release panics.
type LineWrapperHandle struct {
	*LineWrapper
	textSystem *TextSystem
}

// Release returns the wrapper's character widths to the pool. It is idempotent.
func (h *LineWrapperHandle) Release() {
	if h.LineWrapper == nil {
		return
	}
	wrapper := h.LineWrapper
	h.LineWrapper = nil
	h.textSystem.wrapperPoolMu.Lock()
	defer h.textSystem.wrapperPoolMu.Unlock()
	key := fontIDWithSize{fontID: wrapper.fontID, fontSize: wrapper.fontSize}
	pool, ok := h.textSystem.wrapperPool[key]
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	h.textSystem.wrapperPool[key] = append(pool, wrapper.intoCharWidths())
}

// FontWeight is the degree of blackness or stroke thickness of a font, from 100 to 900 with
// 400 as normal.
type FontWeight float32

const (
	// FontWeightThin is the thin weight (100), the thinnest value.
	FontWeightThin FontWeight = 100.0
	// FontWeightExtraLight is the extra light weight (200).
	FontWeightExtraLight FontWeight = 200.0
	// FontWeightLight is the light weight (300).
	FontWeightLight FontWeight = 300.0
	// FontWeightNormal is the normal weight (400).
	FontWeightNormal FontWeight = 400.0
	// FontWeightMedium is the medium weight (500).
	FontWeightMedium FontWeight = 500.0
	// FontWeightSemibold is the semibold weight (600).
	FontWeightSemibold FontWeight = 600.0
	// FontWeightBold is the bold weight (700).
	FontWeightBold FontWeight = 700.0
	// FontWeightExtraBold is the extra-bold weight (800).
	FontWeightExtraBold FontWeight = 800.0
	// FontWeightBlack is the black weight (900), the thickest value.
	FontWeightBlack FontWeight = 900.0
)

// FontWeightAll lists all of the font weights, from thinnest to thickest.
var FontWeightAll = [9]FontWeight{
	FontWeightThin,
	FontWeightExtraLight,
	FontWeightLight,
	FontWeightNormal,
	FontWeightMedium,
	FontWeightSemibold,
	FontWeightBold,
	FontWeightExtraBold,
	FontWeightBlack,
}

// DefaultFontWeight returns FontWeightNormal.
func DefaultFontWeight() FontWeight {
	return FontWeightNormal
}

// String formats the weight as a number.
func (w FontWeight) String() string {
	return strconv.FormatFloat(float64(w), 'f', -1, 32)
}

// FontWeightFromStr parses a weight from a number.
func FontWeightFromStr(s string) (FontWeight, error) {
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid font weight %q: %w", s, err)
	}
	return FontWeight(v), nil
}

// JSONSchema describes the weight as a number between thin and black.
func (FontWeight) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        jsonschema.Types{"number"},
		Minimum:     float32(FontWeightThin),
		Maximum:     float32(FontWeightBlack),
		Description: "Font weight value between 100 (thin) and 900 (black)",
	}
}

// FontStyle selects italic or oblique faces.
type FontStyle uint8

const (
	// FontStyleNormal is a face that is neither italic nor obliqued.
	FontStyleNormal FontStyle = iota
	// FontStyleItalic is a form that is generally cursive in nature.
	FontStyleItalic
	// FontStyleOblique is a typically-sloped version of the regular face.
	FontStyleOblique
)

var fontStyleNames = [...]string{"Normal", "Italic", "Oblique"}

// String returns the variant name.
func (s FontStyle) String() string {
	if int(s) < len(fontStyleNames) {
		return fontStyleNames[s]
	}
	return "FontStyle(" + strconv.Itoa(int(s)) + ")"
}

// MarshalText encodes the style as its variant name.
func (s FontStyle) MarshalText() ([]byte, error) {
	if int(s) >= len(fontStyleNames) {
		return nil, fmt.Errorf("invalid FontStyle %d", s)
	}
	return []byte(fontStyleNames[s]), nil
}

// UnmarshalText decodes the style from its variant name.
func (s *FontStyle) UnmarshalText(text []byte) error {
	for i, name := range fontStyleNames {
		if string(text) == name {
			*s = FontStyle(i)
			return nil
		}
	}
	return fmt.Errorf("unknown variant `%s`, expected one of `Normal`, `Italic`, `Oblique`", text)
}

// JSONSchema describes the style as one of its variant names.
func (FontStyle) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: jsonschema.Types{"string"},
		Enum: []any{"Normal", "Italic", "Oblique"},
	}
}

// TextRun is a styled run of text.
type TextRun struct {
	// Len is the number of UTF-8 bytes.
	Len int
	// Font is the font to use for this run.
	Font Font
	// Color is the color.
	Color Hsla
	// BackgroundColor is the background color, if any.
	BackgroundColor Option[Hsla]
	// Underline is the underline style, if any.
	Underline Option[UnderlineStyle]
	// Strikethrough is the strikethrough style, if any.
	Strikethrough Option[StrikethroughStyle]
}

// DefaultTextRun returns a zero-length run in the default font.
func DefaultTextRun() TextRun {
	return TextRun{Font: DefaultFont()}
}

// GlyphID is an identifier for a specific glyph, as returned by WindowTextSystem.LayoutLine.
type GlyphID uint32

// RenderGlyphParams are the parameters for rendering a glyph, used as cache keys for raster
// bounds and sprite atlas entries; they stay comparable.
type RenderGlyphParams struct {
	// FontID is the font.
	FontID FontID
	// GlyphID is the glyph.
	GlyphID GlyphID
	// FontSize is the font size.
	FontSize Pixels
	// SubpixelVariant is the subpixel variant on each axis.
	SubpixelVariant PointOf[uint8]
	// ScaleFactor is the window scale factor.
	ScaleFactor float32
	// IsEmoji reports whether the glyph is an emoji.
	IsEmoji bool
	// SubpixelRendering reports whether the glyph uses subpixel rendering.
	SubpixelRendering bool
	// Dilation is the dilation level.
	Dilation uint8
}

// Font identifies a specific font. It is comparable by value.
type Font struct {
	// Family is the font family name; ".SystemUIFont" identifies the platform's system UI font.
	Family SharedString
	// Features are the font features to use.
	Features FontFeatures
	// Fallbacks are the fallback fonts to use.
	Fallbacks Option[FontFallbacks]
	// Weight is the font weight.
	Weight FontWeight
	// Style is the font style.
	Style FontStyle
}

// DefaultFont returns the system UI font.
func DefaultFont() Font {
	return NewFont(".SystemUIFont")
}

// NewFont returns a Font for the family name.
func NewFont(family SharedString) Font {
	return Font{
		Family:    family,
		Features:  DefaultFontFeatures(),
		Weight:    DefaultFontWeight(),
		Style:     FontStyleNormal,
		Fallbacks: None[FontFallbacks](),
	}
}

// Bold returns the font with the bold weight.
func (f Font) Bold() Font {
	f.Weight = FontWeightBold
	return f
}

// Italic returns the font with the italic style.
func (f Font) Italic() Font {
	f.Style = FontStyleItalic
	return f
}

// FontMetrics are the measurements of a typeface, in font units.
type FontMetrics struct {
	// UnitsPerEm is the number of font units in the em square.
	UnitsPerEm uint32
	// Ascent is the distance from the baseline to the top of the glyph covers.
	Ascent float32
	// Descent is the distance from the baseline to the bottom of the glyph covers.
	Descent float32
	// LineGap is the recommended additional space between lines.
	LineGap float32
	// UnderlinePosition is the suggested position of the underline.
	UnderlinePosition float32
	// UnderlineThickness is the suggested thickness of the underline.
	UnderlineThickness float32
	// CapHeight is the height of a capital letter measured from the baseline.
	CapHeight float32
	// XHeight is the height of a lowercase x.
	XHeight float32
	// BoundingBox is the outer limits of the area that the font covers (head table xMin..yMax).
	BoundingBox Bounds[float32]
}

func (m FontMetrics) scale(v float32, fontSize Pixels) Pixels {
	return Pixels((v / float32(m.UnitsPerEm)) * float32(fontSize))
}

// AscentPixels returns the distance from the baseline to the top of the glyph covers in pixels
// (upstream's `ascent` method, D-FIELDMETHOD).
func (m FontMetrics) AscentPixels(fontSize Pixels) Pixels {
	return m.scale(m.Ascent, fontSize)
}

// DescentPixels returns the distance from the baseline to the bottom of the glyph covers in pixels.
func (m FontMetrics) DescentPixels(fontSize Pixels) Pixels {
	return m.scale(m.Descent, fontSize)
}

// LineGapPixels returns the recommended additional space between lines in pixels.
func (m FontMetrics) LineGapPixels(fontSize Pixels) Pixels {
	return m.scale(m.LineGap, fontSize)
}

// UnderlinePositionPixels returns the suggested position of the underline in pixels.
func (m FontMetrics) UnderlinePositionPixels(fontSize Pixels) Pixels {
	return m.scale(m.UnderlinePosition, fontSize)
}

// UnderlineThicknessPixels returns the suggested thickness of the underline in pixels.
func (m FontMetrics) UnderlineThicknessPixels(fontSize Pixels) Pixels {
	return m.scale(m.UnderlineThickness, fontSize)
}

// CapHeightPixels returns the height of a capital letter in pixels.
func (m FontMetrics) CapHeightPixels(fontSize Pixels) Pixels {
	return m.scale(m.CapHeight, fontSize)
}

// XHeightPixels returns the height of a lowercase x in pixels.
func (m FontMetrics) XHeightPixels(fontSize Pixels) Pixels {
	return m.scale(m.XHeight, fontSize)
}

// BoundingBoxPixels returns the outer limits of the area that the font covers in pixels.
func (m FontMetrics) BoundingBoxPixels(fontSize Pixels) Bounds[Pixels] {
	return m.BoundingBox.Div(float32(m.UnitsPerEm)).Mul(float32(fontSize)).Map(func(v float32) Pixels { return Px(v) })
}

// FontNameWithFallbacks maps well-known virtual font names to their concrete equivalents.
func FontNameWithFallbacks(name, system string) string {
	switch name {
	case ".SystemUIFont":
		return system
	case ".ZedSans", "Zed Plex Sans":
		return "IBM Plex Sans"
	case ".ZedMono", "Zed Plex Mono":
		return "Lilex"
	default:
		return name
	}
}

// FontNameWithFallbacksShared is FontNameWithFallbacks over SharedString.
func FontNameWithFallbacksShared(name, system SharedString) SharedString {
	return SharedString(FontNameWithFallbacks(string(name), string(system)))
}
