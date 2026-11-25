//go:build windows

// Package win32 holds the user32, gdi32, dwmapi, shcore, imm32, ole32, shell32, kernel32,
// advapi32 and comctl32 procedures and structures that golang.org/x/sys/windows lacks.
package win32

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

// HWND is a window handle.
type HWND = windows.HWND

// HANDLE is a kernel object handle.
type HANDLE = windows.Handle

// HMODULE is a loaded module handle.
type HMODULE = windows.Handle

// HINSTANCE is a module instance handle.
type HINSTANCE = windows.Handle

// Handle types that x/sys/windows does not declare.
type (
	HMONITOR     uintptr
	HDC          uintptr
	HCURSOR      uintptr
	HICON        uintptr
	HBRUSH       uintptr
	HMENU        uintptr
	HGDIOBJ      uintptr
	HIMC         uintptr
	HGLOBAL      uintptr
	HKL          uintptr
	HPOWERNOTIFY uintptr
	HDROP        uintptr
	HWINSTA      uintptr
	PTP_TIMER    uintptr
	PTP_WORK     uintptr
)

// Message parameter and result types.
type (
	WPARAM  uintptr
	LPARAM  uintptr
	LRESULT uintptr
)

// BOOL is the Win32 32-bit boolean.
type BOOL int32

// ATOM is a registered class or string atom.
type ATOM uint16

// Bool converts a Go bool to a BOOL.
func Bool(b bool) BOOL {
	if b {
		return 1
	}
	return 0
}

// AsBool reports whether b is nonzero.
func (b BOOL) AsBool() bool {
	return b != 0
}

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	shcore   = windows.NewLazySystemDLL("shcore.dll")
	imm32    = windows.NewLazySystemDLL("imm32.dll")
	ole32    = windows.NewLazySystemDLL("ole32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")
	comctl32 = windows.NewLazySystemDLL("comctl32.dll")
)

var errUnknown = errors.New("win32: call failed without an error code")

func lastError(err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == 0 {
		return errUnknown
	}
	return err
}

func boolResult(r, _ uintptr, err error) error {
	if r == 0 {
		return lastError(err)
	}
	return nil
}

func hresult(r uintptr) error {
	return com.HRESULTFromUintptr(r).Err()
}

// PtrFromUintptr reinterprets an address of native memory returned by the system as a pointer.
// It must never be used for Go memory.
func PtrFromUintptr(addr uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

// UTF16PtrFromString converts s to a NUL-terminated UTF-16 string, panicking on an interior NUL
// as windows-rs's w! macro rejects one at compile time.
func UTF16PtrFromString(s string) *uint16 {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		panic(err)
	}
	return p
}

// MakeIntResource is MAKEINTRESOURCEW.
func MakeIntResource(id uint16) *uint16 {
	return (*uint16)(PtrFromUintptr(uintptr(id)))
}

// PackPoint passes a POINT by value as one register-sized argument.
func PackPoint(pt POINT) uintptr {
	return uintptr(uint64(uint32(pt.X)) | uint64(uint32(pt.Y))<<32)
}
