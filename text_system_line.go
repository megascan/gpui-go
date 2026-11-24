package gpui

import (
	"slices"
	"unicode/utf8"
)

// GlyphRasterData is pre-computed glyph data for painting without per-glyph cache lookups.
type GlyphRasterData struct {
	// Bounds are the raster bounds for each glyph, in paint order.
	Bounds []Bounds[DevicePixels]
	// Params are the render params for each glyph (needed for sprite atlas lookup).
	Params []RenderGlyphParams
}

// DecorationRun sets the text decoration for a run of text.
type DecorationRun struct {
	// Len is the length of the run in UTF-8 bytes.
	Len uint32
	// Color is the color for this run.
	Color Hsla
	// BackgroundColor is the background color for this run.
	BackgroundColor Option[Hsla]
	// Underline is the underline style for this run.
	Underline Option[UnderlineStyle]
	// Strikethrough is the strikethrough style for this run.
	Strikethrough Option[StrikethroughStyle]
}

// ShapedLine is a line of text that has been shaped and decorated; it embeds its layout
// (upstream's Deref). The layout is shared and never mutated.
type ShapedLine struct {
	*LineLayout
	// Text is the text that was shaped for this line.
	Text           SharedString
	decorationRuns []DecorationRun
}

// UnderlineHandler receives each underline stroke of a painted line: the byte range it covers,
// its origin, width and style.
type UnderlineHandler func(rng Range, origin Point[Pixels], width Pixels, style *UnderlineStyle, window *Window)

func defaultUnderlineHandler(_ Range, origin Point[Pixels], width Pixels, style *UnderlineStyle, window *Window) {
	window.PaintUnderline(origin, width, *style)
}

func textIsCharBoundary(s string, index int) bool {
	if index == 0 || index == len(s) {
		return true
	}
	if index > len(s) || index < 0 {
		return false
	}
	return utf8.RuneStart(s[index])
}

// Cursor returns a forward-only cursor for this shaped line.
func (l ShapedLine) Cursor() *ShapedLineCursor {
	if l.Len() != len(l.Text) {
		panic("assertion `left == right` failed: cannot split a shaped line with an adjusted length")
	}
	byteOrdered := true
	prev := 0
	first := true
outer:
	for _, run := range l.LineLayout.Runs {
		for _, glyph := range run.Glyphs {
			if !first && glyph.Index < prev {
				byteOrdered = false
				break outer
			}
			prev = glyph.Index
			first = false
		}
	}
	cursor := &ShapedLineCursor{line: l}
	if !byteOrdered {
		cursor.unorderedRemainder = Some(l)
	}
	return cursor
}

// Len returns the length of the line in UTF-8 bytes.
func (l ShapedLine) Len() int {
	if l.LineLayout == nil {
		return 0
	}
	return l.LineLayout.Len
}

// Width returns the glyph advance width of the shaped line in pixels.
func (l ShapedLine) Width() Pixels {
	if l.LineLayout == nil {
		return 0
	}
	return l.LineLayout.Width
}

// WithLen overrides the length, useful when rendering text a as text b (e.g. invisibles).
func (l ShapedLine) WithLen(length int) ShapedLine {
	layout := l.LineLayout
	l.LineLayout = &LineLayout{
		FontSize: layout.FontSize,
		Width:    layout.Width,
		Ascent:   layout.Ascent,
		Descent:  layout.Descent,
		Runs:     slices.Clone(layout.Runs),
		Len:      length,
	}
	return l
}

// Paint paints the line of text to the window.
func (l ShapedLine) Paint(origin Point[Pixels], lineHeight Pixels, align TextAlign, alignWidth Option[Pixels], window *Window, cx *App) error {
	return l.PaintWithUnderlineHandler(origin, lineHeight, align, alignWidth, window, cx, defaultUnderlineHandler)
}

// PaintWithUnderlineHandler paints the line with a handler for each underline.
func (l ShapedLine) PaintWithUnderlineHandler(origin Point[Pixels], lineHeight Pixels, align TextAlign, alignWidth Option[Pixels], window *Window, cx *App, paintUnderline UnderlineHandler) error {
	return paintLine(origin, l.LineLayout, lineHeight, align, alignWidth, l.decorationRuns, nil, window, cx, paintUnderline)
}

// PaintBackground paints the background of the line to the window.
func (l ShapedLine) PaintBackground(origin Point[Pixels], lineHeight Pixels, align TextAlign, alignWidth Option[Pixels], window *Window, cx *App) error {
	if !hasBackground(l.decorationRuns) {
		return nil
	}
	return paintLineBackground(origin, l.LineLayout, lineHeight, align, alignWidth, l.decorationRuns, nil, window, cx)
}

