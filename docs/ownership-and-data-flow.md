# Ownership and data flow

In GPUI, every model or view in the application is owned by a single top-level object, the `App`. When you create a model or a view (together called _entities_), you hand ownership of its state to the application, so the entity can use app services and interact with other entities.

Take the small app below. We start it by calling `Run` with a callback, which receives the `*gpui.App` that owns all application state. The `App` is the gateway to application-level services such as opening windows and showing dialogs. Its `New` method creates an entity and gives the application ownership of it.

```go
type Counter struct {
	count int
}

platform.Application().Run(func(cx *gpui.App) {
	counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{count: 0} })
	defer counter.Release()
	// ...
})
```

`New` returns an _entity handle_, an `Entity[Counter]`, typed by the state it refers to. The handle alone gives no access to the state. It is an inert identifier with a compile-time type, holding a counted reference to the `Counter` the app owns.

`Clone()` makes another counted handle and `Release()` drops one, so several owners can share the state. Unlike a plain pointer, the handle only gives access to the state while you have an `App`: it does not own the state, it lets you reach the state through its real owner, the `App`. Leaving out some setup:

```go
platform.Application().Run(func(cx *gpui.App) {
	counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{count: 0} })
	defer counter.Release()
	counter.Update(cx, func(counter *Counter, _ *gpui.Context[Counter]) {
		counter.count += 1
	})
})
```

To change the counter, call `Update` on the handle with the context and a callback. The callback gets a pointer to the counter's state, which it can modify.

The callback also gets a `*gpui.Context[Counter]`. It is like the `*App` passed to the `Run` callback: a `Context` wraps the `App` together with the entity it belongs to, here the counter.

Besides the application services of `App`, a `Context` offers entity services. For example, it can tell this entity's observers that its state changed. Add that to the example with `cx.Notify()`:

```go
platform.Application().Run(func(cx *gpui.App) {
	counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{count: 0} })
	defer counter.Release()
	counter.Update(cx, func(counter *Counter, cx *gpui.Context[Counter]) {
		counter.count += 1
		cx.Notify()
	})
})
```

Next, something has to observe these notifications. Before updating the counter, create a second counter that observes it: whenever the first counter changes, the second gets twice its count. `Observe` is called on the second counter's `Context`, so the second counter is notified whenever the first one calls `Notify`. `Observe` returns a `Subscription`, which we `Detach` to keep the observation for as long as both counters exist. We could instead keep the subscription and call `Unsubscribe` whenever we want the observation to stop.

The `Observe` callback receives the observer's state and a handle to the observed counter, whose state it reads with `Read`.

```go
platform.Application().Run(func(cx *gpui.App) {
	firstCounter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{count: 0} })
	defer firstCounter.Release()

	secondCounter := cx.New(func(cx *gpui.Context[Counter]) *Counter {
		cx.Observe(firstCounter, func(second *Counter, first gpui.Entity[Counter], cx *gpui.Context[Counter]) {
			second.count = first.Read(cx.App).count * 2
		}).Detach()

		return &Counter{count: 0}
	})
	defer secondCounter.Release()

	firstCounter.Update(cx, func(counter *Counter, cx *gpui.Context[Counter]) {
		counter.count += 1
		cx.Notify()
	})

	if secondCounter.Read(cx).count != 2 {
		panic("observer did not run")
	}
})
```

The observation was set up before the second counter existed: the callback is registered inside its constructor. After the first counter is updated, the second counter's state follows it through the subscription.

`Observe` and `Notify` say that an entity's state changed. GPUI also has `Subscribe` and `Emit`, which let entities emit typed events. Any Go type can be an event; subscribers name the event type they want.

Add an event type, `CounterChangeEvent`:

```go
type CounterChangeEvent struct {
	increment int
}
```

Now replace the observation with a subscription. Each time the counter is incremented, it emits a `CounterChangeEvent` carrying the size of the increase.

```go
platform.Application().Run(func(cx *gpui.App) {
	firstCounter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{count: 0} })
	defer firstCounter.Release()

	secondCounter := cx.New(func(cx *gpui.Context[Counter]) *Counter {
		cx.Subscribe(firstCounter, func(second *Counter, _ gpui.Entity[Counter], event *CounterChangeEvent, _ *gpui.Context[Counter]) {
			second.count += event.increment * 2
		}).Detach()

		return &Counter{count: firstCounter.Read(cx.App).count * 2}
	})
	defer secondCounter.Release()

	firstCounter.Update(cx, func(first *Counter, cx *gpui.Context[Counter]) {
		first.count += 2
		cx.Emit(CounterChangeEvent{increment: 2})
		cx.Notify()
	})

	if secondCounter.Read(cx).count != 4 {
		panic("subscriber did not run")
	}
})
```

`examples/ownership_post` runs this last program.
