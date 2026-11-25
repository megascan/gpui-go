package gpui

import (
	"fmt"
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

type powerState uint8

const (
	powerStateAwake powerState = iota
	powerStateSuspended
)

// TaskPollFloor is the poll duration below which task polls are folded into a PollSummary
// instead of being recorded individually.
const TaskPollFloor = 100 * time.Microsecond

// FrameDeadline is how long a dirty, unpresented frame blocks foreground-idle boundaries.
const FrameDeadline = time.Second

const minDetectedSuspend = 50 * time.Millisecond

const maxIntervalEvents = 16 * 1024

const maxJournalEntries = (4 * 1024 * 1024) / (unsafe.Sizeof(journalSlot{}) + unsafe.Sizeof(PresentedFrame{}))

const maxPendingJournalEntries = 64

// ForegroundEvent is one completed piece of foreground work: ForegroundEventTaskPoll,
// ForegroundEventAction, ForegroundEventInput, ForegroundEventDraw, ForegroundEventView,
// ForegroundEventPresent or ForegroundEventSmallPolls.
type ForegroundEvent interface {
	// StartTime returns when the work began. For draws this is the start of the draw itself.
	StartTime() scheduler.Instant
	// EndTime returns when the work ended.
	EndTime() scheduler.Instant
	// Duration returns how long the work took; for small polls, the span the summary covers.
	Duration() time.Duration
	isForegroundEvent()
}

// ForegroundEventTaskPoll is a foreground task poll at least TaskPollFloor long.
type ForegroundEventTaskPoll struct{ TaskTiming }

// ForegroundEventAction is a completed action handler.
type ForegroundEventAction struct{ ActionTiming }

// ForegroundEventInput is a dispatched platform input event.
type ForegroundEventInput struct{ InputTiming }

// ForegroundEventDraw is a completed window draw.
type ForegroundEventDraw struct{ FrameTiming }

// ForegroundEventView is one phase of drawing a view.
type ForegroundEventView struct{ ViewTiming }

// ForegroundEventPresent is work spent submitting a frame to the platform.
type ForegroundEventPresent struct{ PresentTiming }

// ForegroundEventSmallPolls aggregates task polls below TaskPollFloor.
type ForegroundEventSmallPolls struct{ SmallPollFlush }

func (ForegroundEventTaskPoll) isForegroundEvent()   {}
func (ForegroundEventAction) isForegroundEvent()     {}
func (ForegroundEventInput) isForegroundEvent()      {}
func (ForegroundEventDraw) isForegroundEvent()       {}
func (ForegroundEventView) isForegroundEvent()       {}
func (ForegroundEventPresent) isForegroundEvent()    {}
func (ForegroundEventSmallPolls) isForegroundEvent() {}

// StartTime returns when the poll began.
func (e ForegroundEventTaskPoll) StartTime() scheduler.Instant { return e.Start }

// StartTime returns when the action handler began.
func (e ForegroundEventAction) StartTime() scheduler.Instant { return e.Start }

// StartTime returns when the input dispatch began.
func (e ForegroundEventInput) StartTime() scheduler.Instant { return e.Start }

// StartTime returns when the draw began.
func (e ForegroundEventDraw) StartTime() scheduler.Instant { return e.DrawStart }

// StartTime returns when the view phase began.
func (e ForegroundEventView) StartTime() scheduler.Instant { return e.Start }

// StartTime returns when the presentation began.
func (e ForegroundEventPresent) StartTime() scheduler.Instant { return e.PresentStart }

// StartTime returns when the first folded poll began.
func (e ForegroundEventSmallPolls) StartTime() scheduler.Instant { return e.Since }

// EndTime returns when the poll ended.
func (e ForegroundEventTaskPoll) EndTime() scheduler.Instant { return e.End.Instant }

// EndTime returns when the action handler ended.
func (e ForegroundEventAction) EndTime() scheduler.Instant { return e.End }

// EndTime returns when the input dispatch ended.
func (e ForegroundEventInput) EndTime() scheduler.Instant { return e.End }

// EndTime returns when the draw ended.
func (e ForegroundEventDraw) EndTime() scheduler.Instant { return e.DrawEnd }

// EndTime returns when the view phase ended.
func (e ForegroundEventView) EndTime() scheduler.Instant { return e.End }

// EndTime returns when the presentation ended.
func (e ForegroundEventPresent) EndTime() scheduler.Instant { return e.PresentEnd }

// EndTime returns when the last folded poll ended.
func (e ForegroundEventSmallPolls) EndTime() scheduler.Instant { return e.Until }

func foregroundEventDuration(e ForegroundEvent) time.Duration {
	return e.EndTime().Sub(e.StartTime())
}

// Duration returns how long the poll took.
func (e ForegroundEventTaskPoll) Duration() time.Duration { return foregroundEventDuration(e) }

// Duration returns how long the action handler took.
func (e ForegroundEventAction) Duration() time.Duration { return foregroundEventDuration(e) }

// Duration returns how long the input dispatch took.
func (e ForegroundEventInput) Duration() time.Duration { return foregroundEventDuration(e) }

// Duration returns how long the draw took.
func (e ForegroundEventDraw) Duration() time.Duration { return foregroundEventDuration(e) }

// Duration returns how long the view phase took.
func (e ForegroundEventView) Duration() time.Duration { return foregroundEventDuration(e) }

// Duration returns how long the presentation took.
func (e ForegroundEventPresent) Duration() time.Duration { return foregroundEventDuration(e) }

// Duration returns the span the summary covers, not the time spent polling.
func (e ForegroundEventSmallPolls) Duration() time.Duration { return foregroundEventDuration(e) }

// InputTiming is the timing of one platform input dispatch on a window.
type InputTiming struct {
	// Kind is the platform input variant dispatched.
	Kind string
	// Start is when the input dispatch started.
	Start scheduler.Instant
	// End is when the input dispatch finished.
	End scheduler.Instant
	// CausedInvalidation is whether handling the input invalidated a window.
	CausedInvalidation bool
}

// PlatformSignal is a frame callback entering the foreground, without implying foreground work.
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

// FrameSkipReason is why a completed frame request did not draw a new frame.
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

// FrameSkipped is a completed frame request that returned without drawing a new frame.
type FrameSkipped struct {
	// WindowID is the window whose callback returned.
	WindowID WindowID
	// At is when the foreground completed this callback.
	At scheduler.Instant
	// Reason is why no new frame was drawn.
	Reason FrameSkipReason
}

// PollSummary is the exact count and total duration of task polls below TaskPollFloor.
type PollSummary struct {
	// Count is the number of polls below the floor.
	Count uint64
	// Total is the total duration of polls below the floor.
	Total time.Duration
}

func (s *PollSummary) add(other PollSummary) {
	s.Count += other.Count
	s.Total += other.Total
}

// SmallPollFlush is a flushed PollSummary and the tightest span containing its folded polls.
type SmallPollFlush struct {
	// Summary is the folded polls.
	Summary PollSummary
	// Since is when the first folded poll began.
	Since scheduler.Instant
	// Until is when the last folded poll ended.
	Until scheduler.Instant
}

// PresentedFrame is a newly drawn frame and the platform submission that completed its interval.
type PresentedFrame struct {
	// Frame is the draw whose rendered scene was submitted.
	Frame FrameTiming
	// Presentation is the work spent submitting that scene to the platform.
	Presentation PresentTiming
}

// DirtyToPresentDuration returns the time from the frame's first invalidation through platform submission.
func (p PresentedFrame) DirtyToPresentDuration() (time.Duration, bool) {
	dirtyAt, ok := p.Frame.DirtyAt.Get()
	if !ok {
		return 0, false
	}
	return p.Presentation.PresentEnd.Sub(dirtyAt), true
}

// IntervalBoundary is the semantic event that completed a foreground activity interval:
// IntervalBoundaryPresented, IntervalBoundaryFrameSkipped, IntervalBoundaryIdle or
// IntervalBoundaryPowerTransition.
type IntervalBoundary interface {
	// EndTime returns when the interval ended.
	EndTime() scheduler.Instant
	// DirtyAt returns the first invalidation of the frame satisfied by this boundary, if any.
	DirtyAt() (scheduler.Instant, bool)
	isIntervalBoundary()
}

// IntervalBoundaryPresented means a newly drawn frame was submitted to the platform.
type IntervalBoundaryPresented struct{ PresentedFrame }

// IntervalBoundaryFrameSkipped means a callback completed without drawing, excluding reentrant callbacks.
type IntervalBoundaryFrameSkipped struct{ FrameSkipped }

// IntervalBoundaryIdle means the foreground returned to an idle platform loop with no unexpired frame pending.
type IntervalBoundaryIdle struct {
	// EndedAt is when the foreground went idle.
	EndedAt scheduler.Instant
}

// IntervalBoundaryPowerTransition means a system suspend interrupted the measurement interval.
type IntervalBoundaryPowerTransition struct {
	// EndedAt is when the foreground noticed the suspend.
	EndedAt scheduler.Instant
}

func (IntervalBoundaryPresented) isIntervalBoundary()       {}
func (IntervalBoundaryFrameSkipped) isIntervalBoundary()    {}
func (IntervalBoundaryIdle) isIntervalBoundary()            {}
func (IntervalBoundaryPowerTransition) isIntervalBoundary() {}

// EndTime returns when the presentation ended.
func (b IntervalBoundaryPresented) EndTime() scheduler.Instant { return b.Presentation.PresentEnd }

// EndTime returns when the skipped callback completed.
func (b IntervalBoundaryFrameSkipped) EndTime() scheduler.Instant { return b.At }

// EndTime returns when the foreground went idle.
func (b IntervalBoundaryIdle) EndTime() scheduler.Instant { return b.EndedAt }

// EndTime returns when the suspend was noticed.
func (b IntervalBoundaryPowerTransition) EndTime() scheduler.Instant { return b.EndedAt }

// DirtyAt returns the presented frame's first invalidation.
func (b IntervalBoundaryPresented) DirtyAt() (scheduler.Instant, bool) { return b.Frame.DirtyAt.Get() }

// DirtyAt returns false: a skip satisfies no frame.
func (IntervalBoundaryFrameSkipped) DirtyAt() (scheduler.Instant, bool) {
	return scheduler.Instant{}, false
}

// DirtyAt returns false: idle satisfies no frame.
func (IntervalBoundaryIdle) DirtyAt() (scheduler.Instant, bool) { return scheduler.Instant{}, false }

// DirtyAt returns false: a power transition satisfies no frame.
func (IntervalBoundaryPowerTransition) DirtyAt() (scheduler.Instant, bool) {
	return scheduler.Instant{}, false
}

// FrameStateChange is a control-plane change to one window's pending-frame state:
// FrameStateChangePending or FrameStateChangeClosed.
type FrameStateChange interface {
	isFrameStateChange()
}

// FrameStateChangePending means the window has a frame waiting to be presented.
type FrameStateChangePending struct {
	// WindowID is the window waiting for presentation.
	WindowID WindowID
	// DirtyAt is when this frame generation first became dirty.
	DirtyAt scheduler.Instant
}

// FrameStateChangeClosed means the window closed, so its pending frame no longer blocks idle boundaries.
type FrameStateChangeClosed struct {
	// WindowID is the window that closed.
	WindowID WindowID
	// At is when the window closed.
	At scheduler.Instant
}

func (FrameStateChangePending) isFrameStateChange() {}
func (FrameStateChangeClosed) isFrameStateChange()  {}

// ForegroundJournalEntry is one item retained in the foreground journal:
// ForegroundJournalEntryEvent, ForegroundJournalEntryBoundary, ForegroundJournalEntryFrameState,
// ForegroundJournalEntryPlatformSignal, ForegroundJournalEntryReentrantFrameSkipped or
// ForegroundJournalEntryDiscontinuity.
type ForegroundJournalEntry interface {
	isForegroundJournalEntry()
}

// ForegroundJournalEntryEvent is a completed piece of foreground work.
type ForegroundJournalEntryEvent struct{ Event ForegroundEvent }

// ForegroundJournalEntryBoundary is a semantic interval boundary.
type ForegroundJournalEntryBoundary struct{ Boundary IntervalBoundary }

// ForegroundJournalEntryFrameState is a change to pending-frame state; metadata, not work.
type ForegroundJournalEntryFrameState struct{ State FrameStateChange }

// ForegroundJournalEntryPlatformSignal is a callback entering the foreground; not work or a boundary.
type ForegroundJournalEntryPlatformSignal struct{ Signal PlatformSignal }

// ForegroundJournalEntryReentrantFrameSkipped is a nested frame request deferred while an
// enclosing draw was still running.
type ForegroundJournalEntryReentrantFrameSkipped struct {
	// WindowID is the window whose nested frame request was deferred.
	WindowID WindowID
	// At is when the nested frame request was deferred.
	At scheduler.Instant
}

// ForegroundJournalEntryDiscontinuity means logical entries were unavailable at this point in
// the stream; consumers must not infer interval boundaries across it.
type ForegroundJournalEntryDiscontinuity struct {
	// Lost is the number of unavailable logical entries.
	Lost uint64
}

func (ForegroundJournalEntryEvent) isForegroundJournalEntry()                 {}
func (ForegroundJournalEntryBoundary) isForegroundJournalEntry()              {}
func (ForegroundJournalEntryFrameState) isForegroundJournalEntry()            {}
func (ForegroundJournalEntryPlatformSignal) isForegroundJournalEntry()        {}
func (ForegroundJournalEntryReentrantFrameSkipped) isForegroundJournalEntry() {}
func (ForegroundJournalEntryDiscontinuity) isForegroundJournalEntry()         {}

// FrameSnapshot is an immutable view of one sealed foreground interval, produced by IntervalSealer.
type FrameSnapshot struct {
	// IntervalStart is when the interval started.
	IntervalStart scheduler.Instant
	// Boundary is the semantic event that completed the interval.
	Boundary IntervalBoundary
	// Events is the foreground work recorded during the interval, in completion order, without small polls.
	Events []ForegroundEvent
	// SmallPolls is the span-tagged aggregates of task polls below TaskPollFloor.
	SmallPolls []SmallPollFlush
	// DroppedEvents is events lost to the interval's event cap plus reported ring losses.
	DroppedEvents uint64
	// JournalDiscontinuous is whether the journal had an unobserved gap during this interval.
	JournalDiscontinuous bool
}

// IntervalEnd returns when the interval ended.
func (s *FrameSnapshot) IntervalEnd() scheduler.Instant {
	return s.Boundary.EndTime()
}

// SmallPollSummary returns the combined count and total of all folded sub-floor polls in the interval.
func (s *FrameSnapshot) SmallPollSummary() PollSummary {
	var summary PollSummary
	for _, flush := range s.SmallPolls {
		summary.add(flush.Summary)
	}
	return summary
}

// Occupancy returns the total foreground time occupied within the interval.
func (s *FrameSnapshot) Occupancy() time.Duration {
	return s.OccupancyWithin(s.IntervalStart, s.IntervalEnd())
}

// OccupancyWithin is Occupancy measured against an arbitrary window, apportioning folded polls
// by how much of their span overlaps it.
func (s *FrameSnapshot) OccupancyWithin(windowStart, windowEnd scheduler.Instant) time.Duration {
	type span struct{ start, end scheduler.Instant }
	spans := make([]span, 0, len(s.Events))
	for _, event := range s.Events {
		start := instantMax(event.StartTime(), windowStart)
		end := instantMax(event.EndTime().Min(windowEnd), start)
		spans = append(spans, span{start, end})
	}
	slices.SortStableFunc(spans, func(a, b span) int { return a.start.Compare(b.start) })

	var occupied time.Duration
	var mergedUntil Option[scheduler.Instant]
	for _, sp := range spans {
		start := sp.start
		if until, ok := mergedUntil.Get(); ok {
			start = instantMax(start, until)
		}
		occupied += sp.end.Sub(start)
		if until, ok := mergedUntil.Get(); ok {
			mergedUntil = Some(instantMax(until, sp.end))
		} else {
			mergedUntil = Some(sp.end)
		}
	}

	var apportioned time.Duration
	for _, flush := range s.SmallPolls {
		spanDuration := flush.Until.Sub(flush.Since)
		if spanDuration == 0 {
			if !flush.Since.Before(windowStart) && !flush.Since.After(windowEnd) {
				apportioned += flush.Summary.Total
			}
		} else {
			overlapStart := instantMax(flush.Since, windowStart)
			overlapEnd := instantMax(flush.Until.Min(windowEnd), overlapStart)
			overlap := overlapEnd.Sub(overlapStart)
			apportioned += durationMulF64(flush.Summary.Total, durationDivDurationF64(overlap, spanDuration))
		}
	}
	return occupied + apportioned
}

// BusyFraction returns the fraction of the interval the foreground spent working, in 0..=1.
func (s *FrameSnapshot) BusyFraction() float64 {
	interval := s.IntervalEnd().Sub(s.IntervalStart)
	if interval == 0 {
		return 1.0
	}
	return math.Min(durationDivDurationF64(s.Occupancy(), interval), 1.0)
}

func instantMax(a, b scheduler.Instant) scheduler.Instant {
	if b.After(a) {
		return b
	}
	return a
}

func durationDivDurationF64(a, b time.Duration) float64 {
	return float64(a) / float64(b)
}

func durationSecsF64(d time.Duration) float64 {
	return float64(d/time.Second) + float64(d%time.Second)/1e9
}

func durationMulF64(d time.Duration, rhs float64) time.Duration {
	secs := rhs * durationSecsF64(d)
	whole := math.Floor(secs)
	return time.Duration(whole)*time.Second + time.Duration(math.Round((secs-whole)*1e9))
}

type foregroundRunnableCounter struct {
	count *atomic.Uint64
}

func newForegroundRunnableCounter() foregroundRunnableCounter {
	return foregroundRunnableCounter{count: new(atomic.Uint64)}
}

func (c foregroundRunnableCounter) queued() {
	c.count.Add(1)
}

func (c foregroundRunnableCounter) finished() {
	for {
		count := c.count.Load()
		if count == 0 {
			return
		}
		if c.count.CompareAndSwap(count, count-1) {
			return
		}
	}
}

func (c foregroundRunnableCounter) hasRunnables() bool {
	return c.count.Load() > 0
}

type windowFrameState struct {
	visibility          WindowVisibility
	dirtyAt             Option[scheduler.Instant]
	visibilityChangedAt Option[scheduler.Instant]
}

func newWindowFrameState(visibility WindowVisibility) *windowFrameState {
	return &windowFrameState{visibility: visibility}
}

type foregroundJournalWriter struct {
	foregroundRunnables   foregroundRunnableCounter
	publisher             *journalPublisher
	turnDepth             int
	windows               map[WindowID]*windowFrameState
	power                 powerState
	powerChangedAt        Option[scheduler.Instant]
	readSuspendedTime     func() (time.Duration, bool)
	suspendedTime         Option[time.Duration]
	retainedSinceBoundary bool
	smallPolls            Option[SmallPollFlush]
}

func newForegroundJournalWriter(foregroundRunnables foregroundRunnableCounter, publisher *journalPublisher) *foregroundJournalWriter {
	return &foregroundJournalWriter{
		foregroundRunnables: foregroundRunnables,
		publisher:           publisher,
		windows:             map[WindowID]*windowFrameState{},
		power:               powerStateAwake,
		readSuspendedTime:   systemSuspendedTime,
		suspendedTime:       OptionFrom(systemSuspendedTime()),
	}
}

func (w *foregroundJournalWriter) beginTurn() {
	w.turnDepth++
}

func (w *foregroundJournalWriter) endTurn(endedAt scheduler.Instant) {
	if w.turnDepth == 0 {
		return
	}
	w.turnDepth--
	if w.turnDepth > 0 || w.foregroundRunnables.hasRunnables() || w.hasUnexpiredPendingFrame(endedAt) {
		return
	}
	if !w.retainedSinceBoundary {
		w.smallPolls = None[SmallPollFlush]()
		return
	}
	w.recordEntry(ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: endedAt}})
}

