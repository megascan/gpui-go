//go:build windows

package com

import (
	"slices"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Vtable is a native vtable shared by every Go object that implements one COM interface (and the
// interfaces it derives from). Its memory is never freed.
type Vtable struct {
	iids  []GUID
	table unsafe.Pointer
}

// NewVtable builds a vtable whose first three slots are the shared IUnknown thunks, followed by
// methods: thunks made once with windows.NewCallback whose first parameter is `this *IUnknown`.
// QueryInterface answers IID_IUnknown and iids (the interface and its bases) with this vtable.
func NewVtable(iids []GUID, methods ...uintptr) *Vtable {
	thunks := iunknownThunks()
	n := IUnknownSlots + len(methods)
	table := CoTaskMemAlloc(uintptr(n) * ptrSize)
	if table == nil {
		panic("com: out of memory allocating a vtable")
	}
	slots := unsafe.Slice((*uintptr)(table), n)
	slots[SlotQueryInterface] = thunks.queryInterface
	slots[SlotAddRef] = thunks.addRef
	slots[SlotRelease] = thunks.release
	copy(slots[IUnknownSlots:], methods)
	return &Vtable{iids: slices.Clone(iids), table: table}
}

// LazyVtable builds a Vtable on first use, so callback thunks are created once per process.
type LazyVtable struct {
	once  sync.Once
	build func() *Vtable
	vt    *Vtable
}

// NewLazyVtable returns a LazyVtable that calls build once.
func NewLazyVtable(build func() *Vtable) *LazyVtable {
	return &LazyVtable{build: build}
}

// Get returns the vtable, building it on the first call.
func (l *LazyVtable) Get() *Vtable {
	l.once.Do(func() { l.vt = l.build() })
	return l.vt
}

// InterfaceResolver lets an object's implementation answer QueryInterface itself: it returns the
// index of the vtable to hand out for iid, or false to fall back to the vtables' iid lists.
type InterfaceResolver interface {
	ResolveInterface(iid *GUID) (int, bool)
}

// FinalReleaser is implemented by an object's implementation that must run code when the last
// reference is released.
type FinalReleaser interface {
	FinalRelease()
}

// Object is a COM object implemented in Go. Its interface pointers point to native memory, one
// vtable pointer per interface, so native code never holds a Go pointer; the Go side is found by
// address. It lives until its reference count reaches zero.
type Object struct {
	slots   unsafe.Pointer
	vtables []*Vtable
	refs    atomic.Int32
	impl    any
}

var objects sync.Map

// NewObject creates an object with one reference, implementing the interface of each vtable.
// Interface(0) is its identity (IUnknown) pointer.
func NewObject(impl any, vtables ...*Vtable) *Object {
	if len(vtables) == 0 {
		panic("com: an object needs at least one vtable")
	}
	slots := CoTaskMemAlloc(uintptr(len(vtables)) * ptrSize)
	if slots == nil {
		panic("com: out of memory allocating an object")
	}
	o := &Object{slots: slots, vtables: slices.Clone(vtables), impl: impl}
	o.refs.Store(1)
	for i, vt := range vtables {
		*(*unsafe.Pointer)(unsafe.Add(slots, uintptr(i)*ptrSize)) = vt.table
		objects.Store(uintptr(slots)+uintptr(i)*ptrSize, o)
	}
	return o
}

// Interface returns the interface pointer of vtable i, without adding a reference.
func (o *Object) Interface(i int) *IUnknown {
	if i < 0 || i >= len(o.vtables) {
		panic("com: interface index out of range")
	}
	return (*IUnknown)(unsafe.Add(o.slots, uintptr(i)*ptrSize))
}

// Impl returns the Go implementation.
func (o *Object) Impl() any {
	return o.impl
}

// AddRef adds a reference and returns the new count.
func (o *Object) AddRef() uint32 {
	return uint32(o.refs.Add(1))
}

// Release drops a reference and returns the new count; the last one frees the object.
func (o *Object) Release() uint32 {
	n := o.refs.Add(-1)
	switch {
	case n == 0:
		o.destroy()
	case n < 0:
		panic("com: object released more times than referenced")
	}
	return uint32(n)
}

// RefCount returns the current reference count.
func (o *Object) RefCount() uint32 {
	return uint32(o.refs.Load())
}

func (o *Object) destroy() {
	for i := range o.vtables {
		objects.Delete(uintptr(o.slots) + uintptr(i)*ptrSize)
	}
	CoTaskMemFree(o.slots)
	o.slots = nil
	if f, ok := o.impl.(FinalReleaser); ok {
		f.FinalRelease()
	}
}

func (o *Object) queryInterface(iid *GUID) (*IUnknown, bool) {
	if r, ok := o.impl.(InterfaceResolver); ok {
		if i, ok := r.ResolveInterface(iid); ok {
			return o.Interface(i), true
		}
	}
	if *iid == IID_IUnknown {
		return o.Interface(0), true
	}
	for i, vt := range o.vtables {
		if slices.Contains(vt.iids, *iid) {
			return o.Interface(i), true
		}
	}
	return nil, false
}

// ObjectOf returns the Go object behind an interface pointer, or nil if it is not one.
func ObjectOf(this *IUnknown) *Object {
	v, ok := objects.Load(uintptr(unsafe.Pointer(this)))
	if !ok {
		return nil
	}
	return v.(*Object)
}

// ImplOf returns the Go implementation behind an interface pointer as a T; ok is false if this
// is not a Go object or its implementation is not a T.
func ImplOf[T any](this *IUnknown) (T, bool) {
	o := ObjectOf(this)
	if o == nil {
		var zero T
		return zero, false
	}
	impl, ok := o.impl.(T)
	return impl, ok
}

type unknownThunks struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
}

var (
	unknownThunksOnce sync.Once
	unknownThunksVal  unknownThunks
)

func iunknownThunks() unknownThunks {
	unknownThunksOnce.Do(func() {
		unknownThunksVal = unknownThunks{
			queryInterface: windows.NewCallback(queryInterfaceThunk),
			addRef:         windows.NewCallback(addRefThunk),
			release:        windows.NewCallback(releaseThunk),
		}
	})
	return unknownThunksVal
}

func queryInterfaceThunk(this *IUnknown, iid *GUID, out **IUnknown) uintptr {
	if out == nil {
		return E_POINTER.Uintptr()
	}
	*out = nil
	if iid == nil {
		return E_POINTER.Uintptr()
	}
	o := ObjectOf(this)
	if o == nil {
		return E_NOINTERFACE.Uintptr()
	}
	p, ok := o.queryInterface(iid)
	if !ok {
		return E_NOINTERFACE.Uintptr()
	}
	o.AddRef()
	*out = p
	return S_OK.Uintptr()
}

func addRefThunk(this *IUnknown) uintptr {
	o := ObjectOf(this)
	if o == nil {
		return 0
	}
	return uintptr(o.AddRef())
}

func releaseThunk(this *IUnknown) uintptr {
	o := ObjectOf(this)
	if o == nil {
		return 0
	}
	return uintptr(o.Release())
}
