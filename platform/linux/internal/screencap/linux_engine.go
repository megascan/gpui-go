//go:build linux

package screencap

import (
	"errors"
	"log/slog"
	"os"

	gpui "github.com/megascan/gpui-go"
)

// LinuxCapturerImpl is one Linux capture engine.
type LinuxCapturerImpl interface {
	StartCapture()
	StopCapture()
	Target() (gpui.ScapTarget, bool)
}

// LinuxCapturer wraps the engine picked for the session.
type LinuxCapturer struct {
	Imp LinuxCapturerImpl
}

const unsupportedPlatformMsg = "Unsupported platform. Could not detect Wayland or X11 displays"

// NewLinuxCapturer picks the Wayland engine when WAYLAND_DISPLAY is set, else the X11 engine when
// DISPLAY is set.
func NewLinuxCapturer(options gpui.ScapCapturerOptions, tx *frameQueue) (*LinuxCapturer, error) {
	if _, ok := os.LookupEnv("WAYLAND_DISPLAY"); ok {
		slog.Debug("Creating new Wayland screen capturer.")
		imp, err := NewWaylandCapturer(options, tx)
		if err != nil {
			return nil, err
		}
		return &LinuxCapturer{Imp: imp}, nil
	}
	if _, ok := os.LookupEnv("DISPLAY"); ok {
		slog.Debug("Creating new X11 screen capturer.")
		imp, err := NewX11Capturer(options, tx)
		if err != nil {
			return nil, err
		}
		return &LinuxCapturer{Imp: imp}, nil
	}
	return nil, errors.New(unsupportedPlatformMsg)
}

// CreateCapturer is NewLinuxCapturer.
func CreateCapturer(options gpui.ScapCapturerOptions, tx *frameQueue) (*LinuxCapturer, error) {
	return NewLinuxCapturer(options, tx)
}
