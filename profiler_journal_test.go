package gpui

import (
	"math/rand/v2"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func TestJournal_PlatformSignalsAreJournaledWithoutSealingOrOccupyingTime(t *testing.T) {
	journal, guard := installTestForegroundJournal(32, 4)
	defer guard.End()
	collector := journal.Collector()
	start := scheduler.InstantNow()
	sealer := NewIntervalSealer(start)
	windowID := WindowID(1)
	for _, source := range []FrameRequestSource{FrameRequestSourceNativeCallback, FrameRequestSourceLocalSchedule} {
		for _, signalAt := range []Option[scheduler.Instant]{None[scheduler.Instant](), Some(start.Add(5 * time.Second))} {
			recordPlatformSignal(PlatformSignal{
				WindowID:  windowID,
				SignalAt:  signalAt,
				HandledAt: start.Add(5 * time.Millisecond),
				Source:    source,
			})
			entries := collector.CollectUnseen().Entries
			if len(entries) != 1 {
				t.Fatalf("expected a raw platform signal, got %#v", entries)
			}
			signal, ok := entries[0].(ForegroundJournalEntryPlatformSignal)
			if !ok {
				t.Fatalf("expected a raw platform signal, got %#v", entries)
			}
			if signal.Signal.WindowID != windowID || signal.Signal.SignalAt != signalAt ||
				signal.Signal.HandledAt != start.Add(5*time.Millisecond) || signal.Signal.Source != source {
				t.Fatalf("unexpected signal %#v", signal)
			}
			if got := sealer.PushEntries(entries); len(got) != 0 {
				t.Fatalf("expected no snapshots, got %#v", got)
			}
		}
	}
	endedAt := start.Add(10 * time.Millisecond)
	snapshots := sealer.PushEntries([]ForegroundJournalEntry{
		inputEntry(start, endedAt),
		ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: endedAt}},
	})
	if len(snapshots) != 1 {
		t.Fatalf("expected one snapshot, got %#v", snapshots)
	}
	snapshot := snapshots[0]
	assertEq(t, snapshot.IntervalStart, start)
	assertEq(t, snapshot.IntervalEnd(), endedAt)
	assertEq(t, snapshot.Occupancy(), 10*time.Millisecond)
	assertEq(t, len(snapshot.Events), 1)
}

func TestJournal_CompletedSkipsSealWithoutRetiringPendingDemand(t *testing.T) {
	start := scheduler.InstantNow()
	at := start.Add(10 * time.Millisecond)
	windowID := WindowID(1)
	for _, reason := range []FrameSkipReason{
		FrameSkipReasonNoRenderNeeded,
		FrameSkipReasonInactiveFrameRateLimit,
		FrameSkipReasonThermalFrameRateLimit,
	} {
		writer, collector := testJournal(newForegroundRunnableCounter())
		writer.beginTurn()
		writer.recordFramePending(windowID, start)
		writer.recordEvent(ForegroundEventInput{InputTiming{Kind: "test", Start: start, End: start.Add(2 * time.Millisecond)}})
		writer.recordFrameSkipped(FrameSkipped{WindowID: windowID, At: at, Reason: reason})

		sealer := NewIntervalSealer(start)
		snapshots := sealer.PushEntries(collector.CollectUnseen().Entries)
		if len(snapshots) != 1 {
			t.Fatalf("expected a completed skip, got %#v", snapshots)
		}
		snapshot := snapshots[0]
		skipped, ok := snapshot.Boundary.(IntervalBoundaryFrameSkipped)
		if !ok || skipped.WindowID != windowID || skipped.At != at || skipped.Reason != reason {
			t.Fatalf("unexpected boundary %#v", snapshot.Boundary)
		}
		assertEq(t, snapshot.IntervalEnd(), at)
		if _, ok := snapshot.Boundary.DirtyAt(); ok {
			t.Fatal("expected no dirty_at")
		}
		assertEq(t, snapshot.Occupancy(), 2*time.Millisecond)

		writer.recordEvent(ForegroundEventInput{InputTiming{Kind: "test", Start: at, End: at.Add(time.Millisecond)}})
		writer.endTurn(at.Add(time.Millisecond))
		for _, entry := range collector.CollectUnseen().Entries {
			if _, ok := entry.(ForegroundJournalEntryBoundary); ok {
				t.Fatal("pending demand must still block idle boundaries")
			}
		}
	}
}

func TestJournal_IdleRefreshesSealNothingAndKeepPollsFolded(t *testing.T) {
	start := scheduler.InstantNow()
	windowID := WindowID(1)
	writer, collector := testJournal(newForegroundRunnableCounter())
	for refresh := range 3 {
		at := start.Add(16 * time.Millisecond * time.Duration(refresh))
		writer.beginTurn()
		writer.foldSmallPoll(taskTiming(at, at.Add(50*time.Microsecond)))
		writer.recordPlatformSignal(PlatformSignal{
			WindowID:  windowID,
			SignalAt:  Some(at),
			HandledAt: at,
			Source:    FrameRequestSourceNativeCallback,
		})
		writer.recordFrameSkipped(FrameSkipped{WindowID: windowID, At: at, Reason: FrameSkipReasonNoRenderNeeded})
		writer.endTurn(at.Add(60 * time.Microsecond))
	}

	entries := collector.CollectUnseen().Entries
	for _, entry := range entries {
		if _, ok := entry.(ForegroundJournalEntryPlatformSignal); !ok {
			t.Fatalf("expected only platform signals, got %#v", entries)
		}
	}
	if got := NewIntervalSealer(start).PushEntries(entries); len(got) != 0 {
		t.Fatalf("expected no snapshots, got %#v", got)
	}
}

func TestJournal_ReentrantSkipDoesNotSealTheEnclosingDraw(t *testing.T) {
	journal, guard := installTestForegroundJournal(32, 4)
	defer guard.End()
	collector := journal.Collector()
	start := scheduler.InstantNow()
	windowID := WindowID(1)
	sealer := NewIntervalSealer(start)
	recordPlatformSignal(PlatformSignal{
		WindowID:  windowID,
		HandledAt: start.Add(5 * time.Millisecond),
		Source:    FrameRequestSourceLocalSchedule,
	})
	recordReentrantFrameSkipped(windowID, start.Add(6*time.Millisecond))
	entries := collector.CollectUnseen().Entries
	if len(entries) != 2 {
		t.Fatalf("unexpected entries %#v", entries)
	}
	if _, ok := entries[0].(ForegroundJournalEntryPlatformSignal); !ok {
		t.Fatalf("unexpected entries %#v", entries)
	}
	if skipped, ok := entries[1].(ForegroundJournalEntryReentrantFrameSkipped); !ok || skipped.WindowID != windowID || skipped.At != start.Add(6*time.Millisecond) {
		t.Fatalf("unexpected entries %#v", entries)
	}
	if got := sealer.PushEntries(entries); len(got) != 0 {
		t.Fatalf("expected no snapshots, got %#v", got)
	}

	frame := FrameTiming{
		WindowID:      windowID,
		DirtyAt:       Some(start),
		Invalidations: 1,
		DrawStart:     start,
		DrawEnd:       start.Add(20 * time.Millisecond),
	}
	recordDraw(frame)
	recordPresent(presentationTiming(windowID, frame.DrawEnd), Some(frame))
	snapshots := sealer.PushEntries(collector.CollectUnseen().Entries)
	if len(snapshots) != 1 {
		t.Fatalf("expected the enclosing presentation, got %#v", snapshots)
	}
	if _, ok := snapshots[0].Boundary.(IntervalBoundaryPresented); !ok {
		t.Fatalf("unexpected boundary %#v", snapshots[0].Boundary)
	}
	assertEq(t, snapshots[0].Occupancy(), 20*time.Millisecond)
}

type journalTestEmptyView struct{}

func (journalTestEmptyView) Render(_ *Window, _ *Context[journalTestEmptyView]) IntoElement {
	return Empty{}
}

func TestJournal_WakeRechecksNativeVisibility(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		if !profilerFeatureEnabled {
			cx.Update(observePower)
		}
		handle := cx.AddWindow(func(_ *Window, _ *Context[journalTestEmptyView]) *journalTestEmptyView {
			return &journalTestEmptyView{}
		})
		platform := cx.testWindow(handle.AnyWindowHandle)
		platform.OnVisibilityChange(func(WindowVisibility) {})
		platform.SimulateVisibilityChange(WindowVisibilityHidden)
		if err := handle.Update(cx, func(_ *journalTestEmptyView, window *Window, _ *Context[journalTestEmptyView]) {
			if !window.IsVisible() {
				t.Fatal("window should still be visible")
			}
		}); err != nil {
			t.Fatalf("window: %v", err)
		}
		cx.Update(func(cx *App) {
			cx.systemWakeObservers.retain(func(callback *handler) bool { return (*callback)(cx) })
		})
		cx.RunUntilParked()
		if err := handle.Update(cx, func(_ *journalTestEmptyView, window *Window, _ *Context[journalTestEmptyView]) {
			if window.IsVisible() {
				t.Fatal("window should be hidden after wake")
			}
		}); err != nil {
			t.Fatalf("window: %v", err)
		}
	})
}

