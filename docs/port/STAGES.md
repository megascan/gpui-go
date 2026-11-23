# Stage plan

Build plan for the gpui-go port: 101 stages, grouped by milestone, for at most three builders working at the same time in this worktree (`/home/strange/Projects/gpui-go-wt/port`, branch `feat/port`). Design: [DESIGN.md](DESIGN.md). Item and test mapping: [PARITY.md](PARITY.md). Library choices: [RESEARCH-deps.md](RESEARCH-deps.md).

Nothing is built yet. Every stage below lists what it owns, what it needs, its task, and the commands that check only its own files. Sizes in parentheses are upstream lines of Rust and upstream test counts.

## How the plan is laid out

- A stage fits one builder: about 5k upstream lines at most, usually 3-4k.
- `dependsOn` lists a stage only when this stage reads code or a file format that stage creates. Two stages share an owned path only when one depends on the other.
- The root package `gpui` is about 100k upstream lines in one Go package. Two builders editing one package break each other's build, so its 38 stages form a single chain (the *root lane*). The other two lanes take every other package.
- Exit stages (`M0-exit`, `M1-exit`, `M2-exit`, `M3-exit`, `M4-ci-macos`, `M4-ci-windows`, `M4-exit`) are links of the root chain. They are the only stages with authority to fix code across packages, and the only ones that commit and push.
- When more stages are ready than there are free builders, take the earliest in this file.
- Each stage's verify commands are listed in its section and collected in `docs/port/verify.sh`; `sh docs/port/verify.sh <stage>` runs them from the worktree root and stops at the first failure. `sh docs/port/verify.sh final` runs the Final verify list.

Root lane order: M0-geometry -> M0-contracts -> M0-platform-types -> M0-input-types -> M0-scene -> M0-text-core -> M0-profiler-core -> M0-entities -> M0-app -> M0-app-contexts -> M0-test-contexts -> M0-keymap -> M0-gestures-spring -> M0-style -> M0-element -> M0-window-core -> M0-window-paint -> M0-window-dispatch -> M0-div-api -> M0-div-paint -> M0-text-elements -> M0-window-tests -> M0-blocked-tests -> M0-elements-basic -> M0-list -> M0-elements-media -> M0-window-a11y -> M0-exit -> M3-profiler-hang -> M3-profiler-journal -> M3-bench-randomized -> M3-root-misc -> M1-exit -> M2-exit -> M3-exit -> M4-ci-macos -> M4-ci-windows -> M4-exit. The four `M3-examples-*` stages run between `M2-exit` and `M3-exit`.

What the other lanes must deliver for the root lane, in the order it needs them: `M0-foundation` (before `M0-geometry`), `M0-scheduler-core` and `M0-a11y-schema` (before `M0-contracts`), `M0-jsonschema` (before `M0-input-types`), `M0-lyon` (before `M0-scene`), `M0-scheduler-test` (before `M0-entities`), the taffy chain up to `M0-taffy-grid-sizing` (before `M0-element`), `M0-sumtree` (before `M0-list`).

What the root lane unlocks for the others: `M0-scene` -> wgpu renderer; `M0-text-core` -> `text/gotext`; `M0-test-contexts` -> Linux platform; `M0-exit` -> macOS and Windows platform packages (they touch only their own directories, are checked locally by cross-vet, and run their tests in CI).

## Shared decisions

Builders cannot see each other's work. These rules are what keeps their output compatible.

1. Read first, in this order: your stage's section in docs/port/STAGES.md (the full brief; the stage record handed to you is its summary), docs/port/DESIGN.md sections 5, 6 and 9, docs/port/PARITY.md section 3, and docs/api-contracts.md once it exists. They are binding. Where STAGES.md differs from DESIGN or PARITY, STAGES.md wins.
2. Port, do not redesign: upstream is zed @ 6501df15765d66d6c2a2b66b1eb22434ccef3f86. Keep upstream's names (PARITY rules N1-N14), order of operations, constants and panic messages. Re-express only what DESIGN 5.1-5.7 lists (D-NAME, D-RAII, D-OPT, D-VOID, D-EMIT, D-COND, D-INTO, D-GEOM, D-STRING, D-ASYNC, D-RNG, D-ARENA, D-TOKIO, D-HOTRELOAD, D-PROPTEST, D-BENCH, D-SCHEMA, D-FEATURE, D-HTTP, D-MACRO).
3. Reference sources are read-only: `_ref/<crate>/` for Zed; Rust dependencies under `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/<crate>-<version>/` (taffy-0.15.0, etagere-0.2.15, lyon_path-1.0.19, lyon_geom-1.0.20, tiny-skia-path-0.11.4, accesskit-0.24.1, accesskit_consumer-0.38.0, accesskit_atspi_common-0.19.1, accesskit_unix-0.22.1, accesskit_macos-0.26.3, accesskit_windows-0.34.0, cosmic-text-0.19.0, zed-xim-0.4.0-zed, xim-parser-0.2.2, xim-ctext-0.3.0, x11-clipboard-0.9.3, x11rb-protocol-0.13.2, xkbcommon-0.8.0, wayland-protocols-wlr-0.3.12, wayland-protocols-plasma-0.3.12); taffy's test suite in `_ext/taffy` (gitignored clone of tag v0.15.0); Wayland XML in /usr/share/wayland and /usr/share/wayland-protocols.
4. Layout: module `github.com/megascan/gpui-go`, Go 1.27; root package `gpui`; packages collections, gpuiutil, refineable, sumtree, scheduler, a11y (+ a11y/consumer, a11y/atspi, a11y/macos, a11y/windows), internal/taffy, internal/etagere, internal/lyon, internal/jsonschema, internal/cmd/{paritycheck,refinegen,actiongen,stylegen,wlgen,wlspike,shoot}, render, render/wgpu, text/gotext, platform, platform/linux (+ internal/common, internal/wl, internal/wlproto, internal/xkb, internal/xcbshim, wayland, x11), platform/apple, platform/macos, platform/ios, platform/windows, platform/web, examples/<name>. One upstream file maps to one Go file (PARITY F1-F6).
5. The root package has one writer at a time: root stages form a chain through dependsOn. Forward references live in root `stubs.go`: a stage that ports a file deletes that file's symbols from stubs.go and may add panicking stubs (`panic("gpui: unimplemented <symbol>")`) with final signatures for symbols a later stage owns. Other packages only compile against stubs and never call them in tests. stubs.go is deleted by M3-root-misc. internal/taffy and the two Linux backend packages use the same pattern with a package-local stubs.go.
6. A piece never shares a Go package with its stage's core task or with another piece, so pieces can be built at the same time without breaking each other's build. Stay inside your owned paths. A bug in a package you do not own: write a minimal failing repro in your own paths and an entry in docs/port/handoff/<stage>.md; exit stages (M0-exit, M1-exit, M2-exit, M3-exit, M4-ci-*, M4-exit) have fix authority and work through those files.
7. File rule exceptions: window.rs is split into window.go (lines 1-2561, 7891-8411), window_paint.go (2562-5700), window_dispatch.go (5700-7890), window_test.go; div.rs into elements_div.go (1-1904 + Interactivity struct), elements_div_paint.go (1905-4539), elements_div_test.go. Documentation goes under lowercase `docs/` (port tracking in docs/port/); never create `DOCS/`, the macOS and Windows runners have case-insensitive file systems. PARITY F6's `DOCS/` pages become docs/*.md.
8. go.mod and go.sum are owned by M0-bootstrap and already require every module in the Installs list. Builders never run `go get`, `go mod tidy` or `go install` and never add a dependency; if one is missing, stop and write it to docs/port/handoff/<stage>.md. Pinned: purego v0.11.3, oliverbestmann/webgpu v1.38.1, go-text/typesetting v0.3.5, xo/okage v0.20.1, jezek/xgb v1.3.1, jezek/xgbutil v0.0.0-20260124183602-9fd151d6a51a, godbus/dbus/v5 v5.2.4, x/image v0.48.0, x/sys v0.50.0, gen2brain/avif v0.6.0.
9. Parity bookkeeping: each stage edits only the area files in its owns list, row by row (never rewrite a whole file), in the same change as the code. Status is `todo`, `done`, `dev:<id>`, `n/a:<id>` or `blocked:<stage name|owner>` (a test or item that needs a later stage or an owner decision). New deviation ids are registered in a `## Deviations registered here` table at the end of the stage's own area file, not in PARITY.md. Items the scan missed get new rows. `go run ./internal/cmd/paritycheck -area <a> [-done <upstream prefix>,...] [-all-done [-allow-owner]] [-lint] [-rescan]` must pass before a stage is finished. External ports get docs/port/parity/ext-<name>.md created by their first stage.
10. AGENT.md, TECH_DEBT.md, docs/port/PARITY.md and .github/workflows/ci.yml are edited only by M0-bootstrap and the exit stages. Every other stage writes hacks and known issues to docs/port/debt/<stage>.md at the moment it writes them, handoffs to docs/port/handoff/<stage>.md, and on completion creates docs/port/status/<stage>.done with its verify result (one file per stage, so there are no edit conflicts).
11. Go API shape (DESIGN 5-6): generic methods on concrete types (`Entity[T].Update`, `App.New`, `Context[T].Subscribe`), free generic functions only where the receiver would be an interface; closures returning a value use the `R` suffix (`Update` / `UpdateR`); `anyhow::Result<T>` -> `(T, error)`, panics stay panics; returned `Option<T>` -> `(T, bool)` or a nil-able type, Option struct fields -> `gpui.Option[T]` (alias of gpuiutil.Option); iterators -> iter.Seq / iter.Seq2, `impl IntoIterator` parameters -> slices or variadics; field-less enums -> `type X uint8` with `XVariant` constants, data enums -> sealed interfaces or comparable tagged structs; `impl Into<X>` -> the concrete type, a sealed `IntoX` interface, or a union-constrained generic; TypeId -> reflect.Type; `gpui.Task[T]` is an alias of scheduler.Task[T].
12. Elements: `Element[RL, PP]` with typed state behind an unexported drawable, never `any`. Fluent traits are self-typed mixins embedded in each element (`Styled[S]`, `Interactive[S]`, `StatefulInteractive[S]`, `Parent[S]`, `Fluent[S]`) and bound in the element's constructor with the helpers documented in docs/api-contracts.md; chains keep the concrete type (`NewDiv().Flex().SizeFull()` is `*Div`). No element type may declare a method whose name is in docs/port/parity/styled-methods.tsv or the interactive method sets. `Element::id` is `ElementID`, constructors that collide with a type are `New<Type>`.
13. Handles (D-RAII): a handle value aliases one counted handle; `Clone()` makes a new one, `Release()` is the explicit drop (idempotent, use after release panics). Ported code calls Release / Cancel / Unsubscribe / Detach exactly where upstream drops, moves into `detach()`, or lets a value go out of scope. Subscriptions and tasks never stop because of garbage collection; runtime.AddCleanup is only the safety net for entity and focus handles and the leak reporter. Tests in this repo run with strict leak detection.
14. Async (D-ASYNC): tasks are iter.Pull coroutines scheduled by the scheduler package in tests and production; `.await` is `task.Await(cx)`; a foreground task holds `*AsyncApp`, not `*App`, and blocks only in gpui / scheduler primitives (Await, Timer, Chan, Oneshot, Select, Yield), never on a raw channel, mutex wait or time.Sleep. Test randomness comes from the scheduler's seeded PCG (D-RNG).
15. Actions: an action type implements `Name`, `BoxedClone`, `PartialEq` and registers itself with `gpui.RegisterAction` from `init` (by hand or with `//gpui:action`, actiongen). JSON goes through encoding/json/v2 (`encoding/json/v2`, `encoding/json/jsontext`) everywhere in the port; schemas through internal/jsonschema.
16. Tests: every upstream test is one Go test named by rule N14 (`TestKeymap_Foo`, inner module name after the file stem), in `<file>_test.go` of the same package; `#[gpui::test]` bodies run inside `gpui.RunTest` with upstream's seeds / iterations / retries; `#[ignore]` becomes t.Skip with upstream's reason; cfg(target_os) gates become build constraints; benches become `Benchmark<File>_<Name>` on gpui.RunBench. Fix the code, not the test, when a ported test fails. Test support (TestAppContext, TestPlatform) lives in ordinary files of the root package.
17. Test isolation on the developer's machine: `go test` never opens a window, never talks to the session bus, the a11y bus, the keyring, the notification daemon or the system clipboard, and never starts or stops user services. Tests that need a display run only with GPUI_TEST_DISPLAY=1; D-Bus tests use a fake connection or a private dbus-daemon started by the test; credential tests use unique names and delete only what they created; GPU tests skip when no adapter is found.
18. Generated code is committed, named `*_gen.go`, starts with `// Code generated by <tool>; DO NOT EDIT.`, and is produced by a `//go:generate go run github.com/megascan/gpui-go/internal/cmd/<tool> ...` line in a hand-written file of the same package; `go generate ./...` must leave every generated file byte-identical (verify with checksums, not `git diff`: the tree is not committed between stages). Directives are `//gpui:refineable` and `//gpui:action`.
19. Build rules: the root package imports no platform, renderer, cgo or purego code (DESIGN section 4 import rules, enforced by `paritycheck -lint`); render/wgpu is the only package with cgo and is constrained to `linux || js`; Linux packages are `linux`, Apple packages `darwin` (macos: `darwin && !ios`, ios: `ios`), platform/windows `windows`, platform/web `js && wasm`. Everything except render/wgpu must pass `CGO_ENABLED=0 go vet` for darwin, windows and js/wasm from Linux. `runtime.LockOSThread()` is called in `init` of package platform only; the root package never locks a thread. File stems never end in a GOOS / GOARCH word by accident (F3).
20. Seams between packages: platform windows hold a `render.Renderer` (interface in package render); Linux constructs it with `wgpu.NewRenderer`, macOS and Windows with their own renderers; all atlases use internal/etagere. The Linux and web text system is text/gotext (`NewTextSystem`, `NewTextSystemWithoutSystemFonts`). Conversions between gpui types and internal/taffy, internal/etagere or internal/lyon types live in the bridge files (taffy.go, path_builder.go, the atlas implementations), never in geometry.go, style.go or platform.go. Each platform window package has an `a11y.go` seam that the adapter stages fill.
21. GPU data: scene primitive structs keep upstream field order with fixed-size 4-byte-aligned fields and are uploaded with unsafe.Slice; scene_layout_test.go pins sizes and offsets. Shader files (4 WGSL, shaders.metal, shaders.hlsl, alpha_correction.hlsl) are byte-identical copies embedded with //go:embed and checked by sha256 against `_ref`.
22. Code style: gofmt; no comments in code except terse godoc on exported API (first sentence starts with the symbol name), `//go:` and `//gpui:` directives and generated-file headers; no per-file licence headers (LICENSE + NOTICE cover it); errors wrapped with `%w`; no new third-party dependency; no `init` side effects other than registration and the platform thread lock.
23. Verification: a stage is finished when `sh /home/strange/Projects/gpui-go-wt/port/docs/port/verify.sh <stage>` exits 0. It runs exactly the commands in the stage's Verify block in STAGES.md (build, vet, test and gofmt on the stage's own packages, plus paritycheck), never a whole-repo build except in exit stages. Builders do not edit verify.sh or STAGES.md; an exit stage may, changing both together.
24. Running things: shell commands start with `cd /home/strange/Projects/gpui-go-wt/port &&` (the working directory resets between calls). Anything that can take more than about 20 seconds (full test runs, cgo builds of render/wgpu, CI waits) runs in the background; never sleep-poll in the foreground. Run examples and spikes only through internal/cmd/shoot or with `timeout`, at low priority, and leave no window, process or headless output behind.
25. Screenshots: internal/cmd/shoot (built in M1-exit) is the only way examples are captured; images go to docs/port/screenshots/<group>/<example>.png. Open every image with the Read tool and check it for missing content, clipping, blur and wrong framing before it is recorded as verified. What cannot be checked on this machine (macOS, Windows, iOS, a browser) is written to docs/port/OWNER-CHECKS.md as `YOU: <what to check, where>`; it is never claimed as verified.
26. Git: branch feat/port, base dev, no Co-Authored-By lines. Only the exit and CI stages commit and push, and only the paths of stages that have a docs/port/status/<stage>.done file (skip if the harness already committed them); never `git add -A`, never force-push. M0-exit opens the draft PR so the macOS and Windows runners compile the root and leaf packages from then on. The M4 platform packages (platform/apple, macos, ios, windows, web, a11y/macos, a11y/windows) are first committed by M4-ci-macos, M4-ci-windows and M4-exit, so an unfinished platform never turns an earlier exit stage's CI red.
27. Scheduling: at most three stages run at once; when more are ready, take the earliest in the order of docs/port/STAGES.md (milestone order). The root chain is the critical path; the other two lanes feed it (foundation, scheduler, a11y schema, jsonschema, lyon, taffy, sumtree) before anything else.
28. Assumed answers to DESIGN section 14, to be confirmed by the owner: (1) Wayland goes through libwayland-client with purego, with bindings from the in-repo generator; (2) D-Bus is github.com/godbus/dbus/v5; (3) accessibility adapters are native Go ports of accesskit_consumer / atspi / macos / windows, not accesskit-c; (4) action JSON schema comes from internal/jsonschema; (5) what the chosen libraries cannot do (WebGL fallback, multi-threaded web dispatcher, wgpu-native v29 vs upstream v30 differences, screen-capture back ends, iOS gaps) is ported up to the interface and its rows are `blocked:owner`; they are reported, never silently turned into limits.
29. Known parity limits are only L-GLYPH (Linux glyph pixels vs swash), L-JPEG (JPEG decode values) and L-SVGTEXT (text and embedded images inside SVG); each row using one cites the id. Licence: Apache-2.0 LICENSE plus NOTICE crediting Zed Industries (GPUI is Apache-2.0).

## Installs

Run once by the orchestrator before any stage (builders may not install). They pin every Go module the port uses and fetch taffy's test suite, which is not in the published crate.

```sh
cd /home/strange/Projects/gpui-go-wt/port && (test -f go.mod || go mod init github.com/megascan/gpui-go)
cd /home/strange/Projects/gpui-go-wt/port && go get github.com/ebitengine/purego@v0.11.3
cd /home/strange/Projects/gpui-go-wt/port && go get github.com/oliverbestmann/webgpu@v1.38.1
cd /home/strange/Projects/gpui-go-wt/port && go get github.com/go-text/typesetting@v0.3.5
cd /home/strange/Projects/gpui-go-wt/port && go get github.com/xo/okage@v0.20.1
cd /home/strange/Projects/gpui-go-wt/port && go get github.com/jezek/xgb@v1.3.1
cd /home/strange/Projects/gpui-go-wt/port && go get github.com/jezek/xgbutil@v0.0.0-20260124183602-9fd151d6a51a
cd /home/strange/Projects/gpui-go-wt/port && go get github.com/godbus/dbus/v5@v5.2.4
cd /home/strange/Projects/gpui-go-wt/port && go get golang.org/x/image@v0.48.0
cd /home/strange/Projects/gpui-go-wt/port && go get golang.org/x/sys@v0.50.0
cd /home/strange/Projects/gpui-go-wt/port && go get github.com/gen2brain/avif@v0.6.0
git clone --depth 1 --branch v0.15.0 https://github.com/DioxusLabs/taffy /home/strange/Projects/gpui-go-wt/port/_ext/taffy
```

## Parity file split

Done by `M0-bootstrap` (piece `parity-split`) so that stages running at the same time never edit the same area file. Rows move verbatim under their upstream-file headings.

