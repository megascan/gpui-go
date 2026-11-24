# Parity: ext-a11y-windows

Scope: accesskit_windows 0.34.0 (`src/*.rs`, `src/tests/*.rs`). Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_windows-0.34.0/`.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`, `blocked:<stage>`. M4-windows-a11y-node ports `node.rs`, `util.rs`, `context.rs`, `filters.rs` and `window_handle.rs` into package `a11y/windows` (UIA constants and interop in `uia.go`, the float-argument trampolines in `node_amd64.{go,s}` / `node_arm64.{go,s}`); `stubs.go` declares the `text.rs` / `adapter.rs` / `subclass.rs` types for M4-windows-a11y, which ports them with `lib.rs` and `tests/` and deletes `stubs.go`.

Notes on the port:

- windows-rs `Result<T>` is `(T, error)` with the error an `com.HRESULT`; `Err(Error::empty())` (HRESULT 0) is `(nil, nil)`, which the vtable thunks turn into S_OK with a null out pointer, as windows-rs does.
- An interface handed out (`into_interface`, `to_interface`, `cast`) carries one reference that the receiver owns; `Context` holds one reference per platform node and drops it in `removePlatformNode` / `Release`.
- `GetPatternProvider` and `ControllerFor` return the identity (`IUnknown`) pointer, as upstream's `cast::<IUnknown>()` does; QueryInterface answers all 13 provider IIDs.
- `IValueProvider::SetValue` decodes the PCWSTR with `windows.UTF16PtrToString` (lossy on unpaired surrogates, where upstream's `to_string().unwrap()` panics).
- Tests run only on Windows (CI); locally they are cross-compiled.

## src/lib.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `mod context` | mod | 6 |  | `Context` | done |
| `mod filters` | mod | 7 |  | `filter` | done |
| `mod node` | mod | 8 |  | `platformNode` | done |
| `mod text` | mod | 9 |  | `platformRange` | blocked:M4-windows-a11y |
| `mod util` | mod | 10 |  | `Variant` | done |
| `mod window_handle` | mod | 11 |  | `WindowHandle` | done |
| `pub use adapter::{Adapter, QueuedEvents}` | use | 14 |  | `Adapter` | blocked:M4-windows-a11y |
| `pub use subclass::SubclassingAdapter` | use | 17 |  | `SubclassingAdapter` | blocked:M4-windows-a11y |
| `pub use windows::Win32::Foundation::{HWND, LPARAM, LRESULT, WPARAM}` | use | 19 |  | `HWND` | blocked:M4-windows-a11y |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `mod tests` | cfg | 21 | test | `TestSimple_HasNativeUia` | blocked:M4-windows-a11y |

## src/window_handle.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WindowHandle` | struct | 9 |  | `WindowHandle` | done |
| `impl From<HWND> for WindowHandle` | impl | 15 |  | `WindowHandleFromHWND` | done |
| `impl From<WindowHandle> for HWND` | impl | 21 |  | `WindowHandle.HWND` | done |
| `unsafe impl Send/Sync for WindowHandle` | impl | 12 |  | `WindowHandle` | n/a:D-WINSENDSYNC |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests::assert_impl_all!(WindowHandle: Send, Sync)` | static | 30 | test | `WindowHandle` | n/a:D-WINSENDSYNC |

## src/filters.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `common_filter as filter` | use | 6 |  | `filter` | done |
| `common_filter_with_root_exception as filter_with_root_exception` | use | 7 |  | `filterWithRootException` | done |

## src/context.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ActionHandlerNoMut` | trait | 15 |  | `actionHandlerNoMut` | done |
| `ActionHandlerWrapper` | struct | 19 |  | `actionHandlerWrapper` | done |
| `ActionHandlerWrapper::new` | fn | 22 |  | `newActionHandlerWrapper` | done |
| `impl ActionHandlerNoMut for ActionHandlerWrapper` | impl | 27 |  | `actionHandlerWrapper.DoAction` | done |
| `Context` | struct | 33 |  | `Context` | dev:D-WINWEAK |
| `impl Debug for Context` | impl | 41 |  | `Context.String` | done |
| `Context::new` | fn | 22 |  | `newContext` | done |
| `Context::read_tree` | fn | 68 |  | `Context.readTree` | done |
| `Context::client_top_left` | fn | 72 |  | `Context.clientTopLeft` | done |
| `Context::do_action` | fn | 16 |  | `Context.doAction` | done |
| `Context::get_or_create_platform_node` | fn | 80 |  | `Context.getOrCreatePlatformNode` | done |
| `Context::remove_platform_node` | fn | 94 |  | `Context.removePlatformNode` | done |
| `drop of Arc<Context>` | drop | 33 |  | `Context.Release` | dev:D-WINWEAK |

## src/util.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `STRING_SCRATCH` | static | 29 |  | `stringScratchPool` | dev:D-WINSTRBUF |
| `MAX_RETAINED_SCRATCH_CAPACITY` | const | 32 |  | `maxRetainedScratchCapacity` | done |
| `MAX_POOLED_BUFFERS` | const | 33 |  | `maxPooledBuffers` | done |
| `StringBuffer` | struct | 36 |  | `stringBuffer` | dev:D-WINSTRBUF |
| `StringBuffer::acquire` | fn | 39 |  | `acquireStringBuffer` | done |
| `impl Drop for StringBuffer` | drop | 45 |  | `stringBuffer.release` | done |
| `impl Deref/DerefMut for StringBuffer` | impl | 60 |  | `stringBuffer.buf` | done |
| `WideString` | struct | 73 |  | `wideString` | done |
| `WideString::new` | fn | 76 |  | `newWideString` | done |
| `impl Write for WideString` | impl | 81 |  | `wideString.WriteString` | done |
| `impl PartialEq for WideString` | impl | 93 |  | `wideString.equal` | done |
| `impl From<WideString> for BSTR` | impl | 99 |  | `wideString.toBSTR` | done |
| `LocaleName` | struct | 107 |  | `localeName` | dev:D-WINSTRBUF |
| `impl From<LocaleName> for Variant` | impl | 108 |  | `variantFromLocaleName` | done |
| `Variant` | struct | 116 |  | `Variant` | done |
| `impl From<Variant> for VARIANT` | impl | 117 |  | `Variant.into` | done |
| `Variant::empty` | fn | 125 |  | `variantEmpty` | done |
| `Variant::is_empty` | fn | 129 |  | `Variant.isEmpty` | done |
| `impl From<BSTR> for Variant` | impl | 133 |  | `variantFromBSTR` | done |
| `impl From<WideString> for Variant` | impl | 139 |  | `wideString.toVariant` | done |
| `StrWrapper` | struct | 147 |  | `optString` | dev:D-WINSTRBUF |
| `StrWrapper::new` | fn | 76 |  | `variantFromString` | dev:D-WINSTRBUF |
| `impl PartialEq for StrWrapper` | impl | 157 |  | `optString` | dev:D-WINSTRBUF |
| `impl From<StrWrapper> for Variant` | impl | 163 |  | `variantFromString` | done |
| `impl From<IUnknown> for Variant` | impl | 171 |  | `variantFromUnknown` | done |
| `impl From<i32> for Variant` | impl | 177 |  | `variantFromI32` | done |
| `impl From<f64> for Variant` | impl | 183 |  | `variantFromF64` | done |
| `impl From<ToggleState> for Variant` | impl | 189 |  | `i32Value.toVariant` | done |
| `impl From<ExpandCollapseState> for Variant` | impl | 195 |  | `i32Value.toVariant` | done |
| `impl From<LiveSetting> for Variant` | impl | 201 |  | `i32Value.toVariant` | done |
| `impl From<CaretPosition> for Variant` | impl | 207 |  | `variantFromI32` | done |
| `impl From<UIA_CONTROLTYPE_ID> for Variant` | impl | 213 |  | `i32Value.toVariant` | done |
| `impl From<OrientationType> for Variant` | impl | 219 |  | `i32Value.toVariant` | done |
| `impl From<Color> for Variant` | impl | 225 |  | `variantFromColor` | done |
| `impl From<TextDecorationStyle> for Variant` | impl | 233 |  | `variantFromTextDecorationStyle` | done |
| `impl From<TextAlign> for Variant` | impl | 246 |  | `variantFromTextAlign` | done |
| `impl From<bool> for Variant` | impl | 258 |  | `variantFromBool` | done |
| `impl From<Option<T>> for Variant` | impl | 264 |  | `variantFromOption` | done |
| `impl From<TextRangePropertyValue<T>> for Variant` | impl | 270 |  | `variantFromTextRangePropertyValue` | done |
| `impl From<Vec<IUnknown>> for Variant` | impl | 281 |  | `variantFromUnknowns` | done |
| `safe_array_from_primitive_slice` | fn | 303 |  | `safeArrayFromPrimitiveSlice` | done |
| `safe_array_from_i32_slice` | fn | 315 |  | `safeArrayFromI32Slice` | done |
| `safe_array_from_f64_slice` | fn | 319 |  | `safeArrayFromF64Slice` | done |
| `safe_array_from_com_slice` | fn | 323 |  | `safeArrayFromComSlice` | done |
| `QueuedEvent` | enum | 335 |  | `queuedEvent` | done |
| `QueuedEvent::Simple` | variant | 336 |  | `queuedEventSimple` | done |
| `QueuedEvent::PropertyChanged` | variant | 340 |  | `queuedEventPropertyChanged` | done |
| `drop of QueuedEvent` | drop | 335 |  | `queuedEvent.release` | done |
| `not_implemented` | fn | 348 |  | `notImplemented` | done |
| `invalid_arg` | fn | 352 |  | `invalidArg` | done |
| `required_param` | fn | 356 |  | `requiredParam` | done |
| `element_not_available` | fn | 360 |  | `elementNotAvailable` | done |
| `element_not_enabled` | fn | 364 |  | `elementNotEnabled` | done |
| `invalid_operation` | fn | 368 |  | `invalidOperation` | done |
| `not_supported` | fn | 372 |  | `notSupported` | done |
| `client_top_left` | fn | 376 |  | `clientTopLeft` | done |
| `window_title` | fn | 384 |  | `windowTitle` | done |
| `toolkit_description` | fn | 413 |  | `toolkitDescription` | done |
| `upgrade` | fn | 426 |  | `upgrade` | dev:D-WINWEAK |
| `AriaProperties` | struct | 433 |  | `ariaProperties` | done |
| `AriaProperties::new` | fn | 76 |  | `newAriaProperties` | done |
| `AriaProperties::write_property` | fn | 446 |  | `ariaProperties.writeProperty` | done |
| `AriaProperties::write_bool_property` | fn | 457 |  | `ariaProperties.writeBoolProperty` | done |
| `AriaProperties::write_usize_property` | fn | 461 |  | `ariaProperties.writeUsizeProperty` | done |
| `AriaProperties::has_properties` | fn | 465 |  | `ariaProperties.hasProperties` | done |
| `UIA constants and interop (windows crate: UI::Accessibility, LocaleNameToLCID, SysAllocStringLen)` | use | 15 |  | `uiaHostProviderFromHwnd` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `acquire_reuses_dropped_allocation` | test | 479 |  | `TestUtil_AcquireReusesDroppedAllocation` | dev:D-WINSTRBUF |
| `simultaneous_leases_are_distinct` | test | 493 |  | `TestUtil_SimultaneousLeasesAreDistinct` | dev:D-WINSTRBUF |
| `outsized_allocation_is_released_on_drop` | test | 506 |  | `TestUtil_OutsizedAllocationIsReleasedOnDrop` | dev:D-WINSTRBUF |
| `pool_retains_at_most_max_pooled_buffers` | test | 519 |  | `TestUtil_PoolRetainsAtMostMaxPooledBuffers` | dev:D-WINSTRBUF |
| `tests::on_fresh_thread` | fixture | 474 | test | `stringScratch` | dev:D-WINSTRBUF |

## src/node.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `RUNTIME_ID_SIZE` | const | 38 |  | `runtimeIDSize` | done |
| `runtime_id_from_node_id` | fn | 40 |  | `runtimeIDFromNodeID` | done |
| `NodeWrapper` | struct | 52 |  | `nodeWrapper` | done |
| `NodeWrapper::control_type` | fn | 58 |  | `nodeWrapper.controlType` | done |
| `NodeWrapper::localized_control_type` | fn | 267 |  | `nodeWrapper.localizedControlType` | done |
| `NodeWrapper::aria_role` | fn | 273 |  | `nodeWrapper.ariaRole` | done |
| `NodeWrapper::name` | fn | 404 |  | `nodeWrapper.name` | done |
| `NodeWrapper::description` | fn | 415 |  | `nodeWrapper.description` | done |
| `NodeWrapper::culture` | fn | 423 |  | `nodeWrapper.culture` | done |
| `NodeWrapper::placeholder` | fn | 427 |  | `nodeWrapper.placeholder` | done |
| `NodeWrapper::is_content_element` | fn | 433 |  | `nodeWrapper.isContentElement` | done |
| `NodeWrapper::aria_properties` | fn | 437 |  | `nodeWrapper.ariaProperties` | done |
| `NodeWrapper::is_enabled` | fn | 534 |  | `nodeWrapper.isEnabled` | done |
| `NodeWrapper::is_focusable` | fn | 538 |  | `nodeWrapper.isFocusable` | done |
| `NodeWrapper::is_focused` | fn | 542 |  | `nodeWrapper.isFocused` | done |
| `NodeWrapper::live_setting` | fn | 546 |  | `nodeWrapper.liveSetting` | done |
| `NodeWrapper::automation_id` | fn | 555 |  | `nodeWrapper.automationID` | done |
| `NodeWrapper::class_name` | fn | 561 |  | `nodeWrapper.className` | done |
| `NodeWrapper::orientation` | fn | 567 |  | `nodeWrapper.orientation` | done |
| `NodeWrapper::is_toggle_pattern_supported` | fn | 575 |  | `nodeWrapper.isTogglePatternSupported` | done |
| `NodeWrapper::toggle_state` | fn | 579 |  | `nodeWrapper.toggleState` | done |
| `NodeWrapper::is_invoke_pattern_supported` | fn | 587 |  | `nodeWrapper.isInvokePatternSupported` | done |
| `NodeWrapper::is_value_pattern_supported` | fn | 591 |  | `nodeWrapper.isValuePatternSupported` | done |
| `NodeWrapper::is_range_value_pattern_supported` | fn | 598 |  | `nodeWrapper.isRangeValuePatternSupported` | done |
| `NodeWrapper::value` | fn | 602 |  | `nodeWrapper.value` | done |
| `NodeWrapper::is_read_only` | fn | 613 |  | `nodeWrapper.isReadOnly` | done |
| `NodeWrapper::numeric_value` | fn | 617 |  | `nodeWrapper.numericValue` | done |
| `NodeWrapper::min_numeric_value` | fn | 621 |  | `nodeWrapper.minNumericValue` | done |
| `NodeWrapper::max_numeric_value` | fn | 625 |  | `nodeWrapper.maxNumericValue` | done |
| `NodeWrapper::numeric_value_step` | fn | 629 |  | `nodeWrapper.numericValueStep` | done |
| `NodeWrapper::numeric_value_jump` | fn | 633 |  | `nodeWrapper.numericValueJump` | done |
| `NodeWrapper::is_required` | fn | 639 |  | `nodeWrapper.isRequired` | done |
| `NodeWrapper::is_scroll_item_pattern_supported` | fn | 643 |  | `nodeWrapper.isScrollItemPatternSupported` | done |
| `NodeWrapper::is_selection_item_pattern_supported` | fn | 647 |  | `nodeWrapper.isSelectionItemPatternSupported` | done |
| `NodeWrapper::is_selected` | fn | 668 |  | `nodeWrapper.isSelected` | done |
| `NodeWrapper::position_in_set` | fn | 681 |  | `nodeWrapper.positionInSet` | done |
| `NodeWrapper::size_of_set` | fn | 688 |  | `nodeWrapper.sizeOfSet` | done |
| `NodeWrapper::level` | fn | 694 |  | `nodeWrapper.level` | done |
| `NodeWrapper::is_selection_pattern_supported` | fn | 701 |  | `nodeWrapper.isSelectionPatternSupported` | done |
| `NodeWrapper::is_multiselectable` | fn | 705 |  | `nodeWrapper.isMultiselectable` | done |
| `NodeWrapper::is_text_pattern_supported` | fn | 709 |  | `nodeWrapper.isTextPatternSupported` | done |
| `NodeWrapper::is_expand_collapse_pattern_supported` | fn | 713 |  | `nodeWrapper.isExpandCollapsePatternSupported` | done |
| `NodeWrapper::expand_collapse_state` | fn | 717 |  | `nodeWrapper.expandCollapseState` | done |
| `NodeWrapper::is_password` | fn | 726 |  | `nodeWrapper.isPassword` | done |
| `NodeWrapper::is_dialog` | fn | 730 |  | `nodeWrapper.isDialog` | done |
| `NodeWrapper::is_window_pattern_supported` | fn | 734 |  | `nodeWrapper.isWindowPatternSupported` | done |
| `NodeWrapper::is_modal` | fn | 738 |  | `nodeWrapper.isModal` | done |
| `NodeWrapper::enqueue_property_changes` | fn | 742 |  | `nodeWrapper.enqueuePropertyChanges` | done |
| `NodeWrapper::enqueue_property_implied_events` | fn | 754 |  | `nodeWrapper.enqueuePropertyImpliedEvents` | done |
| `enqueue_property_change` | fn | 772 |  | `enqueuePropertyChange` | done |
| `PlatformNode` | struct | 804 |  | `platformNode` | dev:D-WINWEAK |
| `#[implement(...)] for PlatformNode` | attr | 797 |  | `buildPlatformNodeVtables` | dev:D-WINCOM |
| `PlatformNode::new` | fn | 810 |  | `newPlatformNode` | done |
| `PlatformNode::unspecified_root` | fn | 818 |  | `unspecifiedRootPlatformNode` | done |
| `PlatformNode::upgrade_context` | fn | 826 |  | `platformNode.upgradeContext` | done |
| `PlatformNode::with_tree_state_and_context` | fn | 830 |  | `withTreeStateAndContext` | done |
| `PlatformNode::with_tree_and_context` | fn | 837 |  | `withTreeAndContext` | done |
| `PlatformNode::node` | fn | 846 |  | `platformNode.node` | done |
| `PlatformNode::node_with_location` | fn | 859 |  | `platformNode.nodeWithLocation` | done |
| `PlatformNode::resolve_with_context` | fn | 868 |  | `resolveWithContext` | done |
| `PlatformNode::resolve_with_tree_state_and_context` | fn | 878 |  | `resolveWithTreeStateAndContext` | done |
| `PlatformNode::resolve` | fn | 888 |  | `resolve` | done |
| `PlatformNode::resolve_with_context_for_text_pattern` | fn | 895 |  | `resolveWithContextForTextPattern` | done |
| `PlatformNode::resolve_for_text_pattern` | fn | 909 |  | `resolveForTextPattern` | done |
| `PlatformNode::do_complex_action` | fn | 916 |  | `platformNode.doComplexAction` | done |
| `PlatformNode::do_action` | fn | 933 |  | `platformNode.doAction` | done |
| `PlatformNode::click` | fn | 951 |  | `platformNode.click` | done |
| `PlatformNode::set_expanded` | fn | 955 |  | `platformNode.setExpanded` | done |
| `PlatformNode::set_selected` | fn | 979 |  | `platformNode.setSelected` | done |
| `PlatformNode::is_root` | fn | 1001 |  | `platformNode.isRoot` | done |
| `IRawElementProviderSimple_Impl::ProviderOptions` | fn | 1008 |  | `platformNode.providerOptions` | done |
| `IRawElementProviderSimple_Impl::GetPatternProvider` | fn | 1012 |  | `platformNode.getPatternProvider` | done |
| `IRawElementProviderSimple_Impl::GetPropertyValue` | fn | 1016 |  | `platformNode.getPropertyValue` | done |
| `IRawElementProviderSimple_Impl::HostRawElementProvider` | fn | 1060 |  | `platformNode.hostRawElementProvider` | done |
| `IRawElementProviderFragment_Impl::Navigate` | fn | 1073 |  | `platformNode.navigate` | done |
| `IRawElementProviderFragment_Impl::GetRuntimeId` | fn | 1094 |  | `platformNode.getRuntimeID` | done |
| `IRawElementProviderFragment_Impl::BoundingRectangle` | fn | 1109 |  | `platformNode.boundingRectangle` | done |
| `IRawElementProviderFragment_Impl::GetEmbeddedFragmentRoots` | fn | 1124 |  | `platformNode.getEmbeddedFragmentRoots` | done |
| `IRawElementProviderFragment_Impl::SetFocus` | fn | 1128 |  | `platformNode.setFocus` | done |
| `IRawElementProviderFragment_Impl::FragmentRoot` | fn | 1132 |  | `platformNode.fragmentRoot` | done |
| `IRawElementProviderFragmentRoot_Impl::ElementProviderFromPoint` | fn | 1148 |  | `platformNode.elementProviderFromPoint` | dev:D-WINFLOAT |
| `IRawElementProviderFragmentRoot_Impl::GetFocus` | fn | 1164 |  | `platformNode.getFocus` | done |
| `macro_rules! properties` | macro | 1182 |  | `nodeProperties` | dev:D-WINCOM |
| `NodeWrapper::get_property_value` | fn | 1185 |  | `nodeWrapper.getPropertyValue` | done |
| `NodeWrapper::enqueue_simple_property_changes` | fn | 1193 |  | `nodeWrapper.enqueueSimplePropertyChanges` | done |
| `macro_rules! patterns` | macro | 1244 |  | `nodePatterns` | dev:D-WINCOM |
| `PlatformNode_Impl::pattern_provider` | fn | 1251 |  | `platformNode.patternProvider` | done |
| `NodeWrapper::enqueue_pattern_property_changes` | fn | 1267 |  | `nodeWrapper.enqueuePatternPropertyChanges` | done |
| `patterns! generated getters` | fn | 1300 |  | `patternGetter` | done |
| `properties! { ... } list` | macro | 1313 |  | `nodeProperties` | done |
| `IToggleProvider_Impl::Toggle` | fn | 1335 |  | `platformNode.toggle` | done |
| `IToggleProvider_Impl::ToggleState` | fn | 1341 |  | `nodeWrapper.toggleState` | done |
| `IInvokeProvider_Impl::Invoke` | fn | 1340 |  | `platformNode.invoke` | done |
| `IValueProvider_Impl::SetValue` | fn | 1348 |  | `platformNode.valueSetValue` | done |
| `IValueProvider_Impl::Value / IsReadOnly` | fn | 1353 |  | `nodeWrapper.value` | done |
| `IRangeValueProvider_Impl::SetValue` | fn | 1363 |  | `platformNode.rangeValueSetValue` | dev:D-WINFLOAT |
| `IRangeValueProvider_Impl::Value / IsReadOnly / Minimum / Maximum / SmallChange / LargeChange` | fn | 1364 |  | `f64Getter` | done |
| `IScrollItemProvider_Impl::ScrollIntoView` | fn | 1370 |  | `platformNode.scrollIntoView` | done |
| `ISelectionItemProvider_Impl::IsSelected` | fn | 1382 |  | `platformNode.selectionItemIsSelected` | done |
| `ISelectionItemProvider_Impl::Select` | fn | 1393 |  | `platformNode.selectionItemSelect` | done |
| `ISelectionItemProvider_Impl::AddToSelection` | fn | 1397 |  | `platformNode.selectionItemAddToSelection` | done |
| `ISelectionItemProvider_Impl::RemoveFromSelection` | fn | 1401 |  | `platformNode.selectionItemRemoveFromSelection` | done |
| `ISelectionItemProvider_Impl::SelectionContainer` | fn | 1405 |  | `platformNode.selectionContainer` | done |
| `ISelectionProvider_Impl::GetSelection` | fn | 1419 |  | `platformNode.selectionGetSelection` | done |
| `ISelectionProvider_Impl::CanSelectMultiple / IsSelectionRequired` | fn | 1424 |  | `boolGetter` | done |
| `ITextProvider_Impl::GetSelection` | fn | 1432 |  | `platformNode.textGetSelection` | done |
| `ITextProvider_Impl::GetVisibleRanges` | fn | 1444 |  | `platformNode.textGetVisibleRanges` | done |
| `ITextProvider_Impl::RangeFromChild` | fn | 1451 |  | `platformNode.textRangeFromChild` | done |
| `ITextProvider_Impl::RangeFromPoint` | fn | 1456 |  | `platformNode.textRangeFromPoint` | dev:D-WINFLOAT |
| `ITextProvider_Impl::DocumentRange` | fn | 1467 |  | `platformNode.textDocumentRange` | done |
| `ITextProvider_Impl::SupportedTextSelection` | fn | 1474 |  | `platformNode.textSupportedTextSelection` | done |
| `IExpandCollapseProvider_Impl::Expand` | fn | 1487 |  | `platformNode.expand` | done |
| `IExpandCollapseProvider_Impl::Collapse` | fn | 1491 |  | `platformNode.collapse` | done |
| `IExpandCollapseProvider_Impl::ExpandCollapseState` | fn | 1493 |  | `nodeWrapper.expandCollapseState` | done |
| `IWindowProvider_Impl::SetVisualState` | fn | 1498 |  | `platformNode.windowSetVisualState` | done |
| `IWindowProvider_Impl::Close` | fn | 1502 |  | `platformNode.windowClose` | done |
| `IWindowProvider_Impl::WaitForInputIdle` | fn | 1506 |  | `platformNode.windowWaitForInputIdle` | done |
| `IWindowProvider_Impl::CanMaximize` | fn | 1510 |  | `platformNode.windowCanMaximize` | done |
| `IWindowProvider_Impl::CanMinimize` | fn | 1514 |  | `platformNode.windowCanMinimize` | done |
| `IWindowProvider_Impl::IsModal` | fn | 1504 |  | `nodeWrapper.isModal` | done |
| `IWindowProvider_Impl::WindowVisualState` | fn | 1518 |  | `platformNode.windowVisualState` | done |
| `IWindowProvider_Impl::WindowInteractionState` | fn | 1522 |  | `platformNode.windowInteractionState` | done |
| `IWindowProvider_Impl::IsTopmost` | fn | 1526 |  | `platformNode.windowIsTopmost` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `platform_node_impl_send_sync` | test | 1535 |  | `platformNode` | n/a:D-WINSENDSYNC |

## src/text.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `upgrade_range` | fn | 23 |  | `upgradeRange` | blocked:M4-windows-a11y |
| `upgrade_range_node` | fn | 31 |  | `upgradeRangeNode` | blocked:M4-windows-a11y |
| `weak_comparable_position_from_endpoint` | fn | 39 |  | `weakComparablePositionFromEndpoint` | blocked:M4-windows-a11y |
| `position_from_endpoint` | fn | 50 |  | `positionFromEndpoint` | blocked:M4-windows-a11y |
| `set_endpoint_position` | fn | 61 |  | `setEndpointPosition` | blocked:M4-windows-a11y |
| `back_to_unit_start` | fn | 80 |  | `backToUnitStart` | blocked:M4-windows-a11y |
| `move_forward_to_start` | fn | 134 |  | `moveForwardToStart` | blocked:M4-windows-a11y |
| `move_forward_to_end` | fn | 147 |  | `moveForwardToEnd` | blocked:M4-windows-a11y |
| `move_backward` | fn | 160 |  | `moveBackward` | blocked:M4-windows-a11y |
| `move_position` | fn | 173 |  | `movePosition` | blocked:M4-windows-a11y |
| `PlatformRange` | struct | 209 |  | `platformRange` | blocked:M4-windows-a11y |
| `PlatformRange::new` | fn | 215 |  | `newPlatformRange` | blocked:M4-windows-a11y |
| `PlatformRange::upgrade_context` | fn | 222 |  | `platformRange.upgradeContext` | blocked:M4-windows-a11y |
| `PlatformRange::with_tree_state_and_context` | fn | 226 |  | `platformRangeWithTreeStateAndContext` | blocked:M4-windows-a11y |
| `PlatformRange::with_tree_state` | fn | 235 |  | `platformRangeWithTreeState` | blocked:M4-windows-a11y |
| `PlatformRange::upgrade_node` | fn | 242 |  | `platformRange.upgradeNode` | blocked:M4-windows-a11y |
| `PlatformRange::upgrade_for_read` | fn | 247 |  | `platformRange.upgradeForRead` | blocked:M4-windows-a11y |
| `PlatformRange::read_with_context` | fn | 252 |  | `platformRangeReadWithContext` | blocked:M4-windows-a11y |
| `PlatformRange::read` | fn | 262 |  | `platformRangeRead` | blocked:M4-windows-a11y |
| `PlatformRange::write` | fn | 269 |  | `platformRangeWrite` | blocked:M4-windows-a11y |
| `PlatformRange::do_action` | fn | 282 |  | `platformRange.doAction` | blocked:M4-windows-a11y |
| `PlatformRange::require_same_context` | fn | 295 |  | `platformRange.requireSameContext` | blocked:M4-windows-a11y |
| `impl Clone for PlatformRange` | impl | 304 |  | `platformRange.clone` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::Clone` | fn | 320 |  | `platformRange.clone` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::Compare` | fn | 324 |  | `platformRange.compare` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::CompareEndpoints` | fn | 331 |  | `platformRange.compareEndpoints` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::ExpandToEnclosingUnit` | fn | 364 |  | `platformRange.expandToEnclosingUnit` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::FindAttribute` | fn | 395 |  | `platformRange.findAttribute` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::FindText` | fn | 406 |  | `platformRange.findText` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::GetAttributeValue` | fn | 418 |  | `platformRange.getAttributeValue` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::GetBoundingRectangles` | fn | 493 |  | `platformRange.getBoundingRectangles` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::GetEnclosingElement` | fn | 511 |  | `platformRange.getEnclosingElement` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::GetText` | fn | 519 |  | `platformRange.getText` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::Move` | fn | 530 |  | `platformRange.move` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::MoveEndpointByUnit` | fn | 553 |  | `platformRange.moveEndpointByUnit` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::MoveEndpointByRange` | fn | 568 |  | `platformRange.moveEndpointByRange` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::Select` | fn | 593 |  | `platformRange.select` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::AddToSelection` | fn | 605 |  | `platformRange.addToSelection` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::RemoveFromSelection` | fn | 610 |  | `platformRange.removeFromSelection` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::ScrollIntoView` | fn | 615 |  | `platformRange.scrollIntoView` | blocked:M4-windows-a11y |
| `ITextRangeProvider_Impl::GetChildren` | fn | 639 |  | `platformRange.getChildren` | blocked:M4-windows-a11y |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `platform_range_impl_send_sync` | test | 648 |  | `platformRange` | n/a:D-WINSENDSYNC |

## src/adapter.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `focus_event` | fn | 27 |  | `focusEvent` | blocked:M4-windows-a11y |
| `AdapterChangeHandler` | struct | 36 |  | `adapterChangeHandler` | blocked:M4-windows-a11y |
| `AdapterChangeHandler::new` | fn | 44 |  | `newAdapterChangeHandler` | blocked:M4-windows-a11y |
| `AdapterChangeHandler::insert_text_change_if_needed_parent` | fn | 55 |  | `adapterChangeHandler.insertTextChangeIfNeededParent` | blocked:M4-windows-a11y |
| `AdapterChangeHandler::insert_text_change_if_needed` | fn | 78 |  | `adapterChangeHandler.insertTextChangeIfNeeded` | blocked:M4-windows-a11y |
| `AdapterChangeHandler::element_for_possibly_removed_node` | fn | 87 |  | `adapterChangeHandler.elementForPossiblyRemovedNode` | blocked:M4-windows-a11y |
| `AdapterChangeHandler::handle_selection_state_change` | fn | 101 |  | `adapterChangeHandler.handleSelectionStateChange` | blocked:M4-windows-a11y |
| `AdapterChangeHandler::enqueue_selection_changes` | fn | 126 |  | `adapterChangeHandler.enqueueSelectionChanges` | blocked:M4-windows-a11y |
| `INVALIDATE_LIMIT` | const | 180 |  | `invalidateLimit` | blocked:M4-windows-a11y |
| `SelectionChanges` | struct | 233 |  | `selectionChanges` | blocked:M4-windows-a11y |
| `TreeChangeHandler::node_added` | fn | 239 |  | `adapterChangeHandler.NodeAdded` | blocked:M4-windows-a11y |
| `TreeChangeHandler::node_updated` | fn | 270 |  | `adapterChangeHandler.NodeUpdated` | blocked:M4-windows-a11y |
| `TreeChangeHandler::focus_moved` | fn | 342 |  | `adapterChangeHandler.FocusMoved` | blocked:M4-windows-a11y |
| `TreeChangeHandler::node_removed` | fn | 348 |  | `adapterChangeHandler.NodeRemoved` | blocked:M4-windows-a11y |
| `PLACEHOLDER_ROOT_ID` | const | 375 |  | `placeholderRootID` | blocked:M4-windows-a11y |
| `State` | enum | 377 |  | `adapterState` | blocked:M4-windows-a11y |
| `impl Debug for State` | impl | 387 |  | `adapterState.String` | blocked:M4-windows-a11y |
| `Adapter` | struct | 407 |  | `Adapter` | blocked:M4-windows-a11y |
| `Adapter::new` | fn | 44 |  | `NewAdapter` | blocked:M4-windows-a11y |
| `Adapter::with_wrapped_action_handler` | fn | 437 |  | `newAdapterWithWrappedActionHandler` | blocked:M4-windows-a11y |
| `Adapter::update_if_active` | fn | 464 |  | `Adapter.UpdateIfActive` | blocked:M4-windows-a11y |
| `Adapter::update_window_focus_state` | fn | 501 |  | `Adapter.UpdateWindowFocusState` | blocked:M4-windows-a11y |
| `Adapter::handle_wm_getobject` | fn | 535 |  | `Adapter.HandleWmGetobject` | blocked:M4-windows-a11y |
| `init_uia` | fn | 597 |  | `initUia` | blocked:M4-windows-a11y |
| `normalize_objid` | fn | 608 |  | `normalizeObjid` | blocked:M4-windows-a11y |
| `WmGetObjectResult` | struct | 616 |  | `wmGetObjectResult` | blocked:M4-windows-a11y |
| `impl From<WmGetObjectResult> for LRESULT` | impl | 623 |  | `wmGetObjectResult.lresult` | blocked:M4-windows-a11y |
| `QueuedEvents` | struct | 631 |  | `QueuedEvents` | blocked:M4-windows-a11y |
| `QueuedEvents::raise` | fn | 644 |  | `QueuedEvents.Raise` | blocked:M4-windows-a11y |

## src/subclass.rs

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `win32_error` | fn | 19 |  | `win32Error` | blocked:M4-windows-a11y |
| `LongPtr` | type | 26 |  | `longPtr` | blocked:M4-windows-a11y |
| `PROP_NAME` | const | 30 |  | `propName` | blocked:M4-windows-a11y |
| `SubclassState` | struct | 32 |  | `subclassState` | blocked:M4-windows-a11y |
| `SubclassImpl` | struct | 37 |  | `subclassImpl` | blocked:M4-windows-a11y |
| `SubclassImpl::new` | fn | 77 |  | `newSubclassImpl` | blocked:M4-windows-a11y |
| `SubclassImpl::install` | fn | 95 |  | `subclassImpl.install` | blocked:M4-windows-a11y |
| `SubclassImpl::update_window_focus_state` | fn | 118 |  | `subclassImpl.updateWindowFocusState` | blocked:M4-windows-a11y |
| `SubclassImpl::uninstall` | fn | 126 |  | `subclassImpl.uninstall` | blocked:M4-windows-a11y |
| `SubclassingAdapter` | struct | 148 |  | `SubclassingAdapter` | blocked:M4-windows-a11y |
| `SubclassingAdapter::new` | fn | 77 |  | `NewSubclassingAdapter` | blocked:M4-windows-a11y |
| `SubclassingAdapter::update_if_active` | fn | 186 |  | `SubclassingAdapter.UpdateIfActive` | blocked:M4-windows-a11y |
| `impl Drop for SubclassingAdapter` | drop | 200 |  | `SubclassingAdapter.Release` | blocked:M4-windows-a11y |

## src/tests

Go target: `a11y/windows/`, in `simple_test.go`, `subclassed_test.go` and `mod_test.go`.

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests/simple.rs::has_native_uia` | test | 71 |  | `TestSimple_HasNativeUia` | blocked:M4-windows-a11y |
| `tests/simple.rs::navigation` | test | 95 |  | `TestSimple_Navigation` | blocked:M4-windows-a11y |
| `tests/simple.rs::focus` | test | 176 |  | `TestSimple_Focus` | blocked:M4-windows-a11y |
| `tests/subclassed.rs::has_native_uia` | test | 111 |  | `TestSubclassed_HasNativeUia` | blocked:M4-windows-a11y |
| `tests/subclassed.rs::double_instantiate` | test | 170 |  | `TestSubclassed_DoubleInstantiate` | blocked:M4-windows-a11y |
| `tests/mod.rs::scope` | fixture | 188 | test | `scope` | blocked:M4-windows-a11y |
| `tests/mod.rs::FocusEventHandler` | fixture | 304 | test | `focusEventHandler` | blocked:M4-windows-a11y |
| `tests/mod.rs::ReceivedFocusEvent` | fixture | 266 | test | `receivedFocusEvent` | blocked:M4-windows-a11y |
| `tests/mod.rs::create_window` | fixture | 137 | test | `createWindow` | blocked:M4-windows-a11y |

## src/node.rs: added tests

Go target: `a11y/windows/`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `vtables build (added)` | test | 0 |  | `TestNode_PlatformNodeVtablesBuild` | done |
| `control type covers every role (added)` | test | 0 |  | `TestNode_ControlTypeCoversEveryRole` | done |
| `properties and patterns (added)` | test | 0 |  | `TestNode_PropertiesAndPatterns` | done |
| `invoke sends click (added)` | test | 0 |  | `TestNode_InvokeSendsClick` | done |
| `released context is not available (added)` | test | 0 |  | `TestNode_ReleasedContextIsNotAvailable` | done |
| `runtime id from node id (added)` | test | 0 |  | `TestNode_RuntimeIDFromNodeID` | done |
| `platform node reference counting (added)` | test | 0 |  | `TestNode_PlatformNodeReferenceCounting` | done |
| `aria properties separators (added)` | test | 0 |  | `TestUtil_AriaPropertiesSeparators` | done |

## Deviations registered here

| Id | Meaning | Owner |
|---|---|---|
| `D-WINWEAK` | `Weak<Context>` is `weak.Pointer[Context]`, which only fails after garbage collection, so `Context` carries a `released` flag set by `Context.Release`, the drop of the last `Arc<Context>` (it also releases the held platform nodes). | M4-windows-a11y-node |
| `D-WINSTRBUF` | The thread-local `STRING_SCRATCH` pool is a mutex-guarded `stringScratch` (Go has no thread-locals); tests use a fresh pool where upstream uses a fresh thread. Property getters return comparable Go values (`optString`, `wideValue`, `localeName`, `i32Value`, ...) standing in for `StrWrapper` / `WideString` / `LocaleName`; the scratch buffer is used only to encode UTF-16 when a value becomes a VARIANT. | M4-windows-a11y-node |
| `D-WINCOM` | `#[implement]` is one `com.Object` with 13 native vtables built once in `buildPlatformNodeVtables` (thunks from `windows.NewCallback`); the `properties!` / `patterns!` macros are the `nodeProperties` / `nodePatterns` tables walked in the same order. | M4-windows-a11y-node |
| `D-WINFLOAT` | Go callbacks cannot take floating-point arguments, so `ElementProviderFromPoint`, `IRangeValueProvider::SetValue` and (arm64, where `UiaPoint` is an HFA in d0/d1) `ITextProvider::RangeFromPoint` enter through assembly trampolines that move the float registers into integer registers before jumping to the callback. | M4-windows-a11y-node |
| `D-WINSENDSYNC` | `Send` / `Sync` impls and their static assertions have no Go counterpart. | M4-windows-a11y-node |
