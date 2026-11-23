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
- style_helpers traits with an associated const (`TaffyZero::ZERO`, `TaffyAuto::AUTO`, `TaffyMinContent::MIN_CONTENT`, `TaffyMaxContent::MAX_CONTENT`) are generic interfaces with one method on a value receiver (`Zero() T`, `Auto() T`, `MinContent() T`, `MaxContent() T`), so a generic helper calls it on the zero value (`var z T; z.Zero()`). `FromLength`, `FromPercent`, `FromFr` take `float64` (upstream `Input: Into<f64>`), `TaffyFitContent` takes a `LengthPercentage`. The const value itself is a package var `<Type><Const>` (`CompactLengthZero`, `DimensionZero`, `AvailableSpaceZero`) unless an associated fn of the same name exists (`Dimension::auto()` -> `DimensionAuto()`, which the `AUTO` const reuses). Until M0-taffy-style-grid ports style_helpers.rs the interfaces live in stubs.go.
- MaybeMath (`D-TAFFY-MAYBEMATH`): Go cannot add methods to `Option[float32]` or `float32`, and `AvailableSpace` has two impls. The impls are: lhs `Option<f32>` -> `OptionF32Maybe<Op>`, lhs `f32` -> `F32Maybe<Op>`, lhs `AvailableSpace` -> method `Maybe<Op>`; an `f32` right-hand side adds the suffix `F32` (an `Option<f32>` right-hand side adds nothing); the blanket `impl for Size<T>` is `Size<Lhs>Maybe<Op>[F32]` (`SizeOptionF32MaybeSub`, `SizeF32MaybeMax`, `SizeAvailableSpaceMaybeMinF32`). `x.maybe_min(y)` with `x: Option<f32>, y: f32` is `OptionF32MaybeMinF32(x, y)`.
- Floating point: arithmetic stays in `float32`, in upstream order. Rust's `f32::min` / `max` return the non-NaN operand, Go's `min` / `max` / `math.Min` propagate NaN, so every `.min()` / `.max()` uses `f32Min` / `f32Max` (util_sys.go); never the builtins. `sys::round` is `floor(v + 0.5)` with the add in `float32`, not `math.Round`. The Go spec lets the compiler fuse a multiplication with a later addition into an FMA, even across statements and through inlined calls (arm64 does; Rust never does), and only an explicit conversion stops it: every float32 multiplication is written `float32(a * b)`. Signed-zero results of `min(0.0, -0.0)` may differ from LLVM's `minnum`; layouts never depend on them.
- Panics keep upstream's text: `called `Option::unwrap()` on a `None` value`, and `unreachable!(msg)` as `internal error: entered unreachable code: msg`.
- Forward references: a stage that needs a type or function from a later stage declares it in internal/taffy/stubs.go with its final signature and a `panic("taffy: unimplemented <symbol>")` body; the owning stage moves it out. Tests never reach a stub.

## src/compute/block.rs

Go target: `internal/taffy/compute_block.go` (package `taffy`), tests in `compute_block_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_block_align_content_offset` | fn | 35 |  | `ComputeBlockAlignContentOffset` | todo |
| `BlockFormattingContext` | struct | 41 |  | `BlockFormattingContext` | todo |
| `BlockFormattingContext::new` | assoc-fn | 58 |  | `NewBlockFormattingContext` | todo |
| `BlockFormattingContext::root_block_context` | method | 63 |  | `(BlockFormattingContext).RootBlockContext` | todo |
| `BlockContext` | struct | 82 |  | `BlockContext` | todo |
| `BlockContext::sub_context` | method | 112 |  | `(BlockContext).SubContext` | todo |
| `BlockContext::is_bfc_root` | method | 131 |  | `(BlockContext).IsBfcRoot` | todo |
| `BlockContext::set_width` | method | 143 | float_layout | `(BlockContext).SetWidth` | todo |
| `BlockContext::apply_content_box_inset` | method | 149 | float_layout | `(BlockContext).ApplyContentBoxInset` | todo |
| `BlockContext::has_floats` | method | 156 | float_layout | `(BlockContext).HasFloats` | todo |
| `BlockContext::has_active_floats` | method | 162 | float_layout | `(BlockContext).HasActiveFloats` | todo |
| `BlockContext::place_floated_box` | method | 167 | float_layout | `(BlockContext).PlaceFloatedBox` | todo |
| `BlockContext::find_content_slot` | method | 194 | float_layout | `(BlockContext).FindContentSlot` | todo |
| `BlockContext::find_bfc_slot` | method | 204 | float_layout | `(BlockContext).FindBfcSlot` | todo |
| `BlockContext::cleared_threshold` | method | 226 | float_layout | `(BlockContext).ClearedThreshold` | todo |
| `BlockContext::has_adjoining_float` | method | 232 | float_layout | `(BlockContext).HasAdjoiningFloat` | todo |
| `BlockContext::add_child_floated_content_height_contribution` | method | 265 | float_layout | `(BlockContext).AddChildFloatedContentHeightContribution` | todo |
| `BlockContext::floated_content_height_contribution` | method | 270 | float_layout | `(BlockContext).FloatedContentHeightContribution` | todo |
| `compute_block_layout` | fn | 374 |  | `ComputeBlockLayout` | todo |

## src/compute/common/alignment.rs

Go target: `internal/taffy/compute_common_alignment.go` (package `taffy`), tests in `compute_common_alignment_test.go`

No public items or tests.

## src/compute/common/mod.rs

Go target: `internal/taffy/compute_common_mod.go` (package `taffy`), tests in `compute_common_mod_test.go`

No public items or tests.

## src/compute/common/scrollable_overflow.rs

Go target: `internal/taffy/compute_common_scrollable_overflow.go` (package `taffy`), tests in `compute_common_scrollable_overflow_test.go`

No public items or tests.

## src/compute/common/sizing_keyword.rs

Go target: `internal/taffy/compute_common_sizing_keyword.go` (package `taffy`), tests in `compute_common_sizing_keyword_test.go`

No public items or tests.

## src/compute/flexbox.rs

Go target: `internal/taffy/compute_flexbox.go` (package `taffy`), tests in `compute_flexbox_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_flexbox_layout` | fn | 234 |  | `ComputeFlexboxLayout` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `zero_sized_item_rule` | test | 3399 | flexbox_balance | `TestComputeFlexbox_ZeroSizedItemRule` | todo |
| `matches_naive_dp` | test | 3414 | flexbox_balance | `TestComputeFlexbox_MatchesNaiveDp` | todo |

## src/compute/float.rs

Go target: `internal/taffy/compute_float.go` (package `taffy`), tests in `compute_float_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ContentSlot` | struct | 41 |  | `ContentSlot` | todo |
| `BfcSlot` | struct | 60 |  | `BfcSlot` | todo |
| `PlacedFloatedBox` | struct | 79 |  | `PlacedFloatedBox` | todo |
| `FloatContext` | struct | 211 |  | `FloatContext` | todo |
| `FloatContext::new` | assoc-fn | 256 |  | `NewFloatContext` | todo |
| `FloatContext::has_floats` | method | 262 |  | `(FloatContext).HasFloats` | todo |
| `FloatContext::has_active_floats` | method | 268 |  | `(FloatContext).HasActiveFloats` | todo |
| `FloatContext::set_width` | method | 273 |  | `(FloatContext).SetWidth` | todo |
| `FloatContext::left_floats` | method | 278 |  | `(FloatContext).LeftFloats` | todo |
| `FloatContext::right_floats` | method | 283 |  | `(FloatContext).RightFloats` | todo |
| `FloatContext::place_floated_box` | method | 311 |  | `(FloatContext).PlaceFloatedBox` | todo |
| `FloatContext::cleared_threshold` | method | 581 |  | `(FloatContext).ClearedThreshold` | todo |
| `FloatContext::find_content_slot` | method | 594 |  | `(FloatContext).FindContentSlot` | todo |
| `FloatContext::find_bfc_slot` | method | 667 |  | `(FloatContext).FindBfcSlot` | todo |
| `FloatIntrinsicWidthCalculator` | struct | 747 |  | `FloatIntrinsicWidthCalculator` | todo |
| `FloatIntrinsicWidthCalculator::new` | assoc-fn | 760 |  | `NewFloatIntrinsicWidthCalculator` | todo |
| `FloatIntrinsicWidthCalculator::add_float` | method | 765 |  | `(FloatIntrinsicWidthCalculator).AddFloat` | todo |
| `FloatIntrinsicWidthCalculator::result` | method | 788 |  | `(FloatIntrinsicWidthCalculator).Result` | todo |

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
| `compute_leaf_layout` | fn | 20 |  | `ComputeLeafLayout` | todo |

## src/compute/mod.rs

Go target: `internal/taffy/compute_mod.go` (package `taffy`), tests in `compute_mod_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compute_root_layout` | fn | 77 |  | `ComputeRootLayout` | todo |
| `compute_cached_layout` | fn | 330 |  | `ComputeCachedLayout` | todo |
| `round_layout` | fn | 375 |  | `RoundLayout` | todo |
| `compute_hidden_layout` | fn | 456 |  | `ComputeHiddenLayout` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `hidden_layout_should_hide_recursively` | test | 486 |  | `TestComputeMod_HiddenLayoutShouldHideRecursively` | todo |

## src/compute/oof.rs

