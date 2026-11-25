package taffy

import "testing"

func TestStyleFloat_Methods(t *testing.T) {
	if !FloatLeft.IsFloated() || !FloatRight.IsFloated() || FloatNone.IsFloated() {
		t.Error("IsFloated")
	}
	if FloatLeft.FloatDirection() != some(FloatDirectionLeft) || FloatRight.FloatDirection() != some(FloatDirectionRight) || FloatNone.FloatDirection().IsSome() {
		t.Error("FloatDirection")
	}
	if FloatDefault != FloatNone || ClearDefault != ClearNone || FloatDirectionRight != 1 {
		t.Error("defaults")
	}
}
