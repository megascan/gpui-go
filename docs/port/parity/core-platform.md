# Parity: core-platform

Scope: gpui: platform interfaces, TestPlatform, dispatchers. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/platform.rs

Go target: `platform.go` (package `gpui`), tests in `platform_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WindowVisibility` | enum | 107 |  | `WindowVisibility` | todo |
| `WindowVisibility::is_visible` | method | 118 |  | `(WindowVisibility).IsVisible` | todo |
| `DisplayEvent` | enum | 127 |  | `DisplayEvent` | todo |
| `refresh_interval_from_hz` | fn | 155 |  | `RefreshIntervalFromHz` | todo |
| `ActivationPolicy` | enum | 166 |  | `ActivationPolicy` | todo |
| `ActivityGuard` | struct | 187 |  | `ActivityGuard` | todo |
| `ActivityGuard::new` | assoc-fn | 193 |  | `NewActivityGuard` | todo |
| `ActivityGuard::noop` | assoc-fn | 200 |  | `ActivityGuardNoop` | todo |
| `guess_compositor` | fn | 210 | linux,freebsd | `GuessCompositor` | todo |
| `GraphicalEnvironment` | struct | 227 | linux,freebsd | `GraphicalEnvironment` | todo |
| `GraphicalEnvironment` | struct | 254 | windows | `GraphicalEnvironment` | todo |
| `GraphicalEnvironment::detect` | assoc-fn | 263 | windows | `GraphicalEnvironmentDetect` | todo |
| `GraphicalEnvironment::apply_to` | method | 279 | windows | `(GraphicalEnvironment).ApplyTo` | todo |
| `GraphicalEnvironment::desktop_environment` | method | 282 | windows | `(GraphicalEnvironment).DesktopEnvironment` | todo |
| `GraphicalEnvironment` | struct | 290 | linux,freebsd,windows | `GraphicalEnvironment` | todo |
| `GraphicalEnvironment::detect` | assoc-fn | 295 | linux,freebsd,windows | `GraphicalEnvironmentDetect` | todo |
| `GraphicalEnvironment::apply_to` | method | 300 | linux,freebsd,windows | `(GraphicalEnvironment).ApplyTo` | todo |
| `GraphicalEnvironment::desktop_environment` | method | 303 | linux,freebsd,windows | `(GraphicalEnvironment).DesktopEnvironment` | todo |
| `WindowingRequest` | enum | 310 |  | `WindowingRequest` | todo |
| `GraphicalEnvironment::detect` | assoc-fn | 323 | linux,freebsd | `GraphicalEnvironmentDetect` | todo |
| `GraphicalEnvironment::apply_to` | method | 336 | linux,freebsd | `(GraphicalEnvironment).ApplyTo` | todo |
| `GraphicalEnvironment::desktop_environment` | method | 353 | linux,freebsd | `(GraphicalEnvironment).DesktopEnvironment` | todo |
| `GraphicalEnvironment::guess_compositor` | method | 364 | linux,freebsd | `(GraphicalEnvironment).GuessCompositor` | todo |
| `Platform` | trait | 394 |  | `Platform` | todo |
| `Platform::platform_info` | trait-method | 396 |  | `Platform.PlatformInfo` | todo |
| `Platform::background_executor` | trait-method | 398 |  | `Platform.BackgroundExecutor` | todo |
| `Platform::foreground_executor` | trait-method | 399 |  | `Platform.ForegroundExecutor` | todo |
| `Platform::text_system` | trait-method | 400 |  | `Platform.TextSystem` | todo |
| `Platform::run` | trait-method | 402 |  | `Platform.Run` | todo |
| `Platform::quit` | trait-method | 403 |  | `Platform.Quit` | todo |
| `Platform::set_initial_windowing` | trait-method | 408 |  | `Platform.SetInitialWindowing` | todo |
| `Platform::request_windowing` | trait-method | 409 |  | `Platform.RequestWindowing` | todo |
| `Platform::restart` | trait-method | 414 |  | `Platform.Restart` | todo |
| `Platform::activate` | trait-method | 415 |  | `Platform.Activate` | todo |
| `Platform::set_activation_policy` | trait-method | 417 |  | `Platform.SetActivationPolicy` | todo |
| `Platform::hide` | trait-method | 418 |  | `Platform.Hide` | todo |
| `Platform::hide_other_apps` | trait-method | 419 |  | `Platform.HideOtherApps` | todo |
| `Platform::unhide_other_apps` | trait-method | 420 |  | `Platform.UnhideOtherApps` | todo |
| `Platform::displays` | trait-method | 422 |  | `Platform.Displays` | todo |
| `Platform::primary_display` | trait-method | 423 |  | `Platform.PrimaryDisplay` | todo |
| `Platform::on_displays_changed` | trait-method | 428 |  | `Platform.OnDisplaysChanged` | todo |
| `Platform::active_window` | trait-method | 429 |  | `Platform.ActiveWindow` | todo |
| `Platform::window_stack` | trait-method | 430 |  | `Platform.WindowStack` | todo |
| `Platform::is_screen_capture_supported` | trait-method | 434 |  | `Platform.IsScreenCaptureSupported` | todo |
| `Platform::screen_capture_sources` | trait-method | 438 |  | `Platform.ScreenCaptureSources` | todo |
| `Platform::open_window` | trait-method | 450 |  | `Platform.OpenWindow` | todo |
| `Platform::window_appearance` | trait-method | 457 |  | `Platform.WindowAppearance` | todo |
| `Platform::set_window_appearance` | trait-method | 467 |  | `Platform.SetWindowAppearance` | todo |
| `Platform::button_layout` | trait-method | 470 |  | `Platform.ButtonLayout` | todo |
| `Platform::text_scale_factor` | trait-method | 475 |  | `Platform.TextScaleFactor` | todo |
| `Platform::on_text_scale_factor_change` | trait-method | 479 |  | `Platform.OnTextScaleFactorChange` | todo |
| `Platform::open_url` | trait-method | 481 |  | `Platform.OpenURL` | todo |
| `Platform::on_open_urls` | trait-method | 482 |  | `Platform.OnOpenURLs` | todo |
| `Platform::register_url_scheme` | trait-method | 483 |  | `Platform.RegisterURLScheme` | todo |
| `Platform::prompt_for_paths` | trait-method | 485 |  | `Platform.PromptForPaths` | todo |
| `Platform::prompt_for_new_path` | trait-method | 489 |  | `Platform.PromptForNewPath` | todo |
| `Platform::can_select_mixed_files_and_dirs` | trait-method | 494 |  | `Platform.CanSelectMixedFilesAndDirs` | todo |
| `Platform::reveal_path` | trait-method | 495 |  | `Platform.RevealPath` | todo |
| `Platform::open_with_system` | trait-method | 496 |  | `Platform.OpenWithSystem` | todo |
| `Platform::on_quit` | trait-method | 498 |  | `Platform.OnQuit` | todo |
| `Platform::on_reopen` | trait-method | 499 |  | `Platform.OnReopen` | todo |
| `Platform::on_system_sleep` | trait-method | 501 |  | `Platform.OnSystemSleep` | todo |
| `Platform::on_system_wake` | trait-method | 503 |  | `Platform.OnSystemWake` | todo |
| `Platform::on_app_lifecycle` | trait-method | 514 |  | `Platform.OnAppLifecycle` | todo |
| `Platform::on_memory_warning` | trait-method | 520 |  | `Platform.OnMemoryWarning` | todo |
| `Platform::gestures` | trait-method | 525 |  | `Platform.Gestures` | todo |
| `Platform::set_menus` | trait-method | 529 |  | `Platform.SetMenus` | todo |
| `Platform::get_menus` | trait-method | 530 |  | `Platform.GetMenus` | todo |
| `Platform::set_dock_menu` | trait-method | 534 |  | `Platform.SetDockMenu` | todo |
| `Platform::perform_dock_menu_action` | trait-method | 535 |  | `Platform.PerformDockMenuAction` | todo |
| `Platform::add_recent_document` | trait-method | 536 |  | `Platform.AddRecentDocument` | todo |
| `Platform::update_jump_list` | trait-method | 537 |  | `Platform.UpdateJumpList` | todo |
| `Platform::on_app_menu_action` | trait-method | 544 |  | `Platform.OnAppMenuAction` | todo |
| `Platform::on_will_open_app_menu` | trait-method | 545 |  | `Platform.OnWillOpenAppMenu` | todo |
| `Platform::on_validate_app_menu_command` | trait-method | 546 |  | `Platform.OnValidateAppMenuCommand` | todo |
| `Platform::debug_state` | trait-method | 549 |  | `Platform.DebugState` | todo |
| `Platform::thermal_state` | trait-method | 553 |  | `Platform.ThermalState` | todo |
| `Platform::on_thermal_state_change` | trait-method | 554 |  | `Platform.OnThermalStateChange` | todo |
| `Platform::prevent_idle_sleep` | trait-method | 555 |  | `Platform.PreventIdleSleep` | todo |
| `Platform::power_mode` | trait-method | 559 |  | `Platform.PowerMode` | todo |
| `Platform::on_power_mode_change` | trait-method | 563 |  | `Platform.OnPowerModeChange` | todo |
| `Platform::set_app_identity` | trait-method | 571 |  | `Platform.SetAppIdentity` | todo |
| `Platform::show_system_notification` | trait-method | 581 |  | `Platform.ShowSystemNotification` | todo |
| `Platform::dismiss_system_notification` | trait-method | 592 |  | `Platform.DismissSystemNotification` | todo |
| `Platform::on_system_notification_response` | trait-method | 602 |  | `Platform.OnSystemNotificationResponse` | todo |
| `Platform::compositor_name` | trait-method | 609 |  | `Platform.CompositorName` | todo |
| `Platform::graphical_environment` | trait-method | 614 |  | `Platform.GraphicalEnvironment` | todo |
| `Platform::app_path` | trait-method | 617 |  | `Platform.AppPath` | todo |
| `Platform::path_for_auxiliary_executable` | trait-method | 618 |  | `Platform.PathForAuxiliaryExecutable` | todo |
| `Platform::set_cursor_style` | trait-method | 620 |  | `Platform.SetCursorStyle` | todo |
| `Platform::hide_cursor_until_mouse_moves` | trait-method | 624 |  | `Platform.HideCursorUntilMouseMoves` | todo |
| `Platform::is_cursor_visible` | trait-method | 627 |  | `Platform.IsCursorVisible` | todo |
| `Platform::should_auto_hide_scrollbars` | trait-method | 629 |  | `Platform.ShouldAutoHideScrollbars` | todo |
| `Platform::read_from_clipboard` | trait-method | 631 |  | `Platform.ReadFromClipboard` | todo |
| `Platform::write_to_clipboard` | trait-method | 632 |  | `Platform.WriteToClipboard` | todo |
| `Platform::read_from_clipboard_async` | trait-method | 641 |  | `Platform.ReadFromClipboardAsync` | todo |
| `Platform::read_from_primary` | trait-method | 646 | linux,freebsd | `Platform.ReadFromPrimary` | todo |
| `Platform::write_to_primary` | trait-method | 648 | linux,freebsd | `Platform.WriteToPrimary` | todo |
| `Platform::read_from_find_pasteboard` | trait-method | 651 | macos | `Platform.ReadFromFindPasteboard` | todo |
| `Platform::write_to_find_pasteboard` | trait-method | 653 | macos | `Platform.WriteToFindPasteboard` | todo |
| `Platform::write_credentials` | trait-method | 655 |  | `Platform.WriteCredentials` | todo |
| `Platform::write_credentials_if_not_exists` | trait-method | 661 | macos | `Platform.WriteCredentialsIfNotExists` | todo |
| `Platform::read_credentials` | trait-method | 672 |  | `Platform.ReadCredentials` | todo |
| `Platform::delete_credentials` | trait-method | 673 |  | `Platform.DeleteCredentials` | todo |
| `Platform::keyboard_layout` | trait-method | 675 |  | `Platform.KeyboardLayout` | todo |
| `Platform::keyboard_mapper` | trait-method | 676 |  | `Platform.KeyboardMapper` | todo |
| `Platform::on_keyboard_layout_change` | trait-method | 677 |  | `Platform.OnKeyboardLayoutChange` | todo |
| `PlatformDisplay` | trait | 681 |  | `PlatformDisplay` | todo |
| `PlatformDisplay::id` | trait-method | 683 |  | `PlatformDisplay.ID` | todo |
| `PlatformDisplay::uuid` | trait-method | 687 |  | `PlatformDisplay.UUID` | todo |
| `PlatformDisplay::bounds` | trait-method | 690 |  | `PlatformDisplay.Bounds` | todo |
| `PlatformDisplay::refresh_interval` | trait-method | 698 |  | `PlatformDisplay.RefreshInterval` | todo |
| `PlatformDisplay::visible_bounds` | trait-method | 703 |  | `PlatformDisplay.VisibleBounds` | todo |
| `PlatformDisplay::default_bounds` | trait-method | 708 |  | `PlatformDisplay.DefaultBounds` | todo |
| `SystemNotification` | struct | 722 |  | `SystemNotification` | todo |
| `SystemNotificationAction` | struct | 738 |  | `SystemNotificationAction` | todo |
| `SystemNotificationResponse` | struct | 748 |  | `SystemNotificationResponse` | todo |
| `PowerMode` | enum | 758 |  | `PowerMode` | todo |
| `ThermalState` | enum | 772 |  | `ThermalState` | todo |
| `SourceMetadata` | struct | 785 |  | `SourceMetadata` | todo |
| `ScreenCaptureSource` | trait | 797 |  | `ScreenCaptureSource` | todo |
| `ScreenCaptureSource::metadata` | trait-method | 799 |  | `ScreenCaptureSource.Metadata` | todo |
| `ScreenCaptureSource::stream` | trait-method | 803 |  | `ScreenCaptureSource.Stream` | todo |
| `ScreenCaptureStream` | trait | 811 |  | `ScreenCaptureStream` | todo |
| `ScreenCaptureStream::metadata` | trait-method | 813 |  | `ScreenCaptureStream.Metadata` | todo |
| `ScreenCaptureFrame` | struct | 817 |  | `ScreenCaptureFrame` | todo |
| `DisplayId` | struct | 821 |  | `DisplayID` | todo |
| `DisplayId::new` | assoc-fn | 825 |  | `NewDisplayID` | todo |
| `ResizeEdge` | enum | 850 |  | `ResizeEdge` | todo |
| `WindowDecorations` | enum | 871 |  | `WindowDecorations` | todo |
| `Decorations` | enum | 881 |  | `Decorations` | todo |
| `WindowControls` | struct | 894 |  | `WindowControls` | todo |
| `WindowButton` | enum | 919 |  | `WindowButton` | todo |
| `WindowButton::id` | method | 930 |  | `(WindowButton).ID` | todo |
| `MAX_BUTTONS_PER_SIDE` | const | 949 |  | `MaxButtonsPerSide` | todo |
| `WindowButtonLayout` | struct | 956 |  | `WindowButtonLayout` | todo |
| `WindowButtonLayout::linux_default` | assoc-fn | 966 | linux,freebsd | `WindowButtonLayoutLinuxDefault` | todo |
| `WindowButtonLayout::parse` | assoc-fn | 978 | linux,freebsd | `WindowButtonLayoutParse` | todo |
| `Tiling` | struct | 1058 |  | `Tiling` | todo |
| `Tiling::tiled` | assoc-fn | 1071 |  | `TilingTiled` | todo |
| `Tiling::is_tiled` | method | 1081 |  | `(Tiling).IsTiled` | todo |
| `A11yCallbacks` | struct | 1087 |  | `A11yCallbacks` | todo |
| `FrameRequestSource` | enum | 1098 |  | `FrameRequestSource` | todo |
| `RequestFrameOptions` | struct | 1108 |  | `RequestFrameOptions` | todo |
| `PlatformFrameSignal` | struct | 1131 |  | `PlatformFrameSignal` | todo |
| `PlatformFrameSignal::new` | assoc-fn | 1146 |  | `NewPlatformFrameSignal` | todo |
| `PlatformFrameSignal::capture` | assoc-fn | 1159 |  | `PlatformFrameSignalCapture` | todo |
| `PlatformFrameSignal::record` | method | 1170 |  | `(PlatformFrameSignal).Record` | todo |
| `PlatformFrameSignal::take` | method | 1190 |  | `(PlatformFrameSignal).Take` | todo |
| `AppLifecyclePhase` | enum | 1224 |  | `AppLifecyclePhase` | todo |
| `WindowInsets` | struct | 1247 |  | `WindowInsets` | todo |
| `WindowInsets::effective` | method | 1266 |  | `(WindowInsets).Effective` | todo |
| `TextInputStateChange` | enum | 1278 |  | `TextInputStateChange` | todo |
| `PlatformWindow` | trait | 1290 |  | `PlatformWindow` | todo |
| `PlatformWindow::bounds` | trait-method | 1291 |  | `PlatformWindow.Bounds` | todo |
| `PlatformWindow::is_maximized` | trait-method | 1292 |  | `PlatformWindow.IsMaximized` | todo |
| `PlatformWindow::window_bounds` | trait-method | 1293 |  | `PlatformWindow.WindowBounds` | todo |
| `PlatformWindow::content_size` | trait-method | 1294 |  | `PlatformWindow.ContentSize` | todo |
| `PlatformWindow::visual_viewport_bounds` | trait-method | 1300 |  | `PlatformWindow.VisualViewportBounds` | todo |
| `PlatformWindow::on_visual_viewport_changed` | trait-method | 1307 |  | `PlatformWindow.OnVisualViewportChanged` | todo |
| `PlatformWindow::prepare_frame` | trait-method | 1312 |  | `PlatformWindow.PrepareFrame` | todo |
| `PlatformWindow::resize` | trait-method | 1315 |  | `PlatformWindow.Resize` | todo |
| `PlatformWindow::scale_factor` | trait-method | 1316 |  | `PlatformWindow.ScaleFactor` | todo |
| `PlatformWindow::appearance` | trait-method | 1317 |  | `PlatformWindow.Appearance` | todo |
| `PlatformWindow::display` | trait-method | 1318 |  | `PlatformWindow.Display` | todo |
| `PlatformWindow::mouse_position` | trait-method | 1319 |  | `PlatformWindow.MousePosition` | todo |
| `PlatformWindow::modifiers` | trait-method | 1320 |  | `PlatformWindow.Modifiers` | todo |
| `PlatformWindow::capslock` | trait-method | 1321 |  | `PlatformWindow.Capslock` | todo |
| `PlatformWindow::set_input_handler` | trait-method | 1322 |  | `PlatformWindow.SetInputHandler` | todo |
| `PlatformWindow::take_input_handler` | trait-method | 1323 |  | `PlatformWindow.TakeInputHandler` | todo |
| `PlatformWindow::set_text_input_configuration` | trait-method | 1328 |  | `PlatformWindow.SetTextInputConfiguration` | todo |
| `PlatformWindow::prompt` | trait-method | 1329 |  | `PlatformWindow.Prompt` | todo |
| `PlatformWindow::activate` | trait-method | 1336 |  | `PlatformWindow.Activate` | todo |
| `PlatformWindow::request_attention` | trait-method | 1338 |  | `PlatformWindow.RequestAttention` | todo |
| `PlatformWindow::is_active` | trait-method | 1339 |  | `PlatformWindow.IsActive` | todo |
| `PlatformWindow::visibility` | trait-method | 1342 |  | `PlatformWindow.Visibility` | todo |
| `PlatformWindow::is_hovered` | trait-method | 1343 |  | `PlatformWindow.IsHovered` | todo |
| `PlatformWindow::background_appearance` | trait-method | 1344 |  | `PlatformWindow.BackgroundAppearance` | todo |
| `PlatformWindow::set_title` | trait-method | 1345 |  | `PlatformWindow.SetTitle` | todo |
| `PlatformWindow::set_background_appearance` | trait-method | 1346 |  | `PlatformWindow.SetBackgroundAppearance` | todo |
| `PlatformWindow::minimize` | trait-method | 1347 |  | `PlatformWindow.Minimize` | todo |
| `PlatformWindow::zoom` | trait-method | 1348 |  | `PlatformWindow.Zoom` | todo |
| `PlatformWindow::toggle_fullscreen` | trait-method | 1349 |  | `PlatformWindow.ToggleFullscreen` | todo |
| `PlatformWindow::is_fullscreen` | trait-method | 1350 |  | `PlatformWindow.IsFullscreen` | todo |
| `PlatformWindow::debug_state` | trait-method | 1352 |  | `PlatformWindow.DebugState` | todo |
| `PlatformWindow::frame_waker` | trait-method | 1355 |  | `PlatformWindow.FrameWaker` | todo |
| `PlatformWindow::on_request_frame` | trait-method | 1358 |  | `PlatformWindow.OnRequestFrame` | todo |
| `PlatformWindow::on_input` | trait-method | 1359 |  | `PlatformWindow.OnInput` | todo |
| `PlatformWindow::on_active_status_change` | trait-method | 1360 |  | `PlatformWindow.OnActiveStatusChange` | todo |
| `PlatformWindow::on_visibility_change` | trait-method | 1364 |  | `PlatformWindow.OnVisibilityChange` | todo |
| `PlatformWindow::on_hover_status_change` | trait-method | 1365 |  | `PlatformWindow.OnHoverStatusChange` | todo |
| `PlatformWindow::on_resize` | trait-method | 1366 |  | `PlatformWindow.OnResize` | todo |
| `PlatformWindow::on_moved` | trait-method | 1367 |  | `PlatformWindow.OnMoved` | todo |
| `PlatformWindow::on_display_changed` | trait-method | 1371 |  | `PlatformWindow.OnDisplayChanged` | todo |
| `PlatformWindow::on_should_close` | trait-method | 1372 |  | `PlatformWindow.OnShouldClose` | todo |
| `PlatformWindow::on_hit_test_window_control` | trait-method | 1373 |  | `PlatformWindow.OnHitTestWindowControl` | todo |
| `PlatformWindow::on_close` | trait-method | 1374 |  | `PlatformWindow.OnClose` | todo |
| `PlatformWindow::attach_native_view` | trait-method | 1385 |  | `PlatformWindow.AttachNativeView` | todo |
| `PlatformWindow::set_focused_native_view` | trait-method | 1394 |  | `PlatformWindow.SetFocusedNativeView` | todo |
| `PlatformWindow::on_hit_test_native_view` | trait-method | 1397 |  | `PlatformWindow.OnHitTestNativeView` | todo |
| `PlatformWindow::on_native_view_focused` | trait-method | 1404 |  | `PlatformWindow.OnNativeViewFocused` | todo |
| `PlatformWindow::on_key_down_before_native_view` | trait-method | 1410 |  | `PlatformWindow.OnKeyDownBeforeNativeView` | todo |
| `PlatformWindow::on_appearance_changed` | trait-method | 1415 |  | `PlatformWindow.OnAppearanceChanged` | todo |
| `PlatformWindow::on_button_layout_changed` | trait-method | 1416 |  | `PlatformWindow.OnButtonLayoutChanged` | todo |
| `PlatformWindow::draw` | trait-method | 1417 |  | `PlatformWindow.Draw` | todo |
| `PlatformWindow::schedule_frame` | trait-method | 1418 |  | `PlatformWindow.ScheduleFrame` | todo |
| `PlatformWindow::sprite_atlas` | trait-method | 1419 |  | `PlatformWindow.SpriteAtlas` | todo |
| `PlatformWindow::is_subpixel_rendering_supported` | trait-method | 1420 |  | `PlatformWindow.IsSubpixelRenderingSupported` | todo |
| `PlatformWindow::get_title` | trait-method | 1423 |  | `PlatformWindow.GetTitle` | todo |
| `PlatformWindow::tabbed_windows` | trait-method | 1426 |  | `PlatformWindow.TabbedWindows` | todo |
| `PlatformWindow::tab_bar_visible` | trait-method | 1429 |  | `PlatformWindow.TabBarVisible` | todo |
| `PlatformWindow::set_edited` | trait-method | 1432 |  | `PlatformWindow.SetEdited` | todo |
| `PlatformWindow::set_document_path` | trait-method | 1433 |  | `PlatformWindow.SetDocumentPath` | todo |
| `PlatformWindow::toggle_simple_fullscreen` | trait-method | 1434 |  | `PlatformWindow.ToggleSimpleFullscreen` | todo |
| `PlatformWindow::is_simple_fullscreen` | trait-method | 1435 |  | `PlatformWindow.IsSimpleFullscreen` | todo |
| `PlatformWindow::set_traffic_light_position` | trait-method | 1439 | macos | `PlatformWindow.SetTrafficLightPosition` | todo |
| `PlatformWindow::set_native_titlebar_visibility` | trait-method | 1441 | macos | `PlatformWindow.SetNativeTitlebarVisibility` | todo |
| `PlatformWindow::show_character_palette` | trait-method | 1442 |  | `PlatformWindow.ShowCharacterPalette` | todo |
| `PlatformWindow::titlebar_double_click` | trait-method | 1443 |  | `PlatformWindow.TitlebarDoubleClick` | todo |
| `PlatformWindow::on_move_tab_to_new_window` | trait-method | 1444 |  | `PlatformWindow.OnMoveTabToNewWindow` | todo |
| `PlatformWindow::on_merge_all_windows` | trait-method | 1445 |  | `PlatformWindow.OnMergeAllWindows` | todo |
| `PlatformWindow::on_select_previous_tab` | trait-method | 1446 |  | `PlatformWindow.OnSelectPreviousTab` | todo |
| `PlatformWindow::on_select_next_tab` | trait-method | 1447 |  | `PlatformWindow.OnSelectNextTab` | todo |
| `PlatformWindow::on_toggle_tab_bar` | trait-method | 1448 |  | `PlatformWindow.OnToggleTabBar` | todo |
| `PlatformWindow::merge_all_windows` | trait-method | 1449 |  | `PlatformWindow.MergeAllWindows` | todo |
| `PlatformWindow::move_tab_to_new_window` | trait-method | 1450 |  | `PlatformWindow.MoveTabToNewWindow` | todo |
| `PlatformWindow::toggle_window_tab_overview` | trait-method | 1451 |  | `PlatformWindow.ToggleWindowTabOverview` | todo |
| `PlatformWindow::set_tabbing_identifier` | trait-method | 1452 |  | `PlatformWindow.SetTabbingIdentifier` | todo |
| `PlatformWindow::native_window_state` | trait-method | 1454 |  | `PlatformWindow.NativeWindowState` | todo |
| `PlatformWindow::restore_native_window_state` | trait-method | 1457 |  | `PlatformWindow.RestoreNativeWindowState` | todo |
| `PlatformWindow::get_raw_handle` | trait-method | 1460 | windows | `PlatformWindow.GetRawHandle` | todo |
| `PlatformWindow::inner_window_bounds` | trait-method | 1463 |  | `PlatformWindow.InnerWindowBounds` | todo |
| `PlatformWindow::request_decorations` | trait-method | 1466 |  | `PlatformWindow.RequestDecorations` | todo |
| `PlatformWindow::show_window_menu` | trait-method | 1467 |  | `PlatformWindow.ShowWindowMenu` | todo |
| `PlatformWindow::start_window_move` | trait-method | 1468 |  | `PlatformWindow.StartWindowMove` | todo |
| `PlatformWindow::can_start_external_drag` | trait-method | 1469 |  | `PlatformWindow.CanStartExternalDrag` | todo |
| `PlatformWindow::start_external_drag` | trait-method | 1472 |  | `PlatformWindow.StartExternalDrag` | todo |
| `PlatformWindow::start_window_resize` | trait-method | 1475 |  | `PlatformWindow.StartWindowResize` | todo |
| `PlatformWindow::set_exclusive_zone` | trait-method | 1476 |  | `PlatformWindow.SetExclusiveZone` | todo |
| `PlatformWindow::set_exclusive_edge` | trait-method | 1478 | linux,feat:wayland | `PlatformWindow.SetExclusiveEdge` | todo |
| `PlatformWindow::set_input_region` | trait-method | 1479 |  | `PlatformWindow.SetInputRegion` | todo |
| `PlatformWindow::window_decorations` | trait-method | 1480 |  | `PlatformWindow.WindowDecorations` | todo |
| `PlatformWindow::set_app_id` | trait-method | 1483 |  | `PlatformWindow.SetAppID` | todo |
| `PlatformWindow::map_window` | trait-method | 1484 |  | `PlatformWindow.MapWindow` | todo |
| `PlatformWindow::window_controls` | trait-method | 1487 |  | `PlatformWindow.WindowControls` | todo |
| `PlatformWindow::set_client_inset` | trait-method | 1490 |  | `PlatformWindow.SetClientInset` | todo |
| `PlatformWindow::set_client_corner_radii` | trait-method | 1491 |  | `PlatformWindow.SetClientCornerRadii` | todo |
| `PlatformWindow::gpu_specs` | trait-method | 1492 |  | `PlatformWindow.GPUSpecs` | todo |
| `PlatformWindow::update_ime_position` | trait-method | 1494 |  | `PlatformWindow.UpdateIMEPosition` | todo |
| `PlatformWindow::insets` | trait-method | 1500 |  | `PlatformWindow.Insets` | todo |
| `PlatformWindow::on_insets_changed` | trait-method | 1509 |  | `PlatformWindow.OnInsetsChanged` | todo |
| `PlatformWindow::set_back_handler` | trait-method | 1513 |  | `PlatformWindow.SetBackHandler` | todo |
| `PlatformWindow::set_back_enabled` | trait-method | 1517 |  | `PlatformWindow.SetBackEnabled` | todo |
| `PlatformWindow::show_soft_keyboard` | trait-method | 1520 |  | `PlatformWindow.ShowSoftKeyboard` | todo |
| `PlatformWindow::hide_soft_keyboard` | trait-method | 1523 |  | `PlatformWindow.HideSoftKeyboard` | todo |
| `PlatformWindow::text_input_state_changed` | trait-method | 1526 |  | `PlatformWindow.TextInputStateChanged` | todo |
| `PlatformWindow::play_system_bell` | trait-method | 1528 |  | `PlatformWindow.PlaySystemBell` | todo |
| `PlatformWindow::a11y_init` | trait-method | 1531 |  | `PlatformWindow.A11yInit` | todo |
| `PlatformWindow::a11y_tree_update` | trait-method | 1534 |  | `PlatformWindow.A11yTreeUpdate` | todo |
| `PlatformWindow::a11y_update_window_bounds` | trait-method | 1537 |  | `PlatformWindow.A11yUpdateWindowBounds` | todo |
| `PlatformWindow::as_test` | trait-method | 1540 | feat:test-support,feat:bench-support | `PlatformWindow.AsTest` | todo |
| `PlatformWindow::render_to_image` | trait-method | 1548 | feat:test-support | `PlatformWindow.RenderToImage` | todo |
| `PlatformHeadlessRenderer` | trait | 1555 | feat:test-support,feat:bench-support | `PlatformHeadlessRenderer` | todo |
| `PlatformHeadlessRenderer::render_scene_to_image` | trait-method | 1557 | feat:test-support,feat:bench-support | `PlatformHeadlessRenderer.RenderSceneToImage` | todo |
| `PlatformHeadlessRenderer::render_scene` | trait-method | 1568 | feat:test-support,feat:bench-support | `PlatformHeadlessRenderer.RenderScene` | todo |
| `PlatformHeadlessRenderer::sprite_atlas` | trait-method | 1571 | feat:test-support,feat:bench-support | `PlatformHeadlessRenderer.SpriteAtlas` | todo |
| `RunnableVariant` | type | 1577 |  | `RunnableVariant` | todo |
| `TimerResolutionGuard` | type | 1580 |  | `TimerResolutionGuard` | todo |
| `TasksIncluded` | enum | 1583 |  | `TasksIncluded` | todo |
| `PlatformDispatcher` | trait | 1591 |  | `PlatformDispatcher` | todo |
| `PlatformDispatcher::is_main_thread` | trait-method | 1592 |  | `PlatformDispatcher.IsMainThread` | todo |
| `PlatformDispatcher::dispatch` | trait-method | 1593 |  | `PlatformDispatcher.Dispatch` | todo |
| `PlatformDispatcher::dispatch_on_main_thread` | trait-method | 1594 |  | `PlatformDispatcher.DispatchOnMainThread` | todo |
| `PlatformDispatcher::dispatch_after` | trait-method | 1595 |  | `PlatformDispatcher.DispatchAfter` | todo |
| `PlatformDispatcher::dispatch_on_main_thread_when_idle` | trait-method | 1597 |  | `PlatformDispatcher.DispatchOnMainThreadWhenIdle` | todo |
| `PlatformDispatcher::idle_time_remaining` | trait-method | 1606 |  | `PlatformDispatcher.IdleTimeRemaining` | todo |
| `PlatformDispatcher::spawn_realtime` | trait-method | 1610 |  | `PlatformDispatcher.SpawnRealtime` | todo |
| `PlatformDispatcher::now` | trait-method | 1612 |  | `PlatformDispatcher.Now` | todo |
| `PlatformDispatcher::increase_timer_resolution` | trait-method | 1616 |  | `PlatformDispatcher.IncreaseTimerResolution` | todo |
| `PlatformDispatcher::prevent_app_nap` | trait-method | 1620 |  | `PlatformDispatcher.PreventAppNap` | todo |
| `PlatformDispatcher::as_test` | trait-method | 1625 | feat:test-support,feat:bench-support | `PlatformDispatcher.AsTest` | todo |
| `PlatformDispatcher::as_threaded` | trait-method | 1632 | feat:test-support,feat:bench-support | `PlatformDispatcher.AsThreaded` | todo |
| `PlatformTextSystem` | trait | 1638 |  | `PlatformTextSystem` | todo |
| `PlatformTextSystem::add_fonts` | trait-method | 1639 |  | `PlatformTextSystem.AddFonts` | todo |
| `PlatformTextSystem::set_missing_glyph_sink` | trait-method | 1641 |  | `PlatformTextSystem.SetMissingGlyphSink` | todo |
| `PlatformTextSystem::all_font_names` | trait-method | 1643 |  | `PlatformTextSystem.AllFontNames` | todo |
| `PlatformTextSystem::font_id` | trait-method | 1645 |  | `PlatformTextSystem.FontID` | todo |
| `PlatformTextSystem::prewarm_fonts` | trait-method | 1647 |  | `PlatformTextSystem.PrewarmFonts` | todo |
| `PlatformTextSystem::font_metrics` | trait-method | 1649 |  | `PlatformTextSystem.FontMetrics` | todo |
| `PlatformTextSystem::typographic_bounds` | trait-method | 1651 |  | `PlatformTextSystem.TypographicBounds` | todo |
| `PlatformTextSystem::advance` | trait-method | 1653 |  | `PlatformTextSystem.Advance` | todo |
| `PlatformTextSystem::glyph_for_char` | trait-method | 1655 |  | `PlatformTextSystem.GlyphForChar` | todo |
| `PlatformTextSystem::glyph_raster_bounds` | trait-method | 1657 |  | `PlatformTextSystem.GlyphRasterBounds` | todo |
| `PlatformTextSystem::rasterize_glyph` | trait-method | 1659 |  | `PlatformTextSystem.RasterizeGlyph` | todo |
| `PlatformTextSystem::layout_line` | trait-method | 1665 |  | `PlatformTextSystem.LayoutLine` | todo |
| `PlatformTextSystem::recommended_rendering_mode` | trait-method | 1667 |  | `PlatformTextSystem.RecommendedRenderingMode` | todo |
| `PlatformTextSystem::glyph_dilation_for_color` | trait-method | 1670 |  | `PlatformTextSystem.GlyphDilationForColor` | todo |
| `NoopTextSystem` | struct | 1676 |  | `NoopTextSystem` | todo |
| `NoopTextSystem::new` | assoc-fn | 1681 |  | `NewNoopTextSystem` | todo |
| `get_gamma_correction_ratios` | fn | 1810 |  | `GetGammaCorrectionRatios` | todo |
| `AtlasKey` | enum | 1843 |  | `AtlasKey` | todo |
| `AtlasKey::texture_kind` | method | 1851 |  | `(AtlasKey).TextureKind` | todo |
| `PlatformAtlas` | trait | 1887 |  | `PlatformAtlas` | todo |
| `PlatformAtlas::get_or_insert_with` | trait-method | 1889 |  | `PlatformAtlas.GetOrInsertWith` | todo |
| `PlatformAtlas::remove` | trait-method | 1894 |  | `PlatformAtlas.Remove` | todo |
| `PlatformAtlas::contains` | trait-method | 1897 | feat:test-support,feat:bench-support | `PlatformAtlas.Contains` | todo |
| `AtlasBackend` | trait | 1903 |  | `AtlasBackend` | todo |
| `AtlasBackend::insert` | trait-method | 1904 |  | `AtlasBackend.Insert` | todo |
| `AtlasBackend::remove` | trait-method | 1911 |  | `AtlasBackend.Remove` | todo |
| `AtlasState` | struct | 1915 |  | `AtlasState` | todo |
| `AtlasState::new` | assoc-fn | 1921 |  | `NewAtlasState` | todo |
| `AtlasState::contains` | method | 1928 |  | `(AtlasState).Contains` | todo |
| `AtlasState::clear` | method | 1932 |  | `(AtlasState).Clear` | todo |
| `AtlasState::get_or_insert_with` | method | 1945 |  | `(AtlasState).GetOrInsertWith` | todo |
| `AtlasState::remove` | method | 1966 |  | `(AtlasState).Remove` | todo |
| `HeadlessAtlas` | struct | 1977 |  | `HeadlessAtlas` | todo |
| `HeadlessAtlasBackend` | struct | 1981 |  | `HeadlessAtlasBackend` | todo |
| `AtlasTextureList` | struct | 2033 |  | `AtlasTextureList` | todo |
| `AtlasTextureList::drain` | method | 2057 |  | `(AtlasTextureList).Drain` | todo |
| `AtlasTextureList::iter_mut` | method | 2063 |  | `(AtlasTextureList).IterMut` | todo |
| `AtlasTile` | struct | 2071 |  | `AtlasTile` | todo |
| `AtlasTextureId` | struct | 2085 |  | `AtlasTextureID` | todo |
| `AtlasTextureKind` | enum | 2103 |  | `AtlasTextureKind` | todo |
| `TileId` | struct | 2112 |  | `TileID` | todo |
| `PlatformInputHandler` | struct | 2127 |  | `PlatformInputHandler` | todo |
| `PlatformInputHandler::new` | assoc-fn | 2142 |  | `NewPlatformInputHandler` | todo |
| `PlatformInputHandler::focus_id` | method | 2159 |  | `(PlatformInputHandler).FocusID` | todo |
| `PlatformInputHandler::selected_text_range` | method | 2163 |  | `(PlatformInputHandler).SelectedTextRange` | todo |
| `PlatformInputHandler::marked_text_range` | method | 2174 |  | `(PlatformInputHandler).MarkedTextRange` | todo |
| `PlatformInputHandler::text_for_range` | method | 2185 |  | `(PlatformInputHandler).TextForRange` | todo |
| `PlatformInputHandler::replace_text_in_range` | method | 2199 |  | `(PlatformInputHandler).ReplaceTextInRange` | todo |
| `PlatformInputHandler::replace_and_mark_text_in_range` | method | 2208 |  | `(PlatformInputHandler).ReplaceAndMarkTextInRange` | todo |
| `PlatformInputHandler::unmark_text` | method | 2228 |  | `(PlatformInputHandler).UnmarkText` | todo |
| `PlatformInputHandler::paste` | method | 2234 |  | `(PlatformInputHandler).Paste` | todo |
| `PlatformInputHandler::bounds_for_range` | method | 2240 |  | `(PlatformInputHandler).BoundsForRange` | todo |
| `PlatformInputHandler::selection_bounds_for_range` | method | 2247 |  | `(PlatformInputHandler).SelectionBoundsForRange` | todo |
| `PlatformInputHandler::apple_press_and_hold_enabled` | method | 2257 |  | `(PlatformInputHandler).ApplePressAndHoldEnabled` | todo |
| `PlatformInputHandler::dispatch_input` | method | 2267 |  | `(PlatformInputHandler).DispatchInput` | todo |
| `PlatformInputHandler::compute_ime_candidate_bounds` | assoc-fn | 2271 |  | `PlatformInputHandlerComputeIMECandidateBounds` | todo |
| `PlatformInputHandler::selected_bounds` | method | 2306 |  | `(PlatformInputHandler).SelectedBounds` | todo |
| `PlatformInputHandler::ime_candidate_bounds` | method | 2314 |  | `(PlatformInputHandler).IMECandidateBounds` | todo |
| `PlatformInputHandler::character_index_for_point` | method | 2323 |  | `(PlatformInputHandler).CharacterIndexForPoint` | todo |
| `PlatformInputHandler::set_selected_text_range` | method | 2331 |  | `(PlatformInputHandler).SetSelectedTextRange` | todo |
| `PlatformInputHandler::element_bounds` | method | 2341 |  | `(PlatformInputHandler).ElementBounds` | todo |
| `PlatformInputHandler::text_length_utf16` | method | 2349 |  | `(PlatformInputHandler).TextLengthUTF16` | todo |
| `PlatformInputHandler::accepts_text_input` | method | 2357 |  | `(PlatformInputHandler).AcceptsTextInput` | todo |
| `PlatformInputHandler::query_accepts_text_input` | method | 2362 |  | `(PlatformInputHandler).QueryAcceptsTextInput` | todo |
| `PlatformInputHandler::query_accepts_focused_text_input` | method | 2372 |  | `(PlatformInputHandler).QueryAcceptsFocusedTextInput` | todo |
| `PlatformInputHandler::query_prefers_ime_for_printable_keys` | method | 2388 |  | `(PlatformInputHandler).QueryPrefersIMEForPrintableKeys` | todo |
| `PlatformInputHandler::text_input_configuration` | method | 2399 |  | `(PlatformInputHandler).TextInputConfiguration` | todo |
| `PlatformInputHandler::text_input_editable_range` | method | 2408 |  | `(PlatformInputHandler).TextInputEditableRange` | todo |
| `UTF16Selection` | struct | 2419 |  | `UTF16Selection` | todo |
| `InputHandler` | trait | 2432 |  | `InputHandler` | todo |
| `InputHandler::selected_text_range` | trait-method | 2437 |  | `InputHandler.SelectedTextRange` | todo |
| `InputHandler::marked_text_range` | trait-method | 2448 |  | `InputHandler.MarkedTextRange` | todo |
| `InputHandler::text_for_range` | trait-method | 2454 |  | `InputHandler.TextForRange` | todo |
| `InputHandler::replace_text_in_range` | trait-method | 2466 |  | `InputHandler.ReplaceTextInRange` | todo |
| `InputHandler::replace_and_mark_text_in_range` | trait-method | 2480 |  | `InputHandler.ReplaceAndMarkTextInRange` | todo |
| `InputHandler::unmark_text` | trait-method | 2491 |  | `InputHandler.UnmarkText` | todo |
| `InputHandler::paste` | trait-method | 2499 |  | `InputHandler.Paste` | todo |
| `InputHandler::bounds_for_range` | trait-method | 2509 |  | `InputHandler.BoundsForRange` | todo |
| `InputHandler::selection_bounds_for_range` | trait-method | 2522 |  | `InputHandler.SelectionBoundsForRange` | todo |
| `InputHandler::character_index_for_point` | trait-method | 2534 |  | `InputHandler.CharacterIndexForPoint` | todo |
| `InputHandler::set_selected_text_range` | trait-method | 2550 |  | `InputHandler.SetSelectedTextRange` | todo |
| `InputHandler::element_bounds` | trait-method | 2564 |  | `InputHandler.ElementBounds` | todo |
| `InputHandler::text_length_utf16` | trait-method | 2569 |  | `InputHandler.TextLengthUTF16` | todo |
| `InputHandler::apple_press_and_hold_enabled` | trait-method | 2578 |  | `InputHandler.ApplePressAndHoldEnabled` | todo |
| `InputHandler::accepts_text_input` | trait-method | 2583 |  | `InputHandler.AcceptsTextInput` | todo |
| `InputHandler::text_input_editable_range` | trait-method | 2597 |  | `InputHandler.TextInputEditableRange` | todo |
| `InputHandler::prefers_ime_for_printable_keys` | trait-method | 2613 |  | `InputHandler.PrefersIMEForPrintableKeys` | todo |
| `InputHandler::text_input_configuration` | trait-method | 2622 |  | `InputHandler.TextInputConfiguration` | todo |
| `TextInputConfiguration` | struct | 2641 |  | `TextInputConfiguration` | todo |
| `Autocapitalize` | enum | 2654 |  | `Autocapitalize` | todo |
| `TextInputAction` | enum | 2676 |  | `TextInputAction` | todo |
| `WindowOptions` | struct | 2698 |  | `WindowOptions` | todo |
| `WindowParams` | struct | 2795 |  | `WindowParams` | todo |
| `WindowBounds` | enum | 2855 |  | `WindowBounds` | todo |
| `WindowBounds::get_bounds` | method | 2874 |  | `(WindowBounds).GetBounds` | todo |
| `WindowBounds::centered` | assoc-fn | 2884 |  | `WindowBoundsCentered` | todo |
| `TitlebarOptions` | struct | 2922 |  | `TitlebarOptions` | todo |
| `WindowKind` | enum | 2936 |  | `WindowKind` | todo |
| `WindowAppearance` | enum | 2970 |  | `WindowAppearance` | todo |
| `WindowBackgroundAppearance` | enum | 2996 |  | `WindowBackgroundAppearance` | todo |
| `TextRenderingMode` | enum | 3020 |  | `TextRenderingMode` | todo |
| `PathPromptOptions` | struct | 3032 |  | `PathPromptOptions` | todo |
| `PromptLevel` | enum | 3045 |  | `PromptLevel` | todo |
| `PromptButton` | enum | 3058 |  | `PromptButton` | todo |
| `PromptButton::new` | assoc-fn | 3069 |  | `NewPromptButton` | todo |
| `PromptButton::ok` | assoc-fn | 3074 |  | `PromptButtonOk` | todo |
| `PromptButton::cancel` | assoc-fn | 3079 |  | `PromptButtonCancel` | todo |
| `PromptButton::is_cancel` | method | 3085 |  | `(PromptButton).IsCancel` | todo |
| `PromptButton::label` | method | 3090 |  | `(PromptButton).Label` | todo |
| `CursorStyle` | enum | 3113 |  | `CursorStyle` | todo |
| `ClipboardItem` | struct | 3201 |  | `ClipboardItem` | todo |
| `ClipboardReadError` | enum | 3211 |  | `ClipboardReadError` | todo |
| `ClipboardEntry` | enum | 3242 |  | `ClipboardEntry` | todo |
| `ClipboardItem::new_string` | assoc-fn | 3253 |  | `NewClipboardItemString` | todo |
| `ClipboardItem::new_string_with_metadata` | assoc-fn | 3260 |  | `NewClipboardItemStringWithMetadata` | todo |
| `ClipboardItem::new_string_with_json_metadata` | assoc-fn | 3270 |  | `NewClipboardItemStringWithJSONMetadata` | todo |
| `ClipboardItem::new_image` | assoc-fn | 3279 |  | `NewClipboardItemImage` | todo |
| `ClipboardItem::text` | method | 3287 |  | `(ClipboardItem).Text` | todo |
| `ClipboardItem::metadata` | method | 3316 |  | `(ClipboardItem).Metadata` | todo |
| `ClipboardItem::entries` | method | 3326 |  | `(ClipboardItem).Entries` | todo |
| `ClipboardItem::into_entries` | method | 3331 |  | `(ClipboardItem).IntoEntries` | todo |
| `ImageFormat` | enum | 3376 |  | `ImageFormat` | todo |
| `ImageFormat::mime_type` | method | 3402 |  | `(ImageFormat).MimeType` | todo |
| `ImageFormat::extension` | method | 3417 |  | `(ImageFormat).Extension` | todo |
| `ImageFormat::from_mime_type` | assoc-fn | 3432 |  | `ImageFormatFromMimeType` | todo |
| `Image` | struct | 3453 |  | `Image` | todo |
| `Image::empty` | assoc-fn | 3541 |  | `ImageEmpty` | todo |
| `Image::from_bytes` | assoc-fn | 3546 |  | `ImageFromBytes` | todo |
| `Image::id` | method | 3555 |  | `(Image).ID` | todo |
| `Image::use_render_image` | method | 3560 |  | `(Image).UseRenderImage` | todo |
| `Image::get_render_image` | method | 3571 |  | `(Image).GetRenderImage` | todo |
| `Image::remove_asset` | method | 3582 |  | `(Image).RemoveAsset` | todo |
| `Image::is_asset_cached` | method | 3589 | feat:test-support | `(Image).IsAssetCached` | todo |
| `Image::to_image_data` | method | 3594 |  | `(Image).ToImageData` | todo |
| `Image::format` | method | 3615 |  | `(Image).Format` | todo |
| `Image::bytes` | method | 3620 |  | `(Image).Bytes` | todo |
| `ClipboardString` | struct | 3627 |  | `ClipboardString` | todo |
| `ClipboardString::new` | assoc-fn | 3636 |  | `NewClipboardString` | todo |
| `ClipboardString::with_json_metadata` | method | 3645 |  | `(ClipboardString).WithJSONMetadata` | todo |
| `ClipboardString::text` | method | 3651 |  | `(ClipboardString).Text` | todo |
| `ClipboardString::into_text` | method | 3656 |  | `(ClipboardString).IntoText` | todo |
| `ClipboardString::metadata_json` | method | 3661 |  | `(ClipboardString).MetadataJSON` | todo |
| `ClipboardString::text_hash` | assoc-fn | 3672 |  | `ClipboardStringTextHash` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_image_to_image_data_applies_exif_orientation` | test | 3694 |  | `TestPlatformImageTests_ImageToImageDataAppliesExifOrientation` | todo |
| `test_svg_image_to_image_data_converts_to_bgra` | test | 3710 |  | `TestPlatformImageTests_SvgImageToImageDataConvertsToBgra` | todo |
| `test_animated_webp_image_to_image_data_decodes_frames` | test | 3728 |  | `TestPlatformImageTests_AnimatedWebpImageToImageDataDecodesFrames` | todo |
| `only_successful_inserts_are_cached` | test | 3813 |  | `TestPlatformAtlasTests_OnlySuccessfulInsertsAreCached` | todo |
| `remove_and_clear_invalidate_keys` | test | 3850 |  | `TestPlatformAtlasTests_RemoveAndClearInvalidateKeys` | todo |
| `coalesced_signals_retain_the_earliest_until_drained` | test | 3882 | feat:profiler | `TestPlatformFrameSignalTests_CoalescedSignalsRetainTheEarliestUntilDrained` | todo |
| `disabled_profiling_does_not_capture_or_store_timestamps` | test | 3910 | feat:profiler | `TestPlatformFrameSignalTests_DisabledProfilingDoesNotCaptureOrStoreTimestamps` | todo |
| `test_refresh_interval_from_hz` | test | 3927 | linux,freebsd | `TestPlatform_RefreshIntervalFromHz` | todo |
| `test_window_button_layout_parse_standard` | test | 3942 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseStandard` | todo |
| `test_window_button_layout_parse_right_only` | test | 3956 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseRightOnly` | todo |
| `test_window_button_layout_parse_left_only` | test | 3970 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseLeftOnly` | todo |
| `test_window_button_layout_parse_with_whitespace` | test | 3984 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseWithWhitespace` | todo |
| `test_window_button_layout_parse_empty` | test | 3998 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseEmpty` | todo |
| `test_window_button_layout_parse_intentionally_empty` | test | 4005 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseIntentionallyEmpty` | todo |
| `test_window_button_layout_parse_invalid_buttons` | test | 4012 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseInvalidButtons` | todo |
| `test_window_button_layout_parse_deduplicates_same_side_buttons` | test | 4026 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseDeduplicatesSameSideButtons` | todo |
| `test_window_button_layout_parse_deduplicates_buttons_across_sides` | test | 4040 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseDeduplicatesButtonsAcrossSides` | todo |
| `test_window_button_layout_parse_gnome_style` | test | 4065 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseGnomeStyle` | todo |
| `test_window_button_layout_parse_elementary_style` | test | 4072 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseElementaryStyle` | todo |
| `test_window_button_layout_round_trip` | test | 4079 | linux,freebsd | `TestPlatform_WindowButtonLayoutRoundTrip` | todo |
| `test_window_button_layout_linux_default` | test | 4096 | linux,freebsd | `TestPlatform_WindowButtonLayoutLinuxDefault` | todo |
| `test_window_button_layout_parse_all_invalid` | test | 4113 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseAllInvalid` | todo |

## gpui/src/platform/app_menu.rs

Go target: `platform_app_menu.go` (package `gpui`), tests in `platform_app_menu_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Menu` | struct | 4 |  | `Menu` | todo |
| `Menu::new` | assoc-fn | 17 |  | `NewMenu` | todo |
| `Menu::items` | method | 26 |  | `(Menu).Items` | todo |
| `Menu::disabled` | method | 32 |  | `(Menu).Disabled` | todo |
| `Menu::owned` | method | 38 |  | `(Menu).Owned` | todo |
| `OsMenu` | struct | 50 |  | `OsMenu` | todo |
| `OsMenu::owned` | method | 60 |  | `(OsMenu).Owned` | todo |
| `SystemMenuType` | enum | 70 |  | `SystemMenuType` | todo |
| `MenuItem` | enum | 76 |  | `MenuItem` | todo |
| `MenuItem::separator` | assoc-fn | 108 |  | `MenuItemSeparator` | todo |
| `MenuItem::submenu` | assoc-fn | 113 |  | `MenuItemSubmenu` | todo |
| `MenuItem::os_submenu` | assoc-fn | 118 |  | `MenuItemOsSubmenu` | todo |
| `MenuItem::action` | assoc-fn | 126 |  | `MenuItemAction` | todo |
| `MenuItem::os_action` | assoc-fn | 137 |  | `MenuItemOsAction` | todo |
| `MenuItem::owned` | method | 152 |  | `(MenuItem).Owned` | todo |
| `MenuItem::checked` | method | 176 |  | `(MenuItem).Checked` | todo |
| `MenuItem::is_checked` | method | 190 |  | `(MenuItem).IsChecked` | todo |
| `MenuItem::disabled` | method | 198 |  | `(MenuItem).Disabled` | todo |
| `MenuItem::is_disabled` | method | 215 |  | `(MenuItem).IsDisabled` | todo |
| `OwnedOsMenu` | struct | 228 |  | `OwnedOsMenu` | todo |
| `OwnedMenu` | struct | 238 |  | `OwnedMenu` | todo |
| `OwnedMenuItem` | enum | 250 |  | `OwnedMenuItem` | todo |
| `OsAction` | enum | 311 |  | `OsAction` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_menu` | test | 366 |  | `TestAppMenu_Menu` | todo |
| `test_menu_item_builder` | test | 380 |  | `TestAppMenu_MenuItemBuilder` | todo |

