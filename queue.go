package gpui

import (
	"fmt"
	"iter"
	"math/bits"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/megascan/gpui-go/scheduler"
)

type priorityQueues[T any] struct {
	highPriority   []T
	mediumPriority []T
	lowPriority    []T
}

func (q *priorityQueues[T]) isEmpty() bool {
	return len(q.highPriority) == 0 && len(q.mediumPriority) == 0 && len(q.lowPriority) == 0
}

func popFront[T any](queue *[]T) T {
	q := *queue
	item := q[0]
	var zero T
	q[0] = zero
	*queue = q[1:]
	return item
}

type priorityQueueState[T any] struct {
	mu            sync.Mutex
	queues        priorityQueues[T]
	condvar       *sync.Cond
	receiverCount atomic.Int64
	senderCount   atomic.Int64
}

func (s *priorityQueueState[T]) send(priority Priority, item T) error {
	if s.receiverCount.Load() == 0 {
		return &SendError[T]{Item: item}
	}
	s.mu.Lock()
	s.push(priority, item)
	s.mu.Unlock()
	s.condvar.Signal()
	return nil
}

func (s *priorityQueueState[T]) spinLock() {
	for !s.mu.TryLock() {
		runtime.Gosched()
	}
}

func (s *priorityQueueState[T]) spinSend(priority Priority, item T) error {
	if s.receiverCount.Load() == 0 {
		return &SendError[T]{Item: item}
	}
	s.spinLock()
	s.push(priority, item)
	s.mu.Unlock()
	s.condvar.Signal()
	return nil
}

func (s *priorityQueueState[T]) push(priority Priority, item T) {
	switch priority {
	case scheduler.PriorityRealtimeAudio:
		panic("internal error: entered unreachable code: Realtime audio priority runs on a dedicated thread and is never queued")
	case scheduler.PriorityHigh:
		s.queues.highPriority = append(s.queues.highPriority, item)
	case scheduler.PriorityMedium:
		s.queues.mediumPriority = append(s.queues.mediumPriority, item)
	case scheduler.PriorityLow:
		s.queues.lowPriority = append(s.queues.lowPriority, item)
	}
}

func (s *priorityQueueState[T]) recv() error {
	s.mu.Lock()
	for s.queues.isEmpty() {
		if s.senderCount.Load() == 0 {
			s.mu.Unlock()
			return RecvError{}
		}
		s.condvar.Wait()
	}
	return nil
}

func (s *priorityQueueState[T]) tryRecvLocked() (bool, error) {
	senderCount := s.senderCount.Load()
	if s.queues.isEmpty() && senderCount == 0 {
		s.mu.Unlock()
		return false, RecvError{}
	}
	if s.queues.isEmpty() {
		s.mu.Unlock()
		return false, nil
	}
	return true, nil
}

func (s *priorityQueueState[T]) tryRecv() (bool, error) {
	s.mu.Lock()
	return s.tryRecvLocked()
}

func (s *priorityQueueState[T]) spinTryRecv() (bool, error) {
	s.spinLock()
	return s.tryRecvLocked()
}

// PriorityQueueSender enqueues items into a priority queue. Close is its drop.
type PriorityQueueSender[T any] struct {
	state  *priorityQueueState[T]
	closed atomic.Bool
}

func newPriorityQueueSender[T any](state *priorityQueueState[T]) *PriorityQueueSender[T] {
	return &PriorityQueueSender[T]{state: state}
}

// Send enqueues an item at the given priority, waiting for the queue lock if necessary. It
// returns a *SendError holding the item if all receivers were closed, and panics for
// PriorityRealtimeAudio while a receiver still exists.
func (s *PriorityQueueSender[T]) Send(priority Priority, item T) error {
	return s.state.send(priority, item)
}

// SpinSend is Send, spinning until the queue lock is acquired.
func (s *PriorityQueueSender[T]) SpinSend(priority Priority, item T) error {
	return s.state.spinSend(priority, item)
}

