package scheduler

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func goroutineBaseline(t *testing.T) func() {
	t.Helper()
	before := runtime.NumGoroutine()
	return func() {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for {
			n := runtime.NumGoroutine()
			if n <= before {
				return
			}
			if time.Now().After(deadline) {
				buf := make([]byte, 1<<20)
				buf = buf[:runtime.Stack(buf, true)]
				t.Fatalf("goroutine leak: %d before, %d after\n%s", before, n, buf)
			}
			runtime.Gosched()
			time.Sleep(time.Millisecond)
		}
	}
}

func TestExecutor_ForegroundSpawnAwait(t *testing.T) {
	s := newFIFOScheduler()
	task := s.foreground().Spawn(func(*Scope) (int, error) { return 42, nil })
	v, err := awaitTask(t, s, task)
	if v != 42 || err != nil {
		t.Fatalf("got %d, %v", v, err)
	}
}

func TestExecutor_BackgroundSpawnError(t *testing.T) {
	s := newFIFOScheduler()
	want := errors.New("boom")
	task := s.background().Spawn(func(*Scope) (string, error) { return "partial", want })
	v, err := awaitTask(t, s, task)
	if v != "partial" || !errors.Is(err, want) {
		t.Fatalf("got %q, %v", v, err)
	}
}

func TestExecutor_TaskAwaitsTask(t *testing.T) {
	s := newFIFOScheduler()
	bg := s.background()
	fg := s.foreground()
	tx, rx := Oneshot[int]()
	bg.Spawn(func(*Scope) (struct{}, error) {
		return struct{}{}, tx.Send(7)
	}).Detach()
	outer := fg.Spawn(func(cx *Scope) (int, error) {
		inner := bg.Spawn(func(cx *Scope) (int, error) {
			v, err := rx.Recv(cx)
			return v * 6, err
		})
		return inner.Await(cx)
	})
	v, err := awaitTask(t, s, outer)
	if v != 42 || err != nil {
		t.Fatalf("got %d, %v", v, err)
	}
}

type parkedProbe struct {
	reachedPark  atomic.Bool
	afterPark    atomic.Bool
	deferred     atomic.Bool
	deferredRuns atomic.Int32
}

func (p *parkedProbe) body(rx *OneshotReceiver[int]) func(cx *Scope) (int, error) {
	return func(cx *Scope) (int, error) {
		defer func() {
			p.deferred.Store(true)
			p.deferredRuns.Add(1)
		}()
		p.reachedPark.Store(true)
		v, err := rx.Recv(cx)
		p.afterPark.Store(true)
		return v, err
	}
}

func TestExecutor_CancelFromBackgroundGoroutine(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	tx, rx := Oneshot[int]()
	var p parkedProbe
	task := s.foreground().Spawn(p.body(rx))
	s.run()
	if !p.reachedPark.Load() {
		t.Fatal("task did not run to its park point")
	}
	var wg sync.WaitGroup
	wg.Go(task.Cancel)
	wg.Wait()
	if p.deferred.Load() {
		t.Fatal("Cancel unwound the body itself; only the poll loop may stop the coroutine")
	}
	if s.pending() != 1 {
		t.Fatalf("Cancel must reschedule the idle task once, %d queued", s.pending())
	}
	s.run()
	if !p.deferred.Load() {
		t.Fatal("deferred call did not run when the poll loop dropped the task")
	}
	if !tx.IsCanceled() {
		t.Fatal("unwinding did not drop the awaited receiver")
	}
	if err := tx.Send(1); !errors.Is(err, ErrCanceled) {
		t.Fatalf("Send after cancel: %v", err)
	}
	s.run()
	if p.afterPark.Load() {
		t.Fatal("body resumed after its park point")
	}
	if p.deferredRuns.Load() != 1 {
		t.Fatalf("deferred ran %d times", p.deferredRuns.Load())
	}
	check()
}

