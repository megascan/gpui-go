# Parity: core-app

Scope: gpui: app, entities, contexts, executor, subscriptions, actions, globals, assets, test harness. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/action.rs

Go target: `action.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `actions` | macro | 24 |  | `see DESIGN §9 (actions!)` | dev:D-MACRO |
| `Action` | trait | 117 |  | `Action` | done |
| `Action::boxed_clone` | trait-method | 119 |  | `Action.BoxedClone` | done |
| `Action::partial_eq` | trait-method | 122 |  | `Action.PartialEq` | done |
| `Action::name` | trait-method | 125 |  | `Action.Name` | done |
| `Action::name_for_type` | trait-method | 128 |  | `ActionData.Name` | dev:D-ACTIONDATA |
| `Action::build` | trait-method | 134 |  | `ActionData.Build` | dev:D-ACTIONDATA |
| `Action::action_json_schema` | trait-method | 139 |  | `ActionData.JSONSchema` | dev:D-ACTIONDATA |
| `Action::deprecated_aliases` | trait-method | 149 |  | `ActionData.DeprecatedAliases` | dev:D-ACTIONDATA |
| `Action::deprecation_message` | trait-method | 158 |  | `ActionData.DeprecationMessage` | dev:D-ACTIONDATA |
| `Action::documentation` | trait-method | 167 |  | `ActionData.Documentation` | dev:D-ACTIONDATA |
| `Action::as_any` | method | 185 |  | type assertion on `Action` | n/a:D-ACTIONDATA |
| `ActionBuildError` | enum | 193 |  | `ActionBuildError` | done |
| `MacroActionBuilder` | struct | 267 |  | `RegisterAction` | n/a:D-ACTIONDATA |
| `MacroActionData` | struct | 272 |  | `ActionData` | dev:D-ACTIONDATA |
| `generate_list_of_all_registered_actions` | fn | 417 |  | `GenerateListOfAllRegisteredActions` | done |
| `Unbind` | struct | 445 |  | `Unbind` | done |
| `is_no_action` | fn | 448 |  | `IsNoAction` | done |
| `is_unbind` | fn | 453 |  | `IsUnbind` | done |

## gpui/src/app.rs

Go target: `app.go` (package `gpui`), tests in `app_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SHUTDOWN_TIMEOUT` | const | 80 |  | `ShutdownTimeout` | done |
| `AppCell` | struct | 85 |  | `AppCell` | done |
| `AppCell::borrow` | method | 128 |  | `(AppCell).Borrow` | dev:D-BORROWLOC |
| `AppCell::borrow_mut` | method | 155 |  | `(AppCell).BorrowMut` | dev:D-BORROWLOC |
| `AppCell::try_borrow_mut` | method | 161 |  | `(AppCell).TryBorrowMut` | dev:D-BORROWLOC |
| `AppRef` | struct | 185 |  | `AppRef` | dev:D-BORROWLOC |
| `AppRefMut` | struct | 198 |  | `AppRefMut` | dev:D-BORROWLOC |
| `Application` | struct | 211 |  | `Application` | done |
| `ApplicationHandle` | struct | 218 |  | `ApplicationHandle` | done |
| `ApplicationHandle::update` | method | 226 |  | `(ApplicationHandle).Update` | done |
| `ApplicationHandle::to_async` | method | 233 |  | `(ApplicationHandle).ToAsync` | done |
| `Application::with_platform` | assoc-fn | 242 |  | `ApplicationWithPlatform` | done |
| `Application::new_inaccessible` | assoc-fn | 259 |  | `NewApplicationInaccessible` | done |
| `Application::with_assets` | method | 266 |  | `(Application).WithAssets` | done |
| `Application::with_restart_arguments` | method | 276 |  | `(Application).WithRestartArguments` | done |
| `Application::with_http_client` | method | 282 |  | `(Application).WithHTTPClient` | done |
| `Application::with_quit_mode` | method | 291 |  | `(Application).WithQuitMode` | done |
| `Application::with_windowing` | method | 302 |  | `(Application).WithWindowing` | done |
| `Application::run` | method | 312 |  | `(Application).Run` | done |
| `Application::run_embedded` | method | 340 |  | `(Application).RunEmbedded` | done |
| `Application::on_open_urls` | method | 359 |  | `(Application).OnOpenURLs` | done |
| `Application::on_reopen` | method | 369 |  | `(Application).OnReopen` | done |
| `Application::background_executor` | method | 383 |  | `(Application).BackgroundExecutor` | done |
| `Application::foreground_executor` | method | 388 |  | `(Application).ForegroundExecutor` | done |
| `Application::text_system` | method | 393 |  | `(Application).TextSystem` | done |
| `Application::path_for_auxiliary_executable` | method | 398 |  | `(Application).PathForAuxiliaryExecutable` | done |
| `QuitMode` | enum | 526 |  | `QuitMode` | done |
| `CursorHideMode` | enum | 541 |  | `CursorHideMode` | done |
| `SystemWindowTab` | struct | 554 |  | `SystemWindowTab` | done |
| `SystemWindowTab::new` | assoc-fn | 563 |  | `NewSystemWindowTab` | done |
| `SystemWindowTabController` | struct | 575 |  | `SystemWindowTabController` | done |
| `SystemWindowTabController::new` | assoc-fn | 584 |  | `NewSystemWindowTabController` | done |
| `SystemWindowTabController::init` | assoc-fn | 592 |  | `SystemWindowTabControllerInit` | done |
| `SystemWindowTabController::tab_groups` | method | 597 |  | `(SystemWindowTabController).TabGroups` | done |
| `SystemWindowTabController::get_next_tab_group_window` | assoc-fn | 602 |  | `SystemWindowTabControllerGetNextTabGroupWindow` | done |
| `SystemWindowTabController::get_prev_tab_group_window` | assoc-fn | 627 |  | `SystemWindowTabControllerGetPrevTabGroupWindow` | done |
| `SystemWindowTabController::tabs` | method | 656 |  | `(SystemWindowTabController).Tabs` | done |
| `SystemWindowTabController::init_visible` | assoc-fn | 663 |  | `SystemWindowTabControllerInitVisible` | done |
| `SystemWindowTabController::is_visible` | method | 671 |  | `(SystemWindowTabController).IsVisible` | done |
| `SystemWindowTabController::set_visible` | assoc-fn | 676 |  | `SystemWindowTabControllerSetVisible` | done |
| `SystemWindowTabController::update_last_active` | assoc-fn | 682 |  | `SystemWindowTabControllerUpdateLastActive` | done |
| `SystemWindowTabController::update_tab_position` | assoc-fn | 694 |  | `SystemWindowTabControllerUpdateTabPosition` | done |
| `SystemWindowTabController::update_tab_title` | assoc-fn | 708 |  | `SystemWindowTabControllerUpdateTabTitle` | done |
| `SystemWindowTabController::add_tab` | assoc-fn | 732 |  | `SystemWindowTabControllerAddTab` | done |
| `SystemWindowTabController::remove_tab` | assoc-fn | 765 |  | `SystemWindowTabControllerRemoveTab` | done |
| `SystemWindowTabController::move_tab_to_new_window` | assoc-fn | 780 |  | `SystemWindowTabControllerMoveTabToNewWindow` | done |
| `SystemWindowTabController::merge_all_windows` | assoc-fn | 791 |  | `SystemWindowTabControllerMergeAllWindows` | done |
| `SystemWindowTabController::select_next_tab` | assoc-fn | 809 |  | `SystemWindowTabControllerSelectNextTab` | done |
| `SystemWindowTabController::select_previous_tab` | assoc-fn | 824 |  | `SystemWindowTabControllerSelectPreviousTab` | done |
| `App` | struct | 884 |  | `App` | done |
| `App::ref_counts_drop_handle` | method | 1248 |  | `(App).RefCountsDropHandle` | done |
| `App::leak_detector_snapshot` | method | 1258 |  | `(App).LeakDetectorSnapshot` | done |
| `App::assert_no_new_leaks` | method | 1274 |  | `(App).AssertNoNewLeaks` | done |
| `App::leak_detector_snapshot` | method | 1280 | feat:test-support | `(App).LeakDetectorSnapshot` | done |
| `App::assert_no_new_leaks` | method | 1287 | feat:test-support | `(App).AssertNoNewLeaks` | done |
| `App::shutdown` | method | 1294 |  | `(App).Shutdown` | done |
| `App::keyboard_layout` | method | 1342 |  | `(App).KeyboardLayout` | done |
| `App::keyboard_mapper` | method | 1347 |  | `(App).KeyboardMapper` | done |
| `App::on_keyboard_layout_change` | method | 1352 |  | `(App).OnKeyboardLayoutChange` | done |
| `App::on_text_scale_factor_change` | method | 1363 |  | `(App).OnTextScaleFactorChange` | done |
| `App::quit` | method | 1374 |  | `(App).Quit` | done |
| `App::request_windowing` | method | 1391 |  | `(App).RequestWindowing` | done |
| `App::graphical_environment` | method | 1409 |  | `(App).GraphicalEnvironment` | done |
| `App::cursor_hide_mode` | method | 1415 |  | `(App).CursorHideMode` | done |
| `App::set_cursor_hide_mode` | method | 1421 |  | `(App).SetCursorHideMode` | done |
| `App::is_cursor_visible` | method | 1430 |  | `(App).IsCursorVisible` | done |
| `App::reduce_motion` | method | 1436 |  | `(App).ReduceMotion` | done |
| `App::set_reduce_motion` | method | 1442 |  | `(App).SetReduceMotion` | done |
| `App::refresh_windows` | method | 1451 |  | `(App).RefreshWindows` | done |
| `App::observe` | method | 1476 |  | `(App).Observe` | done |
| `App::subscribe` | method | 1558 |  | `(App).Subscribe` | done |
| `App::windows` | method | 1612 |  | `(App).Windows` | done |
| `App::window_stack` | method | 1624 |  | `(App).WindowStack` | done |
| `App::active_window` | method | 1629 |  | `(App).ActiveWindow` | done |
| `App::active_display` | method | 1634 |  | `(App).ActiveDisplay` | done |
| `App::open_window` | method | 1670 |  | `(App).OpenWindow` | done |
| `App::activate` | method | 1710 |  | `(App).Activate` | done |
| `App::hide` | method | 1715 |  | `(App).Hide` | done |
| `App::hide_other_apps` | method | 1720 |  | `(App).HideOtherApps` | done |
| `App::unhide_other_apps` | method | 1725 |  | `(App).UnhideOtherApps` | done |
| `App::displays` | method | 1730 |  | `(App).Displays` | done |
| `App::primary_display` | method | 1735 |  | `(App).PrimaryDisplay` | done |
| `App::is_screen_capture_supported` | method | 1740 |  | `(App).IsScreenCaptureSupported` | done |
| `App::screen_capture_sources` | method | 1745 |  | `(App).ScreenCaptureSources` | done |
| `App::find_display` | method | 1752 |  | `(App).FindDisplay` | done |
| `App::debug_state` | method | 1765 |  | `(App).DebugState` | done |
| `App::thermal_state` | method | 1820 |  | `(App).ThermalState` | done |
| `App::prevent_idle_sleep` | method | 1825 |  | `(App).PreventIdleSleep` | done |
| `App::on_thermal_state_change` | method | 1830 |  | `(App).OnThermalStateChange` | done |
| `App::on_system_sleep` | method | 1844 |  | `(App).OnSystemSleep` | done |
| `App::observe_displays` | method | 1890 |  | `(App).ObserveDisplays` | done |
| `App::on_system_wake` | method | 1901 |  | `(App).OnSystemWake` | done |
| `App::on_app_lifecycle` | method | 1918 |  | `(App).OnAppLifecycle` | done |
| `App::on_memory_warning` | method | 1931 |  | `(App).OnMemoryWarning` | done |
| `App::window_appearance` | method | 1939 |  | `(App).WindowAppearance` | done |
| `App::set_window_appearance` | method | 1953 |  | `(App).SetWindowAppearance` | done |
| `App::button_layout` | method | 1958 |  | `(App).ButtonLayout` | done |
| `App::text_scale_factor` | method | 1963 |  | `(App).TextScaleFactor` | done |
| `App::read_from_clipboard` | method | 1968 |  | `(App).ReadFromClipboard` | done |
| `App::read_from_clipboard_async` | method | 1979 |  | `(App).ReadFromClipboardAsync` | done |
| `App::set_text_rendering_mode` | method | 1986 |  | `(App).SetTextRenderingMode` | done |
| `App::text_rendering_mode` | method | 1991 |  | `(App).TextRenderingMode` | done |
| `App::power_mode` | method | 1997 |  | `(App).PowerMode` | done |
| `App::write_to_clipboard` | method | 2002 |  | `(App).WriteToClipboard` | done |
| `App::read_from_primary` | method | 2009 | linux,freebsd | `(App).ReadFromPrimary` | done |
| `App::write_to_primary` | method | 2016 | linux,freebsd | `(App).WriteToPrimary` | done |
| `App::read_from_find_pasteboard` | method | 2026 | macos | `(App).ReadFromFindPasteboard` | done |
| `App::write_to_find_pasteboard` | method | 2036 | macos | `(App).WriteToFindPasteboard` | done |
| `App::write_credentials` | method | 2041 |  | `(App).WriteCredentials` | done |
| `App::write_credentials_if_not_exists` | method | 2058 | macos | `(App).WriteCredentialsIfNotExists` | done |
| `App::read_credentials` | method | 2069 |  | `(App).ReadCredentials` | done |
| `App::delete_credentials` | method | 2074 |  | `(App).DeleteCredentials` | done |
| `App::open_url` | method | 2079 |  | `(App).OpenURL` | done |
| `App::register_url_scheme` | method | 2089 |  | `(App).RegisterURLScheme` | done |
| `App::set_app_identity` | method | 2099 |  | `(App).SetAppIdentity` | done |
| `App::show_system_notification` | method | 2109 |  | `(App).ShowSystemNotification` | done |
| `App::dismiss_system_notification` | method | 2120 |  | `(App).DismissSystemNotification` | done |
| `App::on_system_notification_response` | method | 2127 |  | `(App).OnSystemNotificationResponse` | done |
| `App::app_path` | method | 2143 |  | `(App).AppPath` | done |
| `App::compositor_name` | method | 2150 |  | `(App).CompositorName` | done |
| `App::platform_info` | method | 2155 |  | `(App).PlatformInfo` | done |
| `App::path_for_auxiliary_executable` | method | 2160 |  | `(App).PathForAuxiliaryExecutable` | done |
| `App::prompt_for_paths` | method | 2169 |  | `(App).PromptForPaths` | done |
| `App::prompt_for_new_path` | method | 2182 |  | `(App).PromptForNewPath` | done |
| `App::reveal_path` | method | 2191 |  | `(App).RevealPath` | done |
| `App::open_with_system` | method | 2196 |  | `(App).OpenWithSystem` | done |
| `App::should_auto_hide_scrollbars` | method | 2201 |  | `(App).ShouldAutoHideScrollbars` | done |
| `App::restart` | method | 2206 |  | `(App).Restart` | done |
| `App::set_restart_path` | method | 2217 |  | `(App).SetRestartPath` | done |
| `App::http_client` | method | 2222 |  | `(App).HTTPClient` | done |
| `App::set_http_client` | method | 2227 |  | `(App).SetHTTPClient` | done |
| `App::set_quit_mode` | method | 2233 |  | `(App).SetQuitMode` | done |
| `App::set_activation_policy` | method | 2245 |  | `(App).SetActivationPolicy` | done |
| `App::svg_renderer` | method | 2250 |  | `(App).SvgRenderer` | done |
| `App::with_window` | method | 2473 |  | `(App).WithWindow` | done |
| `App::to_async` | method | 2509 |  | `(App).ToAsync` | done |
| `App::background_executor` | method | 2518 |  | `(App).BackgroundExecutor` | done |
| `App::is_test` | method | 2524 |  | `(App).IsTest` | done |
| `App::foreground_executor` | method | 2529 |  | `(App).ForegroundExecutor` | done |
| `App::foreground_journal` | method | 2539 | feat:profiler | `(App).ForegroundJournal` | done |
| `App::start_hang_monitor` | method | 2560 | wasm,feat:profiler | `(App).StartHangMonitor` | done |
| `App::spawn` | method | 2586 |  | `(App).Spawn` | done |
| `App::spawn_with_priority` | method | 2600 |  | `(App).SpawnWithPriority` | done |
| `App::defer` | method | 2617 |  | `(App).Defer` | done |
| `App::asset_source` | method | 2624 |  | `(App).AssetSource` | done |
| `App::text_system` | method | 2629 |  | `(App).TextSystem` | done |
| `App::on_missing_glyphs` | method | 2643 |  | `(App).OnMissingGlyphs` | done |
| `App::has_global` | method | 2670 |  | `(App).HasGlobal` | done |
| `App::global` | method | 2676 |  | `(App).Global` | done |
| `App::try_global` | method | 2684 |  | `(App).TryGlobal` | done |
| `App::global_mut` | method | 2692 |  | `(App).GlobalMut` | done |
| `App::default_global` | method | 2703 |  | `(App).DefaultGlobal` | done |
| `App::set_global` | method | 2714 |  | `(App).SetGlobal` | done |
| `App::clear_globals` | method | 2722 | feat:test-support | `(App).ClearGlobals` | done |
| `App::remove_global` | method | 2727 |  | `(App).RemoveGlobal` | done |
| `App::observe_global` | method | 2739 |  | `(App).ObserveGlobal` | done |
| `App::observe_new` | method | 2783 |  | `(App).ObserveNew` | done |
| `App::observe_release` | method | 2804 |  | `(App).ObserveRelease` | done |
| `App::observe_release_in` | method | 2823 |  | `(App).ObserveReleaseIn` | done |
| `App::observe_keystrokes` | method | 2842 |  | `(App).ObserveKeystrokes` | done |
| `App::intercept_keystrokes` | method | 2857 |  | `(App).InterceptKeystrokes` | done |
| `App::bind_keys` | method | 2869 |  | `(App).BindKeys` | done |
| `App::clear_key_bindings` | method | 2875 |  | `(App).ClearKeyBindings` | done |
| `App::key_bindings` | method | 2881 |  | `(App).KeyBindings` | done |
| `App::on_action` | method | 2888 |  | `(App).OnAction` | done |
| `App::stop_propagation` | method | 2908 |  | `(App).StopPropagation` | done |
| `App::propagate` | method | 2916 |  | `(App).Propagate` | done |
| `App::build_action` | method | 2921 |  | `(App).BuildAction` | done |
| `App::all_action_names` | method | 2931 |  | `(App).AllActionNames` | done |
| `App::all_bindings_for_input` | method | 2938 |  | `(App).AllBindingsForInput` | done |
| `App::action_schemas` | method | 2943 |  | `(App).ActionSchemas` | done |
| `App::action_schema_by_name` | method | 2954 |  | `(App).ActionSchemaByName` | done |
| `App::deprecated_actions_to_preferred_actions` | method | 2963 |  | `(App).DeprecatedActionsToPreferredActions` | done |
| `App::action_deprecation_messages` | method | 2968 |  | `(App).ActionDeprecationMessages` | done |
| `App::action_documentation` | method | 2973 |  | `(App).ActionDocumentation` | done |
| `App::on_app_quit` | method | 2979 |  | `(App).OnAppQuit` | done |
| `App::on_app_restart` | method | 2995 |  | `(App).OnAppRestart` | done |
| `App::on_window_closed` | method | 3004 |  | `(App).OnWindowClosed` | done |
| `App::is_action_available` | method | 3023 |  | `(App).IsActionAvailable` | done |
| `App::set_menus` | method | 3039 |  | `(App).SetMenus` | done |
| `App::get_menus` | method | 3045 |  | `(App).GetMenus` | done |
| `App::set_dock_menu` | method | 3050 |  | `(App).SetDockMenu` | done |
| `App::perform_dock_menu_action` | method | 3055 |  | `(App).PerformDockMenuAction` | done |
| `App::add_recent_document` | method | 3063 |  | `(App).AddRecentDocument` | done |
| `App::update_jump_list` | method | 3069 |  | `(App).UpdateJumpList` | done |
| `App::dispatch_action` | method | 3079 |  | `(App).DispatchAction` | done |
| `App::has_active_drag` | method | 3139 |  | `(App).HasActiveDrag` | done |
| `App::active_drag_cursor_style` | method | 3144 |  | `(App).ActiveDragCursorStyle` | done |
| `App::stop_active_drag` | method | 3149 |  | `(App).StopActiveDrag` | done |
| `App::set_active_drag_cursor_style` | method | 3223 |  | `(App).SetActiveDragCursorStyle` | done |
| `App::set_prompt_builder` | method | 3239 |  | `(App).SetPromptBuilder` | done |
| `App::reset_prompt_builder` | method | 3256 |  | `(App).ResetPromptBuilder` | done |
| `App::remove_asset` | method | 3261 |  | `(App).RemoveAsset` | done |
| `App::has_asset` | method | 3269 | feat:test-support | `(App).HasAsset` | done |
| `App::fetch_asset` | method | 3278 |  | `(App).FetchAsset` | done |
| `App::focus_handle` | method | 3298 |  | `(App).FocusHandle` | done |
| `App::notify` | method | 3303 |  | `(App).Notify` | done |
| `App::get_name` | method | 3341 | feat:test-support,debug | `(App).GetName` | done |
| `App::can_select_mixed_files_and_dirs` | method | 3346 |  | `(App).CanSelectMixedFilesAndDirs` | done |
| `App::drop_image` | method | 3354 |  | `(App).DropImage` | done |
| `App::set_inspector_renderer` | method | 3368 | feat:inspector,debug | `(App).SetInspectorRenderer` | done |
| `App::register_inspector_element` | method | 3374 | feat:inspector,debug | `(App).RegisterInspectorElement` | done |
| `App::init_colors` | method | 3386 |  | `(App).InitColors` | done |
| `AnyDrag` | struct | 3676 |  | `AnyDrag` | done |
| `ExternalDragPayloadSource` | type | 3697 |  | `ExternalDragPayloadSource` | done |
| `AnyTooltip` | struct | 3703 |  | `AnyTooltip` | done |
| `InputPreference` | enum | 3718 |  | `InputPreference` | done |
| `KeystrokeEvent` | struct | 3727 |  | `KeystrokeEvent` | done |
| `GpuiBorrow` | struct | 3773 |  | `GpuiBorrow` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `app_borrow_diagnostics` | gpui::test | 3842 |  | `TestApp_AppBorrowDiagnostics` | done |
| `async_app_refresh_flushes_refresh_effect` | gpui::test | 3900 |  | `TestApp_AsyncAppRefreshFlushesRefreshEffect` | done |
| `missing_glyph_callbacks_follow_subscription_lifetime` | gpui::test | 3917 |  | `TestApp_MissingGlyphCallbacksFollowSubscriptionLifetime` | done |
| `test_gpui_borrow` | test | 3976 |  | `TestApp_GpuiBorrow` | done |
| `test_restart_preserves_path_and_arguments` | gpui::test | 4007 |  | `TestApp_RestartPreservesPathAndArguments` | done |

## gpui/src/asset_cache.rs

Go target: `asset_cache.go` (package `gpui`), tests in `asset_cache_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Resource` | enum | 71 |  | `Resource` | done |
| `Asset` | trait | 99 |  | `Asset` | done |
| `Asset::load` | trait-method | 107 |  | `Asset.Load` | done |
| `AssetLogger` | enum | 114 |  | `AssetLogger` | done |
| `hash` | fn | 139 |  | `Hash` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `cached_load_caches_success_and_error` | gpui::test | 160 |  | `TestAssetCache_CachedLoadCachesSuccessAndError` | done |
| `completed_load_without_subscribers_is_cached_and_deduplicated` | gpui::test | 178 |  | `TestAssetCache_CompletedLoadWithoutSubscribersIsCachedAndDeduplicated` | done |
| `load_completion_notifies_each_observing_entity_once` | gpui::test | 201 |  | `TestAssetCache_LoadCompletionNotifiesEachObservingEntityOnce` | done |
| `removing_pending_asset_cancels_it_and_replacement_ignores_stale_completion` | gpui::test | 241 |  | `TestAssetCache_RemovingPendingAssetCancelsItAndReplacementIgnoresStaleCompletion` | done |
| `eviction_after_worker_completion_does_not_publish_into_replacement` | gpui::test | 271 |  | `TestAssetCache_EvictionAfterWorkerCompletionDoesNotPublishIntoReplacement` | done |

## gpui/src/assets.rs

Go target: `assets.go` (package `gpui`), tests in `assets_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AssetSource` | trait | 13 |  | `AssetSource` | done |
| `AssetSource::load` | trait-method | 15 |  | `AssetSource.Load` | done |
| `AssetSource::list` | trait-method | 18 |  | `AssetSource.List` | done |
| `ImageId` | struct | 33 |  | `ImageID` | done |
| `RenderImageParams` | struct | 37 |  | `RenderImageParams` | done |
| `RenderImage` | struct | 43 |  | `RenderImage` | done |
| `RenderImage::new` | assoc-fn | 61 |  | `NewRenderImage` | done |
| `RenderImage::as_bytes` | method | 72 |  | `(RenderImage).AsBytes` | done |
| `RenderImage::size` | method | 79 |  | `(RenderImage).Size` | done |
| `RenderImage::delay` | method | 96 |  | `(RenderImage).Delay` | done |
| `RenderImage::frame_count` | method | 104 |  | `(RenderImage).FrameCount` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `empty_render_image_does_not_panic` | test | 124 |  | `TestAssets_EmptyRenderImageDoesNotPanic` | done |

## gpui/src/executor.rs

Go target: `executor.go` (package `gpui`), tests in `executor_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `BackgroundExecutor` | struct | 19 |  | `BackgroundExecutor` | done |
| `ForegroundExecutor` | struct | 27 |  | `ForegroundExecutor` | done |
| `SendForegroundExecutor` | struct | 37 |  | `SendForegroundExecutor` | done |
| `SendForegroundExecutor::spawn` | method | 44 |  | `(SendForegroundExecutor).Spawn` | done |
| `TaskExt` | trait | 55 |  | `TaskDetachAndLogErr`, `TaskDetachAndLogErrWithBacktrace` (funcs) | dev:D-TASKEXT |
| `TaskExt::detach_and_log_err` | trait-method | 57 |  | `TaskDetachAndLogErr` | dev:D-TASKEXT |
| `TaskExt::detach_and_log_err_with_backtrace` | trait-method | 60 |  | `TaskDetachAndLogErrWithBacktrace` | dev:D-TASKEXT |
| `BackgroundExecutor::new` | assoc-fn | 87 |  | `NewBackgroundExecutor` | done |
| `BackgroundExecutor::scheduler_executor` | method | 107 |  | `(BackgroundExecutor).SchedulerExecutor` | done |
| `BackgroundExecutor::prevent_app_nap` | method | 114 |  | `(BackgroundExecutor).PreventAppNap` | done |
| `BackgroundExecutor::spawn` | method | 120 |  | `(BackgroundExecutor).Spawn` | done |
| `BackgroundExecutor::spawn_with_priority` | method | 132 |  | `(BackgroundExecutor).SpawnWithPriority` | done |
| `BackgroundExecutor::scoped` | method | 152 | wasm | `(BackgroundExecutor).Scoped` | done |
| `BackgroundExecutor::scoped_priority` | method | 173 | wasm | `(BackgroundExecutor).ScopedPriority` | done |
| `BackgroundExecutor::now` | method | 192 |  | `(BackgroundExecutor).Now` | done |
| `BackgroundExecutor::timer` | method | 200 |  | `(BackgroundExecutor).Timer` | done |
| `BackgroundExecutor::simulate_random_delay` | method | 209 | feat:test-support | `(BackgroundExecutor).SimulateRandomDelay` | done |
| `BackgroundExecutor::advance_clock` | method | 215 | feat:test-support | `(BackgroundExecutor).AdvanceClock` | done |
| `BackgroundExecutor::tick` | method | 221 | feat:test-support | `(BackgroundExecutor).Tick` | done |
| `BackgroundExecutor::run_until_parked` | method | 232 | feat:test-support | `(BackgroundExecutor).RunUntilParked` | done |
| `BackgroundExecutor::allow_parking` | method | 239 | feat:test-support | `(BackgroundExecutor).AllowParking` | done |
| `BackgroundExecutor::set_block_on_ticks` | method | 253 | feat:test-support | `(BackgroundExecutor).SetBlockOnTicks` | done |
| `BackgroundExecutor::forbid_parking` | method | 263 | feat:test-support | `(BackgroundExecutor).ForbidParking` | done |
| `BackgroundExecutor::rng` | method | 273 | feat:test-support | `(BackgroundExecutor).Rng` | done |
| `BackgroundExecutor::num_cpus` | method | 278 |  | `(BackgroundExecutor).NumCPUs` | done |
| `BackgroundExecutor::set_num_cpus` | method | 289 | feat:test-support | `(BackgroundExecutor).SetNumCPUs` | done |
| `BackgroundExecutor::is_main_thread` | method | 297 |  | `(BackgroundExecutor).IsMainThread` | done |
| `BackgroundExecutor::is_test` | method | 306 |  | `(BackgroundExecutor).IsTest` | done |
| `BackgroundExecutor::dispatcher` | method | 311 |  | `(BackgroundExecutor).Dispatcher` | done |
| `ForegroundExecutor::to_send` | method | 318 |  | `(ForegroundExecutor).ToSend` | done |
| `ForegroundExecutor::new` | assoc-fn | 325 |  | `NewForegroundExecutor` | done |
| `ForegroundExecutor::spawn` | method | 378 |  | `(ForegroundExecutor).Spawn` | done |
| `ForegroundExecutor::spawn_with_priority` | method | 387 |  | `(ForegroundExecutor).SpawnWithPriority` | done |
| `ForegroundExecutor::spawn_when_idle` | method | 409 |  | `(ForegroundExecutor).SpawnWhenIdle` | done |
| `ForegroundExecutor::idle_time_remaining` | method | 435 |  | `(ForegroundExecutor).IdleTimeRemaining` | done |
| `ForegroundExecutor::block_test` | method | 442 | wasm,feat:test-support | `(ForegroundExecutor).BlockTest` | done |
| `ForegroundExecutor::block_on` | method | 464 | wasm | `(ForegroundExecutor).BlockOn` | done |
| `ForegroundExecutor::block_with_timeout` | method | 470 | wasm | `(ForegroundExecutor).BlockWithTimeout` | done |
| `ForegroundExecutor::dispatcher` | method | 479 |  | `(ForegroundExecutor).Dispatcher` | done |
| `ForegroundExecutor::scheduler_executor` | method | 484 |  | `(ForegroundExecutor).SchedulerExecutor` | done |
| `Scope` | struct | 491 | wasm | `Scope` | done |
| `Scope::num_cpus` | method | 515 | wasm | `(Scope).NumCPUs` | done |
| `Scope::spawn` | method | 521 | wasm | `(Scope).Spawn` | done |
| `Drop for Scope` | impl | 543 | wasm | `(Scope).Close` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `send_foreground_executor_submits_from_worker` | test | 583 |  | `TestExecutor_SendForegroundExecutorSubmitsFromWorker` | done |
| `sanity_test_tasks_run` | test | 602 |  | `TestExecutor_SanityTestTasksRun` | done |

