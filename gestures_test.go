package gpui

import (
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func gestureTouchEvent(id TouchID, phase TouchPhase, x, y float32) TouchEvent {
	return TouchEvent{ID: id, Phase: phase, Position: NewPoint(Px(x), Px(y))}
}

func gesturePt(x, y float32) Point[Pixels] { return NewPoint(Px(x), Px(y)) }

func gestureMs(n int64) time.Duration { return time.Duration(n) * time.Millisecond }

func gestureOnlyScroll(t *testing.T, recognized []recognizedTouchGesture) ScrollEvent {
	t.Helper()
	if len(recognized) != 1 {
		t.Fatalf("expected scroll, got %+v", recognized)
	}
	scroll, ok := recognized[0].(recognizedTouchGestureScroll)
	if !ok {
		t.Fatalf("expected scroll, got %+v", recognized)
	}
	return scroll.ScrollEvent
}

func gestureOnlyTap(t *testing.T, recognized []recognizedTouchGesture) recognizedTouchGestureTap {
	t.Helper()
	if len(recognized) != 1 {
		t.Fatalf("expected tap, got %+v", recognized)
	}
	tap, ok := recognized[0].(recognizedTouchGestureTap)
	if !ok {
		t.Fatalf("expected tap, got %+v", recognized)
	}
	return tap
}

func gestureScrollDelta(s ScrollEvent) Point[Pixels] { return s.Delta.PixelDelta(Px(16.)) }

func gestureExpectAxis(t *testing.T, s OngoingScroll, want Option[Axis]) {
	t.Helper()
	if s.axis != want {
		t.Fatalf("axis = %v, want %v", s.axis, want)
	}
}

func gestureExpectPt(t *testing.T, got, want Point[Pixels]) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGestures_OngoingScrollLocksToDominantAxis(t *testing.T) {
	now := scheduler.InstantNow()
	var ongoingScroll OngoingScroll
	horizontalDelta := gesturePt(10., 2.)
	ongoingScroll.filterAt(&horizontalDelta, TouchPhaseStarted, now)
	gestureExpectAxis(t, ongoingScroll, Some(AxisHorizontal))
	gestureExpectPt(t, horizontalDelta, gesturePt(10., 0.))

	continuedDelta := gesturePt(3., 2.)
	ongoingScroll.filterAt(&continuedDelta, TouchPhaseMoved, now.Add(gestureMs(1)))
	gestureExpectAxis(t, ongoingScroll, Some(AxisHorizontal))
	gestureExpectPt(t, continuedDelta, gesturePt(3., 0.))
}

func TestGestures_OngoingScrollUnlocksWhenDirectionChanges(t *testing.T) {
	now := scheduler.InstantNow()
	var ongoingScroll OngoingScroll
	horizontalDelta := gesturePt(10., 2.)
	ongoingScroll.filterAt(&horizontalDelta, TouchPhaseStarted, now)

	verticalDelta := gesturePt(2., 10.)
	ongoingScroll.filterAt(&verticalDelta, TouchPhaseMoved, now.Add(gestureMs(1)))
	gestureExpectAxis(t, ongoingScroll, None[Axis]())
	gestureExpectPt(t, verticalDelta, gesturePt(2., 10.))
}

func TestGestures_OngoingScrollStartsNewGestureAtTimeoutBoundary(t *testing.T) {
	now := scheduler.InstantNow()
	var ongoingScroll OngoingScroll
	horizontalDelta := gesturePt(10., 2.)
	ongoingScroll.filterAt(&horizontalDelta, TouchPhaseMoved, now)

	verticalDelta := gesturePt(2., 10.)
	ongoingScroll.filterAt(&verticalDelta, TouchPhaseMoved, now.Add(scrollEventSeparation))
	gestureExpectAxis(t, ongoingScroll, Some(AxisVertical))
	gestureExpectPt(t, verticalDelta, gesturePt(0., 10.))
}

func TestGestures_OngoingScrollIgnoresZeroDeltaAndResetsWhenEnded(t *testing.T) {
	now := scheduler.InstantNow()
	var ongoingScroll OngoingScroll
	horizontalDelta := gesturePt(10., 2.)
	ongoingScroll.filterAt(&horizontalDelta, TouchPhaseStarted, now)

	var zeroDelta Point[Pixels]
	ongoingScroll.filterAt(&zeroDelta, TouchPhaseEnded, now.Add(gestureMs(1)))
	gestureExpectAxis(t, ongoingScroll, None[Axis]())

	verticalDelta := gesturePt(2., 3.)
	ongoingScroll.filterAt(&verticalDelta, TouchPhaseMoved, now.Add(gestureMs(2)))
	gestureExpectAxis(t, ongoingScroll, Some(AxisVertical))
	gestureExpectPt(t, verticalDelta, gesturePt(0., 3.))
}

func TestGestures_OngoingScrollIgnoresZeroDeltaMovement(t *testing.T) {
	now := scheduler.InstantNow()
	var ongoingScroll OngoingScroll
	horizontalDelta := gesturePt(10., 2.)
	ongoingScroll.filterAt(&horizontalDelta, TouchPhaseStarted, now)

	var zeroDelta Point[Pixels]
	ongoingScroll.filterAt(&zeroDelta, TouchPhaseMoved, now.Add(scrollEventSeparation))

	verticalDelta := gesturePt(2., 10.)
	ongoingScroll.filterAt(&verticalDelta, TouchPhaseMoved, now.Add(scrollEventSeparation))
	gestureExpectAxis(t, ongoingScroll, Some(AxisVertical))
	gestureExpectPt(t, verticalDelta, gesturePt(0., 10.))
}

func TestGestures_OngoingScrollSupportsMovedOnlyPlatforms(t *testing.T) {
	now := scheduler.InstantNow()
	var ongoingScroll OngoingScroll
	horizontalDelta := gesturePt(10., 2.)
	ongoingScroll.filterAt(&horizontalDelta, TouchPhaseMoved, now)
	gestureExpectAxis(t, ongoingScroll, Some(AxisHorizontal))
	gestureExpectPt(t, horizontalDelta, gesturePt(10., 0.))
}

func gestureHandle(r *touchGestureRecognizer, id TouchID, phase TouchPhase, x, y float32, at scheduler.Instant) []recognizedTouchGesture {
	event := gestureTouchEvent(id, phase, x, y)
	return r.handleEventAt(&event, at)
}

func gestureHandleNow(r *touchGestureRecognizer, id TouchID, phase TouchPhase, x, y float32) []recognizedTouchGesture {
	event := gestureTouchEvent(id, phase, x, y)
	return r.handleEvent(&event)
}

func gestureHandlePredicted(r *touchGestureRecognizer, id TouchID, x, y, px, py float32, at scheduler.Instant) []recognizedTouchGesture {
	event := gestureTouchEvent(id, TouchPhaseMoved, x, y)
	event.PredictedPosition = Some(gesturePt(px, py))
	return r.handleEventAt(&event, at)
}

func TestGestures_TouchWithinSlopResolvesToTap(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	if got := gestureHandle(r, touch, TouchPhaseStarted, 10., 10., now); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	if got := gestureHandle(r, touch, TouchPhaseMoved, 12., 11., now.Add(gestureMs(20))); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	tap := gestureOnlyTap(t, gestureHandle(r, touch, TouchPhaseEnded, 12., 11., now.Add(gestureMs(60))))
	if tap.down.ClickCount != 1 || tap.up.ClickCount != 1 {
		t.Fatalf("tap %+v", tap)
	}
	gestureExpectPt(t, tap.down.Position, gesturePt(12., 11.))
}

func TestGestures_ConsecutiveTapsAccumulateTapCount(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()

	gestureHandle(r, 1, TouchPhaseStarted, 10., 10., now)
	gestureHandle(r, 1, TouchPhaseEnded, 10., 10., now.Add(gestureMs(40)))

	secondDown := now.Add(gestureMs(200))
	gestureHandle(r, 2, TouchPhaseStarted, 14., 10., secondDown)
	if tap := gestureOnlyTap(t, gestureHandle(r, 2, TouchPhaseEnded, 14., 10., secondDown.Add(gestureMs(40)))); tap.down.ClickCount != 2 {
		t.Fatalf("click count %d", tap.down.ClickCount)
	}

	lateDown := secondDown.Add(2 * time.Second)
	gestureHandle(r, 3, TouchPhaseStarted, 14., 10., lateDown)
	if tap := gestureOnlyTap(t, gestureHandle(r, 3, TouchPhaseEnded, 14., 10., lateDown.Add(gestureMs(40)))); tap.down.ClickCount != 1 {
		t.Fatalf("click count %d", tap.down.ClickCount)
	}
}

func TestGestures_TouchBeyondSlopResolvesToPan(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 100., now)

	scroll := gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseMoved, 100., 120., now.Add(gestureMs(16))))
	if scroll.TouchPhase != TouchPhaseStarted {
		t.Fatal("phase")
	}
	gestureExpectPt(t, scroll.Position, gesturePt(100., 100.))
	gestureExpectPt(t, gestureScrollDelta(scroll), gesturePt(0., 20.))

	scroll = gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseMoved, 100., 135., now.Add(gestureMs(32))))
	if scroll.TouchPhase != TouchPhaseMoved {
		t.Fatal("phase")
	}
	gestureExpectPt(t, scroll.Position, gesturePt(100., 100.))
	gestureExpectPt(t, gestureScrollDelta(scroll), gesturePt(0., 15.))

	scroll = gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseEnded, 100., 135., now.Add(gestureMs(48))))
	if scroll.TouchPhase != TouchPhaseEnded {
		t.Fatal("phase")
	}
}

