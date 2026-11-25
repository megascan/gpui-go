# gpui-go API cheatsheet

## Contents
- Packages and entry points
- Application and App
- Windows
- Entities and contexts
- Who releases what
- Observe, subscribe, emit
- Globals
- Generic code over contexts
- Leak detection

Package `gpui` = `github.com/megascan/gpui-go`. Also used: `github.com/megascan/gpui-go/platform`, `github.com/megascan/gpui-go/scheduler`, `github.com/megascan/gpui-go/jsonschema`.

## Packages and entry points

| Call | Notes |
|---|---|
| `platform.Application() *gpui.Application` | Desktop app for the current OS. On desktop, `Run` blocks |
| `platform.Headless() *gpui.Application` | No windows |
| `platform.Linux(allowed gpui.WindowingModes)` | Linux only. Restricts backends, e.g. `gpui.WindowingModesX11 \| gpui.WindowingModesHeadless` |
| `platform.CurrentPlatform(headless bool)`, `platform.CurrentHeadlessRenderer()` | For headless rendering and screenshots |
| `(*Application).Run(func(cx *gpui.App))` | Call from `main`. Importing `platform` pins main to the OS thread |
| Builders on `*Application` | `WithAssets(AssetSource)`, `WithQuitMode`, `WithHTTPClient`, `OnOpenURLs`, `OnReopen` |

## Application and App

`*gpui.App` methods you'll use most:
- `New[T](func(cx *gpui.Context[T]) *T) gpui.Entity[T]`: create an entity. The caller releases it.
- `OpenWindow[V](options gpui.WindowOptions, build func(*gpui.Window, *gpui.App) gpui.Entity[V]) (gpui.WindowHandle[V], error)`
- `Activate(bool)`, `Quit()`, `Windows()`, `ActiveWindow()`, `Defer(func(*gpui.App))`, `RefreshWindows()`
- `BindKeys([]gpui.KeyBinding)`, `OnAction[A](func(A, *gpui.App)) *gpui.App`, `DispatchAction(gpui.Action)`
- `Spawn[R](func(*gpui.AsyncApp) (R, error)) *gpui.Task[R]`, `BackgroundSpawn[R](func(*scheduler.Scope) (R, error))`, `BackgroundExecutor()`, `ToAsync()`
- `FocusHandle() gpui.FocusHandle`. The caller releases it.
- `SetMenus([]gpui.Menu)`, `OpenURL(string)`, `WriteToClipboard`/`ReadFromClipboard`, `TextSystem()`, `OnWindowClosed(...) gpui.Subscription`

`*gpui.Context[T]` embeds `*gpui.App`. Pass `cx.App` where an `*App` is wanted. Context-only methods:
- `Notify()`, `Emit[E](event E)`, `Entity()` (the caller releases it), `WeakEntity()`, `EntityID()`
- `Listener[E](func(this *T, e *E, w *gpui.Window, cx *gpui.Context[T])) func(*E, *gpui.Window, *gpui.App)`: weak, safe to hand to elements
- `Processor[E, R](...)`: holds a **strong** handle (see pitfalls). Prefer snapshots or `WeakEntity`.
- `Spawn[R](func(this gpui.WeakEntity[T], cx *gpui.AsyncApp) (R, error))`, `SpawnIn[R](window, func(gpui.WeakEntity[T], *gpui.AsyncWindowContext) (R, error))`
- `Observe`, `Subscribe`, `SubscribeSelf`, `ObserveSelf`, `ObserveRelease`, `ObserveGlobal`, `SubscribeIn`, `ObserveIn`
- `OnFocus`, `OnBlur`, `OnFocusIn`, `OnFocusOut`, `OnFocusLost`, `FocusSelf(window)`, `FocusView(view, window)`
- `OnNextFrame(window, f)`, `DeferIn(window, f)`

## Windows

