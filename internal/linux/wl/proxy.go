//go:build linux

package wl

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/megascan/gpui-go/gpuiutil"
)

const marshalFlagDestroy = 1

// Object is a typed proxy wrapper. Generated wrappers embed Proxy, which provides the
// unexported part of the interface.
type Object interface {
	base() *Proxy
	Dispatch(opcode uint32, args Args)
}

// Proxy is one wl_proxy. Requests on a destroyed or inert proxy are ignored, as in
// wayland-backend.
type Proxy struct {
	ptr      uintptr
	iface    *Interface
	display  *Display
	version  uint32
	id       uint32
	alive    bool
	managed  bool
	userData any
}

func (p *Proxy) base() *Proxy { return p }

// Ptr returns the wl_proxy pointer, or 0 for an inert proxy.
func (p *Proxy) Ptr() uintptr { return p.ptr }

// Interface returns the interface the proxy was created with.
func (p *Proxy) Interface() *Interface { return p.iface }

// Version returns the proxy's protocol version.
func (p *Proxy) Version() uint32 { return p.version }

// ID returns the protocol object id, or 0 for an inert proxy.
func (p *Proxy) ID() uint32 { return p.id }

// IsAlive reports whether the proxy still refers to a live protocol object.
func (p *Proxy) IsAlive() bool { return p.alive }

// Display returns the connection the proxy belongs to.
func (p *Proxy) Display() *Display { return p.display }

// UserData returns the value stored with SetUserData.
func (p *Proxy) UserData() any { return p.userData }

// SetUserData stores a value with the proxy.
func (p *Proxy) SetUserData(v any) { p.userData = v }

// String returns interface@id, the form wayland-backend prints.
func (p *Proxy) String() string {
	name := "<nil>"
	if p.iface != nil {
		name = p.iface.Name
	}
	return fmt.Sprintf("%s@%d", name, p.id)
}

// Destroy destroys the local proxy without sending a request. It is idempotent and a
// no-op for proxies owned by another library.
func (p *Proxy) Destroy() {
	if !p.alive {
		return
	}
	if !p.managed {
		p.alive = false
		return
	}
	ptr := p.ptr
	p.forget()
	wl_proxy_destroy(ptr)
}

// SetQueue moves the proxy's events to q, or to the default queue when q is nil.
func (p *Proxy) SetQueue(q *EventQueue) {
	if !p.alive {
		return
	}
	var qp uintptr
	if q != nil {
		q.mustLive(p.display)
		qp = q.ptr
	}
	wl_proxy_set_queue(p.ptr, qp)
}

func (p *Proxy) forget() {
	registryMu.Lock()
	delete(registry, p.ptr)
	registryMu.Unlock()
	p.alive = false
}

var (
	registryMu sync.Mutex
	registry   = map[uintptr]Object{}
)

func lookup(ptr uintptr) Object {
	registryMu.Lock()
	defer registryMu.Unlock()
	return registry[ptr]
}

func manage(d *Display, iface *Interface, ptr uintptr) Object {
	obj := newObject(iface)
	p := obj.base()
	p.ptr = ptr
	p.iface = iface
	p.display = d
	p.version = wl_proxy_get_version(ptr)
	p.id = wl_proxy_get_id(ptr)
	p.alive = true
	p.managed = true
	registryMu.Lock()
	registry[ptr] = obj
	registryMu.Unlock()
	wl_proxy_add_dispatcher(ptr, dispatcherFn, uintptr(unsafe.Pointer(&dispatcherTag)), 0)
	return obj
}

func newObject(iface *Interface) Object {
	if iface.New == nil {
		panic(fmt.Sprintf("wl: interface %s has no constructor", iface.Name))
	}
	return iface.New()
}

func inert(iface *Interface, d *Display) Object {
	obj := newObject(iface)
	p := obj.base()
	p.iface = iface
	p.display = d
	return obj
}

// Wrap returns a non-owning wrapper for a proxy created by another library, such as a
// libwayland-cursor buffer. Its events are not dispatched and Destroy leaves it alone.
func Wrap(d *Display, ptr uintptr, iface *Interface) Object {
	obj := newObject(iface)
	p := obj.base()
	p.iface = iface
	p.display = d
	if ptr != 0 {
		p.ptr = ptr
		p.version = wl_proxy_get_version(ptr)
		p.id = wl_proxy_get_id(ptr)
		p.alive = true
	}
	return obj
}

type argKind uint8

const (
	argInt argKind = iota
	argUint
	argFixed
	argString
	argNullString
	argObject
	argNewID
	argArray
	argFD
)

// Arg is one request argument.
type Arg struct {
	kind argKind
	u    uint64
	s    string
	b    []byte
	p    *Proxy
}

// Int is an int argument.
func Int(v int32) Arg { return Arg{kind: argInt, u: uint64(uint32(v))} }

// Uint is a uint argument.
func Uint(v uint32) Arg { return Arg{kind: argUint, u: uint64(v)} }

// FixedArg is a fixed argument converted from a float64.
func FixedArg(v float64) Arg {
	return Arg{kind: argFixed, u: uint64(uint32(FixedFromFloat64(v)))}
}

// String is a non-null string argument.
func String(s string) Arg { return Arg{kind: argString, s: s} }