func TestGestures_TouchPanStaysLockedToItsInitialDominantAxis(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 100., now)
	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseMoved, 104., 120., now.Add(gestureMs(16))))), gesturePt(0., 20.))
	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseMoved, 134., 125., now.Add(gestureMs(32))))), gesturePt(0., 5.))
}

func TestGestures_TouchPanLocksToHorizontalAxis(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 100., now)
	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseMoved, 120., 104., now.Add(gestureMs(16))))), gesturePt(20., 0.))
}

func TestGestures_PredictedPositionsLeadThePanButTotalsConvergeOnRelease(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 100., now)

	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandlePredicted(r, touch, 100., 120., 106., 128., now.Add(gestureMs(16))))), gesturePt(0., 28.))
	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandlePredicted(r, touch, 100., 130., 104., 134., now.Add(gestureMs(32))))), gesturePt(0., 6.))

	scroll := gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseEnded, 100., 130., now.Add(gestureMs(120))))
	if scroll.TouchPhase != TouchPhaseEnded {
		t.Fatal("phase")
	}
	if r.hasMomentum() {
		t.Fatal("unexpected momentum")
	}
	gestureExpectPt(t, gestureScrollDelta(scroll), gesturePt(0., -4.))
}

func TestGestures_PredictedPositionsDoNotEmitFalseReversals(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 100., now)

	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandlePredicted(r, touch, 100., 120., 100., 130., now.Add(gestureMs(16))))), gesturePt(0., 30.))
	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandlePredicted(r, touch, 100., 125., 100., 127., now.Add(gestureMs(32))))), Point[Pixels]{})
	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandlePredicted(r, touch, 100., 125., 100., 126., now.Add(gestureMs(40))))), Point[Pixels]{})
	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandlePredicted(r, touch, 100., 132., 100., 136., now.Add(gestureMs(48))))), gesturePt(0., 6.))
	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandlePredicted(r, touch, 100., 124., 100., 140., now.Add(gestureMs(64))))), gesturePt(0., -12.))
}

