# M0-sumtree debt

- `Arc::make_mut` is ported as unconditional copy-on-write (`SumTree.makeMut` clones the node and its slices on every mutation), since Go cannot see whether a node is shared. Behaviour matches; mutation of unshared nodes costs an extra copy per level. Revisit if list/sum_tree benches show it.
- Keyed operations (`InsertOrReplace`, `Remove`, `Edit`, `Get`) are generic methods taking the key type `K`; `T` implementing `Key() K` is checked at run time (D-COND). `InsertOrReplace` needs `K` spelled out (`tree.InsertOrReplace[K](item, cx)`).
- `SumTreeFromIter` / `Extend` take slices, so upstream's `.fuse()` behaviour in test_from_iter is moot; the test keeps the iterator-restart scenario by collecting first.
