//go:build darwin

package apple

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/apple/gcd"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/scheduler"
)

// AppleDispatcher is the PlatformDispatcher on Grand Central Dispatch.
type AppleDispatcher struct {
	gpui.PlatformDispatcherDefaults
}

// NewAppleDispatcher returns a dispatcher on GCD's main and global queues.
func NewAppleDispatcher() *AppleDispatcher {
	d := &AppleDispatcher{}
	gpui.BindPlatformDispatcherDefaults(&d.PlatformDispatcherDefaults, d)
	return d
}

// IsMainThread reports whether the caller runs on the main thread.
func (d *AppleDispatcher) IsMainThread() bool {
	return objc.IsMainThread()
}

// Dispatch runs runnable on the global queue matching priority.
func (d *AppleDispatcher) Dispatch(runnable gpui.RunnableVariant, priority gpui.Priority) {
	context := runnables.Insert(runnable)

	var queuePriority gcd.QueuePriority
	switch priority {
	case scheduler.PriorityRealtimeAudio:
		runnables.Take(context)
		panic("RealtimeAudio priority should use spawn_realtime, not dispatch")
	case scheduler.PriorityHigh:
		queuePriority = gcd.QueuePriorityHigh
	case scheduler.PriorityMedium:
		queuePriority = gcd.QueuePriorityDefault
	case scheduler.PriorityLow:
		queuePriority = gcd.QueuePriorityLow
	}

	gcd.GlobalQueue(queuePriority).ExecAsyncF(context, trampolineFunction())
}

// DispatchOnMainThread runs runnable on the main queue.
func (d *AppleDispatcher) DispatchOnMainThread(runnable gpui.RunnableVariant, priority gpui.Priority) {
	context := runnables.Insert(runnable)
	gcd.MainQueue().ExecAsyncF(context, trampolineFunction())
}

// DispatchAfter runs runnable on the high-priority global queue after duration.
func (d *AppleDispatcher) DispatchAfter(duration time.Duration, runnable gpui.RunnableVariant) {
	context := runnables.Insert(runnable)
	queue := gcd.GlobalQueue(gcd.QueuePriorityHigh)
	when := gcd.TimeNow.Time(duration.Nanoseconds())
	gcd.ExecAfterF(when, queue, context, trampolineFunction())
}

// SpawnRealtime runs f on a new OS thread with audio-thread scheduling.
func (d *AppleDispatcher) SpawnRealtime(f func()) {
	if err := gcd.SpawnThread(func() {
		gpuiutil.Result(struct{}{}, setAudioThreadPriority()).LogErr()
		f()
	}); err != nil {
		panic(fmt.Sprintf("failed to spawn thread: %v", err))
	}
}

// PreventAppNap keeps the app from being napped until the guard is released.
func (d *AppleDispatcher) PreventAppNap(reason string) *gpui.ActivityGuard {
	return AppleActivityBegin(reason, objc.NSActivityUserInitiatedAllowingIdleSystemSleep)
}

// AppleActivity is an NSProcessInfo activity; End is its drop.
type AppleActivity struct {
	activity *objc.Retained
	ended    atomic.Bool
}

// AppleActivityBegin begins an activity with options and reason, ending it when the guard is released.
func AppleActivityBegin(reason string, options objc.NSActivityOptions) *gpui.ActivityGuard {
	r := objc.NSStringFromGo(reason)
	defer objc.Release(r)
	token := objc.Send[objc.ID](processInfo(), objc.Sel("beginActivityWithOptions:reason:"), uint64(options), r)
	activity := &AppleActivity{activity: objc.RetainObject(token)}
	return gpui.NewActivityGuard(activity.End)
}

// End ends the activity; it runs at most once.
func (a *AppleActivity) End() {
	if a.ended.Swap(true) {
		return
	}
	objc.SendVoid(processInfo(), objc.Sel("endActivity:"), a.activity.ID())
	a.activity.Release()
}

func processInfo() objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSProcessInfo"), objc.Sel("processInfo"))
}

const (
	kernSuccess                     = 0
	threadExtendedPolicy            = 1
	threadTimeConstraintPolicy      = 2
	threadPrecedencePolicy          = 3
	threadExtendedPolicyCount       = 1
	threadTimeConstraintPolicyCount = 4
	threadPrecedencePolicyCount     = 1
)

