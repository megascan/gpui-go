# M0-style debt

- `StyleFromRefinement`, `TextStyleFromRefinement` (refinegen) fill unset fields with Go zero values, not Rust `Default` (`FontWeight` 0 instead of 400, `FontFamily` "" instead of ".SystemUIFont", `Display` Flex matches). Upstream's `from_refinement` is only reached through `Refineable::from_cascade`; callers that need upstream defaults must refine `DefaultStyle()` / `DefaultTextStyle()`. Fix belongs in refinegen (a `Default<T>()` hook), not owned here.
- `CombineHighlights` folds the active highlights in ascending insertion order. Upstream iterates an FxHashSet<usize>, whose order the upstream test depends on and which is ascending for these small ids; any divergence would only show with many overlapping highlights.
- `HighlightStyle.Hash` writes the fields through fmt, not Rust's `Hash` byte stream; equal values hash equally, which is all callers rely on.
- `cfg(debug_assertions)` items (`DebugBelow`, `Style.Debug`, `Style.DebugBelow`, `Styled.Debug`, `Styled.DebugBelow`) are always compiled (Go has no debug/release split).
- JSON: unit enums use serde variant names; `TextOverflow` uses serde's externally tagged form. `Fill` (sealed interface) has no serde-shaped JSON (`{"Color": ...}`) and the refinements derive no JSON schema yet (upstream `JsonSchema`); nothing in the port serializes styles today.
- The `From<gpui enum> for taffy::style::*` impls of style.rs:1249-1329 are not here: M0-element writes them in taffy.go (bridge-file rule).
