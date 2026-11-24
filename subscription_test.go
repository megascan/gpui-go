package gpui

import (
	"slices"
	"strings"
	"testing"
)

func TestSubscription_ImmediateAndPendingInsertions(t *testing.T) {
	subscribers := newSubscriberSet[int, int]()
	first := subscribers.insert(1, 10)
	second, activateSecond := subscribers.insertPending(1, 20)
	var visited []int
	subscribers.retain(1, func(callback *int) bool {
		visited = append(visited, *callback)
		subscribers.insert(1, 30).Detach()
		return true
	})
	if !slices.Equal(visited, []int{10}) {
		t.Fatalf("visited = %v, want [10]", visited)
	}
	activateSecond()
	if got := slices.Collect(subscribers.remove(1)); !slices.Equal(got, []int{10, 20, 30}) {
		t.Fatalf("remove = %v, want [10 20 30]", got)
	}
	first.Unsubscribe()
	second.Unsubscribe()

	pending, _ := subscribers.insertPending(1, 40)
	if got := slices.Collect(subscribers.remove(1)); len(got) != 0 {
		t.Fatalf("remove = %v, want none", got)
	}
	pending.Unsubscribe()
}

func TestSubscription_SingleImmediateAndPendingInsertions(t *testing.T) {
	subscribers := newSingleSubscriberSet[int]()
	first := subscribers.insert(10)
	second, activateSecond := subscribers.insertPending(20)
	var visited []int
	subscribers.retain(func(callback *int) bool {
		visited = append(visited, *callback)
		subscribers.insert(30).Detach()
		return true
	})
	if !slices.Equal(visited, []int{10}) {
		t.Fatalf("visited = %v, want [10]", visited)
	}
	activateSecond()
	if got := slices.Collect(subscribers.remove()); !slices.Equal(got, []int{10, 20, 30}) {
		t.Fatalf("remove = %v, want [10 20 30]", got)
	}
	first.Unsubscribe()
	second.Unsubscribe()

	pending, _ := subscribers.insertPending(40)
	if got := slices.Collect(subscribers.remove()); len(got) != 0 {
		t.Fatalf("remove = %v, want none", got)
	}
	pending.Unsubscribe()
}

type subscriptionTestPending struct {
	subscription Subscription
	activate     func()
}

func TestSubscription_ActivationAfterDropDuringRetain(t *testing.T) {
	subscribers := newSubscriberSet[int, int]()
	first, activateFirst := subscribers.insertPending(1, 10)
	second, activateSecond := subscribers.insertPending(1, 20)
	subscribers.retain(1, func(*int) bool { panic("pending subscriber invoked") })
	activateFirst()
	pending := &subscriptionTestPending{subscription: second, activate: activateSecond}
	var visited []int

	subscribers.retain(1, func(callback *int) bool {
		visited = append(visited, *callback)
		if pending == nil {
			panic("only the first callback runs")
		}
		p := pending
		pending = nil
		p.subscription.Unsubscribe()
		p.activate()
		return true
	})

	if !slices.Equal(visited, []int{10}) {
		t.Fatalf("visited = %v, want [10]", visited)
	}
	if got := slices.Collect(subscribers.remove(1)); !slices.Equal(got, []int{10}) {
		t.Fatalf("remove = %v, want [10]", got)
	}
	first.Unsubscribe()
}

func TestSubscription_SingleActivationAfterDropDuringRetain(t *testing.T) {
	subscribers := newSingleSubscriberSet[int]()
	first, activateFirst := subscribers.insertPending(10)
	second, activateSecond := subscribers.insertPending(20)
	subscribers.retain(func(*int) bool { panic("pending subscriber invoked") })
	activateFirst()
	pending := &subscriptionTestPending{subscription: second, activate: activateSecond}
	var visited []int

	subscribers.retain(func(callback *int) bool {
		visited = append(visited, *callback)
		if pending == nil {
			panic("only the first callback runs")
		}
		p := pending
		pending = nil
		p.subscription.Unsubscribe()
		p.activate()
		return true
	})

	if !slices.Equal(visited, []int{10}) {
		t.Fatalf("visited = %v, want [10]", visited)
	}
	if got := slices.Collect(subscribers.remove()); !slices.Equal(got, []int{10}) {
		t.Fatalf("remove = %v, want [10]", got)
	}
	first.Unsubscribe()
}

