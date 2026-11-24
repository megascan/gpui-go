# Parity: gotext

Scope: gpui_wgpu, cosmic-text system (cosmic_text_system.rs, benches/layout_line.rs). Milestone: M2. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_wgpu/src/cosmic_text_system.rs

Go target: `text/gotext/text_system.go` (package `gotext`), tests in `text_system_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CosmicTextSystem` | struct | 24 |  | `CosmicTextSystem` | done |
| `CosmicTextSystem::font_weight_and_style` | method | 96 |  | `(CosmicTextSystem).FontWeightAndStyle` | done |
| `CosmicTextSystem::missing_glyphs` | method | 120 |  | `(CosmicTextSystem).MissingGlyphs` | done |
| `CosmicTextSystem::new` | assoc-fn | 131 |  | `NewTextSystem` | dev:D-GOTEXT-CTOR |
| `CosmicTextSystem::new_without_system_fonts` | assoc-fn | 147 |  | `NewTextSystemWithoutSystemFonts` | dev:D-GOTEXT-CTOR |
| `CosmicTextSystem::glyph_raster_bounds` | trait-method | 261 |  | `(CosmicTextSystem).GlyphRasterBounds` | dev:L-GLYPH |
| `CosmicTextSystem::rasterize_glyph` | trait-method | 265 |  | `(CosmicTextSystem).RasterizeGlyph` | dev:L-GLYPH |
| `CosmicTextSystemState::raster_bounds` | method | 465 |  | `(cosmicTextSystemState).rasterBounds` | done |
| `CosmicTextSystemState::rasterize_glyph` | method | 478 |  | `(cosmicTextSystemState).rasterizeGlyph` | done |
| `CosmicTextSystemState::render_glyph_image` | method | 512 |  | `(cosmicTextSystemState).renderGlyphImage` | dev:L-GLYPH |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `all_font_names_tracks_available_families` | test | 1222 |  | `TestCosmicTextSystem_AllFontNamesTracksAvailableFamilies` | done |
| `cached_font_resolution_does_not_wait_for_cold_lookups` | test | 1277 |  | `TestCosmicTextSystem_CachedFontResolutionDoesNotWaitForColdLookups` | blocked:M2-exit |
| `adding_fonts_retries_cached_font_failures` | test | 1313 |  | `TestCosmicTextSystem_AddingFontsRetriesCachedFontFailures` | done |
| `font_properties_describe_the_selected_face` | test | 1341 |  | `TestCosmicTextSystem_FontPropertiesDescribeTheSelectedFace` | done |
| `combining_marks_are_positioned_over_their_base` | test | 1375 |  | `TestCosmicTextSystem_CombiningMarksArePositionedOverTheirBase` | done |
| `shape_text_with_mixed_direction_paragraphs` | test | 1400 |  | `TestCosmicTextSystem_ShapeTextWithMixedDirectionParagraphs` | done |
| `reports_graphemes_that_exhaust_font_fallback` | test | 1422 | feat:test-support | `TestCosmicTextSystem_ReportsGraphemesThatExhaustFontFallback` | blocked:M0-test-contexts |
| `combines_missing_glyphs_from_one_grapheme` | test | 1458 |  | `TestCosmicTextSystem_CombinesMissingGlyphsFromOneGrapheme` | done |
| `adding_fonts_invalidates_cached_line_layouts` | test | 1478 |  | `TestCosmicTextSystem_AddingFontsInvalidatesCachedLineLayouts` | done |
| `layout_line_with_mixed_direction_paragraphs` | test | 1513 |  | `TestCosmicTextSystem_LayoutLineWithMixedDirectionParagraphs` | done |
| `layout_line_with_separators_at_line_edges` | test | 1536 |  | `TestCosmicTextSystem_LayoutLineWithSeparatorsAtLineEdges` | done |
| `layout_line_keeps_indices_and_positions_ordered_across_paragraphs` | test | 1558 |  | `TestCosmicTextSystem_LayoutLineKeepsIndicesAndPositionsOrderedAcrossParagraphs` | done |
| `layout_line_with_font_run_straddling_a_separator` | test | 1584 |  | `TestCosmicTextSystem_LayoutLineWithFontRunStraddlingASeparator` | done |
| `layout_line_without_separators_takes_fast_path` | test | 1610 |  | `TestCosmicTextSystem_LayoutLineWithoutSeparatorsTakesFastPath` | done |
| `index_for_position_past_line_width` | test | 1633 |  | `TestCosmicTextSystem_IndexForPositionPastLineWidth` | done |
| `paragraph_separator_detection` | test | 1672 |  | `TestCosmicTextSystem_ParagraphSeparatorDetection` | done |
| `font_runs_are_clipped_to_segment` | test | 1690 |  | `TestCosmicTextSystem_FontRunsAreClippedToSegment` | done |
| `primary_wins_over_current_fallback_when_primary_covers` | test | 1727 |  | `TestCosmicTextSystem_PrimaryWinsOverCurrentFallbackWhenPrimaryCovers` | done |
| `primary_preferred_over_fallback_when_both_cover` | test | 1738 |  | `TestCosmicTextSystem_PrimaryPreferredOverFallbackWhenBothCover` | done |
| `falls_through_chain_in_order` | test | 1746 |  | `TestCosmicTextSystem_FallsThroughChainInOrder` | done |
| `no_coverage_returns_primary` | test | 1758 |  | `TestCosmicTextSystem_NoCoverageReturnsPrimary` | done |
| `empty_chain_always_returns_primary` | test | 1771 |  | `TestCosmicTextSystem_EmptyChainAlwaysReturnsPrimary` | done |
| `slot_font_id_resolution` | test | 1779 |  | `TestCosmicTextSystem_SlotFontIDResolution` | done |
| `run_spans_with_no_chain_emit_one_primary_span` | test | 1788 |  | `TestCosmicTextSystem_RunSpansWithNoChainEmitOnePrimarySpan` | done |
| `run_spans_use_byte_offsets_for_multibyte_chars` | test | 1798 |  | `TestCosmicTextSystem_RunSpansUseByteOffsetsForMultibyteChars` | done |
| `run_spans_respect_run_offset` | test | 1823 |  | `TestCosmicTextSystem_RunSpansRespectRunOffset` | done |
| `run_spans_keep_combining_marks_with_base_in_fallback` | test | 1845 |  | `TestCosmicTextSystem_RunSpansKeepCombiningMarksWithBaseInFallback` | done |
| `run_spans_keep_zwj_inside_emoji_cluster` | test | 1865 |  | `TestCosmicTextSystem_RunSpansKeepZwjInsideEmojiCluster` | done |
| `run_spans_collapse_adjacent_same_slot` | test | 1877 |  | `TestCosmicTextSystem_RunSpansCollapseAdjacentSameSlot` | done |
| `run_spans_empty_run_returns_no_spans` | test | 1893 |  | `TestCosmicTextSystem_RunSpansEmptyRunReturnsNoSpans` | done |

