package testingguide

import (
	"testing"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

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

func TestCounter_EmitsChanged(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{} })
		defer counter.Release()

		var events []int
		subscription := cx.UpdateR(func(cx *gpui.App) gpui.Subscription {
			return cx.Subscribe(counter, func(_ gpui.Entity[Counter], event *Changed, _ *gpui.App) {
				events = append(events, event.Count)
			})
		})
		defer subscription.Unsubscribe()

		counter.Update(cx, func(c *Counter, cx *gpui.Context[Counter]) {
			c.Add(1, cx)
			c.Add(1, cx)
		})

		if len(events) != 2 || events[1] != 2 {
			t.Fatalf("events = %v, want [1 2]", events)
		}
	})
}

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

func TestCounter_RandomOperations(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		rng := tc.Rand()
		counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{} })
		defer counter.Release()

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
}

func TestCounter_AwaitTask(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{} })
		defer counter.Release()

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

		got := counter.ReadWith(cx, func(c *Counter, _ *gpui.App) int { return c.count })
		if got != 10 {
			t.Fatalf("count = %d, want 10", got)
		}
	})
}