// SplitAt splits this shaped line at a byte index, returning (prefix, suffix). Decoration runs
// are partitioned at the boundary; a run that straddles it is split in two.
func (l ShapedLine) SplitAt(byteIndex int) (ShapedLine, ShapedLine) {
	leftLayout, rightLayout := l.LineLayout.SplitAt(byteIndex)

	var leftDecorations, rightDecorations []DecorationRun
	decorationOffset := uint32(0)
	splitPoint := uint32(byteIndex)

	for _, decoration := range l.decorationRuns {
		runEnd := decorationOffset + decoration.Len

		if runEnd <= splitPoint {
			leftDecorations = append(leftDecorations, decoration)
		} else if decorationOffset >= splitPoint {
			rightDecorations = append(rightDecorations, decoration)
		} else {
			left := decoration
			left.Len = splitPoint - decorationOffset
			right := decoration
			right.Len = runEnd - splitPoint
			leftDecorations = append(leftDecorations, left)
			rightDecorations = append(rightDecorations, right)
		}

		decorationOffset = runEnd
	}

	leftText := l.Text
	if byteIndex != len(l.Text) {
		leftText = l.Text[:byteIndex]
	}
	rightText := l.Text
	if byteIndex != 0 {
		rightText = l.Text[byteIndex:]
	}

	return ShapedLine{LineLayout: &leftLayout, Text: leftText, decorationRuns: leftDecorations},
		ShapedLine{LineLayout: &rightLayout, Text: rightText, decorationRuns: rightDecorations}
}

// ShapedLineCursor incrementally splits a ShapedLine at increasing UTF-8 byte boundaries.
type ShapedLineCursor struct {
	line               ShapedLine
	unorderedRemainder Option[ShapedLine]
	byteIndex          int
	runIndex           int
	glyphIndex         int
	decorationIndex    int
	decorationOffset   uint32
	xOffset            Pixels
}

// TakeUntil takes the bytes since the previous boundary. It panics if the boundary precedes the
// previous one, exceeds the line's length, or falls inside a UTF-8 character.
func (c *ShapedLineCursor) TakeUntil(byteIndex int) ShapedLine {
	if byteIndex < c.byteIndex {
		panic("split boundary moved backwards")
	}
	if byteIndex > c.line.Len() {
		panic("split boundary exceeds line length")
	}
	if !textIsCharBoundary(string(c.line.Text), byteIndex) {
		panic("split boundary is not a UTF-8 character boundary")
	}
	previousIndex := c.byteIndex
	previousX := c.xOffset
	if remainder, ok := c.unorderedRemainder.Get(); ok {
		piece, rest := remainder.SplitAt(byteIndex - previousIndex)
		c.unorderedRemainder = Some(rest)
		c.byteIndex = byteIndex
		c.xOffset = c.line.LineLayout.XForIndex(byteIndex)
		return piece
	}
	var runs []ShapedRun
	nextX := c.line.LineLayout.Width
	layoutRuns := c.line.LineLayout.Runs
	for c.runIndex < len(layoutRuns) {
		run := &layoutRuns[c.runIndex]
		start := c.glyphIndex
		for c.glyphIndex < len(run.Glyphs) {
			if run.Glyphs[c.glyphIndex].Index >= byteIndex {
				break
			}
			c.glyphIndex++
		}
		end := c.glyphIndex
		if start < end {
			glyphs := make([]ShapedGlyph, 0, end-start)
			for _, glyph := range run.Glyphs[start:end] {
				glyphs = append(glyphs, ShapedGlyph{
					ID:       glyph.ID,
					Position: NewPoint(glyph.Position.X-previousX, glyph.Position.Y),
					Index:    glyph.Index - previousIndex,
					IsEmoji:  glyph.IsEmoji,
				})
			}
			runs = append(runs, ShapedRun{FontID: run.FontID, Glyphs: glyphs})
		}
		if c.glyphIndex < len(run.Glyphs) {
			nextX = run.Glyphs[c.glyphIndex].Position.X
			break
		}
		c.runIndex++
		c.glyphIndex = 0
	}
	var decorations []DecorationRun
	boundary := uint32(byteIndex)
	for c.decorationIndex < len(c.line.decorationRuns) {
		decoration := c.line.decorationRuns[c.decorationIndex]
		if !(c.decorationOffset < boundary || (decoration.Len == 0 && c.decorationOffset == boundary)) {
			break
		}
		end := c.decorationOffset + decoration.Len
		start := max(c.decorationOffset, uint32(previousIndex))
		length := min(end, boundary) - start
		if length > 0 || decoration.Len == 0 {
			piece := decoration
			piece.Len = length
			decorations = append(decorations, piece)
		}
		if end <= boundary {
			c.decorationIndex++
			c.decorationOffset = end
		} else {
			break
		}
	}
	c.byteIndex = byteIndex
	c.xOffset = nextX
	return ShapedLine{
		LineLayout: &LineLayout{
			FontSize: c.line.LineLayout.FontSize,
			Width:    nextX - previousX,
			Ascent:   c.line.LineLayout.Ascent,
			Descent:  c.line.LineLayout.Descent,
			Runs:     runs,
			Len:      byteIndex - previousIndex,
		},
		Text:           c.line.Text[previousIndex:byteIndex],
		decorationRuns: decorations,
	}
}

