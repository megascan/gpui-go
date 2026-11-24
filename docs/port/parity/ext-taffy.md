# Parity: ext-taffy

Scope: taffy 0.15.0 (`src/`), with the features gpui enables (`gpui/Cargo.toml:108-116`, default features off): `std`, `taffy_tree`, `flexbox`, `grid`, `block_layout`, `float_layout`, `content_size`. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md. Go package: `internal/taffy` (package `taffy`).

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`, `blocked:<stage>`. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/taffy-0.15.0/`; the integration tests (`tests/`) from `_ext/taffy` (tag v0.15.0). Rows of `tests/xml` fixtures and `tests/hand_written` are added by M0-taffy-fixtures and M0-taffy-handwritten-a/b.

The rows below were seeded from a scan of `src/` that lists `pub` items, trait methods and tests. Trait impls are not in the scan; each stage adds `trait-impl` rows for the impls of its files. The scan drops the names of inner test modules; a stage fixes those rows when it ports the tests (`module::test` with the module after the file stem in the Go name, as in `util/math.rs`). The Cfg column lists the `cfg(feature = ...)` names seen on an item and is not a gate: read each attribute (`not(...)` alternatives appear as the bare feature name).

Notes on the port (conventions every taffy stage follows):

- Files: one Go file per Rust file, named by its path under `src/` with `/` replaced by `_` (`style/dimension.rs` -> `style_dimension.go`, `style/mod.rs` -> `style_mod.go`). Tests live in `<stem>_test.go`. Test names use the Go file stem, not the Rust basename, so the several `mod.rs` files cannot collide: `fn foo` in `style/dimension.rs` inside `mod expand_tests` is `TestStyleDimension_ExpandTests_Foo`; the plain `tests` module adds nothing; a leading `test_` is dropped.
- Visibility: `pub` items are exported; `pub(crate)`, `pub(super)` and private items are unexported camelCase (`zip_size` -> `zipSize`, compute functions likewise).
- Names follow PARITY N1-N14. Field-less enums are `type X uint8` with `XVariant` constants. Data-carrying enums are comparable tagged structs with unexported `kind` / `value` fields, a `Kind()` accessor returning `XKind` (constants `XKindVariant`), constructor functions `XVariant(v)` for variants with a payload and package vars `XVariant` for unit variants (`AvailableSpaceDefinite(v)`, `AvailableSpaceMinContent`, `ExpandedDimensionAuto`); the `Expanded*` types also have `Value()`. Code inside the package reads the fields directly.
- Newtypes over `CompactLength` (`LengthPercentage`, `LengthPercentageAuto`, `Dimension`, and the grid track types later) are structs with one unexported field `inner`; upstream's `.0` is `.inner`.
- `usize` is `int`; `u16`, `i16`, `u32`, `f32` keep their Go equivalents. `Option<T>` is `taffy.Option[T]`, an alias of `gpuiutil.Option[T]`; inside the package `some(v)` / `none[T]()` build it. Returned `Option<T>` stays `Option[T]` in this package (it is passed straight into the maybe-math helpers), not `(T, bool)`.
- Geometry: `Rect[T]`, `Line[T]`, `Size[T]`, `Point[T]`, `MinMax[Min, Max]` take any element type, field order as upstream. Methods with closures of other types are Go generic methods (`Map[R]`, `ZipMap[Other, Ret]`, `zipSize[U, R]`).
- Impls bounded by an operator trait (`impl<T: Add> Rect<T>`, `impl Add for Size<T>`, `Line::sum`, ...) and methods of `impl Size<f32>` / `impl Rect<f32>` are methods on the generic type that work for numeric element types (an unexported `add` / `sub` dispatch, or a `*Size[float32]` assertion) and panic naming the type otherwise (`dev:D-COND`). They do not allocate (`TestGeometry_ArithmeticDoesNotAllocate`). Upstream's `+` / `-` on these types is `.Add(x)` / `.Sub(x)`.
- Consts and associated functions of an impl on one instantiation are free functions or package vars named `<Type><Inst><Name>` (PARITY N7), `new` -> `New<Type><Inst>`, with Inst `F32` (`f32`), `OptionF32` (`Option<f32>`), `Option` (`Option<T>`), `Bool`, `Dimension`, `AvailableSpace`: `SizeF32Zero`, `RectOptionF32None`, `NewSizeOptionF32`, `SizeDimensionFromLengths`. Methods with `self` of a non-numeric instantiation are free functions of the same shape taking the receiver first: `SizeOptionF32MaybeApplyAspectRatio(s, r)`, `SizeAvailableSpaceIntoOptions(s)`, `SizeOptionUnwrapOr(s, alt)`. A `Type<T>::zero()`-style associated fn generic over `T` keeps `<Type><Name>` (`SizeZero`), which is why the instantiation names carry Inst.
- `impl From<A> for B` is `BFromA` (`SizeFromPoint`, `DimensionFromLengthPercentage`, `AvailableSpaceFromOptionF32`).
- style_helpers traits with an associated const (`TaffyZero::ZERO`, `TaffyAuto::AUTO`, `TaffyMinContent::MIN_CONTENT`, `TaffyMaxContent::MAX_CONTENT`) are generic interfaces with one method on a value receiver (`Zero() T`, `Auto() T`, `MinContent() T`, `MaxContent() T`), so a generic helper calls it on the zero value (`var z T; z.Zero()`). `FromLength`, `FromPercent`, `FromFr` take `float64` (upstream `Input: Into<f64>`), `TaffyFitContent` takes a `LengthPercentage`. The const value itself is a package var `<Type><Const>` (`CompactLengthZero`, `DimensionZero`, `AvailableSpaceZero`) unless an associated fn of the same name exists (`Dimension::auto()` -> `DimensionAuto()`, which the `AUTO` const reuses). The interfaces live in style_helpers.go; the generic helpers (`Zero[T]`, `Length[T]`, ...) take the interface as constraint, and `Point` / `Line` / `Size` / `Rect` / `MinMax` implement them for every element type with a runtime check (D-COND).
- MaybeMath (`D-TAFFY-MAYBEMATH`): Go cannot add methods to `Option[float32]` or `float32`, and `AvailableSpace` has two impls. The impls are: lhs `Option<f32>` -> `OptionF32Maybe<Op>`, lhs `f32` -> `F32Maybe<Op>`, lhs `AvailableSpace` -> method `Maybe<Op>`; an `f32` right-hand side adds the suffix `F32` (an `Option<f32>` right-hand side adds nothing); the blanket `impl for Size<T>` is `Size<Lhs>Maybe<Op>[F32]` (`SizeOptionF32MaybeSub`, `SizeF32MaybeMax`, `SizeAvailableSpaceMaybeMinF32`). `x.maybe_min(y)` with `x: Option<f32>, y: f32` is `OptionF32MaybeMinF32(x, y)`.
- Floating point: arithmetic stays in `float32`, in upstream order. Rust's `f32::min` / `max` return the non-NaN operand, Go's `min` / `max` / `math.Min` propagate NaN, so every `.min()` / `.max()` uses `f32Min` / `f32Max` (util_sys.go); never the builtins. `sys::round` is `floor(v + 0.5)` with the add in `float32`, not `math.Round`. The Go spec lets the compiler fuse a multiplication with a later addition into an FMA, even across statements and through inlined calls (arm64 does; Rust never does), and only an explicit conversion stops it: every float32 multiplication is written `float32(a * b)`. Signed-zero results of `min(0.0, -0.0)` may differ from LLVM's `minnum`; layouts never depend on them.
- Panics keep upstream's text: `called `Option::unwrap()` on a `None` value`, and `unreachable!(msg)` as `internal error: entered unreachable code: msg`.
- Forward references: a stage that needs a type or function from a later stage declares it in internal/taffy/stubs.go with its final signature and a `panic("taffy: unimplemented <symbol>")` body; the owning stage moves it out. Tests never reach a stub.

## src/compute/block.rs

Go target: `internal/taffy/compute_block.go` (package `taffy`), tests in `compute_block_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_block_align_content_offset` | fn | 35 |  | `ComputeBlockAlignContentOffset` | done |
| `BlockFormattingContext` | struct | 41 |  | `BlockFormattingContext` | done |
| `BlockFormattingContext::new` | assoc-fn | 58 |  | `NewBlockFormattingContext` | done |
| `BlockFormattingContext::root_block_context` | method | 63 |  | `(BlockFormattingContext).RootBlockContext` | done |
| `BlockContext` | struct | 82 |  | `BlockContext` | done |
| `BlockContext::sub_context` | method | 112 |  | `(BlockContext).SubContext` | done |
| `BlockContext::is_bfc_root` | method | 131 |  | `(BlockContext).IsBfcRoot` | done |
| `BlockContext::set_width` | method | 143 | float_layout | `(BlockContext).SetWidth` | done |
| `BlockContext::apply_content_box_inset` | method | 149 | float_layout | `(BlockContext).ApplyContentBoxInset` | done |
| `BlockContext::has_floats` | method | 156 | float_layout | `(BlockContext).HasFloats` | done |
| `BlockContext::has_active_floats` | method | 162 | float_layout | `(BlockContext).HasActiveFloats` | done |
| `BlockContext::place_floated_box` | method | 167 | float_layout | `(BlockContext).PlaceFloatedBox` | done |
| `BlockContext::find_content_slot` | method | 194 | float_layout | `(BlockContext).FindContentSlot` | done |
| `BlockContext::find_bfc_slot` | method | 204 | float_layout | `(BlockContext).FindBfcSlot` | done |
| `BlockContext::cleared_threshold` | method | 226 | float_layout | `(BlockContext).ClearedThreshold` | done |
| `BlockContext::has_adjoining_float` | method | 232 | float_layout | `(BlockContext).HasAdjoiningFloat` | done |
| `BlockContext::add_child_floated_content_height_contribution` | method | 265 | float_layout | `(BlockContext).AddChildFloatedContentHeightContribution` | done |
| `BlockContext::floated_content_height_contribution` | method | 270 | float_layout | `(BlockContext).FloatedContentHeightContribution` | done |
| `compute_block_layout` | fn | 374 |  | `ComputeBlockLayout` | done |
| `BlockItem` (private) | struct | 290 |  | `blockItem` | done |
| `BlockContext::merge_adjoining_floats` (private) | method | 242 | float_layout | `(BlockContext).mergeAdjoiningFloats` | done |
| `BlockContext::commit_strut` (private) | method | 250 | float_layout | `(BlockContext).commitStrut` | done |
| `BlockContext::top_adjoining_floats` (private) | method | 259 | float_layout | `(BlockContext).getTopAdjoiningFloats` | done |
| `BlockContext::float_content_contribution` (private) | method | 279 | !float_layout | - | n/a:D-TAFFY-FEATURE |
| `compute_inner` (private) | fn | 478 |  | `computeInner` | done |
| `generate_item_list` (private) | fn | 876 |  | `generateItemList` | done |
| `resolve_stretch_height` (private) | fn | 982 |  | `resolveStretchHeight` | done |
| `determine_content_based_container_width` (private) | fn | 1003 |  | `determineContentBasedContainerWidth` | done |
| `perform_final_layout_on_in_flow_children` (private) | fn | 1061 |  | `performFinalLayoutOnInFlowChildren` | done |

## src/compute/common/alignment.rs

Go target: `internal/taffy/compute_common_alignment.go` (package `taffy`), tests in `compute_common_alignment_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `resolve_self_alignment_safety` (`pub(crate)`) | fn | 11 |  | `resolveSelfAlignmentSafety` | done |
| `apply_alignment_fallback` (`pub(crate)`) | fn | 28 |  | `applyAlignmentFallback` | done |
| `compute_alignment_offset` (`pub(crate)`) | fn | 72 |  | `computeAlignmentOffset` | done |

## src/compute/common/mod.rs

Go target: `internal/taffy/compute_common_mod.go` (package `taffy`), tests in `compute_common_mod_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `alignment`, `sizing_keyword`, `scrollable_overflow` modules | mod | 2 | content_size | - | n/a:D-TAFFY-SYS |

## src/compute/common/scrollable_overflow.rs

Go target: `internal/taffy/compute_common_scrollable_overflow.go` (package `taffy`), tests in `compute_common_scrollable_overflow_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_scrollable_overflow_contribution` (`pub(crate)`) | fn | 23 |  | `computeScrollableOverflowContribution` | done |

## src/compute/common/sizing_keyword.rs

Go target: `internal/taffy/compute_common_sizing_keyword.go` (package `taffy`), tests in `compute_common_sizing_keyword_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SizingKeywordResolution` (`pub(crate)`) | enum | 9 |  | `sizingKeywordResolution` | done |
| `resolve_sizing_keyword` (`pub(crate)`) | fn | 28 | calc | `resolveSizingKeyword` | done |
| `resolve_absolute_sizing_keywords` (`pub(crate)`) | fn | 65 |  | `resolveAbsoluteSizingKeywords` | done |

## src/compute/flexbox.rs

Go target: `internal/taffy/compute_flexbox.go` (package `taffy`), tests in `compute_flexbox_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_flexbox_layout` | fn | 234 |  | `ComputeFlexboxLayout` | done |
| `FlexItem` (private) | struct | 26 |  | `flexItem` | done |
| `FlexLine` (private) | struct | 134 |  | `flexLine` | done |
| `AlgoConstants` (private) | struct | 144 |  | `algoConstants` | done |
| `compute_preliminary` (private) | fn | 329 |  | `computePreliminary` | done |
| `resolve_normal_alignment` (private) | fn | 570 |  | `resolveNormalAlignment` | done |
| `compute_constants` (private) | fn | 579 |  | `computeConstants` | done |
| `resolve_relative_inset` (private) | fn | 696 |  | `resolveRelativeInset` | done |
| `generate_anonymous_flex_items` (private) | fn | 722 |  | `generateAnonymousFlexItems` | done |
| `determine_available_space` (private) | fn | 834 |  | `determineAvailableSpace` | done |
| `determine_flex_base_size` (private) | fn | 885 |  | `determineFlexBaseSize` | done |
| `collect_flex_lines` (private) | fn | 1181 |  | `collectFlexLines` | done |
| `item_known_dimension_definiteness` (private) | fn | 1346 |  | `itemKnownDimensionDefiniteness` | done |
| `determine_container_main_size` (private) | fn | 1369 |  | `determineContainerMainSize` | done |
| `resolve_flexible_lengths` (private) | fn | 1696 |  | `resolveFlexibleLengths` | done |
| `determine_hypothetical_cross_size` (private) | fn | 1892 |  | `determineHypotheticalCrossSize` | done |
| `calculate_children_base_lines` (private) | fn | 1978 |  | `calculateChildrenBaseLines` | done |
| `calculate_cross_size` (private) | fn | 2068 |  | `calculateCrossSize` | done |
| `handle_align_content_stretch` (private) | fn | 2132 |  | `handleAlignContentStretch` | done |
| `determine_used_cross_size` (private) | fn | 2168 |  | `determineUsedCrossSize` | done |
| `distribute_remaining_free_space` (private) | fn | 2237 |  | `distributeRemainingFreeSpace` | done |
| `resolve_cross_axis_auto_margins` (private) | fn | 2309 |  | `resolveCrossAxisAutoMargins` | done |
| `align_flex_items_along_cross_axis` (private) | fn | 2364 |  | `alignFlexItemsAlongCrossAxis` | done |
| `determine_container_cross_size` (private) | fn | 2458 |  | `determineContainerCrossSize` | done |
| `align_flex_lines_per_align_content` (private) | fn | 2489 |  | `alignFlexLinesPerAlignContent` | done |
| `calculate_flex_item` (private) | fn | 2511 |  | `calculateFlexItem` | done |
| `calculate_layout_line` (private) | fn | 2644 |  | `calculateLayoutLine` | done |
| `final_layout_pass` (private) | fn | 2707 |  | `finalLayoutPass` | done |
| `collect_oof_candidates` (private) | fn | 2771 |  | `collectOofCandidates` | done |
| `sum_axis_gaps` (private) | fn | 2924 |  | `sumAxisGaps` | done |
| `collect_balanced_flex_lines`, `balance` module | fn | 1271 | flexbox_balance | - | n/a:D-TAFFY-FEATURE |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `zero_sized_item_rule` | test | 3399 | flexbox_balance | `TestComputeFlexbox_ZeroSizedItemRule` | n/a:D-TAFFY-FEATURE |
| `matches_naive_dp` | test | 3414 | flexbox_balance | `TestComputeFlexbox_MatchesNaiveDp` | n/a:D-TAFFY-FEATURE |

## src/compute/float.rs

Go target: `internal/taffy/compute_float.go` (package `taffy`), tests in `compute_float_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ContentSlot` | struct | 41 |  | `ContentSlot` | done |
| `BfcSlot` | struct | 60 |  | `BfcSlot` | done |
| `PlacedFloatedBox` | struct | 79 |  | `PlacedFloatedBox` | done |
| `FloatContext` | struct | 211 |  | `FloatContext` | done |
| `FloatContext::new` | assoc-fn | 256 |  | `NewFloatContext` | done |
| `FloatContext::has_floats` | method | 262 |  | `(FloatContext).HasFloats` | done |
| `FloatContext::has_active_floats` | method | 268 |  | `(FloatContext).HasActiveFloats` | done |
| `FloatContext::set_width` | method | 273 |  | `(FloatContext).SetWidth` | done |
| `FloatContext::left_floats` | method | 278 |  | `(FloatContext).LeftFloats` | done |
| `FloatContext::right_floats` | method | 283 |  | `(FloatContext).RightFloats` | done |
| `FloatContext::place_floated_box` | method | 311 |  | `(FloatContext).PlaceFloatedBox` | done |
| `FloatContext::cleared_threshold` | method | 581 |  | `(FloatContext).ClearedThreshold` | done |
| `FloatContext::find_content_slot` | method | 594 |  | `(FloatContext).FindContentSlot` | done |
| `FloatContext::find_bfc_slot` | method | 667 |  | `(FloatContext).FindBfcSlot` | done |
| `FIT_TOLERANCE` (`pub(crate)`) | const | 36 |  | `fitTolerance` | done |
| `Segment` (private) | struct | 95 |  | `segment` | done |
| `Segment::fits_float_width` (private) | method | 110 |  | `(*segment).fitsFloatWidth` | done |
| `float_fits_horizontally` (private) | fn | 134 |  | `floatFitsHorizontally` | done |
| `FloatFitter` (private) | struct | 156 |  | `floatFitter` | done |
| `FloatFitter::new` (private) | assoc-fn | 169 |  | `newFloatFitter` | done |
| `FloatFitter::union_insets` (private) | method | 176 |  | `(*floatFitter).unionInsets` | done |
| `FloatFitter::placed_inset` (private) | method | 183 |  | `(*floatFitter).placedInset` | done |
| `FloatFitter::fits_horiontally` (private) | method | 192 |  | `(*floatFitter).fitsHoriontally` | done |
| `FloatFitter::add_height` (private) | method | 199 |  | `(*floatFitter).addHeight` | done |
| `FloatFitter::fits_vertically` (private) | method | 204 |  | `(*floatFitter).fitsVertically` | done |
| `FloatContext::subdivide_segment` (private) | method | 289 |  | `(*FloatContext).subdivideSegment` | done |
| `FloatContext::update_last_placed_float` (private) | method | 304 |  | `(*FloatContext).updateLastPlacedFloat` | done |
| `FloatContext::place_floated_box_inner` (private) | method | 345 |  | `(*FloatContext).placeFloatedBoxInner` | done |
| `FloatContext::cleared_segment` (private) | method | 569 |  | `(*FloatContext).clearedSegment` | done |
| `FloatIntrinsicWidthCalculator` | struct | 747 |  | `FloatIntrinsicWidthCalculator` | done |
| `FloatIntrinsicWidthCalculator::new` | assoc-fn | 760 |  | `NewFloatIntrinsicWidthCalculator` | done |
| `FloatIntrinsicWidthCalculator::add_float` | method | 765 |  | `(FloatIntrinsicWidthCalculator).AddFloat` | done |
| `FloatIntrinsicWidthCalculator::result` | method | 788 |  | `(FloatIntrinsicWidthCalculator).Result` | done |

## src/compute/grid/alignment.rs

Go target: `internal/taffy/compute_grid_alignment.go` (package `taffy`), tests in `compute_grid_alignment_test.go`

No public items or tests.

## src/compute/grid/explicit_grid.rs

Go target: `internal/taffy/compute_grid_explicit_grid.go` (package `taffy`), tests in `compute_grid_explicit_grid_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `explicit_grid_sizing_no_repeats` | test | 397 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingNoRepeats` | todo |
| `explicit_grid_sizing_auto_fill_exact_fit` | test | 421 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillExactFit` | todo |
| `explicit_grid_sizing_auto_fill_non_exact_fit` | test | 452 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillNonExactFit` | todo |
| `explicit_grid_sizing_auto_fill_min_size_exact_fit` | test | 483 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillMinSizeExactFit` | todo |
| `explicit_grid_sizing_auto_fill_min_size_non_exact_fit` | test | 514 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillMinSizeNonExactFit` | todo |
| `explicit_grid_sizing_auto_fill_multiple_repeated_tracks` | test | 545 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillMultipleRepeatedTracks` | todo |
| `explicit_grid_sizing_auto_fill_gap` | test | 576 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingAutoFillGap` | todo |
| `explicit_grid_sizing_no_defined_size` | test | 608 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingNoDefinedSize` | todo |
| `explicit_grid_sizing_mix_repeated_and_non_repeated` | test | 639 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingMixRepeatedAndNonRepeated` | todo |
| `explicit_grid_sizing_mix_with_padding` | test | 671 |  | `TestComputeGridExplicitGrid_ExplicitGridSizingMixWithPadding` | todo |
| `auto_repeat_clamping_preserves_template_source_order` | test | 703 |  | `TestComputeGridExplicitGrid_AutoRepeatClampingPreservesTemplateSourceOrder` | todo |
| `explicit_track_count_accumulation_saturates` | test | 747 |  | `TestComputeGridExplicitGrid_ExplicitTrackCountAccumulationSaturates` | todo |
| `test_initialize_grid_tracks` | test | 771 |  | `TestComputeGridExplicitGrid_InitializeGridTracks` | todo |

## src/compute/grid/implicit_grid.rs

