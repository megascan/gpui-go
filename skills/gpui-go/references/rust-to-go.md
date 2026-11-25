# Porting Rust GPUI to gpui-go

## Contents
- Porting procedure
- Worked example
- Common API mapping
- Core deviations (D-*)
- Element and platform deviations you will hit

## Porting procedure

1. Translate the types first. Each `struct` stays a struct. `impl Render for X` becomes `func (x *X) Render(window *gpui.Window, cx *gpui.Context[X]) gpui.IntoElement`. Each `#[derive(IntoElement)]` + `RenderOnce` becomes a value type with `Render(*gpui.Window, *gpui.App)` and `IntoAnyElement() { return gpui.NewComponent(x) }`.
2. Rename builders: `div()` becomes `gpui.NewDiv()`, `snake_case` methods become `CamelCase` (`gap_2` is `Gap2`, `w_1_2` is `W1_2`, `text_color` is `TextColor`). Literals need typed units: `px(10.)` becomes `gpui.Px(10)` and `rems(1.)` becomes `gpui.Rems(1)`.
3. Every place Rust drops something becomes an explicit call. A dropped `Entity` or `FocusHandle` needs `Release()`. A dropped `Subscription` needs `Unsubscribe()` (and `_subscriptions: Vec<Subscription>` becomes a field released in `Drop`). A dropped `Task` needs `Cancel()`. Each `.detach()` stays `.Detach()`.
4. `Option`/`Result`: `foo()?` becomes `v, err := foo(); if err != nil { return ..., err }`. An `Option` return becomes `(T, bool)`. An `Option` field becomes `gpui.Option[T]`.
5. Closures generic over a return type split in two: the void form (`Update`) and the `R` form (`UpdateR`).
6. `async move |this, cx| { ... .await ... }` becomes `cx.Spawn(func(this gpui.WeakEntity[T], cx *gpui.AsyncApp) (R, error) {...})`. Each `.await` becomes `.Await(cx)`. `this.update(cx, ...)?` becomes `if err := this.Update(cx, ...); err != nil { return ..., err }`.
7. Actions: `actions!(ns, [A, B])` or `#[derive(Action)]` becomes `//gpui:action namespace=ns` + actiongen, or a hand-written `Name`/`BoxedClone`/`PartialEq` + `RegisterAction` in `init`.
8. Run `go vet` and then `go test`. The strict leak check catches whatever step 3 missed.

## Worked example

Rust (upstream style):

```rust
struct Clock {
    ticks: usize,
    _task: Task<()>,
}

impl Clock {
    fn new(cx: &mut Context<Self>) -> Self {
        let task = cx.spawn(async move |this, cx| loop {
            cx.background_executor().timer(Duration::from_secs(1)).await;
            if this.update(cx, |this, cx| { this.ticks += 1; cx.notify(); }).is_err() {
                break;
            }
        });
        Self { ticks: 0, _task: task }
    }
}

impl Render for Clock {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        div()
            .flex()
            .gap_2()
            .child(format!("{}", self.ticks))
            .child(
                div()
                    .id("reset")
                    .on_click(cx.listener(|this, _, _, cx| {
                        this.ticks = 0;
                        cx.notify();
                    }))
                    .child("Reset"),
            )
    }
}
```

Go:

```go
type Clock struct {
	ticks int
	task  *gpui.Task[struct{}]
}

func NewClock(cx *gpui.Context[Clock]) *Clock {
	c := &Clock{}
	c.task = cx.Spawn(func(this gpui.WeakEntity[Clock], cx *gpui.AsyncApp) (struct{}, error) {
		for {
			if _, err := cx.BackgroundExecutor().Timer(time.Second).Await(cx); err != nil {
				return struct{}{}, err
			}
			if err := this.Update(cx, func(c *Clock, cx *gpui.Context[Clock]) {
				c.ticks++
				cx.Notify()
			}); err != nil {
				return struct{}{}, nil
			}
		}
	})
	return c
}

func (c *Clock) Drop(_ *gpui.App) {
	c.task.Cancel()
}

func (c *Clock) Render(_ *gpui.Window, cx *gpui.Context[Clock]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		Gap2().
		ChildText(fmt.Sprintf("%d", c.ticks)).
		Child(
			gpui.NewDiv().
				ID("reset").
				OnClick(cx.Listener(func(this *Clock, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Clock]) {
					this.ticks = 0
					cx.Notify()
				})).
				ChildText("Reset"),
		)
}
```