type threadExtendedPolicyData struct {
	timeshare int32
}

type threadPrecedencePolicyData struct {
	importance int32
}

type threadTimeConstraintPolicyData struct {
	period      uint32
	computation uint32
	constraint  uint32
	preemptible int32
}

type machTimebaseInfoData struct {
	numer uint32
	denom uint32
}

var (
	machOnce           sync.Once
	machErr            error
	thread_policy_set  func(thread uint32, flavor uint32, policy unsafe.Pointer, count uint32) int32
	mach_timebase_info func(info *machTimebaseInfoData) int32
)

func loadMach() error {
	machOnce.Do(func() {
		lib := gcd.LibSystem()
		for _, b := range []struct {
			fptr any
			name string
		}{
			{&thread_policy_set, "thread_policy_set"},
			{&mach_timebase_info, "mach_timebase_info"},
		} {
			addr, err := purego.Dlsym(lib, b.name)
			if err != nil {
				machErr = fmt.Errorf("apple: %w", err)
				return
			}
			purego.RegisterFunc(b.fptr, addr)
		}
	})
	return machErr
}

func setAudioThreadPriority() error {
	if err := loadMach(); err != nil {
		return err
	}

	threadID := gcd.CurrentMachThread()

	policy := threadExtendedPolicyData{timeshare: 0}

	result := thread_policy_set(threadID, threadExtendedPolicy, unsafe.Pointer(&policy), threadExtendedPolicyCount)

	if result != kernSuccess {
		return errors.New("failed to set thread extended policy")
	}

	precedence := threadPrecedencePolicyData{importance: 63}

	result = thread_policy_set(threadID, threadPrecedencePolicy, unsafe.Pointer(&precedence), threadPrecedencePolicyCount)

	if result != kernSuccess {
		return errors.New("failed to set thread precedence policy")
	}

	const guaranteedAudioDutyCycle float32 = 0.75
	const maxAudioDutyCycle float32 = 0.85

	const timeQuantum float32 = 2.9

	const audioTimeNeeded float32 = guaranteedAudioDutyCycle * timeQuantum
	const maxTimeAllowed float32 = maxAudioDutyCycle * timeQuantum

	timebaseInfo := machTimebaseInfoData{numer: 0, denom: 0}
	mach_timebase_info(&timebaseInfo)

	msToAbsTime := (float32(timebaseInfo.denom) / float32(timebaseInfo.numer)) * 1000000

	timeConstraints := threadTimeConstraintPolicyData{
		period:      uint32(timeQuantum * msToAbsTime),
		computation: uint32(audioTimeNeeded * msToAbsTime),
		constraint:  uint32(maxTimeAllowed * msToAbsTime),
		preemptible: 0,
	}

	result = thread_policy_set(threadID, threadTimeConstraintPolicy, unsafe.Pointer(&timeConstraints), threadTimeConstraintPolicyCount)

	if result != kernSuccess {
		return errors.New("failed to set thread time constraint policy")
	}

	return nil
}

var (
	runnables       gcd.Handles[gpui.RunnableVariant]
	trampolineOnce  sync.Once
	trampolineFn    gcd.Function
	profilerThreads sync.Map
)

func trampolineFunction() gcd.Function {
	trampolineOnce.Do(func() {
		trampolineFn = gcd.NewFunction(trampoline)
	})
	return trampolineFn
}

func currentProfilerThread() gpui.ThreadID {
	osThread := gcd.CurrentThreadID()
	if id, ok := profilerThreads.Load(osThread); ok {
		return id.(gpui.ThreadID)
	}
	name := gpui.None[string]()
	if gcd.IsMainThread() {
		name = gpui.Some("main")
	}
	id, _ := profilerThreads.LoadOrStore(osThread, gpui.NewThreadID(name))
	return id.(gpui.ThreadID)
}

func trampoline(context uintptr) {
	runnable, ok := runnables.Take(context)
	if !ok {
		return
	}

	thread := currentProfilerThread()
	location := runnable.Metadata().Location
	spawned := runnable.Metadata().Spawned
	gpui.UpdateRunningTask(thread, spawned, location)
	runnable.Run()
	gpui.SaveTaskTiming(thread)
}
