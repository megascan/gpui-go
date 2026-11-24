# Parity: core-input

Scope: gpui: keymap, key dispatch, interactive events, input handler, gestures, tab stops. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/gestures.rs

Go target: `gestures.go` (package `gpui`), tests in `gestures_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `OngoingScroll` | struct | 45 |  | `OngoingScroll` | todo |
| `OngoingScroll::filter` | method | 55 |  | `(OngoingScroll).Filter` | todo |
| `GestureTuning` | struct | 110 |  | `GestureTuning` | todo |
| `ScrollPhysics` | enum | 147 |  | `ScrollPhysics` | todo |
| `ScrollPhysics::ios` | assoc-fn | 173 |  | `ScrollPhysicsIos` | todo |
| `ScrollPhysics::android` | assoc-fn | 185 |  | `ScrollPhysicsAndroid` | todo |
| `GestureKinds` | struct | 349 |  | `GestureKinds` | todo |
| `GestureKinds::NONE` | assoc-const | 363 |  | `GestureKindsNone` | todo |
| `GestureKinds::ALL` | assoc-const | 371 |  | `GestureKindsAll` | todo |
| `TouchDragEvent` | struct | 382 |  | `TouchDragEvent` | todo |
| `BackGestureEvent` | struct | 410 |  | `BackGestureEvent` | todo |
| `BackGesturePhase` | enum | 421 |  | `BackGesturePhase` | todo |
| `LongPressEvent` | struct | 445 |  | `LongPressEvent` | todo |
| `PlatformGestures` | trait | 477 |  | `PlatformGestures` | todo |
| `PlatformGestures::tuning` | trait-method | 479 |  | `PlatformGestures.Tuning` | todo |
| `PlatformGestures::native_recognizers` | trait-method | 484 |  | `PlatformGestures.NativeRecognizers` | todo |
| `NullPlatformGestures` | struct | 491 |  | `NullPlatformGestures` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `ongoing_scroll_locks_to_dominant_axis` | test | 1191 |  | `TestGestures_OngoingScrollLocksToDominantAxis` | todo |
| `ongoing_scroll_unlocks_when_direction_changes` | test | 1210 |  | `TestGestures_OngoingScrollUnlocksWhenDirectionChanges` | todo |
| `ongoing_scroll_starts_new_gesture_at_timeout_boundary` | test | 1227 |  | `TestGestures_OngoingScrollStartsNewGestureAtTimeoutBoundary` | todo |
| `ongoing_scroll_ignores_zero_delta_and_resets_when_ended` | test | 1244 |  | `TestGestures_OngoingScrollIgnoresZeroDeltaAndResetsWhenEnded` | todo |
| `ongoing_scroll_ignores_zero_delta_movement` | test | 1269 |  | `TestGestures_OngoingScrollIgnoresZeroDeltaMovement` | todo |
| `ongoing_scroll_supports_moved_only_platforms` | test | 1293 |  | `TestGestures_OngoingScrollSupportsMovedOnlyPlatforms` | todo |
| `touch_within_slop_resolves_to_tap` | test | 1303 |  | `TestGestures_TouchWithinSlopResolvesToTap` | todo |
| `consecutive_taps_accumulate_tap_count` | test | 1330 |  | `TestGestures_ConsecutiveTapsAccumulateTapCount` | todo |
| `touch_beyond_slop_resolves_to_pan` | test | 1370 |  | `TestGestures_TouchBeyondSlopResolvesToPan` | todo |
| `touch_pan_stays_locked_to_its_initial_dominant_axis` | test | 1410 |  | `TestGestures_TouchPanStaysLockedToItsInitialDominantAxis` | todo |
| `touch_pan_locks_to_horizontal_axis` | test | 1437 |  | `TestGestures_TouchPanLocksToHorizontalAxis` | todo |
| `predicted_positions_lead_the_pan_but_totals_converge_on_release` | test | 1455 |  | `TestGestures_PredictedPositionsLeadThePanButTotalsConvergeOnRelease` | todo |
| `predicted_positions_do_not_emit_false_reversals` | test | 1497 |  | `TestGestures_PredictedPositionsDoNotEmitFalseReversals` | todo |
| `predicted_overshoot_folds_into_the_fling_without_scrolling_backwards` | test | 1552 |  | `TestGestures_PredictedOvershootFoldsIntoTheFlingWithoutScrollingBackwards` | todo |
| `fast_release_starts_momentum_that_decays_to_a_stop` | test | 1620 |  | `TestGestures_FastReleaseStartsMomentumThatDecaysToAStop` | todo |
| `diagonal_release_flings_only_on_locked_axis` | test | 1673 |  | `TestGestures_DiagonalReleaseFlingsOnlyOnLockedAxis` | todo |
| `slow_release_does_not_start_momentum` | test | 1713 |  | `TestGestures_SlowReleaseDoesNotStartMomentum` | todo |
| `new_touch_interrupts_momentum` | test | 1735 |  | `TestGestures_NewTouchInterruptsMomentum` | todo |
| `catching_a_fling_pans_immediately_and_never_taps` | test | 1779 |  | `TestGestures_CatchingAFlingPansImmediatelyAndNeverTaps` | todo |
| `catching_a_fling_takes_the_axis_from_the_new_touch` | test | 1832 |  | `TestGestures_CatchingAFlingTakesTheAxisFromTheNewTouch` | todo |
| `cancelled_pan_emits_cancelled_scroll_and_no_tap` | test | 1886 |  | `TestGestures_CancelledPanEmitsCancelledScrollAndNoTap` | todo |
| `concurrent_touches_are_ignored_while_one_is_active` | test | 1916 |  | `TestGestures_ConcurrentTouchesAreIgnoredWhileOneIsActive` | todo |
| `spline_position_table_matches_the_bezier_curve` | test | 1952 |  | `TestGestures_SplinePositionTableMatchesTheBezierCurve` | todo |
| `fling_curves_are_sane_for_both_physics` | test | 1989 |  | `TestGestures_FlingCurvesAreSaneForBothPhysics` | todo |
| `momentum_is_frame_rate_independent` | test | 2011 |  | `TestGestures_MomentumIsFrameRateIndependent` | todo |
| `flick_velocity_reflects_release_speed_not_window_average` | test | 2066 |  | `TestGestures_FlickVelocityReflectsReleaseSpeedNotWindowAverage` | todo |
| `samples_before_a_pause_do_not_contribute_velocity` | test | 2090 |  | `TestGestures_SamplesBeforeAPauseDoNotContributeVelocity` | todo |
| `claimed_touch_drag_emits_phased_stream_without_pan_or_tap` | test | 2109 |  | `TestGestures_ClaimedTouchDragEmitsPhasedStreamWithoutPanOrTap` | todo |
| `unclaimed_touch_drag_remains_a_pan_candidate` | test | 2144 |  | `TestGestures_UnclaimedTouchDragRemainsAPanCandidate` | todo |
| `claimed_long_press_emits_phased_stream_without_tap` | test | 2166 |  | `TestGestures_ClaimedLongPressEmitsPhasedStreamWithoutTap` | todo |
| `unclaimed_long_press_remains_a_tap_candidate` | test | 2199 |  | `TestGestures_UnclaimedLongPressRemainsATapCandidate` | todo |
| `unclaimed_long_press_can_still_become_a_pan` | test | 2218 |  | `TestGestures_UnclaimedLongPressCanStillBecomeAPan` | todo |
| `long_press_offer_is_one_shot_and_specific_to_pending_touch` | test | 2240 |  | `TestGestures_LongPressOfferIsOneShotAndSpecificToPendingTouch` | todo |
| `long_press_cannot_be_offered_after_pending_touch_resolves` | test | 2256 |  | `TestGestures_LongPressCannotBeOfferedAfterPendingTouchResolves` | todo |
| `claimed_long_press_emits_cancelled_for_its_touch_only` | test | 2272 |  | `TestGestures_ClaimedLongPressEmitsCancelledForItsTouchOnly` | todo |
| `unrelated_touch_cannot_end_or_cancel_pending_touch` | test | 2294 |  | `TestGestures_UnrelatedTouchCannotEndOrCancelPendingTouch` | todo |
| `completed_touch_id_cannot_claim_replacement_touch` | test | 2310 |  | `TestGestures_CompletedTouchIDCannotClaimReplacementTouch` | todo |

## gpui/src/input.rs

Go target: `input.go` (package `gpui`), tests in `input_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `EntityInputHandler` | trait | 26 |  | `EntityInputHandler` | todo |
| `EntityInputHandler::text_for_range` | trait-method | 28 |  | `EntityInputHandler.TextForRange` | todo |
| `EntityInputHandler::selected_text_range` | trait-method | 37 |  | `EntityInputHandler.SelectedTextRange` | todo |
| `EntityInputHandler::marked_text_range` | trait-method | 45 |  | `EntityInputHandler.MarkedTextRange` | todo |
| `EntityInputHandler::unmark_text` | trait-method | 52 |  | `EntityInputHandler.UnmarkText` | todo |
| `EntityInputHandler::paste` | trait-method | 55 |  | `EntityInputHandler.Paste` | todo |
| `EntityInputHandler::replace_text_in_range` | trait-method | 62 |  | `EntityInputHandler.ReplaceTextInRange` | todo |
| `EntityInputHandler::replace_and_mark_text_in_range` | trait-method | 71 |  | `EntityInputHandler.ReplaceAndMarkTextInRange` | todo |
| `EntityInputHandler::bounds_for_range` | trait-method | 81 |  | `EntityInputHandler.BoundsForRange` | todo |
| `EntityInputHandler::selection_bounds_for_range` | trait-method | 90 |  | `EntityInputHandler.SelectionBoundsForRange` | todo |
| `EntityInputHandler::character_index_for_point` | trait-method | 101 |  | `EntityInputHandler.CharacterIndexForPoint` | todo |
| `EntityInputHandler::set_selected_text_range` | trait-method | 109 |  | `EntityInputHandler.SetSelectedTextRange` | todo |
| `EntityInputHandler::text_length_utf16` | trait-method | 118 |  | `EntityInputHandler.TextLengthUTF16` | todo |
| `EntityInputHandler::accepts_text_input` | trait-method | 127 |  | `EntityInputHandler.AcceptsTextInput` | todo |
| `EntityInputHandler::text_input_configuration` | trait-method | 132 |  | `EntityInputHandler.TextInputConfiguration` | todo |
| `EntityInputHandler::text_input_editable_range` | trait-method | 141 |  | `EntityInputHandler.TextInputEditableRange` | todo |
| `ElementInputHandler` | struct | 152 |  | `ElementInputHandler` | todo |
| `ElementInputHandler::new` | assoc-fn | 161 |  | `NewElementInputHandler` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `text_input_configuration_and_focus_state_are_forwarded_on_change` | gpui::test | 331 |  | `TestInput_TextInputConfigurationAndFocusStateAreForwardedOnChange` | todo |
| `text_notifications_require_a_focused_input_and_track_field_changes` | gpui::test | 421 |  | `TestInput_TextNotificationsRequireAFocusedInputAndTrackFieldChanges` | todo |

