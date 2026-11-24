# Concepts

This page explains the model behind gpui-go: who owns state, how you reach it, when changes show up on screen, and how cleanup works without destructors. The code samples come from [snippets/concepts/concepts.go](snippets/concepts/concepts.go), which has a leak-checked test.

## The App owns all state

In most Go UI code you would hold pointers to your models and views. In GPUI you do not. You hand each piece of state to the `App`, and it gives you back a handle:

```go
	document := cx.New(func(_ *gpui.Context[Document]) *Document {
		return &Document{title: "untitled"}
	})
	defer document.Release()
```

That state is an *entity*. `document` is a `gpui.Entity[Document]`: a typed id plus a reference count. It has no access to the state by itself. To read or change the state you go through a context:

- `document.Read(cx)` returns a `*Document` for reading.
- `document.ReadWith(cx, func(d *Document, cx *gpui.App) R) R` reads a value out.
- `document.Update(cx, func(d *Document, cx *gpui.Context[Document]) {...})` changes it.

An entity whose pointer type has a `Render` method is a *view*. Everything else is a *model*. They are the same machinery.

Why this design: because every entity lives in one place, GPUI can notify observers, re-render the right windows, and keep callbacks from touching state that is in the middle of an update. Entities can refer to each other freely through handles without Go pointer cycles between your structs. Zed's post [Ownership and data flow in GPUI](https://zed.dev/blog/gpui-ownership) explains the original design; [docs/ownership-and-data-flow.md](../ownership-and-data-flow.md) walks through it in Go.

## Handles are counted

An `Entity[T]` handle counts as one owner until you call `Release`. The entity lives while any handle to it is unreleased.

- `Clone()` returns a new handle and adds one to the count.
- `Release()` drops your handle. Releasing the same handle twice is a no-op; using it after release panics.
- Copying the handle value (`b := a`) does not add to the count. `a` and `b` are the same handle, so release only one of them.
- `Downgrade()` returns a `gpui.WeakEntity[T]` that does not keep the entity alive. `Upgrade()` turns it back into a strong handle if the entity still exists:

```go
	weak := document.Downgrade()
	if strong, ok := weak.Upgrade(); ok {
		fmt.Println(strong.Read(cx).title)
		strong.Release()
	}
```

Use weak handles for back references (a child pointing at its parent) and in long-running tasks. Two entities that hold strong handles to each other keep each other alive forever.

## Contexts

GPUI passes a context, usually named `cx`, into every callback. Which one you get tells you what you can do:

| Context | Where you get it | What it adds |
|---|---|---|
| `*gpui.App` | `Run`, `OpenWindow`, element callbacks | everything app-wide: entities, windows, globals, keymaps, the platform |
| `*gpui.Context[T]` | `cx.New`, `entity.Update`, `Render` | `*App` plus the entity `T` itself: `Notify`, `Emit`, `Observe`, `Subscribe`, `Spawn`, `Listener` |
| `*gpui.AsyncApp` | `Spawn`, `cx.ToAsync()` | survives across awaits; holds the app weakly |
| `*gpui.AsyncWindowContext` | `SpawnIn` | an async context tied to one window |
| `*gpui.TestAppContext` | `tc.Cx()` in tests | an app on the test platform |

`Context[T]` embeds `*App`, so it has every `App` method. When a function wants an `*App`, pass `cx.App`.

`*gpui.Window` is not a context. Functions that need a window take it next to `cx`: `Render(window, cx)`, `OnClick(func(event, window, cx))`.

Contexts are only valid for the call they were passed to. Do not store them. A callback that runs later (a click handler, a timer) gets its own.

[docs/contexts.md](../contexts.md) has the full list.

## Updates are leases

While `Update` runs, GPUI takes the entity's state out of the app and lends it to your callback. When the callback returns, the state goes back. This is why a nested update of the same entity panics with "cannot update ... while it is already being updated": the state is not there to lend.

You rarely hit this, because inside `Update` you already have the state as `d`. When you do need to update the same entity after the current update finishes, defer it:

```go
func reentrant(document gpui.Entity[Document], cx *gpui.App) {
	document.Update(cx, func(d *Document, cx *gpui.Context[Document]) {
		d.title = "draft"
		cx.Defer(func(cx *gpui.App) {
			document.Update(cx, func(d *Document, cx *gpui.Context[Document]) {
				d.Rename("final", cx)
			})
		})
	})
}
```

## Notify, emit and the effect queue

Entities talk to each other in two ways:

- `cx.Notify()` says "my state changed". Others listen with `cx.Observe(entity, ...)`.
- `cx.Emit(event)` sends a typed event. Others listen with `cx.Subscribe(entity, ...)` and name the event type in their callback. Any Go type can be an event.

```go
func (d *Document) Rename(title string, cx *gpui.Context[Document]) {
	d.title = title
	cx.Notify()
	cx.Emit(Changed{})
}
```

