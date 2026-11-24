//go:build windows

package windows

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

const defaultTimeout = 5 * time.Second

const (
	testWsOverlappedWindow = 0x00CF0000
	testSwShow             = 5
)

var (
	clsidCUIAutomation8                      = com.MustGUID("{E22AD333-B25F-460C-83D0-0581107395C9}")
	iidIUIAutomation                         = com.MustGUID("{30CBE57D-D9D0-452A-AB13-7AC5AC4825EE}")
	iidIUIAutomationFocusChangedEventHandler = com.MustGUID("{C270F6B5-5C69-4290-9745-7A7F97169468}")
	iidIUIAutomationRangeValuePattern        = com.MustGUID("{59213F4F-7346-49E5-B120-80555987A148}")
	testUser32                               = windows.NewLazySystemDLL("user32.dll")
	testProcLoadCursorW                      = testUser32.NewProc("LoadCursorW")
	testUiautomationcore                     = windows.NewLazySystemDLL("uiautomationcore.dll")
	testProcUiaHasServerSideProvider         = testUiautomationcore.NewProc("UiaHasServerSideProvider")
	errEmpty                                 = errors.New("empty")
)

const (
	uiaCompareElements             = 3
	uiaGetRootElement              = 5
	uiaElementFromHandle           = 6
	uiaElementFromPoint            = 7
	uiaGetFocusedElement           = 8
	uiaControlViewWalker           = 14
	uiaAddFocusChangedEventHandler = 39

	walkerGetParentElement          = 3
	walkerGetFirstChildElement      = 4
	walkerGetLastChildElement       = 5
	walkerGetNextSiblingElement     = 6
	walkerGetPreviousSiblingElement = 7

	elementGetCurrentPatternAs        = 14
	elementCurrentControlType         = 21
	elementCurrentName                = 23
	elementCurrentHasKeyboardFocus    = 26
	elementCurrentIsKeyboardFocusable = 27

	rangeValueSetValue = 3
)

func uiaHasServerSideProvider(hwnd HWND) bool {
	r, _, _ := testProcUiaHasServerSideProvider.Call(uintptr(hwnd))
	return int32(r) != 0
}

func outElement(out *com.IUnknown, err error) (*com.IUnknown, error) {
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, errEmpty
	}
	return out, nil
}

func uiaElementFromHandleCall(uia *com.IUnknown, hwnd HWND) (*com.IUnknown, error) {
	var out *com.IUnknown
	err := com.CallHR(uia.Ptr(), uiaElementFromHandle, uintptr(hwnd), uintptr(unsafe.Pointer(&out)))
	return outElement(out, err)
}

func uiaElementFromPointCall(uia *com.IUnknown, pt win32.POINT) (*com.IUnknown, error) {
	var out *com.IUnknown
	packed := uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32
	err := com.CallHR(uia.Ptr(), uiaElementFromPoint, packed, uintptr(unsafe.Pointer(&out)))
	return outElement(out, err)
}

func uiaNoArgElement(this *com.IUnknown, slot int) (*com.IUnknown, error) {
	var out *com.IUnknown
	err := com.CallHR(this.Ptr(), slot, uintptr(unsafe.Pointer(&out)))
	return outElement(out, err)
}

func walkerCall(walker *com.IUnknown, slot int, element *com.IUnknown) (*com.IUnknown, error) {
	var out *com.IUnknown
	err := com.CallHR(walker.Ptr(), slot, uintptr(element.Ptr()), uintptr(unsafe.Pointer(&out)))
	return outElement(out, err)
}

func uiaCompareElementsCall(uia, a, b *com.IUnknown) (bool, error) {
	var out int32
	err := com.CallHR(uia.Ptr(), uiaCompareElements, uintptr(a.Ptr()), uintptr(b.Ptr()), uintptr(unsafe.Pointer(&out)))
	return out != 0, err
}

func elementInt32(element *com.IUnknown, slot int) (int32, error) {
	var out int32
	err := com.CallHR(element.Ptr(), slot, uintptr(unsafe.Pointer(&out)))
	return out, err
}

func elementName(element *com.IUnknown) (string, error) {
	var out com.BSTR
	if err := com.CallHR(element.Ptr(), elementCurrentName, uintptr(unsafe.Pointer(&out))); err != nil {
		return "", err
	}
	defer com.FreeBSTR(out)
	return com.BSTRToString(out), nil
}

func releaseAll(elements ...*com.IUnknown) {
	for _, e := range elements {
		if e != nil {
			e.Release()
		}
	}
}

