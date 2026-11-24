# Parity: core-input

Scope: gpui: keymap, key dispatch, interactive events, input handler, gestures, tab stops. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/gestures.rs

Go target: `gestures.go` (package `gpui`), tests in `gestures_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `OngoingScroll` | struct | 45 |  | `OngoingScroll` | done |
| `OngoingScroll::filter` | method | 55 |  | `(OngoingScroll).Filter` | done |
| `GestureTuning` | struct | 110 |  | `GestureTuning` | done |
| `ScrollPhysics` | enum | 147 |  | `ScrollPhysics` | done |
| `ScrollPhysics::ios` | assoc-fn | 173 |  | `ScrollPhysicsIos` | done |
| `ScrollPhysics::android` | assoc-fn | 185 |  | `ScrollPhysicsAndroid` | done |
| `GestureKinds` | struct | 349 |  | `GestureKinds` | done |
| `GestureKinds::NONE` | assoc-const | 363 |  | `GestureKindsNone` | done |
| `GestureKinds::ALL` | assoc-const | 371 |  | `GestureKindsAll` | done |
| `TouchDragEvent` | struct | 382 |  | `TouchDragEvent` | done |
| `BackGestureEvent` | struct | 410 |  | `BackGestureEvent` | done |
| `BackGesturePhase` | enum | 421 |  | `BackGesturePhase` | done |
| `LongPressEvent` | struct | 445 |  | `LongPressEvent` | done |
| `PlatformGestures` | trait | 477 |  | `PlatformGestures` | done |
| `PlatformGestures::tuning` | trait-method | 479 |  | `PlatformGestures.Tuning` | done |
| `PlatformGestures::native_recognizers` | trait-method | 484 |  | `PlatformGestures.NativeRecognizers` | done |
| `NullPlatformGestures` | struct | 491 |  | `NullPlatformGestures` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `ongoing_scroll_locks_to_dominant_axis` | test | 1191 |  | `TestGestures_OngoingScrollLocksToDominantAxis` | done |
| `ongoing_scroll_unlocks_when_direction_changes` | test | 1210 |  | `TestGestures_OngoingScrollUnlocksWhenDirectionChanges` | done |
| `ongoing_scroll_starts_new_gesture_at_timeout_boundary` | test | 1227 |  | `TestGestures_OngoingScrollStartsNewGestureAtTimeoutBoundary` | done |
| `ongoing_scroll_ignores_zero_delta_and_resets_when_ended` | test | 1244 |  | `TestGestures_OngoingScrollIgnoresZeroDeltaAndResetsWhenEnded` | done |
| `ongoing_scroll_ignores_zero_delta_movement` | test | 1269 |  | `TestGestures_OngoingScrollIgnoresZeroDeltaMovement` | done |
| `ongoing_scroll_supports_moved_only_platforms` | test | 1293 |  | `TestGestures_OngoingScrollSupportsMovedOnlyPlatforms` | done |
| `touch_within_slop_resolves_to_tap` | test | 1303 |  | `TestGestures_TouchWithinSlopResolvesToTap` | done |
| `consecutive_taps_accumulate_tap_count` | test | 1330 |  | `TestGestures_ConsecutiveTapsAccumulateTapCount` | done |
| `touch_beyond_slop_resolves_to_pan` | test | 1370 |  | `TestGestures_TouchBeyondSlopResolvesToPan` | done |
| `touch_pan_stays_locked_to_its_initial_dominant_axis` | test | 1410 |  | `TestGestures_TouchPanStaysLockedToItsInitialDominantAxis` | done |
| `touch_pan_locks_to_horizontal_axis` | test | 1437 |  | `TestGestures_TouchPanLocksToHorizontalAxis` | done |
| `predicted_positions_lead_the_pan_but_totals_converge_on_release` | test | 1455 |  | `TestGestures_PredictedPositionsLeadThePanButTotalsConvergeOnRelease` | done |
| `predicted_positions_do_not_emit_false_reversals` | test | 1497 |  | `TestGestures_PredictedPositionsDoNotEmitFalseReversals` | done |
| `predicted_overshoot_folds_into_the_fling_without_scrolling_backwards` | test | 1552 |  | `TestGestures_PredictedOvershootFoldsIntoTheFlingWithoutScrollingBackwards` | done |
| `fast_release_starts_momentum_that_decays_to_a_stop` | test | 1620 |  | `TestGestures_FastReleaseStartsMomentumThatDecaysToAStop` | done |
| `diagonal_release_flings_only_on_locked_axis` | test | 1673 |  | `TestGestures_DiagonalReleaseFlingsOnlyOnLockedAxis` | done |
| `slow_release_does_not_start_momentum` | test | 1713 |  | `TestGestures_SlowReleaseDoesNotStartMomentum` | done |
| `new_touch_interrupts_momentum` | test | 1735 |  | `TestGestures_NewTouchInterruptsMomentum` | done |
| `catching_a_fling_pans_immediately_and_never_taps` | test | 1779 |  | `TestGestures_CatchingAFlingPansImmediatelyAndNeverTaps` | done |
| `catching_a_fling_takes_the_axis_from_the_new_touch` | test | 1832 |  | `TestGestures_CatchingAFlingTakesTheAxisFromTheNewTouch` | done |
| `cancelled_pan_emits_cancelled_scroll_and_no_tap` | test | 1886 |  | `TestGestures_CancelledPanEmitsCancelledScrollAndNoTap` | done |
| `concurrent_touches_are_ignored_while_one_is_active` | test | 1916 |  | `TestGestures_ConcurrentTouchesAreIgnoredWhileOneIsActive` | done |
| `spline_position_table_matches_the_bezier_curve` | test | 1952 |  | `TestGestures_SplinePositionTableMatchesTheBezierCurve` | done |
| `fling_curves_are_sane_for_both_physics` | test | 1989 |  | `TestGestures_FlingCurvesAreSaneForBothPhysics` | done |
| `momentum_is_frame_rate_independent` | test | 2011 |  | `TestGestures_MomentumIsFrameRateIndependent` | done |
| `flick_velocity_reflects_release_speed_not_window_average` | test | 2066 |  | `TestGestures_FlickVelocityReflectsReleaseSpeedNotWindowAverage` | done |
| `samples_before_a_pause_do_not_contribute_velocity` | test | 2090 |  | `TestGestures_SamplesBeforeAPauseDoNotContributeVelocity` | done |
| `claimed_touch_drag_emits_phased_stream_without_pan_or_tap` | test | 2109 |  | `TestGestures_ClaimedTouchDragEmitsPhasedStreamWithoutPanOrTap` | done |
| `unclaimed_touch_drag_remains_a_pan_candidate` | test | 2144 |  | `TestGestures_UnclaimedTouchDragRemainsAPanCandidate` | done |
| `claimed_long_press_emits_phased_stream_without_tap` | test | 2166 |  | `TestGestures_ClaimedLongPressEmitsPhasedStreamWithoutTap` | done |
| `unclaimed_long_press_remains_a_tap_candidate` | test | 2199 |  | `TestGestures_UnclaimedLongPressRemainsATapCandidate` | done |
| `unclaimed_long_press_can_still_become_a_pan` | test | 2218 |  | `TestGestures_UnclaimedLongPressCanStillBecomeAPan` | done |
| `long_press_offer_is_one_shot_and_specific_to_pending_touch` | test | 2240 |  | `TestGestures_LongPressOfferIsOneShotAndSpecificToPendingTouch` | done |
| `long_press_cannot_be_offered_after_pending_touch_resolves` | test | 2256 |  | `TestGestures_LongPressCannotBeOfferedAfterPendingTouchResolves` | done |
| `claimed_long_press_emits_cancelled_for_its_touch_only` | test | 2272 |  | `TestGestures_ClaimedLongPressEmitsCancelledForItsTouchOnly` | done |
| `unrelated_touch_cannot_end_or_cancel_pending_touch` | test | 2294 |  | `TestGestures_UnrelatedTouchCannotEndOrCancelPendingTouch` | done |
| `completed_touch_id_cannot_claim_replacement_touch` | test | 2310 |  | `TestGestures_CompletedTouchIDCannotClaimReplacementTouch` | done |

## gpui/src/input.rs

Go target: `input.go` (package `gpui`), tests in `input_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `EntityInputHandler` | trait | 26 |  | `EntityInputHandler` | done |
| `EntityInputHandler::text_for_range` | trait-method | 28 |  | `EntityInputHandler.TextForRange` | done |
| `EntityInputHandler::selected_text_range` | trait-method | 37 |  | `EntityInputHandler.SelectedTextRange` | done |
| `EntityInputHandler::marked_text_range` | trait-method | 45 |  | `EntityInputHandler.MarkedTextRange` | done |
| `EntityInputHandler::unmark_text` | trait-method | 52 |  | `EntityInputHandler.UnmarkText` | done |
| `EntityInputHandler::paste` | trait-method | 55 |  | `EntityInputHandler.Paste` | done |
| `EntityInputHandler::replace_text_in_range` | trait-method | 62 |  | `EntityInputHandler.ReplaceTextInRange` | done |
| `EntityInputHandler::replace_and_mark_text_in_range` | trait-method | 71 |  | `EntityInputHandler.ReplaceAndMarkTextInRange` | done |
| `EntityInputHandler::bounds_for_range` | trait-method | 81 |  | `EntityInputHandler.BoundsForRange` | done |
| `EntityInputHandler::selection_bounds_for_range` | trait-method | 90 |  | `EntityInputHandler.SelectionBoundsForRange` | done |
| `EntityInputHandler::character_index_for_point` | trait-method | 101 |  | `EntityInputHandler.CharacterIndexForPoint` | done |
| `EntityInputHandler::set_selected_text_range` | trait-method | 109 |  | `EntityInputHandler.SetSelectedTextRange` | done |
| `EntityInputHandler::text_length_utf16` | trait-method | 118 |  | `EntityInputHandler.TextLengthUTF16` | done |
| `EntityInputHandler::accepts_text_input` | trait-method | 127 |  | `EntityInputHandler.AcceptsTextInput` | done |
| `EntityInputHandler::text_input_configuration` | trait-method | 132 |  | `EntityInputHandler.TextInputConfiguration` | done |
| `EntityInputHandler::text_input_editable_range` | trait-method | 141 |  | `EntityInputHandler.TextInputEditableRange` | done |
| `ElementInputHandler` | struct | 152 |  | `ElementInputHandler` | done |
| `ElementInputHandler::new` | assoc-fn | 161 |  | `NewElementInputHandler` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `text_input_configuration_and_focus_state_are_forwarded_on_change` | gpui::test | 331 |  | `TestInput_TextInputConfigurationAndFocusStateAreForwardedOnChange` | done |
| `text_notifications_require_a_focused_input_and_track_field_changes` | gpui::test | 421 |  | `TestInput_TextNotificationsRequireAFocusedInputAndTrackFieldChanges` | done |

