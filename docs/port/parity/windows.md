# Parity: windows

Scope: gpui_windows. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_windows/build.rs

Go target: `-` (package `-`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `compile_shaders` | fn | 23 | windows,debug | `CompileShaders` | todo |
| `find_latest_windows_sdk_binary` | fn | 62 | windows,debug | `FindLatestWindowsSdkBinary` | todo |

## gpui_windows/src/dialog.rs

Go target: `platform/windows/dialog.go` (package `windows`), tests in `dialog_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `task_dialog_cancellation_requires_cancel_button_or_internal_request` | test | 252 |  | `TestDialog_TaskDialogCancellationRequiresCancelButtonOrInternalRequest` | todo |
| `closing_owner_cancels_every_outstanding_dialog` | test | 280 |  | `TestDialog_ClosingOwnerCancelsEveryOutstandingDialog` | todo |
| `closed_owner_waits_for_active_dialog_but_queued_requests_stay_cancelled` | test | 296 |  | `TestDialog_ClosedOwnerWaitsForActiveDialogButQueuedRequestsStayCancelled` | todo |
| `cancelling_one_request_does_not_cancel_its_owner_or_other_requests` | test | 311 |  | `TestDialog_CancellingOneRequestDoesNotCancelItsOwnerOrOtherRequests` | todo |
| `dialogs_serialize_per_owner` | test | 322 |  | `TestDialog_DialogsSerializePerOwner` | todo |
| `ownerless_request_can_be_cancelled_without_a_window` | test | 334 |  | `TestDialog_OwnerlessRequestCanBeCancelledWithoutAWindow` | todo |

## gpui_windows/src/direct_write.rs

Go target: `platform/windows/direct_write.go` (package `windows`), tests in `direct_write_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tabs_preserve_native_advances_and_cursor` | test | 1808 |  | `TestDirectWrite_TabsPreserveNativeAdvancesAndCursor` | todo |
| `unicode_line_separators_preserve_shaped_line_cursor` | test | 1934 |  | `TestDirectWrite_UnicodeLineSeparatorsPreserveShapedLineCursor` | todo |
| `unicode_paragraph_separators_preserve_shaped_line_cursor` | test | 1939 |  | `TestDirectWrite_UnicodeParagraphSeparatorsPreserveShapedLineCursor` | todo |
| `test_cluster_map` | test | 2041 |  | `TestDirectWrite_ClusterMap` | todo |

## gpui_windows/src/directx_atlas.rs

Go target: `platform/windows/directx_atlas.go` (package `windows`), tests in `directx_atlas_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_remove_deallocates_tile_space_for_reuse` | test | 393 |  | `TestDirectxAtlas_RemoveDeallocatesTileSpaceForReuse` | todo |

## gpui_windows/src/directx_renderer.rs

Go target: `platform/windows/directx_renderer.go` (package `windows`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AGSGPUInfo` | struct | 1991 |  | `AGSGPUInfo` | todo |

## gpui_windows/src/keyboard.rs

Go target: `platform/windows/keyboard.go` (package `windows`), tests in `keyboard_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_keyboard_mapper` | test | 323 |  | `TestKeyboard_KeyboardMapper` | todo |

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
| `successful_compositor_wait_reports_native_callback` | test | 92 |  | `TestVsync_SuccessfulCompositorWaitReportsNativeCallback` | todo |
| `failed_compositor_wait_reports_local_schedule_even_after_threshold` | test | 108 |  | `TestVsync_FailedCompositorWaitReportsLocalScheduleEvenAfterThreshold` | todo |

## gpui_windows/src/window.rs

Go target: `platform/windows/window.go` (package `windows`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WindowsWindowState` | struct | 45 |  | `WindowsWindowState` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_double_click_interval` | test | 1695 |  | `TestWindow_DoubleClickInterval` | todo |
| `test_double_click_spatial_tolerance` | test | 1723 |  | `TestWindow_DoubleClickSpatialTolerance` | todo |

