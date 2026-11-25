package gpui

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

type windowTestEmptyView struct{}

func (*windowTestEmptyView) Render(_ *Window, _ *Context[windowTestEmptyView]) IntoElement {
	return NewDiv()
}

func windowTestAddEmpty(cx *TestAppContext) WindowHandle[windowTestEmptyView] {
	return cx.AddWindow(func(_ *Window, _ *Context[windowTestEmptyView]) *windowTestEmptyView {
		return &windowTestEmptyView{}
	})
}

func windowTestMust(err error) {
	if err != nil {
		panic(err)
	}
}

func windowTestLiveFocusHandles(m *focusMap) []FocusID {
	var ids []FocusID
	for idx, slot := range m.slots {
		if slot.ref != nil {
			ids = append(ids, FocusID(uint64(slot.version)<<32|uint64(idx)))
		}
	}
	return ids
}

func TestWindow_FocusHandleRelease(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var handle, survivor FocusHandle
		cx.Update(func(cx *App) { handle, survivor = cx.FocusHandle(), cx.FocusHandle() })
		handleID := handle.id
		weakHandle := handle.Downgrade()
		retained := handle.Clone()
		handle.Release()

		cx.Update(func(*App) {})
		cx.Read(func(cx *App) {
			handles := cx.focusHandles
			if n := len(windowTestLiveFocusHandles(handles)); n != 2 {
				t.Fatalf("handles = %d", n)
			}
			if len(handles.droppedHandles) != 0 {
				t.Fatalf("dropped = %v", handles.droppedHandles)
			}
		})
		upgraded, ok := weakHandle.Upgrade()
		if !ok || !upgraded.Equal(retained) {
			t.Fatal("upgrade should return the retained handle")
		}
		upgraded.Release()

		tc.BlockTest(func(s *scheduler.Scope) {
			_, _ = cx.BackgroundSpawn(func(*scheduler.Scope) (struct{}, error) {
				retained.Release()
				return struct{}{}, nil
			}).Await(s)
		})
		if _, ok := weakHandle.Upgrade(); ok {
			t.Fatal("upgrade after release")
		}
		cx.Read(func(cx *App) {
			if !slices.Equal(cx.focusHandles.droppedHandles, []FocusID{handleID}) {
				t.Fatalf("dropped = %v", cx.focusHandles.droppedHandles)
			}
		})
		cx.Update(func(*App) {})
		cx.Read(func(cx *App) {
			if ids := windowTestLiveFocusHandles(cx.focusHandles); !slices.Equal(ids, []FocusID{survivor.id}) {
				t.Fatalf("handles = %v", ids)
			}
			if len(cx.focusHandles.droppedHandles) != 0 {
				t.Fatal("dropped not empty")
			}
		})

		var replacements [2]FocusHandle
		cx.Update(func(cx *App) { replacements = [2]FocusHandle{cx.FocusHandle(), cx.FocusHandle()} })
		if _, ok := weakHandle.Upgrade(); ok {
			t.Fatal("upgrade after replacement")
		}
		cx.Update(func(cx *App) {
			for _, r := range replacements {
				r.Release()
			}
			cx.Defer(func(*App) { survivor.Release() })
		})
		cx.Read(func(cx *App) {
			if n := len(windowTestLiveFocusHandles(cx.focusHandles)); n != 0 {
				t.Fatalf("handles = %d", n)
			}
			if len(cx.focusHandles.droppedHandles) != 0 {
				t.Fatal("dropped not empty")
			}
		})
	})
}

func TestWindow_DroppedFocusHandleBlursOnlyItsWindows(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var windows [5]WindowHandle[windowTestEmptyView]
		for i := range windows {
			windows[i] = windowTestAddEmpty(cx)
		}
		var handle, otherHandle, survivor FocusHandle
		cx.Update(func(cx *App) { handle, otherHandle, survivor = cx.FocusHandle(), cx.FocusHandle(), cx.FocusHandle() })
		for _, window := range windows[:2] {
			windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, cx *Context[windowTestEmptyView]) {
				window.Focus(handle, cx.App)
			}))
		}
		windowTestMust(windows[2].Update(cx, func(_ *windowTestEmptyView, window *Window, cx *Context[windowTestEmptyView]) {
			window.Focus(otherHandle, cx.App)
		}))
		windowTestMust(windows[3].Update(cx, func(_ *windowTestEmptyView, window *Window, cx *Context[windowTestEmptyView]) {
			window.Focus(survivor, cx.App)
		}))

		cx.Update(func(cx *App) {
			for range 1_000 {
				cx.FocusHandle().Release()
			}
			handle.Release()
			otherHandle.Release()
		})
		for _, window := range windows[:3] {
			windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
				if window.focus.IsSome() {
					t.Fatal("window should be blurred")
				}
			}))
		}
		windowTestMust(windows[3].Update(cx, func(_ *windowTestEmptyView, window *Window, cx *Context[windowTestEmptyView]) {
			focused, ok := window.Focused(cx.App)
			if !ok || !focused.Equal(survivor) {
				t.Fatal("survivor should stay focused")
			}
			focused.Release()
			if window.focusGeneration != 1 {
				t.Fatalf("focus generation = %d", window.focusGeneration)
			}
		}))
		windowTestMust(windows[4].Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			if window.focus.IsSome() || window.focusGeneration != 0 {
				t.Fatal("untouched window changed")
			}
		}))
		survivor.Release()
	})
}

func TestWindow_NativeCloseDefersAppAccess(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		platformWindow := cx.testWindow(window.AnyWindowHandle)
		{
			app := cx.app.BorrowMut()
			platformWindow.SimulateClose()
			if !app.windows.containsKey(window.WindowID()) {
				t.Fatal("window removed while app borrowed")
			}
			app.Release()
		}
		cx.RunUntilParked()
		app := cx.app.Borrow()
		defer app.Release()
		if app.Get().windows.containsKey(window.WindowID()) {
			t.Fatal("window not removed")
		}
	})
}

func TestWindow_FrameRequestsAndSkipsAreJournaled(t *testing.T) {
	if !profilerFeatureEnabled {
		t.Skip("requires -tags gpui_profiler")
	}
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *App) { cx.mode = gpuiModeProduction })
		window := cx.AddWindow(func(_ *Window, _ *Context[windowTestEmptyView]) *windowTestEmptyView {
			return &windowTestEmptyView{}
		})
		platformWindow := cx.testWindow(window.AnyWindowHandle)
		platformWindow.SimulateFrameRequest(RequestFrameOptions{})
		journal, guard := installTestForegroundJournal(64, 4)
		defer guard.End()
		collector := journal.Collector()

		for _, c := range []struct {
			signalAt Option[scheduler.Instant]
			source   FrameRequestSource
		}{
			{Some(scheduler.InstantNow()), FrameRequestSourceNativeCallback},
			{Some(scheduler.InstantNow()), FrameRequestSourceLocalSchedule},
			{None[scheduler.Instant](), FrameRequestSourceNativeCallback},
		} {
			platformWindow.SimulateFrameRequest(RequestFrameOptions{
				SignalAt:            c.signalAt,
				SignalSource:        c.source,
				RequirePresentation: true,
			})
			entries := collector.CollectUnseen().Entries
			signalIndex := slices.IndexFunc(entries, func(entry ForegroundJournalEntry) bool {
				e, ok := entry.(ForegroundJournalEntryPlatformSignal)
				if !ok {
					return false
				}
				signal := e.Signal
				at, hasAt := c.signalAt.Get()
				return signal.WindowID == window.WindowID() && signal.SignalAt == c.signalAt && signal.Source == c.source && (!hasAt || !signal.HandledAt.Before(at))
			})
			if signalIndex < 0 {
				t.Fatal("frame request metadata")
			}
			skipIndex := slices.IndexFunc(entries, func(entry ForegroundJournalEntry) bool {
				e, ok := entry.(ForegroundJournalEntryBoundary)
				if !ok {
					return false
				}
				skipped, ok := e.Boundary.(IntervalBoundaryFrameSkipped)
				return ok && skipped.WindowID == window.WindowID() && skipped.Reason == FrameSkipReasonNoRenderNeeded
			})
			if skipIndex < 0 {
				t.Fatal("completed no-render boundary")
			}
			if signalIndex >= skipIndex {
				t.Fatalf("signal %d not before skip %d", signalIndex, skipIndex)
			}
		}
	})
}

func TestWindow_FrameDemandSurvivesThrottling(t *testing.T) {
	if !profilerFeatureEnabled {
		t.Skip("requires -tags gpui_profiler")
	}
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *App) { cx.mode = gpuiModeProduction })
		journal, guard := installTestForegroundJournal(256, 4)
		defer guard.End()
		collector := journal.Collector()
		for _, c := range []struct{ initiallyDirty, scheduleCallback bool }{{true, false}, {false, true}} {
			var window WindowHandle[windowTestEmptyView]
			cx.Update(func(cx *App) {
				options := DefaultWindowOptions()
				options.InactiveFrameInterval = Some(time.Hour)
				handle, err := cx.OpenWindow(options, func(_ *Window, cx *App) Entity[windowTestEmptyView] {
					return cx.New(func(*Context[windowTestEmptyView]) *windowTestEmptyView { return &windowTestEmptyView{} })
				})
				if err != nil {
					t.Fatalf("open inactive test window: %v", err)
				}
				window = handle
			})
			platformWindow := cx.testWindow(window.AnyWindowHandle)
			platformWindow.simulateActiveStatusChange(false)
			platformWindow.SimulateFrameRequest(RequestFrameOptions{})
			callbackRan := false
			var drawsBefore uint64
			windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
				drawsBefore = window.FrameDurationSnapshot().DrawDurationHistogram.Len()
				if c.initiallyDirty {
					window.Refresh()
				}
				if c.scheduleCallback {
					window.OnNextFrame(func(window *Window, _ *App) {
						callbackRan = true
						window.Refresh()
					})
				}
			}))
			inputAt := scheduler.InstantNow()
			recordInput(InputTiming{Kind: "test", Start: inputAt, End: inputAt, CausedInvalidation: false})
			collector.CollectUnseen()
			wakesBefore := platformWindow.FrameWakeCount()
			platformWindow.SimulateFrameRequest(RequestFrameOptions{SignalAt: Some(scheduler.InstantNow())})
			windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
				if got := window.FrameDurationSnapshot().DrawDurationHistogram.Len(); got != drawsBefore {
					t.Fatalf("deferred frame drew: %d != %d", got, drawsBefore)
				}
			}))
			if callbackRan {
				t.Fatal("callback ran while throttled")
			}
			if !platformWindow.FrameScheduled() {
				t.Fatal("frame not scheduled")
			}
			if platformWindow.FrameWakeCount() <= wakesBefore {
				t.Fatal("no frame wake")
			}
			if !slices.ContainsFunc(collector.CollectUnseen().Entries, func(entry ForegroundJournalEntry) bool {
				e, ok := entry.(ForegroundJournalEntryBoundary)
				if !ok {
					return false
				}
				skipped, ok := e.Boundary.(IntervalBoundaryFrameSkipped)
				return ok && skipped.WindowID == window.WindowID() && skipped.Reason == FrameSkipReasonInactiveFrameRateLimit
			}) {
				t.Fatal("no inactive frame rate limit skip")
			}
			platformWindow.SimulateFrameRequest(RequestFrameOptions{SignalAt: Some(scheduler.InstantNow()), RequirePresentation: true})
			windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
				if got := window.FrameDurationSnapshot().DrawDurationHistogram.Len(); got != drawsBefore+1 {
					t.Fatalf("presented frame: %d != %d", got, drawsBefore+1)
				}
			}))
			if callbackRan != c.scheduleCallback {
				t.Fatalf("callbackRan = %v", callbackRan)
			}
		}
	})
}

func TestWindow_FrameThrottleTracksDisplayRefreshes(t *testing.T) {
	start := scheduler.InstantNow()
	for _, c := range []struct{ elapsed, interval time.Duration }{
		{16_666 * time.Microsecond, 16_667 * time.Microsecond},
		{33_332 * time.Microsecond, 33_333 * time.Microsecond},
	} {
		var pacer framePacer
		if pacer.shouldDefer(start, Some(c.interval)) {
			t.Fatal("first frame deferred")
		}
		if !pacer.shouldDefer(start.Add(c.elapsed/2), Some(c.interval)) {
			t.Fatal("half interval not deferred")
		}
		if pacer.shouldDefer(start.Add(c.elapsed), Some(c.interval)) {
			t.Fatal("full interval deferred")
		}
	}
}

