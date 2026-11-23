package scheduler

import "sync"

const (
	stateScheduled uint8 = 1 << iota
	stateRunning
	stateCompleted
	stateClosed
	stateTask
)

type taskImpl interface {
	poll(cx *Context) bool
	dropFuture()
	dropOutput()
}

type header struct {
	mu       sync.Mutex
	state    uint8
	awaiter  Waker
	meta     RunnableMeta
	schedule func(*Runnable)
	cx       *Context
	impl     taskImpl
}

type taskWaker struct {
	h *header
}

func (w taskWaker) Wake()        { w.h.wake() }
func (w taskWaker) WakeByRef()   { w.h.wake() }
func (w taskWaker) Clone() Waker { return w }
func (w taskWaker) Drop()        {}

func (h *header) wake() {
	h.mu.Lock()
	if h.state&(stateCompleted|stateClosed|stateScheduled) != 0 {
		h.mu.Unlock()
		return
	}
	h.state |= stateScheduled
	if h.state&stateRunning != 0 {
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()
	h.schedule(&Runnable{h: h})
}

func (h *header) takeAwaiterLocked() Waker {
	aw := h.awaiter
	h.awaiter = nil
	return aw
}

func (h *header) registerAwaiter(w Waker) {
	clone := w.Clone()
	h.mu.Lock()
	old := h.awaiter
	h.awaiter = clone
	h.mu.Unlock()
	if old != nil {
		old.Drop()
	}
}

func wakeAwaiter(aw Waker) {
	if aw != nil {
		aw.Wake()
	}
}

func dropAwaiter(aw Waker) {
	if aw != nil {
		aw.Drop()
	}
}

// Runnable is the right to poll a scheduled task once (async_task::Runnable). A scheduler
// either runs it with Run or drops it with Close; exactly one of the two happens.
type Runnable struct {
	h *header
}

func (r *Runnable) take() *header {
	h := r.h
	if h == nil {
		panic("scheduler: Runnable used after Run or Close")
	}
	r.h = nil
	return h
}

// Metadata returns the task's metadata.
func (r *Runnable) Metadata() *RunnableMeta {
	if r.h == nil {
		panic("scheduler: Runnable used after Run or Close")
	}
	return &r.h.meta
}

// Waker returns a waker that reschedules the task.
func (r *Runnable) Waker() Waker {
	if r.h == nil {
		panic("scheduler: Runnable used after Run or Close")
	}
	return taskWaker{h: r.h}
}

// Schedule hands the runnable to the task's schedule function.
func (r *Runnable) Schedule() {
	h := r.take()
	h.schedule(&Runnable{h: h})
}

// Run polls the task once. It returns true if the task was woken while running and has been
// rescheduled. A panic in the future closes the task and propagates out of Run.
func (r *Runnable) Run() bool {
	h := r.take()
	h.mu.Lock()
	if h.state&stateClosed != 0 {
		h.state &^= stateScheduled
		aw := h.takeAwaiterLocked()
		h.mu.Unlock()
		h.impl.dropFuture()
		wakeAwaiter(aw)
		return false
	}
	h.state = h.state&^stateScheduled | stateRunning
	h.mu.Unlock()

	panicking := true
	defer func() {
		if !panicking {
			return
		}
		h.mu.Lock()
		h.state = h.state&^(stateRunning|stateScheduled) | stateClosed
		aw := h.takeAwaiterLocked()
		h.mu.Unlock()
		h.impl.dropFuture()
		wakeAwaiter(aw)
	}()
	ready := h.impl.poll(h.cx)
	panicking = false

	h.mu.Lock()
	if ready {
		h.state = h.state&^(stateRunning|stateScheduled) | stateCompleted
		dropOutput := h.state&stateTask == 0
		if dropOutput {
			h.state |= stateClosed
		}
		aw := h.takeAwaiterLocked()
		h.mu.Unlock()
		if dropOutput {
			h.impl.dropOutput()
		}
		wakeAwaiter(aw)
		return false
	}
	if h.state&stateClosed != 0 {
		h.state &^= stateRunning | stateScheduled
		aw := h.takeAwaiterLocked()
		h.mu.Unlock()
		h.impl.dropFuture()
		wakeAwaiter(aw)
		return false
	}
	if h.state&stateScheduled != 0 {
		h.state &^= stateRunning
		h.mu.Unlock()
		h.schedule(&Runnable{h: h})
		return true
	}
	h.state &^= stateRunning
	h.mu.Unlock()
	return false
}

// Close drops the runnable without running it: the task is closed, its future is dropped and
// whoever awaits it is woken.
func (r *Runnable) Close() {
	if r.h == nil {
		return
	}
	h := r.take()
	h.mu.Lock()
	h.state = h.state&^stateScheduled | stateClosed
	aw := h.takeAwaiterLocked()
	h.mu.Unlock()
	h.impl.dropFuture()
	wakeAwaiter(aw)
}

type rawTask[T any] struct {
	header
	future  Future[T]
	out     T
	dropped bool
}

func (t *rawTask[T]) poll(cx *Context) bool {
	v, ok := t.future.Poll(cx)
	if ok {
		t.out = v
		t.future = nil
	}
	return ok
}

func (t *rawTask[T]) dropFuture() {
	if t.dropped || t.future == nil {
		t.dropped = true
		return
	}
	t.dropped = true
	f := t.future
	t.future = nil
	cancelFuture(f)
}

func (t *rawTask[T]) dropOutput() {
	var zero T
	t.out = zero
}

// AsyncTask is the handle to a spawned future (async_task::Task). Cancel is its drop: it
// cancels the future at its next park unless it already finished.
type AsyncTask[T any] struct {
	raw *rawTask[T]
}

// SpawnAsyncTask creates a task for future whose runnables go to schedule
// (async_task::Builder::metadata(meta).spawn). The returned runnable must be scheduled or run
// to start the task.
func SpawnAsyncTask[T any](meta RunnableMeta, future Future[T], schedule func(*Runnable)) (*Runnable, *AsyncTask[T]) {
	raw := &rawTask[T]{future: future}
	raw.state = stateScheduled | stateTask
	raw.meta = meta
	raw.schedule = schedule
	raw.impl = raw
	raw.cx = NewContext(taskWaker{h: &raw.header})
	return &Runnable{h: &raw.header}, &AsyncTask[T]{raw: raw}
}

// Metadata returns the task's metadata.
func (t *AsyncTask[T]) Metadata() *RunnableMeta {
	return &t.raw.meta
}

// IsFinished reports whether the task completed or was closed.
func (t *AsyncTask[T]) IsFinished() bool {
	h := &t.raw.header
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state&(stateCompleted|stateClosed) != 0
}

// Cancel drops the handle and cancels the task. A task that is idle is scheduled once more so
// the executor drops its future; a running task is dropped when its current poll returns.
func (t *AsyncTask[T]) Cancel() {
	h := &t.raw.header
	h.mu.Lock()
	if h.state&stateTask == 0 {
		h.mu.Unlock()
		return
	}
	h.state &^= stateTask
	schedule := false
	dropOutput := false
	var aw Waker
	switch {
	case h.state&(stateCompleted|stateClosed) == 0:
		if h.state&(stateScheduled|stateRunning) == 0 {
			schedule = true
			h.state |= stateScheduled
		}
		h.state |= stateClosed
		aw = h.takeAwaiterLocked()
	case h.state&stateCompleted != 0 && h.state&stateClosed == 0:
		h.state |= stateClosed
		dropOutput = true
	}
	h.mu.Unlock()
	if dropOutput {
		t.raw.dropOutput()
	}
	if schedule {
		h.schedule(&Runnable{h: h})
	}
	wakeAwaiter(aw)
}

// Detach drops the handle and lets the task run to completion.
func (t *AsyncTask[T]) Detach() {
	h := &t.raw.header
	h.mu.Lock()
	if h.state&stateTask == 0 {
		h.mu.Unlock()
		return
	}
	h.state &^= stateTask
	dropOutput := false
	if h.state&stateCompleted != 0 && h.state&stateClosed == 0 {
		h.state |= stateClosed
		dropOutput = true
	}
	h.mu.Unlock()
	if dropOutput {
		t.raw.dropOutput()
	}
}

func (t *AsyncTask[T]) pollTask(cx *Context) (v T, some bool, ready bool) {
	h := &t.raw.header
	h.mu.Lock()
	if h.state&stateClosed != 0 {
		if h.state&(stateScheduled|stateRunning) != 0 {
			h.mu.Unlock()
			h.registerAwaiter(cx.Waker())
			h.mu.Lock()
			pending := h.state&(stateScheduled|stateRunning) != 0
			h.mu.Unlock()
			if pending {
				return v, false, false
			}
			return t.pollTask(cx)
		}
		aw := h.takeAwaiterLocked()
		h.mu.Unlock()
		dropAwaiter(aw)
		return v, false, true
	}
	if h.state&stateCompleted == 0 {
		h.mu.Unlock()
		h.registerAwaiter(cx.Waker())
		h.mu.Lock()
		finished := h.state&(stateCompleted|stateClosed) != 0
		h.mu.Unlock()
		if !finished {
			return v, false, false
		}
		return t.pollTask(cx)
	}
	h.state |= stateClosed
	v = t.raw.out
	var zero T
	t.raw.out = zero
	aw := h.takeAwaiterLocked()
	h.mu.Unlock()
	dropAwaiter(aw)
	return v, true, true
}

// Poll returns the task's output once it completed. It panics if the task was closed without
// output, as async_task does.
func (t *AsyncTask[T]) Poll(cx *Context) (T, bool) {
	v, some, ready := t.pollTask(cx)
	if !ready {
		return v, false
	}
	if !some {
		panic("Task polled after completion")
	}
	return v, true
}

func (h *header) finished() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state&(stateCompleted|stateClosed) != 0
}
