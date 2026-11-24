package scheduler

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// LocalExecutor is pinned to one session. Its tasks run in order on whichever goroutine drains
// the dispatch destination given at construction: usually the main thread, or a dedicated
// goroutine for sessions made by SpawnDedicatedThread.
type LocalExecutor struct {
	sessionID SessionID
	scheduler Scheduler
	dispatch  func(*Runnable)
}

// NewLocalExecutor returns an executor that runs spawned tasks by passing their runnables to
// dispatch. scheduler provides clocks and timers.
func NewLocalExecutor(sessionID SessionID, scheduler Scheduler, dispatch func(*Runnable)) LocalExecutor {
	return LocalExecutor{sessionID: sessionID, scheduler: scheduler, dispatch: dispatch}
}

// ToSend returns a handle that submits tasks from any goroutine to this executor's dispatch
// destination.
func (e LocalExecutor) ToSend() SendLocalExecutor {
	return SendLocalExecutor{scheduler: e.scheduler, dispatch: e.dispatch}
}

// SessionID returns the executor's session.
func (e LocalExecutor) SessionID() SessionID {
	return e.sessionID
}

// Scheduler returns the executor's scheduler.
func (e LocalExecutor) Scheduler() Scheduler {
	return e.scheduler
}

// IsTest reports whether the executor runs on a TestScheduler.
func (e LocalExecutor) IsTest() bool {
	return e.scheduler.AsTest() != nil
}

// Spawn runs body as a task on this executor's session.
func (e LocalExecutor) Spawn[T any](body func(s *Scope) (T, error)) *Task[T] {
	return e.spawnAt(CallerLocation(1), body)
}

// SpawnAt is Spawn with an explicit spawn location, for wrappers that forward their caller's.
func (e LocalExecutor) SpawnAt[T any](location Location, body func(s *Scope) (T, error)) *Task[T] {
	return e.spawnAt(location, body)
}

func (e LocalExecutor) spawnAt[T any](location Location, body func(s *Scope) (T, error)) *Task[T] {
	return spawnTask(newRunnableMeta(location), body, e.schedule())
}

// SpawnWithDispatch is Spawn with the task's runnables routed through dispatch instead of the
// executor's own destination. dispatch must deliver them to this executor's goroutine.
func (e LocalExecutor) SpawnWithDispatch[T any](body func(s *Scope) (T, error), dispatch func(*Runnable)) *Task[T] {
	return spawnTask(newRunnableMeta(CallerLocation(1)), body, dispatch)
}

// BlockOn drives future to completion through the scheduler's Block, without running other
// tasks of this session.
func (e LocalExecutor) BlockOn[T any](future Future[T]) T {
	var output T
	done := false
	wrapper := PollFn(func(cx *Context) (struct{}, bool) {
		v, ok := future.Poll(cx)
		if ok {
			output = v
			done = true
		}
		return struct{}{}, ok
	})
	sessionID := e.sessionID
	e.scheduler.Block(&sessionID, wrapper, nil)
	if !done {
		panic("block_on future did not complete")
	}
	return output
}

// BlockWithTimeout drives future until it completes or timeout elapses. On timeout ok is false
// and future is left unfinished and owned by the caller, who may keep polling it.
func (e LocalExecutor) BlockWithTimeout[T any](timeout time.Duration, future Future[T]) (output T, ok bool) {
	wrapper := PollFn(func(cx *Context) (struct{}, bool) {
		v, ready := future.Poll(cx)
		if ready {
			output = v
			ok = true
		}
		return struct{}{}, ready
	})
	sessionID := e.sessionID
	e.scheduler.Block(&sessionID, wrapper, &timeout)
	return output, ok
}

// Timer returns a timer that fires after duration.
func (e LocalExecutor) Timer(duration time.Duration) *Timer {
	return e.scheduler.Timer(duration)
}

// Now returns the scheduler clock's current instant.
func (e LocalExecutor) Now() Instant {
	return e.scheduler.Clock().Now()
}

// SpawnDedicated runs f on a fresh session pinned to its own LocalExecutor: a new goroutine
// under a production scheduler, the test loop under TestScheduler. Canceling the returned task
// cancels the dedicated work; detaching lets it run on its own.
func (e LocalExecutor) SpawnDedicated[T any](f func(executor LocalExecutor, s *Scope) (T, error)) *Task[T] {
	return TaskAnyDowncast[T](e.scheduler.SpawnDedicated(boxDedicated(f)))
}

