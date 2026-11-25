// Package render declares the contract between platform windows and GPU renderers.
package render

import gpui "github.com/megascan/gpui-go"

// Renderer draws scenes into a platform window's surface; it is the method set of
// upstream's WgpuRenderer, which the Metal and DirectX renderers also provide.
type Renderer interface {
	// Draw renders the scene and presents it, reporting whether a frame was presented.
	Draw(scene *gpui.Scene) bool
	// UpdateDrawableSize resizes the drawable surface.
	UpdateDrawableSize(size gpui.Size[gpui.DevicePixels])
	// UpdateTransparency switches between an opaque and a transparent surface.
	UpdateTransparency(transparent bool)
	// SetSubpixelLayout selects BGR or RGB subpixel order for subpixel text.
	SetSubpixelLayout(isBGR bool)
	// SpriteAtlas returns the atlas the renderer samples sprites from.
	SpriteAtlas() gpui.PlatformAtlas
	// SupportsDualSourceBlending reports whether subpixel text can be blended in one pass.
	SupportsDualSourceBlending() bool
	// GPUSpecs describes the GPU, when known.
	GPUSpecs() (gpui.GPUSpecs, bool)
	// DeviceLost reports whether the GPU device was lost and needs Recover.
	DeviceLost() bool
	// Recover recreates the GPU state for the window after a device loss.
	Recover(handle gpui.RawWindowHandle) error
	// Destroy releases the GPU resources; call it before the surface and display connection go away.
	Destroy()
}
