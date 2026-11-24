# Parity: ext-a11y-macos

Scope: accesskit_macos 0.26.3 (`src/*.rs`). Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_macos-0.26.3/`.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`, `blocked:<stage>`. M4-macos-a11y ports every file into package `a11y/macos` (one Go file per upstream file; the libobjc / AppKit functions and constant lookups the crate gets from objc2 live in `appkit.go`) and wires `SubclassingAdapter::for_window` into `platform/macos/a11y.go`, where `_ref/gpui_macos/src/window.rs` uses it.

Notes on the port:

- `declare_class!` classes are registered with `objc.RegisterClass` (purego); IMPs are Go functions taking `(objc.ID, objc.SEL, ...)`, with `NSRect` / `NSRange` / `NSPoint` passed as `objc.CGRect` / `objc.NSRange` / `objc.CGPoint`.
- `#[method_id]` IMPs return +0 (autoreleased) objects; AppKit string constants are returned as they are, `ns_string!` literals are cached +1 NSStrings that live for the process.
- `Rc<Context>` is `*Context`; its `Drop` (posting `NSAccessibilityUIElementDestroyedNotification` for every platform node) is `Context.Release`, called where upstream drops the last `Rc`: when the placeholder state is replaced and when the adapter is dropped (`D-MACA11YWEAK`).
- `WeakId<NSView>` is `weakID` over `objc_initWeak` / `objc_loadWeakRetained` / `objc_destroyWeak`.
- `RefCell<AssociatedObjectState>::borrow_mut` keeps upstream's panic on re-entrant borrow.
- `Option<T>` returns are `(T, bool)`; `Option<Value>` is a `Value` whose zero kind is `None`.
- accesskit_macos has no unit tests; `node_test.go` adds coverage that runs on the macOS CI runner (locally it is cross-compiled only).

## src/lib.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `mod context` | mod | 8 |  | `Context` | done |
| `mod filters` | mod | 9 |  | `filter` | done |
| `mod node` | mod | 10 |  | `newPlatformNode` | done |
| `mod util` | mod | 11 |  | `toNSRect` | done |
| `pub use adapter::Adapter` | use | 14 |  | `Adapter` | done |
| `pub use event::QueuedEvents` | use | 17 |  | `QueuedEvents` | done |
| `pub use patch::add_focus_forwarder_to_window_class` | use | 20 |  | `AddFocusForwarderToWindowClass` | done |
| `pub use subclass::SubclassingAdapter` | use | 23 |  | `SubclassingAdapter` | done |
| `pub use objc2_foundation::{NSArray, NSObject, NSPoint}` | use | 25 |  | `objc.ID`, `objc.CGPoint` | n/a:D-MACA11YOBJC |

## src/filters.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `common_filter as filter` | use | 6 |  | `filter` | done |

## src/util.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `from_ns_range` | fn | 13 |  | `fromNSRange` | done |
| `to_ns_range` | fn | 24 |  | `toNSRange` | done |
| `to_ns_range_for_character` | fn | 30 |  | `toNSRangeForCharacter` | done |
| `from_ns_point` | fn | 38 |  | `fromNSPoint` | done |
| `to_ns_rect` | fn | 57 |  | `toNSRect` | done |
| `color_channel_to_f64` | fn | 83 |  | `colorChannelToF64` | done |
| `CGColor` | struct | 89 |  | `objc.ID` | n/a:D-MACA11YOBJC |
| `unsafe impl RefEncode for CGColor` | impl | 93 |  | `objc.ID` | n/a:D-MACA11YOBJC |
| `to_color_attribute` | fn | 97 |  | `toColorAttribute` | done |

## src/context.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ActionHandlerNoMut` | trait | 16 |  | `actionHandlerNoMut` | done |
| `ActionHandlerWrapper` | struct | 20 |  | `actionHandlerWrapper` | done |
| `ActionHandlerWrapper::new` | fn | 23 |  | `newActionHandlerWrapper` | done |
| `impl ActionHandlerNoMut for ActionHandlerWrapper` | impl | 28 |  | `actionHandlerWrapper.DoAction` | done |
| `Context` | struct | 34 |  | `Context` | done |
| `impl Debug for Context` | impl | 42 |  | `Context.String` | done |
| `Context::new` | fn | 55 |  | `newContext` | done |
| `Context::get_or_create_platform_node` | fn | 70 |  | `Context.getOrCreatePlatformNode` | done |
| `Context::remove_platform_node` | fn | 81 |  | `Context.removePlatformNode` | done |
| `Context::do_action` | fn | 86 |  | `Context.doAction` | done |
| `impl Drop for Context` | impl | 91 |  | `Context.Release` | dev:D-MACA11YWEAK |

