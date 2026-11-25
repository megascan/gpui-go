//go:build linux || freebsd

package gpui

import (
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"
)

// GuessCompositor returns which compositor we're guessing we'll use; it does not attempt to connect.
func GuessCompositor() string {
	if _, ok := os.LookupEnv("ZED_HEADLESS"); ok {
		return "headless"
	}
	return GraphicalEnvironmentDetect().GuessCompositor()
}

// GraphicalEnvironment is the graphical session to connect to: the variables that locate its
// display server, as some process sees them.
type GraphicalEnvironment struct {
	// WaylandDisplay is WAYLAND_DISPLAY: a socket name relative to XdgRuntimeDir, or an absolute path.
	WaylandDisplay Option[string]
	// X11Display is DISPLAY: the X11 display name.
	X11Display Option[string]
	// XdgRuntimeDir is XDG_RUNTIME_DIR: the directory containing Wayland sockets.
	XdgRuntimeDir Option[string]
	// CurrentDesktop is XDG_CURRENT_DESKTOP; ApplyTo leaves it alone.
	CurrentDesktop Option[string]
	// ActivationToken is XDG_ACTIVATION_TOKEN: lets the first window take focus on Wayland.
	ActivationToken Option[string]
}

func lookupEnvOption(name string) Option[string] {
	value, ok := os.LookupEnv(name)
	return OptionFrom(value, ok)
}

// GraphicalEnvironmentDetect reads the display variables from this process's environment; it
// leaves ActivationToken unset.
func GraphicalEnvironmentDetect() GraphicalEnvironment {
	return GraphicalEnvironment{
		WaylandDisplay:  lookupEnvOption("WAYLAND_DISPLAY"),
		X11Display:      lookupEnvOption("DISPLAY"),
		XdgRuntimeDir:   lookupEnvOption("XDG_RUNTIME_DIR"),
		CurrentDesktop:  lookupEnvOption("XDG_CURRENT_DESKTOP"),
		ActivationToken: None[string](),
	}
}

// ApplyTo sets this environment's display variables on command and removes the ones it doesn't
// set; XDG_ACTIVATION_TOKEN is left alone.
func (e GraphicalEnvironment) ApplyTo(command *exec.Cmd) {
	if command.Env == nil {
		command.Env = os.Environ()
	}
	for _, entry := range []struct {
		name  string
		value Option[string]
	}{
		{"WAYLAND_DISPLAY", e.WaylandDisplay},
		{"DISPLAY", e.X11Display},
		{"XDG_RUNTIME_DIR", e.XdgRuntimeDir},
	} {
		env := command.Env[:0:0]
		for _, kv := range command.Env {
			if name, _, _ := strings.Cut(kv, "="); name != entry.name {
				env = append(env, kv)
			}
		}
		if value, ok := entry.value.Get(); ok {
			env = append(env, entry.name+"="+value)
		}
		command.Env = env
	}
}

// DesktopEnvironment returns the session's desktop environment, if XDG_CURRENT_DESKTOP names one.
func (e GraphicalEnvironment) DesktopEnvironment() (string, bool) {
	desktop, ok := e.CurrentDesktop.Get()
	if !ok || !utf8.ValidString(desktop) || desktop == "" {
		return "", false
	}
	return desktop, true
}

// GuessCompositor returns the compositor this environment selects: Wayland, then X11, then headless.
func (e GraphicalEnvironment) GuessCompositor() string {
	isSet := func(value Option[string]) bool {
		v, ok := value.Get()
		return ok && v != ""
	}
	if isSet(e.WaylandDisplay) {
		return "Wayland"
	} else if isSet(e.X11Display) {
		return "X11"
	}
	return "headless"
}

// WindowingModes is the set of windowing modes a platform may start in or switch to.
type WindowingModes uint8

const (
	// WindowingModesWayland is connected to a Wayland compositor.
	WindowingModesWayland WindowingModes = 1 << 0
	// WindowingModesX11 is connected to an X server.
	WindowingModesX11 WindowingModes = 1 << 1
	// WindowingModesHeadless has no display server.
	WindowingModesHeadless WindowingModes = 1 << 2
)

// Contains reports whether every flag of other is set in m.
func (m WindowingModes) Contains(other WindowingModes) bool {
	return m&other == other
}
