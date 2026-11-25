# M1-wgpu-renderer debt

- Headless readback waits with `Device.Poll(true)`, which has no timeout; upstream polls with a 30 s timeout and then waits 30 s on the map callback. A hung GPU blocks the caller instead of returning an error.
- GPU objects are released explicitly where upstream drops them (per-frame bind groups, command buffers and encoders after submit; pipeline layouts and shader modules after pipeline creation; replaced pipelines, depth textures, instance buffers/textures, atlas bind groups and headless targets). The device and queue belong to the WgpuContext and are never released by the renderer.
- `WgpuRenderer.Destroy` releases the surface and core; the atlas stays alive (shared with the window's sprite atlas, as upstream's `Arc<WgpuAtlas>`).
- `Recover` sleeps 350 ms on the calling (platform) thread as upstream's `thread::sleep`.
- `NewWgpuRendererFromCanvas` (wasm) is left to M4-web-wgpu-js; `NewWgpuRendererFromSurface` takes an `internal/gpu` Surface, so only render/wgpu code can call it.
