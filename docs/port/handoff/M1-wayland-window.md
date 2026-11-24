# M1-wayland-window handoff

## For M3-exit (a11y seam)

- platform/linux/wayland/a11y.go: `newA11yAdapter(gpui.A11yCallbacks) a11yAdapter` returns nil. `a11yAdapter` is `UpdateIfActive(func() a11y.TreeUpdate)` + `UpdateWindowFocusState(bool)`, which `*a11y/atspi/bus.Adapter` satisfies; wire it to `bus.NewAdapter` with activation / action / deactivation handlers wrapping the callbacks (upstream's Trivial*Handler). Only a11y.go needs editing.

## For M1-linux-platform

- `WaylandWindow.Close()` is upstream's Drop: renderer destroyed first, wl_surface last, then `DropWindow` and a deferred `WaylandWindowStatePtr.Close`. It must run before `WaylandConnection.Close`. Idempotent.
- `TestClient_AttachClose` (client_test.go) is gated by `GPUI_TEST_DISPLAY=1` + `WAYLAND_DISPLAY`; run it under `dbus-run-session --` (Attach wires the XDP portal source). Passed on Hyprland.
- The owed `WindowIdentifier` test (handoff from M1-wayland-client) is not written: exporting needs a mapped toplevel, i.e. a full app with a window. Cover it with a shoot-verified example (check the identifier starts with `wayland:`).
- Resolution (2026-11-24, M3-exit): a11y.go now wires bus.NewAdapter with the Trivial* handlers; the adapter is released in WaylandWindow.Close.
