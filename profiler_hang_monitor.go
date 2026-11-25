package gpui

import (
	"fmt"
	"sync"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

// HangMonitorConfig holds detection thresholds and polling cadence for App.StartHangMonitor.
type HangMonitorConfig struct {
	Threshold   time.Duration
	FrameBudget time.Duration
	Interval    time.Duration
}

// HangMonitorError is why App.StartHangMonitor failed: HangMonitorErrorAlreadyStarted or
// *HangMonitorErrorSpawn.
type HangMonitorError interface {
	error
	isHangMonitorError()
}

// HangMonitorErrorAlreadyStarted reports that the monitor was already started for this app.
type HangMonitorErrorAlreadyStarted struct{}

// HangMonitorErrorSpawn reports that the monitor thread couldn't be spawned.
type HangMonitorErrorSpawn struct {
	Err error
}

func (HangMonitorErrorAlreadyStarted) Error() string {
	return "the hang monitor was already started"
}

func (e *HangMonitorErrorSpawn) Error() string {
	return fmt.Sprintf("failed to spawn the hang monitor thread: %v", e.Err)
}

// Unwrap returns the spawn error.
func (e *HangMonitorErrorSpawn) Unwrap() error {
	return e.Err
}

func (HangMonitorErrorAlreadyStarted) isHangMonitorError() {}
func (*HangMonitorErrorSpawn) isHangMonitorError()         {}

// HangMonitorPollReason is why a hang monitor polled its detector.
type HangMonitorPollReason uint8

const (
	// HangMonitorPollReasonInterval means the monitor's regular interval elapsed.
	HangMonitorPollReasonInterval HangMonitorPollReason = iota
	// HangMonitorPollReasonFlush means a poll was requested before the application quits.
	HangMonitorPollReasonFlush
)

// HangMonitorPoll is the incidents a hang monitor collected in one poll.
type HangMonitorPoll struct {
	Incidents      []HangIncident
	FirstPresentAt Option[scheduler.Instant]
	ActiveTime     time.Duration
	Reason         HangMonitorPollReason
}

type hangMonitorRequest struct {
	done chan struct{}
}

type hangMonitor struct {
	mu           sync.Mutex
	pending      []hangMonitorRequest
	wake         chan struct{}
	disconnected bool
	exited       bool
}

func spawnHangMonitor(detector *HangDetector, interval time.Duration, onPoll func(HangMonitorPoll)) (*hangMonitor, error) {
	m := &hangMonitor{wake: make(chan struct{}, 1)}
	go m.run(detector, interval, onPoll)
	return m, nil
}

func (m *hangMonitor) recvTimeout(interval time.Duration) (hangMonitorRequest, bool, bool) {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		m.mu.Lock()
		if len(m.pending) > 0 {
			request := m.pending[0]
			m.pending = m.pending[1:]
			m.mu.Unlock()
			return request, true, false
		}
		if m.disconnected {
			m.mu.Unlock()
			return hangMonitorRequest{}, false, true
		}
		m.mu.Unlock()
		select {
		case <-m.wake:
		case <-timer.C:
			return hangMonitorRequest{}, false, false
		}
	}
}

func (m *hangMonitor) run(detector *HangDetector, interval time.Duration, onPoll func(HangMonitorPoll)) {
	defer func() {
		m.mu.Lock()
		m.exited = true
		m.mu.Unlock()
	}()
	for {
		request, flush, disconnected := m.recvTimeout(interval)
		if disconnected {
			return
		}
		reason := HangMonitorPollReasonInterval
		if flush {
			reason = HangMonitorPollReasonFlush
		}
		incidents := detector.Poll()
		activeTime := detector.TakeActiveTime(scheduler.InstantNow(), interval*2)
		onPoll(HangMonitorPoll{
			Incidents:      incidents,
			FirstPresentAt: OptionFrom(detector.FirstPresentAt()),
			ActiveTime:     activeTime,
			Reason:         reason,
		})
		if flush {
			close(request.done)
		}
	}
}

func (m *hangMonitor) send(request hangMonitorRequest) bool {
	m.mu.Lock()
	if m.exited || m.disconnected {
		m.mu.Unlock()
		return false
	}
	m.pending = append(m.pending, request)
	m.mu.Unlock()
	select {
	case m.wake <- struct{}{}:
	default:
	}
	return true
}

func (m *hangMonitor) requestFlush() (<-chan struct{}, bool) {
	done := make(chan struct{})
	if !m.send(hangMonitorRequest{done: done}) {
		return nil, false
	}
	return done, true
}

// Close stops the monitor thread, as upstream's Drop of the request sender.
func (m *hangMonitor) Close() {
	m.mu.Lock()
	m.disconnected = true
	m.mu.Unlock()
	select {
	case m.wake <- struct{}{}:
	default:
	}
}
