package gpui

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

type hangyAction struct{}

func (hangyAction) Name() string         { return "hang_test::HangyAction" }
func (a hangyAction) BoxedClone() Action { return a }
func (hangyAction) PartialEq(other Action) bool {
	_, ok := other.(hangyAction)
	return ok
}

func init() {
	RegisterAction(ActionData{
		Name: "hang_test::HangyAction",
		Type: reflect.TypeFor[hangyAction](),
		Build: func(jsontext.Value) (Action, error) {
			return hangyAction{}, nil
		},
	})
}

const hangTestThreshold = 10 * time.Millisecond

const hangTestFrameBudget = hangTestThreshold

const hangTestProfilerSkip = "requires the gpui_profiler build tag (upstream's profiler::hang is cfg(feature = \"profiler\"))"

func TestHang_CompletedSkipsPreserveRealHangsAndSerializeTheirBoundary(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	windowID := WindowID(1)
	for _, discontinuous := range []bool{false, true} {
		for _, trigger := range []HangTrigger{HangTriggerThreshold, HangTriggerBlockingWork} {
			sealer := NewIntervalSealer(startup)
			var entries []ForegroundJournalEntry
			if trigger == HangTriggerThreshold {
				entries = append(entries, ForegroundJournalEntryEvent{hangTaskPollEvent(at(10), at(30))})
			} else {
				entries = append(entries,
					ForegroundJournalEntryEvent{hangTaskPollEvent(at(10), at(16))},
					ForegroundJournalEntryEvent{hangTaskPollEvent(at(20), at(26))},
				)
			}
			if discontinuous {
				entries = append(entries, ForegroundJournalEntryDiscontinuity{Lost: 1})
			}
			entries = append(entries, ForegroundJournalEntryBoundary{IntervalBoundaryFrameSkipped{FrameSkipped{
				WindowID: windowID,
				At:       at(60),
				Reason:   FrameSkipReasonNoRenderNeeded,
			}}})
			snapshots := sealer.PushEntries(entries)
			if len(snapshots) != 1 {
				t.Fatalf("expected one snapshot, got %#v", snapshots)
			}
			incident, ok := HangIncidentDetect(snapshots[0], hangTestThreshold, hangTestFrameBudget)
			if trigger == HangTriggerBlockingWork {
				if ok {
					t.Fatal("short work isn't blocking")
				}
				continue
			}
			if !ok {
				t.Fatal("real work still qualifies across a skip")
			}
			assertEq(t, incident.Trigger, trigger)
			serialized := SerializedHangIncidentConvert(startup, &incident, 10, None[scheduler.Instant]())
			assertEq(t, serialized.SealedBy, "frame_skipped")
			assertEq(t, serialized.DirtyToPresentMs, None[float64]())
			assertEq(t, serialized.StartMs, 10.0)
			assertEq(t, serialized.ActiveMs, 50.0)
			assertEq(t, serialized.JournalDiscontinuous, discontinuous)
			assertEq(t, serialized.EventCount, len(incident.Contributors))
			data, err := json.Marshal(serialized)
			if err != nil {
				t.Fatalf("serialize incident: %v", err)
			}
			var value map[string]any
			if err := json.Unmarshal(data, &value); err != nil {
				t.Fatalf("decode incident: %v", err)
			}
			assertEq[any](t, value["sealed_by"], "frame_skipped")
			if v, present := value["dirty_to_present_ms"]; !present || v != nil {
				t.Fatalf("dirty_to_present_ms = %v (present %v), want null", v, present)
			}
		}
	}
}

func TestHang_DelayedPlatformSignalsWithoutWorkAreNotHangs(t *testing.T) {
	start := scheduler.InstantNow()
	windowID := WindowID(1)
	sealer := NewIntervalSealer(start)
	snapshots := sealer.PushEntries([]ForegroundJournalEntry{
		ForegroundJournalEntryPlatformSignal{PlatformSignal{
			WindowID:  windowID,
			SignalAt:  Some(start),
			HandledAt: start.Add(5 * time.Second),
			Source:    FrameRequestSourceNativeCallback,
		}},
		ForegroundJournalEntryBoundary{IntervalBoundaryFrameSkipped{FrameSkipped{
			WindowID: windowID,
			At:       start.Add(6 * time.Second),
			Reason:   FrameSkipReasonNoRenderNeeded,
		}}},
	})
	if len(snapshots) != 0 {
		t.Fatalf("a late signal without work seals nothing, got %#v", snapshots)
	}
}

func TestHang_AHangOutlivingTheFrameDeadlineKeepsItsFrameAssociation(t *testing.T) {
	start := scheduler.InstantNow()
	windowID := WindowID(0x51E17)
	hangEnd := start.Add(FrameDeadline * 5)
	presentedAt := hangEnd.Add(16 * time.Millisecond)
	journal, guard := installTestForegroundJournal(64, 8)
	defer guard.End()
	detector := NewHangDetector(journal, hangTestThreshold, hangTestFrameBudget)

	snapshots := detector.sealer.PushEntries([]ForegroundJournalEntry{
		ForegroundJournalEntryFrameState{FrameStateChangePending{WindowID: windowID, DirtyAt: start}},
		ForegroundJournalEntryEvent{hangTaskPollEvent(start, hangEnd)},
	})
	if len(snapshots) != 0 {
		t.Fatal("nothing seals mid-hang")
	}

	snapshots = detector.sealer.PushEntries([]ForegroundJournalEntry{
		ForegroundJournalEntryBoundary{IntervalBoundaryPresented{PresentedFrame{
			Frame: hangFrame(windowID, presentedAt),
			Presentation: PresentTiming{
				WindowID:     windowID,
				PresentStart: presentedAt.Add(-time.Millisecond),
				PresentEnd:   presentedAt,
			},
		}}},
	})
	if len(snapshots) != 1 {
		t.Fatalf("expected one presented snapshot, got %#v", snapshots)
	}
	incident, ok := HangIncidentDetect(snapshots[0], hangTestThreshold, hangTestFrameBudget)
	if !ok {
		t.Fatal("the hang qualifies")
	}
	if poll, ok := incident.Contributors[0].(ForegroundEventTaskPoll); !ok || !poll.End.Equal(hangEnd) {
		t.Fatalf("unexpected contributor %#v", incident.Contributors[0])
	}
	if _, ok := incident.Snapshot.Boundary.(IntervalBoundaryPresented); !ok {
		t.Fatalf("unexpected boundary %#v", incident.Snapshot.Boundary)
	}
}

