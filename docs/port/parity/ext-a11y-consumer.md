# Parity: ext-a11y-consumer

Scope: accesskit_consumer 0.38.0 (`src/lib.rs`, `src/tree.rs`, `src/node.rs`, `src/iterators.rs`, `src/filters.rs`, `src/text.rs`). Milestone: M3. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_consumer-0.38.0/`.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`, `blocked:<stage>`. M3-a11y-consumer-tree ports `lib.rs` and `tree.rs` into package `a11y/consumer` (`lib.go`, `tree.go`, tests in `tree_test.go`, the `lib.rs` test fixtures in `lib_test.go`). M3-a11y-consumer-node adds the sections for `node.rs`, `iterators.rs` and `filters.rs`, M3-a11y-consumer-text the one for `text.rs`.

Notes on the port:

- `stubs.go` holds the `node.rs` items `tree.rs` needs (`NodeID`, `nodeState`, `parentAndIndex`, `Node` and the methods `Data`, `IsFocused`, `IsGraft`, `ParentID`, `Parent`, `ChildIDs`, `Children`, `ID`, `Role`, `IsDialog`, `Label`, `ActiveDescendant`), ported from node.rs so the tree.rs tests run. `Label` falls back to a panicking stub only where upstream would read `labelled_by` descendants. M3-a11y-consumer-node moves them into `node.go` and deletes `stubs.go`.
- `hashbrown` maps and sets are Go maps; both iterate in unspecified order, so change-handler call order stays unspecified as upstream.
- `Option<&Node>` handler arguments are `*Node` (nil for None); `Option<Node>` returns are `(Node, bool)`.
- `ChangeHandler` methods take `*Node`; `&mut impl ChangeHandler` parameters take the interface.
- `TreeUpdate` node data is cloned when stored, standing in for Rust's move.

## src/lib.rs: public items

Go target: `a11y/consumer/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use tree::ChangeHandler as TreeChangeHandler` | use | 11 |  | `TreeChangeHandler` | done |
| `pub use tree::State as TreeState` | use | 11 |  | `TreeState` | done |
| `pub use tree::Tree` | use | 11 |  | `Tree` | done |
| `pub use node::{Node, NodeId}` | use | 14 |  | `Node` | done |
| `pub use filters::{FilterResult, common_filter, common_filter_with_root_exception}` | use | 17 |  | `FilterResult` | blocked:M3-a11y-consumer-node |
| `pub use text::{Position as TextPosition, Range as TextRange, RangePropertyValue as TextRangePropertyValue, WeakRange as WeakTextRange}` | use | 22 |  | `TextPosition` | blocked:M3-a11y-consumer-text |

## src/lib.rs: test fixtures

