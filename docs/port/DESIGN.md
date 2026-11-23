# gpui-go port design

Target: zed-industries/zed @ `6501df15765d66d6c2a2b66b1eb22434ccef3f86`. Module `github.com/megascan/gpui-go`, root package `gpui`, toolchain go1.27.4.

All upstream paths below are relative to `_ref/` (the read-only checkout of `crates/`) and are written `path:line`. Library choices come from `docs/port/RESEARCH-deps.md` and are not re-argued here. The item-level mapping is in `docs/port/PARITY.md` and `docs/port/parity/*.md`.

Every Go signature in section 6 was compiled and run on go1.27.4 as a scratch program before being written down (self-typed mixins, generic methods on generic receivers, method-based inference for `Element[RL, PP]`, `iter.Pull` under `runtime.LockOSThread`, `weak`, `runtime.AddCleanup`, `encoding/json/v2` without a GOEXPERIMENT). Nothing else in this document has been built.

## 1. Scope and what "done" means

The port covers 17 upstream crates. The definition of done is the brief's: PARITY maps every public item and every test, ported tests pass on Linux, CI builds and tests every platform package on ubuntu / macos arm64 / windows, every example in `gpui/examples` runs, and the only parity limits are the documented ones.

Two things in this design go beyond what RESEARCH-deps.md assumed, and both need a decision from the owner before M1 (section 14):

1. The Wayland binding cannot be a pure wire-protocol client. Upstream hands the C `wl_surface*` and `wl_display*` to the renderer (`gpui_linux/src/linux/wayland/window.rs:573-578`, `:1722-1738`) and builds wayland-backend with `client_system` + `dlopen` (`gpui_linux/Cargo.toml:95-99`). Vulkan presentation needs those libwayland objects. The in-repo generator therefore emits bindings over `libwayland-client.so` loaded with purego, not a socket client (section 8.1).
2. The Linux platform needs D-Bus (portal file dialogs and settings via ashpd, notifications via notify-rust, secret service via oo7, AT-SPI via accesskit_unix: `gpui_linux/Cargo.toml:55-93`). RESEARCH-deps.md names no D-Bus library.

Decisions the brief asked this document to settle:

| Question | Decision | Section |
|---|---|---|
| Generic methods vs free functions | Go 1.27 generic methods on concrete types (`Entity[T].UpdateR`, `Context[T].Subscribe`, `App.New`); free functions only where the receiver would be an interface | 5.4, 6.2 |
| `Element` associated types | generic interface `Element[RL, PP]` behind a typed `drawable`; no `any` state | 6.5 |
| Fluent `Styled` chaining | self-typed generic mixins embedded in each element; the 3099 macro methods are generated once, on the mixin | 6.5, 9 |
| Derives and macros | in-repo `go:generate` tools (`stylegen`, `refinegen`, `actiongen`, `wlgen`); embedding and reflection for the rest | 9 |
| Actions | `RegisterAction` from `init`, `encoding/json/v2` | 6.4 |
| Main thread | `runtime.LockOSThread` in `init` of package `platform` | 4 |
| Errors | `error` for `anyhow::Result`, panics where upstream panics | 5.3 |
| Handles and drop | explicit `Release` / `Cancel` / `Unsubscribe`, `weak` for weak handles, `runtime.AddCleanup` as safety net and leak detector | 5.2 |
| Iterators | `iter.Seq` where upstream returns iterators | 5.5 |
| Async | coroutine tasks on `iter.Pull`, one scheduler for tests and production | 5.6 |

Conventions for all stages: no code comments except terse godoc on exported API; gofmt; generated files committed as `*_gen.go`; Apache-2.0 `LICENSE` and a `NOTICE` crediting Zed Industries (GPUI is Apache-2.0, `gpui/Cargo.toml:9`); hacks go to `TECH_DEBT.md` when written.

## 2. Upstream inventory

Counts come from a scanner over the pinned sources (rules in PARITY.md section 9). "Pub items" are syntactically `pub` types, functions, constants, methods of public types, methods of public traits, exported macros and proc macros. "Tests" are functions under `#[test]`, `#[gpui::test]`, `#[crate::test]`, `#[gpui::bench]`.

| Crate | Files | LOC | Pub items | Tests | Role |
|---|---|---|---|---|---|
| gpui | 154 | 109267 | 6071 | 578 | core framework; 3099 of the pub items are macro-generated `Styled` methods |
| gpui_platform | 5 | 2820 | 10 | 30 | platform selection (`gpui_platform/src/gpui_platform.rs:62-82`) |
| gpui_linux | 28 | 17620 | 91 | 55 | Wayland, X11, headless |
| gpui_wgpu | 6 | 6335 | 57 | 48 | wgpu renderer, atlas, cosmic-text system, 4 WGSL files (1868 lines) |
| gpui_macos | 12 | 12007 | 60 | 18 | AppKit platform |
| gpui_apple | 10 | 4750 | 53 | 21 | Metal renderer, CoreText, dispatcher, keychain, `shaders.metal` (1354 lines) |
| gpui_windows | 21 | 13443 | 6 | 20 | Win32, D3D11, DirectWrite, `shaders.hlsl` (1372 lines) |
| gpui_web | 21 | 8064 | 23 | 21 | wasm platform |
| gpui_ios | 10 | 3705 | 11 | 12 | UIKit platform |
| gpui_macros | 15 | 3631 | 19 | 3 | proc macros |
| gpui_util | 2 | 846 | 37 | 2 | helpers |
| gpui_shared_string | 1 | 223 | 4 | 0 | `SharedString` |
| gpui_tokio | 1 | 100 | 6 | 0 | tokio bridge |
| refineable | 2 | 680 | 15 | 0 | `Refineable` + derive |
| scheduler | 5 | 3215 | 100 | 31 | executors, deterministic `TestScheduler` |
| sum_tree | 4 | 3555 | 116 | 11 | B-tree with summaries |
| collections | 3 | 418 | 27 | 19 | `VecMap`, hash aliases |
| Total | 300 | 190679 | 6706 | 869 | |

The gpui crate is one compilation unit whose modules reach into each other through `pub(crate)` (`gpui/src/gpui.rs:28-62`, e.g. `Drawable::request_layout` pops `window.element_id_stack` directly at `gpui/src/element.rs:398-400`). It does not depend on any platform crate; `gpui_platform` picks one (`gpui_platform/src/gpui_platform.rs:62-82`) and examples start from `gpui_platform::application()` (`gpui/examples/hello_world.rs:10`).

## 3. Architecture and package layout

```
examples/*                      one main package per upstream example
   |
platform                        gpui_platform: Application(), Headless(), CurrentPlatform()
   |-- platform/linux           LinuxPlatform, backend selection, headless
   |     |-- wayland, x11       clients and windows
   |     `-- internal/{common,wl,wlgen-output,xkb,dbus}
   |-- platform/macos -> platform/apple
   |-- platform/ios   -> platform/apple
   |-- platform/windows
   `-- platform/web
render, render/wgpu             renderer contract; gpui_wgpu renderer + atlas (cgo: wgpu-native; GOOS=js: browser WebGPU)
text/gotext                     go-text based PlatformTextSystem (replaces cosmic_text_system.rs)
   |
gpui (root)                     the whole gpui crate: app, entities, window, elements, style, text front end,
   |                            platform interfaces, scene, TestPlatform, test contexts
   |-- a11y                     pure-Go AccessKit schema (Role, Action, Node, TreeUpdate)
   |-- scheduler                scheduler crate
   |-- sumtree, collections, refineable, gpuiutil
   `-- internal/{taffy,etagere,lyon,jsonschema}
