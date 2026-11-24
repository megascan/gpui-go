//go:build js && wasm

package web

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"syscall/js"

	"github.com/megascan/gpui-go"
)

const (
	maxRasterDimension = 4096
	maxRasterPixels    = 4 * 1024 * 1024
)

type canvasTextSlot struct {
	borrowed bool
	canvas   *textCanvas
}

var canvasTextState canvasTextSlot

type canvasTextMetrics struct {
	advance float32
	left    float32
	right   float32
	ascent  float32
	descent float32
}

type textCanvas struct {
	canvas  js.Value
	context js.Value
}

func newTextCanvas() (*textCanvas, error) {
	canvas, err := jsNew(js.Global().Get("OffscreenCanvas"), 1, 1)
	if err != nil {
		return nil, fmt.Errorf("creating text OffscreenCanvas: %s", jsErrorDebug(err))
	}
	options := js.Global().Get("Object").New()
	assignedValue, err := jsCall(js.Global().Get("Reflect"), "set", options, "willReadFrequently", true)
	if err != nil {
		return nil, fmt.Errorf("setting Canvas readback option: %s", jsErrorDebug(err))
	}
	if assigned, _ := jsBool(assignedValue); !assigned {
		return nil, errors.New("Canvas readback option could not be set")
	}
	context, err := jsCall(canvas, "getContext", "2d", options)
	if err != nil {
		return nil, fmt.Errorf("getting OffscreenCanvas 2D context: %s", jsErrorDebug(err))
	}
	if jsIsNullish(context) {
		return nil, errors.New("OffscreenCanvas 2D text rendering is unavailable")
	}
	if !jsInstanceOf(context, "OffscreenCanvasRenderingContext2D") {
		return nil, fmt.Errorf("unexpected OffscreenCanvas 2D context: %s", jsDebug(context))
	}
	return &textCanvas{canvas: canvas, context: context}, nil
}

func (c *textCanvas) configure(cssFont string) error {
	if strings.TrimSpace(cssFont) == "" {
		return errors.New("Canvas text font is empty")
	}
	if err := jsTry(func() {
		c.context.Set("font", "10px sans-serif")
		c.context.Set("font", cssFont)
		c.context.Set("textAlign", "left")
		c.context.Set("textBaseline", "alphabetic")
		c.context.Set("fillStyle", "white")
	}); err != nil {
		return err
	}
	assignedValue, err := jsCall(js.Global().Get("Reflect"), "set", c.context, "direction", "ltr")
	if err != nil {
		return fmt.Errorf("setting Canvas text direction: %s", jsErrorDebug(err))
	}
	if assigned, _ := jsBool(assignedValue); !assigned {
		return errors.New("Canvas text direction could not be set")
	}
	return nil
}

func (c *textCanvas) dimension(name string) uint32 {
	v, _ := jsNumber(jsGet(c.canvas, name))
	return uint32(v)
}

func withCanvas[T any](operation func(*textCanvas) (T, error)) (T, error) {
	var zero T
	if canvasTextState.borrowed {
		return zero, errors.New("Canvas text renderer was called reentrantly")
	}
	canvasTextState.borrowed = true
	defer func() { canvasTextState.borrowed = false }()
	if canvasTextState.canvas == nil {
		canvas, err := newTextCanvas()
		if err != nil {
			return zero, err
		}
		canvasTextState.canvas = canvas
	}
	return operation(canvasTextState.canvas)
}

