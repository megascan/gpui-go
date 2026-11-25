package gpui

import (
	"fmt"
	"math"
	"math/bits"
	"time"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// Animation is an animation that can be applied to an element.
type Animation struct {
	// Duration is the amount of time for which this animation should run.
	Duration time.Duration
	// Oneshot is whether the animation runs once instead of repeating when it finishes.
	Oneshot bool
	// Synced is whether to derive the phase from a shared clock. See RepeatSynced.
	Synced bool
	// Easing maps normalized time to an animated value. The result may exceed 0..1 for easing
	// functions that overshoot.
	Easing func(delta float32) float32
	// MaxFps is the maximum number of times per second this animation re-renders. When None,
	// the animation re-renders on every frame.
	MaxFps Option[float32]
	// Steps is the number of equal steps this animation advances in, never zero. See WithSteps.
	// When None, the animation advances continuously.
	Steps Option[int]
}

// NewAnimation creates a new animation with the given duration. By default the animation will
// only run once and will use a linear easing function.
func NewAnimation(duration time.Duration) Animation {
	return Animation{
		Duration: duration,
		Oneshot:  true,
		Synced:   false,
		Easing:   Linear,
		MaxFps:   None[float32](),
		Steps:    None[int](),
	}
}

// Repeat sets the animation to loop when it finishes.
func (a Animation) Repeat() Animation {
	a.Oneshot = false
	return a
}

// RepeatSynced sets the animation to loop when it finishes, phase-locked to a clock shared by
// the whole App.
func (a Animation) RepeatSynced() Animation {
	a.Oneshot = false
	a.Synced = true
	return a
}

// WithEasing sets the easing function used to map normalized time to an animated value.
//
// The output is not clamped, allowing physical easing functions such as springs to overshoot.
func (a Animation) WithEasing(easing func(delta float32) float32) Animation {
	a.Easing = easing
	return a
}

// WithMaxFps limits how often this animation re-renders. Instead of re-rendering on every frame,
// the animation schedules its next render 1 / maxFps seconds after the current one. Values that
// are not finite and positive are ignored.
func (a Animation) WithMaxFps(maxFps float32) Animation {
	a.MaxFps = Some(maxFps)
	return a
}

// WithSteps advances in steps equal steps instead of continuously, like CSS steps(). The
// animator receives 0, then 1 / steps, and so on. The animation re-renders only when it steps,
// so WithMaxFps has no effect. A steps of zero is ignored.
func (a Animation) WithSteps(steps int) Animation {
	if steps == 0 {
		a.Steps = None[int]()
	} else {
		a.Steps = Some(steps)
	}
	return a
}

// WithAnimation renders element with an animation; it is upstream's AnimationExt::with_animation.
//
// Animations rendered this way respect App.ReduceMotion: when it is set, the element is rendered
// in a static state (the end state for oneshot animations, the start state for repeating ones)
// and no animation frames are scheduled.
func WithAnimation[E IntoElement, I ElementIDSource](element E, id I, animation Animation, animator func(element E, delta float32) E) *AnimationElement[E] {
	return newAnimationElement(NewElementID(id), element, []Animation{animation}, func(this E, _ int, value float32) E {
		return animator(this, value)
	})
}

// WithAnimations renders element with a chain of animations; it is upstream's
// AnimationExt::with_animations.
func WithAnimations[E IntoElement, I ElementIDSource](element E, id I, animations []Animation, animator func(element E, animationIx int, delta float32) E) *AnimationElement[E] {
	return newAnimationElement(NewElementID(id), element, animations, animator)
}

// WithSpring renders element at the value produced by a spring; it is upstream's
// AnimationExt::with_spring.
//
// The element ID preserves position and velocity across target changes. A newly mounted spring
// starts at its target unless configured with SpringAnimation.From.
func WithSpring[E IntoElement, I ElementIDSource, T, O any](element E, id I, animation SpringAnimation[T], animator func(element E, value O) E) *SpringAnimationElement[E] {
	target := animation.target
	scalarTarget := springTargetOf(target)
	s := &SpringAnimationElement[E]{
		id:       NewElementID(id),
		element:  Some(element),
		config:   animation.config,
		target:   scalarTarget,
		epsilon:  animation.epsilon,
		initial:  animation.initial,
		playback: animation.playback,
		animator: func(this E, value float32) E {
			return animator(this, springResolve[T, O](target, value))
		},
	}
	BindParent(&s.Parent, s)
	BindFluent(&s.Fluent, s)
	return s
}

func springResolve[T, O any](target T, value float32) O {
	if t, ok := any(target).(SpringTarget[O]); ok {
		return t.Resolve(value)
	}
	return springResolveOf(target, value).(O)
}

func newAnimationElement[E IntoElement](id ElementID, element E, animations []Animation, animator func(E, int, float32) E) *AnimationElement[E] {
	a := &AnimationElement[E]{
		id:         id,
		element:    Some(element),
		animations: animations,
		animator:   animator,
	}
	BindParent(&a.Parent, a)
	BindFluent(&a.Fluent, a)
	return a
}

// AnimationElement is a GPUI element that applies an animation to another element.
type AnimationElement[E IntoElement] struct {
	Parent[*AnimationElement[E]]
	Fluent[*AnimationElement[E]]
	id         ElementID
	element    Option[E]
	animations []Animation
	animator   func(element E, animationIx int, delta float32) E
}

// SpringAnimationElement is a GPUI element driven by a stateful spring.
type SpringAnimationElement[E IntoElement] struct {
	Parent[*SpringAnimationElement[E]]
	Fluent[*SpringAnimationElement[E]]
	id       ElementID
	element  Option[E]
	config   SpringConfig
	target   float32
	epsilon  float32
	initial  Option[float32]
	playback SpringPlayback
	animator func(element E, value float32) E
}

func extendAnimated[E any](element *Option[E], elements []AnyElement) {
	inner := element.AsMut()
	if inner == nil {
		return
	}
	parent, ok := any(*inner).(ParentElement)
	if !ok {
		panic(fmt.Sprintf("gpui: %T does not implement ParentElement", *inner))
	}
	parent.Extend(elements)
}

// Extend adds child elements to the animated element, which must implement ParentElement.
func (s *SpringAnimationElement[E]) Extend(elements []AnyElement) {
	extendAnimated(&s.element, elements)
}

// MapElement applies f to the element being animated.
func (s *SpringAnimationElement[E]) MapElement(f func(element E) E) *SpringAnimationElement[E] {
	if element, ok := s.element.Get(); ok {
		s.element = Some(f(element))
	}
	return s
}

// Extend adds child elements to the animated element, which must implement ParentElement.
func (a *AnimationElement[E]) Extend(elements []AnyElement) {
	extendAnimated(&a.element, elements)
}

// MapElement applies f to the element being animated.
func (a *AnimationElement[E]) MapElement(f func(element E) E) *AnimationElement[E] {
	if element, ok := a.element.Get(); ok {
		a.element = Some(f(element))
	}
	return a
}

type animationStepWake struct {
	due  scheduler.Instant
	task *Task[struct{}]
}

type animationState struct {
	start               scheduler.Instant
	animationIx         int
	delayedFramePending *bool
	stepWake            Option[animationStepWake]
}

func (s *animationState) renderAt(instant scheduler.Instant, window *Window, cx *App) {
	now := cx.BackgroundExecutor().Now()
	if wake, ok := s.stepWake.Get(); ok && now.Before(wake.due) && !instant.Before(wake.due) {
		return
	}
	view := window.CurrentView()
	wake := windowSpawn(window, cx, func(cx *AsyncWindowContext) {
		for {
			delay := instant.SaturatingDurationSince(cx.BackgroundExecutor().Now())
			if delay == 0 {
				break
			}
			_, _ = cx.BackgroundExecutor().Timer(delay).Await(cx)
		}
		_ = cx.Update(func(_ *Window, cx *App) { cx.Notify(view) })
	})
	if previous, ok := s.stepWake.Replace(animationStepWake{due: instant, task: wake}).Get(); ok {
		previous.task.Cancel()
	}
}

type springElementState struct {
	spring    SpringState
	target    float32
	config    SpringConfig
	initial   float32
	playback  SpringPlayback
	updatedAt scheduler.Instant
}

type animatedLayout struct {
	layoutID LayoutID
	element  AnyElement
}

// ElementID returns the spring element's id.
func (s *SpringAnimationElement[E]) ElementID() (ElementID, bool) {
	return s.id, true
}

// SourceLocation returns nil.
func (s *SpringAnimationElement[E]) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout advances the spring and requests the layout of the animated element.
func (s *SpringAnimationElement[E]) RequestLayout(globalID *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, AnyElement) {
	if globalID == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	result := window.WithElementState(globalID, func(stateOpt Option[springElementState], window *Window) (animatedLayout, springElementState) {
		now := cx.BackgroundExecutor().Now()
		initial := s.initial.UnwrapOr(s.target)
		state, ok := stateOpt.Get()
		if !ok {
			state = springElementState{
				spring:    SpringState{Position: initial, Velocity: 0.0},
				target:    s.target,
				config:    s.config,
				initial:   initial,
				playback:  s.playback,
				updatedAt: now,
			}
		}

		elapsed := durationAsSecsF32(now.SaturatingDurationSince(state.updatedAt))
		if state.playback == SpringPlaybackRunning {
			state.spring = state.config.Step(state.spring, state.target, elapsed)
		}

		state.config = s.config
		state.target = s.target

		var done bool
		switch s.playback {
		case SpringPlaybackRunning:
			if cx.ReduceMotion() {
				state.spring = SpringState{Position: state.target, Velocity: 0.0}
				done = true
			} else {
				done = state.config.IsSettled(state.spring, state.target, s.epsilon)
				if done {
					state.spring = SpringState{Position: state.target, Velocity: 0.0}
				}
			}
		case SpringPlaybackPaused:
			done = true
		case SpringPlaybackStopped:
			state.spring.Velocity = 0.0
			done = true
		case SpringPlaybackCompleted:
			state.spring = SpringState{Position: state.target, Velocity: 0.0}
			done = true
		case SpringPlaybackCancelled:
			state.spring = SpringState{Position: state.initial, Velocity: 0.0}
			done = true
		}
		state.playback = s.playback
		state.updatedAt = now

		element := s.element.Take().Expect("should only be called once")
		animator := s.animator
		if animator == nil {
			panic("should only be called once")
		}
		s.animator = nil
		anyElement := animator(element, state.spring.Position).IntoAnyElement()

		if !done {
			window.RequestAnimationFrame()
		}

		return animatedLayout{layoutID: anyElement.RequestLayout(window, cx), element: anyElement}, state
	})
	return result.layoutID, result.element
}

// Prepaint prepaints the animated element.
func (s *SpringAnimationElement[E]) Prepaint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], element *AnyElement, window *Window, cx *App) struct{} {
	element.Prepaint(window, cx)
	return struct{}{}
}

