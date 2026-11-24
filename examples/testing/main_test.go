package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

func TestTesting_BasicTesting(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		counter := cx.New(NewCounter)
		defer counter.Release()

		counter.Update(cx, func(counter *Counter, _ *gpui.Context[Counter]) {
			counter.count = 42
		})

		updated := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
		if updated != 42 {
			t.Fatalf("count = %d, want 42", updated)
		}

		counter.Update(cx, func(_ *Counter, cx *gpui.Context[Counter]) {
			cx.Emit(CounterEvent{})
		})

		countAfterUpdate := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
		if countAfterUpdate != 999 {
			t.Fatalf("Side effects should run after update completes: count = %d", countAfterUpdate)
		}
	})
}

func openCounterWindow(cx *gpui.TestAppContext) gpui.WindowHandle[Counter] {
	return cx.UpdateR(func(cx *gpui.App) gpui.WindowHandle[Counter] {
		window, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Counter] {
			return cx.New(NewCounter)
		})
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		return window
	})
}

func TestTesting_CounterInWindow(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		window := openCounterWindow(tc.Cx())

		cx := gpui.VisualTestContextFromWindow(window.AnyWindowHandle, tc.Cx())
		counter, err := window.Root(cx)
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		defer counter.Release()

		focusHandle := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) gpui.FocusHandle { return counter.focusHandle.Clone() })
		defer focusHandle.Release()
		cx.Update(func(window *gpui.Window, cx *gpui.App) {
			focusHandle.DispatchAction(Increment{}, window, cx)
		})

		countAfter := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
		if countAfter != 1 {
			t.Fatalf("Action dispatched via focus handle should increment: count = %d", countAfter)
		}
	})
}

func TestTesting_AsyncOperations(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		tc.BlockTest(func(s *scheduler.Scope) {
			cx := tc.Cx()
			counter := cx.New(NewCounter)
			defer counter.Release()

			if _, err := counter.UpdateR(cx, func(counter *Counter, cx *gpui.Context[Counter]) *gpui.Task[struct{}] { return counter.load(cx) }).Await(s); err != nil {
				panic(err)
			}

			count := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
			if count != 100 {
				t.Fatalf("Load task should have set count to 100: count = %d", count)
			}

			counter.Update(cx, func(counter *Counter, cx *gpui.Context[Counter]) { counter.reload(cx) })

			count = counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
			if count != 100 {
				t.Fatalf("Detached reload task shouldn't have run yet: count = %d", count)
			}

			cx.RunUntilParked()

			count = counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
			if count != 150 {
				t.Fatalf("Reload task should have run after parking: count = %d", count)
			}
		})
	})
}

func TestTesting_AllowParking(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			cx.Executor().AllowParking()

			tx, rx := scheduler.Oneshot[int]()
			go func() {
				time.Sleep(5 * time.Millisecond)
				_ = tx.Send(42)
			}()

			result, err := rx.Recv(s)
			if err != nil {
				panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
			}
			if result != 42 {
				t.Fatalf("result = %d, want 42", result)
			}
		})
	})
}

func TestTesting_CounterRandomOperations(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		rng := tc.Rand()
		window := openCounterWindow(tc.Cx())
		cx := gpui.VisualTestContextFromWindow(window.AnyWindowHandle, tc.Cx())

		counter := cx.New(NewCounter)
		defer counter.Release()

		var expected int32
		for range 100 {
			if rng.Float64() < 0.5 {
				expected++
				_ = counter.UpdateIn(cx, func(counter *Counter, window *gpui.Window, cx *gpui.Context[Counter]) {
					counter.increment(&Increment{}, window, cx)
				})
			} else {
				expected--
				_ = counter.UpdateIn(cx, func(counter *Counter, window *gpui.Window, cx *gpui.Context[Counter]) {
					counter.decrement(&Decrement{}, window, cx)
				})
			}
		}

		actual := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
		if actual != expected {
			t.Fatalf("Counter should match expected after random ops: %d != %d", actual, expected)
		}
	}, gpui.Iterations(10))
}

type mockNetworkState struct {
	ordering []int32
	aToB     []int32
	bToA     []int32
}

type mockNetwork struct {
	mu    *sync.Mutex
	state *mockNetworkState
}

func newMockNetwork() mockNetwork {
	return mockNetwork{mu: &sync.Mutex{}, state: &mockNetworkState{}}
}

func (n mockNetwork) aClient() networkClient {
	return networkClient{network: n, isA: true}
}

func (n mockNetwork) bClient() networkClient {
	return networkClient{network: n, isA: false}
}

type networkClient struct {
	network mockNetwork
	isA     bool
}

func (c networkClient) send(value int32) {
	c.network.mu.Lock()
	defer c.network.mu.Unlock()
	network := c.network.state
	network.ordering = append(network.ordering, value)
	if c.isA {
		network.bToA = append(network.bToA, value)
	} else {
		network.aToB = append(network.aToB, value)
	}
}