func TestWindow_FrameThrottlePreservesCadenceWithoutCatchingUpAfterStalls(t *testing.T) {
	start := scheduler.InstantNow()
	interval := 33_333 * time.Microsecond
	var pacer framePacer
	check := func(at scheduler.Instant, limit Option[time.Duration], want bool) {
		t.Helper()
		if got := pacer.shouldDefer(at, limit); got != want {
			t.Fatalf("shouldDefer(%v) = %v", at, got)
		}
	}
	check(start, Some(interval), false)
	delayedFrame := start.Add(interval + 5*time.Millisecond)
	check(delayedFrame, Some(interval), false)
	check(start.Add(interval+interval/2), Some(interval), true)
	check(start.Add(interval*2), Some(interval), false)

	afterStall := start.Add(interval*4 - time.Millisecond)
	check(afterStall, Some(interval), false)
	check(afterStall.Add(interval/2), Some(interval), true)
	check(afterStall.Add(interval), Some(interval), false)
	mandatoryFrame := afterStall.Add(interval + 2*time.Millisecond)
	check(mandatoryFrame, None[time.Duration](), false)
	check(mandatoryFrame.Add(interval/2), Some(interval), true)
}

func TestWindow_PowerModesPreserveFrameLimits(t *testing.T) {
	options := DefaultWindowOptions()
	for _, c := range []struct {
		powerMode       PowerMode
		activeInterval  Option[time.Duration]
		thermalInterval time.Duration
	}{
		{PowerModeLowPower, Some(33_333 * time.Microsecond), 33_333 * time.Microsecond},
		{PowerModeUnknown, None[time.Duration](), 16_667 * time.Microsecond},
		{PowerModeBalanced, None[time.Duration](), 16_667 * time.Microsecond},
		{PowerModeHighPower, None[time.Duration](), 16_667 * time.Microsecond},
	} {
		if got := minimumFrameInterval(true, false, options.InactiveFrameInterval, options.LowPowerFrameInterval, options.ThermalFrameInterval, ThermalStateNominal, c.powerMode); got != c.activeInterval {
			t.Fatalf("%v nominal: %v", c.powerMode, got)
		}
		if got := minimumFrameInterval(true, false, options.InactiveFrameInterval, options.LowPowerFrameInterval, options.ThermalFrameInterval, ThermalStateSerious, c.powerMode); got != Some(c.thermalInterval) {
			t.Fatalf("%v serious: %v", c.powerMode, got)
		}
	}
}

func TestWindow_FrameLimitsCombineEnabledIntervals(t *testing.T) {
	short := Some(10 * time.Millisecond)
	long := Some(100 * time.Millisecond)
	none := None[time.Duration]()
	for _, c := range [][4]Option[time.Duration]{
		{long, short, short, long},
		{short, long, short, long},
		{short, short, long, long},
		{long, none, none, long},
		{none, long, none, long},
		{none, none, long, long},
		{none, none, none, none},
	} {
		if got := minimumFrameInterval(true, true, c[0], c[1], c[2], ThermalStateCritical, PowerModeLowPower); got != c[3] {
			t.Fatalf("%v: %v", c, got)
		}
	}
	if got := minimumFrameInterval(false, true, long, long, long, ThermalStateCritical, PowerModeLowPower); got != none {
		t.Fatalf("unthrottled: %v", got)
	}
}

type windowTestCanvas struct {
	sizeFull bool
	prepaint func(bounds Bounds[Pixels], window *Window, cx *App)
	paint    func(bounds Bounds[Pixels], window *Window, cx *App)
}

func (c *windowTestCanvas) ElementID() (ElementID, bool) { return ElementID{}, false }

func (c *windowTestCanvas) SourceLocation() *SourceLocation { return nil }

func (c *windowTestCanvas) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	style := DefaultStyle()
	if c.sizeFull {
		style.Size = SizeOf[Length]{Width: Relative(1.).ToLength(), Height: Relative(1.).ToLength()}
	}
	return window.RequestLayout(style, nil, cx), struct{}{}
}

func (c *windowTestCanvas) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, window *Window, cx *App) struct{} {
	if c.prepaint != nil {
		c.prepaint(bounds, window, cx)
	}
	return struct{}{}
}

func (c *windowTestCanvas) Paint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, _ *struct{}, window *Window, cx *App) {
	if c.paint != nil {
		c.paint(bounds, window, cx)
	}
}

func (c *windowTestCanvas) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](c)
}

type windowTestPlainEmpty struct{}

func (*windowTestPlainEmpty) Render(_ *Window, _ *Context[windowTestPlainEmpty]) IntoElement {
	return Empty{}
}

func windowTestDebugWindowState(cx *TestAppContext, windowID uint64) map[string]any {
	var raw jsontext.Value
	cx.Update(func(cx *App) { raw = cx.DebugState() })
	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		panic(err)
	}
	for _, window := range state["windows"].([]any) {
		w := window.(map[string]any)
		if uint64(w["id"].(float64)) == windowID {
			return w
		}
	}
	panic("the open window is listed")
}

func TestWindow_DebugStateReportsCachedAndPlatformValues(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		windowID := window.WindowID().AsU64()
		visibility := func() map[string]any {
			return windowTestDebugWindowState(cx, windowID)["visibility"].(map[string]any)
		}
		state := visibility()
		if state["cached"] != "Visible" || state["platform"] != "Visible" {
			t.Fatalf("visibility = %v", state)
		}
		cx.testWindow(window.AnyWindowHandle).SimulateVisibilityChange(WindowVisibilityHidden)
		state = visibility()
		if state["cached"] != "Hidden" || state["platform"] != "Hidden" {
			t.Fatalf("visibility = %v", state)
		}
	})
}

func TestWindow_WindowVisibility(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		var observed []WindowVisibility
		subscription, err := window.UpdateR(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) Subscription {
			if window.Visibility() != WindowVisibilityVisible || !window.IsVisible() {
				t.Fatal("window should start visible")
			}
			return window.ObserveWindowVisibility(func(visibility WindowVisibility, window *Window, _ *App) {
				if window.Visibility() != visibility {
					t.Fatal("visibility not stored before observers run")
				}
				observed = append(observed, visibility)
			})
		})
		windowTestMust(err)
		defer subscription.Unsubscribe()
		testWindow := cx.testWindow(window.AnyWindowHandle)
		frameWakeCount := testWindow.FrameWakeCount()

		testWindow.SimulateVisibilityChange(WindowVisibilityHidden)
		if !slices.Equal(observed, []WindowVisibility{WindowVisibilityHidden}) {
			t.Fatalf("observed = %v", observed)
		}
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			if window.IsVisible() {
				t.Fatal("window should be hidden")
			}
		}))

		testWindow.SimulateVisibilityChange(WindowVisibilityHidden)
		if len(observed) != 1 {
			t.Fatalf("observed = %v", observed)
		}

		testWindow.SimulateVisibilityChange(WindowVisibilityVisible)
		if !slices.Equal(observed, []WindowVisibility{WindowVisibilityHidden, WindowVisibilityVisible}) {
			t.Fatalf("observed = %v", observed)
		}
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			if !window.IsVisible() {
				t.Fatal("window should be visible")
			}
		}))
		if testWindow.FrameWakeCount() != frameWakeCount {
			t.Fatal("visibility change woke the frame source")
		}
	})
}

type windowTestDisplayTracker struct {
	refreshIntervals []Option[time.Duration]
	subscription     Subscription
}

func (*windowTestDisplayTracker) Render(_ *Window, _ *Context[windowTestDisplayTracker]) IntoElement {
	return Empty{}
}

type windowTestDisplayNotification struct {
	displayID       Option[DisplayID]
	refreshInterval Option[time.Duration]
}

func TestWindow_DisplayTracking(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		sixtyHertz := time.Second / 60
		oneHundredTwentyHertz := time.Second / 120
		oneHundredFortyFourHertz := time.Second / 144

		var displayEvents []DisplayEvent
		var displaySubscription Subscription
		cx.Update(func(cx *App) {
			displaySubscription = cx.ObserveDisplays(func(event DisplayEvent, _ *App) { displayEvents = append(displayEvents, event) })
		})
		defer displaySubscription.Unsubscribe()

		window := cx.AddWindow(func(window *Window, cx *Context[windowTestDisplayTracker]) *windowTestDisplayTracker {
			return &windowTestDisplayTracker{subscription: cx.ObserveWindowDisplay(window, func(tracker *windowTestDisplayTracker, window *Window, cx *Context[windowTestDisplayTracker]) {
				tracker.refreshIntervals = append(tracker.refreshIntervals, OptionFrom(window.RefreshInterval(cx.App)))
			})}
		})
		defer func() {
			_ = window.Update(cx, func(tracker *windowTestDisplayTracker, _ *Window, _ *Context[windowTestDisplayTracker]) {
				tracker.subscription.Unsubscribe()
			})
		}()
		var windowNotifications []windowTestDisplayNotification
		windowSubscription, err := window.UpdateR(cx, func(_ *windowTestDisplayTracker, window *Window, cx *Context[windowTestDisplayTracker]) Subscription {
			if got := OptionFrom(window.RefreshInterval(cx.App)); got != Some(sixtyHertz) {
				t.Fatalf("refresh interval = %v", got)
			}
			return window.ObserveWindowDisplay(func(window *Window, cx *App) {
				windowNotifications = append(windowNotifications, windowTestDisplayNotification{window.displayID, OptionFrom(window.RefreshInterval(cx))})
			})
		})
		windowTestMust(err)
		defer windowSubscription.Unsubscribe()
		testWindow := cx.testWindow(window.AnyWindowHandle)
		takeNotifications := func() []windowTestDisplayNotification {
			n := windowNotifications
			windowNotifications = nil
			return n
		}
		takeEvents := func() []DisplayEvent {
			e := displayEvents
			displayEvents = nil
			return e
		}

		cx.testPlatform.ConnectDisplay(DisplayID(2), Some(oneHundredTwentyHertz))
		testWindow.SimulateMoveToDisplay(DisplayID(2))
		if got := takeNotifications(); !slices.Equal(got, []windowTestDisplayNotification{{Some(DisplayID(2)), Some(oneHundredTwentyHertz)}}) {
			t.Fatalf("notifications = %v", got)
		}
		if got := takeEvents(); !slices.Equal(got, []DisplayEvent{DisplayEventAdded{ID: DisplayID(2)}}) {
			t.Fatalf("events = %v", got)
		}
		cx.SimulateDisplayRefreshIntervalChange(DisplayID(2), Some(oneHundredTwentyHertz))
		if len(windowNotifications) != 0 {
			t.Fatal("unchanged refresh interval notified")
		}

		testWindow.SimulateMoveToDisplay(DisplayID(2))
		if len(windowNotifications) != 0 {
			t.Fatal("move that isn't one notified")
		}

		cx.SimulateDisplayRefreshIntervalChange(DisplayID(2), Some(oneHundredFortyFourHertz))
		if got := takeNotifications(); !slices.Equal(got, []windowTestDisplayNotification{{Some(DisplayID(2)), Some(oneHundredFortyFourHertz)}}) {
			t.Fatalf("notifications = %v", got)
		}

		cx.SimulateDisplayRefreshIntervalChange(DisplayID(1), Some(oneHundredTwentyHertz))
		cx.SimulateDisplayRemoved(DisplayID(1))
		if len(windowNotifications) != 0 {
			t.Fatal("other display notified the window")
		}

		if got := takeEvents(); !slices.Equal(got, []DisplayEvent{
			DisplayEventChanged{ID: DisplayID(2), Changes: DisplayChangesRefreshInterval},
			DisplayEventChanged{ID: DisplayID(1), Changes: DisplayChangesRefreshInterval},
			DisplayEventRemoved{ID: DisplayID(1)},
		}) {
			t.Fatalf("events = %v", got)
		}
		windowTestMust(window.Update(cx, func(tracker *windowTestDisplayTracker, _ *Window, _ *Context[windowTestDisplayTracker]) {
			if !slices.Equal(tracker.refreshIntervals, []Option[time.Duration]{Some(oneHundredTwentyHertz), Some(oneHundredFortyFourHertz)}) {
				t.Fatalf("refresh intervals = %v", tracker.refreshIntervals)
			}
		}))
	})
}

type windowTestBoundsObserver struct {
	subscription Subscription
}

func (*windowTestBoundsObserver) Render(_ *Window, _ *Context[windowTestBoundsObserver]) IntoElement {
	return Empty{}
}

