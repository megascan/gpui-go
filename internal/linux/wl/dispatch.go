//go:build linux

package wl

import (
	"fmt"
	"log/slog"
	"syscall"

	"github.com/megascan/gpui-go/gpuiutil"
)

var (
	dispatcherFn  uintptr
	dispatcherTag byte
	pendingPanic  any
	hasPanic      bool
)

const dispatchFailed = ^uintptr(0)

// Args gives typed access to the wl_argument array of the event being dispatched. It is
// valid only during Dispatch; strings and arrays are copied out.
type Args struct {
	ptr     uintptr
	created []Object
}

func (a Args) raw(i int) uint64 { return load64(a.ptr + uintptr(i)*8) }

// Int returns argument i as an int.
func (a Args) Int(i int) int32 { return int32(uint32(a.raw(i))) }

// Uint returns argument i as a uint.
func (a Args) Uint(i int) uint32 { return uint32(a.raw(i)) }

// Fixed returns argument i as a float64.
func (a Args) Fixed(i int) float64 { return Fixed(int32(uint32(a.raw(i)))).Float64() }

// String returns argument i as a string; null reads as "".
func (a Args) String(i int) string { return goString(uintptr(a.raw(i))) }

// NullableString returns argument i, None when it is null.
func (a Args) NullableString(i int) gpuiutil.Option[string] {
	p := uintptr(a.raw(i))
	if p == 0 {
		return gpuiutil.None[string]()
	}
	return gpuiutil.Some(goString(p))
}

// Array returns a copy of array argument i.
func (a Args) Array(i int) []byte {
	arr := uintptr(a.raw(i))
	if arr == 0 {
		return []byte{}
	}
	size := loadPtr(arr)
	data := loadPtr(arr + 16)
	return goBytes(data, int(size))
}

// FD returns fd argument i. The receiver owns it.
func (a Args) FD(i int) int { return int(int32(uint32(a.raw(i)))) }

// Object returns object argument i, or nil when it is null or not managed here.
func (a Args) Object(i int) Object {
	p := uintptr(a.raw(i))
	if p == 0 {
		return nil
	}
	return lookup(p)
}

// NewID returns the object created by new_id argument i.
func (a Args) NewID(i int) Object {
	if i < len(a.created) {
		return a.created[i]
	}
	return nil
}

// CloseFD closes a received file descriptor nobody handled.
func CloseFD(fd int) {
	if fd >= 0 {
		_ = syscall.Close(fd)
	}
}

func dispatcher(impl, target, opcode, msg, args uintptr) uintptr {
	obj := lookup(target)
	if obj == nil {
		discardArgs(msg, args)
		return 0
	}
	p := obj.base()
	d := p.display
	if d != nil && d.violation != nil {
		discardArgs(msg, args)
		return 0
	}
	op := uint32(opcode)
	if int(op) >= len(p.iface.Events) {
		protocolViolation(d, fmt.Sprintf("Unknown event opcode %d for interface %s.", op, p.iface.Name))
		discardArgs(msg, args)
		return dispatchFailed
	}
	ev := &p.iface.Events[op]
	a := Args{ptr: args}
	codes := ev.codes()
	for k, code := range codes {
		if code.kind != 'o' {
			continue
		}
		ptr := uintptr(a.raw(k))
		if ptr == 0 {
			continue
		}
		o := lookup(ptr)
		if o == nil {
			continue
		}
		if k < len(ev.Types) && ev.Types[k] != nil && !sameInterface(ev.Types[k], o.base().iface) {
			protocolViolation(d, fmt.Sprintf("Received object %s@%d in %s.%s but expected interface %s.", o.base().iface.Name, o.base().id, p.iface.Name, ev.Name, ev.Types[k].Name))
			discardArgs(msg, args)
			return dispatchFailed
		}
	}
	for k, code := range codes {
		if code.kind != 'n' {
			continue
		}
		ptr := uintptr(a.raw(k))
		if ptr == 0 {
			continue
		}
		var child *Interface
		if k < len(ev.Types) {
			child = ev.Types[k]
		}
		if child == nil {
			slog.Warn(fmt.Sprintf("Event %s.%d creates an anonymous object.", p.iface.Name, op))
			wl_proxy_destroy(ptr)
			continue
		}
		if a.created == nil {
			a.created = make([]Object, len(codes))
		}
		a.created[k] = manage(d, child, ptr)
	}
	callListener(obj, op, a)
	if ev.Destructor {
		p.Destroy()
	}
	return 0
}

func callListener(obj Object, op uint32, a Args) {
	defer func() {
		if r := recover(); r != nil && !hasPanic {
			pendingPanic = r
			hasPanic = true
		}
	}()
	obj.Dispatch(op, a)
}

func protocolViolation(d *Display, msg string) {
	slog.Error(msg)
	if d != nil && d.violation == nil {
		d.violation = fmt.Errorf("%w: %s", ErrProtocolViolation, msg)
	}
}

func discardArgs(msg, args uintptr) {
	if msg == 0 || args == 0 {
		return
	}
	sigPtr := loadPtr(msg + 8)
	if sigPtr == 0 {
		return
	}
	a := Args{ptr: args}
	for k, code := range parseSignature(goString(sigPtr)) {
		switch code.kind {
		case 'h':
			CloseFD(a.FD(k))
		case 'n':
			ptr := uintptr(a.raw(k))
			if ptr == 0 {
				continue
			}
			if o := lookup(ptr); o != nil {
				o.base().Destroy()
			} else {
				wl_proxy_destroy(ptr)
			}
		}
	}
}

// DiscardNewID releases an object an event created that no listener took: it sends the
// interface's argument-less destructor request when the bound version has one, and
// destroys the local proxy otherwise.
func DiscardNewID(o Object) {
	if o == nil {
		return
	}
	p := o.base()
	if !p.alive {
		return
	}
	for k := range p.iface.Methods {
		m := &p.iface.Methods[k]
		if m.Destructor && m.Signature == "" && m.Since <= p.version {
			p.Marshal(uint32(k))
			return
		}
	}
	p.Destroy()
}

func rethrow() {
	if hasPanic {
		r := pendingPanic
		pendingPanic = nil
		hasPanic = false
		panic(r)
	}
}
