package scheduler

import (
	"errors"
	"iter"
	"runtime"
)

// ErrCanceled reports that the other half of a oneshot, or a task, was dropped before producing a value.
var ErrCanceled = errors.New("scheduler: canceled")

// ErrClosed reports that a channel has no senders left or no receiver.
var ErrClosed = errors.New("scheduler: channel closed")

// ErrFull reports that a bounded channel has no free capacity.
var ErrFull = errors.New("scheduler: channel full")

// Waker wakes a parked future (std::task::Waker). A stored waker is obtained with Clone and is
// eventually consumed by Wake or released by Drop.
type Waker interface {
	Wake()
	WakeByRef()
	Clone() Waker
	Drop()
}

type wakerFn func()

func (w wakerFn) Wake()        { w() }
func (w wakerFn) WakeByRef()   { w() }
func (w wakerFn) Clone() Waker { return w }
func (w wakerFn) Drop()        {}

// WakerFn returns a Waker that calls f on every wake.
func WakerFn(f func()) Waker {
	return wakerFn(f)
}

// Context carries the waker of the current poll (std::task::Context).
type Context struct {
	waker Waker
}

// NewContext returns a Context for polling with waker.
func NewContext(waker Waker) *Context {
	return &Context{waker: waker}
}

// Waker returns the context's waker. Callers that keep it must Clone it.
func (cx *Context) Waker() Waker {
	return cx.waker
}

// Future is a value that becomes ready after being polled zero or more times. Poll returns the
// output and true once ready; while pending it arranges for cx's waker to be woken.
type Future[T any] interface {
	Poll(cx *Context) (T, bool)
}

// Result is the output of a fallible future.
type Result[T any] struct {
	Value T
	Err   error
}

type canceler interface {
	Cancel()
}

func cancelFuture[T any](f Future[T]) {
	if c, ok := f.(canceler); ok {
		c.Cancel()
	}
}

type pollFn[T any] func(cx *Context) (T, bool)

func (f pollFn[T]) Poll(cx *Context) (T, bool) { return f(cx) }

// PollFn returns a Future whose Poll calls f (futures::future::poll_fn).
func PollFn[T any](f func(cx *Context) (T, bool)) Future[T] {
	return pollFn[T](f)
}

type unowned[T any] struct {
	f Future[T]
}

func (u unowned[T]) Poll(cx *Context) (T, bool) { return u.f.Poll(cx) }

// Unowned wraps f so that Await, Select and Join poll it without dropping it (Rust's `&mut fut`).
func Unowned[T any](f Future[T]) Future[T] {
	return unowned[T]{f: f}
}

// Awaiter is the context a coroutine body parks through. Types outside this package implement
// it by embedding *Scope.
type Awaiter interface {
	scope() *Scope
}

// Scope is the parking context of one coroutine. A body must await only through the Scope it
// was given.
type Scope struct {
	cx      *Context
	yield   func(struct{}) bool
	stopped bool
}

func (s *Scope) scope() *Scope { return s }

type cancelUnwind struct{}

func (cancelUnwind) Error() string { return "scheduler: coroutine canceled" }

// IsCancelUnwind reports whether r, a value returned by recover, is the unwind of a canceled
// coroutine body. A deferred recover inside a body must re-panic such a value: the body must not
// continue after its future was dropped.
func IsCancelUnwind(r any) bool {
	_, ok := r.(cancelUnwind)
	return ok
}

func (s *Scope) park() {
	if s.yield == nil {
		panic("scheduler: await outside a coroutine body")
	}
	if s.stopped {
		panic(cancelUnwind{})
	}
	if !s.yield(struct{}{}) {
		s.stopped = true
		panic(cancelUnwind{})
	}
}

func (s *Scope) context() *Context {
	if s.stopped {
		panic(cancelUnwind{})
	}
	if s.cx == nil {
		panic("scheduler: await outside a coroutine body")
	}
	return s.cx
}

type coroutine[T any] struct {
	body    func(s *Scope) T
	scope   Scope
	next    func() (struct{}, bool)
	stop    func()
	out     T
	done    bool
	polling bool
}

// Async returns a Future that runs body as a coroutine (an `async` block). One Poll resumes the
// body until it parks in an await or returns. The coroutine's goroutine is created on the first
// Poll and ends when the body returns or the future is canceled. A canceled body unwinds from its
// await point with a panic; a deferred recover in the body must re-panic it (see IsCancelUnwind),
// and every later await in that body panics again.
func Async[T any](body func(s *Scope) T) Future[T] {
	return &coroutine[T]{body: body}
}

func (c *coroutine[T]) seq(yield func(struct{}) bool) {
	c.scope.yield = yield
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(cancelUnwind); ok {
				return
			}
			panic(r)
		}
	}()
	out := c.body(&c.scope)
	if !c.scope.stopped {
		c.out = out
	}
}

// Poll resumes the coroutine once.
func (c *coroutine[T]) Poll(cx *Context) (T, bool) {
	var zero T
	if c.done {
		panic("`async fn` resumed after completion")
	}
	if c.polling {
		panic("scheduler: coroutine polled re-entrantly")
	}
	if c.next == nil {
		c.next, c.stop = iter.Pull(c.seq)
	}
	c.polling = true
	c.scope.cx = cx
	finished := true
	defer func() {
		c.scope.cx = nil
		c.polling = false
		if finished {
			c.done = true
			c.body = nil
		}
	}()
	if _, more := c.next(); more {
		finished = false
		return zero, false
	}
	out := c.out
	c.out = zero
	return out, true
}

