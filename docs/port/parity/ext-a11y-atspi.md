# Parity: ext-a11y-atspi

Scope: accesskit_atspi_common 0.19.1 (`src/*.rs`) and the atspi-common 0.13.0 types it re-exports; accesskit_unix 0.22.1 sections are added by the stage that ports it. Milestone: M3. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_atspi_common-0.19.1/`.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`, `blocked:<stage>`. M3-a11y-atspi-node ports node.rs, util.rs, rect.rs, text_attributes.rs, action.rs, error.rs, callback.rs and filters.rs into package `a11y/atspi` (one Go file per upstream file, tests in `node_test.go`); M3-a11y-atspi-adapter ports adapter.rs, simplified.rs, context.rs, events.rs and lib.rs and deletes `stubs.go`.

Notes on the port:

- `Result<T>` returns are `(T, error)` with an `Error` value; `Option<NodeId>` returns are `(consumer.NodeID, bool, error)`.
- `FromIterator` collectors (`map_children`, `map_descendants`, ...) are generic methods returning slices; `HashMap<&'static str, String>` is `map[string]string`.
- `Weak<Context>` / `Weak<RwLock<AppContext>>` are `weak.Pointer`; see `D-ATSPIWEAK`.
- M3-a11y-atspi-adapter replaced `stubs.go` with `context.go`, `events.go`, `adapter.go` and `lib.go`; `impl Drop for Adapter` is `(*Adapter).Release` (DESIGN 5.2), which also sets the context's `released` flag (`D-ATSPIWEAK`). Data enums (`Event`, `CacheEvent`, `ObjectEvent`, `Property`, `EventData`) are sealed interfaces. `Debug` impls are `String` methods that print shared contexts by pointer / adapter IDs instead of recursing.
- None of the files ported here has upstream unit tests; `node_test.go` adds coverage of its own.

