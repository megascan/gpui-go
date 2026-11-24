# M0-window-tests debt

- window_test.go: `windowTestCanvas` stands in for `canvas` (M0-elements-basic is later in the chain); swap to `Canvas` when it lands.
- window_test.go: leak workarounds for app.go teardown / drag release bugs (handoff items 1-3): `windowTestReleaseAll`, hand-released root aliases, drag view releases.
- window_test.go: `LongPressListener` captures a WeakEntity and upgrades per event instead of holding `cx.Entity()` in the paint closure: Go closures cannot release a captured strong handle when the frame drops them.
- window_test.go: `test_window_reports_no_raw_handle_instead_of_panicking` checks `RawWindowHandle().Kind == RawWindowHandleNone` (the Go shape of raw_window_handle's NotSupported).
