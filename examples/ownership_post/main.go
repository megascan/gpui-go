package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Counter counts.
type Counter struct {
	count int
}

// Change is emitted by a Counter.
type Change struct {
	increment int
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{count: 0} })
		subscriber := cx.New(func(cx *gpui.Context[Counter]) *Counter {
			cx.Subscribe(counter, func(subscriber *Counter, _ gpui.Entity[Counter], event *Change, _ *gpui.Context[Counter]) {
				subscriber.count += event.increment * 2
			}).Detach()

			return &Counter{
				count: counter.Read(cx.App).count * 2,
			}
		})

		counter.Update(cx, func(counter *Counter, cx *gpui.Context[Counter]) {
			counter.count += 2
			cx.Notify()
			cx.Emit(Change{increment: 2})
		})

		if got := subscriber.Read(cx).count; got != 4 {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", got, 4))
		}
		subscriber.Release()
		counter.Release()
	})
}

func main() {
	runExample()
}
