package gpui

import (
	"encoding/binary"
	"math"
	"sort"
	"sync/atomic"
)

// LineLayout is a laid out and styled line of text.
type LineLayout struct {
	// FontSize is the font size of the line.
	FontSize Pixels
	// Width is the width of the line.
	Width Pixels
	// Ascent is the ascent of the line.
	Ascent Pixels
	// Descent is the descent of the line.
	Descent Pixels
	// Runs are the shaped runs that make up the line.
	Runs []ShapedRun
	// Len is the length of the line in UTF-8 bytes.
	Len int
}

// ShapedRun is a run of shaped glyphs in one font.
type ShapedRun struct {
	// FontID is the font of the run.
	FontID FontID
	// Glyphs are the glyphs that make up the run.
	Glyphs []ShapedGlyph
}

// ShapedGlyph is a single glyph, ready to paint.
type ShapedGlyph struct {
	// ID is the glyph id, as determined by the text system.
	ID GlyphID
	// Position is the position of the glyph in its containing line.
	Position Point[Pixels]
	// Index is the UTF-8 byte index of the glyph in the original text.
	Index int
	// IsEmoji reports whether the glyph is an emoji.
	IsEmoji bool
}

// IndexForX returns the index of the character at the x coordinate, or false if x is past
// the end of the line.
func (l *LineLayout) IndexForX(x Pixels) (int, bool) {
	if x >= l.Width {
		return 0, false
	}
	for r := len(l.Runs) - 1; r >= 0; r-- {
		glyphs := l.Runs[r].Glyphs
		for g := len(glyphs) - 1; g >= 0; g-- {
			if glyphs[g].Position.X <= x {
				return glyphs[g].Index, true
			}
		}
	}
	return 0, true
}

// ClosestIndexForX returns the character boundary closest to the x coordinate.
func (l *LineLayout) ClosestIndexForX(x Pixels) int {
	prevIndex := 0
	prevX := Px(0.)
	for _, run := range l.Runs {
		for _, glyph := range run.Glyphs {
			if glyph.Position.X >= x {
				if glyph.Position.X-x < x-prevX {
					return glyph.Index
				}
				return prevIndex
			}
			prevIndex = glyph.Index
			prevX = glyph.Position.X
		}
	}
	if l.Len == 1 {
		if x > l.Width/2. {
			return 1
		}
		return 0
	}
	return l.Len
}

// XForIndex returns the x position of the character at the index.
func (l *LineLayout) XForIndex(index int) Pixels {
	for _, run := range l.Runs {
		for _, glyph := range run.Glyphs {
			if glyph.Index >= index {
				return glyph.Position.X
			}
		}
	}
	return l.Width
}

// FontIDForIndex returns the font of the character at the index.
func (l *LineLayout) FontIDForIndex(index int) (FontID, bool) {
	for _, run := range l.Runs {
		for _, glyph := range run.Glyphs {
			if glyph.Index >= index {
				return run.FontID, true
			}
		}
	}
	return 0, false
}

// SplitAt splits the layout at a byte index into a prefix with the glyphs of [0, byteIndex)
// at their original positions and a suffix with the glyphs of [byteIndex, Len) shifted to
// start at x=0 with indices rebased to 0. FontSize, Ascent and Descent are copied to both.
func (l *LineLayout) SplitAt(byteIndex int) (LineLayout, LineLayout) {
	xOffset := l.XForIndex(byteIndex)
	var leftRuns, rightRuns []ShapedRun
	for _, run := range l.Runs {
		splitPos := sort.Search(len(run.Glyphs), func(i int) bool { return run.Glyphs[i].Index >= byteIndex })
		if splitPos > 0 {
			leftRuns = append(leftRuns, ShapedRun{
				FontID: run.FontID,
				Glyphs: append([]ShapedGlyph(nil), run.Glyphs[:splitPos]...),
			})
		}
		if splitPos < len(run.Glyphs) {
			rightGlyphs := make([]ShapedGlyph, 0, len(run.Glyphs)-splitPos)
			for _, g := range run.Glyphs[splitPos:] {
				rightGlyphs = append(rightGlyphs, ShapedGlyph{
					ID:       g.ID,
					Position: NewPoint(g.Position.X-xOffset, g.Position.Y),
					Index:    g.Index - byteIndex,
					IsEmoji:  g.IsEmoji,
				})
			}
			rightRuns = append(rightRuns, ShapedRun{FontID: run.FontID, Glyphs: rightGlyphs})
		}
	}
	left := LineLayout{
		FontSize: l.FontSize,
		Width:    xOffset,
		Ascent:   l.Ascent,
		Descent:  l.Descent,
		Runs:     leftRuns,
		Len:      byteIndex,
	}
	right := LineLayout{
		FontSize: l.FontSize,
		Width:    l.Width - xOffset,
		Ascent:   l.Ascent,
		Descent:  l.Descent,
		Runs:     rightRuns,
		Len:      l.Len - byteIndex,
	}
	return left, right
}

