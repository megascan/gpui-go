# Parity: ext-a11y

Scope: accesskit 0.24.1 (`src/lib.rs`, `src/geometry.rs`). Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md (`accesskit (schema only)`).

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`. `lib.rs` rows stay `todo` until M0-a11y-schema; `geometry.rs` rows are `done` in the same change as `a11y/geometry.go`. accesskit 0.24.1 `geometry.rs` has no `#[test]` items, so it has no test rows; its behaviour checks live in `a11y/geometry_test.go`. `Insets` (named in STAGES.md) does not exist in accesskit 0.24.1 and is not ported.

## src/lib.rs: public items

Go target: `a11y/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use uuid::Uuid` | use | 32 |  | - | todo |
| `pub use geometry::{Affine, Point, Rect, Size, Vec2}` | use | 35 |  | - | todo |
| `Role` | enum | 62 |  | `Role` | todo |
| `Action` | enum | 289 |  | `Action` | todo |
| `Orientation` | enum | 417 |  | `Orientation` | todo |
| `TextDirection` | enum | 439 |  | `TextDirection` | todo |
| `Invalid` | enum | 465 |  | `Invalid` | todo |
| `Toggled` | enum | 486 |  | `Toggled` | todo |
| `impl From<bool> for Toggled` | trait-impl | 492 |  | - | dev:D-CONV |
| `SortDirection` | enum | 517 |  | `SortDirection` | todo |
| `AriaCurrent` | enum | 538 |  | `AriaCurrent` | todo |
| `AutoComplete` | enum | 563 |  | `AutoComplete` | todo |
| `Live` | enum | 584 |  | `Live` | todo |
| `HasPopup` | enum | 605 |  | `HasPopup` | todo |
| `ListStyle` | enum | 628 |  | `ListStyle` | todo |
| `TextAlign` | enum | 653 |  | `TextAlign` | todo |
| `VerticalOffset` | enum | 675 |  | `VerticalOffset` | todo |
| `TextDecorationStyle` | enum | 695 |  | `TextDecorationStyle` | todo |
| `NodeIdContent` | type | 703 |  | `NodeIDContent` | todo |
| `NodeId` | struct | 715 |  | `NodeID` | todo |
| `NodeId::from(NodeIdContent)` | trait-impl | 717 |  | - | dev:D-CONV |
| `NodeIdContent::from(NodeId)` | trait-impl | 724 |  | - | dev:D-CONV |
| `impl fmt::Debug for NodeId` | trait-impl | 731 |  | `(NodeID).String` | todo |
| `TreeId` | struct | 745 |  | `TreeID` | todo |
| `TreeId::ROOT` | assoc-const | 749 |  | `TreeIDRoot` | todo |
| `CustomAction` | struct | 761 |  | `CustomAction` | todo |
| `CustomAction.id` | field | 762 |  | `CustomAction.ID` | todo |
| `CustomAction.description` | field | 763 |  | `CustomAction.Description` | todo |
| `TextPosition` | struct | 771 |  | `TextPosition` | todo |
| `TextPosition.node` | field | 773 |  | `TextPosition.Node` | todo |
| `TextPosition.character_index` | field | 776 |  | `TextPosition.CharacterIndex` | todo |
| `TextSelection` | struct | 784 |  | `TextSelection` | todo |
| `TextSelection.anchor` | field | 789 |  | `TextSelection.Anchor` | todo |
| `TextSelection.focus` | field | 793 |  | `TextSelection.Focus` | todo |
| `Color` | struct | 834 |  | `Color` | todo |
| `Color.red` | field | 835 |  | `Color.Red` | todo |
| `Color.green` | field | 836 |  | `Color.Green` | todo |
| `Color.blue` | field | 837 |  | `Color.Blue` | todo |
| `Color.alpha` | field | 838 |  | `Color.Alpha` | todo |
| `TextDecoration` | struct | 847 |  | `TextDecoration` | todo |
| `TextDecoration.style` | field | 848 |  | `TextDecoration.Style` | todo |
| `TextDecoration.color` | field | 849 |  | `TextDecoration.Color` | todo |
| `Node` | struct | 1037 |  | `Node` | todo |
| `Node::new` | assoc-fn | 1737 |  | `NewNode` | todo |
| `Node::role` | method | 1747 |  | `(Node).Role` | todo |
| `Node::set_role` | method | 1751 |  | `(Node).SetRole` | todo |
| `Node::supports_action` | method | 1756 |  | `(Node).SupportsAction` | todo |
| `Node::add_action` | method | 1760 |  | `(Node).AddAction` | todo |
| `Node::remove_action` | method | 1764 |  | `(Node).RemoveAction` | todo |
| `Node::clear_actions` | method | 1768 |  | `(Node).ClearActions` | todo |
| `Node::child_supports_action` | method | 1775 |  | `(Node).ChildSupportsAction` | todo |
| `Node::add_child_action` | method | 1781 |  | `(Node).AddChildAction` | todo |
| `Node::remove_child_action` | method | 1787 |  | `(Node).RemoveChildAction` | todo |
| `Node::clear_child_actions` | method | 1793 |  | `(Node).ClearChildActions` | todo |
| `impl fmt::Debug for Node` | trait-impl | 2380 |  | `(Node).String` | todo |
| `impl Serialize for Properties` | trait-impl | 2448 | serde | - | todo |
| `impl Deserialize for Properties` | trait-impl | 2635 | serde | - | todo |
| `impl JsonSchema for Properties` | trait-impl | 2662 | schemars | - | todo |
| `Tree` | struct | 2793 |  | `Tree` | todo |
| `Tree.root` | field | 2795 |  | `Tree.Root` | todo |
| `Tree.toolkit_name` | field | 2797 |  | `Tree.ToolkitName` | todo |
| `Tree.toolkit_version` | field | 2799 |  | `Tree.ToolkitVersion` | todo |
| `Tree::new` | assoc-fn | 2804 |  | `NewTree` | todo |
| `TreeUpdate` | struct | 2829 |  | `TreeUpdate` | todo |
| `TreeUpdate.nodes` | field | 2850 |  | `TreeUpdate.Nodes` | todo |
| `TreeUpdate.tree` | field | 2856 |  | `TreeUpdate.Tree` | todo |
| `TreeUpdate.tree_id` | field | 2867 |  | `TreeUpdate.TreeID` | todo |
| `TreeUpdate.focus` | field | 2878 |  | `TreeUpdate.Focus` | todo |
| `ScrollUnit` | enum | 2892 |  | `ScrollUnit` | todo |
| `ScrollHint` | enum | 2912 |  | `ScrollHint` | todo |
| `ActionData` | enum | 2926 |  | `ActionData` | todo |
| `ActionRequest` | struct | 2949 |  | `ActionRequest` | todo |
| `ActionRequest.action` | field | 2950 |  | `ActionRequest.Action` | todo |
| `ActionRequest.target_tree` | field | 2951 |  | `ActionRequest.TargetTree` | todo |
| `ActionRequest.target_node` | field | 2952 |  | `ActionRequest.TargetNode` | todo |
| `ActionRequest.data` | field | 2953 |  | `ActionRequest.Data` | todo |
| `ActivationHandler` | trait | 2957 |  | `ActivationHandler` | todo |
| `ActivationHandler::request_initial_tree` | trait-method | 2979 |  | `ActivationHandler.RequestInitialTree` | todo |
| `ActionHandler` | trait | 2983 |  | `ActionHandler` | todo |
| `ActionHandler::do_action` | trait-method | 2993 |  | `ActionHandler.DoAction` | todo |
| `DeactivationHandler` | trait | 2997 |  | `DeactivationHandler` | todo |
| `DeactivationHandler::deactivate_accessibility` | trait-method | 3005 |  | `DeactivationHandler.DeactivateAccessibility` | todo |

## src/lib.rs: Node property methods and tests

