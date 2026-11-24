//go:build windows

package win32

import (
	"unsafe"

	"github.com/megascan/gpui-go/internal/win/com"
)

var (
	procDragQueryFileW                          = shell32.NewProc("DragQueryFileW")
	procSHCreateItemFromParsingName             = shell32.NewProc("SHCreateItemFromParsingName")
	procSetCurrentProcessExplicitAppUserModelID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	procShellExecuteW                           = shell32.NewProc("ShellExecuteW")
)

// DragQueryFileW copies dropped file index into buf and returns its length; index 0xFFFFFFFF
// returns the file count and a nil buf the length needed.
func DragQueryFileW(drop HDROP, index uint32, buf []uint16) uint32 {
	var first *uint16
	if len(buf) > 0 {
		first = &buf[0]
	}
	r, _, _ := procDragQueryFileW.Call(uintptr(drop), uintptr(index), uintptr(unsafe.Pointer(first)), uintptr(len(buf)))
	return uint32(r)
}

// SHCreateItemFromParsingName creates a shell item, interface iid, from a path.
func SHCreateItemFromParsingName(path *uint16, iid *GUID) (*com.IUnknown, error) {
	var out *com.IUnknown
	r, _, _ := procSHCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if err := hresult(r); err != nil {
		return nil, err
	}
	return out, nil
}

// SetCurrentProcessExplicitAppUserModelID sets the process's AppUserModelID.
func SetCurrentProcessExplicitAppUserModelID(id *uint16) error {
	r, _, _ := procSetCurrentProcessExplicitAppUserModelID.Call(uintptr(unsafe.Pointer(id)))
	return hresult(r)
}

// ShellExecuteW runs an operation on a file; a result above 32 is success.
func ShellExecuteW(hwnd HWND, operation, file, parameters, directory *uint16, showCmd int32) HINSTANCE {
	r, _, _ := procShellExecuteW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(operation)), uintptr(unsafe.Pointer(file)), uintptr(unsafe.Pointer(parameters)), uintptr(unsafe.Pointer(directory)), uintptr(showCmd))
	return HINSTANCE(r)
}
