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
| `FontId` | struct | 44 |  | `FontID` | todo |
| `FontFamilyId` | struct | 48 |  | `FontFamilyID` | todo |
| `SUBPIXEL_VARIANTS_X` | const | 51 |  | `SubpixelVariantsX` | todo |
| `SUBPIXEL_VARIANTS_Y` | const | 54 |  | `SubpixelVariantsY` | todo |
| `underline_y_offset` | fn | 60 |  | `UnderlineYOffset` | todo |
| `FallbackFontClass` | enum | 69 |  | `FallbackFontClass` | todo |
| `MissingGlyph` | struct | 78 |  | `MissingGlyph` | todo |
| `MissingGlyph::new` | assoc-fn | 85 |  | `NewMissingGlyph` | todo |
| `MissingGlyph::grapheme` | method | 93 |  | `(MissingGlyph).Grapheme` | todo |
| `MissingGlyph::font_class` | method | 98 |  | `(MissingGlyph).FontClass` | todo |
| `MissingGlyphSink` | trait | 104 |  | `MissingGlyphSink` | todo |
| `MissingGlyphSink::report` | trait-method | 106 |  | `MissingGlyphSink.Report` | todo |
| `TextSystem` | struct | 232 |  | `TextSystem` | todo |
| `TextSystem::new` | assoc-fn | 253 |  | `NewTextSystem` | todo |
| `TextSystem::all_font_names` | method | 291 |  | `(TextSystem).AllFontNames` | todo |
| `TextSystem::add_fonts` | method | 302 |  | `(TextSystem).AddFonts` | todo |
| `TextSystem::get_font_for_id` | method | 336 |  | `(TextSystem).GetFontForID` | todo |
| `TextSystem::resolve_font` | method | 352 |  | `(TextSystem).ResolveFont` | todo |
| `TextSystem::prewarm_fonts` | method | 366 |  | `(TextSystem).PrewarmFonts` | todo |
| `TextSystem::bounding_box` | method | 380 |  | `(TextSystem).BoundingBox` | todo |
| `TextSystem::typographic_bounds` | method | 385 |  | `(TextSystem).TypographicBounds` | todo |
| `TextSystem::advance` | method | 404 |  | `(TextSystem).Advance` | todo |
| `TextSystem::layout_width` | method | 417 |  | `(TextSystem).LayoutWidth` | todo |
| `TextSystem::em_width` | method | 435 |  | `(TextSystem).EmWidth` | todo |
| `TextSystem::em_advance` | method | 442 |  | `(TextSystem).EmAdvance` | todo |
| `TextSystem::ch_width` | method | 449 |  | `(TextSystem).ChWidth` | todo |
| `TextSystem::ch_advance` | method | 456 |  | `(TextSystem).ChAdvance` | todo |
| `TextSystem::units_per_em` | method | 463 |  | `(TextSystem).UnitsPerEm` | todo |
| `TextSystem::cap_height` | method | 468 |  | `(TextSystem).CapHeight` | todo |
| `TextSystem::x_height` | method | 473 |  | `(TextSystem).XHeight` | todo |
| `TextSystem::ascent` | method | 478 |  | `(TextSystem).Ascent` | todo |
| `TextSystem::descent` | method | 484 |  | `(TextSystem).Descent` | todo |
| `TextSystem::baseline_offset` | method | 489 |  | `(TextSystem).BaselineOffset` | todo |
| `TextSystem::line_wrapper` | method | 516 |  | `(TextSystem).LineWrapper` | todo |
| `WindowTextSystem` | struct | 635 |  | `WindowTextSystem` | todo |
| `WindowTextSystem::new` | assoc-fn | 644 |  | `NewWindowTextSystem` | todo |
| `WindowTextSystem::shape_line` | method | 673 |  | `(WindowTextSystem).ShapeLine` | todo |
| `WindowTextSystem::shape_text` | method | 717 |  | `(WindowTextSystem).ShapeText` | todo |
| `WindowTextSystem::layout_line` | method | 853 |  | `(WindowTextSystem).LayoutLine` | todo |
| `WindowTextSystem::layout_width` | method | 909 |  | `(WindowTextSystem).LayoutWidth` | todo |
| `WindowTextSystem::em_layout_width` | method | 926 |  | `(WindowTextSystem).EmLayoutWidth` | todo |
| `LineWrapperHandle` | struct | 938 |  | `LineWrapperHandle` | todo |
| `FontWeight` | struct | 975 |  | `FontWeight` | todo |
| `FontWeight::THIN` | assoc-const | 1006 |  | `FontWeightThin` | todo |
| `FontWeight::EXTRA_LIGHT` | assoc-const | 1008 |  | `FontWeightExtraLight` | todo |
| `FontWeight::LIGHT` | assoc-const | 1010 |  | `FontWeightLight` | todo |
| `FontWeight::NORMAL` | assoc-const | 1012 |  | `FontWeightNormal` | todo |
| `FontWeight::MEDIUM` | assoc-const | 1014 |  | `FontWeightMedium` | todo |
| `FontWeight::SEMIBOLD` | assoc-const | 1016 |  | `FontWeightSemibold` | todo |
| `FontWeight::BOLD` | assoc-const | 1018 |  | `FontWeightBold` | todo |
| `FontWeight::EXTRA_BOLD` | assoc-const | 1020 |  | `FontWeightExtraBold` | todo |
| `FontWeight::BLACK` | assoc-const | 1022 |  | `FontWeightBlack` | todo |
| `FontWeight::ALL` | assoc-const | 1025 |  | `FontWeightAll` | todo |
| `FontStyle` | enum | 1057 |  | `FontStyle` | todo |
| `TextRun` | struct | 1075 |  | `TextRun` | todo |
| `GlyphId` | struct | 1102 |  | `GlyphID` | todo |
| `RenderGlyphParams` | struct | 1111 |  | `RenderGlyphParams` | todo |
| `Font` | struct | 1139 |  | `Font` | todo |
| `font` | fn | 1165 |  | `NewFont` | todo |
| `Font::bold` | method | 1177 |  | `(Font).Bold` | todo |
| `Font::italic` | method | 1183 |  | `(Font).Italic` | todo |
| `FontMetrics` | struct | 1192 |  | `FontMetrics` | todo |
| `FontMetrics::ascent` | method | 1225 |  | `(FontMetrics).Ascent` | todo |
| `FontMetrics::descent` | method | 1230 |  | `(FontMetrics).Descent` | todo |
| `FontMetrics::line_gap` | method | 1235 |  | `(FontMetrics).LineGap` | todo |
| `FontMetrics::underline_position` | method | 1240 |  | `(FontMetrics).UnderlinePosition` | todo |
| `FontMetrics::underline_thickness` | method | 1245 |  | `(FontMetrics).UnderlineThickness` | todo |
| `FontMetrics::cap_height` | method | 1250 |  | `(FontMetrics).CapHeight` | todo |
| `FontMetrics::x_height` | method | 1255 |  | `(FontMetrics).XHeight` | todo |
| `FontMetrics::bounding_box` | method | 1260 |  | `(FontMetrics).BoundingBox` | todo |
| `font_name_with_fallbacks` | fn | 1267 |  | `FontNameWithFallbacks` | todo |
| `font_name_with_fallbacks_shared` | fn | 1281 |  | `FontNameWithFallbacksShared` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `caches_resolved_font_fallbacks` | test | 1302 |  | `TestTextSystemFontResolutionTests_CachesResolvedFontFallbacks` | todo |
| `pooled_line_wrappers_do_not_retain_text_system` | test | 1329 |  | `TestTextSystemLineWrapperPoolTests_PooledLineWrappersDoNotRetainTextSystem` | todo |
| `bounds_retained_missing_glyphs` | test | 1351 |  | `TestTextSystemMissingGlyphTests_BoundsRetainedMissingGlyphs` | todo |
| `dropping_receiver_closes_and_clears_reports` | test | 1373 |  | `TestTextSystemMissingGlyphTests_DroppingReceiverClosesAndClearsReports` | todo |

