# Handoff: M0-geometry

For M0-geometry-units (owns geometry.go after this stage):

- Rows left `blocked:M0-geometry-units` because they need Length / DefiniteLength / AbsoluteLength: `SizeLengthFull`, `SizeLengthAuto`, `EdgesLengthAuto`, `EdgesLengthZero`, `EdgesDefiniteLengthZero`, `EdgesDefiniteLengthToPixels`, `EdgesAbsoluteLengthZero`, `EdgesAbsoluteLengthToPixels` (free functions over `SizeOf[...]` / `EdgesOf[...]`, PARITY N7). Also unported for the same reason: `From<Size<Pixels>> for Size<DefiniteLength>` / `Size<AbsoluteLength>` (geometry.rs:651-667), `From<f32>` / `From<Pixels> for Edges<Pixels>` (:2227-2243, equal to `EdgesAll(Pixels(v))`).
- Pixels, DevicePixels, ScaledPixels and Rems are declared at the end of geometry.go with no methods. `Half` and `IsZero` rows are yours; geometry.go uses a private `half[T Number]` (`v / 2`, identical to every upstream Half impl).
- Private helpers you can reuse: `saturatingF32ToI32` (Rust `as i32`), `totalCmpF32`, `geomPartialCmp` / `geomGT` / `geomGE` / `geomLT` / `geomLE` / `geomMax` (Pixels and ScaledPixels ordering), `scalePixels`, `roundToDevicePixels`.
- Refinements are generated for Point, Size, Bounds, Edges and the Of family PointOf, SizeOf, EdgesOf (`PointOfRefinement[T]`, ...), as upstream derives Refineable on all of them; Corners / CornersOf are yours.
- Conversion methods added for upstream `From` impls: `(Point).ToSize`, `(Size).ToPoint`, `(SizeOf).ToPoint`.
- Axis and Anchor (geometry.rs:24-41, :2245-2341) are already ported and marked done in geometry.go; do not redeclare them.