func TestGestures_PredictedOvershootFoldsIntoTheFlingWithoutScrollingBackwards(t *testing.T) {
	now := scheduler.InstantNow()
	var totalWithPrediction, totalWithoutPrediction float32
	for _, usePrediction := range []bool{true, false} {
		r := newTouchGestureRecognizer(DefaultGestureTuning())
		var total float32
		drain := func(recognized []recognizedTouchGesture, upwardOnly bool) {
			for _, gesture := range recognized {
				scroll, ok := gesture.(recognizedTouchGestureScroll)
				if !ok {
					t.Fatalf("expected scroll, got %+v", gesture)
				}
				delta := gestureScrollDelta(scroll.ScrollEvent).Y
				if upwardOnly && delta > Px(0.) {
					t.Fatalf("content moved backwards by %v during an upward gesture", delta)
				}
				total += float32(delta)
			}
		}

		gestureHandle(r, 1, TouchPhaseStarted, 100., 500., now)
		for step := int64(1); step <= 5; step++ {
			rawY := 500. - float32(step)*40.
			event := gestureTouchEvent(1, TouchPhaseMoved, 100., rawY)
			if usePrediction {
				event.PredictedPosition = Some(gesturePt(100., rawY-25.))
			}
			drain(r.handleEventAt(&event, now.Add(gestureMs(step*16))), usePrediction)
		}
		drain(gestureHandle(r, 1, TouchPhaseEnded, 100., 300., now.Add(gestureMs(90))), usePrediction)
		if !r.hasMomentum() {
			t.Fatal("expected momentum")
		}
		tick := now.Add(gestureMs(91))
		for r.hasMomentum() {
			if gesture, ok := r.tickMomentumAt(tick); ok {
				drain([]recognizedTouchGesture{gesture}, usePrediction)
			}
			tick = tick.Add(gestureMs(16))
		}

		if usePrediction {
			totalWithPrediction = total
		} else {
			totalWithoutPrediction = total
		}
	}
	if absF32(totalWithPrediction-totalWithoutPrediction) >= 0.01 {
		t.Fatalf("totals diverged: %v vs %v", totalWithPrediction, totalWithoutPrediction)
	}
}