## gpui/src/interactive.rs

Go target: `interactive.go` (package `gpui`), tests in `interactive_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `InputEvent` | trait | 9 |  | `InputEvent` | todo |
| `InputEvent::to_platform_input` | trait-method | 11 |  | `InputEvent.ToPlatformInput` | todo |
| `KeyEvent` | trait | 15 |  | `KeyEvent` | todo |
| `MouseEvent` | trait | 18 |  | `MouseEvent` | todo |
| `GestureEvent` | trait | 21 |  | `GestureEvent` | todo |
| `KeyDownEvent` | struct | 25 |  | `KeyDownEvent` | todo |
| `KeyUpEvent` | struct | 47 |  | `KeyUpEvent` | todo |
| `ModifiersChangedEvent` | struct | 62 |  | `ModifiersChangedEvent` | todo |
| `TouchPhase` | enum | 88 |  | `TouchPhase` | todo |
| `TouchId` | struct | 109 |  | `TouchID` | todo |
| `TouchEvent` | struct | 119 |  | `TouchEvent` | todo |
| `MouseDownEvent` | struct | 148 |  | `MouseDownEvent` | todo |
| `MouseDownEvent::is_focusing` | method | 175 |  | `(MouseDownEvent).IsFocusing` | todo |
| `MouseUpEvent` | struct | 185 |  | `MouseUpEvent` | todo |
| `MouseUpEvent::is_focusing` | method | 210 |  | `(MouseUpEvent).IsFocusing` | todo |
| `MouseClickEvent` | struct | 220 |  | `MouseClickEvent` | todo |
| `PressureStage` | enum | 230 |  | `PressureStage` | todo |
| `MousePressureEvent` | struct | 243 |  | `MousePressureEvent` | todo |
| `KeyboardClickEvent` | struct | 264 |  | `KeyboardClickEvent` | todo |
| `TouchClickEvent` | struct | 275 |  | `TouchClickEvent` | todo |
| `ClickEvent` | enum | 290 |  | `ClickEvent` | todo |
| `ClickEvent::modifiers` | method | 310 |  | `(ClickEvent).Modifiers` | todo |
| `ClickEvent::position` | method | 328 |  | `(ClickEvent).Position` | todo |
| `ClickEvent::mouse_position` | method | 341 |  | `(ClickEvent).MousePosition` | todo |
| `ClickEvent::is_right_click` | method | 353 |  | `(ClickEvent).IsRightClick` | todo |
| `ClickEvent::is_middle_click` | method | 367 |  | `(ClickEvent).IsMiddleClick` | todo |
| `ClickEvent::is_secondary` | method | 381 |  | `(ClickEvent).IsSecondary` | todo |
| `ClickEvent::standard_click` | method | 396 |  | `(ClickEvent).StandardClick` | todo |
| `ClickEvent::first_focus` | method | 411 |  | `(ClickEvent).FirstFocus` | todo |
| `ClickEvent::click_count` | method | 424 |  | `(ClickEvent).ClickCount` | todo |
| `ClickEvent::is_keyboard` | method | 433 |  | `(ClickEvent).IsKeyboard` | todo |
| `KeyboardButton` | enum | 443 |  | `KeyboardButton` | todo |
| `MouseButton` | enum | 453 |  | `MouseButton` | todo |
| `MouseButton::all` | assoc-fn | 470 |  | `MouseButtonAll` | todo |
| `NavigationDirection` | enum | 483 |  | `NavigationDirection` | todo |
| `MouseMoveEvent` | struct | 494 |  | `MouseMoveEvent` | todo |
| `MouseMoveEvent::dragging` | method | 515 |  | `(MouseMoveEvent).Dragging` | todo |
| `ScrollEvent` | struct | 522 |  | `ScrollEvent` | todo |
| `ScrollWheelEvent` | type | 542 |  | `ScrollWheelEvent` | todo |
| `ScrollSource` | enum | 546 |  | `ScrollSource` | todo |
| `ScrollDelta` | enum | 575 |  | `ScrollDelta` | todo |
| `PinchEvent` | struct | 592 |  | `PinchEvent` | todo |
| `ScrollDelta::precise` | method | 627 |  | `(ScrollDelta).Precise` | todo |
| `ScrollDelta::pixel_delta` | method | 635 |  | `(ScrollDelta).PixelDelta` | todo |
| `ScrollDelta::coalesce` | method | 646 |  | `(ScrollDelta).Coalesce` | todo |
| `MouseExitEvent` | struct | 687 |  | `MouseExitEvent` | todo |
| `ExternalPaths` | struct | 715 |  | `ExternalPaths` | todo |
| `ExternalPaths::paths` | method | 719 |  | `(ExternalPaths).Paths` | todo |
| `ExternalDragPayload` | enum | 727 |  | `ExternalDragPayload` | todo |
| `FileDragPaths` | struct | 735 |  | `FileDragPaths` | todo |
| `FileDragPaths::new` | assoc-fn | 739 |  | `NewFileDragPaths` | todo |
| `FileDragPaths::entries` | method | 744 |  | `(FileDragPaths).Entries` | todo |
| `FileDropEvent` | enum | 758 |  | `FileDropEvent` | todo |
| `PlatformInput` | enum | 792 |  | `PlatformInput` | todo |
| `PlatformInput::kind_name` | method | 869 |  | `(PlatformInput).KindName` | todo |
| `PlatformInput::touch_event` | method | 890 |  | `(PlatformInput).TouchEvent` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_on_events` | gpui::test | 937 |  | `TestInteractive_OnEvents` | todo |
| `test_multi_modifier_gesture_does_not_dispatch_standalone_modifier_binding` | gpui::test | 972 |  | `TestInteractive_MultiModifierGestureDoesNotDispatchStandaloneModifierBinding` | todo |

