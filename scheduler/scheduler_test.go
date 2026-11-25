package scheduler

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"weak"
)

func gcUntilCollected[T any](p weak.Pointer[T]) bool {
	for range 5 {
		runtime.GC()
		if p.Value() == nil {
			return true
		}
	}
	return false
}

func TestScheduler_ForegroundExecutorSpawn(t *testing.T) {
	result := TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) int {
		task := scheduler.Foreground().Spawn(func(*Scope) (int, error) { return 42, nil })
		v, _ := task.Await(s)
		return v
	})
	if result != 42 {
		t.Fatalf("got %d", result)
	}
}

func TestScheduler_SendLocalExecutorWorkerSubmissionAndWakeup(t *testing.T) {
	scheduler := NewTestScheduler(DefaultTestSchedulerConfig())
	runnables := make(chan *Runnable, 16)
	executor := NewLocalExecutor(NewSessionID(0), scheduler, func(runnable *Runnable) {
		runnables <- runnable
	})
	sendExecutor := executor.ToSend()
	wakers := make(chan Waker, 1)
	results := make(chan int, 1)
	const timeout = 5 * time.Second

	done := make(chan struct{})
	go func() {
		defer close(done)
		polls := 0
		task := sendExecutor.Spawn(func(s *Scope) (int, error) {
			return Await(s, PollFn(func(cx *Context) (int, bool) {
				polls++
				if polls == 1 {
					wakers <- cx.Waker().Clone()
					return 0, false
				}
				if polls != 2 {
					panic(fmt.Sprintf("polls = %d", polls))
				}
				return 42, true
			})), nil
		})
		select {
		case w := <-wakers:
			w.Wake()
		case <-time.After(timeout):
			panic("task was not polled")
		}
		r := BlockOn(Future[Result[int]](task))
		results <- r.Value
	}()

	for range 2 {
		select {
		case r := <-runnables:
			r.Run()
		case <-time.After(timeout):
			t.Fatal("task was not dispatched")
		}
	}
	select {
	case v := <-results:
		if v != 42 {
			t.Fatalf("got %d", v)
		}
	case <-time.After(timeout):
		t.Fatal("task did not complete")
	}
	<-done
}

func TestScheduler_SendLocalExecutorRetainsScheduler(t *testing.T) {
	scheduler := NewTestScheduler(DefaultTestSchedulerConfig())
	weakScheduler := weak.Make(scheduler)
	executor := scheduler.Foreground()
	sendExecutor := executor.ToSend()
	executorClone := sendExecutor
	sendExecutor = SendLocalExecutor{}
	executor = LocalExecutor{}
	scheduler = nil
	_, _ = sendExecutor, executor

	type out struct {
		executor SendLocalExecutor
		task     *Task[int]
	}
	ch := make(chan out)
	go func() {
		task := executorClone.Spawn(func(*Scope) (int, error) { return 42, nil })
		ch <- out{executorClone, task}
	}()
	o := <-ch
	executorClone = SendLocalExecutor{}
	runtime.GC()
	scheduler = weakScheduler.Value()
	if scheduler == nil {
		t.Fatal("scheduler was dropped")
	}
	scheduler.Run()
	if r := BlockOn(Future[Result[int]](o.task)); r.Value != 42 {
		t.Fatalf("got %d", r.Value)
	}
	runtime.KeepAlive(o.executor)
	scheduler = nil
	o.executor = SendLocalExecutor{}
	if !gcUntilCollected(weakScheduler) {
		t.Fatal("scheduler was not dropped")
	}
}

func TestScheduler_BackgroundExecutorSpawn(t *testing.T) {
	TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) struct{} {
		task := scheduler.Background().Spawn(func(*Scope) (int, error) { return 42, nil })
		result, _ := task.Await(s)
		if result != 42 {
			t.Errorf("got %d", result)
		}
		return struct{}{}
	})
}

func TestScheduler_DedicatedExecutorSpawn(t *testing.T) {
	TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) struct{} {
		dedicated := NewDedicatedExecutor(scheduler.Background())
		defer dedicated.Close()
		task := dedicated.Spawn(func(*Scope) (int, error) { return 42, nil })
		if v, _ := task.Await(s); v != 42 {
			t.Errorf("got %d", v)
		}
		return struct{}{}
	})
}

