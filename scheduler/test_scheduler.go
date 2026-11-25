package scheduler

import (
	"bytes"
	"fmt"
	"math/bits"
	"math/rand/v2"
	"os"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"weak"
)

const pendingTracesVarName = "PENDING_TRACES"

// TestScheduler is a deterministic Scheduler for tests: one goroutine drives every runnable,
// picking the next one with a seeded RNG, and timers run on a virtual clock.
type TestScheduler struct {
	clock *TestClock
	rng   *SharedRng
	state *testSchedulerShared
}

type testSchedulerShared struct {
	mu        sync.Mutex
	state     schedulerState
	owner     int64
	unparker  *signal
	coroutine sync.Map
}

// TestSchedulerOnce runs f once with the default configuration (seed 0).
func TestSchedulerOnce[R any](f func(scheduler *TestScheduler, s *Scope) R) R {
	return testSchedulerWithSeed(0, f)
}

// TestSchedulerMany runs f with sequential seeds starting at SEED (default 0), ITERATIONS times
// (default defaultIterations), and returns the outputs in order.
func TestSchedulerMany[R any](defaultIterations int, f func(scheduler *TestScheduler, s *Scope) R) []R {
	numIterations := defaultIterations
	if v, ok := os.LookupEnv("ITERATIONS"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			panic(err)
		}
		numIterations = n
	}
	var seed uint64
	if v, ok := os.LookupEnv("SEED"); ok {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(err)
		}
		seed = n
	}
	_, nonInteractive := os.LookupEnv("SCHEDULER_NONINTERACTIVE")
	interactive := !nonInteractive

	results := make([]R, 0, numIterations)
	for s := seed; s < seed+uint64(numIterations); s++ {
		if interactive {
			fmt.Fprintf(os.Stderr, "Running seed: %d\n", s)
		}
		results = append(results, runSeedReportingFailure(s, f))
	}
	return results
}

func runSeedReportingFailure[R any](seed uint64, f func(scheduler *TestScheduler, s *Scope) R) R {
	ok := false
	defer func() {
		if !ok {
			fmt.Fprintf(os.Stderr, "\x1b[31mFailing Seed: %d\x1b[0m\n", seed)
		}
	}()
	r := testSchedulerWithSeed(seed, f)
	ok = true
	return r
}

func testSchedulerWithSeed[R any](seed uint64, f func(scheduler *TestScheduler, s *Scope) R) R {
	scheduler := NewTestScheduler(TestSchedulerConfigWithSeed(seed))
	var output R
	hasOutput := false
	future := Async(func(s *Scope) struct{} {
		output = f(scheduler, s)
		hasOutput = true
		return struct{}{}
	})
	scheduler.blockUntil(nil, future, nil)
	if !hasOutput {
		panic("test future did not complete")
	}
	scheduler.Run()
	return output
}

func (s *TestScheduler) blockUntil(sessionID *SessionID, future Future[struct{}], timeout *time.Duration) bool {
	sh := s.state
	if sessionID != nil {
		sh.mu.Lock()
		sh.state.blockedSessions = append(sh.state.blockedSessions, *sessionID)
		sh.mu.Unlock()
	}

	var deadline *Instant
	if timeout != nil {
		d := InstantNow().Add(*timeout)
		deadline = &d
	}
	awoken := &atomic.Bool{}
	waker := &tracingWaker{awoken: awoken, shared: sh}
	maxTicks := int(^uint(0) >> 1)
	if timeout != nil {
		sh.mu.Lock()
		ticks := sh.state.timeoutTicks
		sh.mu.Unlock()
		maxTicks = s.rng.randomRangeInclusive(ticks.Start, ticks.End)
	}
	cx := NewContext(waker)

	completed := false
	for range maxTicks {
		if _, ok := future.Poll(cx); ok {
			completed = true
			break
		}

		var stepped *bool
		for Random[bool](s.rng) {
			if stepped == nil {
				f := false
				stepped = &f
			}
			if s.step() {
				*stepped = true
			} else {
				break
			}
		}

		didStep := stepped == nil || *stepped
		wasAwoken := awoken.Swap(false)
		if !didStep && !wasAwoken {
			sh.mu.Lock()
			parkingAllowed := sh.state.allowParking
			sh.mu.Unlock()
			advancedToTimer := !parkingAllowed && s.AdvanceClockToNextTimer()
			if !advancedToTimer && !s.park(deadline) {
				break
			}
		}
	}

	if sessionID != nil {
		sh.mu.Lock()
		sh.state.blockedSessions = sh.state.blockedSessions[:len(sh.state.blockedSessions)-1]
		sh.mu.Unlock()
	}

	return completed
}