What changed: `_task` (dropped by Rust when the view goes away) became a field cancelled in `Drop`. `.child(String)` became `ChildText`. `on_click` needs `.ID(...)` first, as in Rust, but Go reports a missing one as a compile error. `cx.listener` became `cx.Listener` with an explicit event type.

## Common API mapping

| Rust | Go |
|---|---|
| `Application::new().run(\|cx: &mut App\| ...)` | `platform.Application().Run(func(cx *gpui.App) {...})` |
| `cx.open_window(WindowOptions { .., ..Default::default() }, \|w, cx\| cx.new(...))` | `o := gpui.DefaultWindowOptions(); o.X = ...; cx.OpenWindow(o, func(w *gpui.Window, cx *gpui.App) gpui.Entity[V] {...})` |
| `cx.new(\|cx\| V::new(cx))` | `cx.New(NewV)` with `func NewV(cx *gpui.Context[V]) *V` |
| `entity.update(cx, \|this, cx\| ...)` / returning a value | `entity.Update(cx, f)` / `entity.UpdateR(cx, f)` |
| `entity.read(cx)` | `entity.Read(cx)` (needs `*App`) / `entity.ReadWith(cx, f)` |
| `entity.downgrade()`, `weak.upgrade()` | `Downgrade()`, `Upgrade() (Entity[T], bool)` (release the result) |
| `weak.update(cx, ..)?` | `if err := weak.Update(cx, ...); err != nil {...}` |
| `cx.notify()`, `cx.emit(E)` | `cx.Notify()`, `cx.Emit(E{})` |
| `impl EventEmitter<E> for T {}` | nothing (unchecked, D-EMIT) |
| `cx.subscribe(&e, \|this, e, ev, cx\| ..)` | `cx.Subscribe(e, func(this *T, e gpui.Entity[U], ev *E, cx *gpui.Context[T]) {...})` |
| `cx.observe(&e, ..)` | `cx.Observe(e, func(this *T, e gpui.Entity[U], cx *gpui.Context[T]) {...})` |
| `cx.listener(\|this, ev: &ClickEvent, window, cx\| ..)` | `cx.Listener(func(this *T, ev *gpui.ClickEvent, window *gpui.Window, cx *gpui.Context[T]) {...})` |
| `cx.processor(..)` | `cx.Processor(...)` holds a strong handle. Prefer snapshot + `WeakEntity` |
| `.on_action(cx.listener(Self::f))` | `.OnAction(actionListener(cx.Listener((*T).f)))` with the 4-line adapter `func actionListener[A any](l func(*A, *gpui.Window, *gpui.App)) func(A, *gpui.Window, *gpui.App)`, which passes `&action` on |
| `.when(c, \|d\| ..)`, `.when_some(o, \|d, v\| ..)` | `.When(c, func(d *gpui.Div) *gpui.Div {...})`, `.WhenSome(o, func(d *gpui.Div, v V) *gpui.Div {...})` |
| `.hover(\|s\| s.bg(..))` | `.Hover(func(s *gpui.StyleRefinement) *gpui.StyleRefinement { ... BindStyled ... })` |
| `.child("text")` / `.child(shared_string)` | `.ChildText("text")` / `.Child(sharedString)` |
| `.children(iter)` | `.Children(slice...)` with a `[]gpui.IntoElement` |
| `uniform_list("id", n, cx.processor(\|this, range, w, cx\| ..))` | `gpui.NewUniformList("id", n, func(r gpui.Range, w *gpui.Window, cx *gpui.App) []R {...})` |
| `.track_scroll(&handle)` on uniform_list | `.TrackUniformListScroll(handle)` |
| `list(state, \|ix, w, cx\| ..)` | `gpui.NewList(state, func(ix int, w *gpui.Window, cx *gpui.App) gpui.AnyElement {...})` |
| `el.with_animation(id, anim, \|el, d\| ..)` | `gpui.WithAnimation(el, id, anim, func(el E, d float32) E {...})` |
| `canvas(prepaint, paint)` | `gpui.NewCanvas(prepaint, paint)` |
| `deferred(anchored().child(..))` | `gpui.NewDeferred(gpui.NewAnchored().Child(...))` |
| `cx.focus_handle()`, `window.focus(&h)` | `cx.FocusHandle()` (release it), `window.Focus(h, cx)` / `h.Focus(window, cx)` |
| `impl Focusable for T` | `func (t *T) FocusHandle(cx *gpui.App) gpui.FocusHandle { return t.fh.Clone() }` |
| `KeyBinding::new("cmd-s", Save, Some("Editor"))` | `gpui.NewKeyBinding("cmd-s", Save{}, gpui.Some("Editor"))` |
| `cx.spawn(async move \|cx\| ..)` on App | `cx.Spawn(func(cx *gpui.AsyncApp) (R, error) {...})` |
| `cx.spawn_in(window, async move \|this, cx\| ..)` | `cx.SpawnIn[R](window, func(this gpui.WeakEntity[T], cx *gpui.AsyncWindowContext) (R, error) {...})` |
| `cx.background_spawn(async move { .. })` | `cx.BackgroundSpawn(func(s *scheduler.Scope) (R, error) {...})` |
| `timer(d).await` | `cx.BackgroundExecutor().Timer(d).Await(cx)` |
| `task.detach_and_log_err(cx)` | `gpui.TaskDetachAndLogErr(task, cx)` |
| `cx.global::<G>()`, `cx.set_global(g)`, `cx.update_global(\|g, cx\| ..)` | `cx.Global[G]()`, `cx.SetGlobal(g)`, `gpui.UpdateGlobal(cx, func(g *G, cx *gpui.App) {...})` |
| `px(1.)`, `rems(1.)`, `relative(0.5)`, `auto()` | `gpui.Px(1)`, `gpui.Rems(1)`, `gpui.Relative(0.5)`, `gpui.Auto()` |
| `rgb(0xffffff)`, `hsla(..)`, `point(x, y)`, `size(w, h)`, `Bounds::centered(..)` | `gpui.Rgb(0xffffff)`, `gpui.NewHsla(..)`, `gpui.NewPoint(x, y)`, `gpui.NewSize(w, h)`, `gpui.BoundsCentered(..)` |
| `Range<usize>` | `gpui.Range{Start, End int}` |
| `#[gpui::test] fn t(cx: &mut TestAppContext)` | `func TestX(t *testing.T) { gpui.RunTest(t, func(tc *gpui.TestRun) { cx := tc.Cx() ... }) }` |
| `cx.add_window_view(\|w, cx\| ..)` | `view, vcx := tc.Cx().AddWindowView(func(w *gpui.Window, cx *gpui.Context[V]) *V {...})` |
| `cx.executor().advance_clock(d)` | `cx.Executor().AdvanceClock(d)` |
| `cx.run_until_parked()` | `cx.RunUntilParked()` |

