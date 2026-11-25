package scheduler

import (
	"sort"
	"sync"
	"testing"
	"time"
)

type fifoTimer struct {
	at Instant
	tx *OneshotSender[struct{}]
}

type fifoScheduler struct {
	mu          sync.Mutex
	queue       []*Runnable
	timers      []fifoTimer
	closed      bool
	clock       *TestClock
	nextSession uint16
	wake        *signal
	threaded    bool
}

func newFIFOScheduler() *fifoScheduler {
	return &fifoScheduler{clock: NewTestClock(), wake: newSignal()}
}

func newThreadedScheduler() *fifoScheduler {
	s := newFIFOScheduler()
	s.threaded = true
	return s
}

func (s *fifoScheduler) push(r *Runnable) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		r.Close()
		return
	}
	if s.threaded {
		s.mu.Unlock()
		go r.Run()
		return
	}
	s.queue = append(s.queue, r)
	s.mu.Unlock()
	s.wake.notify()
}

func (s *fifoScheduler) tick() bool {
	s.mu.Lock()
	if len(s.queue) == 0 {
		s.mu.Unlock()
		return false
	}
	r := s.queue[0]
	s.queue = s.queue[1:]
	s.mu.Unlock()
	r.Run()
	return true
}

func (s *fifoScheduler) run() {
	for s.tick() {
	}
}

func (s *fifoScheduler) fireNextTimer() bool {
	s.mu.Lock()
	if len(s.timers) == 0 {
		s.mu.Unlock()
		return false
	}
	t := s.timers[0]
	s.timers = s.timers[1:]
	s.mu.Unlock()
	if d := t.at.Sub(s.clock.Now()); d > 0 {
		s.clock.Advance(d)
	}
	t.tx.Close()
	return true
}

func (s *fifoScheduler) close() {
	s.mu.Lock()
	s.closed = true
	queue := s.queue
	s.queue = nil
	timers := s.timers
	s.timers = nil
	s.mu.Unlock()
	for _, r := range queue {
		r.Close()
	}
	for _, t := range timers {
		t.tx.Close()
	}
}

func (s *fifoScheduler) pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue)
}

func (s *fifoScheduler) Block(sessionID *SessionID, future Future[struct{}], timeout *time.Duration) bool {
	sig := newSignal()
	cx := NewContext(WakerFn(sig.notify))
	for {
		if _, ok := future.Poll(cx); ok {
			return true
		}
		if s.tick() {
			continue
		}
		select {
		case <-sig.ch:
			continue
		default:
		}
		if timeout != nil {
			return false
		}
		if s.fireNextTimer() {
			continue
		}
		select {
		case <-sig.ch:
		case <-s.wake.ch:
		}
	}
}

func (s *fifoScheduler) ScheduleLocal(sessionID SessionID, runnable *Runnable) {
	s.push(runnable)
}

func (s *fifoScheduler) ScheduleBackgroundWithPriority(runnable *Runnable, priority Priority) {
	s.push(runnable)
}

func (s *fifoScheduler) ScheduleBackground(runnable *Runnable) {
	s.ScheduleBackgroundWithPriority(runnable, DefaultPriority())
}

func (s *fifoScheduler) SpawnRealtime(f func()) {
	go f()
}

func (s *fifoScheduler) Timer(timeout time.Duration) *Timer {
	tx, rx := Oneshot[struct{}]()
	s.mu.Lock()
	s.timers = append(s.timers, fifoTimer{at: s.clock.Now().Add(timeout), tx: tx})
	sort.SliceStable(s.timers, func(i, j int) bool { return s.timers[i].at.Before(s.timers[j].at) })
	s.mu.Unlock()
	return NewTimer(rx)
}

func (s *fifoScheduler) Clock() Clock {
	return s.clock
}

func (s *fifoScheduler) allocateSessionID() SessionID {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSession++
	return SessionID(s.nextSession)
}

func (s *fifoScheduler) SpawnDedicated(f func(executor LocalExecutor, s *Scope) (any, error)) *Task[any] {
	return SpawnDedicatedThread(s.allocateSessionID(), s, f)
}

func (s *fifoScheduler) AsTest() *TestScheduler {
	return nil
}

func (s *fifoScheduler) foreground() LocalExecutor {
	id := s.allocateSessionID()
	return NewLocalExecutor(id, s, func(r *Runnable) { s.ScheduleLocal(id, r) })
}

func (s *fifoScheduler) background() BackgroundExecutor {
	return NewBackgroundExecutor(s)
}

func awaitTask[T any](t *testing.T, s *fifoScheduler, task *Task[T]) (T, error) {
	t.Helper()
	r := s.foreground().BlockOn(Future[Result[T]](task))
	return r.Value, r.Err
}
