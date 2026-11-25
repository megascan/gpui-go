package gpui

import (
	"bytes"
	"container/heap"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const minThreads = 2

// ThreadedDispatcher is a multithreaded PlatformDispatcher for tests and benchmarks. Background
// tasks run in parallel on a pool of worker goroutines and timers fire in real time on a timer
// goroutine; main-thread tasks are queued until the creating goroutine drains them with
// RunUntilIdle, since no platform run loop pumps them.
type ThreadedDispatcher struct {
	PlatformDispatcherDefaults
	backgroundSender *PriorityQueueSender[RunnableVariant]
	mainSender       *PriorityQueueSender[RunnableVariant]
	mainMu           sync.Mutex
	mainReceiver     *PriorityQueueReceiver[RunnableVariant]
	timers           *timerQueue
	idle             *idleTracker
	mainThreadID     int64
	mainProfilerID   ThreadID
}

type condvar struct {
	cond *sync.Cond
}

func newCondvar(mu *sync.Mutex) condvar {
	return condvar{cond: sync.NewCond(mu)}
}

func (c condvar) wait() {
	c.cond.Wait()
}

func (c condvar) waitFor(d time.Duration) {
	mu := c.cond.L
	t := time.AfterFunc(d, func() {
		mu.Lock()
		c.cond.Broadcast()
		mu.Unlock()
	})
	c.cond.Wait()
	t.Stop()
}

func (c condvar) waitUntil(due time.Time) {
	c.waitFor(time.Until(due))
}

func (c condvar) notifyAll() {
	c.cond.Broadcast()
}

type idleTracker struct {
	mu       sync.Mutex
	inflight int
	condvar  condvar
}

func newIdleTracker() *idleTracker {
	t := &idleTracker{}
	t.condvar = newCondvar(&t.mu)
	return t
}

func (t *idleTracker) increment() {
	t.mu.Lock()
	t.inflight++
	t.mu.Unlock()
}

func (t *idleTracker) decrement() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inflight--
	if t.inflight == 0 {
		t.condvar.notifyAll()
	}
}

func (t *idleTracker) notifyUnderLock() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.condvar.notifyAll()
}

type timerQueue struct {
	mu      sync.Mutex
	state   timerQueueState
	condvar condvar
}

type timerQueueState struct {
	heap    timerHeap
	nextSeq uint64
}

type timerEntry struct {
	due      time.Time
	seq      uint64
	runnable RunnableVariant
}

type timerHeap []timerEntry

func (h timerHeap) Len() int { return len(h) }

func (h timerHeap) Less(i, j int) bool {
	if c := h[i].due.Compare(h[j].due); c != 0 {
		return c < 0
	}
	return h[i].seq < h[j].seq
}

func (h timerHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *timerHeap) Push(x any) { *h = append(*h, x.(timerEntry)) }

func (h *timerHeap) Pop() any {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = timerEntry{}
	*h = old[:n-1]
	return entry
}

func (h timerHeap) peek() (timerEntry, bool) {
	if len(h) == 0 {
		return timerEntry{}, false
	}
	return h[0], true
}

func runProfiled(thread ThreadID, runnable RunnableVariant) {
	meta := runnable.Metadata()
	location := meta.Location
	spawned := meta.Spawned
	UpdateRunningTask(thread, spawned, location)
	runnable.Run()
	SaveTaskTiming(thread)
}

// NewThreadedDispatcher returns a dispatcher whose main thread is the calling goroutine. Its
// worker and timer goroutines live for the lifetime of the process; create it once and reuse it.
func NewThreadedDispatcher() *ThreadedDispatcher {
	backgroundSender, backgroundReceiver := NewPriorityQueueReceiver[RunnableVariant]()
	mainSender, mainReceiver := NewPriorityQueueReceiver[RunnableVariant]()
	idle := newIdleTracker()

	threadCount := max(runtime.NumCPU(), minThreads)
	for i := range threadCount {
		receiver := backgroundReceiver.Clone()
		thread := NewThreadID(Some(fmt.Sprintf("ThreadedDispatcherWorker-%d", i)))
		go func() {
			for {
				runnable, err := receiver.Pop()
				if err != nil {
					return
				}
				func() {
					defer idle.decrement()
					runProfiled(thread, runnable)
				}()
			}
		}()
	}
	backgroundReceiver.Close()

	timers := &timerQueue{}
	timers.condvar = newCondvar(&timers.mu)
	{
		thread := NewThreadID(Some("ThreadedDispatcherTimer"))
		go func() {
			timers.mu.Lock()
			for {
				entry, ok := timers.state.heap.peek()
				if !ok {
					timers.condvar.wait()
					continue
				}
				due := entry.due
				if due.After(time.Now()) {
					timers.condvar.waitUntil(due)
					continue
				}
				if timers.state.heap.Len() == 0 {
					continue
				}
				entry = heap.Pop(&timers.state.heap).(timerEntry)
				idle.increment()
				timers.mu.Unlock()

				func() {
					defer idle.decrement()
					runProfiled(thread, entry.runnable)
				}()

				timers.mu.Lock()
			}
		}()
	}

	d := &ThreadedDispatcher{
		backgroundSender: backgroundSender,
		mainSender:       mainSender,
		mainReceiver:     mainReceiver,
		timers:           timers,
		idle:             idle,
		mainThreadID:     goroutineID(),
		mainProfilerID:   NewMainThreadID(None[string]()),
	}
	BindPlatformDispatcherDefaults(&d.PlatformDispatcherDefaults, d)
	return d
}

