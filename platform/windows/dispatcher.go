//go:build windows

package windows

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/scheduler"
)

// WindowsDispatcher is the PlatformDispatcher on the Windows thread pool, waking the main thread
// with a message to the platform's message window.
type WindowsDispatcher struct {
	gpui.PlatformDispatcherDefaults
	WakePosted           atomic.Bool
	mainSender           *gpui.PriorityQueueSender[gpui.RunnableVariant]
	mainThreadID         uint32
	PlatformWindowHandle SafeHwnd
	validationNumber     uintptr
}

var _ gpui.PlatformDispatcher = (*WindowsDispatcher)(nil)

// NewWindowsDispatcher creates the dispatcher on the calling (main) thread.
func NewWindowsDispatcher(mainSender *gpui.PriorityQueueSender[gpui.RunnableVariant], platformWindowHandle HWND, validationNumber uintptr) *WindowsDispatcher {
	mainThreadID := xwin.GetCurrentThreadId()
	mainOSThreadID.Store(mainThreadID)

	d := &WindowsDispatcher{
		mainSender:           mainSender,
		mainThreadID:         mainThreadID,
		PlatformWindowHandle: SafeHwndFrom(platformWindowHandle),
		validationNumber:     validationNumber,
	}
	gpui.BindPlatformDispatcherDefaults(&d.PlatformDispatcherDefaults, d)
	return d
}

func (d *WindowsDispatcher) dispatchOnThreadpool(priority int32, runnable gpui.RunnableVariant) {
	environ := win32.TP_CALLBACK_ENVIRON_V3{
		Version:          3,
		CallbackPriority: priority,
	}
	environ.Size = uint32(unsafe.Sizeof(environ))

	context := runnables.Insert(runnable)

	if err := win32.TrySubmitThreadpoolCallback(runWorkCallbackPtr(), context, &environ); err != nil {
		gpuiutil.LogErr(err)
		if runnable, ok := runnables.Take(context); ok {
			go WindowsDispatcherExecuteRunnable(runnable)
		}
	}
}

func (d *WindowsDispatcher) dispatchOnThreadpoolAfter(runnable gpui.RunnableVariant, duration time.Duration) {
	context := runnables.Insert(runnable)

	timer, err := win32.CreateThreadpoolTimer(runTimerCallbackPtr(), context, nil)
	if err != nil {
		gpuiutil.LogErr(err)
		if runnable, ok := runnables.Take(context); ok {
			time.AfterFunc(max(duration, 0), func() {
				WindowsDispatcherExecuteRunnable(runnable)
			})
		}
		return
	}
	ticks := max(duration, 0).Nanoseconds() / 100
	due := uint64(-ticks)
	dueTime := win32.FILETIME{
		LowDateTime:  uint32(due),
		HighDateTime: uint32(due >> 32),
	}
	win32.SetThreadpoolTimer(timer, &dueTime, 0, 0)
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

// Dispatch runs runnable on the thread pool at the matching callback priority.
func (d *WindowsDispatcher) Dispatch(runnable gpui.RunnableVariant, priority gpui.Priority) {
	var tpPriority int32
	switch priority {
	case scheduler.PriorityRealtimeAudio:
		panic("RealtimeAudio priority should use spawn_realtime, not dispatch")
	case scheduler.PriorityHigh:
		tpPriority = win32.TP_CALLBACK_PRIORITY_HIGH
	case scheduler.PriorityMedium:
		tpPriority = win32.TP_CALLBACK_PRIORITY_NORMAL
	case scheduler.PriorityLow:
		tpPriority = win32.TP_CALLBACK_PRIORITY_LOW
	}
	d.dispatchOnThreadpool(tpPriority, runnable)
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

// DispatchAfter runs runnable on the thread pool after duration.
func (d *WindowsDispatcher) DispatchAfter(duration time.Duration, runnable gpui.RunnableVariant) {
	d.dispatchOnThreadpoolAfter(runnable, duration)
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

// Close drops the dispatcher's main-thread sender.
func (d *WindowsDispatcher) Close() {
	d.mainSender.Close()
}

// IncreaseTimerResolution raises the system timer resolution to 1ms until the guard runs.
func (d *WindowsDispatcher) IncreaseTimerResolution() gpui.TimerResolutionGuard {
	xwin.TimeBeginPeriod(1)
	return gpuiutil.Defer(func() {
		xwin.TimeEndPeriod(1)
	})
}

var (
	runnables            win32.Handles[gpui.RunnableVariant]
	realtimeFuncs        win32.Handles[func()]
	callbacksOnce        sync.Once
	runWorkCallbackAddr  uintptr
	runTimerCallbackAddr uintptr
	realtimeStartAddr    uintptr
	mainOSThreadID       atomic.Uint32
	profilerThreads      sync.Map
)

func initCallbacks() {
	callbacksOnce.Do(func() {
		runWorkCallbackAddr = xwin.NewCallback(runWorkCallbackFn)
		runTimerCallbackAddr = xwin.NewCallback(runTimerCallbackFn)
		realtimeStartAddr = xwin.NewCallback(realtimeThreadStartFn)
	})
}

func runWorkCallbackPtr() uintptr {
	initCallbacks()
	return runWorkCallbackAddr
}

func runTimerCallbackPtr() uintptr {
	initCallbacks()
	return runTimerCallbackAddr
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

func runWorkCallbackFn(_ uintptr, context uintptr) uintptr {
	if runnable, ok := runnables.Take(context); ok {
		WindowsDispatcherExecuteRunnable(runnable)
	}
	return 0
}

func runTimerCallbackFn(_ uintptr, context uintptr, timer win32.PTP_TIMER) uintptr {
	if runnable, ok := runnables.Take(context); ok {
		WindowsDispatcherExecuteRunnable(runnable)
	}
	win32.CloseThreadpoolTimer(timer)
	return 0
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
