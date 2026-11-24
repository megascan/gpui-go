# M0-app-core debt

- D-BORROWLOC (core-app.md): `TRACK_THREAD_BORROWS` is read from the process environment once at startup (upstream: compile-time `option_env!`); the printed `ThreadId(N)` is the goroutine id.
- Borrow diagnostics print `file:line` from `runtime.Caller` (absolute path, no column); upstream's `Location` prints `file:line:column`.
- Iteration over `FxHashMap` / `HashMap` keys (system window tab groups, display snapshots in `displayEvents`) uses ascending key order; upstream's order is arbitrary. Deterministic, but a different "arbitrary".
- `Shutdown`: a hang-monitor flush the monitor never completes waits for `ShutdownTimeout` without upstream's "hang monitor exited before flushing" warning, because `(*hangMonitor).requestFlush`'s channel is closed only on success. Its future wakes from a helper goroutine.
- stubs.go placeholder bodies so App construction does not panic before their owners land: `NewSvgRenderer` (stores the asset source), `DefaultKeymap`, `defaultFocusMap`, `defaultInspectorElementRegistry`, `installForegroundJournal` (zero journal) and `observePower` (no-op), the last two reached only under `-tags gpui_profiler`.
- `(App).RefCountsDropHandle` returns `any` (upstream `impl Sized`).
- Field / getter collisions: `Window.last_window_bounds()` / `display_id()` are read as the fields `lastWindowBounds` / `displayID` (D-GETTER); `GpuiMode::skip_drawing` is `shouldSkipDrawing`; App's `background_executor` field is `bgExecutor` (the name collides with the AppContext primitive `backgroundExecutor`), also in the AsyncApp stub.
- wasm branches (`Run`'s `mem::forget`, `Shutdown`'s spawn instead of block) are `runtime.GOOS == "js"` checks in app.go, not build-tagged files; `Run` retains the Application in a package-level slice on js.
- `foregroundJournal` / `hangMonitor` App fields are always compiled and set only when `profilerFeatureEnabled` (D-FEATURE).
- `element_arena` / `event_arena` are not ported (D-ARENA).