type wrapGlyph struct {
	boundary WrapBoundary
	index    int
	x        Pixels
}

func (l *LineLayout) computeWrapBoundaries(text string, wrapWidth Pixels, maxLines Option[int], scratch *wrapScratch) []WrapBoundary {
	var boundaries []WrapBoundary
	var currentSegment struct {
		start, end int
		ok         bool
	}
	var lastCandidateIx Option[WrapBoundary]
	lastCandidateX := Px(0.)
	lastBoundary := WrapBoundary{RunIx: 0, GlyphIx: 0}
	lastBoundaryX := Px(0.)

	if scratch.glyphIndices == nil {
		scratch.glyphIndices = map[int]struct{}{}
	}
	clear(scratch.glyphIndices)
	for _, run := range l.Runs {
		for _, glyph := range run.Glyphs {
			scratch.glyphIndices[glyph.Index] = struct{}{}
		}
	}
	scratch.breakIndices = scratch.breakIndices[:0]
	for index := range lineWrapperBreakIndices(text) {
		if _, ok := scratch.glyphIndices[index]; ok {
			scratch.breakIndices = append(scratch.breakIndices, index)
		}
	}
	breakIndices := scratch.breakIndices

	var glyphs []wrapGlyph
	for runIx, run := range l.Runs {
		for glyphIx, glyph := range run.Glyphs {
			if n := len(glyphs); n > 0 && glyphs[n-1].index == glyph.Index {
				continue
			}
			glyphs = append(glyphs, wrapGlyph{
				boundary: WrapBoundary{RunIx: runIx, GlyphIx: glyphIx},
				index:    glyph.Index,
				x:        glyph.Position.X,
			})
		}
	}

	for i, item := range glyphs {
		boundary, index, x := item.boundary, item.index, item.x
		b := text[index]
		if b == '\n' {
			continue
		}

		if b != ' ' && !(currentSegment.ok && currentSegment.start <= index && index < currentSegment.end) {
			segment := sort.Search(len(breakIndices), func(j int) bool { return breakIndices[j] > index })
			if currentSegment.ok {
				lastCandidateIx = Some(boundary)
				lastCandidateX = x
			}
			start := 0
			if segment > 0 {
				start = breakIndices[segment-1]
			}
			end := len(text)
			if segment < len(breakIndices) {
				end = breakIndices[segment]
			}
			currentSegment.start, currentSegment.end, currentSegment.ok = start, end, true
		}

		nextX := l.Width
		if i+1 < len(glyphs) {
			nextX = glyphs[i+1].x
		}
		width := nextX - lastBoundaryX

		if width > wrapWidth && lastBoundary.Less(boundary) {
			if max, ok := maxLines.Get(); ok && len(boundaries) >= saturatingSub(max, 1) {
				break
			}
			if candidate, ok := lastCandidateIx.Take().Get(); ok {
				lastBoundary = candidate
				lastBoundaryX = lastCandidateX
			} else {
				lastBoundary = boundary
				lastBoundaryX = x
			}
			boundaries = append(boundaries, lastBoundary)
		}
	}
	return boundaries
}

