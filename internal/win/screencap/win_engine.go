//go:build windows

package screencap

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/wgc"
)

type winCapturer struct {
	tx   *frameQueue
	crop gpui.Option[gpui.ScapArea]
}

type flagStruct struct {
	tx   *frameQueue
	crop gpui.Option[gpui.ScapArea]
}

// WCStream is a Windows.Graphics.Capture stream for one target.
type WCStream struct {
	settings       wgc.Settings[flagStruct]
	captureControl *wgc.CaptureControl
}

func newWinCapturer(ctx wgc.Context[flagStruct]) (wgc.GraphicsCaptureApiHandler, error) {
	return &winCapturer{tx: ctx.Flags.tx, crop: ctx.Flags.crop}, nil
}

func f64ToU32(v float64) uint32 {
	switch {
	case math.IsNaN(v) || v <= 0:
		return 0
	case v >= math.MaxUint32:
		return math.MaxUint32
	}
	return uint32(v)
}

func f64ToI32(v float64) int32 {
	switch {
	case math.IsNaN(v):
		return 0
	case v <= math.MinInt32:
		return math.MinInt32
	case v >= math.MaxInt32:
		return math.MaxInt32
	}
	return int32(v)
}

func (c *winCapturer) OnFrameArrived(frame *wgc.Frame, _ wgc.InternalCaptureControl) error {
	colorFormat := frame.ColorFormat()
	var buffer *wgc.FrameBuffer
	if croppedArea, ok := c.crop.Get(); ok {
		startX := min(f64ToU32(croppedArea.X), frame.Width())
		startY := min(f64ToU32(croppedArea.Y), frame.Height())
		endX := min(f64ToU32(croppedArea.X+croppedArea.Width), frame.Width())
		endY := min(f64ToU32(croppedArea.Y+croppedArea.Height), frame.Height())
		croppedBuffer, err := frame.BufferCrop(startX, startY, endX, endY)
		if err != nil {
			return fmt.Errorf("Failed to crop buffer: %w", err)
		}
		buffer = croppedBuffer
	} else {
		frameBuffer, err := frame.Buffer()
		if err != nil {
			return fmt.Errorf("Failed to get frame buffer: %w", err)
		}
		buffer = frameBuffer
	}
	raw, err := buffer.AsNopaddingBuffer()
	if err != nil {
		return errors.New("Failed to get raw buffer")
	}
	width := int32(buffer.Width())
	height := int32(buffer.Height())
	data := append([]byte(nil), raw...)

	displayTime := uint64(time.Now().UnixNano())

	var out gpui.ScapFrame
	switch colorFormat {
	case wgc.ColorFormatRgba16F:
		return errors.New("Rgba16F is not yet supported")
	case wgc.ColorFormatRgba8:
		out = gpui.ScapFrameRGBx{ScapPixels: gpui.ScapPixels{DisplayTime: displayTime, Width: width, Height: height, Data: data}}
	case wgc.ColorFormatBgra8:
		out = gpui.ScapFrameBGRA{ScapPixels: gpui.ScapPixels{DisplayTime: displayTime, Width: width, Height: height, Data: data}}
	}
	return c.tx.send(out, nil)
}

func (c *winCapturer) OnClosed() error {
	slog.Debug("Screen capture stream closed.")
	return nil
}

// StartCapture starts the capture thread. A start failure, and the error the thread later ends
// with, are delivered to the frame queue, which is then closed.
func (s *WCStream) StartCapture() {
	tx := s.settings.Flags().tx
	cc, err := wgc.StartFreeThreaded(s.settings, newWinCapturer)
	if err != nil {
		tx.finish(err)
		return
	}
	s.captureControl = cc
	go func() {
		tx.finish(cc.ExitErr())
	}()
}

// StopCapture stops the capture thread; it panics when the capture was not started.
func (s *WCStream) StopCapture() {
	captureControl := s.captureControl
	s.captureControl = nil
	if captureControl == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	if err := captureControl.Stop(); err != nil {
		s.settings.Flags().tx.finish(err)
	}
}