func (w *foregroundJournalWriter) hasUnexpiredPendingFrame(now scheduler.Instant) bool {
	for _, window := range w.windows {
		if at, ok := window.dirtyAt.Get(); ok && now.SaturatingDurationSince(at) >= FrameDeadline {
			window.dirtyAt = None[scheduler.Instant]()
		}
	}
	if w.power != powerStateAwake {
		return false
	}
	for _, window := range w.windows {
		if window.dirtyAt.IsSome() {
			return true
		}
	}
	return false
}

func (w *foregroundJournalWriter) powerTransition(power powerState, at scheduler.Instant) {
	if w.power == power {
		return
	}
	w.power = power
	w.suspendedTime = OptionFrom(w.readSuspendedTime())
	w.interruptMeasurements(at)
}

func (w *foregroundJournalWriter) detectSuspend(at scheduler.Instant) {
	suspendedTime, ok := w.readSuspendedTime()
	if !ok {
		return
	}
	previous := w.suspendedTime.Replace(suspendedTime)
	if prev, ok := previous.Get(); ok && saturatingSubDuration(suspendedTime, prev) >= minDetectedSuspend {
		w.interruptMeasurements(at)
	}
}

func (w *foregroundJournalWriter) interruptMeasurements(at scheduler.Instant) {
	w.recordEntry(ForegroundJournalEntryBoundary{IntervalBoundaryPowerTransition{EndedAt: at}})
	w.powerChangedAt = Some(at)
	for _, window := range w.windows {
		window.dirtyAt = None[scheduler.Instant]()
	}
}