## gpui/src/key_dispatch.rs

Go target: `key_dispatch.go` (package `gpui`), tests in `key_dispatch_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_keybinding_for_action_bounds` | test | 961 |  | `TestKeyDispatch_KeybindingForActionBounds` | todo |
| `test_bindings_for_action_hides_targeted_unbind_in_active_context` | test | 979 |  | `TestKeyDispatch_BindingsForActionHidesTargetedUnbindInActiveContext` | todo |
| `test_bindings_for_action_keeps_targeted_binding_outside_unbind_context` | test | 1011 |  | `TestKeyDispatch_BindingsForActionKeepsTargetedBindingOutsideUnbindContext` | todo |
| `test_overlay_after_base_restores_shadowed_picker_binding` | test | 1048 |  | `TestKeyDispatch_OverlayAfterBaseRestoresShadowedPickerBinding` | todo |
| `test_pending_has_binding_state` | test | 1100 |  | `TestKeyDispatch_PendingHasBindingState` | todo |
| `test_pending_input_observers_notified_on_focus_change_and_blur` | crate::test | 1140 |  | `TestKeyDispatch_PendingInputObserversNotifiedOnFocusChangeAndBlur` | todo |
| `test_standalone_modifier_dispatch` | crate::test | 1385 |  | `TestKeyDispatch_StandaloneModifierDispatch` | todo |
| `test_unbound_standalone_modifier_observation` | crate::test | 1445 |  | `TestKeyDispatch_UnboundStandaloneModifierObservation` | todo |
| `test_raw_modifier_handler_can_suppress_observation` | crate::test | 1476 |  | `TestKeyDispatch_RawModifierHandlerCanSuppressObservation` | todo |
| `test_pending_modifier_observed_when_binding_resolves` | crate::test | 1539 |  | `TestKeyDispatch_PendingModifierObservedWhenBindingResolves` | todo |
| `test_combined_shortcut_has_no_standalone_modifier` | crate::test | 1582 |  | `TestKeyDispatch_CombinedShortcutHasNoStandaloneModifier` | todo |
| `test_keystroke_input_preference` | crate::test | 1626 |  | `TestKeyDispatch_KeystrokeInputPreference` | todo |
| `test_input_preference_rechecked_after_interception` | crate::test | 1686 |  | `TestKeyDispatch_InputPreferenceRecheckedAfterInterception` | todo |
| `test_keystroke_interceptors_can_consume_preferred_character_input` | crate::test | 1736 |  | `TestKeyDispatch_KeystrokeInterceptorsCanConsumePreferredCharacterInput` | todo |
| `test_consumed_modifier_preserves_pending_input` | crate::test | 1757 |  | `TestKeyDispatch_ConsumedModifierPreservesPendingInput` | todo |
| `test_printable_pending_input_replays_on_timeout` | crate::test | 1817 |  | `TestKeyDispatch_PrintablePendingInputReplaysOnTimeout` | todo |
| `test_pending_input_timeout_dispatches_shorter_binding` | crate::test | 1851 |  | `TestKeyDispatch_PendingInputTimeoutDispatchesShorterBinding` | todo |
| `test_running_pending_input_timeout_resets_when_binding_advances` | crate::test | 1885 |  | `TestKeyDispatch_RunningPendingInputTimeoutResetsWhenBindingAdvances` | todo |
| `test_pending_input_timeout_starts_when_binding_becomes_ambiguous` | crate::test | 1919 |  | `TestKeyDispatch_PendingInputTimeoutStartsWhenBindingBecomesAmbiguous` | todo |
| `test_invalid_continuation_while_timeout_paused_replays_pending_input` | crate::test | 1952 |  | `TestKeyDispatch_InvalidContinuationWhileTimeoutPausedReplaysPendingInput` | todo |
| `test_pending_input_timeout_pauses_and_resumes` | crate::test | 1989 |  | `TestKeyDispatch_PendingInputTimeoutPausesAndResumes` | todo |
| `test_pending_input_timeout_resumes_when_owner_is_released` | crate::test | 2043 |  | `TestKeyDispatch_PendingInputTimeoutResumesWhenOwnerIsReleased` | todo |
| `test_pending_input_timeout_resets_when_binding_advances` | crate::test | 2079 |  | `TestKeyDispatch_PendingInputTimeoutResetsWhenBindingAdvances` | todo |
| `test_clearing_pending_input_invalidates_timeout_pause` | crate::test | 2128 |  | `TestKeyDispatch_ClearingPendingInputInvalidatesTimeoutPause` | todo |
| `test_input_handler_pending` | crate::test | 2150 |  | `TestKeyDispatch_InputHandlerPending` | todo |

