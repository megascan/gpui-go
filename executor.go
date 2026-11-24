package gpui

import (
	"log/slog"
	"os"
	"runtime"
	"sync/atomic"
	"time"
	"weak"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// BackgroundExecutor spawns tasks on background threads.
type BackgroundExecutor struct {
	inner      scheduler.BackgroundExecutor
	dispatcher PlatformDispatcher
}

// ForegroundExecutor spawns tasks on the main thread.
type ForegroundExecutor struct {
	inner      scheduler.LocalExecutor
	dispatcher PlatformDispatcher
}

// SendForegroundExecutor is a handle that spawns tasks on the main thread from any goroutine.
type SendForegroundExecutor struct {
	inner scheduler.SendLocalExecutor
}

// Spawn enqueues body to run on the main thread, even when submitted from another goroutine.
func (e SendForegroundExecutor) Spawn[R any](body func(s *scheduler.Scope) (R, error)) *Task[R] {
	return e.inner.SpawnAt(scheduler.CallerLocation(1), body)
}

// TaskDetachAndLogErr runs task to completion on the foreground executor and logs its error
// (TaskExt::detach_and_log_err).
func TaskDetachAndLogErr[T any](task *Task[T], cx *App) {
	location := gpuiutil.CallerLocation(1)
	cx.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		gpuiutil.TryFutureExt[T](func() (T, error) { return task.Await(s) }).LogTrackedErr(location).Run()
		return struct{}{}, nil
	}).Detach()
}

// TaskDetachAndLogErrWithBacktrace is TaskDetachAndLogErr formatting the error with %+v
// (TaskExt::detach_and_log_err_with_backtrace).
func TaskDetachAndLogErrWithBacktrace[T any](task *Task[T], cx *App) {
	location := gpuiutil.CallerLocation(1)
	cx.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		gpuiutil.TryFutureExtBacktrace[T](func() (T, error) { return task.Await(s) }).LogTrackedErrWithBacktrace(location).Run()
		return struct{}{}, nil
	}).Detach()
}

// NewBackgroundExecutor returns a background executor over dispatcher: the dispatcher's
// TestScheduler for a TestDispatcher, a PlatformScheduler otherwise.
func NewBackgroundExecutor(dispatcher PlatformDispatcher) BackgroundExecutor {
	var s scheduler.Scheduler
	if testDispatcher := dispatcher.AsTest(); testDispatcher != nil {
		s = testDispatcher.Scheduler()
	} else {
		s = NewPlatformScheduler(dispatcher)
	}
	return BackgroundExecutor{
		inner:      scheduler.NewBackgroundExecutor(s),
		dispatcher: dispatcher,
	}
}

// SchedulerExecutor returns the underlying scheduler.BackgroundExecutor.
func (e BackgroundExecutor) SchedulerExecutor() scheduler.BackgroundExecutor {
	return e.inner
}

// PreventAppNap prevents App Nap-style throttling until the returned guard is released.
func (e BackgroundExecutor) PreventAppNap(reason string) *ActivityGuard {
	return e.dispatcher.PreventAppNap(reason)
}

// Spawn enqueues body to run to completion on a background thread.
func (e BackgroundExecutor) Spawn[R any](body func(s *scheduler.Scope) (R, error)) *Task[R] {
	return e.spawnWithPriorityAt(scheduler.CallerLocation(1), scheduler.DefaultPriority(), body)
}

// SpawnWithPriority enqueues body on a background thread with priority; PriorityRealtimeAudio
// runs it on a dedicated realtime thread.
func (e BackgroundExecutor) SpawnWithPriority[R any](priority Priority, body func(s *scheduler.Scope) (R, error)) *Task[R] {
	return e.spawnWithPriorityAt(scheduler.CallerLocation(1), priority, body)
}

