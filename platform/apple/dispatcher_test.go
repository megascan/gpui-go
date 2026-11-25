//go:build darwin

package apple

import (
	"testing"
	"time"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/gcd"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/scheduler"
)

func spawnProbe(t *testing.T, ran chan<- bool) *scheduler.Runnable {
	t.Helper()
	future := scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
		ran <- gcd.IsMainThread()
		return struct{}{}, true
	})
	runnable, task := scheduler.SpawnAsyncTask(scheduler.NewRunnableMetaWithCallersLocation(), future, func(*scheduler.Runnable) {})
	task.Detach()
	return runnable
}

func waitRan(t *testing.T, ran <-chan bool) bool {
	t.Helper()
	select {
	case onMain := <-ran:
		return onMain
	case <-time.After(10 * time.Second):
		t.Fatal("runnable did not run")
		return false
	}
}

func TestDispatcher_DispatchRunsOnGlobalQueues(t *testing.T) {
	d := NewAppleDispatcher()
	var _ gpui.PlatformDispatcher = d
	for _, p := range []gpui.Priority{scheduler.PriorityHigh, scheduler.PriorityMedium, scheduler.PriorityLow} {
		ran := make(chan bool, 1)
		d.Dispatch(spawnProbe(t, ran), p)
		if waitRan(t, ran) {
			t.Errorf("priority %v ran on the main thread", p)
		}
	}
	if runnables.Len() != 0 {
		t.Errorf("%d runnables left in the handle table", runnables.Len())
	}
}

func TestDispatcher_DispatchAfter(t *testing.T) {
	d := NewAppleDispatcher()
	ran := make(chan bool, 1)
	start := time.Now()
	d.DispatchAfter(30*time.Millisecond, spawnProbe(t, ran))
	waitRan(t, ran)
	if time.Since(start) < 20*time.Millisecond {
		t.Fatal("DispatchAfter ran early")
	}
}

func TestDispatcher_RealtimeAudioPanics(t *testing.T) {
	d := NewAppleDispatcher()
	ran := make(chan bool, 1)
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

func TestDispatcher_SpawnRealtime(t *testing.T) {
	d := NewAppleDispatcher()
	done := make(chan bool, 1)
	d.SpawnRealtime(func() { done <- gcd.IsMainThread() })
	select {
	case onMain := <-done:
		if onMain {
			t.Fatal("realtime thread is the main thread")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("realtime function did not run")
	}
}

func TestDispatcher_PreventAppNap(t *testing.T) {
	d := NewAppleDispatcher()
	guard := d.PreventAppNap("gpui dispatcher test")
	guard.Release()
	guard.Release()
}

func TestDispatcher_AppleActivityEndsOnce(t *testing.T) {
	reason := objc.NSStringFromGo("gpui activity test")
	defer objc.Release(reason)
	token := objc.Send[objc.ID](processInfo(), objc.Sel("beginActivityWithOptions:reason:"), uint64(objc.NSActivityUserInitiatedAllowingIdleSystemSleep), reason)
	if token == objc.Nil {
		t.Fatal("beginActivityWithOptions:reason: returned nil")
	}
	activity := &AppleActivity{activity: objc.RetainObject(token)}
	activity.End()
	activity.End()
	if !activity.ended.Load() {
		t.Fatal("End did not mark the activity ended")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("activity token used after End did not panic")
		}
	}()
	_ = activity.activity.ID()
}

func TestDispatcher_MachBindings(t *testing.T) {
	if err := loadMach(); err != nil {
		t.Fatal(err)
	}
	sizes := []struct {
		name  string
		size  uintptr
		count uintptr
	}{
		{"thread_extended_policy_data_t", unsafe.Sizeof(threadExtendedPolicyData{}), threadExtendedPolicyCount},
		{"thread_precedence_policy_data_t", unsafe.Sizeof(threadPrecedencePolicyData{}), threadPrecedencePolicyCount},
		{"thread_time_constraint_policy_data_t", unsafe.Sizeof(threadTimeConstraintPolicyData{}), threadTimeConstraintPolicyCount},
	}
	for _, s := range sizes {
		if s.size != s.count*4 {
			t.Errorf("sizeof(%s) = %d, want %d", s.name, s.size, s.count*4)
		}
	}
	if s := unsafe.Sizeof(machTimebaseInfoData{}); s != 8 {
		t.Errorf("sizeof(mach_timebase_info_data_t) = %d, want 8", s)
	}
	done := make(chan error, 1)
	if err := gcd.SpawnThread(func() { done <- setAudioThreadPriority() }); err != nil {
		t.Fatal(err)
	}
	t.Logf("setAudioThreadPriority on a fresh thread: %v", <-done)
}