type windowState struct {
	activationHandler a11y.ActivationHandler
	adapter           *Adapter
	borrowedAdapter   bool
	borrowedHandler   bool
}

type windowCreateParams struct {
	activationHandler a11y.ActivationHandler
	actionHandler     actionHandlerNoMut
}

var (
	testWindowsMu      sync.Mutex
	testWindowStates   = map[HWND]*windowState{}
	testCreateParams   = map[uintptr]windowCreateParams{}
	testCreateParamKey uintptr

	windowClassOnce sync.Once
	windowClassAtom win32.ATOM
)

func getWindowState(window HWND) *windowState {
	testWindowsMu.Lock()
	defer testWindowsMu.Unlock()
	return testWindowStates[window]
}

func borrow(flag *bool) func() {
	if *flag {
		panic("already borrowed: BorrowMutError")
	}
	*flag = true
	return func() { *flag = false }
}

func testUpdateWindowFocusState(window HWND, isFocused bool) {
	state := getWindowState(window)
	release := borrow(&state.borrowedAdapter)
	events, ok := state.adapter.UpdateWindowFocusState(isFocused)
	release()
	if ok {
		events.Raise()
	}
}

func testWndProc(window HWND, message uint32, wparam WPARAM, lparam LPARAM) LRESULT {
	switch message {
	case win32.WM_NCCREATE:
		createStruct := (*win32.CREATESTRUCTW)(com.PtrFromUintptr(uintptr(lparam)))
		testWindowsMu.Lock()
		params := testCreateParams[createStruct.LpCreateParams]
		delete(testCreateParams, createStruct.LpCreateParams)
		testWindowsMu.Unlock()
		adapter := newAdapterWithWrappedActionHandler(window, false, params.actionHandler)
		testWindowsMu.Lock()
		testWindowStates[window] = &windowState{activationHandler: params.activationHandler, adapter: adapter}
		testWindowsMu.Unlock()
		return win32.DefWindowProcW(window, message, wparam, lparam)
	case win32.WM_PAINT:
		if !win32.ValidateRect(window, nil) {
			panic("called `Result::unwrap()` on an `Err` value: ValidateRect failed")
		}
		return 0
	case win32.WM_DESTROY:
		testWindowsMu.Lock()
		state := testWindowStates[window]
		delete(testWindowStates, window)
		testWindowsMu.Unlock()
		if state != nil {
			state.adapter.Release()
		}
		win32.PostQuitMessage(0)
		return 0
	case wmGetObject:
		state := getWindowState(window)
		if state == nil {
			return win32.DefWindowProcW(window, message, wparam, lparam)
		}
		releaseAdapter := borrow(&state.borrowedAdapter)
		releaseHandler := borrow(&state.borrowedHandler)
		result, ok := state.adapter.HandleWmGetobject(wparam, lparam, state.activationHandler)
		releaseHandler()
		releaseAdapter()
		if !ok {
			return win32.DefWindowProcW(window, message, wparam, lparam)
		}
		return result()
	case wmSetFocus, wmExitMenuLoop, wmExitSizeMove:
		testUpdateWindowFocusState(window, true)
		return 0
	case wmKillFocus, wmEnterMenuLoop, wmEnterSizeMove:
		testUpdateWindowFocusState(window, false)
		return 0
	default:
		return win32.DefWindowProcW(window, message, wparam, lparam)
	}
}

