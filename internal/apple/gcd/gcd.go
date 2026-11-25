//go:build darwin

// Package gcd binds Grand Central Dispatch and the libSystem thread primitives the Apple platform
// packages use, on purego without cgo.
package gcd

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/purego"
)

// LibSystemPath is the install path of libSystem, which exports libdispatch and pthreads.
const LibSystemPath = "/usr/lib/libSystem.B.dylib"

// Queue is a dispatch_queue_t.
type Queue uintptr

// Time is a dispatch_time_t.
type Time uint64

// Function is a C function pointer taking one context argument (dispatch_function_t).
type Function uintptr

// QueuePriority is a dispatch_queue_priority_t for GlobalQueue.
type QueuePriority int

const (
	// QueuePriorityHigh is DISPATCH_QUEUE_PRIORITY_HIGH.
	QueuePriorityHigh QueuePriority = 2
	// QueuePriorityDefault is DISPATCH_QUEUE_PRIORITY_DEFAULT.
	QueuePriorityDefault QueuePriority = 0
	// QueuePriorityLow is DISPATCH_QUEUE_PRIORITY_LOW.
	QueuePriorityLow QueuePriority = -2
	// QueuePriorityBackground is DISPATCH_QUEUE_PRIORITY_BACKGROUND.
	QueuePriorityBackground QueuePriority = -1 << 15
)

// QoSClass is a qos_class_t for GlobalQueueQoS.
type QoSClass uint32

const (
	// QoSClassUserInteractive is QOS_CLASS_USER_INTERACTIVE.
	QoSClassUserInteractive QoSClass = 0x21
	// QoSClassUserInitiated is QOS_CLASS_USER_INITIATED.
	QoSClassUserInitiated QoSClass = 0x19
	// QoSClassDefault is QOS_CLASS_DEFAULT.
	QoSClassDefault QoSClass = 0x15
	// QoSClassUtility is QOS_CLASS_UTILITY.
	QoSClassUtility QoSClass = 0x11
	// QoSClassBackground is QOS_CLASS_BACKGROUND.
	QoSClassBackground QoSClass = 0x09
	// QoSClassUnspecified is QOS_CLASS_UNSPECIFIED.
	QoSClassUnspecified QoSClass = 0x00
)

const (
	// TimeNow is DISPATCH_TIME_NOW.
	TimeNow Time = 0
	// TimeForever is DISPATCH_TIME_FOREVER.
	TimeForever Time = ^Time(0)
)

var (
	loadOnce sync.Once
	loadErr  error
	lib      uintptr

	mainQueueAddr uintptr

	dispatch_get_global_queue func(identifier int, flags uintptr) Queue
	dispatch_async_f          func(queue Queue, context uintptr, work Function)
	dispatch_after_f          func(when Time, queue Queue, context uintptr, work Function)
	dispatch_time             func(when Time, delta int64) Time

	pthread_self           func() uintptr
	pthread_main_np        func() int32
	pthread_threadid_np    func(thread uintptr, id *uint64) int32
	pthread_create         func(thread *uintptr, attr uintptr, start uintptr, arg uintptr) int32
	pthread_detach         func(thread uintptr) int32
	pthread_mach_thread_np func(thread uintptr) uint32
)