- `gpui.DefaultWindowOptions()` sets `Show`, `Focus`, `IsMovable`, `IsResizable` and `IsMinimizable` to true, plus an empty titlebar, `Kind: gpui.WindowKindNormal{}` and an opaque background. Always start from it.
- Fields: `WindowBounds gpui.Option[gpui.WindowBounds]`, `Titlebar gpui.Option[gpui.TitlebarOptions]` (`TitlebarOptions{Title: gpui.Some[gpui.SharedString]("x")}`), `WindowMinSize`, `WindowDecorations`, `AppID`, `WindowBackground`, `DisplayID`, `Kind`.
- Bounds: `gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(w), gpui.Px(h)), cx)` wrapped in `gpui.WindowBoundsWindowed(b)`. Also `gpui.WindowBoundsMaximized`, `gpui.WindowBoundsFullscreen`, `gpui.WindowBoundsCentered`.
- `gpui.WindowHandle[V]`: `Root(cx) (gpui.Entity[V], error)` returns a new count that the caller releases. `Update(cx, func(*V, *gpui.Window, *gpui.Context[V])) error`, `Read(cx *gpui.App) (*V, error)`, `ReadWith(cx, func(*V, *gpui.App) R) (R, error)`, `Entity(cx) (gpui.Entity[V], error)` (the caller releases it), `AnyWindowHandle`.
- `*gpui.Window` is not a context. It is passed next to `cx`. Its methods include `Focus(h, cx)`, `Blur(cx)`, `FocusNext(cx)`, `FocusPrev(cx)`, `DispatchAction(a, cx)`, `SetWindowTitle(s)`, `Bounds()`, `ViewportSize()`, `ScaleFactor()`, `MousePosition()`, `Modifiers()`, `Refresh()`, `RemoveWindow()`, `ActivateWindow()`, `RequestAnimationFrame()`, `Prompt(...)`, `Focused(cx) (gpui.FocusHandle, bool)` (the caller releases the result) and `UseState`/`UseKeyedState` (the caller releases the result).

## Entities and contexts

```go
func EntityLifecycle(cx *gpui.App) {
	counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{} })
	defer counter.Release()

	counter.Update(cx, func(c *Counter, cx *gpui.Context[Counter]) {
		c.count++
		cx.Notify()
		cx.Emit(Changed{Delta: 1})
	})
	n := counter.ReadWith(cx, func(c *Counter, _ *gpui.App) int { return c.count })
	doubled := counter.UpdateR(cx, func(c *Counter, _ *gpui.Context[Counter]) int { return c.count * 2 })

	weak := counter.Downgrade()
	if err := weak.Update(cx, func(c *Counter, _ *gpui.Context[Counter]) { c.count = 0 }); err != nil {
		fmt.Println("entity released:", err)
	}
	_, _ = n, doubled
}
```

| `Entity[T]` | `WeakEntity[T]` |
|---|---|
| `Update(cx, f)`, `UpdateR[R](cx, f) R`, `Read(cx *App) *T`, `ReadWith[R](cx, f) R` | `Update(cx, f) error`, `UpdateR[R](cx, f) (R, error)`, `ReadWith[R](cx, f) (R, error)` |
| `UpdateIn(cx VisualContext, f) error` (with window) | `Upgrade() (Entity[T], bool)` (the caller releases the result) |
| `Clone()`, `Release()` (idempotent), `Downgrade()`, `IntoAny()`, `EntityID()`, `FocusHandle(cx)` (needs `Focusable`) | Errors once the entity is released ("entity released") |

`R`-suffixed methods (`UpdateR`, `UpdateInR`) are the value-returning forms. The plain forms return nothing, or only `error`. Entities have no `ReadR`; `ReadR` exists only on the test and bench contexts (`TestAppContext`, `VisualTestAppContext`, `TestApp`, `TestAppWindow`, `BenchAppContext`).

Async contexts differ:
- `(*AsyncApp).Update(func(*App))` returns nothing and panics if the app is gone. `UpdateR[R]` returns `R`.
- `(*AsyncWindowContext).Update(func(*Window, *App)) error`, `UpdateR[R] (R, error)`.
- `WeakEntity.Update(...) error`. In a loop, stop when it returns an error.

## Who releases what

