//go:build linux

package linux

import (
	"errors"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
	"github.com/megascan/gpui-go/scheduler"
)

func selectBackend(modes gpui.WindowingModes, environment gpui.GraphicalEnvironment) (backend, bool) {
	isSet := func(value gpui.Option[string]) bool {
		v, ok := value.Get()
		return ok && v != ""
	}
	if modes.Contains(gpui.WindowingModesWayland) && isSet(environment.WaylandDisplay) {
		return backendWayland, true
	}
	if modes.Contains(gpui.WindowingModesX11) && isSet(environment.X11Display) {
		return backendX11, true
	}
	return 0, false
}

type backend uint8

const (
	backendWayland backend = iota
	backendX11
)

type displayConnection struct {
	headless  *common.HeadlessConnection
	connected common.LinuxClient
}

func displayConnectionHeadless(connection *common.HeadlessConnection) *displayConnection {
	return &displayConnection{headless: connection}
}

func displayConnectionConnected(connection common.LinuxClient) *displayConnection {
	return &displayConnection{connected: connection}
}

func (c *displayConnection) close() {
	if c.connected != nil {
		c.connected.Close()
	}
}

func (c *displayConnection) isHeadless() bool {
	return c.connected == nil
}

func (c *displayConnection) compositorName() string {
	if c.connected != nil {
		return c.connected.CompositorName()
	}
	return "headless"
}

func (c *displayConnection) hasWindows() bool {
	if c.connected != nil {
		return c.connected.HasWindows()
	}
	return c.headless.HasWindows()
}

func (c *displayConnection) graphicalEnvironment() (gpui.GraphicalEnvironment, bool) {
	if c.connected != nil {
		return c.connected.GraphicalEnvironment(), true
	}
	return gpui.GraphicalEnvironment{}, false
}

func (c *displayConnection) keyboardLayout() gpui.PlatformKeyboardLayout {
	if c.connected != nil {
		return c.connected.KeyboardLayout()
	}
	return common.NewLinuxKeyboardLayout("unknown")
}

func (c *displayConnection) displays() []gpui.PlatformDisplay {
	if c.connected != nil {
		return c.connected.Displays()
	}
	return c.headless.Displays()
}

func (c *displayConnection) primaryDisplay() gpui.PlatformDisplay {
	if c.connected != nil {
		return c.connected.PrimaryDisplay()
	}
	return c.headless.PrimaryDisplay()
}

func (c *displayConnection) openWindow(handle gpui.AnyWindowHandle, params gpui.WindowParams) (gpui.PlatformWindow, error) {
	if c.connected != nil {
		return c.connected.OpenWindow(handle, params)
	}
	return c.headless.OpenWindow(handle, params)
}

func (c *displayConnection) activeWindow() (gpui.AnyWindowHandle, bool) {
	if c.connected != nil {
		return c.connected.ActiveWindow()
	}
	return gpui.AnyWindowHandle{}, false
}

func (c *displayConnection) windowStack() ([]gpui.AnyWindowHandle, bool) {
	if c.connected != nil {
		return c.connected.WindowStack()
	}
	return nil, false
}

func (c *displayConnection) setCursorStyle(style gpui.CursorStyle) {
	if c.connected != nil {
		c.connected.SetCursorStyle(style)
	}
}

func (c *displayConnection) hideCursorUntilMouseMoves() {
	if c.connected != nil {
		c.connected.HideCursorUntilMouseMoves()
	}
}

func (c *displayConnection) isCursorVisible() bool {
	if c.connected != nil {
		return c.connected.IsCursorVisible()
	}
	return true
}

func (c *displayConnection) openURI(uri string) {
	if c.connected != nil {
		c.connected.OpenURI(uri)
	}
}

func (c *displayConnection) revealPath(path string) {
	if c.connected != nil {
		c.connected.RevealPath(path)
	}
}

func (c *displayConnection) writeToPrimary(item gpui.ClipboardItem) {
	if c.connected != nil {
		c.connected.WriteToPrimary(item)
	}
}

func (c *displayConnection) writeToClipboard(item gpui.ClipboardItem) {
	if c.connected != nil {
		c.connected.WriteToClipboard(item)
	}
}

func (c *displayConnection) readFromPrimary() (gpui.ClipboardItem, bool) {
	if c.connected != nil {
		return c.connected.ReadFromPrimary()
	}
	return gpui.ClipboardItem{}, false
}

func (c *displayConnection) readFromClipboard() (gpui.ClipboardItem, bool) {
	if c.connected != nil {
		return c.connected.ReadFromClipboard()
	}
	return gpui.ClipboardItem{}, false
}

func (c *displayConnection) windowIdentifier() *gpui.Task[gpui.Option[string]] {
	if c.connected != nil {
		return c.connected.WindowIdentifier()
	}
	return scheduler.ReadyTask(gpui.None[string]())
}

func (c *displayConnection) isScreenCaptureSupported() bool {
	if c.connected != nil {
		return c.connected.IsScreenCaptureSupported()
	}
	return false
}

func (c *displayConnection) screenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	if c.connected != nil {
		return c.connected.ScreenCaptureSources()
	}
	sender, receiver := scheduler.Oneshot[scheduler.Result[[]gpui.ScreenCaptureSource]]()
	_ = sender.Send(scheduler.Result[[]gpui.ScreenCaptureSource]{Err: errors.New("Headless mode does not support screen capture.")})
	return receiver
}
