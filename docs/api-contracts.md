# Root API contracts

Binding for every stage that writes or calls package `gpui` (STAGES.md decision 1). Created by M0-contracts; later root stages update the sections they own. DESIGN.md sections 5 and 6 give the reasons; this file records the shapes as built.

## 1. Stub protocol

- `stubs.go` holds forward declarations with their final names and signatures. A stub body is `panic("gpui: unimplemented <symbol>")`; unexported fields of stub structs are placeholders and may be replaced freely.
- A stage that ports an upstream file deletes that file's symbols from `stubs.go` and writes them in the mapped Go file (PARITY F1). It may add stubs for symbols a later stage owns. A signature change to a stubbed exported symbol is a contract change: update this file in the same change and record it in `docs/port/handoff/<stage>.md`.
- Other packages compile against stubs and never call them, in tests or in `init`. `RegisterAction` is a stub until M0-input-types, so a package that registers actions from `init` must not be linked into a test binary before then.
- `stubs.go` is deleted by M3-root-capture. internal/taffy and the platform packages use the same protocol with a package-local `stubs.go`.
- The only real bodies in `stubs.go` are `AnyElement.IntoAnyElement` and the mixin bind helpers (section 3); the stage that takes over the type keeps them unchanged.

Stub owners (the stage that replaces the symbol):

| Symbols | Stage |
|---|---|
| `DisplayID`, `PlatformDisplay` | M0-platform-types |
| `WindowOptions` | M0-platform-data |
| `ClickEvent`, `PlatformInput`, `Modifiers`, `Keystroke`, `KeystrokeParse`, `Action`, `ActionData`, `RegisterAction`, `Fluent`, `BindFluent`, `(Fluent).When`, `(Fluent).Map` | M0-input-types |
| `Scene`, `PrimitiveBatch`, `(Scene).Batches` | M0-scene |
| `TextSystem`, `WindowTextSystem`, `Font`, `FontRun`, `TextRun`, `FontMetrics`, `LineLayout`, `ShapedRun`, `ShapedGlyph`, `WrappedLineLayout`, `LineLayoutCache` | M0-text-system |
| `LineWrapper`, `Boundary` | M0-line-wrapper |
| `EntityID`, `AnyEntity`, `AnyWeakEntity`, `Entity`, `WeakEntity`, `slot`, `entityFromAny`, `Subscription` | M0-entities |
| `BackgroundExecutor`, `ForegroundExecutor` | M0-executors |
| `App` (type), `AppCell`, `AppRef`, `AppRefMut`, `(App).FindDisplay`, `PrimaryDisplay`, `Spawn`, `OpenWindow`, `WithWindow` | M0-app-core |
| `GpuiBorrow`, `newGpuiBorrow`, `globalLease`, `(App).SetGlobal`, `DefaultGlobal`, `leaseGlobal`, `endGlobalLease`, `Observe`, `Defer`, the unexported `App` primitives | M0-app-effects |
| `Context`, `newContext`, `AsyncApp`, `(App).New`, `ReserveEntity`, `InsertEntity`, `AsMut`, `ReadWindow`, `BackgroundSpawn`, `ReadGlobal`, `TestOption`, `Seed`, `Seeds`, `Iterations`, `Retries`, `OnFailure`, `RunTest`, `TestRun` | M0-app-contexts |
| `TestAppContext` | M0-test-contexts |
| `Styler`, `Styled`, `BindStyled`, `(Styled).Flex`, `SizeFull`, `Style`, `StyleRefinement` | M0-style |
| `Element`, `ElementA11y`, `ElementA11ySyntheticChildren`, `IntoElement`, `AnyElement`, `NewAnyElement`, `Render`, `RenderOnce`, `View`, `AnyView`, `anyViewDowncast`, `ParentElement`, `Parent`, `BindParent`, `GlobalElementID`, `InspectorElementID`, `SourceLocation`, `LayoutID`, `AvailableSpace`, `(Entity).IntoAnyElement` | M0-element |
| `Window` (type), `WindowID`, `AnyWindowHandle`, `WindowHandle`, `ArenaClearNeeded`, `DispatchPhase`, `FocusHandle`, `Focusable`, `ContentMask`, `ElementID`, `ElementIDSource`, `Hitbox`, `HitboxBehavior`, `PaintQuad` | M0-window-core |
| `(Window).RemSize`, `Draw`, `WithElementState`, `PaintQuad`, `Refresh`, `OnNextFrame`, `RequestAnimationFrame`, `ReplaceRoot` | M0-window-paint |
| `(Window).RequestLayout`, `InsertHitbox`, `DispatchKeystroke`, `DispatchEvent`, `DispatchEventResult` | M0-window-dispatch |
| `Interactivity`, `InteractiveElement`, `Interactive`, `BindInteractive`, `InteractiveID`, `BindInteractiveID`, `StatefulInteractiveElement`, `StatefulInteractive`, `BindStatefulInteractive`, `Div`, `NewDiv` and its methods except `IntoAnyElement` | M0-div-api |
| `(Div).IntoAnyElement`, `Stateful` and its methods | M0-div-paint |
| `(SharedString).IntoAnyElement` | M0-text-elements |
| `A11ySubtreeBuilder` | M0-window-a11y |

