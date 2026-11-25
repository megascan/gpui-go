# Testing

gpui-go ships a test platform: no real windows, a deterministic scheduler, a fake clock and simulated input. Tests run with plain `go test`. The tests in this guide are in [snippets/testing](snippets/testing/counter_test.go) and run with `go test ./...` like the rest of the repo.

The view under test:

```go
type Counter struct {
	count int
}

// Add changes the count, notifies observers and emits Changed.
func (c *Counter) Add(delta int, cx *gpui.Context[Counter]) {
	c.count += delta
	cx.Notify()
	cx.Emit(Changed{Count: c.count})
}
```

## Write a test

Wrap the test body in `gpui.RunTest`. It builds a test app, runs your function and tears everything down:

```go
func TestCounter_Add(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{} })
		defer counter.Release()

		counter.Update(cx, func(c *Counter, cx *gpui.Context[Counter]) {
			c.Add(2, cx)
		})

		got := counter.ReadWith(cx, func(c *Counter, _ *gpui.App) int { return c.count })
		if got != 2 {
			t.Fatalf("count = %d, want 2", got)
		}
	})
}
```

- `tc.Cx()` returns a `*gpui.TestAppContext`. It works like an `*gpui.App` for creating, reading and updating entities.
- `ReadWith` reads a value out of the entity. Copy values out; do not keep the `*Counter` pointer past the callback.
- Release every handle you create. Test binaries run with strict leak detection (see [below](#leak-detection)).

## Check events

Subscribe from the test to see what an entity emits:

```go
		var events []int
		subscription := cx.UpdateR(func(cx *gpui.App) gpui.Subscription {
			return cx.Subscribe(counter, func(_ gpui.Entity[Counter], event *Changed, _ *gpui.App) {
				events = append(events, event.Count)
			})
		})
		defer subscription.Unsubscribe()
```

Events and notifications are delivered after the outermost update returns, so check them after `counter.Update(...)`, not inside it.

## Open a window

`AddWindowView` opens a test window with your view as root and returns the view plus a `*gpui.VisualTestContext`. The window is laid out and painted, so you can drive it with input.

### Click an element

Name the element with `DebugSelector`:

```go
			gpui.NewDiv().
				ID("increment").
				DebugSelector(func() string { return "increment" }).
				Size8().
```

Then find its bounds and click it:

```go
func TestCounter_Click(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		counter, vcx := tc.Cx().AddWindowView(func(_ *gpui.Window, _ *gpui.Context[Counter]) *Counter {
			return &Counter{}
		})
		defer counter.Release()

		bounds, ok := vcx.DebugBounds("increment")
		if !ok {
			t.Fatal("increment button was not painted")
		}
		vcx.SimulateClick(bounds.Center(), gpui.Modifiers{})

		got := counter.ReadWith(vcx, func(c *Counter, _ *gpui.App) int { return c.count })
		if got != 1 {
			t.Fatalf("count = %d, want 1", got)
		}
	})
}
```

`DebugSelector` is recorded only when debug assertions are on, which is the default. A build with `-tags gpui_release` drops it.

Other input helpers on `VisualTestContext`: `SimulateMouseMove`, `SimulateMouseDown`, `SimulateMouseUp`, `SimulateInput` (typed text), `SimulateModifiersChange`, `SimulateResize`, `SimulateEvent`.

### Press keys

Focus the view, then type keystrokes. Bindings and `OnAction` handlers run as they would in a real window. From [snippets/actions/menu_test.go](snippets/actions/menu_test.go):

```go
func TestMenu_KeyboardNavigation(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		cx.Update(BindMenuKeys)

		menu, vcx := cx.AddWindowView(func(_ *gpui.Window, cx *gpui.Context[Menu]) *Menu {
			return NewMenu(cx)
		})
		defer menu.Release()
		vcx.Focus(menu)

		vcx.SimulateKeystrokes("down down up enter")

		got := menu.ReadWith(vcx, func(m *Menu, _ *gpui.App) int { return m.confirmed })
		if got != 1 {
			t.Fatalf("confirmed = %d, want 1", got)
		}
	})
}
```

`vcx.DispatchAction(action)` dispatches an action directly, without a key binding.

## Time and tasks

Nothing runs in the background on its own. Pending tasks run when you call `RunUntilParked`, and timers fire when you move the fake clock:

```go
func TestClock_Ticks(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		clock := cx.New(NewClock)
		defer clock.Release()

		cx.Executor().AdvanceClock(3 * time.Second)
		cx.RunUntilParked()

		got := clock.ReadWith(cx, func(c *Clock, _ *gpui.App) int { return c.seconds })
		if got != 3 {
			t.Fatalf("seconds = %d, want 3", got)
		}
	})
}
```

`AdvanceClock` lives on the executor (`cx.Executor()`), not on the test context. Use GPUI timers in code under test. `time.Sleep` and `time.After` use the real clock, which the test scheduler does not control.

To await a task from the test itself, run the body inside `tc.BlockTest`, which gives you a `*scheduler.Scope` to await on:

```go
		tc.BlockTest(func(s *scheduler.Scope) {
			task := counter.UpdateR(cx, func(_ *Counter, cx *gpui.Context[Counter]) *gpui.Task[struct{}] {
				return cx.Spawn(func(this gpui.WeakEntity[Counter], cx *gpui.AsyncApp) (struct{}, error) {
					return struct{}{}, this.Update(cx, func(c *Counter, cx *gpui.Context[Counter]) {
						c.Add(10, cx)
					})
				})
			})
			if _, err := task.Await(s); err != nil {
				t.Fatal(err)
			}
		})
```

## Randomized tests and seeds

The scheduler's ordering and `tc.Rand()` come from a seed. Run a test many times with different seeds to shake out ordering bugs:

```go
		want := 0
		for range 50 {
			delta := rng.IntN(3) - 1
			want += delta
			counter.Update(cx, func(c *Counter, cx *gpui.Context[Counter]) {
				c.Add(delta, cx)
			})
		}

		got := counter.ReadWith(cx, func(c *Counter, _ *gpui.App) int { return c.count })
		if got != want {
			t.Fatalf("seed %d: count = %d, want %d", tc.Seed(), got, want)
		}
	}, gpui.Iterations(20))
```

When a run fails, the output names the failing seed. Re-run just that seed:

```sh
SEED=7 ITERATIONS=1 go test -run TestCounter_RandomOperations ./...
```

With `SEED` set, the test runs seeds `SEED` to `SEED+ITERATIONS-1` and ignores the seeds given in code.

Options: `gpui.Iterations(n)`, `gpui.Seed(s)`, `gpui.Seeds(...)`, `gpui.Retries(n)`. The environment variables `SEED` and `ITERATIONS` override them. `gpui.PropertyTest` runs a property test over random inputs (seeded, no shrinking).

## Leak detection

Inside a test binary, gpui-go reports entity handles, focus handles and subscriptions that were garbage-collected without `Release`, `Unsubscribe` or `Detach`. The report fails the test at teardown:

```
panic: Exited with leaked handles:
	Handle for entity lists.Table (EntityId(1v1)) was collected without Release: (export LEAK_BACKTRACE to find allocation site)
```

| `GPUI_LEAK_DETECTION` | In tests | Outside tests |
|---|---|---|
| unset | strict | off |
| `strict` | strict | strict |
| any other non-empty value, including `0` | on (not strict) | on |

- Set `LEAK_BACKTRACE=1` to record where each leaked handle was created.
- Leak reports are process-wide. Do not use `t.Parallel()` in tests that rely on them.
- Tasks are not tracked. A task you forget to cancel keeps running and is not reported.

## Rules for tests

- `AddWindowView` and `AddWindow` open windows on the test platform, so tests need no display.
- Do not touch the real clipboard, notifications or other user services from tests. The test platform fakes them: `cx.WriteToClipboard`, `cx.SimulatePromptAnswer`, `cx.ShownSystemNotifications`.
- Profiler code only runs with `-tags gpui_profiler`.

## Other test contexts

| Type | Use |
|---|---|
| `*gpui.TestAppContext` | from `tc.Cx()`; entities, test windows, simulated platform |
| `*gpui.VisualTestContext` | one test window; input, `DebugBounds`, `Draw` |
| `*gpui.VisualTestAppContext` | a real platform with offscreen windows and `CaptureScreenshot` |
| `*gpui.HeadlessAppContext` | headless rendering to an image |

See [examples/testing](../../examples/testing/main_test.go) for more patterns, including two apps talking over a simulated network.
