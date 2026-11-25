# M1-wgpu-context handoff

## Feature probe (wgpu-native v29.0.1.1 via oliverbestmann/webgpu v1.38.1, L-WGPUVER)

Run `go test -v -run FeatureProbe ./render/wgpu/` to repeat it. Result on the dev machine (RTX 4060 Laptop, NVIDIA 615.78.08; adapters: Vulkan discrete + GL "Other"):

- `EnumerateAdapters`: exported and working (2 adapters). Adapter selection follows upstream (`wgpu_context.rs:215`, `:531`); no fallback needed, rows `done`.
- `DUAL_SOURCE_BLENDING`: reported by both adapters, requestable, and all four shader variants (storage, webgl, subpixel_storage, subpixel_webgl) compile on the device. No fallback needed; the non-subpixel path is still taken when an adapter lacks the feature, as upstream.
- `TRANSIENT_ATTACHMENT`: missing. wgpu-native v29 panics converting the usage bit ("invalid texture usage for texture descriptor", src/lib.rs:2547) and the panic aborts the process. Fallback: `webgpu.SupportsTransientAttachment = false`, so `(*WgpuContext).SupportsTransientAttachments` returns false on native (`dev:L-WGPUVER`) and the intermediate textures use plain RENDER_ATTACHMENT. The facade also refuses the bit with a validation error instead of passing it to wgpu-native. The probe runs this check in a child process and says when a newer binding accepts it.

## For M1-wgpu-renderer

- Write the renderer only against `render/wgpu/internal/gpu`. Every Device / Queue / Texture / Encoder call reports validation errors to the innermost `PushErrorScope` (Go-side stack per device, like wgpu's) or the `OnUncapturedError` handler that `installDeviceErrorState` installs; creation calls return inert objects on failure (methods on them are no-ops that report "X is invalid"). There is no handler-less fallback: without one an error panics, like wgpu's default.
- wgpu-native makes queue-submit validation errors fatal (abort) whatever the error scope; wgpu routes them to `on_uncaptured_error`. Keep resources alive until their submission is done.
- wgpu-native v29 aborts the process on any `wgpuSurfaceConfigure` error (`handle_error_fatal`, D-WGPUCONFIGURE): validate the configuration before calling `Configure` (the size clamp and capability picks in `wgpuRendererSurfaceConfiguration` already do). `wgpuSurfacePresent` only logs a warning and returns an error status. Both go through the binding's own `Configure` / `Present`.
- Already ported in wgpu_renderer.go: the four shader strings (`storageBufferShaders`, `webglShaders`, `subpixelStorageBufferShaders`, `subpixelWebglShaders`, `wgpu_renderer.rs:25-46`), `WgpuSurfaceConfig` and `wgpuRendererSurfaceConfiguration` (upstream `WgpuRenderer::surface_configuration`, used by adapter selection), plus the four `*_is_valid_wgsl` tests in wgpu_renderer_test.go (compiled on a device; they skip without an adapter or, for the subpixel variants, without dual-source blending). `least_common_multiple` and the rest of the file are yours.
- `wgpuContextNewHeadless` (`new_headless`) returns a copy of the cached context plus the headless target format; it is unexported, as upstream's `pub(crate)`.
- `TexelCopyBufferLayout`: zero `BytesPerRow` / `RowsPerImage` means None. `BindingResource.Size` zero binds the whole buffer. `MultisampleState` zero is count 1, mask all.
- Tests that need the GPU share `testDeviceAndQueue(t, features)` in wgpu_atlas_test.go (skips without an adapter).

## For M1-wayland-client and M1-x11-client

- Because a failing surface configure aborts instead of trying the next adapter (D-WGPUCONFIGURE), pass the compositor GPU hint (`detect_compositor_gpu`, `compositor_gpu_hint_from_dev_t`) whenever upstream does: on hybrid laptops it is what puts the compositor's GPU first in adapter order.

## For M4-web-wgpu-js

- Shared files (`gpui_wgpu.go`, `wgpu_context.go`, `wgpu_atlas.go`, `wgpu_renderer.go`, the facade in `internal/gpu`) build for `linux || js`. The js-only pieces are the panicking stubs in `render/wgpu/stubs.go` (`createInstance`, `browserSupportsTransientAttachments`) plus `webGlPrefersRgba = true` and `nativeSupportsTransientAttachments`; replace stubs.go with real `_js.go` files and delete it. The native counterparts are in `instance_native.go`.
- Rows `WebBackendPreference`, `PreparedWebGraphics`, `NewWgpuContextWeb` are `blocked:M4-web-wgpu-js`.
- `WgpuBackend` is a comparable struct (`Kind` + `Native`); `WgpuBackendBrowserWebGpu` / `WgpuBackendGl` are the web kinds.
