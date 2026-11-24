# Parity: core-text

Scope: gpui: text system, line layout/wrapping, svg renderer. Milestone: M0/M2. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/svg_renderer.rs

Go target: `svg_renderer.go` (package `gpui`), tests in `svg_renderer_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SMOOTH_SVG_SCALE_FACTOR` | const | 84 |  | `SmoothSvgScaleFactor` | todo |
| `RenderSvgParams` | struct | 88 |  | `RenderSvgParams` | todo |
| `SvgRenderer` | struct | 95 |  | `SvgRenderer` | todo |
| `ParsedSvg` | struct | 106 |  | `ParsedSvg` | todo |
| `SvgTextFont` | struct | 110 |  | `SvgTextFont` | todo |
| `SvgSize` | enum | 123 |  | `SvgSize` | todo |
| `SvgRenderer::new` | assoc-fn | 140 |  | `NewSvgRenderer` | todo |
| `SvgRenderer::parse_svg` | method | 225 |  | `(SvgRenderer).ParseSvg` | todo |
| `SvgRenderer::measure_text_width` | method | 230 |  | `(SvgRenderer).MeasureTextWidth` | todo |
| `SvgRenderer::render_parsed` | method | 253 |  | `(SvgRenderer).RenderParsed` | todo |
| `SvgRenderer::render_single_frame` | method | 280 |  | `(SvgRenderer).RenderSingleFrame` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `renders_parsed_svg_at_requested_size` | test | 558 |  | `TestSvgRenderer_RendersParsedSvgAtRequestedSize` | todo |
| `preserves_aspect_ratio_for_width_constrained_size` | test | 571 |  | `TestSvgRenderer_PreservesAspectRatioForWidthConstrainedSize` | todo |
| `measured_text_width_matches_rasterized_advance` | test | 586 |  | `TestSvgRenderer_MeasuredTextWidthMatchesRasterizedAdvance` | todo |
| `owns_font_bytes_within_budget` | test | 614 |  | `TestSvgRenderer_OwnsFontBytesWithinBudget` | todo |
| `fallback_faces_are_remembered_until_excluded` | test | 653 |  | `TestSvgRenderer_FallbackFacesAreRememberedUntilExcluded` | todo |
| `text_with_split_glyph_clusters_in_mixed_fonts_does_not_panic` | test | 690 |  | `TestSvgRenderer_TextWithSplitGlyphClustersInMixedFontsDoesNotPanic` | todo |
| `test_is_emoji_presentation` | test | 715 |  | `TestSvgRenderer_IsEmojiPresentation` | todo |
| `fix_generic_font_families_sets_all_families` | test | 746 |  | `TestSvgRenderer_FixGenericFontFamiliesSetsAllFamilies` | todo |
| `test_select_emoji_font_skips_family_without_glyph` | test | 771 |  | `TestSvgRenderer_SelectEmojiFontSkipsFamilyWithoutGlyph` | todo |
| `fix_generic_font_families_monospace_resolves_to_lilex` | test | 798 |  | `TestSvgRenderer_FixGenericFontFamiliesMonospaceResolvesToLilex` | todo |

## gpui/src/text_system.rs

