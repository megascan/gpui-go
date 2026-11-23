# Parity: ext-a11y

Scope: accesskit 0.24.1 (`src/lib.rs`, `src/geometry.rs`). Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md (`accesskit (schema only)`).

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`. `lib.rs` rows are `done` with M0-a11y-schema (`a11y/lib.go`, generated `a11y/lib_enums_gen.go`, `a11y/lib_props_gen.go`, `a11y/lib_props_gen_test.go` from `a11y/internal/gen`); `geometry.rs` rows are `done` in the same change as `a11y/geometry.go`. accesskit 0.24.1 `geometry.rs` has no `#[test]` items, so it has no test rows; its behaviour checks live in `a11y/geometry_test.go`. `Insets` (named in STAGES.md) does not exist in accesskit 0.24.1 and is not ported.

## src/lib.rs: public items

Go target: `a11y/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use uuid::Uuid` | use | 32 |  | `Uuid` | done |
| `pub use geometry::{Affine, Point, Rect, Size, Vec2}` | use | 35 |  | `Affine` | done |
| `Role` | enum | 62 |  | `Role` | done |
| `Action` | enum | 289 |  | `Action` | done |
| `Orientation` | enum | 417 |  | `Orientation` | done |
| `TextDirection` | enum | 439 |  | `TextDirection` | done |
| `Invalid` | enum | 465 |  | `Invalid` | done |
| `Toggled` | enum | 486 |  | `Toggled` | done |
| `impl From<bool> for Toggled` | trait-impl | 492 |  | - | dev:D-CONV |
| `SortDirection` | enum | 517 |  | `SortDirection` | done |
| `AriaCurrent` | enum | 538 |  | `AriaCurrent` | done |
| `AutoComplete` | enum | 563 |  | `AutoComplete` | done |
| `Live` | enum | 584 |  | `Live` | done |
| `HasPopup` | enum | 605 |  | `HasPopup` | done |
| `ListStyle` | enum | 628 |  | `ListStyle` | done |
| `TextAlign` | enum | 653 |  | `TextAlign` | done |
| `VerticalOffset` | enum | 675 |  | `VerticalOffset` | done |
| `TextDecorationStyle` | enum | 695 |  | `TextDecorationStyle` | done |
| `NodeIdContent` | type | 703 |  | `NodeIDContent` | done |
| `NodeId` | struct | 715 |  | `NodeID` | done |
| `NodeId::from(NodeIdContent)` | trait-impl | 717 |  | - | dev:D-CONV |
| `NodeIdContent::from(NodeId)` | trait-impl | 724 |  | - | dev:D-CONV |
| `impl fmt::Debug for NodeId` | trait-impl | 731 |  | `(NodeID).String` | done |
| `TreeId` | struct | 745 |  | `TreeID` | done |
| `TreeId::ROOT` | assoc-const | 749 |  | `TreeIDRoot` | done |
| `CustomAction` | struct | 761 |  | `CustomAction` | done |
| `CustomAction.id` | field | 762 |  | `CustomAction.ID` | done |
| `CustomAction.description` | field | 763 |  | `CustomAction.Description` | done |
| `TextPosition` | struct | 771 |  | `TextPosition` | done |
| `TextPosition.node` | field | 773 |  | `TextPosition.Node` | done |
| `TextPosition.character_index` | field | 776 |  | `TextPosition.CharacterIndex` | done |
| `TextSelection` | struct | 784 |  | `TextSelection` | done |
| `TextSelection.anchor` | field | 789 |  | `TextSelection.Anchor` | done |
| `TextSelection.focus` | field | 793 |  | `TextSelection.Focus` | done |
| `Color` | struct | 834 |  | `Color` | done |
| `Color.red` | field | 835 |  | `Color.Red` | done |
| `Color.green` | field | 836 |  | `Color.Green` | done |
| `Color.blue` | field | 837 |  | `Color.Blue` | done |
| `Color.alpha` | field | 838 |  | `Color.Alpha` | done |
| `TextDecoration` | struct | 847 |  | `TextDecoration` | done |
| `TextDecoration.style` | field | 848 |  | `TextDecoration.Style` | done |
| `TextDecoration.color` | field | 849 |  | `TextDecoration.Color` | done |
| `Node` | struct | 1037 |  | `Node` | done |
| `Node::new` | assoc-fn | 1737 |  | `NewNode` | done |
| `Node::role` | method | 1747 |  | `(Node).Role` | done |
| `Node::set_role` | method | 1751 |  | `(Node).SetRole` | done |
| `Node::supports_action` | method | 1756 |  | `(Node).SupportsAction` | done |
| `Node::add_action` | method | 1760 |  | `(Node).AddAction` | done |
| `Node::remove_action` | method | 1764 |  | `(Node).RemoveAction` | done |
| `Node::clear_actions` | method | 1768 |  | `(Node).ClearActions` | done |
| `Node::child_supports_action` | method | 1775 |  | `(Node).ChildSupportsAction` | done |
| `Node::add_child_action` | method | 1781 |  | `(Node).AddChildAction` | done |
| `Node::remove_child_action` | method | 1787 |  | `(Node).RemoveChildAction` | done |
| `Node::clear_child_actions` | method | 1793 |  | `(Node).ClearChildActions` | done |
| `impl fmt::Debug for Node` | trait-impl | 2380 |  | `(Node).String` | done |
| `impl Serialize for Properties` | trait-impl | 2448 | serde | `(properties).MarshalJSONTo` | done |
| `impl Deserialize for Properties` | trait-impl | 2635 | serde | `(properties).UnmarshalJSONFrom` | done |
| `impl JsonSchema for Properties` | trait-impl | 2662 | schemars | `(properties).JSONSchema` | dev:D-SCHEMA |
| `Tree` | struct | 2793 |  | `Tree` | done |
| `Tree.root` | field | 2795 |  | `Tree.Root` | done |
| `Tree.toolkit_name` | field | 2797 |  | `Tree.ToolkitName` | done |
| `Tree.toolkit_version` | field | 2799 |  | `Tree.ToolkitVersion` | done |
| `Tree::new` | assoc-fn | 2804 |  | `NewTree` | done |
| `TreeUpdate` | struct | 2829 |  | `TreeUpdate` | done |
| `TreeUpdate.nodes` | field | 2850 |  | `TreeUpdate.Nodes` | done |
| `TreeUpdate.tree` | field | 2856 |  | `TreeUpdate.Tree` | done |
| `TreeUpdate.tree_id` | field | 2867 |  | `TreeUpdate.TreeID` | done |
| `TreeUpdate.focus` | field | 2878 |  | `TreeUpdate.Focus` | done |
| `ScrollUnit` | enum | 2892 |  | `ScrollUnit` | done |
| `ScrollHint` | enum | 2912 |  | `ScrollHint` | done |
| `ActionData` | enum | 2926 |  | `ActionData` | done |
| `ActionRequest` | struct | 2949 |  | `ActionRequest` | done |
| `ActionRequest.action` | field | 2950 |  | `ActionRequest.Action` | done |
| `ActionRequest.target_tree` | field | 2951 |  | `ActionRequest.TargetTree` | done |
| `ActionRequest.target_node` | field | 2952 |  | `ActionRequest.TargetNode` | done |
| `ActionRequest.data` | field | 2953 |  | `ActionRequest.Data` | done |
| `ActivationHandler` | trait | 2957 |  | `ActivationHandler` | done |
| `ActivationHandler::request_initial_tree` | trait-method | 2979 |  | `ActivationHandler.RequestInitialTree` | done |
| `ActionHandler` | trait | 2983 |  | `ActionHandler` | done |
| `ActionHandler::do_action` | trait-method | 2993 |  | `ActionHandler.DoAction` | done |
| `DeactivationHandler` | trait | 2997 |  | `DeactivationHandler` | done |
| `DeactivationHandler::deactivate_accessibility` | trait-method | 3005 |  | `DeactivationHandler.DeactivateAccessibility` | done |

## src/lib.rs: Node property methods and tests

