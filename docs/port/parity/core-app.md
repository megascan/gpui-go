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
| `SHUTDOWN_TIMEOUT` | const | 80 |  | `ShutdownTimeout` | todo |
| `AppCell` | struct | 85 |  | `AppCell` | todo |
| `AppCell::borrow` | method | 128 |  | `(AppCell).Borrow` | todo |
| `AppCell::borrow_mut` | method | 155 |  | `(AppCell).BorrowMut` | todo |
| `AppCell::try_borrow_mut` | method | 161 |  | `(AppCell).TryBorrowMut` | todo |
| `AppRef` | struct | 185 |  | `AppRef` | todo |
| `AppRefMut` | struct | 198 |  | `AppRefMut` | todo |
| `Application` | struct | 211 |  | `Application` | todo |
| `ApplicationHandle` | struct | 218 |  | `ApplicationHandle` | todo |
| `ApplicationHandle::update` | method | 226 |  | `(ApplicationHandle).Update` | todo |
| `ApplicationHandle::to_async` | method | 233 |  | `(ApplicationHandle).ToAsync` | todo |
| `Application::with_platform` | assoc-fn | 242 |  | `ApplicationWithPlatform` | todo |
| `Application::new_inaccessible` | assoc-fn | 259 |  | `NewApplicationInaccessible` | todo |
| `Application::with_assets` | method | 266 |  | `(Application).WithAssets` | todo |
| `Application::with_restart_arguments` | method | 276 |  | `(Application).WithRestartArguments` | todo |
| `Application::with_http_client` | method | 282 |  | `(Application).WithHTTPClient` | todo |
| `Application::with_quit_mode` | method | 291 |  | `(Application).WithQuitMode` | todo |
| `Application::with_windowing` | method | 302 |  | `(Application).WithWindowing` | todo |
| `Application::run` | method | 312 |  | `(Application).Run` | todo |
| `Application::run_embedded` | method | 340 |  | `(Application).RunEmbedded` | todo |
| `Application::on_open_urls` | method | 359 |  | `(Application).OnOpenURLs` | todo |
| `Application::on_reopen` | method | 369 |  | `(Application).OnReopen` | todo |
| `Application::background_executor` | method | 383 |  | `(Application).BackgroundExecutor` | todo |
| `Application::foreground_executor` | method | 388 |  | `(Application).ForegroundExecutor` | todo |
| `Application::text_system` | method | 393 |  | `(Application).TextSystem` | todo |
| `Application::path_for_auxiliary_executable` | method | 398 |  | `(Application).PathForAuxiliaryExecutable` | todo |
| `QuitMode` | enum | 526 |  | `QuitMode` | todo |
| `CursorHideMode` | enum | 541 |  | `CursorHideMode` | todo |
| `SystemWindowTab` | struct | 554 |  | `SystemWindowTab` | todo |
| `SystemWindowTab::new` | assoc-fn | 563 |  | `NewSystemWindowTab` | todo |
| `SystemWindowTabController` | struct | 575 |  | `SystemWindowTabController` | todo |
| `SystemWindowTabController::new` | assoc-fn | 584 |  | `NewSystemWindowTabController` | todo |
| `SystemWindowTabController::init` | assoc-fn | 592 |  | `SystemWindowTabControllerInit` | todo |
| `SystemWindowTabController::tab_groups` | method | 597 |  | `(SystemWindowTabController).TabGroups` | todo |
| `SystemWindowTabController::get_next_tab_group_window` | assoc-fn | 602 |  | `SystemWindowTabControllerGetNextTabGroupWindow` | todo |
| `SystemWindowTabController::get_prev_tab_group_window` | assoc-fn | 627 |  | `SystemWindowTabControllerGetPrevTabGroupWindow` | todo |
| `SystemWindowTabController::tabs` | method | 656 |  | `(SystemWindowTabController).Tabs` | todo |
| `SystemWindowTabController::init_visible` | assoc-fn | 663 |  | `SystemWindowTabControllerInitVisible` | todo |
| `SystemWindowTabController::is_visible` | method | 671 |  | `(SystemWindowTabController).IsVisible` | todo |
| `SystemWindowTabController::set_visible` | assoc-fn | 676 |  | `SystemWindowTabControllerSetVisible` | todo |
| `SystemWindowTabController::update_last_active` | assoc-fn | 682 |  | `SystemWindowTabControllerUpdateLastActive` | todo |
| `SystemWindowTabController::update_tab_position` | assoc-fn | 694 |  | `SystemWindowTabControllerUpdateTabPosition` | todo |
| `SystemWindowTabController::update_tab_title` | assoc-fn | 708 |  | `SystemWindowTabControllerUpdateTabTitle` | todo |
| `SystemWindowTabController::add_tab` | assoc-fn | 732 |  | `SystemWindowTabControllerAddTab` | todo |
| `SystemWindowTabController::remove_tab` | assoc-fn | 765 |  | `SystemWindowTabControllerRemoveTab` | todo |
| `SystemWindowTabController::move_tab_to_new_window` | assoc-fn | 780 |  | `SystemWindowTabControllerMoveTabToNewWindow` | todo |
| `SystemWindowTabController::merge_all_windows` | assoc-fn | 791 |  | `SystemWindowTabControllerMergeAllWindows` | todo |
| `SystemWindowTabController::select_next_tab` | assoc-fn | 809 |  | `SystemWindowTabControllerSelectNextTab` | todo |
| `SystemWindowTabController::select_previous_tab` | assoc-fn | 824 |  | `SystemWindowTabControllerSelectPreviousTab` | todo |
| `App` | struct | 884 |  | `App` | todo |
| `App::ref_counts_drop_handle` | method | 1248 |  | `(App).RefCountsDropHandle` | todo |
| `App::leak_detector_snapshot` | method | 1258 |  | `(App).LeakDetectorSnapshot` | todo |
| `App::assert_no_new_leaks` | method | 1274 |  | `(App).AssertNoNewLeaks` | todo |
| `App::leak_detector_snapshot` | method | 1280 | feat:test-support | `(App).LeakDetectorSnapshot` | todo |
| `App::assert_no_new_leaks` | method | 1287 | feat:test-support | `(App).AssertNoNewLeaks` | todo |
| `App::shutdown` | method | 1294 |  | `(App).Shutdown` | todo |
| `App::keyboard_layout` | method | 1342 |  | `(App).KeyboardLayout` | todo |
| `App::keyboard_mapper` | method | 1347 |  | `(App).KeyboardMapper` | todo |
| `App::on_keyboard_layout_change` | method | 1352 |  | `(App).OnKeyboardLayoutChange` | todo |
| `App::on_text_scale_factor_change` | method | 1363 |  | `(App).OnTextScaleFactorChange` | todo |
| `App::quit` | method | 1374 |  | `(App).Quit` | todo |
| `App::request_windowing` | method | 1391 |  | `(App).RequestWindowing` | todo |
| `App::graphical_environment` | method | 1409 |  | `(App).GraphicalEnvironment` | todo |
| `App::cursor_hide_mode` | method | 1415 |  | `(App).CursorHideMode` | todo |
| `App::set_cursor_hide_mode` | method | 1421 |  | `(App).SetCursorHideMode` | todo |
| `App::is_cursor_visible` | method | 1430 |  | `(App).IsCursorVisible` | todo |
| `App::reduce_motion` | method | 1436 |  | `(App).ReduceMotion` | todo |
| `App::set_reduce_motion` | method | 1442 |  | `(App).SetReduceMotion` | todo |
| `App::refresh_windows` | method | 1451 |  | `(App).RefreshWindows` | todo |
| `App::observe` | method | 1476 |  | `(App).Observe` | todo |
| `App::subscribe` | method | 1558 |  | `(App).Subscribe` | todo |
| `App::windows` | method | 1612 |  | `(App).Windows` | todo |
| `App::window_stack` | method | 1624 |  | `(App).WindowStack` | todo |
| `App::active_window` | method | 1629 |  | `(App).ActiveWindow` | todo |
| `App::active_display` | method | 1634 |  | `(App).ActiveDisplay` | todo |
| `App::open_window` | method | 1670 |  | `(App).OpenWindow` | todo |
| `App::activate` | method | 1710 |  | `(App).Activate` | todo |
| `App::hide` | method | 1715 |  | `(App).Hide` | todo |
| `App::hide_other_apps` | method | 1720 |  | `(App).HideOtherApps` | todo |
| `App::unhide_other_apps` | method | 1725 |  | `(App).UnhideOtherApps` | todo |
| `App::displays` | method | 1730 |  | `(App).Displays` | todo |
| `App::primary_display` | method | 1735 |  | `(App).PrimaryDisplay` | todo |
| `App::is_screen_capture_supported` | method | 1740 |  | `(App).IsScreenCaptureSupported` | todo |
| `App::screen_capture_sources` | method | 1745 |  | `(App).ScreenCaptureSources` | todo |
| `App::find_display` | method | 1752 |  | `(App).FindDisplay` | todo |
| `App::debug_state` | method | 1765 |  | `(App).DebugState` | todo |
| `App::thermal_state` | method | 1820 |  | `(App).ThermalState` | todo |
| `App::prevent_idle_sleep` | method | 1825 |  | `(App).PreventIdleSleep` | todo |
| `App::on_thermal_state_change` | method | 1830 |  | `(App).OnThermalStateChange` | todo |
| `App::on_system_sleep` | method | 1844 |  | `(App).OnSystemSleep` | todo |
| `App::observe_displays` | method | 1890 |  | `(App).ObserveDisplays` | todo |
| `App::on_system_wake` | method | 1901 |  | `(App).OnSystemWake` | todo |
| `App::on_app_lifecycle` | method | 1918 |  | `(App).OnAppLifecycle` | todo |
| `App::on_memory_warning` | method | 1931 |  | `(App).OnMemoryWarning` | todo |
| `App::window_appearance` | method | 1939 |  | `(App).WindowAppearance` | todo |
| `App::set_window_appearance` | method | 1953 |  | `(App).SetWindowAppearance` | todo |
| `App::button_layout` | method | 1958 |  | `(App).ButtonLayout` | todo |
| `App::text_scale_factor` | method | 1963 |  | `(App).TextScaleFactor` | todo |
| `App::read_from_clipboard` | method | 1968 |  | `(App).ReadFromClipboard` | todo |
| `App::read_from_clipboard_async` | method | 1979 |  | `(App).ReadFromClipboardAsync` | todo |
| `App::set_text_rendering_mode` | method | 1986 |  | `(App).SetTextRenderingMode` | todo |
| `App::text_rendering_mode` | method | 1991 |  | `(App).TextRenderingMode` | todo |
| `App::power_mode` | method | 1997 |  | `(App).PowerMode` | todo |
| `App::write_to_clipboard` | method | 2002 |  | `(App).WriteToClipboard` | todo |
| `App::read_from_primary` | method | 2009 | linux,freebsd | `(App).ReadFromPrimary` | todo |
| `App::write_to_primary` | method | 2016 | linux,freebsd | `(App).WriteToPrimary` | todo |
| `App::read_from_find_pasteboard` | method | 2026 | macos | `(App).ReadFromFindPasteboard` | todo |
| `App::write_to_find_pasteboard` | method | 2036 | macos | `(App).WriteToFindPasteboard` | todo |
| `App::write_credentials` | method | 2041 |  | `(App).WriteCredentials` | todo |
| `App::write_credentials_if_not_exists` | method | 2058 | macos | `(App).WriteCredentialsIfNotExists` | todo |
| `App::read_credentials` | method | 2069 |  | `(App).ReadCredentials` | todo |
| `App::delete_credentials` | method | 2074 |  | `(App).DeleteCredentials` | todo |
| `App::open_url` | method | 2079 |  | `(App).OpenURL` | todo |
| `App::register_url_scheme` | method | 2089 |  | `(App).RegisterURLScheme` | todo |
| `App::set_app_identity` | method | 2099 |  | `(App).SetAppIdentity` | todo |
| `App::show_system_notification` | method | 2109 |  | `(App).ShowSystemNotification` | todo |
| `App::dismiss_system_notification` | method | 2120 |  | `(App).DismissSystemNotification` | todo |
| `App::on_system_notification_response` | method | 2127 |  | `(App).OnSystemNotificationResponse` | todo |
| `App::app_path` | method | 2143 |  | `(App).AppPath` | todo |
| `App::compositor_name` | method | 2150 |  | `(App).CompositorName` | todo |
| `App::platform_info` | method | 2155 |  | `(App).PlatformInfo` | todo |
| `App::path_for_auxiliary_executable` | method | 2160 |  | `(App).PathForAuxiliaryExecutable` | todo |
| `App::prompt_for_paths` | method | 2169 |  | `(App).PromptForPaths` | todo |
| `App::prompt_for_new_path` | method | 2182 |  | `(App).PromptForNewPath` | todo |
| `App::reveal_path` | method | 2191 |  | `(App).RevealPath` | todo |
| `App::open_with_system` | method | 2196 |  | `(App).OpenWithSystem` | todo |
| `App::should_auto_hide_scrollbars` | method | 2201 |  | `(App).ShouldAutoHideScrollbars` | todo |
| `App::restart` | method | 2206 |  | `(App).Restart` | todo |
| `App::set_restart_path` | method | 2217 |  | `(App).SetRestartPath` | todo |
| `App::http_client` | method | 2222 |  | `(App).HTTPClient` | todo |
| `App::set_http_client` | method | 2227 |  | `(App).SetHTTPClient` | todo |
| `App::set_quit_mode` | method | 2233 |  | `(App).SetQuitMode` | todo |
| `App::set_activation_policy` | method | 2245 |  | `(App).SetActivationPolicy` | todo |
| `App::svg_renderer` | method | 2250 |  | `(App).SvgRenderer` | todo |
| `App::with_window` | method | 2473 |  | `(App).WithWindow` | todo |
| `App::to_async` | method | 2509 |  | `(App).ToAsync` | todo |
| `App::background_executor` | method | 2518 |  | `(App).BackgroundExecutor` | todo |
| `App::is_test` | method | 2524 |  | `(App).IsTest` | todo |
| `App::foreground_executor` | method | 2529 |  | `(App).ForegroundExecutor` | todo |
| `App::foreground_journal` | method | 2539 | feat:profiler | `(App).ForegroundJournal` | todo |
| `App::start_hang_monitor` | method | 2560 | wasm,feat:profiler | `(App).StartHangMonitor` | todo |
| `App::spawn` | method | 2586 |  | `(App).Spawn` | todo |
| `App::spawn_with_priority` | method | 2600 |  | `(App).SpawnWithPriority` | todo |
| `App::defer` | method | 2617 |  | `(App).Defer` | todo |
| `App::asset_source` | method | 2624 |  | `(App).AssetSource` | todo |
| `App::text_system` | method | 2629 |  | `(App).TextSystem` | todo |
| `App::on_missing_glyphs` | method | 2643 |  | `(App).OnMissingGlyphs` | todo |
| `App::has_global` | method | 2670 |  | `(App).HasGlobal` | todo |
| `App::global` | method | 2676 |  | `(App).Global` | todo |
| `App::try_global` | method | 2684 |  | `(App).TryGlobal` | todo |
| `App::global_mut` | method | 2692 |  | `(App).GlobalMut` | todo |
| `App::default_global` | method | 2703 |  | `(App).DefaultGlobal` | todo |
| `App::set_global` | method | 2714 |  | `(App).SetGlobal` | todo |
| `App::clear_globals` | method | 2722 | feat:test-support | `(App).ClearGlobals` | todo |
| `App::remove_global` | method | 2727 |  | `(App).RemoveGlobal` | todo |
| `App::observe_global` | method | 2739 |  | `(App).ObserveGlobal` | todo |
| `App::observe_new` | method | 2783 |  | `(App).ObserveNew` | todo |
| `App::observe_release` | method | 2804 |  | `(App).ObserveRelease` | todo |
| `App::observe_release_in` | method | 2823 |  | `(App).ObserveReleaseIn` | todo |
| `App::observe_keystrokes` | method | 2842 |  | `(App).ObserveKeystrokes` | todo |
| `App::intercept_keystrokes` | method | 2857 |  | `(App).InterceptKeystrokes` | todo |
| `App::bind_keys` | method | 2869 |  | `(App).BindKeys` | todo |
| `App::clear_key_bindings` | method | 2875 |  | `(App).ClearKeyBindings` | todo |
| `App::key_bindings` | method | 2881 |  | `(App).KeyBindings` | todo |
| `App::on_action` | method | 2888 |  | `(App).OnAction` | todo |
| `App::stop_propagation` | method | 2908 |  | `(App).StopPropagation` | todo |
| `App::propagate` | method | 2916 |  | `(App).Propagate` | todo |
| `App::build_action` | method | 2921 |  | `(App).BuildAction` | todo |
| `App::all_action_names` | method | 2931 |  | `(App).AllActionNames` | todo |
| `App::all_bindings_for_input` | method | 2938 |  | `(App).AllBindingsForInput` | todo |
| `App::action_schemas` | method | 2943 |  | `(App).ActionSchemas` | todo |
| `App::action_schema_by_name` | method | 2954 |  | `(App).ActionSchemaByName` | todo |
| `App::deprecated_actions_to_preferred_actions` | method | 2963 |  | `(App).DeprecatedActionsToPreferredActions` | todo |
| `App::action_deprecation_messages` | method | 2968 |  | `(App).ActionDeprecationMessages` | todo |
| `App::action_documentation` | method | 2973 |  | `(App).ActionDocumentation` | todo |
| `App::on_app_quit` | method | 2979 |  | `(App).OnAppQuit` | todo |
| `App::on_app_restart` | method | 2995 |  | `(App).OnAppRestart` | todo |
| `App::on_window_closed` | method | 3004 |  | `(App).OnWindowClosed` | todo |
| `App::is_action_available` | method | 3023 |  | `(App).IsActionAvailable` | todo |
| `App::set_menus` | method | 3039 |  | `(App).SetMenus` | todo |
| `App::get_menus` | method | 3045 |  | `(App).GetMenus` | todo |
| `App::set_dock_menu` | method | 3050 |  | `(App).SetDockMenu` | todo |
| `App::perform_dock_menu_action` | method | 3055 |  | `(App).PerformDockMenuAction` | todo |
| `App::add_recent_document` | method | 3063 |  | `(App).AddRecentDocument` | todo |
| `App::update_jump_list` | method | 3069 |  | `(App).UpdateJumpList` | todo |
| `App::dispatch_action` | method | 3079 |  | `(App).DispatchAction` | todo |
| `App::has_active_drag` | method | 3139 |  | `(App).HasActiveDrag` | todo |
| `App::active_drag_cursor_style` | method | 3144 |  | `(App).ActiveDragCursorStyle` | todo |
| `App::stop_active_drag` | method | 3149 |  | `(App).StopActiveDrag` | todo |
| `App::set_active_drag_cursor_style` | method | 3223 |  | `(App).SetActiveDragCursorStyle` | todo |
| `App::set_prompt_builder` | method | 3239 |  | `(App).SetPromptBuilder` | todo |
| `App::reset_prompt_builder` | method | 3256 |  | `(App).ResetPromptBuilder` | todo |
| `App::remove_asset` | method | 3261 |  | `(App).RemoveAsset` | todo |
| `App::has_asset` | method | 3269 | feat:test-support | `(App).HasAsset` | todo |
| `App::fetch_asset` | method | 3278 |  | `(App).FetchAsset` | todo |
| `App::focus_handle` | method | 3298 |  | `(App).FocusHandle` | todo |
| `App::notify` | method | 3303 |  | `(App).Notify` | todo |
| `App::get_name` | method | 3341 | feat:test-support,debug | `(App).GetName` | todo |
| `App::can_select_mixed_files_and_dirs` | method | 3346 |  | `(App).CanSelectMixedFilesAndDirs` | todo |
| `App::drop_image` | method | 3354 |  | `(App).DropImage` | todo |
| `App::set_inspector_renderer` | method | 3368 | feat:inspector,debug | `(App).SetInspectorRenderer` | todo |
| `App::register_inspector_element` | method | 3374 | feat:inspector,debug | `(App).RegisterInspectorElement` | todo |
| `App::init_colors` | method | 3386 |  | `(App).InitColors` | todo |
| `AnyDrag` | struct | 3676 |  | `AnyDrag` | todo |
| `ExternalDragPayloadSource` | type | 3697 |  | `ExternalDragPayloadSource` | todo |
| `AnyTooltip` | struct | 3703 |  | `AnyTooltip` | todo |
| `InputPreference` | enum | 3718 |  | `InputPreference` | todo |
| `KeystrokeEvent` | struct | 3727 |  | `KeystrokeEvent` | todo |
| `GpuiBorrow` | struct | 3773 |  | `GpuiBorrow` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `app_borrow_diagnostics` | gpui::test | 3842 |  | `TestApp_AppBorrowDiagnostics` | todo |
| `async_app_refresh_flushes_refresh_effect` | gpui::test | 3900 |  | `TestApp_AsyncAppRefreshFlushesRefreshEffect` | todo |
| `missing_glyph_callbacks_follow_subscription_lifetime` | gpui::test | 3917 |  | `TestApp_MissingGlyphCallbacksFollowSubscriptionLifetime` | todo |
| `test_gpui_borrow` | test | 3976 |  | `TestApp_GpuiBorrow` | todo |
| `test_restart_preserves_path_and_arguments` | gpui::test | 4007 |  | `TestApp_RestartPreservesPathAndArguments` | todo |

