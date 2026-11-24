//go:build windows

package win32

import (
	"fmt"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

type layout struct {
	name string
	got  uintptr
	want uintptr
}

func checkLayouts(t *testing.T, layouts []layout) {
	t.Helper()
	for _, l := range layouts {
		if l.got != l.want {
			t.Errorf("%s = %d, want %d", l.name, l.got, l.want)
		}
	}
}

func TestTypes_StructSizes(t *testing.T) {
	checkLayouts(t, []layout{
		{"POINT", unsafe.Sizeof(POINT{}), 8},
		{"SIZE", unsafe.Sizeof(SIZE{}), 8},
		{"RECT", unsafe.Sizeof(RECT{}), 16},
		{"MSG", unsafe.Sizeof(MSG{}), 48},
		{"WNDCLASSW", unsafe.Sizeof(WNDCLASSW{}), 72},
		{"CREATESTRUCTW", unsafe.Sizeof(CREATESTRUCTW{}), 80},
		{"WINDOWPLACEMENT", unsafe.Sizeof(WINDOWPLACEMENT{}), 44},
		{"WINDOWPOS", unsafe.Sizeof(WINDOWPOS{}), 40},
		{"NCCALCSIZE_PARAMS", unsafe.Sizeof(NCCALCSIZE_PARAMS{}), 56},
		{"MINMAXINFO", unsafe.Sizeof(MINMAXINFO{}), 40},
		{"STYLESTRUCT", unsafe.Sizeof(STYLESTRUCT{}), 8},
		{"MONITORINFO", unsafe.Sizeof(MONITORINFO{}), 40},
		{"MONITORINFOEXW", unsafe.Sizeof(MONITORINFOEXW{}), 104},
		{"DEVMODEW", unsafe.Sizeof(DEVMODEW{}), 220},
		{"TRACKMOUSEEVENT", unsafe.Sizeof(TRACKMOUSEEVENT{}), 24},
		{"FLASHWINFO", unsafe.Sizeof(FLASHWINFO{}), 32},
		{"MOUSEINPUT", unsafe.Sizeof(MOUSEINPUT{}), 32},
		{"KEYBDINPUT", unsafe.Sizeof(KEYBDINPUT{}), 24},
		{"HARDWAREINPUT", unsafe.Sizeof(HARDWAREINPUT{}), 8},
		{"INPUT", unsafe.Sizeof(INPUT{}), 40},
		{"COMPOSITIONFORM", unsafe.Sizeof(COMPOSITIONFORM{}), 28},
		{"CANDIDATEFORM", unsafe.Sizeof(CANDIDATEFORM{}), 32},
		{"TP_CALLBACK_ENVIRON_V3", unsafe.Sizeof(TP_CALLBACK_ENVIRON_V3{}), 72},
		{"UNSIGNED_RATIO", unsafe.Sizeof(UNSIGNED_RATIO{}), 8},
		{"DWM_TIMING_INFO", unsafe.Sizeof(DWM_TIMING_INFO{}), 292},
		{"MARGINS", unsafe.Sizeof(MARGINS{}), 16},
		{"USEROBJECTFLAGS", unsafe.Sizeof(USEROBJECTFLAGS{}), 12},
		{"POWERBROADCAST_SETTING", unsafe.Sizeof(POWERBROADCAST_SETTING{}), 24},
		{"SYSTEM_POWER_STATUS", unsafe.Sizeof(SYSTEM_POWER_STATUS{}), 12},
		{"REASON_CONTEXT", unsafe.Sizeof(REASON_CONTEXT{}), 32},
		{"CREDENTIAL_ATTRIBUTEW", unsafe.Sizeof(CREDENTIAL_ATTRIBUTEW{}), 24},
		{"CREDENTIALW", unsafe.Sizeof(CREDENTIALW{}), 80},
		{"FORMATETC", unsafe.Sizeof(FORMATETC{}), 32},
		{"STGMEDIUM", unsafe.Sizeof(STGMEDIUM{}), 24},
		{"PROPERTYKEY", unsafe.Sizeof(PROPERTYKEY{}), 20},
		{"COMDLG_FILTERSPEC", unsafe.Sizeof(COMDLG_FILTERSPEC{}), 16},
	})
}

func TestTypes_FieldOffsets(t *testing.T) {
	var msg MSG
	var class WNDCLASSW
	var create CREATESTRUCTW
	var mode DEVMODEW
	var info MONITORINFOEXW
	var environ TP_CALLBACK_ENVIRON_V3
	var timing DWM_TIMING_INFO
	var input INPUT
	var credential CREDENTIALW
	var formatetc FORMATETC
	var flash FLASHWINFO
	checkLayouts(t, []layout{
		{"MSG.WParam", unsafe.Offsetof(msg.WParam), 16},
		{"MSG.Time", unsafe.Offsetof(msg.Time), 32},
		{"MSG.Pt", unsafe.Offsetof(msg.Pt), 36},
		{"WNDCLASSW.LpfnWndProc", unsafe.Offsetof(class.LpfnWndProc), 8},
		{"WNDCLASSW.LpszClassName", unsafe.Offsetof(class.LpszClassName), 64},
		{"CREATESTRUCTW.Cy", unsafe.Offsetof(create.Cy), 32},
		{"CREATESTRUCTW.LpszName", unsafe.Offsetof(create.LpszName), 56},
		{"CREATESTRUCTW.DwExStyle", unsafe.Offsetof(create.DwExStyle), 72},
		{"DEVMODEW.DmSize", unsafe.Offsetof(mode.DmSize), 68},
		{"DEVMODEW.DmFields", unsafe.Offsetof(mode.DmFields), 72},
		{"DEVMODEW.DmPosition", unsafe.Offsetof(mode.DmPosition), 76},
		{"DEVMODEW.DmColor", unsafe.Offsetof(mode.DmColor), 92},
		{"DEVMODEW.DmFormName", unsafe.Offsetof(mode.DmFormName), 102},
		{"DEVMODEW.DmLogPixels", unsafe.Offsetof(mode.DmLogPixels), 166},
		{"DEVMODEW.DmBitsPerPel", unsafe.Offsetof(mode.DmBitsPerPel), 168},
		{"DEVMODEW.DmDisplayFrequency", unsafe.Offsetof(mode.DmDisplayFrequency), 184},
		{"DEVMODEW.DmPanningHeight", unsafe.Offsetof(mode.DmPanningHeight), 216},
		{"MONITORINFOEXW.SzDevice", unsafe.Offsetof(info.SzDevice), 40},
		{"TP_CALLBACK_ENVIRON_V3.Flags", unsafe.Offsetof(environ.Flags), 56},
		{"TP_CALLBACK_ENVIRON_V3.CallbackPriority", unsafe.Offsetof(environ.CallbackPriority), 60},
		{"TP_CALLBACK_ENVIRON_V3.Size", unsafe.Offsetof(environ.Size), 64},
		{"DWM_TIMING_INFO.RateRefresh", unsafe.Offsetof(timing.RateRefresh), 4},
		{"DWM_TIMING_INFO.QpcRefreshPeriod", unsafe.Offsetof(timing.QpcRefreshPeriod), 12},
		{"DWM_TIMING_INFO.CDXRefresh", unsafe.Offsetof(timing.CDXRefresh), 44},
		{"DWM_TIMING_INFO.QpcCompose", unsafe.Offsetof(timing.QpcCompose), 48},
		{"DWM_TIMING_INFO.CFramesOutstanding", unsafe.Offsetof(timing.CFramesOutstanding), 120},
		{"DWM_TIMING_INFO.CFrameDisplayed", unsafe.Offsetof(timing.CFrameDisplayed), 124},
		{"DWM_TIMING_INFO.CBuffersEmpty", unsafe.Offsetof(timing.CBuffersEmpty), 284},
		{"INPUT.union", unsafe.Offsetof(input.union), 8},
		{"CREDENTIALW.LastWritten", unsafe.Offsetof(credential.LastWritten), 24},
		{"CREDENTIALW.CredentialBlob", unsafe.Offsetof(credential.CredentialBlob), 40},
		{"CREDENTIALW.UserName", unsafe.Offsetof(credential.UserName), 72},
		{"FORMATETC.Ptd", unsafe.Offsetof(formatetc.Ptd), 8},
		{"FORMATETC.Tymed", unsafe.Offsetof(formatetc.Tymed), 24},
		{"FLASHWINFO.DwFlags", unsafe.Offsetof(flash.DwFlags), 16},
	})
}

func TestTypes_PackedU64(t *testing.T) {
	p := PackedU64{0x89abcdef, 0x01234567}
	if got := p.Get(); got != 0x0123456789abcdef {
		t.Fatalf("PackedU64.Get() = %#x", got)
	}
}

func TestWin32_PackPoint(t *testing.T) {
	got := PackPoint(POINT{X: -1, Y: 2})
	if uint64(got) != 0x00000002_FFFFFFFF {
		t.Fatalf("PackPoint = %#x", got)
	}
}

var (
	nativeThreadOnce  sync.Once
	nativeThreadStart uintptr
	nativeThreadFuncs Handles[func()]
)

func runOnNativeThread(t *testing.T, f func()) {
	t.Helper()
	nativeThreadOnce.Do(func() {
		nativeThreadStart = windows.NewCallback(func(parameter uintptr) uintptr {
			if f, ok := nativeThreadFuncs.Take(parameter); ok {
				f()
			}
			return 0
		})
	})
	done := make(chan struct{})
	key := nativeThreadFuncs.Insert(func() {
		defer close(done)
		f()
	})
	handle, _, err := CreateThread(0, nativeThreadStart, key, 0)
	if err != nil {
		nativeThreadFuncs.Take(key)
		t.Fatal(err)
	}
	windows.CloseHandle(handle)
	<-done
}

func TestWin32_MessageOnlyWindowRoundTrip(t *testing.T) {
	var failure string
	runOnNativeThread(t, func() {
		failure = messageOnlyWindowRoundTrip()
	})
	if failure != "" {
		t.Fatal(failure)
	}
}

func messageOnlyWindowRoundTrip() string {
	className := UTF16PtrFromString("GpuiWin32TestClass")
	instance, err := GetModuleHandleW(nil)
	if err != nil {
		return err.Error()
	}
	class := WNDCLASSW{
		LpfnWndProc:   procDefWindowProcW.Addr(),
		HInstance:     instance,
		LpszClassName: className,
	}
	if _, err := RegisterClassW(&class); err != nil {
		return err.Error()
	}
	defer UnregisterClassW(className, instance)
	hwnd, err := CreateWindowExW(0, className, nil, 0, 0, 0, 0, 0, HWND(HWND_MESSAGE), 0, instance, 0)
	if err != nil {
		return err.Error()
	}
	defer DestroyWindow(hwnd)
	if IsWindowVisible(hwnd) {
		return "message-only window is visible"
	}
	if err := PostMessageW(hwnd, WM_USER, 7, 9); err != nil {
		return err.Error()
	}
	var msg MSG
	if !PeekMessageW(&msg, hwnd, WM_USER, WM_USER, PM_REMOVE) {
		return "posted message not received"
	}
	if msg.WParam != 7 || msg.LParam != 9 {
		return fmt.Sprintf("message = %+v", msg)
	}
	return ""
}

func TestWin32_Handles(t *testing.T) {
	var h Handles[string]
	key := h.Insert("a")
	if key == 0 {
		t.Fatal("key 0 handed out")
	}
	if v, ok := h.Get(key); !ok || v != "a" {
		t.Fatalf("Get = %q, %v", v, ok)
	}
	if v, ok := h.Take(key); !ok || v != "a" {
		t.Fatalf("Take = %q, %v", v, ok)
	}
	if _, ok := h.Take(key); ok {
		t.Fatal("Take twice succeeded")
	}
	if h.Len() != 0 {
		t.Fatalf("Len = %d", h.Len())
	}
}
