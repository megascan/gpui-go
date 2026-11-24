# Debt: M0-platform-types

- `RefreshIntervalFromHz` saturates at `time.Duration`'s maximum (about 292 years) where Rust's `Duration::from_secs_f64` only panics beyond u64 seconds; refresh rates below ~3.4e-9 Hz are affected.
- `profilerFeatureEnabled` is a const in stubs.go until M0-profiler-core adds the `gpui_profiler` tag files.
