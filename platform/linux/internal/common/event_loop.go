//go:build linux

package common

import (
	"container/heap"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/megascan/gpui-go"
	"golang.org/x/sys/unix"
)

// PostAction is what an event source asks the loop to do with it after processing its events.
type PostAction uint8

const (
	// PostActionContinue keeps the source registered.
	PostActionContinue PostAction = iota
	// PostActionReregister re-reads the source's fd and interest.
	PostActionReregister
	// PostActionDisable stops polling the source until EnableSource.
	PostActionDisable
	// PostActionRemove removes the source and closes it.
	PostActionRemove
)

// Interest is the readiness a source waits for.
type Interest struct {
	Readable bool
	Writable bool
}

// InterestRead waits for the fd to become readable.
var InterestRead = Interest{Readable: true}

// Readiness is the readiness the poll reported for a source.
type Readiness struct {
	Readable bool
	Writable bool
	Error    bool
}

// EventSource is a source the event loop polls (calloop::EventSource with its callback bound).
type EventSource interface {
	Fd() int
	Interest() Interest
	ProcessEvents(readiness Readiness) (PostAction, error)
}

// LifecycleEventSource is an EventSource that runs before the loop sleeps and before events are
// handled (calloop's NEEDS_EXTRA_LIFECYCLE_EVENTS), as the Wayland source does for prepare_read.
type LifecycleEventSource interface {
	EventSource
	BeforeSleep() (Readiness, bool, error)
	BeforeHandleEvents(readiness Readiness, ready bool)
}

// TypedEventSource is an event source that hands events of type E to the callback given to
// InsertSource.
type TypedEventSource[E any] interface {
	Fd() int
	Interest() Interest
	ProcessEvents(readiness Readiness, callback func(E)) (PostAction, error)
}

type closer interface{ Close() }

type typedSource[E any] struct {
	source   TypedEventSource[E]
	callback func(E)
}

func (s *typedSource[E]) Fd() int            { return s.source.Fd() }
func (s *typedSource[E]) Interest() Interest { return s.source.Interest() }
func (s *typedSource[E]) ProcessEvents(readiness Readiness) (PostAction, error) {
	return s.source.ProcessEvents(readiness, s.callback)
}
func (s *typedSource[E]) Close() {
	if c, ok := s.source.(closer); ok {
		c.Close()
	}
}

// RegistrationToken identifies a source or timer registered on a loop.
type RegistrationToken struct{ id uint64 }

// TimeoutAction is what a timer callback asks the loop to do with its timer.
type TimeoutAction struct {
	kind     uint8
	duration time.Duration
	instant  time.Time
}

// TimeoutActionDrop removes the timer.
func TimeoutActionDrop() TimeoutAction { return TimeoutAction{} }

// TimeoutActionToDuration fires the timer again after d.
func TimeoutActionToDuration(d time.Duration) TimeoutAction {
	return TimeoutAction{kind: 1, duration: d}
}

// TimeoutActionToInstant fires the timer again at t.
func TimeoutActionToInstant(t time.Time) TimeoutAction {
	return TimeoutAction{kind: 2, instant: t}
}

type sourceEntry struct {
	source  EventSource
	enabled bool
}

type loopTimer struct {
	deadline time.Time
	seq      uint64
	id       uint64
	callback func(deadline time.Time) TimeoutAction
}

type loopTimerHeap []*loopTimer

func (h loopTimerHeap) Len() int { return len(h) }
func (h loopTimerHeap) Less(i, j int) bool {
	if c := h[i].deadline.Compare(h[j].deadline); c != 0 {
		return c < 0
	}
	return h[i].seq < h[j].seq
}
func (h loopTimerHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *loopTimerHeap) Push(x any)   { *h = append(*h, x.(*loopTimer)) }
func (h *loopTimerHeap) Pop() any {
	old := *h
	t := old[len(old)-1]
	old[len(old)-1] = nil
	*h = old[:len(old)-1]
	return t
}

type idleEntry struct {
	callback  func()
	cancelled bool
}

// Idle is an idle callback registered with InsertIdle.
type Idle struct{ entry *idleEntry }

// Cancel keeps the idle callback from running.
func (i Idle) Cancel() { i.entry.cancelled = true }

type loopInner struct {
	sources   map[uint64]*sourceEntry
	order     []uint64
	timers    loopTimerHeap
	timerIDs  map[uint64]*loopTimer
	idles     []*idleEntry
	nextID    uint64
	nextSeq   uint64
	closed    bool
	wake      *PingSource
	stop      *atomic.Bool
	wakePing  *Ping
	pollFds   []unix.PollFd
	pollOwner []uint64
}