internal/cmd/{stylegen,refinegen,actiongen,wlgen,paritycheck}
```

| Go package | Upstream | cgo | Build constraint |
|---|---|---|---|
| `gpui` | `gpui/src/**`, `gpui_shared_string` | no | all GOOS |
| `collections`, `gpuiutil`, `refineable`, `sumtree`, `scheduler` | same-named crates | no | all |
| `a11y` | accesskit 0.24 data model (`gpui/src/gpui.rs:5`, `:115`) | no | all |
| `internal/taffy` | taffy =0.15.0 (`../Cargo.toml:932`, features at `gpui/Cargo.toml:108-116`) | no | all |
| `internal/etagere` | etagere 0.2 (`gpui_wgpu/src/wgpu_atlas.rs:2`) | no | all |
| `internal/lyon` | the lyon_path / tiny-skia-path subset `gpui/src/path_builder.rs:3-6` uses | no | all |
| `platform` | `gpui_platform` | no | per-OS files |
| `platform/linux/**` | `gpui_linux` | none of its own (purego); links `render/wgpu` | `linux` |
| `render` | renderer contract (6.9); no upstream counterpart | no | all |
| `render/wgpu` | `gpui_wgpu` minus text | yes on native | `linux || js` (others optional) |
| `text/gotext` | `gpui_wgpu/src/cosmic_text_system.rs` | no | all |
| `platform/apple`, `platform/macos`, `platform/ios` | `gpui_apple`, `gpui_macos`, `gpui_ios` | no (purego/objc) | `darwin`, `ios` |
| `platform/windows` | `gpui_windows` | no | `windows` |
| `platform/web` | `gpui_web` | no | `js && wasm` |
| `examples/<name>` | `gpui/examples/*`, `gpui_web/examples`, `gpui_ios/examples` | inherits | per example |

One upstream source file maps to one Go file by a mechanical rule (PARITY.md section 3), e.g. `gpui/src/app/entity_map.rs` -> `app_entity_map.go`, `gpui/src/elements/div.rs` -> `elements_div.go`. PARITY.md section 4 lists all 300 files.

Why the root package is not split: splitting would force exporting the `pub(crate)` surface or re-exporting hundreds of symbols through aliases, and `gpui.Div`, `gpui.Point`, `gpui.App` must live in one package for the API to read like upstream. Leaf subsystems with no dependency on `App` or `Window` are the ones that leave the root: taffy, etagere, path geometry, the a11y schema, the scheduler.

Test support stays in the root package as ordinary (non-`_test.go`) files, because other packages' tests need `TestAppContext` and `TestPlatform` (upstream gates them with `cfg(any(test, feature = "test-support"))`, `gpui/src/gpui.rs:91-94`, `gpui/src/platform.rs:13-17`). The linker drops them from applications.

## 4. Module boundaries

Import rules, enforced by `go vet` cross-compiles and a paritycheck lint:

1. `gpui` imports only `a11y`, `scheduler`, `sumtree`, `collections`, `refineable`, `gpuiutil`, `internal/*`, the standard library, `go-text/typesetting/segmenter` (line breaking), `xo/okage` (SVG, path rasterisation) and image decoders. It never imports a platform, a renderer, cgo or purego.
2. Platform and renderer packages import `gpui`; nothing imports them except `platform` and examples.
3. `render/wgpu` is the only package that pulls in cgo (through `oliverbestmann/webgpu`). On darwin and windows nothing on the default path needs cgo, so `GOOS=darwin`, `GOOS=windows` and `GOOS=js GOARCH=wasm` `go vet ./...` run from Linux.
4. `platform/linux` owns `LinuxPlatform` and backend selection (`gpui_linux/src/linux/platform.rs:295`, `gpui_linux/src/linux/display_connection.rs:23-66`); `wayland` and `x11` are sub-packages; the state both share (`LinuxCommon`, `gpui_linux/src/linux/platform.rs:91`) moves to `platform/linux/internal/common` to break the cycle.
5. Main thread: `platform` calls `runtime.LockOSThread()` in `init`, so the main goroutine stays on the process main thread; `Application.Run` must be called from it. The root package never locks a thread, so headless tests run on any goroutine.

Ownership for parallel stages: each `docs/port/parity/<area>.md` file belongs to the stage that owns the matching Go paths. The root package gets one writing stage at a time (section 11), because two stages editing one Go package break each other's `go build`.

## 5. Language mapping rules

These rules are the contract between builders who cannot see each other. A builder who needs to break one records it in the area parity file with the deviation id.

### 5.1 Naming (deterministic)

| Upstream | Go |
|---|---|
| type `EntityId`, `GpuSpecs` | same name with initialisms fixed: `EntityID`, `GPUSpecs` (list: ID, URL, URI, HTTP, JSON, UTF8, UTF16, IME, GPU, CPU, UUID) |
| free fn `foo_bar` | `FooBar` |
| free fn whose Go name equals a type in the package (`div`, `point`, `size`, `hsla`, `img`, `svg`, `list`) | `NewDiv`, `NewPoint`, ... (D-NAME); `Type::new` maps to the same `NewType` |
| `rems(x)`, `radians(x)`, `percentage(x)` | Go conversion `Rems(x)` (newtype over a number) |
| method with `self` | `(T).FooBar` |
| associated fn without `self` | `new` -> `NewT`, `new_x` -> `NewTX`, `default` -> `DefaultT`, else `TFooBar` |
| `impl Edges<Length> { fn zero() }` (impl on one non-numeric instantiation) | free function `EdgesLengthZero` |
| trait method | interface method, or mixin method for the five fluent traits (5.4) |
| `SCREAMING_CONST` | `ScreamingConst` |
| digit-digit boundary `w_1_2` vs `w_12` | underscore kept: `W1_2`, `W12` |
| test `fn test_foo` in `keymap.rs` | `TestKeymap_Foo` in `keymap_test.go` |

The scanner applied these rules to all 6706 items and found 22 remaining Go-name clashes, all `#[cfg]` alternatives of one item or a constructor fn plus `::new` (`docs/port/parity/_collisions.md`).

### 5.2 Ownership, handles and Drop (D-RAII)

Upstream leans on `Drop` in four places that carry behaviour:

- `Entity<T>` / `AnyEntity`: clone increments an atomic count, drop decrements, and the last drop queues the id for release at the next effect flush (`gpui/src/app/entity_map.rs:412-466`, `:262-285`, `gpui/src/app.rs:2275-2278`, `:2347`).
- `FocusHandle`: ref-counted in `FocusMap` (`gpui/src/window.rs:497-503`, `:674`).
- `Subscription`: drop unsubscribes, `detach` keeps it forever (`gpui/src/subscription.rs:262-300`).
- `Task`: drop cancels, `detach` lets it finish (`scheduler/src/executor.rs:437`, `:610`).

Go rule:

- A handle value wraps a pointer to a per-clone handle object. Copying the Go value aliases the same handle (a Rust move or borrow). `Clone()` makes a new counted handle. `Release()` is the explicit drop: idempotent per handle, decrements once. Using a released handle panics.
- `runtime.AddCleanup` on the handle object is the safety net: a handle that becomes unreachable without `Release` is decremented from the cleanup, through the same thread-safe queue upstream's `Drop` uses (it also runs off the main thread: `Arc<RwLock<EntityRefCounts>>`, `gpui/src/app/entity_map.rs:60`, `:86-92`). Application code that never calls `Release` therefore gets release-on-GC; code and tests that need upstream's timing call `Release` where upstream drops.
- A strong cycle leaks exactly as upstream: the entity map holds the state, the state holds the handle, the handle stays reachable.
- Entity state that owns handles may implement `interface{ Drop(cx *App) }`; the app calls it when the entity is released, after release observers, where Rust drop glue would run.
- `Subscription` stays active until `Unsubscribe()`; `Detach()` marks it intentional. `Task` runs until `Cancel()`; `Detach()` marks it intentional. No GC-driven cancel or unsubscribe: a task that stops at a random collection is worse than one that keeps running.
- Leak detection is ported 1:1 (`LeakDetector`, `HandleID`, snapshots, `assert_released`, the exit report: `gpui/src/app/entity_map.rs:1041-1228`), on in tests and under `GPUI_LEAK_DETECTION=1`, with `LEAK_BACKTRACE=1` capturing `runtime.Callers`. In strict mode (`GPUI_LEAK_DETECTION=strict`, default in this repo's own tests) the cleanup also reports handles, tasks and subscriptions that were collected without `Release` / `Cancel` / `Detach` / `Unsubscribe`, which is how a missed implicit-drop site in ported code shows up.
- `WeakEntity[T]` holds the id and a `weak.Pointer` to the ref-count table (upstream `Weak<RwLock<EntityRefCounts>>`, `gpui/src/app/entity_map.rs:350`); `Upgrade` is increment-if-not-zero (`:694-703`).
- The entity lease is kept: updating an entity removes it from the map for the duration, and a nested update or read panics with upstream's message (`gpui/src/app/entity_map.rs:166-190`, `:311-312`). Windows are leased the same way (`gpui/src/app.rs:900`, `:2490`). `AppCell`'s `RefCell` (`gpui/src/app.rs:85-211`) becomes a plain `*App`.

### 5.3 Option, Result, panics (D-OPT)

- `anyhow::Result<T>` -> `(T, error)`; `Result<()>` -> `error`. Panics stay panics with the same message.
- Returned `Option<T>` -> `(T, bool)`, or a nil-able Go type when `T` is a pointer, interface, func, slice or map.
- `Option<T>` struct fields -> `Option[T]` (a value type: `Some(v)`, `None[T]()`, `.Get() (T, bool)`; declared in `gpuiutil` so leaf packages can use it, aliased as `gpui.Option`), because refinements are built per element per frame and must not allocate, and because `Some(0)` and `None` differ (`aspect_ratio: Option<f32>`, `gpui/src/style.rs:240`).
- `Task<Result<T>>` and `Task<T>` both -> `*Task[T]` whose `Await` returns `(T, error)` (5.6).

### 5.4 Traits

| Upstream form | Go form |
|---|---|
| object-safe trait used as `dyn` (`Platform`, `PlatformWindow`, `PlatformDispatcher`, `PlatformTextSystem`, `PlatformAtlas`, `InputHandler`, `AssetSource`, `Action`) | interface, 1:1 method set |
| default trait method bodies (`gpui/src/platform.rs:408`, `:1300`) | embeddable `PlatformDefaults`, `PlatformWindowDefaults` structs carrying the upstream bodies |
| fluent extension traits returning `Self`: `Styled`, `InteractiveElement`, `StatefulInteractiveElement`, `ParentElement`, `FluentBuilder` (`gpui/src/styled.rs:22`, `gpui/src/elements/div.rs:792`, `:1353`, `gpui/src/element.rs:197`, `gpui/src/util.rs:11`) | self-typed generic mixins embedded in each element (6.5) |
| trait with associated types (`Element`) | generic interface `Element[RL, PP]` |
| marker traits (`Global`, `EventEmitter<E>`, `Send`, `Sync`, `'static`) | dropped; `EventEmitter` is not checked at compile time (D-EMIT) |
| conditional impls (`impl<E: ParentElement> ParentElement for Stateful<E>`, `gpui/src/elements/div.rs:4281`; `impl<V: Render> IntoElement for Entity<V>`, `gpui/src/view.rs:112`) | method always present, runtime check, panic naming the type (D-COND) |
| generic context bound `C: AppContext` (`gpui/src/gpui.rs:174-250`) | sealed interface with type-erased primitives; typed entry points are generic methods on `Entity[T]` |
| `impl Into<X>` parameters | the concrete type when one source exists; a sealed `IntoX` interface when several do (`IntoLength`, `IntoHsla`, `IntoFill`); a union-constrained generic method for `ElementID` sources (D-INTO) |
| `TypeId`, `type_name` | `reflect.Type` |

Closures that return a generic `R` come in two forms, because Go cannot overload on a void closure: `Update(cx, func(*T, *Context[T]))` and `UpdateR[R](cx, func(*T, *Context[T]) R) R` (D-VOID). The `R` suffix is the only spelling used.

Go generic methods cannot appear in interfaces, so anything that must be called through `AppContext` with its own type parameter is either a generic method on a concrete type or a free function taking the interface.

### 5.5 Enums, strings, iterators

- Field-less enum -> `type X uint8` with constants `XVariant`. Data-carrying enum -> sealed interface with one struct per variant, or a tagged struct when it must be comparable (`ElementID`, `gpui/src/window.rs:8131-8142`, is a map key).
- `SharedString` -> `type SharedString string` where a method-bearing type is needed (it is an element: `gpui/src/elements/text.rs:312`); every `impl Into<SharedString>` parameter takes `string`. Upstream backs it with `SmolStr` for cheap clones (`gpui_shared_string/gpui_shared_string.rs:15`), which a Go string already is. `ArcCow` (`gpui_util/src/arc_cow.rs:9`) is not ported (D-STRING).
- Functions returning iterators return `iter.Seq` / `iter.Seq2`. Parameters that take `impl IntoIterator` take slices or variadics.

### 5.6 Async (D-ASYNC)

Upstream tasks are futures polled by a `Scheduler` (`scheduler/src/scheduler.rs:100-135`); the deterministic `TestScheduler` picks the next runnable with a seeded RNG, one candidate per foreground session, all background runnables, weighted by priority (`scheduler/src/test_scheduler.rs:313-417`, weights at `scheduler/src/scheduler.rs:61-68`).

Go tasks are coroutines built on `iter.Pull`. One poll is one `next()`; the body runs until it parks in a gpui primitive (`Await`, `Timer`, channel receive, `Yield`). Consequences:

- The scheduler keeps upstream's shape: a run queue of runnables, the same candidate selection, a virtual clock. The same code path runs in production (`PlatformScheduler`, `gpui/src/platform_scheduler.rs:27-176`) and in tests.
- `Cancel()` is `stop()`: the parked primitive sees `yield` return false and unwinds the body with a private panic value that the task wrapper recovers, so deferred functions run. This is upstream's "dropping a task cancels it at the next await".
- A foreground task body never holds `*App`. It gets `*AsyncApp` (upstream holds `Weak<AppCell>`, `gpui/src/app/async_context.rs:22-34`); `cx.Update(func(*App))` parks the coroutine with a request, the poller (the main thread, or the test goroutine) runs the closure and resumes the body in the same poll. All app and platform access from tasks therefore happens on the main OS thread even though the body is a different goroutine.
- Await takes the context explicitly: `task.Await(cx)`. Go has no ambient task identity, and the explicit `Awaiter` is what lets one primitive work under the test scheduler, on the main thread and on background goroutines.
- Foreground task bodies must block only through gpui primitives; a raw channel receive there blocks the UI thread, the same rule as blocking inside an async fn upstream.
- The RNG is `math/rand/v2` PCG, not Rust's `StdRng`. A seed gives a deterministic but different interleaving than upstream (D-RNG).
- `futures` channels and combinators map to `scheduler.Oneshot[T]`, `scheduler.Chan[T]`, `scheduler.Select`, `scheduler.Join`.

### 5.7 Things that do not port

| Upstream | Go | Id |
|---|---|---|
| element arena (`gpui/src/arena.rs:81`, `ArenaBox` at `:218`, used by `AnyElement`, `gpui/src/element.rs:658`) | GC plus per-frame slice pools in `Window`; arena tests are n/a | D-ARENA |
| `gpui_tokio` (`gpui_tokio/src/gpui_tokio.rs:12-97`) | not ported; `BackgroundExecutor.Spawn` is the equivalent | D-TOKIO |
| hot reload via subsecond (`gpui/src/gpui.rs:69-70`, `gpui/src/hot_reload.rs`) | not ported | D-HOTRELOAD |
| proptest (`gpui/src/test.rs:43-66`, `sum_tree/src/property_test.rs`) | seeded generators through `gpui.PropertyTest`, no shrinking | D-PROPTEST |
| criterion benches (`gpui_macros/src/gpui_macros.rs:222`, `gpui/src/bench.rs:18-51`) | `testing.B` with the same seed handling | D-BENCH |
| schemars (`gpui/src/action.rs:139`) | `internal/jsonschema` reflector | D-SCHEMA |
| cargo features | `test-support`: always compiled. `inspector` (also on in debug builds, `gpui/src/element.rs:34`): always compiled, runtime switch. `profiler` (`gpui/src/gpui.rs:64-67`): build tag `gpui_profiler`. `leak-detection`: runtime (5.2). `wayland`, `x11`: always compiled on linux | D-FEATURE |
| `http_client` crate (`gpui/src/gpui.rs:6`, `gpui/src/app.rs:3747`) | `gpui.HTTPClient` interface over `net/http` | D-HTTP |

### 5.8 What stays 1:1

Everything that is behaviour rather than Rust idiom is transliterated, file by file, with the same names, the same order of operations and the same panic messages:

| Area | Upstream |
|---|---|
| effect queue, flush loop, notify / emit / defer ordering | `gpui/src/app.rs:2254-2440`, `:3607-3628` |
| entity map, leases, access tracking for view caching | `gpui/src/app/entity_map.rs:57-300` |
| subscriber sets (insert-pending, retain, remove during dispatch) | `gpui/src/subscription.rs:14-260` |
| window draw phases, frames, element state, hitboxes, focus, tooltips, deferred draws | `gpui/src/window.rs:1053-1100`, `:1442`, `:3837-4100`, `:4809`, `:5789` |
| key dispatch tree, pending input, keymap precedence, context predicates | `gpui/src/key_dispatch.rs:71-620`, `gpui/src/keymap.rs:51-274`, `gpui/src/keymap/context.rs:170-440` |
| style cascade and refinement, taffy conversion and rounding | `gpui/src/style.rs`, `gpui/src/taffy.rs:31-690` |
| scene ordering, bounds tree, batching | `gpui/src/scene.rs:53-830`, `gpui/src/bounds_tree.rs` |
| line layout cache, wrapping, truncation, line painting | `gpui/src/text_system/*.rs` |
| every element | `gpui/src/elements/*.rs` |
| test platform and deterministic scheduler algorithm | `gpui/src/platform/test/*.rs`, `scheduler/src/test_scheduler.rs` |
| renderer pipelines, atlas, shaders (verbatim) | `gpui_wgpu/src/*`, `gpui_apple/src/metal_*.rs`, `gpui_windows/src/directx_*.rs` |
| platform event translation, IME, clipboard, window state | the platform crates |

Re-expressed: only what sections 5.1-5.7 list, each with a deviation id.

## 6. Shared contracts

Signatures are abbreviated; the parity files carry every method. These are the declarations the contracts stage (section 11, S2) lands first.

### 6.1 Geometry and color (`geometry.go`, `color.go`)

Upstream `Point<T>`, `Size<T>`, `Bounds<T>`, `Edges<T>`, `Corners<T>` (`gpui/src/geometry.rs:86`, `:397`, `:724`, `:1785`, `:2349`) are used with numeric units and with non-numeric contents (`Point<Overflow>` at `gpui/src/style.rs:190`, `Edges<Length>`, `Size<Length>` and others through `:180-322`). Go cannot attach arithmetic to only some instantiations, so there are two families (D-GEOM):

```go
type Number interface{ ~float32 | ~float64 | ~int32 | ~int64 | ~int | ~uint32 | ~uint64 | ~uint }

type Pixels float32        // gpui/src/geometry.rs:2768
type DevicePixels int32    // :3081
type ScaledPixels float32  // :3174
type Rems float32          // :3349

type Point[T Number] struct{ X, Y T }
type Size[T Number] struct{ Width, Height T }
type Bounds[T Number] struct{ Origin Point[T]; Size Size[T] }
type Edges[T Number] struct{ Top, Right, Bottom, Left T }
type Corners[T Number] struct{ TopLeft, TopRight, BottomRight, BottomLeft T }

func NewPoint[T Number](x, y T) Point[T]
func (p Point[T]) Add(o Point[T]) Point[T]
func (p Point[T]) Scale(factor float32) Point[ScaledPixels]

type PointOf[T any] struct{ X, Y T }   // SizeOf, EdgesOf, CornersOf likewise
func Px(v float32) Pixels

type Length interface{ isLength() }     // Definite | Auto, gpui/src/geometry.rs:3722
type IntoLength interface{ ToLength() Length }
type Hsla struct{ H, S, L, A float32 }  // gpui/src/color.rs:334
type Rgba struct{ R, G, B, A float32 }  // :39
```

Operators become methods (`Add`, `Sub`, `Mul`, `Neg`); the unit types keep native operators because they are defined numeric types. The numeric family must keep the exact field order and 4-byte fields: these structs are copied into GPU buffers (6.8).

### 6.2 Entities, App, contexts (`app*.go`)

```go
type EntityID uint64                     // slotmap key, gpui/src/app/entity_map.rs:28-48

type Entity[T any] struct{ /* *handle */ }
func (e Entity[T]) EntityID() EntityID
func (e Entity[T]) Clone() Entity[T]
func (e Entity[T]) Release()
func (e Entity[T]) Downgrade() WeakEntity[T]
func (e Entity[T]) IntoAny() AnyEntity
func (e Entity[T]) Read(cx *App) *T
func (e Entity[T]) ReadWith[R any](cx AppContext, f func(*T, *App) R) R
func (e Entity[T]) Update(cx AppContext, f func(*T, *Context[T]))
func (e Entity[T]) UpdateR[R any](cx AppContext, f func(*T, *Context[T]) R) R
func (e Entity[T]) UpdateIn(cx VisualContext, f func(*T, *Window, *Context[T]))
func (e Entity[T]) IntoAnyElement() AnyElement      // panics unless *T implements Render[T]

