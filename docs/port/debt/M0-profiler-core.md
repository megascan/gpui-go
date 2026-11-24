# M0-profiler-core debt

- `Histogram` (port of hdrhistogram 7.6.0 `Histogram<u64>`: new(3) auto-resize, record, len, is_empty, min, max, mean, value_at_quantile, count_between, subtract, clone) lives in profiler.go because the stage owns no other path. Should move to `internal/hdrhistogram` with `docs/port/parity/ext-hdrhistogram.md` and tests from the crate.
- `scheduler.Location` has no column, so `SerializedLocation.Column` is always 0 and `TaskStatistics.String` prints 0 where upstream prints the column.
- `TaskStatistics.String` / `ActionStatistics.String` format durations with Go's `time.Duration.String`, not Rust's `Debug` (`0s` vs `0ns`, otherwise equal for typical values).
- Journal entry points in stubs.go (`beginForegroundTurn`, `endForegroundTurn`, `recordTaskPoll`, `recordFramePending`, `recordWindowClosed`, `powerInterruptedSince`, `frameSampleIsValid`, `recordInput`, `recordAction`, `recordView`, `recordDraw`, `recordPresent`) behave as upstream does with no journal installed (`with_journal` no-op), not as panics, so WindowProfiler tests run before M3.
- `TestProfiler_FramesAreSplitByRefreshIntervalAndClassified` omits `journal::install_test_foreground_journal(64, 4)`; with no visibility or power changes the result is identical.
- `spawnHangMonitor` never returns `HangMonitorErrorSpawn` (a goroutine cannot fail to start); the variant is kept for parity.
- `ThreadTimings.Release` removes the registry entry immediately; upstream removes it when the last `Arc` drops, which can be after a concurrent collector's upgrade.
