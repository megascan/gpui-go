package gpui

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func TestThreadedDispatcher_IsIdleTracksQueuedWorkButIgnoresUndueTimers(t *testing.T) {
	dispatcher := NewThreadedDispatcher()
	foreground := NewForegroundExecutor(dispatcher)
	defer runtime.KeepAlive(foreground)
	if !dispatcher.isIdle() {
		t.Fatal("new dispatcher must be idle")
	}

	foreground.Spawn(func(*scheduler.Scope) (struct{}, error) { return struct{}{}, nil }).Detach()
	if dispatcher.isIdle() {
		t.Fatal("queued main task must count as work")
	}
	dispatcher.RunUntilIdle()
	if !dispatcher.isIdle() {
		t.Fatal("dispatcher must be idle after RunUntilIdle")
	}

	background := NewBackgroundExecutor(dispatcher)
	defer runtime.KeepAlive(background)
	timer := background.Timer(60 * time.Second)
	dispatcher.RunUntilIdle()
	if !dispatcher.isIdle() {
		t.Fatal("a timer that is not due yet should not count as pending work")
	}
	timer.Cancel()
	dispatcher.CancelPendingTimers()
	dispatcher.RunUntilIdle()
	if !dispatcher.isIdle() {
		t.Fatal("dispatcher must be idle after canceling timers")
	}
}

func spawnBackgroundToMainHandoff(dispatcher *ThreadedDispatcher) (*atomic.Bool, BackgroundExecutor, ForegroundExecutor) {
	background := NewBackgroundExecutor(dispatcher)
	defer runtime.KeepAlive(background)
	foreground := NewForegroundExecutor(dispatcher)
	defer runtime.KeepAlive(foreground)

	sender, receiver := scheduler.Oneshot[struct{}]()
	background.Spawn(func(*scheduler.Scope) (struct{}, error) {
		time.Sleep(10 * time.Millisecond)
		_ = sender.Send(struct{}{})
		return struct{}{}, nil
	}).Detach()

	completed := new(atomic.Bool)
	foreground.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		_, _ = receiver.Recv(s)
		completed.Store(true)
		return struct{}{}, nil
	}).Detach()
	return completed, background, foreground
}

func TestThreadedDispatcher_RunReadyMainTasksDoesNotWaitForBackgroundHandoffs(t *testing.T) {
	dispatcher := NewThreadedDispatcher()
	completed, background, foreground := spawnBackgroundToMainHandoff(dispatcher)
	defer runtime.KeepAlive(background)
	defer runtime.KeepAlive(foreground)

	if !dispatcher.RunReadyMainTasks() {
		t.Fatal("RunReadyMainTasks must run the queued task")
	}
	if completed.Load() {
		t.Fatal("RunReadyMainTasks must not wait for the background handoff")
	}

	dispatcher.RunUntilIdle()
	if !completed.Load() {
		t.Fatal("RunUntilIdle must complete the handoff")
	}
}

func TestThreadedDispatcher_RunUntilIdleCompletesBackgroundToMainHandoffs(t *testing.T) {
	dispatcher := NewThreadedDispatcher()
	completed, background, foreground := spawnBackgroundToMainHandoff(dispatcher)
	defer runtime.KeepAlive(background)
	defer runtime.KeepAlive(foreground)

	dispatcher.RunUntilIdle()
	if !completed.Load() {
		t.Fatal("RunUntilIdle must complete the handoff")
	}
}

func TestThreadedDispatcher_RunUntilWaitsForUntrackedExternalWakes(t *testing.T) {
	dispatcher := NewThreadedDispatcher()
	foreground := NewForegroundExecutor(dispatcher)
	defer runtime.KeepAlive(foreground)
	sender, receiver := scheduler.Oneshot[struct{}]()
	senderDone := make(chan struct{})
	go func() {
		defer close(senderDone)
		time.Sleep(10 * time.Millisecond)
		if err := sender.Send(struct{}{}); err != nil {
			t.Error("foreground receiver should remain alive")
		}
	}()

	completed := new(atomic.Bool)
	foreground.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		if _, err := receiver.Recv(s); err != nil {
			t.Error("external sender should deliver its wake")
		}
		completed.Store(true)
		return struct{}{}, nil
	}).Detach()

	dispatcher.runUntil(completed.Load)
	<-senderDone
	if !completed.Load() {
		t.Fatal("task did not complete")
	}
}

