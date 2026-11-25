# Round 1 raw findings (pre-dedupe)

Status key:
- **R** = reproduced by a breaker; the harness test FAILs.
- **S** = static review only.

## Decoders (B1 = R)

- **F1** Raster header dimensions are allocated with no budget.
  - `platform.go:4003/4609/3932` (std/ICO/imageIntoRGBA8) and `:5060` (PNM).
  - Tests: `TestS01_03/06/07/08/09/14/15/22`.
- **F2** The PNM/PAM size computation wraps and then panics.
  - `platform.go:5058/5060/5145`.
  - Tests: `TestS01_04/05`.
- **F3** GIF logical screen and animated WebP have no cross-frame budget.
  - `platform.go:4054/4069/4362`.
  - Tests: `TestS01_01/02/12`.
- **F4** okage's default SVG image-href resolver reads local files, including `/dev/zero` and FIFOs.
  - Sink: `svg_renderer.go:256`.
  - Tests: `TestS02_01/02/04/05`.
- **F5** A `data:` PNG inside an SVG bypasses the 8192 clamp.
  - Test: `TestS02_06`.
- **F6** go-text panics on a malformed glyf/GDEF.
  - Reached through font fallback in `LayoutLine`.
  - Tests: `TestS03_05/08/09`.
- **F7** CBDT strike dimensions wrap and then allocate.
  - `strike.go:209/366`, `scale_bitmap.go:107`.
  - Tests: `TestS03_01/02`.
- **F8** SVG filter CPU amplification.
  - Tests: `TestS02_08/08b`.
- **F9** Path resource reads `/dev/zero` or a FIFO.
  - `elements_img.go:594`.
  - Tests: `TestS01_20/21`.
- **S** S03 strike count is trusted (`strike.go:74`): an O(4e9) loop.
- **S** S03 `swashBitmap.decode` length check is missing (`strike.go:332`, explicit panic).
- **S** S03 zeno/scale size math is uint32 (`mask.go:46`, `scale.go:380`).
- **S** S02 SVG renders are retained in the asset cache, up to 256 MiB each.

## Open, fetch (B2 = R)

- **S16-1** Unbounded `io.ReadAll` of the image body.
  - `elements_img.go:609`.
  - Gzip bomb amplifier via the support client.
- **S16-2** Unbounded read in the support HTTP client.
  - `examples/internal/support/http_client.go:44`.
- **S16-3** Error body is echoed in full, including control characters.
  - `elements_img.go:614-624`.
- **S16-4** Image requests cannot be cancelled and have no deadline.
  - `elements_img.go:601`.
- **S04-1** `RevealPath` on a FIFO blocks a worker, and enough of them starve the pool.
  - `common.go:497`.
- **S04-2** `OpenWithSystem` passes a leading-dash path to `xdg-open`.
  - `platform/linux/platform.go:721`.
- **S04-3** `OpenURL` has no scheme allowlist on any platform.
  - Linux `common.go:471`, Windows `ShellExecuteW` (`platform/windows/platform.go:1515`, S), web `window.open` with no noopener (`platform/web/platform.go:413`, S), macOS/iOS (S).
- **S** UNC paths on Windows in `RevealPath`/`OpenWithSystem`.
- **S** Web clipboard blob read is unbounded.

## Session IPC (B3 = R)

- **B3-F1** Forged notification `ActionInvoked`: no sender pin.
  - `dbus/system_notifications.go:73-101`.
- **B3-F2** Forged FileChooser `Response`: no sender pin, no timeout.
  - `platform/linux/platform.go:652-689`.
- **B3-F3** Forged Secret Service prompt `Completed`.
  - `secret_service.go:166-188`.
- **B3-F4** Secret Service falls back to a plain session.
  - `secret_service.go:76-84`.
  - Same fallback upstream.
- **B3-F5** Keymap fd size is not checked, causing SIGBUS.
  - `internal/linux/xkb/xkb.go:181`.
  - Also a panic on a bad keymap at `wayland/client_input.go:120`.
- **B3-F6** xrm `#include`: `/dev/zero`, FIFO, and 2^100 fan-out.
  - `xrm.go:196-211`.