## accesskit_atspi_common-0.19.1/src/node.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `NodeWrapper` | struct | 35 |  | `nodeWrapper` | done |
| `NodeWrapper::name` | method | 38 |  | `(nodeWrapper).name` | done |
| `NodeWrapper::description` | method | 46 |  | `(nodeWrapper).description` | done |
| `NodeWrapper::parent_id` | method | 50 |  | `(nodeWrapper).parentID` | done |
| `NodeWrapper::id` | method | 54 |  | `(nodeWrapper).id` | done |
| `NodeWrapper::cache_node` | method | 58 |  | `(nodeWrapper).cacheNode` | done |
| `NodeWrapper::filtered_child_ids` | method | 83 |  | `(nodeWrapper).filteredChildIDs` | done |
| `NodeWrapper::role` | method | 87 |  | `(nodeWrapper).role` | done |
| `NodeWrapper::is_focused` | method | 297 |  | `(nodeWrapper).isFocused` | done |
| `NodeWrapper::state` | method | 301 |  | `(nodeWrapper).state` | done |
| `NodeWrapper::placeholder` | method | 386 |  | `(nodeWrapper).placeholder` | done |
| `NodeWrapper::position_in_set` | method | 390 |  | `(nodeWrapper).positionInSet` | done |
| `NodeWrapper::size_of_set` | method | 394 |  | `(nodeWrapper).sizeOfSet` | done |
| `NodeWrapper::braille_label` | method | 400 |  | `(nodeWrapper).brailleLabel` | done |
| `NodeWrapper::braille_role_description` | method | 404 |  | `(nodeWrapper).brailleRoleDescription` | done |
| `NodeWrapper::attributes` | method | 408 |  | `(nodeWrapper).attributes` | done |
| `NodeWrapper::is_root` | method | 429 |  | `(nodeWrapper).isRoot` | done |
| `NodeWrapper::supports_action` | method | 433 |  | `(nodeWrapper).supportsAction` | done |
| `NodeWrapper::supports_component` | method | 437 |  | `(nodeWrapper).supportsComponent` | done |
| `NodeWrapper::supports_hyperlink` | method | 441 |  | `(nodeWrapper).supportsHyperlink` | done |
| `NodeWrapper::supports_selection` | method | 445 |  | `(nodeWrapper).supportsSelection` | done |
| `NodeWrapper::supports_text` | method | 449 |  | `(nodeWrapper).supportsText` | done |
| `NodeWrapper::supports_value` | method | 453 |  | `(nodeWrapper).supportsValue` | done |
| `NodeWrapper::interfaces` | method | 457 |  | `(nodeWrapper).interfaces` | done |
| `NodeWrapper::live` | method | 480 |  | `(nodeWrapper).live` | done |
| `NodeWrapper::n_actions` | method | 489 |  | `(nodeWrapper).nActions` | done |
| `NodeWrapper::get_action_name` | method | 493 |  | `(nodeWrapper).getActionName` | done |
| `NodeWrapper::raw_bounds_and_transform` | method | 504 |  | `(nodeWrapper).rawBoundsAndTransform` | done |
| `NodeWrapper::extents` | method | 509 |  | `(nodeWrapper).extents` | done |
| `NodeWrapper::current_value` | method | 534 |  | `(nodeWrapper).currentValue` | done |
| `NodeWrapper::notify_changes` | method | 538 |  | `(nodeWrapper).notifyChanges` | done |
| `NodeWrapper::notify_state_changes` | method | 550 |  | `(nodeWrapper).notifyStateChanges` | done |
| `NodeWrapper::notify_property_changes` | method | 566 |  | `(nodeWrapper).notifyPropertyChanges` | done |
| `NodeWrapper::notify_bounds_changes` | method | 617 |  | `(nodeWrapper).notifyBoundsChanges` | done |
| `NodeWrapper::notify_children_changes` | method | 630 |  | `(nodeWrapper).notifyChildrenChanges` | done |
| `PlatformNode` | struct | 647 |  | `PlatformNode` | done |
| `PlatformNode::new` | method | 654 |  | `newPlatformNode` | done |
| `PlatformNode::from_adapter_root` | method | 662 |  | `platformNodeFromAdapterRoot` | done |
| `PlatformNode::upgrade_context` | method | 667 |  | `(PlatformNode).upgradeContext` | done |
| `PlatformNode::with_tree_state` | method | 675 |  | `withTreeState` | done |
| `PlatformNode::with_tree_and_context` | method | 684 |  | `withTreeAndContext` | done |
| `PlatformNode::resolve_with_context` | method | 693 |  | `resolveWithContext` | done |
| `PlatformNode::resolve_for_selection_with_context` | method | 706 |  | `resolveForSelectionWithContext` | done |
| `PlatformNode::resolve_for_text_with_context` | method | 720 |  | `resolveForTextWithContext` | done |
| `PlatformNode::resolve` | method | 734 |  | `resolve` | done |
| `PlatformNode::resolve_for_selection` | method | 741 |  | `resolveForSelection` | done |
| `PlatformNode::resolve_for_text` | method | 755 |  | `resolveForText` | done |
| `PlatformNode::do_action_internal` | method | 762 |  | `(PlatformNode).doActionInternal` | done |
| `PlatformNode::name` | method | 778 |  | `(PlatformNode).Name` | done |
| `PlatformNode::description` | method | 785 |  | `(PlatformNode).Description` | done |
| `PlatformNode::relative` | method | 792 |  | `(PlatformNode).Relative` | done |
| `PlatformNode::root` | method | 800 |  | `(PlatformNode).Root` | done |
| `PlatformNode::toolkit_name` | method | 805 |  | `(PlatformNode).ToolkitName` | done |
| `PlatformNode::toolkit_version` | method | 809 |  | `(PlatformNode).ToolkitVersion` | done |
| `PlatformNode::parent` | method | 813 |  | `(PlatformNode).Parent` | done |
| `PlatformNode::child_count` | method | 822 |  | `(PlatformNode).ChildCount` | done |
| `PlatformNode::adapter_id` | method | 829 |  | `(PlatformNode).AdapterID` | done |
| `PlatformNode::id` | method | 833 |  | `(PlatformNode).ID` | done |
| `PlatformNode::accessible_id` | method | 837 |  | `(PlatformNode).AccessibleID` | done |
| `PlatformNode::child_at_index` | method | 847 |  | `(PlatformNode).ChildAtIndex` | done |
| `PlatformNode::map_children` | method | 857 |  | `(PlatformNode).MapChildren` | done |
| `PlatformNode::index_in_parent` | method | 871 |  | `(PlatformNode).IndexInParent` | done |
| `PlatformNode::cache_node` | method | 886 |  | `(PlatformNode).CacheNode` | done |
| `PlatformNode::relation_set` | method | 910 |  | `(PlatformNode).RelationSet` | done |
| `PlatformNode::role` | method | 929 |  | `(PlatformNode).Role` | done |
| `PlatformNode::localized_role_name` | method | 936 |  | `(PlatformNode).LocalizedRoleName` | done |
| `PlatformNode::state` | method | 940 |  | `(PlatformNode).State` | done |
| `PlatformNode::attributes` | method | 948 |  | `(PlatformNode).Attributes` | done |
| `PlatformNode::supports_action` | method | 955 |  | `(PlatformNode).SupportsAction` | done |
| `PlatformNode::supports_component` | method | 962 |  | `(PlatformNode).SupportsComponent` | done |
| `PlatformNode::supports_hyperlink` | method | 969 |  | `(PlatformNode).SupportsHyperlink` | done |
| `PlatformNode::supports_selection` | method | 976 |  | `(PlatformNode).SupportsSelection` | done |
| `PlatformNode::supports_text` | method | 983 |  | `(PlatformNode).SupportsText` | done |
| `PlatformNode::supports_value` | method | 990 |  | `(PlatformNode).SupportsValue` | done |
| `PlatformNode::interfaces` | method | 997 |  | `(PlatformNode).Interfaces` | done |
| `PlatformNode::n_actions` | method | 1004 |  | `(PlatformNode).NActions` | done |
| `PlatformNode::action_name` | method | 1011 |  | `(PlatformNode).ActionName` | done |
| `PlatformNode::actions` | method | 1018 |  | `(PlatformNode).Actions` | done |
| `PlatformNode::do_action` | method | 1034 |  | `(PlatformNode).DoAction` | done |
| `PlatformNode::contains` | method | 1047 |  | `(PlatformNode).Contains` | done |
| `PlatformNode::accessible_at_point` | method | 1059 |  | `(PlatformNode).AccessibleAtPoint` | done |
| `PlatformNode::extents` | method | 1077 |  | `(PlatformNode).Extents` | done |
| `PlatformNode::layer` | method | 1087 |  | `(PlatformNode).Layer` | done |
| `PlatformNode::grab_focus` | method | 1098 |  | `(PlatformNode).GrabFocus` | done |
| `PlatformNode::scroll_to` | method | 1108 |  | `(PlatformNode).ScrollTo` | done |
| `PlatformNode::scroll_to_point` | method | 1118 |  | `(PlatformNode).ScrollToPoint` | done |
| `PlatformNode::n_anchors` | method | 1139 |  | `(PlatformNode).NAnchors` | done |
| `PlatformNode::hyperlink_start_index` | method | 1143 |  | `(PlatformNode).HyperlinkStartIndex` | done |
| `PlatformNode::hyperlink_end_index` | method | 1150 |  | `(PlatformNode).HyperlinkEndIndex` | done |
| `PlatformNode::hyperlink_object` | method | 1157 |  | `(PlatformNode).HyperlinkObject` | done |
| `PlatformNode::uri` | method | 1167 |  | `(PlatformNode).URI` | done |
| `PlatformNode::hyperlink_is_valid` | method | 1177 |  | `(PlatformNode).HyperlinkIsValid` | done |
| `PlatformNode::n_selected_children` | method | 1181 |  | `(PlatformNode).NSelectedChildren` | done |
| `PlatformNode::selected_child` | method | 1191 |  | `(PlatformNode).SelectedChild` | done |
| `PlatformNode::select_child` | method | 1201 |  | `(PlatformNode).SelectChild` | done |
| `PlatformNode::deselect_selected_child` | method | 1225 |  | `(PlatformNode).DeselectSelectedChild` | done |
| `PlatformNode::is_child_selected` | method | 1251 |  | `(PlatformNode).IsChildSelected` | done |
| `PlatformNode::select_all` | method | 1260 |  | `(PlatformNode).SelectAll` | done |
| `PlatformNode::clear_selection` | method | 1265 |  | `(PlatformNode).ClearSelection` | done |
| `PlatformNode::deselect_child` | method | 1270 |  | `(PlatformNode).DeselectChild` | done |
| `PlatformNode::character_count` | method | 1294 |  | `(PlatformNode).CharacterCount` | done |
| `PlatformNode::caret_offset` | method | 1304 |  | `(PlatformNode).CaretOffset` | done |
| `PlatformNode::string_at_offset` | method | 1315 |  | `(PlatformNode).StringAtOffset` | done |
| `PlatformNode::text` | method | 1338 |  | `(PlatformNode).Text` | done |
| `PlatformNode::set_caret_offset` | method | 1346 |  | `(PlatformNode).SetCaretOffset` | done |
| `PlatformNode::text_attribute_value` | method | 1363 |  | `(PlatformNode).TextAttributeValue` | done |
| `PlatformNode::text_attributes` | method | 1373 |  | `(PlatformNode).TextAttributes` | done |
| `PlatformNode::default_text_attributes` | method | 1380 |  | `(PlatformNode).DefaultTextAttributes` | done |
| `PlatformNode::character_extents` | method | 1392 |  | `(PlatformNode).CharacterExtents` | done |
| `PlatformNode::offset_at_point` | method | 1409 |  | `(PlatformNode).OffsetAtPoint` | done |
| `PlatformNode::n_selections` | method | 1425 |  | `(PlatformNode).NSelections` | done |
| `PlatformNode::selection` | method | 1434 |  | `(PlatformNode).Selection` | done |
| `PlatformNode::add_selection` | method | 1459 |  | `(PlatformNode).AddSelection` | done |
| `PlatformNode::remove_selection` | method | 1464 |  | `(PlatformNode).RemoveSelection` | done |
| `PlatformNode::set_selection` | method | 1489 |  | `(PlatformNode).SetSelection` | done |
| `PlatformNode::range_extents` | method | 1514 |  | `(PlatformNode).RangeExtents` | done |
| `PlatformNode::text_attribute_run` | method | 1535 |  | `(PlatformNode).TextAttributeRun` | done |
| `PlatformNode::scroll_substring_to` | method | 1574 |  | `(PlatformNode).ScrollSubstringTo` | done |
| `PlatformNode::scroll_substring_to_point` | method | 1607 |  | `(PlatformNode).ScrollSubstringToPoint` | done |
| `PlatformNode::minimum_value` | method | 1639 |  | `(PlatformNode).MinimumValue` | done |
| `PlatformNode::maximum_value` | method | 1643 |  | `(PlatformNode).MaximumValue` | done |
| `PlatformNode::minimum_increment` | method | 1647 |  | `(PlatformNode).MinimumIncrement` | done |
| `PlatformNode::current_value` | method | 1651 |  | `(PlatformNode).CurrentValue` | done |
| `PlatformNode::set_current_value` | method | 1658 |  | `(PlatformNode).SetCurrentValue` | done |
| `impl PartialEq for PlatformNode: eq` | method | 1669 |  | `(PlatformNode).Equal` | done |
| `impl Eq for PlatformNode` | trait-impl | 1674 |  | `(PlatformNode).Equal` | done |
| `impl Hash for PlatformNode: hash` | method | 1677 |  | `(PlatformNode).Key` | done |
| `PlatformRoot` | struct | 1684 |  | `PlatformRoot` | done |
| `PlatformRoot::new` | method | 1689 |  | `NewPlatformRoot` | done |
| `PlatformRoot::resolve_app_context` | method | 1695 |  | `resolveAppContext` | done |
| `PlatformRoot::name` | method | 1707 |  | `(PlatformRoot).Name` | done |
| `PlatformRoot::description` | method | 1711 |  | `(PlatformRoot).Description` | done |
| `PlatformRoot::child_count` | method | 1715 |  | `(PlatformRoot).ChildCount` | done |
| `PlatformRoot::interfaces` | method | 1721 |  | `(PlatformRoot).Interfaces` | done |
| `PlatformRoot::role` | method | 1725 |  | `(PlatformRoot).Role` | done |
| `PlatformRoot::index_in_parent` | method | 1729 |  | `(PlatformRoot).IndexInParent` | done |
| `PlatformRoot::state` | method | 1733 |  | `(PlatformRoot).State` | done |
| `PlatformRoot::child_at_index` | method | 1737 |  | `(PlatformRoot).ChildAtIndex` | done |
| `PlatformRoot::child_id_at_index` | method | 1747 |  | `(PlatformRoot).ChildIDAtIndex` | done |
| `PlatformRoot::map_children` | method | 1757 |  | `(PlatformRoot).MapChildren` | done |
| `PlatformRoot::map_child_ids` | method | 1772 |  | `(PlatformRoot).MapChildIDs` | done |
| `PlatformRoot::toolkit_name` | method | 1787 |  | `(PlatformRoot).ToolkitName` | done |
| `PlatformRoot::toolkit_version` | method | 1791 |  | `(PlatformRoot).ToolkitVersion` | done |
| `PlatformRoot::id` | method | 1795 |  | `(PlatformRoot).ID` | done |
| `PlatformRoot::set_id` | method | 1799 |  | `(PlatformRoot).SetID` | done |
| `PlatformRoot::map_descendants` | method | 1809 |  | `(PlatformRoot).MapDescendants` | done |
| `PlatformRoot::collect_descendants` | method | 1813 |  | `collectDescendants` | done |
| `PlatformRoot::map_descendant_cache_nodes` | method | 1852 |  | `(PlatformRoot).MapDescendantCacheNodes` | done |
| `PlatformRoot::collect_descendants` | method | 1856 |  | `collectDescendantCacheNodes` | done |
| `impl PartialEq for PlatformRoot: eq` | method | 1916 |  | `(PlatformRoot).Equal` | done |
| `impl Hash for PlatformRoot: hash` | method | 1922 |  | `PlatformRoot` | done |
| `NodeIdOrRoot` | enum | 1928 |  | `NodeIDOrRoot` | done |
| `CacheNode` | struct | 1933 |  | `CacheNode` | done |

