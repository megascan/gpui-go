# M0-taffy-flexbox handoff

- compute/oof.rs is already ported (internal/taffy/compute_oof.go, compute_oof_test.go): every flex container returns an OofPositioningArea, so ComputeOofLayout runs for each flex node and no flex fixture could pass without it. M0-taffy-block ports block.rs and float.rs only; its oof.rs rows in ext-taffy.md are done.
- stubs.go keeps a panicking `(*DetailedGridInfo[S]).resolveAbsoluteGridArea` (owner M0-taffy-grid-sizing, called by layoutOofBox for grid owners); the grid stage replaces it with the ported signature.
- TestXml/flex: 2512 pass, 136 n/a (flexbox_balance), 44 skip on ComputeBlockLayout, 16 on ComputeGridLayout. They pass once those stages land.
