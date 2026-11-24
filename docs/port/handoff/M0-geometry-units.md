# Handoff: M0-geometry-units

- internal/jsonschema.Schema lacks `pattern`; upstream's AbsoluteLength / DefiniteLength / Length / Rgba schemas carry one (geometry.rs AbsoluteLength JsonSchema, color.rs:105). Whoever owns internal/jsonschema (exit stage) should add `Pattern string \`json:"pattern,omitzero"\`` and the five JSONSchema methods then set it: `^-?\d+(\.\d+)?(px|rem)$`, `^-?\d+(\.\d+)?(px|rem|%)$`, `^(auto|-?\d+(\.\d+)?(px|rem|%))$`, `^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`.
- stubs.go gained `Window` and `(*Window).RemSize` (panicking) for RemsFromPixels; M0-window-core removes them. stubs.go had no geometry entries to remove.
- `From<GridPlacement> for taffy::GridPlacement` is left for M0-element (bridge file).
- For M0-style: the Of family is `EdgesOf[Length]`, `SizeOf[Length]`, `CornersOf[AbsoluteLength]` with refinements `EdgesOfRefinement[T]`, `SizeOfRefinement[T]`, `CornersOfRefinement[T]` (stylegen's contract names them EdgesRefinement etc.; field names match).