// LoopHandle inserts and removes sources, timers and idle callbacks on an EventLoop
// (calloop::LoopHandle). It is used only on the loop's goroutine.
type LoopHandle struct{ inner *loopInner }

// LoopSignal stops or wakes an EventLoop from any goroutine (calloop::LoopSignal).
type LoopSignal struct {
	stop *atomic.Bool
	ping *Ping
}

// Stop makes Run return once the loop finishes waiting for events.
func (s LoopSignal) Stop() { s.stop.Store(true) }

// Wakeup makes the loop's current or next wait return early.
func (s LoopSignal) Wakeup() { s.ping.Ping() }

// EventLoop is the Go replacement of calloop's EventLoop: the goroutine that calls Dispatch or Run
// polls the registered fds, fires due timers and then runs the idle callbacks.
type EventLoop struct {
	handle *LoopHandle
}

// NewEventLoop creates an event loop (calloop::EventLoop::try_new).
func NewEventLoop() (*EventLoop, error) {
	ping, source, err := MakePing()
	if err != nil {
		return nil, err
	}
	inner := &loopInner{
		sources:  map[uint64]*sourceEntry{},
		timerIDs: map[uint64]*loopTimer{},
		wake:     source,
		stop:     new(atomic.Bool),
		wakePing: ping,
	}
	return &EventLoop{handle: &LoopHandle{inner: inner}}, nil
}

// Handle returns the loop's handle.
func (l *EventLoop) Handle() *LoopHandle { return l.handle }

// GetSignal returns a signal that stops or wakes the loop.
func (l *EventLoop) GetSignal() LoopSignal {
	return LoopSignal{stop: l.handle.inner.stop, ping: l.handle.inner.wakePing}
}

// Close removes and closes every source and the loop's wakeup fd.
func (l *EventLoop) Close() {
	inner := l.handle.inner
	if inner.closed {
		return
	}
	inner.closed = true
	for _, id := range slices.Clone(inner.order) {
		l.handle.Remove(RegistrationToken{id})
	}
	inner.timers = nil
	clear(inner.timerIDs)
	inner.idles = nil
	inner.wake.Close()
}

// InsertSource registers source on the loop, delivering its events to callback.
func InsertSource[E any](h *LoopHandle, source TypedEventSource[E], callback func(E)) (RegistrationToken, error) {
	return h.InsertEventSource(&typedSource[E]{source: source, callback: callback})
}

// InsertEventSource registers source on the loop.
func (h *LoopHandle) InsertEventSource(source EventSource) (RegistrationToken, error) {
	inner := h.inner
	if inner.closed {
		return RegistrationToken{}, errors.New("event loop is closed")
	}
	inner.nextID++
	id := inner.nextID
	inner.sources[id] = &sourceEntry{source: source, enabled: true}
	inner.order = append(inner.order, id)
	return RegistrationToken{id}, nil
}

// InsertIdle runs callback after the loop next finishes dispatching events.
func (h *LoopHandle) InsertIdle(callback func()) Idle {
	entry := &idleEntry{callback: callback}
	h.inner.idles = append(h.inner.idles, entry)
	return Idle{entry: entry}
}

// InsertTimer fires callback at deadline (calloop::timer::Timer::from_deadline); the callback's
// TimeoutAction drops or reschedules the timer.
func (h *LoopHandle) InsertTimer(deadline time.Time, callback func(deadline time.Time) TimeoutAction) RegistrationToken {
	inner := h.inner
	inner.nextID++
	id := inner.nextID
	h.scheduleTimer(id, deadline, callback)
	return RegistrationToken{id}
}

// InsertTimerAfter fires callback after d (calloop::timer::Timer::from_duration).
func (h *LoopHandle) InsertTimerAfter(d time.Duration, callback func(deadline time.Time) TimeoutAction) RegistrationToken {
	return h.InsertTimer(time.Now().Add(d), callback)
}

func (h *LoopHandle) scheduleTimer(id uint64, deadline time.Time, callback func(time.Time) TimeoutAction) {
	inner := h.inner
	inner.nextSeq++
	t := &loopTimer{deadline: deadline, seq: inner.nextSeq, id: id, callback: callback}
	inner.timerIDs[id] = t
	heap.Push(&inner.timers, t)
}

