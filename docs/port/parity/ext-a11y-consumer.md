# Parity: ext-a11y-consumer

Scope: accesskit_consumer 0.38.0 (`src/lib.rs`, `src/tree.rs`, `src/node.rs`, `src/iterators.rs`, `src/filters.rs`, `src/text.rs`). Milestone: M3. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_consumer-0.38.0/`.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`, `blocked:<stage>`. M3-a11y-consumer-tree ports `lib.rs` and `tree.rs` into package `a11y/consumer` (`lib.go`, `tree.go`, tests in `tree_test.go`, the `lib.rs` test fixtures in `lib_test.go`). M3-a11y-consumer-node adds the sections for `node.rs`, `iterators.rs` and `filters.rs`, M3-a11y-consumer-text the one for `text.rs`.

Notes on the port:

- The `text.rs` items `node.rs` needs (`text_node_filter`, `Node::text_runs`, `Node::supports_text_ranges`, `document_range().write_text(..)`) live in `text.go`; the package has no `stubs.go`.
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
| `pub use filters::{FilterResult, common_filter, common_filter_with_root_exception}` | use | 17 |  | `FilterResult` | done |
| `pub use text::{Position as TextPosition, Range as TextRange, RangePropertyValue as TextRangePropertyValue, WeakRange as WeakTextRange}` | use | 22 |  | `TextPosition` | done |

## src/lib.rs: test fixtures