## gpui/src/asset_cache.rs

Go target: `asset_cache.go` (package `gpui`), tests in `asset_cache_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Resource` | enum | 71 |  | `Resource` | todo |
| `Asset` | trait | 99 |  | `Asset` | todo |
| `Asset::load` | trait-method | 107 |  | `Asset.Load` | todo |
| `AssetLogger` | enum | 114 |  | `AssetLogger` | todo |
| `hash` | fn | 139 |  | `Hash` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `cached_load_caches_success_and_error` | gpui::test | 160 |  | `TestAssetCache_CachedLoadCachesSuccessAndError` | todo |
| `completed_load_without_subscribers_is_cached_and_deduplicated` | gpui::test | 178 |  | `TestAssetCache_CompletedLoadWithoutSubscribersIsCachedAndDeduplicated` | todo |
| `load_completion_notifies_each_observing_entity_once` | gpui::test | 201 |  | `TestAssetCache_LoadCompletionNotifiesEachObservingEntityOnce` | todo |
| `removing_pending_asset_cancels_it_and_replacement_ignores_stale_completion` | gpui::test | 241 |  | `TestAssetCache_RemovingPendingAssetCancelsItAndReplacementIgnoresStaleCompletion` | todo |
| `eviction_after_worker_completion_does_not_publish_into_replacement` | gpui::test | 271 |  | `TestAssetCache_EvictionAfterWorkerCompletionDoesNotPublishIntoReplacement` | todo |

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
| `sanity_test_tasks_run` | test | 602 |  | `TestExecutor_SanityTestTasksRun` | blocked:M0-test-platform |

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
| `test_unsubscribe_during_callback_with_insert` | test | 416 |  | `TestSubscription_UnsubscribeDuringCallbackWithInsert` | blocked:M0-test-contexts |
| `test_callback_dropped_by_earlier_callback_does_not_fire` | test | 467 |  | `TestSubscription_CallbackDroppedByEarlierCallbackDoesNotFire` | blocked:M0-test-contexts |
| `test_self_drop_during_callback` | test | 507 |  | `TestSubscription_SelfDropDuringCallback` | blocked:M0-test-contexts |
| `test_subscription_drop` | test | 536 |  | `TestSubscription_SubscriptionDrop` | blocked:M0-test-contexts |