Go target: `internal/taffy/compute_grid_implicit_grid.go` (package `taffy`), tests in `compute_grid_implicit_grid_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `child_min_max_line_auto` | test | 175 |  | `TestComputeGridImplicitGrid_ChildMinMaxLineAuto` | todo |
| `child_min_max_line_negative_track` | test | 185 |  | `TestComputeGridImplicitGrid_ChildMinMaxLineNegativeTrack` | todo |
| `explicit_grid_sizing_with_children` | test | 216 |  | `TestComputeGridImplicitGrid_ExplicitGridSizingWithChildren` | todo |
| `negative_implicit_grid_sizing` | test | 233 |  | `TestComputeGridImplicitGrid_NegativeImplicitGridSizing` | todo |

## src/compute/grid/mod.rs

Go target: `internal/taffy/compute_grid_mod.go` (package `taffy`), tests in `compute_grid_mod_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_grid_layout` | fn | 219 |  | `ComputeGridLayout` | todo |
| `DetailedGridInfo` | struct | 1062 |  | `DetailedGridInfo` | todo |
| `DetailedGridInfo::write_grid_template_rows` | method | 1157 |  | `(DetailedGridInfo).WriteGridTemplateRows` | todo |
| `DetailedGridInfo::write_grid_template_columns` | method | 1165 |  | `(DetailedGridInfo).WriteGridTemplateColumns` | todo |
| `DetailedGridInfo::grid_template_rows` | method | 1172 |  | `(DetailedGridInfo).GridTemplateRows` | todo |
| `DetailedGridInfo::grid_template_columns` | method | 1179 |  | `(DetailedGridInfo).GridTemplateColumns` | todo |
| `DetailedGridInfo::resolve_absolute_grid_area` | method | 1185 |  | `(DetailedGridInfo).ResolveAbsoluteGridArea` | todo |
| `DetailedGridInfo::item_grid_area` | method | 1219 |  | `(DetailedGridInfo).ItemGridArea` | todo |
| `DetailedGridTracksInfo` | struct | 1233 |  | `DetailedGridTracksInfo` | todo |
| `DetailedGridTracksInfo::names_for_line` | method | 1306 |  | `(DetailedGridTracksInfo).NamesForLine` | todo |
| `DetailedGridTracksInfo::iter_line_names` | method | 1316 |  | `(DetailedGridTracksInfo).IterLineNames` | todo |
| `DetailedGridTracksInfo::write_track_list` | method | 1330 |  | `(DetailedGridTracksInfo).WriteTrackList` | todo |
| `DetailedGridTracksInfo::to_track_list_string` | method | 1375 |  | `(DetailedGridTracksInfo).ToTrackListString` | todo |
| `DetailedGridItemsInfo` | struct | 1387 |  | `DetailedGridItemsInfo` | todo |

## src/compute/grid/placement.rs

Go target: `internal/taffy/compute_grid_placement.go` (package `taffy`), tests in `compute_grid_placement_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_only_fixed_placement` | test | 410 |  | `TestComputeGridPlacement_OnlyFixedPlacement` | todo |
| `test_placement_spanning_origin` | test | 429 |  | `TestComputeGridPlacement_PlacementSpanningOrigin` | todo |
| `test_only_auto_placement_row_flow` | test | 448 |  | `TestComputeGridPlacement_OnlyAutoPlacementRowFlow` | todo |
| `test_only_auto_placement_column_flow` | test | 472 |  | `TestComputeGridPlacement_OnlyAutoPlacementColumnFlow` | todo |
| `test_oversized_item` | test | 496 |  | `TestComputeGridPlacement_OversizedItem` | todo |
| `test_fixed_in_secondary_axis` | test | 512 |  | `TestComputeGridPlacement_FixedInSecondaryAxis` | todo |
| `test_definite_in_secondary_axis_with_fully_definite_negative` | test | 531 |  | `TestComputeGridPlacement_DefiniteInSecondaryAxisWithFullyDefiniteNegative` | todo |
| `test_dense_packing_algorithm` | test | 549 |  | `TestComputeGridPlacement_DensePackingAlgorithm` | todo |
| `test_sparse_packing_algorithm` | test | 567 |  | `TestComputeGridPlacement_SparsePackingAlgorithm` | todo |
| `test_auto_placement_in_negative_tracks` | test | 585 |  | `TestComputeGridPlacement_AutoPlacementInNegativeTracks` | todo |
| `test_overlarge_placement_is_clamped` | test | 603 |  | `TestComputeGridPlacement_OverlargePlacementIsClamped` | todo |
| `auto_placement_cursor_saturates_at_integer_bounds` | test | 627 |  | `TestComputeGridPlacement_AutoPlacementCursorSaturatesAtIntegerBounds` | todo |
| `indefinite_spans_saturate_at_integer_bounds` | test | 632 |  | `TestComputeGridPlacement_IndefiniteSpansSaturateAtIntegerBounds` | todo |

## src/compute/grid/track_sizing.rs

Go target: `internal/taffy/compute_grid_track_sizing.go` (package `taffy`), tests in `compute_grid_track_sizing_test.go`

No public items or tests.

## src/compute/grid/types/cell_occupancy.rs

Go target: `internal/taffy/compute_grid_types_cell_occupancy.go` (package `taffy`), tests in `compute_grid_types_cell_occupancy_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `paint_merges_touching_same_state_intervals` | test | 410 |  | `TestComputeGridTypesCellOccupancy_PaintMergesTouchingSameStateIntervals` | todo |
| `paint_does_not_merge_different_states` | test | 419 |  | `TestComputeGridTypesCellOccupancy_PaintDoesNotMergeDifferentStates` | todo |
| `paint_overwrites_overlapped_cells` | test | 427 |  | `TestComputeGridTypesCellOccupancy_PaintOverwritesOverlappedCells` | todo |
| `paint_overwrites_multiple_intervals` | test | 442 |  | `TestComputeGridTypesCellOccupancy_PaintOverwritesMultipleIntervals` | todo |
| `collision_extent_finds_extremal_occupied_cell` | test | 455 |  | `TestComputeGridTypesCellOccupancy_CollisionExtentFindsExtremalOccupiedCell` | todo |
| `state_at_distinguishes_occupied_intervals_and_gaps` | test | 467 |  | `TestComputeGridTypesCellOccupancy_StateAtDistinguishesOccupiedIntervalsAndGaps` | todo |
| `definitely_placed_overwrite_hides_auto_placed_cells` | test | 481 |  | `TestComputeGridTypesCellOccupancy_DefinitelyPlacedOverwriteHidesAutoPlacedCells` | todo |
| `negative_expansion_preserves_occupancy` | test | 501 |  | `TestComputeGridTypesCellOccupancy_NegativeExpansionPreservesOccupancy` | todo |
| `auto_placed_item_only_advances_cursor_in_first_secondary_track` | test | 526 |  | `TestComputeGridTypesCellOccupancy_AutoPlacedItemOnlyAdvancesCursorInFirstSecondaryTrack` | todo |
| `cursors_survive_interval_merging_and_grid_expansion` | test | 543 |  | `TestComputeGridTypesCellOccupancy_CursorsSurviveIntervalMergingAndGridExpansion` | todo |
| `alternating_spans_merge_occupancy_and_skip_the_occupied_prefix` | test | 567 |  | `TestComputeGridTypesCellOccupancy_AlternatingSpansMergeOccupancyAndSkipTheOccupiedPrefix` | todo |
| `collision_jump_returns_next_search_position` | test | 585 |  | `TestComputeGridTypesCellOccupancy_CollisionJumpReturnsNextSearchPosition` | todo |
| `occupied_track_jump_skips_non_empty_tracks` | test | 598 |  | `TestComputeGridTypesCellOccupancy_OccupiedTrackJumpSkipsNonEmptyTracks` | todo |

## src/compute/grid/types/coordinates.rs

Go target: `internal/taffy/compute_grid_types_coordinates.go` (package `taffy`), tests in `compute_grid_types_coordinates_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `GridLine` | struct | 32 |  | `GridLine` | todo |
| `GridLine::as_i16` | method | 42 |  | `(GridLine).AsI16` | todo |
| `OriginZeroLine` | struct | 68 |  | `OriginZeroLine` | todo |
| `GridCoordinate` | trait | 182 |  | `GridCoordinate` | todo |

## src/compute/grid/types/grid_item.rs

Go target: `internal/taffy/compute_grid_types_grid_item.go` (package `taffy`), tests in `compute_grid_types_grid_item_test.go`

No public items or tests.

## src/compute/grid/types/grid_track.rs

Go target: `internal/taffy/compute_grid_types_grid_track.go` (package `taffy`), tests in `compute_grid_types_grid_track_test.go`

No public items or tests.

## src/compute/grid/types/grid_track_counts.rs

Go target: `internal/taffy/compute_grid_types_grid_track_counts.go` (package `taffy`), tests in `compute_grid_types_grid_track_counts_test.go`

No public items or tests.

## src/compute/grid/types/mod.rs

Go target: `internal/taffy/compute_grid_types_mod.go` (package `taffy`), tests in `compute_grid_types_mod_test.go`

No public items or tests.

## src/compute/grid/types/named.rs

Go target: `internal/taffy/compute_grid_types_named.go` (package `taffy`), tests in `compute_grid_types_named_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `GridLineNames` | struct | 664 |  | `GridLineNames` | todo |
| `GridLineNames::is_empty` | method | 710 |  | `(GridLineNames).IsEmpty` | todo |
| `GridLineNames::line_count` | method | 715 |  | `(GridLineNames).LineCount` | todo |
| `GridLineNames::line` | method | 721 |  | `(GridLineNames).Line` | todo |
| `GridLineNames::iter` | method | 729 |  | `(GridLineNames).Iter` | todo |
| `GridLineNamesIter` | struct | 752 |  | `GridLineNamesIter` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `extreme_missing_named_line_indices_do_not_overflow` | test | 817 |  | `TestComputeGridTypesNamed_ExtremeMissingNamedLineIndicesDoNotOverflow` | todo |
| `large_named_span_does_not_wrap_negative` | test | 831 |  | `TestComputeGridTypesNamed_LargeNamedSpanDoesNotWrapNegative` | todo |
| `area_lines_saturate_when_converted_to_grid_lines` | test | 844 |  | `TestComputeGridTypesNamed_AreaLinesSaturateWhenConvertedToGridLines` | todo |
| `bare_ident_resolves_to_area_edge` | test | 913 |  | `TestComputeGridTypesNamed_BareIdentResolvesToAreaEdge` | todo |
| `bare_ident_resolves_to_edge_of_implicitly_named_area` | test | 924 |  | `TestComputeGridTypesNamed_BareIdentResolvesToEdgeOfImplicitlyNamedArea` | todo |
| `area_edges_are_lines_named_with_start_and_end_suffixes` | test | 931 |  | `TestComputeGridTypesNamed_AreaEdgesAreLinesNamedWithStartAndEndSuffixes` | todo |
| `area_edges_are_merged_with_explicitly_named_lines` | test | 942 |  | `TestComputeGridTypesNamed_AreaEdgesAreMergedWithExplicitlyNamedLines` | todo |
| `area_edges_are_only_matched_by_bare_idents` | test | 966 |  | `TestComputeGridTypesNamed_AreaEdgesAreOnlyMatchedByBareIdents` | todo |
| `area_edge_is_the_first_line_with_the_edge_name` | test | 980 |  | `TestComputeGridTypesNamed_AreaEdgeIsTheFirstLineWithTheEdgeName` | todo |

## src/compute/grid/util/mod.rs

Go target: `internal/taffy/compute_grid_util_mod.go` (package `taffy`), tests in `compute_grid_util_mod_test.go`

No public items or tests.

## src/compute/grid/util/test_helpers.rs

Go target: `internal/taffy/compute_grid_util_test_helpers.go` (package `taffy`), tests in `compute_grid_util_test_helpers_test.go`

No public items or tests.

## src/compute/leaf.rs

Go target: `internal/taffy/compute_leaf.go` (package `taffy`), tests in `compute_leaf_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_leaf_layout` | fn | 20 |  | `ComputeLeafLayout` | done |

## src/compute/mod.rs

Go target: `internal/taffy/compute_mod.go` (package `taffy`), tests in `compute_mod_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_root_layout` | fn | 77 |  | `ComputeRootLayout` | done |
| `compute_in_flow_root_layout` (private) | fn | 188 | block_layout | `computeInFlowRootLayout` | done |
| `impl LayoutContainingBlock + CacheTree` bound of `compute_root_layout` | type | 78 |  | `RootLayoutTree` | done |
| `impl LayoutPartialTree + CacheTree` bound of `compute_hidden_layout` | type | 456 |  | `HiddenLayoutTree` | done |
| `compute_cached_layout` | fn | 330 |  | `ComputeCachedLayout` | done |
| `round_layout` | fn | 375 |  | `RoundLayout` | done |
| `round_layout::round_layout_inner` (private) | fn | 379 |  | `roundLayoutInner` | done |
| `round_layout::round_scrollable_overflow_rect` (private) | fn | 441 | content_size | `roundScrollableOverflowRect` | done |
| `detailed_info` re-export module | mod | 471 | grid | - | n/a:D-TAFFY-FEATURE |
| `compute_hidden_layout` | fn | 456 |  | `ComputeHiddenLayout` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `hidden_layout_should_hide_recursively` | test | 486 |  | `TestComputeMod_HiddenLayoutShouldHideRecursively` | done |

## src/compute/oof.rs

Go target: `internal/taffy/compute_oof.go` (package `taffy`), tests in `compute_oof_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `resolve_static_offset` | fn | 35 |  | `ResolveStaticOffset` | done |
| `resolve_static_offset_axis` (private) | fn | 54 |  | `resolveStaticOffsetAxis` | done |
| `align_in_line` (private) | fn | 62 |  | `alignInLine` | done |
| `OofAlignment` (private) | enum | 73 |  | `oofAlignment` | done |
| `OofAxis` (private) | struct | 125 |  | `oofAxis` | done |
| `OofBoxLayout` (private) | struct | 418 |  | `oofBoxLayout` | done |
| `layout_oof_box` (private) | fn | 447 |  | `layoutOofBox` | done |
| `perform_oof_layout` (private) | fn | 727 |  | `performOofLayout` | done |
| `compute_oof_layout` | fn | 348 |  | `ComputeOofLayout` | done |
| `OofLayoutResult` | struct | 370 |  | `OofLayoutResult` | done |
| `compute_oof_layout_for_area` | fn | 389 |  | `ComputeOofLayoutForArea` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `start_alignment` | test | 865 |  | `TestComputeOof_StartAlignment` | done |
| `end_alignment` | test | 872 |  | `TestComputeOof_EndAlignment` | done |
| `center_alignment` | test | 879 |  | `TestComputeOof_CenterAlignment` | done |
| `safe_alignment_falls_back_only_on_overflow` | test | 887 |  | `TestComputeOof_SafeAlignmentFallsBackOnlyOnOverflow` | done |
| `unsafe_alignment_never_falls_back` | test | 899 |  | `TestComputeOof_UnsafeAlignmentNeverFallsBack` | done |
| `degenerate_area` | test | 905 |  | `TestComputeOof_DegenerateArea` | done |

## src/geometry.rs

Go target: `internal/taffy/geometry.go` (package `taffy`), tests in `geometry_test.go`

The `pub(crate)` helpers are ported unexported: `InBothAbsAxis` (`inBothAbsAxis`, `get`), `Rect::zip_size`, `main_axis_sum`, `cross_axis_sum`, `main_start`, `main_end`, `cross_start`, `cross_end`, `Size::set_main`, `set_cross`, `with_main`, `with_cross`, `map_main`, `map_cross`, `main`, `cross`, `get`, `set`, `with`, `Point::main`, `cross`. The flexbox-gated ones call `FlexDirection.IsRow`, a stub until M0-taffy-style-mod. Upstream has no tests in this file; geometry_test.go covers it.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AbsoluteAxis` | enum | 13 |  | `AbsoluteAxis` | done |
| `AbsoluteAxis::other_axis` | method | 23 |  | `(AbsoluteAxis).OtherAxis` | done |
| `Size::get_abs` | method | 34 |  | `(Size).GetAbs` | done |
| `Size::transpose` | method | 43 |  | `(Size).Transpose` | done |
| `Rect::grid_axis_sum` | method | 51 |  | `(Rect).GridAxisSum` | dev:D-COND |
| `AbstractAxis` | enum | 62 |  | `AbstractAxis` | done |
| `AbstractAxis::other` | method | 72 |  | `(AbstractAxis).Other` | done |
| `AbstractAxis::as_abs_naive` | method | 82 |  | `(AbstractAxis).AsAbsNaive` | done |
| `Rect` | struct | 114 |  | `Rect` | done |
| `impl Add<Rect<U>> for Rect<T>` | trait-impl | 135 |  | `(Rect).Add` | dev:D-COND |
| `Rect::transpose` | method | 151 |  | `(Rect).Transpose` | done |
| `Rect::map` | method | 179 |  | `(Rect).Map` | done |
| `Rect::horizontal_components` | method | 187 |  | `(Rect).HorizontalComponents` | done |
| `Rect::vertical_components` | method | 192 |  | `(Rect).VerticalComponents` | done |
| `Rect::horizontal_axis_sum` | method | 207 |  | `(Rect).HorizontalAxisSum` | dev:D-COND |
| `Rect::vertical_axis_sum` | method | 217 |  | `(Rect).VerticalAxisSum` | dev:D-COND |
| `Rect::sum_axes` | method | 226 |  | `(Rect).SumAxes` | dev:D-COND |
| `Rect<Option<f32>>::NONE` | assoc-const | 306 |  | `RectOptionF32None` | done |
| `Rect<f32>::ZERO` | assoc-const | 311 |  | `RectF32Zero` | done |
| `Rect<f32>::new` | assoc-fn | 315 |  | `NewRectF32` | done |
| `Rect<f32>::union` | method | 323 |  | `(Rect).Union` | dev:D-COND |
| `Line` | struct | 337 |  | `Line` | done |
| `Line::map` | method | 348 |  | `(Line).Map` | done |
| `Line<bool>::TRUE` | assoc-const | 358 |  | `LineBoolTrue` | done |
| `Line<bool>::FALSE` | assoc-const | 360 |  | `LineBoolFalse` | done |
| `Line::sum` | method | 365 |  | `(Line).Sum` | dev:D-COND |
| `Size` | struct | 373 |  | `Size` | done |
| `impl Add<Size<U>> for Size<T>` | trait-impl | 381 |  | `(Size).Add` | dev:D-COND |
| `impl Sub<Size<U>> for Size<T>` | trait-impl | 390 |  | `(Size).Sub` | dev:D-COND |
| `Size::map` | method | 405 |  | `(Size).Map` | done |
| `Size::map_width` | method | 413 |  | `(Size).MapWidth` | done |
| `Size::map_height` | method | 421 |  | `(Size).MapHeight` | done |
| `Size::zip_map` | method | 430 |  | `(Size).ZipMap` | done |
| `Size<f32>::ZERO` | assoc-const | 575 |  | `SizeF32Zero` | done |
| `Size<f32>::f32_max` | method | 579 |  | `(Size).F32Max` | dev:D-COND |
| `Size<f32>::f32_min` | method | 585 |  | `(Size).F32Min` | dev:D-COND |
| `Size<f32>::has_non_zero_area` | method | 591 |  | `(Size).HasNonZeroArea` | dev:D-COND |
| `Size<Option<f32>>::NONE` | assoc-const | 598 |  | `SizeOptionF32None` | done |
| `Size<Option<f32>>::new` | assoc-fn | 602 |  | `NewSizeOptionF32` | done |
| `Size<Option<f32>>::from_cross` | assoc-fn | 608 | flexbox | `SizeOptionF32FromCross` | done |
| `Size<Option<f32>>::maybe_apply_aspect_ratio` | method | 623 |  | `SizeOptionF32MaybeApplyAspectRatio` | done |
| `Size<Option<T>>::unwrap_or` | method | 637 |  | `SizeOptionUnwrapOr` | done |
| `Size<Option<T>>::or` | method | 642 |  | `SizeOptionOr` | done |
| `Size<Option<T>>::both_axis_defined` | method | 648 |  | `SizeOptionBothAxisDefined` | done |
| `Size<Dimension>::from_lengths` | assoc-fn | 656 |  | `SizeDimensionFromLengths` | done |
| `Size<Dimension>::from_percent` | assoc-fn | 662 |  | `SizeDimensionFromPercent` | done |
| `Point` | struct | 672 |  | `Point` | done |
| `Point<f32>::ZERO` | assoc-const | 681 |  | `PointF32Zero` | done |
| `Point<Option<f32>>::NONE` | assoc-const | 686 |  | `PointOptionF32None` | done |
| `impl Add<Point<U>> for Point<T>` | trait-impl | 690 |  | `(Point).Add` | dev:D-COND |
| `Point::map` | method | 702 |  | `(Point).Map` | done |
| `Point::get` | method | 712 | grid | `(Point).Get` | done |
| `Point::transpose` | method | 720 |  | `(Point).Transpose` | done |
| `Point::set` | method | 727 | grid | `(Point).Set` | done |
| `impl From<Point<T>> for Size<T>` | trait-impl | 759 |  | `SizeFromPoint` | done |
| `MinMax` | struct | 768 |  | `MinMax` | done |

## src/lib.rs

Go target: `internal/taffy/lib.go` (package `taffy`), tests in `lib_test.go`

No public items or tests.

## src/prelude.rs

Go target: `internal/taffy/prelude.go` (package `taffy`), tests in `prelude_test.go`

No public items or tests.

## src/style/alignment.rs

