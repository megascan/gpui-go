package gpui

import (
	"math"
	"time"
)

const criticalDampingTolerance float32 = 1e-4
const defaultSpringEpsilon float32 = 0.001

const durationMax = time.Duration(math.MaxInt64)

// SpringConfig holds the physical parameters of a damped harmonic oscillator. Stiffness and
// Mass must be finite and positive; Damping must be finite and non-negative.
type SpringConfig struct {
	// Stiffness is the spring stiffness, conventionally written as k.
	Stiffness float32
	// Damping is the viscous damping coefficient, conventionally written as c.
	Damping float32
	// Mass is the moving mass, conventionally written as m.
	Mass float32
}

// NewSpringConfig creates a spring from its physical parameters.
func NewSpringConfig(stiffness, damping, mass float32) SpringConfig {
	return SpringConfig{Stiffness: stiffness, Damping: damping, Mass: mass}
}

func expF32(v float32) float32 { return float32(math.Exp(float64(v))) }

// Canonical returns the natural angular frequency and damping ratio.
func (c SpringConfig) Canonical() (float32, float32) {
	naturalFrequency := sqrtF32(c.Stiffness / c.Mass)
	dampingRatio := c.Damping / (2.0 * sqrtF32(c.Stiffness*c.Mass))
	return naturalFrequency, dampingRatio
}

// Step advances a spring toward a target that remains fixed for deltaTime seconds.
func (c SpringConfig) Step(state SpringState, target, deltaTime float32) SpringState {
	propagator := c.Propagator(deltaTime)
	displacement := state.Position - target
	return SpringState{
		Position: target + propagator[0][0]*displacement + propagator[0][1]*state.Velocity,
		Velocity: propagator[1][0]*displacement + propagator[1][1]*state.Velocity,
	}
}

// StepRamp advances a spring toward a target moving at a constant velocity.
func (c SpringConfig) StepRamp(state SpringState, target, targetVelocity, deltaTime float32) SpringState {
	naturalFrequency, dampingRatio := c.Canonical()
	steadyStateLag := -2.0 * dampingRatio * targetVelocity / naturalFrequency
	displacement := state.Position - target - steadyStateLag
	velocity := state.Velocity - targetVelocity
	propagator := c.Propagator(deltaTime)
	target = target + targetVelocity*deltaTime
	return SpringState{
		Position: target + steadyStateLag + propagator[0][0]*displacement + propagator[0][1]*velocity,
		Velocity: targetVelocity + propagator[1][0]*displacement + propagator[1][1]*velocity,
	}
}

// Propagator returns the exact state-transition matrix for a constant target over deltaTime.
func (c SpringConfig) Propagator(deltaTime float32) [2][2]float32 {
	naturalFrequency, dampingRatio := c.Canonical()

	if dampingRatio < 1.0-criticalDampingTolerance {
		decay := dampingRatio * naturalFrequency
		dampedFrequency := naturalFrequency * sqrtF32(1.0-dampingRatio*dampingRatio)
		exponential := expF32(-decay * deltaTime)
		s, co := math.Sincos(float64(dampedFrequency * deltaTime))
		sine, cosine := float32(s), float32(co)
		sineOverFrequency := sine / dampedFrequency
		return [2][2]float32{
			{exponential * (cosine + decay*sineOverFrequency), exponential * sineOverFrequency},
			{-exponential * naturalFrequency * naturalFrequency * sineOverFrequency, exponential * (cosine - decay*sineOverFrequency)},
		}
	} else if dampingRatio > 1.0+criticalDampingTolerance {
		root := sqrtF32(dampingRatio*dampingRatio - 1.0)
		rootSum := dampingRatio + root
		slowRoot := -naturalFrequency / rootSum
		fastRoot := -naturalFrequency * rootSum
		denominator := slowRoot - fastRoot
		slowExponential := expF32(slowRoot * deltaTime)
		fastExponential := expF32(fastRoot * deltaTime)
		return [2][2]float32{
			{(-fastRoot*slowExponential + slowRoot*fastExponential) / denominator, (slowExponential - fastExponential) / denominator},
			{slowRoot * fastRoot * (fastExponential - slowExponential) / denominator, (slowRoot*slowExponential - fastRoot*fastExponential) / denominator},
		}
	}
	exponential := expF32(-naturalFrequency * deltaTime)
	return [2][2]float32{
		{exponential * (1.0 + naturalFrequency*deltaTime), exponential * deltaTime},
		{-exponential * naturalFrequency * naturalFrequency * deltaTime, exponential * (1.0 - naturalFrequency*deltaTime)},
	}
}

