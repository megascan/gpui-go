package main

import (
	"fmt"
	"strings"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

const usage = "commands: ls | create [message] | open [DISPLAY=… WAYLAND_DISPLAY=… XDG_RUNTIME_DIR=… XDG_ACTIVATION_TOKEN=…] | close | quit"

func newApplication() *gpui.Application {
	return platform.Linux(gpui.WindowingModesWayland | gpui.WindowingModesX11 | gpui.WindowingModesHeadless)
}

func displayEnvironment(arguments string) (gpui.GraphicalEnvironment, error) {
	if strings.TrimSpace(arguments) == "" {
		return gpui.GraphicalEnvironmentDetect(), nil
	}
	var environment gpui.GraphicalEnvironment
	for _, assignment := range strings.Fields(arguments) {
		name, value, ok := strings.Cut(assignment, "=")
		if !ok {
			return gpui.GraphicalEnvironment{}, fmt.Errorf("expected NAME=value, got %q", assignment)
		}
		switch name {
		case "WAYLAND_DISPLAY":
			environment.WaylandDisplay = gpui.Some(value)
		case "DISPLAY":
			environment.X11Display = gpui.Some(value)
		case "XDG_RUNTIME_DIR":
			environment.XdgRuntimeDir = gpui.Some(value)
		case "XDG_ACTIVATION_TOKEN":
			environment.ActivationToken = gpui.Some(value)
		default:
			return gpui.GraphicalEnvironment{}, fmt.Errorf("unknown display variable %s", name)
		}
	}
	return environment, nil
}