Go target: `a11y/consumer/`, in `lib_test.go`. These are `#[cfg(test)]` helpers, not tests; they are listed in a test table so the checker looks them up in test files.

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests::nid` | fixture | 38 | test | `nid` | done |
| `tests::ROOT_ID` .. `tests::EMPTY_CONTAINER_3_3_IGNORED_ID` | fixture | 42 | test | `rootID` | done |
| `tests::test_tree` | fixture | 60 | test | `testTree` | done |
| `tests::test_tree_filter` | fixture | 196 | test | `testTreeFilter` | blocked:M3-a11y-consumer-node |

## src/tree.rs

Go target: `a11y/consumer/tree.go`, tests in `tree_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TreeIndex` | struct | 15 |  | `treeIndex` | done |
| `TreeIndexMap` | struct | 18 |  | `treeIndexMap` | done |
| `TreeIndexMap::get_or_create_index` | method | 25 |  | `(treeIndexMap).getOrCreateIndex` | done |
| `TreeIndexMap::get_index` | method | 34 |  | `(treeIndexMap).getIndex` | done |
| `TreeIndexMap::get_id` | method | 38 |  | `(treeIndexMap).getID` | done |
| `SubtreeState` | struct | 45 |  | `subtreeState` | done |
| `State` | struct | 51 |  | `State` | done |
| `InternalChanges` | struct | 63 |  | `internalChanges` | done |
| `State::validate_global` | method | 70 |  | `(State).validateGlobal` | done |
| `State::compute_effective_focus` | method | 85 |  | `(State).computeEffectiveFocus` | done |
| `State::update` | method | 108 |  | `(State).update` | done |
| `State::update::record_graft` | fn | 171 |  | `recordGraft` | done |
| `State::update::add_node` | fn | 190 |  | `addNode` | done |
| `State::update::traverse_unreachable` | fn | 345 |  | `traverseUnreachable` | done |
| `State::update::traverse_subtree` | fn | 386 |  | `traverseSubtree` | done |
| `State::update_host_focus_state` | method | 479 |  | `(State).updateHostFocusState` | done |
| `State::has_node` | method | 494 |  | `(State).HasNode` | done |
| `State::node_by_id` | method | 498 |  | `(State).NodeByID` | done |
| `State::node_by_tree_local_id` | method | 506 |  | `(State).NodeByTreeLocalID` | done |
| `State::root_id` | method | 515 |  | `(State).RootID` | done |
| `State::root` | method | 519 |  | `(State).Root` | done |
| `State::subtree_root` | method | 524 |  | `(State).SubtreeRoot` | done |
| `State::is_host_focused` | method | 528 |  | `(State).IsHostFocused` | done |
| `State::focus_id_in_tree` | method | 532 |  | `(State).FocusIDInTree` | done |
| `State::focus_in_tree` | method | 536 |  | `(State).FocusInTree` | done |
| `State::focus_id` | method | 540 |  | `(State).FocusID` | done |
| `State::focus` | method | 544 |  | `(State).Focus` | done |
| `State::active_dialog` | method | 551 |  | `(State).ActiveDialog` | done |
| `State::toolkit_name` | method | 562 |  | `(State).ToolkitName` | done |
| `State::toolkit_version` | method | 566 |  | `(State).ToolkitVersion` | done |
| `State::locate_node` | method | 573 |  | `(State).LocateNode` | done |
| `ChangeHandler` | trait | 584 |  | `ChangeHandler` | done |
| `ChangeHandler::node_added` | method | 585 |  | `ChangeHandler` | done |
| `ChangeHandler::node_updated` | method | 586 |  | `ChangeHandler` | done |
| `ChangeHandler::focus_moved` | method | 587 |  | `ChangeHandler` | done |
| `ChangeHandler::node_removed` | method | 588 |  | `ChangeHandler` | done |
| `Tree` | struct | 592 |  | `Tree` | done |
| `#[derive(Debug)] for Tree, State` | trait-impl | 591 |  | - | n/a:D-CDEBUG |
| `Tree::new` | assoc-fn | 598 |  | `NewTree` | done |
| `Tree::update_and_process_changes` | method | 626 |  | `(Tree).UpdateAndProcessChanges` | done |
| `Tree::update_host_focus_state_and_process_changes` | method | 637 |  | `(Tree).UpdateHostFocusStateAndProcessChanges` | done |
| `Tree::process_changes` | method | 648 |  | `(Tree).processChanges` | done |
| `Tree::state` | method | 716 |  | `(Tree).State` | done |
| `ShortNodeList` | struct | 721 |  | `shortNodeList` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tree_index_map_assigns_sequential_indices` | test | 764 |  | `TestTree_TreeIndexMapAssignsSequentialIndices` | done |
| `tree_index_map_returns_same_index_for_same_id` | test | 780 |  | `TestTree_TreeIndexMapReturnsSameIndexForSameId` | done |
| `tree_index_map_get_id_returns_correct_id` | test | 791 |  | `TestTree_TreeIndexMapGetIdReturnsCorrectId` | done |
| `tree_index_map_get_id_returns_none_for_unknown_index` | test | 804 |  | `TestTree_TreeIndexMapGetIdReturnsNoneForUnknownIndex` | done |
| `tree_index_map_get_index_returns_existing_index` | test | 811 |  | `TestTree_TreeIndexMapGetIndexReturnsExistingIndex` | done |
| `tree_index_map_get_index_returns_none_for_unknown_id` | test | 824 |  | `TestTree_TreeIndexMapGetIndexReturnsNoneForUnknownId` | done |
| `tree_index_map_get_index_does_not_create_mapping` | test | 831 |  | `TestTree_TreeIndexMapGetIndexDoesNotCreateMapping` | done |
| `init_tree_with_root_node` | test | 843 |  | `TestTree_InitTreeWithRootNode` | done |
| `init_tree_with_non_root_tree_id_panics` | test, should_panic | 860 |  | `TestTree_InitTreeWithNonRootTreeIdPanics` | done |
| `root_node_has_children` | test | 871 |  | `TestTree_RootNodeHasChildren` | done |
| `add_child_to_root_node` | test | 900 |  | `TestTree_AddChildToRootNode` | done |
| `remove_child_from_root_node` | test | 976 |  | `TestTree_RemoveChildFromRootNode` | done |
| `move_focus_between_siblings` | test | 1047 |  | `TestTree_MoveFocusBetweenSiblings` | done |
| `update_node` | test | 1132 |  | `TestTree_UpdateNode` | done |
| `no_change_update` | test | 1213 |  | `TestTree_NoChangeUpdate` | done |
| `move_node` | test | 1259 |  | `TestTree_MoveNode` | done |
| `graft_node_tracks_subtree` | test | 1360 |  | `TestTree_GraftNodeTracksSubtree` | done |
| `duplicate_graft_parent_panics` | test, should_panic | 1387 |  | `TestTree_DuplicateGraftParentPanics` | done |
| `reparent_subtree_by_removing_old_graft` | test | 1414 |  | `TestTree_ReparentSubtreeByRemovingOldGraft` | done |
| `reparent_subtree_by_clearing_old_graft_tree_id` | test | 1464 |  | `TestTree_ReparentSubtreeByClearingOldGraftTreeId` | done |
| `duplicate_graft_parent_on_update_panics` | test, should_panic | 1511 |  | `TestTree_DuplicateGraftParentOnUpdatePanics` | done |
| `graft_root_tree_panics` | test, should_panic | 1547 |  | `TestTree_GraftRootTreePanics` | done |
| `graft_root_tree_on_update_panics` | test, should_panic | 1570 |  | `TestTree_GraftRootTreeOnUpdatePanics` | done |
| `node_by_tree_local_id_finds_root_tree_node` | test | 1604 |  | `TestTree_NodeByTreeLocalIdFindsRootTreeNode` | done |
| `node_by_tree_local_id_finds_subtree_node` | test | 1636 |  | `TestTree_NodeByTreeLocalIdFindsSubtreeNode` | done |
| `node_by_tree_local_id_returns_none_for_unknown_tree_id` | test | 1694 |  | `TestTree_NodeByTreeLocalIdReturnsNoneForUnknownTreeId` | done |
| `node_by_tree_local_id_returns_none_for_unknown_local_id` | test | 1711 |  | `TestTree_NodeByTreeLocalIdReturnsNoneForUnknownLocalId` | done |
| `subtree_root_parent_is_graft_when_graft_exists_first` | test | 1728 |  | `TestTree_SubtreeRootParentIsGraftWhenGraftExistsFirst` | done |
| `subtree_push_without_graft_panics` | test, should_panic | 1767 |  | `TestTree_SubtreePushWithoutGraftPanics` | done |
| `subtree_update_without_tree_data_panics` | test, should_panic | 1787 |  | `TestTree_SubtreeUpdateWithoutTreeDataPanics` | done |
| `subtree_nodes_removed_when_graft_removed` | test | 1817 |  | `TestTree_SubtreeNodesRemovedWhenGraftRemoved` | done |
| `subtree_nodes_removed_when_graft_tree_id_cleared` | test | 1897 |  | `TestTree_SubtreeNodesRemovedWhenGraftTreeIdCleared` | done |
| `graft_node_has_no_children_when_subtree_not_pushed` | test | 1949 |  | `TestTree_GraftNodeHasNoChildrenWhenSubtreeNotPushed` | done |
| `graft_node_with_children_panics` | test, should_panic | 1976 |  | `TestTree_GraftNodeWithChildrenPanics` | done |
| `node_added_called_when_subtree_pushed` | test | 2000 |  | `TestTree_NodeAddedCalledWhenSubtreePushed` | done |
| `node_removed_called_when_graft_removed` | test | 2059 |  | `TestTree_NodeRemovedCalledWhenGraftRemoved` | done |
| `node_updated_called_when_subtree_reparented` | test | 2132 |  | `TestTree_NodeUpdatedCalledWhenSubtreeReparented` | done |
| `focus_moved_called_when_focus_moves_to_subtree` | test | 2202 |  | `TestTree_FocusMovedCalledWhenFocusMovesToSubtree` | done |
| `focus_moved_called_when_subtree_focus_changes` | test | 2269 |  | `TestTree_FocusMovedCalledWhenSubtreeFocusChanges` | done |
| `nested_subtree_focus_follows_graft_chain` | test | 2352 |  | `TestTree_NestedSubtreeFocusFollowsGraftChain` | done |
| `nested_subtree_focus_update_changes_effective_focus` | test | 2426 |  | `TestTree_NestedSubtreeFocusUpdateChangesEffectiveFocus` | done |
| `removing_nested_subtree_while_intermediate_focus_on_graft_panics` | test, should_panic | 2504 |  | `TestTree_RemovingNestedSubtreeWhileIntermediateFocusOnGraftPanics` | done |
| `nested_subtree_root_lookup_for_focus_only_update` | test | 2561 |  | `TestTree_NestedSubtreeRootLookupForFocusOnlyUpdate` | done |
| `subtree_root_change_updates_graft_and_parent` | test | 2631 |  | `TestTree_SubtreeRootChangeUpdatesGraftAndParent` | done |
| `subtree_root_change_to_existing_child` | test | 2727 |  | `TestTree_SubtreeRootChangeToExistingChild` | done |
| `subtree_root_change_to_new_parent_of_old_root` | test | 2841 |  | `TestTree_SubtreeRootChangeToNewParentOfOldRoot` | done |
| `subtree_update_without_tree_preserves_root` | test | 2943 |  | `TestTree_SubtreeUpdateWithoutTreePreservesRoot` | done |
| `focus_returns_focused_node` | test | 3035 |  | `TestTree_FocusReturnsFocusedNode` | done |
| `focus_returns_active_descendant` | test | 3054 |  | `TestTree_FocusReturnsActiveDescendant` | done |
| `focus_moved_when_active_descendant_changes` | test | 3079 |  | `TestTree_FocusMovedWhenActiveDescendantChanges` | done |
| `toolkit_info_is_reported_when_both_name_and_version_are_set` | test | 3156 |  | `TestTree_ToolkitInfoIsReportedWhenBothNameAndVersionAreSet` | done |
| `toolkit_version_is_none_when_only_name_is_set` | test | 3164 |  | `TestTree_ToolkitVersionIsNoneWhenOnlyNameIsSet` | done |
| `toolkit_name_falls_back_to_accesskit_when_unset` | test | 3172 |  | `TestTree_ToolkitNameFallsBackToAccesskitWhenUnset` | done |
| `toolkit_version_is_suppressed_when_name_is_unset` | test | 3180 |  | `TestTree_ToolkitVersionIsSuppressedWhenNameIsUnset` | done |

## Deviations registered here

| Id | Meaning | Owner |
|---|---|---|
| `D-CDEBUG` | Rust's derived `Debug` for `Tree` and `State` has no Go counterpart; no caller formats them. | M3-a11y-consumer-tree |
