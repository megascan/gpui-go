package scheduler

import "sync"

type chanState[T any] struct {
	mu             sync.Mutex
	buf            []T
	head           int
	capacity       int
	senders        int
	receiverClosed bool
	rxWaker        Waker
	txWaiters      []*parkedSender
}

type parkedSender struct {
	waker  Waker
	queued bool
}

// ChanSender is one counted sending handle of a channel. Clone makes another; Close is its drop,
// and the channel closes when the last sender is closed.
type ChanSender[T any] struct {
	s      *chanState[T]
	mu     sync.Mutex
	closed bool
}

// ChanReceiver is the receiving half of a channel. Close is its drop.
type ChanReceiver[T any] struct {
	s *chanState[T]
}

// Chan returns a multi-producer, single-consumer channel (futures::channel::mpsc). A capacity of
// zero or less makes it unbounded; otherwise Send parks while capacity values are queued.
func Chan[T any](capacity int) (*ChanSender[T], *ChanReceiver[T]) {
	s := &chanState[T]{capacity: capacity, senders: 1}
	return &ChanSender[T]{s: s}, &ChanReceiver[T]{s: s}
}

func (s *chanState[T]) lenLocked() int {
	return len(s.buf) - s.head
}

func (s *chanState[T]) popLocked() (T, bool) {
	var zero T
	if s.head == len(s.buf) {
		return zero, false
	}
	v := s.buf[s.head]
	s.buf[s.head] = zero
	s.head++
	if s.head == len(s.buf) {
		s.buf = s.buf[:0]
		s.head = 0
	}
	return v, true
}

func (s *chanState[T]) takeTxWakersLocked() []Waker {
	if len(s.txWaiters) == 0 {
		return nil
	}
	w := make([]Waker, 0, len(s.txWaiters))
	for _, p := range s.txWaiters {
		w = append(w, p.waker)
		p.waker = nil
		p.queued = false
	}
	s.txWaiters = nil
	return w
}

func (s *chanState[T]) unparkLocked(p *parkedSender) Waker {
	if p == nil || !p.queued {
		return nil
	}
	for i, q := range s.txWaiters {
		if q == p {
			s.txWaiters = append(s.txWaiters[:i], s.txWaiters[i+1:]...)
			break
		}
	}
	w := p.waker
	p.waker = nil
	p.queued = false
	return w
}

func wakeAll(ws []Waker) {
	for _, w := range ws {
		w.Wake()
	}
}

// Clone returns a new sending handle for the same channel.
func (tx *ChanSender[T]) Clone() *ChanSender[T] {
	tx.mu.Lock()
	closed := tx.closed
	tx.mu.Unlock()
	if closed {
		panic("scheduler: ChanSender used after Close")
	}
	tx.s.mu.Lock()
	tx.s.senders++
	tx.s.mu.Unlock()
	return &ChanSender[T]{s: tx.s}
}

// Close drops this sender. It is idempotent.
func (tx *ChanSender[T]) Close() {
	tx.mu.Lock()
	if tx.closed {
		tx.mu.Unlock()
		return
	}
	tx.closed = true
	tx.mu.Unlock()
	s := tx.s
	s.mu.Lock()
	s.senders--
	var w Waker
	if s.senders == 0 {
		w = s.rxWaker
		s.rxWaker = nil
	}
	s.mu.Unlock()
	wakeAwaiter(w)
}

// IsClosed reports whether the receiver was dropped.
func (tx *ChanSender[T]) IsClosed() bool {
	tx.s.mu.Lock()
	defer tx.s.mu.Unlock()
	return tx.s.receiverClosed
}

func (tx *ChanSender[T]) checkOpen() {
	tx.mu.Lock()
	closed := tx.closed
	tx.mu.Unlock()
	if closed {
		panic("scheduler: ChanSender used after Close")
	}
}

// TrySend queues v without parking. It returns ErrClosed if the receiver is gone and ErrFull if
// a bounded channel is full.
func (tx *ChanSender[T]) TrySend(v T) error {
	tx.checkOpen()
	return tx.trySend(v)
}

func (tx *ChanSender[T]) trySendIfOpen(v T) error {
	tx.mu.Lock()
	closed := tx.closed
	tx.mu.Unlock()
	if closed {
		return ErrClosed
	}
	return tx.trySend(v)
}

