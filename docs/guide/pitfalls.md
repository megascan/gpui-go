# Pitfalls

Mistakes that compile fine and then misbehave. The first section is specific to the Go port. The second applies to GPUI in general, in Rust or Go.

## Go port

### A `WindowOptions{}` literal hides the window

A struct literal leaves every field at its zero value: `Show`, `Focus`, `IsMovable`, `IsResizable`, `IsMinimizable` are false and there is no title bar. On macOS and Windows a window with `Show` false never appears. Linux ignores `Show`, so the bug can hide until you test elsewhere. (This comes from reading the macOS and Windows backends; they have not been run on real hardware yet.)

Fix: start from the defaults and change fields.

```go
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
```

### Forgetting `Release`

These all hand you a counted handle that you must release: `cx.New`, `entity.Clone()`, `cx.Entity()`, `cx.FocusHandle()`, `focusHandle.Clone()`, `entity.FocusHandle(cx)`, `WindowHandle.Root(cx)`, `weak.Upgrade()`, `window.UseState` and `window.UseKeyedState`.

Fix: `defer h.Release()` for short-lived handles. For handles stored in an entity, release them in the entity's `Drop(cx *gpui.App)` method. Outside tests nothing reports the miss; the GC releases the handle at some later point. In tests, strict leak detection fails the test.

### Giving a handle away by accident

`Child(entity)`, `gpui.AnyViewFromEntity(entity)` and the entity returned from the `OpenWindow` builder take over the handle you pass. If you keep using it, or pass it again on the next `Render`, it has already been released.

Fix: pass a clone when you keep the handle, as in [concepts.go](snippets/concepts/concepts.go):

```go
	return gpui.NewDiv().SizeFull().Child(w.editor.Clone())
```

`TrackFocus(handle)` is the opposite: it clones internally, so your handle is still yours to release.

### Tasks run until you stop them

A `*gpui.Task` that you neither store nor `Detach` keeps running. Leak detection does not track tasks, so nothing reports it.

Fix: store the task in the entity and `Cancel` it in `Drop`, or `Detach` it on purpose. Cancel the old task before you replace a stored one. `Cancel` on a nil `*gpui.Task` panics, so start the field with a finished task:

```go
	return &Report{status: "idle", job: scheduler.ReadyTask(struct{}{})}
```

### Subscriptions keep firing

The `gpui.Subscription` from `Observe`, `Subscribe`, `OnRelease`, `ObserveGlobal` and similar calls stays active until you end it. Dropping the variable does nothing.

Fix: store it and call `Unsubscribe` in `Drop`, or call `Detach()` to keep it for the lifetime of the entities involved. In tests, strict mode reports a subscription the GC collected without either.

### `cx.Processor` keeps the view alive forever

`cx.Processor` captures a strong handle to the entity, and Go has no way to release it when the closure goes away. The view is never freed, and tests fail with a leaked-handle report.