## gpui/src/test.rs

Go target: `testing.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `seed_strategy` | fn | 43 |  | `SeedStrategy` | todo |
| `apply_seed_to_proptest_config` | fn | 57 |  | `ApplySeedToProptestConfig` | todo |
| `run_test_once` | fn | 74 |  | `RunTestOnce` | todo |
| `run_test` | fn | 95 |  | `RunTest` | todo |
| `Observation` | struct | 145 |  | `Observation` | todo |
| `observe` | fn | 162 |  | `Observe` | todo |

## gpui/src/app/async_context.rs

Go target: `app_async_context.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AsyncApp` | struct | 22 |  | `AsyncApp` | todo |
| `AsyncApp::refresh` | method | 150 |  | `(AsyncApp).Refresh` | todo |
| `AsyncApp::background_executor` | method | 159 |  | `(AsyncApp).BackgroundExecutor` | todo |
| `AsyncApp::is_test` | method | 165 |  | `(AsyncApp).IsTest` | todo |
| `AsyncApp::foreground_executor` | method | 170 |  | `(AsyncApp).ForegroundExecutor` | todo |
| `AsyncApp::update` | method | 176 |  | `(AsyncApp).Update` | todo |
| `AsyncApp::subscribe` | method | 184 |  | `(AsyncApp).Subscribe` | todo |
| `AsyncApp::open_window` | method | 199 |  | `(AsyncApp).OpenWindow` | todo |
| `AsyncApp::spawn` | method | 218 |  | `(AsyncApp).Spawn` | todo |
| `AsyncApp::has_global` | method | 229 |  | `(AsyncApp).HasGlobal` | todo |
| `AsyncApp::read_global` | method | 239 |  | `(AsyncApp).ReadGlobal` | todo |
| `AsyncApp::try_read_global` | method | 248 |  | `(AsyncApp).TryReadGlobal` | todo |
| `AsyncApp::read_default_global` | method | 259 |  | `(AsyncApp).ReadDefaultGlobal` | todo |
| `AsyncApp::update_global` | method | 273 |  | `(AsyncApp).UpdateGlobal` | todo |
| `AsyncApp::on_drop` | assoc-fn | 280 |  | `AsyncAppOnDrop` | todo |
| `AsyncWindowContext` | struct | 296 |  | `AsyncWindowContext` | todo |
| `AsyncWindowContext::window_handle` | method | 309 |  | `(AsyncWindowContext).WindowHandle` | todo |
| `AsyncWindowContext::update` | method | 316 |  | `(AsyncWindowContext).Update` | todo |
| `AsyncWindowContext::update_root` | method | 324 |  | `(AsyncWindowContext).UpdateRoot` | todo |
| `AsyncWindowContext::on_next_frame` | method | 332 |  | `(AsyncWindowContext).OnNextFrame` | todo |
| `AsyncWindowContext::read_global` | method | 339 |  | `(AsyncWindowContext).ReadGlobal` | todo |
| `AsyncWindowContext::update_global` | method | 349 |  | `(AsyncWindowContext).UpdateGlobal` | todo |
| `AsyncWindowContext::spawn` | method | 365 |  | `(AsyncWindowContext).Spawn` | todo |
| `AsyncWindowContext::prompt` | method | 378 |  | `(AsyncWindowContext).Prompt` | todo |

## gpui/src/app/context.rs

Go target: `app_context.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Context` | struct | 20 |  | `Context` | todo |
| `Context::entity_id` | method | 45 |  | `(Context).EntityID` | todo |
| `Context::entity` | method | 50 |  | `(Context).Entity` | todo |
| `Context::weak_entity` | method | 57 |  | `(Context).WeakEntity` | todo |
| `Context::observe` | method | 63 |  | `(Context).Observe` | todo |
| `Context::observe_self` | method | 79 |  | `(Context).ObserveSelf` | todo |
| `Context::subscribe` | method | 93 |  | `(Context).Subscribe` | todo |
| `Context::subscribe_self` | method | 111 |  | `(Context).SubscribeSelf` | todo |
| `Context::on_release` | method | 126 |  | `(Context).OnRelease` | todo |
| `Context::observe_release` | method | 140 |  | `(Context).ObserveRelease` | todo |
| `Context::observe_global` | method | 163 |  | `(Context).ObserveGlobal` | todo |
| `Context::on_app_restart` | method | 180 |  | `(Context).OnAppRestart` | todo |
| `Context::on_app_quit` | method | 195 |  | `(Context).OnAppQuit` | todo |
| `Context::notify` | method | 216 |  | `(Context).Notify` | todo |
| `Context::spawn` | method | 225 |  | `(Context).Spawn` | todo |
| `Context::listener` | method | 241 |  | `(Context).Listener` | todo |
| `Context::processor` | method | 255 |  | `(Context).Processor` | todo |
| `Context::on_drop` | method | 266 |  | `(Context).OnDrop` | todo |
| `Context::focus_view` | method | 278 |  | `(Context).FocusView` | todo |
| `Context::on_next_frame` | method | 283 |  | `(Context).OnNextFrame` | todo |
| `Context::defer_in` | method | 296 |  | `(Context).DeferIn` | todo |
| `Context::observe_in` | method | 312 |  | `(Context).ObserveIn` | todo |
| `Context::subscribe_in` | method | 336 |  | `(Context).SubscribeIn` | todo |
| `Context::on_release_in` | method | 365 |  | `(Context).OnReleaseIn` | todo |
| `Context::observe_release_in` | method | 375 |  | `(Context).ObserveReleaseIn` | todo |
| `Context::observe_window_bounds` | method | 397 |  | `(Context).ObserveWindowBounds` | todo |
| `Context::observe_window_activation` | method | 410 |  | `(Context).ObserveWindowActivation` | todo |
| `Context::observe_window_visibility` | method | 426 |  | `(Context).ObserveWindowVisibility` | todo |
| `Context::observe_window_appearance` | method | 441 |  | `(Context).ObserveWindowAppearance` | todo |
| `Context::observe_window_display` | method | 458 |  | `(Context).ObserveWindowDisplay` | todo |
| `Context::observe_button_layout_changed` | method | 471 |  | `(Context).ObserveButtonLayoutChanged` | todo |
| `Context::observe_keystrokes` | method | 486 |  | `(Context).ObserveKeystrokes` | todo |
| `Context::observe_pending_input` | method | 503 |  | `(Context).ObservePendingInput` | todo |
| `Context::on_focus` | method | 519 |  | `(Context).OnFocus` | todo |
| `Context::on_focus_in` | method | 545 |  | `(Context).OnFocusIn` | todo |
| `Context::on_blur` | method | 568 |  | `(Context).OnBlur` | todo |
| `Context::on_focus_lost` | method | 595 |  | `(Context).OnFocusLost` | todo |
| `Context::on_focus_out` | method | 614 |  | `(Context).OnFocusOut` | todo |
| `Context::spawn_in` | method | 649 |  | `(Context).SpawnIn` | todo |
| `Context::spawn_in_with_priority` | method | 664 |  | `(Context).SpawnInWithPriority` | todo |
| `Context::observe_global_in` | method | 679 |  | `(Context).ObserveGlobalIn` | todo |
| `Context::on_action` | method | 709 |  | `(Context).OnAction` | todo |
| `Context::focus_self` | method | 726 |  | `(Context).FocusSelf` | todo |
| `Context::emit` | method | 739 |  | `(Context).Emit` | todo |

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
| `HeadlessAppContext` | struct | 38 |  | `HeadlessAppContext` | todo |
| `HeadlessAppContext::new` | assoc-fn | 51 |  | `NewHeadlessAppContext` | todo |
| `HeadlessAppContext::with_asset_source` | assoc-fn | 56 |  | `HeadlessAppContextWithAssetSource` | todo |
| `HeadlessAppContext::with_platform` | assoc-fn | 65 |  | `HeadlessAppContextWithPlatform` | todo |
| `HeadlessAppContext::open_window` | method | 104 |  | `(HeadlessAppContext).OpenWindow` | todo |
| `HeadlessAppContext::run_until_parked` | method | 129 |  | `(HeadlessAppContext).RunUntilParked` | todo |
| `HeadlessAppContext::simulate_window_scale_factor_change` | method | 134 |  | `(HeadlessAppContext).SimulateWindowScaleFactorChange` | todo |
| `HeadlessAppContext::advance_clock` | method | 143 |  | `(HeadlessAppContext).AdvanceClock` | todo |
| `HeadlessAppContext::allow_parking` | method | 148 |  | `(HeadlessAppContext).AllowParking` | todo |
| `HeadlessAppContext::forbid_parking` | method | 153 |  | `(HeadlessAppContext).ForbidParking` | todo |
| `HeadlessAppContext::update` | method | 158 |  | `(HeadlessAppContext).Update` | todo |
| `HeadlessAppContext::update_window` | method | 164 |  | `(HeadlessAppContext).UpdateWindow` | todo |
| `HeadlessAppContext::capture_screenshot` | method | 177 |  | `(HeadlessAppContext).CaptureScreenshot` | todo |
| `HeadlessAppContext::text_system` | method | 183 |  | `(HeadlessAppContext).TextSystem` | todo |
| `HeadlessAppContext::background_executor` | method | 188 |  | `(HeadlessAppContext).BackgroundExecutor` | todo |
| `HeadlessAppContext::foreground_executor` | method | 193 |  | `(HeadlessAppContext).ForegroundExecutor` | todo |

## gpui/src/app/test_app.rs

Go target: `app_test_app.go` (package `gpui`), tests in `app_test_app_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestApp` | struct | 40 |  | `TestApp` | todo |
| `TestApp::new` | assoc-fn | 52 |  | `NewTestApp` | todo |
| `TestApp::with_seed` | assoc-fn | 57 |  | `TestAppWithSeed` | todo |
| `TestApp::with_text_system` | assoc-fn | 62 |  | `TestAppWithTextSystem` | todo |
| `TestApp::with_text_system_and_assets` | assoc-fn | 67 |  | `TestAppWithTextSystemAndAssets` | todo |
| `TestApp::update` | method | 111 |  | `(TestApp).Update` | todo |
| `TestApp::read` | method | 121 |  | `(TestApp).Read` | todo |
| `TestApp::new_entity` | method | 127 |  | `(TestApp).NewEntity` | todo |
| `TestApp::update_entity` | method | 135 |  | `(TestApp).UpdateEntity` | todo |
| `TestApp::read_entity` | method | 144 |  | `(TestApp).ReadEntity` | todo |
| `TestApp::open_window` | method | 153 |  | `(TestApp).OpenWindow` | todo |
| `TestApp::open_window_with_options` | method | 178 |  | `(TestApp).OpenWindowWithOptions` | todo |
| `TestApp::run_until_parked` | method | 197 |  | `(TestApp).RunUntilParked` | todo |
| `TestApp::advance_clock` | method | 202 |  | `(TestApp).AdvanceClock` | todo |
| `TestApp::spawn` | method | 207 |  | `(TestApp).Spawn` | todo |
| `TestApp::background_spawn` | method | 216 |  | `(TestApp).BackgroundSpawn` | todo |
| `TestApp::to_async` | method | 224 |  | `(TestApp).ToAsync` | todo |
| `TestApp::background_executor` | method | 233 |  | `(TestApp).BackgroundExecutor` | todo |
| `TestApp::foreground_executor` | method | 238 |  | `(TestApp).ForegroundExecutor` | todo |
| `TestApp::text_system` | method | 243 |  | `(TestApp).TextSystem` | todo |
| `TestApp::has_global` | method | 248 |  | `(TestApp).HasGlobal` | todo |
| `TestApp::set_global` | method | 253 |  | `(TestApp).SetGlobal` | todo |
| `TestApp::read_global` | method | 258 |  | `(TestApp).ReadGlobal` | todo |
| `TestApp::update_global` | method | 263 |  | `(TestApp).UpdateGlobal` | todo |
| `TestApp::write_to_clipboard` | method | 270 |  | `(TestApp).WriteToClipboard` | todo |
| `TestApp::read_from_clipboard` | method | 275 |  | `(TestApp).ReadFromClipboard` | todo |
| `TestApp::opened_url` | method | 280 |  | `(TestApp).OpenedURL` | todo |
| `TestApp::did_prompt_for_new_path` | method | 285 |  | `(TestApp).DidPromptForNewPath` | todo |
| `TestApp::simulate_new_path_selection` | method | 290 |  | `(TestApp).SimulateNewPathSelection` | todo |
| `TestApp::has_pending_prompt` | method | 298 |  | `(TestApp).HasPendingPrompt` | todo |
| `TestApp::simulate_prompt_answer` | method | 303 |  | `(TestApp).SimulatePromptAnswer` | todo |
| `TestApp::windows` | method | 308 |  | `(TestApp).Windows` | todo |
| `TestAppWindow` | struct | 320 |  | `TestAppWindow` | todo |
| `TestAppWindow::handle` | method | 329 |  | `(TestAppWindow).Handle` | todo |
| `TestAppWindow::root` | method | 334 |  | `(TestAppWindow).Root` | todo |
| `TestAppWindow::update` | method | 344 |  | `(TestAppWindow).Update` | todo |
| `TestAppWindow::read` | method | 359 |  | `(TestAppWindow).Read` | todo |
| `TestAppWindow::title` | method | 374 |  | `(TestAppWindow).Title` | todo |
| `TestAppWindow::simulate_keystroke` | method | 384 |  | `(TestAppWindow).SimulateKeystroke` | todo |
| `TestAppWindow::simulate_keystrokes` | method | 399 |  | `(TestAppWindow).SimulateKeystrokes` | todo |
| `TestAppWindow::simulate_input` | method | 406 |  | `(TestAppWindow).SimulateInput` | todo |
| `TestAppWindow::simulate_mouse_move` | method | 413 |  | `(TestAppWindow).SimulateMouseMove` | todo |
| `TestAppWindow::simulate_mouse_down` | method | 422 |  | `(TestAppWindow).SimulateMouseDown` | todo |
| `TestAppWindow::simulate_mouse_up` | method | 433 |  | `(TestAppWindow).SimulateMouseUp` | todo |
| `TestAppWindow::simulate_click` | method | 443 |  | `(TestAppWindow).SimulateClick` | todo |
| `TestAppWindow::simulate_scroll` | method | 449 |  | `(TestAppWindow).SimulateScroll` | todo |
| `TestAppWindow::simulate_event` | method | 460 |  | `(TestAppWindow).SimulateEvent` | todo |
| `TestAppWindow::simulate_resize` | method | 474 |  | `(TestAppWindow).SimulateResize` | todo |
| `TestAppWindow::simulate_scale_factor_change` | method | 487 |  | `(TestAppWindow).SimulateScaleFactorChange` | todo |
| `TestAppWindow::draw` | method | 505 |  | `(TestAppWindow).Draw` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_basic_usage` | test | 563 |  | `TestTestApp_BasicUsage` | todo |
| `test_simulate_scale_factor_change` | test | 581 |  | `TestTestApp_SimulateScaleFactorChange` | todo |
| `test_entity_creation` | test | 601 |  | `TestTestApp_EntityCreation` | todo |
| `test_globals` | test | 623 |  | `TestTestApp_Globals` | todo |