// Paint paints the animated element.
func (s *SpringAnimationElement[E]) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], element *AnyElement, _ *struct{}, window *Window, cx *App) {
	element.Paint(window, cx)
}

// IntoAnyElement converts the spring element into an AnyElement.
func (s *SpringAnimationElement[E]) IntoAnyElement() AnyElement {
	return NewAnyElement[AnyElement, struct{}](s)
}

// ElementID returns the animation element's id.
func (a *AnimationElement[E]) ElementID() (ElementID, bool) {
	return a.id, true
}

// SourceLocation returns nil.
func (a *AnimationElement[E]) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout computes the animated value and requests the layout of the animated element.
func (a *AnimationElement[E]) RequestLayout(globalID *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, AnyElement) {
	if globalID == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	result := window.WithElementState(globalID, func(stateOpt Option[*animationState], window *Window) (animatedLayout, *animationState) {
		now := cx.BackgroundExecutor().Now()
		state, ok := stateOpt.Get()
		if !ok {
			pending := false
			state = &animationState{
				start:               now,
				animationIx:         0,
				delayedFramePending: &pending,
				stepWake:            None[animationStepWake](),
			}
		}
		var animationIx int
		var delta float32
		var done bool
		nextStep := None[scheduler.Instant]()
		if cx.ReduceMotion() {
			animationIx = len(a.animations) - 1
			if a.animations[animationIx].Oneshot {
				delta = 1.0
			} else {
				delta = 0.0
			}
			done = true
		} else {
			animationIx = state.animationIx
			animation := a.animations[animationIx]
			duration := animation.Duration

			var elapsed time.Duration
			if animation.Synced && duration != 0 {
				elapsed = now.Sub(cx.syncedAnimationEpoch)
				elapsed = time.Duration(uint64(elapsed) % uint64(duration))
			} else {
				elapsed = now.Sub(state.start)
			}
			delta = durationAsSecsF32(elapsed) / durationAsSecsF32(duration)
			finished := delta > 1.0
			if steps, ok := animation.Steps.Get(); ok && duration != 0 {
				elapsedNanos, durationNanos := uint64(elapsed), uint64(duration)
				stepCount := uint64(steps)
				step := mulDiv64(elapsedNanos, stepCount, durationNanos)
				delta = float32(step%stepCount) / float32(stepCount)
				finished = animation.Oneshot && step >= stepCount
				if !finished {
					startsIn := mulDivCeil64(step+1, durationNanos, stepCount) - elapsedNanos
					nextStep = Some(now.Add(time.Duration(startsIn)))
				}
			}

			if finished {
				if animation.Oneshot {
					if animationIx >= len(a.animations)-1 {
						done = true
					} else {
						state.start = now
						state.animationIx++
					}
					delta = 1.0
				} else {
					delta = float32(math.Mod(float64(delta), 1.0))
				}
			}
		}
		delta = a.animations[animationIx].Easing(delta)

		if gpuiutil.DebugAssertions && !isFiniteF32(delta) {
			panic("animated value should be finite")
		}

		element := a.element.Take().Expect("should only be called once")
		anyElement := a.animator(element, animationIx, delta).IntoAnyElement()

		if next, ok := nextStep.Get(); ok {
			state.renderAt(next, window, cx)
		} else if !done {
			animation := a.animations[animationIx]
			maxFps, capped := animation.MaxFps.Get()
			if capped && animation.Steps.IsNone() && isFiniteF32(maxFps) && maxFps > 0.0 {
				if !*state.delayedFramePending {
					*state.delayedFramePending = true
					delayedFramePending := state.delayedFramePending
					view := window.CurrentView()
					interval := durationFromSecsF32(1.0 / maxFps)
					windowSpawn(window, cx, func(cx *AsyncWindowContext) {
						_, _ = cx.BackgroundExecutor().Timer(interval).Await(cx)
						*delayedFramePending = false
						_ = cx.Update(func(_ *Window, cx *App) { cx.Notify(view) })
					}).Detach()
				}
			} else {
				window.RequestAnimationFrame()
			}
		}

		return animatedLayout{layoutID: anyElement.RequestLayout(window, cx), element: anyElement}, state
	})
	return result.layoutID, result.element
}