// NullableString is a string argument that may be null.
func NullableString(s gpuiutil.Option[string]) Arg {
	v, ok := s.Get()
	if !ok {
		return Arg{kind: argNullString}
	}
	return Arg{kind: argString, s: v}
}

// Array is an array argument.
func Array(b []byte) Arg { return Arg{kind: argArray, b: b} }

// FD is a file descriptor argument. libwayland duplicates it, the caller keeps ownership.
func FD(fd int) Arg { return Arg{kind: argFD, u: uint64(uint32(int32(fd)))} }

// NewID is the placeholder for the new object of a constructor request.
func NewID() Arg { return Arg{kind: argNewID} }

// ObjectArg is an object argument; a nil o sends a null object.
func ObjectArg[T any, PT interface {
	*T
	Object
}](o PT) Arg {
	if o == nil {
		return Arg{kind: argObject}
	}
	return Arg{kind: argObject, p: o.base()}
}

// AnyObjectArg is an object argument of an untyped object; nil sends a null object.
func AnyObjectArg(o Object) Arg {
	if o == nil {
		return Arg{kind: argObject}
	}
	return Arg{kind: argObject, p: o.base()}
}

type cArray struct {
	size  uintptr
	alloc uintptr
	data  unsafe.Pointer
}

// Marshal sends request opcode.
func (p *Proxy) Marshal(opcode uint32, args ...Arg) {
	p.marshal(opcode, nil, 0, args)
}

// MarshalConstructor sends constructor request opcode and returns the new object, an
// inert one when the parent or an argument is no longer alive.
func (p *Proxy) MarshalConstructor(opcode uint32, iface *Interface, version uint32, args ...Arg) Object {
	return p.marshal(opcode, iface, version, args)
}

func (p *Proxy) marshal(opcode uint32, child *Interface, childVersion uint32, args []Arg) Object {
	if int(opcode) >= len(p.iface.Methods) {
		panic(fmt.Sprintf("Unknown opcode %d for object %s@%d.", opcode, p.iface.Name, p.id))
	}
	msg := &p.iface.Methods[opcode]
	if !p.alive {
		if child != nil {
			return inert(child, p.display)
		}
		return nil
	}
	if p.version != 0 && msg.Since > p.version {
		panic(fmt.Sprintf("Request %s@%d.%s requires version %d but the object has version %d.", p.iface.Name, p.id, msg.Name, msg.Since, p.version))
	}
	codes := msg.codes()
	if len(codes) != len(args) {
		panic(fmt.Sprintf("Unexpected signature for request %s@%d.%s: expected %q, got %d arguments.", p.iface.Name, p.id, msg.Name, msg.Signature, len(args)))
	}
	if child != nil && childVersion == 0 {
		childVersion = p.version
	}
	var pin runtime.Pinner
	defer pin.Unpin()
	vals := make([]uint64, len(args)+1)
	for k := range args {
		a := &args[k]
		switch a.kind {
		case argInt, argUint, argFixed, argFD:
			vals[k] = a.u
		case argString:
			b, err := cStringChecked(a.s)
			if err != nil {
				panic(fmt.Sprintf("Request %s@%d.%s: string argument %d: %v.", p.iface.Name, p.id, msg.Name, k, err))
			}
			pin.Pin(&b[0])
			vals[k] = uint64(uintptr(unsafe.Pointer(&b[0])))
		case argNullString:
			vals[k] = 0
		case argArray:
			arr := &cArray{size: uintptr(len(a.b)), alloc: uintptr(len(a.b))}
			if len(a.b) > 0 {
				pin.Pin(&a.b[0])
				arr.data = unsafe.Pointer(&a.b[0])
			}
			pin.Pin(arr)
			vals[k] = uint64(uintptr(unsafe.Pointer(arr)))
		case argObject:
			if a.p != nil {
				if !a.p.alive {
					if child != nil {
						return inert(child, p.display)
					}
					return nil
				}
				if k < len(msg.Types) && msg.Types[k] != nil && !sameInterface(msg.Types[k], a.p.iface) {
					panic(fmt.Sprintf("Request %s@%d.%s expects an argument of interface %s but %s was provided instead.", p.iface.Name, p.id, msg.Name, msg.Types[k].Name, a.p.iface.Name))
				}
				vals[k] = uint64(a.p.ptr)
			} else {
				if !codes[k].nullable {
					panic(fmt.Sprintf("Request %s@%d.%s expects an non-null object argument.", p.iface.Name, p.id, msg.Name))
				}
				vals[k] = 0
			}
		case argNewID:
			vals[k] = 0
		}
	}
	var flags uint32
	if msg.Destructor {
		flags |= marshalFlagDestroy
	}
	var childC uintptr
	version := p.version
	if child != nil {
		childC = child.C()
		version = childVersion
	}
	ptr := p.ptr
	if msg.Destructor {
		p.forget()
	}
	ret := wl_proxy_marshal_array_flags(ptr, opcode, childC, version, flags, unsafe.Pointer(&vals[0]))
	runtime.KeepAlive(args)
	if child == nil {
		return nil
	}
	if ret == 0 {
		panic("[wayland-backend-sys] libwayland reported an allocation failure.")
	}
	return manage(p.display, child, ret)
}

func sameInterface(a, b *Interface) bool {
	return a == b || a.Name == b.Name
}