func TestHang_SerializedIncidentReportsPresentedSealFields(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	windowID := WindowID(0xF1E1D)

	presentation := PresentTiming{WindowID: windowID, PresentStart: at(380), PresentEnd: at(400)}
	frame := FrameTiming{
		WindowID:      windowID,
		DirtyAt:       Some(at(100)),
		Invalidations: 3,
		DrawStart:     at(350),
		DrawEnd:       at(380),
	}
	snapshot := FrameSnapshot{
		IntervalStart: at(150),
		Boundary:      IntervalBoundaryPresented{PresentedFrame{Frame: frame, Presentation: presentation}},
		Events: []ForegroundEvent{
			hangTaskPollEvent(at(150), at(300)),
			ForegroundEventAction{ActionTiming{Name: "test::SlowAction", Start: at(300), End: at(340)}},
			ForegroundEventInput{InputTiming{Kind: "test", Start: at(340), End: at(345)}},
			ForegroundEventPresent{presentation},
		},
		SmallPolls: []SmallPollFlush{{
			Summary: PollSummary{Count: 2, Total: time.Millisecond},
			Since:   at(105),
			Until:   at(145),
		}},
	}

	incident, ok := HangIncidentDetect(snapshot, hangTestThreshold, hangTestFrameBudget)
	if !ok {
		t.Fatal("has contributors")
	}
	serialized := SerializedHangIncidentConvert(startup, &incident, 1, Some(at(50)))

	assertEq(t, serialized.Phase, "steady")
	assertEq(t, serialized.Trigger, HangTriggerThreshold)
	assertEq(t, serialized.StartMs, 100.0)
	assertEq(t, serialized.ActiveMs, 300.0)
	assertEq(t, serialized.StallMs, 150.0)
	assertEq(t, serialized.DirtyToPresentMs, Some(300.0))
	assertEq(t, serialized.SealedBy, "present")
	assertEq(t, serialized.BusyFraction, 0.72)
	assertEq(t, serialized.EventCount, 4)
	assertEq(t, serialized.SmallPollCount, uint64(2))
	assertEq(t, serialized.SmallPollTotalMs, 1.0)
	assertEq(t, serialized.DroppedEvents, uint64(0))
	assertEq(t, len(serialized.Contributors), 1)
	assertEq(t, serialized.ContributorsElided, 2)
	if poll, ok := serialized.Contributors[0].(SerializedHangContributorTaskPoll); !ok || poll.StartMs != 150.0 || poll.DurationMs != 150.0 {
		t.Fatalf("unexpected contributor %#v", serialized.Contributors[0])
	}
}

type hangMonitorReport struct {
	reason    HangMonitorPollReason
	incidents int
}

func TestHang_HangMonitorPollsOnItsIntervalAndOnFlush(t *testing.T) {
	if !profilerFeatureEnabled {
		t.Skip(hangTestProfilerSkip)
	}
	journal, guard := installTestForegroundJournal(64, 4)
	defer guard.End()
	thread := NewThreadID(Some("hang-test"))
	defer ReleaseThreadTimings(thread)
	spawnMonitor := func(journal ForegroundJournal, interval time.Duration) (*hangMonitor, chan hangMonitorReport) {
		receiver := make(chan hangMonitorReport, 64)
		monitor, err := spawnHangMonitor(
			NewHangDetector(journal, hangTestThreshold, hangTestFrameBudget),
			interval,
			func(poll HangMonitorPoll) {
				if len(poll.Incidents) > 0 {
					receiver <- hangMonitorReport{poll.Reason, len(poll.Incidents)}
				}
			},
		)
		if err != nil {
			t.Fatalf("spawn monitor thread: %v", err)
		}
		return monitor, receiver
	}
	assertDisconnected := func(monitor *hangMonitor, receiver chan hangMonitorReport) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for {
			monitor.mu.Lock()
			exited := monitor.exited
			monitor.mu.Unlock()
			if exited {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("the monitor thread did not exit")
			}
			time.Sleep(time.Millisecond)
		}
		select {
		case report := <-receiver:
			t.Fatalf("unexpected report %#v after the monitor exited", report)
		default:
		}
	}

	monitor, receiver := spawnMonitor(journal, 10*time.Millisecond)
	hangSimulateBlockedForegroundPoll(thread, hangTestThreshold*2)
	select {
	case report := <-receiver:
		assertEq(t, report, hangMonitorReport{HangMonitorPollReasonInterval, 1})
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for an interval poll")
	}
	monitor.Close()
	assertDisconnected(monitor, receiver)

	monitor, receiver = spawnMonitor(journal, time.Hour)
	hangSimulateBlockedForegroundPoll(thread, hangTestThreshold*2)
	flushed, ok := monitor.requestFlush()
	if !ok {
		t.Fatal("monitor thread is running")
	}
	<-flushed
	select {
	case report := <-receiver:
		assertEq(t, report, hangMonitorReport{HangMonitorPollReasonFlush, 1})
	default:
		t.Fatal("expected a flush report")
	}
	monitor.Close()
	assertDisconnected(monitor, receiver)
}

