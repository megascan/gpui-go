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

// InputTiming is one input dispatch (M3-profiler-journal ports it).
type InputTiming struct {
	Kind               string
	Start              scheduler.Instant
	End                scheduler.Instant
	CausedInvalidation bool
}

func beginForegroundTurn() {}

func endForegroundTurn() {}

func recordTaskPoll(timing TaskTiming) {}

func recordFramePending(windowID WindowID, dirtyAt scheduler.Instant) {}

func recordWindowClosed(windowID WindowID) {}

func powerInterruptedSince(start scheduler.Instant) bool {
	return false
}

func frameSampleIsValid(id WindowID, start scheduler.Instant) bool {
	return true
}

func recordInput(timing InputTiming) {}

func recordAction(timing ActionTiming) {}

func recordView(timing ViewTiming) {}

func recordDraw(timing FrameTiming) {}

func recordPresent(timing PresentTiming, frame Option[FrameTiming]) {}

// NewHangDetector creates a detector over the foreground journal (M3-profiler-hang ports it).
func NewHangDetector(journal ForegroundJournal, threshold time.Duration, frameBudget time.Duration) *HangDetector {
	panic("gpui: unimplemented NewHangDetector")
}

func observePower(cx *App) {}

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

// PlatformSignal is a platform frame callback recorded in the foreground journal (M3-profiler-journal ports it).
type PlatformSignal struct {
	// WindowID is the window receiving the callback.
	WindowID WindowID
	// SignalAt is when the platform requested the callback, if its clock is supported.
	SignalAt Option[scheduler.Instant]
	// HandledAt is when foreground handling began.
	HandledAt scheduler.Instant
	// Source is whether the callback came from the platform or local scheduling.
	Source FrameRequestSource
}

// FrameSkipReason is why a frame request returned without drawing a new frame (M3-profiler-journal ports it).
type FrameSkipReason uint8

const (
	// FrameSkipReasonNoRenderNeeded means the callback completed without needing a new render.
	FrameSkipReasonNoRenderNeeded FrameSkipReason = iota
	// FrameSkipReasonInactiveFrameRateLimit means an inactive window deferred drawing.
	FrameSkipReasonInactiveFrameRateLimit
	// FrameSkipReasonThermalFrameRateLimit means thermal throttling deferred drawing.
	FrameSkipReasonThermalFrameRateLimit
	// FrameSkipReasonLowPowerFrameRateLimit means low power mode deferred drawing.
	FrameSkipReasonLowPowerFrameRateLimit
	// FrameSkipReasonAnimationFrameRateLimit means a scene animation's frame rate cap deferred presenting.
	FrameSkipReasonAnimationFrameRateLimit
)

func recordWindowVisibility(id WindowID, visibility WindowVisibility) {}

func recordPlatformSignal(signal PlatformSignal) {}

func recordReentrantFrameSkipped(windowID WindowID, at scheduler.Instant) {}

func recordFrameSkipped(windowID WindowID, at scheduler.Instant, reason FrameSkipReason) {}

// ForegroundJournal is the foreground work journal of the app's thread (M3-profiler-journal ports it).
type ForegroundJournal struct{}

func installForegroundJournal() ForegroundJournal {
	return ForegroundJournal{}
}

// VisualTestPlatform combines a real platform's rendering with a TestDispatcher (M3-root-capture ports it).
type VisualTestPlatform struct {
	Platform
}