## gpui/src/interactive.rs

Go target: `interactive.go` (package `gpui`), tests in `interactive_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `InputEvent` | trait | 9 |  | `InputEvent` | done |
| `InputEvent::to_platform_input` | trait-method | 11 |  | `InputEvent.ToPlatformInput` | done |
| `KeyEvent` | trait | 15 |  | `KeyEvent` | done |
| `MouseEvent` | trait | 18 |  | `MouseEvent` | done |
| `GestureEvent` | trait | 21 |  | `GestureEvent` | done |
| `KeyDownEvent` | struct | 25 |  | `KeyDownEvent` | done |
| `KeyUpEvent` | struct | 47 |  | `KeyUpEvent` | done |
| `ModifiersChangedEvent` | struct | 62 |  | `ModifiersChangedEvent` | done |
| `TouchPhase` | enum | 88 |  | `TouchPhase` (zero value Moved) | dev:D-FLATENUM |
| `TouchId` | struct | 109 |  | `TouchID` | done |
| `TouchEvent` | struct | 119 |  | `TouchEvent` | done |
| `MouseDownEvent` | struct | 148 |  | `MouseDownEvent` | done |
| `MouseDownEvent::is_focusing` | method | 175 |  | `(MouseDownEvent).IsFocusing` | done |
| `MouseUpEvent` | struct | 185 |  | `MouseUpEvent` | done |
| `MouseUpEvent::is_focusing` | method | 210 |  | `(MouseUpEvent).IsFocusing` | done |
| `MouseClickEvent` | struct | 220 |  | `MouseClickEvent` | done |
| `PressureStage` | enum | 230 |  | `PressureStage` | done |
| `MousePressureEvent` | struct | 243 |  | `MousePressureEvent` | done |
| `KeyboardClickEvent` | struct | 264 |  | `KeyboardClickEvent` | done |
| `TouchClickEvent` | struct | 275 |  | `TouchClickEvent` | done |
| `ClickEvent` | enum | 290 |  | `ClickEvent` | done |
| `ClickEvent::modifiers` | method | 310 |  | `(ClickEvent).Modifiers` | done |
| `ClickEvent::position` | method | 328 |  | `(ClickEvent).Position` | done |
| `ClickEvent::mouse_position` | method | 341 |  | `(ClickEvent).MousePosition` | done |
| `ClickEvent::is_right_click` | method | 353 |  | `(ClickEvent).IsRightClick` | done |
| `ClickEvent::is_middle_click` | method | 367 |  | `(ClickEvent).IsMiddleClick` | done |
| `ClickEvent::is_secondary` | method | 381 |  | `(ClickEvent).IsSecondary` | done |
| `ClickEvent::standard_click` | method | 396 |  | `(ClickEvent).StandardClick` | done |
| `ClickEvent::first_focus` | method | 411 |  | `(ClickEvent).FirstFocus` | done |
| `ClickEvent::click_count` | method | 424 |  | `(ClickEvent).ClickCount` | done |
| `ClickEvent::is_keyboard` | method | 433 |  | `(ClickEvent).IsKeyboard` | done |
| `KeyboardButton` | enum | 443 |  | `KeyboardButton` | done |
| `MouseButton` | enum | 453 |  | `MouseButton` (Navigate flattened) | dev:D-FLATENUM |
| `MouseButton::all` | assoc-fn | 470 |  | `MouseButtonAll` | done |
| `NavigationDirection` | enum | 483 |  | `NavigationDirection` | done |
| `MouseMoveEvent` | struct | 494 |  | `MouseMoveEvent` | done |
| `MouseMoveEvent::dragging` | method | 515 |  | `(MouseMoveEvent).Dragging` | done |
| `ScrollEvent` | struct | 522 |  | `ScrollEvent` | done |
| `ScrollWheelEvent` | type | 542 |  | `ScrollWheelEvent` | done |
| `ScrollSource` | enum | 546 |  | `ScrollSource` | done |
| `ScrollDelta` | enum | 575 |  | `ScrollDelta` | done |
| `PinchEvent` | struct | 592 |  | `PinchEvent` | done |
| `ScrollDelta::precise` | method | 627 |  | `(ScrollDelta).Precise` | done |
| `ScrollDelta::pixel_delta` | method | 635 |  | `(ScrollDelta).PixelDelta` | done |
| `ScrollDelta::coalesce` | method | 646 |  | `(ScrollDelta).Coalesce` | done |
| `MouseExitEvent` | struct | 687 |  | `MouseExitEvent` | done |
| `ExternalPaths` | struct | 715 |  | `ExternalPaths` | done |
| `ExternalPaths::paths` | method | 719 |  | `(ExternalPaths).Paths` | done |
| `ExternalDragPayload` | enum | 727 |  | `ExternalDragPayload` | done |
| `FileDragPaths` | struct | 735 |  | `FileDragPaths` | done |
| `FileDragPaths::new` | assoc-fn | 739 |  | `NewFileDragPaths` | done |
| `FileDragPaths::entries` | method | 744 |  | `(FileDragPaths).Entries` | done |
| `FileDropEvent` | enum | 758 |  | `FileDropEvent` | done |
| `PlatformInput` | enum | 792 |  | `PlatformInput` | done |
| `PlatformInput::kind_name` | method | 869 |  | `(PlatformInput).KindName` | done |
| `PlatformInput::touch_event` | method | 890 |  | `(PlatformInput).TouchEvent` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_on_events` | gpui::test | 937 |  | `TestInteractive_OnEvents` | done |
| `test_multi_modifier_gesture_does_not_dispatch_standalone_modifier_binding` | gpui::test | 972 |  | `TestInteractive_MultiModifierGestureDoesNotDispatchStandaloneModifierBinding` | done |

