# Parity: windows

Scope: gpui_windows. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_windows/build.rs

Go target: `-` (package `-`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compile_shaders` | fn | 23 | windows,debug | `buildShaderBlob` | dev:D-WINSHADERRT |
| `find_latest_windows_sdk_binary` | fn | 62 | windows,debug | `-` | n/a:D-WINSHADERRT |

## gpui_windows/src/dialog.rs

Go target: `platform/windows/dialog.go` (package `windows`), tests in `dialog_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `task_dialog_cancellation_requires_cancel_button_or_internal_request` | test | 252 |  | `TestDialog_TaskDialogCancellationRequiresCancelButtonOrInternalRequest` | done |
| `closing_owner_cancels_every_outstanding_dialog` | test | 280 |  | `TestDialog_ClosingOwnerCancelsEveryOutstandingDialog` | done |
| `closed_owner_waits_for_active_dialog_but_queued_requests_stay_cancelled` | test | 296 |  | `TestDialog_ClosedOwnerWaitsForActiveDialogButQueuedRequestsStayCancelled` | done |
| `cancelling_one_request_does_not_cancel_its_owner_or_other_requests` | test | 311 |  | `TestDialog_CancellingOneRequestDoesNotCancelItsOwnerOrOtherRequests` | done |
| `dialogs_serialize_per_owner` | test | 322 |  | `TestDialog_DialogsSerializePerOwner` | done |
| `ownerless_request_can_be_cancelled_without_a_window` | test | 334 |  | `TestDialog_OwnerlessRequestCanBeCancelledWithoutAWindow` | done |

## gpui_windows/src/direct_write.rs

Go target: `platform/windows/direct_write.go` (package `windows`), tests in `direct_write_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tabs_preserve_native_advances_and_cursor` | test | 1808 |  | `TestDirectWrite_TabsPreserveNativeAdvancesAndCursor` | blocked:M0-text-elements |
| `unicode_line_separators_preserve_shaped_line_cursor` | test | 1934 |  | `TestDirectWrite_UnicodeLineSeparatorsPreserveShapedLineCursor` | blocked:M0-text-elements |
| `unicode_paragraph_separators_preserve_shaped_line_cursor` | test | 1939 |  | `TestDirectWrite_UnicodeParagraphSeparatorsPreserveShapedLineCursor` | blocked:M0-text-elements |
| `test_cluster_map` | test | 2041 |  | `TestDirectWrite_ClusterMap` | done |

## gpui_windows/src/directx_atlas.rs

Go target: `platform/windows/directx_atlas.go` (package `windows`), tests in `directx_atlas_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_remove_deallocates_tile_space_for_reuse` | test | 393 |  | `TestDirectxAtlas_RemoveDeallocatesTileSpaceForReuse` | done |

## gpui_windows/src/directx_renderer.rs

Go target: `platform/windows/directx_renderer.go` (package `windows`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AGSGPUInfo` | struct | 1991 |  | `AGSGPUInfo` | done |

## gpui_windows/src/keyboard.rs

Go target: `platform/windows/keyboard.go` (package `windows`), tests in `keyboard_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_keyboard_mapper` | test | 323 |  | `TestKeyboard_KeyboardMapper` | done |

## gpui_windows/src/platform.rs

Go target: `platform/windows/platform.go` (package `windows`), tests in `platform_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WindowsPlatform` | struct | 53 |  | `WindowsPlatform` | todo |
| `WindowsPlatform::new` | assoc-fn | 184 |  | `NewWindowsPlatform` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_read_credential_with_username` | test | 1935 |  | `TestPlatform_ReadCredentialWithUsername` | todo |
| `test_read_credential_without_username` | test | 1943 |  | `TestPlatform_ReadCredentialWithoutUsername` | todo |
| `test_encode_restart_arguments` | test | 1988 |  | `TestPlatform_EncodeRestartArguments` | todo |
| `test_clipboard` | test | 2004 |  | `TestPlatform_Clipboard` | todo |

## gpui_windows/src/vsync.rs

Go target: `platform/windows/vsync.go` (package `windows`), tests in `vsync_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `successful_compositor_wait_reports_native_callback` | test | 92 |  | `TestVsync_SuccessfulCompositorWaitReportsNativeCallback` | done |
| `failed_compositor_wait_reports_local_schedule_even_after_threshold` | test | 108 |  | `TestVsync_FailedCompositorWaitReportsLocalScheduleEvenAfterThreshold` | done |

## gpui_windows/src/window.rs

Go target: `platform/windows/window.go` (package `windows`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WindowsWindow` | struct | 35 |  | `WindowsWindow` | done |
| `WindowsWindowState` | struct | 45 |  | `WindowsWindowState` | done |
| `WindowsWindowInner` | struct | 94 |  | `WindowsWindowInner` | done |
| `Callbacks` | struct | 377 |  | `Callbacks` | done |
| `WindowsWindow::new` | fn | 418 |  | `NewWindowsWindow` | done |
| `Drop for WindowsWindow` | impl | 600 |  | `WindowsWindow.Close` | done |
| `PlatformWindow for WindowsWindow` | impl | 621 |  | `WindowsWindow` | done |
| `A11yState` | struct | 1123 |  | `A11yState` | done |
| `WindowsDragDropHandler` | struct | 1145 |  | `windowsDragDropHandler` | done |
| `ClickState` | struct | 1281 |  | `ClickState` | done |
| `WindowBorderOffset` | struct | 1386 |  | `WindowBorderOffset` | done |
| `register_window_class` | fn | 1435 |  | `registerWindowClass` | done |
| `window_procedure` | fn | 1451 |  | `windowProcedure` | done |
| `window_from_hwnd` | fn | 1494 |  | `windowFromHwnd` | done |
| `get_module_handle` | fn | 1508 |  | `getModuleHandle` | done |
| `register_drag_drop` | fn | 1522 |  | `registerDragDrop` | done |
| `calculate_window_rect` | fn | 1536 |  | `calculateWindowRect` | done |
| `calculate_client_rect` | fn | 1562 |  | `calculateClientRect` | done |
| `retrieve_window_placement` | fn | 1582 |  | `retrieveWindowPlacement` | done |
| `dwm_set_window_composition_attribute` | fn | 1610 |  | `dwmSetWindowCompositionAttribute` | done |
| `set_window_composition_attribute` | fn | 1634 |  | `setWindowCompositionAttribute` | done |
| `set_non_rude_hwnd` | fn | 1682 |  | `setNonRudeHwnd` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_double_click_interval` | test | 1695 |  | `TestWindow_DoubleClickInterval` | done |
| `test_double_click_spatial_tolerance` | test | 1723 |  | `TestWindow_DoubleClickSpatialTolerance` | done |


## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-WINVSYNCDIV | `retrieveDuration` returns `(time.Duration, error)`: a zero tick rate is an error (so `NewVSyncProvider` falls back to `defaultVsyncInterval`), and a tick rate under 1 MHz (the `rateRefresh` fallback passes `uiNumerator`, e.g. 60) is converted exactly as `counts*1e6/ticksPerSecond` microseconds; at 1 MHz and above upstream's formula is kept | `gpui_windows/src/vsync.rs:62-85` | upstream divides by `ticks_per_second / 1_000_000`, which is 0 on the fallback path and panics |
| D-WINDISPFAIL | When `TrySubmitThreadpoolCallback` or `CreateThreadpoolTimer` fails, the runnable is taken back out of the handle table and run on a goroutine (`time.AfterFunc` for the timer); a failed wake `PostMessageW` resets `WakePosted` to false | `gpui_windows/src/dispatcher.rs:54-90`, `:118-132` | upstream leaks the runnable (it cannot drop a scheduled runnable) and leaves `wake_posted` set, so the task never runs and the main queue stops being woken |
| D-WINLASTERR | `WindowsDisplayPrimaryMonitor` reads the last error captured by the `MonitorFromPoint` call itself (`win32.MonitorFromPointLastError`) | `gpui_windows/src/display.rs:85-92` | Go runtime code between the call and `GetLastError` can overwrite the thread's last error |
| D-WINSTRNUL | `com.NewHSTRING` / `NewBSTR` encode with `utf16.Encode` and `HSTRING.String` / `BSTRToString` decode the full counted length with `utf16.Decode`, so embedded NULs round-trip | windows-rs `HSTRING::from`, `BSTR::from` | as windows-rs, which never truncates or rejects interior NULs |
| D-WINSHADERRT | build.rs's fxc step is replaced by compiling the embedded, byte-identical shaders.hlsl with `D3DCompile` (d3dcompiler_47) on first use (`NewRawShaderBytes`, platform/windows/directx_shaders.go, the `shader_resources` module of directx_renderer.rs), with the build.rs entry points and vs_4_1/ps_4_1 targets; flags are `D3DCOMPILE_OPTIMIZATION_LEVEL3` (fxc `/O3`), or upstream's debug flags under `gpuiutil.DebugAssertions`; the include is served by a Go `ID3DInclude` instead of fxc's file-system include; no SDK lookup | `gpui_windows/build.rs:23-201`, `gpui_windows/src/directx_renderer.rs:1722-1880` | Go has no build scripts; DESIGN 8.4 |
| D-WINDXIFACE | `DirectXRenderer` keeps upstream's methods (`DrawWithAppearance`, `Resize`, `HandleDeviceLost`, `MarkDrawable`, `RenderToImage`) and adds the `render.Renderer` set: `Draw(scene)` clears with the background chosen by `UpdateTransparency`, `GPUSpecs` logs `gpu_specs`' error, `DeviceLost` is false and `Recover` a no-op because the platform detects loss and calls `HandleDeviceLost`, `SupportsDualSourceBlending` is true, `SetSubpixelLayout` a no-op (the order comes from DirectWrite) | `gpui_windows/src/directx_renderer.rs:148-830` | the shared `render.Renderer` interface has no background-appearance or devices parameter |