Go target: `internal/taffy/compute_oof.go` (package `taffy`), tests in `compute_oof_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `resolve_static_offset` | fn | 35 |  | `ResolveStaticOffset` | todo |
| `compute_oof_layout` | fn | 348 |  | `ComputeOofLayout` | todo |
| `OofLayoutResult` | struct | 370 |  | `OofLayoutResult` | todo |
| `compute_oof_layout_for_area` | fn | 389 |  | `ComputeOofLayoutForArea` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `start_alignment` | test | 865 |  | `TestComputeOof_StartAlignment` | todo |
| `end_alignment` | test | 872 |  | `TestComputeOof_EndAlignment` | todo |
| `center_alignment` | test | 879 |  | `TestComputeOof_CenterAlignment` | todo |
| `safe_alignment_falls_back_only_on_overflow` | test | 887 |  | `TestComputeOof_SafeAlignmentFallsBackOnlyOnOverflow` | todo |
| `unsafe_alignment_never_falls_back` | test | 899 |  | `TestComputeOof_UnsafeAlignmentNeverFallsBack` | todo |
| `degenerate_area` | test | 905 |  | `TestComputeOof_DegenerateArea` | todo |

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
| `AlignItemsKeyword` | enum | 24 |  | `AlignItemsKeyword` | todo |
| `AlignContentKeyword` | enum | 69 |  | `AlignContentKeyword` | todo |
| `AlignmentSafety` | enum | 135 |  | `AlignmentSafety` | todo |
| `AlignItems` | struct | 155 |  | `AlignItems` | todo |
| `AlignItems::NORMAL` | assoc-const | 164 |  | `AlignItemsNormal` | todo |
| `AlignItems::START` | assoc-const | 166 |  | `AlignItemsStart` | todo |
| `AlignItems::END` | assoc-const | 168 |  | `AlignItemsEnd` | todo |
| `AlignItems::FLEX_START` | assoc-const | 170 |  | `AlignItemsFlexStart` | todo |
| `AlignItems::FLEX_END` | assoc-const | 172 |  | `AlignItemsFlexEnd` | todo |
| `AlignItems::SELF_START` | assoc-const | 174 |  | `AlignItemsSelfStart` | todo |
| `AlignItems::SELF_END` | assoc-const | 176 |  | `AlignItemsSelfEnd` | todo |
| `AlignItems::CENTER` | assoc-const | 178 |  | `AlignItemsCenter` | todo |
| `AlignItems::BASELINE` | assoc-const | 180 |  | `AlignItemsBaseline` | todo |
| `AlignItems::STRETCH` | assoc-const | 182 |  | `AlignItemsStretch` | todo |
| `AlignItems::SAFE_START` | assoc-const | 185 |  | `AlignItemsSafeStart` | todo |
| `AlignItems::SAFE_END` | assoc-const | 188 |  | `AlignItemsSafeEnd` | todo |
| `AlignItems::SAFE_FLEX_START` | assoc-const | 191 |  | `AlignItemsSafeFlexStart` | todo |
| `AlignItems::SAFE_FLEX_END` | assoc-const | 194 |  | `AlignItemsSafeFlexEnd` | todo |
| `AlignItems::SAFE_CENTER` | assoc-const | 197 |  | `AlignItemsSafeCenter` | todo |
| `AlignItems::SAFE_SELF_START` | assoc-const | 200 |  | `AlignItemsSafeSelfStart` | todo |
| `AlignItems::SAFE_SELF_END` | assoc-const | 203 |  | `AlignItemsSafeSelfEnd` | todo |
| `AlignItems::is_safe` | method | 207 |  | `(AlignItems).IsSafe` | todo |
| `AlignItems::with_safety` | method | 213 |  | `(AlignItems).WithSafety` | todo |
| `AlignItems::keyword` | method | 219 |  | `(AlignItems).Keyword` | todo |
| `AlignItems::is_stretch_or_normal` | method | 226 |  | `(AlignItems).IsStretchOrNormal` | todo |
| `AlignItems::resolve_self_relative` | method | 241 |  | `(AlignItems).ResolveSelfRelative` | todo |
| `JustifyItems` | type | 324 |  | `JustifyItems` | todo |
| `AlignSelf` | type | 332 |  | `AlignSelf` | todo |
| `JustifySelf` | type | 340 |  | `JustifySelf` | todo |
| `AlignContent` | struct | 348 |  | `AlignContent` | todo |
| `AlignContent::NORMAL` | assoc-const | 357 |  | `AlignContentNormal` | todo |
| `AlignContent::START` | assoc-const | 359 |  | `AlignContentStart` | todo |
| `AlignContent::END` | assoc-const | 361 |  | `AlignContentEnd` | todo |
| `AlignContent::FLEX_START` | assoc-const | 363 |  | `AlignContentFlexStart` | todo |
| `AlignContent::FLEX_END` | assoc-const | 365 |  | `AlignContentFlexEnd` | todo |
| `AlignContent::CENTER` | assoc-const | 367 |  | `AlignContentCenter` | todo |
| `AlignContent::STRETCH` | assoc-const | 369 |  | `AlignContentStretch` | todo |
| `AlignContent::SPACE_BETWEEN` | assoc-const | 371 |  | `AlignContentSpaceBetween` | todo |
| `AlignContent::SPACE_EVENLY` | assoc-const | 374 |  | `AlignContentSpaceEvenly` | todo |
| `AlignContent::SPACE_AROUND` | assoc-const | 376 |  | `AlignContentSpaceAround` | todo |
| `AlignContent::SAFE_START` | assoc-const | 379 |  | `AlignContentSafeStart` | todo |
| `AlignContent::SAFE_END` | assoc-const | 382 |  | `AlignContentSafeEnd` | todo |
| `AlignContent::SAFE_FLEX_START` | assoc-const | 385 |  | `AlignContentSafeFlexStart` | todo |
| `AlignContent::SAFE_FLEX_END` | assoc-const | 388 |  | `AlignContentSafeFlexEnd` | todo |
| `AlignContent::SAFE_CENTER` | assoc-const | 391 |  | `AlignContentSafeCenter` | todo |
| `AlignContent::is_safe` | method | 395 |  | `(AlignContent).IsSafe` | todo |
| `AlignContent::with_safety` | method | 401 |  | `(AlignContent).WithSafety` | todo |
| `AlignContent::keyword` | method | 407 |  | `(AlignContent).Keyword` | todo |
| `JustifyContent` | type | 463 |  | `JustifyContent` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `align_types_within_size_budget` | test | 679 |  | `TestStyleAlignment_AlignTypesWithinSizeBudget` | todo |
| `align_items_is_safe` | test | 686 |  | `TestStyleAlignment_AlignItemsIsSafe` | todo |
| `align_items_keyword_strips_safe` | test | 707 |  | `TestStyleAlignment_AlignItemsKeywordStripsSafe` | todo |
| `align_items_keyword_passthrough` | test | 718 |  | `TestStyleAlignment_AlignItemsKeywordPassthrough` | todo |
| `align_items_default_is_normal` | test | 727 |  | `TestStyleAlignment_AlignItemsDefaultIsNormal` | todo |
| `resolve_self_relative_inline_axis` | test | 734 |  | `TestStyleAlignment_ResolveSelfRelativeInlineAxis` | todo |
| `resolve_self_relative_block_axis` | test | 755 |  | `TestStyleAlignment_ResolveSelfRelativeBlockAxis` | todo |
| `align_content_is_safe` | test | 763 |  | `TestStyleAlignment_AlignContentIsSafe` | todo |
| `align_content_keyword_strips_safe` | test | 772 |  | `TestStyleAlignment_AlignContentKeywordStripsSafe` | todo |
| `align_content_keyword_reversed_swaps_start_end` | test | 782 |  | `TestStyleAlignment_AlignContentKeywordReversedSwapsStartEnd` | todo |
| `parse_align_items_plain` | test | 798 | parse | `TestStyleAlignment_ParseAlignItemsPlain` | todo |
| `align_content_default_is_normal` | test | 811 |  | `TestStyleAlignment_AlignContentDefaultIsNormal` | todo |
| `parse_align_items_normal` | test | 818 | parse | `TestStyleAlignment_ParseAlignItemsNormal` | todo |
| `parse_align_items_safe` | test | 828 | parse | `TestStyleAlignment_ParseAlignItemsSafe` | todo |
| `parse_align_items_safe_case_insensitive` | test | 840 | parse | `TestStyleAlignment_ParseAlignItemsSafeCaseInsensitive` | todo |
| `parse_align_items_unsafe` | test | 847 | parse | `TestStyleAlignment_ParseAlignItemsUnsafe` | todo |
| `parse_align_items_rejects_invalid_safe_combos` | test | 860 | parse | `TestStyleAlignment_ParseAlignItemsRejectsInvalidSafeCombos` | todo |
| `parse_align_content_plain` | test | 876 | parse | `TestStyleAlignment_ParseAlignContentPlain` | todo |
| `parse_align_content_safe` | test | 889 | parse | `TestStyleAlignment_ParseAlignContentSafe` | todo |
| `parse_align_content_unsafe` | test | 899 | parse | `TestStyleAlignment_ParseAlignContentUnsafe` | todo |
| `parse_align_content_rejects_invalid_safe_combos` | test | 907 | parse | `TestStyleAlignment_ParseAlignContentRejectsInvalidSafeCombos` | todo |
| `serde_align_items_round_trip` | test | 922 |  | `TestStyleAlignment_SerdeAlignItemsRoundTrip` | todo |
| `serde_align_content_round_trip` | test | 953 |  | `TestStyleAlignment_SerdeAlignContentRoundTrip` | todo |

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
| `BlockContainerStyle` | trait | 6 |  | `BlockContainerStyle` | todo |
| `BlockContainerStyle::text_align` | trait-method | 9 |  | `BlockContainerStyle.TextAlign` | todo |
| `BlockContainerStyle::align_content` | trait-method | 15 |  | `BlockContainerStyle.AlignContent` | todo |
| `BlockContainerStyle::justify_items` | trait-method | 23 |  | `BlockContainerStyle.JustifyItems` | todo |
| `BlockItemStyle` | trait | 29 |  | `BlockItemStyle` | todo |
| `BlockItemStyle::is_table` | trait-method | 32 |  | `BlockItemStyle.IsTable` | todo |
| `BlockItemStyle::align_self` | trait-method | 39 |  | `BlockItemStyle.AlignSelf` | todo |
| `BlockItemStyle::justify_self` | trait-method | 46 |  | `BlockItemStyle.JustifySelf` | todo |
| `BlockItemStyle::align_content` | trait-method | 55 |  | `BlockItemStyle.AlignContent` | todo |
| `BlockItemStyle::float` | trait-method | 62 | float_layout | `BlockItemStyle.Float` | todo |
| `BlockItemStyle::clear` | trait-method | 69 | float_layout | `BlockItemStyle.Clear` | todo |
| `TextAlign` | enum | 77 |  | `TextAlign` | todo |

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
| `FlexboxContainerStyle` | trait | 6 |  | `FlexboxContainerStyle` | todo |
| `FlexboxContainerStyle::flex_direction` | trait-method | 9 |  | `FlexboxContainerStyle.FlexDirection` | todo |
| `FlexboxContainerStyle::flex_wrap` | trait-method | 14 |  | `FlexboxContainerStyle.FlexWrap` | todo |
| `FlexboxContainerStyle::flex_line_count` | trait-method | 23 | flexbox_balance | `FlexboxContainerStyle.FlexLineCount` | todo |
| `FlexboxContainerStyle::gap` | trait-method | 29 |  | `FlexboxContainerStyle.Gap` | todo |
| `FlexboxContainerStyle::align_content` | trait-method | 37 |  | `FlexboxContainerStyle.AlignContent` | todo |
| `FlexboxContainerStyle::align_items` | trait-method | 42 |  | `FlexboxContainerStyle.AlignItems` | todo |
| `FlexboxContainerStyle::justify_content` | trait-method | 47 |  | `FlexboxContainerStyle.JustifyContent` | todo |
| `FlexboxItemStyle` | trait | 53 |  | `FlexboxItemStyle` | todo |
| `FlexboxItemStyle::flex_basis` | trait-method | 56 |  | `FlexboxItemStyle.FlexBasis` | todo |
| `FlexboxItemStyle::flex_grow` | trait-method | 61 |  | `FlexboxItemStyle.FlexGrow` | todo |
| `FlexboxItemStyle::flex_shrink` | trait-method | 66 |  | `FlexboxItemStyle.FlexShrink` | todo |
| `FlexboxItemStyle::align_self` | trait-method | 73 |  | `FlexboxItemStyle.AlignSelf` | todo |
| `FlexWrap` | enum | 87 |  | `FlexWrap` | todo |
| `FlexDirection` | enum | 198 |  | `FlexDirection` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `flex_direction_is_row` | test | 270 |  | `TestStyleFlex_FlexDirectionIsRow` | todo |
| `flex_direction_is_column` | test | 278 |  | `TestStyleFlex_FlexDirectionIsColumn` | todo |
| `flex_direction_is_reverse` | test | 286 |  | `TestStyleFlex_FlexDirectionIsReverse` | todo |

## src/style/float.rs

Go target: `internal/taffy/style_float.go` (package `taffy`), tests in `style_float_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Float` | enum | 9 |  | `Float` | todo |
| `FloatDirection` | enum | 33 |  | `FloatDirection` | todo |
| `Float::is_floated` | method | 42 |  | `(Float).IsFloated` | todo |
| `Float::float_direction` | method | 47 |  | `(Float).FloatDirection` | todo |
| `Clear` | enum | 62 |  | `Clear` | todo |

## src/style/grid.rs

Go target: `internal/taffy/style_grid.go` (package `taffy`), tests in `style_grid_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `GridTemplateAreas` | struct | 25 |  | `GridTemplateAreas` | todo |
| `GridTemplateArea` | struct | 37 |  | `GridTemplateArea` | todo |
| `NamedGridLine` | struct | 54 |  | `NamedGridLine` | todo |
| `GenericRepetition` | trait | 81 |  | `GenericRepetition` | todo |
| `GenericRepetition::count` | trait-method | 94 |  | `GenericRepetition.Count` | todo |
| `GenericRepetition::tracks` | trait-method | 96 |  | `GenericRepetition.Tracks` | todo |
| `GenericRepetition::track_count` | trait-method | 98 |  | `GenericRepetition.TrackCount` | todo |
| `GenericRepetition::lines_names` | trait-method | 106 |  | `GenericRepetition.LinesNames` | todo |
| `TemplateLineNames` | trait | 112 |  | `TemplateLineNames` | todo |
| `GenericGridTemplateComponent` | enum | 130 |  | `GenericGridTemplateComponent` | todo |
| `GenericGridTemplateComponent::is_auto_repetition` | method | 147 |  | `(GenericGridTemplateComponent).IsAutoRepetition` | todo |
| `GridContainerStyle` | trait | 156 |  | `GridContainerStyle` | todo |
| `GridContainerStyle::grid_template_rows` | trait-method | 189 |  | `GridContainerStyle.GridTemplateRows` | todo |
| `GridContainerStyle::grid_template_columns` | trait-method | 191 |  | `GridContainerStyle.GridTemplateColumns` | todo |
| `GridContainerStyle::grid_auto_rows` | trait-method | 193 |  | `GridContainerStyle.GridAutoRows` | todo |
| `GridContainerStyle::grid_auto_columns` | trait-method | 195 |  | `GridContainerStyle.GridAutoColumns` | todo |
| `GridContainerStyle::grid_template_areas` | trait-method | 198 |  | `GridContainerStyle.GridTemplateAreas` | todo |
| `GridContainerStyle::grid_template_area_row_count` | trait-method | 201 |  | `GridContainerStyle.GridTemplateAreaRowCount` | todo |
| `GridContainerStyle::grid_template_area_column_count` | trait-method | 208 |  | `GridContainerStyle.GridTemplateAreaColumnCount` | todo |
| `GridContainerStyle::grid_template_column_names` | trait-method | 214 |  | `GridContainerStyle.GridTemplateColumnNames` | todo |
| `GridContainerStyle::grid_template_row_names` | trait-method | 216 |  | `GridContainerStyle.GridTemplateRowNames` | todo |
| `GridContainerStyle::grid_auto_flow` | trait-method | 220 |  | `GridContainerStyle.GridAutoFlow` | todo |
| `GridContainerStyle::gap` | trait-method | 226 |  | `GridContainerStyle.Gap` | todo |
| `GridContainerStyle::align_content` | trait-method | 234 |  | `GridContainerStyle.AlignContent` | todo |
| `GridContainerStyle::justify_content` | trait-method | 239 |  | `GridContainerStyle.JustifyContent` | todo |
| `GridContainerStyle::align_items` | trait-method | 244 |  | `GridContainerStyle.AlignItems` | todo |
| `GridContainerStyle::justify_items` | trait-method | 249 |  | `GridContainerStyle.JustifyItems` | todo |
| `GridContainerStyle::grid_template_tracks` | trait-method | 255 |  | `GridContainerStyle.GridTemplateTracks` | todo |
| `GridContainerStyle::grid_align_content` | trait-method | 265 |  | `GridContainerStyle.GridAlignContent` | todo |
| `GridItemStyle` | trait | 278 |  | `GridItemStyle` | todo |
| `GridItemStyle::grid_row` | trait-method | 281 |  | `GridItemStyle.GridRow` | todo |
| `GridItemStyle::grid_column` | trait-method | 286 |  | `GridItemStyle.GridColumn` | todo |
| `GridItemStyle::align_self` | trait-method | 293 |  | `GridItemStyle.AlignSelf` | todo |
| `GridItemStyle::justify_self` | trait-method | 299 |  | `GridItemStyle.JustifySelf` | todo |
| `GridItemStyle::grid_placement` | trait-method | 305 |  | `GridItemStyle.GridPlacement` | todo |
| `GridAutoFlow` | enum | 322 |  | `GridAutoFlow` | todo |
| `GridAutoFlow::is_dense` | method | 378 |  | `(GridAutoFlow).IsDense` | todo |
| `GridAutoFlow::primary_axis` | method | 387 |  | `(GridAutoFlow).PrimaryAxis` | todo |
| `GenericGridPlacement` | enum | 402 |  | `GenericGridPlacement` | todo |
| `GridPlacement` | enum | 426 |  | `GridPlacement` | todo |
| `GridPlacement::into_origin_zero_placement_ignoring_named` | method | 542 |  | `(GridPlacement).IntoOriginZeroPlacementIgnoringNamed` | todo |
| `Line::into_origin_zero_ignoring_named` | method | 562 |  | `(Line).IntoOriginZeroIgnoringNamed` | todo |
| `Line::indefinite_span` | method | 595 |  | `(Line).IndefiniteSpan` | todo |
| `Line::is_definite` | method | 616 |  | `(Line).IsDefinite` | todo |
| `Line<NonNamedGridPlacement>::is_definite` | method | 632 |  | `(Line).IsDefinite` | todo |
| `Line<NonNamedGridPlacement>::into_origin_zero` | method | 641 |  | `(Line).IntoOriginZero` | todo |
| `Line<OriginZeroGridPlacement>::is_definite` | method | 653 |  | `(Line).IsDefinite` | todo |
| `Line<OriginZeroGridPlacement>::resolve_definite_grid_lines` | method | 659 |  | `(Line).ResolveDefiniteGridLines` | todo |
| `Line<OriginZeroGridPlacement>::resolve_absolutely_positioned_grid_tracks` | method | 686 |  | `(Line).ResolveAbsolutelyPositionedGridTracks` | todo |
| `Line<OriginZeroGridPlacement>::resolve_indefinite_grid_tracks` | method | 706 |  | `(Line).ResolveIndefiniteGridTracks` | todo |
| `MaxTrackSizingFunction` | struct | 732 |  | `MaxTrackSizingFunction` | todo |
| `MaxTrackSizingFunction::length` | assoc-fn | 853 |  | `MaxTrackSizingFunctionLength` | todo |
| `MaxTrackSizingFunction::percent` | assoc-fn | 861 |  | `MaxTrackSizingFunctionPercent` | todo |
| `MaxTrackSizingFunction::auto` | assoc-fn | 868 |  | `MaxTrackSizingFunctionAuto` | todo |
| `MaxTrackSizingFunction::min_content` | assoc-fn | 875 |  | `MaxTrackSizingFunctionMinContent` | todo |
| `MaxTrackSizingFunction::max_content` | assoc-fn | 882 |  | `MaxTrackSizingFunctionMaxContent` | todo |
| `MaxTrackSizingFunction::fit_content_px` | assoc-fn | 896 |  | `MaxTrackSizingFunctionFitContentPx` | todo |
| `MaxTrackSizingFunction::fit_content_percent` | assoc-fn | 910 |  | `MaxTrackSizingFunctionFitContentPercent` | todo |
| `MaxTrackSizingFunction::fr` | assoc-fn | 918 |  | `MaxTrackSizingFunctionFr` | todo |
| `MaxTrackSizingFunction::calc` | assoc-fn | 928 | calc | `MaxTrackSizingFunctionCalc` | todo |
| `MaxTrackSizingFunction::fit_content_calc` | assoc-fn | 938 | calc | `MaxTrackSizingFunctionFitContentCalc` | todo |
| `MaxTrackSizingFunction::from_raw` | assoc-fn | 946 |  | `MaxTrackSizingFunctionFromRaw` | todo |
| `MaxTrackSizingFunction::into_raw` | method | 951 |  | `(MaxTrackSizingFunction).IntoRaw` | todo |
| `MaxTrackSizingFunction::is_intrinsic` | method | 957 |  | `(MaxTrackSizingFunction).IsIntrinsic` | todo |
| `MaxTrackSizingFunction::is_max_content_alike` | method | 965 |  | `(MaxTrackSizingFunction).IsMaxContentAlike` | todo |
| `MaxTrackSizingFunction::is_fr` | method | 971 |  | `(MaxTrackSizingFunction).IsFr` | todo |
| `MaxTrackSizingFunction::is_auto` | method | 977 |  | `(MaxTrackSizingFunction).IsAuto` | todo |
| `MaxTrackSizingFunction::is_min_content` | method | 983 |  | `(MaxTrackSizingFunction).IsMinContent` | todo |
| `MaxTrackSizingFunction::is_max_content` | method | 989 |  | `(MaxTrackSizingFunction).IsMaxContent` | todo |
| `MaxTrackSizingFunction::is_fit_content` | method | 995 |  | `(MaxTrackSizingFunction).IsFitContent` | todo |
| `MaxTrackSizingFunction::is_max_or_fit_content` | method | 1001 |  | `(MaxTrackSizingFunction).IsMaxOrFitContent` | todo |
| `MaxTrackSizingFunction::has_definite_value` | method | 1007 |  | `(MaxTrackSizingFunction).HasDefiniteValue` | todo |
| `MaxTrackSizingFunction::definite_value` | method | 1021 |  | `(MaxTrackSizingFunction).DefiniteValue` | todo |
| `MaxTrackSizingFunction::fit_content_limit` | method | 1039 |  | `(MaxTrackSizingFunction).FitContentLimit` | todo |
| `MaxTrackSizingFunction::definite_limit` | method | 1062 |  | `(MaxTrackSizingFunction).DefiniteLimit` | todo |
| `MaxTrackSizingFunction::resolved_percentage_size` | method | 1079 |  | `(MaxTrackSizingFunction).ResolvedPercentageSize` | todo |
| `MaxTrackSizingFunction::uses_percentage` | method | 1089 |  | `(MaxTrackSizingFunction).UsesPercentage` | todo |
| `MaxTrackSizingFunction::expand` | method | 1098 |  | `(MaxTrackSizingFunction).Expand` | todo |
| `ExpandedMaxTrackSizingFunction` | enum | 1122 |  | `ExpandedMaxTrackSizingFunction` | todo |
| `MinTrackSizingFunction` | struct | 1181 |  | `MinTrackSizingFunction` | todo |
| `MinTrackSizingFunction::length` | assoc-fn | 1289 |  | `MinTrackSizingFunctionLength` | todo |
| `MinTrackSizingFunction::percent` | assoc-fn | 1297 |  | `MinTrackSizingFunctionPercent` | todo |
| `MinTrackSizingFunction::auto` | assoc-fn | 1304 |  | `MinTrackSizingFunctionAuto` | todo |
| `MinTrackSizingFunction::min_content` | assoc-fn | 1311 |  | `MinTrackSizingFunctionMinContent` | todo |
| `MinTrackSizingFunction::max_content` | assoc-fn | 1318 |  | `MinTrackSizingFunctionMaxContent` | todo |
| `MinTrackSizingFunction::calc` | assoc-fn | 1328 | calc | `MinTrackSizingFunctionCalc` | todo |
| `MinTrackSizingFunction::from_raw` | assoc-fn | 1336 |  | `MinTrackSizingFunctionFromRaw` | todo |
| `MinTrackSizingFunction::into_raw` | method | 1341 |  | `(MinTrackSizingFunction).IntoRaw` | todo |
| `MinTrackSizingFunction::is_intrinsic` | method | 1347 |  | `(MinTrackSizingFunction).IsIntrinsic` | todo |
| `MinTrackSizingFunction::is_min_or_max_content` | method | 1353 |  | `(MinTrackSizingFunction).IsMinOrMaxContent` | todo |
| `MinTrackSizingFunction::is_fr` | method | 1359 |  | `(MinTrackSizingFunction).IsFr` | todo |
| `MinTrackSizingFunction::is_auto` | method | 1365 |  | `(MinTrackSizingFunction).IsAuto` | todo |
| `MinTrackSizingFunction::behaves_as_auto` | method | 1371 |  | `(MinTrackSizingFunction).BehavesAsAuto` | todo |
| `MinTrackSizingFunction::is_min_content` | method | 1377 |  | `(MinTrackSizingFunction).IsMinContent` | todo |
| `MinTrackSizingFunction::is_max_content` | method | 1383 |  | `(MinTrackSizingFunction).IsMaxContent` | todo |
| `MinTrackSizingFunction::definite_value` | method | 1391 |  | `(MinTrackSizingFunction).DefiniteValue` | todo |
| `MinTrackSizingFunction::resolved_percentage_size` | method | 1408 |  | `(MinTrackSizingFunction).ResolvedPercentageSize` | todo |
| `MinTrackSizingFunction::uses_percentage` | method | 1418 |  | `(MinTrackSizingFunction).UsesPercentage` | todo |
| `MinTrackSizingFunction::expand` | method | 1427 |  | `(MinTrackSizingFunction).Expand` | todo |
| `ExpandedMinTrackSizingFunction` | enum | 1446 |  | `ExpandedMinTrackSizingFunction` | todo |
| `TrackSizingFunction` | type | 1487 |  | `TrackSizingFunction` | todo |
| `TrackSizingFunction::min_sizing_function` | method | 1490 |  | `(TrackSizingFunction).MinSizingFunction` | todo |
| `TrackSizingFunction::max_sizing_function` | method | 1494 |  | `(TrackSizingFunction).MaxSizingFunction` | todo |
| `TrackSizingFunction::has_fixed_component` | method | 1498 |  | `(TrackSizingFunction).HasFixedComponent` | todo |
| `RepetitionCount` | enum | 1585 |  | `RepetitionCount` | todo |
| `InvalidStringRepetitionValue` | struct | 1604 |  | `InvalidStringRepetitionValue` | todo |
| `GridTemplateRepetition` | struct | 1642 |  | `GridTemplateRepetition` | todo |
| `GridTemplateComponent` | enum | 1684 |  | `GridTemplateComponent` | todo |
| `GridTemplateComponent::as_component_ref` | method | 1694 |  | `(GridTemplateComponent).AsComponentRef` | todo |
| `GridTemplateComponent::is_auto_repetition` | method | 1704 |  | `(GridTemplateComponent).IsAutoRepetition` | todo |
| `GridTemplateTracks` | struct | 1782 |  | `GridTemplateTracks` | todo |
| `GridAutoTracks` | struct | 1838 |  | `GridAutoTracks` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `grid_placement_parser_saturates_numeric_values` | test | 1862 | parse | `TestStyleGrid_GridPlacementParserSaturatesNumericValues` | todo |
| `repetition_parser_saturates_numeric_values` | test | 1881 | parse | `TestStyleGrid_RepetitionParserSaturatesNumericValues` | todo |
| `repetition_track_count_saturates` | test | 1892 | parse | `TestStyleGrid_RepetitionTrackCountSaturates` | todo |
| `max_track_sizing_function_round_trips` | test | 1907 |  | `TestStyleGrid_MaxTrackSizingFunctionRoundTrips` | todo |
| `min_track_sizing_function_round_trips` | test | 1930 |  | `TestStyleGrid_MinTrackSizingFunctionRoundTrips` | todo |
| `track_sizing_function_calc_round_trips` | test | 1947 | calc | `TestStyleGrid_TrackSizingFunctionCalcRoundTrips` | todo |