## gpui/src/key_dispatch.rs

Go target: `key_dispatch.go` (package `gpui`), tests in `key_dispatch_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_keybinding_for_action_bounds` | test | 961 |  | `TestKeyDispatch_KeybindingForActionBounds` | done |
| `test_bindings_for_action_hides_targeted_unbind_in_active_context` | test | 979 |  | `TestKeyDispatch_BindingsForActionHidesTargetedUnbindInActiveContext` | done |
| `test_bindings_for_action_keeps_targeted_binding_outside_unbind_context` | test | 1011 |  | `TestKeyDispatch_BindingsForActionKeepsTargetedBindingOutsideUnbindContext` | done |
| `test_overlay_after_base_restores_shadowed_picker_binding` | test | 1048 |  | `TestKeyDispatch_OverlayAfterBaseRestoresShadowedPickerBinding` | done |
| `test_pending_has_binding_state` | test | 1100 |  | `TestKeyDispatch_PendingHasBindingState` | done |
| `test_pending_input_observers_notified_on_focus_change_and_blur` | crate::test | 1140 |  | `TestKeyDispatch_PendingInputObserversNotifiedOnFocusChangeAndBlur` | done |
| `test_standalone_modifier_dispatch` | crate::test | 1385 |  | `TestKeyDispatch_StandaloneModifierDispatch` | done |
| `test_unbound_standalone_modifier_observation` | crate::test | 1445 |  | `TestKeyDispatch_UnboundStandaloneModifierObservation` | done |
| `test_raw_modifier_handler_can_suppress_observation` | crate::test | 1476 |  | `TestKeyDispatch_RawModifierHandlerCanSuppressObservation` | done |
| `test_pending_modifier_observed_when_binding_resolves` | crate::test | 1539 |  | `TestKeyDispatch_PendingModifierObservedWhenBindingResolves` | done |
| `test_combined_shortcut_has_no_standalone_modifier` | crate::test | 1582 |  | `TestKeyDispatch_CombinedShortcutHasNoStandaloneModifier` | done |
| `test_keystroke_input_preference` | crate::test | 1626 |  | `TestKeyDispatch_KeystrokeInputPreference` | done |
| `test_input_preference_rechecked_after_interception` | crate::test | 1686 |  | `TestKeyDispatch_InputPreferenceRecheckedAfterInterception` | done |
| `test_keystroke_interceptors_can_consume_preferred_character_input` | crate::test | 1736 |  | `TestKeyDispatch_KeystrokeInterceptorsCanConsumePreferredCharacterInput` | done |
| `test_consumed_modifier_preserves_pending_input` | crate::test | 1757 |  | `TestKeyDispatch_ConsumedModifierPreservesPendingInput` | done |
| `test_printable_pending_input_replays_on_timeout` | crate::test | 1817 |  | `TestKeyDispatch_PrintablePendingInputReplaysOnTimeout` | done |
| `test_pending_input_timeout_dispatches_shorter_binding` | crate::test | 1851 |  | `TestKeyDispatch_PendingInputTimeoutDispatchesShorterBinding` | done |
| `test_running_pending_input_timeout_resets_when_binding_advances` | crate::test | 1885 |  | `TestKeyDispatch_RunningPendingInputTimeoutResetsWhenBindingAdvances` | done |
| `test_pending_input_timeout_starts_when_binding_becomes_ambiguous` | crate::test | 1919 |  | `TestKeyDispatch_PendingInputTimeoutStartsWhenBindingBecomesAmbiguous` | done |
| `test_invalid_continuation_while_timeout_paused_replays_pending_input` | crate::test | 1952 |  | `TestKeyDispatch_InvalidContinuationWhileTimeoutPausedReplaysPendingInput` | done |
| `test_pending_input_timeout_pauses_and_resumes` | crate::test | 1989 |  | `TestKeyDispatch_PendingInputTimeoutPausesAndResumes` | done |
| `test_pending_input_timeout_resumes_when_owner_is_released` | crate::test | 2043 |  | `TestKeyDispatch_PendingInputTimeoutResumesWhenOwnerIsReleased` | done |
| `test_pending_input_timeout_resets_when_binding_advances` | crate::test | 2079 |  | `TestKeyDispatch_PendingInputTimeoutResetsWhenBindingAdvances` | done |
| `test_clearing_pending_input_invalidates_timeout_pause` | crate::test | 2128 |  | `TestKeyDispatch_ClearingPendingInputInvalidatesTimeoutPause` | done |
| `test_input_handler_pending` | crate::test | 2150 |  | `TestKeyDispatch_InputHandlerPending` | done |

