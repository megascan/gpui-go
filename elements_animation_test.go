package gpui

import (
	"math"
	"slices"
	"testing"
	"time"
)

type animationTestView struct {
	renderedDeltas *[]float32
	animations     []Animation
}

type animationTestSyncedView struct {
	showSecond   bool
	firstDeltas  *[]float32
	secondDeltas *[]float32
}

type animationTestSpringView struct {
	target         Pixels
	initial        Option[Pixels]
	playback       SpringPlayback
	renderedValues *[]Pixels
}

func (v *animationTestSpringView) Render(_ *Window, _ *Context[animationTestSpringView]) IntoElement {
	renderedValues := v.renderedValues
	animation := NewSpringAnimation(NewSpringConfig(100.0, 2.0, 1.0)).
		To(v.target).
		WithEpsilon(0.01).
		Playback(v.playback)
	if initial, ok := v.initial.Get(); ok {
		animation = animation.From(initial)
	}
	return WithSpring(NewDiv(), "spring-animation", animation, func(this *Div, value Pixels) *Div {
		*renderedValues = append(*renderedValues, value)
		return this.Left(value)
	})
}

func (v *animationTestSyncedView) Render(_ *Window, _ *Context[animationTestSyncedView]) IntoElement {
	recordDeltas := func(deltas *[]float32) func(*Div, float32) *Div {
		return func(this *Div, delta float32) *Div {
			*deltas = append(*deltas, delta)
			return this
		}
	}
	return NewDiv().
		SizeFull().
		Child(WithAnimation(NewDiv(), "first-synced-animation", NewAnimation(time.Second).RepeatSynced(), recordDeltas(v.firstDeltas))).
		When(v.showSecond, func(this *Div) *Div {
			return this.Child(WithAnimation(NewDiv(), "second-synced-animation", NewAnimation(time.Second).RepeatSynced(), recordDeltas(v.secondDeltas)))
		})
}

func (v *animationTestView) Render(_ *Window, _ *Context[animationTestView]) IntoElement {
	renderedDeltas := v.renderedDeltas
	return NewDiv().SizeFull().Child(WithAnimations(NewDiv(), "repeating-animation", slices.Clone(v.animations), func(this *Div, _ int, delta float32) *Div {
		*renderedDeltas = append(*renderedDeltas, delta)
		return this
	}))
}

func animationTestOpenWindow(cx *TestAppContext) (*[]float32, WindowHandle[animationTestView]) {
	return animationTestOpenWindowWithAnimations(cx, []Animation{animationTestOneSecond().Repeat()})
}

func animationTestOneSecond() Animation {
	return NewAnimation(time.Second)
}

func animationTestOpenWindowWithAnimations(cx *TestAppContext, animations []Animation) (*[]float32, WindowHandle[animationTestView]) {
	renderedDeltas := &[]float32{}
	window := cx.OpenWindow(NewSize(Px(100.), Px(100.)), func(*Window, *Context[animationTestView]) *animationTestView {
		return &animationTestView{renderedDeltas: renderedDeltas, animations: animations}
	})
	cx.RunUntilParked()
	return renderedDeltas, window
}

func animationTestSimulateNextFrame[V any](window WindowHandle[V], cx *TestAppContext) int {
	callbackCount, err := window.UpdateR(cx, func(_ *V, window *Window, cx *Context[V]) int {
		return window.SimulateNextFrame(cx.App)
	})
	windowTestMust(err)
	cx.RunUntilParked()
	return callbackCount
}

func animationTestOpenSpringWindow(cx *TestAppContext, target Pixels, initial Option[Pixels]) (*[]Pixels, WindowHandle[animationTestSpringView]) {
	renderedValues := &[]Pixels{}
	window := cx.OpenWindow(NewSize(Px(100.0), Px(100.0)), func(*Window, *Context[animationTestSpringView]) *animationTestSpringView {
		return &animationTestSpringView{target: target, initial: initial, playback: SpringPlaybackRunning, renderedValues: renderedValues}
	})
	cx.RunUntilParked()
	return renderedValues, window
}

func animationTestUpdateSpring(cx *TestAppContext, window WindowHandle[animationTestSpringView], update func(view *animationTestSpringView)) {
	windowTestMust(window.Update(cx, func(view *animationTestSpringView, _ *Window, cx *Context[animationTestSpringView]) {
		update(view)
		cx.Notify()
	}))
	cx.RunUntilParked()
}

