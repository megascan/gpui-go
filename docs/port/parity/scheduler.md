# Parity: scheduler

Scope: scheduler. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## scheduler/src/clock.rs

Go target: `scheduler/clock.go` (package `scheduler`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Clock` | trait | 7 |  | `Clock` | todo |
| `Clock::utc_now` | trait-method | 8 |  | `Clock.UtcNow` | todo |
| `Clock::now` | trait-method | 9 |  | `Clock.Now` | todo |
| `TestClock` | struct | 12 |  | `TestClock` | todo |
| `TestClock::new` | assoc-fn | 20 |  | `NewTestClock` | todo |
| `TestClock::set_utc_now` | method | 29 |  | `(TestClock).SetUtcNow` | todo |
| `TestClock::advance` | method | 34 |  | `(TestClock).Advance` | todo |

## scheduler/src/executor.rs

Go target: `scheduler/executor.go` (package `scheduler`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LocalExecutor` | struct | 22 |  | `LocalExecutor` | todo |
| `LocalExecutor::new` | assoc-fn | 41 |  | `NewLocalExecutor` | todo |
| `LocalExecutor::to_send` | method | 55 |  | `(LocalExecutor).ToSend` | todo |
| `LocalExecutor::session_id` | method | 62 |  | `(LocalExecutor).SessionID` | todo |
| `LocalExecutor::scheduler` | method | 66 |  | `(LocalExecutor).Scheduler` | todo |
| `LocalExecutor::is_test` | method | 71 |  | `(LocalExecutor).IsTest` | todo |
| `LocalExecutor::spawn` | method | 76 |  | `(LocalExecutor).Spawn` | todo |
| `LocalExecutor::spawn_with_dispatch` | method | 100 |  | `(LocalExecutor).SpawnWithDispatch` | todo |
| `LocalExecutor::block_on` | method | 123 | wasm | `(LocalExecutor).BlockOn` | todo |
| `LocalExecutor::block_with_timeout` | method | 141 | wasm | `(LocalExecutor).BlockWithTimeout` | todo |
| `LocalExecutor::timer` | method | 169 |  | `(LocalExecutor).Timer` | todo |
| `LocalExecutor::now` | method | 173 |  | `(LocalExecutor).Now` | todo |
| `LocalExecutor::spawn_dedicated` | method | 186 |  | `(LocalExecutor).SpawnDedicated` | todo |
| `SendLocalExecutor` | struct | 208 |  | `SendLocalExecutor` | todo |
| `SendLocalExecutor::spawn` | method | 216 |  | `(SendLocalExecutor).Spawn` | todo |
| `BackgroundExecutor` | struct | 256 |  | `BackgroundExecutor` | todo |
| `BackgroundExecutor::new` | assoc-fn | 261 |  | `NewBackgroundExecutor` | todo |
| `BackgroundExecutor::spawn` | method | 266 |  | `(BackgroundExecutor).Spawn` | todo |
| `BackgroundExecutor::spawn_with_priority` | method | 275 |  | `(BackgroundExecutor).SpawnWithPriority` | todo |
| `BackgroundExecutor::spawn_realtime` | method | 294 |  | `(BackgroundExecutor).SpawnRealtime` | todo |
| `BackgroundExecutor::timer` | method | 324 |  | `(BackgroundExecutor).Timer` | todo |
| `BackgroundExecutor::now` | method | 328 |  | `(BackgroundExecutor).Now` | todo |
| `BackgroundExecutor::scheduler` | method | 332 |  | `(BackgroundExecutor).Scheduler` | todo |
| `BackgroundExecutor::is_test` | method | 337 |  | `(BackgroundExecutor).IsTest` | todo |
| `BackgroundExecutor::spawn_dedicated` | method | 350 |  | `(BackgroundExecutor).SpawnDedicated` | todo |
| `DedicatedExecutor` | struct | 385 |  | `DedicatedExecutor` | todo |
| `DedicatedExecutor::new` | assoc-fn | 395 |  | `NewDedicatedExecutor` | todo |
| `DedicatedExecutor::spawn` | method | 414 |  | `(DedicatedExecutor).Spawn` | todo |
| `Task` | struct | 440 |  | `Task` | todo |
| `Task::ready` | assoc-fn | 577 |  | `TaskReady` | todo |
| `Task::from_async_task` | assoc-fn | 596 |  | `TaskFromAsyncTask` | todo |
| `Task::is_ready` | method | 600 |  | `(Task).IsReady` | todo |
| `Task::detach` | method | 610 |  | `(Task).Detach` | todo |
| `Task::fallible` | method | 620 |  | `(Task).Fallible` | todo |
| `Task::downcast` | method | 642 |  | `(Task).Downcast` | todo |
| `FallibleTask` | struct | 665 |  | `FallibleTask` | todo |
| `FallibleTask::ready` | assoc-fn | 686 |  | `FallibleTaskReady` | todo |
| `FallibleTask::detach` | method | 691 |  | `(FallibleTask).Detach` | todo |

## scheduler/src/scheduler.rs

Go target: `scheduler/scheduler.go` (package `scheduler`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `test_scheduler_created` | fn | 21 |  | `TestSchedulerCreated` | todo |
| `Priority` | enum | 44 |  | `Priority` | todo |
| `Priority::weight` | method | 61 |  | `(Priority).Weight` | todo |
| `SpawnTime` | struct | 73 |  | `SpawnTime` | todo |
| `RunnableMeta` | struct | 77 |  | `RunnableMeta` | todo |
| `RunnableMeta::new_with_callers_location` | assoc-fn | 86 |  | `NewRunnableMetaWithCallersLocation` | todo |
| `Scheduler` | trait | 94 |  | `Scheduler` | todo |
| `Scheduler::block` | trait-method | 101 | wasm | `Scheduler.Block` | todo |
| `Scheduler::schedule_local` | trait-method | 112 |  | `Scheduler.ScheduleLocal` | todo |
| `Scheduler::schedule_background_with_priority` | trait-method | 115 |  | `Scheduler.ScheduleBackgroundWithPriority` | todo |
| `Scheduler::spawn_realtime` | trait-method | 122 |  | `Scheduler.SpawnRealtime` | todo |
| `Scheduler::schedule_background` | trait-method | 125 |  | `Scheduler.ScheduleBackground` | todo |
| `Scheduler::timer` | trait-method | 130 |  | `Scheduler.Timer` | todo |
| `Scheduler::clock` | trait-method | 131 |  | `Scheduler.Clock` | todo |
| `Scheduler::spawn_dedicated` | trait-method | 146 |  | `Scheduler.SpawnDedicated` | todo |
| `Scheduler::as_test` | trait-method | 158 |  | `Scheduler.AsTest` | todo |
| `spawn_dedicated_thread` | fn | 187 |  | `SpawnDedicatedThread` | todo |
| `SessionId` | struct | 241 |  | `SessionID` | todo |
| `SessionId::new` | assoc-fn | 244 |  | `NewSessionID` | todo |
| `Timer` | struct | 249 |  | `Timer` | todo |
| `Timer::new` | assoc-fn | 252 |  | `NewTimer` | todo |

## scheduler/src/test_scheduler.rs

Go target: `scheduler/test_scheduler.go` (package `scheduler`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestScheduler` | struct | 34 |  | `TestScheduler` | todo |
| `TestScheduler::once` | assoc-fn | 43 |  | `TestSchedulerOnce` | todo |
| `TestScheduler::many` | assoc-fn | 48 |  | `TestSchedulerMany` | todo |
| `TestScheduler::new` | assoc-fn | 161 |  | `NewTestScheduler` | todo |
| `TestScheduler::end_test` | method | 187 |  | `(TestScheduler).EndTest` | todo |
| `TestScheduler::clock` | method | 200 |  | `(TestScheduler).Clock` | todo |
| `TestScheduler::rng` | method | 204 |  | `(TestScheduler).Rng` | todo |
| `TestScheduler::set_timeout_ticks` | method | 208 |  | `(TestScheduler).SetTimeoutTicks` | todo |
| `TestScheduler::allow_parking` | method | 212 |  | `(TestScheduler).AllowParking` | todo |
| `TestScheduler::forbid_parking` | method | 218 |  | `(TestScheduler).ForbidParking` | todo |
| `TestScheduler::parking_allowed` | method | 222 |  | `(TestScheduler).ParkingAllowed` | todo |
| `TestScheduler::is_main_thread` | method | 226 |  | `(TestScheduler).IsMainThread` | todo |
| `TestScheduler::allocate_session_id` | method | 230 |  | `(TestScheduler).AllocateSessionID` | todo |
| `TestScheduler::foreground` | method | 237 |  | `(TestScheduler).Foreground` | todo |
| `TestScheduler::background` | method | 247 |  | `(TestScheduler).Background` | todo |
| `TestScheduler::yield_random` | method | 251 |  | `(TestScheduler).YieldRandom` | todo |
| `TestScheduler::run` | method | 260 |  | `(TestScheduler).Run` | todo |
| `TestScheduler::run_with_clock_advancement` | method | 266 |  | `(TestScheduler).RunWithClockAdvancement` | todo |
| `TestScheduler::tick` | method | 276 |  | `(TestScheduler).Tick` | todo |
| `TestScheduler::tick_background_only` | method | 282 |  | `(TestScheduler).TickBackgroundOnly` | todo |
| `TestScheduler::has_pending_tasks` | method | 287 |  | `(TestScheduler).HasPendingTasks` | todo |
| `TestScheduler::pending_task_counts` | method | 294 |  | `(TestScheduler).PendingTaskCounts` | todo |
| `TestScheduler::drain_tasks` | method | 425 |  | `(TestScheduler).DrainTasks` | todo |
| `TestScheduler::advance_clock_to_next_timer` | method | 442 |  | `(TestScheduler).AdvanceClockToNextTimer` | todo |
| `TestScheduler::advance_clock` | method | 451 |  | `(TestScheduler).AdvanceClock` | todo |
| `TestSchedulerConfig` | struct | 743 |  | `TestSchedulerConfig` | todo |
| `TestSchedulerConfig::with_seed` | assoc-fn | 752 |  | `TestSchedulerConfigWithSeed` | todo |
| `Yield` | struct | 899 |  | `Yield` | todo |
| `SharedRng` | struct | 904 |  | `SharedRng` | todo |
| `SharedRng::lock` | method | 909 |  | `(SharedRng).Lock` | todo |
| `SharedRng::random_range` | method | 914 |  | `(SharedRng).RandomRange` | todo |
| `SharedRng::random_bool` | method | 923 |  | `(SharedRng).RandomBool` | todo |
| `SharedRng::random` | method | 928 |  | `(SharedRng).Random` | todo |
| `SharedRng::random_ratio` | method | 936 |  | `(SharedRng).RandomRatio` | todo |

## scheduler/src/tests.rs

Go target: `scheduler/scheduler_test.go` (package `scheduler`), tests in `scheduler_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_foreground_executor_spawn` | test | 20 |  | `TestScheduler_ForegroundExecutorSpawn` | todo |
| `test_send_local_executor_worker_submission_and_wakeup` | test | 29 |  | `TestScheduler_SendLocalExecutorWorkerSubmissionAndWakeup` | todo |
| `test_send_local_executor_retains_scheduler` | test | 86 |  | `TestScheduler_SendLocalExecutorRetainsScheduler` | todo |
| `test_background_executor_spawn` | test | 112 |  | `TestScheduler_BackgroundExecutorSpawn` | todo |
| `test_dedicated_executor_spawn` | test | 121 |  | `TestScheduler_DedicatedExecutorSpawn` | todo |
| `test_scheduler_drops_with_stalled_detached_foreground_task` | test | 130 |  | `TestScheduler_SchedulerDropsWithStalledDetachedForegroundTask` | todo |
| `test_scheduler_drops_with_stalled_detached_background_task` | test | 149 |  | `TestScheduler_SchedulerDropsWithStalledDetachedBackgroundTask` | todo |
| `test_scheduler_drops_with_never_polled_dedicated_task` | test | 171 |  | `TestScheduler_SchedulerDropsWithNeverPolledDedicatedTask` | todo |
| `test_foreground_ordering` | test | 185 |  | `TestScheduler_ForegroundOrdering` | todo |
| `test_timer_ordering` | test | 249 |  | `TestScheduler_TimerOrdering` | todo |
| `test_foreground_task_can_hold_mut_borrow_across_await` | test | 279 |  | `TestScheduler_ForegroundTaskCanHoldMutBorrowAcrossAwait` | todo |
| `test_send_from_bg_to_fg` | test | 297 |  | `TestScheduler_SendFromBgToFg` | todo |
| `test_randomize_order` | test | 317 |  | `TestScheduler_RandomizeOrder` | todo |
| `test_block` | test | 388 |  | `TestScheduler_Block` | todo |
| `test_parking_panics` | test | 407 |  | `TestScheduler_ParkingPanics` | todo |
| `test_block_with_parking` | test | 420 |  | `TestScheduler_BlockWithParking` | todo |
| `test_helper_methods` | test | 442 |  | `TestScheduler_HelperMethods` | todo |
| `test_many_with_arbitrary_seed` | test | 459 |  | `TestScheduler_ManyWithArbitrarySeed` | todo |
| `test_block_with_timeout` | test | 484 |  | `TestScheduler_BlockWithTimeout` | todo |
| `test_block_does_not_progress_same_session_foreground` | test | 570 |  | `TestScheduler_BlockDoesNotProgressSameSessionForeground` | todo |
| `test_nondeterministic_wake_detection` | test | 614 |  | `TestScheduler_NondeterministicWakeDetection` | todo |
| `test_nondeterministic_wake_allowed_with_parking` | test | 674 |  | `TestScheduler_NondeterministicWakeAllowedWithParking` | todo |
| `test_nondeterministic_waker_drop_detection` | test | 721 |  | `TestScheduler_NondeterministicWakerDropDetection` | todo |
| `test_background_priority_scheduling` | test | 781 |  | `TestScheduler_BackgroundPriorityScheduling` | todo |
| `test_spawn_dedicated_basic_round_trip` | test | 842 |  | `TestScheduler_SpawnDedicatedBasicRoundTrip` | todo |
| `test_spawn_dedicated_not_send_future` | test | 853 |  | `TestScheduler_SpawnDedicatedNotSendFuture` | todo |
| `test_spawn_dedicated_send_closure_captures` | test | 872 |  | `TestScheduler_SpawnDedicatedSendClosureCaptures` | todo |
| `test_spawn_dedicated_inner_spawn_local` | test | 896 |  | `TestScheduler_SpawnDedicatedInnerSpawnLocal` | todo |
| `test_spawn_dedicated_determinism_under_many` | test | 917 |  | `TestScheduler_SpawnDedicatedDeterminismUnderMany` | todo |
| `test_spawn_dedicated_dropping_task_cancels_future` | test | 992 |  | `TestScheduler_SpawnDedicatedDroppingTaskCancelsFuture` | todo |
| `test_spawn_dedicated_detached_child_runs_after_root_completes` | test | 1033 |  | `TestScheduler_SpawnDedicatedDetachedChildRunsAfterRootCompletes` | todo |