## accesskit_atspi_common-0.19.1/src/util.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WindowBounds` | struct | 13 |  | `WindowBounds` | done |
| `WindowBounds::new` | method | 19 |  | `NewWindowBounds` | done |
| `WindowBounds::accesskit_point_to_atspi_point` | method | 23 |  | `(*WindowBounds).accesskitPointToAtspiPoint` | done |
| `WindowBounds::atspi_point_to_accesskit_point` | method | 33 |  | `(*WindowBounds).atspiPointToAccesskitPoint` | done |
| `WindowBounds::origin` | method | 43 |  | `(*WindowBounds).origin` | done |
| `text_position_from_offset` | fn | 59 |  | `textPositionFromOffset` | done |
| `text_range_from_offset` | fn | 67 |  | `textRangeFromOffset` | done |
| `text_range_from_offsets` | fn | 96 |  | `textRangeFromOffsets` | done |
| `text_range_bounds_from_offsets` | fn | 113 |  | `textRangeBoundsFromOffsets` | done |
| `atspi_scroll_type_to_scroll_hint` | fn | 124 |  | `atspiScrollTypeToScrollHint` | done |

## accesskit_atspi_common-0.19.1/src/rect.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Rect` | struct | 12 |  | `Rect` | done |
| `impl From<accesskit::Rect> for Rect: from` | method | 29 |  | `RectFromAccessKit` | done |
| `Rect::INVALID` | const | 20 |  | `RectInvalid` | done |

