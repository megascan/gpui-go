# Parity: linux-common

Scope: gpui_linux, shared Linux state (linux/dispatcher.rs, linux/platform.rs). Milestone: M1 (text_system.rs: M2-exit). Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_linux/src/linux/dispatcher.rs

Go target: `platform/linux/internal/common/dispatcher.go` (package `common`), tests in `dispatcher_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PriorityQueueCalloopSender` | struct | 167 |  | `PriorityQueueCalloopSender` | done |
| `PriorityQueueCalloopReceiver` | struct | 192 |  | `PriorityQueueCalloopReceiver` | done |
| `PriorityQueueCalloopReceiver::new` | assoc-fn | 199 |  | `NewPriorityQueueCalloopReceiver` | done |
| `ChannelError` | struct | 218 |  | `ChannelError` | done |
| `TimerAfter` | struct | 15 |  | `timerAfter` | done |
| `LinuxDispatcher` | struct | 20 |  | `LinuxDispatcher` | done |
| `LinuxDispatcher::new` | assoc-fn | 31 |  | `NewLinuxDispatcher` | done |
| `impl PlatformDispatcher for LinuxDispatcher` | impl | 108 |  | `(*LinuxDispatcher).Dispatch` | done |
| `LinuxDispatcher::spawn_realtime` | method | 145 |  | `(*LinuxDispatcher).SpawnRealtime` | dev:D-LINUX-RTPRIO |
| `PriorityQueueCalloopSender::send` | method | 177 |  | `(*PriorityQueueCalloopSender).Send` | done |
| `impl Drop for PriorityQueueCalloopSender` | impl | 186 |  | `(*PriorityQueueCalloopSender).Close` | done |
| `impl EventSource for PriorityQueueCalloopReceiver` | impl | 232 |  | `(*PriorityQueueCalloopReceiver).ProcessEvents` | dev:D-LINUX-CALLOOP |
| calloop `EventLoop`, `LoopHandle`, `LoopSignal`, ping, channel, generic and timer sources | dep | — |  | `event_loop.go`: `EventLoop`, `LoopHandle`, `LoopSignal`, `MakePing`, `NewChannel`, `NewGeneric`, `InsertSource`, `(*LoopHandle).InsertTimer` | dev:D-LINUX-CALLOOP |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `calloop_works` | test | 315 |  | `TestDispatcher_CalloopWorks` | done |

## gpui_linux/src/linux/platform.rs

