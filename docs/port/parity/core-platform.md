# Parity: core-platform

Scope: gpui: platform interfaces, TestPlatform, dispatchers. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/platform.rs

Go target: `platform.go` (package `gpui`), tests in `platform_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WindowVisibility` | enum | 107 |  | `WindowVisibility` | done |
| `WindowVisibility::is_visible` | method | 118 |  | `(WindowVisibility).IsVisible` | done |
| `DisplayEvent` | enum | 127 |  | `DisplayEvent` | done |
| `DisplayChanges` | struct | 145 |  | `DisplayChanges` | done |
| `refresh_interval_from_hz` | fn | 155 |  | `RefreshIntervalFromHz` | done |
| `ActivationPolicy` | enum | 166 |  | `ActivationPolicy` | done |
| `ActivityGuard` | struct | 187 |  | `ActivityGuard` | done |
| `ActivityGuard::new` | assoc-fn | 193 |  | `NewActivityGuard` | done |
| `ActivityGuard::noop` | assoc-fn | 200 |  | `ActivityGuardNoop` | done |
| `guess_compositor` | fn | 210 | linux,freebsd | `GuessCompositor` | done |
| `GraphicalEnvironment` | struct | 227 | linux,freebsd | `GraphicalEnvironment` | done |
| `GraphicalEnvironment` | struct | 254 | windows | `GraphicalEnvironment` | done |
| `GraphicalEnvironment::detect` | assoc-fn | 263 | windows | `GraphicalEnvironmentDetect` | done |
| `GraphicalEnvironment::apply_to` | method | 279 | windows | `(GraphicalEnvironment).ApplyTo` | done |
| `GraphicalEnvironment::desktop_environment` | method | 282 | windows | `(GraphicalEnvironment).DesktopEnvironment` | done |
| `GraphicalEnvironment` | struct | 290 | linux,freebsd,windows | `GraphicalEnvironment` | done |
| `GraphicalEnvironment::detect` | assoc-fn | 295 | linux,freebsd,windows | `GraphicalEnvironmentDetect` | done |
| `GraphicalEnvironment::apply_to` | method | 300 | linux,freebsd,windows | `(GraphicalEnvironment).ApplyTo` | done |
| `GraphicalEnvironment::desktop_environment` | method | 303 | linux,freebsd,windows | `(GraphicalEnvironment).DesktopEnvironment` | done |
| `WindowingRequest` | enum | 310 |  | `WindowingRequest` | done |
| `GraphicalEnvironment::detect` | assoc-fn | 323 | linux,freebsd | `GraphicalEnvironmentDetect` | done |
| `GraphicalEnvironment::apply_to` | method | 336 | linux,freebsd | `(GraphicalEnvironment).ApplyTo` | done |
| `GraphicalEnvironment::desktop_environment` | method | 353 | linux,freebsd | `(GraphicalEnvironment).DesktopEnvironment` | done |
| `GraphicalEnvironment::guess_compositor` | method | 364 | linux,freebsd | `(GraphicalEnvironment).GuessCompositor` | done |
| `WindowingModes` | struct | 381 | linux,freebsd | `WindowingModes` | done |
| `Platform` | trait | 394 |  | `Platform` | done |
| `Platform::platform_info` | trait-method | 396 |  | `Platform.PlatformInfo` | done |
| `Platform::background_executor` | trait-method | 398 |  | `Platform.BackgroundExecutor` | done |
| `Platform::foreground_executor` | trait-method | 399 |  | `Platform.ForegroundExecutor` | done |
| `Platform::text_system` | trait-method | 400 |  | `Platform.TextSystem` | done |
| `Platform::run` | trait-method | 402 |  | `Platform.Run` | done |
| `Platform::quit` | trait-method | 403 |  | `Platform.Quit` | done |
| `Platform::set_initial_windowing` | trait-method | 408 |  | `Platform.SetInitialWindowing` | done |
| `Platform::request_windowing` | trait-method | 409 |  | `Platform.RequestWindowing` | done |
| `Platform::restart` | trait-method | 414 |  | `Platform.Restart` | done |
| `Platform::activate` | trait-method | 415 |  | `Platform.Activate` | done |
| `Platform::set_activation_policy` | trait-method | 417 |  | `Platform.SetActivationPolicy` | done |
| `Platform::hide` | trait-method | 418 |  | `Platform.Hide` | done |
| `Platform::hide_other_apps` | trait-method | 419 |  | `Platform.HideOtherApps` | done |
| `Platform::unhide_other_apps` | trait-method | 420 |  | `Platform.UnhideOtherApps` | done |
| `Platform::displays` | trait-method | 422 |  | `Platform.Displays` | done |
| `Platform::primary_display` | trait-method | 423 |  | `Platform.PrimaryDisplay` | done |
| `Platform::on_displays_changed` | trait-method | 428 |  | `Platform.OnDisplaysChanged` | done |
| `Platform::active_window` | trait-method | 429 |  | `Platform.ActiveWindow` | done |
| `Platform::window_stack` | trait-method | 430 |  | `Platform.WindowStack` | done |
| `Platform::is_screen_capture_supported` | trait-method | 434 |  | `Platform.IsScreenCaptureSupported` | done |
| `Platform::screen_capture_sources` | trait-method | 438 |  | `Platform.ScreenCaptureSources` | done |
| `Platform::open_window` | trait-method | 450 |  | `Platform.OpenWindow` | done |
| `Platform::window_appearance` | trait-method | 457 |  | `Platform.WindowAppearance` | done |
| `Platform::set_window_appearance` | trait-method | 467 |  | `Platform.SetWindowAppearance` | done |
| `Platform::button_layout` | trait-method | 470 |  | `Platform.ButtonLayout` | done |
| `Platform::text_scale_factor` | trait-method | 475 |  | `Platform.TextScaleFactor` | done |
| `Platform::on_text_scale_factor_change` | trait-method | 479 |  | `Platform.OnTextScaleFactorChange` | done |
| `Platform::open_url` | trait-method | 481 |  | `Platform.OpenURL` | done |
| `Platform::on_open_urls` | trait-method | 482 |  | `Platform.OnOpenURLs` | done |
| `Platform::register_url_scheme` | trait-method | 483 |  | `Platform.RegisterURLScheme` | done |
| `Platform::prompt_for_paths` | trait-method | 485 |  | `Platform.PromptForPaths` | done |
| `Platform::prompt_for_new_path` | trait-method | 489 |  | `Platform.PromptForNewPath` | done |
| `Platform::can_select_mixed_files_and_dirs` | trait-method | 494 |  | `Platform.CanSelectMixedFilesAndDirs` | done |
| `Platform::reveal_path` | trait-method | 495 |  | `Platform.RevealPath` | done |
| `Platform::open_with_system` | trait-method | 496 |  | `Platform.OpenWithSystem` | done |
| `Platform::on_quit` | trait-method | 498 |  | `Platform.OnQuit` | done |
| `Platform::on_reopen` | trait-method | 499 |  | `Platform.OnReopen` | done |
| `Platform::on_system_sleep` | trait-method | 501 |  | `Platform.OnSystemSleep` | done |
| `Platform::on_system_wake` | trait-method | 503 |  | `Platform.OnSystemWake` | done |
| `Platform::on_app_lifecycle` | trait-method | 514 |  | `Platform.OnAppLifecycle` | done |
| `Platform::on_memory_warning` | trait-method | 520 |  | `Platform.OnMemoryWarning` | done |
| `Platform::gestures` | trait-method | 525 |  | `Platform.Gestures` | done |
| `Platform::set_menus` | trait-method | 529 |  | `Platform.SetMenus` | done |
| `Platform::get_menus` | trait-method | 530 |  | `Platform.GetMenus` | done |
| `Platform::set_dock_menu` | trait-method | 534 |  | `Platform.SetDockMenu` | done |
| `Platform::perform_dock_menu_action` | trait-method | 535 |  | `Platform.PerformDockMenuAction` | done |
| `Platform::add_recent_document` | trait-method | 536 |  | `Platform.AddRecentDocument` | done |
| `Platform::update_jump_list` | trait-method | 537 |  | `Platform.UpdateJumpList` | done |
| `Platform::on_app_menu_action` | trait-method | 544 |  | `Platform.OnAppMenuAction` | done |
| `Platform::on_will_open_app_menu` | trait-method | 545 |  | `Platform.OnWillOpenAppMenu` | done |
| `Platform::on_validate_app_menu_command` | trait-method | 546 |  | `Platform.OnValidateAppMenuCommand` | done |
| `Platform::debug_state` | trait-method | 549 |  | `Platform.DebugState` | done |
| `Platform::thermal_state` | trait-method | 553 |  | `Platform.ThermalState` | done |
| `Platform::on_thermal_state_change` | trait-method | 554 |  | `Platform.OnThermalStateChange` | done |
| `Platform::prevent_idle_sleep` | trait-method | 555 |  | `Platform.PreventIdleSleep` | done |
| `Platform::power_mode` | trait-method | 559 |  | `Platform.PowerMode` | done |
| `Platform::on_power_mode_change` | trait-method | 563 |  | `Platform.OnPowerModeChange` | done |
| `Platform::set_app_identity` | trait-method | 571 |  | `Platform.SetAppIdentity` | done |
| `Platform::show_system_notification` | trait-method | 581 |  | `Platform.ShowSystemNotification` | done |
| `Platform::dismiss_system_notification` | trait-method | 592 |  | `Platform.DismissSystemNotification` | done |
| `Platform::on_system_notification_response` | trait-method | 602 |  | `Platform.OnSystemNotificationResponse` | done |
| `Platform::compositor_name` | trait-method | 609 |  | `Platform.CompositorName` | done |
| `Platform::graphical_environment` | trait-method | 614 |  | `Platform.GraphicalEnvironment` | done |
| `Platform::app_path` | trait-method | 617 |  | `Platform.AppPath` | done |
| `Platform::path_for_auxiliary_executable` | trait-method | 618 |  | `Platform.PathForAuxiliaryExecutable` | done |
| `Platform::set_cursor_style` | trait-method | 620 |  | `Platform.SetCursorStyle` | done |
| `Platform::hide_cursor_until_mouse_moves` | trait-method | 624 |  | `Platform.HideCursorUntilMouseMoves` | done |
| `Platform::is_cursor_visible` | trait-method | 627 |  | `Platform.IsCursorVisible` | done |
| `Platform::should_auto_hide_scrollbars` | trait-method | 629 |  | `Platform.ShouldAutoHideScrollbars` | done |
| `Platform::read_from_clipboard` | trait-method | 631 |  | `Platform.ReadFromClipboard` | done |
| `Platform::write_to_clipboard` | trait-method | 632 |  | `Platform.WriteToClipboard` | done |
| `Platform::read_from_clipboard_async` | trait-method | 641 |  | `Platform.ReadFromClipboardAsync` | done |
| `Platform::read_from_primary` | trait-method | 646 | linux,freebsd | `Platform.ReadFromPrimary` | done |
| `Platform::write_to_primary` | trait-method | 648 | linux,freebsd | `Platform.WriteToPrimary` | done |
| `Platform::read_from_find_pasteboard` | trait-method | 651 | macos | `Platform.ReadFromFindPasteboard` | done |
| `Platform::write_to_find_pasteboard` | trait-method | 653 | macos | `Platform.WriteToFindPasteboard` | done |
| `Platform::write_credentials` | trait-method | 655 |  | `Platform.WriteCredentials` | done |
| `Platform::write_credentials_if_not_exists` | trait-method | 661 | macos | `Platform.WriteCredentialsIfNotExists` | done |
| `Platform::read_credentials` | trait-method | 672 |  | `Platform.ReadCredentials` | done |
| `Platform::delete_credentials` | trait-method | 673 |  | `Platform.DeleteCredentials` | done |
| `Platform::keyboard_layout` | trait-method | 675 |  | `Platform.KeyboardLayout` | done |
| `Platform::keyboard_mapper` | trait-method | 676 |  | `Platform.KeyboardMapper` | done |
| `Platform::on_keyboard_layout_change` | trait-method | 677 |  | `Platform.OnKeyboardLayoutChange` | done |
| `PlatformDisplay` | trait | 681 |  | `PlatformDisplay` | done |
| `PlatformDisplay::id` | trait-method | 683 |  | `PlatformDisplay.ID` | done |
| `PlatformDisplay::uuid` | trait-method | 687 |  | `PlatformDisplay.UUID` | done |
| `PlatformDisplay::bounds` | trait-method | 690 |  | `PlatformDisplay.Bounds` | done |
| `PlatformDisplay::refresh_interval` | trait-method | 698 |  | `PlatformDisplay.RefreshInterval` | done |
| `PlatformDisplay::visible_bounds` | trait-method | 703 |  | `PlatformDisplay.VisibleBounds` | done |
| `PlatformDisplay::default_bounds` | trait-method | 708 |  | `PlatformDisplay.DefaultBounds` | done |
| `SystemNotification` | struct | 722 |  | `SystemNotification` | done |
| `SystemNotificationAction` | struct | 738 |  | `SystemNotificationAction` | done |
| `SystemNotificationResponse` | struct | 748 |  | `SystemNotificationResponse` | done |
| `PowerMode` | enum | 758 |  | `PowerMode` | done |
| `ThermalState` | enum | 772 |  | `ThermalState` | done |
| `SourceMetadata` | struct | 785 |  | `SourceMetadata` | done |
| `ScreenCaptureSource` | trait | 797 |  | `ScreenCaptureSource` | done |
| `ScreenCaptureSource::metadata` | trait-method | 799 |  | `ScreenCaptureSource.Metadata` | done |
| `ScreenCaptureSource::stream` | trait-method | 803 |  | `ScreenCaptureSource.Stream` | done |
| `ScreenCaptureStream` | trait | 811 |  | `ScreenCaptureStream` | done |
| `ScreenCaptureStream::metadata` | trait-method | 813 |  | `ScreenCaptureStream.Metadata` | done |
| `ScreenCaptureFrame` | struct | 817 |  | `ScreenCaptureFrame` | done |
| `DisplayId` | struct | 821 |  | `DisplayID` | done |
| `DisplayId::new` | assoc-fn | 825 |  | `NewDisplayID` | done |
| `ResizeEdge` | enum | 850 |  | `ResizeEdge` | done |
| `WindowDecorations` | enum | 871 |  | `WindowDecorations` | done |
| `Decorations` | enum | 881 |  | `Decorations` | done |
| `WindowControls` | struct | 894 |  | `WindowControls` | done |
| `WindowButton` | enum | 919 |  | `WindowButton` | done |
| `WindowButton::id` | method | 930 |  | `(WindowButton).ID` | done |
| `MAX_BUTTONS_PER_SIDE` | const | 949 |  | `MaxButtonsPerSide` | done |
| `WindowButtonLayout` | struct | 956 |  | `WindowButtonLayout` | done |
| `WindowButtonLayout::linux_default` | assoc-fn | 966 | linux,freebsd | `WindowButtonLayoutLinuxDefault` | done |
| `WindowButtonLayout::parse` | assoc-fn | 978 | linux,freebsd | `WindowButtonLayoutParse` | done |
| `WindowButtonLayout::format` | method | 1038 | linux,freebsd,test | `(WindowButtonLayout).format` | done |
| `Tiling` | struct | 1058 |  | `Tiling` | done |
| `Tiling::tiled` | assoc-fn | 1071 |  | `TilingTiled` | done |
| `Tiling::is_tiled` | method | 1081 |  | `(Tiling).IsTiled` | done |
| `A11yCallbacks` | struct | 1087 |  | `A11yCallbacks` | done |
| `FrameRequestSource` | enum | 1098 |  | `FrameRequestSource` | done |
| `RequestFrameOptions` | struct | 1108 |  | `RequestFrameOptions` | done |
| `PlatformFrameSignal` | struct | 1131 |  | `PlatformFrameSignal` | done |
| `PlatformFrameSignal::new` | assoc-fn | 1146 |  | `NewPlatformFrameSignal` | done |
| `PlatformFrameSignal::capture` | assoc-fn | 1159 |  | `PlatformFrameSignalCapture` | done |
| `PlatformFrameSignal::record` | method | 1170 |  | `(PlatformFrameSignal).Record` | done |
| `PlatformFrameSignal::take` | method | 1190 |  | `(PlatformFrameSignal).Take` | done |
| `AppLifecyclePhase` | enum | 1224 |  | `AppLifecyclePhase` | done |
| `WindowInsets` | struct | 1247 |  | `WindowInsets` | done |
| `WindowInsets::effective` | method | 1266 |  | `(WindowInsets).Effective` | done |
| `TextInputStateChange` | enum | 1278 |  | `TextInputStateChange` | done |
| `PlatformWindow` | trait | 1290 |  | `PlatformWindow` | done |
| `PlatformWindow::bounds` | trait-method | 1291 |  | `PlatformWindow.Bounds` | done |
| `PlatformWindow::is_maximized` | trait-method | 1292 |  | `PlatformWindow.IsMaximized` | done |
| `PlatformWindow::window_bounds` | trait-method | 1293 |  | `PlatformWindow.WindowBounds` | done |
| `PlatformWindow::content_size` | trait-method | 1294 |  | `PlatformWindow.ContentSize` | done |
| `PlatformWindow::visual_viewport_bounds` | trait-method | 1300 |  | `PlatformWindow.VisualViewportBounds` | done |
| `PlatformWindow::on_visual_viewport_changed` | trait-method | 1307 |  | `PlatformWindow.OnVisualViewportChanged` | done |
| `PlatformWindow::prepare_frame` | trait-method | 1312 |  | `PlatformWindow.PrepareFrame` | done |
| `PlatformWindow::resize` | trait-method | 1315 |  | `PlatformWindow.Resize` | done |
| `PlatformWindow::scale_factor` | trait-method | 1316 |  | `PlatformWindow.ScaleFactor` | done |
| `PlatformWindow::appearance` | trait-method | 1317 |  | `PlatformWindow.Appearance` | done |
| `PlatformWindow::display` | trait-method | 1318 |  | `PlatformWindow.Display` | done |
| `PlatformWindow::mouse_position` | trait-method | 1319 |  | `PlatformWindow.MousePosition` | done |
| `PlatformWindow::modifiers` | trait-method | 1320 |  | `PlatformWindow.Modifiers` | done |
| `PlatformWindow::capslock` | trait-method | 1321 |  | `PlatformWindow.Capslock` | done |
| `PlatformWindow::set_input_handler` | trait-method | 1322 |  | `PlatformWindow.SetInputHandler` | done |
| `PlatformWindow::take_input_handler` | trait-method | 1323 |  | `PlatformWindow.TakeInputHandler` | done |
| `PlatformWindow::set_text_input_configuration` | trait-method | 1328 |  | `PlatformWindow.SetTextInputConfiguration` | done |
| `PlatformWindow::prompt` | trait-method | 1329 |  | `PlatformWindow.Prompt` | done |
| `PlatformWindow::activate` | trait-method | 1336 |  | `PlatformWindow.Activate` | done |
| `PlatformWindow::request_attention` | trait-method | 1338 |  | `PlatformWindow.RequestAttention` | done |
| `PlatformWindow::is_active` | trait-method | 1339 |  | `PlatformWindow.IsActive` | done |
| `PlatformWindow::visibility` | trait-method | 1342 |  | `PlatformWindow.Visibility` | done |
| `PlatformWindow::is_hovered` | trait-method | 1343 |  | `PlatformWindow.IsHovered` | done |
| `PlatformWindow::background_appearance` | trait-method | 1344 |  | `PlatformWindow.BackgroundAppearance` | done |
| `PlatformWindow::set_title` | trait-method | 1345 |  | `PlatformWindow.SetTitle` | done |
| `PlatformWindow::set_background_appearance` | trait-method | 1346 |  | `PlatformWindow.SetBackgroundAppearance` | done |
| `PlatformWindow::minimize` | trait-method | 1347 |  | `PlatformWindow.Minimize` | done |
| `PlatformWindow::zoom` | trait-method | 1348 |  | `PlatformWindow.Zoom` | done |
| `PlatformWindow::toggle_fullscreen` | trait-method | 1349 |  | `PlatformWindow.ToggleFullscreen` | done |
| `PlatformWindow::is_fullscreen` | trait-method | 1350 |  | `PlatformWindow.IsFullscreen` | done |
| `PlatformWindow::debug_state` | trait-method | 1352 |  | `PlatformWindow.DebugState` | done |
| `PlatformWindow::frame_waker` | trait-method | 1355 |  | `PlatformWindow.FrameWaker` | done |
| `PlatformWindow::on_request_frame` | trait-method | 1358 |  | `PlatformWindow.OnRequestFrame` | done |
| `PlatformWindow::on_input` | trait-method | 1359 |  | `PlatformWindow.OnInput` | done |
| `PlatformWindow::on_active_status_change` | trait-method | 1360 |  | `PlatformWindow.OnActiveStatusChange` | done |
| `PlatformWindow::on_visibility_change` | trait-method | 1364 |  | `PlatformWindow.OnVisibilityChange` | done |
| `PlatformWindow::on_hover_status_change` | trait-method | 1365 |  | `PlatformWindow.OnHoverStatusChange` | done |
| `PlatformWindow::on_resize` | trait-method | 1366 |  | `PlatformWindow.OnResize` | done |
| `PlatformWindow::on_moved` | trait-method | 1367 |  | `PlatformWindow.OnMoved` | done |
| `PlatformWindow::on_display_changed` | trait-method | 1371 |  | `PlatformWindow.OnDisplayChanged` | done |
| `PlatformWindow::on_should_close` | trait-method | 1372 |  | `PlatformWindow.OnShouldClose` | done |
| `PlatformWindow::on_hit_test_window_control` | trait-method | 1373 |  | `PlatformWindow.OnHitTestWindowControl` | done |
| `PlatformWindow::on_close` | trait-method | 1374 |  | `PlatformWindow.OnClose` | done |
| `PlatformWindow::attach_native_view` | trait-method | 1385 |  | `PlatformWindow.AttachNativeView` | done |
| `PlatformWindow::set_focused_native_view` | trait-method | 1394 |  | `PlatformWindow.SetFocusedNativeView` | done |
| `PlatformWindow::on_hit_test_native_view` | trait-method | 1397 |  | `PlatformWindow.OnHitTestNativeView` | done |
| `PlatformWindow::on_native_view_focused` | trait-method | 1404 |  | `PlatformWindow.OnNativeViewFocused` | done |
| `PlatformWindow::on_key_down_before_native_view` | trait-method | 1410 |  | `PlatformWindow.OnKeyDownBeforeNativeView` | done |
| `PlatformWindow::on_appearance_changed` | trait-method | 1415 |  | `PlatformWindow.OnAppearanceChanged` | done |
| `PlatformWindow::on_button_layout_changed` | trait-method | 1416 |  | `PlatformWindow.OnButtonLayoutChanged` | done |
| `PlatformWindow::draw` | trait-method | 1417 |  | `PlatformWindow.Draw` | done |
| `PlatformWindow::schedule_frame` | trait-method | 1418 |  | `PlatformWindow.ScheduleFrame` | done |
| `PlatformWindow::sprite_atlas` | trait-method | 1419 |  | `PlatformWindow.SpriteAtlas` | done |
| `PlatformWindow::is_subpixel_rendering_supported` | trait-method | 1420 |  | `PlatformWindow.IsSubpixelRenderingSupported` | done |
| `PlatformWindow::get_title` | trait-method | 1423 |  | `PlatformWindow.GetTitle` | done |
| `PlatformWindow::tabbed_windows` | trait-method | 1426 |  | `PlatformWindow.TabbedWindows` | done |
| `PlatformWindow::tab_bar_visible` | trait-method | 1429 |  | `PlatformWindow.TabBarVisible` | done |
| `PlatformWindow::set_edited` | trait-method | 1432 |  | `PlatformWindow.SetEdited` | done |
| `PlatformWindow::set_document_path` | trait-method | 1433 |  | `PlatformWindow.SetDocumentPath` | done |
| `PlatformWindow::toggle_simple_fullscreen` | trait-method | 1434 |  | `PlatformWindow.ToggleSimpleFullscreen` | done |
| `PlatformWindow::is_simple_fullscreen` | trait-method | 1435 |  | `PlatformWindow.IsSimpleFullscreen` | done |
| `PlatformWindow::set_traffic_light_position` | trait-method | 1439 | macos | `PlatformWindow.SetTrafficLightPosition` | done |
| `PlatformWindow::set_native_titlebar_visibility` | trait-method | 1441 | macos | `PlatformWindow.SetNativeTitlebarVisibility` | done |
| `PlatformWindow::show_character_palette` | trait-method | 1442 |  | `PlatformWindow.ShowCharacterPalette` | done |
| `PlatformWindow::titlebar_double_click` | trait-method | 1443 |  | `PlatformWindow.TitlebarDoubleClick` | done |
| `PlatformWindow::on_move_tab_to_new_window` | trait-method | 1444 |  | `PlatformWindow.OnMoveTabToNewWindow` | done |
| `PlatformWindow::on_merge_all_windows` | trait-method | 1445 |  | `PlatformWindow.OnMergeAllWindows` | done |
| `PlatformWindow::on_select_previous_tab` | trait-method | 1446 |  | `PlatformWindow.OnSelectPreviousTab` | done |
| `PlatformWindow::on_select_next_tab` | trait-method | 1447 |  | `PlatformWindow.OnSelectNextTab` | done |
| `PlatformWindow::on_toggle_tab_bar` | trait-method | 1448 |  | `PlatformWindow.OnToggleTabBar` | done |
| `PlatformWindow::merge_all_windows` | trait-method | 1449 |  | `PlatformWindow.MergeAllWindows` | done |
| `PlatformWindow::move_tab_to_new_window` | trait-method | 1450 |  | `PlatformWindow.MoveTabToNewWindow` | done |
| `PlatformWindow::toggle_window_tab_overview` | trait-method | 1451 |  | `PlatformWindow.ToggleWindowTabOverview` | done |
| `PlatformWindow::set_tabbing_identifier` | trait-method | 1452 |  | `PlatformWindow.SetTabbingIdentifier` | done |
| `PlatformWindow::native_window_state` | trait-method | 1454 |  | `PlatformWindow.NativeWindowState` | done |
| `PlatformWindow::restore_native_window_state` | trait-method | 1457 |  | `PlatformWindow.RestoreNativeWindowState` | done |
| `PlatformWindow::get_raw_handle` | trait-method | 1460 | windows | `PlatformWindow.GetRawHandle` | done |
| `PlatformWindow::inner_window_bounds` | trait-method | 1463 |  | `PlatformWindow.InnerWindowBounds` | done |
| `PlatformWindow::request_decorations` | trait-method | 1466 |  | `PlatformWindow.RequestDecorations` | done |
| `PlatformWindow::show_window_menu` | trait-method | 1467 |  | `PlatformWindow.ShowWindowMenu` | done |
| `PlatformWindow::start_window_move` | trait-method | 1468 |  | `PlatformWindow.StartWindowMove` | done |
| `PlatformWindow::can_start_external_drag` | trait-method | 1469 |  | `PlatformWindow.CanStartExternalDrag` | done |
| `PlatformWindow::start_external_drag` | trait-method | 1472 |  | `PlatformWindow.StartExternalDrag` | done |
| `PlatformWindow::start_window_resize` | trait-method | 1475 |  | `PlatformWindow.StartWindowResize` | done |
| `PlatformWindow::set_exclusive_zone` | trait-method | 1476 |  | `PlatformWindow.SetExclusiveZone` | done |
| `PlatformWindow::set_exclusive_edge` | trait-method | 1478 | linux,feat:wayland | `PlatformWindow.SetExclusiveEdge` | done |
| `PlatformWindow::set_input_region` | trait-method | 1479 |  | `PlatformWindow.SetInputRegion` | done |
| `PlatformWindow::window_decorations` | trait-method | 1480 |  | `PlatformWindow.WindowDecorations` | done |
| `PlatformWindow::set_app_id` | trait-method | 1483 |  | `PlatformWindow.SetAppID` | done |
| `PlatformWindow::map_window` | trait-method | 1484 |  | `PlatformWindow.MapWindow` | done |
| `PlatformWindow::window_controls` | trait-method | 1487 |  | `PlatformWindow.WindowControls` | done |
| `PlatformWindow::set_client_inset` | trait-method | 1490 |  | `PlatformWindow.SetClientInset` | done |
| `PlatformWindow::set_client_corner_radii` | trait-method | 1491 |  | `PlatformWindow.SetClientCornerRadii` | done |
| `PlatformWindow::gpu_specs` | trait-method | 1492 |  | `PlatformWindow.GPUSpecs` | done |
| `PlatformWindow::update_ime_position` | trait-method | 1494 |  | `PlatformWindow.UpdateIMEPosition` | done |
| `PlatformWindow::insets` | trait-method | 1500 |  | `PlatformWindow.Insets` | done |
| `PlatformWindow::on_insets_changed` | trait-method | 1509 |  | `PlatformWindow.OnInsetsChanged` | done |
| `PlatformWindow::set_back_handler` | trait-method | 1513 |  | `PlatformWindow.SetBackHandler` | done |
| `PlatformWindow::set_back_enabled` | trait-method | 1517 |  | `PlatformWindow.SetBackEnabled` | done |
| `PlatformWindow::show_soft_keyboard` | trait-method | 1520 |  | `PlatformWindow.ShowSoftKeyboard` | done |
| `PlatformWindow::hide_soft_keyboard` | trait-method | 1523 |  | `PlatformWindow.HideSoftKeyboard` | done |
| `PlatformWindow::text_input_state_changed` | trait-method | 1526 |  | `PlatformWindow.TextInputStateChanged` | done |
| `PlatformWindow::play_system_bell` | trait-method | 1528 |  | `PlatformWindow.PlaySystemBell` | done |
| `PlatformWindow::a11y_init` | trait-method | 1531 |  | `PlatformWindow.A11yInit` | done |
| `PlatformWindow::a11y_tree_update` | trait-method | 1534 |  | `PlatformWindow.A11yTreeUpdate` | done |
| `PlatformWindow::a11y_update_window_bounds` | trait-method | 1537 |  | `PlatformWindow.A11yUpdateWindowBounds` | done |
| `PlatformWindow::as_test` | trait-method | 1540 | feat:test-support,feat:bench-support | `PlatformWindow.AsTest` | done |
| `PlatformWindow::render_to_image` | trait-method | 1548 | feat:test-support | `PlatformWindow.RenderToImage` | done |
| `PlatformHeadlessRenderer` | trait | 1555 | feat:test-support,feat:bench-support | `PlatformHeadlessRenderer` | done |
| `PlatformHeadlessRenderer::render_scene_to_image` | trait-method | 1557 | feat:test-support,feat:bench-support | `PlatformHeadlessRenderer.RenderSceneToImage` | done |
| `PlatformHeadlessRenderer::render_scene` | trait-method | 1568 | feat:test-support,feat:bench-support | `PlatformHeadlessRenderer.RenderScene` | done |
| `PlatformHeadlessRenderer::sprite_atlas` | trait-method | 1571 | feat:test-support,feat:bench-support | `PlatformHeadlessRenderer.SpriteAtlas` | done |
| `RunnableVariant` | type | 1577 |  | `RunnableVariant` | done |
| `TimerResolutionGuard` | type | 1580 |  | `TimerResolutionGuard` | done |
| `TasksIncluded` | enum | 1583 |  | `TasksIncluded` | done |
| `PlatformDispatcher` | trait | 1591 |  | `PlatformDispatcher` | done |
| `PlatformDispatcher::is_main_thread` | trait-method | 1592 |  | `PlatformDispatcher.IsMainThread` | done |
| `PlatformDispatcher::dispatch` | trait-method | 1593 |  | `PlatformDispatcher.Dispatch` | done |
| `PlatformDispatcher::dispatch_on_main_thread` | trait-method | 1594 |  | `PlatformDispatcher.DispatchOnMainThread` | done |
| `PlatformDispatcher::dispatch_after` | trait-method | 1595 |  | `PlatformDispatcher.DispatchAfter` | done |
| `PlatformDispatcher::dispatch_on_main_thread_when_idle` | trait-method | 1597 |  | `PlatformDispatcher.DispatchOnMainThreadWhenIdle` | done |
| `PlatformDispatcher::idle_time_remaining` | trait-method | 1606 |  | `PlatformDispatcher.IdleTimeRemaining` | done |
| `PlatformDispatcher::spawn_realtime` | trait-method | 1610 |  | `PlatformDispatcher.SpawnRealtime` | done |
| `PlatformDispatcher::now` | trait-method | 1612 |  | `PlatformDispatcher.Now` | done |
| `PlatformDispatcher::increase_timer_resolution` | trait-method | 1616 |  | `PlatformDispatcher.IncreaseTimerResolution` | done |
| `PlatformDispatcher::prevent_app_nap` | trait-method | 1620 |  | `PlatformDispatcher.PreventAppNap` | done |
| `PlatformDispatcher::as_test` | trait-method | 1625 | feat:test-support,feat:bench-support | `PlatformDispatcher.AsTest` | done |
| `PlatformDispatcher::as_threaded` | trait-method | 1632 | feat:test-support,feat:bench-support | `PlatformDispatcher.AsThreaded` | done |
| `PlatformTextSystem` | trait | 1638 |  | `PlatformTextSystem` | done |
| `PlatformTextSystem::add_fonts` | trait-method | 1639 |  | `PlatformTextSystem.AddFonts` | done |
| `PlatformTextSystem::set_missing_glyph_sink` | trait-method | 1641 |  | `PlatformTextSystem.SetMissingGlyphSink` | done |
| `PlatformTextSystem::all_font_names` | trait-method | 1643 |  | `PlatformTextSystem.AllFontNames` | done |
| `PlatformTextSystem::font_id` | trait-method | 1645 |  | `PlatformTextSystem.FontID` | done |
| `PlatformTextSystem::prewarm_fonts` | trait-method | 1647 |  | `PlatformTextSystem.PrewarmFonts` | done |
| `PlatformTextSystem::font_metrics` | trait-method | 1649 |  | `PlatformTextSystem.FontMetrics` | done |
| `PlatformTextSystem::typographic_bounds` | trait-method | 1651 |  | `PlatformTextSystem.TypographicBounds` | done |
| `PlatformTextSystem::advance` | trait-method | 1653 |  | `PlatformTextSystem.Advance` | done |
| `PlatformTextSystem::glyph_for_char` | trait-method | 1655 |  | `PlatformTextSystem.GlyphForChar` | done |
| `PlatformTextSystem::glyph_raster_bounds` | trait-method | 1657 |  | `PlatformTextSystem.GlyphRasterBounds` | done |
| `PlatformTextSystem::rasterize_glyph` | trait-method | 1659 |  | `PlatformTextSystem.RasterizeGlyph` | done |
| `PlatformTextSystem::layout_line` | trait-method | 1665 |  | `PlatformTextSystem.LayoutLine` | done |
| `PlatformTextSystem::recommended_rendering_mode` | trait-method | 1667 |  | `PlatformTextSystem.RecommendedRenderingMode` | done |
| `PlatformTextSystem::glyph_dilation_for_color` | trait-method | 1670 |  | `PlatformTextSystem.GlyphDilationForColor` | done |
| `NoopTextSystem` | struct | 1676 |  | `NoopTextSystem` | done |
| `NoopTextSystem::new` | assoc-fn | 1681 |  | `NewNoopTextSystem` | done |
| `get_gamma_correction_ratios` | fn | 1810 |  | `GetGammaCorrectionRatios` | done |
| `AtlasKey` | enum | 1843 |  | `AtlasKey` | done |
| `AtlasKey::texture_kind` | method | 1851 |  | `(AtlasKey).TextureKind` | done |
| `PlatformAtlas` | trait | 1887 |  | `PlatformAtlas` | done |
| `PlatformAtlas::get_or_insert_with` | trait-method | 1889 |  | `PlatformAtlas.GetOrInsertWith` | done |
| `PlatformAtlas::remove` | trait-method | 1894 |  | `PlatformAtlas.Remove` | done |
| `PlatformAtlas::contains` | trait-method | 1897 | feat:test-support,feat:bench-support | `PlatformAtlas.Contains` | done |
| `AtlasBackend` | trait | 1903 |  | `AtlasBackend` | done |
| `AtlasBackend::insert` | trait-method | 1904 |  | `AtlasBackend.Insert` | done |
| `AtlasBackend::remove` | trait-method | 1911 |  | `AtlasBackend.Remove` | done |
| `AtlasState` | struct | 1915 |  | `AtlasState` | done |
| `AtlasState::new` | assoc-fn | 1921 |  | `NewAtlasState` | done |
| `AtlasState::contains` | method | 1928 |  | `(AtlasState).Contains` | done |
| `AtlasState::clear` | method | 1932 |  | `(AtlasState).Clear` | done |
| `AtlasState::get_or_insert_with` | method | 1945 |  | `(AtlasState).GetOrInsertWith` | done |
| `AtlasState::remove` | method | 1966 |  | `(AtlasState).Remove` | done |
| `HeadlessAtlas` | struct | 1977 |  | `HeadlessAtlas` | done |
| `HeadlessAtlasBackend` | struct | 1981 |  | `HeadlessAtlasBackend` | done |
| `AtlasTextureList` | struct | 2033 |  | `AtlasTextureList` | done |
| `AtlasTextureList::drain` | method | 2057 |  | `(AtlasTextureList).Drain` | done |
| `AtlasTextureList::iter_mut` | method | 2063 |  | `(AtlasTextureList).IterMut` | done |
| `AtlasTile` | struct | 2071 |  | `AtlasTile` | done |
| `AtlasTextureId` | struct | 2085 |  | `AtlasTextureID` | done |
| `AtlasTextureKind` | enum | 2103 |  | `AtlasTextureKind` | done |
| `TileId` | struct | 2112 |  | `TileID` | done |
| `PlatformInputHandler` | struct | 2127 |  | `PlatformInputHandler` | done |
| `PlatformInputHandler::new` | assoc-fn | 2142 |  | `NewPlatformInputHandler` | done |
| `PlatformInputHandler::focus_id` | method | 2159 |  | `(PlatformInputHandler).FocusID` | done |
| `PlatformInputHandler::selected_text_range` | method | 2163 |  | `(PlatformInputHandler).SelectedTextRange` | done |
| `PlatformInputHandler::marked_text_range` | method | 2174 |  | `(PlatformInputHandler).MarkedTextRange` | done |
| `PlatformInputHandler::text_for_range` | method | 2185 |  | `(PlatformInputHandler).TextForRange` | done |
| `PlatformInputHandler::replace_text_in_range` | method | 2199 |  | `(PlatformInputHandler).ReplaceTextInRange` | done |
| `PlatformInputHandler::replace_and_mark_text_in_range` | method | 2208 |  | `(PlatformInputHandler).ReplaceAndMarkTextInRange` | done |
| `PlatformInputHandler::unmark_text` | method | 2228 |  | `(PlatformInputHandler).UnmarkText` | done |
| `PlatformInputHandler::paste` | method | 2234 |  | `(PlatformInputHandler).Paste` | done |
| `PlatformInputHandler::bounds_for_range` | method | 2240 |  | `(PlatformInputHandler).BoundsForRange` | done |
| `PlatformInputHandler::selection_bounds_for_range` | method | 2247 |  | `(PlatformInputHandler).SelectionBoundsForRange` | done |
| `PlatformInputHandler::apple_press_and_hold_enabled` | method | 2257 |  | `(PlatformInputHandler).ApplePressAndHoldEnabled` | done |
| `PlatformInputHandler::dispatch_input` | method | 2267 |  | `(PlatformInputHandler).DispatchInput` | done |
| `PlatformInputHandler::compute_ime_candidate_bounds` | assoc-fn | 2271 |  | `PlatformInputHandlerComputeIMECandidateBounds` | done |
| `PlatformInputHandler::selected_bounds` | method | 2306 |  | `(PlatformInputHandler).SelectedBounds` | done |
| `PlatformInputHandler::ime_candidate_bounds` | method | 2314 |  | `(PlatformInputHandler).IMECandidateBounds` | done |
| `PlatformInputHandler::character_index_for_point` | method | 2323 |  | `(PlatformInputHandler).CharacterIndexForPoint` | done |
| `PlatformInputHandler::set_selected_text_range` | method | 2331 |  | `(PlatformInputHandler).SetSelectedTextRange` | done |
| `PlatformInputHandler::element_bounds` | method | 2341 |  | `(PlatformInputHandler).ElementBounds` | done |
| `PlatformInputHandler::text_length_utf16` | method | 2349 |  | `(PlatformInputHandler).TextLengthUTF16` | done |
| `PlatformInputHandler::accepts_text_input` | method | 2357 |  | `(PlatformInputHandler).AcceptsTextInput` | done |
| `PlatformInputHandler::query_accepts_text_input` | method | 2362 |  | `(PlatformInputHandler).QueryAcceptsTextInput` | done |
| `PlatformInputHandler::query_accepts_focused_text_input` | method | 2372 |  | `(PlatformInputHandler).QueryAcceptsFocusedTextInput` | done |
| `PlatformInputHandler::query_prefers_ime_for_printable_keys` | method | 2388 |  | `(PlatformInputHandler).QueryPrefersIMEForPrintableKeys` | done |
| `PlatformInputHandler::text_input_configuration` | method | 2399 |  | `(PlatformInputHandler).TextInputConfiguration` | done |
| `PlatformInputHandler::text_input_editable_range` | method | 2408 |  | `(PlatformInputHandler).TextInputEditableRange` | done |
| `UTF16Selection` | struct | 2419 |  | `UTF16Selection` | done |
| `InputHandler` | trait | 2432 |  | `InputHandler` | done |
| `InputHandler::selected_text_range` | trait-method | 2437 |  | `InputHandler.SelectedTextRange` | done |
| `InputHandler::marked_text_range` | trait-method | 2448 |  | `InputHandler.MarkedTextRange` | done |
| `InputHandler::text_for_range` | trait-method | 2454 |  | `InputHandler.TextForRange` | done |
| `InputHandler::replace_text_in_range` | trait-method | 2466 |  | `InputHandler.ReplaceTextInRange` | done |
| `InputHandler::replace_and_mark_text_in_range` | trait-method | 2480 |  | `InputHandler.ReplaceAndMarkTextInRange` | done |
| `InputHandler::unmark_text` | trait-method | 2491 |  | `InputHandler.UnmarkText` | done |
| `InputHandler::paste` | trait-method | 2499 |  | `InputHandler.Paste` | done |
| `InputHandler::bounds_for_range` | trait-method | 2509 |  | `InputHandler.BoundsForRange` | done |
| `InputHandler::selection_bounds_for_range` | trait-method | 2522 |  | `InputHandler.SelectionBoundsForRange` | done |
| `InputHandler::character_index_for_point` | trait-method | 2534 |  | `InputHandler.CharacterIndexForPoint` | done |
| `InputHandler::set_selected_text_range` | trait-method | 2550 |  | `InputHandler.SetSelectedTextRange` | done |
| `InputHandler::element_bounds` | trait-method | 2564 |  | `InputHandler.ElementBounds` | done |
| `InputHandler::text_length_utf16` | trait-method | 2569 |  | `InputHandler.TextLengthUTF16` | done |
| `InputHandler::apple_press_and_hold_enabled` | trait-method | 2578 |  | `InputHandler.ApplePressAndHoldEnabled` | done |
| `InputHandler::accepts_text_input` | trait-method | 2583 |  | `InputHandler.AcceptsTextInput` | done |
| `InputHandler::text_input_editable_range` | trait-method | 2597 |  | `InputHandler.TextInputEditableRange` | done |
| `InputHandler::prefers_ime_for_printable_keys` | trait-method | 2613 |  | `InputHandler.PrefersIMEForPrintableKeys` | done |
| `InputHandler::text_input_configuration` | trait-method | 2622 |  | `InputHandler.TextInputConfiguration` | done |
| `TextInputConfiguration` | struct | 2641 |  | `TextInputConfiguration` | done |
| `Autocapitalize` | enum | 2654 |  | `Autocapitalize` | done |
| `TextInputAction` | enum | 2676 |  | `TextInputAction` | done |
| `WindowOptions` | struct | 2698 |  | `WindowOptions` | done |
| `WindowParams` | struct | 2795 |  | `WindowParams` | done |
| `WindowBounds` | enum | 2855 |  | `WindowBounds` | done |
| `WindowBounds::get_bounds` | method | 2874 |  | `(WindowBounds).GetBounds` | done |
| `WindowBounds::centered` | assoc-fn | 2884 |  | `WindowBoundsCentered` | done |
| `TitlebarOptions` | struct | 2922 |  | `TitlebarOptions` | done |
| `WindowKind` | enum | 2936 |  | `WindowKind` | done |
| `WindowAppearance` | enum | 2970 |  | `WindowAppearance` | done |
| `WindowBackgroundAppearance` | enum | 2996 |  | `WindowBackgroundAppearance` | done |
| `TextRenderingMode` | enum | 3020 |  | `TextRenderingMode` | done |
| `PathPromptOptions` | struct | 3032 |  | `PathPromptOptions` | done |
| `PromptLevel` | enum | 3045 |  | `PromptLevel` | done |
| `PromptButton` | enum | 3058 |  | `PromptButton` | done |
| `PromptButton::new` | assoc-fn | 3069 |  | `NewPromptButton` | done |
| `PromptButton::ok` | assoc-fn | 3074 |  | `PromptButtonOk` | done |
| `PromptButton::cancel` | assoc-fn | 3079 |  | `PromptButtonCancel` | done |
| `PromptButton::is_cancel` | method | 3085 |  | `(PromptButton).IsCancel` | done |
| `PromptButton::label` | method | 3090 |  | `(PromptButton).Label` | done |
| `CursorStyle` | enum | 3113 |  | `CursorStyle` | done |
| `ClipboardItem` | struct | 3201 |  | `ClipboardItem` | done |
| `ClipboardReadError` | enum | 3211 |  | `ClipboardReadError` | done |
| `ClipboardEntry` | enum | 3242 |  | `ClipboardEntry` | done |
| `ClipboardItem::new_string` | assoc-fn | 3253 |  | `NewClipboardItemString` | done |
| `ClipboardItem::new_string_with_metadata` | assoc-fn | 3260 |  | `NewClipboardItemStringWithMetadata` | done |
| `ClipboardItem::new_string_with_json_metadata` | assoc-fn | 3270 |  | `NewClipboardItemStringWithJSONMetadata` | done |
| `ClipboardItem::new_image` | assoc-fn | 3279 |  | `NewClipboardItemImage` | done |
| `ClipboardItem::text` | method | 3287 |  | `(ClipboardItem).Text` | done |
| `ClipboardItem::metadata` | method | 3316 |  | `(ClipboardItem).Metadata` | done |
| `ClipboardItem::entries` | method | 3326 |  | `(ClipboardItem).Entries` | n/a:D-GETTER |
| `ClipboardItem::into_entries` | method | 3331 |  | `(ClipboardItem).IntoEntries` | done |
| `ImageFormat` | enum | 3376 |  | `ImageFormat` | done |
| `ImageFormat::mime_type` | method | 3402 |  | `(ImageFormat).MimeType` | done |
| `ImageFormat::extension` | method | 3417 |  | `(ImageFormat).Extension` | done |
| `ImageFormat::from_mime_type` | assoc-fn | 3432 |  | `ImageFormatFromMimeType` | done |
| `Image` | struct | 3453 |  | `Image` | done |
| `Image::empty` | assoc-fn | 3541 |  | `ImageEmpty` | done |
| `Image::from_bytes` | assoc-fn | 3546 |  | `ImageFromBytes` | done |
| `Image::id` | method | 3555 |  | `(Image).ID` | n/a:D-GETTER |
| `Image::use_render_image` | method | 3560 |  | `(Image).UseRenderImage` | done |
| `Image::get_render_image` | method | 3571 |  | `(Image).GetRenderImage` | done |
| `Image::remove_asset` | method | 3582 |  | `(Image).RemoveAsset` | done |
| `Image::is_asset_cached` | method | 3589 | feat:test-support | `(Image).IsAssetCached` | done |
| `Image::to_image_data` | method | 3594 |  | `(Image).ToImageData` | dev:L-JPEG |
| `Image::format` | method | 3615 |  | `(Image).Format` | n/a:D-GETTER |
| `Image::bytes` | method | 3620 |  | `(Image).Bytes` | n/a:D-GETTER |
| `ClipboardString` | struct | 3627 |  | `ClipboardString` | done |
| `ClipboardString::new` | assoc-fn | 3636 |  | `NewClipboardString` | done |
| `ClipboardString::with_json_metadata` | method | 3645 |  | `(ClipboardString).WithJSONMetadata` | done |
| `ClipboardString::text` | method | 3651 |  | `(ClipboardString).Text` | n/a:D-GETTER |
| `ClipboardString::into_text` | method | 3656 |  | `(ClipboardString).IntoText` | done |
| `ClipboardString::metadata_json` | method | 3661 |  | `(ClipboardString).MetadataJSON` | done |
| `ClipboardString::text_hash` | assoc-fn | 3672 |  | `ClipboardStringTextHash` | done |
| `decode_static_image` | fn | 3462 |  | `decodeStaticImage` | dev:L-JPEG |
| `decode_static_image_from_decoder` | fn | 3472 |  | `decodeStaticImageFromDecoder` | dev:L-JPEG |
| `decode_animated_frames` | fn | 3489 |  | `decodeAnimatedFrames` | done |
| `decode_gif_image` | fn | 3517 |  | `decodeGifImage` | dev:D-IMGDECODE |
| `decode_webp_image` | fn | 3522 |  | `decodeWebpImage` | dev:D-IMGDECODE |
| `image::Frame` (decoded frame) | extern | 57 |  | `ImageFrame` | dev:D-IMAGEFRAME |
| `PlatformInputHandler::with_focus` | method | 2150 |  | `(PlatformInputHandler).withFocus` | done |
| `PlatformInputHandler::unmark_text_during_update` | method | 2263 |  | `(PlatformInputHandler).unmarkTextDuringUpdate` | done |
| `impl From<&str> for PromptButton` | impl | 3099 |  | `PromptButtonFromStr` | done |
| `impl From<ClipboardString> for ClipboardEntry` | impl | 3336 |  | `ClipboardEntryFromClipboardString` | done |
| `impl From<String> for ClipboardEntry` | impl | 3342 |  | `ClipboardEntryFromString` | done |
| `impl From<Image> for ClipboardEntry` | impl | 3348 |  | `ClipboardEntryFromImage` | done |
| `impl From<ClipboardEntry> for ClipboardItem` | impl | 3354 |  | `ClipboardItemFromEntry` | done |
| `impl From<String> for ClipboardItem` | impl | 3362 |  | `ClipboardItemFromString` | done |
| `impl From<Image> for ClipboardItem` | impl | 3368 |  | `ClipboardItemFromImage` | done |
| `impl From<String> for ClipboardString` | impl | 3679 |  | `ClipboardStringFromString` | done |
| `impl Display for ClipboardReadError` | impl | 3224 |  | `(ClipboardReadError).Error` | done |
| `impl Default for WindowOptions` | impl | 2889 |  | `DefaultWindowOptions` | done |
| `CursorStyle` Serialize derive | impl | 3111 |  | `(CursorStyle).MarshalJSONTo` | done |
| `CursorStyle` Deserialize derive | impl | 3111 |  | `(CursorStyle).UnmarshalJSONFrom` | done |
| `CursorStyle` JsonSchema derive | impl | 3111 |  | `(CursorStyle).JSONSchemaEnum` | done |
| `CursorStyle` EnumIter derive | impl | 3111 |  | `CursorStyleIter` | done |
| `ImageFormat` EnumIter derive | impl | 3374 |  | `ImageFormatIter` | done |
| `InputHandler` provided methods | impl | 2432 |  | `InputHandlerDefaults` | dev:D-DEFAULTS |
| `InputHandler` provided methods (binding) | impl | 2432 |  | `BindInputHandlerDefaults` | dev:D-DEFAULTS |
| `std::ops::Range<usize>` in input handler signatures | extern | 2174 |  | `Range` | dev:D-RANGE |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_image_to_image_data_applies_exif_orientation` | test | 3694 |  | `TestPlatformImageTests_ImageToImageDataAppliesExifOrientation` | done |
| `test_svg_image_to_image_data_converts_to_bgra` | test | 3710 |  | `TestPlatformImageTests_SvgImageToImageDataConvertsToBgra` | done |
| `test_animated_webp_image_to_image_data_decodes_frames` | test | 3728 |  | `TestPlatformImageTests_AnimatedWebpImageToImageDataDecodesFrames` | done |
| `only_successful_inserts_are_cached` | test | 3813 |  | `TestPlatformAtlasTests_OnlySuccessfulInsertsAreCached` | done |
| `remove_and_clear_invalidate_keys` | test | 3850 |  | `TestPlatformAtlasTests_RemoveAndClearInvalidateKeys` | done |
| `coalesced_signals_retain_the_earliest_until_drained` | test | 3882 | feat:profiler | `TestPlatformFrameSignalTests_CoalescedSignalsRetainTheEarliestUntilDrained` | done |
| `disabled_profiling_does_not_capture_or_store_timestamps` | test | 3910 | feat:profiler | `TestPlatformFrameSignalTests_DisabledProfilingDoesNotCaptureOrStoreTimestamps` | done |
| `test_refresh_interval_from_hz` | test | 3927 | linux,freebsd | `TestPlatform_RefreshIntervalFromHz` | done |
| `test_window_button_layout_parse_standard` | test | 3942 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseStandard` | done |
| `test_window_button_layout_parse_right_only` | test | 3956 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseRightOnly` | done |
| `test_window_button_layout_parse_left_only` | test | 3970 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseLeftOnly` | done |
| `test_window_button_layout_parse_with_whitespace` | test | 3984 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseWithWhitespace` | done |
| `test_window_button_layout_parse_empty` | test | 3998 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseEmpty` | done |
| `test_window_button_layout_parse_intentionally_empty` | test | 4005 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseIntentionallyEmpty` | done |
| `test_window_button_layout_parse_invalid_buttons` | test | 4012 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseInvalidButtons` | done |
| `test_window_button_layout_parse_deduplicates_same_side_buttons` | test | 4026 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseDeduplicatesSameSideButtons` | done |
| `test_window_button_layout_parse_deduplicates_buttons_across_sides` | test | 4040 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseDeduplicatesButtonsAcrossSides` | done |
| `test_window_button_layout_parse_gnome_style` | test | 4065 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseGnomeStyle` | done |
| `test_window_button_layout_parse_elementary_style` | test | 4072 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseElementaryStyle` | done |
| `test_window_button_layout_round_trip` | test | 4079 | linux,freebsd | `TestPlatform_WindowButtonLayoutRoundTrip` | done |
| `test_window_button_layout_linux_default` | test | 4096 | linux,freebsd | `TestPlatform_WindowButtonLayoutLinuxDefault` | done |
| `test_window_button_layout_parse_all_invalid` | test | 4113 | linux,freebsd | `TestPlatform_WindowButtonLayoutParseAllInvalid` | done |