## gpui/src/platform/layer_shell.rs

Go target: `platform_layer_shell.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Layer` | enum | 9 |  | `Layer` | todo |
| `KeyboardInteractivity` | enum | 43 |  | `KeyboardInteractivity` | todo |
| `LayerShellOptions` | struct | 59 |  | `LayerShellOptions` | todo |
| `LayerShellNotSupportedError` | struct | 83 |  | `LayerShellNotSupportedError` | todo |

## gpui/src/platform/platform_info.rs

Go target: `platform_platform_info.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PlatformInfo` | struct | 9 |  | `PlatformInfo` | todo |
| `OperatingSystem` | enum | 18 |  | `OperatingSystem` | todo |
| `PlatformTarget` | enum | 37 |  | `PlatformTarget` | todo |
| `PlatformInfo::is_mac` | method | 46 |  | `(PlatformInfo).IsMac` | todo |

## gpui/src/platform/popup.rs

Go target: `platform_popup.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PopupOptions` | struct | 16 |  | `PopupOptions` | todo |
| `PopupAnchor` | enum | 57 |  | `PopupAnchor` | todo |
| `PopupGravity` | enum | 84 |  | `PopupGravity` | todo |
| `PopupNotSupportedError` | struct | 134 |  | `PopupNotSupportedError` | todo |

