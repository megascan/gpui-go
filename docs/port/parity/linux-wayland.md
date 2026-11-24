# Parity: linux-wayland

Scope: gpui_linux, Wayland back end (linux/wayland.rs, linux/wayland/*). Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_linux/src/linux/wayland/client.rs

Go target: `platform/linux/wayland/client.go` (lines 1-1967) and `client_input.go` (1968-3454) (package `wayland`), tests in `client_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Globals` | struct | 243 |  | `Globals` | done |
| `InProgressOutput` | struct | 325 |  | `InProgressOutput` | done |
| `Output` | struct | 352 |  | `Output` | done |
| `ClickState` | struct | 539 |  | `ClickState` | done |
| `WaylandClientStatePtr` | struct | 577 |  | `WaylandClientStatePtr` | done |
| `WaylandClientStatePtr::get_client` | method | 580 |  | `(WaylandClientStatePtr).GetClient` | done |
| `WaylandClientStatePtr::try_get_client` | method | 603 |  | `(WaylandClientStatePtr).TryGetClient` | done |
| `WaylandClientStatePtr::sole_output` | method | 608 |  | `(WaylandClientStatePtr).SoleOutput` | done |
| `WaylandClientStatePtr::dispatch_scheduled_frames` | method | 619 |  | `(WaylandClientStatePtr).DispatchScheduledFrames` | done |
| `WaylandClientStatePtr::schedule_frame_retry` | method | 639 |  | `(WaylandClientStatePtr).ScheduleFrameRetry` | done |
| `WaylandClientStatePtr::get_serial` | method | 661 |  | `(WaylandClientStatePtr).GetSerial` | done |
| `WaylandClientStatePtr::activation_context` | method | 665 |  | `(WaylandClientStatePtr).ActivationContext` | done |
| `WaylandClientStatePtr::start_external_drag` | method | 673 |  | `(WaylandClientStatePtr).StartExternalDrag` | done |
| `WaylandClientStatePtr::enable_ime` | method | 713 |  | `(WaylandClientStatePtr).EnableIME` | done |
| `WaylandClientStatePtr::disable_ime` | method | 739 |  | `(WaylandClientStatePtr).DisableIME` | done |
| `WaylandClientStatePtr::ime_enabled` | method | 750 |  | `(WaylandClientStatePtr).IMEEnabled` | done |
| `WaylandClientStatePtr::update_ime_position` | method | 755 |  | `(WaylandClientStatePtr).UpdateIMEPosition` | done |
| `WaylandClientStatePtr::handle_keyboard_layout_change` | method | 767 |  | `(WaylandClientStatePtr).HandleKeyboardLayoutChange` | done |
| `WaylandClientStatePtr::drop_window` | method | 805 |  | `(WaylandClientStatePtr).DropWindow` | done |
| `WaylandConnection` | struct | 884 |  | `WaylandConnection` | done |
| `WaylandConnection::window_identifier` | method | 1532 |  | `(WaylandConnection).WindowIdentifier` | dev:D-WLFOREIGN |
| `file_uri_list` | fn | 527 |  | `fileURIList` | done |
| `linux_button_to_gpui` | fn | 2361 |  | `linuxButtonToGPUI` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `rejects_required_globals_below_supported_versions` | test | 3207 |  | `TestClient_RejectsRequiredGlobalsBelowSupportedVersions` | done |
| `leave_discards_pending_drag_uri_read` | test | 3275 |  | `TestClient_LeaveDiscardsPendingDragURIRead` | done |
| `drop_then_leave_discards_pending_drag_uri_read` | test | 3294 |  | `TestClient_DropThenLeaveDiscardsPendingDragURIRead` | done |
| `stale_completion_preserves_newer_drag_destination` | test | 3316 |  | `TestClient_StaleCompletionPreservesNewerDragDestination` | done |
| `builds_terminated_uri_list_for_each_dragged_path` | test | 3360 |  | `TestClient_BuildsTerminatedURIListForEachDraggedPath` | done |
| `skips_paths_that_have_no_file_url` | test | 3373 |  | `TestClient_SkipsPathsThatHaveNoFileURL` | done |
| `builds_empty_uri_list_without_convertible_paths` | test | 3383 |  | `TestClient_BuildsEmptyURIListWithoutConvertiblePaths` | done |
| `caches_cursor_rectangle_committed_after_done` | test | 3395 |  | `TestClient_CachesCursorRectangleCommittedAfterDone` | done |
| `skips_unchanged_cursor_rectangle_after_done` | test | 3415 |  | `TestClient_SkipsUnchangedCursorRectangleAfterDone` | done |
| `skips_cursor_rectangles_with_unchanged_protocol_coordinates` | test | 3433 |  | `TestClient_SkipsCursorRectanglesWithUnchangedProtocolCoordinates` | done |

## gpui_linux/src/linux/wayland/cursor.rs

Go target: `platform/linux/wayland/cursor.go` (package `wayland`), tests in `cursor_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Cursor` | struct | 11 |  | `Cursor` | done |
| `LoadedTheme` | struct | 20 |  | `LoadedTheme` | done |
| `Cursor::drop` | method | 27 |  | `(Cursor).Destroy` | done |
| `Cursor::new` | method | 34 |  | `NewCursor` | done |
| `Cursor::set_theme` | method | 76 |  | `(Cursor).SetTheme` | done |
| `Cursor::set_size` | method | 89 |  | `(Cursor).SetSize` | done |
| `Cursor::set_icon` | method | 94 |  | `(Cursor).SetIcon` | done |

## gpui_linux/src/linux/wayland.rs

Go target: `platform/linux/wayland/wayland.go` (package `wayland`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `to_shape` | fn | 18 |  | `toShape` | done |

## gpui_linux/src/linux/wayland/display.rs

Go target: `platform/linux/wayland/display.go` (package `wayland`), tests in `display_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_refresh_interval_from_millihertz` | test | 61 |  | `TestDisplay_RefreshIntervalFromMillihertz` | done |

## gpui_linux/src/linux/wayland/serial.rs

Go target: `platform/linux/wayland/serial.go` (package `wayland`), tests in `serial_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_selection_serial_ignores_unrelated_serial_kinds` | test | 121 |  | `TestSerial_SelectionSerialIgnoresUnrelatedSerialKinds` | done |
| `test_selection_serial_uses_mouse_press_without_key_press` | test | 133 |  | `TestSerial_SelectionSerialUsesMousePressWithoutKeyPress` | done |
| `test_selection_serial_uses_event_arrival_order_across_rollover` | test | 141 |  | `TestSerial_SelectionSerialUsesEventArrivalOrderAcrossRollover` | done |
| `test_selection_serial_is_unavailable_without_eligible_input` | test | 150 |  | `TestSerial_SelectionSerialIsUnavailableWithoutEligibleInput` | done |
| `test_zero_is_a_valid_selection_serial` | test | 163 |  | `TestSerial_ZeroIsAValidSelectionSerial` | done |

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

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-WLFOREIGN | `WaylandConnection.WindowIdentifier` exports the focused surface through xdg-foreign (zxdg_exporter_v2, else v1) synchronously on the main goroutine with a private event queue, as ashpd does on a helper thread, and keeps one export per surface until the window drops or the connection closes; later calls for the same surface return the same `wayland:<handle>`. Upstream creates a new export per call and destroys it when the returned identifier drops | `gpui_linux/src/linux/wayland/client.rs:1532-1549`, ashpd-0.13.13 `window_identifier/wayland.rs` | `common.LinuxClient.WindowIdentifier` returns a plain string the caller cannot release; xdg-foreign is hand-defined in `xdg_foreign.go` because wlproto has no bindings for it |
