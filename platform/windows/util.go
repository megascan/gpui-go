//go:build windows

package windows

import (
	"fmt"
	"sync"
	"unsafe"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

type hiLoWord interface {
	~uintptr
}

func hiword[T hiLoWord](v T) uint16 {
	return uint16((uintptr(v) >> 16) & 0xFFFF)
}

func loword[T hiLoWord](v T) uint16 {
	return uint16(uintptr(v) & 0xFFFF)
}

func signedHiword[T hiLoWord](v T) int16 {
	return int16((uintptr(v) >> 16) & 0xFFFF)
}

func signedLoword[T hiLoWord](v T) int16 {
	return int16(uintptr(v) & 0xFFFF)
}

func getWindowLong(hwnd HWND, nindex int32) int {
	return int(win32.GetWindowLongPtrW(hwnd, nindex))
}

func setWindowLong(hwnd HWND, nindex int32, dwnewlong int) int {
	return int(win32.SetWindowLongPtrW(hwnd, nindex, uintptr(dwnewlong)))
}

func windowsCredentialsTargetName(url string) string {
	return fmt.Sprintf("zed:url=%s", url)
}

type cursorCache struct {
	once   sync.Once
	cursor SafeCursor
}

var (
	cursorArrow    cursorCache
	cursorIBeam    cursorCache
	cursorCross    cursorCache
	cursorHand     cursorCache
	cursorSizeWE   cursorCache
	cursorSizeNS   cursorCache
	cursorSizeNWSE cursorCache
	cursorSizeNESW cursorCache
	cursorNo       cursorCache
)

func loadCursor(style gpui.CursorStyle) (win32.HCURSOR, bool) {
	var lock *cursorCache
	var name uint16
	switch style {
	case gpui.CursorStyleIBeam, gpui.CursorStyleIBeamCursorForVerticalLayout:
		lock, name = &cursorIBeam, win32.IDC_IBEAM
	case gpui.CursorStyleCrosshair:
		lock, name = &cursorCross, win32.IDC_CROSS
	case gpui.CursorStylePointingHand, gpui.CursorStyleDragLink:
		lock, name = &cursorHand, win32.IDC_HAND
	case gpui.CursorStyleResizeLeft, gpui.CursorStyleResizeRight, gpui.CursorStyleResizeLeftRight, gpui.CursorStyleResizeColumn:
		lock, name = &cursorSizeWE, win32.IDC_SIZEWE
	case gpui.CursorStyleResizeUp, gpui.CursorStyleResizeDown, gpui.CursorStyleResizeUpDown, gpui.CursorStyleResizeRow:
		lock, name = &cursorSizeNS, win32.IDC_SIZENS
	case gpui.CursorStyleResizeUpLeftDownRight:
		lock, name = &cursorSizeNWSE, win32.IDC_SIZENWSE
	case gpui.CursorStyleResizeUpRightDownLeft:
		lock, name = &cursorSizeNESW, win32.IDC_SIZENESW
	case gpui.CursorStyleOperationNotAllowed:
		lock, name = &cursorNo, win32.IDC_NO
	default:
		lock, name = &cursorArrow, win32.IDC_ARROW
	}
	lock.once.Do(func() {
		handle, _ := gpuiutil.Result(win32.LoadImageW(0, win32.MakeIntResource(name), win32.IMAGE_CURSOR, 0, 0, win32.LR_DEFAULTSIZE|win32.LR_SHARED)).LogErr()
		lock.cursor = SafeCursorFrom(win32.HCURSOR(handle))
	})
	return lock.cursor.Deref(), true
}

func configureDwmDarkMode(hwnd HWND, appearance gpui.WindowAppearance) {
	var darkModeEnabled win32.BOOL
	switch appearance {
	case gpui.WindowAppearanceDark, gpui.WindowAppearanceVibrantDark:
		darkModeEnabled = win32.Bool(true)
	case gpui.WindowAppearanceLight, gpui.WindowAppearanceVibrantLight:
		darkModeEnabled = win32.Bool(false)
	}
	if err := win32.DwmSetWindowAttribute(hwnd, win32.DWMWA_USE_IMMERSIVE_DARK_MODE, unsafe.Pointer(&darkModeEnabled), uint32(unsafe.Sizeof(darkModeEnabled))); err != nil {
		gpuiutil.LogErr(err)
	}
}

func logicalPoint(x, y, scaleFactor float32) gpui.Point[gpui.Pixels] {
	return gpui.Point[gpui.Pixels]{
		X: gpui.Px(x / scaleFactor),
		Y: gpui.Px(y / scaleFactor),
	}
}

const uiSettingsClassName = "Windows.UI.ViewManagement.UISettings"

var iidIUISettings3 = com.MustGUID("{03021BE4-5254-4781-8194-5168F7D06D7B}")

const (
	uiColorTypeForeground     int32 = 1
	iuiSettings3GetColorValue       = com.IInspectableSlots
)

type uiColor struct {
	A uint8
	R uint8
	G uint8
	B uint8
}

func systemAppearance() (gpui.WindowAppearance, error) {
	uiSettings, err := com.RoActivateInstance(uiSettingsClassName)
	if err != nil {
		return 0, err
	}
	defer uiSettings.Release()
	settings3, err := com.Cast[com.IInspectable](&uiSettings.IUnknown, &iidIUISettings3)
	if err != nil {
		return 0, err
	}
	defer settings3.Release()
	var foregroundColor uiColor
	if err := com.CallHR(settings3.Ptr(), iuiSettings3GetColorValue, uintptr(uiColorTypeForeground), uintptr(unsafe.Pointer(&foregroundColor))); err != nil {
		return 0, err
	}
	if isColorLight(&foregroundColor) {
		return gpui.WindowAppearanceDark, nil
	}
	return gpui.WindowAppearanceLight, nil
}

func isColorLight(color *uiColor) bool {
	return (5*uint32(color.G) + 2*uint32(color.R) + uint32(color.B)) > (8 * 128)
}

func withDllLibrary[R any](dllName string, f func(win32.HMODULE) (R, error)) (R, error) {
	var zero R
	name, err := xwin.BytePtrFromString(dllName)
	if err != nil {
		return zero, fmt.Errorf("Loading dll: %s: %w", dllName, err)
	}
	library, err := win32.LoadLibraryA(name)
	if err != nil {
		return zero, fmt.Errorf("Loading dll: %s: %w", dllName, err)
	}
	result, resultErr := f(library)
	if err := xwin.FreeLibrary(library); err != nil {
		gpuiutil.LogErr(fmt.Errorf("Freeing dll: %s: %w", dllName, err))
	}
	return result, resultErr
}