## gpui/src/platform/app_menu.rs

Go target: `platform_app_menu.go` (package `gpui`), tests in `platform_app_menu_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Menu` | struct | 4 |  | `Menu` | done |
| `Menu::new` | assoc-fn | 17 |  | `NewMenu` | done |
| `Menu::items` | method | 26 |  | `(Menu).WithItems` | dev:D-WITH |
| `Menu::disabled` | method | 32 |  | `(Menu).WithDisabled` | dev:D-WITH |
| `Menu::owned` | method | 38 |  | `(Menu).Owned` | done |
| `OsMenu` | struct | 50 |  | `OsMenu` | done |
| `OsMenu::owned` | method | 60 |  | `(OsMenu).Owned` | done |
| `SystemMenuType` | enum | 70 |  | `SystemMenuType` | done |
| `MenuItem` | enum | 76 |  | `MenuItem` | done |
| `MenuItem::separator` | assoc-fn | 108 |  | `NewMenuItemSeparator` | dev:D-NAME |
| `MenuItem::submenu` | assoc-fn | 113 |  | `NewMenuItemSubmenu` | dev:D-NAME |
| `MenuItem::os_submenu` | assoc-fn | 118 |  | `MenuItemOsSubmenu` | done |
| `MenuItem::action` | assoc-fn | 126 |  | `NewMenuItemAction` | dev:D-NAME |
| `MenuItem::os_action` | assoc-fn | 137 |  | `MenuItemOsAction` | done |
| `MenuItem::owned` | method | 152 |  | `(MenuItem).Owned` | done |
| `MenuItem::checked` | method | 176 |  | `(MenuItem).WithChecked` | dev:D-WITH |
| `MenuItem::is_checked` | method | 190 |  | `(MenuItem).IsChecked` | done |
| `MenuItem::disabled` | method | 198 |  | `(MenuItem).WithDisabled` | dev:D-WITH |
| `MenuItem::is_disabled` | method | 215 |  | `(MenuItem).IsDisabled` | done |
| `OwnedOsMenu` | struct | 228 |  | `OwnedOsMenu` | done |
| `OwnedMenu` | struct | 238 |  | `OwnedMenu` | done |
| `OwnedMenuItem` | enum | 250 |  | `OwnedMenuItem` | done |
| `OsAction` | enum | 311 |  | `OsAction` | done |
| `init_app_menus` | fn | 331 |  | `initAppMenus` | done |
| `impl Clone for OwnedMenuItem` | impl | 280 |  | `(OwnedMenuItem).Clone` | done |
| `OwnedMenu` Clone derive | impl | 237 |  | `(OwnedMenu).Clone` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_menu` | test | 366 |  | `TestAppMenu_Menu` | done |
| `test_menu_item_builder` | test | 380 |  | `TestAppMenu_MenuItemBuilder` | done |

## gpui/src/platform/layer_shell.rs

Go target: `platform_layer_shell.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Layer` | enum | 9 |  | `Layer` | done |
| `Anchor` | struct | 26 | linux,feat:wayland | `LayerShellAnchor` | dev:D-SUBMOD |
| `KeyboardInteractivity` | enum | 43 |  | `KeyboardInteractivity` | done |
| `LayerShellOptions` | struct | 59 |  | `LayerShellOptions` | done |
| `LayerShellNotSupportedError` | struct | 83 |  | `LayerShellNotSupportedError` | done |

## gpui/src/platform/platform_info.rs

Go target: `platform_platform_info.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PlatformInfo` | struct | 9 |  | `PlatformInfo` | done |
| `OperatingSystem` | enum | 18 |  | `OperatingSystem` | done |
| `PlatformTarget` | enum | 37 |  | `PlatformTarget` | done |
| `PlatformInfo::is_mac` | method | 46 |  | `(PlatformInfo).IsMac` | done |

## gpui/src/platform/popup.rs

Go target: `platform_popup.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PopupOptions` | struct | 16 |  | `PopupOptions` | done |
| `PopupAnchor` | enum | 57 |  | `PopupAnchor` | done |
| `PopupGravity` | enum | 84 |  | `PopupGravity` | done |
| `PopupNotSupportedError` | struct | 134 |  | `PopupNotSupportedError` | done |
| `PopupConstraintAdjustment` | bitflags | 111 |  | `PopupConstraintAdjustment` | done |

## gpui/src/platform/scap_screen_capture.rs

Go target: `platform_scap_screen_capture.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `scap_screen_sources` | fn | 18 |  | `ScapScreenSources` | blocked:M3-root-capture |