func animationTestLast[T any](values *[]T) T {
	return (*values)[len(*values)-1]
}

func TestAnimation_AnimationParent(t *testing.T) {
	WithAnimation(NewDiv().ID("id"), "animation", NewAnimation(time.Second), func(el *Stateful[*Div], _ float32) *Stateful[*Div] {
		return el
	}).Child(NewDiv())
}

func TestAnimation_SpringAnimationParent(t *testing.T) {
	WithSpring(NewDiv().ID("id"), "spring-animation", NewSpringAnimation(NewSpringConfig(100.0, 10.0, 1.0)).To(Px(10.0)).From(Px(0.0)), func(element *Stateful[*Div], value Pixels) *Stateful[*Div] {
		return element.Left(value)
	}).Child(NewDiv())
}

func TestAnimation_SpringAnimationPreservesVelocityWhenRetargeted(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		renderedValues, window := animationTestOpenSpringWindow(cx, Px(0.0), None[Pixels]())
		if !slices.Equal(*renderedValues, []Pixels{Px(0.0)}) {
			t.Fatalf("values = %v", *renderedValues)
		}

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) { view.target = Px(100.0) })

		cx.Executor().AdvanceClock(50 * time.Millisecond)
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}
		valueBeforeRetargeting := animationTestLast(renderedValues)
		if !(valueBeforeRetargeting > Px(0.0)) || !(valueBeforeRetargeting < Px(100.0)) {
			t.Fatalf("value before retargeting = %v", valueBeforeRetargeting)
		}

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) { view.target = Px(0.0) })

		cx.Executor().AdvanceClock(5 * time.Millisecond)
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}
		valueAfterRetargeting := animationTestLast(renderedValues)
		if !(valueAfterRetargeting > valueBeforeRetargeting) {
			t.Fatalf("value after retargeting = %v, before = %v", valueAfterRetargeting, valueBeforeRetargeting)
		}
	})
}

func TestAnimation_PausedSpringResumesWithItsVelocity(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		renderedValues, window := animationTestOpenSpringWindow(cx, Px(0.0), None[Pixels]())

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) { view.target = Px(100.0) })
		cx.Executor().AdvanceClock(50 * time.Millisecond)
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) {
			view.target = Px(0.0)
			view.playback = SpringPlaybackPaused
		})
		pausedValue := animationTestLast(renderedValues)

		cx.Executor().AdvanceClock(500 * time.Millisecond)
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}
		if got := animationTestLast(renderedValues); got != pausedValue {
			t.Fatalf("paused value = %v, want %v", got, pausedValue)
		}
		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) { view.playback = SpringPlaybackRunning })
		cx.Executor().AdvanceClock(5 * time.Millisecond)
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}
		if got := animationTestLast(renderedValues); !(got > pausedValue) {
			t.Fatalf("resumed value = %v, paused = %v", got, pausedValue)
		}
	})
}

func TestAnimation_StoppedSpringResumesWithoutVelocity(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		renderedValues, window := animationTestOpenSpringWindow(cx, Px(0.0), None[Pixels]())

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) { view.target = Px(1_000_000.0) })
		cx.Executor().AdvanceClock(50 * time.Millisecond)
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) {
			view.target = Px(0.0)
			view.playback = SpringPlaybackStopped
		})
		stoppedValue := animationTestLast(renderedValues)

		cx.Executor().AdvanceClock(500 * time.Millisecond)
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}
		if got := animationTestLast(renderedValues); got != stoppedValue {
			t.Fatalf("stopped value = %v, want %v", got, stoppedValue)
		}
		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) {
			view.target = stoppedValue
			view.playback = SpringPlaybackRunning
		})
		if got := animationTestLast(renderedValues); got != stoppedValue {
			t.Fatalf("resumed value = %v, want %v", got, stoppedValue)
		}
		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}
	})
}

func TestAnimation_CancelledAndCompletedSpringsResolveTheirEndpoints(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		renderedValues, window := animationTestOpenSpringWindow(cx, Px(100.0), Some(Px(20.0)))
		if !slices.Equal(*renderedValues, []Pixels{Px(20.0)}) {
			t.Fatalf("values = %v", *renderedValues)
		}

		cx.Executor().AdvanceClock(50 * time.Millisecond)
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}
		if got := animationTestLast(renderedValues); !(got > Px(20.0)) {
			t.Fatalf("value = %v", got)
		}

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) { view.playback = SpringPlaybackCancelled })
		if got := animationTestLast(renderedValues); got != Px(20.0) {
			t.Fatalf("cancelled value = %v", got)
		}
		if animationTestSimulateNextFrame(window, cx) <= 0 {
			t.Fatal("no frame callback")
		}
		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}

		animationTestUpdateSpring(cx, window, func(view *animationTestSpringView) { view.playback = SpringPlaybackCompleted })
		if got := animationTestLast(renderedValues); got != Px(100.0) {
			t.Fatalf("completed value = %v", got)
		}
		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}
	})
}