func TestWindow_WindowOpenedFromBoundsObserverUsesNewBounds(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		openedWindow := None[AnyWindowHandle]()
		window := cx.AddWindow(func(window *Window, cx *Context[windowTestBoundsObserver]) *windowTestBoundsObserver {
			return &windowTestBoundsObserver{subscription: cx.ObserveWindowBounds(window, func(_ *windowTestBoundsObserver, _ *Window, cx *Context[windowTestBoundsObserver]) {
				handle, err := cx.OpenWindow(DefaultWindowOptions(), func(_ *Window, cx *App) Entity[windowTestEmptyView] {
					return cx.New(func(*Context[windowTestEmptyView]) *windowTestEmptyView { return &windowTestEmptyView{} })
				})
				windowTestMust(err)
				openedWindow = Some(handle.AnyWindowHandle)
			})}
		})
		defer func() {
			_ = window.Update(cx, func(v *windowTestBoundsObserver, _ *Window, _ *Context[windowTestBoundsObserver]) {
				v.subscription.Unsubscribe()
			})
		}()
		windowTestMust(window.Update(cx, func(_ *windowTestBoundsObserver, window *Window, _ *Context[windowTestBoundsObserver]) {
			window.ActivateWindow()
		}))

		newSize := NewSize(Px(400.), Px(300.))
		cx.SimulateWindowResize(window.AnyWindowHandle, newSize)

		opened := openedWindow.Take().Expect("bounds observer should have opened a window")
		var openedSize Size[Pixels]
		windowTestMust(opened.Update(cx, func(_ AnyView, window *Window, _ *App) { openedSize = window.Bounds().Size }))
		if openedSize != newSize {
			t.Fatalf("opened size = %v", openedSize)
		}
	})
}

func TestWindow_FullyVisibleBoundsPreserveLayoutViewport(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		platformWindow := cx.testWindow(window.AnyWindowHandle)
		platformWindow.SimulateResize(NewSize(Px(400.), Px(800.)))
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			if window.VisualViewportBounds() != NewBounds(Point[Pixels]{}, NewSize(Px(400.), Px(800.))) {
				t.Fatalf("visual viewport = %v", window.VisualViewportBounds())
			}
			if window.FullyVisibleBounds() != window.VisualViewportBounds() {
				t.Fatalf("fully visible = %v", window.FullyVisibleBounds())
			}
		}))

		platformWindow.SimulateFrameRequest(RequestFrameOptions{})
		wakes := platformWindow.FrameWakeCount()
		visualBounds := NewBounds(NewPoint(Px(10.), Px(40.)), NewSize(Px(380.), Px(460.)))
		platformWindow.SimulateVisualViewportChange(visualBounds)
		if platformWindow.FrameWakeCount() <= wakes {
			t.Fatal("visual viewport change did not wake")
		}
		platformWindow.SimulateFrameRequest(RequestFrameOptions{})
		wakes = platformWindow.FrameWakeCount()
		insets := WindowInsets{
			SafeArea: Edges[Pixels]{Top: Px(60.), Right: Px(20.), Bottom: Px(30.), Left: Px(-10.)},
			IME:      Edges[Pixels]{Bottom: Px(350.)},
		}
		platformWindow.SimulateInsetsChange(insets)
		if platformWindow.FrameWakeCount() <= wakes {
			t.Fatal("insets change did not wake")
		}
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, cx *Context[windowTestEmptyView]) {
			focus := cx.FocusHandle()
			defer focus.Release()
			window.Focus(focus, cx.App)
			if window.Insets() != insets {
				t.Fatalf("insets = %v", window.Insets())
			}
			if window.ViewportSize() != NewSize(Px(400.), Px(800.)) {
				t.Fatalf("viewport = %v", window.ViewportSize())
			}
			if window.VisualViewportBounds() != visualBounds {
				t.Fatalf("visual viewport = %v", window.VisualViewportBounds())
			}
			if got := window.FullyVisibleBounds(); got != NewBounds(NewPoint(Px(10.), Px(60.)), NewSize(Px(370.), Px(390.))) {
				t.Fatalf("fully visible = %v", got)
			}
			window.RequestVirtualKeyboard()
			window.DismissVirtualKeyboard()
			if !focus.IsFocused(window) {
				t.Fatal("focus lost")
			}
			if window.ViewportSize() != NewSize(Px(400.), Px(800.)) {
				t.Fatalf("viewport = %v", window.ViewportSize())
			}
		}))
		if platformWindow.VirtualKeyboardRequests() != 1 || platformWindow.VirtualKeyboardDismissals() != 1 {
			t.Fatal("virtual keyboard counts")
		}

		platformWindow.SimulateInsetsChange(WindowInsets{SafeArea: Edges[Pixels]{Top: Px(900.), Left: Px(500.)}})
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			if got := window.FullyVisibleBounds().Size; got != NewSize(Pixels(0), Pixels(0)) {
				t.Fatalf("fully visible size = %v", got)
			}
		}))

		platformWindow.OnInsetsChanged(func(WindowInsets) {})
		platformWindow.SimulateInsetsChange(WindowInsets{})
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			if window.Insets() != (WindowInsets{}) {
				t.Fatalf("insets = %v", window.Insets())
			}
		}))
	})
}

type windowTestOpensWindowOnPaint struct {
	opened *bool
}

func (v *windowTestOpensWindowOnPaint) Render(_ *Window, _ *Context[windowTestOpensWindowOnPaint]) IntoElement {
	opened := v.opened
	return NewDiv().SizeFull().
		Child(&windowTestCanvas{paint: func(_ Bounds[Pixels], _ *Window, cx *App) {
			if !*opened {
				*opened = true
				_, err := cx.OpenWindow(DefaultWindowOptions(), func(_ *Window, cx *App) Entity[windowTestEmptyView] {
					return cx.New(func(*Context[windowTestEmptyView]) *windowTestEmptyView { return &windowTestEmptyView{} })
				})
				windowTestMust(err)
			}
		}}).
		Child(NewDiv().Child(SharedString("after")))
}

func windowTestSingle(t *testing.T) *TestAppContext {
	cx := TestAppContextSingle()
	t.Cleanup(func() {
		cx.Close()
	})
	return cx
}

func TestWindow_WindowOpenedDuringDrawDefersArenaClear(t *testing.T) {
	cx := windowTestSingle(t)
	opened := false
	window := cx.AddWindow(func(_ *Window, _ *Context[windowTestOpensWindowOnPaint]) *windowTestOpensWindowOnPaint {
		return &windowTestOpensWindowOnPaint{opened: &opened}
	})
	if !opened {
		t.Fatal("nested window not opened")
	}
	if len(cx.Windows()) != 2 {
		t.Fatalf("windows = %d", len(cx.Windows()))
	}
	windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) { window.Draw(cx).Clear(cx) }))
}

func TestWindow_ScaleFactorChangePreservesBoundsAndSurvivesResize(t *testing.T) {
	cx := windowTestSingle(t)
	window := windowTestAddEmpty(cx)
	handle := window.AnyWindowHandle
	type state struct {
		scaleFactor float32
		bounds      Bounds[Pixels]
		viewport    Size[Pixels]
	}
	windowState := func() state {
		var s state
		windowTestMust(cx.updateWindow(handle, func(_ AnyView, window *Window, _ *App) {
			s = state{window.ScaleFactor(), window.Bounds(), window.ViewportSize()}
		}))
		return s
	}
	initial := windowState()
	if initial.scaleFactor != 2.0 {
		t.Fatalf("scale factor = %v", initial.scaleFactor)
	}
	expectedBounds := initial.bounds
	for _, c := range []struct {
		scaleFactor float32
		resized     Size[Pixels]
	}{
		{1.0, NewSize(Px(800.), Px(600.))},
		{1.25, NewSize(Px(640.), Px(480.))},
		{2.0, NewSize(Px(1024.), Px(768.))},
	} {
		cx.SimulateWindowScaleFactorChange(handle, c.scaleFactor)
		if got := windowState(); got != (state{c.scaleFactor, expectedBounds, expectedBounds.Size}) {
			t.Fatalf("after scale: %v", got)
		}
		cx.SimulateWindowResize(handle, c.resized)
		expectedBounds.Size = c.resized
		if got := windowState(); got != (state{c.scaleFactor, expectedBounds, c.resized}) {
			t.Fatalf("after resize: %v", got)
		}
	}
}

func TestWindow_FullscreenToggleReportsCurrentBounds(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		handle := window.AnyWindowHandle
		notifications := 0
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, cx *Context[windowTestEmptyView]) {
			cx.ObserveWindowBounds(window, func(*windowTestEmptyView, *Window, *Context[windowTestEmptyView]) { notifications++ }).Detach()
		}))
		type state struct {
			fullscreen bool
			size       Size[Pixels]
			viewport   Size[Pixels]
		}
		windowState := func() state {
			var s state
			windowTestMust(cx.updateWindow(handle, func(_ AnyView, window *Window, _ *App) {
				s = state{window.IsFullscreen(), window.Bounds().Size, window.ViewportSize()}
			}))
			return s
		}
		initialSize := windowState().size

		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			window.ToggleFullscreen()
		}))
		cx.RunUntilParked()
		if got := windowState(); got != (state{true, initialSize, initialSize}) {
			t.Fatalf("fullscreen state = %v", got)
		}
		if notifications != 1 {
			t.Fatalf("notifications = %d", notifications)
		}

		resizedSize := NewSize(Px(800.), Px(600.))
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			window.ToggleFullscreen()
		}))
		cx.SimulateWindowResize(handle, resizedSize)
		cx.RunUntilParked()
		if got := windowState(); got != (state{false, resizedSize, resizedSize}) {
			t.Fatalf("windowed state = %v", got)
		}
	})
}

func TestWindow_FrameWakerFiresOnFrameDemand(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		testWindow := cx.testWindow(window.AnyWindowHandle)
		testWindow.simulateActiveStatusChange(true)

		if testWindow.FrameWakeCount() < 1 {
			t.Fatal("opening a window must wake the frame source for the initial frame")
		}
		testWindow.SimulateFrameRequest(RequestFrameOptions{})

		baseline := testWindow.FrameWakeCount()
		testWindow.SimulateFrameRequest(RequestFrameOptions{})
		windowTestMust(window.Update(cx, func(*windowTestEmptyView, *Window, *Context[windowTestEmptyView]) {}))
		if testWindow.FrameWakeCount() != baseline {
			t.Fatal("clean frames and non-notifying updates must not wake the frame source")
		}

		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, _ *Window, cx *Context[windowTestEmptyView]) { cx.Notify() }))
		if testWindow.FrameWakeCount() <= baseline {
			t.Fatal("notifying a view in an idle window must wake the frame source")
		}

		testWindow.SimulateFrameRequest(RequestFrameOptions{})
		baseline = testWindow.FrameWakeCount()
		testWindow.SimulateFrameRequest(RequestFrameOptions{})
		if testWindow.FrameWakeCount() != baseline {
			t.Fatal("serving demand must return the window to idle")
		}

		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			window.OnNextFrame(func(*Window, *App) {})
		}))
		if testWindow.FrameWakeCount() <= baseline {
			t.Fatal("scheduling a next-frame callback in an idle window must wake the frame source")
		}
	})
}

func TestWindow_PendingNextFrameCallbacksAreNotStranded(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		testWindow := cx.testWindow(window.AnyWindowHandle)
		testWindow.SimulateFrameRequest(RequestFrameOptions{})

		callbackRan := false
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			window.OnNextFrame(func(*Window, *App) { callbackRan = true })
		}))

		baseline := testWindow.FrameWakeCount()
		testWindow.SimulateFrameRequest(RequestFrameOptions{})
		if !(testWindow.FrameWakeCount() > baseline || callbackRan) {
			t.Fatal("a frame request with pending next-frame callbacks must either run them or re-arm the frame source")
		}
	})
}

func TestWindow_WindowReportsNoRawHandleInsteadOfPanicking(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		windowTestMust(window.Update(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) {
			if kind := window.RawWindowHandle().Kind; kind != RawWindowHandleNone {
				t.Fatalf("raw handle kind = %v", kind)
			}
		}))
	})
}

func TestWindow_AppearanceChangeRunsAfterAppUpdate(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddEmpty(cx)
		observedAppearance := None[WindowAppearance]()
		subscription, err := window.UpdateR(cx, func(_ *windowTestEmptyView, window *Window, _ *Context[windowTestEmptyView]) Subscription {
			return window.ObserveWindowAppearance(func(window *Window, _ *App) { observedAppearance = Some(window.Appearance()) })
		})
		windowTestMust(err)
		defer subscription.Unsubscribe()
		testWindow := cx.testWindow(window.AnyWindowHandle)

		cx.Update(func(*App) {
			testWindow.SimulateAppearanceChange(WindowAppearanceDark)
			if observedAppearance.IsSome() {
				t.Fatal("appearance observer ran inside the app update")
			}
		})
		cx.RunUntilParked()
		if observedAppearance != Some(WindowAppearanceDark) {
			t.Fatalf("observed = %v", observedAppearance)
		}
	})
}

