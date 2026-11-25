# Parity: apple

Scope: gpui_apple. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_apple/build.rs

Go target: `-` (package `-`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `run` | fn | 19 | macos,ios | `internal/cmd/metalhdr`, `stitchedShaderSource` | dev:D-APPLE-METALHDR |

## gpui_apple/src/dispatcher.rs

Go target: `platform/apple/dispatcher.go` (package `apple`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AppleDispatcher` | struct | 20 |  | `AppleDispatcher` | done |
| `AppleDispatcher::new` | assoc-fn | 23 |  | `NewAppleDispatcher` | done |
| `AppleActivity` | struct | 84 |  | `AppleActivity` | done |
| `AppleActivity::begin` | assoc-fn | 92 |  | `AppleActivityBegin` | done |

## gpui_apple/src/display_link.rs

Go target: `platform/apple/display_link.go` (package `apple`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MetalFrameSource` | struct | 52 |  | `MetalFrameSource` | done |
| `MetalFrameSource::is_available` | assoc-fn | 61 |  | `MetalFrameSourceIsAvailable` | done |
| `MetalFrameSource::new` | assoc-fn | 67 |  | `NewMetalFrameSource` | done |
| `MetalFrameSource::set_paused` | method | 91 |  | `(MetalFrameSource).SetPaused` | done |
| `MetalFrameSource::is_paused` | method | 95 |  | `(MetalFrameSource).IsPaused` | done |

## gpui_apple/src/gpui_apple.rs

Go target: `platform/apple/gpui_apple.go` (package `apple`), tests in `gpui_apple_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `main_thread_marker` | fn | 27 |  | `MainThreadMarker` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `main_thread_marker_rejects_background_threads` | test | 39 |  | `TestGpuiApple_MainThreadMarkerRejectsBackgroundThreads` | done |

## gpui_apple/src/keychain.rs

Go target: `platform/apple/keychain.go` (package `apple`), tests in `keychain_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `KeychainOperation` | enum | 10 |  | `KeychainOperation` | done |
| `KeychainError` | enum | 18 |  | `KeychainError` | done |
| `write_credentials` | fn | 30 |  | `WriteCredentials` | done |
| `write_credentials_if_not_exists` | fn | 70 |  | `WriteCredentialsIfNotExists` | done |
| `read_credentials` | fn | 102 |  | `ReadCredentials` | done |
| `delete_credentials` | fn | 168 |  | `DeleteCredentials` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `read_distinguishes_absence_and_cancellation_from_errors` | test | 216 |  | `TestKeychain_ReadDistinguishesAbsenceAndCancellationFromErrors` | done |
| `platform_error_keeps_native_operation_and_status` | test | 235 |  | `TestKeychain_PlatformErrorKeepsNativeOperationAndStatus` | done |
| `delete_preserves_macos_missing_item_error` | test | 249 |  | `TestKeychain_DeletePreservesMacosMissingItemError` | done |
| `create_reports_existing_items_without_error` | test | 272 |  | `TestKeychain_CreateReportsExistingItemsWithoutError` | done |

## gpui_apple/src/metal_atlas.rs

Go target: `platform/apple/metal_atlas.go` (package `apple`), tests in `metal_atlas_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MetalAtlas` | struct | 15 |  | `MetalAtlas` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_remove_clears_stale_keys_from_tiles_by_key` | test | 348 |  | `TestMetalAtlas_RemoveClearsStaleKeysFromTilesByKey` | done |
| `test_metal_texture_is_none_after_last_tile_removed` | test | 389 |  | `TestMetalAtlas_MetalTextureIsNoneAfterLastTileRemoved` | done |
| `test_remove_deallocates_tile_space_for_reuse` | test | 412 |  | `TestMetalAtlas_RemoveDeallocatesTileSpaceForReuse` | done |
| `test_remove_nonexistent_key_is_noop` | test | 440 |  | `TestMetalAtlas_RemoveNonexistentKeyIsNoop` | done |
| `test_invalid_upload_does_not_allocate` | test | 449 |  | `TestMetalAtlas_InvalidUploadDoesNotAllocate` | done |
| `test_texture_upload_preserves_pixel_layout` | test | 492 |  | `TestMetalAtlas_TextureUploadPreservesPixelLayout` | done |

## gpui_apple/src/metal_renderer.rs

Go target: `platform/apple/metal_renderer.go` (package `apple`), tests in `metal_renderer_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Context` | type | 45 |  | `Context` | done |
| `Renderer` | type | 46 |  | `Renderer` | done |
| `new_renderer` | fn | 48 |  | `NewRenderer` | done |
| `InstanceBufferPool` | struct | 56 |  | `InstanceBufferPool` | done |
| `MetalRenderer` | struct | 123 |  | `MetalRenderer` | done |
| `MetalRenderer::new` | assoc-fn | 158 |  | `NewMetalRenderer` | done |
| `MetalRenderer::from_layer` | assoc-fn | 172 |  | `MetalRendererFromLayer` | done |
| `MetalRenderer::new_headless` | assoc-fn | 225 | feat:bench-support,feat:test-support | `NewMetalRendererHeadless` | done |
| `MetalRenderer::layer` | method | 424 |  | `(MetalRenderer).Layer` | done |
| `MetalRenderer::layer_ptr` | method | 428 |  | `(MetalRenderer).LayerPtr` | done |
| `MetalRenderer::sprite_atlas` | method | 435 |  | `(MetalRenderer).SpriteAtlas` | done |
| `MetalRenderer::set_frame_drawable` | method | 451 |  | `(MetalRenderer).SetFrameDrawable` | done |
| `MetalRenderer::set_display_link_attached` | method | 471 |  | `(MetalRenderer).SetDisplayLinkAttached` | done |
| `MetalRenderer::is_display_link_attached` | method | 479 |  | `(MetalRenderer).IsDisplayLinkAttached` | done |
| `MetalRenderer::presents_with_transaction` | method | 483 |  | `(MetalRenderer).PresentsWithTransaction` | done |
| `MetalRenderer::set_presents_with_transaction` | method | 487 |  | `(MetalRenderer).SetPresentsWithTransaction` | done |
| `MetalRenderer::update_drawable_size` | method | 494 |  | `(MetalRenderer).UpdateDrawableSize` | done |
| `MetalRenderer::update_transparency` | method | 500 |  | `(MetalRenderer).UpdateTransparency` | done |
| `MetalRenderer::destroy` | method | 507 |  | `(MetalRenderer).Destroy` | done |
| `MetalRenderer::draw` | method | 511 |  | `(MetalRenderer).Draw` | done |
| `MetalRenderer::render_to_image` | method | 630 | feat:test-support | `(MetalRenderer).RenderToImage` | done |
| `MetalRenderer::render_scene_to_image` | method | 649 | feat:bench-support,feat:test-support | `(MetalRenderer).RenderSceneToImage` | done |
| `MetalRenderer::render_scene` | method | 719 | feat:bench-support,feat:test-support | `(MetalRenderer).RenderScene` | done |
| `SurfaceBounds` | struct | 1695 |  | `SurfaceBounds` | done |
| `MetalHeadlessRenderer` | struct | 1701 | feat:bench-support,feat:test-support | `MetalHeadlessRenderer` | done |
| `MetalHeadlessRenderer::new` | assoc-fn | 1707 | feat:bench-support,feat:test-support | `NewMetalHeadlessRenderer` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `render_to_image_rejects_zero_sized_layer` | test | 1746 |  | `TestMetalRenderer_RenderToImageRejectsZeroSizedLayer` | done |
| `display_link_mode_consumes_and_clears_drawables` | test | 1756 |  | `TestMetalRenderer_DisplayLinkModeConsumesAndClearsDrawables` | done |
| `render_core_video_surface` | test | 1801 |  | `TestMetalRenderer_RenderCoreVideoSurface` | done |

## gpui_apple/src/open_type.rs

Go target: `platform/apple/open_type.go` (package `apple`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `apply_features_and_fallbacks` | fn | 28 |  | `ApplyFeaturesAndFallbacks` | done |

## gpui_apple/src/process_info.rs

Go target: `platform/apple/process_info.go` (package `apple`), tests in `process_info_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `thermal_state` | fn | 11 |  | `ThermalState` | done |
| `power_mode` | fn | 20 |  | `PowerMode` | done |
| `observe_thermal_state` | fn | 29 |  | `ObserveThermalState` | done |
| `observe_power_mode` | fn | 43 |  | `ObservePowerMode` | done |
| `ProcessInfoObserver` | struct | 54 |  | `ProcessInfoObserver` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `notifications_are_deferred_and_cancelled_on_drop` | gpui::test | 129 |  | `TestProcessInfo_NotificationsAreDeferredAndCancelledOnDrop` | done |

## gpui_apple/src/text_system.rs

Go target: `platform/apple/text_system.go` (package `apple`), tests in `text_system_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AppleTextSystem` | struct | 63 |  | `AppleTextSystem` | done |
| `AppleTextSystem::new` | assoc-fn | 84 |  | `NewAppleTextSystem` | done |
| `family_name` | fn | 752 |  | `FamilyName` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_reload_family_with_features_and_fallbacks` | test | 791 |  | `TestTextSystem_ReloadFamilyWithFeaturesAndFallbacks` | done |
| `test_layout_line_sets_a_lone_run_at_its_exact_size` | test | 814 |  | `TestTextSystem_LayoutLineSetsALoneRunAtItsExactSize` | done |
| `test_layout_line_breaks_ligatures_between_runs` | test | 867 |  | `TestTextSystem_LayoutLineBreaksLigaturesBetweenRuns` | done |
| `test_layout_line_bom_char` | test | 934 |  | `TestTextSystem_LayoutLineBomChar` | done |
| `test_layout_line_zwnj_insertion` | test | 978 |  | `TestTextSystem_LayoutLineZwnjInsertion` | done |
| `test_layout_line_zwnj_edge_cases` | test | 1029 |  | `TestTextSystem_LayoutLineZwnjEdgeCases` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-APPLE-METALHDR | The build script becomes `internal/cmd/metalhdr` (go:generate in platform/apple/shaders.go, output committed as shaders_scene_gen.go) plus the runtime_shaders path (`stitchedShaderSource` compiled with newLibraryWithSource at start); the prebuilt metallib path is not ported. cbindgen gets the six sources concatenated into one file with inner attributes stripped, where build.rs passes them through Builder::with_src | `gpui_apple/build.rs:19-120`, `gpui_apple/src/metal_renderer.rs:37-40` | Go has no build scripts and cannot run the Metal compiler at build time; DESIGN 8.3 |
| D-APPLE-NESTEDLOOP | Main-queue runnables run only when `scheduler.CallbackDepth()` is 1; one that arrives inside a nested AppKit loop entered from a Go callback (live resize or title-bar drag through `sendEvent:`, `performWindowDragWithEvent:`, menu tracking, modal sessions) is queued and retried every 8 ms until the loop returns, and later main-queue runnables run the queue first | `gpui_apple/src/dispatcher.rs:dispatch_on_main_thread`, `trampoline` | resuming an `iter.Pull` coroutine (D-ASYNC) at a different cgo-callback depth than it started at is a fatal runtime error |