func TestHang_AppFlushesItsHangMonitorOnShutdown(t *testing.T) {
	if !profilerFeatureEnabled {
		t.Skip(hangTestProfilerSkip)
	}
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.Dispatcher().AllowParking()

		receiver := make(chan HangMonitorPollReason, 8)
		cx.Update(func(cx *App) {
			err := cx.StartHangMonitor(HangMonitorConfig{
				Threshold:   hangTestThreshold,
				FrameBudget: hangTestFrameBudget,
				Interval:    time.Hour,
			}, func(poll HangMonitorPoll) {
				receiver <- poll.Reason
			})
			if err != nil {
				t.Fatalf("start hang monitor: %v", err)
			}
		})
		defer cx.Update(func(cx *App) { cx.hangMonitor.Close() })
		select {
		case reason := <-receiver:
			t.Fatalf("unexpected poll %v", reason)
		default:
		}

		cx.Update(func(cx *App) { cx.Shutdown() })
		select {
		case reason := <-receiver:
			assertEq(t, reason, HangMonitorPollReasonFlush)
		default:
			t.Fatal("expected a flush poll")
		}
	})
}

func TestHang_StartingTheHangMonitorTwiceIsRejected(t *testing.T) {
	if !profilerFeatureEnabled {
		t.Skip(hangTestProfilerSkip)
	}
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		config := HangMonitorConfig{
			Threshold:   hangTestThreshold,
			FrameBudget: hangTestFrameBudget,
			Interval:    time.Hour,
		}
		cx.Update(func(cx *App) {
			if err := cx.StartHangMonitor(config, func(HangMonitorPoll) {}); err != nil {
				t.Fatalf("start hang monitor: %v", err)
			}
			defer cx.hangMonitor.Close()
			var second error
			panicked := func() (panicked bool) {
				defer func() {
					if recover() != nil {
						panicked = true
					}
				}()
				second = cx.StartHangMonitor(config, func(HangMonitorPoll) {})
				return false
			}()
			if gpuiutil.DebugAssertions {
				if !panicked {
					t.Fatal("expected a debug assertion")
				}
			} else if _, ok := second.(HangMonitorErrorAlreadyStarted); panicked || !ok {
				t.Fatalf("second start = %v, panicked %v", second, panicked)
			}
		})
	})
}

func TestHang_SerializedIncidentReportsIdleSealFields(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }

	idle := FrameSnapshot{
		IntervalStart: at(200),
		Boundary:      IntervalBoundaryIdle{EndedAt: at(260)},
		Events:        []ForegroundEvent{hangTaskPollEvent(at(200), at(260))},
	}
	incident, ok := HangIncidentDetect(idle, hangTestThreshold, hangTestFrameBudget)
	if !ok {
		t.Fatal("has contributors")
	}
	serialized := SerializedHangIncidentConvert(startup, &incident, 8, None[scheduler.Instant]())
	assertEq(t, serialized.Phase, "startup")
	assertEq(t, serialized.SealedBy, "idle")
	assertEq(t, serialized.DirtyToPresentMs, None[float64]())
	assertEq(t, serialized.StartMs, 200.0)
	assertEq(t, serialized.ActiveMs, 60.0)
	assertEq(t, serialized.StallMs, 60.0)
	assertEq(t, serialized.BusyFraction, 1.0)
}

func TestHang_PhaseIsStartupUntilTheFirstPresent(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	snapshot := FrameSnapshot{
		IntervalStart: at(500),
		Boundary:      IntervalBoundaryIdle{EndedAt: at(600)},
		Events:        []ForegroundEvent{hangTaskPollEvent(at(500), at(600))},
	}
	incident, ok := HangIncidentDetect(snapshot, hangTestThreshold, hangTestFrameBudget)
	if !ok {
		t.Fatal("has contributors")
	}

	phase := func(firstPresentAt Option[scheduler.Instant]) string {
		return SerializedHangIncidentConvert(startup, &incident, 8, firstPresentAt).Phase
	}
	if got := phase(None[scheduler.Instant]()); got != "startup" {
		t.Fatalf("nothing has been presented yet: %q", got)
	}
	if got := phase(Some(at(700))); got != "startup" {
		t.Fatalf("the incident began before the first presentation: %q", got)
	}
	if got := phase(Some(at(500))); got != "steady" {
		t.Fatalf("an incident beginning exactly at first presentation is steady: %q", got)
	}
	assertEq(t, phase(Some(at(100))), "steady")
}

func TestHang_FirstPresentAtLatchesOnTheFirstObservedPresentation(t *testing.T) {
	journal, guard := installTestForegroundJournal(64, 8)
	defer guard.End()
	detector := NewHangDetector(journal, hangTestThreshold, hangTestFrameBudget)
	windowID := WindowID(0x1A7C4)

	firstPresentEnd := scheduler.InstantNow()
	recordPresent(hangPresentation(windowID, firstPresentEnd), Some(hangFrame(windowID, firstPresentEnd)))
	detector.Poll()
	latched, ok := detector.FirstPresentAt()
	if !ok {
		t.Fatal("a presentation was recorded")
	}
	if !latched.Equal(firstPresentEnd) {
		t.Fatalf("latched %v, want %v", latched, firstPresentEnd)
	}

	next := firstPresentEnd.Add(16 * time.Millisecond)
	recordPresent(hangPresentation(windowID, next), Some(hangFrame(windowID, next)))
	detector.Poll()
	if got, ok := detector.FirstPresentAt(); !ok || !got.Equal(latched) {
		t.Fatalf("first present moved to %v", got)
	}
}

