//go:build windows

package win32

import (
	"unsafe"
)

var (
	procCloseThreadpoolTimer        = kernel32.NewProc("CloseThreadpoolTimer")
	procCreateThread                = kernel32.NewProc("CreateThread")
	procCreateThreadpoolTimer       = kernel32.NewProc("CreateThreadpoolTimer")
	procGetCurrentPackageFullName   = kernel32.NewProc("GetCurrentPackageFullName")
	procGetModuleHandleW            = kernel32.NewProc("GetModuleHandleW")
	procGetSystemPowerStatus        = kernel32.NewProc("GetSystemPowerStatus")
	procGetUserDefaultLocaleName    = kernel32.NewProc("GetUserDefaultLocaleName")
	procGlobalAlloc                 = kernel32.NewProc("GlobalAlloc")
	procGlobalFree                  = kernel32.NewProc("GlobalFree")
	procGlobalLock                  = kernel32.NewProc("GlobalLock")
	procGlobalSize                  = kernel32.NewProc("GlobalSize")
	procGlobalUnlock                = kernel32.NewProc("GlobalUnlock")
	procLoadLibraryA                = kernel32.NewProc("LoadLibraryA")
	procPowerClearRequest           = kernel32.NewProc("PowerClearRequest")
	procPowerCreateRequest          = kernel32.NewProc("PowerCreateRequest")
	procPowerSetRequest             = kernel32.NewProc("PowerSetRequest")
	procQueryPerformanceCounter     = kernel32.NewProc("QueryPerformanceCounter")
	procQueryPerformanceFrequency   = kernel32.NewProc("QueryPerformanceFrequency")
	procSetThreadPriority           = kernel32.NewProc("SetThreadPriority")
	procSetThreadpoolTimer          = kernel32.NewProc("SetThreadpoolTimer")
	procTrySubmitThreadpoolCallback = kernel32.NewProc("TrySubmitThreadpoolCallback")
)

// GetCurrentThread returns the pseudo handle of the calling thread.
func GetCurrentThread() HANDLE {
	return HANDLE(^uintptr(1))
}

// CloseThreadpoolTimer releases a thread pool timer.
func CloseThreadpoolTimer(timer PTP_TIMER) {
	procCloseThreadpoolTimer.Call(uintptr(timer))
}

// CreateThread starts a native thread at start, a windows.NewCallback LPTHREAD_START_ROUTINE,
// with parameter.
func CreateThread(stackSize uintptr, start uintptr, parameter uintptr, flags uint32) (HANDLE, uint32, error) {
	var id uint32
	r, _, err := procCreateThread.Call(0, stackSize, start, parameter, uintptr(flags), uintptr(unsafe.Pointer(&id)))
	if r == 0 {
		return 0, 0, lastError(err)
	}
	return HANDLE(r), id, nil
}

// CreateThreadpoolTimer creates a thread pool timer; callback is a windows.NewCallback
// PTP_TIMER_CALLBACK.
func CreateThreadpoolTimer(callback uintptr, context uintptr, environ *TP_CALLBACK_ENVIRON_V3) (PTP_TIMER, error) {
	r, _, err := procCreateThreadpoolTimer.Call(callback, context, uintptr(unsafe.Pointer(environ)))
	if r == 0 {
		return 0, lastError(err)
	}
	return PTP_TIMER(r), nil
}

// GetCurrentPackageFullName copies the calling process's package full name into buf; length is
// its capacity in and the needed length out. It returns a Win32 error code.
func GetCurrentPackageFullName(length *uint32, buf []uint16) uint32 {
	var first *uint16
	if len(buf) > 0 {
		first = &buf[0]
	}
	r, _, _ := procGetCurrentPackageFullName.Call(uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(first)))
	return uint32(r)
}

// GetModuleHandleW returns a loaded module's handle, or the executable's when name is nil.
func GetModuleHandleW(name *uint16) (HMODULE, error) {
	r, _, err := procGetModuleHandleW.Call(uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, lastError(err)
	}
	return HMODULE(r), nil
}

// GetSystemPowerStatus reads the system's power status.
func GetSystemPowerStatus(status *SYSTEM_POWER_STATUS) error {
	return boolResult(procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(status))))
}

