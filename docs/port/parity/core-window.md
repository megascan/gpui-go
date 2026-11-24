# Parity: core-window

Scope: gpui: window, element, view, inspector, a11y, native views, prompts. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/element.rs

Go target: `element.go` (package `gpui`), tests in `element_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Element` | trait | 52 |  | `Element` | done |
| `Element::id` | trait-method | 66 |  | `Element.ElementID` | dev:D-NAME |
| `Element::source_location` | trait-method | 70 |  | `Element.SourceLocation` | done |
| `Element::request_layout` | trait-method | 74 |  | `Element.RequestLayout` | done |
| `Element::prepaint` | trait-method | 84 |  | `Element.Prepaint` | done |
| `Element::paint` | trait-method | 96 |  | `Element.Paint` | done |
| `Element::a11y_role` | trait-method | 113 |  | `ElementA11y.A11yRole` | done |
| `Element::write_a11y_info` | trait-method | 121 |  | `ElementA11y.WriteA11yInfo` | done |
| `Element::a11y_synthetic_children` | trait-method | 132 |  | `ElementA11ySyntheticChildren.ElementA11ySyntheticChildren` | dev:D-NAME |
| `Element::into_any` | trait-method | 140 |  | `NewAnyElement` | done |
| `IntoElement` | trait | 154 |  | `IntoElement` | done |
| `IntoElement::into_element` | trait-method | 160 |  | `IntoElement.IntoElement` | n/a:D-INTOELEMENT |
| `IntoElement::into_any_element` | trait-method | 163 |  | `IntoElement.IntoAnyElement` | done |
| `Render` | trait | 172 |  | `Render` | done |
| `Render::render` | trait-method | 174 |  | `Render.Render` | done |
| `RenderOnce` | trait | 188 |  | `RenderOnce` | done |
| `RenderOnce::render` | trait-method | 192 |  | `RenderOnce.Render` | done |
| `ParentElement` | trait | 197 |  | `Parent[S] mixin + ParentElement interface` | done |
| `ParentElement::extend` | trait-method | 199 |  | `ParentElement.Extend` | done |
| `ParentElement::child` | trait-method | 202 |  | `(Parent[S]).Child` | done |
| `ParentElement::children` | trait-method | 211 |  | `(Parent[S]).Children` | done |
| `GlobalElementId` | struct | 228 |  | `GlobalElementID` | done |
| `Drawable` | struct | 324 |  | `Drawable` | done |
| `AnyElement` | struct | 658 |  | `AnyElement` | done |
| `AnyElement::downcast_mut` | method | 677 |  | `(AnyElement).DowncastMut` | done |
| `AnyElement::request_layout` | method | 683 |  | `(AnyElement).RequestLayout` | done |
| `AnyElement::prepaint` | method | 689 |  | `(AnyElement).Prepaint` | dev:D-PREPAINTFOCUS |
| `AnyElement::paint` | method | 702 |  | `(AnyElement).Paint` | done |
| `AnyElement::layout_as_root` | method | 707 |  | `(AnyElement).LayoutAsRoot` | done |
| `AnyElement::prepaint_at` | method | 718 |  | `(AnyElement).PrepaintAt` | dev:D-PREPAINTFOCUS |
| `AnyElement::prepaint_as_root` | method | 729 |  | `(AnyElement).PrepaintAsRoot` | dev:D-PREPAINTFOCUS |
| `Empty` | struct | 795 |  | `Empty` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `equal_paths_are_equal_and_hash_alike` | test | 892 |  | `TestElementGlobalElementIDTests_EqualPathsAreEqualAndHashAlike` | done |
| `frame_cost` | crate::test ignore | 941 |  | `TestElementGlobalElementIDTests_FrameCost` | done |
| `hash_cost` | test ignore | 973 |  | `TestElementGlobalElementIDTests_HashCost` | done |

## gpui/src/inspector.rs

