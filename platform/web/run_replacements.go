//go:build js && wasm

package web

import (
	"slices"
	"sort"

	"github.com/megascan/gpui-go"
)

type candidate struct {
	source  gpui.Range
	fontID  gpui.FontID
	isEmoji bool
	color   bool
	glyphs  *glyphSpan
}

type glyphSpan struct {
	glyphRange  gpui.Range
	position    gpui.Point[gpui.Pixels]
	endX        gpui.Pixels
	contiguous  bool
	missing     bool
	nativeColor bool
}

func collectCandidates(text string, fontRuns []gpui.FontRun, layout *gpui.LineLayout) []candidate {
	next := 0
	current := -1
	if next < len(fontRuns) {
		current = next
		next++
	}
	runEnd := 0
	if current >= 0 {
		runEnd = fontRuns[current].Len
	}
	var candidates []candidate
	for _, g := range graphemeIndices(text) {
		start := g.start
		for start >= runEnd {
			if next < len(fontRuns) {
				current = next
				next++
			} else {
				current = -1
				break
			}
			runEnd += fontRuns[current].Len
		}
		if current < 0 {
			break
		}
		run := fontRuns[current]
		for next < len(fontRuns) && fontRuns[next].FontID == run.FontID {
			runEnd += fontRuns[next].Len
			next++
		}
		end := start + len(g.text)
		if end > runEnd {
			continue
		}
		fallback, ok := ClassifyCanvasFallback(g.text)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate{
			source:  gpui.Range{Start: start, End: end},
			fontID:  run.FontID,
			isEmoji: fallback.IsEmoji,
			color:   fallback.EmojiPresentation,
		})
	}
	if len(candidates) == 0 {
		return candidates
	}

	type flatGlyph struct {
		index int
		glyph *gpui.ShapedGlyph
	}
	var flat []flatGlyph
	index := 0
	for r := range layout.Runs {
		for g := range layout.Runs[r].Glyphs {
			flat = append(flat, flatGlyph{index, &layout.Runs[r].Glyphs[g]})
			index++
		}
	}
	for i, entry := range flat {
		glyph := entry.glyph
		candidateIndex := sort.Search(len(candidates), func(c int) bool {
			return !(candidates[c].source.End <= glyph.Index)
		})
		if candidateIndex >= len(candidates) {
			continue
		}
		c := &candidates[candidateIndex]
		if !(c.source.Start <= glyph.Index && glyph.Index < c.source.End) {
			continue
		}
		if c.glyphs == nil {
			c.glyphs = &glyphSpan{
				glyphRange:  gpui.Range{Start: entry.index, End: entry.index},
				position:    glyph.Position,
				endX:        glyph.Position.X,
				contiguous:  true,
				missing:     false,
				nativeColor: true,
			}
		}
		span := c.glyphs
		span.contiguous = span.contiguous && span.glyphRange.End == entry.index
		span.glyphRange.End = entry.index + 1
		if i+1 < len(flat) {
			span.endX = flat[i+1].glyph.Position.X
		} else {
			span.endX = layout.Width
		}
		span.missing = span.missing || glyph.ID == 0
		span.nativeColor = span.nativeColor && glyph.IsEmoji
	}
	return candidates
}

type replacement struct {
	glyphRange gpui.Range
	fontID     gpui.FontID
	glyph      gpui.ShapedGlyph
	widthDelta gpui.Pixels
}

type segment struct {
	fontID gpui.FontID
	length int
}

func applyReplacements(layout *gpui.LineLayout, replacements []replacement) {
	if len(replacements) == 0 {
		return
	}

	nextReplacement := 0
	originalIndex := 0
	consumedEnd := 0
	shift := gpui.Pixels(0)
	var segmentsByRun [][]segment
	var spareBuffers [][]gpui.ShapedGlyph
	finalRunCount := 0

	runWrite := 0
	for r := range layout.Runs {
		run := layout.Runs[r]
		var segments []segment
		glyphWrite := 0
		for g := range run.Glyphs {
			glyph := run.Glyphs[g]
			index := originalIndex
			originalIndex++
			if index < consumedEnd {
				continue
			}

			var fontID gpui.FontID
			if nextReplacement < len(replacements) && replacements[nextReplacement].glyphRange.Start == index {
				repl := replacements[nextReplacement]
				nextReplacement++
				consumedEnd = repl.glyphRange.End
				glyph = repl.glyph
				glyph.Position.X += shift
				shift += repl.widthDelta
				fontID = repl.fontID
			} else {
				glyph.Position.X += shift
				fontID = run.FontID
			}

			if n := len(segments); n > 0 && segments[n-1].fontID == fontID {
				segments[n-1].length++
			} else {
				segments = append(segments, segment{fontID: fontID, length: 1})
			}
			run.Glyphs[glyphWrite] = glyph
			glyphWrite++
		}
		clear(run.Glyphs[glyphWrite:])
		run.Glyphs = run.Glyphs[:glyphWrite]

		if len(segments) == 0 {
			spareBuffers = append(spareBuffers, run.Glyphs[:0])
			continue
		}
		finalRunCount += len(segments)
		segmentsByRun = append(segmentsByRun, segments)
		layout.Runs[runWrite] = run
		runWrite++
	}
	clear(layout.Runs[runWrite:])
	layout.Runs = layout.Runs[:runWrite]
	layout.Width += shift

	sourceRunCount := len(layout.Runs)
	if finalRunCount > sourceRunCount {
		layout.Runs = slices.Grow(layout.Runs, finalRunCount-sourceRunCount)
	}
	layout.Runs = layout.Runs[:finalRunCount]
	for i := sourceRunCount; i < finalRunCount; i++ {
		layout.Runs[i] = gpui.ShapedRun{FontID: 0}
	}

	destinationEnd := finalRunCount
	for sourceIndex := sourceRunCount - 1; sourceIndex >= 0; sourceIndex-- {
		segments := segmentsByRun[sourceIndex]
		destinationStart := destinationEnd - len(segments)
		glyphEnd := len(layout.Runs[sourceIndex].Glyphs)
		for segmentIndex := len(segments) - 1; segmentIndex >= 0; segmentIndex-- {
			seg := segments[segmentIndex]
			destinationIndex := destinationStart + segmentIndex
			if segmentIndex == 0 {
				layout.Runs[sourceIndex].FontID = seg.fontID
				layout.Runs[sourceIndex], layout.Runs[destinationIndex] = layout.Runs[destinationIndex], layout.Runs[sourceIndex]
			} else {
				glyphStart := glyphEnd - seg.length
				var glyphs []gpui.ShapedGlyph
				if n := len(spareBuffers); n > 0 {
					glyphs = spareBuffers[n-1]
					spareBuffers = spareBuffers[:n-1]
				}
				source := layout.Runs[sourceIndex].Glyphs
				glyphs = append(glyphs, source[glyphStart:glyphEnd]...)
				layout.Runs[sourceIndex].Glyphs = slices.Delete(source, glyphStart, glyphEnd)
				layout.Runs[destinationIndex] = gpui.ShapedRun{FontID: seg.fontID, Glyphs: glyphs}
				glyphEnd = glyphStart
			}
		}
		destinationEnd = destinationStart
	}
}
