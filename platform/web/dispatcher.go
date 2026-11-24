//go:build js && wasm

package web

import (
	"fmt"
	"log/slog"
	"math"
	"syscall/js"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

// WebDispatcher is the browser PlatformDispatcher. Go wasm has no threads, so every runnable runs
// on the page's main thread through timers, microtasks and idle callbacks (L-WEBTHREADS).
type WebDispatcher struct {
	gpui.PlatformDispatcherDefaults
	browserWindow   js.Value
	supportsThreads bool
}

var _ gpui.PlatformDispatcher = (*WebDispatcher)(nil)

// NewWebDispatcher returns a dispatcher for browserWindow; allowThreads has no effect because
// the multi-threaded mode does not exist in Go wasm.
func NewWebDispatcher(browserWindow js.Value, allowThreads bool) *WebDispatcher {
	d := &WebDispatcher{
		browserWindow:   browserWindow,
		supportsThreads: false,
	}
	gpui.BindPlatformDispatcherDefaults(&d.PlatformDispatcherDefaults, d)
	return d
}

func (d *WebDispatcher) onMainThread() bool {
	return true
}

// IsMainThread reports true: Go wasm runs every goroutine on the page's main thread.
func (d *WebDispatcher) IsMainThread() bool {
	return d.onMainThread()
}

// Dispatch runs runnable on the main thread.
func (d *WebDispatcher) Dispatch(runnable gpui.RunnableVariant, priority gpui.Priority) {
	if !d.supportsThreads {
		d.DispatchOnMainThread(runnable, priority)
		return
	}
}

// DispatchOnMainThread schedules runnable as a microtask for realtime audio and a zero-delay
// timeout otherwise.
func (d *WebDispatcher) DispatchOnMainThread(runnable gpui.RunnableVariant, priority gpui.Priority) {
	scheduleRunnable(browserWindow(), runnable, priority)
}

// DispatchAfter runs runnable after duration with setTimeout.
func (d *WebDispatcher) DispatchAfter(duration time.Duration, runnable gpui.RunnableVariant) {
	millis := int32(min(duration.Milliseconds(), math.MaxInt32))
	callback := jsFuncOnce(func([]js.Value) {
		runnable.Run()
	})
	if _, err := jsCall(browserWindow(), "setTimeout", callback, millis); err != nil {
		callback.Release()
	}
}

// SpawnRealtime runs function as a microtask.
func (d *WebDispatcher) SpawnRealtime(function func()) {
	callback := jsFuncOnce(func([]js.Value) {
		function()
	})
	if _, err := jsCall(browserWindow(), "queueMicrotask", callback); err != nil {
		callback.Release()
	}
}

// DispatchOnMainThreadWhenIdle runs runnable from requestIdleCallback, or as a low-priority
// task where the browser has no idle callbacks.
func (d *WebDispatcher) DispatchOnMainThreadWhenIdle(runnable gpui.RunnableVariant, timeout gpui.Option[time.Duration]) {
	scheduleIdleRunnable(browserWindow(), runnable, timeout)
}

// IdleTimeRemaining returns the remaining time of the idle callback currently running.
func (d *WebDispatcher) IdleTimeRemaining() (time.Duration, bool) {
	if !d.onMainThread() || jsIsNullish(idleDeadline) {
		return 0, false
	}
	remaining, err := jsCall(idleDeadline, "timeRemaining")
	if err != nil {
		return 0, false
	}
	millis, _ := jsNumber(remaining)
	return time.Duration(millis / 1000.0 * float64(time.Second)), true
}

// Now returns the current instant.
func (d *WebDispatcher) Now() scheduler.Instant {
	return scheduler.InstantNow()
}

var (
	idleDeadline           = js.Undefined()
	idleCallbackSupportedV gpui.Option[bool]
)

func scheduleIdleRunnable(window js.Value, runnable gpui.RunnableVariant, timeout gpui.Option[time.Duration]) {
	if !idleCallbackSupported(window) {
		scheduleRunnable(window, runnable, scheduler.PriorityLow)
		return
	}
	callback := jsFuncOnce(func(args []js.Value) {
		idleDeadline = argOrUndefined(args)
		defer func() { idleDeadline = js.Undefined() }()
		runnable.Run()
	})
	var err error
	if t, ok := timeout.Get(); ok {
		options := js.Global().Get("Object").New()
		options.Set("timeout", uint32(min(t.Milliseconds(), math.MaxUint32)))
		_, err = jsCall(window, "requestIdleCallback", callback, options)
	} else {
		_, err = jsCall(window, "requestIdleCallback", callback)
	}
	if err != nil {
		callback.Release()
		slog.Error(fmt.Sprintf("requestIdleCallback failed: %s", jsErrorDebug(err)))
	}
}

func idleCallbackSupported(window js.Value) bool {
	if supported, ok := idleCallbackSupportedV.Get(); ok {
		return supported
	}
	probed := jsHas(window, "requestIdleCallback")
	idleCallbackSupportedV = gpui.Some(probed)
	return probed
}

func scheduleRunnable(window js.Value, runnable gpui.RunnableVariant, priority gpui.Priority) {
	callback := jsFuncOnce(func([]js.Value) {
		runnable.Run()
	})
	var err error
	switch priority {
	case scheduler.PriorityRealtimeAudio:
		_, err = jsCall(window, "queueMicrotask", callback)
	default:
		_, err = jsCall(window, "setTimeout", callback, 0)
	}
	if err != nil {
		callback.Release()
	}
}