// RunUntilIdle runs queued main-thread tasks and waits until no background or timer work is
// queued, running or already due. Timers that are not due yet are not waited for. It must be
// called on the goroutine that created the dispatcher.
func (d *ThreadedDispatcher) RunUntilIdle() {
	d.runUntilIdleWith(d.drainMainQueue)
}

func (d *ThreadedDispatcher) runUntilIdleWith(runTurn func() bool) {
	if !d.IsMainThread() {
		panic("run_until_idle must be called on the threaded dispatcher's main thread")
	}
	for {
		if runTurn() {
			continue
		}

		if d.hasDueTimer() {
			d.idle.mu.Lock()
			d.idle.condvar.waitFor(time.Millisecond)
			d.idle.mu.Unlock()
			continue
		}

		d.idle.mu.Lock()
		if d.mainQueueHasWork() {
			d.idle.mu.Unlock()
			continue
		}
		if d.idle.inflight == 0 {
			d.idle.mu.Unlock()
			return
		}
		d.idle.condvar.wait()
		d.idle.mu.Unlock()
	}
}

func (d *ThreadedDispatcher) runUntil(ready func() bool) {
	d.runUntilWithFrames(ready, func() bool { return false })
}

func (d *ThreadedDispatcher) runOneMainTask() bool {
	if !d.IsMainThread() {
		panic("main tasks must run on the main thread")
	}
	d.mainMu.Lock()
	runnable, ok, err := d.mainReceiver.TryPop()
	d.mainMu.Unlock()
	if err != nil || !ok {
		return false
	}
	runProfiled(d.mainProfilerID, runnable)
	return true
}

// RunReadyMainTasks runs up to the initial queue length of main-thread tasks and reports
// whether any ran. Newly dispatched higher-priority tasks can overtake the original ones.
func (d *ThreadedDispatcher) RunReadyMainTasks() bool {
	return d.runReadyMainTasksWith(func() bool { return true }, func() {})
}

func (d *ThreadedDispatcher) runReadyMainTasksWith(shouldContinue func() bool, afterPoll func()) bool {
	if !d.IsMainThread() {
		panic("run_ready_main_tasks must be called on the threaded dispatcher's main thread")
	}
	d.mainMu.Lock()
	pending := d.mainReceiver.len()
	d.mainMu.Unlock()
	ranAny := false
	for range pending {
		if !shouldContinue() || !d.runOneMainTask() {
			break
		}
		ranAny = true
		afterPoll()
	}
	return ranAny
}

func (d *ThreadedDispatcher) runUntilWithFrames(ready func() bool, dispatchFrames func() bool) {
	if !d.IsMainThread() {
		panic("main tasks must run on the main thread")
	}
	for {
		if ready() {
			return
		}
		ranTask := d.runOneMainTask()
		if ready() {
			return
		}
		dispatchedFrames := dispatchFrames()
		if ranTask || dispatchedFrames {
			continue
		}

		d.idle.mu.Lock()
		if d.mainQueueHasWork() {
			d.idle.mu.Unlock()
			continue
		}
		d.idle.condvar.wait()
		d.idle.mu.Unlock()
	}
}

// CancelPendingTimers cancels every pending timer so timers armed by one workload cannot fire
// during a later one, returning how many were canceled. Closing a timer runnable wakes the task
// awaiting the timer; call RunUntilIdle afterwards to drain the work that unblocks.
func (d *ThreadedDispatcher) CancelPendingTimers() int {
	d.timers.mu.Lock()
	timers := d.timers.state.heap
	d.timers.state.heap = nil
	d.timers.condvar.notifyAll()
	d.timers.mu.Unlock()
	canceled := len(timers)
	for _, entry := range timers {
		entry.runnable.Close()
	}
	return canceled
}