func TestHang_BusyFractionExcludesSmallPollsOutsideTheActiveWindow(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	snapshot := FrameSnapshot{
		IntervalStart: at(0),
		Boundary:      IntervalBoundaryIdle{EndedAt: at(1000)},
		Events:        []ForegroundEvent{hangTaskPollEvent(at(900), at(950))},
		SmallPolls: []SmallPollFlush{{
			Summary: PollSummary{Count: 500, Total: 400 * time.Millisecond},
			Since:   at(0),
			Until:   at(800),
		}},
	}

	incident, ok := HangIncidentDetect(snapshot, hangTestThreshold, hangTestFrameBudget)
	if !ok {
		t.Fatal("has contributors")
	}
	serialized := SerializedHangIncidentConvert(startup, &incident, 8, None[scheduler.Instant]())
	assertEq(t, serialized.BusyFraction, 0.5)
}

func TestHang_AnIntervalOfSmallWorkOverBudgetIsAnIncident(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	windowID := WindowID(0xB0D6E7)
	snapshot := FrameSnapshot{
		IntervalStart: at(0),
		Boundary: IntervalBoundaryPresented{PresentedFrame{
			Frame: FrameTiming{
				SignalAt:      Some(at(0)),
				WindowID:      windowID,
				DirtyAt:       Some(at(0)),
				Invalidations: 1,
				DrawStart:     at(140),
				DrawEnd:       at(145),
			},
			Presentation: PresentTiming{WindowID: windowID, PresentStart: at(145), PresentEnd: at(150)},
		}},
		Events: []ForegroundEvent{
			hangTaskPollEvent(at(0), at(5)),
			hangTaskPollEvent(at(5), at(13)),
			hangTaskPollEvent(at(13), at(18)),
		},
	}

	incident, ok := HangIncidentDetect(snapshot, hangTestThreshold, hangTestFrameBudget)
	if !ok {
		t.Fatal("the frame missed the 60 Hz budget used for unknown displays")
	}
	assertEq(t, incident.Trigger, HangTriggerLateFrame)
	assertEq(t, len(incident.Contributors), 3)
	assertEq(t, incident.Contributors[0].Duration(), 8*time.Millisecond)
	serialized := SerializedHangIncidentConvert(startup, &incident, 8, Some(startup))
	assertEq(t, serialized.Trigger, HangTriggerLateFrame)
	assertEq(t, serialized.StallMs, 8.0)
	assertEq(t, serialized.DirtyToPresentMs, Some(150.0))
	assertEq(t, serialized.SealedBy, "present")
}

func TestHang_AJournalGapSuppressesBudgetInferenceButRetainsObservedHangs(t *testing.T) {
	start := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return start.Add(time.Duration(ms) * time.Millisecond) }
	sealer := NewIntervalSealer(start)
	snapshots := sealer.PushEntries([]ForegroundJournalEntry{
		ForegroundJournalEntryEvent{hangTaskPollEvent(at(0), at(5))},
		ForegroundJournalEntryDiscontinuity{Lost: 1},
		ForegroundJournalEntryEvent{hangTaskPollEvent(at(10), at(15))},
		ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: at(15)}},
		ForegroundJournalEntryDiscontinuity{Lost: 1},
		ForegroundJournalEntryEvent{hangTaskPollEvent(at(20), at(32))},
		ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: at(32)}},
		ForegroundJournalEntryEvent{hangTaskPollEvent(at(40), at(52))},
		ForegroundJournalEntryBoundary{IntervalBoundaryIdle{EndedAt: at(52)}},
	})
	if len(snapshots) != 3 {
		t.Fatalf("expected two discontinuous snapshots followed by a clean one, got %#v", snapshots)
	}
	budgetOnly, observedHang, clean := snapshots[0], snapshots[1], snapshots[2]

	if !budgetOnly.JournalDiscontinuous {
		t.Fatal("expected a discontinuous snapshot")
	}
	assertEq(t, budgetOnly.DroppedEvents, uint64(1))
	if _, ok := HangIncidentDetect(budgetOnly, hangTestThreshold, hangTestFrameBudget); ok {
		t.Fatal("budget-only snapshot across a gap is not an incident")
	}

	observed, ok := HangIncidentDetect(observedHang, hangTestThreshold, hangTestFrameBudget)
	if !ok {
		t.Fatal("the directly observed threshold-qualified hang remains valid")
	}
	if !observed.Snapshot.JournalDiscontinuous {
		t.Fatal("expected a discontinuous snapshot")
	}
	assertEq(t, len(observed.Contributors), 1)
	serialized := SerializedHangIncidentConvert(start, &observed, 8, None[scheduler.Instant]())
	if !serialized.JournalDiscontinuous {
		t.Fatal("expected journal_discontinuous")
	}

	if clean.JournalDiscontinuous {
		t.Fatal("expected a clean snapshot")
	}
	if _, ok := HangIncidentDetect(clean, hangTestThreshold, hangTestFrameBudget); !ok {
		t.Fatal("expected an incident")
	}
}

func TestHang_ASlowFrameWithLittleForegroundSpendIsNotAnIncident(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	windowID := WindowID(0xFA57)
	snapshot := FrameSnapshot{
		IntervalStart: at(0),
		Boundary: IntervalBoundaryPresented{PresentedFrame{
			Frame: FrameTiming{
				WindowID:      windowID,
				DirtyAt:       Some(at(0)),
				Invalidations: 1,
				DrawStart:     at(145),
				DrawEnd:       at(147),
			},
			Presentation: PresentTiming{WindowID: windowID, PresentStart: at(149), PresentEnd: at(150)},
		}},
		Events: []ForegroundEvent{hangTaskPollEvent(at(0), at(5))},
	}

	if _, ok := HangIncidentDetect(snapshot, hangTestThreshold, hangTestFrameBudget); ok {
		t.Fatal("expected no incident")
	}
}

