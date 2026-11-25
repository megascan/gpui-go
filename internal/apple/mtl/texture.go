//go:build darwin

package mtl

import (
	"unsafe"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

// TextureDescriptor wraps an MTLTextureDescriptor.
type TextureDescriptor objc.ID

// ID returns the underlying object.
func (d TextureDescriptor) ID() objc.ID { return objc.ID(d) }

// NewTextureDescriptor returns a +1 texture descriptor.
func NewTextureDescriptor() TextureDescriptor {
	return TextureDescriptor(newObject("MTLTextureDescriptor"))
}

// SetWidth sets the width in pixels.
func (d TextureDescriptor) SetWidth(width uint) { sendVoid(d.ID(), "setWidth:", width) }

// SetHeight sets the height in pixels.
func (d TextureDescriptor) SetHeight(height uint) { sendVoid(d.ID(), "setHeight:", height) }

// SetPixelFormat sets the pixel format.
func (d TextureDescriptor) SetPixelFormat(format PixelFormat) {
	sendVoid(d.ID(), "setPixelFormat:", format)
}

// SetUsage sets how the texture is used.
func (d TextureDescriptor) SetUsage(usage TextureUsage) { sendVoid(d.ID(), "setUsage:", usage) }

// SetStorageMode sets where the texture lives.
func (d TextureDescriptor) SetStorageMode(mode StorageMode) {
	sendVoid(d.ID(), "setStorageMode:", mode)
}

// SetTextureType sets the texture type.
func (d TextureDescriptor) SetTextureType(t TextureType) { sendVoid(d.ID(), "setTextureType:", t) }

// SetSampleCount sets the multisample count.
func (d TextureDescriptor) SetSampleCount(count uint) { sendVoid(d.ID(), "setSampleCount:", count) }

// Texture wraps an id<MTLTexture>.
type Texture objc.ID

// ID returns the underlying object.
func (t Texture) ID() objc.ID { return objc.ID(t) }

// Width returns the width in pixels.
func (t Texture) Width() uint { return send[uint](t.ID(), "width") }

// Height returns the height in pixels.
func (t Texture) Height() uint { return send[uint](t.ID(), "height") }

// PixelFormat returns the pixel format.
func (t Texture) PixelFormat() PixelFormat { return send[PixelFormat](t.ID(), "pixelFormat") }

// SetLabel sets the debug label.
func (t Texture) SetLabel(label string) { setLabel(t.ID(), label) }

// ReplaceRegion copies bytes into region of mipmap level; bytes must cover every row.
func (t Texture) ReplaceRegion(region Region, level uint, bytes unsafe.Pointer, bytesPerRow uint) {
	mustLoad()
	msgSendReplaceRgn(t.ID(), objc.Sel("replaceRegion:mipmapLevel:withBytes:bytesPerRow:"), region, level, bytes, bytesPerRow)
}

// GetBytes copies region of mipmap level into bytes.
func (t Texture) GetBytes(bytes unsafe.Pointer, bytesPerRow uint, region Region, level uint) {
	mustLoad()
	msgSendGetBytes(t.ID(), objc.Sel("getBytes:bytesPerRow:fromRegion:mipmapLevel:"), bytes, bytesPerRow, region, level)
}
