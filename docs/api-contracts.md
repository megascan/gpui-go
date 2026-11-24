# Root API contracts

Binding for every stage that writes or calls package `gpui` (STAGES.md decision 1). Created by M0-contracts; later root stages update the sections they own. DESIGN.md sections 5 and 6 give the reasons; this file records the shapes as built.

## 1. Stub protocol

- `stubs.go` holds forward declarations with their final names and signatures. A stub body is `panic("gpui: unimplemented <symbol>")`; unexported fields of stub structs are placeholders and may be replaced freely.
- A stage that ports an upstream file deletes that file's symbols from `stubs.go` and writes them in the mapped Go file (PARITY F1). It may add stubs for symbols a later stage owns. A signature change to a stubbed exported symbol is a contract change: update this file in the same change and record it in `docs/port/handoff/<stage>.md`.
- Other packages compile against stubs and never call them, in tests or in `init`. `RegisterAction` is a stub until M0-input-types, so a package that registers actions from `init` must not be linked into a test binary before then.
- `stubs.go` is deleted by M3-root-capture. internal/taffy and the platform packages use the same protocol with a package-local `stubs.go`.
- The only real bodies in `stubs.go` are `AnyElement.IntoAnyElement` and the mixin bind helpers (section 3), plus the final values of `DefaultWindowSize` and `profilerFeatureEnabled`; the stage that takes over the symbol keeps them unchanged.

Stub owners (the stage that replaces the symbol):

| Symbols | Stage |
|---|---|
| `WindowOptions`, `WindowParams`, `WindowBounds`, `WindowAppearance`, `WindowBackgroundAppearance`, `TextRenderingMode`, `PathPromptOptions`, `PromptLevel`, `PromptButton`, `CursorStyle`, `ClipboardItem`, `PlatformInputHandler`, `TextInputConfiguration`, `Menu`, `MenuItem`, `OwnedMenu` | M0-platform-data |
| `Capslock`, `KeybindingKeystroke`, `KeybindingKeystrokeFromKeystroke`, `KeyDownEvent`, `ExternalDragPayload`, `RenderImageParams`, `ImageID` | M0-input-types |
| `profilerFeatureEnabled` (const, false until the `gpui_profiler` tag files exist) | M0-profiler-core |
| `TestDispatcher`, `ThreadedDispatcher` | M0-executors |
| `TestWindow` | M0-test-platform |
| `Keymap` | M0-keymap |
| `PlatformGestures`, `GestureTuning`, `GestureKinds` | M0-gestures-spring |
| `SystemWindowTab` | M0-app-core |
| `RenderSvgParams`, `TimedTransformation`, `Transformation` (and their unexported `minFrameInterval`, `at`, `alphaFactor`, `intoMatrix`) | M0-elements-media |
| `NativeViewID` | M0-window-a11y |
| `ClickEvent`, `PlatformInput`, `Modifiers`, `Keystroke`, `KeystrokeParse`, `Action`, `ActionData`, `RegisterAction`, `Fluent`, `BindFluent`, `(Fluent).When`, `(Fluent).Map` | M0-input-types |
| `TextSystem`, `WindowTextSystem`, `Font`, `FontRun`, `TextRun`, `FontMetrics`, `LineLayout`, `ShapedRun`, `ShapedGlyph`, `WrappedLineLayout`, `LineLayoutCache`, `FontID`, `GlyphID`, `RenderGlyphParams`, `MissingGlyphSink`, `MissingGlyph` | M0-text-system |
| `LineWrapper`, `Boundary`, `charWidthCache`, `defaultCharWidthCache`, `newLineWrapper`, `intoCharWidths`, `lineWrapperBreakIndices` (real bodies, see section 8) | M0-line-wrapper |
| `ShapedLine`, `WrappedLine`, `DecorationRun` | M0-text-elements |
| `BackgroundExecutor`, `ForegroundExecutor` | M0-executors |
| `App` (type; the stub carries the final field `entities *entityMap`, which `Entity.Read` uses), `AppCell`, `AppRef`, `AppRefMut`, `(App).FindDisplay`, `PrimaryDisplay`, `Spawn`, `OpenWindow`, `WithWindow` | M0-app-core |
| `GpuiBorrow`, `newGpuiBorrow`, `globalLease`, `(App).SetGlobal`, `DefaultGlobal`, `leaseGlobal`, `endGlobalLease`, `Observe`, `Defer`, the unexported `App` primitives | M0-app-effects |
| `TestAppContext`, `TestAppContextBuild`, `(TestAppContext).Update`, `RunUntilParked`, unexported `drop` | M0-test-contexts |
| `FocusOutEvent`, `WeakFocusHandle`, `newWeakFocusHandle`, `windowFocusEvent` (`isFocusIn`, `isFocusOut`), `anyObserver`, `anyWindowFocusListener`, `FocusHandle.id`, the `Window` fields `displayObservers`, `focusListeners`, `focusLostListeners`, `boundsObservers`, `appearanceObservers`, `buttonLayoutObservers`, `visibilityObservers`, `activationObservers`, `pendingInputObservers`, `(Window).newFocusListener`, `Focus`, `Spawn`, `SpawnWithPriority`, `Prompt` | M0-window-core |
| `(Window).OnAction` | M0-window-dispatch |
| `Element`, `ElementA11y`, `ElementA11ySyntheticChildren`, `IntoElement`, `AnyElement`, `NewAnyElement`, `Render`, `RenderOnce`, `View`, `AnyView`, `anyViewDowncast`, `ParentElement`, `Parent`, `BindParent`, `GlobalElementID`, `InspectorElementID`, `SourceLocation`, `LayoutID`, `AvailableSpace`, `(Entity).IntoAnyElement` | M0-element |
| `HitboxID` (final type `uint64`, so tests build ids directly), `Window` (type), `WindowID`, `AnyWindowHandle`, `WindowHandle`, `ArenaClearNeeded`, `DispatchPhase`, `FocusHandle`, `Focusable`, `ContentMask`, `ElementID`, `ElementIDSource`, `Hitbox`, `HitboxBehavior`, `PaintQuad`, `NewQuad`, `Outline`, `DefaultWindowSize`, `WindowControlArea` | M0-window-core |
| `(Window).RemSize`, `PaintDropShadows`, `PaintInsetShadows`, `Draw`, `WithElementState`, `PaintQuad`, `Refresh`, `OnNextFrame`, `RequestAnimationFrame`, `ReplaceRoot` | M0-window-paint |
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