// DebugState describes the dispatcher's idle-tracking state.
func (d *ThreadedDispatcher) DebugState() string {
	d.idle.mu.Lock()
	inflight := d.idle.inflight
	d.idle.mu.Unlock()
	d.timers.mu.Lock()
	timers := d.timers.state.heap.Len()
	d.timers.mu.Unlock()
	mainQueueHasWork := d.mainQueueHasWork()
	return fmt.Sprintf("ThreadedDispatcher { inflight: %d, pending_timers: %d, main_queue_has_work: %t }",
		inflight, timers, mainQueueHasWork)
}

func (d *ThreadedDispatcher) isIdle() bool {
	if d.mainQueueHasWork() || d.hasDueTimer() {
		return false
	}
	d.idle.mu.Lock()
	defer d.idle.mu.Unlock()
	return d.idle.inflight == 0
}

func (d *ThreadedDispatcher) hasDueTimer() bool {
	d.timers.mu.Lock()
	defer d.timers.mu.Unlock()
	entry, ok := d.timers.state.heap.peek()
	return ok && !entry.due.After(time.Now())
}

func (d *ThreadedDispatcher) mainQueueHasWork() bool {
	d.mainMu.Lock()
	defer d.mainMu.Unlock()
	return !d.mainReceiver.IsEmpty()
}

func (d *ThreadedDispatcher) drainMainQueue() bool {
	ranAny := false
	for {
		d.mainMu.Lock()
		runnable, ok, err := d.mainReceiver.TryPop()
		d.mainMu.Unlock()
		if err != nil || !ok {
			return ranAny
		}
		runProfiled(d.mainProfilerID, runnable)
		ranAny = true
	}
}

// IsMainThread reports whether the caller runs on the creating goroutine, or in a task body that
// goroutine is polling.
func (d *ThreadedDispatcher) IsMainThread() bool {
	return currentThreadGoroutineID() == d.mainThreadID
}

// Dispatch queues a runnable on the worker pool.
func (d *ThreadedDispatcher) Dispatch(runnable RunnableVariant, priority Priority) {
	d.idle.increment()
	if err := d.backgroundSender.Send(priority, runnable); err != nil {
		panic("threaded dispatcher workers are no longer running")
	}
}

// DispatchOnMainThread queues a runnable for the main goroutine.
func (d *ThreadedDispatcher) DispatchOnMainThread(runnable RunnableVariant, priority Priority) {
	if err := d.mainSender.Send(priority, runnable); err != nil {
		return
	}
	d.idle.notifyUnderLock()
}

// DispatchAfter runs the runnable on the timer goroutine once duration elapsed.
func (d *ThreadedDispatcher) DispatchAfter(duration time.Duration, runnable RunnableVariant) {
	d.timers.mu.Lock()
	defer d.timers.mu.Unlock()
	seq := d.timers.state.nextSeq
	d.timers.state.nextSeq++
	heap.Push(&d.timers.state.heap, timerEntry{
		due:      time.Now().Add(duration),
		seq:      seq,
		runnable: runnable,
	})
	d.timers.condvar.notifyAll()
}

// SpawnRealtime runs f on a plain new goroutine.
func (d *ThreadedDispatcher) SpawnRealtime(f func()) {
	go f()
}

// AsThreaded returns d.
func (d *ThreadedDispatcher) AsThreaded() *ThreadedDispatcher {
	return d
}

func goroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	b := bytes.TrimPrefix(buf[:n], []byte("goroutine "))
	if i := bytes.IndexByte(b, ' '); i >= 0 {
		b = b[:i]
	}
	id, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return -1
	}
	return id
}

func currentThreadGoroutineID() int64 {
	return coroutineCreator(goroutineID())
}

func coroutineCreator(id int64) int64 {
	buf := make([]byte, 8<<10)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	marker := []byte("\ncreated by iter.Pull")
	i := bytes.Index(buf, marker)
	if i < 0 {
		return id
	}
	rest := buf[i+len(marker):]
	if j := bytes.IndexByte(rest, '\n'); j >= 0 {
		rest = rest[:j]
	}
	k := bytes.LastIndex(rest, []byte(" in goroutine "))
	if k < 0 {
		return id
	}
	creator, err := strconv.ParseInt(string(bytes.TrimSpace(rest[k+len(" in goroutine "):])), 10, 64)
	if err != nil {
		return id
	}
	return creator
}
