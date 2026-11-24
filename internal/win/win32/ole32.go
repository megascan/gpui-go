//go:build windows

package win32

import (
	"unsafe"

	"github.com/megascan/gpui-go/internal/win/com"
)

var (
	procRegisterDragDrop = ole32.NewProc("RegisterDragDrop")
	procReleaseStgMedium = ole32.NewProc("ReleaseStgMedium")
	procRevokeDragDrop   = ole32.NewProc("RevokeDragDrop")
)

// RegisterDragDrop registers target, an IDropTarget, for a window.
func RegisterDragDrop(hwnd HWND, target *com.IUnknown) error {
	r, _, _ := procRegisterDragDrop.Call(uintptr(hwnd), uintptr(unsafe.Pointer(target)))
	return hresult(r)
}

// RevokeDragDrop revokes a window's drop target.
func RevokeDragDrop(hwnd HWND) error {
	r, _, _ := procRevokeDragDrop.Call(uintptr(hwnd))
	return hresult(r)
}

// ReleaseStgMedium frees a storage medium.
func ReleaseStgMedium(medium *STGMEDIUM) {
	procReleaseStgMedium.Call(uintptr(unsafe.Pointer(medium)))
}