// Remove unregisters and closes the source or timer token names.
func (h *LoopHandle) Remove(token RegistrationToken) {
	inner := h.inner
	if t, ok := inner.timerIDs[token.id]; ok {
		delete(inner.timerIDs, token.id)
		if i := slices.Index(inner.timers, t); i >= 0 {
			heap.Remove(&inner.timers, i)
		}
		return
	}
	entry, ok := inner.sources[token.id]
	if !ok {
		return
	}
	delete(inner.sources, token.id)
	if i := slices.Index(inner.order, token.id); i >= 0 {
		inner.order = slices.Delete(inner.order, i, i+1)
	}
	if c, ok := entry.source.(closer); ok {
		c.Close()
	}
}

// Disable stops polling the source until Enable.
func (h *LoopHandle) Disable(token RegistrationToken) {
	if entry, ok := h.inner.sources[token.id]; ok {
		entry.enabled = false
	}
}

// Enable resumes polling a disabled source.
func (h *LoopHandle) Enable(token RegistrationToken) {
	if entry, ok := h.inner.sources[token.id]; ok {
		entry.enabled = true
	}
}

type pendingEvent struct {
	id        uint64
	readiness Readiness
}

func readinessFrom(revents int16, interest Interest) (Readiness, bool) {
	var r Readiness
	hup := revents&(unix.POLLHUP|unix.POLLERR) != 0
	if interest.Readable && (revents&unix.POLLIN != 0 || hup) {
		r.Readable = true
	}
	if interest.Writable && (revents&unix.POLLOUT != 0 || hup) {
		r.Writable = true
	}
	if revents&(unix.POLLERR|unix.POLLNVAL) != 0 {
		r.Error = true
	}
	return r, r.Readable || r.Writable || r.Error
}

// Dispatch waits up to timeout (forever when none) for events, processes them, fires due timers
// and then runs the pending idle callbacks (calloop::EventLoop::dispatch).
func (l *EventLoop) Dispatch(timeout gpui.Option[time.Duration]) error {
	if err := l.dispatchEvents(timeout); err != nil {
		return err
	}
	l.dispatchIdles()
	return nil
}

// Run dispatches until LoopSignal.Stop, calling cb after every dispatch (calloop::EventLoop::run).
func (l *EventLoop) Run(timeout gpui.Option[time.Duration], cb func()) error {
	inner := l.handle.inner
	inner.stop.Store(false)
	for !inner.stop.Load() {
		if err := l.Dispatch(timeout); err != nil {
			return err
		}
		cb()
	}
	return nil
}

func (l *EventLoop) dispatchEvents(timeoutOpt gpui.Option[time.Duration]) error {
	inner := l.handle.inner
	now := time.Now()
	timeout, hasTimeout := timeoutOpt.Get()
	var synthetic []pendingEvent
	var prepared []LifecycleEventSource
	for _, id := range slices.Clone(inner.order) {
		entry, ok := inner.sources[id]
		if !ok || !entry.enabled {
			continue
		}
		lifecycle, ok := entry.source.(LifecycleEventSource)
		if !ok {
			continue
		}
		readiness, ready, err := lifecycle.BeforeSleep()
		if err != nil {
			for _, p := range prepared {
				p.BeforeHandleEvents(Readiness{}, false)
			}
			return err
		}
		prepared = append(prepared, lifecycle)
		if ready {
			timeout, hasTimeout = 0, true
			synthetic = append(synthetic, pendingEvent{id: id, readiness: readiness})
		}
	}
	if next, ok := inner.timers.peek(); ok {
		untilNext := max(next.deadline.Sub(now), 0)
		if !hasTimeout || untilNext < timeout {
			timeout, hasTimeout = untilNext, true
		}
	}

	inner.pollFds = append(inner.pollFds[:0], unix.PollFd{Fd: int32(inner.wake.Fd()), Events: unix.POLLIN})
	inner.pollOwner = append(inner.pollOwner[:0], 0)
	for _, id := range inner.order {
		entry := inner.sources[id]
		if !entry.enabled {
			continue
		}
		fd := entry.source.Fd()
		if fd < 0 {
			continue
		}
		interest := entry.source.Interest()
		var events int16
		if interest.Readable {
			events |= unix.POLLIN
		}
		if interest.Writable {
			events |= unix.POLLOUT
		}
		inner.pollFds = append(inner.pollFds, unix.PollFd{Fd: int32(fd), Events: events})
		inner.pollOwner = append(inner.pollOwner, id)
	}
	remaining := timeout
	for {
		ms := -1
		if hasTimeout {
			ms = int((remaining + time.Millisecond - 1) / time.Millisecond)
		}
		_, err := unix.Poll(inner.pollFds, ms)
		if err == nil {
			break
		}
		if errors.Is(err, unix.EINTR) {
			if hasTimeout {
				elapsed := time.Since(now)
				if elapsed >= timeout {
					for i := range inner.pollFds {
						inner.pollFds[i].Revents = 0
					}
					break
				}
				remaining = timeout - elapsed
			}
			continue
		}
		for _, p := range prepared {
			p.BeforeHandleEvents(Readiness{}, false)
		}
		return fmt.Errorf("event loop poll: %w", err)
	}

	events := synthetic
	polled := map[uint64]Readiness{}
	for i, pfd := range inner.pollFds {
		owner := inner.pollOwner[i]
		if owner == 0 {
			if pfd.Revents != 0 {
				inner.wake.drain()
			}
			continue
		}
		entry, ok := inner.sources[owner]
		if !ok {
			continue
		}
		if readiness, ready := readinessFrom(pfd.Revents, entry.source.Interest()); ready {
			polled[owner] = readiness
			events = append(events, pendingEvent{id: owner, readiness: readiness})
		}
	}
	for _, id := range slices.Clone(inner.order) {
		entry, ok := inner.sources[id]
		if !ok {
			continue
		}
		if lifecycle, ok := entry.source.(LifecycleEventSource); ok {
			readiness, ready := polled[id]
			lifecycle.BeforeHandleEvents(readiness, ready)
		}
	}

	for _, event := range events {
		entry, ok := inner.sources[event.id]
		if !ok {
			continue
		}
		action, err := entry.source.ProcessEvents(event.readiness)
		if err != nil {
			return err
		}
		switch action {
		case PostActionDisable:
			entry.enabled = false
		case PostActionRemove:
			l.handle.Remove(RegistrationToken{event.id})
		}
	}

	fired := time.Now()
	var due []*loopTimer
	for {
		next, ok := inner.timers.peek()
		if !ok || next.deadline.After(fired) {
			break
		}
		heap.Pop(&inner.timers)
		due = append(due, next)
	}
	for _, next := range due {
		if inner.timerIDs[next.id] != next {
			continue
		}
		delete(inner.timerIDs, next.id)
		action := next.callback(next.deadline)
		switch action.kind {
		case 1:
			l.handle.scheduleTimer(next.id, time.Now().Add(action.duration), next.callback)
		case 2:
			l.handle.scheduleTimer(next.id, action.instant, next.callback)
		}
	}
	return nil
}