func TestScheduler_SchedulerDropsWithStalledDetachedForegroundTask(t *testing.T) {
	scheduler := NewTestScheduler(DefaultTestSchedulerConfig())
	weakScheduler := weak.Make(scheduler)
	sender, receiver := Oneshot[struct{}]()

	scheduler.Foreground().Spawn(func(s *Scope) (struct{}, error) {
		_, _ = receiver.Recv(s)
		return struct{}{}, nil
	}).Detach()
	scheduler.Run()

	scheduler = nil
	if !gcUntilCollected(weakScheduler) {
		t.Fatal("scheduler was not dropped")
	}
	sender.Close()
}

func TestScheduler_SchedulerDropsWithStalledDetachedBackgroundTask(t *testing.T) {
	scheduler := NewTestScheduler(DefaultTestSchedulerConfig())
	weakScheduler := weak.Make(scheduler)
	sender, receiver := Oneshot[struct{}]()

	scheduler.Background().Spawn(func(s *Scope) (struct{}, error) {
		_, _ = receiver.Recv(s)
		return struct{}{}, nil
	}).Detach()
	scheduler.Run()

	scheduler = nil
	if !gcUntilCollected(weakScheduler) {
		t.Fatal("scheduler was not dropped")
	}
	sender.Close()
}

func TestScheduler_SchedulerDropsWithNeverPolledDedicatedTask(t *testing.T) {
	scheduler := NewTestScheduler(DefaultTestSchedulerConfig())
	weakScheduler := weak.Make(scheduler)

	scheduler.Background().SpawnDedicated(func(LocalExecutor, *Scope) (struct{}, error) {
		return struct{}{}, nil
	}).Detach()

	scheduler = nil
	if !gcUntilCollected(weakScheduler) {
		t.Fatal("scheduler was not dropped")
	}
}

func TestScheduler_ForegroundOrdering(t *testing.T) {
	traces := map[string]struct{}{}

	type traceEntry struct {
		session int
		task    int
	}

	TestSchedulerMany(100, func(scheduler *TestScheduler, s *Scope) struct{} {
		var trace []traceEntry

		foreground1 := scheduler.Foreground()
		for task := range 10 {
			foreground1.Spawn(func(*Scope) (struct{}, error) {
				trace = append(trace, traceEntry{session: 0, task: task})
				return struct{}{}, nil
			}).Detach()
		}

		foreground2 := scheduler.Foreground()
		for task := range 10 {
			foreground2.Spawn(func(*Scope) (struct{}, error) {
				trace = append(trace, traceEntry{session: 1, task: task})
				return struct{}{}, nil
			}).Detach()
		}

		scheduler.Run()

		want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		for session := range 2 {
			var got []int
			for _, e := range trace {
				if e.session == session {
					got = append(got, e.task)
				}
			}
			if !slices.Equal(got, want) {
				t.Errorf("session %d: got %v", session, got)
			}
		}

		traces[fmt.Sprint(trace)] = struct{}{}
		return struct{}{}
	})

	if len(traces) <= 1 {
		t.Fatal("Expected at least two traces")
	}
}

func TestScheduler_TimerOrdering(t *testing.T) {
	TestSchedulerMany(1, func(scheduler *TestScheduler, s *Scope) struct{} {
		background := scheduler.Background()
		type entry struct {
			timer *Timer
			value int
			done  bool
		}
		futures := []*entry{
			{timer: background.Timer(100 * time.Millisecond), value: 2},
			{timer: background.Timer(50 * time.Millisecond), value: 1},
			{timer: background.Timer(150 * time.Millisecond), value: 3},
		}
		var collected []int
		Await(s, PollFn(func(cx *Context) (struct{}, bool) {
			for _, f := range futures {
				if f.done {
					continue
				}
				if _, ok := f.timer.Poll(cx); ok {
					f.done = true
					collected = append(collected, f.value)
				}
			}
			return struct{}{}, len(collected) == len(futures)
		}))
		if !slices.Equal(collected, []int{1, 2, 3}) {
			t.Errorf("got %v", collected)
		}
		return struct{}{}
	})
}