## src/patch.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `focus_forwarder` | fn | 17 |  | `focusForwarder` | done |
| `add_focus_forwarder_to_window_class` | fn | 39 |  | `AddFocusForwarderToWindowClass` | done |
| `add_method` | fn | 53 |  | `addMethod` | done |
| `count_args` | fn | 81 |  | `countArgs` | done |
| `method_type_encoding` | fn | 85 |  | `methodTypeEncoding` | done |

## src/node.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SCROLL_TO_VISIBLE_ACTION` | const | 34 |  | `scrollToVisibleAction` | done |
| `ns_role` | fn | 36 |  | `nsRole` | done |
| `ns_sub_role` | fn | 233 |  | `nsSubRole` | done |
| `can_be_focused` | fn | 288 |  | `canBeFocused` | done |
| `Value` | enum | 293 |  | `Value` | done |
| `NodeWrapper` | struct | 299 |  | `nodeWrapper` | done |
| `NodeWrapper::is_root` | fn | 302 |  | `nodeWrapper.isRoot` | done |
| `NodeWrapper::title` | fn | 306 |  | `nodeWrapper.title` | done |
| `NodeWrapper::description` | fn | 315 |  | `nodeWrapper.description` | done |
| `NodeWrapper::placeholder` | fn | 319 |  | `nodeWrapper.placeholder` | done |
| `NodeWrapper::value` | fn | 323 |  | `nodeWrapper.value` | done |
| `NodeWrapper::supports_text_ranges` | fn | 341 |  | `nodeWrapper.supportsTextRanges` | done |
| `NodeWrapper::raw_text_selection` | fn | 345 |  | `nodeWrapper.rawTextSelection` | done |
| `NodeWrapper::is_container_with_selectable_children` | fn | 349 |  | `nodeWrapper.isContainerWithSelectableChildren` | done |
| `NodeWrapper::is_item_like` | fn | 353 |  | `nodeWrapper.isItemLike` | done |
| `downcast_ref` | fn | 360 |  | `downcastRef` | done |
| `PlatformNodeIvars` | struct | 365 |  | `platformNodeIvars` | dev:D-MACA11YIVARS |
| `PlatformNode` (`AccessKitNode`) | struct | 372 |  | `platformNodeClass` | done |
| `PlatformNode::parent` (`accessibilityParent`) | method | 387 |  | `platformNodeParent` | done |
| `PlatformNode::window` (`accessibilityWindow`) | method | 402 |  | `platformNodeWindow` | done |
| `PlatformNode::top_level` (`accessibilityTopLevelUIElement`) | method | 413 |  | `platformNodeTopLevel` | done |
| `PlatformNode::children` (`accessibilityChildren`) | method | 424 |  | `platformNodeChildren` | done |
| `PlatformNode::children_in_navigation_order` (`accessibilityChildrenInNavigationOrder`) | method | 429 |  | `platformNodeChildrenInNavigationOrder` | done |
| `PlatformNode::selected_children` (`accessibilitySelectedChildren`) | method | 435 |  | `platformNodeSelectedChildren` | done |
| `PlatformNode::frame` (`accessibilityFrame`) | method | 452 |  | `platformNodeFrame` | done |
| `PlatformNode::role` (`accessibilityRole`) | method | 476 |  | `platformNodeRole` | done |
| `PlatformNode::sub_role` (`accessibilitySubrole`) | method | 483 |  | `platformNodeSubRole` | done |
| `PlatformNode::role_description` (`accessibilityRoleDescription`) | method | 490 |  | `platformNodeRoleDescription` | done |
| `PlatformNode::identifier` (`accessibilityIdentifier`) | method | 502 |  | `platformNodeIdentifier` | done |
| `PlatformNode::title` (`accessibilityTitle`) | method | 510 |  | `platformNodeTitle` | done |
| `PlatformNode::description` (`accessibilityHelp`) | method | 519 |  | `platformNodeDescription` | done |
| `PlatformNode::placeholder` (`accessibilityPlaceholderValue`) | method | 528 |  | `platformNodePlaceholder` | done |
| `PlatformNode::value` (`accessibilityValue`) | method | 537 |  | `platformNodeValue` | done |
| `PlatformNode::set_value` (`setAccessibilityValue:`) | method | 556 |  | `platformNodeSetValue` | done |
| `PlatformNode::min_value` (`accessibilityMinValue`) | method | 583 |  | `platformNodeMinValue` | done |
| `PlatformNode::max_value` (`accessibilityMaxValue`) | method | 591 |  | `platformNodeMaxValue` | done |
| `PlatformNode::url` (`accessibilityURL`) | method | 599 |  | `platformNodeURL` | done |
| `PlatformNode::orientation` (`accessibilityOrientation`) | method | 610 |  | `platformNodeOrientation` | done |
| `PlatformNode::is_accessibility_element` (`isAccessibilityElement`) | method | 622 |  | `platformNodeIsAccessibilityElement` | done |
| `PlatformNode::is_focused` (`isAccessibilityFocused`) | method | 628 |  | `platformNodeIsFocused` | done |
| `PlatformNode::is_enabled` (`isAccessibilityEnabled`) | method | 634 |  | `platformNodeIsEnabled` | done |
| `PlatformNode::set_focused` (`setAccessibilityFocused:`) | method | 639 |  | `platformNodeSetFocused` | done |
| `PlatformNode::press` (`accessibilityPerformPress`) | method | 669 |  | `platformNodePress` | done |
| `PlatformNode::increment` (`accessibilityPerformIncrement`) | method | 688 |  | `platformNodeIncrement` | done |
| `PlatformNode::decrement` (`accessibilityPerformDecrement`) | method | 707 |  | `platformNodeDecrement` | done |
| `PlatformNode::notifies_when_destroyed` (`accessibilityNotifiesWhenDestroyed`) | method | 726 |  | `platformNodeNotifiesWhenDestroyed` | done |
| `PlatformNode::number_of_characters` (`accessibilityNumberOfCharacters`) | method | 731 |  | `platformNodeNumberOfCharacters` | done |
| `PlatformNode::selected_text` (`accessibilitySelectedText`) | method | 743 |  | `platformNodeSelectedText` | done |
| `PlatformNode::selected_text_range` (`accessibilitySelectedTextRange`) | method | 757 |  | `platformNodeSelectedTextRange` | done |
| `PlatformNode::insertion_point_line_number` (`accessibilityInsertionPointLineNumber`) | method | 770 |  | `platformNodeInsertionPointLineNumber` | done |
| `PlatformNode::range_for_line` (`accessibilityRangeForLine:`) | method | 783 |  | `platformNodeRangeForLine` | done |
| `PlatformNode::range_for_position` (`accessibilityRangeForPosition:`) | method | 796 |  | `platformNodeRangeForPosition` | done |
| `PlatformNode::string_for_range` (`accessibilityStringForRange:`) | method | 816 |  | `platformNodeStringForRange` | done |
| `PlatformNode::attributed_string_for_range` (`accessibilityAttributedStringForRange:`) | method | 830 |  | `platformNodeAttributedStringForRange` | done |
| `PlatformNode::frame_for_range` (`accessibilityFrameForRange:`) | method | 936 |  | `platformNodeFrameForRange` | done |
| `PlatformNode::line_for_index` (`accessibilityLineForIndex:`) | method | 961 |  | `platformNodeLineForIndex` | done |
| `PlatformNode::range_for_index` (`accessibilityRangeForIndex:`) | method | 974 |  | `platformNodeRangeForIndex` | done |
| `PlatformNode::style_range_for_index` (`accessibilityStyleRangeForIndex:`) | method | 987 |  | `platformNodeStyleRangeForIndex` | done |
| `PlatformNode::set_selected_text_range` (`setAccessibilitySelectedTextRange:`) | method | 1007 |  | `platformNodeSetSelectedTextRange` | done |
| `PlatformNode::is_required` (`isAccessibilityRequired`) | method | 1025 |  | `platformNodeIsRequired` | done |
| `PlatformNode::is_selected` (`isAccessibilitySelected`) | method | 1031 |  | `platformNodeIsSelected` | done |
| `PlatformNode::set_selected` (`setAccessibilitySelected:`) | method | 1042 |  | `platformNodeSetSelected` | done |
| `PlatformNode::accessibility_attribute_value` (`accessibilityAttributeValue:`) | method | 1066 |  | `platformNodeAccessibilityAttributeValue` | done |
| `PlatformNode::rows` (`accessibilityRows`) | method | 1080 |  | `platformNodeRows` | done |
| `PlatformNode::selected_rows` (`accessibilitySelectedRows`) | method | 1096 |  | `platformNodeSelectedRows` | done |
| `PlatformNode::pick` (`accessibilityPerformPick`) | method | 1113 |  | `platformNodePick` | done |
| `PlatformNode::linked_ui_elements` (`accessibilityLinkedUIElements`) | method | 1135 |  | `platformNodeLinkedUIElements` | done |
| `PlatformNode::tabs` (`accessibilityTabs`) | method | 1152 |  | `platformNodeTabs` | done |
| `PlatformNode::is_modal` (`isAccessibilityModal`) | method | 1168 |  | `platformNodeIsModal` | done |
| `PlatformNode::action_names` (`accessibilityActionNames`) | method | 1181 |  | `platformNodeActionNames` | done |
| `PlatformNode::perform_action` (`accessibilityPerformAction:`) | method | 1192 |  | `platformNodePerformAction` | done |
| `PlatformNode::is_selector_allowed` (`isAccessibilitySelectorAllowed:`) | method | 1208 |  | `platformNodeIsSelectorAllowed` | done |
| `PlatformNode::new` | fn | 786 |  | `newPlatformNode` | done |
| `PlatformNode::resolve_with_context` | fn | 792 |  | `resolveWithContext` | done |
| `PlatformNode::resolve` | fn | 803 |  | `resolve` | done |
| `PlatformNode::children_internal` | fn | 810 |  | `childrenInternal` | done |
| `PlatformNode` dealloc of ivars | impl | 372 |  | `platformNodeDealloc` | dev:D-MACA11YIVARS |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `text alignment values (added)` | test | 0 |  | `TestNode_TextAlignmentValues` | done |
| `wrapper value and title (added)` | test | 0 |  | `TestNode_WrapperValueAndTitle` | done |
| `update queues title and value changes (added)` | test | 0 |  | `TestEvent_UpdateQueuesTitleAndValueChanges` | done |
| `method type encoding (added)` | test | 0 |  | `TestPatch_MethodTypeEncoding` | done |