func windowTestAddPlainEmpty(cx *TestAppContext) WindowHandle[windowTestPlainEmpty] {
	return cx.AddWindow(func(*Window, *Context[windowTestPlainEmpty]) *windowTestPlainEmpty { return &windowTestPlainEmpty{} })
}

func TestWindow_QueuedFrameCallbackWakesAParkedRenderLoop(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddPlainEmpty(cx)
		testWindow := cx.testWindow(window.AnyWindowHandle)

		if !testWindow.SimulateScheduledFrame() || !testWindow.SimulateScheduledFrame() || testWindow.FrameScheduled() {
			t.Fatal("render loop should park")
		}
		windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, _ *App) {
			*window.active = true
			window.OnNextFrame(func(*Window, *App) {})
		}))
		if !testWindow.FrameScheduled() {
			t.Fatal("queuing work on a parked window must wake the render loop")
		}
		if !testWindow.SimulateScheduledFrame() {
			t.Fatal("frame")
		}
		if !testWindow.FrameScheduled() {
			t.Fatal("presenting the frame must await one compositor callback")
		}
		if !testWindow.SimulateScheduledFrame() || testWindow.FrameScheduled() {
			t.Fatal("render loop should park again")
		}
	})
}

func TestWindow_PendingPresentationWakesAParkedRenderLoop(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddPlainEmpty(cx)
		testWindow := cx.testWindow(window.AnyWindowHandle)

		if !testWindow.SimulateScheduledFrame() || !testWindow.SimulateScheduledFrame() || testWindow.FrameScheduled() {
			t.Fatal("render loop should park")
		}
		windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) { window.Draw(cx).Clear(cx) }))
		if !testWindow.FrameScheduled() {
			t.Fatal("a rendered scene awaiting presentation must wake the render loop")
		}
	})
}

func TestWindow_CallbackQueuedDuringAFrameRequestsAFollowUp(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := windowTestAddPlainEmpty(cx)
		testWindow := cx.testWindow(window.AnyWindowHandle)

		callbackRan := false
		windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, _ *App) {
			*window.active = true
			window.OnNextFrame(func(window *Window, _ *App) {
				window.OnNextFrame(func(*Window, *App) { callbackRan = true })
			})
		}))

		if !testWindow.SimulateScheduledFrame() || callbackRan {
			t.Fatal("first frame")
		}
		if !testWindow.FrameScheduled() {
			t.Fatal("a callback queued mid-frame must schedule a follow-up before the loop parks")
		}
		if !testWindow.SimulateScheduledFrame() || !callbackRan {
			t.Fatal("follow-up frame")
		}
	})
}

type windowTestRootView struct {
	explicitSize bool
	childBounds  *Bounds[Pixels]
}

func (v *windowTestRootView) Render(_ *Window, _ *Context[windowTestRootView]) IntoElement {
	childBounds := v.childBounds
	root := NewDiv().Flex().FlexCol().Child(&windowTestCanvas{sizeFull: true, prepaint: func(bounds Bounds[Pixels], _ *Window, _ *App) { *childBounds = bounds }})
	if v.explicitSize {
		return root.W(Px(300.).ToLength()).H(Px(200.).ToLength())
	}
	return root
}

func TestWindow_AutoSizedWindowRootFillsTheWindow(t *testing.T) {
	cx := windowTestSingle(t)
	var childBounds Bounds[Pixels]
	window := cx.AddWindow(func(*Window, *Context[windowTestRootView]) *windowTestRootView {
		return &windowTestRootView{childBounds: &childBounds}
	})
	var viewportSize Size[Pixels]
	windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) {
		window.Draw(cx).Clear(cx)
		viewportSize = window.ViewportSize()
	}))
	if childBounds.Size != viewportSize {
		t.Fatalf("child size %v, viewport %v", childBounds.Size, viewportSize)
	}
}

func TestWindow_ExplicitlySizedWindowRootKeepsItsSize(t *testing.T) {
	cx := windowTestSingle(t)
	var childBounds Bounds[Pixels]
	window := cx.AddWindow(func(*Window, *Context[windowTestRootView]) *windowTestRootView {
		return &windowTestRootView{explicitSize: true, childBounds: &childBounds}
	})
	windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) { window.Draw(cx).Clear(cx) }))
	if childBounds.Size != NewSize(Px(300.), Px(200.)) {
		t.Fatalf("child size %v", childBounds.Size)
	}
}

type windowTestFileDragView struct {
	path              string
	observedDragMoves *[]Point[Pixels]
	observedDrops     *[]string
}

type windowTestFileDropExitView struct {
	count *int
}

func (v *windowTestFileDropExitView) Render(_ *Window, _ *Context[windowTestFileDropExitView]) IntoElement {
	count := v.count
	return NewDiv().SizeFull().OnFileDropExit(func(FileDropEvent, *Window, *App) { *count++ })
}

type windowTestEmptyEntity struct{}

func (*windowTestEmptyEntity) Render(_ *Window, _ *Context[windowTestEmptyEntity]) IntoElement {
	return Empty{}
}

func windowTestNewEmpty(cx *App) Entity[windowTestEmptyEntity] {
	return cx.New(func(*Context[windowTestEmptyEntity]) *windowTestEmptyEntity { return &windowTestEmptyEntity{} })
}

func (v *windowTestFileDragView) Render(_ *Window, _ *Context[windowTestFileDragView]) IntoElement {
	moves, drops := v.observedDragMoves, v.observedDrops
	d := NewDiv().ID("file-drag").SizeFull()
	d = d.OnDrag(v.path, func(_ string, _ Point[Pixels], _ *Window, cx *App) Entity[windowTestEmptyEntity] {
		return windowTestNewEmpty(cx)
	})
	d = d.ExternalDragPayload(func(path string, _ *Window, _ *App) (ExternalDragPayload, bool) {
		return ExternalDragPayloadFiles{NewFileDragPaths([]FileDragPath{{Path: path, IsDir: true}})}, true
	})
	d = d.OnDragMove(func(event *DragMoveEvent[string], _ *Window, _ *App) {
		*moves = append(*moves, event.Event.Position)
	})
	return d.OnDrop(func(path string, _ *Window, _ *App) { *drops = append(*drops, path) })
}

type windowTestDrag struct {
	window            AnyWindowHandle
	observedDragMoves *[]Point[Pixels]
	observedDrops     *[]string
}

func windowTestActiveDragIs[T any](cx *App) bool {
	drag, ok := cx.activeDrag.Get()
	if !ok {
		return false
	}
	_, ok = drag.Value.(T)
	return ok
}

func TestWindow_FileDragIsPromotedOnceAndRestoredInSourceWindow(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		startDrag := func(path string, platformResult bool) windowTestDrag {
			moves := &[]Point[Pixels]{}
			drops := &[]string{}
			window := cx.AddWindow(func(*Window, *Context[windowTestFileDragView]) *windowTestFileDragView {
				return &windowTestFileDragView{path: path, observedDragMoves: moves, observedDrops: drops}
			}).AnyWindowHandle
			cx.testWindow(window).SetStartExternalDragResult(platformResult)
			windowTestMust(cx.updateWindow(window, func(_ AnyView, window *Window, cx *App) {
				window.Draw(cx).Clear(cx)
				window.DispatchEvent(MouseDownEvent{Position: NewPoint(Px(10.), Px(10.)), Button: MouseButtonLeft, ClickCount: 1}, cx)
				window.DispatchEvent(MouseMoveEvent{Position: NewPoint(Px(20.), Px(20.)), PressedButton: Some(MouseButtonLeft)}, cx)
				if cx.activeDrag.IsNone() {
					t.Fatal("failed to start drag")
				}
			}))
			if len(cx.testWindow(window).ExternalDragFiles()) != 0 {
				t.Fatal("external drag started early")
			}
			return windowTestDrag{window, moves, drops}
		}
		lastMove := func(d windowTestDrag) Point[Pixels] {
			moves := *d.observedDragMoves
			if len(moves) == 0 {
				t.Fatal("no drag moves")
			}
			return moves[len(moves)-1]
		}

		successfulPath := "/tmp/successful-drag"
		successful := startDrag(successfulPath, true)
		outsidePosition := NewPoint(Px(-1.), Px(20.))
		windowTestMust(cx.updateWindow(successful.window, func(_ AnyView, window *Window, cx *App) {
			window.DispatchEvent(MouseMoveEvent{Position: outsidePosition, PressedButton: Some(MouseButtonLeft)}, cx)
			if cx.activeDrag.IsSome() {
				t.Fatal("drag not promoted")
			}
		}))
		if got := cx.testWindow(successful.window).ExternalDragFiles(); !slices.Equal(got, []FileDragPath{{Path: successfulPath, IsDir: true}}) {
			t.Fatalf("external drag files = %v", got)
		}
		if lastMove(successful) != outsidePosition {
			t.Fatal("views must see the move that leaves the window")
		}

		firstExitCount, secondExitCount := 0, 0
		firstDestination := cx.AddWindow(func(*Window, *Context[windowTestFileDropExitView]) *windowTestFileDropExitView {
			return &windowTestFileDropExitView{count: &firstExitCount}
		}).AnyWindowHandle
		secondDestination := cx.AddWindow(func(*Window, *Context[windowTestFileDropExitView]) *windowTestFileDropExitView {
			return &windowTestFileDropExitView{count: &secondExitCount}
		}).AnyWindowHandle
		reentryPosition := NewPoint(Px(30.), Px(30.))
		externalPaths := func() ExternalPaths { return ExternalPaths{successfulPath} }
		windowTestMust(cx.updateWindow(firstDestination, func(_ AnyView, window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			window.DispatchEvent(FileDropEventEntered{Position: reentryPosition, Paths: externalPaths()}, cx)
			if !windowTestActiveDragIs[ExternalPaths](cx) {
				t.Fatal("first destination should see external paths")
			}
			window.DispatchEvent(FileDropEventExited{}, cx)
			if cx.activeDrag.IsSome() || firstExitCount != 1 || secondExitCount != 0 {
				t.Fatalf("first destination exit: %d %d", firstExitCount, secondExitCount)
			}
		}))

		windowTestMust(cx.updateWindow(secondDestination, func(_ AnyView, window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			keystroke, err := KeystrokeParse("down")
			windowTestMust(err)
			window.DispatchEvent(KeyDownEvent{Keystroke: keystroke}, cx)
			window.DispatchEvent(FileDropEventEntered{Position: reentryPosition, Paths: externalPaths()}, cx)
			if !windowTestActiveDragIs[ExternalPaths](cx) {
				t.Fatal("second destination should see external paths")
			}
			if firstExitCount != 1 || secondExitCount != 0 {
				t.Fatal("exit counts before exit")
			}
			window.DispatchEvent(FileDropEventExited{}, cx)
			if cx.activeDrag.IsSome() || firstExitCount != 1 || secondExitCount != 1 {
				t.Fatalf("second destination exit: %d %d", firstExitCount, secondExitCount)
			}
		}))

		windowTestMust(cx.updateWindow(successful.window, func(_ AnyView, window *Window, cx *App) {
			window.DispatchEvent(FileDropEventEntered{Position: reentryPosition, Paths: externalPaths()}, cx)
			if !windowTestActiveDragIs[string](cx) {
				t.Fatal("source window should restore its drag")
			}
			if lastMove(successful) != reentryPosition {
				t.Fatal("restored drag should see the reentry move")
			}
			window.DispatchEvent(FileDropEventExited{}, cx)
			if cx.activeDrag.IsSome() {
				t.Fatal("exit should suspend the drag")
			}
			window.DispatchEvent(FileDropEventEntered{Position: reentryPosition, Paths: externalPaths()}, cx)
			if !windowTestActiveDragIs[string](cx) {
				t.Fatal("source window should restore its drag again")
			}
			window.DispatchEvent(FileDropEventSubmit{Position: reentryPosition}, cx)
			if !slices.Equal(*successful.observedDrops, []string{successfulPath}) {
				t.Fatalf("drops = %v", *successful.observedDrops)
			}
			if cx.activeDrag.IsSome() {
				t.Fatal("drop should end the drag")
			}
			window.DispatchEvent(FileDropEventExited{}, cx)
			if cx.activeDrag.IsSome() {
				t.Fatal("exit after drop")
			}
			window.DispatchEvent(FileDropEventEnded{}, cx)
			if cx.activeDrag.IsSome() {
				t.Fatal("ended")
			}
			window.DispatchEvent(FileDropEventEntered{Position: reentryPosition, Paths: externalPaths()}, cx)
			if !windowTestActiveDragIs[ExternalPaths](cx) {
				t.Fatal("finished drag should not be restored")
			}
			window.DispatchEvent(FileDropEventExited{}, cx)
		}))

		cancelledPath := "/tmp/cancelled-drag"
		cancelled := startDrag(cancelledPath, true)
		windowTestMust(cx.updateWindow(cancelled.window, func(_ AnyView, window *Window, cx *App) {
			window.DispatchEvent(MouseMoveEvent{Position: outsidePosition, PressedButton: Some(MouseButtonLeft)}, cx)
			if cx.activeDrag.IsSome() {
				t.Fatal("drag not promoted")
			}
			window.DispatchEvent(FileDropEventEntered{Position: reentryPosition, Paths: ExternalPaths{cancelledPath}}, cx)
			if !windowTestActiveDragIs[string](cx) {
				t.Fatal("cancelled drag should restore")
			}
			if !cx.StopActiveDrag(window) || cx.activeDrag.IsSome() {
				t.Fatal("stop active drag")
			}
		}))
		if cx.UpdateR(func(cx *App) bool { return cx.endPlatformDrag(cancelled.window.WindowID()) }) {
			t.Fatal("cancelled drag still pending")
		}

		removed := startDrag("/tmp/removed-window-drag", true)
		removedWindowID := removed.window.WindowID()
		windowTestMust(cx.updateWindow(removed.window, func(_ AnyView, window *Window, cx *App) {
			window.DispatchEvent(MouseMoveEvent{Position: outsidePosition, PressedButton: Some(MouseButtonLeft)}, cx)
			if cx.activeDrag.IsSome() {
				t.Fatal("drag not promoted")
			}
			window.RemoveWindow()
		}))
		if cx.UpdateR(func(cx *App) bool { return cx.endPlatformDrag(removedWindowID) }) {
			t.Fatal("removed window drag still pending")
		}

		failedPath := "/tmp/failed-drag"
		failed := startDrag(failedPath, false)
		windowTestMust(cx.updateWindow(failed.window, func(_ AnyView, window *Window, cx *App) {
			for _, x := range []float32{-1., -2.} {
				window.DispatchEvent(MouseMoveEvent{Position: NewPoint(Px(x), Px(20.)), PressedButton: Some(MouseButtonLeft)}, cx)
			}
			if cx.activeDrag.IsNone() {
				t.Fatal("failed platform drag should keep the drag")
			}
		}))
		if got := cx.testWindow(failed.window).ExternalDragFiles(); !slices.Equal(got, []FileDragPath{{Path: failedPath, IsDir: true}}) {
			t.Fatalf("external drag files = %v", got)
		}
	})
}