func TestHang_ActiveUseCountsTimeWithinAMinuteOfDeliberateInput(t *testing.T) {
	start := scheduler.InstantNow()
	at := func(seconds uint64) scheduler.Instant { return start.Add(time.Duration(seconds) * time.Second) }
	inputOfKind := func(kind string, seconds uint64) ForegroundJournalEntry {
		return ForegroundJournalEntryEvent{ForegroundEventInput{InputTiming{
			Kind:               kind,
			Start:              at(seconds),
			End:                at(seconds),
			CausedInvalidation: true,
		}}}
	}
	input := func(seconds uint64) ForegroundJournalEntry { return inputOfKind("key_down", seconds) }
	maxElapsed := 10 * time.Second
	var activity activeUse
	assertEq(t, activity.take(at(0), maxElapsed), time.Duration(0))
	activity.observe(inputOfKind("mouse_move", 1))
	activity.observe(inputOfKind("mouse_exited", 2))
	assertEq(t, activity.take(at(4), maxElapsed), time.Duration(0))
	activity.observe(input(5))
	assertEq(t, activity.take(at(8), maxElapsed), 3*time.Second)
	assertEq(t, activity.take(at(16), maxElapsed), 8*time.Second)
	for second := uint64(24); second <= 64; second += 8 {
		activity.take(at(second), maxElapsed)
	}
	assertEq(t, activity.take(at(70), maxElapsed), time.Second)
	assertEq(t, activity.take(at(78), maxElapsed), time.Duration(0))
	activity.observe(input(80))
	activity.observe(ForegroundJournalEntryBoundary{IntervalBoundaryPowerTransition{EndedAt: at(81)}})
	assertEq(t, activity.take(at(84), maxElapsed), time.Duration(0))
	assertEq(t, activity.take(at(88), maxElapsed), 4*time.Second)
	assertEq(t, activity.take(at(130), maxElapsed), maxElapsed)

	if !activity.wasActiveDuring(at(79), at(80)) {
		t.Fatal("expected active during 79..80")
	}
	if !activity.wasActiveDuring(at(100), at(101)) {
		t.Fatal("expected active during 100..101")
	}
	if activity.wasActiveDuring(at(141), at(142)) {
		t.Fatal("expected inactive during 141..142")
	}
	if activity.wasActiveDuring(at(66), at(70)) {
		t.Fatal("expected inactive during 66..70")
	}
}

func TestHang_OnlySingleEventsOverTheFrameBudgetAreBlockingWork(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	snapshot := func(longestMs uint64) FrameSnapshot {
		var events []ForegroundEvent
		for i := range uint64(10) {
			events = append(events, hangTaskPollEvent(at(i*20), at(i*20+5)))
		}
		events = append(events, hangTaskPollEvent(at(200), at(200+longestMs)))
		return FrameSnapshot{
			IntervalStart: at(0),
			Boundary:      IntervalBoundaryIdle{EndedAt: at(200)},
			Events:        events,
		}
	}
	frameBudget := 7 * time.Millisecond
	if _, ok := HangIncidentDetect(snapshot(6), hangTestThreshold, frameBudget); ok {
		t.Fatal("expected no incident")
	}
	incident, ok := HangIncidentDetect(snapshot(7), hangTestThreshold, frameBudget)
	if !ok {
		t.Fatal("one event as long as the frame budget")
	}
	assertEq(t, incident.Trigger, HangTriggerBlockingWork)
	assertEq(t, len(incident.Contributors), 1)
}

func TestHang_LateFramesAreBudgetHangs(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	windowID := WindowID(0xB0D6E7)
	detect := func(refreshInterval time.Duration, signalAt Option[scheduler.Instant], submittedMs uint64) (HangIncident, bool) {
		snapshot := FrameSnapshot{
			IntervalStart: at(0),
			Boundary: IntervalBoundaryPresented{PresentedFrame{
				Frame: FrameTiming{
					WindowID:        windowID,
					DirtyAt:         Some(at(0)),
					Invalidations:   1,
					DrawStart:       at(1),
					DrawEnd:         at(submittedMs - 1),
					RefreshInterval: Some(refreshInterval),
					SignalAt:        signalAt,
				},
				Presentation: PresentTiming{
					WindowID:     windowID,
					PresentStart: at(submittedMs),
					PresentEnd:   at(submittedMs + 50),
				},
			}},
			Events: []ForegroundEvent{
				hangTaskPollEvent(at(1), at(6)),
				hangTaskPollEvent(at(10), at(15)),
			},
		}
		return HangIncidentDetect(snapshot, hangTestThreshold, hangTestFrameBudget)
	}

	sixtyHertz := time.Second / 60
	incident, ok := detect(sixtyHertz, Some(at(0)), 25)
	if !ok {
		t.Fatal("work exceeded one refresh")
	}
	assertEq(t, incident.Trigger, HangTriggerLateFrame)
	assertEq(t, len(incident.Contributors), 2)
	incident, ok = detect(sixtyHertz, Some(at(8)), 25)
	if !ok {
		t.Fatal("17 ms of work at 60 Hz")
	}
	assertEq(t, len(incident.Contributors), 1)
	if _, ok := detect(sixtyHertz, Some(at(0)), 16); ok {
		t.Fatal("16 ms at 60 Hz is not late")
	}
	if _, ok := detect(sixtyHertz, None[scheduler.Instant](), 25); ok {
		t.Fatal("an unmeasured frame is not late")
	}

	threeHundredSixtyHertz := time.Second / 360
	if _, ok := detect(threeHundredSixtyHertz, Some(at(0)), 8); ok {
		t.Fatal("8 ms is within the 120 Hz floor")
	}
	if _, ok := detect(threeHundredSixtyHertz, Some(at(0)), 9); !ok {
		t.Fatal("9 ms exceeds the 120 Hz floor")
	}
}

