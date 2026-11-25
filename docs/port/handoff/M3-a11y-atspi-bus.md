# M3-a11y-atspi-bus handoff

- M3-exit: the accesskit_unix adapter is `bus.NewAdapter(activation, action, deactivation)` in package `github.com/megascan/gpui-go/a11y/atspi/bus` (STAGES core task), not `a11y/atspi.NewAdapter` (that name is accesskit_atspi_common's adapter). Methods: `UpdateIfActive(func() a11y.TreeUpdate)`, `UpdateWindowFocusState(bool)`, `SetRootWindowBounds(outer, inner a11y.Rect)`, `Release()` (upstream Drop; call where the window drops `accesskit_adapter`).
- The first `NewAdapter` connects to `DBUS_SESSION_BUS_ADDRESS`; tests that create one must point it at a private dbus-daemon first (see `a11y/atspi/bus/bus_test.go` TestMain).
- Resolution (2026-11-24, M3-exit): bus.NewAdapter wired into both Linux windows; Release called from Close.