## gpui/src/platform/scap_screen_capture.rs

Go target: `platform_scap_screen_capture.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `scap_screen_sources` | fn | 18 |  | `ScapScreenSources` | todo |

## gpui/src/platform/threaded_dispatcher.rs

Go target: `platform_threaded_dispatcher.go` (package `gpui`), tests in `platform_threaded_dispatcher_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ThreadedDispatcher` | struct | 28 |  | `ThreadedDispatcher` | todo |
| `ThreadedDispatcher::new` | assoc-fn | 125 |  | `NewThreadedDispatcher` | todo |
| `ThreadedDispatcher::run_until_idle` | method | 220 |  | `(ThreadedDispatcher).RunUntilIdle` | todo |
| `ThreadedDispatcher::run_ready_main_tasks` | method | 307 |  | `(ThreadedDispatcher).RunReadyMainTasks` | todo |
| `ThreadedDispatcher::cancel_pending_timers` | method | 373 |  | `(ThreadedDispatcher).CancelPendingTimers` | todo |
| `ThreadedDispatcher::debug_state` | method | 387 |  | `(ThreadedDispatcher).DebugState` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `is_idle_tracks_queued_work_but_ignores_undue_timers` | test | 498 |  | `TestThreadedDispatcher_IsIdleTracksQueuedWorkButIgnoresUndueTimers` | todo |
| `run_ready_main_tasks_does_not_wait_for_background_handoffs` | test | 524 |  | `TestThreadedDispatcher_RunReadyMainTasksDoesNotWaitForBackgroundHandoffs` | todo |
| `run_until_idle_completes_background_to_main_handoffs` | test | 556 |  | `TestThreadedDispatcher_RunUntilIdleCompletesBackgroundToMainHandoffs` | todo |
| `run_until_waits_for_untracked_external_wakes` | test | 585 |  | `TestThreadedDispatcher_RunUntilWaitsForUntrackedExternalWakes` | todo |
| `run_until_returns_at_readiness_despite_requeuing_main_work` | test | 615 |  | `TestThreadedDispatcher_RunUntilReturnsAtReadinessDespiteRequeuingMainWork` | todo |
| `run_ready_main_tasks_advances_requeuing_work_one_batch_per_call` | test | 671 |  | `TestThreadedDispatcher_RunReadyMainTasksAdvancesRequeuingWorkOneBatchPerCall` | todo |
| `timers_fire_in_real_time` | test | 697 |  | `TestThreadedDispatcher_TimersFireInRealTime` | todo |
| `cancel_pending_timers_wakes_waiters_without_waiting_for_deadline` | test | 721 |  | `TestThreadedDispatcher_CancelPendingTimersWakesWaitersWithoutWaitingForDeadline` | todo |

## gpui/src/platform/visual_test.rs

Go target: `platform_visual_testing.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `VisualTestPlatform` | struct | 31 |  | `VisualTestPlatform` | todo |
| `VisualTestPlatform::new` | assoc-fn | 44 |  | `NewVisualTestPlatform` | todo |
| `VisualTestPlatform::dispatcher` | method | 62 |  | `(VisualTestPlatform).Dispatcher` | todo |