func (w *foregroundJournalWriter) powerInterruptedSince(start scheduler.Instant) bool {
	if w.power == powerStateSuspended {
		return true
	}
	at, ok := w.powerChangedAt.Get()
	return ok && !start.After(at)
}

func (w *foregroundJournalWriter) setVisibility(id WindowID, visibility WindowVisibility, at scheduler.Instant) {
	window, ok := w.windows[id]
	if !ok {
		window = newWindowFrameState(visibility)
		w.windows[id] = window
	}
	if window.visibility != visibility {
		window.visibility = visibility
		window.visibilityChangedAt = Some(at)
		window.dirtyAt = None[scheduler.Instant]()
	}
}

func (w *foregroundJournalWriter) foldSmallPoll(timing TaskTiming) {
	flush := w.smallPolls.AsMut()
	if flush == nil {
		w.smallPolls = Some(SmallPollFlush{Since: timing.Start, Until: timing.End.Instant})
		flush = w.smallPolls.AsMut()
	}
	flush.Summary.Count++
	flush.Summary.Total += timing.PollDuration()
	flush.Since = flush.Since.Min(timing.Start)
	flush.Until = instantMax(flush.Until, timing.End.Instant)
}

func (w *foregroundJournalWriter) takeSmallPolls() Option[SmallPollFlush] {
	return w.smallPolls.Take()
}

