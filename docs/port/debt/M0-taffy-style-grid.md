# M0-taffy-style-grid debt

- internal/taffy/stubs.go: `GridLine` (with `GridLineFromI16`, `AsI16`), `OriginZeroLine` (`type OriginZeroLine int16`, upstream's `.0` is a conversion, `+ u16` is `+ OriginZeroLine(u)`), `GridCoordinate` and `maxGridTracks` are declared for compute/grid/types/coordinates.rs; `GridLine.intoOriginZeroLine` panics. Owner: M0-taffy-grid-types, which moves them into compute_grid_types_coordinates.go. Until then `GridPlacement.IntoOriginZeroPlacementIgnoringNamed` / `nonNamedGridPlacementIntoOriginZeroPlacement` panic for definite line placements.
- `StyleRef.GridTemplateRows` / `GridTemplateColumns` allocate a `[]GenericGridTemplateComponent` per call (D-TAFFY-GRIDLISTS). If grid layout profiling shows it, cache the ref slice or switch the trait to an indexable view.
- `Option<T>` helper impls (`TaffyZero`, `TaffyAuto`, ... for `Option<T>`) only work for `Option[float32]` (D-TAFFY-HELPERS).