func loadArrowCursor() win32.HCURSOR {
	r, _, err := testProcLoadCursorW.Call(0, win32.IDC_ARROW)
	if r == 0 {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	return win32.HCURSOR(r)
}

func registerWindowClass(name string, proc uintptr) win32.ATOM {
	module, err := win32.GetModuleHandleW(nil)
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	wc := win32.WNDCLASSW{
		HCursor:       loadArrowCursor(),
		HInstance:     win32.HINSTANCE(module),
		LpszClassName: windows.StringToUTF16Ptr(name),
		Style:         win32.CS_HREDRAW | win32.CS_VREDRAW,
		LpfnWndProc:   proc,
	}
	atom, err := win32.RegisterClassW(&wc)
	if err != nil {
		panic(err.Error())
	}
	return atom
}

func createWindow(title string, activationHandler a11y.ActivationHandler, actionHandler a11y.ActionHandler) (HWND, error) {
	windowClassOnce.Do(func() {
		windowClassAtom = registerWindowClass("AccessKitTest", windows.NewCallback(testWndProc))
	})
	testWindowsMu.Lock()
	testCreateParamKey++
	key := testCreateParamKey
	testCreateParams[key] = windowCreateParams{activationHandler: activationHandler, actionHandler: newActionHandlerWrapper(actionHandler)}
	testWindowsMu.Unlock()
	module, err := win32.GetModuleHandleW(nil)
	if err != nil {
		return 0, err
	}
	return win32.CreateWindowExW(
		0,
		(*uint16)(com.PtrFromUintptr(uintptr(windowClassAtom))),
		windows.StringToUTF16Ptr(title),
		testWsOverlappedWindow,
		win32.CW_USEDEFAULT, win32.CW_USEDEFAULT, win32.CW_USEDEFAULT, win32.CW_USEDEFAULT,
		0, 0, win32.HINSTANCE(module), key,
	)
}

type testScope struct {
	uia    *com.IUnknown
	window WindowHandle
}

func (s *testScope) showAndFocusWindow() {
	win32.ShowWindow(s.window.HWND, testSwShow)
	win32.SetForegroundWindow(s.window.HWND)
}

var testMutex sync.Mutex

func scope(t *testing.T, windowTitle string, activationHandler a11y.ActivationHandler, actionHandler a11y.ActionHandler, f func(*testScope) error) error {
	t.Helper()
	testMutex.Lock()
	defer testMutex.Unlock()

	windowCh := make(chan HWND, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		window, err := createWindow(windowTitle, activationHandler, actionHandler)
		if err != nil {
			panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
		}
		windowCh <- window
		var message win32.MSG
		for win32.GetMessageW(&message, 0, 0, 0) > 0 {
			win32.TranslateMessage(&message)
			win32.DispatchMessageW(&message)
		}
	}()
	defer func() { <-done }()

	window := <-windowCh
	defer func() {
		if err := win32.PostMessageW(window, win32.WM_CLOSE, 0, 0); err != nil {
			panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
		}
	}()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := com.CoInitializeEx(com.COINIT_MULTITHREADED); err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	defer com.CoUninitialize()

	uia, err := com.CoCreateInstance(&clsidCUIAutomation8, nil, com.CLSCTX_INPROC_SERVER, &iidIUIAutomation)
	if err != nil {
		return err
	}
	defer uia.Release()

	s := &testScope{uia: uia, window: WindowHandleFromHWND(window)}
	return f(s)
}

type receivedFocusEvent struct {
	mu       sync.Mutex
	received *com.IUnknown
	notify   chan struct{}
}

func newReceivedFocusEvent() *receivedFocusEvent {
	return &receivedFocusEvent{notify: make(chan struct{}, 1)}
}

func (r *receivedFocusEvent) wait(t *testing.T, f func(*com.IUnknown) bool) *com.IUnknown {
	t.Helper()
	for {
		r.mu.Lock()
		element := r.received
		r.received = nil
		r.mu.Unlock()
		if element != nil {
			if f(element) {
				return element
			}
			element.Release()
		}
		select {
		case <-r.notify:
		case <-time.After(defaultTimeout):
			t.Fatal("assertion failed: !result.timed_out()")
		}
	}
}

func (r *receivedFocusEvent) put(element *com.IUnknown) {
	r.mu.Lock()
	old := r.received
	r.received = element
	r.mu.Unlock()
	if old != nil {
		old.Release()
	}
	select {
	case r.notify <- struct{}{}:
	default:
	}
}

type focusEventHandler struct {
	received *receivedFocusEvent
}

var (
	focusEventHandlerVtableOnce sync.Once
	focusEventHandlerVtable     *com.Vtable
)

func newFocusEventHandler() (*com.Object, *receivedFocusEvent) {
	focusEventHandlerVtableOnce.Do(func() {
		focusEventHandlerVtable = com.NewVtable([]com.GUID{iidIUIAutomationFocusChangedEventHandler}, windows.NewCallback(func(this *com.IUnknown, sender *com.IUnknown) uintptr {
			h, ok := com.ImplOf[*focusEventHandler](this)
			if !ok {
				return com.E_UNEXPECTED.Uintptr()
			}
			if sender == nil {
				panic("called `Option::unwrap()` on a `None` value")
			}
			sender.AddRef()
			h.received.put(sender)
			return com.S_OK.Uintptr()
		}))
	})
	received := newReceivedFocusEvent()
	return com.NewObject(&focusEventHandler{received: received}, focusEventHandlerVtable), received
}