func TestGestures_FastReleaseStartsMomentumThatDecaysToAStop(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 300., now)
	for step := int64(1); step <= 5; step++ {
		gestureHandle(r, touch, TouchPhaseMoved, 100., 300.-float32(step)*20., now.Add(gestureMs(step*16)))
	}
	gestureHandle(r, touch, TouchPhaseEnded, 100., 200., now.Add(gestureMs(6*16)))
	if !r.hasMomentum() {
		t.Fatal("expected momentum")
	}

	tick := now.Add(gestureMs(6*16 + 16))
	gesture, ok := r.tickMomentumAt(tick)
	scroll, isScroll := gesture.(recognizedTouchGestureScroll)
	if !ok || !isScroll {
		t.Fatalf("expected momentum scroll, got %+v", gesture)
	}
	if scroll.TouchPhase != TouchPhaseMoved {
		t.Fatal("phase")
	}
	gestureExpectPt(t, scroll.Position, gesturePt(100., 300.))
	delta := gestureScrollDelta(scroll.ScrollEvent)
	if delta.Y >= Px(0.) {
		t.Fatalf("momentum should continue upward, got %v", delta)
	}
	if delta.X.Abs() >= Px(0.001) {
		t.Fatalf("expected no x motion, got %v", delta)
	}

	lastPhase := TouchPhaseMoved
	ticks := 0
	at := tick
	for r.hasMomentum() {
		at = at.Add(gestureMs(16))
		ticks++
		if ticks >= 1000 {
			t.Fatal("momentum never stopped")
		}
		if gesture, ok := r.tickMomentumAt(at); ok {
			if scroll, ok := gesture.(recognizedTouchGestureScroll); ok {
				lastPhase = scroll.TouchPhase
			}
		}
	}
	if lastPhase != TouchPhaseEnded {
		t.Fatal("last phase")
	}
	if _, ok := r.tickMomentumAt(at); ok {
		t.Fatal("tick after stop")
	}
}

func TestGestures_DiagonalReleaseFlingsOnlyOnLockedAxis(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 300., now)
	for step := int64(1); step <= 5; step++ {
		scroll := gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseMoved, 100.+float32(step)*3., 300.-float32(step)*20., now.Add(gestureMs(step*16))))
		if gestureScrollDelta(scroll).X != Px(0.) {
			t.Fatal("x moved")
		}
	}
	gestureHandle(r, touch, TouchPhaseEnded, 115., 200., now.Add(gestureMs(6*16)))
	if !r.hasMomentum() {
		t.Fatal("expected momentum")
	}

	at := now.Add(gestureMs(6 * 16))
	for r.hasMomentum() {
		at = at.Add(gestureMs(16))
		if gesture, ok := r.tickMomentumAt(at); ok {
			if scroll, ok := gesture.(recognizedTouchGestureScroll); ok {
				delta := gestureScrollDelta(scroll.ScrollEvent)
				if delta.X != Px(0.) || delta.Y > Px(0.) {
					t.Fatalf("delta %v", delta)
				}
			}
		}
	}
}

func TestGestures_SlowReleaseDoesNotStartMomentum(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 300., now)
	gestureHandle(r, touch, TouchPhaseMoved, 100., 280., now.Add(gestureMs(16)))
	gestureHandle(r, touch, TouchPhaseMoved, 100., 279., now.Add(gestureMs(500)))
	gestureHandle(r, touch, TouchPhaseEnded, 100., 279., now.Add(gestureMs(600)))
	if r.hasMomentum() {
		t.Fatal("unexpected momentum")
	}
}

func gestureFlingUp(t *testing.T, r *touchGestureRecognizer, now scheduler.Instant) {
	t.Helper()
	gestureHandle(r, 1, TouchPhaseStarted, 100., 300., now)
	for step := int64(1); step <= 3; step++ {
		gestureHandle(r, 1, TouchPhaseMoved, 100., 300.-float32(step)*33., now.Add(gestureMs(step*16)))
	}
	gestureHandle(r, 1, TouchPhaseEnded, 100., 200., now.Add(gestureMs(64)))
	if !r.hasMomentum() {
		t.Fatal("expected momentum")
	}
}

func TestGestures_NewTouchInterruptsMomentum(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	gestureFlingUp(t, r, now)

	recognized := gestureHandle(r, 2, TouchPhaseStarted, 100., 200., now.Add(gestureMs(200)))
	if r.hasMomentum() {
		t.Fatal("momentum not interrupted")
	}
	if len(recognized) != 2 {
		t.Fatalf("expected closing and opening scrolls, got %+v", recognized)
	}
	closing, ok1 := recognized[0].(recognizedTouchGestureScroll)
	opening, ok2 := recognized[1].(recognizedTouchGestureScroll)
	if !ok1 || !ok2 {
		t.Fatalf("expected closing and opening scrolls, got %+v", recognized)
	}
	if closing.TouchPhase != TouchPhaseEnded || !gestureScrollDelta(closing.ScrollEvent).IsZero() {
		t.Fatalf("closing %+v", closing)
	}
	if opening.TouchPhase != TouchPhaseStarted || !gestureScrollDelta(opening.ScrollEvent).IsZero() {
		t.Fatalf("opening %+v", opening)
	}
}

