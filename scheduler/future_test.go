package scheduler

import (
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func TestFuture_OneshotSendRecv(t *testing.T) {
	s := newFIFOScheduler()
	tx, rx := Oneshot[string]()
	task := s.foreground().Spawn(func(cx *Scope) (string, error) { return rx.Recv(cx) })
	s.run()
	if task.IsReady() {
		t.Fatal("receiver completed before send")
	}
	if err := tx.Send("hi"); err != nil {
		t.Fatal(err)
	}
	if v, err := awaitTask(t, s, task); v != "hi" || err != nil {
		t.Fatalf("got %q, %v", v, err)
	}
}

func TestFuture_OneshotCanceled(t *testing.T) {
	tx, rx := Oneshot[int]()
	tx.Close()
	if r := BlockOn[Result[int]](rx); !errors.Is(r.Err, ErrCanceled) {
		t.Fatalf("dropped sender: %v", r)
	}
	tx2, rx2 := Oneshot[int]()
	if _, ok, err := rx2.TryRecv(); ok || err != nil {
		t.Fatal("TryRecv before send")
	}
	rx2.Close()
	if !tx2.IsCanceled() {
		t.Fatal("IsCanceled after receiver drop")
	}
	if err := tx2.Send(1); !errors.Is(err, ErrCanceled) {
		t.Fatalf("Send to dropped receiver: %v", err)
	}
	tx3, rx3 := Oneshot[int]()
	_ = tx3.Send(9)
	if v, ok, err := rx3.TryRecv(); v != 9 || !ok || err != nil {
		t.Fatalf("TryRecv after send: %d %v %v", v, ok, err)
	}
}

func TestFuture_ChanUnbounded(t *testing.T) {
	s := newFIFOScheduler()
	tx, rx := Chan[int](0)
	fg := s.foreground()
	for i := range 3 {
		sender := tx.Clone()
		fg.Spawn(func(cx *Scope) (struct{}, error) {
			defer sender.Close()
			return struct{}{}, sender.Send(cx, i)
		}).Detach()
	}
	tx.Close()
	collect := fg.Spawn(func(cx *Scope) ([]int, error) {
		var out []int
		for {
			v, ok := rx.Recv(cx)
			if !ok {
				return out, nil
			}
			out = append(out, v)
		}
	})
	got, _ := awaitTask(t, s, collect)
	if !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("got %v", got)
	}
	if _, ok, closed := rx.TryRecv(); ok || !closed {
		t.Fatal("drained channel must report closed")
	}
}

func TestFuture_ChanBoundedParksSender(t *testing.T) {
	s := newFIFOScheduler()
	tx, rx := Chan[int](1)
	fg := s.foreground()
	var sent atomic.Int32
	fg.Spawn(func(cx *Scope) (struct{}, error) {
		defer tx.Close()
		for i := range 3 {
			if err := tx.Send(cx, i); err != nil {
				return struct{}{}, err
			}
			sent.Add(1)
		}
		return struct{}{}, nil
	}).Detach()
	s.run()
	if sent.Load() != 1 {
		t.Fatalf("bounded channel accepted %d values", sent.Load())
	}
	if err := tx.TrySend(5); !errors.Is(err, ErrFull) {
		t.Fatalf("TrySend on a full channel: %v", err)
	}
	var got []int
	for {
		v, ok, closed := rx.TryRecv()
		if ok {
			got = append(got, v)
			s.run()
			continue
		}
		if closed {
			break
		}
		s.run()
	}
	if !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("got %v", got)
	}
	rx.Close()
}

func TestFuture_ChanReceiverClosed(t *testing.T) {
	tx, rx := Chan[int](0)
	rx.Close()
	if !tx.IsClosed() {
		t.Fatal("IsClosed after receiver drop")
	}
	if err := tx.TrySend(1); !errors.Is(err, ErrClosed) {
		t.Fatalf("send after receiver drop: %v", err)
	}
}

func TestFuture_ChanRecvBlockingAcrossGoroutines(t *testing.T) {
	tx, rx := Chan[int](0)
	go func() {
		for i := range 100 {
			_ = tx.TrySend(i)
		}
		tx.Close()
	}()
	sum := 0
	for {
		v, ok := rx.RecvBlocking()
		if !ok {
			break
		}
		sum += v
	}
	if sum != 4950 {
		t.Fatalf("sum %d", sum)
	}
}

