//go:build linux

package common

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/xkb"
	"github.com/megascan/gpui-go/scheduler"
	"golang.org/x/sys/unix"
)

func TestPlatform_PowerModeCallbacksCanReenterAndReplaceThemselves(t *testing.T) {
	eventLoop, err := NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	defer eventLoop.Close()
	common, mainReceiver, powerReceiver := NewLinuxCommon(eventLoop.GetSignal())
	defer common.Close()
	defer mainReceiver.Close()
	defer powerReceiver.Close()

	type call struct {
		mode     gpui.PowerMode
		replaced bool
	}
	var calls []call
	common.Callbacks.PowerModeChange = func() {
		calls = append(calls, call{common.PowerMode, false})
		common.Callbacks.PowerModeChange = func() {
			calls = append(calls, call{common.PowerMode, true})
		}
	}
	for _, mode := range []gpui.PowerMode{gpui.PowerModeLowPower, gpui.PowerModeLowPower, gpui.PowerModeBalanced} {
		handleSystemPowerEvent(common, SystemPowerEventModeChangedTo(mode))
	}
	want := []call{{gpui.PowerModeLowPower, false}, {gpui.PowerModeBalanced, true}}
	if !slices.Equal(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestPlatform_RejectsNullXkbContext(t *testing.T) {
	context := xkb.ContextFromRawPtr(0)
	_, err := validateXkbContext(context)
	if err == nil {
		t.Fatal("null XKB context should be rejected")
	}
	if got := err.Error(); got != "libxkbcommon failed to create an XKB context" {
		t.Fatalf("error = %q", got)
	}
}

func TestPlatform_IsWithinClickDistance(t *testing.T) {
	pt := func(x, y float32) gpui.Point[gpui.Pixels] {
		return gpui.Point[gpui.Pixels]{X: gpui.Px(x), Y: gpui.Px(y)}
	}
	zero := pt(0, 0)
	if !IsWithinClickDistance(zero, pt(5, 5)) {
		t.Fatal("(5, 5) should be within the click distance")
	}
	if !IsWithinClickDistance(zero, pt(-4.9, 5)) {
		t.Fatal("(-4.9, 5) should be within the click distance")
	}
	if !IsWithinClickDistance(pt(3, 2), pt(-2, -2)) {
		t.Fatal("(3, 2) and (-2, -2) should be within the click distance")
	}
	if IsWithinClickDistance(zero, pt(5, 5.1)) {
		t.Fatal("(5, 5.1) should not be within the click distance")
	}
}

type guardResult = scheduler.Result[*gpui.ActivityGuard]

func releaseGuard(released *atomic.Int64) *gpui.ActivityGuard {
	return gpui.NewActivityGuard(func() { released.Add(1) })
}

func TestPlatformIdleSleepPrevention_GuardPassesThroughAndReleasesOnDrop(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		var released atomic.Int64
		guardTx, guardRx := scheduler.Oneshot[guardResult]()
		if err := guardTx.Send(guardResult{Value: releaseGuard(&released)}); err != nil {
			t.Fatal(err)
		}
		tc.BlockTest(func(s *scheduler.Scope) {
			guard, err := AwaitIdleSleepPrevention(s, guardRx, tc.BackgroundExecutor())
			if err != nil {
				t.Fatalf("acquisition should succeed: %v", err)
			}
			if released.Load() != 0 {
				t.Fatal("released before drop")
			}
			guard.Release()
			if released.Load() != 1 {
				t.Fatal("not released on drop")
			}
		})
	})
}

func TestPlatformIdleSleepPrevention_AcquisitionErrorPassesThrough(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		guardTx, guardRx := scheduler.Oneshot[guardResult]()
		if err := guardTx.Send(guardResult{Err: errors.New("inhibition rejected")}); err != nil {
			t.Fatal(err)
		}
		tc.BlockTest(func(s *scheduler.Scope) {
			_, err := AwaitIdleSleepPrevention(s, guardRx, tc.BackgroundExecutor())
			if err == nil {
				t.Fatal("acquisition should fail")
			}
			if err.Error() != "inhibition rejected" {
				t.Fatalf("error = %q", err)
			}
		})
	})
}

func TestPlatformIdleSleepPrevention_AbandonedAcquisitionIsAnError(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		guardTx, guardRx := scheduler.Oneshot[guardResult]()
		guardTx.Close()
		tc.BlockTest(func(s *scheduler.Scope) {
			_, err := AwaitIdleSleepPrevention(s, guardRx, tc.BackgroundExecutor())
			if err == nil {
				t.Fatal("acquisition should fail")
			}
			if err.Error() != "Idle sleep prevention request was abandoned" {
				t.Fatalf("error = %q", err)
			}
		})
	})
}