type windowTestFocusForwarder struct {
	a, b FocusHandle
}

func (v *windowTestFocusForwarder) Render(_ *Window, _ *Context[windowTestFocusForwarder]) IntoElement {
	return NewDiv().SizeFull().
		Child(NewDiv().W(Px(50.).ToLength()).H(Px(50.).ToLength()).TrackFocus(v.a)).
		Child(NewDiv().W(Px(50.).ToLength()).H(Px(50.).ToLength()).TrackFocus(v.b))
}

func TestWindow_FocusMovedByFocusListenerIsDispatched(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		bFocusCount := 0
		window := cx.AddWindow(func(window *Window, cx *Context[windowTestFocusForwarder]) *windowTestFocusForwarder {
			a := cx.FocusHandle()
			b := cx.FocusHandle()
			cx.OnFocus(a, window, func(this *windowTestFocusForwarder, window *Window, cx *Context[windowTestFocusForwarder]) {
				window.Focus(this.b, cx.App)
			}).Detach()
			cx.OnFocus(b, window, func(*windowTestFocusForwarder, *Window, *Context[windowTestFocusForwarder]) {
				bFocusCount++
			}).Detach()
			return &windowTestFocusForwarder{a: a, b: b}
		})
		var a, b FocusHandle
		windowTestMust(window.Update(cx, func(v *windowTestFocusForwarder, _ *Window, _ *Context[windowTestFocusForwarder]) { a, b = v.a, v.b }))
		defer b.Release()
		defer a.Release()

		windowTestMust(window.Update(cx, func(_ *windowTestFocusForwarder, window *Window, _ *Context[windowTestFocusForwarder]) {
			window.ActivateWindow()
		}))
		cx.Executor().RunUntilParked()

		windowTestMust(window.Update(cx, func(this *windowTestFocusForwarder, window *Window, cx *Context[windowTestFocusForwarder]) {
			window.Focus(this.a, cx.App)
		}))
		cx.Executor().RunUntilParked()

		windowTestMust(window.Update(cx, func(this *windowTestFocusForwarder, window *Window, _ *Context[windowTestFocusForwarder]) {
			if !this.b.IsFocused(window) {
				t.Fatal("b should be focused")
			}
		}))
		if bFocusCount != 1 {
			t.Fatalf("b focus count = %d", bFocusCount)
		}
	})
}

type windowTestTouchPhaseX struct {
	phase TouchPhase
	x     Pixels
}

type windowTestTouchDragListener struct {
	events *[]windowTestTouchPhaseX
}

func (v *windowTestTouchDragListener) Render(_ *Window, _ *Context[windowTestTouchDragListener]) IntoElement {
	events := v.events
	return &windowTestCanvas{paint: func(_ Bounds[Pixels], window *Window, _ *App) {
		window.OnMouseEvent(func(event *TouchDragEvent, phase DispatchPhase, window *Window, _ *App) {
			if phase != DispatchPhaseBubble {
				return
			}
			*events = append(*events, windowTestTouchPhaseX{event.Phase, event.Position.X})
			if event.Phase == TouchPhaseStarted {
				window.PreventDefault()
			}
		})
	}}
}

func windowTestDispatchTouch[V any](window WindowHandle[V], cx *TestAppContext, id TouchID, phase TouchPhase, x float32) {
	windowTestMust(window.Update(cx, func(_ *V, window *Window, cx *Context[V]) {
		window.DispatchEvent(TouchEvent{ID: id, Phase: phase, Position: NewPoint(Px(x), Px(0.))}, cx.App)
	}))
}

func TestWindow_ClaimedTouchDragReceivesMovementAndRelease(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var events []windowTestTouchPhaseX
		window := cx.AddWindow(func(*Window, *Context[windowTestTouchDragListener]) *windowTestTouchDragListener {
			return &windowTestTouchDragListener{events: &events}
		})
		touch := TouchID(1)
		windowTestDispatchTouch(window, cx, touch, TouchPhaseStarted, 10.)
		windowTestDispatchTouch(window, cx, touch, TouchPhaseMoved, 30.)
		windowTestDispatchTouch(window, cx, touch, TouchPhaseEnded, 40.)
		if !slices.Equal(events, []windowTestTouchPhaseX{{TouchPhaseStarted, Px(10.)}, {TouchPhaseMoved, Px(30.)}, {TouchPhaseEnded, Px(40.)}}) {
			t.Fatalf("events = %v", events)
		}
	})
}

type windowTestBackListeners struct {
	claims *[]string
}

func (v *windowTestBackListeners) Render(_ *Window, _ *Context[windowTestBackListeners]) IntoElement {
	claim := func(name string) func(*BackGestureEvent, *Window, *App) {
		claims := v.claims
		return func(*BackGestureEvent, *Window, *App) { *claims = append(*claims, name) }
	}
	return NewDiv().SizeFull().Child(
		NewDiv().W(Px(100.).ToLength()).HFull().OnBackGesture(claim("outer")).
			Child(NewDiv().W(Px(50.).ToLength()).HFull().OnBackGesture(claim("inner"))),
	)
}

func TestWindow_BackGestureIsClaimedByTheInnermostListener(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var claims []string
		window := cx.AddWindow(func(*Window, *Context[windowTestBackListeners]) *windowTestBackListeners {
			return &windowTestBackListeners{claims: &claims}
		})
		dispatchBack := func(x float32) bool {
			result, err := window.UpdateR(cx, func(_ *windowTestBackListeners, window *Window, cx *Context[windowTestBackListeners]) bool {
				return window.DispatchEvent(BackGestureEvent{Phase: BackGesturePhaseStarted, StartPosition: NewPoint(Px(x), Px(10.))}, cx.App).DefaultPrevented
			})
			windowTestMust(err)
			return result
		}
		if !dispatchBack(10.) || !dispatchBack(80.) || dispatchBack(150.) {
			t.Fatal("back gesture claims")
		}
		if !slices.Equal(claims, []string{"inner", "outer"}) {
			t.Fatalf("claims = %v", claims)
		}
	})
}

type windowTestLongPressResponse uint8

const (
	windowTestLongPressPreventDefault windowTestLongPressResponse = iota
	windowTestLongPressStopPropagation
	windowTestLongPressNone
)

type windowTestLongPressListener struct {
	phases   *[]TouchPhase
	response windowTestLongPressResponse
}

func (v *windowTestLongPressListener) Render(_ *Window, cx *Context[windowTestLongPressListener]) IntoElement {
	weakEntity := cx.WeakEntity()
	phases := v.phases
	response := v.response
	return &windowTestCanvas{paint: func(_ Bounds[Pixels], window *Window, _ *App) {
		window.OnMouseEvent(func(event *LongPressEvent, phase DispatchPhase, window *Window, cx *App) {
			if phase != DispatchPhaseBubble {
				return
			}
			*phases = append(*phases, event.Phase)
			switch response {
			case windowTestLongPressPreventDefault:
				if entity, ok := weakEntity.Upgrade(); ok {
					window.CaptureLongPress(entity)
					entity.Release()
				}
				window.PreventDefault()
			case windowTestLongPressStopPropagation:
				cx.StopPropagation()
			}
		})
	}}
}

func windowTestAddLongPress(cx *TestAppContext, phases *[]TouchPhase, response windowTestLongPressResponse) WindowHandle[windowTestLongPressListener] {
	return cx.AddWindow(func(*Window, *Context[windowTestLongPressListener]) *windowTestLongPressListener {
		return &windowTestLongPressListener{phases: phases, response: response}
	})
}

func TestWindow_LongPressIsClaimedOnlyWhenStartedPreventsDefault(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		for _, response := range []windowTestLongPressResponse{windowTestLongPressPreventDefault, windowTestLongPressStopPropagation, windowTestLongPressNone} {
			var phases []TouchPhase
			window := windowTestAddLongPress(cx, &phases, response)
			windowTestDispatchTouch(window, cx, TouchID(1), TouchPhaseStarted, 0.)
			cx.Executor().AdvanceClock(501 * time.Millisecond)
			cx.Executor().RunUntilParked()
			windowTestMust(window.Update(cx, func(_ *windowTestLongPressListener, window *Window, _ *Context[windowTestLongPressListener]) {
				if window.longPressCapture.IsSome() != (response == windowTestLongPressPreventDefault) {
					t.Fatalf("response %d: capture = %v", response, window.longPressCapture)
				}
			}))
			windowTestDispatchTouch(window, cx, TouchID(1), TouchPhaseMoved, 2.)
			windowTestDispatchTouch(window, cx, TouchID(1), TouchPhaseEnded, 2.)
			windowTestMust(window.Update(cx, func(_ *windowTestLongPressListener, window *Window, _ *Context[windowTestLongPressListener]) {
				if window.longPressCapture.IsSome() {
					t.Fatal("capture survived the touch")
				}
			}))
			want := []TouchPhase{TouchPhaseStarted}
			if response == windowTestLongPressPreventDefault {
				want = []TouchPhase{TouchPhaseStarted, TouchPhaseMoved, TouchPhaseEnded}
			}
			if !slices.Equal(phases, want) {
				t.Fatalf("response %d: phases = %v", response, phases)
			}
		}
	})
}

func TestWindow_StaleDefaultPreventionDoesNotClaimLongPress(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var phases []TouchPhase
		window := windowTestAddLongPress(cx, &phases, windowTestLongPressNone)
		windowTestMust(window.Update(cx, func(_ *windowTestLongPressListener, window *Window, _ *Context[windowTestLongPressListener]) {
			window.PreventDefault()
		}))
		windowTestDispatchTouch(window, cx, TouchID(1), TouchPhaseStarted, 0.)
		cx.Executor().AdvanceClock(501 * time.Millisecond)
		cx.Executor().RunUntilParked()
		windowTestDispatchTouch(window, cx, TouchID(1), TouchPhaseMoved, 2.)
		if !slices.Equal(phases, []TouchPhase{TouchPhaseStarted}) {
			t.Fatalf("phases = %v", phases)
		}
	})
}