type subscriptionTestGlobal struct{}

func takeSubscription(slot **Subscription) {
	if sub := *slot; sub != nil {
		*slot = nil
		sub.Unsubscribe()
	}
}

func TestSubscription_UnsubscribeDuringCallbackWithInsert(t *testing.T) {
	app := NewTestApp()
	defer app.Close()
	app.SetGlobal(subscriptionTestGlobal{})

	observerACount := 0
	observerBCount := 0

	var subA, subB *Subscription

	a := app.UpdateR(func(cx *App) Subscription {
		return cx.ObserveGlobal[subscriptionTestGlobal](func(cx *App) {
			observerACount++
			takeSubscription(&subA)
			cx.ObserveGlobal[subscriptionTestGlobal](func(*App) {}).Detach()
		})
	})
	subA = &a

	b := app.UpdateR(func(cx *App) Subscription {
		return cx.ObserveGlobal[subscriptionTestGlobal](func(*App) {
			observerBCount++
			takeSubscription(&subB)
		})
	})
	subB = &b

	app.Update(func(cx *App) { cx.SetGlobal(subscriptionTestGlobal{}) })
	if observerACount != 1 || observerBCount != 1 {
		t.Fatalf("counts = %d, %d, want 1, 1", observerACount, observerBCount)
	}

	app.Update(func(cx *App) { cx.SetGlobal(subscriptionTestGlobal{}) })
	if observerACount != 1 {
		t.Fatalf("observer A count = %d, want 1", observerACount)
	}
	if observerBCount != 1 {
		t.Fatalf("orphaned subscriber fired again: %d", observerBCount)
	}
}

func TestSubscription_CallbackDroppedByEarlierCallbackDoesNotFire(t *testing.T) {
	app := NewTestApp()
	defer app.Close()
	app.SetGlobal(subscriptionTestGlobal{})

	observerBCount := 0
	var subB *Subscription

	app.Update(func(cx *App) {
		cx.ObserveGlobal[subscriptionTestGlobal](func(*App) {
			takeSubscription(&subB)
		}).Detach()
	})

	b := app.UpdateR(func(cx *App) Subscription {
		return cx.ObserveGlobal[subscriptionTestGlobal](func(*App) {
			observerBCount++
		})
	})
	subB = &b

	app.Update(func(cx *App) { cx.SetGlobal(subscriptionTestGlobal{}) })
	if observerBCount != 0 {
		t.Fatalf("B should not fire — A dropped its subscription: %d", observerBCount)
	}
}

func TestSubscription_SelfDropDuringCallback(t *testing.T) {
	app := NewTestApp()
	defer app.Close()
	app.SetGlobal(subscriptionTestGlobal{})

	count := 0
	var sub *Subscription

	s := app.UpdateR(func(cx *App) Subscription {
		return cx.ObserveGlobal[subscriptionTestGlobal](func(*App) {
			count++
			takeSubscription(&sub)
		})
	})
	sub = &s

	app.Update(func(cx *App) { cx.SetGlobal(subscriptionTestGlobal{}) })
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	app.Update(func(cx *App) { cx.SetGlobal(subscriptionTestGlobal{}) })
	if count != 1 {
		t.Fatalf("should not fire after self-drop: %d", count)
	}
}

func TestSubscription_SubscriptionDrop(t *testing.T) {
	app := NewTestApp()
	defer app.Close()
	app.SetGlobal(subscriptionTestGlobal{})

	count := 0

	subscription := app.UpdateR(func(cx *App) Subscription {
		return cx.ObserveGlobal[subscriptionTestGlobal](func(*App) {
			count++
		})
	})

	subscription.Unsubscribe()

	app.Update(func(cx *App) { cx.SetGlobal(subscriptionTestGlobal{}) })
	if count != 0 {
		t.Fatalf("should not fire after drop: %d", count)
	}
}

