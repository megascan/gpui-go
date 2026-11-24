# M4-apple-metal-renderer handoff

For M4-macos-window / M4-ios-window:
- `(*MetalRenderer).Destroy` releases everything (pipelines, queue, atlas, CoreVideo cache, layer, device): upstream's `destroy()` is empty and the release happens in Rust's Drop. Call it where MacWindow's Drop calls `renderer.destroy()`; the renderer, its atlas and its layer are unusable afterwards. `MetalHeadlessRenderer.Destroy` is its drop.
- `SetFrameDrawable(d)` takes a +1 drawable and returns the replaced +1 drawable (0 = None), which the caller releases. `Draw` returns true when it submitted a frame.
- `render.Renderer` methods without an upstream counterpart: `SetSubpixelLayout` no-op, `SupportsDualSourceBlending` false, `GPUSpecs` false, `DeviceLost` false, `Recover` nil.
- `Context` is `*InstanceBufferPool` (mutex inside); create with `NewInstanceBufferPool()` for upstream's `Default::default()`.
- CoreVideo (CVMetalTextureCache, CVPixelBuffer getters) and CGColorSpaceCreateWithName bindings live in platform/apple/metal_renderer.go; if platform/macos screen capture needs them, an exit stage can move them into internal/apple.

Resolved: M4-apple-ffi's TestProcessInfo_NotificationsAreDeferredAndCancelledOnDrop now runs under gpui.RunTest. M4-apple-metal-atlas's renderer notes: InputIndex values and SurfaceBounds layout are checked by TestMetalRenderer_InputIndicesMatchSceneHeader (keep metalhdr's local surfaceBounds mirror in sync).
