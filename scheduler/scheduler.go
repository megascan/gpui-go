// Package scheduler ports Zed's scheduler crate: the Scheduler contract, executors and tasks.
// Tasks are coroutines built on iter.Pull; one poll resumes a body until it parks in Await,
// Select, Join, a channel or a timer.
package scheduler

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"
	"unsafe"
	"weak"
)

var testSchedulerCreatedFlag atomic.Bool

// TestSchedulerCreated reports whether this process has created a TestScheduler. Code with an
// executor at hand should prefer BackgroundExecutor.IsTest.
func TestSchedulerCreated() bool {
	return testSchedulerCreatedFlag.Load()
}

// Priority is the scheduling priority of a background task.
type Priority uint8

const (
	// PriorityRealtimeAudio runs the task on a dedicated thread; only use it for audio.
	PriorityRealtimeAudio Priority = iota
	// PriorityHigh is for tasks critical to responsiveness.
	PriorityHigh
	// PriorityMedium suits most tasks; it is the default.
	PriorityMedium
	// PriorityLow is for work that can be deprioritized.
	PriorityLow
)

// DefaultPriority returns PriorityMedium, Priority's default.
func DefaultPriority() Priority {
	return PriorityMedium
}

// Weight returns the relative probability weight schedulers use to pick a task of this priority.
func (p Priority) Weight() uint32 {
	switch p {
	case PriorityHigh:
		return 60
	case PriorityMedium:
		return 30
	case PriorityLow:
		return 10
	default:
		return 0
	}
}

// String returns the variant name.
func (p Priority) String() string {
	switch p {
	case PriorityRealtimeAudio:
		return "RealtimeAudio"
	case PriorityHigh:
		return "High"
	case PriorityMedium:
		return "Medium"
	case PriorityLow:
		return "Low"
	default:
		return fmt.Sprintf("Priority(%d)", uint8(p))
	}
}

// SpawnTime is the moment a task was spawned.
type SpawnTime struct {
	Instant
}

// Location is a source position captured at a spawn site (std::panic::Location).
type Location struct {
	pc uintptr
}

// CallerLocation returns a call site on the current stack: skip 0 is where CallerLocation is
// called, skip 1 is where the function calling it was called, and so on.
func CallerLocation(skip int) Location {
	var pcs [1]uintptr
	if runtime.Callers(skip+2, pcs[:]) == 0 {
		return Location{}
	}
	return Location{pc: pcs[0]}
}

func (l Location) frame() runtime.Frame {
	if l.pc == 0 {
		return runtime.Frame{}
	}
	f, _ := runtime.CallersFrames([]uintptr{l.pc}).Next()
	return f
}

// File returns the source file, or "" for an unknown location.
func (l Location) File() string {
	return l.frame().File
}

// Line returns the source line, or 0 for an unknown location.
func (l Location) Line() int {
	return l.frame().Line
}

// Function returns the qualified name of the enclosing function.
func (l Location) Function() string {
	return l.frame().Function
}

// String formats the location as file:line.
func (l Location) String() string {
	f := l.frame()
	if f.File == "" {
		return "<unknown>"
	}
	return fmt.Sprintf("%s:%d", f.File, f.Line)
}

// RunnableMeta is debugging and profiling metadata attached to every runnable.
type RunnableMeta struct {
	Location Location
	Spawned  SpawnTime
}

// NewRunnableMetaWithCallersLocation returns metadata for the caller's call site, spawned now.
func NewRunnableMetaWithCallersLocation() RunnableMeta {
	return newRunnableMeta(CallerLocation(1))
}

func newRunnableMeta(location Location) RunnableMeta {
	return RunnableMeta{Location: location, Spawned: SpawnTime{InstantNow()}}
}

// Scheduler decides where and when runnables run.
type Scheduler interface {
	// Block polls future until it completes or timeout elapses, returning whether it completed.
	// The caller keeps ownership of future and may keep polling it after a timeout. A nil
	// sessionID or timeout means none.
	Block(sessionID *SessionID, future Future[struct{}], timeout *time.Duration) bool
	// ScheduleLocal queues runnable on the session-pinned queue of sessionID.
	ScheduleLocal(sessionID SessionID, runnable *Runnable)
	// ScheduleBackgroundWithPriority queues a background runnable.
	ScheduleBackgroundWithPriority(runnable *Runnable, priority Priority)
	// SpawnRealtime runs f on a dedicated realtime thread for audio processing.
	SpawnRealtime(f func())
	// ScheduleBackground queues a background runnable with DefaultPriority; implementations
	// call ScheduleBackgroundWithPriority(runnable, DefaultPriority()).
	ScheduleBackground(runnable *Runnable)
	// Timer returns a timer that fires after timeout.
	Timer(timeout time.Duration) *Timer
	// Clock returns the scheduler's clock.
	Clock() Clock
	// SpawnDedicated runs f on a fresh session pinned to its own LocalExecutor.
	SpawnDedicated(f func(executor LocalExecutor, s *Scope) (any, error)) *Task[any]
	// AsTest returns the scheduler as a TestScheduler, or nil.
	AsTest() *TestScheduler
}