type WeakEntity[T any] struct{ /* id, weak.Pointer[refCounts] */ }
func (w WeakEntity[T]) Upgrade() (Entity[T], bool)
func (w WeakEntity[T]) Update(cx AppContext, f func(*T, *Context[T])) error

type AppContext interface {              // sealed; gpui/src/gpui.rs:174-250
    updateEntity(h AnyEntity, f func(state any, cx *App))
    readEntity(h AnyEntity, f func(state any, cx *App))
    reserveEntity(t reflect.Type) AnyEntity
    insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity
    updateWindow(h AnyWindowHandle, f func(root AnyView, w *Window, cx *App)) error
    BackgroundExecutor() BackgroundExecutor
}
type VisualContext interface{ AppContext; WindowHandle() AnyWindowHandle /* ... */ }

type App struct{ /* gpui/src/app.rs:884-994 */ }
func (cx *App) New[T any](build func(cx *Context[T]) *T) Entity[T]
func (cx *App) Observe[W any](e Entity[W], f func(Entity[W], *App)) Subscription
func (cx *App) Spawn[R any](f func(cx *AsyncApp) (R, error)) *Task[R]
func (cx *App) Defer(f func(cx *App))
func (cx *App) OpenWindow[V any](opts WindowOptions, build func(*Window, *App) Entity[V]) (WindowHandle[V], error)

