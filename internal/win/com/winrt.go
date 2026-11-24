//go:build windows

package com

import (
	"errors"
	"unicode/utf16"
	"unsafe"
)

var (
	procRoActivateInstance        = combase.NewProc("RoActivateInstance")
	procRoGetActivationFactory    = combase.NewProc("RoGetActivationFactory")
	procWindowsCreateString       = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString       = combase.NewProc("WindowsDeleteString")
	procWindowsGetStringRawBuffer = combase.NewProc("WindowsGetStringRawBuffer")
)

// HSTRING is an immutable Windows Runtime string; 0 is the empty string.
type HSTRING uintptr

// NewHSTRING creates an HSTRING holding s; Delete frees it.
func NewHSTRING(s string) (HSTRING, error) {
	u := utf16.Encode([]rune(s))
	if len(u) == 0 {
		return 0, nil
	}
	var h HSTRING
	r, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)), uintptr(unsafe.Pointer(&h)))
	if err := HRESULTFromUintptr(r).Err(); err != nil {
		return 0, err
	}
	return h, nil
}

// String converts h to a Go string.
func (h HSTRING) String() string {
	if h == 0 {
		return ""
	}
	var n uint32
	r, _, _ := procWindowsGetStringRawBuffer.Call(uintptr(h), uintptr(unsafe.Pointer(&n)))
	if r == 0 || n == 0 {
		return ""
	}
	return string(utf16.Decode(unsafe.Slice((*uint16)(PtrFromUintptr(r)), n)))
}

// Delete frees h; it is the drop of a windows-rs HSTRING. 0 is ignored.
func (h HSTRING) Delete() {
	if h == 0 {
		return
	}
	procWindowsDeleteString.Call(uintptr(h))
}

func withClassID(classID string, f func(HSTRING) HRESULT) error {
	name, err := NewHSTRING(classID)
	if err != nil {
		return err
	}
	defer name.Delete()
	hr := f(name)
	if hr == CO_E_NOTINITIALIZED {
		if _, err := CoIncrementMTAUsage(); err != nil {
			return errors.Join(hr, err)
		}
		hr = f(name)
	}
	return hr.Err()
}

// RoGetActivationFactory returns the activation factory interface iid of a runtime class. Like
// windows-rs, it joins the multithreaded apartment and retries when COM is not initialized.
func RoGetActivationFactory(classID string, iid *GUID) (*IUnknown, error) {
	var out *IUnknown
	err := withClassID(classID, func(name HSTRING) HRESULT {
		r, _, _ := procRoGetActivationFactory.Call(uintptr(name), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
		return HRESULTFromUintptr(r)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RoActivateInstance activates a runtime class with its default constructor, as windows-rs's
// generated `Class::new()` does, joining the multithreaded apartment when COM is not initialized.
func RoActivateInstance(classID string) (*IInspectable, error) {
	var out *IInspectable
	err := withClassID(classID, func(name HSTRING) HRESULT {
		r, _, _ := procRoActivateInstance.Call(uintptr(name), uintptr(unsafe.Pointer(&out)))
		return HRESULTFromUintptr(r)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
