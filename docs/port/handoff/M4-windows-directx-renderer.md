# M4-windows-directx-renderer handoffs

- M4-windows-window / M4-windows-events: `NewDirectXRenderer(hwnd, devices, disableDirectComposition)`; call `DrawWithAppearance(scene, background)` where window.rs calls `draw`, `Resize` (events.rs:264), `HandleDeviceLost(devices)` (events.rs:1343, panic "Device lost: ..." on error), `MarkDrawable`, `GPUSpecs()` for `gpu_specs().log_err()`, `RenderToImage(scene, background)` for test-support, and `Destroy()` where the window drops its renderer. `directXFontInfo` and `directXRendererGetFontInfo` moved from stubs.go into directx_renderer.go.
- M4-ci-windows: `TestDirectxRenderer_ConstantBufferLayout` runs on the runner; the renderer itself is exercised only once a window exists.
