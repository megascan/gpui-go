---
name: gpui-go
description: Builds, tests and debugs desktop UIs in Go with gpui-go (github.com/megascan/gpui-go), the Go port of Zed's GPUI. Covers entities and views, Render and div styling, uniform lists, actions, keymaps and focus, async tasks, handle release, and tests built on TestAppContext and VisualTestContext. Use it when code imports github.com/megascan/gpui-go or its platform or scheduler packages, when someone wants a native or GPU-accelerated Go desktop UI built with GPUI, and when Rust GPUI or Zed UI code is being ported to Go. Also use it for gpui-go errors, including leaked handles reported by go test ("collected without Release"), "cannot update X while it is already being updated", windows that never appear, actions or key bindings that never fire, OnClick missing from a div, and cgo or wgpu build failures on Linux.
---

# gpui-go

gpui-go is a Go port of Zed's GPUI (Rust). Its API follows upstream closely, with a fixed set of Go deviations. Treat the code as the source of truth. When a signature is in doubt, grep the module source (`go env GOMODCACHE`, or the `replace` path) and don't trust memory of Rust GPUI.

## Core model

- One `*gpui.App` owns all state. You hold `gpui.Entity[T]` handles, which are counted references, not pointers. Each handle comes from `cx.New`, `Clone`, `cx.Entity()`, `WindowHandle.Root`, `cx.FocusHandle()` or `Focusable.FocusHandle`, and you must `Release()` it yourself. For entity and focus handles the GC is only a late safety net, and `go test` reports a collected handle as a leak. The GC never stops a task or cancels a subscription.
- A view is an entity whose `*T` has `Render(window *gpui.Window, cx *gpui.Context[T]) gpui.IntoElement`. Render runs again after `cx.Notify()`. It builds a fresh element tree each frame from `gpui.NewDiv()` chains.
- You change state with `entity.Update(cx, func(t *T, cx *gpui.Context[T]) {...})`. Effects such as notify and emit are flushed after the outermost update.
- The keyboard works through actions. An action handler runs only when its element is on the focused path: `KeyContext` + `TrackFocus` + a focused handle + a binding for that context.
- Async code runs as coroutines on the main thread. `cx.Spawn` gives a `WeakEntity[T]` plus an `*AsyncApp`. Code may only block by awaiting gpui or scheduler primitives.

## Default workflow

Copy this checklist and tick items as you go:

```
- [ ] 1. Module + deps set up; `go build` of the skeleton passes with cgo on
- [ ] 2. Window opened from gpui.DefaultWindowOptions(), fields set afterwards
- [ ] 3. Root view entity with Render; children are entities or components
- [ ] 4. Elements/styling; .ID(...) before OnClick/Active/Tooltip/OverflowScroll
- [ ] 5. Actions + keymap + focus wired (KeyContext, TrackFocus, OnAction, BindKeys, initial focus)
- [ ] 6. Async work in stored or detached tasks; entity access via WeakEntity
- [ ] 7. Release discipline: every handle, subscription and task has an owner (Drop)
- [ ] 8. Tests with gpui.RunTest + TestAppContext/VisualTestContext
- [ ] 9. go vet ./... && go test ./... clean (strict leak detection on); run or screenshot the UI
```

1. **Module.** Go 1.27.3+ is required, because the port uses generic methods. Linux needs cgo and a C toolchain, since wgpu-native is linked statically. No version tag is published yet, so consume a local checkout through a `replace` directive:
   ```
   require github.com/megascan/gpui-go v0.0.0
   replace github.com/megascan/gpui-go => /path/to/gpui-go
   ```
   Copy the checkout's `go.sum` and its `require` blocks (direct and indirect), or run `go mod tidy` in *your* module. Import `github.com/megascan/gpui-go` (package `gpui`) and `github.com/megascan/gpui-go/platform`. Importing `platform` locks the main goroutine to the OS thread, so call `platform.Application().Run(...)` from `main`.
