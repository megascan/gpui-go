# Parity: wgpu

Scope: gpui_wgpu (gpui_wgpu.rs, wgpu_atlas.rs, wgpu_context.rs, wgpu_renderer.rs). Milestone: M1, M4 (web). Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_wgpu/src/wgpu_atlas.rs

Go target: `render/wgpu/wgpu_atlas.go` (package `wgpu`), tests in `wgpu_atlas_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WgpuAtlas` | struct | 23 |  | `WgpuAtlas` | dev:D-WGPUATLASBOUNDS |
| `WgpuTextureInfo` | struct | 41 |  | `WgpuTextureInfo` | done |
| `WgpuAtlas::new` | assoc-fn | 48 |  | `NewWgpuAtlas` | done |
| `WgpuAtlas::from_context` | assoc-fn | 65 |  | `WgpuAtlasFromContext` | done |
| `WgpuAtlas::before_frame` | method | 73 |  | `(WgpuAtlas).BeforeFrame` | done |
| `WgpuAtlas::get_texture_info` | method | 82 |  | `(WgpuAtlas).GetTextureInfo` | done |
| `WgpuAtlas::clear` | method | 93 |  | `(WgpuAtlas).Clear` | done |
| `WgpuAtlas::handle_device_lost` | method | 102 |  | `(WgpuAtlas).HandleDeviceLost` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `before_frame_skips_uploads_for_removed_texture` | test | 438 | wasm | `TestWgpuAtlas_BeforeFrameSkipsUploadsForRemovedTexture` | done |
| `remove_deallocates_tile_space_for_reuse` | test | 462 | wasm | `TestWgpuAtlas_RemoveDeallocatesTileSpaceForReuse` | done |
| `reused_texture_id_has_new_generation` | test | 506 | wasm | `TestWgpuAtlas_ReusedTextureIDHasNewGeneration` | done |
| `swizzle_upload_data_preserves_bgra_uploads` | test | 552 | wasm | `TestWgpuAtlas_SwizzleUploadDataPreservesBgraUploads` | done |
| `swizzle_upload_data_converts_bgra_to_rgba` | test | 561 | wasm | `TestWgpuAtlas_SwizzleUploadDataConvertsBgraToRgba` | done |

## gpui_wgpu/src/wgpu_context.rs

Go target: `render/wgpu/wgpu_context.go` (package `wgpu`), tests in `wgpu_context_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WgpuContext` | struct | 13 |  | `WgpuContext` | done |
| `DeviceErrorState` | struct | 31 |  | `DeviceErrorState` | done |
| `DeviceErrorState::device_lost` | method | 56 |  | `(DeviceErrorState).DeviceLost` | done |
| `DeviceErrorState::observe_error` | method | 70 |  | `(DeviceErrorState).ObserveError` | done |
| `DeviceErrorState::current_generation` | method | 81 |  | `(DeviceErrorState).CurrentGeneration` | done |
| `WgpuBackend` | enum | 87 |  | `WgpuBackend` | done |
| `WebBackendPreference` | enum | 95 | wasm | `WebBackendPreference` | blocked:M4-web-wgpu-js |
| `PreparedWebGraphics` | struct | 103 | wasm | `PreparedWebGraphics` | blocked:M4-web-wgpu-js |
| `CompositorGpuHint` | struct | 126 |  | `CompositorGPUHint` | done |
| `WgpuContext::new` | assoc-fn | 133 | wasm | `NewWgpuContext` | dev:D-WGPUCONFIGURE |
| `WgpuContext::new_rejecting_software` | assoc-fn | 143 | wasm | `NewWgpuContextRejectingSoftware` | dev:D-WGPUCONFIGURE |
| `WgpuContext::supports_transient_attachments` | method | 341 |  | `(WgpuContext).SupportsTransientAttachments` | dev:L-WGPUVER |
| `WgpuContext::new_web` | assoc-fn | 354 | wasm | `NewWgpuContextWeb` | blocked:M4-web-wgpu-js |
| `WgpuContext::instance` | assoc-fn | 486 | wasm | `WgpuContextInstance` | done |
| `WgpuContext::check_compatible_with_surface` | method | 496 |  | `(WgpuContext).CheckCompatibleWithSurface` | done |
| `WgpuContext::backend` | method | 731 |  | `(WgpuContext).Backend` | done |
| `WgpuContext::uses_texture_instance_data` | method | 735 |  | `(WgpuContext).UsesTextureInstanceData` | dev:D-WGPUCAPS |
| `WgpuContext::supports_dual_source_blending` | method | 749 |  | `(WgpuContext).SupportsDualSourceBlending` | done |
| `WgpuContext::color_texture_format` | method | 753 |  | `(WgpuContext).ColorTextureFormat` | done |
| `WgpuContext::device_lost` | method | 759 |  | `(WgpuContext).DeviceLost` | done |
| `WgpuContext::errors` | method | 764 |  | `(WgpuContext).Errors` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `device_errors_are_observed_independently` | test | 791 |  | `TestWgpuContext_DeviceErrorsAreObservedIndependently` | done |
| `device_errors_coalesce_and_new_observers_see_latest` | test | 810 |  | `TestWgpuContext_DeviceErrorsCoalesceAndNewObserversSeeLatest` | done |
| `test_parse_device_id` | test | 832 |  | `TestWgpuContext_ParseDeviceID` | done |

