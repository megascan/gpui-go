//go:build linux

package screencap

import gpui "github.com/megascan/gpui-go"

// GetOutputFrameSize returns the output frame size for options; Linux cannot compute it.
func GetOutputFrameSize(options gpui.ScapCapturerOptions) [2]uint32 { return [2]uint32{0, 0} }

// Engine is the platform capture engine behind a Capturer.
type Engine struct {
	options gpui.ScapCapturerOptions
	target  gpui.ScapTarget
	linux   *LinuxCapturer
}

// NewEngine creates the Linux engine; frames are sent to tx.
func NewEngine(options gpui.ScapCapturerOptions, tx *frameQueue) (*Engine, error) {
	linux, err := CreateCapturer(options, tx)
	if err != nil {
		return nil, err
	}
	var target gpui.ScapTarget
	if t, ok := linux.Imp.Target(); ok {
		target = t
	}
	return &Engine{linux: linux, options: options, target: target}, nil
}

// Start starts the engine.
func (e *Engine) Start() { e.linux.Imp.StartCapture() }

// Stop stops the engine.
func (e *Engine) Stop() { e.linux.Imp.StopCapture() }

// GetOutputFrameSize returns GetOutputFrameSize of the engine's options.
func (e *Engine) GetOutputFrameSize() [2]uint32 { return GetOutputFrameSize(e.options) }

// ProcessChannelItem turns a channel item into a frame; on Linux items are frames.
func (e *Engine) ProcessChannelItem(data gpui.ScapFrame) (gpui.ScapFrame, bool) { return data, true }

// Target returns the engine's target.
func (e *Engine) Target() (gpui.ScapTarget, bool) { return e.target, e.target != nil }

func (e *Engine) close() {
	if c, ok := e.linux.Imp.(interface{ close() }); ok {
		c.close()
	}
}