// XOffset returns the original line's x position at the current boundary.
func (c *ShapedLineCursor) XOffset() Pixels {
	return c.xOffset
}

// Paint paints this layout to the window, using decorationRuns to color glyphs and draw
// underlines and strikethroughs.
func (l *LineLayout) Paint(origin Point[Pixels], lineHeight Pixels, align TextAlign, alignWidth Option[Pixels], decorationRuns []DecorationRun, window *Window, cx *App) error {
	return paintLine(origin, l, lineHeight, align, alignWidth, decorationRuns, nil, window, cx, defaultUnderlineHandler)
}

// PaintBackground paints the background of this layout to the window, using decorationRuns to
// determine background colors.
func (l *LineLayout) PaintBackground(origin Point[Pixels], lineHeight Pixels, align TextAlign, alignWidth Option[Pixels], decorationRuns []DecorationRun, window *Window, cx *App) error {
	if !hasBackground(decorationRuns) {
		return nil
	}
	return paintLineBackground(origin, l, lineHeight, align, alignWidth, decorationRuns, nil, window, cx)
}

// WrappedLine is a line of text that has been shaped, decorated and wrapped; it embeds its
// layout (upstream's Deref). The layout is shared and never mutated.
type WrappedLine struct {
	*WrappedLineLayout
	// Text is the text that was shaped for this line.
	Text           SharedString
	decorationRuns []DecorationRun
}

// Len returns the length of the underlying, unwrapped layout in UTF-8 bytes.
func (l WrappedLine) Len() int {
	return l.WrappedLineLayout.Len()
}

func (l WrappedLine) alignWidth(bounds Option[Bounds[Pixels]]) Option[Pixels] {
	if b, ok := bounds.Get(); ok {
		return Some(b.Size.Width)
	}
	return l.WrappedLineLayout.WrapWidth
}

// Paint paints this line of text to the window.
func (l WrappedLine) Paint(origin Point[Pixels], lineHeight Pixels, align TextAlign, bounds Option[Bounds[Pixels]], window *Window, cx *App) error {
	alignWidth := l.alignWidth(bounds)
	return paintLine(origin, l.UnwrappedLayout, lineHeight, align, alignWidth, l.decorationRuns, l.WrapBoundaries, window, cx, defaultUnderlineHandler)
}

// PaintBackground paints the background of this line of text to the window.
func (l WrappedLine) PaintBackground(origin Point[Pixels], lineHeight Pixels, align TextAlign, bounds Option[Bounds[Pixels]], window *Window, cx *App) error {
	alignWidth := l.alignWidth(bounds)
	if !hasBackground(l.decorationRuns) {
		return nil
	}
	return paintLineBackground(origin, l.UnwrappedLayout, lineHeight, align, alignWidth, l.decorationRuns, l.WrapBoundaries, window, cx)
}

type pendingUnderline struct {
	origin Point[Pixels]
	style  UnderlineStyle
	rng    Range
}

type pendingStrikethrough struct {
	origin Point[Pixels]
	style  StrikethroughStyle
}

type pendingBackground struct {
	origin Point[Pixels]
	color  Hsla
}

