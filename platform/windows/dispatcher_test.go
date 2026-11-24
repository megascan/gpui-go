//go:build windows

package windows

import (
	"testing"
	"time"
	"unsafe"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/scheduler"
)

func spawnProbe(t *testing.T, ran chan<- uint32) *scheduler.Runnable {
	t.Helper()
	future := scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
		ran <- xwin.GetCurrentThreadId()
		return struct{}{}, true
	})
	runnable, task := scheduler.SpawnAsyncTask(scheduler.NewRunnableMetaWithCallersLocation(), future, func(*scheduler.Runnable) {})
	task.Detach()
	return runnable
}

func waitRan(t *testing.T, ran <-chan uint32) uint32 {
	t.Helper()
	select {
	case thread := <-ran:
		return thread
	case <-time.After(10 * time.Second):
		t.Fatal("runnable did not run")
		return 0
	}
}

func newTestDispatcher(t *testing.T) (*WindowsDispatcher, *gpui.PriorityQueueReceiver[gpui.RunnableVariant]) {
	t.Helper()
	sender, receiver := gpui.NewPriorityQueueReceiver[gpui.RunnableVariant]()
	d := NewWindowsDispatcher(sender, 0, 1)
	t.Cleanup(func() {
		d.Close()
		receiver.Close()
	})
	return d, receiver
}

func TestDispatcher_DispatchRunsOnThreadPool(t *testing.T) {
	d, _ := newTestDispatcher(t)
	var _ gpui.PlatformDispatcher = d
	for _, p := range []gpui.Priority{scheduler.PriorityHigh, scheduler.PriorityMedium, scheduler.PriorityLow} {
		ran := make(chan uint32, 1)
		d.Dispatch(spawnProbe(t, ran), p)
		waitRan(t, ran)
	}
	if runnables.Len() != 0 {
		t.Errorf("%d runnables left in the handle table", runnables.Len())
	}
}

func TestDispatcher_DispatchAfter(t *testing.T) {
	d, _ := newTestDispatcher(t)
	ran := make(chan uint32, 1)
	start := time.Now()
	d.DispatchAfter(30*time.Millisecond, spawnProbe(t, ran))
	waitRan(t, ran)
	if time.Since(start) < 20*time.Millisecond {
		t.Fatal("DispatchAfter ran early")
	}
}

func TestDispatcher_RealtimeAudioPanics(t *testing.T) {
	d, _ := newTestDispatcher(t)
	ran := make(chan uint32, 1)
	r := spawnProbe(t, ran)
	defer r.Close()
	defer func() {
		if got := recover(); got != "RealtimeAudio priority should use spawn_realtime, not dispatch" {
			t.Fatalf("recover() = %v", got)
		}
		if runnables.Len() != 0 {
			t.Fatal("panicking dispatch left its runnable in the handle table")
		}
	}()
	d.Dispatch(r, scheduler.PriorityRealtimeAudio)
}

func TestDispatcher_DispatchOnMainThreadPostsOneWake(t *testing.T) {
	d, receiver := newTestDispatcher(t)
	first := make(chan uint32, 1)
	second := make(chan uint32, 1)
	d.DispatchOnMainThread(spawnProbe(t, first), scheduler.PriorityMedium)
	if !d.WakePosted.Load() {
		t.Fatal("first main-thread dispatch did not post a wake-up")
	}
	d.DispatchOnMainThread(spawnProbe(t, second), scheduler.PriorityMedium)
	count := 0
	for {
		runnable, ok, err := receiver.TryPop()
		if err != nil || !ok {
			break
		}
		WindowsDispatcherExecuteRunnable(runnable)
		count++
	}
	if count != 2 {
		t.Fatalf("main queue held %d runnables, want 2", count)
	}
	waitRan(t, first)
	waitRan(t, second)
}

func TestDispatcher_DispatchOnMainThreadAfterReceiverClosed(t *testing.T) {
	sender, receiver := gpui.NewPriorityQueueReceiver[gpui.RunnableVariant]()
	d := NewWindowsDispatcher(sender, 0, 1)
	defer d.Close()
	receiver.Close()
	ran := make(chan uint32, 1)
	d.DispatchOnMainThread(spawnProbe(t, ran), scheduler.PriorityMedium)
	if d.WakePosted.Load() {
		t.Fatal("a failed send posted a wake-up")
	}
}

func TestDispatcher_CloseDisconnectsMainQueue(t *testing.T) {
	sender, receiver := gpui.NewPriorityQueueReceiver[gpui.RunnableVariant]()
	defer receiver.Close()
	d := NewWindowsDispatcher(sender, 0, 1)
	d.Close()
	d.Close()
	if _, ok, err := receiver.TryPop(); ok || err == nil {
		t.Fatalf("TryPop after Close = %v, %v; want a disconnect", ok, err)
	}
}

func TestDispatcher_SpawnRealtime(t *testing.T) {
	d, _ := newTestDispatcher(t)
	done := make(chan uint32, 1)
	caller := xwin.GetCurrentThreadId()
	d.SpawnRealtime(func() { done <- xwin.GetCurrentThreadId() })
	select {
	case thread := <-done:
		if thread == caller {
			t.Fatal("realtime function ran on the calling thread")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("realtime function did not run")
	}
}

func TestDispatcher_IncreaseTimerResolution(t *testing.T) {
	d, _ := newTestDispatcher(t)
	guard := d.IncreaseTimerResolution()
	guard.Run()
}

func TestDispatcher_CallbackEnvironmentLayout(t *testing.T) {
	var environ win32.TP_CALLBACK_ENVIRON_V3
	if unsafe.Sizeof(environ) != 72 || unsafe.Offsetof(environ.CallbackPriority) != 60 {
		t.Fatalf("TP_CALLBACK_ENVIRON_V3 size %d, CallbackPriority at %d", unsafe.Sizeof(environ), unsafe.Offsetof(environ.CallbackPriority))
	}
}
