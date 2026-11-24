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

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-GOTEXT-CTOR | `CosmicTextSystem::new` / `new_without_system_fonts` are `NewTextSystem` / `NewTextSystemWithoutSystemFonts` (the type keeps upstream's name); cosmic-text, fontdb, swash metrics and skrifa line metrics are re-expressed on go-text/typesetting in `fontdb.go`, `font.go`, `shape.go` and `fallback_*.go` | `gpui_wgpu/src/cosmic_text_system.rs:131`, `:147` | Constructor names fixed by STAGES.md M2-gotext-layout; cosmic-text has no Go port |