Go target: `text_system.go` (package `gpui`), tests in `text_system_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FontId` | struct | 44 |  | `FontID` | done |
| `FontFamilyId` | struct | 48 |  | `FontFamilyID` | done |
| `SUBPIXEL_VARIANTS_X` | const | 51 |  | `SubpixelVariantsX` | done |
| `SUBPIXEL_VARIANTS_Y` | const | 54 |  | `SubpixelVariantsY` | done |
| `underline_y_offset` | fn | 60 |  | `UnderlineYOffset` | done |
| `FallbackFontClass` | enum | 69 |  | `FallbackFontClass` | done |
| `MissingGlyph` | struct | 78 |  | `MissingGlyph` | done |
| `MissingGlyph::new` | assoc-fn | 85 |  | `NewMissingGlyph` | done |
| `MissingGlyph::grapheme` | method | 93 |  | `(MissingGlyph).Grapheme` | done |
| `MissingGlyph::font_class` | method | 98 |  | `(MissingGlyph).FontClass` | done |
| `MissingGlyphSink` | trait | 104 |  | `MissingGlyphSink` | done |
| `MissingGlyphSink::report` | trait-method | 106 |  | `MissingGlyphSink.Report` | done |
| `TextSystem` | struct | 232 |  | `TextSystem` | done |
| `TextSystem::new` | assoc-fn | 253 |  | `NewTextSystem` | done |
| `TextSystem::all_font_names` | method | 291 |  | `(TextSystem).AllFontNames` | done |
| `TextSystem::add_fonts` | method | 302 |  | `(TextSystem).AddFonts` | done |
| `TextSystem::get_font_for_id` | method | 336 |  | `(TextSystem).GetFontForID` | done |
| `TextSystem::resolve_font` | method | 352 |  | `(TextSystem).ResolveFont` | done |
| `TextSystem::prewarm_fonts` | method | 366 |  | `(TextSystem).PrewarmFonts` | done |
| `TextSystem::bounding_box` | method | 380 |  | `(TextSystem).BoundingBox` | done |
| `TextSystem::typographic_bounds` | method | 385 |  | `(TextSystem).TypographicBounds` | done |
| `TextSystem::advance` | method | 404 |  | `(TextSystem).Advance` | done |
| `TextSystem::layout_width` | method | 417 |  | `(TextSystem).LayoutWidth` | done |
| `TextSystem::em_width` | method | 435 |  | `(TextSystem).EmWidth` | done |
| `TextSystem::em_advance` | method | 442 |  | `(TextSystem).EmAdvance` | done |
| `TextSystem::ch_width` | method | 449 |  | `(TextSystem).ChWidth` | done |
| `TextSystem::ch_advance` | method | 456 |  | `(TextSystem).ChAdvance` | done |
| `TextSystem::units_per_em` | method | 463 |  | `(TextSystem).UnitsPerEm` | done |
| `TextSystem::cap_height` | method | 468 |  | `(TextSystem).CapHeight` | done |
| `TextSystem::x_height` | method | 473 |  | `(TextSystem).XHeight` | done |
| `TextSystem::ascent` | method | 478 |  | `(TextSystem).Ascent` | done |
| `TextSystem::descent` | method | 484 |  | `(TextSystem).Descent` | done |
| `TextSystem::baseline_offset` | method | 489 |  | `(TextSystem).BaselineOffset` | done |
| `TextSystem::line_wrapper` | method | 516 |  | `(TextSystem).LineWrapper` | done |
| `WindowTextSystem` | struct | 635 |  | `WindowTextSystem` | done |
| `WindowTextSystem::new` | assoc-fn | 644 |  | `NewWindowTextSystem` | done |
| `WindowTextSystem::shape_line` | method | 673 |  | `(WindowTextSystem).ShapeLine` | done |
| `WindowTextSystem::shape_text` | method | 717 |  | `(WindowTextSystem).ShapeText` | done |
| `WindowTextSystem::layout_line` | method | 853 |  | `(WindowTextSystem).LayoutLine` | done |
| `WindowTextSystem::layout_width` | method | 909 |  | `(WindowTextSystem).LayoutWidth` | done |
| `WindowTextSystem::em_layout_width` | method | 926 |  | `(WindowTextSystem).EmLayoutWidth` | done |
| `LineWrapperHandle` | struct | 938 |  | `LineWrapperHandle` | done |
| `FontWeight` | struct | 975 |  | `FontWeight` | done |
| `FontWeight::THIN` | assoc-const | 1006 |  | `FontWeightThin` | done |
| `FontWeight::EXTRA_LIGHT` | assoc-const | 1008 |  | `FontWeightExtraLight` | done |
| `FontWeight::LIGHT` | assoc-const | 1010 |  | `FontWeightLight` | done |
| `FontWeight::NORMAL` | assoc-const | 1012 |  | `FontWeightNormal` | done |
| `FontWeight::MEDIUM` | assoc-const | 1014 |  | `FontWeightMedium` | done |
| `FontWeight::SEMIBOLD` | assoc-const | 1016 |  | `FontWeightSemibold` | done |
| `FontWeight::BOLD` | assoc-const | 1018 |  | `FontWeightBold` | done |
| `FontWeight::EXTRA_BOLD` | assoc-const | 1020 |  | `FontWeightExtraBold` | done |
| `FontWeight::BLACK` | assoc-const | 1022 |  | `FontWeightBlack` | done |
| `FontWeight::ALL` | assoc-const | 1025 |  | `FontWeightAll` | done |
| `FontStyle` | enum | 1057 |  | `FontStyle` | done |
| `TextRun` | struct | 1075 |  | `TextRun` | done |
| `GlyphId` | struct | 1102 |  | `GlyphID` | done |
| `RenderGlyphParams` | struct | 1111 |  | `RenderGlyphParams` | done |
| `Font` | struct | 1139 |  | `Font` | done |
| `font` | fn | 1165 |  | `NewFont` | done |
| `Font::bold` | method | 1177 |  | `(Font).Bold` | done |
| `Font::italic` | method | 1183 |  | `(Font).Italic` | done |
| `FontMetrics` | struct | 1192 |  | `FontMetrics` | done |
| `FontMetrics::ascent` | method | 1225 |  | `(FontMetrics).AscentPixels` | dev:D-FIELDMETHOD |
| `FontMetrics::descent` | method | 1230 |  | `(FontMetrics).DescentPixels` | dev:D-FIELDMETHOD |
| `FontMetrics::line_gap` | method | 1235 |  | `(FontMetrics).LineGapPixels` | dev:D-FIELDMETHOD |
| `FontMetrics::underline_position` | method | 1240 |  | `(FontMetrics).UnderlinePositionPixels` | dev:D-FIELDMETHOD |
| `FontMetrics::underline_thickness` | method | 1245 |  | `(FontMetrics).UnderlineThicknessPixels` | dev:D-FIELDMETHOD |
| `FontMetrics::cap_height` | method | 1250 |  | `(FontMetrics).CapHeightPixels` | dev:D-FIELDMETHOD |
| `FontMetrics::x_height` | method | 1255 |  | `(FontMetrics).XHeightPixels` | dev:D-FIELDMETHOD |
| `FontMetrics::bounding_box` | method | 1260 |  | `(FontMetrics).BoundingBoxPixels` | dev:D-FIELDMETHOD |
| `font_name_with_fallbacks` | fn | 1267 |  | `FontNameWithFallbacks` | done |
| `font_name_with_fallbacks_shared` | fn | 1281 |  | `FontNameWithFallbacksShared` | done |
| `MissingGlyphReceiver` | struct | 165 |  | `MissingGlyphReceiver` | done |
| `MissingGlyphReceiver::recv` | method | 177 |  | `(MissingGlyphReceiver).Recv` | done |
| `impl Drop for MissingGlyphReceiver` | impl | 224 |  | `(MissingGlyphReceiver).Close` | done |
| `impl Drop for LineWrapperHandle` | impl | 943 |  | `(LineWrapperHandle).Release` | done |
| `impl Deref for LineWrapperHandle` | impl | 957 |  | `LineWrapperHandle` | done |
| `impl Deref for WindowTextSystem` | impl | 633 |  | `WindowTextSystem` | done |
| `impl Display for FontWeight` | impl | 977 |  | `(FontWeight).String` | done |
| `impl FromStr for FontWeight` | impl | 974 |  | `FontWeightFromStr` | done |
| `impl Default for FontWeight` | impl | 989 |  | `DefaultFontWeight` | done |
| `impl JsonSchema for FontWeight` | impl | 1037 |  | `(FontWeight).JSONSchema` | dev:D-SCHEMAKW |
| `impl Display for FontStyle` | impl | 1068 |  | `(FontStyle).String` | done |
| `impl Serialize for FontStyle` | impl | 1056 |  | `(FontStyle).MarshalText` | done |
| `impl Deserialize for FontStyle` | impl | 1056 |  | `(FontStyle).UnmarshalText` | done |
| `impl Default for TextRun` | impl | 1074 |  | `DefaultTextRun` | done |
| `impl Default for Font` | impl | 1158 |  | `DefaultFont` | done |
| `impl Hash for RenderGlyphParams` | impl | 1124 |  | `RenderGlyphParams` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `caches_resolved_font_fallbacks` | test | 1302 |  | `TestTextSystemFontResolutionTests_CachesResolvedFontFallbacks` | done |
| `pooled_line_wrappers_do_not_retain_text_system` | test | 1329 |  | `TestTextSystemLineWrapperPoolTests_PooledLineWrappersDoNotRetainTextSystem` | blocked:M0-line-wrapper |
| `bounds_retained_missing_glyphs` | test | 1351 |  | `TestTextSystemMissingGlyphTests_BoundsRetainedMissingGlyphs` | done |
| `dropping_receiver_closes_and_clears_reports` | test | 1373 |  | `TestTextSystemMissingGlyphTests_DroppingReceiverClosesAndClearsReports` | done |

