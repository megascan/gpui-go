//go:build windows

package com

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ole32   = windows.NewLazySystemDLL("ole32.dll")
	oleaut  = windows.NewLazySystemDLL("oleaut32.dll")
	combase = windows.NewLazySystemDLL("combase.dll")

	procCoCreateInstance     = ole32.NewProc("CoCreateInstance")
	procCoIncrementMTAUsage  = ole32.NewProc("CoIncrementMTAUsage")
	procCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	procCoTaskMemAlloc       = ole32.NewProc("CoTaskMemAlloc")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
	procCoUninitialize       = ole32.NewProc("CoUninitialize")
	procOleInitialize        = ole32.NewProc("OleInitialize")
	procOleUninitialize      = ole32.NewProc("OleUninitialize")
	procCoWaitForMultipleHnd = ole32.NewProc("CoWaitForMultipleHandles")
)

// CoInitializeEx flags and CoCreateInstance contexts.
const (
	COINIT_MULTITHREADED     = 0x0
	COINIT_APARTMENTTHREADED = 0x2
	COINIT_DISABLE_OLE1DDE   = 0x4
	COINIT_SPEED_OVER_MEMORY = 0x8

	CLSCTX_INPROC_SERVER  = 0x1
	CLSCTX_INPROC_HANDLER = 0x2
	CLSCTX_LOCAL_SERVER   = 0x4
	CLSCTX_REMOTE_SERVER  = 0x10
	CLSCTX_ALL            = CLSCTX_INPROC_SERVER | CLSCTX_INPROC_HANDLER | CLSCTX_LOCAL_SERVER | CLSCTX_REMOTE_SERVER
)

// CoInitializeEx initializes COM on the calling thread. S_FALSE (already initialized) is success.
func CoInitializeEx(coinit uint32) error {
	r, _, _ := procCoInitializeEx.Call(0, uintptr(coinit))
	return HRESULTFromUintptr(r).Err()
}

// CoUninitialize balances a successful CoInitializeEx.
func CoUninitialize() {
	procCoUninitialize.Call()
}

// OleInitialize initializes OLE (an STA plus clipboard and drag and drop) on the calling thread.
func OleInitialize() error {
	r, _, _ := procOleInitialize.Call(0)
	return HRESULTFromUintptr(r).Err()
}

// OleUninitialize balances a successful OleInitialize.
func OleUninitialize() {
	procOleUninitialize.Call()
}

// CoIncrementMTAUsage keeps the multithreaded apartment alive and returns its cookie.
func CoIncrementMTAUsage() (uintptr, error) {
	var cookie uintptr
	r, _, _ := procCoIncrementMTAUsage.Call(uintptr(unsafe.Pointer(&cookie)))
	return cookie, HRESULTFromUintptr(r).Err()
}

// CoCreateInstance creates an object of class clsid and returns its interface iid.
func CoCreateInstance(clsid *GUID, outer *IUnknown, context uint32, iid *GUID) (*IUnknown, error) {
	var out *IUnknown
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(clsid)), uintptr(unsafe.Pointer(outer)), uintptr(context), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if err := HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateInstance is CoCreateInstance returning the interface as a *T.
func CreateInstance[T any](clsid *GUID, context uint32, iid *GUID) (*T, error) {
	out, err := CoCreateInstance(clsid, nil, context, iid)
	if err != nil {
		return nil, err
	}
	return (*T)(unsafe.Pointer(out)), nil
}

// CoTaskMemAlloc allocates size bytes of COM task memory; it returns nil when out of memory.
func CoTaskMemAlloc(size uintptr) unsafe.Pointer {
	r, _, _ := procCoTaskMemAlloc.Call(size)
	return PtrFromUintptr(r)
}

// CoTaskMemFree frees COM task memory; nil is ignored.
func CoTaskMemFree(p unsafe.Pointer) {
	procCoTaskMemFree.Call(uintptr(p))
}

// CoWaitForMultipleHandles waits for handles while pumping COM calls and returns the index of
// the signalled handle.
func CoWaitForMultipleHandles(flags uint32, timeout uint32, handles []windows.Handle) (uint32, error) {
	var index uint32
	var first *windows.Handle
	if len(handles) > 0 {
		first = &handles[0]
	}
	r, _, _ := procCoWaitForMultipleHnd.Call(uintptr(flags), uintptr(timeout), uintptr(len(handles)), uintptr(unsafe.Pointer(first)), uintptr(unsafe.Pointer(&index)))
	return index, HRESULTFromUintptr(r).Err()
}

// TaskMemUTF16 copies a NUL-terminated UTF-16 string from COM task memory (as returned by shell
// APIs) and frees it.
func TaskMemUTF16(p *uint16) string {
	if p == nil {
		return ""
	}
	s := windows.UTF16PtrToString(p)
	CoTaskMemFree(unsafe.Pointer(p))
	return s
}