## src/event.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `QueuedEvent` | enum | 22 |  | `queuedEvent` | done |
| `QueuedEvent::live_region_announcement` | fn | 35 |  | `liveRegionAnnouncement` | done |
| `QueuedEvent::raise` | fn | 46 |  | `queuedEvent.raise` | done |
| `QueuedEvents` | struct | 108 |  | `QueuedEvents` | done |
| `QueuedEvents::new` | fn | 114 |  | `newQueuedEvents` | done |
| `QueuedEvents::raise` | fn | 125 |  | `QueuedEvents.Raise` | done |
| `focus_event` | fn | 132 |  | `focusEvent` | done |
| `EventGenerator` | struct | 139 |  | `eventGenerator` | done |
| `EventGenerator::new` | fn | 147 |  | `newEventGenerator` | done |
| `EventGenerator::into_result` | fn | 156 |  | `eventGenerator.intoResult` | done |
| `EventGenerator::remove_subtree` | fn | 160 |  | `eventGenerator.removeSubtree` | done |
| `EventGenerator::insert_text_change_if_needed_parent` | fn | 173 |  | `eventGenerator.insertTextChangeIfNeededParent` | done |
| `EventGenerator::insert_text_change_if_needed` | fn | 194 |  | `eventGenerator.insertTextChangeIfNeeded` | done |
| `EventGenerator::enqueue_selected_rows_change_if_needed_parent` | fn | 203 |  | `eventGenerator.enqueueSelectedRowsChangeIfNeededParent` | done |
| `EventGenerator::enqueue_selected_rows_change_if_needed` | fn | 215 |  | `eventGenerator.enqueueSelectedRowsChangeIfNeeded` | done |
| `TreeChangeHandler::node_added` | method | 227 |  | `eventGenerator.NodeAdded` | done |
| `TreeChangeHandler::node_updated` | method | 241 |  | `eventGenerator.NodeUpdated` | done |
| `TreeChangeHandler::focus_moved` | method | 308 |  | `eventGenerator.FocusMoved` | done |
| `TreeChangeHandler::node_removed` | method | 317 |  | `eventGenerator.NodeRemoved` | done |