func (w *foregroundJournalWriter) recordEvent(event ForegroundEvent) {
	w.retainedSinceBoundary = true
	w.recordEntry(ForegroundJournalEntryEvent{event})
}

func (w *foregroundJournalWriter) recordEntry(entry ForegroundJournalEntry) {
	if flush, ok := w.takeSmallPolls().Get(); ok {
		w.publisher.publish(ForegroundJournalEntryEvent{ForegroundEventSmallPolls{flush}}, entry)
	} else {
		w.publisher.publish(entry)
	}
	if _, ok := entry.(ForegroundJournalEntryBoundary); ok {
		w.retainedSinceBoundary = false
	}
}

func (w *foregroundJournalWriter) recordMarker(entry ForegroundJournalEntry) {
	w.publisher.publish(entry)
}

func (w *foregroundJournalWriter) recordFrameState(change FrameStateChange) {
	w.recordEntry(ForegroundJournalEntryFrameState{change})
}

func (w *foregroundJournalWriter) recordPlatformSignal(signal PlatformSignal) {
	w.recordMarker(ForegroundJournalEntryPlatformSignal{signal})
}

func (w *foregroundJournalWriter) recordFrameSkipped(skipped FrameSkipped) {
	if !w.retainedSinceBoundary {
		return
	}
	w.recordEntry(ForegroundJournalEntryBoundary{IntervalBoundaryFrameSkipped{skipped}})
}