// Load opens libSystem and resolves the dispatch and pthread symbols. It is safe to call
// repeatedly and from any thread.
func Load() error {
	loadOnce.Do(func() {
		h, err := purego.Dlopen(LibSystemPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("gcd: open %s: %w", LibSystemPath, err)
			return
		}
		lib = h
		for _, b := range []struct {
			fptr any
			name string
		}{
			{&dispatch_get_global_queue, "dispatch_get_global_queue"},
			{&dispatch_async_f, "dispatch_async_f"},
			{&dispatch_after_f, "dispatch_after_f"},
			{&dispatch_time, "dispatch_time"},
			{&pthread_self, "pthread_self"},
			{&pthread_main_np, "pthread_main_np"},
			{&pthread_threadid_np, "pthread_threadid_np"},
			{&pthread_create, "pthread_create"},
			{&pthread_detach, "pthread_detach"},
			{&pthread_mach_thread_np, "pthread_mach_thread_np"},
		} {
			addr, err := purego.Dlsym(h, b.name)
			if err != nil {
				loadErr = fmt.Errorf("gcd: %w", err)
				return
			}
			purego.RegisterFunc(b.fptr, addr)
		}
		if mainQueueAddr, err = purego.Dlsym(h, "_dispatch_main_q"); err != nil {
			loadErr = fmt.Errorf("gcd: %w", err)
		}
	})
	return loadErr
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// LibSystem returns the dlopen handle of libSystem.
func LibSystem() uintptr {
	mustLoad()
	return lib
}

// MainQueue returns the main queue, the address of _dispatch_main_q (dispatch_get_main_queue).
func MainQueue() Queue {
	mustLoad()
	return Queue(mainQueueAddr)
}

// GlobalQueue returns the global concurrent queue of priority.
func GlobalQueue(priority QueuePriority) Queue {
	mustLoad()
	return dispatch_get_global_queue(int(priority), 0)
}

// GlobalQueueQoS returns the global concurrent queue of the quality-of-service class qos.
func GlobalQueueQoS(qos QoSClass) Queue {
	mustLoad()
	return dispatch_get_global_queue(int(qos), 0)
}

// ExecAsyncF submits work(context) to q (dispatch_async_f).
func (q Queue) ExecAsyncF(context uintptr, work Function) {
	mustLoad()
	dispatch_async_f(q, context, work)
}

// ExecAfterF submits work(context) to q at when (dispatch_after_f).
func ExecAfterF(when Time, q Queue, context uintptr, work Function) {
	mustLoad()
	dispatch_after_f(when, q, context, work)
}

// Time returns t advanced by delta nanoseconds (dispatch_time).
func (t Time) Time(delta int64) Time {
	mustLoad()
	return dispatch_time(t, delta)
}

// After returns the dispatch time d from now.
func After(d time.Duration) Time {
	return TimeNow.Time(d.Nanoseconds())
}

// NewFunction returns a C function pointer that calls fn with its context argument. purego caps a
// process at 2000 callbacks, so create functions once, not per call.
func NewFunction(fn func(context uintptr)) Function {
	return Function(purego.NewCallback(fn))
}

// Handles maps small integer keys to Go values so a value can travel through a C context pointer
// without handing C a Go pointer. Key 0 is never used.
type Handles[T any] struct {
	mu   sync.Mutex
	next atomic.Uintptr
	m    map[uintptr]T
}

// Insert stores v and returns its key.
func (h *Handles[T]) Insert(v T) uintptr {
	key := h.next.Add(1)
	h.mu.Lock()
	if h.m == nil {
		h.m = map[uintptr]T{}
	}
	h.m[key] = v
	h.mu.Unlock()
	return key
}

// Take removes and returns the value stored under key.
func (h *Handles[T]) Take(key uintptr) (T, bool) {
	h.mu.Lock()
	v, ok := h.m[key]
	delete(h.m, key)
	h.mu.Unlock()
	return v, ok
}

// Get returns the value stored under key without removing it.
func (h *Handles[T]) Get(key uintptr) (T, bool) {
	h.mu.Lock()
	v, ok := h.m[key]
	h.mu.Unlock()
	return v, ok
}

// Len returns the number of stored values.
func (h *Handles[T]) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.m)
}

var (
	closures      Handles[func()]
	closureFnOnce sync.Once
	closureFn     Function
	threadFnOnce  sync.Once
	threadFn      uintptr
	threadBodies  Handles[func()]
)

func closureTrampoline() Function {
	closureFnOnce.Do(func() {
		closureFn = NewFunction(func(context uintptr) {
			if f, ok := closures.Take(context); ok {
				f()
			}
		})
	})
	return closureFn
}

// Async submits f to q.
func (q Queue) Async(f func()) {
	q.ExecAsyncF(closures.Insert(f), closureTrampoline())
}

// AsyncAfter submits f to q at when.
func (q Queue) AsyncAfter(when Time, f func()) {
	ExecAfterF(when, q, closures.Insert(f), closureTrampoline())
}

// IsMainThread reports whether the calling OS thread is the process's main thread (pthread_main_np).
func IsMainThread() bool {
	mustLoad()
	return pthread_main_np() != 0
}

// CurrentThreadID returns the system-wide id of the calling OS thread (pthread_threadid_np).
func CurrentThreadID() uint64 {
	mustLoad()
	var id uint64
	pthread_threadid_np(0, &id)
	return id
}

// CurrentMachThread returns the mach port of the calling OS thread (pthread_mach_thread_np).
func CurrentMachThread() uint32 {
	mustLoad()
	return pthread_mach_thread_np(pthread_self())
}

// SpawnThread runs f on a new detached OS thread (std::thread::spawn). The thread exits when f
// returns.
func SpawnThread(f func()) error {
	mustLoad()
	threadFnOnce.Do(func() {
		threadFn = purego.NewCallback(func(arg uintptr) uintptr {
			if body, ok := threadBodies.Take(arg); ok {
				body()
			}
			return 0
		})
	})
	key := threadBodies.Insert(f)
	var thread uintptr
	if rc := pthread_create(&thread, 0, threadFn, key); rc != 0 {
		threadBodies.Take(key)
		return fmt.Errorf("gcd: pthread_create: errno %d", rc)
	}
	pthread_detach(thread)
	return nil
}