// Prepaint prepaints the animated element.
func (a *AnimationElement[E]) Prepaint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], element *AnyElement, window *Window, cx *App) struct{} {
	element.Prepaint(window, cx)
	return struct{}{}
}

// Paint paints the animated element.
func (a *AnimationElement[E]) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], element *AnyElement, _ *struct{}, window *Window, cx *App) {
	element.Paint(window, cx)
}

// IntoAnyElement converts the animation element into an AnyElement.
func (a *AnimationElement[E]) IntoAnyElement() AnyElement {
	return NewAnyElement[AnyElement, struct{}](a)
}

func durationAsSecsF32(d time.Duration) float32 {
	return float32(d/time.Second) + float32(d%time.Second)/1e9
}

func mulDiv64(a, b, d uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	q, _ := bits.Div64(hi, lo, d)
	return q
}

func mulDivCeil64(a, b, d uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	q, r := bits.Div64(hi, lo, d)
	if r != 0 {
		q++
	}
	return q
}

// Linear is the linear easing function, or delta itself.
func Linear(delta float32) float32 {
	return delta
}

// Quadratic is the quadratic easing function, delta * delta.
func Quadratic(delta float32) float32 {
	return delta * delta
}

// EaseInOut is the quadratic ease-in-out function, which starts and ends slowly but speeds up in
// the middle.
func EaseInOut(delta float32) float32 {
	if delta < 0.5 {
		return 2.0 * delta * delta
	}
	x := -2.0*delta + 2.0
	return 1.0 - x*x/2.0
}