func (w *foregroundJournalWriter) recordFramePending(windowID WindowID, dirtyAt scheduler.Instant) {
	window, ok := w.windows[windowID]
	if !ok {
		window = newWindowFrameState(WindowVisibilityVisible)
		w.windows[windowID] = window
	}
	if w.power == powerStateSuspended || !window.visibility.IsVisible() {
		return
	}
	if at, ok := window.dirtyAt.Get(); ok && dirtyAt.SaturatingDurationSince(at) < FrameDeadline {
		return
	}
	window.dirtyAt = Some(dirtyAt)
	w.recordFrameState(FrameStateChangePending{WindowID: windowID, DirtyAt: dirtyAt})
}

func (w *foregroundJournalWriter) recordWindowClosed(windowID WindowID, at scheduler.Instant) {
	delete(w.windows, windowID)
	w.recordFrameState(FrameStateChangeClosed{WindowID: windowID, At: at})
}

func (w *foregroundJournalWriter) recordPresent(timing PresentTiming, frame Option[FrameTiming]) {
	if window, ok := w.windows[timing.WindowID]; ok {
		window.dirtyAt = None[scheduler.Instant]()
	}
	if w.powerInterruptedSince(timing.PresentStart) {
		return
	}
	if f, ok := frame.Get(); ok {
		w.recordEntry(ForegroundJournalEntryBoundary{IntervalBoundaryPresented{PresentedFrame{Frame: f, Presentation: timing}}})
	} else {
		w.recordEvent(ForegroundEventPresent{timing})
	}
}

var foregroundRunnables = newForegroundRunnableCounter()

var foregroundJournal struct {
	mu     sync.Mutex
	writer *foregroundJournalWriter
}

func currentForegroundRunnableCounter() foregroundRunnableCounter {
	return foregroundRunnables
}

func installForegroundJournal() ForegroundJournal {
	runnables := currentForegroundRunnableCounter()
	foregroundJournal.mu.Lock()
	defer foregroundJournal.mu.Unlock()
	if foregroundJournal.writer != nil {
		return ForegroundJournal{ring: foregroundJournal.writer.publisher.ring}
	}
	handle, publisher := newForegroundJournal(int(maxJournalEntries), maxPendingJournalEntries)
	foregroundJournal.writer = newForegroundJournalWriter(runnables, publisher)
	return handle
}

type testForegroundJournalGuard struct {
	previous *foregroundJournalWriter
	ended    bool
}

// End restores the journal that was installed before installTestForegroundJournal.
func (g *testForegroundJournalGuard) End() {
	if g.ended {
		return
	}
	g.ended = true
	foregroundJournal.mu.Lock()
	defer foregroundJournal.mu.Unlock()
	foregroundJournal.writer = g.previous
	g.previous = nil
}

func installTestForegroundJournal(capacity, pendingCapacity int) (ForegroundJournal, *testForegroundJournalGuard) {
	runnables := newForegroundRunnableCounter()
	handle, publisher := newForegroundJournal(capacity, pendingCapacity)
	writer := newForegroundJournalWriter(runnables, publisher).withTestSuspendClock()
	foregroundJournal.mu.Lock()
	defer foregroundJournal.mu.Unlock()
	previous := foregroundJournal.writer
	foregroundJournal.writer = writer
	return handle, &testForegroundJournalGuard{previous: previous}
}

var testSuspendedTimeNanos atomic.Int64

func testSuspendedTime() (time.Duration, bool) {
	return time.Duration(testSuspendedTimeNanos.Load()), true
}

func advanceTestSuspendedTime(duration time.Duration) {
	testSuspendedTimeNanos.Add(int64(duration))
}

func (w *foregroundJournalWriter) withTestSuspendClock() *foregroundJournalWriter {
	testSuspendedTimeNanos.Store(0)
	w.readSuspendedTime = testSuspendedTime
	w.suspendedTime = OptionFrom(testSuspendedTime())
	return w
}