- **B3-F7** xcursor allocates `4*w*h` before reading, and the TOC fans out.
  - `xcursor/parser.go:85/125`.
- **B3-F8** Clipboard `ReadFdWithTimeout` has no total deadline.
  - `common.go:628`.
- **B3-F9** File-URL converters accept NUL, `..`, and `%2F`.
  - INFO.
- **S** XDND spoofing.
  - No sender or selection check at `x11/client.go:1103-1205`.
  - 1024-word truncation.
- **S** INCR `minDataLen` is passed to `Grow` before the cap.
  - `x11/clipboard.go:535`.
- **S** INCR serving state is never pruned.
  - `x11/clipboard.go:645`.
- **S** Direct clipboard read is not capped.
  - `x11/clipboard.go:513`.
- **S** Wayland offer MIME append is quadratic.
  - `wayland/clipboard.go:62`.
- **S** XIM `XConnect` sender is not checked.
  - `xim/client.go:235`.
- **S** Settings `SettingChanged` is not sender-pinned.
  - `xdg_desktop_portal.go:190-206`.

## Runtime, parsers (B4 = R)

- **B4-F1** Asset cache and `Image.ID` keyed by an unkeyed 64-bit FxHash, so collisions can be forged.
  - `asset_cache.go:181`, `elements_image_cache.go:222`, `platform.go:3536`.
- **B4-F2** `TruncateWrappedLine` panics on invalid UTF-8.
  - `text_system_line_wrapper.go:530` (sibling at `:363`).
- **B4-F3** `KeyContextParse` recurses infinitely.
  - `keymap_context.go:109`.
- **B4-F4** Predicate parser has no depth limit.
  - `keymap_context.go:598/628`.
- **B4-F5** Font feature and fallback intern tables are never evicted.
  - `liga` uint32 truncation.
- **B4-F6** Asset cache is unbounded and errors never expire.
  - `asset_cache.go:186`.
- **B4-F7** A background-task panic kills the process.
  - `platform/linux/internal/common/dispatcher.go:41`, `scheduler/future.go:168/204`.
- **B4-F8** jsonschema overflows on recursive non-struct types.
  - `jsonschema/reflect.go:89/122`.
  - INFO.
- **B4-F9** Bidi overrides are rendered.
  - INFO.

## Static only (reviewers)

- **AT-SPI**
  - Paragraph-granularity livelock while holding the tree `RLock` (`a11y/consumer/text.go:500-510`).
  - `GetItems` is unbounded.
  - Error text is echoed back to the peer.
  - NaN accepted for `SetCurrentValue`.
  - Note: B3's fuzz passed, but it did not build the empty-first-run shape.
- **Screencap**
  - `copyFrameData` uint64 wrap (`screencap/wayland.go:166-172`).
  - Metas `Data` nil, `NMetas` unbounded (`:76-79`).
  - `PodFromC` size (`pw/spa.go:480`).
- **wl `goString` unbounded**: INFO.
- **xcbshim setup walk**: INFO.
- **Windows**
  - `TASKDIALOGCONFIG` offsets wrong (`platform/windows/window.go:675-731`).
  - `HDROP` lock mismatch (`clipboard.go:141/352`, `window.go:1119`).
  - Jump-list quoting (`destination_list.go:246`).
  - Raw `lparam` dereferences.
  - Bare-name `LoadLibraryA` of `amd_ags_x64.dll`/`nvapi64.dll` (`util.go:158`).
  - UIA adapters lack recover (`a11y/windows/node.go:1772`, `text.go:424/610/704`).
  - Reentrant `borrowMut` panic (`a11y/windows/subclass.go:58`).
  - UIA move count is unbounded (`text.go:196`).
  - IME length allocation.
- **Apple**
  - Keychain update is not account-scoped (`platform/apple/keychain.go:176`).
  - `nsURLFromDirectoryPath` has no UTF-8 guard (`platform/macos/platform.go:863`).
  - Pasteboard is uncapped.
- **Linux notification body markup**: not escaped (`dbus/system_notifications.go:56-84`).
- **NOTICE** lacks per-crate copyright and licence texts.
- **Fallback interning** (duplicate of B4-F5).
- **Failed `PaintSvg` is not negatively cached** (`window_paint.go:2428`).