type Context[T any] struct{ *App /* + WeakEntity[T] */ }    // gpui/src/app/context.rs:20-37
func (cx *Context[T]) Notify()
func (cx *Context[T]) Observe[W any](e Entity[W], f func(*T, Entity[W], *Context[T])) Subscription
func (cx *Context[T]) Subscribe[T2, E any](e Entity[T2], f func(*T, Entity[T2], *E, *Context[T])) Subscription
func (cx *Context[T]) Emit[E any](event E)
func (cx *Context[T]) Spawn[R any](f func(this WeakEntity[T], cx *AsyncApp) (R, error)) *Task[R]
func (cx *Context[T]) Listener[E any](f func(*T, *E, *Window, *Context[T])) func(*E, *Window, *App)

type Subscription struct{ /* ... */ }    // gpui/src/subscription.rs:262
func (s Subscription) Unsubscribe()
func (s Subscription) Detach()
```

`Context[T]` embeds `*App`, which is upstream's `Deref<Target = App>` (`gpui/src/app/context.rs:25-37`); a type in another package that embeds `*App` satisfies `AppContext`, which replaces `#[derive(AppContext)]` (`gpui_macros/src/gpui_macros.rs:59`).

Effects keep upstream's structure: `Effect` variants and the flush loop (`gpui/src/app.rs:3607-3628`, `:2275-2300`), `pending_updates` nesting (`:1456-1474`), `SubscriberSet` semantics (`gpui/src/subscription.rs:14-160`). Events and globals are keyed by `reflect.Type` where upstream keys by `TypeId` (`gpui/src/app.rs:911`, `:947`).

Window (`window.go`) keeps upstream's shape: `Window` owns the rendered and next `Frame`, the dispatch tree, the element-id stack and the platform window (`gpui/src/window.rs:1232-1328`, `Frame` at `:1053`), and is passed next to the context, never inside it.

```go
type DispatchPhase uint8                                   // Bubble, Capture; gpui/src/window.rs:99
func (w *Window) Draw(cx *App) ArenaClearNeeded            // :3837
func (w *Window) RequestLayout(style Style, children []LayoutID, cx *App) LayoutID   // :5703
func (w *Window) WithElementState[S, R any](id *GlobalElementID, f func(state Option[S], w *Window) (R, S)) R   // :4809
func (w *Window) PaintQuad(quad PaintQuad)                 // :5112
func (w *Window) InsertHitbox(bounds Bounds[Pixels], behavior HitboxBehavior) Hitbox  // :5789
func (w *Window) DispatchKeystroke(k Keystroke, cx *App) bool                         // :6068
func (w *Window) DispatchEvent(e PlatformInput, cx *App) DispatchEventResult          // :6110
func (w *Window) Refresh()                                 // :2734
func (w *Window) OnNextFrame(f func(*Window, *App))        // :3076
func (w *Window) RequestAnimationFrame()                   // :3096

type FocusHandle struct{ /* id, *handle; TabIndex int; TabStop bool */ }   // :539-546
type WindowHandle[V any] struct{ /* ... */ }               // :7907
type AnyWindowHandle struct{ /* ... */ }                   // :8051
type ContentMask[T Number] struct{ Bounds Bounds[T] }      // :2542
```

### 6.3 Scheduler and executors (`scheduler/`, `executor.go`)