## gpui/src/global.rs

Go target: `global.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Global` | trait | 22 |  | none (marker trait dropped) | n/a:D-GLOBAL |
| `ReadGlobal` | trait | 30 |  | `Global[G]` (func) | dev:D-GLOBAL |
| `ReadGlobal::global` | trait-method | 34 |  | `Global[G]` (func) | dev:D-GLOBAL |
| `UpdateGlobal` | trait | 44 |  | `UpdateGlobal` / `UpdateGlobalR` / `SetGlobal` (gpui.go funcs) | dev:D-GLOBAL |
| `UpdateGlobal::update_global` | trait-method | 48 |  | `UpdateGlobal` / `UpdateGlobalR` (gpui.go) | dev:D-GLOBAL |
| `UpdateGlobal::set_global` | trait-method | 54 |  | `SetGlobal` (gpui.go) | dev:D-GLOBAL |

## gpui/src/platform_scheduler.rs

Go target: `platform_scheduler.go` (package `gpui`), tests in `platform_scheduler_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PlatformScheduler` | struct | 27 |  | `PlatformScheduler` | done |
| `PlatformScheduler::new` | assoc-fn | 36 |  | `NewPlatformScheduler` | done |
| `PlatformScheduler::foreground_executor` | method | 46 |  | `(PlatformScheduler).ForegroundExecutor` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `dedicated_executor_tasks_share_one_thread` | test | 222 | wasm | `TestPlatformScheduler_DedicatedExecutorTasksShareOneThread` | done |
| `spawn_dedicated_runs_on_a_real_separate_thread` | test | 242 | wasm | `TestPlatformScheduler_SpawnDedicatedRunsOnARealSeparateThread` | done |
| `spawn_dedicated_returns_not_send_future_output` | test | 272 | wasm | `TestPlatformScheduler_SpawnDedicatedReturnsNotSendFutureOutput` | done |
| `spawn_dedicated_dropping_task_cancels_future` | test | 293 | wasm | `TestPlatformScheduler_SpawnDedicatedDroppingTaskCancelsFuture` | done |
| `spawn_dedicated_thread_tears_down_after_work_completes` | test | 344 | wasm | `TestPlatformScheduler_SpawnDedicatedThreadTearsDownAfterWorkCompletes` | done |
| `spawn_dedicated_detached_child_outlives_root` | test | 402 | wasm | `TestPlatformScheduler_SpawnDedicatedDetachedChildOutlivesRoot` | done |