func TestFuture_SelectDropsLosers(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	tx, rx := Oneshot[int]()
	var loserDeferred atomic.Bool
	loser := Async(func(cx *Scope) int {
		defer loserDeferred.Store(true)
		y := Yield(1000)
		Await[struct{}](cx, &y)
		return 0
	})
	task := s.foreground().Spawn(func(cx *Scope) (int, error) {
		var got int
		i := Select(cx,
			Case(s.Timer(time.Second), func(struct{}) { got = -1 }),
			Case[Result[int]](rx, func(r Result[int]) { got = r.Value }),
			Case(loser, func(int) { got = -2 }),
		)
		return got*10 + i, nil
	})
	s.tick()
	_ = tx.Send(4)
	v, _ := awaitTask(t, s, task)
	if v != 41 {
		t.Fatalf("select result %d", v)
	}
	if !loserDeferred.Load() {
		t.Fatal("losing coroutine was not dropped")
	}
	check()
}

func TestFuture_JoinAndJoin2(t *testing.T) {
	s := newFIFOScheduler()
	bg := s.background()
	task := s.foreground().Spawn(func(cx *Scope) (int, error) {
		futs := []Future[Result[int]]{
			bg.Spawn(func(cx *Scope) (int, error) {
				s.Timer(20 * time.Millisecond).Await(cx)
				return 1, nil
			}),
			bg.Spawn(func(cx *Scope) (int, error) {
				s.Timer(10 * time.Millisecond).Await(cx)
				return 2, nil
			}),
		}
		rs := Join(cx, futs...)
		a, b := Join2(cx, Async(func(*Scope) int { return 10 }), Async(func(cx *Scope) string {
			y := Yield(2)
			Await[struct{}](cx, &y)
			return "x"
		}))
		if b != "x" {
			return 0, errors.New(b)
		}
		return rs[0].Value*100 + rs[1].Value*10 + a, nil
	})
	v, err := awaitTask(t, s, task)
	if v != 130 || err != nil {
		t.Fatalf("got %d, %v", v, err)
	}
}

func TestFuture_NestedCoroutineDroppedWithParent(t *testing.T) {
	check := goroutineBaseline(t)
	s := newFIFOScheduler()
	tx, rx := Oneshot[int]()
	var innerDeferred, outerDeferred atomic.Bool
	task := s.foreground().Spawn(func(cx *Scope) (int, error) {
		defer outerDeferred.Store(true)
		inner := Async(func(cx *Scope) int {
			defer innerDeferred.Store(true)
			v, _ := rx.Recv(cx)
			return v
		})
		return Await(cx, inner), nil
	})
	s.run()
	task.Cancel()
	s.run()
	if !outerDeferred.Load() || !innerDeferred.Load() {
		t.Fatalf("outer %v inner %v", outerDeferred.Load(), innerDeferred.Load())
	}
	if !tx.IsCanceled() {
		t.Fatal("inner receiver was not dropped")
	}
	check()
}

func TestFuture_UnownedIsNotDropped(t *testing.T) {
	s := newFIFOScheduler()
	tx, rx := Oneshot[int]()
	task := s.foreground().Spawn(func(cx *Scope) (int, error) {
		Select(cx,
			Case(s.Timer(time.Millisecond), nil),
			Case(Unowned[Result[int]](rx), nil),
		)
		return 0, nil
	})
	if _, err := awaitTask(t, s, task); err != nil {
		t.Fatal(err)
	}
	if tx.IsCanceled() {
		t.Fatal("an Unowned future was dropped by Select")
	}
	_ = tx.Send(3)
	if r := BlockOn[Result[int]](rx); r.Value != 3 {
		t.Fatalf("receiver after select: %v", r)
	}
}

func TestFuture_YieldAndPollFn(t *testing.T) {
	polls := 0
	f := PollFn(func(cx *Context) (int, bool) {
		polls++
		if polls < 3 {
			cx.Waker().WakeByRef()
			return 0, false
		}
		return polls, true
	})
	if v := BlockOn(f); v != 3 {
		t.Fatalf("poll_fn: %d", v)
	}
	y := Yield(2)
	cx := NewContext(noopWaker{})
	if _, ok := y.Poll(cx); ok {
		t.Fatal("Yield(2) ready on first poll")
	}
	if _, ok := y.Poll(cx); ok {
		t.Fatal("Yield(2) ready on second poll")
	}
	if _, ok := y.Poll(cx); !ok {
		t.Fatal("Yield(2) not ready on third poll")
	}
}

func TestFuture_AsyncResumedAfterCompletionPanics(t *testing.T) {
	f := Async(func(*Scope) int { return 1 })
	cx := NewContext(noopWaker{})
	if v, ok := f.Poll(cx); !ok || v != 1 {
		t.Fatal("async block")
	}
	got := func() (r any) {
		defer func() { r = recover() }()
		f.Poll(cx)
		return nil
	}()
	if got != "`async fn` resumed after completion" {
		t.Fatalf("got %v", got)
	}
}

func TestFuture_AwaitOutsideCoroutinePanics(t *testing.T) {
	got := func() (r any) {
		defer func() { r = recover() }()
		Await[struct{}](&Scope{}, PollFn(func(*Context) (struct{}, bool) { return struct{}{}, true }))
		return nil
	}()
	if got != "scheduler: await outside a coroutine body" {
		t.Fatalf("got %v", got)
	}
}

