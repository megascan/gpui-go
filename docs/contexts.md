# Contexts

GPUI passes _context parameters_ (usually named `cx`) to your code. A context gives access to application state and services: global state, windows, entities and the platform.

---

## `*gpui.App`

The root context, with access to the application's global state. It owns the data of every entity and can read or update the data behind an `Entity[T]`.

## `*gpui.Context[T]`

The context you get when you work with an `Entity[T]`. It adds methods for that entity, such as notifying observers (`cx.Notify()`) and emitting events (`cx.Emit(event)`). `Context[T]` embeds `*App`, so it has every `App` method, and `cx.App` passes it to any function that takes an `*App`.

## `*gpui.AsyncApp` and `*gpui.AsyncWindowContext`

The contexts above are only valid during the call they are passed to. Call `cx.ToAsync()` to get an async context, or use the one a foreground task receives from `cx.Spawn`. An async context can be kept across awaits (`task.Await(acx)`). Entity and window calls through an async context return an `error`, because the window or the whole app may be gone by the time the call runs.

## `*gpui.TestAppContext`

Works like the async contexts, but panics when the app or window it refers to no longer exists, and adds test helpers (simulated input, prompts, `RunUntilParked`, ...). Tests usually get one from `gpui.RunTest`.

---

# Other core types

## `*gpui.Window`

The state of one application window. A window has a root view (an `Entity` whose `*T` implements `Render[T]`) that it can read and update. `Window` is not a context: functions that also touch global state take the window and an `*App` (or a context that embeds one) side by side. `WindowHandle[V].Update` gives you the `*Window` for a handle.

## `gpui.Entity[T]`

A handle to a piece of state. The `App` owns the data; you read and change it through a context: `entity.Read(cx)`, `entity.Update(cx, func(state *T, cx *gpui.Context[T]) {...})`. If `*T` implements `Render[T]`, the entity is also called a view. Other entities and windows can observe an entity; their callbacks run when the entity's `Context` calls `Notify`.

An `Entity[T]` value is a counted handle: `Clone()` makes another one, and `Release()` drops yours. Release every handle you create, usually with `defer`.
