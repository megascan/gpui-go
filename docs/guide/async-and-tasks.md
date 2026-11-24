# Async work and tasks

GPUI runs all UI code on one thread, the main thread. Long work runs in *tasks*: foreground tasks on the main thread that can wait without blocking it, and background tasks on a thread pool. The code in this guide is in [snippets/async](snippets/async/async.go) and tested in [async_test.go](snippets/async/async_test.go).

## The rules

- App state (entities, windows, globals) is only touched on the main thread, inside an update.
- A foreground task may wait only on GPUI primitives: `task.Await(cx)`, timers, `scheduler.Oneshot` receivers, `scheduler.Select`. A plain `time.Sleep`, a raw `<-ch` or a mutex wait inside a foreground task freezes the UI.
- A task runs until it finishes or you cancel it. Store the `*gpui.Task` and cancel it when you no longer need it, or call `Detach()` to let it run on its own. Garbage collection never stops a task.

## Run a loop on the main thread

`cx.Spawn` on a `*gpui.Context[T]` starts a foreground task. The callback gets a weak handle to the entity and an `*gpui.AsyncApp`:

```go
type Clock struct {
	seconds int
	ticker  *gpui.Task[struct{}]
}

// NewClock creates a clock and starts its ticker.
func NewClock(cx *gpui.Context[Clock]) *Clock {
	return &Clock{ticker: startTicker(cx)}
}

func startTicker(cx *gpui.Context[Clock]) *gpui.Task[struct{}] {
	return cx.Spawn(func(this gpui.WeakEntity[Clock], cx *gpui.AsyncApp) (struct{}, error) {
		for {
			if _, err := cx.BackgroundExecutor().Timer(time.Second).Await(cx); err != nil {
				return struct{}{}, fmt.Errorf("tick timer: %w", err)
			}
			err := this.Update(cx, func(clock *Clock, cx *gpui.Context[Clock]) {
				clock.seconds++
				cx.Notify()
			})
			if err != nil {
				return struct{}{}, nil
			}
		}
	})
}
```

- The task body is a normal Go function. `Await` parks it until the timer fires and lets the main thread do other work in the meantime.
- `this` is a `gpui.WeakEntity`. It does not keep the clock alive. `this.Update` returns an error once the clock has been released, which is the signal to stop the loop.
- Every task returns `(R, error)`. Use `struct{}` when there is no value.

The clock stores its task and cancels it when it goes away:

```go
func (c *Clock) Reset(cx *gpui.Context[Clock]) {
	c.ticker.Cancel()
	c.seconds = 0
	c.ticker = startTicker(cx)
	cx.Notify()
}

// Drop stops the ticker.
func (c *Clock) Drop(_ *gpui.App) {
	c.ticker.Cancel()
}
```

GPUI calls `Drop(cx *gpui.App)` on entity state when the last handle to the entity is released. That is where you cancel tasks, unsubscribe and release handles. Cancel the old task before you replace it, or both keep running.

## Do heavy work in the background

Use `BackgroundSpawn` (or `cx.BackgroundExecutor().Spawn`) for CPU-heavy or blocking work, then come back to the main thread to store the result:

```go
func (r *Report) Analyze(text string, cx *gpui.Context[Report]) {
	r.status = "working"
	cx.Notify()
	r.job.Cancel()
	r.job = cx.Spawn(func(this gpui.WeakEntity[Report], cx *gpui.AsyncApp) (struct{}, error) {
		words, err := cx.BackgroundSpawn(func(_ *scheduler.Scope) (int, error) {
			return countWords(text), nil
		}).Await(cx)
		if err != nil {
			return struct{}{}, err
		}
		_ = this.Update(cx, func(report *Report, cx *gpui.Context[Report]) {
			report.status = fmt.Sprintf("%d words", words)
			cx.Notify()
		})
		return struct{}{}, nil
	})
}
```