```go
package scheduler

type Priority uint8                  // High 60, Medium 30, Low 10, RealtimeAudio 0; scheduler.rs:44-69
type SessionID uint16                // scheduler.rs:241
type Instant struct{ /* monotonic */ }
type Clock interface{ UTCNow() time.Time; Now() Instant }     // clock.rs:8-9

type Scheduler interface {           // scheduler.rs:100-160
    Block(session *SessionID, poll func() bool, timeout *time.Duration) bool
    ScheduleLocal(session SessionID, r *Runnable)
    ScheduleBackgroundWithPriority(r *Runnable, p Priority)
    SpawnRealtime(f func())
    Timer(d time.Duration) *Timer
    Clock() Clock
    AsTest() *TestScheduler
}

type Awaiter interface{ scope() *Scope }
type Task[T any] struct{ /* ... */ }                          // executor.rs:440
func (t *Task[T]) Await(cx Awaiter) (T, error)
func (t *Task[T]) Wait(cx Awaiter) error
func (t *Task[T]) Detach()
func (t *Task[T]) Cancel()
func (t *Task[T]) IsReady() bool
func ReadyTask[T any](v T) *Task[T]

type TestScheduler struct{ /* test_scheduler.rs:34 */ }
func NewTestScheduler(cfg TestSchedulerConfig) *TestScheduler   // seed, randomize_order, allow_parking, timeout_ticks: :743-771
func (s *TestScheduler) Run()
func (s *TestScheduler) Tick() bool
func (s *TestScheduler) AdvanceClock(d time.Duration)
func (s *TestScheduler) Rng() *SharedRng
```

`gpui.BackgroundExecutor` and `gpui.ForegroundExecutor` wrap these as upstream does (`gpui/src/executor.rs:19-37`, methods `:87-489`). `TestDispatcher` wraps a `TestScheduler` (`gpui/src/platform/test/dispatcher.rs:17-36`).

### 6.4 Actions and keymap (`action.go`, `keymap*.go`)

```go
type Action interface {              // gpui/src/action.rs:117-166
    Name() string
    BoxedClone() Action
    PartialEq(other Action) bool
}
type ActionData struct {
    Name string; Type reflect.Type
    Build func(json jsontext.Value) (Action, error)
    JSONSchema func() []byte
    DeprecatedAliases []string; DeprecationMessage, Documentation string
}
func RegisterAction(data ActionData)                 // called from init(); replaces inventory, action.rs:282-287
func OnAction[A Action](f func(a A, w *Window, cx *App))   // keyed by reflect.Type

type Keystroke struct{ Modifiers Modifiers; Key string; KeyChar Option[string] }
func KeystrokeParse(s string) (Keystroke, error)
type KeyBinding struct{ /* ... */ }                   // gpui/src/keymap/binding.rs
type Keymap struct{ /* ... */ }                       // gpui/src/keymap.rs:51-274
type KeyContext struct{ /* ... */ }                   // gpui/src/keymap/context.rs:10-150
type KeyBindingContextPredicate interface{ /* ... */ } // :170, parser at :247
```

Action JSON uses `encoding/json/v2` (present in go1.27.4 without an experiment flag); `Build` receives the raw value as upstream's `build(serde_json::Value)` does (`gpui/src/action.rs:134`). `actiongen` writes the three methods plus the `init` registration from a `//gpui:action namespace=...` directive, covering the derive attributes `name`, `namespace`, `no_json`, `no_register`, `deprecated_aliases`, `deprecated` (`gpui_macros/src/derive_action.rs:1-60`). Hand-written actions are equally valid.

### 6.5 Elements (`element.go`, `elements_*.go`, `view.go`)

```go
type Element[RL, PP any] interface {      // gpui/src/element.rs:52-143
    ElementID() (ElementID, bool)          // upstream id(); renamed, clashes with InteractiveElement.id
    SourceLocation() *SourceLocation
    RequestLayout(id *GlobalElementID, insp *InspectorElementID, w *Window, cx *App) (LayoutID, RL)
    Prepaint(id *GlobalElementID, insp *InspectorElementID, bounds Bounds[Pixels], rl *RL, w *Window, cx *App) PP
    Paint(id *GlobalElementID, insp *InspectorElementID, bounds Bounds[Pixels], rl *RL, pp *PP, w *Window, cx *App)
}
type ElementA11y interface{ A11yRole() (a11y.Role, bool); WriteA11yInfo(n *a11y.Node) }   // optional, element.rs:113-138

type IntoElement interface{ IntoAnyElement() AnyElement }
type AnyElement struct{ /* elementObject */ }                 // element.rs:658
func NewAnyElement[RL, PP any](e Element[RL, PP]) AnyElement  // RL, PP inferred from e's methods
func (e *AnyElement) RequestLayout(w *Window, cx *App) LayoutID
func (e *AnyElement) LayoutAsRoot(avail SizeOf[AvailableSpace], w *Window, cx *App) Size[Pixels]
func (e *AnyElement) PrepaintAt(origin Point[Pixels], w *Window, cx *App)
func (e *AnyElement) Paint(w *Window, cx *App)

type Render[T any] interface{ Render(w *Window, cx *Context[T]) IntoElement }   // element.rs:172
type RenderOnce interface{ Render(w *Window, cx *App) IntoElement }             // element.rs:188
type View interface{ EntityID() (EntityID, bool); Render(w *Window, cx *App) IntoElement }  // view.rs:208
```

`AnyElement` wraps an unexported `drawable[RL, PP]` that holds the typed state between phases and enforces the phase order with upstream's panics (`gpui/src/element.rs:306-358`, `:367-412`). Element state stays typed; nothing is boxed in `any`.

Fluent traits are self-typed mixins. Each stores `self S` and returns it, so `NewDiv().Flex().SizeFull()` stays `*Div`:

```go
type Styler interface{ Style() *StyleRefinement }
type Styled[S Styler] struct{ self S }
func (s *Styled[S]) Flex() S                       // 116 hand-written + 3099 generated methods

type Interactive[S any] struct{ self S }
func (i *Interactive[S]) ID[I ElementIDSource](id I) *Stateful[S]
func (i *Interactive[S]) OnClick(f func(*ClickEvent, *Window, *App)) S

type Parent[S any] struct{ self S }
func (p *Parent[S]) Child(child IntoElement) S
func (p *Parent[S]) Children(children ...IntoElement) S
func (p *Parent[S]) ChildText(text string) S       // Go convenience for .child("...")

type Fluent[S any] struct{ self S }
func (f *Fluent[S]) When(cond bool, then func(S) S) S
func (f *Fluent[S]) Map[U any](fn func(S) U) U

type Div struct {
    Styled[*Div]; Interactive[*Div]; Parent[*Div]; Fluent[*Div]
    /* gpui/src/elements/div.rs:1905 */
}
func NewDiv() *Div
type Stateful[E any] struct { Styled[*Stateful[E]]; Interactive[*Stateful[E]]; StatefulInteractive[*Stateful[E]]; Parent[*Stateful[E]]; Fluent[*Stateful[E]] /* div.rs:4176 */ }
```

The scanner checked the five method sets against each other and against `Element`: the only clashes are `id` and `a11y_synthetic_children`, both resolved by prefixing the `Element` method (`ElementID`, `ElementA11ySyntheticChildren`). Types that implement the traits are listed at `gpui/src/elements/div.rs:1973-1995`, `:4180-4281`, `gpui/src/elements/img.rs:509-521`, `gpui/src/elements/svg.rs:230-236`, `gpui/src/elements/uniform_list.rs:257-709`, `gpui/src/elements/list.rs:1660`, `gpui/src/elements/canvas.rs:83`, `gpui/src/elements/surface.rs:122`, `gpui/src/elements/image_cache.rs:77-83`, `gpui/src/elements/container_query.rs:114`, `gpui/src/elements/anchored.rs:106`.

`ElementID` (`gpui/src/window.rs:8131-8142`) is a comparable tagged struct. `GlobalElementID` keeps the precomputed path hash (`gpui/src/element.rs:228-258`); element-state lookup must compare the full path, not only the hash.

### 6.6 Style (`style.go`, `styled.go`, `refineable/`)

```go
package refineable
type Refineable[R any] interface {        // refineable/src/refineable.rs:29-63
    Refine(r *R); Refined(r R) Self; IsSupersetOf(r *R) bool; Subtract(r *R) R
}
type Cascade[S any, R any] struct{ /* :80-131 */ }
```

`Style` keeps upstream's fields in upstream's order (`gpui/src/style.rs:180-322`); `StyleRefinement`, `TextStyleRefinement`, `BoundsRefinement`, `EdgesOfRefinement[T]`, `CornersOfRefinement[T]` are produced by `refinegen` from `//gpui:refineable` directives, matching the five derive sites (`gpui/src/style.rs:178`, `:436`, `gpui/src/geometry.rs:721`, `:1782`, `:2346`) and the nested `#[refineable]` fields. Refinement fields are `Option[T]`.

### 6.7 Layout (`taffy.go`, `internal/taffy`)

`TaffyLayoutEngine` (`gpui/src/taffy.rs:31`) is ported over `internal/taffy`, a line-for-line port of taffy 0.15.0 with the features upstream enables: flexbox, grid, block, float, content size (`gpui/Cargo.toml:108-116`). `LayoutID` wraps the taffy node id (`gpui/src/taffy.rs:396-402`); `AvailableSpace` is gpui's own enum (`:689`). The style conversion keeps upstream's device-pixel rounding (`gpui/src/taffy.rs:130-134`, `:423-440`, `:450-580`). taffy's generated fixture tests are ported with it and listed in PARITY.md section 7.