## src/style/mod.rs

Go target: `internal/taffy/style_mod.go` (package `taffy`), tests in `style_mod_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CheapCloneStr` | trait | 63 |  | `CheapCloneStr` | todo |
| `CoreStyle` | trait | 77 |  | `CoreStyle` | todo |
| `CoreStyle::box_generation_mode` | trait-method | 83 |  | `CoreStyle.BoxGenerationMode` | todo |
| `CoreStyle::is_block` | trait-method | 92 |  | `CoreStyle.IsBlock` | todo |
| `CoreStyle::is_compressible_replaced` | trait-method | 101 |  | `CoreStyle.IsCompressibleReplaced` | todo |
| `CoreStyle::is_replaced` | trait-method | 109 |  | `CoreStyle.IsReplaced` | todo |
| `CoreStyle::box_sizing` | trait-method | 114 |  | `CoreStyle.BoxSizing` | todo |
| `CoreStyle::direction` | trait-method | 120 |  | `CoreStyle.Direction` | todo |
| `CoreStyle::overflow` | trait-method | 127 |  | `CoreStyle.Overflow` | todo |
| `CoreStyle::scrollbar_width` | trait-method | 132 |  | `CoreStyle.ScrollbarWidth` | todo |
| `CoreStyle::position` | trait-method | 139 |  | `CoreStyle.Position` | todo |
| `CoreStyle::inset` | trait-method | 144 |  | `CoreStyle.Inset` | todo |
| `CoreStyle::is_containing_block` | trait-method | 158 |  | `CoreStyle.IsContainingBlock` | todo |
| `CoreStyle::size` | trait-method | 165 |  | `CoreStyle.Size` | todo |
| `CoreStyle::min_size` | trait-method | 170 |  | `CoreStyle.MinSize` | todo |
| `CoreStyle::max_size` | trait-method | 175 |  | `CoreStyle.MaxSize` | todo |
| `CoreStyle::aspect_ratio` | trait-method | 181 |  | `CoreStyle.AspectRatio` | todo |
| `CoreStyle::margin` | trait-method | 188 |  | `CoreStyle.Margin` | todo |
| `CoreStyle::padding` | trait-method | 193 |  | `CoreStyle.Padding` | todo |
| `CoreStyle::border` | trait-method | 198 |  | `CoreStyle.Border` | todo |
| `CoreStyle::contain` | trait-method | 204 |  | `CoreStyle.Contain` | todo |
| `OofItemStyle` | trait | 211 |  | `OofItemStyle` | todo |
| `OofItemStyle::align_self` | trait-method | 215 |  | `OofItemStyle.AlignSelf` | todo |
| `OofItemStyle::justify_self` | trait-method | 221 |  | `OofItemStyle.JustifySelf` | todo |
| `OofItemStyle::is_table` | trait-method | 227 |  | `OofItemStyle.IsTable` | todo |
| `OofItemStyle::grid_row` | trait-method | 234 | grid | `OofItemStyle.GridRow` | todo |
| `OofItemStyle::grid_column` | trait-method | 241 | grid | `OofItemStyle.GridColumn` | todo |
| `Display` | enum | 251 |  | `Display` | todo |
| `Display::DEFAULT` | assoc-const | 271 | flexbox | `DisplayDefault` | todo |
| `Display::DEFAULT` | assoc-const | 275 | grid, flexbox | `DisplayDefault` | todo |
| `Display::DEFAULT` | assoc-const | 279 | block_layout, flexbox, grid | `DisplayDefault` | todo |
| `Display::DEFAULT` | assoc-const | 283 | flexbox, grid, block_layout | `DisplayDefault` | todo |
| `BoxGenerationMode` | enum | 325 |  | `BoxGenerationMode` | todo |
| `BoxGenerationMode::DEFAULT` | assoc-const | 334 |  | `BoxGenerationModeDefault` | todo |
| `Position` | enum | 354 |  | `Position` | todo |
| `Position::is_out_of_flow` | method | 382 |  | `(Position).IsOutOfFlow` | todo |
| `Position::is_positioned` | method | 389 |  | `(Position).IsPositioned` | todo |
| `ContainingBlockClaims` | struct | 398 |  | `ContainingBlockClaims` | todo |
| `ContainingBlockClaims::NONE` | assoc-const | 407 |  | `ContainingBlockClaimsNone` | todo |
| `ContainingBlockClaims::ALL` | assoc-const | 409 |  | `ContainingBlockClaimsAll` | todo |
| `ContainingBlockClaims::for_position` | method | 413 |  | `(ContainingBlockClaims).ForPosition` | todo |
| `BoxSizing` | enum | 446 |  | `BoxSizing` | todo |
| `Overflow` | enum | 475 |  | `Overflow` | todo |
| `Overflow::is_scroll_container` | method | 496 |  | `(Overflow).IsScrollContainer` | todo |
| `Contain` | struct | 540 |  | `Contain` | todo |
| `Contain::NONE` | assoc-const | 544 |  | `ContainNone` | todo |
| `Contain::LAYOUT` | assoc-const | 548 |  | `ContainLayout` | todo |
| `Contain::PAINT` | assoc-const | 552 |  | `ContainPaint` | todo |
| `Contain::CONTENT` | assoc-const | 554 |  | `ContainContent` | todo |
| `Contain::DEFAULT` | assoc-const | 557 |  | `ContainDefault` | todo |
| `Contain::contains` | method | 561 |  | `(Contain).Contains` | todo |
| `Contain::intersects` | method | 567 |  | `(Contain).Intersects` | todo |
| `Contain::union` | method | 573 |  | `(Contain).Union` | todo |
| `Contain::establishes_independent_formatting_context` | method | 580 |  | `(Contain).EstablishesIndependentFormattingContext` | todo |
| `Contain::suppresses_baseline` | method | 587 |  | `(Contain).SuppressesBaseline` | todo |
| `Contain::contains_scrollable_overflow` | method | 595 |  | `(Contain).ContainsScrollableOverflow` | todo |
| `Direction` | enum | 670 |  | `Direction` | todo |
| `Direction::is_rtl` | method | 681 |  | `(Direction).IsRtl` | todo |
| `Style` | struct | 709 |  | `Style` | todo |
| `Style::DEFAULT` | assoc-const | 876 |  | `StyleDefault` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `defaults_match` | test | 1631 |  | `TestStyleMod_DefaultsMatch` | todo |
| `parse_contain` | test | 1712 | parse | `TestStyleMod_ParseContain` | todo |
| `style_sizes` | test | 1742 |  | `TestStyleMod_StyleSizes` | todo |

