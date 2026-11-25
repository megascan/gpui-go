//go:build linux

// Package linux ports gpui_linux: LinuxPlatform, the Platform for Linux, which connects to a
// Wayland compositor or an X server, or runs headless, and switches between those modes.
package linux

import (
	"os"

	"github.com/megascan/gpui-go"
)

const windowingModesAll = gpui.WindowingModesWayland | gpui.WindowingModesX11 | gpui.WindowingModesHeadless

// CurrentPlatform returns the default platform implementation for the current OS.
//
// A windowed platform connects to the display server the process environment names, or starts
// headless when it names none, and can switch between those modes later. A headless platform
// stays headless.
func CurrentPlatform(headless bool) gpui.Platform {
	if _, ok := os.LookupEnv("ZED_HEADLESS"); headless || ok {
		return LinuxPlatform(gpui.WindowingModesHeadless)
	}
	return LinuxPlatform(windowingModesAll)
}

// LinuxPlatform returns a platform that may switch among allowedModes.
//
// It starts windowed in the process's own environment, or headless if that names no allowed
// display server. Set another initial mode with gpui.Application.WithWindowing.
func LinuxPlatform(allowedModes gpui.WindowingModes) gpui.Platform {
	return newLinuxPlatform(allowedModes)
}