## gpui/src/platform/threaded_dispatcher.rs

Go target: `platform_threaded_dispatcher.go` (package `gpui`), tests in `platform_threaded_dispatcher_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ThreadedDispatcher` | struct | 28 |  | `ThreadedDispatcher` | done |
| `ThreadedDispatcher::new` | assoc-fn | 125 |  | `NewThreadedDispatcher` | done |
| `ThreadedDispatcher::run_until_idle` | method | 220 |  | `(ThreadedDispatcher).RunUntilIdle` | done |
| `ThreadedDispatcher::run_ready_main_tasks` | method | 307 |  | `(ThreadedDispatcher).RunReadyMainTasks` | done |
| `ThreadedDispatcher::cancel_pending_timers` | method | 373 |  | `(ThreadedDispatcher).CancelPendingTimers` | done |
| `ThreadedDispatcher::debug_state` | method | 387 |  | `(ThreadedDispatcher).DebugState` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `is_idle_tracks_queued_work_but_ignores_undue_timers` | test | 498 |  | `TestThreadedDispatcher_IsIdleTracksQueuedWorkButIgnoresUndueTimers` | done |
| `run_ready_main_tasks_does_not_wait_for_background_handoffs` | test | 524 |  | `TestThreadedDispatcher_RunReadyMainTasksDoesNotWaitForBackgroundHandoffs` | done |
| `run_until_idle_completes_background_to_main_handoffs` | test | 556 |  | `TestThreadedDispatcher_RunUntilIdleCompletesBackgroundToMainHandoffs` | done |
| `run_until_waits_for_untracked_external_wakes` | test | 585 |  | `TestThreadedDispatcher_RunUntilWaitsForUntrackedExternalWakes` | done |
| `run_until_returns_at_readiness_despite_requeuing_main_work` | test | 615 |  | `TestThreadedDispatcher_RunUntilReturnsAtReadinessDespiteRequeuingMainWork` | done |
| `run_ready_main_tasks_advances_requeuing_work_one_batch_per_call` | test | 671 |  | `TestThreadedDispatcher_RunReadyMainTasksAdvancesRequeuingWorkOneBatchPerCall` | done |
| `timers_fire_in_real_time` | test | 697 |  | `TestThreadedDispatcher_TimersFireInRealTime` | done |
| `cancel_pending_timers_wakes_waiters_without_waiting_for_deadline` | test | 721 |  | `TestThreadedDispatcher_CancelPendingTimersWakesWaitersWithoutWaitingForDeadline` | done |

