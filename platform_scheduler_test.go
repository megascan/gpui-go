package gpui

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

type smokeDispatcher struct {
	PlatformDispatcherDefaults
}

func newSmokeDispatcher() *smokeDispatcher {
	d := &smokeDispatcher{}
	BindPlatformDispatcherDefaults(&d.PlatformDispatcherDefaults, d)
	return d
}

func (*smokeDispatcher) IsMainThread() bool { return false }

func (*smokeDispatcher) Dispatch(RunnableVariant, Priority) {
	panic("SmokeDispatcher should not be asked to dispatch in this test")
}

func (*smokeDispatcher) DispatchOnMainThread(RunnableVariant, Priority) {
	panic("SmokeDispatcher does not implement a main thread")
}

func (*smokeDispatcher) DispatchAfter(time.Duration, RunnableVariant) {
	panic("SmokeDispatcher does not implement timers")
}

func (*smokeDispatcher) SpawnRealtime(func()) {
	panic("SmokeDispatcher does not implement realtime")
}

func newSmokeBackground() scheduler.BackgroundExecutor {
	return scheduler.NewBackgroundExecutor(NewPlatformScheduler(newSmokeDispatcher()))
}

func blockOnTask[T any](t *testing.T, task *Task[T]) T {
	t.Helper()
	r := scheduler.BlockOn[scheduler.Result[T]](task)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	return r.Value
}

func TestPlatformScheduler_DedicatedExecutorTasksShareOneThread(t *testing.T) {
	background := newSmokeBackground()
	dedicated := scheduler.NewDedicatedExecutor(background)
	defer dedicated.Close()

	first := dedicated.Spawn(func(*scheduler.Scope) (int64, error) { return currentThreadGoroutineID(), nil })
	second := dedicated.Spawn(func(*scheduler.Scope) (int64, error) { return currentThreadGoroutineID(), nil })

	firstID := blockOnTask(t, first)
	secondID := blockOnTask(t, second)

	if firstID != secondID {
		t.Fatal("tasks must share the dedicated thread")
	}
	if firstID == currentThreadGoroutineID() {
		t.Fatal("dedicated tasks must not run on the spawning thread")
	}
}

func TestPlatformScheduler_SpawnDedicatedRunsOnARealSeparateThread(t *testing.T) {
	background := newSmokeBackground()
	started := time.Now()
	type result struct {
		thread  int64
		message string
	}
	task := background.SpawnDedicated(func(scheduler.LocalExecutor, *scheduler.Scope) (result, error) {
		before := currentThreadGoroutineID()
		time.Sleep(50 * time.Millisecond)
		after := currentThreadGoroutineID()
		if before != after {
			t.Error("thread changed across the sleep")
		}
		return result{before, "slept"}, nil
	})
	r := blockOnTask(t, task)
	elapsed := time.Since(started)
	if r.message != "slept" {
		t.Fatalf("message %q", r.message)
	}
	if r.thread == currentThreadGoroutineID() {
		t.Fatal("dedicated future ran on the test thread")
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("expected the dedicated thread to genuinely sleep, elapsed = %v", elapsed)
	}
}

func TestPlatformScheduler_SpawnDedicatedReturnsNotSendFutureOutput(t *testing.T) {
	background := newSmokeBackground()
	task := background.SpawnDedicated(func(scheduler.LocalExecutor, *scheduler.Scope) (int32, error) {
		state := new(int32)
		for range 3 {
			*state++
		}
		return *state, nil
	})
	if output := blockOnTask(t, task); output != 3 {
		t.Fatalf("got %d, want 3", output)
	}
}

func TestPlatformScheduler_SpawnDedicatedDroppingTaskCancelsFuture(t *testing.T) {
	background := newSmokeBackground()

	startedTx := make(chan struct{}, 1)
	afterParkTx := make(chan struct{}, 1)
	var observedPostAwaitWrite atomic.Bool

	task := background.SpawnDedicated(func(_ scheduler.LocalExecutor, s *scheduler.Scope) (struct{}, error) {
		startedTx <- struct{}{}
		scheduler.Await[struct{}](s, scheduler.PollFn(func(*scheduler.Context) (struct{}, bool) {
			return struct{}{}, false
		}))
		observedPostAwaitWrite.Store(true)
		afterParkTx <- struct{}{}
		return struct{}{}, nil
	})

	select {
	case <-startedTx:
	case <-time.After(2 * time.Second):
		t.Fatal("dedicated future failed to start")
	}

	task.Cancel()

	select {
	case <-afterParkTx:
		t.Fatal("dedicated future advanced past the await after its Task was dropped")
	case <-time.After(100 * time.Millisecond):
	}
	if observedPostAwaitWrite.Load() {
		t.Fatal("dedicated future ran code past the cancellation point")
	}
}

func TestPlatformScheduler_SpawnDedicatedThreadTearsDownAfterWorkCompletes(t *testing.T) {
	background := newSmokeBackground()
	startedTx := make(chan int64, 1)
	dropTx := make(chan int64, 1)

	task := background.SpawnDedicated(func(scheduler.LocalExecutor, *scheduler.Scope) (struct{}, error) {
		defer func() { dropTx <- currentThreadGoroutineID() }()
		startedTx <- currentThreadGoroutineID()
		return struct{}{}, nil
	})

	var dedicatedThreadID int64
	select {
	case dedicatedThreadID = <-startedTx:
	case <-time.After(2 * time.Second):
		t.Fatal("dedicated future failed to start")
	}
	if dedicatedThreadID == currentThreadGoroutineID() {
		t.Fatal("dedicated future ran on the test thread")
	}

	blockOnTask(t, task)

	select {
	case dropThreadID := <-dropTx:
		if dropThreadID != dedicatedThreadID {
			t.Fatal("dedicated future's captured state must be dropped on the dedicated thread, not elsewhere")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dedicated future's captured state was not dropped after task completion")
	}
}

func TestPlatformScheduler_SpawnDedicatedDetachedChildOutlivesRoot(t *testing.T) {
	background := newSmokeBackground()

	gate := make(chan struct{})
	childDone := make(chan int64, 1)

	task := background.SpawnDedicated(func(executor scheduler.LocalExecutor, _ *scheduler.Scope) (struct{}, error) {
		executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
			<-gate
			childDone <- currentThreadGoroutineID()
			return struct{}{}, nil
		}).Detach()
		return struct{}{}, nil
	})

	blockOnTask(t, task)

	select {
	case <-childDone:
		t.Fatal("detached child finished before being released")
	case <-time.After(50 * time.Millisecond):
	}

	close(gate)

	select {
	case childThreadID := <-childDone:
		if childThreadID == currentThreadGoroutineID() {
			t.Fatal("detached child ran on the test thread instead of the dedicated thread")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("detached child failed to complete after gate was released")
	}
}
