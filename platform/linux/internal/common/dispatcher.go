//go:build linux

package common

import (
	"bytes"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/megascan/gpui-go"
)

type timerAfter struct {
	duration time.Duration
	runnable gpui.RunnableVariant
}

// LinuxDispatcher is gpui_linux's PlatformDispatcher: background runnables run on a pool of
// worker goroutines, timers on a timer goroutine with its own event loop, and main-thread
// runnables are sent to the platform's event loop.
type LinuxDispatcher struct {
	gpui.PlatformDispatcherDefaults
	mainSender       *PriorityQueueCalloopSender[gpui.RunnableVariant]
	timerSender      *ChannelSender[timerAfter]
	backgroundSender *gpui.PriorityQueueSender[gpui.RunnableVariant]
	mainThreadID     int64
	closed           atomic.Bool
}

const minThreads = 2

func runProfiled(thread gpui.ThreadID, runnable gpui.RunnableVariant) {
	meta := runnable.Metadata()
	location := meta.Location
	spawned := meta.Spawned
	gpui.UpdateRunningTask(thread, spawned, location)
	runnable.Run()
	gpui.SaveTaskTiming(thread)
}

// NewLinuxDispatcher starts the worker and timer goroutines; the calling goroutine is the main
// thread.
func NewLinuxDispatcher(mainSender *PriorityQueueCalloopSender[gpui.RunnableVariant]) *LinuxDispatcher {
	backgroundSender, backgroundReceiver := gpui.NewPriorityQueueReceiver[gpui.RunnableVariant]()
	threadCount := max(runtime.NumCPU(), minThreads)
	for i := range threadCount {
		receiver := backgroundReceiver.Clone()
		thread := gpui.NewThreadID(gpui.Some(fmt.Sprintf("Worker-%d", i)))
		go func() {
			for runnable := range receiver.Iter() {
				runProfiled(thread, runnable)
			}
		}()
	}
	backgroundReceiver.Close()

	timerSender, timerChannel, err := NewChannel[timerAfter]()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize timer loop!: %v", err))
	}
	ready := make(chan struct{})
	go func() {
		eventLoop, err := NewEventLoop()
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize timer loop!: %v", err))
		}
		defer eventLoop.Close()
		thread := gpui.NewThreadID(gpui.Some("Timer"))
		handle := eventLoop.Handle()
		signal := eventLoop.GetSignal()
		if _, err := InsertSource(handle, timerChannel, func(e ChannelEvent[timerAfter]) {
			if e.Closed {
				signal.Stop()
				return
			}
			runnable := e.Msg.runnable
			handle.InsertTimerAfter(e.Msg.duration, func(time.Time) TimeoutAction {
				if runnable != nil {
					r := runnable
					runnable = nil
					runProfiled(thread, r)
				}
				return TimeoutActionDrop()
			})
		}); err != nil {
			panic(fmt.Sprintf("Failed to start timer thread: %v", err))
		}
		close(ready)
		if err := eventLoop.Run(gpui.None[time.Duration](), func() {}); err != nil {
			slog.Error(err.Error())
		}
	}()
	<-ready

	d := &LinuxDispatcher{
		mainSender:       mainSender,
		timerSender:      timerSender,
		backgroundSender: backgroundSender,
		mainThreadID:     goroutineID(),
	}
	gpui.BindPlatformDispatcherDefaults(&d.PlatformDispatcherDefaults, d)
	return d
}

// Close drops the dispatcher's senders, which ends the worker and timer goroutines; runnables
// dispatched afterwards are dropped.
func (d *LinuxDispatcher) Close() {
	if d.closed.Swap(true) {
		return
	}
	d.backgroundSender.Close()
	d.timerSender.Close()
	d.mainSender.Close()
}

// IsMainThread reports whether the caller runs on the goroutine that created the dispatcher.
func (d *LinuxDispatcher) IsMainThread() bool {
	return currentThreadGoroutineID() == d.mainThreadID
}

// Dispatch runs runnable on a worker goroutine.
func (d *LinuxDispatcher) Dispatch(runnable gpui.RunnableVariant, priority gpui.Priority) {
	if err := d.backgroundSender.Send(priority, runnable); err != nil && !d.closed.Load() {
		panic("blocking sender returned without value")
	}
}

// DispatchOnMainThread sends runnable to the platform's event loop. If the loop is gone the
// runnable is forgotten, since the app is shutting down.
func (d *LinuxDispatcher) DispatchOnMainThread(runnable gpui.RunnableVariant, priority gpui.Priority) {
	_ = d.mainSender.Send(priority, runnable)
}