// SpawnDedicatedThread runs f on a fresh goroutine that is exclusive to the returned task and
// everything spawned on the LocalExecutor it is given. It never blocks: the returned task
// resolves to the root task once the dedicated goroutine spawned it. Dropping the task cancels
// the root; detached children keep running. sessionID must be distinct from every other live
// session on scheduler.
func SpawnDedicatedThread[T any](sessionID SessionID, scheduler Scheduler, f func(executor LocalExecutor, s *Scope) (T, error)) *Task[T] {
	return spawnDedicatedThreadAt(CallerLocation(1), sessionID, scheduler, f)
}

type dedicatedSender struct {
	tx *ChanSender[*Runnable]
}

func spawnDedicatedThreadAt[T any](location Location, sessionID SessionID, scheduler Scheduler, f func(executor LocalExecutor, s *Scope) (T, error)) *Task[T] {
	task, delivery := taskRendezvous[T]()
	threadBody := func() {
		runnableReceiver := startDedicatedSession(location, sessionID, scheduler, f, delivery)
		for {
			runnable, ok := runnableReceiver.RecvBlocking()
			if !ok {
				break
			}
			if !runIsolated(runnable, fmt.Sprintf("spawn_dedicated session %v", sessionID)) {
				closeRunQueue(runnableReceiver)
				return
			}
		}
	}
	spawnDedicatedOSThread(sessionID, threadBody)
	return task
}

func startDedicatedSession[T any](location Location, sessionID SessionID, scheduler Scheduler, f func(executor LocalExecutor, s *Scope) (T, error), delivery *rendezvousSender[T]) *ChanReceiver[*Runnable] {
	runnableSender, runnableReceiver := Chan[*Runnable](0)
	sender := &dedicatedSender{tx: runnableSender}
	runtime.AddCleanup(sender, func(tx *ChanSender[*Runnable]) { tx.Close() }, runnableSender)
	dispatch := func(runnable *Runnable) {
		if sender.tx.TrySend(runnable) != nil {
			runnable.Close()
		}
	}
	executor := NewLocalExecutor(sessionID, scheduler, dispatch)
	rootTask := executor.spawnAt(location, func(s *Scope) (T, error) {
		return f(executor, s)
	})
	delivery.deliver(rootTask)
	return runnableReceiver
}

func runIsolated(runnable *Runnable, thread string) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "thread '%s' panicked: %v\n%s", thread, r, debug.Stack())
			ok = false
		}
	}()
	runnable.Run()
	return true
}

func closeRunQueue(rx *ChanReceiver[*Runnable]) {
	rx.Close()
	for {
		runnable, ok, _ := rx.TryRecv()
		if !ok {
			return
		}
		runnable.Close()
	}
}

func spawnDedicatedOSThread(sessionID SessionID, threadBody func()) {
	go threadBody()
}

// SessionID identifies one session-pinned queue.
type SessionID uint16

// NewSessionID returns the session with the given id.
func NewSessionID(id uint16) SessionID {
	return SessionID(id)
}

// String formats the id as upstream's Debug does.
func (id SessionID) String() string {
	return fmt.Sprintf("SessionId(%d)", uint16(id))
}

// Timer is a future that resolves when its sender fires or is dropped.
type Timer struct {
	rx *OneshotReceiver[struct{}]
}

// NewTimer returns a timer that resolves when rx's sender sends or is closed.
func NewTimer(rx *OneshotReceiver[struct{}]) *Timer {
	return &Timer{rx: rx}
}

// Poll resolves once the timer fired.
func (t *Timer) Poll(cx *Context) (struct{}, bool) {
	_, ok := t.rx.Poll(cx)
	return struct{}{}, ok
}

// Await parks the coroutine body until the timer fires.
func (t *Timer) Await(cx Awaiter) {
	Await[struct{}](cx, t)
}

// Cancel drops the timer.
func (t *Timer) Cancel() {
	t.rx.Close()
}

type weakScheduler struct {
	strong Scheduler
	typ    reflect.Type
	ptr    weak.Pointer[byte]
}

func downgradeScheduler(s Scheduler) weakScheduler {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Type().Elem().Size() == 0 {
		return weakScheduler{strong: s}
	}
	return weakScheduler{typ: v.Type().Elem(), ptr: weak.Make((*byte)(v.UnsafePointer()))}
}

func (w weakScheduler) upgrade() Scheduler {
	if w.strong != nil {
		return w.strong
	}
	if w.typ == nil {
		return nil
	}
	p := w.ptr.Value()
	if p == nil {
		return nil
	}
	s, _ := reflect.NewAt(w.typ, unsafe.Pointer(p)).Interface().(Scheduler)
	return s
}