## gpui/src/text_system/font_fallbacks.rs

Go target: `text_system_font_fallbacks.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FontFallbacks` | struct | 9 |  | `FontFallbacks` | todo |
| `FontFallbacks::fallback_list` | method | 13 |  | `(FontFallbacks).FallbackList` | todo |
| `FontFallbacks::from_fonts` | assoc-fn | 18 |  | `FontFallbacksFromFonts` | todo |

## gpui/src/text_system/font_features.rs

Go target: `text_system_font_features.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FontFeatures` | struct | 8 |  | `FontFeatures` | todo |
| `FontFeatures::disable_ligatures` | assoc-fn | 21 |  | `FontFeaturesDisableLigatures` | todo |
| `FontFeatures::tag_value_list` | method | 27 |  | `(FontFeatures).TagValueList` | todo |
| `FontFeatures::is_calt_enabled` | method | 34 |  | `(FontFeatures).IsCaltEnabled` | todo |

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
| `LineLayout` | struct | 20 |  | `LineLayout` | todo |
| `ShapedRun` | struct | 37 |  | `ShapedRun` | todo |
| `ShapedGlyph` | struct | 46 |  | `ShapedGlyph` | todo |
| `LineLayout::index_for_x` | method | 62 |  | `(LineLayout).IndexForX` | todo |
| `LineLayout::closest_index_for_x` | method | 79 |  | `(LineLayout).ClosestIndexForX` | todo |
| `LineLayout::x_for_index` | method | 109 |  | `(LineLayout).XForIndex` | todo |
| `LineLayout::font_id_for_index` | method | 121 |  | `(LineLayout).FontIDForIndex` | todo |
| `LineLayout::split_at` | method | 139 |  | `(LineLayout).SplitAt` | todo |
| `WrappedLineLayout` | struct | 297 |  | `WrappedLineLayout` | todo |
| `WrapBoundary` | struct | 310 |  | `WrapBoundary` | todo |
| `WrappedLineLayout::len` | method | 319 |  | `(WrappedLineLayout).Len` | todo |
| `WrappedLineLayout::width` | method | 324 |  | `(WrappedLineLayout).Width` | todo |
| `WrappedLineLayout::size` | method | 332 |  | `(WrappedLineLayout).Size` | todo |
| `WrappedLineLayout::ascent` | method | 340 |  | `(WrappedLineLayout).Ascent` | todo |
| `WrappedLineLayout::descent` | method | 345 |  | `(WrappedLineLayout).Descent` | todo |
| `WrappedLineLayout::wrap_boundaries` | method | 350 |  | `(WrappedLineLayout).WrapBoundaries` | todo |
| `WrappedLineLayout::font_size` | method | 355 |  | `(WrappedLineLayout).FontSize` | todo |
| `WrappedLineLayout::runs` | method | 360 |  | `(WrappedLineLayout).Runs` | todo |
| `WrappedLineLayout::index_for_position` | method | 367 |  | `(WrappedLineLayout).IndexForPosition` | todo |
| `WrappedLineLayout::closest_index_for_position` | method | 380 |  | `(WrappedLineLayout).ClosestIndexForPosition` | todo |
| `WrappedLineLayout::position_for_index` | method | 447 |  | `(WrappedLineLayout).PositionForIndex` | todo |
| `FontRun` | struct | 780 |  | `FontRun` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `shape_line_shares_its_text_with_the_layout_cache` | test | 863 |  | `TestLineLayout_ShapeLineSharesItsTextWithTheLayoutCache` | todo |
| `test_wrap_preserves_shaped_clusters` | test | 912 |  | `TestLineLayout_WrapPreservesShapedClusters` | todo |
| `test_wrap_right_to_left_text` | test | 942 |  | `TestLineLayout_WrapRightToLeftText` | todo |
| `test_wrap_ignores_breaks_inside_shaped_clusters` | test | 971 |  | `TestLineLayout_WrapIgnoresBreaksInsideShapedClusters` | todo |
| `test_force_width_latin_unchanged` | test | 993 |  | `TestLineLayout_ForceWidthLatinUnchanged` | todo |
| `test_force_width_combining_marks_not_advanced` | test | 1004 |  | `TestLineLayout_ForceWidthCombiningMarksNotAdvanced` | todo |
| `test_force_width_base_after_combining_mark` | test | 1019 |  | `TestLineLayout_ForceWidthBaseAfterCombiningMark` | todo |
| `test_force_width_multiple_combining_marks` | test | 1030 |  | `TestLineLayout_ForceWidthMultipleCombiningMarks` | todo |
| `test_force_width_corrects_drifted_base_positions` | test | 1047 |  | `TestLineLayout_ForceWidthCorrectsDriftedBasePositions` | todo |
| `test_force_width_combining_mark_after_within_tolerance_base` | test | 1063 |  | `TestLineLayout_ForceWidthCombiningMarkAfterWithinToleranceBase` | todo |
| `test_wrap_ignores_the_sign_of_a_zero_wrap_width_in_the_cache` | test | 1076 |  | `TestLineLayout_WrapIgnoresTheSignOfAZeroWrapWidthInTheCache` | todo |

