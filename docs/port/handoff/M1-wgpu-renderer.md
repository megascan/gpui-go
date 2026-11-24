# M1-wgpu-renderer handoff

## For M1-wayland-window, M1-x11-window (and other Linux windows)

- Shared context: create one `&wgpu.GPUContext{}` per platform (upstream `Rc<RefCell<Option<WgpuContext>>>`; nil `Context` = None) and pass the same pointer to every window's renderer.
- Construct: `wgpu.NewRenderer(gpuContext, handle gpui.RawWindowHandle, wgpu.WgpuSurfaceConfig{Size, Transparent, PreferredPresentMode}, compositorGPU gpuiutil.Option[wgpu.CompositorGPUHint]) (render.Renderer, error)`; `wgpu.NewWgpuRenderer` is the same returning `*wgpu.WgpuRenderer` (extra methods: `NeedsRedraw`, `MaxTextureSize`, `ViewportSize`, `UnconfigureSurface`, `WgpuAtlas`).
- Handles: `RawWindowHandleWayland` (WaylandDisplay + WaylandSurface) and `RawWindowHandleXcb` (XcbConnection + XcbWindow) are supported; other kinds return an error.
- `Destroy()` must run before the wl_surface / xcb window and the display connection go away. It is idempotent.
- Device loss: when `DeviceLost()` is true call `Recover(handle)` and request another frame on error, as upstream.

## For M4-web-wgpu-js

- `NewWgpuRendererFromCanvas` row is `blocked:M4-web-wgpu-js`; build it on `NewWgpuRendererFromSurface` (exists, shared). `createSurface` maps `RawWindowHandleWeb` to `SurfaceTarget.CanvasID`. Draw's wasm device-lost branch is gated by `isWasm`.