## gpui/src/text_system/font_fallbacks.rs

Go target: `text_system_font_fallbacks.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FontFallbacks` | struct | 9 |  | `FontFallbacks` | done |
| `FontFallbacks::fallback_list` | method | 13 |  | `(FontFallbacks).FallbackList` | done |
| `FontFallbacks::from_fonts` | assoc-fn | 18 |  | `FontFallbacksFromFonts` | done |
| `impl Serialize for FontFallbacks` | impl | 8 |  | `(FontFallbacks).MarshalJSONTo` | done |
| `impl Deserialize for FontFallbacks` | impl | 8 |  | `(FontFallbacks).UnmarshalJSONFrom` | done |
| `impl JsonSchema for FontFallbacks` | impl | 8 |  | `(FontFallbacks).JSONSchema` | done |

## gpui/src/text_system/font_features.rs

Go target: `text_system_font_features.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FontFeatures` | struct | 8 |  | `FontFeatures` | done |
| `FontFeatures::disable_ligatures` | assoc-fn | 21 |  | `FontFeaturesDisableLigatures` | done |
| `FontFeatures::tag_value_list` | method | 27 |  | `(FontFeatures).TagValueList` | done |
| `FontFeatures::is_calt_enabled` | method | 34 |  | `(FontFeatures).IsCaltEnabled` | done |
| `impl Default for FontFeatures` | impl | 10 |  | `DefaultFontFeatures` | done |
| `impl Debug for FontFeatures` | impl | 44 |  | `(FontFeatures).String` | done |
| `impl Deserialize for FontFeatures` | impl | 62 |  | `(FontFeatures).UnmarshalJSONFrom` | done |
| `impl Serialize for FontFeatures` | impl | 131 |  | `(FontFeatures).MarshalJSONTo` | done |
| `impl JsonSchema for FontFeatures` | impl | 146 |  | `(FontFeatures).JSONSchema` | dev:D-SCHEMAKW |

