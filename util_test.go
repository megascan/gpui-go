package gpui

import (
	"errors"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func TestUtil_RoundHalfTowardZero(t *testing.T) {
	cases := [][2]float32{
		{0.5, 0}, {1.5, 1}, {2.5, 2}, {-0.5, 0}, {-1.5, -1}, {-2.5, -2},
		{1.5001, 2}, {1.4999, 1}, {-1.5001, -2}, {-1.4999, -1},
		{0, 0}, {3, 3}, {-3, -3},
	}
	for _, c := range cases {
		if got := roundHalfTowardZero(c[0]); got != c[1] {
			t.Errorf("roundHalfTowardZero(%v) = %v, want %v", c[0], got, c[1])
		}
	}
}

func TestUtil_DevicePixelHelpers(t *testing.T) {
	check := func(name string, got, want float32) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	check("round(1.0, 1.5)", roundToDevicePixel(1.0, 1.5), 1.0)
	check("round(0.3, 2.0)", roundToDevicePixel(0.3, 2.0), 1.0)
	check("round(1.4, 1.0)", roundToDevicePixel(1.4, 1.0), 1.0)
	check("round(1.6, 1.0)", roundToDevicePixel(1.6, 1.0), 2.0)

	check("stroke(0.0, 1.0)", roundStrokeToDevicePixel(0.0, 1.0), 0.0)
	check("stroke(0.4, 1.0)", roundStrokeToDevicePixel(0.4, 1.0), 1.0)
	check("stroke(0.5, 1.0)", roundStrokeToDevicePixel(0.5, 1.0), 1.0)
	check("stroke(1.0, 1.5)", roundStrokeToDevicePixel(1.0, 1.5), 1.0)
	check("stroke(1.6, 1.0)", roundStrokeToDevicePixel(1.6, 1.0), 2.0)

	check("floor(0.3, 2.0)", floorToDevicePixel(0.3, 2.0), 0.0)
	check("ceil(0.3, 2.0)", ceilToDevicePixel(0.3, 2.0), 1.0)
	check("floor(2.1, 1.0)", floorToDevicePixel(2.1, 1.0), 2.0)
	check("ceil(2.1, 1.0)", ceilToDevicePixel(2.1, 1.0), 3.0)

	check("round(2.0, 2.0)", roundToDevicePixel(2.0, 2.0), 4.0)
	check("floor(2.0, 2.0)", floorToDevicePixel(2.0, 2.0), 4.0)
	check("ceil(2.0, 2.0)", ceilToDevicePixel(2.0, 2.0), 4.0)
}

func TestUtil_RoundHalfTowardZeroF64(t *testing.T) {
	cases := [][2]float64{{0.5, 0}, {-0.5, 0}, {1.5, 1}, {-1.5, -1}, {2.5001, 3}}
	for _, c := range cases {
		if got := roundHalfTowardZeroF64(c[0]); got != c[1] {
			t.Errorf("roundHalfTowardZeroF64(%v) = %v, want %v", c[0], got, c[1])
		}
	}
}

func TestUtil_WithTimeout(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			ready := scheduler.Await[scheduler.Result[scheduler.Result[struct{}]]](s, FutureWithTimeout[scheduler.Result[struct{}]](scheduler.ReadyTask(struct{}{}), time.Second, cx.Executor()))
			if ready.Err != nil {
				t.Fatalf("Timeout should be noop: %v", ready.Err)
			}

			longDuration := 6000 * time.Second
			shortDuration := time.Second
			timedOut := scheduler.Await[scheduler.Result[scheduler.Result[struct{}]]](s, FutureWithTimeout[scheduler.Result[struct{}]](cx.Executor().Timer(longDuration), shortDuration, cx.Executor()))
			if !errors.Is(timedOut.Err, Timeout{}) {
				t.Fatalf("timeout should have triggered, got %v", timedOut.Err)
			}

			fut := FutureWithTimeout[scheduler.Result[struct{}]](cx.Executor().Timer(longDuration), shortDuration, cx.Executor())
			cx.Executor().AdvanceClock(shortDuration * 2)
			result, ok := fut.Poll(scheduler.NewContext(scheduler.WakerFn(func() {})))
			if !ok {
				panic("timeout should have triggered")
			}
			if result.Err == nil {
				t.Fatal("timeout: expected an error")
			}
		})
	})
}
