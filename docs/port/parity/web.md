# Parity: web

Scope: gpui_web. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_web/src/canvas_fallback.rs

Go target: `platform/web/canvas_fallback.go` (package `web`), tests in `canvas_fallback_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CanvasFontFallback` | enum | 19 |  | `CanvasFontFallback` | todo |
| `CanvasFallback` | struct | 42 |  | `CanvasFallback` | todo |
| `classify_canvas_fallback` | fn | 51 |  | `ClassifyCanvasFallback` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `canvas_font_fallback_policy` | test | 234 |  | `TestCanvasFallback_CanvasFontFallbackPolicy` | todo |
| `ascii_is_ineligible_but_keycaps_are_preserved` | test | 262 |  | `TestCanvasFallback_AsciiIsIneligibleButKeycapsArePreserved` | todo |
| `ordinary_cjk_clusters` | test | 284 |  | `TestCanvasFallback_OrdinaryCjkClusters` | todo |
| `emoji_presentation_is_preserved` | test | 323 |  | `TestCanvasFallback_EmojiPresentationIsPreserved` | todo |
| `unsupported_scripts_and_cjk_forms_stay_on_cosmic` | test | 372 |  | `TestCanvasFallback_UnsupportedScriptsAndCjkFormsStayOnCosmic` | todo |
| `malformed_or_out_of_policy_emoji_stay_on_cosmic` | test | 430 |  | `TestCanvasFallback_MalformedOrOutOfPolicyEmojiStayOnCosmic` | todo |
| `accepts_only_one_whole_extended_grapheme` | test | 461 |  | `TestCanvasFallback_AcceptsOnlyOneWholeExtendedGrapheme` | todo |

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
| `WebEventListeners` | struct | 15 |  | `WebEventListeners` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `browser_pointer_id_reuse_gets_a_new_touch_id` | test | 1488 |  | `TestEvents_BrowserPointerIDReuseGetsANewTouchID` | todo |
| `shift_is_dropped_only_from_symbol_keys` | test | 1504 |  | `TestEvents_ShiftIsDroppedOnlyFromSymbolKeys` | todo |

## gpui_web/src/glyph_cache.rs

Go target: `platform/web/glyph_cache.go` (package `web`), tests in `glyph_cache_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `warm_lookup_borrows_text_and_preserves_storage` | test | 59 |  | `TestGlyphCache_WarmLookupBorrowsTextAndPreservesStorage` | todo |
| `font_and_presentation_have_distinct_ids` | test | 95 |  | `TestGlyphCache_FontAndPresentationHaveDistinctIDs` | todo |
| `insertion_rechecks_a_previous_miss` | test | 123 |  | `TestGlyphCache_InsertionRechecksAPreviousMiss` | todo |

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
| `candidates_capture_bounds_without_a_flattened_vector` | test | 219 |  | `TestRunReplacements_CandidatesCaptureBoundsWithoutAFlattenedVector` | todo |
| `candidates_keep_graphemes_across_same_font_boundaries` | test | 264 |  | `TestRunReplacements_CandidatesKeepGraphemesAcrossSameFontBoundaries` | todo |
| `candidates_handle_nonmonotonic_and_disjoint_source_indices` | test | 296 |  | `TestRunReplacements_CandidatesHandleNonmonotonicAndDisjointSourceIndices` | todo |
| `candidates_preserve_native_coverage_and_color` | test | 332 |  | `TestRunReplacements_CandidatesPreserveNativeCoverageAndColor` | todo |
| `no_replacements_preserves_even_empty_runs` | test | 476 |  | `TestRunReplacements_NoReplacementsPreservesEvenEmptyRuns` | todo |
| `whole_run_reuses_both_allocations` | test | 490 |  | `TestRunReplacements_WholeRunReusesBothAllocations` | todo |
| `splits_reuse_outer_prefix_unaffected_and_emptied_buffers` | test | 513 |  | `TestRunReplacements_SplitsReuseOuterPrefixUnaffectedAndEmptiedBuffers` | todo |
| `multiple_splits_and_shifts` | test | 540 |  | `TestRunReplacements_MultipleSplitsAndShifts` | todo |
| `exhaustive_small_streams_match_reference` | test | 559 |  | `TestRunReplacements_ExhaustiveSmallStreamsMatchReference` | todo |

## gpui_web/src/window.rs

Go target: `platform/web/window.go` (package `web`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `WebWindow` | struct | 93 |  | `WebWindow` | dev:WEB-CANVASID |


## Notes (M4-web-platform)

- dispatcher.rs: only the single-threaded path exists; `MainThreadMailbox`, the waker loop, background workers and `allow_threads` are absent (L-WEBTHREADS). `IsMainThread` is always true.
- viewport.rs and gpui_web.rs have no public items; ported into viewport.go and gpui_web.go (gpui_web.go also holds the syscall/js helpers: exception recovery, self-releasing once funcs, promise await through scheduler.Oneshot).
- events.rs, ime_mirror.rs, text_system.rs and canvas_fallback.rs symbols used here are package-local stubs in stubs.go until M4-web-input / M4-web-text; `EventListenerHandle` is implemented there already.
- Auto backend: `wgpu::util::is_browser_webgpu_supported` is ported locally (navigator.gpu + requestAdapter non-null).

## Deviations registered here

| Id | Item | Difference |
|---|---|---|
| WEB-CANVASID | `WebWindow::prepare_canvas`, `HasWindowHandle` | The canvas gets a unique `id` (`gpui-canvas-N`) because gpui.RawWindowHandle names a web canvas by id (render/wgpu surface recovery looks it up). |
| WEB-REDIRECT | `FetchHttpClient::send` | gpui.HTTPClient (D-HTTP) takes a net/http request with no `RedirectPolicy` extension, so fetch always uses the browser default (follow). Request bodies are read synchronously inside the send task; response bodies must be read from a goroutine, not a task or browser callback. |