Go target: `inspector.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `InspectorElementId` | struct | 3 |  | `InspectorElementID` | done |
| `InspectorElementPath` | struct | 30 | feat:inspector,debug | `InspectorElementPath` | done |
| `InspectorRenderer` | type | 55 | feat:inspector,debug | `InspectorRenderer` | done |
| `Inspector` | struct | 60 | feat:inspector,debug | `Inspector` | done |
| `Inspector::active_element_id` | method | 117 | feat:inspector,debug | `(Inspector).ActiveElementID` | done |
| `Inspector::start_picking` | method | 148 | feat:inspector,debug | `(Inspector).StartPicking` | done |
| `Inspector::is_picking` | method | 153 | feat:inspector,debug | `(Inspector).IsPicking` | done |
| `Inspector::render_inspector_states` | method | 158 | feat:inspector,debug | `(Inspector).RenderInspectorStates` | dev:D-INSPECTORORDER |
| `FunctionReflection` | struct | 250 | feat:inspector,debug | `FunctionReflection` | done |
| `FunctionReflection::invoke` | method | 263 | feat:inspector,debug | `(FunctionReflection).Invoke` | done |

## gpui/src/view.rs

Go target: `view.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AnyView` | struct | 24 |  | `AnyView` | done |
| `AnyView::cached` | method | 46 |  | `(AnyView).Cached` | done |
| `AnyView::downgrade` | method | 51 |  | `(AnyView).Downgrade` | done |
| `AnyView::downcast` | method | 61 |  | `(AnyView).Downcast` | done |
| `AnyView::entity_type` | method | 73 |  | `(AnyView).EntityType` | done |
| `AnyView::entity_id` | method | 78 |  | `(AnyView).EntityID` | done |
| `AnyWeakView` | struct | 135 |  | `AnyWeakView` | done |
| `AnyWeakView::upgrade` | method | 143 |  | `(AnyWeakView).Upgrade` | done |
| `View` | trait | 208 |  | `View` | dev:D-VIEWADAPT |
| `View::entity_id` | trait-method | 217 |  | `View.EntityID` | done |
| `View::render` | trait-method | 220 |  | `View.Render` | done |
| `Entity::cached` | method | 258 |  | `(Entity).Cached` | done |
| `ViewElement` | struct | 266 |  | `ViewElement` | dev:D-VIEWADAPT |
| `ViewElement::new` | assoc-fn | 279 |  | `NewViewElement` | done |
| `EmptyView` | struct | 470 |  | `EmptyView` | done |

## gpui/src/window.rs

Go target: `window.go` (package `gpui`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `DEFAULT_WINDOW_SIZE` | const | 88 |  | `DefaultWindowSize` | done |
| `DEFAULT_ADDITIONAL_WINDOW_SIZE` | const | 92 |  | `DefaultAdditionalWindowSize` | done |
| `DispatchPhase` | enum | 99 |  | `DispatchPhase` | done |
| `DispatchPhase::bubble` | method | 117 |  | `(DispatchPhase).Bubble` | done |
| `DispatchPhase::capture` | method | 123 |  | `(DispatchPhase).Capture` | done |
| `FocusOutEvent` | struct | 328 |  | `FocusOutEvent` | done |
| `FocusId` | struct | 335 |  | `FocusID` | done |
| `ArenaClearNeeded` | struct | 464 |  | `ArenaClearNeeded` | dev:D-ARENA |
| `ArenaClearNeeded::clear` | method | 487 |  | `(ArenaClearNeeded).Clear` | dev:D-ARENA |
| `FocusId::is_focused` | method | 510 |  | `(FocusID).IsFocused` | done |
| `FocusId::contains_focused` | method | 516 |  | `(FocusID).ContainsFocused` | done |
| `FocusId::within_focused` | method | 524 |  | `(FocusID).WithinFocused` | done |
| `FocusHandle` | struct | 539 |  | `FocusHandle` | done |
| `FocusHandle::tab_index` | method | 585 |  | `(FocusHandle).WithTabIndex` | dev:D-FIELDMETHOD |
| `FocusHandle::tab_stop` | method | 596 |  | `(FocusHandle).WithTabStop` | dev:D-FIELDMETHOD |
| `FocusHandle::downgrade` | method | 605 |  | `(FocusHandle).Downgrade` | done |
| `FocusHandle::focus` | method | 613 |  | `(FocusHandle).Focus` | done |
| `FocusHandle::is_focused` | method | 618 |  | `(FocusHandle).IsFocused` | done |
| `FocusHandle::contains_focused` | method | 624 |  | `(FocusHandle).ContainsFocused` | done |
| `FocusHandle::within_focused` | method | 630 |  | `(FocusHandle).WithinFocused` | done |
| `FocusHandle::contains` | method | 635 |  | `(FocusHandle).Contains` | done |
| `FocusHandle::is_rendered` | method | 640 |  | `(FocusHandle).IsRendered` | done |
| `FocusHandle::dispatch_action` | method | 649 |  | `(FocusHandle).DispatchAction` | done |
| `WeakFocusHandle` | struct | 690 |  | `WeakFocusHandle` | done |
| `WeakFocusHandle::upgrade` | method | 697 |  | `(WeakFocusHandle).Upgrade` | done |
| `Focusable` | trait | 725 |  | `Focusable` | done |
| `Focusable::focus_handle` | trait-method | 727 |  | `Focusable.FocusHandle` | done |
| `ManagedView` | trait | 738 |  | `ManagedView` | done |
| `DismissEvent` | struct | 743 |  | `DismissEvent` | done |
| `WindowControlArea` | enum | 764 |  | `WindowControlArea` | done |
| `HitboxId` | struct | 777 |  | `HitboxID` | done |
| `HitboxId::placeholder` | assoc-fn | 785 | feat:test-support | `HitboxIDPlaceholder` | done |
| `HitboxId::is_hovered` | method | 797 |  | `(HitboxID).IsHovered` | done |
| `HitboxId::should_handle_scroll` | method | 834 |  | `(HitboxID).ShouldHandleScroll` | done |
| `Hitbox` | struct | 846 |  | `Hitbox` | done |
| `Hitbox::is_hovered` | method | 875 |  | `(Hitbox).IsHovered` | done |
| `Hitbox::is_hovered_at` | method | 881 |  | `(Hitbox).IsHoveredAt` | done |
| `Hitbox::should_handle_scroll` | method | 896 |  | `(Hitbox).ShouldHandleScroll` | done |
| `HitboxBehavior` | enum | 903 |  | `HitboxBehavior` | done |
| `TooltipId` | struct | 962 |  | `TooltipID` | done |
| `TooltipId::is_hovered` | method | 966 |  | `(TooltipID).IsHovered` | done |
| `Window` | struct | 1232 |  | `Window` | done |
| `PendingInputStatus` | struct | 1452 |  | `PendingInputStatus` | done |
| `PendingInputStatus::keystrokes` | method | 1459 |  | `(PendingInputStatus).Keystrokes` | done |
| `PendingInputStatus::timeout` | method | 1464 |  | `(PendingInputStatus).Timeout` | done |
| `PendingInputTimeoutStatus` | struct | 1471 |  | `PendingInputTimeoutStatus` | done |
| `PendingInputTimeoutStatus::duration` | method | 1480 |  | `(PendingInputTimeoutStatus).Duration` | done |
| `PendingInputTimeoutStatus::remaining` | method | 1485 |  | `(PendingInputTimeoutStatus).Remaining` | done |
| `PendingInputTimeoutStatus::is_paused` | method | 1495 |  | `(PendingInputTimeoutStatus).IsPaused` | done |
| `DispatchEventResult` | struct | 2532 |  | `DispatchEventResult` | done |
| `ContentMask` | struct | 2542 |  | `ContentMask` | done |
| `ContentMask<Pixels>::scale` | method | 2549 |  | `(ContentMask).Scale` | done |
| `ContentMask<Pixels>::intersect` | method | 2556 |  | `(ContentMask).Intersect` | done |
| `Window::visibility` | method | 2654 |  | `(Window).Visibility` | done |
| `Window::is_visible` | method | 2663 |  | `(Window).IsVisible` | done |
| `Window::observe_window_visibility` | method | 2668 |  | `(Window).ObserveWindowVisibility` | done |
| `Window::observe_window_appearance` | method | 2680 |  | `(Window).ObserveWindowAppearance` | done |
| `Window::observe_button_layout_changed` | method | 2692 |  | `(Window).ObserveButtonLayoutChanged` | done |
| `Window::replace_root` | method | 2704 |  | `(Window).ReplaceRoot` | done |
| `Window::root` | method | 2719 |  | `(Window).Root` | done |
| `Window::window_handle` | method | 2729 |  | `(Window).WindowHandle` | done |
| `Window::refresh` | method | 2734 |  | `(Window).Refresh` | done |
| `Window::remove_window` | method | 2742 |  | `(Window).RemoveWindow` | done |
| `Window::focused` | method | 2747 |  | `(Window).Focused` | done |
| `Window::focus_lost_restore_target` | method | 2755 |  | `(Window).FocusLostRestoreTarget` | done |
| `Window::focus` | method | 2764 |  | `(Window).Focus` | done |
| `Window::blur` | method | 2779 |  | `(Window).Blur` | done |
| `Window::disable_focus` | method | 2795 |  | `(Window).DisableFocus` | done |
| `Window::focus_next` | method | 2801 |  | `(Window).FocusNext` | done |
| `Window::focus_prev` | method | 2812 |  | `(Window).FocusPrev` | done |
| `Window::text_system` | method | 2823 |  | `(Window).TextSystem` | done |
| `Window::text_style` | method | 2828 |  | `(Window).TextStyle` | done |
| `Window::is_maximized` | method | 2839 |  | `(Window).IsMaximized` | done |
| `Window::request_decorations` | method | 2844 |  | `(Window).RequestDecorations` | done |
| `Window::set_exclusive_zone` | method | 2853 |  | `(Window).SetExclusiveZone` | done |
| `Window::set_exclusive_edge` | method | 2862 | linux,feat:wayland | `(Window).SetExclusiveEdge` | done |
| `Window::start_window_resize` | method | 2867 |  | `(Window).StartWindowResize` | done |
| `Window::set_input_region` | method | 2879 |  | `(Window).SetInputRegion` | done |
| `Window::window_bounds` | method | 2885 |  | `(Window).WindowBounds` | done |
| `Window::inner_window_bounds` | method | 2898 |  | `(Window).InnerWindowBounds` | done |
| `Window::native_window_state` | method | 2904 |  | `(Window).NativeWindowState` | done |
| `Window::restore_native_window_state` | method | 2910 |  | `(Window).RestoreNativeWindowState` | done |
| `Window::dispatch_action` | method | 2915 |  | `(Window).DispatchAction` | done |
| `Window::defer` | method | 2975 |  | `(Window).Defer` | done |
| `Window::observe` | method | 2985 |  | `(Window).Observe` | done |
| `Window::subscribe` | method | 3014 |  | `(Window).Subscribe` | done |
| `Window::observe_release` | method | 3049 |  | `(Window).ObserveRelease` | done |
| `Window::to_async` | method | 3071 |  | `(Window).ToAsync` | done |
| `Window::on_next_frame` | method | 3076 |  | `(Window).OnNextFrame` | done |
| `Window::request_animation_frame` | method | 3096 |  | `(Window).RequestAnimationFrame` | done |
| `Window::simulate_next_frame` | method | 3111 | feat:test-support | `(Window).SimulateNextFrame` | done |
| `Window::spawn` | method | 3124 |  | `(Window).Spawn` | done |
| `Window::spawn_with_priority` | method | 3140 |  | `(Window).SpawnWithPriority` | done |
| `Window::bounds_changed` | method | 3162 |  | `(Window).BoundsChanged` | done |
| `Window::refresh_interval` | method | 3224 |  | `(Window).RefreshInterval` | done |
| `Window::observe_window_display` | method | 3232 |  | `(Window).ObserveWindowDisplay` | done |
| `Window::bounds` | method | 3243 |  | `(Window).Bounds` | done |
| `Window::insets` | method | 3248 |  | `(Window).Insets` | done |
| `Window::render_to_image` | method | 3256 | feat:test-support | `(Window).RenderToImage` | done |
| `Window::rendered_scene` | method | 3264 | feat:bench-support | `(Window).RenderedScene` | done |
| `Window::painted_quads` | method | 3273 | feat:test-support | `(Window).PaintedQuads` | done |
| `Window::painted_underlines` | method | 3279 | feat:test-support | `(Window).PaintedUnderlines` | done |
| `Window::resize` | method | 3284 |  | `(Window).Resize` | done |
| `Window::is_fullscreen` | method | 3289 |  | `(Window).IsFullscreen` | done |
| `Window::is_simple_fullscreen` | method | 3296 |  | `(Window).IsSimpleFullscreen` | done |
| `Window::appearance` | method | 3315 |  | `(Window).Appearance` | done |
| `Window::viewport_size` | method | 3320 |  | `(Window).ViewportSize` | done |
| `Window::visual_viewport_bounds` | method | 3329 |  | `(Window).VisualViewportBounds` | done |
| `Window::fully_visible_bounds` | method | 3337 |  | `(Window).FullyVisibleBounds` | done |
| `Window::request_virtual_keyboard` | method | 3355 |  | `(Window).RequestVirtualKeyboard` | done |
| `Window::dismiss_virtual_keyboard` | method | 3360 |  | `(Window).DismissVirtualKeyboard` | done |
| `Window::is_window_active` | method | 3365 |  | `(Window).IsWindowActive` | done |
| `Window::is_window_hovered` | method | 3372 |  | `(Window).IsWindowHovered` | done |
| `Window::zoom_window` | method | 3385 |  | `(Window).ZoomWindow` | done |
| `Window::show_window_menu` | method | 3390 |  | `(Window).ShowWindowMenu` | done |
| `Window::start_window_move` | method | 3398 |  | `(Window).StartWindowMove` | done |
| `Window::set_client_inset` | method | 3403 |  | `(Window).SetClientInset` | done |
| `Window::set_client_corner_radii` | method | 3409 |  | `(Window).SetClientCornerRadii` | done |
| `Window::client_inset` | method | 3414 |  | `(Window).ClientInset` | done |
| `Window::window_decorations` | method | 3419 |  | `(Window).WindowDecorations` | done |
| `Window::is_resizable` | method | 3424 |  | `(Window).IsResizable` | done |
| `Window::is_minimizable` | method | 3429 |  | `(Window).IsMinimizable` | done |
| `Window::window_controls` | method | 3434 |  | `(Window).WindowControls` | done |
| `Window::set_window_title` | method | 3439 |  | `(Window).SetWindowTitle` | done |
| `Window::set_traffic_light_position` | method | 3446 | macos | `(Window).SetTrafficLightPosition` | done |
| `Window::set_native_titlebar_visibility` | method | 3452 | macos | `(Window).SetNativeTitlebarVisibility` | done |
| `Window::set_app_id` | method | 3457 |  | `(Window).SetAppID` | done |
| `Window::set_background_appearance` | method | 3462 |  | `(Window).SetBackgroundAppearance` | done |
| `Window::set_window_edited` | method | 3468 |  | `(Window).SetWindowEdited` | done |
| `Window::set_document_path` | method | 3474 |  | `(Window).SetDocumentPath` | done |
| `Window::display` | method | 3479 |  | `(Window).Display` | done |
| `Window::show_character_palette` | method | 3487 |  | `(Window).ShowCharacterPalette` | done |
| `Window::scale_factor` | method | 3494 |  | `(Window).ScaleFactor` | done |
| `Window::set_scale_factor` | method | 3500 | feat:test-support | `(Window).SetScaleFactor` | done |
| `Window::rem_size` | method | 3507 |  | `(Window).RemSize` | done |
| `Window::set_rem_size` | method | 3516 |  | `(Window).SetRemSize` | done |
| `Window::with_global_id` | method | 3522 |  | `(Window).WithGlobalID` | done |
| `Window::with_id` | method | 3536 |  | `(Window).WithID` | done |
| `Window::with_rem_size` | method | 3553 |  | `(Window).WithRemSize` | done |
| `Window::line_height` | method | 3570 |  | `(Window).LineHeight` | done |
| `Window::pixel_snap` | method | 3576 |  | `(Window).PixelSnap` | done |
| `Window::pixel_snap_f64` | method | 3582 |  | `(Window).PixelSnapF64` | done |
| `Window::pixel_snap_bounds` | method | 3589 |  | `(Window).PixelSnapBounds` | done |
| `Window::pixel_snap_point` | method | 3595 |  | `(Window).PixelSnapPoint` | done |
| `Window::underline_bounds` | method | 3600 |  | `(Window).UnderlineBounds` | done |
| `Window::paint_underline_with_exclusions` | method | 3620 |  | `(Window).PaintUnderlineWithExclusions` | dev:D-SPRANGE |
| `Window::prevent_default` | method | 3748 |  | `(Window).PreventDefault` | done |
| `Window::default_prevented` | method | 3753 |  | `(Window).DefaultPrevented` | done |
| `Window::is_action_available` | method | 3758 |  | `(Window).IsActionAvailable` | done |
| `Window::is_action_available_in` | method | 3767 |  | `(Window).IsActionAvailableIn` | done |
| `Window::mouse_position` | method | 3775 |  | `(Window).MousePosition` | done |
| `Window::capture_pointer` | method | 3785 |  | `(Window).CapturePointer` | done |
| `Window::release_pointer` | method | 3790 |  | `(Window).ReleasePointer` | done |
| `Window::captured_hitbox` | method | 3795 |  | `(Window).CapturedHitbox` | done |
| `Window::capture_long_press` | method | 3804 |  | `(Window).CaptureLongPress` | done |
| `Window::has_long_press_capture` | method | 3809 |  | `(Window).HasLongPressCapture` | done |
| `Window::modifiers` | method | 3814 |  | `(Window).Modifiers` | done |
| `Window::last_input_was_keyboard` | method | 3820 |  | `(Window).LastInputWasKeyboard` | done |
| `Window::last_input_was_touch` | method | 3825 |  | `(Window).LastInputWasTouch` | done |
| `Window::capslock` | method | 3830 |  | `(Window).Capslock` | done |
| `Window::draw` | method | 3837 |  | `(Window).Draw` | done |
| `Window::input_latency_snapshot` | method | 4045 | feat:profiler | `(Window).InputLatencySnapshot` | done |
| `Window::frame_duration_snapshot` | method | 4051 | feat:profiler | `(Window).FrameDurationSnapshot` | done |
| `Window::debug_frame_overlay_mode` | method | 4057 | feat:profiler | `(Window).DebugFrameOverlayMode` | done |
| `Window::set_debug_frame_overlay_mode` | method | 4063 | feat:profiler | `(Window).SetDebugFrameOverlayMode` | done |
| `Window::cycle_debug_frame_overlay_mode` | method | 4071 | feat:profiler | `(Window).CycleDebugFrameOverlayMode` | done |
| `Window::reset_debug_frame_overlay_stats` | method | 4078 | feat:profiler | `(Window).ResetDebugFrameOverlayStats` | done |
| `Window::with_text_style` | method | 4538 |  | `(Window).WithTextStyle` | done |
| `Window::set_cursor_style` | method | 4555 |  | `(Window).SetCursorStyle` | done |
| `Window::set_window_cursor_style` | method | 4567 |  | `(Window).SetWindowCursorStyle` | done |
| `Window::set_tooltip` | method | 4577 |  | `(Window).SetTooltip` | done |
| `Window::with_content_mask` | method | 4591 |  | `(Window).WithContentMask` | done |
| `Window::with_element_offset` | method | 4610 |  | `(Window).WithElementOffset` | done |
| `Window::with_absolute_element_offset` | method | 4628 |  | `(Window).WithAbsoluteElementOffset` | done |
| `Window::transact` | method | 4663 |  | `(Window).Transact` | done |
| `Window::request_autoscroll` | method | 4691 |  | `(Window).RequestAutoscroll` | done |
| `Window::take_autoscroll` | method | 4698 |  | `(Window).TakeAutoscroll` | done |
| `Window::use_asset` | method | 4708 |  | `(Window).UseAsset` | done |
| `Window::get_asset` | method | 4717 |  | `(Window).GetAsset` | done |
| `Window::element_offset` | method | 4722 |  | `(Window).ElementOffset` | done |
| `Window::content_mask` | method | 4739 |  | `(Window).ContentMask` | done |
| `Window::with_element_namespace` | method | 4754 |  | `(Window).WithElementNamespace` | done |
| `Window::use_keyed_state` | method | 4766 |  | `(Window).UseKeyedState` | done |
| `Window::use_state` | method | 4792 |  | `(Window).UseState` | dev:D-SRCLOC |
| `Window::with_element_state` | method | 4809 |  | `(Window).WithElementState` | done |
| `Window::with_optional_element_state` | method | 4884 |  | `(Window).WithOptionalElementState` | done |
| `Window::with_tab_group` | method | 4913 |  | `(Window).WithTabGroup` | done |
| `Window::defer_draw` | method | 4932 |  | `(Window).DeferDraw` | done |
| `Window::paint_layer` | method | 4965 |  | `(Window).PaintLayer` | done |
| `Window::paint_drop_shadows` | method | 4979 |  | `(Window).PaintDropShadows` | done |
| `Window::paint_inset_shadows` | method | 5019 |  | `(Window).PaintInsetShadows` | done |
| `Window::paint_quad` | method | 5112 |  | `(Window).PaintQuad` | done |
| `Window::paint_path` | method | 5224 |  | `(Window).PaintPath` | done |
| `Window::paint_underline` | method | 5242 |  | `(Window).PaintUnderline` | done |
| `Window::paint_strikethrough` | method | 5256 |  | `(Window).PaintStrikethrough` | done |
| `Window::paint_glyph` | method | 5291 |  | `(Window).PaintGlyph` | done |
| `Window::paint_emoji` | method | 5398 |  | `(Window).PaintEmoji` | done |
| `Window::paint_svg` | method | 5455 |  | `(Window).PaintSvg` | done |
| `Window::paint_image` | method | 5547 |  | `(Window).PaintImage` | done |
| `Window::paint_surface` | method | 5649 | macos,ios | `(Window).PaintSurface` | done |
| `Window::drop_image` | method | 5669 |  | `(Window).DropImage` | done |
| `Window::has_image_atlas_entry` | method | 5684 | feat:test-support | `(Window).HasImageAtlasEntry` | done |
| `Window::request_layout` | method | 5703 |  | `(Window).RequestLayout` | done |
| `Window::request_measured_layout` | method | 5732 |  | `(Window).RequestMeasuredLayout` | done |
| `Window::compute_layout` | method | 5752 |  | `(Window).ComputeLayout` | done |
| `Window::layout_bounds` | method | 5769 |  | `(Window).LayoutBounds` | done |
| `Window::insert_hitbox` | method | 5789 |  | `(Window).InsertHitbox` | done |
| `Window::insert_window_control_hitbox` | method | 5808 |  | `(Window).InsertWindowControlHitbox` | done |
| `Window::set_key_context` | method | 5817 |  | `(Window).SetKeyContext` | done |
| `Window::set_focus_handle` | method | 5826 |  | `(Window).SetFocusHandle` | done |
| `Window::set_view_id` | method | 5839 |  | `(Window).SetViewID` | done |
| `Window::current_view` | method | 5845 |  | `(Window).CurrentView` | done |
| `Window::with_image_cache` | method | 5863 |  | `(Window).WithImageCache` | done |
| `Window::handle_input` | method | 5885 |  | `(Window).HandleInput` | done |
| `Window::notify_text_input_changed` | method | 5905 |  | `(Window).NotifyTextInputChanged` | done |
| `Window::notify_text_selection_changed` | method | 5916 |  | `(Window).NotifyTextSelectionChanged` | done |
| `Window::on_mouse_event` | method | 5948 |  | `(Window).OnMouseEvent` | done |
| `Window::on_key_event` | method | 5971 |  | `(Window).OnKeyEvent` | done |
| `Window::on_modifiers_changed` | method | 5992 |  | `(Window).OnModifiersChanged` | done |
| `Window::on_focus_in` | method | 6008 |  | `(Window).OnFocusIn` | done |
| `Window::on_focus_out` | method | 6028 |  | `(Window).OnFocusOut` | done |
| `Window::dispatch_keystroke` | method | 6068 |  | `(Window).DispatchKeystroke` | done |
| `Window::keystroke_text_for` | method | 6095 |  | `(Window).KeystrokeTextFor` | done |
| `Window::dispatch_event` | method | 6110 |  | `(Window).DispatchEvent` | done |
| `Window::touch_prediction_enabled` | method | 6317 |  | `(Window).TouchPredictionEnabled` | done |
| `Window::set_touch_prediction_enabled` | method | 6324 |  | `(Window).SetTouchPredictionEnabled` | done |
| `Window::has_pending_keystrokes` | method | 6852 |  | `(Window).HasPendingKeystrokes` | done |
| `Window::pending_input` | method | 6869 |  | `(Window).PendingInput` | done |
| `Window::set_pending_input_timeout_paused` | method | 6886 |  | `(Window).SetPendingInputTimeoutPaused` | done |
| `Window::pending_input_keystrokes` | method | 6959 |  | `(Window).PendingInputKeystrokes` | done |
| `Window::observe_global` | method | 7146 |  | `(Window).ObserveGlobal` | done |
| `Window::activate_window` | method | 7165 |  | `(Window).ActivateWindow` | done |
| `Window::request_attention` | method | 7170 |  | `(Window).RequestAttention` | done |
| `Window::minimize_window` | method | 7175 |  | `(Window).MinimizeWindow` | done |
| `Window::toggle_fullscreen` | method | 7180 |  | `(Window).ToggleFullscreen` | done |
| `Window::toggle_simple_fullscreen` | method | 7188 |  | `(Window).ToggleSimpleFullscreen` | done |
| `Window::invalidate_character_coordinates` | method | 7193 |  | `(Window).InvalidateCharacterCoordinates` | done |
| `Window::prompt` | method | 7207 |  | `(Window).Prompt` | done |
| `Window::has_active_prompt` | method | 7265 |  | `(Window).HasActivePrompt` | done |
| `Window::context_stack` | method | 7270 |  | `(Window).ContextStack` | done |
| `Window::available_actions` | method | 7281 |  | `(Window).AvailableActions` | done |
| `Window::bindings_for_action` | method | 7297 |  | `(Window).BindingsForAction` | done |
| `Window::highest_precedence_binding_for_action` | method | 7305 |  | `(Window).HighestPrecedenceBindingForAction` | done |
| `Window::bindings_for_action_in_context` | method | 7315 |  | `(Window).BindingsForActionInContext` | done |
| `Window::highest_precedence_binding_for_action_in_context` | method | 7326 |  | `(Window).HighestPrecedenceBindingForActionInContext` | done |
| `Window::bindings_for_action_in` | method | 7338 |  | `(Window).BindingsForActionIn` | done |
| `Window::highest_precedence_binding_for_action_in` | method | 7353 |  | `(Window).HighestPrecedenceBindingForActionIn` | done |
| `Window::possible_bindings_for_input` | method | 7364 |  | `(Window).PossibleBindingsForInput` | done |
| `Window::listener_for` | method | 7385 |  | `(Window).ListenerFor` | done |
| `Window::handler_for` | method | 7397 |  | `(Window).HandlerFor` | done |
| `Window::on_window_should_close` | method | 7410 |  | `(Window).OnWindowShouldClose` | done |
| `Window::on_action` | method | 7429 |  | `(Window).OnAction` | done |
| `Window::on_action_when` | method | 7449 |  | `(Window).OnActionWhen` | done |
| `Window::gpu_specs` | method | 7466 |  | `(Window).GPUSpecs` | done |
| `Window::titlebar_double_click` | method | 7472 |  | `(Window).TitlebarDoubleClick` | done |
| `Window::window_title` | method | 7479 |  | `(Window).WindowTitle` | done |
| `Window::tabbed_windows` | method | 7485 |  | `(Window).TabbedWindows` | done |
| `Window::tab_bar_visible` | method | 7491 |  | `(Window).TabBarVisible` | done |
| `Window::merge_all_windows` | method | 7497 |  | `(Window).MergeAllWindows` | done |
| `Window::move_tab_to_new_window` | method | 7503 |  | `(Window).MoveTabToNewWindow` | done |
| `Window::toggle_window_tab_overview` | method | 7509 |  | `(Window).ToggleWindowTabOverview` | done |
| `Window::set_tabbing_identifier` | method | 7515 |  | `(Window).SetTabbingIdentifier` | done |
| `Window::play_system_bell` | method | 7522 |  | `(Window).PlaySystemBell` | done |
| `Window::is_a11y_enabled` | method | 7532 |  | `(Window).IsA11yEnabled` | done |
| `Window::is_a11y_active` | method | 7546 |  | `(Window).IsA11yActive` | done |
| `Window::debug_a11y_tree_json` | method | 7551 |  | `(Window).DebugA11yTreeJSON` | done |
| `Window::on_a11y_action` | method | 7560 |  | `(Window).OnA11yAction` | done |
| `Window::toggle_inspector` | method | 7638 | feat:inspector,debug | `(Window).ToggleInspector` | done |
| `Window::is_inspector_picking` | method | 7653 |  | `(Window).IsInspectorPicking` | done |
| `Window::with_inspector_state` | method | 7665 | feat:inspector,debug | `(Window).WithInspectorState` | done |
| `Window::insert_inspector_hitbox` | method | 7731 | feat:inspector,debug | `(Window).InsertInspectorHitbox` | done |
| `Window::set_modifiers` | method | 7834 | feat:test-support | `(Window).SetModifiers` | done |
| `Window::simulate_mouse_move` | method | 7842 | feat:test-support | `(Window).SimulateMouseMove` | done |
| `WindowId` | struct | 7888 |  | `WindowID` | done |
| `WindowId::as_u64` | method | 7893 |  | `(WindowID).AsU64` | done |
| `WindowHandle` | struct | 7907 |  | `WindowHandle` | done |
| `WindowHandle::new` | assoc-fn | 7925 |  | `NewWindowHandle` | done |
| `WindowHandle::root` | method | 7940 | feat:test-support | `(WindowHandle).Root` | done |
| `WindowHandle::update` | method | 7955 |  | `(WindowHandle).Update` | done |
| `WindowHandle::read` | method | 7975 |  | `(WindowHandle).Read` | done |
| `WindowHandle::read_with` | method | 7994 |  | `(WindowHandle).ReadWith` | done |
| `WindowHandle::entity` | method | 8004 |  | `(WindowHandle).Entity` | done |
| `WindowHandle::is_active` | method | 8015 |  | `(WindowHandle).IsActive` | done |
| `AnyWindowHandle` | struct | 8051 |  | `AnyWindowHandle` | done |
| `AnyWindowHandle::window_id` | method | 8059 |  | `(AnyWindowHandle).WindowID` | done |
| `AnyWindowHandle::root_entity_type_name` | method | 8064 |  | `(AnyWindowHandle).RootEntityTypeName` | done |
| `AnyWindowHandle::downcast` | method | 8070 |  | `(AnyWindowHandle).Downcast` | done |
| `AnyWindowHandle::update` | method | 8085 |  | `(AnyWindowHandle).Update` | done |
| `AnyWindowHandle::read` | method | 8099 |  | `(AnyWindowHandle).Read` | done |
| `ElementId` | enum | 8131 |  | `ElementID` | done |
| `ElementId::named_usize` | assoc-fn | 8156 |  | `ElementIDNamedUsize` | done |
| `PaintQuad` | struct | 8303 |  | `PaintQuad` | done |
| `PaintQuad::corner_radii` | method | 8320 |  | `(PaintQuad).WithCornerRadii` | dev:D-FIELDMETHOD |
| `PaintQuad::border_widths` | method | 8328 |  | `(PaintQuad).WithBorderWidths` | dev:D-FIELDMETHOD |
| `PaintQuad::border_color` | method | 8336 |  | `(PaintQuad).WithBorderColor` | dev:D-FIELDMETHOD |
| `PaintQuad::background` | method | 8344 |  | `(PaintQuad).WithBackground` | dev:D-FIELDMETHOD |
| `quad` | fn | 8353 |  | `NewQuad` | done |
| `fill` | fn | 8372 |  | `NewFill` | done |
| `outline` | fn | 8384 |  | `Outline` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_focus_handle_release` | gpui::test | 8434 |  | `TestWindow_FocusHandleRelease` | done |
| `test_dropped_focus_handle_blurs_only_its_windows` | gpui::test | 8475 |  | `TestWindow_DroppedFocusHandleBlursOnlyItsWindows` | done |
| `test_native_close_defers_app_access` | gpui::test | 8523 |  | `TestWindow_NativeCloseDefersAppAccess` | done |
| `test_frame_requests_and_skips_are_journaled` | gpui::test | 8537 | feat:profiler | `TestWindow_FrameRequestsAndSkipsAreJournaled` | done |
| `test_frame_demand_survives_throttling` | gpui::test | 8593 | feat:profiler | `TestWindow_FrameDemandSurvivesThrottling` | done |
| `test_debug_state_reports_cached_and_platform_values` | gpui::test | 8700 |  | `TestWindow_DebugStateReportsCachedAndPlatformValues` | done |
| `test_window_visibility` | gpui::test | 8729 |  | `TestWindow_WindowVisibility` | done |
| `test_display_tracking` | gpui::test | 8774 |  | `TestWindow_DisplayTracking` | done |
| `test_window_opened_from_bounds_observer_uses_new_bounds` | gpui::test | 8884 |  | `TestWindow_WindowOpenedFromBoundsObserverUsesNewBounds` | done |
| `test_fully_visible_bounds_preserve_layout_viewport` | gpui::test | 8926 |  | `TestWindow_FullyVisibleBoundsPreserveLayoutViewport` | done |
| `frame_throttle_tracks_display_refreshes` | test | 9014 |  | `TestWindow_FrameThrottleTracksDisplayRefreshes` | done |
| `frame_throttle_preserves_cadence_without_catching_up_after_stalls` | test | 9028 |  | `TestWindow_FrameThrottlePreservesCadenceWithoutCatchingUpAfterStalls` | done |
| `power_modes_preserve_frame_limits` | test | 9050 |  | `TestWindow_PowerModesPreserveFrameLimits` | done |
| `frame_limits_combine_enabled_intervals` | test | 9102 |  | `TestWindow_FrameLimitsCombineEnabledIntervals` | done |
| `test_window_opened_during_draw_defers_arena_clear` | test | 9180 |  | `TestWindow_WindowOpenedDuringDrawDefersArenaClear` | done |
| `test_scale_factor_change_preserves_bounds_and_survives_resize` | test | 9200 |  | `TestWindow_ScaleFactorChangePreservesBoundsAndSurvivesResize` | done |
| `test_fullscreen_toggle_reports_current_bounds` | gpui::test | 9239 |  | `TestWindow_FullscreenToggleReportsCurrentBounds` | done |
| `test_frame_waker_fires_on_frame_demand` | gpui::test | 9287 |  | `TestWindow_FrameWakerFiresOnFrameDemand` | done |
| `test_pending_next_frame_callbacks_are_not_stranded` | gpui::test | 9347 |  | `TestWindow_PendingNextFrameCallbacksAreNotStranded` | done |
| `test_window_reports_no_raw_handle_instead_of_panicking` | gpui::test | 9379 |  | `TestWindow_WindowReportsNoRawHandleInsteadOfPanicking` | done |
| `test_appearance_change_runs_after_app_update` | gpui::test | 9398 |  | `TestWindow_AppearanceChangeRunsAfterAppUpdate` | done |
| `queued_frame_callback_wakes_a_parked_render_loop` | gpui::test | 9423 |  | `TestWindow_QueuedFrameCallbackWakesAParkedRenderLoop` | done |
| `pending_presentation_wakes_a_parked_render_loop` | gpui::test | 9451 |  | `TestWindow_PendingPresentationWakesAParkedRenderLoop` | done |
| `callback_queued_during_a_frame_requests_a_follow_up` | gpui::test | 9469 |  | `TestWindow_CallbackQueuedDuringAFrameRequestsAFollowUp` | done |
| `auto_sized_window_root_fills_the_window` | test | 9520 |  | `TestWindow_AutoSizedWindowRootFillsTheWindow` | done |
| `explicitly_sized_window_root_keeps_its_size` | test | 9542 |  | `TestWindow_ExplicitlySizedWindowRootKeepsItsSize` | done |
| `file_drag_is_promoted_once_and_restored_in_source_window` | gpui::test | 9604 |  | `TestWindow_FileDragIsPromotedOnceAndRestoredInSourceWindow` | done |
| `test_focus_moved_by_focus_listener_is_dispatched` | gpui::test | 9948 |  | `TestWindow_FocusMovedByFocusListenerIsDispatched` | done |
| `claimed_touch_drag_receives_movement_and_release` | gpui::test | 9990 |  | `TestWindow_ClaimedTouchDragReceivesMovementAndRelease` | done |
| `back_gesture_is_claimed_by_the_innermost_listener` | gpui::test | 10013 |  | `TestWindow_BackGestureIsClaimedByTheInnermostListener` | done |
| `long_press_is_claimed_only_when_started_prevents_default` | gpui::test | 10089 |  | `TestWindow_LongPressIsClaimedOnlyWhenStartedPreventsDefault` | done |
| `stale_default_prevention_does_not_claim_long_press` | gpui::test | 10132 |  | `TestWindow_StaleDefaultPreventionDoesNotClaimLongPress` | done |
| `resolved_touch_cancels_scheduled_long_press` | gpui::test | 10156 |  | `TestWindow_ResolvedTouchCancelsScheduledLongPress` | done |
| `stale_long_press_timer_cannot_affect_replacement_touch` | gpui::test | 10180 |  | `TestWindow_StaleLongPressTimerCannotAffectReplacementTouch` | done |
| `test_underline_exclusions_preserve_device_geometry` | gpui::test | 10205 |  | `TestWindow_UnderlineExclusionsPreserveDeviceGeometry` | done |
| `test_underline_exclusions_preserve_helvetica_warning_edge` | gpui::test | 10314 |  | `TestWindow_UnderlineExclusionsPreserveHelveticaWarningEdge` | done |
| `test_underline_exclusions_preserve_content_mask_and_opacity` | gpui::test | 10353 |  | `TestWindow_UnderlineExclusionsPreserveContentMaskAndOpacity` | done |
| `test_underline_exclusions_adversarial_permutations` | gpui::test | 10411 |  | `TestWindow_UnderlineExclusionsAdversarialPermutations` | done |
| `test_empty_content_masks_remain_empty_when_snapped` | gpui::test | 10457 |  | `TestWindow_EmptyContentMasksRemainEmptyWhenSnapped` | done |
| `test_nonempty_content_masks_keep_covering_fractional_edges` | gpui::test | 10503 |  | `TestWindow_NonemptyContentMasksKeepCoveringFractionalEdges` | done |
| `test_empty_shadow_bounds_are_not_painted` | gpui::test | 10520 |  | `TestWindow_EmptyShadowBoundsAreNotPainted` | done |
| `test_collapsed_inset_shadow_hole_is_an_empty_point_on_the_pixel_grid` | gpui::test | 10560 |  | `TestWindow_CollapsedInsetShadowHoleIsAnEmptyPointOnThePixelGrid` | done |
| `test_standalone_modifier_sequence_timeout_and_recovery` | gpui::test | 10701 |  | `TestWindow_StandaloneModifierSequenceTimeoutAndRecovery` | done |
| `test_standalone_modifier_sequence_dispatches_before_timeout` | gpui::test | 10754 |  | `TestWindow_StandaloneModifierSequenceDispatchesBeforeTimeout` | done |
| `cached_view_follows_inherited_opacity` | gpui::test | 10823 |  | `TestWindow_CachedViewFollowsInheritedOpacity` | done |
| `test_drag_cursor_applies_before_next_paint` | gpui::test | 10928 |  | `TestWindow_DragCursorAppliesBeforeNextPaint` | done |
| `inspector_only_tracks_its_open_window` | gpui::test | 11017 | feat:inspector,debug | `TestWindowInspectorTests_InspectorOnlyTracksItsOpenWindow` | done |

