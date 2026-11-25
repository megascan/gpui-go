# M1-wayland-window debt

- Windows use `wgpu.NewWgpuRenderer` (the concrete `*WgpuRenderer`), not `wgpu.NewRenderer`: upstream calls `max_texture_size` and `needs_redraw`, which `render.Renderer` lacks.
- `RetryTimerFired(time.Time)`: scheduler has no `time.Time` -> `Instant` constructor, so the frame signal is `InstantNow().Add(-time.Since(deadline))` (only evaluated when profiling is on).
- `WaylandWindow.Display()` looks the `*wlproto.WlOutput` up in the client's `wlOutputs` (ObjectID is a `*wl.Proxy` and cannot be converted back); it returns nil once the connection is closed, where upstream still returns the cached output.
- `applyExclusiveEdge`'s warning formats the anchors by hand (`Anchor(TOP | LEFT)`), approximating bitflags' Debug output.
- The clipboard UTF-8 error log says "invalid utf-8 sequence" without Rust's byte index.
- As upstream, wp_fractional_scale_v1 objects and a pending frame wl_callback are never destroyed when a window closes.