// WrappedLineLayout is a line of text that has been wrapped to fit a given width.
type WrappedLineLayout struct {
	// UnwrappedLayout is the line layout before wrapping.
	UnwrappedLayout *LineLayout
	// WrapBoundaries are the boundaries at which the line was wrapped.
	WrapBoundaries []WrapBoundary
	// WrapWidth is the width of the line, if it was wrapped.
	WrapWidth Option[Pixels]
}

// WrapBoundary is a boundary at which a line was wrapped.
type WrapBoundary struct {
	// RunIx is the index of the run just before the line was wrapped.
	RunIx int
	// GlyphIx is the index of the glyph just before the line was wrapped.
	GlyphIx int
}

// Less orders boundaries by run, then glyph.
func (b WrapBoundary) Less(other WrapBoundary) bool {
	if b.RunIx != other.RunIx {
		return b.RunIx < other.RunIx
	}
	return b.GlyphIx < other.GlyphIx
}

var emptyLineLayout LineLayout

func (w *WrappedLineLayout) unwrapped() *LineLayout {
	if w.UnwrappedLayout == nil {
		return &emptyLineLayout
	}
	return w.UnwrappedLayout
}

// Len returns the length of the underlying text in UTF-8 bytes.
func (w *WrappedLineLayout) Len() int {
	return w.unwrapped().Len
}

// Width returns the width of the line in pixels, whether or not it was wrapped.
func (w *WrappedLineLayout) Width() Pixels {
	return min(w.WrapWidth.UnwrapOr(PixelsMax), w.unwrapped().Width)
}

// Size returns the size of the whole wrapped text for the line height.
func (w *WrappedLineLayout) Size(lineHeight Pixels) Size[Pixels] {
	return Size[Pixels]{
		Width:  w.Width(),
		Height: lineHeight * Pixels(float32(len(w.WrapBoundaries)+1)),
	}
}

// Ascent returns the ascent of a line in this layout.
func (w *WrappedLineLayout) Ascent() Pixels {
	return w.unwrapped().Ascent
}

// Descent returns the descent of a line in this layout.
func (w *WrappedLineLayout) Descent() Pixels {
	return w.unwrapped().Descent
}

// FontSize returns the font size of this layout.
func (w *WrappedLineLayout) FontSize() Pixels {
	return w.unwrapped().FontSize
}

// Runs returns the runs in this layout, without wrapping.
func (w *WrappedLineLayout) Runs() []ShapedRun {
	return w.unwrapped().Runs
}

// IndexForPosition returns the index for a position in this layout at the line height;
// ok false carries upstream's Err index.
func (w *WrappedLineLayout) IndexForPosition(position Point[Pixels], lineHeight Pixels) (int, bool) {
	return w.indexForPosition(position, lineHeight, false)
}

// ClosestIndexForPosition returns the character boundary closest to the position in this
// layout at the line height; ok false carries upstream's Err index.
func (w *WrappedLineLayout) ClosestIndexForPosition(position Point[Pixels], lineHeight Pixels) (int, bool) {
	return w.indexForPosition(position, lineHeight, true)
}

func floatToUsize(v float32) int {
	if !(v > 0) {
		return 0
	}
	if v >= math.MaxInt {
		return math.MaxInt
	}
	return int(v)
}

