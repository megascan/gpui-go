//go:build windows

package windows

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/bits"
	"sync"
	"time"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/win32"
)

var (
	qpcTicksPerSecondOnce sync.Once
	qpcTicksPerSecondVal  uint64
)

func qpcTicksPerSecond() uint64 {
	qpcTicksPerSecondOnce.Do(func() {
		var frequency int64
		if err := win32.QueryPerformanceFrequency(&frequency); err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		qpcTicksPerSecondVal = uint64(frequency)
	})
	return qpcTicksPerSecondVal
}

const (
	vsyncIntervalThreshold = time.Millisecond
	defaultVsyncInterval   = 16_666 * time.Microsecond
)

// VSyncProvider paces the vsync thread on DWM composition.
type VSyncProvider struct {
	interval time.Duration
	f        func() bool
}

// NewVSyncProvider measures the DWM refresh interval and waits with DwmFlush.
func NewVSyncProvider() *VSyncProvider {
	interval, err := getDwmInterval()
	if err != nil {
		err = fmt.Errorf("Failed to get DWM interval: %w", err)
	}
	interval, ok := gpuiutil.Result(interval, err).LogErr()
	if !ok {
		interval = defaultVsyncInterval
	}
	f := func() bool { return win32.DwmFlush() == nil }
	return &VSyncProvider{interval: interval, f: f}
}

// WaitForVsync blocks until the next composition and reports what paced the frame.
func (p *VSyncProvider) WaitForVsync() gpui.FrameRequestSource {
	vsyncStart := time.Now()
	waitSucceeded := p.f()
	elapsed := time.Since(vsyncStart)
	if !waitSucceeded || elapsed < vsyncIntervalThreshold {
		slog.Debug("VSyncProvider::wait_for_vsync() took less time than expected")
		time.Sleep(p.interval)
		return gpui.FrameRequestSourceLocalSchedule
	}
	return gpui.FrameRequestSourceNativeCallback
}

func getDwmInterval() (time.Duration, error) {
	var timingInfo win32.DWM_TIMING_INFO
	timingInfo.CbSize = uint32(unsafe.Sizeof(timingInfo))
	if err := win32.DwmGetCompositionTimingInfo(0, &timingInfo); err != nil {
		return 0, err
	}
	interval, err := retrieveDuration(timingInfo.QpcRefreshPeriod.Get(), qpcTicksPerSecond())
	if err != nil {
		return 0, err
	}
	if interval < vsyncIntervalThreshold {
		return retrieveDuration(uint64(timingInfo.RateRefresh.UiDenominator), uint64(timingInfo.RateRefresh.UiNumerator))
	}
	return interval, nil
}

func retrieveDuration(counts uint64, ticksPerSecond uint64) (time.Duration, error) {
	if ticksPerSecond == 0 {
		return 0, errors.New("zero ticks per second")
	}
	if ticksPerMicrosecond := ticksPerSecond / 1_000_000; ticksPerMicrosecond > 0 {
		return time.Duration(counts/ticksPerMicrosecond) * time.Microsecond, nil
	}
	hi, lo := bits.Mul64(counts, 1_000_000)
	if hi >= ticksPerSecond {
		return 0, fmt.Errorf("duration of %d ticks at %d ticks per second overflows", counts, ticksPerSecond)
	}
	micros, _ := bits.Div64(hi, lo, ticksPerSecond)
	if micros > uint64(math.MaxInt64/int64(time.Microsecond)) {
		return 0, fmt.Errorf("duration of %d ticks at %d ticks per second overflows", counts, ticksPerSecond)
	}
	return time.Duration(micros) * time.Microsecond, nil
}