func TestExecutor_CancelFromBackgroundGoroutineWhileRunning(t *testing.T) {
	check := goroutineBaseline(t)
	s := newThreadedScheduler()
	bg := s.background()
	const tasks = 64
	var resumedAfterCancel, started, deferred atomic.Int32
	handles := make([]*Task[int], tasks)
	headers := make([]*header, tasks)
	canceled := make([]atomic.Bool, tasks)
	for i := range handles {
		handles[i] = bg.Spawn(func(cx *Scope) (int, error) {
			started.Add(1)
			defer deferred.Add(1)
			for {
				wasCanceled := canceled[i].Load()
				y := Yield(1)
				Await[struct{}](cx, &y)
				if wasCanceled {
					resumedAfterCancel.Add(1)
				}
			}
		})
		headers[i] = &handles[i].spawned.raw.header
	}
	var wg sync.WaitGroup
	for i := range handles {
		wg.Go(func() {
			for range i % 4 {
				runtime.Gosched()
			}
			handles[i].Cancel()
			canceled[i].Store(true)
		})
	}
	wg.Wait()
	deadline := time.Now().Add(10 * time.Second)
	for _, h := range headers {
		for {
			h.mu.Lock()
			idle := h.state&(stateScheduled|stateRunning) == 0 && h.state&stateClosed != 0
			h.mu.Unlock()
			if idle {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("a canceled task was never dropped")
			}
			time.Sleep(time.Millisecond)
		}
	}
	if started.Load() != deferred.Load() {
		t.Fatalf("%d tasks started but %d unwound", started.Load(), deferred.Load())
	}
	if n := resumedAfterCancel.Load(); n != 0 {
		t.Fatalf("%d tasks resumed past a park after Cancel returned", n)
	}
	check()
}

func TestExecutor_CancelFromSiblingTask(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	fg := s.foreground()
	tx, rx := Oneshot[int]()
	var p parkedProbe
	victim := fg.Spawn(p.body(rx))
	killer := fg.Spawn(func(cx *Scope) (bool, error) {
		y := Yield(1)
		Await[struct{}](cx, &y)
		victim.Cancel()
		return p.deferred.Load(), nil
	})
	s.run()
	sawDeferInline, err := awaitTask(t, s, killer)
	if err != nil {
		t.Fatal(err)
	}
	if sawDeferInline {
		t.Fatal("sibling's Cancel unwound the victim inline")
	}
	if !p.deferred.Load() {
		t.Fatal("victim's deferred call did not run")
	}
	_ = tx.Send(1)
	s.run()
	if p.afterPark.Load() {
		t.Fatal("victim resumed after its park point")
	}
	check()
}

func TestExecutor_CancelFromOwnBody(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	fg := s.foreground()
	var self atomic.Pointer[Task[int]]
	var beforePark, afterPark, deferred atomic.Bool
	task := fg.Spawn(func(cx *Scope) (int, error) {
		defer deferred.Store(true)
		y := Yield(1)
		Await[struct{}](cx, &y)
		self.Load().Cancel()
		beforePark.Store(true)
		y = Yield(1)
		Await[struct{}](cx, &y)
		afterPark.Store(true)
		return 1, nil
	})
	self.Store(task)
	s.run()
	if !beforePark.Load() {
		t.Fatal("body stopped before reaching its next park")
	}
	if afterPark.Load() {
		t.Fatal("body ran past the park after canceling itself")
	}
	if !deferred.Load() {
		t.Fatal("deferred call did not run")
	}
	check()
}

func TestExecutor_RecoveredSentinelStillTerminates(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	tx, rx := Oneshot[int]()
	var recovered, parkedAgain atomic.Bool
	task := s.foreground().Spawn(func(cx *Scope) (int, error) {
		defer func() {
			if r := recover(); r != nil {
				recovered.Store(true)
			}
			y := Yield(1)
			Await[struct{}](cx, &y)
			parkedAgain.Store(true)
		}()
		return rx.Recv(cx)
	})
	s.run()
	task.Cancel()
	s.run()
	if !recovered.Load() {
		t.Fatal("body did not observe the unwind")
	}
	if parkedAgain.Load() {
		t.Fatal("a canceled body resumed after parking again")
	}
	tx.Close()
	s.run()
	check()
}