## gpui/src/keymap.rs

Go target: `keymap.go` (package `gpui`), tests in `keymap_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `KeymapVersion` | struct | 14 |  | `KeymapVersion` | done |
| `Keymap` | struct | 18 |  | `Keymap` | done |
| `BindingIndex` | struct | 27 |  | `BindingIndex` | done |
| `Keymap::new` | assoc-fn | 51 |  | `NewKeymap` | done |
| `Keymap::version` | method | 58 |  | `(Keymap).Version` | done |
| `Keymap::add_bindings` | method | 63 |  | `(Keymap).AddBindings` | done |
| `Keymap::clear` | method | 81 |  | `(Keymap).Clear` | done |
| `Keymap::bindings` | method | 89 |  | `(Keymap).Bindings` | done |
| `Keymap::bindings_for_action` | method | 95 |  | `(Keymap).BindingsForAction` | done |
| `Keymap::all_bindings_for_input` | method | 138 |  | `(Keymap).AllBindingsForInput` | done |
| `Keymap::bindings_for_input` | method | 165 |  | `(Keymap).BindingsForInput` | done |
| `Keymap::possible_next_bindings_for_input` | method | 285 |  | `(Keymap).PossibleNextBindingsForInput` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_keymap` | test | 352 |  | `TestKeymap_Keymap` | done |
| `test_depth_precedence` | test | 393 |  | `TestKeymap_DepthPrecedence` | done |
| `test_keymap_disabled` | test | 417 |  | `TestKeymap_KeymapDisabled` | done |
| `test_multiple_keystroke_binding_disabled` | test | 473 |  | `TestKeymap_MultipleKeystrokeBindingDisabled` | done |
| `test_override_multikey` | test | 568 |  | `TestKeymap_OverrideMultikey` | done |
| `test_simple_disable` | test | 603 |  | `TestKeymap_SimpleDisable` | done |
| `test_disable_weaker_sources_only` | test | 622 |  | `TestKeymap_DisableWeakerSourcesOnly` | done |
| `test_fail_to_disable` | test | 691 |  | `TestKeymap_FailToDisable` | done |
| `test_disable_deeper` | test | 714 |  | `TestKeymap_DisableDeeper` | done |
| `test_pending_match_enabled` | test | 736 |  | `TestKeymap_PendingMatchEnabled` | done |
| `test_pending_match_enabled_extended` | test | 759 |  | `TestKeymap_PendingMatchEnabledExtended` | done |
| `test_overriding_prefix` | test | 801 |  | `TestKeymap_OverridingPrefix` | done |
| `test_context_precedence_with_same_source` | test | 824 |  | `TestKeymap_ContextPrecedenceWithSameSource` | done |
| `test_bindings_for_action` | test | 851 |  | `TestKeymap_BindingsForAction` | done |
| `test_targeted_unbind_ignores_target_context` | test | 882 |  | `TestKeymap_TargetedUnbindIgnoresTargetContext` | done |
| `test_bindings_for_action_keeps_binding_for_narrower_targeted_unbind` | test | 907 |  | `TestKeymap_BindingsForActionKeepsBindingForNarrowerTargetedUnbind` | done |
| `test_bindings_for_action_removes_binding_for_broader_targeted_unbind` | test | 939 |  | `TestKeymap_BindingsForActionRemovesBindingForBroaderTargetedUnbind` | done |
| `test_source_precedence_sorting` | test | 956 |  | `TestKeymap_SourcePrecedenceSorting` | done |
| `test_possible_next_bindings_exclude_null_bindings` | test | 984 |  | `TestKeymap_PossibleNextBindingsExcludeNullBindings` | done |
| `test_possible_next_bindings_exclude_unbound_bindings` | test | 1002 |  | `TestKeymap_PossibleNextBindingsExcludeUnboundBindings` | done |
| `test_possible_next_bindings_keep_user_binding_over_base_keymap_null` | test | 1024 |  | `TestKeymap_PossibleNextBindingsKeepUserBindingOverBaseKeymapNull` | done |
| `test_possible_next_bindings_apply_disabling_by_context` | test | 1047 |  | `TestKeymap_PossibleNextBindingsApplyDisablingByContext` | done |
| `test_possible_next_bindings_exclude_longer_chord_disabled_in_deeper_context` | test | 1079 |  | `TestKeymap_PossibleNextBindingsExcludeLongerChordDisabledInDeeperContext` | done |
| `test_possible_next_bindings_exclude_disabled_bindings_with_typed_input` | test | 1104 |  | `TestKeymap_PossibleNextBindingsExcludeDisabledBindingsWithTypedInput` | done |

