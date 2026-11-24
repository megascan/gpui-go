//go:build windows

package win32

import "unsafe"

var (
	procImmAssociateContextEx    = imm32.NewProc("ImmAssociateContextEx")
	procImmGetCompositionStringW = imm32.NewProc("ImmGetCompositionStringW")
	procImmGetContext            = imm32.NewProc("ImmGetContext")
	procImmNotifyIME             = imm32.NewProc("ImmNotifyIME")
	procImmReleaseContext        = imm32.NewProc("ImmReleaseContext")
	procImmSetCandidateWindow    = imm32.NewProc("ImmSetCandidateWindow")
	procImmSetCompositionWindow  = imm32.NewProc("ImmSetCompositionWindow")
)

// ImmAssociateContextEx changes the input context association of a window.
func ImmAssociateContextEx(hwnd HWND, himc HIMC, flags uint32) bool {
	r, _, _ := procImmAssociateContextEx.Call(uintptr(hwnd), uintptr(himc), uintptr(flags))
	return r != 0
}

// ImmGetCompositionStringW reads composition information into buf (nil queries the size in bytes).
func ImmGetCompositionStringW(himc HIMC, index uint32, buf unsafe.Pointer, bufLen uint32) int32 {
	r, _, _ := procImmGetCompositionStringW.Call(uintptr(himc), uintptr(index), uintptr(buf), uintptr(bufLen))
	return int32(r)
}

// ImmGetContext returns a window's input context.
func ImmGetContext(hwnd HWND) HIMC {
	r, _, _ := procImmGetContext.Call(uintptr(hwnd))
	return HIMC(r)
}

// ImmNotifyIME notifies the IME of an input context change.
func ImmNotifyIME(himc HIMC, action uint32, index uint32, value uint32) bool {
	r, _, _ := procImmNotifyIME.Call(uintptr(himc), uintptr(action), uintptr(index), uintptr(value))
	return r != 0
}

// ImmReleaseContext releases an input context.
func ImmReleaseContext(hwnd HWND, himc HIMC) bool {
	r, _, _ := procImmReleaseContext.Call(uintptr(hwnd), uintptr(himc))
	return r != 0
}

// ImmSetCandidateWindow positions a candidate window.
func ImmSetCandidateWindow(himc HIMC, form *CANDIDATEFORM) bool {
	r, _, _ := procImmSetCandidateWindow.Call(uintptr(himc), uintptr(unsafe.Pointer(form)))
	return r != 0
}

// ImmSetCompositionWindow positions the composition window.
func ImmSetCompositionWindow(himc HIMC, form *COMPOSITIONFORM) bool {
	r, _, _ := procImmSetCompositionWindow.Call(uintptr(himc), uintptr(unsafe.Pointer(form)))
	return r != 0
}
