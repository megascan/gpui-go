package scheduler

import (
	"sync"
	"time"
)

// Instant is a monotonic point in time (web_time::Instant).
type Instant struct {
	t time.Time
}

// InstantNow returns the current monotonic instant.
func InstantNow() Instant {
	return Instant{t: time.Now()}
}

// Add returns the instant d after i.
func (i Instant) Add(d time.Duration) Instant {
	return Instant{t: i.t.Add(d)}
}

// Sub returns i - earlier, saturating at zero as Rust's Instant subtraction does.
func (i Instant) Sub(earlier Instant) time.Duration {
	return i.SaturatingDurationSince(earlier)
}

// SaturatingDurationSince returns the time elapsed from earlier to i, or zero if earlier is later.
func (i Instant) SaturatingDurationSince(earlier Instant) time.Duration {
	d := i.t.Sub(earlier.t)
	if d < 0 {
		return 0
	}
	return d
}

// Elapsed returns the real time elapsed since i.
func (i Instant) Elapsed() time.Duration {
	return InstantNow().SaturatingDurationSince(i)
}

// Before reports whether i is earlier than other.
func (i Instant) Before(other Instant) bool {
	return i.t.Before(other.t)
}

// After reports whether i is later than other.
func (i Instant) After(other Instant) bool {
	return i.t.After(other.t)
}

// Compare returns -1, 0 or +1 as i is before, equal to or after other.
func (i Instant) Compare(other Instant) int {
	return i.t.Compare(other.t)
}

// Equal reports whether i and other are the same instant.
func (i Instant) Equal(other Instant) bool {
	return i.t.Equal(other.t)
}

// Min returns the earlier of i and other.
func (i Instant) Min(other Instant) Instant {
	if other.Before(i) {
		return other
	}
	return i
}

// Time returns the instant as a time.Time carrying a monotonic reading.
func (i Instant) Time() time.Time {
	return i.t
}

// Clock is a source of wall-clock and monotonic time.
type Clock interface {
	UTCNow() time.Time
	Now() Instant
}

// TestClock is a manually advanced Clock.
type TestClock struct {
	mu    sync.Mutex
	state testClockState
}

type testClockState struct {
	now    Instant
	utcNow time.Time
}

// NewTestClock returns a TestClock whose wall clock starts at 2025-07-01T23:59:58Z.
func NewTestClock() *TestClock {
	const startTime = "2025-07-01T23:59:58-00:00"
	utcNow, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		panic(err)
	}
	return &TestClock{state: testClockState{
		now:    InstantNow(),
		utcNow: utcNow.UTC(),
	}}
}

// SetUTCNow sets the wall-clock time.
func (c *TestClock) SetUTCNow(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.utcNow = now.UTC()
}

// Advance moves both clocks forward by duration.
func (c *TestClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.now = c.state.now.Add(duration)
	c.state.utcNow = c.state.utcNow.Add(duration)
}

// UTCNow returns the wall-clock time.
func (c *TestClock) UTCNow() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state.utcNow
}

// Now returns the monotonic time.
func (c *TestClock) Now() Instant {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state.now
}