Go target: `a11y/consumer/`, in `lib_test.go`. These are `#[cfg(test)]` helpers, not tests; they are listed in a test table so the checker looks them up in test files.

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests::nid` | fixture | 38 | test | `nid` | done |
| `tests::ROOT_ID` .. `tests::EMPTY_CONTAINER_3_3_IGNORED_ID` | fixture | 42 | test | `rootID` | done |
| `tests::test_tree` | fixture | 60 | test | `testTree` | done |
| `tests::test_tree_filter` | fixture | 196 | test | `testTreeFilter` | done |

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


## src/node.rs

Go target: `a11y/consumer/node.go`, tests in `node_test.go`. Double-ended iterators become iterator structs with `Next`, `NextBack`, `Len` (exact-size ones), `All` and `Backward` (iter.Seq); `impl Fn(&Node) -> FilterResult` is `func(*Node) FilterResult`; `fmt::Write` is `io.StringWriter`. `Node::controls` returns `iter.Seq[Node]` (never reversed by callers).

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `NodeId` | struct | 30 |  | `NodeID` | done |
| `ParentAndIndex` | struct | 55 |  | `parentAndIndex` | done |
| `NodeState` | struct | 58 |  | `nodeState` | done |
| `Node` | struct | 64 |  | `Node` | done |
| `NodeId::new` | assoc-fn | 33 |  | `newNodeID` | done |
| `NodeId::with_same_tree` | method | 37 |  | `(NodeID).withSameTree` | done |
| `NodeId::to_components` | method | 41 |  | `(NodeID).toComponents` | done |
| `impl From<NodeId> for u128` | trait-impl | 46 |  | `(NodeID).ToU128` | done |
| `Node::data` | method | 71 |  | `(Node).Data` | done |
| `Node::is_focused` | method | 75 |  | `(Node).IsFocused` | done |
| `Node::is_focused_in_tree` | method | 93 |  | `(Node).IsFocusedInTree` | done |
| `Node::is_focusable` | method | 97 |  | `(Node).IsFocusable` | done |
| `Node::is_root` | method | 101 |  | `(Node).IsRoot` | done |
| `Node::is_graft` | method | 108 |  | `(Node).IsGraft` | done |
| `Node::parent_id` | method | 112 |  | `(Node).ParentID` | done |
| `Node::parent` | method | 119 |  | `(Node).Parent` | done |
| `Node::filtered_parent` | method | 124 |  | `(Node).FilteredParent` | done |
| `Node::parent_and_index` | method | 134 |  | `(Node).ParentAndIndex` | done |
| `Node::graft_child_id` | method | 144 |  | `(Node).graftChildID` | done |
| `Node::child_ids` | method | 151 |  | `(Node).ChildIDs` | done |
| `Node::children` | method | 164 |  | `(Node).Children` | done |
| `Node::filtered_children` | method | 172 |  | `(Node).FilteredChildren` | done |
| `Node::following_sibling_ids` | method | 179 |  | `(Node).FollowingSiblingIDs` | done |
| `Node::following_siblings` | method | 185 |  | `(Node).FollowingSiblings` | done |
| `Node::following_filtered_siblings` | method | 193 |  | `(Node).FollowingFilteredSiblings` | done |
| `Node::preceding_sibling_ids` | method | 200 |  | `(Node).PrecedingSiblingIDs` | done |
| `Node::preceding_siblings` | method | 206 |  | `(Node).PrecedingSiblings` | done |
| `Node::preceding_filtered_siblings` | method | 214 |  | `(Node).PrecedingFilteredSiblings` | done |
| `Node::deepest_first_child` | method | 221 |  | `(Node).DeepestFirstChild` | done |
| `Node::deepest_first_filtered_child` | method | 229 |  | `(Node).DeepestFirstFilteredChild` | done |
| `Node::deepest_last_child` | method | 240 |  | `(Node).DeepestLastChild` | done |
| `Node::deepest_last_filtered_child` | method | 248 |  | `(Node).DeepestLastFilteredChild` | done |
| `Node::is_descendant_of` | method | 259 |  | `(Node).IsDescendantOf` | done |
| `Node::direct_transform` | method | 274 |  | `(Node).DirectTransform` | done |
| `Node::transform` | method | 282 |  | `(Node).Transform` | done |
| `Node::relative_transform` | method | 292 |  | `(Node).relativeTransform` | done |
| `Node::raw_bounds` | method | 305 |  | `(Node).RawBounds` | done |
| `Node::has_bounds` | method | 309 |  | `(Node).HasBounds` | done |
| `Node::bounding_box` | method | 315 |  | `(Node).BoundingBox` | done |
| `Node::bounding_box_in_coordinate_space` | method | 321 |  | `(Node).boundingBoxInCoordinateSpace` | done |
| `Node::hit_test` | method | 327 |  | `(Node).hitTest` | done |
| `Node::node_at_point` | method | 372 |  | `(Node).NodeAtPoint` | done |
| `Node::id` | method | 380 |  | `(Node).ID` | done |
| `Node::locate` | method | 384 |  | `(Node).Locate` | done |
| `Node::role` | method | 388 |  | `(Node).Role` | done |
| `Node::role_description` | method | 392 |  | `(Node).RoleDescription` | done |
| `Node::has_role_description` | method | 396 |  | `(Node).HasRoleDescription` | done |
| `Node::is_live_atomic` | method | 400 |  | `(Node).IsLiveAtomic` | done |
| `Node::is_busy` | method | 404 |  | `(Node).IsBusy` | done |
| `Node::column_index_text` | method | 408 |  | `(Node).ColumnIndexText` | done |
| `Node::row_index_text` | method | 412 |  | `(Node).RowIndexText` | done |
| `Node::braille_label` | method | 416 |  | `(Node).BrailleLabel` | done |
| `Node::has_braille_label` | method | 420 |  | `(Node).HasBrailleLabel` | done |
| `Node::braille_role_description` | method | 424 |  | `(Node).BrailleRoleDescription` | done |
| `Node::has_braille_role_description` | method | 428 |  | `(Node).HasBrailleRoleDescription` | done |
| `Node::aria_current` | method | 432 |  | `(Node).AriaCurrent` | done |
| `Node::has_popup` | method | 436 |  | `(Node).HasPopup` | done |
| `Node::is_hidden` | method | 440 |  | `(Node).IsHidden` | done |
| `Node::level` | method | 444 |  | `(Node).Level` | done |
| `Node::is_disabled` | method | 448 |  | `(Node).IsDisabled` | done |
| `Node::is_read_only` | method | 452 |  | `(Node).IsReadOnly` | done |
| `Node::is_read_only_or_disabled` | method | 461 |  | `(Node).IsReadOnlyOrDisabled` | done |
| `Node::toggled` | method | 465 |  | `(Node).Toggled` | done |
| `Node::numeric_value` | method | 469 |  | `(Node).NumericValue` | done |
| `Node::min_numeric_value` | method | 473 |  | `(Node).MinNumericValue` | done |
| `Node::max_numeric_value` | method | 477 |  | `(Node).MaxNumericValue` | done |
| `Node::numeric_value_step` | method | 481 |  | `(Node).NumericValueStep` | done |
| `Node::numeric_value_jump` | method | 485 |  | `(Node).NumericValueJump` | done |
| `Node::clips_children` | method | 489 |  | `(Node).ClipsChildren` | done |
| `Node::scroll_x` | method | 493 |  | `(Node).ScrollX` | done |
| `Node::scroll_x_min` | method | 497 |  | `(Node).ScrollXMin` | done |
| `Node::scroll_x_max` | method | 501 |  | `(Node).ScrollXMax` | done |
| `Node::scroll_y` | method | 505 |  | `(Node).ScrollY` | done |
| `Node::scroll_y_min` | method | 509 |  | `(Node).ScrollYMin` | done |
| `Node::scroll_y_max` | method | 513 |  | `(Node).ScrollYMax` | done |
| `Node::fetch_inherited_property` | method | 517 |  | `fetchInheritedProperty` | done |
| `Node::fetch_inherited_flag` | method | 531 |  | `(Node).fetchInheritedFlag` | done |
| `Node::is_text_input` | method | 545 |  | `(Node).IsTextInput` | done |
| `Node::is_multiline` | method | 566 |  | `(Node).IsMultiline` | done |
| `Node::orientation` | method | 570 |  | `(Node).Orientation` | done |
| `Node::is_dialog` | method | 582 |  | `(Node).IsDialog` | done |
| `Node::is_modal` | method | 586 |  | `(Node).IsModal` | done |
| `Node::is_clickable` | method | 599 |  | `(Node).IsClickable` | done |
| `Node::is_selectable` | method | 603 |  | `(Node).IsSelectable` | done |
| `Node::is_multiselectable` | method | 608 |  | `(Node).IsMultiselectable` | done |
| `Node::size_of_set_from_container` | method | 612 |  | `(Node).SizeOfSetFromContainer` | done |
| `Node::size_of_set` | method | 626 |  | `(Node).SizeOfSet` | done |
| `Node::position_in_set` | method | 631 |  | `(Node).PositionInSet` | done |
| `Node::sort_direction` | method | 636 |  | `(Node).SortDirection` | done |
| `Node::supports_toggle` | method | 640 |  | `(Node).SupportsToggle` | done |
| `Node::supports_expand_collapse` | method | 644 |  | `(Node).SupportsExpandCollapse` | done |
| `Node::is_invocable` | method | 653 |  | `(Node).IsInvocable` | done |
| `Node::supports_action` | method | 670 |  | `(Node).SupportsAction` | done |
| `Node::supports_increment` | method | 684 |  | `(Node).SupportsIncrement` | done |
| `Node::supports_decrement` | method | 688 |  | `(Node).SupportsDecrement` | done |
| `Node::labelled_by` | method | 702 |  | `(Node).LabelledBy` | done |
| `Node::label_comes_from_value` | method | 727 |  | `(Node).LabelComesFromValue` | done |
| `Node::label` | method | 731 |  | `(Node).Label` | done |
| `Node::has_label` | method | 736 |  | `(Node).HasLabel` | done |
| `Node::write_label_direct` | method | 749 |  | `(Node).writeLabelDirect` | done |
| `Node::write_label` | method | 758 |  | `(Node).WriteLabel` | done |
| `Node::description` | method | 779 |  | `(Node).Description` | done |
| `Node::has_description` | method | 785 |  | `(Node).HasDescription` | done |
| `Node::url` | method | 789 |  | `(Node).URL` | done |
| `Node::supports_url` | method | 793 |  | `(Node).SupportsURL` | done |
| `Node::is_empty_text_input` | method | 804 |  | `(Node).isEmptyTextInput` | done |
| `Node::placeholder` | method | 817 |  | `(Node).Placeholder` | done |
| `Node::value` | method | 823 |  | `(Node).Value` | done |
| `Node::write_value` | method | 828 |  | `(Node).WriteValue` | done |
| `Node::has_value` | method | 840 |  | `(Node).HasValue` | done |
| `Node::is_read_only_supported` | method | 844 |  | `(Node).IsReadOnlySupported` | done |
| `Node::should_have_read_only_state_by_default` | method | 864 |  | `(Node).ShouldHaveReadOnlyStateByDefault` | done |
| `Node::is_required` | method | 885 |  | `(Node).IsRequired` | done |
| `Node::live` | method | 889 |  | `(Node).Live` | done |
| `Node::is_selected` | method | 895 |  | `(Node).IsSelected` | done |
| `Node::is_touch_transparent` | method | 899 |  | `(Node).IsTouchTransparent` | done |
| `Node::is_item_like` | method | 903 |  | `(Node).IsItemLike` | done |
| `Node::is_container_with_selectable_children` | method | 921 |  | `(Node).IsContainerWithSelectableChildren` | done |
| `Node::controls` | method | 940 |  | `(Node).Controls` | done |
| `Node::active_descendant` | method | 949 |  | `(Node).ActiveDescendant` | done |
| `Node::raw_text_selection` | method | 956 |  | `(Node).RawTextSelection` | done |
| `Node::raw_value` | method | 960 |  | `(Node).RawValue` | done |
| `Node::author_id` | method | 964 |  | `(Node).AuthorID` | done |
| `Node::class_name` | method | 968 |  | `(Node).ClassName` | done |
| `Node::index_path` | method | 972 |  | `(Node).IndexPath` | done |
| `Node::relative_index_path` | method | 976 |  | `(Node).RelativeIndexPath` | done |
| `Node::first_filtered_child` | method | 988 |  | `(Node).firstFilteredChild` | done |
| `Node::last_filtered_child` | method | 1007 |  | `(Node).lastFilteredChild` | done |
| `Node::selection_container` | method | 1026 |  | `(Node).SelectionContainer` | done |
| `Node::items` | method | 1036 |  | `(Node).Items` | done |
| `SpacePrefixingWriter::write_prefix_if_needed` | method | 1054 |  | `(spacePrefixingWriter).writePrefixIfNeeded` | done |
| `SpacePrefixingWriter::write_str` | method | 1064 |  | `(spacePrefixingWriter).WriteString` | done |
| `SpacePrefixingWriter::write_char` | method | 1069 |  | `(spacePrefixingWriter).WriteString` | done |
| `descendant_label_filter` | fn | 693 |  | `descendantLabelFilter` | done |
| `SpacePrefixingWriter` | struct | 1048 |  | `spacePrefixingWriter` | done |
| `#[derive(Debug)] for NodeId, ParentAndIndex, NodeState, Node` | trait-impl | 29 |  | - | n/a:D-CDEBUG |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `parent_and_index` | test | 1086 |  | `TestNode_ParentAndIndex` | done |
| `deepest_first_child` | test | 1116 |  | `TestNode_DeepestFirstChild` | done |
| `filtered_parent` | test | 1149 |  | `TestNode_FilteredParent` | done |
| `deepest_first_filtered_child` | test | 1171 |  | `TestNode_DeepestFirstFilteredChild` | done |
| `deepest_last_child` | test | 1200 |  | `TestNode_DeepestLastChild` | done |
| `deepest_last_filtered_child` | test | 1233 |  | `TestNode_DeepestLastFilteredChild` | done |
| `is_descendant_of` | test | 1273 |  | `TestNode_IsDescendantOf` | done |
| `is_root` | test | 1310 |  | `TestNode_IsRoot` | done |
| `locate` | test | 1323 |  | `TestNode_Locate` | done |
| `bounding_box` | test | 1332 |  | `TestNode_BoundingBox` | done |
| `node_at_point` | test | 1368 |  | `TestNode_NodeAtPoint` | done |
| `first_filtered_child` | test | 1399 |  | `TestNode_FirstFilteredChild` | done |
| `last_filtered_child` | test | 1428 |  | `TestNode_LastFilteredChild` | done |
| `transform` | test | 1450 |  | `TestNode_Transform` | done |
| `relative_transform` | test | 1473 |  | `TestNode_RelativeTransform` | done |
| `no_label_or_labelled_by` | test | 1491 |  | `TestNode_NoLabelOrLabelledBy` | done |
| `label_from_labelled_by` | test | 1513 |  | `TestNode_LabelFromLabelledBy` | done |
| `label_from_descendant_label` | test | 1563 |  | `TestNode_LabelFromDescendantLabel` | done |
| `placeholder_should_be_exposed_on_empty_text_input` | test | 1753 |  | `TestNode_PlaceholderShouldBeExposedOnEmptyTextInput` | done |
| `placeholder_should_be_ignored_on_non_empty_text_input` | test | 1821 |  | `TestNode_PlaceholderShouldBeIgnoredOnNonEmptyTextInput` | done |
| `hidden_flag_should_be_inherited` | test | 1890 |  | `TestNode_HiddenFlagShouldBeInherited` | done |
| `node_id::new_and_to_components_round_trip` | test | 1928 |  | `TestNode_NodeId_NewAndToComponentsRoundTrip` | done |
| `node_id::with_same_tree_preserves_tree_index` | test | 1938 |  | `TestNode_NodeId_WithSameTreePreservesTreeIndex` | done |
| `node_id::into_u128` | test | 1952 |  | `TestNode_NodeId_IntoU128` | done |
| `node_id::equality` | test | 1962 |  | `TestNode_NodeId_Equality` | done |
| `is_focused_when_node_has_focus` | test | 1975 |  | `TestNode_IsFocusedWhenNodeHasFocus` | done |
| `is_focused_when_node_does_not_have_focus` | test | 2002 |  | `TestNode_IsFocusedWhenNodeDoesNotHaveFocus` | done |
| `is_focused_active_descendant_is_focused` | test | 2030 |  | `TestNode_IsFocusedActiveDescendantIsFocused` | done |
| `is_focused_node_with_active_descendant_is_not_focused` | test | 2059 |  | `TestNode_IsFocusedNodeWithActiveDescendantIsNotFocused` | done |