### 2.1 Context, AsyncApp, AsyncWindowContext (app_context.go, app_async_context.go; M0-app-contexts)

- `Context[T]` is `struct{ *App; entityState WeakEntity[T] }`, built by `newContext(app, weak)`. Its own methods keep upstream's names and argument order; window-taking methods take `window *Window` where upstream takes `&mut Window`. `Emit[E](event E)` pushes `effectEmit{event: &event}`; subscribers receive `*E`. `Entity()` returns a strong handle the caller releases; every temporary strong handle inside these methods is released where Rust drops it. `Processor` keeps a strong handle in the returned closure, as upstream. `OnDrop` / `AsyncAppOnDrop` return `gpuiutil.Deferred`; run it where upstream drops the value. `OnAction(actionType reflect.Type, window, func(this *T, action any, phase, window, cx))`.
- The window-taking `Context` methods (`ObserveIn`, `SubscribeIn`, `OnFocus*`, `OnBlur`, `ObserveWindow*`, `ObservePendingInput`, `SpawnIn*`, `OnAction`, `FocusSelf`, `FocusView`) go through the `Window` stubs listed in section 1; M0-window-core / M0-window-dispatch keep those names and signatures.
- `AsyncApp` is `struct{ *scheduler.Scope; app weak.Pointer[AppCell]; bgExecutor; foregroundExecutor }`. Every method reaches the App through the AppCell guard with upstream's borrow kind (`readEntity`, `readWindow`: shared; the rest mutable; `updateWindow` / `withWindow` use `tryBorrowMut` and return the borrow error / false). A foreground task body is an iter.Pull coroutine that the main thread resumes synchronously (runtime/coro.go hands the locked OS thread to the coroutine), so calling into the App from the body runs on the main thread while the poller is suspended: that is how `Update` "parks and runs on the poller". It also works outside a task (`ToAsync()` in tests). Errors: "app was released", "app is quitting"; panics: "app was released before async operation completed", "Cannot as_mut with an async context. Try calling update() first".
- `AsyncApp.Spawn` / `AsyncWindowContext.Spawn` copy the context and set the copy's `Scope` to the new task's; a context value never changes its Scope.
- `AsyncWindowContext` is `struct{ *AsyncApp; window AnyWindowHandle }` and implements `VisualContext` (`updateWindowContext` = `update_window(self.window)`). Upstream's `update_window_entity` uses `with_window(view)`; the shared `VisualContextUpdateWindowEntity` uses the context window instead. Methods that return `Result` upstream return `error` / `(R, error)` with `R`-suffixed forms (`UpdateR`, `UpdateRootR`, `UpdateGlobalR`). `Prompt(level, message, detail Option[string], answers []PromptButton) *scheduler.OneshotReceiver[int]` returns a closed receiver when the window is gone.

