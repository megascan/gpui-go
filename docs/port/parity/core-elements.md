# Parity: core-elements

Scope: gpui: elements/*. Milestone: M0/M3. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/elements/anchored.rs

Go target: `elements_anchored.go` (package `gpui`), tests in `elements_anchored_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AnchoredState` | struct | 12 |  | `AnchoredState` | todo |
| `Anchored` | struct | 18 |  | `Anchored` | todo |
| `anchored` | fn | 39 |  | `NewAnchored` | todo |
| `Anchored::anchor` | method | 52 |  | `(Anchored).Anchor` | todo |
| `Anchored::position` | method | 59 |  | `(Anchored).Position` | todo |
| `Anchored::attach` | method | 72 |  | `(Anchored).Attach` | todo |
| `Anchored::offset` | method | 79 |  | `(Anchored).Offset` | todo |
| `Anchored::position_mode` | method | 88 |  | `(Anchored).PositionMode` | todo |
| `Anchored::snap_to_window` | method | 94 |  | `(Anchored).SnapToWindow` | todo |
| `Anchored::snap_to_window_with_margin` | method | 100 |  | `(Anchored).SnapToWindowWithMargin` | todo |
| `AnchoredFitMode` | enum | 329 |  | `AnchoredFitMode` | todo |
| `AnchoredPositionMode` | enum | 340 |  | `AnchoredPositionMode` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_anchored_position_without_scroll` | gpui::test | 416 |  | `TestAnchored_AnchoredPositionWithoutScroll` | todo |
| `test_anchored_position_when_scrolled` | gpui::test | 435 |  | `TestAnchored_AnchoredPositionWhenScrolled` | todo |
| `test_anchored_snaps_to_window` | gpui::test | 467 |  | `TestAnchored_AnchoredSnapsToWindow` | todo |
| `test_anchored_attach_moves_to_the_side_with_room` | gpui::test | 513 |  | `TestAnchored_AnchoredAttachMovesToTheSideWithRoom` | todo |

## gpui/src/elements/animation.rs

Go target: `elements_animation.go` (package `gpui`), tests in `elements_animation_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Animation` | struct | 15 |  | `Animation` | todo |
| `Animation::new` | assoc-fn | 36 |  | `NewAnimation` | todo |
| `Animation::repeat` | method | 48 |  | `(Animation).Repeat` | todo |
| `Animation::repeat_synced` | method | 54 |  | `(Animation).RepeatSynced` | todo |
| `Animation::with_easing` | method | 64 |  | `(Animation).WithEasing` | todo |
| `Animation::with_max_fps` | method | 73 |  | `(Animation).WithMaxFps` | todo |
| `Animation::with_steps` | method | 82 |  | `(Animation).WithSteps` | todo |
| `AnimationExt` | trait | 95 |  | `AnimationExt` | todo |
| `AnimationExt::with_animation` | trait-method | 97 |  | `AnimationExt.WithAnimation` | todo |
| `AnimationExt::with_animations` | trait-method | 115 |  | `AnimationExt.WithAnimations` | todo |
| `AnimationExt::with_spring` | trait-method | 137 |  | `AnimationExt.WithSpring` | todo |
| `AnimationElement` | struct | 174 |  | `AnimationElement` | todo |
| `SpringAnimationElement` | struct | 182 |  | `SpringAnimationElement` | todo |
| `SpringAnimationElement::map_element` | method | 206 |  | `(SpringAnimationElement).MapElement` | todo |
| `AnimationElement::map_element` | method | 225 |  | `(AnimationElement).MapElement` | todo |
| `linear` | fn | 555 |  | `Linear` | todo |
| `quadratic` | fn | 560 |  | `Quadratic` | todo |
| `ease_in_out` | fn | 565 |  | `EaseInOut` | todo |
| `ease_out_quint` | fn | 575 |  | `EaseOutQuint` | todo |
| `heartbeat` | fn | 580 |  | `Heartbeat` | todo |
| `bounce` | fn | 589 |  | `Bounce` | todo |
| `pulsating_between` | fn | 600 |  | `PulsatingBetween` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_animation_parent` | test | 738 |  | `TestAnimation_AnimationParent` | todo |
| `test_spring_animation_parent` | test | 757 |  | `TestAnimation_SpringAnimationParent` | todo |
| `test_spring_animation_preserves_velocity_when_retargeted` | gpui::test | 771 |  | `TestAnimation_SpringAnimationPreservesVelocityWhenRetargeted` | todo |
| `test_paused_spring_resumes_with_its_velocity` | gpui::test | 814 |  | `TestAnimation_PausedSpringResumesWithItsVelocity` | todo |
| `test_stopped_spring_resumes_without_velocity` | gpui::test | 865 |  | `TestAnimation_StoppedSpringResumesWithoutVelocity` | todo |
| `test_cancelled_and_completed_springs_resolve_their_endpoints` | gpui::test | 916 |  | `TestAnimation_CancelledAndCompletedSpringsResolveTheirEndpoints` | todo |
| `test_spring_animation_respects_reduced_motion` | gpui::test | 957 |  | `TestAnimation_SpringAnimationRespectsReducedMotion` | todo |
| `test_repeating_animation_schedules_animation_frames` | gpui::test | 976 |  | `TestAnimation_RepeatingAnimationSchedulesAnimationFrames` | todo |
| `test_max_fps_schedules_timer_driven_frames` | gpui::test | 988 |  | `TestAnimation_MaxFpsSchedulesTimerDrivenFrames` | todo |
| `test_synced_steps_render_once_each` | gpui::test | 1022 |  | `TestAnimation_SyncedStepsRenderOnceEach` | todo |
| `test_steps_finish_and_repeat_on_time` | gpui::test | 1040 |  | `TestAnimation_StepsFinishAndRepeatOnTime` | todo |
| `test_sooner_step_replaces_the_pending_one` | gpui::test | 1058 |  | `TestAnimation_SoonerStepReplacesThePendingOne` | todo |
| `test_synced_animations_share_phase_across_elements` | gpui::test | 1080 |  | `TestAnimation_SyncedAnimationsSharePhaseAcrossElements` | todo |
| `test_reduce_motion_renders_single_static_frame` | gpui::test | 1133 |  | `TestAnimation_ReduceMotionRendersSingleStaticFrame` | todo |

## gpui/src/elements/canvas.rs

Go target: `elements_canvas.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `canvas` | fn | 10 |  | `NewCanvas` | todo |
| `Canvas` | struct | 23 |  | `Canvas` | todo |

## gpui/src/elements/container_query.rs

Go target: `elements_container_query.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `container_query` | fn | 33 |  | `NewContainerQuery` | todo |
| `ContainerQuery` | struct | 52 |  | `ContainerQuery` | todo |

## gpui/src/elements/deferred.rs

Go target: `elements_deferred.go` (package `gpui`), tests in `elements_deferred_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `deferred` | fn | 7 |  | `NewDeferred` | todo |
| `Deferred` | struct | 16 |  | `Deferred` | todo |
| `Deferred::with_priority` | method | 25 |  | `(Deferred).WithPriority` | todo |
| `Deferred::priority` | method | 84 |  | `(Deferred).Priority` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_cached_reuse_keeps_entity_subscriptions` | gpui::test | 152 |  | `TestDeferred_CachedReuseKeepsEntitySubscriptions` | todo |
| `test_nested_deferred_draws_with_reused_views` | gpui::test | 261 |  | `TestDeferred_NestedDeferredDrawsWithReusedViews` | todo |

## gpui/src/elements/div.rs

Go target: `elements_div.go` (package `gpui`), tests in `elements_div_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `GroupStyle` | struct | 57 |  | `GroupStyle` | done |
| `DragMoveEvent` | struct | 67 |  | `DragMoveEvent` | done |
| `DragMoveEvent::drag` | method | 79 |  | `(DragMoveEvent[T]).Drag` | done |
| `DragMoveEvent::dragged_item` | method | 87 |  | `(DragMoveEvent[T]).DraggedItem` | done |
| `Interactivity::new` | assoc-fn | 96 | feat:inspector,debug | `NewInteractivity` | done |
| `Interactivity::new` | assoc-fn | 105 | feat:inspector,debug | `NewInteractivity` | done |
| `Interactivity::source_location` | method | 110 |  | `(Interactivity).SourceLocation` | done |
| `Interactivity::on_mouse_down` | method | 126 |  | `(Interactivity).OnMouseDown` | done |
| `Interactivity::capture_any_mouse_down` | method | 146 |  | `(Interactivity).CaptureAnyMouseDown` | done |
| `Interactivity::on_any_mouse_down` | method | 162 |  | `(Interactivity).OnAnyMouseDown` | done |
| `Interactivity::on_mouse_pressure` | method | 178 |  | `(Interactivity).OnMousePressure` | done |
| `Interactivity::capture_mouse_pressure` | method | 194 |  | `(Interactivity).CaptureMousePressure` | done |
| `Interactivity::on_mouse_up` | method | 210 |  | `(Interactivity).OnMouseUp` | done |
| `Interactivity::capture_any_mouse_up` | method | 230 |  | `(Interactivity).CaptureAnyMouseUp` | done |
| `Interactivity::on_any_mouse_up` | method | 246 |  | `(Interactivity).OnAnyMouseUp` | done |
| `Interactivity::on_mouse_down_out` | method | 263 |  | `(Interactivity).OnMouseDownOut` | done |
| `Interactivity::on_mouse_up_out` | method | 283 |  | `(Interactivity).OnMouseUpOut` | done |
| `Interactivity::on_mouse_move` | method | 303 |  | `(Interactivity).OnMouseMove` | done |
| `Interactivity::on_mouse_exit` | method | 319 |  | `(Interactivity).OnMouseExit` | done |
| `Interactivity::on_file_drop_exit` | method | 338 |  | `(Interactivity).OnFileDropExit` | done |
| `Interactivity::on_drag_move` | method | 360 |  | `(Interactivity).OnDragMove` | done |
| `Interactivity::on_scroll` | method | 390 |  | `(Interactivity).OnScroll` | done |
| `Interactivity::on_scroll_wheel` | method | 401 |  | `(Interactivity).OnScrollWheel` | done |
| `Interactivity::on_pinch` | method | 411 |  | `(Interactivity).OnPinch` | done |
| `Interactivity::on_back_gesture` | method | 424 |  | `(Interactivity).OnBackGesture` | done |
| `Interactivity::capture_pinch` | method | 441 |  | `(Interactivity).CapturePinch` | done |
| `Interactivity::capture_action` | method | 459 |  | `(Interactivity).CaptureAction` | done |
| `Interactivity::on_action` | method | 482 |  | `(Interactivity).OnAction` | done |
| `Interactivity::on_boxed_action` | method | 500 |  | `(Interactivity).OnBoxedAction` | done |
| `Interactivity::on_key_down` | method | 520 |  | `(Interactivity).OnKeyDown` | done |
| `Interactivity::capture_key_down` | method | 536 |  | `(Interactivity).CaptureKeyDown` | done |
| `Interactivity::on_key_up` | method | 552 |  | `(Interactivity).OnKeyUp` | done |
| `Interactivity::capture_key_up` | method | 565 |  | `(Interactivity).CaptureKeyUp` | done |
| `Interactivity::on_modifiers_changed` | method | 581 |  | `(Interactivity).OnModifiersChanged` | done |
| `Interactivity::on_drop` | method | 592 |  | `(Interactivity).OnDrop` | done |
| `Interactivity::can_drop` | method | 603 |  | `(Interactivity).CanDrop` | done |
| `Interactivity::on_click` | method | 615 |  | `(Interactivity).OnClick` | done |
| `Interactivity::on_aux_click` | method | 626 |  | `(Interactivity).OnAuxClick` | done |
| `Interactivity::on_drag` | method | 639 |  | `(Interactivity).OnDrag` | done |
| `Interactivity::external_drag_payload` | method | 665 |  | `(Interactivity).ExternalDragPayload` | done |
| `Interactivity::on_hover` | method | 699 |  | `(Interactivity).OnHover` | done |
| `Interactivity::hover_listener_mode` | method | 713 |  | `(Interactivity).HoverListenerMode` | done |
| `Interactivity::tooltip` | method | 722 |  | `(Interactivity).Tooltip` | done |
| `Interactivity::hoverable_tooltip` | method | 739 |  | `(Interactivity).HoverableTooltip` | done |
| `Interactivity::tooltip_show_delay` | method | 759 |  | `(Interactivity).TooltipShowDelay` | done |
| `Interactivity::occlude_mouse` | method | 767 |  | `(Interactivity).OccludeMouse` | done |
| `Interactivity::window_control_area` | method | 773 |  | `(Interactivity).WindowControlArea` | done |
| `Interactivity::block_mouse_except_scroll` | method | 781 |  | `(Interactivity).BlockMouseExceptScroll` | done |
| `InteractiveElement` | trait | 792 |  | `Interactive[S] mixin + InteractiveElement interface` | done |
| `InteractiveElement::interactivity` | trait-method | 794 |  | `(Div).Interactivity` | done |
| `InteractiveElement::group` | trait-method | 797 |  | `(Interactive[S]).Group` | done |
| `InteractiveElement::id` | trait-method | 803 |  | `(InteractiveID[S]).ID` | dev:D-IDMIXIN |
| `InteractiveElement::track_focus` | trait-method | 812 |  | `(Interactive[S]).TrackFocus` | done |
| `InteractiveElement::tab_stop` | trait-method | 824 |  | `(Interactive[S]).TabStop` | done |
| `InteractiveElement::tab_index` | trait-method | 833 |  | `(Interactive[S]).TabIndex` | done |
| `InteractiveElement::tab_group` | trait-method | 844 |  | `(Interactive[S]).TabGroup` | done |
| `InteractiveElement::key_context` | trait-method | 854 |  | `(Interactive[S]).KeyContext` | done |
| `InteractiveElement::hover` | trait-method | 866 |  | `(Interactive[S]).Hover` | done |
| `InteractiveElement::group_hover` | trait-method | 876 |  | `(Interactive[S]).GroupHover` | done |
| `InteractiveElement::on_mouse_down` | trait-method | 893 |  | `(Interactive[S]).OnMouseDown` | done |
| `InteractiveElement::debug_selector` | trait-method | 909 | feat:test-support,debug | `(Interactive[S]).DebugSelector` | done |
| `InteractiveElement::debug_selector` | trait-method | 922 | feat:test-support,debug | `(Interactive[S]).DebugSelector` | done |
| `InteractiveElement::capture_any_mouse_down` | trait-method | 930 |  | `(Interactive[S]).CaptureAnyMouseDown` | done |
| `InteractiveElement::on_any_mouse_down` | trait-method | 942 |  | `(Interactive[S]).OnAnyMouseDown` | done |
| `InteractiveElement::on_mouse_up` | trait-method | 954 |  | `(Interactive[S]).OnMouseUp` | done |
| `InteractiveElement::capture_any_mouse_up` | trait-method | 967 |  | `(Interactive[S]).CaptureAnyMouseUp` | done |
| `InteractiveElement::on_mouse_pressure` | trait-method | 979 |  | `(Interactive[S]).OnMousePressure` | done |
| `InteractiveElement::capture_mouse_pressure` | trait-method | 991 |  | `(Interactive[S]).CaptureMousePressure` | done |
| `InteractiveElement::on_mouse_down_out` | trait-method | 1004 |  | `(Interactive[S]).OnMouseDownOut` | done |
| `InteractiveElement::on_mouse_up_out` | trait-method | 1017 |  | `(Interactive[S]).OnMouseUpOut` | done |
| `InteractiveElement::on_mouse_move` | trait-method | 1030 |  | `(Interactive[S]).OnMouseMove` | done |
| `InteractiveElement::on_mouse_exit` | trait-method | 1042 |  | `(Interactive[S]).OnMouseExit` | done |
| `InteractiveElement::on_file_drop_exit` | trait-method | 1057 |  | `(Interactive[S]).OnFileDropExit` | done |
| `InteractiveElement::on_drag_move` | trait-method | 1072 |  | `(Interactive[S]).OnDragMove` | done |
| `InteractiveElement::on_scroll` | trait-method | 1084 |  | `(Interactive[S]).OnScroll` | done |
| `InteractiveElement::on_scroll_wheel` | trait-method | 1094 |  | `(Interactive[S]).OnScrollWheel` | done |
| `InteractiveElement::on_pinch` | trait-method | 1105 |  | `(Interactive[S]).OnPinch` | done |
| `InteractiveElement::on_back_gesture` | trait-method | 1116 |  | `(Interactive[S]).OnBackGesture` | done |
| `InteractiveElement::capture_pinch` | trait-method | 1128 |  | `(Interactive[S]).CapturePinch` | done |
| `InteractiveElement::capture_action` | trait-method | 1139 |  | `(Interactive[S]).CaptureAction` | done |
| `InteractiveElement::on_action` | trait-method | 1153 |  | `(Interactive[S]).OnAction` | done |
| `InteractiveElement::on_boxed_action` | trait-method | 1167 |  | `(Interactive[S]).OnBoxedAction` | done |
| `InteractiveElement::on_key_down` | trait-method | 1180 |  | `(Interactive[S]).OnKeyDown` | done |
| `InteractiveElement::capture_key_down` | trait-method | 1192 |  | `(Interactive[S]).CaptureKeyDown` | done |
| `InteractiveElement::on_key_up` | trait-method | 1204 |  | `(Interactive[S]).OnKeyUp` | done |
| `InteractiveElement::capture_key_up` | trait-method | 1216 |  | `(Interactive[S]).CaptureKeyUp` | done |
| `InteractiveElement::on_modifiers_changed` | trait-method | 1229 |  | `(Interactive[S]).OnModifiersChanged` | done |
| `InteractiveElement::drag_over` | trait-method | 1238 |  | `(Interactive[S]).DragOver` | done |
| `InteractiveElement::group_drag_over` | trait-method | 1257 |  | `(Interactive[S]).GroupDragOver` | done |
| `InteractiveElement::on_drop` | trait-method | 1276 |  | `(Interactive[S]).OnDrop` | done |
| `InteractiveElement::can_drop` | trait-method | 1286 |  | `(Interactive[S]).CanDrop` | done |
| `InteractiveElement::occlude` | trait-method | 1297 |  | `(Interactive[S]).Occlude` | done |
| `InteractiveElement::window_control_area` | trait-method | 1304 |  | `(Interactive[S]).WindowControlArea` | done |
| `InteractiveElement::block_mouse_except_scroll` | trait-method | 1313 |  | `(Interactive[S]).BlockMouseExceptScroll` | done |
| `InteractiveElement::focus` | trait-method | 1320 |  | `(Interactive[S]).Focus` | done |
| `InteractiveElement::in_focus` | trait-method | 1330 |  | `(Interactive[S]).InFocus` | done |
| `InteractiveElement::focus_visible` | trait-method | 1342 |  | `(Interactive[S]).FocusVisible` | done |
| `StatefulInteractiveElement` | trait | 1353 |  | `StatefulInteractive[S] mixin + StatefulInteractiveElement interface` | done |
| `StatefulInteractiveElement::role` | trait-method | 1357 |  | `(StatefulInteractive[S]).Role` | done |
| `StatefulInteractiveElement::accessibility_id` | trait-method | 1373 |  | `(StatefulInteractive[S]).AccessibilityID` | done |
| `StatefulInteractiveElement::aria_label` | trait-method | 1379 |  | `(StatefulInteractive[S]).AriaLabel` | done |
| `StatefulInteractiveElement::aria_description` | trait-method | 1388 |  | `(StatefulInteractive[S]).AriaDescription` | done |
| `StatefulInteractiveElement::aria_keyshortcuts` | trait-method | 1398 |  | `(StatefulInteractive[S]).AriaKeyshortcuts` | done |
| `StatefulInteractiveElement::aria_active_descendant` | trait-method | 1418 |  | `(StatefulInteractive[S]).AriaActiveDescendant` | done |
| `StatefulInteractiveElement::a11y_synthetic_children` | trait-method | 1432 |  | `(StatefulInteractive[S]).A11ySyntheticChildren` | done |
| `StatefulInteractiveElement::aria_selected` | trait-method | 1441 |  | `(StatefulInteractive[S]).AriaSelected` | done |
| `StatefulInteractiveElement::aria_expanded` | trait-method | 1447 |  | `(StatefulInteractive[S]).AriaExpanded` | done |
| `StatefulInteractiveElement::aria_toggled` | trait-method | 1453 |  | `(StatefulInteractive[S]).AriaToggled` | done |
| `StatefulInteractiveElement::aria_numeric_value` | trait-method | 1459 |  | `(StatefulInteractive[S]).AriaNumericValue` | done |
| `StatefulInteractiveElement::aria_numeric_value_step` | trait-method | 1466 |  | `(StatefulInteractive[S]).AriaNumericValueStep` | done |
| `StatefulInteractiveElement::aria_value` | trait-method | 1473 |  | `(StatefulInteractive[S]).AriaValue` | done |
| `StatefulInteractiveElement::aria_placeholder` | trait-method | 1480 |  | `(StatefulInteractive[S]).AriaPlaceholder` | done |
| `StatefulInteractiveElement::aria_min_numeric_value` | trait-method | 1486 |  | `(StatefulInteractive[S]).AriaMinNumericValue` | done |
| `StatefulInteractiveElement::aria_max_numeric_value` | trait-method | 1492 |  | `(StatefulInteractive[S]).AriaMaxNumericValue` | done |
| `StatefulInteractiveElement::aria_orientation` | trait-method | 1498 |  | `(StatefulInteractive[S]).AriaOrientation` | done |
| `StatefulInteractiveElement::aria_level` | trait-method | 1504 |  | `(StatefulInteractive[S]).AriaLevel` | done |
| `StatefulInteractiveElement::aria_position_in_set` | trait-method | 1510 |  | `(StatefulInteractive[S]).AriaPositionInSet` | done |
| `StatefulInteractiveElement::aria_size_of_set` | trait-method | 1516 |  | `(StatefulInteractive[S]).AriaSizeOfSet` | done |
| `StatefulInteractiveElement::aria_row_index` | trait-method | 1522 |  | `(StatefulInteractive[S]).AriaRowIndex` | done |
| `StatefulInteractiveElement::aria_column_index` | trait-method | 1528 |  | `(StatefulInteractive[S]).AriaColumnIndex` | done |
| `StatefulInteractiveElement::aria_row_count` | trait-method | 1534 |  | `(StatefulInteractive[S]).AriaRowCount` | done |
| `StatefulInteractiveElement::aria_column_count` | trait-method | 1540 |  | `(StatefulInteractive[S]).AriaColumnCount` | done |
| `StatefulInteractiveElement::on_a11y_action` | trait-method | 1550 |  | `(StatefulInteractive[S]).OnA11yAction` | done |
| `StatefulInteractiveElement::focusable` | trait-method | 1563 |  | `(StatefulInteractive[S]).Focusable` | done |
| `StatefulInteractiveElement::overflow_scroll` | trait-method | 1569 |  | `(StatefulInteractive[S]).OverflowScroll` | done |
| `StatefulInteractiveElement::overflow_x_scroll` | trait-method | 1576 |  | `(StatefulInteractive[S]).OverflowXScroll` | done |
| `StatefulInteractiveElement::overflow_y_scroll` | trait-method | 1582 |  | `(StatefulInteractive[S]).OverflowYScroll` | done |
| `StatefulInteractiveElement::restrict_scroll_to_axis` | trait-method | 1590 |  | `(StatefulInteractive[S]).RestrictScrollToAxis` | done |
| `StatefulInteractiveElement::track_scroll` | trait-method | 1596 |  | `(StatefulInteractive[S]).TrackScroll` | done |
| `StatefulInteractiveElement::anchor_scroll` | trait-method | 1602 |  | `(StatefulInteractive[S]).AnchorScroll` | done |
| `StatefulInteractiveElement::active` | trait-method | 1608 |  | `(StatefulInteractive[S]).Active` | done |
| `StatefulInteractiveElement::group_active` | trait-method | 1617 |  | `(StatefulInteractive[S]).GroupActive` | done |
| `StatefulInteractiveElement::on_click` | trait-method | 1637 |  | `(StatefulInteractive[S]).OnClick` | done |
| `StatefulInteractiveElement::on_aux_click` | trait-method | 1649 |  | `(StatefulInteractive[S]).OnAuxClick` | done |
| `StatefulInteractiveElement::on_drag` | trait-method | 1667 |  | `(StatefulInteractive[S]).OnDrag` | done |
| `StatefulInteractiveElement::external_drag_payload` | trait-method | 1686 |  | `(StatefulInteractive[S]).ExternalDragPayload` | done |
| `StatefulInteractiveElement::on_hover` | trait-method | 1708 |  | `(StatefulInteractive[S]).OnHover` | done |
| `StatefulInteractiveElement::hover_listener_mode` | trait-method | 1719 |  | `(StatefulInteractive[S]).HoverListenerMode` | done |
| `StatefulInteractiveElement::tooltip` | trait-method | 1729 |  | `(StatefulInteractive[S]).Tooltip` | done |
| `StatefulInteractiveElement::hoverable_tooltip` | trait-method | 1740 |  | `(StatefulInteractive[S]).HoverableTooltip` | done |
| `StatefulInteractiveElement::tooltip_show_delay` | trait-method | 1755 |  | `(StatefulInteractive[S]).TooltipShowDelay` | done |
| `HoverListenerMode` | enum | 1792 |  | `HoverListenerMode` | done |
| `div` | fn | 1844 |  | `NewDiv` | done |
| `Div` | struct | 1905 |  | `Div` | done |
| `Div::on_children_prepainted` | method | 1918 |  | `(Div).OnChildrenPrepainted` | done |
| `Div::image_cache` | method | 1927 |  | `(Div).ImageCache` | done |
| `Div::with_dynamic_prepaint_order` | method | 1940 |  | `(Div).WithDynamicPrepaintOrder` | done |
| `DivFrameState` | struct | 1955 |  | `DivFrameState` | done |
| `DivInspectorState` | struct | 1961 |  | `DivInspectorState` | done |
| `Interactivity` | struct | 2223 |  | `Interactivity` | done |
| `Interactivity::request_layout` | method | 2300 |  | `(Interactivity).RequestLayout` | todo |
| `Interactivity::prepaint` | method | 2393 |  | `(Interactivity).Prepaint` | todo |
| `Interactivity::paint` | method | 2609 |  | `(Interactivity).Paint` | todo |
| `Interactivity::compute_style` | method | 3499 |  | `(Interactivity).ComputeStyle` | todo |
| `InteractiveElementState` | struct | 3718 |  | `InteractiveElementState` | todo |
| `ElementClickedState` | struct | 3740 |  | `ElementClickedState` | todo |
| `ElementHoverState` | struct | 3756 |  | `ElementHoverState` | todo |
| `Stateful` | struct | 4176 |  | `Stateful` | todo |
| `ScrollAnchor` | struct | 4293 |  | `ScrollAnchor` | todo |
| `ScrollAnchor::for_handle` | assoc-fn | 4300 |  | `ScrollAnchorForHandle` | todo |
| `ScrollAnchor::scroll_to` | method | 4307 |  | `(ScrollAnchor).ScrollTo` | todo |
| `ScrollHandle` | struct | 4347 |  | `ScrollHandle` | todo |
| `ScrollHandle::new` | assoc-fn | 4357 |  | `NewScrollHandle` | todo |
| `ScrollHandle::offset` | method | 4362 |  | `(ScrollHandle).Offset` | todo |
| `ScrollHandle::max_offset` | method | 4367 |  | `(ScrollHandle).MaxOffset` | todo |
| `ScrollHandle::top_item` | method | 4372 |  | `(ScrollHandle).TopItem` | todo |
| `ScrollHandle::bottom_item` | method | 4391 |  | `(ScrollHandle).BottomItem` | todo |
| `ScrollHandle::bounds` | method | 4410 |  | `(ScrollHandle).Bounds` | todo |
| `ScrollHandle::bounds_for_item` | method | 4415 |  | `(ScrollHandle).BoundsForItem` | todo |
| `ScrollHandle::scroll_to_item` | method | 4420 |  | `(ScrollHandle).ScrollToItem` | todo |
| `ScrollHandle::scroll_to_top_of_item` | method | 4430 |  | `(ScrollHandle).ScrollToTopOfItem` | todo |
| `ScrollHandle::scroll_to_bottom` | method | 4490 |  | `(ScrollHandle).ScrollToBottom` | todo |
| `ScrollHandle::set_offset` | method | 4498 |  | `(ScrollHandle).SetOffset` | todo |
| `ScrollHandle::logical_scroll_top` | method | 4504 |  | `(ScrollHandle).LogicalScrollTop` | todo |
| `ScrollHandle::logical_scroll_bottom` | method | 4519 |  | `(ScrollHandle).LogicalScrollBottom` | todo |
| `ScrollHandle::children_count` | method | 4534 |  | `(ScrollHandle).ChildrenCount` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `group_hover_styles_update_only_on_transitions` | gpui::test | 4598 |  | `TestDiv_GroupHoverStylesUpdateOnlyOnTransitions` | todo |
| `default_hover_listener_updates_when_layout_changes_under_stationary_mouse` | gpui::test | 4672 |  | `TestDiv_DefaultHoverListenerUpdatesWhenLayoutChangesUnderStationaryMouse` | todo |
| `default_hover_listener_ends_after_key_press` | gpui::test | 4714 |  | `TestDiv_DefaultHoverListenerEndsAfterKeyPress` | todo |
| `input_modality_independent_hover_listener_updates_after_key_press` | gpui::test | 4762 |  | `TestDiv_InputModalityIndependentHoverListenerUpdatesAfterKeyPress` | todo |
| `default_hover_listener_remains_hovered_during_stationary_mouse_press` | gpui::test | 4819 |  | `TestDiv_DefaultHoverListenerRemainsHoveredDuringStationaryMousePress` | todo |
| `scroll_handle_aligns_wide_children_to_left_edge` | test | 4975 |  | `TestDiv_ScrollHandleAlignsWideChildrenToLeftEdge` | todo |
| `scroll_handle_aligns_tall_children_to_top_edge` | test | 4994 |  | `TestDiv_ScrollHandleAlignsTallChildrenToTopEdge` | todo |
| `long_press_shows_tooltip_until_touch_ends` | test | 5013 |  | `TestDiv_LongPressShowsTooltipUntilTouchEnds` | todo |
| `tooltip_waiting_for_show_is_released_when_its_owner_disappears` | test | 5151 |  | `TestDiv_TooltipWaitingForShowIsReleasedWhenItsOwnerDisappears` | todo |
| `tooltip_respects_custom_show_delay` | test | 5173 |  | `TestDiv_TooltipRespectsCustomShowDelay` | todo |
| `tooltip_is_released_when_its_owner_disappears` | test | 5202 |  | `TestDiv_TooltipIsReleasedWhenItsOwnerDisappears` | todo |
| `tooltip_hides_after_mouse_leaves_origin` | test | 5230 |  | `TestDiv_TooltipHidesAfterMouseLeavesOrigin` | todo |
| `mouse_down_out_is_suppressed_while_window_prompt_is_active` | test | 5281 |  | `TestDiv_MouseDownOutIsSuppressedWhileWindowPromptIsActive` | todo |
| `touch_tap_out_fires_once_after_release` | test | 5345 |  | `TestDiv_TouchTapOutFiresOnceAfterRelease` | todo |
| `touch_pan_and_cancel_do_not_fire_mouse_down_out` | crate::test | 5387 |  | `TestDiv_TouchPanAndCancelDoNotFireMouseDownOut` | todo |
| `test_accessibility_id_builder_writes_author_id` | test | 5435 |  | `TestDiv_AccessibilityIDBuilderWritesAuthorID` | todo |
| `test_write_a11y_info_string_and_numeric_properties` | test | 5447 |  | `TestDiv_WriteA11yInfoStringAndNumericProperties` | todo |
| `keyboard_activation_fires_click_on_same_element` | test | 5567 |  | `TestDiv_KeyboardActivationFiresClickOnSameElement` | todo |
| `keyboard_activation_does_not_leak_across_focus_change` | test | 5582 |  | `TestDiv_KeyboardActivationDoesNotLeakAcrossFocusChange` | todo |
| `keyboard_activation_does_not_leak_when_focus_returns` | test | 5603 |  | `TestDiv_KeyboardActivationDoesNotLeakWhenFocusReturns` | todo |
| `keyboard_activation_cleared_by_intervening_key_release` | test | 5625 |  | `TestDiv_KeyboardActivationClearedByInterveningKeyRelease` | todo |
| `keyboard_activation_does_not_distinguish_space_and_enter` | test | 5641 |  | `TestDiv_KeyboardActivationDoesNotDistinguishSpaceAndEnter` | todo |
| `keyboard_activation_cleared_by_intervening_keydown` | test | 5654 |  | `TestDiv_KeyboardActivationClearedByInterveningKeydown` | todo |
| `keyboard_activation_ignores_modified_keys` | test | 5668 |  | `TestDiv_KeyboardActivationIgnoresModifiedKeys` | todo |
| `focus_next_from_tab_group_container_enters_that_group` | test | 5706 |  | `TestDiv_FocusNextFromTabGroupContainerEntersThatGroup` | todo |
| `test_fractional_padding_does_not_make_a_fitting_container_scrollable` | gpui::test | 5744 |  | `TestDiv_FractionalPaddingDoesNotMakeAFittingContainerScrollable` | todo |
| `test_scrolls_to_an_item_requested_before_the_first_frame` | gpui::test | 5786 |  | `TestDiv_ScrollsToAnItemRequestedBeforeTheFirstFrame` | todo |
| `grid_cols_max_content_sizes_columns_to_their_content` | gpui::test | 5848 |  | `TestDiv_GridColsMaxContentSizesColumnsToTheirContent` | todo |

## gpui/src/elements/image_cache.rs

Go target: `elements_image_cache.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `image_cache` | fn | 14 |  | `NewImageCache` | todo |
| `AnyImageCache` | struct | 24 |  | `AnyImageCache` | todo |
| `AnyImageCache::load` | method | 46 |  | `(AnyImageCache).Load` | todo |
| `ImageCacheElement` | struct | 71 |  | `ImageCacheElement` | todo |
| `ImageCacheItem` | struct | 159 |  | `ImageCacheItem` | todo |
| `ImageCacheItem::new` | assoc-fn | 171 |  | `NewImageCacheItem` | todo |
| `ImageCacheItem::get` | method | 177 |  | `(ImageCacheItem).Get` | todo |
| `ImageCacheItem::use_image` | method | 182 |  | `(ImageCacheItem).UseImage` | todo |
| `ImageCache` | trait | 189 |  | `ImageCache` | todo |
| `ImageCache::load` | trait-method | 192 |  | `ImageCache.Load` | todo |
| `ImageCacheProvider` | trait | 202 |  | `ImageCacheProvider` | todo |
| `ImageCacheProvider::provide` | trait-method | 204 |  | `ImageCacheProvider.Provide` | todo |
| `RetainAllImageCache` | struct | 214 |  | `RetainAllImageCache` | todo |
| `RetainAllImageCache::new` | assoc-fn | 227 |  | `NewRetainAllImageCache` | todo |
| `RetainAllImageCache::load` | method | 243 |  | `(RetainAllImageCache).Load` | todo |
| `RetainAllImageCache::clear` | method | 258 |  | `(RetainAllImageCache).Clear` | todo |
| `RetainAllImageCache::remove` | method | 267 |  | `(RetainAllImageCache).Remove` | todo |
| `RetainAllImageCache::len` | method | 277 |  | `(RetainAllImageCache).Len` | todo |
| `RetainAllImageCache::is_empty` | method | 282 |  | `(RetainAllImageCache).IsEmpty` | todo |
| `retain_all` | fn | 299 |  | `RetainAll` | todo |
| `RetainAllImageCacheProvider` | struct | 304 |  | `RetainAllImageCacheProvider` | todo |

## gpui/src/elements/img.rs

Go target: `elements_img.go` (package `gpui`), tests in `elements_img_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LOADING_DELAY` | const | 28 |  | `LoadingDelay` | todo |
| `ImgResourceLoader` | type | 34 |  | `ImgResourceLoader` | todo |
| `ImageSource` | enum | 38 |  | `ImageSource` | todo |
| `ImageStyle` | struct | 125 |  | `ImageStyle` | todo |
| `StyledImage` | trait | 144 |  | `StyledImage` | todo |
| `StyledImage::image_style` | trait-method | 146 |  | `StyledImage.ImageStyle` | todo |
| `StyledImage::grayscale` | trait-method | 149 |  | `StyledImage.Grayscale` | todo |
| `StyledImage::object_fit` | trait-method | 155 |  | `StyledImage.ObjectFit` | todo |
| `StyledImage::with_fallback` | trait-method | 162 |  | `StyledImage.WithFallback` | todo |
| `StyledImage::with_loading` | trait-method | 169 |  | `StyledImage.WithLoading` | todo |
| `Img` | struct | 188 |  | `Img` | todo |
| `img` | fn | 197 |  | `NewImg` | todo |
| `Img::extensions` | assoc-fn | 208 |  | `ImgExtensions` | todo |
| `Img::image_cache` | method | 226 |  | `(Img).ImageCache` | todo |
| `ImgLayoutState` | struct | 256 |  | `ImgLayoutState` | todo |
| `ImageSource::remove_asset` | method | 565 |  | `(ImageSource).RemoveAsset` | todo |
| `ImageSource::is_asset_cached` | method | 578 | feat:test-support | `(ImageSource).IsAssetCached` | todo |
| `ImageAssetLoader` | enum | 605 |  | `ImageAssetLoader` | todo |
| `ImageCacheError` | enum | 676 |  | `ImageCacheError` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `zero_frame_image_does_not_panic_on_paint` | gpui::test | 766 |  | `TestImg_ZeroFrameImageDoesNotPanicOnPaint` | todo |
| `image_object_fit_cover_crops_to_element_bounds` | gpui::test | 774 |  | `TestImg_ImageObjectFitCoverCropsToElementBounds` | todo |
| `explicit_aspect_ratio_is_not_overridden_by_intrinsic_ratio` | gpui::test | 829 |  | `TestImg_ExplicitAspectRatioIsNotOverriddenByIntrinsicRatio` | todo |
| `image_object_fit_cover_clamps_corner_radii_to_visible_bounds` | gpui::test | 867 |  | `TestImg_ImageObjectFitCoverClampsCornerRadiiToVisibleBounds` | todo |
| `stale_frame_index_is_clamped_when_image_changes` | gpui::test | 892 |  | `TestImg_StaleFrameIndexIsClampedWhenImageChanges` | todo |

## gpui/src/elements/list.rs

Go target: `elements_list.go` (package `gpui`), tests in `elements_list_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `list` | fn | 24 |  | `NewList` | todo |
| `List` | struct | 37 |  | `List` | todo |
| `List::with_sizing_behavior` | method | 46 |  | `(List).WithSizingBehavior` | todo |
| `ListState` | struct | 54 |  | `ListState` | todo |
| `FollowMode` | enum | 98 |  | `FollowMode` | todo |
| `ListAlignment` | enum | 149 |  | `ListAlignment` | todo |
| `ListScrollEvent` | struct | 157 |  | `ListScrollEvent` | todo |
| `ListSizingBehavior` | enum | 173 |  | `ListSizingBehavior` | todo |
| `ListMeasuringBehavior` | enum | 183 |  | `ListMeasuringBehavior` | todo |
| `ListHorizontalSizingBehavior` | enum | 203 |  | `ListHorizontalSizingBehavior` | todo |
| `ListPrepaintState` | struct | 224 |  | `ListPrepaintState` | todo |
| `ListState::new` | assoc-fn | 313 |  | `NewListState` | todo |
| `ListState::measure_all` | method | 335 |  | `(ListState).MeasureAll` | todo |
| `ListState::with_uniform_item_height` | method | 346 |  | `(ListState).WithUniformItemHeight` | todo |
| `ListState::reset` | method | 354 |  | `(ListState).Reset` | todo |
| `ListState::reset_with_uniform_height` | method | 370 |  | `(ListState).ResetWithUniformHeight` | todo |
| `ListState::remeasure` | method | 398 |  | `(ListState).Remeasure` | todo |
| `ListState::remeasure_items` | method | 410 |  | `(ListState).RemeasureItems` | todo |
| `ListState::item_count` | method | 464 |  | `(ListState).ItemCount` | todo |
| `ListState::is_scrolled_to_end` | method | 470 |  | `(ListState).IsScrolledToEnd` | todo |
| `ListState::splice` | method | 489 |  | `(ListState).Splice` | todo |
| `ListState::splice_focusable` | method | 497 |  | `(ListState).SpliceFocusable` | todo |
| `ListState::set_scroll_handler` | method | 541 |  | `(ListState).SetScrollHandler` | todo |
| `ListState::logical_scroll_top` | method | 549 |  | `(ListState).LogicalScrollTop` | todo |
| `ListState::scroll_by` | method | 554 |  | `(ListState).ScrollBy` | todo |
| `ListState::scroll_to_end` | method | 591 |  | `(ListState).ScrollToEnd` | todo |
| `ListState::set_follow_mode` | method | 604 |  | `(ListState).SetFollowMode` | todo |
| `ListState::pause_following_tail` | method | 636 |  | `(ListState).PauseFollowingTail` | todo |
| `ListState::set_item_measurement_readiness` | method | 649 |  | `(ListState).SetItemMeasurementReadiness` | todo |
| `ListState::is_following_tail` | method | 655 |  | `(ListState).IsFollowingTail` | todo |
| `ListState::scroll_to` | method | 663 |  | `(ListState).ScrollTo` | todo |
| `ListState::scroll_to_reveal_item` | method | 679 |  | `(ListState).ScrollToRevealItem` | todo |
| `ListState::bounds_for_item` | method | 712 |  | `(ListState).BoundsForItem` | todo |
| `ListState::scrollbar_drag_started` | method | 745 |  | `(ListState).ScrollbarDragStarted` | todo |
| `ListState::scrollbar_drag_ended` | method | 753 |  | `(ListState).ScrollbarDragEnded` | todo |
| `ListState::is_scrollbar_dragging` | method | 763 |  | `(ListState).IsScrollbarDragging` | todo |
| `ListState::set_offset_from_scrollbar` | method | 768 |  | `(ListState).SetOffsetFromScrollbar` | todo |
| `ListState::max_offset_for_scrollbar` | method | 774 |  | `(ListState).MaxOffsetForScrollbar` | todo |
| `ListState::scroll_px_offset_for_scrollbar` | method | 783 |  | `(ListState).ScrollPxOffsetForScrollbar` | todo |
| `ListState::viewport_bounds` | method | 801 |  | `(ListState).ViewportBounds` | todo |
| `ListState::item_is_above_viewport` | method | 812 |  | `(ListState).ItemIsAboveViewport` | todo |
| `ListState::item_is_below_viewport` | method | 831 |  | `(ListState).ItemIsBelowViewport` | todo |
| `ListOffset` | struct | 1448 |  | `ListOffset` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_autoscroll_above_item_top_renders_items_above` | gpui::test | 1762 |  | `TestList_AutoscrollAboveItemTopRendersItemsAbove` | todo |
| `test_reset_after_paint_before_scroll` | gpui::test | 1815 |  | `TestList_ResetAfterPaintBeforeScroll` | todo |
| `test_scroll_by_positive_and_negative_distance` | gpui::test | 1858 |  | `TestList_ScrollByPositiveAndNegativeDistance` | todo |
| `test_child_scroll_handler_can_stop_list_scroll` | gpui::test | 1903 |  | `TestList_ChildScrollHandlerCanStopListScroll` | todo |
| `test_item_viewport_queries_return_none_before_layout` | gpui::test | 1957 |  | `TestList_ItemViewportQueriesReturnNoneBeforeLayout` | todo |
| `test_item_viewport_queries_before_logical_scroll_top` | gpui::test | 1965 |  | `TestList_ItemViewportQueriesBeforeLogicalScrollTop` | todo |
| `test_item_viewport_queries_measured_item_inside_viewport` | gpui::test | 1983 |  | `TestList_ItemViewportQueriesMeasuredItemInsideViewport` | todo |
| `test_item_viewport_queries_measured_item_above_viewport` | gpui::test | 2001 |  | `TestList_ItemViewportQueriesMeasuredItemAboveViewport` | todo |
| `test_item_viewport_queries_measured_item_below_viewport` | gpui::test | 2019 |  | `TestList_ItemViewportQueriesMeasuredItemBelowViewport` | todo |
| `test_item_viewport_queries_remain_stable_with_zero_height_viewport` | gpui::test | 2037 |  | `TestList_ItemViewportQueriesRemainStableWithZeroHeightViewport` | todo |
| `test_item_viewport_queries_after_scroll_to_end_before_layout` | gpui::test | 2068 |  | `TestList_ItemViewportQueriesAfterScrollToEndBeforeLayout` | todo |
| `test_measure_all_after_width_change` | gpui::test | 2085 |  | `TestList_MeasureAllAfterWidthChange` | todo |
| `test_remeasure` | gpui::test | 2120 |  | `TestList_Remeasure` | todo |
| `test_remeasure_item_preserves_scroll_offset` | gpui::test | 2186 |  | `TestList_RemeasureItemPreservesScrollOffset` | todo |
| `test_provisional_item_heights_keep_scroll_anchor` | gpui::test | 2240 |  | `TestList_ProvisionalItemHeightsKeepScrollAnchor` | todo |
| `test_unready_measurements_preserve_pending_fraction` | gpui::test | 2286 |  | `TestList_UnreadyMeasurementsPreservePendingFraction` | todo |
| `test_navigation_cancels_provisional_and_paints_destination` | gpui::test | 2337 |  | `TestList_NavigationCancelsProvisionalAndPaintsDestination` | todo |
| `test_unready_overdraw_and_backfill_preserve_scroll_anchor` | gpui::test | 2504 |  | `TestList_UnreadyOverdrawAndBackfillPreserveScrollAnchor` | todo |
| `test_unready_overdraw_stops_at_first_pending_item` | gpui::test | 2564 |  | `TestList_UnreadyOverdrawStopsAtFirstPendingItem` | todo |
| `test_scrollbar_and_reset_do_not_resurrect_provisional` | gpui::test | 2624 |  | `TestList_ScrollbarAndResetDoNotResurrectProvisional` | todo |
| `test_splice_preserves_only_surviving_scroll_offsets` | gpui::test | 2679 |  | `TestList_SplicePreservesOnlySurvivingScrollOffsets` | todo |
| `test_child_autoscroll_overrides_pending_and_release_fills_viewport` | gpui::test | 2742 |  | `TestList_ChildAutoscrollOverridesPendingAndReleaseFillsViewport` | todo |
| `test_remeasure_then_scroll_does_not_revert_scroll_position` | gpui::test | 2828 |  | `TestList_RemeasureThenScrollDoesNotRevertScrollPosition` | todo |
| `test_scroll_after_remeasure_clamps_to_shrunk_item_height` | gpui::test | 2884 |  | `TestList_ScrollAfterRemeasureClampsToShrunkItemHeight` | todo |
| `test_follow_tail_stays_at_bottom_as_items_grow` | gpui::test | 2932 |  | `TestList_FollowTailStaysAtBottomAsItemsGrow` | todo |
| `test_pause_following_tail_reengages_when_still_at_bottom` | gpui::test | 3001 |  | `TestList_PauseFollowingTailReengagesWhenStillAtBottom` | todo |
| `test_pause_following_tail_freezes_off_bottom` | gpui::test | 3042 |  | `TestList_PauseFollowingTailFreezesOffBottom` | todo |
| `test_follow_tail_disengages_on_user_scroll` | gpui::test | 3111 |  | `TestList_FollowTailDisengagesOnUserScroll` | todo |
| `test_follow_tail_disengages_on_scrollbar_reposition` | gpui::test | 3151 |  | `TestList_FollowTailDisengagesOnScrollbarReposition` | todo |
| `test_scrollbar_drag_with_growing_content` | gpui::test | 3201 |  | `TestList_ScrollbarDragWithGrowingContent` | todo |
| `test_set_follow_tail_snaps_to_bottom` | gpui::test | 3266 |  | `TestList_SetFollowTailSnapsToBottom` | todo |
| `test_bottom_aligned_scrollbar_offset_at_end` | gpui::test | 3317 |  | `TestList_BottomAlignedScrollbarOffsetAtEnd` | todo |
| `test_follow_tail_reengages_when_scrolled_back_to_bottom` | gpui::test | 3362 |  | `TestList_FollowTailReengagesWhenScrolledBackToBottom` | todo |
| `test_follow_tail_reengagement_not_fooled_by_unmeasured_items` | gpui::test | 3417 |  | `TestList_FollowTailReengagementNotFooledByUnmeasuredItems` | todo |
| `test_follow_tail_reengages_after_scrollbar_disengagement` | gpui::test | 3476 |  | `TestList_FollowTailReengagesAfterScrollbarDisengagement` | todo |
| `test_follow_tail_reengages_after_scrollbar_drag_to_bottom_while_growing` | gpui::test | 3518 |  | `TestList_FollowTailReengagesAfterScrollbarDragToBottomWhileGrowing` | todo |
| `test_horizontal_padding_insets_items` | gpui::test | 3567 |  | `TestList_HorizontalPaddingInsetsItems` | todo |
| `test_inferred_width_includes_horizontal_padding` | gpui::test | 3614 |  | `TestList_InferredWidthIncludesHorizontalPadding` | todo |
| `test_autoscroll_above_item_top_with_horizontal_padding` | gpui::test | 3668 |  | `TestList_AutoscrollAboveItemTopWithHorizontalPadding` | todo |
| `test_horizontal_padding_change_remeasures_items` | gpui::test | 3716 |  | `TestList_HorizontalPaddingChangeRemeasuresItems` | todo |
| `test_border_and_padding_inset_items` | gpui::test | 3752 |  | `TestList_BorderAndPaddingInsetItems` | todo |

## gpui/src/elements/surface.rs

Go target: `elements_surface.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SurfaceSource` | enum | 13 |  | `SurfaceSource` | todo |
| `Surface` | struct | 27 |  | `Surface` | todo |
| `surface` | fn | 35 | macos,ios | `NewSurface` | todo |
| `Surface::object_fit` | method | 45 |  | `(Surface).ObjectFit` | todo |

## gpui/src/elements/svg.rs

Go target: `elements_svg.go` (package `gpui`), tests in `elements_svg_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Svg` | struct | 19 |  | `Svg` | todo |
| `svg` | fn | 30 |  | `NewSvg` | todo |
| `Svg::path` | method | 43 |  | `(Svg).Path` | todo |
| `Svg::external_path` | method | 49 |  | `(Svg).ExternalPath` | todo |
| `Svg::data` | method | 56 |  | `(Svg).Data` | todo |
| `Svg::with_transformation` | method | 69 |  | `(Svg).WithTransformation` | todo |
| `Svg::with_timed_transformation` | method | 83 |  | `(Svg).WithTimedTransformation` | todo |
| `Transformation` | struct | 244 |  | `Transformation` | todo |
| `Transformation::scale` | assoc-fn | 264 |  | `TransformationScale` | todo |
| `Transformation::translate` | assoc-fn | 272 |  | `TransformationTranslate` | todo |
| `Transformation::rotate` | assoc-fn | 280 |  | `TransformationRotate` | todo |
| `Transformation::alpha` | assoc-fn | 289 |  | `TransformationAlpha` | todo |
| `Transformation::with_alpha` | method | 298 |  | `(Transformation).WithAlpha` | todo |
| `Transformation::with_scaling` | method | 308 |  | `(Transformation).WithScaling` | todo |
| `Transformation::with_translation` | method | 314 |  | `(Transformation).WithTranslation` | todo |
| `Transformation::with_rotation` | method | 320 |  | `(Transformation).WithRotation` | todo |
| `Easing` | enum | 341 |  | `Easing` | todo |
| `TimedTransformation` | struct | 375 |  | `TimedTransformation` | todo |
| `TimedTransformation::new` | assoc-fn | 390 |  | `NewTimedTransformation` | todo |
| `TimedTransformation::rotation` | assoc-fn | 406 |  | `TimedTransformationRotation` | todo |
| `TimedTransformation::from` | method | 417 |  | `(TimedTransformation).From` | todo |
| `TimedTransformation::to` | method | 423 |  | `(TimedTransformation).To` | todo |
| `TimedTransformation::with_easing` | method | 429 |  | `(TimedTransformation).WithEasing` | todo |
| `TimedTransformation::with_scale_easing` | method | 438 |  | `(TimedTransformation).WithScaleEasing` | todo |
| `TimedTransformation::with_translation_easing` | method | 444 |  | `(TimedTransformation).WithTranslationEasing` | todo |
| `TimedTransformation::with_rotation_easing` | method | 450 |  | `(TimedTransformation).WithRotationEasing` | todo |
| `TimedTransformation::with_alpha_easing` | method | 456 |  | `(TimedTransformation).WithAlphaEasing` | todo |
| `TimedTransformation::alternate` | method | 463 |  | `(TimedTransformation).Alternate` | todo |
| `TimedTransformation::with_max_fps` | method | 471 |  | `(TimedTransformation).WithMaxFps` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_rotation_advances_without_rerendering` | crate::test | 635 |  | `TestSvg_RotationAdvancesWithoutRerendering` | todo |
| `test_timed_transformation_eases_each_value_separately` | crate::test | 664 |  | `TestSvg_TimedTransformationEasesEachValueSeparately` | todo |
| `test_heartbeat_returns_to_its_start` | crate::test | 687 |  | `TestSvg_HeartbeatReturnsToItsStart` | todo |
| `test_scene_presents_as_often_as_its_least_capped_animation` | crate::test | 704 |  | `TestSvg_ScenePresentsAsOftenAsItsLeastCappedAnimation` | todo |

## gpui/src/elements/text.rs

Go target: `elements_text.go` (package `gpui`), tests in `elements_text_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Text` | struct | 67 |  | `Text` | todo |
| `Text::new` | assoc-fn | 78 |  | `NewText` | todo |
| `Text::new_inaccessible` | assoc-fn | 92 |  | `NewTextInaccessible` | todo |
| `Text::id` | method | 98 |  | `(Text).ID` | todo |
| `Text::with_id` | method | 103 |  | `(Text).WithID` | todo |
| `Text::text` | method | 110 |  | `(Text).Text` | todo |
| `__hash_text_macro_location_unstable_do_not_use` | fn | 132 |  | `HashTextMacroLocationUnstableDoNotUse` | todo |
| `text` | macro | 160 |  | `see DESIGN §9 (text!)` | todo |
| `StyledText` | struct | 367 |  | `StyledText` | todo |
| `StyledText::new` | assoc-fn | 377 |  | `NewStyledText` | todo |
| `StyledText::layout` | method | 388 |  | `(StyledText).Layout` | todo |
| `StyledText::with_default_highlights` | method | 394 |  | `(StyledText).WithDefaultHighlights` | todo |
| `StyledText::with_highlights` | method | 409 |  | `(StyledText).WithHighlights` | todo |
| `StyledText::with_font_family_overrides` | method | 464 |  | `(StyledText).WithFontFamilyOverrides` | todo |
| `StyledText::with_runs` | method | 502 |  | `(StyledText).WithRuns` | todo |
| `TextLayout` | struct | 582 |  | `TextLayout` | todo |
| `TextLayout::paint_background` | method | 779 |  | `(TextLayout).PaintBackground` | todo |
| `TextLayout::paint_foreground` | method | 784 |  | `(TextLayout).PaintForeground` | todo |
| `TextLayout::index_for_position` | method | 834 |  | `(TextLayout).IndexForPosition` | todo |
| `TextLayout::position_for_index` | method | 868 |  | `(TextLayout).PositionForIndex` | todo |
| `TextLayout::line_layout_for_index` | method | 899 |  | `(TextLayout).LineLayoutForIndex` | todo |
| `TextLayout::line_layouts` | method | 922 |  | `(TextLayout).LineLayouts` | todo |
| `TextLayout::bounds` | method | 934 |  | `(TextLayout).Bounds` | todo |
| `TextLayout::line_height` | method | 939 |  | `(TextLayout).LineHeight` | todo |
| `TextLayout::len` | method | 944 |  | `(TextLayout).Len` | todo |
| `TextLayout::text` | method | 949 |  | `(TextLayout).Text` | todo |
| `TextLayout::wrapped_text` | method | 961 |  | `(TextLayout).WrappedText` | todo |
| `InteractiveText` | struct | 985 |  | `InteractiveText` | todo |
| `InteractiveTextState` | struct | 1003 |  | `InteractiveTextState` | todo |
| `InteractiveText::new` | assoc-fn | 1013 |  | `NewInteractiveText` | todo |
| `InteractiveText::on_click` | method | 1027 |  | `(InteractiveText).OnClick` | todo |
| `InteractiveText::on_hover` | method | 1046 |  | `(InteractiveText).OnHover` | todo |
| `InteractiveText::tooltip` | method | 1055 |  | `(InteractiveText).Tooltip` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_into_element_for` | test | 1284 |  | `TestText_IntoElementFor` | todo |
| `text_macro_id` | test | 1295 |  | `TestText_TextMacroID` | todo |

## gpui/src/elements/uniform_list.rs

Go target: `elements_uniform_list.go` (package `gpui`), tests in `elements_uniform_list_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `uniform_list` | fn | 22 |  | `NewUniformList` | todo |
| `UniformList` | struct | 58 |  | `UniformList` | todo |
| `UniformListFrameState` | struct | 72 |  | `UniformListFrameState` | todo |
| `UniformListScrollHandle` | struct | 80 |  | `UniformListScrollHandle` | todo |
| `ScrollStrategy` | enum | 84 |  | `ScrollStrategy` | todo |
| `DeferredScrollToItem` | struct | 103 |  | `DeferredScrollToItem` | todo |
| `UniformListScrollState` | struct | 115 |  | `UniformListScrollState` | todo |
| `ItemSize` | struct | 126 |  | `ItemSize` | todo |
| `UniformListScrollHandle::new` | assoc-fn | 136 |  | `NewUniformListScrollHandle` | todo |
| `UniformListScrollHandle::scroll_to_item` | method | 150 |  | `(UniformListScrollHandle).ScrollToItem` | todo |
| `UniformListScrollHandle::scroll_to_item_strict` | method | 163 |  | `(UniformListScrollHandle).ScrollToItemStrict` | todo |
| `UniformListScrollHandle::scroll_to_item_with_offset` | method | 182 |  | `(UniformListScrollHandle).ScrollToItemWithOffset` | todo |
| `UniformListScrollHandle::scroll_to_item_strict_with_offset` | method | 201 |  | `(UniformListScrollHandle).ScrollToItemStrictWithOffset` | todo |
| `UniformListScrollHandle::y_flipped` | method | 216 |  | `(UniformListScrollHandle).YFlipped` | todo |
| `UniformListScrollHandle::logical_scroll_top_index` | method | 222 | feat:test-support | `(UniformListScrollHandle).LogicalScrollTopIndex` | todo |
| `UniformListScrollHandle::is_scrollable` | method | 231 |  | `(UniformListScrollHandle).IsScrollable` | todo |
| `UniformListScrollHandle::is_scrolled_to_end` | method | 241 |  | `(UniformListScrollHandle).IsScrolledToEnd` | todo |
| `UniformListScrollHandle::scroll_to_bottom` | method | 252 |  | `(UniformListScrollHandle).ScrollToBottom` | todo |
| `UniformListDecoration` | trait | 575 |  | `UniformListDecoration` | todo |
| `UniformListDecoration::compute` | trait-method | 578 |  | `UniformListDecoration.Compute` | todo |
| `UniformList::with_width_from_item` | method | 617 |  | `(UniformList).WithWidthFromItem` | todo |
| `UniformList::with_sizing_behavior` | method | 623 |  | `(UniformList).WithSizingBehavior` | todo |
| `UniformList::with_horizontal_sizing_behavior` | method | 631 |  | `(UniformList).WithHorizontalSizingBehavior` | todo |
| `UniformList::with_decoration` | method | 648 |  | `(UniformList).WithDecoration` | todo |
| `UniformList::track_scroll` | method | 678 |  | `(UniformList).TrackScroll` | todo |
| `UniformList::y_flipped` | method | 685 |  | `(UniformList).YFlipped` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_scroll_strategy_nearest` | gpui::test | 720 |  | `TestUniformList_ScrollStrategyNearest` | todo |
| `test_item_positions_far_down_long_list` | gpui::test | 862 |  | `TestUniformList_ItemPositionsFarDownLongList` | todo |