func TestHang_SerializedContributorsAreChronologicalWithNestingDepths(t *testing.T) {
	startup := scheduler.InstantNow()
	at := func(ms uint64) scheduler.Instant { return startup.Add(time.Duration(ms) * time.Millisecond) }
	windowID := WindowID(0x2E57ED)
	snapshot := FrameSnapshot{
		IntervalStart: at(0),
		Boundary:      IntervalBoundaryIdle{EndedAt: at(80)},
		Events: []ForegroundEvent{
			hangTaskPollEvent(at(50), at(70)),
			ForegroundEventInput{InputTiming{Kind: "mouse_move", Start: at(0), End: at(40), CausedInvalidation: true}},
			ForegroundEventDraw{FrameTiming{
				WindowID:      windowID,
				DirtyAt:       Some(at(0)),
				Invalidations: 1,
				DrawStart:     at(1),
				DrawEnd:       at(39),
			}},
		},
	}

	incident, ok := HangIncidentDetect(snapshot, hangTestThreshold, hangTestFrameBudget)
	if !ok {
		t.Fatal("has contributors")
	}
	serialized := SerializedHangIncidentConvert(startup, &incident, 8, Some(startup))

	assertEq(t, serialized.StallMs, 40.0)
	contributors := serialized.Contributors
	if len(contributors) != 3 {
		t.Fatalf("unexpected contributors %#v", contributors)
	}
	input, ok0 := contributors[0].(SerializedHangContributorInput)
	draw, ok1 := contributors[1].(SerializedHangContributorDraw)
	poll, ok2 := contributors[2].(SerializedHangContributorTaskPoll)
	if !ok0 || !ok1 || !ok2 || input.InputKind != "mouse_move" || input.Depth != 0 || draw.Depth != 1 || poll.Depth != 0 {
		t.Fatalf("unexpected contributors %#v", contributors)
	}
}

func TestHang_DetectionMatchesThresholdAndBlockingWorkModelAtBoundaries(t *testing.T) {
	PropertyTest(t, func(tc *TestRun, rng *rand.Rand) {
		t := tc.T()
		durationsMs := make([]uint16, rng.IntN(17))
		for i := range durationsMs {
			durationsMs[i] = uint16(rng.IntN(51))
		}
		thresholdMs := uint16(rng.IntN(61))
		frameBudgetMs := uint16(rng.IntN(61))
		smallPollTotalMs := uint16(rng.IntN(101))

		origin := scheduler.InstantNow()
		cursorMs := uint64(0)
		events := make([]ForegroundEvent, 0, len(durationsMs))
		for _, durationMs := range durationsMs {
			start := origin.Add(time.Duration(cursorMs) * time.Millisecond)
			cursorMs += uint64(durationMs)
			end := origin.Add(time.Duration(cursorMs) * time.Millisecond)
			cursorMs++
			events = append(events, ForegroundEventInput{InputTiming{Kind: "test", Start: start, End: end}})
		}
		intervalEndMs := max(cursorMs, 1)
		intervalEnd := origin.Add(time.Duration(intervalEndMs) * time.Millisecond)
		windowID := WindowID(1)
		smallPollCount := uint64(0)
		if smallPollTotalMs > 0 {
			smallPollCount = 1
		}
		snapshot := FrameSnapshot{
			IntervalStart: origin,
			Boundary: IntervalBoundaryPresented{PresentedFrame{
				Frame: FrameTiming{
					WindowID:      windowID,
					DirtyAt:       Some(origin),
					Invalidations: 1,
					DrawStart:     intervalEnd,
					DrawEnd:       intervalEnd,
				},
				Presentation: PresentTiming{WindowID: windowID, PresentStart: intervalEnd, PresentEnd: intervalEnd},
			}},
			Events: events,
			SmallPolls: []SmallPollFlush{{
				Summary: PollSummary{Count: smallPollCount, Total: time.Duration(smallPollTotalMs) * time.Millisecond},
				Since:   origin,
				Until:   origin.Add(time.Duration(intervalEndMs) * time.Millisecond),
			}},
		}
		var qualifying, blocking []uint16
		for _, duration := range durationsMs {
			if duration >= thresholdMs {
				qualifying = append(qualifying, duration)
			}
			if duration >= frameBudgetMs {
				blocking = append(blocking, duration)
			}
		}
		expectedIncident := len(qualifying) > 0 || len(blocking) > 0

		incident, ok := HangIncidentDetect(
			snapshot,
			time.Duration(thresholdMs)*time.Millisecond,
			time.Duration(frameBudgetMs)*time.Millisecond,
		)
		if ok != expectedIncident {
			t.Fatalf("incident = %v, want %v (durations %v, threshold %d, budget %d)", ok, expectedIncident, durationsMs, thresholdMs, frameBudgetMs)
		}
		if ok {
			expected := qualifying
			if len(qualifying) == 0 {
				expected = blocking
			}
			slices.SortFunc(expected, func(a, b uint16) int { return int(b) - int(a) })
			observed := make([]uint16, 0, len(incident.Contributors))
			for _, event := range incident.Contributors {
				observed = append(observed, uint16(event.Duration()/time.Millisecond))
			}
			if !slices.Equal(observed, expected) {
				t.Fatalf("contributors = %v, want %v", observed, expected)
			}
		}
	})
}