Go target: `internal/taffy/style_alignment.go` (package `taffy`), tests in `style_alignment_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AlignItemsKeyword` | enum | 24 |  | `AlignItemsKeyword` | done |
| `AlignContentKeyword` | enum | 69 |  | `AlignContentKeyword` | done |
| `AlignmentSafety` | enum | 135 |  | `AlignmentSafety` | done |
| `AlignItems` | struct | 155 |  | `AlignItems` | done |
| `AlignItems::NORMAL` | assoc-const | 164 |  | `AlignItemsNormal` | done |
| `AlignItems::START` | assoc-const | 166 |  | `AlignItemsStart` | done |
| `AlignItems::END` | assoc-const | 168 |  | `AlignItemsEnd` | done |
| `AlignItems::FLEX_START` | assoc-const | 170 |  | `AlignItemsFlexStart` | done |
| `AlignItems::FLEX_END` | assoc-const | 172 |  | `AlignItemsFlexEnd` | done |
| `AlignItems::SELF_START` | assoc-const | 174 |  | `AlignItemsSelfStart` | done |
| `AlignItems::SELF_END` | assoc-const | 176 |  | `AlignItemsSelfEnd` | done |
| `AlignItems::CENTER` | assoc-const | 178 |  | `AlignItemsCenter` | done |
| `AlignItems::BASELINE` | assoc-const | 180 |  | `AlignItemsBaseline` | done |
| `AlignItems::STRETCH` | assoc-const | 182 |  | `AlignItemsStretch` | done |
| `AlignItems::SAFE_START` | assoc-const | 185 |  | `AlignItemsSafeStart` | done |
| `AlignItems::SAFE_END` | assoc-const | 188 |  | `AlignItemsSafeEnd` | done |
| `AlignItems::SAFE_FLEX_START` | assoc-const | 191 |  | `AlignItemsSafeFlexStart` | done |
| `AlignItems::SAFE_FLEX_END` | assoc-const | 194 |  | `AlignItemsSafeFlexEnd` | done |
| `AlignItems::SAFE_CENTER` | assoc-const | 197 |  | `AlignItemsSafeCenter` | done |
| `AlignItems::SAFE_SELF_START` | assoc-const | 200 |  | `AlignItemsSafeSelfStart` | done |
| `AlignItems::SAFE_SELF_END` | assoc-const | 203 |  | `AlignItemsSafeSelfEnd` | done |
| `AlignItems::is_safe` | method | 207 |  | `(AlignItems).IsSafe` | done |
| `AlignItems::with_safety` | method | 213 |  | `(AlignItems).WithSafety` | done |
| `AlignItems::keyword` | method | 219 |  | `(AlignItems).Keyword` | done |
| `AlignItems::is_stretch_or_normal` | method | 226 |  | `(AlignItems).IsStretchOrNormal` | done |
| `AlignItems::resolve_self_relative` | method | 241 |  | `(AlignItems).ResolveSelfRelative` | done |
| `JustifyItems` | type | 324 |  | `JustifyItems` | done |
| `AlignSelf` | type | 332 |  | `AlignSelf` | done |
| `JustifySelf` | type | 340 |  | `JustifySelf` | done |
| `AlignContent` | struct | 348 |  | `AlignContent` | done |
| `AlignContent::NORMAL` | assoc-const | 357 |  | `AlignContentNormal` | done |
| `AlignContent::START` | assoc-const | 359 |  | `AlignContentStart` | done |
| `AlignContent::END` | assoc-const | 361 |  | `AlignContentEnd` | done |
| `AlignContent::FLEX_START` | assoc-const | 363 |  | `AlignContentFlexStart` | done |
| `AlignContent::FLEX_END` | assoc-const | 365 |  | `AlignContentFlexEnd` | done |
| `AlignContent::CENTER` | assoc-const | 367 |  | `AlignContentCenter` | done |
| `AlignContent::STRETCH` | assoc-const | 369 |  | `AlignContentStretch` | done |
| `AlignContent::SPACE_BETWEEN` | assoc-const | 371 |  | `AlignContentSpaceBetween` | done |
| `AlignContent::SPACE_EVENLY` | assoc-const | 374 |  | `AlignContentSpaceEvenly` | done |
| `AlignContent::SPACE_AROUND` | assoc-const | 376 |  | `AlignContentSpaceAround` | done |
| `AlignContent::SAFE_START` | assoc-const | 379 |  | `AlignContentSafeStart` | done |
| `AlignContent::SAFE_END` | assoc-const | 382 |  | `AlignContentSafeEnd` | done |
| `AlignContent::SAFE_FLEX_START` | assoc-const | 385 |  | `AlignContentSafeFlexStart` | done |
| `AlignContent::SAFE_FLEX_END` | assoc-const | 388 |  | `AlignContentSafeFlexEnd` | done |
| `AlignContent::SAFE_CENTER` | assoc-const | 391 |  | `AlignContentSafeCenter` | done |
| `AlignContent::is_safe` | method | 395 |  | `(AlignContent).IsSafe` | done |
| `AlignContent::with_safety` | method | 401 |  | `(AlignContent).WithSafety` | done |
| `AlignContent::keyword` | method | 407 |  | `(AlignContent).Keyword` | done |
| `JustifyContent` | type | 463 |  | `JustifyContent` | done |
| `AlignContentKeyword::reversed` (`pub(crate)`) | method | 71 |  | `(AlignContentKeyword).reversed` | done |
| `AlignItems::keyword` / `AlignItems::safety` (fields) | field | 157 |  | `(AlignItems).Keyword` | dev:D-TAFFY-ALIGNFIELDS |
| `AlignItems::keyword` / `AlignItems::safety` (fields) | field | 157 |  | `(AlignItems).Safety` | dev:D-TAFFY-ALIGNFIELDS |
| `AlignItems::keyword` / `AlignItems::safety` (fields) | field | 157 |  | `NewAlignItems` | dev:D-TAFFY-ALIGNFIELDS |
| `AlignContent::keyword` / `AlignContent::safety` (fields) | field | 350 |  | `(AlignContent).Keyword` | dev:D-TAFFY-ALIGNFIELDS |
| `AlignContent::keyword` / `AlignContent::safety` (fields) | field | 350 |  | `(AlignContent).Safety` | dev:D-TAFFY-ALIGNFIELDS |
| `AlignContent::keyword` / `AlignContent::safety` (fields) | field | 350 |  | `NewAlignContent` | dev:D-TAFFY-ALIGNFIELDS |
| `impl Default for AlignItemsKeyword` / `AlignContentKeyword` / `AlignItems` / `AlignContent` | trait-impl | 21 |  | `AlignItemsKeywordDefault` | done |
| `impl Default for AlignItemsKeyword` / `AlignContentKeyword` / `AlignItems` / `AlignContent` | trait-impl | 21 |  | `AlignContentKeywordDefault` | done |
| `impl Default for AlignItemsKeyword` / `AlignContentKeyword` / `AlignItems` / `AlignContent` | trait-impl | 21 |  | `AlignItemsDefault` | done |
| `impl Default for AlignItemsKeyword` / `AlignContentKeyword` / `AlignItems` / `AlignContent` | trait-impl | 21 |  | `AlignContentDefault` | done |
| `impl FromCss for AlignItems` / `AlignContent` | trait-impl | 252 | parse | - | n/a:D-TAFFY-FEATURE |
| `impl Serialize` / `Deserialize for AlignItems` / `AlignContent` | trait-impl | 497 | serde | - | n/a:D-TAFFY-FEATURE |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `align_types_within_size_budget` | test | 679 |  | `TestStyleAlignment_AlignTypesWithinSizeBudget` | done |
| `align_items_is_safe` | test | 686 |  | `TestStyleAlignment_AlignItemsIsSafe` | done |
| `align_items_keyword_strips_safe` | test | 707 |  | `TestStyleAlignment_AlignItemsKeywordStripsSafe` | done |
| `align_items_keyword_passthrough` | test | 718 |  | `TestStyleAlignment_AlignItemsKeywordPassthrough` | done |
| `align_items_default_is_normal` | test | 727 |  | `TestStyleAlignment_AlignItemsDefaultIsNormal` | done |
| `resolve_self_relative_inline_axis` | test | 734 |  | `TestStyleAlignment_ResolveSelfRelativeInlineAxis` | done |
| `resolve_self_relative_block_axis` | test | 755 |  | `TestStyleAlignment_ResolveSelfRelativeBlockAxis` | done |
| `align_content_is_safe` | test | 763 |  | `TestStyleAlignment_AlignContentIsSafe` | done |
| `align_content_keyword_strips_safe` | test | 772 |  | `TestStyleAlignment_AlignContentKeywordStripsSafe` | done |
| `align_content_keyword_reversed_swaps_start_end` | test | 782 |  | `TestStyleAlignment_AlignContentKeywordReversedSwapsStartEnd` | done |
| `parse_align_items_plain` | test | 798 | parse | `TestStyleAlignment_ParseAlignItemsPlain` | n/a:D-TAFFY-FEATURE |
| `align_content_default_is_normal` | test | 811 |  | `TestStyleAlignment_AlignContentDefaultIsNormal` | done |
| `parse_align_items_normal` | test | 818 | parse | `TestStyleAlignment_ParseAlignItemsNormal` | n/a:D-TAFFY-FEATURE |
| `parse_align_items_safe` | test | 828 | parse | `TestStyleAlignment_ParseAlignItemsSafe` | n/a:D-TAFFY-FEATURE |
| `parse_align_items_safe_case_insensitive` | test | 840 | parse | `TestStyleAlignment_ParseAlignItemsSafeCaseInsensitive` | n/a:D-TAFFY-FEATURE |
| `parse_align_items_unsafe` | test | 847 | parse | `TestStyleAlignment_ParseAlignItemsUnsafe` | n/a:D-TAFFY-FEATURE |
| `parse_align_items_rejects_invalid_safe_combos` | test | 860 | parse | `TestStyleAlignment_ParseAlignItemsRejectsInvalidSafeCombos` | n/a:D-TAFFY-FEATURE |
| `parse_align_content_plain` | test | 876 | parse | `TestStyleAlignment_ParseAlignContentPlain` | n/a:D-TAFFY-FEATURE |
| `parse_align_content_safe` | test | 889 | parse | `TestStyleAlignment_ParseAlignContentSafe` | n/a:D-TAFFY-FEATURE |
| `parse_align_content_unsafe` | test | 899 | parse | `TestStyleAlignment_ParseAlignContentUnsafe` | n/a:D-TAFFY-FEATURE |
| `parse_align_content_rejects_invalid_safe_combos` | test | 907 | parse | `TestStyleAlignment_ParseAlignContentRejectsInvalidSafeCombos` | n/a:D-TAFFY-FEATURE |
| `serde_align_items_round_trip` | test | 922 |  | `TestStyleAlignment_SerdeAlignItemsRoundTrip` | n/a:D-TAFFY-FEATURE |
| `serde_align_content_round_trip` | test | 953 |  | `TestStyleAlignment_SerdeAlignContentRoundTrip` | n/a:D-TAFFY-FEATURE |

## src/style/available_space.rs

Go target: `internal/taffy/style_available_space.go` (package `taffy`), tests in `style_available_space_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AvailableSpace` | enum | 15 |  | `AvailableSpace` | done |
| `impl TaffyZero for AvailableSpace` | trait-impl | 23 |  | `AvailableSpaceZero` | done |
| `impl TaffyZero for AvailableSpace` (method) | trait-impl | 23 |  | `(AvailableSpace).Zero` | done |
| `impl TaffyMaxContent for AvailableSpace` | trait-impl | 26 |  | `(AvailableSpace).MaxContent` | done |
| `impl TaffyMinContent for AvailableSpace` | trait-impl | 29 |  | `(AvailableSpace).MinContent` | done |
| `impl FromLength for AvailableSpace` | trait-impl | 32 |  | `(AvailableSpace).FromLength` | done |
| `impl FromCss for AvailableSpace` | trait-impl | 39 | parse | - | n/a:D-TAFFY-FEATURE |
| `AvailableSpace::is_definite` | method | 55 |  | `(AvailableSpace).IsDefinite` | done |
| `AvailableSpace::into_option` | method | 61 |  | `(AvailableSpace).IntoOption` | done |
| `AvailableSpace::unwrap_or` | method | 69 |  | `(AvailableSpace).UnwrapOr` | done |
| `AvailableSpace::unwrap` | method | 75 |  | `(AvailableSpace).Unwrap` | done |
| `AvailableSpace::or` | method | 80 |  | `(AvailableSpace).Or` | done |
| `AvailableSpace::or_else` | method | 88 |  | `(AvailableSpace).OrElse` | done |
| `AvailableSpace::unwrap_or_else` | method | 96 |  | `(AvailableSpace).UnwrapOrElse` | done |
| `AvailableSpace::maybe_set` | method | 101 |  | `(AvailableSpace).MaybeSet` | done |
| `AvailableSpace::map_definite_value` | method | 109 |  | `(AvailableSpace).MapDefiniteValue` | done |
| `AvailableSpace::compute_free_space` | method | 117 |  | `(AvailableSpace).ComputeFreeSpace` | done |
| `AvailableSpace::is_roughly_equal` | method | 127 |  | `(AvailableSpace).IsRoughlyEqual` | done |
| `impl From<f32> for AvailableSpace` | trait-impl | 138 |  | `AvailableSpaceFromF32` | done |
| `impl From<Option<f32>> for AvailableSpace` | trait-impl | 144 |  | `AvailableSpaceFromOptionF32` | done |
| `Size<AvailableSpace>::into_options` | method | 155 |  | `SizeAvailableSpaceIntoOptions` | done |
| `Size<AvailableSpace>::maybe_set` | method | 160 |  | `SizeAvailableSpaceMaybeSet` | done |

Upstream has no tests in this file; style_available_space_test.go covers it.

## src/style/block.rs

Go target: `internal/taffy/style_block.go` (package `taffy`), tests in `style_block_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `BlockContainerStyle` | trait | 6 |  | `BlockContainerStyle` | done |
| `BlockContainerStyle::text_align` | trait-method | 9 |  | `BlockContainerStyle.TextAlign` | done |
| `BlockContainerStyle::align_content` | trait-method | 15 |  | `BlockContainerStyle.AlignContent` | done |
| `BlockContainerStyle::justify_items` | trait-method | 23 |  | `BlockContainerStyle.JustifyItems` | done |
| `BlockItemStyle` | trait | 29 |  | `BlockItemStyle` | done |
| `BlockItemStyle::is_table` | trait-method | 32 |  | `BlockItemStyle.IsTable` | done |
| `BlockItemStyle::align_self` | trait-method | 39 |  | `BlockItemStyle.AlignSelf` | done |
| `BlockItemStyle::justify_self` | trait-method | 46 |  | `BlockItemStyle.JustifySelf` | done |
| `BlockItemStyle::align_content` | trait-method | 55 |  | `BlockItemStyle.AlignContent` | done |
| `BlockItemStyle::float` | trait-method | 62 | float_layout | `BlockItemStyle.Float` | done |
| `BlockItemStyle::clear` | trait-method | 69 | float_layout | `BlockItemStyle.Clear` | done |
| `TextAlign` | enum | 77 |  | `TextAlign` | done |
| default methods of `BlockContainerStyle` / `BlockItemStyle` | trait-impl | 6 |  | `BlockContainerStyleDefaults` | dev:D-TAFFY-TRAITS |
| default methods of `BlockContainerStyle` / `BlockItemStyle` | trait-impl | 6 |  | `BlockItemStyleDefaults` | dev:D-TAFFY-TRAITS |
| `impl Default for TextAlign` | trait-impl | 75 |  | `TextAlignDefault` | done |
| `impl FromCss for TextAlign` | trait-impl | 89 | parse | - | n/a:D-TAFFY-FEATURE |

## src/style/compact_length.rs

Go target: `internal/taffy/style_compact_length.go` (package `taffy`), tests in `style_compact_length_test.go`

`CompactLength` is `struct{ tag uint8; bits uint32 }` (D-TAFFY-COMPACT): the f32 payload is kept as its bit pattern, so `==` and `IsZero` compare bitwise as upstream's derived `PartialEq` on the tagged pointer does (`-0.0` differs from `0.0`, equal NaN bits are equal). The `calc` feature is off, so no value has a pointer tag and `calc_resolver` arguments are never called; their type is `func(unsafe.Pointer, float32) float32`. The test helper `from_raw_tag` is `compactLengthFromTag`; `cfg!(feature = "calc")` arms take the false branch. `calc_round_trips_pointer` has no cfg of its own but calls the calc-gated constructors, so upstream builds it only with `calc`.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compat::f32_to_bits` | fn | 17 |  | - | dev:D-TAFFY-COMPACT |
| `compat::f32_from_bits` | fn | 22 |  | - | dev:D-TAFFY-COMPACT |
| `compat::tag_ptr` | fn | 30 | strict_provenance | - | n/a:D-TAFFY-COMPACT |
| `compat::tag_ptr` (not strict_provenance) | fn | 37 |  | - | n/a:D-TAFFY-COMPACT |
| `compat::untag_ptr` | fn | 44 | strict_provenance | - | n/a:D-TAFFY-COMPACT |
| `compat::untag_ptr` (not strict_provenance) | fn | 51 |  | - | n/a:D-TAFFY-COMPACT |
| `CompactLength` | struct | 222 |  | `CompactLength` | done |
| `CompactLength::NON_POINTER_BIT` | assoc-const | 245 |  | `CompactLengthNonPointerBit` | done |
| `CompactLength::KEYWORD_BIT` | assoc-const | 247 |  | `CompactLengthKeywordBit` | done |
| `CompactLength::PERCENTAGE_BIT` | assoc-const | 249 |  | `CompactLengthPercentageBit` | done |
| `CompactLength::SIZING_KEYWORD_BIT` | assoc-const | 251 |  | `CompactLengthSizingKeywordBit` | done |
| `CompactLength::CALC_TAG` | assoc-const | 255 | calc | - | n/a:D-TAFFY-FEATURE |
| `CompactLength::FIT_CONTENT_CALC_TAG` | assoc-const | 258 | calc | - | n/a:D-TAFFY-FEATURE |
| `CompactLength::LENGTH_TAG` | assoc-const | 260 |  | `CompactLengthLengthTag` | done |
| `CompactLength::PERCENT_TAG` | assoc-const | 262 |  | `CompactLengthPercentTag` | done |
| `CompactLength::AUTO_TAG` | assoc-const | 264 |  | `CompactLengthAutoTag` | done |
| `CompactLength::FR_TAG` | assoc-const | 266 |  | `CompactLengthFrTag` | done |
| `CompactLength::CONTENT_TAG` | assoc-const | 268 |  | `CompactLengthContentTag` | done |
| `CompactLength::MIN_CONTENT_TAG` | assoc-const | 270 |  | `CompactLengthMinContentTag` | done |
| `CompactLength::MAX_CONTENT_TAG` | assoc-const | 272 |  | `CompactLengthMaxContentTag` | done |
| `CompactLength::FIT_CONTENT_PX_TAG` | assoc-const | 274 |  | `CompactLengthFitContentPxTag` | done |
| `CompactLength::FIT_CONTENT_KEYWORD_TAG` | assoc-const | 276 |  | `CompactLengthFitContentKeywordTag` | done |
| `CompactLength::STRETCH_TAG` | assoc-const | 278 |  | `CompactLengthStretchTag` | done |
| `CompactLength::FIT_CONTENT_PERCENT_TAG` | assoc-const | 280 |  | `CompactLengthFitContentPercentTag` | done |
| `CompactLength::length` | assoc-fn | 287 |  | `CompactLengthLength` | done |
| `CompactLength::percent` | assoc-fn | 295 |  | `CompactLengthPercent` | done |
| `CompactLength::calc` | assoc-fn | 305 | calc | - | n/a:D-TAFFY-FEATURE |
| `CompactLength::fit_content_calc` | assoc-fn | 317 | calc | - | n/a:D-TAFFY-FEATURE |
| `CompactLength::auto` | assoc-fn | 326 |  | `CompactLengthAuto` | done |
| `CompactLength::fr` | assoc-fn | 334 |  | `CompactLengthFr` | done |
| `CompactLength::min_content` | assoc-fn | 341 |  | `CompactLengthMinContent` | done |
| `CompactLength::max_content` | assoc-fn | 348 |  | `CompactLengthMaxContent` | done |
| `CompactLength::fit_content_px` | assoc-fn | 362 |  | `CompactLengthFitContentPx` | done |
| `CompactLength::fit_content_percent` | assoc-fn | 376 |  | `CompactLengthFitContentPercent` | done |
| `CompactLength::fit_content_keyword` | assoc-fn | 387 |  | `CompactLengthFitContentKeyword` | done |
| `CompactLength::stretch` | assoc-fn | 395 |  | `CompactLengthStretch` | done |
| `CompactLength::content` | assoc-fn | 404 |  | `CompactLengthContent` | done |
| `CompactLength::tag` | method | 410 |  | `(CompactLength).Tag` | done |
| `CompactLength::value` | method | 417 |  | `(CompactLength).Value` | done |
| `CompactLength::calc_value` | method | 424 | calc | - | n/a:D-TAFFY-FEATURE |
| `CompactLength::is_plain_calc` | method | 431 | calc | - | n/a:D-TAFFY-FEATURE |
| `CompactLength::is_fit_content_calc` | method | 437 |  | `(CompactLength).IsFitContentCalc` | done |
| `CompactLength::has_calc` | method | 444 |  | `(CompactLength).HasCalc` | done |
| `CompactLength::is_zero` | method | 450 |  | `(CompactLength).IsZero` | done |
| `CompactLength::is_length_or_percentage` | method | 456 |  | `(CompactLength).IsLengthOrPercentage` | done |
| `CompactLength::is_auto` | method | 462 |  | `(CompactLength).IsAuto` | done |
| `CompactLength::is_content` | method | 468 |  | `(CompactLength).IsContent` | done |
| `CompactLength::is_min_content` | method | 474 |  | `(CompactLength).IsMinContent` | done |
| `CompactLength::is_max_content` | method | 480 |  | `(CompactLength).IsMaxContent` | done |
| `CompactLength::is_fit_content` | method | 486 |  | `(CompactLength).IsFitContent` | done |
| `CompactLength::is_sizing_keyword` | method | 492 |  | `(CompactLength).IsSizingKeyword` | done |
| `CompactLength::is_max_or_fit_content` | method | 500 |  | `(CompactLength).IsMaxOrFitContent` | done |
| `CompactLength::is_max_content_alike` | method | 509 |  | `(CompactLength).IsMaxContentAlike` | done |
| `CompactLength::is_min_or_max_content` | method | 521 |  | `(CompactLength).IsMinOrMaxContent` | done |
| `CompactLength::is_intrinsic` | method | 527 |  | `(CompactLength).IsIntrinsic` | done |
| `CompactLength::is_fr` | method | 540 |  | `(CompactLength).IsFr` | done |
| `CompactLength::uses_percentage` | method | 546 |  | `(CompactLength).UsesPercentage` | done |
| `CompactLength::resolved_percentage_size` | method | 553 |  | `(CompactLength).ResolvedPercentageSize` | done |
| `impl TaffyZero for CompactLength` | trait-impl | 567 |  | `CompactLengthZero` | done |
| `impl TaffyZero for CompactLength` (method) | trait-impl | 567 |  | `(CompactLength).Zero` | done |
| `impl TaffyAuto for CompactLength` | trait-impl | 570 |  | `(CompactLength).Auto` | done |
| `impl TaffyMinContent for CompactLength` | trait-impl | 573 |  | `(CompactLength).MinContent` | done |
| `impl TaffyMaxContent for CompactLength` | trait-impl | 576 |  | `(CompactLength).MaxContent` | done |
| `impl FromLength for CompactLength` | trait-impl | 579 |  | `(CompactLength).FromLength` | done |
| `impl FromPercent for CompactLength` | trait-impl | 584 |  | `(CompactLength).FromPercent` | done |
| `impl FromFr for CompactLength` | trait-impl | 589 |  | `(CompactLength).FromFr` | done |
| `impl TaffyFitContent for CompactLength` | trait-impl | 594 |  | `(CompactLength).FitContent` | done |
| `impl serde::Serialize for CompactLength` | trait-impl | 608 | serde | - | n/a:D-TAFFY-FEATURE |
| `impl serde::Deserialize for CompactLength` | trait-impl | 628 | serde | - | n/a:D-TAFFY-FEATURE |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tags_are_unique_and_fit_in_a_byte` | test | 682 |  | `TestStyleCompactLength_TagsAreUniqueAndFitInAByte` | done |
| `non_pointer_tags_have_non_pointer_bit_set` | test | 695 |  | `TestStyleCompactLength_NonPointerTagsHaveNonPointerBitSet` | done |
| `keyword_bits_match_predicates` | test | 704 |  | `TestStyleCompactLength_KeywordBitsMatchPredicates` | done |
| `percentage_bit_matches_uses_percentage` | test | 732 |  | `TestStyleCompactLength_PercentageBitMatchesUsesPercentage` | done |
| `fit_content_calc_round_trips_pointer` | test | 757 | calc | - | n/a:D-TAFFY-FEATURE |
| `calc_round_trips_pointer` | test | 782 |  | - | n/a:D-TAFFY-FEATURE |

## src/style/dimension.rs

Go target: `internal/taffy/style_dimension.go` (package `taffy`), tests in `style_dimension_test.go`

The `Calc` variants of the `Expanded*` enums and the calc constructors need the `calc` feature (n/a). `from_raw` is `unsafe` upstream because it skips the tag check; the Go functions skip it too.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LengthPercentage` | struct | 13 |  | `LengthPercentage` | done |
| `impl TaffyZero for LengthPercentage` | trait-impl | 14 |  | `LengthPercentageZero` | done |
| `impl TaffyZero for LengthPercentage` (method) | trait-impl | 14 |  | `(LengthPercentage).Zero` | done |
| `impl FromLength for LengthPercentage` | trait-impl | 17 |  | `(LengthPercentage).FromLength` | done |
| `impl FromPercent for LengthPercentage` | trait-impl | 22 |  | `(LengthPercentage).FromPercent` | done |
| `impl FromCss for LengthPercentage` | trait-impl | 29 | parse | - | n/a:D-TAFFY-FEATURE |
| `LengthPercentage::length` | assoc-fn | 45 |  | `LengthPercentageLength` | done |
| `LengthPercentage::percent` | assoc-fn | 53 |  | `LengthPercentagePercent` | done |
| `LengthPercentage::calc` | assoc-fn | 63 | calc | - | n/a:D-TAFFY-FEATURE |
| `LengthPercentage::from_raw` | assoc-fn | 71 |  | `LengthPercentageFromRaw` | done |
| `LengthPercentage::into_raw` | method | 76 |  | `(LengthPercentage).IntoRaw` | done |
| `LengthPercentage::expand` | method | 85 |  | `(LengthPercentage).Expand` | done |
| `ExpandedLengthPercentage` | enum | 101 |  | `ExpandedLengthPercentage` | done |
| `impl From<LengthPercentage> for ExpandedLengthPercentage` | trait-impl | 112 |  | `ExpandedLengthPercentageFromLengthPercentage` | done |
| `impl From<ExpandedLengthPercentage> for LengthPercentage` | trait-impl | 118 |  | `LengthPercentageFromExpandedLengthPercentage` | done |
| `impl serde::Deserialize for LengthPercentage` | trait-impl | 130 | serde | - | n/a:D-TAFFY-FEATURE |
| `LengthPercentageAuto` | struct | 150 |  | `LengthPercentageAuto` | done |
| `impl TaffyZero for LengthPercentageAuto` | trait-impl | 151 |  | `LengthPercentageAutoZero` | done |
| `impl TaffyZero for LengthPercentageAuto` (method) | trait-impl | 151 |  | `(LengthPercentageAuto).Zero` | done |
| `impl TaffyAuto for LengthPercentageAuto` | trait-impl | 154 |  | `(LengthPercentageAuto).Auto` | done |
| `impl FromLength for LengthPercentageAuto` | trait-impl | 157 |  | `(LengthPercentageAuto).FromLength` | done |
| `impl FromPercent for LengthPercentageAuto` | trait-impl | 162 |  | `(LengthPercentageAuto).FromPercent` | done |
| `impl From<LengthPercentage> for LengthPercentageAuto` | trait-impl | 167 |  | `LengthPercentageAutoFromLengthPercentage` | done |
| `impl FromCss for LengthPercentageAuto` | trait-impl | 174 | parse | - | n/a:D-TAFFY-FEATURE |
| `LengthPercentageAuto::length` | assoc-fn | 191 |  | `LengthPercentageAutoLength` | done |
| `LengthPercentageAuto::percent` | assoc-fn | 199 |  | `LengthPercentageAutoPercent` | done |
| `LengthPercentageAuto::auto` | assoc-fn | 206 |  | `LengthPercentageAutoAuto` | done |
| `LengthPercentageAuto::calc` | assoc-fn | 216 | calc | - | n/a:D-TAFFY-FEATURE |
| `LengthPercentageAuto::from_raw` | assoc-fn | 224 |  | `LengthPercentageAutoFromRaw` | done |
| `LengthPercentageAuto::into_raw` | method | 229 |  | `(LengthPercentageAuto).IntoRaw` | done |
| `LengthPercentageAuto::resolve_to_option` | method | 238 |  | `(LengthPercentageAuto).ResolveToOption` | done |
| `LengthPercentageAuto::is_auto` | method | 251 |  | `(LengthPercentageAuto).IsAuto` | done |
| `LengthPercentageAuto::expand` | method | 260 |  | `(LengthPercentageAuto).Expand` | done |
| `ExpandedLengthPercentageAuto` | enum | 277 |  | `ExpandedLengthPercentageAuto` | done |
| `impl From<LengthPercentageAuto> for ExpandedLengthPercentageAuto` | trait-impl | 290 |  | `ExpandedLengthPercentageAutoFromLengthPercentageAuto` | done |
| `impl From<ExpandedLengthPercentageAuto> for LengthPercentageAuto` | trait-impl | 296 |  | `LengthPercentageAutoFromExpandedLengthPercentageAuto` | done |
| `impl serde::Deserialize for LengthPercentageAuto` | trait-impl | 309 | serde | - | n/a:D-TAFFY-FEATURE |
| `Dimension` | struct | 329 |  | `Dimension` | done |
| `impl TaffyZero for Dimension` | trait-impl | 330 |  | `DimensionZero` | done |
| `impl TaffyZero for Dimension` (method) | trait-impl | 330 |  | `(Dimension).Zero` | done |
| `impl TaffyAuto for Dimension` | trait-impl | 333 |  | `(Dimension).Auto` | done |
| `impl FromLength for Dimension` | trait-impl | 336 |  | `(Dimension).FromLength` | done |
| `impl FromPercent for Dimension` | trait-impl | 341 |  | `(Dimension).FromPercent` | done |
| `impl From<LengthPercentage> for Dimension` | trait-impl | 346 |  | `DimensionFromLengthPercentage` | done |
| `impl From<LengthPercentageAuto> for Dimension` | trait-impl | 351 |  | `DimensionFromLengthPercentageAuto` | done |
| `impl FromCss for Dimension` | trait-impl | 358 | parse | - | n/a:D-TAFFY-FEATURE |
| `Dimension::length` | assoc-fn | 392 |  | `DimensionLength` | done |
| `Dimension::percent` | assoc-fn | 400 |  | `DimensionPercent` | done |
| `Dimension::auto` | assoc-fn | 407 |  | `DimensionAuto` | done |
| `Dimension::min_content` | assoc-fn | 414 |  | `DimensionMinContent` | done |
| `Dimension::max_content` | assoc-fn | 421 |  | `DimensionMaxContent` | done |
| `Dimension::fit_content` | assoc-fn | 429 |  | `DimensionFitContent` | done |
| `Dimension::fit_content_px` | assoc-fn | 437 |  | `DimensionFitContentPx` | done |
| `Dimension::stretch` | assoc-fn | 445 |  | `DimensionStretch` | done |
| `Dimension::content` | assoc-fn | 454 |  | `DimensionContent` | done |
| `Dimension::fit_content_percent` | assoc-fn | 462 |  | `DimensionFitContentPercent` | done |
| `Dimension::calc` | assoc-fn | 472 | calc | - | n/a:D-TAFFY-FEATURE |
| `Dimension::fit_content_calc` | assoc-fn | 482 | calc | - | n/a:D-TAFFY-FEATURE |
| `Dimension::is_fit_content_calc` | method | 488 |  | `(Dimension).IsFitContentCalc` | done |
| `Dimension::calc_value` | method | 495 | calc | - | n/a:D-TAFFY-FEATURE |
| `Dimension::from_raw` | assoc-fn | 503 |  | `DimensionFromRaw` | done |
| `Dimension::into_raw` | method | 508 |  | `(Dimension).IntoRaw` | done |
| `Dimension::into_option` | method | 514 | grid | `(Dimension).IntoOption` | done |
| `Dimension::is_auto` | method | 522 |  | `(Dimension).IsAuto` | done |
| `Dimension::is_sizing_keyword` | method | 528 |  | `(Dimension).IsSizingKeyword` | done |
| `Dimension::is_stretch` | method | 534 |  | `(Dimension).IsStretch` | done |
| `Dimension::is_content` | method | 540 |  | `(Dimension).IsContent` | done |
| `Dimension::tag` | method | 545 |  | `(Dimension).Tag` | done |
| `Dimension::value` | method | 550 |  | `(Dimension).Value` | done |
| `Dimension::expand` | method | 559 |  | `(Dimension).Expand` | done |
| `ExpandedDimension` | enum | 585 |  | `ExpandedDimension` | done |
| `impl From<Dimension> for ExpandedDimension` | trait-impl | 615 |  | `ExpandedDimensionFromDimension` | done |
| `impl From<ExpandedDimension> for Dimension` | trait-impl | 621 |  | `DimensionFromExpandedDimension` | done |
| `impl serde::Deserialize for Dimension` | trait-impl | 643 | serde | - | n/a:D-TAFFY-FEATURE |
| `Rect<Dimension>::from_length` | assoc-fn | 673 |  | `RectDimensionFromLength` | done |
| `Rect<Dimension>::from_percent` | assoc-fn | 684 |  | `RectDimensionFromPercent` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `expand_tests::length_percentage_round_trips` | test | 710 |  | `TestStyleDimension_ExpandTests_LengthPercentageRoundTrips` | done |
| `expand_tests::length_percentage_auto_round_trips` | test | 721 |  | `TestStyleDimension_ExpandTests_LengthPercentageAutoRoundTrips` | done |
| `expand_tests::dimension_round_trips_all_keywords` | test | 731 |  | `TestStyleDimension_ExpandTests_DimensionRoundTripsAllKeywords` | done |
| `expand_tests::calc_round_trips` | test | 753 | calc | - | n/a:D-TAFFY-FEATURE |

