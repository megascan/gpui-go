//go:build windows

package windows

import (
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
)

func TestVsync_SuccessfulCompositorWaitReportsNativeCallback(t *testing.T) {
	provider := &VSyncProvider{
		interval: 0,
		f: func() bool {
			time.Sleep(vsyncIntervalThreshold * 2)
			return true
		},
	}

	if got := provider.WaitForVsync(); got != gpui.FrameRequestSourceNativeCallback {
		t.Fatalf("assertion `left == right` failed\n  left: %v\n right: %v", got, gpui.FrameRequestSourceNativeCallback)
	}
}

func TestVsync_FailedCompositorWaitReportsLocalScheduleEvenAfterThreshold(t *testing.T) {
	provider := &VSyncProvider{
		interval: 0,
		f: func() bool {
			time.Sleep(vsyncIntervalThreshold * 2)
			return false
		},
	}

	if got := provider.WaitForVsync(); got != gpui.FrameRequestSourceLocalSchedule {
		t.Fatalf("assertion `left == right` failed\n  left: %v\n right: %v", got, gpui.FrameRequestSourceLocalSchedule)
	}
}

func TestVsync_RetrieveDurationHandlesLowTickRates(t *testing.T) {
	cases := []struct {
		counts, ticksPerSecond uint64
		want                   time.Duration
	}{
		{166_666, 10_000_000, 16_666 * time.Microsecond},
		{1, 60, 16_666 * time.Microsecond},
		{1001, 60_000, 16_683 * time.Microsecond},
		{1, 144, 6_944 * time.Microsecond},
	}
	for _, c := range cases {
		got, err := retrieveDuration(c.counts, c.ticksPerSecond)
		if err != nil || got != c.want {
			t.Fatalf("retrieveDuration(%d, %d) = %v, %v; want %v", c.counts, c.ticksPerSecond, got, err, c.want)
		}
	}
	if _, err := retrieveDuration(1, 0); err == nil {
		t.Fatal("retrieveDuration with zero ticks per second returned no error")
	}
}
