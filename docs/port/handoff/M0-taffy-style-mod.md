# M0-taffy-style-mod handoff

- To M0-taffy-style-grid: `Style` is `Style[S CheapCloneStr]` (`CheapCloneStr` = `~string`); the trait impls for Style live on `StyleRef[S]` (`s.Ref()`), see D-TAFFY-TRAITS. Add `GridContainerStyle` / `GridItemStyle[S]` methods there and replace the grid placeholders at the end of stubs.go (keep `GridPlacement`'s zero value = Auto, `GridAutoFlowRow` = 0).
- `FlexDirection.IsRow` from the previous stage's stub is now the unexported `isRow` (upstream is `pub(crate)`); geometry.go call sites were updated.