## gpui/src/platform/visual_test.rs

Go target: `platform_visual_testing.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `VisualTestPlatform` | struct | 31 |  | `VisualTestPlatform` | blocked:M3-root-capture |
| `VisualTestPlatform::new` | assoc-fn | 44 |  | `NewVisualTestPlatform` | blocked:M3-root-capture |
| `VisualTestPlatform::dispatcher` | method | 62 |  | `(VisualTestPlatform).Dispatcher` | blocked:M3-root-capture |

## gpui/src/platform/test/dispatcher.rs

Go target: `platform_testing_dispatcher.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestDispatcher` | struct | 17 |  | `TestDispatcher` | done |
| `TestDispatcher::new` | assoc-fn | 24 |  | `NewTestDispatcher` | done |
| `TestDispatcher::from_scheduler` | assoc-fn | 36 |  | `TestDispatcherFromScheduler` | done |
| `TestDispatcher::scheduler` | method | 44 |  | `(TestDispatcher).Scheduler` | done |
| `TestDispatcher::session_id` | method | 48 |  | `(TestDispatcher).SessionID` | done |
| `TestDispatcher::drain_tasks` | method | 52 |  | `(TestDispatcher).DrainTasks` | done |
| `TestDispatcher::advance_clock` | method | 56 |  | `(TestDispatcher).AdvanceClock` | done |
| `TestDispatcher::advance_clock_to_next_timer` | method | 60 |  | `(TestDispatcher).AdvanceClockToNextTimer` | done |
| `TestDispatcher::simulate_random_delay` | method | 64 |  | `(TestDispatcher).SimulateRandomDelay` | done |
| `TestDispatcher::tick` | method | 68 |  | `(TestDispatcher).Tick` | done |
| `TestDispatcher::run_until_parked` | method | 76 |  | `(TestDispatcher).RunUntilParked` | done |
| `TestDispatcher::allow_parking` | method | 80 |  | `(TestDispatcher).AllowParking` | done |
| `TestDispatcher::forbid_parking` | method | 84 |  | `(TestDispatcher).ForbidParking` | done |
| `TestDispatcher::set_num_cpus` | method | 90 |  | `(TestDispatcher).SetNumCPUs` | done |
| `TestDispatcher::num_cpus_override` | method | 95 |  | `(TestDispatcher).NumCPUsOverride` | done |
| `Clone for TestDispatcher` | impl | 103 |  | `(TestDispatcher).Clone` | done |

## gpui/src/platform/test/platform.rs

Go target: `platform_testing_platform.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestScreenCaptureSource` | struct | 64 |  | `TestScreenCaptureSource` | done |
| `TestScreenCaptureStream` | struct | 67 |  | `TestScreenCaptureStream` | done |
| `TestScreenCaptureSource::new` | assoc-fn | 846 | feat:test-support | `NewTestScreenCaptureSource` | done |

## gpui/src/platform/test/window.rs

Go target: `platform_testing_window.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestWindow` | struct | 71 |  | `TestWindow` | done |
| `TestWindow::simulate_close` | method | 152 |  | `(TestWindow).SimulateClose` | done |
| `TestWindow::simulate_scheduled_frame` | method | 159 |  | `(TestWindow).SimulateScheduledFrame` | done |
| `TestWindow::frame_scheduled` | method | 178 |  | `(TestWindow).FrameScheduled` | done |
| `TestWindow::simulate_visibility_change` | method | 182 |  | `(TestWindow).SimulateVisibilityChange` | done |
| `TestWindow::simulate_move_to_display` | method | 195 |  | `(TestWindow).SimulateMoveToDisplay` | done |
| `TestWindow::simulate_visual_viewport_change` | method | 207 |  | `(TestWindow).SimulateVisualViewportChange` | done |
| `TestWindow::simulate_insets_change` | method | 219 |  | `(TestWindow).SimulateInsetsChange` | done |
| `TestWindow::virtual_keyboard_requests` | method | 231 |  | `(TestWindow).VirtualKeyboardRequests` | done |
| `TestWindow::virtual_keyboard_dismissals` | method | 235 |  | `(TestWindow).VirtualKeyboardDismissals` | done |
| `TestWindow::text_input_configurations` | method | 240 |  | `(TestWindow).TextInputConfigurations` | done |
| `TestWindow::text_input_state_changes` | method | 244 |  | `(TestWindow).TextInputStateChanges` | done |
| `TestWindow::simulate_resize` | method | 248 |  | `(TestWindow).SimulateResize` | done |
| `TestWindow::simulate_scale_factor_change` | method | 262 |  | `(TestWindow).SimulateScaleFactorChange` | done |
| `TestWindow::attached_native_view_count` | method | 272 |  | `(TestWindow).AttachedNativeViewCount` | done |
| `TestWindow::focused_native_view` | method | 277 |  | `(TestWindow).FocusedNativeView` | done |
| `TestWindow::hit_test_native_view` | method | 282 |  | `(TestWindow).HitTestNativeView` | done |
| `TestWindow::simulate_key_down_before_native_view` | method | 291 |  | `(TestWindow).SimulateKeyDownBeforeNativeView` | done |
| `TestWindow::simulate_native_view_focused` | method | 301 |  | `(TestWindow).SimulateNativeViewFocused` | done |
| `TestWindow::simulate_appearance_change` | method | 322 |  | `(TestWindow).SimulateAppearanceChange` | done |
| `TestWindow::simulate_decorations_change` | method | 333 |  | `(TestWindow).SimulateDecorationsChange` | done |
| `TestWindow::frame_wake_count` | method | 345 |  | `(TestWindow).FrameWakeCount` | done |
| `TestWindow::simulate_frame_request` | method | 351 |  | `(TestWindow).SimulateFrameRequest` | done |
| `TestWindow::simulate_input` | method | 361 |  | `(TestWindow).SimulateInput` | done |
| `TestWindow::external_drag_files` | method | 372 |  | `(TestWindow).ExternalDragFiles` | done |
| `TestWindow::set_start_external_drag_result` | method | 376 |  | `(TestWindow).SetStartExternalDragResult` | done |


## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-SUBMOD | An item of a public submodule whose name collides with a root item gets the submodule name as a prefix: `layer_shell::Anchor` is `LayerShellAnchor` (geometry.rs already has `Anchor`) | `gpui/src/platform/layer_shell.rs:26`, `gpui/src/geometry.rs:2247` | `platform_layer_shell.go` |
| D-TUPLE | Tuples inside generic containers become small named structs (`Credentials` for `read_credentials`' `(String, Vec<u8>)`), same-typed tuples become arrays (`LayerShellOptions.Margin` is `Option[[4]Pixels]`), and tuple returns of closures and methods become multiple results (`AtlasBuildFunc`, `RasterizeGlyph`, `PlatformFrameSignal.Take`) | `gpui/src/platform.rs:672`, `:1659`, `:1889`, `:1190`, `gpui/src/platform/layer_shell.rs:78` | `platform.go`, `platform_layer_shell.go` |
| D-DEFAULTS | Provided trait methods live on embeddable `PlatformDefaults`, `PlatformWindowDefaults`, `PlatformDisplayDefaults`, `PlatformDispatcherDefaults`, `PlatformTextSystemDefaults`, `PlatformAtlasDefaults`; the first four are bound to the implementor with `Bind<Name>(&d, self)` because some defaults call other trait methods. `cfg`-gated trait methods exist on every GOOS with a no-op default where upstream has none (`read_from_primary`, `get_raw_handle`, ...), and the Linux-only button-layout functions (`WindowButtonLayoutLinuxDefault`, `WindowButtonLayoutParse`, `format`) build on every GOOS so their tests compile everywhere. Rust `#[default]` variants that are not the first become `Default<Type>` constructors (`DefaultLayer`, `DefaultKeyboardInteractivity`, `DefaultLayerShellOptions`, `DefaultWindowControls`) | `gpui/src/platform.rs:394-1552`, `gpui/src/platform/layer_shell.rs:21`, `:55` | `platform.go`, `platform_layer_shell.go` |
| D-GPUENUM | `repr(C)` enums that are part of GPU data are 4-byte `uint32` types, not `uint8` (`AtlasTextureKind`), so `AtlasTile` keeps upstream's 32-byte layout | `gpui/src/platform.rs:2103` | `platform.go` |
| D-PLATSEAM | Port-introduced seams with no upstream item: `RawWindowHandle` (tagged struct standing in for the `HasWindowHandle + HasDisplayHandle` supertraits of `PlatformWindow`, returned by `PlatformWindow.RawWindowHandle`); `Close` on `Platform` and `PlatformWindow` (their drop); `ScreenCaptureStream.Stop` (`ScapStream`'s drop); `NativeViewAttachment` (the `Box<dyn Any>` of `attach_native_view`, dropped with `Close`); the `ScapBackend` seam (`ScapBackend`, `ScapCapturer`, `ScapTarget`, `ScapDisplay`, `ScapWindow`, `ScapFrame*`, `ScapCapturerOptions`, `RegisterScapBackend`) mirroring the zed-scap API the glue calls; `UUID` / `NewUUIDV4` for the uuid crate | `gpui/src/platform.rs:1290`, `:811`, `:1385`, `gpui/src/platform/scap_screen_capture.rs:1-325`, zed-scap 0.0.8 | `platform.go` |
| D-RANGE | `std::ops::Range<usize>` is the root struct `Range{Start, End int}`; `&mut Option<Range<usize>>` out-parameters are `*Option[Range]`. Later stages reuse `Range` for every `Range<usize>` in the root package | `gpui/src/platform.rs:2174-2420` | `platform.go` |
| D-WITH | A builder method whose name equals an exported field of the same type gets a `With` prefix, because Go cannot declare both: `Menu::items` / `disabled` are `WithItems` / `WithDisabled`, `MenuItem::checked` / `disabled` are `WithChecked` / `WithDisabled` | `gpui/src/platform/app_menu.rs:26`, `:32`, `:176`, `:198` | `platform_app_menu.go` |
| D-GETTER | A getter that returns a public field of the same name is not declared; callers read the exported field (`Image.ID`, `Image.Format`, `Image.Bytes`, `ClipboardItem.Entries`, `ClipboardString.Text`) | `gpui/src/platform.rs:3326`, `:3555`, `:3615`, `:3620`, `:3651` | `platform.go` |
| D-IMAGEFRAME | The image crate's `Frame` is `ImageFrame` (window.rs owns a `Frame`), with the BGRA bytes in an `*image.RGBA` container (not premultiplied) and `Delay` as `time.Duration` instead of `image::Delay`'s millisecond ratio | `gpui/src/platform.rs:57`, `:3462-3531` | `platform.go` |
| D-IMGDECODE | Decoders are the Go standard library (png, jpeg, gif), x/image (bmp, tiff, webp still images and frame bitstreams) and in-repo ports of the image crate's ICO, PNM, EXIF-orientation and WebP animation compositing (image-webp 0.2.4). Differences: `image/gif` decodes all frames or fails, so one bad GIF frame fails the image where the image crate skips that frame; BMP files cover x/image/bmp's subset (1/4/8/24/32-bit uncompressed), and BMP entries inside ICO the same subset via the in-repo DIB reader; lossy WebP pixel values come from x/image's YCbCr conversion (as L-JPEG for JPEG); TIFF orientation is read from IFD0 tag 274 directly; the animated WebP canvas is capped at 512 MiB (the image crate's default `Limits::max_alloc`) and a frame that fails to decode is yielded as an error and skipped | `gpui/src/platform.rs:3462-3531` | `platform.go` |