## gpui/src/window/a11y.rs

Go target: `window_a11y.go` (package `gpui`), tests in `window_a11y_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `A11ySubtreeBuilder` | struct | 304 |  | `A11ySubtreeBuilder` | done |
| `A11ySubtreeBuilder::synthetic_node_id` | method | 335 |  | `(A11ySubtreeBuilder).SyntheticNodeID` | dev:D-A11YHASH |
| `A11ySubtreeBuilder::push_child` | method | 346 |  | `(A11ySubtreeBuilder).PushChild` | done |
| `A11ySubtreeBuilder::parent_node` | method | 364 |  | `(A11ySubtreeBuilder).ParentNode` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `accessibility_enabled_is_independent_of_activation` | test | 662 |  | `TestA11y_AccessibilityEnabledIsIndependentOfActivation` | done |
| `active_descendant_honored_when_container_focused` | test | 675 |  | `TestA11y_ActiveDescendantHonoredWhenContainerFocused` | done |
| `active_descendant_honored_for_deep_descendant` | test | 696 |  | `TestA11y_ActiveDescendantHonoredForDeepDescendant` | done |
| `active_descendant_ignored_when_focus_in_other_subtree` | test | 720 |  | `TestA11y_ActiveDescendantIgnoredWhenFocusInOtherSubtree` | done |
| `active_descendant_ignored_when_nothing_focused` | test | 747 |  | `TestA11y_ActiveDescendantIgnoredWhenNothingFocused` | done |
| `regular_focus_used_when_no_active_descendant` | test | 766 |  | `TestA11y_RegularFocusUsedWhenNoActiveDescendant` | done |
| `focus_is_ancestor_excludes_self_and_non_ancestors` | test | 779 |  | `TestA11y_FocusIsAncestorExcludesSelfAndNonAncestors` | done |
| `multiple_active_descendant_claims_panic_in_debug` | test | 806 |  | `TestA11y_MultipleActiveDescendantClaimsPanicInDebug` | done |
| `setting_focus_twice_panics_in_debug` | test | 819 |  | `TestA11y_SettingFocusTwicePanicsInDebug` | done |
| `set_focus_without_set_focusable` | test | 832 |  | `TestA11y_SetFocusWithoutSetFocusable` | done |
| `set_active_descendant_on_focused_node` | test | 844 |  | `TestA11y_SetActiveDescendantOnFocusedNode` | done |
| `two_siblings_claiming_active_descendant` | test | 861 |  | `TestA11y_TwoSiblingsClaimingActiveDescendant` | done |
| `active_descendant_in_unfocused_subtree_keeps_real_focus` | test | 885 |  | `TestA11y_ActiveDescendantInUnfocusedSubtreeKeepsRealFocus` | done |

## gpui/src/window/native_view.rs

Go target: `window_native_view.go` (package `gpui`), tests in `window_native_view_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `NativeViewId` | struct | 20 |  | `NativeViewID` | done |
| `NativeViewId::for_test` | assoc-fn | 25 | feat:test-support | `NativeViewIDForTest` | done |
| `NativeView` | struct | 69 |  | `NativeView` | done |
| `NativeView::new` | assoc-fn | 102 |  | `NewNativeView` | done |
| `NativeView::set_background` | method | 130 |  | `(NativeView).SetBackground` | done |
| `NativeView::reserve_action` | method | 148 |  | `(NativeView).ReserveAction` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `focusing_a_native_view_gives_it_keyboard_input` | gpui::test | 408 |  | `TestNativeView_FocusingANativeViewGivesItKeyboardInput` | done |
| `platform_focus_of_a_native_view_focuses_its_handle` | gpui::test | 435 |  | `TestNativeView_PlatformFocusOfANativeViewFocusesItsHandle` | done |
| `only_reserved_and_pending_bindings_apply_before_a_native_view` | gpui::test | 456 |  | `TestNativeView_OnlyReservedAndPendingBindingsApplyBeforeANativeView` | done |
| `mouse_reaches_a_native_view_only_where_nothing_covers_it` | gpui::test | 534 |  | `TestNativeView_MouseReachesANativeViewOnlyWhereNothingCoversIt` | done |
| `mouse_press_on_a_native_view_reaches_nothing_behind_it` | gpui::test | 551 |  | `TestNativeView_MousePressOnANativeViewReachesNothingBehindIt` | done |
| `changing_the_background_repaints_the_native_view` | gpui::test | 570 |  | `TestNativeView_ChangingTheBackgroundRepaintsTheNativeView` | done |
| `native_view_painted_twice_shows_where_it_was_painted_first` | gpui::test | 596 |  | `TestNativeView_NativeViewPaintedTwiceShowsWhereItWasPaintedFirst` | done |
| `native_view_is_drawn_in_paint_order` | gpui::test | 641 |  | `TestNativeView_NativeViewIsDrawnInPaintOrder` | done |
| `native_view_stays_attached_until_its_last_handle_is_dropped` | gpui::test | 661 |  | `TestNativeView_NativeViewStaysAttachedUntilItsLastHandleIsDropped` | done |
| `native_view_shows_nothing_in_another_window` | gpui::test | 681 |  | `TestNativeView_NativeViewShowsNothingInAnotherWindow` | done |
| `focusing_a_native_view_ends_a_composition` | gpui::test | 710 |  | `TestNativeView_FocusingANativeViewEndsAComposition` | done |

