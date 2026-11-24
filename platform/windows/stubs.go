//go:build windows

package windows

import (
	"sync/atomic"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// WindowCreationInfo is what the platform hands a new window (platform.rs); M4-windows-platform
// moves it into platform.go.
type WindowCreationInfo struct {
	icon                     win32.HICON
	executor                 gpui.ForegroundExecutor
	currentCursor            win32.HCURSOR
	cursorVisible            *atomic.Bool
	dropTargetHelper         *IDropTargetHelper
	validationNumber         uintptr
	mainReceiver             *gpui.PriorityQueueReceiver[gpui.RunnableVariant]
	platformWindowHandle     HWND
	disableDirectComposition bool
	directxDevices           *DirectXDevices
	invalidateDevices        *atomic.Bool
	drawCoordinator          *DrawCoordinator
}