Go target: `a11y/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Node::is_hidden` | method | 1801 |  | `(Node).IsHidden` | todo |
| `Node::set_hidden` | method | 1801 |  | `(Node).SetHidden` | todo |
| `Node::clear_hidden` | method | 1801 |  | `(Node).ClearHidden` | todo |
| `Node::is_multiselectable` | method | 1802 |  | `(Node).IsMultiselectable` | todo |
| `Node::set_multiselectable` | method | 1802 |  | `(Node).SetMultiselectable` | todo |
| `Node::clear_multiselectable` | method | 1802 |  | `(Node).ClearMultiselectable` | todo |
| `Node::is_required` | method | 1803 |  | `(Node).IsRequired` | todo |
| `Node::set_required` | method | 1803 |  | `(Node).SetRequired` | todo |
| `Node::clear_required` | method | 1803 |  | `(Node).ClearRequired` | todo |
| `Node::is_visited` | method | 1804 |  | `(Node).IsVisited` | todo |
| `Node::set_visited` | method | 1804 |  | `(Node).SetVisited` | todo |
| `Node::clear_visited` | method | 1804 |  | `(Node).ClearVisited` | todo |
| `Node::is_busy` | method | 1805 |  | `(Node).IsBusy` | todo |
| `Node::set_busy` | method | 1805 |  | `(Node).SetBusy` | todo |
| `Node::clear_busy` | method | 1805 |  | `(Node).ClearBusy` | todo |
| `Node::is_live_atomic` | method | 1806 |  | `(Node).IsLiveAtomic` | todo |
| `Node::set_live_atomic` | method | 1806 |  | `(Node).SetLiveAtomic` | todo |
| `Node::clear_live_atomic` | method | 1806 |  | `(Node).ClearLiveAtomic` | todo |
| `Node::is_modal` | method | 1808 |  | `(Node).IsModal` | todo |
| `Node::set_modal` | method | 1808 |  | `(Node).SetModal` | todo |
| `Node::clear_modal` | method | 1808 |  | `(Node).ClearModal` | todo |
| `Node::is_touch_transparent` | method | 1812 |  | `(Node).IsTouchTransparent` | todo |
| `Node::set_touch_transparent` | method | 1812 |  | `(Node).SetTouchTransparent` | todo |
| `Node::clear_touch_transparent` | method | 1812 |  | `(Node).ClearTouchTransparent` | todo |
| `Node::is_read_only` | method | 1814 |  | `(Node).IsReadOnly` | todo |
| `Node::set_read_only` | method | 1814 |  | `(Node).SetReadOnly` | todo |
| `Node::clear_read_only` | method | 1814 |  | `(Node).ClearReadOnly` | todo |
| `Node::is_disabled` | method | 1816 |  | `(Node).IsDisabled` | todo |
| `Node::set_disabled` | method | 1816 |  | `(Node).SetDisabled` | todo |
| `Node::clear_disabled` | method | 1816 |  | `(Node).ClearDisabled` | todo |
| `Node::is_italic` | method | 1817 |  | `(Node).IsItalic` | todo |
| `Node::set_italic` | method | 1817 |  | `(Node).SetItalic` | todo |
| `Node::clear_italic` | method | 1817 |  | `(Node).ClearItalic` | todo |
| `Node::clips_children` | method | 1820 |  | `(Node).ClipsChildren` | todo |
| `Node::set_clips_children` | method | 1820 |  | `(Node).SetClipsChildren` | todo |
| `Node::clear_clips_children` | method | 1820 |  | `(Node).ClearClipsChildren` | todo |
| `Node::is_line_breaking_object` | method | 1823 |  | `(Node).IsLineBreakingObject` | todo |
| `Node::set_is_line_breaking_object` | method | 1823 |  | `(Node).SetIsLineBreakingObject` | todo |
| `Node::clear_is_line_breaking_object` | method | 1823 |  | `(Node).ClearIsLineBreakingObject` | todo |
| `Node::is_page_breaking_object` | method | 1825 |  | `(Node).IsPageBreakingObject` | todo |
| `Node::set_is_page_breaking_object` | method | 1825 |  | `(Node).SetIsPageBreakingObject` | todo |
| `Node::clear_is_page_breaking_object` | method | 1825 |  | `(Node).ClearIsPageBreakingObject` | todo |
| `Node::is_spelling_error` | method | 1826 |  | `(Node).IsSpellingError` | todo |
| `Node::set_is_spelling_error` | method | 1826 |  | `(Node).SetIsSpellingError` | todo |
| `Node::clear_is_spelling_error` | method | 1826 |  | `(Node).ClearIsSpellingError` | todo |
| `Node::is_grammar_error` | method | 1827 |  | `(Node).IsGrammarError` | todo |
| `Node::set_is_grammar_error` | method | 1827 |  | `(Node).SetIsGrammarError` | todo |
| `Node::clear_is_grammar_error` | method | 1827 |  | `(Node).ClearIsGrammarError` | todo |
| `Node::is_search_match` | method | 1828 |  | `(Node).IsSearchMatch` | todo |
| `Node::set_is_search_match` | method | 1828 |  | `(Node).SetIsSearchMatch` | todo |
| `Node::clear_is_search_match` | method | 1828 |  | `(Node).ClearIsSearchMatch` | todo |
| `Node::is_suggestion` | method | 1829 |  | `(Node).IsSuggestion` | todo |
| `Node::set_is_suggestion` | method | 1829 |  | `(Node).SetIsSuggestion` | todo |
| `Node::clear_is_suggestion` | method | 1829 |  | `(Node).ClearIsSuggestion` | todo |
| `Node::children` | method | 1881 |  | `(Node).Children` | todo |
| `Node::set_children` | method | 1881 |  | `(Node).SetChildren` | todo |
| `Node::push_child` | method | 1881 |  | `(Node).PushChild` | todo |
| `Node::clear_children` | method | 1881 |  | `(Node).ClearChildren` | todo |
| `Node::controls` | method | 1882 |  | `(Node).Controls` | todo |
| `Node::set_controls` | method | 1882 |  | `(Node).SetControls` | todo |
| `Node::push_controlled` | method | 1882 |  | `(Node).PushControlled` | todo |
| `Node::clear_controls` | method | 1882 |  | `(Node).ClearControls` | todo |
| `Node::details` | method | 1883 |  | `(Node).Details` | todo |
| `Node::set_details` | method | 1883 |  | `(Node).SetDetails` | todo |
| `Node::push_detail` | method | 1883 |  | `(Node).PushDetail` | todo |
| `Node::clear_details` | method | 1883 |  | `(Node).ClearDetails` | todo |
| `Node::described_by` | method | 1884 |  | `(Node).DescribedBy` | todo |
| `Node::set_described_by` | method | 1884 |  | `(Node).SetDescribedBy` | todo |
| `Node::push_described_by` | method | 1884 |  | `(Node).PushDescribedBy` | todo |
| `Node::clear_described_by` | method | 1884 |  | `(Node).ClearDescribedBy` | todo |
| `Node::flow_to` | method | 1885 |  | `(Node).FlowTo` | todo |
| `Node::set_flow_to` | method | 1885 |  | `(Node).SetFlowTo` | todo |
| `Node::push_flow_to` | method | 1885 |  | `(Node).PushFlowTo` | todo |
| `Node::clear_flow_to` | method | 1885 |  | `(Node).ClearFlowTo` | todo |
| `Node::labelled_by` | method | 1886 |  | `(Node).LabelledBy` | todo |
| `Node::set_labelled_by` | method | 1886 |  | `(Node).SetLabelledBy` | todo |
| `Node::push_labelled_by` | method | 1886 |  | `(Node).PushLabelledBy` | todo |
| `Node::clear_labelled_by` | method | 1886 |  | `(Node).ClearLabelledBy` | todo |
| `Node::owns` | method | 1892 |  | `(Node).Owns` | todo |
| `Node::set_owns` | method | 1892 |  | `(Node).SetOwns` | todo |
| `Node::push_owned` | method | 1892 |  | `(Node).PushOwned` | todo |
| `Node::clear_owns` | method | 1892 |  | `(Node).ClearOwns` | todo |
| `Node::radio_group` | method | 1895 |  | `(Node).RadioGroup` | todo |
| `Node::set_radio_group` | method | 1895 |  | `(Node).SetRadioGroup` | todo |
| `Node::push_to_radio_group` | method | 1895 |  | `(Node).PushToRadioGroup` | todo |
| `Node::clear_radio_group` | method | 1895 |  | `(Node).ClearRadioGroup` | todo |
| `Node::active_descendant` | method | 1902 |  | `(Node).ActiveDescendant` | todo |
| `Node::set_active_descendant` | method | 1902 |  | `(Node).SetActiveDescendant` | todo |
| `Node::clear_active_descendant` | method | 1902 |  | `(Node).ClearActiveDescendant` | todo |
| `Node::error_message` | method | 1903 |  | `(Node).ErrorMessage` | todo |
| `Node::set_error_message` | method | 1903 |  | `(Node).SetErrorMessage` | todo |
| `Node::clear_error_message` | method | 1903 |  | `(Node).ClearErrorMessage` | todo |
| `Node::in_page_link_target` | method | 1904 |  | `(Node).InPageLinkTarget` | todo |
| `Node::set_in_page_link_target` | method | 1904 |  | `(Node).SetInPageLinkTarget` | todo |
| `Node::clear_in_page_link_target` | method | 1904 |  | `(Node).ClearInPageLinkTarget` | todo |
| `Node::member_of` | method | 1905 |  | `(Node).MemberOf` | todo |
| `Node::set_member_of` | method | 1905 |  | `(Node).SetMemberOf` | todo |
| `Node::clear_member_of` | method | 1905 |  | `(Node).ClearMemberOf` | todo |
| `Node::next_on_line` | method | 1906 |  | `(Node).NextOnLine` | todo |
| `Node::set_next_on_line` | method | 1906 |  | `(Node).SetNextOnLine` | todo |
| `Node::clear_next_on_line` | method | 1906 |  | `(Node).ClearNextOnLine` | todo |
| `Node::previous_on_line` | method | 1907 |  | `(Node).PreviousOnLine` | todo |
| `Node::set_previous_on_line` | method | 1907 |  | `(Node).SetPreviousOnLine` | todo |
| `Node::clear_previous_on_line` | method | 1907 |  | `(Node).ClearPreviousOnLine` | todo |
| `Node::popup_for` | method | 1908 |  | `(Node).PopupFor` | todo |
| `Node::set_popup_for` | method | 1908 |  | `(Node).SetPopupFor` | todo |
| `Node::clear_popup_for` | method | 1908 |  | `(Node).ClearPopupFor` | todo |
| `Node::label` | method | 1916 |  | `(Node).Label` | todo |
| `Node::set_label` | method | 1916 |  | `(Node).SetLabel` | todo |
| `Node::clear_label` | method | 1916 |  | `(Node).ClearLabel` | todo |
| `Node::description` | method | 1917 |  | `(Node).Description` | todo |
| `Node::set_description` | method | 1917 |  | `(Node).SetDescription` | todo |
| `Node::clear_description` | method | 1917 |  | `(Node).ClearDescription` | todo |
| `Node::value` | method | 1918 |  | `(Node).Value` | todo |
| `Node::set_value` | method | 1918 |  | `(Node).SetValue` | todo |
| `Node::clear_value` | method | 1918 |  | `(Node).ClearValue` | todo |
| `Node::access_key` | method | 1926 |  | `(Node).AccessKey` | todo |
| `Node::set_access_key` | method | 1926 |  | `(Node).SetAccessKey` | todo |
| `Node::clear_access_key` | method | 1926 |  | `(Node).ClearAccessKey` | todo |
| `Node::author_id` | method | 1929 |  | `(Node).AuthorID` | todo |
| `Node::set_author_id` | method | 1929 |  | `(Node).SetAuthorID` | todo |
| `Node::clear_author_id` | method | 1929 |  | `(Node).ClearAuthorID` | todo |
| `Node::class_name` | method | 1930 |  | `(Node).ClassName` | todo |
| `Node::set_class_name` | method | 1930 |  | `(Node).SetClassName` | todo |
| `Node::clear_class_name` | method | 1930 |  | `(Node).ClearClassName` | todo |
| `Node::font_family` | method | 1932 |  | `(Node).FontFamily` | todo |
| `Node::set_font_family` | method | 1932 |  | `(Node).SetFontFamily` | todo |
| `Node::clear_font_family` | method | 1932 |  | `(Node).ClearFontFamily` | todo |
| `Node::html_tag` | method | 1933 |  | `(Node).HtmlTag` | todo |
| `Node::set_html_tag` | method | 1933 |  | `(Node).SetHtmlTag` | todo |
| `Node::clear_html_tag` | method | 1933 |  | `(Node).ClearHtmlTag` | todo |
| `Node::inner_html` | method | 1936 |  | `(Node).InnerHtml` | todo |
| `Node::set_inner_html` | method | 1936 |  | `(Node).SetInnerHtml` | todo |
| `Node::clear_inner_html` | method | 1936 |  | `(Node).ClearInnerHtml` | todo |
| `Node::keyboard_shortcut` | method | 1940 |  | `(Node).KeyboardShortcut` | todo |
| `Node::set_keyboard_shortcut` | method | 1940 |  | `(Node).SetKeyboardShortcut` | todo |
| `Node::clear_keyboard_shortcut` | method | 1940 |  | `(Node).ClearKeyboardShortcut` | todo |
| `Node::language` | method | 1943 |  | `(Node).Language` | todo |
| `Node::set_language` | method | 1943 |  | `(Node).SetLanguage` | todo |
| `Node::clear_language` | method | 1943 |  | `(Node).ClearLanguage` | todo |
| `Node::placeholder` | method | 1948 |  | `(Node).Placeholder` | todo |
| `Node::set_placeholder` | method | 1948 |  | `(Node).SetPlaceholder` | todo |
| `Node::clear_placeholder` | method | 1948 |  | `(Node).ClearPlaceholder` | todo |
| `Node::role_description` | method | 1952 |  | `(Node).RoleDescription` | todo |
| `Node::set_role_description` | method | 1952 |  | `(Node).SetRoleDescription` | todo |
| `Node::clear_role_description` | method | 1952 |  | `(Node).ClearRoleDescription` | todo |
| `Node::state_description` | method | 1957 |  | `(Node).StateDescription` | todo |
| `Node::set_state_description` | method | 1957 |  | `(Node).SetStateDescription` | todo |
| `Node::clear_state_description` | method | 1957 |  | `(Node).ClearStateDescription` | todo |
| `Node::tooltip` | method | 1962 |  | `(Node).Tooltip` | todo |
| `Node::set_tooltip` | method | 1962 |  | `(Node).SetTooltip` | todo |
| `Node::clear_tooltip` | method | 1962 |  | `(Node).ClearTooltip` | todo |
| `Node::url` | method | 1963 |  | `(Node).URL` | todo |
| `Node::set_url` | method | 1963 |  | `(Node).SetURL` | todo |
| `Node::clear_url` | method | 1963 |  | `(Node).ClearURL` | todo |
| `Node::row_index_text` | method | 1964 |  | `(Node).RowIndexText` | todo |
| `Node::set_row_index_text` | method | 1964 |  | `(Node).SetRowIndexText` | todo |
| `Node::clear_row_index_text` | method | 1964 |  | `(Node).ClearRowIndexText` | todo |
| `Node::column_index_text` | method | 1965 |  | `(Node).ColumnIndexText` | todo |
| `Node::set_column_index_text` | method | 1965 |  | `(Node).SetColumnIndexText` | todo |
| `Node::clear_column_index_text` | method | 1965 |  | `(Node).ClearColumnIndexText` | todo |
| `Node::braille_label` | method | 1966 |  | `(Node).BrailleLabel` | todo |
| `Node::set_braille_label` | method | 1966 |  | `(Node).SetBrailleLabel` | todo |
| `Node::clear_braille_label` | method | 1966 |  | `(Node).ClearBrailleLabel` | todo |
| `Node::braille_role_description` | method | 1967 |  | `(Node).BrailleRoleDescription` | todo |
| `Node::set_braille_role_description` | method | 1967 |  | `(Node).SetBrailleRoleDescription` | todo |
| `Node::clear_braille_role_description` | method | 1967 |  | `(Node).ClearBrailleRoleDescription` | todo |
| `Node::row_count` | method | 1993 |  | `(Node).RowCount` | todo |
| `Node::set_row_count` | method | 1993 |  | `(Node).SetRowCount` | todo |
| `Node::clear_row_count` | method | 1993 |  | `(Node).ClearRowCount` | todo |
| `Node::column_count` | method | 1994 |  | `(Node).ColumnCount` | todo |
| `Node::set_column_count` | method | 1994 |  | `(Node).SetColumnCount` | todo |
| `Node::clear_column_count` | method | 1994 |  | `(Node).ClearColumnCount` | todo |
| `Node::row_index` | method | 1995 |  | `(Node).RowIndex` | todo |
| `Node::set_row_index` | method | 1995 |  | `(Node).SetRowIndex` | todo |
| `Node::clear_row_index` | method | 1995 |  | `(Node).ClearRowIndex` | todo |
| `Node::column_index` | method | 1996 |  | `(Node).ColumnIndex` | todo |
| `Node::set_column_index` | method | 1996 |  | `(Node).SetColumnIndex` | todo |
| `Node::clear_column_index` | method | 1996 |  | `(Node).ClearColumnIndex` | todo |
| `Node::row_span` | method | 1997 |  | `(Node).RowSpan` | todo |
| `Node::set_row_span` | method | 1997 |  | `(Node).SetRowSpan` | todo |
| `Node::clear_row_span` | method | 1997 |  | `(Node).ClearRowSpan` | todo |
| `Node::column_span` | method | 1998 |  | `(Node).ColumnSpan` | todo |
| `Node::set_column_span` | method | 1998 |  | `(Node).SetColumnSpan` | todo |
| `Node::clear_column_span` | method | 1998 |  | `(Node).ClearColumnSpan` | todo |
| `Node::level` | method | 1999 |  | `(Node).Level` | todo |
| `Node::set_level` | method | 1999 |  | `(Node).SetLevel` | todo |
| `Node::clear_level` | method | 1999 |  | `(Node).ClearLevel` | todo |
| `Node::size_of_set` | method | 2001 |  | `(Node).SizeOfSet` | todo |
| `Node::set_size_of_set` | method | 2001 |  | `(Node).SetSizeOfSet` | todo |
| `Node::clear_size_of_set` | method | 2001 |  | `(Node).ClearSizeOfSet` | todo |
| `Node::position_in_set` | method | 2006 |  | `(Node).PositionInSet` | todo |
| `Node::set_position_in_set` | method | 2006 |  | `(Node).SetPositionInSet` | todo |
| `Node::clear_position_in_set` | method | 2006 |  | `(Node).ClearPositionInSet` | todo |
| `Node::color_value` | method | 2011 |  | `(Node).ColorValue` | todo |
| `Node::set_color_value` | method | 2011 |  | `(Node).SetColorValue` | todo |
| `Node::clear_color_value` | method | 2011 |  | `(Node).ClearColorValue` | todo |
| `Node::background_color` | method | 2013 |  | `(Node).BackgroundColor` | todo |
| `Node::set_background_color` | method | 2013 |  | `(Node).SetBackgroundColor` | todo |
| `Node::clear_background_color` | method | 2013 |  | `(Node).ClearBackgroundColor` | todo |
| `Node::foreground_color` | method | 2015 |  | `(Node).ForegroundColor` | todo |
| `Node::set_foreground_color` | method | 2015 |  | `(Node).SetForegroundColor` | todo |
| `Node::clear_foreground_color` | method | 2015 |  | `(Node).ClearForegroundColor` | todo |
| `Node::overline` | method | 2019 |  | `(Node).Overline` | todo |
| `Node::set_overline` | method | 2019 |  | `(Node).SetOverline` | todo |
| `Node::clear_overline` | method | 2019 |  | `(Node).ClearOverline` | todo |
| `Node::strikethrough` | method | 2020 |  | `(Node).Strikethrough` | todo |
| `Node::set_strikethrough` | method | 2020 |  | `(Node).SetStrikethrough` | todo |
| `Node::clear_strikethrough` | method | 2020 |  | `(Node).ClearStrikethrough` | todo |
| `Node::underline` | method | 2021 |  | `(Node).Underline` | todo |
| `Node::set_underline` | method | 2021 |  | `(Node).SetUnderline` | todo |
| `Node::clear_underline` | method | 2021 |  | `(Node).ClearUnderline` | todo |
| `Node::character_lengths` | method | 2043 |  | `(Node).CharacterLengths` | todo |
| `Node::set_character_lengths` | method | 2043 |  | `(Node).SetCharacterLengths` | todo |
| `Node::clear_character_lengths` | method | 2043 |  | `(Node).ClearCharacterLengths` | todo |
| `Node::word_starts` | method | 2074 |  | `(Node).WordStarts` | todo |
| `Node::set_word_starts` | method | 2074 |  | `(Node).SetWordStarts` | todo |
| `Node::clear_word_starts` | method | 2074 |  | `(Node).ClearWordStarts` | todo |
| `Node::character_positions` | method | 2094 |  | `(Node).CharacterPositions` | todo |
| `Node::set_character_positions` | method | 2094 |  | `(Node).SetCharacterPositions` | todo |
| `Node::clear_character_positions` | method | 2094 |  | `(Node).ClearCharacterPositions` | todo |
| `Node::character_widths` | method | 2114 |  | `(Node).CharacterWidths` | todo |
| `Node::set_character_widths` | method | 2114 |  | `(Node).SetCharacterWidths` | todo |
| `Node::clear_character_widths` | method | 2114 |  | `(Node).ClearCharacterWidths` | todo |
| `Node::is_expanded` | method | 2122 |  | `(Node).IsExpanded` | todo |
| `Node::set_expanded` | method | 2122 |  | `(Node).SetExpanded` | todo |
| `Node::clear_expanded` | method | 2122 |  | `(Node).ClearExpanded` | todo |
| `Node::is_selected` | method | 2133 |  | `(Node).IsSelected` | todo |
| `Node::set_selected` | method | 2133 |  | `(Node).SetSelected` | todo |
| `Node::clear_selected` | method | 2133 |  | `(Node).ClearSelected` | todo |
| `Node::invalid` | method | 2137 |  | `(Node).Invalid` | todo |
| `Node::set_invalid` | method | 2137 |  | `(Node).SetInvalid` | todo |
| `Node::clear_invalid` | method | 2137 |  | `(Node).ClearInvalid` | todo |
| `Node::toggled` | method | 2138 |  | `(Node).Toggled` | todo |
| `Node::set_toggled` | method | 2138 |  | `(Node).SetToggled` | todo |
| `Node::clear_toggled` | method | 2138 |  | `(Node).ClearToggled` | todo |
| `Node::live` | method | 2139 |  | `(Node).Live` | todo |
| `Node::set_live` | method | 2139 |  | `(Node).SetLive` | todo |
| `Node::clear_live` | method | 2139 |  | `(Node).ClearLive` | todo |
| `Node::text_direction` | method | 2140 |  | `(Node).TextDirection` | todo |
| `Node::set_text_direction` | method | 2140 |  | `(Node).SetTextDirection` | todo |
| `Node::clear_text_direction` | method | 2140 |  | `(Node).ClearTextDirection` | todo |
| `Node::orientation` | method | 2141 |  | `(Node).Orientation` | todo |
| `Node::set_orientation` | method | 2141 |  | `(Node).SetOrientation` | todo |
| `Node::clear_orientation` | method | 2141 |  | `(Node).ClearOrientation` | todo |
| `Node::sort_direction` | method | 2142 |  | `(Node).SortDirection` | todo |
| `Node::set_sort_direction` | method | 2142 |  | `(Node).SetSortDirection` | todo |
| `Node::clear_sort_direction` | method | 2142 |  | `(Node).ClearSortDirection` | todo |
| `Node::aria_current` | method | 2143 |  | `(Node).AriaCurrent` | todo |
| `Node::set_aria_current` | method | 2143 |  | `(Node).SetAriaCurrent` | todo |
| `Node::clear_aria_current` | method | 2143 |  | `(Node).ClearAriaCurrent` | todo |
| `Node::auto_complete` | method | 2144 |  | `(Node).AutoComplete` | todo |
| `Node::set_auto_complete` | method | 2144 |  | `(Node).SetAutoComplete` | todo |
| `Node::clear_auto_complete` | method | 2144 |  | `(Node).ClearAutoComplete` | todo |
| `Node::has_popup` | method | 2145 |  | `(Node).HasPopup` | todo |
| `Node::set_has_popup` | method | 2145 |  | `(Node).SetHasPopup` | todo |
| `Node::clear_has_popup` | method | 2145 |  | `(Node).ClearHasPopup` | todo |
| `Node::list_style` | method | 2147 |  | `(Node).ListStyle` | todo |
| `Node::set_list_style` | method | 2147 |  | `(Node).SetListStyle` | todo |
| `Node::clear_list_style` | method | 2147 |  | `(Node).ClearListStyle` | todo |
| `Node::text_align` | method | 2148 |  | `(Node).TextAlign` | todo |
| `Node::set_text_align` | method | 2148 |  | `(Node).SetTextAlign` | todo |
| `Node::clear_text_align` | method | 2148 |  | `(Node).ClearTextAlign` | todo |
| `Node::vertical_offset` | method | 2149 |  | `(Node).VerticalOffset` | todo |
| `Node::set_vertical_offset` | method | 2149 |  | `(Node).SetVerticalOffset` | todo |
| `Node::clear_vertical_offset` | method | 2149 |  | `(Node).ClearVerticalOffset` | todo |
| `Node::transform` | method | 2165 |  | `(Node).Transform` | todo |
| `Node::set_transform` | method | 2165 |  | `(Node).SetTransform` | todo |
| `Node::clear_transform` | method | 2165 |  | `(Node).ClearTransform` | todo |
| `Node::bounds` | method | 2177 |  | `(Node).Bounds` | todo |
| `Node::set_bounds` | method | 2177 |  | `(Node).SetBounds` | todo |
| `Node::clear_bounds` | method | 2177 |  | `(Node).ClearBounds` | todo |
| `Node::text_selection` | method | 2179 |  | `(Node).TextSelection` | todo |
| `Node::set_text_selection` | method | 2179 |  | `(Node).SetTextSelection` | todo |
| `Node::clear_text_selection` | method | 2179 |  | `(Node).ClearTextSelection` | todo |
| `Node::tree_id` | method | 2188 |  | `(Node).TreeID` | todo |
| `Node::set_tree_id` | method | 2188 |  | `(Node).SetTreeID` | todo |
| `Node::clear_tree_id` | method | 2188 |  | `(Node).ClearTreeID` | todo |
| `Node::custom_actions` | method | 2323 |  | `(Node).CustomActions` | todo |
| `Node::set_custom_actions` | method | 2323 |  | `(Node).SetCustomActions` | todo |
| `Node::push_custom_action` | method | 2323 |  | `(Node).PushCustomAction` | todo |
| `Node::clear_custom_actions` | method | 2323 |  | `(Node).ClearCustomActions` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `is_hidden::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsHidden_GetterShouldReturnDefaultValue` | todo |
| `is_hidden::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsHidden_SetterShouldUpdateTheProperty` | todo |
| `is_hidden::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsHidden_ClearerShouldResetTheProperty` | todo |
| `is_multiselectable::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsMultiselectable_GetterShouldReturnDefaultValue` | todo |
| `is_multiselectable::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsMultiselectable_SetterShouldUpdateTheProperty` | todo |
| `is_multiselectable::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsMultiselectable_ClearerShouldResetTheProperty` | todo |
| `is_required::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsRequired_GetterShouldReturnDefaultValue` | todo |
| `is_required::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsRequired_SetterShouldUpdateTheProperty` | todo |
| `is_required::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsRequired_ClearerShouldResetTheProperty` | todo |
| `is_visited::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsVisited_GetterShouldReturnDefaultValue` | todo |
| `is_visited::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsVisited_SetterShouldUpdateTheProperty` | todo |
| `is_visited::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsVisited_ClearerShouldResetTheProperty` | todo |
| `is_busy::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsBusy_GetterShouldReturnDefaultValue` | todo |
| `is_busy::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsBusy_SetterShouldUpdateTheProperty` | todo |
| `is_busy::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsBusy_ClearerShouldResetTheProperty` | todo |
| `is_live_atomic::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsLiveAtomic_GetterShouldReturnDefaultValue` | todo |
| `is_live_atomic::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsLiveAtomic_SetterShouldUpdateTheProperty` | todo |
| `is_live_atomic::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsLiveAtomic_ClearerShouldResetTheProperty` | todo |
| `is_modal::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsModal_GetterShouldReturnDefaultValue` | todo |
| `is_modal::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsModal_SetterShouldUpdateTheProperty` | todo |
| `is_modal::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsModal_ClearerShouldResetTheProperty` | todo |
| `is_touch_transparent::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsTouchTransparent_GetterShouldReturnDefaultValue` | todo |
| `is_touch_transparent::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsTouchTransparent_SetterShouldUpdateTheProperty` | todo |
| `is_touch_transparent::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsTouchTransparent_ClearerShouldResetTheProperty` | todo |
| `is_read_only::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsReadOnly_GetterShouldReturnDefaultValue` | todo |
| `is_read_only::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsReadOnly_SetterShouldUpdateTheProperty` | todo |
| `is_read_only::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsReadOnly_ClearerShouldResetTheProperty` | todo |
| `is_disabled::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsDisabled_GetterShouldReturnDefaultValue` | todo |
| `is_disabled::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsDisabled_SetterShouldUpdateTheProperty` | todo |
| `is_disabled::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsDisabled_ClearerShouldResetTheProperty` | todo |
| `is_italic::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsItalic_GetterShouldReturnDefaultValue` | todo |
| `is_italic::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsItalic_SetterShouldUpdateTheProperty` | todo |
| `is_italic::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsItalic_ClearerShouldResetTheProperty` | todo |
| `clips_children::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_ClipsChildren_GetterShouldReturnDefaultValue` | todo |
| `clips_children::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_ClipsChildren_SetterShouldUpdateTheProperty` | todo |
| `clips_children::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_ClipsChildren_ClearerShouldResetTheProperty` | todo |
| `is_line_breaking_object::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsLineBreakingObject_GetterShouldReturnDefaultValue` | todo |
| `is_line_breaking_object::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsLineBreakingObject_SetterShouldUpdateTheProperty` | todo |
| `is_line_breaking_object::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsLineBreakingObject_ClearerShouldResetTheProperty` | todo |
| `is_page_breaking_object::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsPageBreakingObject_GetterShouldReturnDefaultValue` | todo |
| `is_page_breaking_object::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsPageBreakingObject_SetterShouldUpdateTheProperty` | todo |
| `is_page_breaking_object::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsPageBreakingObject_ClearerShouldResetTheProperty` | todo |
| `is_spelling_error::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsSpellingError_GetterShouldReturnDefaultValue` | todo |
| `is_spelling_error::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsSpellingError_SetterShouldUpdateTheProperty` | todo |
| `is_spelling_error::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsSpellingError_ClearerShouldResetTheProperty` | todo |
| `is_grammar_error::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsGrammarError_GetterShouldReturnDefaultValue` | todo |
| `is_grammar_error::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsGrammarError_SetterShouldUpdateTheProperty` | todo |
| `is_grammar_error::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsGrammarError_ClearerShouldResetTheProperty` | todo |
| `is_search_match::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsSearchMatch_GetterShouldReturnDefaultValue` | todo |
| `is_search_match::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsSearchMatch_SetterShouldUpdateTheProperty` | todo |
| `is_search_match::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsSearchMatch_ClearerShouldResetTheProperty` | todo |
| `is_suggestion::getter_should_return_default_value` | test | 1116 | cfg(test) | `TestLib_IsSuggestion_GetterShouldReturnDefaultValue` | todo |
| `is_suggestion::setter_should_update_the_property` | test | 1122 | cfg(test) | `TestLib_IsSuggestion_SetterShouldUpdateTheProperty` | todo |
| `is_suggestion::clearer_should_reset_the_property` | test | 1129 | cfg(test) | `TestLib_IsSuggestion_ClearerShouldResetTheProperty` | todo |
| `children::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_Children_GetterShouldReturnDefaultValue` | todo |
| `children::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_Children_SetterShouldUpdateTheProperty` | todo |
| `children::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_Children_PusherShouldUpdateTheProperty` | todo |
| `children::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_Children_ClearerShouldResetTheProperty` | todo |
| `controls::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_Controls_GetterShouldReturnDefaultValue` | todo |
| `controls::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_Controls_SetterShouldUpdateTheProperty` | todo |
| `controls::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_Controls_PusherShouldUpdateTheProperty` | todo |
| `controls::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_Controls_ClearerShouldResetTheProperty` | todo |
| `details::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_Details_GetterShouldReturnDefaultValue` | todo |
| `details::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_Details_SetterShouldUpdateTheProperty` | todo |
| `details::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_Details_PusherShouldUpdateTheProperty` | todo |
| `details::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_Details_ClearerShouldResetTheProperty` | todo |
| `described_by::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_DescribedBy_GetterShouldReturnDefaultValue` | todo |
| `described_by::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_DescribedBy_SetterShouldUpdateTheProperty` | todo |
| `described_by::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_DescribedBy_PusherShouldUpdateTheProperty` | todo |
| `described_by::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_DescribedBy_ClearerShouldResetTheProperty` | todo |
| `flow_to::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_FlowTo_GetterShouldReturnDefaultValue` | todo |
| `flow_to::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_FlowTo_SetterShouldUpdateTheProperty` | todo |
| `flow_to::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_FlowTo_PusherShouldUpdateTheProperty` | todo |
| `flow_to::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_FlowTo_ClearerShouldResetTheProperty` | todo |
| `labelled_by::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_LabelledBy_GetterShouldReturnDefaultValue` | todo |
| `labelled_by::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_LabelledBy_SetterShouldUpdateTheProperty` | todo |
| `labelled_by::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_LabelledBy_PusherShouldUpdateTheProperty` | todo |
| `labelled_by::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_LabelledBy_ClearerShouldResetTheProperty` | todo |
| `owns::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_Owns_GetterShouldReturnDefaultValue` | todo |
| `owns::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_Owns_SetterShouldUpdateTheProperty` | todo |
| `owns::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_Owns_PusherShouldUpdateTheProperty` | todo |
| `owns::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_Owns_ClearerShouldResetTheProperty` | todo |
| `radio_group::getter_should_return_default_value` | test | 1278 | cfg(test) | `TestLib_RadioGroup_GetterShouldReturnDefaultValue` | todo |
| `radio_group::setter_should_update_the_property` | test | 1283 | cfg(test) | `TestLib_RadioGroup_SetterShouldUpdateTheProperty` | todo |
| `radio_group::pusher_should_update_the_property` | test | 1291 | cfg(test) | `TestLib_RadioGroup_PusherShouldUpdateTheProperty` | todo |
| `radio_group::clearer_should_reset_the_property` | test | 1299 | cfg(test) | `TestLib_RadioGroup_ClearerShouldResetTheProperty` | todo |
| `active_descendant::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_ActiveDescendant_GetterShouldReturnDefaultValue` | todo |
| `active_descendant::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_ActiveDescendant_SetterShouldUpdateTheProperty` | todo |
| `active_descendant::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_ActiveDescendant_ClearerShouldResetTheProperty` | todo |
| `error_message::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_ErrorMessage_GetterShouldReturnDefaultValue` | todo |
| `error_message::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_ErrorMessage_SetterShouldUpdateTheProperty` | todo |
| `error_message::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_ErrorMessage_ClearerShouldResetTheProperty` | todo |
| `in_page_link_target::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_InPageLinkTarget_GetterShouldReturnDefaultValue` | todo |
| `in_page_link_target::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_InPageLinkTarget_SetterShouldUpdateTheProperty` | todo |
| `in_page_link_target::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_InPageLinkTarget_ClearerShouldResetTheProperty` | todo |
| `member_of::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_MemberOf_GetterShouldReturnDefaultValue` | todo |
| `member_of::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_MemberOf_SetterShouldUpdateTheProperty` | todo |
| `member_of::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_MemberOf_ClearerShouldResetTheProperty` | todo |
| `next_on_line::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_NextOnLine_GetterShouldReturnDefaultValue` | todo |
| `next_on_line::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_NextOnLine_SetterShouldUpdateTheProperty` | todo |
| `next_on_line::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_NextOnLine_ClearerShouldResetTheProperty` | todo |
| `previous_on_line::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_PreviousOnLine_GetterShouldReturnDefaultValue` | todo |
| `previous_on_line::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_PreviousOnLine_SetterShouldUpdateTheProperty` | todo |
| `previous_on_line::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_PreviousOnLine_ClearerShouldResetTheProperty` | todo |
| `popup_for::getter_should_return_default_value` | test | 1335 | cfg(test) | `TestLib_PopupFor_GetterShouldReturnDefaultValue` | todo |
| `popup_for::setter_should_update_the_property` | test | 1340 | cfg(test) | `TestLib_PopupFor_SetterShouldUpdateTheProperty` | todo |
| `popup_for::clearer_should_reset_the_property` | test | 1346 | cfg(test) | `TestLib_PopupFor_ClearerShouldResetTheProperty` | todo |
| `label::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Label_GetterShouldReturnDefaultValue` | todo |
| `label::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Label_SetterShouldUpdateTheProperty` | todo |
| `label::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Label_ClearerShouldResetTheProperty` | todo |
| `description::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Description_GetterShouldReturnDefaultValue` | todo |
| `description::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Description_SetterShouldUpdateTheProperty` | todo |
| `description::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Description_ClearerShouldResetTheProperty` | todo |
| `value::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Value_GetterShouldReturnDefaultValue` | todo |
| `value::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Value_SetterShouldUpdateTheProperty` | todo |
| `value::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Value_ClearerShouldResetTheProperty` | todo |
| `access_key::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_AccessKey_GetterShouldReturnDefaultValue` | todo |
| `access_key::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_AccessKey_SetterShouldUpdateTheProperty` | todo |
| `access_key::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_AccessKey_ClearerShouldResetTheProperty` | todo |
| `author_id::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_AuthorID_GetterShouldReturnDefaultValue` | todo |
| `author_id::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_AuthorID_SetterShouldUpdateTheProperty` | todo |
| `author_id::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_AuthorID_ClearerShouldResetTheProperty` | todo |
| `class_name::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_ClassName_GetterShouldReturnDefaultValue` | todo |
| `class_name::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_ClassName_SetterShouldUpdateTheProperty` | todo |
| `class_name::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_ClassName_ClearerShouldResetTheProperty` | todo |
| `font_family::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_FontFamily_GetterShouldReturnDefaultValue` | todo |
| `font_family::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_FontFamily_SetterShouldUpdateTheProperty` | todo |
| `font_family::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_FontFamily_ClearerShouldResetTheProperty` | todo |
| `html_tag::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_HtmlTag_GetterShouldReturnDefaultValue` | todo |
| `html_tag::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_HtmlTag_SetterShouldUpdateTheProperty` | todo |
| `html_tag::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_HtmlTag_ClearerShouldResetTheProperty` | todo |
| `inner_html::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_InnerHtml_GetterShouldReturnDefaultValue` | todo |
| `inner_html::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_InnerHtml_SetterShouldUpdateTheProperty` | todo |
| `inner_html::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_InnerHtml_ClearerShouldResetTheProperty` | todo |
| `keyboard_shortcut::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_KeyboardShortcut_GetterShouldReturnDefaultValue` | todo |
| `keyboard_shortcut::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_KeyboardShortcut_SetterShouldUpdateTheProperty` | todo |
| `keyboard_shortcut::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_KeyboardShortcut_ClearerShouldResetTheProperty` | todo |
| `language::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Language_GetterShouldReturnDefaultValue` | todo |
| `language::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Language_SetterShouldUpdateTheProperty` | todo |
| `language::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Language_ClearerShouldResetTheProperty` | todo |
| `placeholder::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Placeholder_GetterShouldReturnDefaultValue` | todo |
| `placeholder::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Placeholder_SetterShouldUpdateTheProperty` | todo |
| `placeholder::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Placeholder_ClearerShouldResetTheProperty` | todo |
| `role_description::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_RoleDescription_GetterShouldReturnDefaultValue` | todo |
| `role_description::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_RoleDescription_SetterShouldUpdateTheProperty` | todo |
| `role_description::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_RoleDescription_ClearerShouldResetTheProperty` | todo |
| `state_description::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_StateDescription_GetterShouldReturnDefaultValue` | todo |
| `state_description::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_StateDescription_SetterShouldUpdateTheProperty` | todo |
| `state_description::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_StateDescription_ClearerShouldResetTheProperty` | todo |
| `tooltip::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_Tooltip_GetterShouldReturnDefaultValue` | todo |
| `tooltip::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_Tooltip_SetterShouldUpdateTheProperty` | todo |
| `tooltip::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_Tooltip_ClearerShouldResetTheProperty` | todo |
| `url::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_URL_GetterShouldReturnDefaultValue` | todo |
| `url::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_URL_SetterShouldUpdateTheProperty` | todo |
| `url::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_URL_ClearerShouldResetTheProperty` | todo |
| `row_index_text::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_RowIndexText_GetterShouldReturnDefaultValue` | todo |
| `row_index_text::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_RowIndexText_SetterShouldUpdateTheProperty` | todo |
| `row_index_text::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_RowIndexText_ClearerShouldResetTheProperty` | todo |
| `column_index_text::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_ColumnIndexText_GetterShouldReturnDefaultValue` | todo |
| `column_index_text::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_ColumnIndexText_SetterShouldUpdateTheProperty` | todo |
| `column_index_text::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_ColumnIndexText_ClearerShouldResetTheProperty` | todo |
| `braille_label::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_BrailleLabel_GetterShouldReturnDefaultValue` | todo |
| `braille_label::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_BrailleLabel_SetterShouldUpdateTheProperty` | todo |
| `braille_label::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_BrailleLabel_ClearerShouldResetTheProperty` | todo |
| `braille_role_description::getter_should_return_default_value` | test | 1370 | cfg(test) | `TestLib_BrailleRoleDescription_GetterShouldReturnDefaultValue` | todo |
| `braille_role_description::setter_should_update_the_property` | test | 1375 | cfg(test) | `TestLib_BrailleRoleDescription_SetterShouldUpdateTheProperty` | todo |
| `braille_role_description::clearer_should_reset_the_property` | test | 1381 | cfg(test) | `TestLib_BrailleRoleDescription_ClearerShouldResetTheProperty` | todo |
| `row_count::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_RowCount_GetterShouldReturnDefaultValue` | todo |
| `row_count::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_RowCount_SetterShouldUpdateTheProperty` | todo |
| `row_count::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_RowCount_ClearerShouldResetTheProperty` | todo |
| `column_count::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_ColumnCount_GetterShouldReturnDefaultValue` | todo |
| `column_count::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_ColumnCount_SetterShouldUpdateTheProperty` | todo |
| `column_count::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_ColumnCount_ClearerShouldResetTheProperty` | todo |
| `row_index::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_RowIndex_GetterShouldReturnDefaultValue` | todo |
| `row_index::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_RowIndex_SetterShouldUpdateTheProperty` | todo |
| `row_index::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_RowIndex_ClearerShouldResetTheProperty` | todo |
| `column_index::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_ColumnIndex_GetterShouldReturnDefaultValue` | todo |
| `column_index::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_ColumnIndex_SetterShouldUpdateTheProperty` | todo |
| `column_index::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_ColumnIndex_ClearerShouldResetTheProperty` | todo |
| `row_span::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_RowSpan_GetterShouldReturnDefaultValue` | todo |
| `row_span::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_RowSpan_SetterShouldUpdateTheProperty` | todo |
| `row_span::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_RowSpan_ClearerShouldResetTheProperty` | todo |
| `column_span::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_ColumnSpan_GetterShouldReturnDefaultValue` | todo |
| `column_span::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_ColumnSpan_SetterShouldUpdateTheProperty` | todo |
| `column_span::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_ColumnSpan_ClearerShouldResetTheProperty` | todo |
| `level::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_Level_GetterShouldReturnDefaultValue` | todo |
| `level::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_Level_SetterShouldUpdateTheProperty` | todo |
| `level::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_Level_ClearerShouldResetTheProperty` | todo |
| `size_of_set::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_SizeOfSet_GetterShouldReturnDefaultValue` | todo |
| `size_of_set::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_SizeOfSet_SetterShouldUpdateTheProperty` | todo |
| `size_of_set::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_SizeOfSet_ClearerShouldResetTheProperty` | todo |
| `position_in_set::getter_should_return_default_value` | test | 1475 | cfg(test) | `TestLib_PositionInSet_GetterShouldReturnDefaultValue` | todo |
| `position_in_set::setter_should_update_the_property` | test | 1480 | cfg(test) | `TestLib_PositionInSet_SetterShouldUpdateTheProperty` | todo |
| `position_in_set::clearer_should_reset_the_property` | test | 1486 | cfg(test) | `TestLib_PositionInSet_ClearerShouldResetTheProperty` | todo |
| `color_value::getter_should_return_default_value` | test | 1510 | cfg(test) | `TestLib_ColorValue_GetterShouldReturnDefaultValue` | todo |
| `color_value::setter_should_update_the_property` | test | 1515 | cfg(test) | `TestLib_ColorValue_SetterShouldUpdateTheProperty` | todo |
| `color_value::clearer_should_reset_the_property` | test | 1521 | cfg(test) | `TestLib_ColorValue_ClearerShouldResetTheProperty` | todo |
| `background_color::getter_should_return_default_value` | test | 1510 | cfg(test) | `TestLib_BackgroundColor_GetterShouldReturnDefaultValue` | todo |
| `background_color::setter_should_update_the_property` | test | 1515 | cfg(test) | `TestLib_BackgroundColor_SetterShouldUpdateTheProperty` | todo |
| `background_color::clearer_should_reset_the_property` | test | 1521 | cfg(test) | `TestLib_BackgroundColor_ClearerShouldResetTheProperty` | todo |
| `foreground_color::getter_should_return_default_value` | test | 1510 | cfg(test) | `TestLib_ForegroundColor_GetterShouldReturnDefaultValue` | todo |
| `foreground_color::setter_should_update_the_property` | test | 1515 | cfg(test) | `TestLib_ForegroundColor_SetterShouldUpdateTheProperty` | todo |
| `foreground_color::clearer_should_reset_the_property` | test | 1521 | cfg(test) | `TestLib_ForegroundColor_ClearerShouldResetTheProperty` | todo |
| `overline::getter_should_return_default_value` | test | 1555 | cfg(test) | `TestLib_Overline_GetterShouldReturnDefaultValue` | todo |
| `overline::setter_should_update_the_property` | test | 1560 | cfg(test) | `TestLib_Overline_SetterShouldUpdateTheProperty` | todo |
| `overline::clearer_should_reset_the_property` | test | 1566 | cfg(test) | `TestLib_Overline_ClearerShouldResetTheProperty` | todo |
| `strikethrough::getter_should_return_default_value` | test | 1555 | cfg(test) | `TestLib_Strikethrough_GetterShouldReturnDefaultValue` | todo |
| `strikethrough::setter_should_update_the_property` | test | 1560 | cfg(test) | `TestLib_Strikethrough_SetterShouldUpdateTheProperty` | todo |
| `strikethrough::clearer_should_reset_the_property` | test | 1566 | cfg(test) | `TestLib_Strikethrough_ClearerShouldResetTheProperty` | todo |
| `underline::getter_should_return_default_value` | test | 1555 | cfg(test) | `TestLib_Underline_GetterShouldReturnDefaultValue` | todo |
| `underline::setter_should_update_the_property` | test | 1560 | cfg(test) | `TestLib_Underline_SetterShouldUpdateTheProperty` | todo |
| `underline::clearer_should_reset_the_property` | test | 1566 | cfg(test) | `TestLib_Underline_ClearerShouldResetTheProperty` | todo |
| `character_lengths::getter_should_return_default_value` | test | 1590 | cfg(test) | `TestLib_CharacterLengths_GetterShouldReturnDefaultValue` | todo |
| `character_lengths::setter_should_update_the_property` | test | 1595 | cfg(test) | `TestLib_CharacterLengths_SetterShouldUpdateTheProperty` | todo |
| `character_lengths::clearer_should_reset_the_property` | test | 1603 | cfg(test) | `TestLib_CharacterLengths_ClearerShouldResetTheProperty` | todo |
| `word_starts::getter_should_return_default_value` | test | 1590 | cfg(test) | `TestLib_WordStarts_GetterShouldReturnDefaultValue` | todo |
| `word_starts::setter_should_update_the_property` | test | 1595 | cfg(test) | `TestLib_WordStarts_SetterShouldUpdateTheProperty` | todo |
| `word_starts::clearer_should_reset_the_property` | test | 1603 | cfg(test) | `TestLib_WordStarts_ClearerShouldResetTheProperty` | todo |
| `character_positions::getter_should_return_default_value` | test | 1627 | cfg(test) | `TestLib_CharacterPositions_GetterShouldReturnDefaultValue` | todo |
| `character_positions::setter_should_update_the_property` | test | 1632 | cfg(test) | `TestLib_CharacterPositions_SetterShouldUpdateTheProperty` | todo |
| `character_positions::clearer_should_reset_the_property` | test | 1642 | cfg(test) | `TestLib_CharacterPositions_ClearerShouldResetTheProperty` | todo |
| `character_widths::getter_should_return_default_value` | test | 1627 | cfg(test) | `TestLib_CharacterWidths_GetterShouldReturnDefaultValue` | todo |
| `character_widths::setter_should_update_the_property` | test | 1632 | cfg(test) | `TestLib_CharacterWidths_SetterShouldUpdateTheProperty` | todo |
| `character_widths::clearer_should_reset_the_property` | test | 1642 | cfg(test) | `TestLib_CharacterWidths_ClearerShouldResetTheProperty` | todo |
| `is_expanded::getter_should_return_default_value` | test | 1666 | cfg(test) | `TestLib_IsExpanded_GetterShouldReturnDefaultValue` | todo |
| `is_expanded::setter_should_update_the_property` | test | 1671 | cfg(test) | `TestLib_IsExpanded_SetterShouldUpdateTheProperty` | todo |
| `is_expanded::clearer_should_reset_the_property` | test | 1677 | cfg(test) | `TestLib_IsExpanded_ClearerShouldResetTheProperty` | todo |
| `is_selected::getter_should_return_default_value` | test | 1666 | cfg(test) | `TestLib_IsSelected_GetterShouldReturnDefaultValue` | todo |
| `is_selected::setter_should_update_the_property` | test | 1671 | cfg(test) | `TestLib_IsSelected_SetterShouldUpdateTheProperty` | todo |
| `is_selected::clearer_should_reset_the_property` | test | 1677 | cfg(test) | `TestLib_IsSelected_ClearerShouldResetTheProperty` | todo |
| `invalid::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_Invalid_GetterShouldReturnDefaultValue` | todo |
| `invalid::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_Invalid_SetterShouldUpdateTheProperty` | todo |
| `invalid::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_Invalid_ClearerShouldResetTheProperty` | todo |
| `toggled::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_Toggled_GetterShouldReturnDefaultValue` | todo |
| `toggled::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_Toggled_SetterShouldUpdateTheProperty` | todo |
| `toggled::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_Toggled_ClearerShouldResetTheProperty` | todo |
| `live::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_Live_GetterShouldReturnDefaultValue` | todo |
| `live::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_Live_SetterShouldUpdateTheProperty` | todo |
| `live::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_Live_ClearerShouldResetTheProperty` | todo |
| `text_direction::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_TextDirection_GetterShouldReturnDefaultValue` | todo |
| `text_direction::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_TextDirection_SetterShouldUpdateTheProperty` | todo |
| `text_direction::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_TextDirection_ClearerShouldResetTheProperty` | todo |
| `orientation::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_Orientation_GetterShouldReturnDefaultValue` | todo |
| `orientation::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_Orientation_SetterShouldUpdateTheProperty` | todo |
| `orientation::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_Orientation_ClearerShouldResetTheProperty` | todo |
| `sort_direction::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_SortDirection_GetterShouldReturnDefaultValue` | todo |
| `sort_direction::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_SortDirection_SetterShouldUpdateTheProperty` | todo |
| `sort_direction::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_SortDirection_ClearerShouldResetTheProperty` | todo |
| `aria_current::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_AriaCurrent_GetterShouldReturnDefaultValue` | todo |
| `aria_current::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_AriaCurrent_SetterShouldUpdateTheProperty` | todo |
| `aria_current::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_AriaCurrent_ClearerShouldResetTheProperty` | todo |
| `auto_complete::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_AutoComplete_GetterShouldReturnDefaultValue` | todo |
| `auto_complete::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_AutoComplete_SetterShouldUpdateTheProperty` | todo |
| `auto_complete::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_AutoComplete_ClearerShouldResetTheProperty` | todo |
| `has_popup::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_HasPopup_GetterShouldReturnDefaultValue` | todo |
| `has_popup::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_HasPopup_SetterShouldUpdateTheProperty` | todo |
| `has_popup::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_HasPopup_ClearerShouldResetTheProperty` | todo |
| `list_style::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_ListStyle_GetterShouldReturnDefaultValue` | todo |
| `list_style::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_ListStyle_SetterShouldUpdateTheProperty` | todo |
| `list_style::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_ListStyle_ClearerShouldResetTheProperty` | todo |
| `text_align::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_TextAlign_GetterShouldReturnDefaultValue` | todo |
| `text_align::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_TextAlign_SetterShouldUpdateTheProperty` | todo |
| `text_align::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_TextAlign_ClearerShouldResetTheProperty` | todo |
| `vertical_offset::getter_should_return_default_value` | test | 1713 | cfg(test) | `TestLib_VerticalOffset_GetterShouldReturnDefaultValue` | todo |
| `vertical_offset::setter_should_update_the_property` | test | 1718 | cfg(test) | `TestLib_VerticalOffset_SetterShouldUpdateTheProperty` | todo |
| `vertical_offset::clearer_should_reset_the_property` | test | 1725 | cfg(test) | `TestLib_VerticalOffset_ClearerShouldResetTheProperty` | todo |
| `transform::getter_should_return_default_value` | test | 2200 | cfg(test) | `TestLib_Transform_GetterShouldReturnDefaultValue` | todo |
| `transform::setter_should_update_the_property` | test | 2205 | cfg(test) | `TestLib_Transform_SetterShouldUpdateTheProperty` | todo |
| `transform::clearer_should_reset_the_property` | test | 2211 | cfg(test) | `TestLib_Transform_ClearerShouldResetTheProperty` | todo |
| `bounds::getter_should_return_default_value` | test | 2224 | cfg(test) | `TestLib_Bounds_GetterShouldReturnDefaultValue` | todo |
| `bounds::setter_should_update_the_property` | test | 2229 | cfg(test) | `TestLib_Bounds_SetterShouldUpdateTheProperty` | todo |
| `bounds::clearer_should_reset_the_property` | test | 2241 | cfg(test) | `TestLib_Bounds_ClearerShouldResetTheProperty` | todo |
| `text_selection::getter_should_return_default_value` | test | 2259 | cfg(test) | `TestLib_TextSelection_GetterShouldReturnDefaultValue` | todo |
| `text_selection::setter_should_update_the_property` | test | 2264 | cfg(test) | `TestLib_TextSelection_SetterShouldUpdateTheProperty` | todo |
| `text_selection::clearer_should_reset_the_property` | test | 2280 | cfg(test) | `TestLib_TextSelection_ClearerShouldResetTheProperty` | todo |
| `tree_id::getter_should_return_default_value` | test | 2302 | cfg(test) | `TestLib_TreeID_GetterShouldReturnDefaultValue` | todo |
| `tree_id::setter_should_update_the_property` | test | 2307 | cfg(test) | `TestLib_TreeID_SetterShouldUpdateTheProperty` | todo |
| `tree_id::clearer_should_reset_the_property` | test | 2314 | cfg(test) | `TestLib_TreeID_ClearerShouldResetTheProperty` | todo |
| `custom_actions::getter_should_return_default_value` | test | 2332 | cfg(test) | `TestLib_CustomActions_GetterShouldReturnDefaultValue` | todo |
| `custom_actions::setter_should_update_the_property` | test | 2337 | cfg(test) | `TestLib_CustomActions_SetterShouldUpdateTheProperty` | todo |
| `custom_actions::pusher_should_update_the_property` | test | 2353 | cfg(test) | `TestLib_CustomActions_PusherShouldUpdateTheProperty` | todo |
| `custom_actions::clearer_should_reset_the_property` | test | 2369 | cfg(test) | `TestLib_CustomActions_ClearerShouldResetTheProperty` | todo |
| `tests::u64_should_be_convertible_to_node_id` | test | 3014 | cfg(test) | `TestLib_U64ShouldBeConvertibleToNodeID` | todo |
| `tests::node_id_should_be_convertible_to_u64` | test | 3020 | cfg(test) | `TestLib_NodeIDShouldBeConvertibleToU64` | todo |
| `tests::node_id_should_have_debug_repr` | test | 3026 | cfg(test) | `TestLib_NodeIDShouldHaveDebugRepr` | todo |
| `tests::action_n_should_return_the_corresponding_variant` | test | 3032 | cfg(test) | `TestLib_ActionNShouldReturnTheCorrespondingVariant` | todo |
| `tests::empty_action_mask_should_be_converted_to_empty_vec` | test | 3062 | cfg(test) | `TestLib_EmptyActionMaskShouldBeConvertedToEmptyVec` | todo |
| `tests::action_mask_should_be_convertible_to_vec` | test | 3070 | cfg(test) | `TestLib_ActionMaskShouldBeConvertibleToVec` | todo |
| `tests::new_node_should_have_user_provided_role` | test | 3104 | cfg(test) | `TestLib_NewNodeShouldHaveUserProvidedRole` | todo |
| `tests::node_role_setter_should_update_the_role` | test | 3110 | cfg(test) | `TestLib_NodeRoleSetterShouldUpdateTheRole` | todo |
| `tests::new_node_should_not_support_anyaction` | test | 3124 | cfg(test) | `TestLib_NewNodeShouldNotSupportAnyaction` | todo |
| `tests::node_add_action_should_add_the_action` | test | 3151 | cfg(test) | `TestLib_NodeAddActionShouldAddTheAction` | todo |
| `tests::node_add_child_action_should_add_the_action` | test | 3160 | cfg(test) | `TestLib_NodeAddChildActionShouldAddTheAction` | todo |
| `tests::node_add_action_should_do_nothing_if_the_action_is_already_supported` | test | 3169 | cfg(test) | `TestLib_NodeAddActionShouldDoNothingIfTheActionIsAlreadySupported` | todo |
| `tests::node_add_child_action_should_do_nothing_if_the_action_is_already_supported` | test | 3177 | cfg(test) | `TestLib_NodeAddChildActionShouldDoNothingIfTheActionIsAlreadySupported` | todo |
| `tests::node_remove_action_should_remove_the_action` | test | 3185 | cfg(test) | `TestLib_NodeRemoveActionShouldRemoveTheAction` | todo |
| `tests::node_remove_child_action_should_remove_the_action` | test | 3193 | cfg(test) | `TestLib_NodeRemoveChildActionShouldRemoveTheAction` | todo |
| `tests::node_clear_actions_should_remove_all_actions` | test | 3201 | cfg(test) | `TestLib_NodeClearActionsShouldRemoveAllActions` | todo |
| `tests::node_clear_child_actions_should_remove_all_actions` | test | 3211 | cfg(test) | `TestLib_NodeClearChildActionsShouldRemoveAllActions` | todo |
| `tests::node_should_have_debug_repr` | test | 3221 | cfg(test) | `TestLib_NodeShouldHaveDebugRepr` | todo |
| `tests::new_tree_should_have_root_id` | test | 3242 | cfg(test) | `TestLib_NewTreeShouldHaveRootID` | todo |

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