// NewTestScheduler returns a scheduler driven by the calling goroutine.
func NewTestScheduler(config TestSchedulerConfig) *TestScheduler {
	testSchedulerCreatedFlag.Store(true)
	return &TestScheduler{
		rng: newSharedRng(config.Seed),
		state: &testSchedulerShared{
			state: schedulerState{
				randomizeOrder:       config.RandomizeOrder,
				allowParking:         config.AllowParking,
				timeoutTicks:         config.TimeoutTicks,
				capturePendingTraces: config.CapturePendingTraces,
				pendingTraces:        map[traceID]string{},
				isMainThread:         true,
			},
			owner:    currentGoroutineID(),
			unparker: newSignal(),
		},
		clock: NewTestClock(),
	}
}

// EndTest panics with the recorded non-determinism error, if any, and marks the test finished
// so later activity from other goroutines panics immediately.
func (s *TestScheduler) EndTest() {
	sh := s.state
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if e := sh.state.nonDeterminismError; e != nil {
		panic(fmt.Sprintf("%s\n%s", e.message, e.backtrace))
	}
	sh.state.finished = true
}

// Clock returns the scheduler's virtual clock.
func (s *TestScheduler) Clock() Clock {
	return s.clock
}

// TestClock returns the scheduler's virtual clock as its concrete type.
func (s *TestScheduler) TestClock() *TestClock {
	return s.clock
}

// Rng returns the scheduler's seeded random number generator.
func (s *TestScheduler) Rng() *SharedRng {
	return s.rng
}

// SetTimeoutTicks sets the inclusive range the tick budget of a Block with a timeout is drawn
// from.
func (s *TestScheduler) SetTimeoutTicks(timeoutTicks TickRange) {
	s.state.mu.Lock()
	s.state.state.timeoutTicks = timeoutTicks
	s.state.mu.Unlock()
}

// AllowParking lets Block wait for real time when no work remains, and disables the
// non-determinism check for the rest of the scheduler's life.
func (s *TestScheduler) AllowParking() {
	s.state.mu.Lock()
	s.state.state.allowParking = true
	s.state.state.parkingAllowedOnce = true
	s.state.mu.Unlock()
}

// ForbidParking makes Block panic when no work remains.
func (s *TestScheduler) ForbidParking() {
	s.state.mu.Lock()
	s.state.state.allowParking = false
	s.state.mu.Unlock()
}

// ParkingAllowed reports whether parking is allowed.
func (s *TestScheduler) ParkingAllowed() bool {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.state.allowParking
}

// IsMainThread reports whether the runnable being run is a foreground one (or none is running).
func (s *TestScheduler) IsMainThread() bool {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.state.isMainThread
}

// AllocateSessionID returns a fresh session id.
func (s *TestScheduler) AllocateSessionID() SessionID {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	s.state.state.nextSessionID++
	return s.state.state.nextSessionID
}

// Foreground returns a local executor on a fresh session of this scheduler.
func (s *TestScheduler) Foreground() LocalExecutor {
	sessionID := s.AllocateSessionID()
	return NewLocalExecutor(sessionID, s, scheduleLocalDispatch(weak.Make(s), sessionID))
}

// Background returns a background executor for this scheduler.
func (s *TestScheduler) Background() BackgroundExecutor {
	return NewBackgroundExecutor(s)
}

// YieldRandom returns a Yield of a random length: usually 0 or 1 polls, sometimes 10 to 19.
func (s *TestScheduler) YieldRandom() *Yield {
	rng, unlock := s.rng.Lock()
	defer unlock()
	var y Yield
	if rng.Float64() < 0.1 {
		y = Yield(10 + rng.IntN(10))
	} else {
		y = Yield(rng.IntN(2))
	}
	return &y
}

// Run runs runnables and expired timers until no work remains.
func (s *TestScheduler) Run() {
	for s.step() {
	}
}

// RunWithClockAdvancement is Run that also jumps the clock to the next timer whenever no work
// remains.
func (s *TestScheduler) RunWithClockAdvancement() {
	for s.step() || s.AdvanceClockToNextTimer() {
	}
}

