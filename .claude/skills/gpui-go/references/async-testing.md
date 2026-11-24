# Async and testing

## Contents
- Task model
- Foreground tasks with a weak entity
- Debounce + background work + cancellation
- Fire-and-forget
- Window-bound tasks
- Blocking rules
- Testing: RunTest and contexts
- Entity tests with a fake clock
- Window tests: keystrokes, actions, clicks
- Async test bodies
- Headless screenshot
- Test pitfalls

## Task model

- `*gpui.Task[T]` (an alias of `scheduler.Task[T]`) has `Await(cx) (T, error)`, `Wait(cx) error`, `Detach()`, `Cancel()` and `IsReady()`. Rust's `Task<T>` and `Task<Result<T>>` both map to `*Task[T]`.
- A task you neither store nor `Detach` keeps running. Nothing cancels it on GC, and strict leak detection doesn't report it.
- `Cancel()` on a nil `*Task` panics, so start task fields at `scheduler.ReadyTask(struct{}{})`. Cancel the old task before you replace it.
- Awaiting needs a `scheduler.Awaiter`: `*gpui.AsyncApp`, `*gpui.AsyncWindowContext` and `*scheduler.Scope` all qualify.

| Spawn from | Signature | Body gets |
|---|---|---|
| `*gpui.Context[T]` | `cx.Spawn(func(this gpui.WeakEntity[T], cx *gpui.AsyncApp) (R, error))` | weak self + async app |
| `*gpui.Context[T]` | `cx.SpawnIn[R](window, func(this gpui.WeakEntity[T], cx *gpui.AsyncWindowContext) (R, error))` | weak self + window context |
| `*gpui.App` | `cx.Spawn(func(cx *gpui.AsyncApp) (R, error))` | async app |
| `*gpui.App` | `cx.BackgroundSpawn(func(s *scheduler.Scope) (R, error))` | runs off the main thread |
| executor | `cx.BackgroundExecutor().Spawn(...)`, `.Timer(d)` | — |

## Foreground tasks with a weak entity

The body runs on the main thread. Reach state through the weak handle, whose `Update` returns an error once the entity is released. Use that error to end loops:

```go
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
```

## Debounce + background work + cancellation

```go
func NewSearch(corpus []string) func(cx *gpui.Context[Search]) *Search {
	return func(_ *gpui.Context[Search]) *Search {
		return &Search{corpus: corpus, pending: scheduler.ReadyTask(struct{}{})}
	}
}

func (s *Search) Drop(_ *gpui.App) {
	s.pending.Cancel()
}

func (s *Search) SetQuery(query string, cx *gpui.Context[Search]) {
	s.query = query
	corpus := s.corpus
	s.pending.Cancel()
	s.pending = cx.Spawn(func(this gpui.WeakEntity[Search], cx *gpui.AsyncApp) (struct{}, error) {
		if _, err := cx.BackgroundExecutor().Timer(200 * time.Millisecond).Await(cx); err != nil {
			return struct{}{}, err
		}
		matches, err := cx.BackgroundExecutor().Spawn(func(_ *scheduler.Scope) ([]string, error) {
			var out []string
			for _, line := range corpus {
				if strings.Contains(line, query) {
					out = append(out, line)
				}
			}
			return out, nil
		}).Await(cx)
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, this.Update(cx, func(s *Search, cx *gpui.Context[Search]) {
			s.results = matches
			cx.Notify()
		})
	})
}
```

Copy the inputs (`corpus`, `query`) before the background closure. Background code must never touch entities or `*App`.

## Fire-and-forget

```go
func FireAndForget(cx *gpui.App) {
	cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
		cx.Update(func(cx *gpui.App) { cx.RefreshWindows() })
		return struct{}{}, nil
	}).Detach()

	gpui.TaskDetachAndLogErr(cx.BackgroundSpawn(func(_ *scheduler.Scope) (int, error) {
		return 0, fmt.Errorf("logged, not returned")
	}), cx)
}
```

`AsyncApp.Update` returns nothing. It panics if the app has been released, which can't happen for tasks spawned from the app itself. `TaskDetachAndLogErr` logs the error through `log/slog`.

## Window-bound tasks

```go
func WindowFromTask(cx *gpui.Context[Clock], window *gpui.Window) {
	cx.SpawnIn[struct{}](window, func(_ gpui.WeakEntity[Clock], cx *gpui.AsyncWindowContext) (struct{}, error) {
		return struct{}{}, cx.Update(func(window *gpui.Window, _ *gpui.App) {
			window.SetWindowTitle("ticking")
		})
	}).Detach()
}
```