## gpui/src/tab_stop.rs

Go target: `tab_stop.go` (package `gpui`), tests in `tab_stop_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TabStopOperation` | enum | 23 |  | `TabStopOperation` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_tab_handles` | test | 193 |  | `TestTabStop_TabHandles` | done |
| `test_tab_non_stop_filtering` | test | 282 |  | `TestTabStop_TabNonStopFiltering` | done |
| `test_insert_after_sorting` | test | 393 |  | `TestTabStop_InsertAfterSorting` | done |
| `test_with_disabled_tab_stop` | test | 409 |  | `TestTabStop_WithDisabledTabStop` | done |
| `test_with_multiple_disabled_tab_stops` | test | 419 |  | `TestTabStop_WithMultipleDisabledTabStops` | done |
| `test_tab_group_functionality` | test | 430 |  | `TestTabStop_TabGroupFunctionality` | done |
| `test_sibling_groups` | test | 441 |  | `TestTabStop_SiblingGroups` | done |
| `test_nested_group` | test | 455 |  | `TestTabStop_NestedGroup` | done |
| `test_sibling_nested_groups` | test | 469 |  | `TestTabStop_SiblingNestedGroups` | done |
| `test_sibling_nested_groups_out_of_order` | test | 486 |  | `TestTabStop_SiblingNestedGroupsOutOfOrder` | done |

## gpui/src/keymap/binding.rs

Go target: `keymap_binding.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `KeyBinding` | struct | 10 |  | `KeyBinding` | done |
| `KeyBinding::new` | assoc-fn | 35 |  | `NewKeyBinding` | done |
| `KeyBinding::load` | assoc-fn | 51 |  | `KeyBindingLoad` | done |
| `KeyBinding::with_meta` | method | 101 |  | `(KeyBinding).WithMeta` | done |
| `KeyBinding::set_meta` | method | 107 |  | `(KeyBinding).SetMeta` | done |
| `KeyBinding::match_keystrokes` | method | 112 |  | `(KeyBinding).MatchKeystrokes` | done |
| `KeyBinding::keystrokes` | method | 127 |  | `(KeyBinding).Keystrokes` | done |
| `KeyBinding::action` | method | 132 |  | `(KeyBinding).Action` | done |
| `KeyBinding::predicate` | method | 137 |  | `(KeyBinding).Predicate` | done |
| `KeyBinding::meta` | method | 142 |  | `(KeyBinding).Meta` | done |
| `KeyBinding::action_input` | method | 147 |  | `(KeyBinding).ActionInput` | done |
| `KeyBindingMetaIndex` | struct | 166 |  | `KeyBindingMetaIndex` | done |