func (w *WrappedLineLayout) indexForPosition(position Point[Pixels], lineHeight Pixels, closest bool) (int, bool) {
	unwrapped := w.unwrapped()
	wrappedLineIx := floatToUsize(float32(position.Y / lineHeight))

	var wrappedLineStartIndex int
	var wrappedLineStartX Pixels
	if wrappedLineIx > 0 {
		if wrappedLineIx-1 >= len(w.WrapBoundaries) {
			return 0, false
		}
		lineStartBoundary := w.WrapBoundaries[wrappedLineIx-1]
		glyph := unwrapped.Runs[lineStartBoundary.RunIx].Glyphs[lineStartBoundary.GlyphIx]
		wrappedLineStartIndex = glyph.Index
		wrappedLineStartX = glyph.Position.X
	} else {
		wrappedLineStartIndex = 0
		wrappedLineStartX = PixelsZero
	}

	var wrappedLineEndIndex int
	var wrappedLineEndX Pixels
	if wrappedLineIx < len(w.WrapBoundaries) {
		nextWrapBoundary := w.WrapBoundaries[wrappedLineIx]
		glyph := unwrapped.Runs[nextWrapBoundary.RunIx].Glyphs[nextWrapBoundary.GlyphIx]
		wrappedLineEndIndex = glyph.Index
		wrappedLineEndX = glyph.Position.X
	} else {
		wrappedLineEndIndex = unwrapped.Len
		wrappedLineEndX = unwrapped.Width
	}

	positionInUnwrappedLine := position
	positionInUnwrappedLine.X += wrappedLineStartX
	if positionInUnwrappedLine.X < wrappedLineStartX {
		return wrappedLineStartIndex, false
	} else if positionInUnwrappedLine.X >= wrappedLineEndX {
		return wrappedLineEndIndex, false
	}
	if closest {
		return unwrapped.ClosestIndexForX(positionInUnwrappedLine.X), true
	}
	if index, ok := unwrapped.IndexForX(positionInUnwrappedLine.X); ok {
		return index, true
	}
	return wrappedLineEndIndex, false
}

// PositionForIndex returns the pixel position of the byte index.
func (w *WrappedLineLayout) PositionForIndex(index int, lineHeight Pixels) (Point[Pixels], bool) {
	unwrapped := w.unwrapped()
	lineStartIx := 0
	lineEndIndices := make([]int, 0, len(w.WrapBoundaries)+1)
	for _, wrapBoundary := range w.WrapBoundaries {
		lineEndIndices = append(lineEndIndices, unwrapped.Runs[wrapBoundary.RunIx].Glyphs[wrapBoundary.GlyphIx].Index)
	}
	lineEndIndices = append(lineEndIndices, w.Len())
	for ix, lineEndIx := range lineEndIndices {
		lineY := Pixels(float32(ix)) * lineHeight
		if index < lineStartIx {
			break
		} else if index > lineEndIx {
			lineStartIx = lineEndIx
			continue
		}
		lineStartX := unwrapped.XForIndex(lineStartIx)
		x := unwrapped.XForIndex(index) - lineStartX
		return NewPoint(x, lineY), true
	}
	return Point[Pixels]{}, false
}

// LineLayoutCache caches line layouts across two frames; a window owns one and uses it from
// its thread only.
type LineLayoutCache struct {
	state                lineLayoutCacheState
	wrapScratch          wrapScratch
	platformTextSystem   PlatformTextSystem
	fontGeneration       *atomic.Uint64
	cachedFontGeneration atomic.Uint64
}

type wrapScratch struct {
	glyphIndices map[int]struct{}
	breakIndices []int
}

type lineLayoutCacheState struct {
	previousFrame frameCache
	currentFrame  frameCache
}

type frameCache struct {
	lines            map[lineLayoutCacheKey]*LineLayout
	wrappedLines     map[lineLayoutCacheKey]*WrappedLineLayout
	usedLines        []lineLayoutCacheKey
	usedWrappedLines []lineLayoutCacheKey
}

func newFrameCache() frameCache {
	return frameCache{
		lines:        map[lineLayoutCacheKey]*LineLayout{},
		wrappedLines: map[lineLayoutCacheKey]*WrappedLineLayout{},
	}
}

func newLineLayoutCacheState() lineLayoutCacheState {
	return lineLayoutCacheState{previousFrame: newFrameCache(), currentFrame: newFrameCache()}
}

// LineLayoutIndex is a position in the line layout cache's current frame.
type LineLayoutIndex struct {
	fontGeneration    uint64
	linesIndex        int
	wrappedLinesIndex int
}

