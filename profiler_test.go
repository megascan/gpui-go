//go:build gpui_profiler

package gpui

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func TestProfiler_InterruptionsDropLatencySamplesButHidingKeepsDrawWork(t *testing.T) {
	t.Skip("blocked:M3-profiler-journal: needs install_test_foreground_journal, record_power_transition and record_window_visibility")
}

func TestProfiler_FrameWorkStartsAtTheLaterOfSignalAndInvalidation(t *testing.T) {
	refresh := 10 * time.Millisecond
	start := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return start.Add(time.Duration(ms) * time.Millisecond) }
	optAt := func(ms Option[uint64]) Option[scheduler.Instant] {
		if v, ok := ms.Get(); ok {
			return Some(at(v))
		}
		return None[scheduler.Instant]()
	}
	frameWith := func(dirty, signal Option[uint64]) FrameTiming {
		return FrameTiming{
			WindowID:        WindowID(1),
			DirtyAt:         optAt(dirty),
			Invalidations:   1,
			DrawStart:       at(1),
			DrawEnd:         at(2),
			RefreshInterval: Some(refresh),
			SignalAt:        optAt(signal),
		}
	}
	work := func(dirty, signal Option[uint64]) Option[time.Duration] {
		return OptionFrom(frameWith(dirty, signal).Work(at(50)))
	}
	ms := func(v time.Duration) Option[time.Duration] { return Some(v * time.Millisecond) }

	profilerAssertEq(t, work(Some[uint64](0), Some[uint64](8)), ms(42))
	profilerAssertEq(t, work(Some[uint64](0), Some[uint64](40)), ms(10))
	profilerAssertEq(t, work(Some[uint64](5), Some[uint64](0)), ms(45))
	profilerAssertEq(t, work(None[uint64](), Some[uint64](0)), ms(50))
	profilerAssertEq(t, work(Some[uint64](0), None[uint64]()), None[time.Duration]())

	frame := frameWith(Some[uint64](0), Some[uint64](0))
	for _, c := range [][2]uint64{{7, 0}, {10, 0}, {20, 1}, {25, 2}} {
		profilerAssertEq(t, OptionFrom(frame.MissedRefreshes(at(c[0]))), Some(c[1]))
	}
}

