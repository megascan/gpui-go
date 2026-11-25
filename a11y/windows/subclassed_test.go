//go:build windows

package windows

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

const subclassedWindowTitle = "Simple test"

var (
	subclassWindowClassOnce sync.Once
	subclassWindowClassAtom win32.ATOM
)

func subclassTestWndProc(window HWND, message uint32, wparam WPARAM, lparam LPARAM) LRESULT {
	return win32.DefWindowProcW(window, message, wparam, lparam)
}

func createSubclassWindow(title string) HWND {
	subclassWindowClassOnce.Do(func() {
		subclassWindowClassAtom = registerWindowClass("AccessKitSubclassTest", windows.NewCallback(subclassTestWndProc))
	})
	module, err := win32.GetModuleHandleW(nil)
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	window, err := win32.CreateWindowExW(
		0,
		(*uint16)(com.PtrFromUintptr(uintptr(subclassWindowClassAtom))),
		windows.StringToUTF16Ptr(title),
		testWsOverlappedWindow,
		win32.CW_USEDEFAULT, win32.CW_USEDEFAULT, win32.CW_USEDEFAULT, win32.CW_USEDEFAULT,
		0, 0, win32.HINSTANCE(module), 0,
	)
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	return window
}

func TestSubclassed_HasNativeUia(t *testing.T) {
	testMutex.Lock()
	defer testMutex.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window := createSubclassWindow(subclassedWindowTitle)
	adapter := NewSubclassingAdapter(window, simpleActivationHandler{initial: simpleInitialState}, nullActionHandler{})
	if !uiaHasServerSideProvider(window) {
		t.Fatal("assertion failed: UiaHasServerSideProvider(hwnd).as_bool()")
	}
	adapter.Release()
	if err := win32.DestroyWindow(window); err != nil {
		t.Fatal(err)
	}
}

func TestSubclassed_DoubleInstantiate(t *testing.T) {
	testMutex.Lock()
	defer testMutex.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window := createSubclassWindow(subclassedWindowTitle)
	defer func() {
		if err := win32.DestroyWindow(window); err != nil {
			t.Error(err)
		}
	}()
	adapter1 := NewSubclassingAdapter(window, simpleActivationHandler{initial: simpleInitialState}, nullActionHandler{})
	defer adapter1.Release()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("test did not panic as expected")
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, "already instantiated") {
			t.Fatalf("panic did not contain expected string\n      panic message: %q\n expected substring: %q", msg, "already instantiated")
		}
	}()
	adapter2 := NewSubclassingAdapter(window, simpleActivationHandler{initial: simpleInitialState}, nullActionHandler{})
	adapter2.Release()
}
