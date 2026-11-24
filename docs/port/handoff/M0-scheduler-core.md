# M0-scheduler-core handoff (scheduler contract as built)

For M0-scheduler-test, M0-contracts and M0-executor. Departures from DESIGN 6.3 are marked (6.3).

## Futures and coroutines
- `Future[T]{ Poll(cx *Context) (T, bool) }`, `Context` wraps a `Waker{Wake; WakeByRef; Clone() Waker; Drop}`. Primitives store `cx.Waker().Clone()` and later `Wake()` or `Drop()` it, so TestScheduler's TracingWaker can track clones/drops (pending traces, wrong-thread detection).
- `Async(func(s *Scope) T) Future[T]` is an async block: lazy `iter.Pull` coroutine, created at first poll. Cancel = owner drop: `stop()`, body unwinds via a private panic recovered in the seq func; defers run.
- `Await(cx Awaiter, f) T` owns f (drops it via `Cancel()` if the body unwinds first). `Select(cx, Case(f, then)...) int` (biased), `Join(cx, fs...) []T`, `Join2`, `Unowned(f)` (Rust `&mut fut`), `PollFn`, `BlockOn(f) T` (futures::executor::block_on, plain goroutines only), `Yield` (test_scheduler.rs `Yield(usize)`: `y := Yield(n); Await(cx, &y)`).
- `Awaiter` has only the unexported `scope()`; gpui types (AsyncApp, ...) satisfy it by embedding `*scheduler.Scope` (tested in awaiter_external_test.go).
- Channels: `Oneshot[T]() (*OneshotSender, *OneshotReceiver)`, `Chan[T](capacity) (*ChanSender, *ChanReceiver)`; errors `ErrCanceled`, `ErrClosed`, `ErrFull`. `Close()` is the drop of every half.

## Tasks
- Spawn bodies are `func(s *Scope) (T, error)`; `*Task[T]` is a `Future[Result[T]]`, `Await(cx) (T, error)`, `Wait(cx) error`, `Cancel()` (drop; idempotent), `Detach()`, `IsReady()`, `Fallible()`.
- `ReadyTask(v)` (STAGES name; parity row Go column changed from `TaskReady`), plus `ReadyTaskErr[T](err)`. `TaskFromAsyncTask(*AsyncTask[Result[T]])`. `Task<Box<dyn Any>>::downcast` is `TaskAnyDowncast[T](*Task[any])` (N7). `FallibleTask[T]` resolves to `ErrCanceled` instead of None.
- Awaiting a task closed without output panics "Task polled after completion" (async_task).
- async_task port: `SpawnAsyncTask(meta, future, schedule func(*Runnable)) (*Runnable, *AsyncTask[T])` (PlatformScheduler's timer uses it). `Runnable`: `Run() bool`, `Schedule()`, `Close()` (= dropping the runnable: closes the task, drops the future), `Metadata() *RunnableMeta`, `Waker()`. A schedule function that cannot deliver must call `Close()`.
- Cancel never calls `stop()`: it sets CLOSED and schedules an idle task once; `Run` drops the future. `stop()` only runs inside `Run`, `Runnable.Close` or an owner's `Cancel` of a nested future.

## Scheduler interface
- `Block(sessionID *SessionID, future Future[struct{}], timeout *time.Duration) bool` (6.3 had `poll func() bool`, which cannot carry the scheduler's waker).
- `ScheduleLocal`, `ScheduleBackgroundWithPriority`, `ScheduleBackground` (no default body in Go: implement as `ScheduleBackgroundWithPriority(r, DefaultPriority())`), `SpawnRealtime(func())`, `Timer(d) *Timer`, `Clock() Clock`, `SpawnDedicated(func(LocalExecutor, *Scope) (any, error)) *Task[any]`, `AsTest() *TestScheduler`.
- `Clock{ UTCNow() time.Time; Now() Instant }` (parity rows renamed from UtcNow); `TestClock.SetUTCNow`.
- `Priority` keeps upstream's repr order, so its zero value is `PriorityRealtimeAudio`; use `DefaultPriority()` wherever upstream writes `Priority::default()`.
- Spawn locations: every spawn method captures its caller; wrappers that are `#[track_caller]` upstream pass theirs with the `...At(location, ...)` variants and `CallerLocation(skip)`.
- `Timer` wraps `*OneshotReceiver[struct{}]`; a scheduler fires it by `Send` or `Close` on the sender (TestScheduler: closing elapsed `ScheduledTimer` senders = dropping them).

## For M0-scheduler-test
- Delete `TestScheduler` from `scheduler/stubs.go` (delete the file) and set `testSchedulerCreatedFlag.Store(true)` in `NewTestScheduler`.
- `schedule_local_dispatch` holds `weak.Pointer[TestScheduler]`; on failed upgrade call `runnable.Close()`. `DrainTasks` closes queued runnables and timer senders. `BackgroundExecutor` already holds its scheduler weakly (reflect + weak.Pointer).
- Executors are values (`LocalExecutor`, `BackgroundExecutor`, `SendLocalExecutor`); `DedicatedExecutor` is a pointer with `Close()` as its drop.
- tests.rs declares its own `struct Yield { polls }`; in Go it collides with the package-level `Yield` in test_scheduler.go, so the port must rename it (e.g. `pollsYield`).
- Package `scheduler` test files already declare `fifoScheduler`, `fifoTimer`, `newFIFOScheduler`, `newThreadedScheduler`, `awaitTask`, `goroutineBaseline`, `parkedProbe`, `itoa` (executor_test.go, fifo_scheduler_test.go, future_test.go) and `inlineScheduler`, `asyncContext` in package `scheduler_test`.
- The brief's "ForegroundExecutor" is upstream's `LocalExecutor` at this commit; gpui's `ForegroundExecutor` belongs to M0-executor.