func TestProfiler_FramesAreSplitByRefreshIntervalAndClassified(t *testing.T) {
	profiler := mustWindowProfiler(t, WindowID(1))
	defer profiler.Close()
	sixtyHertz := time.Second / 60
	oneHundredTwentyHertz := time.Second / 120
	threeHundredSixtyHertz := time.Second / 360
	base := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return base.Add(time.Duration(ms) * time.Millisecond) }

	type frameCase struct {
		refreshInterval  Option[time.Duration]
		signalMs         Option[uint64]
		presentStartMs   uint64
		respondedToInput bool
	}
	frames := []frameCase{
		{Some(sixtyHertz), Some[uint64](1), 5, true},
		{Some(sixtyHertz), Some[uint64](0), 40, true},
		{Some(sixtyHertz), None[uint64](), 5, false},
		{Some(sixtyHertz), Some[uint64](40), 42, false},
		{Some(oneHundredTwentyHertz), Some[uint64](0), 12, false},
		{Some(threeHundredSixtyHertz), Some[uint64](0), 6, false},
		{Some(threeHundredSixtyHertz), Some[uint64](0), 9, false},
		{None[time.Duration](), Some[uint64](0), 2, false},
	}
	for _, frame := range frames {
		profiler.SetRefreshInterval(frame.refreshInterval)
		signalAt := None[scheduler.Instant]()
		if v, ok := frame.signalMs.Get(); ok {
			signalAt = Some(at(v))
		}
		profiler.recordDrawTiming(FrameTiming{
			WindowID:        profiler.windowID,
			DirtyAt:         Some(at(0)),
			Invalidations:   1,
			DrawStart:       at(0),
			DrawEnd:         at(1),
			RefreshInterval: frame.refreshInterval,
			SignalAt:        signalAt,
		})
		if frame.respondedToInput {
			profiler.firstInputAt = Some(at(0))
		}
		profiler.recordPresentAt(at(frame.presentStartMs), at(frame.presentStartMs+1), true, false)
	}

	snapshot := profiler.FrameDurationSnapshot()
	framesFor := func(refreshInterval Option[time.Duration]) RefreshIntervalFrames {
		index := slices.IndexFunc(snapshot.ByRefreshInterval, func(f RefreshIntervalFrames) bool {
			return f.RefreshInterval == refreshInterval
		})
		if index < 0 {
			t.Fatal("frames for the refresh interval")
		}
		return snapshot.ByRefreshInterval[index].clone()
	}
	workBuckets := func(work *Histogram, refreshInterval time.Duration) []uint64 {
		bounds := make([]uint64, len(FrameWorkBucketsPercent))
		for i, percent := range FrameWorkBucketsPercent {
			bounds[i] = uint64(refreshInterval * time.Duration(uint32(percent)) / 100)
		}
		return HistogramBucketCounts(work, bounds)
	}
	sixty := framesFor(Some(sixtyHertz))
	profilerAssertEq(t, sixty.FramesDrawn, uint64(4))
	profilerAssertEq(t, sixty.FramesOnTime, uint64(2))
	profilerAssertEq(t, sixty.FramesLate, uint64(1))
	profilerAssertEq(t, sixty.MissedRefreshes, uint64(2))
	profilerAssertEq(t, sixty.FramesUnmeasured, uint64(1))
	profilerAssertSliceEq(t, workBuckets(sixty.Work, sixtyHertz), []uint64{2, 0, 0, 0, 0, 0, 1, 0})
	profilerAssertEq(t, sixty.InputFramesOnTime, uint64(1))
	profilerAssertEq(t, sixty.InputFramesLate, uint64(1))
	profilerAssertSliceEq(t, workBuckets(sixty.InputWork, sixtyHertz), []uint64{1, 0, 0, 0, 0, 0, 1, 0})

	oneHundredTwenty := framesFor(Some(oneHundredTwentyHertz))
	profilerAssertEq(t, oneHundredTwenty.FramesLate, uint64(1))
	profilerAssertEq(t, oneHundredTwenty.MissedRefreshes, uint64(1))
	profilerAssertEq(t, oneHundredTwenty.InputFramesLate, uint64(0))

	threeHundredSixty := framesFor(Some(threeHundredSixtyHertz))
	profilerAssertEq(t, threeHundredSixty.FramesOnTime, uint64(1))
	profilerAssertEq(t, threeHundredSixty.FramesLate, uint64(1))
	profilerAssertEq(t, threeHundredSixty.MissedRefreshes, uint64(5))

	unknown := framesFor(None[time.Duration]())
	profilerAssertEq(t, unknown.FramesDrawn, uint64(1))
	profilerAssertEq(t, unknown.FramesOnTime, uint64(1))
	profilerAssertEq(t, unknown.MissedRefreshes, uint64(0))

	later := profiler.FrameDurationSnapshot()
	delta := later.ByRefreshInterval[0].Since(snapshot.ByRefreshInterval[0])
	profilerAssertEq(t, delta.FramesDrawn+delta.FramesOnTime+delta.FramesLate, uint64(0))
	profilerAssertEq(t, delta.DrawDuration.Len()+delta.Work.Len()+delta.InputWork.Len(), uint64(0))
}

func TestProfiler_RecordsDrawEventsOnlyWhileTracing(t *testing.T) {
	guard := newTraceTestGuard()
	defer guard.End()
	windowID := WindowID(0xD0A0)
	windowProfiler := mustWindowProfiler(t, windowID)
	defer windowProfiler.Close()
	dirtyAt := scheduler.InstantNow()
	collector := NewFrameTimingCollector()

	windowProfiler.BeginDraw()
	windowProfiler.EndDraw(Some(dirtyAt), 3)
	for _, event := range collector.CollectUnseen() {
		if eventMatchesWindow(event, windowID) {
			t.Fatal("draw event recorded while not tracing")
		}
	}

	SetTraceEnabled(true)
	collector = NewFrameTimingCollector()
	windowProfiler.BeginDraw()
	windowProfiler.EndDraw(Some(dirtyAt), 3)

	var timing FrameTiming
	found := false
	for _, event := range collector.CollectUnseen() {
		if draw, ok := event.(FrameEventDraw); ok && draw.WindowID == windowID {
			timing, found = draw.FrameTiming, true
			break
		}
	}
	if !found {
		t.Fatal("draw event should be recorded while tracing")
	}
	got, ok := timing.DirtyAt.Get()
	if !ok || !got.Equal(dirtyAt) {
		t.Fatalf("dirty_at = %v, want %v", timing.DirtyAt, dirtyAt)
	}
	profilerAssertEq(t, timing.Invalidations, uint64(3))
	if timing.DrawStart.Before(dirtyAt) {
		t.Fatal("draw_start before dirty_at")
	}
}

