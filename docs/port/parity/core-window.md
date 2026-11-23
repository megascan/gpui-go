# Parity: core-window

Scope: gpui: window, element, view, inspector, a11y, native views, prompts. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/element.rs

Go target: `element.go` (package `gpui`), tests in `element_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Element` | trait | 52 |  | `Element` | todo |
| `Element::id` | trait-method | 66 |  | `Element.ElementID` | todo dev:D-NAME |
| `Element::source_location` | trait-method | 70 |  | `Element.SourceLocation` | todo |
| `Element::request_layout` | trait-method | 74 |  | `Element.RequestLayout` | todo |
| `Element::prepaint` | trait-method | 84 |  | `Element.Prepaint` | todo |
| `Element::paint` | trait-method | 96 |  | `Element.Paint` | todo |
| `Element::a11y_role` | trait-method | 113 |  | `Element.A11yRole` | todo |
| `Element::write_a11y_info` | trait-method | 121 |  | `Element.WriteA11yInfo` | todo |
| `Element::a11y_synthetic_children` | trait-method | 132 |  | `Element.ElementA11ySyntheticChildren` | todo dev:D-NAME |
| `Element::into_any` | trait-method | 140 |  | `Element.IntoAny` | todo |
| `IntoElement` | trait | 154 |  | `IntoElement` | todo |
| `IntoElement::into_element` | trait-method | 160 |  | `IntoElement.IntoElement` | todo |
| `IntoElement::into_any_element` | trait-method | 163 |  | `IntoElement.IntoAnyElement` | todo |
| `Render` | trait | 172 |  | `Render` | todo |
| `Render::render` | trait-method | 174 |  | `Render.Render` | todo |
| `RenderOnce` | trait | 188 |  | `RenderOnce` | todo |
| `RenderOnce::render` | trait-method | 192 |  | `RenderOnce.Render` | todo |
| `ParentElement` | trait | 197 |  | `Parent[S] mixin + ParentElement interface` | todo |
| `ParentElement::extend` | trait-method | 199 |  | `(Parent[S]).Extend` | todo |
| `ParentElement::child` | trait-method | 202 |  | `(Parent[S]).Child` | todo |
| `ParentElement::children` | trait-method | 211 |  | `(Parent[S]).Children` | todo |
| `GlobalElementId` | struct | 228 |  | `GlobalElementID` | todo |
| `Drawable` | struct | 324 |  | `Drawable` | todo |
| `AnyElement` | struct | 658 |  | `AnyElement` | todo |
| `AnyElement::downcast_mut` | method | 677 |  | `(AnyElement).DowncastMut` | todo |
| `AnyElement::request_layout` | method | 683 |  | `(AnyElement).RequestLayout` | todo |
| `AnyElement::prepaint` | method | 689 |  | `(AnyElement).Prepaint` | todo |
| `AnyElement::paint` | method | 702 |  | `(AnyElement).Paint` | todo |
| `AnyElement::layout_as_root` | method | 707 |  | `(AnyElement).LayoutAsRoot` | todo |
| `AnyElement::prepaint_at` | method | 718 |  | `(AnyElement).PrepaintAt` | todo |
| `AnyElement::prepaint_as_root` | method | 729 |  | `(AnyElement).PrepaintAsRoot` | todo |
| `Empty` | struct | 795 |  | `Empty` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `equal_paths_are_equal_and_hash_alike` | test | 892 |  | `TestElementGlobalElementIDTests_EqualPathsAreEqualAndHashAlike` | todo |
| `frame_cost` | crate::test ignore | 941 |  | `TestElementGlobalElementIDTests_FrameCost` | todo |
| `hash_cost` | test ignore | 973 |  | `TestElementGlobalElementIDTests_HashCost` | todo |

## gpui/src/inspector.rs