func TestJournal_PowerInterruptionDropsThePollBeforeSmallPollFolding(t *testing.T) {
	journal, guard := installTestForegroundJournal(32, 4)
	defer guard.End()
	collector := journal.Collector()
	start := scheduler.InstantNow()
	beginForegroundTurn()
	recordPowerTransition(powerStateSuspended)
	recordPowerTransition(powerStateAwake)
	if !powerInterruptedSince(start) {
		t.Fatal("expected interruption")
	}
	recordTaskPoll(taskTiming(start, start.Add(10*time.Microsecond)))
	for _, entry := range collector.CollectUnseen().Entries {
		if !isBoundary[IntervalBoundaryPowerTransition](entry) {
			t.Fatalf("unexpected entry %#v", entry)
		}
	}
	withJournal(func(writer *foregroundJournalWriter) {
		if writer.smallPolls.IsSome() {
			t.Fatal("small polls should be empty")
		}
	})
	beginForegroundTurn()
	start = scheduler.InstantNow()
	if powerInterruptedSince(start) {
		t.Fatal("unexpected interruption")
	}
	recordTaskPoll(taskTiming(start, start.Add(TaskPollFloor)))
	if !anyEntry(collector.CollectUnseen().Entries, isEvent[ForegroundEventTaskPoll]) {
		t.Fatal("expected a task poll")
	}
}

func TestJournal_APollSpanningAnUnnotifiedSuspendIsDropped(t *testing.T) {
	journal, guard := installTestForegroundJournal(32, 4)
	defer guard.End()
	collector := journal.Collector()
	window := WindowID(1)
	recordFramePending(window, scheduler.InstantNow())
	beginForegroundTurn()
	start := scheduler.InstantNow()
	advanceTestSuspendedTime(2 * time.Second)
	recordTaskPoll(taskTiming(start, scheduler.InstantNow()))
	entries := collector.CollectUnseen().Entries
	if !anyEntry(entries, isBoundary[IntervalBoundaryPowerTransition]) {
		t.Fatal("expected a power transition")
	}
	if anyEntry(entries, isEvent[ForegroundEventTaskPoll]) {
		t.Fatal("unexpected task poll")
	}
	withJournal(func(writer *foregroundJournalWriter) {
		if writer.windows[window].dirtyAt.IsSome() {
			t.Fatal("dirty_at should be cleared")
		}
	})

	beginForegroundTurn()
	start = scheduler.InstantNow()
	recordTaskPoll(taskTiming(start, start.Add(TaskPollFloor)))
	entries = collector.CollectUnseen().Entries
	if !anyEntry(entries, isEvent[ForegroundEventTaskPoll]) {
		t.Fatal("expected a task poll")
	}
	if anyEntry(entries, isBoundary[IntervalBoundaryPowerTransition]) {
		t.Fatal("unexpected power transition")
	}
}

func TestJournal_SuspendClockIgnoresJitterAndNotifiedSuspends(t *testing.T) {
	journal, guard := installTestForegroundJournal(32, 4)
	defer guard.End()
	collector := journal.Collector()
	recordPowerTransition(powerStateSuspended)
	advanceTestSuspendedTime(60 * time.Second)
	recordPowerTransition(powerStateAwake)
	advanceTestSuspendedTime(minDetectedSuspend - time.Millisecond)
	beginForegroundTurn()
	start := scheduler.InstantNow()
	recordTaskPoll(taskTiming(start, start.Add(TaskPollFloor)))
	entries := collector.CollectUnseen().Entries
	transitions := 0
	for _, entry := range entries {
		if isBoundary[IntervalBoundaryPowerTransition](entry) {
			transitions++
		}
	}
	if transitions != 2 {
		t.Fatalf("only the two notifications: got %d", transitions)
	}
	if !anyEntry(entries, isEvent[ForegroundEventTaskPoll]) {
		t.Fatal("expected a task poll")
	}
}

func TestJournal_VisibilityAndPowerChangesClearPendingFrames(t *testing.T) {
	writer, _ := testJournal(newForegroundRunnableCounter())
	start := scheduler.InstantNow()
	first := WindowID(1)
	second := WindowID(2)
	writer.setVisibility(first, WindowVisibilityHidden, start)
	writer.recordFramePending(first, start)
	assertEq(t, writer.hasUnexpiredPendingFrame(start), false)
	writer.recordFramePending(second, start)
	assertEq(t, writer.hasUnexpiredPendingFrame(start), true)
	writer.setVisibility(second, WindowVisibilityHidden, start)
	assertEq(t, writer.hasUnexpiredPendingFrame(start), false)
	writer.setVisibility(first, WindowVisibilityVisible, start)
	writer.recordFramePending(first, start)
	assertEq(t, writer.hasUnexpiredPendingFrame(start), true)
	writer.powerTransition(powerStateSuspended, start)
	assertEq(t, writer.windows[first].dirtyAt.IsSome(), false)
	writer.recordFramePending(first, start)
	assertEq(t, writer.hasUnexpiredPendingFrame(start), false)
	writer.powerTransition(powerStateSuspended, start.Add(FrameDeadline))
	assertEq(t, writer.powerChangedAt, Some(start))
	writer.powerTransition(powerStateAwake, start)
	writer.recordFramePending(first, start)
	assertEq(t, writer.hasUnexpiredPendingFrame(start), true)
	writer.recordPresent(presentationTiming(first, start.Add(FrameDeadline)), None[FrameTiming]())
	assertEq(t, writer.windows[first].dirtyAt.IsSome(), false)
	writer.recordWindowClosed(first, start)
	if _, ok := writer.windows[first]; ok {
		t.Fatal("window should be removed")
	}
}

func TestJournal_DrawWaitsForItsPresentationBoundary(t *testing.T) {
	start := scheduler.InstantNow()
	input := InputTiming{Kind: "test", Start: start.Add(time.Millisecond), End: start.Add(2 * time.Millisecond), CausedInvalidation: true}
	frame := FrameTiming{
		WindowID:      WindowID(1),
		DirtyAt:       Some(input.Start),
		Invalidations: 1,
		DrawStart:     start.Add(3 * time.Millisecond),
		DrawEnd:       start.Add(4 * time.Millisecond),
	}
	presentation := PresentTiming{
		WindowID:     frame.WindowID,
		PresentStart: start.Add(5 * time.Millisecond),
		PresentEnd:   start.Add(6 * time.Millisecond),
	}
	sealer := NewIntervalSealer(start)

	snapshots := sealer.PushEntries([]ForegroundJournalEntry{
		ForegroundJournalEntryEvent{ForegroundEventInput{input}},
		ForegroundJournalEntryEvent{ForegroundEventDraw{frame}},
	})
	assertEq(t, len(snapshots), 0)

	snapshots = sealer.PushEntries([]ForegroundJournalEntry{
		ForegroundJournalEntryBoundary{IntervalBoundaryPresented{PresentedFrame{Frame: frame, Presentation: presentation}}},
	})
	if len(snapshots) != 1 {
		t.Fatalf("expected one presentation-sealed snapshot, got %#v", snapshots)
	}
	snapshot := snapshots[0]
	assertEq(t, snapshot.IntervalStart, input.Start)
	assertEq(t, snapshot.IntervalEnd(), presentation.PresentEnd)
	assertEq(t, len(snapshot.Events), 3)
	if last, ok := snapshot.Events[len(snapshot.Events)-1].(ForegroundEventPresent); !ok || last.PresentDuration() != time.Millisecond {
		t.Fatalf("unexpected last event %#v", snapshot.Events[len(snapshot.Events)-1])
	}
	presented, ok := snapshot.Boundary.(IntervalBoundaryPresented)
	if !ok {
		t.Fatalf("unexpected boundary %#v", snapshot.Boundary)
	}
	d, ok := presented.DirtyToPresentDuration()
	if !ok || d != 5*time.Millisecond {
		t.Fatalf("dirty_to_present = %v, %v", d, ok)
	}
}

func TestJournal_OutermostTurnSealsNoFrameWorkWhenIdle(t *testing.T) {
	start := scheduler.InstantNow()
	journal, collector := testJournal(newForegroundRunnableCounter())
	events := []ForegroundEvent{
		ForegroundEventInput{InputTiming{Kind: "test", Start: start, End: start.Add(time.Millisecond)}},
		ForegroundEventAction{ActionTiming{Name: "test.action", Start: start.Add(2 * time.Millisecond), End: start.Add(3 * time.Millisecond)}},
		ForegroundEventTaskPoll{taskTiming(start.Add(4*time.Millisecond), start.Add(5*time.Millisecond))},
	}
	for _, event := range events {
		journal.beginTurn()
		journal.recordEvent(event)
		journal.endTurn(event.EndTime())
	}
	entries := collector.CollectUnseen().Entries
	for _, event := range events {
		if !hasIdleAt(entries, event.EndTime()) {
			t.Fatalf("expected an idle boundary at %v", event.EndTime())
		}
	}
}

