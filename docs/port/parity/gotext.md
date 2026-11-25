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
| `cached_font_resolution_does_not_wait_for_cold_lookups` | test | 1277 |  | `TestCosmicTextSystem_CachedFontResolutionDoesNotWaitForColdLookups` | done |
| `adding_fonts_retries_cached_font_failures` | test | 1313 |  | `TestCosmicTextSystem_AddingFontsRetriesCachedFontFailures` | done |
| `font_properties_describe_the_selected_face` | test | 1341 |  | `TestCosmicTextSystem_FontPropertiesDescribeTheSelectedFace` | done |
| `combining_marks_are_positioned_over_their_base` | test | 1375 |  | `TestCosmicTextSystem_CombiningMarksArePositionedOverTheirBase` | done |
| `shape_text_with_mixed_direction_paragraphs` | test | 1400 |  | `TestCosmicTextSystem_ShapeTextWithMixedDirectionParagraphs` | done |
| `reports_graphemes_that_exhaust_font_fallback` | test | 1422 | feat:test-support | `TestCosmicTextSystem_ReportsGraphemesThatExhaustFontFallback` | done |
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
| `Render::render_into` (Source::ColorOutline, ColorBitmap, Bitmap) | method | 885 |  | `(swashRender).renderInto` | done |
| `Render::render` | method | 985 |  | `(swashRender).render` | done |
| `Render::new` / `default_color` | assoc-fn | 789 |  | `newSwashRender` | done |
| `Scaler::scale_color_outline_impl` | method | 592 |  | `(swashScaler).scaleColorOutlineImpl` | done |
| `Scaler::has_color_outlines` | method | 518 |  | `(swashScaler).hasColorOutlines` | done |
| `Scaler::has_bitmaps` | method | 518 |  | `(swashScaler).hasBitmaps` | done |
| `Scaler::has_color_bitmaps` | method | 518 |  | `(swashScaler).hasColorBitmaps` | done |
| `Scaler::scale_bitmap_into` | method | 620 |  | `(swashScaler).scaleBitmapInto` | done |
| `Scaler::scale_color_bitmap_into` | method | 620 |  | `(swashScaler).scaleColorBitmapInto` | done |
| `Scaler::scale_bitmap_impl` | method | 665 |  | `(swashScaler).scaleBitmapImpl` | done |
| `StrikeWith` | enum | 1 |  | `swashStrikeWith` | done |

## swash-0.2.10/src/scale/outline.rs (layer subset)

Go target: `text/gotext/scale.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Outline::len` | method | 30 |  | `(swashOutline).len` | done |
| `Outline::get` | method | 30 |  | `(swashOutline).get` | done |
| `Outline::path` | method | 30 |  | `(swashOutline).path` | done |
| `Outline::bounds` | method | 30 |  | `(swashOutline).bounds` | done |
| `Outline::clear` | method | 30 |  | `(swashOutline).clear` | done |
| `Outline::begin_layer` | method | 269 |  | `(swashOutline).beginLayer` | done |
| `Outline::maybe_close` | method | 269 |  | `(swashOutline).maybeClose` | done |
| `Outline::finish` | method | 269 |  | `(swashOutline).finish` | done |
| `Layer` | struct | 118 |  | `swashOutlineLayer` | done |

## swash-0.2.10/src/scale/image.rs

Go target: `text/gotext/scale.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Image::clear` | method | 46 |  | `(swashImage).clear` | done |

## swash-0.2.10/src/scale/color.rs

Go target: `text/gotext/scale_color.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ColorProxy` | struct | 11 |  | `swashColorProxy` | done |
| `ColorProxy::from_font` | assoc-fn | 11 |  | `newSwashColorProxy` | done |
| `ColorProxy::layers` | method | 24 |  | `(swashColorProxy).layers` | done |
| `ColorProxy::palette` | method | 57 |  | `(swashColorProxy).palette` | done |
| `Layers` | struct | 66 |  | `swashLayers` | done |
| `Layers::get` | method | 66 |  | `(swashLayers).get` | done |
| `Layer` | struct | 92 |  | `swashLayer` | done |

## swash-0.2.10/src/palette.rs (render subset)