## src/style_helpers.rs

Go target: `internal/taffy/style_helpers.go` (package `taffy`), tests in `style_helpers_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `repeat` | fn | 23 | grid | `Repeat` | todo |
| `evenly_sized_tracks` | fn | 38 | grid | `EvenlySizedTracks` | todo |
| `line` | fn | 52 |  | `Line` | todo |
| `TaffyGridLine` | trait | 56 |  | `TaffyGridLine` | todo |
| `TaffyGridLine::from_line_index` | trait-method | 58 |  | `TaffyGridLine.FromLineIndex` | todo |
| `span` | fn | 63 |  | `Span` | todo |
| `TaffyGridSpan` | trait | 67 |  | `TaffyGridSpan` | todo |
| `TaffyGridSpan::from_span` | trait-method | 69 |  | `TaffyGridSpan.FromSpan` | todo |
| `minmax` | fn | 75 | grid | `Minmax` | todo |
| `flex` | fn | 85 | grid | `Flex` | todo |
| `zero` | fn | 95 |  | `Zero` | todo |
| `TaffyZero` | trait | 100 |  | `TaffyZero` | todo |
| `Point::zero` | assoc-fn | 117 |  | `PointZero` | todo |
| `Line::zero` | assoc-fn | 128 |  | `LineZero` | todo |
| `Size::zero` | assoc-fn | 139 |  | `SizeZero` | todo |
| `Rect::zero` | assoc-fn | 150 |  | `RectZero` | todo |
| `auto` | fn | 157 |  | `Auto` | todo |
| `TaffyAuto` | trait | 162 |  | `TaffyAuto` | todo |
| `Point::auto` | assoc-fn | 176 |  | `PointAuto` | todo |
| `Line::auto` | assoc-fn | 187 |  | `LineAuto` | todo |
| `Size::auto` | assoc-fn | 198 |  | `SizeAuto` | todo |
| `Rect::auto` | assoc-fn | 209 |  | `RectAuto` | todo |
| `min_content` | fn | 216 |  | `MinContent` | todo |
| `TaffyMinContent` | trait | 221 |  | `TaffyMinContent` | todo |
| `Point::min_content` | assoc-fn | 235 |  | `PointMinContent` | todo |
| `Line::min_content` | assoc-fn | 246 |  | `LineMinContent` | todo |
| `Size::min_content` | assoc-fn | 257 |  | `SizeMinContent` | todo |
| `Rect::min_content` | assoc-fn | 269 |  | `RectMinContent` | todo |
| `max_content` | fn | 276 |  | `MaxContent` | todo |
| `TaffyMaxContent` | trait | 281 |  | `TaffyMaxContent` | todo |
| `Point::max_content` | assoc-fn | 295 |  | `PointMaxContent` | todo |
| `Line::max_content` | assoc-fn | 306 |  | `LineMaxContent` | todo |
| `Size::max_content` | assoc-fn | 317 |  | `SizeMaxContent` | todo |
| `Rect::max_content` | assoc-fn | 329 |  | `RectMaxContent` | todo |
| `fit_content` | fn | 337 |  | `FitContent` | todo |
| `TaffyFitContent` | trait | 342 |  | `TaffyFitContent` | todo |
| `TaffyFitContent::fit_content` | trait-method | 344 |  | `TaffyFitContent.FitContent` | todo |
| `Point::fit_content` | assoc-fn | 356 |  | `PointFitContent` | todo |
| `Line::fit_content` | assoc-fn | 370 |  | `LineFitContent` | todo |
| `Size::fit_content` | assoc-fn | 384 |  | `SizeFitContent` | todo |
| `Rect::fit_content` | assoc-fn | 403 |  | `RectFitContent` | todo |
| `length` | fn | 410 |  | `Length` | todo |
| `FromLength` | trait | 415 |  | `FromLength` | todo |
| `FromLength::from_length` | trait-method | 417 |  | `FromLength.FromLength` | todo |
| `Point::length` | assoc-fn | 440 |  | `PointLength` | todo |
| `Line::length` | assoc-fn | 453 |  | `LineLength` | todo |
| `Size::length` | assoc-fn | 466 |  | `SizeLength` | todo |
| `Rect::length` | assoc-fn | 484 |  | `RectLength` | todo |
| `percent` | fn | 491 |  | `Percent` | todo |
| `FromPercent` | trait | 496 |  | `FromPercent` | todo |
| `FromPercent::from_percent` | trait-method | 498 |  | `FromPercent.FromPercent` | todo |
| `Point::percent` | assoc-fn | 522 |  | `PointPercent` | todo |
| `Line::percent` | assoc-fn | 536 |  | `LinePercent` | todo |
| `Size::percent` | assoc-fn | 550 |  | `SizePercent` | todo |
| `Rect::percent` | assoc-fn | 569 |  | `RectPercent` | todo |
| `fr` | fn | 577 | grid | `Fr` | todo |
| `FromFr` | trait | 582 |  | `FromFr` | todo |
| `FromFr::from_fr` | trait-method | 584 |  | `FromFr.FromFr` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_repeat_u16` | test | 600 | grid | `TestStyleHelpers_RepeatU16` | todo |
| `test_repeat_auto_fit_str` | test | 612 | grid | `TestStyleHelpers_RepeatAutoFitStr` | todo |
| `test_repeat_auto_fill_str` | test | 624 | grid | `TestStyleHelpers_RepeatAutoFillStr` | todo |