Go target: `a11y/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Node::is_hidden` | method | 1801 |  | `(Node).IsHidden` | done |
| `Node::set_hidden` | method | 1801 |  | `(Node).SetHidden` | done |
| `Node::clear_hidden` | method | 1801 |  | `(Node).ClearHidden` | done |
| `Node::is_multiselectable` | method | 1802 |  | `(Node).IsMultiselectable` | done |
| `Node::set_multiselectable` | method | 1802 |  | `(Node).SetMultiselectable` | done |
| `Node::clear_multiselectable` | method | 1802 |  | `(Node).ClearMultiselectable` | done |
| `Node::is_required` | method | 1803 |  | `(Node).IsRequired` | done |
| `Node::set_required` | method | 1803 |  | `(Node).SetRequired` | done |
| `Node::clear_required` | method | 1803 |  | `(Node).ClearRequired` | done |
| `Node::is_visited` | method | 1804 |  | `(Node).IsVisited` | done |
| `Node::set_visited` | method | 1804 |  | `(Node).SetVisited` | done |
| `Node::clear_visited` | method | 1804 |  | `(Node).ClearVisited` | done |
| `Node::is_busy` | method | 1805 |  | `(Node).IsBusy` | done |
| `Node::set_busy` | method | 1805 |  | `(Node).SetBusy` | done |
| `Node::clear_busy` | method | 1805 |  | `(Node).ClearBusy` | done |
| `Node::is_live_atomic` | method | 1806 |  | `(Node).IsLiveAtomic` | done |
| `Node::set_live_atomic` | method | 1806 |  | `(Node).SetLiveAtomic` | done |
| `Node::clear_live_atomic` | method | 1806 |  | `(Node).ClearLiveAtomic` | done |
| `Node::is_modal` | method | 1808 |  | `(Node).IsModal` | done |
| `Node::set_modal` | method | 1808 |  | `(Node).SetModal` | done |
| `Node::clear_modal` | method | 1808 |  | `(Node).ClearModal` | done |
| `Node::is_touch_transparent` | method | 1812 |  | `(Node).IsTouchTransparent` | done |
| `Node::set_touch_transparent` | method | 1812 |  | `(Node).SetTouchTransparent` | done |
| `Node::clear_touch_transparent` | method | 1812 |  | `(Node).ClearTouchTransparent` | done |
| `Node::is_read_only` | method | 1814 |  | `(Node).IsReadOnly` | done |
| `Node::set_read_only` | method | 1814 |  | `(Node).SetReadOnly` | done |
| `Node::clear_read_only` | method | 1814 |  | `(Node).ClearReadOnly` | done |
| `Node::is_disabled` | method | 1816 |  | `(Node).IsDisabled` | done |
| `Node::set_disabled` | method | 1816 |  | `(Node).SetDisabled` | done |
| `Node::clear_disabled` | method | 1816 |  | `(Node).ClearDisabled` | done |
| `Node::is_italic` | method | 1817 |  | `(Node).IsItalic` | done |
| `Node::set_italic` | method | 1817 |  | `(Node).SetItalic` | done |
| `Node::clear_italic` | method | 1817 |  | `(Node).ClearItalic` | done |
| `Node::clips_children` | method | 1820 |  | `(Node).ClipsChildren` | done |
| `Node::set_clips_children` | method | 1820 |  | `(Node).SetClipsChildren` | done |
| `Node::clear_clips_children` | method | 1820 |  | `(Node).ClearClipsChildren` | done |
| `Node::is_line_breaking_object` | method | 1823 |  | `(Node).IsLineBreakingObject` | done |
| `Node::set_is_line_breaking_object` | method | 1823 |  | `(Node).SetIsLineBreakingObject` | done |
| `Node::clear_is_line_breaking_object` | method | 1823 |  | `(Node).ClearIsLineBreakingObject` | done |
| `Node::is_page_breaking_object` | method | 1825 |  | `(Node).IsPageBreakingObject` | done |
| `Node::set_is_page_breaking_object` | method | 1825 |  | `(Node).SetIsPageBreakingObject` | done |
| `Node::clear_is_page_breaking_object` | method | 1825 |  | `(Node).ClearIsPageBreakingObject` | done |
| `Node::is_spelling_error` | method | 1826 |  | `(Node).IsSpellingError` | done |
| `Node::set_is_spelling_error` | method | 1826 |  | `(Node).SetIsSpellingError` | done |
| `Node::clear_is_spelling_error` | method | 1826 |  | `(Node).ClearIsSpellingError` | done |
| `Node::is_grammar_error` | method | 1827 |  | `(Node).IsGrammarError` | done |
| `Node::set_is_grammar_error` | method | 1827 |  | `(Node).SetIsGrammarError` | done |
| `Node::clear_is_grammar_error` | method | 1827 |  | `(Node).ClearIsGrammarError` | done |
| `Node::is_search_match` | method | 1828 |  | `(Node).IsSearchMatch` | done |
| `Node::set_is_search_match` | method | 1828 |  | `(Node).SetIsSearchMatch` | done |
| `Node::clear_is_search_match` | method | 1828 |  | `(Node).ClearIsSearchMatch` | done |
| `Node::is_suggestion` | method | 1829 |  | `(Node).IsSuggestion` | done |
| `Node::set_is_suggestion` | method | 1829 |  | `(Node).SetIsSuggestion` | done |
| `Node::clear_is_suggestion` | method | 1829 |  | `(Node).ClearIsSuggestion` | done |
| `Node::children` | method | 1881 |  | `(Node).Children` | done |
| `Node::set_children` | method | 1881 |  | `(Node).SetChildren` | done |
| `Node::push_child` | method | 1881 |  | `(Node).PushChild` | done |
| `Node::clear_children` | method | 1881 |  | `(Node).ClearChildren` | done |
| `Node::controls` | method | 1882 |  | `(Node).Controls` | done |
| `Node::set_controls` | method | 1882 |  | `(Node).SetControls` | done |
| `Node::push_controlled` | method | 1882 |  | `(Node).PushControlled` | done |
| `Node::clear_controls` | method | 1882 |  | `(Node).ClearControls` | done |
| `Node::details` | method | 1883 |  | `(Node).Details` | done |
| `Node::set_details` | method | 1883 |  | `(Node).SetDetails` | done |
| `Node::push_detail` | method | 1883 |  | `(Node).PushDetail` | done |
| `Node::clear_details` | method | 1883 |  | `(Node).ClearDetails` | done |
| `Node::described_by` | method | 1884 |  | `(Node).DescribedBy` | done |
| `Node::set_described_by` | method | 1884 |  | `(Node).SetDescribedBy` | done |
| `Node::push_described_by` | method | 1884 |  | `(Node).PushDescribedBy` | done |
| `Node::clear_described_by` | method | 1884 |  | `(Node).ClearDescribedBy` | done |
| `Node::flow_to` | method | 1885 |  | `(Node).FlowTo` | done |
| `Node::set_flow_to` | method | 1885 |  | `(Node).SetFlowTo` | done |
| `Node::push_flow_to` | method | 1885 |  | `(Node).PushFlowTo` | done |
| `Node::clear_flow_to` | method | 1885 |  | `(Node).ClearFlowTo` | done |
| `Node::labelled_by` | method | 1886 |  | `(Node).LabelledBy` | done |
| `Node::set_labelled_by` | method | 1886 |  | `(Node).SetLabelledBy` | done |
| `Node::push_labelled_by` | method | 1886 |  | `(Node).PushLabelledBy` | done |
| `Node::clear_labelled_by` | method | 1886 |  | `(Node).ClearLabelledBy` | done |
| `Node::owns` | method | 1892 |  | `(Node).Owns` | done |
| `Node::set_owns` | method | 1892 |  | `(Node).SetOwns` | done |
| `Node::push_owned` | method | 1892 |  | `(Node).PushOwned` | done |
| `Node::clear_owns` | method | 1892 |  | `(Node).ClearOwns` | done |
| `Node::radio_group` | method | 1895 |  | `(Node).RadioGroup` | done |
| `Node::set_radio_group` | method | 1895 |  | `(Node).SetRadioGroup` | done |
| `Node::push_to_radio_group` | method | 1895 |  | `(Node).PushToRadioGroup` | done |
| `Node::clear_radio_group` | method | 1895 |  | `(Node).ClearRadioGroup` | done |
| `Node::active_descendant` | method | 1902 |  | `(Node).ActiveDescendant` | done |
| `Node::set_active_descendant` | method | 1902 |  | `(Node).SetActiveDescendant` | done |
| `Node::clear_active_descendant` | method | 1902 |  | `(Node).ClearActiveDescendant` | done |
| `Node::error_message` | method | 1903 |  | `(Node).ErrorMessage` | done |
| `Node::set_error_message` | method | 1903 |  | `(Node).SetErrorMessage` | done |
| `Node::clear_error_message` | method | 1903 |  | `(Node).ClearErrorMessage` | done |
| `Node::in_page_link_target` | method | 1904 |  | `(Node).InPageLinkTarget` | done |
| `Node::set_in_page_link_target` | method | 1904 |  | `(Node).SetInPageLinkTarget` | done |
| `Node::clear_in_page_link_target` | method | 1904 |  | `(Node).ClearInPageLinkTarget` | done |
| `Node::member_of` | method | 1905 |  | `(Node).MemberOf` | done |
| `Node::set_member_of` | method | 1905 |  | `(Node).SetMemberOf` | done |
| `Node::clear_member_of` | method | 1905 |  | `(Node).ClearMemberOf` | done |
| `Node::next_on_line` | method | 1906 |  | `(Node).NextOnLine` | done |
| `Node::set_next_on_line` | method | 1906 |  | `(Node).SetNextOnLine` | done |
| `Node::clear_next_on_line` | method | 1906 |  | `(Node).ClearNextOnLine` | done |
| `Node::previous_on_line` | method | 1907 |  | `(Node).PreviousOnLine` | done |
| `Node::set_previous_on_line` | method | 1907 |  | `(Node).SetPreviousOnLine` | done |
| `Node::clear_previous_on_line` | method | 1907 |  | `(Node).ClearPreviousOnLine` | done |
| `Node::popup_for` | method | 1908 |  | `(Node).PopupFor` | done |
| `Node::set_popup_for` | method | 1908 |  | `(Node).SetPopupFor` | done |
| `Node::clear_popup_for` | method | 1908 |  | `(Node).ClearPopupFor` | done |
| `Node::label` | method | 1916 |  | `(Node).Label` | done |
| `Node::set_label` | method | 1916 |  | `(Node).SetLabel` | done |
| `Node::clear_label` | method | 1916 |  | `(Node).ClearLabel` | done |
| `Node::description` | method | 1917 |  | `(Node).Description` | done |
| `Node::set_description` | method | 1917 |  | `(Node).SetDescription` | done |
| `Node::clear_description` | method | 1917 |  | `(Node).ClearDescription` | done |
| `Node::value` | method | 1918 |  | `(Node).Value` | done |
| `Node::set_value` | method | 1918 |  | `(Node).SetValue` | done |
| `Node::clear_value` | method | 1918 |  | `(Node).ClearValue` | done |
| `Node::access_key` | method | 1926 |  | `(Node).AccessKey` | done |
| `Node::set_access_key` | method | 1926 |  | `(Node).SetAccessKey` | done |
| `Node::clear_access_key` | method | 1926 |  | `(Node).ClearAccessKey` | done |
| `Node::author_id` | method | 1929 |  | `(Node).AuthorID` | done |
| `Node::set_author_id` | method | 1929 |  | `(Node).SetAuthorID` | done |
| `Node::clear_author_id` | method | 1929 |  | `(Node).ClearAuthorID` | done |
| `Node::class_name` | method | 1930 |  | `(Node).ClassName` | done |
| `Node::set_class_name` | method | 1930 |  | `(Node).SetClassName` | done |
| `Node::clear_class_name` | method | 1930 |  | `(Node).ClearClassName` | done |
| `Node::font_family` | method | 1932 |  | `(Node).FontFamily` | done |
| `Node::set_font_family` | method | 1932 |  | `(Node).SetFontFamily` | done |
| `Node::clear_font_family` | method | 1932 |  | `(Node).ClearFontFamily` | done |
| `Node::html_tag` | method | 1933 |  | `(Node).HtmlTag` | done |
| `Node::set_html_tag` | method | 1933 |  | `(Node).SetHtmlTag` | done |
| `Node::clear_html_tag` | method | 1933 |  | `(Node).ClearHtmlTag` | done |
| `Node::inner_html` | method | 1936 |  | `(Node).InnerHtml` | done |
| `Node::set_inner_html` | method | 1936 |  | `(Node).SetInnerHtml` | done |
| `Node::clear_inner_html` | method | 1936 |  | `(Node).ClearInnerHtml` | done |
| `Node::keyboard_shortcut` | method | 1940 |  | `(Node).KeyboardShortcut` | done |
| `Node::set_keyboard_shortcut` | method | 1940 |  | `(Node).SetKeyboardShortcut` | done |
| `Node::clear_keyboard_shortcut` | method | 1940 |  | `(Node).ClearKeyboardShortcut` | done |
| `Node::language` | method | 1943 |  | `(Node).Language` | done |
| `Node::set_language` | method | 1943 |  | `(Node).SetLanguage` | done |
| `Node::clear_language` | method | 1943 |  | `(Node).ClearLanguage` | done |
| `Node::placeholder` | method | 1948 |  | `(Node).Placeholder` | done |
| `Node::set_placeholder` | method | 1948 |  | `(Node).SetPlaceholder` | done |
| `Node::clear_placeholder` | method | 1948 |  | `(Node).ClearPlaceholder` | done |
| `Node::role_description` | method | 1952 |  | `(Node).RoleDescription` | done |
| `Node::set_role_description` | method | 1952 |  | `(Node).SetRoleDescription` | done |
| `Node::clear_role_description` | method | 1952 |  | `(Node).ClearRoleDescription` | done |
| `Node::state_description` | method | 1957 |  | `(Node).StateDescription` | done |
| `Node::set_state_description` | method | 1957 |  | `(Node).SetStateDescription` | done |
| `Node::clear_state_description` | method | 1957 |  | `(Node).ClearStateDescription` | done |
| `Node::tooltip` | method | 1962 |  | `(Node).Tooltip` | done |
| `Node::set_tooltip` | method | 1962 |  | `(Node).SetTooltip` | done |
| `Node::clear_tooltip` | method | 1962 |  | `(Node).ClearTooltip` | done |
| `Node::url` | method | 1963 |  | `(Node).URL` | done |
| `Node::set_url` | method | 1963 |  | `(Node).SetURL` | done |
| `Node::clear_url` | method | 1963 |  | `(Node).ClearURL` | done |
| `Node::row_index_text` | method | 1964 |  | `(Node).RowIndexText` | done |
| `Node::set_row_index_text` | method | 1964 |  | `(Node).SetRowIndexText` | done |
| `Node::clear_row_index_text` | method | 1964 |  | `(Node).ClearRowIndexText` | done |
| `Node::column_index_text` | method | 1965 |  | `(Node).ColumnIndexText` | done |
| `Node::set_column_index_text` | method | 1965 |  | `(Node).SetColumnIndexText` | done |
| `Node::clear_column_index_text` | method | 1965 |  | `(Node).ClearColumnIndexText` | done |
| `Node::braille_label` | method | 1966 |  | `(Node).BrailleLabel` | done |
| `Node::set_braille_label` | method | 1966 |  | `(Node).SetBrailleLabel` | done |
| `Node::clear_braille_label` | method | 1966 |  | `(Node).ClearBrailleLabel` | done |
| `Node::braille_role_description` | method | 1967 |  | `(Node).BrailleRoleDescription` | done |
| `Node::set_braille_role_description` | method | 1967 |  | `(Node).SetBrailleRoleDescription` | done |
| `Node::clear_braille_role_description` | method | 1967 |  | `(Node).ClearBrailleRoleDescription` | done |
| `Node::scroll_x` | method | 1971 |  | `(Node).ScrollX` | done |
| `Node::set_scroll_x` | method | 1971 |  | `(Node).SetScrollX` | done |
| `Node::clear_scroll_x` | method | 1971 |  | `(Node).ClearScrollX` | done |
| `Node::scroll_x_min` | method | 1972 |  | `(Node).ScrollXMin` | done |
| `Node::set_scroll_x_min` | method | 1972 |  | `(Node).SetScrollXMin` | done |
| `Node::clear_scroll_x_min` | method | 1972 |  | `(Node).ClearScrollXMin` | done |
| `Node::scroll_x_max` | method | 1973 |  | `(Node).ScrollXMax` | done |
| `Node::set_scroll_x_max` | method | 1973 |  | `(Node).SetScrollXMax` | done |
| `Node::clear_scroll_x_max` | method | 1973 |  | `(Node).ClearScrollXMax` | done |
| `Node::scroll_y` | method | 1974 |  | `(Node).ScrollY` | done |
| `Node::set_scroll_y` | method | 1974 |  | `(Node).SetScrollY` | done |
| `Node::clear_scroll_y` | method | 1974 |  | `(Node).ClearScrollY` | done |
| `Node::scroll_y_min` | method | 1975 |  | `(Node).ScrollYMin` | done |
| `Node::set_scroll_y_min` | method | 1975 |  | `(Node).SetScrollYMin` | done |
| `Node::clear_scroll_y_min` | method | 1975 |  | `(Node).ClearScrollYMin` | done |
| `Node::scroll_y_max` | method | 1976 |  | `(Node).ScrollYMax` | done |
| `Node::set_scroll_y_max` | method | 1976 |  | `(Node).SetScrollYMax` | done |
| `Node::clear_scroll_y_max` | method | 1976 |  | `(Node).ClearScrollYMax` | done |
| `Node::numeric_value` | method | 1977 |  | `(Node).NumericValue` | done |
| `Node::set_numeric_value` | method | 1977 |  | `(Node).SetNumericValue` | done |
| `Node::clear_numeric_value` | method | 1977 |  | `(Node).ClearNumericValue` | done |
| `Node::min_numeric_value` | method | 1978 |  | `(Node).MinNumericValue` | done |
| `Node::set_min_numeric_value` | method | 1978 |  | `(Node).SetMinNumericValue` | done |
| `Node::clear_min_numeric_value` | method | 1978 |  | `(Node).ClearMinNumericValue` | done |
| `Node::max_numeric_value` | method | 1979 |  | `(Node).MaxNumericValue` | done |
| `Node::set_max_numeric_value` | method | 1979 |  | `(Node).SetMaxNumericValue` | done |
| `Node::clear_max_numeric_value` | method | 1979 |  | `(Node).ClearMaxNumericValue` | done |
| `Node::numeric_value_step` | method | 1980 |  | `(Node).NumericValueStep` | done |
| `Node::set_numeric_value_step` | method | 1980 |  | `(Node).SetNumericValueStep` | done |
| `Node::clear_numeric_value_step` | method | 1980 |  | `(Node).ClearNumericValueStep` | done |
| `Node::numeric_value_jump` | method | 1981 |  | `(Node).NumericValueJump` | done |
| `Node::set_numeric_value_jump` | method | 1981 |  | `(Node).SetNumericValueJump` | done |
| `Node::clear_numeric_value_jump` | method | 1981 |  | `(Node).ClearNumericValueJump` | done |
| `Node::font_size` | method | 1986 |  | `(Node).FontSize` | done |
| `Node::set_font_size` | method | 1986 |  | `(Node).SetFontSize` | done |
| `Node::clear_font_size` | method | 1986 |  | `(Node).ClearFontSize` | done |
| `Node::font_weight` | method | 1989 |  | `(Node).FontWeight` | done |
| `Node::set_font_weight` | method | 1989 |  | `(Node).SetFontWeight` | done |
| `Node::clear_font_weight` | method | 1989 |  | `(Node).ClearFontWeight` | done |
| `Node::row_count` | method | 1993 |  | `(Node).RowCount` | done |
| `Node::set_row_count` | method | 1993 |  | `(Node).SetRowCount` | done |
| `Node::clear_row_count` | method | 1993 |  | `(Node).ClearRowCount` | done |
| `Node::column_count` | method | 1994 |  | `(Node).ColumnCount` | done |
| `Node::set_column_count` | method | 1994 |  | `(Node).SetColumnCount` | done |
| `Node::clear_column_count` | method | 1994 |  | `(Node).ClearColumnCount` | done |
| `Node::row_index` | method | 1995 |  | `(Node).RowIndex` | done |
| `Node::set_row_index` | method | 1995 |  | `(Node).SetRowIndex` | done |
| `Node::clear_row_index` | method | 1995 |  | `(Node).ClearRowIndex` | done |
| `Node::column_index` | method | 1996 |  | `(Node).ColumnIndex` | done |
| `Node::set_column_index` | method | 1996 |  | `(Node).SetColumnIndex` | done |
| `Node::clear_column_index` | method | 1996 |  | `(Node).ClearColumnIndex` | done |
| `Node::row_span` | method | 1997 |  | `(Node).RowSpan` | done |
| `Node::set_row_span` | method | 1997 |  | `(Node).SetRowSpan` | done |
| `Node::clear_row_span` | method | 1997 |  | `(Node).ClearRowSpan` | done |
| `Node::column_span` | method | 1998 |  | `(Node).ColumnSpan` | done |
| `Node::set_column_span` | method | 1998 |  | `(Node).SetColumnSpan` | done |
| `Node::clear_column_span` | method | 1998 |  | `(Node).ClearColumnSpan` | done |
| `Node::level` | method | 1999 |  | `(Node).Level` | done |
| `Node::set_level` | method | 1999 |  | `(Node).SetLevel` | done |
| `Node::clear_level` | method | 1999 |  | `(Node).ClearLevel` | done |
| `Node::size_of_set` | method | 2001 |  | `(Node).SizeOfSet` | done |
| `Node::set_size_of_set` | method | 2001 |  | `(Node).SetSizeOfSet` | done |
| `Node::clear_size_of_set` | method | 2001 |  | `(Node).ClearSizeOfSet` | done |
| `Node::position_in_set` | method | 2006 |  | `(Node).PositionInSet` | done |
| `Node::set_position_in_set` | method | 2006 |  | `(Node).SetPositionInSet` | done |
| `Node::clear_position_in_set` | method | 2006 |  | `(Node).ClearPositionInSet` | done |
| `Node::color_value` | method | 2011 |  | `(Node).ColorValue` | done |
| `Node::set_color_value` | method | 2011 |  | `(Node).SetColorValue` | done |
| `Node::clear_color_value` | method | 2011 |  | `(Node).ClearColorValue` | done |
| `Node::background_color` | method | 2013 |  | `(Node).BackgroundColor` | done |
| `Node::set_background_color` | method | 2013 |  | `(Node).SetBackgroundColor` | done |
| `Node::clear_background_color` | method | 2013 |  | `(Node).ClearBackgroundColor` | done |
| `Node::foreground_color` | method | 2015 |  | `(Node).ForegroundColor` | done |
| `Node::set_foreground_color` | method | 2015 |  | `(Node).SetForegroundColor` | done |
| `Node::clear_foreground_color` | method | 2015 |  | `(Node).ClearForegroundColor` | done |
| `Node::overline` | method | 2019 |  | `(Node).Overline` | done |
| `Node::set_overline` | method | 2019 |  | `(Node).SetOverline` | done |
| `Node::clear_overline` | method | 2019 |  | `(Node).ClearOverline` | done |
| `Node::strikethrough` | method | 2020 |  | `(Node).Strikethrough` | done |
| `Node::set_strikethrough` | method | 2020 |  | `(Node).SetStrikethrough` | done |
| `Node::clear_strikethrough` | method | 2020 |  | `(Node).ClearStrikethrough` | done |
| `Node::underline` | method | 2021 |  | `(Node).Underline` | done |
| `Node::set_underline` | method | 2021 |  | `(Node).SetUnderline` | done |
| `Node::clear_underline` | method | 2021 |  | `(Node).ClearUnderline` | done |
| `Node::character_lengths` | method | 2043 |  | `(Node).CharacterLengths` | done |
| `Node::set_character_lengths` | method | 2043 |  | `(Node).SetCharacterLengths` | done |
| `Node::clear_character_lengths` | method | 2043 |  | `(Node).ClearCharacterLengths` | done |
| `Node::word_starts` | method | 2074 |  | `(Node).WordStarts` | done |
| `Node::set_word_starts` | method | 2074 |  | `(Node).SetWordStarts` | done |
| `Node::clear_word_starts` | method | 2074 |  | `(Node).ClearWordStarts` | done |
| `Node::character_positions` | method | 2094 |  | `(Node).CharacterPositions` | done |
| `Node::set_character_positions` | method | 2094 |  | `(Node).SetCharacterPositions` | done |
| `Node::clear_character_positions` | method | 2094 |  | `(Node).ClearCharacterPositions` | done |
| `Node::character_widths` | method | 2114 |  | `(Node).CharacterWidths` | done |
| `Node::set_character_widths` | method | 2114 |  | `(Node).SetCharacterWidths` | done |
| `Node::clear_character_widths` | method | 2114 |  | `(Node).ClearCharacterWidths` | done |
| `Node::is_expanded` | method | 2122 |  | `(Node).IsExpanded` | done |
| `Node::set_expanded` | method | 2122 |  | `(Node).SetExpanded` | done |
| `Node::clear_expanded` | method | 2122 |  | `(Node).ClearExpanded` | done |
| `Node::is_selected` | method | 2133 |  | `(Node).IsSelected` | done |
| `Node::set_selected` | method | 2133 |  | `(Node).SetSelected` | done |
| `Node::clear_selected` | method | 2133 |  | `(Node).ClearSelected` | done |
| `Node::invalid` | method | 2137 |  | `(Node).Invalid` | done |
| `Node::set_invalid` | method | 2137 |  | `(Node).SetInvalid` | done |
| `Node::clear_invalid` | method | 2137 |  | `(Node).ClearInvalid` | done |
| `Node::toggled` | method | 2138 |  | `(Node).Toggled` | done |
| `Node::set_toggled` | method | 2138 |  | `(Node).SetToggled` | done |
| `Node::clear_toggled` | method | 2138 |  | `(Node).ClearToggled` | done |
| `Node::live` | method | 2139 |  | `(Node).Live` | done |
| `Node::set_live` | method | 2139 |  | `(Node).SetLive` | done |
| `Node::clear_live` | method | 2139 |  | `(Node).ClearLive` | done |
| `Node::text_direction` | method | 2140 |  | `(Node).TextDirection` | done |
| `Node::set_text_direction` | method | 2140 |  | `(Node).SetTextDirection` | done |
| `Node::clear_text_direction` | method | 2140 |  | `(Node).ClearTextDirection` | done |
| `Node::orientation` | method | 2141 |  | `(Node).Orientation` | done |
| `Node::set_orientation` | method | 2141 |  | `(Node).SetOrientation` | done |
| `Node::clear_orientation` | method | 2141 |  | `(Node).ClearOrientation` | done |
| `Node::sort_direction` | method | 2142 |  | `(Node).SortDirection` | done |
| `Node::set_sort_direction` | method | 2142 |  | `(Node).SetSortDirection` | done |
| `Node::clear_sort_direction` | method | 2142 |  | `(Node).ClearSortDirection` | done |
| `Node::aria_current` | method | 2143 |  | `(Node).AriaCurrent` | done |
| `Node::set_aria_current` | method | 2143 |  | `(Node).SetAriaCurrent` | done |
| `Node::clear_aria_current` | method | 2143 |  | `(Node).ClearAriaCurrent` | done |
| `Node::auto_complete` | method | 2144 |  | `(Node).AutoComplete` | done |
| `Node::set_auto_complete` | method | 2144 |  | `(Node).SetAutoComplete` | done |
| `Node::clear_auto_complete` | method | 2144 |  | `(Node).ClearAutoComplete` | done |
| `Node::has_popup` | method | 2145 |  | `(Node).HasPopup` | done |
| `Node::set_has_popup` | method | 2145 |  | `(Node).SetHasPopup` | done |
| `Node::clear_has_popup` | method | 2145 |  | `(Node).ClearHasPopup` | done |
| `Node::list_style` | method | 2147 |  | `(Node).ListStyle` | done |
| `Node::set_list_style` | method | 2147 |  | `(Node).SetListStyle` | done |
| `Node::clear_list_style` | method | 2147 |  | `(Node).ClearListStyle` | done |
| `Node::text_align` | method | 2148 |  | `(Node).TextAlign` | done |
| `Node::set_text_align` | method | 2148 |  | `(Node).SetTextAlign` | done |
| `Node::clear_text_align` | method | 2148 |  | `(Node).ClearTextAlign` | done |
| `Node::vertical_offset` | method | 2149 |  | `(Node).VerticalOffset` | done |
| `Node::set_vertical_offset` | method | 2149 |  | `(Node).SetVerticalOffset` | done |
| `Node::clear_vertical_offset` | method | 2149 |  | `(Node).ClearVerticalOffset` | done |
| `Node::transform` | method | 2165 |  | `(Node).Transform` | done |
| `Node::set_transform` | method | 2165 |  | `(Node).SetTransform` | done |
| `Node::clear_transform` | method | 2165 |  | `(Node).ClearTransform` | done |
| `Node::bounds` | method | 2177 |  | `(Node).Bounds` | done |
| `Node::set_bounds` | method | 2177 |  | `(Node).SetBounds` | done |
| `Node::clear_bounds` | method | 2177 |  | `(Node).ClearBounds` | done |
| `Node::text_selection` | method | 2179 |  | `(Node).TextSelection` | done |
| `Node::set_text_selection` | method | 2179 |  | `(Node).SetTextSelection` | done |
| `Node::clear_text_selection` | method | 2179 |  | `(Node).ClearTextSelection` | done |
| `Node::tree_id` | method | 2188 |  | `(Node).TreeID` | done |
| `Node::set_tree_id` | method | 2188 |  | `(Node).SetTreeID` | done |
| `Node::clear_tree_id` | method | 2188 |  | `(Node).ClearTreeID` | done |
| `Node::custom_actions` | method | 2323 |  | `(Node).CustomActions` | done |
| `Node::set_custom_actions` | method | 2323 |  | `(Node).SetCustomActions` | done |
| `Node::push_custom_action` | method | 2323 |  | `(Node).PushCustomAction` | done |
| `Node::clear_custom_actions` | method | 2323 |  | `(Node).ClearCustomActions` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `is_hidden::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsHidden_GetterShouldReturnDefaultValue` | done |
| `is_hidden::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsHidden_SetterShouldUpdateTheProperty` | done |
| `is_hidden::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsHidden_ClearerShouldResetTheProperty` | done |
| `is_multiselectable::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsMultiselectable_GetterShouldReturnDefaultValue` | done |
| `is_multiselectable::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsMultiselectable_SetterShouldUpdateTheProperty` | done |
| `is_multiselectable::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsMultiselectable_ClearerShouldResetTheProperty` | done |
| `is_required::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsRequired_GetterShouldReturnDefaultValue` | done |
| `is_required::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsRequired_SetterShouldUpdateTheProperty` | done |
| `is_required::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsRequired_ClearerShouldResetTheProperty` | done |
| `is_visited::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsVisited_GetterShouldReturnDefaultValue` | done |
| `is_visited::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsVisited_SetterShouldUpdateTheProperty` | done |
| `is_visited::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsVisited_ClearerShouldResetTheProperty` | done |
| `is_busy::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsBusy_GetterShouldReturnDefaultValue` | done |
| `is_busy::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsBusy_SetterShouldUpdateTheProperty` | done |
| `is_busy::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsBusy_ClearerShouldResetTheProperty` | done |
| `is_live_atomic::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsLiveAtomic_GetterShouldReturnDefaultValue` | done |
| `is_live_atomic::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsLiveAtomic_SetterShouldUpdateTheProperty` | done |
| `is_live_atomic::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsLiveAtomic_ClearerShouldResetTheProperty` | done |
| `is_modal::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsModal_GetterShouldReturnDefaultValue` | done |
| `is_modal::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsModal_SetterShouldUpdateTheProperty` | done |
| `is_modal::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsModal_ClearerShouldResetTheProperty` | done |
| `is_touch_transparent::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsTouchTransparent_GetterShouldReturnDefaultValue` | done |
| `is_touch_transparent::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsTouchTransparent_SetterShouldUpdateTheProperty` | done |
| `is_touch_transparent::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsTouchTransparent_ClearerShouldResetTheProperty` | done |
| `is_read_only::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsReadOnly_GetterShouldReturnDefaultValue` | done |
| `is_read_only::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsReadOnly_SetterShouldUpdateTheProperty` | done |
| `is_read_only::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsReadOnly_ClearerShouldResetTheProperty` | done |
| `is_disabled::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsDisabled_GetterShouldReturnDefaultValue` | done |
| `is_disabled::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsDisabled_SetterShouldUpdateTheProperty` | done |
| `is_disabled::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsDisabled_ClearerShouldResetTheProperty` | done |
| `is_italic::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsItalic_GetterShouldReturnDefaultValue` | done |
| `is_italic::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsItalic_SetterShouldUpdateTheProperty` | done |
| `is_italic::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsItalic_ClearerShouldResetTheProperty` | done |
| `clips_children::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_ClipsChildren_GetterShouldReturnDefaultValue` | done |
| `clips_children::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_ClipsChildren_SetterShouldUpdateTheProperty` | done |
| `clips_children::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_ClipsChildren_ClearerShouldResetTheProperty` | done |
| `is_line_breaking_object::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsLineBreakingObject_GetterShouldReturnDefaultValue` | done |
| `is_line_breaking_object::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsLineBreakingObject_SetterShouldUpdateTheProperty` | done |
| `is_line_breaking_object::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsLineBreakingObject_ClearerShouldResetTheProperty` | done |
| `is_page_breaking_object::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsPageBreakingObject_GetterShouldReturnDefaultValue` | done |
| `is_page_breaking_object::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsPageBreakingObject_SetterShouldUpdateTheProperty` | done |
| `is_page_breaking_object::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsPageBreakingObject_ClearerShouldResetTheProperty` | done |
| `is_spelling_error::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsSpellingError_GetterShouldReturnDefaultValue` | done |
| `is_spelling_error::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsSpellingError_SetterShouldUpdateTheProperty` | done |
| `is_spelling_error::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsSpellingError_ClearerShouldResetTheProperty` | done |
| `is_grammar_error::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsGrammarError_GetterShouldReturnDefaultValue` | done |
| `is_grammar_error::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsGrammarError_SetterShouldUpdateTheProperty` | done |
| `is_grammar_error::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsGrammarError_ClearerShouldResetTheProperty` | done |
| `is_search_match::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsSearchMatch_GetterShouldReturnDefaultValue` | done |
| `is_search_match::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsSearchMatch_SetterShouldUpdateTheProperty` | done |
| `is_search_match::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsSearchMatch_ClearerShouldResetTheProperty` | done |
| `is_suggestion::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsSuggestion_GetterShouldReturnDefaultValue` | done |
| `is_suggestion::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsSuggestion_SetterShouldUpdateTheProperty` | done |
| `is_suggestion::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsSuggestion_ClearerShouldResetTheProperty` | done |
| `children::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_Children_GetterShouldReturnDefaultValue` | done |
| `children::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_Children_SetterShouldUpdateTheProperty` | done |
| `children::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_Children_PusherShouldUpdateTheProperty` | done |
| `children::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_Children_ClearerShouldResetTheProperty` | done |
| `controls::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_Controls_GetterShouldReturnDefaultValue` | done |
| `controls::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_Controls_SetterShouldUpdateTheProperty` | done |
| `controls::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_Controls_PusherShouldUpdateTheProperty` | done |
| `controls::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_Controls_ClearerShouldResetTheProperty` | done |
| `details::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_Details_GetterShouldReturnDefaultValue` | done |
| `details::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_Details_SetterShouldUpdateTheProperty` | done |
| `details::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_Details_PusherShouldUpdateTheProperty` | done |
| `details::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_Details_ClearerShouldResetTheProperty` | done |
| `described_by::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_DescribedBy_GetterShouldReturnDefaultValue` | done |
| `described_by::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_DescribedBy_SetterShouldUpdateTheProperty` | done |
| `described_by::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_DescribedBy_PusherShouldUpdateTheProperty` | done |
| `described_by::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_DescribedBy_ClearerShouldResetTheProperty` | done |
| `flow_to::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_FlowTo_GetterShouldReturnDefaultValue` | done |
| `flow_to::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_FlowTo_SetterShouldUpdateTheProperty` | done |
| `flow_to::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_FlowTo_PusherShouldUpdateTheProperty` | done |
| `flow_to::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_FlowTo_ClearerShouldResetTheProperty` | done |
| `labelled_by::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_LabelledBy_GetterShouldReturnDefaultValue` | done |
| `labelled_by::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_LabelledBy_SetterShouldUpdateTheProperty` | done |
| `labelled_by::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_LabelledBy_PusherShouldUpdateTheProperty` | done |
| `labelled_by::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_LabelledBy_ClearerShouldResetTheProperty` | done |
| `owns::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_Owns_GetterShouldReturnDefaultValue` | done |
| `owns::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_Owns_SetterShouldUpdateTheProperty` | done |
| `owns::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_Owns_PusherShouldUpdateTheProperty` | done |
| `owns::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_Owns_ClearerShouldResetTheProperty` | done |
| `radio_group::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_RadioGroup_GetterShouldReturnDefaultValue` | done |
| `radio_group::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_RadioGroup_SetterShouldUpdateTheProperty` | done |
| `radio_group::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_RadioGroup_PusherShouldUpdateTheProperty` | done |
| `radio_group::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_RadioGroup_ClearerShouldResetTheProperty` | done |
| `active_descendant::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_ActiveDescendant_GetterShouldReturnDefaultValue` | done |
| `active_descendant::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_ActiveDescendant_SetterShouldUpdateTheProperty` | done |
| `active_descendant::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_ActiveDescendant_ClearerShouldResetTheProperty` | done |
| `error_message::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_ErrorMessage_GetterShouldReturnDefaultValue` | done |
| `error_message::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_ErrorMessage_SetterShouldUpdateTheProperty` | done |
| `error_message::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_ErrorMessage_ClearerShouldResetTheProperty` | done |
| `in_page_link_target::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_InPageLinkTarget_GetterShouldReturnDefaultValue` | done |
| `in_page_link_target::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_InPageLinkTarget_SetterShouldUpdateTheProperty` | done |
| `in_page_link_target::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_InPageLinkTarget_ClearerShouldResetTheProperty` | done |
| `member_of::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_MemberOf_GetterShouldReturnDefaultValue` | done |
| `member_of::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_MemberOf_SetterShouldUpdateTheProperty` | done |
| `member_of::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_MemberOf_ClearerShouldResetTheProperty` | done |
| `next_on_line::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_NextOnLine_GetterShouldReturnDefaultValue` | done |
| `next_on_line::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_NextOnLine_SetterShouldUpdateTheProperty` | done |
| `next_on_line::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_NextOnLine_ClearerShouldResetTheProperty` | done |
| `previous_on_line::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_PreviousOnLine_GetterShouldReturnDefaultValue` | done |
| `previous_on_line::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_PreviousOnLine_SetterShouldUpdateTheProperty` | done |
| `previous_on_line::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_PreviousOnLine_ClearerShouldResetTheProperty` | done |
| `popup_for::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_PopupFor_GetterShouldReturnDefaultValue` | done |
| `popup_for::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_PopupFor_SetterShouldUpdateTheProperty` | done |
| `popup_for::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_PopupFor_ClearerShouldResetTheProperty` | done |
| `label::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Label_GetterShouldReturnDefaultValue` | done |
| `label::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Label_SetterShouldUpdateTheProperty` | done |
| `label::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Label_ClearerShouldResetTheProperty` | done |
| `description::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Description_GetterShouldReturnDefaultValue` | done |
| `description::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Description_SetterShouldUpdateTheProperty` | done |
| `description::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Description_ClearerShouldResetTheProperty` | done |
| `value::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Value_GetterShouldReturnDefaultValue` | done |
| `value::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Value_SetterShouldUpdateTheProperty` | done |
| `value::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Value_ClearerShouldResetTheProperty` | done |
| `access_key::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_AccessKey_GetterShouldReturnDefaultValue` | done |
| `access_key::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_AccessKey_SetterShouldUpdateTheProperty` | done |
| `access_key::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_AccessKey_ClearerShouldResetTheProperty` | done |
| `author_id::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_AuthorID_GetterShouldReturnDefaultValue` | done |
| `author_id::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_AuthorID_SetterShouldUpdateTheProperty` | done |
| `author_id::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_AuthorID_ClearerShouldResetTheProperty` | done |
| `class_name::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_ClassName_GetterShouldReturnDefaultValue` | done |
| `class_name::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_ClassName_SetterShouldUpdateTheProperty` | done |
| `class_name::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_ClassName_ClearerShouldResetTheProperty` | done |
| `font_family::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_FontFamily_GetterShouldReturnDefaultValue` | done |
| `font_family::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_FontFamily_SetterShouldUpdateTheProperty` | done |
| `font_family::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_FontFamily_ClearerShouldResetTheProperty` | done |
| `html_tag::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_HtmlTag_GetterShouldReturnDefaultValue` | done |
| `html_tag::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_HtmlTag_SetterShouldUpdateTheProperty` | done |
| `html_tag::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_HtmlTag_ClearerShouldResetTheProperty` | done |
| `inner_html::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_InnerHtml_GetterShouldReturnDefaultValue` | done |
| `inner_html::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_InnerHtml_SetterShouldUpdateTheProperty` | done |
| `inner_html::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_InnerHtml_ClearerShouldResetTheProperty` | done |
| `keyboard_shortcut::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_KeyboardShortcut_GetterShouldReturnDefaultValue` | done |
| `keyboard_shortcut::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_KeyboardShortcut_SetterShouldUpdateTheProperty` | done |
| `keyboard_shortcut::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_KeyboardShortcut_ClearerShouldResetTheProperty` | done |
| `language::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Language_GetterShouldReturnDefaultValue` | done |
| `language::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Language_SetterShouldUpdateTheProperty` | done |
| `language::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Language_ClearerShouldResetTheProperty` | done |
| `placeholder::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Placeholder_GetterShouldReturnDefaultValue` | done |
| `placeholder::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Placeholder_SetterShouldUpdateTheProperty` | done |
| `placeholder::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Placeholder_ClearerShouldResetTheProperty` | done |
| `role_description::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_RoleDescription_GetterShouldReturnDefaultValue` | done |
| `role_description::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_RoleDescription_SetterShouldUpdateTheProperty` | done |
| `role_description::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_RoleDescription_ClearerShouldResetTheProperty` | done |
| `state_description::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_StateDescription_GetterShouldReturnDefaultValue` | done |
| `state_description::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_StateDescription_SetterShouldUpdateTheProperty` | done |
| `state_description::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_StateDescription_ClearerShouldResetTheProperty` | done |
| `tooltip::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Tooltip_GetterShouldReturnDefaultValue` | done |
| `tooltip::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Tooltip_SetterShouldUpdateTheProperty` | done |
| `tooltip::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Tooltip_ClearerShouldResetTheProperty` | done |
| `url::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_URL_GetterShouldReturnDefaultValue` | done |
| `url::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_URL_SetterShouldUpdateTheProperty` | done |
| `url::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_URL_ClearerShouldResetTheProperty` | done |
| `row_index_text::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_RowIndexText_GetterShouldReturnDefaultValue` | done |
| `row_index_text::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_RowIndexText_SetterShouldUpdateTheProperty` | done |
| `row_index_text::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_RowIndexText_ClearerShouldResetTheProperty` | done |
| `column_index_text::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_ColumnIndexText_GetterShouldReturnDefaultValue` | done |
| `column_index_text::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_ColumnIndexText_SetterShouldUpdateTheProperty` | done |
| `column_index_text::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_ColumnIndexText_ClearerShouldResetTheProperty` | done |
| `braille_label::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_BrailleLabel_GetterShouldReturnDefaultValue` | done |
| `braille_label::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_BrailleLabel_SetterShouldUpdateTheProperty` | done |
| `braille_label::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_BrailleLabel_ClearerShouldResetTheProperty` | done |
| `braille_role_description::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_BrailleRoleDescription_GetterShouldReturnDefaultValue` | done |
| `braille_role_description::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_BrailleRoleDescription_SetterShouldUpdateTheProperty` | done |
| `braille_role_description::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_BrailleRoleDescription_ClearerShouldResetTheProperty` | done |
| `scroll_x::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_ScrollX_GetterShouldReturnDefaultValue` | done |
| `scroll_x::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_ScrollX_SetterShouldUpdateTheProperty` | done |
| `scroll_x::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_ScrollX_ClearerShouldResetTheProperty` | done |
| `scroll_x_min::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_ScrollXMin_GetterShouldReturnDefaultValue` | done |
| `scroll_x_min::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_ScrollXMin_SetterShouldUpdateTheProperty` | done |
| `scroll_x_min::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_ScrollXMin_ClearerShouldResetTheProperty` | done |
| `scroll_x_max::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_ScrollXMax_GetterShouldReturnDefaultValue` | done |
| `scroll_x_max::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_ScrollXMax_SetterShouldUpdateTheProperty` | done |
| `scroll_x_max::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_ScrollXMax_ClearerShouldResetTheProperty` | done |
| `scroll_y::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_ScrollY_GetterShouldReturnDefaultValue` | done |
| `scroll_y::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_ScrollY_SetterShouldUpdateTheProperty` | done |
| `scroll_y::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_ScrollY_ClearerShouldResetTheProperty` | done |
| `scroll_y_min::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_ScrollYMin_GetterShouldReturnDefaultValue` | done |
| `scroll_y_min::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_ScrollYMin_SetterShouldUpdateTheProperty` | done |
| `scroll_y_min::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_ScrollYMin_ClearerShouldResetTheProperty` | done |
| `scroll_y_max::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_ScrollYMax_GetterShouldReturnDefaultValue` | done |
| `scroll_y_max::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_ScrollYMax_SetterShouldUpdateTheProperty` | done |
| `scroll_y_max::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_ScrollYMax_ClearerShouldResetTheProperty` | done |
| `numeric_value::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_NumericValue_GetterShouldReturnDefaultValue` | done |
| `numeric_value::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_NumericValue_SetterShouldUpdateTheProperty` | done |
| `numeric_value::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_NumericValue_ClearerShouldResetTheProperty` | done |
| `min_numeric_value::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_MinNumericValue_GetterShouldReturnDefaultValue` | done |
| `min_numeric_value::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_MinNumericValue_SetterShouldUpdateTheProperty` | done |
| `min_numeric_value::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_MinNumericValue_ClearerShouldResetTheProperty` | done |
| `max_numeric_value::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_MaxNumericValue_GetterShouldReturnDefaultValue` | done |
| `max_numeric_value::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_MaxNumericValue_SetterShouldUpdateTheProperty` | done |
| `max_numeric_value::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_MaxNumericValue_ClearerShouldResetTheProperty` | done |
| `numeric_value_step::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_NumericValueStep_GetterShouldReturnDefaultValue` | done |
| `numeric_value_step::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_NumericValueStep_SetterShouldUpdateTheProperty` | done |
| `numeric_value_step::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_NumericValueStep_ClearerShouldResetTheProperty` | done |
| `numeric_value_jump::getter_should_return_default_value` | test | 1405 | cfg(test) | `TestLib_NumericValueJump_GetterShouldReturnDefaultValue` | done |
| `numeric_value_jump::setter_should_update_the_property` | test | 1410 | cfg(test) | `TestLib_NumericValueJump_SetterShouldUpdateTheProperty` | done |
| `numeric_value_jump::clearer_should_reset_the_property` | test | 1416 | cfg(test) | `TestLib_NumericValueJump_ClearerShouldResetTheProperty` | done |
| `font_size::getter_should_return_default_value` | test | 1440 | cfg(test) | `TestLib_FontSize_GetterShouldReturnDefaultValue` | done |
| `font_size::setter_should_update_the_property` | test | 1445 | cfg(test) | `TestLib_FontSize_SetterShouldUpdateTheProperty` | done |
| `font_size::clearer_should_reset_the_property` | test | 1451 | cfg(test) | `TestLib_FontSize_ClearerShouldResetTheProperty` | done |
| `font_weight::getter_should_return_default_value` | test | 1440 | cfg(test) | `TestLib_FontWeight_GetterShouldReturnDefaultValue` | done |
| `font_weight::setter_should_update_the_property` | test | 1445 | cfg(test) | `TestLib_FontWeight_SetterShouldUpdateTheProperty` | done |
| `font_weight::clearer_should_reset_the_property` | test | 1451 | cfg(test) | `TestLib_FontWeight_ClearerShouldResetTheProperty` | done |
| `row_count::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_RowCount_GetterShouldReturnDefaultValue` | done |
| `row_count::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_RowCount_SetterShouldUpdateTheProperty` | done |
| `row_count::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_RowCount_ClearerShouldResetTheProperty` | done |
| `column_count::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_ColumnCount_GetterShouldReturnDefaultValue` | done |
| `column_count::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_ColumnCount_SetterShouldUpdateTheProperty` | done |
| `column_count::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_ColumnCount_ClearerShouldResetTheProperty` | done |
| `row_index::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_RowIndex_GetterShouldReturnDefaultValue` | done |
| `row_index::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_RowIndex_SetterShouldUpdateTheProperty` | done |
| `row_index::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_RowIndex_ClearerShouldResetTheProperty` | done |
| `column_index::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_ColumnIndex_GetterShouldReturnDefaultValue` | done |
| `column_index::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_ColumnIndex_SetterShouldUpdateTheProperty` | done |
| `column_index::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_ColumnIndex_ClearerShouldResetTheProperty` | done |
| `row_span::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_RowSpan_GetterShouldReturnDefaultValue` | done |
| `row_span::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_RowSpan_SetterShouldUpdateTheProperty` | done |
| `row_span::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_RowSpan_ClearerShouldResetTheProperty` | done |
| `column_span::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_ColumnSpan_GetterShouldReturnDefaultValue` | done |
| `column_span::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_ColumnSpan_SetterShouldUpdateTheProperty` | done |
| `column_span::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_ColumnSpan_ClearerShouldResetTheProperty` | done |
| `level::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_Level_GetterShouldReturnDefaultValue` | done |
| `level::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_Level_SetterShouldUpdateTheProperty` | done |
| `level::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_Level_ClearerShouldResetTheProperty` | done |
| `size_of_set::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_SizeOfSet_GetterShouldReturnDefaultValue` | done |
| `size_of_set::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_SizeOfSet_SetterShouldUpdateTheProperty` | done |
| `size_of_set::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_SizeOfSet_ClearerShouldResetTheProperty` | done |
| `position_in_set::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_PositionInSet_GetterShouldReturnDefaultValue` | done |
| `position_in_set::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_PositionInSet_SetterShouldUpdateTheProperty` | done |
| `position_in_set::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_PositionInSet_ClearerShouldResetTheProperty` | done |
| `color_value::getter_should_return_default_value` | test | 1510 | cfg(test) | `TestLib_ColorValue_GetterShouldReturnDefaultValue` | done |
| `color_value::setter_should_update_the_property` | test | 1515 | cfg(test) | `TestLib_ColorValue_SetterShouldUpdateTheProperty` | done |
| `color_value::clearer_should_reset_the_property` | test | 1521 | cfg(test) | `TestLib_ColorValue_ClearerShouldResetTheProperty` | done |
| `background_color::getter_should_return_default_value` | test | 1510 | cfg(test) | `TestLib_BackgroundColor_GetterShouldReturnDefaultValue` | done |
| `background_color::setter_should_update_the_property` | test | 1515 | cfg(test) | `TestLib_BackgroundColor_SetterShouldUpdateTheProperty` | done |
| `background_color::clearer_should_reset_the_property` | test | 1521 | cfg(test) | `TestLib_BackgroundColor_ClearerShouldResetTheProperty` | done |
| `foreground_color::getter_should_return_default_value` | test | 1510 | cfg(test) | `TestLib_ForegroundColor_GetterShouldReturnDefaultValue` | done |
| `foreground_color::setter_should_update_the_property` | test | 1515 | cfg(test) | `TestLib_ForegroundColor_SetterShouldUpdateTheProperty` | done |
| `foreground_color::clearer_should_reset_the_property` | test | 1521 | cfg(test) | `TestLib_ForegroundColor_ClearerShouldResetTheProperty` | done |
| `overline::getter_should_return_default_value` | test | 1555 | cfg(test) | `TestLib_Overline_GetterShouldReturnDefaultValue` | done |
| `overline::setter_should_update_the_property` | test | 1560 | cfg(test) | `TestLib_Overline_SetterShouldUpdateTheProperty` | done |
| `overline::clearer_should_reset_the_property` | test | 1566 | cfg(test) | `TestLib_Overline_ClearerShouldResetTheProperty` | done |
| `strikethrough::getter_should_return_default_value` | test | 1555 | cfg(test) | `TestLib_Strikethrough_GetterShouldReturnDefaultValue` | done |
| `strikethrough::setter_should_update_the_property` | test | 1560 | cfg(test) | `TestLib_Strikethrough_SetterShouldUpdateTheProperty` | done |
| `strikethrough::clearer_should_reset_the_property` | test | 1566 | cfg(test) | `TestLib_Strikethrough_ClearerShouldResetTheProperty` | done |
| `underline::getter_should_return_default_value` | test | 1555 | cfg(test) | `TestLib_Underline_GetterShouldReturnDefaultValue` | done |
| `underline::setter_should_update_the_property` | test | 1560 | cfg(test) | `TestLib_Underline_SetterShouldUpdateTheProperty` | done |
| `underline::clearer_should_reset_the_property` | test | 1566 | cfg(test) | `TestLib_Underline_ClearerShouldResetTheProperty` | done |
| `character_lengths::getter_should_return_default_value` | test | 1590 | cfg(test) | `TestLib_CharacterLengths_GetterShouldReturnDefaultValue` | done |
| `character_lengths::setter_should_update_the_property` | test | 1595 | cfg(test) | `TestLib_CharacterLengths_SetterShouldUpdateTheProperty` | done |
| `character_lengths::clearer_should_reset_the_property` | test | 1603 | cfg(test) | `TestLib_CharacterLengths_ClearerShouldResetTheProperty` | done |
| `word_starts::getter_should_return_default_value` | test | 1590 | cfg(test) | `TestLib_WordStarts_GetterShouldReturnDefaultValue` | done |
| `word_starts::setter_should_update_the_property` | test | 1595 | cfg(test) | `TestLib_WordStarts_SetterShouldUpdateTheProperty` | done |
| `word_starts::clearer_should_reset_the_property` | test | 1603 | cfg(test) | `TestLib_WordStarts_ClearerShouldResetTheProperty` | done |
| `character_positions::getter_should_return_default_value` | test | 1627 | cfg(test) | `TestLib_CharacterPositions_GetterShouldReturnDefaultValue` | done |
| `character_positions::setter_should_update_the_property` | test | 1632 | cfg(test) | `TestLib_CharacterPositions_SetterShouldUpdateTheProperty` | done |
| `character_positions::clearer_should_reset_the_property` | test | 1642 | cfg(test) | `TestLib_CharacterPositions_ClearerShouldResetTheProperty` | done |
| `character_widths::getter_should_return_default_value` | test | 1627 | cfg(test) | `TestLib_CharacterWidths_GetterShouldReturnDefaultValue` | done |
| `character_widths::setter_should_update_the_property` | test | 1632 | cfg(test) | `TestLib_CharacterWidths_SetterShouldUpdateTheProperty` | done |
| `character_widths::clearer_should_reset_the_property` | test | 1642 | cfg(test) | `TestLib_CharacterWidths_ClearerShouldResetTheProperty` | done |
| `is_expanded::getter_should_return_default_value` | test | 1666 | cfg(test) | `TestLib_IsExpanded_GetterShouldReturnDefaultValue` | done |
| `is_expanded::setter_should_update_the_property` | test | 1671 | cfg(test) | `TestLib_IsExpanded_SetterShouldUpdateTheProperty` | done |
| `is_expanded::clearer_should_reset_the_property` | test | 1677 | cfg(test) | `TestLib_IsExpanded_ClearerShouldResetTheProperty` | done |
| `is_selected::getter_should_return_default_value` | test | 1666 | cfg(test) | `TestLib_IsSelected_GetterShouldReturnDefaultValue` | done |
| `is_selected::setter_should_update_the_property` | test | 1671 | cfg(test) | `TestLib_IsSelected_SetterShouldUpdateTheProperty` | done |
| `is_selected::clearer_should_reset_the_property` | test | 1677 | cfg(test) | `TestLib_IsSelected_ClearerShouldResetTheProperty` | done |
| `invalid::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_Invalid_GetterShouldReturnDefaultValue` | done |
| `invalid::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_Invalid_SetterShouldUpdateTheProperty` | done |
| `invalid::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_Invalid_ClearerShouldResetTheProperty` | done |
| `toggled::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_Toggled_GetterShouldReturnDefaultValue` | done |
| `toggled::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_Toggled_SetterShouldUpdateTheProperty` | done |
| `toggled::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_Toggled_ClearerShouldResetTheProperty` | done |
| `live::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_Live_GetterShouldReturnDefaultValue` | done |
| `live::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_Live_SetterShouldUpdateTheProperty` | done |
| `live::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_Live_ClearerShouldResetTheProperty` | done |
| `text_direction::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_TextDirection_GetterShouldReturnDefaultValue` | done |
| `text_direction::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_TextDirection_SetterShouldUpdateTheProperty` | done |
| `text_direction::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_TextDirection_ClearerShouldResetTheProperty` | done |
| `orientation::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_Orientation_GetterShouldReturnDefaultValue` | done |
| `orientation::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_Orientation_SetterShouldUpdateTheProperty` | done |
| `orientation::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_Orientation_ClearerShouldResetTheProperty` | done |
| `sort_direction::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_SortDirection_GetterShouldReturnDefaultValue` | done |
| `sort_direction::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_SortDirection_SetterShouldUpdateTheProperty` | done |
| `sort_direction::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_SortDirection_ClearerShouldResetTheProperty` | done |
| `aria_current::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_AriaCurrent_GetterShouldReturnDefaultValue` | done |
| `aria_current::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_AriaCurrent_SetterShouldUpdateTheProperty` | done |
| `aria_current::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_AriaCurrent_ClearerShouldResetTheProperty` | done |
| `auto_complete::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_AutoComplete_GetterShouldReturnDefaultValue` | done |
| `auto_complete::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_AutoComplete_SetterShouldUpdateTheProperty` | done |
| `auto_complete::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_AutoComplete_ClearerShouldResetTheProperty` | done |
| `has_popup::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_HasPopup_GetterShouldReturnDefaultValue` | done |
| `has_popup::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_HasPopup_SetterShouldUpdateTheProperty` | done |
| `has_popup::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_HasPopup_ClearerShouldResetTheProperty` | done |
| `list_style::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_ListStyle_GetterShouldReturnDefaultValue` | done |
| `list_style::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_ListStyle_SetterShouldUpdateTheProperty` | done |
| `list_style::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_ListStyle_ClearerShouldResetTheProperty` | done |
| `text_align::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_TextAlign_GetterShouldReturnDefaultValue` | done |
| `text_align::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_TextAlign_SetterShouldUpdateTheProperty` | done |
| `text_align::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_TextAlign_ClearerShouldResetTheProperty` | done |
| `vertical_offset::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_VerticalOffset_GetterShouldReturnDefaultValue` | done |
| `vertical_offset::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_VerticalOffset_SetterShouldUpdateTheProperty` | done |
| `vertical_offset::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_VerticalOffset_ClearerShouldResetTheProperty` | done |
| `transform::getter_should_return_default_value` | test | 2200 | cfg(test) | `TestLib_Transform_GetterShouldReturnDefaultValue` | done |
| `transform::setter_should_update_the_property` | test | 2205 | cfg(test) | `TestLib_Transform_SetterShouldUpdateTheProperty` | done |
| `transform::clearer_should_reset_the_property` | test | 2211 | cfg(test) | `TestLib_Transform_ClearerShouldResetTheProperty` | done |
| `bounds::getter_should_return_default_value` | test | 2224 | cfg(test) | `TestLib_Bounds_GetterShouldReturnDefaultValue` | done |
| `bounds::setter_should_update_the_property` | test | 2229 | cfg(test) | `TestLib_Bounds_SetterShouldUpdateTheProperty` | done |
| `bounds::clearer_should_reset_the_property` | test | 2241 | cfg(test) | `TestLib_Bounds_ClearerShouldResetTheProperty` | done |
| `text_selection::getter_should_return_default_value` | test | 2259 | cfg(test) | `TestLib_TextSelection_GetterShouldReturnDefaultValue` | done |
| `text_selection::setter_should_update_the_property` | test | 2264 | cfg(test) | `TestLib_TextSelection_SetterShouldUpdateTheProperty` | done |
| `text_selection::clearer_should_reset_the_property` | test | 2280 | cfg(test) | `TestLib_TextSelection_ClearerShouldResetTheProperty` | done |
| `tree_id::getter_should_return_default_value` | test | 2302 | cfg(test) | `TestLib_TreeID_GetterShouldReturnDefaultValue` | done |
| `tree_id::setter_should_update_the_property` | test | 2307 | cfg(test) | `TestLib_TreeID_SetterShouldUpdateTheProperty` | done |
| `tree_id::clearer_should_reset_the_property` | test | 2314 | cfg(test) | `TestLib_TreeID_ClearerShouldResetTheProperty` | done |
| `custom_actions::getter_should_return_default_value` | test | 2332 | cfg(test) | `TestLib_CustomActions_GetterShouldReturnDefaultValue` | done |
| `custom_actions::setter_should_update_the_property` | test | 2337 | cfg(test) | `TestLib_CustomActions_SetterShouldUpdateTheProperty` | done |
| `custom_actions::pusher_should_update_the_property` | test | 2353 | cfg(test) | `TestLib_CustomActions_PusherShouldUpdateTheProperty` | done |
| `custom_actions::clearer_should_reset_the_property` | test | 2369 | cfg(test) | `TestLib_CustomActions_ClearerShouldResetTheProperty` | done |
| `tests::u64_should_be_convertible_to_node_id` | test | 3014 | cfg(test) | `TestLib_U64ShouldBeConvertibleToNodeID` | done |
| `tests::node_id_should_be_convertible_to_u64` | test | 3020 | cfg(test) | `TestLib_NodeIDShouldBeConvertibleToU64` | done |
| `tests::node_id_should_have_debug_repr` | test | 3026 | cfg(test) | `TestLib_NodeIDShouldHaveDebugRepr` | done |
| `tests::action_n_should_return_the_corresponding_variant` | test | 3032 | cfg(test) | `TestLib_ActionNShouldReturnTheCorrespondingVariant` | done |
| `tests::empty_action_mask_should_be_converted_to_empty_vec` | test | 3062 | cfg(test) | `TestLib_EmptyActionMaskShouldBeConvertedToEmptyVec` | done |
| `tests::action_mask_should_be_convertible_to_vec` | test | 3070 | cfg(test) | `TestLib_ActionMaskShouldBeConvertibleToVec` | done |
| `tests::new_node_should_have_user_provided_role` | test | 3104 | cfg(test) | `TestLib_NewNodeShouldHaveUserProvidedRole` | done |
| `tests::node_role_setter_should_update_the_role` | test | 3110 | cfg(test) | `TestLib_NodeRoleSetterShouldUpdateTheRole` | done |
| `tests::new_node_should_not_support_anyaction` | test | 3124 | cfg(test) | `TestLib_NewNodeShouldNotSupportAnyaction` | done |
| `tests::node_add_action_should_add_the_action` | test | 3151 | cfg(test) | `TestLib_NodeAddActionShouldAddTheAction` | done |
| `tests::node_add_child_action_should_add_the_action` | test | 3160 | cfg(test) | `TestLib_NodeAddChildActionShouldAddTheAction` | done |
| `tests::node_add_action_should_do_nothing_if_the_action_is_already_supported` | test | 3169 | cfg(test) | `TestLib_NodeAddActionShouldDoNothingIfTheActionIsAlreadySupported` | done |
| `tests::node_add_child_action_should_do_nothing_if_the_action_is_already_supported` | test | 3177 | cfg(test) | `TestLib_NodeAddChildActionShouldDoNothingIfTheActionIsAlreadySupported` | done |
| `tests::node_remove_action_should_remove_the_action` | test | 3185 | cfg(test) | `TestLib_NodeRemoveActionShouldRemoveTheAction` | done |
| `tests::node_remove_child_action_should_remove_the_action` | test | 3193 | cfg(test) | `TestLib_NodeRemoveChildActionShouldRemoveTheAction` | done |
| `tests::node_clear_actions_should_remove_all_actions` | test | 3201 | cfg(test) | `TestLib_NodeClearActionsShouldRemoveAllActions` | done |
| `tests::node_clear_child_actions_should_remove_all_actions` | test | 3211 | cfg(test) | `TestLib_NodeClearChildActionsShouldRemoveAllActions` | done |
| `tests::node_should_have_debug_repr` | test | 3221 | cfg(test) | `TestLib_NodeShouldHaveDebugRepr` | done |
| `tests::new_tree_should_have_root_id` | test | 3242 | cfg(test) | `TestLib_NewTreeShouldHaveRootID` | done |