func TestScheduler_ForegroundTaskCanHoldMutBorrowAcrossAwait(t *testing.T) {
	TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) struct{} {
		foreground := scheduler.Foreground()
		sender, receiver := Chan[struct{}](0)

		foreground.Spawn(func(s *Scope) (struct{}, error) {
			receiver.Recv(s)
			return struct{}{}, nil
		}).Detach()

		scheduler.Run()
		if err := sender.TrySend(struct{}{}); err != nil {
			t.Error(err)
		}
		scheduler.Run()
		sender.Close()
		return struct{}{}
	})
}

func TestScheduler_SendFromBgToFg(t *testing.T) {
	TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) struct{} {
		foreground := scheduler.Foreground()
		background := scheduler.Background()

		sender, receiver := Oneshot[int]()

		background.Spawn(func(*Scope) (struct{}, error) {
			if err := sender.Send(42); err != nil {
				panic(err)
			}
			return struct{}{}, nil
		}).Detach()

		task := foreground.Spawn(func(s *Scope) (int, error) {
			v, err := receiver.Recv(s)
			if err != nil {
				panic(err)
			}
			return v, nil
		})
		result, _ := task.Await(s)
		if result != 42 {
			t.Errorf("got %d", result)
		}
		return struct{}{}
	})
}

func TestScheduler_RandomizeOrder(t *testing.T) {
	deterministicResults := map[string]struct{}{}
	for seed := range uint64(10) {
		config := DefaultTestSchedulerConfig()
		config.Seed = seed
		config.RandomizeOrder = false
		order := captureExecutionOrder(config)
		if len(order) != 6 {
			t.Fatalf("got %v", order)
		}
		deterministicResults[strings.Join(order, ",")] = struct{}{}
	}

	if len(deterministicResults) != 1 {
		t.Fatal("Deterministic mode should always produce same execution order")
	}

	randomizedResults := map[string]struct{}{}
	for seed := range uint64(20) {
		config := TestSchedulerConfigWithSeed(seed)
		order := captureExecutionOrder(config)
		if len(order) != 6 {
			t.Fatalf("got %v", order)
		}
		randomizedResults[strings.Join(order, ",")] = struct{}{}
	}

	if len(randomizedResults) <= 1 {
		t.Fatal("Randomized mode should produce multiple different orders")
	}
}

func captureExecutionOrder(config TestSchedulerConfig) []string {
	scheduler := NewTestScheduler(config)
	foreground := scheduler.Foreground()
	background := scheduler.Background()

	sender, receiver := Chan[string](0)

	for i := range 3 {
		sender := sender.Clone()
		foreground.Spawn(func(s *Scope) (struct{}, error) {
			defer sender.Close()
			_ = sender.Send(s, fmt.Sprintf("fg-%d", i))
			return struct{}{}, nil
		}).Detach()
	}

	for i := range 3 {
		sender := sender.Clone()
		background.Spawn(func(s *Scope) (struct{}, error) {
			defer sender.Close()
			_ = sender.Send(s, fmt.Sprintf("bg-%d", i))
			return struct{}{}, nil
		}).Detach()
	}

	sender.Close()
	scheduler.Run()

	var out []string
	for {
		v, ok := receiver.RecvBlocking()
		if !ok {
			return out
		}
		out = append(out, v)
	}
}

func TestScheduler_Block(t *testing.T) {
	scheduler := NewTestScheduler(DefaultTestSchedulerConfig())
	tx, rx := Oneshot[int]()

	scheduler.Background().Spawn(func(*Scope) (struct{}, error) {
		if err := tx.Send(42); err != nil {
			panic(err)
		}
		return struct{}{}, nil
	}).Detach()

	result := scheduler.Foreground().BlockOn(Future[Result[int]](rx))
	if result.Err != nil || result.Value != 42 {
		t.Fatalf("got %v", result)
	}
}

