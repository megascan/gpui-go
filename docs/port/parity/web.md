# Parity: web

Scope: gpui_web. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_web/src/canvas_fallback.rs

Go target: `platform/web/canvas_fallback.go` (package `web`), tests in `canvas_fallback_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CanvasFontFallback` | enum | 19 |  | `CanvasFontFallback` | done |
| `CanvasFallback` | struct | 42 |  | `CanvasFallback` | done |
| `classify_canvas_fallback` | fn | 51 |  | `ClassifyCanvasFallback` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `canvas_font_fallback_policy` | test | 234 |  | `TestCanvasFallback_CanvasFontFallbackPolicy` | done |
| `ascii_is_ineligible_but_keycaps_are_preserved` | test | 262 |  | `TestCanvasFallback_AsciiIsIneligibleButKeycapsArePreserved` | done |
| `ordinary_cjk_clusters` | test | 284 |  | `TestCanvasFallback_OrdinaryCjkClusters` | done |
| `emoji_presentation_is_preserved` | test | 323 |  | `TestCanvasFallback_EmojiPresentationIsPreserved` | done |
| `unsupported_scripts_and_cjk_forms_stay_on_cosmic` | test | 372 |  | `TestCanvasFallback_UnsupportedScriptsAndCjkFormsStayOnCosmic` | done |
| `malformed_or_out_of_policy_emoji_stay_on_cosmic` | test | 430 |  | `TestCanvasFallback_MalformedOrOutOfPolicyEmojiStayOnCosmic` | done |
| `accepts_only_one_whole_extended_grapheme` | test | 461 |  | `TestCanvasFallback_AcceptsOnlyOneWholeExtendedGrapheme` | done |

## gpui_web/src/dispatcher.rs