// Tick processes expired timers or runs at most one runnable, and reports whether it did any
// work.
func (s *TestScheduler) Tick() bool {
	return s.stepFiltered(false)
}

// TickBackgroundOnly is Tick restricted to background runnables.
func (s *TestScheduler) TickBackgroundOnly() bool {
	return s.stepFiltered(true)
}

// HasPendingTasks reports whether any runnable or timer is queued.
func (s *TestScheduler) HasPendingTasks() bool {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return len(s.state.state.runnables) != 0 || len(s.state.state.timers) != 0
}

// PendingTaskCounts returns the number of queued foreground (session) and background runnables.
func (s *TestScheduler) PendingTaskCounts() (foreground int, background int) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	for _, r := range s.state.state.runnables {
		if r.hasSession {
			foreground++
		} else {
			background++
		}
	}
	return foreground, background
}

func (s *TestScheduler) step() bool {
	return s.stepFiltered(false)
}

func (s *TestScheduler) stepFiltered(backgroundOnly bool) bool {
	sh := s.state
	sh.mu.Lock()
	now := s.clock.Now()
	endIx, _ := slices.BinarySearchFunc(sh.state.timers, now, func(t scheduledTimer, now Instant) int {
		if t.expiration.After(now) {
			return 1
		}
		return -1
	})
	elapsed := slices.Clone(sh.state.timers[:endIx])
	sh.state.timers = slices.Delete(sh.state.timers, 0, endIx)
	runnablesBefore := len(sh.state.runnables)
	sh.mu.Unlock()
	for _, t := range elapsed {
		t.notify.Close()
	}

	if len(elapsed) > 0 {
		sh.mu.Lock()
		runnablesAfter := len(sh.state.runnables)
		sh.mu.Unlock()
		if _, ok := os.LookupEnv("DEBUG_SCHEDULER"); ok {
			fmt.Fprintf(os.Stderr, "[scheduler] Expired %d timers at %v, runnables: %d -> %d\n",
				len(elapsed), s.clock.Now().Time(), runnablesBefore, runnablesAfter)
		}
		return true
	}

	var runnable *scheduledRunnable
	sh.mu.Lock()
	seenSessions := map[SessionID]struct{}{}
	var candidateIndices []int
	for ix, r := range sh.state.runnables {
		if r.hasSession {
			if backgroundOnly {
				continue
			}
			if slices.Contains(sh.state.blockedSessions, r.sessionID) {
				continue
			}
			if _, seen := seenSessions[r.sessionID]; seen {
				continue
			}
			seenSessions[r.sessionID] = struct{}{}
		}
		candidateIndices = append(candidateIndices, ix)
	}

	if len(candidateIndices) != 0 {
		var ix int
		if sh.state.randomizeOrder {
			weights := make([]uint32, len(candidateIndices))
			var totalWeight uint32
			for i, cix := range candidateIndices {
				weights[i] = sh.state.runnables[cix].priority.Weight()
				totalWeight += weights[i]
			}
			if totalWeight == 0 {
				ix = candidateIndices[RandomRange(s.rng, 0, len(candidateIndices))]
			} else {
				target := RandomRange(s.rng, 0, totalWeight)
				selected := 0
				for i, w := range weights {
					if target < w {
						selected = i
						break
					}
					target -= w
				}
				ix = candidateIndices[selected]
			}
		} else {
			ix = candidateIndices[0]
		}
		r := sh.state.runnables[ix]
		sh.state.runnables = slices.Delete(sh.state.runnables, ix, ix+1)
		runnable = &r
	}
	sh.mu.Unlock()

	if runnable != nil {
		sh.mu.Lock()
		wasMainThread := sh.state.isMainThread
		sh.state.isMainThread = runnable.hasSession
		sh.mu.Unlock()
		runnable.runnable.Run()
		sh.mu.Lock()
		sh.state.isMainThread = wasMainThread
		sh.mu.Unlock()
		return true
	}
	return false
}