func yieldOnce() scheduler.Future[struct{}] {
	yielded := false
	return scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
		if yielded {
			return struct{}{}, true
		}
		yielded = true
		cx.Waker().WakeByRef()
		return struct{}{}, false
	})
}

func spawnRequeuingWork(foreground ForegroundExecutor, limit int) *atomic.Int64 {
	iterations := new(atomic.Int64)
	foreground.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		for range limit {
			iterations.Add(1)
			scheduler.Await(s, yieldOnce())
		}
		return struct{}{}, nil
	}).Detach()
	return iterations
}

func TestThreadedDispatcher_RunUntilReturnsAtReadinessDespiteRequeuingMainWork(t *testing.T) {
	const requeueLimit = 10_000

	dispatcher := NewThreadedDispatcher()
	foreground := NewForegroundExecutor(dispatcher)
	defer runtime.KeepAlive(foreground)

	iterations := spawnRequeuingWork(foreground, requeueLimit)

	completed := new(atomic.Bool)
	foreground.Spawn(func(*scheduler.Scope) (struct{}, error) {
		completed.Store(true)
		return struct{}{}, nil
	}).Detach()

	dispatcher.runUntil(completed.Load)
	if iterations.Load() >= requeueLimit {
		t.Fatal("run_until should return at readiness instead of draining re-queued main work")
	}
}

func TestThreadedDispatcher_RunReadyMainTasksAdvancesRequeuingWorkOneBatchPerCall(t *testing.T) {
	const requeueLimit = 10_000

	dispatcher := NewThreadedDispatcher()
	foreground := NewForegroundExecutor(dispatcher)
	defer runtime.KeepAlive(foreground)

	iterations := spawnRequeuingWork(foreground, requeueLimit)

	if !dispatcher.RunReadyMainTasks() {
		t.Fatal("first batch must run")
	}
	if n := iterations.Load(); n != 1 {
		t.Fatalf("iterations %d, want 1", n)
	}
	if !dispatcher.RunReadyMainTasks() {
		t.Fatal("second batch must run")
	}
	if n := iterations.Load(); n != 2 {
		t.Fatalf("iterations %d, want 2", n)
	}
}

func TestThreadedDispatcher_TimersFireInRealTime(t *testing.T) {
	dispatcher := NewThreadedDispatcher()
	background := NewBackgroundExecutor(dispatcher)
	defer runtime.KeepAlive(background)

	fired := new(atomic.Bool)
	timer := background.Timer(10 * time.Millisecond)
	background.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		_, _ = timer.Await(s)
		fired.Store(true)
		return struct{}{}, nil
	}).Detach()

	deadline := time.Now().Add(10 * time.Second)
	for !fired.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !fired.Load() {
		t.Fatal("timer did not fire")
	}
}

func TestThreadedDispatcher_CancelPendingTimersWakesWaitersWithoutWaitingForDeadline(t *testing.T) {
	dispatcher := NewThreadedDispatcher()
	background := NewBackgroundExecutor(dispatcher)
	defer runtime.KeepAlive(background)

	fired := new(atomic.Bool)
	timer := background.Timer(10 * time.Second)
	background.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		_, _ = timer.Await(s)
		fired.Store(true)
		return struct{}{}, nil
	}).Detach()

	dispatcher.RunUntilIdle()
	if n := dispatcher.CancelPendingTimers(); n != 1 {
		t.Fatalf("canceled %d, want 1", n)
	}
	dispatcher.RunUntilIdle()

	if !fired.Load() {
		t.Fatal("canceling the timer must wake its waiter")
	}
	if n := dispatcher.CancelPendingTimers(); n != 0 {
		t.Fatalf("canceled %d, want 0", n)
	}
}

func TestThreadedDispatcher_DebugStateDescribesIdleState(t *testing.T) {
	dispatcher := NewThreadedDispatcher()
	want := "ThreadedDispatcher { inflight: 0, pending_timers: 0, main_queue_has_work: false }"
	if got := dispatcher.DebugState(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