## gpui/src/keymap.rs

Go target: `keymap.go` (package `gpui`), tests in `keymap_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `KeymapVersion` | struct | 14 |  | `KeymapVersion` | todo |
| `Keymap` | struct | 18 |  | `Keymap` | todo |
| `BindingIndex` | struct | 27 |  | `BindingIndex` | todo |
| `Keymap::new` | assoc-fn | 51 |  | `NewKeymap` | todo |
| `Keymap::version` | method | 58 |  | `(Keymap).Version` | todo |
| `Keymap::add_bindings` | method | 63 |  | `(Keymap).AddBindings` | todo |
| `Keymap::clear` | method | 81 |  | `(Keymap).Clear` | todo |
| `Keymap::bindings` | method | 89 |  | `(Keymap).Bindings` | todo |
| `Keymap::bindings_for_action` | method | 95 |  | `(Keymap).BindingsForAction` | todo |
| `Keymap::all_bindings_for_input` | method | 138 |  | `(Keymap).AllBindingsForInput` | todo |
| `Keymap::bindings_for_input` | method | 165 |  | `(Keymap).BindingsForInput` | todo |
| `Keymap::possible_next_bindings_for_input` | method | 285 |  | `(Keymap).PossibleNextBindingsForInput` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_keymap` | test | 352 |  | `TestKeymap_Keymap` | todo |
| `test_depth_precedence` | test | 393 |  | `TestKeymap_DepthPrecedence` | todo |
| `test_keymap_disabled` | test | 417 |  | `TestKeymap_KeymapDisabled` | todo |
| `test_multiple_keystroke_binding_disabled` | test | 473 |  | `TestKeymap_MultipleKeystrokeBindingDisabled` | todo |
| `test_override_multikey` | test | 568 |  | `TestKeymap_OverrideMultikey` | todo |
| `test_simple_disable` | test | 603 |  | `TestKeymap_SimpleDisable` | todo |
| `test_disable_weaker_sources_only` | test | 622 |  | `TestKeymap_DisableWeakerSourcesOnly` | todo |
| `test_fail_to_disable` | test | 691 |  | `TestKeymap_FailToDisable` | todo |
| `test_disable_deeper` | test | 714 |  | `TestKeymap_DisableDeeper` | todo |
| `test_pending_match_enabled` | test | 736 |  | `TestKeymap_PendingMatchEnabled` | todo |
| `test_pending_match_enabled_extended` | test | 759 |  | `TestKeymap_PendingMatchEnabledExtended` | todo |
| `test_overriding_prefix` | test | 801 |  | `TestKeymap_OverridingPrefix` | todo |
| `test_context_precedence_with_same_source` | test | 824 |  | `TestKeymap_ContextPrecedenceWithSameSource` | todo |
| `test_bindings_for_action` | test | 851 |  | `TestKeymap_BindingsForAction` | todo |
| `test_targeted_unbind_ignores_target_context` | test | 882 |  | `TestKeymap_TargetedUnbindIgnoresTargetContext` | todo |
| `test_bindings_for_action_keeps_binding_for_narrower_targeted_unbind` | test | 907 |  | `TestKeymap_BindingsForActionKeepsBindingForNarrowerTargetedUnbind` | todo |
| `test_bindings_for_action_removes_binding_for_broader_targeted_unbind` | test | 939 |  | `TestKeymap_BindingsForActionRemovesBindingForBroaderTargetedUnbind` | todo |
| `test_source_precedence_sorting` | test | 956 |  | `TestKeymap_SourcePrecedenceSorting` | todo |
| `test_possible_next_bindings_exclude_null_bindings` | test | 984 |  | `TestKeymap_PossibleNextBindingsExcludeNullBindings` | todo |
| `test_possible_next_bindings_exclude_unbound_bindings` | test | 1002 |  | `TestKeymap_PossibleNextBindingsExcludeUnboundBindings` | todo |
| `test_possible_next_bindings_keep_user_binding_over_base_keymap_null` | test | 1024 |  | `TestKeymap_PossibleNextBindingsKeepUserBindingOverBaseKeymapNull` | todo |
| `test_possible_next_bindings_apply_disabling_by_context` | test | 1047 |  | `TestKeymap_PossibleNextBindingsApplyDisablingByContext` | todo |
| `test_possible_next_bindings_exclude_longer_chord_disabled_in_deeper_context` | test | 1079 |  | `TestKeymap_PossibleNextBindingsExcludeLongerChordDisabledInDeeperContext` | todo |
| `test_possible_next_bindings_exclude_disabled_bindings_with_typed_input` | test | 1104 |  | `TestKeymap_PossibleNextBindingsExcludeDisabledBindingsWithTypedInput` | todo |