## gpui/src/keymap/context.rs

Go target: `keymap_context.go` (package `gpui`), tests in `keymap_context_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `KeyContext` | struct | 10 |  | `KeyContext` | done |
| `ContextEntry` | struct | 14 |  | `ContextEntry` | done |
| `KeyContext::new_with_defaults` | assoc-fn | 31 |  | `NewKeyContextWithDefaults` | done |
| `KeyContext::primary` | method | 48 |  | `(KeyContext).Primary` | done |
| `KeyContext::secondary` | method | 53 |  | `(KeyContext).Secondary` | done |
| `KeyContext::parse` | assoc-fn | 63 |  | `KeyContextParse` | done |
| `KeyContext::is_empty` | method | 96 |  | `(KeyContext).IsEmpty` | done |
| `KeyContext::clear` | method | 101 |  | `(KeyContext).Clear` | done |
| `KeyContext::extend` | method | 106 |  | `(KeyContext).Extend` | done |
| `KeyContext::add` | method | 115 |  | `(KeyContext).Add` | done |
| `KeyContext::set` | method | 124 |  | `(KeyContext).Set` | done |
| `KeyContext::contains` | method | 135 |  | `(KeyContext).Contains` | done |
| `KeyContext::get` | method | 140 |  | `(KeyContext).Get` | done |
| `KeyBindingContextPredicate` | enum | 170 |  | `KeyBindingContextPredicate` | done |
| `KeyBindingContextPredicate::parse` | assoc-fn | 247 |  | `KeyBindingContextPredicateParse` | done |
| `KeyBindingContextPredicate::depth_of` | method | 258 |  | `(KeyBindingContextPredicate).DepthOf` | done |
| `KeyBindingContextPredicate::eval` | method | 269 |  | `(KeyBindingContextPredicate).Eval` | done |
| `KeyBindingContextPredicate::eval_inner` | method | 274 |  | `(KeyBindingContextPredicate).EvalInner` | done |
| `KeyBindingContextPredicate::is_superset` | method | 325 |  | `(KeyBindingContextPredicate).IsSuperset` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_actions_definition` | test | 521 |  | `TestContext_ActionsDefinition` | done |
| `test_parse_context` | test | 537 |  | `TestContext_ParseContext` | done |
| `test_parse_identifiers` | test | 551 |  | `TestContext_ParseIdentifiers` | done |
| `test_parse_negations` | test | 564 |  | `TestContext_ParseNegations` | done |
| `test_parse_equality_operators` | test | 576 |  | `TestContext_ParseEqualityOperators` | done |
| `test_parse_boolean_operators` | test | 594 |  | `TestContext_ParseBooleanOperators` | done |
| `test_parse_parenthesized_expressions` | test | 654 |  | `TestContext_ParseParenthesizedExpressions` | done |
| `test_is_superset` | test | 675 |  | `TestContext_IsSuperset` | done |
| `test_child_operator` | test | 698 |  | `TestContext_ChildOperator` | done |
| `test_not_operator` | test | 734 |  | `TestContext_NotOperator` | done |
| `test_context_display` | test | 814 |  | `TestContext_ContextDisplay` | done |

## gpui/src/platform/keyboard.rs

Go target: `platform_keyboard.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PlatformKeyboardLayout` | trait | 6 |  | `PlatformKeyboardLayout` | done |
| `PlatformKeyboardLayout::id` | trait-method | 8 |  | `PlatformKeyboardLayout.ID` | done |
| `PlatformKeyboardLayout::name` | trait-method | 10 |  | `PlatformKeyboardLayout.Name` | done |
| `PlatformKeyboardMapper` | trait | 14 |  | `PlatformKeyboardMapper` | done |
| `PlatformKeyboardMapper::map_key_equivalent` | trait-method | 16 |  | `PlatformKeyboardMapper.MapKeyEquivalent` | done |
| `PlatformKeyboardMapper::get_key_equivalents` | trait-method | 23 |  | `PlatformKeyboardMapper.GetKeyEquivalents` | done |
| `DummyKeyboardMapper` | struct | 27 |  | `DummyKeyboardMapper` | done |

## gpui/src/platform/keystroke.rs

