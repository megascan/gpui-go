# M3-bench-context debt

- `BenchAppContext` uses `NullHTTPClient` where upstream uses `http_client::BlockedHttpClient` (crate not vendored; same effect: every request fails).
- Bench apps install the foreground journal themselves when `newApp` left it zero (no `gpui_profiler` tag); task polls are only journaled with `-tags gpui_profiler`, so the three foreground-work tests skip without it (upstream `bench-support` implies `profiler`).
- `testWindowFrameWakerSchedulesFrame` (upstream cfg `bench-support`) is set by `buildBenchAppContext` and cleared by `Teardown` instead of being a compile-time feature.
- `BenchPlatform` caches one `ThreadedDispatcher` per goroutine; RunBench runs every invocation on one persistent benchmark goroutine (`onBenchThread`) because testing.B calls the benchmark function on varying goroutines. A `b.Skip`/`b.Fatal` inside it ends that goroutine and its cached dispatcher is leaked.
- `BenchmarkFocusCleanup_*` skip under debug assertions (`use -tags gpui_release`) where upstream asserts `!cfg!(debug_assertions)`, so `go test -bench` without the tag does not fail.
- RunBench without inputs or rng runs in the top-level benchmark, so testing.B's repeated calls print one report per call (Criterion prints once).
- `BenchOptions.SampleSize` (`sample_size = N`) has no testing.B equivalent and is not ported; use `-benchtime`.
