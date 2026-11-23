# Parity: wgpu

Scope: gpui_wgpu. Milestone: M1/M2. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_wgpu/src/cosmic_text_system.rs

Go target: `text/gotext/text_system.go` (package `gotext`), tests in `text_system_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CosmicTextSystem` | struct | 24 |  | `CosmicTextSystem` | todo |
| `CosmicTextSystem::font_weight_and_style` | method | 96 |  | `(CosmicTextSystem).FontWeightAndStyle` | todo |
| `CosmicTextSystem::missing_glyphs` | method | 120 |  | `(CosmicTextSystem).MissingGlyphs` | todo |
| `CosmicTextSystem::new` | assoc-fn | 131 |  | `NewCosmicTextSystem` | todo |
| `CosmicTextSystem::new_without_system_fonts` | assoc-fn | 147 |  | `NewCosmicTextSystemWithoutSystemFonts` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `all_font_names_tracks_available_families` | test | 1222 |  | `TestCosmicTextSystem_AllFontNamesTracksAvailableFamilies` | todo |
| `cached_font_resolution_does_not_wait_for_cold_lookups` | test | 1277 |  | `TestCosmicTextSystem_CachedFontResolutionDoesNotWaitForColdLookups` | todo |
| `adding_fonts_retries_cached_font_failures` | test | 1313 |  | `TestCosmicTextSystem_AddingFontsRetriesCachedFontFailures` | todo |
| `font_properties_describe_the_selected_face` | test | 1341 |  | `TestCosmicTextSystem_FontPropertiesDescribeTheSelectedFace` | todo |
| `combining_marks_are_positioned_over_their_base` | test | 1375 |  | `TestCosmicTextSystem_CombiningMarksArePositionedOverTheirBase` | todo |
| `shape_text_with_mixed_direction_paragraphs` | test | 1400 |  | `TestCosmicTextSystem_ShapeTextWithMixedDirectionParagraphs` | todo |
| `reports_graphemes_that_exhaust_font_fallback` | test | 1422 | feat:test-support | `TestCosmicTextSystem_ReportsGraphemesThatExhaustFontFallback` | todo |
| `combines_missing_glyphs_from_one_grapheme` | test | 1458 |  | `TestCosmicTextSystem_CombinesMissingGlyphsFromOneGrapheme` | todo |
| `adding_fonts_invalidates_cached_line_layouts` | test | 1478 |  | `TestCosmicTextSystem_AddingFontsInvalidatesCachedLineLayouts` | todo |
| `layout_line_with_mixed_direction_paragraphs` | test | 1513 |  | `TestCosmicTextSystem_LayoutLineWithMixedDirectionParagraphs` | todo |
| `layout_line_with_separators_at_line_edges` | test | 1536 |  | `TestCosmicTextSystem_LayoutLineWithSeparatorsAtLineEdges` | todo |
| `layout_line_keeps_indices_and_positions_ordered_across_paragraphs` | test | 1558 |  | `TestCosmicTextSystem_LayoutLineKeepsIndicesAndPositionsOrderedAcrossParagraphs` | todo |
| `layout_line_with_font_run_straddling_a_separator` | test | 1584 |  | `TestCosmicTextSystem_LayoutLineWithFontRunStraddlingASeparator` | todo |
| `layout_line_without_separators_takes_fast_path` | test | 1610 |  | `TestCosmicTextSystem_LayoutLineWithoutSeparatorsTakesFastPath` | todo |
| `index_for_position_past_line_width` | test | 1633 |  | `TestCosmicTextSystem_IndexForPositionPastLineWidth` | todo |
| `paragraph_separator_detection` | test | 1672 |  | `TestCosmicTextSystem_ParagraphSeparatorDetection` | todo |
| `font_runs_are_clipped_to_segment` | test | 1690 |  | `TestCosmicTextSystem_FontRunsAreClippedToSegment` | todo |
| `primary_wins_over_current_fallback_when_primary_covers` | test | 1727 |  | `TestCosmicTextSystem_PrimaryWinsOverCurrentFallbackWhenPrimaryCovers` | todo |
| `primary_preferred_over_fallback_when_both_cover` | test | 1738 |  | `TestCosmicTextSystem_PrimaryPreferredOverFallbackWhenBothCover` | todo |
| `falls_through_chain_in_order` | test | 1746 |  | `TestCosmicTextSystem_FallsThroughChainInOrder` | todo |
| `no_coverage_returns_primary` | test | 1758 |  | `TestCosmicTextSystem_NoCoverageReturnsPrimary` | todo |
| `empty_chain_always_returns_primary` | test | 1771 |  | `TestCosmicTextSystem_EmptyChainAlwaysReturnsPrimary` | todo |
| `slot_font_id_resolution` | test | 1779 |  | `TestCosmicTextSystem_SlotFontIDResolution` | todo |
| `run_spans_with_no_chain_emit_one_primary_span` | test | 1788 |  | `TestCosmicTextSystem_RunSpansWithNoChainEmitOnePrimarySpan` | todo |
| `run_spans_use_byte_offsets_for_multibyte_chars` | test | 1798 |  | `TestCosmicTextSystem_RunSpansUseByteOffsetsForMultibyteChars` | todo |
| `run_spans_respect_run_offset` | test | 1823 |  | `TestCosmicTextSystem_RunSpansRespectRunOffset` | todo |
| `run_spans_keep_combining_marks_with_base_in_fallback` | test | 1845 |  | `TestCosmicTextSystem_RunSpansKeepCombiningMarksWithBaseInFallback` | todo |
| `run_spans_keep_zwj_inside_emoji_cluster` | test | 1865 |  | `TestCosmicTextSystem_RunSpansKeepZwjInsideEmojiCluster` | todo |
| `run_spans_collapse_adjacent_same_slot` | test | 1877 |  | `TestCosmicTextSystem_RunSpansCollapseAdjacentSameSlot` | todo |
| `run_spans_empty_run_returns_no_spans` | test | 1893 |  | `TestCosmicTextSystem_RunSpansEmptyRunReturnsNoSpans` | todo |