func (e BackgroundExecutor) spawnWithPriorityAt[R any](location scheduler.Location, priority Priority, body func(s *scheduler.Scope) (R, error)) *Task[R] {
	if priority == scheduler.PriorityRealtimeAudio {
		return e.inner.SpawnRealtimeAt(location, body)
	}
	return e.inner.SpawnWithPriorityAt(location, priority, body)
}

// Scoped runs the background tasks f spawns into its Scope and waits for all of them. If the
// awaiting task is canceled, the scoped tasks are canceled and Scoped waits for their bodies to
// finish before unwinding.
func (e BackgroundExecutor) Scoped(cx scheduler.Awaiter, f func(scope *Scope)) {
	e.scopedPriority(cx, scheduler.DefaultPriority(), f)
}

// ScopedPriority is Scoped with the scoped tasks spawned at priority.
func (e BackgroundExecutor) ScopedPriority(cx scheduler.Awaiter, priority Priority, f func(scope *Scope)) {
	e.scopedPriority(cx, priority, f)
}

func (e BackgroundExecutor) scopedPriority(cx scheduler.Awaiter, priority Priority, f func(scope *Scope)) {
	scope := newScope(e, priority)
	defer scope.Close()
	f(scope)
	spawned := make([]*Task[struct{}], 0, len(scope.futures))
	awaited := 0
	defer func() {
		for _, task := range spawned[awaited:] {
			task.Cancel()
		}
	}()
	for _, future := range scope.futures {
		spawned = append(spawned, e.SpawnWithPriority(scope.priority, future.run))
	}
	for i, task := range spawned {
		_, _ = task.Await(cx)
		awaited = i + 1
	}
}

// Now returns the current time on the scheduler's clock, so tests can use fake timers.
func (e BackgroundExecutor) Now() scheduler.Instant {
	return e.inner.Scheduler().Clock().Now()
}

// Timer returns a task that completes after duration; it may take longer depending on other
// concurrent tasks.
func (e BackgroundExecutor) Timer(duration time.Duration) *Task[struct{}] {
	if duration == 0 {
		return scheduler.ReadyTask(struct{}{})
	}
	timer := e.inner.Scheduler().Timer(duration)
	return e.spawnWithPriorityAt(scheduler.CallerLocation(1), scheduler.DefaultPriority(), func(s *scheduler.Scope) (struct{}, error) {
		timer.Await(s)
		return struct{}{}, nil
	})
}

func (e BackgroundExecutor) testDispatcher() *TestDispatcher {
	d := e.dispatcher.AsTest()
	if d == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return d
}

// SimulateRandomDelay returns a future that runs a random number of other tasks, in tests.
func (e BackgroundExecutor) SimulateRandomDelay() *scheduler.Yield {
	return e.testDispatcher().SimulateRandomDelay()
}

// AdvanceClock moves time forward in tests, making timers ready without running tasks.
func (e BackgroundExecutor) AdvanceClock(duration time.Duration) {
	e.testDispatcher().AdvanceClock(duration)
}

// Tick runs one task, in tests.
func (e BackgroundExecutor) Tick() bool {
	return e.testDispatcher().Scheduler().Tick()
}

// RunUntilParked runs tasks until the scheduler would park, in tests.
func (e BackgroundExecutor) RunUntilParked() {
	e.testDispatcher().Scheduler().Run()
}

// AllowParking keeps RunUntilParked from panicking on outstanding tasks, in tests.
func (e BackgroundExecutor) AllowParking() {
	e.testDispatcher().Scheduler().AllowParking()

	if v, ok := os.LookupEnv("GPUI_RUN_UNTIL_PARKED_LOG"); ok && v == "1" {
		slog.Warn("[gpui::executor] allow_parking: enabled")
	}
}

// SetBlockOnTicks sets the range of ticks to run before block_on times out, in tests.
func (e BackgroundExecutor) SetBlockOnTicks(r scheduler.TickRange) {
	e.testDispatcher().Scheduler().SetTimeoutTicks(r)
}

// ForbidParking undoes AllowParking.
func (e BackgroundExecutor) ForbidParking() {
	e.testDispatcher().Scheduler().ForbidParking()
}

