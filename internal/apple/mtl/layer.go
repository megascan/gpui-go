//go:build darwin

package mtl

import (
	"github.com/megascan/gpui-go/internal/apple/objc"
)

// Drawable wraps an id<CAMetalDrawable>.
type Drawable objc.ID

// ID returns the underlying object.
func (d Drawable) ID() objc.ID { return objc.ID(d) }

// Texture returns the drawable's texture.
func (d Drawable) Texture() Texture { return Texture(send[objc.ID](d.ID(), "texture")) }

// Present presents the drawable as soon as possible.
func (d Drawable) Present() { sendVoid(d.ID(), "present") }

// MetalLayer wraps a CAMetalLayer.
type MetalLayer objc.ID

// ID returns the underlying object.
func (l MetalLayer) ID() objc.ID { return objc.ID(l) }

// NewMetalLayer returns a +1 CAMetalLayer.
func NewMetalLayer() MetalLayer { return MetalLayer(newObject("CAMetalLayer")) }

// SetDevice sets the device that creates the layer's textures.
func (l MetalLayer) SetDevice(device Device) { sendVoid(l.ID(), "setDevice:", device.ID()) }

// Device returns the layer's device.
func (l MetalLayer) Device() Device { return Device(send[objc.ID](l.ID(), "device")) }

// SetPixelFormat sets the drawable pixel format.
func (l MetalLayer) SetPixelFormat(format PixelFormat) {
	sendVoid(l.ID(), "setPixelFormat:", format)
}

// PixelFormat returns the drawable pixel format.
func (l MetalLayer) PixelFormat() PixelFormat { return send[PixelFormat](l.ID(), "pixelFormat") }

// SetOpaque sets whether the layer is opaque.
func (l MetalLayer) SetOpaque(opaque bool) { sendVoid(l.ID(), "setOpaque:", opaque) }

// IsOpaque reports whether the layer is opaque.
func (l MetalLayer) IsOpaque() bool { return send[bool](l.ID(), "isOpaque") }

// SetFramebufferOnly sets whether drawables are only render targets.
func (l MetalLayer) SetFramebufferOnly(only bool) { sendVoid(l.ID(), "setFramebufferOnly:", only) }

// SetPresentsWithTransaction sets whether presentation is synchronised with Core Animation.
func (l MetalLayer) SetPresentsWithTransaction(v bool) {
	sendVoid(l.ID(), "setPresentsWithTransaction:", v)
}

// PresentsWithTransaction reports whether presentation is synchronised with Core Animation.
func (l MetalLayer) PresentsWithTransaction() bool {
	return send[bool](l.ID(), "presentsWithTransaction")
}

// SetMaximumDrawableCount sets the drawable pool size (2 or 3).
func (l MetalLayer) SetMaximumDrawableCount(count uint) {
	sendVoid(l.ID(), "setMaximumDrawableCount:", count)
}

// SetAllowsNextDrawableTimeout sets whether nextDrawable may time out.
func (l MetalLayer) SetAllowsNextDrawableTimeout(v bool) {
	sendVoid(l.ID(), "setAllowsNextDrawableTimeout:", v)
}

// SetDisplaySyncEnabled sets whether presentation waits for vsync (macOS only).
func (l MetalLayer) SetDisplaySyncEnabled(v bool) { sendVoid(l.ID(), "setDisplaySyncEnabled:", v) }

// SetNeedsDisplayOnBoundsChange sets whether a bounds change marks the layer for display.
func (l MetalLayer) SetNeedsDisplayOnBoundsChange(v bool) {
	sendVoid(l.ID(), "setNeedsDisplayOnBoundsChange:", v)
}

// SetAutoresizingMask sets the CALayer autoresizing mask.
func (l MetalLayer) SetAutoresizingMask(mask uint32) {
	sendVoid(l.ID(), "setAutoresizingMask:", mask)
}

// SetColorspace sets the layer's CGColorSpaceRef.
func (l MetalLayer) SetColorspace(colorspace uintptr) {
	sendVoid(l.ID(), "setColorspace:", colorspace)
}

// SetContentsScale sets the layer's contents scale.
func (l MetalLayer) SetContentsScale(scale objc.CGFloat) {
	sendVoid(l.ID(), "setContentsScale:", scale)
}

// SetDrawableSize sets the drawable size in pixels.
func (l MetalLayer) SetDrawableSize(size objc.CGSize) {
	mustLoad()
	msgSendCGSize(l.ID(), objc.Sel("setDrawableSize:"), size)
}

// DrawableSize returns the drawable size in pixels.
func (l MetalLayer) DrawableSize() objc.CGSize { return send[objc.CGSize](l.ID(), "drawableSize") }

// NextDrawable returns the next autoreleased drawable, or false when none is available in time.
func (l MetalLayer) NextDrawable() (Drawable, bool) {
	d := Drawable(send[objc.ID](l.ID(), "nextDrawable"))
	return d, d != 0
}
