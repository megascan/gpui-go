//go:build !linux && !freebsd && !windows

package gpui

import "os/exec"

// PlatformScreenCaptureFrame is the frame value on platforms without scap: on macOS a retained
// CVImageBufferRef owned by the ScreenCaptureKit back end, elsewhere zero.
type PlatformScreenCaptureFrame = uintptr

// GraphicalEnvironment is the graphical session to connect to; it carries nothing on this platform.
type GraphicalEnvironment struct{}

// GraphicalEnvironmentDetect returns the environment of this process's graphical session.
func GraphicalEnvironmentDetect() GraphicalEnvironment {
	return GraphicalEnvironment{}
}

// ApplyTo does nothing on this platform.
func (e GraphicalEnvironment) ApplyTo(command *exec.Cmd) {}

// DesktopEnvironment reports none: this platform has no desktop environments to tell apart.
func (e GraphicalEnvironment) DesktopEnvironment() (string, bool) {
	return "", false
}
