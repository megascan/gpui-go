# M1-linux-dbus handoff (for M1-linux-common / M1-linux-screencap)

Package `platform/linux/internal/dbus` (imported as godbus alias inside; exports):
- `MonitorPowerMode(ctx, conn, onChange func(gpui.PowerMode) error) error` blocks; run on a goroutine, marshal onChange to main thread.
- `NewSystemNotificationState(conn)`; `Show(appName gpui.Option[string], n)`, `Dismiss(tag)` (no-op as upstream), `OnResponse(dispatch func(func()), cb)` where dispatch runs the func on the main thread.
- `NewXDPEventSource(conn, send func(Event)) *XDPEventSource`; `Close()` is the Drop (cancels listener, removes matches). Events: EventWindowAppearance, EventCursorTheme, EventCursorSize, EventButtonLayout, EventTextScaleFactor. send is called from the listener goroutine.
- `WriteCredentials(conn, url, user, pw) error`, `ReadCredentials(conn, url) (user, pw, ok, err)`, `DeleteCredentials(conn, url) error`; `KeyringLabel`. Blocking; call from background tasks. The Secret Service rows in parity/linux-common.md (platform.rs:1085-1150) can be marked done by M1-linux-common when it wires these.
- Tests use a private dbus-daemon with an isolated config (no service activation dirs): see testbus_test.go `startTestBus`.
