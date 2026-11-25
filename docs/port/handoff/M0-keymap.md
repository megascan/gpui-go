# M0-keymap handoff

- M0-window-core: stubs.go `FocusHandle.Clone` / `Release` now have placeholder bodies (copy / no-op); replace them with the counted handle. tab_stop_test.go's `testFocusHandles.handle` builds `FocusHandle{id, TabStop, TabIndex}` literals; keep those field names or update the helper to `newFocusHandle(...)` plus the tab builders.
- Window stages: the dispatch tree and tab stop map are unexported (`dispatchTree`, `tabStopMap`); the method list is in docs/port/parity/core-input.md "Notes (M0-keymap)".
- M0-blocked-tests: the 20 `#[gpui::test]` tests of key_dispatch.rs are `t.Skip` placeholders in key_dispatch_test.go; test actions `dispatchTestTestAction` / `dispatchTestSecondaryTestAction` and `testDispatchTree` already exist there.
