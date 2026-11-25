//go:build windows

// Package screencap ports zed-scap's Windows engine on Windows.Graphics.Capture (internal/win/wgc).
// It implements gpui.ScapBackend.
package screencap

import gpui "github.com/megascan/gpui-go"

// GetAllTargets returns the targets that can be captured.
func GetAllTargets() ([]gpui.ScapTarget, error) { return winGetAllTargets() }

// GetScaleFactor returns the target's scale factor.
func GetScaleFactor(target gpui.ScapTarget) float64 { return winGetScaleFactor(target) }

// GetMainDisplay returns the primary display.
func GetMainDisplay() (*gpui.ScapDisplay, error) { return winGetMainDisplay() }

// GetTargetDimensions returns the target's width and height in pixels.
func GetTargetDimensions(target gpui.ScapTarget) (uint64, uint64) {
	return winGetTargetDimensions(target)
}