// Rng returns the dispatcher's rng, in tests.
func (e BackgroundExecutor) Rng() *scheduler.SharedRng {
	return e.testDispatcher().Scheduler().Rng()
}

// NumCPUs returns how many CPUs are available to the dispatcher.
func (e BackgroundExecutor) NumCPUs() int {
	if test := e.dispatcher.AsTest(); test != nil {
		if n, ok := test.NumCPUsOverride(); ok {
			return n
		}
		return 4
	}
	return runtime.NumCPU()
}

// SetNumCPUs overrides NumCPUs in tests; it panics on a non-test executor.
func (e BackgroundExecutor) SetNumCPUs(count int) {
	test := e.dispatcher.AsTest()
	if test == nil {
		panic("set_num_cpus can only be called on a test executor")
	}
	test.SetNumCPUs(count)
}

// IsMainThread reports whether the caller is on the main thread.
func (e BackgroundExecutor) IsMainThread() bool {
	return e.dispatcher.IsMainThread()
}

// IsTest reports whether the executor runs on the deterministic test scheduler.
func (e BackgroundExecutor) IsTest() bool {
	return e.inner.IsTest()
}

// Dispatcher returns the platform dispatcher.
func (e BackgroundExecutor) Dispatcher() PlatformDispatcher {
	return e.dispatcher
}

// ToSend returns a handle that submits tasks to this executor from any goroutine.
func (e ForegroundExecutor) ToSend() SendForegroundExecutor {
	return SendForegroundExecutor{inner: e.inner.ToSend()}
}

// NewForegroundExecutor returns a foreground executor over dispatcher: a session of the
// dispatcher's TestScheduler for a TestDispatcher, a PlatformScheduler session otherwise.
func NewForegroundExecutor(dispatcher PlatformDispatcher) ForegroundExecutor {
	testDispatcher := dispatcher.AsTest()
	if testDispatcher == nil {
		platformScheduler := NewPlatformScheduler(dispatcher)
		return ForegroundExecutor{
			inner:      platformScheduler.ForegroundExecutor(),
			dispatcher: dispatcher,
		}
	}
	s := testDispatcher.Scheduler()
	sessionID := testDispatcher.SessionID()
	schedulerForDispatch := weak.Make(s)
	inner := scheduler.NewLocalExecutor(sessionID, s, func(runnable *scheduler.Runnable) {
		if strong := schedulerForDispatch.Value(); strong != nil {
			strong.ScheduleLocal(sessionID, runnable)
		}
	})
	return ForegroundExecutor{inner: inner, dispatcher: dispatcher}
}

// Spawn enqueues body to run on the main thread.
func (e ForegroundExecutor) Spawn[R any](body func(s *scheduler.Scope) (R, error)) *Task[R] {
	return e.inner.SpawnAt(scheduler.CallerLocation(1), body)
}

// SpawnWithPriority enqueues body on the main thread; foreground tasks ignore priority and run
// in order.
func (e ForegroundExecutor) SpawnWithPriority[R any](priority Priority, body func(s *scheduler.Scope) (R, error)) *Task[R] {
	return e.inner.SpawnAt(scheduler.CallerLocation(1), body)
}

// SpawnWhenIdle enqueues body to run on the main thread during platform idle time where the
// platform supports it; with a timeout, a poll still waiting after that long runs as ordinary
// main-thread work. Elsewhere it runs at low priority and ignores timeout.
func (e ForegroundExecutor) SpawnWhenIdle[R any](timeout Option[time.Duration], body func(s *scheduler.Scope) (R, error)) *Task[R] {
	dispatcher := e.dispatcher
	return e.inner.SpawnWithDispatch(body, func(runnable *scheduler.Runnable) {
		dispatcher.DispatchOnMainThreadWhenIdle(runnable, timeout)
	})
}