func TestGestures_CatchingAFlingPansImmediatelyAndNeverTaps(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	gestureFlingUp(t, r, now)

	gestureHandle(r, 2, TouchPhaseStarted, 100., 200., now.Add(gestureMs(200)))

	scroll := gestureOnlyScroll(t, gestureHandle(r, 2, TouchPhaseMoved, 100., 197., now.Add(gestureMs(216))))
	if scroll.TouchPhase != TouchPhaseMoved {
		t.Fatal("phase")
	}
	gestureExpectPt(t, gestureScrollDelta(scroll), gesturePt(0., -3.))

	scroll = gestureOnlyScroll(t, gestureHandle(r, 2, TouchPhaseEnded, 100., 197., now.Add(gestureMs(232))))
	if scroll.TouchPhase != TouchPhaseEnded {
		t.Fatal("phase")
	}
}

func TestGestures_CatchingAFlingTakesTheAxisFromTheNewTouch(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()

	gestureHandle(r, 1, TouchPhaseStarted, 300., 300., now)
	for step := int64(1); step <= 3; step++ {
		gestureHandle(r, 1, TouchPhaseMoved, 300.-float32(step)*33., 300., now.Add(gestureMs(step*16)))
	}
	gestureHandle(r, 1, TouchPhaseEnded, 200., 300., now.Add(gestureMs(64)))
	if !r.hasMomentum() {
		t.Fatal("expected momentum")
	}

	gestureHandle(r, 2, TouchPhaseStarted, 200., 300., now.Add(gestureMs(200)))
	scroll := gestureOnlyScroll(t, gestureHandle(r, 2, TouchPhaseMoved, 201., 280., now.Add(gestureMs(216))))
	if scroll.TouchPhase != TouchPhaseMoved {
		t.Fatal("phase")
	}
	gestureExpectPt(t, gestureScrollDelta(scroll), gesturePt(0., -20.))

	gestureExpectPt(t, gestureScrollDelta(gestureOnlyScroll(t, gestureHandle(r, 2, TouchPhaseMoved, 205., 260., now.Add(gestureMs(232))))), gesturePt(0., -20.))
}

func TestGestures_CancelledPanEmitsCancelledScrollAndNoTap(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()
	touch := TouchID(1)

	gestureHandle(r, touch, TouchPhaseStarted, 100., 100., now)
	gestureHandle(r, touch, TouchPhaseMoved, 100., 150., now.Add(gestureMs(16)))
	scroll := gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseCancelled, 100., 150., now.Add(gestureMs(32))))
	if scroll.TouchPhase != TouchPhaseCancelled {
		t.Fatal("phase")
	}
	if r.hasMomentum() {
		t.Fatal("unexpected momentum")
	}

	r = newTouchGestureRecognizer(DefaultGestureTuning())
	gestureHandle(r, touch, TouchPhaseStarted, 100., 100., now)
	if got := gestureHandle(r, touch, TouchPhaseCancelled, 100., 102., now.Add(gestureMs(16))); len(got) != 0 {
		t.Fatal("cancelled tap must not click")
	}
}

func TestGestures_ConcurrentTouchesAreIgnoredWhileOneIsActive(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	now := scheduler.InstantNow()

	gestureHandle(r, 1, TouchPhaseStarted, 100., 100., now)
	if got := gestureHandle(r, 2, TouchPhaseStarted, 200., 200., now.Add(gestureMs(8))); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	if got := gestureHandle(r, 2, TouchPhaseMoved, 200., 300., now.Add(gestureMs(16))); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	if got := gestureHandle(r, 2, TouchPhaseEnded, 200., 300., now.Add(gestureMs(24))); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}

	if scroll := gestureOnlyScroll(t, gestureHandle(r, 1, TouchPhaseMoved, 100., 150., now.Add(gestureMs(32)))); scroll.TouchPhase != TouchPhaseStarted {
		t.Fatal("phase")
	}
}

func frictionSplineBezierTimeAndPosition(parameter float32) (float32, float32) {
	coefficient := 3. * parameter * (1. - parameter)
	cubed := parameter * parameter * parameter
	return coefficient*((1.-parameter)*frictionSplineP1+parameter*frictionSplineP2) + cubed,
		coefficient*((1.-parameter)*frictionSplineStartTension+parameter) + cubed
}