## gpui/src/window/prompts.rs

Go target: `window_prompts.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PromptResponse` | struct | 16 |  | `PromptResponse` | done |
| `Prompt` | trait | 19 |  | `Prompt` | done |
| `PromptHandle` | struct | 24 |  | `PromptHandle` | done |
| `PromptHandle::with_view` | method | 34 |  | `(PromptHandle).WithView` | done |
| `RenderablePromptHandle` | struct | 67 |  | `RenderablePromptHandle` | done |
| `fallback_prompt_renderer` | fn | 73 |  | `NewFallbackPromptRenderer` | done |
| `FallbackPromptRenderer` | struct | 94 |  | `FallbackPromptRenderer` | done |


## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-PREPAINTFOCUS | `AnyElement.Prepaint`, `PrepaintAt` and `PrepaintAsRoot` return nothing; upstream returns `Option<FocusHandle>` for a focus assigned inside the subtree | `gpui/src/element.rs:689-738` | no caller in zed@6501df1 reads the result; a returned counted `FocusHandle` would have to be released by every caller (D-RAII), and ignoring it would leak a count |
| D-INTOELEMENT | `IntoElement` has only `IntoAnyElement`; `into_element` and the associated `Element` type do not exist, and `AnyElement` does not implement `Element` (its own `RequestLayout(window, cx)` / `Paint` / `Prepaint` would clash with the trait's) | `gpui/src/element.rs:154-166`, `:741-792` | a Go interface cannot carry an associated type; every use goes through `AnyElement` |
| D-VIEWADAPT | `Entity[T]`, `AnyView` and `RenderOnce` components are not `View` values themselves: unexported adapters (`entityView[T]`, `anyViewView`, `componentView[C]`) implement `View` for them, and `ViewElement` is a non-generic struct holding a `View` plus the type name upstream takes from `type_name::<V>()`. `IntoAnyElement`, `Cached` and `NewComponent` build it. `Entity::into_any_element` (and `AnyView`'s) moves the handle into the element, which releases it after rendering or, when the view was never rendered (a cache hit), at the window's arena clear (`Window.deferArenaDrop`) | `gpui/src/view.rs:93-120`, `:208-258`, `:266-320` | `Entity[T].EntityID() EntityID` and `AnyView.EntityID() EntityID` clash with `View.EntityID() (EntityID, bool)`, and `RenderOnce.Render` has `View.Render`'s signature |
| D-INSPECTORORDER | `Inspector.RenderInspectorStates` renders the active element's states in the order they were first stored; upstream iterates a `TypeIdHashMap` | `gpui/src/inspector.rs:158-196` | Go map iteration is randomized per run; insertion order is the deterministic choice |
| D-SRCLOC | `SourceLocation.Column` is 0: Go's runtime reports file and line only. `CallerSourceLocation(skip)` replaces `#[track_caller]` + `Location::caller()` and interns one value per call site | `gpui/src/element.rs:70`, `gpui/src/view.rs:297` | `runtime.Frame` has no column |
| D-SPRANGE | `Window.PaintUnderlineWithExclusions` takes `[]ScaledPixelsRange{Start, End ScaledPixels}` for upstream's `&[Range<ScaledPixels>]` | `gpui/src/window.rs:3620-3626` | D-RANGE covers `Range<usize>` only; Go has no generic std range type |
| D-A11YHASH | `A11ySubtreeBuilder.SyntheticNodeID` hashes the parent id and `gpui.Hash(key)` with FxHasher instead of std `DefaultHasher` (SipHash-1-3), so synthetic node ids differ numerically from upstream; uniqueness and stability per parent and key are the same | `gpui/src/window/a11y.rs:335` | Go has no std SipHash-1-3 and keys are generic, not `impl Hash` |