// DrainTasks drops every queued runnable and timer, repeating until none remain because
// dropping one may schedule others. The leak detector uses it so queued tasks do not keep
// entities alive.
func (s *TestScheduler) DrainTasks() {
	sh := s.state
	for {
		sh.mu.Lock()
		if len(sh.state.runnables) == 0 && len(sh.state.timers) == 0 {
			sh.mu.Unlock()
			break
		}
		runnables := sh.state.runnables
		timers := sh.state.timers
		sh.state.runnables = nil
		sh.state.timers = nil
		sh.mu.Unlock()
		for _, t := range timers {
			t.notify.Close()
		}
		for _, r := range runnables {
			r.runnable.Close()
		}
	}
}

// AdvanceClockToNextTimer moves the clock to the earliest timer, reporting whether there was one.
func (s *TestScheduler) AdvanceClockToNextTimer() bool {
	s.state.mu.Lock()
	if len(s.state.state.timers) == 0 {
		s.state.mu.Unlock()
		return false
	}
	expiration := s.state.state.timers[0].expiration
	s.state.mu.Unlock()
	s.clock.Advance(expiration.Sub(s.clock.Now()))
	return true
}

// AdvanceClock runs work and fires timers until the clock has moved forward by duration.
func (s *TestScheduler) AdvanceClock(duration time.Duration) {
	_, debugScheduler := os.LookupEnv("DEBUG_SCHEDULER")
	start := s.clock.Now()
	nextNow := start.Add(duration)
	if debugScheduler {
		s.state.mu.Lock()
		timerCount := len(s.state.state.timers)
		s.state.mu.Unlock()
		fmt.Fprintf(os.Stderr, "[scheduler] advance_clock(%v) from %v, %d pending timers\n", duration, start.Time(), timerCount)
	}
	for {
		s.Run()
		s.state.mu.Lock()
		var advanceTo Instant
		fire := false
		if len(s.state.state.timers) != 0 && !s.state.state.timers[0].expiration.After(nextNow) {
			advanceTo = s.state.state.timers[0].expiration
			fire = true
		}
		s.state.mu.Unlock()
		if !fire {
			break
		}
		if debugScheduler {
			fmt.Fprintf(os.Stderr, "[scheduler] Advancing clock %v -> %v for timer\n", s.clock.Now().Time(), advanceTo.Time())
		}
		s.clock.Advance(advanceTo.Sub(s.clock.Now()))
	}
	s.clock.Advance(nextNow.Sub(s.clock.Now()))
	if debugScheduler {
		fmt.Fprintf(os.Stderr, "[scheduler] advance_clock done, now at %v\n", s.clock.Now().Time())
	}
}

func (s *TestScheduler) park(deadline *Instant) bool {
	sh := s.state
	sh.mu.Lock()
	allowParking := sh.state.allowParking
	capture := sh.state.capturePendingTraces
	sh.mu.Unlock()
	if allowParking {
		start := InstantNow()
		hardDeadline := start.Add(15 * time.Second)
		effectiveDeadline := hardDeadline
		if deadline != nil {
			effectiveDeadline = deadline.Min(hardDeadline)
		}
		const parkInterval = 100 * time.Millisecond
		for {
			now := InstantNow()
			if !now.Before(effectiveDeadline) {
				if !now.Before(hardDeadline) {
					panic("Test timed out after 15 seconds while parking. This may indicate a deadlock or missing waker.")
				}
				return false
			}

			remaining := effectiveDeadline.SaturatingDurationSince(now)
			parkDuration := min(remaining, parkInterval)
			beforePark := InstantNow()
			sh.unparker.parkTimeout(parkDuration)
			elapsed := beforePark.Elapsed()

			s.clock.Advance(elapsed)

			sh.mu.Lock()
			timerExpired := len(sh.state.timers) != 0 && !sh.state.timers[0].expiration.After(s.clock.Now())
			sh.mu.Unlock()
			if timerExpired {
				return true
			}

			sh.mu.Lock()
			unparked := sh.state.unparked
			sh.state.unparked = false
			sh.mu.Unlock()
			if unparked {
				return true
			}
		}
	} else if deadline != nil {
		return false
	} else if capture {
		var pendingTraces strings.Builder
		sh.mu.Lock()
		traces := sh.state.pendingTraces
		sh.state.pendingTraces = map[traceID]string{}
		sh.mu.Unlock()
		for _, id := range slices.Sorted(mapKeys(traces)) {
			fmt.Fprintf(&pendingTraces, "%s\n", traces[id])
		}
		panic(fmt.Sprintf("Parking forbidden. Pending traces:\n%s", pendingTraces.String()))
	} else {
		panic(fmt.Sprintf("Parking forbidden. Re-run with %s=1 to show pending traces", pendingTracesVarName))
	}
}