## 2. Contexts (gpui.go)

`AppContext` is sealed: its primitives are unexported, so only gpui's contexts and types that embed one (`*App`, which `Context[T]` embeds; a type in another package that embeds `*App` replaces `#[derive(AppContext)]`) implement it. The primitives, in upstream's trait order:

| Primitive | Upstream | Contract |
|---|---|---|
| `newEntity(t, build func(slot AnyEntity, cx *App) any) AnyEntity` | `new` | reserve a slot, call `build` with it, push `EntityCreated`, insert (app.rs:3484). `insertEntity` does not push the effect, so `New` cannot be composed from reserve + insert |
| `reserveEntity(t) AnyEntity` | `reserve_entity` | the slot's handle |
| `insertEntity(slot, build func(cx *App) any) AnyEntity` | `insert_entity` | |
| `updateEntity(h, f func(state any, cx *App))` | `update_entity` | lease the entity for `f`; `state` is the `*T` |
| `asMut(h) *App` | `as_mut` | the App to lease from; async contexts panic with upstream's message |
| `readEntity(h, f func(state any, cx *App))` | `read_entity` | |
| `updateWindow(h AnyWindowHandle, f func(AnyView, *Window, *App)) error` | `update_window` | |
| `withWindow(id EntityID, f func(*Window, *App)) bool` | `with_window` | false where upstream returns `None` |
| `readWindow(h AnyWindowHandle, f func(root AnyView, cx *App)) error` | `read_window` | the free function downcasts the root and returns "root view's type has changed" |
| `backgroundExecutor() BackgroundExecutor` | `background_spawn` | |
| `readGlobal(t, f func(global any, cx *App))` | `read_global` | `global` is the `*G`; a missing global panics as `App::global` does |

Each `AppContext*` free function wraps one primitive with types; return types follow upstream's trait at zed@6501df1: plain values, except `AppContextUpdateWindow` / `AppContextReadWindow` `(R, error)` and `AppContextWithWindow` `(R, bool)`. Implementations panic where upstream panics (`AsyncApp` after the app is released: "app was released before async operation completed").

`VisualContext` adds `WindowHandle()` and the primitive `updateWindowContext(f func(*Window, *App)) error`, which runs `f` inside the context's window (`self.app.update_window(self.window, ...)` upstream). `VisualContextNewWindowEntity`, `VisualContextUpdateWindowEntity`, `VisualContextReplaceRootView`, `VisualContextFocus` are built on it and return `error` / `(R, error)` for upstream's `Self::Result` (gpui.rs:265). `Entity[T].UpdateIn` / `UpdateInR` call `VisualContextUpdateWindowEntity`.

Per context type, the upstream-named generic methods (`New`, `ReserveEntity`, `InsertEntity`, `ReadGlobal`, `BackgroundSpawn`, `ReadWindow`, `WithWindow`, `AsMut`, `Update` / `UpdateR`) are one-line delegates to the free functions, so ported call sites read `cx.New(...)` in any context. `App` declares them; `Context[T]` gets them by embedding `*App` (generic methods are promoted) and declares only the ones upstream gives it different signatures (`Observe`, `Spawn`, ...). `AsyncApp`, `AsyncWindowContext`, the test contexts and the bench contexts declare their own. Return types follow each upstream method.