func TestGestures_SplinePositionTableMatchesTheBezierCurve(t *testing.T) {
	samples := frictionSplinePosition()
	if absF32(samples[0]) >= 1e-4 {
		t.Fatalf("sample 0 = %v", samples[0])
	}
	if samples[100] != 1. {
		t.Fatal("sample 100")
	}
	for i := 0; i+1 < len(samples); i++ {
		if !(samples[i] < samples[i+1]) {
			t.Fatal("table must be strictly increasing")
		}
	}
	for i := range 100 {
		storedPosition := samples[i]
		alpha := float32(i) / 100.
		var lower, upper float32 = 0, 1
		for range 50 {
			middle := (lower + upper) / 2.
			tm, _ := frictionSplineBezierTimeAndPosition(middle)
			if tm > alpha {
				upper = middle
			} else {
				lower = middle
			}
		}
		tm, position := frictionSplineBezierTimeAndPosition((lower + upper) / 2.)
		if absF32(tm-alpha) >= 1e-4 {
			t.Fatalf("sample %d: time %v != %v", i, tm, alpha)
		}
		if absF32(position-storedPosition) >= 1e-3 {
			t.Fatalf("sample %d: position %v != stored %v", i, position, storedPosition)
		}
	}
}

func TestGestures_FlingCurvesAreSaneForBothPhysics(t *testing.T) {
	for _, physics := range []ScrollPhysics{ScrollPhysicsIos(), ScrollPhysicsAndroid()} {
		slow := physics.flingDuration(500.)
		fast := physics.flingDuration(4000.)
		if slow <= 0 {
			t.Fatalf("%+v", physics)
		}
		if fast <= slow {
			t.Fatalf("faster flings must coast longer: %+v", physics)
		}

		halfway := physics.flingDistance(4000., fast/2)
		total := physics.flingDistance(4000., fast)
		if !(halfway > 0. && halfway < total) {
			t.Fatalf("%+v", physics)
		}
		if physics.flingDistance(4000., fast*2) != total {
			t.Fatalf("distance must not grow past the fling duration: %+v", physics)
		}
		if !(physics.flingDistance(4000., fast) > physics.flingDistance(500., slow)) {
			t.Fatalf("faster flings must travel further: %+v", physics)
		}
	}
}

func TestGestures_MomentumIsFrameRateIndependent(t *testing.T) {
	totalDistanceWithTickLength := func(tick time.Duration) float32 {
		tuning := DefaultGestureTuning()
		tuning.ScrollPhysics = ScrollPhysicsAndroid()
		r := newTouchGestureRecognizer(tuning)
		now := scheduler.InstantNow()
		gestureHandle(r, 1, TouchPhaseStarted, 100., 500., now)
		for step := int64(1); step <= 3; step++ {
			gestureHandle(r, 1, TouchPhaseMoved, 100., 500.-float32(step)*40., now.Add(gestureMs(step*16)))
		}
		gestureHandle(r, 1, TouchPhaseEnded, 100., 380., now.Add(gestureMs(64)))
		if !r.hasMomentum() {
			t.Fatal("expected momentum")
		}

		var total float32
		at := now.Add(gestureMs(64))
		guard := 0
		for r.hasMomentum() {
			at = at.Add(tick)
			guard++
			if guard >= 10_000 {
				t.Fatal("momentum never stopped")
			}
			if gesture, ok := r.tickMomentumAt(at); ok {
				if scroll, ok := gesture.(recognizedTouchGestureScroll); ok {
					total += float32(gestureScrollDelta(scroll.ScrollEvent).Y)
				}
			}
		}
		return total
	}

	smooth := totalDistanceWithTickLength(gestureMs(16))
	stalled := totalDistanceWithTickLength(10 * time.Second)
	if absF32(smooth-stalled) >= 0.01 {
		t.Fatalf("expected identical fling distance, got %v vs %v", smooth, stalled)
	}
}

func TestGestures_FlickVelocityReflectsReleaseSpeedNotWindowAverage(t *testing.T) {
	var tracker velocityTracker
	start := scheduler.InstantNow()
	for step := int64(0); step <= 6; step++ {
		tm := float32(step) * 0.016
		tracker.push(start.Add(gestureMs(step*16)), gesturePt(0., 1000.*tm*tm))
	}
	velocity := tracker.velocity()
	var releaseSpeed float32 = 2. * 1000. * 0.096
	if absF32(velocity.Y-releaseSpeed) >= 1. {
		t.Fatalf("expected ≈%v px/s at release, got %v px/s", releaseSpeed, velocity.Y)
	}
	if velocity.X != 0. {
		t.Fatalf("x velocity %v", velocity.X)
	}
}

