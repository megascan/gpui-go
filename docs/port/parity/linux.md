# Parity: linux

Scope: gpui_linux. Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_linux/src/linux.rs

Go target: `platform/linux/linux.go` (package `linux`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `current_platform` | fn | 40 |  | `CurrentPlatform` | todo |
| `linux_platform` | fn | 52 |  | `LinuxPlatform` | todo |

## gpui_linux/src/linux/dispatcher.rs

Go target: `platform/linux/internal/common/dispatcher.go` (package `common`), tests in `dispatcher_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PriorityQueueCalloopSender` | struct | 167 |  | `PriorityQueueCalloopSender` | todo |
| `PriorityQueueCalloopReceiver` | struct | 192 |  | `PriorityQueueCalloopReceiver` | todo |
| `PriorityQueueCalloopReceiver::new` | assoc-fn | 199 |  | `NewPriorityQueueCalloopReceiver` | todo |
| `ChannelError` | struct | 218 |  | `ChannelError` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `calloop_works` | test | 315 |  | `TestDispatcher_CalloopWorks` | todo |

## gpui_linux/src/linux/platform.rs

Go target: `platform/linux (+internal/common)/platform.go (+common.go)` (package `linux`), tests in `platform.go (+common._test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `keystroke_underlying_dead_key` | fn | 1602 | feat:wayland,feat:x11 | `KeystrokeUnderlyingDeadKey` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `power_mode_callbacks_can_reenter_and_replace_themselves` | test | 1773 |  | `TestPlatform_PowerModeCallbacksCanReenterAndReplaceThemselves` | todo |
| `rejects_null_xkb_context` | test | 1811 | feat:wayland,feat:x11 | `TestPlatform_RejectsNullXkbContext` | todo |
| `test_is_within_click_distance` | test | 1827 |  | `TestPlatform_IsWithinClickDistance` | todo |
| `guard_passes_through_and_releases_on_drop` | gpui::test | 1859 | feat:wayland,feat:x11 | `TestPlatformIdleSleepPrevention_GuardPassesThroughAndReleasesOnDrop` | todo |
| `acquisition_error_passes_through` | gpui::test | 1873 | feat:wayland,feat:x11 | `TestPlatformIdleSleepPrevention_AcquisitionErrorPassesThrough` | todo |
| `abandoned_acquisition_is_an_error` | gpui::test | 1887 | feat:wayland,feat:x11 | `TestPlatformIdleSleepPrevention_AbandonedAcquisitionIsAnError` | todo |
| `late_guard_after_timeout_is_released` | gpui::test | 1901 | feat:wayland,feat:x11 | `TestPlatformIdleSleepPrevention_LateGuardAfterTimeoutIsReleased` | todo |
| `reads_data_written_before_close` | test | 1943 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_ReadsDataWrittenBeforeClose` | todo |
| `returns_empty_when_writer_closes_without_writing` | test | 1953 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_ReturnsEmptyWhenWriterClosesWithoutWriting` | todo |
| `times_out_when_writer_never_writes` | test | 1962 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_TimesOutWhenWriterNeverWrites` | todo |
| `times_out_when_writer_stalls_after_partial_write` | test | 1980 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_TimesOutWhenWriterStallsAfterPartialWrite` | todo |
| `slow_writer_resets_deadline_between_chunks` | test | 1993 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_SlowWriterResetsDeadlineBetweenChunks` | todo |
| `reads_payload_larger_than_pipe_capacity` | test | 2016 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_ReadsPayloadLargerThanPipeCapacity` | todo |
| `failed_wayland_attach_preserves_headless_app_and_allows_retry` | test | 2049 | feat:wayland | `TestPlatformDisplayModeTests_FailedWaylandAttachPreservesHeadlessAppAndAllowsRetry` | todo |
| `headless_windows_block_switching_to_a_display_server` | test | 2102 | feat:wayland | `TestPlatformDisplayModeTests_HeadlessWindowsBlockSwitchingToADisplayServer` | todo |
| `display_modes_limit_switching` | test | 2134 |  | `TestPlatformDisplayModeTests_DisplayModesLimitSwitching` | todo |

## gpui_linux/src/linux/power.rs

Go target: `platform/linux/internal/common/power.go` (package `common`), tests in `power_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `maps_profiles_without_guessing_unknown_values` | test | 154 |  | `TestPower_MapsProfilesWithoutGuessingUnknownValues` | todo |
| `monitors_profiles_and_service_restarts` | test | 171 |  | `TestPower_MonitorsProfilesAndServiceRestarts` | todo |

## gpui_linux/src/linux/system_notifications.rs

Go target: `platform/linux/internal/common/system_notifications.go` (package `common`), tests in `system_notifications_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `caller_action_ids_do_not_collide_with_transport_action_ids` | test | 137 |  | `TestSystemNotifications_CallerActionIDsDoNotCollideWithTransportActionIDs` | todo |

## gpui_linux/src/linux/xdg_desktop_portal.rs

Go target: `platform/linux/internal/common/xdg_desktop_portal.go` (package `common`), tests in `xdg_desktop_portal_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Event` | enum | 13 |  | `Event` | todo |
| `XDPEventSource` | struct | 28 |  | `XDPEventSource` | todo |
| `XDPEventSource::new` | assoc-fn | 34 |  | `NewXDPEventSource` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `text_scale_factor_is_clamped_to_a_usable_range` | test | 238 |  | `TestXdgDesktopPortal_TextScaleFactorIsClampedToAUsableRange` | todo |

## gpui_linux/src/linux/wayland/client.rs

Go target: `platform/linux/wayland/client.go` (package `wayland`), tests in `client_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Globals` | struct | 243 |  | `Globals` | todo |
| `InProgressOutput` | struct | 325 |  | `InProgressOutput` | todo |
| `Output` | struct | 352 |  | `Output` | todo |
| `ClickState` | struct | 539 |  | `ClickState` | todo |
| `WaylandClientStatePtr` | struct | 577 |  | `WaylandClientStatePtr` | todo |
| `WaylandClientStatePtr::get_client` | method | 580 |  | `(WaylandClientStatePtr).GetClient` | todo |
| `WaylandClientStatePtr::try_get_client` | method | 603 |  | `(WaylandClientStatePtr).TryGetClient` | todo |
| `WaylandClientStatePtr::sole_output` | method | 608 |  | `(WaylandClientStatePtr).SoleOutput` | todo |
| `WaylandClientStatePtr::dispatch_scheduled_frames` | method | 619 |  | `(WaylandClientStatePtr).DispatchScheduledFrames` | todo |
| `WaylandClientStatePtr::schedule_frame_retry` | method | 639 |  | `(WaylandClientStatePtr).ScheduleFrameRetry` | todo |
| `WaylandClientStatePtr::get_serial` | method | 661 |  | `(WaylandClientStatePtr).GetSerial` | todo |
| `WaylandClientStatePtr::activation_context` | method | 665 |  | `(WaylandClientStatePtr).ActivationContext` | todo |
| `WaylandClientStatePtr::start_external_drag` | method | 673 |  | `(WaylandClientStatePtr).StartExternalDrag` | todo |
| `WaylandClientStatePtr::enable_ime` | method | 713 |  | `(WaylandClientStatePtr).EnableIME` | todo |
| `WaylandClientStatePtr::disable_ime` | method | 739 |  | `(WaylandClientStatePtr).DisableIME` | todo |
| `WaylandClientStatePtr::ime_enabled` | method | 750 |  | `(WaylandClientStatePtr).IMEEnabled` | todo |
| `WaylandClientStatePtr::update_ime_position` | method | 755 |  | `(WaylandClientStatePtr).UpdateIMEPosition` | todo |
| `WaylandClientStatePtr::handle_keyboard_layout_change` | method | 767 |  | `(WaylandClientStatePtr).HandleKeyboardLayoutChange` | todo |
| `WaylandClientStatePtr::drop_window` | method | 805 |  | `(WaylandClientStatePtr).DropWindow` | todo |
| `WaylandConnection` | struct | 884 |  | `WaylandConnection` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `rejects_required_globals_below_supported_versions` | test | 3207 |  | `TestClient_RejectsRequiredGlobalsBelowSupportedVersions` | todo |
| `leave_discards_pending_drag_uri_read` | test | 3275 |  | `TestClient_LeaveDiscardsPendingDragURIRead` | todo |
| `drop_then_leave_discards_pending_drag_uri_read` | test | 3294 |  | `TestClient_DropThenLeaveDiscardsPendingDragURIRead` | todo |
| `stale_completion_preserves_newer_drag_destination` | test | 3316 |  | `TestClient_StaleCompletionPreservesNewerDragDestination` | todo |
| `builds_terminated_uri_list_for_each_dragged_path` | test | 3360 |  | `TestClient_BuildsTerminatedURIListForEachDraggedPath` | todo |
| `skips_paths_that_have_no_file_url` | test | 3373 |  | `TestClient_SkipsPathsThatHaveNoFileURL` | todo |
| `builds_empty_uri_list_without_convertible_paths` | test | 3383 |  | `TestClient_BuildsEmptyURIListWithoutConvertiblePaths` | todo |
| `caches_cursor_rectangle_committed_after_done` | test | 3395 |  | `TestClient_CachesCursorRectangleCommittedAfterDone` | todo |
| `skips_unchanged_cursor_rectangle_after_done` | test | 3415 |  | `TestClient_SkipsUnchangedCursorRectangleAfterDone` | todo |
| `skips_cursor_rectangles_with_unchanged_protocol_coordinates` | test | 3433 |  | `TestClient_SkipsCursorRectanglesWithUnchangedProtocolCoordinates` | todo |

## gpui_linux/src/linux/wayland/display.rs

Go target: `platform/linux/wayland/display.go` (package `wayland`), tests in `display_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_refresh_interval_from_millihertz` | test | 61 |  | `TestDisplay_RefreshIntervalFromMillihertz` | todo |

## gpui_linux/src/linux/wayland/serial.rs

Go target: `platform/linux/wayland/serial.go` (package `wayland`), tests in `serial_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_selection_serial_ignores_unrelated_serial_kinds` | test | 121 |  | `TestSerial_SelectionSerialIgnoresUnrelatedSerialKinds` | todo |
| `test_selection_serial_uses_mouse_press_without_key_press` | test | 133 |  | `TestSerial_SelectionSerialUsesMousePressWithoutKeyPress` | todo |
| `test_selection_serial_uses_event_arrival_order_across_rollover` | test | 141 |  | `TestSerial_SelectionSerialUsesEventArrivalOrderAcrossRollover` | todo |
| `test_selection_serial_is_unavailable_without_eligible_input` | test | 150 |  | `TestSerial_SelectionSerialIsUnavailableWithoutEligibleInput` | todo |
| `test_zero_is_a_valid_selection_serial` | test | 163 |  | `TestSerial_ZeroIsAValidSelectionSerial` | todo |

## gpui_linux/src/linux/wayland/window.rs

Go target: `platform/linux/wayland/window.go` (package `wayland`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WaylandWindowState` | struct | 103 |  | `WaylandWindowState` | todo |
| `WaylandSurfaceState` | enum | 148 |  | `WaylandSurfaceState` | todo |
| `WaylandXdgSurfaceState` | struct | 307 |  | `WaylandXdgSurfaceState` | todo |
| `WaylandLayerSurfaceState` | struct | 314 |  | `WaylandLayerSurfaceState` | todo |
| `WaylandPopupSurfaceState` | struct | 319 |  | `WaylandPopupSurfaceState` | todo |
| `WaylandWindowStatePtr` | struct | 549 |  | `WaylandWindowStatePtr` | todo |
| `WaylandWindowState::is_transparent` | method | 651 |  | `(WaylandWindowState).IsTransparent` | todo |
| `WaylandWindowState::primary_output_scale` | method | 668 |  | `(WaylandWindowState).PrimaryOutputScale` | todo |
| `WaylandWindowState::inset` | method | 690 |  | `(WaylandWindowState).Inset` | todo |
| `ImeInput` | enum | 768 |  | `IMEInput` | todo |
| `WaylandWindowStatePtr::handle` | method | 901 |  | `(WaylandWindowStatePtr).Handle` | todo |
| `WaylandWindowStatePtr::surface` | method | 905 |  | `(WaylandWindowStatePtr).Surface` | todo |
| `WaylandWindowStatePtr::toplevel` | method | 909 |  | `(WaylandWindowStatePtr).Toplevel` | todo |
| `WaylandWindowStatePtr::xdg_surface` | method | 914 |  | `(WaylandWindowStatePtr).XdgSurface` | todo |
| `WaylandWindowStatePtr::layer_surface` | method | 919 |  | `(WaylandWindowStatePtr).LayerSurface` | todo |
| `WaylandWindowStatePtr::window_geometry` | method | 925 |  | `(WaylandWindowStatePtr).WindowGeometry` | todo |
| `WaylandWindowStatePtr::ptr_eq` | method | 934 |  | `(WaylandWindowStatePtr).PtrEq` | todo |
| `WaylandWindowStatePtr::add_child` | method | 938 |  | `(WaylandWindowStatePtr).AddChild` | todo |
| `WaylandWindowStatePtr::is_blocked` | method | 943 |  | `(WaylandWindowStatePtr).IsBlocked` | todo |
| `WaylandWindowStatePtr::frame` | method | 948 |  | `(WaylandWindowStatePtr).Frame` | todo |
| `WaylandWindowStatePtr::frame_callback_fired` | method | 1019 |  | `(WaylandWindowStatePtr).FrameCallbackFired` | todo |
| `WaylandWindowStatePtr::scheduled_frame_fired` | method | 1028 |  | `(WaylandWindowStatePtr).ScheduledFrameFired` | todo |
| `WaylandWindowStatePtr::retry_timer_fired` | method | 1037 |  | `(WaylandWindowStatePtr).RetryTimerFired` | todo |
| `WaylandWindowStatePtr::is_configured` | method | 1046 |  | `(WaylandWindowStatePtr).IsConfigured` | todo |
| `WaylandWindowStatePtr::schedule_frame` | method | 1050 |  | `(WaylandWindowStatePtr).ScheduleFrame` | todo |
| `WaylandWindowStatePtr::handle_xdg_surface_event` | method | 1098 |  | `(WaylandWindowStatePtr).HandleXdgSurfaceEvent` | todo |
| `WaylandWindowStatePtr::handle_toplevel_decoration_event` | method | 1198 |  | `(WaylandWindowStatePtr).HandleToplevelDecorationEvent` | todo |
| `WaylandWindowStatePtr::handle_fractional_scale_event` | method | 1232 |  | `(WaylandWindowStatePtr).HandleFractionalScaleEvent` | todo |
| `WaylandWindowStatePtr::handle_toplevel_event` | method | 1239 |  | `(WaylandWindowStatePtr).HandleToplevelEvent` | todo |
| `WaylandWindowStatePtr::handle_layersurface_event` | method | 1354 |  | `(WaylandWindowStatePtr).HandleLayersurfaceEvent` | todo |
| `WaylandWindowStatePtr::handle_popup_event` | method | 1392 |  | `(WaylandWindowStatePtr).HandlePopupEvent` | todo |
| `WaylandWindowStatePtr::handle_surface_event` | method | 1422 |  | `(WaylandWindowStatePtr).HandleSurfaceEvent` | todo |
| `WaylandWindowStatePtr::handle_ime` | method | 1493 |  | `(WaylandWindowStatePtr).HandleIME` | todo |
| `WaylandWindowStatePtr::get_ime_area` | method | 1520 |  | `(WaylandWindowStatePtr).GetIMEArea` | todo |
| `WaylandWindowStatePtr::set_size_and_scale` | method | 1531 |  | `(WaylandWindowStatePtr).SetSizeAndScale` | todo |
| `WaylandWindowStatePtr::resize` | method | 1577 |  | `(WaylandWindowStatePtr).Resize` | todo |
| `WaylandWindowStatePtr::rescale` | method | 1581 |  | `(WaylandWindowStatePtr).Rescale` | todo |
| `WaylandWindowStatePtr::close` | method | 1585 |  | `(WaylandWindowStatePtr).Close` | todo |
| `WaylandWindowStatePtr::handle_input` | method | 1610 |  | `(WaylandWindowStatePtr).HandleInput` | todo |
| `WaylandWindowStatePtr::set_focused` | method | 1635 |  | `(WaylandWindowStatePtr).SetFocused` | todo |
| `WaylandWindowStatePtr::set_hovered` | method | 1647 |  | `(WaylandWindowStatePtr).SetHovered` | todo |
| `WaylandWindowStatePtr::handle_output_changed` | method | 1665 |  | `(WaylandWindowStatePtr).HandleOutputChanged` | todo |
| `WaylandWindowStatePtr::set_appearance` | method | 1686 |  | `(WaylandWindowStatePtr).SetAppearance` | todo |
| `WaylandWindowStatePtr::set_button_layout` | method | 1696 |  | `(WaylandWindowStatePtr).SetButtonLayout` | todo |
| `WaylandWindowStatePtr::primary_output_scale` | method | 1704 |  | `(WaylandWindowStatePtr).PrimaryOutputScale` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `failure_tracks_whether_the_surface_has_presented` | test | 727 |  | `TestWindowPresentationStateTests_FailureTracksWhetherTheSurfaceHasPresented` | todo |
| `only_retry_states_require_presentation` | test | 747 |  | `TestWindowPresentationStateTests_OnlyRetryStatesRequirePresentation` | todo |
| `square_corners` | test | 2414 |  | `TestWindowRoundedRectTests_SquareCorners` | todo |
| `independent_corners` | test | 2420 |  | `TestWindowRoundedRectTests_IndependentCorners` | todo |
| `clamps_radii` | test | 2444 |  | `TestWindowRoundedRectTests_ClampsRadii` | todo |
| `empty_area` | test | 2457 |  | `TestWindowRoundedRectTests_EmptyArea` | todo |

## gpui_linux/src/linux/x11/client.rs

Go target: `platform/linux/x11/client.go` (package `x11`), tests in `client_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `EventHandlerError` | enum | 126 |  | `EventHandlerError` | todo |
| `Xdnd` | struct | 155 |  | `Xdnd` | todo |
| `X11ClientState` | struct | 182 |  | `X11ClientState` | todo |
| `X11ClientStatePtr` | struct | 245 |  | `X11ClientStatePtr` | todo |
| `X11ClientStatePtr::get_client` | method | 248 |  | `(X11ClientStatePtr).GetClient` | todo |
| `X11ClientStatePtr::drop_window` | method | 252 |  | `(X11ClientStatePtr).DropWindow` | todo |
| `X11ClientStatePtr::update_ime_position` | method | 277 |  | `(X11ClientStatePtr).UpdateIMEPosition` | todo |
| `mode_refresh_rate` | fn | 2228 |  | `ModeRefreshRate` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `key_event_state_uses_event_modifiers_without_mutating_server_state` | test | 2944 |  | `TestClient_KeyEventStateUsesEventModifiersWithoutMutatingServerState` | todo |
| `key_event_state_ignores_pointer_button_bits` | test | 2968 |  | `TestClient_KeyEventStateIgnoresPointerButtonBits` | todo |
| `key_event_state_preserves_non_locked_layout_components` | test | 2997 |  | `TestClient_KeyEventStatePreservesNonLockedLayoutComponents` | todo |
| `capslock_toggle_produces_uppercase` | test | 3023 |  | `TestClient_CapslockToggleProducesUppercase` | todo |
| `neo2_level3_via_capslock_produces_ellipsis` | test | 3052 |  | `TestClient_Neo2Level3ViaCapslockProducesEllipsis` | todo |
| `neo2_latched_mod5_preserved` | test | 3073 |  | `TestClient_Neo2LatchedMod5Preserved` | todo |
| `german_layout_correct_key_resolution` | test | 3101 |  | `TestClient_GermanLayoutCorrectKeyResolution` | todo |
| `space_works_with_cyrillic_layout_active` | test | 3119 |  | `TestClient_SpaceWorksWithCyrillicLayoutActive` | todo |
| `macro_shift_bracket_produces_brace` | test | 3142 |  | `TestClient_MacroShiftBracketProducesBrace` | todo |
| `sequential_key_events_do_not_corrupt_state` | test | 3164 |  | `TestClient_SequentialKeyEventsDoNotCorruptState` | todo |
| `space_works_with_czech_layout_active` | test | 3203 |  | `TestClient_SpaceWorksWithCzechLayoutActive` | todo |

## gpui_linux/src/linux/x11/clipboard.rs

Go target: `platform/linux/x11/clipboard.go` (package `x11`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ClipboardKind` | enum | 1182 |  | `ClipboardKind` | todo |
| `Error` | enum | 1218 |  | `Error` | todo |

## gpui_linux/src/linux/x11/event.rs

Go target: `platform/linux/x11/event.go` (package `x11`), tests in `event_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_get_valuator_axis_index` | test | 130 |  | `TestEvent_GetValuatorAxisIndex` | todo |

## gpui_linux/src/linux/x11/window.rs

Go target: `platform/linux/x11/window.go` (package `x11`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Callbacks` | struct | 245 |  | `Callbacks` | todo |
| `X11WindowState` | struct | 259 |  | `X11WindowState` | todo |
| `X11WindowState::new` | assoc-fn | 464 |  | `NewX11WindowState` | todo |

## gpui_linux/src/linux/x11/xim_handler.rs

Go target: `platform/linux/x11/xim_handler.go` (package `x11`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `XimCallbackEvent` | enum | 6 |  | `XimCallbackEvent` | todo |
| `XimHandler` | struct | 12 |  | `XimHandler` | todo |
| `XimHandler::new` | assoc-fn | 21 |  | `NewXimHandler` | todo |

