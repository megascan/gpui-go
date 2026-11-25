//go:build windows

package windows

import (
	"fmt"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

func win32Error(err error) {
	if errno, ok := err.(syscall.Errno); ok {
		panic(com.HRESULTFromWin32(uint32(errno)).Error())
	}
	panic(fmt.Sprint(err))
}

type longPtr = uintptr

var propName = windows.StringToUTF16Ptr("AccessKitAdapter")

type subclassState struct {
	adapter           *Adapter
	activationHandler a11y.ActivationHandler
}

type subclassImpl struct {
	hwnd            HWND
	key             uintptr
	state           subclassState
	borrowed        bool
	prevWndProc     longPtr
	windowDestroyed bool
}

var (
	subclassRegistryMu sync.Mutex
	subclassRegistry   = map[uintptr]*subclassImpl{}
	subclassNextKey    uintptr

	subclassWndProcOnce sync.Once
	subclassWndProcPtr  uintptr
)

func subclassWndProc() uintptr {
	subclassWndProcOnce.Do(func() {
		subclassWndProcPtr = windows.NewCallback(wndProc)
	})
	return subclassWndProcPtr
}

func (s *subclassImpl) borrowMut() func() {
	if s.borrowed {
		panic("already borrowed: BorrowMutError")
	}
	s.borrowed = true
	return func() { s.borrowed = false }
}

func wndProc(window HWND, message uint32, wparam WPARAM, lparam LPARAM) LRESULT {
	handle := win32.GetPropW(window, propName)
	subclassRegistryMu.Lock()
	impl := subclassRegistry[uintptr(handle)]
	subclassRegistryMu.Unlock()
	if impl == nil {
		panic("assertion failed: !impl_ptr.is_null()")
	}
	switch message {
	case wmGetObject:
		release := impl.borrowMut()
		result, ok := impl.state.adapter.HandleWmGetobject(wparam, lparam, impl.state.activationHandler)
		release()
		if ok {
			return result()
		}
	case wmSetFocus, wmExitMenuLoop, wmExitSizeMove:
		impl.updateWindowFocusState(true)
	case wmKillFocus, wmEnterMenuLoop, wmEnterSizeMove:
		impl.updateWindowFocusState(false)
	case wmNCDestroy:
		impl.windowDestroyed = true
	}
	return callWindowProcW(impl.prevWndProc, window, message, wparam, lparam)
}

func newSubclassImpl(hwnd HWND, activationHandler a11y.ActivationHandler, actionHandler a11y.ActionHandler) *subclassImpl {
	adapter := NewAdapter(hwnd, false, actionHandler)
	return &subclassImpl{
		hwnd:  hwnd,
		state: subclassState{adapter: adapter, activationHandler: activationHandler},
	}
}

func (s *subclassImpl) install() {
	if win32.GetPropW(s.hwnd, propName) != 0 {
		panic(fmt.Sprintf("subclassing adapter already instantiated on window %#x", uintptr(s.hwnd)))
	}
	subclassRegistryMu.Lock()
	subclassNextKey++
	s.key = subclassNextKey
	subclassRegistry[s.key] = s
	subclassRegistryMu.Unlock()
	if err := win32.SetPropW(s.hwnd, propName, win32.HANDLE(s.key)); err != nil {
		s.unregister()
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	result, err := setWindowLongPtrW(s.hwnd, gwlpWndProc, subclassWndProc())
	if err != nil {
		win32Error(err)
	}
	s.prevWndProc = result
}

func (s *subclassImpl) unregister() {
	subclassRegistryMu.Lock()
	delete(subclassRegistry, s.key)
	subclassRegistryMu.Unlock()
}

func (s *subclassImpl) updateWindowFocusState(isFocused bool) {
	release := s.borrowMut()
	events, ok := s.state.adapter.UpdateWindowFocusState(isFocused)
	release()
	if ok {
		events.Raise()
	}
}

func (s *subclassImpl) uninstall() {
	if s.windowDestroyed {
		return
	}
	if _, err := setWindowLongPtrW(s.hwnd, gwlpWndProc, s.prevWndProc); err != nil {
		win32Error(err)
	}
	if _, err := win32.RemovePropW(s.hwnd, propName); err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
}

// SubclassingAdapter uses Win32 subclassing to handle WM_GETOBJECT messages on a window
// that provides no other way of adding custom message handlers.
type SubclassingAdapter struct {
	impl *subclassImpl
}

// NewSubclassingAdapter creates a new Windows platform adapter using window subclassing.
// This must be done before the window is shown or focused for the first time, on the
// thread that owns the window. It panics if the window is already visible.
func NewSubclassingAdapter(hwnd HWND, activationHandler a11y.ActivationHandler, actionHandler a11y.ActionHandler) *SubclassingAdapter {
	if win32.IsWindowVisible(hwnd) {
		panic("The AccessKit Windows subclassing adapter must be created before the window is shown (made visible) for the first time.")
	}
	impl := newSubclassImpl(hwnd, activationHandler, actionHandler)
	func() {
		defer func() {
			if r := recover(); r != nil {
				impl.state.adapter.Release()
				panic(r)
			}
		}()
		impl.install()
	}()
	return &SubclassingAdapter{impl: impl}
}

// UpdateIfActive calls updateFactory and applies the resulting update if and only if the
// tree has been initialized. If events are returned, the caller must call Raise on them.
func (a *SubclassingAdapter) UpdateIfActive(updateFactory func() a11y.TreeUpdate) (*QueuedEvents, bool) {
	release := a.impl.borrowMut()
	defer release()
	return a.impl.state.adapter.UpdateIfActive(updateFactory)
}

// Release restores the window's original window procedure (the drop of the adapter).
func (a *SubclassingAdapter) Release() {
	impl := a.impl
	if impl == nil {
		return
	}
	a.impl = nil
	impl.uninstall()
	impl.unregister()
	impl.state.adapter.Release()
}