func mapKeys[K comparable, V any](m map[K]V) func(yield func(K) bool) {
	return func(yield func(K) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func (s *signal) parkTimeout(d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-s.ch:
	case <-timer.C:
	}
}

func currentGoroutineID() int64 {
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

func isCoroutineGoroutine() bool {
	buf := make([]byte, 8<<10)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return bytes.Contains(buf[:n], []byte("\ncreated by iter.Pull"))
		}
		buf = make([]byte, 2*len(buf))
	}
}

func (sh *testSchedulerShared) onOwnerThread() bool {
	id := currentGoroutineID()
	if id == sh.owner {
		return true
	}
	if v, ok := sh.coroutine.Load(id); ok {
		return v.(bool)
	}
	c := isCoroutineGoroutine()
	sh.coroutine.Store(id, c)
	return c
}

func assertCorrectThread(sh *testSchedulerShared) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if sh.state.parkingAllowedOnce {
		return
	}
	if sh.onOwnerThread() {
		return
	}

	message := fmt.Sprintf(
		"Detected activity on goroutine %d, but test scheduler is running on goroutine %d. Your test is not deterministic.",
		currentGoroutineID(),
		sh.owner,
	)
	backtrace := string(debug.Stack())
	if sh.state.finished {
		panic(message)
	}
	sh.state.nonDeterminismError = &nonDeterminismError{message: message, backtrace: backtrace}
}

// Block polls future until it completes or the timeout's tick budget runs out, running other
// work in between. When nothing can progress it jumps the clock to the next timer, or, if
// parking is allowed, waits for real time; otherwise it panics with "Parking forbidden.". The
// caller keeps ownership of future.
func (s *TestScheduler) Block(sessionID *SessionID, future Future[struct{}], timeout *time.Duration) bool {
	return s.blockUntil(sessionID, future, timeout)
}

// ScheduleLocal queues runnable behind the session's earlier runnables, at a random position
// when the order is randomized.
func (s *TestScheduler) ScheduleLocal(sessionID SessionID, runnable *Runnable) {
	sh := s.state
	assertCorrectThread(sh)
	sh.mu.Lock()
	ix := len(sh.state.runnables)
	if sh.state.randomizeOrder {
		startIx := 0
		for i := len(sh.state.runnables) - 1; i >= 0; i-- {
			if r := sh.state.runnables[i]; r.hasSession && r.sessionID == sessionID {
				startIx = i + 1
				break
			}
		}
		ix = s.rng.randomRangeInclusive(startIx, len(sh.state.runnables))
	}
	sh.state.runnables = slices.Insert(sh.state.runnables, ix, scheduledRunnable{
		sessionID:  sessionID,
		hasSession: true,
		priority:   DefaultPriority(),
		runnable:   runnable,
	})
	sh.state.unparked = true
	sh.mu.Unlock()
	sh.unparker.notify()
}

// ScheduleBackgroundWithPriority queues a background runnable, at a random position when the
// order is randomized.
func (s *TestScheduler) ScheduleBackgroundWithPriority(runnable *Runnable, priority Priority) {
	sh := s.state
	assertCorrectThread(sh)
	sh.mu.Lock()
	ix := len(sh.state.runnables)
	if sh.state.randomizeOrder {
		ix = s.rng.randomRangeInclusive(0, len(sh.state.runnables))
	}
	sh.state.runnables = slices.Insert(sh.state.runnables, ix, scheduledRunnable{
		priority: priority,
		runnable: runnable,
	})
	sh.state.unparked = true
	sh.mu.Unlock()
	sh.unparker.notify()
}

// ScheduleBackground queues a background runnable with DefaultPriority.
func (s *TestScheduler) ScheduleBackground(runnable *Runnable) {
	s.ScheduleBackgroundWithPriority(runnable, DefaultPriority())
}

// SpawnRealtime runs f on a new goroutine.
func (s *TestScheduler) SpawnRealtime(f func()) {
	go f()
}

