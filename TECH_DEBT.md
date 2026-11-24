# Tech debt

Hacks, workarounds and known issues. Remove entries when resolved. Stages write new items to `docs/port/debt/<stage>.md`; exit stages fold them in here.

## Index

| Id | Area | Summary |
|---|---|---|
| L-WGPUVER | render/wgpu | wgpu-native v29 binding against upstream's wgpu 30 |
| L-WEBTHREADS | platform/web | no multi-threaded web dispatcher |
| GO-DIRECTIVE | go.mod | `go 1.27.3` instead of `go 1.27` |
| CI-BILLING | .github | Actions jobs are not started (billing / spending limit, macos-14 runner config) |
| M0-EXIT-OPEN | root | handoff items M0-exit left open |
| STAGE-DEBT | all | per-stage debt folded below |

## L-WGPUVER

Bump github.com/oliverbestmann/webgpu when it ships wgpu-native v30, then re-run the M1-wgpu-context feature probe and drop the fallbacks it no longer needs.

## L-WEBTHREADS

Go wasm is single-threaded; revisit if Go gains wasm threads.

## GO-DIRECTIVE

github.com/xo/okage v0.20.1 declares `go 1.27.3`, so the main module's go line cannot be lower than that; go.mod says `go 1.27.3` rather than the planned `go 1.27`. CI's `1.27.x` satisfies it. Drop when the module moves to a newer Go anyway.

## CI-BILLING

GitHub Actions refuses to start the ci workflow's jobs on megascan/gpui-go: ubuntu and windows report "recent account payments have failed or your spending limit needs to be increased", macos-14 reports "The configuration 'macos-14-arm64-us-xl' is not supported". Every run on feat/port is cancelled after a few seconds, so the macOS and Windows compile checks have never run. Owner action: fix billing / the spending limit, then re-run the workflow on the draft PR.

## M0-EXIT-OPEN

- `Point[T].GoString` prints large floats as `3.4028235e+38`; Rust `{:?}` prints `3.4028235e38` (M0-scene handoff). Cosmetic, only visible in log output.
- docs/api-contracts.md is stale against the ported code (handoffs of M0-contracts, M0-app-core, M0-app-effects, M0-profiler-core, M0-div-paint list the corrections: DESIGN 6.2 primitives superseded by api-contracts section 2, Spawn/WithWindow/Observe/ToAsync/ForegroundExecutor ownership, stubs.go now holding the journal no-op stubs, the WrappedLine and windowA11y stub list). The file has no owner after M0; update it in the next docs pass.
- DESIGN 5.2 Drop table additions requested by M0-test-contexts: `HeadlessAppContext` -> `Close`; Go-only `Close` on `TestApp`, `VisualTestAppContext`, `TestAppContextSingle`. Also not listed: `ElementInputHandler.Release` (frames release input handlers), `FallbackPromptRenderer.Drop` (releases its focus handle), `Window.drop` (root, prompt, both frames, the platform input handler and the a11y channels, run when app.go removes the window and at shutdown / test teardown). `RendezvousReceiver` has no exported `Close`: the rendezvous is internal to `scheduler.Task` and ends with the task.
- input_test.go keeps `inputTestReleaseHandlerViews`: the test's Render records one `cx.Entity()` clone per render in the view's own state for its canvas paint closure (upstream moves it into an `FnOnce` that drops with the element); Go elements have no drop hook, so the test releases them.
- elementStateMap now releases a dropped state through `dropElementState` only for `InteractiveElementState` (active tooltip). Other element states that own handles (list / uniform-list scroll handles, image cache items, keyed-state entities from `UseKeyedState`) still rely on the GC safety net when they stop being rendered; give them `dropElementState` when one shows up in strict leak mode.

## STAGE-DEBT

Folded by M0-exit from docs/port/debt/<stage>.md for every stage with a docs/port/status/<stage>.done file. Later exits fold only files not listed here: M0-a11y-schema, M0-app-contexts, M0-app-core, M0-app-effects, M0-blocked-tests, M0-div-api, M0-div-paint, M0-element, M0-elements-basic, M0-elements-media, M0-entities, M0-etagere, M0-executors, M0-geometry-units, M0-geometry, M0-gestures-spring, M0-input-types, M0-keymap, M0-lyon, M0-parity-rescan, M0-platform-data, M0-platform-types, M0-profiler-core, M0-scene, M0-scheduler-core, M0-scheduler-test, M0-style, M0-sumtree, M0-taffy-compute, M0-taffy-fixtures, M0-taffy-grid-types, M0-taffy-style-grid, M0-taffy-style-mod, M0-taffy-style, M0-taffy-tree, M0-test-contexts, M0-test-platform, M0-text-system, M0-window-a11y, M0-window-core, M0-window-dispatch, M0-window-paint, M0-window-tests, M1-linux-common, M1-linux-dbus, M1-linux-platform, M1-linux-screencap, M1-wayland-client, M1-wayland-ffi, M1-wayland-window, M1-wgpu-context, M1-wgpu-renderer, M1-x11-client, M1-x11-window, M2-gotext-layout, M2-gotext-raster, M3-a11y-atspi-bus, M3-a11y-atspi-node, M3-a11y-consumer-tree, M4-apple-ffi, M4-apple-metal-atlas, M4-apple-text, M4-macos-appkit, M4-macos-platform, M4-macos-window-objc, M4-macos-window, M4-web-input, M4-web-platform, M4-web-webgl-shaders, M4-web-webgl, M4-web-wgpu-js, M4-windows-a11y-node, M4-windows-com, M4-windows-directwrite, M4-windows-directx-renderer.

### M0-a11y-schema

- a11y JSON vs serde_json: f64 values print in Go's shortest form (`1` where serde_json writes `1.0`); values round-trip but bytes differ. Decoding `null` into a non-Option field, or omitting a required field of the plainly-decoded structs (CustomAction, TextPosition, TextSelection, Color, TextDecoration, Tree, TreeUpdate), is accepted where serde errors; Node and ActionRequest check missing fields by hand.
- `deny_unknown_fields` structs decode with `json.RejectUnknownMembers(true)`, which also applies to nested geometry types (serde accepts unknown fields there).
- `debugStr` approximates Rust's `str` Debug escaping (`\u{..}` for non-printable runes via unicode.IsPrint, not Rust's grapheme-extend table).
- Node getters for slice properties return the internal slice (Rust returns `&[T]`); callers must not mutate it.

### M0-app-contexts

- RunTest retries only panics: t.Fatal / t.Error cannot be undone, so such a failure ends the run after OnFailure and the failing-seed report (testing.go).
- Context.Processor captures a strong entity handle for the closure's lifetime, as upstream; under strict leak detection a dropped processor is reported as a collected handle.
- Hash: values that are neither Hashable nor a basic Rust-equivalent type hash through hash/maphash (per-process seed), not FxHasher; Resource hashes a path as a str, not Rust's component-wise Path hash.
- VisualContextUpdateWindowEntity (gpui.go, not owned) uses the context window for AsyncWindowContext; upstream's AsyncWindowContext::update_window_entity uses with_window(view) ("entity has no current window").
- asset_cache.rs tests are blocked:M0-test-contexts (need TestAppContext); asset_cache_test.go has FxHasher vector tests and an inference compile check.
- App.FetchAsset / RemoveAsset / HasAsset (app.rs rows) live in asset_cache.go because app.go is not owned by this stage.

### M0-app-core

- D-BORROWLOC (core-app.md): `TRACK_THREAD_BORROWS` is read from the process environment once at startup (upstream: compile-time `option_env!`); the printed `ThreadId(N)` is the goroutine id.
- Borrow diagnostics print `file:line` from `runtime.Caller` (absolute path, no column); upstream's `Location` prints `file:line:column`.
- Iteration over `FxHashMap` / `HashMap` keys (system window tab groups, display snapshots in `displayEvents`) uses ascending key order; upstream's order is arbitrary. Deterministic, but a different "arbitrary".
- `Shutdown`: a hang-monitor flush the monitor never completes waits for `ShutdownTimeout` without upstream's "hang monitor exited before flushing" warning, because `(*hangMonitor).requestFlush`'s channel is closed only on success. Its future wakes from a helper goroutine.
- stubs.go placeholder bodies so App construction does not panic before their owners land: `NewSvgRenderer` (stores the asset source), `DefaultKeymap`, `defaultFocusMap`, `defaultInspectorElementRegistry`, `installForegroundJournal` (zero journal) and `observePower` (no-op), the last two reached only under `-tags gpui_profiler`.
- `(App).RefCountsDropHandle` returns `any` (upstream `impl Sized`).
- Field / getter collisions: `Window.last_window_bounds()` / `display_id()` are read as the fields `lastWindowBounds` / `displayID` (D-GETTER); `GpuiMode::skip_drawing` is `shouldSkipDrawing`; App's `background_executor` field is `bgExecutor` (the name collides with the AppContext primitive `backgroundExecutor`), also in the AsyncApp stub.
- wasm branches (`Run`'s `mem::forget`, `Shutdown`'s spawn instead of block) are `runtime.GOOS == "js"` checks in app.go, not build-tagged files; `Run` retains the Application in a package-level slice on js.
- `foregroundJournal` / `hangMonitor` App fields are always compiled and set only when `profilerFeatureEnabled` (D-FEATURE).
- `element_arena` / `event_arena` are not ported (D-ARENA).

### M0-app-effects

