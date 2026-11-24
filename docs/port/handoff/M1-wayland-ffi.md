# M1-wayland-ffi handoff

## For M1-wayland-client: cursor themes

- Upstream cursor.rs loads themes with the pure-Rust wayland-cursor crate (xcursor). Per STAGES.md this port uses libwayland-cursor (`wl.LoadCursorTheme`, `CursorTheme.Cursor`, `Cursor.Images`, `CursorImage.Buffer`, `Cursor.FrameAndDuration`). Theme lookup and fallback (XCURSOR_PATH, inherits) follow libwayland-cursor's xcursor.c; check row by row against cursor.rs when porting it and register a deviation if behaviour differs.
- `CursorImage.Buffer()` returns a wl_buffer owned by the theme: wrap it with `wl.Wrap(display, ptr, wlproto.WlBufferInterface).(*wlproto.WlBuffer)`; never Destroy it, destroy the theme instead.

## API notes (internal/linux/wl, internal/linux/wlproto)

- `wl.Connect` / `Display.{Dispatch,DispatchPending,Roundtrip,Flush,PrepareRead,ReadEvents,CancelRead,FD,Err,Disconnect}`; `wlproto.GetDisplay(d)` is the wl_display proxy; `wlproto.Bind[T](registry, name, iface, version)`.
- Each interface `X` has `XInterface`, request methods, `XListener` (struct of funcs) with `SetListener`, enum types `X<Enum>` with `X<Enum><Entry>` constants (bitfields have `Contains`), and `X<Message>SinceVersion` constants.
- Requests on destroyed proxies are ignored and constructors return inert objects (wayland-backend semantics). Destructor requests and destructor events destroy the proxy; `Proxy.Destroy()` is the local destroy for interfaces without a destructor request.
- Unhandled events with fd arguments close the fd. Listener panics are re-raised from the Dispatch call that ran them.
- `Display.Disconnect` destroys every proxy still managed on the connection, so destroy wgpu surfaces first.