## zeno-0.3.3/src/geometry.rs

Go target: `text/gotext/internal/zeno/geometry.go` (package `zeno`). Fill-path subset only; `Angle`, vector length/normalize/angle helpers and rotation/scale transforms are used only by the stroker and SVG arcs.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Vector` | struct | 50 |  | `Vector` | done |
| `Point` | struct | 50 |  | `Point` | done |
| `Vector::ceil` | method | 108 |  | `(Vector).Ceil` | done |
| `Vector::floor` | method | 108 |  | `(Vector).Floor` | done |
| `Transform` | struct | 258 |  | `Transform` | done |
| `Transform::transform_point` | method | 425 |  | `(Transform).TransformPoint` | done |
| `Transform::transform_vector` | method | 425 |  | `(Transform).TransformVector` | done |
| `Origin` | enum | 444 |  | `Origin` | done |
| `Placement` | struct | 459 |  | `Placement` | done |
| `Bounds` | struct | 507 |  | `Bounds` | done |
| `BoundsBuilder` | struct | 553 |  | `BoundsBuilder` | done |

## zeno-0.3.3/src/command.rs

Go target: `text/gotext/internal/zeno/command.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Verb` | enum | 59 |  | `Verb` | done |
| `PointsCommands` | struct | 68 |  | `PointsCommands` | done |
| `PointsCommands::copy_to` | method | 86 |  | `(PointsCommands).CopyTo` | done |

## zeno-0.3.3/src/path_builder.rs

Go target: `text/gotext/internal/zeno/path_builder.go`. Relative, arc and shape helpers are not on the fill path.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PathBuilder` | trait | 32 |  | `PathBuilder` | done |
| `TransformSink` | struct | 263 |  | `TransformSink` | done |
| `impl PathBuilder for BoundsBuilder` | impl | 311 |  | `(BoundsBuilder).MoveTo` | done |

## zeno-0.3.3/src/path_data.rs

Go target: `text/gotext/internal/zeno/path_data.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PathData` | trait | 85 |  | `PathData` | done |
| `bounds` | fn | 124 |  | `PathBounds` | done |
| `apply` | fn | 139 |  | `Apply` | dev:D-ZENO-FILL |

## zeno-0.3.3/src/style.rs

Go target: `text/gotext/internal/zeno/style.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Fill` | enum | 5 |  | `Fill` | done |
| `Stroke` / `Join` / `Cap` / `Style` | struct | 36 |  | (none) | dev:D-ZENO-FILL |

## zeno-0.3.3/src/scratch.rs

Go target: `text/gotext/internal/zeno/scratch.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Scratch` | struct | 16 |  | `Scratch` | done |
| `Scratch::apply` / `Scratch::bounds` | method | 28 |  | `(Scratch).Apply`, `(Scratch).Bounds` | dev:D-ZENO-FILL |

## zeno-0.3.3/src/raster.rs

