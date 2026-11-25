# Parity: ext-scap-windows

Scope: zed-scap 0.0.8-zed, the Windows engine (`src/targets/win`, `src/capturer/engine/win`, `src/utils/win`, `src/frame` and the platform-neutral glue they sit in), plus the parts of windows-capture 1.5.0 it calls (`capture.rs`, `graphics_capture_api.rs`, `frame.rs`, `monitor.rs`, `window.rs`, `settings.rs`, `d3d11.rs`). Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/zed-scap-0.0.8-zed/` and `.../windows-capture-1.5.0/`.

Notes on the port:

- The WinRT, interop and Win32 calls windows-rs makes are bound in `internal/win/wgc` over `internal/win/com`, `internal/win/d3d` and `internal/win/win32`; IIDs and vtable slots come from windows 0.62.2 and windows-future 0.3.2. The two `TypedEventHandler<_, IInspectable>` IIDs are the WinRT pinterface hashes (checked by `TestWinRT_PinterfaceIIDs`). Delegates are `com.Object`s that answer their IID and `IAgileObject`.
- `internal/win/screencap.Backend` implements `gpui.ScapBackend`; platform/platform_windows.go registers it (M4-ci-windows).
- Upstream quirks kept: the crop area is always `Some`, so every frame goes through `buffer_crop` + `as_nopadding_buffer` (an invalid crop is a handler error, see D-SCAPW-HARDEN); every output type other than `BGRAFrame` captures Rgba8 and yields `ScapFrameRGBx`; `display_time` is wall-clock nanoseconds; `Monitor::name` compares the monitor's device name with itself, so it returns the first path whose source lookup succeeds; `stop_capture` without `start_capture` panics; the `PostThreadMessageW` retry on `0x800705A4`; the two `GetMessageW` loops treat -1 as true; `is_supported` panics with "Failed to check support" when ApiInformation cannot be activated (every Windows Go supports has it, so unsupported builds return `CapturerBuildError::NotSupported`).
- Live capture tests run on the Windows runner only with `GPUI_TEST_DISPLAY=1`; the owner check is in docs/port/owner-checks/M4-windows-screencap.md.

## zed-scap src/lib.rs

Go target: `internal/win/screencap/targets.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub mod capturer` | mod | 3 |  | - | n/a:D-SCAP-MODS |
| `pub mod frame` | mod | 4 |  | - | n/a:D-SCAP-MODS |
| `pub use targets::get_all_targets` | use | 9 |  | `GetAllTargets` | done |
| `pub use targets::Target` | use | 10 |  | `gpui.ScapTarget` | dev:D-SCAP-ROOTTYPES |
| `pub use targets::{Display, Window}` | use | 11 |  | `gpui.ScapDisplay`, `gpui.ScapWindow` | dev:D-SCAP-ROOTTYPES |
| `pub use utils::has_permission` | use | 12 |  | `HasPermission` | done |
| `pub use utils::is_supported` | use | 13 |  | `IsSupported` | done |
| `pub use utils::request_permission` | use | 14 |  | `RequestPermission` | done |
| `pub mod engine` | mod | 17 | target_os=macos | - | n/a:D-SCAPW-NONWIN |

## zed-scap src/targets/mod.rs

Go target: `internal/win/screencap/targets.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Window` | struct | 13 |  | `gpui.ScapWindow` | dev:D-SCAP-ROOTTYPES |
| `Display` | struct | 28 |  | `gpui.ScapDisplay` | dev:D-SCAP-ROOTTYPES |
| `Target` | enum | 51 |  | `gpui.ScapTarget` | dev:D-SCAP-ROOTTYPES |
| `unsafe impl Send/Sync for Display, Window` | trait-impl | 56 | target_os=windows | - | n/a:D-SCAPW-GENERICS |
| `get_all_targets` | fn | 68 |  | `GetAllTargets` | done |
| `get_scale_factor` | fn | 79 |  | `GetScaleFactor` | done |
| `get_main_display` | fn | 90 |  | `GetMainDisplay` | done |
| `get_target_dimensions` | fn | 101 |  | `GetTargetDimensions` | done |

## zed-scap src/targets/win/mod.rs

Go target: `internal/win/screencap/win_targets.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `get_all_targets` | fn | 13 |  | `winGetAllTargets` | done |
| `get_main_display` | fn | 52 |  | `winGetMainDisplay` | done |
| `monitor_title` | fn | 65 |  | `monitorTitle` | done |
| `get_scale_factor` | fn | 73 |  | `winGetScaleFactor` | done |
| `get_target_dimensions` | fn | 101 |  | `winGetTargetDimensions` | done |