Go target: `platform/linux/internal/common/common.go` (package `common`), tests in `common_test.go`; LinuxPlatform and the display-mode tests are in the second platform.rs section

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SCROLL_LINES` | const | 49 | feat:wayland,feat:x11 | `ScrollLines` | done |
| `DOUBLE_CLICK_INTERVAL` | const | 54 | feat:wayland,feat:x11 | `DoubleClickInterval` | done |
| `DOUBLE_CLICK_DISTANCE` | const | 56 | feat:wayland,feat:x11 | `DoubleClickDistance` | done |
| `KEYRING_LABEL` | const | 57 |  | `KeyringLabel` | done |
| `FILE_PICKER_PORTAL_MISSING` | const | 60 | feat:wayland,feat:x11 | `FilePickerPortalMissing` | done |
| `PlatformHandlers` | struct | 64 |  | `PlatformHandlers` | done |
| `SystemPowerEvent` | enum | 85 |  | `SystemPowerEvent` | done |
| `LinuxCommon` | struct | 91 |  | `LinuxCommon` | dev:D-LINUX-BUS |
| `LinuxCommon::new` | assoc-fn | 121 |  | `NewLinuxCommon` | done |
| `LinuxCommon::start_power_listener` | method | 168 |  | `(*LinuxCommon).StartPowerListener` | done |
| `LinuxCommon::register_sources` | assoc-fn | 189 |  | `RegisterSources` | done |
| `LinuxCommon::handle_system_power_event` | assoc-fn | 232 |  | `handleSystemPowerEvent` | done |
| `listen_for_system_power_events` | fn | 264 | target_os=linux | `listenForSystemPowerEvents` | done |
| `DisplayConnection` dispatch targets (WaylandConnection / X11Connection methods) | trait-like | display_connection.rs:98 |  | `LinuxClient` | dev:D-LINUX-CLIENT |
| `run_open_commands` | fn | 1194 | feat:wayland,feat:x11 | `runOpenCommands` (+ `openCommands`, open-5.4.5) | dev:D-LINUX-OPEN |
| `open_uri_internal` | fn | 1218 |  | `OpenURIInternal` | done |
| `reveal_path_internal` | fn | 1253 | feat:wayland,feat:x11 | `RevealPathInternal` | done |
| `is_within_click_distance` | fn | 1287 | feat:wayland,feat:x11 | `IsWithinClickDistance` | done |
| `new_xkb_context` | fn | 1297 | feat:wayland,feat:x11 | `NewXkbContext` | done |
| `validate_xkb_context` | fn | 1302 | feat:wayland,feat:x11 | `validateXkbContext` | done |
| `get_xkb_compose_state` | fn | 1311 | feat:wayland,feat:x11 | `GetXkbComposeState` | done |
| `PIPE_READ_TIMEOUT` | const | 1333 | feat:wayland,feat:x11 | `PipeReadTimeout` | done |
| `read_fd_with_timeout` | fn | 1336 | feat:wayland,feat:x11 | `ReadFdWithTimeout` | done |
| `DEFAULT_CURSOR_ICON_NAME` | const | 1373 | feat:wayland,feat:x11 | `DefaultCursorIconName` | done |
| `cursor_style_to_icon_names` | fn | 1376 | feat:wayland,feat:x11 | `CursorStyleToIconNames` | done |
| `log_cursor_icon_warning` | fn | 1405 | feat:wayland,feat:x11 | `LogCursorIconWarning` | done |
| `guess_ascii` | fn | 1419 | feat:wayland,feat:x11 | `guessASCII` | done |
| `keystroke_from_xkb` | fn | 1473 | feat:wayland,feat:x11 | `KeystrokeFromXkb` | done |
| `keystroke_underlying_dead_key` | fn | 1602 | feat:wayland,feat:x11 | `KeystrokeUnderlyingDeadKey` | done |
| `modifiers_from_xkb` | fn | 1657 | feat:wayland,feat:x11 | `ModifiersFromXkb` | done |
| `capslock_from_xkb` | fn | 1672 | feat:wayland,feat:x11 | `CapslockFromXkb` | done |
| `compositor_gpu_hint_from_dev_t` | fn | 1681 | feat:wayland,feat:x11 | `CompositorGPUHintFromDevT` | dev:D-LINUX-GPUHINT |
| `inhibit_idle_sleep` | fn | 1716 | feat:wayland,feat:x11 | `InhibitIdleSleep` | done |
| `release_on_drop` | fn | 1737 | feat:wayland,feat:x11 | `releaseOnDrop` | done |
| `await_idle_sleep_prevention` | fn | 1750 | feat:wayland,feat:x11 | `AwaitIdleSleepPrevention` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `power_mode_callbacks_can_reenter_and_replace_themselves` | test | 1773 |  | `TestPlatform_PowerModeCallbacksCanReenterAndReplaceThemselves` | done |
| `rejects_null_xkb_context` | test | 1811 | feat:wayland,feat:x11 | `TestPlatform_RejectsNullXkbContext` | done |
| `test_is_within_click_distance` | test | 1827 |  | `TestPlatform_IsWithinClickDistance` | done |
| `guard_passes_through_and_releases_on_drop` | gpui::test | 1859 | feat:wayland,feat:x11 | `TestPlatformIdleSleepPrevention_GuardPassesThroughAndReleasesOnDrop` | done |
| `acquisition_error_passes_through` | gpui::test | 1873 | feat:wayland,feat:x11 | `TestPlatformIdleSleepPrevention_AcquisitionErrorPassesThrough` | done |
| `abandoned_acquisition_is_an_error` | gpui::test | 1887 | feat:wayland,feat:x11 | `TestPlatformIdleSleepPrevention_AbandonedAcquisitionIsAnError` | done |
| `late_guard_after_timeout_is_released` | gpui::test | 1901 | feat:wayland,feat:x11 | `TestPlatformIdleSleepPrevention_LateGuardAfterTimeoutIsReleased` | done |
| `reads_data_written_before_close` | test | 1943 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_ReadsDataWrittenBeforeClose` | done |
| `returns_empty_when_writer_closes_without_writing` | test | 1953 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_ReturnsEmptyWhenWriterClosesWithoutWriting` | done |
| `times_out_when_writer_never_writes` | test | 1962 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_TimesOutWhenWriterNeverWrites` | done |
| `times_out_when_writer_stalls_after_partial_write` | test | 1980 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_TimesOutWhenWriterStallsAfterPartialWrite` | done |
| `slow_writer_resets_deadline_between_chunks` | test | 1993 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_SlowWriterResetsDeadlineBetweenChunks` | done |
| `reads_payload_larger_than_pipe_capacity` | test | 2016 | feat:wayland,feat:x11 | `TestPlatformReadFdWithTimeout_ReadsPayloadLargerThanPipeCapacity` | done |

