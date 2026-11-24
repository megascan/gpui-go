package gpui

import (
	"slices"
	"testing"
	"time"
)

func TestDebugOverlay_AllRenderedCharactersHaveGlyphs(t *testing.T) {
	overlay := NewDebugFrameOverlay()
	overlay.setMode(DebugFrameOverlayModeFull)

	var lines []string
	for _, duration := range []time.Duration{
		0,
		time.Microsecond,
		8_333 * time.Microsecond,
		123 * time.Millisecond,
		2 * time.Second,
	} {
		overlay.recordFrame(duration)
		lines = append(lines, overlay.lines()...)
	}
	overlay.totalFrameCount = 100_000
	lines = append(lines, overlay.lines()...)

	empty := NewDebugFrameOverlay()
	empty.setMode(DebugFrameOverlayModeFull)
	lines = append(lines, empty.lines()...)

	for _, line := range lines {
		for _, character := range line {
			if _, ok := debugOverlayGlyph(character); character != ' ' && !ok {
				t.Fatalf("no glyph for %q in line %q", character, line)
			}
		}
	}
}

func TestDebugOverlay_PercentileLowsAreReportedAsTimes(t *testing.T) {
	overlay := NewDebugFrameOverlay()
	overlay.setMode(DebugFrameOverlayModeFull)
	for milliseconds := 1; milliseconds <= 100; milliseconds++ {
		overlay.recordFrame(time.Duration(milliseconds) * time.Millisecond)
	}
	lines := overlay.lines()
	debugOverlayAssertEq(t, lines[0], "CUR 100.0 MS")
	debugOverlayAssertEq(t, lines[1], "1%   99.0 MS")
	debugOverlayAssertEq(t, lines[2], "10%  90.0 MS")
	debugOverlayAssertEq(t, lines[3], "MAX 100.0 MS")
	debugOverlayAssertEq(t, lines[4], "FRAMES   100")
}

func TestDebugOverlay_ResetClearsDurationsButKeepsFrameCount(t *testing.T) {
	overlay := NewDebugFrameOverlay()
	overlay.setMode(DebugFrameOverlayModeFull)
	for range 10 {
		overlay.recordFrame(10 * time.Millisecond)
	}
	overlay.resetStats()
	lines := overlay.lines()
	debugOverlayAssertEq(t, lines[0], "CUR    -- MS")
	debugOverlayAssertEq(t, lines[3], "MAX    -- MS")
	debugOverlayAssertEq(t, lines[4], "FRAMES    10")
	overlay.recordFrame(20 * time.Millisecond)
	lines = overlay.lines()
	debugOverlayAssertEq(t, lines[0], "CUR  20.0 MS")
	debugOverlayAssertEq(t, lines[4], "FRAMES    11")
}

func TestDebugOverlay_FrameCountAccumulatesAcrossModeChanges(t *testing.T) {
	overlay := NewDebugFrameOverlay()
	for range 3 {
		overlay.recordFrame(10 * time.Millisecond)
	}
	overlay.setMode(DebugFrameOverlayModeMinimal)
	if got := overlay.lines(); !slices.Equal(got, []string{" 10.0 MS"}) {
		t.Fatalf("lines = %q", got)
	}
	overlay.setMode(DebugFrameOverlayModeFull)
	overlay.recordFrame(10 * time.Millisecond)
	debugOverlayAssertEq(t, overlay.lines()[4], "FRAMES     4")
	overlay.setMode(DebugFrameOverlayModeHidden)
	overlay.recordFrame(10 * time.Millisecond)
	overlay.setMode(DebugFrameOverlayModeFull)
	debugOverlayAssertEq(t, overlay.lines()[4], "FRAMES     5")
}

func TestDebugOverlay_FrameCountIsRightAlignedAndSaturates(t *testing.T) {
	overlay := NewDebugFrameOverlay()
	overlay.setMode(DebugFrameOverlayModeFull)
	overlay.recordFrame(10 * time.Millisecond)
	debugOverlayAssertEq(t, overlay.lines()[4], "FRAMES     1")
	overlay.totalFrameCount = 99_999
	debugOverlayAssertEq(t, overlay.lines()[4], "FRAMES 99999")
	overlay.totalFrameCount = 100_000
	debugOverlayAssertEq(t, overlay.lines()[4], "FRAMES  LOTS")
}

func TestDebugOverlay_TogglingOnShowsPreviousFrameImmediately(t *testing.T) {
	overlay := NewDebugFrameOverlay()
	overlay.recordFrame(10 * time.Millisecond)
	overlay.setMode(DebugFrameOverlayModeMinimal)
	if got := overlay.lines(); !slices.Equal(got, []string{" 10.0 MS"}) {
		t.Fatalf("lines = %q", got)
	}
}

func debugOverlayAssertEq(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("assertion `left == right` failed\n  left: %q\n right: %q", got, want)
	}
}