## gpui/src/tab_stop.rs

Go target: `tab_stop.go` (package `gpui`), tests in `tab_stop_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TabStopOperation` | enum | 23 |  | `TabStopOperation` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_tab_handles` | test | 193 |  | `TestTabStop_TabHandles` | todo |
| `test_tab_non_stop_filtering` | test | 282 |  | `TestTabStop_TabNonStopFiltering` | todo |
| `test_insert_after_sorting` | test | 393 |  | `TestTabStop_InsertAfterSorting` | todo |
| `test_with_disabled_tab_stop` | test | 409 |  | `TestTabStop_WithDisabledTabStop` | todo |
| `test_with_multiple_disabled_tab_stops` | test | 419 |  | `TestTabStop_WithMultipleDisabledTabStops` | todo |
| `test_tab_group_functionality` | test | 430 |  | `TestTabStop_TabGroupFunctionality` | todo |
| `test_sibling_groups` | test | 441 |  | `TestTabStop_SiblingGroups` | todo |
| `test_nested_group` | test | 455 |  | `TestTabStop_NestedGroup` | todo |
| `test_sibling_nested_groups` | test | 469 |  | `TestTabStop_SiblingNestedGroups` | todo |
| `test_sibling_nested_groups_out_of_order` | test | 486 |  | `TestTabStop_SiblingNestedGroupsOutOfOrder` | todo |

## gpui/src/keymap/binding.rs

Go target: `keymap_binding.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `KeyBinding` | struct | 10 |  | `KeyBinding` | todo |
| `KeyBinding::new` | assoc-fn | 35 |  | `NewKeyBinding` | todo |
| `KeyBinding::load` | assoc-fn | 51 |  | `KeyBindingLoad` | todo |
| `KeyBinding::with_meta` | method | 101 |  | `(KeyBinding).WithMeta` | todo |
| `KeyBinding::set_meta` | method | 107 |  | `(KeyBinding).SetMeta` | todo |
| `KeyBinding::match_keystrokes` | method | 112 |  | `(KeyBinding).MatchKeystrokes` | todo |
| `KeyBinding::keystrokes` | method | 127 |  | `(KeyBinding).Keystrokes` | todo |
| `KeyBinding::action` | method | 132 |  | `(KeyBinding).Action` | todo |
| `KeyBinding::predicate` | method | 137 |  | `(KeyBinding).Predicate` | todo |
| `KeyBinding::meta` | method | 142 |  | `(KeyBinding).Meta` | todo |
| `KeyBinding::action_input` | method | 147 |  | `(KeyBinding).ActionInput` | todo |
| `KeyBindingMetaIndex` | struct | 166 |  | `KeyBindingMetaIndex` | todo |

