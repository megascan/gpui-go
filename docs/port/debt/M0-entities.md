# M0-entities debt

- Strict leak reports (handles, subscriptions, leases collected without their drop method) go to one process-global list drained by `entityMap.report()`. Parallel tests in one binary can see each other's reports.
- Strict mode is the default in every test binary (`testing.Testing()` with `GPUI_LEAK_DETECTION` unset), not only this repo's. Downstream users opt out with `GPUI_LEAK_DETECTION=1`.
- A lease abandoned by a panic that a test recovers is reported in strict mode; upstream skips the LeaseInner check while panicking, which Go cannot detect.
- Over-release panics inside a `runtime.AddCleanup` callback run on the cleanup goroutine and abort the process (upstream panics on the dropping thread).
- `Clone` / `Upgrade` after the ref-count table was collected skip leak tracking; upstream's `upgrade().unwrap()` panics there.
- Tasks are not covered by strict collection reports yet; the scheduler owns `Task` (M0-executors / scheduler).
- `debug_assert!` checks (wrong context, dropped-while-referenced, pop without push, scope left open) follow `gpuiutil.DebugAssertions`.