## accesskit_atspi_common-0.19.1/src/text_attributes.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `color_to_string` | fn | 10 |  | `colorToString` | done |
| `family_name` | fn | 14 |  | `familyName` | done |
| `size` | fn | 18 |  | `size` | done |
| `weight` | fn | 22 |  | `weight` | done |
| `style` | fn | 26 |  | `style` | done |
| `strikethrough` | fn | 30 |  | `strikethrough` | done |
| `underline` | fn | 34 |  | `underline` | done |
| `bg_color` | fn | 44 |  | `bgColor` | done |
| `fg_color` | fn | 48 |  | `fgColor` | done |
| `language` | fn | 52 |  | `language` | done |
| `justification` | fn | 56 |  | `justification` | done |
| `ATTRIBUTE_GETTERS` | const | 68 |  | `attributeGetters` | done |

## accesskit_atspi_common-0.19.1/src/action.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Action` | struct | 10 |  | `Action` | done |

## accesskit_atspi_common-0.19.1/src/error.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Error` | enum | 9 |  | `Error` | done |
| `impl fmt::Display for Error: fmt` | method | 19 |  | `(Error).Error` | done |
| `impl std::error::Error for Error` | trait-impl | 31 |  | `(Error).Error` | done |
| `Result` | type | 33 |  | `Error` | dev:D-ATSPIRESULT |

## accesskit_atspi_common-0.19.1/src/callback.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AdapterCallback` | trait | 11 |  | `AdapterCallback` | done |
| `AdapterCallback::register_interfaces` | method | 12 |  | `AdapterCallback` | done |
| `AdapterCallback::unregister_interfaces` | method | 13 |  | `AdapterCallback` | done |
| `AdapterCallback::emit_event` | method | 14 |  | `AdapterCallback` | done |

## accesskit_atspi_common-0.19.1/src/filters.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub(crate) use accesskit_consumer::common_filter as filter` | use | 6 |  | `filter` | done |

## atspi-common-0.13.0 (types re-exported by lib.rs)

