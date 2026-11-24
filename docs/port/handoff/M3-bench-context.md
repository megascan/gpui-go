# M3-bench-context handoffs

- M3-platform-benches: upstream's `#[gpui::bench]` passes `gpui_platform::current_headless_renderer` and `gpui_platform::bench_text_system()`; the root cannot import platform, so pass them through `gpui.BenchOptions{TextSystem, HeadlessRendererFactory}` (nil text system = NoopTextSystem).
- M3-exit / CI: the foreground-work bench tests (`TestBenchContext_ForegroundWork*`, `TestBenchContext_BenchTaskReportsLongTaskWithoutWindow`) only run under `go test -tags gpui_profiler .`; `BenchmarkFocusCleanup_*` only measure under `-tags gpui_release`.