// IsSettled tests both displacement and velocity against a positional tolerance.
func (c SpringConfig) IsSettled(state SpringState, target, epsilon float32) bool {
	naturalFrequency, _ := c.Canonical()
	return isFiniteF32(epsilon) &&
		epsilon >= 0.0 &&
		absF32(state.Position-target) <= epsilon &&
		absF32(state.Velocity) <= epsilon*naturalFrequency
}

func hypotF32(a, b float32) float32 { return float32(math.Hypot(float64(a), float64(b))) }

// SettleTime returns a conservative time after which the spring remains settled. An undamped
// spring has no finite settling time and returns the largest time.Duration.
func (c SpringConfig) SettleTime(state SpringState, target, epsilon float32) time.Duration {
	displacement := state.Position - target
	if displacement == 0.0 && state.Velocity == 0.0 {
		return 0
	}

	naturalFrequency, dampingRatio := c.Canonical()
	if !isFiniteF32(naturalFrequency) ||
		naturalFrequency <= 0.0 ||
		!isFiniteF32(dampingRatio) ||
		dampingRatio <= 0.0 ||
		!isFiniteF32(epsilon) ||
		epsilon <= 0.0 {
		return durationMax
	}

	velocityThreshold := epsilon * naturalFrequency

	if dampingRatio < 1.0-criticalDampingTolerance {
		decay := dampingRatio * naturalFrequency
		dampedFrequency := naturalFrequency * sqrtF32(1.0-dampingRatio*dampingRatio)
		sineCoefficient := (state.Velocity + decay*displacement) / dampedFrequency
		positionEnvelope := hypotF32(displacement, sineCoefficient)
		velocityCosine := dampedFrequency*sineCoefficient - decay*displacement
		velocitySine := -dampedFrequency*displacement - decay*sineCoefficient
		velocityEnvelope := hypotF32(velocityCosine, velocitySine)
		return findSettleTime(epsilon, velocityThreshold, 0.0, naturalFrequency, func(t float32) (float32, float32) {
			exponential := expF32(-decay * t)
			return positionEnvelope * exponential, velocityEnvelope * exponential
		})
	} else if dampingRatio > 1.0+criticalDampingTolerance {
		root := sqrtF32(dampingRatio*dampingRatio - 1.0)
		rootSum := dampingRatio + root
		slowRoot := -naturalFrequency / rootSum
		fastRoot := -naturalFrequency * rootSum
		denominator := slowRoot - fastRoot
		slowCoefficient := (state.Velocity - fastRoot*displacement) / denominator
		fastCoefficient := (slowRoot*displacement - state.Velocity) / denominator
		return findSettleTime(epsilon, velocityThreshold, 0.0, naturalFrequency, func(t float32) (float32, float32) {
			slowTerm := absF32(slowCoefficient) * expF32(slowRoot*t)
			fastTerm := absF32(fastCoefficient) * expF32(fastRoot*t)
			return slowTerm + fastTerm, absF32(slowRoot)*slowTerm + absF32(fastRoot)*fastTerm
		})
	}
	linearCoefficient := state.Velocity + naturalFrequency*displacement
	positionConstant := absF32(displacement)
	positionLinear := absF32(linearCoefficient)
	velocityConstant := absF32(linearCoefficient - naturalFrequency*displacement)
	velocityLinear := naturalFrequency * absF32(linearCoefficient)
	positionDecayStart := envelopeDecayStart(positionConstant, positionLinear, naturalFrequency)
	velocityDecayStart := envelopeDecayStart(velocityConstant, velocityLinear, naturalFrequency)
	return findSettleTime(epsilon, velocityThreshold, max(positionDecayStart, velocityDecayStart), naturalFrequency, func(t float32) (float32, float32) {
		exponential := expF32(-naturalFrequency * t)
		return (positionConstant + positionLinear*t) * exponential, (velocityConstant + velocityLinear*t) * exponential
	})
}

// SpringState is the instantaneous position and velocity of a spring.
type SpringState struct {
	// Position is the current value in the animated unit.
	Position float32
	// Velocity is the current value's change per second.
	Velocity float32
}

// SpringTarget is a value that can be targeted by a one-dimensional spring. float32 and bool
// are spring targets too (bool resolves to AnimationPhase); see springTargetOf.
type SpringTarget[O any] interface {
	// Target returns the target in the spring's coordinate space.
	Target() float32
	// Resolve projects a spring coordinate into the animated output.
	Resolve(value float32) O
}

func springTargetOf[T any](target T) float32 {
	switch t := any(target).(type) {
	case float32:
		return t
	case bool:
		if t {
			return 1.0
		}
		return 0.0
	case interface{ Target() float32 }:
		return t.Target()
	}
	panic("gpui: type does not implement SpringTarget")
}