func TestJournal_NestedTurnsOnlySealAfterTheOutermostTurn(t *testing.T) {
	start := scheduler.InstantNow()
	journal, collector := testJournal(newForegroundRunnableCounter())
	action := ForegroundEventAction{ActionTiming{Name: "test.action", Start: start, End: start.Add(time.Millisecond)}}
	input := ForegroundEventInput{InputTiming{Kind: "test", Start: start, End: start.Add(2 * time.Millisecond)}}

	journal.beginTurn()
	journal.beginTurn()
	journal.recordEvent(action)
	journal.endTurn(action.EndTime())
	if hasBoundaryAt(collector.CollectUnseen().Entries, action.EndTime()) {
		t.Fatal("nested turn must not seal")
	}

	journal.recordEvent(input)
	journal.endTurn(input.EndTime())
	if !hasIdleAt(collector.CollectUnseen().Entries, input.EndTime()) {
		t.Fatal("outermost turn must seal")
	}
}

func TestJournal_AnImmediatelyReadyRunnablePreventsAnIdleBoundaryBetweenPolls(t *testing.T) {
	start := scheduler.InstantNow()
	counter := newForegroundRunnableCounter()
	journal, collector := testJournal(counter)
	counter.queued()
	counter.queued()

	first := ForegroundEventTaskPoll{taskTiming(start, start.Add(time.Millisecond))}
	journal.beginTurn()
	journal.recordEvent(first)
	counter.finished()
	journal.endTurn(first.EndTime())
	if hasBoundaryAt(collector.CollectUnseen().Entries, first.EndTime()) {
		t.Fatal("a queued runnable must prevent idle")
	}

	second := ForegroundEventTaskPoll{taskTiming(start.Add(2*time.Millisecond), start.Add(3*time.Millisecond))}
	journal.beginTurn()
	journal.recordEvent(second)
	counter.finished()
	journal.endTurn(second.EndTime())
	if !hasIdleAt(collector.CollectUnseen().Entries, second.EndTime()) {
		t.Fatal("expected idle after the last runnable")
	}
}

func TestJournal_APendingFramePreventsIdleUntilPresentation(t *testing.T) {
	start := scheduler.InstantNow()
	windowID := WindowID(0xD17A)
	journal, collector := testJournal(newForegroundRunnableCounter())
	journal.recordFramePending(windowID, start)
	journal.beginTurn()
	journal.recordEvent(ForegroundEventInput{InputTiming{Kind: "test", Start: start, End: start.Add(time.Millisecond), CausedInvalidation: true}})
	inputEnd := start.Add(time.Millisecond)
	journal.endTurn(inputEnd)
	if hasBoundaryAt(collector.CollectUnseen().Entries, inputEnd) {
		t.Fatal("pending frame must block idle")
	}

	frame := frameTiming(windowID, start, start.Add(2*time.Millisecond))
	presentation := presentationTiming(windowID, start.Add(3*time.Millisecond))
	journal.recordPresent(presentation, Some(frame))
	if !anyEntry(collector.CollectUnseen().Entries, func(entry ForegroundJournalEntry) bool {
		b, ok := entry.(ForegroundJournalEntryBoundary)
		if !ok {
			return false
		}
		p, ok := b.Boundary.(IntervalBoundaryPresented)
		return ok && p.Frame.WindowID == windowID
	}) {
		t.Fatal("expected a presented boundary")
	}
}

func TestJournal_APresentWithoutADrawClearsTheWindowPendingState(t *testing.T) {
	start := scheduler.InstantNow()
	presentedAt := start.Add(time.Millisecond)
	windowID := WindowID(0xD17B)
	journal, collector := testJournal(newForegroundRunnableCounter())
	journal.recordFramePending(windowID, start)
	journal.beginTurn()
	journal.recordPresent(presentationTiming(windowID, presentedAt), None[FrameTiming]())
	journal.endTurn(presentedAt)

	entries := collector.CollectUnseen().Entries
	if !anyEntry(entries, func(entry ForegroundJournalEntry) bool {
		e, ok := entry.(ForegroundJournalEntryEvent)
		if !ok {
			return false
		}
		p, ok := e.Event.(ForegroundEventPresent)
		return ok && p.WindowID == windowID
	}) {
		t.Fatal("expected a present event")
	}
	if !hasBoundaryAt(entries, presentedAt) {
		t.Fatal("expected a boundary at the presentation")
	}
}

func TestJournal_APresentationAfterTheDeadlineSealsTheWholeInterval(t *testing.T) {
	start := scheduler.InstantNow()
	windowID := WindowID(0xDEA1)
	presentedAt := start.Add(FrameDeadline + 250*time.Millisecond)
	sealer := NewIntervalSealer(start)
	snapshots := sealer.PushEntries([]ForegroundJournalEntry{
		pendingFrame(windowID, start),
		inputEntry(start, start.Add(20*time.Millisecond)),
		ForegroundJournalEntryBoundary{presentedBoundary(windowID, start, presentedAt)},
	})
	if len(snapshots) != 1 {
		t.Fatalf("expected one presented snapshot, got %#v", snapshots)
	}
	presented, ok := snapshots[0].Boundary.(IntervalBoundaryPresented)
	d, dok := presented.DirtyToPresentDuration()
	if !ok || presented.Frame.WindowID != windowID || !dok || d != presentedAt.Sub(start) {
		t.Fatalf("unexpected boundary %#v", snapshots[0].Boundary)
	}
	assertEq(t, snapshots[0].IntervalEnd(), presentedAt)
	assertEq(t, len(snapshots[0].Events), 2)
}

func TestJournal_AnExpiredFrameNoLongerBlocksIdleBoundaries(t *testing.T) {
	start := scheduler.InstantNow()
	windowID := WindowID(0xDEA2)
	journal, collector := testJournal(newForegroundRunnableCounter())
	journal.recordFramePending(windowID, start)

	blockedEnd := start.Add(20 * time.Millisecond)
	journal.beginTurn()
	journal.recordEvent(ForegroundEventInput{InputTiming{Kind: "test", Start: start, End: blockedEnd, CausedInvalidation: true}})
	journal.endTurn(blockedEnd)
	if hasBoundaryAt(collector.CollectUnseen().Entries, blockedEnd) {
		t.Fatal("unexpired frame must block idle")
	}

	unblockedEnd := start.Add(FrameDeadline + time.Millisecond)
	journal.beginTurn()
	journal.recordEvent(ForegroundEventInput{InputTiming{Kind: "test", Start: start.Add(FrameDeadline), End: unblockedEnd}})
	journal.endTurn(unblockedEnd)
	if !hasBoundaryAt(collector.CollectUnseen().Entries, unblockedEnd) {
		t.Fatal("expired frame must not block idle")
	}
}

func TestJournal_ClosingAWindowUnblocksIdleBoundaries(t *testing.T) {
	start := scheduler.InstantNow()
	windowID := WindowID(0xDEA7)
	journal, collector := testJournal(newForegroundRunnableCounter())
	journal.recordFramePending(windowID, start)
	journal.recordWindowClosed(windowID, start.Add(time.Millisecond))

	eventEnd := start.Add(2 * time.Millisecond)
	journal.beginTurn()
	journal.recordEvent(ForegroundEventInput{InputTiming{Kind: "test", Start: start.Add(time.Millisecond), End: eventEnd}})
	journal.endTurn(eventEnd)
	if !hasBoundaryAt(collector.CollectUnseen().Entries, eventEnd) {
		t.Fatal("expected an idle boundary")
	}
}

func TestJournal_PresentingOneWindowDoesNotClearAnotherPendingWindow(t *testing.T) {
	start := scheduler.InstantNow()
	firstWindow := WindowID(0xDEA5)
	secondWindow := WindowID(0xDEA6)
	journal, collector := testJournal(newForegroundRunnableCounter())
	journal.recordFramePending(firstWindow, start)
	journal.recordFramePending(secondWindow, start.Add(100*time.Millisecond))
	journal.recordPresent(
		presentationTiming(firstWindow, start.Add(500*time.Millisecond)),
		Some(frameTiming(firstWindow, start, start.Add(500*time.Millisecond))),
	)

	eventEnd := start.Add(620 * time.Millisecond)
	journal.beginTurn()
	journal.recordEvent(ForegroundEventInput{InputTiming{Kind: "test", Start: start.Add(600 * time.Millisecond), End: eventEnd}})
	journal.endTurn(eventEnd)

	entries := collector.CollectUnseen().Entries
	if !anyEntry(entries, func(entry ForegroundJournalEntry) bool {
		b, ok := entry.(ForegroundJournalEntryBoundary)
		if !ok {
			return false
		}
		p, ok := b.Boundary.(IntervalBoundaryPresented)
		return ok && p.Frame.WindowID == firstWindow
	}) {
		t.Fatal("expected the first window's presentation")
	}
	if hasBoundaryAt(entries, eventEnd) {
		t.Fatal("second window must still block idle")
	}
}