func TestAnimation_SpringAnimationRespectsReducedMotion(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *App) { cx.SetReduceMotion(true) })
		renderedValues, window := animationTestOpenSpringWindow(cx, Px(100.0), None[Pixels]())

		if !slices.Equal(*renderedValues, []Pixels{Px(100.0)}) {
			t.Fatalf("values = %v", *renderedValues)
		}
		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}
	})
}

func TestAnimation_RepeatingAnimationSchedulesAnimationFrames(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		renderedDeltas, window := animationTestOpenWindow(cx)

		if len(*renderedDeltas) != 1 {
			t.Fatalf("deltas = %v", *renderedDeltas)
		}

		for expectedFrames := 2; expectedFrames <= 3; expectedFrames++ {
			if got := animationTestSimulateNextFrame(window, cx); got != 1 {
				t.Fatalf("frame callbacks = %d, want 1", got)
			}
			if len(*renderedDeltas) != expectedFrames {
				t.Fatalf("deltas = %v, want %d", *renderedDeltas, expectedFrames)
			}
		}
	})
}

func TestAnimation_MaxFpsSchedulesTimerDrivenFrames(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		capped := animationTestOneSecond().RepeatSynced().WithMaxFps(10.0)
		renderedDeltas, window := animationTestOpenWindowWithAnimations(cx, []Animation{capped})

		assertDeltasApproxEq := func(expected ...float32) {
			t.Helper()
			actual := *renderedDeltas
			if len(actual) != len(expected) {
				t.Fatalf("deltas: %v", actual)
			}
			for i := range actual {
				if math.Abs(float64(actual[i]-expected[i])) >= 1e-2 {
					t.Fatalf("expected %v, got %v", expected[i], actual[i])
				}
			}
		}

		assertDeltasApproxEq(0.0)

		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}
		assertDeltasApproxEq(0.0)

		cx.Executor().AdvanceClock(105 * time.Millisecond)
		cx.RunUntilParked()
		assertDeltasApproxEq(0.0, 0.105)

		cx.Executor().AdvanceClock(105 * time.Millisecond)
		cx.RunUntilParked()
		assertDeltasApproxEq(0.0, 0.105, 0.21)
	})
}

func TestAnimation_SyncedStepsRenderOnceEach(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		uptime := 300*24*time.Hour + 30*time.Millisecond
		cx.Executor().AdvanceClock(uptime)
		sevenths := animationTestOneSecond().RepeatSynced().WithSteps(7)
		renderedDeltas, window := animationTestOpenWindowWithAnimations(cx, []Animation{sevenths})
		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}

		windowTestMust(window.Update(cx, func(_ *animationTestView, _ *Window, cx *Context[animationTestView]) { cx.Notify() }))
		cx.Executor().AdvanceClock(time.Second)
		cx.RunUntilParked()
		var steps []float32
		for _, step := range []int{0, 0, 1, 2, 3, 4, 5, 6, 0} {
			steps = append(steps, float32(step)/7.0)
		}
		if !slices.Equal(*renderedDeltas, steps) {
			t.Fatalf("deltas = %v, want %v", *renderedDeltas, steps)
		}
	})
}

func TestAnimation_StepsFinishAndRepeatOnTime(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		quarters := animationTestOneSecond().WithSteps(4)
		chain := []Animation{quarters.WithMaxFps(0.1), animationTestOneSecond()}
		chained, window := animationTestOpenWindowWithAnimations(cx, chain)
		repeating, _ := animationTestOpenWindowWithAnimations(cx, []Animation{quarters.Repeat()})

		cx.Executor().AdvanceClock(1500 * time.Millisecond)
		cx.RunUntilParked()
		if want := []float32{0.0, 0.25, 0.5, 0.75, 1.0}; !slices.Equal(*chained, want) {
			t.Fatalf("chained = %v, want %v", *chained, want)
		}
		if want := []float32{0.0, 0.25, 0.5, 0.75, 0.0, 0.25, 0.5}; !slices.Equal(*repeating, want) {
			t.Fatalf("repeating = %v, want %v", *repeating, want)
		}

		if got := animationTestSimulateNextFrame(window, cx); got != 1 {
			t.Fatalf("frame callbacks = %d, want 1", got)
		}
		if got := animationTestLast(chained); got != 0.5 {
			t.Fatalf("last chained = %v, want 0.5", got)
		}
	})
}