// Cancel drops the coroutine: a parked body unwinds from its await point, running its deferred
// calls. Only the owner of the future, which is the only one allowed to poll it, may call Cancel.
func (c *coroutine[T]) Cancel() {
	if c.polling {
		panic("scheduler: coroutine canceled while it is being polled")
	}
	if c.done {
		return
	}
	c.done = true
	c.body = nil
	if c.stop != nil {
		c.polling = true
		c.scope.cx = NewContext(noopWaker{})
		defer func() {
			c.scope.cx = nil
			c.polling = false
		}()
		c.stop()
	}
}

type noopWaker struct{}

func (noopWaker) Wake()        {}
func (noopWaker) WakeByRef()   {}
func (noopWaker) Clone() Waker { return noopWaker{} }
func (noopWaker) Drop()        {}

// Await polls f from a coroutine body until it is ready, parking between polls. Await owns f:
// when the body unwinds before f is ready, f is dropped (Task: canceled).
func Await[T any](cx Awaiter, f Future[T]) T {
	s := cx.scope()
	completed := false
	defer func() {
		if !completed {
			cancelFuture(f)
		}
	}()
	for {
		if v, ok := f.Poll(s.context()); ok {
			completed = true
			return v
		}
		s.park()
	}
}

type signal struct {
	ch chan struct{}
}

func newSignal() *signal {
	return &signal{ch: make(chan struct{}, 1)}
}

func (s *signal) notify() {
	select {
	case s.ch <- struct{}{}:
	default:
	}
}

// BlockOn polls f on the calling goroutine until it is ready, blocking the goroutine between
// polls (futures::executor::block_on). It must not be called from a coroutine body.
func BlockOn[T any](f Future[T]) T {
	sig := newSignal()
	cx := NewContext(WakerFn(sig.notify))
	for {
		if v, ok := f.Poll(cx); ok {
			return v
		}
		<-sig.ch
	}
}

// SelectCase is one arm of Select.
type SelectCase struct {
	poll   func(cx *Context) bool
	cancel func()
}

// Case returns a Select arm that runs then with f's output when f is the first ready future.
func Case[T any](f Future[T], then func(T)) SelectCase {
	return SelectCase{
		poll: func(cx *Context) bool {
			v, ok := f.Poll(cx)
			if ok && then != nil {
				then(v)
			}
			return ok
		},
		cancel: func() { cancelFuture(f) },
	}
}

// Select polls the cases in order until one is ready, runs its callback and returns its index
// (futures::select_biased!). Select owns the futures: the others are dropped when it returns or
// the body unwinds. It panics when given no cases.
func Select(cx Awaiter, cases ...SelectCase) int {
	if len(cases) == 0 {
		panic("scheduler: Select with no cases")
	}
	s := cx.scope()
	winner := -1
	defer func() {
		for i, c := range cases {
			if i != winner {
				c.cancel()
			}
		}
	}()
	for {
		for i, c := range cases {
			if c.poll(s.context()) {
				winner = i
				return i
			}
		}
		s.park()
	}
}

// Join polls every future until all are ready and returns their outputs in order
// (futures::future::join_all). Join owns the futures: unfinished ones are dropped if the body
// unwinds.
func Join[T any](cx Awaiter, futures ...Future[T]) []T {
	s := cx.scope()
	out := make([]T, len(futures))
	done := make([]bool, len(futures))
	remaining := len(futures)
	defer func() {
		for i, f := range futures {
			if !done[i] {
				cancelFuture(f)
			}
		}
	}()
	for remaining > 0 {
		for i, f := range futures {
			if done[i] {
				continue
			}
			if v, ok := f.Poll(s.context()); ok {
				out[i] = v
				done[i] = true
				remaining--
			}
		}
		if remaining > 0 {
			s.park()
		}
	}
	return out
}

// Join2 polls a and b until both are ready (futures::join!).
func Join2[A, B any](cx Awaiter, a Future[A], b Future[B]) (A, B) {
	s := cx.scope()
	var va A
	var vb B
	doneA, doneB := false, false
	defer func() {
		if !doneA {
			cancelFuture(a)
		}
		if !doneB {
			cancelFuture(b)
		}
	}()
	for {
		if !doneA {
			va, doneA = a.Poll(s.context())
		}
		if !doneB {
			vb, doneB = b.Poll(s.context())
		}
		if doneA && doneB {
			return va, vb
		}
		s.park()
	}
}

// CallbackDepth returns the number of native-to-Go callback frames on the calling goroutine's
// stack. Each one raises the OS-thread lock count that an Async coroutine records when it starts;
// resuming it at another depth is a fatal runtime error, so a main-thread dispatcher polls
// foreground tasks only at one fixed depth.
func CallbackDepth() int {
	pcs := make([]uintptr, 256)
	for {
		n := runtime.Callers(1, pcs)
		if n < len(pcs) {
			pcs = pcs[:n]
			break
		}
		pcs = make([]uintptr, 2*len(pcs))
	}
	depth := 0
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		if frame.Function == "runtime.cgocallbackg" {
			depth++
		}
		if !more {
			return depth
		}
	}
}