The background function runs on another goroutine. It must not touch entities, windows or the `App`. Pass in the data it needs (here `text`) and return the result.

`r.job` starts out as a finished task so that the first `Cancel` is safe. Calling `Cancel` on a nil `*gpui.Task` panics:

```go
func NewReport(_ *gpui.Context[Report]) *Report {
	return &Report{status: "idle", job: scheduler.ReadyTask(struct{}{})}
}
```

Background code can block freely, with `time.Sleep`, file I/O or network calls. Only foreground tasks have the "GPUI primitives only" rule.

## Work with a window

`cx.SpawnIn(window, ...)` gives the task an `*gpui.AsyncWindowContext`, which can reach the window:

```go
func (t *Title) StartCountdown(window *gpui.Window, cx *gpui.Context[Title]) {
	cx.SpawnIn(window, func(_ gpui.WeakEntity[Title], cx *gpui.AsyncWindowContext) (struct{}, error) {
		for n := 3; n > 0; n-- {
			err := cx.Update(func(window *gpui.Window, _ *gpui.App) {
				window.SetWindowTitle(fmt.Sprintf("Closing in %d", n))
			})
			if err != nil {
				return struct{}{}, nil
			}
			if _, err := cx.BackgroundExecutor().Timer(time.Second).Await(cx); err != nil {
				return struct{}{}, err
			}
		}
		return struct{}{}, nil
	}).Detach()
}
```

`AsyncWindowContext.Update` returns an error if the window has closed.

## Which calls can fail

| Call | Result when the target is gone |
|---|---|
| `weak.Update(cx, ...)`, `weak.UpdateR`, `weak.ReadWith` | returns an error ("entity released") |
| `AsyncWindowContext.Update`, `UpdateR` | returns an error (window closed or app quitting) |
| `AsyncApp.Update(func(cx *gpui.App))`, `UpdateR` | no error; panics if the app is gone, which cannot happen inside a task started with `Spawn` |
| `entity.Update(cx, ...)` on a strong `Entity` | no error; the strong handle keeps the entity alive |

In a loop, check the error from the weak handle to know when to stop.

## Hand values between goroutines

Use `scheduler.Oneshot` (or `scheduler.Chan`) to send a value from any goroutine to a foreground task. The receiver's `Recv(cx)` parks the task instead of blocking the thread:

```go
func waitForAnswer(report gpui.WeakEntity[Report], cx *gpui.App) {
	tx, rx := scheduler.Oneshot[string]()
	cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
		answer, err := rx.Recv(cx)
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, report.Update(cx, func(report *Report, cx *gpui.Context[Report]) {
			report.status = answer
			cx.Notify()
		})
	}).Detach()
	cx.BackgroundExecutor().Spawn(func(_ *scheduler.Scope) (struct{}, error) {
		return struct{}{}, tx.Send("42")
	}).Detach()
}
```

To wait on several things at once, use `scheduler.Select(cx, cases...)` or `scheduler.Join(cx, futures...)`.

## Fire and forget, with logging

`Detach` drops the task's result, including its error. To log the error instead, use `gpui.TaskDetachAndLogErr`:

```go
func saveSettings(cx *gpui.App) {
	task := cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
		return struct{}{}, errOffline
	})
	gpui.TaskDetachAndLogErr(task, cx)
}
```

## Async contexts outside tasks

`cx.ToAsync()` returns an `*gpui.AsyncApp` you can keep across awaits. Most code never needs it, since `Spawn` hands you one.

## Recovering panics in tasks

Cancelling a task unwinds it with a special panic value. If a task body has a deferred `recover()`, re-panic values for which `scheduler.IsCancelUnwind(r)` is true, or cancellation stops working.

## Test async code

The test scheduler is deterministic and has a fake clock. Advance it and run pending work:

```go
		cx.Executor().AdvanceClock(3 * time.Second)
		cx.RunUntilParked()
```

See [Testing](testing.md#time-and-tasks).