func (e LocalExecutor) schedule() func(*Runnable) {
	dispatch := e.dispatch
	return func(runnable *Runnable) { dispatch(runnable) }
}

// SendLocalExecutor submits tasks from any goroutine to a LocalExecutor's dispatch destination.
// It retains the scheduler and destination independently of the original executor.
type SendLocalExecutor struct {
	scheduler Scheduler
	dispatch  func(*Runnable)
}

// Spawn runs body as a task whose polls happen on the original executor's goroutine.
func (e SendLocalExecutor) Spawn[T any](body func(s *Scope) (T, error)) *Task[T] {
	return e.SpawnAt(CallerLocation(1), body)
}

// SpawnAt is Spawn with an explicit spawn location.
func (e SendLocalExecutor) SpawnAt[T any](location Location, body func(s *Scope) (T, error)) *Task[T] {
	dispatch := e.dispatch
	return spawnTask(newRunnableMeta(location), body, func(runnable *Runnable) { dispatch(runnable) })
}

func boxDedicated[T any](f func(executor LocalExecutor, s *Scope) (T, error)) func(executor LocalExecutor, s *Scope) (any, error) {
	return func(executor LocalExecutor, s *Scope) (any, error) {
		v, err := f(executor, s)
		return v, err
	}
}

// BackgroundExecutor spawns tasks on the scheduler's background queues.
type BackgroundExecutor struct {
	scheduler Scheduler
	weak      weakScheduler
}

// NewBackgroundExecutor returns a background executor for scheduler.
func NewBackgroundExecutor(scheduler Scheduler) BackgroundExecutor {
	return BackgroundExecutor{scheduler: scheduler, weak: downgradeScheduler(scheduler)}
}

// Spawn runs body as a background task with DefaultPriority.
func (e BackgroundExecutor) Spawn[T any](body func(s *Scope) (T, error)) *Task[T] {
	return e.SpawnWithPriorityAt(CallerLocation(1), DefaultPriority(), body)
}

// SpawnWithPriority runs body as a background task with priority.
func (e BackgroundExecutor) SpawnWithPriority[T any](priority Priority, body func(s *Scope) (T, error)) *Task[T] {
	return e.SpawnWithPriorityAt(CallerLocation(1), priority, body)
}

// SpawnWithPriorityAt is SpawnWithPriority with an explicit spawn location.
func (e BackgroundExecutor) SpawnWithPriorityAt[T any](location Location, priority Priority, body func(s *Scope) (T, error)) *Task[T] {
	return spawnTask(newRunnableMeta(location), recoverBackgroundPanic(body), e.scheduleWithPriority(priority))
}

func recoverBackgroundPanic[T any](body func(s *Scope) (T, error)) func(s *Scope) (T, error) {
	return func(s *Scope) (v T, err error) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			if IsCancelUnwind(r) {
				panic(r)
			}
			stack := debug.Stack()
			fmt.Fprintf(os.Stderr, "background task panicked: %v\n%s", r, stack)
			var zero T
			v = zero
			err = &PanicError{Value: r, Stack: stack}
		}()
		return body(s)
	}
}

// SpawnRealtime runs body on a dedicated realtime thread for audio processing.
func (e BackgroundExecutor) SpawnRealtime[T any](body func(s *Scope) (T, error)) *Task[T] {
	return e.SpawnRealtimeAt(CallerLocation(1), body)
}

// SpawnRealtimeAt is SpawnRealtime with an explicit spawn location.
func (e BackgroundExecutor) SpawnRealtimeAt[T any](location Location, body func(s *Scope) (T, error)) *Task[T] {
	tx, rx := Chan[*Runnable](1)
	e.scheduler.SpawnRealtime(func() {
		defer closeRunQueue(rx)
		for {
			runnable, ok := rx.RecvBlocking()
			if !ok {
				return
			}
			h := runnable.h
			if !runIsolated(runnable, "<unnamed>") || h.finished() {
				return
			}
		}
	})
	return spawnTask(newRunnableMeta(location), body, func(runnable *Runnable) {
		if tx.TrySend(runnable) != nil {
			runnable.Close()
		}
	})
}

// Timer returns a timer that fires after duration.
func (e BackgroundExecutor) Timer(duration time.Duration) *Timer {
	return e.scheduler.Timer(duration)
}

