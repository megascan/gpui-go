package gpui

import (
	"math"
	"sync"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

const scrollEventSeparation = 28 * time.Millisecond

func dominantAxis(delta Point[Pixels]) Axis {
	if delta.X.Abs() <= delta.Y.Abs() {
		return AxisVertical
	}
	return AxisHorizontal
}

func lockDeltaToAxis(delta *Point[Pixels], axis Axis) {
	switch axis {
	case AxisVertical:
		delta.X = PixelsZero
	case AxisHorizontal:
		delta.Y = PixelsZero
	}
}

func movementsOppose(left, right Point[Pixels]) bool {
	return float32(left.X)*float32(right.X)+float32(left.Y)*float32(right.Y) < 0.
}

// OngoingScroll tracks the dominant axis across the events in a scroll gesture.
type OngoingScroll struct {
	lastEvent Option[scheduler.Instant]
	axis      Option[Axis]
}

// Filter filters delta to the dominant axis of the current scroll gesture. Gestures are
// delimited by their touch phase when available, with a timeout fallback for platforms that
// only emit TouchPhaseMoved.
func (s *OngoingScroll) Filter(delta *Point[Pixels], touchPhase TouchPhase) {
	s.filterAt(delta, touchPhase, scheduler.InstantNow())
}

func (s *OngoingScroll) filterAt(delta *Point[Pixels], touchPhase TouchPhase, now scheduler.Instant) {
	const unlockPercent float32 = 1.9
	const unlockLowerBound Pixels = 6.

	if touchPhase == TouchPhaseEnded || touchPhase == TouchPhaseCancelled {
		s.lastEvent = None[scheduler.Instant]()
		s.axis = None[Axis]()
		return
	}

	x := delta.X.Abs()
	y := delta.Y.Abs()
	if x.IsZero() && y.IsZero() {
		if touchPhase == TouchPhaseStarted {
			s.lastEvent = None[scheduler.Instant]()
			s.axis = None[Axis]()
		}
		return
	}

	startsNewGesture := touchPhase == TouchPhaseStarted
	if !startsNewGesture {
		lastEvent, ok := s.lastEvent.Get()
		startsNewGesture = !ok || now.Sub(lastEvent) >= scrollEventSeparation
	}
	axis := s.axis
	if startsNewGesture {
		axis = Some(dominantAxis(*delta))
	} else if max(x, y) >= unlockLowerBound {
		if current, ok := axis.Get(); ok {
			switch {
			case current == AxisVertical && x > y && x >= y*Pixels(unlockPercent):
				axis = None[Axis]()
			case current == AxisHorizontal && y > x && y >= x*Pixels(unlockPercent):
				axis = None[Axis]()
			}
		}
	}

	s.lastEvent = Some(now)
	s.axis = axis
	if axis, ok := axis.Get(); ok {
		lockDeltaToAxis(delta, axis)
	}
}

// GestureTuning holds feel constants consumed by gesture recognizers, provided on a
// best-effort basis depending on each platform's support. DefaultGestureTuning returns GPUI's
// own (iOS flavored) values.
type GestureTuning struct {
	// TouchSlop is the distance a touch may travel before it stops being a potential tap and
	// becomes a pan or drag.
	TouchSlop Pixels
	// MultiTapInterval is the maximum interval between taps for them to accumulate a tap count.
	MultiTapInterval time.Duration
	// MultiTapSlop is the maximum distance between taps for them to accumulate a tap count.
	MultiTapSlop Pixels
	// LongPressDuration is how long a touch must remain within TouchSlop to be recognized as a
	// long press.
	LongPressDuration time.Duration
	// ScrollPhysics is how scroll momentum decelerates after a fling.
	ScrollPhysics ScrollPhysics
	// MinFlingVelocity is the minimum release velocity, in pixels per second, required to
	// start scroll momentum.
	MinFlingVelocity float32
}

// DefaultGestureTuning returns GPUI's default gesture tuning.
func DefaultGestureTuning() GestureTuning {
	return GestureTuning{
		TouchSlop:         Px(8.),
		MultiTapInterval:  400 * time.Millisecond,
		MultiTapSlop:      Px(16.),
		LongPressDuration: 500 * time.Millisecond,
		ScrollPhysics:     ScrollPhysicsIos(),
		MinFlingVelocity:  50.,
	}
}

// ScrollPhysics is how free scrolling decelerates after a fling: ScrollPhysicsExponential or
// ScrollPhysicsFrictionSpline. Boundary behavior is the scroll container's policy.
type ScrollPhysics interface {
	flingDuration(speed float32) time.Duration
	flingDistance(speed float32, elapsed time.Duration) float32
}

// ScrollPhysicsExponential is exponential velocity decay, the UIScrollView model:
// velocity(t) = v0 * DecayPerMs^ms.
type ScrollPhysicsExponential struct {
	// DecayPerMs is the per-millisecond velocity decay factor; UIScrollView's normal
	// deceleration rate is 0.998.
	DecayPerMs float32
}

// ScrollPhysicsFrictionSpline is the friction spline of Android's OverScroller, transcribed
// from AOSP's SplineOverScroller (Apache-2.0).
type ScrollPhysicsFrictionSpline struct {
	// Friction is the scroll friction coefficient; 0.015 on Android.
	Friction float32
	// PixelsPerInch is the pixels per physical inch of the display, in the coordinate space
	// the fling runs in.
	PixelsPerInch float32
}

// ScrollPhysicsIos returns the iOS scroll feel: UIScrollView's normal deceleration rate.
func ScrollPhysicsIos() ScrollPhysics {
	return ScrollPhysicsExponential{DecayPerMs: 0.998}
}

// ScrollPhysicsAndroid returns the Android scroll feel: OverScroller with stock friction, at
// Android's nominal density of 160 density-independent pixels per inch.
func ScrollPhysicsAndroid() ScrollPhysics {
	return ScrollPhysicsFrictionSpline{Friction: 0.015, PixelsPerInch: 160.}
}

func lnF32(v float32) float32 { return float32(math.Log(float64(v))) }

func (p ScrollPhysicsExponential) flingDuration(speed float32) time.Duration {
	if speed <= momentumStopVelocity {
		return 0
	}
	milliseconds := lnF32(momentumStopVelocity/speed) / lnF32(p.DecayPerMs)
	return durationFromSecsF32(milliseconds / 1000.)
}

func (p ScrollPhysicsFrictionSpline) flingDuration(speed float32) time.Duration {
	if speed <= 0. {
		return 0
	}
	deceleration := frictionSplineDeceleration(speed, p.Friction, p.PixelsPerInch)
	seconds := math.Exp(deceleration / (frictionSplineDecelerationRate() - 1.))
	return time.Duration(seconds * 1e9)
}

func durationSecsF32(d time.Duration) float32 {
	return float32(d.Seconds())
}

func (p ScrollPhysicsExponential) flingDistance(speed float32, elapsed time.Duration) float32 {
	duration := p.flingDuration(speed)
	if duration == 0 {
		return 0.
	}
	elapsed = min(elapsed, duration)
	milliseconds := durationSecsF32(elapsed) * 1000.
	return (speed / 1000.) * (float32(math.Pow(float64(p.DecayPerMs), float64(milliseconds))) - 1.) / lnF32(p.DecayPerMs)
}

func (p ScrollPhysicsFrictionSpline) flingDistance(speed float32, elapsed time.Duration) float32 {
	duration := p.flingDuration(speed)
	if duration == 0 {
		return 0.
	}
	elapsed = min(elapsed, duration)
	deceleration := frictionSplineDeceleration(speed, p.Friction, p.PixelsPerInch)
	rate := frictionSplineDecelerationRate()
	totalDistance := float64(p.Friction) * frictionSplinePhysicalCoefficient(p.PixelsPerInch) * math.Exp(rate/(rate-1.)*deceleration)
	progress := elapsed.Seconds() / duration.Seconds()
	return float32(totalDistance) * frictionSplineDistanceCoefficient(float32(progress))
}

const (
	frictionSplineNbSamples    = 100
	frictionSplineInflexion    = float32(0.35)
	frictionSplineStartTension = float32(0.5)
	frictionSplineEndTension   = float32(1.0)
	frictionSplineP1           = frictionSplineStartTension * frictionSplineInflexion
	frictionSplineP2           = 1.0 - frictionSplineEndTension*(1.0-frictionSplineInflexion)
)

func frictionSplineDecelerationRate() float64 {
	return math.Log(0.78) / math.Log(0.9)
}

var frictionSplinePosition = sync.OnceValue(func() *[frictionSplineNbSamples + 1]float32 {
	var splinePosition [frictionSplineNbSamples + 1]float32
	var xMin float32
	for i := range frictionSplineNbSamples {
		alpha := float32(i) / float32(frictionSplineNbSamples)
		var xMax float32 = 1
		var x, coefficient float32
		for {
			x = xMin + (xMax-xMin)/2.
			coefficient = 3. * x * (1. - x)
			t := coefficient*((1.-x)*frictionSplineP1+x*frictionSplineP2) + x*x*x
			if absF32(t-alpha) < 1e-5 {
				break
			}
			if t > alpha {
				xMax = x
			} else {
				xMin = x
			}
		}
		splinePosition[i] = coefficient*((1.-x)*frictionSplineStartTension+x) + x*x*x
	}
	splinePosition[frictionSplineNbSamples] = 1.
	return &splinePosition
})

func frictionSplinePhysicalCoefficient(pixelsPerInch float32) float64 {
	return 9.80665 * 39.37 * float64(pixelsPerInch) * 0.84
}

func frictionSplineDeceleration(speed, friction, pixelsPerInch float32) float64 {
	return math.Log(float64(frictionSplineInflexion) * float64(speed) / (float64(friction) * frictionSplinePhysicalCoefficient(pixelsPerInch)))
}

func frictionSplineDistanceCoefficient(t float32) float32 {
	if t >= 1. {
		return 1.
	}
	index := min(int(float32(frictionSplineNbSamples)*t), frictionSplineNbSamples-1)
	timeLower := float32(index) / float32(frictionSplineNbSamples)
	timeUpper := float32(index+1) / float32(frictionSplineNbSamples)
	samples := frictionSplinePosition()
	distanceLower := samples[index]
	distanceUpper := samples[index+1]
	velocityCoefficient := (distanceUpper - distanceLower) / (timeUpper - timeLower)
	return distanceLower + (t-timeLower)*velocityCoefficient
}

// GestureKinds is the set of gesture kinds that participate in recognition, used by
// PlatformGestures.NativeRecognizers to declare which gestures the platform recognizes natively.
type GestureKinds struct {
	// Tap is tap and multi-tap, synthesized as mouse presses by the portable recognizer.
	Tap bool
	// LongPress is long press, surfaced as LongPressEvent.
	LongPress bool
	// Pan is pan and scroll, including fling momentum, surfaced as ScrollEvents.
	Pan bool
	// Pinch is pinch to zoom, surfaced as PinchEvents.
	Pinch bool
}

// GestureKindsNone is no gestures; GPUI core's portable recognizers handle everything.
var GestureKindsNone = GestureKinds{}

// GestureKindsAll is all gesture kinds.
var GestureKindsAll = GestureKinds{Tap: true, LongPress: true, Pan: true, Pinch: true}

// TouchDragEvent is a direct touch drag claimed by an element before touch input becomes a
// tap, long press or scrolling gesture.
type TouchDragEvent struct {
	// Phase is the phase of the touch drag.
	Phase TouchPhase
	// StartPosition is the position where the touch started.
	StartPosition Point[Pixels]
	// Position is the touch's current position.
	Position Point[Pixels]
}

// ToPlatformInput returns e as a PlatformInput.
func (e TouchDragEvent) ToPlatformInput() PlatformInput { return e }

func (TouchDragEvent) sealedInputEvent() {}

func (TouchDragEvent) isGestureEvent() {}

func (TouchDragEvent) isMouseEvent() {}

// BackGestureEvent is a back-navigation gesture, such as the iOS swipe from the leading screen
// edge. The innermost element under StartPosition claims the whole gesture when it starts;
// every phase is delivered at the start position.
type BackGestureEvent struct {
	// Phase is the phase of the gesture.
	Phase BackGesturePhase
	// StartPosition is where the gesture started.
	StartPosition Point[Pixels]
	// Progress is how far the gesture has travelled toward going back, from 0 to 1.
	Progress float32
}

// BackGesturePhase is the phase of a BackGestureEvent.
type BackGesturePhase uint8

const (
	// BackGesturePhaseStarted means the gesture began; claiming it here routes the remaining
	// phases to the same element.
	BackGesturePhaseStarted BackGesturePhase = iota
	// BackGesturePhaseProgressed means the gesture moved and Progress changed.
	BackGesturePhaseProgressed
	// BackGesturePhaseCommitted means the gesture ended and the platform decided to go back.
	BackGesturePhaseCommitted
	// BackGesturePhaseCancelled means the gesture ended without going back.
	BackGesturePhaseCancelled
)

// ToPlatformInput returns e as a PlatformInput.
func (e BackGestureEvent) ToPlatformInput() PlatformInput { return e }

func (BackGestureEvent) sealedInputEvent() {}

func (BackGestureEvent) isGestureEvent() {}

func (BackGestureEvent) isMouseEvent() {}

// LongPressEvent is a phased long-press gesture recognized from a touch. Its zero value has
// TouchPhaseMoved; DefaultLongPressEvent returns upstream's default (TouchPhaseStarted).
type LongPressEvent struct {
	// Phase is the phase of the long press.
	Phase TouchPhase
	// StartPosition is the position where the touch started.
	StartPosition Point[Pixels]
	// Position is the touch's current position.
	Position Point[Pixels]
}

// DefaultLongPressEvent returns a started long press at the origin.
func DefaultLongPressEvent() LongPressEvent {
	return LongPressEvent{Phase: TouchPhaseStarted}
}

// ToPlatformInput returns e as a PlatformInput.
func (e LongPressEvent) ToPlatformInput() PlatformInput { return e }

func (LongPressEvent) sealedInputEvent() {}

func (LongPressEvent) isGestureEvent() {}

func (LongPressEvent) isMouseEvent() {}

// PlatformGestures are platform gesture recognition services. Implementations embed
// NullPlatformGestures for upstream's default methods.
type PlatformGestures interface {
	// Tuning returns feel constants for the portable recognizers on this platform.
	Tuning() GestureTuning
	// NativeRecognizers returns the gesture kinds this platform recognizes natively.
	NativeRecognizers() GestureKinds
}

// NullPlatformGestures is a no-op PlatformGestures: no native recognizers and default tuning.
type NullPlatformGestures struct{}

// Tuning returns DefaultGestureTuning.
func (NullPlatformGestures) Tuning() GestureTuning { return DefaultGestureTuning() }

// NativeRecognizers returns GestureKindsNone.
func (NullPlatformGestures) NativeRecognizers() GestureKinds { return GestureKindsNone }

const maxFlingVelocity float32 = 8000.

const momentumStopVelocity float32 = 10.

const velocityWindow = 100 * time.Millisecond

const velocityAssumeStoppedGap = 40 * time.Millisecond

const velocityMaxSamples = 20

type touchGestureRecognizer struct {
	tuning   GestureTuning
	state    touchGestureState
	momentum *momentum
	lastTap  *completedTap
}

type recognizedTouchGesture interface {
	isRecognizedTouchGesture()
}

type recognizedTouchGestureScroll struct{ ScrollEvent }

type recognizedTouchGestureTap struct {
	down MouseDownEvent
	up   MouseUpEvent
}

type recognizedTouchGestureTouchDrag struct{ TouchDragEvent }

type recognizedTouchGestureLongPress struct{ LongPressEvent }

func (recognizedTouchGestureScroll) isRecognizedTouchGesture()    {}
func (recognizedTouchGestureTap) isRecognizedTouchGesture()       {}
func (recognizedTouchGestureTouchDrag) isRecognizedTouchGesture() {}
func (recognizedTouchGestureLongPress) isRecognizedTouchGesture() {}

type touchGestureStateKind uint8

const (
	touchGestureStateIdle touchGestureStateKind = iota
	touchGestureStatePending
	touchGestureStatePanning
	touchGestureStateCaught
	touchGestureStateLongPressing
	touchGestureStateTouchDragging
)

type touchGestureState struct {
	kind             touchGestureStateKind
	touch            activeTouch
	deadline         scheduler.Instant
	longPressOffered bool
	touchDragOffered bool
	axis             Axis
}

type activeTouch struct {
	id              TouchID
	startPosition   Point[Pixels]
	lastPosition    Point[Pixels]
	emittedPosition Point[Pixels]
	lastMovement    Point[Pixels]
	velocityTracker velocityTracker
}

type completedTap struct {
	position Point[Pixels]
	time     scheduler.Instant
	count    int
}

type momentum struct {
	position        Point[Pixels]
	direction       Point[float32]
	speed           float32
	startedAt       scheduler.Instant
	duration        time.Duration
	emittedDistance float32
}

func newTouchGestureRecognizer(tuning GestureTuning) *touchGestureRecognizer {
	return &touchGestureRecognizer{tuning: tuning}
}

func (r *touchGestureRecognizer) handleEvent(event *TouchEvent) []recognizedTouchGesture {
	return r.handleEventAt(event, scheduler.InstantNow())
}

func (r *touchGestureRecognizer) takeState() touchGestureState {
	state := r.state
	r.state = touchGestureState{}
	return state
}

func (r *touchGestureRecognizer) handleEventAt(event *TouchEvent, now scheduler.Instant) []recognizedTouchGesture {
	var recognized []recognizedTouchGesture
	switch event.Phase {
	case TouchPhaseStarted:
		caughtFling := false
		if m := r.momentum; m != nil {
			r.momentum = nil
			recognized = append(recognized, recognizedTouchGestureScroll{touchScrollEvent(m.position, Point[Pixels]{}, TouchPhaseEnded)})
			caughtFling = true
		}
		if r.state.kind == touchGestureStateIdle {
			var tracker velocityTracker
			tracker.push(now, event.Position)
			touch := activeTouch{
				id:              event.ID,
				startPosition:   event.Position,
				lastPosition:    event.Position,
				emittedPosition: event.Position,
				velocityTracker: tracker,
			}
			if caughtFling {
				recognized = append(recognized, recognizedTouchGestureScroll{touchScrollEvent(touch.startPosition, Point[Pixels]{}, TouchPhaseStarted)})
				r.state = touchGestureState{kind: touchGestureStateCaught, touch: touch}
			} else {
				r.state = touchGestureState{
					kind:     touchGestureStatePending,
					touch:    touch,
					deadline: now.Add(r.tuning.LongPressDuration),
				}
			}
		}
	case TouchPhaseMoved:
		state := r.takeState()
		touch := state.touch
		matches := state.kind != touchGestureStateIdle && touch.id == event.ID
		switch {
		case matches && state.kind == touchGestureStatePending:
			touch.velocityTracker.push(now, event.Position)
			touch.lastPosition = event.Position
			accumulated := event.Position.Sub(touch.startPosition)
			if accumulated.Magnitude() > float64(r.tuning.TouchSlop) {
				target := event.PredictedPosition.UnwrapOr(event.Position)
				axis := dominantAxis(accumulated)
				delta := target.Sub(touch.startPosition)
				lockDeltaToAxis(&delta, axis)
				touch.lastMovement = accumulated
				lockDeltaToAxis(&touch.lastMovement, axis)
				if movementsOppose(delta, touch.lastMovement) {
					target = event.Position
					delta = accumulated
					lockDeltaToAxis(&delta, axis)
				}
				touch.emittedPosition = target
				recognized = append(recognized, recognizedTouchGestureScroll{touchScrollEvent(touch.startPosition, delta, TouchPhaseStarted)})
				r.state = touchGestureState{kind: touchGestureStatePanning, touch: touch, axis: axis}
			} else {
				state.touch = touch
				r.state = state
			}
		case matches && state.kind == touchGestureStateCaught:
			accumulated := event.Position.Sub(touch.startPosition)
			if accumulated == (Point[Pixels]{}) {
				r.state = state
			} else {
				axis := dominantAxis(accumulated)
				r.state = touchGestureState{kind: touchGestureStatePanning, touch: touch, axis: axis}
				return r.handleEventAt(event, now)
			}
		case matches && state.kind == touchGestureStatePanning:
			axis := state.axis
			rawDelta := event.Position.Sub(touch.lastPosition)
			lockDeltaToAxis(&rawDelta, axis)
			if rawDelta != (Point[Pixels]{}) {
				touch.lastMovement = rawDelta
			}
			touch.velocityTracker.push(now, event.Position)
			touch.lastPosition = event.Position
			target := event.PredictedPosition.UnwrapOr(event.Position)
			delta := target.Sub(touch.emittedPosition)
			lockDeltaToAxis(&delta, axis)
			if movementsOppose(delta, touch.lastMovement) {
				target = event.Position
				delta = target.Sub(touch.emittedPosition)
				lockDeltaToAxis(&delta, axis)
				if movementsOppose(delta, touch.lastMovement) {
					target = touch.emittedPosition
					delta = Point[Pixels]{}
				}
			}
			touch.emittedPosition = target
			recognized = append(recognized, recognizedTouchGestureScroll{touchScrollEvent(touch.startPosition, delta, TouchPhaseMoved)})
			r.state = touchGestureState{kind: touchGestureStatePanning, touch: touch, axis: axis}
		case matches && state.kind == touchGestureStateLongPressing:
			touch.lastPosition = event.Position
			recognized = append(recognized, recognizedTouchGestureLongPress{LongPressEvent{
				Phase:         TouchPhaseMoved,
				StartPosition: touch.startPosition,
				Position:      event.Position,
			}})
			r.state = touchGestureState{kind: touchGestureStateLongPressing, touch: touch}
		case matches && state.kind == touchGestureStateTouchDragging:
			touch.lastPosition = event.Position
			recognized = append(recognized, recognizedTouchGestureTouchDrag{TouchDragEvent{
				Phase:         TouchPhaseMoved,
				StartPosition: touch.startPosition,
				Position:      event.Position,
			}})
			r.state = touchGestureState{kind: touchGestureStateTouchDragging, touch: touch}
		default:
			r.state = state
		}
	case TouchPhaseEnded:
		state := r.takeState()
		touch := state.touch
		matches := state.kind != touchGestureStateIdle && touch.id == event.ID
		switch {
		case matches && state.kind == touchGestureStatePending:
			tapCount := 1
			if tap := r.lastTap; tap != nil &&
				now.Sub(tap.time) <= r.tuning.MultiTapInterval &&
				event.Position.Sub(tap.position).Magnitude() <= float64(r.tuning.MultiTapSlop) {
				tapCount = tap.count + 1
			}
			r.lastTap = &completedTap{position: event.Position, time: now, count: tapCount}
			recognized = append(recognized, recognizedTouchGestureTap{
				down: MouseDownEvent{
					Button:     MouseButtonLeft,
					Position:   event.Position,
					Modifiers:  Modifiers{},
					ClickCount: tapCount,
					FirstMouse: false,
				},
				up: MouseUpEvent{
					Button:     MouseButtonLeft,
					Position:   event.Position,
					Modifiers:  Modifiers{},
					ClickCount: tapCount,
				},
			})
		case matches && state.kind == touchGestureStateCaught:
			recognized = append(recognized, recognizedTouchGestureScroll{touchScrollEvent(touch.startPosition, Point[Pixels]{}, TouchPhaseEnded)})
		case matches && state.kind == touchGestureStatePanning:
			axis := state.axis
			latest, ok := touch.velocityTracker.latestSampleTime()
			fingerStopped := !ok || now.Sub(latest) > velocityAssumeStoppedGap
			var velocity Point[float32]
			if !fingerStopped {
				velocity = touch.velocityTracker.velocity()
			}
			switch axis {
			case AxisVertical:
				velocity.X = 0.
			case AxisHorizontal:
				velocity.Y = 0.
			}
			speed := sqrtF32(velocity.X*velocity.X + velocity.Y*velocity.Y)
			releaseDelta := event.Position.Sub(touch.emittedPosition)
			lockDeltaToAxis(&releaseDelta, axis)
			if speed >= r.tuning.MinFlingVelocity {
				direction := NewPoint(velocity.X/speed, velocity.Y/speed)
				speed := min(speed, maxFlingVelocity)
				duration := r.tuning.ScrollPhysics.flingDuration(speed)
				if duration != 0 {
					totalDistance := r.tuning.ScrollPhysics.flingDistance(speed, duration)
					overshoot := -(float32(releaseDelta.X)*direction.X + float32(releaseDelta.Y)*direction.Y)
					var emittedDistance float32
					if overshoot > 0. && overshoot < totalDistance {
						releaseDelta = releaseDelta.Add(NewPoint(Px(direction.X*overshoot), Px(direction.Y*overshoot)))
						emittedDistance = overshoot
					}
					r.momentum = &momentum{
						position:        touch.startPosition,
						direction:       direction,
						speed:           speed,
						startedAt:       now,
						duration:        duration,
						emittedDistance: emittedDistance,
					}
				}
			}
			recognized = append(recognized, recognizedTouchGestureScroll{touchScrollEvent(touch.startPosition, releaseDelta, TouchPhaseEnded)})
		case matches && state.kind == touchGestureStateLongPressing:
			recognized = append(recognized, recognizedTouchGestureLongPress{LongPressEvent{
				Phase:         TouchPhaseEnded,
				StartPosition: touch.startPosition,
				Position:      event.Position,
			}})
		case matches && state.kind == touchGestureStateTouchDragging:
			recognized = append(recognized, recognizedTouchGestureTouchDrag{TouchDragEvent{
				Phase:         TouchPhaseEnded,
				StartPosition: touch.startPosition,
				Position:      event.Position,
			}})
		default:
			r.state = state
		}
	case TouchPhaseCancelled:
		state := r.takeState()
		touch := state.touch
		matches := state.kind != touchGestureStateIdle && touch.id == event.ID
		switch {
		case matches && state.kind == touchGestureStatePending:
		case matches && (state.kind == touchGestureStatePanning || state.kind == touchGestureStateCaught):
			recognized = append(recognized, recognizedTouchGestureScroll{touchScrollEvent(touch.startPosition, Point[Pixels]{}, TouchPhaseCancelled)})
		case matches && state.kind == touchGestureStateLongPressing:
			recognized = append(recognized, recognizedTouchGestureLongPress{LongPressEvent{
				Phase:         TouchPhaseCancelled,
				StartPosition: touch.startPosition,
				Position:      event.Position,
			}})
		case matches && state.kind == touchGestureStateTouchDragging:
			recognized = append(recognized, recognizedTouchGestureTouchDrag{TouchDragEvent{
				Phase:         TouchPhaseCancelled,
				StartPosition: touch.startPosition,
				Position:      event.Position,
			}})
		default:
			r.state = state
		}
	}
	return recognized
}

func (r *touchGestureRecognizer) pendingLongPress() (TouchID, time.Duration, bool) {
	if r.state.kind != touchGestureStatePending || r.state.longPressOffered {
		return 0, 0, false
	}
	return r.state.touch.id, r.state.deadline.SaturatingDurationSince(scheduler.InstantNow()), true
}

func (r *touchGestureRecognizer) offerLongPress(id TouchID) (recognizedTouchGesture, bool) {
	if r.state.kind != touchGestureStatePending {
		return nil, false
	}
	if r.state.touch.id != id || r.state.longPressOffered {
		return nil, false
	}
	r.state.longPressOffered = true
	return recognizedTouchGestureLongPress{LongPressEvent{
		Phase:         TouchPhaseStarted,
		StartPosition: r.state.touch.startPosition,
		Position:      r.state.touch.lastPosition,
	}}, true
}

func (r *touchGestureRecognizer) resolveLongPress(claimed bool) {
	if !claimed {
		return
	}
	if r.state.kind == touchGestureStatePending && r.state.longPressOffered {
		r.state = touchGestureState{kind: touchGestureStateLongPressing, touch: r.state.touch}
	}
}

func (r *touchGestureRecognizer) offerTouchDrag(id TouchID) (recognizedTouchGesture, bool) {
	if r.state.kind != touchGestureStatePending {
		return nil, false
	}
	if r.state.touch.id != id || r.state.touchDragOffered {
		return nil, false
	}
	r.state.touchDragOffered = true
	return recognizedTouchGestureTouchDrag{TouchDragEvent{
		Phase:         TouchPhaseStarted,
		StartPosition: r.state.touch.startPosition,
		Position:      r.state.touch.lastPosition,
	}}, true
}

func (r *touchGestureRecognizer) resolveTouchDrag(claimed bool) {
	if !claimed {
		return
	}
	if r.state.kind == touchGestureStatePending && r.state.touchDragOffered {
		r.state = touchGestureState{kind: touchGestureStateTouchDragging, touch: r.state.touch}
	}
}

func (r *touchGestureRecognizer) hasMomentum() bool {
	return r.momentum != nil
}

func (r *touchGestureRecognizer) tickMomentum() (recognizedTouchGesture, bool) {
	return r.tickMomentumAt(scheduler.InstantNow())
}

func (r *touchGestureRecognizer) tickMomentumAt(now scheduler.Instant) (recognizedTouchGesture, bool) {
	m := r.momentum
	if m == nil {
		return nil, false
	}
	elapsed := now.Sub(m.startedAt)
	distance := r.tuning.ScrollPhysics.flingDistance(m.speed, elapsed)
	step := max(distance-m.emittedDistance, 0.)
	m.emittedDistance = max(m.emittedDistance, distance)
	delta := NewPoint(Px(m.direction.X*step), Px(m.direction.Y*step))
	position := m.position
	if elapsed >= m.duration {
		r.momentum = nil
		return recognizedTouchGestureScroll{touchScrollEvent(position, delta, TouchPhaseEnded)}, true
	}
	return recognizedTouchGestureScroll{touchScrollEvent(position, delta, TouchPhaseMoved)}, true
}

func touchScrollEvent(position, delta Point[Pixels], touchPhase TouchPhase) ScrollEvent {
	return ScrollEvent{
		Position:   position,
		Delta:      ScrollDeltaPixels(delta),
		Modifiers:  Modifiers{},
		TouchPhase: touchPhase,
		Source:     ScrollSourceTouch,
	}
}

type velocitySample struct {
	time     scheduler.Instant
	position Point[Pixels]
}

type velocityTracker struct {
	samples []velocitySample
}

func (v *velocityTracker) push(t scheduler.Instant, position Point[Pixels]) {
	v.samples = append(v.samples, velocitySample{time: t, position: position})
	for len(v.samples) > velocityMaxSamples {
		v.samples = v.samples[1:]
	}
}

func (v *velocityTracker) latestSampleTime() (scheduler.Instant, bool) {
	if len(v.samples) == 0 {
		return scheduler.Instant{}, false
	}
	return v.samples[len(v.samples)-1].time, true
}

func (v *velocityTracker) velocity() Point[float32] {
	if len(v.samples) == 0 {
		return Point[float32]{}
	}
	newestTime := v.samples[len(v.samples)-1].time
	timesSeconds := make([]float64, 0, velocityMaxSamples)
	horizontal := make([]float64, 0, velocityMaxSamples)
	vertical := make([]float64, 0, velocityMaxSamples)
	previousTime := newestTime
	for i := len(v.samples) - 1; i >= 0; i-- {
		sample := v.samples[i]
		age := newestTime.Sub(sample.time)
		if age > velocityWindow || previousTime.Sub(sample.time) > velocityAssumeStoppedGap {
			break
		}
		previousTime = sample.time
		timesSeconds = append(timesSeconds, -age.Seconds())
		horizontal = append(horizontal, float64(float32(sample.position.X)))
		vertical = append(vertical, float64(float32(sample.position.Y)))
	}

	endpointEstimate := func(values []float64) float32 {
		var elapsed float64
		if len(timesSeconds) > 0 {
			elapsed = -timesSeconds[len(timesSeconds)-1]
		}
		if elapsed <= 2.220446049250313e-16 {
			return 0.
		}
		var first, last float64
		if len(values) > 0 {
			first, last = values[0], values[len(values)-1]
		}
		return float32((first - last) / elapsed)
	}
	if len(timesSeconds) < 3 {
		return NewPoint(endpointEstimate(horizontal), endpointEstimate(vertical))
	}
	estimate := func(values []float64) float32 {
		if velocity, ok := quadraticVelocityAtNewest(timesSeconds, values); ok {
			return float32(velocity)
		}
		return endpointEstimate(values)
	}
	return NewPoint(estimate(horizontal), estimate(vertical))
}

func quadraticVelocityAtNewest(times, values []float64) (float64, bool) {
	count := float64(len(times))
	var sumT1, sumT2, sumT3, sumT4 float64
	var sumV, sumVt, sumVt2 float64
	for i, t := range times {
		if i >= len(values) {
			break
		}
		value := values[i]
		timeSquared := t * t
		sumT1 += t
		sumT2 += timeSquared
		sumT3 += timeSquared * t
		sumT4 += timeSquared * timeSquared
		sumV += value
		sumVt += value * t
		sumVt2 += value * timeSquared
	}
	determinant := count*(sumT2*sumT4-sumT3*sumT3) -
		sumT1*(sumT1*sumT4-sumT3*sumT2) +
		sumT2*(sumT1*sumT3-sumT2*sumT2)
	if math.Abs(determinant) < 1e-12 {
		return 0, false
	}
	linearDeterminant := count*(sumVt*sumT4-sumT3*sumVt2) -
		sumV*(sumT1*sumT4-sumT3*sumT2) +
		sumT2*(sumT1*sumVt2-sumVt*sumT2)
	return linearDeterminant / determinant, true
}