- `App.Notify` visits a window's invalidators in ascending WindowID order (upstream: FxHashMap order).
- cfg-gated methods (`ReadFromPrimary`, `WriteToPrimary`, `ReadFromFindPasteboard`, `WriteToFindPasteboard`, `WriteCredentialsIfNotExists`, `ClearGlobals`, `HasAsset`-to-be, `GetName`, `SetInspectorRenderer`, `RegisterInspectorElement`, `ForegroundJournal`, `StartHangMonitor`) exist on every GOOS / without feature tags, as the Platform interface does (D-FEATURE).
- `ActionSchemas` / `ActionSchemaByName` take no schema generator (D-SCHEMA: the registry calls each action's reflector); `ActionSchemas` returns exported `ActionSchema{Name, Schema}`.
- `HTTPClient` (D-HTTP) is `Send(*http.Request) *Task[*http.Response]`, `UserAgent`, `Proxy`; none of http_client's provided helpers.
- `flushEffects`' test-mode redraw runs whenever `mode.test` (upstream additionally cfg(test/test-support)); `event_arena.clear()` is dropped (D-ARENA).
- Panic texts for anyhow unwraps are rendered as `called \`Result::unwrap()\` on an \`Err\` value: <msg>`; `ObserveNew` downcast failure omits upstream's Debug of the entity.

### M0-blocked-tests

- input_test.go releases the views captured by `NewElementInputHandler` by hand (`inputTestReleaseHandlerViews`); see M0-EXIT-OPEN.
- input_test.go uses window_test.go's `windowTestCanvas` in place of `canvas(...)`, which M0-elements-basic ports later; switch to `Canvas` once it exists.
- `TestLineWrapper_WrapShapedLine` is gated with a runtime.GOOS skip instead of a darwin build constraint (the gate is one test in a shared file). It uses the NoopTextSystem as upstream does and also passes on Linux.

### M0-div-api

- stubs.go: `ScrollHandle` / `ScrollAnchor` are placeholder structs (M0-div-paint replaces them); `ImageCacheProvider` placeholder interface `Provide(window, cx) AnyImageCache` (M0-elements-media replaces it).
- Style closures (`Hover`, `Focus`, `Active`, `GroupHover`, `DragOver`, ...) take `func(*StyleRefinement) *StyleRefinement` and receive a fresh zero refinement (upstream `FnOnce(StyleRefinement) -> StyleRefinement`).
- `Div.interactivity` is an inline value, not upstream's arena/heap `InteractivityBox` (D-ARENA).
- `Interactive.OnClick` moved to `StatefulInteractive.OnClick` as upstream (it was misplaced in the M0-contracts stub).
- `Div` Element impl (div.rs:1995-2195: ElementID, SourceLocation, A11yRole, WriteA11yInfo, request_layout, prepaint, paint) left to M0-div-paint.

### M0-div-paint

- The focus-on-mouse-down listener captures a `WeakFocusHandle` and upgrades on dispatch instead of holding a counted clone (frame listeners have no drop hook, so a clone would leak). Behaviour differs only when every strong handle is gone while the listener is still in the rendered frame.
- `Interactivity.trackedFocusHandle` (TrackFocus clone or the element-state clone) is released with `deferArenaDrop` at arena clear; a Div that is built but never laid out leaves it to the GC net.
- The `FocusHandle` stored in `InteractiveElementState` is not released when a stale element state is discarded at frame swap (elementStateMap has no drop hook).
- FileDropEvent exit listeners are appended to `nextFrame.mouseListeners` directly: the `FileDropEvent` interface does not satisfy `OnMouseEvent`'s `MouseEvent` constraint.
- elements_div_test.go: the test-local `divTestCanvas` stands in for `canvas` (M0-elements-basic).

### M0-element

- A `ViewElement` for an entity or `AnyView` that is built but never laid out (dropped before `RequestLayout`) never registers with `Window.deferArenaDrop`, so its handle is released only by the GC safety net; strict leak mode reports it. Upstream drops it with the arena.
- `(*AnyElement).DowncastMut[T]` matches only elements stored as `*T`; a value element (such as `Empty{}`) cannot be downcast to a pointer into the drawable.
- `ElementID` placeholder bodies in stubs.go (`NewElementID`, `Hash`, `String`, `GoString`) cover Integer, Name and NamedInteger only; M0-window-core replaces them.
- `ViewElement` component ids use `typeNameOf` (`pkgpath.Name`), not Rust's `type_name` path; ids only need to be unique per type.

### M0-elements-basic

- Animation element state (`animationState`) holds the pending step wake task; upstream cancels it when the state is dropped with the frame. The window's element-state map has no drop hook, so a discarded state's wake still fires and notifies the view once (a spurious re-render, no leak).
- `Surface` reads the CVPixelBuffer size through `RegisterSurfaceSourceSize`; nothing registers it yet (handoff to the Apple stages).
- `UniformListScrollHandle{}` (zero value) has no state and panics on use; upstream's `Default` builds a fresh state. Use `NewUniformListScrollHandle()`.
- `PulsatingBetween` and `Heartbeat` evaluate sin / exp in float64 and round to float32, which can differ from Rust's f32 libm by one ulp.
- `TestUniformList_*` use a local weak-entity processor (`uniformListTestProcessor`) instead of upstream's `cx.processor`, because `Context.Processor` leaks its strong handle (handoff).

### M0-elements-media

- `ImageAssetLoader` recognises all 22 magic-byte signatures of image 0.25.10's `guess_format`, but decodes only formats that `newImageDecoder` (platform.go) supports: PNG, JPEG, GIF, WebP, BMP, TIFF, ICO and PNM. AVIF, DDS, HDR, OpenEXR, QOI and farbfeld fail with "The image format X is not supported". `ImgExtensions` still lists them, as upstream does. gen2brain/avif is a pinned dependency, but nothing wires it in yet.
- `isURI` approximates `url::Url::from_str`. It accepts a WHATWG scheme (`[A-Za-z][A-Za-z0-9+.-]*:`), and for http, https, ws, wss and ftp it also needs `net/url` to parse a host. Inputs where the url crate's full host and path validation disagree with this can end up on the other branch (URI vs embedded).
- `RetainAllImageCacheProvider` stores the cache `Entity` in window element state. The element-state map (window.go `elementStateMap.clear`) never releases dropped states, so under strict leak detection a `RetainAll` cache outlives its element (handoff).
- `AnyImageCache` is weak (D-WEAKIMAGECACHE). An `Img`/`ImageCacheElement` whose cache entity was released panics on load, where upstream kept the entity alive.
- `Svg.Data` hashes with SipHash-1-3, keys 0, over a length prefix plus the bytes. This is std's `DefaultHasher` today. Rust does not promise that this algorithm stays fixed.
- `TimedTransformation.WithMaxFps` truncates `1/max_fps` seconds to nanoseconds. Rust's `Duration::from_secs_f64` may round the last nanosecond.
- The SVG system font database walks the same folders as okage's `LoadOSFonts` one file at a time, so it can record each face's file (needed by `owned_face`). If okage changes its folder list, this copy has to change too.

### M0-entities

- Strict leak reports (handles, subscriptions, leases collected without their drop method) go to one process-global list drained by `entityMap.report()`. Parallel tests in one binary can see each other's reports.
- Strict mode is the default in every test binary (`testing.Testing()` with `GPUI_LEAK_DETECTION` unset), not only this repo's. Downstream users opt out with `GPUI_LEAK_DETECTION=1`.
- A lease abandoned by a panic that a test recovers is reported in strict mode; upstream skips the LeaseInner check while panicking, which Go cannot detect.
- Over-release panics inside a `runtime.AddCleanup` callback run on the cleanup goroutine and abort the process (upstream panics on the dropping thread).
- `Clone` / `Upgrade` after the ref-count table was collected skip leak tracking; upstream's `upgrade().unwrap()` panics there.
- Tasks are not covered by strict collection reports yet; the scheduler owns `Task` (M0-executors / scheduler).
- `debug_assert!` checks (wrong context, dropped-while-referenced, pop without push, scope left open) follow `gpuiutil.DebugAssertions`.

### M0-etagere

- E-COALESCE-OOB: upstream `coalesce_shelves` (`etagere-0.2.15/src/bucketed.rs:441-447`) reads `shelves[shelf_index + i]` before its bound check, so an empty last shelf that is too narrow for the request makes `Allocate` panic with an index-out-of-range error instead of returning none. Kept for parity; gpui's atlases can reach it only when the atlas is full and its last shelf is empty. Fix upstream-compatibly by moving the bound check above the read if gpui hits it.
- E-DEBUGASSERT: upstream `debug_assert!` checks are always on here (they are debug-build-only in Rust release builds). They never fire on valid use.

### M0-executors

- `ForegroundExecutor` has no `foreground_runnables` counter (upstream `#[cfg(feature = "profiler")]`, executor.rs:30, :409-433, platform_scheduler.rs:32, :110): the journal's `ForegroundRunnableCounter` belongs to M3-profiler-journal. See docs/port/handoff/M0-executors.md.
- `ThreadedDispatcher.IsMainThread` identifies the main thread by goroutine id; a task body counts as the creating goroutine only when its iter.Pull coroutine was created there (parsed from `runtime.Stack`'s "created by iter.Pull ... in goroutine N"). A foreground body first polled from another goroutine reports false. `currentThreadGoroutineID` parses a stack trace on every call.
- `Scope`, `Scoped`, `ScopedPriority`, `BlockOn`, `BlockWithTimeout`, `BlockTest` compile on js/wasm too (upstream gates them out with `cfg(not(target_family = "wasm"))`), following scheduler's `LocalExecutor.BlockOn`. Parking there deadlocks the single wasm thread.
- A scoped future canceled before its first poll is released by `Scope.Close` claiming it (CAS on its state) instead of by a dropped future closing its sender clone.
- `ThreadedDispatcher` worker and timer goroutines live for the lifetime of the process, as upstream's threads do; each `NewThreadedDispatcher` in tests adds NumCPU+1 goroutines.
- Foreground task dispatch holds the scheduler weakly, as upstream's `Arc::downgrade` does. Go frees by liveness, not scope, so the last use of a `ForegroundExecutor` or `BackgroundExecutor` over a `PlatformScheduler` can drop its pending tasks before the enclosing block ends; tests keep executors alive with `runtime.KeepAlive`, and the App must hold its executors for its lifetime.

### M0-geometry-units

- `(Pixels).Pow` goes through float64 math.Pow; Rust powf on f32 may differ by 1 ulp.
- JSONSchema for AbsoluteLength, DefiniteLength, Length, Rgba and Hsla is `{"type":"string"}` without upstream's `pattern`: internal/jsonschema.Schema has no Pattern field (handoff written).
- Pixels/ScaledPixels `Sum` impls are not ported (Go loops add natively); `Hash` impls rely on Go `==` map keys (no canonical_float_bits, so +0/-0 hash equal as in upstream, NaN keys differ).
- `IsZero` methods on Point/Size/Bounds/units are also picked up by encoding/json/v2 `omitzero`; Size.IsZero is true when either side is zero, so a bare `omitzero` Size field would be dropped then. Refinements wrap fields in Option, so generated code is unaffected.
- Parse error texts follow Rust's ParseFloatError / ParseIntError wording; anyhow context chains collapse to the outermost message except ParseDefiniteLength's percent branch, which appends the cause with %w.
- Float32 maths in color.go wraps products in float32() before adds to keep the compiler from fusing them into FMA on arm64.

### M0-geometry

- Debug output (`GoString`) and Display (`String`) format numbers with Go `%v`: `5` where Rust Debug prints `5.0` for f32 fields. Pixels gets its own String in M0-geometry-units.
- `SharedString.GoString` approximates Rust `str::escape_debug`: non-printable runes become `\u{..}` via `unicode.IsPrint`; Rust also escapes grapheme extenders at string start, Go does not.
- Operator impls with a foreign right-hand type (`Point<T>: Mul<Rhs>`, `Size<T> * Rhs -> Size<Rhs>`, `Bounds<T> * Rhs -> Bounds<Rhs>`, `Div<S>`, `MulAssign<S>`) take `rhs T` (same unit as the container). Callers convert the scalar (`p.Mul(Pixels(f))`), which is exact for Pixels/ScaledPixels * f32 and DevicePixels * i32.
- Generic comparisons go through `geomPartialCmp`, which type-switches on Pixels (partial_cmp, then total_cmp for NaN) and ScaledPixels (total_cmp); every other T uses IEEE / integer ordering, as Rust's PartialOrd does. Rems has no PartialOrd upstream; Go orders it natively.
- `stubs.go` holds `App.FindDisplay`, `App.PrimaryDisplay`, `DisplayID` and a three-method `PlatformDisplay` (ID, Bounds, VisibleBounds) for BoundsCentered / BoundsMaximized; M0-app-core and M0-platform-types replace them. BoundsCentered / BoundsMaximized are not exercised by tests until then.

### M0-gestures-spring

- spring.go: `springResolveOf` returns `any` (float32 -> float32, bool -> AnimationPhase); the stage that ports the spring animation element (animation.rs consumer) should type its output when it wires `SpringAnimation[T]`.
- spring.go / gestures.go: f32 maths go through float64 `math` calls cast back to float32; results can differ from Rust's f32 intrinsics in the last ulp.

### M0-input-types

- `KeybindingKeystroke` keeps the Windows display fields on every OS and branches on `runtime.GOOS` instead of per-GOOS files (the stage owned only platform_keystroke.go). Behaviour matches upstream per OS; `NewKeybindingKeystroke` is callable on every OS.
- `ActionData.JSONSchema` takes no shared `jsonschema.Generator`, so schemas do not share `$defs` across actions as schemars' `SchemaGenerator` does. Revisit when keymap schema generation is ported.

### M0-keymap

- stubs.go: `FocusHandle.Clone` returns a copy and `FocusHandle.Release` does nothing (placeholder bodies, so tab_stop.go can clone and release where upstream does). M0-window-core replaces both with the counted handle.
- tab_stop_test.go builds `FocusHandle{id: ..., TabStop: ..., TabIndex: ...}` literals instead of upstream's `FocusHandle::new(&focus_map).tab_stop(..).tab_index(..)`, because `newFocusHandle` is a panicking stub. If M0-window-core renames those fields, switch the helper `testFocusHandles.handle` to the real constructor and builders.

### M0-lyon

- internal/lyon/geom: every float product that feeds a sum is wrapped in `float32(...)` so Go does not fuse it into an FMA (Go may fuse on arm64, ppc64, s390x, riscv64 and amd64 v3; Rust never does). New arithmetic in this package must follow the same rule, or arm64 output stops matching upstream. `TestParity_RustGoldenEvents` (internal/lyon/parity_test.go) catches drift; its golden text was produced by lyon_path 1.0.19 / lyon_geom 1.0.20 with rustc 1.99 on x86_64.
- f32 trig goes through float64 (D-LYON-TRIG); a one-ulp difference from libm `sinf` is possible on inputs the golden test does not cover.
- `Path` shares its point and verb slices with the builder that produced it (Rust moves the builder). Reusing a builder after `Build` would alias; `Transformed` copies before mutating.

### M0-parity-rescan

- `scan.areasFromArgs` reads `-area` from `os.Args` because `matrix.Rescanner` has no areas parameter. Fix: add the parameter in matrix/rescan.go and main.go (not owned by this stage), then drop areasFromArgs.

### M0-platform-data

- IMG-GIF-FRAMES: `image/gif` decodes all frames or fails; the image crate skips a bad frame and keeps the rest (`decode_animated_frames`). One corrupt GIF frame fails the whole image in Go (D-IMGDECODE).
- IMG-BMP-SUBSET: BMP files decode through x/image/bmp (1/4/8/24/32-bit uncompressed, BI_BITFIELDS only with default masks); the image crate also reads 16-bit, RLE4/RLE8 and arbitrary bitfields. ICO BMP entries use the in-repo DIB reader with the same subset (D-IMGDECODE).
- IMG-WEBP-LOSSY: lossy WebP pixels come from x/image's VP8 decoder and `color.YCbCrToRGB`, not image-webp's upsampling; values can differ slightly (like L-JPEG).
- IMG-ICO-PNG16: an ICO PNG entry is accepted only when `png.DecodeConfig` reports `color.NRGBAModel` (8-bit RGBA), matching the image crate's `PngNotRgba` check.

### M0-platform-types

- `RefreshIntervalFromHz` saturates at `time.Duration`'s maximum (about 292 years) where Rust's `Duration::from_secs_f64` only panics beyond u64 seconds; refresh rates below ~3.4e-9 Hz are affected.
- `profilerFeatureEnabled` is a const in stubs.go until M0-profiler-core adds the `gpui_profiler` tag files.

### M0-profiler-core

- `Histogram` (port of hdrhistogram 7.6.0 `Histogram<u64>`: new(3) auto-resize, record, len, is_empty, min, max, mean, value_at_quantile, count_between, subtract, clone) lives in profiler.go because the stage owns no other path. Should move to `internal/hdrhistogram` with `docs/port/parity/ext-hdrhistogram.md` and tests from the crate.
- `scheduler.Location` has no column, so `SerializedLocation.Column` is always 0 and `TaskStatistics.String` prints 0 where upstream prints the column.
- `TaskStatistics.String` / `ActionStatistics.String` format durations with Go's `time.Duration.String`, not Rust's `Debug` (`0s` vs `0ns`, otherwise equal for typical values).
- Journal entry points in stubs.go (`beginForegroundTurn`, `endForegroundTurn`, `recordTaskPoll`, `recordFramePending`, `recordWindowClosed`, `powerInterruptedSince`, `frameSampleIsValid`, `recordInput`, `recordAction`, `recordView`, `recordDraw`, `recordPresent`) behave as upstream does with no journal installed (`with_journal` no-op), not as panics, so WindowProfiler tests run before M3.
- `TestProfiler_FramesAreSplitByRefreshIntervalAndClassified` omits `journal::install_test_foreground_journal(64, 4)`; with no visibility or power changes the result is identical.
- `spawnHangMonitor` never returns `HangMonitorErrorSpawn` (a goroutine cannot fail to start); the variant is kept for parity.
- `ThreadTimings.Release` removes the registry entry immediately; upstream removes it when the last `Arc` drops, which can be after a concurrent collector's upgrade.

### M0-scene

- path_winding.go: upstream's `thread_local!` BINNER scratch is a `sync.Pool` of `*pathBinner`; same reuse, but a binner can move between goroutines and the pool may drop it under GC pressure.
- path_winding.go: `listed` is an `iter.Seq` closure per cell; upstream's iterator is allocation-free. Profile once the wgpu renderer draws real paths.
- scene.go: `Primitive` values are interfaces, so every `InsertPrimitive` boxes the primitive (one allocation); upstream stores the enum inline. Revisit with the window paint benches.
- path_builder.go: strokes use xo/okage/path, a port of tiny-skia-path 0.12.0, while upstream locks 0.11.4. Diffed stroker.rs, dash.rs, path.rs, path_builder.rs, path_geometry.rs and scalar.rs between the two: only lifetimes, derives and one unused helper differ, no behaviour.
- path_builder.go: `Transform`'s constructors (`TransformTranslation`, `TransformRotation`, ...) and `Angle` live in internal/lyon/geom, so code outside the module can build a `Transform` only as a composite literal (`gpui.Transform{M11: ...}`) or through `PathBuilder.Translate` / `Scale` / `Rotate`. Upstream users call euclid directly.
- scene.go: `TransformationMatrix.Rotate` uses `math.Cos` / `math.Sin` in float64, rounded to float32; Rust's f32 `cos` / `sin` may differ by one ulp.

### M0-scheduler-core

- S-THREADCHECK: local tasks are not checked for being polled or dropped on their spawning thread (`scheduler/src/executor.rs:780-859` panics). Go has no goroutine identity. Fix only if a cheap goroutine id appears.
- S-DEDICATEDEXIT: a `SpawnDedicatedThread` session goroutine exits when a `runtime.AddCleanup` on its dispatch handle closes the run queue, i.e. after GC finds every executor copy and task of the session unreachable. Upstream exits at the last `Arc` drop. Only thread lifetime depends on GC; tasks never stop through GC.
- Waker reference counts are not tracked. A coroutine parked on a primitive whose other side is abandoned without `Close`/`Send` keeps its goroutine until that primitive is closed; upstream frees the future when the last waker is dropped. Every primitive in this package wakes or drops its stored wakers on `Close`.
- `Chan[T](n)` with n > 0 holds exactly n values; futures' bounded mpsc holds n plus one per sender.
- `Select` is biased (`select_biased!`); futures' `select!` picks a random ready arm.
- A panic in a task body is re-raised by iter.Pull's `next`, so the stack trace printed by the poll loop starts at `next`, not at the original panic site.
- `Location` records file and line, no column.
- `BlockOn`, `BlockWithTimeout` and `Scheduler.Block` also exist on js/wasm (upstream `cfg(not(target_family = "wasm"))`).
- The realtime loop of `BackgroundExecutor.SpawnRealtime` exits when its task finishes or is dropped (upstream: when its sender's last copy drops, which is the same moment).
- `*Task[T]`'s own fields (`kind`, `spawned`, ...) are not synchronized: canceling one handle on one goroutine while another goroutine polls that same handle is a data race. Upstream's uniquely owned `Task` cannot express it; cancel from another goroutine only with a happens-before edge to the handle's owner (the task state behind the handle is mutex-protected).
- A body that recovers the cancel unwind (`IsCancelUnwind` reports true) keeps running inside `stop()` until it returns; every later await in it panics again and its output is discarded. Go cannot forbid recover; Rust drops the future without running it further.
- A panic in a task on a `SpawnDedicatedThread` session or a realtime loop is recovered, printed to stderr with its stack (Rust's default panic hook), and ends that goroutine: queued runnables are closed and later spawns on the session resolve to canceled tasks, as a Rust thread unwinding drops its receiver.
- A pop from a bounded `Chan` wakes every parked sender (each Send future keeps one slot, replaced on re-poll and removed on cancel or completion); futures' mpsc unparks one sender per freed slot.
- `Select` with no cases panics ("scheduler: Select with no cases"); `select_biased!` with no branches does not compile.
- `DedicatedExecutor` spawns share the handle's sender instead of a clone each; the session's lifetime is tied to `Close`, which cancels it, so only the sender count differs.

### M0-scheduler-test

- S-GOID: wrong-thread detection parses goroutine ids from runtime.Stack; a non-owner goroutine is checked once for `created by iter.Pull` and cached per scheduler. A foreign goroutine waking during a poll of a coroutine is not distinguished from the coroutine itself only if it was created by iter.Pull.
- D-RNG: seeds expand through Rust's PCG32 seed_from_u64 stream into math/rand/v2 PCG; draws (random_range, random_bool, random::<bool>) differ from rand's StdRng, so seeds give different interleavings than upstream. Statistical tests (test_block_with_timeout's None/Some mix, priority, interleaving) pass with this seeding.
- Pending traces are Go call stacks captured at TracingWaker clone (frames above Clone), not resolved Rust backtraces.
- test_send_local_executor_worker_submission_and_wakeup cannot assert the polling thread (S-THREADCHECK).

### M0-style

- `StyleFromRefinement`, `TextStyleFromRefinement` (refinegen) fill unset fields with Go zero values, not Rust `Default` (`FontWeight` 0 instead of 400, `FontFamily` "" instead of ".SystemUIFont", `Display` Flex matches). Upstream's `from_refinement` is only reached through `Refineable::from_cascade`; callers that need upstream defaults must refine `DefaultStyle()` / `DefaultTextStyle()`. Fix belongs in refinegen (a `Default<T>()` hook), not owned here.
- `CombineHighlights` folds the active highlights in ascending insertion order. Upstream iterates an FxHashSet<usize>, whose order the upstream test depends on and which is ascending for these small ids; any divergence would only show with many overlapping highlights.
- `HighlightStyle.Hash` writes the fields through fmt, not Rust's `Hash` byte stream; equal values hash equally, which is all callers rely on.
- `cfg(debug_assertions)` items (`DebugBelow`, `Style.Debug`, `Style.DebugBelow`, `Styled.Debug`, `Styled.DebugBelow`) are always compiled (Go has no debug/release split).
- JSON: unit enums use serde variant names; `TextOverflow` uses serde's externally tagged form. `Fill` (sealed interface) has no serde-shaped JSON (`{"Color": ...}`) and the refinements derive no JSON schema yet (upstream `JsonSchema`); nothing in the port serializes styles today.
- The `From<gpui enum> for taffy::style::*` impls of style.rs:1249-1329 are not here: M0-element writes them in taffy.go (bridge-file rule).

### M0-sumtree

- `Arc::make_mut` is ported as unconditional copy-on-write (`SumTree.makeMut` clones the node and its slices on every mutation), since Go cannot see whether a node is shared. Behaviour matches; mutation of unshared nodes costs an extra copy per level. Revisit if list/sum_tree benches show it.
- Keyed operations (`InsertOrReplace`, `Remove`, `Edit`, `Get`) are generic methods taking the key type `K`; `T` implementing `Key() K` is checked at run time (D-COND). `InsertOrReplace` needs `K` spelled out (`tree.InsertOrReplace[K](item, cx)`).
- `SumTreeFromIter` / `Extend` take slices, so upstream's `.fuse()` behaviour in test_from_iter is moot; the test keeps the iterator-restart scenario by collecting first.

### M0-taffy-compute

- internal/taffy/stubs.go holds panicking stubs with final signatures: `ComputeFlexboxLayout` (M0-taffy-flexbox), `ComputeBlockLayout`, `ComputeOofLayoutForArea`, `OofLayoutResult`, `oofBoxLayout`, `layoutOofBox`, `performOofLayout` (M0-taffy-block), `ComputeGridLayout` (M0-taffy-grid-sizing). Any TaffyTree layout of a node with children reaches a stub until those stages land; three taffy_tree tests are skipped and `blocked:M0-taffy-flexbox`.
- `ComputeOofLayout` (oof.rs:348) is already ported in compute_oof.go so leaf layouts (which have no positioning area) run; M0-taffy-block adds the rest of oof.rs to that file.

### M0-taffy-fixtures

- testdata/xml.zip is deflate-compressed with compress/flate BestCompression; byte-identical regeneration holds for one Go toolchain, a Go version that changes flate output changes the archive (regenerate and commit).
- xml_css_test.go is a minimal cssparser-like tokenizer (no escapes, strings, comments, urls); enough for the fixtures, not general CSS.

### M0-taffy-grid-types

- D-TAFFY-SMALLVEC: `TrackIntervals.intervals` is a plain slice; the `spilled()` assertion is checked as `len <= 2`.
- `GridItem` sums (`spanned_track_limit`, `grid_area_size`) start from `+0.0`; Rust's `f32` `Sum` starts from `-0.0`, so an empty span yields `+0.0` instead of `-0.0` (no layout depends on the sign).
- `itemPlacement` alias lives in stubs.go until M0-taffy-grid-placement ports placement.rs.
- Methods that collide with fields: `gridTrack.getFitContentLimit`, `namedLineResolver.getAreaColumnCount` / `getAreaRowCount`.

### M0-taffy-style-grid

- internal/taffy/stubs.go: `GridLine` (with `GridLineFromI16`, `AsI16`), `OriginZeroLine` (`type OriginZeroLine int16`, upstream's `.0` is a conversion, `+ u16` is `+ OriginZeroLine(u)`), `GridCoordinate` and `maxGridTracks` are declared for compute/grid/types/coordinates.rs; `GridLine.intoOriginZeroLine` panics. Owner: M0-taffy-grid-types, which moves them into compute_grid_types_coordinates.go. Until then `GridPlacement.IntoOriginZeroPlacementIgnoringNamed` / `nonNamedGridPlacementIntoOriginZeroPlacement` panic for definite line placements.
- `StyleRef.GridTemplateRows` / `GridTemplateColumns` allocate a `[]GenericGridTemplateComponent` per call (D-TAFFY-GRIDLISTS). If grid layout profiling shows it, cache the ref slice or switch the trait to an indexable view.
- `Option<T>` helper impls (`TaffyZero`, `TaffyAuto`, ... for `Option<T>`) only work for `Option[float32]` (D-TAFFY-HELPERS).

### M0-taffy-style-mod

- internal/taffy/stubs.go: grid style types declared with placeholder shapes so `Style[S]` can name them: `GridTemplateComponent[S]`, `GridTemplateRepetition[S]`, `MinTrackSizingFunction`, `MaxTrackSizingFunction`, `TrackSizingFunction` (alias of `MinMax[...]`), `GridAutoFlow` (+ variants), `GridTemplateAreas[S]`, `GridTemplateArea[S]`, `GridPlacement[S]` (zero value must stay Auto: `StyleDefault` and `OofItemStyleDefaults.GridRow` rely on it). Owner: M0-taffy-style-grid, which replaces them with the real ports.
- `impl GridContainerStyle for Style` / `impl GridItemStyle for Style` (style/mod.rs:1386-1600) are not written: M0-taffy-style-grid adds them as `StyleRef[S]` methods (rows `blocked:M0-taffy-style-grid` in ext-taffy.md).
- `CoreStyleDefaults` etc. are embeddable; embedding two Defaults structs that share a method (AlignSelf, AlignContent, IsTable) makes it ambiguous, so the implementor must define that method itself. The compiler catches it.
- `StyleDefault[S]()` builds a fresh Style per call; trait default methods call it, which costs a struct copy each time. Fine for now; revisit if profiling shows it.

### M0-taffy-style

- internal/taffy/stubs.go: `FlexDirection` with a panicking `IsRow` (owner: M0-taffy-style-mod, style/flex.rs) and the style_helpers trait interfaces `TaffyZero`, `TaffyAuto`, `TaffyMinContent`, `TaffyMaxContent`, `TaffyFitContent`, `FromLength`, `FromPercent`, `FromFr` (owner: M0-taffy-style-grid, style_helpers.rs). Move them out when those files are ported; stubs.go is deleted by M0-taffy-grid-sizing.
- geometry.go arithmetic on `Rect[T]` / `Size[T]` / `Point[T]` / `Line[T]` dispatches on the element type at run time (`add`, `sub`, `asF32Size`, `asF32Rect`; D-COND). It does not allocate, but costs a type switch per operation; revisit if layout profiling shows it.
- `f32Min` / `f32Max` return the second operand for `(0.0, -0.0)` ties; LLVM's `minnum` / `maxnum` leave the sign of a zero result unspecified, so bit-exact parity on signed zeros is not guaranteed.
- FMA: every float32 multiplication must be written `float32(a * b)` (rule in docs/port/parity/ext-taffy.md); this stage does so in ResolvedPercentageSize, ResolveToOption and SizeOptionF32MaybeApplyAspectRatio. Only the arm64 CI runner can show a miss.
- The stub trait interfaces take methods on a value receiver, which `float32` and `Option[T]` cannot have. Upstream implements `TaffyZero` (and other style_helpers traits) for `f32` and `Option<T>`, so M0-taffy-style-grid's generic helpers (`zero::<T>()`, ...) need a separate path for those types (a type switch or dedicated functions).

### M0-taffy-tree

- `internal/taffy/stubs.go`: `DetailedGridInfo[S]` (M0-taffy-grid-sizing) and `BlockContext` (M0-taffy-block) are empty placeholder structs; the owners replace them. `DetailedLayoutInfo` has no `Equal` until `DetailedGridInfo` exists.
- `CacheModeChangeEvictions` is one process-wide atomic counter (upstream: thread-local), so a test asserting it must not run in parallel with other layout tests (D-TAFFY-CLONE).
- `NodeID` <-> slotmap `DefaultKey` conversions are left to M0-taffy-compute (rows `blocked:M0-taffy-compute`).

### M0-test-contexts

- Leak report runs before RunTest drains the scheduler (see handoff); pending tasks holding handles show as leaks.
- `reportAppLeaks` waits for GC cleanups with a sentinel cleanup plus up to 1s per round and stops after two stable rounds (max 32); cleanups may run concurrently, so a slow cleanup could be missed.
- `TestAppContext.drop` runs OnQuit callbacks, which upstream's test macro never calls (only `App::quit`); harmless in Go since IntoMut allocates nothing.
- `Entity.NextNotification` does not end its stream when the entity is released (upstream's sender is dropped with the observer).

### M0-test-platform

- No unit tests of their own: upstream has none for platform/test; coverage comes from M0-test-contexts and later gpui::tests.

### M0-text-system

- FontFeatures / FontFallbacks intern every distinct list in a process-wide sync.Map that never shrinks (upstream shares Arc lists and frees them). Feature and fallback sets are few in practice.
- D-SCHEMAKW: FontFeatures and FontWeight JSON schemas are reduced until internal/jsonschema gains patternProperties / multipleOf / default.
- TestLineLayout_BreakIndicesMatchICU pins go-text segmenter output against ICU's for two inputs only; ICU new_for_non_complex_scripts and go-text's UAX #14 may differ on other scripts.

### M0-window-a11y

- window_a11y_debug.go writes the debug JSON with a hand-rolled ordered writer (serde_json `preserve_order` + `to_string_pretty`), including ryu-style float text (`800.0`, `1e16`); ryu's exact shortest-digit choice for f32 is approximated with strconv's shortest f32 formatting.
- NativeView's window list holds weak pointers; a released view is detected through `nativeViewState.dropped` because weak pointers only clear after GC.
- window_native_view_test.go's input handler holds a WeakEntity of the test view instead of upstream's strong Entity (a strong handle inside a per-frame input handler has no release point under strict leak detection).

### M0-window-core

- Element arena (D-ARENA): each App's `elementArena` (scope depth plus the `deferArenaDrop` queue) lives in a package-level map keyed by `App.this` (`weak.Pointer[AppCell]`), cleaned up with `runtime.AddCleanup` on the AppCell, because app.go has no `element_arena` field. Move it onto App when app.go is next edited. `currentElementArena` is package-global (upstream's thread-local); root tests must stay non-parallel.
- `elementArenaScope.End` only logs when the scope ends without `exit`; upstream also debug-asserts when not unwinding, which Go cannot detect without recover.
- `FocusHandle` literals without a focus map (tab_stop_test.go builds them) are uncounted: `Clone` copies, `Release` does nothing.
- `ElementID` NamedChild parents are interned in a process-wide map that never shrinks, so value equality works with `==`. `Path` ids compare by raw string (upstream compares path components); the hash follows std's component-wise `Path::hash` for `/` separators only.
- `newWindow` does not close the platform window when `NewWindowProfiler` fails after `OpenWindow` succeeded (upstream drops the Box, which closes it).
- `deferArenaDrop` calls made outside any draw scope go to `fallbackElementArena`, which nothing clears (upstream's thread-local fallback arena is never cleared either, but there it only holds memory). `VisualTestContext.Draw` currently runs without a scope, so its view drops land there.

### M0-window-dispatch

- window_smoke_test.go releases the window root by hand because window removal does not release it (see handoff).

### M0-window-paint

- Element states that own handles other than `InteractiveElementState` still have no `dropElementState`; see M0-EXIT-OPEN.
- `IsWindowHovered` picks the hovered flag with `runtime.GOOS` (windows, linux, freebsd) as upstream's `cfg!`; js/wasm and darwin use `IsWindowActive`.
- `profiling::finish_frame!()` in `present` is not ported (no puffin).
- The four `feat:profiler` debug-overlay methods call M3-profiler-hang placeholders: `DebugFrameOverlayMode` returns 0 and `CycleDebugFrameOverlayMode` panics until then.

### M0-window-tests

- window_test.go: `windowTestCanvas` stands in for `canvas` (M0-elements-basic is later in the chain); swap to `Canvas` when it lands.
- window_test.go: `LongPressListener` captures a WeakEntity and upgrades per event instead of holding `cx.Entity()` in the paint closure: Go closures cannot release a captured strong handle when the frame drops them.
- window_test.go: `test_window_reports_no_raw_handle_instead_of_panicking` checks `RawWindowHandle().Kind == RawWindowHandleNone` (the Go shape of raw_window_handle's NotSupported).

### M1-linux-common

- `openCommands` omits open-5.4.5's WSL PowerShell launcher (D-LINUX-OPEN).
- `runOpenCommands` (and the portal calls in `OpenURIInternal` / `RevealPathInternal` / `InhibitIdleSleep`) block the background worker goroutine on `exec.Cmd.Wait` and on the portal `Response` signal, where upstream awaits them on smol; a portal that never answers keeps one worker busy (ashpd also waits without a timeout).
- `LinuxDispatcher.IsMainThread` parses `runtime.Stack` (copied from the root's unexported `goroutineID` / coroutine-creator helpers); costly if called on hot paths.
- `SpawnRealtime` does not raise the thread to SCHED_FIFO 65 (D-LINUX-RTPRIO).
- `newTextSystem` returns `gpui.NoopTextSystem` until M2-exit wires text/gotext.

### M1-linux-dbus

- Secret Service: oo7's sandboxed path (org.freedesktop.portal.Secret + file keyring) is not ported; credentials always go through org.freedesktop.secrets. Flatpak builds without host keyring access will fail where oo7 would use the file backend.
- Secret Service: no 30s method timeout (oo7 sets one on its own connection); calls use the caller's conn defaults.
- XDG portal: ColorScheme values other than 1/2 map to Light (ashpd may reject unknown values and skip the event).

### M1-linux-platform

- platform/linux/platform.go `fileChooserRequest` duplicates the portal request/Response plumbing of internal/common's unexported `portalCall` (which drops the Response results the FileChooser needs). Export a results-returning variant from common and use it here.
- `fileChooserRequest` waits for the portal Response without a timeout, as ashpd does; the dialog is interactive.

### M1-linux-screencap

- `internal/linux/pw`: `Loop.Iterate` calls `pw_loop_enter`, `pw_loop_iterate`, `pw_loop_leave` as three purego calls from a goroutine that is not locked to an OS thread (only package platform may call `runtime.LockOSThread`). The goroutine can migrate between the calls, so the loop's recorded owner thread may differ from the iterating thread. The calls work today; if PipeWire starts checking the thread on iterate, move the three calls into one pinned thread.
- `internal/linux/pw`: when libpipewire does not export `pw_loop_enter/leave/iterate` (releases before 1.2), the binding calls `spa_loop_control_methods` through the vtable at offsets that only `pw_test.go` checks.
- `screencap.decodeCompoundText` duplicates a small part of `platform/linux/x11/internal/xim/ctext.go`, which the screencap package cannot import (internal visibility). See handoff.
- Wayland: the portal `Response` signal match is added after the method call returns, as upstream; a portal that answers before the match is installed loses the response ("Did not get response").
- `Capturer.Close` releases the Wayland D-Bus connection; nothing calls it until the root glue (M3-root-capture) does.
- X11 on Xwayland (this machine, :0): core `GetImage` of a full RandR display rect on the root window fails with BadMatch, while MIT-SHM `GetImage` succeeds. Upstream (core only) would fail there too; the core fallback is only reached when SHM is unavailable.
- The gated X11 test passes an explicit RandR display: Xwayland reports no primary output, so the default-target path (`GetOutputPrimary`) fails with BadOutput, as upstream would.

### M1-wayland-client

- `WaylandClientState` is a plain pointer behind `clientCell`; upstream's `RefCell` borrow panics on re-entrant mutable borrows are not reproduced. `WaylandClientStatePtr.GetClient` / `TryGetClient` model only the `Weak` upgrade (nil once `WaylandConnection.Close` ran).
- D-WLFOREIGN: xdg-foreign exports live until the window drops instead of until the identifier drops; xdg-foreign is hand-defined in platform/linux/wayland/xdg_foreign.go (no wlgen output). `waylandExportHandle` has no automated test: exporting needs a toplevel surface, i.e. a window.
- Map iteration order (`Displays`, window ticks) is Go-random where upstream iterates a HashMap in its own unspecified order.
- `Cursor` loads themes through libwayland-cursor; the no-name load reads XCURSOR_THEME / XCURSOR_SIZE itself as wayland-cursor's `CursorTheme::load` does. libwayland-cursor loads every cursor of the theme up front, wayland-cursor loads lazily.
- `Attach` panics until M1-wayland-input and M1-wayland-window replace the package stubs.go (seat / data-device listeners, `NewClipboard`).

### M1-wayland-ffi

- Protocol XML in internal/linux/wlproto/xml comes from the Rust crates upstream compiles against (wayland-client-0.31.15/wayland.xml, wayland-protocols-0.32.13, -wlr-0.3.12, -plasma-0.3.12), not /usr/share: the system copies (wayland 1.26, wayland-protocols 1.50) differ only in description typos. tablet-v2.xml is vendored because cursor-shape-v1 references zwp_tablet_tool_v2 (the crate builds cursor_shape against wp::tablet::zv2); ext-background-effect-v1.xml because client.rs:44 binds it.
- wlspike does not lock its OS thread (paritycheck -lint allows LockOSThread only in package platform); libwayland and wgpu tolerate the main goroutine moving between threads.
- wl reads errno through libc's __errno_location; Display brackets each failing-capable libwayland call and the errno read with a scoped runtime.LockOSThread/UnlockOSThread (nests with platform's init lock). Listener code that runs inside wl_display_dispatch can still overwrite errno before libwayland returns.
- The dispatcher's pending-panic slot is a package global, not per display or thread; libwayland dispatch must stay on one thread at a time.
- Event object arguments that are not managed by wl (proxies created by Mesa/wgpu or libwayland-cursor) decode as nil instead of wayland-backend's unmanaged ObjectId.
- Requests check `since` against the proxy version and panic locally (wayland-backend does not; the compositor would answer invalid_method). Proxies with version 0 (legacy, unversioned) skip the check.
- A listener panic does not abort as upstream's extern "C" unwind does: the first panic is kept, the rest of the libwayland dispatch batch is still delivered to listeners (so new_id objects, fds and state are not dropped), and the panic is re-raised when the Display call returns. FDs the panicking handler had received are not closed: once the handler ran it owns them, and closing after a panic could close a reused descriptor.
- libwayland ignores the dispatcher's return value, so a protocol violation (unknown opcode, object argument of the wrong interface) is made fatal on the Go side: the Display stores an ErrProtocolViolation error, drops (and releases the fds/new_id proxies of) every later event, and every Dispatch*/Roundtrip*/Flush/ReadEvents/Err returns it. The socket itself stays open until Disconnect.
- Unhandled new_id objects (no listener for the creating event) are released with wl.DiscardNewID: the first argument-less destructor request the bound version has (e.g. wl_data_offer.destroy), else a local wl_proxy_destroy.

### M1-wayland-window

- Windows use `wgpu.NewWgpuRenderer` (the concrete `*WgpuRenderer`), not `wgpu.NewRenderer`: upstream calls `max_texture_size` and `needs_redraw`, which `render.Renderer` lacks.
- `RetryTimerFired(time.Time)`: scheduler has no `time.Time` -> `Instant` constructor, so the frame signal is `InstantNow().Add(-time.Since(deadline))` (only evaluated when profiling is on).
- `WaylandWindow.Display()` looks the `*wlproto.WlOutput` up in the client's `wlOutputs` (ObjectID is a `*wl.Proxy` and cannot be converted back); it returns nil once the connection is closed, where upstream still returns the cached output.
- `applyExclusiveEdge`'s warning formats the anchors by hand (`Anchor(TOP | LEFT)`), approximating bitflags' Debug output.
- The clipboard UTF-8 error log says "invalid utf-8 sequence" without Rust's byte index.
- As upstream, wp_fractional_scale_v1 objects and a pending frame wl_callback are never destroyed when a window closes.

### M1-wgpu-context

- D-WGPUCONFIGURE: wgpu-native v29 aborts on surface configure errors. The facade pre-validates configurations in Go (wgpu-core's checks) and reports failures into the error scope, so adapter selection falls through; the usage check, SURFACE_VIEW_FORMATS downlevel flag, colour space and hal-level configure / GPU-wait failures are not pre-checkable and still abort. Revisit when the binding is bumped (TECH_DEBT L-WGPUVER).
- The binding has no uncaptured-error callback API (wgpuDeviceSetUncapturedErrorCallback is in its TODO). The facade emulates wgpu's error sink in Go: the binding's `Try*` errors (Validation scope only) go to the device's Go scope stack or handler. Out-of-memory and internal errors from calls the binding wraps only in a Validation scope, and every queue-submit error, still abort the process inside wgpu-native (as do surface configure errors, D-WGPUCONFIGURE).
- D-WGPUCAPS: no adapter format-features / downlevel-capabilities query in the C API; answered from the WebGPU guaranteed-usage table and, on GL, `MaxStorageBuffersPerShaderStage > 0`. A GL driver without vertex-stage storage buffers but with fragment ones would be misjudged.
- Adapter info mapping (wgpu-native v29): `Name` = WGPUAdapterInfo.device, `Driver` = .vendor (Vulkan driver name, e.g. "NVIDIA"), `DriverInfo` = .description. wgpu-native reports both wgpu `Other` and `VirtualGpu` as `Unknown`, mapped to `Other` (sort priority 2).
- Device loss: wgpu-native's `CallbackCancelled` (device released) is mapped to `Destroyed`, so releasing a device never marks it lost.
- `WgpuContextInstance` ignores its display argument (wgpu-native takes the display from each surface descriptor). wgpu's `InstanceFlags` / `BackendOptions` / `MemoryBudgetThresholds` have no binding equivalent; wgpu-native defaults apply.
- GPU objects are released explicitly where upstream drops them in the atlas (removed / cleared textures and views); adapters not selected by context creation (skipped, failed or unused) and the device / queue of an adapter that fails the surface test are released explicitly; the selected WgpuContext's device, queue and adapter (Arc-shared upstream) are left to the binding's GC finalizers.
- The four `*_is_valid_wgsl` tests validate by compiling on a device (upstream uses naga without a GPU), so they skip on machines without an adapter.
- D-WGPUATLASBOUNDS: atlas Insert rejects short tile data and oversized tiles up front and drops a pushed texture whose first allocation fails (upstream would leak it).

### M1-wgpu-renderer

- Headless readback waits with `Device.Poll(true)`, which has no timeout; upstream polls with a 30 s timeout and then waits 30 s on the map callback. A hung GPU blocks the caller instead of returning an error.
- GPU objects are released explicitly where upstream drops them (per-frame bind groups, command buffers and encoders after submit; pipeline layouts and shader modules after pipeline creation; replaced pipelines, depth textures, instance buffers/textures, atlas bind groups and headless targets). The device and queue belong to the WgpuContext and are never released by the renderer.
- `WgpuRenderer.Destroy` releases the surface and core; the atlas stays alive (shared with the window's sprite atlas, as upstream's `Arc<WgpuAtlas>`).
- `Recover` sleeps 350 ms on the calling (platform) thread as upstream's `thread::sleep`.
- `NewWgpuRendererFromCanvas` (wasm) is left to M4-web-wgpu-js; `NewWgpuRendererFromSurface` takes an `internal/gpu` Surface, so only render/wgpu code can call it.

### M1-x11-client

- Event pump: xgb owns the socket, so a goroutine drains `WaitForEvent` into a mutex queue and pings the loop (`common.MakePing`). Upstream polls the fd directly. X errors arrive as `xErrorEvent` in the same queue.
- Second libxcb connection (`xcbshim.Connect`) exists only for xkbcommon-x11 keymap/state and is handed to the window stage; two X connections per client.
- `internal/xrm` and `internal/xcursor` are local ports of x11rb's resource_manager/cursor and the xcursor crate (xgb has neither).
- `xcbFlush` is a no-op: xgb writes requests immediately.
- `NewClipboard` stub returns an empty clipboard instead of panicking so attach works before M1-x11-window; its methods panic.
- `X11Connection.Close` closes the X connection (upstream relies on Rc drop).
- Legacy RandR (<1.5) path iterates CRTCs in server order; upstream matches.
- `ScreenCaptureSources` calls a panicking local `scapScreenSources` until the scap back end lands.
- `getScaleFactor` approximates Rust `f32::from_str` with `strconv.ParseFloat` plus a reject list for hex/underscore forms.

### M1-x11-window

- platform/linux/x11/window.go: X11WindowState has no RefCell borrow guard; upstream's re-entrant `borrow_mut` panics are not reproduced (callbacks are still taken out and restored around calls, as upstream).
- platform/linux/x11/window.go: `xcbFlush` is a no-op; xgb writes each request as it is issued. The round trip before surface creation is an explicit GetInputFocus reply.
- platform/linux/x11/clipboard.go: selection condvars are re-expressed as close-and-replace channels; the reader connection pumps events on a goroutine instead of polling with a 1 ms sleep (same timeouts).
- platform/linux/x11/a11y.go: `newA11yAdapter` returns nil until M3-exit wires a11y/atspi.
- No live-display test for X11Window or Clipboard (a window test would open a window, a clipboard test would touch the user's selection).
- window.go: the round trip runs after CreateWindow and the property/XInput setup, before the surface; mapping stays in MapWindow as upstream (decision 15 says "created and mapped", but mapping early would break Show=false and the WM-hint ordering).

### M2-gotext-layout

- Bidi levels come from go-text `bidi.Paragraph`, whose `Runs` split on direction parity only, so nested levels of equal parity (for example 2 and 4 from explicit embeddings) collapse into one span before `reorder`. The paragraph level is read from a trailing space appended before segmentation (L1 resets it to the paragraph level). Fix: expose per-rune levels (port unicode-bidi's level resolution) if embeddings matter.
- cosmic-text's `Family::Monospace` fallback branch (monospace fallback buffer, per-script monospace ids) is not ported: gpui only sends `Family::Name`.
- `FontMatchKey.variable_weight_match` is always false and `get_font` ignores the requested weight (no `wght` variation applied): go-text keeps `fvar` unexported.
- cosmic-text's `shape-run-cache` is not ported; one reused harfbuzz buffer carries the shape-plan cache. BenchmarkLayoutLine_* runs at about 8.5 ms per 3.8k-char line on an i9-13900H, mostly go-text GSUB application (Lilex `calt`); upstream numbers not measured.
- The harfbuzz buffer language is cleared after `GuessSegmentProperties` to match harfrust's unset language; go-text would otherwise take it from the process locale.
- System fonts are found with `fontscan.DefaultFontDirectories` plus a recursive walk (ttf/ttc/otf/otc), not fontdb's fontconfig parser; face metadata is read eagerly, font bytes lazily on first use.
- `systemLocale` reads LC_ALL / LC_CTYPE / LANG like sys-locale on Unix; other platforms fall back to en-US.
- Glyph rasterisation (`GlyphRasterBounds`, `RasterizeGlyph`) returns an error until M2-gotext-raster.
- Test fonts (IBM Plex Sans Regular, Lilex Regular/Bold, OFL) are copies of zed@6501df1 `assets/fonts` in `text/gotext/testdata/fonts`.
- `GOOS=ios CGO_ENABLED=0 go vet ./text/gotext` fails with "requires external (cgo) linking", the same as for every package (e.g. ./gpuiutil); not checked for ios.
- Benchmarks use `testing.B.Loop` directly: `gpui.RunBench` does not exist yet (only mentioned in seeds.go).

### M2-gotext-raster

- No TrueType/CFF hinting: swash renders with `.hint(true)` (skrifa hinting instance); go-text has no hinter, so outlines are drawn unhinted (L-GLYPH). Glyph pixels and occasionally raster bounds differ by up to a pixel from upstream.
- `Source::Bitmap(StrikeWith::ExactSize)` (non-emoji bitmap strikes) and the colour sources are skipped in `swashRender.renderInto`; M2-gotext-color owns them (text/gotext/scale.go).
- Outline extraction is go-text's `Face.GlyphDataOutline` (implicit TrueType on-curve points become explicit quad endpoints) instead of skrifa's pen; coverage matches only where both emit the same curves (L-GLYPH).

### M3-a11y-atspi-bus

- `AddAccessible` / `RemoveAccessible` cache signals are sent flattened (`(so)(so)(so)iiassusau` / `so`) because zbus flattens a struct body; at-spi2-core's Cache.xml declares one struct argument. Matches upstream on the wire; revisit if accesskit fixes it upstream.
- `Introspect` XML is generated by `object_server.go`; equivalent to zbus's output but not byte-identical (D-ATSPIBUSSERVER).
- Upstream quirk kept: when `IsEnabled` changes true -> true, a new `Bus` replaces the old one but already-active adapters do not re-register their node interfaces on it.
- `TestBus_AdapterOnPrivateBus` polls with short `time.Sleep` while waiting for the registry Embed and the first tree registration; replace with signal-driven waits if it ever flakes.
- The session connection uses godbus `SessionBusPrivateNoAutoStartup` (never runs `dbus-launch`, like zbus's `Builder::session()`); when `DBUS_SESSION_BUS_ADDRESS` is unset, godbus's `$XDG_RUNTIME_DIR/bus` discovery writes the found address back into the process environment, which zbus does not do.

### M3-a11y-atspi-node

- a11y/atspi/stubs.go holds real (non-panicking) context.rs code (`Context`, `AppContext`, lock helpers) so PlatformNode is testable; M3-a11y-atspi-adapter moves it to context.go.
- Nothing sets `Context.released` yet, so a PlatformNode reports Defunct only after GC until the adapter stage sets it on adapter drop (D-ATSPIWEAK).
- `Context` is exported though upstream is `pub(crate)`; the adapter stage decides the final visibility.

### M3-a11y-consumer-tree

- `a11y/consumer/stubs.go` carries working ports of the node.rs items tree.rs uses (NodeID, nodeState, parentAndIndex, Node.Data/IsFocused/IsGraft/ParentID/Parent/ChildIDs/Children/ID/Role/IsDialog/Label/ActiveDescendant). M3-a11y-consumer-node moves them to node.go, settles the iterator shape of ChildIDs/Children (slices here; upstream is DoubleEnded+ExactSize, likely iter.Seq plus reverse/len helpers), replaces `labelFromLabelledBy`'s panic with the labelled_by port, and updates tree_test.go if the shape changes.
- `testTreeFilter` (lib.rs tests::test_tree_filter) needs FilterResult and Node.IsHidden; row is blocked:M3-a11y-consumer-node.

### M4-apple-ffi

- Profiler thread identity: the GCD trampoline maps the OS thread (pthread_threadid_np) to a gpui.ThreadID through a package-level sync.Map, naming the main thread "main" and leaving GCD worker threads unnamed (Rust's thread-local timings per std::thread). Entries are never removed because GCD owns its worker threads' lifetimes.
- Runnables and closures cross dispatch_async_f / dispatch_after_f / pthread_create as integer keys into gcd.Handles tables, not as raw pointers, so C never holds a Go pointer. A runnable whose queue never drains (the main queue in a test binary with no run loop) stays in the table.
- SpawnRealtime creates a real detached pthread through a purego callback (paritycheck -lint bans runtime.LockOSThread outside package platform, so a locked goroutine is not an option). A pthread_create failure panics with "failed to spawn thread: ..." like std::thread::spawn.
- MainThreadMarker checks +[NSThread isMainThread] on every call; Go has no thread-local cache. It is only meaningful from the main goroutine once package platform has locked it to the main thread, or from a native thread.
- The Security framework, CoreFoundation, Foundation, QuartzCore and libSystem are dlopened lazily; Keychain calls return KeychainErrorInvalidResponse{"Security framework unavailable: ..."} if Security cannot be opened (upstream links it, so this path does not exist there).
- The display-link delegate keeps its callback in a Go map keyed by the delegate object instead of an ivar; MetalFrameSource.Close removes the entry before releasing the delegate. A source dropped without Close leaks its entry.
- TestProcessInfo_NotificationsAreDeferredAndCancelledOnDrop skips until M0-test-contexts lands; its body (checkNotificationLifecycle) is written against ForegroundExecutor / BackgroundExecutor and a run-until-parked func and needs wiring to TestAppContext then.
- None of these packages has run on a Mac yet: locally only darwin/arm64 and darwin/amd64 vet + go test -c pass. The first CI run (M4-ci-macos) is the first execution of every test here, including the Keychain round trip (unique *.invalid server name, deleted in cleanup).
- GOOS=ios is covered by the darwin constraint but was not vetted locally (ios/arm64 needs external linking).

### M4-apple-metal-atlas

- metalhdr feeds cbindgen one concatenated file (the six build.rs sources with `#![...]` / `//!` lines stripped) instead of Builder::with_src per file; output is believed identical, not diffed against a real build.rs run on macOS.
- platform/apple is `darwin`-only, so `go generate ./...` on Linux skips the metalhdr directive; regenerate with `GOOS=darwin go generate ./platform/apple` (cbindgen 0.28.0 from PATH or ~/.cargo/bin). TestMetalhdr_GeneratedFileIsCurrent catches staleness on Linux.
- mtl wrappers are raw objc.ID value types with explicit mtl.Retain / mtl.Release, not objc.Retained handles; ownership (+1 for new*/Create/Copy, +0 autoreleased otherwise) is documented per method.
- internal/apple/mtl, metal_atlas and shader tests only run on the macOS CI runner; locally only cross vet / test -c.
- GOOS=ios CGO_ENABLED=0 go vet fails before analysis with "ios/arm64 requires external (cgo) linking" (toolchain, not this code); the STAGES ios Verify form hits the same wall.

### M4-apple-text

- font-kit (zed-font-kit 0.14.1) has no Go equivalent: the slice gpui_apple uses (CoreText loader Font, MemSource, CoreTextSource.select_family_by_name, matching::find_best_match, weight/stretch mapping, OTC and data-fork unpacking) is ported as unexported helpers in platform/apple/font_kit.go, with font-kit's own core_text weight/stretch tests plus a find_best_match test.
- AddFonts takes [][]byte, so upstream's Cow::Borrowed path (CGDataProvider over a static slice, CGFont, CTFont) does not exist; every font goes through Handle::from_memory (CTFontManagerCreateFontDescriptorFromData), as Cow::Owned does upstream.
- CTRun glyphs, positions and string indices are always copied with CTRunGet{Glyphs,Positions,StringIndices}; the *Ptr fast paths are not bound (same values).
- The CoreFoundation types package cf lacks (CFNumber, CFMutableArray, CFMutableAttributedString, CFLocale, CFPreferences, CFURL path) are bound in internal/apple/ct/cfx.go; see handoff.
- internal/apple/cg binds only bitmap contexts, colour spaces and CGFont retain/release; CGPath is not bound because no ported text-system code uses it (font-kit's outline() is not used by gpui_apple).
- None of this has run on a Mac: locally only darwin/arm64 and darwin/amd64 vet + go test -c pass; M4-ci-macos is the first execution of every test here (including Lilex-based ligature test and glyph ids 68/69 for Helvetica a/b).

### M4-macos-appkit

- events.go, keyboard.go and internal/apple/appkit were only cross-compiled (darwin arm64/amd64 vet + test -c); no test has run on a Mac. First macOS CI run must exercise TestEvents_*, TestKeyboard_*, TestGpuiMacos_* and TestAppkit_*.
- appkit wrapper selectors and return types (bool for BOOL, float32 for `pressure`, uint16 for `keyCode`) are unverified at runtime until macOS CI.

### M4-macos-platform

- platform/macos/stubs.go: panicking stubs for MacWindow, MacWindowOpen, MacWindowActiveWindow, MacWindowOrderedWindows, setActiveWindowCursorStyle, nativeCursor and rendererContext (upstream renderer::Context, Arc<Mutex<InstanceBufferPool>>). M4-macos-window replaces them with the window.rs port and the platform/apple metal renderer context, then deletes stubs.go. Until then OpenWindow, ActiveWindow, WindowStack, SetCursorStyle and DebugState panic.
- invokeForeignBlock (system_notifications.go) calls Objective-C-created completion blocks through the Block_layout invoke pointer at offset 16 with purego.SyscallN, because purego's Block.Invoke only knows Go-created blocks. Move to internal/apple/objc if another package needs it.
- Menu items are built with alloc/init (+1) and released after being added to their parent; separatorItem is retained to match. The dock menu is held in MacPlatformState.dockMenu and released when replaced.
- ScreenCaptureFrame.Frame is a borrowed CVImageBufferRef (D-MACOS-SCKFRAME).

### M4-macos-window-objc

- platform/macos/stubs.go now also holds panicking stubs for native_views.rs symbols window.rs calls: nativeViewsAddWindowMethods, nativeViewsAddViewMethods (class registration), checkPointerAfterFrame, nativeViewSetsCursor, isRoutingKeyEquivalent, gpuiViewIsFocused. Class registration (and so MacWindowOpen) panics until M4-macos-native-views replaces them.
- Super calls go through objc.SendSuper (superclass of the receiver's class), so GPUIPanel's dealloc/close/tab overrides reach NSPanel's implementation before NSWindow's; upstream names NSWindow explicitly. Equivalent since NSPanel does not override these.
- attributedSubstringForProposedRange: writes actualRange only when AppKit passes a non-nil pointer (upstream writes unconditionally).
- a11yAdapter gained UpdateViewFocusState(isFocused) (accesskit_macos SubclassingAdapter::update_view_focus_state); M4-macos-a11y must implement it.
- Pre-existing, not from this stage: `paritycheck -area macos -lint` reports platform/macos/system_notifications.go:111 (runtime.LockOSThread outside package platform) and platform/ios/gpui_ios_impl.go (build constraint).

### M4-macos-window

- platform/macos/stubs.go survives this stage (contrary to M4-macos-platform's debt note): it now holds panicking stubs for the second half of window.rs (class accessors windowClass/panelClass/viewClass/blurredViewClass/windowStateArchiverDelegateClass/windowStateUnarchiverClass, isGpuiWindow, getWindowState, addMouseTrackingArea, nativeCursor, visibility, isMacosVersionAtLeast, nsErrorDescription, stepCallback, metalFrame, displayIDForScreen) and for window/native_views.rs (nativeViews type, nativeViews* funcs). Until M4-macos-window-objc and M4-macos-native-views land, MacWindowOpen and most window methods panic.
- Window state "ivar": a package map objc.ID -> *lockedWindowState (storeWindowState / loadWindowState / clearWindowState in window.go), same pattern as D-MACOS-PLATFORMIVAR, because Go pointers cannot live in ObjC ivars.
- gpui.WindowKindLayerShell (Linux-only upstream, present on every GOOS in Go) opens as a Normal window.
- MergeAllWindows / MoveTabToNewWindow use gcd.MainQueue().Async(closure) for dispatch_async_f.
- Standard window buttons and titlebar container views are used as +0 references for the duration of a call instead of objc2's Retained.
- platform/macos/a11y.go is a nil seam (newA11yAdapter returns nil) until M4-macos-a11y.
- paritycheck -lint flags platform/macos/system_notifications.go:111 (runtime.LockOSThread outside package platform), written by M4-macos-platform; not changed here.

### M4-web-input

- go_js_wasm_exec aborts when the environment is large (Go wasm env/argv limit); CI and verify.sh need a trimmed environment on heavy developer shells.
- Event handlers are not exercised by tests (no DOM under node); covered by the web example owner check.

### M4-web-platform

- platform/web/stubs.go holds panicking stubs for events.rs / ime_mirror.rs / text_system.rs / canvas_fallback.rs symbols (registerEventListeners, showVirtualKeyboard, setVirtualKeyboardEnabled, focusImeMirror, isMacPlatform, platformInfo, imeMirror methods, newWebTextSystem). NewWebPlatform panics until M4-web-input and M4-web-text replace them. EventListenerHandle, WebEventListeners, clickState, touchIDs and CanvasFontFallback are real and should move to events.go / canvas_fallback.go.
- FetchHTTPClient: no RedirectPolicy (WEB-REDIRECT); request bodies are read with a plain io.Copy inside the send task (fine for in-memory bodies, blocks the page for a streaming body); response Body.Read must run on a goroutine, never in a task or browser callback (Go wasm deadlocks).
- WebPlatform.Run returns immediately like upstream; a Go wasm main must then block (select {}) or the module exits.
- InitLogging always uses Info (no debug_assertions in Go); console_error_panic_hook has no Go counterpart (Go wasm panics already print to the console). Log target is the caller's Go package path.
- WebDispatcher.IsMainThread is always true (L-WEBTHREADS).
- Untested in a browser: verify only compiles for js/wasm. M4-web-example's screenshots are the first runtime check.

### M4-web-webgl-shaders

- naga-cli has no `binding_map` option; ES 3.00 has no `layout(binding)` anyway, so wgpu's GL backend binds by name. glslgen reflects block/sampler names from the GLSL text with regexes and pairs textures with samplers from WGSL `textureSample*` calls. If naga's naming changes, the regexes in internal/cmd/glslgen/main.go must follow.
- Subpixel (dual_source_blending) variant is not generated: WebGL2 has no dual-source blending, so upstream never builds it on the GL path.
- Installed naga-cli reports 30.0.0 (wgpu 30 line).
- A broken mise `naga` shim sits on PATH; glslgen tries $NAGA, ~/.cargo/bin/naga, then PATH, keeping the first that runs `--version`.

### M4-web-webgl

- render/wgpu/internal/gpu/webgl2/jsgl.go: `getUniformLocation` stores every WebGLUniformLocation in the handle table and never removes it. Each pipeline creation adds a few entries. That is harmless while pipelines are created once per renderer, but the table grows if pipelines are rebuilt many times, for example on repeated device recovery.

### M4-web-wgpu-js

- WebGL2 branch of `NewWgpuContextWeb` returns `errWebGlUnavailable` until M4-web-webgl (D-WEBGLPENDING).
- oliverbestmann/webgpu js `Adapter.GetInfo` is empty: adapter name, vendor and device are blank in logs and `CheckCompatibleWithSurface` messages on the web.
- If the awaiting task unwinds before the promises resolve, the helper goroutine releases the graphics it created; promises themselves cannot be aborted.

### M4-windows-a11y-node

- a11y/windows builds only for windows/amd64 and windows/arm64: the float-argument trampolines (D-WINFLOAT, node_{amd64,arm64}.{go,s}) have no 386 version.
- The trampolines' register moves follow the x64 and ARM64 Windows ABIs (doubles in XMM1/XMM2 resp. D0/D1, `UiaPoint` as an HFA on arm64) and are exercised only by real UIA clients on CI/hardware; TestNode_PlatformNodeVtablesBuild only checks the vtables build.
- `IValueProvider::SetValue` decodes with windows.UTF16PtrToString, which replaces unpaired surrogates instead of panicking like upstream's `to_string().unwrap()`.
- Rows for text.rs, adapter.rs, subclass.rs, lib.rs and tests/ are `blocked:M4-windows-a11y`; stubs.go holds their placeholder types and a panicking newPlatformRange.

### M4-windows-com

- Nothing here has run on Windows yet: locally only windows/amd64 and windows/arm64 vet + `go test -c` pass. The first CI run (M4-ci-windows) is the first execution of every test in internal/win/com, internal/win/win32 and platform/windows, including the struct size and offset pins.
- Native callbacks (thread pool work and timer, monitor enumeration, the realtime thread start, the IUnknown thunks of Go COM objects) are windows.NewCallback thunks made once per process behind sync.Once; the runtime caps thunks at about 2000 and never frees them, so later stages must also create theirs once (com.LazyVtable).
- Runnables, realtime closures and monitor lists cross native context parameters as integer keys into win32.Handles tables, never as Go pointers. A runnable whose TrySubmitThreadpoolCallback or CreateThreadpoolTimer fails is taken back out of the table and run on a goroutine (D-WINDISPFAIL), unlike upstream, which leaks it.
- SpawnRealtime starts a native thread with CreateThread and a callback thunk (runtime.LockOSThread is reserved for package platform), raises it to THREAD_PRIORITY_TIME_CRITICAL and runs f inside the callback; the thread handle is closed at once, as dropping a std JoinHandle detaches.
- Profiler thread identity: GetCurrentThreadId maps to a gpui.ThreadID in a package-level sync.Map, the dispatcher's creating thread named "main", thread pool threads unnamed. Entries are never removed; a recycled OS thread id reuses its entry.
- DWM_TIMING_INFO and UNSIGNED_RATIO are pack(1) in dwmapi.h. Every field happens to sit at a 4-byte offset, so the Go struct uses 4-byte aligned PackedU64 pairs for the 64-bit fields (size 292, pinned by TestTypes_StructSizes). TASKDIALOGCONFIG is also pack(1) and is not declared: TaskDialogIndirect takes an unsafe.Pointer the caller lays out.
- Go COM objects: interface pointers point to CoTaskMemAlloc'd slots holding vtable pointers; the Go object is found through a sync.Map keyed by slot address and kept alive by it until the reference count reaches zero. Vtables are never freed.
- The HiLoWord trait (util.rs) is four generic functions over ~uintptr (hiword, loword, signedHiword, signedLoword) because Go cannot add methods to win32.WPARAM / LPARAM from this package.
- system_appearance activates Windows.UI.ViewManagement.UISettings through RoActivateInstance and calls IUISettings3::GetColorValue by vtable slot; like windows-rs it calls CoIncrementMTAUsage and retries on CO_E_NOTINITIALIZED, but it does not implement windows-rs's reg-free activation fallback for REGDB_E_CLASSNOTREG (UISettings is always registered on Windows 10+).
- The WM_GPUI_* message constants live in platform/windows/stubs.go with their final values until M4-windows-events ports events.rs and deletes them.

### M4-windows-directwrite

- Nothing here has run on Windows yet. Locally only windows/amd64 and windows/arm64 `go vet` and `go test -c` pass. The first CI run (M4-ci-windows) is the first time the vtable slot numbers (internal/win/dwrite/interfaces.go, taken from windows-rs 0.62.2's `*_Vtbl` structs), the struct layout pins (TestDwrite_StructLayout) and every DirectWrite test actually execute.
- Methods that take or return floats (CreateTextFormat, CreateTextLayout, Factory3 CreateGlyphRunAnalysis, Factory4 TranslateColorGlyphRun, FontFace3 GetRecommendedRenderingMode, FontFallbackBuilder AddMapping, TextLayout SetFontSize and Draw, TextFormat GetIncrementalTabStop, the RenderingParams getters) go through purego.RegisterFunc with typed signatures, cached per function address and signature. syscall.SyscallN loads no FP registers on windows/arm64 and com.Call drops the float return. TranslateColorGlyphRun's by-value Vector2 is packed into one uint64 on amd64 and passed as two float32s (an HFA) on arm64.
- The IDWriteTextRenderer callbacks cannot receive floats (windows.NewCallback rejects float parameters), and upstream ignores the baseline origins. DrawGlyphRun has two thunks: amd64 declares the two float slots as ignored uintptrs, and arm64 leaves them out because the floats travel in V registers. The thunk is picked by runtime.GOARCH when the vtable is built. DrawUnderline, DrawStrikethrough and DrawInlineObject read only `this` and the context.
- The RendererContext crosses IDWriteTextLayout.Draw as a win32.Handles key (rendererContexts), never as a Go pointer. DrawGlyphRun runs synchronously under the text system's write lock and uses the locked state directly.
- upstream's `RwLockUpgradableReadGuard` in `font_id` is a read-lock check, then a write lock that checks again before `select_and_cache_font`.
- `DirectXRenderer::get_font_info` (directx_renderer.rs) is implemented for real in platform/windows/stubs.go, as `directXRendererGetFontInfo` / `directXFontInfo`, so colour emoji rasterisation works before M4-windows-d3d-atlas. The type is renamed because direct_write.rs's private `FontInfo` is `fontInfo` in the same package.
- Upstream's `font_style_from_dwrite` maps 1 to Italic and 2 to Oblique, the reverse of the DWRITE constants. It is kept as is.
- The locale is the full 85-unit GetUserDefaultLocaleName buffer with its trailing NULs, as upstream's `HSTRING::from_wide`.
- There are no TextAnalysisSource or TextAnalysisSink implementations: upstream gpui_windows uses neither.
- windows-rs's `Error::new` messages travel back to the caller through IErrorInfo. The port's dwrite.Error messages stop at the COM boundary, so `Draw` returns only the HRESULT.
- CreateTextLayout gets a pointer to a static zero code unit for empty text, as Rust's dangling non-null `Vec::as_ptr()`.

### M4-windows-directx-renderer

- `tryToRecoverFromDeviceLost` (directx_devices.go, M4-windows-d3d-atlas) sleeps with time.Sleep between retries as upstream's thread::sleep; HandleDeviceLost inherits it on the main thread.
- No end-to-end test creates a DirectXRenderer (needs an HWND); first real coverage comes with M4-windows-window and the Windows CI runner.
