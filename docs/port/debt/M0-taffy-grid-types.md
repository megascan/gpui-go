# M0-taffy-grid-types debt

- D-TAFFY-SMALLVEC: `TrackIntervals.intervals` is a plain slice; the `spilled()` assertion is checked as `len <= 2`.
- `GridItem` sums (`spanned_track_limit`, `grid_area_size`) start from `+0.0`; Rust's `f32` `Sum` starts from `-0.0`, so an empty span yields `+0.0` instead of `-0.0` (no layout depends on the sign).
- `itemPlacement` alias lives in stubs.go until M0-taffy-grid-placement ports placement.rs.
- Methods that collide with fields: `gridTrack.getFitContentLimit`, `namedLineResolver.getAreaColumnCount` / `getAreaRowCount`.