type hangControls struct {
	render *Option[time.Duration]
	input  *Option[time.Duration]
	action *Option[time.Duration]
}

func newHangControls() hangControls {
	return hangControls{render: new(Option[time.Duration]), input: new(Option[time.Duration]), action: new(Option[time.Duration])}
}

type hangyView struct {
	controls    hangControls
	focusHandle FocusHandle
}

func (v *hangyView) Render(_ *Window, cx *Context[hangyView]) IntoElement {
	if duration, ok := v.controls.render.Take().Get(); ok {
		time.Sleep(duration)
	}
	actionControls := v.controls
	inputControls := v.controls
	onAction := cx.Listener(func(_ *hangyView, _ *hangyAction, _ *Window, _ *Context[hangyView]) {
		if duration, ok := actionControls.action.Take().Get(); ok {
			time.Sleep(duration)
		}
	})
	return NewDiv().
		SizeFull().
		TrackFocus(v.focusHandle).
		OnAction(func(action hangyAction, window *Window, cx *App) { onAction(&action, window, cx) }).
		OnMouseDown(MouseButtonLeft, func(*MouseDownEvent, *Window, *App) {
			if duration, ok := inputControls.input.Take().Get(); ok {
				time.Sleep(duration)
			}
		})
}

type hangKind uint8

const (
	hangKindRender hangKind = iota
	hangKindInput
	hangKindAction
	hangKindPoll
)

func (k hangKind) String() string {
	return [...]string{"Render", "Input", "Action", "Poll"}[k]
}

func TestHang_DetectsRandomlyPlacedForegroundHangs(t *testing.T) {
	if !profilerFeatureEnabled {
		t.Skip(hangTestProfilerSkip)
	}
	RunTest(t, func(tc *TestRun) {
		rng := tc.Rand()
		journal, guard := installTestForegroundJournal(1024, 16)
		defer guard.End()
		thread := NewThreadID(Some("hang-test"))
		defer ReleaseThreadTimings(thread)
		detector := NewHangDetector(journal, hangTestThreshold, hangTestFrameBudget)

		controls := newHangControls()
		view, cx := tc.Cx().AddWindowView(func(_ *Window, cx *Context[hangyView]) *hangyView {
			return &hangyView{controls: controls, focusHandle: cx.FocusHandle()}
		})
		defer func() {
			var focusHandle FocusHandle
			cx.Update(func(_ *Window, cx *App) { focusHandle = view.Read(cx).focusHandle })
			view.Release()
			focusHandle.Release()
		}()
		if err := view.UpdateIn(cx, func(view *hangyView, window *Window, cx *Context[hangyView]) {
			window.Focus(view.focusHandle, cx.App)
		}); err != nil {
			t.Fatalf("focus: %v", err)
		}
		hangDrawWindow(cx)

		var injections []hangInjection
		for range 1 + rng.IntN(4) {
			duration := hangTestThreshold + time.Duration(5+rng.IntN(20))*time.Millisecond
			started := scheduler.InstantNow()
			var kind hangKind
			switch rng.IntN(4) {
			case 0:
				*controls.render = Some(duration)
				if err := view.UpdateIn(cx, func(_ *hangyView, _ *Window, cx *Context[hangyView]) { cx.Notify() }); err != nil {
					t.Fatalf("notify: %v", err)
				}
				hangDrawWindow(cx)
				kind = hangKindRender
			case 1:
				*controls.input = Some(duration)
				cx.SimulateMouseDown(NewPoint(Px(5.), Px(5.)), MouseButtonLeft, Modifiers{})
				kind = hangKindInput
			case 2:
				*controls.action = Some(duration)
				cx.DispatchAction(hangyAction{})
				kind = hangKindAction
			default:
				hangSimulateBlockedForegroundPoll(thread, duration)
				kind = hangKindPoll
			}
			injections = append(injections, hangInjection{
				kind:     kind,
				duration: duration,
				start:    started,
				end:      scheduler.InstantNow(),
			})

			if rng.Float64() < 0.5 {
				hangDrawWindow(cx)
			}
			if rng.Float64() < 0.5 {
				cx.SimulateMouseUp(NewPoint(Px(5.), Px(5.)), MouseButtonLeft, Modifiers{})
			}
		}

		hangDrawWindow(cx)

		var contributors []ForegroundEvent
		for _, incident := range detector.Poll() {
			contributors = append(contributors, incident.Contributors...)
		}
		for _, injection := range injections {
			hangAssertDetectedOnce(t, injection, contributors)
		}
	}, Iterations(5))
}

const (
	hangTestSlowParentRender = 3 * time.Millisecond
	hangTestSlowChildRender  = 2 * time.Millisecond
)

type hangSlowParent struct {
	child AnyView
}

func (v *hangSlowParent) Render(_ *Window, _ *Context[hangSlowParent]) IntoElement {
	time.Sleep(hangTestSlowParentRender)
	return NewDiv().SizeFull().Child(v.child.Clone())
}

type hangSlowChild struct{}

func (*hangSlowChild) Render(_ *Window, _ *Context[hangSlowChild]) IntoElement {
	time.Sleep(hangTestSlowChildRender)
	return NewDiv().SizeFull()
}