func TestProfiler_RecordsPresentEventsForNewlyDrawnFrames(t *testing.T) {
	guard := newTraceTestGuard()
	defer guard.End()
	SetTraceEnabled(true)
	windowID := WindowID(0xA11E)
	windowProfiler := mustWindowProfiler(t, windowID)
	defer windowProfiler.Close()
	start := scheduler.InstantNow()
	collector := NewFrameTimingCollector()

	recordTestDraw(windowProfiler, start)
	windowProfiler.recordPresentAt(start, start, true, true)
	recordTestDraw(windowProfiler, start.Add(testFrame))
	windowProfiler.recordPresentAt(start.Add(testFrame), start.Add(testFrame), true, true)
	windowProfiler.recordPresentAt(start.Add(testFrame+testFrame/2), start.Add(testFrame+testFrame/2), true, true)

	var presentTimings []PresentTiming
	for _, event := range collector.CollectUnseen() {
		if present, ok := event.(FrameEventPresent); ok && present.WindowID == windowID {
			presentTimings = append(presentTimings, present.PresentTiming)
		}
	}
	if len(presentTimings) != 2 {
		t.Fatalf("expected exactly two present events, got %v", presentTimings)
	}
	profilerAssertEq(t, presentTimings[0].AnimationInterval, None[time.Duration]())
	profilerAssertEq(t, presentTimings[1].AnimationInterval, Some(testFrame))

	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(1))
	if windowProfiler.presentIntervalHistogram.Max() < uint64(presentTimings[1].AnimationInterval.Expect("second present should have an animation interval")) {
		t.Fatal("present interval max too small")
	}
}

func TestProfiler_DisablingTracingClearsFrameEvents(t *testing.T) {
	guard := newTraceTestGuard()
	defer guard.End()
	SetTraceEnabled(true)
	windowID := WindowID(0xC1EA)
	windowProfiler := mustWindowProfiler(t, windowID)
	defer windowProfiler.Close()
	collector := NewFrameTimingCollector()

	windowProfiler.BeginDraw()
	windowProfiler.EndDraw(None[scheduler.Instant](), 0)
	frameTimings.mu.Lock()
	recorded := slices.ContainsFunc(frameTimings.timings, func(event FrameEvent) bool { return eventMatchesWindow(event, windowID) })
	frameTimings.mu.Unlock()
	if !recorded {
		t.Fatal("draw event should be recorded while tracing")
	}

	SetTraceEnabled(false)
	for _, event := range collector.CollectUnseen() {
		if eventMatchesWindow(event, windowID) {
			t.Fatal("frame events should be cleared")
		}
	}
}

func TestProfiler_RecordsIntervalsOnlyBetweenAnimationFrames(t *testing.T) {
	windowProfiler := mustWindowProfiler(t, WindowID(1))
	defer windowProfiler.Close()
	start := scheduler.InstantNow()

	drawAndPresent(windowProfiler, start, true, true)
	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(0))
	drawAndPresent(windowProfiler, start.Add(testFrame), true, true)
	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(1))
	drawAndPresent(windowProfiler, start.Add(testFrame*2), true, false)
	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(2))
	drawAndPresent(windowProfiler, start.Add(testFrame*100), true, true)
	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(2))
}

func TestProfiler_MissedFramesStretchTheRecordedInterval(t *testing.T) {
	windowProfiler := mustWindowProfiler(t, WindowID(2))
	defer windowProfiler.Close()
	start := scheduler.InstantNow()

	drawAndPresent(windowProfiler, start, true, true)
	drawAndPresent(windowProfiler, start.Add(testFrame*5), true, true)

	if recorded := windowProfiler.presentIntervalHistogram.Max(); recorded < uint64(testFrame*4) {
		t.Fatalf("recorded %d", recorded)
	}
}