## gpui/src/keymap/context.rs

Go target: `keymap_context.go` (package `gpui`), tests in `keymap_context_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `KeyContext` | struct | 10 |  | `KeyContext` | todo |
| `ContextEntry` | struct | 14 |  | `ContextEntry` | todo |
| `KeyContext::new_with_defaults` | assoc-fn | 31 |  | `NewKeyContextWithDefaults` | todo |
| `KeyContext::primary` | method | 48 |  | `(KeyContext).Primary` | todo |
| `KeyContext::secondary` | method | 53 |  | `(KeyContext).Secondary` | todo |
| `KeyContext::parse` | assoc-fn | 63 |  | `KeyContextParse` | todo |
| `KeyContext::is_empty` | method | 96 |  | `(KeyContext).IsEmpty` | todo |
| `KeyContext::clear` | method | 101 |  | `(KeyContext).Clear` | todo |
| `KeyContext::extend` | method | 106 |  | `(KeyContext).Extend` | todo |
| `KeyContext::add` | method | 115 |  | `(KeyContext).Add` | todo |
| `KeyContext::set` | method | 124 |  | `(KeyContext).Set` | todo |
| `KeyContext::contains` | method | 135 |  | `(KeyContext).Contains` | todo |
| `KeyContext::get` | method | 140 |  | `(KeyContext).Get` | todo |
| `KeyBindingContextPredicate` | enum | 170 |  | `KeyBindingContextPredicate` | todo |
| `KeyBindingContextPredicate::parse` | assoc-fn | 247 |  | `KeyBindingContextPredicateParse` | todo |
| `KeyBindingContextPredicate::depth_of` | method | 258 |  | `(KeyBindingContextPredicate).DepthOf` | todo |
| `KeyBindingContextPredicate::eval` | method | 269 |  | `(KeyBindingContextPredicate).Eval` | todo |
| `KeyBindingContextPredicate::eval_inner` | method | 274 |  | `(KeyBindingContextPredicate).EvalInner` | todo |
| `KeyBindingContextPredicate::is_superset` | method | 325 |  | `(KeyBindingContextPredicate).IsSuperset` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_actions_definition` | test | 521 |  | `TestContext_ActionsDefinition` | todo |
| `test_parse_context` | test | 537 |  | `TestContext_ParseContext` | todo |
| `test_parse_identifiers` | test | 551 |  | `TestContext_ParseIdentifiers` | todo |
| `test_parse_negations` | test | 564 |  | `TestContext_ParseNegations` | todo |
| `test_parse_equality_operators` | test | 576 |  | `TestContext_ParseEqualityOperators` | todo |
| `test_parse_boolean_operators` | test | 594 |  | `TestContext_ParseBooleanOperators` | todo |
| `test_parse_parenthesized_expressions` | test | 654 |  | `TestContext_ParseParenthesizedExpressions` | todo |
| `test_is_superset` | test | 675 |  | `TestContext_IsSuperset` | todo |
| `test_child_operator` | test | 698 |  | `TestContext_ChildOperator` | todo |
| `test_not_operator` | test | 734 |  | `TestContext_NotOperator` | todo |
| `test_context_display` | test | 814 |  | `TestContext_ContextDisplay` | todo |

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
| `AsKeystroke` | trait | 11 |  | `AsKeystroke` | todo |
| `AsKeystroke::as_keystroke` | trait-method | 13 |  | `AsKeystroke.AsKeystroke` | todo |
| `Keystroke` | struct | 18 |  | `Keystroke` | todo |
| `KeybindingKeystroke` | struct | 37 |  | `KeybindingKeystroke` | todo |
| `InvalidKeystrokeError` | struct | 51 |  | `InvalidKeystrokeError` | todo |
| `KEYSTROKE_PARSE_EXPECTED_MESSAGE` | const | 69 |  | `KeystrokeParseExpectedMessage` | todo |
| `Keystroke::should_match` | method | 82 |  | `(Keystroke).ShouldMatch` | todo |
| `Keystroke::parse` | assoc-fn | 120 |  | `KeystrokeParse` | todo |
| `Keystroke::parse_with_os` | assoc-fn | 125 |  | `KeystrokeParseWithOs` | todo |
| `Keystroke::unparse` | method | 241 |  | `(Keystroke).Unparse` | todo |
| `Keystroke::display` | method | 246 |  | `(Keystroke).Display` | todo |
| `Keystroke::is_ime_in_progress` | method | 256 |  | `(Keystroke).IsIMEInProgress` | todo |
| `Keystroke::with_simulated_ime` | method | 268 |  | `(Keystroke).WithSimulatedIME` | todo |
| `Keystroke::is_modifier_key` | method | 293 |  | `(Keystroke).IsModifierKey` | todo |
| `KeybindingKeystroke::display` | method | 303 |  | `(KeybindingKeystroke).Display` | todo |
| `KeybindingKeystroke::new` | assoc-fn | 313 | windows | `NewKeybindingKeystroke` | todo |
| `KeybindingKeystroke::new_with_mapper` | assoc-fn | 322 |  | `NewKeybindingKeystrokeWithMapper` | todo |
| `KeybindingKeystroke::from_keystroke` | assoc-fn | 331 |  | `KeybindingKeystrokeFromKeystroke` | todo |
| `KeybindingKeystroke::inner` | method | 349 |  | `(KeybindingKeystroke).Inner` | todo |
| `KeybindingKeystroke::modifiers` | method | 359 |  | `(KeybindingKeystroke).Modifiers` | todo |
| `KeybindingKeystroke::key` | method | 375 |  | `(KeybindingKeystroke).Key` | todo |
| `KeybindingKeystroke::set_modifiers` | method | 387 |  | `(KeybindingKeystroke).SetModifiers` | todo |
| `KeybindingKeystroke::set_key` | method | 396 |  | `(KeybindingKeystroke).SetKey` | todo |
| `KeybindingKeystroke::unparse` | method | 405 |  | `(KeybindingKeystroke).Unparse` | todo |
| `KeybindingKeystroke::remove_key_char` | method | 417 |  | `(KeybindingKeystroke).RemoveKeyChar` | todo |
| `KeystrokeDisplay` | struct | 477 |  | `KeystrokeDisplay` | todo |
| `Modifiers` | struct | 492 |  | `Modifiers` | todo |
| `Modifiers::modified` | method | 519 |  | `(Modifiers).Modified` | todo |
| `Modifiers::secondary` | method | 527 |  | `(Modifiers).Secondary` | todo |
| `Modifiers::number_of_modifiers` | method | 536 |  | `(Modifiers).NumberOfModifiers` | todo |
| `Modifiers::none` | assoc-fn | 545 |  | `ModifiersNone` | todo |
| `Modifiers::command` | assoc-fn | 550 |  | `ModifiersCommand` | todo |
| `Modifiers::secondary_key` | assoc-fn | 558 |  | `ModifiersSecondaryKey` | todo |
| `Modifiers::windows` | assoc-fn | 573 |  | `ModifiersWindows` | todo |
| `Modifiers::super_key` | assoc-fn | 581 |  | `ModifiersSuperKey` | todo |
| `Modifiers::control` | assoc-fn | 589 |  | `ModifiersControl` | todo |
| `Modifiers::alt` | assoc-fn | 597 |  | `ModifiersAlt` | todo |
| `Modifiers::shift` | assoc-fn | 605 |  | `ModifiersShift` | todo |
| `Modifiers::function` | assoc-fn | 613 |  | `ModifiersFunction` | todo |
| `Modifiers::command_shift` | assoc-fn | 621 |  | `ModifiersCommandShift` | todo |
| `Modifiers::control_shift` | assoc-fn | 630 |  | `ModifiersControlShift` | todo |
| `Modifiers::is_subset_of` | method | 639 |  | `(Modifiers).IsSubsetOf` | todo |
| `Capslock` | struct | 701 |  | `Capslock` | todo |