func recoverMessage(f func()) (msg string, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
			panicked = true
		}
	}()
	f()
	return "", false
}

func TestScheduler_ParkingPanics(t *testing.T) {
	config := DefaultTestSchedulerConfig()
	config.CapturePendingTraces = true
	scheduler := NewTestScheduler(config)
	tx, rx := Oneshot[struct{}]()
	defer tx.Close()
	msg, panicked := recoverMessage(func() {
		scheduler.Foreground().BlockOn(Async(func(s *Scope) struct{} {
			if _, err := rx.Recv(s); err != nil {
				panic(err)
			}
			return struct{}{}
		}))
	})
	if !panicked || !strings.Contains(msg, "Parking forbidden.") {
		t.Fatalf("expected panic containing %q, got %v %q", "Parking forbidden.", panicked, msg)
	}
}

func TestScheduler_BlockWithParking(t *testing.T) {
	config := DefaultTestSchedulerConfig()
	config.AllowParking = true
	scheduler := NewTestScheduler(config)
	tx, rx := Oneshot[int]()

	scheduler.Background().Spawn(func(*Scope) (struct{}, error) {
		if err := tx.Send(42); err != nil {
			panic(err)
		}
		return struct{}{}, nil
	}).Detach()

	result := scheduler.Foreground().BlockOn(Future[Result[int]](rx))
	if result.Err != nil || result.Value != 42 {
		t.Fatalf("got %v", result)
	}
}

func TestScheduler_HelperMethods(t *testing.T) {
	result := TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) int {
		background := scheduler.Background()
		v, _ := background.Spawn(func(*Scope) (int, error) { return 42, nil }).Await(s)
		return v
	})
	if result != 42 {
		t.Fatalf("got %d", result)
	}

	results := TestSchedulerMany(3, func(scheduler *TestScheduler, s *Scope) int {
		background := scheduler.Background()
		v, _ := background.Spawn(func(*Scope) (int, error) { return 10, nil }).Await(s)
		return v
	})
	if !slices.Equal(results, []int{10, 10, 10}) {
		t.Fatalf("got %v", results)
	}
}

func TestScheduler_ManyWithArbitrarySeed(t *testing.T) {
	for _, seed := range []uint64{0, 1, 5, 42} {
		var seedsSeen []uint64
		const iterations = 3

		for currentSeed := seed; currentSeed < seed+iterations; currentSeed++ {
			scheduler := NewTestScheduler(TestSchedulerConfigWithSeed(currentSeed))
			capturedSeed := currentSeed
			scheduler.Foreground().BlockOn(PollFn(func(*Context) (struct{}, bool) {
				seedsSeen = append(seedsSeen, capturedSeed)
				return struct{}{}, true
			}))
			scheduler.Run()
		}

		want := []uint64{seed, seed + 1, seed + 2}
		if !slices.Equal(seedsSeen, want) {
			t.Fatalf("Expected %d iterations starting at seed %d, got %v", iterations, seed, seedsSeen)
		}
	}
}

type pollsYield struct {
	polls int
}

func (y *pollsYield) Poll(cx *Context) (struct{}, bool) {
	y.polls--
	if y.polls == 0 {
		return struct{}{}, true
	}
	cx.Waker().WakeByRef()
	return struct{}{}, false
}