## src/tree/cache.rs

Go target: `internal/taffy/tree_cache.go` (package `taffy`), tests in `tree_cache_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `cache_mode_change_evictions` | fn | 197 | std | `CacheModeChangeEvictions` | todo |
| `Cache` | struct | 214 |  | `Cache` | todo |
| `Cache::new` | assoc-fn | 239 |  | `NewCache` | todo |
| `Cache::get` | method | 254 |  | `(Cache).Get` | todo |
| `Cache::store` | method | 283 |  | `(Cache).Store` | todo |
| `Cache::clear` | method | 341 |  | `(Cache).Clear` | todo |
| `Cache::is_empty` | method | 356 |  | `(Cache).IsEmpty` | todo |
| `ClearState` | enum | 362 |  | `ClearState` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `recently_used_measure_entries_get_a_second_chance` | test | 393 |  | `TestTreeCache_RecentlyUsedMeasureEntriesGetASecondChance` | todo |
| `storing_an_existing_measurement_updates_it_in_place` | test | 408 |  | `TestTreeCache_StoringAnExistingMeasurementUpdatesItInPlace` | todo |
| `measurements_with_margin_collapse_metadata_are_not_cached` | test | 419 |  | `TestTreeCache_MeasurementsWithMarginCollapseMetadataAreNotCached` | todo |
| `retrieving_a_measurement_only_marks_its_slot_as_used` | test | 434 |  | `TestTreeCache_RetrievingAMeasurementOnlyMarksItsSlotAsUsed` | todo |
| `cache_mode_distinguishes_every_combination_of_inputs` | test | 466 |  | `TestTreeCache_CacheModeDistinguishesEveryCombinationOfInputs` | todo |
| `retrieving_with_a_different_mode_misses_and_keeps_the_entries` | test | 481 |  | `TestTreeCache_RetrievingWithADifferentModeMissesAndKeepsTheEntries` | todo |
| `storing_with_a_different_mode_drops_the_existing_entries` | test | 498 |  | `TestTreeCache_StoringWithADifferentModeDropsTheExistingEntries` | todo |
| `hidden_layouts_do_not_affect_the_cache` | test | 530 |  | `TestTreeCache_HiddenLayoutsDoNotAffectTheCache` | todo |
| `mode_change_evictions_are_counted_for_non_empty_caches_only` | test | 539 | std | `TestTreeCache_ModeChangeEvictionsAreCountedForNonEmptyCachesOnly` | todo |

## src/tree/layout.rs

Go target: `internal/taffy/tree_layout.go` (package `taffy`), tests in `tree_layout_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `RunMode` | enum | 12 |  | `RunMode` | todo |
| `SizingMode` | enum | 25 |  | `SizingMode` | todo |
| `CollapsibleMarginSet` | struct | 35 |  | `CollapsibleMarginSet` | todo |
| `CollapsibleMarginSet::ZERO` | assoc-const | 44 |  | `CollapsibleMarginSetZero` | todo |
| `CollapsibleMarginSet::from_margin` | assoc-fn | 47 |  | `CollapsibleMarginSetFromMargin` | todo |
| `CollapsibleMarginSet::collapse_with_margin` | method | 56 |  | `(CollapsibleMarginSet).CollapseWithMargin` | todo |
| `CollapsibleMarginSet::collapse_with_set` | method | 66 |  | `(CollapsibleMarginSet).CollapseWithSet` | todo |
| `CollapsibleMarginSet::resolve` | method | 74 |  | `(CollapsibleMarginSet).Resolve` | todo |
| `RequestedAxis` | enum | 82 |  | `RequestedAxis` | todo |
| `RequestedAxis::transpose` | method | 94 |  | `(RequestedAxis).Transpose` | todo |
| `LayoutInput` | struct | 125 |  | `LayoutInput` | todo |
| `LayoutInput::HIDDEN` | assoc-const | 168 |  | `LayoutInputHidden` | todo |
| `LayoutInput::transpose` | method | 187 |  | `(LayoutInput).Transpose` | todo |
| `Baselines` | struct | 208 |  | `Baselines` | todo |
| `Baselines::NONE` | assoc-const | 217 |  | `BaselinesNone` | todo |
| `Baselines::from_first` | assoc-fn | 220 |  | `BaselinesFromFirst` | todo |
| `AxisStaticEdge` | enum | 234 |  | `AxisStaticEdge` | todo |
| `AxisStaticAlign` | struct | 250 |  | `AxisStaticAlign` | todo |
| `AxisStaticAlign::from_keyword` | assoc-fn | 263 |  | `AxisStaticAlignFromKeyword` | todo |
| `AxisStaticPosition` | struct | 278 |  | `AxisStaticPosition` | todo |
| `AxisStaticPosition::from_edge` | assoc-fn | 288 |  | `AxisStaticPositionFromEdge` | todo |
| `AxisStaticPosition::from_alignment` | assoc-fn | 302 |  | `AxisStaticPositionFromAlignment` | todo |
| `OofCandidate` | struct | 341 |  | `OofCandidate` | todo |
| `OofCandidates` | struct | 356 |  | `OofCandidates` | todo |
| `OofCandidates::NONE` | assoc-const | 363 |  | `OofCandidatesNone` | todo |
| `OofCandidates::new` | assoc-fn | 366 |  | `NewOofCandidates` | todo |
| `OofCandidates::is_empty` | method | 372 |  | `(OofCandidates).IsEmpty` | todo |
| `OofCandidates::as_slice` | method | 378 |  | `(OofCandidates).AsSlice` | todo |
| `OofCandidates::as_mut_slice` | method | 384 |  | `(OofCandidates).AsMutSlice` | todo |
| `OofCandidates::push` | method | 390 |  | `(OofCandidates).Push` | todo |
| `OofCandidates::append` | method | 396 |  | `(OofCandidates).Append` | todo |
| `OofCandidates::take` | method | 408 |  | `(OofCandidates).Take` | todo |
| `OofCandidates::clear` | method | 414 |  | `(OofCandidates).Clear` | todo |
| `OofCandidates::iter` | method | 420 |  | `(OofCandidates).Iter` | todo |
| `OofCandidates::translate` | method | 429 |  | `(OofCandidates).Translate` | todo |
| `OofPositioningArea` | struct | 457 |  | `OofPositioningArea` | todo |
| `LayoutOutput` | struct | 472 |  | `LayoutOutput` | todo |
| `LayoutOutput::HIDDEN` | assoc-const | 511 |  | `LayoutOutputHidden` | todo |
| `LayoutOutput::DEFAULT` | assoc-const | 524 |  | `LayoutOutputDefault` | todo |
| `LayoutOutput::from_sizes_and_baselines` | assoc-fn | 527 |  | `LayoutOutputFromSizesAndBaselines` | todo |
| `LayoutOutput::from_sizes` | assoc-fn | 546 |  | `LayoutOutputFromSizes` | todo |
| `LayoutOutput::transpose` | method | 554 |  | `(LayoutOutput).Transpose` | todo |
| `LayoutOutput::from_outer_size` | assoc-fn | 571 |  | `LayoutOutputFromOuterSize` | todo |
| `Layout` | struct | 579 |  | `Layout` | todo |
| `Layout::new` | assoc-fn | 627 |  | `NewLayout` | todo |
| `Layout::with_order` | assoc-fn | 646 |  | `LayoutWithOrder` | todo |
| `Layout::content_box_width` | method | 662 |  | `(Layout).ContentBoxWidth` | todo |
| `Layout::content_box_height` | method | 668 |  | `(Layout).ContentBoxHeight` | todo |
| `Layout::content_box_size` | method | 674 |  | `(Layout).ContentBoxSize` | todo |
| `Layout::content_box_x` | method | 679 |  | `(Layout).ContentBoxX` | todo |
| `Layout::content_box_y` | method | 684 |  | `(Layout).ContentBoxY` | todo |
| `Layout::scroll_width` | method | 693 | content_size | `(Layout).ScrollWidth` | todo |
| `Layout::scroll_height` | method | 704 | content_size | `(Layout).ScrollHeight` | todo |
| `DetailedLayoutInfo` | enum | 717 |  | `DetailedLayoutInfo` | todo |

