package gpui

import "testing"

const springTestEpsilon float32 = 1e-4

func TestSpring_SpringTargetsResolveTypedOutputs(t *testing.T) {
	if springTargetOf(float32(12.0)) != 12.0 {
		t.Fatal("f32 target")
	}
	if springResolveOf(float32(12.0), 14.0) != any(float32(14.0)) {
		t.Fatal("f32 resolve")
	}
	if Px(12.0).Target() != 12.0 {
		t.Fatal("pixels target")
	}
	if Px(12.0).Resolve(14.0) != Px(14.0) {
		t.Fatal("pixels resolve")
	}
	if springTargetOf(false) != 0.0 {
		t.Fatal("false target")
	}
	if springTargetOf(true) != 1.0 {
		t.Fatal("true target")
	}
	if springResolveOf(true, 1.25) != any(AnimationPhase(1.25)) {
		t.Fatal("bool resolve")
	}
}

func TestSpring_AnimationPhasesInterpolateOverArbitraryRanges(t *testing.T) {
	phase := AnimationPhase(1.5)
	if got := phase.InterpolateBetween[float32](1.0, 2.0, 10.0, 20.0); got != 15.0 {
		t.Fatalf("got %v", got)
	}
	if got := phase.InterpolateBetweenClamped[float32](2.0, 3.0, 10.0, 20.0); got != 10.0 {
		t.Fatalf("got %v", got)
	}
	if got := AnimationPhase(3.5).InterpolateBetween[float32](2.0, 3.0, 10.0, 20.0); got != 25.0 {
		t.Fatalf("got %v", got)
	}
}

func TestSpring_HslaInterpolationTakesTheShortestHuePath(t *testing.T) {
	from := Hsla{H: 0.9, S: 0.5, L: 0.5, A: 1.0}
	to := Hsla{H: 0.1, S: 1.0, L: 0.75, A: 0.5}
	result := AnimationPhase(0.5).Interpolate(from, to)

	if !(result.H < springTestEpsilon || (1.0-result.H) < springTestEpsilon) {
		t.Fatalf("h %v", result.H)
	}
	if absF32(result.S-0.75) >= springTestEpsilon || absF32(result.L-0.625) >= springTestEpsilon || absF32(result.A-0.75) >= springTestEpsilon {
		t.Fatalf("result %+v", result)
	}
}

func TestSpring_PropagatorsComposeAndHaveExpectedDeterminant(t *testing.T) {
	for _, dampingRatio := range []float32{0.4, 1.0, 1.5} {
		var naturalFrequency float32 = 12.0
		config := NewSpringConfig(naturalFrequency*naturalFrequency, 2.0*dampingRatio*naturalFrequency, 1.0)
		first := config.Propagator(0.013)
		second := config.Propagator(0.021)
		combined := springMultiply(second, first)
		direct := config.Propagator(0.034)

		for row := range 2 {
			for column := range 2 {
				if absF32(combined[row][column]-direct[row][column]) >= 2e-4 {
					t.Fatalf("%v: %v != %v", dampingRatio, combined, direct)
				}
			}
		}

		determinant := direct[0][0]*direct[1][1] - direct[0][1]*direct[1][0]
		expected := expF32(-2.0 * dampingRatio * naturalFrequency * 0.034)
		if absF32(determinant-expected) >= 2e-4 {
			t.Fatalf("%v: determinant %v != %v", dampingRatio, determinant, expected)
		}
	}
}

func TestSpring_StepPreservesSemigroupForEveryDampingRegime(t *testing.T) {
	state := SpringState{Position: -3.0, Velocity: 5.0}
	for _, damping := range []float32{4.0, 20.0, 40.0} {
		config := NewSpringConfig(100.0, damping, 1.0)
		stepped := config.Step(config.Step(state, 7.0, 0.013), 7.0, 0.021)
		direct := config.Step(state, 7.0, 0.034)

		if absF32(stepped.Position-direct.Position) >= 2e-4 || absF32(stepped.Velocity-direct.Velocity) >= 2e-4 {
			t.Fatalf("%v: %+v != %+v", damping, stepped, direct)
		}
	}
}

