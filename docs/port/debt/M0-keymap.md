# M0-keymap debt

- stubs.go: `FocusHandle.Clone` returns a copy and `FocusHandle.Release` does nothing (placeholder bodies, so tab_stop.go can clone and release where upstream does). M0-window-core replaces both with the counted handle.
- tab_stop_test.go builds `FocusHandle{id: ..., TabStop: ..., TabIndex: ...}` literals instead of upstream's `FocusHandle::new(&focus_map).tab_stop(..).tab_index(..)`, because `newFocusHandle` is a panicking stub. If M0-window-core renames those fields, switch the helper `testFocusHandles.handle` to the real constructor and builders.