func TestHang_SlowViewsAreJournaledWithTheirOwnTime(t *testing.T) {
	if !profilerFeatureEnabled {
		t.Skip(hangTestProfilerSkip)
	}
	RunTest(t, func(tc *TestRun) {
		journal, guard := installTestForegroundJournal(1024, 16)
		defer guard.End()
		collector := journal.Collector()
		view, cx := tc.Cx().AddWindowView(func(_ *Window, cx *Context[hangSlowParent]) *hangSlowParent {
			return &hangSlowParent{child: AnyViewFromEntity(cx.New(func(*Context[hangSlowChild]) *hangSlowChild { return &hangSlowChild{} }))}
		})
		defer func() {
			var child AnyView
			cx.Update(func(_ *Window, cx *App) { child = view.Read(cx).child })
			view.Release()
			child.Release()
		}()
		hangDrawWindow(cx)

		var events []ForegroundEvent
		for _, entry := range collector.CollectUnseen().Entries {
			if event, ok := entry.(ForegroundJournalEntryEvent); ok {
				events = append(events, event.Event)
			}
		}
		viewTiming := func(suffix string, phase ViewPhase) ViewTiming {
			for _, event := range slices.Backward(events) {
				if timing, ok := event.(ForegroundEventView); ok && strings.HasSuffix(timing.Name, suffix) && timing.Phase == phase {
					return timing.ViewTiming
				}
			}
			panic("expected a " + phase.AsStr() + " timing for " + suffix)
		}
		parent := viewTiming(".hangSlowParent", ViewPhaseRequestLayout)
		child := viewTiming(".hangSlowChild", ViewPhaseRequestLayout)
		parentDuration := parent.End.SaturatingDurationSince(parent.Start)
		childDuration := child.End.SaturatingDurationSince(child.Start)
		if child.SelfDuration < hangTestSlowChildRender {
			t.Fatalf("child self duration %v", child.SelfDuration)
		}
		if parent.SelfDuration < hangTestSlowParentRender {
			t.Fatalf("parent self duration %v", parent.SelfDuration)
		}
		if parentDuration < childDuration+parent.SelfDuration {
			t.Fatalf("parent %v < child %v + self %v", parentDuration, childDuration, parent.SelfDuration)
		}
		if slices.ContainsFunc(events, func(event ForegroundEvent) bool {
			timing, ok := event.(ForegroundEventView)
			return ok && timing.Phase == ViewPhasePaint
		}) {
			t.Fatal("fast paint phases aren't journaled")
		}

		var draw FrameTiming
		found := false
		for _, event := range slices.Backward(events) {
			if timing, ok := event.(ForegroundEventDraw); ok {
				draw, found = timing.FrameTiming, true
				break
			}
		}
		if !found {
			t.Fatal("a draw")
		}
		if draw.Phases.RequestLayout < hangTestSlowParentRender+hangTestSlowChildRender {
			t.Fatalf("request layout %v", draw.Phases.RequestLayout)
		}
		if draw.Phases.RequestLayout+draw.Phases.Prepaint+draw.Phases.Paint > draw.DrawDuration() {
			t.Fatalf("phases %#v exceed the draw %v", draw.Phases, draw.DrawDuration())
		}
	})
}

type hangInjection struct {
	kind     hangKind
	duration time.Duration
	start    scheduler.Instant
	end      scheduler.Instant
}

func hangAssertDetectedOnce(t *testing.T, injection hangInjection, contributors []ForegroundEvent) {
	t.Helper()
	var matches []ForegroundEvent
	for _, event := range contributors {
		if hangMatchesKind(event, injection.kind) && !event.StartTime().Before(injection.start) && !injection.end.Before(event.EndTime()) {
			matches = append(matches, event)
		}
	}
	if len(matches) != 1 {
		durations := make([]time.Duration, 0, len(matches))
		for _, event := range matches {
			durations = append(durations, event.Duration())
		}
		t.Fatalf("expected injected %v hang of %v to be reported exactly once; observed %v", injection.kind, injection.duration, durations)
	}
	if matches[0].Duration() < injection.duration {
		t.Fatalf("injected %v hang of %v was reported as %v", injection.kind, injection.duration, matches[0].Duration())
	}
}

func hangMatchesKind(event ForegroundEvent, kind hangKind) bool {
	switch timing := event.(type) {
	case ForegroundEventDraw:
		return kind == hangKindRender
	case ForegroundEventInput:
		return kind == hangKindInput
	case ForegroundEventAction:
		return kind == hangKindAction && strings.HasSuffix(timing.Name, "HangyAction")
	case ForegroundEventTaskPoll:
		return kind == hangKindPoll && strings.HasSuffix(timing.Location.File(), "profiler_hang_test.go")
	default:
		return false
	}
}

func hangDrawWindow(cx *VisualTestContext) {
	cx.Update(func(window *Window, cx *App) {
		arenaClear := window.Draw(cx)
		window.presentIfNeeded()
		arenaClear.Clear(cx)
	})
}

func hangSimulateBlockedForegroundPoll(thread ThreadID, duration time.Duration) {
	location := scheduler.CallerLocation(0)
	UpdateRunningTask(thread, scheduler.SpawnTime{Instant: scheduler.InstantNow()}, location)
	time.Sleep(duration)
	SaveTaskTiming(thread)
}

func hangTaskPollEvent(start, end scheduler.Instant) ForegroundEvent {
	return ForegroundEventTaskPoll{TaskTiming{
		Location: scheduler.CallerLocation(1),
		Spawned:  scheduler.SpawnTime{Instant: start},
		Start:    start,
		End:      YieldTime{end},
	}}
}

func hangPresentation(windowID WindowID, presentEnd scheduler.Instant) PresentTiming {
	return PresentTiming{
		WindowID:     windowID,
		PresentStart: presentEnd.Add(-time.Millisecond),
		PresentEnd:   presentEnd,
	}
}

func hangFrame(windowID WindowID, drawEnd scheduler.Instant) FrameTiming {
	return FrameTiming{
		WindowID:      windowID,
		DirtyAt:       Some(drawEnd.Add(-2 * time.Millisecond)),
		Invalidations: 1,
		DrawStart:     drawEnd.Add(-time.Millisecond),
		DrawEnd:       drawEnd,
	}
}
