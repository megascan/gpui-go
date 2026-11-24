# Stage plan

Build plan for the gpui-go port: 130 stages, grouped by milestone, for at most three builders working at the same time in this worktree (`/home/strange/Projects/gpui-go-wt/port`, branch `feat/port`). Design: [DESIGN.md](DESIGN.md). Item and test mapping: [PARITY.md](PARITY.md). Library choices: [RESEARCH-deps.md](RESEARCH-deps.md).

Nothing is built yet. Every stage below lists what it owns, what it needs, its task, and the commands that check only its own files. Sizes in parentheses are upstream lines of Rust and upstream test counts.

## How the plan is laid out

- A stage fits one builder: about 3.6k upstream lines at most (plus its tests), usually 2-3.5k. Standard-tier stages stay under about 100K tokens of work; every stage stays under about 150 tool calls.
- `dependsOn` lists a stage when this stage reads code or a file format that stage creates, or when both own a path (two stages share an owned path only when one depends on the other).
- The root package `gpui` is about 100k upstream lines in one Go package. Two builders editing one package break each other's build, so its 45 stages form a single chain (the *root lane*). The other lanes take every other package.
- Exit stages (`M0-exit`, `M1-exit`, `M2-exit`, `M3-exit`, `M4-ci-macos`, `M4-ci-windows`, `M4-exit`) are links of the root chain. They are the only stages with authority to fix code across packages, and the only ones that commit and push.
- When more stages are ready than there are free builders, take the highest Prio in the Stage index (longest chain to M4-exit), ties by order in this file; at most one core stage runs at a time.
- Each stage's verify commands are listed in its section and collected in `docs/port/verify.sh`; `sh docs/port/verify.sh <stage>` runs them from the worktree root and stops at the first failure. `sh docs/port/verify.sh final` runs the Final verify list.

Root lane order: M0-geometry -> M0-geometry-units -> M0-contracts -> M0-platform-types -> M0-platform-data -> M0-input-types -> M0-scene -> M0-text-system -> M0-line-wrapper -> M0-profiler-core -> M0-entities -> M0-executors -> M0-app-core -> M0-app-effects -> M0-app-contexts -> M0-test-platform -> M0-test-contexts -> M0-keymap -> M0-gestures-spring -> M0-style -> M0-element -> M0-window-core -> M0-window-paint -> M0-window-dispatch -> M0-div-api -> M0-div-paint -> M0-text-elements -> M0-window-tests -> M0-blocked-tests -> M0-elements-basic -> M0-list -> M0-elements-media -> M0-window-a11y -> M0-exit -> M1-exit -> M2-exit -> M3-profiler-journal -> M3-bench-context -> M3-randomized-tree -> M3-profiler-hang -> M3-root-capture -> M3-exit -> M4-ci-macos -> M4-ci-windows -> M4-exit.

What the other lanes must deliver for the root lane, in the order it needs them: `M0-foundation` (before `M0-geometry`), `M0-scheduler-core` and `M0-a11y-schema` (before `M0-contracts`), `M0-jsonschema` and `M0-actiongen` (before `M0-input-types`), `M0-lyon` (before `M0-scene`), `M0-scheduler-test` (before `M0-entities`), `M0-stylegen` (before `M0-style`), the taffy chain (before `M0-element`), `M0-sumtree` (before `M0-list`), every Linux, web and screen-capture stage (before `M1-exit`), the examples, AT-SPI, benches, web example and iOS (before `M3-exit`).

What the root lane unlocks for the others: `M0-platform-data` -> `M1-linux-dbus`; `M0-scene` -> wgpu renderer and the Metal / DirectX atlases; `M0-text-system` -> `text/gotext` and the Apple lane; `M0-executors` -> the Windows lane; `M0-test-contexts` -> Linux platform; `M0-keymap` -> macOS, Windows and web platforms; `M0-window-a11y` -> macOS native views. The Apple and Windows packages are checked locally by cross-vet plus `go test -c`, and run their tests in CI.

## Shared decisions

Builders cannot see each other's work. These rules are what keeps their output compatible.