func TestProfiler_IgnoresRePresentsOfUnchangedFrames(t *testing.T) {
	windowProfiler := mustWindowProfiler(t, WindowID(3))
	defer windowProfiler.Close()
	start := scheduler.InstantNow()

	drawAndPresent(windowProfiler, start, true, true)
	windowProfiler.recordPresentAt(start.Add(testFrame/2), start.Add(testFrame/2), true, true)
	drawAndPresent(windowProfiler, start.Add(testFrame), true, true)

	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(1))
	if windowProfiler.presentIntervalHistogram.Max() < uint64(testFrame*3/4) {
		t.Fatal("present interval max too small")
	}
}

func TestProfiler_SkipsIntervalsForInactiveWindows(t *testing.T) {
	windowProfiler := mustWindowProfiler(t, WindowID(4))
	defer windowProfiler.Close()
	start := scheduler.InstantNow()

	drawAndPresent(windowProfiler, start, false, true)
	drawAndPresent(windowProfiler, start.Add(testFrame), false, true)
	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(0))
	drawAndPresent(windowProfiler, start.Add(testFrame*2), true, true)
	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(0))
	drawAndPresent(windowProfiler, start.Add(testFrame*3), true, true)
	profilerAssertEq(t, windowProfiler.presentIntervalHistogram.Len(), uint64(1))
}

func TestProfiler_RecordsDirtyToPresentDurations(t *testing.T) {
	windowProfiler := mustWindowProfiler(t, WindowID(8))
	defer windowProfiler.Close()
	drawEnd := scheduler.InstantNow()
	presentEnd := drawEnd.Add(6 * time.Millisecond)

	recordTestDraw(windowProfiler, drawEnd)
	windowProfiler.recordPresentAt(presentEnd, presentEnd, true, false)

	histogram := windowProfiler.FrameDurationSnapshot().DirtyToPresentHistogram
	profilerAssertEq(t, histogram.Len(), uint64(1))
	if histogram.Max() < uint64(10*time.Millisecond) {
		t.Fatal("dirty-to-present max too small")
	}
}

func TestProfiler_RecordsEveryDrawDuration(t *testing.T) {
	windowProfiler := mustWindowProfiler(t, WindowID(5))
	defer windowProfiler.Close()

	windowProfiler.recordDrawDuration(2 * time.Millisecond)
	windowProfiler.recordDrawDuration(40 * time.Millisecond)

	snapshot := windowProfiler.FrameDurationSnapshot()
	profilerAssertEq(t, snapshot.DrawDurationHistogram.Len(), uint64(2))
	if snapshot.DrawDurationHistogram.Max() < 39_000_000 {
		t.Fatal("draw duration max too small")
	}
}

func TestProfiler_RecordsInputLatencyAtTheFramePresentationTimestamp(t *testing.T) {
	windowProfiler := mustWindowProfiler(t, WindowID(6))
	defer windowProfiler.Close()
	firstInputAt := scheduler.InstantNow()
	presentedAt := firstInputAt.Add(12 * time.Millisecond)

	beginInputAt(windowProfiler, firstInputAt)
	windowProfiler.EndInput(true)
	beginInputAt(windowProfiler, firstInputAt.Add(2*time.Millisecond))
	windowProfiler.EndInput(true)
	recordTestDraw(windowProfiler, presentedAt)
	windowProfiler.recordPresentAt(presentedAt, presentedAt, true, false)

	snapshot := windowProfiler.InputLatencySnapshot()
	profilerAssertEq(t, snapshot.LatencyHistogram.Len(), uint64(1))
	if snapshot.LatencyHistogram.Max() < uint64(12*time.Millisecond) {
		t.Fatal("latency max too small")
	}
	profilerAssertEq(t, snapshot.EventsPerFrameHistogram.Len(), uint64(1))
	profilerAssertEq(t, snapshot.EventsPerFrameHistogram.Max(), uint64(2))
	profilerAssertEq(t, snapshot.MidDrawEventsDropped, uint64(0))
}

