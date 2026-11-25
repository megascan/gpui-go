//go:build linux

// Package screencap ports zed-scap's Linux engines: the xdg-desktop-portal ScreenCast session
// with a PipeWire stream on Wayland and root-window capture on X11. It implements gpui.ScapBackend.
package screencap

import gpui "github.com/megascan/gpui-go"

// GetAllTargets returns the targets that can be captured.
func GetAllTargets() ([]gpui.ScapTarget, error) { return linuxGetAllTargets() }

// GetScaleFactor returns the target's scale factor; always 1 on Linux.
func GetScaleFactor(target gpui.ScapTarget) float64 { return 1.0 }

// GetMainDisplay returns the primary display.
func GetMainDisplay() (*gpui.ScapDisplay, error) { return linuxGetMainDisplay() }

// GetTargetDimensions returns the target's width and height; windows report 0, 0.
func GetTargetDimensions(target gpui.ScapTarget) (uint64, uint64) {
	return linuxGetTargetDimensions(target)
}