`AsyncWindowContext.Update` returns `error`, which is set once the window is closed. `SpawnIn` often needs its type argument written out (`[struct{}]`).

## Blocking rules

- A foreground body may block only through gpui or scheduler primitives: `Task.Await`, `Timer(...).Await`, `scheduler.Oneshot[T]()` (`tx.Send(v)` and `rx.Recv(cx)`), `scheduler.Select`, `scheduler.Join`.
- `time.Sleep`, `<-ch`, `sync.WaitGroup.Wait` or a mutex wait in a foreground task freezes the UI and deadlocks tests.
- To bridge from a plain goroutine, send through `scheduler.Oneshot` and receive in the task.
- A `recover()` inside a task body must re-panic `scheduler.IsCancelUnwind(r)` values.

## Testing: RunTest and contexts

`gpui.RunTest(t, func(tc *gpui.TestRun) {...}, opts...)` is the Go form of `#[gpui::test]`. It gives a deterministic single-threaded scheduler, a fake clock, a test platform with no real windows, and teardown that releases contexts, drains tasks and then panics on strict leaks.

| You need | Use |
|---|---|
| Entities, globals, tasks | `cx := tc.Cx()` (`*gpui.TestAppContext`): `New`, `Update(func(*gpui.App))`, `UpdateR`, `RunUntilParked()`, `Executor()` |
| A window + view | `view, vcx := tc.Cx().AddWindowView(func(*gpui.Window, *gpui.Context[V]) *V)`. Release `view` |
| Input simulation | `vcx.SimulateKeystrokes("down space")`, `SimulateInput("hi")`, `SimulateClick(pt, gpui.Modifiers{})`, `SimulateMouseMove/Down/Up`, `SimulateResize(size)`, `DispatchAction(a)` |
| Locating elements | `.DebugSelector(func() string { return "row-1" })` on the element, then `vcx.DebugBounds("row-1")` |
| Focus | `vcx.Focus(view)` (the view must implement `Focusable`) |
| Fake time | `cx.Executor().AdvanceClock(d)`. There is no `AdvanceClock` on `TestAppContext` itself |
| Seeds | `gpui.Iterations(n)`, `gpui.Seed(s)`, `tc.Rand()`, env `SEED=`, `ITERATIONS=`. `gpui.PropertyTest` for property tests |
| Async test body | `tc.BlockTest(func(s *scheduler.Scope) {...})`, then `task.Await(s)` |

Others: `cx.SimulatePromptAnswer`, `cx.Events[T, E](entity)`, `cx.Notifications[T](entity)` and `entity.NextEvent[E](cx)`. A `VisualTestAppContext` is a different type: a real platform with offscreen windows, which avoid in CI.

## Entity tests with a fake clock

```go
func TestSearchDebounce(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		search := cx.New(NewSearch([]string{"apple", "banana", "grape"}))
		defer search.Release()

		search.Update(cx, func(s *Search, cx *gpui.Context[Search]) { s.SetQuery("ap", cx) })
		cx.RunUntilParked()
		if n := search.ReadWith(cx, func(s *Search, _ *gpui.App) int { return len(s.results) }); n != 0 {
			t.Fatalf("results before debounce = %d, want 0", n)
		}

		cx.Executor().AdvanceClock(200 * time.Millisecond)
		cx.RunUntilParked()
		if n := search.ReadWith(cx, func(s *Search, _ *gpui.App) int { return len(s.results) }); n != 2 {
			t.Fatalf("results = %d, want 2", n)
		}
	})
}
```

## Window tests: keystrokes, actions, clicks

These test the keyboard todo list (`TodoList`, `NewTodoList`, `Item`, and the actions `SelectNext` and `ToggleDone`) from the actions and focus reference:

```go
func testItems() []Item {
	return []Item{{Title: "a"}, {Title: "b"}, {Title: "c"}}
}

func TestTodoKeyboard(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		tc.Cx().Update(func(cx *gpui.App) {
			cx.BindKeys([]gpui.KeyBinding{
				gpui.NewKeyBinding("down", SelectNext{}, gpui.Some("TodoList")),
				gpui.NewKeyBinding("space", ToggleDone{}, gpui.Some("TodoList")),
			})
		})
		view, cx := tc.Cx().AddWindowView(func(_ *gpui.Window, cx *gpui.Context[TodoList]) *TodoList {
			return NewTodoList(testItems())(cx)
		})
		defer view.Release()
		cx.Focus(view)

		cx.SimulateKeystrokes("down down space")

		selected, done := cx.UpdateR(func(_ *gpui.Window, cx *gpui.App) int { return view.Read(cx).selected }),
			cx.UpdateR(func(_ *gpui.Window, cx *gpui.App) bool { return view.Read(cx).items[2].Done })
		if selected != 2 || !done {
			t.Fatalf("selected = %d, done = %v; want 2, true", selected, done)
		}
	})
}

func TestTodoClickSelectsRow(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		view, cx := tc.Cx().AddWindowView(func(_ *gpui.Window, cx *gpui.Context[TodoList]) *TodoList {
			return NewTodoList(testItems())(cx)
		})
		defer view.Release()
		cx.SimulateResize(gpui.NewSize(gpui.Px(300), gpui.Px(300)))
		cx.RunUntilParked()

		bounds, ok := cx.DebugBounds("row-1")
		if !ok {
			t.Fatal("row-1 was not rendered")
		}
		cx.SimulateClick(bounds.Center(), gpui.Modifiers{})
		cx.RunUntilParked()

		if got := cx.UpdateR(func(_ *gpui.Window, cx *gpui.App) int { return view.Read(cx).selected }); got != 1 {
			t.Fatalf("selected = %d, want 1", got)
		}
	})
}
```

Focus and blur listeners (`cx.OnFocus`/`OnBlur`) only fire while the test window is active:

```go
		cx.Update(func(window *gpui.Window, _ *gpui.App) { window.ActivateWindow() })
		cx.Focus(editor)
```

## Async test bodies

From examples/testing/main_test.go. Await a task the view returns, then drain the detached work:

```go
		tc.BlockTest(func(s *scheduler.Scope) {
			cx := tc.Cx()
			counter := cx.New(NewCounter)
			defer counter.Release()

			if _, err := counter.UpdateR(cx, func(counter *Counter, cx *gpui.Context[Counter]) *gpui.Task[struct{}] { return counter.load(cx) }).Await(s); err != nil {
				panic(err)
			}
```

## Headless screenshot

This renders a view to a PNG with the real GPU renderer and no window. It was verified on Linux with Vulkan. The headless context opens a test-platform window, which defaults to scale factor 2, so a 400x200 window gives an 800x400 PNG. Run it with `ZED_HEADLESS=1 go run ./shot out.png`, then open the PNG and look at it:

```go
func main() {
	cx := gpui.HeadlessAppContextWithPlatform(
		platform.CurrentPlatform(true).TextSystem(),
		gpui.NoAssets{},
		platform.CurrentHeadlessRenderer,
	)
	defer cx.Close()
	window, err := cx.OpenWindow(gpui.NewSize(gpui.Px(400), gpui.Px(200)), func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Card] {
		return cx.New(func(_ *gpui.Context[Card]) *Card { return &Card{} })
	})
	if err != nil {
		panic(err)
	}
	cx.RunUntilParked()
	img, err := cx.CaptureScreenshot(window.AnyWindowHandle)
	if err != nil {
		panic(err)
	}
	out, err := os.Create(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		panic(err)
	}
}
```

Pass your own `AssetSource` instead of `gpui.NoAssets{}` when the view uses SVG or asset images. Windows has no headless renderer.

## Test pitfalls

- **Leaks panic the test.** Inside `go test`, leak detection defaults to strict. Every `cx.New`, `Clone`, `Root`, `FocusHandle` and `Upgrade` needs its `Release`, and every subscription needs `Unsubscribe` or `Detach`. Rerun with `LEAK_BACKTRACE=1` to see where the leaked handle was allocated.
- **`cx.Processor` leaks.** It holds a strong handle, so a list built with it fails strict mode. Snapshot the data in `Render`, or use `cx.WeakEntity()`.
- **One failure cascades.** `t.Fatalf` inside `RunTest` skips teardown. The leaked handles then show up as "Exited with leaked handles" in the *next* test. Fix the first failure in the log.
- **Process-global reports.** Don't `t.Parallel()` tests that rely on leak checks.
- **Inactive window.** Focus listeners need `window.ActivateWindow()`. Key dispatch and `DispatchAction` work without it.
- **Nothing happened yet.** Effects and spawned tasks run on `RunUntilParked()`, and timers fire only after `Executor().AdvanceClock(d)`.
- **`DebugSelector` needs debug assertions.** It is a no-op under `-tags gpui_release`.
- **No real windows, clipboard or session bus in tests.** The test platform fakes them. Keep display-touching checks (real windows, screenshots) out of `go test`, or gate them behind an env var of your own.