`BorrowAppContext` (unexported `borrowApp() *App`, implemented by `*App` in gpui.go and promoted to `Context[T]`) carries `SetGlobal[G](cx, g)`, `UpdateGlobal` / `UpdateGlobalR[G](cx, func(*G, C) R)` and `UpdateDefaultGlobal` / `UpdateDefaultGlobalR`; `C` is the caller's own context type, upstream's `&mut Self`. Call as `gpui.UpdateGlobal(cx, func(s *Settings, cx *gpui.App) {...})`. `SetGlobal`, `UpdateGlobal`, `UpdateGlobalR`, `UpdateDefaultGlobal` and `UpdateDefaultGlobalR` are taken at package level, so global.rs's `UpdateGlobal` trait cannot become a Go type of that name (`ReadGlobal` exists only as methods and `AppContextReadGlobal`, so that name is free).

Globals and events are keyed by `reflect.Type` of `G` / `E`; the stored value of a global is `*G`.

Background work: `BackgroundSpawn` / `BackgroundExecutor.Spawn` take `func(s *scheduler.Scope) (R, error)`, the scheduler's task body, so the body can await (`t.Await(s)`); DESIGN 6.2's `func() R` cannot. Foreground task bodies get `*AsyncApp`, which embeds `*scheduler.Scope` and is therefore an `Awaiter`: `v, err := task.Await(acx)`.

## 3. Mixins and elements

The five fluent traits are self-typed mixins. Each stores `self S` and returns it, so chains keep the concrete type. An element embeds the mixins and binds each one in its constructor:

```go
type Div struct {
	gpui.Styled[*Div]; gpui.Interactive[*Div]; gpui.InteractiveID[*Div]; gpui.Parent[*Div]; gpui.Fluent[*Div]
	// ...
}

func NewDiv() *Div {
	d := &Div{}
	gpui.BindStyled(&d.Styled, d)
	gpui.BindInteractive(&d.Interactive, d)
	gpui.BindInteractiveID(&d.InteractiveID, d)
	gpui.BindParent(&d.Parent, d)
	gpui.BindFluent(&d.Fluent, d)
	return d
}
```

| Mixin | Constraint the element implements | Upstream |
|---|---|---|
| `Styled[S Styler]` | `Style() *StyleRefinement` | `Styled::style` |
| `Interactive[S InteractiveElement]` | `Interactivity() *Interactivity` | `InteractiveElement::interactivity` |
| `InteractiveID[S InteractiveElement]` | same | `InteractiveElement::id` (D-IDMIXIN) |
| `StatefulInteractive[S StatefulInteractiveElement]` | same | `StatefulInteractiveElement` |
| `Parent[S ParentElement]` | `Extend(elements []AnyElement)` | `ParentElement::extend` |
| `Fluent[S any]` | none | `FluentBuilder` |