// Now returns the scheduler clock's current instant.
func (e BackgroundExecutor) Now() Instant {
	return e.scheduler.Clock().Now()
}

// Scheduler returns the executor's scheduler.
func (e BackgroundExecutor) Scheduler() Scheduler {
	return e.scheduler
}

// IsTest reports whether the executor runs on a TestScheduler.
func (e BackgroundExecutor) IsTest() bool {
	return e.scheduler.AsTest() != nil
}

// SpawnDedicated runs f on a fresh session pinned to its own LocalExecutor; see
// LocalExecutor.SpawnDedicated.
func (e BackgroundExecutor) SpawnDedicated[T any](f func(executor LocalExecutor, s *Scope) (T, error)) *Task[T] {
	return TaskAnyDowncast[T](e.scheduler.SpawnDedicated(boxDedicated(f)))
}

func (e BackgroundExecutor) scheduleWithPriority(priority Priority) func(*Runnable) {
	scheduler := e.weak
	return func(runnable *Runnable) {
		if s := scheduler.upgrade(); s != nil {
			s.ScheduleBackgroundWithPriority(runnable, priority)
		} else {
			runnable.Close()
		}
	}
}

// DedicatedExecutor is a long-lived handle to one dedicated session: every task spawned on it
// runs on that session's single goroutine. Close is its drop: it cancels the session, and
// queued and later spawns resolve to canceled tasks.
type DedicatedExecutor struct {
	sender  *ChanSender[*Runnable]
	session *Task[struct{}]
}

// NewDedicatedExecutor starts a dedicated session on executor. Under TestScheduler the session
// runs on the deterministic test loop.
func NewDedicatedExecutor(executor BackgroundExecutor) *DedicatedExecutor {
	sender, receiver := Chan[*Runnable](0)
	session := executor.SpawnDedicated(func(_ LocalExecutor, s *Scope) (struct{}, error) {
		defer func() {
			receiver.Close()
			for {
				runnable, ok, _ := receiver.TryRecv()
				if !ok {
					break
				}
				runnable.Close()
			}
		}()
		for {
			runnable, ok := receiver.Recv(s)
			if !ok {
				break
			}
			runnable.Run()
		}
		return struct{}{}, nil
	})
	return &DedicatedExecutor{sender: sender, session: session}
}

// Spawn runs body on the dedicated session's goroutine.
func (e *DedicatedExecutor) Spawn[T any](body func(s *Scope) (T, error)) *Task[T] {
	return e.SpawnAt(CallerLocation(1), body)
}

// SpawnAt is Spawn with an explicit spawn location.
func (e *DedicatedExecutor) SpawnAt[T any](location Location, body func(s *Scope) (T, error)) *Task[T] {
	sender := e.sender
	return spawnTask(newRunnableMeta(location), body, func(runnable *Runnable) {
		if sender.trySendIfOpen(runnable) != nil {
			runnable.Close()
		}
	})
}

// Close drops the handle and cancels the session.
func (e *DedicatedExecutor) Close() {
	e.sender.Close()
	e.session.Cancel()
}

func spawnTask[T any](meta RunnableMeta, body func(s *Scope) (T, error), schedule func(*Runnable)) *Task[T] {
	future := Async(func(s *Scope) Result[T] {
		v, err := body(s)
		return Result[T]{Value: v, Err: err}
	})
	runnable, task := SpawnAsyncTask(meta, future, schedule)
	runnable.Schedule()
	return TaskFromAsyncTask(task)
}

type taskKind uint8

const (
	taskKindConsumed taskKind = iota
	taskKindReady
	taskKindSpawned
	taskKindDowncast
	taskKindRendezvous
)

// Task is the handle to work running on an executor. Await yields the body's output and error.
// Cancel is the handle's drop: the work stops at its next park. Detach lets it run to completion.
type Task[T any] struct {
	kind       taskKind
	ready      Result[T]
	spawned    *AsyncTask[Result[T]]
	inner      *Task[any]
	rendezvous *rendezvousShared[T]
}

// ReadyTask returns a task that is already complete with v.
func ReadyTask[T any](v T) *Task[T] {
	return &Task[T]{kind: taskKindReady, ready: Result[T]{Value: v}}
}

// ReadyTaskErr returns a task that is already complete with err.
func ReadyTaskErr[T any](err error) *Task[T] {
	return &Task[T]{kind: taskKindReady, ready: Result[T]{Err: err}}
}