## gpui_linux/src/linux/platform.rs

Go target: `platform/linux/platform.go` (package `linux`), tests in `platform_test.go` (LinuxPlatform, M1-linux-platform)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LinuxPlatform` | struct | 295 |  | `linuxPlatform` | done |
| `StartupEnvironment` | struct | 317 |  | `startupEnvironment` | done |
| `LinuxPlatform::new` | assoc-fn | 328 |  | `newLinuxPlatform` | done |
| `LinuxPlatform::connect_initially` | method | 361 |  | `(*linuxPlatform).connectInitially` | done |
| `LinuxPlatform::with_common` | method | 395 |  | `linuxPlatform` (field access) | dev:D-LINUX-NOCELL |
| `LinuxPlatform::connect` | method | 410 |  | `(*linuxPlatform).connect` | done |
| `LinuxPlatform::replace_connection` | method | 446 |  | `(*linuxPlatform).replaceConnection` | done |
| `LinuxPlatform::apply_pending_mode` | method | 480 |  | `(*linuxPlatform).applyPendingMode` | done |
| `LinuxPlatform::request_mode` | method | 510 |  | `(*linuxPlatform).requestMode` | done |
| `impl Drop for LinuxPlatform` | impl | 560 |  | `(*linuxPlatform).Close` | done |
| `impl Platform for LinuxPlatform` | impl | 568 |  | `(*linuxPlatform).Run` | done |
| `{:?}` of `WindowingModes` in panic and error messages | fmt | 389 |  | `windowingModesDebug` | done |
| `prompt_for_paths` / `prompt_for_new_path` (ashpd FileChooser) | method | 822 |  | `fileChooserRequest` | dev:D-LINUX-FILECHOOSER |
| `write_credentials` / `read_credentials` / `delete_credentials` (oo7) | method | 1086 |  | `(*linuxPlatform).WriteCredentials` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `failed_wayland_attach_preserves_headless_app_and_allows_retry` | test | 2049 | feat:wayland | `TestPlatformDisplayModeTests_FailedWaylandAttachPreservesHeadlessAppAndAllowsRetry` | done |
| `headless_windows_block_switching_to_a_display_server` | test | 2102 | feat:wayland | `TestPlatformDisplayModeTests_HeadlessWindowsBlockSwitchingToADisplayServer` | done |
| `display_modes_limit_switching` | test | 2134 |  | `TestPlatformDisplayModeTests_DisplayModesLimitSwitching` | done |

## gpui_linux/src/linux/headless.rs

Go target: `platform/linux/internal/common/headless.go` (package `common`), tests in `headless_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `HeadlessConnection` | struct | 13 |  | `HeadlessConnection` | done |
| `HeadlessConnection::new` | assoc-fn | 19 |  | `NewHeadlessConnection` | done |
| `HeadlessConnection::has_windows` | method | 25 |  | `(*HeadlessConnection).HasWindows` | done |
| `HeadlessConnection::displays` | method | 29 |  | `(*HeadlessConnection).Displays` | done |
| `HeadlessConnection::primary_display` | method | 33 |  | `(*HeadlessConnection).PrimaryDisplay` | done |
| `HeadlessConnection::open_window` | method | 37 |  | `(*HeadlessConnection).OpenWindow` | done |

## gpui_linux/src/linux/headless/window.rs

