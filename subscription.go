package gpui

import (
	"iter"
	"runtime"
	"slices"

	"github.com/megascan/gpui-go/gpuiutil"
)

type singleSubscriberSet[C any] struct {
	state *singleSubscriberSetState[C]
}

type subscriberSet[K comparable, C any] struct {
	state *subscriberSetState[K, C]
}

type subscriberSetState[K comparable, C any] struct {
	subscribers      map[K]*subscriberList[C]
	nextSubscriberID int
}

type subscriberState uint8

const (
	subscriberStatePending subscriberState = iota
	subscriberStateActive
	subscriberStateDropped
)

type subscriber[C any] struct {
	state    *subscriberState
	callback C
}

type subscriberEntry[C any] struct {
	id         int
	subscriber *subscriber[C]
}

type subscriberList[C any] struct {
	entries []subscriberEntry[C]
}

func (l *subscriberList[C]) search(id int) (int, bool) {
	return slices.BinarySearchFunc(l.entries, id, func(e subscriberEntry[C], id int) int {
		return e.id - id
	})
}

func (l *subscriberList[C]) insert(id int, s *subscriber[C]) {
	i, found := l.search(id)
	if found {
		l.entries[i].subscriber = s
		return
	}
	l.entries = slices.Insert(l.entries, i, subscriberEntry[C]{id: id, subscriber: s})
}

func (l *subscriberList[C]) remove(id int) {
	if i, found := l.search(id); found {
		l.entries = slices.Delete(l.entries, i, i+1)
	}
}

func (l *subscriberList[C]) appendEntries(other []subscriberEntry[C]) {
	if len(other) == 0 {
		return
	}
	if len(l.entries) == 0 {
		l.entries = other
		return
	}
	merged := make([]subscriberEntry[C], 0, len(l.entries)+len(other))
	a, b := l.entries, other
	for len(a) > 0 && len(b) > 0 {
		switch {
		case a[0].id < b[0].id:
			merged = append(merged, a[0])
			a = a[1:]
		case a[0].id > b[0].id:
			merged = append(merged, b[0])
			b = b[1:]
		default:
			merged = append(merged, b[0])
			a, b = a[1:], b[1:]
		}
	}
	merged = append(merged, a...)
	merged = append(merged, b...)
	l.entries = merged
}

func retainSubscribers[C any](entries []subscriberEntry[C], f func(callback *C) bool) []subscriberEntry[C] {
	kept := entries[:0]
	for _, e := range entries {
		if e.subscriber.retain(f) {
			kept = append(kept, e)
		}
	}
	clear(entries[len(kept):])
	return kept
}

func newSubscriberSet[K comparable, C any]() subscriberSet[K, C] {
	return subscriberSet[K, C]{state: &subscriberSetState[K, C]{
		subscribers:      map[K]*subscriberList[C]{},
		nextSubscriberID: 0,
	}}
}

func (s subscriberSet[K, C]) insert(emitterKey K, callback C) Subscription {
	subscription, activate := s.insertPending(emitterKey, callback)
	activate()
	return subscription
}

func (s subscriberSet[K, C]) insertPending(emitterKey K, callback C) (Subscription, func()) {
	state := new(subscriberStatePending)
	lock := s.state
	subscriberID := gpuiutil.PostInc(&lock.nextSubscriberID)
	list, ok := lock.subscribers[emitterKey]
	if !ok {
		list = &subscriberList[C]{}
		lock.subscribers[emitterKey] = list
	}
	list.insert(subscriberID, &subscriber[C]{state: state, callback: callback})
	this := s.state

	subscription := NewSubscription(func() {
		*state = subscriberStateDropped
		subscribers, ok := this.subscribers[emitterKey]
		if !ok {
			return
		}
		subscribers.remove(subscriberID)
		if len(subscribers.entries) == 0 {
			delete(this.subscribers, emitterKey)
		}
	})
	return subscription, func() {
		if *state == subscriberStatePending {
			*state = subscriberStateActive
		}
	}
}

func (s subscriberSet[K, C]) remove(emitter K) iter.Seq[C] {
	subscribers, ok := s.state.subscribers[emitter]
	if ok {
		delete(s.state.subscribers, emitter)
	}
	return func(yield func(C) bool) {
		if !ok {
			return
		}
		for _, e := range subscribers.entries {
			if *e.subscriber.state == subscriberStateActive {
				if !yield(e.subscriber.callback) {
					return
				}
			}
		}
	}
}