// NewLineLayoutCache creates a cache over the platform text system; fontGeneration advances
// whenever TextSystem.AddFonts changes the font database.
func NewLineLayoutCache(platformTextSystem PlatformTextSystem, fontGeneration *atomic.Uint64) *LineLayoutCache {
	c := &LineLayoutCache{
		state:              newLineLayoutCacheState(),
		platformTextSystem: platformTextSystem,
		fontGeneration:     fontGeneration,
	}
	c.cachedFontGeneration.Store(fontGeneration.Load())
	return c
}

func (c *LineLayoutCache) layoutIndex() LineLayoutIndex {
	fontGeneration := c.clearIfFontGenerationChanged()
	frame := &c.state.currentFrame
	return LineLayoutIndex{
		fontGeneration:    fontGeneration,
		linesIndex:        len(frame.usedLines),
		wrappedLinesIndex: len(frame.usedWrappedLines),
	}
}

func (c *LineLayoutCache) reuseLayouts(start, end LineLayoutIndex) {
	fontGeneration := c.clearIfFontGenerationChanged()
	if start.fontGeneration != fontGeneration || end.fontGeneration != fontGeneration {
		return
	}
	previousFrame := &c.state.previousFrame
	currentFrame := &c.state.currentFrame
	for _, key := range previousFrame.usedLines[start.linesIndex:end.linesIndex] {
		if line, ok := previousFrame.lines[key]; ok {
			delete(previousFrame.lines, key)
			currentFrame.lines[key] = line
		}
		currentFrame.usedLines = append(currentFrame.usedLines, key)
	}
	for _, key := range previousFrame.usedWrappedLines[start.wrappedLinesIndex:end.wrappedLinesIndex] {
		if line, ok := previousFrame.wrappedLines[key]; ok {
			delete(previousFrame.wrappedLines, key)
			currentFrame.wrappedLines[key] = line
		}
		currentFrame.usedWrappedLines = append(currentFrame.usedWrappedLines, key)
	}
}

func (c *LineLayoutCache) truncateLayouts(index LineLayoutIndex) {
	fontGeneration := c.clearIfFontGenerationChanged()
	if index.fontGeneration != fontGeneration {
		return
	}
	currentFrame := &c.state.currentFrame
	currentFrame.usedLines = currentFrame.usedLines[:min(index.linesIndex, len(currentFrame.usedLines))]
	currentFrame.usedWrappedLines = currentFrame.usedWrappedLines[:min(index.wrappedLinesIndex, len(currentFrame.usedWrappedLines))]
}

func (c *LineLayoutCache) finishFrame() {
	c.clearIfFontGenerationChanged()
	c.state.previousFrame, c.state.currentFrame = c.state.currentFrame, c.state.previousFrame
	currentFrame := &c.state.currentFrame
	clear(currentFrame.lines)
	clear(currentFrame.wrappedLines)
	currentFrame.usedLines = currentFrame.usedLines[:0]
	currentFrame.usedWrappedLines = currentFrame.usedWrappedLines[:0]
}

func (c *LineLayoutCache) layoutWrappedLine(text string, fontSize Pixels, runs []FontRun, wrapWidth Option[Pixels], maxLines Option[int]) *WrappedLineLayout {
	c.clearIfFontGenerationChanged()
	key := newLineLayoutCacheKey(text, fontSize, runs, wrapWidth, None[Pixels]())

	if layout, ok := c.state.currentFrame.wrappedLines[key]; ok {
		return layout
	}
	if layout, ok := c.state.previousFrame.wrappedLines[key]; ok {
		delete(c.state.previousFrame.wrappedLines, key)
		c.state.currentFrame.wrappedLines[key] = layout
		c.state.currentFrame.usedWrappedLines = append(c.state.currentFrame.usedWrappedLines, key)
		return layout
	}
	unwrappedLayout := c.layoutLine(text, fontSize, runs, None[Pixels]())
	var wrapBoundaries []WrapBoundary
	if width, ok := wrapWidth.Get(); ok {
		wrapBoundaries = unwrappedLayout.computeWrapBoundaries(text, width, maxLines, &c.wrapScratch)
	}
	layout := &WrappedLineLayout{
		UnwrappedLayout: unwrappedLayout,
		WrapBoundaries:  wrapBoundaries,
		WrapWidth:       wrapWidth,
	}
	c.state.currentFrame.wrappedLines[key] = layout
	c.state.currentFrame.usedWrappedLines = append(c.state.currentFrame.usedWrappedLines, key)
	return layout
}