// IdleTimeRemaining returns the time left in the current idle slice, when called from a task
// spawned with SpawnWhenIdle on a platform that meters idle time.
func (e ForegroundExecutor) IdleTimeRemaining() (time.Duration, bool) {
	return e.dispatcher.IdleTimeRemaining()
}

// BlockTest runs future to completion for the test harness, letting the test session's
// foreground tasks progress meanwhile.
func (e ForegroundExecutor) BlockTest[R any](future scheduler.Future[R]) R {
	s := e.inner.Scheduler()

	var output R
	done := false
	wrapper := scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
		v, ok := future.Poll(cx)
		if ok {
			output = v
			done = true
		}
		return struct{}{}, ok
	})

	s.Block(nil, wrapper, nil)

	if !done {
		panic("block_test future did not complete")
	}
	return output
}

// BlockOn blocks the current goroutine until future resolves; prefer BlockWithTimeout.
func (e ForegroundExecutor) BlockOn[R any](future scheduler.Future[R]) R {
	return e.inner.BlockOn(future)
}

// BlockWithTimeout blocks until future resolves or duration elapses. On timeout ok is false and
// the caller keeps future, which it may keep polling.
func (e ForegroundExecutor) BlockWithTimeout[R any](duration time.Duration, future scheduler.Future[R]) (R, bool) {
	return e.inner.BlockWithTimeout(duration, future)
}

// Dispatcher returns the platform dispatcher.
func (e ForegroundExecutor) Dispatcher() PlatformDispatcher {
	return e.dispatcher
}

// SchedulerExecutor returns the underlying scheduler.LocalExecutor.
func (e ForegroundExecutor) SchedulerExecutor() SchedulerLocalExecutor {
	return e.inner
}

const (
	scopeFutureIdle int32 = iota
	scopeFutureStarted
	scopeFutureClaimed
)

type scopeFuture struct {
	f     func(s *scheduler.Scope)
	tx    *scheduler.ChanSender[struct{}]
	state atomic.Int32
}

func (f *scopeFuture) run(s *scheduler.Scope) (struct{}, error) {
	if !f.state.CompareAndSwap(scopeFutureIdle, scopeFutureStarted) {
		return struct{}{}, nil
	}
	defer f.tx.Close()
	f.f(s)
	return struct{}{}, nil
}

func (f *scopeFuture) claim() {
	if f.state.CompareAndSwap(scopeFutureIdle, scopeFutureClaimed) {
		f.tx.Close()
	}
}

// Scope is a set of tasks that are enqueued and waited on together; see BackgroundExecutor.Scoped.
// Close is its drop: it blocks until every spawned future resolved.
type Scope struct {
	executor BackgroundExecutor
	priority Priority
	futures  []*scopeFuture
	tx       *scheduler.ChanSender[struct{}]
	rx       *scheduler.ChanReceiver[struct{}]
	closed   bool
}

func newScope(executor BackgroundExecutor, priority Priority) *Scope {
	tx, rx := scheduler.Chan[struct{}](1)
	return &Scope{
		executor: executor,
		priority: priority,
		tx:       tx,
		rx:       rx,
	}
}

// NumCPUs returns how many CPUs are available to the dispatcher.
func (s *Scope) NumCPUs() int {
	return s.executor.NumCPUs()
}

// Spawn adds f to the scope.
func (s *Scope) Spawn(f func(s *scheduler.Scope)) {
	if s.closed {
		panic("called `Option::unwrap()` on a `None` value")
	}
	tx := s.tx.Clone()
	s.futures = append(s.futures, &scopeFuture{f: f, tx: tx})
}

// Close drops the scope: it waits until the futures spawned into it resolved. It is idempotent.
func (s *Scope) Close() {
	if s.closed {
		return
	}
	s.closed = true
	s.tx.Close()
	for _, f := range s.futures {
		f.claim()
	}
	s.futures = nil

	rx := s.rx
	future := scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
		_, ok := rx.Poll(cx)
		return struct{}{}, ok
	})
	s.executor.inner.Scheduler().Block(nil, future, nil)
}