## gpui_wgpu/src/wgpu_renderer.rs

Go target: `render/wgpu/wgpu_renderer.go` (package `wgpu`), tests in `wgpu_renderer_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WgpuSurfaceConfig` | struct | 113 |  | `WgpuSurfaceConfig` | done |
| `GpuContext` | type | 182 |  | `GPUContext` | done |
| `WgpuRenderer` | struct | 253 |  | `WgpuRenderer` | done |
| `WgpuRenderer::new` | assoc-fn | 299 | wasm | `NewWgpuRenderer` | done |
| `WgpuRenderer::new_from_canvas` | assoc-fn | 365 | wasm | `NewWgpuRendererFromCanvas` | blocked:M4-web-wgpu-js |
| `WgpuRenderer::new_from_surface` | assoc-fn | 379 | wasm | `NewWgpuRendererFromSurface` | done |
| `WgpuRenderer::update_drawable_size` | method | 1003 |  | `(WgpuRenderer).UpdateDrawableSize` | done |
| `WgpuRenderer::set_subpixel_layout` | method | 1028 |  | `(WgpuRenderer).SetSubpixelLayout` | done |
| `WgpuRenderer::update_transparency` | method | 1035 |  | `(WgpuRenderer).UpdateTransparency` | done |
| `WgpuRenderer::viewport_size` | method | 1058 |  | `(WgpuRenderer).ViewportSize` | done |
| `WgpuRenderer::sprite_atlas` | method | 1065 |  | `(WgpuRenderer).SpriteAtlas` | done |
| `WgpuRenderer::supports_dual_source_blending` | method | 1069 |  | `(WgpuRenderer).SupportsDualSourceBlending` | done |
| `WgpuRenderer::gpu_specs` | method | 1075 |  | `(WgpuRenderer).GPUSpecs` | done |
| `WgpuRenderer::max_texture_size` | method | 1085 |  | `(WgpuRenderer).MaxTextureSize` | done |
| `WgpuRenderer::draw` | method | 1089 |  | `(WgpuRenderer).Draw` | done |
| `WgpuRenderer::unconfigure_surface` | method | 2079 |  | `(WgpuRenderer).UnconfigureSurface` | done |
| `WgpuRenderer::replace_surface` | method | 2098 | wasm | `(WgpuRenderer).ReplaceSurface` | done |
| `WgpuRenderer::destroy` | method | 2149 |  | `(WgpuRenderer).Destroy` | done |
| `WgpuRenderer::device_lost` | method | 2156 |  | `(WgpuRenderer).DeviceLost` | done |
| `WgpuRenderer::needs_redraw` | method | 2162 |  | `(WgpuRenderer).NeedsRedraw` | done |
| `WgpuRenderer::recover` | method | 2174 | wasm | `(WgpuRenderer).Recover` | done |
| `WgpuHeadlessRenderer` | struct | 2278 | wasm,feat:bench-support,feat:test-support | `WgpuHeadlessRenderer` | done |
| `WgpuHeadlessRenderer::new` | assoc-fn | 2290 | wasm,feat:bench-support,feat:test-support | `NewWgpuHeadlessRenderer` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `headless_renderer_draws_quads_with_distinct_colors` | test | 2640 | linux | `TestWgpuRenderer_HeadlessRendererDrawsQuadsWithDistinctColors` | done |
| `headless_renderer_captures_each_requested_size` | test | 2658 | linux | `TestWgpuRenderer_HeadlessRendererCapturesEachRequestedSize` | done |
| `headless_renderer_reuses_target_for_same_size` | test | 2684 | linux | `TestWgpuRenderer_HeadlessRendererReusesTargetForSameSize` | done |
| `headless_renderer_rejects_invalid_sizes` | test | 2707 | linux | `TestWgpuRenderer_HeadlessRendererRejectsInvalidSizes` | done |
| `webgl_shader_is_valid_wgsl_without_storage_buffers` | test | 2736 |  | `TestWgpuRenderer_WebglShaderIsValidWgslWithoutStorageBuffers` | done |
| `storage_buffer_shader_is_valid_wgsl` | test | 2742 |  | `TestWgpuRenderer_StorageBufferShaderIsValidWgsl` | done |
| `subpixel_storage_buffer_shader_is_valid_wgsl` | test | 2747 |  | `TestWgpuRenderer_SubpixelStorageBufferShaderIsValidWgsl` | done |
| `subpixel_webgl_shader_is_valid_wgsl_without_storage_buffers` | test | 2755 |  | `TestWgpuRenderer_SubpixelWebglShaderIsValidWgslWithoutStorageBuffers` | done |
| `webgl_record_sizes_match_shader_word_strides` | test | 2771 |  | `TestWgpuRenderer_WebglRecordSizesMatchShaderWordStrides` | done |
| `webgl_quad_layout_matches_fixed_decoder` | test | 2784 |  | `TestWgpuRenderer_WebglQuadLayoutMatchesFixedDecoder` | done |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-WGPUCAPS | wgpu-native's C API has no adapter texture-format-features or downlevel-capabilities query; the webgpu facade answers `GetTextureFormatFeatures` from the usages WebGPU guarantees for the formats gpui uses (Bgra8Unorm, Rgba8Unorm, R8Unorm, Rgba32Uint, Depth32Float) and `GetDownlevelCapabilities` on GL from `MaxStorageBuffersPerShaderStage > 0` (VERTEX_STORAGE and FRAGMENT_STORAGE together). Affects `select_color_texture_format`, `headless_target_format` and `requires_texture_instance_data` | `gpui_wgpu/src/wgpu_context.rs:279-293`, `:693-747` | `render/wgpu/internal/gpu/webgpu/webgpu.go` |
| D-WGPUCONFIGURE | wgpu-native v29 handles every `wgpuSurfaceConfigure` error with `handle_error_fatal` (process abort) instead of the device error sink, and the binding has no uncaptured-error callback. The webgpu facade's `Surface.Configure` therefore runs wgpu-core's `configure_surface` / `validate_surface_configuration` checks in Go first (view formats differing only in sRGB-ness, TooLarge against the device's `max_texture_dimension_2d`, present mode with Undefined as Fifo, format, alpha mode with Auto falling back to Opaque / Inherit, ZeroArea, in that order and with wgpu-core's messages) and reports a failure into the device's error scope without calling wgpu-native, so `try_adapter_with_surface` falls through to the next adapter as upstream. Not pre-checked, still fatal: the usage check (the binding drops `SurfaceCapabilities::usages`), the SURFACE_VIEW_FORMATS downlevel flag, colour space, and hal-level configure / GPU-wait failures | `gpui_wgpu/src/wgpu_context.rs:657-691`; `wgpu-core/src/device/resource.rs` `configure_surface`, `device/surface_config.rs`; wgpu-native v29.0.1.1 `src/lib.rs` `wgpuSurfaceConfigure` | `render/wgpu/wgpu_context.go`, `render/wgpu/internal/gpu/webgpu/webgpu.go` |
| D-WGPUATLASBOUNDS | `insert` rejects a tile whose bytes are fewer than width x height x bytes-per-pixel (1 for monochrome, 4 otherwise) with an error before allocating, instead of queueing a short upload that fails later in `write_texture`; `allocate` returns None for a tile larger than `max_texture_dimension_2d` without creating a texture, and drops a freshly pushed texture when its first allocation fails, where upstream keeps an empty clamped texture in storage forever | `gpui_wgpu/src/wgpu_atlas.rs:128-137`, `:162-185`, `:256-265` | `render/wgpu/wgpu_atlas.go` |
