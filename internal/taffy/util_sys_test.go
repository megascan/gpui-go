package taffy

import (
	"math"
	"testing"
)

func float32FromBits(b uint32) float32 {
	return math.Float32frombits(b)
}

func negZero() float32 {
	return math.Float32frombits(1 << 31)
}

func TestUtilSys_MinMaxNaN(t *testing.T) {
	nan := float32FromBits(0x7fc00000)
	cases := []struct {
		name      string
		got, want float32
	}{
		{"max(nan, 1)", f32Max(nan, 1), 1},
		{"max(1, nan)", f32Max(1, nan), 1},
		{"min(nan, 1)", f32Min(nan, 1), 1},
		{"min(1, nan)", f32Min(1, nan), 1},
		{"max(1, 2)", f32Max(1, 2), 2},
		{"min(1, 2)", f32Min(1, 2), 1},
		{"max(-inf, 2)", f32Max(float32(math.Inf(-1)), 2), 2},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if v := f32Max(nan, nan); v == v {
		t.Errorf("max(nan, nan) = %v, want NaN", v)
	}
}

func TestUtilSys_Rounding(t *testing.T) {
	cases := []struct {
		name      string
		got, want float32
	}{
		{"round(2.5)", round(2.5), 3},
		{"round(-2.5)", round(-2.5), -2},
		{"round(1.4)", round(1.4), 1},
		{"ceil(1.1)", ceil(1.1), 2},
		{"floor(-1.1)", floor(-1.1), -2},
		{"abs(-3)", abs(-3), 3},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if math.Signbit(float64(abs(negZero()))) {
		t.Error("abs(-0.0) must be +0.0")
	}
	if v := newVecWithCapacity[int](4); len(v) != 0 || cap(v) != 4 {
		t.Errorf("newVecWithCapacity = len %d cap %d", len(v), cap(v))
	}
}