Go target: `a11y/atspi/`, in `atspi_common.go` (see `D-ATSPICOMMON`).

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Role` (role.rs) | enum | 14 |  | `Role` | done |
| `Role::try_from(u32)` (role.rs) | trait-impl | 333 |  | `RoleFromU32` | done |
| `Role::name` (role.rs) | method | 609 |  | `(Role).Name` | done |
| `State` (state.rs) | enum | 17 |  | `State` | done |
| `State::to_static_str` (state.rs) | method | 231 |  | `(State).ToStaticStr` | done |
| `impl From<&str> for State` (state.rs) | trait-impl | 300 |  | `StateFromStr` | done |
| `StateSet` (state.rs) | struct | 354 |  | `StateSet` | done |
| `Interface` (interface.rs) | enum | 19 |  | `Interface` | done |
| `InterfaceSet` (interface.rs) | struct | 84 |  | `InterfaceSet` | done |
| `impl BitAnd for InterfaceSet` (interface.rs) | trait-impl | 205 |  | `(InterfaceSet).Intersection` | done |
| `impl BitXor for InterfaceSet` (interface.rs) | trait-impl | 213 |  | `(InterfaceSet).SymmetricDifference` | done |
| `CoordType` (lib.rs) | enum | 74 |  | `CoordType` | done |
| `Granularity` (lib.rs) | enum | 100 |  | `Granularity` | done |
| `Layer` (lib.rs) | enum | 128 |  | `Layer` | done |
| `ScrollType` (lib.rs) | enum | 154 |  | `ScrollType` | done |
| `Politeness` (lib.rs) | enum | 180 |  | `Politeness` | done |
| `RelationType` (relation_type.rs) | enum | 6 |  | `RelationType` | done |
## accesskit_atspi_common-0.19.1/src/adapter.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AdapterChangeHandler` | struct | 30 |  | `adapterChangeHandler` | done |
| `AdapterChangeHandler::new` | method | 39 |  | `newAdapterChangeHandler` | done |
| `AdapterChangeHandler::add_node` | method | 49 |  | `(adapterChangeHandler).addNode` | done |
| `AdapterChangeHandler::add_subtree` | method | 84 |  | `(adapterChangeHandler).addSubtree` | done |
| `AdapterChangeHandler::remove_node` | method | 91 |  | `(adapterChangeHandler).removeNode` | done |
| `AdapterChangeHandler::remove_subtree` | method | 114 |  | `(adapterChangeHandler).removeSubtree` | done |
| `AdapterChangeHandler::emit_text_change_if_needed_parent` | method | 121 |  | `(adapterChangeHandler).emitTextChangeIfNeededParent` | done |
| `AdapterChangeHandler::emit_text_change_if_needed` | method | 184 |  | `(adapterChangeHandler).emitTextChangeIfNeeded` | done |
| `AdapterChangeHandler::emit_text_selection_change` | method | 197 |  | `(adapterChangeHandler).emitTextSelectionChange` | done |
| `AdapterChangeHandler::enqueue_selection_changed_if_needed_parent` | method | 249 |  | `(adapterChangeHandler).enqueueSelectionChangedIfNeededParent` | done |
| `AdapterChangeHandler::enqueue_selection_changed_if_needed` | method | 260 |  | `(adapterChangeHandler).enqueueSelectionChangedIfNeeded` | done |
| `AdapterChangeHandler::emit_selection_changed` | method | 269 |  | `(adapterChangeHandler).emitSelectionChanged` | done |
| `impl TreeChangeHandler for AdapterChangeHandler: node_added` | method | 281 |  | `(adapterChangeHandler).NodeAdded` | done |
| `impl TreeChangeHandler for AdapterChangeHandler: node_updated` | method | 287 |  | `(adapterChangeHandler).NodeUpdated` | done |
| `impl TreeChangeHandler for AdapterChangeHandler: focus_moved` | method | 324 |  | `(adapterChangeHandler).FocusMoved` | done |
| `impl TreeChangeHandler for AdapterChangeHandler: node_removed` | method | 345 |  | `(adapterChangeHandler).NodeRemoved` | done |
| `next_adapter_id` | fn | 356 |  | `NextAdapterID` | done |
| `Adapter` | struct | 360 |  | `Adapter` | done |
| `impl Debug for Adapter: fmt` | method | 367 |  | `(Adapter).String` | done |
| `Adapter::new` | method | 377 |  | `NewAdapter` | done |
| `Adapter::with_id` | method | 397 |  | `NewAdapterWithID` | done |
| `Adapter::with_wrapped_action_handler` | method | 419 |  | `NewAdapterWithWrappedActionHandler` | done |
| `Adapter::register_tree` | method | 444 |  | `(Adapter).registerTree` | done |
| `Adapter::add_children` | method | 445 |  | `addChildren` | done |
| `Adapter::platform_node` | method | 480 |  | `(Adapter).PlatformNode` | done |
| `Adapter::root_id` | method | 484 |  | `(Adapter).RootID` | done |
| `Adapter::platform_root` | method | 488 |  | `(Adapter).PlatformRoot` | done |
| `Adapter::register_interfaces` | method | 492 |  | `(Adapter).registerInterfaces` | done |
| `Adapter::unregister_interfaces` | method | 496 |  | `(Adapter).unregisterInterfaces` | done |
| `Adapter::emit_object_event` | method | 501 |  | `(Adapter).emitObjectEvent` | done |
| `Adapter::emit_root_object_event` | method | 507 |  | `(Adapter).emitRootObjectEvent` | done |
| `Adapter::emit_cache_added` | method | 513 |  | `(Adapter).emitCacheAdded` | done |
| `Adapter::emit_cache_removed` | method | 518 |  | `(Adapter).emitCacheRemoved` | done |
| `Adapter::set_root_window_bounds` | method | 523 |  | `(Adapter).SetRootWindowBounds` | done |
| `Adapter::update` | method | 528 |  | `(Adapter).Update` | done |
| `Adapter::update_window_focus_state` | method | 536 |  | `(Adapter).UpdateWindowFocusState` | done |
| `Adapter::window_created` | method | 542 |  | `(Adapter).windowCreated` | done |
| `Adapter::window_activated` | method | 546 |  | `(Adapter).windowActivated` | done |
| `Adapter::window_deactivated` | method | 559 |  | `(Adapter).windowDeactivated` | done |
| `Adapter::window_destroyed` | method | 571 |  | `(Adapter).windowDestroyed` | done |
| `Adapter::id` | method | 575 |  | `(Adapter).ID` | done |
| `Adapter::is_window_focused` | method | 579 |  | `(Adapter).IsWindowFocused` | done |
| `Adapter::root_window_bounds` | method | 583 |  | `(Adapter).RootWindowBounds` | done |
| `Adapter::wrapped_action_handler` | method | 589 |  | `(Adapter).WrappedActionHandler` | done |
| `root_window` | fn | 594 |  | `rootWindow` | done |
| `impl Drop for Adapter: drop` | method | 605 |  | `(Adapter).Release` | done |
| `register_interfaces` | fn | 636 |  | `registerInterfaces` | dev:D-ATSPITESTHELPERS |
| `unregister_interfaces` | fn | 637 |  | `unregisterInterfaces` | dev:D-ATSPITESTHELPERS |
| `emit_event` | fn | 638 |  | `emitEvent` | dev:D-ATSPITESTHELPERS |
| `do_action` | fn | 650 |  | `doAction` | dev:D-ATSPITESTHELPERS |
| `with_children` | fn | 653 |  | `withChildren` | dev:D-ATSPITESTHELPERS |
| `build` | fn | 659 |  | `build` | dev:D-ATSPITESTHELPERS |
| `initial_tree` | fn | 673 |  | `initialTree` | dev:D-ATSPITESTHELPERS |
| `update` | fn | 688 |  | `update` | dev:D-ATSPITESTHELPERS |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests::no_cache_events_on_construction` | test | 698 | test | `TestAdapter_NoCacheEventsOnConstruction` | done |
| `tests::add_node_emits_one_added` | test | 704 | test | `TestAdapter_AddNodeEmitsOneAdded` | done |
| `tests::remove_node_emits_removed_for_same_id` | test | 720 | test | `TestAdapter_RemoveNodeEmitsRemovedForSameID` | done |
| `tests::subtree_add_emits_added_per_node` | test | 742 | test | `TestAdapter_SubtreeAddEmitsAddedPerNode` | done |
| `tests::subtree_remove_emits_removed_per_node` | test | 763 | test | `TestAdapter_SubtreeRemoveEmitsRemovedPerNode` | done |
| `tests::filter_transition_into_tree_emits_added` | test | 788 | test | `TestAdapter_FilterTransitionIntoTreeEmitsAdded` | done |
| `tests::filter_transition_out_of_tree_emits_removed` | test | 812 | test | `TestAdapter_FilterTransitionOutOfTreeEmitsRemoved` | done |

## accesskit_atspi_common-0.19.1/src/context.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ActionHandlerNoMut` | trait | 15 |  | `ActionHandlerNoMut` | done |
| `ActionHandlerNoMut::do_action` | method | 16 |  | `(ActionHandlerNoMut).DoAction` | done |
| `ActionHandlerWrapper` | struct | 21 |  | `ActionHandlerWrapper` | done |
| `ActionHandlerWrapper::new` | method | 24 |  | `NewActionHandlerWrapper` | done |
| `impl ActionHandlerNoMut for ActionHandlerWrapper: do_action` | method | 30 |  | `(ActionHandlerWrapper).DoAction` | done |
| `Context` | struct | 35 |  | `Context` | done |
| `impl Debug for Context: fmt` | method | 43 |  | `(Context).String` | done |
| `Context::new` | method | 54 |  | `newContext` | done |
| `Context::read_tree` | method | 68 |  | `(Context).readTree` | done |
| `Context::read_root_window_bounds` | method | 72 |  | `(Context).readRootWindowBounds` | done |
| `Context::do_action` | method | 76 |  | `(Context).DoAction` | done |
| `Context::read_app_context` | method | 80 |  | `(Context).readAppContext` | done |
| `Context::write_app_context` | method | 84 |  | `(Context).writeAppContext` | done |
| `AppContext` | struct | 90 |  | `AppContext` | done |
| `AppContext::new` | method | 99 |  | `NewAppContext` | done |
| `AppContext::adapter_index` | method | 109 |  | `(AppContext).adapterIndex` | done |
| `AppContext::push_adapter` | method | 113 |  | `(AppContext).pushAdapter` | done |
| `AppContext::remove_adapter` | method | 117 |  | `(AppContext).removeAdapter` | done |