// EaseOutQuint returns the Quint ease-out function, which starts quickly and decelerates to a
// stop.
func EaseOutQuint() func(delta float32) float32 {
	return func(delta float32) float32 {
		x := 1.0 - delta
		x2 := x * x
		x4 := x2 * x2
		return 1.0 - x*x4
	}
}

// Heartbeat is a heartbeat: a strong beat, a weaker one, then a rest. Starts and ends at 0 and
// peaks at 1.
func Heartbeat(delta float32) float32 {
	beat := func(center, width, height float32) float32 {
		x := (delta - center) / width
		return height * expF32(-x*x)
	}
	return beat(0.2, 0.06, 1.0) + beat(0.4, 0.05, 0.58)
}

// Bounce applies easing, first in the forward direction and then in the reverse direction.
func Bounce(easing func(delta float32) float32) func(delta float32) float32 {
	return func(delta float32) float32 {
		if delta < 0.5 {
			return easing(delta * 2.0)
		}
		return easing((1.0 - delta) * 2.0)
	}
}

// PulsatingBetween returns a custom easing function for pulsating alpha that slows down as it
// approaches 0.1.
func PulsatingBetween(min, max float32) func(delta float32) float32 {
	r := max - min

	return func(delta float32) float32 {
		t := float32(math.Sin(float64(delta * 2.0 * float32(math.Pi))))
		breath := (t*t*t + t) / 2.0

		normalizedAlpha := (breath + 1.0) / 2.0

		return min + (normalizedAlpha * r)
	}
}