func TestPlatformIdleSleepPrevention_LateGuardAfterTimeoutIsReleased(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		var released atomic.Int64
		guardTx, guardRx := scheduler.Oneshot[guardResult]()
		executor := tc.BackgroundExecutor()
		task := executor.Spawn(func(s *scheduler.Scope) (*gpui.ActivityGuard, error) {
			return AwaitIdleSleepPrevention(s, guardRx, executor)
		})
		executor.RunUntilParked()
		executor.AdvanceClock(9 * time.Second)
		executor.RunUntilParked()
		if guardTx.IsCanceled() {
			t.Fatal("receiver dropped before the timeout")
		}
		executor.AdvanceClock(1 * time.Second)
		tc.BlockTest(func(s *scheduler.Scope) {
			_, err := task.Await(s)
			if err == nil {
				t.Fatal("acquisition should time out")
			}
			if err.Error() != "Idle sleep prevention acquisition timed out after 10 seconds" {
				t.Fatalf("error = %q", err)
			}
		})
		if !guardTx.IsCanceled() {
			t.Fatal("receiver should be dropped after the timeout")
		}
		guard := releaseGuard(&released)
		if err := guardTx.Send(guardResult{Value: guard}); err != nil {
			if released.Load() != 0 {
				t.Fatal("released before drop")
			}
			guard.Release()
		}
		if released.Load() != 1 {
			t.Fatal("late guard was not released")
		}
	})
}

func newPipe(t *testing.T) (int, int) {
	t.Helper()
	var fds [2]int
	if err := unix.Pipe2(fds[:], unix.O_CLOEXEC); err != nil {
		t.Fatal(err)
	}
	return fds[0], fds[1]
}

func writeAll(t *testing.T, fd int, data []byte) {
	for len(data) > 0 {
		n, err := unix.Write(fd, data)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			t.Error(err)
			return
		}
		data = data[n:]
	}
}

func TestPlatformReadFdWithTimeout_ReadsDataWrittenBeforeClose(t *testing.T) {
	read, write := newPipe(t)
	writeAll(t, write, []byte("hello clipboard"))
	unix.Close(write)

	bytes, err := ReadFdWithTimeout(read, PipeReadTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes) != "hello clipboard" {
		t.Fatalf("bytes = %q", bytes)
	}
}

func TestPlatformReadFdWithTimeout_ReturnsEmptyWhenWriterClosesWithoutWriting(t *testing.T) {
	read, write := newPipe(t)
	unix.Close(write)

	bytes, err := ReadFdWithTimeout(read, PipeReadTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes) != 0 {
		t.Fatalf("bytes = %q", bytes)
	}
}

func TestPlatformReadFdWithTimeout_TimesOutWhenWriterNeverWrites(t *testing.T) {
	read, write := newPipe(t)
	defer unix.Close(write)

	timeout := 50 * time.Millisecond
	started := time.Now()
	_, err := ReadFdWithTimeout(read, timeout)
	elapsed := time.Since(started)

	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < timeout {
		t.Fatal("returned before the timeout elapsed")
	}
}

func TestPlatformReadFdWithTimeout_TimesOutWhenWriterStallsAfterPartialWrite(t *testing.T) {
	read, write := newPipe(t)
	writeAll(t, write, []byte("partial"))
	defer unix.Close(write)

	_, err := ReadFdWithTimeout(read, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlatformReadFdWithTimeout_SlowWriterResetsDeadlineBetweenChunks(t *testing.T) {
	read, write := newPipe(t)
	const chunks = 12
	gap := 40 * time.Millisecond
	timeout := 400 * time.Millisecond

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer unix.Close(write)
		for range chunks {
			time.Sleep(gap)
			writeAll(t, write, bytes.Repeat([]byte{'x'}, 1000))
		}
	}()
	got, err := ReadFdWithTimeout(read, timeout)
	<-done
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bytes.Repeat([]byte{'x'}, 1000*chunks)) {
		t.Fatalf("read %d bytes", len(got))
	}
}

func TestPlatformReadFdWithTimeout_ReadsPayloadLargerThanPipeCapacity(t *testing.T) {
	read, write := newPipe(t)
	payload := bytes.Repeat([]byte{'z'}, 1024*1024)

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer unix.Close(write)
		writeAll(t, write, payload)
	}()
	got, err := ReadFdWithTimeout(read, PipeReadTimeout)
	<-done
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("read %d bytes", len(got))
	}
}
