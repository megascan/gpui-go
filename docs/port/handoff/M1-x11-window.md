# M1-x11-window handoff

- `scapScreenSources` (platform/linux/x11/client.go, end of file) still panics: the root has no `gpui.ScapScreenSources` yet. M3-root-capture (or the exit after it) should replace the body with a call to the root function (client.rs:1688).
- `WaitForClipboardHandovers` (clipboard.go) must be called by M1-linux-platform where upstream platform.rs:564 and :683 call `wait_for_clipboard_handovers`.
- `newA11yAdapter` in platform/linux/x11/a11y.go: wire to a11y/atspi (M3-exit). The seam interface is `UpdateWindowFocusState`, `UpdateIfActive`, `SetRootWindowBounds`.