func TestGestures_SamplesBeforeAPauseDoNotContributeVelocity(t *testing.T) {
	var tracker velocityTracker
	start := scheduler.InstantNow()
	tracker.push(start, gesturePt(0., 0.))
	tracker.push(start.Add(gestureMs(16)), gesturePt(0., 50.))
	tracker.push(start.Add(gestureMs(80)), gesturePt(0., 52.))
	tracker.push(start.Add(gestureMs(96)), gesturePt(0., 54.))
	if velocity := tracker.velocity(); velocity.Y >= 200. {
		t.Fatalf("pre-pause motion leaked into the estimate: %v px/s", velocity.Y)
	}
}

func TestGestures_ClaimedTouchDragEmitsPhasedStreamWithoutPanOrTap(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	touch := TouchID(1)
	now := scheduler.InstantNow()
	gestureHandle(r, touch, TouchPhaseStarted, 10., 20., now)
	gesture, ok := r.offerTouchDrag(touch)
	started, isDrag := gesture.(recognizedTouchGestureTouchDrag)
	if !ok || !isDrag {
		t.Fatal("expected touch drag")
	}
	if started.Phase != TouchPhaseStarted {
		t.Fatal("phase")
	}
	gestureExpectPt(t, started.StartPosition, gesturePt(10., 20.))
	r.resolveTouchDrag(true)

	moved := gestureHandle(r, touch, TouchPhaseMoved, 40., 50., now.Add(gestureMs(10)))
	if len(moved) != 1 {
		t.Fatalf("expected moved touch drag, got %+v", moved)
	}
	drag, ok := moved[0].(recognizedTouchGestureTouchDrag)
	if !ok || drag.Phase != TouchPhaseMoved {
		t.Fatalf("expected moved touch drag, got %+v", moved)
	}
	gestureExpectPt(t, drag.Position, gesturePt(40., 50.))

	ended := gestureHandle(r, touch, TouchPhaseEnded, 45., 55., now.Add(gestureMs(20)))
	if len(ended) != 1 {
		t.Fatalf("expected ended touch drag, got %+v", ended)
	}
	drag, ok = ended[0].(recognizedTouchGestureTouchDrag)
	if !ok || drag.Phase != TouchPhaseEnded {
		t.Fatalf("expected ended touch drag, got %+v", ended)
	}
	gestureExpectPt(t, drag.Position, gesturePt(45., 55.))
}

func TestGestures_UnclaimedTouchDragRemainsAPanCandidate(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	touch := TouchID(1)
	now := scheduler.InstantNow()
	gestureHandle(r, touch, TouchPhaseStarted, 0., 0., now)
	if _, ok := r.offerTouchDrag(touch); !ok {
		t.Fatal("no offer")
	}
	r.resolveTouchDrag(false)

	if scroll := gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseMoved, 20., 0., now.Add(gestureMs(10)))); scroll.TouchPhase != TouchPhaseStarted {
		t.Fatal("phase")
	}
}

func TestGestures_ClaimedLongPressEmitsPhasedStreamWithoutTap(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	touch := TouchID(1)
	now := scheduler.InstantNow()
	gestureHandle(r, touch, TouchPhaseStarted, 10., 20., now)
	gesture, ok := r.offerLongPress(touch)
	started, isPress := gesture.(recognizedTouchGestureLongPress)
	if !ok || !isPress {
		t.Fatal("expected long press")
	}
	if started.Phase != TouchPhaseStarted {
		t.Fatal("phase")
	}
	gestureExpectPt(t, started.StartPosition, gesturePt(10., 20.))
	r.resolveLongPress(true)

	moved := gestureHandle(r, touch, TouchPhaseMoved, 12., 21., now.Add(gestureMs(510)))
	if len(moved) != 1 {
		t.Fatalf("expected moved long press, got %+v", moved)
	}
	if press, ok := moved[0].(recognizedTouchGestureLongPress); !ok || press.Phase != TouchPhaseMoved {
		t.Fatalf("expected moved long press, got %+v", moved)
	}

	ended := gestureHandle(r, touch, TouchPhaseEnded, 12., 21., now.Add(gestureMs(520)))
	if len(ended) != 1 {
		t.Fatalf("expected ended long press, got %+v", ended)
	}
	if press, ok := ended[0].(recognizedTouchGestureLongPress); !ok || press.Phase != TouchPhaseEnded {
		t.Fatalf("expected ended long press, got %+v", ended)
	}
}

func TestGestures_UnclaimedLongPressRemainsATapCandidate(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	touch := TouchID(1)
	now := scheduler.InstantNow()
	gestureHandle(r, touch, TouchPhaseStarted, 10., 20., now)
	if _, ok := r.offerLongPress(touch); !ok {
		t.Fatal("no offer")
	}
	r.resolveLongPress(false)

	gestureOnlyTap(t, gestureHandle(r, touch, TouchPhaseEnded, 10., 20., now.Add(gestureMs(510))))
}