func (h loopTimerHeap) peek() (*loopTimer, bool) {
	if len(h) == 0 {
		return nil, false
	}
	return h[0], true
}

func (l *EventLoop) dispatchIdles() {
	inner := l.handle.inner
	idles := inner.idles
	inner.idles = nil
	for _, idle := range idles {
		if !idle.cancelled {
			idle.callback()
		}
	}
}

type pingInner struct {
	mu     sync.Mutex
	fd     int
	closed bool
}

// Ping wakes the PingSource it was made with, from any goroutine (calloop::ping::Ping).
type Ping struct{ inner *pingInner }

// PingSource is the event source of a Ping (calloop::ping::PingSource). Close closes its fd.
type PingSource struct{ inner *pingInner }

// MakePing creates a Ping and its source over an eventfd (calloop::ping::make_ping).
func MakePing() (*Ping, *PingSource, error) {
	fd, err := unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create a Ping.: %w", err)
	}
	inner := &pingInner{fd: fd}
	return &Ping{inner: inner}, &PingSource{inner: inner}, nil
}

// Clone returns another handle on the same ping.
func (p *Ping) Clone() *Ping { return &Ping{inner: p.inner} }

// Ping makes the source's next poll report it ready.
func (p *Ping) Ping() {
	p.inner.mu.Lock()
	defer p.inner.mu.Unlock()
	if p.inner.closed {
		return
	}
	var one = [8]byte{1}
	_, _ = unix.Write(p.inner.fd, one[:])
}

// Fd returns the eventfd.
func (s *PingSource) Fd() int { return s.inner.fd }

// Interest returns InterestRead.
func (s *PingSource) Interest() Interest { return InterestRead }

func (s *PingSource) drain() bool {
	var buf [8]byte
	n, err := unix.Read(s.inner.fd, buf[:])
	return err == nil && n == 8
}

// ProcessEvents clears the ping and calls callback once if it was pinged.
func (s *PingSource) ProcessEvents(readiness Readiness, callback func(struct{})) (PostAction, error) {
	if s.drain() {
		callback(struct{}{})
	}
	return PostActionContinue, nil
}

// Close closes the eventfd; later pings do nothing.
func (s *PingSource) Close() {
	s.inner.mu.Lock()
	defer s.inner.mu.Unlock()
	if s.inner.closed {
		return
	}
	s.inner.closed = true
	_ = unix.Close(s.inner.fd)
}

