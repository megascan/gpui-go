# Parity: ext-scap-linux

Scope: zed-scap 0.0.8-zed, the Linux engines (`src/targets/linux`, `src/capturer/engine/linux/{wayland,x11}`, `src/frame`, and the platform-neutral glue they sit in), plus the pipewire-rs / libspa calls the Wayland engine makes. Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/zed-scap-0.0.8-zed/`.

Notes on the port:

- Both engines are built, as with scap's default features (`wayland` + `x11`); the platform-detection error is the both-features string. Wayland is picked when `WAYLAND_DISPLAY` is set, as upstream.
- scap's target, option and frame types are the root's `gpui.Scap*` types (M0-platform-types), so the back end and the root glue share them.
- The Wayland engine connects to the default PipeWire daemon (`context.connect(None)`) as upstream does; `OpenPipeWireRemote` and `pw_context_connect_fd` are bound but unused, as upstream.
- `mpsc::channel` becomes a mutex/cond queue bounded at 8 items that drops the oldest when full (D-SCAP-QUEUE) and closes when the producing goroutine exits, so `GetNextFrame` fails with "receiving on a closed channel" like `RecvError`.
- Wayland frames are repacked from `chunk.offset`/`chunk.stride` into tightly packed rows (D-SCAP-PACKED).
- Upstream quirks kept: X11 cursor blending writes R into byte 0 of BGRx; a STRING-typed `_NET_WM_NAME` hits xcb's format assertion panic.
- The interactive portal dialog is an owner check (docs/port/owner-checks/M1-linux-screencap.md).

## src/lib.rs

Go target: `platform/linux/internal/screencap/targets.go`

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
| `pub mod engine` | mod | 17 | target_os=macos | - | n/a:D-SCAP-NONLINUX |

## src/targets/mod.rs

Go target: `platform/linux/internal/screencap/targets.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Window` | struct | 13 |  | `gpui.ScapWindow` | dev:D-SCAP-ROOTTYPES |
| `Display` | struct | 28 |  | `gpui.ScapDisplay` | dev:D-SCAP-ROOTTYPES |
| `Target` | enum | 51 |  | `gpui.ScapTarget` | dev:D-SCAP-ROOTTYPES |
| `get_all_targets` | fn | 68 |  | `GetAllTargets` | done |
| `get_scale_factor` | fn | 79 |  | `GetScaleFactor` | done |
| `get_main_display` | fn | 90 |  | `GetMainDisplay` | done |
| `get_target_dimensions` | fn | 101 |  | `GetTargetDimensions` | done |

## src/targets/linux/mod.rs

Go target: `platform/linux/internal/screencap/linux_targets.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `get_atom` | fn | 23 | x11 | `getAtom` | done |
| `get_property` | fn | 32 | x11 | `getProperty` | done |
| `GetPropertyReply::value` (xcb format assertion) | method | 150 | x11 | `propertyValue` | done |
| `decode_compound_text` | fn | 51 | x11 | `decodeCompoundText` | dev:D-SCAP-CTEXT |
| `get_x11_targets` | fn | 98 | x11 | `getX11Targets` | done |
| `get_all_targets` | fn | 190 |  | `linuxGetAllTargets` | done |
| `get_default_x_display` | fn | 213 | x11 | `getDefaultXDisplay` | done |
| `get_main_display` | fn | 243 |  | `linuxGetMainDisplay` | done |
| `get_target_dimensions` | fn | 270 |  | `linuxGetTargetDimensions` | done |

## src/capturer/mod.rs

Go target: `platform/linux/internal/screencap/capturer.go`

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
| `impl Drop` (implicit, `Capturer`) | trait-impl | 78 |  | `(*Capturer).Close` | done |
| `RawCapturer` | struct | 168 |  | `RawCapturer` | done |

## src/capturer/engine/mod.rs

