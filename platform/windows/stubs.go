//go:build windows

package windows

import (
	"github.com/megascan/gpui-go/internal/win/win32"
)

// Window messages gpui posts to its own windows (events.rs).
const (
	WM_GPUI_CURSOR_STYLE_CHANGED           uint32 = win32.WM_USER + 1
	WM_GPUI_CLOSE_ONE_WINDOW               uint32 = win32.WM_USER + 2
	WM_GPUI_TASK_DISPATCHED_ON_MAIN_THREAD uint32 = win32.WM_USER + 3
	WM_GPUI_DOCK_MENU_ACTION               uint32 = win32.WM_USER + 4
	WM_GPUI_FORCE_UPDATE_WINDOW            uint32 = win32.WM_USER + 5
	WM_GPUI_KEYBOARD_LAYOUT_CHANGED        uint32 = win32.WM_USER + 6
	WM_GPUI_GPU_DEVICE_LOST                uint32 = win32.WM_USER + 7
	WM_GPUI_DISPLAYS_CHANGED               uint32 = win32.WM_USER + 10
)