func TestExecutor_PanicPropagatesOutOfRun(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	task := s.foreground().Spawn(func(cx *Scope) (int, error) {
		y := Yield(1)
		Await[struct{}](cx, &y)
		panic("boom")
	})
	fallible := task.Fallible()
	got := func() (r any) {
		defer func() { r = recover() }()
		s.run()
		return nil
	}()
	if got != "boom" {
		t.Fatalf("Run propagated %v, want boom", got)
	}
	r := BlockOn[Result[int]](fallible)
	if !errors.Is(r.Err, ErrCanceled) {
		t.Fatalf("fallible task after panic: %v", r.Err)
	}
	other := s.foreground().Spawn(func(*Scope) (int, error) { panic("again") })
	func() {
		defer func() { _ = recover() }()
		s.run()
	}()
	msg := func() (r any) {
		defer func() { r = recover() }()
		BlockOn[Result[int]](other)
		return nil
	}()
	if msg != "Task polled after completion" {
		t.Fatalf("awaiting a panicked task: %v", msg)
	}
	check()
}

func TestExecutor_GoexitPropagates(t *testing.T) {
	s := newFIFOScheduler()
	s.foreground().Spawn(func(*Scope) (int, error) {
		runtime.Goexit()
		return 0, nil
	}).Detach()
	done := make(chan bool)
	go func() {
		defer func() { done <- true }()
		s.run()
		done <- false
	}()
	if exited := <-done; !exited {
		t.Fatal("runtime.Goexit in a body did not propagate to the poll loop")
	}
}

func TestExecutor_DetachRunsToCompletion(t *testing.T) {
	s := newFIFOScheduler()
	var done atomic.Bool
	tx, rx := Oneshot[int]()
	s.background().Spawn(func(cx *Scope) (struct{}, error) {
		_, err := rx.Recv(cx)
		done.Store(err == nil)
		return struct{}{}, nil
	}).Detach()
	s.run()
	if err := tx.Send(1); err != nil {
		t.Fatal(err)
	}
	s.run()
	if !done.Load() {
		t.Fatal("detached task did not finish")
	}
}

func TestExecutor_CancelIdempotentAndUseAfterCancelPanics(t *testing.T) {
	s := newFIFOScheduler()
	task := s.foreground().Spawn(func(*Scope) (int, error) { return 1, nil })
	task.Cancel()
	task.Cancel()
	task.Detach()
	s.run()
	got := func() (r any) {
		defer func() { r = recover() }()
		BlockOn[Result[int]](task)
		return nil
	}()
	if got == nil {
		t.Fatal("awaiting a canceled task must panic")
	}
}

func TestExecutor_NeverPolledTaskOwnsNoGoroutine(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	fg := s.foreground()
	tasks := make([]*FallibleTask[int], 100)
	for i := range tasks {
		tasks[i] = fg.Spawn(func(*Scope) (int, error) { return i, nil }).Fallible()
	}
	check()
	s.close()
	for _, task := range tasks {
		if r := BlockOn[Result[int]](task); !errors.Is(r.Err, ErrCanceled) {
			t.Fatalf("task of a closed scheduler: %v", r)
		}
	}
	check()
}

func TestExecutor_ClosedSchedulerStopsParkedTasks(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	tx, rx := Oneshot[int]()
	var p parkedProbe
	s.foreground().Spawn(p.body(rx)).Detach()
	s.background().Spawn(func(cx *Scope) (struct{}, error) {
		s.Timer(time.Hour).Await(cx)
		return struct{}{}, nil
	}).Detach()
	s.run()
	s.close()
	tx.Close()
	if !p.deferred.Load() {
		t.Fatal("a wake into a closed scheduler must drop the runnable and unwind the body")
	}
	if p.afterPark.Load() {
		t.Fatal("body resumed")
	}
	check()
}