func TestWindow_ResolvedTouchCancelsScheduledLongPress(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		for _, c := range []struct {
			phase    TouchPhase
			position float32
		}{{TouchPhaseEnded, 0.}, {TouchPhaseCancelled, 0.}, {TouchPhaseMoved, 20.}} {
			var phases []TouchPhase
			window := windowTestAddLongPress(cx, &phases, windowTestLongPressPreventDefault)
			windowTestDispatchTouch(window, cx, TouchID(1), TouchPhaseStarted, 0.)
			windowTestDispatchTouch(window, cx, TouchID(1), c.phase, c.position)
			cx.Executor().AdvanceClock(501 * time.Millisecond)
			cx.Executor().RunUntilParked()
			if len(phases) != 0 {
				t.Fatalf("%v allowed long press", c.phase)
			}
		}
	})
}

func TestWindow_StaleLongPressTimerCannotAffectReplacementTouch(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var phases []TouchPhase
		window := windowTestAddLongPress(cx, &phases, windowTestLongPressPreventDefault)
		firstTouch := TouchID(1)
		windowTestDispatchTouch(window, cx, firstTouch, TouchPhaseStarted, 0.)
		cx.Executor().AdvanceClock(250 * time.Millisecond)
		windowTestDispatchTouch(window, cx, firstTouch, TouchPhaseCancelled, 0.)
		windowTestDispatchTouch(window, cx, TouchID(2), TouchPhaseStarted, 10.)

		cx.Executor().AdvanceClock(251 * time.Millisecond)
		cx.Executor().RunUntilParked()
		if len(phases) != 0 {
			t.Fatalf("phases = %v", phases)
		}
		cx.Executor().AdvanceClock(250 * time.Millisecond)
		cx.Executor().RunUntilParked()
		if !slices.Equal(phases, []TouchPhase{TouchPhaseStarted}) {
			t.Fatalf("phases = %v", phases)
		}
	})
}

type windowTestPaintView struct {
	paint func(window *Window)
}

func (v *windowTestPaintView) Render(_ *Window, _ *Context[windowTestPaintView]) IntoElement {
	paint := v.paint
	return &windowTestCanvas{sizeFull: true, paint: func(_ Bounds[Pixels], window *Window, _ *App) {
		window.contentMaskStack = append(window.contentMaskStack, ContentMask[Pixels]{Bounds: BoundsFromCorners(NewPoint(Px(-1000.), Px(-1000.)), NewPoint(Px(1000.), Px(1000.)))})
		paint(window)
		window.contentMaskStack = window.contentMaskStack[:len(window.contentMaskStack)-1]
	}}
}

func windowTestPaintAtScales(cx *TestAppContext, paint func(window *Window)) {
	window := cx.AddWindow(func(*Window, *Context[windowTestPaintView]) *windowTestPaintView {
		return &windowTestPaintView{paint: paint}
	})
	for _, scale := range []float32{1., 1.25, 1.5, 2., 3.} {
		cx.SimulateWindowScaleFactorChange(window.AnyWindowHandle, scale)
		windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) { window.Draw(cx).Clear(cx) }))
	}
}

func windowTestPaintUnderlines(window *Window, paint func(window *Window)) []Underline {
	window.nextFrame.scene.Clear()
	paint(window)
	return window.nextFrame.scene.Underlines
}

func windowTestAssertSameUnderlineGeometry(t *testing.T, actual, expected Underline) {
	t.Helper()
	if actual.Bounds != expected.Bounds || actual.Thickness != expected.Thickness || actual.Wavy != expected.Wavy || actual.Color != expected.Color || actual.Pad != expected.Pad {
		t.Fatalf("underline geometry %+v != %+v", actual, expected)
	}
}

func windowTestUnderlineDeviceColumns(underlines []Underline) []int32 {
	columns := []int32{}
	for _, underline := range underlines {
		bounds := underline.Bounds.Intersect(underline.ContentMask.Bounds)
		for column := int32(bounds.Left()); column < int32(bounds.Right()); column++ {
			columns = append(columns, column)
		}
	}
	return columns
}

func windowTestRunPaint(t *testing.T, paint func(t *testing.T, window *Window)) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		windowTestPaintAtScales(cx, func(window *Window) { paint(t, window) })
	})
}

func TestWindow_UnderlineExclusionsPreserveDeviceGeometry(t *testing.T) {
	windowTestRunPaint(t, func(t *testing.T, window *Window) {
		var numerator, denominator int32
		switch window.ScaleFactor() {
		case 1.0:
			numerator, denominator = 1, 1
		case 1.25:
			numerator, denominator = 5, 4
		case 1.5:
			numerator, denominator = 3, 2
		case 2.0:
			numerator, denominator = 2, 1
		case 3.0:
			numerator, denominator = 3, 1
		default:
			panic(fmt.Sprintf("unexpected scale %v", window.ScaleFactor()))
		}
		abs := func(v int32) int32 {
			if v < 0 {
				return -v
			}
			return v
		}
		signum := func(v int32) int32 {
			switch {
			case v > 0:
				return 1
			case v < 0:
				return -1
			}
			return 0
		}
		snap := func(quarters int32) ScaledPixels {
			scaled := quarters * numerator
			divisor := 4 * denominator
			rounded := abs(scaled) / divisor
			if 2*(abs(scaled)%divisor) > divisor {
				rounded++
			}
			return ScaledPixels(float32(signum(scaled) * rounded))
		}
		stroke := func(quarters int32) ScaledPixels {
			if quarters == 0 {
				return 0
			}
			return max(snap(quarters), ScaledPixels(1.))
		}
		color := NewHsla(0.25, 0.5, 0.75, 0.5)

		for _, origin := range []int32{-101, -3, -2, -1, 0, 1, 2, 3, 101} {
			for _, width := range []int32{0, 1, 3, 84} {
				for _, thickness := range []int32{0, 1, 4, 7} {
					for _, wavy := range []bool{false, true} {
						style := UnderlineStyle{Thickness: Px(float32(thickness) / 4.), Color: Some(color), Wavy: wavy}
						heightFactor := ScaledPixels(1.)
						if wavy {
							heightFactor = 3.
						}
						bounds := NewBounds(NewPoint(snap(origin), snap(-origin)), NewSize(stroke(width), stroke(thickness)*heightFactor))
						originPoint := NewPoint(Px(float32(origin)/4.), Px(float32(-origin)/4.))
						widthPx := Px(float32(width) / 4.)
						if got := window.UnderlineBounds(originPoint, widthPx, style); got != bounds {
							t.Fatalf("underline bounds %v != %v", got, bounds)
						}
						original := slices.Clone(windowTestPaintUnderlines(window, func(window *Window) {
							window.PaintUnderline(originPoint, widthPx, style)
						}))
						wantLen := 0
						if !bounds.IsEmpty() {
							wantLen = 1
						}
						if len(original) != wantLen {
							t.Fatalf("underlines = %d, want %d", len(original), wantLen)
						}
						if len(original) > 0 {
							u := original[0]
							if u.Bounds != bounds || u.Thickness != stroke(thickness) || u.Wavy != PaddedBool32FromBool(wavy) || u.Color != color {
								t.Fatalf("underline %+v", u)
							}
						}

						underlines := windowTestPaintUnderlines(window, func(window *Window) {
							window.PaintUnderlineWithExclusions(originPoint, widthPx, style, nil)
						})
						if len(underlines) != len(original) {
							t.Fatal("exclusion-free underline count")
						}
						for i := range underlines {
							windowTestAssertSameUnderlineGeometry(t, underlines[i], original[i])
							if underlines[i].ContentMask != original[i].ContentMask {
								t.Fatal("content mask")
							}
						}

						left := int32(bounds.Left())
						right := int32(bounds.Right())
						var exclusions []ScaledPixelsRange
						for _, span := range [][2]int32{{15, 25}, {3, 7}, {5, 10}, {10, 12}, {0, 0}, {4, 2}, {-50, -1}, {70, 100}} {
							exclusions = append(exclusions, ScaledPixelsRange{Start: ScaledPixels(float32(left + span[0])), End: ScaledPixels(float32(left + span[1]))})
						}
						underlines = windowTestPaintUnderlines(window, func(window *Window) {
							window.PaintUnderlineWithExclusions(originPoint, widthPx, style, exclusions)
						})
						expected := []int32{}
						for column := left; column < right; column++ {
							excluded := false
							for _, span := range exclusions {
								c := ScaledPixels(float32(column))
								if span.Start <= c && c < span.End {
									excluded = true
								}
							}
							if !bounds.IsEmpty() && !excluded {
								expected = append(expected, column)
							}
						}
						if got := windowTestUnderlineDeviceColumns(underlines); !slices.Equal(got, expected) {
							t.Fatalf("scale %v origin %d width %d thickness %d wavy %v: columns %v, want %v", window.ScaleFactor(), origin, width, thickness, wavy, got, expected)
						}
						for _, underline := range underlines {
							windowTestAssertSameUnderlineGeometry(t, underline, original[0])
						}
					}
				}
			}
		}
	})
}

func TestWindow_UnderlineExclusionsPreserveHelveticaWarningEdge(t *testing.T) {
	windowTestRunPaint(t, func(t *testing.T, window *Window) {
		if window.ScaleFactor() != 2.0 {
			return
		}
		style := UnderlineStyle{Thickness: Px(1.), Wavy: true}
		origin := NewPoint(Px(2.888_183_8), Px(15.357_917))
		width := Px(23.848_145) - origin.X
		original := windowTestPaintUnderlines(window, func(window *Window) {
			window.PaintUnderline(origin, width, style)
		})[0]
		if original.Bounds.Left() != 6. || original.Bounds.Right() != 48. {
			t.Fatalf("original bounds %v", original.Bounds)
		}
		pointBounds := window.UnderlineBounds(origin, Px(10.829_102), style)
		if pointBounds.Left() != 6. || pointBounds.Right() != 28. {
			t.Fatalf("point bounds %v", pointBounds)
		}
		underlines := windowTestPaintUnderlines(window, func(window *Window) {
			window.PaintUnderlineWithExclusions(origin, width, style, []ScaledPixelsRange{{Start: pointBounds.Left(), End: pointBounds.Right()}})
		})
		if len(underlines) != 1 {
			t.Fatalf("underlines = %d", len(underlines))
		}
		windowTestAssertSameUnderlineGeometry(t, underlines[0], original)
		want := []int32{}
		for c := int32(28); c < 48; c++ {
			want = append(want, c)
		}
		if got := windowTestUnderlineDeviceColumns(underlines); !slices.Equal(got, want) {
			t.Fatalf("columns %v", got)
		}
	})
}

func TestWindow_UnderlineExclusionsPreserveContentMaskAndOpacity(t *testing.T) {
	windowTestRunPaint(t, func(t *testing.T, window *Window) {
		scale := window.ScaleFactor()
		originalMask := window.ContentMask()
		originalOpacity := window.elementOpacity()
		mask := ContentMask[Pixels]{Bounds: BoundsFromCorners(NewPoint(Px(4.), Px(10.)), NewPoint(Px(16.), Px(12.)))}
		style := UnderlineStyle{Thickness: Px(2.), Color: Some(NewHsla(0.25, 0.5, 0.75, 0.5)), Wavy: true}
		origin := NewPoint(Px(-1.25), Px(9.25))
		width := Px(24.)
		window.WithContentMask(Some(mask), func(window *Window) struct{} {
			window.withElementOpacity(Some[float32](0.5), func(window *Window) struct{} {
				window.withElementOpacity(Some[float32](0.5), func(window *Window) struct{} {
					original := windowTestPaintUnderlines(window, func(window *Window) {
						window.PaintUnderline(origin, width, style)
					})[0]
					if original.Color != NewHsla(0.25, 0.5, 0.75, 0.125) {
						t.Fatalf("color %v", original.Color)
					}
					underlines := windowTestPaintUnderlines(window, func(window *Window) {
						window.PaintUnderlineWithExclusions(origin, width, style, []ScaledPixelsRange{{Start: ScaledPixels(6. * scale), End: ScaledPixels(8. * scale)}})
					})
					if len(underlines) != 2 {
						t.Fatalf("underlines = %d", len(underlines))
					}
					for i, lr := range [][2]float32{{4. * scale, 6. * scale}, {8. * scale, 16. * scale}} {
						expected := BoundsFromCorners(
							NewPoint(ScaledPixels(lr[0]), ScaledPixels(float32(math.Floor(float64(10.*scale))))),
							NewPoint(ScaledPixels(lr[1]), ScaledPixels(float32(math.Ceil(float64(12.*scale))))),
						)
						if underlines[i].ContentMask.Bounds != expected {
							t.Fatalf("mask %d: %v != %v", i, underlines[i].ContentMask.Bounds, expected)
						}
					}
					for _, underline := range underlines {
						windowTestAssertSameUnderlineGeometry(t, underline, original)
					}
					return struct{}{}
				})
				return struct{}{}
			})
			return struct{}{}
		})
		if window.ContentMask() != originalMask || window.elementOpacity() != originalOpacity {
			t.Fatal("mask or opacity not restored")
		}
	})
}