## src/tree/mod.rs

Go target: `internal/taffy/tree_mod.go` (package `taffy`), tests in `tree_mod_test.go`

No public items or tests.

## src/tree/node.rs

Go target: `internal/taffy/tree_node.go` (package `taffy`), tests in `tree_node_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `NodeId` | struct | 13 |  | `NodeID` | todo |
| `NodeId::new` | assoc-fn | 16 |  | `NewNodeID` | todo |

## src/tree/taffy_tree.rs

Go target: `internal/taffy/tree_taffy_tree.go` (package `taffy`), tests in `tree_taffy_tree_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TaffyResult` | type | 38 |  | `TaffyResult` | todo |
| `TaffyError` | enum | 42 |  | `TaffyError` | todo |
| `TaffyTree` | struct | 150 |  | `TaffyTree` | todo |
| `TaffyTreeChildIter` | struct | 178 |  | `TaffyTreeChildIter` | todo |
| `TaffyTree::new` | assoc-fn | 591 |  | `NewTaffyTree` | todo |
| `TaffyTree::with_capacity` | assoc-fn | 597 |  | `TaffyTreeWithCapacity` | todo |
| `TaffyTree::enable_rounding` | method | 610 |  | `(TaffyTree).EnableRounding` | todo |
| `TaffyTree::disable_rounding` | method | 615 |  | `(TaffyTree).DisableRounding` | todo |
| `TaffyTree::new_leaf` | method | 620 |  | `(TaffyTree).NewLeaf` | todo |
| `TaffyTree::new_leaf_with_context` | method | 631 |  | `(TaffyTree).NewLeafWithContext` | todo |
| `TaffyTree::new_with_children` | method | 645 |  | `(TaffyTree).NewWithChildren` | todo |
| `TaffyTree::clear` | method | 659 |  | `(TaffyTree).Clear` | todo |
| `TaffyTree::remove` | method | 672 |  | `(TaffyTree).Remove` | todo |
| `TaffyTree::set_node_context` | method | 698 |  | `(TaffyTree).SetNodeContext` | todo |
| `TaffyTree::get_node_context` | method | 715 |  | `(TaffyTree).GetNodeContext` | todo |
| `TaffyTree::get_node_context_mut` | method | 721 |  | `(TaffyTree).GetNodeContextMut` | todo |
| `TaffyTree::get_disjoint_node_context_mut` | method | 726 |  | `(TaffyTree).GetDisjointNodeContextMut` | todo |
| `TaffyTree::add_child` | method | 734 |  | `(TaffyTree).AddChild` | todo |
| `TaffyTree::insert_child_at_index` | method | 745 |  | `(TaffyTree).InsertChildAtIndex` | todo |
| `TaffyTree::set_children` | method | 761 |  | `(TaffyTree).SetChildren` | todo |
| `TaffyTree::remove_child` | method | 790 |  | `(TaffyTree).RemoveChild` | todo |
| `TaffyTree::remove_child_at_index` | method | 798 |  | `(TaffyTree).RemoveChildAtIndex` | todo |
| `TaffyTree::remove_children_range` | method | 818 |  | `(TaffyTree).RemoveChildrenRange` | todo |
| `TaffyTree::replace_child_at_index` | method | 834 |  | `(TaffyTree).ReplaceChildAtIndex` | todo |
| `TaffyTree::child_at_index` | method | 858 |  | `(TaffyTree).ChildAtIndex` | todo |
| `TaffyTree::total_node_count` | method | 870 |  | `(TaffyTree).TotalNodeCount` | todo |
| `TaffyTree::parent` | method | 879 |  | `(TaffyTree).Parent` | todo |
| `TaffyTree::children` | method | 884 |  | `(TaffyTree).Children` | todo |
| `TaffyTree::hoisted_children` | method | 891 |  | `(TaffyTree).HoistedChildren` | todo |
| `TaffyTree::set_style` | method | 897 |  | `(TaffyTree).SetStyle` | todo |
| `TaffyTree::style` | method | 905 |  | `(TaffyTree).Style` | todo |
| `TaffyTree::layout` | method | 911 |  | `(TaffyTree).Layout` | todo |
| `TaffyTree::unrounded_layout` | method | 921 |  | `(TaffyTree).UnroundedLayout` | todo |
| `TaffyTree::detailed_layout_info` | method | 930 |  | `(TaffyTree).DetailedLayoutInfo` | todo |
| `TaffyTree::mark_dirty` | method | 935 |  | `(TaffyTree).MarkDirty` | todo |
| `TaffyTree::dirty` | method | 962 |  | `(TaffyTree).Dirty` | todo |
| `TaffyTree::compute_layout_with_measure` | method | 967 |  | `(TaffyTree).ComputeLayoutWithMeasure` | todo |
| `TaffyTree::compute_layout` | method | 986 |  | `(TaffyTree).ComputeLayout` | todo |
| `TaffyTree::print_tree` | method | 994 | std | `(TaffyTree).PrintTree` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `new_should_allocate_default_capacity` | test | 1033 |  | `TestTreeTaffyTree_NewShouldAllocateDefaultCapacity` | todo |
| `test_with_capacity` | test | 1043 |  | `TestTreeTaffyTree_WithCapacity` | todo |
| `test_new_leaf` | test | 1053 |  | `TestTreeTaffyTree_NewLeaf` | todo |
| `new_leaf_with_context` | test | 1065 |  | `TestTreeTaffyTree_NewLeafWithContext` | todo |
| `test_new_with_children` | test | 1078 |  | `TestTreeTaffyTree_NewWithChildren` | todo |
| `remove_node_should_remove` | test | 1091 |  | `TestTreeTaffyTree_RemoveNodeShouldRemove` | todo |
| `remove_node_should_detach_hierarchy` | test | 1100 |  | `TestTreeTaffyTree_RemoveNodeShouldDetachHierarchy` | todo |
| `remove_last_node` | test | 1121 |  | `TestTreeTaffyTree_RemoveLastNode` | todo |
| `remove_node_marks_parent_dirty` | test | 1134 |  | `TestTreeTaffyTree_RemoveNodeMarksParentDirty` | todo |
| `set_measure` | test | 1157 |  | `TestTreeTaffyTree_SetMeasure` | todo |
| `set_measure_of_previously_unmeasured_node` | test | 1169 |  | `TestTreeTaffyTree_SetMeasureOfPreviouslyUnmeasuredNode` | todo |
| `add_child` | test | 1182 |  | `TestTreeTaffyTree_AddChild` | todo |
| `insert_child_at_index` | test | 1197 |  | `TestTreeTaffyTree_InsertChildAtIndex` | todo |
| `set_children` | test | 1224 |  | `TestTreeTaffyTree_SetChildren` | todo |
| `remove_child` | test | 1246 |  | `TestTreeTaffyTree_RemoveChild` | todo |
| `remove_child_at_index` | test | 1263 |  | `TestTreeTaffyTree_RemoveChildAtIndex` | todo |
| `remove_children_range` | test | 1280 |  | `TestTreeTaffyTree_RemoveChildrenRange` | todo |
| `remove_child_updates_parents` | test | 1303 |  | `TestTreeTaffyTree_RemoveChildUpdatesParents` | todo |
| `replace_child_at_index` | test | 1318 |  | `TestTreeTaffyTree_ReplaceChildAtIndex` | todo |
| `test_child_at_index` | test | 1333 |  | `TestTreeTaffyTree_ChildAtIndex` | todo |
| `test_child_count` | test | 1345 |  | `TestTreeTaffyTree_ChildCount` | todo |
| `test_children` | test | 1358 |  | `TestTreeTaffyTree_Children` | todo |
| `test_set_style` | test | 1374 |  | `TestTreeTaffyTree_SetStyle` | todo |
| `test_style` | test | 1384 |  | `TestTreeTaffyTree_Style` | todo |
| `test_layout` | test | 1396 |  | `TestTreeTaffyTree_Layout` | todo |
| `test_mark_dirty` | test | 1406 |  | `TestTreeTaffyTree_MarkDirty` | todo |
| `compute_layout_should_produce_valid_result` | test | 1431 |  | `TestTreeTaffyTree_ComputeLayoutShouldProduceValidResult` | todo |
| `make_sure_layout_location_is_top_left` | test | 1447 |  | `TestTreeTaffyTree_MakeSureLayoutLocationIsTopLeft` | todo |
| `set_children_reparents` | test | 1491 |  | `TestTreeTaffyTree_SetChildrenReparents` | todo |
| `remove_drops_the_nodes_context` | test | 1513 |  | `TestTreeTaffyTree_RemoveDropsTheNodesContext` | todo |
| `clear_drops_every_nodes_context` | test | 1528 |  | `TestTreeTaffyTree_ClearDropsEveryNodesContext` | todo |

## src/tree/traits.rs

Go target: `internal/taffy/tree_traits.go` (package `taffy`), tests in `tree_traits_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TraversePartialTree` | trait | 147 |  | `TraversePartialTree` | todo |
| `TraversePartialTree::child_ids` | trait-method | 154 |  | `TraversePartialTree.ChildIds` | todo |
| `TraversePartialTree::child_count` | trait-method | 157 |  | `TraversePartialTree.ChildCount` | todo |
| `TraversePartialTree::get_child_id` | trait-method | 160 |  | `TraversePartialTree.GetChildID` | todo |
| `TraverseTree` | trait | 167 |  | `TraverseTree` | todo |
| `LayoutPartialTree` | trait | 173 |  | `LayoutPartialTree` | todo |
| `LayoutPartialTree::get_core_container_style` | trait-method | 189 |  | `LayoutPartialTree.GetCoreContainerStyle` | todo |
| `LayoutPartialTree::resolve_calc_value` | trait-method | 193 |  | `LayoutPartialTree.ResolveCalcValue` | todo |
| `LayoutPartialTree::set_unrounded_layout` | trait-method | 200 |  | `LayoutPartialTree.SetUnroundedLayout` | todo |
| `LayoutPartialTree::compute_child_layout` | trait-method | 203 |  | `LayoutPartialTree.ComputeChildLayout` | todo |
| `LayoutContainingBlock` | trait | 212 |  | `LayoutContainingBlock` | todo |
| `LayoutContainingBlock::get_oof_item_style` | trait-method | 222 |  | `LayoutContainingBlock.GetOofItemStyle` | todo |
| `LayoutContainingBlock::clear_hoisted_children` | trait-method | 229 |  | `LayoutContainingBlock.ClearHoistedChildren` | todo |
| `LayoutContainingBlock::add_hoisted_children` | trait-method | 240 |  | `LayoutContainingBlock.AddHoistedChildren` | todo |
| `LayoutContainingBlock::get_detailed_layout_info` | trait-method | 251 |  | `LayoutContainingBlock.GetDetailedLayoutInfo` | todo |
| `CacheTree` | trait | 260 |  | `CacheTree` | todo |
| `CacheTree::cache_get` | trait-method | 262 |  | `CacheTree.CacheGet` | todo |
| `CacheTree::cache_store` | trait-method | 265 |  | `CacheTree.CacheStore` | todo |
| `CacheTree::cache_clear` | trait-method | 268 |  | `CacheTree.CacheClear` | todo |
| `RoundTree` | trait | 275 |  | `RoundTree` | todo |
| `RoundTree::get_unrounded_layout` | trait-method | 277 |  | `RoundTree.GetUnroundedLayout` | todo |
| `RoundTree::set_final_layout` | trait-method | 279 |  | `RoundTree.SetFinalLayout` | todo |
| `RoundTree::is_out_of_flow` | trait-method | 286 |  | `RoundTree.IsOutOfFlow` | todo |
| `RoundTree::hoisted_child_count` | trait-method | 289 |  | `RoundTree.HoistedChildCount` | todo |
| `RoundTree::get_hoisted_child_id` | trait-method | 292 |  | `RoundTree.GetHoistedChildID` | todo |
| `PrintTree` | trait | 298 |  | `PrintTree` | todo |
| `PrintTree::get_debug_label` | trait-method | 300 |  | `PrintTree.GetDebugLabel` | todo |
| `PrintTree::get_final_layout` | trait-method | 302 |  | `PrintTree.GetFinalLayout` | todo |
| `LayoutFlexboxContainer` | trait | 307 | flexbox | `LayoutFlexboxContainer` | todo |
| `LayoutFlexboxContainer::get_flexbox_container_style` | trait-method | 318 | flexbox | `LayoutFlexboxContainer.GetFlexboxContainerStyle` | todo |
| `LayoutFlexboxContainer::get_flexbox_child_style` | trait-method | 323 | flexbox | `LayoutFlexboxContainer.GetFlexboxChildStyle` | todo |
| `LayoutGridContainer` | trait | 328 | grid | `LayoutGridContainer` | todo |
| `LayoutGridContainer::get_grid_container_style` | trait-method | 340 | grid | `LayoutGridContainer.GetGridContainerStyle` | todo |
| `LayoutGridContainer::get_grid_child_style` | trait-method | 345 | grid | `LayoutGridContainer.GetGridChildStyle` | todo |
| `LayoutGridContainer::set_detailed_grid_info` | trait-method | 351 | grid | `LayoutGridContainer.SetDetailedGridInfo` | todo |
| `LayoutBlockContainer` | trait | 358 | block_layout | `LayoutBlockContainer` | todo |
| `LayoutBlockContainer::get_block_container_style` | trait-method | 369 | block_layout | `LayoutBlockContainer.GetBlockContainerStyle` | todo |
| `LayoutBlockContainer::get_block_child_style` | trait-method | 374 | block_layout | `LayoutBlockContainer.GetBlockChildStyle` | todo |
| `LayoutBlockContainer::compute_block_child_layout` | trait-method | 378 | block_layout | `LayoutBlockContainer.ComputeBlockChildLayout` | todo |

## src/util/debug.rs

Go target: `internal/taffy/util_debug.go` (package `taffy`), tests in `util_debug_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `DebugLogger` | struct | 10 | debug, profile | `DebugLogger` | todo |
| `DebugLogger::new` | assoc-fn | 18 | debug, profile | `NewDebugLogger` | todo |
| `DebugLogger::push_node` | method | 22 | debug, profile | `(DebugLogger).PushNode` | todo |
| `DebugLogger::pop_node` | method | 29 | debug, profile | `(DebugLogger).PopNode` | todo |
| `DebugLogger::log` | method | 34 | debug, profile | `(DebugLogger).Log` | todo |
| `DebugLogger::labelled_log` | method | 42 | debug, profile | `(DebugLogger).LabelledLog` | todo |
| `DebugLogger::debug_log` | method | 50 | debug, profile | `(DebugLogger).DebugLog` | todo |
| `DebugLogger::labelled_debug_log` | method | 58 | debug, profile | `(DebugLogger).LabelledDebugLog` | todo |

