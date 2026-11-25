# M0-window-core debt

- Element arena (D-ARENA): each App's `elementArena` (scope depth plus the `deferArenaDrop` queue) lives in a package-level map keyed by `App.this` (`weak.Pointer[AppCell]`), cleaned up with `runtime.AddCleanup` on the AppCell, because app.go has no `element_arena` field. Move it onto App when app.go is next edited. `currentElementArena` is package-global (upstream's thread-local); root tests must stay non-parallel.
- `elementArenaScope.End` only logs when the scope ends without `exit`; upstream also debug-asserts when not unwinding, which Go cannot detect without recover.
- `FocusHandle` literals without a focus map (tab_stop_test.go builds them) are uncounted: `Clone` copies, `Release` does nothing.
- `ElementID` NamedChild parents are interned in a process-wide map that never shrinks, so value equality works with `==`. `Path` ids compare by raw string (upstream compares path components); the hash follows std's component-wise `Path::hash` for `/` separators only.
- The a11y channel senders created in `newWindow` are never closed, so the three receive-loop tasks live until the app exits; whoever ports `remove_window` should close them (upstream drops them with the window's callbacks).
- `newWindow` does not close the platform window when `NewWindowProfiler` fails after `OpenWindow` succeeded (upstream drops the Box, which closes it).
- `deferArenaDrop` calls made outside any draw scope go to `fallbackElementArena`, which nothing clears (upstream's thread-local fallback arena is never cleared either, but there it only holds memory). `VisualTestContext.Draw` currently runs without a scope, so its view drops land there.