1. Read first, in this order: your stage's section in docs/port/STAGES.md (the full brief; the stage record handed to you is its summary), docs/port/DESIGN.md sections 5, 6 and 9, docs/port/PARITY.md section 3, and docs/api-contracts.md once it exists. They are binding. Where STAGES.md differs from DESIGN or PARITY, STAGES.md wins.
2. Port, do not redesign: upstream is zed @ 6501df15765d66d6c2a2b66b1eb22434ccef3f86. Keep upstream's names (PARITY rules N1-N14), order of operations, constants and panic messages. Re-express only what DESIGN 5.1-5.7 lists (D-NAME, D-RAII, D-OPT, D-VOID, D-EMIT, D-COND, D-INTO, D-GEOM, D-STRING, D-ASYNC, D-RNG, D-ARENA, D-TOKIO, D-HOTRELOAD, D-PROPTEST, D-BENCH, D-SCHEMA, D-FEATURE, D-HTTP, D-MACRO).
3. Reference sources are read-only: `_ref/<crate>/` for Zed; Rust dependencies under `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/<crate>-<version>/` (taffy-0.15.0, etagere-0.2.15, lyon_path-1.0.19, lyon_geom-1.0.20, tiny-skia-path-0.11.4, accesskit-0.24.1, accesskit_consumer-0.38.0, accesskit_atspi_common-0.19.1, accesskit_unix-0.22.1, accesskit_macos-0.26.3, accesskit_windows-0.34.0, cosmic-text-0.19.0, swash-0.2.10, zeno-0.3.3, zed-scap-0.0.8-zed, windows-capture-1.5.0, pipewire-0.10.1, wgpu-hal-30.0.1, naga-30.0.1, zed-xim-0.4.0-zed, xim-parser-0.2.2, xim-ctext-0.3.0, x11-clipboard-0.9.3, x11rb-protocol-0.13.2, xkbcommon-0.8.0, wayland-protocols-wlr-0.3.12, wayland-protocols-plasma-0.3.12); taffy's test suite in `_ext/taffy` (gitignored clone of tag v0.15.0); Wayland XML in /usr/share/wayland and /usr/share/wayland-protocols.
4. Layout: module `github.com/megascan/gpui-go`, Go 1.27; root package `gpui`; packages collections, gpuiutil, refineable, sumtree, scheduler, a11y (+ a11y/consumer, a11y/atspi, a11y/macos, a11y/windows), internal/{taffy,etagere,lyon,jsonschema,webglsl}, internal/cmd/{paritycheck,refinegen,actiongen,stylegen,wlgen,wlspike,shoot,metalhdr,glslgen}, internal/linux/{wl,wlproto,xkb,xcbshim,pw}, internal/apple/{objc,cf,gcd,ct,cg,mtl,appkit,uikit}, internal/win/{com,win32,dwrite,d3d,wgc,screencap}, render, render/wgpu (+ internal/gpu, internal/gpu/webgpu, internal/gpu/webgl2), text/gotext, platform, platform/linux (+ internal/common, internal/dbus, internal/screencap, wayland, x11), platform/apple, platform/macos, platform/ios, platform/windows, platform/web, examples/<name>. OS bindings that more than one package imports live under the repo-root internal/, because Go lets only the parent tree of an internal/ directory import it (platform/macos could not import platform/apple/internal/objc). Wrappers specific to one package stay in that package. One upstream file maps to one Go file (PARITY F1-F6).
5. The root package has one writer at a time: root stages form a chain through dependsOn. Forward references live in root `stubs.go`: a stage that ports a file deletes that file's symbols from stubs.go and may add panicking stubs (`panic("gpui: unimplemented <symbol>")`) with final signatures for symbols a later stage owns. Other packages only compile against stubs and never call them in tests. stubs.go is created by M0-geometry and deleted by M3-root-capture, the last root stage before M3-exit. internal/taffy and the platform packages use the same pattern with a package-local stubs.go.
6. A piece never shares a Go package with its stage's core task or with another piece, so pieces can be built at the same time without breaking each other's build. Stay inside your owned paths. A bug in a package you do not own: write a minimal failing repro in your own paths and an entry in docs/port/handoff/<stage>.md; exit stages (M0-exit, M1-exit, M2-exit, M3-exit, M4-ci-*, M4-exit) have fix authority and work through those files.
7. File rule exceptions (one upstream file written by two chained stages, or split across Go files): geometry.rs lines 1-2346 by M0-geometry and 2347-4165 by M0-geometry-units, both into geometry.go; platform.rs 1-2126 by M0-platform-types and 2127-4116 by M0-platform-data, both into platform.go; app.rs 1-1999 by M0-app-core and 2000-4031 by M0-app-effects, both into app.go; key_dispatch.rs code (1-620) by M0-keymap and its gpui::tests by M0-blocked-tests; window.rs into window.go (1-2561, 7891-8411), window_paint.go (2562-5700), window_dispatch.go (5700-7890), window_test.go; div.rs into elements_div.go (1-2297), elements_div_paint.go (2298-4539), elements_div_test.go; list.rs tests 1748-2930 into elements_list_test.go (M0-list) and 2931-3883 into elements_list_follow_test.go (M0-window-a11y); wayland/client.rs into client.go (1-1967) and client_input.go (1968-3454); macOS window.rs 1-2650 and 2651-4344 by two chained stages. Documentation goes under lowercase `docs/` (port tracking in docs/port/); never create `DOCS/`, the macOS and Windows runners have case-insensitive file systems. PARITY F6's `DOCS/` pages become docs/*.md.
8. go.mod and go.sum are owned by M0-bootstrap and already require every module in the Installs list. Builders never run `go get`, `go mod tidy` or `go install` and never add a dependency; if one is missing, stop and write it to docs/port/handoff/<stage>.md. Pinned: purego v0.11.3, oliverbestmann/webgpu v1.38.1 (wgpu-native v29.0.1.1), go-text/typesetting v0.3.5, xo/okage v0.20.1, jezek/xgb v1.3.1, jezek/xgbutil v0.0.0-20260124183602-9fd151d6a51a, godbus/dbus/v5 v5.2.4, x/image v0.48.0, x/sys v0.50.0, gen2brain/avif v0.6.0.
9. Parity bookkeeping: each stage edits only the area files in its owns list, row by row (never rewrite a whole file), in the same change as the code. Status is `todo`, `done`, `dev:<id>`, `n/a:<id>` or `blocked:<stage name>` (a test or item that needs a later stage). `blocked:owner` is not used: every item is ported, and what only a person can confirm (a device, a browser, an interactive dialog) is an owner check (decision 30) while the row is `done`. A row that uses an accepted limit cites it as `dev:L-<x>` (behaviour differs) or `n/a:L-<x>` (the item cannot exist). New deviation ids are registered in a `## Deviations registered here` table at the end of the stage's own area file, not in PARITY.md. Items the scan missed get new rows. `go run ./internal/cmd/paritycheck -area <a> [-done <upstream prefix>,...] [-all-done [-allow-blocked]] [-lint] [-rescan]` must pass before a stage is finished; `-allow-blocked` tolerates `blocked:<stage>` only while that stage has no docs/port/status/<stage>.done. External ports get docs/port/parity/ext-<name>.md created by their first stage.
10. AGENT.md, TECH_DEBT.md, docs/port/PARITY.md, docs/port/OWNER-CHECKS.md and .github/workflows/ci.yml are edited only by M0-bootstrap and the exit and CI stages. Every other stage writes hacks and known issues to docs/port/debt/<stage>.md at the moment it writes them, handoffs to docs/port/handoff/<stage>.md, owner checks to docs/port/owner-checks/<stage>.md, and on completion creates docs/port/status/<stage>.done with its verify result (one file per stage in shared directories, so there are no edit conflicts).
11. Go API shape (DESIGN 5-6): generic methods on concrete types (`Entity[T].Update`, `App.New`, `Context[T].Subscribe`); a generic method cannot be part of an interface, so AppContext is sealed with unexported non-generic primitives and the generic operations are free functions over it (`gpui.AppContextNew`, `AppContextReadGlobal`, ...) plus same-named generic methods on each concrete context type (DESIGN 6.2); closures returning a value use the `R` suffix (`Update` / `UpdateR`); `anyhow::Result<T>` -> `(T, error)` and upstream's `VisualContext::Result<T>` -> `error` / `(R, error)` (`UpdateIn`, `UpdateInR`), panics stay panics; returned `Option<T>` -> `(T, bool)` or a nil-able type, Option struct fields -> `gpui.Option[T]` (alias of gpuiutil.Option); iterators -> iter.Seq / iter.Seq2, `impl IntoIterator` parameters -> slices or variadics; field-less enums -> `type X uint8` with `XVariant` constants, data enums -> sealed interfaces or comparable tagged structs; `impl Into<X>` -> the concrete type, a sealed `IntoX` interface, or a union-constrained generic; TypeId -> reflect.Type; `gpui.Task[T]` is an alias of scheduler.Task[T].
12. Elements: `Element[RL, PP]` with typed state behind an unexported drawable, never `any`; optional hooks are separate interfaces the drawable type-asserts (`ElementA11y`, `ElementA11ySyntheticChildren[PP]`). Fluent traits are self-typed mixins embedded in each element (`Styled[S]`, `Interactive[S]`, `StatefulInteractive[S]`, `Parent[S]`, `Fluent[S]`) and bound in the element's constructor with the helpers documented in docs/api-contracts.md; chains keep the concrete type (`NewDiv().Flex().SizeFull()` is `*Div`). `Stateful[E]` is IntoElement-only and delegates to its inner element (DESIGN 6.5). SharedString implements IntoElement; plain strings go through `Parent.ChildText`. No element type may declare a method whose name is in docs/port/parity/styled-methods.tsv or the interactive method sets. `Element::id` is `ElementID`, constructors that collide with a type are `New<Type>`.
13. Handles and Drop (D-RAII): a handle value aliases one counted handle; `Clone()` makes a new one, `Release()` is the explicit drop (idempotent, use after release panics). Every upstream `impl Drop` has a row in the Drop table of DESIGN 5.2 naming its Go method (Release, Close, Destroy, End, Cancel, Unsubscribe) and when it runs; ported code calls it exactly where upstream drops, moves into `detach()`, or lets a value go out of scope, usually with `defer`. The App borrow guard (AppCell, DESIGN 5.2) panics on re-entrant mutable borrows with upstream's message. Subscriptions and tasks never stop because of garbage collection; runtime.AddCleanup is only the safety net for entity and focus handles and the leak reporter. Tests in this repo run with strict leak detection.
14. Async (D-ASYNC): tasks are iter.Pull coroutines scheduled by the scheduler package in tests and production; `.await` is `task.Await(cx)`; a foreground task holds `*AsyncApp`, not `*App`, and blocks only in gpui / scheduler primitives (Await, Timer, Chan, Oneshot, Select, Yield), never on a raw channel, mutex wait or time.Sleep. iter.Pull's next and stop must not run concurrently or re-entrantly, so `Cancel` only sets an atomic flag and wakes the task; the poll loop that owns the coroutine calls stop() on its next poll (DESIGN 5.6). Test randomness comes from the scheduler's seeded PCG (D-RNG).
15. Actions: an action type implements `Name`, `BoxedClone`, `PartialEq` and registers itself with `gpui.RegisterAction` from `init` (by hand or with `//gpui:action`, actiongen). JSON goes through encoding/json/v2 (`encoding/json/v2`, `encoding/json/jsontext`) everywhere in the port; schemas through internal/jsonschema.
16. Tests: every upstream test is one Go test named by rule N14 (`TestKeymap_Foo`, inner module name after the file stem), in `<file>_test.go` of the same package; `#[gpui::test]` bodies run inside `gpui.RunTest` with upstream's seeds / iterations / retries; `#[ignore]` becomes t.Skip with upstream's reason; cfg(target_os) gates become build constraints; benches become `Benchmark<File>_<Name>` on gpui.RunBench. Fix the code, not the test, when a ported test fails. Test support (TestAppContext, TestPlatform) lives in ordinary files of the root package.
17. Test isolation on the developer's machine: `go test` never opens a window, never talks to the session bus, the a11y bus, the keyring, the notification daemon or the system clipboard, and never starts or stops user services. Tests that need a display run only with GPUI_TEST_DISPLAY=1; D-Bus tests use a fake connection or a private dbus-daemon started by the test, and programs that need a bus run under `dbus-run-session --`; credential tests use unique names and delete only what they created; GPU tests skip when no adapter is found.
18. Generated code is committed, named `*_gen.go` (or the tool's documented data files), starts with `// Code generated by <tool>; DO NOT EDIT.`, and is produced by a `//go:generate go run github.com/megascan/gpui-go/internal/cmd/<tool> ...` line in a hand-written file of the same package; `go generate ./...` must leave every generated file byte-identical (verify with checksums, not `git diff`: the tree is not committed between stages). Tools that wrap an external program (metalhdr: cbindgen, glslgen: naga) exit 0 with a message when it is missing, so the committed output is used. Directives are `//gpui:refineable` and `//gpui:action`.
19. Build rules: the root package imports no platform, renderer, cgo or purego code (DESIGN section 4 import rules, enforced by `paritycheck -lint`). cgo is allowed only in render/wgpu (constrained to `linux || js`) and in the dev tool internal/cmd/wlspike. internal/linux/** is `linux`, internal/apple/** `darwin`, internal/win/** `windows`, and only packages with the same constraint import them. Linux packages are `linux`, Apple packages `darwin` (macos: `darwin && !ios`, ios: `ios`), platform/windows `windows`, platform/web `js && wasm`. Everything except render/wgpu's cgo files must pass `CGO_ENABLED=0 go vet` and `go test -c -o /dev/null` for darwin, windows and js/wasm from Linux (ios: vet only, it needs external linking). `runtime.LockOSThread()` is called in `init` of package platform only; the root package never locks a thread. File stems never end in a GOOS / GOARCH word by accident (F3).
20. Seams between packages: render/render.go (the `Renderer` interface, created by M0-scene) is what platform windows hold; Linux constructs it with `wgpu.NewRenderer`, macOS and Windows with their own renderers. render/wgpu is written against its GPU facade render/wgpu/internal/gpu, implemented by internal/gpu/webgpu (native and browser WebGPU) and internal/gpu/webgl2 (the browser fallback). All atlases use internal/etagere. The Linux and web text system is text/gotext (`NewTextSystem`, `NewTextSystemWithoutSystemFonts`). Screen capture: the root declares `ScapBackend` and `RegisterScapBackend` (M0-platform-types) and ports upstream's scap glue on it (M3-root-capture); the Linux back end registers from platform/linux, the Windows one from platform/platform_windows.go; macOS uses its own ScreenCaptureKit path as upstream does. Conversions between gpui types and internal/taffy, internal/etagere or internal/lyon types live in the bridge files (taffy.go, path_builder.go, the atlas implementations), never in geometry.go, style.go or platform.go. Each platform window package has an `a11y.go` seam that the adapter stages fill.
21. GPU data: scene primitive structs keep upstream field order with fixed-size 4-byte-aligned fields and are uploaded with unsafe.Slice; scene_layout_test.go pins sizes and offsets. Shader files (4 WGSL, shaders.metal, shaders.hlsl, alpha_correction.hlsl) are byte-identical copies embedded with //go:embed and checked by sha256 against `_ref`. Metal compiles the cbindgen scene.h header (internal/cmd/metalhdr, cbindgen 0.28.0, as gpui_apple/build.rs) followed by shaders.metal, as upstream's runtime_shaders path does; HLSL's `#include "alpha_correction.hlsl"` is served by an ID3DInclude handler; the WebGL2 path uses GLSL ES 3.00 generated from the WGSL by naga (internal/cmd/glslgen).
22. Code style: gofmt; no comments in code except terse godoc on exported API (first sentence starts with the symbol name), `//go:` and `//gpui:` directives and generated-file headers; no per-file licence headers (LICENSE + NOTICE cover it); errors wrapped with `%w`; no new third-party dependency; no `init` side effects other than registration and the platform thread lock.
23. Verification: a stage is finished when `sh /home/strange/Projects/gpui-go-wt/port/docs/port/verify.sh <stage>` exits 0. It runs exactly the commands in the stage's Verify block in STAGES.md (build, vet, test and gofmt on the stage's own packages, `go test -c -o /dev/null` for other targets, plus paritycheck), never a whole-repo build except in exit stages. Build outputs go to /dev/null, never /tmp. Builders do not edit verify.sh or STAGES.md; an exit stage may, changing both together.
24. Running things: shell commands start with `cd /home/strange/Projects/gpui-go-wt/port &&` (the working directory resets between calls). Anything that can take more than about 20 seconds (full test runs, cgo builds of render/wgpu, CI waits) runs in the background; never sleep-poll in the foreground. Run examples and spikes only through internal/cmd/shoot or with `timeout`, at low priority, and leave no window, process or headless output behind.
25. Screenshots: internal/cmd/shoot (built in M1-exit) is the only way examples are captured; images go to docs/port/screenshots/<group>/<example>.png. `-may-exit` accepts a program that exits 0 before a window appears; `-no-window <duration>` is for programs that never open one (success if alive at the deadline or exited 0). On Hyprland shoot never shows the example to the owner: a `hyprctl eval` Lua hook on `window.open_early` moves every window of the child's process group, unfocused and floating, to a hidden `special:gpui-shoot-<pid>` workspace and resizes it to `-size` (default 800x600 logical); the capture is `grim -T <stableId>` of the toplevel, so pixel size is the logical size times the scale of the monitor that had focus. Open every image with the Read tool and check it for missing content, clipping, blur and wrong framing before it is recorded as verified. What cannot be checked on this machine is an owner check (decision 30); it is never claimed as verified.
26. Git: branch feat/port, base dev, no Co-Authored-By lines. Only the exit and CI stages commit and push, and only the paths of stages that have a docs/port/status/<stage>.done file (skip if the harness already committed them); never `git add -A`, never force-push. M0-exit opens the draft PR so the macOS and Windows runners compile the root and leaf packages from then on. Exits before M4 never commit internal/apple, internal/win, platform/apple, platform/macos, platform/ios, platform/windows, a11y/macos, a11y/windows or examples/ios_ios, even when those stages are done: the Apple and iOS paths are first committed by M4-ci-macos, the Windows paths by M4-ci-windows, so an unfinished platform never turns an earlier exit's CI red. Web stages are committed like Linux code by the first exit after they are done (the web renderer files before M1-exit, which depends on them; platform/web and the web example by M3-exit at the latest).
27. Scheduling: at most three stages run at once and at most one core stage at a time. When more are ready than there are free builders, take the highest Prio in the Stage index (the length of the longest dependency chain from the stage to M4-exit), ties by order in this file. The root chain is the critical path; the other lanes feed it.
28. Owner decisions (DESIGN 14, all decided): (1) Wayland goes through libwayland-client with purego, with typed wrappers from the in-repo generator wlgen; (2) D-Bus is github.com/godbus/dbus/v5 v5.2.4; (3) the WebGL fallback is ported (M4-web-webgl-shaders, M4-web-webgl); wgpu-native v29 against upstream's wgpu 30 is the accepted limit L-WGPUVER with a TECH_DEBT entry to bump the binding; the multi-threaded web dispatcher is the accepted limit L-WEBTHREADS and every shared path follows upstream's single-threaded branch; (4) action JSON schema comes from internal/jsonschema; (5) screen capture is ported: Linux through the xdg ScreenCast portal plus PipeWire (purego over libpipewire-0.3) and X11 capture, Windows through Windows.Graphics.Capture, macOS through ScreenCaptureKit as upstream, plus visual_test.rs. Accessibility adapters are native Go ports of accesskit_consumer / atspi / macos / windows.
29. Known parity limits are only L-GLYPH (Linux glyph pixels vs swash), L-JPEG (JPEG decode values), L-SVGTEXT (text and embedded images inside SVG), L-WGPUVER (wgpu-native v29 binding against upstream's wgpu 30: missing EnumerateAdapters, DUAL_SOURCE_BLENDING or TRANSIENT_ATTACHMENT use the fallbacks of M1-wgpu-context) and L-WEBTHREADS (Go wasm has no threads, so the multi-threaded web dispatcher is absent); each row using one cites the id. Licence: Apache-2.0 LICENSE plus NOTICE crediting Zed Industries (GPUI is Apache-2.0).
30. Owner checks: what only a person can confirm (macOS, Windows or iOS on screen, a browser, an interactive portal dialog, VoiceOver / Narrator) is written to docs/port/owner-checks/<stage>.md as `YOU: <what to check, where>`; exit and CI stages fold those files into docs/port/OWNER-CHECKS.md. An owner check never leaves a row `blocked`.

## Installs

Run once by the orchestrator before any stage (builders may not install). They pin every Go module the port uses, fetch taffy's test suite (not in the published crate), and install cbindgen (Metal scene.h, M4-apple-metal-atlas) and naga-cli (WGSL to GLSL ES, M4-web-webgl-shaders) at the versions upstream uses. zed-scap, windows-capture, zeno, swash, wgpu-hal and naga sources are already in the cargo registry.

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
cargo install cbindgen --version 0.28.0 --locked
cargo install naga-cli --git https://github.com/gfx-rs/wgpu --rev cddb2f5d80dc0009b2ef6767635140cacf90a007 --locked
```

## Parity file split

Done by `M0-bootstrap` (piece `parity-split`) so that stages running at the same time never edit the same area file. Rows move verbatim under their upstream-file headings.

| Current file | New file | Upstream files | Edited by |
|---|---|---|---|
| `parity/linux.md` | `parity/linux-common.md` | linux/dispatcher.rs, headless.rs, headless/window.rs, keyboard.rs, platform.rs, text_system.rs | M1-linux-common, then M1-linux-platform (Platform impl and display-mode test rows), M2-exit (text_system.rs) |
| `parity/linux.md` | `parity/linux-dbus.md` | linux/power.rs, system_notifications.rs, xdg_desktop_portal.rs | M1-linux-dbus (the Secret Service rows of platform.rs stay in linux-common.md and are marked done by M1-linux-dbus's package through M1-linux-common) |
| `parity/linux.md` | `parity/linux-platform.md` | gpui_linux.rs, linux.rs, linux/display_connection.rs | M1-linux-platform |
| `parity/linux.md` | `parity/linux-wayland.md` | linux/wayland.rs, linux/wayland/* | M1-wayland-client, M1-wayland-input, M1-wayland-window |
| `parity/linux.md` | `parity/linux-x11.md` | linux/x11.rs, linux/x11/* | M1-x11-client, M1-x11-window |
| `parity/wgpu.md` | `parity/wgpu.md` | gpui_wgpu.rs, wgpu_atlas.rs, wgpu_context.rs, wgpu_renderer.rs | M1-wgpu-context, M1-wgpu-renderer, M4-web-wgpu-js, M4-web-webgl |
| `parity/wgpu.md` | `parity/gotext.md` | cosmic_text_system.rs, benches/layout_line.rs (plus rows for the ported zeno and swash files) | M2-gotext-layout, M2-gotext-raster, M2-gotext-color |
| `parity/core-examples.md` | `parity/examples-m2.md` | examples/hello_world.rs, text.rs, svg/, image/, example_support/fonts.rs | M1-exit, M2-exit |
| `parity/core-examples.md` | `parity/examples-a.md` | active_state_bug, anchor, grid_layout, opacity, gradient, pattern, scrollable, tree, uniform_list, list_example, data_table, paths_bench, popover | M3-examples-a |
| `parity/core-examples.md` | `parity/examples-b.md` | text_layout, text_wrapper, focus_visible, tab_stop, drag_drop, mouse_pressure, a11y, svg_animation, ownership_post, shadow, painting | M3-examples-b |
| `parity/core-examples.md` | `parity/examples-c.md` | window, window_movable, window_positioning, window_shadow, layer_shell, on_window_close_quit, move_entity_between_windows, set_menus, system_notifications, toggle_headless, gif_viewer, image_gallery, image_loading, native_view, native_view_appkit_field | M3-examples-c |
| `parity/core-examples.md` | `parity/examples-d.md` | view_example, testing, input, animation | M3-examples-d |
| `parity/core-examples.md` | `parity/core-tests.md` | gpui/build.rs, benches/focus_cleanup.rs, tests/action_macros.rs, tests/eq_hash.rs | M0-exit, M3-bench-context, M3-randomized-tree |

Unchanged files and their editors: `foundation.md` (M0-foundation, M0-geometry), `sumtree.md`, `scheduler.md`, `macros.md` and all `core-*.md` (root lane only), `platform-select.md` (M1-linux-platform, M3-platform-benches), `apple.md`, `macos.md`, `ios.md`, `windows.md`, `web.md` (one lane each). New files: `ext-taffy.md`, `ext-etagere.md`, `ext-lyon.md`, `ext-a11y.md`, `ext-a11y-consumer.md`, `ext-a11y-atspi.md`, `ext-a11y-macos.md`, `ext-a11y-windows.md`, `ext-scap-linux.md`, `ext-scap-windows.md`, `_rescan.md`.

## Final verify

```sh
cd /home/strange/Projects/gpui-go-wt/port && test -z "$(gofmt -l .)"
cd /home/strange/Projects/gpui-go-wt/port && go vet ./...
cd /home/strange/Projects/gpui-go-wt/port && go test ./...
cd /home/strange/Projects/gpui-go-wt/port && go test -race $(go list ./... | grep -vE '/(examples|render/wgpu|internal/cmd/wlspike)(/|$)|/platform/linux/(wayland|x11)(/|$)')
cd /home/strange/Projects/gpui-go-wt/port && go test -tags gpui_profiler .
cd /home/strange/Projects/gpui-go-wt/port && s=$(find . -name '*_gen.go' | sort | xargs sha256sum) && go generate ./... && [ "$s" = "$(find . -name '*_gen.go' | sort | xargs sha256sum)" ]
cd /home/strange/Projects/gpui-go-wt/port && for t in darwin/arm64 darwin/amd64 windows/amd64 windows/arm64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet ./... && GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go test -c -o /dev/null ./... || exit 1; done
cd /home/strange/Projects/gpui-go-wt/port && GOOS=ios GOARCH=arm64 CGO_ENABLED=0 go vet ./internal/apple/... ./platform/apple/... ./platform/ios/... ./platform
cd /home/strange/Projects/gpui-go-wt/port && GOOS=js GOARCH=wasm go build -o /dev/null ./examples/web_hello_web && GOOS=js GOARCH=wasm go build ./render/... ./platform/web/...
cd /home/strange/Projects/gpui-go-wt/port && env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOMODCACHE="$(go env GOMODCACHE)" GOFLAGS="$GOFLAGS" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/... ./render/wgpu/internal/gpu/webgl2/...
cd /home/strange/Projects/gpui-go-wt/port && go build -o /dev/null ./examples/...
cd /home/strange/Projects/gpui-go-wt/port && go run ./internal/cmd/paritycheck -all-done -rescan -lint
cd /home/strange/Projects/gpui-go-wt/port && test ! -e stubs.go && test ! -e DOCS
cd /home/strange/Projects/gpui-go-wt/port && gh pr checks feat/port
```

## Review log

External review (Gemini), each item checked against the source before deciding. Applied 20, rejected 1.

1. internal/ visibility: applied. Go only lets the parent tree of internal/ import it, so platform/macos, platform/ios and a11y/macos could not use platform/apple/internal, nor internal/cmd/wlspike platform/linux/internal/wl. Shared bindings moved to repo-root internal/apple, internal/win, internal/linux (decision 4).
2. Stateful[E]: applied. div.rs:4176-4279 delegates through E::RequestLayoutState / E::PrepaintState, which Go cannot name; Stateful[E] is IntoElement-only and delegates to the inner element (M0-div-paint, DESIGN 6.5, D-COND).
3. Shaders: applied. gpui_apple/build.rs:28-120 stitches cbindgen's scene.h before shaders.metal; internal/cmd/metalhdr runs cbindgen 0.28.0 (M4-apple-metal-atlas). shaders.hlsl line 1 includes alpha_correction.hlsl; an ID3DInclude serves it (M4-windows-d3d-atlas). Shader files stay byte-identical.
4. iter.Pull stop() safety: applied. Cancel sets an atomic flag and wakes; only the owning poll loop calls stop() (M0-scheduler-core, decision 14, DESIGN 5.6).
5. ElementA11ySyntheticChildren[PP]: applied. element.rs:132-137 passes &mut PrepaintState; the hook is a typed optional interface the drawable asserts (M0-contracts, M0-element).
6. UpdateIn returns error: applied. gpui.rs:265 VisualContext::Result<T>; UpdateIn returns error and UpdateInR (R, error) (M0-contracts, decision 11).
7. CI guards and parity gating: applied. Web steps run inside `if [ -d ... ]`; CI runs paritycheck -lint until M4-exit switches it to -all-done; -allow-blocked tolerates blocked:<stage> until that stage is done; blocked:owner is removed (target zero) by scheduling WebGL, screen capture and full iOS, with owner checks instead (decisions 9, 30).
8. Scheduling: applied. render/render.go is created by M0-scene; M4-apple-ffi depends on M0-text-system and M4-windows-com on M0-executors (its dispatcher needs queue.rs's PriorityQueueSender), not M0-exit; the atlas stages depend on M0-scene and M0-etagere, not M1-wgpu-context; M0-stylegen and M0-actiongen are early standard stages; idle slots are filled by priority (decision 27).
9. Move gestures, spring, bounds_tree, path_winding, journal into internal/ packages: rejected. Each imports root types (gestures: PlatformInput, TouchEvent, ScrollEvent, Pixels, Point; spring: Hsla, Pixels, Rems, Rgba; bounds_tree: Bounds, PrimitiveKind; path_winding: Bounds, PathCurve, PathTile, Point, ScaledPixels; journal: App, WindowId, WindowVisibility, FrameRequestSource), so an internal package would import gpui while gpui imports it: an import cycle. It would also break PARITY F1 (one file, one Go file in its package). The root chain is shortened by splitting instead (item 10).
10. Split oversized stages: applied. Every stage is now at most about 3.6k upstream lines: geometry, platform, app, keymap/key_dispatch, test platform, executors, line wrapper, list tests, wayland client, linux dbus, atspi, macOS AppKit/display/window, iOS, Windows events/shell/window/platform/a11y, web input/text, the gotext raster and the taffy chain were split (130 stages).
11. AppContext primitives, non-App contexts, Drop audit: applied. Sealed primitives readGlobal, withWindow, readWindow, asMut, backgroundExecutor added; non-App contexts expose upstream-named generic methods delegating to them (DESIGN 6.2). Audit: 76 `impl Drop` across the ported crates, 31 in 15 gpui/src files (DESIGN said 13); each has a row in the new DESIGN 5.2 Drop table.
12. wgpu-native v29 features: applied. M1-wgpu-context probes EnumerateAdapters (wgpu_context.rs:215, 531), DUAL_SOURCE_BLENDING (:442-446) and TRANSIENT_ATTACHMENT (wgpu_renderer.rs:1571) with fallbacks (RequestAdapter + compositor hint, the non-subpixel path, RENDER_ATTACHMENT) cited as L-WGPUVER.
13. system_notifications verify loop: applied. It runs under `dbus-run-session --` so it never reaches the user's notification daemon (M3-examples-c).
14. ownership_post verify loop: applied. It opens no window and never quits (ownership_post.rs:20-47); shoot gains `-no-window <duration>` (M1-exit) and M3-examples-b uses it.
15. X11 surface ordering: applied. Round trip (GetInputFocus) after creating and mapping the window, before creating the surface on the xcbshim connection; teardown destroys renderer and surface before the window and before xcb_disconnect (M1-x11-window, M1-linux-ffi).
16. M1-linux-dbus split, LCD subpixel, colour emoji: applied, the LCD part in corrected form. D-Bus is its own stage. zeno 0.3.3's CustomSubpixel([0.3, 0, -0.3]) is three offset coverage passes into channels 0/1/2, not 3x oversampling plus a filter; it is ported as zeno does it. Emoji follow swash 0.2.10's source order: COLR v0 outlines, then CBDT/sbix bitmaps (BestFit), then outline (M2-gotext-raster, M2-gotext-color).
17. Verification: applied. Final verify and the exits run `go test -race` over the non-display packages, cross-OS `go test -c -o /dev/null` for darwin, windows and js, and the wasm build covers render/... and render/wgpu/internal/gpu/webgl2.
18. Borrow guard on App: applied. app.rs:85-183 AppCell borrow tracking is ported with its panic text, origin capture via runtime.Caller and TRACK_THREAD_BORROWS (M0-app-core, DESIGN 5.2).
19. SharedString IntoElement, ChildText: applied. SharedString implements IntoElement as upstream's impl Element for SharedString; Parent.ChildText takes Go strings (M0-text-elements, DESIGN 5.5, 6.5).
20. Doc drift: applied. DESIGN section 11 points to STAGES.md instead of the stale S0-S23 table, S-refs are gone, the Drop count is corrected, and the PARITY arc_cow row maps to `-` with n/a:D-STRING.
21. `-o /dev/null`: applied. Build outputs in verify commands go to /dev/null (M4-web-example and the cross-OS test -c commands; decision 23).

## Stage index

| # | Stage | Tier | Risky | Prio | Depends on |
|---|---|---|---|---|---|
| 1 | [M0-bootstrap](#m0-bootstrap) | core | yes | 47 | - |
| 2 | [M0-foundation](#m0-foundation) | complex | no | 46 | M0-bootstrap |
| 3 | [M0-scheduler-core](#m0-scheduler-core) | core | yes | 44 | M0-bootstrap |
| 4 | [M0-scheduler-test](#m0-scheduler-test) | complex | no | 36 | M0-scheduler-core |
| 5 | [M0-sumtree](#m0-sumtree) | complex | no | 16 | M0-bootstrap |
| 6 | [M0-a11y-geometry](#m0-a11y-geometry) | standard | no | 45 | M0-bootstrap |
| 7 | [M0-a11y-schema](#m0-a11y-schema) | complex | yes | 44 | M0-a11y-geometry |
| 8 | [M0-jsonschema](#m0-jsonschema) | standard | no | 41 | M0-bootstrap |
| 9 | [M0-etagere](#m0-etagere) | standard | no | 18 | M0-bootstrap |
| 10 | [M0-lyon](#m0-lyon) | complex | no | 40 | M0-bootstrap |
| 11 | [M0-parity-rescan](#m0-parity-rescan) | complex | no | 13 | M0-bootstrap |
| 12 | [M0-stylegen](#m0-stylegen) | complex | no | 27 | M0-bootstrap |
| 13 | [M0-actiongen](#m0-actiongen) | standard | no | 41 | M0-bootstrap |
| 14 | [M0-taffy-style](#m0-taffy-style) | core | no | 38 | M0-foundation |
| 15 | [M0-taffy-style-mod](#m0-taffy-style-mod) | complex | no | 37 | M0-taffy-style |
| 16 | [M0-taffy-style-grid](#m0-taffy-style-grid) | complex | no | 36 | M0-taffy-style-mod |
| 17 | [M0-taffy-tree](#m0-taffy-tree) | complex | no | 35 | M0-taffy-style-grid |
| 18 | [M0-taffy-compute](#m0-taffy-compute) | complex | no | 34 | M0-taffy-tree |
| 19 | [M0-taffy-fixtures](#m0-taffy-fixtures) | complex | no | 33 | M0-taffy-compute |
| 20 | [M0-taffy-flexbox](#m0-taffy-flexbox) | complex | no | 32 | M0-taffy-fixtures |
| 21 | [M0-taffy-block](#m0-taffy-block) | complex | no | 31 | M0-taffy-flexbox |
| 22 | [M0-taffy-grid-types](#m0-taffy-grid-types) | complex | no | 30 | M0-taffy-block |
| 23 | [M0-taffy-grid-placement](#m0-taffy-grid-placement) | complex | no | 29 | M0-taffy-grid-types |
| 24 | [M0-taffy-grid-sizing](#m0-taffy-grid-sizing) | complex | no | 28 | M0-taffy-grid-placement |
| 25 | [M0-taffy-handwritten-a](#m0-taffy-handwritten-a) | complex | no | 27 | M0-taffy-grid-sizing |
| 26 | [M0-taffy-handwritten-b](#m0-taffy-handwritten-b) | complex | no | 26 | M0-taffy-handwritten-a |
| 27 | [M0-geometry](#m0-geometry) | core | no | 45 | M0-foundation |
| 28 | [M0-geometry-units](#m0-geometry-units) | complex | no | 44 | M0-geometry |
| 29 | [M0-contracts](#m0-contracts) | core | yes | 43 | M0-geometry-units, M0-a11y-schema, M0-scheduler-core |
| 30 | [M0-platform-types](#m0-platform-types) | core | yes | 42 | M0-contracts |
| 31 | [M0-platform-data](#m0-platform-data) | complex | yes | 41 | M0-platform-types |
| 32 | [M0-input-types](#m0-input-types) | complex | yes | 40 | M0-platform-data, M0-jsonschema, M0-actiongen |
| 33 | [M0-scene](#m0-scene) | core | yes | 39 | M0-input-types, M0-lyon |
| 34 | [M0-text-system](#m0-text-system) | complex | no | 38 | M0-scene |
| 35 | [M0-line-wrapper](#m0-line-wrapper) | complex | no | 37 | M0-text-system |
| 36 | [M0-profiler-core](#m0-profiler-core) | complex | no | 36 | M0-line-wrapper |
| 37 | [M0-entities](#m0-entities) | core | no | 35 | M0-profiler-core, M0-scheduler-test |
| 38 | [M0-executors](#m0-executors) | complex | no | 34 | M0-entities |
| 39 | [M0-app-core](#m0-app-core) | core | no | 33 | M0-executors |
| 40 | [M0-app-effects](#m0-app-effects) | complex | no | 32 | M0-app-core |
| 41 | [M0-app-contexts](#m0-app-contexts) | complex | no | 31 | M0-app-effects |
| 42 | [M0-test-platform](#m0-test-platform) | complex | no | 30 | M0-app-contexts |
| 43 | [M0-test-contexts](#m0-test-contexts) | complex | no | 29 | M0-test-platform |
| 44 | [M0-keymap](#m0-keymap) | complex | no | 28 | M0-test-contexts |
| 45 | [M0-gestures-spring](#m0-gestures-spring) | complex | no | 27 | M0-keymap |
| 46 | [M0-style](#m0-style) | complex | no | 26 | M0-gestures-spring, M0-stylegen |
| 47 | [M0-element](#m0-element) | core | no | 25 | M0-style, M0-taffy-handwritten-b |
| 48 | [M0-window-core](#m0-window-core) | core | no | 24 | M0-element |
| 49 | [M0-window-paint](#m0-window-paint) | core | no | 23 | M0-window-core |
| 50 | [M0-window-dispatch](#m0-window-dispatch) | complex | no | 22 | M0-window-paint |
| 51 | [M0-div-api](#m0-div-api) | complex | no | 21 | M0-window-dispatch |
| 52 | [M0-div-paint](#m0-div-paint) | complex | no | 20 | M0-div-api |
| 53 | [M0-text-elements](#m0-text-elements) | complex | no | 19 | M0-div-paint |
| 54 | [M0-window-tests](#m0-window-tests) | complex | no | 18 | M0-text-elements |
| 55 | [M0-blocked-tests](#m0-blocked-tests) | complex | no | 17 | M0-window-tests |
| 56 | [M0-elements-basic](#m0-elements-basic) | complex | no | 16 | M0-blocked-tests |
| 57 | [M0-list](#m0-list) | complex | no | 15 | M0-elements-basic, M0-sumtree |
| 58 | [M0-elements-media](#m0-elements-media) | complex | no | 14 | M0-list |
| 59 | [M0-window-a11y](#m0-window-a11y) | complex | no | 13 | M0-elements-media |
| 60 | [M0-exit](#m0-exit) | complex | yes | 12 | M0-window-a11y, M0-taffy-handwritten-b, M0-parity-rescan, M0-sumtree, M0-etagere |
| 61 | [M1-wayland-ffi](#m1-wayland-ffi) | core | yes | 17 | M0-bootstrap |
| 62 | [M1-linux-ffi](#m1-linux-ffi) | complex | no | 18 | M0-bootstrap |
| 63 | [M1-x11-proto](#m1-x11-proto) | complex | yes | 16 | M0-bootstrap |
| 64 | [M1-linux-dbus](#m1-linux-dbus) | complex | no | 18 | M0-platform-data |
| 65 | [M1-wgpu-context](#m1-wgpu-context) | core | yes | 17 | M0-scene, M0-etagere |
| 66 | [M1-wgpu-renderer](#m1-wgpu-renderer) | complex | no | 15 | M1-wgpu-context |
| 67 | [M1-linux-common](#m1-linux-common) | complex | yes | 17 | M0-test-contexts, M1-linux-ffi, M1-linux-dbus |
| 68 | [M1-wayland-client](#m1-wayland-client) | complex | no | 16 | M1-wayland-ffi, M1-linux-common, M1-wgpu-context |
| 69 | [M1-wayland-input](#m1-wayland-input) | complex | no | 15 | M1-wayland-client |
| 70 | [M1-wayland-window](#m1-wayland-window) | complex | no | 14 | M1-wayland-input, M1-wgpu-renderer |
| 71 | [M1-x11-client](#m1-x11-client) | complex | no | 15 | M1-x11-proto, M1-linux-common, M1-wgpu-context |
| 72 | [M1-x11-window](#m1-x11-window) | complex | yes | 14 | M1-x11-client, M1-wgpu-renderer |
| 73 | [M1-linux-platform](#m1-linux-platform) | complex | no | 13 | M1-wayland-window, M1-x11-window |
| 74 | [M1-linux-screencap](#m1-linux-screencap) | complex | yes | 12 | M1-linux-platform, M1-linux-dbus |
| 75 | [M1-exit](#m1-exit) | complex | yes | 11 | M0-exit, M1-linux-platform, M1-linux-screencap, M4-web-webgl |
| 76 | [M2-gotext-layout](#m2-gotext-layout) | complex | no | 13 | M0-text-system |
| 77 | [M2-gotext-raster](#m2-gotext-raster) | complex | no | 12 | M2-gotext-layout |
| 78 | [M2-gotext-color](#m2-gotext-color) | complex | no | 11 | M2-gotext-raster |
| 79 | [M2-exit](#m2-exit) | complex | yes | 10 | M1-exit, M2-gotext-color |
| 80 | [M3-profiler-journal](#m3-profiler-journal) | complex | no | 9 | M2-exit |
| 81 | [M3-bench-context](#m3-bench-context) | complex | no | 8 | M3-profiler-journal |
| 82 | [M3-randomized-tree](#m3-randomized-tree) | complex | no | 7 | M3-bench-context |
| 83 | [M3-profiler-hang](#m3-profiler-hang) | complex | no | 6 | M3-randomized-tree |
| 84 | [M3-root-capture](#m3-root-capture) | complex | no | 5 | M3-profiler-hang |
| 85 | [M3-a11y-consumer-tree](#m3-a11y-consumer-tree) | complex | no | 10 | M0-a11y-schema |
| 86 | [M3-a11y-consumer-node](#m3-a11y-consumer-node) | complex | no | 9 | M3-a11y-consumer-tree |
| 87 | [M3-a11y-consumer-text](#m3-a11y-consumer-text) | complex | no | 8 | M3-a11y-consumer-node |
| 88 | [M3-a11y-atspi-node](#m3-a11y-atspi-node) | complex | no | 7 | M3-a11y-consumer-text |
| 89 | [M3-a11y-atspi-adapter](#m3-a11y-atspi-adapter) | complex | no | 6 | M3-a11y-atspi-node |
| 90 | [M3-a11y-atspi-bus](#m3-a11y-atspi-bus) | complex | yes | 5 | M3-a11y-atspi-adapter |
| 91 | [M3-platform-benches](#m3-platform-benches) | complex | no | 5 | M2-exit, M3-randomized-tree, M3-bench-context |
| 92 | [M3-examples-a](#m3-examples-a) | complex | no | 5 | M2-exit |
| 93 | [M3-examples-b](#m3-examples-b) | complex | no | 5 | M2-exit |
| 94 | [M3-examples-c](#m3-examples-c) | complex | no | 5 | M2-exit |
| 95 | [M3-examples-d](#m3-examples-d) | complex | no | 5 | M2-exit |
| 96 | [M3-exit](#m3-exit) | complex | yes | 4 | M3-examples-a, M3-examples-b, M3-examples-c, M3-examples-d, M3-a11y-atspi-bus, M3-root-capture, M3-platform-benches, M4-web-example, M4-ios-window |
| 97 | [M4-apple-ffi](#m4-apple-ffi) | core | yes | 11 | M0-text-system |
| 98 | [M4-apple-text](#m4-apple-text) | complex | no | 10 | M4-apple-ffi, M0-executors |
| 99 | [M4-apple-metal-atlas](#m4-apple-metal-atlas) | complex | yes | 9 | M4-apple-text, M0-etagere, M0-scene |
| 100 | [M4-apple-metal-renderer](#m4-apple-metal-renderer) | complex | no | 8 | M4-apple-metal-atlas |
| 101 | [M4-macos-appkit](#m4-macos-appkit) | complex | no | 10 | M4-apple-ffi |
| 102 | [M4-macos-display](#m4-macos-display) | complex | no | 9 | M4-macos-appkit |
| 103 | [M4-macos-platform](#m4-macos-platform) | complex | no | 8 | M4-macos-display, M4-apple-text, M0-keymap |
| 104 | [M4-macos-window](#m4-macos-window) | complex | no | 7 | M4-macos-platform, M4-apple-metal-renderer |
| 105 | [M4-macos-window-objc](#m4-macos-window-objc) | complex | no | 6 | M4-macos-window |
| 106 | [M4-macos-native-views](#m4-macos-native-views) | complex | no | 5 | M4-macos-window-objc, M0-window-a11y |
| 107 | [M4-macos-a11y](#m4-macos-a11y) | complex | no | 4 | M4-macos-native-views, M3-a11y-consumer-text |
| 108 | [M4-ios-app](#m4-ios-app) | complex | no | 6 | M4-apple-metal-renderer, M1-linux-platform |
| 109 | [M4-ios-window](#m4-ios-window) | complex | no | 5 | M4-ios-app |
| 110 | [M4-windows-com](#m4-windows-com) | core | yes | 11 | M0-executors |
| 111 | [M4-windows-directwrite](#m4-windows-directwrite) | complex | no | 10 | M4-windows-com, M0-text-system |
| 112 | [M4-windows-d3d-atlas](#m4-windows-d3d-atlas) | complex | yes | 9 | M4-windows-directwrite, M0-etagere, M0-scene |
| 113 | [M4-windows-directx-renderer](#m4-windows-directx-renderer) | complex | no | 8 | M4-windows-d3d-atlas |
| 114 | [M4-windows-events](#m4-windows-events) | complex | no | 7 | M4-windows-directx-renderer, M0-keymap |
| 115 | [M4-windows-shell](#m4-windows-shell) | complex | no | 6 | M4-windows-events |
| 116 | [M4-windows-window](#m4-windows-window) | complex | no | 5 | M4-windows-shell |
| 117 | [M4-windows-platform](#m4-windows-platform) | complex | yes | 4 | M4-windows-window |
| 118 | [M4-windows-a11y-node](#m4-windows-a11y-node) | complex | no | 4 | M4-windows-com, M3-a11y-consumer-text |
| 119 | [M4-windows-a11y](#m4-windows-a11y) | complex | no | 3 | M4-windows-a11y-node, M4-windows-platform |
| 120 | [M4-windows-screencap](#m4-windows-screencap) | complex | yes | 3 | M4-windows-d3d-atlas |
| 121 | [M4-web-wgpu-js](#m4-web-wgpu-js) | complex | no | 13 | M1-wgpu-renderer |
| 122 | [M4-web-webgl-shaders](#m4-web-webgl-shaders) | complex | no | 13 | M1-wgpu-context |
| 123 | [M4-web-webgl](#m4-web-webgl) | complex | yes | 12 | M4-web-wgpu-js, M4-web-webgl-shaders |
| 124 | [M4-web-platform](#m4-web-platform) | complex | no | 8 | M4-web-wgpu-js, M0-keymap |
| 125 | [M4-web-input](#m4-web-input) | complex | no | 7 | M4-web-platform |
| 126 | [M4-web-text](#m4-web-text) | complex | no | 6 | M4-web-input, M2-gotext-color, M1-linux-platform |
| 127 | [M4-web-example](#m4-web-example) | standard | no | 5 | M4-web-text, M4-web-webgl |
| 128 | [M4-ci-macos](#m4-ci-macos) | complex | yes | 3 | M3-exit, M4-macos-a11y, M4-ios-window |
| 129 | [M4-ci-windows](#m4-ci-windows) | complex | yes | 2 | M4-ci-macos, M4-windows-a11y, M4-windows-screencap |
| 130 | [M4-exit](#m4-exit) | complex | yes | 1 | M4-ci-windows |

## M0: foundation and leaf packages

### M0-bootstrap

- Goal: Create the module, licence files, agent docs, CI workflow, the parity checker and the per-stage parity file split.
- Tier: core. Risky: yes.
- Depends on: nothing
- Owns: `go.mod`, `go.sum`, `LICENSE`, `NOTICE`, `AGENT.md`, `TECH_DEBT.md`, `.gitignore`, `.github/workflows/ci.yml`, `doc.go`, `internal/deps/`, `internal/cmd/paritycheck/`, `docs/port/PARITY.md`, `docs/port/parity/`, `docs/port/debt/`, `docs/port/handoff/`, `docs/port/status/`, `docs/port/owner-checks/`
- Core task: go.mod: module github.com/megascan/gpui-go, `go 1.27`, keep every require the install step added (add them with the exact versions from the Installs list if missing); internal/deps/deps.go (`//go:build deps`) blank-imports each so nothing prunes them. LICENSE (Apache-2.0), NOTICE crediting Zed Industries' GPUI, AGENT.md (stack, the Shared decisions of STAGES.md in short form, pointers to DESIGN/PARITY/STAGES), TECH_DEBT.md (index, plus two entries: L-WGPUVER "bump github.com/oliverbestmann/webgpu when it ships wgpu-native v30, then re-run the M1-wgpu-context feature probe and drop the fallbacks it no longer needs" and L-WEBTHREADS "Go wasm is single-threaded; revisit if Go gains wasm threads"), .gitignore adds `/_ext`, doc.go (`package gpui` godoc). ci.yml: pull_request + push to feat/port, matrix ubuntu-latest / macos-14 / windows-latest, Go 1.27.x, ubuntu apt packages for wayland, xkbcommon(+x11), xcb, vulkan + lavapipe, xvfb, dbus, node; every OS runs the gofmt check, `go vet ./...` and `go test ./...`; ubuntu also runs, for darwin/arm64, windows/amd64 and js/wasm, `CGO_ENABLED=0 go vet ./...` and `CGO_ENABLED=0 go test -c -o /dev/null ./...`, then `go generate ./...` followed by `git diff --exit-code`, `go test -race` over the non-display packages (the Final verify list), and `go run ./internal/cmd/paritycheck -lint` (not `-all-done`: rows are still open until M4-exit, which switches the step to `-all-done -lint`). Steps for packages that do not exist yet are guarded so CI stays green without edits: the wasm test and the web example build run inside `if [ -d platform/web ]; then ... fi` and `if [ -d examples/web_hello_web ]; then ... fi`. After this stage ci.yml is edited only by exit stages.
- Pieces (each in its own Go package, independent of the core task):
  - **paritycheck** (`internal/cmd/paritycheck/`): CLI that validates the matrix without rescanning upstream. Reads docs/port/PARITY.md section 5 and every docs/port/parity/*.md table. Flags: `-area a,b` (restrict), `-done <upstream path prefix>,...` (no `todo` row under those upstream-file headings), `-all-done` (no `todo` and no `blocked:` row anywhere; `blocked:owner` always fails it), `-allow-blocked` (with `-all-done`: tolerates `blocked:<stage>` while docs/port/status/<stage>.done does not exist, never `blocked:owner`), `-lint` (DESIGN section 4 import rules: the root package imports only internal/{taffy,etagere,lyon,jsonschema} of the internal tree and no platform, renderer, cgo or purego code; cgo only under render/wgpu/ and internal/cmd/wlspike; internal/apple/** only from darwin packages, internal/win/** only from windows packages, internal/linux/** only from linux packages; mixin shadowing against styled-methods.tsv; no `DOCS/` directory; comments other than godoc on exported symbols / `//go:` / `//gpui:` directives / generated headers). Checks: status is `todo|done|dev:<id>|n/a:<id>|blocked:<stage>` (`blocked:owner` is an error in `-all-done` and a warning otherwise); every id is registered in PARITY.md section 5 or in a `## Deviations registered here` table of an area file; every `done` row's Go symbol or test exists (go/parser over the target package, methods as `(Type).Name`); row counts per area never drop below PARITY.md section 2. Unit tests with fixture tables.
  - **parity-split** (`docs/port/parity/`, `docs/port/PARITY.md`): Split area files by upstream-file heading, moving rows verbatim, exactly as the 'Parity file split' table in docs/port/STAGES.md says (linux.md -> linux-common/linux-dbus/linux-platform/linux-wayland/linux-x11; wgpu.md -> wgpu + gotext; core-examples.md -> examples-m2/a/b/c/d + core-tests). Update PARITY.md section 2 (area table) (section 4 already maps the linux dbus files to platform/linux/internal/dbus/). Totals must still add up to 6706 items and 869 tests.
- Verify (`sh docs/port/verify.sh M0-bootstrap` runs exactly these, from the worktree root):

```sh
go build ./... && go vet ./... && go test ./internal/... && test -z "$(gofmt -l .)"
go run ./internal/cmd/paritycheck -lint
```

### M0-foundation

- Goal: Port gpui_util, refineable and collections, add Option[T], and write the refinegen generator.
- Tier: complex. Risky: no.
- Depends on: M0-bootstrap
- Owns: `gpuiutil/`, `refineable/`, `internal/cmd/refinegen/`, `collections/`, `docs/port/parity/foundation.md`
- Core task: gpuiutil: gpui_util/src/lib.rs (705 LOC, 2 tests, including `Deferred` as a value with `Run` / `Abort` used through `defer`, see the Drop table in DESIGN 5.2; arc_cow.rs is already `n/a:D-STRING`) plus `Option[T]` (value type: `Some(v)`, `None[T]()`, `Get() (T, bool)`, `IsSome`, `IsNone`, `UnwrapOr`; the zero value is None). refineable: refineable.rs (Refineable contract of DESIGN 6.6, Cascade, CascadeSlot). refinegen: `//gpui:refineable` on a struct emits `<file>_gen.go` (header `// Code generated by refinegen; DO NOT EDIT.`) with `<T>Refinement` whose fields are `gpuiutil.Option[F]` (a field tagged `//gpui:refineable` nests the field's Refinement type), and Refine / Refined / IsSupersetOf / Subtract plus the From-style conversions derive_refineable.rs (548 LOC) produces; generic structs supported. Golden-file test under testdata. gpui_tokio rows become n/a:D-TOKIO.
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
- Tier: core. Risky: yes.
- Depends on: M0-bootstrap
- Owns: `scheduler/`, `docs/port/parity/scheduler.md`
- Core task: clock.go, scheduler.go, executor.go from scheduler/src (1174 LOC) exactly to DESIGN 5.6 and 6.3: Priority (weights 60/30/10/0), SessionID, Instant, Clock, Scheduler interface, Runnable + RunnableMeta, Timer, ForegroundExecutor / BackgroundExecutor, `Task[T]` as an `iter.Pull` coroutine (`Await(cx Awaiter) (T, error)`, `Wait`, `Detach`, `Cancel` (sets an atomic cancelled flag and wakes the task; it never calls the coroutine's `stop()` itself), `IsReady`, `ReadyTask`), `Awaiter`/`Scope`, and the primitives Oneshot[T], Chan[T], Select, Join, Yield. One poll = one `next()`; a body parks only inside these primitives. No goroutine leak when a task is cancelled or its scheduler is dropped. Lifecycle unit tests use a tiny inline FIFO scheduler defined in the test file. Cancellation rule (DESIGN 5.6): iter.Pull's `next` and `stop` are not safe to call concurrently or re-entrantly, so only the poll loop that owns a task's `next()` ever calls `stop()`, at the start of the next poll after it sees the flag; the parked primitive then sees `yield` return false and the body unwinds through a private panic value the wrapper recovers, so defers run. Cancel is therefore safe from any goroutine, from inside another task and from the task's own body (it unwinds at its next park). Tests cancel from a background goroutine, from a sibling task and from the task itself, under -race.
- Verify (`sh docs/port/verify.sh M0-scheduler-core` runs exactly these, from the worktree root):

```sh
go build ./scheduler/... && go vet ./scheduler/... && go test ./scheduler/... && test -z "$(gofmt -l scheduler)" && go test -race ./scheduler/...
```

### M0-scheduler-test

- Goal: Port the deterministic TestScheduler and all 31 scheduler tests.
- Tier: complex. Risky: no.
- Depends on: M0-scheduler-core
- Owns: `scheduler/`, `docs/port/parity/scheduler.md`
- Core task: test_scheduler.go from test_scheduler.rs (971 LOC): seeded candidate selection (one runnable per foreground session + all background, priority weights), randomize_order, allow_parking, timeout ticks, virtual clock, AdvanceClock, Tick/Run, block with parking and pending-trace checks, SharedRng on math/rand/v2 PCG (D-RNG), end-of-run task drain. scheduler_test.go: the 31 tests of tests.rs as TestScheduler_*; same seed twice must give the same trace. Runs single-goroutine from the caller's point of view: only the polled coroutine runs at any time.
- Verify (`sh docs/port/verify.sh M0-scheduler-test` runs exactly these, from the worktree root):

```sh
go build ./scheduler/... && go vet ./scheduler/... && go test ./scheduler/... && test -z "$(gofmt -l scheduler)" && go test -count=3 ./scheduler/... && go test -race ./scheduler/...
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

### M0-a11y-geometry

- Goal: Port AccessKit 0.24 geometry and create the a11y parity file.
- Tier: standard. Risky: no.
- Depends on: M0-bootstrap
- Owns: `a11y/geometry.go`, `a11y/geometry_test.go`, `docs/port/parity/ext-a11y.md`
- Core task: Package a11y, file geometry.go from ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit-0.24.1/src/geometry.rs (866): Point, Vec2, Size, Rect, Insets, Affine with every method and operator (operators as methods: Add, Sub, Mul, ...), float64 maths in upstream order, plus its unit tests. Create docs/port/parity/ext-a11y.md with one row per public item and test of src/lib.rs and src/geometry.rs (lib.rs rows stay `todo` for M0-a11y-schema). Files directly in a11y/ only; sub-directories belong to later stages.
- Verify (`sh docs/port/verify.sh M0-a11y-geometry` runs exactly these, from the worktree root):

```sh
go build ./a11y && go vet ./a11y && go test ./a11y && test -z "$(gofmt -l a11y)"
go run ./internal/cmd/paritycheck -area ext-a11y
```

### M0-a11y-schema

- Goal: Pure-Go copy of the AccessKit 0.24 data model (lib.rs) that elements, windows and adapters share.
- Tier: complex. Risky: yes.
- Depends on: M0-a11y-geometry
- Owns: `a11y/*.go`, `a11y/internal/`, `docs/port/parity/ext-a11y.md`
- Core task: Package a11y from accesskit-0.24.1 src/lib.rs (3248): Role, Action, Orientation, Toggled, Invalid, Live, TextDirection, all other enums with the same numeric values and names, NodeID, Node with every property getter / setter / clearer (flags, node-id vectors, strings, numbers, rects, text properties), Tree, TreeUpdate, ActionRequest, ActionData, the ActivationHandler / ActionHandler / DeactivationHandler interfaces; geometry types come from a11y/geometry.go (M0-a11y-geometry). Files directly in a11y/ only (sub-directories belong to later stages). Property storage may be generated by a small go:generate tool in a11y/internal/gen. Fill the lib.rs rows of ext-a11y.md; equality and JSON round-trip tests.
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

### M0-stylegen

- Goal: Write the generator for the 3099 macro-generated Styled methods, ahead of the root lane.
- Tier: complex. Risky: no.
- Depends on: M0-bootstrap
- Owns: `internal/cmd/stylegen/`
- Core task: internal/cmd/stylegen ports the tables and expansion rules of `_ref/gpui_macros/src/styles.rs` (1417 LOC: style_helpers! and the eight *_style_methods! macros, prefix / suffix tables at :699-1417, generate_box_style_methods / generate_methods / border_style_methods at :531-621 and :335-360). Output: one Go file of methods on `Styled[S]` (receiver `*Styled[S]`, returning `S`, parameters per D-INTO: IntoLength, IntoHsla, IntoFill, DefiniteLength, ...), each body setting the same StyleRefinement field path the macro sets, in the macro's order, with terse godoc from the macro doc strings. Flags `-out <file>` and `-pkg <name>`; header `// Code generated by stylegen; DO NOT EDIT.`. Tests: golden output under testdata for a few tables, and the generated method names compared with docs/port/parity/styled-methods.tsv (3099 names, digit boundaries keep `_`). The tool needs no root package to build or test; M0-style runs it to write styled_gen.go and marks the macros.md rows.
- Verify (`sh docs/port/verify.sh M0-stylegen` runs exactly these, from the worktree root):

```sh
go build ./internal/cmd/stylegen/... && go vet ./internal/cmd/stylegen/... && go test ./internal/cmd/stylegen/... && test -z "$(gofmt -l internal/cmd/stylegen)"
```

### M0-actiongen

- Goal: Write the //gpui:action generator ahead of the root lane.
- Tier: standard. Risky: no.
- Depends on: M0-bootstrap
- Owns: `internal/cmd/actiongen/`
- Core task: Generator for `//gpui:action namespace=<ns> [name=<n>] [no_json] [no_register] [deprecated_aliases=a,b] [deprecated=<msg>]` above a struct type: emits `<file>_gen.go` with `Name() string`, `BoxedClone() gpui.Action`, `PartialEq(gpui.Action) bool` and an `init` calling `gpui.RegisterAction(gpui.ActionData{...})` (Build decodes with encoding/json/v2 unless no_json, JSONSchema through internal/jsonschema), mirroring `_ref/gpui_macros/src/derive_action.rs` and `register_action.rs`. Unit-struct lists (`actions!`) use the same directive on `struct{}` types. Golden tests under testdata; the generator needs only DESIGN 6.4, not the root package, to build and test. M0-input-types marks the macros.md rows.
- Verify (`sh docs/port/verify.sh M0-actiongen` runs exactly these, from the worktree root):

```sh
go build ./internal/cmd/actiongen/... && go vet ./internal/cmd/actiongen/... && go test ./internal/cmd/actiongen/... && test -z "$(gofmt -l internal/cmd/actiongen)"
```

## M0: taffy port (`internal/taffy`, one writer at a time)

### M0-taffy-style

- Goal: Start the taffy 0.15.0 port: geometry, lengths, available space and math helpers, and set the package conventions.
- Tier: core. Risky: no.
- Depends on: M0-foundation
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: Source ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/taffy-0.15.0/src; port only code enabled by features std, taffy_tree, flexbox, grid, block_layout, float_layout, content_size (default-features off). This stage: geometry.rs (773), style/compact_length.rs (793, as a small tagged struct, no pointer packing), style/dimension.rs (762), style/available_space.rs (163), util/math.rs (356), util/sys.rs (213), with their inline unit tests (about 3.1k LOC). It sets the conventions the next twelve taffy stages follow: one Go file per Rust file named by its path (`style_dimension.go`), PARITY N-rules for names, `Option<f32>` as gpuiutil.Option[float32], float32 maths in upstream order, field-less enums as uint8, a package-local stubs.go for forward references. Create ext-taffy.md: every public item and inline test of the crate (rows for later stages stay todo).
- Verify (`sh docs/port/verify.sh M0-taffy-style` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-style-mod

- Goal: Port the taffy Style struct with the alignment, flex, block and float style modules.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-style
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: style/mod.rs (1816: Style, Display, Position, Overflow, BoxSizing, the CoreStyle / FlexboxContainerStyle / ... traits as Go interfaces), style/alignment.rs (979), style/flex.rs (293), style/block.rs (95), style/float.rs (80), with inline tests (about 3.3k LOC). Grid fields of Style are declared here in upstream order with their types stubbed in stubs.go for M0-taffy-style-grid.
- Verify (`sh docs/port/verify.sh M0-taffy-style-mod` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-style-grid

- Goal: Port the taffy grid style types and the style helper constructors.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-style-mod
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: style/grid.rs (1978: track sizing functions, repeat, min/max track sizing, grid placement, named lines/areas, GridTemplateComponent), style_helpers.rs (634: the constructor helpers as plain functions, e.g. `Length`, `Percent`, `Auto`, `Fr`, `Minmax`, `Repeat`, `FitContent`, `Line`, `Span`), with inline tests. Replace the grid stubs of the previous stage.
- Verify (`sh docs/port/verify.sh M0-taffy-style-grid` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-tree

- Goal: Port the taffy tree traits, cache, layout output types and utilities.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-style-grid
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: tree/traits.rs (493: the layout tree traits as Go interfaces), tree/cache.rs (555), tree/layout.rs (728), tree/node.rs (60: NodeId), tree/mod.rs, util/resolve.rs (371), util/print.rs, util/debug.rs, util/parse.rs, util/front_back_vec_builder.rs, util/mod.rs, lib.rs and prelude.rs items, with inline tests (about 3k LOC).
- Verify (`sh docs/port/verify.sh M0-taffy-tree` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-compute

- Goal: Port TaffyTree and the algorithm-independent compute code.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-tree
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: tree/taffy_tree.rs (1543: TaffyTree with slot-map node ids whose uint64 encoding matches slotmap's KeyData, because gpui's LayoutID wraps it), compute/mod.rs (516: root layout, rounding, hidden layout, cached dispatch), compute/leaf.rs (188), compute/common/alignment.rs, scrollable_overflow.rs and sizing_keyword.rs, with inline tests (about 2.6k LOC). Dispatch by Display calls computeFlexboxLayout / computeBlockLayout / computeGridLayout, which this stage declares in stubs.go as panicking stubs with the final signatures.
- Verify (`sh docs/port/verify.sh M0-taffy-compute` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-fixtures

- Goal: Bring in taffy's 6505 XML layout fixtures and port the runner.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-compute
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

### M0-taffy-grid-types

- Goal: Port the grid data types and their unit tests.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-block
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: compute/grid/types/* (named.rs 998, grid_item.rs 817, cell_occupancy.rs 608, coordinates.rs 184, grid_track.rs 169, grid_track_counts.rs 102, mod.rs), compute/grid/util/* and util/test_helpers.rs, with the many inline unit tests of these files (they are the main check; grid fixtures stay disabled). About 3k LOC.
- Verify (`sh docs/port/verify.sh M0-taffy-grid-types` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-grid-placement

- Goal: Port explicit / implicit grid resolution and item placement.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-grid-types
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: compute/grid/implicit_grid.rs (249), explicit_grid.rs (833), placement.rs (638) with their inline unit tests; grid fixtures stay disabled.
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

### M0-taffy-handwritten-a

- Goal: Port the first half of taffy's hand-written integration tests.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-grid-sizing
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: `_ext/taffy/tests/hand_written/*.rs` (26 files, about 5k LOC) sorted by file name; this stage takes files in that order until the running total reaches half of the total line count (write the split into ext-taffy.md so the next stage takes the rest). Each becomes hand_written_<name>_test.go, one Go test per upstream test, with rows in ext-taffy.md. Fix any engine bug they expose in the owning file.
- Verify (`sh docs/port/verify.sh M0-taffy-handwritten-a` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
```

### M0-taffy-handwritten-b

- Goal: Port the second half of taffy's hand-written integration tests and close the taffy area.
- Tier: complex. Risky: no.
- Depends on: M0-taffy-handwritten-a
- Owns: `internal/taffy/`, `docs/port/parity/ext-taffy.md`
- Core task: The hand_written files not taken by M0-taffy-handwritten-a (the split is recorded in ext-taffy.md), same rules. Afterwards every ext-taffy.md row is done.
- Verify (`sh docs/port/verify.sh M0-taffy-handwritten-b` runs exactly these, from the worktree root):

```sh
go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)"
go run ./internal/cmd/paritycheck -area ext-taffy -all-done
```

## M0: root package chain (one writer at a time)

### M0-geometry

- Goal: Port the first half of geometry.rs and the string types of the root package.
- Tier: core. Risky: no.
- Depends on: M0-foundation
- Owns: `geometry.go`, `geometry_test.go`, `geometry_gen.go`, `shared_string.go`, `shared_uri.go`, `option.go`, `stubs.go`, `docs/port/parity/core-geometry.md`, `docs/port/parity/foundation.md`
- Core task: geometry.rs:1-2346 (Number, Point, Size, Bounds, Edges, Along, Half, Anchor and their impls) with the two families of DESIGN 6.1 (D-GEOM): numeric `Point[T Number]` / Size / Bounds / Edges with methods, and `PointOf[T any]` / SizeOf / EdgesOf; impls on one non-numeric instantiation become free functions (`EdgesLengthZero`). Declare Pixels, DevicePixels, ScaledPixels and Rems as defined numeric types (their methods come in M0-geometry-units) and put any other later geometry symbol this half needs in stubs.go (this stage creates stubs.go; protocol in Shared decision 5). Field order and 4-byte fields are fixed (GPU buffers). `//gpui:refineable` on Bounds and Edges -> geometry_gen.go via refinegen. `type Option[T any] = gpuiutil.Option[T]`, SharedString (D-STRING; it becomes an element in M0-text-elements), SharedURI. Conversions to taffy types are not written here (rows `blocked:M0-element`).
- Verify (`sh docs/port/verify.sh M0-geometry` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-geometry,foundation -done gpui_shared_string/,gpui/src/shared_uri.rs
```

### M0-geometry-units

- Goal: Port the rest of geometry.rs (Corners, units, lengths) and color.rs.
- Tier: complex. Risky: no.
- Depends on: M0-geometry
- Owns: `geometry.go`, `geometry_test.go`, `geometry_gen.go`, `color.go`, `color_test.go`, `stubs.go`, `docs/port/parity/core-geometry.md`
- Core task: geometry.rs:2347-4165 (Corners and CornersOf with `//gpui:refineable`, Radians, Percentage, the Pixels / DevicePixels / ScaledPixels / Rems methods and conversions, Length / DefiniteLength / AbsoluteLength / IntoLength, GridPlacement, Axis, Corner and the rest, plus the 3 tests) and color.rs (1070, 46 items, 10 tests: Hsla / Rgba / Background / gradients with identical float32 maths). Remove the geometry entries from stubs.go. Conversions to taffy types are not written here (rows `blocked:M0-element`).
- Verify (`sh docs/port/verify.sh M0-geometry-units` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-geometry -done gpui/src/geometry.rs,gpui/src/color.rs
```

### M0-contracts

- Goal: Land the compiling skeleton of every cross-stage root contract so other lanes can build against it.
- Tier: core. Risky: yes.
- Depends on: M0-geometry-units, M0-a11y-schema, M0-scheduler-core
- Owns: `gpui.go`, `stubs.go`, `docs/api-contracts.md`, `contracts_test.go`, `docs/port/parity/core-geometry.md`
- Core task: gpui.go: gpui/src/gpui.rs (sealed AppContext / VisualContext of DESIGN 6.2 with the full primitive set: updateEntity, readEntity, reserveEntity, insertEntity, asMut, updateWindow, withWindow, readWindow, readGlobal, backgroundExecutor; the free generic functions AppContextNew / AppContextReserveEntity / AppContextInsertEntity / AppContextUpdateEntity / AppContextReadEntity / AppContextAsMut / AppContextReadWindow / AppContextWithWindow / AppContextBackgroundSpawn / AppContextReadGlobal over them; VisualContext with `UpdateWindowEntity` / `NewWindowEntity` / `ReplaceRootView` / `Focus` returning `error` for upstream's `Self::Result`, gpui.rs:265; Global, BorrowAppContext as SetGlobal / UpdateGlobal / UpdateDefaultGlobal generic functions, re-export aliases such as `Task[T] = scheduler.Task[T]`, a11y aliases). stubs.go: declarations for DESIGN 6.2-6.6, 6.10 and 6.11 with the exact names and signatures given there and bodies `panic("gpui: unimplemented <symbol>")`: App and AppCell (with the borrow guard of DESIGN 5.2: Borrow / BorrowMut / TryBorrowMut returning AppRef / AppRefMut guards), Entity / WeakEntity / AnyEntity / AnyWeakEntity (`UpdateIn(cx VisualContext, f) error`, `UpdateInR[R](...) (R, error)`), Context[T], AsyncApp, GpuiBorrow[T], Subscription, BackgroundExecutor / ForegroundExecutor, Window with the 6.2 method list, FocusHandle, WindowHandle / AnyWindowHandle, ContentMask, DispatchPhase, ElementID / GlobalElementID, Element[RL, PP], the optional element interfaces ElementA11y and ElementA11ySyntheticChildren[PP], AnyElement, IntoElement, Render, RenderOnce, View, the mixin shells Styled / Interactive / StatefulInteractive / Parent / Fluent with their bind helpers, Div and Stateful[E] (an IntoElement-only pass-through, DESIGN 6.5), Style / StyleRefinement, LayoutID, AvailableSpace, Action / ActionData / RegisterAction, Keystroke / Modifiers, Scene, RunTest / TestRun / TestAppContext. docs/api-contracts.md: the stub protocol, which stage replaces which symbol, mixin binding, handle and Drop rules (DESIGN 5.2 table), D-VOID naming, how each context type exposes New / ReadGlobal / BackgroundSpawn (DESIGN 6.2). contracts_test.go compiles the DESIGN section 6 usage snippets.
- Verify (`sh docs/port/verify.sh M0-contracts` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-geometry -done gpui/src/gpui.rs
```

### M0-platform-types

- Goal: Port the first half of platform.rs: every platform interface, the atlas types and the screen-capture seam.
- Tier: core. Risky: yes.
- Depends on: M0-contracts
- Owns: `platform.go`, `platform_layer_shell.go`, `platform_platform_info.go`, `platform_keyboard.go`, `platform_cfg_linux.go`, `platform_cfg_windows.go`, `platform_cfg_other.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-platform.md`, `docs/port/parity/core-input.md`
- Core task: platform.rs:1-2126 plus platform/keyboard.rs, layer_shell.rs and platform_info.rs. Interfaces method for method in upstream order: Platform (83), PlatformWindow (108), PlatformDisplay, PlatformDispatcher, PlatformTextSystem, PlatformAtlas, AtlasBackend, PlatformHeadlessRenderer, ScreenCaptureSource / Stream, PlatformKeyboardLayout / Mapper; `PlatformDefaults` and `PlatformWindowDefaults` carry upstream's default bodies; cfg-gated methods exist on every GOOS. Data types of this range: WindowVisibility, DisplayEvent, ActivationPolicy, ActivityGuard (Release per the Drop table), the three GraphicalEnvironment cfg variants in platform_cfg_<goos>.go, SystemNotification*, PowerMode, ThermalState, SourceMetadata, ScreenCaptureFrame, DisplayId, ResizeEdge, WindowDecorations, Decorations, WindowControls, WindowButtonLayout, Tiling, A11yCallbacks, FrameRequestSource, RequestFrameOptions, PlatformFrameSignal, AppLifecyclePhase, WindowInsets, TextInputStateChange, TasksIncluded, RunnableVariant, TimerResolutionGuard (a gpuiutil.Deferred), NoopTextSystem, AtlasKey / AtlasTile / AtlasState / HeadlessAtlas / AtlasTextureList / AtlasTextureID / AtlasTextureKind / TileID. `RawWindowHandle` is a tagged struct (Wayland display+surface, Xcb connection+window+visual, AppKit view, UIKit view, Win32 hwnd+hinstance, Web canvas id). Also the screen-capture back-end seam (DESIGN 6.9): `type ScapBackend interface` and `RegisterScapBackend(ScapBackend)` with a real body, so back ends can be written before M3-root-capture ports the glue. Types the interfaces mention but later stages own (Keymap, Menu, WindowOptions, ...) are stubbed. etagere conversions are not written here.
- Verify (`sh docs/port/verify.sh M0-platform-types` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-platform,core-input -done gpui/src/platform/layer_shell.rs,gpui/src/platform/platform_info.rs,gpui/src/platform/keyboard.rs
```

### M0-platform-data

- Goal: Port the second half of platform.rs: input handler, window options, clipboard and images.
- Tier: complex. Risky: yes.
- Depends on: M0-platform-types
- Owns: `platform.go`, `platform_test.go`, `platform_app_menu.go`, `platform_app_menu_test.go`, `platform_popup.go`, `stubs.go`, `docs/port/parity/core-platform.md`
- Core task: platform.rs:2127-4116 (PlatformInputHandler, UTF16Selection, InputHandler, TextInputConfiguration, Autocapitalize, TextInputAction, WindowOptions, WindowParams, WindowBounds, TitlebarOptions, WindowKind, WindowAppearance, WindowBackgroundAppearance, TextRenderingMode, PathPromptOptions, PromptLevel, PromptButton, CursorStyle, ClipboardItem / Entry / ReadError / String, ImageFormat and Image with static and animated decode (stdlib, x/image/webp, gen2brain/avif), and the 22 tests) plus platform/app_menu.rs (426, 2 tests) and popup.rs (134). A decoder that does not build for js/wasm sits in a `!js` file with an unsupported-format error on js, so the root package still vets for GOOS=js. Replace the matching stubs.
- Verify (`sh docs/port/verify.sh M0-platform-data` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-platform -done gpui/src/platform.rs,gpui/src/platform/app_menu.rs,gpui/src/platform/popup.rs
```

### M0-input-types

- Goal: Port keystrokes, input events, the input handler, actions and the small app-level helpers.
- Tier: complex. Risky: yes.
- Depends on: M0-platform-data, M0-jsonschema, M0-actiongen
- Owns: `action.go`, `action_test.go`, `seeds.go`, `seeds_test.go`, `assets.go`, `assets_test.go`, `util.go`, `util_test.go`, `platform_keystroke.go`, `platform_keystroke_test.go`, `interactive.go`, `input.go`, `global.go`, `colors.go`, `stubs.go`, `docs/port/parity/core-input.md`, `docs/port/parity/core-app.md`, `docs/port/parity/core-geometry.md`, `docs/port/parity/macros.md`
- Core task: platform/keystroke.rs (813: Keystroke, Modifiers, parse / unparse, KeybindingKeystroke), interactive.rs (998: every input event and PlatformInput), input.rs (611: EntityInputHandler, ElementInputHandler), action.rs (456: Action, ActionData, RegisterAction called from init, registry keyed by name and reflect.Type, `Build(jsontext.Value)` on encoding/json/v2, schema through internal/jsonschema, deprecated aliases, NoAction), global.rs, seeds.rs, assets.rs, util.rs (the real `Fluent[S]` mixin: When, WhenSome, WhenNone, WhenElse, Map), colors.rs; about 3.5k LOC. Mark the derive_action / register_action / actions! rows of macros.md done against internal/cmd/actiongen (built by M0-actiongen). gpui::tests that need a window get status `blocked:M0-blocked-tests`.
- Verify (`sh docs/port/verify.sh M0-input-types` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-input,core-app -done gpui/src/platform/keystroke.rs,gpui/src/action.rs,gpui/src/global.rs,gpui/src/assets.rs
```

### M0-scene

- Goal: Port the scene, its primitives, the bounds tree and path building, and define the renderer contract.
- Tier: core. Risky: yes.
- Depends on: M0-input-types, M0-lyon
- Owns: `scene.go`, `scene_test.go`, `bounds_tree.go`, `bounds_tree_test.go`, `path_builder.go`, `path_builder_test.go`, `path_winding.go`, `path_winding_test.go`, `scene_layout_test.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-style.md`, `docs/port/parity/core-geometry.md`, `render/render.go`
- Core task: scene.rs (1475, 4 tests), bounds_tree.rs (500, 2), path_builder.rs (577), path_winding.rs (842, 5). Primitive structs (Quad, Shadow, Underline, MonochromeSprite, SubpixelSprite, PolychromeSprite, Path + vertices, PaintSurface, native views) keep upstream field order with fixed-size 4-byte-aligned fields only; Scene keeps one slice per kind, paint operations, ordering through the bounds tree, `Batches() iter.Seq[PrimitiveBatch]`. scene_layout_test.go asserts unsafe.Sizeof and field offsets of every primitive against the struct declarations in `_ref/gpui_wgpu/src/shaders.wgsl`, `_ref/gpui_apple/src/shaders.metal` and `_ref/gpui_windows/src/shaders.hlsl` (expected numbers written into the test). PathBuilder runs on internal/lyon and strokes with xo/okage's tiny-skia-path port; TransformationMatrix, PathStyle, StrokeOptions. Also render/render.go: package render with the `Renderer` interface exactly as DESIGN 6.9 (it imports only gpui), so every platform package and the Metal and DirectX renderers can build against it without the Linux cgo renderer.
- Verify (`sh docs/port/verify.sh M0-scene` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go build ./render && go vet ./render && test -z "$(gofmt -l render/render.go)"
go run ./internal/cmd/paritycheck -area core-style,core-geometry -done gpui/src/scene.rs,gpui/src/bounds_tree.rs,gpui/src/path_builder.rs,gpui/src/path_winding.rs
```

### M0-text-system

- Goal: Port the text system front end: fonts, runs, metrics and the line layout cache.
- Tier: complex. Risky: no.
- Depends on: M0-scene
- Owns: `text_system.go`, `text_system_test.go`, `text_system_line_layout.go`, `text_system_line_layout_test.go`, `text_system_font_fallbacks.go`, `text_system_font_features.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-text.md`
- Core task: text_system.rs (1401, 4 tests): TextSystem, WindowTextSystem, Font, FontID, FontFamilyID, FontWeight, FontStyle, FontRun, TextRun, FontMetrics, GlyphID, RenderGlyphParams, fallback stack handling, MissingGlyphReceiver and LineWrapperHandle with Close / Release per the Drop table; font_fallbacks.rs, font_features.rs; line_layout.rs (1091, 11 tests): LineLayout, ShapedRun, ShapedGlyph, WrappedLineLayout, LineLayoutCache with its frame swap. Offsets stay UTF-8 byte offsets; UTF-16 ranges explicit. LineWrapper is stubbed for M0-line-wrapper; painting (line.rs) comes later. Tests that need TestAppContext are `blocked:M0-blocked-tests`.
- Verify (`sh docs/port/verify.sh M0-text-system` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-text -done gpui/src/text_system/font_fallbacks.rs,gpui/src/text_system/font_features.rs,gpui/src/text_system/line_layout.rs
```

### M0-line-wrapper

- Goal: Port line wrapping and truncation.
- Tier: complex. Risky: no.
- Depends on: M0-text-system
- Owns: `text_system_line_wrapper.go`, `text_system_line_wrapper_test.go`, `stubs.go`, `docs/port/parity/core-text.md`
- Core task: line_wrapper.rs (2068, 20 tests): LineWrapper, Boundary, wrap_line, truncation, with go-text/typesetting/segmenter in place of icu LineSegmenter and unicode_segmentation (line_wrapper.rs:4, :545, :777). Replace the LineWrapper stubs. A test that needs TestAppContext is `blocked:M0-blocked-tests`.
- Verify (`sh docs/port/verify.sh M0-line-wrapper` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
```

### M0-profiler-core

- Goal: Port the task and action profiler that executors and dispatchers call.
- Tier: complex. Risky: no.
- Depends on: M0-line-wrapper
- Owns: `profiler.go`, `profiler_test.go`, `profiler_on.go`, `profiler_off.go`, `profiler_actions.go`, `profiler_hang_monitor.go`, `stubs.go`, `docs/port/parity/core-profiler.md`
- Core task: profiler.rs (2226, 15 tests), profiler/actions.rs (213), profiler/hang/monitor.rs (120). Upstream's `profiler` cargo feature is the build tag `gpui_profiler` (D-FEATURE): the cfg pairs listed in docs/port/parity/_collisions.md (GetAllTimings, GetCurrentThreadTimings, TakeAllStats, GlobalThreadTimings, UpdateRunningTask, SaveTaskTiming, action stats) get one symbol with bodies in profiler_on.go / profiler_off.go. Thread-local timing buffers become per-goroutine-safe structures keyed by the dispatcher thread id. Hang and journal entry points it calls stay in stubs.go until M3. ThreadTimings, TraceGuard and WindowProfiler follow the Drop table (explicit Release at dispatcher-thread exit / scope end / window close).
- Verify (`sh docs/port/verify.sh M0-profiler-core` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler.rs,gpui/src/profiler/actions.rs,gpui/src/profiler/hang/monitor.rs
```

### M0-entities

- Goal: Port the entity map, handles, leak detection and subscriptions.
- Tier: core. Risky: no.
- Depends on: M0-profiler-core, M0-scheduler-test
- Owns: `app_entity_map.go`, `app_entity_map_test.go`, `subscription.go`, `subscription_test.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-app.md`
- Core task: app/entity_map.rs (1421, 5 tests): EntityMap, slots, leases with upstream's double-lease panics and LeaseInner's "Leases must be ended with EntityMap::end_lease" check (strict leak mode reports a lease that was dropped without end), access tracking, ref counts behind a lock, Entity / AnyEntity / WeakEntity / AnyWeakEntity exactly per DESIGN 5.2 (Clone, idempotent Release, runtime.AddCleanup safety net feeding the same dropped-ids queue, `weak.Pointer` to the ref-count table, upgrade = increment-if-not-zero), LeakDetector / HandleID / snapshots / exit report, GPUI_LEAK_DETECTION (strict in this repo's tests) and LEAK_BACKTRACE. subscription.rs (559, 8 tests): SubscriberSet insert-pending / retain / remove-during-dispatch, Subscription Unsubscribe / Detach.
- Verify (`sh docs/port/verify.sh M0-entities` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go test -race -run 'Entity|Subscription' .
go run ./internal/cmd/paritycheck -area core-app -done gpui/src/app/entity_map.rs,gpui/src/subscription.rs
```

### M0-executors

- Goal: Port the executors, the priority queue and the dispatchers behind them.
- Tier: complex. Risky: no.
- Depends on: M0-entities
- Owns: `executor.go`, `executor_test.go`, `platform_scheduler.go`, `platform_scheduler_test.go`, `platform_threaded_dispatcher.go`, `platform_threaded_dispatcher_test.go`, `platform_testing_dispatcher.go`, `queue.go`, `queue_test.go`, `stubs.go`, `docs/port/parity/core-app.md`, `docs/port/parity/core-platform.md`
- Core task: executor.rs (626, 2 tests: BackgroundExecutor / ForegroundExecutor over the scheduler package, Scope with `Close` that blocks until every scoped future resolves, as Scope's Drop does at executor.rs:543), platform_scheduler.rs (457, 6), platform/threaded_dispatcher.rs (744, 8), platform/test/dispatcher.rs (TestDispatcher over scheduler.TestScheduler), queue.rs (541, 3 tests: PriorityQueueSender / Receiver with explicit Close that keeps the sender-count and wake-up ordering of their Drop impls, queue.rs:182 and :428).
- Verify (`sh docs/port/verify.sh M0-executors` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go test -race -run 'Executor|Scheduler|Dispatcher|Queue' .
go run ./internal/cmd/paritycheck -area core-app,core-platform -done gpui/src/executor.rs,gpui/src/platform_scheduler.rs,gpui/src/platform/threaded_dispatcher.rs,gpui/src/queue.rs
```

### M0-app-core

- Goal: Port the first half of app.rs: AppCell with its borrow guard, Application and the App struct.
- Tier: core. Risky: no.
- Depends on: M0-executors
- Owns: `app.go`, `stubs.go`, `docs/port/parity/core-app.md`
- Core task: app.rs:1-1999. AppCell keeps upstream's borrow tracking (app.rs:85-183, DESIGN 5.2): a shared-borrow count and a mutable flag on the cell, `Borrow` / `BorrowMut` / `TryBorrowMut` returning AppRef / AppRefMut guards that are released with `Release` (callers `defer` it), the origin of the current borrow group recorded with runtime.Caller, and the same panic text ("RefCell already borrowed: attempted mutable App borrow at <file:line>; existing shared borrow group began at <file:line>"); TRACK_THREAD_BORROWS prints as upstream. Every place upstream calls `app.borrow_mut()` (Application, AsyncApp, the test contexts) goes through the guard, so a re-entrant update panics where upstream does. Then Application / ApplicationHandle, DisplayState, the missing-glyph callback slots, QuitMode, CursorHideMode, SystemWindowTab / SystemWindowTabController, GpuiMode, the platform-owned drag state, App fields in upstream order (884-1001) and App methods 1002-1999 in order (entity creation, reservation and insertion, update / read with leases, window map with leasing, observers, globals keyed by reflect.Type, display, thermal and power passthroughs). Calls into later code go through stubs.
- Verify (`sh docs/port/verify.sh M0-app-core` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
```

### M0-app-effects

- Goal: Port the second half of app.rs: platform passthroughs, the effect system and the AppContext primitives.
- Tier: complex. Risky: no.
- Depends on: M0-app-core
- Owns: `app.go`, `app_test.go`, `stubs.go`, `docs/port/parity/core-app.md`
- Core task: app.rs:2000-4031: clipboard / credentials / URL / notification / prompt / restart passthroughs, HTTPClient (D-HTTP) and NullHttpClient, push_effect and the Effect enum and flush loop (2254-2440, 3607-3628) with pending_updates nesting, notify / emit / defer / refresh ordering, release and new-entity listeners, keystroke observers and interceptors, active drag, tooltips, quit modes and handlers with SHUTDOWN_TIMEOUT, action building and schema listing, `impl AppContext for App` (3479-3606: every primitive of DESIGN 6.2 incl. asMut, withWindow, readWindow, readGlobal, backgroundSpawn), GlobalLease, AnyDrag, AnyTooltip, InputPreference, KeystrokeEvent, GpuiBorrow (`AsMut` returns a guard whose `Release` ends the lease and notifies, app.rs:3815), and the #[test]s at 3824-4031. gpui::tests are `blocked:M0-test-contexts` or `blocked:M0-blocked-tests`.
- Verify (`sh docs/port/verify.sh M0-app-effects` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-app -done gpui/src/app.rs
```

### M0-app-contexts

- Goal: Port Context[T], the async contexts, the asset cache and the test macro runtime.
- Tier: complex. Risky: no.
- Depends on: M0-app-effects
- Owns: `app_context.go`, `app_async_context.go`, `asset_cache.go`, `asset_cache_test.go`, `testing.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-app.md`, `docs/port/parity/macros.md`
- Core task: app/context.rs (938: Context[T] embedding *App with generic methods Observe, Subscribe, Emit, ObserveRelease, ObserveGlobal, Spawn, Listener, Notify, ...; it satisfies AppContext through the primitives of App and exposes the upstream-named generic methods New / ReadGlobal / BackgroundSpawn by delegating to them, DESIGN 6.2), app/async_context.rs (559: AsyncApp, AsyncWindowContext; `Update` parks the coroutine and the poller runs the closure on the main thread through the AppCell borrow guard, DESIGN 5.6; methods that upstream returns `Result` from return `error`), asset_cache.rs (358), test.rs + gpui_macros/src/test.rs + property_test.rs as testing.go (RunTest with Seed / Seeds / Iterations / Retries / OnFailure, SEED and ITERATIONS env, several TestAppContexts per run, Rand, executor access; PropertyTest without shrinking). TestPlatform is stubbed for M0-test-platform.
- Verify (`sh docs/port/verify.sh M0-app-contexts` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-app -done gpui/src/app/context.rs,gpui/src/app/async_context.rs
```

### M0-test-platform

- Goal: Port the test platform: TestPlatform, TestWindow and TestDisplay.
- Tier: complex. Risky: no.
- Depends on: M0-app-contexts
- Owns: `platform_testing.go`, `platform_testing_platform.go`, `platform_testing_display.go`, `platform_testing_window.go`, `stubs.go`, `docs/port/parity/core-platform.md`
- Core task: platform/test/platform.rs (861), window.rs (742), display.rs (44) and the platform/test.rs module file, method for method: TestPlatform with its prompt / clipboard / credential / cursor / screen-capture-source simulation, TestWindow with simulated input, resize, appearance and its sprite atlas (a HeadlessAtlas), TestDisplay. TestWindow, TestPlatform and their atlases follow the Drop table of DESIGN 5.2 (window close runs the same callbacks as upstream's drop). Replace the matching stubs.
- Verify (`sh docs/port/verify.sh M0-test-platform` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-platform -done gpui/src/platform/test/
```

### M0-test-contexts

- Goal: Port TestAppContext, TestApp, HeadlessAppContext and VisualTestContext; first gpui::tests run.
- Tier: complex. Risky: no.
- Depends on: M0-test-platform
- Owns: `app_test_context.go`, `app_test_context_test.go`, `app_test_app.go`, `app_test_app_test.go`, `app_headless_app_context.go`, `app_visual_test_context.go`, `asset_cache_test.go`, `app_test.go`, `util_test.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-app.md`, `docs/port/parity/core-geometry.md`
- Core task: app/test_context.rs (1562, 94 items), app/test_app.rs (647), app/headless_app_context.rs (293), app/visual_test_context.rs (485). End of each run: quit the app, drain scheduler tasks, runtime.GC until the live-handle count stops falling, then the leak report (DESIGN 6.11). Port every test whose status is `blocked:M0-test-contexts` and every earlier gpui::test that needs no window (asset_cache 5, util 1, the non-window app.rs tests); window-dependent ones move to `blocked:M0-blocked-tests`. HeadlessAppContext has an explicit `Close` that runs the shutdown its Drop impl runs (headless_app_context.rs). Note in docs/api-contracts.md how other packages write a headless test.
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
- Core task: keymap.rs (1143, 24 tests: binding precedence, pending multi-stroke matches, disabled bindings), keymap/binding.rs (166), keymap/context.rs (888, 11 tests: KeyContext, KeyBindingContextPredicate and its parser), key_dispatch.rs code 1-620 (DispatchTree, DispatchNode, focus path, action and key listeners, binding lookup, replay) and the 5 plain #[test]s of its test module; its 20 gpui::tests stay `blocked:M0-blocked-tests`; tab_stop.rs (501, 10 tests).
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
- Core task: gestures.rs (2331 LOC, 17 items, 37 plain tests: the touch recognizer state machine, tap / long-press / pan / pinch / momentum) and spring.rs (830, 28 items, 11 tests). Pure state machines and float maths; durations through scheduler.Instant so tests stay deterministic. Both stay in the root package (they use PlatformInput, TouchEvent, ScrollEvent, Pixels, Point, Hsla and Rems; an internal package would have to import gpui).
- Verify (`sh docs/port/verify.sh M0-gestures-spring` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-input,core-style -done gpui/src/gestures.rs,gpui/src/spring.rs
```

### M0-style

- Goal: Port Style, the Styled mixin and the generated macro methods.
- Tier: complex. Risky: no.
- Depends on: M0-gestures-spring, M0-stylegen
- Owns: `style.go`, `style_test.go`, `style_gen.go`, `styled.go`, `styled_gen.go`, `styled_gen_test.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-style.md`, `docs/port/parity/macros.md`
- Core task: style.rs (1529, 3 tests): Style with upstream fields in upstream order, TextStyle, HighlightStyle, BoxShadow, Fill, borders, the copied taffy enums, cascade and refinement (`//gpui:refineable` -> style_gen.go), the paint helpers (calling Window stubs). styled.rs (904): `Styled[S Styler]` with the 116 hand-written methods. internal/cmd/stylegen (already built by M0-stylegen) writes styled_gen.go from `//go:generate`; fix generator bugs by recording them in docs/port/handoff/M0-style.md, not by editing the tool; styled_gen_test.go checks the method set of `Styled[*Div]` against docs/port/parity/styled-methods.tsv (3099 names, digit boundaries keep `_`). Parameters follow D-INTO (IntoLength, IntoHsla, IntoFill). Conversions to taffy types are left for M0-element.
- Verify (`sh docs/port/verify.sh M0-style` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
s=$(ls styled_gen.go style_gen.go geometry_gen.go | sort | xargs sha256sum) && go generate . && [ "$s" = "$(ls styled_gen.go style_gen.go geometry_gen.go | sort | xargs sha256sum)" ]
go run ./internal/cmd/paritycheck -area core-style -done gpui/src/style.rs,gpui/src/styled.rs
```

### M0-element

- Goal: Port the element machinery, views, the taffy bridge and inspector ids.
- Tier: core. Risky: no.
- Depends on: M0-style, M0-taffy-handwritten-b
- Owns: `element.go`, `element_test.go`, `taffy.go`, `taffy_test.go`, `view.go`, `inspector.go`, `stubs.go`, `docs/api-contracts.md`, `docs/port/parity/core-window.md`, `docs/port/parity/core-style.md`, `docs/port/parity/core-geometry.md`
- Core task: element.rs (1000): Element[RL, PP], the typed drawable with upstream's phase-order panics, AnyElement, IntoElement, Component over RenderOnce (`NewComponent`), the real `Parent[S]` mixin, GlobalElementID with path hash and full-path comparison, the optional interfaces ElementA11y and ElementA11ySyntheticChildren[PP] (upstream's `a11y_synthetic_children(&mut PrepaintState, builder)`, element.rs:132-137: the drawable type-asserts its element to `ElementA11ySyntheticChildren[PP]` with its own PP, so the hook receives the typed prepaint state). view.rs (667): AnyView, AnyWeakView, view caching, `Entity[T].IntoAnyElement` (D-COND). taffy.rs (787, 1 test): TaffyLayoutEngine over internal/taffy with every `From<gpui type> for taffy type` conversion (also those from geometry.rs and style.rs), device-pixel rounding (taffy.rs:130-134, 423-440, 450-580), measured layouts, LayoutID, AvailableSpace. inspector.rs (271). The 2 plain element tests are ported; the gpui::test is `blocked:M0-div-paint`.
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
- Core task: div.rs:1-2297: GroupStyle, DragMoveEvent, the Interactivity listener-registration methods (92-791), `Interactive[S]` with every InteractiveElement method (792-1352) and `StatefulInteractive[S]` (1353-1763), listener type aliases, HoverListenerMode, drag / drop / tooltip builder types, `NewDiv`, the Div struct with its four mixins bound in the constructor, its builder methods to 2195, and the Interactivity struct (2196-2297). Typed listeners (`OnDrag[T]`, `OnDrop[T]`, `OnAction[A]`) are generic methods on the mixin. `ID` takes the ElementIDSource union and returns `*Stateful[S]`.
- Verify (`sh docs/port/verify.sh M0-div-api` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
```

### M0-div-paint

- Goal: Port the layout, prepaint and paint half of div.rs with its tests.
- Tier: complex. Risky: no.
- Depends on: M0-div-api
- Owns: `elements_div_paint.go`, `elements_div_test.go`, `element_test.go`, `stubs.go`, `docs/port/parity/core-elements.md`, `docs/port/parity/core-window.md`
- Core task: div.rs:2298-4539: `impl Element for Div` (request_layout, prepaint with scroll and overflow handling, paint), Interactivity request_layout / prepaint / paint (2298-3717: hover, active, group styles, click and drag state, scroll wheel, focus, key listeners, cursor, inspector), InteractiveElementState, tooltip show / hide machinery (3780-4148), GroupHitboxes, ScrollAnchor, ScrollHandle, and Stateful[E] (4176-4279). Upstream's Stateful<E> implements Element by delegating through E::RequestLayoutState / E::PrepaintState, which a Go type cannot name for a type parameter: `Stateful[E]` therefore implements IntoElement only and its `IntoElement` returns the inner element's AnyElement after setting the id on the shared Interactivity (row `dev:D-COND`); every InteractiveElement and StatefulInteractive method stays available. Port the 28 tests (4540-5865) and the element.rs gpui::test.
- Verify (`sh docs/port/verify.sh M0-div-paint` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-elements,core-window -done gpui/src/elements/div.rs,gpui/src/element.rs
```

### M0-text-elements

- Goal: Port shaped-line painting and the text elements.
- Tier: complex. Risky: no.
- Depends on: M0-div-paint
- Owns: `text_system_line.go`, `text_system_line_test.go`, `elements_text.go`, `elements_text_test.go`, `stubs.go`, `docs/port/parity/core-text.md`, `docs/port/parity/core-elements.md`, `shared_string.go`
- Core task: text_system/line.rs (1871, 12 tests): ShapedLine, WrappedLine, decoration runs, paint and paint_background with underline / strikethrough / background merging. elements/text.rs (1312, 2 tests): `string` and SharedString as elements (SharedString gets `IntoElement`, `RequestLayout` / `Prepaint` / `Paint` as upstream's `impl Element for SharedString`, so `Parent.Child(s)` takes it directly; plain Go strings go through `ChildText(s string)` on the Parent mixin, the one D-INTO helper for `&'static str`), StyledText, TextLayout, InteractiveText with click / hover / tooltip ranges. Tests that need a window run now.
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
- Core task: Find rows with status `blocked:M0-blocked-tests` in the core-*.md files and port them: key_dispatch.rs (the 20 gpui::tests at 621-2200), interactive.rs (2), input.rs (2), app.rs (window tests), app/test_context.rs (8), app/test_app.rs (4), line_wrapper.rs (1) and any other. Fix the code under test when a ported test fails; record anything left in docs/port/handoff/M0-blocked-tests.md. After this stage no core row may be `blocked:` on an M0 stage.
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

- Goal: Port the variable-height list element and the first half of its tests.
- Tier: complex. Risky: no.
- Depends on: M0-elements-basic, M0-sumtree
- Owns: `elements_list.go`, `elements_list_test.go`, `stubs.go`, `docs/port/parity/core-elements.md`
- Core task: elements/list.rs:1-1746: List, ListState over a sumtree of ListItem with Count / Height dimensions, alignment, follow modes, sizing and measuring behaviours, scroll anchors and pending scrolls, splice / reset / remeasure, layout of visible items with overdraw, scroll event handling, ListOffset; then the gpui::tests of `mod test` from 1748 to 2930 into elements_list_test.go. The tests from 2931 to 3883 are ported by M0-window-a11y (leave their rows `todo`).
- Verify (`sh docs/port/verify.sh M0-list` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
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

- Goal: Port the window accessibility tree, native view hosting and the rest of the list tests.
- Tier: complex. Risky: no.
- Depends on: M0-elements-media
- Owns: `window_a11y.go`, `window_a11y_test.go`, `window_native_view.go`, `window_native_view_test.go`, `window_a11y_debug.go`, `stubs.go`, `docs/port/parity/core-window.md`, `elements_list_follow_test.go`, `docs/port/parity/core-elements.md`
- Core task: window/a11y.rs (905, 13 tests: building the a11y.TreeUpdate from the frame, focus, synthetic children, action routing), window/a11y/debug.rs (330: the debug JSON dump), window/native_view.rs (897, 11 gpui::tests: NativeView handles, layering, hit testing; platform bodies exist only on macOS). Replace the a11y and native-view stubs left by the window stages. Also the list.rs gpui::tests from 2931 to 3883 (follow-tail and remeasure cases) into elements_list_follow_test.go.
- Verify (`sh docs/port/verify.sh M0-window-a11y` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-window,core-elements -done gpui/src/window/,gpui/src/elements/list.rs
```

### M0-exit

- Goal: Close M0: audit the headless core, port the integration tests, open the PR and get CI green on three OSes.
- Tier: complex. Risky: yes.
- Depends on: M0-window-a11y, M0-taffy-handwritten-b, M0-parity-rescan, M0-sumtree, M0-etagere
- Owns: `*.go (root package, fixes only)`, `integration_action_macros_test.go`, `integration_eq_hash_test.go`, `macros_derive_context_test.go`, `macros_render_test_test.go`, `.github/workflows/ci.yml`, `docs/port/PARITY.md`, `docs/port/OWNER-CHECKS.md`, `TECH_DEBT.md`, `AGENT.md`, `docs/port/parity/core-tests.md`, `docs/port/parity/macros.md`
- Core task: Port gpui/tests/action_macros.rs, eq_hash.rs and gpui_macros/tests/derive_context.rs, render_test.rs. Audit: stubs.go holds only profiler-hang / journal / debug-overlay / bench / randomized-tree / inspector-reflection / screen-capture glue / visual-test symbols; `paritycheck -rescan` rows added; every M0 row is done, dev, n/a or `blocked:<M1..M4 stage>` (never `blocked:owner`); strict leak mode green; `go test -race` on the root package and scheduler; cross-vet and `go test -c -o /dev/null` for darwin / windows / js over the root and leaf packages; the Drop table of DESIGN 5.2 checked against every M0 type (each row has its Release / Close / Cancel and a test); handoff files addressed. Fold docs/port/debt/*.md into TECH_DEBT.md, fold docs/port/owner-checks/*.md into docs/port/OWNER-CHECKS.md, refresh AGENT.md. Then commit the paths of finished stages only (those with docs/port/status/<stage>.done; never `git add -A`), push feat/port, `gh pr create --draft --base dev`, and fix CI failures in M0 packages on macOS and Windows (ci.yml may be edited only to fix a step, never to drop one).
- Verify (`sh docs/port/verify.sh M0-exit` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
PK=". ./scheduler/... ./sumtree/... ./collections/... ./gpuiutil/... ./refineable/... ./a11y ./internal/taffy/... ./internal/lyon/... ./internal/etagere/... ./internal/jsonschema/... ./internal/cmd/paritycheck/... ./internal/cmd/refinegen/... ./internal/cmd/actiongen/... ./internal/cmd/stylegen/..."; go vet $PK && go test $PK && go test -race . ./scheduler/... && for t in darwin/arm64 windows/amd64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet $PK && GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go test -c -o /dev/null $PK || exit 1; done
go run ./internal/cmd/paritycheck -area core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,macros,core-tests -rescan
gh pr checks feat/port
```

## M1: Linux platform, D-Bus, screen capture and the wgpu renderer

### M1-wayland-ffi

- Goal: Build the Wayland binding: purego runtime over libwayland-client, the protocol generator, generated protocols and a window + GPU surface spike.
- Tier: core. Risky: yes.
- Depends on: M0-bootstrap
- Owns: `internal/cmd/wlgen/`, `internal/linux/wl/`, `internal/linux/wlproto/`, `internal/cmd/wlspike/`
- Core task: Package internal/linux/wl (under the repo-root internal/ so both platform/linux/... and internal/cmd/wlspike can import it; build constraint `linux`): dlopen libwayland-client.so.0 and libwayland-cursor.so.0 with purego; display connect / roundtrip / flush / prepare_read / read_events / dispatch_pending / get_fd; proxies created and called through wl_proxy_marshal_array_flags; events through wl_proxy_add_dispatcher with one purego callback that decodes wl_argument arrays; wl_interface / wl_message tables built in pinned Go memory with C layout; fixed, array, fd, new_id and nullable-object arguments; proxy registry for user data; real `wl_display*` / `wl_surface*` pointers exposed as uintptr (DESIGN 8.1). wlgen: protocol XML -> `internal/linux/wlproto/<protocol>_gen.go` (typed proxy per interface, request methods, an event listener struct of funcs, enums and bitfields as typed constants, `since` versions). Vendor XML under internal/linux/wlproto/xml from /usr/share/wayland, /usr/share/wayland-protocols and the wlr / plasma crates: core, xdg-shell, xdg-decoration, xdg-activation, xdg-dialog, xdg-system-bell, fractional-scale, viewporter, cursor-shape, text-input-v3, primary-selection, pointer-gestures, linux-dmabuf, wlr-layer-shell, kde blur, plus any other interface `_ref/gpui_linux/src/linux/wayland/client.rs` binds. wlspike: xdg_toplevel + a wgpu surface from the raw pointers (oliverbestmann/webgpu), clear colour for N frames, exit. Layout tests for the interface tables and a golden test for the generator. All calls from the locked main thread. wlspike is a dev tool and the one place outside render/wgpu allowed to use cgo, for the webgpu surface only; every wlspike file carries `//go:build linux` so the darwin, windows and js legs of `go vet ./...` and `go test -c ./...` skip it.
- Verify (`sh docs/port/verify.sh M1-wayland-ffi` runs exactly these, from the worktree root):

```sh
go build ./internal/cmd/wlgen/... && go vet ./internal/cmd/wlgen/... && go test ./internal/cmd/wlgen/... && test -z "$(gofmt -l internal/cmd/wlgen)"
go build ./internal/linux/wl/... && go vet ./internal/linux/wl/... && go test ./internal/linux/wl/... && test -z "$(gofmt -l internal/linux/wl)"
go build ./internal/linux/wlproto/... && go vet ./internal/linux/wlproto/... && go test ./internal/linux/wlproto/... && test -z "$(gofmt -l internal/linux/wlproto)"
s=$(find internal/linux/wlproto -name '*_gen.go' | sort | xargs sha256sum) && go generate ./internal/linux/wlproto/... && [ "$s" = "$(find internal/linux/wlproto -name '*_gen.go' | sort | xargs sha256sum)" ]
go vet ./internal/cmd/wlspike && { [ -z "$WAYLAND_DISPLAY" ] || timeout 60 go run ./internal/cmd/wlspike -frames 90; }
```

### M1-linux-ffi

- Goal: purego bindings for xkbcommon and a minimal libxcb shim.
- Tier: complex. Risky: no.
- Depends on: M0-bootstrap
- Owns: `internal/linux/xkb/`, `internal/linux/xcbshim/`
- Core task: Package internal/linux/xkb (build constraint `linux`) over libxkbcommon.so.0 and libxkbcommon-x11.so.0: Context, Keymap (from names, string, fd-mapped buffer, X11 device), State (update_mask, update_key, key_get_one_sym, key_get_utf8, serialize mods / layout, mod and led queries, layout names), keysym helpers, compose Table / State (feed, status, utf8, reset). The API mirrors the xkbcommon-rs calls used in `_ref/gpui_linux/src/linux/platform.rs:1302-1602`, wayland/client.rs and x11/client.rs. Explicit Close on every object (D-RAII). Tests: build the default keymap from names and check that the key for `a` yields "a" and shift gives "A"; a null-context rejection test.
- Pieces (each in its own Go package, independent of the core task):
  - **xcbshim** (`internal/linux/xcbshim/`): purego over libxcb.so.1: xcb_connect, xcb_connection_has_error, xcb_get_setup + screen iteration to find a visual id, xcb_flush, xcb_disconnect. `Disconnect` must be called only after every wgpu surface made from the connection is released (the X11 window tears the renderer down first, x11/window.rs drop order). It exists only to hand a real `xcb_connection_t*` to the wgpu surface for a window id created over jezek/xgb (DESIGN 8.1). Test connects only when DISPLAY is set and GPUI_TEST_DISPLAY=1.
- Verify (`sh docs/port/verify.sh M1-linux-ffi` runs exactly these, from the worktree root):

```sh
go build ./internal/linux/xkb/... && go vet ./internal/linux/xkb/... && go test ./internal/linux/xkb/... && test -z "$(gofmt -l internal/linux/xkb)"
go build ./internal/linux/xcbshim/... && go vet ./internal/linux/xcbshim/... && go test ./internal/linux/xcbshim/... && test -z "$(gofmt -l internal/linux/xcbshim)"
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

### M1-linux-dbus

- Goal: Port the Linux D-Bus services: power, notifications, the desktop portal and the Secret Service keyring.
- Tier: complex. Risky: no.
- Depends on: M0-platform-data
- Owns: `platform/linux/internal/dbus/`, `docs/port/parity/linux-dbus.md`
- Core task: Package platform/linux/internal/dbus on github.com/godbus/dbus/v5 v5.2.4 from `_ref/gpui_linux/src/linux`: power.rs (265, 2 tests: UPower / power-profiles-daemon / logind inhibitors, PowerMode and ThermalState updates), system_notifications.rs (154, 1 test: org.freedesktop.Notifications Notify / CloseNotification / ActionInvoked / NotificationClosed with the same hints and tag replacement), xdg_desktop_portal.rs (249, 1 test: Settings color-scheme / accent / cursor-size watching, OpenURI, FileChooser), and the credential calls of platform.rs:1085-1150 as a Secret Service client (org.freedesktop.secrets: OpenSession, SearchItems, CreateItem, GetSecrets, Delete, Unlock prompt, the same attribute names oo7 writes) with the `plain` session algorithm and the dh-ietf1024-sha256-aes128-cbc-pkcs7 algorithm when the service offers it. Every function takes its *dbus.Conn as a parameter; signal watching runs on a goroutine and posts back through a callback the caller marshals to the main thread. Tests start a private bus with `dbus-daemon --session --print-address --nofork` (skip when the binary is missing) and register fake services on it; they never touch the user's session bus, keyring or notification daemon.
- Verify (`sh docs/port/verify.sh M1-linux-dbus` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/internal/dbus/... && go vet ./platform/linux/internal/dbus/... && go test ./platform/linux/internal/dbus/... && test -z "$(gofmt -l platform/linux/internal/dbus)"
go run ./internal/cmd/paritycheck -area linux-dbus -all-done
```

### M1-wgpu-context

- Goal: Port the wgpu context, atlas and shader loading behind a small GPU facade, and probe the wgpu-native v29 feature set.
- Tier: core. Risky: yes.
- Depends on: M0-scene, M0-etagere
- Owns: `render/wgpu/`, `docs/port/parity/wgpu.md`
- Core task: render/wgpu (build constraint `linux || js`; this stage writes the native files and leaves `_js.go` files to M4; render/render.go already exists from M0-scene). First the facade: render/wgpu/internal/gpu declares the subset of the WebGPU API the renderer uses (Instance, Adapter, Device, Queue, Surface, Buffer, Texture, TextureView, Sampler, BindGroup(Layout), PipelineLayout, RenderPipeline, ShaderModule, CommandEncoder, RenderPass, with the descriptor structs it needs), and render/wgpu/internal/gpu/webgpu implements it over github.com/oliverbestmann/webgpu v1.38.1 (wgpu-native v29.0.1.1); M4-web-webgl adds a WebGL2 implementation of the same interfaces, so the renderer is written once. Then gpui_wgpu.go, wgpu_context.go (848, 3 tests: instance, adapter selection incl. compositor GPU hints, device features, dual-source blending detection), wgpu_atlas.go (568, 5 tests: WgpuAtlas implementing gpui.PlatformAtlas over internal/etagere, per-kind texture lists, uploads), shaders/*.wgsl copied byte for byte from `_ref/gpui_wgpu/src` and embedded, a sha256 test against `_ref` (skips when absent), concatenation as wgpu_renderer.rs:25-39, and a test that compiles all four variants. Feature probe (upstream targets wgpu 30, the binding ships v29; limit L-WGPUVER): a test that reports, on the local adapter, whether EnumerateAdapters is exported (wgpu_context.rs:215, 531), whether the DUAL_SOURCE_BLENDING feature can be requested (:442-446) and whether TRANSIENT_ATTACHMENT texture usage exists (wgpu_renderer.rs:1571). Where one is missing use the fallback and cite L-WGPUVER on that row: adapter selection through RequestAdapter with power preference plus the compositor hint as the device filter; the non-subpixel text path upstream already takes when dual-source blending is absent; plain RENDER_ATTACHMENT for the intermediate textures. Record the probe result in docs/port/handoff/M1-wgpu-context.md. GPU tests skip when no adapter is found. This package and internal/cmd/wlspike are the only cgo users.
- Verify (`sh docs/port/verify.sh M1-wgpu-context` runs exactly these, from the worktree root):

```sh
go build ./render/wgpu/... && go vet ./render/wgpu/... && go test ./render/wgpu/... && test -z "$(gofmt -l render/wgpu)"
go run ./internal/cmd/paritycheck -area wgpu -done gpui_wgpu/src/wgpu_context.rs,gpui_wgpu/src/wgpu_atlas.rs
```

### M1-wgpu-renderer

- Goal: Port WgpuRenderer and the headless renderer.
- Tier: complex. Risky: no.
- Depends on: M1-wgpu-context
- Owns: `render/wgpu/`, `docs/port/parity/wgpu.md`
- Core task: wgpu_renderer.rs (2900, 10 tests), written only against render/wgpu/internal/gpu: surface configuration from a gpui.RawWindowHandle, pipelines per primitive kind, instance buffers filled with unsafe.Slice views of the scene slices, path rasterisation to the intermediate texture, monochrome / subpixel / polychrome sprites with the subpixel fallback when dual-source blending is missing, the TRANSIENT_ATTACHMENT fallback of the probe, surfaces, transparency, drawable resize, device-lost detection and `Recover`, GPUSpecs; WgpuHeadlessRenderer implementing gpui.PlatformHeadlessRenderer (RenderSceneToImage). `NewRenderer` returns a type that satisfies render.Renderer. Add a pixel test: a scene with one solid quad through the headless renderer. WgpuRenderer has `Destroy` (explicit, Drop table) and must be destroyed before the surface's display connection closes.
- Verify (`sh docs/port/verify.sh M1-wgpu-renderer` runs exactly these, from the worktree root):

```sh
go build ./render/wgpu/... && go vet ./render/wgpu/... && go test ./render/wgpu/... && test -z "$(gofmt -l render/wgpu)"
go run ./internal/cmd/paritycheck -area wgpu -done gpui_wgpu/src/
```

### M1-linux-common

- Goal: Port the shared Linux platform layer: client contract, event loop, headless backend and xkb helpers.
- Tier: complex. Risky: yes.
- Depends on: M0-test-contexts, M1-linux-ffi, M1-linux-dbus
- Owns: `platform/linux/internal/common/`, `docs/port/parity/linux-common.md`
- Core task: From `_ref/gpui_linux/src/linux`: the LinuxClient interface and LinuxCommon state (platform.rs:91-295), dispatcher.rs (369, 1 test) with the calloop loop replaced by a main-goroutine loop over display-fd readiness, the priority runnable queue and timers (background runnables on goroutines, profiler hooks kept), headless.rs + headless/window.rs, keyboard.rs, the helpers of platform.rs (xkb keystroke / modifier / compose / dead-key functions 1302-1602, cursor icon names, open_uri, read_fd_with_timeout, click distance, idle-sleep prevention) with their tests except the display-mode ones, The D-Bus services come from platform/linux/internal/dbus (M1-linux-dbus): LinuxCommon opens the session bus lazily and passes it in. text_system.go returns gpui.NoopTextSystem for now. Tests that reach D-Bus use a private dbus-daemon, never the user's session bus.
- Verify (`sh docs/port/verify.sh M1-linux-common` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/internal/common/... && go vet ./platform/linux/internal/common/... && go test ./platform/linux/internal/common/... && test -z "$(gofmt -l platform/linux/internal/common)"
go run ./internal/cmd/paritycheck -area linux-common
```

### M1-wayland-client

- Goal: Port the first half of the Wayland client: globals, outputs, cursors and the LinuxClient implementation.
- Tier: complex. Risky: no.
- Depends on: M1-wayland-ffi, M1-linux-common, M1-wgpu-context
- Owns: `platform/linux/wayland/`, `docs/port/parity/linux-wayland.md`
- Core task: wayland/client.rs:1-1967 as client.go: registry and globals, outputs and scale, cursor-shape and themed cursors (cursor.rs 152), activation tokens, fractional scale, the LinuxClient implementation and event dispatch into windows, and the start of the seat handling up to line 1967; serial.rs (169, 5 tests), display.rs (77, 1 test), wayland.rs (42). Bindings come from internal/linux/wl and internal/linux/wlproto. Methods the second half defines and window methods the client calls are declared in a package-local stubs.go. Tests that need a compositor are gated by GPUI_TEST_DISPLAY=1.
- Verify (`sh docs/port/verify.sh M1-wayland-client` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/wayland/... && go vet ./platform/linux/wayland/... && go test ./platform/linux/wayland/... && test -z "$(gofmt -l platform/linux/wayland)"
go run ./internal/cmd/paritycheck -area linux-wayland -done gpui_linux/src/linux/wayland/serial.rs,gpui_linux/src/linux/wayland/display.rs,gpui_linux/src/linux/wayland/cursor.rs
```

### M1-wayland-input

- Goal: Port the second half of the Wayland client: seats, input, IME and data devices.
- Tier: complex. Risky: no.
- Depends on: M1-wayland-client
- Owns: `platform/linux/wayland/`, `docs/port/parity/linux-wayland.md`
- Core task: wayland/client.rs:1968-3454 into client_input.go: pointer with axis and gestures, keyboard with xkb (internal/linux/xkb), repeat and compose, touch, text-input-v3 IME, data device drag and drop, primary selection, and the 10 tests of client.rs. Replace the client half of the package stubs.go.
- Verify (`sh docs/port/verify.sh M1-wayland-input` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/wayland/... && go vet ./platform/linux/wayland/... && go test ./platform/linux/wayland/... && test -z "$(gofmt -l platform/linux/wayland)"
```

### M1-wayland-window

- Goal: Port the Wayland window, layer shell, popups and clipboard.
- Tier: complex. Risky: no.
- Depends on: M1-wayland-input, M1-wgpu-renderer
- Owns: `platform/linux/wayland/`, `docs/port/parity/linux-wayland.md`
- Core task: wayland/window.rs (2627, 6 tests): WaylandWindow implementing gpui.PlatformWindow (embedding PlatformWindowDefaults), xdg toplevel / popup / layer surfaces (layer_shell.rs, popup.rs), configure and frame callbacks, decorations, tiling and window controls, resize / move, fractional scaling and viewport, blur, input regions, renderer lifecycle through wgpu.NewRenderer with a Wayland RawWindowHandle, device-loss recovery; `Close` destroys the renderer before the surface and the window's proxies. clipboard.rs (257). a11y.go: a small seam (`newA11yAdapter` returning nil) that M3-exit wires to the AT-SPI adapter. Remove the package stubs.go.
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
- Core task: x11/client.rs (3219, 11 tests) on jezek/xgb plus internal/xext: setup, atoms, XInput2 devices with scroll valuators and gestures, xkb state and compose, randr displays (display.rs), event loop source, event translation (event.rs 154, 1 test), xim_handler.rs (133) over internal/xim, cursors, the LinuxClient implementation; screen capture sources go through gpui.ScapScreenSources (client.rs:1688), which resolves to the registered back end at run time. Window methods it calls go in a package-local stubs.go. Tests that need an X server are gated by GPUI_TEST_DISPLAY=1.
- Verify (`sh docs/port/verify.sh M1-x11-client` runs exactly these, from the worktree root):

```sh
go build ./platform/linux/x11 && go vet ./platform/linux/x11 && go test ./platform/linux/x11 && test -z "$(gofmt -l platform/linux/x11)"
go run ./internal/cmd/paritycheck -area linux-x11 -done gpui_linux/src/linux/x11/event.rs,gpui_linux/src/linux/x11/xim_handler.rs
```

### M1-x11-window

- Goal: Port the X11 window and clipboard.
- Tier: complex. Risky: yes.
- Depends on: M1-x11-client, M1-wgpu-renderer
- Owns: `platform/linux/x11/`, `docs/port/parity/linux-x11.md`
- Core task: x11/window.rs (2098): X11Window implementing gpui.PlatformWindow, window creation and properties (EWMH, Motif hints, _NET_WM_SYNC), client-side decoration support, resize / move requests, visuals and transparency, the wgpu surface through internal/linux/xcbshim (second libxcb connection, DESIGN 8.1): the window id is created and mapped over xgb, then xgb is flushed and a round trip (GetInputFocus) is made before the surface is created, so the second connection sees the window; teardown destroys the renderer and surface first, then the window, and only then calls xcbshim Disconnect; present / frame scheduling. x11/clipboard.rs (1287, a port of ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/x11-clipboard-0.9.3 semantics: selections, INCR transfers, targets). a11y.go seam as in the Wayland window. Remove the package stubs.go.
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
for t in darwin/arm64 windows/amd64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet ./platform && GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go test -c -o /dev/null ./platform || exit 1; done
go run ./internal/cmd/paritycheck -area linux-platform,linux-common,platform-select -done gpui_linux/src/,gpui_platform/src/
```

### M1-linux-screencap

- Goal: Port the Linux screen-capture back end: the ScreenCast portal with PipeWire on Wayland, and the X11 engine.
- Tier: complex. Risky: yes.
- Depends on: M1-linux-platform, M1-linux-dbus
- Owns: `internal/linux/pw/`, `platform/linux/internal/screencap/`, `platform/linux/screencap.go`, `docs/port/parity/ext-scap-linux.md`
- Core task: Port the Linux engines of zed-scap (~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/zed-scap-0.0.8-zed, the published copy of the rev the zed workspace pins: src/targets/linux, src/capturer/engine/linux/{wayland,x11}, src/frame) as upstream builds it with the `x11` feature: the Wayland path (xdg-desktop-portal ScreenCast CreateSession / SelectSources / Start / OpenPipeWireRemote over platform/linux/internal/dbus, then a PipeWire stream on the returned fd) and the X11 path (root-window capture through MIT-SHM GetImage, falling back to plain GetImage, over jezek/xgb). internal/linux/pw: purego bindings over libpipewire-0.3.so.0 for exactly the calls the scap pipewire engine makes (pw_init, thread loop, context connect_fd, stream new / connect / dequeue / queue buffer, the stream-events callbacks as purego callbacks, and the SPA pod builder for the format params it sends), with C layouts checked by tests. platform/linux/internal/screencap implements the gpui.ScapBackend seam (targets, start, frames as BGRA buffers with the same size and stride rules as scap); platform/linux/screencap.go registers it from `init`. Create ext-scap-linux.md with a row per public scap item and engine function used. Tests that need a portal, a PipeWire daemon or an X server are gated by GPUI_TEST_DISPLAY=1 and use only the test's own session (dbus-run-session); the portal dialog cannot be answered headlessly, so the interactive check is an owner check written to docs/port/owner-checks/M1-linux-screencap.md, not a `blocked:owner` row.
- Verify (`sh docs/port/verify.sh M1-linux-screencap` runs exactly these, from the worktree root):

```sh
go build ./internal/linux/pw/... && go vet ./internal/linux/pw/... && go test ./internal/linux/pw/... && test -z "$(gofmt -l internal/linux/pw)"
go build ./platform/linux/internal/screencap/... && go vet ./platform/linux/internal/screencap/... && go test ./platform/linux/internal/screencap/... && test -z "$(gofmt -l platform/linux/internal/screencap)"
go vet ./platform/linux && test -z "$(gofmt -l platform/linux/screencap.go)"
go run ./internal/cmd/paritycheck -area ext-scap-linux -all-done
```

### M1-exit

- Goal: Close M1: hello_world opens on Wayland and X11 with quads drawn by wgpu, checked by screenshot.
- Tier: complex. Risky: yes.
- Depends on: M0-exit, M1-linux-platform, M1-linux-screencap, M4-web-webgl
- Owns: `examples/hello_world/`, `internal/cmd/shoot/`, `docs/port/screenshots/m1/`, `*.go (root package, fixes only)`, `platform/platform.go (fixes only)`, `platform/platform_linux.go (fixes only)`, `platform/platform_test.go (fixes only)`, `platform/linux/ (fixes only)`, `internal/linux/ (fixes only)`, `render/ (fixes only)`, `.github/workflows/ci.yml`, `docs/port/OWNER-CHECKS.md`, `docs/port/parity/examples-m2.md`, `TECH_DEBT.md`
- Core task: Port `_ref/gpui/examples/hello_world.rs` to examples/hello_world. Write internal/cmd/shoot: starts an example binary, waits for its window (`hyprctl clients -j` by pid), captures only that window with grim, terminates the process, always cleans up (no leftover windows or headless outputs), `-x11` runs it with WAYLAND_DISPLAY unset, `-may-exit` accepts a program that exits 0 before a window appears (no image written), `-no-window <duration>` is for programs that never open a window (success when the process is still alive at the deadline or exited 0 before it; it is then terminated), a timeout fails the run. shoot runs under the user's compositor only to capture its own child's window; it never creates headless outputs. Run hello_world on Wayland and through XWayland, Read both PNGs and check them (window, coloured quads, no clipping; text is absent until M2). Fix whatever the first end-to-end run exposes in root, platform/linux, render/wgpu. Run the wgpu and platform tests with GPUI_TEST_DISPLAY=1. Fold docs/port/owner-checks/*.md into docs/port/OWNER-CHECKS.md. Commit finished stages, push, check CI.
- Verify (`sh docs/port/verify.sh M1-exit` runs exactly these, from the worktree root):

```sh
go build -o /dev/null ./examples/hello_world && go vet ./examples/hello_world ./internal/cmd/shoot && go test . ./platform/... ./render/... ./internal/linux/...
go test -race . ./scheduler/... ./sumtree/... ./collections/... ./gpuiutil/... ./refineable/... ./internal/taffy/... ./internal/linux/... ./platform/linux/internal/...
go run ./internal/cmd/shoot -pkg ./examples/hello_world -out docs/port/screenshots/m1/hello_world_wayland.png && go run ./internal/cmd/shoot -x11 -pkg ./examples/hello_world -out docs/port/screenshots/m1/hello_world_x11.png
gh pr checks feat/port
```

## M2: text back end, SVG and images on screen

### M2-gotext-layout

- Goal: Port the Linux text system's font database, metrics and line shaping onto go-text/typesetting.
- Tier: complex. Risky: no.
- Depends on: M0-text-system
- Owns: `text/gotext/`, `docs/port/parity/gotext.md`
- Core task: Package gotext implementing gpui.PlatformTextSystem from `_ref/gpui_wgpu/src/cosmic_text_system.rs` (1900 LOC, 30 tests): constructors `NewTextSystem(systemFontFallback string)` and `NewTextSystemWithoutSystemFonts(...)`, AddFonts, AllFontNames, FontID resolution with family / weight / style matching and the same fallback order (fontscan), FontMetrics, TypographicBounds, Advance, GlyphForChar, and LayoutLine (shaping with go-text shaping + bidi, font fallback per run, cluster to UTF-8 byte index mapping, emoji runs) producing gpui.LineLayout identical in structure to upstream. Port the tests that cover these paths; glyph rasterisation methods return an error until the next stage.
- Verify (`sh docs/port/verify.sh M2-gotext-layout` runs exactly these, from the worktree root):

```sh
go build ./text/gotext/... && go vet ./text/gotext/... && go test ./text/gotext/... && test -z "$(gofmt -l text/gotext)"
```

### M2-gotext-raster

- Goal: Port glyph raster bounds and outline rasterisation (monochrome and LCD subpixel masks).
- Tier: complex. Risky: no.
- Depends on: M2-gotext-layout
- Owns: `text/gotext/`, `docs/port/parity/gotext.md`
- Core task: GlyphRasterBounds and RasterizeGlyph for outline glyphs with subpixel variants, scale factor and the same output buffer formats and bounds rounding as cosmic_text_system.rs, on go-text outlines. Coverage: port the fill path of zeno 0.3.3 (~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/zeno-0.3.3/src: mask.rs, raster.rs, segment.rs, geometry.rs and the path-builder subset they use, about 2.5k lines; no stroker, no SVG parser) into text/gotext/internal/zeno: the monochrome alpha mask, and the LCD subpixel mask exactly as zeno's mask.rs `Format::CustomSubpixel([0.3, 0.0, -0.3])` does it (three coverage passes of the outline shifted by those x offsets, written to channels 0, 1 and 2, keeping upstream's note about the B / R swap the renderer expects); it is not an oversample-and-filter scheme. Glyph pixels may still differ from swash where the outline extraction differs: register L-GLYPH on those rows and keep tests that assert exact pixels as structural checks (size, format, non-empty coverage, channel order). Add rows for the ported zeno files to gotext.md under their own upstream-file headings. Colour glyphs return the outline path until M2-gotext-color.
- Verify (`sh docs/port/verify.sh M2-gotext-raster` runs exactly these, from the worktree root):

```sh
go build ./text/gotext/... && go vet ./text/gotext/... && go test ./text/gotext/... && test -z "$(gofmt -l text/gotext)"
```

### M2-gotext-color

- Goal: Port colour glyph rendering and finish the Linux text system.
- Tier: complex. Risky: no.
- Depends on: M2-gotext-raster
- Owns: `text/gotext/`, `docs/port/parity/gotext.md`
- Core task: Colour emoji and the rest of the Linux text system. Follow swash 0.2.10's source order in scale/mod.rs as upstream configures it: ColorOutline (COLR v0 layers painted with CPAL colours, as swash scale/color.rs), then ColorBitmap with BestFit strike selection (CBDT and sbix PNG strikes decoded with image/png and scaled as swash scale/bitmap does), then Outline, producing the polychrome sprite format the atlas expects. Port the remaining tests of cosmic_text_system.rs's 30, and `_ref/gpui_wgpu/benches/layout_line.rs` as BenchmarkLayoutLine_*. Add rows for the ported swash parts to gotext.md under their own upstream-file headings.
- Verify (`sh docs/port/verify.sh M2-gotext-color` runs exactly these, from the worktree root):

```sh
go build ./text/gotext/... && go vet ./text/gotext/... && go test ./text/gotext/... && test -z "$(gofmt -l text/gotext)"
go run ./internal/cmd/paritycheck -area gotext -done gpui_wgpu/
```

### M2-exit

- Goal: Close M2: text, SVG and images render on screen through the atlas.
- Tier: complex. Risky: yes.
- Depends on: M1-exit, M2-gotext-color
- Owns: `platform/linux/internal/common/text_system.go`, `examples/internal/support/`, `examples/text/`, `examples/svg_example/`, `examples/image_example/`, `docs/port/screenshots/m2/`, `*.go (root package, fixes only)`, `platform/platform.go (fixes only)`, `platform/platform_linux.go (fixes only)`, `platform/platform_test.go (fixes only)`, `platform/linux/ (fixes only)`, `internal/linux/ (fixes only)`, `render/ (fixes only)`, `text/ (fixes only)`, `.github/workflows/ci.yml`, `docs/port/parity/examples-m2.md`, `docs/port/parity/linux-common.md`
- Core task: Wire gotext into the Linux platform (text_system.go). Port example_support/fonts.rs to examples/internal/support and the examples text.rs, svg/ (with dragon.svg) and image/ (with its five assets). Screenshot hello_world, text, svg_example and image_example on Wayland with internal/cmd/shoot; Read every PNG and check text is sharp and correctly positioned, SVG and images are present, nothing is clipped. Fix atlas upload, glyph positioning, gamma and sprite issues wherever they are. Commit finished stages, push, check CI.
- Verify (`sh docs/port/verify.sh M2-exit` runs exactly these, from the worktree root):

```sh
go build -o /dev/null ./examples/... && go vet ./examples/... ./text/... ./platform/linux/internal/common && go test . ./text/... ./platform/... ./render/...
for e in hello_world text svg_example image_example; do go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/m2/$e.png || exit 1; done
go run ./internal/cmd/paritycheck -area examples-m2 -all-done
gh pr checks feat/port
```

## M3: root stages after M2-exit (the root chain continues while the examples run)

### M3-profiler-journal

- Goal: Port the profiler journal.
- Tier: complex. Risky: no.
- Depends on: M2-exit
- Owns: `profiler_journal.go`, `profiler_journal_test.go`, `stubs.go`, `docs/port/parity/core-profiler.md`
- Core task: profiler/journal.rs (3544, 34 items, 27 tests): the journal data model, ring buffers, frame / task / action records, serialisation and the queries the inspector and overlay use. Keep the on-disk / wire layout byte-compatible with upstream where the file defines one; tests assert it. Journal scope guards (Drop impls in journal.rs) become values whose `End` the caller defers.
- Verify (`sh docs/port/verify.sh M3-profiler-journal` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler/journal.rs
```

### M3-bench-context

- Goal: Port the bench runtime and BenchAppContext.
- Tier: complex. Risky: no.
- Depends on: M3-profiler-journal
- Owns: `app_bench_context.go`, `app_bench_context_test.go`, `bench.go`, `bench_focus_cleanup_test.go`, `stubs.go`, `docs/port/parity/core-profiler.md`, `docs/port/parity/core-tests.md`, `docs/port/parity/macros.md`
- Core task: bench.rs (65) + gpui_macros/src/bench.rs (304) as `RunBench(b *testing.B, ...)` with upstream's seed handling (D-BENCH), app/bench_context.rs (1614, 6 tests), gpui/benches/focus_cleanup.rs (141) as BenchmarkFocusCleanup_* in an external test package.
- Verify (`sh docs/port/verify.sh M3-bench-context` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-profiler,core-tests -done gpui/src/bench.rs,gpui/src/app/bench_context.rs,gpui/benches/
```

### M3-randomized-tree

- Goal: Port the randomized element tree.
- Tier: complex. Risky: no.
- Depends on: M3-bench-context
- Owns: `randomized_element_tree.go`, `randomized_element_tree_test.go`, `stubs.go`, `docs/port/parity/core-tests.md`, `docs/port/parity/core-geometry.md`, `docs/port/parity/core-window.md`
- Core task: randomized_element_tree.rs (2058, 6 tests: seeded generators for element trees and interaction scripts, used by the platform benches).
- Verify (`sh docs/port/verify.sh M3-randomized-tree` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go test -run xxx -tags gpui_release -bench FocusCleanup -benchtime 1x .
go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/randomized_element_tree.rs
```

### M3-profiler-hang

- Goal: Port hang detection and the debug frame overlay.
- Tier: complex. Risky: no.
- Depends on: M3-randomized-tree
- Owns: `profiler_hang.go`, `profiler_hang_test.go`, `debug_overlay.go`, `debug_overlay_test.go`, `stubs.go`, `docs/port/parity/core-profiler.md`
- Core task: profiler/hang.rs (1865, 21 tests: hang records, stack capture via runtime.Callers, thresholds, reporting) and debug_overlay.rs (428, 6 tests: frame-time and input-latency overlay drawn by the window). Replace their stubs; window.go's overlay calls now hit real code.
- Verify (`sh docs/port/verify.sh M3-profiler-hang` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler/hang.rs,gpui/src/debug_overlay.rs
```

### M3-root-capture

- Goal: Port the screen-capture glue, the visual test platform and inspector reflection, rewrite the upstream docs for Go, close the remaining root rows and delete stubs.go.
- Tier: complex. Risky: no.
- Depends on: M3-profiler-hang
- Owns: `inspector_reflection.go`, `macros_derive_inspector_reflection_test.go`, `platform_scap_screen_capture.go`, `platform_scap_screen_capture_test.go`, `platform_visual_testing.go`, `platform_visual_testing_test.go`, `stubs.go`, `docs/accessibility.md`, `docs/ownership-and-data-flow.md`, `docs/contexts.md`, `docs/key-dispatch.md`, `docs/port/parity/macros.md`, `docs/port/parity/core-platform.md`, `docs/port/parity/core-geometry.md`, `docs/port/parity/core-profiler.md`
- Core task: platform/scap_screen_capture.rs (325): scap_screen_sources, start_scap_default_target_source and the ScreenCaptureSource / ScreenCaptureStream implementations, on top of the ScapBackend seam declared in M0-platform-types (the Linux back end is already registered by M1-linux-screencap; Windows and macOS register theirs in M4); with no back end registered they return upstream's unsupported error. platform/visual_test.rs (280): VisualTestPlatform wrapping any real platform with a TestDispatcher, clipboard and find pasteboard, every method as upstream; its macOS-only tests are ported by M4-macos-platform. inspector_reflection.go: reflect over the Styled method set at run time in place of derive_inspector_reflection (gpui_macros 305 LOC) with its test (133). Rewrite `_ref/gpui/docs/contexts.md`, key_dispatch.md and the two doc modules for the Go API under docs/ (lowercase; run the de-slop skill on them). Mark arena.rs rows n/a:D-ARENA, hot_reload.rs n/a:D-HOTRELOAD, prelude.rs done by alias note. This is the last root stage before M3-exit: delete stubs.go, nothing may be left in it.
- Verify (`sh docs/port/verify.sh M3-root-capture` runs exactly these, from the worktree root):

```sh
go build . && go vet . && go test . && test -z "$(gofmt -l *.go)"
test ! -e stubs.go && test ! -e DOCS && test ! -e docs/DOCS
go run ./internal/cmd/paritycheck -area core-platform,macros -done gpui/src/platform/scap_screen_capture.rs,gpui/src/platform/visual_test.rs
go run ./internal/cmd/paritycheck -area core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,core-profiler,macros
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

### M3-a11y-atspi-node

- Goal: Port the AT-SPI platform node layer.
- Tier: complex. Risky: no.
- Depends on: M3-a11y-consumer-text
- Owns: `a11y/atspi/`, `docs/port/parity/ext-a11y-atspi.md`
- Core task: Package a11y/atspi from ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_atspi_common-0.19.1/src: node.rs (PlatformNode: role and state mapping, the Accessible / Component / Action / Text / Value / Selection interface bodies), util.rs, rect.rs, text_attributes.rs, action.rs, error.rs, callback.rs (`AdapterCallback` interface, so no D-Bus is needed here) and filters.rs, with their unit tests. Adapter-level types node.rs uses are declared in a package stubs.go. Create ext-a11y-atspi.md with rows for all of accesskit_atspi_common.
- Verify (`sh docs/port/verify.sh M3-a11y-atspi-node` runs exactly these, from the worktree root):

```sh
go build ./a11y/atspi/... && go vet ./a11y/atspi/... && go test ./a11y/atspi/... && test -z "$(gofmt -l a11y/atspi)"
```

### M3-a11y-atspi-adapter

- Goal: Port the platform-neutral AT-SPI adapter state and event generation.
- Tier: complex. Risky: no.
- Depends on: M3-a11y-atspi-node
- Owns: `a11y/atspi/`, `docs/port/parity/ext-a11y-atspi.md`
- Core task: From ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_atspi_common-0.19.1/src: adapter.rs (Adapter state, event generation from tree changes through the consumer ChangeHandler), simplified.rs, context.rs (AppContext and the per-adapter context), events.rs, lib.rs, with their unit tests. Remove the package stubs.go.
- Verify (`sh docs/port/verify.sh M3-a11y-atspi-adapter` runs exactly these, from the worktree root):

```sh
go build ./a11y/atspi/... && go vet ./a11y/atspi/... && go test ./a11y/atspi/... && test -z "$(gofmt -l a11y/atspi)"
go run ./internal/cmd/paritycheck -area ext-a11y-atspi -done accesskit_atspi_common-0.19.1/
```

### M3-a11y-atspi-bus

- Goal: Port the AT-SPI D-Bus adapter that Linux windows use.
- Tier: complex. Risky: yes.
- Depends on: M3-a11y-atspi-adapter
- Owns: `a11y/atspi/`, `docs/port/parity/ext-a11y-atspi.md`
- Core task: From ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_unix-0.22.1/src (2532) on godbus/dbus/v5 v5.2.4 (package a11y/atspi/bus): a11y bus discovery (org.a11y.Bus, AT_SPI_BUS_ADDRESS override), registration with the registry, the exported interfaces Accessible, Application, Component, Action, Text, Value, Selection and Cache, event emission, activation tracking, and the public `Adapter` with `NewAdapter(activation, action, deactivation)`, `UpdateIfActive`, `UpdateWindowFocusState`, `SetRootWindowBounds` matching the calls in `_ref/gpui_linux/src/linux/wayland/window.rs` and `x11/window.rs`. The async executor becomes goroutines confined behind the adapter. Tests run against a private dbus-daemon started by the test (skip when the binary is missing); never the user's a11y bus.
- Verify (`sh docs/port/verify.sh M3-a11y-atspi-bus` runs exactly these, from the worktree root):

```sh
go build ./a11y/atspi/... && go vet ./a11y/atspi/... && go test ./a11y/atspi/... && test -z "$(gofmt -l a11y/atspi)"
go run ./internal/cmd/paritycheck -area ext-a11y-atspi -all-done
```

### M3-platform-benches

- Goal: Port the gpui_platform benches.
- Tier: complex. Risky: no.
- Depends on: M2-exit, M3-randomized-tree, M3-bench-context
- Owns: `platform/bench_atlas_upload_test.go`, `platform/bench_inspector_render_test.go`, `platform/bench_primitives_test.go`, `platform/bench_randomized_element_tree_test.go`, `docs/port/parity/platform-select.md`
- Core task: `_ref/gpui_platform/benches/*.rs` (2589 LOC, 27 bench tests) as external-package benchmarks (`package platform_test`) with gpui.RunBench: atlas upload, inspector render, primitives, randomized element tree, on the headless renderer; they skip without a GPU adapter. Names per rule N14 (BenchmarkRandomizedElementTree_*). The root package is still being written by M3-profiler-hang and M3-root-capture while this runs; only add files under platform/.
- Verify (`sh docs/port/verify.sh M3-platform-benches` runs exactly these, from the worktree root):

```sh
go vet ./platform && go test -run xxx -bench Benchmark -benchtime 1x ./platform && test -z "$(gofmt -l platform/bench_atlas_upload_test.go platform/bench_inspector_render_test.go platform/bench_primitives_test.go platform/bench_randomized_element_tree_test.go)"
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
E="active_state_bug anchor grid_layout opacity gradient pattern scrollable tree uniform_list list_example data_table paths_bench popover"; for e in $E; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-a/$e.png || exit 1; done
go run ./internal/cmd/paritycheck -area examples-a -all-done
```

### M3-examples-b

- Goal: Port and run examples: text_layout, text_wrapper, focus_visible, tab_stop, drag_drop, mouse_pressure, a11y, svg_animation, ownership_post, shadow, painting.
- Tier: complex. Risky: no.
- Depends on: M2-exit
- Owns: `examples/text_layout/`, `examples/text_wrapper/`, `examples/focus_visible/`, `examples/tab_stop/`, `examples/drag_drop/`, `examples/mouse_pressure/`, `examples/a11y/`, `examples/svg_animation/`, `examples/ownership_post/`, `examples/shadow/`, `examples/painting/`, `docs/port/screenshots/examples-b/`, `docs/port/parity/examples-b.md`
- Core task: One `package main` per upstream example, a straight port of its .rs file(s) with assets copied next to it; examples with upstream tests keep them as Go tests. For each: build, run with internal/cmd/shoot, Read the PNG and check it against what the upstream source draws. A bug outside examples/ is not fixed here: add a failing repro and an entry to docs/port/handoff/<stage>.md. Examples gated by cfg upstream keep the gate (build constraint plus the upstream fallback message on other OSes). This group is text, focus, input devices and painting (about 2.7k LOC). The a11y example only needs to run here; its tree is checked in M3-exit. ownership_post opens no window and never quits (application().run with assertions, ownership_post.rs:20-47): run it with `shoot -no-window 5s`, which passes when the process is still alive at the deadline or exited 0.
- Verify (`sh docs/port/verify.sh M3-examples-b` runs exactly these, from the worktree root):

```sh
E="text_layout text_wrapper focus_visible tab_stop drag_drop mouse_pressure a11y svg_animation shadow painting"; for e in $E; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-b/$e.png || exit 1; done
go build -o /dev/null ./examples/ownership_post && go vet ./examples/ownership_post && test -z "$(gofmt -l examples/ownership_post)" && go run ./internal/cmd/shoot -no-window 5s -pkg ./examples/ownership_post
go run ./internal/cmd/paritycheck -area examples-b -all-done
```

### M3-examples-c

- Goal: Port and run examples: window, window_movable, window_positioning, window_shadow, layer_shell, on_window_close_quit, move_entity_between_windows, set_menus, system_notifications, toggle_headless, gif_viewer, image_gallery, image_loading, native_view, native_view_appkit_field.
- Tier: complex. Risky: no.
- Depends on: M2-exit
- Owns: `examples/window/`, `examples/window_movable/`, `examples/window_positioning/`, `examples/window_shadow/`, `examples/layer_shell/`, `examples/on_window_close_quit/`, `examples/move_entity_between_windows/`, `examples/set_menus/`, `examples/system_notifications/`, `examples/toggle_headless/`, `examples/gif_viewer/`, `examples/image_gallery/`, `examples/image_loading/`, `examples/native_view/`, `examples/native_view_appkit_field/`, `docs/port/screenshots/examples-c/`, `docs/port/parity/examples-c.md`
- Core task: One `package main` per upstream example, a straight port of its .rs file(s) with assets copied next to it; examples with upstream tests keep them as Go tests. For each: build, run with internal/cmd/shoot, Read the PNG and check it against what the upstream source draws. A bug outside examples/ is not fixed here: add a failing repro and an entry to docs/port/handoff/<stage>.md. Examples gated by cfg upstream keep the gate (build constraint plus the upstream fallback message on other OSes). This group is windows, system integration and images (about 3.3k LOC). layer_shell is Wayland-only; native_view and native_view_appkit_field are macOS-only and print upstream's message elsewhere; system_notifications runs under `dbus-run-session --`, so its notification calls go to a private session bus with no notification daemon and fail the way upstream reports a missing daemon, never reaching the user's; set_menus is only visible on macOS. `shoot -may-exit` accepts a program that exits 0 before showing a window.
- Verify (`sh docs/port/verify.sh M3-examples-c` runs exactly these, from the worktree root):

```sh
E="window window_movable window_positioning window_shadow on_window_close_quit move_entity_between_windows set_menus gif_viewer image_gallery image_loading native_view native_view_appkit_field"; for e in $E; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -may-exit -pkg ./examples/$e -out docs/port/screenshots/examples-c/$e.png || exit 1; done
for e in layer_shell toggle_headless; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -no-window 5s -pkg ./examples/$e || exit 1; done
go build -o /dev/null ./examples/system_notifications && go vet ./examples/system_notifications && test -z "$(gofmt -l examples/system_notifications)" && dbus-run-session -- go run ./internal/cmd/shoot -may-exit -pkg ./examples/system_notifications -out docs/port/screenshots/examples-c/system_notifications.png
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
E="view_example testing input animation"; for e in $E; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-d/$e.png || exit 1; done
go run ./internal/cmd/paritycheck -area examples-d -all-done
```

### M3-exit

- Goal: Close M3: every example runs on Linux, accessibility is wired, handoffs are fixed, M0-M3 parity is complete.
- Tier: complex. Risky: yes.
- Depends on: M3-examples-a, M3-examples-b, M3-examples-c, M3-examples-d, M3-a11y-atspi-bus, M3-root-capture, M3-platform-benches, M4-web-example, M4-ios-window
- Owns: `platform/linux/wayland/a11y.go`, `platform/linux/x11/a11y.go`, `*.go (root package, fixes only)`, `platform/platform.go (fixes only)`, `platform/platform_linux.go (fixes only)`, `platform/platform_test.go (fixes only)`, `platform/linux/ (fixes only)`, `internal/linux/ (fixes only)`, `render/ (fixes only)`, `text/ (fixes only)`, `a11y/*.go (fixes only)`, `a11y/consumer/ (fixes only)`, `a11y/atspi/ (fixes only)`, `examples/ (fixes only)`, `.github/workflows/ci.yml`, `docs/port/screenshots/`, `docs/port/PARITY.md`, `docs/port/OWNER-CHECKS.md`, `TECH_DEBT.md`, `AGENT.md`
- Core task: Wire a11y/atspi.NewAdapter into the Wayland and X11 windows (a11y.go seams) as upstream wires accesskit_unix. Verify examples/a11y by walking its exported tree with a godbus client over a private a11y bus. Work through every docs/port/handoff/*.md entry (fix, then re-run that example's screenshot and Read it). Check that docs/port/screenshots has one checked image per example. Linux `go test ./...`, `go test -race` over the non-display packages, the three cross-vets with `go test -c -o /dev/null`, `go generate` cleanliness, paritycheck for all M0-M3 areas. Fold debt files into TECH_DEBT.md and owner-check files into docs/port/OWNER-CHECKS.md, update PARITY.md statuses and AGENT.md. Commit finished M0-M3 stages, push, CI green on all three runners for what is committed (done web stages are committed with it; the Apple, iOS and Windows paths are not committed yet, see Shared decision 26).
- Verify (`sh docs/port/verify.sh M3-exit` runs exactly these, from the worktree root):

```sh
go vet ./... && go test ./... && test -z "$(gofmt -l .)"
go test -race $(go list ./... | grep -vE '/(examples|render/wgpu|internal/cmd/wlspike)(/|$)|/platform/linux/(wayland|x11)(/|$)')
s=$(find . -name '*_gen.go' -not -path './internal/apple/*' -not -path './internal/win/*' -not -path './platform/apple/*' -not -path './platform/macos/*' -not -path './platform/windows/*' | sort | xargs sha256sum) && go generate . ./a11y/... ./internal/taffy/... ./internal/linux/... ./platform/linux/... && [ "$s" = "$(find . -name '*_gen.go' -not -path './internal/apple/*' -not -path './internal/win/*' -not -path './platform/apple/*' -not -path './platform/macos/*' -not -path './platform/windows/*' | sort | xargs sha256sum)" ]
X='internal/(apple|win)|platform/(apple|macos|ios|windows)|a11y/(macos|windows)'; for t in darwin/arm64 windows/amd64 js/wasm; do P=$(GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go list ./... | grep -vE "$X"); GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet $P && GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go test -c -o /dev/null $P || exit 1; done
go run ./internal/cmd/paritycheck -area foundation,sumtree,scheduler,macros,core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,core-profiler,core-tests,examples-m2,examples-a,examples-b,examples-c,examples-d,platform-select,linux-common,linux-dbus,linux-platform,linux-wayland,linux-x11,wgpu,gotext,ext-scap-linux,web -all-done -allow-blocked -lint -rescan
gh pr checks feat/port
```

## M4: Apple (macOS, iOS)

### M4-apple-ffi

- Goal: Lay the Apple FFI foundation on purego/objc and port the dispatcher, display link, process info and keychain.
- Tier: core. Risky: yes.
- Depends on: M0-text-system
- Owns: `internal/apple/objc/`, `internal/apple/cf/`, `internal/apple/gcd/`, `platform/apple/`, `docs/port/parity/apple.md`
- Core task: Bindings live under the repo-root internal/apple/ so platform/apple, platform/macos, platform/ios and a11y/macos can all import them (build constraint darwin, which includes ios; no cgo). internal/apple/objc: class and selector caches, typed msgSend helpers, class registration with methods and protocols, blocks, autorelease pools, NSString / NSArray / NSDictionary / NSData / NSURL / NSError bridging, CGPoint / CGSize / CGRect / NSRange struct passing on arm64 and amd64. internal/apple/cf (CoreFoundation types and retain / release with explicit Release, D-RAII) and internal/apple/gcd (main and global queues by QoS, dispatch_async_f, dispatch_after, purego callbacks). Then package platform/apple: dispatcher.rs (201: gpui.PlatformDispatcher on GCD with the profiler hooks; the root dispatcher types exist since M0-text-system and the executors are real from M0-executors), display_link.rs (104), process_info.rs (182, 1 test), gpui_apple.rs, keychain.rs (283, 4 tests: SecItem add / copy / update / delete; tests use a unique service name and remove only their own items). Every binding file gets a size / selector smoke test. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-apple-ffi` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./internal/apple/objc/... ./internal/apple/cf/... ./internal/apple/gcd/... ./platform/apple/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/apple/objc/... ./internal/apple/cf/... ./internal/apple/gcd/... ./platform/apple/... || exit 1; done && test -z "$(gofmt -l internal/apple/objc internal/apple/cf internal/apple/gcd platform/apple)"
```

### M4-apple-text

- Goal: Port the CoreText text system.
- Tier: complex. Risky: no.
- Depends on: M4-apple-ffi, M0-executors
- Owns: `internal/apple/ct/`, `internal/apple/cg/`, `platform/apple/`, `docs/port/parity/apple.md`
- Core task: text_system.rs (1065, 6 tests) and open_type.rs (171) with internal/apple/ct (CoreText: font descriptors, collections, CTLine / CTRun, font features, fallbacks) and internal/apple/cg (bitmap contexts, colour spaces, paths): font loading and matching, metrics, LayoutLine with UTF-16 to UTF-8 index mapping, glyph raster bounds, rasterisation of monochrome, subpixel and emoji glyphs into the formats the atlas expects. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-apple-text` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./internal/apple/ct/... ./internal/apple/cg/... ./platform/apple/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/apple/ct/... ./internal/apple/cg/... ./platform/apple/... || exit 1; done && test -z "$(gofmt -l internal/apple/ct internal/apple/cg platform/apple)"
go run ./internal/cmd/paritycheck -area apple -done gpui_apple/src/text_system.rs,gpui_apple/src/open_type.rs,gpui_apple/src/dispatcher.rs,gpui_apple/src/keychain.rs
```

### M4-apple-metal-atlas

- Goal: Bind Metal, port the Metal atlas and load the shaders.
- Tier: complex. Risky: yes.
- Depends on: M4-apple-text, M0-etagere, M0-scene
- Owns: `internal/apple/mtl/`, `internal/cmd/metalhdr/`, `platform/apple/`, `docs/port/parity/apple.md`
- Core task: internal/apple/mtl: MTLDevice, command queue / buffer, render pass descriptors and encoders, render pipeline and depth state, textures, buffers, samplers, library and function objects, CAMetalLayer, all through objc message sends. metal_atlas.rs (572, 6 tests): MetalAtlas implementing gpui.PlatformAtlas over internal/etagere. shaders.metal copied byte for byte and embedded, sha256 test against `_ref`. Upstream's runtime_shaders path (gpui_apple/build.rs:28-120) compiles `scene.h + "\n" + shaders.metal`, where scene.h is generated by cbindgen 0.28.0 from the scene types; internal/cmd/metalhdr runs that cbindgen (Installs) with the same config (language C, the build.rs include list and renames) over `_ref/gpui/src` and writes platform/apple/shaders_scene_gen.go holding the header text (`//go:generate`, skipped with a message when cbindgen or `_ref` is missing); a test checks every struct offset and size in the header against unsafe.Offsetof / Sizeof of the Go scene types. At start the renderer compiles header + "\n" + shaders.metal with newLibraryWithSource (DESIGN 8.3); a CI test compiles it and looks up every vertex / fragment function the renderer names. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-apple-metal-atlas` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./internal/apple/mtl/... ./platform/apple/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/apple/mtl/... ./platform/apple/... || exit 1; done && test -z "$(gofmt -l internal/apple/mtl platform/apple internal/cmd/metalhdr)"
go build ./internal/cmd/metalhdr/... && go vet ./internal/cmd/metalhdr/... && go test ./internal/cmd/metalhdr/... && test -z "$(gofmt -l internal/cmd/metalhdr)"
```

### M4-apple-metal-renderer

- Goal: Port the Metal renderer.
- Tier: complex. Risky: no.
- Depends on: M4-apple-metal-atlas
- Owns: `platform/apple/`, `docs/port/parity/apple.md`
- Core task: metal_renderer.rs (1903, 3 tests): MetalRenderer satisfying render.Renderer, layer setup and drawable size, instance buffer pool, one draw path per primitive batch with the scene slices viewed through unsafe.Slice, path rasterisation to the intermediate texture, sprites, surfaces (CVPixelBuffer), transparency and presentation with or without transaction, headless render-to-image used by tests; `Destroy` per the Drop table. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-apple-metal-renderer` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/apple/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/apple/... || exit 1; done && test -z "$(gofmt -l platform/apple)"
go run ./internal/cmd/paritycheck -area apple -all-done
```

### M4-macos-appkit

- Goal: Start the macOS platform: AppKit bindings, events and the keyboard.
- Tier: complex. Risky: no.
- Depends on: M4-apple-ffi
- Owns: `internal/apple/appkit/`, `platform/macos/`, `docs/port/parity/macos.md`
- Core task: platform/macos (`//go:build darwin && !ios`). internal/apple/appkit: the NSApplication, NSWindow, NSView, NSEvent, NSScreen, NSPasteboard, NSMenu, NSCursor, NSColor, NSAppearance wrappers the crate needs, on internal/apple/objc. events.rs (579: NSEvent to gpui.PlatformInput), keyboard.rs (1508: layouts, key equivalents, dead keys, the keyboard mapper), gpui_macos.rs. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-macos-appkit` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./internal/apple/appkit/... ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/apple/appkit/... ./platform/macos/... || exit 1; done && test -z "$(gofmt -l internal/apple/appkit platform/macos)"
```

### M4-macos-display

- Goal: Port macOS displays, the display link, window appearance and the pasteboard.
- Tier: complex. Risky: no.
- Depends on: M4-macos-appkit
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: display.rs (182), display_link.rs (712, 4 tests: CVDisplayLink driven frame callbacks through purego callbacks), window_appearance.rs (36), pasteboard.rs (512, 6 tests; tests use a uniquely named pasteboard, never the general one). Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-macos-display` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
```

### M4-macos-platform

- Goal: Port MacPlatform, system notifications and screen capture.
- Tier: complex. Risky: no.
- Depends on: M4-macos-display, M4-apple-text, M0-keymap
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: platform.rs (1680): MacPlatform implementing gpui.Platform (embedding PlatformDefaults): application delegate class, run loop, menus and dock menu, open / save panels, URL and file opening, credentials through platform/apple keychain, appearance, cursor, clipboard via pasteboard, displays, thermal and power state, text system and dispatcher from platform/apple. system_notifications.rs (415, 1 test), screen_capture.rs (291: ScreenCaptureKit SCShareableContent / SCStream with the sample-buffer delegate as an objc class, returned through Platform.ScreenCaptureSources as upstream does on macOS). Also the macOS-only VisualTestPlatform tests of `_ref/gpui_platform/src/gpui_platform.rs` (`#[cfg(all(test, target_os = "macos"))]`, ignored upstream because they need the main thread) as darwin tests that skip unless GPUI_TEST_DISPLAY=1. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-macos-platform` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
```

### M4-macos-window

- Goal: Port the first half of the macOS window: MacWindow and its state.
- Tier: complex. Risky: no.
- Depends on: M4-macos-platform, M4-apple-metal-renderer
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: window.rs:1-2650: MacWindow implementing gpui.PlatformWindow, window and panel creation, state, titlebar and traffic lights, fullscreen, tabs, sheets and prompts, display-link driven frames, MetalRenderer lifecycle (renderer destroyed before the layer and window are released). The objc class registration and delegate / responder methods of the second half are declared in a package-local file the next stage fills. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-macos-window` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
```

### M4-macos-window-objc

- Goal: Port the second half of the macOS window: the objc view and window classes, IME and drag and drop.
- Tier: complex. Risky: no.
- Depends on: M4-macos-window
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: window.rs:2651-4344: the NSWindow / NSPanel and NSView subclasses registered through objc with every delegate and responder method upstream implements, NSTextInputClient IME, drag and drop, mouse and key event routing into the window callbacks, and the 1 test. Keep all symbols in package macos. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-macos-window-objc` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
```

### M4-macos-native-views

- Goal: Port native view hosting on macOS.
- Tier: complex. Risky: no.
- Depends on: M4-macos-window-objc, M0-window-a11y
- Owns: `platform/macos/`, `docs/port/parity/macos.md`
- Core task: window/native_views.rs (1622, 6 tests): hosting AppKit views inside a gpui window, layering against the Metal layer, hit testing, focus hand-off, used by examples/native_view and native_view_appkit_field. Add a11y.go with a nil-returning adapter seam for the next stage. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-macos-native-views` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)"
go run ./internal/cmd/paritycheck -area macos -all-done
```

### M4-macos-a11y

- Goal: Port the macOS accessibility adapter and wire it into the window.
- Tier: complex. Risky: no.
- Depends on: M4-macos-native-views, M3-a11y-consumer-text
- Owns: `a11y/macos/`, `platform/macos/a11y.go`, `docs/port/parity/ext-a11y-macos.md`
- Core task: Package a11y/macos from ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_macos-0.26.3/src (2571): node.rs (the NSAccessibilityElement subclass: roles, attributes, actions, text), adapter.rs, subclass.rs (injecting into the content view), event.rs, context.rs, patch.rs, util.rs, on a11y/consumer and the objc helpers. Wire it in platform/macos/a11y.go where `_ref/gpui_macos/src/window.rs` uses accesskit_macos. Create ext-a11y-macos.md. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-macos-a11y` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./a11y/macos/... ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./a11y/macos/... ./platform/macos/... || exit 1; done && test -z "$(gofmt -l a11y/macos platform/macos)"
go run ./internal/cmd/paritycheck -area ext-a11y-macos -all-done
```

### M4-ios-app

- Goal: Start the iOS platform: UIKit bindings, application, displays, events and the platform.
- Tier: complex. Risky: no.
- Depends on: M4-apple-metal-renderer, M1-linux-platform
- Owns: `internal/apple/uikit/`, `platform/platform_ios.go`, `platform/ios/`, `docs/port/parity/ios.md`
- Core task: platform/ios (`//go:build ios`) on platform/apple and internal/apple. internal/apple/uikit: the UIApplication, UIScene, UIWindow, UIView, UIScreen, UITouch, UIPasteboard wrappers the crate needs. gpui_ios/src: application.rs (543, 6 tests: the app delegate class and UIApplicationMain entry from a Go main through purego), display.rs, events.rs (128: touches to gpui input), platform.rs (519), ios.rs, gpui_ios.rs. platform/platform_ios.go returns the iOS platform. Pure logic (event translation, state machines) sits in files without the ios constraint so its tests compile and run on darwin in CI; everything else is ported fully. What only a device or simulator can confirm goes to docs/port/owner-checks/M4-ios-app.md as an owner check; no row is `blocked:owner`. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-ios-app` runs exactly these, from the worktree root):

```sh
sh docs/port/ios-vet.sh ./internal/apple/... ./platform/ios/... ./platform && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go test -c -o /dev/null ./platform/ios/... && test -z "$(gofmt -l platform/ios internal/apple/uikit)"
go run ./internal/cmd/paritycheck -area ios -done gpui_ios/src/ios/application.rs,gpui_ios/src/ios/events.rs,gpui_ios/src/ios/platform.rs
```

### M4-ios-window

- Goal: Port the iOS window, text input and the iOS example.
- Tier: complex. Risky: no.
- Depends on: M4-ios-app
- Owns: `platform/ios/`, `docs/port/parity/ios.md`, `examples/ios_ios/`
- Core task: window.rs (1105, 4 tests: UIWindow / UIView with CAMetalLayer and CADisplayLink, MetalRenderer lifecycle), text_input.rs (1111, 2 tests: UITextInput), tests/text_input.rs, and examples/ios_ios with the Xcode wrapper from `_ref/gpui_ios/examples/ios/` adapted to a Go c-archive build (the build itself needs Xcode, so building it is an owner check). Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-ios-window` runs exactly these, from the worktree root):

```sh
sh docs/port/ios-vet.sh ./internal/apple/... ./platform/ios/... ./platform && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go test -c -o /dev/null ./platform/ios/... && test -z "$(gofmt -l platform/ios examples/ios_ios)"
go run ./internal/cmd/paritycheck -area ios -all-done -allow-blocked
```

## M4: Windows

### M4-windows-com

- Goal: Lay the Windows foundation: COM helpers, Win32 procs, dispatcher, displays, vsync and settings.
- Tier: core. Risky: yes.
- Depends on: M0-executors
- Owns: `internal/win/com/`, `internal/win/win32/`, `platform/windows/`, `docs/port/parity/windows.md`
- Core task: Bindings live under the repo-root internal/win/ so platform/windows and a11y/windows can both import them (build constraint windows; no cgo). internal/win/com: IUnknown, vtable calls through syscall.SyscallN, GUIDs, HRESULT to error, BSTR / VARIANT helpers, ref-counted wrappers with explicit Release, and a way to implement COM interfaces in Go (vtables of windows.NewCallback thunks, used later for drop targets, DirectWrite callbacks and UI Automation). internal/win/win32: the user32 / gdi32 / dwmapi / shcore / imm32 / ole32 / shell32 procs and structs x/sys/windows lacks. Then package platform/windows: util.rs, wrapper.rs, dispatcher.rs (191: thread pool + message-window dispatch with the profiler hooks, on gpui's PriorityQueueSender / Receiver from M0-executors, explicit Close per the Drop table), display.rs (244), vsync.rs (119, 2 tests), system_settings.rs (87), gpui_windows.rs. Struct-size tests for every hand-written struct. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-com` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./internal/win/com/... ./internal/win/win32/... ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/win/com/... ./internal/win/win32/... ./platform/windows/... || exit 1; done && test -z "$(gofmt -l internal/win/com internal/win/win32 platform/windows)"
```

### M4-windows-directwrite

- Goal: Port the DirectWrite text system.
- Tier: complex. Risky: no.
- Depends on: M4-windows-com, M0-text-system
- Owns: `internal/win/dwrite/`, `platform/windows/`, `docs/port/parity/windows.md`
- Core task: internal/win/dwrite (factory, font collections and sets, font faces, text layout and analysis, glyph runs, rendering params, the custom text renderer and analysis source / sink implemented in Go) and direct_write.rs (2076, 4 tests): font loading and fallback, metrics, LayoutLine with UTF-16 to UTF-8 mapping, raster bounds, glyph rasterisation incl. colour emoji through the GPU path upstream uses. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-directwrite` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./internal/win/dwrite/... ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/win/dwrite/... ./platform/windows/... || exit 1; done && test -z "$(gofmt -l internal/win/dwrite platform/windows)"
go run ./internal/cmd/paritycheck -area windows -done gpui_windows/src/direct_write.rs,gpui_windows/src/vsync.rs
```

### M4-windows-d3d-atlas

- Goal: Bind Direct3D 11, DXGI and DirectComposition, port devices and the atlas, load the HLSL shaders.
- Tier: complex. Risky: yes.
- Depends on: M4-windows-directwrite, M0-etagere, M0-scene
- Owns: `internal/win/d3d/`, `platform/windows/`, `docs/port/parity/windows.md`
- Core task: internal/win/d3d: D3D11 device / context / buffers / textures / views / states / shaders, DXGI factory / adapter / swap chain, DirectComposition device / target / visual, D3DCompile from d3dcompiler_47. directx_devices.rs (194), directx_atlas.rs (420, 1 test: DirectXAtlas implementing gpui.PlatformAtlas over internal/etagere). shaders.hlsl and alpha_correction.hlsl copied byte for byte and embedded, sha256 test against `_ref`, compiled at start with D3DCompile using the entry points and targets of `_ref/gpui_windows/build.rs:23-201`. shaders.hlsl begins with `#include "alpha_correction.hlsl"`, so D3DCompile gets an ID3DInclude implemented in Go (a vtable of windows.NewCallback thunks over internal/win/com: Open serves the embedded alpha_correction.hlsl by name and fails for any other name, Close frees nothing) in place of upstream's build-time include path; the shader files stay byte-identical; a CI test compiles every entry point. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-d3d-atlas` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./internal/win/d3d/... ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/win/d3d/... ./platform/windows/... || exit 1; done && test -z "$(gofmt -l internal/win/d3d platform/windows)"
```

### M4-windows-directx-renderer

- Goal: Port the DirectX renderer.
- Tier: complex. Risky: no.
- Depends on: M4-windows-d3d-atlas
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: directx_renderer.rs (2083): DirectXRenderer satisfying render.Renderer, swap chain and composition setup, per-primitive pipelines and instance buffers viewed with unsafe.Slice, path rasterisation with MSAA to the intermediate target, sprites, transparency, resize, device-lost handling and recovery, GPU specs; `Destroy` per the Drop table. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-directx-renderer` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-windows-events

- Goal: Port the Windows window procedure handlers and the keyboard.
- Tier: complex. Risky: no.
- Depends on: M4-windows-directx-renderer, M0-keymap
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: events.rs (1844: the window procedure message handlers, mouse / pointer / touch, keyboard and IME composition, non-client hit testing, DPI changes) and keyboard.rs (371, 1 test). Window and platform methods the handlers call are declared in a package-local file that M4-windows-window and M4-windows-platform fill. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-events` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-windows-shell

- Goal: Port direct manipulation, clipboard, dialogs, jump lists and notifications on Windows.
- Tier: complex. Risky: no.
- Depends on: M4-windows-events
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: direct_manipulation.rs (361), clipboard.rs (402), dialog.rs (340, 6 tests), destination_list.rs (207), system_notifications.rs (198). Clipboard tests are gated behind an env switch so CI does not fight other processes. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-shell` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-windows-window

- Goal: Port the Windows window.
- Tier: complex. Risky: no.
- Depends on: M4-windows-shell
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: window.rs (1742, 2 tests): WindowsWindow implementing gpui.PlatformWindow, class registration, creation, state, the drag and drop target as a Go COM object, renderer lifecycle (renderer destroyed before the HWND). Add a11y.go with a nil adapter seam for M4-windows-a11y. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-window` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
```

### M4-windows-platform

- Goal: Port WindowsPlatform.
- Tier: complex. Risky: yes.
- Depends on: M4-windows-window
- Owns: `platform/windows/`, `docs/port/parity/windows.md`
- Core task: platform.rs (2045, 4 tests): WindowsPlatform implementing gpui.Platform (embedding PlatformDefaults), message loop, credentials through the Windows credential manager (tests use a unique target name and delete only it), file dialogs, shell integration, keyboard layout, displays. Remove the package-local declaration file. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-platform` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)"
go run ./internal/cmd/paritycheck -area windows -all-done
```

### M4-windows-a11y-node

- Goal: Port the UI Automation node providers.
- Tier: complex. Risky: no.
- Depends on: M4-windows-com, M3-a11y-consumer-text
- Owns: `a11y/windows/`, `docs/port/parity/ext-a11y-windows.md`
- Core task: Package a11y/windows from ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_windows-0.34.0/src: node.rs (IRawElementProviderSimple / Fragment / FragmentRoot and the pattern providers as Go COM objects over internal/win/com), util.rs, context.rs, filters.rs and the UIA constants, with their unit tests. Types from adapter.rs and text.rs are declared in a package-local file for the next stage. Create ext-a11y-windows.md with rows for all of accesskit_windows. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-a11y-node` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./a11y/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./a11y/windows/... || exit 1; done && test -z "$(gofmt -l a11y/windows)"
```

### M4-windows-a11y

- Goal: Port the rest of the UI Automation adapter and wire it into the window.
- Tier: complex. Risky: no.
- Depends on: M4-windows-a11y-node, M4-windows-platform
- Owns: `a11y/windows/`, `platform/windows/a11y.go`, `docs/port/parity/ext-a11y-windows.md`
- Core task: From ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/accesskit_windows-0.34.0/src: text.rs (ITextProvider / ITextRangeProvider), adapter.rs (WM_GETOBJECT handling, event raising), subclass.rs, lib.rs and the tests under tests/. Wire it in platform/windows/a11y.go where `_ref/gpui_windows/src/window.rs` and `events.rs` use accesskit_windows. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-a11y` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./a11y/windows/... ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./a11y/windows/... ./platform/windows/... || exit 1; done && test -z "$(gofmt -l a11y/windows platform/windows)"
go run ./internal/cmd/paritycheck -area ext-a11y-windows -all-done
```

### M4-windows-screencap

- Goal: Port the Windows screen-capture back end on Windows.Graphics.Capture.
- Tier: complex. Risky: yes.
- Depends on: M4-windows-d3d-atlas
- Owns: `internal/win/wgc/`, `internal/win/screencap/`, `docs/port/parity/ext-scap-windows.md`
- Core task: Port the Windows engine of zed-scap (~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/zed-scap-0.0.8-zed: src/targets/win, src/capturer/engine/win, src/frame) and the parts of windows-capture-1.5.0 (~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/windows-capture-1.5.0/src) it calls. internal/win/wgc: WinRT activation (RoInitialize, RoGetActivationFactory, HSTRING, IInspectable), GraphicsCaptureItem from HMONITOR / HWND through IGraphicsCaptureItemInterop, Direct3D11CaptureFramePool, GraphicsCaptureSession, the FrameArrived handler as a Go COM delegate, IDirect3DDxgiInterfaceAccess, all over internal/win/com and internal/win/d3d. internal/win/screencap implements gpui.ScapBackend (targets, start, frames copied to a staging texture and returned as BGRA with the same size and stride rules as scap); it is registered from platform/platform_windows.go by M4-ci-windows. Where WGC is unavailable (older builds) return upstream's unsupported error. Create ext-scap-windows.md. Tests here run only in CI on the real OS; locally the check is the cross-vet plus `go test -c -o /dev/null` for both architectures, so every test must compile.
- Verify (`sh docs/port/verify.sh M4-windows-screencap` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./internal/win/wgc/... ./internal/win/screencap/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/win/wgc/... ./internal/win/screencap/... || exit 1; done && test -z "$(gofmt -l internal/win/wgc internal/win/screencap)"
go run ./internal/cmd/paritycheck -area ext-scap-windows -all-done
```

## M4: web (scheduled early: it lands before M1-exit, which owns render/ fixes)

### M4-web-wgpu-js

- Goal: Make render/wgpu work in the browser.
- Tier: complex. Risky: no.
- Depends on: M1-wgpu-renderer
- Owns: `render/wgpu/`, `docs/port/parity/wgpu.md`
- Core task: The `_js.go` side of render/wgpu: render/wgpu/internal/gpu/webgpu on oliverbestmann/webgpu's GOOS=js backend (browser WebGPU), the web parts of wgpu_context.rs (WebBackendPreference, PreparedWebGraphics, async adapter / device acquisition bridged to gpui tasks, the fallback decision that picks WebGL2 when WebGPU is missing or the preference asks for it), canvas surface creation and configuration, the shader variant selection for the web. The WebGL2 implementation itself comes in M4-web-webgl; until then the WebGL branch returns an error. Shared files may only change in ways that keep the Linux build and tests identical.
- Verify (`sh docs/port/verify.sh M4-web-wgpu-js` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./render/... && GOOS=js GOARCH=wasm go build ./render/... && GOOS=js GOARCH=wasm go test -c -o /dev/null ./render/wgpu/... && go build ./render/... && go test ./render/... && test -z "$(gofmt -l render)"
```

### M4-web-webgl-shaders

- Goal: Generate the GLSL ES 3.00 shaders the WebGL2 path needs from the embedded WGSL.
- Tier: complex. Risky: no.
- Depends on: M1-wgpu-context
- Owns: `internal/cmd/glslgen/`, `internal/webglsl/`
- Core task: Upstream reaches WebGL2 through wgpu's GL backend, which translates the WGSL to GLSL ES 3.00 with naga at run time. Go has no naga, so translate at generate time with naga-cli built from the wgpu rev upstream pins (Installs; naga source for reference at ~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/naga-30.0.1). internal/cmd/glslgen: for every shader variant render/wgpu concatenates for the web (shaders_webgl.wgsl and the variants wgpu_renderer.rs selects when dual-source blending is off) and every vertex / fragment entry point, run `naga <in.wgsl> <out.vert|frag> --entry-point <ep> --profile es300` with the bind-group to GLSL binding map wgpu's GL backend uses (naga's `binding_map`: group*16+binding for samplers / textures, and std140 uniform blocks named as naga names them), and write internal/webglsl/<variant>_<entry>.{vert,frag}.glsl plus webglsl_gen.go embedding them with a table of entry point -> (vertex source, fragment source, attribute and uniform-block names, texture unit per binding). `//go:generate` in internal/webglsl; when naga is missing the tool exits with a message and the committed files are used. A test checks the table covers every entry point the renderer names and that each source starts with `#version 300 es`.
- Verify (`sh docs/port/verify.sh M4-web-webgl-shaders` runs exactly these, from the worktree root):

```sh
go build ./internal/cmd/glslgen/... && go vet ./internal/cmd/glslgen/... && go test ./internal/cmd/glslgen/... && test -z "$(gofmt -l internal/cmd/glslgen)"
go build ./internal/webglsl/... && go vet ./internal/webglsl/... && go test ./internal/webglsl/... && test -z "$(gofmt -l internal/webglsl)"
command -v naga >/dev/null || exit 0; s=$(find internal/webglsl -name '*.glsl' -o -name '*_gen.go' | sort | xargs sha256sum) && go generate ./internal/webglsl && [ "$s" = "$(find internal/webglsl -name '*.glsl' -o -name '*_gen.go' | sort | xargs sha256sum)" ]
```

### M4-web-webgl

- Goal: Port the WebGL2 fallback: implement the GPU facade on WebGL2 so the renderer runs where WebGPU is missing.
- Tier: complex. Risky: yes.
- Depends on: M4-web-wgpu-js, M4-web-webgl-shaders
- Owns: `render/wgpu/`, `docs/port/parity/wgpu.md`
- Core task: render/wgpu/internal/gpu/webgl2 (`//go:build js && wasm`, syscall/js): an implementation of the render/wgpu/internal/gpu interfaces over a WebGL2RenderingContext, covering exactly what WgpuRenderer and WgpuAtlas call, mapped the way wgpu-hal's GLES backend maps them (~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/wgpu-hal-30.0.1/src/gles; upstream pins wgpu rev cddb2f5, which adds the WebGL canvas-resize fix wgpu#10348 on top of 30.0.1, so the resize path follows that fix): buffers as GL buffers with bufferSubData writes, textures with texStorage2D / texSubImage2D for the atlas formats (R8, RGBA8, BGRA via swizzle as wgpu-hal does), samplers, render pipelines as linked programs from internal/webglsl with the vertex layout and blend state applied per draw, bind groups as texture-unit and uniform-block bindings from the generated table, render passes as framebuffer binds with clear / load ops, instanced draws, and the canvas default framebuffer as the surface. Wire it into the WebGL branch of the web context from M4-web-wgpu-js. Every js.Func is released (Drop table). Tests: logic tests (format mapping, binding table, pipeline state diffing) run under node with go_js_wasm_exec against a fake context; drawing in a real browser is an owner check written to docs/port/owner-checks/M4-web-webgl.md. Rows for the WebGL path are `done` against this package; a behaviour that differs because the binding is wgpu-native v29 cites L-WGPUVER.
- Verify (`sh docs/port/verify.sh M4-web-webgl` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./render/... && GOOS=js GOARCH=wasm go build ./render/... && GOOS=js GOARCH=wasm go test -c -o /dev/null ./render/wgpu/... && go build ./render/... && go test ./render/... && test -z "$(gofmt -l render)"
env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOMODCACHE="$(go env GOMODCACHE)" GOFLAGS="$GOFLAGS" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./render/wgpu/internal/gpu/webgl2/...
go run ./internal/cmd/paritycheck -area wgpu -done gpui_wgpu/src/
```

### M4-web-platform

- Goal: Port the web platform, window, display, dispatcher and HTTP client.
- Tier: complex. Risky: no.
- Depends on: M4-web-wgpu-js, M0-keymap
- Owns: `platform/web/`, `docs/port/parity/web.md`
- Core task: platform/web (`//go:build js && wasm`, syscall/js): platform.rs (846), window.rs (1054: canvas window, resize observer, device pixel ratio, frame requests), display.rs, dispatcher.rs (425: the single-threaded path ported exactly, timers and microtasks; Go wasm has no threads, so the multi-threaded mode rows are `n/a:L-WEBTHREADS` and every code path that upstream shares between the modes follows the single-threaded branch), viewport.rs (268), keyboard.rs, logging.rs, gpui_web.rs, http_client.rs (312: fetch behind gpui.HTTPClient). js.Func values are released explicitly.
- Verify (`sh docs/port/verify.sh M4-web-platform` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./platform/web/... && GOOS=js GOARCH=wasm go test -c -o /dev/null ./platform/web/... && test -z "$(gofmt -l platform/web)"
```

### M4-web-input

- Goal: Port web events and the IME mirror.
- Tier: complex. Risky: no.
- Depends on: M4-web-platform
- Owns: `platform/web/`, `docs/port/parity/web.md`
- Core task: events.rs (1516, 2 tests: DOM pointer / wheel / keyboard / composition events to gpui.PlatformInput) and ime_mirror.rs (618: the hidden textarea that mirrors IME state). Logic is written against a small DOM interface so the tests run under node through go_js_wasm_exec without a DOM.
- Verify (`sh docs/port/verify.sh M4-web-input` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./platform/web/... && env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOMODCACHE="$(go env GOMODCACHE)" GOFLAGS="$GOFLAGS" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/... && test -z "$(gofmt -l platform/web)"
```

### M4-web-text

- Goal: Port the web text system and select the web platform.
- Tier: complex. Risky: no.
- Depends on: M4-web-input, M2-gotext-color, M1-linux-platform
- Owns: `platform/web/`, `docs/port/parity/web.md`, `platform/platform_js.go`
- Core task: text_system.rs (509: gotext.NewTextSystemWithoutSystemFonts plus canvas text), canvas_text.rs (197), canvas_fallback.rs (490, 7 tests), glyph_cache.rs (133, 3 tests), run_replacements.rs (594, 9 tests). platform/platform_js.go now returns the web platform. Tests run under node through go_js_wasm_exec; anything that needs a DOM is written against a small interface.
- Verify (`sh docs/port/verify.sh M4-web-text` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./platform/... && env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOMODCACHE="$(go env GOMODCACHE)" GOFLAGS="$GOFLAGS" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/... && test -z "$(gofmt -l platform/web platform/platform_js.go)"
go run ./internal/cmd/paritycheck -area web -done gpui_web/src/
```

### M4-web-example

- Goal: Port the hello_web example and its test drivers.
- Tier: standard. Risky: no.
- Depends on: M4-web-text, M4-web-webgl
- Owns: `examples/web_hello_web/`, `docs/port/parity/web.md`
- Core task: `_ref/gpui_web/examples/hello_web`: main.rs, dynamic_fonts.rs, font_fallback_test.rs, http_client_test.rs, queue_test.rs as Go files (the three *_test.rs are runtime self-tests, hence the `_testing.go` names), index.html and gallery.js adapted to load a Go wasm binary with wasm_exec.js, the test_*.mjs drivers adapted likewise. Build the wasm binary; run the node drivers that do not need WebGPU. Opening it in a browser (WebGPU and the WebGL2 fallback) is an owner check: write it to docs/port/owner-checks/M4-web-example.md, do not launch a browser. Every Go file of the example carries `//go:build js && wasm`, so `go build -o /dev/null ./examples/...` on Linux skips it.
- Verify (`sh docs/port/verify.sh M4-web-example` runs exactly these, from the worktree root):

```sh
GOOS=js GOARCH=wasm go vet ./examples/web_hello_web && GOOS=js GOARCH=wasm go build -o /dev/null ./examples/web_hello_web && test -z "$(gofmt -l examples/web_hello_web)"
go run ./internal/cmd/paritycheck -area web -all-done -allow-blocked
```

## M4: CI and final exit

### M4-ci-macos

- Goal: Select the macOS platform, push, and make the macOS CI job pass.
- Tier: complex. Risky: yes.
- Depends on: M3-exit, M4-macos-a11y, M4-ios-window
- Owns: `platform/platform_darwin.go`, `platform/apple/ (fixes only)`, `platform/macos/ (fixes only)`, `platform/ios/ (fixes only)`, `internal/apple/ (fixes only)`, `a11y/macos/ (fixes only)`, `.github/workflows/ci.yml`, `docs/port/OWNER-CHECKS.md`
- Core task: platform/platform_darwin.go returns MacPlatform (and the Metal headless renderer for tests). Commit the Apple packages (first time they are pushed), push, then loop on the macos-14 job with `gh run view --log-failed`: vet, the internal/apple, platform/apple and platform/macos tests (selector and struct smoke tests, text system, atlas, shader compile, keychain, pasteboard), root tests on macOS, and `go build -o /dev/null ./examples/...`. Fix in the Apple packages; root or shared-package problems go to docs/port/handoff/M4-ci-macos.md with a repro unless the fix is local to a darwin file. Wait for CI in the background, never in a foreground sleep. Fold docs/port/owner-checks/*.md into docs/port/OWNER-CHECKS.md and list there what only a person with a Mac / iPhone simulator can confirm (hello_world on screen, IME, native views, VoiceOver).
- Verify (`sh docs/port/verify.sh M4-ci-macos` runs exactly these, from the worktree root):

```sh
for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./... || exit 1; done && test -z "$(gofmt -l platform/apple platform/macos platform/ios internal/apple a11y/macos platform/platform_darwin.go)"
gh pr checks feat/port
```

### M4-ci-windows

- Goal: Select the Windows platform, push, and make the Windows CI job pass.
- Tier: complex. Risky: yes.
- Depends on: M4-ci-macos, M4-windows-a11y, M4-windows-screencap
- Owns: `platform/platform_windows.go`, `platform/windows/ (fixes only)`, `internal/win/ (fixes only)`, `a11y/windows/ (fixes only)`, `.github/workflows/ci.yml`, `docs/port/OWNER-CHECKS.md`
- Core task: platform/platform_windows.go returns WindowsPlatform and registers the WGC screen-capture back end (internal/win/screencap) with gpui.RegisterScapBackend. Commit the Windows packages (first time they are pushed), push, loop on the windows-latest job as in M4-ci-macos: vet, platform/windows tests (struct sizes, COM helpers, DirectWrite, atlas, HLSL compile on WARP, dialogs, vsync), root tests on Windows, `go build -o /dev/null ./examples/...`. Fold docs/port/owner-checks/*.md and add the Windows owner checks (hello_world on screen, IME, DPI changes, Narrator) to docs/port/OWNER-CHECKS.md.
- Verify (`sh docs/port/verify.sh M4-ci-windows` runs exactly these, from the worktree root):

```sh
for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./... || exit 1; done && test -z "$(gofmt -l platform/windows internal/win a11y/windows platform/platform_windows.go)"
gh pr checks feat/port
```

### M4-exit

- Goal: Final audit: the definition of done holds.
- Tier: complex. Risky: yes.
- Depends on: M4-ci-windows
- Owns: `docs/port/PARITY.md`, `docs/port/parity/`, `TECH_DEBT.md`, `AGENT.md`, `README.md`, `.github/workflows/ci.yml`, `docs/port/OWNER-CHECKS.md`, `docs/port/STAGES.md`
- Core task: Run the Final verify list. Switch the ci.yml parity step from `-lint` to `-all-done -lint`. `paritycheck -all-done -rescan -lint` must pass with zero `blocked:` rows of any kind; every deviation cites a registered D- or L- id (the accepted limits are L-GLYPH, L-JPEG, L-SVGTEXT, L-WGPUVER and L-WEBTHREADS). PARITY.md section 2 gets done / deviation / n/a counts per area; sections 5-7 match reality. Fold all debt files into TECH_DEBT.md (keeping the L-WGPUVER binding-bump entry), bring AGENT.md up to date, write README.md (what it is, status per platform, how to run an example; run the de-slop skill on it). docs/port/OWNER-CHECKS.md lists every `YOU:` item (on-device and in-browser checks). Commit, push; CI green on the three runners.
- Verify (`sh docs/port/verify.sh M4-exit` runs exactly these, from the worktree root):

```sh
test -z "$(gofmt -l .)" && go vet ./... && go test ./...
s=$(find . -name '*_gen.go' | sort | xargs sha256sum) && go generate ./... && [ "$s" = "$(find . -name '*_gen.go' | sort | xargs sha256sum)" ]
go run ./internal/cmd/paritycheck -all-done -rescan -lint
gh pr checks feat/port
```

- Re-review (Gemini, APPROVED) minors applied: randomized-tree parity area core-profiler; ios -done paths; M3-platform-benches now depends on M4-web-text, M4-ios-app; platform/ios pure-logic files //go:build darwin; dup M0-scene verify removed; DESIGN 6.9 ref.