## src/util/front_back_vec_builder.rs

Go target: `internal/taffy/util_front_back_vec_builder.go` (package `taffy`), tests in `util_front_back_vec_builder_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `front_items_keep_their_order_and_back_items_are_reversed` | test | 86 |  | `TestUtilFrontBackVecBuilder_FrontItemsKeepTheirOrderAndBackItemsAreReversed` | todo |
| `all_items_pushed_to_one_end` | test | 101 |  | `TestUtilFrontBackVecBuilder_AllItemsPushedToOneEnd` | todo |
| `zero_capacity` | test | 112 |  | `TestUtilFrontBackVecBuilder_ZeroCapacity` | todo |
| `owned_items_are_moved_into_the_vec` | test | 119 |  | `TestUtilFrontBackVecBuilder_OwnedItemsAreMovedIntoTheVec` | todo |
| `pushing_too_many_items_to_the_front_panics_in_into_vec` | test | 130 |  | `TestUtilFrontBackVecBuilder_PushingTooManyItemsToTheFrontPanicsInIntoVec` | todo |
| `pushing_too_many_items_to_the_back_panics_in_into_vec` | test | 139 |  | `TestUtilFrontBackVecBuilder_PushingTooManyItemsToTheBackPanicsInIntoVec` | todo |
| `push_front_out_of_bounds_panics` | test | 148 |  | `TestUtilFrontBackVecBuilder_PushFrontOutOfBoundsPanics` | todo |
| `push_back_out_of_bounds_panics` | test | 155 |  | `TestUtilFrontBackVecBuilder_PushBackOutOfBoundsPanics` | todo |
| `into_vec_before_full_panics` | test | 163 |  | `TestUtilFrontBackVecBuilder_IntoVecBeforeFullPanics` | todo |

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
| `ParseError` | struct | 15 |  | `ParseError` | todo |
| `ParseResult` | type | 36 |  | `ParseResult` | todo |

## src/util/print.rs

Go target: `internal/taffy/util_print.go` (package `taffy`), tests in `util_print_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `print_tree` | fn | 6 |  | `PrintTree` | todo |
| `write_tree` | fn | 14 |  | `WriteTree` | todo |

