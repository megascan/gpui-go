package gpui

import (
	"fmt"
	"slices"
	"strconv"
	"time"
)

// DebugFrameOverlayMode is what the debug frame overlay shows.
type DebugFrameOverlayMode uint8

const (
	// DebugFrameOverlayModeHidden paints nothing.
	DebugFrameOverlayModeHidden DebugFrameOverlayMode = iota
	// DebugFrameOverlayModeMinimal paints the current frame time.
	DebugFrameOverlayModeMinimal
	// DebugFrameOverlayModeFull paints frame-time statistics.
	DebugFrameOverlayModeFull
)

// String formats the mode as upstream's Debug impl.
func (m DebugFrameOverlayMode) String() string {
	switch m {
	case DebugFrameOverlayModeHidden:
		return "Hidden"
	case DebugFrameOverlayModeMinimal:
		return "Minimal"
	default:
		return "Full"
	}
}

// Next returns the next mode in the Hidden → FrameTime → Detailed cycle.
func (m DebugFrameOverlayMode) Next() DebugFrameOverlayMode {
	switch m {
	case DebugFrameOverlayModeHidden:
		return DebugFrameOverlayModeMinimal
	case DebugFrameOverlayModeMinimal:
		return DebugFrameOverlayModeFull
	default:
		return DebugFrameOverlayModeHidden
	}
}

const debugOverlayMaxSamples = 1000

const (
	debugOverlayGlyphWidth   = 5
	debugOverlayGlyphHeight  = 7
	debugOverlayCharAdvance  = float32(debugOverlayGlyphWidth + 1)
	debugOverlayLineAdvance  = float32(debugOverlayGlyphHeight + 2)
	debugOverlayPanelPadding = float32(2.0)
	debugOverlayPanelMargin  = float32(4.0)
	debugOverlayCellSize     = float32(2.0)
)

func debugOverlayTextColor() Hsla {
	return NewRgba(0x33ff33ff).ToHsla()
}

func debugOverlayPanelColor() Hsla {
	return NewRgba(0x000000aa).ToHsla()
}

// DebugFrameOverlay paints frame-time statistics directly into the scene, bypassing layout,
// text and view invalidation.
type DebugFrameOverlay struct {
	currentMode     DebugFrameOverlayMode
	drawDurations   []time.Duration
	totalFrameCount uint64
}

// NewDebugFrameOverlay creates a hidden overlay with no samples.
func NewDebugFrameOverlay() DebugFrameOverlay {
	return DebugFrameOverlay{currentMode: DebugFrameOverlayModeHidden}
}

func (o *DebugFrameOverlay) mode() DebugFrameOverlayMode {
	return o.currentMode
}

func (o *DebugFrameOverlay) setMode(mode DebugFrameOverlayMode) {
	o.currentMode = mode
}

func (o *DebugFrameOverlay) resetStats() {
	o.drawDurations = o.drawDurations[:0]
}

func (o *DebugFrameOverlay) isEnabled() bool {
	return o.currentMode != DebugFrameOverlayModeHidden
}

func (o *DebugFrameOverlay) recordFrame(drawDuration time.Duration) {
	o.totalFrameCount++
	if len(o.drawDurations) >= debugOverlayMaxSamples {
		o.drawDurations = append(o.drawDurations[:0], o.drawDurations[1:]...)
	}
	o.drawDurations = append(o.drawDurations, drawDuration)
}

func (o *DebugFrameOverlay) paint(scene *Scene, viewportSize Size[Pixels], scaleFactor float32) {
	if !o.isEnabled() {
		return
	}

	lines := o.lines()
	maxLineChars := 0
	for _, line := range lines {
		maxLineChars = max(maxLineChars, len(line))
	}
	cell := max(debugOverlayCellSize*scaleFactor, 1.0)

	panelWidth := cell * (float32(maxLineChars)*debugOverlayCharAdvance + 2.0*debugOverlayPanelPadding)
	panelHeight := cell * (float32(len(lines))*debugOverlayLineAdvance + 2.0*debugOverlayPanelPadding)
	viewport := viewportSize.Scale(scaleFactor)
	panelLeft := float32(viewport.Width) - panelWidth - cell*debugOverlayPanelMargin
	panelTop := cell * debugOverlayPanelMargin

	contentMask := ContentMask[ScaledPixels]{
		Bounds: Bounds[ScaledPixels]{
			Origin: NewPoint(ScaledPixels(0), ScaledPixels(0)),
			Size:   viewport,
		},
	}

	scene.InsertPrimitive(debugOverlaySolidQuad(
		debugOverlayScaledBounds(panelLeft, panelTop, panelWidth, panelHeight),
		contentMask,
		debugOverlayPanelColor(),
	))

	textColor := debugOverlayTextColor()
	for lineIndex, line := range lines {
		lineTop := panelTop + cell*(debugOverlayPanelPadding+float32(lineIndex)*debugOverlayLineAdvance)
		charIndex := -1
		for _, character := range line {
			charIndex++
			rows, ok := debugOverlayGlyph(character)
			if !ok {
				continue
			}
			glyphLeft := panelLeft + cell*(debugOverlayPanelPadding+float32(charIndex)*debugOverlayCharAdvance)
			for rowIndex, row := range rows {
				rowTop := lineTop + cell*float32(rowIndex)
				column := 0
				for column < debugOverlayGlyphWidth {
					if row&(1<<(debugOverlayGlyphWidth-1-column)) == 0 {
						column++
						continue
					}
					runStart := column
					for column < debugOverlayGlyphWidth && row&(1<<(debugOverlayGlyphWidth-1-column)) != 0 {
						column++
					}
					scene.InsertPrimitive(debugOverlaySolidQuad(
						debugOverlayScaledBounds(
							glyphLeft+cell*float32(runStart),
							rowTop,
							cell*float32(column-runStart),
							cell,
						),
						contentMask,
						textColor,
					))
				}
			}
		}
	}
}