## gpui/src/queue.rs

Go target: `queue.go` (package `gpui`), tests in `queue_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PriorityQueueSender` | struct | 144 |  | `PriorityQueueSender` | done |
| `PriorityQueueSender::send` | method | 162 |  | `(PriorityQueueSender).Send` | done |
| `PriorityQueueSender::spin_send` | method | 176 |  | `(PriorityQueueSender).SpinSend` | done |
| `PriorityQueueReceiver` | struct | 206 |  | `PriorityQueueReceiver` | done |
| `SendError` | struct | 226 |  | `SendError` | done |
| `RecvError` | struct | 236 |  | `RecvError` | done |
| `PriorityQueueReceiver::new` | assoc-fn | 251 |  | `NewPriorityQueueReceiver` | done |
| `PriorityQueueReceiver::is_empty` | method | 276 |  | `(PriorityQueueReceiver).IsEmpty` | done |
| `PriorityQueueReceiver::try_pop` | method | 304 |  | `(PriorityQueueReceiver).TryPop` | done |
| `PriorityQueueReceiver::spin_try_pop` | method | 316 |  | `(PriorityQueueReceiver).SpinTryPop` | done |
| `PriorityQueueReceiver::pop` | method | 362 |  | `(PriorityQueueReceiver).Pop` | done |
| `PriorityQueueReceiver::try_iter` | method | 368 |  | `(PriorityQueueReceiver).TryIter` | done |
| `PriorityQueueReceiver::iter` | method | 377 |  | `(PriorityQueueReceiver).Iter` | done |
| `Iter` | struct | 437 |  | `Iter` | done |
| `TryIter` | struct | 448 |  | `TryIter` | done |
| `Drop for PriorityQueueSender` | impl | 182 |  | `(PriorityQueueSender).Close` | done |
| `Clone for PriorityQueueReceiver` | impl | 212 |  | `(PriorityQueueReceiver).Clone` | done |
| `Drop for PriorityQueueReceiver` | impl | 428 |  | `(PriorityQueueReceiver).Close` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `blocked_receivers_end_when_the_sender_drops` | test | 475 |  | `TestQueue_BlockedReceiversEndWhenTheSenderDrops` | done |
| `all_tasks_get_yielded` | test | 513 |  | `TestQueue_AllTasksGetYielded` | done |
| `new_high_prio_task_get_scheduled_quickly` | test | 530 |  | `TestQueue_NewHighPrioTaskGetScheduledQuickly` | done |

