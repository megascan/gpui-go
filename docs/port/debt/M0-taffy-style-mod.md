# M0-taffy-style-mod debt

- internal/taffy/stubs.go: grid style types declared with placeholder shapes so `Style[S]` can name them: `GridTemplateComponent[S]`, `GridTemplateRepetition[S]`, `MinTrackSizingFunction`, `MaxTrackSizingFunction`, `TrackSizingFunction` (alias of `MinMax[...]`), `GridAutoFlow` (+ variants), `GridTemplateAreas[S]`, `GridTemplateArea[S]`, `GridPlacement[S]` (zero value must stay Auto: `StyleDefault` and `OofItemStyleDefaults.GridRow` rely on it). Owner: M0-taffy-style-grid, which replaces them with the real ports.
- `impl GridContainerStyle for Style` / `impl GridItemStyle for Style` (style/mod.rs:1386-1600) are not written: M0-taffy-style-grid adds them as `StyleRef[S]` methods (rows `blocked:M0-taffy-style-grid` in ext-taffy.md).
- `CoreStyleDefaults` etc. are embeddable; embedding two Defaults structs that share a method (AlignSelf, AlignContent, IsTable) makes it ambiguous, so the implementor must define that method itself. The compiler catches it.
- `StyleDefault[S]()` builds a fresh Style per call; trait default methods call it, which costs a struct copy each time. Fine for now; revisit if profiling shows it.
