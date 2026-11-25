package scheduler

import "sync"

type oneshotState[T any] struct {
	mu             sync.Mutex
	value          T
	hasValue       bool
	senderClosed   bool
	receiverClosed bool
	rxWaker        Waker
}

// OneshotSender is the sending half of a oneshot channel. Close is its drop.
type OneshotSender[T any] struct {
	s *oneshotState[T]
}

// OneshotReceiver is the receiving half of a oneshot channel; it is a Future of the value.
// Close is its drop.
type OneshotReceiver[T any] struct {
	s *oneshotState[T]
}

// Oneshot returns a channel that carries a single value (futures::channel::oneshot::channel).
func Oneshot[T any]() (*OneshotSender[T], *OneshotReceiver[T]) {
	s := &oneshotState[T]{}
	return &OneshotSender[T]{s: s}, &OneshotReceiver[T]{s: s}
}

// Send delivers v and consumes the sender. It returns ErrCanceled if the receiver is gone.
func (tx *OneshotSender[T]) Send(v T) error {
	s := tx.s
	s.mu.Lock()
	if s.senderClosed {
		s.mu.Unlock()
		panic("scheduler: OneshotSender used after Send or Close")
	}
	s.senderClosed = true
	if s.receiverClosed {
		s.mu.Unlock()
		return ErrCanceled
	}
	s.value = v
	s.hasValue = true
	w := s.rxWaker
	s.rxWaker = nil
	s.mu.Unlock()
	wakeAwaiter(w)
	return nil
}

// Close drops the sender; a receiver that has no value yet resolves to ErrCanceled.
func (tx *OneshotSender[T]) Close() {
	s := tx.s
	s.mu.Lock()
	if s.senderClosed {
		s.mu.Unlock()
		return
	}
	s.senderClosed = true
	w := s.rxWaker
	s.rxWaker = nil
	s.mu.Unlock()
	wakeAwaiter(w)
}

// IsCanceled reports whether the receiver was dropped.
func (tx *OneshotSender[T]) IsCanceled() bool {
	s := tx.s
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.receiverClosed
}

// Poll resolves to the sent value, or to ErrCanceled once the sender is dropped without sending.
func (rx *OneshotReceiver[T]) Poll(cx *Context) (Result[T], bool) {
	s := rx.s
	clone := cx.Waker().Clone()
	s.mu.Lock()
	if r, ok := s.takeLocked(); ok {
		s.mu.Unlock()
		clone.Drop()
		return r, true
	}
	old := s.rxWaker
	s.rxWaker = clone
	s.mu.Unlock()
	dropAwaiter(old)
	return Result[T]{}, false
}

func (s *oneshotState[T]) takeLocked() (Result[T], bool) {
	if s.hasValue {
		v := s.value
		var zero T
		s.value = zero
		s.hasValue = false
		s.receiverClosed = true
		return Result[T]{Value: v}, true
	}
	if s.senderClosed || s.receiverClosed {
		return Result[T]{Err: ErrCanceled}, true
	}
	return Result[T]{}, false
}

// Recv awaits the value from a coroutine body. It consumes the receiver, which is dropped if the
// body unwinds first.
func (rx *OneshotReceiver[T]) Recv(cx Awaiter) (T, error) {
	r := Await[Result[T]](cx, rx)
	return r.Value, r.Err
}

// TryRecv returns the value if it was sent. ok is false while the sender is still alive and has
// not sent; err is ErrCanceled once the sender is gone without sending.
func (rx *OneshotReceiver[T]) TryRecv() (v T, ok bool, err error) {
	s := rx.s
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasValue {
		r, _ := s.takeLocked()
		return r.Value, true, nil
	}
	if s.senderClosed {
		return v, false, ErrCanceled
	}
	return v, false, nil
}

// Close drops the receiver; a later Send fails with ErrCanceled.
func (rx *OneshotReceiver[T]) Close() {
	s := rx.s
	s.mu.Lock()
	s.receiverClosed = true
	var zero T
	s.value = zero
	s.hasValue = false
	w := s.rxWaker
	s.rxWaker = nil
	s.mu.Unlock()
	dropAwaiter(w)
}

// Cancel is Close; it lets Await, Select and Join drop an owned receiver.
func (rx *OneshotReceiver[T]) Cancel() {
	rx.Close()
}
