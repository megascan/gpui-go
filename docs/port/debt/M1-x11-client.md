# Debt: M1-x11-client

- Event pump: xgb owns the socket, so a goroutine drains `WaitForEvent` into a mutex queue and pings the loop (`common.MakePing`). Upstream polls the fd directly. X errors arrive as `xErrorEvent` in the same queue.
- Second libxcb connection (`xcbshim.Connect`) exists only for xkbcommon-x11 keymap/state and is handed to the window stage; two X connections per client.
- `internal/xrm` and `internal/xcursor` are local ports of x11rb's resource_manager/cursor and the xcursor crate (xgb has neither).
- `xcbFlush` is a no-op: xgb writes requests immediately.
- `NewClipboard` stub returns an empty clipboard instead of panicking so attach works before M1-x11-window; its methods panic.
- `X11Connection.Close` closes the X connection (upstream relies on Rc drop).
- Legacy RandR (<1.5) path iterates CRTCs in server order; upstream matches.
- `ScreenCaptureSources` calls a panicking local `scapScreenSources` until the scap back end lands.
- `getScaleFactor` approximates Rust `f32::from_str` with `strconv.ParseFloat` plus a reject list for hex/underscore forms.
