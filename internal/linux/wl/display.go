//go:build linux

package wl

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
)

// Display is a wl_display connection.
type Display struct {
	ptr       uintptr
	object    Object
	closed    bool
	violation error
	queues    map[*EventQueue]struct{}
}

// EventQueue is a wl_event_queue.
type EventQueue struct {
	ptr     uintptr
	display *Display
}

// ProtocolError is a fatal protocol error the compositor reported.
type ProtocolError struct {
	Code      uint32
	Interface string
	ID        uint32
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("wl: protocol error %d on %s@%d", e.Code, e.Interface, e.ID)
}

// ErrConnect is returned when wl_display_connect fails.
var ErrConnect = errors.New("wl: could not connect to a Wayland display")

// ErrProtocolViolation is wrapped by the error a connection reports once the dispatcher
// received an event that does not match the protocol. The connection is dead from then on.
var ErrProtocolViolation = errors.New("wl: protocol violation")

// Connect connects to the display named name, or to $WAYLAND_DISPLAY when name is empty.
func Connect(name string) (*Display, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	var arg *byte
	var buf []byte
	if name != "" {
		var err error
		if buf, err = cStringChecked(name); err != nil {
			return nil, fmt.Errorf("wl: display name: %w", err)
		}
		arg = &buf[0]
	}
	ptr := wl_display_connect(arg)
	runtime.KeepAlive(buf)
	if ptr == 0 {
		if name != "" {
			return nil, fmt.Errorf("%w %q", ErrConnect, name)
		}
		return nil, ErrConnect
	}
	return &Display{ptr: ptr}, nil
}

// ConnectToFD connects over an already open socket; libwayland takes ownership of fd.
func ConnectToFD(fd int) (*Display, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	ptr := wl_display_connect_to_fd(int32(fd))
	if ptr == 0 {
		return nil, ErrConnect
	}
	return &Display{ptr: ptr}, nil
}

// Ptr returns the wl_display pointer.
func (d *Display) Ptr() uintptr {
	d.mustOpen()
	return d.ptr
}

// Object returns the wrapper of the wl_display proxy itself, creating it with iface on
// first use. libwayland handles the display's own events.
func (d *Display) Object(iface *Interface) Object {
	d.mustOpen()
	if d.object == nil {
		obj := newObject(iface)
		p := obj.base()
		p.ptr = d.ptr
		p.iface = iface
		p.display = d
		p.version = 1
		p.id = 1
		p.alive = true
		d.object = obj
	}
	return d.object
}

// FD returns the connection's file descriptor.
func (d *Display) FD() int {
	d.mustOpen()
	return int(wl_display_get_fd(d.ptr))
}

// Dispatch reads and dispatches events on the default queue, blocking when none are queued.
func (d *Display) Dispatch() (int, error) {
	d.mustOpen()
	return d.call(func() int32 { return wl_display_dispatch(d.ptr) })
}

// DispatchPending dispatches already queued events on the default queue.
func (d *Display) DispatchPending() (int, error) {
	d.mustOpen()
	return d.call(func() int32 { return wl_display_dispatch_pending(d.ptr) })
}

// DispatchQueue reads and dispatches events on q, blocking when none are queued.
func (d *Display) DispatchQueue(q *EventQueue) (int, error) {
	d.mustOpen()
	q.mustLive(d)
	return d.call(func() int32 { return wl_display_dispatch_queue(d.ptr, q.ptr) })
}

// DispatchQueuePending dispatches already queued events on q.
func (d *Display) DispatchQueuePending(q *EventQueue) (int, error) {
	d.mustOpen()
	q.mustLive(d)
	return d.call(func() int32 { return wl_display_dispatch_queue_pending(d.ptr, q.ptr) })
}

// Roundtrip blocks until the compositor has processed every request sent so far,
// dispatching the default queue meanwhile.
func (d *Display) Roundtrip() (int, error) {
	d.mustOpen()
	return d.call(func() int32 { return wl_display_roundtrip(d.ptr) })
}

// RoundtripQueue is Roundtrip for q.
func (d *Display) RoundtripQueue(q *EventQueue) (int, error) {
	d.mustOpen()
	q.mustLive(d)
	return d.call(func() int32 { return wl_display_roundtrip_queue(d.ptr, q.ptr) })
}

