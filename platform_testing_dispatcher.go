package gpui

import (
	"os"
	"sync/atomic"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

// TestDispatcher provides deterministic async execution for tests by delegating to the scheduler
// package's TestScheduler. Use Scheduler for clock, rng and parking control.
type TestDispatcher struct {
	PlatformDispatcherDefaults
	sessionID       scheduler.SessionID
	scheduler       *scheduler.TestScheduler
	numCPUsOverride *atomic.Uint64
}

// NewTestDispatcher returns a dispatcher over a new TestScheduler seeded with seed.
func NewTestDispatcher(seed uint64) *TestDispatcher {
	pending := os.Getenv("PENDING_TRACES")
	s := scheduler.NewTestScheduler(scheduler.TestSchedulerConfig{
		Seed:                 seed,
		RandomizeOrder:       true,
		AllowParking:         false,
		CapturePendingTraces: pending == "1" || pending == "true",
		TimeoutTicks:         scheduler.TickRange{Start: 0, End: 1000},
	})
	return TestDispatcherFromScheduler(s)
}

// TestDispatcherFromScheduler returns a dispatcher with a fresh session on s.
func TestDispatcherFromScheduler(s *scheduler.TestScheduler) *TestDispatcher {
	d := &TestDispatcher{
		sessionID:       s.AllocateSessionID(),
		scheduler:       s,
		numCPUsOverride: new(atomic.Uint64),
	}
	BindPlatformDispatcherDefaults(&d.PlatformDispatcherDefaults, d)
	return d
}

// Scheduler returns the underlying TestScheduler.
func (d *TestDispatcher) Scheduler() *scheduler.TestScheduler {
	return d.scheduler
}

// SessionID returns the dispatcher's foreground session.
func (d *TestDispatcher) SessionID() scheduler.SessionID {
	return d.sessionID
}

// DrainTasks drops every queued task without running it.
func (d *TestDispatcher) DrainTasks() {
	d.scheduler.DrainTasks()
}

// AdvanceClock moves the virtual clock forward by by.
func (d *TestDispatcher) AdvanceClock(by time.Duration) {
	d.scheduler.AdvanceClock(by)
}

// AdvanceClockToNextTimer jumps the virtual clock to the next timer, reporting whether there was one.
func (d *TestDispatcher) AdvanceClockToNextTimer() bool {
	return d.scheduler.AdvanceClockToNextTimer()
}

// SimulateRandomDelay returns a future that yields a random number of times.
func (d *TestDispatcher) SimulateRandomDelay() *scheduler.Yield {
	return d.scheduler.YieldRandom()
}

// Tick runs one task, only background ones when backgroundOnly is set.
func (d *TestDispatcher) Tick(backgroundOnly bool) bool {
	if backgroundOnly {
		return d.scheduler.TickBackgroundOnly()
	}
	return d.scheduler.Tick()
}

// RunUntilParked runs tasks until none is runnable.
func (d *TestDispatcher) RunUntilParked() {
	for d.Tick(false) {
	}
}

// AllowParking lets blocking waits park instead of panicking.
func (d *TestDispatcher) AllowParking() {
	d.scheduler.AllowParking()
}

// ForbidParking undoes AllowParking.
func (d *TestDispatcher) ForbidParking() {
	d.scheduler.ForbidParking()
}

// SetNumCPUs overrides BackgroundExecutor.NumCPUs; 0 means no override (the default of 4).
func (d *TestDispatcher) SetNumCPUs(count int) {
	d.numCPUsOverride.Store(uint64(count))
}

// NumCPUsOverride returns the overridden CPU count, if one is set.
func (d *TestDispatcher) NumCPUsOverride() (int, bool) {
	n := d.numCPUsOverride.Load()
	if n == 0 {
		return 0, false
	}
	return int(n), true
}

// Clone returns a dispatcher on the same scheduler with a fresh session.
func (d *TestDispatcher) Clone() *TestDispatcher {
	c := &TestDispatcher{
		sessionID:       d.scheduler.AllocateSessionID(),
		scheduler:       d.scheduler,
		numCPUsOverride: d.numCPUsOverride,
	}
	BindPlatformDispatcherDefaults(&c.PlatformDispatcherDefaults, c)
	return c
}

// IsMainThread reports whether the scheduler is running a foreground task, or none.
func (d *TestDispatcher) IsMainThread() bool {
	return d.scheduler.IsMainThread()
}

// Now returns the virtual clock's instant.
func (d *TestDispatcher) Now() scheduler.Instant {
	return d.scheduler.Clock().Now()
}

// Dispatch queues a background runnable.
func (d *TestDispatcher) Dispatch(runnable RunnableVariant, priority Priority) {
	d.scheduler.ScheduleBackgroundWithPriority(runnable, priority)
}

// DispatchOnMainThread queues a runnable on the dispatcher's session.
func (d *TestDispatcher) DispatchOnMainThread(runnable RunnableVariant, priority Priority) {
	d.scheduler.ScheduleLocal(d.sessionID, runnable)
}

// DispatchAfter panics: tests use the scheduler's own timers.
func (d *TestDispatcher) DispatchAfter(duration time.Duration, runnable RunnableVariant) {
	panic("dispatch_after should not be called in tests. " +
		"Use BackgroundExecutor::timer() which uses the scheduler's native timer.")
}

// AsTest returns d.
func (d *TestDispatcher) AsTest() *TestDispatcher {
	return d
}

// SpawnRealtime runs f on a new goroutine.
func (d *TestDispatcher) SpawnRealtime(f func()) {
	go func() {
		f()
	}()
}