// TaskFromAsyncTask wraps an AsyncTask whose output is a Result.
func TaskFromAsyncTask[T any](task *AsyncTask[Result[T]]) *Task[T] {
	return &Task[T]{kind: taskKindSpawned, spawned: task}
}

// TaskAnyDowncast returns a typed view of a task whose output is boxed as any. Awaiting it
// panics if the output is not a T.
func TaskAnyDowncast[T any](task *Task[any]) *Task[T] {
	return &Task[T]{kind: taskKindDowncast, inner: task}
}

// IsReady reports whether awaiting the task would complete without parking.
func (t *Task[T]) IsReady() bool {
	switch t.kind {
	case taskKindReady:
		return true
	case taskKindSpawned:
		return t.spawned.IsFinished()
	case taskKindDowncast:
		return t.inner.IsReady()
	case taskKindRendezvous:
		return t.rendezvous.isReady()
	default:
		return false
	}
}

// Detach lets the task run to completion in the background. It is a no-op on a task that was
// already awaited, canceled or detached.
func (t *Task[T]) Detach() {
	kind := t.kind
	t.kind = taskKindConsumed
	switch kind {
	case taskKindSpawned:
		t.spawned.Detach()
	case taskKindDowncast:
		t.inner.Detach()
	case taskKindRendezvous:
		t.rendezvous.detach()
	}
	t.clear()
}

// Cancel drops the task: running work stops at its next park and never resumes. It is a no-op
// on a task that was already awaited, canceled or detached.
func (t *Task[T]) Cancel() {
	kind := t.kind
	t.kind = taskKindConsumed
	switch kind {
	case taskKindSpawned:
		t.spawned.Cancel()
	case taskKindDowncast:
		t.inner.Cancel()
	case taskKindRendezvous:
		t.rendezvous.drop()
	}
	t.clear()
}

func (t *Task[T]) clear() {
	t.ready = Result[T]{}
	t.spawned = nil
	t.inner = nil
	t.rendezvous = nil
}

// Fallible converts the task into one that reports cancellation as ErrCanceled instead of
// panicking.
func (t *Task[T]) Fallible() *FallibleTask[T] {
	moved := *t
	t.kind = taskKindConsumed
	t.clear()
	return &FallibleTask[T]{task: &moved}
}

// Poll resolves to the task's output. It panics if the task was closed without output.
func (t *Task[T]) Poll(cx *Context) (Result[T], bool) {
	r, some, ready := t.poll(cx)
	if ready && !some {
		panic("Task polled after completion")
	}
	return r, ready
}

func (t *Task[T]) poll(cx *Context) (r Result[T], some bool, ready bool) {
	for {
		switch t.kind {
		case taskKindReady:
			r = t.ready
			t.kind = taskKindConsumed
			t.clear()
			return r, true, true
		case taskKindSpawned:
			r, some, ready = t.spawned.pollTask(cx)
			if ready {
				t.kind = taskKindConsumed
				t.clear()
			}
			return r, some, ready
		case taskKindDowncast:
			inner, innerSome, innerReady := t.inner.poll(cx)
			if !innerReady {
				return r, false, false
			}
			t.kind = taskKindConsumed
			t.clear()
			if !innerSome {
				return r, false, true
			}
			v, ok := inner.Value.(T)
			if !ok && inner.Value != nil {
				panic("Task::poll: downcast type mismatch")
			}
			return Result[T]{Value: v, Err: inner.Err}, true, true
		case taskKindRendezvous:
			delivered, ok := t.rendezvous.pollTake(cx)
			if !ok {
				return r, false, false
			}
			*t = *delivered
			delivered.kind = taskKindConsumed
			delivered.clear()
		default:
			panic("Task polled after completion")
		}
	}
}

// Await parks the coroutine body until the task completes and returns its output. If the body
// unwinds first, the task is canceled, as dropping an awaited Rust task cancels it.
func (t *Task[T]) Await(cx Awaiter) (T, error) {
	r := Await[Result[T]](cx, t)
	return r.Value, r.Err
}

// Wait is Await without the value.
func (t *Task[T]) Wait(cx Awaiter) error {
	_, err := t.Await(cx)
	return err
}

// FallibleTask is a task that resolves to ErrCanceled instead of panicking when it was
// canceled.
type FallibleTask[T any] struct {
	task *Task[T]
}