func TestClock_TestClock(t *testing.T) {
	c := NewTestClock()
	want := time.Date(2025, 7, 1, 23, 59, 58, 0, time.UTC)
	if !c.UTCNow().Equal(want) {
		t.Fatalf("start %v", c.UTCNow())
	}
	start := c.Now()
	c.Advance(3 * time.Second)
	if d := c.Now().Sub(start); d != 3*time.Second {
		t.Fatalf("advanced %v", d)
	}
	if !c.UTCNow().Equal(want.Add(3 * time.Second)) {
		t.Fatalf("utc after advance %v", c.UTCNow())
	}
	later := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c.SetUTCNow(later)
	if !c.UTCNow().Equal(later) {
		t.Fatal("SetUTCNow")
	}
	if start.Sub(c.Now()) != 0 {
		t.Fatal("Instant subtraction must saturate at zero")
	}
	if !start.Before(c.Now()) || !c.Now().After(start) || start.Compare(c.Now()) != -1 {
		t.Fatal("ordering")
	}
}

func TestFuture_ChanSendKeepsOneParkedWaker(t *testing.T) {
	tx, rx := Chan[int](1)
	if err := tx.TrySend(0); err != nil {
		t.Fatal(err)
	}
	var wakes atomic.Int32
	cx := NewContext(WakerFn(func() { wakes.Add(1) }))
	f := &sendFuture[int]{tx: tx, v: 1}
	for range 5 {
		if _, ok := f.Poll(cx); ok {
			t.Fatal("send on a full channel completed")
		}
	}
	if n := len(tx.s.txWaiters); n != 1 {
		t.Fatalf("re-polled send registered %d wakers", n)
	}
	f.Cancel()
	if n := len(tx.s.txWaiters); n != 0 {
		t.Fatalf("canceled send left %d wakers", n)
	}
	g := &sendFuture[int]{tx: tx, v: 2}
	g.Poll(cx)
	g.Poll(cx)
	if v, ok, _ := rx.TryRecv(); !ok || v != 0 {
		t.Fatalf("TryRecv: %d %v", v, ok)
	}
	if wakes.Load() != 1 {
		t.Fatalf("pop woke the parked sender %d times", wakes.Load())
	}
	if err, ok := g.Poll(cx); !ok || err != nil {
		t.Fatalf("send after pop: %v %v", err, ok)
	}
	if n := len(tx.s.txWaiters); n != 0 {
		t.Fatalf("completed send left %d wakers", n)
	}
	rx.Close()
}

func TestFuture_CanceledSendInTaskDropsWaker(t *testing.T) {
	s := newFIFOScheduler()
	tx, rx := Chan[int](1)
	_ = tx.TrySend(0)
	task := s.foreground().Spawn(func(cx *Scope) (struct{}, error) {
		return struct{}{}, tx.Send(cx, 1)
	})
	s.run()
	if n := len(tx.s.txWaiters); n != 1 {
		t.Fatalf("parked send registered %d wakers", n)
	}
	task.Cancel()
	s.run()
	if n := len(tx.s.txWaiters); n != 0 {
		t.Fatalf("canceled send left %d wakers", n)
	}
	rx.Close()
}

func TestFuture_RecoveredCancelUnwindDoesNotResume(t *testing.T) {
	s := newFIFOScheduler()
	_, rx := Oneshot[int]()
	var isCancel, resumed, awaitedAgain atomic.Bool
	task := s.foreground().Spawn(func(cx *Scope) (int, error) {
		func() {
			defer func() { isCancel.Store(IsCancelUnwind(recover())) }()
			_, _ = rx.Recv(cx)
		}()
		resumed.Store(true)
		y := Yield(1)
		Await[struct{}](cx, &y)
		awaitedAgain.Store(true)
		return 1, nil
	})
	s.run()
	task.Cancel()
	s.run()
	if !isCancel.Load() {
		t.Fatal("recovered value is not the cancel unwind")
	}
	if !resumed.Load() {
		t.Fatal("body did not continue past its recover")
	}
	if awaitedAgain.Load() {
		t.Fatal("a canceled body awaited again")
	}
}

func TestFuture_SelectWithNoCasesPanics(t *testing.T) {
	f := Async(func(cx *Scope) int { return Select(cx) })
	got := func() (r any) {
		defer func() { r = recover() }()
		f.Poll(NewContext(noopWaker{}))
		return nil
	}()
	if got != "scheduler: Select with no cases" {
		t.Fatalf("got %v", got)
	}
}

func TestFuture_CallbackDepthOutsideCallbacks(t *testing.T) {
	if depth := CallbackDepth(); depth != 0 {
		t.Fatalf("CallbackDepth() = %d outside native callbacks, want 0", depth)
	}
}