## src/style/flex.rs

Go target: `internal/taffy/style_flex.go` (package `taffy`), tests in `style_flex_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FlexboxContainerStyle` | trait | 6 |  | `FlexboxContainerStyle` | done |
| `FlexboxContainerStyle::flex_direction` | trait-method | 9 |  | `FlexboxContainerStyle.FlexDirection` | done |
| `FlexboxContainerStyle::flex_wrap` | trait-method | 14 |  | `FlexboxContainerStyle.FlexWrap` | done |
| `FlexboxContainerStyle::flex_line_count` | trait-method | 23 | flexbox_balance | `FlexboxContainerStyle.FlexLineCount` | n/a:D-TAFFY-FEATURE |
| `FlexboxContainerStyle::gap` | trait-method | 29 |  | `FlexboxContainerStyle.Gap` | done |
| `FlexboxContainerStyle::align_content` | trait-method | 37 |  | `FlexboxContainerStyle.AlignContent` | done |
| `FlexboxContainerStyle::align_items` | trait-method | 42 |  | `FlexboxContainerStyle.AlignItems` | done |
| `FlexboxContainerStyle::justify_content` | trait-method | 47 |  | `FlexboxContainerStyle.JustifyContent` | done |
| `FlexboxItemStyle` | trait | 53 |  | `FlexboxItemStyle` | done |
| `FlexboxItemStyle::flex_basis` | trait-method | 56 |  | `FlexboxItemStyle.FlexBasis` | done |
| `FlexboxItemStyle::flex_grow` | trait-method | 61 |  | `FlexboxItemStyle.FlexGrow` | done |
| `FlexboxItemStyle::flex_shrink` | trait-method | 66 |  | `FlexboxItemStyle.FlexShrink` | done |
| `FlexboxItemStyle::align_self` | trait-method | 73 |  | `FlexboxItemStyle.AlignSelf` | done |
| `FlexWrap` | enum | 87 |  | `FlexWrap` | done |
| `FlexDirection` | enum | 198 |  | `FlexDirection` | done |
| default methods of `FlexboxContainerStyle` / `FlexboxItemStyle` | trait-impl | 6 |  | `FlexboxContainerStyleDefaults` | dev:D-TAFFY-TRAITS |
| default methods of `FlexboxContainerStyle` / `FlexboxItemStyle` | trait-impl | 6 |  | `FlexboxItemStyleDefaults` | dev:D-TAFFY-TRAITS |
| `FlexWrap::is_multi_line` (`pub(crate)`) | method | 107 |  | `(FlexWrap).isMultiLine` | done |
| `FlexWrap::is_reverse` (`pub(crate)`) | method | 113 |  | `(FlexWrap).isReverse` | done |
| `FlexWrap::is_balance` (`pub(crate)`) | method | 124 | flexbox_balance | - | n/a:D-TAFFY-FEATURE |
| `FlexDirection::is_row` / `is_column` / `is_reverse` / `main_axis` / `cross_axis` (`pub(crate)`) | method | 228 |  | `(FlexDirection).isRow` | done |
| `FlexDirection::is_row` / `is_column` / `is_reverse` / `main_axis` / `cross_axis` (`pub(crate)`) | method | 228 |  | `(FlexDirection).isColumn` | done |
| `FlexDirection::is_row` / `is_column` / `is_reverse` / `main_axis` / `cross_axis` (`pub(crate)`) | method | 228 |  | `(FlexDirection).isReverse` | done |
| `FlexDirection::is_row` / `is_column` / `is_reverse` / `main_axis` / `cross_axis` (`pub(crate)`) | method | 228 |  | `(FlexDirection).mainAxis` | done |
| `FlexDirection::is_row` / `is_column` / `is_reverse` / `main_axis` / `cross_axis` (`pub(crate)`) | method | 228 |  | `(FlexDirection).crossAxis` | done |
| `impl Default for FlexWrap` / `FlexDirection` | trait-impl | 85 |  | `FlexWrapDefault` | done |
| `impl Default for FlexWrap` / `FlexDirection` | trait-impl | 85 |  | `FlexDirectionDefault` | done |
| `impl FromCss for FlexWrap` / `FlexDirection` | trait-impl | 132 | parse | - | n/a:D-TAFFY-FEATURE |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_flex_direction::flex_direction_is_row` | test | 270 |  | `TestStyleFlex_FlexDirection_FlexDirectionIsRow` | done |
| `test_flex_direction::flex_direction_is_column` | test | 278 |  | `TestStyleFlex_FlexDirection_FlexDirectionIsColumn` | done |
| `test_flex_direction::flex_direction_is_reverse` | test | 286 |  | `TestStyleFlex_FlexDirection_FlexDirectionIsReverse` | done |

## src/style/float.rs

Go target: `internal/taffy/style_float.go` (package `taffy`), tests in `style_float_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Float` | enum | 9 |  | `Float` | done |
| `FloatDirection` | enum | 33 |  | `FloatDirection` | done |
| `Float::is_floated` | method | 42 |  | `(Float).IsFloated` | done |
| `Float::float_direction` | method | 47 |  | `(Float).FloatDirection` | done |
| `Clear` | enum | 62 |  | `Clear` | done |
| `impl Default for Float` / `Clear` | trait-impl | 7 |  | `FloatDefault` | done |
| `impl Default for Float` / `Clear` | trait-impl | 7 |  | `ClearDefault` | done |
| `impl FromCss for Float` / `Clear` | trait-impl | 20 | parse | - | n/a:D-TAFFY-FEATURE |

## src/style/grid.rs

Go target: `internal/taffy/style_grid.go` (package `taffy`), tests in `style_grid_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `GridTemplateAreas` | struct | 25 |  | `GridTemplateAreas` | done |
| `GridTemplateArea` | struct | 37 |  | `GridTemplateArea` | done |
| `NamedGridLine` | struct | 54 |  | `NamedGridLine` | done |
| `GenericRepetition` | trait | 81 |  | `GenericRepetition` | dev:D-TAFFY-GRIDLISTS |
| `GenericRepetition::count` | trait-method | 94 |  | `GenericRepetition.Count` | done |
| `GenericRepetition::tracks` | trait-method | 96 |  | `GenericRepetition.Tracks` | done |
| `GenericRepetition::track_count` | trait-method | 98 |  | `GenericRepetitionDefaultTrackCount` | done |
| `GenericRepetition::lines_names` | trait-method | 106 |  | `GenericRepetition.LinesNames` | dev:D-TAFFY-GRIDLISTS |
| `TemplateLineNames` | trait | 112 |  | `TemplateLineNames` | dev:D-TAFFY-GRIDLISTS |
| `GenericGridTemplateComponent` | enum | 130 |  | `GenericGridTemplateComponent` | done |
| `GenericGridTemplateComponent::is_auto_repetition` | method | 147 |  | `(GenericGridTemplateComponent).IsAutoRepetition` | done |
| `GridContainerStyle` | trait | 156 |  | `GridContainerStyle` | dev:D-TAFFY-GRIDLISTS |
| `GridContainerStyle::grid_template_rows` | trait-method | 189 |  | `GridContainerStyle.GridTemplateRows` | dev:D-TAFFY-GRIDLISTS |
| `GridContainerStyle::grid_template_columns` | trait-method | 191 |  | `GridContainerStyle.GridTemplateColumns` | dev:D-TAFFY-GRIDLISTS |
| `GridContainerStyle::grid_auto_rows` | trait-method | 193 |  | `GridContainerStyle.GridAutoRows` | dev:D-TAFFY-GRIDLISTS |
| `GridContainerStyle::grid_auto_columns` | trait-method | 195 |  | `GridContainerStyle.GridAutoColumns` | dev:D-TAFFY-GRIDLISTS |
| `GridContainerStyle::grid_template_areas` | trait-method | 198 |  | `GridContainerStyle.GridTemplateAreas` | dev:D-TAFFY-GRIDLISTS |
| `GridContainerStyle::grid_template_area_row_count` | trait-method | 201 |  | `GridContainerStyleDefaultGridTemplateAreaRowCount` | done |
| `GridContainerStyle::grid_template_area_column_count` | trait-method | 208 |  | `GridContainerStyleDefaultGridTemplateAreaColumnCount` | done |
| `GridContainerStyle::grid_template_column_names` | trait-method | 214 |  | `GridContainerStyle.GridTemplateColumnNames` | dev:D-TAFFY-GRIDLISTS |
| `GridContainerStyle::grid_template_row_names` | trait-method | 216 |  | `GridContainerStyle.GridTemplateRowNames` | dev:D-TAFFY-GRIDLISTS |
| `GridContainerStyle::grid_auto_flow` | trait-method | 220 |  | `GridContainerStyle.GridAutoFlow` | done |
| `GridContainerStyle::gap` | trait-method | 226 |  | `GridContainerStyle.Gap` | done |
| `GridContainerStyle::align_content` | trait-method | 234 |  | `GridContainerStyle.AlignContent` | done |
| `GridContainerStyle::justify_content` | trait-method | 239 |  | `GridContainerStyle.JustifyContent` | done |
| `GridContainerStyle::align_items` | trait-method | 244 |  | `GridContainerStyle.AlignItems` | done |
| `GridContainerStyle::justify_items` | trait-method | 249 |  | `GridContainerStyle.JustifyItems` | done |
| `GridContainerStyle::grid_template_tracks` | trait-method | 255 |  | `GridContainerStyleDefaultGridTemplateTracks` | done |
| `GridContainerStyle::grid_align_content` | trait-method | 265 |  | `GridContainerStyleDefaultGridAlignContent` | done |
| `GridItemStyle` | trait | 278 |  | `GridItemStyle` | done |
| `GridItemStyle::grid_row` | trait-method | 281 |  | `GridItemStyle.GridRow` | done |
| `GridItemStyle::grid_column` | trait-method | 286 |  | `GridItemStyle.GridColumn` | done |
| `GridItemStyle::align_self` | trait-method | 293 |  | `GridItemStyle.AlignSelf` | done |
| `GridItemStyle::justify_self` | trait-method | 299 |  | `GridItemStyle.JustifySelf` | done |
| `GridItemStyle::grid_placement` | trait-method | 305 |  | `GridItemStyleDefaultGridPlacement` | done |
| `GridAutoFlow` | enum | 322 |  | `GridAutoFlow` | done |
| `GridAutoFlow::is_dense` | method | 378 |  | `(GridAutoFlow).IsDense` | done |
| `GridAutoFlow::primary_axis` | method | 387 |  | `(GridAutoFlow).PrimaryAxis` | done |
| `GenericGridPlacement` | enum | 402 |  | `GenericGridPlacement` | done |
| `GridPlacement` | enum | 426 |  | `GridPlacement` | done |
| `GridPlacement::into_origin_zero_placement_ignoring_named` | method | 542 |  | `(GridPlacement).IntoOriginZeroPlacementIgnoringNamed` | done |
| `Line::into_origin_zero_ignoring_named` | method | 562 |  | `LineGridPlacementIntoOriginZeroIgnoringNamed` | done |
| `Line::indefinite_span` | method | 595 |  | `LineGenericGridPlacementIndefiniteSpan` | done |
| `Line::is_definite` | method | 616 |  | `LineGridPlacementIsDefinite` | done |
| `Line<NonNamedGridPlacement>::is_definite` | method | 632 |  | `lineNonNamedGridPlacementIsDefinite` | done |
| `Line<NonNamedGridPlacement>::into_origin_zero` | method | 641 |  | `lineNonNamedGridPlacementIntoOriginZero` | done |
| `Line<OriginZeroGridPlacement>::is_definite` | method | 653 |  | `LineOriginZeroGridPlacementIsDefinite` | done |
| `Line<OriginZeroGridPlacement>::resolve_definite_grid_lines` | method | 659 |  | `LineOriginZeroGridPlacementResolveDefiniteGridLines` | done |
| `Line<OriginZeroGridPlacement>::resolve_absolutely_positioned_grid_tracks` | method | 686 |  | `LineOriginZeroGridPlacementResolveAbsolutelyPositionedGridTracks` | done |
| `Line<OriginZeroGridPlacement>::resolve_indefinite_grid_tracks` | method | 706 |  | `LineOriginZeroGridPlacementResolveIndefiniteGridTracks` | done |
| `MaxTrackSizingFunction` | struct | 732 |  | `MaxTrackSizingFunction` | done |
| `MaxTrackSizingFunction::length` | assoc-fn | 853 |  | `MaxTrackSizingFunctionLength` | done |
| `MaxTrackSizingFunction::percent` | assoc-fn | 861 |  | `MaxTrackSizingFunctionPercent` | done |
| `MaxTrackSizingFunction::auto` | assoc-fn | 868 |  | `MaxTrackSizingFunctionAuto` | done |
| `MaxTrackSizingFunction::min_content` | assoc-fn | 875 |  | `MaxTrackSizingFunctionMinContent` | done |
| `MaxTrackSizingFunction::max_content` | assoc-fn | 882 |  | `MaxTrackSizingFunctionMaxContent` | done |
| `MaxTrackSizingFunction::fit_content_px` | assoc-fn | 896 |  | `MaxTrackSizingFunctionFitContentPx` | done |
| `MaxTrackSizingFunction::fit_content_percent` | assoc-fn | 910 |  | `MaxTrackSizingFunctionFitContentPercent` | done |
| `MaxTrackSizingFunction::fr` | assoc-fn | 918 |  | `MaxTrackSizingFunctionFr` | done |
| `MaxTrackSizingFunction::calc` | assoc-fn | 928 | calc | `MaxTrackSizingFunctionCalc` | n/a:D-TAFFY-FEATURE |
| `MaxTrackSizingFunction::fit_content_calc` | assoc-fn | 938 | calc | `MaxTrackSizingFunctionFitContentCalc` | n/a:D-TAFFY-FEATURE |
| `MaxTrackSizingFunction::from_raw` | assoc-fn | 946 |  | `MaxTrackSizingFunctionFromRaw` | done |
| `MaxTrackSizingFunction::into_raw` | method | 951 |  | `(MaxTrackSizingFunction).IntoRaw` | done |
| `MaxTrackSizingFunction::is_intrinsic` | method | 957 |  | `(MaxTrackSizingFunction).IsIntrinsic` | done |
| `MaxTrackSizingFunction::is_max_content_alike` | method | 965 |  | `(MaxTrackSizingFunction).IsMaxContentAlike` | done |
| `MaxTrackSizingFunction::is_fr` | method | 971 |  | `(MaxTrackSizingFunction).IsFr` | done |
| `MaxTrackSizingFunction::is_auto` | method | 977 |  | `(MaxTrackSizingFunction).IsAuto` | done |
| `MaxTrackSizingFunction::is_min_content` | method | 983 |  | `(MaxTrackSizingFunction).IsMinContent` | done |
| `MaxTrackSizingFunction::is_max_content` | method | 989 |  | `(MaxTrackSizingFunction).IsMaxContent` | done |
| `MaxTrackSizingFunction::is_fit_content` | method | 995 |  | `(MaxTrackSizingFunction).IsFitContent` | done |
| `MaxTrackSizingFunction::is_max_or_fit_content` | method | 1001 |  | `(MaxTrackSizingFunction).IsMaxOrFitContent` | done |
| `MaxTrackSizingFunction::has_definite_value` | method | 1007 |  | `(MaxTrackSizingFunction).HasDefiniteValue` | done |
| `MaxTrackSizingFunction::definite_value` | method | 1021 |  | `(MaxTrackSizingFunction).DefiniteValue` | done |
| `MaxTrackSizingFunction::fit_content_limit` | method | 1039 |  | `(MaxTrackSizingFunction).FitContentLimit` | done |
| `MaxTrackSizingFunction::definite_limit` | method | 1062 |  | `(MaxTrackSizingFunction).DefiniteLimit` | done |
| `MaxTrackSizingFunction::resolved_percentage_size` | method | 1079 |  | `(MaxTrackSizingFunction).ResolvedPercentageSize` | done |
| `MaxTrackSizingFunction::uses_percentage` | method | 1089 |  | `(MaxTrackSizingFunction).UsesPercentage` | done |
| `MaxTrackSizingFunction::expand` | method | 1098 |  | `(MaxTrackSizingFunction).Expand` | done |
| `ExpandedMaxTrackSizingFunction` | enum | 1122 |  | `ExpandedMaxTrackSizingFunction` | done |
| `MinTrackSizingFunction` | struct | 1181 |  | `MinTrackSizingFunction` | done |
| `MinTrackSizingFunction::length` | assoc-fn | 1289 |  | `MinTrackSizingFunctionLength` | done |
| `MinTrackSizingFunction::percent` | assoc-fn | 1297 |  | `MinTrackSizingFunctionPercent` | done |
| `MinTrackSizingFunction::auto` | assoc-fn | 1304 |  | `MinTrackSizingFunctionAuto` | done |
| `MinTrackSizingFunction::min_content` | assoc-fn | 1311 |  | `MinTrackSizingFunctionMinContent` | done |
| `MinTrackSizingFunction::max_content` | assoc-fn | 1318 |  | `MinTrackSizingFunctionMaxContent` | done |
| `MinTrackSizingFunction::calc` | assoc-fn | 1328 | calc | `MinTrackSizingFunctionCalc` | n/a:D-TAFFY-FEATURE |
| `MinTrackSizingFunction::from_raw` | assoc-fn | 1336 |  | `MinTrackSizingFunctionFromRaw` | done |
| `MinTrackSizingFunction::into_raw` | method | 1341 |  | `(MinTrackSizingFunction).IntoRaw` | done |
| `MinTrackSizingFunction::is_intrinsic` | method | 1347 |  | `(MinTrackSizingFunction).IsIntrinsic` | done |
| `MinTrackSizingFunction::is_min_or_max_content` | method | 1353 |  | `(MinTrackSizingFunction).IsMinOrMaxContent` | done |
| `MinTrackSizingFunction::is_fr` | method | 1359 |  | `(MinTrackSizingFunction).IsFr` | done |
| `MinTrackSizingFunction::is_auto` | method | 1365 |  | `(MinTrackSizingFunction).IsAuto` | done |
| `MinTrackSizingFunction::behaves_as_auto` | method | 1371 |  | `(MinTrackSizingFunction).BehavesAsAuto` | done |
| `MinTrackSizingFunction::is_min_content` | method | 1377 |  | `(MinTrackSizingFunction).IsMinContent` | done |
| `MinTrackSizingFunction::is_max_content` | method | 1383 |  | `(MinTrackSizingFunction).IsMaxContent` | done |
| `MinTrackSizingFunction::definite_value` | method | 1391 |  | `(MinTrackSizingFunction).DefiniteValue` | done |
| `MinTrackSizingFunction::resolved_percentage_size` | method | 1408 |  | `(MinTrackSizingFunction).ResolvedPercentageSize` | done |
| `MinTrackSizingFunction::uses_percentage` | method | 1418 |  | `(MinTrackSizingFunction).UsesPercentage` | done |
| `MinTrackSizingFunction::expand` | method | 1427 |  | `(MinTrackSizingFunction).Expand` | done |
| `ExpandedMinTrackSizingFunction` | enum | 1446 |  | `ExpandedMinTrackSizingFunction` | done |
| `TrackSizingFunction` | type | 1487 |  | `TrackSizingFunction` | done |
| `TrackSizingFunction::min_sizing_function` | method | 1490 |  | `TrackSizingFunctionMinSizingFunction` | done |
| `TrackSizingFunction::max_sizing_function` | method | 1494 |  | `TrackSizingFunctionMaxSizingFunction` | done |
| `TrackSizingFunction::has_fixed_component` | method | 1498 |  | `TrackSizingFunctionHasFixedComponent` | done |
| `RepetitionCount` | enum | 1585 |  | `RepetitionCount` | done |
| `InvalidStringRepetitionValue` | struct | 1604 |  | `InvalidStringRepetitionValue` | done |
| `GridTemplateRepetition` | struct | 1642 |  | `GridTemplateRepetition` | done |
| `GridTemplateComponent` | enum | 1684 |  | `GridTemplateComponent` | done |
| `GridTemplateComponent::as_component_ref` | method | 1694 |  | `(GridTemplateComponent).AsComponentRef` | done |
| `GridTemplateComponent::is_auto_repetition` | method | 1704 |  | `(GridTemplateComponent).IsAutoRepetition` | done |
| `GridTemplateTracks` | struct | 1782 |  | `GridTemplateTracks` | done |
| `GridAutoTracks` | struct | 1838 |  | `GridAutoTracks` | done |
| `GridAreaAxis` (`pub(crate)`) | enum | 64 |  | `gridAreaAxis` | done |
| `GridAreaEnd` (`pub(crate)`) | enum | 73 |  | `gridAreaEnd` | done |
| `OriginZeroGridPlacement` (`pub(crate)`) | type | 419 |  | `originZeroGridPlacement` | done |
| `NonNamedGridPlacement` (`pub(crate)`) | type | 424 |  | `nonNamedGridPlacement` | done |
| `NonNamedGridPlacement::into_origin_zero_placement` | method | 573 |  | `nonNamedGridPlacementIntoOriginZeroPlacement` | done |
| default methods of `GridContainerStyle` / `GridItemStyle` | trait-impl | 156 |  | `GridContainerStyleDefaults` | dev:D-TAFFY-TRAITS |
| default methods of `GridContainerStyle` / `GridItemStyle` | trait-impl | 278 |  | `GridItemStyleDefaults` | dev:D-TAFFY-TRAITS |
| `impl TemplateLineNames for Map<slice::Iter<Vec<S>>, ...>` | trait-impl | 119 |  | `TemplateLineNames` (alias of `[][]S`) | dev:D-TAFFY-GRIDLISTS |
| `impl Default for GridAutoFlow` | trait-impl | 321 |  | `GridAutoFlowDefault` | done |
| `impl FromCss` / `FromStr` for the grid types, `saturating_i16` / `saturating_u16` | trait-impl | 342 | parse | - | n/a:D-TAFFY-FEATURE |
| `impl Default for GridPlacement` | trait-impl | 420 |  | `GridPlacementDefault` | done |
| `impl TaffyAuto for GridPlacement` | trait-impl | 438 |  | `(GridPlacement).Auto` | done |
| `impl TaffyGridLine for GridPlacement` | trait-impl | 441 |  | `(GridPlacement).FromLineIndex` | done |
| `impl TaffyGridLine for Line<GridPlacement>` | trait-impl | 446 |  | `(Line).FromLineIndex` | dev:D-COND |
| `impl TaffyGridSpan for GridPlacement` | trait-impl | 451 |  | `(GridPlacement).FromSpan` | done |
| `impl TaffyGridSpan for Line<GridPlacement>` | trait-impl | 456 |  | `(Line).FromSpan` | dev:D-COND |
| `impl Default for Line<GridPlacement>` | trait-impl | 723 |  | `LineGridPlacementDefault` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `MaxTrackSizingFunctionZero` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `(MaxTrackSizingFunction).Zero` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `(MaxTrackSizingFunction).Auto` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `(MaxTrackSizingFunction).MinContent` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `(MaxTrackSizingFunction).MaxContent` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `(MaxTrackSizingFunction).FromLength` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `(MaxTrackSizingFunction).FromPercent` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `(MaxTrackSizingFunction).FitContent` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent` / `TaffyFitContent` / `FromFr for MaxTrackSizingFunction` | trait-impl | 734 |  | `(MaxTrackSizingFunction).FromFr` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension / MinTrackSizingFunction> for MaxTrackSizingFunction` | trait-impl | 770 |  | `MaxTrackSizingFunctionFromLengthPercentage` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension / MinTrackSizingFunction> for MaxTrackSizingFunction` | trait-impl | 770 |  | `MaxTrackSizingFunctionFromLengthPercentageAuto` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension / MinTrackSizingFunction> for MaxTrackSizingFunction` | trait-impl | 770 |  | `MaxTrackSizingFunctionFromDimension` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension / MinTrackSizingFunction> for MaxTrackSizingFunction` | trait-impl | 770 |  | `MaxTrackSizingFunctionFromMinTrackSizingFunction` | done |
| `impl Deserialize for MaxTrackSizingFunction` / `MinTrackSizingFunction` | trait-impl | 829 | serde | - | n/a:D-TAFFY-FEATURE |
| `impl From<MaxTrackSizingFunction> for ExpandedMaxTrackSizingFunction` and back | trait-impl | 1146 |  | `ExpandedMaxTrackSizingFunctionFromMaxTrackSizingFunction` | done |
| `impl From<MaxTrackSizingFunction> for ExpandedMaxTrackSizingFunction` and back | trait-impl | 1146 |  | `MaxTrackSizingFunctionFromExpandedMaxTrackSizingFunction` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent for MinTrackSizingFunction` | trait-impl | 1183 |  | `MinTrackSizingFunctionZero` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent for MinTrackSizingFunction` | trait-impl | 1183 |  | `(MinTrackSizingFunction).Zero` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent for MinTrackSizingFunction` | trait-impl | 1183 |  | `(MinTrackSizingFunction).Auto` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent for MinTrackSizingFunction` | trait-impl | 1183 |  | `(MinTrackSizingFunction).MinContent` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent for MinTrackSizingFunction` | trait-impl | 1183 |  | `(MinTrackSizingFunction).MaxContent` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent for MinTrackSizingFunction` | trait-impl | 1183 |  | `(MinTrackSizingFunction).FromLength` | done |
| `impl TaffyZero` / `TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `FromLength` / `FromPercent for MinTrackSizingFunction` | trait-impl | 1183 |  | `(MinTrackSizingFunction).FromPercent` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension / MaxTrackSizingFunction> for MinTrackSizingFunction` | trait-impl | 1209 |  | `MinTrackSizingFunctionFromLengthPercentage` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension / MaxTrackSizingFunction> for MinTrackSizingFunction` | trait-impl | 1209 |  | `MinTrackSizingFunctionFromLengthPercentageAuto` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension / MaxTrackSizingFunction> for MinTrackSizingFunction` | trait-impl | 1209 |  | `MinTrackSizingFunctionFromDimension` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension / MaxTrackSizingFunction> for MinTrackSizingFunction` | trait-impl | 1209 |  | `MinTrackSizingFunctionFromMaxTrackSizingFunction` | done |
| `impl From<MinTrackSizingFunction> for ExpandedMinTrackSizingFunction` and back | trait-impl | 1464 |  | `ExpandedMinTrackSizingFunctionFromMinTrackSizingFunction` | done |
| `impl From<MinTrackSizingFunction> for ExpandedMinTrackSizingFunction` and back | trait-impl | 1464 |  | `MinTrackSizingFunctionFromExpandedMinTrackSizingFunction` | done |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for TrackSizingFunction` | trait-impl | 1503 |  | `TrackSizingFunctionAuto` | dev:D-COND |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for TrackSizingFunction` | trait-impl | 1503 |  | `TrackSizingFunctionMinContent` | dev:D-COND |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for TrackSizingFunction` | trait-impl | 1503 |  | `TrackSizingFunctionMaxContent` | dev:D-COND |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for TrackSizingFunction` | trait-impl | 1503 |  | `TrackSizingFunctionZero` | dev:D-COND |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for TrackSizingFunction` | trait-impl | 1503 |  | `TrackSizingFunctionFitContent` | dev:D-COND |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for TrackSizingFunction` | trait-impl | 1503 |  | methods on `MinMax` | dev:D-COND |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension> for TrackSizingFunction` | trait-impl | 1541 |  | `TrackSizingFunctionFromLengthPercentage` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension> for TrackSizingFunction` | trait-impl | 1541 |  | `TrackSizingFunctionFromLengthPercentageAuto` | done |
| `impl From<LengthPercentage / LengthPercentageAuto / Dimension> for TrackSizingFunction` | trait-impl | 1541 |  | `TrackSizingFunctionFromDimension` | done |
| `impl From<u16> for RepetitionCount` | trait-impl | 1597 |  | `RepetitionCountFromU16` | done |
| `impl Display` / `Error for InvalidStringRepetitionValue` | trait-impl | 1608 |  | `(InvalidStringRepetitionValue).Error` | done |
| `impl TryFrom<&str> for RepetitionCount` | trait-impl | 1615 |  | `RepetitionCountTryFromStr` | done |
| `impl GenericRepetition for &GridTemplateRepetition` | trait-impl | 1655 |  | `GridTemplateRepetitionRef` (`(*GridTemplateRepetition).Ref`) | dev:D-TAFFY-TRAITS |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for GridTemplateComponent` | trait-impl | 1715 |  | `(GridTemplateComponent).Auto` | done |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for GridTemplateComponent` | trait-impl | 1715 |  | `(GridTemplateComponent).MinContent` | done |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for GridTemplateComponent` | trait-impl | 1715 |  | `(GridTemplateComponent).MaxContent` | done |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for GridTemplateComponent` | trait-impl | 1715 |  | `(GridTemplateComponent).FitContent` | done |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for GridTemplateComponent` | trait-impl | 1715 |  | `(GridTemplateComponent).Zero` | done |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for GridTemplateComponent` | trait-impl | 1715 |  | `(GridTemplateComponent).FromLength` | done |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for GridTemplateComponent` | trait-impl | 1715 |  | `(GridTemplateComponent).FromPercent` | done |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent` / `TaffyFitContent` / `TaffyZero` / `FromLength` / `FromPercent` / `FromFr for GridTemplateComponent` | trait-impl | 1715 |  | `(GridTemplateComponent).FromFr` | done |
| `impl From<MinMax<...>> for GridTemplateComponent` | trait-impl | 1749 |  | `GridTemplateComponentFromMinMax` | done |
| `impl Default for GridTemplateTracks` | trait-impl | 1790 |  | `GridTemplateTracksDefault` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `grid_placement_parser_saturates_numeric_values` | test | 1862 | parse | `TestStyleGrid_GridPlacementParserSaturatesNumericValues` | n/a:D-TAFFY-FEATURE |
| `repetition_parser_saturates_numeric_values` | test | 1881 | parse | `TestStyleGrid_RepetitionParserSaturatesNumericValues` | n/a:D-TAFFY-FEATURE |
| `repetition_track_count_saturates` | test | 1892 | parse | `TestStyleGrid_RepetitionTrackCountSaturates` | n/a:D-TAFFY-FEATURE |
| `max_track_sizing_function_round_trips` | test | 1907 |  | `TestStyleGrid_ExpandTests_MaxTrackSizingFunctionRoundTrips` | done |
| `min_track_sizing_function_round_trips` | test | 1930 |  | `TestStyleGrid_ExpandTests_MinTrackSizingFunctionRoundTrips` | done |
| `track_sizing_function_calc_round_trips` | test | 1947 | calc | `TestStyleGrid_ExpandTests_TrackSizingFunctionCalcRoundTrips` | n/a:D-TAFFY-FEATURE |

