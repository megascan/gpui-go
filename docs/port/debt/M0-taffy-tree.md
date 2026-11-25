# M0-taffy-tree debt

- `internal/taffy/stubs.go`: `DetailedGridInfo[S]` (M0-taffy-grid-sizing) and `BlockContext` (M0-taffy-block) are empty placeholder structs; the owners replace them. `DetailedLayoutInfo` has no `Equal` until `DetailedGridInfo` exists.
- `CacheModeChangeEvictions` is one process-wide atomic counter (upstream: thread-local), so a test asserting it must not run in parallel with other layout tests (D-TAFFY-CLONE).
- `NodeID` <-> slotmap `DefaultKey` conversions are left to M0-taffy-compute (rows `blocked:M0-taffy-compute`).