func TestJournal_SmallPollsAreFlushedImmediatelyBeforeARetainedEvent(t *testing.T) {
	start := scheduler.InstantNow()
	first := taskTiming(start.Add(time.Millisecond), start.Add(time.Millisecond+20*time.Microsecond))
	second := taskTiming(start.Add(3*time.Millisecond), start.Add(3*time.Millisecond+30*time.Microsecond))
	input := InputTiming{Kind: "test", Start: start.Add(5 * time.Millisecond), End: start.Add(6 * time.Millisecond)}
	journal, collector := testJournal(newForegroundRunnableCounter())

	journal.foldSmallPoll(first)
	journal.foldSmallPoll(second)
	journal.recordEvent(ForegroundEventInput{input})

	drained := collector.CollectUnseen()
	inputIndex := -1
	for i, entry := range drained.Entries {
		if e, ok := entry.(ForegroundJournalEntryEvent); ok {
			if in, ok := e.Event.(ForegroundEventInput); ok && in.Start == input.Start {
				inputIndex = i
				break
			}
		}
	}
	if inputIndex < 0 {
		t.Fatal("input event should be retained")
	}
	if inputIndex < 1 {
		t.Fatal("small-poll summary should immediately precede the input event")
	}
	e, ok := drained.Entries[inputIndex-1].(ForegroundJournalEntryEvent)
	if !ok {
		t.Fatal("small-poll summary should immediately precede the input event")
	}
	flush, ok := e.Event.(ForegroundEventSmallPolls)
	if !ok {
		t.Fatal("small-poll summary should immediately precede the input event")
	}
	assertEq(t, flush.Summary.Count, uint64(2))
	assertEq(t, flush.Summary.Total, 50*time.Microsecond)
	assertEq(t, flush.Since, first.Start)
	assertEq(t, flush.Until, second.End.Instant)
}

func TestJournal_SmallPollsAreFlushedBeforeFrameStateChanges(t *testing.T) {
	start := scheduler.InstantNow()
	windowID := WindowID(0xDEA9)
	poll := taskTiming(start, start.Add(50*time.Microsecond))
	journal, collector := testJournal(newForegroundRunnableCounter())
	journal.foldSmallPoll(poll)
	journal.recordFramePending(windowID, poll.End.Instant)

	entries := collector.CollectUnseen().Entries
	pendingIndex := -1
	for i, entry := range entries {
		if fs, ok := entry.(ForegroundJournalEntryFrameState); ok {
			if p, ok := fs.State.(FrameStateChangePending); ok && p.WindowID == windowID {
				pendingIndex = i
				break
			}
		}
	}
	if pendingIndex < 0 {
		t.Fatal("pending frame should be retained")
	}
	ok := false
	if pendingIndex >= 1 {
		if e, isEvent := entries[pendingIndex-1].(ForegroundJournalEntryEvent); isEvent {
			if flush, isFlush := e.Event.(ForegroundEventSmallPolls); isFlush {
				ok = flush.Since == poll.Start && flush.Until == poll.End.Instant
			}
		}
	}
	if !ok {
		t.Fatalf("expected a small-poll flush before the pending frame, got %#v", entries)
	}
}

func TestJournal_SparseSmallPollsAreDiscardedWhenTheForegroundReturnsToIdle(t *testing.T) {
	start := scheduler.InstantNow()
	journal, collector := testJournal(newForegroundRunnableCounter())
	tinyWake := func(endedAt scheduler.Instant) {
		journal.beginTurn()
		journal.foldSmallPoll(taskTiming(instantSub(endedAt, 50*time.Microsecond), endedAt))
		journal.endTurn(endedAt)
	}
	for second := 1; second <= 160; second++ {
		tinyWake(start.Add(time.Duration(second) * time.Second))
	}
	if quiet := collector.CollectUnseen().Entries; len(quiet) != 0 {
		t.Fatalf("expected no entries, got %#v", quiet)
	}

	retained := ForegroundEventTaskPoll{taskTiming(start.Add(161*time.Second), start.Add(161*time.Second+time.Millisecond))}
	journal.beginTurn()
	journal.recordEvent(retained)
	journal.endTurn(retained.EndTime())
	entries := collector.CollectUnseen().Entries
	if !hasBoundaryAt(entries, retained.EndTime()) {
		t.Fatal("expected a boundary")
	}
	if anyEntry(entries, isEvent[ForegroundEventSmallPolls]) {
		t.Fatal("unexpected small polls")
	}
}

func TestJournal_NoteLostIsReportedOnTheNextSnapshotOnly(t *testing.T) {
	start := scheduler.InstantNow()
	sealer := NewIntervalSealer(start)
	sealer.NoteLost(7)
	snapshots := sealer.PushEntries([]ForegroundJournalEntry{
		inputEntry(start, start.Add(time.Millisecond)),
		ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: start.Add(time.Millisecond)}},
		inputEntry(start.Add(2*time.Millisecond), start.Add(3*time.Millisecond)),
		ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: start.Add(3 * time.Millisecond)}},
	})
	assertEq(t, len(snapshots), 2)
	assertEq(t, snapshots[0].DroppedEvents, uint64(7))
	assertEq(t, snapshots[1].DroppedEvents, uint64(0))
}

func TestJournal_LossesAloneSealASnapshotAtTheNextBoundary(t *testing.T) {
	start := scheduler.InstantNow()
	sealer := NewIntervalSealer(start)
	sealer.NoteLost(3)
	snapshots := sealer.PushEntries([]ForegroundJournalEntry{
		ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: start.Add(time.Millisecond)}},
	})
	assertEq(t, len(snapshots), 1)
	assertEq(t, snapshots[0].DroppedEvents, uint64(3))
	assertEq(t, len(snapshots[0].Events), 0)
}

func TestJournal_IntervalEventCapCountsOverflowingEvents(t *testing.T) {
	start := scheduler.InstantNow()
	sealer := NewIntervalSealer(start)
	const overflow = 5
	entries := make([]ForegroundJournalEntry, 0, maxIntervalEvents+overflow+1)
	for i := range uint64(maxIntervalEvents + overflow) {
		entries = append(entries, inputEntry(micros(start, i), micros(start, i+1)))
	}
	entries = append(entries, ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: micros(start, maxIntervalEvents+overflow+1)}})

	snapshots := sealer.PushEntries(entries)
	assertEq(t, len(snapshots), 1)
	assertEq(t, len(snapshots[0].Events), maxIntervalEvents)
	assertEq(t, snapshots[0].DroppedEvents, uint64(overflow))
}

func TestJournal_CollectUnseenReturnsEachEntryOnceInOrder(t *testing.T) {
	start := scheduler.InstantNow()
	journal, publisher := newForegroundJournal(32, 4)
	collector := journal.Collector()
	timestamps := make([]scheduler.Instant, 0, 19)
	entries := make([]ForegroundJournalEntry, 0, 19)
	for i := range uint64(19) {
		at := micros(start, i+1)
		timestamps = append(timestamps, at)
		entries = append(entries, inputEntry(at, at))
	}
	publisher.publish(entries...)

	ours := func(entries []ForegroundJournalEntry) []scheduler.Instant {
		var out []scheduler.Instant
		for _, entry := range entries {
			e, ok := entry.(ForegroundJournalEntryEvent)
			if !ok {
				continue
			}
			in, ok := e.Event.(ForegroundEventInput)
			if !ok {
				continue
			}
			for _, ts := range timestamps {
				if ts == in.End {
					out = append(out, in.End)
					break
				}
			}
		}
		return out
	}

	assertEq(t, ours(collector.CollectUnseen().Entries), timestamps)
	if got := ours(collector.CollectUnseen().Entries); len(got) != 0 {
		t.Fatalf("expected nothing on the second drain, got %v", got)
	}
}