Go target: `platform/linux/internal/screencap/engine.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub mod mac` | mod | 9 | target_os=macos | - | n/a:D-SCAP-NONLINUX |
| `ChannelItem` (macOS) | type | 18 | target_os=macos | - | n/a:D-SCAP-NONLINUX |
| `ChannelItem` | type | 23 |  | `gpui.ScapFrame` | dev:D-SCAP-ROOTTYPES |
| `get_output_frame_size` | fn | 25 |  | `GetOutputFrameSize` | done |
| `Engine` | struct | 43 |  | `Engine` | done |
| `Engine::new` | assoc-fn | 59 |  | `NewEngine` | done |
| `Engine::start` | method | 96 |  | `(*Engine).Start` | done |
| `Engine::stop` | method | 114 |  | `(*Engine).Stop` | done |
| `Engine::get_output_frame_size` | method | 131 |  | `(*Engine).GetOutputFrameSize` | done |
| `Engine::process_channel_item` | method | 135 |  | `(*Engine).ProcessChannelItem` | done |
| `Engine::target` | method | 143 |  | `(*Engine).Target` | done |

## src/capturer/engine/linux/mod.rs

Go target: `platform/linux/internal/screencap/linux_engine.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LinuxCapturerImpl` | trait | 24 |  | `LinuxCapturerImpl` | done |
| `LinuxCapturerImpl::target` (default) | method | 27 |  | `LinuxCapturerImpl` | done |
| `LinuxCapturer` | struct | 32 |  | `LinuxCapturer` | done |
| `Type` | type | 36 |  | `frameQueue` | done |
| `LinuxCapturer::new` | assoc-fn | 39 |  | `NewLinuxCapturer` | done |
| `create_capturer` | fn | 67 |  | `CreateCapturer` | done |

## src/capturer/engine/linux/error.rs

Go target: `platform/linux/internal/screencap/linux_error.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `LinCapError` | struct | 11 |  | `LinCapError` | done |
| `impl Error for LinCapError` | trait-impl | 15 |  | `(*LinCapError).Error` | done |
| `LinCapError::new` | assoc-fn | 18 |  | `NewLinCapError` | done |
| `impl Display for LinCapError` | trait-impl | 23 |  | `(*LinCapError).Error` | done |
| `impl From<pipewire::Error>` | trait-impl | 30 | wayland | `linCapErrorFrom` | done |
| `impl From<SendError<bool>>` | trait-impl | 36 |  | `linCapErrorFrom` | done |
| `impl From<GenError>` | trait-impl | 43 | wayland | `linCapErrorFrom` | done |
| `impl From<dbus::Error>` | trait-impl | 50 | wayland | `linCapErrorFrom` | done |
| `impl From<PoisonError<T>>` | trait-impl | 56 |  | `linCapErrorFrom` | done |

## src/capturer/engine/linux/x11/mod.rs

Go target: `platform/linux/internal/screencap/x11.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `X11Capturer` | struct | 17 |  | `X11Capturer` | done |
| `draw_cursor` | fn | 23 |  | `drawCursor` | done |
| `grab` | fn | 121 |  | `grab` | dev:D-SCAP-XSHM |
| `query_xfixes_version` | fn | 177 |  | `queryXfixesVersion` | done |
| `X11Capturer::new` | assoc-fn | 187 |  | `NewX11Capturer` | done |
| `X11Capturer::start_capture` | method | 245 |  | `(*X11Capturer).StartCapture` | done |
| `X11Capturer::stop_capture` | method | 249 |  | `(*X11Capturer).StopCapture` | done |
| `X11Capturer::target` | method | 258 |  | `(*X11Capturer).Target` | done |

## src/capturer/engine/linux/wayland/mod.rs

Go target: `platform/linux/internal/screencap/wayland.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CAPTURER_STATE` | static | 46 |  | `waylandState` | dev:D-SCAP-STATE |
| `STREAM_STATE_CHANGED_TO_ERROR` | static | 47 |  | `waylandState` | dev:D-SCAP-STATE |
| `ListenerUserData` | struct | 50 |  | `listenerUserData` | done |
| `param_changed_callback` | fn | 55 |  | `paramChangedCallback` | done |
| `state_changed_callback` | fn | 83 |  | `stateChangedCallback` | done |
| `get_timestamp` | fn | 98 |  | `getTimestamp` | done |
| `process_callback` | fn | 117 |  | `processCallback` | dev:D-SCAP-NULLDATA |
| `process_callback_impl` | fn | 135 |  | `processCallbackImpl` | dev:D-SCAP-NULLDATA |
| `start_pipewire_capturer` | fn | 193 |  | `startPipewireCapturer` | dev:D-SCAP-PWLIFETIME |
| `pipewire_capturer` | fn | 323 |  | `pipewireCapturer` | done |
| `WaylandCapturer` | struct | 355 |  | `WaylandCapturer` | done |
| `WaylandCapturer::new` | assoc-fn | 364 |  | `NewWaylandCapturer` | done |
| `WaylandCapturer::start_capture` | method | 400 |  | `(*WaylandCapturer).StartCapture` | done |
| `WaylandCapturer::stop_capture` | method | 404 |  | `(*WaylandCapturer).StopCapture` | done |
| `impl Drop` (implicit, `_connection`) | trait-impl | 359 |  | `(*WaylandCapturer).close` | done |