### 2.2 Assets (asset_cache.go)

`Asset[S, O]` is implemented by a marker type: `Load(source S, cx *App) func(s *scheduler.Scope) O`, the body run on the background executor. `cx.FetchAsset[A](source) (O, bool)`, `cx.RemoveAsset[A](source)`, `cx.HasAsset[A](source)` infer `S` and `O` from `A`. `RemoveAsset` cancels a pending load. `AssetLogger[A, S, R]` wraps an asset whose output is `scheduler.Result[R]`. `Hash[T](data)` is rustc-hash 2.1's FxHasher (`FxHasher`); types with a custom Rust `Hash` implement `Hashable`.

### 2.3 Test runtime (testing.go)

`RunTest(t, fn, Seed/Seeds/Iterations/Retries/OnFailure)` runs `fn(*TestRun)` on the test goroutine per seed from `CalculateSeeds` with a fresh `TestDispatcher`. `tc.Cx()` / `tc.NewContext()` call `TestAppContextBuild(dispatcher.Clone(), Some(t.Name()))`; after `fn` each context is run until parked, quit (`ForbidParking`, `Quit`), run until parked and `drop()`ped (M0-test-contexts: GC until the live-handle count stops falling, then the leak report), then `DrainTasks` and `EndTest`. Retries apply to panics only: a `t.Fatal` / `t.Error` failure cannot be undone; `OnFailure` and the failing-seed report run for both. `tc.BlockTest(body)` runs an async test body. `PropertyTest(t, func(tc, rng), Cases(n))` draws each case's scheduler seed with `SeedStrategy` from a generator seeded with `PropertyTestRNGSeed()` (no shrinking, D-PROPTEST). `Observe(entity, cx) *Observation[struct{}]` (`Next(awaiter)`, `Close()`).

### 2.4 Test contexts (app_test_context.go, app_test_app.go, app_headless_app_context.go, app_visual_test_context.go; M0-test-contexts)

`TestAppContext`, `HeadlessAppContext`, `VisualTestAppContext` implement `AppContext`; `VisualTestContext` (embeds `*TestAppContext`) implements `VisualContext`. Each has the same-named generic methods (`New`, `ReserveEntity`, `InsertEntity`, `ReadWindow`, `BackgroundSpawn`, `ReadGlobal`) plus `Update(func(*App))` / `UpdateR[R]`; entity access is `entity.Update(cx, ...)` / `entity.ReadWith(cx, ...)`. `TestAppContext` adds the platform simulators (`SimulatePathPromptResponse`, `SimulateSystemNotificationResponse`, `ExpectRestart`, `SimulateWindowResize`, ...), `Notifications`, `Events[T, E]`, `Condition(awaiter, entity, pred)`, `Spawn`, `ToAsync`, and `entity.NextEvent[E](cx)`, `NextNotification`, `Condition[Evt]` futures. Every context ends with the leak report: RunTest's teardown calls `drop()`; standalone contexts (`TestAppContextSingle()`, `NewTestApp()`, `NewHeadlessAppContext(ts)`, `NewVisualTestAppContext(p)`) must be ended with `defer cx.Close()`. Tests release every entity handle and unsubscribe or detach every subscription they create; strict leak mode fails the test otherwise.

Writing a headless test from another package (e.g. a text system):

```go
func TestFoo(t *testing.T) {
	cx := gpui.NewHeadlessAppContext(mytext.NewTextSystem())
	defer cx.Close()
	cx.Update(func(app *gpui.App) { /* app.TextSystem()... */ })
	cx.RunUntilParked()
}
```

or, for a seeded run with upstream's #[gpui::test] semantics, `gpui.RunTest(t, func(tc *gpui.TestRun) { cx := gpui.TestAppContextBuildWithTextSystem(tc.Dispatcher().Clone(), gpui.Some(t.Name()), ts); defer cx.Close(); ... })` (plain `tc.Cx()` uses the NoopTextSystem). Async bodies run inside `tc.BlockTest(func(s *scheduler.Scope) { ... })` and await with `task.Await(s)` / `rx.Recv(s)`.

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

### 3.1 Style (style.go, styled.go; M0-style)