func TestJournal_ConcurrentCollectionPreservesTheCompleteLogicalSequence(t *testing.T) {
	const entryCount uint64 = 20_000

	origin := scheduler.InstantNow()
	journal, publisher := newForegroundJournal(8, 4)
	collector := journal.Collector()
	var done atomic.Bool
	type result struct {
		observed []modelDrainEntry
		panicked any
	}
	results := make(chan result, 1)
	go func() {
		var observed []modelDrainEntry
		defer func() {
			if r := recover(); r != nil {
				results <- result{panicked: r}
			}
		}()
		for {
			observed = append(observed, normalizeRingDrain(origin, collector.CollectUnseen().Entries)...)
			if done.Load() {
				observed = append(observed, normalizeRingDrain(origin, collector.CollectUnseen().Entries)...)
				results <- result{observed: observed}
				return
			}
			runtime.Gosched()
		}
	}()

	for sequence := range entryCount {
		publisher.publish(ringEntry(origin, sequence))
	}
	done.Store(true)

	r := <-results
	if r.panicked != nil {
		t.Fatalf("collector goroutine should not panic: %v", r.panicked)
	}
	observed := r.observed
	publisher.flushPending()
	observed = append(observed, normalizeRingDrain(origin, collector.CollectUnseen().Entries)...)

	var expected uint64
	for _, entry := range observed {
		if entry.discontinuity {
			expected += entry.value
		} else {
			assertEq(t, entry.value, expected)
			expected++
		}
	}
	assertEq(t, expected, entryCount)
	assertEq(t, publisher.ring.finalized.Load(), entryCount)
}

type modelDrainEntry struct {
	discontinuity bool
	value         uint64
}

type modelRingSlot struct {
	sequence, value uint64
}

type modelRing struct {
	capacity            int
	pendingCapacity     int
	nextSequence        uint64
	finalized           uint64
	offered             uint64
	slots               []Option[modelRingSlot]
	pinned              []bool
	pending             []modelRingSlot
	droppedAfterPending uint64
}

func newModelRing(capacity, pendingCapacity int) *modelRing {
	return &modelRing{
		capacity:        capacity,
		pendingCapacity: pendingCapacity,
		slots:           make([]Option[modelRingSlot], capacity),
		pinned:          make([]bool, capacity),
	}
}

func (m *modelRing) publish() {
	m.flushPending()
	sequence := m.nextSequence
	m.nextSequence++
	m.offered = m.nextSequence

	if len(m.pending) > 0 || m.droppedAfterPending > 0 {
		if m.droppedAfterPending == 0 && len(m.pending) < m.pendingCapacity {
			m.pending = append(m.pending, modelRingSlot{sequence, sequence})
		} else {
			m.droppedAfterPending++
		}
	} else {
		index := int(sequence % uint64(m.capacity))
		if m.pinned[index] {
			if m.pendingCapacity > 0 {
				m.pending = append(m.pending, modelRingSlot{sequence, sequence})
			} else {
				m.droppedAfterPending = 1
			}
		} else {
			m.slots[index] = Some(modelRingSlot{sequence, sequence})
			m.finalized = sequence + 1
		}
	}
	m.flushPending()
}

func (m *modelRing) flushPending() {
	for len(m.pending) > 0 {
		front := m.pending[0]
		index := int(front.sequence % uint64(m.capacity))
		if m.pinned[index] {
			return
		}
		m.slots[index] = Some(front)
		m.pending = m.pending[1:]
		m.finalized = front.sequence + 1
	}
	if m.droppedAfterPending > 0 {
		m.finalized = m.nextSequence
		m.droppedAfterPending = 0
	}
}

func (m *modelRing) collect(cursor *uint64) ([]modelDrainEntry, uint64) {
	end := m.finalized
	if *cursor >= end {
		return nil, 0
	}
	retainedStart := saturatingSubU64(end, uint64(m.capacity))
	lost := saturatingSubU64(retainedStart, *cursor)
	*cursor = max(*cursor, retainedStart)
	var entries []modelDrainEntry
	if lost > 0 {
		entries = append(entries, modelDrainEntry{discontinuity: true, value: lost})
	}
	for *cursor < end {
		index := int(*cursor % uint64(m.capacity))
		if slot, ok := m.slots[index].Get(); ok && slot.sequence == *cursor {
			entries = append(entries, modelDrainEntry{value: slot.value})
		} else {
			lost++
			if n := len(entries); n > 0 && entries[n-1].discontinuity {
				entries[n-1].value++
			} else {
				entries = append(entries, modelDrainEntry{discontinuity: true, value: 1})
			}
		}
		*cursor++
	}
	return entries, lost
}

func ringEntry(origin scheduler.Instant, sequence uint64) ForegroundJournalEntry {
	at := micros(origin, sequence)
	return inputEntry(at, at)
}

func normalizeRingDrain(origin scheduler.Instant, entries []ForegroundJournalEntry) []modelDrainEntry {
	out := make([]modelDrainEntry, 0, len(entries))
	for _, entry := range entries {
		switch entry := entry.(type) {
		case ForegroundJournalEntryEvent:
			in, ok := entry.Event.(ForegroundEventInput)
			if !ok {
				panic("ring property observed an unexpected entry")
			}
			out = append(out, modelDrainEntry{value: asMicros(in.Start.Sub(origin))})
		case ForegroundJournalEntryDiscontinuity:
			out = append(out, modelDrainEntry{discontinuity: true, value: entry.Lost})
		default:
			panic("ring property observed an unexpected entry")
		}
	}
	return out
}

type normalizedEvent struct {
	kind       uint8
	start, end uint64
}

type normalizedSmallPolls struct {
	count, totalMicros, since, until uint64
}

type normalizedSnapshot struct {
	intervalStart        uint64
	boundaryKind         uint8
	intervalEnd          uint64
	events               []normalizedEvent
	smallPolls           []normalizedSmallPolls
	droppedEvents        uint64
	journalDiscontinuous bool
}

type normalizedSealerTail struct {
	intervalStart        uint64
	events               []normalizedEvent
	smallPolls           []normalizedSmallPolls
	droppedEvents        uint64
	journalDiscontinuous bool
}

func completionOrderEntries(origin scheduler.Instant, specifications [][3]uint8) []ForegroundJournalEntry {
	completedAt := uint64(1)
	out := make([]ForegroundJournalEntry, 0, len(specifications))
	for _, spec := range specifications {
		advance, span, kind := spec[0], spec[1], spec[2]
		completedAt += uint64(advance) + 1
		startedAt := saturatingSubU64(completedAt, uint64(span))
		start := micros(origin, startedAt)
		end := micros(origin, completedAt)
		var entry ForegroundJournalEntry
		switch kind % 7 {
		case 0:
			entry = inputEntry(start, end)
		case 1:
			entry = ForegroundJournalEntryEvent{ForegroundEventAction{ActionTiming{Name: "test.action", Start: start, End: end}}}
		case 2:
			entry = ForegroundJournalEntryEvent{ForegroundEventTaskPoll{taskTiming(start, end)}}
		case 3:
			entry = ForegroundJournalEntryEvent{ForegroundEventSmallPolls{SmallPollFlush{
				Summary: PollSummary{Count: uint64(span%5) + 1, Total: time.Duration(span) * time.Microsecond},
				Since:   start,
				Until:   end,
			}}}
		case 4:
			entry = pendingFrame(WindowID(completedAt), start)
		case 5:
			entry = ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: end}}
		default:
			entry = ForegroundJournalEntryBoundary{presentedBoundary(WindowID(completedAt), start, end)}
		}
		out = append(out, entry)
	}
	return out
}

func normalizeEvent(origin scheduler.Instant, event ForegroundEvent) normalizedEvent {
	var kind uint8
	switch event.(type) {
	case ForegroundEventTaskPoll:
		kind = 0
	case ForegroundEventAction:
		kind = 1
	case ForegroundEventInput:
		kind = 2
	case ForegroundEventDraw:
		kind = 3
	case ForegroundEventPresent:
		kind = 4
	case ForegroundEventSmallPolls:
		kind = 5
	case ForegroundEventView:
		kind = 6
	}
	return normalizedEvent{
		kind:  kind,
		start: asMicros(event.StartTime().Sub(origin)),
		end:   asMicros(event.EndTime().Sub(origin)),
	}
}

func normalizeSmallPolls(origin scheduler.Instant, flush SmallPollFlush) normalizedSmallPolls {
	return normalizedSmallPolls{
		count:       flush.Summary.Count,
		totalMicros: asMicros(flush.Summary.Total),
		since:       asMicros(flush.Since.Sub(origin)),
		until:       asMicros(flush.Until.Sub(origin)),
	}
}

func boundaryKind(boundary IntervalBoundary) uint8 {
	switch boundary.(type) {
	case IntervalBoundaryIdle:
		return 0
	case IntervalBoundaryPresented:
		return 1
	case IntervalBoundaryPowerTransition:
		return 2
	default:
		return 3
	}
}

func normalizeEvents(origin scheduler.Instant, events []ForegroundEvent) []normalizedEvent {
	out := []normalizedEvent{}
	for _, event := range events {
		out = append(out, normalizeEvent(origin, event))
	}
	return out
}

func normalizeFlushes(origin scheduler.Instant, flushes []SmallPollFlush) []normalizedSmallPolls {
	out := []normalizedSmallPolls{}
	for _, flush := range flushes {
		out = append(out, normalizeSmallPolls(origin, flush))
	}
	return out
}