## src/geometry.rs

Go target: `a11y/geometry.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Affine` | struct | 22 |  | `Affine` | done |
| `Affine::IDENTITY` | assoc-const | 26 |  | `AffineIdentity` | done |
| `Affine::FLIP_Y` | assoc-const | 30 |  | `AffineFlipY` | done |
| `Affine::FLIP_X` | assoc-const | 33 |  | `AffineFlipX` | done |
| `Affine::new` | assoc-fn | 53 |  | `NewAffine` | done |
| `Affine::scale` | assoc-fn | 59 |  | `AffineScale` | done |
| `Affine::scale_non_uniform` | assoc-fn | 66 |  | `AffineScaleNonUniform` | done |
| `Affine::translate` | assoc-fn | 72 |  | `AffineTranslate` | done |
| `Affine::map_unit_square` | assoc-fn | 81 |  | `AffineMapUnitSquare` | done |
| `Affine::as_coeffs` | method | 87 |  | `(Affine).AsCoeffs` | done |
| `Affine::determinant` | method | 92 |  | `(Affine).Determinant` | done |
| `Affine::inverse` | method | 99 |  | `(Affine).Inverse` | done |
| `Affine::transform_rect_bbox` | method | 118 |  | `(Affine).TransformRectBbox` | done |
| `Affine::is_finite` | method | 128 |  | `(Affine).IsFinite` | done |
| `Affine::is_nan` | method | 139 |  | `(Affine).IsNan` | done |
| `impl Default for Affine` | trait-impl | 149 |  | `DefaultAffine` | done |
| `impl Mul<Point> for Affine` | trait-impl | 156 |  | `(Affine).MulPoint` | done |
| `impl Mul for Affine` | trait-impl | 168 |  | `(Affine).Mul` | done |
| `impl MulAssign for Affine` | trait-impl | 184 |  | `(Affine).MulAssign` | done |
| `impl Mul<Affine> for f64` | trait-impl | 191 |  | `Float64MulAffine` | done |
| `Point` | struct | 212 |  | `Point` | done |
| `Point.x` | field | 214 |  | `Point.X` | done |
| `Point.y` | field | 216 |  | `Point.Y` | done |
| `Point::ZERO` | assoc-const | 221 |  | `PointZero` | done |
| `Point::ORIGIN` | assoc-const | 224 |  | `PointOrigin` | done |
| `Point::new` | assoc-fn | 228 |  | `NewPoint` | done |
| `Point::to_vec2` | method | 234 |  | `(Point).ToVec2` | done |
| `impl From<(f64, f64)> for Point` | trait-impl | 239 |  | - | dev:D-CONV |
| `impl From<Point> for (f64, f64)` | trait-impl | 246 |  | - | dev:D-CONV |
| `impl Add<Vec2> for Point` | trait-impl | 253 |  | `(Point).Add` | done |
| `impl AddAssign<Vec2> for Point` | trait-impl | 262 |  | `(Point).AddAssign` | done |
| `impl Sub<Vec2> for Point` | trait-impl | 269 |  | `(Point).Sub` | done |
| `impl SubAssign<Vec2> for Point` | trait-impl | 278 |  | `(Point).SubAssign` | done |
| `impl Add<(f64, f64)> for Point` | trait-impl | 285 |  | - | dev:D-CONV |
| `impl AddAssign<(f64, f64)> for Point` | trait-impl | 294 |  | - | dev:D-CONV |
| `impl Sub<(f64, f64)> for Point` | trait-impl | 301 |  | - | dev:D-CONV |
| `impl SubAssign<(f64, f64)> for Point` | trait-impl | 310 |  | - | dev:D-CONV |
| `impl Sub<Point> for Point` | trait-impl | 317 |  | `(Point).SubPoint` | done |
| `impl fmt::Debug for Point` | trait-impl | 326 |  | `(Point).String` | done |
| `Rect` | struct | 337 |  | `Rect` | done |
| `Rect.x0` | field | 339 |  | `Rect.X0` | done |
| `Rect.y0` | field | 341 |  | `Rect.Y0` | done |
| `Rect.x1` | field | 343 |  | `Rect.X1` | done |
| `Rect.y1` | field | 345 |  | `Rect.Y1` | done |
| `impl From<(Point, Point)> for Rect` | trait-impl | 348 |  | - | dev:D-CONV |
| `impl From<(Point, Size)> for Rect` | trait-impl | 354 |  | - | dev:D-CONV |
| `impl Add<Vec2> for Rect` | trait-impl | 360 |  | `(Rect).Add` | done |
| `impl Sub<Vec2> for Rect` | trait-impl | 369 |  | `(Rect).Sub` | done |
| `impl fmt::Debug for Rect` | trait-impl | 378 |  | `(Rect).String` | done |
| `impl fmt::Debug for Rect` (alternate `{:#?}` form) | trait-impl | 379 |  | `(Rect).String` | dev:D-ALT |
| `Rect::ZERO` | assoc-const | 399 |  | `RectZero` | done |
| `Rect::new` | assoc-fn | 403 |  | `NewRect` | done |
| `Rect::from_points` | assoc-fn | 411 |  | `RectFromPoints` | done |
| `Rect::from_origin_size` | assoc-fn | 421 |  | `RectFromOriginSize` | done |
| `Rect::with_origin` | method | 428 |  | `(Rect).WithOrigin` | done |
| `Rect::with_size` | method | 434 |  | `(Rect).WithSize` | done |
| `Rect::width` | method | 442 |  | `(Rect).Width` | done |
| `Rect::height` | method | 450 |  | `(Rect).Height` | done |
| `Rect::min_x` | method | 456 |  | `(Rect).MinX` | done |
| `Rect::max_x` | method | 462 |  | `(Rect).MaxX` | done |
| `Rect::min_y` | method | 468 |  | `(Rect).MinY` | done |
| `Rect::max_y` | method | 474 |  | `(Rect).MaxY` | done |
| `Rect::origin` | method | 483 |  | `(Rect).Origin` | done |
| `Rect::size` | method | 489 |  | `(Rect).Size` | done |
| `Rect::abs` | method | 498 |  | `(Rect).Abs` | done |
| `Rect::area` | method | 505 |  | `(Rect).Area` | done |
| `Rect::is_empty` | method | 513 |  | `(Rect).IsEmpty` | done |
| `Rect::contains` | method | 519 |  | `(Rect).Contains` | done |
| `Rect::union` | method | 527 |  | `(Rect).Union` | done |
| `Rect::union_pt` | method | 543 |  | `(Rect).UnionPt` | done |
| `Rect::intersect` | method | 557 |  | `(Rect).Intersect` | done |
| `Size` | struct | 571 |  | `Size` | done |
| `Size.width` | field | 573 |  | `Size.Width` | done |
| `Size.height` | field | 575 |  | `Size.Height` | done |
| `Size::ZERO` | assoc-const | 580 |  | `SizeZero` | done |
| `Size::new` | assoc-fn | 584 |  | `NewSize` | done |
| `Size::to_vec2` | method | 591 |  | `(Size).ToVec2` | done |
| `impl fmt::Debug for Size` | trait-impl | 596 |  | `(Size).String` | done |
| `impl MulAssign<f64> for Size` | trait-impl | 602 |  | `(Size).MulAssign` | done |
| `impl Mul<Size> for f64` | trait-impl | 612 |  | `Float64MulSize` | done |
| `impl Mul<f64> for Size` | trait-impl | 621 |  | `(Size).Mul` | done |
| `impl DivAssign<f64> for Size` | trait-impl | 633 |  | `(Size).DivAssign` | done |
| `impl Div<f64> for Size` | trait-impl | 643 |  | `(Size).Div` | done |
| `impl Add<Size> for Size` | trait-impl | 655 |  | `(Size).Add` | done |
| `impl AddAssign<Size> for Size` | trait-impl | 666 |  | `(Size).AddAssign` | done |
| `impl Sub<Size> for Size` | trait-impl | 673 |  | `(Size).Sub` | done |
| `impl SubAssign<Size> for Size` | trait-impl | 684 |  | `(Size).SubAssign` | done |
| `impl From<(f64, f64)> for Size` | trait-impl | 691 |  | - | dev:D-CONV |
| `impl From<Size> for (f64, f64)` | trait-impl | 701 |  | - | dev:D-CONV |
| `Vec2` | struct | 717 |  | `Vec2` | done |
| `Vec2.x` | field | 719 |  | `Vec2.X` | done |
| `Vec2.y` | field | 721 |  | `Vec2.Y` | done |
| `Vec2::ZERO` | assoc-const | 726 |  | `Vec2Zero` | done |
| `Vec2::new` | assoc-fn | 730 |  | `NewVec2` | done |
| `Vec2::to_point` | method | 736 |  | `(Vec2).ToPoint` | done |
| `Vec2::to_size` | method | 742 |  | `(Vec2).ToSize` | done |
| `impl From<(f64, f64)> for Vec2` | trait-impl | 747 |  | - | dev:D-CONV |
| `impl From<Vec2> for (f64, f64)` | trait-impl | 754 |  | - | dev:D-CONV |
| `impl Add for Vec2` | trait-impl | 761 |  | `(Vec2).Add` | done |
| `impl AddAssign for Vec2` | trait-impl | 773 |  | `(Vec2).AddAssign` | done |
| `impl Sub for Vec2` | trait-impl | 783 |  | `(Vec2).Sub` | done |
| `impl SubAssign for Vec2` | trait-impl | 795 |  | `(Vec2).SubAssign` | done |
| `impl Mul<f64> for Vec2` | trait-impl | 805 |  | `(Vec2).Mul` | done |
| `impl MulAssign<f64> for Vec2` | trait-impl | 817 |  | `(Vec2).MulAssign` | done |
| `impl Mul<Vec2> for f64` | trait-impl | 827 |  | `Float64MulVec2` | done |
| `impl Div<f64> for Vec2` | trait-impl | 836 |  | `(Vec2).Div` | done |
| `impl DivAssign<f64> for Vec2` | trait-impl | 849 |  | `(Vec2).DivAssign` | done |
| `impl Neg for Vec2` | trait-impl | 856 |  | `(Vec2).Neg` | done |

## Deviations registered here

| Id | Meaning | Owner |
|---|---|---|
| `D-CONV` | Rust `From`/`Into` impls for tuples and between types have no Go equivalent; values are built with `New*` constructors or composite literals. | M0-a11y-geometry |
| `D-ALT` | Rust's alternate `{:#?}` Debug form is not provided; `String` returns the default form only. | M0-a11y-geometry |