## src/adapter.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PLACEHOLDER_ROOT_ID` | const | 29 |  | `placeholderRootID` | done |
| `State` | enum | 31 |  | `state` | done |
| `impl Debug for State` | impl | 46 |  | `state.String` | done |
| `PlaceholderActionHandler` | struct | 74 |  | `placeholderActionHandler` | done |
| `Adapter` | struct | 81 |  | `Adapter` | done |
| `Adapter::new` | fn | 94 |  | `NewAdapter` | done |
| `Adapter::update_if_active` | fn | 119 |  | `Adapter.UpdateIfActive` | done |
| `Adapter::update_view_focus_state` | fn | 156 |  | `Adapter.UpdateViewFocusState` | done |
| `Adapter::get_or_init_context` | fn | 179 |  | `Adapter.getOrInitContext` | done |
| `Adapter::view_children` | fn | 228 |  | `Adapter.ViewChildren` | done |
| `Adapter::focus` | fn | 253 |  | `Adapter.Focus` | done |
| `Adapter::weak_view` | fn | 269 |  | `Adapter.weakView` | done |
| `Adapter::hit_test` | fn | 280 |  | `Adapter.HitTest` | done |
| drop of `Adapter` (implicit) | impl | 81 |  | `Adapter.Release` | dev:D-MACA11YWEAK |