| Current file | New file | Upstream files | Edited by |
|---|---|---|---|
| `parity/linux.md` | `parity/linux-common.md` | linux/dispatcher.rs, headless.rs, headless/window.rs, keyboard.rs, platform.rs, power.rs, system_notifications.rs, text_system.rs, xdg_desktop_portal.rs | M1-linux-common, then M1-linux-platform (Platform impl and display-mode test rows), M2-exit (text_system.rs) |
| `parity/linux.md` | `parity/linux-platform.md` | gpui_linux.rs, linux.rs, linux/display_connection.rs | M1-linux-platform |
| `parity/linux.md` | `parity/linux-wayland.md` | linux/wayland.rs, linux/wayland/* | M1-wayland-client, M1-wayland-window |
| `parity/linux.md` | `parity/linux-x11.md` | linux/x11.rs, linux/x11/* | M1-x11-client, M1-x11-window |
| `parity/wgpu.md` | `parity/wgpu.md` | gpui_wgpu.rs, wgpu_atlas.rs, wgpu_context.rs, wgpu_renderer.rs | M1-wgpu-context, M1-wgpu-renderer, M4-web-wgpu-js |
| `parity/wgpu.md` | `parity/gotext.md` | cosmic_text_system.rs, benches/layout_line.rs | M2-gotext-layout, M2-gotext-raster |
| `parity/core-examples.md` | `parity/examples-m2.md` | examples/hello_world.rs, text.rs, svg/, image/, example_support/fonts.rs | M1-exit, M2-exit |
| `parity/core-examples.md` | `parity/examples-a.md` | active_state_bug, anchor, grid_layout, opacity, gradient, pattern, scrollable, tree, uniform_list, list_example, data_table, paths_bench, popover | M3-examples-a |
| `parity/core-examples.md` | `parity/examples-b.md` | text_layout, text_wrapper, focus_visible, tab_stop, drag_drop, mouse_pressure, a11y, svg_animation, ownership_post, shadow, painting | M3-examples-b |
| `parity/core-examples.md` | `parity/examples-c.md` | window, window_movable, window_positioning, window_shadow, layer_shell, on_window_close_quit, move_entity_between_windows, set_menus, system_notifications, toggle_headless, gif_viewer, image_gallery, image_loading, native_view, native_view_appkit_field | M3-examples-c |
| `parity/core-examples.md` | `parity/examples-d.md` | view_example, testing, input, animation | M3-examples-d |
| `parity/core-examples.md` | `parity/core-tests.md` | gpui/build.rs, benches/focus_cleanup.rs, tests/action_macros.rs, tests/eq_hash.rs | M0-exit, M3-bench-randomized |

Unchanged files and their editors: `foundation.md` (M0-foundation, M0-geometry), `sumtree.md`, `scheduler.md`, `macros.md` and all `core-*.md` (root lane only), `platform-select.md` (M1-linux-platform, M3-platform-benches), `apple.md`, `macos.md`, `ios.md`, `windows.md`, `web.md` (one lane each). New files: `ext-taffy.md`, `ext-etagere.md`, `ext-lyon.md`, `ext-a11y.md`, `ext-a11y-consumer.md`, `ext-a11y-atspi.md`, `ext-a11y-macos.md`, `ext-a11y-windows.md`, `_rescan.md`.

## Final verify

```sh
cd /home/strange/Projects/gpui-go-wt/port && test -z "$(gofmt -l .)"
cd /home/strange/Projects/gpui-go-wt/port && go vet ./...
cd /home/strange/Projects/gpui-go-wt/port && go test ./...
cd /home/strange/Projects/gpui-go-wt/port && go test -race . ./scheduler/...
cd /home/strange/Projects/gpui-go-wt/port && go test -tags gpui_profiler .
cd /home/strange/Projects/gpui-go-wt/port && s=$(find . -name '*_gen.go' | sort | xargs sha256sum) && go generate ./... && [ "$s" = "$(find . -name '*_gen.go' | sort | xargs sha256sum)" ]
cd /home/strange/Projects/gpui-go-wt/port && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go vet ./...
cd /home/strange/Projects/gpui-go-wt/port && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...
cd /home/strange/Projects/gpui-go-wt/port && GOOS=js GOARCH=wasm CGO_ENABLED=0 go vet ./...
cd /home/strange/Projects/gpui-go-wt/port && GOOS=ios GOARCH=arm64 CGO_ENABLED=0 go vet ./platform/apple/... ./platform/ios/... ./platform
cd /home/strange/Projects/gpui-go-wt/port && GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/...
cd /home/strange/Projects/gpui-go-wt/port && go build ./examples/...
cd /home/strange/Projects/gpui-go-wt/port && go run ./internal/cmd/paritycheck -all-done -rescan -lint
cd /home/strange/Projects/gpui-go-wt/port && test ! -e stubs.go && test ! -e DOCS
cd /home/strange/Projects/gpui-go-wt/port && gh pr checks feat/port
```

## Stage index

| # | Stage | Tier | Risky | Depends on |
|---|---|---|---|---|
| 1 | [M0-bootstrap](#m0-bootstrap) | core | yes | - |
| 2 | [M0-foundation](#m0-foundation) | core | no | M0-bootstrap |
| 3 | [M0-scheduler-core](#m0-scheduler-core) | core | no | M0-bootstrap |
| 4 | [M0-scheduler-test](#m0-scheduler-test) | core | no | M0-scheduler-core |
| 5 | [M0-sumtree](#m0-sumtree) | complex | no | M0-bootstrap |
| 6 | [M0-a11y-schema](#m0-a11y-schema) | complex | yes | M0-bootstrap |
| 7 | [M0-jsonschema](#m0-jsonschema) | standard | no | M0-bootstrap |
| 8 | [M0-etagere](#m0-etagere) | standard | no | M0-bootstrap |
| 9 | [M0-lyon](#m0-lyon) | complex | no | M0-bootstrap |
| 10 | [M0-parity-rescan](#m0-parity-rescan) | complex | no | M0-bootstrap |
| 11 | [M0-taffy-style](#m0-taffy-style) | core | no | M0-foundation |
| 12 | [M0-taffy-style-grid](#m0-taffy-style-grid) | complex | no | M0-taffy-style |
| 13 | [M0-taffy-tree](#m0-taffy-tree) | core | no | M0-taffy-style-grid |
| 14 | [M0-taffy-fixtures](#m0-taffy-fixtures) | complex | no | M0-taffy-tree |
| 15 | [M0-taffy-flexbox](#m0-taffy-flexbox) | complex | no | M0-taffy-fixtures |
| 16 | [M0-taffy-block](#m0-taffy-block) | complex | no | M0-taffy-flexbox |
| 17 | [M0-taffy-grid-placement](#m0-taffy-grid-placement) | complex | no | M0-taffy-block |
| 18 | [M0-taffy-grid-sizing](#m0-taffy-grid-sizing) | complex | no | M0-taffy-grid-placement |
| 19 | [M0-taffy-handwritten-tests](#m0-taffy-handwritten-tests) | complex | no | M0-taffy-grid-sizing |
| 20 | [M0-geometry](#m0-geometry) | core | no | M0-foundation |
| 21 | [M0-contracts](#m0-contracts) | core | yes | M0-geometry, M0-a11y-schema, M0-scheduler-core |
| 22 | [M0-platform-types](#m0-platform-types) | core | yes | M0-contracts |
| 23 | [M0-input-types](#m0-input-types) | complex | yes | M0-platform-types, M0-jsonschema |
| 24 | [M0-scene](#m0-scene) | core | yes | M0-input-types, M0-lyon |
| 25 | [M0-text-core](#m0-text-core) | complex | no | M0-scene |
| 26 | [M0-profiler-core](#m0-profiler-core) | complex | no | M0-text-core |
| 27 | [M0-entities](#m0-entities) | core | no | M0-profiler-core, M0-scheduler-test |
| 28 | [M0-app](#m0-app) | core | no | M0-entities |
| 29 | [M0-app-contexts](#m0-app-contexts) | complex | no | M0-app |
| 30 | [M0-test-contexts](#m0-test-contexts) | complex | no | M0-app-contexts |
| 31 | [M0-keymap](#m0-keymap) | complex | no | M0-test-contexts |
| 32 | [M0-gestures-spring](#m0-gestures-spring) | complex | no | M0-keymap |
| 33 | [M0-style](#m0-style) | core | no | M0-gestures-spring |
| 34 | [M0-element](#m0-element) | core | no | M0-style, M0-taffy-grid-sizing |
| 35 | [M0-window-core](#m0-window-core) | core | no | M0-element |
| 36 | [M0-window-paint](#m0-window-paint) | core | no | M0-window-core |
| 37 | [M0-window-dispatch](#m0-window-dispatch) | complex | no | M0-window-paint |
| 38 | [M0-div-api](#m0-div-api) | complex | no | M0-window-dispatch |
| 39 | [M0-div-paint](#m0-div-paint) | complex | no | M0-div-api |
| 40 | [M0-text-elements](#m0-text-elements) | complex | no | M0-div-paint |
| 41 | [M0-window-tests](#m0-window-tests) | complex | no | M0-text-elements |
| 42 | [M0-blocked-tests](#m0-blocked-tests) | complex | no | M0-window-tests |
| 43 | [M0-elements-basic](#m0-elements-basic) | complex | no | M0-blocked-tests |
| 44 | [M0-list](#m0-list) | complex | no | M0-elements-basic, M0-sumtree |
| 45 | [M0-elements-media](#m0-elements-media) | complex | no | M0-list |
| 46 | [M0-window-a11y](#m0-window-a11y) | complex | no | M0-elements-media |
| 47 | [M0-exit](#m0-exit) | complex | yes | M0-window-a11y, M0-taffy-handwritten-tests, M0-parity-rescan |
| 48 | [M1-wayland-ffi](#m1-wayland-ffi) | core | yes | M0-bootstrap |
| 49 | [M1-linux-ffi](#m1-linux-ffi) | complex | no | M0-bootstrap |
| 50 | [M1-x11-proto](#m1-x11-proto) | complex | yes | M0-bootstrap |
| 51 | [M1-wgpu-context](#m1-wgpu-context) | core | no | M0-scene, M0-etagere |
| 52 | [M1-wgpu-renderer](#m1-wgpu-renderer) | complex | no | M1-wgpu-context |
| 53 | [M1-linux-common](#m1-linux-common) | complex | yes | M0-test-contexts, M1-linux-ffi |
| 54 | [M1-wayland-client](#m1-wayland-client) | complex | no | M1-wayland-ffi, M1-linux-common, M1-wgpu-context |
| 55 | [M1-wayland-window](#m1-wayland-window) | complex | no | M1-wayland-client, M1-wgpu-renderer |
| 56 | [M1-x11-client](#m1-x11-client) | complex | no | M1-x11-proto, M1-linux-common, M1-wgpu-context |
| 57 | [M1-x11-window](#m1-x11-window) | complex | no | M1-x11-client, M1-wgpu-renderer |
| 58 | [M1-linux-platform](#m1-linux-platform) | complex | no | M1-wayland-window, M1-x11-window |
| 59 | [M3-profiler-hang](#m3-profiler-hang) | complex | no | M0-exit |
| 60 | [M3-profiler-journal](#m3-profiler-journal) | complex | no | M3-profiler-hang |
| 61 | [M3-bench-randomized](#m3-bench-randomized) | complex | no | M3-profiler-journal |
| 62 | [M3-root-misc](#m3-root-misc) | standard | no | M3-bench-randomized |
| 63 | [M1-exit](#m1-exit) | complex | yes | M3-root-misc, M1-linux-platform |
| 64 | [M2-gotext-layout](#m2-gotext-layout) | complex | no | M0-text-core |
| 65 | [M2-gotext-raster](#m2-gotext-raster) | complex | no | M2-gotext-layout |
| 66 | [M2-exit](#m2-exit) | complex | yes | M1-exit, M2-gotext-raster |
| 67 | [M3-a11y-consumer-tree](#m3-a11y-consumer-tree) | complex | no | M0-a11y-schema |
| 68 | [M3-a11y-consumer-node](#m3-a11y-consumer-node) | complex | no | M3-a11y-consumer-tree |
| 69 | [M3-a11y-consumer-text](#m3-a11y-consumer-text) | complex | no | M3-a11y-consumer-node |
| 70 | [M3-a11y-atspi-common](#m3-a11y-atspi-common) | complex | no | M3-a11y-consumer-text |
| 71 | [M3-a11y-atspi-bus](#m3-a11y-atspi-bus) | complex | yes | M3-a11y-atspi-common |
| 72 | [M3-platform-benches](#m3-platform-benches) | standard | no | M2-exit, M3-bench-randomized |
| 73 | [M3-examples-a](#m3-examples-a) | complex | no | M2-exit |
| 74 | [M3-examples-b](#m3-examples-b) | complex | no | M2-exit |
| 75 | [M3-examples-c](#m3-examples-c) | complex | no | M2-exit |
| 76 | [M3-examples-d](#m3-examples-d) | complex | no | M2-exit |
| 77 | [M3-exit](#m3-exit) | complex | yes | M3-examples-a, M3-examples-b, M3-examples-c, M3-examples-d, M3-a11y-atspi-bus, M3-platform-benches |
| 78 | [M4-apple-ffi](#m4-apple-ffi) | core | yes | M0-exit |
| 79 | [M4-apple-text](#m4-apple-text) | complex | no | M4-apple-ffi |
| 80 | [M4-apple-metal-atlas](#m4-apple-metal-atlas) | complex | no | M4-apple-text, M0-etagere, M1-wgpu-context |
| 81 | [M4-apple-metal-renderer](#m4-apple-metal-renderer) | complex | no | M4-apple-metal-atlas |
| 82 | [M4-macos-input](#m4-macos-input) | complex | no | M4-apple-ffi |
| 83 | [M4-macos-platform](#m4-macos-platform) | complex | no | M4-macos-input, M4-apple-text |
| 84 | [M4-macos-window](#m4-macos-window) | complex | no | M4-macos-platform, M4-apple-metal-renderer |
| 85 | [M4-macos-native-views](#m4-macos-native-views) | complex | no | M4-macos-window |
| 86 | [M4-macos-a11y](#m4-macos-a11y) | complex | no | M4-macos-native-views, M3-a11y-consumer-text |
| 87 | [M4-windows-com](#m4-windows-com) | core | yes | M0-exit |
| 88 | [M4-windows-directwrite](#m4-windows-directwrite) | complex | no | M4-windows-com |
| 89 | [M4-windows-d3d-atlas](#m4-windows-d3d-atlas) | complex | no | M4-windows-directwrite, M0-etagere, M1-wgpu-context |
| 90 | [M4-windows-directx-renderer](#m4-windows-directx-renderer) | complex | no | M4-windows-d3d-atlas |
| 91 | [M4-windows-input-shell](#m4-windows-input-shell) | complex | no | M4-windows-directx-renderer |
| 92 | [M4-windows-window-platform](#m4-windows-window-platform) | complex | yes | M4-windows-input-shell |
| 93 | [M4-windows-a11y](#m4-windows-a11y) | complex | no | M4-windows-window-platform, M3-a11y-consumer-text |
| 94 | [M4-web-wgpu-js](#m4-web-wgpu-js) | complex | no | M3-exit |
| 95 | [M4-web-platform](#m4-web-platform) | complex | no | M4-web-wgpu-js |
| 96 | [M4-web-input-text](#m4-web-input-text) | complex | no | M4-web-platform |
| 97 | [M4-web-example](#m4-web-example) | standard | no | M4-web-input-text |
| 98 | [M4-ios](#m4-ios) | complex | no | M4-apple-metal-renderer, M3-exit |
| 99 | [M4-ci-macos](#m4-ci-macos) | complex | yes | M3-exit, M4-macos-a11y, M4-ios |
| 100 | [M4-ci-windows](#m4-ci-windows) | complex | yes | M4-ci-macos, M4-windows-a11y |
| 101 | [M4-exit](#m4-exit) | complex | yes | M4-ci-windows, M4-web-example |

Tiers: core 20, complex 76, standard 5. Risky: 20.



## M0: foundation and leaf packages

### M0-bootstrap

- Goal: Create the module, licence files, agent docs, CI workflow, the parity checker and the per-stage parity file split.
- Tier: core. Risky: yes.
- Depends on: nothing
- Owns: `go.mod`, `go.sum`, `LICENSE`, `NOTICE`, `AGENT.md`, `TECH_DEBT.md`, `.gitignore`, `.github/workflows/ci.yml`, `doc.go`, `internal/deps/`, `internal/cmd/paritycheck/`, `docs/port/PARITY.md`, `docs/port/parity/`, `docs/port/debt/`, `docs/port/handoff/`, `docs/port/status/`
- Core task: go.mod: module github.com/megascan/gpui-go, `go 1.27`, keep every require the install step added (add them with the exact versions from the Installs list if missing); internal/deps/deps.go (`//go:build deps`) blank-imports each so nothing prunes them. LICENSE (Apache-2.0), NOTICE crediting Zed Industries' GPUI, AGENT.md (stack, the Shared decisions of STAGES.md in short form, pointers to DESIGN/PARITY/STAGES), TECH_DEBT.md (index plus DESIGN section 14 open items), .gitignore adds `/_ext`, doc.go (`package gpui` godoc). ci.yml: pull_request + push to feat/port, matrix ubuntu-latest / macos-14 / windows-latest, Go 1.27.x, ubuntu apt packages for wayland, xkbcommon(+x11), xcb, vulkan + lavapipe, xvfb, node; every OS runs gofmt check, `go vet ./...`, `go test ./...`; ubuntu also runs the three cross-vets, `go generate ./... && git diff --exit-code`, paritycheck and the wasm tests. Written once so no later stage edits it.
- Pieces (each in its own Go package, independent of the core task):
  - **paritycheck** (`internal/cmd/paritycheck/`): CLI that validates the matrix without rescanning upstream. Reads docs/port/PARITY.md section 5 and every docs/port/parity/*.md table. Flags: `-area a,b` (restrict), `-done <upstream path prefix>,...` (no `todo` row under those upstream-file headings), `-all-done` (no `todo` or `blocked:` anywhere; `-allow-owner` tolerates `blocked:owner`), `-lint` (DESIGN section 4 import rules, mixin shadowing against styled-methods.tsv, no `DOCS/` directory, comments other than godoc on exported symbols / `//go:` / `//gpui:` directives / generated headers). Checks: status is `todo|done|dev:<id>|n/a:<id>|blocked:<stage or owner>`; every id is registered in PARITY.md section 5 or in a `## Deviations registered here` table of an area file; every `done` row's Go symbol or test exists (go/parser over the target package, methods as `(Type).Name`); row counts per area never drop below PARITY.md section 2. Unit tests with fixture tables.
  - **parity-split** (`docs/port/parity/`, `docs/port/PARITY.md`): Split area files by upstream-file heading, moving rows verbatim, exactly as the 'Parity file split' table in docs/port/STAGES.md says (linux.md -> linux-common/linux-platform/linux-wayland/linux-x11; wgpu.md -> wgpu + gotext; core-examples.md -> examples-m2/a/b/c/d + core-tests). Update PARITY.md section 2 (area table) and section 8 (add the `blocked:<stage>` status and the rule that new deviation ids are registered in the stage's own area file). Totals must still add up to 6706 items and 869 tests.
- Verify (`sh docs/port/verify.sh M0-bootstrap` runs exactly these, from the worktree root):

```sh
go build ./... && go vet ./... && go test ./internal/... && test -z "$(gofmt -l .)"
go run ./internal/cmd/paritycheck -lint
```

### M0-foundation

- Goal: Port gpui_util, refineable and collections, add Option[T], and write the refinegen generator.
- Tier: core. Risky: no.
- Depends on: M0-bootstrap
- Owns: `gpuiutil/`, `refineable/`, `internal/cmd/refinegen/`, `collections/`, `docs/port/parity/foundation.md`
- Core task: gpuiutil: gpui_util/src/lib.rs (705 LOC, 2 tests; arc_cow.rs is n/a D-STRING) plus `Option[T]` (value type: `Some(v)`, `None[T]()`, `Get() (T, bool)`, `IsSome`, `IsNone`, `UnwrapOr`; the zero value is None). refineable: refineable.rs (Refineable contract of DESIGN 6.6, Cascade, CascadeSlot). refinegen: `//gpui:refineable` on a struct emits `<file>_gen.go` (header `// Code generated by refinegen; DO NOT EDIT.`) with `<T>Refinement` whose fields are `gpuiutil.Option[F]` (a field tagged `//gpui:refineable` nests the field's Refinement type), and Refine / Refined / IsSupersetOf / Subtract plus the From-style conversions derive_refineable.rs (548 LOC) produces; generic structs supported. Golden-file test under testdata. gpui_tokio rows become n/a:D-TOKIO.
- Pieces (each in its own Go package, independent of the core task):
  - **collections** (`collections/`): collections.rs + vecmap.rs (VecMap with its entry API; hash map/set aliases as Go generic aliases or helpers) and the 19 tests of vecmap_tests.rs as TestVecmap_*.
- Verify (`sh docs/port/verify.sh M0-foundation` runs exactly these, from the worktree root):

```sh
go build ./gpuiutil/... && go vet ./gpuiutil/... && go test ./gpuiutil/... && test -z "$(gofmt -l gpuiutil)"
go build ./refineable/... && go vet ./refineable/... && go test ./refineable/... && test -z "$(gofmt -l refineable)"
go build ./internal/cmd/refinegen/... && go vet ./internal/cmd/refinegen/... && go test ./internal/cmd/refinegen/... && test -z "$(gofmt -l internal/cmd/refinegen)"
go build ./collections/... && go vet ./collections/... && go test ./collections/... && test -z "$(gofmt -l collections)"
go run ./internal/cmd/paritycheck -area foundation -done gpui_util/,refineable/,collections/,gpui_tokio/
```

### M0-scheduler-core

- Goal: Port the scheduler crate's clock, scheduler contract and executors with coroutine tasks.
- Tier: core. Risky: no.
- Depends on: M0-bootstrap
- Owns: `scheduler/`, `docs/port/parity/scheduler.md`
- Core task: clock.go, scheduler.go, executor.go from scheduler/src (1174 LOC) exactly to DESIGN 5.6 and 6.3: Priority (weights 60/30/10/0), SessionID, Instant, Clock, Scheduler interface, Runnable + RunnableMeta, Timer, ForegroundExecutor / BackgroundExecutor, `Task[T]` as an `iter.Pull` coroutine (`Await(cx Awaiter) (T, error)`, `Wait`, `Detach`, `Cancel` = stop + private panic unwind so defers run, `IsReady`, `ReadyTask`), `Awaiter`/`Scope`, and the primitives Oneshot[T], Chan[T], Select, Join, Yield. One poll = one `next()`; a body parks only inside these primitives. No goroutine leak when a task is cancelled or its scheduler is dropped. Lifecycle unit tests use a tiny inline FIFO scheduler defined in the test file.
- Verify (`sh docs/port/verify.sh M0-scheduler-core` runs exactly these, from the worktree root):

```sh
go build ./scheduler/... && go vet ./scheduler/... && go test ./scheduler/... && test -z "$(gofmt -l scheduler)"
```

### M0-scheduler-test

- Goal: Port the deterministic TestScheduler and all 31 scheduler tests.
- Tier: core. Risky: no.
- Depends on: M0-scheduler-core
- Owns: `scheduler/`, `docs/port/parity/scheduler.md`
- Core task: test_scheduler.go from test_scheduler.rs (971 LOC): seeded candidate selection (one runnable per foreground session + all background, priority weights), randomize_order, allow_parking, timeout ticks, virtual clock, AdvanceClock, Tick/Run, block with parking and pending-trace checks, SharedRng on math/rand/v2 PCG (D-RNG), end-of-run task drain. scheduler_test.go: the 31 tests of tests.rs as TestScheduler_*; same seed twice must give the same trace. Runs single-goroutine from the caller's point of view: only the polled coroutine runs at any time.
- Verify (`sh docs/port/verify.sh M0-scheduler-test` runs exactly these, from the worktree root):

```sh
go build ./scheduler/... && go vet ./scheduler/... && go test ./scheduler/... && test -z "$(gofmt -l scheduler)" && go test -count=3 ./scheduler/...
go run ./internal/cmd/paritycheck -area scheduler -done scheduler/
```

### M0-sumtree

- Goal: Port the sum_tree crate with its 11 tests.
- Tier: complex. Risky: no.
- Depends on: M0-bootstrap
- Owns: `sumtree/`, `docs/port/parity/sumtree.md`
- Core task: sum_tree.go (2131), cursor.go (861), tree_map.go (531), property_testing.go from sum_tree/src. Item / KeyedItem / Summary / Dimension / SeekTarget / ContextLessSummary become Go interfaces with type parameters; Bias, Edit, SumTree[T], Cursor, FilterCursor, TreeMap, TreeSet keep upstream names; iteration via iter.Seq. Node fan-out constants and split/merge logic identical (tests assert shapes). rayon constructors become sequential with the same result. The randomized tests use a local seeded loop (D-PROPTEST); this package must not import gpui.
- Verify (`sh docs/port/verify.sh M0-sumtree` runs exactly these, from the worktree root):

```sh
go build ./sumtree/... && go vet ./sumtree/... && go test ./sumtree/... && test -z "$(gofmt -l sumtree)"
go run ./internal/cmd/paritycheck -area sumtree -done sum_tree/
```

### M0-a11y-schema

- Goal: Pure-Go copy of the AccessKit 0.24 data model that elements, windows and adapters share.
- Tier: complex. Risky: yes.
- Depends on: M0-bootstrap
- Owns: `a11y/`, `docs/port/parity/ext-a11y.md`
- Core task: Package a11y from accesskit-0.24.1 src/lib.rs (3248) and geometry.rs (866): Role, Action, Orientation, Toggled, Invalid, Live, TextDirection, all other enums with the same numeric values and names, NodeID, Node with every property getter / setter / clearer (flags, node-id vectors, strings, numbers, rects, text properties), Tree, TreeUpdate, ActionRequest, ActionData, the ActivationHandler / ActionHandler / DeactivationHandler interfaces, Rect / Point / Size / Vec2 / Affine. Files directly in a11y/ only (sub-directories belong to later stages). Property storage may be generated by a small go:generate tool in a11y/internal/gen. Create ext-a11y.md with one row per public item; equality and JSON round-trip tests.
- Verify (`sh docs/port/verify.sh M0-a11y-schema` runs exactly these, from the worktree root):

```sh
go build ./a11y && go vet ./a11y && go test ./a11y && test -z "$(gofmt -l a11y)"
go run ./internal/cmd/paritycheck -area ext-a11y
```

### M0-jsonschema

- Goal: In-repo JSON Schema reflector used for action schemas (D-SCHEMA).
- Tier: standard. Risky: no.
- Depends on: M0-bootstrap
- Owns: `internal/jsonschema/`
- Core task: Package jsonschema: `Generator` with a definitions map, `Reflect(t reflect.Type) *Schema`, `Schema` marshalling with encoding/json/v2. Covers what schemars emits for upstream action structs (gpui/src/action.rs:139 and the derive in gpui_macros/src/derive_action.rs): objects with properties/required, json tag names and omitempty, enums as string variants, Option-like fields as nullable, slices, maps, numbers, strings, bools, nested `$defs` references, descriptions from a `doc` struct tag, `additionalProperties: false` when the type opts in. Table-driven tests.
- Verify (`sh docs/port/verify.sh M0-jsonschema` runs exactly these, from the worktree root):

```sh
go build ./internal/jsonschema/... && go vet ./internal/jsonschema/... && go test ./internal/jsonschema/... && test -z "$(gofmt -l internal/jsonschema)"
```

### M0-etagere

- Goal: Port etagere's BucketedAtlasAllocator for the three atlas back ends.
- Tier: standard. Risky: no.
- Depends on: M0-bootstrap
- Owns: `internal/etagere/`, `docs/port/parity/ext-etagere.md`
- Core task: From etagere-0.2.15 src/bucketed.rs (1109) and the types it needs from lib.rs: BucketedAtlasAllocator (New, WithOptions, Allocate, Deallocate, Clear, IsEmpty, Size, Grow, allocated/free space), AllocID, Allocation, AllocatorOptions, int32 Size/Rectangle. Same shelf/bucket selection and id encoding so tile placement matches upstream. Port every unit test in bucketed.rs; list items and tests in ext-etagere.md. allocator.rs (AtlasAllocator) is not used by gpui and is not ported.
- Verify (`sh docs/port/verify.sh M0-etagere` runs exactly these, from the worktree root):

```sh
go build ./internal/etagere/... && go vet ./internal/etagere/... && go test ./internal/etagere/... && test -z "$(gofmt -l internal/etagere)"
go run ./internal/cmd/paritycheck -area ext-etagere
```

### M0-lyon

- Goal: Port the lyon_path / lyon_geom subset that gpui's PathBuilder uses.
- Tier: complex. Risky: no.
- Depends on: M0-bootstrap
- Owns: `internal/lyon/`, `docs/port/parity/ext-lyon.md`
- Core task: Only what gpui/src/path_builder.rs:3-9 imports: Path, Builder, BuilderImpl, WithSvg (SvgPathBuilder: move/line/quadratic/cubic/arc, relative and smooth variants, close), path Event iteration, ArcFlags, Polygon, FillRule / LineCap / LineJoin, math Point / Vector / Transform / Angle, CubicBezierSegment, QuadraticBezierSegment and the SVG-arc to bezier conversion (lyon_geom arc.rs). float32 arithmetic in upstream's operation order so emitted events match value for value. Port the unit tests of the functions you port (lyon_path builder.rs / path.rs, lyon_geom arc.rs); rows in ext-lyon.md. Stroking is done with xo/okage's tiny-skia-path port by the scene stage, not here.
- Verify (`sh docs/port/verify.sh M0-lyon` runs exactly these, from the worktree root):

```sh
go build ./internal/lyon/... && go vet ./internal/lyon/... && go test ./internal/lyon/... && test -z "$(gofmt -l internal/lyon)"
go run ./internal/cmd/paritycheck -area ext-lyon
```

### M0-parity-rescan

- Goal: Add the upstream scanner to paritycheck so completeness of the matrix can be proven.
- Tier: complex. Risky: no.
- Depends on: M0-bootstrap
- Owns: `internal/cmd/paritycheck/scan/`, `docs/port/parity/_rescan.md`
- Core task: Subpackage scan implementing PARITY.md section 9 over `_ref/`: blank comments and strings, track brace depth through mod / impl / trait / fn, collect public items and test functions with kind, line and cfg, expand the style macro tables (gpui_macros/src/styles.rs:699-1417). `paritycheck -rescan` reports items or tests without a row and rows whose upstream item is gone; it must reproduce the per-crate totals of PARITY.md section 2 (6706 items, 869 tests) or explain each difference in _rescan.md. Additionally report (as warnings) items from `bitflags!` and `actions!` invocations and tests inside macro invocations, which the original scan missed. Skips cleanly when `_ref/` is absent (CI).
- Verify (`sh docs/port/verify.sh M0-parity-rescan` runs exactly these, from the worktree root):

```sh
go build ./internal/cmd/paritycheck/... && go vet ./internal/cmd/paritycheck/... && go test ./internal/cmd/paritycheck/... && test -z "$(gofmt -l internal/cmd/paritycheck)"
go run ./internal/cmd/paritycheck -rescan -area foundation,sumtree,scheduler
```

## M0: taffy port (`internal/taffy`, one writer at a time)

### M0-taffy-style

- Goal: Start the taffy 0.15.0 port: geometry, core style types, lengths and math helpers.
- Tier: core. Risky: no.
- Depends on: M0-foundation
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: Source ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/taffy-0.15.0/src; port only code enabled by features std, taffy_tree, flexbox, grid, block_layout, float_layout, content_size (default-features off). This stage: geometry.rs (773), style/mod.rs (1816), style/dimension.rs (762), style/compact_length.rs (793, as a small tagged struct, no pointer packing), style/available_space.rs (163), util/math.rs (356), util/sys.rs (213), with their inline unit tests. It sets the package conventions the next eight stages follow: one Go file per Rust file named by its path (`style_dimension.go`), PARITY N-rules for names, `Option<f32>` as gpuiutil.Option[float32], float32 maths in upstream order, field-less enums as uint8, a package-local stubs.go for forward references. Create ext-taffy.md: every public item and inline test of the crate (rows for later stages stay todo).
- Verify (`sh docs/port/verify.sh M0-taffy-style` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-style-grid

- Goal: Port the remaining taffy style modules: alignment, flex, block, float, grid and the style helpers.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-style
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: style/alignment.rs (979), style/flex.rs (293), style/block.rs (95), style/float.rs (80), style/grid.rs (1978: track sizing functions, repeat, min/max track sizing, grid placement, named lines/areas, GridTemplateComponent), style_helpers.rs (634: the constructor helpers as plain functions, e.g. `Length`, `Percent`, `Auto`, `Fr`, `Minmax`, `Repeat`, `FitContent`, `Line`, `Span`), with inline tests. The Style struct from the previous stage gains its grid and alignment fields here in upstream order.
- Verify (`sh docs/port/verify.sh M0-taffy-style-grid` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-tree

- Goal: Port the taffy tree, cache, layout output types and the algorithm-independent compute code.
- Tier: core. Risky: no.
- Depends on: M0-taffy-style-grid
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: tree/traits.rs (493: the layout tree traits as Go interfaces), tree/cache.rs (555), tree/layout.rs (728), tree/taffy_tree.rs (1543: TaffyTree with slot-map node ids whose uint64 encoding matches slotmap's KeyData, because gpui's LayoutID wraps it), compute/mod.rs (516: root layout, rounding, hidden layout, cached dispatch), compute/leaf.rs (188), compute/common/alignment.rs and sizing_keyword.rs, util/resolve.rs, util/print.rs, util/debug.rs, plus inline tests. Dispatch by Display calls computeFlexboxLayout / computeBlockLayout / computeGridLayout, which this stage declares in stubs.go as panicking stubs with the final signatures.
- Verify (`sh docs/port/verify.sh M0-taffy-tree` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-fixtures

- Goal: Bring in taffy's 6505 XML layout fixtures and port the runner.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-tree
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: From `_ext/taffy` (tag v0.15.0): port tests/xml.rs and tests/common as xml_test.go (parse a fixture, build the tree, install the test measure function, run layout with and without rounding as upstream, compare every node). Pack tests/xml into internal/taffy/testdata/xml.zip with a deterministic tool (internal/taffy/internal/mkfixtures, wired by go:generate; sorted entries, zeroed timestamps). One subtest per fixture named `<category>/<file>`. fixtures_enabled_test.go holds the enabled category list (starts with `leaf`); later stages append theirs. Record the fixture count per category in ext-taffy.md. The runner fails, not skips, on an enabled fixture it cannot parse. The go:generate step does nothing when `_ext/taffy` is absent (CI).
- Verify (`sh docs/port/verify.sh M0-taffy-fixtures` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
s=$(ls internal/taffy/testdata/xml.zip | sort | xargs sha256sum) && go generate ./internal/taffy/... && [ "$s" = "$(ls internal/taffy/testdata/xml.zip | sort | xargs sha256sum)" ]
```

### M0-taffy-flexbox

- Goal: Port the flexbox algorithm and pass every flex fixture.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-fixtures
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: compute/flexbox.rs (3460) with its inline tests: line collection, flexible length resolution, cross sizing, alignment, absolute children, baseline handling, intrinsic sizing paths, in upstream's step order and function names (camelCased). Replace the computeFlexboxLayout stub. Enable the `flex` category in fixtures_enabled_test.go; all of it passes.
- Verify (`sh docs/port/verify.sh M0-taffy-flexbox` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-block

- Goal: Port block layout, floats and out-of-flow positioning and pass their fixtures.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-flexbox
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: compute/block.rs (1824), compute/float.rs (797), compute/oof.rs (912) with inline tests: margin collapsing, block formatting contexts, float placement, absolute/out-of-flow resolution shared with the other algorithms. Replace computeBlockLayout. Enable `block`, `float`, `blockflex`, `contain` and any other category that needs only block + flex; all enabled fixtures pass.
- Verify (`sh docs/port/verify.sh M0-taffy-block` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-grid-placement

- Goal: Port the grid data types, explicit/implicit grid resolution and item placement.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-block
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: compute/grid/types/* (named.rs 998, grid_item.rs 817, cell_occupancy.rs 608, coordinates.rs 184, grid_track.rs 169, grid_track_counts.rs 102), compute/grid/implicit_grid.rs (249), explicit_grid.rs (833), placement.rs (638), util/test_helpers.rs, with the many inline unit tests of these files (they are the main check for this stage; grid fixtures stay disabled).
- Verify (`sh docs/port/verify.sh M0-taffy-grid-placement` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-grid-sizing

- Goal: Port grid track sizing and alignment; the whole fixture suite passes.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-grid-placement
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: compute/grid/mod.rs (1416), track_sizing.rs (1507), alignment.rs (536) with inline tests. Replace computeGridLayout, delete internal/taffy/stubs.go. Enable `grid`, `gridflex`, `blockgrid` and every remaining category: all 6505 fixtures pass. Mark every ext-taffy.md row for src/ done.
- Verify (`sh docs/port/verify.sh M0-taffy-grid-sizing` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
go run ./internal/cmd/paritycheck -area ext-taffy
```

### M0-taffy-handwritten-tests

- Goal: Port taffy's hand-written integration tests.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-grid-sizing
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: `_ext/taffy/tests/hand_written/*.rs` (26 files, about 5k LOC: caching, measure, rounding, relayout, root constraints, min/max overrides, border and padding, and others) as hand_written_<name>_test.go, one Go test per upstream test, rows in ext-taffy.md. Fix any engine bug they expose in the owning file.
- Verify (`sh docs/port/verify.sh M0-taffy-handwritten-tests` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
go run ./internal/cmd/paritycheck -area ext-taffy -all-done
```

## M0: root package chain (one writer at a time)

### M0-geometry

- Goal: Port geometry, color and the string types of the root package.
- Tier: core. Risky: no.
- Depends on: M0-foundation
- Owns: `geometry.go`, `geometry_test.go`, `color.go`, `color_test.go`, `geometry_gen.go`, `shared_string.go`, `shared_uri.go`, `option.go`, `docs/port/parity/core-geometry.md`, `docs/port/parity/foundation.md`
- Core task: geometry.rs (4165 LOC, 139 items, 3 tests) and color.rs (1070, 46 items, 10 tests) in full, with the two families of DESIGN 6.1 (D-GEOM): numeric `Point[T Number]` / Size / Bounds / Edges / Corners with methods, and `PointOf[T any]` / SizeOf / EdgesOf / CornersOf; impls on one non-numeric instantiation become free functions (`EdgesLengthZero`). Pixels, DevicePixels, ScaledPixels, Rems as defined numeric types; Length / DefiniteLength / AbsoluteLength / IntoLength; GridPlacement; Axis, Corner, etc. Field order and 4-byte fields are fixed (GPU buffers). `//gpui:refineable` on Bounds, Edges, Corners -> geometry_gen.go via refinegen. Hsla / Rgba / Background / gradients with identical float32 maths. `type Option[T any] = gpuiutil.Option[T]`, SharedString (D-STRING), SharedURI. Conversions to taffy types are not written here (rows `blocked:M0-element`).
- Verify (`sh docs/port/verify.sh M0-geometry` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-geometry,foundation -done gpui/src/geometry.rs,gpui/src/color.rs,gpui_shared_string/,gpui/src/shared_uri.rs
```

### M0-contracts

- Goal: Land the compiling skeleton of every cross-stage root contract so other lanes can build against it.
- Tier: core. Risky: yes.
- Depends on: M0-geometry, M0-a11y-schema, M0-scheduler-core
- Owns: `gpui.go`, `stubs.go`, `docs/api-contracts.md`, `contracts_test.go`, `docs/port/parity/core-geometry.md`
- Core task: gpui.go: gpui/src/gpui.rs (sealed AppContext / VisualContext of DESIGN 6.2, Global, BorrowAppContext, re-export aliases such as `Task[T] = scheduler.Task[T]`, a11y aliases). stubs.go: declarations for DESIGN 6.2-6.6, 6.10 and 6.11 with the exact names and signatures given there and bodies `panic("gpui: unimplemented <symbol>")`: App, Entity / WeakEntity / AnyEntity / AnyWeakEntity, Context[T], AsyncApp, Subscription, BackgroundExecutor / ForegroundExecutor, Window with the 6.2 method list, FocusHandle, WindowHandle / AnyWindowHandle, ContentMask, DispatchPhase, ElementID / GlobalElementID, Element[RL, PP], AnyElement, IntoElement, Render, RenderOnce, View, the mixin shells Styled / Interactive / StatefulInteractive / Parent / Fluent with their bind helpers, Div / Stateful shells, Style / StyleRefinement, LayoutID, AvailableSpace, Action / ActionData / RegisterAction, Keystroke / Modifiers, Scene, RunTest / TestRun / TestAppContext. docs/api-contracts.md: the stub protocol, which stage replaces which symbol, mixin binding, handle rules, D-VOID naming. contracts_test.go compiles the DESIGN section 6 usage snippets.
- Verify (`sh docs/port/verify.sh M0-contracts` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-geometry -done gpui/src/gpui.rs
```

### M0-platform-types

- Goal: Port platform.rs: every platform interface and platform data type.
- Tier: core. Risky: yes.
- Depends on: M0-contracts
- Owns: `platform.go`, `platform_test.go`, `platform_app_menu.go`, `platform_app_menu_test.go`, `platform_layer_shell.go`, `platform_platform_info.go`, `platform_popup.go`, `platform_keyboard.go`, `platform_cfg_linux.go`, `platform_cfg_windows.go`, `platform_cfg_other.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-platform.md`, `docs/port/parity/core-input.md`
- Core task: platform.rs (4116 LOC, 426 items, 22 tests) plus app_menu.rs, layer_shell.rs, platform_info.rs, popup.rs, keyboard.rs. Interfaces method for method in upstream order: Platform (83), PlatformWindow (108), PlatformDisplay, PlatformDispatcher, PlatformTextSystem, PlatformAtlas, AtlasBackend, PlatformHeadlessRenderer, InputHandler, ScreenCaptureSource / Stream, PlatformKeyboardLayout / Mapper; `PlatformDefaults` and `PlatformWindowDefaults` carry upstream's default bodies; cfg-gated methods exist on every GOOS. Data types: WindowOptions, WindowParams, WindowBounds, decorations, CursorStyle, ClipboardItem / Entry / String, Image + ImageFormat with static and animated decode (stdlib, x/image/webp, gen2brain/avif), AtlasKey / AtlasTile / AtlasState / HeadlessAtlas / AtlasTextureList, NoopTextSystem, PlatformInputHandler, PlatformFrameSignal, WindowButtonLayout, notifications, power. `RawWindowHandle` is a tagged struct (Wayland display+surface, Xcb connection+window+visual, AppKit view, UIKit view, Win32 hwnd+hinstance, Web canvas id). The three GraphicalEnvironment cfg variants go to platform_cfg_<goos>.go. A decoder that does not build for js/wasm sits in a `!js` file with an unsupported-format error on js, so the root package still vets for GOOS=js. etagere conversions are not written here.
- Verify (`sh docs/port/verify.sh M0-platform-types` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-platform -done gpui/src/platform.rs,gpui/src/platform/app_menu.rs,gpui/src/platform/layer_shell.rs,gpui/src/platform/platform_info.rs,gpui/src/platform/popup.rs
```

### M0-input-types

- Goal: Port keystrokes, input events, the input handler, actions and the small app-level helpers.
- Tier: complex. Risky: yes.
- Depends on: M0-platform-types, M0-jsonschema
- Owns: `action.go`, `action_test.go`, `queue.go`, `queue_test.go`, `seeds.go`, `seeds_test.go`, `assets.go`, `assets_test.go`, `util.go`, `util_test.go`, `platform_keystroke.go`, `platform_keystroke_test.go`, `interactive.go`, `input.go`, `global.go`, `colors.go`, `stubs.go`, `docs/port/parity/core-input.md`, `docs/port/parity/core-app.md`, `docs/port/parity/core-geometry.md`, `docs/port/parity/macros.md`, `internal/cmd/actiongen/`
- Core task: platform/keystroke.rs (813: Keystroke, Modifiers, parse / unparse, KeybindingKeystroke), interactive.rs (998: every input event and PlatformInput), input.rs (611: EntityInputHandler, ElementInputHandler), action.rs (456: Action, ActionData, RegisterAction called from init, registry keyed by name and reflect.Type, `Build(jsontext.Value)` on encoding/json/v2, schema through internal/jsonschema, deprecated aliases, NoAction), queue.rs (541, 3 tests), global.rs, seeds.rs, assets.rs, util.rs (the real `Fluent[S]` mixin: When, WhenSome, WhenNone, WhenElse, Map), colors.rs. gpui::tests that need a window get status `blocked:M0-blocked-tests`.
- Pieces (each in its own Go package, independent of the core task):
  - **actiongen** (`internal/cmd/actiongen/`): Generator for `//gpui:action namespace=<ns> [name=<n>] [no_json] [no_register] [deprecated_aliases=a,b] [deprecated=<msg>]` above a struct type: emits `<file>_gen.go` with `Name() string`, `BoxedClone() gpui.Action`, `PartialEq(gpui.Action) bool` and an `init` calling `gpui.RegisterAction(gpui.ActionData{...})` (Build decodes with encoding/json/v2 unless no_json), mirroring gpui_macros/src/derive_action.rs and register_action.rs. Unit-struct lists (`actions!`) use the same directive on `struct{}` types. Golden tests under testdata; the generator only needs DESIGN 6.4, not the root package, to build.
- Verify (`sh docs/port/verify.sh M0-input-types` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go build ./internal/cmd/actiongen/... && go vet ./internal/cmd/actiongen/... && go test ./internal/cmd/actiongen/... && test -z "$(gofmt -l internal/cmd/actiongen)"
go run ./internal/cmd/paritycheck -area core-input,core-app -done gpui/src/platform/keystroke.rs,gpui/src/action.rs,gpui/src/queue.rs,gpui/src/global.rs,gpui/src/assets.rs
```

### M0-scene

- Goal: Port the scene, its primitives, the bounds tree and path building.
- Tier: core. Risky: yes.
- Depends on: M0-input-types, M0-lyon
- Owns: `scene.go`, `scene_test.go`, `bounds_tree.go`, `bounds_tree_test.go`, `path_builder.go`, `path_builder_test.go`, `path_winding.go`, `path_winding_test.go`, `scene_layout_test.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-style.md`, `docs/port/parity/core-geometry.md`
- Core task: scene.rs (1475, 4 tests), bounds_tree.rs (500, 2), path_builder.rs (577), path_winding.rs (842, 5). Primitive structs (Quad, Shadow, Underline, MonochromeSprite, SubpixelSprite, PolychromeSprite, Path + vertices, PaintSurface, native views) keep upstream field order with fixed-size 4-byte-aligned fields only; Scene keeps one slice per kind, paint operations, ordering through the bounds tree, `Batches() iter.Seq[PrimitiveBatch]`. scene_layout_test.go asserts unsafe.Sizeof and field offsets of every primitive against the struct declarations in `_ref/gpui_wgpu/src/shaders.wgsl`, `_ref/gpui_apple/src/shaders.metal` and `_ref/gpui_windows/src/shaders.hlsl` (expected numbers written into the test). PathBuilder runs on internal/lyon and strokes with xo/okage's tiny-skia-path port; TransformationMatrix, PathStyle, StrokeOptions.
- Verify (`sh docs/port/verify.sh M0-scene` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-style,core-geometry -done gpui/src/scene.rs,gpui/src/bounds_tree.rs,gpui/src/path_builder.rs,gpui/src/path_winding.rs
```

### M0-text-core

- Goal: Port the text system front end: fonts, line layout cache and line wrapping.
- Tier: complex. Risky: no.
- Depends on: M0-scene
- Owns: `text_system.go`, `text_system_test.go`, `text_system_line_layout.go`, `text_system_line_layout_test.go`, `text_system_line_wrapper.go`, `text_system_line_wrapper_test.go`, `text_system_font_fallbacks.go`, `text_system_font_features.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-text.md`
- Core task: text_system.rs (1401, 4 tests): TextSystem, WindowTextSystem, Font, FontID, FontFamilyID, FontWeight, FontStyle, FontRun, TextRun, FontMetrics, GlyphID, RenderGlyphParams, fallback stack handling; font_fallbacks.rs, font_features.rs; line_layout.rs (1091, 11 tests): LineLayout, ShapedRun, ShapedGlyph, WrappedLineLayout, LineLayoutCache with its frame swap; line_wrapper.rs (2068, 21 tests): LineWrapper, Boundary, truncation, with go-text/typesetting/segmenter in place of icu LineSegmenter and unicode_segmentation. Offsets stay UTF-8 byte offsets; UTF-16 ranges explicit. Painting (line.rs) comes later. Tests that need TestAppContext are `blocked:M0-blocked-tests`.
- Verify (`sh docs/port/verify.sh M0-text-core` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-text -done gpui/src/text_system/font_fallbacks.rs,gpui/src/text_system/font_features.rs,gpui/src/text_system/line_layout.rs
```

### M0-profiler-core

- Goal: Port the task and action profiler that executors and dispatchers call.
- Tier: complex. Risky: no.
- Depends on: M0-text-core
- Owns: `profiler.go`, `profiler_test.go`, `profiler_on.go`, `profiler_off.go`, `profiler_actions.go`, `profiler_hang_monitor.go`, `stubs.go`, `docs/port/parity/core-profiler.md`
- Core task: profiler.rs (2226, 15 tests), profiler/actions.rs (213), profiler/hang/monitor.rs (120). Upstream's `profiler` cargo feature is the build tag `gpui_profiler` (D-FEATURE): the cfg pairs listed in docs/port/parity/_collisions.md (GetAllTimings, GetCurrentThreadTimings, TakeAllStats, GlobalThreadTimings, UpdateRunningTask, SaveTaskTiming, action stats) get one symbol with bodies in profiler_on.go / profiler_off.go. Thread-local timing buffers become per-goroutine-safe structures keyed by the dispatcher thread id. Hang and journal entry points it calls stay in stubs.go until M3.
- Verify (`sh docs/port/verify.sh M0-profiler-core` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" && go vet -tags gpui_profiler . && go test -tags gpui_profiler -run Profiler .
go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler.rs,gpui/src/profiler/actions.rs,gpui/src/profiler/hang/monitor.rs
```

### M0-entities

- Goal: Port the entity map, subscriptions, executors and the dispatchers behind them.
- Tier: core. Risky: no.
- Depends on: M0-profiler-core, M0-scheduler-test
- Owns: `app_entity_map.go`, `app_entity_map_test.go`, `subscription.go`, `subscription_test.go`, `executor.go`, `executor_test.go`, `platform_scheduler.go`, `platform_scheduler_test.go`, `platform_threaded_dispatcher.go`, `platform_threaded_dispatcher_test.go`, `platform_testing_dispatcher.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-app.md`, `docs/port/parity/core-platform.md`
- Core task: app/entity_map.rs (1421, 5 tests): EntityMap, slots, leases with upstream's double-lease panics, access tracking, ref counts behind a lock, Entity / AnyEntity / WeakEntity / AnyWeakEntity exactly per DESIGN 5.2 (Clone, idempotent Release, runtime.AddCleanup safety net feeding the same dropped-ids queue, `weak.Pointer` to the ref-count table, upgrade = increment-if-not-zero), LeakDetector / HandleID / snapshots / exit report, GPUI_LEAK_DETECTION (strict in this repo's tests) and LEAK_BACKTRACE. subscription.rs (559, 8 tests): SubscriberSet insert-pending / retain / remove-during-dispatch, Subscription Unsubscribe / Detach. executor.rs (626, 2 tests), platform_scheduler.rs (457, 6), platform/threaded_dispatcher.rs (744, 8), platform/test/dispatcher.rs (TestDispatcher over scheduler.TestScheduler).
- Verify (`sh docs/port/verify.sh M0-entities` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" && go test -race -run 'Entity|Subscription|Executor|Scheduler|Dispatcher' .
go run ./internal/cmd/paritycheck -area core-app,core-platform -done gpui/src/app/entity_map.rs,gpui/src/subscription.rs,gpui/src/executor.rs,gpui/src/platform_scheduler.rs,gpui/src/platform/threaded_dispatcher.rs
```

### M0-app

- Goal: Port app.rs: App, Application and the effect system.
- Tier: core. Risky: no.
- Depends on: M0-entities
- Owns: `app.go`, `app_test.go`, `stubs.go`, `docs/port/parity/core-app.md`
- Core task: app.rs (4031 LOC, 209 items): Application / ApplicationHandle (AppCell's RefCell becomes a plain *App), App fields in upstream order, `New` / reserve / insert entity, update / read with leases, the Effect enum and flush loop (app.rs:2254-2440, 3607-3628) with pending_updates nesting, notify / emit / defer / refresh ordering, observers, subscriptions, release and new-entity listeners, globals keyed by reflect.Type with GlobalLease, window map with leasing (update_window), quit modes and handlers with SHUTDOWN_TIMEOUT, keystroke observers and interceptors, active drag, tooltips, SystemWindowTabController, display state and events, HTTPClient interface (D-HTTP), action building and schema listing, the `impl AppContext for App` primitives. Calls into keymap, window and text code go through stubs. The #[test] is ported; gpui::tests are `blocked:M0-test-contexts` or `blocked:M0-blocked-tests`.
- Verify (`sh docs/port/verify.sh M0-app` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
```

### M0-app-contexts

- Goal: Port Context[T], the async contexts, the asset cache, the test macro runtime and TestPlatform.
- Tier: complex. Risky: no.
- Depends on: M0-app
- Owns: `asset_cache.go`, `asset_cache_test.go`, `app_context.go`, `app_async_context.go`, `testing.go`, `platform_testing.go`, `platform_testing_platform.go`, `platform_testing_display.go`, `platform_testing_window.go`, `stubs.go`, `docs/port/parity/core-app.md`, `docs/port/parity/core-platform.md`, `docs/port/parity/macros.md`
- Core task: app/context.rs (938: Context[T] embedding *App with generic methods Observe, Subscribe, Emit, ObserveRelease, ObserveGlobal, Spawn, Listener, Notify, ...), app/async_context.rs (559: AsyncApp, AsyncWindowContext; `Update` parks the coroutine and the poller runs the closure on the main thread, DESIGN 5.6), asset_cache.rs (358), test.rs + gpui_macros/src/test.rs + property_test.rs as testing.go (RunTest with Seed / Seeds / Iterations / Retries / OnFailure, SEED and ITERATIONS env, several TestAppContexts per run, Rand, executor access; PropertyTest without shrinking), platform/test/platform.rs (861), window.rs (742), display.rs: TestPlatform, TestWindow, TestDisplay method for method.
- Verify (`sh docs/port/verify.sh M0-app-contexts` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-app,core-platform -done gpui/src/app/context.rs,gpui/src/app/async_context.rs,gpui/src/platform/test/
```

### M0-test-contexts

- Goal: Port TestAppContext, TestApp, HeadlessAppContext and VisualTestContext; first gpui::tests run.
- Tier: complex. Risky: no.
- Depends on: M0-app-contexts
- Owns: `app_test_context.go`, `app_test_context_test.go`, `app_test_app.go`, `app_test_app_test.go`, `app_headless_app_context.go`, `app_visual_test_context.go`, `asset_cache_test.go`, `app_test.go`, `util_test.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-app.md`, `docs/port/parity/core-geometry.md`
- Core task: app/test_context.rs (1562, 94 items), app/test_app.rs (647), app/headless_app_context.rs (293), app/visual_test_context.rs (485). End of each run: quit the app, drain scheduler tasks, runtime.GC until the live-handle count stops falling, then the leak report (DESIGN 6.11). Port every test whose status is `blocked:M0-test-contexts` and every earlier gpui::test that needs no window (asset_cache 5, util 1, the non-window app.rs tests); window-dependent ones move to `blocked:M0-blocked-tests`. Note in docs/api-contracts.md how other packages write a headless test.
- Verify (`sh docs/port/verify.sh M0-test-contexts` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-app -done gpui/src/app/headless_app_context.rs,gpui/src/asset_cache.rs
```

### M0-keymap

- Goal: Port the keymap, key contexts, the key dispatch tree and tab stops.
- Tier: complex. Risky: no.
- Depends on: M0-test-contexts
- Owns: `keymap.go`, `keymap_test.go`, `keymap_context.go`, `keymap_context_test.go`, `key_dispatch.go`, `key_dispatch_test.go`, `tab_stop.go`, `tab_stop_test.go`, `keymap_binding.go`, `stubs.go`, `docs/port/parity/core-input.md`
- Core task: keymap.rs (1143, 24 tests: binding precedence, pending multi-stroke matches, disabled bindings), keymap/binding.rs (166), keymap/context.rs (888, 11 tests: KeyContext, KeyBindingContextPredicate and its parser), key_dispatch.rs (2200: DispatchTree, DispatchNode, focus path, action and key listeners, binding lookup, replay; 5 plain tests ported, the 20 gpui::tests `blocked:M0-blocked-tests`), tab_stop.rs (501, 10 tests).
- Verify (`sh docs/port/verify.sh M0-keymap` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-input -done gpui/src/keymap.rs,gpui/src/keymap/,gpui/src/tab_stop.rs
```

### M0-gestures-spring

- Goal: Port touch gesture recognition and spring animation maths.
- Tier: complex. Risky: no.
- Depends on: M0-keymap
- Owns: `gestures.go`, `gestures_test.go`, `spring.go`, `spring_test.go`, `stubs.go`, `docs/port/parity/core-input.md`, `docs/port/parity/core-style.md`
- Core task: gestures.rs (2331 LOC, 17 items, 37 plain tests: the touch recognizer state machine, tap / long-press / pan / pinch / momentum) and spring.rs (830, 28 items, 11 tests). Pure state machines and float maths; durations through scheduler.Instant so tests stay deterministic.
- Verify (`sh docs/port/verify.sh M0-gestures-spring` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-input,core-style -done gpui/src/gestures.rs,gpui/src/spring.rs
```

### M0-style

- Goal: Port Style, the Styled mixin and the generator for its 3099 macro methods.
- Tier: core. Risky: no.
- Depends on: M0-gestures-spring
- Owns: `style.go`, `style_test.go`, `style_gen.go`, `styled.go`, `styled_gen.go`, `styled_gen_test.go`, `internal/cmd/stylegen/`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-style.md`, `docs/port/parity/macros.md`
- Core task: style.rs (1529, 3 tests): Style with upstream fields in upstream order, TextStyle, HighlightStyle, BoxShadow, Fill, borders, the copied taffy enums, cascade and refinement (`//gpui:refineable` -> style_gen.go), the paint helpers (calling Window stubs). styled.rs (904): `Styled[S Styler]` with the 116 hand-written methods. internal/cmd/stylegen ports the tables and expansion rules of gpui_macros/src/styles.rs (1417) and writes styled_gen.go; styled_gen_test.go checks the method set of `Styled[*Div]` against docs/port/parity/styled-methods.tsv (3099 names, digit boundaries keep `_`). Parameters follow D-INTO (IntoLength, IntoHsla, IntoFill). Conversions to taffy types are left for M0-element.
- Verify (`sh docs/port/verify.sh M0-style` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go build ./internal/cmd/stylegen/... && go vet ./internal/cmd/stylegen/... && go test ./internal/cmd/stylegen/... && test -z "$(gofmt -l internal/cmd/stylegen)"
s=$(ls styled_gen.go style_gen.go geometry_gen.go | sort | xargs sha256sum) && go generate . && [ "$s" = "$(ls styled_gen.go style_gen.go geometry_gen.go | sort | xargs sha256sum)" ]
go run ./internal/cmd/paritycheck -area core-style -done gpui/src/style.rs,gpui/src/styled.rs
```

### M0-element

- Goal: Port the element machinery, views, the taffy bridge and inspector ids.
- Tier: core. Risky: no.
- Depends on: M0-style, M0-taffy-grid-sizing
- Owns: `element.go`, `element_test.go`, `taffy.go`, `taffy_test.go`, `view.go`, `inspector.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-window.md`, `docs/port/parity/core-style.md`, `docs/port/parity/core-geometry.md`
- Core task: element.rs (1000): Element[RL, PP], the typed drawable with upstream's phase-order panics, AnyElement, IntoElement, Component over RenderOnce (`NewComponent`), the real `Parent[S]` mixin, GlobalElementID with path hash and full-path comparison, ElementA11y. view.rs (667): AnyView, AnyWeakView, view caching, `Entity[T].IntoAnyElement` (D-COND). taffy.rs (787, 1 test): TaffyLayoutEngine over internal/taffy with every `From<gpui type> for taffy type` conversion (also those from geometry.rs and style.rs), device-pixel rounding (taffy.rs:130-134, 423-440, 450-580), measured layouts, LayoutID, AvailableSpace. inspector.rs (271). The 2 plain element tests are ported; the gpui::test is `blocked:M0-div-paint`.
- Verify (`sh docs/port/verify.sh M0-element` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-window,core-style -done gpui/src/view.rs,gpui/src/taffy.rs,gpui/src/inspector.rs
```

### M0-window-core

- Goal: Port the first third of window.rs: window types, focus handles, frames and Window construction.
- Tier: core. Risky: no.
- Depends on: M0-element
- Owns: `window.go`, `stubs.go`, `docs/port/parity/core-window.md`
- Core task: window.rs:1-2561 and 7891-8411 into window.go: DispatchPhase, WindowInvalidator and frame dirtiness, FocusMap / FocusHandle / WeakFocusHandle (ref-counted, D-RAII), Focusable, ManagedView, hitbox types and HitboxBehavior, tooltip types, ElementIdStack, DeferredDraw, Frame with its indices, Window struct fields in upstream order, modifier / input-rate / frame-pacer state, PendingInput and its timeout state machine, default_bounds, `Window::new` (platform window creation and every platform callback wiring), ContentMask, WindowHandle[V] / AnyWindowHandle, ElementID (comparable tagged struct, all From sources through one union-constrained constructor), PaintQuad, quad / fill / outline. The element arena is replaced by per-frame slice pools (D-ARENA). Methods from the later parts stay in stubs.go.
- Verify (`sh docs/port/verify.sh M0-window-core` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
```

### M0-window-paint

- Goal: Port the middle of window.rs: state accessors, the draw cycle, element state and painting.
- Tier: core. Risky: no.
- Depends on: M0-window-core
- Owns: `window_paint.go`, `stubs.go`, `docs/port/parity/core-window.md`
- Core task: window.rs:2562-5700 into window_paint.go: view dirtiness, visibility, focus / blur / focus_next, window state passthroughs, observers and subscriptions, on_next_frame / request_animation_frame, bounds and display changes, rem size, pixel snapping helpers, mouse and modifier accessors, `draw` (3837) and `draw_roots`, entity access recording, present, tooltips, deferred draws, prepaint / paint reuse ranges, with_text_style / content mask / element offset / opacity, transact, autoscroll, assets, `with_element_state` family, tab groups, defer_draw, paint_layer, shadows, paint_quad, paint_path, underlines, paint_glyph / emoji / svg / image / surface. Same order of operations as upstream.
- Verify (`sh docs/port/verify.sh M0-window-paint` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
```

### M0-window-dispatch

- Goal: Port the last third of window.rs: layout requests, hitboxes, event and key dispatch, prompts.
- Tier: complex. Risky: no.
- Depends on: M0-window-paint
- Owns: `window_dispatch.go`, `window_prompts.go`, `window_smoke_test.go`, `stubs.go`, `docs/port/parity/core-window.md`
- Core task: window.rs:5700-7890 into window_dispatch.go and window/prompts.rs (232): request_layout / compute_layout / layout_bounds, insert_hitbox, key context and focus handle registration, input handler, on_mouse_event / on_key_event listeners, dispatch_event, touch dispatch and gesture hand-off, dispatch_mouse_event, dispatch_key_event with pending input and replay, action dispatch on the tree, bindings queries, prompts, on_action, window tab passthroughs, a11y and inspector hooks (bodies that live in later files stay stubs). window_smoke_test.go: a TestAppContext window whose root is a hand-written Element that requests layout, inserts a hitbox and paints a quad; assert the scene has the quad and a simulated click reaches the listener.
- Verify (`sh docs/port/verify.sh M0-window-dispatch` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-window -done gpui/src/window/prompts.rs
```

### M0-div-api

- Goal: Port the builder half of div.rs: Interactivity and the interactive mixins.
- Tier: complex. Risky: no.
- Depends on: M0-window-dispatch
- Owns: `elements_div.go`, `elements_mod.go`, `stubs.go`, `docs/port/parity/core-elements.md`
- Core task: div.rs:1-1904 plus the Interactivity struct (2196-2297): GroupStyle, DragMoveEvent, the Interactivity listener-registration methods (92-791), `Interactive[S]` with every InteractiveElement method (792-1352) and `StatefulInteractive[S]` (1353-1763), listener type aliases, HoverListenerMode, drag / drop / tooltip builder types, `NewDiv`, the Div struct with its four mixins bound in the constructor. Typed listeners (`OnDrag[T]`, `OnDrop[T]`, `OnAction[A]`) are generic methods on the mixin. `ID` takes the ElementIDSource union and returns `*Stateful[S]`.
- Verify (`sh docs/port/verify.sh M0-div-api` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
```

### M0-div-paint

- Goal: Port the layout, prepaint and paint half of div.rs with its tests.
- Tier: complex. Risky: no.
- Depends on: M0-div-api
- Owns: `elements_div_paint.go`, `elements_div_test.go`, `element_test.go`, `stubs.go`, `docs/port/parity/core-elements.md`, `docs/port/parity/core-window.md`
- Core task: div.rs:1905-4539: `impl Element for Div` (request_layout, prepaint with scroll and overflow handling, paint), Interactivity request_layout / prepaint / paint (2298-3717: hover, active, group styles, click and drag state, scroll wheel, focus, key listeners, cursor, inspector), InteractiveElementState, tooltip show / hide machinery (3780-4148), GroupHitboxes, Stateful[E] and its Element impl, ScrollAnchor, ScrollHandle. Port the 28 tests (4540-5865) and the element.rs gpui::test.
- Verify (`sh docs/port/verify.sh M0-div-paint` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-elements,core-window -done gpui/src/elements/div.rs,gpui/src/element.rs
```

### M0-text-elements

- Goal: Port shaped-line painting and the text elements.
- Tier: complex. Risky: no.
- Depends on: M0-div-paint
- Owns: `text_system_line.go`, `text_system_line_test.go`, `elements_text.go`, `elements_text_test.go`, `stubs.go`, `docs/port/parity/core-text.md`, `docs/port/parity/core-elements.md`
- Core task: text_system/line.rs (1871, 12 tests): ShapedLine, WrappedLine, decoration runs, paint and paint_background with underline / strikethrough / background merging. elements/text.rs (1312, 2 tests): `string` and SharedString as elements, StyledText, TextLayout, InteractiveText with click / hover / tooltip ranges. Tests that need a window run now.
- Verify (`sh docs/port/verify.sh M0-text-elements` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-text,core-elements -done gpui/src/text_system/line.rs,gpui/src/elements/text.rs,gpui/src/text_system.rs,gpui/src/text_system/line_wrapper.rs
```

### M0-window-tests

- Goal: Port the window.rs test modules and fix what they expose.
- Tier: complex. Risky: no.
- Depends on: M0-text-elements
- Owns: `window_test.go`, `window.go`, `window_paint.go`, `window_dispatch.go`, `elements_div.go`, `elements_div_paint.go`, `stubs.go`, `docs/port/parity/core-window.md`
- Core task: window.rs:8412-11169: the 47 tests of `mod tests` and `mod inspector_tests` as TestWindow_* in window_test.go, each through RunTest with upstream's seeds / iterations. This is the first time the whole draw and dispatch path runs under upstream's own tests: fix bugs in the window and div files rather than adapting a test. Delete window_smoke_test.go if it is now redundant.
- Verify (`sh docs/port/verify.sh M0-window-tests` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-window -done gpui/src/window.rs
```

### M0-blocked-tests

- Goal: Port every test that was waiting for windows and elements.
- Tier: complex. Risky: no.
- Depends on: M0-window-tests
- Owns: `key_dispatch_test.go`, `interactive_test.go`, `input_test.go`, `app_test.go`, `app_test_context_test.go`, `app_test_app_test.go`, `text_system_line_wrapper_test.go`, `util_test.go`, `stubs.go`, `docs/port/parity/core-input.md`, `docs/port/parity/core-app.md`, `docs/port/parity/core-text.md`, `docs/port/parity/core-geometry.md`
- Core task: Find rows with status `blocked:M0-blocked-tests` in the core-*.md files and port them: key_dispatch.rs (20 gpui::tests), interactive.rs (2), input.rs (2), app.rs (window tests), app/test_context.rs (8), app/test_app.rs (4), line_wrapper.rs (1) and any other. Fix the code under test when a ported test fails; record anything left in docs/port/handoff/M0-blocked-tests.md. After this stage no core row may be `blocked:` on an M0 stage.
- Verify (`sh docs/port/verify.sh M0-blocked-tests` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-input,core-app -done gpui/src/key_dispatch.rs,gpui/src/interactive.rs,gpui/src/input.rs,gpui/src/app.rs,gpui/src/app/test_context.rs,gpui/src/app/test_app.rs
```

### M0-elements-basic

- Goal: Port anchored, animation, canvas, container query, deferred, surface and uniform list.
- Tier: complex. Risky: no.
- Depends on: M0-blocked-tests
- Owns: `elements_anchored.go`, `elements_anchored_test.go`, `elements_animation.go`, `elements_animation_test.go`, `elements_deferred.go`, `elements_deferred_test.go`, `elements_uniform_list.go`, `elements_uniform_list_test.go`, `elements_canvas.go`, `elements_container_query.go`, `elements_surface.go`, `stubs.go`, `docs/port/parity/core-elements.md`
- Core task: elements/anchored.rs (571, 4 tests), animation.rs (1142, 14 tests: Animation, easing functions, AnimationElement, transitions), canvas.rs (87), container_query.rs (118), deferred.rs (311, 2), surface.rs (126), uniform_list.rs (928, 2: UniformList, UniformListScrollHandle, decorations). Each element embeds the mixins upstream implements for it (DESIGN 6.5 list) and binds them in its constructor.
- Verify (`sh docs/port/verify.sh M0-elements-basic` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-elements -done gpui/src/elements/anchored.rs,gpui/src/elements/animation.rs,gpui/src/elements/canvas.rs,gpui/src/elements/container_query.rs,gpui/src/elements/deferred.rs,gpui/src/elements/surface.rs,gpui/src/elements/uniform_list.rs
```

### M0-list

- Goal: Port the variable-height list element and its 41 tests.
- Tier: complex. Risky: no.
- Depends on: M0-elements-basic, M0-sumtree
- Owns: `elements_list.go`, `elements_list_test.go`, `stubs.go`, `docs/port/parity/core-elements.md`
- Core task: elements/list.rs (3883): List, ListState over a sumtree of ListItem with Count / Height dimensions, alignment, follow modes, sizing and measuring behaviours, scroll anchors and pending scrolls, splice / reset / remeasure, layout of visible items with overdraw, scroll event handling, ListOffset; the 41 gpui::tests of `mod test` (1748-3883).
- Verify (`sh docs/port/verify.sh M0-list` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-elements -done gpui/src/elements/list.rs
```

### M0-elements-media

- Goal: Port the image and SVG elements, the image cache and the SVG renderer.
- Tier: complex. Risky: no.
- Depends on: M0-list
- Owns: `elements_img.go`, `elements_img_test.go`, `elements_svg.go`, `elements_svg_test.go`, `svg_renderer.go`, `svg_renderer_test.go`, `elements_image_cache.go`, `stubs.go`, `docs/port/parity/core-elements.md`, `docs/port/parity/core-text.md`
- Core task: elements/img.rs (910, 5 tests: sources, loading / fallback, object fit, animated frames), image_cache.rs (322), svg.rs (729, 4 tests: Svg, transformations, animated svg), svg_renderer.rs (854, 10 tests) on xo/okage in place of resvg / usvg / tiny-skia. Record L-SVGTEXT (svg_renderer.rs:369-517) and L-JPEG rows as the documented limits; everything else must match, including the alpha-mask output format the atlas expects.
- Verify (`sh docs/port/verify.sh M0-elements-media` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-elements,core-text -done gpui/src/elements/img.rs,gpui/src/elements/image_cache.rs,gpui/src/elements/svg.rs,gpui/src/svg_renderer.rs
```

### M0-window-a11y

- Goal: Port the window accessibility tree and native view hosting.
- Tier: complex. Risky: no.
- Depends on: M0-elements-media
- Owns: `window_a11y.go`, `window_a11y_test.go`, `window_native_view.go`, `window_native_view_test.go`, `window_a11y_debug.go`, `stubs.go`, `docs/port/parity/core-window.md`
- Core task: window/a11y.rs (905, 13 tests: building the a11y.TreeUpdate from the frame, focus, synthetic children, action routing), window/a11y/debug.rs (330: the debug JSON dump), window/native_view.rs (897, 11 gpui::tests: NativeView handles, layering, hit testing; platform bodies exist only on macOS). Replace the a11y and native-view stubs left by the window stages.
- Verify (`sh docs/port/verify.sh M0-window-a11y` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-window -done gpui/src/window/
```

### M0-exit

- Goal: Close M0: audit the headless core, port the integration tests, open the PR and get CI green on three OSes.
- Tier: complex. Risky: yes.
- Depends on: M0-window-a11y, M0-taffy-handwritten-tests, M0-parity-rescan
- Owns: `*.go (root package, fixes only)`, `integration_action_macros_test.go`, `integration_eq_hash_test.go`, `macros_derive_context_test.go`, `macros_render_test_test.go`, `docs/port/PARITY.md`, `TECH_DEBT.md`, `AGENT.md`, `docs/port/parity/core-tests.md`, `docs/port/parity/macros.md`
- Core task: Port gpui/tests/action_macros.rs, eq_hash.rs and gpui_macros/tests/derive_context.rs, render_test.rs. Audit: stubs.go holds only profiler-hang / journal / debug-overlay / bench / randomized-tree / inspector-reflection / screen-capture symbols; `paritycheck -rescan` rows added; every M0 row is done, dev, n/a or blocked on an M3 stage or owner; strict leak mode green; `go test -race` on scheduler and root executor tests; cross-vet darwin / windows / js for the root and leaf packages; handoff files addressed. Fold docs/port/debt/*.md into TECH_DEBT.md, refresh AGENT.md. Then commit the paths of finished M0 stages only (those with docs/port/status/<stage>.done; never `git add -A`), push feat/port, `gh pr create --draft --base dev`, and fix CI failures in M0 packages on macOS and Windows.
- Verify (`sh docs/port/verify.sh M0-exit` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
PK=". ./scheduler/... ./sumtree/... ./collections/... ./gpuiutil/... ./refineable/... ./a11y ./internal/taffy/... ./internal/lyon/... ./internal/etagere/... ./internal/jsonschema/... ./internal/cmd/paritycheck/... ./internal/cmd/refinegen/... ./internal/cmd/actiongen/... ./internal/cmd/stylegen/..."; go vet $PK && go test $PK && go test -race . ./scheduler/... && for t in darwin/arm64 windows/amd64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet $PK || exit 1; done
go run ./internal/cmd/paritycheck -area core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,macros,core-tests -rescan
gh pr checks feat/port
```

## M1: Linux platform and wgpu renderer

### M1-wayland-ffi

- Goal: Build the Wayland binding: purego runtime over libwayland-client, the protocol generator, generated protocols and a window + GPU surface spike.
- Tier: core. Risky: yes.
- Depends on: M0-bootstrap
- Owns: `internal/cmd/wlgen/`, `platform/linux/internal/wl/`, `platform/linux/internal/wlproto/`, `internal/cmd/wlspike/`
- Core task: Package wl: dlopen libwayland-client.so.0 and libwayland-cursor.so.0 with purego; display connect / roundtrip / flush / prepare_read / read_events / dispatch_pending / get_fd; proxies created and called through wl_proxy_marshal_array_flags; events through wl_proxy_add_dispatcher with one purego callback that decodes wl_argument arrays; wl_interface / wl_message tables built in pinned Go memory with C layout; fixed, array, fd, new_id and nullable-object arguments; proxy registry for user data; real `wl_display*` / `wl_surface*` pointers exposed as uintptr (DESIGN 8.1). wlgen: protocol XML -> `wlproto/<protocol>_gen.go` (typed proxy per interface, request methods, an event listener struct of funcs, enums and bitfields as typed constants, `since` versions). Vendor XML under wlproto/xml from /usr/share/wayland, /usr/share/wayland-protocols and the wlr / plasma crates: core, xdg-shell, xdg-decoration, xdg-activation, xdg-dialog, xdg-system-bell, fractional-scale, viewporter, cursor-shape, text-input-v3, primary-selection, pointer-gestures, linux-dmabuf, wlr-layer-shell, kde blur, plus any other interface `_ref/gpui_linux/src/linux/wayland/client.rs` binds. wlspike: xdg_toplevel + a wgpu surface from the raw pointers (oliverbestmann/webgpu), clear colour for N frames, exit. Layout tests for the interface tables and a golden test for the generator. All calls from the locked main thread.
- Verify (`sh docs/port/verify.sh M1-wayland-ffi` runs exactly these, from the worktree root):

```sh
go build ./internal/cmd/wlgen/... && go vet ./internal/cmd/wlgen/... && go test ./internal/cmd/wlgen/... && test -z "$(gofmt -l internal/cmd/wlgen)"
go build ./platform/linux/internal/wl/... && go vet ./platform/linux/internal/wl/... && go test ./platform/linux/internal/wl/... && test -z "$(gofmt -l platform/linux/internal/wl)"
go build ./platform/linux/internal/wlproto/... && go vet ./platform/linux/internal/wlproto/... && go test ./platform/linux/internal/wlproto/... && test -z "$(gofmt -l platform/linux/internal/wlproto)"
s=$(find platform/linux/internal/wlproto -name '*_gen.go' | sort | xargs sha256sum) && go generate ./platform/linux/internal/wlproto/... && [ "$s" = "$(find platform/linux/internal/wlproto -name '*_gen.go' | sort | xargs sha256sum)" ]
go vet ./internal/cmd/wlspike && { [ -z "$WAYLAND_DISPLAY" ] || timeout 60 go run ./internal/cmd/wlspike -frames 90; }
```

### M1-linux-ffi

- Goal: purego bindings for xkbcommon and a minimal libxcb shim.
- Tier: complex. Risky: no.
- Depends on: M0-bootstrap
- Owns: `platform/linux/internal/xkb/`, `platform/linux/internal/xcbshim/`
- Core task: Package xkb over libxkbcommon.so.0 and libxkbcommon-x11.so.0: Context, Keymap (from names, string, fd-mapped buffer, X11 device), State (update_mask, update_key, key_get_one_sym, key_get_utf8, serialize mods / layout, mod and led queries, layout names), keysym helpers, compose Table / State (feed, status, utf8, reset). The API mirrors the xkbcommon-rs calls used in `_ref/gpui_linux/src/linux/platform.rs:1302-1602`, wayland/client.rs and x11/client.rs. Explicit Close on every object (D-RAII). Tests: build the default keymap from names and check that the key for `a` yields "a" and shift gives "A"; a null-context rejection test.
- Pieces (each in its own Go package, independent of the core task):
  - **xcbshim** (`platform/linux/internal/xcbshim/`): purego over libxcb.so.1: xcb_connect, xcb_connection_has_error, xcb_get_setup + screen iteration to find a visual id, xcb_flush, xcb_disconnect. It exists only to hand a real `xcb_connection_t*` to the wgpu surface for a window id created over jezek/xgb (DESIGN 8.1). Test connects only when DISPLAY is set and GPUI_TEST_DISPLAY=1.
- Verify (`sh docs/port/verify.sh M1-linux-ffi` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/internal/xkb/... && go vet ./platform/linux/internal/xkb/... && go test ./platform/linux/internal/xkb/... && test -z "$(gofmt -l platform/linux/internal/xkb)"
go build ./platform/linux/internal/xcbshim/... && go vet ./platform/linux/internal/xcbshim/... && go test ./platform/linux/internal/xcbshim/... && test -z "$(gofmt -l platform/linux/internal/xcbshim)"
```

### M1-x11-proto

- Goal: Fill the X11 protocol gaps of jezek/xgb and port the XIM client.
- Tier: complex. Risky: yes.
- Depends on: M0-bootstrap
- Owns: `platform/linux/x11/internal/xext/`, `platform/linux/x11/internal/xim/`
- Core task: Check which extensions used by `_ref/gpui_linux/src/linux/x11/{client,window,event}.rs` (xinput 2.4 incl. gesture pinch and scroll valuators, xkb, randr, render, dri3, present, sync, xfixes, shape) jezek/xgb v1.3.1 lacks, and implement the missing requests, replies and event decoders in package xext on top of xgb's extension and raw-request API, with the same field layouts as x11rb-protocol-0.13.2. Encode / decode tests from byte fixtures.
- Pieces (each in its own Go package, independent of the core task):
  - **xim** (`platform/linux/x11/internal/xim/`): Port the client side of ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/zed-xim-0.4.0-zed with xim-parser-0.2.2 and xim-ctext-0.3.0, limited to what `_ref/gpui_linux/src/linux/x11/xim_handler.rs` and client.rs use: connect through the XIM_SERVERS selection, open, create IC with preedit callbacks and spot location, forward key events, commit, preedit draw / done, close. The transport is an interface (send client message, set / get property) so it does not import the client package; unit tests encode and decode frames against byte fixtures.
- Verify (`sh docs/port/verify.sh M1-x11-proto` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/x11/internal/xext/... && go vet ./platform/linux/x11/internal/xext/... && go test ./platform/linux/x11/internal/xext/... && test -z "$(gofmt -l platform/linux/x11/internal/xext)"
go build ./platform/linux/x11/internal/xim/... && go vet ./platform/linux/x11/internal/xim/... && go test ./platform/linux/x11/internal/xim/... && test -z "$(gofmt -l platform/linux/x11/internal/xim)"
```

### M1-wgpu-context

- Goal: Define the renderer contract and port the wgpu context, atlas and shader loading.
- Tier: core. Risky: no.
- Depends on: M0-scene, M0-etagere
- Owns: `render/`, `docs/port/parity/wgpu.md`
- Core task: render/render.go: the `Renderer` interface exactly as DESIGN 6.9. render/wgpu (build constraint `linux || js`; this stage writes the native files and leaves `_js.go` files to M4): gpui_wgpu.go, wgpu_context.go (848, 3 tests: instance, adapter selection incl. compositor GPU hints, device features, dual-source blending detection), wgpu_atlas.go (568, 5 tests: WgpuAtlas implementing gpui.PlatformAtlas over internal/etagere, per-kind texture lists, uploads), shaders/*.wgsl copied byte for byte from `_ref/gpui_wgpu/src` and embedded, a sha256 test against `_ref` (skips when absent), concatenation as wgpu_renderer.rs:25-39, and a test that compiles all four variants on wgpu-native v29. GPU tests skip when no adapter is found. This is the only package allowed to use cgo.
- Verify (`sh docs/port/verify.sh M1-wgpu-context` runs exactly these, from the worktree root):

```sh
go build ./render/... && go vet ./render/... && go test ./render/... && test -z "$(gofmt -l render)"
go run ./internal/cmd/paritycheck -area wgpu -done gpui_wgpu/src/wgpu_context.rs,gpui_wgpu/src/wgpu_atlas.rs
```

### M1-wgpu-renderer

- Goal: Port WgpuRenderer and the headless renderer.
- Tier: complex. Risky: no.
- Depends on: M1-wgpu-context
- Owns: `render/wgpu/`, `docs/port/parity/wgpu.md`
- Core task: wgpu_renderer.rs (2900, 10 tests): surface configuration from a gpui.RawWindowHandle, pipelines per primitive kind, instance buffers filled with unsafe.Slice views of the scene slices, path rasterisation to the intermediate texture, monochrome / subpixel / polychrome sprites with the subpixel fallback when dual-source blending is missing, surfaces, transparency, drawable resize, device-lost detection and `Recover`, GPUSpecs; WgpuHeadlessRenderer implementing gpui.PlatformHeadlessRenderer (RenderSceneToImage). `NewRenderer` returns a type that satisfies render.Renderer. Add a pixel test: a scene with one solid quad through the headless renderer.
- Verify (`sh docs/port/verify.sh M1-wgpu-renderer` runs exactly these, from the worktree root):

```sh
go build ./render/... && go vet ./render/... && go test ./render/... && test -z "$(gofmt -l render)"
go run ./internal/cmd/paritycheck -area wgpu -done gpui_wgpu/src/
```

### M1-linux-common

- Goal: Port the shared Linux platform layer: client contract, event loop, headless backend, xkb helpers and the D-Bus services.
- Tier: complex. Risky: yes.
- Depends on: M0-test-contexts, M1-linux-ffi
- Owns: `platform/linux/internal/common/`, `docs/port/parity/linux-common.md`
- Core task: From `_ref/gpui_linux/src/linux`: the LinuxClient interface and LinuxCommon state (platform.rs:91-295), dispatcher.rs (369, 1 test) with the calloop loop replaced by a main-goroutine loop over display-fd readiness, the priority runnable queue and timers (background runnables on goroutines, profiler hooks kept), headless.rs + headless/window.rs, keyboard.rs, the helpers of platform.rs (xkb keystroke / modifier / compose / dead-key functions 1302-1602, cursor icon names, open_uri, read_fd_with_timeout, click distance, idle-sleep prevention) with their tests except the display-mode ones, power.rs (265, 2 tests), system_notifications.rs (154, 1), xdg_desktop_portal.rs (249, 1), credentials through the Secret Service, all D-Bus over godbus/dbus/v5. text_system.go returns gpui.NoopTextSystem for now. D-Bus code takes its connection as a parameter; tests use a fake or a private dbus-daemon and never the user's session bus, keyring or notification daemon.
- Verify (`sh docs/port/verify.sh M1-linux-common` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/internal/common/... && go vet ./platform/linux/internal/common/... && go test ./platform/linux/internal/common/... && test -z "$(gofmt -l platform/linux/internal/common)"
go run ./internal/cmd/paritycheck -area linux-common
```

### M1-wayland-client

- Goal: Port the Wayland client: globals, seats, input, IME, data devices and outputs.
- Tier: complex. Risky: no.
- Depends on: M1-wayland-ffi, M1-linux-common, M1-wgpu-context
- Owns: `platform/linux/wayland/`, `docs/port/parity/linux-wayland.md`
- Core task: wayland/client.rs (3454, 10 tests) as client.go: registry and globals, outputs and scale, seats (pointer with axis / gestures, keyboard with xkb and repeat, touch), cursor-shape and themed cursors (cursor.rs 152), text-input-v3 IME, data device drag and drop, primary selection, activation tokens, fractional scale, the LinuxClient implementation, event dispatch into windows. serial.rs (169, 5 tests), display.rs (77, 1 test), wayland.rs. Window methods the client calls are declared in a package-local stubs.go for the next stage. Tests that need a compositor are gated by GPUI_TEST_DISPLAY=1.
- Verify (`sh docs/port/verify.sh M1-wayland-client` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/wayland/... && go vet ./platform/linux/wayland/... && go test ./platform/linux/wayland/... && test -z "$(gofmt -l platform/linux/wayland)"
go run ./internal/cmd/paritycheck -area linux-wayland -done gpui_linux/src/linux/wayland/serial.rs,gpui_linux/src/linux/wayland/display.rs
```

### M1-wayland-window

- Goal: Port the Wayland window, layer shell, popups and clipboard.
- Tier: complex. Risky: no.
- Depends on: M1-wayland-client, M1-wgpu-renderer
- Owns: `platform/linux/wayland/`, `docs/port/parity/linux-wayland.md`
- Core task: wayland/window.rs (2627, 6 tests): WaylandWindow implementing gpui.PlatformWindow (embedding PlatformWindowDefaults), xdg toplevel / popup / layer surfaces (layer_shell.rs, popup.rs), configure and frame callbacks, decorations, tiling and window controls, resize / move, fractional scaling and viewport, blur, input regions, renderer lifecycle through wgpu.NewRenderer with a Wayland RawWindowHandle, device-loss recovery. clipboard.rs (257). a11y.go: a small seam (`newA11yAdapter` returning nil) that M3-exit wires to the AT-SPI adapter. Remove the package stubs.go.
- Verify (`sh docs/port/verify.sh M1-wayland-window` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/wayland/... && go vet ./platform/linux/wayland/... && go test ./platform/linux/wayland/... && test -z "$(gofmt -l platform/linux/wayland)"
go run ./internal/cmd/paritycheck -area linux-wayland -done gpui_linux/src/linux/wayland
```

### M1-x11-client

- Goal: Port the X11 client: connection, input devices, xkb, XIM and event translation.
- Tier: complex. Risky: no.
- Depends on: M1-x11-proto, M1-linux-common, M1-wgpu-context
- Owns: `platform/linux/x11/`, `docs/port/parity/linux-x11.md`
- Core task: x11/client.rs (3219, 11 tests) on jezek/xgb plus internal/xext: setup, atoms, XInput2 devices with scroll valuators and gestures, xkb state and compose, randr displays (display.rs), event loop source, event translation (event.rs 154, 1 test), xim_handler.rs (133) over internal/xim, cursors, the LinuxClient implementation. Window methods it calls go in a package-local stubs.go. Tests that need an X server are gated by GPUI_TEST_DISPLAY=1.
- Verify (`sh docs/port/verify.sh M1-x11-client` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/x11 && go vet ./platform/linux/x11 && go test ./platform/linux/x11 && test -z "$(gofmt -l platform/linux/x11)"
go run ./internal/cmd/paritycheck -area linux-x11 -done gpui_linux/src/linux/x11/event.rs,gpui_linux/src/linux/x11/xim_handler.rs
```

### M1-x11-window

- Goal: Port the X11 window and clipboard.
- Tier: complex. Risky: no.
- Depends on: M1-x11-client, M1-wgpu-renderer
- Owns: `platform/linux/x11/`, `docs/port/parity/linux-x11.md`
- Core task: x11/window.rs (2098): X11Window implementing gpui.PlatformWindow, window creation and properties (EWMH, Motif hints, _NET_WM_SYNC), client-side decoration support, resize / move requests, visuals and transparency, the wgpu surface through internal/xcbshim (second libxcb connection, DESIGN 8.1), present / frame scheduling. x11/clipboard.rs (1287, a port of ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/x11-clipboard-0.9.3 semantics: selections, INCR transfers, targets). a11y.go seam as in the Wayland window. Remove the package stubs.go.
- Verify (`sh docs/port/verify.sh M1-x11-window` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/x11/... && go vet ./platform/linux/x11/... && go test ./platform/linux/x11/... && test -z "$(gofmt -l platform/linux/x11)"
go run ./internal/cmd/paritycheck -area linux-x11 -done gpui_linux/src/linux/x11
```

### M1-linux-platform

- Goal: Port LinuxPlatform, backend selection and the gpui_platform entry points.
- Tier: complex. Risky: no.
- Depends on: M1-wayland-window, M1-x11-window
- Owns: `platform/linux/linux.go`, `platform/linux/platform.go`, `platform/linux/platform_test.go`, `platform/linux/display_connection.go`, `platform/linux/display_connection_test.go`, `platform/platform.go`, `platform/platform_test.go`, `platform/platform_linux.go`, `platform/platform_darwin.go`, `platform/platform_ios.go`, `platform/platform_windows.go`, `platform/platform_js.go`, `platform/platform_other.go`, `docs/port/parity/linux-platform.md`, `docs/port/parity/linux-common.md`, `docs/port/parity/platform-select.md`
- Core task: platform/linux: `impl Platform for LinuxClient` of platform.rs as LinuxPlatform (embedding gpui.PlatformDefaults), linux.rs, display_connection.rs (222: Wayland / X11 / headless selection and switching) with the display-mode tests of platform.rs:2049-2134. Package platform (gpui_platform.rs, 231 LOC, 3 tests): `runtime.LockOSThread()` in init, Application(), Headless(), CurrentPlatform(headless), the headless renderer lookup. The darwin / ios / windows / js files compile and return a platform-not-ported error until their M4 stage replaces them (platform_darwin.go carries `//go:build darwin && !ios`), so CI builds on every OS.
- Verify (`sh docs/port/verify.sh M1-linux-platform` runs exactly these, from the worktree root):

```sh
go build ./platform/... && go vet ./platform/... && go test ./platform ./platform/linux && test -z "$(gofmt -l platform/*.go platform/linux/*.go)"
for t in darwin/arm64 windows/amd64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet ./platform || exit 1; done
go run ./internal/cmd/paritycheck -area linux-platform,linux-common,platform-select -done gpui_linux/src/,gpui_platform/src/
```

## M3: root stages that need only M0 (they keep the root lane busy while the Linux lanes finish)

### M3-profiler-hang

- Goal: Port hang detection and the debug frame overlay.
- Tier: complex. Risky: no.
- Depends on: M0-exit
- Owns: `profiler_hang.go`, `profiler_hang_test.go`, `debug_overlay.go`, `debug_overlay_test.go`, `stubs.go`, `docs/port/parity/core-profiler.md`
- Core task: profiler/hang.rs (1865, 21 tests: hang records, stack capture via runtime.Callers, thresholds, reporting) and debug_overlay.rs (428, 6 tests: frame-time and input-latency overlay drawn by the window). Replace their stubs; window.go's overlay calls now hit real code.
- Verify (`sh docs/port/verify.sh M3-profiler-hang` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" && go test -tags gpui_profiler -run 'Hang|DebugOverlay' .
go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler/hang.rs,gpui/src/debug_overlay.rs
```

### M3-profiler-journal

- Goal: Port the profiler journal.
- Tier: complex. Risky: no.
- Depends on: M3-profiler-hang
- Owns: `profiler_journal.go`, `profiler_journal_test.go`, `stubs.go`, `docs/port/parity/core-profiler.md`
- Core task: profiler/journal.rs (3544, 34 items, 27 tests): the journal data model, ring buffers, frame / task / action records, serialisation and the queries the inspector and overlay use. Keep the on-disk / wire layout byte-compatible with upstream where the file defines one; tests assert it.
- Verify (`sh docs/port/verify.sh M3-profiler-journal` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" && go test -tags gpui_profiler -run Journal .
go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler/journal.rs
```

### M3-bench-randomized

- Goal: Port the bench runtime, BenchAppContext and the randomized element tree.
- Tier: complex. Risky: no.
- Depends on: M3-profiler-journal
- Owns: `app_bench_context.go`, `app_bench_context_test.go`, `randomized_element_tree.go`, `randomized_element_tree_test.go`, `bench.go`, `bench_focus_cleanup_test.go`, `stubs.go`, `docs/port/parity/core-profiler.md`, `docs/port/parity/core-tests.md`, `docs/port/parity/macros.md`
- Core task: bench.rs + gpui_macros/src/bench.rs as `RunBench(b *testing.B, ...)` with upstream's seed handling (D-BENCH), app/bench_context.rs (1614, 6 tests), randomized_element_tree.rs (2058, 6 tests: seeded generators for element trees and interaction scripts, used by the platform benches), gpui/benches/focus_cleanup.rs as BenchmarkFocusCleanup_* in an external test package.
- Verify (`sh docs/port/verify.sh M3-bench-randomized` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" && go test -run xxx -bench FocusCleanup -benchtime 1x .
go run ./internal/cmd/paritycheck -area core-profiler,core-tests -done gpui/src/bench.rs,gpui/src/app/bench_context.rs,gpui/src/randomized_element_tree.rs,gpui/benches/
```

### M3-root-misc

- Goal: Finish the root package: inspector reflection, screen-capture and visual-test types, docs, and remove stubs.go.
- Tier: standard. Risky: no.
- Depends on: M3-bench-randomized
- Owns: `inspector_reflection.go`, `macros_derive_inspector_reflection_test.go`, `platform_scap_screen_capture.go`, `platform_visual_testing.go`, `stubs.go`, `docs/accessibility.md`, `docs/ownership-and-data-flow.md`, `docs/contexts.md`, `docs/key-dispatch.md`, `docs/port/parity/macros.md`, `docs/port/parity/core-platform.md`, `docs/port/parity/core-geometry.md`, `docs/port/parity/core-profiler.md`
- Core task: inspector_reflection.go: reflect over the Styled method set at run time in place of derive_inspector_reflection (gpui_macros 305 LOC) with its test (133). platform/scap_screen_capture.rs (325) and platform/visual_test.rs (280): port the types and interfaces; back ends that have no Go library return upstream's unsupported error and the rows are `blocked:owner` (DESIGN 14.5). Mark arena.rs rows n/a:D-ARENA, hot_reload.rs n/a:D-HOTRELOAD, prelude.rs done by alias note. Rewrite `_ref/gpui/docs/contexts.md`, key_dispatch.md and the two doc modules for the Go API under docs/ (lowercase; run the de-slop skill on them). Delete stubs.go: nothing may be left in it.
- Verify (`sh docs/port/verify.sh M3-root-misc` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" && test ! -e stubs.go && test ! -e DOCS
go run ./internal/cmd/paritycheck -area core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,core-profiler,macros
```

## M1 exit, M2: text back end, SVG and images on screen

### M1-exit

- Goal: Close M1: hello_world opens on Wayland and X11 with quads drawn by wgpu, checked by screenshot.
- Tier: complex. Risky: yes.
- Depends on: M3-root-misc, M1-linux-platform
- Owns: `examples/hello_world/`, `internal/cmd/shoot/`, `docs/port/screenshots/m1/`, `*.go (root package, fixes only)`, `platform/*.go and platform/linux/ (fixes only)`, `render/ (fixes only)`, `docs/port/parity/examples-m2.md`, `TECH_DEBT.md`
- Core task: Port `_ref/gpui/examples/hello_world.rs` to examples/hello_world. Write internal/cmd/shoot: starts an example binary, waits for its window (`hyprctl clients -j` by pid), captures only that window with grim, terminates the process, always cleans up (no leftover windows or headless outputs), `-x11` runs it with WAYLAND_DISPLAY unset, `-may-exit` accepts a program that exits 0 before a window appears (no image written), a timeout fails the run. Run hello_world on Wayland and through XWayland, Read both PNGs and check them (window, coloured quads, no clipping; text is absent until M2). Fix whatever the first end-to-end run exposes in root, platform/linux, render/wgpu. Run the wgpu and platform tests with GPUI_TEST_DISPLAY=1. Commit finished stages, push, check CI.
- Verify (`sh docs/port/verify.sh M1-exit` runs exactly these, from the worktree root):

```sh
go build ./examples/hello_world && go vet ./examples/hello_world ./internal/cmd/shoot && go test . ./platform/... ./render/...
go run ./internal/cmd/shoot -pkg ./examples/hello_world -out docs/port/screenshots/m1/hello_world_wayland.png && go run ./internal/cmd/shoot -x11 -pkg ./examples/hello_world -out docs/port/screenshots/m1/hello_world_x11.png
gh pr checks feat/port
```

### M2-gotext-layout

- Goal: Port the Linux text system's font database, metrics and line shaping onto go-text/typesetting.
- Tier: complex. Risky: no.
- Depends on: M0-text-core
- Owns: `text/gotext/`, `docs/port/parity/gotext.md`
- Core task: Package gotext implementing gpui.PlatformTextSystem from `_ref/gpui_wgpu/src/cosmic_text_system.rs` (1900 LOC, 30 tests): constructors `NewTextSystem(systemFontFallback string)` and `NewTextSystemWithoutSystemFonts(...)`, AddFonts, AllFontNames, FontID resolution with family / weight / style matching and the same fallback order (fontscan), FontMetrics, TypographicBounds, Advance, GlyphForChar, and LayoutLine (shaping with go-text shaping + bidi, font fallback per run, cluster to UTF-8 byte index mapping, emoji runs) producing gpui.LineLayout identical in structure to upstream. Port the tests that cover these paths; glyph rasterisation methods return an error until the next stage.
- Verify (`sh docs/port/verify.sh M2-gotext-layout` runs exactly these, from the worktree root):

```sh
go build ./text/gotext/... && go vet ./text/gotext/... && go test ./text/gotext/... && test -z "$(gofmt -l text/gotext)"
```

### M2-gotext-raster

- Goal: Port glyph raster bounds and rasterisation, and the layout benchmark.
- Tier: complex. Risky: no.
- Depends on: M2-gotext-layout
- Owns: `text/gotext/`, `docs/port/parity/gotext.md`
- Core task: GlyphRasterBounds and RasterizeGlyph with subpixel variants, scale factor, monochrome alpha masks, subpixel (LCD) masks, colour emoji bitmaps, the same output buffer formats and bounds rounding as cosmic_text_system.rs, using xo/okage (or x/image/vector) on go-text outlines. Glyph pixels may differ from swash: register L-GLYPH on those rows and keep tests that assert exact pixels as structural checks (size, format, non-empty coverage). Remaining tests of the 30, and `_ref/gpui_wgpu/benches/layout_line.rs` as BenchmarkLayoutLine_*.
- Verify (`sh docs/port/verify.sh M2-gotext-raster` runs exactly these, from the worktree root):

```sh
go build ./text/gotext/... && go vet ./text/gotext/... && go test ./text/gotext/... && test -z "$(gofmt -l text/gotext)"
go run ./internal/cmd/paritycheck -area gotext -done gpui_wgpu/
```

### M2-exit

- Goal: Close M2: text, SVG and images render on screen through the atlas.
- Tier: complex. Risky: yes.
- Depends on: M1-exit, M2-gotext-raster
- Owns: `platform/linux/internal/common/text_system.go`, `examples/internal/support/`, `examples/text/`, `examples/svg_example/`, `examples/image_example/`, `docs/port/screenshots/m2/`, `*.go (root package, fixes only)`, `platform/*.go and platform/linux/ (fixes only)`, `render/ (fixes only)`, `text/ (fixes only)`, `docs/port/parity/examples-m2.md`, `docs/port/parity/linux-common.md`
- Core task: Wire gotext into the Linux platform (text_system.go). Port example_support/fonts.rs to examples/internal/support and the examples text.rs, svg/ (with dragon.svg) and image/ (with its five assets). Screenshot hello_world, text, svg_example and image_example on Wayland with internal/cmd/shoot; Read every PNG and check text is sharp and correctly positioned, SVG and images are present, nothing is clipped. Fix atlas upload, glyph positioning, gamma and sprite issues wherever they are. Commit finished stages, push, check CI.
- Verify (`sh docs/port/verify.sh M2-exit` runs exactly these, from the worktree root):

```sh
go build ./examples/... && go vet ./examples/... ./text/... ./platform/linux/internal/common && go test . ./text/... ./platform/... ./render/...
for e in hello_world text svg_example image_example; do go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/m2/$e.png || exit 1; done
go run ./internal/cmd/paritycheck -area examples-m2 -all-done
gh pr checks feat/port
```

## M3: accessibility adapters, benches, examples

### M3-a11y-consumer-tree

- Goal: Port the AccessKit consumer tree state.
- Tier: complex. Risky: no.
- Depends on: M0-a11y-schema
- Owns: `a11y/consumer/`, `docs/port/parity/ext-a11y-consumer.md`
- Core task: Package a11y/consumer from ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_consumer-0.38.0/src: lib.rs and tree.rs (3187 incl. tests): Tree, State, ChangeHandler, applying a11y.TreeUpdate (node add / update / remove, focus changes, subtrees), with every unit test in tree.rs. Node methods used by tree.rs are declared in a package stubs.go. Create ext-a11y-consumer.md.
- Verify (`sh docs/port/verify.sh M3-a11y-consumer-tree` runs exactly these, from the worktree root):

```sh
go build ./a11y/consumer/... && go vet ./a11y/consumer/... && go test ./a11y/consumer/... && test -z "$(gofmt -l a11y/consumer)"
```

### M3-a11y-consumer-node

- Goal: Port the consumer node API, iterators and filters.
- Tier: complex. Risky: no.
- Depends on: M3-a11y-consumer-tree
- Owns: `a11y/consumer/`, `docs/port/parity/ext-a11y-consumer.md`
- Core task: node.rs (2092), iterators.rs (1060), filters.rs (421) with their unit tests: Node accessors, relations, bounding boxes and transforms, hit testing, labels and descriptions, the sibling / child iterators with filter results.
- Verify (`sh docs/port/verify.sh M3-a11y-consumer-node` runs exactly these, from the worktree root):

```sh
go build ./a11y/consumer/... && go vet ./a11y/consumer/... && go test ./a11y/consumer/... && test -z "$(gofmt -l a11y/consumer)"
```

### M3-a11y-consumer-text

- Goal: Port the consumer text model.
- Tier: complex. Risky: no.
- Depends on: M3-a11y-consumer-node
- Owns: `a11y/consumer/`, `docs/port/parity/ext-a11y-consumer.md`
- Core task: text.rs (2836) with its tests: text positions and ranges over inline text boxes, character / word / line / paragraph navigation, selections, attributes, bounding rectangles. Remove the package stubs.go; all rows done.
- Verify (`sh docs/port/verify.sh M3-a11y-consumer-text` runs exactly these, from the worktree root):

```sh
go build ./a11y/consumer/... && go vet ./a11y/consumer/... && go test ./a11y/consumer/... && test -z "$(gofmt -l a11y/consumer)"
go run ./internal/cmd/paritycheck -area ext-a11y-consumer -all-done
```

### M3-a11y-atspi-common

- Goal: Port the platform-neutral AT-SPI adapter layer.
- Tier: complex. Risky: no.
- Depends on: M3-a11y-consumer-text
- Owns: `a11y/atspi/`, `docs/port/parity/ext-a11y-atspi.md`
- Core task: Package a11y/atspi from ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_atspi_common-0.19.1/src (4209): node.rs (PlatformNode: role and state mapping, interfaces, actions, text, value, component), adapter.rs (Adapter state, event generation from tree changes), simplified.rs, context.rs, events.rs, rect.rs, util.rs, text_attributes.rs, behind an `AdapterCallback` interface so no D-Bus is needed here. Unit tests as upstream; create ext-a11y-atspi.md.
- Verify (`sh docs/port/verify.sh M3-a11y-atspi-common` runs exactly these, from the worktree root):

```sh
go build ./a11y/atspi && go vet ./a11y/atspi && go test ./a11y/atspi && test -z "$(gofmt -l a11y/atspi)"
```

### M3-a11y-atspi-bus

- Goal: Port the AT-SPI D-Bus adapter that Linux windows use.
- Tier: complex. Risky: yes.
- Depends on: M3-a11y-atspi-common
- Owns: `a11y/atspi/`, `docs/port/parity/ext-a11y-atspi.md`
- Core task: From ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_unix-0.22.1/src (2532) on godbus/dbus/v5: a11y bus discovery (org.a11y.Bus, AT_SPI_BUS_ADDRESS override), registration with the registry, the exported interfaces Accessible, Application, Component, Action, Text, Value, Selection and Cache, event emission, activation tracking, and the public `Adapter` with `NewAdapter(activation, action, deactivation)`, `UpdateIfActive`, `UpdateWindowFocusState`, `SetRootWindowBounds` matching the calls in `_ref/gpui_linux/src/linux/wayland/window.rs` and `x11/window.rs`. The async executor becomes goroutines confined behind the adapter. Tests run against a private dbus-daemon started by the test (skip when the binary is missing); never the user's a11y bus.
- Verify (`sh docs/port/verify.sh M3-a11y-atspi-bus` runs exactly these, from the worktree root):

```sh
go build ./a11y/atspi/... && go vet ./a11y/atspi/... && go test ./a11y/atspi/... && test -z "$(gofmt -l a11y/atspi)"
go run ./internal/cmd/paritycheck -area ext-a11y-atspi -all-done
```

### M3-platform-benches

- Goal: Port the gpui_platform benches.
- Tier: standard. Risky: no.
- Depends on: M2-exit, M3-bench-randomized
- Owns: `platform/bench_atlas_upload_test.go`, `platform/bench_inspector_render_test.go`, `platform/bench_primitives_test.go`, `platform/bench_randomized_element_tree_test.go`, `docs/port/parity/platform-select.md`
- Core task: `_ref/gpui_platform/benches/*.rs` (2589 LOC, 27 bench tests) as external-package benchmarks (`package platform_test`) with gpui.RunBench: atlas upload, inspector render, primitives, randomized element tree, on the headless renderer; they skip without a GPU adapter. Names per rule N14 (BenchmarkRandomizedElementTree_*).
- Verify (`sh docs/port/verify.sh M3-platform-benches` runs exactly these, from the worktree root):

```sh
go vet ./platform && go test -run xxx -bench . -benchtime 1x ./platform && test -z "$(gofmt -l platform/*.go)"
go run ./internal/cmd/paritycheck -area platform-select -all-done
```

### M3-examples-a

- Goal: Port and run examples: active_state_bug, anchor, grid_layout, opacity, gradient, pattern, scrollable, tree, uniform_list, list_example, data_table, paths_bench, popover.
- Tier: complex. Risky: no.
- Depends on: M2-exit
- Owns: `examples/active_state_bug/`, `examples/anchor/`, `examples/grid_layout/`, `examples/opacity/`, `examples/gradient/`, `examples/pattern/`, `examples/scrollable/`, `examples/tree/`, `examples/uniform_list/`, `examples/list_example/`, `examples/data_table/`, `examples/paths_bench/`, `examples/popover/`, `docs/port/screenshots/examples-a/`, `docs/port/parity/examples-a.md`
- Core task: One `package main` per upstream example, a straight port of its .rs file(s) with assets copied next to it; examples with upstream tests keep them as Go tests. For each: build, run with internal/cmd/shoot, Read the PNG and check it against what the upstream source draws. A bug outside examples/ is not fixed here: add a failing repro and an entry to docs/port/handoff/<stage>.md. Examples gated by cfg upstream keep the gate (build constraint plus the upstream fallback message on other OSes). This group is layout and basic painting (about 2.2k upstream LOC).
- Verify (`sh docs/port/verify.sh M3-examples-a` runs exactly these, from the worktree root):

```sh
E="active_state_bug anchor grid_layout opacity gradient pattern scrollable tree uniform_list list_example data_table paths_bench popover"; for e in $E; do go build ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-a/$e.png || exit 1; done
go run ./internal/cmd/paritycheck -area examples-a -all-done
```

### M3-examples-b

- Goal: Port and run examples: text_layout, text_wrapper, focus_visible, tab_stop, drag_drop, mouse_pressure, a11y, svg_animation, ownership_post, shadow, painting.
- Tier: complex. Risky: no.
- Depends on: M2-exit
- Owns: `examples/text_layout/`, `examples/text_wrapper/`, `examples/focus_visible/`, `examples/tab_stop/`, `examples/drag_drop/`, `examples/mouse_pressure/`, `examples/a11y/`, `examples/svg_animation/`, `examples/ownership_post/`, `examples/shadow/`, `examples/painting/`, `docs/port/screenshots/examples-b/`, `docs/port/parity/examples-b.md`
- Core task: One `package main` per upstream example, a straight port of its .rs file(s) with assets copied next to it; examples with upstream tests keep them as Go tests. For each: build, run with internal/cmd/shoot, Read the PNG and check it against what the upstream source draws. A bug outside examples/ is not fixed here: add a failing repro and an entry to docs/port/handoff/<stage>.md. Examples gated by cfg upstream keep the gate (build constraint plus the upstream fallback message on other OSes). This group is text, focus, input devices and painting (about 2.7k LOC). The a11y example only needs to run here; its tree is checked in M3-exit.
- Verify (`sh docs/port/verify.sh M3-examples-b` runs exactly these, from the worktree root):

```sh
E="text_layout text_wrapper focus_visible tab_stop drag_drop mouse_pressure a11y svg_animation ownership_post shadow painting"; for e in $E; do go build ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-b/$e.png || exit 1; done
go run ./internal/cmd/paritycheck -area examples-b -all-done
```

### M3-examples-c

- Goal: Port and run examples: window, window_movable, window_positioning, window_shadow, layer_shell, on_window_close_quit, move_entity_between_windows, set_menus, system_notifications, toggle_headless, gif_viewer, image_gallery, image_loading, native_view, native_view_appkit_field.
- Tier: complex. Risky: no.
- Depends on: M2-exit
- Owns: `examples/window/`, `examples/window_movable/`, `examples/window_positioning/`, `examples/window_shadow/`, `examples/layer_shell/`, `examples/on_window_close_quit/`, `examples/move_entity_between_windows/`, `examples/set_menus/`, `examples/system_notifications/`, `examples/toggle_headless/`, `examples/gif_viewer/`, `examples/image_gallery/`, `examples/image_loading/`, `examples/native_view/`, `examples/native_view_appkit_field/`, `docs/port/screenshots/examples-c/`, `docs/port/parity/examples-c.md`
- Core task: One `package main` per upstream example, a straight port of its .rs file(s) with assets copied next to it; examples with upstream tests keep them as Go tests. For each: build, run with internal/cmd/shoot, Read the PNG and check it against what the upstream source draws. A bug outside examples/ is not fixed here: add a failing repro and an entry to docs/port/handoff/<stage>.md. Examples gated by cfg upstream keep the gate (build constraint plus the upstream fallback message on other OSes). This group is windows, system integration and images (about 3.3k LOC). layer_shell is Wayland-only; native_view and native_view_appkit_field are macOS-only and print upstream's message elsewhere; system_notifications must be run with a private notification target or only built, never sending to the user's notification daemon; set_menus is only visible on macOS. `shoot -may-exit` accepts a program that exits 0 before showing a window.
- Verify (`sh docs/port/verify.sh M3-examples-c` runs exactly these, from the worktree root):

```sh
E="window window_movable window_positioning window_shadow layer_shell on_window_close_quit move_entity_between_windows set_menus system_notifications toggle_headless gif_viewer image_gallery image_loading native_view native_view_appkit_field"; for e in $E; do go build ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -may-exit -pkg ./examples/$e -out docs/port/screenshots/examples-c/$e.png || exit 1; done
go run ./internal/cmd/paritycheck -area examples-c -all-done
```

### M3-examples-d

- Goal: Port and run examples: view_example, testing, input, animation.
- Tier: complex. Risky: no.
- Depends on: M2-exit
- Owns: `examples/view_example/`, `examples/testing/`, `examples/input/`, `examples/animation/`, `docs/port/screenshots/examples-d/`, `docs/port/parity/examples-d.md`
- Core task: One `package main` per upstream example, a straight port of its .rs file(s) with assets copied next to it; examples with upstream tests keep them as Go tests. For each: build, run with internal/cmd/shoot, Read the PNG and check it against what the upstream source draws. A bug outside examples/ is not fixed here: add a failing repro and an entry to docs/port/handoff/<stage>.md. Examples gated by cfg upstream keep the gate (build constraint plus the upstream fallback message on other OSes). This group has upstream tests (view_example 4, testing 7, input 3, animation 1) and about 2.9k LOC; view_example is five files.
- Verify (`sh docs/port/verify.sh M3-examples-d` runs exactly these, from the worktree root):

```sh
E="view_example testing input animation"; for e in $E; do go build ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-d/$e.png || exit 1; done
go run ./internal/cmd/paritycheck -area examples-d -all-done
```

### M3-exit

- Goal: Close M3: every example runs on Linux, accessibility is wired, handoffs are fixed, M0-M3 parity is complete.
- Tier: complex. Risky: yes.
- Depends on: M3-examples-a, M3-examples-b, M3-examples-c, M3-examples-d, M3-a11y-atspi-bus, M3-platform-benches
- Owns: `platform/linux/wayland/a11y.go`, `platform/linux/x11/a11y.go`, `*.go (root package, fixes only)`, `platform/*.go and platform/linux/ (fixes only)`, `render/ (fixes only)`, `text/ (fixes only)`, `a11y/*.go, a11y/consumer/, a11y/atspi/ (fixes only)`, `examples/ (fixes only)`, `docs/port/screenshots/`, `docs/port/PARITY.md`, `TECH_DEBT.md`, `AGENT.md`
- Core task: Wire a11y/atspi.NewAdapter into the Wayland and X11 windows (a11y.go seams) as upstream wires accesskit_unix. Verify examples/a11y by walking its exported tree with a godbus client over a private a11y bus. Work through every docs/port/handoff/*.md entry (fix, then re-run that example's screenshot and Read it). Check that docs/port/screenshots has one checked image per example. Linux `go test ./...`, the three cross-vets, `go generate` cleanliness, paritycheck for all M0-M3 areas. Fold debt files into TECH_DEBT.md, update PARITY.md statuses and AGENT.md. Commit finished M0-M3 stages, push, CI green on all three runners for what is committed (the M4 platform packages are not committed yet).
- Verify (`sh docs/port/verify.sh M3-exit` runs exactly these, from the worktree root):

```sh
go vet ./... && go test ./... && test -z "$(gofmt -l *.go a11y/*.go a11y/consumer a11y/atspi collections gpuiutil refineable scheduler sumtree internal render text platform/*.go platform/linux examples)"
s=$(find . -name '*_gen.go' -not -path './platform/apple/*' -not -path './platform/macos/*' -not -path './platform/windows/*' | sort | xargs sha256sum) && go generate . ./a11y/... ./internal/taffy/... ./platform/linux/... && [ "$s" = "$(find . -name '*_gen.go' -not -path './platform/apple/*' -not -path './platform/macos/*' -not -path './platform/windows/*' | sort | xargs sha256sum)" ]
X='platform/(apple|macos|ios|windows|web)|a11y/(macos|windows)|examples/(web_hello_web|ios_ios)'; for t in darwin/arm64 windows/amd64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet $(GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go list ./... | grep -vE "$X") || exit 1; done
go run ./internal/cmd/paritycheck -lint -rescan -area foundation,sumtree,scheduler,macros,core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,core-profiler,core-tests,examples-m2,examples-a,examples-b,examples-c,examples-d,platform-select,linux-common,linux-platform,linux-wayland,linux-x11,wgpu,gotext -all-done -allow-owner
gh pr checks feat/port
```

## M4: macOS, iOS, Windows, web

### M4-apple-ffi

- Goal: Lay the Apple FFI foundation on purego/objc and port the dispatcher, display link, process info and keychain.
- Tier: core. Risky: yes.
- Depends on: M0-exit
- Owns: `platform/apple/`, `docs/port/parity/apple.md`
- Core task: platform/apple (build constraint darwin, which includes ios; no cgo). internal/objc: class and selector caches, typed msgSend helpers, class registration with methods and protocols, blocks, autorelease pools, NSString / NSArray / NSDictionary / NSData / NSURL / NSError bridging, CGPoint / CGSize / CGRect / NSRange struct passing on arm64 and amd64. internal/cf (CoreFoundation types and retain / release with explicit Release, D-RAII) and internal/gcd (main and global queues by QoS, dispatch_async_f, dispatch_after, purego callbacks). Then dispatcher.rs (201: gpui.PlatformDispatcher on GCD with the profiler hooks), display_link.rs (104), process_info.rs (182, 1 test), gpui_apple.rs, keychain.rs (283, 4 tests: SecItem add / copy / update / delete; tests use a unique service name and remove only their own items). Every binding file gets a size / selector smoke test. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-apple-ffi` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/apple/... || exit 1; done && test -z "$(gofmt -l platform/apple)"
```

### M4-apple-text

- Goal: Port the CoreText text system.
- Tier: complex. Risky: no.
- Depends on: M4-apple-ffi
- Owns: `platform/apple/`, `docs/port/parity/apple.md`
- Core task: text_system.rs (1065, 6 tests) and open_type.rs (171) with internal/ct (CoreText: font descriptors, collections, CTLine / CTRun, font features, fallbacks) and internal/cg (bitmap contexts, colour spaces, paths): font loading and matching, metrics, LayoutLine with UTF-16 to UTF-8 index mapping, glyph raster bounds, rasterisation of monochrome, subpixel and emoji glyphs into the formats the atlas expects. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-apple-text` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/apple/... || exit 1; done && test -z "$(gofmt -l platform/apple)"
go run ./internal/cmd/paritycheck -area apple -done gpui_apple/src/text_system.rs,gpui_apple/src/open_type.rs,gpui_apple/src/dispatcher.rs,gpui_apple/src/keychain.rs
```

### M4-apple-metal-atlas

- Goal: Bind Metal, port the Metal atlas and load the shaders.
- Tier: complex. Risky: no.
- Depends on: M4-apple-text, M0-etagere, M1-wgpu-context
- Owns: `platform/apple/`, `docs/port/parity/apple.md`
- Core task: internal/mtl: MTLDevice, command queue / buffer, render pass descriptors and encoders, render pipeline and depth state, textures, buffers, samplers, library and function objects, CAMetalLayer, all through objc message sends. metal_atlas.rs (572, 6 tests): MetalAtlas implementing gpui.PlatformAtlas over internal/etagere. shaders.metal copied byte for byte and embedded, sha256 test against `_ref`, compiled at start with newLibraryWithSource (upstream's runtime_shaders path, DESIGN 8.3); a CI test compiles it and looks up every vertex / fragment function the renderer names. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-apple-metal-atlas` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/apple/... || exit 1; done && test -z "$(gofmt -l platform/apple)"
```

### M4-apple-metal-renderer

- Goal: Port the Metal renderer.
- Tier: complex. Risky: no.
- Depends on: M4-apple-metal-atlas
- Owns: `platform/apple/`, `docs/port/parity/apple.md`
- Core task: metal_renderer.rs (1903, 3 tests): MetalRenderer satisfying render.Renderer, layer setup and drawable size, instance buffer pool, one draw path per primitive batch with the scene slices viewed through unsafe.Slice, path rasterisation to the intermediate texture, sprites, surfaces (CVPixelBuffer), transparency and presentation with or without transaction, headless render-to-image used by tests. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-apple-metal-renderer` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/apple/... || exit 1; done && test -z "$(gofmt -l platform/apple)"
go run ./internal/cmd/paritycheck -area apple -all-done
```

### M4-macos-input

- Goal: Start the macOS platform: AppKit bindings, events, keyboard, displays and pasteboard.
- Tier: complex. Risky: no.
- Depends on: M4-apple-ffi
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: platform/macos (`//go:build darwin && !ios`). internal/appkit: the NSApplication, NSWindow, NSView, NSEvent, NSScreen, NSPasteboard, NSMenu, NSCursor, NSColor, NSAppearance wrappers the crate needs, on platform/apple/internal/objc. events.rs (579: NSEvent to gpui.PlatformInput), keyboard.rs (1508: layouts, key equivalents, dead keys, the keyboard mapper), display.rs (182), display_link.rs (712, 4 tests), window_appearance.rs, pasteboard.rs (512, 6 tests; tests use a uniquely named pasteboard, never the general one), gpui_macos.rs. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-macos-input` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
```

### M4-macos-platform

- Goal: Port MacPlatform, system notifications and screen capture.
- Tier: complex. Risky: no.
- Depends on: M4-macos-input, M4-apple-text
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: platform.rs (1680): MacPlatform implementing gpui.Platform (embedding PlatformDefaults): application delegate class, run loop, menus and dock menu, open / save panels, URL and file opening, credentials through platform/apple keychain, appearance, cursor, clipboard via pasteboard, displays, thermal and power state, text system and dispatcher from platform/apple. system_notifications.rs (415, 1 test), screen_capture.rs (291). Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-macos-platform` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
```

### M4-macos-window

- Goal: Port the macOS window.
- Tier: complex. Risky: no.
- Depends on: M4-macos-platform, M4-apple-metal-renderer
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: window.rs (4344, 1 test): MacWindow implementing gpui.PlatformWindow; NSWindow / NSPanel and NSView subclasses registered through objc with every delegate and responder method upstream implements; NSTextInputClient IME; drag and drop; traffic lights and titlebar; fullscreen, tabs, sheets and prompts; display-link driven frames; MetalRenderer lifecycle. Split the Go file by concern if needed but keep all symbols in package macos. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-macos-window` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
```

### M4-macos-native-views

- Goal: Port native view hosting on macOS.
- Tier: complex. Risky: no.
- Depends on: M4-macos-window
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: window/native_views.rs (1622, 6 tests): hosting AppKit views inside a gpui window, layering against the Metal layer, hit testing, focus hand-off, used by examples/native_view and native_view_appkit_field. Add a11y.go with a nil-returning adapter seam for the next stage. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-macos-native-views` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
go run ./internal/cmd/paritycheck -area macos -all-done
```

### M4-macos-a11y

- Goal: Port the macOS accessibility adapter and wire it into the window.
- Tier: complex. Risky: no.
- Depends on: M4-macos-native-views, M3-a11y-consumer-text
- Owns: `a11y/macos/`, `platform/macos/a11y.go`, `docs/port/parity/ext-a11y-macos.md`
- Core task: Package a11y/macos from ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_macos-0.26.3/src (2571): node.rs (the NSAccessibilityElement subclass: roles, attributes, actions, text), adapter.rs, subclass.rs (injecting into the content view), event.rs, context.rs, patch.rs, util.rs, on a11y/consumer and the objc helpers. Wire it in platform/macos/a11y.go where `_ref/gpui_macos/src/window.rs` uses accesskit_macos. Create ext-a11y-macos.md. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-macos-a11y` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./a11y/macos/... || exit 1; done && test -z "$(gofmt -l a11y/macos)"
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
```

### M4-windows-com

- Goal: Lay the Windows foundation: COM helpers, Win32 procs, dispatcher, displays, vsync and settings.
- Tier: core. Risky: yes.
- Depends on: M0-exit
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: platform/windows (build constraint windows; no cgo). internal/com: IUnknown, vtable calls through syscall.SyscallN, GUIDs, HRESULT to error, BSTR / VARIANT helpers, ref-counted wrappers with explicit Release, and a way to implement COM interfaces in Go (vtables of windows.NewCallback thunks, used later for drop targets, DirectWrite callbacks and UI Automation). internal/win32: the user32 / gdi32 / dwmapi / shcore / imm32 / ole32 / shell32 procs and structs x/sys/windows lacks. Then util.rs, wrapper.rs, dispatcher.rs (191: thread pool + message-window dispatch with the profiler hooks), display.rs (244), vsync.rs (119, 2 tests), system_settings.rs (87), gpui_windows.rs. Struct-size tests for every hand-written struct. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-windows-com` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-windows-directwrite

- Goal: Port the DirectWrite text system.
- Tier: complex. Risky: no.
- Depends on: M4-windows-com
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: internal/dwrite (factory, font collections and sets, font faces, text layout and analysis, glyph runs, rendering params, the custom text renderer and analysis source / sink implemented in Go) and direct_write.rs (2076, 4 tests): font loading and fallback, metrics, LayoutLine with UTF-16 to UTF-8 mapping, raster bounds, glyph rasterisation incl. colour emoji through the GPU path upstream uses. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-windows-directwrite` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
go run ./internal/cmd/paritycheck -area windows -done gpui_windows/src/direct_write.rs,gpui_windows/src/dispatcher.rs,gpui_windows/src/vsync.rs
```

### M4-windows-d3d-atlas

- Goal: Bind Direct3D 11, DXGI and DirectComposition, port devices and the atlas, load the HLSL shaders.
- Tier: complex. Risky: no.
- Depends on: M4-windows-directwrite, M0-etagere, M1-wgpu-context
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: internal/d3d: D3D11 device / context / buffers / textures / views / states / shaders, DXGI factory / adapter / swap chain, DirectComposition device / target / visual, D3DCompile from d3dcompiler_47. directx_devices.rs (194), directx_atlas.rs (420, 1 test: DirectXAtlas implementing gpui.PlatformAtlas over internal/etagere). shaders.hlsl and alpha_correction.hlsl copied byte for byte and embedded, sha256 test against `_ref`, compiled at start with D3DCompile using the entry points and targets of `_ref/gpui_windows/build.rs:23-201`; a CI test compiles every entry point. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-windows-d3d-atlas` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-windows-directx-renderer

- Goal: Port the DirectX renderer.
- Tier: complex. Risky: no.
- Depends on: M4-windows-d3d-atlas
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: directx_renderer.rs (2083): DirectXRenderer satisfying render.Renderer, swap chain and composition setup, per-primitive pipelines and instance buffers viewed with unsafe.Slice, path rasterisation with MSAA to the intermediate target, sprites, transparency, resize, device-lost handling and recovery, GPU specs. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-windows-directx-renderer` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-windows-input-shell

- Goal: Port Windows event handling, keyboard, direct manipulation, clipboard, dialogs, jump lists and notifications.
- Tier: complex. Risky: no.
- Depends on: M4-windows-directx-renderer
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: events.rs (1844: the window procedure message handlers, mouse / pointer / touch, keyboard and IME composition, non-client hit testing, DPI changes), keyboard.rs (371, 1 test), direct_manipulation.rs (361), clipboard.rs (402), dialog.rs (340, 6 tests), destination_list.rs (207), system_notifications.rs (198). Clipboard tests are gated behind an env switch so CI does not fight other processes. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-windows-input-shell` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-windows-window-platform

- Goal: Port the Windows window and WindowsPlatform.
- Tier: complex. Risky: yes.
- Depends on: M4-windows-input-shell
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: window.rs (1742, 2 tests): WindowsWindow implementing gpui.PlatformWindow, class registration, creation, state, drag and drop target as a Go COM object, renderer lifecycle. platform.rs (2045, 4 tests): WindowsPlatform implementing gpui.Platform, message loop, credentials through the Windows credential manager (tests use a unique target name and delete only it), file dialogs, shell integration, keyboard layout, displays. Add a11y.go with a nil adapter seam. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-windows-window-platform` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
go run ./internal/cmd/paritycheck -area windows -all-done
```

### M4-windows-a11y

- Goal: Port the UI Automation adapter and wire it into the window.
- Tier: complex. Risky: no.
- Depends on: M4-windows-window-platform, M3-a11y-consumer-text
- Owns: `a11y/windows/`, `platform/windows/a11y.go`, `docs/port/parity/ext-a11y-windows.md`
- Core task: Package a11y/windows from ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_windows-0.34.0/src (4476 incl. tests): node.rs (IRawElementProviderSimple / Fragment / FragmentRoot and the pattern providers as Go COM objects), text.rs (ITextProvider / ITextRangeProvider), adapter.rs (WM_GETOBJECT handling, event raising), util.rs, subclass.rs, and the tests under tests/. Wire it in platform/windows/a11y.go where `_ref/gpui_windows/src/window.rs` and `events.rs` use accesskit_windows. Create ext-a11y-windows.md. Tests here run only in CI on the real OS; locally the check is the cross-vet, so keep every test compiling.
- Verify (`sh docs/port/verify.sh M4-windows-a11y` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./a11y/windows/... || exit 1; done && test -z "$(gofmt -l a11y/windows)"
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-web-wgpu-js

- Goal: Make render/wgpu work in the browser.
- Tier: complex. Risky: no.
- Depends on: M3-exit
- Owns: `render/wgpu/`, `docs/port/parity/wgpu.md`
- Core task: The `_js.go` side of render/wgpu on oliverbestmann/webgpu's GOOS=js backend: the web parts of wgpu_context.rs (WebBackendPreference, PreparedWebGraphics, async adapter / device acquisition bridged to gpui tasks), canvas surface creation and configuration, the shader variant selection for the web. shaders_webgl.wgsl stays embedded; the WebGL fallback itself has no backend in the chosen library, so those rows are `blocked:owner` (DESIGN 14.3). Shared files may only change in ways that keep the Linux build and tests identical.
- Verify (`sh docs/port/verify.sh M4-web-wgpu-js` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./render/... && go build ./render/... && go test ./render/... && test -z "$(gofmt -l render)"
```

### M4-web-platform

- Goal: Port the web platform, window, display, dispatcher and HTTP client.
- Tier: complex. Risky: no.
- Depends on: M4-web-wgpu-js
- Owns: `platform/web/`, `docs/port/parity/web.md`
- Core task: platform/web (`//go:build js && wasm`, syscall/js): platform.rs (846), window.rs (1054: canvas window, resize observer, device pixel ratio, frame requests), display.rs, dispatcher.rs (425: single-threaded mode only, timers and microtasks; the multi-threaded mode rows are `blocked:owner`), viewport.rs (268), keyboard.rs, logging.rs, gpui_web.rs, http_client.rs (312: fetch behind gpui.HTTPClient). js.Func values are released explicitly.
- Verify (`sh docs/port/verify.sh M4-web-platform` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./platform/web/... && test -z "$(gofmt -l platform/web)"
```

### M4-web-input-text

- Goal: Port web events, the IME mirror and the web text system; select the web platform.
- Tier: complex. Risky: no.
- Depends on: M4-web-platform
- Owns: `platform/web/`, `docs/port/parity/web.md`, `platform/platform_js.go`
- Core task: events.rs (1516, 2 tests: DOM pointer / wheel / keyboard / composition events to gpui.PlatformInput), ime_mirror.rs (618), text_system.rs (509: gotext.NewTextSystemWithoutSystemFonts plus canvas text), canvas_text.rs (197), canvas_fallback.rs (490, 7 tests), glyph_cache.rs (133, 3 tests), run_replacements.rs (594, 9 tests). platform/platform_js.go now returns the web platform. Tests run under node through go_js_wasm_exec; anything that needs a DOM is written against a small interface so the logic tests do not.
- Verify (`sh docs/port/verify.sh M4-web-input-text` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./platform/... && GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/... && test -z "$(gofmt -l platform/web platform/platform_js.go)"
go run ./internal/cmd/paritycheck -area web -done gpui_web/src/
```

### M4-web-example

- Goal: Port the hello_web example and its test drivers.
- Tier: standard. Risky: no.
- Depends on: M4-web-input-text
- Owns: `examples/web_hello_web/`, `docs/port/parity/web.md`
- Core task: `_ref/gpui_web/examples/hello_web`: main.rs, dynamic_fonts.rs, font_fallback_test.rs, http_client_test.rs, queue_test.rs as Go files (the three *_test.rs are runtime self-tests, hence the `_testing.go` names), index.html and gallery.js adapted to load a Go wasm binary with wasm_exec.js, the test_*.mjs drivers adapted likewise. Build the wasm binary; run the node drivers that do not need WebGPU. Opening it in a browser is an owner check: add it to docs/port/OWNER-CHECKS.md, do not launch a browser.
- Verify (`sh docs/port/verify.sh M4-web-example` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./examples/web_hello_web && GOOS=js GOARCH=wasm go build -o /tmp/hello_web.wasm ./examples/web_hello_web && test -z "$(gofmt -l examples/web_hello_web)"
go run ./internal/cmd/paritycheck -area web -all-done -allow-owner
```

### M4-ios

- Goal: Port the iOS platform and its example as far as GOOS=ios allows.
- Tier: complex. Risky: no.
- Depends on: M4-apple-metal-renderer, M3-exit
- Owns: `platform/ios/`, `platform/platform_ios.go`, `examples/ios_ios/`, `docs/port/parity/ios.md`
- Core task: platform/ios (`//go:build ios`) on platform/apple: ios/application.rs (543, 6 tests), display.rs, events.rs (128: touches to gpui input), platform.rs (519), text_input.rs (1111, 2 tests: UITextInput), window.rs (1105, 4 tests: UIWindow / UIView with CAMetalLayer and CADisplayLink), ios.rs, gpui_ios.rs, tests/text_input.rs. examples/ios_ios with the Xcode wrapper from `_ref/gpui_ios/examples/ios/` adapted to a Go archive. Pure logic tests are written so they also compile and run on darwin in CI. What Go cannot do on iOS is recorded per row as `blocked:owner` with the reason.
- Verify (`sh docs/port/verify.sh M4-ios` runs exactly these, from the worktree root):

```sh
GOOS=ios GOARCH=arm64 CGO_ENABLED=0 go vet ./platform/ios/... ./platform && test -z "$(gofmt -l platform/ios examples/ios_ios)"
go run ./internal/cmd/paritycheck -area ios
```

### M4-ci-macos

- Goal: Select the macOS platform, push, and make the macOS CI job pass.
- Tier: complex. Risky: yes.
- Depends on: M3-exit, M4-macos-a11y, M4-ios
- Owns: `platform/platform_darwin.go`, `platform/apple/ (fixes only)`, `platform/macos/ (fixes only)`, `platform/ios/ (fixes only)`, `a11y/macos/ (fixes only)`, `docs/port/OWNER-CHECKS.md`
- Core task: platform/platform_darwin.go returns MacPlatform (and the Metal headless renderer for tests). Commit the Apple packages (first time they are pushed), push, then loop on the macos-14 job with `gh run view --log-failed`: vet, the platform/apple and platform/macos tests (selector and struct smoke tests, text system, atlas, shader compile, keychain, pasteboard), root tests on macOS, and `go build ./examples/...`. Fix in the Apple packages; root or shared-package problems go to docs/port/handoff/M4-ci-macos.md with a repro unless the fix is local to a darwin file. Wait for CI in the background, never in a foreground sleep. List in docs/port/OWNER-CHECKS.md what only a person with a Mac / iPhone simulator can confirm (hello_world on screen, IME, native views, VoiceOver).
- Verify (`sh docs/port/verify.sh M4-ci-macos` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./... || exit 1; done && test -z "$(gofmt -l platform/apple platform/macos platform/ios a11y/macos platform/platform_darwin.go)"
gh pr checks feat/port
```

### M4-ci-windows

- Goal: Select the Windows platform, push, and make the Windows CI job pass.
- Tier: complex. Risky: yes.
- Depends on: M4-ci-macos, M4-windows-a11y
- Owns: `platform/platform_windows.go`, `platform/windows/ (fixes only)`, `a11y/windows/ (fixes only)`, `docs/port/OWNER-CHECKS.md`
- Core task: platform/platform_windows.go returns WindowsPlatform. Commit the Windows packages (first time they are pushed), push, loop on the windows-latest job as in M4-ci-macos: vet, platform/windows tests (struct sizes, COM helpers, DirectWrite, atlas, HLSL compile on WARP, dialogs, vsync), root tests on Windows, `go build ./examples/...`. Add the Windows owner checks (hello_world on screen, IME, DPI changes, Narrator) to docs/port/OWNER-CHECKS.md.
- Verify (`sh docs/port/verify.sh M4-ci-windows` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./... || exit 1; done && test -z "$(gofmt -l platform/windows a11y/windows platform/platform_windows.go)"
gh pr checks feat/port
```

### M4-exit

- Goal: Final audit: the definition of done holds or every gap is listed.
- Tier: complex. Risky: yes.
- Depends on: M4-ci-windows, M4-web-example
- Owns: `docs/port/PARITY.md`, `docs/port/parity/`, `TECH_DEBT.md`, `AGENT.md`, `README.md`, `docs/port/OWNER-CHECKS.md`, `docs/port/STAGES.md`
- Core task: Run the Final verify list. `paritycheck -all-done -rescan -lint` must pass; rows still `blocked:owner` are listed in PARITY.md section 5 under 'awaiting owner' with upstream locations, not given L- ids. PARITY.md section 2 gets done / deviation / n/a counts per area; sections 5-7 match reality (limits are only L-GLYPH, L-JPEG, L-SVGTEXT plus whatever the owner accepted). Fold all debt files into TECH_DEBT.md, bring AGENT.md up to date, write README.md (what it is, status per platform, how to run an example; run the de-slop skill on it). docs/port/OWNER-CHECKS.md lists every `YOU:` item. Commit, push; CI green on the three runners.
- Verify (`sh docs/port/verify.sh M4-exit` runs exactly these, from the worktree root):

```sh
test -z "$(gofmt -l .)" && go vet ./... && go test ./...
s=$(find . -name '*_gen.go' | sort | xargs sha256sum) && go generate ./... && [ "$s" = "$(find . -name '*_gen.go' | sort | xargs sha256sum)" ]
go run ./internal/cmd/paritycheck -all-done -rescan -lint
gh pr checks feat/port
```