func normalizeSnapshot(origin scheduler.Instant, snapshot *FrameSnapshot) normalizedSnapshot {
	return normalizedSnapshot{
		intervalStart:        asMicros(snapshot.IntervalStart.Sub(origin)),
		boundaryKind:         boundaryKind(snapshot.Boundary),
		intervalEnd:          asMicros(snapshot.IntervalEnd().Sub(origin)),
		events:               normalizeEvents(origin, snapshot.Events),
		smallPolls:           normalizeFlushes(origin, snapshot.SmallPolls),
		droppedEvents:        snapshot.DroppedEvents,
		journalDiscontinuous: snapshot.JournalDiscontinuous,
	}
}

func normalizeTail(origin scheduler.Instant, sealer *IntervalSealer) normalizedSealerTail {
	return normalizedSealerTail{
		intervalStart:        asMicros(sealer.intervalStart.Sub(origin)),
		events:               normalizeEvents(origin, sealer.events),
		smallPolls:           normalizeFlushes(origin, sealer.smallPolls),
		droppedEvents:        sealer.droppedEvents,
		journalDiscontinuous: sealer.journalDiscontinuous,
	}
}

func referenceSeal(origin scheduler.Instant, entries []ForegroundJournalEntry) ([]normalizedSnapshot, normalizedSealerTail) {
	var intervalStart uint64
	events := []normalizedEvent{}
	smallPolls := []normalizedSmallPolls{}
	snapshots := []normalizedSnapshot{}
	var droppedEvents uint64
	journalDiscontinuous := false
	empty := func() bool {
		return len(events) == 0 && len(smallPolls) == 0 && droppedEvents == 0 && !journalDiscontinuous
	}

	for _, entry := range entries {
		switch entry := entry.(type) {
		case ForegroundJournalEntryEvent:
			eventStart := asMicros(entry.Event.StartTime().Sub(origin))
			if empty() {
				intervalStart = max(intervalStart, eventStart)
			}
			if flush, ok := entry.Event.(ForegroundEventSmallPolls); ok {
				smallPolls = append(smallPolls, normalizeSmallPolls(origin, flush.SmallPollFlush))
			} else {
				events = append(events, normalizeEvent(origin, entry.Event))
			}
		case ForegroundJournalEntryBoundary:
			if presented, ok := entry.Boundary.(IntervalBoundaryPresented); ok {
				present := ForegroundEventPresent{presented.Presentation}
				if empty() {
					intervalStart = max(intervalStart, asMicros(present.StartTime().Sub(origin)))
				}
				events = append(events, normalizeEvent(origin, present))
			}
			intervalEnd := asMicros(entry.Boundary.EndTime().Sub(origin))
			if empty() {
				intervalStart = max(intervalStart, intervalEnd)
			} else {
				snapshots = append(snapshots, normalizedSnapshot{
					intervalStart:        intervalStart,
					boundaryKind:         boundaryKind(entry.Boundary),
					intervalEnd:          intervalEnd,
					events:               events,
					smallPolls:           smallPolls,
					droppedEvents:        droppedEvents,
					journalDiscontinuous: journalDiscontinuous,
				})
				events = []normalizedEvent{}
				smallPolls = []normalizedSmallPolls{}
				droppedEvents = 0
				journalDiscontinuous = false
				intervalStart = intervalEnd
			}
		case ForegroundJournalEntryDiscontinuity:
			droppedEvents += entry.Lost
			journalDiscontinuous = true
		}
	}
	return snapshots, normalizedSealerTail{
		intervalStart:        intervalStart,
		events:               events,
		smallPolls:           smallPolls,
		droppedEvents:        droppedEvents,
		journalDiscontinuous: journalDiscontinuous,
	}
}

func referenceOccupancyMicros(eventSpans [][2]uint8, smallPollSpans [][3]uint8, windowStart, windowEnd uint8) uint64 {
	type span struct{ start, end uint8 }
	clamped := make([]span, 0, len(eventSpans))
	for _, s := range eventSpans {
		start := max(min(s[0], s[1]), windowStart)
		end := max(min(max(s[0], s[1]), windowEnd), start)
		clamped = append(clamped, span{start, end})
	}
	for i := 1; i < len(clamped); i++ {
		for j := i; j > 0 && (clamped[j].start < clamped[j-1].start || (clamped[j].start == clamped[j-1].start && clamped[j].end < clamped[j-1].end)); j-- {
			clamped[j], clamped[j-1] = clamped[j-1], clamped[j]
		}
	}
	var occupied uint64
	var mergedUntil Option[uint8]
	for _, s := range clamped {
		start := s.start
		if until, ok := mergedUntil.Get(); ok {
			start = max(start, until)
		}
		if s.end > start {
			occupied += uint64(s.end - start)
		}
		if until, ok := mergedUntil.Get(); ok {
			mergedUntil = Some(max(until, s.end))
		} else {
			mergedUntil = Some(s.end)
		}
	}
	var polls uint64
	for _, s := range smallPollSpans {
		since := min(s[0], s[1])
		until := max(s[0], s[1])
		factor := s[2]
		if since == until {
			if since >= windowStart && since <= windowEnd {
				polls += uint64(factor)
			}
		} else {
			overlapStart := max(since, windowStart)
			overlapEnd := max(min(until, windowEnd), overlapStart)
			polls += uint64(overlapEnd-overlapStart) * uint64(factor)
		}
	}
	return occupied + polls
}

type writerEntryKind uint8

const (
	writerEntryInput writerEntryKind = iota
	writerEntrySmallPolls
	writerEntryPending
	writerEntryClosed
	writerEntryPresent
	writerEntryPresented
	writerEntryIdle
)

type writerEntry struct {
	kind                             writerEntryKind
	window, at                       uint64
	count, totalMicros, since, until uint64
}

type modelSmallPolls struct {
	count, totalMicros, since, until uint64
}

type modelWriter struct {
	turnDepth             int
	runnables             int
	pendingFrames         map[uint64]uint64
	retainedSinceBoundary bool
	smallPolls            Option[modelSmallPolls]
	entries               []writerEntry
}

func (m *modelWriter) recordEntry(entry writerEntry) {
	if sp, ok := m.smallPolls.Take().Get(); ok {
		m.entries = append(m.entries, writerEntry{kind: writerEntrySmallPolls, count: sp.count, totalMicros: sp.totalMicros, since: sp.since, until: sp.until})
	}
	boundary := entry.kind == writerEntryPresented || entry.kind == writerEntryIdle
	m.entries = append(m.entries, entry)
	if boundary {
		m.retainedSinceBoundary = false
	}
}

func (m *modelWriter) recordInput(at uint64) {
	m.retainedSinceBoundary = true
	m.recordEntry(writerEntry{kind: writerEntryInput, at: at})
}

func (m *modelWriter) foldSmallPoll(since, until uint64) {
	sp := m.smallPolls.AsMut()
	if sp == nil {
		m.smallPolls = Some(modelSmallPolls{since: since, until: until})
		sp = m.smallPolls.AsMut()
	}
	sp.count++
	sp.totalMicros += until - since
	sp.since = min(sp.since, since)
	sp.until = max(sp.until, until)
}

func (m *modelWriter) recordPending(window, at uint64) {
	previous, ok := m.pendingFrames[window]
	if !ok || saturatingSubU64(at, previous) >= 1_000_000 {
		m.pendingFrames[window] = at
		m.recordEntry(writerEntry{kind: writerEntryPending, window: window, at: at})
	}
}

func (m *modelWriter) recordClosed(window, at uint64) {
	delete(m.pendingFrames, window)
	m.recordEntry(writerEntry{kind: writerEntryClosed, window: window, at: at})
}

func (m *modelWriter) recordPresent(window, at uint64, hasFrame bool) {
	delete(m.pendingFrames, window)
	if hasFrame {
		m.recordEntry(writerEntry{kind: writerEntryPresented, window: window, at: at})
	} else {
		m.retainedSinceBoundary = true
		m.recordEntry(writerEntry{kind: writerEntryPresent, window: window, at: at})
	}
}

func (m *modelWriter) endTurn(at uint64) {
	if m.turnDepth == 0 {
		return
	}
	m.turnDepth--
	if m.turnDepth > 0 || m.runnables > 0 {
		return
	}
	for window, dirtyAt := range m.pendingFrames {
		if saturatingSubU64(at, dirtyAt) >= 1_000_000 {
			delete(m.pendingFrames, window)
		}
	}
	if len(m.pendingFrames) > 0 {
		return
	}
	if !m.retainedSinceBoundary {
		m.smallPolls = None[modelSmallPolls]()
		return
	}
	m.recordEntry(writerEntry{kind: writerEntryIdle, at: at})
}