func withJournal(f func(journal *foregroundJournalWriter)) {
	foregroundJournal.mu.Lock()
	defer foregroundJournal.mu.Unlock()
	if foregroundJournal.writer != nil {
		f(foregroundJournal.writer)
	}
}

func observePower(cx *App) {
	cx.OnSystemSleep(func(*App) {
		recordPowerTransition(powerStateSuspended)
	}).Detach()
	cx.OnSystemWake(func(cx *App) {
		recordPowerTransition(powerStateAwake)
		cx.Spawn(func(cx *AsyncApp) (struct{}, error) {
			cx.Update(func(cx *App) {
				for _, handle := range cx.Windows() {
					if err := handle.Update(cx, func(_ AnyView, window *Window, cx *App) {
						window.refreshVisibility(cx)
					}); err != nil {
						gpuiutil.LogErr(err)
					}
				}
			})
			return struct{}{}, nil
		}).Detach()
	}).Detach()
}

var systemSuspendedTime = func() (time.Duration, bool) {
	return 0, false
}

func recordPowerTransition(state powerState) {
	withJournal(func(journal *foregroundJournalWriter) {
		journal.powerTransition(state, scheduler.InstantNow())
	})
}

func powerInterruptedSince(start scheduler.Instant) bool {
	interrupted := false
	withJournal(func(journal *foregroundJournalWriter) {
		journal.detectSuspend(scheduler.InstantNow())
		interrupted = journal.powerInterruptedSince(start)
	})
	return interrupted
}

func recordWindowVisibility(id WindowID, visibility WindowVisibility) {
	withJournal(func(journal *foregroundJournalWriter) {
		journal.setVisibility(id, visibility, scheduler.InstantNow())
	})
}

func frameSampleIsValid(id WindowID, start scheduler.Instant) bool {
	valid := true
	withJournal(func(journal *foregroundJournalWriter) {
		journal.detectSuspend(scheduler.InstantNow())
		valid = !journal.powerInterruptedSince(start)
		if window, ok := journal.windows[id]; ok && valid {
			changedAt, changed := window.visibilityChangedAt.Get()
			valid = window.visibility.IsVisible() && (!changed || start.After(changedAt))
		}
	})
	return valid
}

type foregroundTurnGuard struct{}

// End ends the foreground turn begun by foregroundTurn.
func (foregroundTurnGuard) End() {
	endForegroundTurn()
}

func foregroundTurn() foregroundTurnGuard {
	beginForegroundTurn()
	return foregroundTurnGuard{}
}

func beginForegroundTurn() {
	withJournal(func(journal *foregroundJournalWriter) {
		journal.detectSuspend(scheduler.InstantNow())
		journal.beginTurn()
	})
}

func endForegroundTurn() {
	withJournal(func(journal *foregroundJournalWriter) {
		journal.endTurn(scheduler.InstantNow())
	})
}

func recordTaskPoll(timing TaskTiming) {
	foregroundRunnables.finished()
	withJournal(func(journal *foregroundJournalWriter) {
		journal.detectSuspend(timing.End.Instant)
		if journal.powerInterruptedSince(timing.Start) {
			journal.endTurn(timing.End.Instant)
			return
		}
		if timing.PollDuration() >= TaskPollFloor {
			journal.recordEvent(ForegroundEventTaskPoll{timing})
		} else {
			journal.foldSmallPoll(timing)
		}
		journal.endTurn(timing.End.Instant)
	})
}

func recordAction(timing ActionTiming) {
	withJournal(func(journal *foregroundJournalWriter) { journal.recordEvent(ForegroundEventAction{timing}) })
}

func recordInput(timing InputTiming) {
	withJournal(func(journal *foregroundJournalWriter) { journal.recordEvent(ForegroundEventInput{timing}) })
}

func recordView(timing ViewTiming) {
	withJournal(func(journal *foregroundJournalWriter) { journal.recordEvent(ForegroundEventView{timing}) })
}

func recordDraw(timing FrameTiming) {
	withJournal(func(journal *foregroundJournalWriter) { journal.recordEvent(ForegroundEventDraw{timing}) })
}

func recordPresent(timing PresentTiming, frame Option[FrameTiming]) {
	withJournal(func(journal *foregroundJournalWriter) { journal.recordPresent(timing, frame) })
}

func recordPlatformSignal(signal PlatformSignal) {
	withJournal(func(journal *foregroundJournalWriter) { journal.recordPlatformSignal(signal) })
}

func recordReentrantFrameSkipped(windowID WindowID, at scheduler.Instant) {
	withJournal(func(journal *foregroundJournalWriter) {
		journal.recordMarker(ForegroundJournalEntryReentrantFrameSkipped{WindowID: windowID, At: at})
	})
}

func recordFrameSkipped(windowID WindowID, at scheduler.Instant, reason FrameSkipReason) {
	withJournal(func(journal *foregroundJournalWriter) {
		journal.recordFrameSkipped(FrameSkipped{WindowID: windowID, At: at, Reason: reason})
	})
}

func recordFramePending(windowID WindowID, dirtyAt scheduler.Instant) {
	withJournal(func(journal *foregroundJournalWriter) { journal.recordFramePending(windowID, dirtyAt) })
}

func recordWindowClosed(windowID WindowID) {
	at := scheduler.InstantNow()
	withJournal(func(journal *foregroundJournalWriter) { journal.recordWindowClosed(windowID, at) })
}

const (
	slotWriter     uint64 = 1 << 63
	slotReaderMask uint64 = ^slotWriter
	emptySequence  uint64 = math.MaxUint64
)

type journalSlot struct {
	users    atomic.Uint64
	sequence atomic.Uint64
	entry    ForegroundJournalEntry
}