## gpui/src/app/test_context.rs

Go target: `app_test_context.go` (package `gpui`), tests in `app_test_context_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TestAppContext` | struct | 35 |  | `TestAppContext` | todo |
| `TestAppContext::set_platform_info` | method | 141 |  | `(TestAppContext).SetPlatformInfo` | todo |
| `TestAppContext::build` | assoc-fn | 146 |  | `TestAppContextBuild` | todo |
| `TestAppContext::build_with_text_system` | assoc-fn | 161 |  | `TestAppContextBuildWithTextSystem` | todo |
| `TestAppContext::disable_accessibility` | method | 210 |  | `(TestAppContext).DisableAccessibility` | todo |
| `TestAppContext::skip_drawing` | method | 215 |  | `(TestAppContext).SkipDrawing` | todo |
| `TestAppContext::single` | assoc-fn | 220 |  | `TestAppContextSingle` | todo |
| `TestAppContext::test_function_name` | method | 226 |  | `(TestAppContext).TestFunctionName` | todo |
| `TestAppContext::did_prompt_for_new_path` | method | 231 |  | `(TestAppContext).DidPromptForNewPath` | todo |
| `TestAppContext::active_idle_sleep_preventions` | method | 236 |  | `(TestAppContext).ActiveIdleSleepPreventions` | todo |
| `TestAppContext::set_idle_sleep_prevention_delay` | method | 241 |  | `(TestAppContext).SetIdleSleepPreventionDelay` | todo |
| `TestAppContext::set_idle_sleep_prevention_fails` | method | 246 |  | `(TestAppContext).SetIdleSleepPreventionFails` | todo |
| `TestAppContext::new_app` | method | 251 |  | `(TestAppContext).NewApp` | todo |
| `TestAppContext::quit` | method | 257 |  | `(TestAppContext).Quit` | todo |
| `TestAppContext::on_quit` | method | 263 |  | `(TestAppContext).OnQuit` | todo |
| `TestAppContext::refresh` | method | 268 |  | `(TestAppContext).Refresh` | todo |
| `TestAppContext::executor` | method | 275 |  | `(TestAppContext).Executor` | todo |
| `TestAppContext::foreground_executor` | method | 280 |  | `(TestAppContext).ForegroundExecutor` | todo |
| `TestAppContext::update` | method | 285 |  | `(TestAppContext).Update` | todo |
| `TestAppContext::read` | method | 291 |  | `(TestAppContext).Read` | todo |
| `TestAppContext::add_window` | method | 298 |  | `(TestAppContext).AddWindow` | todo |
| `TestAppContext::open_window` | method | 321 |  | `(TestAppContext).OpenWindow` | todo |
| `TestAppContext::add_empty_window` | method | 345 |  | `(TestAppContext).AddEmptyWindow` | todo |
| `TestAppContext::add_window_view` | method | 366 |  | `(TestAppContext).AddWindowView` | todo |
| `TestAppContext::text_system` | method | 395 |  | `(TestAppContext).TextSystem` | todo |
| `TestAppContext::write_to_clipboard` | method | 400 |  | `(TestAppContext).WriteToClipboard` | todo |
| `TestAppContext::read_from_clipboard` | method | 406 |  | `(TestAppContext).ReadFromClipboard` | todo |
| `TestAppContext::simulate_new_path_selection` | method | 411 |  | `(TestAppContext).SimulateNewPathSelection` | todo |
| `TestAppContext::simulate_path_prompt_response` | method | 419 |  | `(TestAppContext).SimulatePathPromptResponse` | todo |
| `TestAppContext::did_prompt_for_paths` | method | 428 |  | `(TestAppContext).DidPromptForPaths` | todo |
| `TestAppContext::simulate_prompt_answer` | method | 434 |  | `(TestAppContext).SimulatePromptAnswer` | todo |
| `TestAppContext::has_pending_prompt` | method | 439 |  | `(TestAppContext).HasPendingPrompt` | todo |
| `TestAppContext::pending_prompt` | method | 444 |  | `(TestAppContext).PendingPrompt` | todo |
| `TestAppContext::opened_url` | method | 449 |  | `(TestAppContext).OpenedURL` | todo |
| `TestAppContext::app_identity` | method | 454 |  | `(TestAppContext).AppIdentity` | todo |
| `TestAppContext::shown_system_notifications` | method | 459 |  | `(TestAppContext).ShownSystemNotifications` | todo |
| `TestAppContext::delivered_system_notifications` | method | 464 |  | `(TestAppContext).DeliveredSystemNotifications` | todo |
| `TestAppContext::dismissed_system_notifications` | method | 469 |  | `(TestAppContext).DismissedSystemNotifications` | todo |
| `TestAppContext::simulate_system_notification_response` | method | 474 |  | `(TestAppContext).SimulateSystemNotificationResponse` | todo |
| `TestAppContext::simulate_window_resize` | method | 480 |  | `(TestAppContext).SimulateWindowResize` | todo |
| `TestAppContext::simulate_window_visibility_change` | method | 485 |  | `(TestAppContext).SimulateWindowVisibilityChange` | todo |
| `TestAppContext::simulate_window_decorations_change` | method | 495 |  | `(TestAppContext).SimulateWindowDecorationsChange` | todo |
| `TestAppContext::simulate_display_added` | method | 505 |  | `(TestAppContext).SimulateDisplayAdded` | todo |
| `TestAppContext::simulate_display_removed` | method | 520 |  | `(TestAppContext).SimulateDisplayRemoved` | todo |
| `TestAppContext::simulate_display_refresh_interval_change` | method | 525 |  | `(TestAppContext).SimulateDisplayRefreshIntervalChange` | todo |
| `TestAppContext::simulate_window_visual_viewport_change` | method | 535 |  | `(TestAppContext).SimulateWindowVisualViewportChange` | todo |
| `TestAppContext::simulate_window_scale_factor_change` | method | 545 |  | `(TestAppContext).SimulateWindowScaleFactorChange` | todo |
| `TestAppContext::expect_restart` | method | 555 |  | `(TestAppContext).ExpectRestart` | todo |
| `TestAppContext::set_screen_capture_sources` | method | 563 |  | `(TestAppContext).SetScreenCaptureSources` | todo |
| `TestAppContext::windows` | method | 568 |  | `(TestAppContext).Windows` | todo |
| `TestAppContext::spawn` | method | 574 |  | `(TestAppContext).Spawn` | todo |
| `TestAppContext::has_global` | method | 583 |  | `(TestAppContext).HasGlobal` | todo |
| `TestAppContext::read_global` | method | 590 |  | `(TestAppContext).ReadGlobal` | todo |
| `TestAppContext::try_read_global` | method | 596 |  | `(TestAppContext).TryReadGlobal` | todo |
| `TestAppContext::set_global` | method | 602 |  | `(TestAppContext).SetGlobal` | todo |
| `TestAppContext::update_global` | method | 608 |  | `(TestAppContext).UpdateGlobal` | todo |
| `TestAppContext::to_async` | method | 615 |  | `(TestAppContext).ToAsync` | todo |
| `TestAppContext::run_until_parked` | method | 624 |  | `(TestAppContext).RunUntilParked` | todo |
| `TestAppContext::dispatch_action` | method | 629 |  | `(TestAppContext).DispatchAction` | todo |
| `TestAppContext::simulate_keystrokes` | method | 646 |  | `(TestAppContext).SimulateKeystrokes` | todo |
| `TestAppContext::simulate_input` | method | 663 |  | `(TestAppContext).SimulateInput` | todo |
| `TestAppContext::dispatch_keystroke` | method | 677 |  | `(TestAppContext).DispatchKeystroke` | todo |
| `TestAppContext::notifications` | method | 690 |  | `(TestAppContext).Notifications` | todo |
| `TestAppContext::events` | method | 710 |  | `(TestAppContext).Events` | todo |
| `TestAppContext::condition` | method | 730 |  | `(TestAppContext).Condition` | todo |
| `TestAppContext::set_name` | method | 762 | feat:test-support | `(TestAppContext).SetName` | todo |
| `Entity::next_event` | method | 769 |  | `(Entity).NextEvent` | todo |
| `Entity::next_notification` | method | 794 |  | `(Entity).NextNotification` | todo |
| `Entity::condition` | method | 819 |  | `(Entity).Condition` | todo |
| `VisualTestContext` | struct | 882 |  | `VisualTestContext` | todo |
| `VisualTestContext::update` | method | 892 |  | `(VisualTestContext).Update` | todo |
| `VisualTestContext::from_window` | assoc-fn | 901 |  | `VisualTestContextFromWindow` | todo |
| `VisualTestContext::run_until_parked` | method | 909 |  | `(VisualTestContext).RunUntilParked` | todo |
| `VisualTestContext::dispatch_action` | method | 914 |  | `(VisualTestContext).DispatchAction` | todo |
| `VisualTestContext::window_title` | method | 922 |  | `(VisualTestContext).WindowTitle` | todo |
| `VisualTestContext::document_path` | method | 927 |  | `(VisualTestContext).DocumentPath` | todo |
| `VisualTestContext::simulate_keystrokes` | method | 938 |  | `(VisualTestContext).SimulateKeystrokes` | todo |
| `VisualTestContext::simulate_input` | method | 944 |  | `(VisualTestContext).SimulateInput` | todo |
| `VisualTestContext::simulate_mouse_move` | method | 949 |  | `(VisualTestContext).SimulateMouseMove` | todo |
| `VisualTestContext::simulate_mouse_down` | method | 963 |  | `(VisualTestContext).SimulateMouseDown` | todo |
| `VisualTestContext::simulate_mouse_up` | method | 979 |  | `(VisualTestContext).SimulateMouseUp` | todo |
| `VisualTestContext::simulate_click` | method | 994 |  | `(VisualTestContext).SimulateClick` | todo |
| `VisualTestContext::simulate_modifiers_change` | method | 1011 |  | `(VisualTestContext).SimulateModifiersChange` | todo |
| `VisualTestContext::simulate_capslock_change` | method | 1019 |  | `(VisualTestContext).SimulateCapslockChange` | todo |
| `VisualTestContext::simulate_resize` | method | 1027 |  | `(VisualTestContext).SimulateResize` | todo |
| `VisualTestContext::simulate_visibility_change` | method | 1032 |  | `(VisualTestContext).SimulateVisibilityChange` | todo |
| `VisualTestContext::simulate_scale_factor_change` | method | 1037 |  | `(VisualTestContext).SimulateScaleFactorChange` | todo |
| `VisualTestContext::debug_bounds` | method | 1045 |  | `(VisualTestContext).DebugBounds` | todo |
| `VisualTestContext::draw` | method | 1050 |  | `(VisualTestContext).Draw` | todo |
| `VisualTestContext::simulate_event` | method | 1082 |  | `(VisualTestContext).SimulateEvent` | todo |
| `VisualTestContext::deactivate_window` | method | 1089 |  | `(VisualTestContext).DeactivateWindow` | todo |
| `VisualTestContext::simulate_close` | method | 1098 |  | `(VisualTestContext).SimulateClose` | todo |
| `VisualTestContext::into_mut` | method | 1127 |  | `(VisualTestContext).IntoMut` | todo |
| `AnyWindowHandle::build_entity` | method | 1290 |  | `(AnyWindowHandle).BuildEntity` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_mobile_lifecycle_subscriptions` | gpui::test | 1311 |  | `TestTestContext_MobileLifecycleSubscriptions` | todo |
| `test_memory_warning_subscriptions` | gpui::test | 1356 |  | `TestTestContext_MemoryWarningSubscriptions` | todo |
| `test_system_notifications_require_identity_and_replace_matching_tags` | gpui::test | 1379 |  | `TestTestContext_SystemNotificationsRequireIdentityAndReplaceMatchingTags` | todo |
| `test_system_notification_body_and_action_responses` | gpui::test | 1436 |  | `TestTestContext_SystemNotificationBodyAndActionResponses` | todo |
| `test_system_notification_response_handler_can_be_replaced` | gpui::test | 1470 |  | `TestTestContext_SystemNotificationResponseHandlerCanBeReplaced` | todo |
| `test_system_notification_response_handler_can_reenter_app` | gpui::test | 1495 |  | `TestTestContext_SystemNotificationResponseHandlerCanReenterApp` | todo |
| `test_simulate_path_prompt_response` | gpui::test | 1519 |  | `TestTestContext_SimulatePathPromptResponse` | todo |
| `test_simulate_path_prompt_cancellation` | gpui::test | 1547 |  | `TestTestContext_SimulatePathPromptCancellation` | todo |

## gpui/src/app/visual_test_context.rs

Go target: `app_visual_test_context.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `VisualTestAppContext` | struct | 21 |  | `VisualTestAppContext` | todo |
| `VisualTestAppContext::new` | assoc-fn | 45 |  | `NewVisualTestAppContext` | todo |
| `VisualTestAppContext::with_asset_source` | assoc-fn | 53 |  | `VisualTestAppContextWithAssetSource` | todo |
| `VisualTestAppContext::open_offscreen_window` | method | 97 |  | `(VisualTestAppContext).OpenOffscreenWindow` | todo |
| `VisualTestAppContext::open_offscreen_window_default` | method | 122 |  | `(VisualTestAppContext).OpenOffscreenWindowDefault` | todo |
| `VisualTestAppContext::is_screen_capture_supported` | method | 131 |  | `(VisualTestAppContext).IsScreenCaptureSupported` | todo |
| `VisualTestAppContext::text_system` | method | 136 |  | `(VisualTestAppContext).TextSystem` | todo |
| `VisualTestAppContext::executor` | method | 141 |  | `(VisualTestAppContext).Executor` | todo |
| `VisualTestAppContext::foreground_executor` | method | 146 |  | `(VisualTestAppContext).ForegroundExecutor` | todo |
| `VisualTestAppContext::run_until_parked` | method | 152 |  | `(VisualTestAppContext).RunUntilParked` | todo |
| `VisualTestAppContext::advance_clock` | method | 159 |  | `(VisualTestAppContext).AdvanceClock` | todo |
| `VisualTestAppContext::update` | method | 164 |  | `(VisualTestAppContext).Update` | todo |
| `VisualTestAppContext::read` | method | 170 |  | `(VisualTestAppContext).Read` | todo |
| `VisualTestAppContext::update_window` | method | 176 |  | `(VisualTestAppContext).UpdateWindow` | todo |
| `VisualTestAppContext::spawn` | method | 185 |  | `(VisualTestAppContext).Spawn` | todo |
| `VisualTestAppContext::has_global` | method | 194 |  | `(VisualTestAppContext).HasGlobal` | todo |
| `VisualTestAppContext::read_global` | method | 200 |  | `(VisualTestAppContext).ReadGlobal` | todo |
| `VisualTestAppContext::set_global` | method | 206 |  | `(VisualTestAppContext).SetGlobal` | todo |
| `VisualTestAppContext::update_global` | method | 212 |  | `(VisualTestAppContext).UpdateGlobal` | todo |
| `VisualTestAppContext::simulate_keystrokes` | method | 225 |  | `(VisualTestAppContext).SimulateKeystrokes` | todo |
| `VisualTestAppContext::dispatch_keystroke` | method | 236 |  | `(VisualTestAppContext).DispatchKeystroke` | todo |
| `VisualTestAppContext::simulate_input` | method | 244 |  | `(VisualTestAppContext).SimulateInput` | todo |
| `VisualTestAppContext::simulate_mouse_move` | method | 258 |  | `(VisualTestAppContext).SimulateMouseMove` | todo |
| `VisualTestAppContext::simulate_mouse_down` | method | 276 |  | `(VisualTestAppContext).SimulateMouseDown` | todo |
| `VisualTestAppContext::simulate_mouse_up` | method | 296 |  | `(VisualTestAppContext).SimulateMouseUp` | todo |
| `VisualTestAppContext::simulate_click` | method | 315 |  | `(VisualTestAppContext).SimulateClick` | todo |
| `VisualTestAppContext::simulate_event` | method | 326 |  | `(VisualTestAppContext).SimulateEvent` | todo |
| `VisualTestAppContext::dispatch_action` | method | 335 |  | `(VisualTestAppContext).DispatchAction` | todo |
| `VisualTestAppContext::write_to_clipboard` | method | 344 |  | `(VisualTestAppContext).WriteToClipboard` | todo |
| `VisualTestAppContext::read_from_clipboard` | method | 349 |  | `(VisualTestAppContext).ReadFromClipboard` | todo |
| `VisualTestAppContext::wait_for` | method | 354 |  | `(VisualTestAppContext).WaitFor` | todo |
| `VisualTestAppContext::capture_screenshot` | method | 385 | feat:test-support | `(VisualTestAppContext).CaptureScreenshot` | todo |
| `VisualTestAppContext::wait_for_animations` | method | 390 |  | `(VisualTestAppContext).WaitForAnimations` | todo |