func springResolveOf[T any](target T, value float32) any {
	switch t := any(target).(type) {
	case float32:
		return value
	case bool:
		return AnimationPhase(value)
	case Pixels:
		return t.Resolve(value)
	case Rems:
		return t.Resolve(value)
	case AnimationPhase:
		return t.Resolve(value)
	}
	panic("gpui: type does not implement SpringTarget")
}

// Target returns p as a spring coordinate.
func (p Pixels) Target() float32 { return p.AsF32() }

// Resolve returns value as Pixels.
func (Pixels) Resolve(value float32) Pixels { return Pixels(value) }

// Target returns r as a spring coordinate.
func (r Rems) Target() float32 { return float32(r) }

// Resolve returns value as Rems.
func (Rems) Resolve(value float32) Rems { return Rems(value) }

// AnimationPhase is a potentially overshooting coordinate within an animation; phases are not
// restricted to 0..1.
type AnimationPhase float32

// Clamp restricts this phase to the bounds of the inclusive range first..=second.
func (p AnimationPhase) Clamp(first, second float32) AnimationPhase {
	return AnimationPhase(clampF32(float32(p), min(first, second), max(first, second)))
}

// Interpolate interpolates between values using 0 and 1 as their phase coordinates.
func (p AnimationPhase) Interpolate[T any](from, to T) T {
	return interpolateValue(from, to, float32(p))
}

// InterpolateClamped interpolates between values without extrapolating beyond 0 and 1.
func (p AnimationPhase) InterpolateClamped[T any](from, to T) T {
	return interpolateValue(from, to, clampF32(float32(p), 0.0, 1.0))
}

// InterpolateBetween interpolates between values assigned to the phase coordinates start..=end.
func (p AnimationPhase) InterpolateBetween[T any](start, end float32, from, to T) T {
	var phase float32
	if start == end {
		if float32(p) < start {
			phase = 0.0
		} else {
			phase = 1.0
		}
	} else {
		phase = (float32(p) - start) / (end - start)
	}
	return interpolateValue(from, to, phase)
}

// InterpolateBetweenClamped interpolates over start..=end without extrapolating.
func (p AnimationPhase) InterpolateBetweenClamped[T any](start, end float32, from, to T) T {
	var phase float32
	if start == end {
		if float32(p) < start {
			phase = 0.0
		} else {
			phase = 1.0
		}
	} else {
		phase = clampF32((float32(p)-start)/(end-start), 0.0, 1.0)
	}
	return interpolateValue(from, to, phase)
}

// AnimationPhaseFromBool returns 1 for true and 0 for false.
func AnimationPhaseFromBool(value bool) AnimationPhase {
	if value {
		return 1.0
	}
	return 0.0
}

// Target returns p as a spring coordinate.
func (p AnimationPhase) Target() float32 { return float32(p) }

// Resolve returns value as an AnimationPhase.
func (AnimationPhase) Resolve(value float32) AnimationPhase { return AnimationPhase(value) }

// Interpolate is a value that supports linear interpolation and extrapolation. float32 is
// interpolated too.
type Interpolate[T any] interface {
	// Interpolate resolves the value at phase, where 0 is from and 1 is to.
	Interpolate(from, to T, phase float32) T
}

func interpolateF32(from, to, phase float32) float32 {
	return from + (to-from)*phase
}

func interpolateValue[T any](from, to T, phase float32) T {
	switch f := any(from).(type) {
	case float32:
		return any(interpolateF32(f, any(to).(float32), phase)).(T)
	case Interpolate[T]:
		return f.Interpolate(from, to, phase)
	}
	panic("gpui: type does not implement Interpolate")
}

// Interpolate resolves the value at phase, where 0 is from and 1 is to.
func (Pixels) Interpolate(from, to Pixels, phase float32) Pixels {
	return from + (to-from)*Pixels(phase)
}

// Interpolate resolves the value at phase, where 0 is from and 1 is to.
func (Rems) Interpolate(from, to Rems, phase float32) Rems {
	return from + (to-from)*Rems(phase)
}

// Interpolate resolves the value at phase, where 0 is from and 1 is to.
func (Rgba) Interpolate(from, to Rgba, phase float32) Rgba {
	return Rgba{
		R: interpolateF32(from.R, to.R, phase),
		G: interpolateF32(from.G, to.G, phase),
		B: interpolateF32(from.B, to.B, phase),
		A: interpolateF32(from.A, to.A, phase),
	}
}