// GetUserDefaultLocaleName copies the user's default locale name into buf and returns its length
// including the terminating NUL, or 0 on failure.
func GetUserDefaultLocaleName(buf []uint16) int32 {
	if len(buf) == 0 {
		return 0
	}
	r, _, _ := procGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return int32(r)
}

// GlobalAlloc allocates global memory.
func GlobalAlloc(flags uint32, size uintptr) (HGLOBAL, error) {
	r, _, err := procGlobalAlloc.Call(uintptr(flags), size)
	if r == 0 {
		return 0, lastError(err)
	}
	return HGLOBAL(r), nil
}

// GlobalFree frees global memory; it returns 0 on success.
func GlobalFree(mem HGLOBAL) HGLOBAL {
	r, _, _ := procGlobalFree.Call(uintptr(mem))
	return HGLOBAL(r)
}

// GlobalLock locks global memory and returns its address.
func GlobalLock(mem HGLOBAL) (unsafe.Pointer, error) {
	r, _, err := procGlobalLock.Call(uintptr(mem))
	if r == 0 {
		return nil, lastError(err)
	}
	return PtrFromUintptr(r), nil
}

// GlobalSize returns the size of global memory in bytes, or 0 on failure.
func GlobalSize(mem HGLOBAL) uintptr {
	r, _, _ := procGlobalSize.Call(uintptr(mem))
	return r
}

// GlobalUnlock unlocks global memory and reports whether it is still locked.
func GlobalUnlock(mem HGLOBAL) bool {
	r, _, _ := procGlobalUnlock.Call(uintptr(mem))
	return r != 0
}

// LoadLibraryA loads a module by its ANSI name.
func LoadLibraryA(name *byte) (HMODULE, error) {
	r, _, err := procLoadLibraryA.Call(uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, lastError(err)
	}
	return HMODULE(r), nil
}

// PowerClearRequest decrements a power request type.
func PowerClearRequest(request HANDLE, requestType int32) error {
	return boolResult(procPowerClearRequest.Call(uintptr(request), uintptr(requestType)))
}

// PowerCreateRequest creates a power request object.
func PowerCreateRequest(context *REASON_CONTEXT) (HANDLE, error) {
	r, _, err := procPowerCreateRequest.Call(uintptr(unsafe.Pointer(context)))
	if HANDLE(r) == HANDLE(^uintptr(0)) || r == 0 {
		return 0, lastError(err)
	}
	return HANDLE(r), nil
}

// PowerSetRequest increments a power request type.
func PowerSetRequest(request HANDLE, requestType int32) error {
	return boolResult(procPowerSetRequest.Call(uintptr(request), uintptr(requestType)))
}

// QueryPerformanceCounter returns the performance counter.
func QueryPerformanceCounter(count *int64) error {
	return boolResult(procQueryPerformanceCounter.Call(uintptr(unsafe.Pointer(count))))
}

// QueryPerformanceFrequency returns the performance counter's ticks per second.
func QueryPerformanceFrequency(frequency *int64) error {
	return boolResult(procQueryPerformanceFrequency.Call(uintptr(unsafe.Pointer(frequency))))
}

// SetThreadPriority sets a thread's priority.
func SetThreadPriority(thread HANDLE, priority int32) error {
	return boolResult(procSetThreadPriority.Call(uintptr(thread), uintptr(priority)))
}

// SetThreadpoolTimer starts (or with a nil due time, stops) a thread pool timer. A negative due
// time is relative, in 100-nanosecond ticks.
func SetThreadpoolTimer(timer PTP_TIMER, dueTime *FILETIME, period uint32, windowLength uint32) {
	procSetThreadpoolTimer.Call(uintptr(timer), uintptr(unsafe.Pointer(dueTime)), uintptr(period), uintptr(windowLength))
}

// TrySubmitThreadpoolCallback queues callback, a windows.NewCallback PTP_SIMPLE_CALLBACK, on the
// thread pool.
func TrySubmitThreadpoolCallback(callback uintptr, context uintptr, environ *TP_CALLBACK_ENVIRON_V3) error {
	return boolResult(procTrySubmitThreadpoolCallback.Call(callback, context, uintptr(unsafe.Pointer(environ))))
}