## src/style/mod.rs

Go target: `internal/taffy/style_mod.go` (package `taffy`), tests in `style_mod_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CheapCloneStr` | trait | 63 |  | `CheapCloneStr` | done |
| `CoreStyle` | trait | 77 |  | `CoreStyle` | done |
| `CoreStyle::box_generation_mode` | trait-method | 83 |  | `CoreStyle.BoxGenerationMode` | done |
| `CoreStyle::is_block` | trait-method | 92 |  | `CoreStyle.IsBlock` | done |
| `CoreStyle::is_compressible_replaced` | trait-method | 101 |  | `CoreStyle.IsCompressibleReplaced` | done |
| `CoreStyle::is_replaced` | trait-method | 109 |  | `CoreStyle.IsReplaced` | done |
| `CoreStyle::box_sizing` | trait-method | 114 |  | `CoreStyle.BoxSizing` | done |
| `CoreStyle::direction` | trait-method | 120 |  | `CoreStyle.Direction` | done |
| `CoreStyle::overflow` | trait-method | 127 |  | `CoreStyle.Overflow` | done |
| `CoreStyle::scrollbar_width` | trait-method | 132 |  | `CoreStyle.ScrollbarWidth` | done |
| `CoreStyle::position` | trait-method | 139 |  | `CoreStyle.Position` | done |
| `CoreStyle::inset` | trait-method | 144 |  | `CoreStyle.Inset` | done |
| `CoreStyle::is_containing_block` | trait-method | 158 |  | `CoreStyle.IsContainingBlock` | done |
| `CoreStyle::size` | trait-method | 165 |  | `CoreStyle.Size` | done |
| `CoreStyle::min_size` | trait-method | 170 |  | `CoreStyle.MinSize` | done |
| `CoreStyle::max_size` | trait-method | 175 |  | `CoreStyle.MaxSize` | done |
| `CoreStyle::aspect_ratio` | trait-method | 181 |  | `CoreStyle.AspectRatio` | done |
| `CoreStyle::margin` | trait-method | 188 |  | `CoreStyle.Margin` | done |
| `CoreStyle::padding` | trait-method | 193 |  | `CoreStyle.Padding` | done |
| `CoreStyle::border` | trait-method | 198 |  | `CoreStyle.Border` | done |
| `CoreStyle::contain` | trait-method | 204 |  | `CoreStyle.Contain` | done |
| `OofItemStyle` | trait | 211 |  | `OofItemStyle` | done |
| `OofItemStyle::align_self` | trait-method | 215 |  | `OofItemStyle.AlignSelf` | done |
| `OofItemStyle::justify_self` | trait-method | 221 |  | `OofItemStyle.JustifySelf` | done |
| `OofItemStyle::is_table` | trait-method | 227 |  | `OofItemStyle.IsTable` | done |
| `OofItemStyle::grid_row` | trait-method | 234 | grid | `OofItemStyle.GridRow` | done |
| `OofItemStyle::grid_column` | trait-method | 241 | grid | `OofItemStyle.GridColumn` | done |
| `Display` | enum | 251 |  | `Display` | done |
| `Display::DEFAULT` | assoc-const | 271 | flexbox | `DisplayDefault` | done |
| `Display::DEFAULT` | assoc-const | 275 | grid, flexbox | `DisplayDefault` | n/a:D-TAFFY-FEATURE |
| `Display::DEFAULT` | assoc-const | 279 | block_layout, flexbox, grid | `DisplayDefault` | n/a:D-TAFFY-FEATURE |
| `Display::DEFAULT` | assoc-const | 283 | flexbox, grid, block_layout | `DisplayDefault` | n/a:D-TAFFY-FEATURE |
| `BoxGenerationMode` | enum | 325 |  | `BoxGenerationMode` | done |
| `BoxGenerationMode::DEFAULT` | assoc-const | 334 |  | `BoxGenerationModeDefault` | done |
| `Position` | enum | 354 |  | `Position` | done |
| `Position::is_out_of_flow` | method | 382 |  | `(Position).IsOutOfFlow` | done |
| `Position::is_positioned` | method | 389 |  | `(Position).IsPositioned` | done |
| `ContainingBlockClaims` | struct | 398 |  | `ContainingBlockClaims` | done |
| `ContainingBlockClaims::NONE` | assoc-const | 407 |  | `ContainingBlockClaimsNone` | done |
| `ContainingBlockClaims::ALL` | assoc-const | 409 |  | `ContainingBlockClaimsAll` | done |
| `ContainingBlockClaims::for_position` | method | 413 |  | `(ContainingBlockClaims).ForPosition` | done |
| `BoxSizing` | enum | 446 |  | `BoxSizing` | done |
| `Overflow` | enum | 475 |  | `Overflow` | done |
| `Overflow::is_scroll_container` | method | 496 |  | `(Overflow).IsScrollContainer` | done |
| `Contain` | struct | 540 |  | `Contain` | done |
| `Contain::NONE` | assoc-const | 544 |  | `ContainNone` | done |
| `Contain::LAYOUT` | assoc-const | 548 |  | `ContainLayout` | done |
| `Contain::PAINT` | assoc-const | 552 |  | `ContainPaint` | done |
| `Contain::CONTENT` | assoc-const | 554 |  | `ContainContent` | done |
| `Contain::DEFAULT` | assoc-const | 557 |  | `ContainDefault` | done |
| `Contain::contains` | method | 561 |  | `(Contain).Contains` | done |
| `Contain::intersects` | method | 567 |  | `(Contain).Intersects` | done |
| `Contain::union` | method | 573 |  | `(Contain).Union` | done |
| `Contain::establishes_independent_formatting_context` | method | 580 |  | `(Contain).EstablishesIndependentFormattingContext` | done |
| `Contain::suppresses_baseline` | method | 587 |  | `(Contain).SuppressesBaseline` | done |
| `Contain::contains_scrollable_overflow` | method | 595 |  | `(Contain).ContainsScrollableOverflow` | done |
| `Direction` | enum | 670 |  | `Direction` | done |
| `Direction::is_rtl` | method | 681 |  | `(Direction).IsRtl` | done |
| `Style` | struct | 709 |  | `Style` | done |
| `Style::DEFAULT` | assoc-const | 876 |  | `StyleDefault` | done |
| default methods of `CoreStyle` / `OofItemStyle` | trait-impl | 77 |  | `CoreStyleDefaults` | dev:D-TAFFY-TRAITS |
| default methods of `CoreStyle` / `OofItemStyle` | trait-impl | 77 |  | `CoreStyleDefaultIsContainingBlock` | dev:D-TAFFY-TRAITS |
| default methods of `CoreStyle` / `OofItemStyle` | trait-impl | 77 |  | `OofItemStyleDefaults` | dev:D-TAFFY-TRAITS |
| `CoreStyle::CustomIdent` (associated type) | assoc-type | 79 |  | type parameter `S` of `OofItemStyle[S]` | dev:D-TAFFY-TRAITS |
| `Overflow::maybe_into_automatic_min_size` (`pub(crate)`) | method | 505 |  | `(Overflow).maybeIntoAutomaticMinSize` | done |
| `impl BitOr` / `BitOrAssign for Contain` | trait-impl | 600 |  | `(Contain).Union` | done |
| `impl Default for Display` / `BoxGenerationMode` / `Position` / `BoxSizing` / `Overflow` / `Contain` / `Direction` | trait-impl | 287 |  | `DisplayDefault` | done |
| `impl Default for Display` / `BoxGenerationMode` / `Position` / `BoxSizing` / `Overflow` / `Contain` / `Direction` | trait-impl | 287 |  | `BoxGenerationModeDefault` | done |
| `impl Default for Display` / `BoxGenerationMode` / `Position` / `BoxSizing` / `Overflow` / `Contain` / `Direction` | trait-impl | 287 |  | `PositionDefault` | done |
| `impl Default for Display` / `BoxGenerationMode` / `Position` / `BoxSizing` / `Overflow` / `Contain` / `Direction` | trait-impl | 287 |  | `BoxSizingDefault` | done |
| `impl Default for Display` / `BoxGenerationMode` / `Position` / `BoxSizing` / `Overflow` / `Contain` / `Direction` | trait-impl | 287 |  | `OverflowDefault` | done |
| `impl Default for Display` / `BoxGenerationMode` / `Position` / `BoxSizing` / `Overflow` / `Contain` / `Direction` | trait-impl | 287 |  | `ContainDefault` | done |
| `impl Default for Display` / `BoxGenerationMode` / `Position` / `BoxSizing` / `Overflow` / `Contain` / `Direction` | trait-impl | 287 |  | `DirectionDefault` | done |
| `impl core::fmt::Display for Display` | trait-impl | 304 |  | `(Display).String` | done |
| `impl FromCss` for `Display` / `Position` / `BoxSizing` / `Overflow` / `Contain` / `Direction` | trait-impl | 292 | parse | - | n/a:D-TAFFY-FEATURE |
| `Style::dummy` (field) | field | 712 |  | - | n/a:D-TAFFY-TRAITS |
| `impl Default for Style` | trait-impl | 966 |  | `StyleDefault` | done |
| `impl CoreStyle for Style` | trait-impl | 972 |  | `StyleRef` | dev:D-TAFFY-TRAITS |
| `impl CoreStyle for Style` | trait-impl | 972 |  | `(*Style).Ref` | dev:D-TAFFY-TRAITS |
| `impl OofItemStyle for Style` | trait-impl | 1061 |  | `StyleRef` | dev:D-TAFFY-TRAITS |
| `impl CoreStyle` / `OofItemStyle` / `BlockContainerStyle` / `BlockItemStyle` / `FlexboxContainerStyle` / `FlexboxItemStyle` / `GridContainerStyle` / `GridItemStyle for &T` | trait-impl | 1086 |  | - (Go interface values already forward) | n/a:D-TAFFY-TRAITS |
| `impl BlockContainerStyle for Style` | trait-impl | 1196 |  | `StyleRef` | dev:D-TAFFY-TRAITS |
| `impl BlockItemStyle for Style` | trait-impl | 1232 |  | `StyleRef` | dev:D-TAFFY-TRAITS |
| `impl FlexboxContainerStyle for Style` | trait-impl | 1290 |  | `StyleRef` | dev:D-TAFFY-TRAITS |
| `impl FlexboxItemStyle for Style` | trait-impl | 1346 |  | `StyleRef` | dev:D-TAFFY-TRAITS |
| `impl GridContainerStyle for Style` | trait-impl | 1386 |  | `StyleRef` | dev:D-TAFFY-GRIDLISTS |
| `impl GridItemStyle for Style` | trait-impl | 1561 |  | `StyleRef` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `defaults_match` | test | 1631 |  | `TestStyleMod_DefaultsMatch` | done |
| `parse_contain` | test | 1712 | parse | `TestStyleMod_ParseContain` | n/a:D-TAFFY-FEATURE |
| `style_sizes` | test | 1742 |  | `TestStyleMod_StyleSizes` | dev:D-TAFFY-SIZES |