## gpui/src/text_system/line_wrapper.rs

Go target: `text_system_line_wrapper.go` (package `gpui`), tests in `text_system_line_wrapper_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TruncateFrom` | enum | 8 |  | `TruncateFrom` | todo |
| `IndentAdjustment` | enum | 19 |  | `IndentAdjustment` | todo |
| `LineWrapper` | struct | 30 |  | `LineWrapper` | todo |
| `LineWrapper::MAX_INDENT` | assoc-const | 55 |  | `LineWrapperMaxIndent` | todo |
| `LineWrapper::wrap_line` | method | 77 |  | `(LineWrapper).WrapLine` | todo |
| `LineWrapper::should_truncate_line` | method | 240 |  | `(LineWrapper).ShouldTruncateLine` | todo |
| `LineWrapper::truncate_line` | method | 347 |  | `(LineWrapper).TruncateLine` | todo |
| `LineWrapper::truncate_wrapped_line` | method | 409 |  | `(LineWrapper).TruncateWrappedLine` | todo |
| `LineFragment` | enum | 684 |  | `LineFragment` | todo |
| `LineFragment::text` | assoc-fn | 701 |  | `LineFragmentText` | todo |
| `LineFragment::element` | assoc-fn | 706 |  | `LineFragmentElement` | todo |
| `Boundary` | struct | 744 |  | `Boundary` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_emergency_wrap_preserves_graphemes` | test | 851 |  | `TestLineWrapper_EmergencyWrapPreservesGraphemes` | todo |
| `test_emergency_wrap_exact_boundaries` | test | 856 |  | `TestLineWrapper_EmergencyWrapExactBoundaries` | todo |
| `test_emergency_wrap_elements_and_graphemes` | test | 897 |  | `TestLineWrapper_EmergencyWrapElementsAndGraphemes` | todo |
| `test_wrap_line` | test | 941 |  | `TestLineWrapper_WrapLine` | todo |
| `test_wrap_line_break_opportunities` | test | 1112 |  | `TestLineWrapper_WrapLineBreakOpportunities` | todo |
| `test_truncate_line_end` | test | 1141 |  | `TestLineWrapper_TruncateLineEnd` | todo |
| `test_truncate_line_start` | test | 1190 |  | `TestLineWrapper_TruncateLineStart` | todo |
| `test_truncate_multiple_runs_end` | test | 1240 |  | `TestLineWrapper_TruncateMultipleRunsEnd` | todo |
| `test_truncate_multiple_runs_start` | test | 1299 |  | `TestLineWrapper_TruncateMultipleRunsStart` | todo |
| `test_update_run_after_truncation_end` | test | 1364 |  | `TestLineWrapper_UpdateRunAfterTruncationEnd` | todo |
| `test_wrap_shaped_line` | crate::test | 1404 | macos | `TestLineWrapper_WrapShapedLine` | todo |
| `test_multiline_truncation_fits_within_wrapped_lines` | test | 1459 |  | `TestLineWrapper_MultilineTruncationFitsWithinWrappedLines` | todo |
| `test_multiline_truncation_no_truncation_needed` | test | 1513 |  | `TestLineWrapper_MultilineTruncationNoTruncationNeeded` | todo |
| `test_multiline_truncation_three_lines` | test | 1540 |  | `TestLineWrapper_MultilineTruncationThreeLines` | todo |
| `test_multiline_truncation_preserves_breaks_after_empty_lines` | test | 1581 |  | `TestLineWrapper_MultilineTruncationPreservesBreaksAfterEmptyLines` | todo |
| `test_multiline_truncation_with_newlines` | test | 1604 |  | `TestLineWrapper_MultilineTruncationWithNewlines` | todo |
| `test_multiline_truncation_newline_on_last_line` | test | 1652 |  | `TestLineWrapper_MultilineTruncationNewlineOnLastLine` | todo |
| `test_truncate_line_middle` | test | 1682 |  | `TestLineWrapper_TruncateLineMiddle` | todo |
| `test_multiline_truncation_trailing_newline` | test | 1757 |  | `TestLineWrapper_MultilineTruncationTrailingNewline` | todo |
| `test_multiline_truncation_newline_fits_exactly` | test | 1784 |  | `TestLineWrapper_MultilineTruncationNewlineFitsExactly` | todo |
| `test_extra_columns_overflow_guard` | test | 1812 |  | `TestLineWrapper_ExtraColumnsOverflowGuard` | todo |