## gpui/src/text_system/line.rs

Go target: `text_system_line.go` (package `gpui`), tests in `text_system_line_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `GlyphRasterData` | struct | 15 |  | `GlyphRasterData` | todo |
| `DecorationRun` | struct | 24 |  | `DecorationRun` | todo |
| `ShapedLine` | struct | 43 |  | `ShapedLine` | todo |
| `ShapedLine::cursor` | method | 54 |  | `(ShapedLine).Cursor` | todo |
| `ShapedLine::len` | method | 79 |  | `(ShapedLine).Len` | todo |
| `ShapedLine::width` | method | 87 |  | `(ShapedLine).Width` | todo |
| `ShapedLine::with_len` | method | 93 |  | `(ShapedLine).WithLen` | todo |
| `ShapedLine::paint` | method | 107 |  | `(ShapedLine).Paint` | todo |
| `ShapedLine::paint_with_underline_handler` | method | 128 |  | `(ShapedLine).PaintWithUnderlineHandler` | todo |
| `ShapedLine::paint_background` | method | 159 |  | `(ShapedLine).PaintBackground` | todo |
| `ShapedLine::split_at` | method | 195 |  | `(ShapedLine).SplitAt` | todo |
| `ShapedLineCursor` | struct | 267 |  | `ShapedLineCursor` | todo |
| `ShapedLineCursor::take_until` | method | 284 |  | `(ShapedLineCursor).TakeUntil` | todo |
| `ShapedLineCursor::x_offset` | method | 379 |  | `(ShapedLineCursor).XOffset` | todo |
| `LineLayout::paint` | method | 390 |  | `(LineLayout).Paint` | todo |
| `LineLayout::paint_background` | method | 419 |  | `(LineLayout).PaintBackground` | todo |
| `WrappedLine` | struct | 448 |  | `WrappedLine` | todo |
| `WrappedLine::len` | method | 459 |  | `(WrappedLine).Len` | todo |
| `WrappedLine::paint` | method | 464 |  | `(WrappedLine).Paint` | todo |
| `WrappedLine::paint_background` | method | 495 |  | `(WrappedLine).PaintBackground` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_underline_handler_matches_default_paint` | gpui::test | 1008 |  | `TestLine_UnderlineHandlerMatchesDefaultPaint` | todo |
| `test_underline_handler_reports_zero_advance_geometry` | gpui::test | 1156 |  | `TestLine_UnderlineHandlerReportsZeroAdvanceGeometry` | todo |
| `test_underline_handler_matches_wrapped_paint` | gpui::test | 1235 |  | `TestLine_UnderlineHandlerMatchesWrappedPaint` | todo |
| `test_split_at_invariants` | test | 1318 |  | `TestLine_SplitAtInvariants` | todo |
| `test_split_at_glyph_rebasing` | test | 1369 |  | `TestLine_SplitAtGlyphRebasing` | todo |
| `test_split_at_decorations` | test | 1466 |  | `TestLine_SplitAtDecorations` | todo |
| `test_cursor_preserves_shaping_metadata_across_runs` | test | 1619 |  | `TestLine_CursorPreservesShapingMetadataAcrossRuns` | todo |
| `test_cursor_preserves_existing_visual_order_splitting` | test | 1706 |  | `TestLine_CursorPreservesExistingVisualOrderSplitting` | todo |
| `test_cursor_partitions_one_decoration_across_three_chunks` | test | 1733 |  | `TestLine_CursorPartitionsOneDecorationAcrossThreeChunks` | todo |
| `test_cursor_matches_successive_splits_at_ordered_boundaries` | test | 1765 |  | `TestLine_CursorMatchesSuccessiveSplitsAtOrderedBoundaries` | todo |
| `test_cursor_empty_chunks_and_repeated_boundaries` | test | 1830 |  | `TestLine_CursorEmptyChunksAndRepeatedBoundaries` | todo |
| `test_cursor_rejects_invalid_boundaries` | test | 1846 |  | `TestLine_CursorRejectsInvalidBoundaries` | todo |