2. **Window.** Start from `options := gpui.DefaultWindowOptions()` and then set fields. A bare `gpui.WindowOptions{}` leaves `Show`, `Focus`, `IsResizable` and the rest false, which gives an invisible window on macOS and Windows. Linux ignores `Show`, which hides the bug there. `OpenWindow`'s builder returns the root entity, and the window takes over that handle, so don't release it.
3. **Views.** Write one struct per stateful view. Create child views once in the constructor, store them, and render them with `Child(child.Clone())`, because `Child` consumes the handle. Release each one in `Drop(cx *gpui.App)`. Stateless pieces become `RenderOnce` components (see references/elements-styling.md).
4. **Elements.** Every styling method returns the receiver type. `.ID(x)` turns `*gpui.Div` into `*gpui.Stateful[*gpui.Div]`, which is what adds `OnClick`, `Active`, `Tooltip`, `OnDrag` and `OverflowScroll`. For long lists, use `gpui.NewUniformList` inside a parent with a bounded height (`Flex1`/`HFull`/`H`).
5. **Actions/focus.** An action implements `Name`, `BoxedClone` and `PartialEq` and is registered in `init` with `gpui.RegisterAction`. You can also generate it with actiongen. Bind keys with `cx.BindKeys([]gpui.KeyBinding{gpui.NewKeyBinding("down", SelectNext{}, gpui.Some("TodoList"))})`. Then put `KeyContext("TodoList")`, `TrackFocus(h)` and `OnAction(...)` on the element and focus `h` once (see the skeleton in references/actions-focus-events.md).
6. **Async.** `task := cx.Spawn(func(this gpui.WeakEntity[T], cx *gpui.AsyncApp) (R, error) {...})`. Either store the task and `Cancel` it in `Drop`, or call `.Detach()`. For timers use `cx.BackgroundExecutor().Timer(d).Await(cx)`. For CPU work use `cx.BackgroundExecutor().Spawn(...).Await(cx)`. Never use `time.Sleep` or a raw channel receive inside a foreground task.
7. **Release.** Give every handle field a matching line in `Drop`. Seed task fields with `scheduler.ReadyTask(struct{}{})` so that `Cancel` is safe. Subscriptions are either stored and `Unsubscribe()`d, or `.Detach()`ed on purpose.
8. **Tests.** Wrap each test in `gpui.RunTest(t, func(tc *gpui.TestRun) {...})`. Use `tc.Cx()` for entity tests and `tc.Cx().AddWindowView(...)` for window, click and keystroke tests. See references/async-testing.md.
9. **Verify.** Run `go vet ./...` and `go test ./...`. In tests, leak detection is strict by default, and a leaked handle, subscription or focus handle panics with "Exited with leaked handles". Then build and run the app (`go run .`) and look at it, or render a PNG headlessly (references/async-testing.md, "Headless screenshot"). Check the result by eye. A clean compile is not enough for UI work.

## Canonical skeleton

`examples/hello_world/main.go`, verbatim. It compiles in an external module as-is:

```go
package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

// HelloWorld is the hello_world example view.
type HelloWorld struct {
	text gpui.SharedString
}

func swatch(bg, border gpui.Hsla) *gpui.Div {
	return gpui.NewDiv().
		Size8().
		Bg(bg).
		Border1().
		BorderDashed().
		RoundedMd().
		BorderColor(border)
}

// Render renders the view.
func (h *HelloWorld) Render(_ *gpui.Window, _ *gpui.Context[HelloWorld]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap3().
		Bg(gpui.Rgb(0x505050)).
		Size(gpui.Px(500.0)).
		JustifyCenter().
		ItemsCenter().
		ShadowLg().
		Border1().
		BorderColor(gpui.Rgb(0x0000ff)).
		TextXl().
		TextColor(gpui.Rgb(0xffffff)).
		ChildText(fmt.Sprintf("Hello, %s!", h.text)).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				Child(swatch(gpui.Red(), gpui.White())).
				Child(swatch(gpui.Green(), gpui.White())).
				Child(swatch(gpui.Blue(), gpui.White())).
				Child(swatch(gpui.Yellow(), gpui.White())).
				Child(swatch(gpui.Black(), gpui.White()).RoundedMd()).
				Child(swatch(gpui.White(), gpui.Black())),
		)
}

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

func main() {
	runExample()
}
```

A full keyboard-driven app (actions, focus, `Drop` and a `uniform_list`) with passing tests is in references/actions-focus-events.md and references/async-testing.md. Most repo examples import `examples/internal/support`, which other modules cannot import, so delete the `support.LoadFonts` call when you copy one. On desktop it only loads fonts for the web build.

## Top pitfalls

Go-port-specific items come first:

| Symptom | Cause | Fix |
|---|---|---|
| Window never appears on macOS/Windows | `gpui.WindowOptions{...}` literal, so `Show` is false | `options := gpui.DefaultWindowOptions()`, then set fields |
| `go test` panics "Exited with leaked handles ... collected without Release" | A counted handle (entity, focus handle) or subscription was dropped unreleased | `defer h.Release()`, or release it in the owner's `Drop(cx *gpui.App)`. Set `LEAK_BACKTRACE=1` to see the allocation site |
| Leak report blames `(*Context).Processor` | `cx.Processor` holds a strong entity handle for the closure's lifetime | Snapshot the data the callback needs in `Render`, or capture `cx.WeakEntity()` and call `weak.Update` |
| `gpui: use of a released entity handle` on the second frame | `Child(entity)`, `AnyViewFromEntity(e)` and the `OpenWindow` builder's return value take over the handle | Pass `e.Clone()` if you keep `e`. `TrackFocus(h)` clones internally, so keep and release your own `h` |
| Work keeps running after its view is gone | Neither the GC nor dropping the value stops `*Task` or `Subscription` | Store it and `Cancel`/`Unsubscribe` it in `Drop`, or `.Detach()` it on purpose. Init task fields with `scheduler.ReadyTask(struct{}{})`, because `Cancel` on a nil `*Task` panics |
| `OnClick undefined (type *gpui.Div ...)` | `OnClick`, `Active`, `Tooltip`, `OnDrag` and `OverflowScroll` are `StatefulInteractive` methods | Call `.ID(...)` first. After it, `When` closures take and return `*gpui.Stateful[*gpui.Div]` |
| `Hover(func(s *gpui.StyleRefinement) ...)` has no fluent methods, or a chain on it fails to compile | The callback gets a bare `*StyleRefinement`. A bound `Styled[*StyleRefinement]` returns `*StyleRefinement`, so you can't chain twice | `var st gpui.Styled[*gpui.StyleRefinement]; gpui.BindStyled(&st, s); st.Border2(); return st.BorderColor(c)` |
| `OnAction(cx.Listener(...))` doesn't type-check | Element `OnAction` wants `func(A, *Window, *App)` by value, and `cx.Listener` yields `func(*A, ...)` | Use the 4-line `actionListener` adapter (references/actions-focus-events.md) |
| Key binding does nothing | Missing key context, focus or registration | `KeyContext("X")` matching the binding's `gpui.Some("X")`, plus `TrackFocus(h)`, plus focus `h` once, plus `RegisterAction` in `init` |
| `cannot update X while it is already being updated` | Re-entrant `Update`/`Read` of the same entity inside its own update | Use the `this *T` you were given, or `cx.Defer(...)` |
| UI freezes | `time.Sleep`, `<-ch` or a mutex wait inside a foreground task | `Timer(d).Await(cx)`, `scheduler.Oneshot`, or `BackgroundExecutor().Spawn(...).Await(cx)` |
| Panic in async code after the app quits | `AsyncApp.Update` returns nothing and panics if the app is gone, unlike `WeakEntity.Update`, which returns `error` | Reach entities through `WeakEntity.Update`/`UpdateR`. Stop the loop when it returns an error |
| `cannot use 500 (untyped int) as gpui.IntoLength` | Lengths are typed | `gpui.Px(500)`, `gpui.Rems(1.5)`, `gpui.Relative(0.5)`. `.Px(...)`/`.Px2()` on an element means horizontal padding |
| `OnFocus`/`OnBlur` never fire in a test | Test windows start inactive | `cx.Update(func(w *gpui.Window, _ *gpui.App) { w.ActivateWindow() })` before focusing |
| A later test reports another test's leaks | A failing `t.Fatalf` inside `RunTest` skips teardown, and leak reports are process-global | Fix the first failure. Don't `t.Parallel()` leak-checked tests |
| `DebugBounds` returns false | `DebugSelector` is ignored without debug assertions (`-tags gpui_release`) | Run such tests in the default build |
| `build constraints exclude all Go files in .../needscgo` | `CGO_ENABLED=0` on Linux | Enable cgo and install a C toolchain plus Vulkan or GL drivers |
| Need a generic helper over any context | Go interfaces cannot carry generic methods | Take `gpui.AppContext` and call free funcs (`gpui.AppContextNew(cx, build)`, `gpui.UpdateGlobal(cx, f)`). Sometimes the type args must be written out: `cx.SpawnIn[struct{}](window, ...)` |

## References

Read only what the task needs:

- references/api-cheatsheet.md: entities, contexts, windows, globals, subscriptions, the "who releases what" table and leak-detection modes.
- references/elements-styling.md: Div and its mixins, units, colors, text, conditional styling, components, uniform_list and list, img/svg/canvas, popovers, animation and tooltips.
- references/actions-focus-events.md: actions (hand-written and actiongen), keymaps, key contexts, focus handles, mouse, keyboard, drag-and-drop, and the full todo-list example.
- references/async-testing.md: tasks, timers, background work, debouncing, RunTest, VisualTestContext clicks and keystrokes, fake clock, and headless screenshots.
- references/platforms.md: per-OS build requirements, backend selection, env vars, the web (wasm) build and known limits.
- references/rust-to-go.md: the D-* deviation table and a Rust to Go API mapping, for porting upstream GPUI or Zed code.
