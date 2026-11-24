# M0-taffy-compute debt

- internal/taffy/stubs.go holds panicking stubs with final signatures: `ComputeFlexboxLayout` (M0-taffy-flexbox), `ComputeBlockLayout`, `ComputeOofLayoutForArea`, `OofLayoutResult`, `oofBoxLayout`, `layoutOofBox`, `performOofLayout` (M0-taffy-block), `ComputeGridLayout` (M0-taffy-grid-sizing). Any TaffyTree layout of a node with children reaches a stub until those stages land; three taffy_tree tests are skipped and `blocked:M0-taffy-flexbox`.
- `ComputeOofLayout` (oof.rs:348) is already ported in compute_oof.go so leaf layouts (which have no positioning area) run; M0-taffy-block adds the rest of oof.rs to that file.