func TestGestures_UnclaimedLongPressCanStillBecomeAPan(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	touch := TouchID(1)
	now := scheduler.InstantNow()
	gestureHandle(r, touch, TouchPhaseStarted, 0., 0., now)
	if _, ok := r.offerLongPress(touch); !ok {
		t.Fatal("no offer")
	}
	r.resolveLongPress(false)

	if scroll := gestureOnlyScroll(t, gestureHandle(r, touch, TouchPhaseMoved, 20., 0., now.Add(gestureMs(510)))); scroll.TouchPhase != TouchPhaseStarted {
		t.Fatal("phase")
	}
}

func TestGestures_LongPressOfferIsOneShotAndSpecificToPendingTouch(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	touch := TouchID(1)
	gestureHandleNow(r, touch, TouchPhaseStarted, 0., 0.)

	if got := gestureHandleNow(r, 2, TouchPhaseMoved, 20., 0.); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	if _, ok := r.offerLongPress(2); ok {
		t.Fatal("offered to other touch")
	}
	if _, ok := r.offerLongPress(touch); !ok {
		t.Fatal("no offer")
	}
	if _, ok := r.offerLongPress(touch); ok {
		t.Fatal("offered twice")
	}
}

func TestGestures_LongPressCannotBeOfferedAfterPendingTouchResolves(t *testing.T) {
	for _, phase := range []TouchPhase{TouchPhaseEnded, TouchPhaseCancelled, TouchPhaseMoved} {
		r := newTouchGestureRecognizer(DefaultGestureTuning())
		touch := TouchID(1)
		now := scheduler.InstantNow()
		gestureHandle(r, touch, TouchPhaseStarted, 0., 0., now)
		var position float32
		if phase == TouchPhaseMoved {
			position = 20.
		}
		gestureHandle(r, touch, phase, position, 0., now.Add(gestureMs(10)))
		if _, ok := r.offerLongPress(touch); ok {
			t.Fatalf("%v: offered after resolve", phase)
		}
	}
}

func TestGestures_ClaimedLongPressEmitsCancelledForItsTouchOnly(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	touch := TouchID(1)
	gestureHandleNow(r, touch, TouchPhaseStarted, 4., 5.)
	if _, ok := r.offerLongPress(touch); !ok {
		t.Fatal("no offer")
	}
	r.resolveLongPress(true)

	if got := gestureHandleNow(r, 2, TouchPhaseCancelled, 9., 9.); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	cancelled := gestureHandleNow(r, touch, TouchPhaseCancelled, 6., 7.)
	if len(cancelled) != 1 {
		t.Fatalf("expected cancelled long press, got %+v", cancelled)
	}
	press, ok := cancelled[0].(recognizedTouchGestureLongPress)
	if !ok || press.Phase != TouchPhaseCancelled {
		t.Fatalf("expected cancelled long press, got %+v", cancelled)
	}
	gestureExpectPt(t, press.StartPosition, gesturePt(4., 5.))
	gestureExpectPt(t, press.Position, gesturePt(6., 7.))
}

func TestGestures_UnrelatedTouchCannotEndOrCancelPendingTouch(t *testing.T) {
	for _, phase := range []TouchPhase{TouchPhaseEnded, TouchPhaseCancelled} {
		r := newTouchGestureRecognizer(DefaultGestureTuning())
		touch := TouchID(1)
		gestureHandleNow(r, touch, TouchPhaseStarted, 4., 5.)

		if got := gestureHandleNow(r, 2, phase, 9., 9.); len(got) != 0 {
			t.Fatalf("got %+v", got)
		}
		if _, ok := r.offerLongPress(touch); !ok {
			t.Fatal("no offer")
		}
	}
}

func TestGestures_CompletedTouchIDCannotClaimReplacementTouch(t *testing.T) {
	r := newTouchGestureRecognizer(DefaultGestureTuning())
	completedTouch := TouchID(1)
	replacementTouch := TouchID(2)
	gestureHandleNow(r, completedTouch, TouchPhaseStarted, 0., 0.)
	gestureHandleNow(r, completedTouch, TouchPhaseCancelled, 0., 0.)
	gestureHandleNow(r, replacementTouch, TouchPhaseStarted, 5., 5.)

	if _, ok := r.offerLongPress(completedTouch); ok {
		t.Fatal("completed touch claimed")
	}
	if _, ok := r.offerLongPress(replacementTouch); !ok {
		t.Fatal("no offer for replacement")
	}
}