## gpui/src/seeds.rs

Go target: `seeds.go` (package `gpui`), tests in `seeds_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `calculate_seeds` | fn | 15 |  | `CalculateSeeds` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `seeds_follow_iterations_explicit_seeds_and_seed_variable` | test | 55 |  | `TestSeeds_SeedsFollowIterationsExplicitSeedsAndSeedVariable` | done |

## gpui/src/subscription.rs

Go target: `subscription.go` (package `gpui`), tests in `subscription_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Subscription` | struct | 262 |  | `Subscription` | done |
| `Subscription::new` | assoc-fn | 269 |  | `NewSubscription` | done |
| `Subscription::detach` | method | 278 |  | `(Subscription).Detach` | done |
| `Subscription::join` | assoc-fn | 284 |  | `SubscriptionJoin` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_immediate_and_pending_insertions` | test | 320 |  | `TestSubscription_ImmediateAndPendingInsertions` | done |
| `test_single_immediate_and_pending_insertions` | test | 345 |  | `TestSubscription_SingleImmediateAndPendingInsertions` | done |
| `test_activation_after_drop_during_retain` | test | 370 |  | `TestSubscription_ActivationAfterDropDuringRetain` | done |
| `test_single_activation_after_drop_during_retain` | test | 393 |  | `TestSubscription_SingleActivationAfterDropDuringRetain` | done |
| `test_unsubscribe_during_callback_with_insert` | test | 416 |  | `TestSubscription_UnsubscribeDuringCallbackWithInsert` | done |
| `test_callback_dropped_by_earlier_callback_does_not_fire` | test | 467 |  | `TestSubscription_CallbackDroppedByEarlierCallbackDoesNotFire` | done |
| `test_self_drop_during_callback` | test | 507 |  | `TestSubscription_SelfDropDuringCallback` | done |
| `test_subscription_drop` | test | 536 |  | `TestSubscription_SubscriptionDrop` | done |

## gpui/src/test.rs

