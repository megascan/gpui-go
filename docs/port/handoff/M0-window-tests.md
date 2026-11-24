# M0-window-tests handoff

window.rs tests 8412-11169 are in window_test.go (45 ported and passing). The two feat:profiler tests (`test_frame_requests_and_skips_are_journaled`, `test_frame_demand_survives_throttling`, window.rs 8537-8700) are `blocked:M3-profiler-journal` and are empty t.Skip stubs: their bodies are NOT ported, because the journal API (install_test_foreground_journal, collector, ForegroundJournalEntry) does not exist yet. M3-profiler-journal does not own window_test.go, so M3-exit must port the bodies (behind the `gpui_profiler` build tag, as upstream's cfg(feature = "profiler")) and flip the two core-window.md rows. window_smoke_test.go was deleted (covered by the ported tests).

Fixes made here:
- elements_div_paint.go: the on_drop listener now releases the taken drag's view (upstream drops it at the end of the loop body); file drops leaked the drag entity.
- frame `nextInspectorInstanceIDs` / `inspectorHitboxes` start nil (upstream `FxHashMap::default()`, capacity 0) and are allocated on first insert; ToggleInspector resets them to nil. `inspector_only_tracks_its_open_window` asserts this as `== nil`.

For the owner of app.go / app_test_context.go (M0-exit), bugs the tests work around (workarounds live in window_test.go and should be dropped once fixed):
1. Window root view is never released on removal or teardown (`updateWindowTrail` / `releaseAppWindows`), see M0-div-paint handoff item 2. window_test.go uses `windowTestReleaseAll` (wraps `divTestReleaseRoot`) and, for `NativeCloseDefersAppAccess` / the removed-window drag, aliases `window.root` and releases it by hand.
2. `App.StopActiveDrag` and `App.endPlatformDrag` (also reached from window removal and `FileDropEventEnded`) set `activeDrag` / `platformOwnedDrag` to None without releasing `AnyDrag.View`. Repro: `TestWindow_FileDragIsPromotedOnceAndRestoredInSourceWindow` without the `stopped` / `suspended` releases leaks the drag entity. Fix: release the view of the dropped drag (and of a suspended platform-owned drag) at those sites.
3. App teardown never releases a still-active drag (`activeDrag` at drop time); the same test releases it by hand at its end.

Not done here (file not owned): M0-text-elements' handoff asks to un-skip five tooltip / prompt tests in elements_div_test.go and flip core-elements.md:305-310; elements_div_test.go and core-elements.md are not in this stage's owns list.

Other notes:
- `TestWindow_WindowOpenedFromBoundsObserverUsesNewBounds` logs `handling active status change for window WindowId(1v1): window not found`: TestPlatform.SetActiveWindow's foreground task runs during teardown after the test-side root release (`windowTestReleaseAll`) has removed the window. Expected to disappear once teardown releases roots itself (item 1).
- M0-div-paint handoff item 7 (frame.clear does not release tooltip views) is in window.go; this stage did not touch it. The window.go fix belongs to M0-exit (window.go has no later builder owner), which M0-blocked-tests / whoever un-skips the tooltip tests depends on.
- Resolution (2026-11-24, M3-exit): TestWindow_FrameRequestsAndSkipsAreJournaled and TestWindow_FrameDemandSurvivesThrottling ported (profiler builds).