| You get a handle from | Who releases it |
|---|---|
| `cx.New`, `Clone()`, `cx.Entity()`, `WeakEntity.Upgrade()`, `WindowHandle.Root`, `Window.UseState`/`UseKeyedState` | You |
| `cx.FocusHandle()`, `Focusable.FocusHandle(cx)` (implementations return `h.Clone()`) | You |
| The return value of `OpenWindow`'s builder | The window (moved) |
| `Child(entity)`, `AnyViewFromEntity(entity)`, a `Tooltip` builder's returned view | The element or view (moved). Pass `Clone()` if you keep yours |
| `TrackFocus(h)` | Clones internally. Your `h` is still yours |
| Handles passed into `Observe`/`Subscribe` callbacks | Borrowed for the call. `Clone()` to keep one |
| `Subscription` from `Observe`/`Subscribe`/`OnFocus`/`OnBlur`/... | You: `Unsubscribe()` it in `Drop`, or `Detach()` |
| `*Task[T]` from `Spawn`/`BackgroundSpawn` | You: `Cancel()` it in `Drop`, `Detach()`, or `Await` |

An entity whose state owns handles implements `Drop(cx *gpui.App)`, which the app calls on release:

```go
func (l *Log) Drop(_ *gpui.App) {
	for _, s := range l.subs {
		s.Unsubscribe()
	}
}
```

## Observe, subscribe, emit

```go
func NewLog(counter gpui.Entity[Counter]) func(cx *gpui.Context[Log]) *Log {
	return func(cx *gpui.Context[Log]) *Log {
		return &Log{subs: []gpui.Subscription{
			cx.Subscribe(counter, func(this *Log, _ gpui.Entity[Counter], e *Changed, cx *gpui.Context[Log]) {
				this.lines = append(this.lines, fmt.Sprintf("changed by %d", e.Delta))
				cx.Notify()
			}),
			cx.Observe(counter, func(this *Log, c gpui.Entity[Counter], cx *gpui.Context[Log]) {
				this.lines = append(this.lines, fmt.Sprintf("now %d", c.Read(cx.App).count))
			}),
		}}
	}
}
```

- `Observe` fires on `cx.Notify()`. `Subscribe[T2, E]` fires on `cx.Emit(E{})`, and the handler receives `*E`.
- Event types are not checked at compile time (D-EMIT). If the subscriber's `E` doesn't match the emitted type, it simply never fires.
- On `*App`, `Observe` and `Subscribe` exist without the `this` parameter.

## Globals

```go
func Globals(cx *gpui.App) {
	cx.SetGlobal(Settings{Theme: "dark"})
	theme := cx.Global[Settings]().Theme
	if s, ok := cx.TryGlobal[Settings](); ok {
		theme = s.Theme
	}
	gpui.UpdateGlobal(cx, func(s *Settings, _ *gpui.App) { s.Theme = "light" })
	_ = theme
}
```

Any type can be a global. `Global[G]` panics if it is unset, so use `TryGlobal` or `HasGlobal`. Use `cx.ObserveGlobal[G](...)` to react to changes.

## Generic code over contexts

`gpui.AppContext` is a sealed interface, because Go interfaces can't have generic methods. Generic operations are free functions:
- `gpui.AppContextNew[T](cx, build) gpui.Entity[T]` (no error)
- `gpui.AppContextUpdateEntity`, `gpui.AppContextReadEntity`
- `gpui.AppContextUpdateWindow[R](cx, window, f) (R, error)`
- `gpui.AppContextWithWindow[R](cx, entityID, f) (R, bool)`
- `gpui.VisualContextNewWindowEntity`, `gpui.VisualContextFocus`
- `gpui.SetGlobal`, `gpui.UpdateGlobal`, `gpui.UpdateGlobalR`, `gpui.Global`

```go
func Generic(cx gpui.AppContext) gpui.Entity[Clock] {
	return gpui.AppContextNew(cx, NewClock)
}
```

## Leak detection

| `GPUI_LEAK_DETECTION` | In `go test` | Outside tests |
|---|---|---|
| unset | strict (panics with "Exited with leaked handles") | off |
| `strict` | strict | strict |
| any other non-empty value, including `0` | on: tracks live handles (for `AssertReleased` and snapshots), but doesn't report collected ones | on |

- Strict mode reports entity handles, focus handles and subscriptions that were garbage-collected without `Release`/`Unsubscribe`/`Detach`. It does **not** track tasks, so a forgotten task is silent and you have to review tasks by hand.
- `LEAK_BACKTRACE=1` adds allocation stacks.
- Reports are process-global, so keep leak-checked tests serial.
- `(*App).LeakDetectorSnapshot()` + `AssertNoNewLeaks(&snapshot)` assert locally.