func TestExecutor_ManyCanceledTasksLeaveNoGoroutines(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	fg := s.foreground()
	senders := make([]*OneshotSender[int], 200)
	tasks := make([]*Task[int], len(senders))
	for i := range senders {
		tx, rx := Oneshot[int]()
		senders[i] = tx
		tasks[i] = fg.Spawn(func(cx *Scope) (int, error) { return rx.Recv(cx) })
	}
	s.run()
	for _, task := range tasks {
		task.Cancel()
	}
	s.run()
	check()
}

func TestExecutor_SpawnLocationIsCaller(t *testing.T) {
	s := newFIFOScheduler()
	var meta RunnableMeta
	dispatch := func(r *Runnable) {
		meta = *r.Metadata()
		s.push(r)
	}
	e := NewLocalExecutor(1, s, dispatch)
	e.Spawn(func(*Scope) (int, error) { return 0, nil }).Detach()
	if !strings.HasSuffix(meta.Location.File(), "executor_test.go") {
		t.Fatalf("spawn location %v", meta.Location)
	}
	if meta.Location.Function() != "github.com/megascan/gpui-go/scheduler.TestExecutor_SpawnLocationIsCaller" {
		t.Fatalf("spawn function %q", meta.Location.Function())
	}
	m := NewRunnableMetaWithCallersLocation()
	if !strings.HasSuffix(m.Location.String(), "executor_test.go:"+itoa(m.Location.Line())) {
		t.Fatalf("meta location %v", m.Location)
	}
	s.run()
}

func itoa(n int) string {
	var b [20]byte
	i := len(b)
	for {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			return string(b[i:])
		}
	}
}

func TestExecutor_BlockOnAndTimeout(t *testing.T) {
	s := newFIFOScheduler()
	fg := s.foreground()
	if v := fg.BlockOn(Async(func(*Scope) int { return 5 })); v != 5 {
		t.Fatalf("BlockOn: %d", v)
	}
	task := s.background().Spawn(func(cx *Scope) (int, error) {
		s.Timer(100 * time.Millisecond).Await(cx)
		return 123, nil
	})
	if _, ok := fg.BlockWithTimeout(50*time.Millisecond, Future[Result[int]](task)); ok {
		t.Fatal("expected a timeout")
	}
	r := fg.BlockOn(Future[Result[int]](task))
	if r.Value != 123 {
		t.Fatalf("future after timeout: %v", r)
	}
}