func (c networkClient) receiveAll() []int32 {
	c.network.mu.Lock()
	defer c.network.mu.Unlock()
	network := c.network.state
	var out []int32
	if c.isA {
		out, network.aToB = network.aToB, nil
	} else {
		out, network.bToA = network.bToA, nil
	}
	return out
}

type networkedCounter struct {
	count  int32
	client networkClient
}

func newNetworkedCounter(client networkClient) *networkedCounter {
	return &networkedCounter{count: 0, client: client}
}

func (n *networkedCounter) increment(delta int32, cx *gpui.Context[networkedCounter]) {
	n.count += delta

	client := n.client
	cx.BackgroundSpawn(func(_ *scheduler.Scope) (struct{}, error) {
		client.send(delta)
		return struct{}{}, nil
	}).Detach()
}

func (n *networkedCounter) sync() {
	for _, delta := range n.client.receiveAll() {
		n.count += delta
	}
}

func readCount(e gpui.Entity[networkedCounter], cx *gpui.TestAppContext) int32 {
	return e.ReadWith(cx, func(n *networkedCounter, _ *gpui.App) int32 { return n.count })
}

func TestTestingDistributedSystems_AppSync(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cxA := tc.Cx()
		cxB := tc.NewContext()
		network := newMockNetwork()

		a := cxA.New(func(*gpui.Context[networkedCounter]) *networkedCounter { return newNetworkedCounter(network.aClient()) })
		defer a.Release()
		b := cxB.New(func(*gpui.Context[networkedCounter]) *networkedCounter { return newNetworkedCounter(network.bClient()) })
		defer b.Release()

		b.Update(cxB, func(b *networkedCounter, cx *gpui.Context[networkedCounter]) { b.increment(42, cx) })
		if got := readCount(b, cxB); got != 42 {
			t.Fatalf("b.count = %d, want 42", got)
		}
		if got := readCount(a, cxA); got != 0 {
			t.Fatalf("a.count = %d, want 0", got)
		}

		cxB.RunUntilParked()
		a.Update(cxA, func(a *networkedCounter, _ *gpui.Context[networkedCounter]) { a.sync() })

		if got := readCount(b, cxB); got != 42 {
			t.Fatalf("b.count = %d, want 42", got)
		}
		if got := readCount(a, cxA); got != 42 {
			t.Fatalf("a.count = %d, want 42", got)
		}
	})
}

func TestTestingDistributedSystems_RandomInterleaving(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cxA := tc.Cx()
		cxB := tc.NewContext()
		rng := tc.Rand()
		network := newMockNetwork()

		var originalOrder []int32
		a := cxA.New(func(*gpui.Context[networkedCounter]) *networkedCounter { return newNetworkedCounter(network.aClient()) })
		defer a.Release()
		b := cxB.New(func(*gpui.Context[networkedCounter]) *networkedCounter { return newNetworkedCounter(network.bClient()) })
		defer b.Release()

		numOperations := 3 + rng.IntN(5)

		for i := range numOperations {
			i := int32(i)
			which := rng.Float64() < 0.5

			originalOrder = append(originalOrder, i)
			if which {
				b.Update(cxB, func(b *networkedCounter, cx *gpui.Context[networkedCounter]) { b.increment(i, cx) })
			} else {
				a.Update(cxA, func(a *networkedCounter, cx *gpui.Context[networkedCounter]) { a.increment(i, cx) })
			}
		}

		cxA.RunUntilParked()

		a.Update(cxA, func(a *networkedCounter, _ *gpui.Context[networkedCounter]) { a.sync() })
		b.Update(cxB, func(b *networkedCounter, _ *gpui.Context[networkedCounter]) { b.sync() })

		aCount := readCount(a, cxA)
		bCount := readCount(b, cxB)

		if aCount != bCount {
			t.Fatalf("A and B should have the same count: %d != %d", aCount, bCount)
		}

		network.mu.Lock()
		actual := append([]int32(nil), network.state.ordering...)
		network.mu.Unlock()
		spawned := make([]string, len(originalOrder))
		for i, n := range originalOrder {
			spawned[i] = fmt.Sprint(n)
		}
		ran := make([]string, len(actual))
		for i, n := range actual {
			ran[i] = fmt.Sprint(n)
		}
		var diff []string
		for i := range min(len(originalOrder), len(actual)) {
			if originalOrder[i] == actual[i] {
				diff = append(diff, " ")
			} else {
				diff = append(diff, "^")
			}
		}
		t.Logf("spawned: [%s]", strings.Join(spawned, ", "))
		t.Logf("ran:     [%s]", strings.Join(ran, ", "))
		t.Logf("         [%s]", strings.Join(diff, ", "))
	}, gpui.Iterations(10))
}