func (s subscriberSet[K, C]) retain(emitter K, f func(callback *C) bool) {
	list, ok := s.state.subscribers[emitter]
	if !ok {
		return
	}
	subscribers := list.entries
	list.entries = nil

	subscribers = retainSubscribers(subscribers, f)
	lock := s.state

	if len(subscribers) == 0 {
		if current, ok := lock.subscribers[emitter]; ok && len(current.entries) == 0 {
			delete(lock.subscribers, emitter)
		}
		return
	}

	current, ok := lock.subscribers[emitter]
	if !ok {
		current = &subscriberList[C]{}
		lock.subscribers[emitter] = current
	}
	current.appendEntries(subscribers)
}

type singleSubscriberSetState[C any] struct {
	subscribers      subscriberList[C]
	nextSubscriberID int
}

func newSingleSubscriberSet[C any]() singleSubscriberSet[C] {
	return singleSubscriberSet[C]{state: &singleSubscriberSetState[C]{nextSubscriberID: 0}}
}

func (s singleSubscriberSet[C]) insert(callback C) Subscription {
	subscription, activate := s.insertPending(callback)
	activate()
	return subscription
}

func (s singleSubscriberSet[C]) insertPending(callback C) (Subscription, func()) {
	state := new(subscriberStatePending)
	lock := s.state
	subscriberID := gpuiutil.PostInc(&lock.nextSubscriberID)
	lock.subscribers.insert(subscriberID, &subscriber[C]{state: state, callback: callback})
	this := s.state

	subscription := NewSubscription(func() {
		*state = subscriberStateDropped
		this.subscribers.remove(subscriberID)
	})
	return subscription, func() {
		if *state == subscriberStatePending {
			*state = subscriberStateActive
		}
	}
}

func (s singleSubscriberSet[C]) remove() iter.Seq[C] {
	subscribers := s.state.subscribers.entries
	s.state.subscribers.entries = nil
	return func(yield func(C) bool) {
		for _, e := range subscribers {
			if *e.subscriber.state == subscriberStateActive {
				if !yield(e.subscriber.callback) {
					return
				}
			}
		}
	}
}

func (s singleSubscriberSet[C]) retain(f func(callback *C) bool) {
	subscribers := s.state.subscribers.entries
	s.state.subscribers.entries = nil
	subscribers = retainSubscribers(subscribers, f)
	s.state.subscribers.appendEntries(subscribers)
}

func (s *subscriber[C]) retain(f func(callback *C) bool) bool {
	switch *s.state {
	case subscriberStatePending:
		return true
	case subscriberStateDropped:
		return false
	default:
		keep := f(&s.callback)
		return keep && *s.state != subscriberStateDropped
	}
}

type subscriptionHandle struct {
	unsubscribe func()
	cleanup     runtime.Cleanup
}

// Subscription is a handle to a subscription created by GPUI. Unsubscribe cancels it, as
// dropping it does upstream; Detach keeps the callback registered until the entities it is
// subscribed to are released. Copying the value aliases one subscription; the zero value is
// an empty subscription.
type Subscription struct {
	h *subscriptionHandle
}

// NewSubscription creates a subscription whose Unsubscribe calls unsubscribe.
func NewSubscription(unsubscribe func()) Subscription {
	h := &subscriptionHandle{unsubscribe: unsubscribe}
	if unsubscribe != nil && currentLeakDetectionMode() == leakDetectionStrict {
		h.cleanup = runtime.AddCleanup(h, func(site *leakBacktraceFrames) {
			msg := "Subscription was collected without Unsubscribe or Detach"
			if site != nil {
				msg += ":\n" + formatLeakBacktrace(site.pcs)
			} else {
				msg += ": (export LEAK_BACKTRACE to find allocation site)"
			}
			recordStrictLeak(msg)
		}, captureLeakBacktrace(2))
	}
	return Subscription{h: h}
}

func (s Subscription) take() func() {
	if s.h == nil {
		return nil
	}
	unsubscribe := s.h.unsubscribe
	s.h.unsubscribe = nil
	s.h.cleanup.Stop()
	return unsubscribe
}

// Detach detaches the subscription from this handle. The callback will continue to be invoked
// until the entities it has been subscribed to are released.
func (s Subscription) Detach() {
	s.take()
}

// Unsubscribe cancels the subscription, so its callback is no longer invoked. It is idempotent.
func (s Subscription) Unsubscribe() {
	if unsubscribe := s.take(); unsubscribe != nil {
		unsubscribe()
	}
}

// SubscriptionJoin joins two subscriptions into a single subscription. Detach detaches both.
func SubscriptionJoin(subscriptionA, subscriptionB Subscription) Subscription {
	aUnsubscribe := subscriptionA.take()
	bUnsubscribe := subscriptionB.take()
	return NewSubscription(func() {
		if aUnsubscribe != nil {
			aUnsubscribe()
		}
		if bUnsubscribe != nil {
			bUnsubscribe()
		}
	})
}