func TestExecutor_SpawnDedicated(t *testing.T) {
	before := runtime.NumGoroutine()
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	bg := s.background()
	task := bg.SpawnDedicated(func(e LocalExecutor, cx *Scope) (int, error) {
		child := e.Spawn(func(*Scope) (int, error) { return 99, nil })
		return child.Await(cx)
	})
	v, err := awaitTask(t, s, task)
	if v != 99 || err != nil {
		t.Fatalf("got %d, %v", v, err)
	}
	tx, rx := Oneshot[int]()
	var p parkedProbe
	parked := bg.SpawnDedicated(func(e LocalExecutor, cx *Scope) (int, error) {
		return p.body(rx)(cx)
	})
	deadline := time.Now().Add(5 * time.Second)
	for !p.reachedPark.Load() {
		if time.Now().After(deadline) {
			t.Fatal("dedicated task did not start")
		}
		time.Sleep(time.Millisecond)
	}
	parked.Cancel()
	for !p.deferred.Load() {
		if time.Now().After(deadline) {
			t.Fatal("dedicated task was not dropped")
		}
		time.Sleep(time.Millisecond)
	}
	if err := tx.Send(1); !errors.Is(err, ErrCanceled) {
		t.Fatalf("send to a dropped dedicated task: %v", err)
	}
	if p.afterPark.Load() {
		t.Fatal("dedicated task resumed")
	}
	task, parked = nil, nil
	for time.Now().Before(deadline.Add(5 * time.Second)) {
		runtime.GC()
		if runtime.NumGoroutine() <= before {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	check()
}

func TestExecutor_RendezvousDispositionBeforeDelivery(t *testing.T) {
	s := newFIFOScheduler()
	fg := s.foreground()
	var canceledRan, detachedRan atomic.Bool
	canceled, canceledDelivery := taskRendezvous[int]()
	canceled.Cancel()
	canceledDelivery.deliver(fg.Spawn(func(*Scope) (int, error) {
		canceledRan.Store(true)
		return 1, nil
	}))
	detached, detachedDelivery := taskRendezvous[int]()
	detached.Detach()
	detachedDelivery.deliver(fg.Spawn(func(*Scope) (int, error) {
		detachedRan.Store(true)
		return 2, nil
	}))
	s.run()
	if canceledRan.Load() {
		t.Fatal("a rendezvous canceled before delivery ran its task")
	}
	if !detachedRan.Load() {
		t.Fatal("a rendezvous detached before delivery did not run its task")
	}
	pending, delivery := taskRendezvous[int]()
	if pending.IsReady() {
		t.Fatal("undelivered rendezvous is ready")
	}
	delivery.deliver(ReadyTask(3))
	if !pending.IsReady() {
		t.Fatal("delivered ready task is not ready")
	}
	if r := BlockOn[Result[int]](pending); r.Value != 3 {
		t.Fatalf("delivered value %v", r)
	}
}

func TestExecutor_DedicatedExecutor(t *testing.T) {
	s := newFIFOScheduler()
	d := NewDedicatedExecutor(s.background())
	task := d.Spawn(func(*Scope) (int, error) { return 42, nil })
	v, err := awaitTask(t, s, task)
	if v != 42 || err != nil {
		t.Fatalf("got %d, %v", v, err)
	}
	tx, rx := Oneshot[int]()
	parked := d.Spawn(func(cx *Scope) (int, error) { return rx.Recv(cx) }).Fallible()
	deadline := time.Now().Add(5 * time.Second)
	for {
		rx.s.mu.Lock()
		registered := rx.s.rxWaker != nil
		rx.s.mu.Unlock()
		if registered {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("dedicated task did not park")
		}
		time.Sleep(time.Millisecond)
	}
	d.Close()
	for !d.sender.IsClosed() {
		if time.Now().After(deadline) {
			t.Fatal("dedicated session was not canceled")
		}
		time.Sleep(time.Millisecond)
	}
	_ = tx.Send(1)
	if r := BlockOn[Result[int]](parked); !errors.Is(r.Err, ErrCanceled) {
		t.Fatalf("task woken after Close: %v", r)
	}
}

func TestExecutor_SpawnRealtime(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	task := s.background().SpawnRealtime(func(cx *Scope) (int, error) {
		y := Yield(3)
		Await[struct{}](cx, &y)
		return 7, nil
	})
	if r := BlockOn[Result[int]](task); r.Value != 7 {
		t.Fatalf("realtime task: %v", r)
	}
	check()
}

func TestExecutor_SendLocalExecutor(t *testing.T) {
	s := newFIFOScheduler()
	send := s.foreground().ToSend()
	var task *Task[int]
	var wg sync.WaitGroup
	wg.Go(func() { task = send.Spawn(func(*Scope) (int, error) { return 3, nil }) })
	wg.Wait()
	v, _ := awaitTask(t, s, task)
	if v != 3 {
		t.Fatalf("got %d", v)
	}
}

func TestExecutor_ReadyAndDowncast(t *testing.T) {
	if r := BlockOn[Result[int]](ReadyTask(4)); r.Value != 4 || r.Err != nil {
		t.Fatalf("ReadyTask: %v", r)
	}
	want := errors.New("x")
	if r := BlockOn[Result[int]](ReadyTaskErr[int](want)); r.Err != want {
		t.Fatalf("ReadyTaskErr: %v", r)
	}
	if r := BlockOn[Result[int]](FallibleTaskReady(5)); r.Value != 5 {
		t.Fatalf("FallibleTaskReady: %v", r)
	}
	if !ReadyTask(1).IsReady() {
		t.Fatal("ready task not ready")
	}
	typed := TaskAnyDowncast[string](ReadyTask[any]("s"))
	if r := BlockOn[Result[string]](typed); r.Value != "s" {
		t.Fatalf("downcast: %v", r)
	}
	bad := TaskAnyDowncast[string](ReadyTask[any](1))
	got := func() (r any) {
		defer func() { r = recover() }()
		BlockOn[Result[string]](bad)
		return nil
	}()
	if got != "Task::poll: downcast type mismatch" {
		t.Fatalf("mismatch panic: %v", got)
	}
}

func TestExecutor_WeakBackgroundScheduler(t *testing.T) {
	s := newFIFOScheduler()
	w := downgradeScheduler(s)
	if w.upgrade() != Scheduler(s) {
		t.Fatal("upgrade of a live scheduler")
	}
	s = nil
	for range 5 {
		runtime.GC()
	}
	if w.upgrade() != nil {
		t.Fatal("weak scheduler kept the scheduler alive")
	}
}

func TestExecutor_PriorityWeights(t *testing.T) {
	cases := map[Priority]uint32{PriorityHigh: 60, PriorityMedium: 30, PriorityLow: 10, PriorityRealtimeAudio: 0}
	for p, w := range cases {
		if p.Weight() != w {
			t.Fatalf("%v weight %d", p, p.Weight())
		}
	}
	if DefaultPriority() != PriorityMedium {
		t.Fatal("default priority")
	}
	if NewSessionID(3).String() != "SessionId(3)" {
		t.Fatal(NewSessionID(3).String())
	}
}

func TestExecutor_DedicatedExecutorSpawnAfterClose(t *testing.T) {
	s := newFIFOScheduler()
	d := NewDedicatedExecutor(s.background())
	d.Close()
	task := d.Spawn(func(*Scope) (int, error) { return 1, nil }).Fallible()
	if r := BlockOn[Result[int]](task); !errors.Is(r.Err, ErrCanceled) {
		t.Fatalf("spawn after Close: %v", r)
	}
}

func TestExecutor_DedicatedSessionPanicIsIsolated(t *testing.T) {
	s := newFIFOScheduler()
	var local atomic.Pointer[LocalExecutor]
	root := s.background().SpawnDedicated(func(executor LocalExecutor, cx *Scope) (struct{}, error) {
		local.Store(&executor)
		return struct{}{}, nil
	})
	if r := BlockOn[Result[struct{}]](root); r.Err != nil {
		t.Fatal(r.Err)
	}
	executor := *local.Load()
	panicking := executor.Spawn(func(*Scope) (int, error) { panic("boom") }).Fallible()
	if r := BlockOn[Result[int]](panicking); !errors.Is(r.Err, ErrCanceled) {
		t.Fatalf("panicking task: %v", r)
	}
	later := executor.Spawn(func(*Scope) (int, error) { return 1, nil }).Fallible()
	if r := BlockOn[Result[int]](later); !errors.Is(r.Err, ErrCanceled) {
		t.Fatalf("task spawned after the session panicked: %v", r)
	}
}

func TestExecutor_SpawnRealtimePanicIsIsolated(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	task := s.background().SpawnRealtime(func(*Scope) (int, error) { panic("boom") }).Fallible()
	r := BlockOn[Result[int]](task)
	var panicErr *PanicError
	if !errors.As(r.Err, &panicErr) {
		t.Fatalf("panicking realtime task: %v, want a *PanicError", r)
	}
	if panicErr.Value != "boom" {
		t.Fatalf("PanicError.Value = %v, want boom", panicErr.Value)
	}
	check()
}