## src/iterators.rs

Go target: `a11y/consumer/iterators.go`, tests in `iterators_test.go`. `children()`, `following_siblings()` and `preceding_siblings()` (a `map` over an ID iterator) return `*Nodes`.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ChildIds` | enum | 22 |  | `ChildIDs` | done |
| `ChildIds::next` | method | 33 |  | `(ChildIDs).Next` | done |
| `ChildIds::size_hint` | method | 46 |  | `(ChildIDs).Len` | done |
| `ChildIds::next_back` | method | 53 |  | `(ChildIDs).NextBack` | done |
| `ChildIds::len` | method | 67 |  | `(ChildIDs).Len` | done |
| `FollowingSiblings` | struct | 80 |  | `FollowingSiblings` | done |
| `FollowingSiblings::new` | assoc-fn | 89 |  | `newFollowingSiblings` | done |
| `FollowingSiblings::next` | method | 121 |  | `(FollowingSiblings).Next` | done |
| `FollowingSiblings::size_hint` | method | 137 |  | `(FollowingSiblings).Len` | done |
| `FollowingSiblings::next_back` | method | 147 |  | `(FollowingSiblings).NextBack` | done |
| `PrecedingSiblings` | struct | 169 |  | `PrecedingSiblings` | done |
| `PrecedingSiblings::new` | assoc-fn | 178 |  | `newPrecedingSiblings` | done |
| `PrecedingSiblings::next` | method | 206 |  | `(PrecedingSiblings).Next` | done |
| `PrecedingSiblings::size_hint` | method | 224 |  | `(PrecedingSiblings).Len` | done |
| `PrecedingSiblings::next_back` | method | 234 |  | `(PrecedingSiblings).NextBack` | done |
| `next_filtered_sibling` | fn | 254 |  | `nextFilteredSibling` | done |
| `previous_filtered_sibling` | fn | 296 |  | `previousFilteredSibling` | done |
| `FollowingFilteredSiblings` | struct | 342 |  | `FollowingFilteredSiblings` | done |
| `FollowingFilteredSiblings::new` | assoc-fn | 350 |  | `newFollowingFilteredSiblings` | done |
| `FollowingFilteredSiblings::next` | method | 367 |  | `(FollowingFilteredSiblings).Next` | done |
| `FollowingFilteredSiblings::next_back` | method | 389 |  | `(FollowingFilteredSiblings).NextBack` | done |
| `PrecedingFilteredSiblings` | struct | 412 |  | `PrecedingFilteredSiblings` | done |
| `PrecedingFilteredSiblings::new` | assoc-fn | 420 |  | `newPrecedingFilteredSiblings` | done |
| `PrecedingFilteredSiblings::next` | method | 437 |  | `(PrecedingFilteredSiblings).Next` | done |
| `PrecedingFilteredSiblings::next_back` | method | 459 |  | `(PrecedingFilteredSiblings).NextBack` | done |
| `FilteredChildren` | struct | 482 |  | `FilteredChildren` | done |
| `FilteredChildren::new` | assoc-fn | 490 |  | `newFilteredChildren` | done |
| `FilteredChildren::next` | method | 506 |  | `(FilteredChildren).Next` | done |
| `FilteredChildren::next_back` | method | 519 |  | `(FilteredChildren).NextBack` | done |
| `LabelledBy` | enum | 536 |  | `LabelledBy` | done |
| `LabelledBy::next` | method | 547 |  | `(LabelledBy).Next` | done |
| `LabelledBy::size_hint` | method | 561 |  | - | n/a:D-CSIZEHINT |
| `LabelledBy::next_back` | method | 570 |  | `(LabelledBy).NextBack` | done |
| `impl FusedIterator for ChildIds, FollowingSiblings, PrecedingSiblings, FollowingFilteredSiblings, PrecedingFilteredSiblings, FilteredChildren, LabelledBy` | trait-impl | 75 |  | - | n/a:D-CSIZEHINT |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `following_siblings` | test | 599 |  | `TestIterators_FollowingSiblings` | done |
| `following_siblings_reversed` | test | 643 |  | `TestIterators_FollowingSiblingsReversed` | done |
| `preceding_siblings` | test | 677 |  | `TestIterators_PrecedingSiblings` | done |
| `preceding_siblings_reversed` | test | 717 |  | `TestIterators_PrecedingSiblingsReversed` | done |
| `following_filtered_siblings` | test | 747 |  | `TestIterators_FollowingFilteredSiblings` | done |
| `following_filtered_siblings_reversed` | test | 785 |  | `TestIterators_FollowingFilteredSiblingsReversed` | done |
| `preceding_filtered_siblings` | test | 825 |  | `TestIterators_PrecedingFilteredSiblings` | done |
| `preceding_filtered_siblings_reversed` | test | 863 |  | `TestIterators_PrecedingFilteredSiblingsReversed` | done |
| `filtered_children` | test | 903 |  | `TestIterators_FilteredChildren` | done |
| `filtered_children_reversed` | test | 938 |  | `TestIterators_FilteredChildrenReversed` | done |
| `graft_node_without_subtree_has_no_filtered_children` | test | 974 |  | `TestIterators_GraftNodeWithoutSubtreeHasNoFilteredChildren` | done |
| `filtered_children_crosses_subtree_boundary` | test | 1002 |  | `TestIterators_FilteredChildrenCrossesSubtreeBoundary` | done |

## src/filters.rs

Go target: `a11y/consumer/filters.go`, tests in `filters_test.go`.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FilterResult` | enum | 11 |  | `FilterResult` | done |
| `common_filter_base` | fn | 17 |  | `commonFilterBase` | done |
| `common_filter_without_parent_checks` | fn | 39 |  | `commonFilterWithoutParentChecks` | done |
| `is_first_sibling_in_parent_bbox` | fn | 43 |  | `isFirstSiblingInParentBbox` | done |
| `common_filter` | fn | 54 |  | `CommonFilter` | done |
| `common_filter_with_root_exception` | fn | 94 |  | `CommonFilterWithRootException` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests::assert_filter_result` | fixture | 113 | test | `assertFilterResult` | done |
| `tests::clipped_children_test_tree` | fixture | 271 | test | `clippedChildrenTestTree` | done |
| `normal` | test | 121 |  | `TestFilters_Normal` | done |
| `hidden` | test | 140 |  | `TestFilters_Hidden` | done |
| `hidden_but_focused` | test | 163 |  | `TestFilters_HiddenButFocused` | done |
| `generic_container` | test | 186 |  | `TestFilters_GenericContainer` | done |
| `hidden_parent` | test | 210 |  | `TestFilters_HiddenParent` | done |
| `hidden_parent_but_focused` | test | 231 |  | `TestFilters_HiddenParentButFocused` | done |
| `text_run` | test | 252 |  | `TestFilters_TextRun` | done |
| `clipped_children_excluded_above` | test | 356 |  | `TestFilters_ClippedChildrenExcludedAbove` | done |
| `clipped_children_included_above` | test | 363 |  | `TestFilters_ClippedChildrenIncludedAbove` | done |
| `clipped_children_hidden` | test | 369 |  | `TestFilters_ClippedChildrenHidden` | done |
| `clipped_children_visible` | test | 376 |  | `TestFilters_ClippedChildrenVisible` | done |
| `clipped_children_included_below` | test | 384 |  | `TestFilters_ClippedChildrenIncludedBelow` | done |
| `clipped_children_excluded_below` | test | 390 |  | `TestFilters_ClippedChildrenExcludedBelow` | done |
| `graft_node` | test | 397 |  | `TestFilters_GraftNode` | done |

## src/text.rs

Go target: `a11y/consumer/text.go`, tests in `text_test.go`. `Position`, `Range` and `WeakRange` keep upstream's names; the crate-root aliases are Go type aliases. `RangePropertyValue::map` (on the `Option<T>` instantiation, N7) is `RangePropertyValueMap`. The macro-generated property modules become `TestText_<Property>_<Test>`; their row line is the test's line inside the macro.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `InnerPosition` | struct | 16 |  | `innerPosition` | done |
| `InnerPosition::upgrade` | assoc-fn | 22 |  | `upgradeInnerPosition` | done |
| `InnerPosition::clamped_upgrade` | assoc-fn | 37 |  | `clampedUpgradeInnerPosition` | done |
| `InnerPosition::is_run_start` | method | 55 |  | `(innerPosition).isRunStart` | done |
| `InnerPosition::is_line_start` | method | 59 |  | `(innerPosition).isLineStart` | done |
| `InnerPosition::is_run_end` | method | 63 |  | `(innerPosition).isRunEnd` | done |
| `InnerPosition::is_line_end` | method | 67 |  | `(innerPosition).isLineEnd` | done |
| `InnerPosition::is_paragraph_end` | method | 71 |  | `(innerPosition).isParagraphEnd` | done |
| `InnerPosition::is_document_start` | method | 75 |  | `(innerPosition).isDocumentStart` | done |
| `InnerPosition::is_document_end` | method | 79 |  | `(innerPosition).isDocumentEnd` | done |
| `InnerPosition::biased_to_start` | method | 83 |  | `(innerPosition).biasedToStart` | done |
| `InnerPosition::biased_to_end` | method | 95 |  | `(innerPosition).biasedToEnd` | done |
| `InnerPosition::comparable` | method | 107 |  | `(innerPosition).comparable` | done |
| `InnerPosition::line_start` | method | 115 |  | `(innerPosition).lineStart` | done |
| `InnerPosition::line_end` | method | 129 |  | `(innerPosition).lineEnd` | done |
| `InnerPosition::downgrade` | method | 143 |  | `(innerPosition).downgrade` | done |
| `impl PartialEq for InnerPosition` | trait-impl | 152 |  | `(innerPosition).eq` | done |
| `Position` | struct | 161 |  | `Position` | done |
| `Position::to_raw` | method | 167 |  | `(Position).ToRaw` | done |
| `Position::inner_node` | method | 171 |  | `(Position).InnerNode` | done |
| `Position::is_format_start` | method | 175 |  | `(Position).IsFormatStart` | done |
| `Position::is_word_start` | method | 188 |  | `(Position).IsWordStart` | done |
| `Position::is_line_start` | method | 199 |  | `(Position).IsLineStart` | done |
| `Position::is_line_end` | method | 203 |  | `(Position).IsLineEnd` | done |
| `Position::is_paragraph_start` | method | 207 |  | `(Position).IsParagraphStart` | done |
| `Position::is_paragraph_end` | method | 213 |  | `(Position).IsParagraphEnd` | done |
| `Position::is_paragraph_separator` | method | 217 |  | `(Position).IsParagraphSeparator` | done |
| `Position::is_page_start` | method | 225 |  | `(Position).IsPageStart` | done |
| `Position::is_document_start` | method | 229 |  | `(Position).IsDocumentStart` | done |
| `Position::is_document_end` | method | 233 |  | `(Position).IsDocumentEnd` | done |
| `Position::to_degenerate_range` | method | 237 |  | `(Position).ToDegenerateRange` | done |
| `Position::to_global_usv_index` | method | 241 |  | `(Position).ToGlobalUsvIndex` | done |
| `Position::to_global_utf16_index` | method | 259 |  | `(Position).ToGlobalUTF16Index` | done |
| `Position::to_line_index` | method | 281 |  | `(Position).ToLineIndex` | done |
| `Position::biased_to_start` | method | 294 |  | `(Position).BiasedToStart` | done |
| `Position::biased_to_end` | method | 301 |  | `(Position).BiasedToEnd` | done |
| `Position::forward_to_character_start` | method | 308 |  | `(Position).ForwardToCharacterStart` | done |
| `Position::forward_to_character_end` | method | 320 |  | `(Position).ForwardToCharacterEnd` | done |
| `Position::backward_to_character_start` | method | 331 |  | `(Position).BackwardToCharacterStart` | done |
| `Position::forward_to_format_start` | method | 343 |  | `(Position).ForwardToFormatStart` | done |
| `Position::forward_to_format_end` | method | 358 |  | `(Position).ForwardToFormatEnd` | done |
| `Position::backward_to_format_start` | method | 362 |  | `(Position).BackwardToFormatStart` | done |
| `Position::forward_to_word_start` | method | 390 |  | `(Position).ForwardToWordStart` | done |
| `Position::forward_to_word_end` | method | 434 |  | `(Position).ForwardToWordEnd` | done |
| `Position::backward_to_word_start` | method | 438 |  | `(Position).BackwardToWordStart` | done |
| `Position::forward_to_line_start` | method | 493 |  | `(Position).ForwardToLineStart` | done |
| `Position::forward_to_line_end` | method | 500 |  | `(Position).ForwardToLineEnd` | done |
| `Position::backward_to_line_start` | method | 508 |  | `(Position).BackwardToLineStart` | done |
| `Position::forward_to_paragraph_start` | method | 516 |  | `(Position).ForwardToParagraphStart` | done |
| `Position::forward_to_paragraph_end` | method | 532 |  | `(Position).ForwardToParagraphEnd` | done |
| `Position::backward_to_paragraph_start` | method | 543 |  | `(Position).BackwardToParagraphStart` | done |
| `Position::forward_to_page_start` | method | 554 |  | `(Position).ForwardToPageStart` | done |
| `Position::forward_to_page_end` | method | 558 |  | `(Position).ForwardToPageEnd` | done |
| `Position::backward_to_page_start` | method | 562 |  | `(Position).BackwardToPageStart` | done |
| `Position::document_end` | method | 566 |  | `(Position).DocumentEnd` | done |
| `Position::document_start` | method | 570 |  | `(Position).DocumentStart` | done |
| `impl PartialEq for Position` | trait-impl | 575 |  | `(Position).Equal` | done |
| `impl PartialOrd for Position` | trait-impl | 583 |  | `(Position).PartialCmp` | done |
| `RangePropertyValue` | enum | 595 |  | `RangePropertyValue` | done |
| `RangePropertyValue::map` | method | 601 |  | `RangePropertyValueMap` | done |
| `Range` | struct | 613 |  | `Range` | done |
| `Range::new` | assoc-fn | 620 |  | `newRange` | done |
| `Range::node` | method | 627 |  | `(Range).Node` | done |
| `Range::start` | method | 631 |  | `(Range).Start` | done |
| `Range::end` | method | 638 |  | `(Range).End` | done |
| `Range::is_degenerate` | method | 645 |  | `(Range).IsDegenerate` | done |
| `Range::walk` | method | 649 |  | `(Range).walk` | done |
| `Range::traverse_text` | method | 680 |  | `(Range).TraverseText` | done |
| `Range::write_text` | method | 719 |  | `(Range).WriteText` | done |
| `Range::text` | method | 727 |  | `(Range).Text` | done |
| `Range::bounding_boxes` | method | 740 |  | `(Range).BoundingBoxes` | done |
| `Range::fetch_property` | method | 823 |  | `fetchProperty` | done |
| `Range::fix_start_bias` | method | 842 |  | `(Range).fixStartBias` | done |
| `Range::set_start` | method | 848 |  | `(Range).SetStart` | done |
| `Range::set_end` | method | 859 |  | `(Range).SetEnd` | done |
| `Range::to_text_selection` | method | 870 |  | `(Range).ToTextSelection` | done |
| `Range::downgrade` | method | 877 |  | `(Range).Downgrade` | done |
| `impl PartialEq for Range` | trait-impl | 888 |  | `(Range).Equal` | done |
| `WeakRange` | struct | 897 |  | `WeakRange` | done |
| `derive(Clone, PartialEq) for WeakRange` | trait-impl | 896 |  | `(WeakRange).Equal` | done |
| `WeakRange::node_id` | method | 906 |  | `(WeakRange).NodeID` | done |
| `WeakRange::start_comparable` | method | 910 |  | `(WeakRange).StartComparable` | done |
| `WeakRange::end_comparable` | method | 914 |  | `(WeakRange).EndComparable` | done |
| `WeakRange::upgrade_node` | method | 918 |  | `(WeakRange).UpgradeNode` | done |
| `WeakRange::upgrade` | method | 924 |  | `(WeakRange).Upgrade` | done |
| `text_node_filter` | fn | 932 |  | `textNodeFilter` | done |
| `character_index_at_point` | fn | 940 |  | `characterIndexAtPoint` | done |
| `Node::text_direction` | method | 1200 |  | `(Node).TextDirection` | done |
| `Node::font_family` | method | 1200 |  | `(Node).FontFamily` | done |
| `Node::language` | method | 1200 |  | `(Node).Language` | done |
| `Node::font_size` | method | 1200 |  | `(Node).FontSize` | done |
| `Node::font_weight` | method | 1200 |  | `(Node).FontWeight` | done |
| `Node::background_color` | method | 1200 |  | `(Node).BackgroundColor` | done |
| `Node::foreground_color` | method | 1200 |  | `(Node).ForegroundColor` | done |
| `Node::overline` | method | 1200 |  | `(Node).Overline` | done |
| `Node::strikethrough` | method | 1200 |  | `(Node).Strikethrough` | done |
| `Node::underline` | method | 1200 |  | `(Node).Underline` | done |
| `Node::text_align` | method | 1200 |  | `(Node).TextAlign` | done |
| `Node::vertical_offset` | method | 1200 |  | `(Node).VerticalOffset` | done |
| `Position::text_direction` | method | 1200 |  | `(Position).TextDirection` | done |
| `Position::font_family` | method | 1200 |  | `(Position).FontFamily` | done |
| `Position::language` | method | 1200 |  | `(Position).Language` | done |
| `Position::font_size` | method | 1200 |  | `(Position).FontSize` | done |
| `Position::font_weight` | method | 1200 |  | `(Position).FontWeight` | done |
| `Position::background_color` | method | 1200 |  | `(Position).BackgroundColor` | done |
| `Position::foreground_color` | method | 1200 |  | `(Position).ForegroundColor` | done |
| `Position::overline` | method | 1200 |  | `(Position).Overline` | done |
| `Position::strikethrough` | method | 1200 |  | `(Position).Strikethrough` | done |
| `Position::underline` | method | 1200 |  | `(Position).Underline` | done |
| `Position::text_align` | method | 1200 |  | `(Position).TextAlign` | done |
| `Position::vertical_offset` | method | 1200 |  | `(Position).VerticalOffset` | done |
| `Range::text_direction` | method | 1200 |  | `(Range).TextDirection` | done |
| `Range::font_family` | method | 1200 |  | `(Range).FontFamily` | done |
| `Range::language` | method | 1200 |  | `(Range).Language` | done |
| `Range::font_size` | method | 1200 |  | `(Range).FontSize` | done |
| `Range::font_weight` | method | 1200 |  | `(Range).FontWeight` | done |
| `Range::background_color` | method | 1200 |  | `(Range).BackgroundColor` | done |
| `Range::foreground_color` | method | 1200 |  | `(Range).ForegroundColor` | done |
| `Range::overline` | method | 1200 |  | `(Range).Overline` | done |
| `Range::strikethrough` | method | 1200 |  | `(Range).Strikethrough` | done |
| `Range::underline` | method | 1200 |  | `(Range).Underline` | done |
| `Range::text_align` | method | 1200 |  | `(Range).TextAlign` | done |
| `Range::vertical_offset` | method | 1200 |  | `(Range).VerticalOffset` | done |
| `Node::is_italic` | method | 1361 |  | `(Node).IsItalic` | done |
| `Position::is_italic` | method | 1361 |  | `(Position).IsItalic` | done |
| `Range::is_italic` | method | 1361 |  | `(Range).IsItalic` | done |
| `Node::text_attributes_differ` | method | 1366 |  | `(Node).textAttributesDiffer` | done |
| `Node::text_runs` | method | 1381 |  | `(Node).textRuns` | done |
| `Node::following_text_runs` | method | 1386 |  | `(Node).followingTextRuns` | done |
| `Node::preceding_text_runs` | method | 1394 |  | `(Node).precedingTextRuns` | done |
| `Node::supports_text_ranges` | method | 1402 |  | `(Node).SupportsTextRanges` | done |
| `Node::document_start_inner` | method | 1408 |  | `(Node).documentStartInner` | done |
| `Node::document_start` | method | 1416 |  | `(Node).DocumentStart` | done |
| `Node::document_end_inner` | method | 1423 |  | `(Node).documentEndInner` | done |
| `Node::document_end` | method | 1431 |  | `(Node).DocumentEnd` | done |
| `Node::document_range` | method | 1438 |  | `(Node).DocumentRange` | done |
| `Node::has_text_selection` | method | 1444 |  | `(Node).HasTextSelection` | done |
| `Node::text_selection` | method | 1448 |  | `(Node).TextSelection` | done |
| `Node::text_selection_anchor` | method | 1459 |  | `(Node).TextSelectionAnchor` | done |
| `Node::text_selection_focus` | method | 1471 |  | `(Node).TextSelectionFocus` | done |
| `Node::text_position_at_point` | method | 1485 |  | `(Node).TextPositionAtPoint` | done |
| `Node::line_range_from_index` | method | 1548 |  | `(Node).LineRangeFromIndex` | done |
| `Node::text_position_from_global_usv_index` | method | 1571 |  | `(Node).TextPositionFromGlobalUsvIndex` | done |
| `Node::text_position_from_global_utf16_index` | method | 1610 |  | `(Node).TextPositionFromGlobalUTF16Index` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests::TEST_TEXT_DECORATION_1` | fixture | 1658 |  | `testTextDecoration1` | done |
| `tests::TEST_TEXT_DECORATION_2` | fixture | 1667 |  | `testTextDecoration2` | done |
| `tests::TEST_TEXT_DECORATION_3` | fixture | 1676 |  | `testTextDecoration3` | done |
| `tests::TEST_TEXT_DECORATION_4` | fixture | 1685 |  | `testTextDecoration4` | done |
| `tests::main_multiline_tree` | fixture | 1697 |  | `mainMultilineTree` | done |
| `tests::multiline_end_selection` | fixture | 1916 |  | `multilineEndSelection` | done |
| `tests::multiline_past_end_selection` | fixture | 1931 |  | `multilinePastEndSelection` | done |
| `tests::multiline_wrapped_line_end_selection` | fixture | 1946 |  | `multilineWrappedLineEndSelection` | done |
| `tests::multiline_first_line_middle_selection` | fixture | 1961 |  | `multilineFirstLineMiddleSelection` | done |
| `tests::multiline_second_line_middle_selection` | fixture | 1976 |  | `multilineSecondLineMiddleSelection` | done |
| `supports_text_ranges` | test | 1991 |  | `TestText_SupportsTextRanges` | done |
| `multiline_document_range` | test | 2009 |  | `TestText_MultilineDocumentRange` | done |
| `multiline_document_range_to_first_format_change` | test | 2083 |  | `TestText_MultilineDocumentRangeToFirstFormatChange` | done |
| `multiline_document_range_from_last_format_change` | test | 2113 |  | `TestText_MultilineDocumentRangeFromLastFormatChange` | done |
| `multiline_end_degenerate_range` | test | 2155 |  | `TestText_MultilineEndDegenerateRange` | done |
| `multiline_wrapped_line_end_range` | test | 2181 |  | `TestText_MultilineWrappedLineEndRange` | done |
| `multiline_find_line_ends_from_middle` | test | 2243 |  | `TestText_MultilineFindLineEndsFromMiddle` | done |
| `multiline_find_wrapped_line_ends_from_middle` | test | 2289 |  | `TestText_MultilineFindWrappedLineEndsFromMiddle` | done |
| `multiline_find_paragraph_ends_from_middle` | test | 2321 |  | `TestText_MultilineFindParagraphEndsFromMiddle` | done |
| `multiline_find_format_ends_from_middle` | test | 2379 |  | `TestText_MultilineFindFormatEndsFromMiddle` | done |
| `multiline_find_word_ends_from_middle` | test | 2407 |  | `TestText_MultilineFindWordEndsFromMiddle` | done |
| `text_position_at_point` | test | 2447 |  | `TestText_TextPositionAtPoint` | done |
| `to_global_usv_index` | test | 2539 |  | `TestText_ToGlobalUsvIndex` | done |
| `to_global_utf16_index` | test | 2564 |  | `TestText_ToGlobalUTF16Index` | done |
| `to_line_index` | test | 2589 |  | `TestText_ToLineIndex` | done |
| `line_range_from_index` | test | 2615 |  | `TestText_LineRangeFromIndex` | done |
| `text_position_from_global_usv_index` | test | 2654 |  | `TestText_TextPositionFromGlobalUsvIndex` | done |
| `text_position_from_global_utf16_index` | test | 2733 |  | `TestText_TextPositionFromGlobalUTF16Index` | done |
| `multiline_selection_clamping` | test | 2812 |  | `TestText_MultilineSelectionClamping` | done |
| `range_property_value_map` | test | 2820 |  | `TestText_RangePropertyValueMap` | done |
| `text_direction::directly_set` | test | 1004 |  | `TestText_TextDirection_DirectlySet` | done |
| `text_direction::set_on_parent` | test | 1033 |  | `TestText_TextDirection_SetOnParent` | done |
| `text_direction::only_child_overrides_parent` | test | 1062 |  | `TestText_TextDirection_OnlyChildOverridesParent` | done |
| `text_direction::unset` | test | 1093 |  | `TestText_TextDirection_Unset` | done |
| `text_direction::mixed_some_and_none` | test | 1121 |  | `TestText_TextDirection_MixedSomeAndNone` | done |
| `text_direction::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_TextDirection_MixedOneChildOverridesParent` | done |
| `font_family::directly_set` | test | 1004 |  | `TestText_FontFamily_DirectlySet` | done |
| `font_family::set_on_parent` | test | 1033 |  | `TestText_FontFamily_SetOnParent` | done |
| `font_family::only_child_overrides_parent` | test | 1062 |  | `TestText_FontFamily_OnlyChildOverridesParent` | done |
| `font_family::unset` | test | 1093 |  | `TestText_FontFamily_Unset` | done |
| `font_family::mixed_some_and_none` | test | 1121 |  | `TestText_FontFamily_MixedSomeAndNone` | done |
| `font_family::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_FontFamily_MixedOneChildOverridesParent` | done |
| `language::directly_set` | test | 1004 |  | `TestText_Language_DirectlySet` | done |
| `language::set_on_parent` | test | 1033 |  | `TestText_Language_SetOnParent` | done |
| `language::only_child_overrides_parent` | test | 1062 |  | `TestText_Language_OnlyChildOverridesParent` | done |
| `language::unset` | test | 1093 |  | `TestText_Language_Unset` | done |
| `language::mixed_some_and_none` | test | 1121 |  | `TestText_Language_MixedSomeAndNone` | done |
| `language::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_Language_MixedOneChildOverridesParent` | done |
| `font_size::directly_set` | test | 1004 |  | `TestText_FontSize_DirectlySet` | done |
| `font_size::set_on_parent` | test | 1033 |  | `TestText_FontSize_SetOnParent` | done |
| `font_size::only_child_overrides_parent` | test | 1062 |  | `TestText_FontSize_OnlyChildOverridesParent` | done |
| `font_size::unset` | test | 1093 |  | `TestText_FontSize_Unset` | done |
| `font_size::mixed_some_and_none` | test | 1121 |  | `TestText_FontSize_MixedSomeAndNone` | done |
| `font_size::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_FontSize_MixedOneChildOverridesParent` | done |
| `font_weight::directly_set` | test | 1004 |  | `TestText_FontWeight_DirectlySet` | done |
| `font_weight::set_on_parent` | test | 1033 |  | `TestText_FontWeight_SetOnParent` | done |
| `font_weight::only_child_overrides_parent` | test | 1062 |  | `TestText_FontWeight_OnlyChildOverridesParent` | done |
| `font_weight::unset` | test | 1093 |  | `TestText_FontWeight_Unset` | done |
| `font_weight::mixed_some_and_none` | test | 1121 |  | `TestText_FontWeight_MixedSomeAndNone` | done |
| `font_weight::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_FontWeight_MixedOneChildOverridesParent` | done |
| `background_color::directly_set` | test | 1004 |  | `TestText_BackgroundColor_DirectlySet` | done |
| `background_color::set_on_parent` | test | 1033 |  | `TestText_BackgroundColor_SetOnParent` | done |
| `background_color::only_child_overrides_parent` | test | 1062 |  | `TestText_BackgroundColor_OnlyChildOverridesParent` | done |
| `background_color::unset` | test | 1093 |  | `TestText_BackgroundColor_Unset` | done |
| `background_color::mixed_some_and_none` | test | 1121 |  | `TestText_BackgroundColor_MixedSomeAndNone` | done |
| `background_color::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_BackgroundColor_MixedOneChildOverridesParent` | done |
| `foreground_color::directly_set` | test | 1004 |  | `TestText_ForegroundColor_DirectlySet` | done |
| `foreground_color::set_on_parent` | test | 1033 |  | `TestText_ForegroundColor_SetOnParent` | done |
| `foreground_color::only_child_overrides_parent` | test | 1062 |  | `TestText_ForegroundColor_OnlyChildOverridesParent` | done |
| `foreground_color::unset` | test | 1093 |  | `TestText_ForegroundColor_Unset` | done |
| `foreground_color::mixed_some_and_none` | test | 1121 |  | `TestText_ForegroundColor_MixedSomeAndNone` | done |
| `foreground_color::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_ForegroundColor_MixedOneChildOverridesParent` | done |
| `overline::directly_set` | test | 1004 |  | `TestText_Overline_DirectlySet` | done |
| `overline::set_on_parent` | test | 1033 |  | `TestText_Overline_SetOnParent` | done |
| `overline::only_child_overrides_parent` | test | 1062 |  | `TestText_Overline_OnlyChildOverridesParent` | done |
| `overline::unset` | test | 1093 |  | `TestText_Overline_Unset` | done |
| `overline::mixed_some_and_none` | test | 1121 |  | `TestText_Overline_MixedSomeAndNone` | done |
| `overline::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_Overline_MixedOneChildOverridesParent` | done |
| `strikethrough::directly_set` | test | 1004 |  | `TestText_Strikethrough_DirectlySet` | done |
| `strikethrough::set_on_parent` | test | 1033 |  | `TestText_Strikethrough_SetOnParent` | done |
| `strikethrough::only_child_overrides_parent` | test | 1062 |  | `TestText_Strikethrough_OnlyChildOverridesParent` | done |
| `strikethrough::unset` | test | 1093 |  | `TestText_Strikethrough_Unset` | done |
| `strikethrough::mixed_some_and_none` | test | 1121 |  | `TestText_Strikethrough_MixedSomeAndNone` | done |
| `strikethrough::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_Strikethrough_MixedOneChildOverridesParent` | done |
| `underline::directly_set` | test | 1004 |  | `TestText_Underline_DirectlySet` | done |
| `underline::set_on_parent` | test | 1033 |  | `TestText_Underline_SetOnParent` | done |
| `underline::only_child_overrides_parent` | test | 1062 |  | `TestText_Underline_OnlyChildOverridesParent` | done |
| `underline::unset` | test | 1093 |  | `TestText_Underline_Unset` | done |
| `underline::mixed_some_and_none` | test | 1121 |  | `TestText_Underline_MixedSomeAndNone` | done |
| `underline::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_Underline_MixedOneChildOverridesParent` | done |
| `text_align::directly_set` | test | 1004 |  | `TestText_TextAlign_DirectlySet` | done |
| `text_align::set_on_parent` | test | 1033 |  | `TestText_TextAlign_SetOnParent` | done |
| `text_align::only_child_overrides_parent` | test | 1062 |  | `TestText_TextAlign_OnlyChildOverridesParent` | done |
| `text_align::unset` | test | 1093 |  | `TestText_TextAlign_Unset` | done |
| `text_align::mixed_some_and_none` | test | 1121 |  | `TestText_TextAlign_MixedSomeAndNone` | done |
| `text_align::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_TextAlign_MixedOneChildOverridesParent` | done |
| `vertical_offset::directly_set` | test | 1004 |  | `TestText_VerticalOffset_DirectlySet` | done |
| `vertical_offset::set_on_parent` | test | 1033 |  | `TestText_VerticalOffset_SetOnParent` | done |
| `vertical_offset::only_child_overrides_parent` | test | 1062 |  | `TestText_VerticalOffset_OnlyChildOverridesParent` | done |
| `vertical_offset::unset` | test | 1093 |  | `TestText_VerticalOffset_Unset` | done |
| `vertical_offset::mixed_some_and_none` | test | 1121 |  | `TestText_VerticalOffset_MixedSomeAndNone` | done |
| `vertical_offset::mixed_one_child_overrides_parent` | test | 1154 |  | `TestText_VerticalOffset_MixedOneChildOverridesParent` | done |
| `is_italic::directly_set` | test | 1239 |  | `TestText_IsItalic_DirectlySet` | done |
| `is_italic::set_on_parent` | test | 1268 |  | `TestText_IsItalic_SetOnParent` | done |
| `is_italic::unset` | test | 1297 |  | `TestText_IsItalic_Unset` | done |
| `is_italic::mixed` | test | 1325 |  | `TestText_IsItalic_Mixed` | done |

## Deviations registered here

| Id | Meaning | Owner |
|---|---|---|
| `D-CDEBUG` | Rust's derived `Debug` for `Tree` and `State` has no Go counterpart; no caller formats them. | M3-a11y-consumer-tree |
| `D-CSIZEHINT` | Rust iterator trait plumbing (`size_hint` on non-exact iterators, `FusedIterator` markers) has no Go counterpart; Go iterator structs stay exhausted once they return false. | M3-a11y-consumer-node |
| `D-SEC-A11YPARA` | `Position.BackwardToParagraphStart` stops when a backward line step makes no progress (an empty text run linked by `next_on_line`), instead of looping forever while holding the tree read lock (SEC-040). | security sweep round 1 |