## src/style_helpers.rs

Go target: `internal/taffy/style_helpers.go` (package `taffy`), tests in `style_helpers_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `repeat` | fn | 23 | grid | `Repeat` | done |
| `evenly_sized_tracks` | fn | 38 | grid | `EvenlySizedTracks` | done |
| `line` | fn | 52 |  | `NewLine` | dev:D-TAFFY-HELPERS |
| `TaffyGridLine` | trait | 56 |  | `TaffyGridLine` | done |
| `TaffyGridLine::from_line_index` | trait-method | 58 |  | `TaffyGridLine.FromLineIndex` | done |
| `span` | fn | 63 |  | `Span` | done |
| `TaffyGridSpan` | trait | 67 |  | `TaffyGridSpan` | done |
| `TaffyGridSpan::from_span` | trait-method | 69 |  | `TaffyGridSpan.FromSpan` | done |
| `minmax` | fn | 75 | grid | `Minmax` | done |
| `flex` | fn | 85 | grid | `Flex` | done |
| `zero` | fn | 95 |  | `Zero` | done |
| `TaffyZero` | trait | 100 |  | `TaffyZero` | done |
| `Point::zero` | assoc-fn | 117 |  | `PointZero` | done |
| `Line::zero` | assoc-fn | 128 |  | `LineZero` | done |
| `Size::zero` | assoc-fn | 139 |  | `SizeZero` | done |
| `Rect::zero` | assoc-fn | 150 |  | `RectZero` | done |
| `auto` | fn | 157 |  | `Auto` | done |
| `TaffyAuto` | trait | 162 |  | `TaffyAuto` | done |
| `Point::auto` | assoc-fn | 176 |  | `PointAuto` | done |
| `Line::auto` | assoc-fn | 187 |  | `LineAuto` | done |
| `Size::auto` | assoc-fn | 198 |  | `SizeAuto` | done |
| `Rect::auto` | assoc-fn | 209 |  | `RectAuto` | done |
| `min_content` | fn | 216 |  | `MinContent` | done |
| `TaffyMinContent` | trait | 221 |  | `TaffyMinContent` | done |
| `Point::min_content` | assoc-fn | 235 |  | `PointMinContent` | done |
| `Line::min_content` | assoc-fn | 246 |  | `LineMinContent` | done |
| `Size::min_content` | assoc-fn | 257 |  | `SizeMinContent` | done |
| `Rect::min_content` | assoc-fn | 269 |  | `RectMinContent` | done |
| `max_content` | fn | 276 |  | `MaxContent` | done |
| `TaffyMaxContent` | trait | 281 |  | `TaffyMaxContent` | done |
| `Point::max_content` | assoc-fn | 295 |  | `PointMaxContent` | done |
| `Line::max_content` | assoc-fn | 306 |  | `LineMaxContent` | done |
| `Size::max_content` | assoc-fn | 317 |  | `SizeMaxContent` | done |
| `Rect::max_content` | assoc-fn | 329 |  | `RectMaxContent` | done |
| `fit_content` | fn | 337 |  | `FitContent` | done |
| `TaffyFitContent` | trait | 342 |  | `TaffyFitContent` | done |
| `TaffyFitContent::fit_content` | trait-method | 344 |  | `TaffyFitContent.FitContent` | done |
| `Point::fit_content` | assoc-fn | 356 |  | `PointFitContent` | done |
| `Line::fit_content` | assoc-fn | 370 |  | `LineFitContent` | done |
| `Size::fit_content` | assoc-fn | 384 |  | `SizeFitContent` | done |
| `Rect::fit_content` | assoc-fn | 403 |  | `RectFitContent` | done |
| `length` | fn | 410 |  | `Length` | done |
| `FromLength` | trait | 415 |  | `FromLength` | done |
| `FromLength::from_length` | trait-method | 417 |  | `FromLength.FromLength` | done |
| `Point::length` | assoc-fn | 440 |  | `PointLength` | done |
| `Line::length` | assoc-fn | 453 |  | `LineLength` | done |
| `Size::length` | assoc-fn | 466 |  | `SizeLength` | done |
| `Rect::length` | assoc-fn | 484 |  | `RectLength` | done |
| `percent` | fn | 491 |  | `Percent` | done |
| `FromPercent` | trait | 496 |  | `FromPercent` | done |
| `FromPercent::from_percent` | trait-method | 498 |  | `FromPercent.FromPercent` | done |
| `Point::percent` | assoc-fn | 522 |  | `PointPercent` | done |
| `Line::percent` | assoc-fn | 536 |  | `LinePercent` | done |
| `Size::percent` | assoc-fn | 550 |  | `SizePercent` | done |
| `Rect::percent` | assoc-fn | 569 |  | `RectPercent` | done |
| `fr` | fn | 577 | grid | `Fr` | done |
| `FromFr` | trait | 582 |  | `FromFr` | done |
| `FromFr::from_fr` | trait-method | 584 |  | `FromFr.FromFr` | done |
| `impl TaffyZero for f32` / `Option<T>`, `impl FromLength` / `FromPercent for f32` / `Option<f32>` | trait-impl | 103 |  | runtime cases of `zeroOf` / `lengthOf` / `percentOf` (only `Option[float32]`) | dev:D-TAFFY-HELPERS |
| `impl TaffyAuto` / `TaffyMinContent` / `TaffyMaxContent for Option<T>` | trait-impl | 165 |  | - | dev:D-TAFFY-HELPERS |
| `impl Taffy* / From* for Point<T>` / `Line<T>` / `Size<T>` / `Rect<T>` | trait-impl | 110 |  | methods `Zero`, `Auto`, `MinContent`, `MaxContent`, `FitContent`, `FromLength`, `FromPercent` on `Point` / `Line` / `Size` / `Rect` | dev:D-COND |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_repeat_u16` | test | 600 | grid | `TestStyleHelpers_RepeatFnTests_RepeatU16` | done |
| `test_repeat_auto_fit_str` | test | 612 | grid | `TestStyleHelpers_RepeatFnTests_RepeatAutoFitStr` | done |
| `test_repeat_auto_fill_str` | test | 624 | grid | `TestStyleHelpers_RepeatFnTests_RepeatAutoFillStr` | done |

## src/tree/cache.rs

Go target: `internal/taffy/tree_cache.go` (package `taffy`), tests in `tree_cache_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `cache_mode_change_evictions` | fn | 197 | std | `CacheModeChangeEvictions` | done |
| `Cache` | struct | 214 |  | `Cache` | done |
| `Cache::new` | assoc-fn | 239 |  | `NewCache` | done |
| `Cache::get` | method | 254 |  | `(Cache).Get` | done |
| `Cache::store` | method | 283 |  | `(Cache).Store` | done |
| `Cache::clear` | method | 341 |  | `(Cache).Clear` | done |
| `Cache::is_empty` | method | 356 |  | `(Cache).IsEmpty` | done |
| `ClearState` | enum | 362 |  | `ClearState` | done |
| `impl Default for Cache` | trait-impl | 231 |  | `NewCache` | done |
| `derive(Clone) for Cache` | trait-impl | 202 |  | `(Cache).Clone` | dev:D-TAFFY-CLONE |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `recently_used_measure_entries_get_a_second_chance` | test | 393 |  | `TestTreeCache_RecentlyUsedMeasureEntriesGetASecondChance` | done |
| `storing_an_existing_measurement_updates_it_in_place` | test | 408 |  | `TestTreeCache_StoringAnExistingMeasurementUpdatesItInPlace` | done |
| `measurements_with_margin_collapse_metadata_are_not_cached` | test | 419 |  | `TestTreeCache_MeasurementsWithMarginCollapseMetadataAreNotCached` | done |
| `retrieving_a_measurement_only_marks_its_slot_as_used` | test | 434 |  | `TestTreeCache_RetrievingAMeasurementOnlyMarksItsSlotAsUsed` | done |
| `cache_mode_distinguishes_every_combination_of_inputs` | test | 466 |  | `TestTreeCache_CacheModeDistinguishesEveryCombinationOfInputs` | done |
| `retrieving_with_a_different_mode_misses_and_keeps_the_entries` | test | 481 |  | `TestTreeCache_RetrievingWithADifferentModeMissesAndKeepsTheEntries` | done |
| `storing_with_a_different_mode_drops_the_existing_entries` | test | 498 |  | `TestTreeCache_StoringWithADifferentModeDropsTheExistingEntries` | done |
| `hidden_layouts_do_not_affect_the_cache` | test | 530 |  | `TestTreeCache_HiddenLayoutsDoNotAffectTheCache` | done |
| `mode_change_evictions_are_counted_for_non_empty_caches_only` | test | 539 | std | `TestTreeCache_ModeChangeEvictionsAreCountedForNonEmptyCachesOnly` | done |

## src/tree/layout.rs

Go target: `internal/taffy/tree_layout.go` (package `taffy`), tests in `tree_layout_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `RunMode` | enum | 12 |  | `RunMode` | done |
| `SizingMode` | enum | 25 |  | `SizingMode` | done |
| `CollapsibleMarginSet` | struct | 35 |  | `CollapsibleMarginSet` | done |
| `CollapsibleMarginSet::ZERO` | assoc-const | 44 |  | `CollapsibleMarginSetZero` | done |
| `CollapsibleMarginSet::from_margin` | assoc-fn | 47 |  | `CollapsibleMarginSetFromMargin` | done |
| `CollapsibleMarginSet::collapse_with_margin` | method | 56 |  | `(CollapsibleMarginSet).CollapseWithMargin` | done |
| `CollapsibleMarginSet::collapse_with_set` | method | 66 |  | `(CollapsibleMarginSet).CollapseWithSet` | done |
| `CollapsibleMarginSet::resolve` | method | 74 |  | `(CollapsibleMarginSet).Resolve` | done |
| `RequestedAxis` | enum | 82 |  | `RequestedAxis` | done |
| `RequestedAxis::transpose` | method | 94 |  | `(RequestedAxis).Transpose` | done |
| `LayoutInput` | struct | 125 |  | `LayoutInput` | done |
| `LayoutInput::HIDDEN` | assoc-const | 168 |  | `LayoutInputHidden` | done |
| `LayoutInput::transpose` | method | 187 |  | `(LayoutInput).Transpose` | done |
| `Baselines` | struct | 208 |  | `Baselines` | done |
| `Baselines::NONE` | assoc-const | 217 |  | `BaselinesNone` | done |
| `Baselines::from_first` | assoc-fn | 220 |  | `BaselinesFromFirst` | done |
| `AxisStaticEdge` | enum | 234 |  | `AxisStaticEdge` | done |
| `AxisStaticAlign` | struct | 250 |  | `AxisStaticAlign` | done |
| `AxisStaticAlign::from_keyword` | assoc-fn | 263 |  | `AxisStaticAlignFromKeyword` | done |
| `AxisStaticPosition` | struct | 278 |  | `AxisStaticPosition` | done |
| `AxisStaticPosition::from_edge` | assoc-fn | 288 |  | `AxisStaticPositionFromEdge` | done |
| `AxisStaticPosition::from_alignment` | assoc-fn | 302 |  | `AxisStaticPositionFromAlignment` | done |
| `OofCandidate` | struct | 341 |  | `OofCandidate` | done |
| `OofCandidates` | struct | 356 |  | `OofCandidates` | done |
| `OofCandidates::NONE` | assoc-const | 363 |  | `OofCandidatesNone` | done |
| `OofCandidates::new` | assoc-fn | 366 |  | `NewOofCandidates` | done |
| `OofCandidates::is_empty` | method | 372 |  | `(OofCandidates).IsEmpty` | done |
| `OofCandidates::as_slice` | method | 378 |  | `(OofCandidates).AsSlice` | done |
| `OofCandidates::as_mut_slice` | method | 384 |  | `(OofCandidates).AsMutSlice` | done |
| `OofCandidates::push` | method | 390 |  | `(OofCandidates).Push` | done |
| `OofCandidates::append` | method | 396 |  | `(OofCandidates).Append` | done |
| `OofCandidates::take` | method | 408 |  | `(OofCandidates).Take` | done |
| `OofCandidates::clear` | method | 414 |  | `(OofCandidates).Clear` | done |
| `OofCandidates::iter` | method | 420 |  | `(OofCandidates).Iter` | done |
| `OofCandidates::translate` | method | 429 |  | `(OofCandidates).Translate` | done |
| `OofPositioningArea` | struct | 457 |  | `OofPositioningArea` | done |
| `LayoutOutput` | struct | 472 |  | `LayoutOutput` | done |
| `LayoutOutput::HIDDEN` | assoc-const | 511 |  | `LayoutOutputHidden` | done |
| `LayoutOutput::DEFAULT` | assoc-const | 524 |  | `LayoutOutputDefault` | done |
| `LayoutOutput::from_sizes_and_baselines` | assoc-fn | 527 |  | `LayoutOutputFromSizesAndBaselines` | done |
| `LayoutOutput::from_sizes` | assoc-fn | 546 |  | `LayoutOutputFromSizes` | done |
| `LayoutOutput::transpose` | method | 554 |  | `(LayoutOutput).Transpose` | done |
| `LayoutOutput::from_outer_size` | assoc-fn | 571 |  | `LayoutOutputFromOuterSize` | done |
| `Layout` | struct | 579 |  | `Layout` | done |
| `Layout::new` | assoc-fn | 627 |  | `NewLayout` | done |
| `Layout::with_order` | assoc-fn | 646 |  | `LayoutWithOrder` | done |
| `Layout::content_box_width` | method | 662 |  | `(Layout).ContentBoxWidth` | done |
| `Layout::content_box_height` | method | 668 |  | `(Layout).ContentBoxHeight` | done |
| `Layout::content_box_size` | method | 674 |  | `(Layout).ContentBoxSize` | done |
| `Layout::content_box_x` | method | 679 |  | `(Layout).ContentBoxX` | done |
| `Layout::content_box_y` | method | 684 |  | `(Layout).ContentBoxY` | done |
| `Layout::scroll_width` | method | 693 | content_size | `(Layout).ScrollWidth` | done |
| `Layout::scroll_height` | method | 704 | content_size | `(Layout).ScrollHeight` | done |
| `DetailedLayoutInfo` | enum | 717 |  | `DetailedLayoutInfo` | done |
| `impl From<AbsoluteAxis> for RequestedAxis` | trait-impl | 103 |  | `RequestedAxisFromAbsoluteAxis` | done |
| `impl TryFrom<RequestedAxis> for AbsoluteAxis` | trait-impl | 111 |  | `AbsoluteAxisTryFromRequestedAxis` | done |
| `impl Default for OofCandidates` | trait-impl | 355 |  | `NewOofCandidates` | done |
| `derive(Clone, PartialEq) for OofCandidates` | trait-impl | 355 |  | `(OofCandidates).Clone` | dev:D-TAFFY-CLONE |
| `derive(Clone, PartialEq) for OofCandidates` | trait-impl | 355 |  | `(OofCandidates).Equal` | dev:D-TAFFY-CLONE |
| `impl IntoIterator for &OofCandidates` | trait-impl | 439 |  | `(OofCandidates).Iter` | done |
| `derive(Clone, PartialEq) for LayoutOutput` | trait-impl | 470 |  | `(LayoutOutput).Clone` | dev:D-TAFFY-CLONE |
| `derive(Clone, PartialEq) for LayoutOutput` | trait-impl | 470 |  | `(LayoutOutput).Equal` | dev:D-TAFFY-CLONE |
| `impl Default for Layout` | trait-impl | 614 |  | `NewLayout` | done |

## src/tree/mod.rs

Go target: `internal/taffy/tree_mod.go` (package `taffy`), tests in `tree_mod_test.go`

No public items or tests.

## src/tree/node.rs

