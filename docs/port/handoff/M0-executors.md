# M0-executors handoff

- M3-profiler-journal: upstream's `ForegroundExecutor.foreground_runnables` (`Option<ForegroundRunnableCounter>`, executor.rs:30, :344-372, :420-431) and `PlatformScheduler.foreground_runnables` (`queued()` in `schedule_local`, platform_scheduler.rs:32, :110) are not ported; executor.go and platform_scheduler.go need the field and the `queued()` calls once the journal counter exists (build tag `gpui_profiler`).
- M0-app-core: stubs.go now has `(*App).ForegroundExecutor() ForegroundExecutor` (used by `TaskDetachAndLogErr`).
- M0-test-platform: `TestExecutor_SanityTestTasksRun` in executor_test.go is a t.Skip placeholder (row `blocked:M0-test-platform`); port its body from executor.rs:602-626 once `App` and `TestPlatform` exist.