// Close drops the sender; once the last sender is closed, blocked receivers end. It is
// idempotent.
func (s *PriorityQueueSender[T]) Close() {
	if s.closed.Swap(true) {
		return
	}
	previous := s.state.senderCount.Add(-1) + 1
	if previous == 1 {
		s.state.mu.Lock()
		s.state.mu.Unlock()
		s.state.condvar.Broadcast()
	}
}

// PriorityQueueReceiver pops items from a priority queue, choosing between priorities by weighted
// random selection. Close is its drop.
type PriorityQueueReceiver[T any] struct {
	state        *priorityQueueState[T]
	rand         smallRng
	disconnected bool
	closed       bool
}

// Clone returns another receiver sharing the queue.
func (r *PriorityQueueReceiver[T]) Clone() *PriorityQueueReceiver[T] {
	r.state.receiverCount.Add(1)
	return &PriorityQueueReceiver[T]{
		state:        r.state,
		rand:         smallRngSeedFromU64(0),
		disconnected: r.disconnected,
	}
}

// SendError is returned by Send when all receivers were closed; it carries the unsent item.
type SendError[T any] struct {
	Item T
}

// Error formats the error as upstream's Debug does.
func (e *SendError[T]) Error() string {
	return fmt.Sprintf("SendError(%v)", e.Item)
}

// RecvError is returned by receive operations once the queue is empty and every sender closed.
type RecvError struct{}

// Error formats the error as upstream's Debug does.
func (RecvError) Error() string {
	return "RecvError"
}

// NewPriorityQueueReceiver creates an empty priority queue and returns its sender and receiver.
func NewPriorityQueueReceiver[T any]() (*PriorityQueueSender[T], *PriorityQueueReceiver[T]) {
	state := &priorityQueueState[T]{}
	state.condvar = sync.NewCond(&state.mu)
	state.receiverCount.Store(1)
	state.senderCount.Store(1)

	sender := newPriorityQueueSender(state)

	receiver := &PriorityQueueReceiver[T]{
		state:        state,
		rand:         smallRngSeedFromU64(0),
		disconnected: false,
	}

	return sender, receiver
}

// IsEmpty reports whether the queue currently contains no elements.
func (r *PriorityQueueReceiver[T]) IsEmpty() bool {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	return r.state.queues.isEmpty()
}

func (r *PriorityQueueReceiver[T]) len() int {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	q := &r.state.queues
	return len(q.highPriority) + len(q.mediumPriority) + len(q.lowPriority)
}

// TryPop pops one element without blocking; ok is false when the queue is empty. It returns
// RecvError once the queue is empty and every sender closed.
func (r *PriorityQueueReceiver[T]) TryPop() (item T, ok bool, err error) {
	return r.popInner(false)
}

// SpinTryPop is TryPop, spinning until the queue lock is acquired.
func (r *PriorityQueueReceiver[T]) SpinTryPop() (item T, ok bool, err error) {
	locked, err := r.state.spinTryRecv()
	if err != nil || !locked {
		return item, false, err
	}
	return r.choose()
}

// Pop pops an element, blocking while the queue is empty and a sender exists. It returns
// RecvError once the queue is empty and every sender closed.
func (r *PriorityQueueReceiver[T]) Pop() (T, error) {
	item, _, err := r.popInner(true)
	return item, err
}

// TryIter consumes the receiver and yields queued elements without waiting for new ones; it
// ends after yielding a RecvError. The receiver is closed when the iteration ends.
func (r *PriorityQueueReceiver[T]) TryIter() iter.Seq2[T, error] {
	it := &TryIter[T]{receiver: r}
	return it.all
}

// Iter consumes the receiver and yields elements, waiting for new ones while a sender exists.
// The receiver is closed when the iteration ends.
func (r *PriorityQueueReceiver[T]) Iter() iter.Seq[T] {
	it := &Iter[T]{receiver: r}
	return it.all
}

func (r *PriorityQueueReceiver[T]) popInner(block bool) (item T, ok bool, err error) {
	if !block {
		locked, err := r.state.tryRecv()
		if err != nil || !locked {
			return item, false, err
		}
	} else if err := r.state.recv(); err != nil {
		return item, false, err
	}
	return r.choose()
}