func TestAnimation_SoonerStepReplacesThePendingOne(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		slow := []Animation{animationTestOneSecond().Repeat().WithSteps(1)}
		renderedDeltas, window := animationTestOpenWindowWithAnimations(cx, slow)

		cx.Executor().AdvanceClock(100 * time.Millisecond)
		fast := []Animation{animationTestOneSecond().Repeat().WithSteps(10)}
		windowTestMust(window.Update(cx, func(view *animationTestView, _ *Window, cx *Context[animationTestView]) {
			view.animations = fast
			cx.Notify()
		}))
		cx.Executor().AdvanceClock(100 * time.Millisecond)
		windowTestMust(window.Update(cx, func(_ *animationTestView, _ *Window, cx *Context[animationTestView]) { cx.Notify() }))
		cx.Executor().AdvanceClock(100 * time.Millisecond)
		cx.RunUntilParked()
		if want := []float32{0.0, 0.1, 0.2, 0.2, 0.3}; !slices.Equal(*renderedDeltas, want) {
			t.Fatalf("deltas = %v, want %v", *renderedDeltas, want)
		}
	})
}

func TestAnimation_SyncedAnimationsSharePhaseAcrossElements(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		firstDeltas := &[]float32{}
		secondDeltas := &[]float32{}
		window := cx.OpenWindow(NewSize(Px(100.), Px(100.)), func(*Window, *Context[animationTestSyncedView]) *animationTestSyncedView {
			return &animationTestSyncedView{showSecond: false, firstDeltas: firstDeltas, secondDeltas: secondDeltas}
		})
		cx.RunUntilParked()

		if !slices.Equal(*firstDeltas, []float32{0.0}) {
			t.Fatalf("first deltas = %v", *firstDeltas)
		}

		cx.Executor().AdvanceClock(250 * time.Millisecond)
		animationTestSimulateNextFrame(window, cx)
		if !slices.Equal(*firstDeltas, []float32{0.0, 0.25}) {
			t.Fatalf("first deltas = %v", *firstDeltas)
		}

		windowTestMust(window.Update(cx, func(view *animationTestSyncedView, _ *Window, cx *Context[animationTestSyncedView]) {
			view.showSecond = true
			cx.Notify()
		}))
		cx.RunUntilParked()
		cx.Executor().AdvanceClock(250 * time.Millisecond)
		animationTestSimulateNextFrame(window, cx)

		if got := animationTestLast(secondDeltas); got != 0.5 {
			t.Fatalf("second delta = %v, want 0.5", got)
		}
		if animationTestLast(firstDeltas) != animationTestLast(secondDeltas) {
			t.Fatalf("first %v, second %v", animationTestLast(firstDeltas), animationTestLast(secondDeltas))
		}
		for _, delta := range *secondDeltas {
			if !(delta > 0.0) {
				t.Fatalf("second deltas = %v", *secondDeltas)
			}
		}

		cx.Executor().AdvanceClock(2250 * time.Millisecond)
		animationTestSimulateNextFrame(window, cx)
		if got := animationTestLast(firstDeltas); got != 0.75 {
			t.Fatalf("first delta = %v, want 0.75", got)
		}

		cx.Executor().AdvanceClock(300*24*time.Hour + 500*time.Millisecond)
		animationTestSimulateNextFrame(window, cx)
		if got := animationTestLast(firstDeltas); got != 0.25 {
			t.Fatalf("first delta = %v, want 0.25", got)
		}
	})
}

func TestAnimation_ReduceMotionRendersSingleStaticFrame(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *App) { cx.SetReduceMotion(true) })
		renderedDeltas, window := animationTestOpenWindow(cx)

		if !slices.Equal(*renderedDeltas, []float32{0.0}) {
			t.Fatalf("deltas = %v", *renderedDeltas)
		}

		if got := animationTestSimulateNextFrame(window, cx); got != 0 {
			t.Fatalf("frame callbacks = %d, want 0", got)
		}
		if !slices.Equal(*renderedDeltas, []float32{0.0}) {
			t.Fatalf("deltas = %v", *renderedDeltas)
		}
	})
}
