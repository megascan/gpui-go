# M0-taffy-compute handoff

- M0-taffy-flexbox: un-skip `TestTreeTaffyTree_RemoveNodeMarksParentDirty`, `TestTreeTaffyTree_MarkDirty`, `TestTreeTaffyTree_MakeSureLayoutLocationIsTopLeft` (tree_taffy_tree_test.go) and mark their ext-taffy rows done.
- M0-taffy-block: compute_oof.go exists and contains the ported `ComputeOofLayout`; move `OofLayoutResult`, `ComputeOofLayoutForArea`, `oofBoxLayout`, `layoutOofBox`, `performOofLayout` from stubs.go into it with their final bodies (signatures are what compute_mod.go calls).
- Exported algorithm entry points are `ComputeFlexboxLayout` / `ComputeBlockLayout` / `ComputeGridLayout[S]` (upstream `pub`, ext-taffy rows), not the lower-case names in the STAGES brief.