Go target: `platform/web/dispatcher.go` (package `web`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WebDispatcher` | struct | 149 |  | `WebDispatcher` | dev:L-WEBTHREADS |
| `WebDispatcher::new` | assoc-fn | 159 |  | `NewWebDispatcher` | done |

## gpui_web/src/display.rs

Go target: `platform/web/display.go` (package `web`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WebDisplay` | struct | 5 |  | `WebDisplay` | done |
| `WebDisplay::new` | assoc-fn | 19 |  | `NewWebDisplay` | done |

## gpui_web/src/events.rs

Go target: `platform/web/events.go` (package `web`), tests in `events_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WebEventListeners` | struct | 15 |  | `WebEventListeners` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `browser_pointer_id_reuse_gets_a_new_touch_id` | test | 1488 |  | `TestEvents_BrowserPointerIDReuseGetsANewTouchID` | done |
| `shift_is_dropped_only_from_symbol_keys` | test | 1504 |  | `TestEvents_ShiftIsDroppedOnlyFromSymbolKeys` | done |

## gpui_web/src/glyph_cache.rs

Go target: `platform/web/glyph_cache.go` (package `web`), tests in `glyph_cache_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `warm_lookup_borrows_text_and_preserves_storage` | test | 59 |  | `TestGlyphCache_WarmLookupBorrowsTextAndPreservesStorage` | done |
| `font_and_presentation_have_distinct_ids` | test | 95 |  | `TestGlyphCache_FontAndPresentationHaveDistinctIDs` | done |
| `insertion_rechecks_a_previous_miss` | test | 123 |  | `TestGlyphCache_InsertionRechecksAPreviousMiss` | done |

## gpui_web/src/http_client.rs

Go target: `platform/web/http_client.go` (package `web`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FetchHttpClient` | struct | 19 |  | `FetchHTTPClient` | dev:WEB-REDIRECT |
| `FetchCredentials` | enum | 27 |  | `FetchCredentials` | done |
| `FetchHttpClient::new` | assoc-fn | 39 |  | `NewFetchHTTPClient` | done |
| `FetchHttpClient::with_user_agent` | assoc-fn | 48 |  | `FetchHTTPClientWithUserAgent` | done |
| `FetchHttpClient::with_credentials` | method | 61 |  | `(FetchHTTPClient).WithCredentials` | done |

## gpui_web/src/keyboard.rs

Go target: `platform/web/keyboard.go` (package `web`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WebKeyboardLayout` | struct | 3 |  | `WebKeyboardLayout` | done |

## gpui_web/src/logging.rs

Go target: `platform/web/logging.go` (package `web`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `init_logging` | fn | 34 |  | `InitLogging` | done |

## gpui_web/src/platform.rs

Go target: `platform/web/platform.go` (package `web`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WebPlatform` | struct | 32 |  | `WebPlatform` | done |
| `WebWindowError` | enum | 97 |  | `WebWindowError` | done |
| `WebPlatform::new` | assoc-fn | 147 |  | `NewWebPlatform` | done |
| `WebPlatform::new_with_backend` | assoc-fn | 151 |  | `NewWebPlatformWithBackend` | done |
| `WebPlatform::new_with_backend_and_font_fallback` | assoc-fn | 166 |  | `NewWebPlatformWithBackendAndFontFallback` | done |
| `WebPlatform::fetch_http_client` | method | 217 |  | `(WebPlatform).FetchHTTPClient` | done |
| `WebPlatform::fetch_http_client_with_user_agent` | method | 222 |  | `(WebPlatform).FetchHTTPClientWithUserAgent` | done |

## gpui_web/src/run_replacements.rs

Go target: `platform/web/run_replacements.go` (package `web`), tests in `run_replacements_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `candidates_capture_bounds_without_a_flattened_vector` | test | 219 |  | `TestRunReplacements_CandidatesCaptureBoundsWithoutAFlattenedVector` | done |
| `candidates_keep_graphemes_across_same_font_boundaries` | test | 264 |  | `TestRunReplacements_CandidatesKeepGraphemesAcrossSameFontBoundaries` | done |
| `candidates_handle_nonmonotonic_and_disjoint_source_indices` | test | 296 |  | `TestRunReplacements_CandidatesHandleNonmonotonicAndDisjointSourceIndices` | done |
| `candidates_preserve_native_coverage_and_color` | test | 332 |  | `TestRunReplacements_CandidatesPreserveNativeCoverageAndColor` | done |
| `no_replacements_preserves_even_empty_runs` | test | 476 |  | `TestRunReplacements_NoReplacementsPreservesEvenEmptyRuns` | done |
| `whole_run_reuses_both_allocations` | test | 490 |  | `TestRunReplacements_WholeRunReusesBothAllocations` | done |
| `splits_reuse_outer_prefix_unaffected_and_emptied_buffers` | test | 513 |  | `TestRunReplacements_SplitsReuseOuterPrefixUnaffectedAndEmptiedBuffers` | done |
| `multiple_splits_and_shifts` | test | 540 |  | `TestRunReplacements_MultipleSplitsAndShifts` | done |
| `exhaustive_small_streams_match_reference` | test | 559 |  | `TestRunReplacements_ExhaustiveSmallStreamsMatchReference` | done |

## gpui_web/src/window.rs

Go target: `platform/web/window.go` (package `web`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WebWindow` | struct | 93 |  | `WebWindow` | dev:WEB-CANVASID |


## gpui_web/examples/hello_web/main.rs

Go target: `examples/web_hello_web/main.go` (package `main`, `js && wasm`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `is_prime` | fn | 14 |  | `IsPrime` | done |
| `count_primes_in_range` | fn | 34 |  | `CountPrimesInRange` | done |
| `NUM_CHUNKS` | const | 48 |  | `NumChunks` | done |
| `Preset` | enum | 51 |  | `Preset` | done |
| `Preset::TenMillion` | variant | 52 |  | `PresetTenMillion` | done |
| `Preset::FiftyMillion` | variant | 53 |  | `PresetFiftyMillion` | done |
| `Preset::HundredMillion` | variant | 54 |  | `PresetHundredMillion` | done |
| `Preset::label` | method | 58 |  | `(Preset).Label` | done |
| `Preset::value` | method | 66 |  | `(Preset).Value` | done |
| `ChunkResult` | struct | 81 |  | `ChunkResult` | done |
| `Run` | struct | 85 |  | `Run` | done |
| `HelloWeb` | struct | 93 |  | `HelloWeb` | done |
| `HelloWeb::new` | assoc-fn | 101 |  | `NewHelloWeb` | done |
| `HelloWeb::start_search` | method | 110 |  | `(HelloWeb).StartSearch` | done |
| `format_number` | fn | 171 |  | `FormatNumber` | done |
| `BG_BASE` | const | 187 |  | `BgBase` | done |
| `BG_SURFACE` | const | 188 |  | `BgSurface` | done |
| `BG_OVERLAY` | const | 189 |  | `BgOverlay` | done |
| `TEXT_PRIMARY` | const | 190 |  | `TextPrimary` | done |
| `TEXT_SECONDARY` | const | 191 |  | `TextSecondary` | done |
| `TEXT_DIM` | const | 192 |  | `TextDim` | done |
| `ACCENT_YELLOW` | const | 193 |  | `AccentYellow` | done |
| `ACCENT_GREEN` | const | 194 |  | `AccentGreen` | done |
| `ACCENT_BLUE` | const | 195 |  | `AccentBlue` | done |
| `ACCENT_MAUVE` | const | 196 |  | `AccentMauve` | done |
| `HelloWeb::render` (`Render` impl) | method | 199 |  | `(HelloWeb).Render` | done |
| `requested_backend` | fn | 411 |  | `RequestedBackend` | done |
| `APPLICATION` | thread-local-static | 433 |  | `application` | dev:WEB-EXAMPLE-DISPATCH |
| `main` | fn | 436 |  | `main` | dev:WEB-EXAMPLE-DISPATCH |

## gpui_web/examples/hello_web/dynamic_fonts.rs

Go target: `examples/web_hello_web/dynamic_fonts.go` (package `main`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FONT_URL` | const | 12 |  | `FontURL` | done |
| `SAMPLE` | const | 13 |  | `Sample` | done |
| `DynamicFonts` | struct | 15 |  | `DynamicFonts` | done |
| `DynamicFonts::new` | assoc-fn | 22 |  | `NewDynamicFonts` | done |
| `DynamicFonts::load_font` | method | 41 |  | `(DynamicFonts).LoadFont` | done |
| `DynamicFonts::render` (`Render` impl) | method | 80 |  | `(DynamicFonts).Render` | done |
| `main` | fn | 106 |  | `RunDynamicFonts` | dev:WEB-EXAMPLE-DISPATCH |

## gpui_web/examples/hello_web/font_fallback_test.rs

Go target: `examples/web_hello_web/font_fallback_testing.go` (package `main`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Reports` | struct | 11 |  | `Reports` | done |
| `MissingGlyphSink::report` for `Reports` | method | 14 |  | `(Reports).Report` | done |
| `test_missing_glyph_notifications` | wasm_bindgen fn | 20 |  | `testMissingGlyphNotifications` | dev:WEB-EXAMPLE-FIXTURE |
| `main` | fn | 8 |  | `registerFontFallbackTest` | dev:WEB-EXAMPLE-DISPATCH |

## gpui_web/examples/hello_web/http_client_test.rs

Go target: `examples/web_hello_web/http_client_testing.go` (package `main`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `test_worker_fetch` | wasm_bindgen async fn | 16 |  | `registerHTTPClientTest` | dev:WEB-EXAMPLE-FIXTURE |
| `main` | fn | 13 |  | `registerHTTPClientTest` | dev:WEB-EXAMPLE-DISPATCH |

## gpui_web/examples/hello_web/queue_test.rs

Go target: `examples/web_hello_web/queue_testing.go` (package `main`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `QueueStress` | struct | 14 |  | `QueueStress` | done |
| `QueueStress::new` | assoc-fn | 28 |  | `NewQueueStress` | done |
| `QueueStress::ready` | method | 62 |  | `(QueueStress).Ready` | done |
| `QueueStress::send_batch` | method | 66 |  | `(QueueStress).SendBatch` | done |
| `QueueStress::drain` | method | 76 |  | `(QueueStress).Drain` | done |
| `QueueStress::stop` | method | 90 |  | `(QueueStress).Stop` | done |
| `QueueStress::stopped` | method | 102 |  | `(QueueStress).Stopped` | done |
| `priority` | fn | 107 |  | `Priority` | done |
| `forbidden_main_thread_wait` | wasm_bindgen fn | 117 |  | `-` | n/a:L-WEBTHREADS |
| `main` | fn | 10 |  | `registerQueueTest` | dev:WEB-EXAMPLE-DISPATCH |

## Notes (M4-web-platform)

- dispatcher.rs: only the single-threaded path exists; `MainThreadMailbox`, the waker loop, background workers and `allow_threads` are absent (L-WEBTHREADS). `IsMainThread` is always true.
- viewport.rs and gpui_web.rs have no public items; ported into viewport.go and gpui_web.go (gpui_web.go also holds the syscall/js helpers: exception recovery, self-releasing once funcs, promise await through scheduler.Oneshot).
- events.rs, ime_mirror.rs, text_system.rs and canvas_fallback.rs symbols used here are package-local stubs in stubs.go until M4-web-input / M4-web-text; `EventListenerHandle` is implemented there already.
- Auto backend: `wgpu::util::is_browser_webgpu_supported` is ported locally (navigator.gpu + requestAdapter non-null).

## Notes (M4-web-input)

- events.rs → events.go, ime_mirror.rs → ime_mirror.go (no public items). Event handlers read DOM events through syscall/js; the IME mirror's value/selection logic runs against the `mirrorTextArea` / `mirrorInputHandler` interfaces so sync, window and diff logic is tested under node without a DOM (ime_mirror_test.go, events_test.go).
- The input-diff of `register_input` is factored into `diffImeEdit`; pasted-image reads chain `arrayBuffer` promises instead of `spawn_local` (same order, same handler-at-resolution semantics).
- `sync` takes the input handler once for the whole sync rather than per call; nothing re-enters the window between the calls, so behaviour is the same.

## Notes (M4-web-text)

- canvas_fallback.go holds `CanvasFontFallback` and a port of unicode-properties 0.1.3's `EMOJI_STATUS` range table (the version gpui_web pins), searched by binary search; graphemes come from go-text/typesetting/segmenter, Han script and `Lo` from package unicode.
- canvas_text.go, glyph_cache.go, run_replacements.go and text_system.go have no public items. `CANVAS_FONT_BIT` is the top bit of `gpui.FontID` (`uint`, 64-bit on Go wasm; upstream's wasm32 usize gives bit 31): same logic.
- canvas_text's thread-local canvas is a package-level slot with a reentrancy flag, created lazily, so node tests never touch OffscreenCanvas.
- `SmallVec`/`retain_mut`/`drain` in run_replacements become in-place compaction and reslicing, so the allocation-reuse tests check the same backing arrays.
- platform/platform_js.go now returns the web platform; `web_init`'s console_error_panic_hook has no Go equivalent to install (Go wasm panics already print to the console), so WebInit only calls InitLogging.

## Notes (M4-web-example)

- One wasm binary serves the example's five upstream bins. The loader (gallery.js, the drivers) sets `go.argv` to the bin's module name and `main` dispatches on `os.Args[1]`; the default is the prime sieve. Each bin's `fn main` therefore has a dispatch row (WEB-EXAMPLE-DISPATCH).
- The three `*_test.rs` fixtures are wasm_bindgen exports, not `#[test]`s: they are registered on `globalThis.gpuiWebTests` by their `register*` functions, so the drivers call them from JS.
- Response bodies in dynamic_fonts are read on a goroutine and handed back through `scheduler.Oneshot`, not inside a task (WEB-REDIRECT).
- queue_test's workers are goroutines on the single Go wasm thread (L-WEBTHREADS). The queue contention driver runs in node (`test_queue.mjs`, 524288 items, 8 workers: PASS). The browser-only checks run in the owner-check drivers.
- Fonts: `fonts/IBMPlexSans-Regular.ttf` and `fonts/Lilex-Regular.ttf` are copies of `text/gotext/testdata/fonts` (go:embed cannot reach outside the package); the upstream `assets/fonts` tree is not in `_ref`.
- The Rust gallery's other links (hello-world, text, text-layout, text-wrapper, input) are built by their own example packages as `<module>.wasm`; this package builds `gpui-web-examples.wasm` for `hello-web` and `dynamic-fonts`.
- Build: `GOOS=js GOARCH=wasm go build -o <dir>/gpui-web-examples.wasm ./examples/web_hello_web`, and copy `$(go env GOROOT)/lib/wasm/wasm_exec.js` next to `index.html` when serving.

## Deviations registered here

| Id | Item | Difference |
|---|---|---|
| WEB-CANVASID | `WebWindow::prepare_canvas`, `HasWindowHandle` | The canvas gets a unique `id` (`gpui-canvas-N`) because gpui.RawWindowHandle names a web canvas by id (render/wgpu surface recovery looks it up). |
| WEB-REDIRECT | `FetchHttpClient::send` | gpui.HTTPClient (D-HTTP) takes a net/http request with no `RedirectPolicy` extension, so fetch always uses the browser default (follow). Request bodies are read synchronously inside the send task; response bodies must be read from a goroutine, not a task or browser callback. |
| WEB-EXAMPLE-DISPATCH | each bin's `fn main` | One wasm binary serves the five bins; `main` dispatches on `os.Args[1]` (set by gallery.js and the drivers) to the bin's run or register function. The dispatch rows name the function each bin's `main` maps to. |
| WEB-EXAMPLE-FIXTURE | `test_missing_glyph_notifications`, `test_worker_fetch` | Exported to JS as members of `globalThis.gpuiWebTests`, not Go test names; the Go symbols are the helpers that implement them. |