// Timer returns a timer that fires when the virtual clock passes now plus duration.
func (s *TestScheduler) Timer(duration time.Duration) *Timer {
	tx, rx := Oneshot[struct{}]()
	s.state.mu.Lock()
	s.state.state.timers = append(s.state.state.timers, scheduledTimer{
		expiration: s.clock.Now().Add(duration),
		notify:     tx,
	})
	slices.SortStableFunc(s.state.state.timers, func(a, b scheduledTimer) int {
		return a.expiration.Compare(b.expiration)
	})
	s.state.mu.Unlock()
	return NewTimer(rx)
}

// SpawnDedicated runs f on a fresh local session driven by the test loop; no goroutine is
// started, so runs stay deterministic.
func (s *TestScheduler) SpawnDedicated(f func(executor LocalExecutor, s *Scope) (any, error)) *Task[any] {
	sessionID := s.AllocateSessionID()
	weakScheduler := weak.Make(s)
	spawner := NewLocalExecutor(sessionID, s, scheduleLocalDispatch(weakScheduler, sessionID))
	return spawner.spawnAt(CallerLocation(1), func(sc *Scope) (any, error) {
		scheduler := weakScheduler.Value()
		if scheduler == nil {
			panic("dedicated tasks are only polled by their scheduler, which is still alive when polled")
		}
		executor := NewLocalExecutor(sessionID, scheduler, scheduleLocalDispatch(weakScheduler, sessionID))
		return f(executor, sc)
	})
}

// AsTest returns s.
func (s *TestScheduler) AsTest() *TestScheduler {
	return s
}

func scheduleLocalDispatch(scheduler weak.Pointer[TestScheduler], sessionID SessionID) func(*Runnable) {
	return func(runnable *Runnable) {
		if s := scheduler.Value(); s != nil {
			s.ScheduleLocal(sessionID, runnable)
		} else {
			runnable.Close()
		}
	}
}

// TickRange is an inclusive range of tick counts (RangeInclusive<usize>).
type TickRange struct {
	Start int
	End   int
}

// TestSchedulerConfig configures a TestScheduler.
type TestSchedulerConfig struct {
	Seed                 uint64
	RandomizeOrder       bool
	AllowParking         bool
	CapturePendingTraces bool
	TimeoutTicks         TickRange
}

// TestSchedulerConfigWithSeed returns the default configuration with seed.
func TestSchedulerConfigWithSeed(seed uint64) TestSchedulerConfig {
	c := DefaultTestSchedulerConfig()
	c.Seed = seed
	return c
}

// DefaultTestSchedulerConfig returns seed 0, randomized order, parking forbidden, pending traces
// from PENDING_TRACES and timeout ticks 1..=1000.
func DefaultTestSchedulerConfig() TestSchedulerConfig {
	v, ok := os.LookupEnv(pendingTracesVarName)
	return TestSchedulerConfig{
		Seed:                 0,
		RandomizeOrder:       true,
		AllowParking:         false,
		CapturePendingTraces: ok && (v == "1" || v == "true"),
		TimeoutTicks:         TickRange{Start: 1, End: 1000},
	}
}

type scheduledRunnable struct {
	sessionID  SessionID
	hasSession bool
	priority   Priority
	runnable   *Runnable
}

type scheduledTimer struct {
	expiration Instant
	notify     *OneshotSender[struct{}]
}

type nonDeterminismError struct {
	message   string
	backtrace string
}

type schedulerState struct {
	runnables            []scheduledRunnable
	timers               []scheduledTimer
	blockedSessions      []SessionID
	randomizeOrder       bool
	allowParking         bool
	timeoutTicks         TickRange
	nextSessionID        SessionID
	capturePendingTraces bool
	nextTraceID          traceID
	pendingTraces        map[traceID]string
	isMainThread         bool
	nonDeterminismError  *nonDeterminismError
	parkingAllowedOnce   bool
	finished             bool
	unparked             bool
}

type traceID int

type tracingWaker struct {
	id     traceID
	hasID  bool
	awoken *atomic.Bool
	shared *testSchedulerShared
}

func (w *tracingWaker) Clone() Waker {
	sh := w.shared
	clone := &tracingWaker{awoken: w.awoken, shared: sh}
	sh.mu.Lock()
	capture := sh.state.capturePendingTraces
	if capture {
		clone.id = sh.state.nextTraceID
		clone.hasID = true
		sh.state.nextTraceID++
	}
	sh.mu.Unlock()
	if capture {
		trace := captureTrace()
		sh.mu.Lock()
		sh.state.pendingTraces[clone.id] = trace
		sh.mu.Unlock()
	}
	return clone
}

