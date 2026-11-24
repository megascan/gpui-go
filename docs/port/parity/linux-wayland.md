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

## gpui_linux/src/linux/wayland/clipboard.rs

Go target: `platform/linux/wayland/clipboard.go` (package `wayland`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Clipboard` | struct | 23 |  | `Clipboard` | done |
| `ReceiveData` | trait | 39 |  | `ReceiveData` | done |
| `DataOffer` | struct | 57 |  | `DataOffer` | done |
| `Clipboard::send_bytes` | method | 235 |  | `(Clipboard).SendBytes` | done |

## gpui_linux/src/linux/wayland/layer_shell.rs

Go target: `platform/linux/wayland/layer_shell.go` (package `wayland`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `wayland_layer` | fn | 5 |  | `waylandLayer` | done |
| `wayland_anchor` | fn | 14 |  | `layerShellWaylandAnchor` | dev:D-SUBMOD |
| `wayland_keyboard_interactivity` | fn | 18 |  | `waylandKeyboardInteractivity` | done |

## gpui_linux/src/linux/wayland/popup.rs

Go target: `platform/linux/wayland/popup.go` (package `wayland`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `wayland_anchor` | fn | 5 |  | `popupWaylandAnchor` | dev:D-SUBMOD |
| `wayland_gravity` | fn | 19 |  | `waylandGravity` | done |
| `wayland_constraint_adjustment` | fn | 33 |  | `waylandConstraintAdjustment` | done |

## gpui_linux/src/linux/wayland/window.rs

Go target: `platform/linux/wayland/window.go` (package `wayland`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WaylandWindowState` | struct | 103 |  | `WaylandWindowState` | done |
| `WaylandSurfaceState` | enum | 148 |  | `WaylandSurfaceState` | done |
| `WaylandXdgSurfaceState` | struct | 307 |  | `WaylandXdgSurfaceState` | done |
| `WaylandLayerSurfaceState` | struct | 314 |  | `WaylandLayerSurfaceState` | done |
| `WaylandPopupSurfaceState` | struct | 319 |  | `WaylandPopupSurfaceState` | done |
| `WaylandWindowStatePtr` | struct | 549 |  | `WaylandWindowStatePtr` | done |
| `WaylandWindowState::is_transparent` | method | 651 |  | `(WaylandWindowState).IsTransparent` | done |
| `WaylandWindowState::primary_output_scale` | method | 668 |  | `(WaylandWindowState).PrimaryOutputScale` | done |
| `WaylandWindowState::inset` | method | 690 |  | `(WaylandWindowState).Inset` | done |
| `ImeInput` | enum | 768 |  | `IMEInput` | done |
| `WaylandWindowStatePtr::handle` | method | 901 |  | `(WaylandWindowStatePtr).Handle` | done |
| `WaylandWindowStatePtr::surface` | method | 905 |  | `(WaylandWindowStatePtr).Surface` | done |
| `WaylandWindowStatePtr::toplevel` | method | 909 |  | `(WaylandWindowStatePtr).Toplevel` | done |
| `WaylandWindowStatePtr::xdg_surface` | method | 914 |  | `(WaylandWindowStatePtr).XdgSurface` | done |
| `WaylandWindowStatePtr::layer_surface` | method | 919 |  | `(WaylandWindowStatePtr).LayerSurface` | done |
| `WaylandWindowStatePtr::window_geometry` | method | 925 |  | `(WaylandWindowStatePtr).WindowGeometry` | done |
| `WaylandWindowStatePtr::ptr_eq` | method | 934 |  | `(WaylandWindowStatePtr).PtrEq` | done |
| `WaylandWindowStatePtr::add_child` | method | 938 |  | `(WaylandWindowStatePtr).AddChild` | done |
| `WaylandWindowStatePtr::is_blocked` | method | 943 |  | `(WaylandWindowStatePtr).IsBlocked` | done |
| `WaylandWindowStatePtr::frame` | method | 948 |  | `(WaylandWindowStatePtr).Frame` | done |
| `WaylandWindowStatePtr::frame_callback_fired` | method | 1019 |  | `(WaylandWindowStatePtr).FrameCallbackFired` | done |
| `WaylandWindowStatePtr::scheduled_frame_fired` | method | 1028 |  | `(WaylandWindowStatePtr).ScheduledFrameFired` | done |
| `WaylandWindowStatePtr::retry_timer_fired` | method | 1037 |  | `(WaylandWindowStatePtr).RetryTimerFired` | done |
| `WaylandWindowStatePtr::is_configured` | method | 1046 |  | `(WaylandWindowStatePtr).IsConfigured` | done |
| `WaylandWindowStatePtr::schedule_frame` | method | 1050 |  | `(WaylandWindowStatePtr).ScheduleFrame` | done |
| `WaylandWindowStatePtr::handle_xdg_surface_event` | method | 1098 |  | `(WaylandWindowStatePtr).HandleXdgSurfaceEvent` | done |
| `WaylandWindowStatePtr::handle_toplevel_decoration_event` | method | 1198 |  | `(WaylandWindowStatePtr).HandleToplevelDecorationEvent` | done |
| `WaylandWindowStatePtr::handle_fractional_scale_event` | method | 1232 |  | `(WaylandWindowStatePtr).HandleFractionalScaleEvent` | done |
| `WaylandWindowStatePtr::handle_toplevel_event` | method | 1239 |  | `(WaylandWindowStatePtr).HandleToplevelEvent` | done |
| `WaylandWindowStatePtr::handle_layersurface_event` | method | 1354 |  | `(WaylandWindowStatePtr).HandleLayersurfaceEvent` | done |
| `WaylandWindowStatePtr::handle_popup_event` | method | 1392 |  | `(WaylandWindowStatePtr).HandlePopupEvent` | done |
| `WaylandWindowStatePtr::handle_surface_event` | method | 1422 |  | `(WaylandWindowStatePtr).HandleSurfaceEvent` | done |
| `WaylandWindowStatePtr::handle_ime` | method | 1493 |  | `(WaylandWindowStatePtr).HandleIME` | done |
| `WaylandWindowStatePtr::get_ime_area` | method | 1520 |  | `(WaylandWindowStatePtr).GetIMEArea` | done |
| `WaylandWindowStatePtr::set_size_and_scale` | method | 1531 |  | `(WaylandWindowStatePtr).SetSizeAndScale` | done |
| `WaylandWindowStatePtr::resize` | method | 1577 |  | `(WaylandWindowStatePtr).Resize` | done |
| `WaylandWindowStatePtr::rescale` | method | 1581 |  | `(WaylandWindowStatePtr).Rescale` | done |
| `WaylandWindowStatePtr::close` | method | 1585 |  | `(WaylandWindowStatePtr).Close` | done |
| `WaylandWindowStatePtr::handle_input` | method | 1610 |  | `(WaylandWindowStatePtr).HandleInput` | done |
| `WaylandWindowStatePtr::set_focused` | method | 1635 |  | `(WaylandWindowStatePtr).SetFocused` | done |
| `WaylandWindowStatePtr::set_hovered` | method | 1647 |  | `(WaylandWindowStatePtr).SetHovered` | done |
| `WaylandWindowStatePtr::handle_output_changed` | method | 1665 |  | `(WaylandWindowStatePtr).HandleOutputChanged` | done |
| `WaylandWindowStatePtr::set_appearance` | method | 1686 |  | `(WaylandWindowStatePtr).SetAppearance` | done |
| `WaylandWindowStatePtr::set_button_layout` | method | 1696 |  | `(WaylandWindowStatePtr).SetButtonLayout` | done |
| `WaylandWindowStatePtr::primary_output_scale` | method | 1704 |  | `(WaylandWindowStatePtr).PrimaryOutputScale` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `failure_tracks_whether_the_surface_has_presented` | test | 727 |  | `TestWindowPresentationStateTests_FailureTracksWhetherTheSurfaceHasPresented` | done |
| `only_retry_states_require_presentation` | test | 747 |  | `TestWindowPresentationStateTests_OnlyRetryStatesRequirePresentation` | done |
| `square_corners` | test | 2414 |  | `TestWindowRoundedRectTests_SquareCorners` | done |
| `independent_corners` | test | 2420 |  | `TestWindowRoundedRectTests_IndependentCorners` | done |
| `clamps_radii` | test | 2444 |  | `TestWindowRoundedRectTests_ClampsRadii` | done |
| `empty_area` | test | 2457 |  | `TestWindowRoundedRectTests_EmptyArea` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-WLFOREIGN | `WaylandConnection.WindowIdentifier` exports the focused surface through xdg-foreign (zxdg_exporter_v2, else v1) synchronously on the main goroutine with a private event queue, as ashpd does on a helper thread, and keeps one export per surface until the window drops or the connection closes; later calls for the same surface return the same `wayland:<handle>`. Upstream creates a new export per call and destroys it when the returned identifier drops | `gpui_linux/src/linux/wayland/client.rs:1532-1549`, ashpd-0.13.13 `window_identifier/wayland.rs` | `common.LinuxClient.WindowIdentifier` returns a plain string the caller cannot release; xdg-foreign is hand-defined in `xdg_foreign.go` because wlproto has no bindings for it |
| D-SEC-KEYMAPFD | The `wl_keyboard.keymap` handler fstat()s the fd and refuses a keymap whose file is smaller than the declared size (mapping past EOF raises SIGBUS); a keymap libxkbcommon rejects is logged and ignored instead of panicking (SEC-024) | `gpui_linux/src/linux/wayland/client.rs` (keymap event), `xkbcommon-rs` `Keymap::new_from_fd` | `internal/linux/xkb/xkb.go` `NewKeymapFromFd`, `client_input.go` |