Go target: `platform_keystroke.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AsKeystroke` | trait | 11 |  | `AsKeystroke` | done |
| `AsKeystroke::as_keystroke` | trait-method | 13 |  | `AsKeystroke.AsKeystroke` | done |
| `Keystroke` | struct | 18 |  | `Keystroke` | done |
| `KeybindingKeystroke` | struct | 37 |  | `KeybindingKeystroke` | done |
| `InvalidKeystrokeError` | struct | 51 |  | `InvalidKeystrokeError` | done |
| `KEYSTROKE_PARSE_EXPECTED_MESSAGE` | const | 69 |  | `KeystrokeParseExpectedMessage` | done |
| `Keystroke::should_match` | method | 82 |  | `(Keystroke).ShouldMatch` | done |
| `Keystroke::parse` | assoc-fn | 120 |  | `KeystrokeParse` | done |
| `Keystroke::parse_with_os` | assoc-fn | 125 |  | `KeystrokeParseWithOs` | done |
| `Keystroke::unparse` | method | 241 |  | `(Keystroke).Unparse` | done |
| `Keystroke::display` | method | 246 |  | `(Keystroke).Display` | done |
| `Keystroke::is_ime_in_progress` | method | 256 |  | `(Keystroke).IsIMEInProgress` | done |
| `Keystroke::with_simulated_ime` | method | 268 |  | `(Keystroke).WithSimulatedIME` | done |
| `Keystroke::is_modifier_key` | method | 293 |  | `(Keystroke).IsModifierKey` | done |
| `KeybindingKeystroke::display` | method | 303 |  | `(KeybindingKeystroke).Display` | done |
| `KeybindingKeystroke::new` | assoc-fn | 313 | windows | `NewKeybindingKeystroke` | done |
| `KeybindingKeystroke::new_with_mapper` | assoc-fn | 322 |  | `NewKeybindingKeystrokeWithMapper` | done |
| `KeybindingKeystroke::from_keystroke` | assoc-fn | 331 |  | `KeybindingKeystrokeFromKeystroke` | done |
| `KeybindingKeystroke::inner` | method | 349 |  | `(KeybindingKeystroke).Inner` | done |
| `KeybindingKeystroke::modifiers` | method | 359 |  | `(KeybindingKeystroke).Modifiers` | done |
| `KeybindingKeystroke::key` | method | 375 |  | `(KeybindingKeystroke).Key` | done |
| `KeybindingKeystroke::set_modifiers` | method | 387 |  | `(KeybindingKeystroke).SetModifiers` | done |
| `KeybindingKeystroke::set_key` | method | 396 |  | `(KeybindingKeystroke).SetKey` | done |
| `KeybindingKeystroke::unparse` | method | 405 |  | `(KeybindingKeystroke).Unparse` | done |
| `KeybindingKeystroke::remove_key_char` | method | 417 |  | `(KeybindingKeystroke).RemoveKeyChar` | done |
| `KeystrokeDisplay` | struct | 477 |  | `KeystrokeDisplay` | done |
| `Modifiers` | struct | 492 |  | `Modifiers` | done |
| `Modifiers::modified` | method | 519 |  | `(Modifiers).Modified` | done |
| `Modifiers::secondary` | method | 527 |  | `(Modifiers).Secondary` | done |
| `Modifiers::number_of_modifiers` | method | 536 |  | `(Modifiers).NumberOfModifiers` | done |
| `Modifiers::none` | assoc-fn | 545 |  | `ModifiersNone` | done |
| `Modifiers::command` | assoc-fn | 550 |  | `ModifiersCommand` | done |
| `Modifiers::secondary_key` | assoc-fn | 558 |  | `ModifiersSecondaryKey` | done |
| `Modifiers::windows` | assoc-fn | 573 |  | `ModifiersWindows` | done |
| `Modifiers::super_key` | assoc-fn | 581 |  | `ModifiersSuperKey` | done |
| `Modifiers::control` | assoc-fn | 589 |  | `ModifiersControl` | done |
| `Modifiers::alt` | assoc-fn | 597 |  | `ModifiersAlt` | done |
| `Modifiers::shift` | assoc-fn | 605 |  | `ModifiersShift` | done |
| `Modifiers::function` | assoc-fn | 613 |  | `ModifiersFunction` | done |
| `Modifiers::command_shift` | assoc-fn | 621 |  | `ModifiersCommandShift` | done |
| `Modifiers::control_shift` | assoc-fn | 630 |  | `ModifiersControlShift` | done |
| `Modifiers::is_subset_of` | method | 639 |  | `(Modifiers).IsSubsetOf` | done |
| `Capslock` | struct | 701 |  | `Capslock` | done |


## Notes (M0-input-types)

