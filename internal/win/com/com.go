//go:build windows

// Package com calls COM interfaces through their vtables, converts HRESULTs to errors, wraps
// BSTR, VARIANT, SAFEARRAY and HSTRING, and implements COM interfaces in Go.
package com

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GUID is a COM GUID.
type GUID = windows.GUID

// MustGUID parses "{xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}" and panics if it is malformed.
func MustGUID(s string) GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic("com: invalid GUID " + s + ": " + err.Error())
	}
	return g
}

// Well-known interface ids.
var (
	IID_IUnknown     = MustGUID("{00000000-0000-0000-C000-000000000046}")
	IID_IInspectable = MustGUID("{AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90}")
	IID_IAgileObject = MustGUID("{94EA2B94-E9CC-49E0-C0FF-EE64CA8F5B90}")
)

const ptrSize = unsafe.Sizeof(uintptr(0))

// IUnknown is the layout of every COM object: a pointer to its vtable. Interface types embed it,
// and an interface pointer is a *IUnknown (or *T for an embedding T) to native memory.
type IUnknown struct {
	vtbl unsafe.Pointer
}

// IUnknown vtable slots.
const (
	SlotQueryInterface = 0
	SlotAddRef         = 1
	SlotRelease        = 2
	// IUnknownSlots is the number of IUnknown methods; an interface's own methods start here.
	IUnknownSlots = 3
	// IInspectableSlots is IUnknownSlots plus GetIids, GetRuntimeClassName and GetTrustLevel.
	IInspectableSlots = 6
)

// PtrFromUintptr reinterprets an address of native memory returned by the system as a pointer.
// It must never be used for Go memory.
func PtrFromUintptr(addr uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

// Method returns the address of vtable slot of the object at this.
func Method(this unsafe.Pointer, slot int) uintptr {
	vtbl := *(*unsafe.Pointer)(this)
	return *(*uintptr)(unsafe.Add(vtbl, uintptr(slot)*ptrSize))
}

// Call invokes vtable slot of the object at this with this as the first argument. Go pointers
// may be passed as uintptr(unsafe.Pointer(p)) directly in the call; they are kept alive and
// unmoved until it returns.
//
//go:uintptrescapes
func Call(this unsafe.Pointer, slot int, args ...uintptr) uintptr {
	fn := Method(this, slot)
	all := make([]uintptr, 0, len(args)+1)
	all = append(all, uintptr(this))
	all = append(all, args...)
	r, _, _ := syscall.SyscallN(fn, all...)
	return r
}

// CallHR is Call for a method returning an HRESULT, returned as an error when it failed.
//
//go:uintptrescapes
func CallHR(this unsafe.Pointer, slot int, args ...uintptr) error {
	fn := Method(this, slot)
	all := make([]uintptr, 0, len(args)+1)
	all = append(all, uintptr(this))
	all = append(all, args...)
	r, _, _ := syscall.SyscallN(fn, all...)
	return HRESULTFromUintptr(r).Err()
}

// Ptr returns u as an unsafe.Pointer for Call.
func (u *IUnknown) Ptr() unsafe.Pointer {
	return unsafe.Pointer(u)
}

// QueryInterface returns the object's interface iid, AddRef'd.
func (u *IUnknown) QueryInterface(iid *GUID) (*IUnknown, error) {
	var out *IUnknown
	if err := CallHR(u.Ptr(), SlotQueryInterface, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

// AddRef increments the object's reference count and returns the new count.
func (u *IUnknown) AddRef() uint32 {
	return uint32(Call(u.Ptr(), SlotAddRef))
}

// Release decrements the object's reference count and returns the new count. It is the drop of
// a windows-rs interface value; nil is ignored.
func (u *IUnknown) Release() uint32 {
	if u == nil {
		return 0
	}
	return uint32(Call(u.Ptr(), SlotRelease))
}

// Cast queries u for iid and returns it as a *T, the Go type of that interface.
func Cast[T any](u *IUnknown, iid *GUID) (*T, error) {
	out, err := u.QueryInterface(iid)
	if err != nil {
		return nil, err
	}
	return (*T)(unsafe.Pointer(out)), nil
}

// As reinterprets an interface pointer of type *S as the *T of a base or derived interface
// without a QueryInterface; it is only valid along the interface's inheritance chain.
func As[T, S any](p *S) *T {
	return (*T)(unsafe.Pointer(p))
}

// Unknown returns p's IUnknown view.
func Unknown[S any](p *S) *IUnknown {
	return (*IUnknown)(unsafe.Pointer(p))
}

// IInspectable is the base of every Windows Runtime interface.
type IInspectable struct {
	IUnknown
}

// GetRuntimeClassName returns the object's runtime class name.
func (i *IInspectable) GetRuntimeClassName() (string, error) {
	var name HSTRING
	if err := CallHR(i.Ptr(), 4, uintptr(unsafe.Pointer(&name))); err != nil {
		return "", err
	}
	defer name.Delete()
	return name.String(), nil
}