func normalizeWriterEntries(origin scheduler.Instant, entries []ForegroundJournalEntry) []writerEntry {
	at := func(instant scheduler.Instant) uint64 { return asMicros(instant.Sub(origin)) }
	out := []writerEntry{}
	for _, entry := range entries {
		var normalized writerEntry
		switch entry := entry.(type) {
		case ForegroundJournalEntryEvent:
			switch event := entry.Event.(type) {
			case ForegroundEventInput:
				normalized = writerEntry{kind: writerEntryInput, at: at(event.End)}
			case ForegroundEventSmallPolls:
				normalized = writerEntry{kind: writerEntrySmallPolls, count: event.Summary.Count, totalMicros: asMicros(event.Summary.Total), since: at(event.Since), until: at(event.Until)}
			case ForegroundEventPresent:
				normalized = writerEntry{kind: writerEntryPresent, window: event.WindowID.AsU64(), at: at(event.PresentEnd)}
			default:
				panic("writer property emitted an unexpected entry")
			}
		case ForegroundJournalEntryFrameState:
			switch state := entry.State.(type) {
			case FrameStateChangePending:
				normalized = writerEntry{kind: writerEntryPending, window: state.WindowID.AsU64(), at: at(state.DirtyAt)}
			case FrameStateChangeClosed:
				normalized = writerEntry{kind: writerEntryClosed, window: state.WindowID.AsU64(), at: at(state.At)}
			}
		case ForegroundJournalEntryBoundary:
			switch boundary := entry.Boundary.(type) {
			case IntervalBoundaryPresented:
				normalized = writerEntry{kind: writerEntryPresented, window: boundary.Frame.WindowID.AsU64(), at: at(boundary.Presentation.PresentEnd)}
			case IntervalBoundaryIdle:
				normalized = writerEntry{kind: writerEntryIdle, at: at(boundary.EndedAt)}
			default:
				panic("writer property emitted an unexpected entry")
			}
		default:
			panic("writer property emitted an unexpected entry")
		}
		out = append(out, normalized)
	}
	return out
}

func journalPropertyRNG() *rand.Rand {
	return rand.New(rand.NewPCG(PropertyTestRNGSeed(), 0))
}

func rangeInclusive(rng *rand.Rand, lo, hi int) int {
	return lo + rng.IntN(hi-lo+1)
}

func TestJournal_RingMatchesReferenceModelUnderWrapCollisionsAndIndependentCursors(t *testing.T) {
	rng := journalPropertyRNG()
	for range defaultPropertyTestCases() {
		capacity := rangeInclusive(rng, 1, 8)
		pendingCapacity := rangeInclusive(rng, 0, 4)
		operationCount := rangeInclusive(rng, 1, 128)

		origin := scheduler.InstantNow()
		journal, publisher := newForegroundJournal(capacity, pendingCapacity)
		collectors := []*ForegroundJournalCollector{journal.Collector()}
		modelCursors := []uint64{0}
		model := newModelRing(capacity, pendingCapacity)
		pins := make([]*journalSlotReadGuard, capacity)

		for range operationCount {
			weight := rng.IntN(13)
			argument := uint8(rng.Uint32())
			switch {
			case weight < 5:
				sequence := model.nextSequence
				publisher.publish(ringEntry(origin, sequence))
				model.publish()
			case weight < 8:
				index := int(argument) % len(collectors)
				drained := collectors[index].CollectUnseen()
				expectedEntries, expectedLost := model.collect(&modelCursors[index])
				observed := normalizeRingDrain(origin, drained.Entries)
				if len(observed) != 0 || len(expectedEntries) != 0 {
					assertEq(t, observed, expectedEntries)
				}
				assertEq(t, drained.Lost, expectedLost)
			case weight < 9:
				if len(collectors) < 4 {
					collectors = append(collectors, journal.Collector())
					modelCursors = append(modelCursors, model.offered)
				}
			case weight < 11:
				index := int(argument) % capacity
				if pins[index] == nil {
					guard, ok := tryNewJournalSlotReadGuard(&journal.ring.slots[index])
					if !ok {
						t.Fatal("pin should succeed")
					}
					pins[index] = guard
					model.pinned[index] = true
				}
			default:
				index := int(argument) % capacity
				if pins[index] != nil {
					pins[index].End()
					pins[index] = nil
				}
				model.pinned[index] = false
			}

			assertEq(t, publisher.nextSequence, model.nextSequence)
			assertEq(t, publisher.ring.offered.Load(), model.offered)
			assertEq(t, publisher.ring.finalized.Load(), model.finalized)
			var publisherPending, modelPending []uint64
			for _, p := range publisher.pending {
				publisherPending = append(publisherPending, p.sequence)
			}
			for _, p := range model.pending {
				modelPending = append(modelPending, p.sequence)
			}
			assertEq(t, publisherPending, modelPending)
			assertEq(t, publisher.droppedAfterPending, model.droppedAfterPending)
			if t.Failed() {
				return
			}
		}

		for i, pin := range pins {
			if pin != nil {
				pin.End()
				pins[i] = nil
			}
		}
		for i := range model.pinned {
			model.pinned[i] = false
		}
		publisher.flushPending()
		model.flushPending()

		for i, collector := range collectors {
			drained := collector.CollectUnseen()
			expectedEntries, expectedLost := model.collect(&modelCursors[i])
			observed := normalizeRingDrain(origin, drained.Entries)
			if len(observed) != 0 || len(expectedEntries) != 0 {
				assertEq(t, observed, expectedEntries)
			}
			assertEq(t, drained.Lost, expectedLost)
		}
		if t.Failed() {
			return
		}
	}
}

func TestJournal_APresentationSealsExactlyOnceRegardlessOfDelay(t *testing.T) {
	rng := journalPropertyRNG()
	for range defaultPropertyTestCases() {
		presentDelayMicros := uint64(rangeInclusive(rng, 1, 2_000_000))
		start := scheduler.InstantNow()
		windowID := WindowID(0xDEA8)
		presentedAt := micros(start, presentDelayMicros)
		sealer := NewIntervalSealer(start)
		snapshots := sealer.PushEntries([]ForegroundJournalEntry{
			pendingFrame(windowID, start),
			inputEntry(start, start.Add(time.Microsecond)),
			ForegroundJournalEntryBoundary{presentedBoundary(windowID, start, presentedAt)},
		})
		assertEq(t, len(snapshots), 1)
		if _, ok := snapshots[0].Boundary.(IntervalBoundaryPresented); !ok {
			t.Fatalf("unexpected boundary %#v", snapshots[0].Boundary)
		}
		assertEq(t, snapshots[0].IntervalEnd(), presentedAt)
		if t.Failed() {
			return
		}
	}
}

func TestJournal_CompletionOrderSealerMatchesReferenceUnderArbitraryBatching(t *testing.T) {
	rng := journalPropertyRNG()
	for range defaultPropertyTestCases() {
		specifications := make([][3]uint8, rangeInclusive(rng, 1, 96))
		for i := range specifications {
			specifications[i] = [3]uint8{uint8(rng.IntN(16)), uint8(rng.IntN(128)), uint8(rng.Uint32())}
		}
		batchSizes := make([]int, rangeInclusive(rng, 1, 24))
		for i := range batchSizes {
			batchSizes[i] = rangeInclusive(rng, 1, 12)
		}

		origin := scheduler.InstantNow()
		entries := completionOrderEntries(origin, specifications)
		expectedSnapshots, expectedTail := referenceSeal(origin, entries)

		oneBatchSealer := NewIntervalSealer(origin)
		oneBatchSnapshots := []normalizedSnapshot{}
		for _, snapshot := range oneBatchSealer.PushEntries(entries) {
			oneBatchSnapshots = append(oneBatchSnapshots, normalizeSnapshot(origin, &snapshot))
		}
		assertEq(t, oneBatchSnapshots, expectedSnapshots)
		assertEq(t, normalizeTail(origin, oneBatchSealer), expectedTail)

		batchedSealer := NewIntervalSealer(origin)
		batchedSnapshots := []normalizedSnapshot{}
		offset := 0
		batchIndex := 0
		for offset < len(entries) {
			batchSize := batchSizes[batchIndex%len(batchSizes)]
			batchEnd := min(offset+batchSize, len(entries))
			for _, snapshot := range batchedSealer.PushEntries(entries[offset:batchEnd]) {
				batchedSnapshots = append(batchedSnapshots, normalizeSnapshot(origin, &snapshot))
			}
			offset = batchEnd
			batchIndex++
		}
		assertEq(t, batchedSnapshots, expectedSnapshots)
		assertEq(t, normalizeTail(origin, batchedSealer), expectedTail)
		if t.Failed() {
			return
		}
	}
}

