# Parity: scheduler

Scope: scheduler. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## scheduler/src/clock.rs

Go target: `scheduler/clock.go` (package `scheduler`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Clock` | trait | 7 |  | `Clock` | done |
| `Clock::utc_now` | trait-method | 8 |  | `Clock.UTCNow` | dev:S-NAME |
| `Clock::now` | trait-method | 9 |  | `Clock.Now` | done |
| `TestClock` | struct | 12 |  | `TestClock` | done |
| `TestClock::new` | assoc-fn | 20 |  | `NewTestClock` | done |
| `TestClock::set_utc_now` | method | 29 |  | `(TestClock).SetUTCNow` | dev:S-NAME |
| `TestClock::advance` | method | 34 |  | `(TestClock).Advance` | done |

## scheduler/src/executor.rs

Go target: `scheduler/executor.go` (package `scheduler`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LocalExecutor` | struct | 22 |  | `LocalExecutor` | done |
| `LocalExecutor::new` | assoc-fn | 41 |  | `NewLocalExecutor` | done |
| `LocalExecutor::to_send` | method | 55 |  | `(LocalExecutor).ToSend` | done |
| `LocalExecutor::session_id` | method | 62 |  | `(LocalExecutor).SessionID` | done |
| `LocalExecutor::scheduler` | method | 66 |  | `(LocalExecutor).Scheduler` | done |
| `LocalExecutor::is_test` | method | 71 |  | `(LocalExecutor).IsTest` | done |
| `LocalExecutor::spawn` | method | 76 |  | `(LocalExecutor).Spawn` | done |
| `LocalExecutor::spawn_with_dispatch` | method | 100 |  | `(LocalExecutor).SpawnWithDispatch` | done |
| `LocalExecutor::block_on` | method | 123 | wasm | `(LocalExecutor).BlockOn` | done |
| `LocalExecutor::block_with_timeout` | method | 141 | wasm | `(LocalExecutor).BlockWithTimeout` | done |
| `LocalExecutor::timer` | method | 169 |  | `(LocalExecutor).Timer` | done |
| `LocalExecutor::now` | method | 173 |  | `(LocalExecutor).Now` | done |
| `LocalExecutor::spawn_dedicated` | method | 186 |  | `(LocalExecutor).SpawnDedicated` | done |
| `SendLocalExecutor` | struct | 208 |  | `SendLocalExecutor` | done |
| `SendLocalExecutor::spawn` | method | 216 |  | `(SendLocalExecutor).Spawn` | done |
| `BackgroundExecutor` | struct | 256 |  | `BackgroundExecutor` | done |
| `BackgroundExecutor::new` | assoc-fn | 261 |  | `NewBackgroundExecutor` | done |
| `BackgroundExecutor::spawn` | method | 266 |  | `(BackgroundExecutor).Spawn` | done |
| `BackgroundExecutor::spawn_with_priority` | method | 275 |  | `(BackgroundExecutor).SpawnWithPriority` | done |
| `BackgroundExecutor::spawn_realtime` | method | 294 |  | `(BackgroundExecutor).SpawnRealtime` | done |
| `BackgroundExecutor::timer` | method | 324 |  | `(BackgroundExecutor).Timer` | done |
| `BackgroundExecutor::now` | method | 328 |  | `(BackgroundExecutor).Now` | done |
| `BackgroundExecutor::scheduler` | method | 332 |  | `(BackgroundExecutor).Scheduler` | done |
| `BackgroundExecutor::is_test` | method | 337 |  | `(BackgroundExecutor).IsTest` | done |
| `BackgroundExecutor::spawn_dedicated` | method | 350 |  | `(BackgroundExecutor).SpawnDedicated` | done |
| `DedicatedExecutor` | struct | 385 |  | `DedicatedExecutor` | done |
| `DedicatedExecutor::new` | assoc-fn | 395 |  | `NewDedicatedExecutor` | done |
| `DedicatedExecutor::spawn` | method | 414 |  | `(DedicatedExecutor).Spawn` | done |
| `Task` | struct | 440 |  | `Task` | done |
| `Task::ready` | assoc-fn | 577 |  | `ReadyTask` | dev:S-NAME |
| `Task::from_async_task` | assoc-fn | 596 |  | `TaskFromAsyncTask` | done |
| `Task::is_ready` | method | 600 |  | `(Task).IsReady` | done |
| `Task::detach` | method | 610 |  | `(Task).Detach` | done |
| `Task::fallible` | method | 620 |  | `(Task).Fallible` | done |
| `Task::downcast` | method | 642 |  | `TaskAnyDowncast` | dev:S-NAME |
| `FallibleTask` | struct | 665 |  | `FallibleTask` | done |
| `FallibleTask::ready` | assoc-fn | 686 |  | `FallibleTaskReady` | done |
| `FallibleTask::detach` | method | 691 |  | `(FallibleTask).Detach` | done |

## scheduler/src/scheduler.rs

Go target: `scheduler/scheduler.go` (package `scheduler`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `test_scheduler_created` | fn | 21 |  | `TestSchedulerCreated` | done |
| `Priority` | enum | 44 |  | `Priority` | done |
| `Priority::weight` | method | 61 |  | `(Priority).Weight` | done |
| `SpawnTime` | struct | 73 |  | `SpawnTime` | done |
| `RunnableMeta` | struct | 77 |  | `RunnableMeta` | done |
| `RunnableMeta::new_with_callers_location` | assoc-fn | 86 |  | `NewRunnableMetaWithCallersLocation` | done |
| `Scheduler` | trait | 94 |  | `Scheduler` | done |
| `Scheduler::block` | trait-method | 101 | wasm | `Scheduler.Block` | done |
| `Scheduler::schedule_local` | trait-method | 112 |  | `Scheduler.ScheduleLocal` | done |
| `Scheduler::schedule_background_with_priority` | trait-method | 115 |  | `Scheduler.ScheduleBackgroundWithPriority` | done |
| `Scheduler::spawn_realtime` | trait-method | 122 |  | `Scheduler.SpawnRealtime` | done |
| `Scheduler::schedule_background` | trait-method | 125 |  | `Scheduler.ScheduleBackground` | done |
| `Scheduler::timer` | trait-method | 130 |  | `Scheduler.Timer` | done |
| `Scheduler::clock` | trait-method | 131 |  | `Scheduler.Clock` | done |
| `Scheduler::spawn_dedicated` | trait-method | 146 |  | `Scheduler.SpawnDedicated` | done |
| `Scheduler::as_test` | trait-method | 158 |  | `Scheduler.AsTest` | done |
| `spawn_dedicated_thread` | fn | 187 |  | `SpawnDedicatedThread` | done |
| `SessionId` | struct | 241 |  | `SessionID` | done |
| `SessionId::new` | assoc-fn | 244 |  | `NewSessionID` | done |
| `Timer` | struct | 249 |  | `Timer` | done |
| `Timer::new` | assoc-fn | 252 |  | `NewTimer` | done |

## scheduler/src/test_scheduler.rs

Go target: `scheduler/test_scheduler.go` (package `scheduler`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestScheduler` | struct | 34 |  | `TestScheduler` | dev:S-GOID |
| `TestScheduler::once` | assoc-fn | 43 |  | `TestSchedulerOnce` | done |
| `TestScheduler::many` | assoc-fn | 48 |  | `TestSchedulerMany` | done |
| `TestScheduler::new` | assoc-fn | 161 |  | `NewTestScheduler` | done |
| `TestScheduler::end_test` | method | 187 |  | `(TestScheduler).EndTest` | done |
| `TestScheduler::clock` | method | 200 |  | `(TestScheduler).Clock` | done |
| `TestScheduler::rng` | method | 204 |  | `(TestScheduler).Rng` | done |
| `TestScheduler::set_timeout_ticks` | method | 208 |  | `(TestScheduler).SetTimeoutTicks` | dev:S-RANGE |
| `TestScheduler::allow_parking` | method | 212 |  | `(TestScheduler).AllowParking` | done |
| `TestScheduler::forbid_parking` | method | 218 |  | `(TestScheduler).ForbidParking` | done |
| `TestScheduler::parking_allowed` | method | 222 |  | `(TestScheduler).ParkingAllowed` | done |
| `TestScheduler::is_main_thread` | method | 226 |  | `(TestScheduler).IsMainThread` | done |
| `TestScheduler::allocate_session_id` | method | 230 |  | `(TestScheduler).AllocateSessionID` | done |
| `TestScheduler::foreground` | method | 237 |  | `(TestScheduler).Foreground` | done |
| `TestScheduler::background` | method | 247 |  | `(TestScheduler).Background` | done |
| `TestScheduler::yield_random` | method | 251 |  | `(TestScheduler).YieldRandom` | done |
| `TestScheduler::run` | method | 260 |  | `(TestScheduler).Run` | done |
| `TestScheduler::run_with_clock_advancement` | method | 266 |  | `(TestScheduler).RunWithClockAdvancement` | done |
| `TestScheduler::tick` | method | 276 |  | `(TestScheduler).Tick` | done |
| `TestScheduler::tick_background_only` | method | 282 |  | `(TestScheduler).TickBackgroundOnly` | done |
| `TestScheduler::has_pending_tasks` | method | 287 |  | `(TestScheduler).HasPendingTasks` | done |
| `TestScheduler::pending_task_counts` | method | 294 |  | `(TestScheduler).PendingTaskCounts` | done |
| `TestScheduler::drain_tasks` | method | 425 |  | `(TestScheduler).DrainTasks` | done |
| `TestScheduler::advance_clock_to_next_timer` | method | 442 |  | `(TestScheduler).AdvanceClockToNextTimer` | done |
| `TestScheduler::advance_clock` | method | 451 |  | `(TestScheduler).AdvanceClock` | done |
| `TestSchedulerConfig` | struct | 743 |  | `TestSchedulerConfig` | done |
| `TestSchedulerConfig::with_seed` | assoc-fn | 752 |  | `TestSchedulerConfigWithSeed` | done |
| `Yield` | struct | 899 |  | `Yield` | done |
| `SharedRng` | struct | 904 |  | `SharedRng` | dev:D-RNG |
| `SharedRng::lock` | method | 909 |  | `(SharedRng).Lock` | dev:S-RANGE |
| `SharedRng::random_range` | method | 914 |  | `RandomRange` | dev:S-NAME |
| `SharedRng::random_bool` | method | 923 |  | `(SharedRng).RandomBool` | done |
| `SharedRng::random` | method | 928 |  | `Random` | dev:S-NAME |
| `SharedRng::random_ratio` | method | 936 |  | `(SharedRng).RandomRatio` | done |

## scheduler/src/tests.rs

Go target: `scheduler/scheduler_test.go` (package `scheduler`), tests in `scheduler_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_foreground_executor_spawn` | test | 20 |  | `TestScheduler_ForegroundExecutorSpawn` | done |
| `test_send_local_executor_worker_submission_and_wakeup` | test | 29 |  | `TestScheduler_SendLocalExecutorWorkerSubmissionAndWakeup` | done |
| `test_send_local_executor_retains_scheduler` | test | 86 |  | `TestScheduler_SendLocalExecutorRetainsScheduler` | done |
| `test_background_executor_spawn` | test | 112 |  | `TestScheduler_BackgroundExecutorSpawn` | done |
| `test_dedicated_executor_spawn` | test | 121 |  | `TestScheduler_DedicatedExecutorSpawn` | done |
| `test_scheduler_drops_with_stalled_detached_foreground_task` | test | 130 |  | `TestScheduler_SchedulerDropsWithStalledDetachedForegroundTask` | done |
| `test_scheduler_drops_with_stalled_detached_background_task` | test | 149 |  | `TestScheduler_SchedulerDropsWithStalledDetachedBackgroundTask` | done |
| `test_scheduler_drops_with_never_polled_dedicated_task` | test | 171 |  | `TestScheduler_SchedulerDropsWithNeverPolledDedicatedTask` | done |
| `test_foreground_ordering` | test | 185 |  | `TestScheduler_ForegroundOrdering` | done |
| `test_timer_ordering` | test | 249 |  | `TestScheduler_TimerOrdering` | done |
| `test_foreground_task_can_hold_mut_borrow_across_await` | test | 279 |  | `TestScheduler_ForegroundTaskCanHoldMutBorrowAcrossAwait` | done |
| `test_send_from_bg_to_fg` | test | 297 |  | `TestScheduler_SendFromBgToFg` | done |
| `test_randomize_order` | test | 317 |  | `TestScheduler_RandomizeOrder` | done |
| `test_block` | test | 388 |  | `TestScheduler_Block` | done |
| `test_parking_panics` | test | 407 |  | `TestScheduler_ParkingPanics` | done |
| `test_block_with_parking` | test | 420 |  | `TestScheduler_BlockWithParking` | done |
| `test_helper_methods` | test | 442 |  | `TestScheduler_HelperMethods` | done |
| `test_many_with_arbitrary_seed` | test | 459 |  | `TestScheduler_ManyWithArbitrarySeed` | done |
| `test_block_with_timeout` | test | 484 |  | `TestScheduler_BlockWithTimeout` | done |
| `test_block_does_not_progress_same_session_foreground` | test | 570 |  | `TestScheduler_BlockDoesNotProgressSameSessionForeground` | done |
| `test_nondeterministic_wake_detection` | test | 614 |  | `TestScheduler_NondeterministicWakeDetection` | done |
| `test_nondeterministic_wake_allowed_with_parking` | test | 674 |  | `TestScheduler_NondeterministicWakeAllowedWithParking` | done |
| `test_nondeterministic_waker_drop_detection` | test | 721 |  | `TestScheduler_NondeterministicWakerDropDetection` | done |
| `test_background_priority_scheduling` | test | 781 |  | `TestScheduler_BackgroundPriorityScheduling` | done |
| `test_spawn_dedicated_basic_round_trip` | test | 842 |  | `TestScheduler_SpawnDedicatedBasicRoundTrip` | done |
| `test_spawn_dedicated_not_send_future` | test | 853 |  | `TestScheduler_SpawnDedicatedNotSendFuture` | done |
| `test_spawn_dedicated_send_closure_captures` | test | 872 |  | `TestScheduler_SpawnDedicatedSendClosureCaptures` | done |
| `test_spawn_dedicated_inner_spawn_local` | test | 896 |  | `TestScheduler_SpawnDedicatedInnerSpawnLocal` | done |
| `test_spawn_dedicated_determinism_under_many` | test | 917 |  | `TestScheduler_SpawnDedicatedDeterminismUnderMany` | done |
| `test_spawn_dedicated_dropping_task_cancels_future` | test | 992 |  | `TestScheduler_SpawnDedicatedDroppingTaskCancelsFuture` | done |
| `test_spawn_dedicated_detached_child_runs_after_root_completes` | test | 1033 |  | `TestScheduler_SpawnDedicatedDetachedChildRunsAfterRootCompletes` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| S-NAME | Go names fixed by DESIGN 6.3 / STAGES or rule N7 instead of the scanned name: `utc_now` -> `UTCNow`, `set_utc_now` -> `SetUTCNow`, `Task::ready` -> `ReadyTask`, `Task<Box<dyn Any>>::downcast` -> `TaskAnyDowncast` | `scheduler/src/clock.rs:8`, `:29`, `scheduler/src/executor.rs:577`, `:642` | Same behaviour; DESIGN 6.3 and STAGES spell these names, and a method on one instantiation is a free function in Go |
| S-THREADCHECK | Local tasks are not checked for being polled or dropped on the spawning thread; upstream panics with "local task polled by a thread that didn't spawn it" | `scheduler/src/executor.rs:780-859` | Go exposes no goroutine or thread identity; only the session's dispatch destination ever runs a local runnable |
| S-DEDICATEDEXIT | The goroutine of a `spawn_dedicated_thread` session exits after the garbage collector finds its dispatch handle unreachable, not at the last `Arc` drop | `scheduler/src/scheduler.rs:187-226` | Go has no reference counts on the dispatch closure copies; tasks themselves are never stopped by GC |
| S-GOID | TestScheduler's wrong-thread check compares goroutine ids (parsed from runtime.Stack) instead of thread ids; goroutines created by iter.Pull count as the scheduler's goroutine because they only run while it polls them; the message names goroutines | `scheduler/src/test_scheduler.rs:564-587` | Go has no thread identity, and every coroutine body runs on its own goroutine |
| S-RANGE | `RangeInclusive<usize>` is `TickRange{Start, End}`; `SharedRng::lock` returns the `*rand.Rand` and an unlock func; generic `random_range` / `random` are free functions `RandomRange(r, lo, hi)` (half-open) and `Random[T](r)` | `scheduler/src/test_scheduler.rs:208`, `:909-933` | Go has no range types or guards; methods keep the same draws |
| D-SEC-TASKPANIC | A panic inside a background task body no longer kills the process: `BackgroundExecutor` (including `SpawnRealtime`, `SpawnDedicated` and `DedicatedExecutor.Spawn`) wraps the body in `recoverBackgroundPanic`, which prints the panic and stack to stderr and resolves the task with `*PanicError{Value, Stack}`; `Scoped` surfaces its tasks' `PanicError`s through its returned error (D-SEC-R2-13); cancellation unwinds are re-panicked and foreground task panics still propagate (SEC-037, R2-13) | `scheduler/src/executor.rs:266` | An unrecovered Go panic on any goroutine exits the process, so one malformed input reaching a background task would crash the app |