## gpui/src/text_system/line_layout.rs

Go target: `text_system_line_layout.go` (package `gpui`), tests in `text_system_line_layout_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LineLayout` | struct | 20 |  | `LineLayout` | done |
| `ShapedRun` | struct | 37 |  | `ShapedRun` | done |
| `ShapedGlyph` | struct | 46 |  | `ShapedGlyph` | done |
| `LineLayout::index_for_x` | method | 62 |  | `(LineLayout).IndexForX` | done |
| `LineLayout::closest_index_for_x` | method | 79 |  | `(LineLayout).ClosestIndexForX` | done |
| `LineLayout::x_for_index` | method | 109 |  | `(LineLayout).XForIndex` | done |
| `LineLayout::font_id_for_index` | method | 121 |  | `(LineLayout).FontIDForIndex` | done |
| `LineLayout::split_at` | method | 139 |  | `(LineLayout).SplitAt` | done |
| `WrappedLineLayout` | struct | 297 |  | `WrappedLineLayout` | done |
| `WrapBoundary` | struct | 310 |  | `WrapBoundary` | done |
| `WrappedLineLayout::len` | method | 319 |  | `(WrappedLineLayout).Len` | done |
| `WrappedLineLayout::width` | method | 324 |  | `(WrappedLineLayout).Width` | done |
| `WrappedLineLayout::size` | method | 332 |  | `(WrappedLineLayout).Size` | done |
| `WrappedLineLayout::ascent` | method | 340 |  | `(WrappedLineLayout).Ascent` | done |
| `WrappedLineLayout::descent` | method | 345 |  | `(WrappedLineLayout).Descent` | done |
| `WrappedLineLayout::wrap_boundaries` | method | 350 |  | `WrappedLineLayout` | n/a:D-GETTER |
| `WrappedLineLayout::font_size` | method | 355 |  | `(WrappedLineLayout).FontSize` | done |
| `WrappedLineLayout::runs` | method | 360 |  | `(WrappedLineLayout).Runs` | done |
| `WrappedLineLayout::index_for_position` | method | 367 |  | `(WrappedLineLayout).IndexForPosition` | done |
| `WrappedLineLayout::closest_index_for_position` | method | 380 |  | `(WrappedLineLayout).ClosestIndexForPosition` | done |
| `WrappedLineLayout::position_for_index` | method | 447 |  | `(WrappedLineLayout).PositionForIndex` | done |
| `FontRun` | struct | 780 |  | `FontRun` | done |
| `LineLayoutCache` | struct | 498 |  | `LineLayoutCache` | done |
| `LineLayoutCache::new` | assoc-fn | 532 |  | `NewLineLayoutCache` | done |
| `LineLayoutIndex` | struct | 524 |  | `LineLayoutIndex` | done |
| `apply_force_width_to_layout` | fn | 755 |  | `applyForceWidthToLayout` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `shape_line_shares_its_text_with_the_layout_cache` | test | 863 |  | `TestLineLayout_ShapeLineSharesItsTextWithTheLayoutCache` | done |
| `test_wrap_preserves_shaped_clusters` | test | 912 |  | `TestLineLayout_WrapPreservesShapedClusters` | done |
| `test_wrap_right_to_left_text` | test | 942 |  | `TestLineLayout_WrapRightToLeftText` | done |
| `test_wrap_ignores_breaks_inside_shaped_clusters` | test | 971 |  | `TestLineLayout_WrapIgnoresBreaksInsideShapedClusters` | done |
| `test_force_width_latin_unchanged` | test | 993 |  | `TestLineLayout_ForceWidthLatinUnchanged` | done |
| `test_force_width_combining_marks_not_advanced` | test | 1004 |  | `TestLineLayout_ForceWidthCombiningMarksNotAdvanced` | done |
| `test_force_width_base_after_combining_mark` | test | 1019 |  | `TestLineLayout_ForceWidthBaseAfterCombiningMark` | done |
| `test_force_width_multiple_combining_marks` | test | 1030 |  | `TestLineLayout_ForceWidthMultipleCombiningMarks` | done |
| `test_force_width_corrects_drifted_base_positions` | test | 1047 |  | `TestLineLayout_ForceWidthCorrectsDriftedBasePositions` | done |
| `test_force_width_combining_mark_after_within_tolerance_base` | test | 1063 |  | `TestLineLayout_ForceWidthCombiningMarkAfterWithinToleranceBase` | done |
| `test_wrap_ignores_the_sign_of_a_zero_wrap_width_in_the_cache` | test | 1076 |  | `TestLineLayout_WrapIgnoresTheSignOfAZeroWrapWidthInTheCache` | done |