func TestJournal_OccupancyMatchesIntervalUnionAndFoldApportionment(t *testing.T) {
	rng := journalPropertyRNG()
	for range defaultPropertyTestCases() {
		eventSpans := make([][2]uint8, rangeInclusive(rng, 0, 16))
		for i := range eventSpans {
			eventSpans[i] = [2]uint8{uint8(rng.IntN(129)), uint8(rng.IntN(129))}
		}
		smallPollSpans := make([][3]uint8, rangeInclusive(rng, 0, 12))
		for i := range smallPollSpans {
			smallPollSpans[i] = [3]uint8{uint8(rng.IntN(129)), uint8(rng.IntN(129)), uint8(rng.IntN(5))}
		}
		window := [2]uint8{uint8(rng.IntN(129)), uint8(rng.IntN(129))}

		origin := scheduler.InstantNow()
		windowStart := min(window[0], window[1])
		windowEnd := max(window[0], window[1])
		events := []ForegroundEvent{}
		for _, s := range eventSpans {
			events = append(events, ForegroundEventInput{InputTiming{
				Kind:  "test",
				Start: micros(origin, uint64(min(s[0], s[1]))),
				End:   micros(origin, uint64(max(s[0], s[1]))),
			}})
		}
		smallPolls := []SmallPollFlush{}
		for _, s := range smallPollSpans {
			since := min(s[0], s[1])
			until := max(s[0], s[1])
			span := uint64(until - since)
			total := uint64(s[2])
			if span != 0 {
				total = span * uint64(s[2])
			}
			smallPolls = append(smallPolls, SmallPollFlush{
				Summary: PollSummary{Count: 1, Total: time.Duration(total) * time.Microsecond},
				Since:   micros(origin, uint64(since)),
				Until:   micros(origin, uint64(until)),
			})
		}
		snapshot := FrameSnapshot{
			IntervalStart: origin,
			Boundary:      IntervalBoundaryIdle{EndedAt: micros(origin, 128)},
			Events:        events,
			SmallPolls:    smallPolls,
		}
		expected := referenceOccupancyMicros(eventSpans, smallPollSpans, windowStart, windowEnd)
		observed := snapshot.OccupancyWithin(micros(origin, uint64(windowStart)), micros(origin, uint64(windowEnd)))
		assertEq(t, observed, time.Duration(expected)*time.Microsecond)

		fullExpected := referenceOccupancyMicros(eventSpans, smallPollSpans, 0, 128)
		assertEq(t, snapshot.Occupancy(), time.Duration(fullExpected)*time.Microsecond)
		busy := snapshot.BusyFraction()
		if busy < 0 || busy > 1 {
			t.Fatalf("busy fraction %v out of range", busy)
		}
		assertEq(t, busy, min(float64(fullExpected)/128.0, 1.0))
		if t.Failed() {
			return
		}
	}
}

func TestJournal_WriterMatchesStateModelAcrossTurnFrameAndRunnableTransitions(t *testing.T) {
	rng := journalPropertyRNG()
	for range defaultPropertyTestCases() {
		operationCount := rangeInclusive(rng, 1, 128)
		origin := scheduler.InstantNow()
		counter := newForegroundRunnableCounter()
		writer, collector := testJournal(counter)
		model := &modelWriter{pendingFrames: map[uint64]uint64{}}
		atMicros := uint64(100)

		for range operationCount {
			kind := uint8(rng.Uint32())
			advance := uint16(rng.Uint32())
			argument := uint8(rng.Uint32())
			if advance%16 == 0 {
				atMicros += asMicros(FrameDeadline)
			} else {
				atMicros += uint64(advance) + 1
			}
			at := micros(origin, atMicros)
			windowID := WindowID(uint64(argument % 4))
			window := windowID.AsU64()

			switch kind % 10 {
			case 0:
				writer.beginTurn()
				model.turnDepth++
			case 1:
				if model.turnDepth > 0 {
					writer.endTurn(at)
					model.endTurn(atMicros)
				}
			case 2:
				writer.recordEvent(ForegroundEventInput{InputTiming{Kind: "test", Start: at, End: at}})
				model.recordInput(atMicros)
			case 3:
				durationMicros := uint64(argument % 100)
				writer.foldSmallPoll(taskTiming(instantSub(at, time.Duration(durationMicros)*time.Microsecond), at))
				model.foldSmallPoll(atMicros-durationMicros, atMicros)
			case 4:
				counter.queued()
				model.runnables++
			case 5:
				if model.runnables > 0 {
					counter.finished()
					model.runnables--
				}
			case 6:
				writer.recordFramePending(windowID, at)
				model.recordPending(window, atMicros)
			case 7:
				writer.recordWindowClosed(windowID, at)
				model.recordClosed(window, atMicros)
			case 8:
				writer.recordPresent(presentationTiming(windowID, at), Some(frameTiming(windowID, at, at)))
				model.recordPresent(window, atMicros, true)
			default:
				writer.recordPresent(presentationTiming(windowID, at), None[FrameTiming]())
				model.recordPresent(window, atMicros, false)
			}

			drained := collector.CollectUnseen()
			assertEq(t, drained.Lost, uint64(0))
			expected := model.entries
			model.entries = nil
			if expected == nil {
				expected = []writerEntry{}
			}
			assertEq(t, normalizeWriterEntries(origin, drained.Entries), expected)
			assertEq(t, writer.turnDepth, model.turnDepth)
			assertEq(t, writer.retainedSinceBoundary, model.retainedSinceBoundary)
			pending := 0
			for _, w := range writer.windows {
				if w.dirtyAt.IsSome() {
					pending++
				}
			}
			assertEq(t, pending, len(model.pendingFrames))
			if t.Failed() {
				return
			}
		}
	}
}

func testJournal(runnables foregroundRunnableCounter) (*foregroundJournalWriter, *ForegroundJournalCollector) {
	journal, publisher := newForegroundJournal(256, 8)
	collector := journal.Collector()
	return newForegroundJournalWriter(runnables, publisher).withTestSuspendClock(), collector
}

func hasBoundaryAt(entries []ForegroundJournalEntry, at scheduler.Instant) bool {
	return anyEntry(entries, func(entry ForegroundJournalEntry) bool {
		b, ok := entry.(ForegroundJournalEntryBoundary)
		return ok && b.Boundary.EndTime() == at
	})
}

func hasIdleAt(entries []ForegroundJournalEntry, at scheduler.Instant) bool {
	return anyEntry(entries, func(entry ForegroundJournalEntry) bool {
		b, ok := entry.(ForegroundJournalEntryBoundary)
		if !ok {
			return false
		}
		idle, ok := b.Boundary.(IntervalBoundaryIdle)
		return ok && idle.EndedAt == at
	})
}

func anyEntry(entries []ForegroundJournalEntry, f func(ForegroundJournalEntry) bool) bool {
	for _, entry := range entries {
		if f(entry) {
			return true
		}
	}
	return false
}

func isBoundary[B IntervalBoundary](entry ForegroundJournalEntry) bool {
	b, ok := entry.(ForegroundJournalEntryBoundary)
	if !ok {
		return false
	}
	_, ok = b.Boundary.(B)
	return ok
}

func isEvent[E ForegroundEvent](entry ForegroundJournalEntry) bool {
	e, ok := entry.(ForegroundJournalEntryEvent)
	if !ok {
		return false
	}
	_, ok = e.Event.(E)
	return ok
}

func inputEntry(start, end scheduler.Instant) ForegroundJournalEntry {
	return ForegroundJournalEntryEvent{ForegroundEventInput{InputTiming{Kind: "test", Start: start, End: end}}}
}

func pendingFrame(windowID WindowID, dirtyAt scheduler.Instant) ForegroundJournalEntry {
	return ForegroundJournalEntryFrameState{FrameStateChangePending{WindowID: windowID, DirtyAt: dirtyAt}}
}

func frameTiming(windowID WindowID, dirtyAt, drawEnd scheduler.Instant) FrameTiming {
	return FrameTiming{
		WindowID:      windowID,
		DirtyAt:       Some(dirtyAt),
		Invalidations: 1,
		DrawStart:     drawEnd,
		DrawEnd:       drawEnd,
	}
}

func presentationTiming(windowID WindowID, presentEnd scheduler.Instant) PresentTiming {
	return PresentTiming{WindowID: windowID, PresentStart: presentEnd, PresentEnd: presentEnd}
}

func presentedBoundary(windowID WindowID, dirtyAt, presentEnd scheduler.Instant) IntervalBoundary {
	return IntervalBoundaryPresented{PresentedFrame{
		Frame:        frameTiming(windowID, dirtyAt, presentEnd),
		Presentation: presentationTiming(windowID, presentEnd),
	}}
}

func taskTiming(start, end scheduler.Instant) TaskTiming {
	return TaskTiming{
		Location: scheduler.CallerLocation(1),
		Spawned:  scheduler.SpawnTime{Instant: start},
		Start:    start,
		End:      YieldTime{end},
	}
}

func micros(origin scheduler.Instant, n uint64) scheduler.Instant {
	return origin.Add(time.Duration(n) * time.Microsecond)
}

func asMicros(d time.Duration) uint64 {
	return uint64(d / time.Microsecond)
}

func instantSub(i scheduler.Instant, d time.Duration) scheduler.Instant {
	return i.Add(-d)
}

func assertEq[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("assertion `left == right` failed\n  left: %#v\n right: %#v", got, want)
	}
}