## gpui/src/platform/test/dispatcher.rs

Go target: `platform_testing_dispatcher.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestDispatcher` | struct | 17 |  | `TestDispatcher` | todo |
| `TestDispatcher::new` | assoc-fn | 24 |  | `NewTestDispatcher` | todo |
| `TestDispatcher::from_scheduler` | assoc-fn | 36 |  | `TestDispatcherFromScheduler` | todo |
| `TestDispatcher::scheduler` | method | 44 |  | `(TestDispatcher).Scheduler` | todo |
| `TestDispatcher::session_id` | method | 48 |  | `(TestDispatcher).SessionID` | todo |
| `TestDispatcher::drain_tasks` | method | 52 |  | `(TestDispatcher).DrainTasks` | todo |
| `TestDispatcher::advance_clock` | method | 56 |  | `(TestDispatcher).AdvanceClock` | todo |
| `TestDispatcher::advance_clock_to_next_timer` | method | 60 |  | `(TestDispatcher).AdvanceClockToNextTimer` | todo |
| `TestDispatcher::simulate_random_delay` | method | 64 |  | `(TestDispatcher).SimulateRandomDelay` | todo |
| `TestDispatcher::tick` | method | 68 |  | `(TestDispatcher).Tick` | todo |
| `TestDispatcher::run_until_parked` | method | 76 |  | `(TestDispatcher).RunUntilParked` | todo |
| `TestDispatcher::allow_parking` | method | 80 |  | `(TestDispatcher).AllowParking` | todo |
| `TestDispatcher::forbid_parking` | method | 84 |  | `(TestDispatcher).ForbidParking` | todo |
| `TestDispatcher::set_num_cpus` | method | 90 |  | `(TestDispatcher).SetNumCPUs` | todo |
| `TestDispatcher::num_cpus_override` | method | 95 |  | `(TestDispatcher).NumCPUsOverride` | todo |