Go target: `testing.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `seed_strategy` | fn | 43 |  | `SeedStrategy` | done |
| `apply_seed_to_proptest_config` | fn | 57 |  | `PropertyTestRNGSeed` | dev:D-PROPTEST |
| `run_test_once` | fn | 74 |  | `RunTestOnce` | done |
| `run_test` | fn | 95 |  | `RunTest` | done |
| `Observation` | struct | 145 |  | `Observation` | done |
| `observe` | fn | 162 |  | `Observe` | done |

## gpui/src/app/async_context.rs

Go target: `app_async_context.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AsyncApp` | struct | 22 |  | `AsyncApp` | done |
| `AsyncApp::refresh` | method | 150 |  | `(AsyncApp).Refresh` | done |
| `AsyncApp::background_executor` | method | 159 |  | `(AsyncApp).BackgroundExecutor` | done |
| `AsyncApp::is_test` | method | 165 |  | `(AsyncApp).IsTest` | done |
| `AsyncApp::foreground_executor` | method | 170 |  | `(AsyncApp).ForegroundExecutor` | done |
| `AsyncApp::update` | method | 176 |  | `(AsyncApp).Update` | done |
| `AsyncApp::subscribe` | method | 184 |  | `(AsyncApp).Subscribe` | done |
| `AsyncApp::open_window` | method | 199 |  | `(AsyncApp).OpenWindow` | done |
| `AsyncApp::spawn` | method | 218 |  | `(AsyncApp).Spawn` | done |
| `AsyncApp::has_global` | method | 229 |  | `(AsyncApp).HasGlobal` | done |
| `AsyncApp::read_global` | method | 239 |  | `(AsyncApp).ReadGlobal` | done |
| `AsyncApp::try_read_global` | method | 248 |  | `(AsyncApp).TryReadGlobal` | done |
| `AsyncApp::read_default_global` | method | 259 |  | `(AsyncApp).ReadDefaultGlobal` | done |
| `AsyncApp::update_global` | method | 273 |  | `(AsyncApp).UpdateGlobal` | done |
| `AsyncApp::on_drop` | assoc-fn | 280 |  | `AsyncAppOnDrop` | done |
| `AsyncWindowContext` | struct | 296 |  | `AsyncWindowContext` | done |
| `AsyncWindowContext::window_handle` | method | 309 |  | `(AsyncWindowContext).WindowHandle` | done |
| `AsyncWindowContext::update` | method | 316 |  | `(AsyncWindowContext).Update` | done |
| `AsyncWindowContext::update_root` | method | 324 |  | `(AsyncWindowContext).UpdateRoot` | done |
| `AsyncWindowContext::on_next_frame` | method | 332 |  | `(AsyncWindowContext).OnNextFrame` | done |
| `AsyncWindowContext::read_global` | method | 339 |  | `(AsyncWindowContext).ReadGlobal` | done |
| `AsyncWindowContext::update_global` | method | 349 |  | `(AsyncWindowContext).UpdateGlobal` | done |
| `AsyncWindowContext::spawn` | method | 365 |  | `(AsyncWindowContext).Spawn` | done |
| `AsyncWindowContext::prompt` | method | 378 |  | `(AsyncWindowContext).Prompt` | done |

## gpui/src/app/context.rs

Go target: `app_context.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Context` | struct | 20 |  | `Context` | done |
| `Context::entity_id` | method | 45 |  | `(Context).EntityID` | done |
| `Context::entity` | method | 50 |  | `(Context).Entity` | done |
| `Context::weak_entity` | method | 57 |  | `(Context).WeakEntity` | done |
| `Context::observe` | method | 63 |  | `(Context).Observe` | done |
| `Context::observe_self` | method | 79 |  | `(Context).ObserveSelf` | done |
| `Context::subscribe` | method | 93 |  | `(Context).Subscribe` | done |
| `Context::subscribe_self` | method | 111 |  | `(Context).SubscribeSelf` | done |
| `Context::on_release` | method | 126 |  | `(Context).OnRelease` | done |
| `Context::observe_release` | method | 140 |  | `(Context).ObserveRelease` | done |
| `Context::observe_global` | method | 163 |  | `(Context).ObserveGlobal` | done |
| `Context::on_app_restart` | method | 180 |  | `(Context).OnAppRestart` | done |
| `Context::on_app_quit` | method | 195 |  | `(Context).OnAppQuit` | done |
| `Context::notify` | method | 216 |  | `(Context).Notify` | done |
| `Context::spawn` | method | 225 |  | `(Context).Spawn` | done |
| `Context::listener` | method | 241 |  | `(Context).Listener` | done |
| `Context::processor` | method | 255 |  | `(Context).Processor` | done |
| `Context::on_drop` | method | 266 |  | `(Context).OnDrop` | done |
| `Context::focus_view` | method | 278 |  | `(Context).FocusView` | done |
| `Context::on_next_frame` | method | 283 |  | `(Context).OnNextFrame` | done |
| `Context::defer_in` | method | 296 |  | `(Context).DeferIn` | done |
| `Context::observe_in` | method | 312 |  | `(Context).ObserveIn` | done |
| `Context::subscribe_in` | method | 336 |  | `(Context).SubscribeIn` | done |
| `Context::on_release_in` | method | 365 |  | `(Context).OnReleaseIn` | done |
| `Context::observe_release_in` | method | 375 |  | `(Context).ObserveReleaseIn` | done |
| `Context::observe_window_bounds` | method | 397 |  | `(Context).ObserveWindowBounds` | done |
| `Context::observe_window_activation` | method | 410 |  | `(Context).ObserveWindowActivation` | done |
| `Context::observe_window_visibility` | method | 426 |  | `(Context).ObserveWindowVisibility` | done |
| `Context::observe_window_appearance` | method | 441 |  | `(Context).ObserveWindowAppearance` | done |
| `Context::observe_window_display` | method | 458 |  | `(Context).ObserveWindowDisplay` | done |
| `Context::observe_button_layout_changed` | method | 471 |  | `(Context).ObserveButtonLayoutChanged` | done |
| `Context::observe_keystrokes` | method | 486 |  | `(Context).ObserveKeystrokes` | done |
| `Context::observe_pending_input` | method | 503 |  | `(Context).ObservePendingInput` | done |
| `Context::on_focus` | method | 519 |  | `(Context).OnFocus` | done |
| `Context::on_focus_in` | method | 545 |  | `(Context).OnFocusIn` | done |
| `Context::on_blur` | method | 568 |  | `(Context).OnBlur` | done |
| `Context::on_focus_lost` | method | 595 |  | `(Context).OnFocusLost` | done |
| `Context::on_focus_out` | method | 614 |  | `(Context).OnFocusOut` | done |
| `Context::spawn_in` | method | 649 |  | `(Context).SpawnIn` | done |
| `Context::spawn_in_with_priority` | method | 664 |  | `(Context).SpawnInWithPriority` | done |
| `Context::observe_global_in` | method | 679 |  | `(Context).ObserveGlobalIn` | done |
| `Context::on_action` | method | 709 |  | `(Context).OnAction` | done |
| `Context::focus_self` | method | 726 |  | `(Context).FocusSelf` | done |
| `Context::emit` | method | 739 |  | `(Context).Emit` | done |

## gpui/src/app/entity_map.rs

Go target: `app_entity_map.go` (package `gpui`), tests in `app_entity_map_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `EntityId` | struct | 30 |  | `EntityID` | done |
| `EntityId::as_non_zero_u64` | method | 41 |  | `(EntityID).AsNonZeroU64` | done |
| `EntityId::as_u64` | method | 46 |  | `(EntityID).AsU64` | done |
| `AnyEntity` | struct | 347 |  | `AnyEntity` | done |
| `AnyEntity::entity_id` | method | 379 |  | `(AnyEntity).EntityID` | done |
| `AnyEntity::entity_type` | method | 385 |  | `(AnyEntity).EntityType` | done |
| `AnyEntity::downgrade` | method | 390 |  | `(AnyEntity).Downgrade` | done |
| `AnyEntity::downcast` | method | 400 |  | `(AnyEntity).Downcast` | done |
| `Entity` | struct | 515 |  | `Entity` | done |
| `Entity::entity_id` | method | 544 |  | `(Entity).EntityID` | done |
| `Entity::downgrade` | method | 550 |  | `(Entity).Downgrade` | done |
| `Entity::into_any` | method | 559 |  | `(Entity).IntoAny` | done |
| `Entity::read` | method | 565 |  | `(Entity).Read` | done |
| `Entity::read_with` | method | 571 |  | `(Entity).ReadWith` | done |
| `Entity::update` | method | 577 |  | `(Entity).Update` | done |
| `Entity::as_mut` | method | 587 |  | `(Entity).AsMut` | done |
| `Entity::write` | method | 592 |  | `(Entity).Write` | done |
| `Entity::update_in` | method | 603 |  | `(Entity).UpdateIn` | done |
| `AnyWeakEntity` | struct | 670 |  | `AnyWeakEntity` | done |
| `AnyWeakEntity::entity_id` | method | 679 |  | `(AnyWeakEntity).EntityID` | done |
| `AnyWeakEntity::is_upgradable` | method | 684 |  | `(AnyWeakEntity).IsUpgradable` | done |
| `AnyWeakEntity::upgrade` | method | 694 |  | `(AnyWeakEntity).Upgrade` | done |
| `AnyWeakEntity::assert_released` | method | 749 | feat:test-support | `(AnyWeakEntity).AssertReleased` | done |
| `AnyWeakEntity::new_invalid` | assoc-fn | 773 |  | `NewAnyWeakEntityInvalid` | done |
| `WeakEntity` | struct | 845 |  | `WeakEntity` | done |
| `WeakEntity::upgrade` | method | 872 |  | `(WeakEntity).Upgrade` | done |
| `WeakEntity::update` | method | 883 |  | `(WeakEntity).Update` | done |
| `WeakEntity::update_in` | method | 899 |  | `(WeakEntity).UpdateIn` | done |
| `WeakEntity::read_with` | method | 918 |  | `(WeakEntity).ReadWith` | done |
| `WeakEntity::new_invalid` | assoc-fn | 928 |  | `NewWeakEntityInvalid` | done |
| `LeakDetectorSnapshot` | struct | 1053 | feat:test-support | `LeakDetectorSnapshot` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_accessed_entity_scopes` | test | 1305 |  | `TestEntityMap_AccessedEntityScopes` | done |
| `test_entity_map_slot_assignment_before_cleanup` | test | 1337 |  | `TestEntityMap_EntityMapSlotAssignmentBeforeCleanup` | done |
| `test_entity_map_weak_upgrade_before_cleanup` | test | 1360 |  | `TestEntityMap_EntityMapWeakUpgradeBeforeCleanup` | done |
| `test_leak_detector_snapshot_no_leaks` | test | 1385 |  | `TestEntityMap_LeakDetectorSnapshotNoLeaks` | done |
| `test_leak_detector_snapshot_detects_new_leak` | test | 1404 |  | `TestEntityMap_LeakDetectorSnapshotDetectsNewLeak` | done |

## gpui/src/app/headless_app_context.rs

Go target: `app_headless_app_context.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `HeadlessAppContext` | struct | 38 |  | `HeadlessAppContext` | done |
| `HeadlessAppContext::new` | assoc-fn | 51 |  | `NewHeadlessAppContext` | done |
| `HeadlessAppContext::with_asset_source` | assoc-fn | 56 |  | `HeadlessAppContextWithAssetSource` | done |
| `HeadlessAppContext::with_platform` | assoc-fn | 65 |  | `HeadlessAppContextWithPlatform` | done |
| `HeadlessAppContext::open_window` | method | 104 |  | `(HeadlessAppContext).OpenWindow` | done |
| `HeadlessAppContext::run_until_parked` | method | 129 |  | `(HeadlessAppContext).RunUntilParked` | done |
| `HeadlessAppContext::simulate_window_scale_factor_change` | method | 134 |  | `(HeadlessAppContext).SimulateWindowScaleFactorChange` | done |
| `HeadlessAppContext::advance_clock` | method | 143 |  | `(HeadlessAppContext).AdvanceClock` | done |
| `HeadlessAppContext::allow_parking` | method | 148 |  | `(HeadlessAppContext).AllowParking` | done |
| `HeadlessAppContext::forbid_parking` | method | 153 |  | `(HeadlessAppContext).ForbidParking` | done |
| `HeadlessAppContext::update` | method | 158 |  | `(HeadlessAppContext).Update` | done |
| `HeadlessAppContext::update_window` | method | 164 |  | `(HeadlessAppContext).UpdateWindow` | done |
| `HeadlessAppContext::capture_screenshot` | method | 177 |  | `(HeadlessAppContext).CaptureScreenshot` | done |
| `HeadlessAppContext::text_system` | method | 183 |  | `(HeadlessAppContext).TextSystem` | done |
| `HeadlessAppContext::background_executor` | method | 188 |  | `(HeadlessAppContext).BackgroundExecutor` | done |
| `HeadlessAppContext::foreground_executor` | method | 193 |  | `(HeadlessAppContext).ForegroundExecutor` | done |
| `HeadlessAppContext::drop` | impl Drop | 199 |  | `(HeadlessAppContext).Close` | dev:D-RAII |

## gpui/src/app/test_app.rs