Go target: `text/gotext/palette.go`. Palette names and usability flags are not used by rendering.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ColorPalettes::new` | method | 20 |  | `newSwashColorPalettes` | done |
| `get` | method | 20 |  | `(swashColorPalettes).get` | done |
| `ColorPalette::get` | method | 140 |  | `(swashColorPalette).get` | done |

## swash-0.2.10/src/strike.rs

Go target: `text/gotext/strike.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `BitmapStrikesProxy` (from_font, has_alpha, has_color, materialize_*) | struct | 13 |  | `swashBitmapStrikesProxy` | done |
| `BitmapStrikes` (get, find_by_nearest_ppem, find_by_exact_ppem, find_by_largest_ppem) | struct | 106 |  | `swashBitmapStrikes` | done |
| `BitmapStrike` (ppem, contains, get) | struct | 228 |  | `swashBitmapStrike` | done |
| `BitmapFormat` | enum | 278 |  | `swashBitmapFormat` | done |
| `Bitmap` (new, decoded_size, scaled_size, decode) | struct | 308 |  | `swashBitmap` | done |
| `Location` | struct | 470 |  | `swashLocation` | done |
| `get_coverage` | fn | 488 |  | `getCoverage` | done |
| `get_location` | fn | 515 |  | `getLocation` | done |
| `get_data` | fn | 643 |  | `getData` | done |
| `sbix_range` | fn | 735 |  | `sbixRange` | done |
| `get_metrics` | fn | 806 |  | `getMetrics` | done |

## swash-0.2.10/src/scale/bitmap/mod.rs

Go target: `text/gotext/scale_bitmap.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `blit` | fn | 18 |  | `swashBlit` | done |
| `Filter` | enum | 84 |  | `swashFilter` | done |
| `resize` | fn | 94 |  | `swashResize` | done |
| `resample` | fn | 199 |  | `swashResample` | done |
| `sample_dir` | fn | 291 |  | `swashSampleDir` | done |
| `sinc` | fn | 350 |  | `swashSinc` | done |
| `lanczos3` | fn | 359 |  | `swashLanczos3` | done |
| `bilinear` | fn | 367 |  | `swashBilinear` | done |
| `bicubic` | fn | 376 |  | `swashBicubic` | done |
| `mitchell` | fn | 393 |  | `swashMitchell` | done |
| `nearest` | fn | 404 |  | `swashNearest` | done |
| `gaussian` | fn | 408 |  | `swashGaussian` | done |
| `decode_png` | fn | 12 |  | `decodePng` | dev:D-SWASH-PNG |

## swash-0.2.10/src/scale/bitmap/png.rs

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| PNG decoder | module | 1 |  | `decodePng` (image/png) | dev:D-SWASH-PNG |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-GOTEXT-CTOR | `CosmicTextSystem::new` / `new_without_system_fonts` are `NewTextSystem` / `NewTextSystemWithoutSystemFonts` (the type keeps upstream's name); cosmic-text, fontdb, swash metrics and skrifa line metrics are re-expressed on go-text/typesetting in `fontdb.go`, `font.go`, `shape.go` and `fallback_*.go` | `gpui_wgpu/src/cosmic_text_system.rs:131`, `:147` | Constructor names fixed by STAGES.md M2-gotext-layout; cosmic-text has no Go port |
| D-ZENO-FILL | Only zeno's fill path is ported (no stroker, segment iterator, SVG parser or AdaptiveStorage); `Style` is reduced to `Fill`, and the unscratched render path uses `HeapStorage` | `zeno-0.3.3/src/style.rs`, `segment.rs`, `raster.rs:605` | STAGES.md M2-gotext-raster scopes the port to glyph fills; AdaptiveStorage yields identical cells |
| D-SWASH-PNG | CBDT / sbix PNG strikes are decoded with Go's image/png into straight RGBA instead of swash's own inflate + PNG decoder | `swash-0.2.10/src/scale/bitmap/png.rs` | STAGES.md M2-gotext-color prescribes image/png; output format (8-bit RGBA, unpremultiplied) is the same |
| D-SEC-FONTPANIC | Panics from the go-text font parser and shaper on malformed font data are recovered: a resource that panics while loading or parsing face info is skipped, `charmap` returns glyph 0, a face whose shaping panics is marked unloadable and the next fallback face is tried, and glyph rendering returns an error (SEC-008) | `cosmic-text` font loading and shaping (`gpui_wgpu/src/cosmic_text_system.rs`) | go-text parses tables lazily and panics on malformed glyf/loca/GDEF; Rust's parsers return errors |
| D-SEC-GLYPHSIZE | Bitmap glyph sizes are computed in checked 64-bit math with each side at most 16384 px and an image at most 256 MiB (`maxGlyphImageDim`, `maxGlyphImageBytes`); PNG strikes check the IHDR size against the target before decoding and the decoded size after; the strike count is bounded by the table length; a glyph data length that differs from `w*h` is rejected instead of panicking; index format 4 lookups advance with `l = i + 1` (SEC-009, 010, 011, 068) | `swash-0.2.10/src/strike.rs:106`, `:308`, `:515`; `swash-0.2.10/src/scale/bitmap/mod.rs`; `zeno` mask | u32 size math wraps on font-controlled dimensions; swash 0.2.10 itself has `l = i + i` in `get_location` (`strike.rs:636`), carried into the port |