// ChannelEvent is an event of a Channel: a message, or Closed once every sender is gone
// (calloop::channel::Event).
type ChannelEvent[T any] struct {
	Msg    T
	Closed bool
}

type channelState[T any] struct {
	mu           sync.Mutex
	queue        []T
	senders      int
	receiverGone bool
	ping         *Ping
}

// ErrChannelClosed is returned by ChannelSender.Send once the Channel is gone.
var ErrChannelClosed = errors.New("sending on a closed channel")

// ChannelSender sends messages to a Channel from any goroutine; Close is its drop.
type ChannelSender[T any] struct {
	state  *channelState[T]
	closed atomic.Bool
}

// Channel is the event-source end of a channel (calloop::channel::Channel). Close is its drop.
type Channel[T any] struct {
	state  *channelState[T]
	source *PingSource
}

// NewChannel creates a channel whose receiving end is an event source (calloop::channel::channel).
func NewChannel[T any]() (*ChannelSender[T], *Channel[T], error) {
	ping, source, err := MakePing()
	if err != nil {
		return nil, nil, err
	}
	state := &channelState[T]{senders: 1, ping: ping}
	return &ChannelSender[T]{state: state}, &Channel[T]{state: state, source: source}, nil
}

// Clone returns another sender on the same channel.
func (s *ChannelSender[T]) Clone() *ChannelSender[T] {
	s.state.mu.Lock()
	s.state.senders++
	s.state.mu.Unlock()
	return &ChannelSender[T]{state: s.state}
}

// Send queues v; it fails with ErrChannelClosed once the receiver or this sender is closed.
func (s *ChannelSender[T]) Send(v T) error {
	if s.closed.Load() {
		return ErrChannelClosed
	}
	s.state.mu.Lock()
	if s.state.receiverGone {
		s.state.mu.Unlock()
		return ErrChannelClosed
	}
	s.state.queue = append(s.state.queue, v)
	s.state.mu.Unlock()
	s.state.ping.Ping()
	return nil
}

// Close drops the sender; the channel reports Closed once the last one is gone.
func (s *ChannelSender[T]) Close() {
	if s.closed.Swap(true) {
		return
	}
	s.state.mu.Lock()
	s.state.senders--
	last := s.state.senders == 0
	s.state.mu.Unlock()
	if last {
		s.state.ping.Ping()
	}
}

// Fd returns the channel's eventfd.
func (c *Channel[T]) Fd() int { return c.source.Fd() }

// Interest returns InterestRead.
func (c *Channel[T]) Interest() Interest { return InterestRead }

// ProcessEvents delivers the queued messages, then Closed and removal once every sender is gone.
func (c *Channel[T]) ProcessEvents(readiness Readiness, callback func(ChannelEvent[T])) (PostAction, error) {
	c.source.drain()
	for {
		c.state.mu.Lock()
		if len(c.state.queue) == 0 {
			disconnected := c.state.senders == 0
			c.state.mu.Unlock()
			if disconnected {
				callback(ChannelEvent[T]{Closed: true})
				return PostActionRemove, nil
			}
			return PostActionContinue, nil
		}
		msg := c.state.queue[0]
		var zero T
		c.state.queue[0] = zero
		c.state.queue = c.state.queue[1:]
		c.state.mu.Unlock()
		callback(ChannelEvent[T]{Msg: msg})
	}
}

// Close drops the receiver: later sends fail and its fd is closed.
func (c *Channel[T]) Close() {
	c.state.mu.Lock()
	c.state.receiverGone = true
	c.state.queue = nil
	c.state.mu.Unlock()
	c.source.Close()
}

// Generic is an event source over an fd the caller owns (calloop::generic::Generic).
type Generic struct {
	fd       int
	interest Interest
	callback func(readiness Readiness) (PostAction, error)
}

// NewGeneric wraps fd; callback runs whenever the fd is ready for interest.
func NewGeneric(fd int, interest Interest, callback func(readiness Readiness) (PostAction, error)) *Generic {
	return &Generic{fd: fd, interest: interest, callback: callback}
}

// Fd returns the wrapped fd.
func (g *Generic) Fd() int { return g.fd }

// Interest returns the readiness the source waits for.
func (g *Generic) Interest() Interest { return g.interest }

// SetInterest changes the readiness the source waits for from the next poll on.
func (g *Generic) SetInterest(interest Interest) { g.interest = interest }

// ProcessEvents calls the callback.
func (g *Generic) ProcessEvents(readiness Readiness) (PostAction, error) {
	return g.callback(readiness)
}