func mainDisplayTarget(options gpui.ScapCapturerOptions) (gpui.ScapTarget, error) {
	if options.Target != nil {
		return options.Target, nil
	}
	display, err := GetMainDisplay()
	if err != nil {
		return nil, fmt.Errorf("Failed to get main display: %w", err)
	}
	return display, nil
}

// CreateCapturer creates the stream for options; frames are sent to tx.
func CreateCapturer(options gpui.ScapCapturerOptions, tx *frameQueue) (*WCStream, gpui.ScapTarget, error) {
	target, err := mainDisplayTarget(options)
	if err != nil {
		return nil, nil, err
	}
	cropArea, err := GetCropArea(options)
	if err != nil {
		return nil, nil, err
	}

	colorFormat := wgc.ColorFormatRgba8
	if options.OutputType == gpui.ScapFrameTypeBGRAFrame {
		colorFormat = wgc.ColorFormatBgra8
	}

	showCursor := wgc.CursorCaptureSettingsWithoutCursor
	if options.ShowCursor {
		showCursor = wgc.CursorCaptureSettingsWithCursor
	}

	var item wgc.TryIntoCaptureItemWithType
	switch t := target.(type) {
	case *gpui.ScapDisplay:
		item = wgc.MonitorFromRawHMonitor(t.RawHandle)
	case *gpui.ScapWindow:
		item = wgc.WindowFromRawHWND(t.RawHandle)
	default:
		return nil, nil, errors.New("screencap: unknown target type")
	}
	settings := wgc.NewSettings(item, showCursor, wgc.DrawBorderSettingsDefault,
		wgc.SecondaryWindowSettingsDefault, wgc.MinimumUpdateIntervalSettings{}, wgc.DirtyRegionSettingsDefault,
		colorFormat, flagStruct{tx: tx, crop: gpui.Some(cropArea)})

	return &WCStream{settings: settings}, target, nil
}

func winGetOutputFrameSize(options gpui.ScapCapturerOptions) [2]uint32 {
	cropArea, err := GetCropArea(options)
	if err != nil {
		return [2]uint32{}
	}

	outputWidth := f64ToU32(cropArea.Width)
	outputHeight := f64ToU32(cropArea.Height)

	if options.OutputResolution != gpui.ScapResolutionCaptured {
		resolved := resolutionValue(options.OutputResolution, float32(cropArea.Width)/float32(cropArea.Height))
		outputWidth = min(outputWidth, resolved[0])
		outputHeight = min(outputHeight, resolved[1])
	}

	outputWidth -= outputWidth % 2
	outputHeight -= outputHeight % 2

	return [2]uint32{outputWidth, outputHeight}
}

func getAbsoluteValue(value, scaleFactor float64) float64 {
	v := math.Floor(value * scaleFactor)
	return v + math.Mod(v, 2.0)
}

// GetCropArea returns the crop area in physical pixels: the option scaled by the target's scale
// factor, or the whole target. Frames clamp it to their own size.
func GetCropArea(options gpui.ScapCapturerOptions) (gpui.ScapArea, error) {
	target, err := mainDisplayTarget(options)
	if err != nil {
		return gpui.ScapArea{}, err
	}

	width, height := GetTargetDimensions(target)

	scaleFactor := GetScaleFactor(target)
	if val, ok := options.CropArea.Get(); ok {
		return gpui.ScapArea{
			X:      getAbsoluteValue(val.X, scaleFactor),
			Y:      getAbsoluteValue(val.Y, scaleFactor),
			Width:  getAbsoluteValue(val.Width, scaleFactor),
			Height: getAbsoluteValue(val.Height, scaleFactor),
		}, nil
	}
	return gpui.ScapArea{X: 0, Y: 0, Width: float64(width), Height: float64(height)}, nil
}
