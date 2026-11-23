# Parity: linux-common

Scope: gpui_linux, shared Linux state (linux/dispatcher.rs, linux/platform.rs). Milestone: M1 (text_system.rs: M2-exit). Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

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

Go target: `platform/linux (+internal/common)/platform.go (+common.go)` (package `linux`), tests in `platform_test.go` (and `internal/common/common_test.go`)

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

