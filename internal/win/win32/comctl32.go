//go:build windows

package win32

import (
	"unsafe"
)

var procTaskDialogIndirect = comctl32.NewProc("TaskDialogIndirect")

// TaskDialogIndirect shows a task dialog. config points to a TASKDIALOGCONFIG, which is
// pack(1) in commctrl.h and so is laid out by the caller.
func TaskDialogIndirect(config unsafe.Pointer, button, radioButton *int32, verificationFlagChecked *BOOL) error {
	r, _, _ := procTaskDialogIndirect.Call(uintptr(config), uintptr(unsafe.Pointer(button)), uintptr(unsafe.Pointer(radioButton)), uintptr(unsafe.Pointer(verificationFlagChecked)))
	return hresult(r)
}