## gpui_wgpu/src/wgpu_atlas.rs

Go target: `render/wgpu/wgpu_atlas.go` (package `wgpu`), tests in `wgpu_atlas_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WgpuAtlas` | struct | 23 |  | `WgpuAtlas` | todo |
| `WgpuTextureInfo` | struct | 41 |  | `WgpuTextureInfo` | todo |
| `WgpuAtlas::new` | assoc-fn | 48 |  | `NewWgpuAtlas` | todo |
| `WgpuAtlas::from_context` | assoc-fn | 65 |  | `WgpuAtlasFromContext` | todo |
| `WgpuAtlas::before_frame` | method | 73 |  | `(WgpuAtlas).BeforeFrame` | todo |
| `WgpuAtlas::get_texture_info` | method | 82 |  | `(WgpuAtlas).GetTextureInfo` | todo |
| `WgpuAtlas::clear` | method | 93 |  | `(WgpuAtlas).Clear` | todo |
| `WgpuAtlas::handle_device_lost` | method | 102 |  | `(WgpuAtlas).HandleDeviceLost` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `before_frame_skips_uploads_for_removed_texture` | test | 438 | wasm | `TestWgpuAtlas_BeforeFrameSkipsUploadsForRemovedTexture` | todo |
| `remove_deallocates_tile_space_for_reuse` | test | 462 | wasm | `TestWgpuAtlas_RemoveDeallocatesTileSpaceForReuse` | todo |
| `reused_texture_id_has_new_generation` | test | 506 | wasm | `TestWgpuAtlas_ReusedTextureIDHasNewGeneration` | todo |
| `swizzle_upload_data_preserves_bgra_uploads` | test | 552 | wasm | `TestWgpuAtlas_SwizzleUploadDataPreservesBgraUploads` | todo |
| `swizzle_upload_data_converts_bgra_to_rgba` | test | 561 | wasm | `TestWgpuAtlas_SwizzleUploadDataConvertsBgraToRgba` | todo |