Go target: `inspector.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `InspectorElementId` | struct | 3 |  | `InspectorElementID` | todo |
| `InspectorElementPath` | struct | 30 | feat:inspector,debug | `InspectorElementPath` | todo |
| `InspectorRenderer` | type | 55 | feat:inspector,debug | `InspectorRenderer` | todo |
| `Inspector` | struct | 60 | feat:inspector,debug | `Inspector` | todo |
| `Inspector::active_element_id` | method | 117 | feat:inspector,debug | `(Inspector).ActiveElementID` | todo |
| `Inspector::start_picking` | method | 148 | feat:inspector,debug | `(Inspector).StartPicking` | todo |
| `Inspector::is_picking` | method | 153 | feat:inspector,debug | `(Inspector).IsPicking` | todo |
| `Inspector::render_inspector_states` | method | 158 | feat:inspector,debug | `(Inspector).RenderInspectorStates` | todo |
| `FunctionReflection` | struct | 250 | feat:inspector,debug | `FunctionReflection` | todo |
| `FunctionReflection::invoke` | method | 263 | feat:inspector,debug | `(FunctionReflection).Invoke` | todo |

## gpui/src/view.rs

Go target: `view.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AnyView` | struct | 24 |  | `AnyView` | todo |
| `AnyView::cached` | method | 46 |  | `(AnyView).Cached` | todo |
| `AnyView::downgrade` | method | 51 |  | `(AnyView).Downgrade` | todo |
| `AnyView::downcast` | method | 61 |  | `(AnyView).Downcast` | todo |
| `AnyView::entity_type` | method | 73 |  | `(AnyView).EntityType` | todo |
| `AnyView::entity_id` | method | 78 |  | `(AnyView).EntityID` | todo |
| `AnyWeakView` | struct | 135 |  | `AnyWeakView` | todo |
| `AnyWeakView::upgrade` | method | 143 |  | `(AnyWeakView).Upgrade` | todo |
| `View` | trait | 208 |  | `View` | todo |
| `View::entity_id` | trait-method | 217 |  | `View.EntityID` | todo |
| `View::render` | trait-method | 220 |  | `View.Render` | todo |
| `Entity::cached` | method | 258 |  | `(Entity).Cached` | todo |
| `ViewElement` | struct | 266 |  | `ViewElement` | todo |
| `ViewElement::new` | assoc-fn | 279 |  | `NewViewElement` | todo |
| `EmptyView` | struct | 470 |  | `EmptyView` | todo |

## gpui/src/window.rs

Go target: `window.go` (package `gpui`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `DEFAULT_WINDOW_SIZE` | const | 88 |  | `DefaultWindowSize` | todo |
| `DEFAULT_ADDITIONAL_WINDOW_SIZE` | const | 92 |  | `DefaultAdditionalWindowSize` | todo |
| `DispatchPhase` | enum | 99 |  | `DispatchPhase` | todo |
| `DispatchPhase::bubble` | method | 117 |  | `(DispatchPhase).Bubble` | todo |
| `DispatchPhase::capture` | method | 123 |  | `(DispatchPhase).Capture` | todo |
| `FocusOutEvent` | struct | 328 |  | `FocusOutEvent` | todo |
| `FocusId` | struct | 335 |  | `FocusID` | todo |
| `ArenaClearNeeded` | struct | 464 |  | `ArenaClearNeeded` | todo |
| `ArenaClearNeeded::clear` | method | 487 |  | `(ArenaClearNeeded).Clear` | todo |
| `FocusId::is_focused` | method | 510 |  | `(FocusID).IsFocused` | todo |
| `FocusId::contains_focused` | method | 516 |  | `(FocusID).ContainsFocused` | todo |
| `FocusId::within_focused` | method | 524 |  | `(FocusID).WithinFocused` | todo |
| `FocusHandle` | struct | 539 |  | `FocusHandle` | todo |
| `FocusHandle::tab_index` | method | 585 |  | `(FocusHandle).TabIndex` | todo |
| `FocusHandle::tab_stop` | method | 596 |  | `(FocusHandle).TabStop` | todo |
| `FocusHandle::downgrade` | method | 605 |  | `(FocusHandle).Downgrade` | todo |
| `FocusHandle::focus` | method | 613 |  | `(FocusHandle).Focus` | todo |
| `FocusHandle::is_focused` | method | 618 |  | `(FocusHandle).IsFocused` | todo |
| `FocusHandle::contains_focused` | method | 624 |  | `(FocusHandle).ContainsFocused` | todo |
| `FocusHandle::within_focused` | method | 630 |  | `(FocusHandle).WithinFocused` | todo |
| `FocusHandle::contains` | method | 635 |  | `(FocusHandle).Contains` | todo |
| `FocusHandle::is_rendered` | method | 640 |  | `(FocusHandle).IsRendered` | todo |
| `FocusHandle::dispatch_action` | method | 649 |  | `(FocusHandle).DispatchAction` | todo |
| `WeakFocusHandle` | struct | 690 |  | `WeakFocusHandle` | todo |
| `WeakFocusHandle::upgrade` | method | 697 |  | `(WeakFocusHandle).Upgrade` | todo |
| `Focusable` | trait | 725 |  | `Focusable` | todo |
| `Focusable::focus_handle` | trait-method | 727 |  | `Focusable.FocusHandle` | todo |
| `ManagedView` | trait | 738 |  | `ManagedView` | todo |
| `DismissEvent` | struct | 743 |  | `DismissEvent` | todo |
| `WindowControlArea` | enum | 764 |  | `WindowControlArea` | todo |
| `HitboxId` | struct | 777 |  | `HitboxID` | todo |
| `HitboxId::placeholder` | assoc-fn | 785 | feat:test-support | `HitboxIDPlaceholder` | todo |
| `HitboxId::is_hovered` | method | 797 |  | `(HitboxID).IsHovered` | todo |
| `HitboxId::should_handle_scroll` | method | 834 |  | `(HitboxID).ShouldHandleScroll` | todo |
| `Hitbox` | struct | 846 |  | `Hitbox` | todo |
| `Hitbox::is_hovered` | method | 875 |  | `(Hitbox).IsHovered` | todo |
| `Hitbox::is_hovered_at` | method | 881 |  | `(Hitbox).IsHoveredAt` | todo |
| `Hitbox::should_handle_scroll` | method | 896 |  | `(Hitbox).ShouldHandleScroll` | todo |
| `HitboxBehavior` | enum | 903 |  | `HitboxBehavior` | todo |
| `TooltipId` | struct | 962 |  | `TooltipID` | todo |
| `TooltipId::is_hovered` | method | 966 |  | `(TooltipID).IsHovered` | todo |
| `Window` | struct | 1232 |  | `Window` | todo |
| `PendingInputStatus` | struct | 1452 |  | `PendingInputStatus` | todo |
| `PendingInputStatus::keystrokes` | method | 1459 |  | `(PendingInputStatus).Keystrokes` | todo |
| `PendingInputStatus::timeout` | method | 1464 |  | `(PendingInputStatus).Timeout` | todo |
| `PendingInputTimeoutStatus` | struct | 1471 |  | `PendingInputTimeoutStatus` | todo |
| `PendingInputTimeoutStatus::duration` | method | 1480 |  | `(PendingInputTimeoutStatus).Duration` | todo |
| `PendingInputTimeoutStatus::remaining` | method | 1485 |  | `(PendingInputTimeoutStatus).Remaining` | todo |
| `PendingInputTimeoutStatus::is_paused` | method | 1495 |  | `(PendingInputTimeoutStatus).IsPaused` | todo |
| `DispatchEventResult` | struct | 2532 |  | `DispatchEventResult` | todo |
| `ContentMask` | struct | 2542 |  | `ContentMask` | todo |
| `ContentMask<Pixels>::scale` | method | 2549 |  | `(ContentMask).Scale` | todo |
| `ContentMask<Pixels>::intersect` | method | 2556 |  | `(ContentMask).Intersect` | todo |
| `Window::visibility` | method | 2654 |  | `(Window).Visibility` | todo |
| `Window::is_visible` | method | 2663 |  | `(Window).IsVisible` | todo |
| `Window::observe_window_visibility` | method | 2668 |  | `(Window).ObserveWindowVisibility` | todo |
| `Window::observe_window_appearance` | method | 2680 |  | `(Window).ObserveWindowAppearance` | todo |
| `Window::observe_button_layout_changed` | method | 2692 |  | `(Window).ObserveButtonLayoutChanged` | todo |
| `Window::replace_root` | method | 2704 |  | `(Window).ReplaceRoot` | todo |
| `Window::root` | method | 2719 |  | `(Window).Root` | todo |
| `Window::window_handle` | method | 2729 |  | `(Window).WindowHandle` | todo |
| `Window::refresh` | method | 2734 |  | `(Window).Refresh` | todo |
| `Window::remove_window` | method | 2742 |  | `(Window).RemoveWindow` | todo |
| `Window::focused` | method | 2747 |  | `(Window).Focused` | todo |
| `Window::focus_lost_restore_target` | method | 2755 |  | `(Window).FocusLostRestoreTarget` | todo |
| `Window::focus` | method | 2764 |  | `(Window).Focus` | todo |
| `Window::blur` | method | 2779 |  | `(Window).Blur` | todo |
| `Window::disable_focus` | method | 2795 |  | `(Window).DisableFocus` | todo |
| `Window::focus_next` | method | 2801 |  | `(Window).FocusNext` | todo |
| `Window::focus_prev` | method | 2812 |  | `(Window).FocusPrev` | todo |
| `Window::text_system` | method | 2823 |  | `(Window).TextSystem` | todo |
| `Window::text_style` | method | 2828 |  | `(Window).TextStyle` | todo |
| `Window::is_maximized` | method | 2839 |  | `(Window).IsMaximized` | todo |
| `Window::request_decorations` | method | 2844 |  | `(Window).RequestDecorations` | todo |
| `Window::set_exclusive_zone` | method | 2853 |  | `(Window).SetExclusiveZone` | todo |
| `Window::set_exclusive_edge` | method | 2862 | linux,feat:wayland | `(Window).SetExclusiveEdge` | todo |
| `Window::start_window_resize` | method | 2867 |  | `(Window).StartWindowResize` | todo |
| `Window::set_input_region` | method | 2879 |  | `(Window).SetInputRegion` | todo |
| `Window::window_bounds` | method | 2885 |  | `(Window).WindowBounds` | todo |
| `Window::inner_window_bounds` | method | 2898 |  | `(Window).InnerWindowBounds` | todo |
| `Window::native_window_state` | method | 2904 |  | `(Window).NativeWindowState` | todo |
| `Window::restore_native_window_state` | method | 2910 |  | `(Window).RestoreNativeWindowState` | todo |
| `Window::dispatch_action` | method | 2915 |  | `(Window).DispatchAction` | todo |
| `Window::defer` | method | 2975 |  | `(Window).Defer` | todo |
| `Window::observe` | method | 2985 |  | `(Window).Observe` | todo |
| `Window::subscribe` | method | 3014 |  | `(Window).Subscribe` | todo |
| `Window::observe_release` | method | 3049 |  | `(Window).ObserveRelease` | todo |
| `Window::to_async` | method | 3071 |  | `(Window).ToAsync` | todo |
| `Window::on_next_frame` | method | 3076 |  | `(Window).OnNextFrame` | todo |
| `Window::request_animation_frame` | method | 3096 |  | `(Window).RequestAnimationFrame` | todo |
| `Window::simulate_next_frame` | method | 3111 | feat:test-support | `(Window).SimulateNextFrame` | todo |
| `Window::spawn` | method | 3124 |  | `(Window).Spawn` | todo |
| `Window::spawn_with_priority` | method | 3140 |  | `(Window).SpawnWithPriority` | todo |
| `Window::bounds_changed` | method | 3162 |  | `(Window).BoundsChanged` | todo |
| `Window::refresh_interval` | method | 3224 |  | `(Window).RefreshInterval` | todo |
| `Window::observe_window_display` | method | 3232 |  | `(Window).ObserveWindowDisplay` | todo |
| `Window::bounds` | method | 3243 |  | `(Window).Bounds` | todo |
| `Window::insets` | method | 3248 |  | `(Window).Insets` | todo |
| `Window::render_to_image` | method | 3256 | feat:test-support | `(Window).RenderToImage` | todo |
| `Window::rendered_scene` | method | 3264 | feat:bench-support | `(Window).RenderedScene` | todo |
| `Window::painted_quads` | method | 3273 | feat:test-support | `(Window).PaintedQuads` | todo |
| `Window::painted_underlines` | method | 3279 | feat:test-support | `(Window).PaintedUnderlines` | todo |
| `Window::resize` | method | 3284 |  | `(Window).Resize` | todo |
| `Window::is_fullscreen` | method | 3289 |  | `(Window).IsFullscreen` | todo |
| `Window::is_simple_fullscreen` | method | 3296 |  | `(Window).IsSimpleFullscreen` | todo |
| `Window::appearance` | method | 3315 |  | `(Window).Appearance` | todo |
| `Window::viewport_size` | method | 3320 |  | `(Window).ViewportSize` | todo |
| `Window::visual_viewport_bounds` | method | 3329 |  | `(Window).VisualViewportBounds` | todo |
| `Window::fully_visible_bounds` | method | 3337 |  | `(Window).FullyVisibleBounds` | todo |
| `Window::request_virtual_keyboard` | method | 3355 |  | `(Window).RequestVirtualKeyboard` | todo |
| `Window::dismiss_virtual_keyboard` | method | 3360 |  | `(Window).DismissVirtualKeyboard` | todo |
| `Window::is_window_active` | method | 3365 |  | `(Window).IsWindowActive` | todo |
| `Window::is_window_hovered` | method | 3372 |  | `(Window).IsWindowHovered` | todo |
| `Window::zoom_window` | method | 3385 |  | `(Window).ZoomWindow` | todo |
| `Window::show_window_menu` | method | 3390 |  | `(Window).ShowWindowMenu` | todo |
| `Window::start_window_move` | method | 3398 |  | `(Window).StartWindowMove` | todo |
| `Window::set_client_inset` | method | 3403 |  | `(Window).SetClientInset` | todo |
| `Window::set_client_corner_radii` | method | 3409 |  | `(Window).SetClientCornerRadii` | todo |
| `Window::client_inset` | method | 3414 |  | `(Window).ClientInset` | todo |
| `Window::window_decorations` | method | 3419 |  | `(Window).WindowDecorations` | todo |
| `Window::is_resizable` | method | 3424 |  | `(Window).IsResizable` | todo |
| `Window::is_minimizable` | method | 3429 |  | `(Window).IsMinimizable` | todo |
| `Window::window_controls` | method | 3434 |  | `(Window).WindowControls` | todo |
| `Window::set_window_title` | method | 3439 |  | `(Window).SetWindowTitle` | todo |
| `Window::set_traffic_light_position` | method | 3446 | macos | `(Window).SetTrafficLightPosition` | todo |
| `Window::set_native_titlebar_visibility` | method | 3452 | macos | `(Window).SetNativeTitlebarVisibility` | todo |
| `Window::set_app_id` | method | 3457 |  | `(Window).SetAppID` | todo |
| `Window::set_background_appearance` | method | 3462 |  | `(Window).SetBackgroundAppearance` | todo |
| `Window::set_window_edited` | method | 3468 |  | `(Window).SetWindowEdited` | todo |
| `Window::set_document_path` | method | 3474 |  | `(Window).SetDocumentPath` | todo |
| `Window::display` | method | 3479 |  | `(Window).Display` | todo |
| `Window::show_character_palette` | method | 3487 |  | `(Window).ShowCharacterPalette` | todo |
| `Window::scale_factor` | method | 3494 |  | `(Window).ScaleFactor` | todo |
| `Window::set_scale_factor` | method | 3500 | feat:test-support | `(Window).SetScaleFactor` | todo |
| `Window::rem_size` | method | 3507 |  | `(Window).RemSize` | todo |
| `Window::set_rem_size` | method | 3516 |  | `(Window).SetRemSize` | todo |
| `Window::with_global_id` | method | 3522 |  | `(Window).WithGlobalID` | todo |
| `Window::with_id` | method | 3536 |  | `(Window).WithID` | todo |
| `Window::with_rem_size` | method | 3553 |  | `(Window).WithRemSize` | todo |
| `Window::line_height` | method | 3570 |  | `(Window).LineHeight` | todo |
| `Window::pixel_snap` | method | 3576 |  | `(Window).PixelSnap` | todo |
| `Window::pixel_snap_f64` | method | 3582 |  | `(Window).PixelSnapF64` | todo |
| `Window::pixel_snap_bounds` | method | 3589 |  | `(Window).PixelSnapBounds` | todo |
| `Window::pixel_snap_point` | method | 3595 |  | `(Window).PixelSnapPoint` | todo |
| `Window::underline_bounds` | method | 3600 |  | `(Window).UnderlineBounds` | todo |
| `Window::paint_underline_with_exclusions` | method | 3620 |  | `(Window).PaintUnderlineWithExclusions` | todo |
| `Window::prevent_default` | method | 3748 |  | `(Window).PreventDefault` | todo |
| `Window::default_prevented` | method | 3753 |  | `(Window).DefaultPrevented` | todo |
| `Window::is_action_available` | method | 3758 |  | `(Window).IsActionAvailable` | todo |
| `Window::is_action_available_in` | method | 3767 |  | `(Window).IsActionAvailableIn` | todo |
| `Window::mouse_position` | method | 3775 |  | `(Window).MousePosition` | todo |
| `Window::capture_pointer` | method | 3785 |  | `(Window).CapturePointer` | todo |
| `Window::release_pointer` | method | 3790 |  | `(Window).ReleasePointer` | todo |
| `Window::captured_hitbox` | method | 3795 |  | `(Window).CapturedHitbox` | todo |
| `Window::capture_long_press` | method | 3804 |  | `(Window).CaptureLongPress` | todo |
| `Window::has_long_press_capture` | method | 3809 |  | `(Window).HasLongPressCapture` | todo |
| `Window::modifiers` | method | 3814 |  | `(Window).Modifiers` | todo |
| `Window::last_input_was_keyboard` | method | 3820 |  | `(Window).LastInputWasKeyboard` | todo |
| `Window::last_input_was_touch` | method | 3825 |  | `(Window).LastInputWasTouch` | todo |
| `Window::capslock` | method | 3830 |  | `(Window).Capslock` | todo |
| `Window::draw` | method | 3837 |  | `(Window).Draw` | todo |
| `Window::input_latency_snapshot` | method | 4045 | feat:profiler | `(Window).InputLatencySnapshot` | todo |
| `Window::frame_duration_snapshot` | method | 4051 | feat:profiler | `(Window).FrameDurationSnapshot` | todo |
| `Window::debug_frame_overlay_mode` | method | 4057 | feat:profiler | `(Window).DebugFrameOverlayMode` | todo |
| `Window::set_debug_frame_overlay_mode` | method | 4063 | feat:profiler | `(Window).SetDebugFrameOverlayMode` | todo |
| `Window::cycle_debug_frame_overlay_mode` | method | 4071 | feat:profiler | `(Window).CycleDebugFrameOverlayMode` | todo |
| `Window::reset_debug_frame_overlay_stats` | method | 4078 | feat:profiler | `(Window).ResetDebugFrameOverlayStats` | todo |
| `Window::with_text_style` | method | 4538 |  | `(Window).WithTextStyle` | todo |
| `Window::set_cursor_style` | method | 4555 |  | `(Window).SetCursorStyle` | todo |
| `Window::set_window_cursor_style` | method | 4567 |  | `(Window).SetWindowCursorStyle` | todo |
| `Window::set_tooltip` | method | 4577 |  | `(Window).SetTooltip` | todo |
| `Window::with_content_mask` | method | 4591 |  | `(Window).WithContentMask` | todo |
| `Window::with_element_offset` | method | 4610 |  | `(Window).WithElementOffset` | todo |
| `Window::with_absolute_element_offset` | method | 4628 |  | `(Window).WithAbsoluteElementOffset` | todo |
| `Window::transact` | method | 4663 |  | `(Window).Transact` | todo |
| `Window::request_autoscroll` | method | 4691 |  | `(Window).RequestAutoscroll` | todo |
| `Window::take_autoscroll` | method | 4698 |  | `(Window).TakeAutoscroll` | todo |
| `Window::use_asset` | method | 4708 |  | `(Window).UseAsset` | todo |
| `Window::get_asset` | method | 4717 |  | `(Window).GetAsset` | todo |
| `Window::element_offset` | method | 4722 |  | `(Window).ElementOffset` | todo |
| `Window::content_mask` | method | 4739 |  | `(Window).ContentMask` | todo |
| `Window::with_element_namespace` | method | 4754 |  | `(Window).WithElementNamespace` | todo |
| `Window::use_keyed_state` | method | 4766 |  | `(Window).UseKeyedState` | todo |
| `Window::use_state` | method | 4792 |  | `(Window).UseState` | todo |
| `Window::with_element_state` | method | 4809 |  | `(Window).WithElementState` | todo |
| `Window::with_optional_element_state` | method | 4884 |  | `(Window).WithOptionalElementState` | todo |
| `Window::with_tab_group` | method | 4913 |  | `(Window).WithTabGroup` | todo |
| `Window::defer_draw` | method | 4932 |  | `(Window).DeferDraw` | todo |
| `Window::paint_layer` | method | 4965 |  | `(Window).PaintLayer` | todo |
| `Window::paint_drop_shadows` | method | 4979 |  | `(Window).PaintDropShadows` | todo |
| `Window::paint_inset_shadows` | method | 5019 |  | `(Window).PaintInsetShadows` | todo |
| `Window::paint_quad` | method | 5112 |  | `(Window).PaintQuad` | todo |
| `Window::paint_path` | method | 5224 |  | `(Window).PaintPath` | todo |
| `Window::paint_underline` | method | 5242 |  | `(Window).PaintUnderline` | todo |
| `Window::paint_strikethrough` | method | 5256 |  | `(Window).PaintStrikethrough` | todo |
| `Window::paint_glyph` | method | 5291 |  | `(Window).PaintGlyph` | todo |
| `Window::paint_emoji` | method | 5398 |  | `(Window).PaintEmoji` | todo |
| `Window::paint_svg` | method | 5455 |  | `(Window).PaintSvg` | todo |
| `Window::paint_image` | method | 5547 |  | `(Window).PaintImage` | todo |
| `Window::paint_surface` | method | 5649 | macos,ios | `(Window).PaintSurface` | todo |
| `Window::drop_image` | method | 5669 |  | `(Window).DropImage` | todo |
| `Window::has_image_atlas_entry` | method | 5684 | feat:test-support | `(Window).HasImageAtlasEntry` | todo |
| `Window::request_layout` | method | 5703 |  | `(Window).RequestLayout` | todo |
| `Window::request_measured_layout` | method | 5732 |  | `(Window).RequestMeasuredLayout` | todo |
| `Window::compute_layout` | method | 5752 |  | `(Window).ComputeLayout` | todo |
| `Window::layout_bounds` | method | 5769 |  | `(Window).LayoutBounds` | todo |
| `Window::insert_hitbox` | method | 5789 |  | `(Window).InsertHitbox` | todo |
| `Window::insert_window_control_hitbox` | method | 5808 |  | `(Window).InsertWindowControlHitbox` | todo |
| `Window::set_key_context` | method | 5817 |  | `(Window).SetKeyContext` | todo |
| `Window::set_focus_handle` | method | 5826 |  | `(Window).SetFocusHandle` | todo |
| `Window::set_view_id` | method | 5839 |  | `(Window).SetViewID` | todo |
| `Window::current_view` | method | 5845 |  | `(Window).CurrentView` | todo |
| `Window::with_image_cache` | method | 5863 |  | `(Window).WithImageCache` | todo |
| `Window::handle_input` | method | 5885 |  | `(Window).HandleInput` | todo |
| `Window::notify_text_input_changed` | method | 5905 |  | `(Window).NotifyTextInputChanged` | todo |
| `Window::notify_text_selection_changed` | method | 5916 |  | `(Window).NotifyTextSelectionChanged` | todo |
| `Window::on_mouse_event` | method | 5948 |  | `(Window).OnMouseEvent` | todo |
| `Window::on_key_event` | method | 5971 |  | `(Window).OnKeyEvent` | todo |
| `Window::on_modifiers_changed` | method | 5992 |  | `(Window).OnModifiersChanged` | todo |
| `Window::on_focus_in` | method | 6008 |  | `(Window).OnFocusIn` | todo |
| `Window::on_focus_out` | method | 6028 |  | `(Window).OnFocusOut` | todo |
| `Window::dispatch_keystroke` | method | 6068 |  | `(Window).DispatchKeystroke` | todo |
| `Window::keystroke_text_for` | method | 6095 |  | `(Window).KeystrokeTextFor` | todo |
| `Window::dispatch_event` | method | 6110 |  | `(Window).DispatchEvent` | todo |
| `Window::touch_prediction_enabled` | method | 6317 |  | `(Window).TouchPredictionEnabled` | todo |
| `Window::set_touch_prediction_enabled` | method | 6324 |  | `(Window).SetTouchPredictionEnabled` | todo |
| `Window::has_pending_keystrokes` | method | 6852 |  | `(Window).HasPendingKeystrokes` | todo |
| `Window::pending_input` | method | 6869 |  | `(Window).PendingInput` | todo |
| `Window::set_pending_input_timeout_paused` | method | 6886 |  | `(Window).SetPendingInputTimeoutPaused` | todo |
| `Window::pending_input_keystrokes` | method | 6959 |  | `(Window).PendingInputKeystrokes` | todo |
| `Window::observe_global` | method | 7146 |  | `(Window).ObserveGlobal` | todo |
| `Window::activate_window` | method | 7165 |  | `(Window).ActivateWindow` | todo |
| `Window::request_attention` | method | 7170 |  | `(Window).RequestAttention` | todo |
| `Window::minimize_window` | method | 7175 |  | `(Window).MinimizeWindow` | todo |
| `Window::toggle_fullscreen` | method | 7180 |  | `(Window).ToggleFullscreen` | todo |
| `Window::toggle_simple_fullscreen` | method | 7188 |  | `(Window).ToggleSimpleFullscreen` | todo |
| `Window::invalidate_character_coordinates` | method | 7193 |  | `(Window).InvalidateCharacterCoordinates` | todo |
| `Window::prompt` | method | 7207 |  | `(Window).Prompt` | todo |
| `Window::has_active_prompt` | method | 7265 |  | `(Window).HasActivePrompt` | todo |
| `Window::context_stack` | method | 7270 |  | `(Window).ContextStack` | todo |
| `Window::available_actions` | method | 7281 |  | `(Window).AvailableActions` | todo |
| `Window::bindings_for_action` | method | 7297 |  | `(Window).BindingsForAction` | todo |
| `Window::highest_precedence_binding_for_action` | method | 7305 |  | `(Window).HighestPrecedenceBindingForAction` | todo |
| `Window::bindings_for_action_in_context` | method | 7315 |  | `(Window).BindingsForActionInContext` | todo |
| `Window::highest_precedence_binding_for_action_in_context` | method | 7326 |  | `(Window).HighestPrecedenceBindingForActionInContext` | todo |
| `Window::bindings_for_action_in` | method | 7338 |  | `(Window).BindingsForActionIn` | todo |
| `Window::highest_precedence_binding_for_action_in` | method | 7353 |  | `(Window).HighestPrecedenceBindingForActionIn` | todo |
| `Window::possible_bindings_for_input` | method | 7364 |  | `(Window).PossibleBindingsForInput` | todo |
| `Window::listener_for` | method | 7385 |  | `(Window).ListenerFor` | todo |
| `Window::handler_for` | assoc-fn | 7397 |  | `WindowHandlerFor` | todo |
| `Window::on_window_should_close` | method | 7410 |  | `(Window).OnWindowShouldClose` | todo |
| `Window::on_action` | method | 7429 |  | `(Window).OnAction` | todo |
| `Window::on_action_when` | method | 7449 |  | `(Window).OnActionWhen` | todo |
| `Window::gpu_specs` | method | 7466 |  | `(Window).GPUSpecs` | todo |
| `Window::titlebar_double_click` | method | 7472 |  | `(Window).TitlebarDoubleClick` | todo |
| `Window::window_title` | method | 7479 |  | `(Window).WindowTitle` | todo |
| `Window::tabbed_windows` | method | 7485 |  | `(Window).TabbedWindows` | todo |
| `Window::tab_bar_visible` | method | 7491 |  | `(Window).TabBarVisible` | todo |
| `Window::merge_all_windows` | method | 7497 |  | `(Window).MergeAllWindows` | todo |
| `Window::move_tab_to_new_window` | method | 7503 |  | `(Window).MoveTabToNewWindow` | todo |
| `Window::toggle_window_tab_overview` | method | 7509 |  | `(Window).ToggleWindowTabOverview` | todo |
| `Window::set_tabbing_identifier` | method | 7515 |  | `(Window).SetTabbingIdentifier` | todo |
| `Window::play_system_bell` | method | 7522 |  | `(Window).PlaySystemBell` | todo |
| `Window::is_a11y_enabled` | method | 7532 |  | `(Window).IsA11yEnabled` | todo |
| `Window::is_a11y_active` | method | 7546 |  | `(Window).IsA11yActive` | todo |
| `Window::debug_a11y_tree_json` | method | 7551 |  | `(Window).DebugA11yTreeJSON` | todo |
| `Window::on_a11y_action` | method | 7560 |  | `(Window).OnA11yAction` | todo |
| `Window::toggle_inspector` | method | 7638 | feat:inspector,debug | `(Window).ToggleInspector` | todo |
| `Window::is_inspector_picking` | method | 7653 |  | `(Window).IsInspectorPicking` | todo |
| `Window::with_inspector_state` | method | 7665 | feat:inspector,debug | `(Window).WithInspectorState` | todo |
| `Window::insert_inspector_hitbox` | method | 7731 | feat:inspector,debug | `(Window).InsertInspectorHitbox` | todo |
| `Window::set_modifiers` | method | 7834 | feat:test-support | `(Window).SetModifiers` | todo |
| `Window::simulate_mouse_move` | method | 7842 | feat:test-support | `(Window).SimulateMouseMove` | todo |
| `WindowId` | struct | 7888 |  | `WindowID` | todo |
| `WindowId::as_u64` | method | 7893 |  | `(WindowID).AsU64` | todo |
| `WindowHandle` | struct | 7907 |  | `WindowHandle` | todo |
| `WindowHandle::new` | assoc-fn | 7925 |  | `NewWindowHandle` | todo |
| `WindowHandle::root` | method | 7940 | feat:test-support | `(WindowHandle).Root` | todo |
| `WindowHandle::update` | method | 7955 |  | `(WindowHandle).Update` | todo |
| `WindowHandle::read` | method | 7975 |  | `(WindowHandle).Read` | todo |
| `WindowHandle::read_with` | method | 7994 |  | `(WindowHandle).ReadWith` | todo |
| `WindowHandle::entity` | method | 8004 |  | `(WindowHandle).Entity` | todo |
| `WindowHandle::is_active` | method | 8015 |  | `(WindowHandle).IsActive` | todo |
| `AnyWindowHandle` | struct | 8051 |  | `AnyWindowHandle` | todo |
| `AnyWindowHandle::window_id` | method | 8059 |  | `(AnyWindowHandle).WindowID` | todo |
| `AnyWindowHandle::root_entity_type_name` | method | 8064 |  | `(AnyWindowHandle).RootEntityTypeName` | todo |
| `AnyWindowHandle::downcast` | method | 8070 |  | `(AnyWindowHandle).Downcast` | todo |
| `AnyWindowHandle::update` | method | 8085 |  | `(AnyWindowHandle).Update` | todo |
| `AnyWindowHandle::read` | method | 8099 |  | `(AnyWindowHandle).Read` | todo |
| `ElementId` | enum | 8131 |  | `ElementID` | todo |
| `ElementId::named_usize` | assoc-fn | 8156 |  | `ElementIDNamedUsize` | todo |
| `PaintQuad` | struct | 8303 |  | `PaintQuad` | todo |
| `PaintQuad::corner_radii` | method | 8320 |  | `(PaintQuad).CornerRadii` | todo |
| `PaintQuad::border_widths` | method | 8328 |  | `(PaintQuad).BorderWidths` | todo |
| `PaintQuad::border_color` | method | 8336 |  | `(PaintQuad).BorderColor` | todo |
| `PaintQuad::background` | method | 8344 |  | `(PaintQuad).Background` | todo |
| `quad` | fn | 8353 |  | `NewQuad` | todo |
| `fill` | fn | 8372 |  | `NewFill` | todo |
| `outline` | fn | 8384 |  | `Outline` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_focus_handle_release` | gpui::test | 8434 |  | `TestWindow_FocusHandleRelease` | todo |
| `test_dropped_focus_handle_blurs_only_its_windows` | gpui::test | 8475 |  | `TestWindow_DroppedFocusHandleBlursOnlyItsWindows` | todo |
| `test_native_close_defers_app_access` | gpui::test | 8523 |  | `TestWindow_NativeCloseDefersAppAccess` | todo |
| `test_frame_requests_and_skips_are_journaled` | gpui::test | 8537 | feat:profiler | `TestWindow_FrameRequestsAndSkipsAreJournaled` | todo |
| `test_frame_demand_survives_throttling` | gpui::test | 8593 | feat:profiler | `TestWindow_FrameDemandSurvivesThrottling` | todo |
| `test_debug_state_reports_cached_and_platform_values` | gpui::test | 8700 |  | `TestWindow_DebugStateReportsCachedAndPlatformValues` | todo |
| `test_window_visibility` | gpui::test | 8729 |  | `TestWindow_WindowVisibility` | todo |
| `test_display_tracking` | gpui::test | 8774 |  | `TestWindow_DisplayTracking` | todo |
| `test_window_opened_from_bounds_observer_uses_new_bounds` | gpui::test | 8884 |  | `TestWindow_WindowOpenedFromBoundsObserverUsesNewBounds` | todo |
| `test_fully_visible_bounds_preserve_layout_viewport` | gpui::test | 8926 |  | `TestWindow_FullyVisibleBoundsPreserveLayoutViewport` | todo |
| `frame_throttle_tracks_display_refreshes` | test | 9014 |  | `TestWindow_FrameThrottleTracksDisplayRefreshes` | todo |
| `frame_throttle_preserves_cadence_without_catching_up_after_stalls` | test | 9028 |  | `TestWindow_FrameThrottlePreservesCadenceWithoutCatchingUpAfterStalls` | todo |
| `power_modes_preserve_frame_limits` | test | 9050 |  | `TestWindow_PowerModesPreserveFrameLimits` | todo |
| `frame_limits_combine_enabled_intervals` | test | 9102 |  | `TestWindow_FrameLimitsCombineEnabledIntervals` | todo |
| `test_window_opened_during_draw_defers_arena_clear` | test | 9180 |  | `TestWindow_WindowOpenedDuringDrawDefersArenaClear` | todo |
| `test_scale_factor_change_preserves_bounds_and_survives_resize` | test | 9200 |  | `TestWindow_ScaleFactorChangePreservesBoundsAndSurvivesResize` | todo |
| `test_fullscreen_toggle_reports_current_bounds` | gpui::test | 9239 |  | `TestWindow_FullscreenToggleReportsCurrentBounds` | todo |
| `test_frame_waker_fires_on_frame_demand` | gpui::test | 9287 |  | `TestWindow_FrameWakerFiresOnFrameDemand` | todo |
| `test_pending_next_frame_callbacks_are_not_stranded` | gpui::test | 9347 |  | `TestWindow_PendingNextFrameCallbacksAreNotStranded` | todo |
| `test_window_reports_no_raw_handle_instead_of_panicking` | gpui::test | 9379 |  | `TestWindow_WindowReportsNoRawHandleInsteadOfPanicking` | todo |
| `test_appearance_change_runs_after_app_update` | gpui::test | 9398 |  | `TestWindow_AppearanceChangeRunsAfterAppUpdate` | todo |
| `queued_frame_callback_wakes_a_parked_render_loop` | gpui::test | 9423 |  | `TestWindow_QueuedFrameCallbackWakesAParkedRenderLoop` | todo |
| `pending_presentation_wakes_a_parked_render_loop` | gpui::test | 9451 |  | `TestWindow_PendingPresentationWakesAParkedRenderLoop` | todo |
| `callback_queued_during_a_frame_requests_a_follow_up` | gpui::test | 9469 |  | `TestWindow_CallbackQueuedDuringAFrameRequestsAFollowUp` | todo |
| `auto_sized_window_root_fills_the_window` | test | 9520 |  | `TestWindow_AutoSizedWindowRootFillsTheWindow` | todo |
| `explicitly_sized_window_root_keeps_its_size` | test | 9542 |  | `TestWindow_ExplicitlySizedWindowRootKeepsItsSize` | todo |
| `file_drag_is_promoted_once_and_restored_in_source_window` | gpui::test | 9604 |  | `TestWindow_FileDragIsPromotedOnceAndRestoredInSourceWindow` | todo |
| `test_focus_moved_by_focus_listener_is_dispatched` | gpui::test | 9948 |  | `TestWindow_FocusMovedByFocusListenerIsDispatched` | todo |
| `claimed_touch_drag_receives_movement_and_release` | gpui::test | 9990 |  | `TestWindow_ClaimedTouchDragReceivesMovementAndRelease` | todo |
| `back_gesture_is_claimed_by_the_innermost_listener` | gpui::test | 10013 |  | `TestWindow_BackGestureIsClaimedByTheInnermostListener` | todo |
| `long_press_is_claimed_only_when_started_prevents_default` | gpui::test | 10089 |  | `TestWindow_LongPressIsClaimedOnlyWhenStartedPreventsDefault` | todo |
| `stale_default_prevention_does_not_claim_long_press` | gpui::test | 10132 |  | `TestWindow_StaleDefaultPreventionDoesNotClaimLongPress` | todo |
| `resolved_touch_cancels_scheduled_long_press` | gpui::test | 10156 |  | `TestWindow_ResolvedTouchCancelsScheduledLongPress` | todo |
| `stale_long_press_timer_cannot_affect_replacement_touch` | gpui::test | 10180 |  | `TestWindow_StaleLongPressTimerCannotAffectReplacementTouch` | todo |
| `test_underline_exclusions_preserve_device_geometry` | gpui::test | 10205 |  | `TestWindow_UnderlineExclusionsPreserveDeviceGeometry` | todo |
| `test_underline_exclusions_preserve_helvetica_warning_edge` | gpui::test | 10314 |  | `TestWindow_UnderlineExclusionsPreserveHelveticaWarningEdge` | todo |
| `test_underline_exclusions_preserve_content_mask_and_opacity` | gpui::test | 10353 |  | `TestWindow_UnderlineExclusionsPreserveContentMaskAndOpacity` | todo |
| `test_underline_exclusions_adversarial_permutations` | gpui::test | 10411 |  | `TestWindow_UnderlineExclusionsAdversarialPermutations` | todo |
| `test_empty_content_masks_remain_empty_when_snapped` | gpui::test | 10457 |  | `TestWindow_EmptyContentMasksRemainEmptyWhenSnapped` | todo |
| `test_nonempty_content_masks_keep_covering_fractional_edges` | gpui::test | 10503 |  | `TestWindow_NonemptyContentMasksKeepCoveringFractionalEdges` | todo |
| `test_empty_shadow_bounds_are_not_painted` | gpui::test | 10520 |  | `TestWindow_EmptyShadowBoundsAreNotPainted` | todo |
| `test_collapsed_inset_shadow_hole_is_an_empty_point_on_the_pixel_grid` | gpui::test | 10560 |  | `TestWindow_CollapsedInsetShadowHoleIsAnEmptyPointOnThePixelGrid` | todo |
| `test_standalone_modifier_sequence_timeout_and_recovery` | gpui::test | 10701 |  | `TestWindow_StandaloneModifierSequenceTimeoutAndRecovery` | todo |
| `test_standalone_modifier_sequence_dispatches_before_timeout` | gpui::test | 10754 |  | `TestWindow_StandaloneModifierSequenceDispatchesBeforeTimeout` | todo |
| `cached_view_follows_inherited_opacity` | gpui::test | 10823 |  | `TestWindow_CachedViewFollowsInheritedOpacity` | todo |
| `test_drag_cursor_applies_before_next_paint` | gpui::test | 10928 |  | `TestWindow_DragCursorAppliesBeforeNextPaint` | todo |
| `inspector_only_tracks_its_open_window` | gpui::test | 11017 | feat:inspector,debug | `TestWindowInspectorTests_InspectorOnlyTracksItsOpenWindow` | todo |

