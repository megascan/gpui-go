# M0-div-api debt

- stubs.go: `ScrollHandle` / `ScrollAnchor` are placeholder structs (M0-div-paint replaces them); `ImageCacheProvider` placeholder interface `Provide(window, cx) AnyImageCache` (M0-elements-media replaces it).
- Style closures (`Hover`, `Focus`, `Active`, `GroupHover`, `DragOver`, ...) take `func(*StyleRefinement) *StyleRefinement` and receive a fresh zero refinement (upstream `FnOnce(StyleRefinement) -> StyleRefinement`).
- `Div.interactivity` is an inline value, not upstream's arena/heap `InteractivityBox` (D-ARENA).
- `Interactive.OnClick` moved to `StatefulInteractive.OnClick` as upstream (it was misplaced in the M0-contracts stub).
- `Div` Element impl (div.rs:1995-2195: ElementID, SourceLocation, A11yRole, WriteA11yInfo, request_layout, prepaint, paint) left to M0-div-paint.