### 6.8 Scene (`scene.go`)

`Scene` holds one slice per primitive kind plus paint operations and a bounds tree (`gpui/src/scene.rs:53-68`), and batches them for the renderer (`PrimitiveBatch`, `:806-826`). Primitives are `#[repr(C)]` and uploaded as bytes (`:864-1236`). In Go:

```go
type DrawOrder = uint32
type Quad struct {                         // gpui/src/scene.rs:866-875
    Order DrawOrder; BorderStyle BorderStyle
    Bounds Bounds[ScaledPixels]; ContentMask ContentMask[ScaledPixels]
    Background Background; BorderColor Hsla
    CornerRadii Corners[ScaledPixels]; BorderWidths Edges[ScaledPixels]
}
type PrimitiveBatch interface{ isBatch() }  // Shadows, Quads, Paths, Underlines, MonochromeSprites, SubpixelSprites, PolychromeSprites, Surfaces, NativeViews
func (s *Scene) Batches() iter.Seq[PrimitiveBatch]
```

Rule: primitive structs contain only fixed-size 4-byte-aligned fields in upstream order; a test asserts `unsafe.Sizeof` and field offsets against the WGSL, Metal and HLSL struct layouts. Renderers view the slices with `unsafe.Slice`.

### 6.9 Platform interfaces (`platform.go`)

One Go interface per upstream trait, method for method:

| Interface | Upstream | Methods |
|---|---|---|
| `Platform` | `gpui/src/platform.rs:394-680` | 83 |
| `PlatformWindow` | `:1290-1552` | 108 |
| `PlatformDisplay` | `:681` | |
| `PlatformDispatcher` | `:1591-1636` | `IsMainThread`, `Dispatch`, `DispatchOnMainThread`, `DispatchAfter`, `DispatchOnMainThreadWhenIdle`, `SpawnRealtime`, `Now`, ... |
| `PlatformTextSystem` | `:1638-1675` | `AddFonts`, `AllFontNames`, `FontID`, `FontMetrics`, `TypographicBounds`, `Advance`, `GlyphForChar`, `GlyphRasterBounds`, `RasterizeGlyph`, `LayoutLine`, ... |
| `PlatformAtlas`, `AtlasBackend` | `:1887-1912` | `GetOrInsertWith`, `Remove` |
| `PlatformHeadlessRenderer` | `:1555-1575` | `RenderSceneToImage`, `RenderScene`, `SpriteAtlas` |
| `InputHandler` | `:2432` | IME text input |
| `PlatformKeyboardLayout`, `PlatformKeyboardMapper` | `gpui/src/platform/keyboard.rs` | |

```go
type Platform interface {
    PlatformInfo() PlatformInfo
    BackgroundExecutor() BackgroundExecutor
    ForegroundExecutor() ForegroundExecutor
    TextSystem() PlatformTextSystem
    Run(onFinishLaunching func())
    Quit()
    OpenWindow(handle AnyWindowHandle, params WindowParams) (PlatformWindow, error)
    Displays() []PlatformDisplay
    ReadFromClipboard() (ClipboardItem, bool)
    WriteToClipboard(item ClipboardItem)
    OnQuit(callback func() bool)
    /* ... */
}
type PlatformWindow interface {
    RawWindowHandle() RawWindowHandle      // HasWindowHandle + HasDisplayHandle
    Bounds() Bounds[Pixels]
    ScaleFactor() float32
    OnRequestFrame(callback func(RequestFrameOptions))
    OnInput(callback func(PlatformInput) DispatchEventResult)
    Draw(scene *Scene)
    SpriteAtlas() PlatformAtlas
    /* ... */
}
```

Callbacks are plain func values. `Rc<dyn Platform>` is the interface value. The platform-specific methods upstream gates with `cfg` (`get_raw_handle` on Windows, layer-shell and decoration methods on Linux, `gpui/src/platform.rs:1459-1491`) are present on every OS in Go, defaulted by `PlatformWindowDefaults`, so the interface does not vary by GOOS.

Upstream has no cross-platform renderer trait; each window owns its renderer and exposes `draw(&Scene)` (`gpui/src/platform.rs:1417`). The Go contract between platform windows and GPU back ends is the method set of `WgpuRenderer` (`gpui_wgpu/src/wgpu_renderer.rs:299-2174`):

```go
package render   // github.com/megascan/gpui-go/render: interface only, so platform packages do not import a concrete renderer
type Renderer interface {
    Draw(scene *gpui.Scene) bool
    UpdateDrawableSize(size gpui.Size[gpui.DevicePixels])
    UpdateTransparency(transparent bool)
    SetSubpixelLayout(isBGR bool)
    SpriteAtlas() gpui.PlatformAtlas
    SupportsDualSourceBlending() bool
    GPUSpecs() (gpui.GPUSpecs, bool)
    DeviceLost() bool
    Recover(handle gpui.RawWindowHandle) error
    Destroy()
}
```

### 6.10 Text (`text_system*.go`)

The front end ports 1:1: `TextSystem`, `WindowTextSystem`, `Font`, `FontRun`, `TextRun`, `FontMetrics` (`gpui/src/text_system.rs:232`, `:635`, `:1139`, `:1075`, `:1192`), `LineLayout`, `ShapedRun`, `ShapedGlyph`, `WrappedLineLayout`, `LineLayoutCache` (`gpui/src/text_system/line_layout.rs:20-508`), `LineWrapper`, `Boundary`, truncation (`gpui/src/text_system/line_wrapper.rs:30-744`), `ShapedLine`, `WrappedLine` and painting (`gpui/src/text_system/line.rs:43-940`). Line breaking uses `go-text/typesetting/segmenter` where upstream uses `icu_segmenter::LineSegmenter` and `unicode_segmentation` (`gpui/src/text_system/line_wrapper.rs:4`, `:545`, `:777`). String offsets stay UTF-8 byte offsets; UTF-16 ranges stay explicit (`UTF16Selection`, `gpui/src/platform.rs:2419`).

### 6.11 Test harness (`testing.go`, `app_test_context.go`, `platform_testing_*.go`)

```go
func RunTest(t *testing.T, fn func(tc *TestRun), opts ...TestOption)
// options: Seed(n), Seeds(...), Iterations(n), Retries(n), OnFailure(fn)  -- gpui_macros/src/gpui_macros.rs:170-186
// env SEED and ITERATIONS behave as upstream                              -- gpui/src/seeds.rs:15-33
type TestRun struct{ /* ... */ }
func (tc *TestRun) Cx() *TestAppContext               // first context
func (tc *TestRun) NewContext() *TestAppContext       // "as many TestAppContext instances as you need"
func (tc *TestRun) Rand() *rand.Rand                  // the StdRng parameter, gpui_macros/src/test.rs:143
func (tc *TestRun) BackgroundExecutor() BackgroundExecutor   // :147
```

`TestAppContext`, `VisualTestContext`, `TestApp`, `HeadlessAppContext`, `TestPlatform`, `TestWindow`, `TestDisplay`, `TestDispatcher` port method for method (`gpui/src/app/test_context.rs:35`, `:882`, `gpui/src/app/test_app.rs:40`, `gpui/src/app/headless_app_context.rs:38`, `gpui/src/platform/test/platform.rs:29`, `gpui/src/platform/test/window.rs:71`). At the end of each run the harness quits the app, drains scheduler tasks (upstream needs the same, `scheduler/src/test_scheduler.rs:419-425`), runs `runtime.GC` until the live-handle count stops falling, and then reports as upstream's `LeakDetector::drop` does (`gpui/src/app/entity_map.rs:1196-1228`).

### 6.12 Accessibility (`a11y/`)

Elements and the window speak AccessKit types directly (`gpui/src/element.rs:113-138`, `gpui/src/window/a11y.rs`, `gpui/src/platform.rs:1531-1537`). `a11y` is a pure-Go copy of the AccessKit 0.24 schema (`Role`, `Action`, `Node`, `NodeID`, `TreeUpdate`, `Orientation`, `Toggled`), re-exported from `gpui` by alias as upstream re-exports them (`gpui/src/gpui.rs:115`). Platform adapters translate `a11y.TreeUpdate`; the core never links accesskit-c.

## 7. Subsystems behind the contracts