func TestScheduler_BlockWithTimeout(t *testing.T) {
	TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) struct{} {
		foreground := scheduler.Foreground()
		future := PollFn(func(*Context) (int, bool) { return 42, true })
		output, ok := foreground.BlockWithTimeout(100*time.Millisecond, future)
		if !ok || output != 42 {
			t.Errorf("got %d, %v", output, ok)
		}
		return struct{}{}
	})

	TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) struct{} {
		scheduler.SetTimeoutTicks(TickRange{Start: 0, End: 0})

		foreground := scheduler.Foreground()
		future := PollFn(func(*Context) (struct{}, bool) { return struct{}{}, false })
		if _, ok := foreground.BlockWithTimeout(50*time.Millisecond, future); ok {
			t.Error("future should not have finished")
		}
		return struct{}{}
	})

	results := map[string]struct{}{}
	TestSchedulerMany(100, func(scheduler *TestScheduler, s *Scope) struct{} {
		task := scheduler.Background().Spawn(func(s *Scope) (int, error) {
			Await[struct{}](s, &pollsYield{polls: 10})
			return 42, nil
		})
		output, ok := scheduler.Foreground().BlockWithTimeout(50*time.Millisecond, Future[Result[int]](task))
		if ok {
			results[fmt.Sprint(output.Value)] = struct{}{}
		} else {
			results["None"] = struct{}{}
			task.Cancel()
		}
		return struct{}{}
	})
	if _, ok := results["None"]; !ok || len(results) != 2 {
		t.Fatalf("got %v", results)
	}
	if _, ok := results["42"]; !ok {
		t.Fatalf("got %v", results)
	}

	TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) struct{} {
		scheduler.SetTimeoutTicks(TickRange{Start: 0, End: 0})

		background := scheduler.Background()

		task := background.Spawn(func(s *Scope) (int, error) {
			scheduler.Timer(100 * time.Millisecond).Await(s)
			return 123, nil
		})

		_, ok := scheduler.Foreground().BlockWithTimeout(50*time.Millisecond, Future[Result[int]](task))
		if ok {
			t.Error("expected timeout before advancing the clock enough for the timer")
		}

		scheduler.AdvanceClock(100 * time.Millisecond)
		scheduler.Run()

		output := scheduler.Foreground().BlockOn(Future[Result[int]](task))
		if output.Value != 123 {
			t.Errorf("got %d", output.Value)
		}
		return struct{}{}
	})
}

func TestScheduler_BlockDoesNotProgressSameSessionForeground(t *testing.T) {
	task2MadeProgressOnce := false
	TestSchedulerMany(1000, func(scheduler *TestScheduler, s *Scope) struct{} {
		foreground1 := scheduler.Foreground()
		foreground2 := scheduler.Foreground()

		task1 := foreground1.Spawn(func(*Scope) (struct{}, error) { return struct{}{}, nil })
		task2 := foreground2.Spawn(func(*Scope) (struct{}, error) { return struct{}{}, nil })

		foreground1.BlockOn(Async(func(s *Scope) struct{} {
			Await[struct{}](s, scheduler.YieldRandom())
			if task1.IsReady() {
				t.Error("task1 made progress while its session was blocked")
			}
			task2MadeProgressOnce = task2MadeProgressOnce || task2.IsReady()
			return struct{}{}
		}))

		_ = task1.Wait(s)
		_ = task2.Wait(s)
		return struct{}{}
	})

	if !task2MadeProgressOnce {
		t.Fatal("Expected task from different foreground executor to make progress (at least once)")
	}
}

func sendWakerToThread(wakerTx chan<- Waker) Future[struct{}] {
	sent := false
	return PollFn(func(cx *Context) (struct{}, bool) {
		if !sent {
			sent = true
			wakerTx <- cx.Waker().Clone()
		}
		return struct{}{}, true
	})
}

func expectNonDeterminismPanic(t *testing.T, scheduler *TestScheduler) {
	t.Helper()
	msg, panicked := recoverMessage(scheduler.EndTest)
	if !panicked {
		t.Fatal("Expected end_test to panic")
	}
	if !strings.Contains(msg, "Your test is not deterministic") {
		t.Fatalf("Expected panic message to contain non-determinism error, got: %s", msg)
	}
}

func TestScheduler_NondeterministicWakeDetection(t *testing.T) {
	config := DefaultTestSchedulerConfig()
	config.AllowParking = false
	scheduler := NewTestScheduler(config)

	wakerTx := make(chan Waker, 1)
	scheduler.Foreground().BlockOn(sendWakerToThread(wakerTx))

	var wg sync.WaitGroup
	wg.Go(func() {
		waker := <-wakerTx
		waker.Wake()
	})
	wg.Wait()

	expectNonDeterminismPanic(t, scheduler)
}