func (c *LineLayoutCache) layoutLine(text string, fontSize Pixels, runs []FontRun, forceWidth Option[Pixels]) *LineLayout {
	c.clearIfFontGenerationChanged()
	key := newLineLayoutCacheKey(text, fontSize, runs, None[Pixels](), forceWidth)

	if layout, ok := c.state.currentFrame.lines[key]; ok {
		return layout
	}
	previousFrame := &c.state.previousFrame
	currentFrame := &c.state.currentFrame
	if layout, ok := previousFrame.lines[key]; ok {
		delete(previousFrame.lines, key)
		currentFrame.lines[key] = layout
		currentFrame.usedLines = append(currentFrame.usedLines, key)
		return layout
	}
	layout := c.platformTextSystem.LayoutLine(text, fontSize, runs)
	if width, ok := forceWidth.Get(); ok {
		applyForceWidthToLayout(&layout, width)
	}
	currentFrame.lines[key] = &layout
	currentFrame.usedLines = append(currentFrame.usedLines, key)
	return &layout
}

func (c *LineLayoutCache) clearIfFontGenerationChanged() uint64 {
	fontGeneration := c.fontGeneration.Load()
	if c.cachedFontGeneration.Load() == fontGeneration {
		return fontGeneration
	}
	c.state = newLineLayoutCacheState()
	c.cachedFontGeneration.Store(fontGeneration)
	return fontGeneration
}

func applyForceWidthToLayout(layout *LineLayout, forceWidth Pixels) {
	glyphPos := 0
	lastBaseShapedX := Px(float32(math.Inf(-1)))
	lastBaseActualX := Px(0.)
	for r := range layout.Runs {
		glyphs := layout.Runs[r].Glyphs
		for g := range glyphs {
			glyph := &glyphs[g]
			shapedX := glyph.Position.X
			if shapedX > lastBaseShapedX+forceWidth*0.5 {
				forcedX := Pixels(float32(glyphPos)) * forceWidth
				if (shapedX - forcedX).Abs() > Px(1.) {
					glyph.Position.X = forcedX
				}
				lastBaseShapedX = shapedX
				lastBaseActualX = glyph.Position.X
				glyphPos++
			} else {
				glyph.Position.X = lastBaseActualX + (shapedX - lastBaseShapedX)
			}
		}
	}
}

// FontRun is a run of text in a single font.
type FontRun struct {
	// Len is the length of the run in UTF-8 bytes.
	Len int
	// FontID is the font of the run.
	FontID FontID
}

type lineLayoutCacheKey struct {
	text       string
	fontSize   Pixels
	runs       string
	wrapWidth  Option[Pixels]
	forceWidth Option[Pixels]
}

func newLineLayoutCacheKey(text string, fontSize Pixels, runs []FontRun, wrapWidth, forceWidth Option[Pixels]) lineLayoutCacheKey {
	encoded := make([]byte, 0, len(runs)*16)
	for _, run := range runs {
		encoded = binary.LittleEndian.AppendUint64(encoded, uint64(run.Len))
		encoded = binary.LittleEndian.AppendUint64(encoded, uint64(run.FontID))
	}
	return lineLayoutCacheKey{
		text:       text,
		fontSize:   fontSize,
		runs:       string(encoded),
		wrapWidth:  canonicalOptionPixels(wrapWidth),
		forceWidth: canonicalOptionPixels(forceWidth),
	}
}

func canonicalOptionPixels(o Option[Pixels]) Option[Pixels] {
	if v, ok := o.Get(); ok {
		if v == 0 {
			return Some(Pixels(0))
		}
		return o
	}
	return None[Pixels]()
}