func TestSubscription_GoUnsubscribeRemovesSubscriber(t *testing.T) {
	subscribers := newSubscriberSet[string, int]()
	a := subscribers.insert("a", 1)
	b := subscribers.insert("a", 2)
	alias := a
	alias.Unsubscribe()
	a.Unsubscribe()
	var visited []int
	subscribers.retain("a", func(callback *int) bool {
		visited = append(visited, *callback)
		return true
	})
	if !slices.Equal(visited, []int{2}) {
		t.Fatalf("visited = %v, want [2]", visited)
	}
	b.Unsubscribe()
	if _, ok := subscribers.state.subscribers["a"]; ok {
		t.Fatal("the emitter entry should be removed with its last subscriber")
	}

	detached := subscribers.insert("b", 3)
	detached.Detach()
	detached.Unsubscribe()
	if got := slices.Collect(subscribers.remove("b")); !slices.Equal(got, []int{3}) {
		t.Fatalf("a detached subscriber was removed: %v", got)
	}

	var zero Subscription
	zero.Unsubscribe()
	zero.Detach()
}

func TestSubscription_GoRetainDropsFalseAndSelfUnsubscribed(t *testing.T) {
	subscribers := newSubscriberSet[int, int]()
	var self Subscription
	self = subscribers.insert(1, 1)
	keep := subscribers.insert(1, 2)
	drop := subscribers.insert(1, 3)
	subscribers.retain(1, func(callback *int) bool {
		switch *callback {
		case 1:
			self.Unsubscribe()
			return true
		case 3:
			return false
		}
		return true
	})
	if got := slices.Collect(subscribers.remove(1)); !slices.Equal(got, []int{2}) {
		t.Fatalf("remove = %v, want [2]", got)
	}
	keep.Unsubscribe()
	drop.Unsubscribe()
}

func TestSubscription_GoRemoveFiltersLazily(t *testing.T) {
	subscribers := newSubscriberSet[int, int]()
	a := subscribers.insert(1, 1)
	b := subscribers.insert(1, 2)
	removed := subscribers.remove(1)
	b.Unsubscribe()
	if got := slices.Collect(removed); !slices.Equal(got, []int{1}) {
		t.Fatalf("remove = %v, want [1]", got)
	}
	a.Unsubscribe()
}

func TestSubscription_GoJoin(t *testing.T) {
	var calls []string
	joined := SubscriptionJoin(
		NewSubscription(func() { calls = append(calls, "a") }),
		NewSubscription(func() { calls = append(calls, "b") }),
	)
	joined.Unsubscribe()
	joined.Unsubscribe()
	if !slices.Equal(calls, []string{"a", "b"}) {
		t.Fatalf("calls = %v, want [a b]", calls)
	}

	calls = nil
	a := NewSubscription(func() { calls = append(calls, "a") })
	SubscriptionJoin(a, NewSubscription(func() { calls = append(calls, "b") })).Detach()
	a.Unsubscribe()
	if len(calls) != 0 {
		t.Fatalf("detaching a join ran %v", calls)
	}
}

//go:noinline
func subscriptionTestAbandon(subscribers subscriberSet[int, int]) {
	_ = subscribers.insert(1, 1)
}

func TestSubscription_GoCollectedWithoutUnsubscribeIsReported(t *testing.T) {
	if currentLeakDetectionMode() != leakDetectionStrict {
		t.Skip("needs GPUI_LEAK_DETECTION=strict")
	}
	subscribers := newSubscriberSet[int, int]()
	takeStrictLeakReports()
	subscriptionTestAbandon(subscribers)
	var reports []string
	if !entityMapWaitFor(func() bool {
		reports = append(reports, takeStrictLeakReports()...)
		return len(reports) > 0
	}) {
		t.Fatal("the abandoned subscription was never reported")
	}
	if !strings.Contains(reports[0], "Subscription was collected without Unsubscribe or Detach") {
		t.Fatalf("report = %q", reports[0])
	}
	if got := slices.Collect(subscribers.remove(1)); !slices.Equal(got, []int{1}) {
		t.Fatalf("a collected subscription must stay subscribed: %v", got)
	}
}