## zed-scap src/capturer/mod.rs

Go target: `internal/win/screencap/capturer.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use engine::get_output_frame_size` | use | 15 |  | `GetOutputFrameSize` | done |
| `Resolution` | enum | 18 |  | `gpui.ScapResolution` | dev:D-SCAP-ROOTTYPES |
| `Resolution::value` | method | 31 |  | `resolutionValue` | done |
| `Point` | struct | 47 |  | `gpui.ScapArea` | dev:D-SCAP-ROOTTYPES |
| `Size` | struct | 53 |  | `gpui.ScapArea` | dev:D-SCAP-ROOTTYPES |
| `Area` | struct | 58 |  | `gpui.ScapArea` | dev:D-SCAP-ROOTTYPES |
| `Options` | struct | 65 |  | `gpui.ScapCapturerOptions` | dev:D-SCAP-ROOTTYPES |
| `Capturer` | struct | 78 |  | `Capturer` | done |
| `CapturerBuildError` | enum | 84 |  | `CapturerBuildError` | done |
| `impl Display for CapturerBuildError` | trait-impl | 89 |  | `(CapturerBuildError).Error` | done |
| `Capturer::new` | assoc-fn | 108 |  | `NewCapturer` | done |
| `Capturer::build` | assoc-fn | 116 |  | `Build` | done |
| `Capturer::start_capture` | method | 134 |  | `(*Capturer).StartCapture` | done |
| `Capturer::stop_capture` | method | 139 |  | `(*Capturer).StopCapture` | done |
| `Capturer::get_next_frame` | method | 144 |  | `(*Capturer).GetNextFrame` | done |
| `Capturer::get_output_frame_size` | method | 155 |  | `(*Capturer).GetOutputFrameSize` | done |
| `Capturer::raw` | method | 159 |  | `(*Capturer).Raw` | done |
| `Capturer::target` | method | 163 |  | `(*Capturer).Target` | done |
| `impl Drop` (implicit, `Capturer`) | trait-impl | 78 |  | `(*Capturer).Close` | dev:D-SCAPW-QUEUE |
| `RawCapturer` | struct | 168 |  | `RawCapturer` | done |
| `mpsc::channel` (frame channel) | type | 125 |  | `frameQueue` | dev:D-SCAPW-QUEUE |

## zed-scap src/capturer/engine/mod.rs

Go target: `internal/win/screencap/engine.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub mod mac` | mod | 9 | target_os=macos | - | n/a:D-SCAPW-NONWIN |
| `mod win` | mod | 12 | target_os=windows | - | n/a:D-SCAP-MODS |
| `mod linux` | mod | 15 | target_os=linux | - | n/a:D-SCAPW-NONWIN |
| `ChannelItem` (macOS) | type | 18 | target_os=macos | - | n/a:D-SCAPW-NONWIN |
| `ChannelItem` | type | 23 |  | `gpui.ScapFrame` | dev:D-SCAP-ROOTTYPES |
| `get_output_frame_size` | fn | 25 |  | `GetOutputFrameSize` | done |
| `Engine` | struct | 43 |  | `Engine` | done |
| `Engine::new` | assoc-fn | 59 |  | `NewEngine` | done |
| `Engine::start` | method | 96 |  | `(*Engine).Start` | done |
| `Engine::stop` | method | 114 |  | `(*Engine).Stop` | done |
| `Engine::get_output_frame_size` | method | 131 |  | `(*Engine).GetOutputFrameSize` | done |
| `Engine::process_channel_item` | method | 135 |  | `(*Engine).ProcessChannelItem` | done |
| `Engine::target` | method | 143 |  | `(*Engine).Target` | done |

## zed-scap src/capturer/engine/win/mod.rs

