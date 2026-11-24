//go:build windows

package screencap

import gpui "github.com/megascan/gpui-go"

// GetOutputFrameSize returns the output frame size for options.
func GetOutputFrameSize(options gpui.ScapCapturerOptions) [2]uint32 {
	return winGetOutputFrameSize(options)
}

// Engine is the platform capture engine behind a Capturer.
type Engine struct {
	options gpui.ScapCapturerOptions
	target  gpui.ScapTarget
	win     *WCStream
	tx      *frameQueue
}

// NewEngine creates the Windows engine; frames are sent to tx.
func NewEngine(options gpui.ScapCapturerOptions, tx *frameQueue) (*Engine, error) {
	win, target, err := CreateCapturer(options, tx)
	if err != nil {
		return nil, err
	}
	return &Engine{win: win, options: options, target: target, tx: tx}, nil
}

// Start starts the engine.
func (e *Engine) Start() { e.win.StartCapture() }

// Stop stops the engine.
func (e *Engine) Stop() { e.win.StopCapture() }

// GetOutputFrameSize returns GetOutputFrameSize of the engine's options.
func (e *Engine) GetOutputFrameSize() [2]uint32 { return GetOutputFrameSize(e.options) }

// ProcessChannelItem turns a channel item into a frame; on Windows items are frames.
func (e *Engine) ProcessChannelItem(data gpui.ScapFrame) (gpui.ScapFrame, bool) { return data, true }

// Target returns the engine's target.
func (e *Engine) Target() (gpui.ScapTarget, bool) { return e.target, e.target != nil }

func (e *Engine) close() { e.tx.close() }