- An element never declares a method that a mixin provides (paritycheck `-lint`, styled-methods.tsv plus the mixin method sets); the required methods above are declared by the element, never by the mixin.
- `InteractiveID[S].ID[I ElementIDSource](id I) *Stateful[S]` is separate from `Interactive` because `Stateful[E]` embeds `Interactive[*Stateful[E]]`: an `ID` on `Interactive` would instantiate `Stateful[*Stateful[E]]`, an instantiation cycle Go rejects. `Stateful[E]` embeds every mixin except `InteractiveID`. Elements that upstream gives `id` (every InteractiveElement) embed and bind `InteractiveID`.
- `Stateful[E]` implements `IntoElement` only (D-COND, DESIGN 6.5): `Style`, `Interactivity`, `Extend` type-assert the inner element and panic naming the type when it lacks the method; `IntoAnyElement` returns the inner element's `AnyElement`.
- `Element[RL, PP]` methods: `ElementID`, `SourceLocation`, `RequestLayout`, `Prepaint`, `Paint`. Optional hooks are separate interfaces the drawable type-asserts: `ElementA11y`, `ElementA11ySyntheticChildren[PP]` (asserted with the drawable's own `PP`). `NewAnyElement(e)` infers `RL` and `PP` from `e`'s methods.
- `IntoElement` is `IntoAnyElement() AnyElement`. `AnyElement`, `SharedString`, `*Div`, `*Stateful[E]` and `Entity[T]` (panics unless `*T` implements `Render[T]`, D-COND) implement it; plain strings go through `Parent.ChildText`.
- `Render[T]` is implemented by `*T` (`Render(w *Window, cx *Context[T]) IntoElement`); `RenderOnce` by component values.

## 4. Handles and Drop (D-RAII)

- A handle value aliases one counted handle: copying an `Entity[T]`, `AnyEntity`, `FocusHandle` is a Rust move or borrow. `Clone()` makes a new counted handle; `Release()` is the explicit drop, idempotent per handle; use after release panics. `Entity[T]` embeds `AnyEntity` (upstream's `Deref`); `IntoAny()` returns the same counted handle, so release one of the two, not both. The free functions pass `e.IntoAny()` to primitives as a borrow and never release it.
- A handle gpui passes into a callback (the root in `read(root, cx)`, the entity in `Observe` / `Subscribe` callbacks) is borrowed for the call: the callback calls `Clone()` to keep it, and whoever created a count releases it. A handle a callee returns (`Clone()`, `Focusable.FocusHandle`) is a new count the caller releases. `anyViewDowncast` returns an alias of the root the primitive passes, with no new count. `newGpuiBorrow(handle, app)` borrows `handle` for the lease; upstream clones it into `GpuiBorrow::new` and drops the clone when `new` returns (app.rs:3779), which nets to no count change.
- Call the drop method exactly where upstream drops the value (end of scope: usually `defer`), moves it into `detach()`, or overwrites it. `runtime.AddCleanup` is only the safety net for entity and focus handles and the leak reporter; subscriptions and tasks never stop because of garbage collection.
- Drop table (DESIGN 5.2): `AnyEntity` / `Entity` / `FocusHandle` `Release`; `Subscription` `Unsubscribe` / `Detach`; `Task` `Cancel` / `Detach`; `AppRef` / `AppRefMut` `Release` (ends the borrow group); `GpuiBorrow` `Release` (notify, end the lease, finish the update); `LeaseInner` checked by `EndLease`; executor `Scope` `Close`; queue senders and receivers `Close`; `MissingGlyphReceiver` `Close`, `LineWrapperHandle` `Release`; profiler guards `Release` / `End` / `Close`; journal guards `End`; `HeadlessAppContext` `Close`; `LeakDetector` `Report`; `ScapStream` `Stop`; `Deferred` `Run` / `Abort`; platform owners `Close` / `Destroy`; scope guards `End` / `Close`; bench helpers `Close` / `End`.
- `AppCell.Borrow` / `BorrowMut` / `TryBorrowMut` return guards; callers `defer g.Release()`. A conflicting borrow panics with upstream's text; `TryBorrowMut` returns the error instead.
- `AppRefMut` embeds `*App` (upstream's `DerefMut`), so it is an `AppContext` and `BorrowAppContext`. `AppRef` (upstream's `Deref` to `&App`) holds the App in an unexported field and exposes only `Get() *App` and `Release()`: it implements neither interface, so a shared borrow cannot be passed to `SetGlobal` / `UpdateGlobal` / `AppContextNew` or call mutating methods by promotion. Code behind `Get()` only reads (fields, `&self` methods); anything that mutates takes `BorrowMut`. `TestContracts_SealedInterfaces` pins this.

## 5. Naming

- D-VOID: a closure API generic over its return type has a void form and an `R`-suffixed form: `Update` / `UpdateR`, `UpdateIn` / `UpdateInR`, `UpdateGlobal` / `UpdateGlobalR`. The `R` suffix is the only spelling.
- Generic trait methods of a sealed context trait are free functions named `<Trait><Method>` (D-CTX): `AppContextNew`, `VisualContextFocus`.
- `anyhow::Result<T>` is `(T, error)`, `Result<()>` is `error`, returned `Option<T>` is `(T, bool)` or a nil-able type, `Option<T>` fields are `Option[T]`.
- Re-exports: `Task`, `FallibleTask`, `Priority`, `DedicatedExecutor`, `SchedulerLocalExecutor` alias package scheduler; `AccessibleAction`, `Orientation`, `Role`, `Toggled` alias package a11y.