```go
func NewEditor(document gpui.Entity[Document], cx *gpui.Context[Editor]) *Editor {
	return &Editor{
		document: document.Clone(),
		observation: cx.Observe(document, func(_ *Editor, _ gpui.Entity[Document], cx *gpui.Context[Editor]) {
			cx.Notify()
		}),
		subscription: cx.Subscribe(document, func(this *Editor, _ gpui.Entity[Document], _ *Changed, _ *gpui.Context[Editor]) {
			this.renames++
		}),
	}
}
```

Neither call runs the listeners right away. Both push an *effect* onto a queue, and the app flushes the queue when the outermost update returns. So listeners always see a finished state, never a half-done one, and a listener may update the entity that notified without hitting the lease panic.

`Observe` and `Subscribe` return a `gpui.Subscription`. Keep it and call `Unsubscribe` when you are done (as `Editor.Drop` below does), or call `Detach()` to keep the listener until either entity is released. A subscription you drop on the floor keeps firing.

Event types are not checked at compile time. If you subscribe to an event type the entity never emits, your callback never runs.

## The render cycle

1. Something calls `cx.Notify()` on a view, or input arrives.
2. GPUI marks the window dirty and schedules a frame.
3. On the next frame it calls `Render` on the root view. Child views that appear in the tree render too.
4. `Render` returns a fresh element tree. GPUI lays it out (flexbox and grid), runs prepaint (hit testing, scroll offsets) and paints it to the GPU.

Elements are rebuilt every frame and thrown away after it. Anything that must survive between frames lives either in your entities or in element state that GPUI keeps under the element's id (hover, click, scroll position). That is why interactive elements need a stable `ID`.

`Render` should be cheap and free of side effects: build elements from state, nothing else. Do work in event handlers and tasks, then `Notify`.

A view that rarely changes can be wrapped with `entity.Cached(style)`, which reuses the last render until the view is notified.

## Views inside views

To show one view inside another, store its handle and pass it to `Child`. `Child(entity)` moves the handle into the element tree, and the tree releases it when the frame is done. Since `Render` runs every frame, pass a clone:

```go
func (w *Workspace) Render(_ *gpui.Window, _ *gpui.Context[Workspace]) gpui.IntoElement {
	return gpui.NewDiv().SizeFull().Child(w.editor.Clone())
}
```

The same goes for `gpui.AnyViewFromEntity` and the entity you return from the `OpenWindow` builder: they take over the handle you pass in.

## Release and Drop

Rust frees things when their last owner goes out of scope. Go has no destructors, so gpui-go makes each release explicit:

| Thing | Rust | gpui-go |
|---|---|---|
| `Entity[T]` | dropped with its owner | `Release()` |
| `FocusHandle` | dropped with its owner | `Release()` |
| `Subscription` | unsubscribes on drop | `Unsubscribe()`, or `Detach()` to keep it |
| `Task` | cancels on drop | `Cancel()`, or `Detach()` to keep it |
| entity state | `Drop` impl | `Drop(cx *gpui.App)` method |

When the last handle to an entity is released, GPUI calls `Drop(cx *gpui.App)` on the state, if the type has one. Put your own cleanup there:

```go
func (e *Editor) Drop(_ *gpui.App) {
	e.observation.Unsubscribe()
	e.subscription.Unsubscribe()
	e.document.Release()
}
```

Rules of thumb:

- Every `cx.New`, `Clone()`, `cx.Entity()`, `cx.FocusHandle()`, `WindowHandle.Root(...)` and `Focusable.FocusHandle(cx)` hands you a handle to release.
- Handles passed into your callbacks (the `Entity[W]` in an `Observe` callback, for example) are borrowed for that call. `Clone()` them to keep them.
- For short-lived handles, `defer h.Release()`. For handles stored in an entity, release them in its `Drop`.
- The garbage collector releases forgotten entity and focus handles eventually, as a safety net. It never cancels tasks or unsubscribes subscriptions.
- Tests run with strict leak detection, which reports handles the GC collected without `Release`. See [Testing](testing.md#leak-detection).

## Globals

App-wide state that is not tied to one entity, such as settings or a theme, goes in a global. Any type can be a global:

```go
func globals(cx *gpui.App) {
	cx.SetGlobal(Theme{Dark: true})
	theme := cx.Global[Theme]()
	fmt.Println(theme.Dark)
	gpui.UpdateGlobal(cx, func(theme *Theme, _ *gpui.App) {
		theme.Dark = false
	})
	if theme, ok := cx.TryGlobal[Theme](); ok {
		fmt.Println(theme.Dark)
	}
}
```

`cx.Global[G]()` panics if the global is not set; `TryGlobal` reports it instead. `cx.ObserveGlobal[G](...)` runs a callback whenever the global is updated.

## One thread

All of the above happens on the main thread. Background goroutines never touch entities directly. They hand results back through a foreground task. See [Async work and tasks](async-and-tasks.md).