## Core deviations (D-*)

| Id | Rust | Go |
|---|---|---|
| D-NAME | `div()`, `img()`, `point()` | `NewDiv()`, `NewImg`, `NewPoint`. Digit underscores kept (`W1_2`) |
| D-RAII | `Drop` on Entity, FocusHandle, Subscription, Task | Explicit `Release`, `Unsubscribe`/`Detach`, `Cancel`/`Detach`. GC cleanup is only a safety net for entity and focus handles |
| D-OPT | `Option<T>`, `Result<T>` | `(T, bool)`, nil-able, `Option[T]` fields, `(T, error)`. Panics keep upstream messages |
| D-VOID | closure generic over `R` | void form + `R` suffix (`Update`/`UpdateR`) |
| D-EMIT | `EventEmitter<E>` bound | not checked at compile time |
| D-COND | conditional impls (`impl<V: Render> IntoElement for Entity<V>`) | method always present, runtime panic naming the type |
| D-INTO | `impl Into<X>` params | sealed `IntoLength`, `IntoHsla`, `IntoFill`, or a union constraint (`ElementIDSource`) |
| D-GEOM | `Point<T>` for any T | numeric `Point[T Number]` + `PointOf[T any]` |
| D-STRING | `SharedString`, `ArcCow` | `type SharedString string`. Plain strings via `ChildText` |
| D-ASYNC | futures, `.await`, `Task<T>`/`Task<Result<T>>` | iter.Pull coroutines, `task.Await(cx)`, `*Task[T]` returning `(T, error)` |
| D-CTX | generic methods on `AppContext`/`VisualContext` | free funcs over a sealed interface (`AppContextNew`, `VisualContextFocus`, `SetGlobal`) + per-context methods |
| D-IDMIXIN | `InteractiveElement::id` | separate `InteractiveID[S]` mixin. `Stateful` has no `ID` |
| D-INTOELEMENT | `into_element` + associated `Element` | only `IntoAnyElement`. `AnyElement` is not an `Element` |
| D-VIEWADAPT | `into_any_element` on Entity | moves the handle |
| D-ACTIONDATA | `Action` trait fns + `inventory` | `ActionData` + `RegisterAction` in `init` |
| D-GLOBAL | `Global` marker trait | any type is a global. `Global[G]`, `UpdateGlobal`, `SetGlobal` |
| D-ANIMEXT | `AnimationExt` methods | free `WithAnimation`, `WithAnimations`, `WithSpring` |
| D-UNIFORMTRACK | `UniformList::track_scroll` | `TrackUniformListScroll` |
| D-LISTCELL | `ListState(Rc<RefCell>)` | value whose copies share state |
| D-STATEFULIMG | `Stateful<Img>` derefs to `Img` | image style methods only before `ID` |
| D-TASKEXT | `TaskExt::detach_and_log_err` | `TaskDetachAndLogErr(task, cx)` |
| D-FUTUREEXT | `with_timeout` | `FutureWithTimeout(future, timeout, executor)` |
| D-RANGE | `Range<usize>` | `Range{Start, End int}` |
| D-WITH | builder named like a field | `With` prefix (`Menu.WithItems`) |
| D-GETTER | getter equal to a public field | read the field |
| D-FIELDMETHOD | field and method share a name | method gets a suffix (`FocusHandle.WithTabIndex`) |
| D-RNG | `StdRng` | `math/rand/v2` PCG: deterministic per seed, different interleavings than Rust |
| D-PROPTEST | proptest | `gpui.PropertyTest`, no shrinking |
| D-BENCH | criterion | `testing.B` via `gpui.RunBench` |
| D-SCHEMA | schemars | public `jsonschema` package reflector |
| D-FEATURE | cargo features | always compiled, or tag `gpui_profiler` |
| D-MACRO | proc macros | generators (`actiongen`), embedding, reflection, helpers |
| D-LOG | `log` crate, `debug_assertions` | `log/slog`, `gpuiutil.DebugAssertions` (off with `-tags gpui_release`) |
| D-TOKIO | `gpui_tokio` | not ported. Use `BackgroundExecutor.Spawn` |
| D-HOTRELOAD | subsecond hot reload | not ported |
| D-HTTP | `http_client` crate | `gpui.HTTPClient` over `net/http` |

## Element and platform deviations you will hit

- `MouseButton::Navigate(dir)` becomes `gpui.MouseButtonNavigateBack`/`gpui.MouseButtonNavigateForward` (D-FLATENUM).
- Tuples become named structs or multiple results (D-TUPLE).
- `UniformListScrollHandle.0` becomes the field `State`. `Default` becomes `gpui.NewUniformListScrollHandle()` (D-UNIFORMHANDLE).
- `impl UniformListDecoration for Entity<T>` becomes `gpui.UniformListDecorationFromEntity[T, *T](entity)` (D-UNIFORMDECOR).
- Explicit type arguments are sometimes required where Rust infers them: `cx.SpawnIn[struct{}](window, ...)`, `cx.SubscribeIn[Emitter, Event](...)`.
- An entity's `Release` inside `Drop(cx *gpui.App)` is the Go form of Rust field drop order. Release children before the handles they depend on.
