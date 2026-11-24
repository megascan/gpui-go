//go:build windows

package windows

import (
	"sync/atomic"

	gpui "github.com/megascan/gpui-go"
	a11ywin "github.com/megascan/gpui-go/a11y/windows"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// WindowsWindowState is the mutable state of a window; M4-windows-window completes it.
type WindowsWindowState struct {
	origin               gpui.Point[gpui.Pixels]
	logicalSize          gpui.Size[gpui.Pixels]
	minSize              gpui.Option[gpui.Size[gpui.Pixels]]
	borderOffset         WindowBorderOffset
	appearance           gpui.WindowAppearance
	scaleFactor          float32
	restoreFromMinimized func(gpui.RequestFrameOptions)

	callbacks             Callbacks
	frameSignal           *gpui.PlatformFrameSignal
	inputHandler          *gpui.PlatformInputHandler
	imeEnabled            bool
	pendingSurrogate      gpui.Option[uint16]
	lastReportedModifiers gpui.Option[gpui.Modifiers]
	lastReportedCapslock  gpui.Option[gpui.Capslock]
	hovered               bool
	lastVisibility        gpui.Option[gpui.WindowVisibility]
	directManipulation    *DirectManipulationHandler

	renderer           *DirectXRenderer
	forceRenderPending bool

	clickState      ClickState
	currentCursor   win32.HCURSOR
	cursorVisible   *atomic.Bool
	ncButtonPressed gpui.Option[uint32]

	display           *WindowsDisplay
	invalidateDevices *atomic.Bool
	drawCoordinator   *DrawCoordinator
	a11y              a11yCell
	mousePosition     gpui.Point[gpui.Pixels]
}

// WindowsWindowInner is the window procedure's view of a window; M4-windows-window completes it.
type WindowsWindowInner struct {
	hwnd                 HWND
	state                WindowsWindowState
	systemSettings       *WindowsSystemSettings
	hideTitleBar         bool
	isMovable            bool
	isResizable          bool
	isMinimizable        bool
	executor             gpui.ForegroundExecutor
	validationNumber     uintptr
	mainReceiver         *gpui.PriorityQueueReceiver[gpui.RunnableVariant]
	platformWindowHandle HWND
	parentHwnd           HWND
}

// Callbacks are the window's registered platform callbacks; each is taken while it runs.
type Callbacks struct {
	requestFrame         func(gpui.RequestFrameOptions)
	input                func(gpui.PlatformInput) gpui.DispatchEventResult
	activeStatusChange   func(bool)
	visibilityChange     func(gpui.WindowVisibility)
	hoveredStatusChange  func(bool)
	resize               func(gpui.Size[gpui.Pixels], float32)
	moved                func()
	displayChanged       func()
	shouldClose          func() bool
	close                func()
	hitTestWindowControl func() (gpui.WindowControlArea, bool)
	appearanceChanged    func()
}

// ClickState tracks multi-click counting; M4-windows-window ports it.
type ClickState struct {
	currentCount int
}

func (c *ClickState) update(button gpui.MouseButton, newPosition gpui.Point[gpui.DevicePixels]) int {
	panic("windows: unimplemented ClickState.update")
}

func (c *ClickState) systemUpdate(wparam uintptr) {
	panic("windows: unimplemented ClickState.systemUpdate")
}

// WindowBorderOffset is the size of the non-client frame; M4-windows-window ports it.
type WindowBorderOffset struct {
	widthOffset  int32
	heightOffset int32
}

func (o *WindowBorderOffset) update(hwnd HWND) error {
	panic("windows: unimplemented WindowBorderOffset.update")
}

func (s *WindowsWindowState) isFullscreen() bool {
	panic("windows: unimplemented WindowsWindowState.isFullscreen")
}

func (s *WindowsWindowState) isMaximized() bool {
	panic("windows: unimplemented WindowsWindowState.isMaximized")
}

func (w *WindowsWindowInner) visibility() gpui.WindowVisibility {
	panic("windows: unimplemented WindowsWindowInner.visibility")
}

// DirectManipulationHandler turns DirectManipulation viewport updates into input; M4-windows-shell ports it.
type DirectManipulationHandler struct{}

func (h *DirectManipulationHandler) setScaleFactor(scaleFactor float32) {
	panic("windows: unimplemented DirectManipulationHandler.setScaleFactor")
}

func (h *DirectManipulationHandler) onPointerHitTest(wparam win32.WPARAM) {
	panic("windows: unimplemented DirectManipulationHandler.onPointerHitTest")
}

func (h *DirectManipulationHandler) update() {
	panic("windows: unimplemented DirectManipulationHandler.update")
}

func (h *DirectManipulationHandler) drainEvents() []gpui.PlatformInput {
	panic("windows: unimplemented DirectManipulationHandler.drainEvents")
}

// A11yState is a window's accessibility adapter; M4-windows-window and M4-windows-a11y port it.
type A11yState struct{}

type a11yCell struct {
	value    *A11yState
	borrowed bool
}

func (s *A11yState) updateWindowFocusState(activated bool) (*a11ywin.QueuedEvents, bool) {
	panic("windows: unimplemented A11yState.updateWindowFocusState")
}

func (s *A11yState) handleWMGetObject(wparam win32.WPARAM, lparam win32.LPARAM) (func() win32.LRESULT, bool) {
	panic("windows: unimplemented A11yState.handleWMGetObject")
}

func raiseA11yEvents(events *a11ywin.QueuedEvents) {
	panic("windows: unimplemented raiseA11yEvents")
}