func TestWindow_UnderlineExclusionsAdversarialPermutations(t *testing.T) {
	windowTestRunPaint(t, func(t *testing.T, window *Window) {
		style := UnderlineStyle{Thickness: Px(1.), Wavy: true}
		origin := NewPoint(Px(-4.), Px(-1.))
		width := Px(8.)
		left := int32(-4. * window.ScaleFactor())
		right := int32(4. * window.ScaleFactor())
		endpoints := []int32{-16, -8, -1, 0, 1, 8, 16}
		for _, firstStart := range endpoints {
			for _, firstEnd := range endpoints {
				for _, secondStart := range endpoints {
					for _, secondEnd := range endpoints {
						spans := [][2]int32{{firstStart, firstEnd}, {secondStart, secondEnd}}
						expected := []int32{}
						for column := left; column < right; column++ {
							excluded := false
							for _, span := range spans {
								if span[0] <= column && column < span[1] {
									excluded = true
								}
							}
							if !excluded {
								expected = append(expected, column)
							}
						}
						exclusions := []ScaledPixelsRange{}
						for _, span := range spans {
							exclusions = append(exclusions, ScaledPixelsRange{Start: ScaledPixels(float32(span[0])), End: ScaledPixels(float32(span[1]))})
						}
						underlines := windowTestPaintUnderlines(window, func(window *Window) {
							window.PaintUnderlineWithExclusions(origin, width, style, exclusions)
						})
						if got := windowTestUnderlineDeviceColumns(underlines); !slices.Equal(got, expected) {
							t.Fatalf("%v: %v != %v", exclusions, got, expected)
						}
					}
				}
			}
		}
	})
}

func TestWindow_EmptyContentMasksRemainEmptyWhenSnapped(t *testing.T) {
	windowTestRunPaint(t, func(t *testing.T, window *Window) {
		viewport := ContentMask[Pixels]{Bounds: BoundsFromCorners(NewPoint(Px(20.), Px(20.)), NewPoint(Px(100.), Px(100.)))}
		for _, fraction := range []float32{0., 0.25, 0.5, 0.75} {
			for _, bounds := range []Bounds[Pixels]{
				NewBounds(NewPoint(Px(30.), Px(120.+fraction)), NewSize(Px(40.), Px(20.))),
				NewBounds(NewPoint(Px(120.+fraction), Px(30.)), NewSize(Px(20.), Px(40.))),
				NewBounds(NewPoint(Px(120.+fraction), Px(120.+fraction)), NewSize(Px(20.), Px(20.))),
				NewBounds(NewPoint(Px(30.+fraction), Px(40.+fraction)), NewSize(Px(0.), Px(20.))),
				NewBounds(NewPoint(Px(30.+fraction), Px(40.+fraction)), NewSize(Px(20.), Px(0.))),
			} {
				window.WithContentMask(Some(viewport), func(window *Window) struct{} {
					return window.WithContentMask(Some(ContentMask[Pixels]{Bounds: bounds}), func(window *Window) struct{} {
						if !window.ContentMask().Bounds.IsEmpty() {
							t.Fatal("content mask not empty")
						}
						if !window.snappedContentMask().Bounds.IsEmpty() {
							t.Fatalf("scale=%v, bounds=%v", window.ScaleFactor(), bounds)
						}
						count := window.nextFrame.scene.Len()
						window.PaintQuad(NewFill(bounds, Red().ToBackground()))
						window.PaintQuad(Outline(bounds, Red(), BorderStyle(0)))
						if window.nextFrame.scene.Len() != count {
							t.Fatal("empty mask painted")
						}
						return struct{}{}
					})
				})
			}
		}
	})
}

func TestWindow_NonemptyContentMasksKeepCoveringFractionalEdges(t *testing.T) {
	windowTestRunPaint(t, func(t *testing.T, window *Window) {
		bounds := NewBounds(NewPoint(Px(30.25), Px(40.25)), NewSize(Px(0.125), Px(0.125)))
		window.WithContentMask(Some(ContentMask[Pixels]{Bounds: bounds}), func(window *Window) struct{} {
			snapped := window.snappedContentMask().Bounds
			scale := float64(window.ScaleFactor())
			if snapped != window.coverBounds(bounds) || snapped.IsEmpty() {
				t.Fatalf("snapped %v", snapped)
			}
			if float64(snapped.Left()) != math.Floor(30.25*scale) || float64(snapped.Top()) != math.Floor(40.25*scale) ||
				float64(snapped.Right()) != math.Ceil(30.375*scale) || float64(snapped.Bottom()) != math.Ceil(40.375*scale) {
				t.Fatalf("snapped %v at %v", snapped, scale)
			}
			return struct{}{}
		})
	})
}

func TestWindow_EmptyShadowBoundsAreNotPainted(t *testing.T) {
	windowTestRunPaint(t, func(t *testing.T, window *Window) {
		shadow := NewBoxShadow(Px(0.25), Px(0.5), Red()).BlurRadius(Px(4.))
		for _, origin := range []Point[Pixels]{NewPoint(Px(30.25), Px(40.25)), NewPoint(Px(-30.25), Px(-40.25))} {
			for _, size := range []Size[Pixels]{NewSize(Px(20.), Px(0.)), NewSize(Px(0.), Px(20.)), NewSize(Px(0.), Px(0.))} {
				bounds := NewBounds(origin, size)
				count := window.nextFrame.scene.Len()
				window.PaintDropShadows(bounds, Corners[Pixels]{}, []BoxShadow{shadow})
				if window.nextFrame.scene.Len() != count {
					t.Fatalf("scale=%v, bounds=%v", window.ScaleFactor(), bounds)
				}
				if !window.coverBounds(bounds).IsEmpty() {
					t.Fatal("cover bounds not empty")
				}
			}
		}
		bounds := NewBounds(NewPoint(Px(30.25), Px(40.25)), NewSize(Px(20.), Px(20.)))
		collapsedShadow := shadow.SpreadRadius(Px(-10.))
		shapeBounds := bounds.Add(collapsedShadow.Offset).Dilate(collapsedShadow.Spread)
		count := window.nextFrame.scene.Len()
		window.PaintDropShadows(bounds, Corners[Pixels]{}, []BoxShadow{collapsedShadow})
		if window.nextFrame.scene.Len() != count {
			t.Fatal("collapsed shadow painted")
		}
		if !window.coverBounds(shapeBounds).IsEmpty() {
			t.Fatal("collapsed shape not empty")
		}
		window.PaintDropShadows(bounds, Corners[Pixels]{}, []BoxShadow{shadow})
		if window.nextFrame.scene.Len() != count+1 {
			t.Fatal("shadow not painted")
		}
	})
}

func TestWindow_CollapsedInsetShadowHoleIsAnEmptyPointOnThePixelGrid(t *testing.T) {
	windowTestRunPaint(t, func(t *testing.T, window *Window) {
		bounds := NewBounds(NewPoint(Px(30.25), Px(40.25)), NewSize(Px(20.), Px(20.)))
		for _, blurRadius := range []Pixels{Px(0.), Px(4.)} {
			for _, spreadRadius := range []Pixels{Px(10.), Px(12.3), Px(15.), Px(30.)} {
				shadow := NewBoxShadow(Px(0.), Px(0.), Red()).BlurRadius(blurRadius).SpreadRadius(spreadRadius).Inset()
				window.nextFrame.scene.Clear()
				window.PaintInsetShadows(bounds, Corners[Pixels]{}, []BoxShadow{shadow})
				shadows := window.nextFrame.scene.Shadows
				if len(shadows) != 1 {
					t.Fatalf("shadows = %d", len(shadows))
				}
				hole := shadows[0].ShapeBounds
				_, fx := math.Modf(float64(hole.Origin.X))
				_, fy := math.Modf(float64(hole.Origin.Y))
				if hole.Size != (Size[ScaledPixels]{}) || fx != 0 || fy != 0 {
					t.Fatalf("scale=%v, blur=%v, spread=%v, hole=%v", window.ScaleFactor(), blurRadius, spreadRadius, hole)
				}
			}
		}
	})
}

type windowTestStandaloneModifierAction struct{}

func (windowTestStandaloneModifierAction) Name() string {
	return "window_test::StandaloneModifierAction"
}
func (a windowTestStandaloneModifierAction) BoxedClone() Action { return a }
func (windowTestStandaloneModifierAction) PartialEq(other Action) bool {
	_, ok := other.(windowTestStandaloneModifierAction)
	return ok
}

type windowTestStandaloneModifierView struct {
	focusHandle FocusHandle
	actionCount int
}

func (v *windowTestStandaloneModifierView) Render(_ *Window, cx *Context[windowTestStandaloneModifierView]) IntoElement {
	listener := cx.Listener(func(this *windowTestStandaloneModifierView, _ *windowTestStandaloneModifierAction, _ *Window, _ *Context[windowTestStandaloneModifierView]) {
		this.actionCount++
	})
	return NewDiv().KeyContext("StandaloneModifier").TrackFocus(v.focusHandle).
		OnAction(func(action windowTestStandaloneModifierAction, window *Window, cx *App) {
			listener(&action, window, cx)
		})
}

func windowTestStandaloneModifierSetup(cx *TestAppContext) (Entity[windowTestStandaloneModifierView], *VisualTestContext, func()) {
	cx.Update(func(cx *App) {
		cx.BindKeys([]KeyBinding{NewKeyBinding("shift shift", windowTestStandaloneModifierAction{}, Some("StandaloneModifier"))})
	})
	view, vcx := cx.AddWindowView(func(window *Window, cx *Context[windowTestStandaloneModifierView]) *windowTestStandaloneModifierView {
		focusHandle := cx.FocusHandle()
		window.Focus(focusHandle, cx.App)
		return &windowTestStandaloneModifierView{focusHandle: focusHandle}
	})
	return view, vcx, func() {
		var focusHandle FocusHandle
		vcx.Update(func(_ *Window, cx *App) { focusHandle = view.Read(cx).focusHandle })
		view.Release()
		focusHandle.Release()
	}
}

func TestWindow_StandaloneModifierSequenceTimeoutAndRecovery(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		view, cx, cleanup := windowTestStandaloneModifierSetup(tc.Cx())
		defer cleanup()
		shift := Modifiers{Shift: true}
		check := func(pending bool, count int) {
			t.Helper()
			cx.Update(func(window *Window, cx *App) {
				if window.HasPendingKeystrokes() != pending || view.Read(cx).actionCount != count {
					t.Fatalf("pending %v count %d", window.HasPendingKeystrokes(), view.Read(cx).actionCount)
				}
			})
		}
		cx.SimulateModifiersChange(shift)
		cx.SimulateModifiersChange(Modifiers{})
		cx.Update(func(window *Window, _ *App) {
			shiftKeystroke, err := KeystrokeParse("shift")
			windowTestMust(err)
			pending, ok := window.PendingInputKeystrokes()
			if !ok || !slices.Equal(pending, []Keystroke{shiftKeystroke}) {
				t.Fatalf("pending = %v", pending)
			}
		})
		cx.Executor().AdvanceClock(2 * time.Second)
		cx.RunUntilParked()
		check(false, 0)

		cx.SimulateModifiersChange(shift)
		cx.SimulateModifiersChange(Modifiers{})
		check(true, 0)

		cx.SimulateModifiersChange(shift)
		cx.SimulateModifiersChange(Modifiers{})
		check(false, 1)
	})
}

func TestWindow_StandaloneModifierSequenceDispatchesBeforeTimeout(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		view, cx, cleanup := windowTestStandaloneModifierSetup(tc.Cx())
		defer cleanup()
		shift := Modifiers{Shift: true}
		check := func(pending bool, count int) {
			t.Helper()
			cx.Update(func(window *Window, cx *App) {
				if window.HasPendingKeystrokes() != pending || view.Read(cx).actionCount != count {
					t.Fatalf("pending %v count %d", window.HasPendingKeystrokes(), view.Read(cx).actionCount)
				}
			})
		}
		cx.SimulateModifiersChange(shift)
		cx.SimulateModifiersChange(Modifiers{})
		cx.Executor().AdvanceClock(500 * time.Millisecond)
		check(true, 0)
		cx.SimulateModifiersChange(shift)
		cx.SimulateModifiersChange(Modifiers{})
		check(false, 1)
		cx.Executor().AdvanceClock(2 * time.Second)
		cx.RunUntilParked()
		check(false, 1)
	})
}

