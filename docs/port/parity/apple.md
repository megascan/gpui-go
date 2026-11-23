# Parity: apple

Scope: gpui_apple. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_apple/build.rs

Go target: `-` (package `-`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `run` | fn | 19 | macos,ios | `Run` | todo |

## gpui_apple/src/dispatcher.rs

Go target: `platform/apple/dispatcher.go` (package `apple`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AppleDispatcher` | struct | 20 |  | `AppleDispatcher` | todo |
| `AppleDispatcher::new` | assoc-fn | 23 |  | `NewAppleDispatcher` | todo |
| `AppleActivity` | struct | 84 |  | `AppleActivity` | todo |
| `AppleActivity::begin` | assoc-fn | 92 |  | `AppleActivityBegin` | todo |

## gpui_apple/src/display_link.rs

Go target: `platform/apple/display_link.go` (package `apple`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MetalFrameSource` | struct | 52 |  | `MetalFrameSource` | todo |
| `MetalFrameSource::is_available` | assoc-fn | 61 |  | `MetalFrameSourceIsAvailable` | todo |
| `MetalFrameSource::new` | assoc-fn | 67 |  | `NewMetalFrameSource` | todo |
| `MetalFrameSource::set_paused` | method | 91 |  | `(MetalFrameSource).SetPaused` | todo |
| `MetalFrameSource::is_paused` | method | 95 |  | `(MetalFrameSource).IsPaused` | todo |

## gpui_apple/src/gpui_apple.rs

Go target: `platform/apple/gpui_apple.go` (package `apple`), tests in `gpui_apple_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `main_thread_marker` | fn | 27 |  | `MainThreadMarker` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `main_thread_marker_rejects_background_threads` | test | 39 |  | `TestGpuiApple_MainThreadMarkerRejectsBackgroundThreads` | todo |

## gpui_apple/src/keychain.rs

Go target: `platform/apple/keychain.go` (package `apple`), tests in `keychain_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `KeychainOperation` | enum | 10 |  | `KeychainOperation` | todo |
| `KeychainError` | enum | 18 |  | `KeychainError` | todo |
| `write_credentials` | fn | 30 |  | `WriteCredentials` | todo |
| `write_credentials_if_not_exists` | fn | 70 |  | `WriteCredentialsIfNotExists` | todo |
| `read_credentials` | fn | 102 |  | `ReadCredentials` | todo |
| `delete_credentials` | fn | 168 |  | `DeleteCredentials` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `read_distinguishes_absence_and_cancellation_from_errors` | test | 216 |  | `TestKeychain_ReadDistinguishesAbsenceAndCancellationFromErrors` | todo |
| `platform_error_keeps_native_operation_and_status` | test | 235 |  | `TestKeychain_PlatformErrorKeepsNativeOperationAndStatus` | todo |
| `delete_preserves_macos_missing_item_error` | test | 249 |  | `TestKeychain_DeletePreservesMacosMissingItemError` | todo |
| `create_reports_existing_items_without_error` | test | 272 |  | `TestKeychain_CreateReportsExistingItemsWithoutError` | todo |

## gpui_apple/src/metal_atlas.rs

Go target: `platform/apple/metal_atlas.go` (package `apple`), tests in `metal_atlas_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MetalAtlas` | struct | 15 |  | `MetalAtlas` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_remove_clears_stale_keys_from_tiles_by_key` | test | 348 |  | `TestMetalAtlas_RemoveClearsStaleKeysFromTilesByKey` | todo |
| `test_metal_texture_is_none_after_last_tile_removed` | test | 389 |  | `TestMetalAtlas_MetalTextureIsNoneAfterLastTileRemoved` | todo |
| `test_remove_deallocates_tile_space_for_reuse` | test | 412 |  | `TestMetalAtlas_RemoveDeallocatesTileSpaceForReuse` | todo |
| `test_remove_nonexistent_key_is_noop` | test | 440 |  | `TestMetalAtlas_RemoveNonexistentKeyIsNoop` | todo |
| `test_invalid_upload_does_not_allocate` | test | 449 |  | `TestMetalAtlas_InvalidUploadDoesNotAllocate` | todo |
| `test_texture_upload_preserves_pixel_layout` | test | 492 |  | `TestMetalAtlas_TextureUploadPreservesPixelLayout` | todo |

## gpui_apple/src/metal_renderer.rs

Go target: `platform/apple/metal_renderer.go` (package `apple`), tests in `metal_renderer_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Context` | type | 45 |  | `Context` | todo |
| `Renderer` | type | 46 |  | `Renderer` | todo |
| `new_renderer` | fn | 48 |  | `NewRenderer` | todo |
| `InstanceBufferPool` | struct | 56 |  | `InstanceBufferPool` | todo |
| `MetalRenderer` | struct | 123 |  | `MetalRenderer` | todo |
| `MetalRenderer::new` | assoc-fn | 158 |  | `NewMetalRenderer` | todo |
| `MetalRenderer::from_layer` | assoc-fn | 172 |  | `MetalRendererFromLayer` | todo |
| `MetalRenderer::new_headless` | assoc-fn | 225 | feat:bench-support,feat:test-support | `NewMetalRendererHeadless` | todo |
| `MetalRenderer::layer` | method | 424 |  | `(MetalRenderer).Layer` | todo |
| `MetalRenderer::layer_ptr` | method | 428 |  | `(MetalRenderer).LayerPtr` | todo |
| `MetalRenderer::sprite_atlas` | method | 435 |  | `(MetalRenderer).SpriteAtlas` | todo |
| `MetalRenderer::set_frame_drawable` | method | 451 |  | `(MetalRenderer).SetFrameDrawable` | todo |
| `MetalRenderer::set_display_link_attached` | method | 471 |  | `(MetalRenderer).SetDisplayLinkAttached` | todo |
| `MetalRenderer::is_display_link_attached` | method | 479 |  | `(MetalRenderer).IsDisplayLinkAttached` | todo |
| `MetalRenderer::presents_with_transaction` | method | 483 |  | `(MetalRenderer).PresentsWithTransaction` | todo |
| `MetalRenderer::set_presents_with_transaction` | method | 487 |  | `(MetalRenderer).SetPresentsWithTransaction` | todo |
| `MetalRenderer::update_drawable_size` | method | 494 |  | `(MetalRenderer).UpdateDrawableSize` | todo |
| `MetalRenderer::update_transparency` | method | 500 |  | `(MetalRenderer).UpdateTransparency` | todo |
| `MetalRenderer::destroy` | method | 507 |  | `(MetalRenderer).Destroy` | todo |
| `MetalRenderer::draw` | method | 511 |  | `(MetalRenderer).Draw` | todo |
| `MetalRenderer::render_to_image` | method | 630 | feat:test-support | `(MetalRenderer).RenderToImage` | todo |
| `MetalRenderer::render_scene_to_image` | method | 649 | feat:bench-support,feat:test-support | `(MetalRenderer).RenderSceneToImage` | todo |
| `MetalRenderer::render_scene` | method | 719 | feat:bench-support,feat:test-support | `(MetalRenderer).RenderScene` | todo |
| `SurfaceBounds` | struct | 1695 |  | `SurfaceBounds` | todo |
| `MetalHeadlessRenderer` | struct | 1701 | feat:bench-support,feat:test-support | `MetalHeadlessRenderer` | todo |
| `MetalHeadlessRenderer::new` | assoc-fn | 1707 | feat:bench-support,feat:test-support | `NewMetalHeadlessRenderer` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `render_to_image_rejects_zero_sized_layer` | test | 1746 |  | `TestMetalRenderer_RenderToImageRejectsZeroSizedLayer` | todo |
| `display_link_mode_consumes_and_clears_drawables` | test | 1756 |  | `TestMetalRenderer_DisplayLinkModeConsumesAndClearsDrawables` | todo |
| `render_core_video_surface` | test | 1801 |  | `TestMetalRenderer_RenderCoreVideoSurface` | todo |

## gpui_apple/src/open_type.rs

Go target: `platform/apple/open_type.go` (package `apple`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `apply_features_and_fallbacks` | fn | 28 |  | `ApplyFeaturesAndFallbacks` | todo |

## gpui_apple/src/process_info.rs

Go target: `platform/apple/process_info.go` (package `apple`), tests in `process_info_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `thermal_state` | fn | 11 |  | `ThermalState` | todo |
| `power_mode` | fn | 20 |  | `PowerMode` | todo |
| `observe_thermal_state` | fn | 29 |  | `ObserveThermalState` | todo |
| `observe_power_mode` | fn | 43 |  | `ObservePowerMode` | todo |
| `ProcessInfoObserver` | struct | 54 |  | `ProcessInfoObserver` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `notifications_are_deferred_and_cancelled_on_drop` | gpui::test | 129 |  | `TestProcessInfo_NotificationsAreDeferredAndCancelledOnDrop` | todo |

## gpui_apple/src/text_system.rs

Go target: `platform/apple/text_system.go` (package `apple`), tests in `text_system_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AppleTextSystem` | struct | 63 |  | `AppleTextSystem` | todo |
| `AppleTextSystem::new` | assoc-fn | 84 |  | `NewAppleTextSystem` | todo |
| `family_name` | fn | 752 |  | `FamilyName` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_reload_family_with_features_and_fallbacks` | test | 791 |  | `TestTextSystem_ReloadFamilyWithFeaturesAndFallbacks` | todo |
| `test_layout_line_sets_a_lone_run_at_its_exact_size` | test | 814 |  | `TestTextSystem_LayoutLineSetsALoneRunAtItsExactSize` | todo |
| `test_layout_line_breaks_ligatures_between_runs` | test | 867 |  | `TestTextSystem_LayoutLineBreaksLigaturesBetweenRuns` | todo |
| `test_layout_line_bom_char` | test | 934 |  | `TestTextSystem_LayoutLineBomChar` | todo |
| `test_layout_line_zwnj_insertion` | test | 978 |  | `TestTextSystem_LayoutLineZwnjInsertion` | todo |
| `test_layout_line_zwnj_edge_cases` | test | 1029 |  | `TestTextSystem_LayoutLineZwnjEdgeCases` | todo |