Go target: `platform/linux/internal/common/headless_window.go` (package `common`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `HeadlessWindowState` | struct | 20 |  | `headlessWindowState` | done |
| `HeadlessWindow` | struct | 29 |  | `HeadlessWindow` | done |
| `HeadlessWindow::new` | assoc-fn | 48 |  | `newHeadlessWindow` | done |
| `impl HasWindowHandle / HasDisplayHandle for HeadlessWindow` | impl | 31 |  | `(*HeadlessWindow).RawWindowHandle` | done |
| `impl PlatformWindow for HeadlessWindow` | impl | 59 |  | `(*HeadlessWindow).Close` | done |

## gpui_linux/src/linux/keyboard.rs

Go target: `platform/linux/internal/common/keyboard.go` (package `common`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LinuxKeyboardLayout` | struct | 4 |  | `LinuxKeyboardLayout` | done |
| `impl PlatformKeyboardLayout for LinuxKeyboardLayout` | impl | 8 |  | `(*LinuxKeyboardLayout).ID` | done |
| `LinuxKeyboardLayout::new` | assoc-fn | 19 |  | `NewLinuxKeyboardLayout` | done |

## gpui_linux/src/linux/text_system.rs

Go target: `platform/linux/internal/common/text_system.go` (package `common`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use gpui_wgpu::CosmicTextSystem` | use | 1 |  | `newTextSystem` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-LINUX-CALLOOP | calloop is replaced by `event_loop.go`: the goroutine that calls `Dispatch` / `Run` polls the registered fds plus an eventfd wakeup with `poll(2)`, fires due timers, then runs idle callbacks, with calloop's `PostAction`, `LoopSignal` (stop flag + wakeup ping), ping (eventfd), channel, generic and timer sources and the before-sleep / before-handle-events hooks; callbacks are bound when a source is inserted (`InsertSource`). The dispatcher's timer thread runs its own `EventLoop` as upstream's does | `gpui_linux/src/linux/dispatcher.rs:1-310`, calloop-0.14.5 `loop_logic.rs` | calloop has no Go port; poll on the loop goroutine keeps Wayland's prepare_read / read_events on the polling thread (DESIGN 8.1) |
| D-LINUX-RTPRIO | `SpawnRealtime` runs f on a plain goroutine without SCHED_FIFO priority 65 | `gpui_linux/src/linux/dispatcher.rs:145-164` | setting a per-thread scheduling policy needs `runtime.LockOSThread`, which `paritycheck -lint` allows only in package platform; handoff in docs/port/handoff/M1-linux-common.md |
| D-LINUX-BUS | LinuxCommon opens the session and system buses lazily through the replaceable `ConnectSessionBus` / `ConnectSystemBus` fields; `SystemNotifications()` creates the notification state on first use; `Close` ends the dispatcher's goroutines and closes the buses | `gpui_linux/src/linux/platform.rs:91-162` | zbus connects per call site; D-Bus code takes its connection as a parameter (STAGES M1-linux-common) and tests must not reach the user's buses |
| D-LINUX-CLIENT | The Wayland and X11 connections satisfy the `LinuxClient` interface, which LinuxPlatform's DisplayConnection holds, instead of an enum matched by the `dispatch!` macro | `gpui_linux/src/linux/display_connection.rs:68-221` | `platform/linux/wayland` and `x11` are separate packages that import `internal/common` (DESIGN section 4) |
| D-LINUX-OPEN | open::commands is ported without its WSL PowerShell launcher; launchers, order and option-safe paths match open-5.4.5 | `open-5.4.5/src/unix.rs:58-149` | WSL needs a Windows host to test; tracked in docs/port/debt/M1-linux-common.md |
| D-LINUX-GPUHINT | `CompositorGPUHintFromDevT` returns `(vendorID, deviceID, ok)` instead of `gpui_wgpu::CompositorGpuHint`; callers build `wgpu.CompositorGPUHint` | `gpui_linux/src/linux/platform.rs:1681-1714` | importing render/wgpu (cgo) into the shared package would make every common test link wgpu-native |
| D-LINUX-BOUNDS | `ReadFdWithTimeout` fails once a pipe delivers more than `MaxPipeReadSize` (256 MiB); a portal request waits at most `PortalResponseTimeout` (5 min) for its Response, then closes the Request and fails | `gpui_linux/src/linux/platform.rs` read_fd / ashpd request | review: another client could stream unbounded clipboard / drag data, and a silent portal left a worker and signal subscription forever |
| D-LINUX-NOCELL | LinuxPlatform's `RefCell` fields and `with_common` become plain fields used on the main goroutine; replacing a connection swaps the field before closing the old one, as upstream does outside the borrow | `gpui_linux/src/linux/platform.rs:295-439` | Go has no borrow checker; the platform is only touched from the main goroutine |
| D-LINUX-FILECHOOSER | The FileChooser portal is called with godbus (`fileChooserRequest`) instead of ashpd: a missing portal is detected from the `version` property or the D-Bus error name and reported as `FILE_PICKER_PORTAL_MISSING`; the blocking call runs on the background executor while the foreground task awaits it | `gpui_linux/src/linux/platform.rs:822-943` | ashpd has no Go port; D-Bus calls block, and foreground tasks must not |
