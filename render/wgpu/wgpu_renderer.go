//go:build linux || js

package wgpu

import (
	_ "embed"
	"fmt"
	"log/slog"
	"slices"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

var (
	//go:embed shaders/shaders.wgsl
	shadersWGSL string
	//go:embed shaders/shaders_storage.wgsl
	shadersStorageWGSL string
	//go:embed shaders/shaders_webgl.wgsl
	shadersWebglWGSL string
	//go:embed shaders/shaders_subpixel.wgsl
	shadersSubpixelWGSL string
)

var (
	storageBufferShaders         = shadersWGSL + shadersStorageWGSL
	webglShaders                 = shadersWGSL + shadersWebglWGSL
	subpixelStorageBufferShaders = "enable dual_source_blending;\n" + shadersWGSL + shadersStorageWGSL + shadersSubpixelWGSL
	subpixelWebglShaders         = "enable dual_source_blending;\n" + shadersWGSL + shadersWebglWGSL + shadersSubpixelWGSL
)

// WgpuSurfaceConfig configures a renderer's surface.
type WgpuSurfaceConfig struct {
	// Size is the drawable size.
	Size gpui.Size[gpui.DevicePixels]
	// Transparent selects a transparent surface.
	Transparent bool
	// PreferredPresentMode is used when the surface supports it; otherwise, and when None,
	// the renderer uses Fifo (VSync).
	PreferredPresentMode gpui.Option[PresentMode]
}

func wgpuRendererSurfaceConfiguration(
	surface gpu.Surface,
	adapter gpu.Adapter,
	device gpu.Device,
	config *WgpuSurfaceConfig,
) (gpu.SurfaceConfiguration, gpu.CompositeAlphaMode, gpu.CompositeAlphaMode, error) {
	surfaceCaps := surface.GetCapabilities(adapter)
	preferredFormats := []gpu.TextureFormat{
		gpu.TextureFormatBgra8Unorm,
		gpu.TextureFormatRgba8Unorm,
	}
	var surfaceFormat gpu.TextureFormat
	found := false
	for _, f := range preferredFormats {
		if slices.Contains(surfaceCaps.Formats, f) {
			surfaceFormat, found = f, true
			break
		}
	}
	if !found {
		for _, f := range surfaceCaps.Formats {
			if !f.HasSrgbSuffix() {
				surfaceFormat, found = f, true
				break
			}
		}
	}
	if !found && len(surfaceCaps.Formats) > 0 {
		surfaceFormat, found = surfaceCaps.Formats[0], true
	}
	if !found {
		return gpu.SurfaceConfiguration{}, 0, 0, fmt.Errorf("Surface reports no supported texture formats for adapter %q", adapter.GetInfo().Name)
	}

	pickAlphaMode := func(preferences []gpu.CompositeAlphaMode) (gpu.CompositeAlphaMode, error) {
		for _, p := range preferences {
			if slices.Contains(surfaceCaps.AlphaModes, p) {
				return p, nil
			}
		}
		if len(surfaceCaps.AlphaModes) > 0 {
			return surfaceCaps.AlphaModes[0], nil
		}
		return 0, fmt.Errorf("Surface reports no supported alpha modes for adapter %q", adapter.GetInfo().Name)
	}

	transparentAlphaMode, err := pickAlphaMode([]gpu.CompositeAlphaMode{
		gpu.CompositeAlphaModePreMultiplied,
		gpu.CompositeAlphaModeInherit,
	})
	if err != nil {
		return gpu.SurfaceConfiguration{}, 0, 0, err
	}
	opaqueAlphaMode, err := pickAlphaMode([]gpu.CompositeAlphaMode{
		gpu.CompositeAlphaModeOpaque,
		gpu.CompositeAlphaModeInherit,
	})
	if err != nil {
		return gpu.SurfaceConfiguration{}, 0, 0, err
	}

	alphaMode := opaqueAlphaMode
	if config.Transparent {
		alphaMode = transparentAlphaMode
	}

	maxTextureSize := device.Limits().MaxTextureDimension2D

	requestedWidth := uint32(config.Size.Width)
	requestedHeight := uint32(config.Size.Height)
	clampedWidth := min(requestedWidth, maxTextureSize)
	clampedHeight := min(requestedHeight, maxTextureSize)

	if clampedWidth != requestedWidth || clampedHeight != requestedHeight {
		slog.Warn(fmt.Sprintf(
			"Requested surface size (%d, %d) exceeds maximum texture dimension %d. Clamping to (%d, %d). Window content may not fill the entire window.",
			requestedWidth, requestedHeight, maxTextureSize, clampedWidth, clampedHeight,
		))
	}

	presentMode := gpu.PresentModeFifo
	if mode, ok := config.PreferredPresentMode.Get(); ok && slices.Contains(surfaceCaps.PresentModes, mode) {
		presentMode = mode
	}

	surfaceConfig := gpu.SurfaceConfiguration{
		Usage:                      gpu.TextureUsagesRenderAttachment,
		Format:                     surfaceFormat,
		Width:                      max(clampedWidth, 1),
		Height:                     max(clampedHeight, 1),
		PresentMode:                presentMode,
		DesiredMaximumFrameLatency: 2,
		AlphaMode:                  alphaMode,
	}

	return surfaceConfig, transparentAlphaMode, opaqueAlphaMode, nil
}
