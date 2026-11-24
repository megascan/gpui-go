package gpui

import (
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

// HangDetector detects hangs from the foreground journal (M3-profiler-hang ports it).
type HangDetector struct{}

// TakeActiveTime returns the time the user was active since the previous call (M3-profiler-hang ports it).
func (d *HangDetector) TakeActiveTime(now scheduler.Instant, maxGap time.Duration) time.Duration {
	panic("gpui: unimplemented HangDetector.TakeActiveTime")
}

// FirstPresentAt returns when the first frame was presented (M3-profiler-hang ports it).
func (d *HangDetector) FirstPresentAt() (scheduler.Instant, bool) {
	panic("gpui: unimplemented HangDetector.FirstPresentAt")
}

// Poll returns the incidents sealed since the previous poll (M3-profiler-hang ports it).
func (d *HangDetector) Poll() []HangIncident {
	panic("gpui: unimplemented HangDetector.Poll")
}

// HangIncident is a detected hang (M3-profiler-hang ports it).
type HangIncident struct{}

// NewHangDetector creates a detector over the foreground journal (M3-profiler-hang ports it).
func NewHangDetector(journal ForegroundJournal, threshold time.Duration, frameBudget time.Duration) *HangDetector {
	panic("gpui: unimplemented NewHangDetector")
}

// NewVisualTestPlatform wraps platform with a TestDispatcher seeded with seed (M3-root-capture ports it).
func NewVisualTestPlatform(platform Platform, seed uint64) *VisualTestPlatform {
	panic("gpui: unimplemented NewVisualTestPlatform")
}

// Dispatcher returns the platform's test dispatcher (M3-root-capture ports it).
func (p *VisualTestPlatform) Dispatcher() *TestDispatcher {
	panic("gpui: unimplemented VisualTestPlatform.Dispatcher")
}

// DebugFrameOverlay draws frame timings over the window (M3-profiler-hang ports it).
type DebugFrameOverlay struct{}

// NewDebugFrameOverlay creates an overlay (M3-profiler-hang ports it). The placeholder body returns the zero value.
func NewDebugFrameOverlay() DebugFrameOverlay {
	return DebugFrameOverlay{}
}

// DebugFrameOverlayMode is what the debug frame overlay shows (M3-profiler-hang ports it).
type DebugFrameOverlayMode uint8

// Next returns the next mode in the cycle (M3-profiler-hang ports it).
func (m DebugFrameOverlayMode) Next() DebugFrameOverlayMode {
	panic("gpui: unimplemented DebugFrameOverlayMode.Next")
}

func (o *DebugFrameOverlay) mode() DebugFrameOverlayMode {
	return 0
}

func (o *DebugFrameOverlay) setMode(mode DebugFrameOverlayMode) {}

func (o *DebugFrameOverlay) resetStats() {}

func (o *DebugFrameOverlay) recordFrame(drawDuration time.Duration) {}

func (o *DebugFrameOverlay) paint(scene *Scene, viewportSize Size[Pixels], scaleFactor float32) {}

// VisualTestPlatform combines a real platform's rendering with a TestDispatcher (M3-root-capture ports it).
type VisualTestPlatform struct {
	Platform
}
