# M0-div-paint debt

- elements_div_paint.go `windowSpawn`: spawns with `cx.Spawn` + `newAsyncWindowContext(app, handle)` instead of `window.Spawn`, because `Window.Spawn` hands out an AsyncWindowContext with a nil scheduler Scope (handoff item 1). Switch back once fixed.
- Tooltip `AnyView`s built by tooltip builders are aliased into `Window.SetTooltip` requests and never released explicitly; the frame's tooltip requests outlive the active-tooltip state, so releasing on drop could hit a use-after-release on view-cache reuse. The entity GC safety net collects them.
- The focus-on-mouse-down listener captures a `WeakFocusHandle` and upgrades on dispatch instead of holding a counted clone (frame listeners have no drop hook, so a clone would leak). Behaviour differs only when every strong handle is gone while the listener is still in the rendered frame.
- `Interactivity.trackedFocusHandle` (TrackFocus clone or the element-state clone) is released with `deferArenaDrop` at arena clear; a Div that is built but never laid out leaves it to the GC net.
- The `FocusHandle` stored in `InteractiveElementState` is not released when a stale element state is discarded at frame swap (elementStateMap has no drop hook).
- FileDropEvent exit listeners are appended to `nextFrame.mouseListeners` directly: the `FileDropEvent` interface does not satisfy `OnMouseEvent`'s `MouseEvent` constraint.
- elements_div_test.go: the test-local `divTestCanvas` stands in for `canvas` (M0-elements-basic); `divTestReleaseRoot` / `divTestRemoveWindow` work around teardown not releasing window roots and frame tab stops (handoff item 2).
