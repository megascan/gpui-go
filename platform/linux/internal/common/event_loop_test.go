//go:build linux

package common

import (
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

func TestEventLoop_TimersIdlesAndStop(t *testing.T) {
	eventLoop, err := NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	defer eventLoop.Close()
	handle := eventLoop.Handle()
	signal := eventLoop.GetSignal()

	var order []string
	fires := 0
	handle.InsertTimerAfter(time.Millisecond, func(time.Time) TimeoutAction {
		fires++
		order = append(order, "timer")
		if fires < 3 {
			return TimeoutActionToDuration(time.Millisecond)
		}
		handle.InsertIdle(func() {
			order = append(order, "idle")
			signal.Stop()
		})
		return TimeoutActionDrop()
	})
	if err := eventLoop.Run(gpui.None[time.Duration](), func() {}); err != nil {
		t.Fatal(err)
	}
	want := []string{"timer", "timer", "timer", "idle"}
	if !slices.Equal(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestEventLoop_WakeupFromAnotherGoroutine(t *testing.T) {
	eventLoop, err := NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	defer eventLoop.Close()
	signal := eventLoop.GetSignal()
	go func() {
		time.Sleep(10 * time.Millisecond)
		signal.Stop()
		signal.Wakeup()
	}()
	if err := eventLoop.Run(gpui.None[time.Duration](), func() {}); err != nil {
		t.Fatal(err)
	}
}

func TestEventLoop_LinuxDispatcherRunsForegroundAndBackground(t *testing.T) {
	eventLoop, err := NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	defer eventLoop.Close()
	common, mainReceiver, powerReceiver := NewLinuxCommon(eventLoop.GetSignal())
	defer common.Close()
	if err := RegisterSources(common, eventLoop.Handle(), mainReceiver, powerReceiver); err != nil {
		t.Fatal(err)
	}
	var afterRunnable atomic.Int64
	common.AfterRunnable = func() { afterRunnable.Add(1) }
	signal := eventLoop.GetSignal()
	background := common.BackgroundExecutor
	var onMain bool
	common.ForegroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		v, err := background.Spawn(func(s *scheduler.Scope) (int, error) {
			if common.dispatcher.IsMainThread() {
				t.Error("background task ran on the main thread")
			}
			return 42, nil
		}).Await(s)
		if err != nil || v != 42 {
			t.Errorf("background = %d, %v", v, err)
		}
		if _, err := background.Timer(5 * time.Millisecond).Await(s); err != nil {
			t.Error(err)
		}
		onMain = common.dispatcher.IsMainThread()
		signal.Stop()
		return struct{}{}, nil
	}).Detach()
	if err := eventLoop.Run(gpui.Some(time.Second), func() {}); err != nil {
		t.Fatal(err)
	}
	if !onMain {
		t.Fatal("foreground task did not run on the main thread")
	}
	if afterRunnable.Load() == 0 {
		t.Fatal("AfterRunnable did not run")
	}
}

func TestEventLoop_RescheduledPastTimerFiresOncePerDispatch(t *testing.T) {
	eventLoop, err := NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	defer eventLoop.Close()
	fires := 0
	eventLoop.Handle().InsertTimer(time.Now(), func(deadline time.Time) TimeoutAction {
		fires++
		if fires > 100 {
			return TimeoutActionDrop()
		}
		return TimeoutActionToInstant(deadline)
	})
	if err := eventLoop.Dispatch(gpui.Some(time.Duration(0))); err != nil {
		t.Fatal(err)
	}
	if fires != 1 {
		t.Fatalf("timer fired %d times in one dispatch, want 1", fires)
	}
}