## src/capturer/engine/linux/wayland/portal.rs

Go target: `platform/linux/internal/screencap/portal.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `OrgFreedesktopPortalScreenCast` | trait | 20 |  | `screenCastProxy` | done |
| `OrgFreedesktopPortalScreenCast::create_session` | method | 46 |  | `(screenCastProxy).CreateSession` | done |
| `OrgFreedesktopPortalScreenCast::select_sources` | method | 55 |  | `(screenCastProxy).SelectSources` | done |
| `OrgFreedesktopPortalScreenCast::start` | method | 68 |  | `(screenCastProxy).Start` | done |
| `OrgFreedesktopPortalScreenCast::open_pipe_wire_remote` | method | 82 |  | `(screenCastProxy).OpenPipeWireRemote` | done |
| `OrgFreedesktopPortalScreenCast::available_source_types` | method | 95 |  | `(screenCastProxy).AvailableSourceTypes` | done |
| `OrgFreedesktopPortalScreenCast::available_cursor_modes` | method | 103 |  | `(screenCastProxy).AvailableCursorModes` | done |
| `OrgFreedesktopPortalScreenCast::version` | method | 111 |  | `(screenCastProxy).Version` | done |
| `OrgFreedesktopPortalRequest::close` | method | 158 |  | `requestClose` | done |
| `OrgFreedesktopPortalRequestResponse` | struct | 129 |  | `RequestResponse` | done |
| `impl ReadAll for OrgFreedesktopPortalRequestResponse` | trait-impl | 141 |  | `readRequestResponse` | done |
| `impl AppendAll for OrgFreedesktopPortalRequestResponse` | trait-impl | 134 |  | - | n/a:D-SCAP-DBUSCODEGEN |
| `impl SignalArgs for OrgFreedesktopPortalRequestResponse` | trait-impl | 150 |  | `portalRequest` | done |
| `Response` | type | 164 |  | `RequestResponse` | done |
| `StreamVardict` | struct | 168 |  | `StreamVardict` | done |
| `Stream` | struct | 178 |  | `Stream` | done |
| `Stream::pw_node_id` | method | 181 |  | `(Stream).PwNodeID` | done |
| `Stream::from_dbus` | assoc-fn | 185 |  | `StreamFromDBus` | done |
| `match_response!` | macro | 204 |  | `matchResponse` | dev:D-SCAP-PORTAL |
| `ScreenCastPortal` | struct | 223 |  | `ScreenCastPortal` | done |
| `ScreenCastPortal::new` | assoc-fn | 230 |  | `NewScreenCastPortal` | done |
| `ScreenCastPortal::create_session_args` | method | 246 |  | `(*ScreenCastPortal).createSessionArgs` | done |
| `ScreenCastPortal::select_sources_args` | method | 259 |  | `(*ScreenCastPortal).selectSourcesArgs` | done |
| `ScreenCastPortal::handle_req_response` | assoc-fn | 277 |  | `callWithResponse` | dev:D-SCAP-PORTAL |
| `ScreenCastPortal::create_session` | method | 313 |  | `(*ScreenCastPortal).createSession` | done |
| `ScreenCastPortal::select_sources` | method | 352 |  | `(*ScreenCastPortal).selectSources` | done |
| `ScreenCastPortal::start` | method | 375 |  | `(*ScreenCastPortal).start` | done |
| `ScreenCastPortal::create_stream` | method | 406 |  | `(*ScreenCastPortal).CreateStream` | done |
| `ScreenCastPortal::show_cursor` | method | 412 |  | `(*ScreenCastPortal).ShowCursor` | done |

## src/frame/mod.rs

Go target: `platform/linux/internal/screencap/frame.go`

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

## src/utils/mod.rs

Go target: `platform/linux/internal/screencap/utils.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `has_permission` | fn | 11 |  | `HasPermission` | done |
| `request_permission` | fn | 23 |  | `RequestPermission` | done |
| `is_supported` | fn | 37 |  | `IsSupported` | done |