func TestScheduler_NondeterministicWakeAllowedWithParking(t *testing.T) {
	config := DefaultTestSchedulerConfig()
	config.AllowParking = true
	scheduler := NewTestScheduler(config)

	wakerTx := make(chan Waker, 1)
	go func() {
		waker := <-wakerTx
		waker.Wake()
	}()

	wakerSent := false
	scheduler.Foreground().BlockOn(PollFn(func(cx *Context) (struct{}, bool) {
		if !wakerSent {
			wakerSent = true
			wakerTx <- cx.Waker().Clone()
			return struct{}{}, false
		}
		return struct{}{}, true
	}))
}

func TestScheduler_NondeterministicWakerDropDetection(t *testing.T) {
	config := DefaultTestSchedulerConfig()
	config.AllowParking = false
	scheduler := NewTestScheduler(config)

	wakerTx := make(chan Waker, 1)
	scheduler.Foreground().BlockOn(sendWakerToThread(wakerTx))

	var wg sync.WaitGroup
	wg.Go(func() {
		waker := <-wakerTx
		waker.Drop()
	})
	wg.Wait()

	expectNonDeterminismPanic(t, scheduler)
}

func TestScheduler_BackgroundPriorityScheduling(t *testing.T) {
	highBeforeLowCount := 0
	const iterations = 100

	for seed := range uint64(iterations) {
		scheduler := NewTestScheduler(TestSchedulerConfigWithSeed(seed))
		background := scheduler.Background()

		var mu sync.Mutex
		var executionOrder []string

		for i := range 3 {
			background.SpawnWithPriority(PriorityLow, func(*Scope) (struct{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, fmt.Sprintf("low-%d", i))
				mu.Unlock()
				return struct{}{}, nil
			}).Detach()
		}

		for i := range 3 {
			background.SpawnWithPriority(PriorityHigh, func(*Scope) (struct{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, fmt.Sprintf("high-%d", i))
				mu.Unlock()
				return struct{}{}, nil
			}).Detach()
		}

		scheduler.Run()

		mu.Lock()
		highInFirstHalf := 0
		for _, s := range executionOrder[:3] {
			if strings.HasPrefix(s, "high") {
				highInFirstHalf++
			}
		}
		mu.Unlock()

		if highInFirstHalf >= 2 {
			highBeforeLowCount++
		}
	}

	if highBeforeLowCount <= iterations/2 {
		t.Fatalf("Expected high priority tasks to run before low priority tasks more often. Got %d out of %d iterations", highBeforeLowCount, iterations)
	}
}

func TestScheduler_SpawnDedicatedBasicRoundTrip(t *testing.T) {
	result := TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) int {
		v, _ := scheduler.Background().SpawnDedicated(func(LocalExecutor, *Scope) (int, error) {
			return 42, nil
		}).Await(s)
		return v
	})
	if result != 42 {
		t.Fatalf("got %d", result)
	}
}

func TestScheduler_SpawnDedicatedNotSendFuture(t *testing.T) {
	result := TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) int32 {
		v, _ := scheduler.Background().SpawnDedicated(func(LocalExecutor, *Scope) (int32, error) {
			state := new(int32)
			for range 5 {
				*state++
			}
			return *state, nil
		}).Await(s)
		return v
	})
	if result != 5 {
		t.Fatalf("got %d", result)
	}
}

func TestScheduler_SpawnDedicatedSendClosureCaptures(t *testing.T) {
	observed := TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) int32 {
		var mu sync.Mutex
		shared := int32(0)
		_, _ = scheduler.Background().SpawnDedicated(func(LocalExecutor, *Scope) (struct{}, error) {
			mu.Lock()
			shared = 7
			mu.Unlock()
			return struct{}{}, nil
		}).Await(s)
		mu.Lock()
		defer mu.Unlock()
		return shared
	})
	if observed != 7 {
		t.Fatalf("got %d", observed)
	}
}

func TestScheduler_SpawnDedicatedInnerSpawnLocal(t *testing.T) {
	result := TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) int32 {
		v, _ := scheduler.Background().SpawnDedicated(func(executor LocalExecutor, s *Scope) (int32, error) {
			inner := new(int32)
			child := executor.Spawn(func(*Scope) (int32, error) {
				*inner = 99
				return *inner, nil
			})
			return child.Await(s)
		}).Await(s)
		return v
	})
	if result != 99 {
		t.Fatalf("got %d", result)
	}
}

