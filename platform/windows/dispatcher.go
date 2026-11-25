//go:build windows

package windows

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/scheduler"
)

// WindowsDispatcher is the PlatformDispatcher: background runnables run on Go worker goroutines,
// main-thread runnables wake the main thread with a message to the platform's message window.
// Upstream uses the Windows thread pool; Go callbacks from pool threads are internally locked to
// foreign OS threads, and an iter.Pull coroutine resumed on another pool thread is a fatal error.
type WindowsDispatcher struct {
	gpui.PlatformDispatcherDefaults
	WakePosted           atomic.Bool
	mainSender           *gpui.PriorityQueueSender[gpui.RunnableVariant]
	backgroundSender     *gpui.PriorityQueueSender[gpui.RunnableVariant]
	closed               atomic.Bool
	mainThreadID         uint32
	PlatformWindowHandle SafeHwnd
	validationNumber     uintptr
}

var _ gpui.PlatformDispatcher = (*WindowsDispatcher)(nil)

// NewWindowsDispatcher creates the dispatcher on the calling (main) thread.
func NewWindowsDispatcher(mainSender *gpui.PriorityQueueSender[gpui.RunnableVariant], platformWindowHandle HWND, validationNumber uintptr) *WindowsDispatcher {
	mainThreadID := xwin.GetCurrentThreadId()
	mainOSThreadID.Store(mainThreadID)

	backgroundSender, backgroundReceiver := gpui.NewPriorityQueueReceiver[gpui.RunnableVariant]()
	for range max(runtime.NumCPU(), minWorkerThreads) {
		receiver := backgroundReceiver.Clone()
		go func() {
			for runnable := range receiver.Iter() {
				WindowsDispatcherExecuteRunnable(runnable)
			}
		}()
	}
	backgroundReceiver.Close()

	d := &WindowsDispatcher{
		mainSender:           mainSender,
		backgroundSender:     backgroundSender,
		mainThreadID:         mainThreadID,
		PlatformWindowHandle: SafeHwndFrom(platformWindowHandle),
		validationNumber:     validationNumber,
	}
	gpui.BindPlatformDispatcherDefaults(&d.PlatformDispatcherDefaults, d)
	return d
}

// WindowsDispatcherExecuteRunnable runs a runnable with the profiler hooks.
func WindowsDispatcherExecuteRunnable(runnable gpui.RunnableVariant) {
	thread := currentProfilerThread()
	location := runnable.Metadata().Location
	spawned := runnable.Metadata().Spawned
	gpui.UpdateRunningTask(thread, spawned, location)
	runnable.Run()
	gpui.SaveTaskTiming(thread)
}

// IsMainThread reports whether the caller runs on the thread that created the dispatcher.
func (d *WindowsDispatcher) IsMainThread() bool {
	return xwin.GetCurrentThreadId() == d.mainThreadID
}

// Dispatch runs runnable on a worker goroutine in priority order.
func (d *WindowsDispatcher) Dispatch(runnable gpui.RunnableVariant, priority gpui.Priority) {
	if priority == scheduler.PriorityRealtimeAudio {
		panic("RealtimeAudio priority should use spawn_realtime, not dispatch")
	}
	if err := d.backgroundSender.Send(priority, runnable); err != nil && !d.closed.Load() {
		gpuiutil.LogErr(err)
	}
}

// DispatchOnMainThread queues runnable for the main thread and posts one wake-up message until
// the main thread drains the queue.
func (d *WindowsDispatcher) DispatchOnMainThread(runnable gpui.RunnableVariant, priority gpui.Priority) {
	if err := d.mainSender.Send(priority, runnable); err != nil {
		return
	}
	if !d.WakePosted.Swap(true) {
		if err := win32.PostMessageW(
			d.PlatformWindowHandle.AsRaw(),
			WM_GPUI_TASK_DISPATCHED_ON_MAIN_THREAD,
			win32.WPARAM(d.validationNumber),
			win32.LPARAM(0),
		); err != nil {
			d.WakePosted.Store(false)
			gpuiutil.LogErr(err)
		}
	}
}

// DispatchAfter runs runnable on a timer goroutine after duration.
func (d *WindowsDispatcher) DispatchAfter(duration time.Duration, runnable gpui.RunnableVariant) {
	time.AfterFunc(max(duration, 0), func() {
		WindowsDispatcherExecuteRunnable(runnable)
	})
}

// SpawnRealtime runs f on a new native thread at THREAD_PRIORITY_TIME_CRITICAL.
func (d *WindowsDispatcher) SpawnRealtime(f func()) {
	key := realtimeFuncs.Insert(f)
	handle, _, err := win32.CreateThread(0, realtimeThreadStartPtr(), key, 0)
	if err != nil {
		realtimeFuncs.Take(key)
		panic(fmt.Sprintf("failed to spawn thread: %v", err))
	}
	xwin.CloseHandle(handle)
}

// Close drops the dispatcher's senders, which ends the worker goroutines.
func (d *WindowsDispatcher) Close() {
	if d.closed.Swap(true) {
		return
	}
	d.backgroundSender.Close()
	d.mainSender.Close()
}

// IncreaseTimerResolution raises the system timer resolution to 1ms until the guard runs.
func (d *WindowsDispatcher) IncreaseTimerResolution() gpui.TimerResolutionGuard {
	xwin.TimeBeginPeriod(1)
	return gpuiutil.Defer(func() {
		xwin.TimeEndPeriod(1)
	})
}

const minWorkerThreads = 2

var (
	realtimeFuncs     win32.Handles[func()]
	callbacksOnce     sync.Once
	realtimeStartAddr uintptr
	mainOSThreadID    atomic.Uint32
	profilerThreads   sync.Map
)

func initCallbacks() {
	callbacksOnce.Do(func() {
		realtimeStartAddr = xwin.NewCallback(realtimeThreadStartFn)
	})
}

func realtimeThreadStartPtr() uintptr {
	initCallbacks()
	return realtimeStartAddr
}

func currentProfilerThread() gpui.ThreadID {
	osThread := xwin.GetCurrentThreadId()
	if id, ok := profilerThreads.Load(osThread); ok {
		return id.(gpui.ThreadID)
	}
	name := gpui.None[string]()
	if osThread == mainOSThreadID.Load() {
		name = gpui.Some("main")
	}
	id, _ := profilerThreads.LoadOrStore(osThread, gpui.NewThreadID(name))
	return id.(gpui.ThreadID)
}

func realtimeThreadStartFn(parameter uintptr) uintptr {
	f, ok := realtimeFuncs.Take(parameter)
	if !ok {
		return 0
	}
	threadHandle := win32.GetCurrentThread()

	if err := win32.SetThreadPriority(threadHandle, win32.THREAD_PRIORITY_TIME_CRITICAL); err != nil {
		gpuiutil.LogErr(fmt.Errorf("thread priority: %w", err))
	}

	f()
	return 0
}