// FallibleTaskReady returns a fallible task that is already complete with v.
func FallibleTaskReady[T any](v T) *FallibleTask[T] {
	return &FallibleTask[T]{task: ReadyTask(v)}
}

// Detach lets the task run to completion in the background.
func (t *FallibleTask[T]) Detach() {
	t.task.Detach()
}

// Cancel drops the task.
func (t *FallibleTask[T]) Cancel() {
	t.task.Cancel()
}

// Poll resolves to the task's output, or to ErrCanceled if the task was canceled.
func (t *FallibleTask[T]) Poll(cx *Context) (Result[T], bool) {
	r, some, ready := t.task.poll(cx)
	if ready && !some {
		return Result[T]{Err: ErrCanceled}, true
	}
	return r, ready
}

// Await parks the coroutine body until the task completes or is canceled.
func (t *FallibleTask[T]) Await(cx Awaiter) (T, error) {
	r := Await[Result[T]](cx, t)
	return r.Value, r.Err
}

type rendezvousStateKind uint8

const (
	rendezvousPending rendezvousStateKind = iota
	rendezvousDelivered
	rendezvousCancelled
	rendezvousDetached
	rendezvousTaken
)

type rendezvousShared[T any] struct {
	mu    sync.Mutex
	state rendezvousStateKind
	waker Waker
	task  *Task[T]
}

type rendezvousSender[T any] struct {
	shared *rendezvousShared[T]
}

func taskRendezvous[T any]() (*Task[T], *rendezvousSender[T]) {
	shared := &rendezvousShared[T]{}
	return &Task[T]{kind: taskKindRendezvous, rendezvous: shared}, &rendezvousSender[T]{shared: shared}
}

func (r *rendezvousShared[T]) pollTake(cx *Context) (*Task[T], bool) {
	clone := cx.Waker().Clone()
	r.mu.Lock()
	switch r.state {
	case rendezvousDelivered:
		task := r.task
		r.task = nil
		r.state = rendezvousTaken
		old := r.waker
		r.waker = nil
		r.mu.Unlock()
		clone.Drop()
		dropAwaiter(old)
		return task, true
	case rendezvousPending:
		old := r.waker
		r.waker = clone
		r.mu.Unlock()
		dropAwaiter(old)
		return nil, false
	default:
		r.mu.Unlock()
		clone.Drop()
		panic("a rendezvous task was polled after its receiver was consumed")
	}
}

func (r *rendezvousShared[T]) isReady() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == rendezvousDelivered {
		return r.task.IsReady()
	}
	return false
}

func (r *rendezvousShared[T]) detach() {
	r.mu.Lock()
	prev := r.state
	r.state = rendezvousDetached
	switch prev {
	case rendezvousDelivered:
		task := r.task
		r.task = nil
		r.state = rendezvousTaken
		r.mu.Unlock()
		task.Detach()
	case rendezvousPending:
		r.mu.Unlock()
	default:
		r.state = prev
		r.mu.Unlock()
		panic("a rendezvous task was detached after its receiver was consumed")
	}
}

func (r *rendezvousShared[T]) drop() {
	r.mu.Lock()
	switch r.state {
	case rendezvousPending:
		r.state = rendezvousCancelled
		w := r.waker
		r.waker = nil
		r.mu.Unlock()
		dropAwaiter(w)
	case rendezvousDelivered:
		task := r.task
		r.task = nil
		r.state = rendezvousCancelled
		r.mu.Unlock()
		task.Cancel()
	default:
		r.mu.Unlock()
	}
}

func (s *rendezvousSender[T]) deliver(task *Task[T]) {
	r := s.shared
	r.mu.Lock()
	prev := r.state
	switch prev {
	case rendezvousPending:
		r.state = rendezvousDelivered
		r.task = task
		w := r.waker
		r.waker = nil
		r.mu.Unlock()
		wakeAwaiter(w)
	case rendezvousCancelled:
		r.mu.Unlock()
		task.Cancel()
	case rendezvousDetached:
		r.mu.Unlock()
		task.Detach()
	default:
		r.mu.Unlock()
		panic("a rendezvous task was delivered twice")
	}
}

// PanicError is the error a background task resolves to when its body panics: Value is the
// recovered value and Stack the panicking goroutine's stack.
type PanicError struct {
	Value any
	Stack []byte
}

// Error returns the panic value's message.
func (e *PanicError) Error() string {
	return fmt.Sprintf("scheduler: background task panicked: %v", e.Value)
}