// DispatchAfter runs runnable on the timer goroutine after duration. If the timer goroutine has
// shut down the runnable is leaked, leaving its task pending.
func (d *LinuxDispatcher) DispatchAfter(duration time.Duration, runnable gpui.RunnableVariant) {
	_ = d.timerSender.Send(timerAfter{duration: duration, runnable: runnable})
}

// SpawnRealtime runs f on a new goroutine; see D-LINUX-RTPRIO in docs/port/parity/linux-common.md.
func (d *LinuxDispatcher) SpawnRealtime(f func()) {
	go f()
}

// PriorityQueueCalloopSender is a priority queue sender that pings the event loop on every
// send and when it is closed. Close is its drop.
type PriorityQueueCalloopSender[T any] struct {
	sender *gpui.PriorityQueueSender[T]
	ping   *Ping
	closed bool
}

func newPriorityQueueCalloopSender[T any](tx *gpui.PriorityQueueSender[T], ping *Ping) *PriorityQueueCalloopSender[T] {
	return &PriorityQueueCalloopSender[T]{sender: tx, ping: ping}
}

// Send queues item at priority and pings the loop.
func (s *PriorityQueueCalloopSender[T]) Send(priority gpui.Priority, item T) error {
	err := s.sender.Send(priority, item)
	if err == nil {
		s.ping.Ping()
	}
	return err
}

// Close drops the sender and pings the loop so the receiver notices the disconnect.
func (s *PriorityQueueCalloopSender[T]) Close() {
	if s.closed {
		return
	}
	s.closed = true
	s.sender.Close()
	s.ping.Ping()
}

// PriorityQueueCalloopReceiver is the event-source end of a priority queue. Close is its drop.
type PriorityQueueCalloopReceiver[T any] struct {
	receiver *gpui.PriorityQueueReceiver[T]
	source   *PingSource
	ping     *Ping
}

// NewPriorityQueueCalloopReceiver creates a priority queue whose receiver is an event source.
func NewPriorityQueueCalloopReceiver[T any]() (*PriorityQueueCalloopSender[T], *PriorityQueueCalloopReceiver[T]) {
	ping, source, err := MakePing()
	if err != nil {
		panic(fmt.Sprintf("Failed to create a Ping.: %v", err))
	}
	tx, rx := gpui.NewPriorityQueueReceiver[T]()
	return newPriorityQueueCalloopSender(tx, ping.Clone()), &PriorityQueueCalloopReceiver[T]{
		receiver: rx,
		source:   source,
		ping:     ping,
	}
}

// ChannelError is an error of the receiver's ping source.
type ChannelError struct{ err error }

// Error returns the ping error's message.
func (e *ChannelError) Error() string { return e.err.Error() }

// Unwrap returns the ping error.
func (e *ChannelError) Unwrap() error { return e.err }

// Fd returns the ping's eventfd.
func (r *PriorityQueueCalloopReceiver[T]) Fd() int { return r.source.Fd() }

// Interest returns InterestRead.
func (r *PriorityQueueCalloopReceiver[T]) Interest() Interest { return InterestRead }

// ProcessEvents delivers every queued item, then Closed and removal once the senders are gone.
func (r *PriorityQueueCalloopReceiver[T]) ProcessEvents(readiness Readiness, callback func(ChannelEvent[T])) (PostAction, error) {
	clearReadiness := false
	disconnected := false
	action, err := r.source.ProcessEvents(readiness, func(struct{}) {
		isEmpty := true
		receiver := r.receiver.Clone()
		defer receiver.Close()
		for item, err := range receiver.TryIter() {
			if err != nil {
				disconnected = true
				continue
			}
			callback(ChannelEvent[T]{Msg: item})
			isEmpty = false
		}
		if disconnected {
			callback(ChannelEvent[T]{Closed: true})
		}
		if isEmpty {
			clearReadiness = true
		}
	})
	if err != nil {
		return PostActionContinue, &ChannelError{err: err}
	}
	if disconnected {
		return PostActionRemove, nil
	} else if clearReadiness {
		return action, nil
	}
	r.ping.Ping()
	return PostActionContinue, nil
}

// Close drops the receiver and closes its ping source.
func (r *PriorityQueueCalloopReceiver[T]) Close() {
	r.receiver.Close()
	r.source.Close()
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
	id := goroutineID()
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