## gpui/src/window/a11y.rs

Go target: `window_a11y.go` (package `gpui`), tests in `window_a11y_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `A11ySubtreeBuilder` | struct | 304 |  | `A11ySubtreeBuilder` | todo |
| `A11ySubtreeBuilder::synthetic_node_id` | method | 335 |  | `(A11ySubtreeBuilder).SyntheticNodeID` | todo |
| `A11ySubtreeBuilder::push_child` | method | 346 |  | `(A11ySubtreeBuilder).PushChild` | todo |
| `A11ySubtreeBuilder::parent_node` | method | 364 |  | `(A11ySubtreeBuilder).ParentNode` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `accessibility_enabled_is_independent_of_activation` | test | 662 |  | `TestA11y_AccessibilityEnabledIsIndependentOfActivation` | todo |
| `active_descendant_honored_when_container_focused` | test | 675 |  | `TestA11y_ActiveDescendantHonoredWhenContainerFocused` | todo |
| `active_descendant_honored_for_deep_descendant` | test | 696 |  | `TestA11y_ActiveDescendantHonoredForDeepDescendant` | todo |
| `active_descendant_ignored_when_focus_in_other_subtree` | test | 720 |  | `TestA11y_ActiveDescendantIgnoredWhenFocusInOtherSubtree` | todo |
| `active_descendant_ignored_when_nothing_focused` | test | 747 |  | `TestA11y_ActiveDescendantIgnoredWhenNothingFocused` | todo |
| `regular_focus_used_when_no_active_descendant` | test | 766 |  | `TestA11y_RegularFocusUsedWhenNoActiveDescendant` | todo |
| `focus_is_ancestor_excludes_self_and_non_ancestors` | test | 779 |  | `TestA11y_FocusIsAncestorExcludesSelfAndNonAncestors` | todo |
| `multiple_active_descendant_claims_panic_in_debug` | test | 806 |  | `TestA11y_MultipleActiveDescendantClaimsPanicInDebug` | todo |
| `setting_focus_twice_panics_in_debug` | test | 819 |  | `TestA11y_SettingFocusTwicePanicsInDebug` | todo |
| `set_focus_without_set_focusable` | test | 832 |  | `TestA11y_SetFocusWithoutSetFocusable` | todo |
| `set_active_descendant_on_focused_node` | test | 844 |  | `TestA11y_SetActiveDescendantOnFocusedNode` | todo |
| `two_siblings_claiming_active_descendant` | test | 861 |  | `TestA11y_TwoSiblingsClaimingActiveDescendant` | todo |
| `active_descendant_in_unfocused_subtree_keeps_real_focus` | test | 885 |  | `TestA11y_ActiveDescendantInUnfocusedSubtreeKeepsRealFocus` | todo |

## gpui/src/window/native_view.rs

Go target: `window_native_view.go` (package `gpui`), tests in `window_native_view_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `NativeViewId` | struct | 20 |  | `NativeViewID` | todo |
| `NativeViewId::for_test` | assoc-fn | 25 | feat:test-support | `NativeViewIDForTest` | todo |
| `NativeView` | struct | 69 |  | `NativeView` | todo |
| `NativeView::new` | assoc-fn | 102 |  | `NewNativeView` | todo |
| `NativeView::set_background` | method | 130 |  | `(NativeView).SetBackground` | todo |
| `NativeView::reserve_action` | method | 148 |  | `(NativeView).ReserveAction` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `focusing_a_native_view_gives_it_keyboard_input` | gpui::test | 408 |  | `TestNativeView_FocusingANativeViewGivesItKeyboardInput` | todo |
| `platform_focus_of_a_native_view_focuses_its_handle` | gpui::test | 435 |  | `TestNativeView_PlatformFocusOfANativeViewFocusesItsHandle` | todo |
| `only_reserved_and_pending_bindings_apply_before_a_native_view` | gpui::test | 456 |  | `TestNativeView_OnlyReservedAndPendingBindingsApplyBeforeANativeView` | todo |
| `mouse_reaches_a_native_view_only_where_nothing_covers_it` | gpui::test | 534 |  | `TestNativeView_MouseReachesANativeViewOnlyWhereNothingCoversIt` | todo |
| `mouse_press_on_a_native_view_reaches_nothing_behind_it` | gpui::test | 551 |  | `TestNativeView_MousePressOnANativeViewReachesNothingBehindIt` | todo |
| `changing_the_background_repaints_the_native_view` | gpui::test | 570 |  | `TestNativeView_ChangingTheBackgroundRepaintsTheNativeView` | todo |
| `native_view_painted_twice_shows_where_it_was_painted_first` | gpui::test | 596 |  | `TestNativeView_NativeViewPaintedTwiceShowsWhereItWasPaintedFirst` | todo |
| `native_view_is_drawn_in_paint_order` | gpui::test | 641 |  | `TestNativeView_NativeViewIsDrawnInPaintOrder` | todo |
| `native_view_stays_attached_until_its_last_handle_is_dropped` | gpui::test | 661 |  | `TestNativeView_NativeViewStaysAttachedUntilItsLastHandleIsDropped` | todo |
| `native_view_shows_nothing_in_another_window` | gpui::test | 681 |  | `TestNativeView_NativeViewShowsNothingInAnotherWindow` | todo |
| `focusing_a_native_view_ends_a_composition` | gpui::test | 710 |  | `TestNativeView_FocusingANativeViewEndsAComposition` | todo |

## gpui/src/window/prompts.rs

Go target: `window_prompts.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PromptResponse` | struct | 16 |  | `PromptResponse` | todo |
| `Prompt` | trait | 19 |  | `Prompt` | todo |
| `PromptHandle` | struct | 24 |  | `PromptHandle` | todo |
| `PromptHandle::with_view` | method | 34 |  | `(PromptHandle).WithView` | todo |
| `RenderablePromptHandle` | struct | 67 |  | `RenderablePromptHandle` | todo |
| `fallback_prompt_renderer` | fn | 73 |  | `NewFallbackPromptRenderer` | todo |
| `FallbackPromptRenderer` | struct | 94 |  | `FallbackPromptRenderer` | todo |