- keystroke.rs: `cfg(target_os = "windows")` parts are runtime branches on `runtime.GOOS == "windows"` (`targetWindows`): `KeybindingKeystroke` carries the display fields on every OS and `NewKeybindingKeystroke` exists everywhere (on other OSes the display fields are never read). Parsing uses ASCII-only case folding as upstream (`eq_ignore_ascii_case`, `make_ascii_lowercase`). `KeystrokeDisplay` formats with `String()`. Modifier `BitOr` / `BitXor` / `BitAnd` (+`Assign`) are methods. JSON tags follow serde's field names.
- interactive.rs: `PlatformInput` is a sealed interface whose variants are the event structs themselves (`KindName`, `TouchEvent` are interface methods; the crate-private `mouse_event` / `keyboard_event` are `platformInputMouseEvent` / `platformInputKeyboardEvent` returning a pointer to a copy, or nil). `InputEvent` / `KeyEvent` / `MouseEvent` / `GestureEvent` are sealed interfaces implemented by value. Events with `Deref<Target = Modifiers>` (`ModifiersChangedEvent`, `ScrollEvent`, `PinchEvent`, `MouseExitEvent`) embed `Modifiers`. `ClickEvent` is a tagged struct (`Kind`, zero value = Keyboard, upstream's default); `ScrollDelta` a tagged struct whose zero value is `Lines(0,0)`. `FileDropEvent` variants are `FileDropEventEntered`, `...Pending`, `...Submit`, `...Exited`, `...Ended`. `FileDragPaths` entries are `FileDragPath{Path, IsDir}` (D-TUPLE). The `LongPressEvent` / `TouchDragEvent` / `BackGestureEvent` PlatformInput methods live in interactive.go; M0-gestures-spring replaces their stub types and adds their `InputEvent` impls in gestures.go.
- input.rs: `EntityInputHandler[T]` is implemented by `*T`; provided methods come from `EntityInputHandlerDefaults[T]` (bind with `BindEntityInputHandlerDefaults`). `NewElementInputHandler` panics if `*V` does not implement it (D-COND) and returns `*ElementInputHandler[V]`, which embeds `InputHandlerDefaults` for `ApplePressAndHoldEnabled`.

## Notes (M0-keymap)

- `KeyBindingContextPredicate` is a sealed interface whose variants are value structs (`KeyBindingContextPredicateIdentifier{Name}`, `...Equal{Left, Right}`, `...NotEqual{Left, Right}`, `...Descendant{Parent, Child}`, `...Not{Predicate}`, `...And{Left, Right}`, `...Or{Left, Right}`), so `==` is upstream's structural `PartialEq`; Display is `String()`. A missing predicate (`Option<Rc<...>>`) is a nil interface.
- `KeyContext` mutators take a pointer receiver; `Clone()` copies the entries (Go slices alias). `Primary` returns a nil-able `*ContextEntry`, `Secondary` an `iter.Seq`, `Get` `(SharedString, bool)`. `TryFrom<&str>` is `KeyContextParse`. Debug is `String()`.
- `NewKeyBinding(keystrokes, action, context Option[string])` panics on parse errors as upstream's `unwrap`. `MatchKeystrokes` and `Keymap.BindingsForInput` take `[]K` where `*K` implements `AsKeystroke` (upstream `&[impl AsKeystroke]`), so both `[]Keystroke` and `[]KeybindingKeystroke` work; `MatchKeystrokes` returns `(pending, ok)` for `Option<bool>`.
- `Keymap.Bindings()` returns the keymap's own slice and `BindingsForAction` returns `[]*KeyBinding` into it (upstream's double-ended iterators; callers reverse by index). `Keymap::default()` is `DefaultKeymap()`; `NewKeymap` returns `*Keymap` (upstream shares it as `Rc<RefCell<Keymap>>`).
- key_dispatch.rs and tab_stop.rs are crate-private: `dispatchTree` (`newDispatchTree`, `pushNode`, `setActiveNode`, `setKeyContext`, `setFocusID`, `setViewID`, `popNode`, `reuseSubtree`, `truncate`, `onKeyEvent`, `onModifiersChanged`, `onAction`, `focusContains`, `availableActions`, `isActionAvailable`, `bindingsForAction`, `highestPrecedenceBindingForAction`, `bindingsForInput`, `possibleNextBindingsForInput`, `dispatchKey`, `flushDispatch`, `dispatchPath`, `focusPath`, `viewPathReversed`, `node`, `focusableNodeID`, `rootNodeID`, `activeNodeID`), `dispatchNode`, `dispatchNodeID`, `reusedSubtree` (`refreshNodeID`, `containsFocus`), `replay`, `dispatchResult`, `keyListener`, `modifiersChangedListener`, `dispatchActionListener`; `tabStopMap` (`insert`, `beginGroup`, `endGroup`, `clear`, `next`, `prev`, `replay`, `paintIndex`, `tabStopCount`, field `insertionHistory`). KeyBinding fields used by the window stages: `action`, `keystrokes`, `contextPredicate`, `meta`, `actionInput`.
- `tabStopMap` clones focus handles where upstream clones (`insert`, `next`, `prev`) and releases the history's handles in `clear` (upstream's drop). Its tests build `FocusHandle{id, TabStop, TabIndex}` literals because the stub `newFocusHandle` panics.
- `dispatchTree.availableActions` keeps its result sorted by `D-TYPEORDER` instead of `TypeId`.

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-FLATENUM | `MouseButton::Navigate(NavigationDirection)` is flattened into `MouseButtonNavigateBack` / `MouseButtonNavigateForward` of one comparable `uint8` enum (`MouseButtonNavigate(dir)`, `(MouseButton).Navigation()`); `TouchPhase` puts `Moved` first so the zero value is upstream's `#[default]` (no `Ord` is derived, so order carries no meaning) | `gpui/src/interactive.rs:88-101`, `:453-480` | Keeps the button usable in `switch`, as a map key and as a constant; keeps zero-valued events equal to upstream's `Default` |
| D-TYPEORDER | `DispatchTree::available_actions` orders actions by the action type's `PkgPath() + "." + String()` | `gpui/src/key_dispatch.rs:364-380` | `reflect.Type` has no ordering; upstream's `TypeId` order is arbitrary per build, so any total order keeps the binary-search dedup |
| D-GESTUREDEFAULTS | `PlatformGestures` default methods live on `NullPlatformGestures`, which implementations embed; `GestureTuning::default()` is `DefaultGestureTuning()`; `LongPressEvent::default()` (phase Started) is `DefaultLongPressEvent()`, the zero value has `TouchPhaseMoved`; `ScrollPhysics` is a sealed interface over `ScrollPhysicsExponential` / `ScrollPhysicsFrictionSpline`; `GestureKinds::NONE` / `ALL` are package vars | `gpui/src/gestures.rs:110-200`, `:349-375`, `:445-500` | Go interfaces have no default methods and Go has no struct constants or custom zero values |
| D-SEC-KEYCTX | `KeyContextParse` returns an `unexpected character` error for a character that starts no identifier instead of recursing forever, and context predicates nest at most 128 levels (`maxPredicateDepth`, `errPredicateTooDeep`) instead of overflowing the stack (SEC-033, 034) | `gpui/src/keymap/context.rs:63` | `keymap_context.go` |