## gpui/src/platform/test/platform.rs

Go target: `platform_testing_platform.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestScreenCaptureSource` | struct | 64 |  | `TestScreenCaptureSource` | todo |
| `TestScreenCaptureStream` | struct | 67 |  | `TestScreenCaptureStream` | todo |
| `TestScreenCaptureSource::new` | assoc-fn | 846 | feat:test-support | `NewTestScreenCaptureSource` | todo |

## gpui/src/platform/test/window.rs

Go target: `platform_testing_window.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestWindow` | struct | 71 |  | `TestWindow` | todo |
| `TestWindow::simulate_close` | method | 152 |  | `(TestWindow).SimulateClose` | todo |
| `TestWindow::simulate_scheduled_frame` | method | 159 |  | `(TestWindow).SimulateScheduledFrame` | todo |
| `TestWindow::frame_scheduled` | method | 178 |  | `(TestWindow).FrameScheduled` | todo |
| `TestWindow::simulate_visibility_change` | method | 182 |  | `(TestWindow).SimulateVisibilityChange` | todo |
| `TestWindow::simulate_move_to_display` | method | 195 |  | `(TestWindow).SimulateMoveToDisplay` | todo |
| `TestWindow::simulate_visual_viewport_change` | method | 207 |  | `(TestWindow).SimulateVisualViewportChange` | todo |
| `TestWindow::simulate_insets_change` | method | 219 |  | `(TestWindow).SimulateInsetsChange` | todo |
| `TestWindow::virtual_keyboard_requests` | method | 231 |  | `(TestWindow).VirtualKeyboardRequests` | todo |
| `TestWindow::virtual_keyboard_dismissals` | method | 235 |  | `(TestWindow).VirtualKeyboardDismissals` | todo |
| `TestWindow::text_input_configurations` | method | 240 |  | `(TestWindow).TextInputConfigurations` | todo |
| `TestWindow::text_input_state_changes` | method | 244 |  | `(TestWindow).TextInputStateChanges` | todo |
| `TestWindow::simulate_resize` | method | 248 |  | `(TestWindow).SimulateResize` | todo |
| `TestWindow::simulate_scale_factor_change` | method | 262 |  | `(TestWindow).SimulateScaleFactorChange` | todo |
| `TestWindow::attached_native_view_count` | method | 272 |  | `(TestWindow).AttachedNativeViewCount` | todo |
| `TestWindow::focused_native_view` | method | 277 |  | `(TestWindow).FocusedNativeView` | todo |
| `TestWindow::hit_test_native_view` | method | 282 |  | `(TestWindow).HitTestNativeView` | todo |
| `TestWindow::simulate_key_down_before_native_view` | method | 291 |  | `(TestWindow).SimulateKeyDownBeforeNativeView` | todo |
| `TestWindow::simulate_native_view_focused` | method | 301 |  | `(TestWindow).SimulateNativeViewFocused` | todo |
| `TestWindow::simulate_appearance_change` | method | 322 |  | `(TestWindow).SimulateAppearanceChange` | todo |
| `TestWindow::simulate_decorations_change` | method | 333 |  | `(TestWindow).SimulateDecorationsChange` | todo |
| `TestWindow::frame_wake_count` | method | 345 |  | `(TestWindow).FrameWakeCount` | todo |
| `TestWindow::simulate_frame_request` | method | 351 |  | `(TestWindow).SimulateFrameRequest` | todo |
| `TestWindow::simulate_input` | method | 361 |  | `(TestWindow).SimulateInput` | todo |
| `TestWindow::external_drag_files` | method | 372 |  | `(TestWindow).ExternalDragFiles` | todo |
| `TestWindow::set_start_external_drag_result` | method | 376 |  | `(TestWindow).SetStartExternalDragResult` | todo |