Go target: `text/gotext/internal/zeno/raster.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `coverage` | fn | 13 |  | `coverage` | done |
| `Rasterizer` | struct | 31 |  | `Rasterizer` | done |
| `Rasterizer::rasterize` | method | 74 |  | `(Rasterizer).Rasterize` | done |
| `Rasterizer::rasterize_write` | method | 171 |  | `(Rasterizer).RasterizeWrite` | done |
| `Rasterizer::set_cell` | method | 258 |  | `(Rasterizer).setCell` | done |
| `Rasterizer::move_to` | method | 258 |  | `(Rasterizer).moveToFixed` | done |
| `Rasterizer::line_to` | method | 258 |  | `(Rasterizer).lineToFixed` | done |
| `Rasterizer::quad_to` | method | 258 |  | `(Rasterizer).quadToFixed` | done |
| `Rasterizer::curve_to` | method | 258 |  | `(Rasterizer).curveToFixed` | done |
| `impl PathBuilder for Rasterizer` | impl | 466 |  | `(Rasterizer).MoveTo` | done |
| `HeapStorage` | struct | 549 |  | `HeapStorage` | done |
| `AdaptiveStorage` | struct | 605 |  | `HeapStorage` | dev:D-ZENO-FILL |
| `split_quad` | fn | 704 |  | `splitQuad` | done |
| `split_cubic` | fn | 704 |  | `splitCubic` | done |
| `FixedPoint` | struct | 753 |  | `FixedPoint` | done |

## zeno-0.3.3/src/mask.rs

Go target: `text/gotext/internal/zeno/mask.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Format` | enum | 15 |  | `Format` | done |
| `Format::Alpha` | enum | 15 |  | `FormatAlpha` | done |
| `Format::Subpixel` | enum | 15 |  | `FormatSubpixel` | done |
| `Format::CustomSubpixel` | enum | 15 |  | `FormatCustomSubpixel` | done |
| `Format::subpixel_bgra` | assoc-fn | 27 |  | `FormatSubpixelBGRA` | done |
| `Format::buffer_size` | method | 33 |  | `(Format).BufferSize` | done |
| `Mask` | struct | 51 |  | `Mask` | done |
| `Mask::new` | method | 71 |  | `NewMask` | done |
| `Mask::with_scratch` | method | 90 |  | `MaskWithScratch` | done |
| `Mask::style` | method | 109 |  | `(Mask).Style` | done |
| `Mask::transform` | method | 115 |  | `(Mask).Transform` | done |
| `Mask::format` | method | 122 |  | `(Mask).Format` | done |
| `Mask::origin` | method | 128 |  | `(Mask).Origin` | done |
| `Mask::offset` | method | 136 |  | `(Mask).Offset` | done |
| `Mask::size` | method | 144 |  | `(Mask).Size` | done |
| `Mask::render_offset` | method | 155 |  | `(Mask).RenderOffset` | done |
| `Mask::inspect` | method | 163 |  | `(Mask).Inspect` | done |
| `Mask::ensure_size` | method | 210 |  | `(Mask).ensureSize` | done |
| `Mask::render_into` | method | 177 |  | `(Mask).RenderInto` | done |
| `Mask::render` | method | 177 |  | `(Mask).Render` | done |
| `Mask::placement` | method | 223 |  | `(Mask).placement` | done |
| `render` | fn | 259 |  | `render` | done |

## zeno-0.3.3/src/segment.rs

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Segment`, `segments` | module | 1 |  | (none) | n/a:D-ZENO-FILL |

## swash-0.2.10/src/scale/mod.rs (outline render subset)

Go target: `text/gotext/scale.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ScaleContext` / `ScalerBuilder` | struct | 262 |  | `swashScaleContext`, `(swashScaleContext).builder` | dev:L-GLYPH |
| `Scaler::scale_outline_impl` | method | 555 |  | `(swashScaler).scaleOutline` | dev:L-GLYPH |
| `Render` | struct | 776 |  | `swashRender` | done |
| `Render::render_into` (Source::Outline) | method | 846 |  | `(swashRender).renderInto` | done |
| `Render::render_into` (Source::ColorOutline, ColorBitmap, Bitmap) | method | 885 |  | `(swashRender).renderInto` | blocked:M2-gotext-color |
| `Render::render` | method | 985 |  | `(swashRender).render` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-GOTEXT-CTOR | `CosmicTextSystem::new` / `new_without_system_fonts` are `NewTextSystem` / `NewTextSystemWithoutSystemFonts` (the type keeps upstream's name); cosmic-text, fontdb, swash metrics and skrifa line metrics are re-expressed on go-text/typesetting in `fontdb.go`, `font.go`, `shape.go` and `fallback_*.go` | `gpui_wgpu/src/cosmic_text_system.rs:131`, `:147` | Constructor names fixed by STAGES.md M2-gotext-layout; cosmic-text has no Go port |
| D-ZENO-FILL | Only zeno's fill path is ported (no stroker, segment iterator, SVG parser or AdaptiveStorage); `Style` is reduced to `Fill`, and the unscratched render path uses `HeapStorage` | `zeno-0.3.3/src/style.rs`, `segment.rs`, `raster.rs:605` | STAGES.md M2-gotext-raster scopes the port to glyph fills; AdaptiveStorage yields identical cells |