Go target: `internal/win/screencap/win_engine.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Capturer` | struct | 19 |  | `winCapturer` | done |
| `Settings` | enum | 25 |  | `WCStream.settings` | dev:D-SCAPW-GENERICS |
| `WCStream` | struct | 30 |  | `WCStream` | done |
| `impl GraphicsCaptureApiHandler for Capturer` | trait-impl | 35 |  | `winCapturer` | done |
| `Capturer::new` | assoc-fn | 39 |  | `newWinCapturer` | done |
| `Capturer::on_frame_arrived` | method | 46 |  | `(*winCapturer).OnFrameArrived` | dev:D-SCAPW-HARDEN |
| `Capturer::on_closed` | method | 111 |  | `(*winCapturer).OnClosed` | done |
| `WCStream::start_capture` | method | 118 |  | `(*WCStream).StartCapture` | dev:D-SCAPW-HARDEN |
| `WCStream::stop_capture` | method | 127 |  | `(*WCStream).StopCapture` | dev:D-SCAPW-HARDEN |
| `FlagStruct` | struct | 134 |  | `flagStruct` | done |
| `create_capturer` | fn | 139 |  | `CreateCapturer` | dev:D-SCAPW-HARDEN |
| `get_output_frame_size` | fn | 183 |  | `winGetOutputFrameSize` | done |
| `get_absolute_value` | fn | 207 |  | `getAbsoluteValue` | done |
| `get_crop_area` | fn | 212 |  | `GetCropArea` | dev:D-SCAPW-HARDEN |
| `as u32` / `as i32` float casts | fn | 55 |  | `f64ToU32` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| (none upstream) `get_absolute_value` rounding | test | 207 |  | `TestWinEngine_GetAbsoluteValue` | done |
| (none upstream) saturating float casts | test | 55 |  | `TestWinEngine_SaturatingCasts` | done |
| (none upstream) live capture of the main display | test | 139 |  | `TestCapturer_MainDisplay` | done |

## zed-scap src/frame/mod.rs

Go target: `internal/win/screencap/frame.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `YUVFrame` | struct | 2 |  | `gpui.ScapFrameYUV` | dev:D-SCAP-ROOTTYPES |
| `RGBFrame` | struct | 13 |  | `gpui.ScapFrameRGB` | dev:D-SCAP-ROOTTYPES |
| `RGB8Frame` | struct | 21 |  | - | n/a:D-SCAP-UNUSEDFRAMES |
| `RGBxFrame` | struct | 28 |  | `gpui.ScapFrameRGBx` | dev:D-SCAP-ROOTTYPES |
| `XBGRFrame` | struct | 36 |  | `gpui.ScapFrameXBGR` | dev:D-SCAP-ROOTTYPES |
| `BGRxFrame` | struct | 44 |  | `gpui.ScapFrameBGRx` | dev:D-SCAP-ROOTTYPES |
| `BGRFrame` | struct | 52 |  | `gpui.ScapFrameBGR0` | dev:D-SCAP-ROOTTYPES |
| `BGRAFrame` | struct | 60 |  | `gpui.ScapFrameBGRA` | dev:D-SCAP-ROOTTYPES |
| `FrameType` | enum | 68 |  | `gpui.ScapFrameType` | dev:D-SCAP-ROOTTYPES |
| `Frame` | enum | 77 |  | `gpui.ScapFrame` | dev:D-SCAP-ROOTTYPES |
| `FrameData` | enum | 87 |  | `FrameData` | done |
| `remove_alpha_channel` | fn | 92 |  | `RemoveAlphaChannel` | done |
| `convert_bgra_to_rgb` | fn | 107 |  | `ConvertBGRAToRGB` | done |
| `get_cropped_data` | fn | 122 |  | `GetCroppedData` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `tests::test_remove_alpha_channel` | test | 145 |  | `TestFrame_RemoveAlphaChannel` | done |
| `tests::test_convert_bgra_to_rgb` | test | 154 |  | `TestFrame_ConvertBGRAToRGB` | done |
| `tests::test_get_cropped_data` | test | 169 |  | `TestFrame_GetCroppedData` | done |

## zed-scap src/utils/mod.rs

Go target: `internal/win/screencap/utils.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `has_permission` | fn | 11 |  | `HasPermission` | done |
| `request_permission` | fn | 23 |  | `RequestPermission` | done |
| `is_supported` | fn | 37 |  | `IsSupported` | done |