func (s *journalSlot) tryPublish(sequence uint64, entry ForegroundJournalEntry) bool {
	if !s.users.CompareAndSwap(0, slotWriter) {
		return false
	}
	s.entry = entry
	s.sequence.Store(sequence)
	s.users.Store(0)
	return true
}

func (s *journalSlot) tryRead(expectedSequence uint64) (ForegroundJournalEntry, bool) {
	guard, ok := tryNewJournalSlotReadGuard(s)
	if !ok {
		return nil, false
	}
	defer guard.End()
	if s.sequence.Load() != expectedSequence {
		return nil, false
	}
	return s.entry, true
}

func (s *journalSlot) tryAddReader() bool {
	users := s.users.Load()
	for {
		if users&slotWriter != 0 || users == slotReaderMask {
			return false
		}
		if s.users.CompareAndSwap(users, users+1) {
			return true
		}
		users = s.users.Load()
	}
}

func (s *journalSlot) removeReader() {
	previous := s.users.Add(^uint64(0)) + 1
	if gpuiutil.DebugAssertions && (previous == 0 || previous&slotWriter != 0) {
		panic(fmt.Sprintf("invalid slot reader state: %#x", previous))
	}
}

type journalSlotReadGuard struct {
	slot  *journalSlot
	ended bool
}

func tryNewJournalSlotReadGuard(slot *journalSlot) (*journalSlotReadGuard, bool) {
	if slot.tryAddReader() {
		return &journalSlotReadGuard{slot: slot}, true
	}
	return nil, false
}

// End releases the reader pin on the slot.
func (g *journalSlotReadGuard) End() {
	if g.ended {
		return
	}
	g.ended = true
	g.slot.removeReader()
}

type journalRing struct {
	slots     []journalSlot
	finalized atomic.Uint64
	offered   atomic.Uint64
}

func newJournalRing(capacity int) *journalRing {
	capacity = max(capacity, 1)
	ring := &journalRing{slots: make([]journalSlot, capacity)}
	for i := range ring.slots {
		ring.slots[i].sequence.Store(emptySequence)
	}
	return ring
}

func (r *journalRing) capacity() int {
	return len(r.slots)
}

func (r *journalRing) tryPublish(sequence uint64, entry ForegroundJournalEntry) bool {
	return r.slots[sequence%uint64(len(r.slots))].tryPublish(sequence, entry)
}

func (r *journalRing) read(sequence uint64) (ForegroundJournalEntry, bool) {
	return r.slots[sequence%uint64(len(r.slots))].tryRead(sequence)
}

type pendingJournalEntry struct {
	sequence uint64
	entry    ForegroundJournalEntry
}

type journalPublisher struct {
	ring                *journalRing
	nextSequence        uint64
	pending             []pendingJournalEntry
	droppedAfterPending uint64
	pendingCapacity     int
}

func newJournalPublisher(ring *journalRing, pendingCapacity int) *journalPublisher {
	return &journalPublisher{
		ring:            ring,
		pending:         make([]pendingJournalEntry, 0, pendingCapacity),
		pendingCapacity: pendingCapacity,
	}
}

func (p *journalPublisher) publish(entries ...ForegroundJournalEntry) {
	for _, entry := range entries {
		p.flushPending()
		p.publishOne(entry)
	}
	p.flushPending()
}

func (p *journalPublisher) publishOne(entry ForegroundJournalEntry) {
	sequence := p.nextSequence
	p.nextSequence++
	p.ring.offered.Store(p.nextSequence)

	if len(p.pending) > 0 || p.droppedAfterPending > 0 {
		if p.droppedAfterPending == 0 && len(p.pending) < p.pendingCapacity {
			p.pending = append(p.pending, pendingJournalEntry{sequence: sequence, entry: entry})
		} else {
			p.droppedAfterPending++
		}
		return
	}

	if p.ring.tryPublish(sequence, entry) {
		p.ring.finalized.Store(sequence + 1)
	} else if p.pendingCapacity > 0 {
		p.pending = append(p.pending, pendingJournalEntry{sequence: sequence, entry: entry})
	} else {
		p.droppedAfterPending = 1
	}
}

func (p *journalPublisher) flushPending() {
	for len(p.pending) > 0 {
		front := p.pending[0]
		if !p.ring.tryPublish(front.sequence, front.entry) {
			return
		}
		p.pending[0] = pendingJournalEntry{}
		p.pending = p.pending[1:]
		p.ring.finalized.Store(front.sequence + 1)
	}
	if len(p.pending) == 0 && cap(p.pending) < p.pendingCapacity {
		p.pending = make([]pendingJournalEntry, 0, p.pendingCapacity)
	}

	if p.droppedAfterPending > 0 {
		p.ring.finalized.Store(p.nextSequence)
		p.droppedAfterPending = 0
	}
}

// ForegroundJournal is a cloneable handle to one foreground journal stream. Each collector has
// an independent cursor; apps on the same foreground thread share the stream.
type ForegroundJournal struct {
	ring *journalRing
}

func newForegroundJournal(capacity, pendingCapacity int) (ForegroundJournal, *journalPublisher) {
	ring := newJournalRing(capacity)
	return ForegroundJournal{ring: ring}, newJournalPublisher(ring, pendingCapacity)
}

// Collector creates an independent collector that observes entries offered after this call.
func (j ForegroundJournal) Collector() *ForegroundJournalCollector {
	return &ForegroundJournalCollector{
		cursor: j.ring.offered.Load(),
		ring:   j.ring,
	}
}

// DrainedEntries is the entries returned by one ForegroundJournalCollector.CollectUnseen call.
type DrainedEntries struct {
	// Entries is the journal entries recorded since the previous drain, in recording order,
	// including ForegroundJournalEntryDiscontinuity markers at unavailable positions.
	Entries []ForegroundJournalEntry
	// Lost is the number of entries unavailable to this collector.
	Lost uint64
}