type windowTestCachedChild struct {
	renders int
}

func (c *windowTestCachedChild) Render(_ *Window, _ *Context[windowTestCachedChild]) IntoElement {
	c.renders++
	return NewDiv().SizeFull().Bg(Rgb(0xff0000))
}

type windowTestCachedRoot struct {
	opacity float32
	width   Pixels
	child   Entity[windowTestCachedChild]
}

func (r *windowTestCachedRoot) Render(_ *Window, _ *Context[windowTestCachedRoot]) IntoElement {
	var childStyle StyleRefinement
	divTestStyled(&childStyle).Size(Px(100.))
	return NewDiv().W(r.width.ToLength()).H(Px(200.).ToLength()).Opacity(r.opacity).
		Hover(func(style *StyleRefinement) *StyleRefinement { return divTestStyled(style).Opacity(1.) }).
		Child(r.child.Clone().Cached(childStyle))
}

func TestWindow_CachedViewFollowsInheritedOpacity(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		child := cx.New(func(*Context[windowTestCachedChild]) *windowTestCachedChild { return &windowTestCachedChild{} })
		window := cx.AddWindow(func(*Window, *Context[windowTestCachedRoot]) *windowTestCachedRoot {
			return &windowTestCachedRoot{opacity: 1., width: Px(200.), child: child.Clone()}
		})
		defer func() {
			_ = window.Update(cx, func(root *windowTestCachedRoot, _ *Window, _ *Context[windowTestCachedRoot]) { root.child.Release() })
			child.Release()
		}()
		type result struct {
			alpha   float32
			renders int
		}
		updateAndDraw := func(update func(root *windowTestCachedRoot, window *Window, cx *Context[windowTestCachedRoot])) result {
			t.Helper()
			windowTestMust(window.Update(cx, func(root *windowTestCachedRoot, window *Window, cx *Context[windowTestCachedRoot]) {
				update(root, window, cx)
				cx.Notify()
			}))
			for range 3 {
				var r result
				var requestedFrames int
				windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) {
					window.Draw(cx).Clear(cx)
					quads := window.renderedFrame.scene.Quads
					if len(quads) != 1 {
						t.Fatalf("quads = %d", len(quads))
					}
					r = result{quads[0].Background.solid.A, child.Read(cx).renders}
					requestedFrames = window.SimulateNextFrame(cx)
				}))
				if requestedFrames == 0 {
					return r
				}
			}
			panic("cached child kept requesting frames")
		}
		outside := NewPoint(Px(500.), Px(500.))
		inside := NewPoint(Px(50.), Px(50.))
		beside := NewPoint(Px(250.), Px(50.))
		noop := func(*windowTestCachedRoot, *Window, *Context[windowTestCachedRoot]) {}
		moveTo := func(position Point[Pixels]) func(*windowTestCachedRoot, *Window, *Context[windowTestCachedRoot]) {
			return func(_ *windowTestCachedRoot, window *Window, cx *Context[windowTestCachedRoot]) {
				window.SimulateMouseMove(position, cx.App)
			}
		}

		if got := updateAndDraw(moveTo(outside)); got != (result{1., 1}) {
			t.Fatalf("move outside: %v", got)
		}
		if got := updateAndDraw(noop); got != (result{1., 1}) {
			t.Fatalf("noop: %v", got)
		}
		if got := updateAndDraw(func(root *windowTestCachedRoot, _ *Window, _ *Context[windowTestCachedRoot]) { root.opacity = 0.5 }); got != (result{0.5, 2}) {
			t.Fatalf("fade: %v", got)
		}
		if got := updateAndDraw(noop); got != (result{0.5, 2}) {
			t.Fatalf("noop after fade: %v", got)
		}

		if got := updateAndDraw(moveTo(inside)).alpha; got != 1. {
			t.Fatalf("hover inside: %v", got)
		}
		if got := updateAndDraw(moveTo(beside)).alpha; got != 0.5 {
			t.Fatalf("hover beside: %v", got)
		}

		if got := updateAndDraw(func(root *windowTestCachedRoot, _ *Window, _ *Context[windowTestCachedRoot]) { root.width = Px(300.) }).alpha; got != 1. {
			t.Fatalf("widen: %v", got)
		}
	})
}

type windowTestResizeHandle struct{}

func (*windowTestResizeHandle) Render(_ *Window, _ *Context[windowTestResizeHandle]) IntoElement {
	return NewDiv().SizeFull().CursorText().Child(
		NewDiv().ID("resize-handle").W(Px(4.).ToLength()).HFull().CursorColResize().
			OnDrag(struct{}{}, func(_ struct{}, _ Point[Pixels], _ *Window, cx *App) Entity[windowTestEmptyEntity] {
				return windowTestNewEmpty(cx)
			}),
	)
}

func TestWindow_DragCursorAppliesBeforeNextPaint(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := cx.AddWindow(func(*Window, *Context[windowTestResizeHandle]) *windowTestResizeHandle {
			return &windowTestResizeHandle{}
		})
		handlePosition := NewPoint(Px(1.), Px(10.))
		dragPosition := NewPoint(Px(20.), Px(10.))
		cursorObserver := cx.testPlatform

		windowTestMust(cx.updateWindow(window.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) {
			*window.hovered = true
			*window.active = true
			window.Draw(cx).Clear(cx)
			window.DispatchEvent(MouseMoveEvent{Position: handlePosition}, cx)
			window.DispatchEvent(MouseDownEvent{Position: handlePosition, Button: MouseButtonLeft}, cx)
			window.DispatchEvent(MouseMoveEvent{Position: dragPosition, PressedButton: Some(MouseButtonLeft)}, cx)
			if !cx.HasActiveDrag() {
				t.Fatal("drag not started")
			}
			if got := cursorObserver.CursorStyle(); got != CursorStyleResizeColumn {
				t.Fatalf("cursor during drag = %v", got)
			}
			window.Draw(cx).Clear(cx)
			window.DispatchEvent(MouseUpEvent{Position: dragPosition, Button: MouseButtonLeft}, cx)
			window.Draw(cx).Clear(cx)
		}))
		if got := cursorObserver.CursorStyle(); got != CursorStyleIBeam {
			t.Fatalf("cursor after drag = %v", got)
		}
	})
}

type windowTestInspectorRoot struct {
	child Entity[windowTestInspectorView]
}

func (r *windowTestInspectorRoot) Render(_ *Window, _ *Context[windowTestInspectorRoot]) IntoElement {
	var style StyleRefinement
	divTestStyled(&style).Size(Px(100.))
	return r.child.Clone().Cached(style)
}

type windowTestInspectorView struct {
	childWidths *[]Pixels
}

func (v *windowTestInspectorView) Render(_ *Window, _ *Context[windowTestInspectorView]) IntoElement {
	childWidths := v.childWidths
	d := NewDiv().
		OnChildrenPrepainted(func(bounds []Bounds[Pixels], _ *Window, _ *App) {
			widths := make([]Pixels, 0, len(bounds))
			for _, b := range bounds {
				widths = append(widths, b.Size.Width)
			}
			*childWidths = widths
		}).
		WithDynamicPrepaintOrder(func(*Window, *App) []int { return []int{2, 0, 1} })
	children := make([]IntoElement, 0, 3)
	for range 3 {
		children = append(children, NewDiv().Size(Px(10.)).FlexShrink0())
	}
	return d.ID("inspector-root").Flex().Children(children...)
}

func windowTestAssertClosedInspector(t *testing.T, window AnyWindowHandle, cx *TestAppContext) {
	t.Helper()
	windowTestMust(cx.updateWindow(window, func(root AnyView, window *Window, cx *App) {
		window.Draw(cx).Clear(cx)
		window.Draw(cx).Clear(cx)
		r, ok := root.Downcast[windowTestInspectorRoot]()
		if !ok {
			t.Fatal("test root")
		}
		if got := *r.Read(cx).child.Read(cx).childWidths; !slices.Equal(got, []Pixels{Px(10.), Px(10.), Px(10.)}) {
			t.Fatalf("child widths = %v", got)
		}
		for _, frame := range []*frame{window.renderedFrame, window.nextFrame} {
			if frame.nextInspectorInstanceIDs != nil || frame.inspectorHitboxes != nil {
				t.Fatal("closed inspector has bookkeeping")
			}
		}
	}))
}

func TestWindowInspectorTests_InspectorOnlyTracksItsOpenWindow(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var children []Entity[windowTestInspectorView]
		newRoot := func(_ *Window, cx *Context[windowTestInspectorRoot]) *windowTestInspectorRoot {
			child := cx.New(func(*Context[windowTestInspectorView]) *windowTestInspectorView {
				return &windowTestInspectorView{childWidths: &[]Pixels{}}
			})
			rootChild := child.Clone()
			children = append(children, child, rootChild)
			return &windowTestInspectorRoot{child: rootChild}
		}
		windows := []WindowHandle[windowTestInspectorRoot]{cx.AddWindow(newRoot), cx.AddWindow(newRoot)}
		defer func() {
			for _, c := range children {
				c.Release()
			}
		}()
		for _, w := range windows {
			windowTestAssertClosedInspector(t, w.AnyWindowHandle, cx)
		}
		for range 2 {
			windowTestMust(cx.updateWindow(windows[0].AnyWindowHandle, func(root AnyView, window *Window, cx *App) {
				r, ok := root.Downcast[windowTestInspectorRoot]()
				if !ok {
					t.Fatal("test root")
				}
				childWidths := r.Read(cx).child.Read(cx).childWidths
				window.ToggleInspector(cx)
				window.Draw(cx).Clear(cx)
				if !slices.Equal(*childWidths, []Pixels{Px(10.), Px(10.), Px(10.)}) {
					t.Fatalf("child widths = %v", *childWidths)
				}
				var path *InspectorElementPath
				for _, bucket := range window.renderedFrame.nextInspectorInstanceIDs {
					for _, entry := range bucket {
						if entry.next == 3 {
							p := entry.path
							path = &p
						}
					}
				}
				if path == nil {
					t.Fatal("anonymous siblings share an inspector path")
				}
				selectedID := InspectorElementID{Path: path, InstanceID: 1}
				var position Point[Pixels]
				found := false
				for _, hitbox := range window.renderedFrame.hitboxes {
					if id, ok := window.renderedFrame.inspectorHitboxes[hitbox.ID]; ok && id.Equal(selectedID) {
						position = hitbox.Bounds.Center()
						found = true
						break
					}
				}
				if !found {
					t.Fatal("middle sibling is pickable")
				}
				window.SimulateMouseMove(position, cx)
				window.DispatchEvent(MouseDownEvent{Position: position, Button: MouseButtonLeft, ClickCount: 1}, cx)
				window.DispatchEvent(MouseUpEvent{Position: position, Button: MouseButtonLeft, ClickCount: 1}, cx)
				if window.IsInspectorPicking(cx) {
					t.Fatal("still picking")
				}
				activeID, ok := window.inspector.Unwrap().Read(cx).ActiveElementID()
				if !ok || !activeID.Equal(selectedID) {
					t.Fatalf("active element = %v", activeID)
				}
				window.Draw(cx).Clear(cx)
				window.WithInspectorState[DivInspectorState, struct{}](&selectedID, cx, func(state *Option[DivInspectorState], _ *Window) struct{} {
					s := state.AsMut()
					if s == nil {
						t.Fatal("selected div has style state")
					}
					if s.BaseStyle.Size.Width != Some(Px(10.).ToLength()) {
						t.Fatalf("base width = %v", s.BaseStyle.Size.Width)
					}
					s.BaseStyle.Size.Width = Some(Px(25.).ToLength())
					return struct{}{}
				})
				window.Refresh()
				window.Draw(cx).Clear(cx)
				if !slices.Equal(*childWidths, []Pixels{Px(10.), Px(25.), Px(10.)}) {
					t.Fatalf("child widths after edit = %v", *childWidths)
				}
			}))
			windowTestAssertClosedInspector(t, windows[1].AnyWindowHandle, cx)
			windowTestMust(windows[0].Update(cx, func(_ *windowTestInspectorRoot, window *Window, cx *Context[windowTestInspectorRoot]) {
				window.ToggleInspector(cx.App)
			}))
			windowTestAssertClosedInspector(t, windows[0].AnyWindowHandle, cx)
		}
	})
}