## zed-scap src/utils/win/mod.rs

Go target: `internal/win/screencap/win_utils.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `is_supported` | fn | 3 |  | `winIsSupported` | done |

## zed-scap src/main.rs

Go target: `internal/win/screencap/targets.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `main` (zed-scap demo binary) | fn | 1 |  | - | n/a:D-SCAP-BIN |

## zed-scap glue: gpui.ScapBackend

Go target: `internal/win/screencap/backend.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| scap free functions + `Capturer::build` as a `gpui.ScapBackend` | struct | 0 |  | `Backend` | done |
| `ScapBackend::get_all_targets` | method | 0 |  | `(Backend).GetAllTargets` | done |
| `ScapBackend::build_capturer` | method | 0 |  | `(Backend).BuildCapturer` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| (none upstream) frame queue close semantics | test | 0 |  | `TestQueue_CloseFailsSendAndRecv` | done |
| (none upstream) Backend satisfies gpui.ScapBackend | test | 0 |  | `TestBackend_Interface` | done |

## windows-capture src/lib.rs

Go target: `internal/win/wgc/doc.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use GraphicsCaptureItem as WindowsCaptureGraphicsCaptureItem` | use | 152 |  | `GraphicsCaptureItem` | done |
| `pub mod capture`, `frame`, `graphics_capture_api`, `monitor`, `settings`, `window` | mod | 155 |  | - | n/a:D-SCAP-MODS |
| `pub mod encoder` | mod | 159 |  | - | n/a:D-SCAPW-ENCODER |

## windows-capture src/d3d11.rs

Go target: `internal/win/wgc/d3d11.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Error` | enum | 17 |  | `D3D11Error`, `ErrFeatureLevelNotSatisfied` | dev:D-SCAPW-ERRORS |
| `SendDirectX` | struct | 25 |  | `SendDirectX` | done |
| `SendDirectX::new` | assoc-fn | 39 |  | `NewSendDirectX` | done |
| `create_d3d_device` | fn | 49 |  | `CreateD3DDevice` | done |
| `create_direct3d_device` | fn | 90 |  | `CreateDirect3DDevice` | done |

## windows-capture src/settings.rs

Go target: `internal/win/wgc/settings.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CaptureItemTypes` | enum | 9 |  | `CaptureItemTypes` | dev:D-SCAPW-GENERICS |
| `TryIntoCaptureItemWithType` | trait | 17 |  | `TryIntoCaptureItemWithType` | done |
| `ColorFormat` | enum | 26 |  | `ColorFormat` | done |
| `impl Default for ColorFormat` | trait-impl | 35 |  | `DefaultColorFormat` | done |
| `CursorCaptureSettings` | enum | 45 |  | `CursorCaptureSettings` | done |
| `DrawBorderSettings` | enum | 56 |  | `DrawBorderSettings` | done |
| `SecondaryWindowSettings` | enum | 67 |  | `SecondaryWindowSettings` | done |
| `MinimumUpdateIntervalSettings` | enum | 78 |  | `MinimumUpdateIntervalSettings` | dev:D-SCAPW-GENERICS |
| `DirtyRegionSettings` | enum | 87 |  | `DirtyRegionSettings` | done |
| `Settings` | struct | 98 |  | `Settings` | done |
| `Settings::new` | assoc-fn | 133 |  | `NewSettings` | dev:D-SCAPW-WCFORK |
| `Settings::item` | method | 162 |  | `(*Settings).Item` | done |
| `Settings::cursor_capture` | method | 173 |  | `(*Settings).CursorCapture` | done |
| `Settings::draw_border` | method | 184 |  | `(*Settings).DrawBorder` | done |
| `Settings::color_format` | method | 195 |  | `(*Settings).ColorFormat` | done |
| `Settings::flags` | method | 206 |  | `(*Settings).Flags` | done |

## windows-capture src/monitor.rs

Go target: `internal/win/wgc/monitor.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Error` | enum | 26 |  | `MonitorError`, `ErrMonitorNotFound` | dev:D-SCAPW-ERRORS |
| `Monitor` | struct | 61 |  | `Monitor` | done |
| `Monitor::primary` | assoc-fn | 74 |  | `MonitorPrimary` | done |
| `Monitor::from_index` | assoc-fn | 96 |  | `MonitorFromIndex` | done |
| `Monitor::index` | method | 116 |  | `(Monitor).Index` | done |
| `Monitor::name` | method | 127 |  | `(Monitor).Name` | done |
| `Monitor::device_name` | method | 241 |  | `(Monitor).DeviceName` | done |
| `Monitor::device_string` | method | 280 |  | `(Monitor).DeviceString` | done |
| `Monitor::refresh_rate` | method | 340 |  | `(Monitor).RefreshRate` | done |
| `Monitor::width` | method | 362 |  | `(Monitor).Width` | done |
| `Monitor::height` | method | 384 |  | `(Monitor).Height` | done |
| `Monitor::enumerate` | assoc-fn | 406 |  | `MonitorEnumerate` | done |
| `Monitor::from_raw_hmonitor` | assoc-fn | 429 |  | `MonitorFromRawHMonitor` | done |
| `Monitor::as_raw_hmonitor` | method | 436 |  | `(Monitor).AsRawHMonitor` | done |
| `Monitor::enum_monitors_callback` | fn | 442 |  | `enumMonitorsCallback` | done |
| `impl TryIntoCaptureItemWithType for Monitor` | trait-impl | 457 |  | `(Monitor).TryIntoCaptureItem` | done |
| `GetDisplayConfigBufferSizes`, `QueryDisplayConfig`, `DisplayConfigGetDeviceInfo`, `EnumDisplayDevicesW` and their structs | fn | 0 |  | `queryDisplayConfig` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| (none upstream) DISPLAYCONFIG and DISPLAY_DEVICEW layouts | test | 0 |  | `TestSys_Layouts` | done |
| (none upstream) monitor enumeration | test | 406 |  | `TestMonitor_Enumerate` | done |
| (none upstream) strict UTF-16 decoding (`String::from_utf16`) | test | 0 |  | `TestSys_UTF16UntilNul` | done |

## windows-capture src/window.rs

Go target: `internal/win/wgc/window.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Error` | enum | 24 |  | `WindowError`, `WindowNotFoundError`, `ErrNoActiveWindow` | dev:D-SCAPW-ERRORS |
| `Window` | struct | 49 |  | `Window` | done |
| `Window::foreground` | assoc-fn | 62 |  | `WindowForeground` | done |
| `Window::from_name` | assoc-fn | 82 |  | `WindowFromName` | done |
| `Window::from_contains_name` | assoc-fn | 103 |  | `WindowFromContainsName` | done |
| `Window::title` | method | 123 |  | `(Window).Title` | done |
| `Window::process_id` | method | 148 |  | `(Window).ProcessID` | done |
| `Window::process_name` | method | 167 |  | `(Window).ProcessName` | done |
| `Window::monitor` | method | 194 |  | `(Window).Monitor` | done |
| `Window::rect` | method | 208 |  | `(Window).Rect` | done |
| `Window::title_bar_height` | method | 224 |  | `(Window).TitleBarHeight` | done |
| `Window::is_valid` | method | 255 |  | `(Window).IsValid` | done |
| `Window::enumerate` | assoc-fn | 291 |  | `WindowEnumerate` | done |
| `Window::from_raw_hwnd` | assoc-fn | 313 |  | `WindowFromRawHWND` | done |
| `Window::as_raw_hwnd` | method | 320 |  | `(Window).AsRawHWND` | done |
| `Window::enum_windows_callback` | fn | 326 |  | `enumWindowsCallback` | done |
| `impl TryIntoCaptureItemWithType for Window` | trait-impl | 338 |  | `(Window).TryIntoCaptureItem` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| (none upstream) window enumeration | test | 291 |  | `TestWindow_Enumerate` | done |

## windows-capture src/frame.rs

Go target: `internal/win/wgc/frame.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Error` | enum | 19 |  | `FrameError`, `ErrFrameInvalidSize` | dev:D-SCAPW-ERRORS |
| `ImageFormat` | enum | 35 |  | - | n/a:D-SCAPW-ENCODER |
| `Frame` | struct | 52 |  | `Frame` | done |
| `Frame::new` | assoc-fn | 87 |  | `NewFrame` | done |
| `Frame::width` | method | 120 |  | `(*Frame).Width` | done |
| `Frame::height` | method | 131 |  | `(*Frame).Height` | done |
| `Frame::timestamp` | method | 142 |  | `(*Frame).Timestamp` | done |
| `Frame::color_format` | method | 153 |  | `(*Frame).ColorFormat` | done |
| `Frame::as_raw_surface` | method | 169 |  | `(*Frame).AsRawSurface` | done |
| `Frame::as_raw_texture` | method | 185 |  | `(*Frame).AsRawTexture` | done |
| `Frame::buffer` | method | 195 |  | `(*Frame).Buffer` | dev:D-SCAPW-UNMAP |
| `Frame::buffer_crop` | method | 264 |  | `(*Frame).BufferCrop` | dev:D-SCAPW-UNMAP |
| `Frame::buffer_without_title_bar` | method | 357 |  | `(*Frame).BufferWithoutTitleBar` | done |
| `Frame::save_as_image` | method | 380 |  | - | n/a:D-SCAPW-ENCODER |
| `FrameBuffer` | struct | 401 |  | `FrameBuffer` | done |
| `FrameBuffer::new` | assoc-fn | 429 |  | `NewFrameBuffer` | done |
| `FrameBuffer::width` | method | 444 |  | `(*FrameBuffer).Width` | done |
| `FrameBuffer::height` | method | 451 |  | `(*FrameBuffer).Height` | done |
| `FrameBuffer::row_pitch` | method | 458 |  | `(*FrameBuffer).RowPitch` | done |
| `FrameBuffer::depth_pitch` | method | 465 |  | `(*FrameBuffer).DepthPitch` | done |
| `FrameBuffer::has_padding` | method | 472 |  | `(*FrameBuffer).HasPadding` | done |
| `FrameBuffer::as_raw_buffer` | method | 479 |  | `(*FrameBuffer).AsRawBuffer` | done |
| `FrameBuffer::as_nopadding_buffer` | method | 489 |  | `(*FrameBuffer).AsNopaddingBuffer` | done |
| `FrameBuffer::save_as_image` | method | 534 |  | - | n/a:D-SCAPW-ENCODER |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| (none upstream) padded rows are packed | test | 489 |  | `TestFrame_AsNopaddingBuffer` | done |

## windows-capture src/graphics_capture_api.rs

Go target: `internal/win/wgc/graphics_capture_api.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Error` | enum | 30 |  | `ErrUnsupported`, `WrappedError` | dev:D-SCAPW-ERRORS |
| `InternalCaptureControl` | struct | 62 |  | `InternalCaptureControl` | done |
| `InternalCaptureControl::new` | assoc-fn | 78 |  | `NewInternalCaptureControl` | done |
| `InternalCaptureControl::stop` | method | 84 |  | `(InternalCaptureControl).Stop` | done |
| `GraphicsCaptureApi` | struct | 90 |  | `GraphicsCaptureApi` | done |
| `GraphicsCaptureApi::new` | assoc-fn | 138 |  | `NewGraphicsCaptureApi` | done |
| `GraphicsCaptureApi::start_capture` | method | 447 |  | `(*GraphicsCaptureApi).StartCapture` | done |
| `GraphicsCaptureApi::stop_capture` | method | 460 |  | `(*GraphicsCaptureApi).StopCapture` | done |
| `GraphicsCaptureApi::halt_handle` | method | 485 |  | `(*GraphicsCaptureApi).HaltHandle` | done |
| `GraphicsCaptureApi::is_supported` | assoc-fn | 495 |  | `GraphicsCaptureApiIsSupported` | done |
| `GraphicsCaptureApi::is_cursor_settings_supported` | assoc-fn | 508 |  | `GraphicsCaptureApiIsCursorSettingsSupported` | done |
| `GraphicsCaptureApi::is_border_settings_supported` | assoc-fn | 521 |  | `GraphicsCaptureApiIsBorderSettingsSupported` | done |
| `GraphicsCaptureApi::is_secondary_windows_supported` | assoc-fn | 534 |  | `GraphicsCaptureApiIsSecondaryWindowsSupported` | done |
| `GraphicsCaptureApi::is_minimum_update_interval_supported` | assoc-fn | 547 |  | `GraphicsCaptureApiIsMinimumUpdateIntervalSupported` | done |
| `GraphicsCaptureApi::is_dirty_region_supported` | assoc-fn | 560 |  | `GraphicsCaptureApiIsDirtyRegionSupported` | done |
| `impl Drop for GraphicsCaptureApi` | trait-impl | 568 |  | `(*GraphicsCaptureApi).Release` | done |
| WinRT bindings: `GraphicsCaptureItem`, `Direct3D11CaptureFramePool`, `Direct3D11CaptureFrame`, `GraphicsCaptureSession`, `ApiInformation`, `IDirect3DDxgiInterfaceAccess`, `IGraphicsCaptureItemInterop` | type | 0 |  | `Direct3D11CaptureFramePool` | done |
| `TypedEventHandler` / `AsyncActionCompletedHandler` delegates | type | 0 |  | `typedEventHandler` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| (none upstream) pinterface IIDs of the delegates | test | 0 |  | `TestWinRT_PinterfaceIIDs` | done |
| (none upstream) support query | test | 495 |  | `TestGraphicsCaptureApi_IsSupported` | done |

## windows-capture src/capture.rs

Go target: `internal/win/wgc/capture.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CaptureControlError` | enum | 29 |  | `ErrFailedToJoinThread`, `WrappedError` | dev:D-SCAPW-ERRORS |
| `CaptureControl` | struct | 43 |  | `CaptureControl` | done |
| `CaptureControl::new` | assoc-fn | 64 |  | `CaptureControl` | done |
| `CaptureControl::is_finished` | method | 79 |  | `(*CaptureControl).IsFinished` | done |
| `CaptureControl::into_thread_handle` | method | 90 |  | - | n/a:D-SCAPW-THREAD |
| `CaptureControl::halt_handle` | method | 101 |  | `(*CaptureControl).HaltHandle` | done |
| `CaptureControl::callback` | method | 112 |  | `(*CaptureControl).Callback` | dev:D-SCAPW-GENERICS |
| `CaptureControl::wait` | method | 122 |  | `(*CaptureControl).Wait` | done |
| `CaptureControl::stop` | method | 143 |  | `(*CaptureControl).Stop` | done |
| `GraphicsCaptureApiError` | enum | 183 |  | `ErrFailedToInitWinRT`, `WrappedError` | dev:D-SCAPW-ERRORS |
| `Context` | struct | 207 |  | `Context` | done |
| `GraphicsCaptureApiHandler` | trait | 217 |  | `GraphicsCaptureApiHandler` | dev:D-SCAPW-GENERICS |
| `GraphicsCaptureApiHandler::start` | method | 234 |  | `Start` | done |
| `GraphicsCaptureApiHandler::start_free_threaded` | method | 368 |  | `StartFreeThreaded` | dev:D-SCAPW-THREAD |
| `GraphicsCaptureApiHandler::new` | method | 544 |  | - | dev:D-SCAPW-GENERICS |
| `GraphicsCaptureApiHandler::on_frame_arrived` | method | 556 |  | `GraphicsCaptureApiHandler.OnFrameArrived` | done |
| `GraphicsCaptureApiHandler::on_closed` | method | 568 |  | `GraphicsCaptureApiHandler.OnClosed` | done |
| `RoInitialize`, `CreateDispatcherQueueController`, `CreateDirect3D11DeviceFromDXGIDevice` | fn | 0 |  | `createDispatcherQueueController` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| (none upstream) live capture of the primary monitor | test | 368 |  | `TestCapture_PrimaryMonitor` | done |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-SCAPW-NONWIN | macOS and Linux items are not part of the Windows back end (ported by M1-linux-screencap and the macOS stages) | `src/lib.rs:17`, `src/capturer/engine/mod.rs` | - |
| D-SCAPW-QUEUE | the frame channel holds at most 8 items and drops the oldest when full (as D-SCAP-QUEUE); it is closed when the capture thread ends (see D-SCAPW-HARDEN); `Capturer.Close` (the drop of `Capturer`, which drops the receiver) closes it, after which the frame handler's send fails, the handler returns the error and the capture thread halts, as upstream's `tx.send` error does | `src/capturer/mod.rs` | `internal/win/screencap/queue.go`, `engine.go` |
| D-SCAPW-WCFORK | zed pins a windows-capture git fork (rev f0d6c1b) whose `Settings::new` takes item, cursor, border, color format and flags; the port follows windows-capture 1.5.0 and passes the three extra settings (secondary windows, minimum update interval, dirty regions) as `Default`, which makes no session calls, so behaviour is the same | `settings.rs:133`, `src/capturer/engine/win/mod.rs:150` | `internal/win/wgc/settings.go`, `internal/win/screencap/win_engine.go` |
| D-SCAPW-THREAD | the capture thread is a native thread from `CreateThread` running a Go callback (only package platform may lock goroutines to threads); join waits on a channel and the thread handle; there is no `JoinHandle` to hand out, and a panic on the thread ends the process instead of surfacing as `FailedToJoinThread` | `capture.rs:368`, `capture.rs:90` | `internal/win/wgc/capture.go` |
| D-SCAPW-ENCODER | `encoder.rs`, `ImageFormat` and the `save_as_image` methods (with their `UnsupportedFormat`, `ImageEncoderError` and `IoError` errors) are not ported; scap never saves images | `encoder.rs`, `frame.rs:35`, `:380`, `:534` | - |
| D-SCAPW-UNMAP | staging textures are unmapped and released when the frame handler returns; upstream releases the texture when `buffer`/`buffer_crop` returns and never unmaps it | `frame.rs:195`, `:264` | `internal/win/wgc/frame.go` |
| D-SCAPW-ERRORS | the thiserror enums become sentinel errors for unit variants and wrapper types (`WrappedError`, `D3D11Error`, `MonitorError`, `WindowError`, `WindowNotFoundError`, `FrameError`) for data variants, with upstream's messages | `capture.rs`, `graphics_capture_api.rs`, `frame.rs`, `monitor.rs`, `window.rs`, `d3d11.rs` | `internal/win/wgc` |
| D-SCAPW-HARDEN | recoverable failures never panic on the native capture thread: `buffer_crop`/`buffer` failures return a handler error; the crop box is clamped to the frame's size each frame and the reported width/height come from the mapped buffer; the non-crop branch packs rows with `AsNopaddingBuffer`; `Frame.BufferCrop` rejects a box past the texture with `ErrFrameInvalidSize`; start, main-display and target errors are returned (`CreateCapturer`, `GetCropArea`) or delivered to the frame queue; when the capture thread ends (stop, item closed, handler error) its error is queued and the queue closed, so `GetNextFrame` returns instead of blocking | `src/capturer/engine/win/mod.rs:46-134`, `frame.rs:264` | `internal/win/screencap/win_engine.go`, `queue.go`, `internal/win/wgc/frame.go`, `capture.go` |
| D-SCAPW-GENERICS | `GraphicsCaptureApiHandler`'s associated `Flags`/`Error` types and `Self::new` become a Go interface plus a `newHandler func(Context[Flags])` argument; `CaptureControl::callback` locks the handler for a callback instead of returning `Arc<Mutex<T>>`; the item type parameter of `Settings` is the `TryIntoCaptureItemWithType` interface, so scap's `Settings::{Window, Display}` enum is one `wgc.Settings`; `CaptureItemTypes` is a struct with one non-nil field; `MinimumUpdateIntervalSettings::Custom(Duration)` is `{IsCustom, Custom}`; `Send`/`Sync` impls have no Go counterpart | `capture.rs:217`, `settings.rs`, `src/capturer/engine/win/mod.rs:25`, `src/targets/mod.rs:56` | `internal/win/wgc`, `internal/win/screencap/win_engine.go` |