## accesskit_atspi_common-0.19.1/src/events.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Event` | enum | 12 |  | `Event` | done |
| `CacheEvent` | enum | 26 |  | `CacheEvent` | done |
| `Property` | enum | 32 |  | `Property` | done |
| `ObjectEvent` | enum | 42 |  | `ObjectEvent` | done |
| `WindowEvent` | enum | 66 |  | `WindowEvent` | done |

## accesskit_atspi_common-0.19.1/src/simplified.rs

Go target: `a11y/atspi/simplified/` (see `D-ATSPISIMPLIFIED`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Accessible` | enum | 22 |  | `Accessible` | done |
| `Accessible::role` | method | 28 |  | `(Accessible).Role` | done |
| `Accessible::localized_role_name` | method | 35 |  | `(Accessible).LocalizedRoleName` | done |
| `Accessible::name` | method | 42 |  | `(Accessible).Name` | done |
| `Accessible::description` | method | 49 |  | `(Accessible).Description` | done |
| `Accessible::state` | method | 56 |  | `(Accessible).State` | done |
| `Accessible::attributes` | method | 63 |  | `(Accessible).Attributes` | done |
| `Accessible::parent` | method | 70 |  | `(Accessible).Parent` | done |
| `Accessible::index_in_parent` | method | 80 |  | `(Accessible).IndexInParent` | done |
| `Accessible::child_count` | method | 87 |  | `(Accessible).ChildCount` | done |
| `Accessible::accessible_id` | method | 94 |  | `(Accessible).AccessibleID` | done |
| `Accessible::child_at_index` | method | 101 |  | `(Accessible).ChildAtIndex` | done |
| `Accessible::map_children` | method | 112 |  | `(Accessible).MapChildren` | done |
| `Accessible::relation_set` | method | 122 |  | `(Accessible).RelationSet` | done |
| `Accessible::application` | method | 129 |  | `(Accessible).Application` | done |
| `Accessible::toolkit_name` | method | 136 |  | `(Accessible).ToolkitName` | done |
| `Accessible::toolkit_version` | method | 143 |  | `(Accessible).ToolkitVersion` | done |
| `Accessible::supports_action` | method | 150 |  | `(Accessible).SupportsAction` | done |
| `Accessible::n_actions` | method | 157 |  | `(Accessible).NActions` | done |
| `Accessible::action_name` | method | 164 |  | `(Accessible).ActionName` | done |
| `Accessible::do_action` | method | 171 |  | `(Accessible).DoAction` | done |
| `Accessible::supports_component` | method | 178 |  | `(Accessible).SupportsComponent` | done |
| `Accessible::contains` | method | 185 |  | `(Accessible).Contains` | done |
| `Accessible::accessible_at_point` | method | 192 |  | `(Accessible).AccessibleAtPoint` | done |
| `Accessible::extents` | method | 206 |  | `(Accessible).Extents` | done |
| `Accessible::layer` | method | 213 |  | `(Accessible).Layer` | done |
| `Accessible::grab_focus` | method | 220 |  | `(Accessible).GrabFocus` | done |
| `Accessible::scroll_to` | method | 227 |  | `(Accessible).ScrollTo` | done |
| `Accessible::scroll_to_point` | method | 234 |  | `(Accessible).ScrollToPoint` | done |
| `Accessible::supports_hyperlink` | method | 241 |  | `(Accessible).SupportsHyperlink` | done |
| `Accessible::n_anchors` | method | 248 |  | `(Accessible).NAnchors` | done |
| `Accessible::hyperlink_start_index` | method | 255 |  | `(Accessible).HyperlinkStartIndex` | done |
| `Accessible::hyperlink_end_index` | method | 262 |  | `(Accessible).HyperlinkEndIndex` | done |
| `Accessible::hyperlink_object` | method | 269 |  | `(Accessible).HyperlinkObject` | done |
| `Accessible::uri` | method | 278 |  | `(Accessible).URI` | done |
| `Accessible::hyperlink_is_valid` | method | 285 |  | `(Accessible).HyperlinkIsValid` | done |
| `Accessible::supports_selection` | method | 292 |  | `(Accessible).SupportsSelection` | done |
| `Accessible::n_selected_children` | method | 299 |  | `(Accessible).NSelectedChildren` | done |
| `Accessible::selected_child` | method | 306 |  | `(Accessible).SelectedChild` | done |
| `Accessible::select_child` | method | 315 |  | `(Accessible).SelectChild` | done |
| `Accessible::deselect_selected_child` | method | 322 |  | `(Accessible).DeselectSelectedChild` | done |
| `Accessible::is_child_selected` | method | 329 |  | `(Accessible).IsChildSelected` | done |
| `Accessible::select_all` | method | 336 |  | `(Accessible).SelectAll` | done |
| `Accessible::clear_selection` | method | 343 |  | `(Accessible).ClearSelection` | done |
| `Accessible::deselect_child` | method | 350 |  | `(Accessible).DeselectChild` | done |
| `Accessible::supports_text` | method | 357 |  | `(Accessible).SupportsText` | done |
| `Accessible::character_count` | method | 364 |  | `(Accessible).CharacterCount` | done |
| `Accessible::caret_offset` | method | 371 |  | `(Accessible).CaretOffset` | done |
| `Accessible::string_at_offset` | method | 378 |  | `(Accessible).StringAtOffset` | done |
| `Accessible::text` | method | 389 |  | `(Accessible).Text` | done |
| `Accessible::set_caret_offset` | method | 396 |  | `(Accessible).SetCaretOffset` | done |
| `Accessible::text_attribute_value` | method | 403 |  | `(Accessible).TextAttributeValue` | done |
| `Accessible::text_attributes` | method | 410 |  | `(Accessible).TextAttributes` | done |
| `Accessible::default_text_attributes` | method | 420 |  | `(Accessible).DefaultTextAttributes` | done |
| `Accessible::character_extents` | method | 427 |  | `(Accessible).CharacterExtents` | done |
| `Accessible::offset_at_point` | method | 434 |  | `(Accessible).OffsetAtPoint` | done |
| `Accessible::n_selections` | method | 441 |  | `(Accessible).NSelections` | done |
| `Accessible::selection` | method | 448 |  | `(Accessible).Selection` | done |
| `Accessible::add_selection` | method | 455 |  | `(Accessible).AddSelection` | done |
| `Accessible::remove_selection` | method | 462 |  | `(Accessible).RemoveSelection` | done |
| `Accessible::set_selection` | method | 469 |  | `(Accessible).SetSelection` | done |
| `Accessible::range_extents` | method | 481 |  | `(Accessible).RangeExtents` | done |
| `Accessible::text_attribute_run` | method | 493 |  | `(Accessible).TextAttributeRun` | done |
| `Accessible::scroll_substring_to` | method | 504 |  | `(Accessible).ScrollSubstringTo` | done |
| `Accessible::scroll_substring_to_point` | method | 516 |  | `(Accessible).ScrollSubstringToPoint` | done |
| `Accessible::supports_value` | method | 532 |  | `(Accessible).SupportsValue` | done |
| `Accessible::minimum_value` | method | 539 |  | `(Accessible).MinimumValue` | done |
| `Accessible::maximum_value` | method | 546 |  | `(Accessible).MaximumValue` | done |
| `Accessible::minimum_increment` | method | 553 |  | `(Accessible).MinimumIncrement` | done |
| `Accessible::current_value` | method | 560 |  | `(Accessible).CurrentValue` | done |
| `Accessible::set_current_value` | method | 567 |  | `(Accessible).SetCurrentValue` | done |
| `Cache` | struct | 575 |  | `Cache` | done |
| `Cache::new` | method | 580 |  | `NewCache` | done |
| `Cache::items` | method | 584 |  | `(Cache).Items` | done |
| `EventData` | enum | 596 |  | `EventData` | done |
| `Event` | struct | 605 |  | `Event` | done |
| `Event::new` | method | 614 |  | `NewEvent` | done |
| `do_action` | fn | 796 |  | `doAction` | dev:D-ATSPITESTHELPERS |
| `register_interfaces` | fn | 804 |  | `registerInterfaces` | dev:D-ATSPITESTHELPERS |
| `unregister_interfaces` | fn | 805 |  | `unregisterInterfaces` | dev:D-ATSPITESTHELPERS |
| `emit_event` | fn | 806 |  | `emitEvent` | dev:D-ATSPITESTHELPERS |
| `register_interfaces` | fn | 816 |  | `registerInterfaces` | dev:D-ATSPITESTHELPERS |
| `unregister_interfaces` | fn | 817 |  | `unregisterInterfaces` | dev:D-ATSPITESTHELPERS |
| `emit_event` | fn | 818 |  | `emitEvent` | dev:D-ATSPITESTHELPERS |
| `with_children` | fn | 821 |  | `withChildren` | dev:D-ATSPITESTHELPERS |
| `initial_tree` | fn | 827 |  | `initialTree` | dev:D-ATSPITESTHELPERS |
| `build_cache` | fn | 880 |  | `buildCache` | dev:D-ATSPITESTHELPERS |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests::cache_events_map_to_kind_strings` | test | 843 | test | `TestSimplified_CacheEventsMapToKindStrings` | done |
| `tests::items_prepends_application_root` | test | 895 | test | `TestSimplified_ItemsPrependsApplicationRoot` | done |

## accesskit_atspi_common-0.19.1/src/lib.rs

Go target: `a11y/atspi/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use accesskit_consumer::NodeId` | use | 20 |  | `NodeID` | done |
| `pub use action::*, callback::AdapterCallback, error::*, events::*, node::{CacheNode, NodeIdOrRoot, PlatformNode, PlatformRoot}, rect::*, util::WindowBounds` | use | 25 |  | `PlatformNode` | done |
| `pub use atspi_common::{CoordType, Granularity, InterfaceSet, Layer, RelationType, Role, ScrollType, State, StateSet}` | use | 21 |  | `Role` | done |
| `pub use adapter::{Adapter, next_adapter_id}` | use | 26 |  | `NextAdapterID` | done |
| `pub use context::{ActionHandlerNoMut, ActionHandlerWrapper, AppContext}` | use | 28 |  | `ActionHandlerWrapper` | done |
| `mod simplified` (`simplified-api` feature) | mod | 16 | feature = "simplified-api" | `simplified` | dev:D-ATSPISIMPLIFIED |
## Deviations registered here

| Id | Meaning | Owner |
|---|---|---|
| `D-ATSPIRESULT` | The crate's `Result<T>` alias has no Go counterpart; functions return `(T, error)` with an `Error` value. | M3-a11y-atspi-node |
| `D-ATSPICOMMON` | The atspi-common 0.13.0 types the crate uses (`Role`, `State`, `StateSet`, `Interface`, `InterfaceSet`, `CoordType`, `Granularity`, `Layer`, `ScrollType`, `Politeness`, `RelationType`) are ported into `atspi_common.go` with their discriminants, bit order and names; their serde / zvariant impls belong to the D-Bus layer. | M3-a11y-atspi-node |
| `D-ATSPIWEAK` | `Weak` upgrades use `weak.Pointer`, which only fails after garbage collection, so `Context` also carries a `released` flag the adapter sets where upstream drops its `Arc<Context>`; `PlatformNode` equality is `Equal` / `Key` (adapter ID and node ID) because Go `==` would also compare the weak pointer. | M3-a11y-atspi-node |
| `D-ATSPISIMPLIFIED` | The `simplified-api` feature module is the always-built package `a11y/atspi/simplified`; a separate package keeps its `Event` struct apart from events.rs `Event`. `Accessible` is a struct with `AccessibleNode` / `AccessibleRoot` constructors and `Equal`; `Option<Accessible>` returns are `(Accessible, bool, error)`. | M3-a11y-atspi-adapter |
| `D-ATSPITESTHELPERS` | Helper fns and trait impls inside upstream `mod tests` live in `_test.go` files (`adapter_test.go`, `simplified/simplified_test.go`) rather than in the package, so they have no production symbol. | M3-a11y-atspi-adapter |