// Interpolate resolves the value at phase along the shortest hue path.
func (Hsla) Interpolate(from, to Hsla, phase float32) Hsla {
	hueDelta := remEuclidF32(to.H-from.H+0.5, 1.0) - 0.5
	return Hsla{
		H: remEuclidF32(from.H+hueDelta*phase, 1.0),
		S: interpolateF32(from.S, to.S, phase),
		L: interpolateF32(from.L, to.L, phase),
		A: interpolateF32(from.A, to.A, phase),
	}
}

// SpringPlayback controls how a spring advances and resolves its presentation value.
type SpringPlayback uint8

const (
	// SpringPlaybackRunning advances toward the latest target, preserving velocity across retargets.
	SpringPlaybackRunning SpringPlayback = iota
	// SpringPlaybackPaused holds the current position and velocity until playback resumes.
	SpringPlaybackPaused
	// SpringPlaybackStopped holds the current position and discards velocity.
	SpringPlaybackStopped
	// SpringPlaybackCompleted snaps to the latest target and discards velocity.
	SpringPlaybackCompleted
	// SpringPlaybackCancelled returns to the initial value and discards velocity.
	SpringPlaybackCancelled
)

// SpringAnimation is a stateful spring animation targeting a value or projected path. The
// target type is struct{} until To sets it.
type SpringAnimation[T any] struct {
	config   SpringConfig
	target   T
	epsilon  float32
	initial  Option[float32]
	playback SpringPlayback
}

// NewSpringAnimation creates a spring animation builder.
func NewSpringAnimation(config SpringConfig) SpringAnimation[struct{}] {
	return SpringAnimation[struct{}]{
		config:   config,
		epsilon:  defaultSpringEpsilon,
		playback: SpringPlaybackRunning,
	}
}

// To sets the value or path targeted by this spring.
func (a SpringAnimation[T]) To[U any](target U) SpringAnimation[U] {
	springTargetOf(target)
	return SpringAnimation[U]{
		config:   a.config,
		target:   target,
		epsilon:  a.epsilon,
		initial:  a.initial,
		playback: a.playback,
	}
}

// WithEpsilon sets the settling tolerance in the target's scalar coordinate space.
func (a SpringAnimation[T]) WithEpsilon(epsilon float32) SpringAnimation[T] {
	a.epsilon = epsilon
	return a
}

// Playback sets how the spring advances or resolves its current value.
func (a SpringAnimation[T]) Playback(playback SpringPlayback) SpringAnimation[T] {
	a.playback = playback
	return a
}

// From sets the coordinate used when this element has no prior spring state.
func (a SpringAnimation[T]) From(initial T) SpringAnimation[T] {
	a.initial = Some(springTargetOf(initial))
	return a
}

// SampledEasing adapts a spring starting at zero with no velocity to GPUI's duration-based
// easing API. The returned easing can overshoot the normalized 0..1 range.
func SampledEasing(config SpringConfig, epsilon float32) (time.Duration, func(float32) float32) {
	initialState := SpringState{Position: 0.0, Velocity: 0.0}
	duration := config.SettleTime(initialState, 1.0, epsilon)
	durationSeconds := float32(duration.Seconds())
	return duration, func(progress float32) float32 {
		if progress <= 0.0 {
			return 0.0
		} else if progress >= 1.0 {
			return 1.0
		}
		return config.Step(initialState, 1.0, progress*durationSeconds).Position
	}
}

func envelopeDecayStart(constant, linear, decay float32) float32 {
	if linear == 0.0 {
		return 0.0
	}
	return max(1.0/decay-constant/linear, 0.0)
}

func findSettleTime(positionThreshold, velocityThreshold, decayStart, naturalFrequency float32, envelope func(float32) (float32, float32)) time.Duration {
	isBelowThreshold := func(t float32) bool {
		position, velocity := envelope(t)
		return position <= positionThreshold && velocity <= velocityThreshold
	}

	if isBelowThreshold(decayStart) {
		return durationFromSecs(decayStart)
	}

	lowerBound := decayStart
	upperBound := max(decayStart, 1.0/naturalFrequency)
	for !isBelowThreshold(upperBound) {
		lowerBound = upperBound
		upperBound *= 2.0
		if !isFiniteF32(upperBound) {
			return durationMax
		}
	}

	for range 32 {
		midpoint := (lowerBound + upperBound) / 2.0
		if isBelowThreshold(midpoint) {
			upperBound = midpoint
		} else {
			lowerBound = midpoint
		}
	}

	return durationFromSecs(upperBound)
}

func durationFromSecs(seconds float32) time.Duration {
	if !isFiniteF32(seconds) || seconds >= float32(durationMax.Seconds()) {
		return durationMax
	} else if seconds <= 0.0 {
		return 0
	}
	return durationFromSecsF32(seconds)
}

func durationFromSecsF32(seconds float32) time.Duration {
	return time.Duration(float64(seconds) * 1e9)
}