Go target: `app_test_app.go` (package `gpui`), tests in `app_test_app_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestApp` | struct | 40 |  | `TestApp` | done |
| `TestApp::new` | assoc-fn | 52 |  | `NewTestApp` | done |
| `TestApp::with_seed` | assoc-fn | 57 |  | `TestAppWithSeed` | done |
| `TestApp::with_text_system` | assoc-fn | 62 |  | `TestAppWithTextSystem` | done |
| `TestApp::with_text_system_and_assets` | assoc-fn | 67 |  | `TestAppWithTextSystemAndAssets` | done |
| `TestApp::update` | method | 111 |  | `(TestApp).Update` | done |
| `TestApp::read` | method | 121 |  | `(TestApp).Read` | done |
| `TestApp::new_entity` | method | 127 |  | `(TestApp).NewEntity` | done |
| `TestApp::update_entity` | method | 135 |  | `(TestApp).UpdateEntity` | done |
| `TestApp::read_entity` | method | 144 |  | `(TestApp).ReadEntity` | done |
| `TestApp::open_window` | method | 153 |  | `(TestApp).OpenWindow` | done |
| `TestApp::open_window_with_options` | method | 178 |  | `(TestApp).OpenWindowWithOptions` | done |
| `TestApp::run_until_parked` | method | 197 |  | `(TestApp).RunUntilParked` | done |
| `TestApp::advance_clock` | method | 202 |  | `(TestApp).AdvanceClock` | done |
| `TestApp::spawn` | method | 207 |  | `(TestApp).Spawn` | done |
| `TestApp::background_spawn` | method | 216 |  | `(TestApp).BackgroundSpawn` | done |
| `TestApp::to_async` | method | 224 |  | `(TestApp).ToAsync` | done |
| `TestApp::background_executor` | method | 233 |  | `(TestApp).BackgroundExecutor` | done |
| `TestApp::foreground_executor` | method | 238 |  | `(TestApp).ForegroundExecutor` | done |
| `TestApp::text_system` | method | 243 |  | `(TestApp).TextSystem` | done |
| `TestApp::has_global` | method | 248 |  | `(TestApp).HasGlobal` | done |
| `TestApp::set_global` | method | 253 |  | `(TestApp).SetGlobal` | done |
| `TestApp::read_global` | method | 258 |  | `(TestApp).ReadGlobal` | done |
| `TestApp::update_global` | method | 263 |  | `(TestApp).UpdateGlobal` | done |
| `TestApp::write_to_clipboard` | method | 270 |  | `(TestApp).WriteToClipboard` | done |
| `TestApp::read_from_clipboard` | method | 275 |  | `(TestApp).ReadFromClipboard` | done |
| `TestApp::opened_url` | method | 280 |  | `(TestApp).OpenedURL` | done |
| `TestApp::did_prompt_for_new_path` | method | 285 |  | `(TestApp).DidPromptForNewPath` | done |
| `TestApp::simulate_new_path_selection` | method | 290 |  | `(TestApp).SimulateNewPathSelection` | done |
| `TestApp::has_pending_prompt` | method | 298 |  | `(TestApp).HasPendingPrompt` | done |
| `TestApp::simulate_prompt_answer` | method | 303 |  | `(TestApp).SimulatePromptAnswer` | done |
| `TestApp::windows` | method | 308 |  | `(TestApp).Windows` | done |
| `TestAppWindow` | struct | 320 |  | `TestAppWindow` | done |
| `TestAppWindow::handle` | method | 329 |  | `(TestAppWindow).Handle` | done |
| `TestAppWindow::root` | method | 334 |  | `(TestAppWindow).Root` | done |
| `TestAppWindow::update` | method | 344 |  | `(TestAppWindow).Update` | done |
| `TestAppWindow::read` | method | 359 |  | `(TestAppWindow).Read` | done |
| `TestAppWindow::title` | method | 374 |  | `(TestAppWindow).Title` | done |
| `TestAppWindow::simulate_keystroke` | method | 384 |  | `(TestAppWindow).SimulateKeystroke` | done |
| `TestAppWindow::simulate_keystrokes` | method | 399 |  | `(TestAppWindow).SimulateKeystrokes` | done |
| `TestAppWindow::simulate_input` | method | 406 |  | `(TestAppWindow).SimulateInput` | done |
| `TestAppWindow::simulate_mouse_move` | method | 413 |  | `(TestAppWindow).SimulateMouseMove` | done |
| `TestAppWindow::simulate_mouse_down` | method | 422 |  | `(TestAppWindow).SimulateMouseDown` | done |
| `TestAppWindow::simulate_mouse_up` | method | 433 |  | `(TestAppWindow).SimulateMouseUp` | done |
| `TestAppWindow::simulate_click` | method | 443 |  | `(TestAppWindow).SimulateClick` | done |
| `TestAppWindow::simulate_scroll` | method | 449 |  | `(TestAppWindow).SimulateScroll` | done |
| `TestAppWindow::simulate_event` | method | 460 |  | `(TestAppWindow).SimulateEvent` | done |
| `TestAppWindow::simulate_resize` | method | 474 |  | `(TestAppWindow).SimulateResize` | done |
| `TestAppWindow::simulate_scale_factor_change` | method | 487 |  | `(TestAppWindow).SimulateScaleFactorChange` | done |
| `TestAppWindow::draw` | method | 505 |  | `(TestAppWindow).Draw` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_basic_usage` | test | 563 |  | `TestTestApp_BasicUsage` | done |
| `test_simulate_scale_factor_change` | test | 581 |  | `TestTestApp_SimulateScaleFactorChange` | done |
| `test_entity_creation` | test | 601 |  | `TestTestApp_EntityCreation` | done |
| `test_globals` | test | 623 |  | `TestTestApp_Globals` | done |

## gpui/src/app/test_context.rs

Go target: `app_test_context.go` (package `gpui`), tests in `app_test_context_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestAppContext` | struct | 35 |  | `TestAppContext` | done |
| `TestAppContext::set_platform_info` | method | 141 |  | `(TestAppContext).SetPlatformInfo` | done |
| `TestAppContext::build` | assoc-fn | 146 |  | `TestAppContextBuild` | done |
| `TestAppContext::build_with_text_system` | assoc-fn | 161 |  | `TestAppContextBuildWithTextSystem` | done |
| `TestAppContext::disable_accessibility` | method | 210 |  | `(TestAppContext).DisableAccessibility` | done |
| `TestAppContext::skip_drawing` | method | 215 |  | `(TestAppContext).SkipDrawing` | done |
| `TestAppContext::single` | assoc-fn | 220 |  | `TestAppContextSingle` | done |
| `TestAppContext::test_function_name` | method | 226 |  | `(TestAppContext).TestFunctionName` | done |
| `TestAppContext::did_prompt_for_new_path` | method | 231 |  | `(TestAppContext).DidPromptForNewPath` | done |
| `TestAppContext::active_idle_sleep_preventions` | method | 236 |  | `(TestAppContext).ActiveIdleSleepPreventions` | done |
| `TestAppContext::set_idle_sleep_prevention_delay` | method | 241 |  | `(TestAppContext).SetIdleSleepPreventionDelay` | done |
| `TestAppContext::set_idle_sleep_prevention_fails` | method | 246 |  | `(TestAppContext).SetIdleSleepPreventionFails` | done |
| `TestAppContext::new_app` | method | 251 |  | `(TestAppContext).NewApp` | done |
| `TestAppContext::quit` | method | 257 |  | `(TestAppContext).Quit` | done |
| `TestAppContext::on_quit` | method | 263 |  | `(TestAppContext).OnQuit` | done |
| `TestAppContext::refresh` | method | 268 |  | `(TestAppContext).Refresh` | done |
| `TestAppContext::executor` | method | 275 |  | `(TestAppContext).Executor` | done |
| `TestAppContext::foreground_executor` | method | 280 |  | `(TestAppContext).ForegroundExecutor` | done |
| `TestAppContext::update` | method | 285 |  | `(TestAppContext).Update` | done |
| `TestAppContext::read` | method | 291 |  | `(TestAppContext).Read` | done |
| `TestAppContext::add_window` | method | 298 |  | `(TestAppContext).AddWindow` | done |
| `TestAppContext::open_window` | method | 321 |  | `(TestAppContext).OpenWindow` | done |
| `TestAppContext::add_empty_window` | method | 345 |  | `(TestAppContext).AddEmptyWindow` | done |
| `TestAppContext::add_window_view` | method | 366 |  | `(TestAppContext).AddWindowView` | done |
| `TestAppContext::text_system` | method | 395 |  | `(TestAppContext).TextSystem` | done |
| `TestAppContext::write_to_clipboard` | method | 400 |  | `(TestAppContext).WriteToClipboard` | done |
| `TestAppContext::read_from_clipboard` | method | 406 |  | `(TestAppContext).ReadFromClipboard` | done |
| `TestAppContext::simulate_new_path_selection` | method | 411 |  | `(TestAppContext).SimulateNewPathSelection` | done |
| `TestAppContext::simulate_path_prompt_response` | method | 419 |  | `(TestAppContext).SimulatePathPromptResponse` | done |
| `TestAppContext::did_prompt_for_paths` | method | 428 |  | `(TestAppContext).DidPromptForPaths` | done |
| `TestAppContext::simulate_prompt_answer` | method | 434 |  | `(TestAppContext).SimulatePromptAnswer` | done |
| `TestAppContext::has_pending_prompt` | method | 439 |  | `(TestAppContext).HasPendingPrompt` | done |
| `TestAppContext::pending_prompt` | method | 444 |  | `(TestAppContext).PendingPrompt` | done |
| `TestAppContext::opened_url` | method | 449 |  | `(TestAppContext).OpenedURL` | done |
| `TestAppContext::app_identity` | method | 454 |  | `(TestAppContext).AppIdentity` | done |
| `TestAppContext::shown_system_notifications` | method | 459 |  | `(TestAppContext).ShownSystemNotifications` | done |
| `TestAppContext::delivered_system_notifications` | method | 464 |  | `(TestAppContext).DeliveredSystemNotifications` | done |
| `TestAppContext::dismissed_system_notifications` | method | 469 |  | `(TestAppContext).DismissedSystemNotifications` | done |
| `TestAppContext::simulate_system_notification_response` | method | 474 |  | `(TestAppContext).SimulateSystemNotificationResponse` | done |
| `TestAppContext::simulate_window_resize` | method | 480 |  | `(TestAppContext).SimulateWindowResize` | done |
| `TestAppContext::simulate_window_visibility_change` | method | 485 |  | `(TestAppContext).SimulateWindowVisibilityChange` | done |
| `TestAppContext::simulate_window_decorations_change` | method | 495 |  | `(TestAppContext).SimulateWindowDecorationsChange` | done |
| `TestAppContext::simulate_display_added` | method | 505 |  | `(TestAppContext).SimulateDisplayAdded` | done |
| `TestAppContext::simulate_display_removed` | method | 520 |  | `(TestAppContext).SimulateDisplayRemoved` | done |
| `TestAppContext::simulate_display_refresh_interval_change` | method | 525 |  | `(TestAppContext).SimulateDisplayRefreshIntervalChange` | done |
| `TestAppContext::simulate_window_visual_viewport_change` | method | 535 |  | `(TestAppContext).SimulateWindowVisualViewportChange` | done |
| `TestAppContext::simulate_window_scale_factor_change` | method | 545 |  | `(TestAppContext).SimulateWindowScaleFactorChange` | done |
| `TestAppContext::expect_restart` | method | 555 |  | `(TestAppContext).ExpectRestart` | done |
| `TestAppContext::set_screen_capture_sources` | method | 563 |  | `(TestAppContext).SetScreenCaptureSources` | done |
| `TestAppContext::windows` | method | 568 |  | `(TestAppContext).Windows` | done |
| `TestAppContext::spawn` | method | 574 |  | `(TestAppContext).Spawn` | done |
| `TestAppContext::has_global` | method | 583 |  | `(TestAppContext).HasGlobal` | done |
| `TestAppContext::read_global` | method | 590 |  | `(TestAppContext).ReadGlobal` | done |
| `TestAppContext::try_read_global` | method | 596 |  | `(TestAppContext).TryReadGlobal` | done |
| `TestAppContext::set_global` | method | 602 |  | `(TestAppContext).SetGlobal` | done |
| `TestAppContext::update_global` | method | 608 |  | `(TestAppContext).UpdateGlobal` | done |
| `TestAppContext::to_async` | method | 615 |  | `(TestAppContext).ToAsync` | done |
| `TestAppContext::run_until_parked` | method | 624 |  | `(TestAppContext).RunUntilParked` | done |
| `TestAppContext::dispatch_action` | method | 629 |  | `(TestAppContext).DispatchAction` | done |
| `TestAppContext::simulate_keystrokes` | method | 646 |  | `(TestAppContext).SimulateKeystrokes` | done |
| `TestAppContext::simulate_input` | method | 663 |  | `(TestAppContext).SimulateInput` | done |
| `TestAppContext::dispatch_keystroke` | method | 677 |  | `(TestAppContext).DispatchKeystroke` | done |
| `TestAppContext::notifications` | method | 690 |  | `(TestAppContext).Notifications` | done |
| `TestAppContext::events` | method | 710 |  | `(TestAppContext).Events` | done |
| `TestAppContext::condition` | method | 730 |  | `(TestAppContext).Condition` | done |
| `TestAppContext::set_name` | method | 762 | feat:test-support | `(TestAppContext).SetName` | done |
| `Entity::next_event` | method | 769 |  | `(Entity).NextEvent` | done |
| `Entity::next_notification` | method | 794 |  | `(Entity).NextNotification` | done |
| `Entity::condition` | method | 819 |  | `(Entity).Condition` | done |
| `VisualTestContext` | struct | 882 |  | `VisualTestContext` | done |
| `VisualTestContext::update` | method | 892 |  | `(VisualTestContext).Update` | done |
| `VisualTestContext::from_window` | assoc-fn | 901 |  | `VisualTestContextFromWindow` | done |
| `VisualTestContext::run_until_parked` | method | 909 |  | `(VisualTestContext).RunUntilParked` | done |
| `VisualTestContext::dispatch_action` | method | 914 |  | `(VisualTestContext).DispatchAction` | done |
| `VisualTestContext::window_title` | method | 922 |  | `(VisualTestContext).WindowTitle` | done |
| `VisualTestContext::document_path` | method | 927 |  | `(VisualTestContext).DocumentPath` | done |
| `VisualTestContext::simulate_keystrokes` | method | 938 |  | `(VisualTestContext).SimulateKeystrokes` | done |
| `VisualTestContext::simulate_input` | method | 944 |  | `(VisualTestContext).SimulateInput` | done |
| `VisualTestContext::simulate_mouse_move` | method | 949 |  | `(VisualTestContext).SimulateMouseMove` | done |
| `VisualTestContext::simulate_mouse_down` | method | 963 |  | `(VisualTestContext).SimulateMouseDown` | done |
| `VisualTestContext::simulate_mouse_up` | method | 979 |  | `(VisualTestContext).SimulateMouseUp` | done |
| `VisualTestContext::simulate_click` | method | 994 |  | `(VisualTestContext).SimulateClick` | done |
| `VisualTestContext::simulate_modifiers_change` | method | 1011 |  | `(VisualTestContext).SimulateModifiersChange` | done |
| `VisualTestContext::simulate_capslock_change` | method | 1019 |  | `(VisualTestContext).SimulateCapslockChange` | done |
| `VisualTestContext::simulate_resize` | method | 1027 |  | `(VisualTestContext).SimulateResize` | done |
| `VisualTestContext::simulate_visibility_change` | method | 1032 |  | `(VisualTestContext).SimulateVisibilityChange` | done |
| `VisualTestContext::simulate_scale_factor_change` | method | 1037 |  | `(VisualTestContext).SimulateScaleFactorChange` | done |
| `VisualTestContext::debug_bounds` | method | 1045 |  | `(VisualTestContext).DebugBounds` | done |
| `VisualTestContext::draw` | method | 1050 |  | `(VisualTestContext).Draw` | done |
| `VisualTestContext::simulate_event` | method | 1082 |  | `(VisualTestContext).SimulateEvent` | done |
| `VisualTestContext::deactivate_window` | method | 1089 |  | `(VisualTestContext).DeactivateWindow` | done |
| `VisualTestContext::simulate_close` | method | 1098 |  | `(VisualTestContext).SimulateClose` | done |
| `VisualTestContext::into_mut` | method | 1127 |  | `(VisualTestContext).IntoMut` | dev:D-RAII |
| `AnyWindowHandle::build_entity` | method | 1290 |  | `(AnyWindowHandle).BuildEntity` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_mobile_lifecycle_subscriptions` | gpui::test | 1311 |  | `TestTestContext_MobileLifecycleSubscriptions` | done |
| `test_memory_warning_subscriptions` | gpui::test | 1356 |  | `TestTestContext_MemoryWarningSubscriptions` | done |
| `test_system_notifications_require_identity_and_replace_matching_tags` | gpui::test | 1379 |  | `TestTestContext_SystemNotificationsRequireIdentityAndReplaceMatchingTags` | done |
| `test_system_notification_body_and_action_responses` | gpui::test | 1436 |  | `TestTestContext_SystemNotificationBodyAndActionResponses` | done |
| `test_system_notification_response_handler_can_be_replaced` | gpui::test | 1470 |  | `TestTestContext_SystemNotificationResponseHandlerCanBeReplaced` | done |
| `test_system_notification_response_handler_can_reenter_app` | gpui::test | 1495 |  | `TestTestContext_SystemNotificationResponseHandlerCanReenterApp` | done |
| `test_simulate_path_prompt_response` | gpui::test | 1519 |  | `TestTestContext_SimulatePathPromptResponse` | done |
| `test_simulate_path_prompt_cancellation` | gpui::test | 1547 |  | `TestTestContext_SimulatePathPromptCancellation` | done |