| Subsystem | Upstream | Go | Notes |
|---|---|---|---|
| Layout engine | taffy 0.15.0 | `internal/taffy` port | largest self-contained port (RESEARCH: 15-20k Go LOC); owns its fixture tests |
| Atlas allocator | etagere `BucketedAtlasAllocator` (`gpui_wgpu/src/wgpu_atlas.rs:2`, `gpui/src/platform.rs:2114-2120`) | `internal/etagere` port | shared by wgpu, Metal and D3D atlases |
| Paths | lyon_path builder (`gpui/src/path_builder.rs:3-6`), `path_winding.rs` | `internal/lyon` subset + okage path | tessellation output must match vertex for vertex |
| SVG | resvg / usvg / tiny-skia (`gpui/src/svg_renderer.rs:8`, `:328-357`) | xo/okage | text in SVG is a known limit |
| Images | `image` crate decode, GIF and WebP frames (`gpui/src/platform.rs:3376-3522`) | stdlib png/jpeg/gif, x/image/webp, gen2brain/avif | JPEG is a known limit |
| Linux text | cosmic-text + swash (`gpui_wgpu/src/cosmic_text_system.rs:3-18`, `:24`; Linux re-exports it, `gpui_linux/src/linux/text_system.rs:1`) | `text/gotext` on go-text/typesetting | glyph pixels are a known limit |
| Mac text | CoreText (`gpui_apple/src/text_system.rs:63`) | same API via purego | |
| Windows text | DirectWrite (`gpui_windows/src/direct_write.rs:36`) | same API via COM | |

## 8. Platforms

### 8.1 Linux (`platform/linux`)

- Event loop: upstream runs calloop with the Wayland or X11 source and a priority queue for main-thread runnables (`gpui_linux/src/linux/dispatcher.rs:20-232`, `gpui/src/queue.rs:144-206`). Go: the main goroutine selects over the display fd (netpoller), the runnable queue and timers; background runnables go to goroutines.
- Wayland: `wlgen` reads protocol XML and emits typed Go proxies that marshal through `wl_proxy_marshal_array_flags` and receive events through a dispatcher callback, both on `libwayland-client.so.0` loaded with purego. This keeps real `wl_display*` / `wl_surface*` pointers for the Vulkan surface (section 1). Protocols needed, from `gpui_linux/src/linux/wayland/client.rs`: core, xdg-shell, xdg-decoration, xdg-activation, xdg-dialog, xdg-system-bell, fractional-scale, viewporter, cursor-shape, text-input-v3 (`:57-61`), primary-selection, pointer-gestures, linux-dmabuf feedback, wlr-layer-shell, kde blur. Cursor themes use `libwayland-cursor` through purego.
- X11: jezek/xgb for the protocol. The wgpu surface needs an `xcb_connection_t*` (upstream passes its own, `gpui_linux/src/linux/x11/window.rs:358`, `:766`); Go opens a second libxcb connection through purego only for surface creation, using the XID of the window created over xgb. XIM is hand-written (`gpui_linux/src/linux/x11/xim_handler.rs`).
- xkbcommon through purego (`gpui_linux/src/linux/wayland/client.rs:82-83`, `gpui_linux/src/linux/platform.rs:1302-1602`).
- D-Bus for portals, notifications, secrets, power and AT-SPI (`gpui_linux/src/linux/xdg_desktop_portal.rs`, `system_notifications.rs`, `power.rs`): open item, section 14.
- Headless backend ports as is (`gpui_linux/src/linux/headless.rs`, `headless/window.rs`).

### 8.2 wgpu renderer (`render/wgpu`)

Port of `WgpuRenderer`, `WgpuAtlas`, `WgpuContext` (`gpui_wgpu/src/wgpu_renderer.rs:253`, `gpui_wgpu/src/wgpu_atlas.rs:23`, `gpui_wgpu/src/wgpu_context.rs:13`). The four WGSL files are embedded byte for byte with `//go:embed` and concatenated as upstream does (`gpui_wgpu/src/wgpu_renderer.rs:25-39`); subpixel text needs dual-source blending and falls back when the device lacks it (`:37-44`, `:1069`). `WgpuHeadlessRenderer` (`:2278`) backs screenshot tests.

### 8.3 macOS and iOS (`platform/apple`, `platform/macos`, `platform/ios`)

purego/objc for AppKit and UIKit classes, delegates and blocks; Metal called through objc with `shaders.metal` embedded and compiled at start with `newLibraryWithSource:` (upstream ships a prebuilt metallib and has a `runtime_shaders` feature for this path, `gpui_apple/src/metal_renderer.rs:37-38`, `gpui_apple/Cargo.toml:19`); CoreText text system; GCD dispatcher (`gpui_apple/src/dispatcher.rs`); keychain. iOS is built as far as Go's `GOOS=ios` support allows; the Xcode wrapper project from `gpui_ios/examples/ios/` is carried over.

### 8.4 Windows (`platform/windows`)

`x/sys/windows` plus hand-written COM vtables for D3D11, DXGI, DirectComposition and DirectWrite; `shaders.hlsl` and `alpha_correction.hlsl` embedded and compiled at start with `D3DCompile`, where upstream runs `fxc.exe` in `build.rs` (`gpui_windows/build.rs:23-201`).

### 8.5 Web (`platform/web`)

`GOOS=js GOARCH=wasm`, `syscall/js` for DOM events, IME mirror element and canvas, `render/wgpu` on browser WebGPU. Go's wasm is single-threaded, which matches upstream's `single_threaded_web` mode (`gpui_platform/src/gpui_platform.rs:48-51`); the `allow_multi_threading` mode (`gpui_web/src/platform.rs:147-156`) has no Go equivalent.

## 9. Macros and code generation

| Upstream macro | Source | Go replacement |
|---|---|---|
| `style_helpers!` and the eight `*_style_methods!` | `gpui_macros/src/styles.rs:40-492`, tables `:699-1417` | `internal/cmd/stylegen` emits `styled_gen.go`: 3099 methods on `Styled[S]`. The expected list is `docs/port/parity/styled-methods.tsv`; a test compares the generated method set to it |
| `#[derive(Refineable)]` | `refineable/derive_refineable/src/derive_refineable.rs` | `internal/cmd/refinegen` from `//gpui:refineable` |
| `#[derive(Action)]`, `actions!`, `register_action!` | `gpui_macros/src/derive_action.rs`, `gpui/src/action.rs:24`, `gpui_macros/src/register_action.rs:41` | `internal/cmd/actiongen` or hand-written methods + `RegisterAction` in `init` |
| `#[derive(IntoElement)]` | `gpui_macros/src/derive_into_element.rs` | one hand-written method: `func (c C) IntoAnyElement() AnyElement { return NewComponent(c) }` |
| `#[derive(Render)]` | `gpui_macros/src/derive_render.rs` | hand-written empty `Render` |
| `#[derive(AppContext)]`, `#[derive(VisualContext)]` | `gpui_macros/src/derive_app_context.rs`, `derive_visual_context.rs` | embed `*App` / the context |
| `#[gpui::test]` | `gpui_macros/src/test.rs` | `gpui.RunTest` (6.11) |
| `#[gpui::bench]`, `bench_group!`, `bench_main!` | `gpui_macros/src/bench.rs`, `gpui/src/bench.rs:18-51` | `gpui.RunBench(b *testing.B, ...)` |
| `#[gpui::property_test]` | `gpui_macros/src/property_test.rs` | `gpui.PropertyTest` |
| `derive_inspector_reflection` | `gpui_macros/src/derive_inspector_reflection.rs`, applied at `gpui/src/styled.rs:17-21` | `reflect` over the `Styled` method set at run time |
| Wayland protocol code (wayland-scanner in the Rust crates) | n/a | `internal/cmd/wlgen` |

Generated files are committed, end in `_gen.go`, and CI fails if `go generate ./...` changes the tree.

## 10. Test strategy and CI

- Every upstream test is a row in a parity file with its Go name. Tests are ported with the code they cover, in the same stage.
- Headless tests use `TestPlatform` and the deterministic scheduler and need no display or GPU.
- Renderer tests use `WgpuHeadlessRenderer` when a Vulkan device (lavapipe in CI) exists and skip otherwise, as upstream's `current_headless_renderer` returns `None` off macOS and Linux (`gpui_platform/src/gpui_platform.rs:102`).
- Examples are verified on Linux by running each one and capturing a screenshot; the image is opened and checked before it is committed.
- Stage verify: `go build ./<pkgs>/... && go vet ./<pkgs>/... && go test ./<pkgs>/... && test -z "$(gofmt -l <paths>)"`.
- Final verify: `gofmt -l .`, `go vet ./...`, `go test ./...` on Linux, `go generate ./... && git diff --exit-code`, `go run ./internal/cmd/paritycheck`, then `GOOS=darwin GOARCH=arm64`, `GOOS=windows` and `GOOS=js GOARCH=wasm` `CGO_ENABLED=0 go vet ./...`.
- CI (`.github/workflows/ci.yml`, created by S0): matrix ubuntu-latest, macos-14 (arm64), windows-latest; each runs gofmt, vet, test for its GOOS; ubuntu also runs the cross-vets, the generate check and paritycheck. The PR is opened at the end of M0 so the other two runners compile from then on.
- `paritycheck` verifies that every upstream item found by the scanner has a row, every row marked `done` names a Go symbol or test that exists, every `dev:`/`n/a:` id is registered, and no Go file violates the import rules in section 4.