## Notes (M0-input-types)

- action.rs: `RegisterAction` only appends to a package inventory (upstream `inventory::submit!`); the registry (`actionRegistry`, built by `newActionRegistry` = `ActionRegistry::default` + `load_actions`) panics on duplicate names with upstream's text when an App builds it. Its methods are unexported (`buildActionType`, `tryResolveAction`, `buildAction(name, Option[jsontext.Value])` defaulting to `{}`, `allActionNames`, `actionSchemas`, `actionSchemaByName`, `deprecatedAliasesMap`, `deprecationMessagesMap`, `documentationMap`). Schema functions take no generator (ActionData contract); a nil `JSONSchema` is upstream's `None`.
- `NoAction` and `Unbind` are hand-written in action.go (no `action_gen.go`): names `zed::NoAction`, `zed::Unbind`; `Unbind` is `type Unbind SharedString`, so its JSON is a bare string as serde's newtype.
- `ActionBuildErrorBuildError.Unwrap` returns `errors.Unwrap(Err)`, upstream's `error.source()`.
- assets.rs: `NoAssets` is `impl AssetSource for ()`; `RenderImage.String` is the Debug impl; `renderSize` is crate-private.
- seeds.rs: `CalculateSeeds` returns `(iter.Seq[uint64], bool)`; panics "invalid ITERATIONS variable: ..." / "invalid SEED variable as integer: ...".
- entity_map.rs: crate-private items are unexported: `entityMap` (`newEntityMap`, generic `reserve` / `insert` / `lease` / `endLease` / `read` plus erased `reserveErased` / `insertErased` / `leaseErased` / `endLeaseErased` / `readErased` for the AppContext primitives, access scopes, `takeDropped() []droppedEntity`, `leakDetectorSnapshot`, `assertNoNewLeaks`, `report`), `entityRefCounts` (slotmap port `entitySlotMap`: sentinel slot 0, key `version<<32|idx`, odd version while occupied), `leaseInner`, `lease[T]`, `slot[T]`, `leakDetector`, `handleID`. Entity state is stored as `*T` in `map[EntityID]any`. `extendAccessed` takes `[]EntityID`.
- `AnyEntity.Downcast[T]() (Entity[T], bool)`: false leaves the handle with the caller (upstream's `Err(self)`). `AsNonZeroU64` returns `uint64` and panics on 0. `EntityID.String` is Display; the Debug form `EntityId(1v1)` is used in leak messages.
- Handles per DESIGN 5.2: `AnyEntity{h *entityHandle}`; `Release` is idempotent and a no-op on the zero value; every other method panics on a zero or released handle. `runtime.AddCleanup` on each handle runs upstream's Drop path when a handle is collected without `Release`. `==` on strong handles compares handle identity; compare `EntityID()` for upstream's Eq / Ord / Hash. `AnyWeakEntity` is a plain value (id, `reflect.Type`, `weak.Pointer[entityRefCounts]`).
- Leak detection (`cfg(any(test, gpui_leak_detection))`) is runtime: on in test binaries and when `GPUI_LEAK_DETECTION` is non-empty and not `0`; strict when it is `strict` or unset in a test binary. Strict mode also records entity handles, subscriptions and leases collected without `Release` / `Unsubscribe` / `Detach` / `endLease`; `report` (upstream `LeakDetector::drop`) panics with "Exited with leaked handles:" listing both. `LEAK_BACKTRACE` captures `runtime.Callers`; frames from `testing.tRunner` on are stripped as upstream strips `test::run_test_in_process`.
- subscription.rs: `subscriberSet[K comparable, C]` / `singleSubscriberSet[C]` keep one id-ordered list per emitter in a Go map (nothing iterates across emitters, so `Ord` on the key is not needed); `remove` returns an `iter.Seq[C]` that checks `Active` when iterated, as upstream's lazy `filter_map`; `retain` takes `func(*C) bool`. `Subscription` aliases one handle: `Unsubscribe` (Drop) is idempotent, `Detach` clears it, the zero value is empty. The four TestApp tests wait for `TestApp` (M0-test-contexts).

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-ACTIONDATA | The `Self: Sized` associated functions of `Action` (`name_for_type`, `build`, `action_json_schema`, `deprecated_aliases`, `deprecation_message`, `documentation`) are fields of the `ActionData` registration record (upstream's `MacroActionData`); `MacroActionBuilder` / `inventory` are `RegisterAction` from `init`; `as_any` is a Go type assertion | `gpui/src/action.rs:117-190`, `:267-290` | Go interfaces have no static methods; DESIGN 6.4 |
| D-GLOBAL | global.rs's blanket traits are free generic functions: `ReadGlobal::global` is `Global[G](cx)`, `UpdateGlobal::update_global` / `set_global` are gpui.go's `UpdateGlobal` / `UpdateGlobalR` / `SetGlobal`; the `Global` marker trait is dropped (any type can be a global) | `gpui/src/global.rs:22-75` | marker traits are dropped (DESIGN 5.4); blanket impls cannot be expressed |
| D-TASKEXT | `TaskExt`, an extension trait on `Task<Result<T, E>>`, is a pair of free generic functions `TaskDetachAndLogErr(task, cx)` / `TaskDetachAndLogErrWithBacktrace(task, cx)`, because `Task` is an alias of scheduler.Task and gets no methods in package gpui | `gpui/src/executor.rs:55-82` | executor.go |