- `Style` fields are upstream's, in upstream's order; `StyleRefinement`, `TextStyleRefinement`, `GridTemplateRefinement`, `UnderlineStyleRefinement`, `StrikethroughStyleRefinement` are generated into `style_gen.go` by refinegen. Non-numeric geometry fields use the D-GEOM `Of` family (`PointOf[Overflow]`, `EdgesOf[Length]`, `SizeOf[Length]`, `CornersOf[AbsoluteLength]`).
- Defaults: `DefaultStyle()`, `DefaultTextStyle()`, `DefaultFill()`. A zero `Style` is not upstream's default (`StyleFromRefinement` fills unset fields with Go zero values, so an unset `FontWeight` is 0, not 400); callers that need upstream's `Default` refine `DefaultStyle()`.
- Field-less enums are `uint8` with `<Type><Variant>` constants; `Display`'s zero value is `DisplayFlex` (upstream's `#[default]`), `DisplayBlock` is 1. JSON uses the variant names. `JustifyItems`, `AlignSelf`, `JustifySelf` alias `AlignItems`; `JustifyContent` aliases `AlignContent`.
- `TextOverflow` is a comparable tagged struct: `TextOverflowTruncate(s)`, `TextOverflowTruncateStart(s)`, `TextOverflowTruncateMiddle(s)`, read with `AsTruncate`, `AsTruncateStart`, `AsTruncateMiddle`.
- `Fill` is a sealed interface whose only variant is `FillColor` (`type FillColor Background`); `(Fill).Color() (Background, bool)`. `IntoFill{ToFill() Fill}` is implemented by `Hsla`, `Rgba`, `Background` and `Fill`.
- `IntoHighlightStyle{ToHighlightStyle() HighlightStyle}` is implemented by `HighlightStyle`, `TextStyle`, `Hsla`, `Rgba`, `FontWeight`, `FontStyle`. `HighlightStyleColor(c)` is upstream's `HighlightStyle::color`.
- `CombineHighlights(a, b iter.Seq2[Range, HighlightStyle]) iter.Seq2[Range, HighlightStyle]`; active highlights fold in ascending insertion order.
- `BoxShadow{Color, Offset, Blur, Spread, IsInset}`: the fields are renamed because the builder methods `BlurRadius`, `SpreadRadius`, `Inset` keep upstream's names; JSON keys stay `blur_radius`, `spread_radius`, `inset`.
- `(*Style).Paint(bounds, window, cx, continuation func(*Window, *App))`, `(*Style).OverflowMask(bounds, remSize) (ContentMask[Pixels], bool)`, `(*Style).TextStyle() *TextStyleRefinement` (nil when unset).
- `Styled[S]` carries the 115 hand-written methods of styled.rs and the 3099 generated ones (`styled_gen.go`, stylegen); `Styled.TextStyle()` returns `*TextStyleRefinement`. `Style()` is never a mixin method: the element declares it. `*StyleRefinement` implements `Styler` (upstream's `impl Styled for StyleRefinement`), so `Styled[*StyleRefinement]` styles a bare refinement.
- `(*StyleRefinement).GridLocationMut() *GridLocation` inserts the default location first.

## 4. Handles and Drop (D-RAII)

- A handle value aliases one counted handle: copying an `Entity[T]`, `AnyEntity`, `FocusHandle` is a Rust move or borrow. `Clone()` makes a new counted handle; `Release()` is the explicit drop, idempotent per handle; use after release panics. `Entity[T]` embeds `AnyEntity` (upstream's `Deref`); `IntoAny()` returns the same counted handle, so release one of the two, not both. The free functions pass `e.IntoAny()` to primitives as a borrow and never release it.
- A handle gpui passes into a callback (the root in `read(root, cx)`, the entity in `Observe` / `Subscribe` callbacks) is borrowed for the call: the callback calls `Clone()` to keep it, and whoever created a count releases it. A handle a callee returns (`Clone()`, `Focusable.FocusHandle`) is a new count the caller releases. `anyViewDowncast` returns an alias of the root the primitive passes, with no new count. `newGpuiBorrow(handle, app)` borrows `handle` for the lease; upstream clones it into `GpuiBorrow::new` and drops the clone when `new` returns (app.rs:3779), which nets to no count change.
- Call the drop method exactly where upstream drops the value (end of scope: usually `defer`), moves it into `detach()`, or overwrites it. `runtime.AddCleanup` is only the safety net for entity and focus handles and the leak reporter; subscriptions and tasks never stop because of garbage collection.
- Drop table (DESIGN 5.2): `AnyEntity` / `Entity` / `FocusHandle` `Release`; `Subscription` `Unsubscribe` / `Detach`; `Task` `Cancel` / `Detach`; `AppRef` / `AppRefMut` `Release` (ends the borrow group); `GpuiBorrow` `Release` (notify, end the lease, finish the update); `LeaseInner` checked by `EndLease`; executor `Scope` `Close`; queue senders and receivers `Close`; `MissingGlyphReceiver` `Close`, `LineWrapperHandle` `Release`; profiler guards `Release` / `End` / `Close`; journal guards `End`; `HeadlessAppContext` `Close`; `LeakDetector` `Report`; `ScapStream` `Stop`; `Deferred` `Run` / `Abort`; platform owners `Close` / `Destroy`; scope guards `End` / `Close`; bench helpers `Close` / `End`.
- `AppCell.Borrow` / `BorrowMut` / `TryBorrowMut` return guards; callers `defer g.Release()`. A conflicting borrow panics with upstream's text; `TryBorrowMut` returns the error instead.
- `AppRefMut` embeds `*App` (upstream's `DerefMut`), so it is an `AppContext` and `BorrowAppContext`. `AppRef` (upstream's `Deref` to `&App`) holds the App in an unexported field and exposes only `Get() *App` and `Release()`: it implements neither interface, so a shared borrow cannot be passed to `SetGlobal` / `UpdateGlobal` / `AppContextNew` or call mutating methods by promotion. Code behind `Get()` only reads (fields, `&self` methods); anything that mutates takes `BorrowMut`. `TestContracts_SealedInterfaces` pins this.

### 4.1 Entity map and subscriber sets (app_entity_map.go, subscription.go)

- `App.entities *entityMap` (from `newEntityMap()`). Primitives for the App stages: `reserveErased(t) AnyEntity`, `insertErased(slot, state any) AnyEntity` (state is the `*T`), `leaseErased(h, typeName) *leaseInner` / `endLeaseErased(id, lease)`, `readErased(h) any`; typed forms `reserve[T]`, `insert[T]`, `lease[T]` / `endLease[T]` (`lease[T].get() *T`), `read[T]`. Access tracking: `pushAccessScope`, `popAccessScope`, `takeAccessed`, `restoreAccessed`, `extendAccessed([]EntityID)`, `clearAccessed`. `takeDropped() []droppedEntity{entityID, entity}` drains the dropped-ids queue in release order; the App runs release observers and `Drop(cx)` on the returned states.
- A nested update or read panics with upstream's "cannot {update|read} {type} while it is already being updated". `endLeaseErased` on an ended lease panics.
- Handles: zero `Entity` / `AnyEntity` values are allowed (`var e Entity[T]`); `Release` on them is a no-op, every other method panics. Equality of entities is `a.EntityID() == b.EntityID()`; `==` on strong handles is handle identity. `AnyWeakEntity` / `WeakEntity` are plain values. `WeakEntity.Update` / `UpdateR` / `UpdateIn` / `UpdateInR` / `ReadWith` upgrade, run, and release the strong handle; errors "entity released" and "entity has no current window".
- Leak detection: `GPUI_LEAK_DETECTION` unset in a test binary or `strict` = strict; set otherwise (not `0`) or any test binary = on; else off. Evaluated once per process. The harness ends a run with `cx.entities.report()` (upstream `LeakDetector::drop`), after quitting, draining tasks and GC. Strict reports (handles, subscriptions, leases collected without their drop method) are process-global, so tests that check them must not run in parallel with tests that leak.
- `subscriberSet[K comparable, C any]` (`newSubscriberSet`) and `singleSubscriberSet[C]` (`newSingleSubscriberSet`): `insert`, `insertPending` (returns `(Subscription, activate func())`), `remove` (`iter.Seq[C]`, lazily filtered to active subscribers), `retain(f func(*C) bool)`. No lock; main thread only, as upstream's `Rc<RefCell>`.
- `Subscription`: `NewSubscription(func())`, `Unsubscribe` (Drop, idempotent), `Detach`, `SubscriptionJoin(a, b)`. Store subscriptions an entity owns in its state and unsubscribe them in its `Drop(cx *App)`.

## 5. Naming

- D-VOID: a closure API generic over its return type has a void form and an `R`-suffixed form: `Update` / `UpdateR`, `UpdateIn` / `UpdateInR`, `UpdateGlobal` / `UpdateGlobalR`. The `R` suffix is the only spelling.
- Generic trait methods of a sealed context trait are free functions named `<Trait><Method>` (D-CTX): `AppContextNew`, `VisualContextFocus`.
- `anyhow::Result<T>` is `(T, error)`, `Result<()>` is `error`, returned `Option<T>` is `(T, bool)` or a nil-able type, `Option<T>` fields are `Option[T]`.
- Re-exports: `Task`, `FallibleTask`, `Priority`, `DedicatedExecutor`, `SchedulerLocalExecutor` alias package scheduler; `AccessibleAction`, `Orientation`, `Role`, `Toggled` alias package a11y.

## 6. Platform interfaces (platform.go)

Built by M0-platform-types; DESIGN 6.9 gives the reasons.

- `Platform`, `PlatformWindow`, `PlatformDisplay`, `PlatformDispatcher`, `PlatformTextSystem`, `PlatformAtlas`, `AtlasBackend`, `PlatformHeadlessRenderer`, `ScreenCaptureSource`, `ScreenCaptureStream`, `PlatformKeyboardLayout`, `PlatformKeyboardMapper` carry upstream's methods in upstream order. Methods upstream gates by `cfg` (`ReadFromPrimary`, `WriteToPrimary`, `ReadFromFindPasteboard`, `WriteToFindPasteboard`, `WriteCredentialsIfNotExists`, `SetTrafficLightPosition`, `SetNativeTitlebarVisibility`, `GetRawHandle`, `SetExclusiveEdge`, `AsTest`, `AsThreaded`, `RenderToImage`, `Contains`) exist on every GOOS.
- Provided methods (D-DEFAULTS): embed the matching `*Defaults` struct. `PlatformDefaults`, `PlatformWindowDefaults`, `PlatformDisplayDefaults` and `PlatformDispatcherDefaults` must be bound in the constructor, because some defaults call other methods of the implementor (`ReadFromClipboardAsync`, `VisualViewportBounds`, `InnerWindowBounds`, `VisibleBounds`, `DefaultBounds`, `DispatchOnMainThreadWhenIdle`); an unbound default panics. `PlatformTextSystemDefaults` and `PlatformAtlasDefaults` need no binding.

```go
type MacWindow struct {
	gpui.PlatformWindowDefaults
	// ...
}

func newMacWindow() *MacWindow {
	w := &MacWindow{}
	gpui.BindPlatformWindowDefaults(&w.PlatformWindowDefaults, w)
	return w
}
```

- cfg-gated methods with no upstream default get a no-op default: an implementation for the OS that has them (Linux `ReadFromPrimary` / `WriteToPrimary`, macOS find pasteboard, Windows `GetRawHandle`) must override it.
- Drop (D-PLATSEAM): `Platform.Close` and `PlatformWindow.Close` are the drop of the platform and window values (default no-op; owners with an upstream `impl Drop` override them). `ScreenCaptureStream.Stop` stops capture. `AttachNativeView` returns a `NativeViewAttachment` (nil when unsupported) whose `Close` removes the view. `ActivityGuard.Release` and `TimerResolutionGuard.Run` (a `gpuiutil.Deferred`) end their activity.
- Return shapes: `Task<Result<()>>` and `Task<()>` are `*Task[struct{}]`; `Task<Result<Option<T>>>` is `*Task[Option[T]]`. `oneshot::Receiver<Result<T>>` is `*scheduler.OneshotReceiver[scheduler.Result[T]]`: `Recv`'s error is the cancellation, `Result.Err` the platform's error. A returned `Option<Rc<dyn X>>` / `Option<Box<dyn X>>` is a nil-able interface; other returned options are `(T, bool)`. `Option<&[T]>` and `Option<Vec<T>>` parameters are `Option[[]T]` where `Some` of an empty slice means something else than `None` (`SetInputRegion`). `serde_json::Value` is `jsontext.Value` (`"null"` by default). Paths are `string`. `Uuid` is `gpui.UUID`.
- `RawWindowHandle` is a tagged struct: `Kind` selects which fields are set (Wayland display + surface, Xcb connection + window + visual, AppKit view, UIKit view, Win32 HWND + HINSTANCE, Web canvas element id). Native pointers are `uintptr`.
- `AtlasKey` is comparable and built only with `AtlasKeyFromGlyph` / `AtlasKeyFromSvg` / `AtlasKeyFromImage`, so `RenderGlyphParams`, `RenderSvgParams` and `RenderImageParams` must stay comparable. `AtlasTile`, `AtlasTextureID`, `AtlasTextureKind` (`uint32`, D-GPUENUM) and `TileID` are GPU data: 32-byte `AtlasTile` in upstream field order. `AtlasState[B AtlasBackend]` is used with a pointer backend (`NewAtlasState(&myTextures{})`); `HeadlessAtlas`'s zero value is ready to use. Conversions between `TileID` and etagere `AllocID` live in the atlas implementations.
- `PlatformFrameSignal` must be created with `NewPlatformFrameSignal` and not copied; it stores nothing while `profilerFeatureEnabled` is false.
- `GraphicalEnvironment` has one struct per OS family (`platform_cfg_freedesktop.go` for `linux || freebsd`, `platform_cfg_windows.go`, `platform_cfg_other.go` for the rest); code that reads its fields must carry the same build constraint. `GuessCompositor` and `WindowingModes` are Linux and FreeBSD only (`linux || freebsd`); `WindowButtonLayoutLinuxDefault`, `WindowButtonLayoutParse` and the unexported `format` build everywhere (Linux-only upstream). `PlatformScreenCaptureFrame` is `ScapFrame` on Linux and Windows (`platform_cfg_linux.go`, `platform_cfg_windows.go`) and a `uintptr` elsewhere: a CVImageBufferRef on macOS (the macOS back end owns its retain and release), zero on FreeBSD (`platform_cfg_freebsd.go`) and the rest (`platform_cfg_other.go`).
- Screen capture seam: `ScapBackend` (`GetAllTargets`, `BuildCapturer`) and `ScapCapturer` (`StartCapture`, `StopCapture`, `GetNextFrame`, `Target`) mirror the zed-scap calls of `scap_screen_capture.rs`; targets are `*ScapDisplay` / `*ScapWindow`, frames the `ScapFrame*` types. Code that builds a `ScreenCaptureFrame` from a `ScapFrame` needs `//go:build linux || windows`. A back end registers once from `init` with `RegisterScapBackend` (a nil or second registration panics); the glue (M3-root-capture) reads it with the unexported `registeredScapBackend`.
- Stub field contracts set here for later stages: `FontMetrics` (`UnitsPerEm`, `Ascent`, `Descent`, `LineGap`, `UnderlinePosition`, `UnderlineThickness`, `CapHeight`, `XHeight`, `BoundingBox`), `LineLayout`, `ShapedRun`, `ShapedGlyph`, `RenderGlyphParams`, `RenderSvgParams`, `RenderImageParams` and `Capslock.On` carry upstream's public fields; `FontID` is `uint`, `GlyphID` `uint32`, `ImageID` `uint`.

## 7. Scene and renderer (scene.go, path_builder.go, render/render.go)

Built by M0-scene; DESIGN 6.8 and 6.9 give the reasons.

- `Scene`'s zero value is an empty scene. Its primitive slices are exported fields (`Shadows`, `Quads`, `Paths`, `Underlines`, `MonochromeSprites`, `SubpixelSprites`, `PolychromeSprites`, `Surfaces`, `NativeViews`), sorted into drawing order by `Finish`. Renderers upload them with `unsafe.Slice`; `scene_layout_test.go` pins every GPU struct's size and offsets against the WGSL, HLSL and Metal (cbindgen C) layouts.
- `InsertPrimitive(IntoPrimitive)` takes any primitive struct (`Quad`, `Shadow`, `Underline`, `MonochromeSprite`, `SubpixelSprite`, `PolychromeSprite`, `PaintSurface`, `PaintNativeView`, `PaintedPath`) or a `Primitive`. `Primitive` is a sealed interface over the wrappers `PrimitiveQuad{Quad}` and so on (D-PRIMITIVE); `Bounds()` and `ContentMask()` are wrapper methods.
- `Batches() iter.Seq[PrimitiveBatch]` (call `Finish` first). Batches are `PrimitiveBatchShadows{Range}`, `PrimitiveBatchQuads`, `PrimitiveBatchPaths`, `PrimitiveBatchUnderlines`, `PrimitiveBatchSurfaces`, `PrimitiveBatchNativeViews` (each embeds `Range`) and `PrimitiveBatchMonochromeSprites` / `SubpixelSprites` / `PolychromeSprites` with `TextureID` and `Range`; type-switch on them. `Label()` gives upstream's debug label. Paths are uploaded with `PaintedPath.AppendTo(&paints, &curves, &tiles)`; `TileCount` counts instances. `QuadDepthScale` / `QuadDepth` are the quad depth helpers.
- Zero values that are not upstream's defaults: a zero `TransformationMatrix` is not the unit matrix and a zero `SpriteAnimationID` is not `SpriteAnimationIDNone`. Every sprite built by ported code sets `Transformation: TransformationMatrixUnit()` (or the element's matrix) and `Animation: SpriteAnimationIDNone`, as upstream's call sites do; a zero matrix collapses the sprite's bounds and the scene drops it. `BorderStyle` is a `uint32` (D-GPUENUM) with JSON as `"Solid"` / `"Dashed"`.
- `PaintSurface.ImageBuffer` is a `uintptr` CVPixelBufferRef on macOS and iOS, zero elsewhere (D-SURFACEBUF); the Apple back end retains it when painting and releases it when the scene is cleared.
- Paths: `PathBuilderFill()` / `PathBuilderStroke(width)` return `*PathBuilder`; the drawing methods mutate it, `WithStyle` / `DashArray` return it, `Build() (Path, error)`. `PathStyle` is `PathStyleStroke{StrokeOptions}` or `PathStyleFill{FillOptions}`. `Transform`, `FillRule`, `LineCap`, `LineJoin` alias internal/lyon with their variant constants re-declared in gpui. `NewPaintedPath(path, scaleFactor, ContentMask[Pixels], Background)` bins a path for painting (`Window.PaintPath`).
- `render.Renderer` (package `render`, imports only gpui) is the method set platform windows hold: `Draw`, `UpdateDrawableSize`, `UpdateTransparency`, `SetSubpixelLayout`, `SpriteAtlas`, `SupportsDualSourceBlending`, `GPUSpecs`, `DeviceLost`, `Recover(RawWindowHandle) error`, `Destroy`. `Destroy` runs before the window's surface and display connection close.

## 8. Text system (text_system*.go)

Built by M0-text-system.

- `NewTextSystem(PlatformTextSystem) *TextSystem` is shared and safe for concurrent use; `NewWindowTextSystem(*TextSystem) *WindowTextSystem` embeds it (upstream's Deref) and is used from its window's thread only. Unexported crate-level hooks for later root stages: `(*TextSystem).getRasterBounds`, `rasterizeGlyph`, `glyphDilationForColor`, `recommendedRenderingMode`, `takeMissingGlyphReceiver`, `enableMissingGlyphReporting`, `disableMissingGlyphReporting`; `(*WindowTextSystem).layoutIndex`, `reuseLayouts(start, end LineLayoutIndex)` (upstream's `Range<LineLayoutIndex>`), `truncateLayouts`, `finishFrame`.
- `Font` is comparable by value and is a map key. `FontFeatures` and `FontFallbacks` intern their lists by content (`NewFontFeatures`, `FontFallbacksFromFonts`), so equal lists compare equal; `TagValueList` and `FallbackList` return the shared interned slice, which is read-only; the zero `FontFeatures` is `DefaultFontFeatures()`. Zero values are not upstream's defaults: use `DefaultFont()` (".SystemUIFont", weight 400), `NewFont(family)`, `DefaultFontWeight()`, `DefaultTextRun()`.
- `FontFeature{Tag, Value}` is upstream's `(String, u32)`. `FontStyle` and `FallbackFontClass` are `uint8` enums; `FontWeight` is `float32`.
- `FontMetrics` keeps upstream's field names; its pixel methods are `AscentPixels`, `DescentPixels`, ..., `BoundingBoxPixels` (D-FIELDMETHOD). `WrappedLineLayout::wrap_boundaries` is not declared; read the `WrapBoundaries` field (D-GETTER).
- Layouts are shared pointers: `LayoutLine` returns `*LineLayout`, `ShapeText` returns `[]WrappedLine` with `*WrappedLineLayout`. `ShapedLine` embeds `*LineLayout` and `WrappedLine` embeds `*WrappedLineLayout` (upstream's Deref); both carry `Text` and the unexported `decorationRuns []DecorationRun`. Their zero values embed nil layouts and panic on use, unlike upstream's `Default` (M0-text-elements decides whether to add a default constructor).
- Returned options: `(LineLayout).IndexForX`, `FontIDForIndex`, `(WrappedLineLayout).PositionForIndex` return `(T, bool)`; `IndexForPosition` / `ClosestIndexForPosition` return `(int, bool)` where `false` carries upstream's `Err` index. `ShapeText`'s `line_clamp` is `Option[int]`; `force_width` / `wrap_width` are `Option[Pixels]`.
- Offsets are UTF-8 byte offsets throughout (`TextRun.Len`, `FontRun.Len`, `ShapedGlyph.Index`).
- Drop: `MissingGlyphReceiver.Close` closes and drains the channel; `LineWrapperHandle.Release` returns the wrapper's character widths to the pool (idempotent; the embedded `*LineWrapper` is nil afterwards). `MissingGlyphReceiver.Recv(cx scheduler.Awaiter)` returns `scheduler.ErrClosed` when the channel is closed.
- Forward stubs for M0-line-wrapper in stubs.go: `LineWrapper` (`textSystem`, `fontID`, `fontSize`, `charWidths`), `charWidthCache`, `defaultCharWidthCache`, `newLineWrapper`, `intoCharWidths` and a working `lineWrapperBreakIndices(text) iter.Seq[int]` (go-text segmenter, yields 0, each break offset and `len(text)`, as ICU's `segment_str`). For M0-text-elements: `DecorationRun`; `UnderlineStyle`, `StrikethroughStyle` now live in style.go (M0-style).