## src/util/resolve.rs

Go target: `internal/taffy/util_resolve.go` (package `taffy`), tests in `util_resolve_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MaybeResolve` | trait | 13 |  | `MaybeResolve` | todo |
| `MaybeResolve::maybe_resolve` | trait-method | 15 |  | `MaybeResolve.MaybeResolve` | todo |
| `ResolveOrZero` | trait | 23 |  | `ResolveOrZero` | todo |
| `ResolveOrZero::resolve_or_zero` | trait-method | 25 |  | `ResolveOrZero.ResolveOrZero` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `resolve_auto` | test | 205 |  | `TestUtilResolve_ResolveAuto` | todo |
| `resolve_length` | test | 217 |  | `TestUtilResolve_ResolveLength` | todo |
| `resolve_percent` | test | 230 |  | `TestUtilResolve_ResolvePercent` | todo |
| `maybe_resolve_auto` | test | 247 |  | `TestUtilResolve_MaybeResolveAuto` | todo |
| `maybe_resolve_length` | test | 259 |  | `TestUtilResolve_MaybeResolveLength` | todo |
| `maybe_resolve_percent` | test | 272 |  | `TestUtilResolve_MaybeResolvePercent` | todo |
| `resolve_or_zero_auto` | test | 286 |  | `TestUtilResolve_ResolveOrZeroAuto` | todo |
| `resolve_or_zero_length` | test | 293 |  | `TestUtilResolve_ResolveOrZeroLength` | todo |
| `resolve_or_zero_percent` | test | 300 |  | `TestUtilResolve_ResolveOrZeroPercent` | todo |
| `resolve_or_zero_auto` | test | 314 |  | `TestUtilResolve_ResolveOrZeroAuto` | todo |
| `resolve_or_zero_length` | test | 322 |  | `TestUtilResolve_ResolveOrZeroLength` | todo |
| `resolve_or_zero_percent` | test | 330 |  | `TestUtilResolve_ResolveOrZeroPercent` | todo |
| `resolve_or_zero_auto` | test | 348 |  | `TestUtilResolve_ResolveOrZeroAuto` | todo |
| `resolve_or_zero_length` | test | 356 |  | `TestUtilResolve_ResolveOrZeroLength` | todo |
| `resolve_or_zero_percent` | test | 364 |  | `TestUtilResolve_ResolveOrZeroPercent` | todo |

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

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-TAFFY-FEATURE | Code behind cargo features gpui does not enable is not ported: `calc`, `serde`, `parse`, `detailed_layout_info`, `flexbox_balance`, `debug`, `profile`, `strict_provenance` and the `not(feature = "std")` branches. A test that needs one of them (also without its own cfg, as `calc_round_trips_pointer`) is n/a. | `Cargo.toml` `[features]`, `gpui/Cargo.toml:108-116` | - |
| D-TAFFY-COMPACT | `CompactLength` is a `{tag uint8, bits uint32}` struct instead of a tagged pointer: no pointer packing, the `compat` transmutes are `math.Float32bits` / `Float32frombits`, `tag_ptr` / `untag_ptr` do not exist. Equality stays bitwise. | `src/style/compact_length.rs:10-222` | `internal/taffy/style_compact_length.go` |
| D-TAFFY-MAYBEMATH | The `MaybeMath` trait is not a Go interface: its impls for `Option<f32>` and `f32` are free functions, the two `AvailableSpace` impls are methods with an `F32` suffix for the `f32` right-hand side, and the blanket `Size<T>` impl is one function per instantiation (naming in the notes). | `src/util/math.rs:11-252` | `internal/taffy/util_math.go` |
| D-TAFFY-SYS | The `sys` type aliases for `String`, `Map`, `Vec`, `Box`, `ChildrenVec` and `GridTrackVec` are not declared; ported code uses `string`, Go maps and slices directly | `src/util/sys.rs:15-28` | - |