Fix: use a weak processor, shown in [Lists and scrolling](lists-and-scrolling.md#reading-view-state-from-the-row-callback). `cx.Listener` is already weak and safe.

### The element type changes after `ID`

`gpui.NewDiv()` returns `*gpui.Div`. After `.ID(...)` the chain is `*gpui.Stateful[*gpui.Div]`. That has two effects:

- `OnClick`, `Active`, `Tooltip`, `OnDrag`, `Focusable` and the `Overflow*Scroll` methods only exist after `ID`. A helper that returns `*gpui.Div` cannot be followed by `OnClick` until you call `ID`.
- Closures for `When`, `WhenSome` and `Map` take and return the current type. After `ID`, write `func(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div]`.

See [Elements and styling](elements-and-styling.md#conditional-styling).

### Hover and active styles get a bare refinement

`Hover`, `Active`, `Focus`, `FocusVisible` and `GroupHover` pass a `*gpui.StyleRefinement`, which has no chainable methods.

Fix: bind it to a `gpui.Styled` with the small `styled` helper from [Elements and styling](elements-and-styling.md#hover-and-active-styles). Setting `Hover` twice on one element panics while debug assertions are on.

### `OnAction` and `cx.Listener` disagree on the argument

Element `OnAction` wants `func(action A, window, cx)` with the action by value. `cx.Listener` produces `func(action *A, window, cx)`.

Fix: the `actionListener` adapter from [Actions, keymaps and focus](actions-keymaps-focus.md#handle-actions).

### Context-generic code

Go interfaces cannot have generic methods, so `gpui.AppContext` only carries untyped primitives. Code that should work with any context takes `gpui.AppContext` and calls the free functions: `gpui.AppContextNew(cx, f)`, `gpui.AppContextUpdateEntity(cx, entity, f)`, `gpui.UpdateGlobal(cx, f)`. The methods `cx.New`, `entity.Update` and so on exist on each concrete context type.

### Touching the app from another goroutine

App state has no locks. Only the main goroutine may touch entities, windows and globals.

Fix: import `github.com/megascan/gpui-go/platform` (its `init` locks the main goroutine to the main OS thread) and call `Run` from `main`. Do background work in `BackgroundSpawn` and bring the result back in a foreground task, as in [Async work and tasks](async-and-tasks.md#do-heavy-work-in-the-background).

### Blocking a foreground task

`time.Sleep`, a raw channel receive or a mutex wait inside `cx.Spawn` blocks the main thread and freezes the UI.

Fix: wait on GPUI primitives: `cx.BackgroundExecutor().Timer(d).Await(cx)`, `task.Await(cx)`, `scheduler.Oneshot` and `scheduler.Chan` receivers, `scheduler.Select`. In a task body with a deferred `recover()`, re-panic values for which `scheduler.IsCancelUnwind(r)` is true.

### Async calls fail in different ways

`weak.Update` and `AsyncWindowContext.Update` return an error when the target is gone. `AsyncApp.Update` returns nothing and panics if the app is gone. `_ = weak.Update(...)` in a loop never notices that the entity was released.

Fix: check the weak handle's error and stop. The table in [Async work and tasks](async-and-tasks.md#which-calls-can-fail) lists each call.

### Two kinds of `Px`, and no untyped lengths

`gpui.Px(10)` builds a length. `.Px(len)` and `.Px2()` on an element set horizontal padding. Untyped numbers do not satisfy the length interfaces, so `Size(500)` does not compile; write `Size(gpui.Px(500))`. Padding and gap take definite lengths, so `gpui.Auto()` is rejected there.

### Leak detection settings

| `GPUI_LEAK_DETECTION` | In tests | Outside tests |
|---|---|---|
| unset | strict | off |
| `strict` | strict | strict |
| any other value, including `0` | on (not strict) | on |

`GPUI_LEAK_DETECTION=0` does not turn detection off. Leak reports are process-wide, so do not combine `t.Parallel()` with tests that rely on them. Tasks are never tracked.

### Test context names

`TestAppContext` has no `AdvanceClock`: call `cx.Executor().AdvanceClock(d)`. `VisualTestContext` is one window inside a test app. `VisualTestAppContext` is a different type that runs on the real platform.

### The web build needs two extra steps

`Run` returns on `GOOS=js`, so end `main` with `select {}`. The browser has no system fonts: load fonts with `cx.TextSystem().AddFonts(...)` before rendering text. See [Platforms](platforms.md#web-goosjs).

### SVGs and asset images render nothing

`Svg.Path` and asset paths in `gpui.ImageSourceFromString` load through the app's asset source. Without `Application.WithAssets(...)` they silently draw nothing. `gpui.ImageSourceFromPath` reads files directly and needs no asset source.

### `examples/internal/support` is not importable

It is an internal package of this module. Copy what you need, such as font loading, into your own code.

## GPUI in general

### Changing state without `Notify`

The view does not redraw and observers are not told. Call `cx.Notify()` after every change that affects what is on screen.

### Updating an entity from inside its own update

Nested `Update` or `Read` of the entity you are updating panics with "cannot update ... while it is already being updated". Use the state you were given (`this`), or defer the work with `cx.Defer`. See [Concepts](concepts.md#updates-are-leases).

### Capturing the wrong `cx`

A closure that runs later (click handler, timer, task) must use the context it is given, not the `cx` of the function that created it. The outer context is only valid during the outer call. `cx.Listener` and `cx.Spawn` hand you a fresh one.

### Strong reference cycles

Two entities that hold strong handles to each other are never released. Make back references (child to parent, item to list) weak with `Downgrade()`.

### Missing or unstable element ids

Click, hover, active styles, tooltips, drag and scroll position are stored under the element's id. Without an id they do not work. An id that changes every frame (a counter, a pointer) resets that state each frame. Two siblings with the same id share state. Use ids derived from your data, such as a row index or a record key.

### Lists without a bounded height

`NewUniformList` and `NewList` only skip off-screen rows when their own height is bounded. Inside a parent of unbounded height they lay out everything. The same applies to `SizeFull`: 100% of an unsized parent is nothing useful. Give an ancestor a definite size.

### Actions that never fire

Work through the chain:

1. Is something focused? Focus the view (`focusHandle.Focus(window, cx)`) when its window opens.
2. Does the element tree track that focus handle (`TrackFocus`)?
3. Is the `OnAction` handler on the focused element or one of its ancestors?
4. Does the binding's context match a `KeyContext` on that path?
5. Is the binding registered (`cx.BindKeys`) and the action registered (`init`)?

`window.BindingsForAction(action)` shows which bindings would trigger an action.

### Reading a global that was never set

`cx.Global[G]()` panics if `G` was not set. Use `cx.TryGlobal[G]()` when it may be missing, or set every global at startup.

### Ignoring errors from tasks

`Detach()` throws away the task's error. Use `gpui.TaskDetachAndLogErr(task, cx)` so failures at least reach the log.