func paintLine(origin Point[Pixels], layout *LineLayout, lineHeight Pixels, align TextAlign, alignWidth Option[Pixels], decorationRuns []DecorationRun, wrapBoundaries []WrapBoundary, window *Window, cx *App, paintUnderline UnderlineHandler) error {
	return window.PaintLayer(func(window *Window) error {
		paddingTop := (lineHeight - layout.Ascent - layout.Descent) / 2.
		baselineOffset := NewPoint(Px(0.), paddingTop+layout.Ascent)
		underlineYOffset := UnderlineYOffset(lineHeight, layout.Ascent, layout.Descent)
		decorationIx := 0
		wrapIx := 0
		peekWrap := func() *WrapBoundary {
			if wrapIx < len(wrapBoundaries) {
				return &wrapBoundaries[wrapIx]
			}
			return nil
		}
		runEnd := 0
		color := Black()
		var currentUnderline *pendingUnderline
		var currentStrikethrough *pendingStrikethrough
		textSystem := cx.TextSystem()
		width := alignWidth.UnwrapOr(layout.Width)
		glyphOrigin := NewPoint(alignedOriginX(origin, width, Px(0.), align, layout, peekWrap()), origin.Y)
		prevGlyphPosition := Point[Pixels]{}
		maxGlyphSize := NewSize(Px(0.), Px(0.))
		firstGlyphX := origin.X
		for runIx := range layout.Runs {
			run := &layout.Runs[runIx]
			maxGlyphSize = textSystem.BoundingBox(run.FontID, layout.FontSize).Size

			for glyphIx := range run.Glyphs {
				glyph := &run.Glyphs[glyphIx]
				glyphOrigin.X += glyph.Position.X - prevGlyphPosition.X
				if glyphIx == 0 && runIx == 0 {
					firstGlyphX = glyphOrigin.X
				}

				if wrap := peekWrap(); wrap != nil && *wrap == (WrapBoundary{RunIx: runIx, GlyphIx: glyphIx}) {
					wrapIx++
					if currentUnderline != nil {
						if glyphOrigin.X == currentUnderline.origin.X {
							currentUnderline.origin.X -= maxGlyphSize.Width.Half()
						}
						paintUnderline(currentUnderline.rng, currentUnderline.origin, glyphOrigin.X-currentUnderline.origin.X, &currentUnderline.style, window)
						if glyph.Index < runEnd {
							currentUnderline.origin.X = origin.X
							currentUnderline.origin.Y += lineHeight
						} else {
							currentUnderline = nil
						}
					}
					if currentStrikethrough != nil {
						if glyphOrigin.X == currentStrikethrough.origin.X {
							currentStrikethrough.origin.X -= maxGlyphSize.Width.Half()
						}
						window.PaintStrikethrough(currentStrikethrough.origin, glyphOrigin.X-currentStrikethrough.origin.X, currentStrikethrough.style)
						if glyph.Index < runEnd {
							currentStrikethrough.origin.X = origin.X
							currentStrikethrough.origin.Y += lineHeight
						} else {
							currentStrikethrough = nil
						}
					}

					glyphOrigin.X = alignedOriginX(origin, width, glyph.Position.X, align, layout, peekWrap())
					glyphOrigin.Y += lineHeight
				}
				prevGlyphPosition = glyph.Position

				var finishedUnderline *pendingUnderline
				var finishedStrikethrough *pendingStrikethrough
				if glyph.Index >= runEnd {
					var styleRun *DecorationRun
					next := func() *DecorationRun {
						if decorationIx < len(decorationRuns) {
							decorationIx++
							return &decorationRuns[decorationIx-1]
						}
						return nil
					}
					styleRun = next()

					for styleRun != nil {
						if glyph.Index < runEnd+int(styleRun.Len) {
							break
						}
						runEnd += int(styleRun.Len)
						styleRun = next()
					}

					if styleRun != nil {
						styleRunStart := runEnd
						if currentUnderline != nil {
							if styleRun.Underline != Some(currentUnderline.style) {
								finishedUnderline = currentUnderline
								currentUnderline = nil
							} else {
								currentUnderline.rng.End = styleRunStart + int(styleRun.Len)
							}
						}
						if runUnderline, ok := styleRun.Underline.Get(); ok && currentUnderline == nil {
							currentUnderline = &pendingUnderline{
								origin: NewPoint(glyphOrigin.X, glyphOrigin.Y+underlineYOffset),
								style: UnderlineStyle{
									Color:     Some(runUnderline.Color.UnwrapOr(styleRun.Color)),
									Thickness: runUnderline.Thickness,
									Wavy:      runUnderline.Wavy,
								},
								rng: Range{Start: styleRunStart, End: styleRunStart + int(styleRun.Len)},
							}
						}
						if currentStrikethrough != nil && styleRun.Strikethrough != Some(currentStrikethrough.style) {
							finishedStrikethrough = currentStrikethrough
							currentStrikethrough = nil
						}
						if runStrikethrough, ok := styleRun.Strikethrough.Get(); ok && currentStrikethrough == nil {
							currentStrikethrough = &pendingStrikethrough{
								origin: NewPoint(glyphOrigin.X, glyphOrigin.Y+(((layout.Ascent*0.5)+baselineOffset.Y)*0.5)),
								style: StrikethroughStyle{
									Color:     Some(runStrikethrough.Color.UnwrapOr(styleRun.Color)),
									Thickness: runStrikethrough.Thickness,
								},
							}
						}

						runEnd += int(styleRun.Len)
						color = styleRun.Color
					} else {
						runEnd = layout.Len
						finishedUnderline = currentUnderline
						currentUnderline = nil
						finishedStrikethrough = currentStrikethrough
						currentStrikethrough = nil
					}
				}

				if finishedUnderline != nil {
					underlineOrigin := finishedUnderline.origin
					if underlineOrigin.X == glyphOrigin.X {
						underlineOrigin.X -= maxGlyphSize.Width.Half()
					}
					paintUnderline(finishedUnderline.rng, underlineOrigin, glyphOrigin.X-underlineOrigin.X, &finishedUnderline.style, window)
				}

				if finishedStrikethrough != nil {
					strikethroughOrigin := finishedStrikethrough.origin
					if strikethroughOrigin.X == glyphOrigin.X {
						strikethroughOrigin.X -= maxGlyphSize.Width.Half()
					}
					window.PaintStrikethrough(strikethroughOrigin, glyphOrigin.X-strikethroughOrigin.X, finishedStrikethrough.style)
				}

				maxGlyphBounds := Bounds[Pixels]{Origin: glyphOrigin, Size: maxGlyphSize}

				contentMask := window.ContentMask()
				if maxGlyphBounds.Intersects(contentMask.Bounds) {
					verticalOffset := NewPoint(Px(0.), glyph.Position.Y)
					if glyph.IsEmoji {
						if err := window.PaintEmoji(glyphOrigin.Add(baselineOffset).Add(verticalOffset), run.FontID, glyph.ID, layout.FontSize); err != nil {
							return err
						}
					} else {
						if err := window.PaintGlyph(glyphOrigin.Add(baselineOffset).Add(verticalOffset), run.FontID, glyph.ID, layout.FontSize, color); err != nil {
							return err
						}
					}
				}
			}
		}

		lastLineEndX := firstGlyphX + layout.Width
		if n := len(wrapBoundaries); n > 0 {
			boundary := wrapBoundaries[n-1]
			lastLineEndX -= layout.Runs[boundary.RunIx].Glyphs[boundary.GlyphIx].Position.X
		}

		if currentUnderline != nil {
			underlineStart := currentUnderline.origin
			if lastLineEndX == underlineStart.X {
				underlineStart.X -= maxGlyphSize.Width.Half()
			}
			paintUnderline(currentUnderline.rng, underlineStart, lastLineEndX-underlineStart.X, &currentUnderline.style, window)
		}

		if currentStrikethrough != nil {
			strikethroughStart := currentStrikethrough.origin
			if lastLineEndX == strikethroughStart.X {
				strikethroughStart.X -= maxGlyphSize.Width.Half()
			}
			window.PaintStrikethrough(strikethroughStart, lastLineEndX-strikethroughStart.X, currentStrikethrough.style)
		}

		return nil
	})
}

