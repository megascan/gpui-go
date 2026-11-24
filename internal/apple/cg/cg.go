//go:build darwin

// Package cg binds the CoreGraphics bitmap contexts, colour spaces and fonts the Apple text system
// uses, on purego without cgo.
package cg

import (
	"fmt"
	"runtime"
	"structs"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

// CoreGraphicsPath is the install path of CoreGraphics.framework.
const CoreGraphicsPath = "/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics"

// Float is CGFloat.
type Float = objc.CGFloat

// Point is CGPoint.
type Point = objc.CGPoint

// Size is CGSize.
type Size = objc.CGSize

// Rect is CGRect.
type Rect = objc.CGRect

// Glyph is CGGlyph.
type Glyph = uint16

// AffineTransform is CGAffineTransform.
type AffineTransform struct {
	_  structs.HostLayout
	A  Float
	B  Float
	C  Float
	D  Float
	Tx Float
	Ty Float
}

// ColorSpace is a CGColorSpaceRef.
type ColorSpace uintptr

// Context is a CGContextRef.
type Context uintptr

// Font is a CGFontRef.
type Font uintptr

// TextDrawingMode is CGTextDrawingMode.
type TextDrawingMode int32

// TextFill is kCGTextFill.
const TextFill TextDrawingMode = 0

// ImageAlphaPremultipliedLast is kCGImageAlphaPremultipliedLast.
const ImageAlphaPremultipliedLast uint32 = 1

// ImageAlphaOnly is kCGImageAlphaOnly.
const ImageAlphaOnly uint32 = 7

var (
	loadOnce sync.Once
	loadErr  error

	cgColorSpaceCreateDeviceRGB                func() ColorSpace
	cgColorSpaceCreateDeviceGray               func() ColorSpace
	cgColorSpaceRelease                        func(cs ColorSpace)
	cgBitmapContextCreate                      func(data unsafe.Pointer, width, height, bitsPerComponent, bytesPerRow uint, cs ColorSpace, info uint32) Context
	cgContextRelease                           func(c Context)
	cgContextTranslateCTM                      func(c Context, tx, ty Float)
	cgContextScaleCTM                          func(c Context, sx, sy Float)
	cgContextSetTextDrawingMode                func(c Context, mode TextDrawingMode)
	cgContextSetAllowsAntialiasing             func(c Context, v bool)
	cgContextSetShouldAntialias                func(c Context, v bool)
	cgContextSetAllowsFontSmoothing            func(c Context, v bool)
	cgContextSetShouldSmoothFonts              func(c Context, v bool)
	cgContextSetAllowsFontSubpixelPositioning  func(c Context, v bool)
	cgContextSetShouldSubpixelPositionFonts    func(c Context, v bool)
	cgContextSetAllowsFontSubpixelQuantization func(c Context, v bool)
	cgContextSetShouldSubpixelQuantizeFonts    func(c Context, v bool)
	cgContextSetGrayFillColor                  func(c Context, gray, alpha Float)
	cgContextSetRGBFillColor                   func(c Context, r, g, b, a Float)
	cgContextFillRect                          func(c Context, r Rect)
	cgFontRelease                              func(f Font)
	cgFontRetain                               func(f Font) Font
)

// Load opens CoreGraphics and resolves its symbols. It is safe to call repeatedly.
func Load() error {
	loadOnce.Do(func() {
		h, err := purego.Dlopen(CoreGraphicsPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("cg: open %s: %w", CoreGraphicsPath, err)
			return
		}
		for _, b := range []struct {
			fptr any
			name string
		}{
			{&cgColorSpaceCreateDeviceRGB, "CGColorSpaceCreateDeviceRGB"},
			{&cgColorSpaceCreateDeviceGray, "CGColorSpaceCreateDeviceGray"},
			{&cgColorSpaceRelease, "CGColorSpaceRelease"},
			{&cgBitmapContextCreate, "CGBitmapContextCreate"},
			{&cgContextRelease, "CGContextRelease"},
			{&cgContextTranslateCTM, "CGContextTranslateCTM"},
			{&cgContextScaleCTM, "CGContextScaleCTM"},
			{&cgContextSetTextDrawingMode, "CGContextSetTextDrawingMode"},
			{&cgContextSetAllowsAntialiasing, "CGContextSetAllowsAntialiasing"},
			{&cgContextSetShouldAntialias, "CGContextSetShouldAntialias"},
			{&cgContextSetAllowsFontSmoothing, "CGContextSetAllowsFontSmoothing"},
			{&cgContextSetShouldSmoothFonts, "CGContextSetShouldSmoothFonts"},
			{&cgContextSetAllowsFontSubpixelPositioning, "CGContextSetAllowsFontSubpixelPositioning"},
			{&cgContextSetShouldSubpixelPositionFonts, "CGContextSetShouldSubpixelPositionFonts"},
			{&cgContextSetAllowsFontSubpixelQuantization, "CGContextSetAllowsFontSubpixelQuantization"},
			{&cgContextSetShouldSubpixelQuantizeFonts, "CGContextSetShouldSubpixelQuantizeFonts"},
			{&cgContextSetGrayFillColor, "CGContextSetGrayFillColor"},
			{&cgContextSetRGBFillColor, "CGContextSetRGBFillColor"},
			{&cgContextFillRect, "CGContextFillRect"},
			{&cgFontRelease, "CGFontRelease"},
			{&cgFontRetain, "CGFontRetain"},
		} {
			addr, err := purego.Dlsym(h, b.name)
			if err != nil {
				loadErr = fmt.Errorf("cg: %w", err)
				return
			}
			purego.RegisterFunc(b.fptr, addr)
		}
	})
	return loadErr
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// ColorSpaceCreateDeviceRGB returns a new device RGB colour space; Release it.
func ColorSpaceCreateDeviceRGB() ColorSpace {
	mustLoad()
	return cgColorSpaceCreateDeviceRGB()
}

// ColorSpaceCreateDeviceGray returns a new device gray colour space; Release it.
func ColorSpaceCreateDeviceGray() ColorSpace {
	mustLoad()
	return cgColorSpaceCreateDeviceGray()
}

// Release releases the colour space. A zero colour space is ignored.
func (cs ColorSpace) Release() {
	mustLoad()
	if cs != 0 {
		cgColorSpaceRelease(cs)
	}
}

// BitmapContext is a CGBitmapContext drawing into Go memory that stays pinned until Release.
type BitmapContext struct {
	Context
	pinner runtime.Pinner
	data   []byte
}

// CreateBitmapContext returns a bitmap context that draws into data, which must hold at least
// height*bytesPerRow bytes. It returns nil when CGBitmapContextCreate fails.
func CreateBitmapContext(data []byte, width, height, bitsPerComponent, bytesPerRow uint, cs ColorSpace, info uint32) *BitmapContext {
	mustLoad()
	if uint(len(data)) < height*bytesPerRow {
		panic("cg: bitmap buffer is smaller than height*bytesPerRow")
	}
	b := &BitmapContext{data: data}
	var p unsafe.Pointer
	if len(data) > 0 {
		p = unsafe.Pointer(&data[0])
		b.pinner.Pin(p)
	}
	c := cgBitmapContextCreate(p, width, height, bitsPerComponent, bytesPerRow, cs, info)
	if c == 0 {
		b.pinner.Unpin()
		return nil
	}
	b.Context = c
	return b
}

// Release releases the context and unpins its buffer. It is idempotent.
func (b *BitmapContext) Release() {
	if b == nil || b.Context == 0 {
		return
	}
	cgContextRelease(b.Context)
	b.Context = 0
	b.pinner.Unpin()
}

// TranslateCTM translates the current transformation matrix.
func (c Context) TranslateCTM(tx, ty Float) { cgContextTranslateCTM(c, tx, ty) }

// ScaleCTM scales the current transformation matrix.
func (c Context) ScaleCTM(sx, sy Float) { cgContextScaleCTM(c, sx, sy) }

// SetTextDrawingMode sets the text drawing mode.
func (c Context) SetTextDrawingMode(mode TextDrawingMode) { cgContextSetTextDrawingMode(c, mode) }

// SetAllowsAntialiasing sets whether antialiasing is allowed.
func (c Context) SetAllowsAntialiasing(v bool) { cgContextSetAllowsAntialiasing(c, v) }

// SetShouldAntialias sets whether to antialias.
func (c Context) SetShouldAntialias(v bool) { cgContextSetShouldAntialias(c, v) }

// SetAllowsFontSmoothing sets whether font smoothing is allowed.
func (c Context) SetAllowsFontSmoothing(v bool) { cgContextSetAllowsFontSmoothing(c, v) }

// SetShouldSmoothFonts sets whether to smooth fonts.
func (c Context) SetShouldSmoothFonts(v bool) { cgContextSetShouldSmoothFonts(c, v) }

// SetAllowsFontSubpixelPositioning sets whether subpixel glyph positioning is allowed.
func (c Context) SetAllowsFontSubpixelPositioning(v bool) {
	cgContextSetAllowsFontSubpixelPositioning(c, v)
}

// SetShouldSubpixelPositionFonts sets whether to position glyphs at subpixel offsets.
func (c Context) SetShouldSubpixelPositionFonts(v bool) {
	cgContextSetShouldSubpixelPositionFonts(c, v)
}

// SetAllowsFontSubpixelQuantization sets whether subpixel quantization is allowed.
func (c Context) SetAllowsFontSubpixelQuantization(v bool) {
	cgContextSetAllowsFontSubpixelQuantization(c, v)
}

// SetShouldSubpixelQuantizeFonts sets whether to quantize subpixel glyph positions.
func (c Context) SetShouldSubpixelQuantizeFonts(v bool) {
	cgContextSetShouldSubpixelQuantizeFonts(c, v)
}

// SetGrayFillColor sets the fill colour to a gray level and alpha.
func (c Context) SetGrayFillColor(gray, alpha Float) { cgContextSetGrayFillColor(c, gray, alpha) }

// SetRGBFillColor sets the fill colour.
func (c Context) SetRGBFillColor(r, g, b, a Float) { cgContextSetRGBFillColor(c, r, g, b, a) }

// FillRect fills the rectangle with the fill colour.
func (c Context) FillRect(r Rect) { cgContextFillRect(c, r) }

// Retain retains the font and returns it.
func (f Font) Retain() Font {
	mustLoad()
	if f == 0 {
		return 0
	}
	return cgFontRetain(f)
}

// Release releases the font. A zero font is ignored.
func (f Font) Release() {
	mustLoad()
	if f != 0 {
		cgFontRelease(f)
	}
}