func (r *PriorityQueueReceiver[T]) choose() (item T, ok bool, err error) {
	q := &r.state.queues
	defer r.state.mu.Unlock()

	high := scheduler.PriorityHigh.Weight() * b2u(len(q.highPriority) != 0)
	medium := scheduler.PriorityMedium.Weight() * b2u(len(q.mediumPriority) != 0)
	low := scheduler.PriorityLow.Weight() * b2u(len(q.lowPriority) != 0)
	mass := high + medium + low

	if len(q.highPriority) != 0 {
		if r.rand.randomRatio(scheduler.PriorityHigh.Weight(), mass) {
			return popFront(&q.highPriority), true, nil
		}
		mass -= scheduler.PriorityHigh.Weight()
	}

	if len(q.mediumPriority) != 0 {
		if r.rand.randomRatio(scheduler.PriorityMedium.Weight(), mass) {
			return popFront(&q.mediumPriority), true, nil
		}
		mass -= scheduler.PriorityMedium.Weight()
	}

	if len(q.lowPriority) != 0 {
		if r.rand.randomRatio(scheduler.PriorityLow.Weight(), mass) {
			return popFront(&q.lowPriority), true, nil
		}
	}

	return item, false, nil
}

// Close drops the receiver. It is idempotent.
func (r *PriorityQueueReceiver[T]) Close() {
	if r.closed {
		return
	}
	r.closed = true
	r.state.receiverCount.Add(-1)
}

func b2u(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// Iter is the blocking iterator over a consumed receiver.
type Iter[T any] struct {
	receiver *PriorityQueueReceiver[T]
}

// Next pops the next element, blocking while the queue is empty; ok is false once it ended.
func (it *Iter[T]) Next() (T, bool) {
	item, err := it.receiver.Pop()
	return item, err == nil
}

func (it *Iter[T]) all(yield func(T) bool) {
	defer it.receiver.Close()
	for {
		item, ok := it.Next()
		if !ok || !yield(item) {
			return
		}
	}
}

// TryIter is the non-blocking iterator over a consumed receiver.
type TryIter[T any] struct {
	receiver *PriorityQueueReceiver[T]
	ended    bool
}

// Next pops the next element without waiting; ok is false when the iterator has no item.
func (it *TryIter[T]) Next() (item T, err error, ok bool) {
	if it.ended {
		return item, nil, false
	}
	item, some, err := it.receiver.TryPop()
	it.ended = err != nil
	if err != nil {
		return item, err, true
	}
	return item, nil, some
}

func (it *TryIter[T]) all(yield func(T, error) bool) {
	defer it.receiver.Close()
	for {
		item, err, ok := it.Next()
		if !ok || !yield(item, err) {
			return
		}
	}
}

type smallRng struct {
	s [4]uint64
}

func smallRngSeedFromU64(state uint64) smallRng {
	const phi = 0x9e3779b97f4a7c15
	var rng smallRng
	for i := range rng.s {
		state += phi
		z := state
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		z = z ^ (z >> 31)
		rng.s[i] = z
	}
	return rng
}

func (r *smallRng) nextU64() uint64 {
	res := bits.RotateLeft64(r.s[0]+r.s[3], 23) + r.s[0]
	t := r.s[1] << 17
	r.s[2] ^= r.s[0]
	r.s[3] ^= r.s[1]
	r.s[1] ^= r.s[2]
	r.s[0] ^= r.s[3]
	r.s[2] ^= t
	r.s[3] = bits.RotateLeft64(r.s[3], 45)
	return res
}

func (r *smallRng) randomRatio(numerator, denominator uint32) bool {
	if numerator > denominator || denominator == 0 {
		panic(fmt.Sprintf("p=%d/%d is outside range [0.0, 1.0]", numerator, denominator))
	}
	if numerator == denominator {
		return true
	}
	const scale = 2.0 * float64(uint64(1)<<63)
	pInt := uint64((float64(numerator) / float64(denominator)) * scale)
	return r.nextU64() < pInt
}