// Flush sends buffered requests. It returns syscall.EAGAIN when the socket is full.
func (d *Display) Flush() error {
	d.mustOpen()
	_, err := d.call(func() int32 { return wl_display_flush(d.ptr) })
	return err
}

// PrepareRead announces the intention to read events from the socket. It returns false
// when the default queue still has events to dispatch first.
func (d *Display) PrepareRead() bool {
	d.mustOpen()
	return wl_display_prepare_read(d.ptr) == 0
}

// PrepareReadQueue is PrepareRead for q.
func (d *Display) PrepareReadQueue(q *EventQueue) bool {
	d.mustOpen()
	q.mustLive(d)
	return wl_display_prepare_read_queue(d.ptr, q.ptr) == 0
}

// ReadEvents reads events from the socket after a successful PrepareRead.
func (d *Display) ReadEvents() error {
	d.mustOpen()
	_, err := d.call(func() int32 { return wl_display_read_events(d.ptr) })
	return err
}

// CancelRead releases a PrepareRead without reading.
func (d *Display) CancelRead() {
	d.mustOpen()
	wl_display_cancel_read(d.ptr)
}

// Err returns the connection's fatal error, or nil.
func (d *Display) Err() error {
	d.mustOpen()
	if d.violation != nil {
		return d.violation
	}
	code := wl_display_get_error(d.ptr)
	if code == 0 {
		return nil
	}
	if syscall.Errno(code) == syscall.EPROTO {
		var iface uintptr
		var id uint32
		pc := wl_display_get_protocol_error(d.ptr, &iface, &id)
		name := ""
		if iface != 0 {
			name = goString(loadPtr(iface))
		}
		return &ProtocolError{Code: pc, Interface: name, ID: id}
	}
	return syscall.Errno(code)
}

// CreateQueue creates an event queue.
func (d *Display) CreateQueue() *EventQueue {
	d.mustOpen()
	q := &EventQueue{ptr: wl_display_create_queue(d.ptr), display: d}
	if q.ptr == 0 {
		panic("wl: wl_display_create_queue failed")
	}
	if d.queues == nil {
		d.queues = map[*EventQueue]struct{}{}
	}
	d.queues[q] = struct{}{}
	return q
}

// Destroy destroys the queue. Proxies still assigned to it must be destroyed first. It is
// idempotent, and a no-op once the display is disconnected, which destroys its queues.
func (q *EventQueue) Destroy() {
	if q.ptr == 0 || q.display.closed {
		return
	}
	delete(q.display.queues, q)
	wl_event_queue_destroy(q.ptr)
	q.ptr = 0
}

func (q *EventQueue) mustLive(d *Display) {
	if q.ptr == 0 {
		panic("wl: use of destroyed event queue")
	}
	if q.display != d {
		panic("wl: event queue belongs to another display")
	}
}

// Disconnect destroys every proxy still managed on the connection and closes it. It is
// idempotent.
func (d *Display) Disconnect() {
	if d.closed {
		return
	}
	registryMu.Lock()
	var owned []*Proxy
	for ptr, obj := range registry {
		if p := obj.base(); p.display == d {
			owned = append(owned, p)
			delete(registry, ptr)
		}
	}
	registryMu.Unlock()
	for _, p := range owned {
		p.alive = false
		wl_proxy_destroy(p.ptr)
	}
	if d.object != nil {
		d.object.base().alive = false
	}
	for q := range d.queues {
		wl_event_queue_destroy(q.ptr)
		q.ptr = 0
	}
	d.queues = nil
	wl_display_disconnect(d.ptr)
	d.closed = true
	d.ptr = 0
}

func (d *Display) mustOpen() {
	if d.closed {
		panic("wl: use of disconnected display")
	}
}

func (d *Display) call(f func() int32) (int, error) {
	if d.violation != nil {
		return -1, d.violation
	}
	runtime.LockOSThread()
	n := f()
	var e syscall.Errno
	if n < 0 {
		e = errno()
	}
	runtime.UnlockOSThread()
	rethrow()
	return d.result(n, e)
}

func (d *Display) result(n int32, e syscall.Errno) (int, error) {
	if d.violation != nil {
		return -1, d.violation
	}
	if n >= 0 {
		return int(n), nil
	}
	if err := d.Err(); err != nil {
		return -1, err
	}
	if e != 0 {
		return -1, e
	}
	return -1, errors.New("wl: display operation failed")
}