func canvasMeasure(text, cssFont string) (canvasTextMetrics, error) {
	return withCanvas(func(canvas *textCanvas) (canvasTextMetrics, error) {
		if err := canvas.configure(cssFont); err != nil {
			return canvasTextMetrics{}, err
		}
		measured, err := jsCall(canvas.context, "measureText", text)
		if err != nil {
			return canvasTextMetrics{}, fmt.Errorf("measuring Canvas text: %s", jsErrorDebug(err))
		}
		number := func(name string) float32 {
			v, ok := jsNumber(jsGet(measured, name))
			if !ok {
				return float32(math.NaN())
			}
			return float32(v)
		}
		metrics := canvasTextMetrics{
			advance: number("width"),
			left:    -number("actualBoundingBoxLeft"),
			right:   number("actualBoundingBoxRight"),
			ascent:  number("actualBoundingBoxAscent"),
			descent: number("actualBoundingBoxDescent"),
		}
		for _, v := range []float32{metrics.advance, metrics.left, metrics.right, metrics.ascent, metrics.descent} {
			if !isFiniteF32(v) {
				return canvasTextMetrics{}, errors.New("Canvas returned non-finite text metrics")
			}
		}
		return metrics, nil
	})
}

func isFiniteF32(v float32) bool {
	return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
}

func canvasRasterize(text, cssFont string, bounds gpui.Bounds[gpui.DevicePixels], subpixelOffset [2]float32, color bool) ([]byte, error) {
	if !isFiniteF32(subpixelOffset[0]) || !isFiniteF32(subpixelOffset[1]) {
		return nil, errors.New("Canvas text subpixel offset must be finite")
	}
	if bounds.Size.Width < 0 {
		return nil, errors.New("Canvas text raster width must be nonnegative")
	}
	if bounds.Size.Height < 0 {
		return nil, errors.New("Canvas text raster height must be nonnegative")
	}
	width := uint32(bounds.Size.Width)
	height := uint32(bounds.Size.Height)
	if width == 0 || height == 0 {
		return []byte{}, nil
	}
	if width > maxRasterDimension || height > maxRasterDimension {
		return nil, fmt.Errorf("Canvas text raster exceeds maximum dimension %d: %dx%d", maxRasterDimension, width, height)
	}
	pixelCount := int(width) * int(height)
	if pixelCount > maxRasterPixels {
		return nil, fmt.Errorf("Canvas text raster exceeds maximum pixel count %d: %dx%d", maxRasterPixels, width, height)
	}
	byteCount := pixelCount * 4

	return withCanvas(func(canvas *textCanvas) ([]byte, error) {
		if err := jsTry(func() {
			if height < canvas.dimension("height") {
				canvas.canvas.Set("height", height)
			}
			if canvas.dimension("width") != width {
				canvas.canvas.Set("width", width)
			}
			if canvas.dimension("height") != height {
				canvas.canvas.Set("height", height)
			}
		}); err != nil {
			return nil, err
		}
		if err := canvas.configure(cssFont); err != nil {
			return nil, err
		}
		if _, err := jsCall(canvas.context, "clearRect", 0.0, 0.0, float64(width), float64(height)); err != nil {
			return nil, err
		}
		if _, err := jsCall(canvas.context, "fillText", text,
			-float64(bounds.Origin.X)+float64(subpixelOffset[0]),
			-float64(bounds.Origin.Y)+float64(subpixelOffset[1])); err != nil {
			return nil, fmt.Errorf("drawing Canvas text: %s", jsErrorDebug(err))
		}
		image, err := jsCall(canvas.context, "getImageData", 0.0, 0.0, float64(width), float64(height))
		if err != nil {
			return nil, fmt.Errorf("reading Canvas text raster pixels: %s", jsErrorDebug(err))
		}
		data := jsGet(image, "data")
		length, _ := jsNumber(jsGet(data, "length"))
		pixels := make([]byte, int(length))
		js.CopyBytesToGo(pixels, data)
		if len(pixels) != byteCount {
			return nil, fmt.Errorf("Canvas text raster returned %d bytes, expected %d", len(pixels), byteCount)
		}
		if color {
			for i := 0; i+3 < len(pixels); i += 4 {
				pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
			}
			return pixels, nil
		}
		alpha := make([]byte, 0, pixelCount)
		for i := 3; i < len(pixels); i += 4 {
			alpha = append(alpha, pixels[i])
		}
		return alpha, nil
	})
}