func (w *tracingWaker) Drop() {
	assertCorrectThread(w.shared)
	if w.hasID {
		w.shared.mu.Lock()
		delete(w.shared.state.pendingTraces, w.id)
		w.shared.mu.Unlock()
	}
}

func (w *tracingWaker) Wake() {
	w.WakeByRef()
	w.Drop()
}

func (w *tracingWaker) WakeByRef() {
	sh := w.shared
	assertCorrectThread(sh)
	sh.mu.Lock()
	if w.hasID {
		delete(sh.state.pendingTraces, w.id)
	}
	sh.state.unparked = true
	sh.mu.Unlock()
	w.awoken.Store(true)
	sh.unparker.notify()
}

func captureTrace() string {
	pcs := make([]uintptr, 64)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var b strings.Builder
	for {
		f, more := frames.Next()
		fmt.Fprintf(&b, "%s\n\t%s:%d\n", f.Function, f.File, f.Line)
		if !more {
			break
		}
	}
	return b.String()
}

// Yield is a future that wakes itself and stays pending for a number of polls before
// completing.
type Yield int

// Poll completes once the remaining count is zero, otherwise decrements it and wakes itself.
func (y *Yield) Poll(cx *Context) (struct{}, bool) {
	if *y == 0 {
		return struct{}{}, true
	}
	*y--
	cx.Waker().WakeByRef()
	return struct{}{}, false
}

// SharedRng is a mutex-guarded seeded PCG generator shared by a scheduler and its tests.
type SharedRng struct {
	mu  sync.Mutex
	rng *rand.Rand
}

func newSharedRng(seed uint64) *SharedRng {
	const mul = 6364136223846793005
	const inc = 11634580027462260723
	state := seed
	next := func() uint64 {
		var out uint64
		for i := range 2 {
			state = state*mul + inc
			xorshifted := uint32(((state >> 18) ^ state) >> 27)
			rot := uint32(state >> 59)
			out |= uint64(bits.RotateLeft32(xorshifted, -int(rot))) << (32 * i)
		}
		return out
	}
	hi := next()
	lo := next()
	return &SharedRng{rng: rand.New(rand.NewPCG(hi, lo))}
}

// Lock locks the generator for several draws; call the returned function to unlock it.
func (r *SharedRng) Lock() (*rand.Rand, func()) {
	r.mu.Lock()
	return r.rng, r.mu.Unlock
}

// RandomRange returns a value in the half-open range [lo, hi).
func RandomRange[T rangeInt](r *SharedRng, lo, hi T) T {
	if hi <= lo {
		panic("cannot sample empty range")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return lo + T(r.rng.Uint64N(uint64(hi-lo)))
}

func (r *SharedRng) randomRangeInclusive(lo, hi int) int {
	if hi < lo {
		panic("cannot sample empty range")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return lo + int(r.rng.Uint64N(uint64(hi-lo)+1))
}

// RandomBool returns true with probability p.
func (r *SharedRng) RandomBool(p float64) bool {
	if !(p >= 0 && p <= 1) {
		panic(fmt.Sprintf("p=%v is outside range [0.0, 1.0]", p))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Float64() < p
}

// Random returns a uniformly distributed value of type T (floats in [0, 1)).
func Random[T randomValue](r *SharedRng) T {
	r.mu.Lock()
	defer r.mu.Unlock()
	var zero T
	switch any(zero).(type) {
	case bool:
		return any(r.rng.Uint64()>>63 == 1).(T)
	case int:
		return any(int(r.rng.Uint64())).(T)
	case uint32:
		return any(r.rng.Uint32()).(T)
	case uint64:
		return any(r.rng.Uint64()).(T)
	case float32:
		return any(r.rng.Float32()).(T)
	default:
		return any(r.rng.Float64()).(T)
	}
}

// RandomRatio returns true with probability numerator/denominator.
func (r *SharedRng) RandomRatio(numerator, denominator uint32) bool {
	if denominator == 0 || numerator > denominator {
		panic(fmt.Sprintf("p={%d}/{%d} is outside range [0.0, 1.0]", numerator, denominator))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Uint32N(denominator) < numerator
}

type rangeInt interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type randomValue interface {
	bool | int | uint32 | uint64 | float32 | float64
}
