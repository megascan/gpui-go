//go:build windows

package windows

import (
	"runtime"
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

func TestDispatcher_DispatchRunsOnWorkers(t *testing.T) {
	d, _ := newTestDispatcher(t)
	var _ gpui.PlatformDispatcher = d
	for _, p := range []gpui.Priority{scheduler.PriorityHigh, scheduler.PriorityMedium, scheduler.PriorityLow} {
		ran := make(chan uint32, 1)
		d.Dispatch(spawnProbe(t, ran), p)
		waitRan(t, ran)
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

func TestDispatcher_CoroutineResumesAcrossWorkers(t *testing.T) {
	d, _ := newTestDispatcher(t)
	const hops = 64
	done := make(chan int, 1)
	future := scheduler.Async(func(s *scheduler.Scope) int {
		n := 0
		for range hops {
			yielded := false
			scheduler.Await(s, scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
				if yielded {
					return struct{}{}, true
				}
				yielded = true
				cx.Waker().WakeByRef()
				return struct{}{}, false
			}))
			n++
		}
		done <- n
		return n
	})
	runnable, task := scheduler.SpawnAsyncTask(scheduler.NewRunnableMetaWithCallersLocation(), future, func(r *scheduler.Runnable) {
		d.Dispatch(r, scheduler.PriorityMedium)
	})
	task.Detach()
	d.Dispatch(runnable, scheduler.PriorityMedium)
	select {
	case n := <-done:
		if n != hops {
			t.Fatalf("body ran %d hops, want %d", n, hops)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("coroutine did not finish")
	}
}

func TestDispatcher_ForegroundDeferredInNestedModalLoop(t *testing.T) {
	failure := make(chan string, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		failure <- nestedModalLoopRoundTrip()
	}()
	select {
	case message := <-failure:
		if message != "" {
			t.Fatal(message)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("nested modal loop round trip timed out")
	}
}

const testModalLoopMsg = win32.WM_USER + 0x200

func nestedModalLoopRoundTrip() string {
	var inner *windowsPlatformInner
	var resumeWaker scheduler.Waker
	released := false
	finished := false
	modalFailure := ""

	nestedModalLoop := func() {
		released = true
		resumeWaker.Wake()
		deadline := time.Now().Add(300 * time.Millisecond)
		var msg win32.MSG
		for time.Now().Before(deadline) {
			for win32.PeekMessageW(&msg, 0, 0, 0, win32.PM_REMOVE) {
				win32.TranslateMessage(&msg)
				win32.DispatchMessageW(&msg)
			}
			time.Sleep(time.Millisecond)
		}
		if finished {
			modalFailure = "foreground task resumed inside the nested modal loop"
		}
		if !inner.dispatcher.ForegroundDeferred.Load() {
			modalFailure = "wake-up inside the nested modal loop was not deferred"
		}
	}

	wndProc := xwin.NewCallback(func(hwnd, msg, wparam, lparam uintptr) uintptr {
		if inner == nil {
			return uintptr(win32.DefWindowProcW(HWND(hwnd), uint32(msg), win32.WPARAM(wparam), win32.LPARAM(lparam)))
		}
		if uint32(msg) == testModalLoopMsg {
			nestedModalLoop()
			return 0
		}
		return uintptr(inner.handleMsg(HWND(hwnd), uint32(msg), win32.WPARAM(wparam), win32.LPARAM(lparam)))
	})

	className := win32.UTF16PtrFromString("GpuiNestedModalLoopTestClass")
	instance, err := win32.GetModuleHandleW(nil)
	if err != nil {
		return err.Error()
	}
	if _, err := win32.RegisterClassW(&win32.WNDCLASSW{LpfnWndProc: wndProc, HInstance: instance, LpszClassName: className}); err != nil {
		return err.Error()
	}
	defer win32.UnregisterClassW(className, instance)
	hwnd, err := win32.CreateWindowExW(0, className, nil, 0, 0, 0, 0, 0, HWND(win32.HWND_MESSAGE), 0, instance, 0)
	if err != nil {
		return err.Error()
	}
	defer win32.DestroyWindow(hwnd)

	sender, receiver := gpui.NewPriorityQueueReceiver[gpui.RunnableVariant]()
	dispatcher := NewWindowsDispatcher(sender, hwnd, 1)
	defer func() {
		dispatcher.Close()
		receiver.Close()
	}()
	inner = &windowsPlatformInner{validationNumber: 1, mainReceiver: receiver, dispatcher: dispatcher}

	future := scheduler.Async(func(s *scheduler.Scope) int {
		scheduler.Await(s, scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
			if released {
				return struct{}{}, true
			}
			resumeWaker = cx.Waker().Clone()
			return struct{}{}, false
		}))
		finished = true
		return 0
	})
	runnable, task := scheduler.SpawnAsyncTask(scheduler.NewRunnableMetaWithCallersLocation(), future, func(r *scheduler.Runnable) {
		dispatcher.DispatchOnMainThread(r, scheduler.PriorityMedium)
	})
	task.Detach()
	dispatcher.DispatchOnMainThread(runnable, scheduler.PriorityMedium)
	if err := win32.PostMessageW(hwnd, testModalLoopMsg, 0, 0); err != nil {
		return err.Error()
	}

	deadline := time.Now().Add(20 * time.Second)
	var msg win32.MSG
	for !finished && time.Now().Before(deadline) {
		if win32.GetMessageW(&msg, 0, 0, 0) <= 0 {
			return "message loop ended early"
		}
		win32.TranslateMessage(&msg)
		win32.DispatchMessageW(&msg)
		dispatcher.RepostDeferredWake()
	}
	if modalFailure != "" {
		return modalFailure
	}
	if !finished {
		return "foreground task did not finish after the nested modal loop returned"
	}
	if dispatcher.ForegroundDeferred.Load() {
		return "deferred wake-up was not reposted"
	}
	return ""
}