Go target: `internal/taffy/tree_node.go` (package `taffy`), tests in `tree_node_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `NodeId` | struct | 13 |  | `NodeID` | done |
| `NodeId::new` | assoc-fn | 16 |  | `NewNodeID` | done |
| `impl From<u64> for NodeId` | trait-impl | 21 |  | `NodeIDFromU64` | done |
| `impl From<NodeId> for u64` | trait-impl | 27 |  | `U64FromNodeID` | done |
| `impl From<usize> for NodeId` | trait-impl | 33 |  | `NodeIDFromUsize` | done |
| `impl From<NodeId> for usize` | trait-impl | 39 |  | `UsizeFromNodeID` | done |
| `impl From<DefaultKey> for NodeId` | trait-impl | 47 | taffy_tree | `nodeIDFromKey` | done |
| `impl From<NodeId> for DefaultKey` | trait-impl | 55 | taffy_tree | `keyFromNodeID` | done |
| `derive(Debug) for NodeId` | trait-impl | 12 |  | `(NodeID).String` | done |

## src/tree/taffy_tree.rs

Go target: `internal/taffy/tree_taffy_tree.go` (package `taffy`), tests in `tree_taffy_tree_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TaffyResult` | type | 38 |  | `TaffyResult` | dev:D-TAFFY-RESULT |
| `TaffyError` | enum | 42 |  | `TaffyError` | done |
| `TaffyError::ChildIndexOutOfBounds` | variant | 44 |  | `TaffyErrorChildIndexOutOfBounds` | done |
| `TaffyError::InvalidParentNode` | variant | 53 |  | `TaffyErrorInvalidParentNode` | done |
| `TaffyError::InvalidChildNode` | variant | 55 |  | `TaffyErrorInvalidChildNode` | done |
| `TaffyError::InvalidInputNode` | variant | 57 |  | `TaffyErrorInvalidInputNode` | done |
| `impl Display for TaffyError` | trait-impl | 60 |  | `(TaffyError).Error` | done |
| `impl std::error::Error for TaffyError` | trait-impl | 76 | std | `(TaffyError).Error` | done |
| `TaffyConfig` (`pub(crate)`) + Default | struct | 80 |  | `taffyConfig` | done |
| `NodeData` (private) | struct | 95 |  | `nodeData` | done |
| `NodeData::new` | assoc-fn | 124 |  | `newNodeData` | done |
| `NodeData::mark_dirty` | method | 141 |  | `(nodeData).markDirty` | done |
| `TaffyTree` | struct | 150 |  | `TaffyTree` | done |
| `TaffyTreeChildIter` | struct | 178 |  | `TaffyTreeChildIter` | done |
| `impl Default for TaffyTree` | trait-impl | 171 |  | `NewTaffyTree` | done |
| `impl TraversePartialTree for TaffyTree` | trait-impl | 189 |  | `(TaffyTree).ChildIds` | done |
| `impl TraverseTree for TaffyTree` | trait-impl | 212 |  | `(TaffyTree).TraverseTree` | done |
| `impl CacheTree for TaffyTree` | trait-impl | 215 |  | `(TaffyTree).CacheGet` | done |
| `impl PrintTree for TaffyTree` | trait-impl | 230 |  | `(TaffyTree).GetDebugLabel` | done |
| `TaffyView` (`pub(crate)`) | struct | 270 |  | `taffyView` | done |
| `TaffyView::compute_child_layout` (dispatch by Display) | method | 287 |  | `(taffyView).computeChildLayout` | done |
| `impl TraversePartialTree / TraverseTree / LayoutPartialTree / LayoutContainingBlock / CacheTree / LayoutBlockContainer / LayoutFlexboxContainer / LayoutGridContainer / RoundTree for TaffyView` | trait-impl | 347 |  | `(taffyView).ComputeChildLayout` | done |
| `FnMut(LayoutInput, NodeId, Option<&mut NodeContext>, &Style) -> LayoutOutput` measure bound | type | 974 |  | `MeasureFunction` | done |
| `TaffyTree::as_layout_tree` (`cfg(test)`) | method | 1000 |  | `(TaffyTree).asLayoutTree` | done |
| `slotmap::SlotMap` (dependency, slotmap 1.1.1) | struct | 6 |  | `slotMap` | done |
| `slotmap::SparseSecondaryMap` (dependency, slotmap 1.1.1) | struct | 5 |  | `sparseSecondaryMap` | done |
| `slotmap::KeyData` with `as_ffi` / `from_ffi` (dependency, slotmap 1.1.1) | struct | 6 |  | `keyData` | done |
| `TaffyTree::new` | assoc-fn | 591 |  | `NewTaffyTree` | done |
| `TaffyTree::with_capacity` | assoc-fn | 597 |  | `TaffyTreeWithCapacity` | done |
| `TaffyTree::enable_rounding` | method | 610 |  | `(TaffyTree).EnableRounding` | done |
| `TaffyTree::disable_rounding` | method | 615 |  | `(TaffyTree).DisableRounding` | done |
| `TaffyTree::new_leaf` | method | 620 |  | `(TaffyTree).NewLeaf` | done |
| `TaffyTree::new_leaf_with_context` | method | 631 |  | `(TaffyTree).NewLeafWithContext` | done |
| `TaffyTree::new_with_children` | method | 645 |  | `(TaffyTree).NewWithChildren` | done |
| `TaffyTree::clear` | method | 659 |  | `(TaffyTree).Clear` | done |
| `TaffyTree::remove` | method | 672 |  | `(TaffyTree).Remove` | done |
| `TaffyTree::set_node_context` | method | 698 |  | `(TaffyTree).SetNodeContext` | done |
| `TaffyTree::get_node_context` | method | 715 |  | `(TaffyTree).GetNodeContext` | done |
| `TaffyTree::get_node_context_mut` | method | 721 |  | `(TaffyTree).GetNodeContextMut` | done |
| `TaffyTree::get_disjoint_node_context_mut` | method | 726 |  | `(TaffyTree).GetDisjointNodeContextMut` | done |
| `TaffyTree::add_child` | method | 734 |  | `(TaffyTree).AddChild` | done |
| `TaffyTree::insert_child_at_index` | method | 745 |  | `(TaffyTree).InsertChildAtIndex` | done |
| `TaffyTree::set_children` | method | 761 |  | `(TaffyTree).SetChildren` | done |
| `TaffyTree::remove_child` | method | 790 |  | `(TaffyTree).RemoveChild` | done |
| `TaffyTree::remove_child_at_index` | method | 798 |  | `(TaffyTree).RemoveChildAtIndex` | done |
| `TaffyTree::remove_children_range` | method | 818 |  | `(TaffyTree).RemoveChildrenRange` | dev:D-TAFFY-RANGE |
| `TaffyTree::replace_child_at_index` | method | 834 |  | `(TaffyTree).ReplaceChildAtIndex` | done |
| `TaffyTree::child_at_index` | method | 858 |  | `(TaffyTree).ChildAtIndex` | done |
| `TaffyTree::total_node_count` | method | 870 |  | `(TaffyTree).TotalNodeCount` | done |
| `TaffyTree::parent` | method | 879 |  | `(TaffyTree).Parent` | done |
| `TaffyTree::children` | method | 884 |  | `(TaffyTree).Children` | done |
| `TaffyTree::hoisted_children` | method | 891 |  | `(TaffyTree).HoistedChildren` | done |
| `TaffyTree::set_style` | method | 897 |  | `(TaffyTree).SetStyle` | done |
| `TaffyTree::style` | method | 905 |  | `(TaffyTree).Style` | done |
| `TaffyTree::layout` | method | 911 |  | `(TaffyTree).Layout` | done |
| `TaffyTree::unrounded_layout` | method | 921 |  | `(TaffyTree).UnroundedLayout` | done |
| `TaffyTree::detailed_layout_info` | method | 930 |  | `(TaffyTree).DetailedLayoutInfo` | done |
| `TaffyTree::mark_dirty` | method | 935 |  | `(TaffyTree).MarkDirty` | done |
| `TaffyTree::dirty` | method | 962 |  | `(TaffyTree).Dirty` | done |
| `TaffyTree::compute_layout_with_measure` | method | 967 |  | `(TaffyTree).ComputeLayoutWithMeasure` | done |
| `TaffyTree::compute_layout` | method | 986 |  | `(TaffyTree).ComputeLayout` | done |
| `TaffyTree::print_tree` | method | 994 | std | `(TaffyTree).PrintTree` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `new_should_allocate_default_capacity` | test | 1033 |  | `TestTreeTaffyTree_NewShouldAllocateDefaultCapacity` | done |
| `test_with_capacity` | test | 1043 |  | `TestTreeTaffyTree_WithCapacity` | done |
| `test_new_leaf` | test | 1053 |  | `TestTreeTaffyTree_NewLeaf` | done |
| `new_leaf_with_context` | test | 1065 |  | `TestTreeTaffyTree_NewLeafWithContext` | done |
| `test_new_with_children` | test | 1078 |  | `TestTreeTaffyTree_NewWithChildren` | done |
| `remove_node_should_remove` | test | 1091 |  | `TestTreeTaffyTree_RemoveNodeShouldRemove` | done |
| `remove_node_should_detach_hierarchy` | test | 1100 |  | `TestTreeTaffyTree_RemoveNodeShouldDetachHierarchy` | done |
| `remove_last_node` | test | 1121 |  | `TestTreeTaffyTree_RemoveLastNode` | done |
| `remove_node_marks_parent_dirty` | test | 1134 |  | `TestTreeTaffyTree_RemoveNodeMarksParentDirty` | done |
| `set_measure` | test | 1157 |  | `TestTreeTaffyTree_SetMeasure` | done |
| `set_measure_of_previously_unmeasured_node` | test | 1169 |  | `TestTreeTaffyTree_SetMeasureOfPreviouslyUnmeasuredNode` | done |
| `add_child` | test | 1182 |  | `TestTreeTaffyTree_AddChild` | done |
| `insert_child_at_index` | test | 1197 |  | `TestTreeTaffyTree_InsertChildAtIndex` | done |
| `set_children` | test | 1224 |  | `TestTreeTaffyTree_SetChildren` | done |
| `remove_child` | test | 1246 |  | `TestTreeTaffyTree_RemoveChild` | done |
| `remove_child_at_index` | test | 1263 |  | `TestTreeTaffyTree_RemoveChildAtIndex` | done |
| `remove_children_range` | test | 1280 |  | `TestTreeTaffyTree_RemoveChildrenRange` | done |
| `remove_child_updates_parents` | test | 1303 |  | `TestTreeTaffyTree_RemoveChildUpdatesParents` | done |
| `replace_child_at_index` | test | 1318 |  | `TestTreeTaffyTree_ReplaceChildAtIndex` | done |
| `test_child_at_index` | test | 1333 |  | `TestTreeTaffyTree_ChildAtIndex` | done |
| `test_child_count` | test | 1345 |  | `TestTreeTaffyTree_ChildCount` | done |
| `test_children` | test | 1358 |  | `TestTreeTaffyTree_Children` | done |
| `test_set_style` | test | 1374 |  | `TestTreeTaffyTree_SetStyle` | done |
| `test_style` | test | 1384 |  | `TestTreeTaffyTree_Style` | done |
| `test_layout` | test | 1396 |  | `TestTreeTaffyTree_Layout` | done |
| `test_mark_dirty` | test | 1406 |  | `TestTreeTaffyTree_MarkDirty` | done |
| `compute_layout_should_produce_valid_result` | test | 1431 |  | `TestTreeTaffyTree_ComputeLayoutShouldProduceValidResult` | done |
| `make_sure_layout_location_is_top_left` | test | 1447 |  | `TestTreeTaffyTree_MakeSureLayoutLocationIsTopLeft` | done |
| `set_children_reparents` | test | 1491 |  | `TestTreeTaffyTree_SetChildrenReparents` | done |
| `remove_drops_the_nodes_context` | test | 1513 |  | `TestTreeTaffyTree_RemoveDropsTheNodesContext` | dev:D-TAFFY-CTXDROP |
| `clear_drops_every_nodes_context` | test | 1528 |  | `TestTreeTaffyTree_ClearDropsEveryNodesContext` | dev:D-TAFFY-CTXDROP |
| Go-only: NodeId encoding equals slotmap `KeyData::as_ffi` (gpui LayoutId wraps it) | test | - |  | `TestTreeTaffyTree_NodeIDMatchesSlotMapKeyData` | done |
| Go-only: `TaffyError` Display text | test | - |  | `TestTreeTaffyTree_ErrorDisplay` | done |
| Go-only: `get_disjoint_node_context_mut` disjointness | test | - |  | `TestTreeTaffyTree_DisjointNodeContextMut` | done |

## src/tree/traits.rs

Go target: `internal/taffy/tree_traits.go` (package `taffy`), tests in `tree_traits_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TraversePartialTree` | trait | 147 |  | `TraversePartialTree` | done |
| `TraversePartialTree::child_ids` | trait-method | 154 |  | `TraversePartialTree.ChildIds` | done |
| `TraversePartialTree::child_count` | trait-method | 157 |  | `TraversePartialTree.ChildCount` | done |
| `TraversePartialTree::get_child_id` | trait-method | 160 |  | `TraversePartialTree.GetChildID` | done |
| `TraverseTree` | trait | 167 |  | `TraverseTree` | done |
| `LayoutPartialTree` | trait | 173 |  | `LayoutPartialTree` | done |
| `LayoutPartialTree::get_core_container_style` | trait-method | 189 |  | `LayoutPartialTree.GetCoreContainerStyle` | done |
| `LayoutPartialTree::resolve_calc_value` | trait-method | 193 |  | `LayoutPartialTree.ResolveCalcValue` | done |
| `LayoutPartialTree::set_unrounded_layout` | trait-method | 200 |  | `LayoutPartialTree.SetUnroundedLayout` | done |
| `LayoutPartialTree::compute_child_layout` | trait-method | 203 |  | `LayoutPartialTree.ComputeChildLayout` | done |
| `LayoutContainingBlock` | trait | 212 |  | `LayoutContainingBlock` | done |
| `LayoutContainingBlock::get_oof_item_style` | trait-method | 222 |  | `LayoutContainingBlock.GetOofItemStyle` | done |
| `LayoutContainingBlock::clear_hoisted_children` | trait-method | 229 |  | `LayoutContainingBlock.ClearHoistedChildren` | done |
| `LayoutContainingBlock::add_hoisted_children` | trait-method | 240 |  | `LayoutContainingBlock.AddHoistedChildren` | done |
| `LayoutContainingBlock::get_detailed_layout_info` | trait-method | 251 |  | `LayoutContainingBlock.GetDetailedLayoutInfo` | done |
| `CacheTree` | trait | 260 |  | `CacheTree` | done |
| `CacheTree::cache_get` | trait-method | 262 |  | `CacheTree.CacheGet` | done |
| `CacheTree::cache_store` | trait-method | 265 |  | `CacheTree.CacheStore` | done |
| `CacheTree::cache_clear` | trait-method | 268 |  | `CacheTree.CacheClear` | done |
| `RoundTree` | trait | 275 |  | `RoundTree` | done |
| `RoundTree::get_unrounded_layout` | trait-method | 277 |  | `RoundTree.GetUnroundedLayout` | done |
| `RoundTree::set_final_layout` | trait-method | 279 |  | `RoundTree.SetFinalLayout` | done |
| `RoundTree::is_out_of_flow` | trait-method | 286 |  | `RoundTree.IsOutOfFlow` | done |
| `RoundTree::hoisted_child_count` | trait-method | 289 |  | `RoundTree.HoistedChildCount` | done |
| `RoundTree::get_hoisted_child_id` | trait-method | 292 |  | `RoundTree.GetHoistedChildID` | done |
| `PrintTree` | trait | 298 |  | `PrintTree` | done |
| `PrintTree::get_debug_label` | trait-method | 300 |  | `PrintTree.GetDebugLabel` | done |
| `PrintTree::get_final_layout` | trait-method | 302 |  | `PrintTree.GetFinalLayout` | done |
| `LayoutFlexboxContainer` | trait | 307 | flexbox | `LayoutFlexboxContainer` | done |
| `LayoutFlexboxContainer::get_flexbox_container_style` | trait-method | 318 | flexbox | `LayoutFlexboxContainer.GetFlexboxContainerStyle` | done |
| `LayoutFlexboxContainer::get_flexbox_child_style` | trait-method | 323 | flexbox | `LayoutFlexboxContainer.GetFlexboxChildStyle` | done |
| `LayoutGridContainer` | trait | 328 | grid | `LayoutGridContainer` | done |
| `LayoutGridContainer::get_grid_container_style` | trait-method | 340 | grid | `LayoutGridContainer.GetGridContainerStyle` | done |
| `LayoutGridContainer::get_grid_child_style` | trait-method | 345 | grid | `LayoutGridContainer.GetGridChildStyle` | done |
| `LayoutGridContainer::set_detailed_grid_info` | trait-method | 351 | grid | `LayoutGridContainer.SetDetailedGridInfo` | done |
| `LayoutBlockContainer` | trait | 358 | block_layout | `LayoutBlockContainer` | done |
| `LayoutBlockContainer::get_block_container_style` | trait-method | 369 | block_layout | `LayoutBlockContainer.GetBlockContainerStyle` | done |
| `LayoutBlockContainer::get_block_child_style` | trait-method | 374 | block_layout | `LayoutBlockContainer.GetBlockChildStyle` | done |
| `LayoutBlockContainer::compute_block_child_layout` | trait-method | 378 | block_layout | `LayoutBlockContainer.ComputeBlockChildLayout` | done |
| `LayoutPartialTreeExt` (`pub(crate)`) | trait | 393 |  | `measureChildSize` | done |
| `LayoutPartialTreeExt` (`pub(crate)`) | trait | 393 |  | `measureChildSizeBoth` | done |
| `LayoutPartialTreeExt` (`pub(crate)`) | trait | 393 |  | `performChildLayout` | done |
| `LayoutPartialTreeExt` (`pub(crate)`) | trait | 393 |  | `calc` | done |
| `LayoutPartialTree::resolve_calc_value` default body | trait-method | 193 |  | `LayoutPartialTreeDefaults.ResolveCalcValue` | done |
| `LayoutContainingBlock::get_detailed_layout_info` default body | trait-method | 251 |  | `LayoutContainingBlockDefaults.GetDetailedLayoutInfo` | done |
| `LayoutGridContainer::set_detailed_grid_info` default body | trait-method | 351 | grid | `LayoutGridContainerDefaults.SetDetailedGridInfo` | done |
| `LayoutBlockContainer::compute_block_child_layout` default body | trait-method | 378 | block_layout | `LayoutBlockContainerDefaultComputeBlockChildLayout` | done |

## src/util/debug.rs

Go target: `internal/taffy/util_debug.go` (package `taffy`), tests in `util_debug_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `DebugLogger` | struct | 10 | debug, profile | `DebugLogger` | n/a:D-TAFFY-FEATURE |
| `DebugLogger::new` | assoc-fn | 18 | debug, profile | `NewDebugLogger` | n/a:D-TAFFY-FEATURE |
| `DebugLogger::push_node` | method | 22 | debug, profile | `(DebugLogger).PushNode` | n/a:D-TAFFY-FEATURE |
| `DebugLogger::pop_node` | method | 29 | debug, profile | `(DebugLogger).PopNode` | n/a:D-TAFFY-FEATURE |
| `DebugLogger::log` | method | 34 | debug, profile | `(DebugLogger).Log` | n/a:D-TAFFY-FEATURE |
| `DebugLogger::labelled_log` | method | 42 | debug, profile | `(DebugLogger).LabelledLog` | n/a:D-TAFFY-FEATURE |
| `DebugLogger::debug_log` | method | 50 | debug, profile | `(DebugLogger).DebugLog` | n/a:D-TAFFY-FEATURE |
| `DebugLogger::labelled_debug_log` | method | 58 | debug, profile | `(DebugLogger).LabelledDebugLog` | n/a:D-TAFFY-FEATURE |

## src/util/front_back_vec_builder.rs

Go target: `internal/taffy/util_front_back_vec_builder.go` (package `taffy`), tests in `util_front_back_vec_builder_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `front_items_keep_their_order_and_back_items_are_reversed` | test | 86 |  | `TestUtilFrontBackVecBuilder_FrontItemsKeepTheirOrderAndBackItemsAreReversed` | done |
| `all_items_pushed_to_one_end` | test | 101 |  | `TestUtilFrontBackVecBuilder_AllItemsPushedToOneEnd` | done |
| `zero_capacity` | test | 112 |  | `TestUtilFrontBackVecBuilder_ZeroCapacity` | done |
| `owned_items_are_moved_into_the_vec` | test | 119 |  | `TestUtilFrontBackVecBuilder_OwnedItemsAreMovedIntoTheVec` | done |
| `pushing_too_many_items_to_the_front_panics_in_into_vec` | test | 130 |  | `TestUtilFrontBackVecBuilder_PushingTooManyItemsToTheFrontPanicsInIntoVec` | done |
| `pushing_too_many_items_to_the_back_panics_in_into_vec` | test | 139 |  | `TestUtilFrontBackVecBuilder_PushingTooManyItemsToTheBackPanicsInIntoVec` | done |
| `push_front_out_of_bounds_panics` | test | 148 |  | `TestUtilFrontBackVecBuilder_PushFrontOutOfBoundsPanics` | done |
| `push_back_out_of_bounds_panics` | test | 155 |  | `TestUtilFrontBackVecBuilder_PushBackOutOfBoundsPanics` | done |
| `into_vec_before_full_panics` | test | 163 |  | `TestUtilFrontBackVecBuilder_IntoVecBeforeFullPanics` | done |

## src/util/math.rs

Go target: `internal/taffy/util_math.go` (package `taffy`), tests in `util_math_test.go`

The trait cannot be a Go interface with these impls (D-TAFFY-MAYBEMATH); each impl method is a function or method named as in the notes above. `.min()` / `.max()` inside them are `f32Min` / `f32Max`.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MaybeMath` | trait | 11 |  | - | dev:D-TAFFY-MAYBEMATH |
| `MaybeMath::maybe_min` | trait-method | 13 |  | - | dev:D-TAFFY-MAYBEMATH |
| `MaybeMath::maybe_max` | trait-method | 16 |  | - | dev:D-TAFFY-MAYBEMATH |
| `MaybeMath::maybe_clamp` | trait-method | 19 |  | - | dev:D-TAFFY-MAYBEMATH |
| `MaybeMath::maybe_add` | trait-method | 22 |  | - | dev:D-TAFFY-MAYBEMATH |
| `MaybeMath::maybe_sub` | trait-method | 25 |  | - | dev:D-TAFFY-MAYBEMATH |
| `impl MaybeMath<Option<f32>, Option<f32>> for Option<f32>` (`maybe_min`) | trait-impl | 28 |  | `OptionF32MaybeMin` | done |
| `impl MaybeMath<Option<f32>, Option<f32>> for Option<f32>` (`maybe_max`) | trait-impl | 28 |  | `OptionF32MaybeMax` | done |
| `impl MaybeMath<Option<f32>, Option<f32>> for Option<f32>` (`maybe_clamp`) | trait-impl | 28 |  | `OptionF32MaybeClamp` | done |
| `impl MaybeMath<Option<f32>, Option<f32>> for Option<f32>` (`maybe_add`) | trait-impl | 28 |  | `OptionF32MaybeAdd` | done |
| `impl MaybeMath<Option<f32>, Option<f32>> for Option<f32>` (`maybe_sub`) | trait-impl | 28 |  | `OptionF32MaybeSub` | done |
| `impl MaybeMath<f32, Option<f32>> for Option<f32>` (`maybe_min`) | trait-impl | 76 |  | `OptionF32MaybeMinF32` | done |
| `impl MaybeMath<f32, Option<f32>> for Option<f32>` (`maybe_max`) | trait-impl | 76 |  | `OptionF32MaybeMaxF32` | done |
| `impl MaybeMath<f32, Option<f32>> for Option<f32>` (`maybe_clamp`) | trait-impl | 76 |  | `OptionF32MaybeClampF32` | done |
| `impl MaybeMath<f32, Option<f32>> for Option<f32>` (`maybe_add`) | trait-impl | 76 |  | `OptionF32MaybeAddF32` | done |
| `impl MaybeMath<f32, Option<f32>> for Option<f32>` (`maybe_sub`) | trait-impl | 76 |  | `OptionF32MaybeSubF32` | done |
| `impl MaybeMath<Option<f32>, f32> for f32` (`maybe_min`) | trait-impl | 98 |  | `F32MaybeMin` | done |
| `impl MaybeMath<Option<f32>, f32> for f32` (`maybe_max`) | trait-impl | 98 |  | `F32MaybeMax` | done |
| `impl MaybeMath<Option<f32>, f32> for f32` (`maybe_clamp`) | trait-impl | 98 |  | `F32MaybeClamp` | done |
| `impl MaybeMath<Option<f32>, f32> for f32` (`maybe_add`) | trait-impl | 98 |  | `F32MaybeAdd` | done |
| `impl MaybeMath<Option<f32>, f32> for f32` (`maybe_sub`) | trait-impl | 98 |  | `F32MaybeSub` | done |
| `impl MaybeMath<f32, AvailableSpace> for AvailableSpace` (`maybe_min`) | trait-impl | 137 |  | `(AvailableSpace).MaybeMinF32` | done |
| `impl MaybeMath<f32, AvailableSpace> for AvailableSpace` (`maybe_max`) | trait-impl | 137 |  | `(AvailableSpace).MaybeMaxF32` | done |
| `impl MaybeMath<f32, AvailableSpace> for AvailableSpace` (`maybe_clamp`) | trait-impl | 137 |  | `(AvailableSpace).MaybeClampF32` | done |
| `impl MaybeMath<f32, AvailableSpace> for AvailableSpace` (`maybe_add`) | trait-impl | 137 |  | `(AvailableSpace).MaybeAddF32` | done |
| `impl MaybeMath<f32, AvailableSpace> for AvailableSpace` (`maybe_sub`) | trait-impl | 137 |  | `(AvailableSpace).MaybeSubF32` | done |
| `impl MaybeMath<Option<f32>, AvailableSpace> for AvailableSpace` (`maybe_min`) | trait-impl | 177 |  | `(AvailableSpace).MaybeMin` | done |
| `impl MaybeMath<Option<f32>, AvailableSpace> for AvailableSpace` (`maybe_max`) | trait-impl | 177 |  | `(AvailableSpace).MaybeMax` | done |
| `impl MaybeMath<Option<f32>, AvailableSpace> for AvailableSpace` (`maybe_clamp`) | trait-impl | 177 |  | `(AvailableSpace).MaybeClamp` | done |
| `impl MaybeMath<Option<f32>, AvailableSpace> for AvailableSpace` (`maybe_add`) | trait-impl | 177 |  | `(AvailableSpace).MaybeAdd` | done |
| `impl MaybeMath<Option<f32>, AvailableSpace> for AvailableSpace` (`maybe_sub`) | trait-impl | 177 |  | `(AvailableSpace).MaybeSub` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_min`) | trait-impl | 226 |  | `SizeOptionF32MaybeMin` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_max`) | trait-impl | 226 |  | `SizeOptionF32MaybeMax` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_clamp`) | trait-impl | 226 |  | `SizeOptionF32MaybeClamp` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_add`) | trait-impl | 226 |  | `SizeOptionF32MaybeAdd` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_sub`) | trait-impl | 226 |  | `SizeOptionF32MaybeSub` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_min`, SizeOptionF32MaybeMinF32) | trait-impl | 226 |  | `SizeOptionF32MaybeMinF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_min`, SizeF32MaybeMin) | trait-impl | 226 |  | `SizeF32MaybeMin` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_min`, SizeAvailableSpaceMaybeMinF32) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeMinF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_min`, SizeAvailableSpaceMaybeMin) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeMin` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_max`, SizeOptionF32MaybeMaxF32) | trait-impl | 226 |  | `SizeOptionF32MaybeMaxF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_max`, SizeF32MaybeMax) | trait-impl | 226 |  | `SizeF32MaybeMax` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_max`, SizeAvailableSpaceMaybeMaxF32) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeMaxF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_max`, SizeAvailableSpaceMaybeMax) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeMax` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_clamp`, SizeOptionF32MaybeClampF32) | trait-impl | 226 |  | `SizeOptionF32MaybeClampF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_clamp`, SizeF32MaybeClamp) | trait-impl | 226 |  | `SizeF32MaybeClamp` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_clamp`, SizeAvailableSpaceMaybeClampF32) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeClampF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_clamp`, SizeAvailableSpaceMaybeClamp) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeClamp` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_add`, SizeOptionF32MaybeAddF32) | trait-impl | 226 |  | `SizeOptionF32MaybeAddF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_add`, SizeF32MaybeAdd) | trait-impl | 226 |  | `SizeF32MaybeAdd` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_add`, SizeAvailableSpaceMaybeAddF32) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeAddF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_add`, SizeAvailableSpaceMaybeAdd) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeAdd` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_sub`, SizeOptionF32MaybeSubF32) | trait-impl | 226 |  | `SizeOptionF32MaybeSubF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_sub`, SizeF32MaybeSub) | trait-impl | 226 |  | `SizeF32MaybeSub` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_sub`, SizeAvailableSpaceMaybeSubF32) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeSubF32` | done |
| `impl MaybeMath<Size<In>, Size<Out>> for Size<T>` (`maybe_sub`, SizeAvailableSpaceMaybeSub) | trait-impl | 226 |  | `SizeAvailableSpaceMaybeSub` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `lhs_option_f32_rhs_option_f32::test_maybe_min` | test | 257 |  | `TestUtilMath_LhsOptionF32RhsOptionF32_MaybeMin` | done |
| `lhs_option_f32_rhs_option_f32::test_maybe_max` | test | 266 |  | `TestUtilMath_LhsOptionF32RhsOptionF32_MaybeMax` | done |
| `lhs_option_f32_rhs_option_f32::test_maybe_add` | test | 275 |  | `TestUtilMath_LhsOptionF32RhsOptionF32_MaybeAdd` | done |
| `lhs_option_f32_rhs_option_f32::test_maybe_sub` | test | 284 |  | `TestUtilMath_LhsOptionF32RhsOptionF32_MaybeSub` | done |
| `lhs_option_f32_rhs_f32::test_maybe_min` | test | 297 |  | `TestUtilMath_LhsOptionF32RhsF32_MaybeMin` | done |
| `lhs_option_f32_rhs_f32::test_maybe_max` | test | 304 |  | `TestUtilMath_LhsOptionF32RhsF32_MaybeMax` | done |
| `lhs_option_f32_rhs_f32::test_maybe_add` | test | 311 |  | `TestUtilMath_LhsOptionF32RhsF32_MaybeAdd` | done |
| `lhs_option_f32_rhs_f32::test_maybe_sub` | test | 318 |  | `TestUtilMath_LhsOptionF32RhsF32_MaybeSub` | done |
| `lhs_f32_rhs_option_f32::test_maybe_min` | test | 329 |  | `TestUtilMath_LhsF32RhsOptionF32_MaybeMin` | done |
| `lhs_f32_rhs_option_f32::test_maybe_max` | test | 336 |  | `TestUtilMath_LhsF32RhsOptionF32_MaybeMax` | done |
| `lhs_f32_rhs_option_f32::test_maybe_add` | test | 343 |  | `TestUtilMath_LhsF32RhsOptionF32_MaybeAdd` | done |
| `lhs_f32_rhs_option_f32::test_maybe_sub` | test | 350 |  | `TestUtilMath_LhsF32RhsOptionF32_MaybeSub` | done |

## src/util/mod.rs

Go target: `internal/taffy/util_mod.go` (package `taffy`), tests in `util_mod_test.go`

No public items or tests.

## src/util/parse.rs

Go target: `internal/taffy/util_parse.go` (package `taffy`), tests in `util_parse_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ParseError` | struct | 15 |  | `ParseError` | n/a:D-TAFFY-FEATURE |
| `ParseResult` | type | 36 |  | `ParseResult` | n/a:D-TAFFY-FEATURE |