func TestSpring_RampTracksSteadyStateLag(t *testing.T) {
	var naturalFrequency float32 = 10.0
	var dampingRatio float32 = 0.8
	var targetVelocity float32 = 3.0
	config := NewSpringConfig(naturalFrequency*naturalFrequency, 2.0*dampingRatio*naturalFrequency, 1.0)
	lag := -2.0 * dampingRatio * targetVelocity / naturalFrequency
	state := SpringState{Position: lag, Velocity: targetVelocity}
	next := config.StepRamp(state, 0.0, targetVelocity, 0.25)

	if absF32(next.Position-(targetVelocity*0.25+lag)) >= springTestEpsilon {
		t.Fatalf("position %v", next.Position)
	}
	if absF32(next.Velocity-targetVelocity) >= springTestEpsilon {
		t.Fatalf("velocity %v", next.Velocity)
	}
}

func TestSpring_SettlingRequiresLowVelocity(t *testing.T) {
	config := NewSpringConfig(100.0, 10.0, 1.0)
	if config.IsSettled(SpringState{Position: 1.0, Velocity: 1.0}, 1.0, 0.01) {
		t.Fatal("fast spring reported settled")
	}
	if !config.IsSettled(SpringState{Position: 1.005, Velocity: 0.05}, 1.0, 0.01) {
		t.Fatal("slow spring not settled")
	}
}

func TestSpring_SettleTimeIsConservativeForEveryDampingRegime(t *testing.T) {
	initialState := SpringState{Position: -2.0, Velocity: 4.0}
	for _, damping := range []float32{4.0, 20.0, 40.0} {
		config := NewSpringConfig(100.0, damping, 1.0)
		duration := config.SettleTime(initialState, 3.0, 0.001)
		if duration == durationMax {
			t.Fatalf("%v: no settle time", damping)
		}

		for _, additionalTime := range []float32{0.0, 0.1, 1.0} {
			state := config.Step(initialState, 3.0, float32(duration.Seconds())+additionalTime)
			if !config.IsSettled(state, 3.0, 0.001) {
				t.Fatalf("%v: %v produced %+v", damping, duration, state)
			}
		}
	}
}

func TestSpring_SettleTimeAccountsForMotionOutsideAnInstantaneousTolerance(t *testing.T) {
	config := NewSpringConfig(100.0, 2.0, 1.0)
	state := SpringState{Position: 1.125, Velocity: 1.25}
	if !config.IsSettled(state, 1.0, 0.125) {
		t.Fatal("not instantaneously settled")
	}

	duration := config.SettleTime(state, 1.0, 0.125)
	if duration <= 0 {
		t.Fatal("zero settle time")
	}
	if !config.IsSettled(config.Step(state, 1.0, float32(duration.Seconds())), 1.0, 0.125) {
		t.Fatal("not settled after settle time")
	}
}

func TestSpring_UndampedSpringNeverSettles(t *testing.T) {
	config := NewSpringConfig(100.0, 0.0, 1.0)
	if got := config.SettleTime(SpringState{Position: 0.0, Velocity: 0.0}, 1.0, 0.001); got != durationMax {
		t.Fatalf("got %v", got)
	}
}

func TestSpring_SampledEasingHasExactEndpointsAndCanOvershoot(t *testing.T) {
	config := NewSpringConfig(100.0, 6.0, 1.0)
	duration, easing := SampledEasing(config, 0.001)

	if duration == durationMax {
		t.Fatal("no duration")
	}
	if easing(0.0) != 0.0 || easing(1.0) != 1.0 {
		t.Fatal("endpoints")
	}
	overshoot := false
	for step := 1; step < 100; step++ {
		if easing(float32(step)/100.0) > 1.0 {
			overshoot = true
		}
	}
	if !overshoot {
		t.Fatal("no overshoot")
	}
}

func springMultiply(left, right [2][2]float32) [2][2]float32 {
	return [2][2]float32{
		{left[0][0]*right[0][0] + left[0][1]*right[1][0], left[0][0]*right[0][1] + left[0][1]*right[1][1]},
		{left[1][0]*right[0][0] + left[1][1]*right[1][0], left[1][0]*right[0][1] + left[1][1]*right[1][1]},
	}
}