func TestProfiler_ExcludesInputThatArrivesDuringADraw(t *testing.T) {
	windowProfiler := mustWindowProfiler(t, WindowID(7))
	defer windowProfiler.Close()

	windowProfiler.BeginDraw()
	beginInputAt(windowProfiler, scheduler.InstantNow())
	windowProfiler.EndInput(true)
	windowProfiler.EndDraw(None[scheduler.Instant](), 0)

	snapshot := windowProfiler.InputLatencySnapshot()
	if !snapshot.LatencyHistogram.IsEmpty() || !snapshot.EventsPerFrameHistogram.IsEmpty() {
		t.Fatal("histograms should be empty")
	}
	profilerAssertEq(t, snapshot.MidDrawEventsDropped, uint64(1))
}

func TestProfiler_OverlappingTraceScopesKeepTracingEnabled(t *testing.T) {
	guard := newTraceTestGuard()
	defer guard.End()
	firstScope := traceScope()
	secondScope := traceScope()

	if !TraceEnabled() {
		t.Fatal("tracing should be enabled")
	}
	firstScope.End()
	if !TraceEnabled() {
		t.Fatal("tracing should stay enabled")
	}
	secondScope.End()
	if TraceEnabled() {
		t.Fatal("tracing should be disabled")
	}
}

const testFrame = 16 * time.Millisecond

var traceTestLock sync.Mutex

type traceTestGuard struct {
	wasEnabled bool
}

func newTraceTestGuard() *traceTestGuard {
	traceTestLock.Lock()
	wasEnabled := TraceEnabled()
	SetTraceEnabled(false)
	return &traceTestGuard{wasEnabled: wasEnabled}
}

func (g *traceTestGuard) End() {
	SetTraceEnabled(false)
	if g.wasEnabled {
		SetTraceEnabled(true)
	}
	traceTestLock.Unlock()
}

func mustWindowProfiler(t *testing.T, id WindowID) *WindowProfiler {
	t.Helper()
	p, err := NewWindowProfiler(id)
	if err != nil {
		t.Fatalf("window profiler should initialize: %v", err)
	}
	return p
}

func eventMatchesWindow(event FrameEvent, windowID WindowID) bool {
	switch e := event.(type) {
	case FrameEventDraw:
		return e.WindowID == windowID
	case FrameEventPresent:
		return e.WindowID == windowID
	}
	return false
}

func beginInputAt(windowProfiler *WindowProfiler, startedAt scheduler.Instant) {
	windowProfiler.activeActivities = append(windowProfiler.activeActivities, windowActivity{startedAt: startedAt, kind: "test"})
}

func drawAndPresent(windowProfiler *WindowProfiler, presentedAt scheduler.Instant, windowActive, nextFrameScheduled bool) {
	recordTestDraw(windowProfiler, presentedAt)
	windowProfiler.recordPresentAt(presentedAt, presentedAt, windowActive, nextFrameScheduled)
}

func recordTestDraw(windowProfiler *WindowProfiler, drawEnd scheduler.Instant) {
	windowProfiler.recordDrawTiming(FrameTiming{
		WindowID:        windowProfiler.windowID,
		DirtyAt:         Some(drawEnd.Add(-4 * time.Millisecond)),
		Invalidations:   1,
		DrawStart:       drawEnd.Add(-2 * time.Millisecond),
		DrawEnd:         drawEnd,
		RefreshInterval: None[time.Duration](),
		SignalAt:        None[scheduler.Instant](),
	})
}

func profilerAssertEq[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func profilerAssertSliceEq[T comparable](t *testing.T, got, want []T) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestProfiler_ReleaseThreadTimingsRemovesTheThread(t *testing.T) {
	thread := NewThreadID(Some("profiler-release-test"))
	UpdateRunningTask(thread, scheduler.SpawnTime{Instant: scheduler.InstantNow()}, scheduler.CallerLocation(0))
	SaveTaskTiming(thread)
	listed := func() bool {
		return slices.ContainsFunc(GetAllTimings(TasksIncludedOnlyCompleted), func(timings ThreadTaskTimings) bool {
			return timings.ThreadID == thread && timings.ThreadName == Some("profiler-release-test")
		})
	}
	if !listed() {
		t.Fatal("thread timings should be registered")
	}
	ReleaseThreadTimings(thread)
	if listed() {
		t.Fatal("released thread timings should be removed")
	}
}