## src/utils/linux/mod.rs

Go target: `platform/linux/internal/screencap/linux_utils.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `is_supported` | fn | 2 |  | `linuxIsSupported` | done |
| `has_permission` | fn | 8 |  | `linuxHasPermission` | done |

## src/main.rs

Go target: `platform/linux/internal/screencap/targets.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `main` (zed-scap demo binary) | fn | 1 |  | - | n/a:D-SCAP-BIN |

## pipewire 0.8 / libspa calls used by the Wayland engine

Go target: `internal/linux/pw/pw.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pw::init` | fn | 0 | wayland | `Init` | done |
| `MainLoop::new` | assoc-fn | 0 | wayland | `NewMainLoop` | done |
| `MainLoop::loop_` | method | 0 | wayland | `(*MainLoop).Loop` | done |
| `impl Drop for MainLoop` | trait-impl | 0 | wayland | `(*MainLoop).Destroy` | done |
| `LoopRef::iterate` | method | 0 | wayland | `(Loop).Iterate` | done |
| `Context::new` | assoc-fn | 0 | wayland | `NewContext` | done |
| `impl Drop for Context` | trait-impl | 0 | wayland | `(*Context).Destroy` | done |
| `Context::connect` | method | 0 | wayland | `(*Context).Connect` | done |
| `Context::connect_fd` | method | 0 | wayland | `(*Context).ConnectFD` | done |
| `impl Drop for Core` | trait-impl | 0 | wayland | `(*Core).Disconnect` | done |
| `properties!` | macro | 0 | wayland | `NewProperties` | done |
| `keys::MEDIA_TYPE`, `MEDIA_CATEGORY`, `MEDIA_ROLE` | const | 0 | wayland | `KeyMediaType` | done |
| `Stream::new` | assoc-fn | 0 | wayland | `NewStream` | done |
| `impl Drop for Stream` | trait-impl | 0 | wayland | `(*Stream).Destroy` | done |
| `Stream::add_local_listener_with_user_data` ... `register` | method | 0 | wayland | `(*Stream).AddListener` | done |
| `impl Drop for StreamListener` | trait-impl | 0 | wayland | `(*StreamListener).Remove` | done |
| `Stream::connect` | method | 0 | wayland | `(*Stream).Connect` | done |
| `StreamRef::dequeue_raw_buffer` | method | 0 | wayland | `(*Stream).DequeueRawBuffer` | done |
| `StreamRef::queue_raw_buffer` | method | 0 | wayland | `(*Stream).QueueRawBuffer` | done |
| `StreamState` | enum | 0 | wayland | `StreamState` | done |
| `StreamFlags::AUTOCONNECT`, `MAP_BUFFERS` | const | 0 | wayland | `StreamFlagAutoconnect` | done |
| `Direction::Input` | const | 0 | wayland | `DirectionInput` | done |
| `spa::pod::object!` / `property!` / `Value` | macro | 0 | wayland | `Object` | done |
| `PodSerializer::serialize` | assoc-fn | 0 | wayland | `Serialize` | done |
| `Pod::from_bytes` | assoc-fn | 0 | wayland | `PodFromC` | done |
| `format_utils::parse_format` (`spa_format_parse`) | fn | 0 | wayland | `FormatParse` | done |
| `VideoInfoRaw::parse` (`spa_format_video_raw_parse`) | method | 0 | wayland | `(*VideoInfoRaw).Parse` | done |
| `spa_buffer`, `spa_meta`, `spa_data`, `spa_meta_header`, `pw_buffer` | struct | 0 | wayland | `SpaBuffer` | done |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-SCAP-ROOTTYPES | scap's `Target`, `Display`, `Window`, `Options`, `Area`, `Resolution`, `FrameType` and the frame structs are the root's `gpui.Scap*` types (one set shared by the seam and every back end); `Area` is flattened to x, y, width, height | `src/targets/mod.rs`, `src/capturer/mod.rs`, `src/frame/mod.rs` | `platform.go` |
| D-SCAP-MODS | Rust module declarations have no Go counterpart; the modules are files of one package | `src/lib.rs` | - |
| D-SCAP-NONLINUX | macOS and Windows items are not part of the Linux back end (ported by M4 stages) | `src/lib.rs:17`, `src/capturer/engine/mod.rs` | - |
| D-SCAP-CTEXT | COMPOUND_TEXT window titles are decoded in Go (ASCII, Latin-1 and the UTF-8 escape) instead of Xlib's `XmbTextPropertyToTextList`; other ISO 2022 charsets give "n/a" | `src/targets/linux/mod.rs:51` | `platform/linux/internal/screencap/linux_targets.go` |
| D-SCAP-XSHM | frames are read through MIT-SHM `GetImage` when the server supports it, falling back to core `GetImage` (upstream always uses core `GetImage`); the bytes are the same | `src/capturer/engine/linux/x11/mod.rs:121` | `platform/linux/internal/screencap/x11.go` |
| D-SCAP-PWLIFETIME | the stream, listener, core and context live until the capture loop ends; upstream drops them when `start_pipewire_capturer` returns, which destroys the stream before any frame arrives | `src/capturer/engine/linux/wayland/mod.rs:193` | `platform/linux/internal/screencap/wayland.go` |
| D-SCAP-NULLDATA | a buffer whose first data plane is not mapped (`data == NULL`) yields no frame; upstream builds a slice from the null pointer (undefined behaviour). A NULL `pw_buffer` from dequeue is not handed back to `pw_stream_queue_buffer`, which would dereference it now that the stream stays alive (D-SCAP-PWLIFETIME) | `src/capturer/engine/linux/wayland/mod.rs:157` | `platform/linux/internal/screencap/wayland.go` |
| D-SCAP-DBUSCODEGEN | dbus-codegen's `AppendAll` for the Response signal is only used to emit it; the port only reads it | `src/capturer/engine/linux/wayland/portal.rs:134` | - |
| D-SCAP-UNUSEDFRAMES | `RGB8Frame` is not a `Frame` variant and is never produced | `src/frame/mod.rs:21` | - |
| D-SCAP-BIN | the zed-scap demo binary is not ported | `src/main.rs` | - |
| D-SCAP-QUEUE | the frame channel holds at most 8 items and drops the oldest when full; upstream's `mpsc::channel` is unbounded and grows by a frame per tick while the consumer stalls | `src/capturer/mod.rs` | `platform/linux/internal/screencap/queue.go` |
| D-SCAP-PACKED | Wayland frames honour `chunk.offset`, `chunk.size` and `chunk.stride` and are repacked to `width*bpp` rows (matching `gpui.ScapPixels` and the X11 path); a chunk smaller than the frame gives an error. Upstream copies `maxsize` raw bytes. RGBA is no longer offered in EnumFormat (upstream offers it and then rejects every RGBA frame) | `src/capturer/engine/linux/wayland/mod.rs:157`, `:221` | `platform/linux/internal/screencap/wayland.go` |
| D-SCAP-STATE | the Wayland capturer state and stream-error flag are per capturer, not process statics, so capturers do not start, stop or fail each other | `src/capturer/engine/linux/wayland/mod.rs:33` | `platform/linux/internal/screencap/wayland.go` |
| D-SCAP-CLOSE | `Capturer.Close` stops and joins the capture goroutine (X11 and Wayland) before releasing the connection, so a capturer closed without `stop_capture` leaks nothing; upstream leaks the thread in that case | `src/capturer/mod.rs` | `platform/linux/internal/screencap/engine.go`, `x11.go`, `wayland.go` |
| D-SCAP-ERRSTOP | an X11 grab error is sent once and ends the capture (the queue closes) instead of being re-sent every frame; a cursor overlay is skipped when the image is smaller than width*height*4 (non-32bpp visuals) instead of panicking | `src/capturer/engine/linux/x11/mod.rs` | `platform/linux/internal/screencap/x11.go` |
| D-SCAP-PORTAL | the portal Response match is added before each request call (filtered by the returned handle), avoiding the signal race; `Start` waits 1200x100ms like `SelectSources` since the picker can appear there; an unknown response code is an error, not `unreachable!()` | `src/capturer/engine/linux/wayland/portal.rs` | `platform/linux/internal/screencap/portal.go` |
