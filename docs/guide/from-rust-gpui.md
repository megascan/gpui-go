# Coming from Rust GPUI

gpui-go ports Zed's GPUI at commit `6501df15`. It keeps upstream's names, order of operations and behavior. Where Rust and Go differ, the port uses a fixed set of re-expressions, each with a deviation id (`D-*`). This page covers the ones you meet in everyday code. The full registry is in [PARITY.md section 5](../port/PARITY.md#5-deviations-and-limits).

## Side by side

Upstream's `hello_world` (`_ref/gpui/examples/hello_world.rs`):

```rust
fn run_example() {
    application().run(|cx: &mut App| {
        if !example_support::load_fonts(cx) {
            return;
        }
        let bounds = Bounds::centered(None, size(px(500.), px(500.0)), cx);
        cx.open_window(
            WindowOptions {
                window_bounds: Some(WindowBounds::Windowed(bounds)),
                ..Default::default()
            },
            |_, cx| {
                cx.new(|_| HelloWorld {
                    text: "World".into(),
                })
            },
        )
        .unwrap();
        cx.activate(true);
    });
}
```

The Go port ([examples/hello_world/main.go](../../examples/hello_world/main.go)):

```go
func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(500.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[HelloWorld] {
			return cx.New(func(_ *gpui.Context[HelloWorld]) *HelloWorld {
				return &HelloWorld{text: "World"}
			})
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}
```

## Common API

| Rust | Go |
|---|---|
| `gpui_platform::application().run(\|cx\| ...)` | `platform.Application().Run(func(cx *gpui.App) {...})` |
| `WindowOptions { .., ..Default::default() }` | `o := gpui.DefaultWindowOptions()`, then set fields |
| `cx.new(\|cx\| State { .. })` | `cx.New(func(cx *gpui.Context[State]) *State { return &State{} })` |
| `entity.read(cx)` | `entity.Read(cx)` or `entity.ReadWith(cx, f)` |
| `entity.update(cx, \|this, cx\| ..)` | `entity.Update(cx, f)`; `entity.UpdateR(cx, f)` when it returns a value |
| `weak.update(cx, ..)?` | `if err := weak.Update(cx, f); err != nil {...}` |
| drop of `Entity<T>` | `entity.Release()` |
| `impl Render for T { fn render(&mut self, window, cx: &mut Context<Self>) -> impl IntoElement }` | `func (t *T) Render(window *gpui.Window, cx *gpui.Context[T]) gpui.IntoElement` |
| `#[derive(IntoElement)]` + `impl RenderOnce` | `IntoAnyElement()` returning `gpui.NewComponent(v)` + `Render(window, cx *gpui.App)` |
| `div()`, `img(src)`, `svg()` | `gpui.NewDiv()`, `gpui.NewImg(src)`, `gpui.NewSvg()` |
| `px(10.)`, `rems(1.)`, `relative(0.5)` | `gpui.Px(10)`, `gpui.Rems(1)`, `gpui.Relative(0.5)` |
| `rgb(0x..)`, `hsla(..)` | `gpui.Rgb(0x..)`, `gpui.NewHsla(..)` |
| `point(..)`, `size(..)`, `Bounds::centered(None, s, cx)` | `gpui.NewPoint(..)`, `gpui.NewSize(..)`, `gpui.BoundsCentered(gpui.None[gpui.DisplayID](), s, cx)` |
| `.child(x)`, `.children(iter)` | `.Child(x)`, `.Children(xs...)`; `.ChildText(s)` for a Go string |
| `.when(c, \|this\| ..)`, `.when_some(o, \|this, v\| ..)` | `.When(c, func(this *gpui.Div) *gpui.Div {..})`, `.WhenSome(o, ...)` |
| `.id("x")` | `.ID("x")`, which returns `*gpui.Stateful[...]` |
| `.on_click(cx.listener(\|this, ev, window, cx\| ..))` | `.OnClick(cx.Listener(func(this *T, ev *gpui.ClickEvent, window *gpui.Window, cx *gpui.Context[T]) {..}))` |
| `.on_action(cx.listener(Self::f))` | `.OnAction(actionListener(cx.Listener((*T).f)))`, with a small adapter (see [actions guide](actions-keymaps-focus.md#handle-actions)) |
| `cx.processor(..)` | `cx.Processor(..)`, but prefer a weak version (see [lists guide](lists-and-scrolling.md#reading-view-state-from-the-row-callback)) |
| `.w_1_2()`, `.size_full()` | `.W1_2()`, `.SizeFull()` |
| `cx.notify()`, `cx.emit(e)` | `cx.Notify()`, `cx.Emit(e)` |
| `cx.observe(&e, \|this, e, cx\| ..)` | `cx.Observe(e, func(this *T, e gpui.Entity[W], cx *gpui.Context[T]) {..})` |
| `cx.subscribe(&e, \|this, e, ev: &E, cx\| ..)` | `cx.Subscribe(e, func(this *T, e gpui.Entity[W], ev *E, cx *gpui.Context[T]) {..})` |
| `impl EventEmitter<E> for T {}` | nothing |
| `subscription.detach()`, drop | `subscription.Detach()`, `subscription.Unsubscribe()` |
| `cx.spawn(async move \|this, cx\| ..)` | `cx.Spawn(func(this gpui.WeakEntity[T], cx *gpui.AsyncApp) (R, error) {..})` |
| `cx.spawn_in(window, ..)` | `cx.SpawnIn(window, func(this gpui.WeakEntity[T], cx *gpui.AsyncWindowContext) (R, error) {..})` |
| `cx.background_spawn(fut)` | `cx.BackgroundSpawn(func(s *scheduler.Scope) (R, error) {..})` |
| `executor.timer(d).await` | `cx.BackgroundExecutor().Timer(d).Await(cx)` |
| `task.await` | `task.Await(cx)` returning `(R, error)` |
| `task.detach()`, drop of `Task` | `task.Detach()`, `task.Cancel()` |
| `actions!(menu, [MoveUp])`, `#[derive(Action)]` | `//gpui:action namespace=menu` + `go generate` (actiongen) |
| `KeyBinding::new("up", MoveUp, Some("Menu"))` | `gpui.NewKeyBinding("up", MoveUp{}, gpui.Some("Menu"))` |
| `cx.bind_keys([..])` | `cx.BindKeys([]gpui.KeyBinding{..})` |
| `.key_context("Menu")`, `.track_focus(&h)` | `.KeyContext("Menu")`, `.TrackFocus(h)` |
| `impl Focusable for T` | `func (t *T) FocusHandle(cx *gpui.App) gpui.FocusHandle` |
| `window.focus(&h, cx)` | `window.Focus(h, cx)` |
| `uniform_list(id, n, f)`, `.track_scroll(&h)` | `gpui.NewUniformList(id, n, f)`, `.TrackUniformListScroll(h)` |
| `list(state, f)`, `ListState::new(..)` | `gpui.NewList(state, f)`, `gpui.NewListState(..)` |
| `Range<usize>` | `gpui.Range{Start, End}` |
| `impl Global for G`, `cx.global::<G>()` | nothing; `cx.Global[G]()` |
| `cx.update_global::<G, _>(\|g, cx\| ..)` | `gpui.UpdateGlobal(cx, func(g *G, cx *gpui.App) {..})` |
| `Option<T>` | `gpui.Option[T]` (`gpui.Some`, `gpui.None`) in fields, `(T, bool)` in results |
| `Result<T>` | `(T, error)` |
| `SharedString`, `"x".into()` | `gpui.SharedString` (a Go `string` type), `gpui.SharedString("x")` |
| `#[gpui::test] fn t(cx: &mut TestAppContext)` | `func TestX(t *testing.T) { gpui.RunTest(t, func(tc *gpui.TestRun) { cx := tc.Cx() ... }) }` |
| `cx.run_until_parked()`, `cx.executor().advance_clock(d)` | `cx.RunUntilParked()`, `cx.Executor().AdvanceClock(d)` |
| `cx.add_window_view(..)` | `cx.AddWindowView(..)` |

## Deviations you will notice

| Id | What changes |
|---|---|
| D-NAME | Free constructors that collide with a type become `New<Type>`: `div()` is `NewDiv()`, `point()` is `NewPoint()`. Digit boundaries keep an underscore: `w_1_2` is `W1_2`. |
| D-RAII | No `Drop`. Release handles, cancel tasks and unsubscribe explicitly. Entity state can have a `Drop(cx *gpui.App)` method. The GC releases forgotten entity and focus handles as a safety net only. See [Concepts](concepts.md#release-and-drop). |
| D-CTX | Go interfaces cannot have generic methods, so `AppContext` is a sealed interface. Generic operations are methods on concrete contexts (`cx.New`) or free functions over `gpui.AppContext` (`gpui.AppContextNew(cx, f)`, `gpui.UpdateGlobal(cx, f)`). |
| D-VOID | Closure APIs come in two forms: `Update` (no result) and `UpdateR` (returns a value). Same for `ReadWith`, `UpdateGlobal`/`UpdateGlobalR` and others. |
| D-OPT | `Option` and `Result` become `(T, bool)`, `(T, error)`, nil, or `gpui.Option[T]` fields. Panic messages stay the same. |
| D-INTO | `impl Into<X>` parameters become concrete types or sealed interfaces (`IntoLength`, `IntoHsla`, `IntoFill`). Untyped numbers do not convert: write `gpui.Px(10)`. |
| D-IDMIXIN | `ID` lives on a separate mixin. After `ID` the element is `*gpui.Stateful[E]`, which has `OnClick`, `Active`, `Tooltip` and scrolling but no second `ID`. |
| D-EMIT | `EventEmitter<E>` is not checked. Any type can be emitted and subscribed to. |
| D-COND | Conditional trait impls become methods that always exist and panic at run time when the condition fails, for example `Child(entity)` when `*T` has no `Render`. |
| D-ASYNC | Futures are coroutines on GPUI's scheduler. `.await` is `Await(cx)`. `Task<T>` and `Task<Result<T>>` are both `*gpui.Task[T]` returning `(T, error)`. A foreground task blocks only through GPUI primitives. |
| D-ACTIONDATA | Action metadata (name, builder, JSON schema) lives in a `gpui.ActionData` passed to `gpui.RegisterAction` from `init`, instead of trait functions and `inventory`. |
| D-MACRO | Proc macros become generators: `actiongen` for actions, `stylegen` for the style methods, `refinegen` for refinements. |
| D-SCHEMA | Action JSON schemas come from the in-repo `github.com/megascan/gpui-go/jsonschema` package instead of schemars. |
| D-ANIMEXT | `.with_animation(..)` is the free function `gpui.WithAnimation(element, id, animation, f)`. |
| D-STRING | `SharedString` is `type SharedString string`. `ArcCow` is not ported. |
| D-GLOBAL | Any type can be a global; there is no `Global` marker trait. |
| D-RNG | Test seeds drive a PCG generator. Runs are deterministic per seed, but a given seed does not produce the same interleaving as in Rust. |
| D-PROPTEST | Property tests are seeded and do not shrink. |
| D-FEATURE | Cargo features are always compiled in or switched at run time, except the profiler (`-tags gpui_profiler`). |
| D-TOKIO | `gpui_tokio` is not ported. Use `BackgroundExecutor.Spawn` for blocking work. |
| D-HOTRELOAD | Hot reload is not ported. |

The full list, including platform- and renderer-level deviations that app code does not see, is in [PARITY.md section 5](../port/PARITY.md#5-deviations-and-limits) and the `docs/port/parity/*.md` area files.

## Habits to change

- Write `defer h.Release()` the way you would rely on scope exit in Rust.
- Store every `Task` and `Subscription` you mean to keep, and release them in `Drop`. Dropping the variable does nothing.
- Pass `entity.Clone()` to `Child` when you keep the entity, where Rust would pass `self.child.clone()`.
- Start window options from `gpui.DefaultWindowOptions()`. A struct literal is not `..Default::default()`: it leaves `Show`, `Focus`, `IsResizable` and the title bar off.
- Read [Pitfalls](pitfalls.md) for the rest.
