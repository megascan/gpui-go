# Handoff: M1-x11-client

## To M1-x11-window
- Replace `platform/linux/x11/stubs.go`: `NewX11Window`, `X11WindowStatePtr` methods, `X11WindowState`, `X11Window`, `Clipboard`/`NewClipboard`/`ClipboardKind`/`WaitConfig`. Keep signatures the client calls (see client.go `OpenWindow`, `handleEvent`). `XcbAtoms`/`NewXcbAtoms`, `checkReply`, `getReply`, `xcbFlush` are real and should move into window.go.
- `NewX11Window` receives `*xcbshim.Connection` (client state `xcb`) for surface creation; the client closes it in `X11Connection.Close`.
- Window must call `X11ClientStatePtr.DropWindow` on destroy and `UpdateIMEPosition` on IME bounds changes.

## To M3-root-capture / M3-exit
- Wire `scapScreenSources(executor)` in stubs.go to the real scap screen source listing; `IsScreenCaptureSupported` already returns true.
- Resolution (2026-11-24, M3-exit): scapScreenSources returns gpui.ScapScreenSources(executor).
