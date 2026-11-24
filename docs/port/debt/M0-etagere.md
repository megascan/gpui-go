# M0-etagere debt

- E-COALESCE-OOB: upstream `coalesce_shelves` (`etagere-0.2.15/src/bucketed.rs:441-447`) reads `shelves[shelf_index + i]` before its bound check, so an empty last shelf that is too narrow for the request makes `Allocate` panic with an index-out-of-range error instead of returning none. Kept for parity; gpui's atlases can reach it only when the atlas is full and its last shelf is empty. Fix upstream-compatibly by moving the bound check above the read if gpui hits it.
- E-DEBUGASSERT: upstream `debug_assert!` checks are always on here (they are debug-build-only in Rust release builds). They never fire on valid use.
