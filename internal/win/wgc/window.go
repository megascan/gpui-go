//go:build windows

package wgc

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// Window errors (window::Error).
var (
	ErrNoActiveWindow                     = errors.New("No active window found.")
	ErrWindowFailedToConvertWindowsString = errors.New("Failed to convert a Windows string from UTF-16")
)

// WindowNotFoundError is window::Error::NotFound.
type WindowNotFoundError struct{ Name string }

func (e *WindowNotFoundError) Error() string {
	return "Failed to find a window with the name: " + e.Name
}

// WindowError is window::Error::WindowsError.
type WindowError struct{ Err error }

func (e *WindowError) Error() string { return fmt.Sprintf("A Windows API call failed: %v", e.Err) }

func (e *WindowError) Unwrap() error { return e.Err }

func lastWin32Error() error {
	if err := windows.GetLastError(); err != nil {
		return err
	}
	return windows.ERROR_SUCCESS
}

// Window is a top-level window.
type Window struct {
	window win32.HWND
}

// WindowForeground returns the foreground window.
func WindowForeground() (Window, error) {
	window := win32.GetForegroundWindow()
	if window == 0 {
		return Window{}, ErrNoActiveWindow
	}
	return Window{window: window}, nil
}

// WindowFromName returns the window titled title.
func WindowFromName(title string) (Window, error) {
	window, err := findWindowW(title)
	if err != nil {
		return Window{}, &WindowError{Err: err}
	}
	if window == 0 {
		return Window{}, &WindowNotFoundError{Name: title}
	}
	return Window{window: window}, nil
}

// WindowFromContainsName returns the first enumerated window whose title contains title.
func WindowFromContainsName(title string) (Window, error) {
	windowsList, err := WindowEnumerate()
	if err != nil {
		return Window{}, err
	}
	for _, window := range windowsList {
		t, err := window.Title()
		if err != nil {
			return Window{}, err
		}
		if strings.Contains(t, title) {
			return window, nil
		}
	}
	return Window{}, &WindowNotFoundError{Name: title}
}

// Title returns the window title.
func (w Window) Title() (string, error) {
	n := getWindowTextLengthW(w.window)
	if n < 0 {
		panic("called `Result::unwrap()` on an `Err` value: TryFromIntError(())")
	}
	name := make([]uint16, int(n)+1)
	if n >= 1 {
		if getWindowTextW(w.window, name) == 0 {
			return "", nil
		}
	}
	s, err := utf16UntilNul(name)
	if err != nil {
		return "", ErrWindowFailedToConvertWindowsString
	}
	return s, nil
}

// ProcessID returns the id of the process owning the window.
func (w Window) ProcessID() (uint32, error) {
	var id uint32
	_, _ = windows.GetWindowThreadProcessId(windows.HWND(w.window), &id)
	if id == 0 {
		return 0, &WindowError{Err: lastWin32Error()}
	}
	return id, nil
}

// ProcessName returns the base name of the owning process's executable.
func (w Window) ProcessName() (string, error) {
	id, err := w.ProcessID()
	if err != nil {
		return "", err
	}
	process, err := windows.OpenProcess(processQueryInformation|processVMRead, false, id)
	if err != nil {
		return "", &WindowError{Err: err}
	}
	defer windows.CloseHandle(process)
	name := make([]uint16, 260)
	if err := windows.GetModuleBaseName(process, 0, &name[0], uint32(len(name))); err != nil {
		return "", &WindowError{Err: err}
	}
	s, err := utf16UntilNul(name)
	if err != nil {
		return "", ErrWindowFailedToConvertWindowsString
	}
	return s, nil
}

// Monitor returns the monitor the window is on.
func (w Window) Monitor() (Monitor, bool) {
	monitor := win32.MonitorFromWindow(w.window, win32.MONITOR_DEFAULTTONULL)
	if monitor == 0 {
		return Monitor{}, false
	}
	return MonitorFromRawHMonitor(uintptr(monitor)), true
}

// Rect returns the window rectangle.
func (w Window) Rect() (win32.RECT, error) {
	var rect win32.RECT
	if err := win32.GetWindowRect(w.window, &rect); err != nil {
		return rect, &WindowError{Err: err}
	}
	return rect, nil
}

// TitleBarHeight returns the height of the window's title bar in physical pixels.
func (w Window) TitleBarHeight() (uint32, error) {
	var windowRect, clientRect win32.RECT
	if err := win32.DwmGetWindowAttribute(w.window, dwmwaExtendedFrameBounds, unsafe.Pointer(&windowRect), uint32(unsafe.Sizeof(windowRect))); err != nil {
		return 0, &WindowError{Err: err}
	}
	if err := win32.GetClientRect(w.window, &clientRect); err != nil {
		return 0, &WindowError{Err: err}
	}
	windowHeight := windowRect.Bottom - windowRect.Top
	dpi := win32.GetDpiForWindow(w.window)
	clientHeight := (clientRect.Bottom - clientRect.Top) * int32(dpi) / 96
	actualTitleHeight := windowHeight - clientHeight
	return uint32(actualTitleHeight), nil
}

// IsValid reports whether the window is a visible, capturable top-level window of another process.
func (w Window) IsValid() bool {
	if !win32.IsWindowVisible(w.window) {
		return false
	}
	var id uint32
	_, _ = windows.GetWindowThreadProcessId(windows.HWND(w.window), &id)
	if id == windows.GetCurrentProcessId() {
		return false
	}
	var rect win32.RECT
	if err := win32.GetClientRect(w.window, &rect); err != nil {
		return false
	}
	styles := int64(int(win32.GetWindowLongPtrW(w.window, win32.GWL_STYLE)))
	exStyles := int64(int(win32.GetWindowLongPtrW(w.window, win32.GWL_EXSTYLE)))
	if exStyles&win32.WS_EX_TOOLWINDOW != 0 {
		return false
	}
	if styles&wsChild != 0 {
		return false
	}
	return true
}

var (
	windowEnum         win32.Handles[*[]Window]
	windowCallbackOnce sync.Once
	windowCallback     uintptr
)

func enumWindowsCallback() uintptr {
	windowCallbackOnce.Do(func() {
		windowCallback = windows.NewCallback(func(window, data uintptr) uintptr {
			if list, ok := windowEnum.Get(data); ok {
				if WindowFromRawHWND(window).IsValid() {
					*list = append(*list, Window{window: win32.HWND(window)})
				}
			}
			return 1
		})
	})
	return windowCallback
}

// WindowEnumerate returns every valid top-level window.
func WindowEnumerate() ([]Window, error) {
	var list []Window
	key := windowEnum.Insert(&list)
	defer windowEnum.Take(key)
	windows.EnumChildWindows(windows.GetDesktopWindow(), enumWindowsCallback(), com.PtrFromUintptr(key))
	return list, nil
}

// WindowFromRawHWND wraps an HWND.
func WindowFromRawHWND(hwnd uintptr) Window { return Window{window: win32.HWND(hwnd)} }

// AsRawHWND returns the HWND.
func (w Window) AsRawHWND() uintptr { return uintptr(w.window) }

// TryIntoCaptureItem creates a capture item for the window.
func (w Window) TryIntoCaptureItem() (*GraphicsCaptureItem, CaptureItemTypes, error) {
	interop, err := GraphicsCaptureItemInteropFactory()
	if err != nil {
		return nil, CaptureItemTypes{}, err
	}
	defer interop.Release()
	item, err := interop.CreateForWindow(w.window)
	if err != nil {
		return nil, CaptureItemTypes{}, err
	}
	return item, CaptureItemTypes{Window: &w}, nil
}