## gpui/src/text_system/line_wrapper.rs

Go target: `text_system_line_wrapper.go` (package `gpui`), tests in `text_system_line_wrapper_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TruncateFrom` | enum | 8 |  | `TruncateFrom` | done |
| `IndentAdjustment` | enum | 19 |  | `IndentAdjustment` | done |
| `LineWrapper` | struct | 30 |  | `LineWrapper` | done |
| `LineWrapper::MAX_INDENT` | assoc-const | 55 |  | `LineWrapperMaxIndent` | done |
| `LineWrapper::wrap_line` | method | 77 |  | `(LineWrapper).WrapLine` | done |
| `LineWrapper::should_truncate_line` | method | 240 |  | `(LineWrapper).ShouldTruncateLine` | done |
| `LineWrapper::truncate_line` | method | 347 |  | `(LineWrapper).TruncateLine` | done |
| `LineWrapper::truncate_wrapped_line` | method | 409 |  | `(LineWrapper).TruncateWrappedLine` | done |
| `LineFragment` | enum | 684 |  | `LineFragment` | done |
| `LineFragment::text` | assoc-fn | 701 |  | `LineFragmentText` | done |
| `LineFragment::element` | assoc-fn | 706 |  | `LineFragmentElement` | done |
| `Boundary` | struct | 744 |  | `Boundary` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_emergency_wrap_preserves_graphemes` | test | 851 |  | `TestLineWrapper_EmergencyWrapPreservesGraphemes` | done |
| `test_emergency_wrap_exact_boundaries` | test | 856 |  | `TestLineWrapper_EmergencyWrapExactBoundaries` | done |
| `test_emergency_wrap_elements_and_graphemes` | test | 897 |  | `TestLineWrapper_EmergencyWrapElementsAndGraphemes` | done |
| `test_wrap_line` | test | 941 |  | `TestLineWrapper_WrapLine` | done |
| `test_wrap_line_break_opportunities` | test | 1112 |  | `TestLineWrapper_WrapLineBreakOpportunities` | done |
| `test_truncate_line_end` | test | 1141 |  | `TestLineWrapper_TruncateLineEnd` | done |
| `test_truncate_line_start` | test | 1190 |  | `TestLineWrapper_TruncateLineStart` | done |
| `test_truncate_multiple_runs_end` | test | 1240 |  | `TestLineWrapper_TruncateMultipleRunsEnd` | done |
| `test_truncate_multiple_runs_start` | test | 1299 |  | `TestLineWrapper_TruncateMultipleRunsStart` | done |
| `test_update_run_after_truncation_end` | test | 1364 |  | `TestLineWrapper_UpdateRunAfterTruncationEnd` | done |
| `test_wrap_shaped_line` | crate::test | 1404 | macos | `TestLineWrapper_WrapShapedLine` | blocked:M0-blocked-tests |
| `test_multiline_truncation_fits_within_wrapped_lines` | test | 1459 |  | `TestLineWrapper_MultilineTruncationFitsWithinWrappedLines` | done |
| `test_multiline_truncation_no_truncation_needed` | test | 1513 |  | `TestLineWrapper_MultilineTruncationNoTruncationNeeded` | done |
| `test_multiline_truncation_three_lines` | test | 1540 |  | `TestLineWrapper_MultilineTruncationThreeLines` | done |
| `test_multiline_truncation_preserves_breaks_after_empty_lines` | test | 1581 |  | `TestLineWrapper_MultilineTruncationPreservesBreaksAfterEmptyLines` | done |
| `test_multiline_truncation_with_newlines` | test | 1604 |  | `TestLineWrapper_MultilineTruncationWithNewlines` | done |
| `test_multiline_truncation_newline_on_last_line` | test | 1652 |  | `TestLineWrapper_MultilineTruncationNewlineOnLastLine` | done |
| `test_truncate_line_middle` | test | 1682 |  | `TestLineWrapper_TruncateLineMiddle` | done |
| `test_multiline_truncation_trailing_newline` | test | 1757 |  | `TestLineWrapper_MultilineTruncationTrailingNewline` | done |
| `test_multiline_truncation_newline_fits_exactly` | test | 1784 |  | `TestLineWrapper_MultilineTruncationNewlineFitsExactly` | done |
| `test_extra_columns_overflow_guard` | test | 1812 |  | `TestLineWrapper_ExtraColumnsOverflowGuard` | done |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-FIELDMETHOD | Go forbids a field and a method of the same name: `FontMetrics`' pixel-scaling methods (`ascent`, `descent`, `line_gap`, `underline_position`, `underline_thickness`, `cap_height`, `x_height`, `bounding_box`) get a `Pixels` suffix; the fields keep upstream's names | `gpui/src/text_system.rs:1224-1262` | `text_system.go` |
| D-SCHEMAKW | internal/jsonschema.Schema has no `patternProperties`, `multipleOf` or `default`: `FontFeatures`' schema is a plain `object` and `FontWeight`'s omits `default: 400` (handoff M0-text-system) | `gpui/src/text_system/font_features.rs:146-162`, `gpui/src/text_system.rs:1037-1052` | `text_system_font_features.go`, `text_system.go` |