## gpui/src/app/visual_test_context.rs

Go target: `app_visual_test_context.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `VisualTestAppContext` | struct | 21 |  | `VisualTestAppContext` | done |
| `VisualTestAppContext::new` | assoc-fn | 45 |  | `NewVisualTestAppContext` | done |
| `VisualTestAppContext::with_asset_source` | assoc-fn | 53 |  | `VisualTestAppContextWithAssetSource` | done |
| `VisualTestAppContext::open_offscreen_window` | method | 97 |  | `(VisualTestAppContext).OpenOffscreenWindow` | done |
| `VisualTestAppContext::open_offscreen_window_default` | method | 122 |  | `(VisualTestAppContext).OpenOffscreenWindowDefault` | done |
| `VisualTestAppContext::is_screen_capture_supported` | method | 131 |  | `(VisualTestAppContext).IsScreenCaptureSupported` | done |
| `VisualTestAppContext::text_system` | method | 136 |  | `(VisualTestAppContext).TextSystem` | done |
| `VisualTestAppContext::executor` | method | 141 |  | `(VisualTestAppContext).Executor` | done |
| `VisualTestAppContext::foreground_executor` | method | 146 |  | `(VisualTestAppContext).ForegroundExecutor` | done |
| `VisualTestAppContext::run_until_parked` | method | 152 |  | `(VisualTestAppContext).RunUntilParked` | done |
| `VisualTestAppContext::advance_clock` | method | 159 |  | `(VisualTestAppContext).AdvanceClock` | done |
| `VisualTestAppContext::update` | method | 164 |  | `(VisualTestAppContext).Update` | done |
| `VisualTestAppContext::read` | method | 170 |  | `(VisualTestAppContext).Read` | done |
| `VisualTestAppContext::update_window` | method | 176 |  | `(VisualTestAppContext).UpdateWindow` | done |
| `VisualTestAppContext::spawn` | method | 185 |  | `(VisualTestAppContext).Spawn` | done |
| `VisualTestAppContext::has_global` | method | 194 |  | `(VisualTestAppContext).HasGlobal` | done |
| `VisualTestAppContext::read_global` | method | 200 |  | `(VisualTestAppContext).ReadGlobal` | done |
| `VisualTestAppContext::set_global` | method | 206 |  | `(VisualTestAppContext).SetGlobal` | done |
| `VisualTestAppContext::update_global` | method | 212 |  | `(VisualTestAppContext).UpdateGlobal` | done |
| `VisualTestAppContext::simulate_keystrokes` | method | 225 |  | `(VisualTestAppContext).SimulateKeystrokes` | done |
| `VisualTestAppContext::dispatch_keystroke` | method | 236 |  | `(VisualTestAppContext).DispatchKeystroke` | done |
| `VisualTestAppContext::simulate_input` | method | 244 |  | `(VisualTestAppContext).SimulateInput` | done |
| `VisualTestAppContext::simulate_mouse_move` | method | 258 |  | `(VisualTestAppContext).SimulateMouseMove` | done |
| `VisualTestAppContext::simulate_mouse_down` | method | 276 |  | `(VisualTestAppContext).SimulateMouseDown` | done |
| `VisualTestAppContext::simulate_mouse_up` | method | 296 |  | `(VisualTestAppContext).SimulateMouseUp` | done |
| `VisualTestAppContext::simulate_click` | method | 315 |  | `(VisualTestAppContext).SimulateClick` | done |
| `VisualTestAppContext::simulate_event` | method | 326 |  | `(VisualTestAppContext).SimulateEvent` | done |
| `VisualTestAppContext::dispatch_action` | method | 335 |  | `(VisualTestAppContext).DispatchAction` | done |
| `VisualTestAppContext::write_to_clipboard` | method | 344 |  | `(VisualTestAppContext).WriteToClipboard` | done |
| `VisualTestAppContext::read_from_clipboard` | method | 349 |  | `(VisualTestAppContext).ReadFromClipboard` | done |
| `VisualTestAppContext::wait_for` | method | 354 |  | `(VisualTestAppContext).WaitFor` | done |
| `VisualTestAppContext::capture_screenshot` | method | 385 | feat:test-support | `(VisualTestAppContext).CaptureScreenshot` | done |
| `VisualTestAppContext::wait_for_animations` | method | 390 |  | `(VisualTestAppContext).WaitForAnimations` | done |


## Notes (M0-input-types)

- action.rs: `RegisterAction` only appends to a package inventory (upstream `inventory::submit!`); the registry (`actionRegistry`, built by `newActionRegistry` = `ActionRegistry::default` + `load_actions`) panics on duplicate names with upstream's text when an App builds it. Its methods are unexported (`buildActionType`, `tryResolveAction`, `buildAction(name, Option[jsontext.Value])` defaulting to `{}`, `allActionNames`, `actionSchemas`, `actionSchemaByName`, `deprecatedAliasesMap`, `deprecationMessagesMap`, `documentationMap`). Schema functions take no generator (ActionData contract); a nil `JSONSchema` is upstream's `None`.
- `NoAction` and `Unbind` are hand-written in action.go (no `action_gen.go`): names `zed::NoAction`, `zed::Unbind`; `Unbind` is `type Unbind SharedString`, so its JSON is a bare string as serde's newtype.
- `ActionBuildErrorBuildError.Unwrap` returns `errors.Unwrap(Err)`, upstream's `error.source()`.
- assets.rs: `NoAssets` is `impl AssetSource for ()`; `RenderImage.String` is the Debug impl; `renderSize` is crate-private.
- seeds.rs: `CalculateSeeds` returns `(iter.Seq[uint64], bool)`; panics "invalid ITERATIONS variable: ..." / "invalid SEED variable as integer: ...".
- entity_map.rs: crate-private items are unexported: `entityMap` (`newEntityMap`, generic `reserve` / `insert` / `lease` / `endLease` / `read` plus erased `reserveErased` / `insertErased` / `leaseErased` / `endLeaseErased` / `readErased` for the AppContext primitives, access scopes, `takeDropped() []droppedEntity`, `leakDetectorSnapshot`, `assertNoNewLeaks`, `report`), `entityRefCounts` (slotmap port `entitySlotMap`: sentinel slot 0, key `version<<32|idx`, odd version while occupied), `leaseInner`, `lease[T]`, `slot[T]`, `leakDetector`, `handleID`. Entity state is stored as `*T` in `map[EntityID]any`. `extendAccessed` takes `[]EntityID`.
- `AnyEntity.Downcast[T]() (Entity[T], bool)`: false leaves the handle with the caller (upstream's `Err(self)`). `AsNonZeroU64` returns `uint64` and panics on 0. `EntityID.String` is Display; the Debug form `EntityId(1v1)` is used in leak messages.
- Handles per DESIGN 5.2: `AnyEntity{h *entityHandle}`; `Release` is idempotent and a no-op on the zero value; every other method panics on a zero or released handle. `runtime.AddCleanup` on each handle runs upstream's Drop path when a handle is collected without `Release`. `==` on strong handles compares handle identity; compare `EntityID()` for upstream's Eq / Ord / Hash. `AnyWeakEntity` is a plain value (id, `reflect.Type`, `weak.Pointer[entityRefCounts]`).
- Leak detection (`cfg(any(test, gpui_leak_detection))`) is runtime: on in test binaries whatever the variable says and elsewhere when `GPUI_LEAK_DETECTION` is non-empty (`0` included, `gpui/build.rs:13-16`); strict when it is `strict` or unset in a test binary. Strict mode also records entity handles, subscriptions and leases collected without `Release` / `Unsubscribe` / `Detach` / `endLease`; `report` (upstream `LeakDetector::drop`) panics with "Exited with leaked handles:" listing both. `LEAK_BACKTRACE` captures `runtime.Callers`; frames from `testing.tRunner` on are stripped as upstream strips `test::run_test_in_process`.
- subscription.rs: `subscriberSet[K comparable, C]` / `singleSubscriberSet[C]` keep one id-ordered list per emitter in a Go map (nothing iterates across emitters, so `Ord` on the key is not needed); `remove` returns an `iter.Seq[C]` that checks `Active` when iterated, as upstream's lazy `filter_map`; `retain` takes `func(*C) bool`. `Subscription` aliases one handle: `Unsubscribe` (Drop) is idempotent, `Detach` clears it, the zero value is empty. The four TestApp tests wait for `TestApp` (M0-test-contexts).

## Notes (M0-test-contexts)

- The test contexts keep upstream's executors and dispatcher as unexported fields (`bgExecutor`, `fgExecutor`, `dispatcher`, `app`); methods are the API. Closures returning a value use the `R` suffix (`UpdateR`, `ReadR`, `UpdateGlobalR`, `UpdateEntityR`). `VisualTestContext` embeds `*TestAppContext` (upstream's Deref) and is a pointer, so `IntoMut` returns it (D-RAII).
- End of a run: `TestAppContext.drop` (called by RunTest's teardown) runs the OnQuit callbacks, releases the app's windows as dropping `App` does, then runs GC plus the cleanup queue and an empty update until the live-handle count is stable and calls `entityMap.report` (upstream `LeakDetector::drop`). RunTest drains the scheduler after the report, upstream drains before it (handoff to M0-exit). Standalone contexts (`TestAppContextSingle`, `TestApp`, `HeadlessAppContext`, `VisualTestAppContext`) have `Close`, which also drains their own scheduler; `HeadlessAppContext.Close` runs `Shutdown` first, as its Drop does.
- `FakeHttpClient::with_404_response` is the unexported `fakeHTTPClient` (D-HTTP). `VisualTestContext.updateWindowContext` panics "window was unexpectedly closed" instead of returning an error (its `Result<T>` is `T`); the concrete `UpdateWindowEntity` uses the entity's current window as upstream.
- `VisualTestContext.Draw`, `DebugBounds`, `CaptureScreenshot`, `NewVisualTestAppContext` reach window and platform symbols that are still panicking stubs (`Drawable`, `WithAbsoluteElementOffset`, `RenderToImage`, `VisualTestPlatform`); there is no element arena (D-ARENA).
- The four subscription.rs TestApp tests live in subscription_test.go, which this stage does not own: `blocked:M0-exit` (docs/port/handoff/M0-test-contexts.md).

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-ACTIONDATA | The `Self: Sized` associated functions of `Action` (`name_for_type`, `build`, `action_json_schema`, `deprecated_aliases`, `deprecation_message`, `documentation`) are fields of the `ActionData` registration record (upstream's `MacroActionData`); `MacroActionBuilder` / `inventory` are `RegisterAction` from `init`; `as_any` is a Go type assertion | `gpui/src/action.rs:117-190`, `:267-290` | Go interfaces have no static methods; DESIGN 6.4 |
| D-GLOBAL | global.rs's blanket traits are free generic functions: `ReadGlobal::global` is `Global[G](cx)`, `UpdateGlobal::update_global` / `set_global` are gpui.go's `UpdateGlobal` / `UpdateGlobalR` / `SetGlobal`; the `Global` marker trait is dropped (any type can be a global) | `gpui/src/global.rs:22-75` | marker traits are dropped (DESIGN 5.4); blanket impls cannot be expressed |
| D-TASKEXT | `TaskExt`, an extension trait on `Task<Result<T, E>>`, is a pair of free generic functions `TaskDetachAndLogErr(task, cx)` / `TaskDetachAndLogErrWithBacktrace(task, cx)`, because `Task` is an alias of scheduler.Task and gets no methods in package gpui | `gpui/src/executor.rs:55-82` | executor.go |
| D-BORROWLOC | AppCell borrow diagnostics print `file:line` from `runtime.Caller` (absolute path, no `:column`), and `TRACK_THREAD_BORROWS` is read from the environment once at process start, printing the goroutine id as `ThreadId(N)` | `gpui/src/app.rs:85-207` | Go has no `#[track_caller]` column or compile-time `option_env!`; app.go |
| D-SEC-ASSETHASH | Asset cache keys hash the source with a per-process random `maphash` seed (`keyedHash`, the FxHash word stream fed into a seeded `maphash.Hash`) instead of the unkeyed 64-bit FxHash, so two sources cannot be forged to share a cache slot; `Image.ID` (`ImageFromBytes`, `imageID`) is a seeded `maphash` of format and bytes, and the clipboard readers on X11, Wayland, Windows and macOS build images through `ImageFromBytes`; `Hash` keeps upstream's FxHash for other callers (SEC-031) | `gpui/src/asset_cache.rs:139`, `gpui/src/platform.rs:3546` | `asset_cache.go` (`keyedHash`, `assetKeySeed`), `platform.go` (`imageID`), `platform/{linux/x11,linux/wayland,windows,macos}` clipboard readers |
| D-SEC-ASSETCACHE | The app's asset cache is an LRU capped at 4096 entries and 1 GiB of loaded bytes (`maxAssetCacheEntries`, `maxAssetCacheBytes`; render images count their frame bytes, `Image`s and byte slices their length). Entries that are still loading, or were used during the current frame or the last drawn frame of any open window, are pinned; trimming (after an insert and at the end of each window's frame) evicts unpinned least-recently-used entries only, cancels their background load, and does not notify views, which reload an evicted entry the next time they draw it. A failed or panicking background load resolves to the result's `Err` field (or a logged zero value) instead of a nil success (SEC-036, R2-2, R2-3, I-1) | `gpui/src/asset_cache.rs`, `gpui/src/app.rs` (`loading_assets`) | `asset_cache.go` (`assetCache`), `app.go`, `elements_image_cache.go` |
| D-SEC-R2-13 | `BackgroundExecutor.Scoped` and `ScopedPriority` return `error`: the joined errors of the scoped tasks, where a panicking task contributes the `*scheduler.PanicError` from D-SEC-TASKPANIC; every task is still awaited after one fails. Upstream's `scoped` returns `()` because a panic there aborts the process (R2-13, SEC-087) | `gpui/src/executor.rs:152` (`scoped`), `:173` | After D-SEC-TASKPANIC a panicking scoped task resolved with an error that `Scoped` dropped, so the caller saw success with partial results |