## gpui_wgpu/src/wgpu_context.rs

Go target: `render/wgpu/wgpu_context.go` (package `wgpu`), tests in `wgpu_context_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WgpuContext` | struct | 13 |  | `WgpuContext` | todo |
| `DeviceErrorState` | struct | 31 |  | `DeviceErrorState` | todo |
| `DeviceErrorState::device_lost` | method | 56 |  | `(DeviceErrorState).DeviceLost` | todo |
| `DeviceErrorState::observe_error` | method | 70 |  | `(DeviceErrorState).ObserveError` | todo |
| `DeviceErrorState::current_generation` | method | 81 |  | `(DeviceErrorState).CurrentGeneration` | todo |
| `WgpuBackend` | enum | 87 |  | `WgpuBackend` | todo |
| `WebBackendPreference` | enum | 95 | wasm | `WebBackendPreference` | todo |
| `PreparedWebGraphics` | struct | 103 | wasm | `PreparedWebGraphics` | todo |
| `CompositorGpuHint` | struct | 126 |  | `CompositorGPUHint` | todo |
| `WgpuContext::new` | assoc-fn | 133 | wasm | `NewWgpuContext` | todo |
| `WgpuContext::new_rejecting_software` | assoc-fn | 143 | wasm | `NewWgpuContextRejectingSoftware` | todo |
| `WgpuContext::supports_transient_attachments` | method | 341 |  | `(WgpuContext).SupportsTransientAttachments` | todo |
| `WgpuContext::new_web` | assoc-fn | 354 | wasm | `NewWgpuContextWeb` | todo |
| `WgpuContext::instance` | assoc-fn | 486 | wasm | `WgpuContextInstance` | todo |
| `WgpuContext::check_compatible_with_surface` | method | 496 |  | `(WgpuContext).CheckCompatibleWithSurface` | todo |
| `WgpuContext::backend` | method | 731 |  | `(WgpuContext).Backend` | todo |
| `WgpuContext::uses_texture_instance_data` | method | 735 |  | `(WgpuContext).UsesTextureInstanceData` | todo |
| `WgpuContext::supports_dual_source_blending` | method | 749 |  | `(WgpuContext).SupportsDualSourceBlending` | todo |
| `WgpuContext::color_texture_format` | method | 753 |  | `(WgpuContext).ColorTextureFormat` | todo |
| `WgpuContext::device_lost` | method | 759 |  | `(WgpuContext).DeviceLost` | todo |
| `WgpuContext::errors` | method | 764 |  | `(WgpuContext).Errors` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `device_errors_are_observed_independently` | test | 791 |  | `TestWgpuContext_DeviceErrorsAreObservedIndependently` | todo |
| `device_errors_coalesce_and_new_observers_see_latest` | test | 810 |  | `TestWgpuContext_DeviceErrorsCoalesceAndNewObserversSeeLatest` | todo |
| `test_parse_device_id` | test | 832 |  | `TestWgpuContext_ParseDeviceID` | todo |

## gpui_wgpu/src/wgpu_renderer.rs

