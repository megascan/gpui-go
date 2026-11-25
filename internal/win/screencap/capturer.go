//go:build windows

package screencap

import (
	"math"

	gpui "github.com/megascan/gpui-go"
)

func resolutionValue(r gpui.ScapResolution, aspectRatio float32) [2]uint32 {
	w := map[gpui.ScapResolution]float32{
		gpui.ScapResolution480p:  640,
		gpui.ScapResolution720p:  1280,
		gpui.ScapResolution1080p: 1920,
		gpui.ScapResolution1440p: 2560,
		gpui.ScapResolution2160p: 3840,
		gpui.ScapResolution4320p: 7680,
	}
	width, ok := w[r]
	if !ok {
		panic(".value should not be called when Resolution type is Captured")
	}
	return [2]uint32{uint32(width), uint32(math.Floor(float64(width / aspectRatio)))}
}

// CapturerBuildError is returned by Build when capture is unavailable.
type CapturerBuildError uint8

const (
	// CapturerBuildErrorNotSupported means screen capture is not supported.
	CapturerBuildErrorNotSupported CapturerBuildError = iota
	// CapturerBuildErrorPermissionNotGranted means permission was not granted.
	CapturerBuildErrorPermissionNotGranted
)

// Error returns the upstream message.
func (e CapturerBuildError) Error() string {
	if e == CapturerBuildErrorNotSupported {
		return "Screen capturing is not supported"
	}
	return "Permission to capture the screen is not granted"
}

// Capturer is a screen capturer (scap::capturer::Capturer).
type Capturer struct {
	engine *Engine
	rx     *frameQueue
}

// NewCapturer builds a capturer without the support and permission checks.
//
// Deprecated: Use Build instead.
func NewCapturer(options gpui.ScapCapturerOptions) (*Capturer, error) {
	rx := newFrameQueue()
	engine, err := NewEngine(options, rx)
	if err != nil {
		return nil, err
	}
	return &Capturer{engine: engine, rx: rx}, nil
}

// Build creates a capturer with options.
func Build(options gpui.ScapCapturerOptions) (*Capturer, error) {
	if !IsSupported() {
		return nil, CapturerBuildErrorNotSupported
	}
	if !HasPermission() {
		return nil, CapturerBuildErrorPermissionNotGranted
	}
	rx := newFrameQueue()
	engine, err := NewEngine(options, rx)
	if err != nil {
		return nil, err
	}
	return &Capturer{engine: engine, rx: rx}, nil
}

// StartCapture starts capturing frames.
func (c *Capturer) StartCapture() { c.engine.Start() }

// StopCapture stops the capturer.
func (c *Capturer) StopCapture() { c.engine.Stop() }

// GetNextFrame blocks until the next frame; it fails once the engine has stopped.
func (c *Capturer) GetNextFrame() (gpui.ScapFrame, error) {
	for {
		res, err := c.rx.recv()
		if err != nil {
			return nil, err
		}
		if res.err != nil {
			return nil, res.err
		}
		if frame, ok := c.engine.ProcessChannelItem(res.frame); ok {
			return frame, nil
		}
	}
}

// GetOutputFrameSize returns the frame size the engine reports.
func (c *Capturer) GetOutputFrameSize() [2]uint32 { return c.engine.GetOutputFrameSize() }

// Raw returns a RawCapturer over c.
func (c *Capturer) Raw() RawCapturer { return RawCapturer{capturer: c} }

// Target returns the captured target, or nil when the engine does not know it.
func (c *Capturer) Target() gpui.ScapTarget {
	t, ok := c.engine.Target()
	if !ok {
		return nil
	}
	return t
}

// Close releases the engine's resources (the drop of Capturer).
func (c *Capturer) Close() { c.engine.close() }

// RawCapturer borrows a Capturer.
type RawCapturer struct{ capturer *Capturer }
