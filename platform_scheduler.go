package gpui

import (
	"sync/atomic"
	"time"
	"weak"

	"github.com/megascan/gpui-go/scheduler"
)

// PlatformScheduler is the production scheduler.Scheduler over a PlatformDispatcher, so the
// scheduler package's executors use the platform's native dispatch.
type PlatformScheduler struct {
	dispatcher    PlatformDispatcher
	clock         *platformClock
	nextSessionID atomic.Uint32
}

// NewPlatformScheduler returns a scheduler that dispatches through dispatcher.
func NewPlatformScheduler(dispatcher PlatformDispatcher) *PlatformScheduler {
	return &PlatformScheduler{
		dispatcher: dispatcher,
		clock:      &platformClock{dispatcher: dispatcher},
	}
}

// ForegroundExecutor returns a local executor on a fresh session whose tasks run on the main thread.
func (s *PlatformScheduler) ForegroundExecutor() scheduler.LocalExecutor {
	sessionID := s.newSessionID()
	weakScheduler := weak.Make(s)
	return scheduler.NewLocalExecutor(sessionID, s, func(runnable *scheduler.Runnable) {
		if strong := weakScheduler.Value(); strong != nil {
			strong.ScheduleLocal(sessionID, runnable)
		}
	})
}

func (s *PlatformScheduler) newSessionID() scheduler.SessionID {
	return scheduler.NewSessionID(uint16(s.nextSessionID.Add(1) - 1))
}

// Block polls future on the calling goroutine, parking between wake-ups, until it completes or
// timeout elapses.
func (s *PlatformScheduler) Block(sessionID *scheduler.SessionID, future scheduler.Future[struct{}], timeout *time.Duration) bool {
	var deadline scheduler.Instant
	hasDeadline := timeout != nil
	if hasDeadline {
		deadline = scheduler.InstantNow().Add(*timeout)
	}
	unparked := make(chan struct{}, 1)
	waker := scheduler.WakerFn(func() {
		select {
		case unparked <- struct{}{}:
		default:
		}
	})
	cx := scheduler.NewContext(waker)
	if _, ok := future.Poll(cx); ok {
		return true
	}

	parkDeadline := func(deadline scheduler.Instant) bool {
		timerGuard := s.dispatcher.IncreaseTimerResolution()
		defer timerGuard.Run()
		wait := deadline.SaturatingDurationSince(scheduler.InstantNow())
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-unparked:
			return true
		case <-timer.C:
			return false
		}
	}

	for {
		if hasDeadline {
			if !parkDeadline(deadline) && !scheduler.InstantNow().Before(deadline) {
				return false
			}
		} else {
			<-unparked
		}
		if _, ok := future.Poll(cx); ok {
			return true
		}
	}
}

// ScheduleLocal dispatches runnable to the main thread.
func (s *PlatformScheduler) ScheduleLocal(sessionID scheduler.SessionID, runnable *scheduler.Runnable) {
	if profilerFeatureEnabled {
		currentForegroundRunnableCounter().queued()
	}
	s.dispatcher.DispatchOnMainThread(runnable, scheduler.DefaultPriority())
}

// ScheduleBackgroundWithPriority dispatches runnable to a background thread.
func (s *PlatformScheduler) ScheduleBackgroundWithPriority(runnable *scheduler.Runnable, priority scheduler.Priority) {
	s.dispatcher.Dispatch(runnable, priority)
}

// ScheduleBackground dispatches runnable to a background thread with the default priority.
func (s *PlatformScheduler) ScheduleBackground(runnable *scheduler.Runnable) {
	s.ScheduleBackgroundWithPriority(runnable, scheduler.DefaultPriority())
}

// SpawnRealtime runs f on the dispatcher's realtime thread.
func (s *PlatformScheduler) SpawnRealtime(f func()) {
	s.dispatcher.SpawnRealtime(f)
}

type timerFuture struct {
	tx *scheduler.OneshotSender[struct{}]
}

func (f *timerFuture) Poll(cx *scheduler.Context) (struct{}, bool) {
	_ = f.tx.Send(struct{}{})
	return struct{}{}, true
}

func (f *timerFuture) Cancel() {
	f.tx.Close()
}

// Timer returns a timer that fires when the dispatcher runs its runnable after duration.
func (s *PlatformScheduler) Timer(duration time.Duration) *scheduler.Timer {
	tx, rx := scheduler.Oneshot[struct{}]()
	dispatcher := s.dispatcher

	location := scheduler.CallerLocation(1)
	runnable, task := scheduler.SpawnAsyncTask(
		scheduler.RunnableMeta{Location: location, Spawned: scheduler.SpawnTime{Instant: scheduler.InstantNow()}},
		scheduler.Future[struct{}](&timerFuture{tx: tx}),
		func(runnable *scheduler.Runnable) {
			dispatcher.DispatchAfter(duration, runnable)
		},
	)
	runnable.Schedule()
	task.Cancel()

	return scheduler.NewTimer(rx)
}

// Clock returns the dispatcher-backed clock.
func (s *PlatformScheduler) Clock() scheduler.Clock {
	return s.clock
}

// SpawnDedicated runs f on a new goroutine pinned to its own session.
func (s *PlatformScheduler) SpawnDedicated(f func(executor scheduler.LocalExecutor, s *scheduler.Scope) (any, error)) *Task[any] {
	sessionID := s.newSessionID()
	return scheduler.SpawnDedicatedThread(sessionID, s, f)
}

// AsTest returns nil.
func (s *PlatformScheduler) AsTest() *scheduler.TestScheduler {
	return nil
}

type platformClock struct {
	dispatcher PlatformDispatcher
}

func (c *platformClock) UTCNow() time.Time {
	return time.Now().UTC()
}

func (c *platformClock) Now() scheduler.Instant {
	return c.dispatcher.Now()
}