func (tx *ChanSender[T]) trySend(v T) error {
	s := tx.s
	s.mu.Lock()
	if s.receiverClosed {
		s.mu.Unlock()
		return ErrClosed
	}
	if s.capacity > 0 && s.lenLocked() >= s.capacity {
		s.mu.Unlock()
		return ErrFull
	}
	s.buf = append(s.buf, v)
	w := s.rxWaker
	s.rxWaker = nil
	s.mu.Unlock()
	wakeAwaiter(w)
	return nil
}

// Send queues v, parking the coroutine body while a bounded channel is full. It returns
// ErrClosed if the receiver is gone.
func (tx *ChanSender[T]) Send(cx Awaiter, v T) error {
	return Await[error](cx, &sendFuture[T]{tx: tx, v: v})
}

type sendFuture[T any] struct {
	tx     *ChanSender[T]
	v      T
	parked *parkedSender
}

func (f *sendFuture[T]) Poll(cx *Context) (error, bool) {
	for {
		err := f.tx.TrySend(f.v)
		if err != ErrFull {
			f.unpark()
			return err, true
		}
		clone := cx.Waker().Clone()
		s := f.tx.s
		s.mu.Lock()
		if !s.receiverClosed && s.lenLocked() >= s.capacity {
			if f.parked == nil {
				f.parked = &parkedSender{}
			}
			old := f.parked.waker
			f.parked.waker = clone
			if !f.parked.queued {
				f.parked.queued = true
				s.txWaiters = append(s.txWaiters, f.parked)
			}
			s.mu.Unlock()
			dropAwaiter(old)
			return nil, false
		}
		s.mu.Unlock()
		clone.Drop()
	}
}

func (f *sendFuture[T]) unpark() {
	if f.parked == nil {
		return
	}
	s := f.tx.s
	s.mu.Lock()
	w := s.unparkLocked(f.parked)
	s.mu.Unlock()
	dropAwaiter(w)
}

func (f *sendFuture[T]) Cancel() {
	f.unpark()
}

// Poll resolves to the next value, or to ErrClosed once every sender is gone and the queue is
// empty (StreamExt::next).
func (rx *ChanReceiver[T]) Poll(cx *Context) (Result[T], bool) {
	s := rx.s
	s.mu.Lock()
	if v, ok := s.popLocked(); ok {
		tw := s.takeTxWakersLocked()
		s.mu.Unlock()
		wakeAll(tw)
		return Result[T]{Value: v}, true
	}
	if s.senders == 0 {
		s.mu.Unlock()
		return Result[T]{Err: ErrClosed}, true
	}
	s.mu.Unlock()
	clone := cx.Waker().Clone()
	s.mu.Lock()
	if s.lenLocked() > 0 || s.senders == 0 {
		s.mu.Unlock()
		clone.Drop()
		return rx.Poll(cx)
	}
	old := s.rxWaker
	s.rxWaker = clone
	s.mu.Unlock()
	dropAwaiter(old)
	return Result[T]{}, false
}

// Next returns a future of the next value that does not drop the receiver.
func (rx *ChanReceiver[T]) Next() Future[Result[T]] {
	return Unowned[Result[T]](rx)
}

// Recv awaits the next value from a coroutine body. ok is false once the channel is closed and
// drained.
func (rx *ChanReceiver[T]) Recv(cx Awaiter) (T, bool) {
	r := Await(cx, rx.Next())
	return r.Value, r.Err == nil
}

// RecvBlocking waits for the next value on the calling goroutine, outside any coroutine body.
func (rx *ChanReceiver[T]) RecvBlocking() (T, bool) {
	r := BlockOn(rx.Next())
	return r.Value, r.Err == nil
}

// TryRecv returns a queued value without parking. closed is true once every sender is gone and
// the queue is empty.
func (rx *ChanReceiver[T]) TryRecv() (v T, ok bool, closed bool) {
	s := rx.s
	s.mu.Lock()
	if v, ok := s.popLocked(); ok {
		tw := s.takeTxWakersLocked()
		s.mu.Unlock()
		wakeAll(tw)
		return v, true, false
	}
	closed = s.senders == 0
	s.mu.Unlock()
	return v, false, closed
}

// Close drops the receiver: later sends fail with ErrClosed. Values already queued stay
// available to TryRecv so the owner can dispose of them.
func (rx *ChanReceiver[T]) Close() {
	s := rx.s
	s.mu.Lock()
	s.receiverClosed = true
	w := s.rxWaker
	s.rxWaker = nil
	tw := s.takeTxWakersLocked()
	s.mu.Unlock()
	dropAwaiter(w)
	wakeAll(tw)
}