func hasBackground(decorationRuns []DecorationRun) bool {
	for i := range decorationRuns {
		if decorationRuns[i].BackgroundColor.IsSome() {
			return true
		}
	}
	return false
}

func paintLineBackground(origin Point[Pixels], layout *LineLayout, lineHeight Pixels, align TextAlign, alignWidth Option[Pixels], decorationRuns []DecorationRun, wrapBoundaries []WrapBoundary, window *Window, cx *App) error {
	return window.PaintLayer(func(window *Window) error {
		decorationIx := 0
		wrapIx := 0
		peekWrap := func() *WrapBoundary {
			if wrapIx < len(wrapBoundaries) {
				return &wrapBoundaries[wrapIx]
			}
			return nil
		}
		runEnd := 0
		var currentBackground *pendingBackground
		textSystem := cx.TextSystem()
		width := alignWidth.UnwrapOr(layout.Width)
		glyphOrigin := NewPoint(alignedOriginX(origin, width, Px(0.), align, layout, peekWrap()), origin.Y)
		prevGlyphPosition := Point[Pixels]{}
		maxGlyphSize := NewSize(Px(0.), Px(0.))
		for runIx := range layout.Runs {
			run := &layout.Runs[runIx]
			maxGlyphSize = textSystem.BoundingBox(run.FontID, layout.FontSize).Size

			for glyphIx := range run.Glyphs {
				glyph := &run.Glyphs[glyphIx]
				glyphOrigin.X += glyph.Position.X - prevGlyphPosition.X

				if wrap := peekWrap(); wrap != nil && *wrap == (WrapBoundary{RunIx: runIx, GlyphIx: glyphIx}) {
					wrapIx++
					if currentBackground != nil {
						if glyphOrigin.X == currentBackground.origin.X {
							currentBackground.origin.X -= maxGlyphSize.Width.Half()
						}
						window.PaintQuad(NewFill(Bounds[Pixels]{
							Origin: currentBackground.origin,
							Size:   NewSize(glyphOrigin.X-currentBackground.origin.X, lineHeight),
						}, currentBackground.color.ToBackground()))
						if glyph.Index < runEnd {
							currentBackground.origin.X = origin.X
							currentBackground.origin.Y += lineHeight
						} else {
							currentBackground = nil
						}
					}

					glyphOrigin.X = alignedOriginX(origin, width, glyph.Position.X, align, layout, peekWrap())
					glyphOrigin.Y += lineHeight
				}
				prevGlyphPosition = glyph.Position

				var finishedBackground *pendingBackground
				if glyph.Index >= runEnd {
					next := func() *DecorationRun {
						if decorationIx < len(decorationRuns) {
							decorationIx++
							return &decorationRuns[decorationIx-1]
						}
						return nil
					}
					styleRun := next()

					for styleRun != nil {
						if glyph.Index < runEnd+int(styleRun.Len) {
							break
						}
						runEnd += int(styleRun.Len)
						styleRun = next()
					}

					if styleRun != nil {
						if currentBackground != nil && styleRun.BackgroundColor != Some(currentBackground.color) {
							finishedBackground = currentBackground
							currentBackground = nil
						}
						if runBackground, ok := styleRun.BackgroundColor.Get(); ok && currentBackground == nil {
							currentBackground = &pendingBackground{origin: NewPoint(glyphOrigin.X, glyphOrigin.Y), color: runBackground}
						}
						runEnd += int(styleRun.Len)
					} else {
						runEnd = layout.Len
						finishedBackground = currentBackground
						currentBackground = nil
					}
				}

				if finishedBackground != nil {
					backgroundOrigin := finishedBackground.origin
					w := glyphOrigin.X - backgroundOrigin.X
					if backgroundOrigin.X == glyphOrigin.X {
						backgroundOrigin.X -= maxGlyphSize.Width.Half()
					}
					window.PaintQuad(NewFill(Bounds[Pixels]{
						Origin: backgroundOrigin,
						Size:   NewSize(w, lineHeight),
					}, finishedBackground.color.ToBackground()))
				}
			}
		}

		lastLineEndX := origin.X + layout.Width
		if n := len(wrapBoundaries); n > 0 {
			boundary := wrapBoundaries[n-1]
			lastLineEndX -= layout.Runs[boundary.RunIx].Glyphs[boundary.GlyphIx].Position.X
		}

		if currentBackground != nil {
			backgroundOrigin := currentBackground.origin
			if lastLineEndX == backgroundOrigin.X {
				backgroundOrigin.X -= maxGlyphSize.Width.Half()
			}
			window.PaintQuad(NewFill(Bounds[Pixels]{
				Origin: backgroundOrigin,
				Size:   NewSize(lastLineEndX-backgroundOrigin.X, lineHeight),
			}, currentBackground.color.ToBackground()))
		}

		return nil
	})
}

func alignedOriginX(origin Point[Pixels], alignWidth Pixels, lastGlyphX Pixels, align TextAlign, layout *LineLayout, wrapBoundary *WrapBoundary) Pixels {
	endOfLine := layout.Width
	if wrapBoundary != nil {
		endOfLine = layout.Runs[wrapBoundary.RunIx].Glyphs[wrapBoundary.GlyphIx].Position.X
	}

	lineWidth := endOfLine - lastGlyphX

	switch align {
	case TextAlignCenter:
		return (origin.X*2.0 + alignWidth - lineWidth) / 2.0
	case TextAlignRight:
		return origin.X + alignWidth - lineWidth
	default:
		return origin.X
	}
}