## src/subclass.rs

Go target: `a11y/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SUBCLASSES` | static | 27 |  | `subclasses` | done |
| `ASSOCIATED_OBJECT_KEY` | static | 29 |  | `associatedObjects` | dev:D-MACA11YIVARS |
| `associated_object_key` | fn | 31 |  | `getAssociatedObject` | dev:D-MACA11YIVARS |
| `AssociatedObjectState` | struct | 35 |  | `associatedObjectState` | done |
| `AssociatedObjectIvars` | struct | 40 |  | `associatedObject` | done |
| `AssociatedObject` (`AccessKitSubclassAssociatedObject`) | struct | 46 |  | `associatedObject` | dev:D-MACA11YIVARS |
| `AssociatedObject::new` | fn | 60 |  | `newSubclassingAdapterInternal` | done |
| `associated_object` | fn | 75 |  | `associatedObjectOf` | done |
| `superclass` | fn | 87 |  | `subclassSuperclass` | done |
| `children` | fn | 92 |  | `subclassChildren` | done |
| `focus` | fn | 101 |  | `subclassFocus` | done |
| `hit_test` | fn | 108 |  | `subclassHitTest` | done |
| `SubclassingAdapter` | struct | 119 |  | `SubclassingAdapter` | done |
| `SubclassingAdapter::new` | fn | 134 |  | `NewSubclassingAdapter` | done |
| `SubclassingAdapter::new_internal` | fn | 144 |  | `newSubclassingAdapterInternal` | done |
| `SubclassingAdapter::for_window` | fn | 223 |  | `SubclassingAdapterForWindow` | done |
| `SubclassingAdapter::update_if_active` | fn | 241 |  | `SubclassingAdapter.UpdateIfActive` | done |
| `SubclassingAdapter::update_view_focus_state` | fn | 253 |  | `SubclassingAdapter.UpdateViewFocusState` | done |
| `impl Drop for SubclassingAdapter` | impl | 259 |  | `SubclassingAdapter.Release` | done |

## gpui_macos/src/window.rs (accesskit_macos use)

Go target: `platform/macos/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MacWindowState::accesskit_adapter` | field | 713 |  | `macWindowState.a11yAdapter` | done |
| `accesskit_macos::SubclassingAdapter::for_window` in `a11y_init` | call | 2540 |  | `newA11yAdapter` | done |
| `SubclassingAdapter::update_if_active` in `a11y_tree_update` | call | 2553 |  | `subclassingA11yAdapter.UpdateIfActive` | done |
| `SubclassingAdapter::update_view_focus_state` in `window_did_change_key_status` | call | 3437 |  | `subclassingA11yAdapter.UpdateViewFocusState` | done |
| `drop(this.accesskit_adapter.take())` in `impl Drop for MacWindow` | call | 1467 |  | `subclassingA11yAdapter.Close` | done |

## Deviations registered here

| Id | Meaning | Owner |
|---|---|---|
| `D-MACA11YWEAK` | `Weak<Context>` is `weak.Pointer[Context]` plus a `released` flag; the drop of the last `Rc<Context>` is the explicit `Context.Release`, run when the placeholder context is replaced and from `Adapter.Release` (the implicit drop of `Adapter`, reached through `SubclassingAdapter.Release`). Contexts still referenced by unraised `QueuedEvents` are released with the adapter, not when the events are dropped. | M4-macos-a11y |
| `D-MACA11YIVARS` | Rust ivars of `AccessKitNode` and the `AccessKitSubclassAssociatedObject` attached with `objc_setAssociatedObject` are Go side tables keyed by the object pointer (`platformNodeIvarMap`, `associatedObjects`), as platform/macos does for its own classes; `AccessKitNode` overrides `dealloc` to drop its entry. No associated-object class is registered. | M4-macos-a11y |
| `D-MACA11YOBJC` | objc2 / objc2-foundation re-exports and encoding helpers (`NSArray`, `NSObject`, `NSPoint`, `CGColor` + `RefEncode`) have no Go type of their own: objects are `objc.ID`, points `objc.CGPoint`. | M4-macos-a11y |
