//go:build linux

package common

import "github.com/megascan/gpui-go"

type windowLease struct{ count int }

// HeadlessConnection is the state of a LinuxPlatform with no display server: it reports no
// displays and opens HeadlessWindows, which lay out and handle input but draw nothing.
type HeadlessConnection struct {
	windowLease *windowLease
}

// NewHeadlessConnection returns a connection with no open windows.
func NewHeadlessConnection() *HeadlessConnection {
	return &HeadlessConnection{windowLease: &windowLease{}}
}

// HasWindows reports whether any window opened on this connection is still open.
func (c *HeadlessConnection) HasWindows() bool {
	return c.windowLease.count > 0
}

// Displays returns no displays.
func (c *HeadlessConnection) Displays() []gpui.PlatformDisplay {
	return nil
}

// PrimaryDisplay returns nil.
func (c *HeadlessConnection) PrimaryDisplay() gpui.PlatformDisplay {
	return nil
}

// OpenWindow opens a HeadlessWindow.
func (c *HeadlessConnection) OpenWindow(handle gpui.AnyWindowHandle, params gpui.WindowParams) (gpui.PlatformWindow, error) {
	return newHeadlessWindow(params, c.windowLease), nil
}