func (o *DebugFrameOverlay) lines() []string {
	current, hasCurrent := time.Duration(0), false
	if n := len(o.drawDurations); n > 0 {
		current, hasCurrent = o.drawDurations[n-1], true
	}
	switch o.currentMode {
	case DebugFrameOverlayModeHidden:
		return []string{}
	case DebugFrameOverlayModeMinimal:
		return []string{debugOverlayFormatMs(current, hasCurrent)}
	default:
		sorted := slices.Clone(o.drawDurations)
		slices.Sort(sorted)
		percentile := func(numerator int) (time.Duration, bool) {
			if len(sorted) == 0 {
				return 0, false
			}
			return sorted[(len(sorted)-1)*numerator/100], true
		}
		var frameCount string
		if o.totalFrameCount > 99_999 {
			frameCount = "LOTS"
		} else {
			frameCount = strconv.FormatUint(o.totalFrameCount, 10)
		}
		maxDuration, hasMax := time.Duration(0), false
		if n := len(sorted); n > 0 {
			maxDuration, hasMax = sorted[n-1], true
		}
		return []string{
			"CUR " + debugOverlayFormatMs(current, hasCurrent),
			"1%  " + debugOverlayFormatMs(percentile(99)),
			"10% " + debugOverlayFormatMs(percentile(90)),
			"MAX " + debugOverlayFormatMs(maxDuration, hasMax),
			fmt.Sprintf("FRAMES %5s", frameCount),
		}
	}
}

func debugOverlayFormatMs(duration time.Duration, ok bool) string {
	if !ok {
		return "   -- MS"
	}
	secs := float32(uint64(duration/time.Second)) + float32(uint32(duration%time.Second))/1e9
	ms := secs * 1000.0
	return fmt.Sprintf("%5.1f MS", float64(ms))
}

func debugOverlayScaledBounds(left, top, width, height float32) Bounds[ScaledPixels] {
	return Bounds[ScaledPixels]{
		Origin: NewPoint(ScaledPixels(left), ScaledPixels(top)),
		Size:   NewSize(ScaledPixels(width), ScaledPixels(height)),
	}
}

func debugOverlaySolidQuad(bounds Bounds[ScaledPixels], contentMask ContentMask[ScaledPixels], color Hsla) Quad {
	return Quad{
		Order:        0,
		BorderStyle:  BorderStyleSolid,
		Bounds:       bounds,
		ContentMask:  contentMask,
		Background:   SolidBackground(color),
		BorderColor:  TransparentBlack(),
		CornerRadii:  Corners[ScaledPixels]{},
		BorderWidths: Edges[ScaledPixels]{},
	}
}

func debugOverlayGlyph(character rune) ([debugOverlayGlyphHeight]uint8, bool) {
	switch character {
	case '0':
		return [7]uint8{0b01110, 0b10001, 0b10011, 0b10101, 0b11001, 0b10001, 0b01110}, true
	case '1':
		return [7]uint8{0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110}, true
	case '2':
		return [7]uint8{0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b01000, 0b11111}, true
	case '3':
		return [7]uint8{0b11111, 0b00010, 0b00100, 0b00010, 0b00001, 0b10001, 0b01110}, true
	case '4':
		return [7]uint8{0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010}, true
	case '5':
		return [7]uint8{0b11111, 0b10000, 0b11110, 0b00001, 0b00001, 0b10001, 0b01110}, true
	case '6':
		return [7]uint8{0b00110, 0b01000, 0b10000, 0b11110, 0b10001, 0b10001, 0b01110}, true
	case '7':
		return [7]uint8{0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000}, true
	case '8':
		return [7]uint8{0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110}, true
	case '9':
		return [7]uint8{0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00010, 0b01100}, true
	case '.':
		return [7]uint8{0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b01100, 0b01100}, true
	case '-':
		return [7]uint8{0b00000, 0b00000, 0b00000, 0b11111, 0b00000, 0b00000, 0b00000}, true
	case '%':
		return [7]uint8{0b11001, 0b11001, 0b00010, 0b00100, 0b01000, 0b10011, 0b10011}, true
	case 'A':
		return [7]uint8{0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001}, true
	case 'C':
		return [7]uint8{0b01110, 0b10001, 0b10000, 0b10000, 0b10000, 0b10001, 0b01110}, true
	case 'E':
		return [7]uint8{0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111}, true
	case 'F':
		return [7]uint8{0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000}, true
	case 'L':
		return [7]uint8{0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111}, true
	case 'M':
		return [7]uint8{0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001}, true
	case 'N':
		return [7]uint8{0b10001, 0b11001, 0b10101, 0b10011, 0b10001, 0b10001, 0b10001}, true
	case 'O':
		return [7]uint8{0b01110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110}, true
	case 'R':
		return [7]uint8{0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001}, true
	case 'S':
		return [7]uint8{0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110}, true
	case 'T':
		return [7]uint8{0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100}, true
	case 'U':
		return [7]uint8{0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110}, true
	case 'X':
		return [7]uint8{0b10001, 0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b10001}, true
	default:
		return [7]uint8{}, false
	}
}