type dedicatedOutcome struct {
	trace   []uint32
	outputs []uint32
}

func dedicatedDeterminismRun(scheduler *TestScheduler, s *Scope) dedicatedOutcome {
	var mu sync.Mutex
	var trace []uint32

	background := scheduler.Background()
	var tasks []*Task[uint32]
	for id := range uint32(4) {
		task := background.SpawnDedicated(func(executor LocalExecutor, s *Scope) (uint32, error) {
			for step := range uint32(3) {
				mu.Lock()
				trace = append(trace, id*100+step)
				mu.Unlock()
				_ = executor.Spawn(func(*Scope) (struct{}, error) { return struct{}{}, nil }).Wait(s)
			}
			return id, nil
		})
		tasks = append(tasks, task)
	}

	var outputs []uint32
	for _, task := range tasks {
		v, _ := task.Await(s)
		outputs = append(outputs, v)
	}

	mu.Lock()
	defer mu.Unlock()
	return dedicatedOutcome{trace: slices.Clone(trace), outputs: outputs}
}

func TestScheduler_SpawnDedicatedDeterminismUnderMany(t *testing.T) {
	outcomes := TestSchedulerMany(20, dedicatedDeterminismRun)
	outcomesReplay := TestSchedulerMany(20, dedicatedDeterminismRun)

	if !slices.EqualFunc(outcomes, outcomesReplay, func(a, b dedicatedOutcome) bool {
		return slices.Equal(a.trace, b.trace) && slices.Equal(a.outputs, b.outputs)
	}) {
		t.Fatal("per-seed outcomes should be reproducible")
	}

	anyInterleaved := slices.ContainsFunc(outcomes, func(o dedicatedOutcome) bool {
		for i := 1; i < len(o.trace); i++ {
			if o.trace[i-1]/100 != o.trace[i]/100 {
				return true
			}
		}
		return false
	})
	if !anyInterleaved {
		t.Fatal("expected at least one seed to interleave dedicated tasks")
	}
}

func TestScheduler_SpawnDedicatedDroppingTaskCancelsFuture(t *testing.T) {
	counterAfter := TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) uint32 {
		var mu sync.Mutex
		counter := uint32(0)
		resumeTx, resumeRx := Oneshot[struct{}]()

		task := scheduler.Background().SpawnDedicated(func(_ LocalExecutor, s *Scope) (struct{}, error) {
			mu.Lock()
			counter = 1
			mu.Unlock()
			_, _ = resumeRx.Recv(s)
			mu.Lock()
			counter = 2
			mu.Unlock()
			return struct{}{}, nil
		})

		scheduler.Run()
		mu.Lock()
		if counter != 1 {
			t.Errorf("counter = %d", counter)
		}
		mu.Unlock()

		task.Cancel()
		_ = resumeTx.Send(struct{}{})
		scheduler.Run()

		mu.Lock()
		defer mu.Unlock()
		return counter
	})

	if counterAfter != 1 {
		t.Fatal("dropping the dedicated task must cancel the root future before its second write")
	}
}

func TestScheduler_SpawnDedicatedDetachedChildRunsAfterRootCompletes(t *testing.T) {
	childRan := TestSchedulerOnce(func(scheduler *TestScheduler, s *Scope) bool {
		var mu sync.Mutex
		childRan := false

		task := scheduler.Background().SpawnDedicated(func(executor LocalExecutor, _ *Scope) (struct{}, error) {
			executor.Spawn(func(*Scope) (struct{}, error) {
				mu.Lock()
				childRan = true
				mu.Unlock()
				return struct{}{}, nil
			}).Detach()
			return struct{}{}, nil
		})

		_ = task.Wait(s)

		scheduler.Run()

		mu.Lock()
		defer mu.Unlock()
		return childRan
	})

	if !childRan {
		t.Fatal("detached child must complete after the root, not be cancelled with it")
	}
}