## src/util/print.rs

Go target: `internal/taffy/util_print.go` (package `taffy`), tests in `util_print_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `print_tree` | fn | 6 |  | `PrintTreeStdout` | dev:D-TAFFY-TREETRAITS |
| `write_tree` | fn | 14 |  | `WriteTree` | done |

## src/util/resolve.rs

Go target: `internal/taffy/util_resolve.go` (package `taffy`), tests in `util_resolve_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MaybeResolve` | trait | 13 |  | `MaybeResolve` | done |
| `MaybeResolve::maybe_resolve` | trait-method | 15 |  | `MaybeResolve.MaybeResolve` | done |
| `ResolveOrZero` | trait | 23 |  | `ResolveOrZero` | done |
| `ResolveOrZero::resolve_or_zero` | trait-method | 25 |  | `ResolveOrZero.ResolveOrZero` | done |
| `impl MaybeResolve<Option<f32>, Option<f32>> for LengthPercentage` / `LengthPercentageAuto` / `Dimension` | trait-impl | 28 |  | `(LengthPercentage).MaybeResolve` | done |
| `impl MaybeResolve<Option<f32>, Option<f32>> for LengthPercentage` / `LengthPercentageAuto` / `Dimension` | trait-impl | 28 |  | `(LengthPercentageAuto).MaybeResolve` | done |
| `impl MaybeResolve<Option<f32>, Option<f32>> for LengthPercentage` / `LengthPercentageAuto` / `Dimension` | trait-impl | 28 |  | `(Dimension).MaybeResolve` | done |
| `impl MaybeResolve<f32, Option<f32>> for T` | trait-impl | 81 |  | `MaybeResolveF32` | dev:D-TAFFY-RESOLVE |
| `impl MaybeResolve<Size<In>, Size<Out>> for Size<T>` | trait-impl | 90 |  | `SizeMaybeResolve` | dev:D-TAFFY-RESOLVE |
| `impl MaybeResolve<Size<In>, Size<Out>> for Size<T>` | trait-impl | 90 |  | `SizeMaybeResolveF32` | dev:D-TAFFY-RESOLVE |
| `impl ResolveOrZero<Option<f32>, f32> for LengthPercentage` / `LengthPercentageAuto` / `Dimension` | trait-impl | 102 |  | `(LengthPercentage).ResolveOrZero` | done |
| `impl ResolveOrZero<Option<f32>, f32> for LengthPercentage` / `LengthPercentageAuto` / `Dimension` | trait-impl | 102 |  | `(LengthPercentageAuto).ResolveOrZero` | done |
| `impl ResolveOrZero<Option<f32>, f32> for LengthPercentage` / `LengthPercentageAuto` / `Dimension` | trait-impl | 102 |  | `(Dimension).ResolveOrZero` | done |
| `impl ResolveOrZero<Size<In>, Size<Out>> for Size<T>` | trait-impl | 124 |  | `SizeResolveOrZero` | dev:D-TAFFY-RESOLVE |
| `impl ResolveOrZero<Size<In>, Rect<Out>> for Rect<T>` | trait-impl | 136 |  | `RectResolveOrZero` | dev:D-TAFFY-RESOLVE |
| `impl ResolveOrZero<Option<f32>, Rect<Out>> for Rect<T>` | trait-impl | 149 |  | `RectResolveOrZeroOptionF32` | dev:D-TAFFY-RESOLVE |
| `impl ResolveOrZero<f32, Rect<Out>> for Rect<T>` | trait-impl | 163 |  | `RectResolveOrZeroF32` | dev:D-TAFFY-RESOLVE |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `maybe_resolve_dimension::resolve_auto` | test | 205 |  | `TestUtilResolve_MaybeResolveDimension_ResolveAuto` | done |
| `maybe_resolve_dimension::resolve_length` | test | 217 |  | `TestUtilResolve_MaybeResolveDimension_ResolveLength` | done |
| `maybe_resolve_dimension::resolve_percent` | test | 230 |  | `TestUtilResolve_MaybeResolveDimension_ResolvePercent` | done |
| `maybe_resolve_size_dimension::maybe_resolve_auto` | test | 247 |  | `TestUtilResolve_MaybeResolveSizeDimension_MaybeResolveAuto` | done |
| `maybe_resolve_size_dimension::maybe_resolve_length` | test | 259 |  | `TestUtilResolve_MaybeResolveSizeDimension_MaybeResolveLength` | done |
| `maybe_resolve_size_dimension::maybe_resolve_percent` | test | 272 |  | `TestUtilResolve_MaybeResolveSizeDimension_MaybeResolvePercent` | done |
| `resolve_or_zero_dimension_to_option_f32::resolve_or_zero_auto` | test | 286 |  | `TestUtilResolve_ResolveOrZeroDimensionToOptionF32_ResolveOrZeroAuto` | done |
| `resolve_or_zero_dimension_to_option_f32::resolve_or_zero_length` | test | 293 |  | `TestUtilResolve_ResolveOrZeroDimensionToOptionF32_ResolveOrZeroLength` | done |
| `resolve_or_zero_dimension_to_option_f32::resolve_or_zero_percent` | test | 300 |  | `TestUtilResolve_ResolveOrZeroDimensionToOptionF32_ResolveOrZeroPercent` | done |
| `resolve_or_zero_rect_dimension_to_rect::resolve_or_zero_auto` | test | 314 |  | `TestUtilResolve_ResolveOrZeroRectDimensionToRect_ResolveOrZeroAuto` | done |
| `resolve_or_zero_rect_dimension_to_rect::resolve_or_zero_length` | test | 322 |  | `TestUtilResolve_ResolveOrZeroRectDimensionToRect_ResolveOrZeroLength` | done |
| `resolve_or_zero_rect_dimension_to_rect::resolve_or_zero_percent` | test | 330 |  | `TestUtilResolve_ResolveOrZeroRectDimensionToRect_ResolveOrZeroPercent` | done |
| `resolve_or_zero_rect_dimension_to_rect_f32_via_option::resolve_or_zero_auto` | test | 348 |  | `TestUtilResolve_ResolveOrZeroRectDimensionToRectF32ViaOption_ResolveOrZeroAuto` | done |
| `resolve_or_zero_rect_dimension_to_rect_f32_via_option::resolve_or_zero_length` | test | 356 |  | `TestUtilResolve_ResolveOrZeroRectDimensionToRectF32ViaOption_ResolveOrZeroLength` | done |
| `resolve_or_zero_rect_dimension_to_rect_f32_via_option::resolve_or_zero_percent` | test | 364 |  | `TestUtilResolve_ResolveOrZeroRectDimensionToRectF32ViaOption_ResolveOrZeroPercent` | done |

## src/util/sys.rs

Go target: `internal/taffy/util_sys.go` (package `taffy`), tests in `util_sys_test.go`

No `pub` items. The `std` branch is ported as unexported helpers: `round`, `ceil`, `floor`, `abs`, `f32Max`, `f32Min` (Rust NaN semantics), `newVecWithCapacity`, `newConstChildrenVec`, and `defaultCheapStr = string`. `String` is `string`, `Map` is a Go map, `Vec`, `Box`, `ChildrenVec` and `GridTrackVec` are slices or plain values. The `alloc` branch and the `polyfill` module are for `not(feature = "std")` (n/a:D-TAFFY-FEATURE). util_sys.go also declares `Option`, the alias of `gpuiutil.Option`, with `some` / `none`.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `std::String` (`pub(crate)`) | type | 15 | std | - | dev:D-TAFFY-SYS |
| `std::DefaultCheapStr` (`pub(crate)`) | type | 17 | std | `defaultCheapStr` | done |
| `std::Map` (`pub(crate)`) | type | 19 | std | - | dev:D-TAFFY-SYS |
| `std::Vec` (`pub(crate)`) | type | 21 | std | - | dev:D-TAFFY-SYS |
| `std::Box` (`pub(crate)`) | type | 23 | std | - | dev:D-TAFFY-SYS |
| `std::ChildrenVec` (`pub(crate)`) | type | 25 | std | - | dev:D-TAFFY-SYS |
| `std::GridTrackVec` (`pub(crate)`) | type | 28 | std, grid | - | dev:D-TAFFY-SYS |
| `std::new_vec_with_capacity` (`pub(crate)`) | fn | 32 | std | `newVecWithCapacity` | done |
| `std::new_const_children_vec` (`pub(crate)`) | fn | 38 | std | `newConstChildrenVec` | done |
| `std::round` (`pub(crate)`) | fn | 45 | std | `round` | done |
| `std::ceil` (`pub(crate)`) | fn | 52 | std | `ceil` | done |
| `std::floor` (`pub(crate)`) | fn | 59 | std | `floor` | done |
| `std::abs` (`pub(crate)`) | fn | 66 | std | `abs` | done |
| `std::f32_max` (`pub(crate)`) | fn | 72 | std | `f32Max` | done |
| `std::f32_min` (`pub(crate)`) | fn | 78 | std | `f32Min` | done |
| `alloc` module (`not(feature = "std")`) | mod | 85 | std | - | n/a:D-TAFFY-FEATURE |
| `polyfill` module (`not(feature = "std")`) | mod | 145 | std | - | n/a:D-TAFFY-FEATURE |

## tests/common/src/lib.rs

Go target: `internal/taffy/tests_common_test.go` (package `taffy`, test-only)

| Upstream test | Kind | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `new_test_tree` | fn | 6 | test | `newTestTree` | done |
| `TestNodeContext` | struct | 13 | test | `TestNodeContext` | done |
| `TestNodeContext::new` | assoc-fn | 22 | test | `newTestNodeContext` | dev:D-TAFFY-TESTNAMES |
| `TestNodeContext::zero` | assoc-fn | 27 | test | `testNodeContextZero` | dev:D-TAFFY-TESTNAMES |
| `TestNodeContext::fixed` | assoc-fn | 32 | test | `testNodeContextFixed` | dev:D-TAFFY-TESTNAMES |
| `TestNodeContext::aspect_ratio` | assoc-fn | 37 | test | `testNodeContextAspectRatio` | dev:D-TAFFY-TESTNAMES |
| `TestNodeContext::ahem_text` | assoc-fn | 43 | test | `testNodeContextAhemText` | dev:D-TAFFY-TESTNAMES |
| `TestMeasureData` | enum | 51 | test | `TestMeasureData` | done |
| `test_measure_function` | fn | 63 | test | `testMeasureFunction` | done |
| `AspectRatioMeasureData` | struct | 97 | test | `AspectRatioMeasureData` | done |
| `WritingMode` | enum | 112 | test | `WritingMode` | done |
| `impl FromStr for WritingMode` | trait-impl | 121 | test | `WritingModeFromStr` | done |
| `AhemTextMeasureData` | struct | 135 | test | `AhemTextMeasureData` | done |

## tests/xml.rs

Go target: `internal/taffy/xml_test.go` (package `taffy`); the `parse`-feature CSS value parsers it calls through `FromStr` are ported test-only in `xml_css_test.go` (D-TAFFY-XMLPARSE); the enabled category list is `fixtures_enabled_test.go`; the fixtures are packed into `internal/taffy/testdata/xml.zip` by `internal/taffy/internal/mkfixtures` (`go generate`, sorted entries, zero timestamps, a no-op when `_ext/taffy` is absent). Each fixture is one subtest of `TestXml` named `<category>/<file>`; trees are printed only on failure.

| Upstream test | Kind | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `OutputNode` | struct | 14 | test | `outputNode` | done |
| `TrackListToken` | enum | 27 | test | `trackListToken` | done |
| `parse_track_list` | fn | 36 | test | `parseTrackList` | done |
| `track_lists_match` | fn | 68 | test | `trackListsMatch` | done |
| `impl PartialEq for OutputNode` | trait-impl | 80 | test | `(*outputNode).eq` | done |
| `OutputNode::write_metrics` | method | 107 | test | `(*outputNode).writeMetrics` | done |
| `OutputNode::write_tree` | method | 126 | test | `(*outputNode).writeTree` | done |
| `OutputNode::print_tree` | method | 152 | test | `(*outputNode).treeString` | done |
| `get_dir` | test | 168 | test | `TestXml_GetDir` | done |
| `run_xml_test` | fn | 175 | test | `runXMLTest` | done |
| `construct_tree` | fn | 241 | test | `constructTree` | done |
| `parse_or_default` / `parse_or` | fn | 284 | test | `parseOr` | done |
| `parse_self_alignment` | fn | 293 | test | `parseSelfAlignment` | done |
| `maybe_parse` | fn | 297 | test | `maybeParse` | done |
| `get_computed_expectations` | fn | 302 | test | `getComputedExpectations` | done |
| `get_resolved_track_lists` | fn | 327 | detailed_layout_info | - | n/a:D-TAFFY-FEATURE |
| `build_expectations` | fn | 342 | test | `buildExpectations` | done |
| `build_style` | fn | 365 | test | `buildStyle` | done |
| `build_style`: `flex_line_count` | field | 449 | flexbox_balance | - | n/a:D-TAFFY-FEATURE |
| `tests/xml/mod.rs` (generated `#[test]` per fixture) | test | 1 | test | `TestXml` | done |

## tests/xml fixtures

Go target: `internal/taffy/xml_test.go` (package `taffy`, test-only), enabled categories in `fixtures_enabled_test.go`.

Fixture counts per category (6504 files in tag v0.15.0). `flex-wrap: balance` fixtures (136, all in `flex`) need `flexbox_balance` and skip as n/a:D-TAFFY-FEATURE. Fixtures that reach a stub skip as blocked until the stage that owns it lands.

| Upstream test | Kind | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests/xml/leaf` (60, all pass) | fixture | - | test | `TestXml` | done |
| `tests/xml/flex` (2708: 2552 pass, 136 n/a, 20 lay out a grid box) | fixture | - | test | `TestXml` | done |
| `tests/xml/flex` fixtures that lay out a grid box (20) | fixture | - | test | `TestXml/flex` | blocked:M0-taffy-grid-sizing |
| `tests/xml/block` (1116: 1112 pass) | fixture | - | test | `TestXml` | done |
| `tests/xml/block` fixtures that lay out a grid box (4) | fixture | - | test | `TestXml/block` | blocked:M0-taffy-grid-sizing |
| `tests/xml/blockflex` (52, all pass) | fixture | - | test | `TestXml` | done |
| `tests/xml/float` (116: 108 pass) | fixture | - | test | `TestXml` | done |
| `tests/xml/float` fixtures that lay out a grid box (8) | fixture | - | test | `TestXml/float` | blocked:M0-taffy-grid-sizing |
| `tests/xml/contain` (32, all pass) | fixture | - | test | `TestXml` | done |
| `tests/xml/grid` (2336) | fixture | - | test | `TestXml/grid` | blocked:M0-taffy-grid-sizing |
| `tests/xml/gridflex` (28) | fixture | - | test | `TestXml/gridflex` | blocked:M0-taffy-grid-sizing |
| `tests/xml/blockgrid` (56) | fixture | - | test | `TestXml/blockgrid` | blocked:M0-taffy-grid-sizing |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-TAFFY-FEATURE | Code behind cargo features gpui does not enable is not ported: `calc`, `serde`, `parse`, `detailed_layout_info`, `flexbox_balance`, `debug`, `profile`, `strict_provenance` and the `not(feature = "std")` branches. A test that needs one of them (also without its own cfg, as `calc_round_trips_pointer`) is n/a. | `Cargo.toml` `[features]`, `gpui/Cargo.toml:108-116` | - |
| D-TAFFY-COMPACT | `CompactLength` is a `{tag uint8, bits uint32}` struct instead of a tagged pointer: no pointer packing, the `compat` transmutes are `math.Float32bits` / `Float32frombits`, `tag_ptr` / `untag_ptr` do not exist. Equality stays bitwise. | `src/style/compact_length.rs:10-222` | `internal/taffy/style_compact_length.go` |
| D-TAFFY-MAYBEMATH | The `MaybeMath` trait is not a Go interface: its impls for `Option<f32>` and `f32` are free functions, the two `AvailableSpace` impls are methods with an `F32` suffix for the `f32` right-hand side, and the blanket `Size<T>` impl is one function per instantiation (naming in the notes). | `src/util/math.rs:11-252` | `internal/taffy/util_math.go` |
| D-TAFFY-SYS | The `sys` type aliases for `String`, `Map`, `Vec`, `Box`, `ChildrenVec` and `GridTrackVec` are not declared; ported code uses `string`, Go maps and slices directly | `src/util/sys.rs:15-28` | - |
| D-TAFFY-TRAITS | Style traits are Go interfaces. Default method bodies live in embeddable `<Trait>Defaults` structs (`CoreStyleDefaults`, `FlexboxItemStyleDefaults`, ...); `CoreStyle::is_containing_block`, whose default calls `position()`, is the free function `CoreStyleDefaultIsContainingBlock(s)` that implementors call. The associated type `CustomIdent` is the type parameter `S` of the interfaces that use it (`OofItemStyle[S]`); `CheapCloneStr` is the constraint `~string`. `Style`'s fields share names with the trait methods, which Go forbids on one type, so `impl ... for Style` is implemented by `StyleRef[S]` (`(*Style[S]).Ref()`, upstream's `&Style`); `PhantomData` field `dummy` and the `impl for &T` forwarding impls do not exist. | `src/style/mod.rs:63-250,880-1600`, `src/style/flex.rs`, `src/style/block.rs` | `internal/taffy/style_mod.go`, `style_flex.go`, `style_block.go` |
| D-TAFFY-ALIGNFIELDS | `AlignItems` / `AlignContent` keep their `keyword` / `safety` fields unexported because the `keyword()` method would collide with an exported field; read them with `Keyword()` / `Safety()`, build with `NewAlignItems` / `NewAlignContent` or the `AlignItems<Name>` vars. Upstream's associated consts are package vars (Go has no struct consts). | `src/style/alignment.rs:155-460` | `internal/taffy/style_alignment.go` |
| D-TAFFY-SIZES | `style_sizes` checks only sizes that Go lays out like Rust; `Option<AlignItems>` (Go 3 bytes, no niche), the string-dependent grid types and `Style` sizes differ and are not asserted. | `src/style/mod.rs:1742-1815` | `internal/taffy/style_mod_test.go` |
| D-TAFFY-GRIDLISTS | The associated iterator types of `GridContainerStyle` and `GenericRepetition` are slices: template track lists are `[]GenericGridTemplateComponent[S, GenericRepetition[S]]` (the repetition is the interface, so `R` is not a trait parameter), auto track lists `[]TrackSizingFunction`, line names `TemplateLineNames[S]` (alias of `[][]S`), areas `[]GridTemplateArea[S]`. `StyleRef.GridTemplateRows` / `GridTemplateColumns` allocate the component-ref slice per call where upstream maps lazily. | `src/style/grid.rs:81-270`, `src/style/mod.rs:1386-1480` | `internal/taffy/style_grid.go`, `style_mod.go` |
| D-TAFFY-HELPERS | `style_helpers`: `line` is `NewLine` (Go cannot have a function and the type `Line` with one name). Go cannot add methods to `float32` or `gpuiutil.Option`, so the `f32` / `Option` impls are runtime cases inside the Point / Line / Size / Rect helpers (`PointZero[float32]`, `SizeLength[Option[float32]]`) and are not usable with the top-level generic helpers; `Option<T>` is covered for `T = f32` only. `repeat`'s `TryInto<RepetitionCount>` input is the union of `uint16`, `string` and `RepetitionCount`. | `src/style_helpers.rs` | `internal/taffy/style_helpers.go` |
| D-TAFFY-TREETRAITS | Tree traits are Go interfaces with the conventions of D-TAFFY-TRAITS: associated style types are the style interfaces (`CoreStyle`, `OofItemStyle[S]`, ...), `CustomIdent` is the type parameter `S` of `LayoutContainingBlock` / `LayoutGridContainer` (`LayoutPartialTree` is not generic because `CoreStyle` is not), `ChildIter` is `iter.Seq[NodeID]`, `*const ()` is `unsafe.Pointer`, `BlockContext<'_>` is `*BlockContext`, and default bodies live in `<Trait>Defaults` structs or, when they call another trait method, in `<Trait>Default<Method>` functions. The marker trait `TraverseTree` has the marker method `TraverseTree()`. `get_detailed_layout_info` returns the value, not a reference. `print_tree` is `PrintTreeStdout`, because the trait `PrintTree` takes the name. `LayoutPartialTreeExt` methods are unexported free functions taking the tree. | `src/tree/traits.rs`, `src/util/print.rs` | `internal/taffy/tree_traits.go`, `util_print.go` |
| D-TAFFY-CLONE | `OofCandidates` holds its lazily allocated list behind a pointer (nil is upstream's `None`, a non-nil pointer to an empty slice is `Some(empty)`), so Go value copies of `OofCandidates`, `LayoutOutput` and `Cache` share it; ported code calls `Clone()` where upstream clones (`Cache::get` does) and `Equal` where upstream compares. The debug-build `MODE_CHANGE_EVICTIONS` thread-local is one process-wide atomic counter, counted while `gpuiutil.DebugAssertions` is set. | `src/tree/layout.rs:355-574`, `src/tree/cache.rs:180-360` | `internal/taffy/tree_layout.go`, `tree_cache.go` |
| D-TAFFY-RESOLVE | `MaybeResolve` / `ResolveOrZero` are generic interfaces implemented by `LengthPercentage`, `LengthPercentageAuto` and `Dimension` for an `Option<f32>` context; the blanket `f32`-context impl is `MaybeResolveF32`, and the `Size` / `Rect` impls are free generic functions (`SizeMaybeResolve`, `SizeMaybeResolveF32`, `SizeResolveOrZero`, `RectResolveOrZero`, `RectResolveOrZeroOptionF32`, `RectResolveOrZeroF32`). The `calc` closure is `func(unsafe.Pointer, float32) float32`. | `src/util/resolve.rs` | `internal/taffy/util_resolve.go` |
| D-TAFFY-RESULT | `TaffyResult<T>` has no Go type: TaffyTree methods return `(T, error)` (or `error` for `()`), the error being a `TaffyError` value comparable with `==`; references (`&Style`, `&Layout`, `&[NodeId]`) are pointers / slices into the tree. `get_node_context(_mut)` return a nil-able pointer, `get_disjoint_node_context_mut` takes variadic ids and returns `([]*C, bool)`. | `src/tree/taffy_tree.rs:38-58,698-731` | `internal/taffy/tree_taffy_tree.go` |
| D-TAFFY-CTXDROP | Go has no destructors: removing or clearing nodes drops the tree's reference to their contexts, which `remove_drops_the_nodes_context` / `clear_drops_every_nodes_context` check as an empty `node_context_data` instead of a drop counter. | `src/tree/taffy_tree.rs:1502-1542` | `internal/taffy/tree_taffy_tree_test.go` |
| D-TAFFY-RANGE | `remove_children_range`'s `RangeBounds<usize>` is the half-open pair `start, end` (`1..=2` is `1, 3`); invalid ranges panic with core's slice-range messages. | `src/tree/taffy_tree.rs:818-829` | `internal/taffy/tree_taffy_tree.go` |
| D-TAFFY-XMLPARSE | The `parse` feature is not ported (D-TAFFY-FEATURE), but `tests/xml.rs` reads every style through `FromStr`; the `FromCss` impls it reaches (keyword enums, alignment, lengths, `AvailableSpace`, `Contain`, grid placement / tracks / repetitions / auto-flow) are ported test-only on a minimal cssparser-like tokenizer in `xml_css_test.go`. Numbers parse as f64 then convert to f32, as cssparser does. | `src/util/parse.rs`, `src/style/*.rs` `FromCss` impls | `internal/taffy/xml_css_test.go` |
| D-TAFFY-TESTNAMES | Test helper constructors of `tests/common` are unexported (`testNodeContextZero`, ...) because `go vet` rejects `Test*` functions that are not tests. | `tests/common/src/lib.rs:21-46` | `internal/taffy/tests_common_test.go` |