Go target: `render/wgpu/wgpu_renderer.go` (package `wgpu`), tests in `wgpu_renderer_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WgpuSurfaceConfig` | struct | 113 |  | `WgpuSurfaceConfig` | todo |
| `GpuContext` | type | 182 |  | `GPUContext` | todo |
| `WgpuRenderer` | struct | 253 |  | `WgpuRenderer` | todo |
| `WgpuRenderer::new` | assoc-fn | 299 | wasm | `NewWgpuRenderer` | todo |
| `WgpuRenderer::new_from_canvas` | assoc-fn | 365 | wasm | `NewWgpuRendererFromCanvas` | todo |
| `WgpuRenderer::new_from_surface` | assoc-fn | 379 | wasm | `NewWgpuRendererFromSurface` | todo |
| `WgpuRenderer::update_drawable_size` | method | 1003 |  | `(WgpuRenderer).UpdateDrawableSize` | todo |
| `WgpuRenderer::set_subpixel_layout` | method | 1028 |  | `(WgpuRenderer).SetSubpixelLayout` | todo |
| `WgpuRenderer::update_transparency` | method | 1035 |  | `(WgpuRenderer).UpdateTransparency` | todo |
| `WgpuRenderer::viewport_size` | method | 1058 |  | `(WgpuRenderer).ViewportSize` | todo |
| `WgpuRenderer::sprite_atlas` | method | 1065 |  | `(WgpuRenderer).SpriteAtlas` | todo |
| `WgpuRenderer::supports_dual_source_blending` | method | 1069 |  | `(WgpuRenderer).SupportsDualSourceBlending` | todo |
| `WgpuRenderer::gpu_specs` | method | 1075 |  | `(WgpuRenderer).GPUSpecs` | todo |
| `WgpuRenderer::max_texture_size` | method | 1085 |  | `(WgpuRenderer).MaxTextureSize` | todo |
| `WgpuRenderer::draw` | method | 1089 |  | `(WgpuRenderer).Draw` | todo |
| `WgpuRenderer::unconfigure_surface` | method | 2079 |  | `(WgpuRenderer).UnconfigureSurface` | todo |
| `WgpuRenderer::replace_surface` | method | 2098 | wasm | `(WgpuRenderer).ReplaceSurface` | todo |
| `WgpuRenderer::destroy` | method | 2149 |  | `(WgpuRenderer).Destroy` | todo |
| `WgpuRenderer::device_lost` | method | 2156 |  | `(WgpuRenderer).DeviceLost` | todo |
| `WgpuRenderer::needs_redraw` | method | 2162 |  | `(WgpuRenderer).NeedsRedraw` | todo |
| `WgpuRenderer::recover` | method | 2174 | wasm | `(WgpuRenderer).Recover` | todo |
| `WgpuHeadlessRenderer` | struct | 2278 | wasm,feat:bench-support,feat:test-support | `WgpuHeadlessRenderer` | todo |
| `WgpuHeadlessRenderer::new` | assoc-fn | 2290 | wasm,feat:bench-support,feat:test-support | `NewWgpuHeadlessRenderer` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `headless_renderer_draws_quads_with_distinct_colors` | test | 2640 | linux | `TestWgpuRenderer_HeadlessRendererDrawsQuadsWithDistinctColors` | todo |
| `headless_renderer_captures_each_requested_size` | test | 2658 | linux | `TestWgpuRenderer_HeadlessRendererCapturesEachRequestedSize` | todo |
| `headless_renderer_reuses_target_for_same_size` | test | 2684 | linux | `TestWgpuRenderer_HeadlessRendererReusesTargetForSameSize` | todo |
| `headless_renderer_rejects_invalid_sizes` | test | 2707 | linux | `TestWgpuRenderer_HeadlessRendererRejectsInvalidSizes` | todo |
| `webgl_shader_is_valid_wgsl_without_storage_buffers` | test | 2736 |  | `TestWgpuRenderer_WebglShaderIsValidWgslWithoutStorageBuffers` | todo |
| `storage_buffer_shader_is_valid_wgsl` | test | 2742 |  | `TestWgpuRenderer_StorageBufferShaderIsValidWgsl` | todo |
| `subpixel_storage_buffer_shader_is_valid_wgsl` | test | 2747 |  | `TestWgpuRenderer_SubpixelStorageBufferShaderIsValidWgsl` | todo |
| `subpixel_webgl_shader_is_valid_wgsl_without_storage_buffers` | test | 2755 |  | `TestWgpuRenderer_SubpixelWebglShaderIsValidWgslWithoutStorageBuffers` | todo |
| `webgl_record_sizes_match_shader_word_strides` | test | 2771 |  | `TestWgpuRenderer_WebglRecordSizesMatchShaderWordStrides` | todo |
| `webgl_quad_layout_matches_fixed_decoder` | test | 2784 |  | `TestWgpuRenderer_WebglQuadLayoutMatchesFixedDecoder` | todo |