## 11. Milestones and stage plan

Three lanes. Lane A is the only lane that writes the root package. Lanes B and C take packages that do not import each other's in-progress work.

| Stage | Lane | Owns | Needs |
|---|---|---|---|
| S0 bootstrap | A | `go.mod`, `LICENSE`, `NOTICE`, `AGENT.md`, `TECH_DEBT.md`, `.github/workflows/ci.yml`, `internal/cmd/paritycheck`, `doc.go` | - |
| S1 foundation | B | `collections/`, `gpuiutil/`, `refineable/`, `internal/cmd/refinegen` | S0 |
| S2 scheduler | C | `scheduler/` with all 31 tests | S0 |
| S3 contracts | A | root: `geometry.go`, `color.go`, `option.go`, `shared_string.go`, and compiling declarations for sections 6.2-6.12 (interfaces, data types, method stubs that panic "unimplemented") ; `a11y/` | S1 |
| S4 sumtree | B | `sumtree/` | S1 |
| S5 taffy | C (B joins after S4) | `internal/taffy/` (style + tree + block, then flexbox, then grid), fixtures | S0 |
| S6 app core | A | entity map, `App`, contexts, subscriptions, globals, executors, `AsyncApp`, actions, assets, `TestPlatform`, `TestDispatcher`, `TestAppContext` | S2, S3 |
| S7 input | A | keystroke, keymap, key context, key dispatch tree, interactive events | S6 |
| S8 style and scene | A | `style.go`, `styled.go` + `stylegen`, taffy bridge, `scene.go`, bounds tree, paths (`internal/lyon` in lane B) | S5, S6 |
| S9 window | A | window, element drawing, hitboxes, focus, tab stops, views, input handler, `VisualTestContext`, a11y tree | S7, S8 |
| S10 text front end | A | text system, layout cache, wrapper, shaped lines | S9 |
| S11 elements | A | all of `gpui/src/elements`, gestures, spring, prompts, inspector | S10 |
| M0 exit | | all ported headless tests pass; PR opened | |
| S12 linux base | B | `platform/`, `platform/linux/internal/*`, `wlgen`, xkb, dispatcher, headless | S3 |
| S13 wgpu | C | `render/`, `render/wgpu`, `internal/etagere` | S3, S8 |
| S14 wayland, S15 x11 | B, C | `platform/linux/wayland`, `platform/linux/x11` | S12, S13 |
| M1 exit | | `examples/hello_world` window and quads on screen under Hyprland, screenshot checked; its text appears at M2 exit | |
| S16 text back end | B | `text/gotext` | S3 |
| S17 svg, images, atlas use, paths | A | `svg_renderer.go`, image decode, `img`, `svg` elements end to end | S13 |
| M2 exit | | text, SVG and images render | |
| S18 examples | A, B, C | `examples/*` split three ways by name | M2 |
| S19 inspector, a11y adapters, profiler, benches | A | remaining root files; AT-SPI adapter in lane B | M2 |
| M3 exit | | every example runs on Linux | |
| S20 macOS, S21 Windows, S22 web, S23 iOS | B, C, A | `platform/apple`, `platform/macos`, `platform/windows`, `platform/web`, `platform/ios` | M3; verified by CI |

S3 is the stage that makes the rest parallel: after it, a platform or renderer stage compiles against real declarations while lane A fills in bodies. Lane A stages are sequential; if the planner wants two root stages at once, their file sets must be disjoint and each must tolerate a transient build break from the other.

## 12. Risks and containment

| Risk | Why | Containment |
|---|---|---|
| Missed implicit-drop sites | upstream cancels tasks, unsubscribes and releases entities by scope exit; Go does nothing (5.2) | strict leak mode on in this repo's tests; porting checklist greps `Subscription`, `Task<`, `impl Drop` per file (13 files have Drop impls in gpui); tests assert release timing as upstream's do |
| Wayland surface for Vulkan | needs libwayland objects (section 1) | S12 starts with a spike: window + wgpu surface + clear color under Hyprland, before any other Linux code |
| wgpu-native v29 vs upstream wgpu v30 (RESEARCH) | WGSL or feature differences | shaders stay verbatim; S13 compiles all four WGSL variants in a test first; `render.Renderer` keeps the back end replaceable |
| Root package size and serial lane | ~70% of the port is one Go package | contracts stage S3; leaf subsystems moved out; lane A never waits on B or C |
| Deterministic scheduler on coroutines | task bodies run on other goroutines; a body that blocks outside gpui primitives stalls the poller | scheduler tests ported first (S2); `Block` has upstream's parking and timeout-tick checks (`scheduler/src/test_scheduler.rs:208-222`); a debug watchdog names the task that did not park |
| Per-frame garbage | upstream uses a bump arena for elements and `SmallVec` everywhere | pooled slices in `Frame`; element state typed, not boxed (6.5); allocation benchmarks from `gpui_platform/benches` ported in S19 and tracked |
| GPU struct layout drift | primitives are raw bytes to shaders (6.8) | size and offset test in S3, before any renderer exists |
| 3099 generated style methods | binary size, godoc size, name clashes | one generic method set, not one per element; golden list test; clash check already run (6.5) |
| Mixin shadowing | an element method with the same name as a mixin method silently wins | paritycheck lint: no element type may declare a method named in `styled-methods.tsv` or the interactive sets |
| macOS, Windows, iOS, web cannot be run locally | dev box is Linux | PR opened at M0; each of those platforms lands behind CI build + headless tests; visual checks are listed for the owner, not claimed |
| Text shaping differences change wrap points | different shaper than cosmic-text | wrap and truncation logic is in the ported front end and tested with the test text system; back-end differences are confined to advances |
| purego/objc and hand COM ABI mistakes | no compiler checks struct layout or selectors | each binding file has a size/selector smoke test that runs in CI on the real OS |
| accesskit schema version | RESEARCH: accesskit-c is 0.23, upstream 0.24 | core uses its own schema package; version skew is confined to adapters |
| Scanner blind spots | macro-generated items other than style methods, items inside `macro_rules!` bodies | paritycheck re-runs the scan; owning stages add rows found by reading the file |

## 13. Known parity limits and deviations

Limits (behaviour differs and stays different):

| Id | Limit | Source |
|---|---|---|
| L-GLYPH | Linux glyph pixels differ from swash's scaler | RESEARCH-deps.md; `gpui_wgpu/src/cosmic_text_system.rs:18`, `:52` |
| L-JPEG | JPEG decode may differ by small pixel values | RESEARCH-deps.md; `gpui/src/platform.rs:3462-3486` |
| L-SVGTEXT | text and embedded images inside SVG | RESEARCH-deps.md; `gpui/src/svg_renderer.rs:369-517` |

Deviations (same behaviour, different form) are D-NAME, D-RAII, D-OPT, D-VOID, D-EMIT, D-COND, D-INTO, D-GEOM, D-STRING, D-ASYNC, D-RNG, D-ARENA, D-TOKIO, D-HOTRELOAD, D-PROPTEST, D-BENCH, D-SCHEMA, D-FEATURE, D-HTTP, defined in sections 5.1-5.7 and registered in PARITY.md section 5.

## 14. Open items that need the owner

1. Wayland binding over libwayland-client through purego instead of a pure socket client (section 1, 8.1). Without it there is no Vulkan surface on Wayland.
2. D-Bus library for the Linux platform. `github.com/godbus/dbus/v5` is the usual pure-Go choice; it is not in RESEARCH-deps.md and was not evaluated here.
3. Three differences are implied by RESEARCH-deps.md but are not in its "known parity limits" list, so the definition of done does not yet allow them: no WebGL fallback on web (`shaders_webgl.wgsl` would be unused), wgpu-native one major behind upstream, and no multi-threaded web dispatcher. Each needs to be accepted as a limit or scheduled.
4. JSON schema for actions (D-SCHEMA): an in-repo reflector is assumed; the alternative is a dependency.
5. `screen-capture` (scap, `gpui/src/platform/scap_screen_capture.rs`) and `gpui/src/platform/visual_test.rs` (macOS-only) are mapped but no Go capture library was researched; the interfaces port, the Linux/Windows implementation is unplanned.