// ForegroundJournalCollector reads the foreground stream with its own cursor.
type ForegroundJournalCollector struct {
	cursor uint64
	ring   *journalRing
}

// CollectUnseen returns entries recorded since the previous call or the collector's creation,
// reporting how many were unavailable.
func (c *ForegroundJournalCollector) CollectUnseen() DrainedEntries {
	end := c.ring.finalized.Load()
	if c.cursor >= end {
		return DrainedEntries{}
	}
	retainedStart := saturatingSubU64(end, uint64(c.ring.capacity()))
	lost := saturatingSubU64(retainedStart, c.cursor)
	c.cursor = max(c.cursor, retainedStart)
	extra := 0
	if lost > 0 {
		extra = 1
	}
	entries := make([]ForegroundJournalEntry, 0, int(end-c.cursor)+extra)
	if lost > 0 {
		entries = append(entries, ForegroundJournalEntryDiscontinuity{Lost: lost})
	}

	for c.cursor < end {
		if entry, ok := c.ring.read(c.cursor); ok {
			entries = append(entries, entry)
		} else {
			lost++
			last := len(entries) - 1
			if d, ok := lastDiscontinuity(entries); ok {
				entries[last] = ForegroundJournalEntryDiscontinuity{Lost: d.Lost + 1}
			} else {
				entries = append(entries, ForegroundJournalEntryDiscontinuity{Lost: 1})
			}
		}
		c.cursor++
	}

	return DrainedEntries{Entries: entries, Lost: lost}
}

func lastDiscontinuity(entries []ForegroundJournalEntry) (ForegroundJournalEntryDiscontinuity, bool) {
	if len(entries) == 0 {
		return ForegroundJournalEntryDiscontinuity{}, false
	}
	d, ok := entries[len(entries)-1].(ForegroundJournalEntryDiscontinuity)
	return d, ok
}

// IntervalSealer is a pure state machine that groups foreground journal entries into FrameSnapshots.
type IntervalSealer struct {
	intervalStart        scheduler.Instant
	events               []ForegroundEvent
	smallPolls           []SmallPollFlush
	droppedEvents        uint64
	journalDiscontinuous bool
}

// NewIntervalSealer creates a sealer whose first interval starts at start.
func NewIntervalSealer(start scheduler.Instant) *IntervalSealer {
	return &IntervalSealer{intervalStart: start}
}

// NoteLost accounts for a loss supplied outside a collector drain.
func (s *IntervalSealer) NoteLost(lost uint64) {
	s.droppedEvents += lost
	s.journalDiscontinuous = s.journalDiscontinuous || lost > 0
}

// PushEntries processes a batch of drained entries, returning the snapshots completed by
// explicit boundaries. Work without a following boundary is carried over.
func (s *IntervalSealer) PushEntries(entries []ForegroundJournalEntry) []FrameSnapshot {
	snapshots := []FrameSnapshot{}
	for _, entry := range entries {
		switch entry := entry.(type) {
		case ForegroundJournalEntryEvent:
			if s.isEmpty() {
				s.intervalStart = instantMax(s.intervalStart, entry.Event.StartTime())
			}
			if flush, ok := entry.Event.(ForegroundEventSmallPolls); ok {
				s.pushSmallPolls(flush.SmallPollFlush)
			} else {
				s.pushEvent(entry.Event)
			}
		case ForegroundJournalEntryBoundary:
			if presented, ok := entry.Boundary.(IntervalBoundaryPresented); ok {
				if s.isEmpty() {
					s.intervalStart = instantMax(s.intervalStart, presented.Presentation.PresentStart)
				}
				s.pushEvent(ForegroundEventPresent{presented.Presentation})
			}
			if s.isEmpty() {
				s.intervalStart = instantMax(s.intervalStart, entry.Boundary.EndTime())
			} else {
				snapshots = append(snapshots, s.seal(entry.Boundary))
			}
		case ForegroundJournalEntryFrameState, ForegroundJournalEntryPlatformSignal, ForegroundJournalEntryReentrantFrameSkipped:
		case ForegroundJournalEntryDiscontinuity:
			s.NoteLost(entry.Lost)
		}
	}
	return snapshots
}

func (s *IntervalSealer) isEmpty() bool {
	return len(s.events) == 0 && len(s.smallPolls) == 0 && s.droppedEvents == 0 && !s.journalDiscontinuous
}

func (s *IntervalSealer) pushEvent(event ForegroundEvent) {
	if len(s.events) >= maxIntervalEvents {
		s.droppedEvents++
	} else {
		s.events = append(s.events, event)
	}
}

func (s *IntervalSealer) pushSmallPolls(flush SmallPollFlush) {
	if len(s.smallPolls) >= maxIntervalEvents && len(s.smallPolls) > 0 {
		last := &s.smallPolls[len(s.smallPolls)-1]
		last.Summary.add(flush.Summary)
		last.Since = last.Since.Min(flush.Since)
		last.Until = instantMax(last.Until, flush.Until)
	} else {
		s.smallPolls = append(s.smallPolls, flush)
	}
}

func (s *IntervalSealer) seal(boundary IntervalBoundary) FrameSnapshot {
	ended := boundary.EndTime()
	snapshot := FrameSnapshot{
		IntervalStart:        s.intervalStart,
		Boundary:             boundary,
		Events:               s.events,
		SmallPolls:           s.smallPolls,
		DroppedEvents:        s.droppedEvents,
		JournalDiscontinuous: s.journalDiscontinuous,
	}
	s.events = nil
	s.smallPolls = nil
	s.droppedEvents = 0
	s.journalDiscontinuous = false
	s.intervalStart = ended
	return snapshot
}
