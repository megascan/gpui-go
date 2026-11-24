//go:build linux || js

package wgpu

import (
	_ "embed"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"math/bits"
	"os"
	"runtime"
	"slices"
	"strconv"
	"time"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/render"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

const maxInstanceBufferSize uint64 = 256 * 1024 * 1024

const instanceTextureTexelSize uint64 = 16

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

var (
	_ render.Renderer               = (*WgpuRenderer)(nil)
	_ gpui.PlatformHeadlessRenderer = (*WgpuHeadlessRenderer)(nil)
)

const isWasm = runtime.GOOS == "js"

func leastCommonMultiple(left, right uint64) uint64 {
	first := left
	second := right
	for second != 0 {
		remainder := first % second
		first = second
		second = remainder
	}
	return left / first * right
}

func nextMultipleOf(value, multiple uint64) uint64 {
	if remainder := value % multiple; remainder != 0 {
		return value + (multiple - remainder)
	}
	return value
}

func nextPowerOfTwo(value uint64) uint64 {
	if value <= 1 {
		return 1
	}
	return 1 << bits.Len64(value-1)
}

type globalParams struct {
	ViewportSize       [2]float32
	PathPaintBase      uint32
	PathCurveBase      uint32
	DepthScale         float32
	QuadCount          uint32
	PremultipliedAlpha uint32
	Pad                uint32
}

const depthFormat = gpu.TextureFormatDepth32Float

type depthTexture struct {
	texture gpu.Texture
	view    gpu.TextureView
}

func (d *depthTexture) release() {
	d.view.Release()
	d.texture.Release()
}

type podBounds struct {
	Origin [2]float32
	Size   [2]float32
}

func podBoundsFrom(bounds gpui.Bounds[gpui.ScaledPixels]) podBounds {
	return podBounds{
		Origin: [2]float32{float32(bounds.Origin.X), float32(bounds.Origin.Y)},
		Size:   [2]float32{float32(bounds.Size.Width), float32(bounds.Size.Height)},
	}
}

type surfaceParams struct {
	Bounds      podBounds
	ContentMask podBounds
}

type gammaParams struct {
	GammaRatios               [4]float32
	GrayscaleEnhancedContrast float32
	SubpixelEnhancedContrast  float32
	IsBGR                     uint32
	pad                       uint32
}

func bytesOf[T any](value *T) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(value)), unsafe.Sizeof(*value))
}

type wgpuPipelines struct {
	quads              gpu.RenderPipeline
	opaqueQuads        gpu.RenderPipeline
	shadows            gpu.RenderPipeline
	pathTiles          gpu.RenderPipeline
	underlines         gpu.RenderPipeline
	monoSprites        gpu.RenderPipeline
	subpixelSprites    gpu.RenderPipeline
	polySprites        gpu.RenderPipeline
	surfaces           gpu.RenderPipeline
	premultipliedAlpha bool
}

func (p *wgpuPipelines) release() {
	for _, pipeline := range []gpu.RenderPipeline{
		p.quads, p.opaqueQuads, p.shadows, p.pathTiles, p.underlines, p.monoSprites, p.subpixelSprites, p.polySprites, p.surfaces,
	} {
		if pipeline != nil {
			pipeline.Release()
		}
	}
}

type instanceBinding struct {
	bindGroup     gpu.BindGroup
	firstInstance uint32
}

type pathTileBinding struct {
	tiles     instanceBinding
	paintBase uint32
	curveBase uint32
}

type instanceAllocation struct {
	offset        uint64
	size          uint64
	firstInstance uint32
}

type instanceBindings struct {
	quads             instanceBinding
	shadows           instanceBinding
	pathTiles         pathTileBinding
	underlines        instanceBinding
	monochromeSprites instanceBinding
	subpixelSprites   instanceBinding
	polychromeSprites instanceBinding
}

func (b *instanceBindings) release() {
	for _, group := range []gpu.BindGroup{
		b.quads.bindGroup, b.shadows.bindGroup, b.pathTiles.tiles.bindGroup, b.underlines.bindGroup,
		b.monochromeSprites.bindGroup, b.subpixelSprites.bindGroup, b.polychromeSprites.bindGroup,
	} {
		if group != nil {
			group.Release()
		}
	}
}

type wgpuBindGroupLayouts struct {
	globals   gpu.BindGroupLayout
	instances gpu.BindGroupLayout
	pathTiles gpu.BindGroupLayout
	texture   gpu.BindGroupLayout
	surfaces  gpu.BindGroupLayout
}

func (l *wgpuBindGroupLayouts) release() {
	l.globals.Release()
	l.instances.Release()
	l.pathTiles.Release()
	l.texture.Release()
	l.surfaces.Release()
}

// GPUContext is the GPU context shared by the windows of one platform, used to coordinate
// device recovery across them. A nil Context is upstream's None: the first window to create a
// renderer initializes it and later windows share it.
type GPUContext struct {
	Context *WgpuContext
}

type instanceData struct {
	storage gpu.Buffer
	texture gpu.Texture
	view    gpu.TextureView
	width   uint32
	height  uint32
}

func (d *instanceData) release() {
	if d.storage != nil {
		d.storage.Release()
		return
	}
	d.view.Release()
	d.texture.Release()
}

type wgpuResources struct {
	device                 gpu.Device
	queue                  gpu.Queue
	pipelines              wgpuPipelines
	bindGroupLayouts       wgpuBindGroupLayouts
	atlasSampler           gpu.Sampler
	atlasTextureBindGroups map[gpui.AtlasTextureID]cachedTextureBindGroup
	globalsBuffer          gpu.Buffer
	globalsBindGroup       gpu.BindGroup
	instanceData           instanceData
}

func (r *wgpuResources) release() {
	r.pipelines.release()
	for _, cached := range r.atlasTextureBindGroups {
		cached.bindGroup.Release()
	}
	clear(r.atlasTextureBindGroups)
	r.globalsBindGroup.Release()
	r.globalsBuffer.Release()
	r.instanceData.release()
	r.atlasSampler.Release()
	r.bindGroupLayouts.release()
}

type cachedTextureBindGroup struct {
	textureGeneration uint64
	bindGroup         gpu.BindGroup
}

type wgpuRendererCore struct {
	resources               wgpuResources
	atlas                   *WgpuAtlas
	gammaOffset             uint64
	instanceDataCapacity    uint64
	maxInstanceDataSize     uint64
	instanceDataAlignment   uint64
	usesTextureInstanceData bool
	renderingParams         renderingParameters
	isBGR                   bool
	dualSourceBlending      bool
	transientAttachments    bool
	adapterInfo             gpu.AdapterInfo
	depthTexture            *depthTexture
	targetFormat            gpu.TextureFormat
	maxTextureSize          uint32
}

func (c *wgpuRendererCore) release() {
	if c.depthTexture != nil {
		c.depthTexture.release()
		c.depthTexture = nil
	}
	c.resources.release()
}

type rendererStateKind uint8

const (
	rendererStateReady rendererStateKind = iota
	rendererStateUnconfigured
	rendererStateReleased
)

type rendererState struct {
	kind    rendererStateKind
	surface gpu.Surface
	core    *wgpuRendererCore
}

// WgpuRenderer draws scenes into a window surface with wgpu.
type WgpuRenderer struct {
	context                 *GPUContext
	compositorGPU           gpuiutil.Option[CompositorGPUHint]
	state                   rendererState
	surfaceConfig           gpu.SurfaceConfiguration
	atlas                   *WgpuAtlas
	transparentAlphaMode    gpu.CompositeAlphaMode
	opaqueAlphaMode         gpu.CompositeAlphaMode
	maxTextureSize          uint32
	isBGR                   bool
	failedFrameCount        uint32
	deviceErrors            *DeviceErrorState
	observedErrorGeneration uint64
	lastSurfaceError        gpuiutil.Option[string]
	needsRedraw             bool
}

func (r *WgpuRenderer) core() *wgpuRendererCore {
	if r.state.kind == rendererStateReleased {
		return nil
	}
	return r.state.core
}

func (r *WgpuRenderer) setState(state rendererState) {
	old := r.state
	r.state = state
	if old.surface != nil && old.surface != state.surface {
		old.surface.Release()
	}
	if old.core != nil && old.core != state.core {
		old.core.release()
	}
}

// NewRenderer creates a window renderer; it is NewWgpuRenderer returned as the render.Renderer
// platform windows hold.
func NewRenderer(gpuContext *GPUContext, window gpui.RawWindowHandle, config WgpuSurfaceConfig, compositorGPU gpuiutil.Option[CompositorGPUHint]) (render.Renderer, error) {
	renderer, err := NewWgpuRenderer(gpuContext, window, config, compositorGPU)
	if err != nil {
		return nil, err
	}
	return renderer, nil
}

// NewWgpuRenderer creates a renderer for the window. The gpuContext is shared by every window
// of the platform: the first renderer initializes it and later ones share it. The window must
// stay valid until the renderer is destroyed.
func NewWgpuRenderer(gpuContext *GPUContext, window gpui.RawWindowHandle, config WgpuSurfaceConfig, compositorGPU gpuiutil.Option[CompositorGPUHint]) (*WgpuRenderer, error) {
	var instance gpu.Instance
	if gpuContext.Context != nil {
		instance = gpuContext.Context.Instance
	} else {
		instance = WgpuContextInstance(gpuiutil.Some(window))
	}

	surface, err := createSurface(instance, window)
	if err != nil {
		return nil, fmt.Errorf("Failed to create surface: %w", err)
	}

	surfaceConfigured := gpuContext.Context == nil
	context := gpuContext.Context
	if context != nil {
		if err := context.CheckCompatibleWithSurface(surface); err != nil {
			surface.Release()
			return nil, err
		}
	} else {
		created, err := NewWgpuContext(instance, surface, &config, compositorGPU)
		if err != nil {
			surface.Release()
			return nil, err
		}
		gpuContext.Context = created
		context = created
	}

	atlas := WgpuAtlasFromContext(context)

	return newWgpuRendererInternal(gpuContext, context, surface, config, surfaceConfigured, compositorGPU, atlas)
}

// NewWgpuRendererFromSurface creates a renderer drawing into an existing surface of context's
// instance.
func NewWgpuRendererFromSurface(context *WgpuContext, surface gpu.Surface, config WgpuSurfaceConfig) (*WgpuRenderer, error) {
	atlas := WgpuAtlasFromContext(context)
	return newWgpuRendererInternal(nil, context, surface, config, false, gpuiutil.None[CompositorGPUHint](), atlas)
}

func newWgpuRendererInternal(
	gpuContext *GPUContext,
	context *WgpuContext,
	surface gpu.Surface,
	config WgpuSurfaceConfig,
	surfaceConfigured bool,
	compositorGPU gpuiutil.Option[CompositorGPUHint],
	atlas *WgpuAtlas,
) (*WgpuRenderer, error) {
	surfaceConfig, transparentAlphaMode, opaqueAlphaMode, err := wgpuRendererSurfaceConfiguration(surface, context.Adapter, context.Device, &config)
	if err != nil {
		surface.Release()
		return nil, err
	}
	if !surfaceConfigured {
		surface.Configure(context.Device, &surfaceConfig)
	}

	surfaceFormat := surfaceConfig.Format
	alphaMode := surfaceConfig.AlphaMode
	maxTextureSize := context.Device.Limits().MaxTextureDimension2D
	core := newWgpuRendererCore(context, atlas, surfaceFormat, alphaMode)

	return &WgpuRenderer{
		context:                 gpuContext,
		compositorGPU:           compositorGPU,
		state:                   rendererState{kind: rendererStateReady, surface: surface, core: core},
		surfaceConfig:           surfaceConfig,
		atlas:                   atlas,
		transparentAlphaMode:    transparentAlphaMode,
		opaqueAlphaMode:         opaqueAlphaMode,
		maxTextureSize:          maxTextureSize,
		isBGR:                   false,
		failedFrameCount:        0,
		deviceErrors:            context.Errors(),
		observedErrorGeneration: 0,
		lastSurfaceError:        gpuiutil.None[string](),
		needsRedraw:             false,
	}, nil
}

func wgpuRendererCoreCreateBindGroupLayouts(device gpu.Device, usesTextureInstanceData bool) wgpuBindGroupLayouts {
	globals := device.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{
		Label: "globals_layout",
		Entries: []gpu.BindGroupLayoutEntry{
			{
				Binding:    0,
				Visibility: gpu.ShaderStagesVertexFragment,
				Type: gpu.BindingType{Buffer: &gpu.BufferBindingLayout{
					Type:           gpu.BufferBindingTypeUniform,
					MinBindingSize: uint64(unsafe.Sizeof(globalParams{})),
				}},
			},
			{
				Binding:    1,
				Visibility: gpu.ShaderStagesFragment,
				Type: gpu.BindingType{Buffer: &gpu.BufferBindingLayout{
					Type:           gpu.BufferBindingTypeUniform,
					MinBindingSize: uint64(unsafe.Sizeof(gammaParams{})),
				}},
			},
		},
	})

	instanceDataEntry := func(binding uint32) gpu.BindGroupLayoutEntry {
		entry := gpu.BindGroupLayoutEntry{Binding: binding, Visibility: gpu.ShaderStagesVertexFragment}
		if usesTextureInstanceData {
			entry.Type = gpu.BindingType{Texture: &gpu.TextureBindingLayout{
				SampleType:    gpu.TextureSampleTypeUint,
				ViewDimension: gpu.TextureViewDimensionD2,
			}}
		} else {
			entry.Type = gpu.BindingType{Buffer: &gpu.BufferBindingLayout{
				Type: gpu.BufferBindingTypeReadOnlyStorage,
			}}
		}
		return entry
	}

	instances := device.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{
		Label:   "instances_layout",
		Entries: []gpu.BindGroupLayoutEntry{instanceDataEntry(0)},
	})

	var pathTiles gpu.BindGroupLayout
	if usesTextureInstanceData {
		pathTiles = device.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{
			Label:   "path_tiles_layout",
			Entries: []gpu.BindGroupLayoutEntry{instanceDataEntry(0)},
		})
	} else {
		pathTiles = device.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{
			Label:   "path_tiles_layout",
			Entries: []gpu.BindGroupLayoutEntry{instanceDataEntry(0), instanceDataEntry(1), instanceDataEntry(2)},
		})
	}

	filterableTexture := func(binding uint32, visibility gpu.ShaderStages) gpu.BindGroupLayoutEntry {
		return gpu.BindGroupLayoutEntry{
			Binding:    binding,
			Visibility: visibility,
			Type: gpu.BindingType{Texture: &gpu.TextureBindingLayout{
				SampleType:    gpu.TextureSampleTypeFloat,
				ViewDimension: gpu.TextureViewDimensionD2,
			}},
		}
	}
	filteringSampler := func(binding uint32, visibility gpu.ShaderStages) gpu.BindGroupLayoutEntry {
		return gpu.BindGroupLayoutEntry{
			Binding:    binding,
			Visibility: visibility,
			Type:       gpu.BindingType{Sampler: &gpu.SamplerBindingLayout{Type: gpu.SamplerBindingTypeFiltering}},
		}
	}

	texture := device.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{
		Label: "texture_layout",
		Entries: []gpu.BindGroupLayoutEntry{
			filterableTexture(0, gpu.ShaderStagesVertexFragment),
			filteringSampler(1, gpu.ShaderStagesVertexFragment),
		},
	})

	surfaces := device.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{
		Label: "surfaces_layout",
		Entries: []gpu.BindGroupLayoutEntry{
			{
				Binding:    0,
				Visibility: gpu.ShaderStagesVertexFragment,
				Type: gpu.BindingType{Buffer: &gpu.BufferBindingLayout{
					Type:           gpu.BufferBindingTypeUniform,
					MinBindingSize: uint64(unsafe.Sizeof(surfaceParams{})),
				}},
			},
			filterableTexture(1, gpu.ShaderStagesFragment),
			filterableTexture(2, gpu.ShaderStagesFragment),
			filteringSampler(3, gpu.ShaderStagesFragment),
		},
	})

	return wgpuBindGroupLayouts{
		globals:   globals,
		instances: instances,
		pathTiles: pathTiles,
		texture:   texture,
		surfaces:  surfaces,
	}
}

func wgpuRendererCoreCreateInstanceTexture(device gpu.Device, requestedCapacity uint64, maxTextureDimension uint32) (instanceData, uint64) {
	texelCount := (requestedCapacity + instanceTextureTexelSize - 1) / instanceTextureTexelSize
	width := uint32(max(min(texelCount, uint64(maxTextureDimension)), 1))
	height := uint32(max(min((texelCount+uint64(width)-1)/uint64(width), uint64(maxTextureDimension)), 1))
	texture := device.CreateTexture(&gpu.TextureDescriptor{
		Label:         "instance_texture",
		Size:          gpu.Extent3d{Width: width, Height: height, DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     gpu.TextureDimensionD2,
		Format:        gpu.TextureFormatRgba32Uint,
		Usage:         gpu.TextureUsagesTextureBinding | gpu.TextureUsagesCopyDst,
	})
	view := texture.CreateView(nil)
	capacity := uint64(width) * uint64(height) * instanceTextureTexelSize
	return instanceData{texture: texture, view: view, width: width, height: height}, capacity
}

func wgpuRendererCoreCreatePipelines(
	device gpu.Device,
	layouts *wgpuBindGroupLayouts,
	surfaceFormat gpu.TextureFormat,
	alphaMode gpu.CompositeAlphaMode,
	dualSourceBlending bool,
	usesTextureInstanceData bool,
) wgpuPipelines {
	deviceHasFeature := device.Features().Contains(gpu.FeaturesDualSourceBlending)
	if dualSourceBlending && !deviceHasFeature {
		slog.Error(fmt.Sprintf(
			"BUG: dual_source_blending flag is true but device does not have DUAL_SOURCE_BLENDING enabled (device features: %#x). Falling back to mono text rendering. Please report this at https://github.com/zed-industries/zed/issues",
			uint64(device.Features()),
		))
	}
	dualSourceBlending = dualSourceBlending && deviceHasFeature

	shaderSource, subpixelShaderSource := storageBufferShaders, subpixelStorageBufferShaders
	if usesTextureInstanceData {
		shaderSource, subpixelShaderSource = webglShaders, subpixelWebglShaders
	}
	shaderModule := device.CreateShaderModule(&gpu.ShaderModuleDescriptor{Label: "gpui_shaders", WGSL: shaderSource})
	defer shaderModule.Release()

	var subpixelShaderModule gpu.ShaderModule
	if dualSourceBlending {
		subpixelShaderModule = device.CreateShaderModule(&gpu.ShaderModuleDescriptor{Label: "gpui_subpixel_shaders", WGSL: subpixelShaderSource})
		defer subpixelShaderModule.Release()
	}

	premultipliedAlpha := alphaMode == gpu.CompositeAlphaModePreMultiplied
	blendMode := gpu.BlendStateAlphaBlending
	if premultipliedAlpha {
		blendMode = gpu.BlendStatePremultipliedAlphaBlending
	}

	colorTarget := gpu.ColorTargetState{
		Format:    surfaceFormat,
		Blend:     &blendMode,
		WriteMask: gpu.ColorWritesAll,
	}

	createPipeline := func(
		name, vsEntry, fsEntry string,
		globalsLayout, dataLayout, textureLayout gpu.BindGroupLayout,
		topology gpu.PrimitiveTopology,
		colorTargets []gpu.ColorTargetState,
		depthWriteEnabled bool,
		module gpu.ShaderModule,
	) gpu.RenderPipeline {
		bindGroupLayouts := []gpu.BindGroupLayout{globalsLayout, dataLayout}
		if textureLayout != nil {
			bindGroupLayouts = append(bindGroupLayouts, textureLayout)
		}
		pipelineLayout := device.CreatePipelineLayout(&gpu.PipelineLayoutDescriptor{
			Label:            name + "_layout",
			BindGroupLayouts: bindGroupLayouts,
		})
		defer pipelineLayout.Release()

		return device.CreateRenderPipeline(&gpu.RenderPipelineDescriptor{
			Label:  name,
			Layout: pipelineLayout,
			Vertex: gpu.VertexState{Module: module, EntryPoint: vsEntry},
			Fragment: &gpu.FragmentState{
				Module:     module,
				EntryPoint: fsEntry,
				Targets:    colorTargets,
			},
			Primitive: gpu.PrimitiveState{
				Topology:  topology,
				FrontFace: gpu.FrontFaceCcw,
				CullMode:  gpu.CullModeNone,
			},
			DepthStencil: &gpu.DepthStencilState{
				Format:            depthFormat,
				DepthWriteEnabled: depthWriteEnabled,
				DepthCompare:      gpu.CompareFunctionGreater,
			},
			Multisample: gpu.MultisampleState{},
		})
	}

	quads := createPipeline(
		"quads", "vs_quad", "fs_quad",
		layouts.globals, layouts.instances, nil,
		gpu.PrimitiveTopologyTriangleStrip,
		[]gpu.ColorTargetState{colorTarget},
		false,
		shaderModule,
	)

	opaqueQuads := createPipeline(
		"opaque_quads", "vs_opaque_quad", "fs_opaque_quad",
		layouts.globals, layouts.instances, nil,
		gpu.PrimitiveTopologyTriangleStrip,
		[]gpu.ColorTargetState{{Format: surfaceFormat, Blend: nil, WriteMask: gpu.ColorWritesAll}},
		true,
		shaderModule,
	)

	shadows := createPipeline(
		"shadows", "vs_shadow", "fs_shadow",
		layouts.globals, layouts.instances, nil,
		gpu.PrimitiveTopologyTriangleStrip,
		[]gpu.ColorTargetState{colorTarget},
		false,
		shaderModule,
	)

	pathTiles := createPipeline(
		"path_tiles", "vs_path_tile", "fs_path_tile",
		layouts.globals, layouts.pathTiles, nil,
		gpu.PrimitiveTopologyTriangleStrip,
		[]gpu.ColorTargetState{colorTarget},
		false,
		shaderModule,
	)

	underlines := createPipeline(
		"underlines", "vs_underline", "fs_underline",
		layouts.globals, layouts.instances, nil,
		gpu.PrimitiveTopologyTriangleStrip,
		[]gpu.ColorTargetState{colorTarget},
		false,
		shaderModule,
	)

	monoSprites := createPipeline(
		"mono_sprites", "vs_mono_sprite", "fs_mono_sprite",
		layouts.globals, layouts.instances, layouts.texture,
		gpu.PrimitiveTopologyTriangleStrip,
		[]gpu.ColorTargetState{colorTarget},
		false,
		shaderModule,
	)

	var subpixelSprites gpu.RenderPipeline
	if subpixelShaderModule != nil {
		subpixelBlend := gpu.BlendState{
			Color: gpu.BlendComponent{
				SrcFactor: gpu.BlendFactorSrc1,
				DstFactor: gpu.BlendFactorOneMinusSrc1,
				Operation: gpu.BlendOperationAdd,
			},
			Alpha: gpu.BlendComponent{
				SrcFactor: gpu.BlendFactorOne,
				DstFactor: gpu.BlendFactorOneMinusSrcAlpha,
				Operation: gpu.BlendOperationAdd,
			},
		}

		subpixelSprites = createPipeline(
			"subpixel_sprites", "vs_mono_sprite", "fs_subpixel_sprite",
			layouts.globals, layouts.instances, layouts.texture,
			gpu.PrimitiveTopologyTriangleStrip,
			[]gpu.ColorTargetState{{Format: surfaceFormat, Blend: &subpixelBlend, WriteMask: gpu.ColorWritesColor}},
			false,
			subpixelShaderModule,
		)
	}

	polySprites := createPipeline(
		"poly_sprites", "vs_poly_sprite", "fs_poly_sprite",
		layouts.globals, layouts.instances, layouts.texture,
		gpu.PrimitiveTopologyTriangleStrip,
		[]gpu.ColorTargetState{colorTarget},
		false,
		shaderModule,
	)

	surfaces := createPipeline(
		"surfaces", "vs_surface", "fs_surface",
		layouts.globals, layouts.surfaces, nil,
		gpu.PrimitiveTopologyTriangleStrip,
		[]gpu.ColorTargetState{colorTarget},
		false,
		shaderModule,
	)

	return wgpuPipelines{
		quads:              quads,
		opaqueQuads:        opaqueQuads,
		shadows:            shadows,
		pathTiles:          pathTiles,
		underlines:         underlines,
		monoSprites:        monoSprites,
		subpixelSprites:    subpixelSprites,
		polySprites:        polySprites,
		surfaces:           surfaces,
		premultipliedAlpha: premultipliedAlpha,
	}
}

// UpdateDrawableSize records the new drawable size and reconfigures the surface when one is
// present. The size is kept even without GPU resources so that recovery restores it.
func (r *WgpuRenderer) UpdateDrawableSize(size gpui.Size[gpui.DevicePixels]) {
	width := uint32(size.Width)
	height := uint32(size.Height)

	if width == r.surfaceConfig.Width && height == r.surfaceConfig.Height {
		return
	}

	clampedWidth := min(width, r.maxTextureSize)
	clampedHeight := min(height, r.maxTextureSize)
	if clampedWidth != width || clampedHeight != height {
		slog.Warn(fmt.Sprintf(
			"Requested surface size (%d, %d) exceeds maximum texture dimension %d. Clamping to (%d, %d). Window content may not fill the entire window.",
			width, height, r.maxTextureSize, clampedWidth, clampedHeight,
		))
	}
	r.surfaceConfig.Width = max(clampedWidth, 1)
	r.surfaceConfig.Height = max(clampedHeight, 1)

	if r.state.kind == rendererStateReady {
		r.state.surface.Configure(r.state.core.resources.device, &r.surfaceConfig)
	}
}

// SetSubpixelLayout selects BGR or RGB subpixel order for subpixel text.
func (r *WgpuRenderer) SetSubpixelLayout(isBGR bool) {
	r.isBGR = isBGR
	if core := r.core(); core != nil {
		core.isBGR = isBGR
	}
}

// UpdateTransparency switches between the surface's opaque and transparent alpha modes.
func (r *WgpuRenderer) UpdateTransparency(transparent bool) {
	newAlphaMode := r.opaqueAlphaMode
	if transparent {
		newAlphaMode = r.transparentAlphaMode
	}
	if newAlphaMode == r.surfaceConfig.AlphaMode {
		return
	}
	r.surfaceConfig.AlphaMode = newAlphaMode
	format := r.surfaceConfig.Format

	core := r.core()
	if core == nil {
		return
	}
	core.rebuildPipelines(format, newAlphaMode)

	if r.state.kind == rendererStateReady {
		r.state.surface.Configure(core.resources.device, &r.surfaceConfig)
	}
}

// ViewportSize is the configured surface size.
func (r *WgpuRenderer) ViewportSize() gpui.Size[gpui.DevicePixels] {
	return gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(int32(r.surfaceConfig.Width)),
		Height: gpui.DevicePixels(int32(r.surfaceConfig.Height)),
	}
}

// SpriteAtlas returns the atlas the renderer samples sprites from.
func (r *WgpuRenderer) SpriteAtlas() gpui.PlatformAtlas {
	return r.atlas
}

// WgpuAtlas returns the renderer's atlas as its concrete type.
func (r *WgpuRenderer) WgpuAtlas() *WgpuAtlas {
	return r.atlas
}

// SupportsDualSourceBlending reports whether subpixel text is drawn with dual-source blending.
func (r *WgpuRenderer) SupportsDualSourceBlending() bool {
	core := r.core()
	return core != nil && core.dualSourceBlending
}

// GPUSpecs describes the GPU; it reports false once GPU resources have been released by
// Destroy or a pending device recovery.
func (r *WgpuRenderer) GPUSpecs() (gpui.GPUSpecs, bool) {
	core := r.core()
	if core == nil {
		return gpui.GPUSpecs{}, false
	}
	adapterInfo := &core.adapterInfo
	return gpui.GPUSpecs{
		IsSoftwareEmulated: adapterInfo.DeviceType == gpu.DeviceTypeCpu,
		DeviceName:         adapterInfo.Name,
		DriverName:         adapterInfo.Driver,
		DriverInfo:         adapterInfo.DriverInfo,
	}, true
}

// MaxTextureSize is the device's maximum 2D texture dimension.
func (r *WgpuRenderer) MaxTextureSize() uint32 {
	return r.maxTextureSize
}

// Draw renders the scene into the surface and presents it, reporting whether a frame was
// presented.
func (r *WgpuRenderer) Draw(scene *gpui.Scene) bool {
	if isWasm && r.DeviceLost() {
		if r.state.kind == rendererStateReady {
			slog.Error("Browser graphics context was lost; rendering has stopped. Reload the page to recover.")
			r.UnconfigureSurface()
		}
		return false
	}

	if r.state.kind != rendererStateReady {
		return false
	}
	surface, core := r.state.surface, r.state.core

	errText, hasError := r.lastSurfaceError.Get()
	r.lastSurfaceError = gpuiutil.None[string]()
	if !hasError {
		errText, hasError = r.deviceErrors.ObserveError(&r.observedErrorGeneration)
	}
	if hasError {
		r.failedFrameCount++
		slog.Error(fmt.Sprintf("GPU error during frame (failure %d of 10): %s", r.failedFrameCount, errText))

		if r.failedFrameCount > 10 {
			panic(fmt.Sprintf("Too many consecutive GPU errors. Last error: %s", errText))
		} else if r.failedFrameCount > 5 {
			r.atlas.Clear()
			r.needsRedraw = true
			r.failedFrameCount = 0
			return false
		}
	} else {
		r.failedFrameCount = 0
	}

	frame := surface.GetCurrentTexture()
	switch frame.Status {
	case gpu.CurrentSurfaceTextureSuccess:
	case gpu.CurrentSurfaceTextureSuboptimal:
		frame.Texture.Release()
		surface.Configure(core.resources.device, &r.surfaceConfig)
		return false
	case gpu.CurrentSurfaceTextureLost, gpu.CurrentSurfaceTextureOutdated:
		surface.Configure(core.resources.device, &r.surfaceConfig)
		return false
	case gpu.CurrentSurfaceTextureTimeout, gpu.CurrentSurfaceTextureOccluded:
		return false
	case gpu.CurrentSurfaceTextureValidation:
		r.lastSurfaceError = gpuiutil.Some("Surface texture validation error")
		return false
	}
	defer frame.Texture.Release()

	size := gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(int32(frame.Texture.Width())),
		Height: gpui.DevicePixels(int32(frame.Texture.Height())),
	}
	frameView := frame.Texture.CreateView(nil)
	defer frameView.Release()
	if _, err := core.renderFrame(scene, frameView, size, gpu.ColorTransparent); err != nil {
		slog.Error(err.Error())
		return false
	}

	surface.Present()
	return true
}

func newWgpuRendererCore(context *WgpuContext, atlas *WgpuAtlas, targetFormat gpu.TextureFormat, alphaMode gpu.CompositeAlphaMode) *wgpuRendererCore {
	device := context.Device
	queue := context.Queue
	renderingParams := newRenderingParameters()
	usesTextureInstanceData := context.UsesTextureInstanceData()
	dualSourceBlending := context.SupportsDualSourceBlending()
	bindGroupLayouts := wgpuRendererCoreCreateBindGroupLayouts(device, usesTextureInstanceData)
	pipelines := wgpuRendererCoreCreatePipelines(
		device,
		&bindGroupLayouts,
		targetFormat,
		alphaMode,
		dualSourceBlending,
		usesTextureInstanceData,
	)
	atlasSampler := device.CreateSampler(&gpu.SamplerDescriptor{
		Label:     "atlas_sampler",
		MagFilter: gpu.FilterModeLinear,
		MinFilter: gpu.FilterModeLinear,
	})
	limits := device.Limits()
	uniformAlignment := uint64(limits.MinUniformBufferOffsetAlignment)
	globalsSize := uint64(unsafe.Sizeof(globalParams{}))
	gammaSize := uint64(unsafe.Sizeof(gammaParams{}))
	gammaOffset := nextMultipleOf(globalsSize, uniformAlignment)
	globalsBuffer := device.CreateBuffer(&gpu.BufferDescriptor{
		Label: "globals_buffer",
		Size:  gammaOffset + gammaSize,
		Usage: gpu.BufferUsagesUniform | gpu.BufferUsagesCopyDst,
	})
	var (
		data                  instanceData
		instanceDataCapacity  uint64
		maxInstanceDataSize   uint64
		instanceDataAlignment uint64
	)
	if usesTextureInstanceData {
		maxTextureDimension := limits.MaxTextureDimension2D
		maxInstanceDataSize = min(uint64(maxTextureDimension)*uint64(maxTextureDimension)*instanceTextureTexelSize, maxInstanceBufferSize)
		initialCapacity := min(uint64(2*1024*1024), maxInstanceDataSize)
		data, instanceDataCapacity = wgpuRendererCoreCreateInstanceTexture(device, initialCapacity, maxTextureDimension)
		instanceDataAlignment = instanceTextureTexelSize
	} else {
		maxBufferSize := min(limits.MaxBufferSize, limits.MaxStorageBufferBindingSize, maxInstanceBufferSize)
		initialCapacity := min(uint64(2*1024*1024), maxBufferSize)
		buffer := device.CreateBuffer(&gpu.BufferDescriptor{
			Label: "instance_buffer",
			Size:  initialCapacity,
			Usage: gpu.BufferUsagesStorage | gpu.BufferUsagesCopyDst,
		})
		data = instanceData{storage: buffer}
		instanceDataCapacity = initialCapacity
		maxInstanceDataSize = maxBufferSize
		instanceDataAlignment = uint64(limits.MinStorageBufferOffsetAlignment)
	}
	globalsBindGroup := device.CreateBindGroup(&gpu.BindGroupDescriptor{
		Label:  "globals_bind_group",
		Layout: bindGroupLayouts.globals,
		Entries: []gpu.BindGroupEntry{
			{Binding: 0, Resource: gpu.BindingResource{Buffer: globalsBuffer, Offset: 0, Size: globalsSize}},
			{Binding: 1, Resource: gpu.BindingResource{Buffer: globalsBuffer, Offset: gammaOffset, Size: gammaSize}},
		},
	})
	maxTextureSize := limits.MaxTextureDimension2D

	return &wgpuRendererCore{
		resources: wgpuResources{
			device:                 device,
			queue:                  queue,
			pipelines:              pipelines,
			bindGroupLayouts:       bindGroupLayouts,
			atlasSampler:           atlasSampler,
			atlasTextureBindGroups: map[gpui.AtlasTextureID]cachedTextureBindGroup{},
			globalsBuffer:          globalsBuffer,
			globalsBindGroup:       globalsBindGroup,
			instanceData:           data,
		},
		atlas:                   atlas,
		gammaOffset:             gammaOffset,
		instanceDataCapacity:    instanceDataCapacity,
		maxInstanceDataSize:     maxInstanceDataSize,
		instanceDataAlignment:   instanceDataAlignment,
		usesTextureInstanceData: usesTextureInstanceData,
		renderingParams:         renderingParams,
		isBGR:                   false,
		dualSourceBlending:      dualSourceBlending,
		transientAttachments:    context.SupportsTransientAttachments(),
		adapterInfo:             context.Adapter.GetInfo(),
		depthTexture:            nil,
		targetFormat:            targetFormat,
		maxTextureSize:          maxTextureSize,
	}
}

func (c *wgpuRendererCore) rebuildPipelines(surfaceFormat gpu.TextureFormat, alphaMode gpu.CompositeAlphaMode) {
	old := c.resources.pipelines
	c.resources.pipelines = wgpuRendererCoreCreatePipelines(
		c.resources.device,
		&c.resources.bindGroupLayouts,
		surfaceFormat,
		alphaMode,
		c.dualSourceBlending,
		c.usesTextureInstanceData,
	)
	old.release()
}

func (c *wgpuRendererCore) renderFrame(scene *gpui.Scene, targetView gpu.TextureView, size gpui.Size[gpui.DevicePixels], clearColor gpu.Color) (gpu.SubmissionIndex, error) {
	if !(size.Width > 0 && size.Height > 0) {
		return 0, fmt.Errorf("invalid render target size: %v", size)
	}
	if !(uint32(size.Width) <= c.maxTextureSize && uint32(size.Height) <= c.maxTextureSize) {
		return 0, fmt.Errorf("render target size %v exceeds maximum texture dimension %d", size, c.maxTextureSize)
	}

	c.atlas.BeforeFrame()

	gamma := gammaParams{
		GammaRatios:               c.renderingParams.gammaRatios,
		GrayscaleEnhancedContrast: c.renderingParams.grayscaleEnhancedContrast,
		SubpixelEnhancedContrast:  c.renderingParams.subpixelEnhancedContrast,
		IsBGR:                     boolToU32(c.isBGR),
	}
	c.resources.queue.WriteBuffer(c.resources.globalsBuffer, c.gammaOffset, bytesOf(&gamma))

	submission, err := c.recordFrame(scene, targetView, size, clearColor)
	if err != nil {
		c.resources.queue.Submit()
	}
	return submission, err
}

func boolToU32(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

func (c *wgpuRendererCore) recordFrame(scene *gpui.Scene, frameView gpu.TextureView, size gpui.Size[gpui.DevicePixels], clearColor gpu.Color) (gpu.SubmissionIndex, error) {
	var instanceOffset uint64
	bindings, err := c.writeInstances(scene, &instanceOffset)
	if err != nil {
		return 0, fmt.Errorf(
			"scene too large: %d paths, %d shadows, %d quads, %d underlines, %d monochrome sprites, %d subpixel sprites, %d polychrome sprites: %w",
			len(scene.Paths),
			len(scene.Shadows),
			len(scene.Quads),
			len(scene.Underlines),
			len(scene.MonochromeSprites),
			len(scene.SubpixelSprites),
			len(scene.PolychromeSprites),
			err,
		)
	}
	defer bindings.release()
	depthScale := gpui.QuadDepthScale(len(scene.Quads))
	globals := globalParams{
		ViewportSize:       [2]float32{float32(size.Width), float32(size.Height)},
		PathPaintBase:      bindings.pathTiles.paintBase,
		PathCurveBase:      bindings.pathTiles.curveBase,
		DepthScale:         depthScale,
		QuadCount:          uint32(len(scene.Quads)),
		PremultipliedAlpha: boolToU32(c.resources.pipelines.premultipliedAlpha),
	}
	c.resources.queue.WriteBuffer(c.resources.globalsBuffer, 0, bytesOf(&globals))
	c.prepareTextureBindGroups(scene)
	c.ensureDepthTexture(size)
	if c.depthTexture == nil {
		return 0, errors.New("depth texture missing")
	}
	depthView := c.depthTexture.view
	if gpuiutil.DebugAssertions && bindings.quads.firstInstance != 0 {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: 0", bindings.quads.firstInstance))
	}

	encoder := c.resources.device.CreateCommandEncoder(&gpu.CommandEncoderDescriptor{Label: "main_encoder"})
	defer encoder.Release()

	pass := encoder.BeginRenderPass(&gpu.RenderPassDescriptor{
		Label: "main_pass",
		ColorAttachments: []gpu.RenderPassColorAttachment{{
			View:       frameView,
			LoadOp:     gpu.LoadOpClear,
			StoreOp:    gpu.StoreOpStore,
			ClearValue: clearColor,
		}},
		DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{
			View:            depthView,
			DepthLoadOp:     gpu.LoadOpClear,
			DepthStoreOp:    gpu.StoreOpDiscard,
			DepthClearValue: 0,
		},
	})
	pass.SetBindGroup(0, c.resources.globalsBindGroup, nil)

	c.drawInstances(&bindings.quads, c.resources.pipelines.opaqueQuads, instanceRange(gpui.Range{Start: 0, End: len(scene.Quads)}), pass)

	viewportWidth := float32(size.Width)
	viewportHeight := float32(size.Height)
	var quadCursor uint32
	pathTileCursor := 0
	for batch := range scene.Batches() {
		minimumDepth, maximumDepth := float32(0), float32(1)
		if _, ok := batch.(gpui.PrimitiveBatchQuads); !ok {
			depth := gpui.QuadDepth(quadCursor, depthScale)
			minimumDepth, maximumDepth = depth, depth
		}
		pass.SetViewport(0, 0, viewportWidth, viewportHeight, minimumDepth, maximumDepth)

		switch batch := batch.(type) {
		case gpui.PrimitiveBatchQuads:
			quadCursor += uint32(batch.End - batch.Start)
			c.drawInstances(&bindings.quads, c.resources.pipelines.quads, instanceRange(batch.Range), pass)
		case gpui.PrimitiveBatchShadows:
			c.drawInstances(&bindings.shadows, c.resources.pipelines.shadows, instanceRange(batch.Range), pass)
		case gpui.PrimitiveBatchPaths:
			tilesStart := pathTileCursor
			for _, path := range scene.Paths[batch.Start:batch.End] {
				pathTileCursor += path.TileCount()
			}
			c.drawInstances(&bindings.pathTiles.tiles, c.resources.pipelines.pathTiles, instanceRange(gpui.Range{Start: tilesStart, End: pathTileCursor}), pass)
		case gpui.PrimitiveBatchUnderlines:
			c.drawInstances(&bindings.underlines, c.resources.pipelines.underlines, instanceRange(batch.Range), pass)
		case gpui.PrimitiveBatchMonochromeSprites:
			c.drawSprites(&bindings.monochromeSprites, batch.TextureID, c.resources.pipelines.monoSprites, instanceRange(batch.Range), pass)
		case gpui.PrimitiveBatchSubpixelSprites:
			pipeline := c.resources.pipelines.subpixelSprites
			if pipeline == nil {
				pipeline = c.resources.pipelines.monoSprites
			}
			c.drawSprites(&bindings.subpixelSprites, batch.TextureID, pipeline, instanceRange(batch.Range), pass)
		case gpui.PrimitiveBatchPolychromeSprites:
			c.drawSprites(&bindings.polychromeSprites, batch.TextureID, c.resources.pipelines.polySprites, instanceRange(batch.Range), pass)
		case gpui.PrimitiveBatchSurfaces:
		case gpui.PrimitiveBatchNativeViews:
		}
	}
	pass.End()

	commands := encoder.Finish()
	defer commands.Release()
	submission := c.resources.queue.Submit(commands)
	return submission, nil
}

func (c *wgpuRendererCore) ensureDepthTexture(size gpui.Size[gpui.DevicePixels]) {
	width := uint32(size.Width)
	height := uint32(size.Height)
	if c.depthTexture != nil && c.depthTexture.texture.Width() == width && c.depthTexture.texture.Height() == height {
		return
	}
	usage := gpu.TextureUsagesRenderAttachment
	if c.transientAttachments {
		usage |= gpu.TextureUsagesTransientAttachment
	}
	texture := c.resources.device.CreateTexture(&gpu.TextureDescriptor{
		Label:         "depth_texture",
		Size:          gpu.Extent3d{Width: width, Height: height, DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     gpu.TextureDimensionD2,
		Format:        depthFormat,
		Usage:         usage,
	})
	view := texture.CreateView(nil)
	if c.depthTexture != nil {
		c.depthTexture.release()
	}
	c.depthTexture = &depthTexture{texture: texture, view: view}
}

func (c *wgpuRendererCore) writeInstances(scene *gpui.Scene, instanceOffset *uint64) (bindings instanceBindings, err error) {
	defer func() {
		if err != nil {
			bindings.release()
			bindings = instanceBindings{}
		}
	}()
	if bindings.quads, err = c.writeInstanceBinding("quads_bind_group", instanceOffset, instanceBytes(scene.Quads)); err != nil {
		return
	}
	if bindings.shadows, err = c.writeInstanceBinding("shadows_bind_group", instanceOffset, instanceBytes(scene.Shadows)); err != nil {
		return
	}
	if bindings.pathTiles, err = c.writePathTiles(scene, instanceOffset); err != nil {
		return
	}
	if bindings.underlines, err = c.writeInstanceBinding("underlines_bind_group", instanceOffset, instanceBytes(scene.Underlines)); err != nil {
		return
	}
	if bindings.monochromeSprites, err = c.writeInstanceBinding("monochrome_sprites_bind_group", instanceOffset, instanceBytes(scene.MonochromeSprites)); err != nil {
		return
	}
	if bindings.subpixelSprites, err = c.writeInstanceBinding("subpixel_sprites_bind_group", instanceOffset, instanceBytes(scene.SubpixelSprites)); err != nil {
		return
	}
	bindings.polychromeSprites, err = c.writeInstanceBinding("polychrome_sprites_bind_group", instanceOffset, instanceBytes(scene.PolychromeSprites))
	return
}

func (c *wgpuRendererCore) createTextureBindGroup(label string, textureView gpu.TextureView) gpu.BindGroup {
	return c.resources.device.CreateBindGroup(&gpu.BindGroupDescriptor{
		Label:  label,
		Layout: c.resources.bindGroupLayouts.texture,
		Entries: []gpu.BindGroupEntry{
			{Binding: 0, Resource: gpu.BindingResource{TextureView: textureView}},
			{Binding: 1, Resource: gpu.BindingResource{Sampler: c.resources.atlasSampler}},
		},
	})
}

func (c *wgpuRendererCore) prepareTextureBindGroups(scene *gpui.Scene) {
	textureIDs := make([]gpui.AtlasTextureID, 0, 8)
	for batch := range scene.Batches() {
		var textureID gpui.AtlasTextureID
		switch batch := batch.(type) {
		case gpui.PrimitiveBatchMonochromeSprites:
			textureID = batch.TextureID
		case gpui.PrimitiveBatchSubpixelSprites:
			textureID = batch.TextureID
		case gpui.PrimitiveBatchPolychromeSprites:
			textureID = batch.TextureID
		default:
			continue
		}
		if !slices.Contains(textureIDs, textureID) {
			textureIDs = append(textureIDs, textureID)
		}
	}

	groups := c.resources.atlasTextureBindGroups
	for textureID, cached := range groups {
		if !slices.Contains(textureIDs, textureID) {
			cached.bindGroup.Release()
			delete(groups, textureID)
		}
	}

	for _, textureID := range textureIDs {
		textureInfo, ok := c.atlas.GetTextureInfo(textureID)
		if !ok {
			if cached, ok := groups[textureID]; ok {
				cached.bindGroup.Release()
				delete(groups, textureID)
			}
			continue
		}
		cached, ok := groups[textureID]
		if ok && cached.textureGeneration == textureInfo.Generation {
			continue
		}

		bindGroup := c.createTextureBindGroup("atlas_texture_bind_group", textureInfo.View)
		if ok {
			cached.bindGroup.Release()
		}
		groups[textureID] = cachedTextureBindGroup{
			textureGeneration: textureInfo.Generation,
			bindGroup:         bindGroup,
		}
	}
}

func (c *wgpuRendererCore) drawInstances(instances *instanceBinding, pipeline gpu.RenderPipeline, r gpui.Range, pass gpu.RenderPass) {
	if r.Start >= r.End {
		return
	}
	pass.SetPipeline(pipeline)
	pass.SetBindGroup(1, instances.bindGroup, nil)
	pass.Draw(4, uint32(r.End-r.Start), 0, instances.firstInstance+uint32(r.Start))
}

func (c *wgpuRendererCore) drawSprites(spriteInstances *instanceBinding, textureID gpui.AtlasTextureID, pipeline gpu.RenderPipeline, r gpui.Range, pass gpu.RenderPass) {
	if r.Start >= r.End {
		return
	}
	texture, ok := c.resources.atlasTextureBindGroups[textureID]
	if !ok {
		return
	}
	pass.SetPipeline(pipeline)
	pass.SetBindGroup(1, spriteInstances.bindGroup, nil)
	pass.SetBindGroup(2, texture.bindGroup, nil)
	pass.Draw(4, uint32(r.End-r.Start), 0, spriteInstances.firstInstance+uint32(r.Start))
}

type instanceSlice struct {
	data   []byte
	stride uint64
}

func instanceBytes[T any](instances []T) instanceSlice {
	var zero T
	stride := max(uint64(unsafe.Sizeof(zero)), 1)
	if len(instances) == 0 {
		return instanceSlice{stride: stride}
	}
	return instanceSlice{
		data:   unsafe.Slice((*byte)(unsafe.Pointer(&instances[0])), len(instances)*int(unsafe.Sizeof(zero))),
		stride: stride,
	}
}

func (c *wgpuRendererCore) instanceAllocationLayout(stride, byteLen uint64) (uint64, uint64) {
	size := max(byteLen, 16)
	if c.usesTextureInstanceData {
		return leastCommonMultiple(c.instanceDataAlignment, stride), nextMultipleOf(size, instanceTextureTexelSize)
	}
	return max(c.instanceDataAlignment, 1), size
}

func (c *wgpuRendererCore) instanceAllocationBound(instances instanceSlice) uint64 {
	alignment, allocationSize := c.instanceAllocationLayout(instances.stride, uint64(len(instances.data)))
	return alignment + allocationSize
}

func (c *wgpuRendererCore) reserveInstanceData(instanceOffset *uint64, size uint64) error {
	if *instanceOffset+size > c.instanceDataCapacity {
		if err := c.growInstanceData(size); err != nil {
			return err
		}
		*instanceOffset = 0
	}
	return nil
}

func (c *wgpuRendererCore) writeInstanceData(instanceOffset *uint64, instances instanceSlice) (instanceAllocation, error) {
	data := instances.data
	stride := instances.stride
	alignment, allocationSize := c.instanceAllocationLayout(stride, uint64(len(data)))
	offset := nextMultipleOf(*instanceOffset, alignment)
	if offset+allocationSize > c.instanceDataCapacity {
		if err := c.growInstanceData(allocationSize); err != nil {
			return instanceAllocation{}, err
		}
		offset = 0
	}
	*instanceOffset = offset + allocationSize

	var firstInstance uint32
	if c.usesTextureInstanceData {
		index := offset / stride
		if index > uint64(^uint32(0)) {
			return instanceAllocation{}, errors.New("instance index exceeds u32 range")
		}
		firstInstance = uint32(index)
	}

	if len(data) != 0 {
		if c.resources.instanceData.storage != nil {
			c.resources.queue.WriteBuffer(c.resources.instanceData.storage, offset, data)
		} else {
			c.writeInstanceTexture(offset, data)
		}
	}
	return instanceAllocation{
		offset:        offset,
		size:          max(uint64(len(data)), 16),
		firstInstance: firstInstance,
	}, nil
}

func (c *wgpuRendererCore) instanceBindingResource(allocation *instanceAllocation) gpu.BindingResource {
	if buffer := c.resources.instanceData.storage; buffer != nil {
		return gpu.BindingResource{Buffer: buffer, Offset: allocation.offset, Size: allocation.size}
	}
	return gpu.BindingResource{TextureView: c.resources.instanceData.view}
}

func (c *wgpuRendererCore) writeInstanceBinding(label string, instanceOffset *uint64, instances instanceSlice) (instanceBinding, error) {
	allocation, err := c.writeInstanceData(instanceOffset, instances)
	if err != nil {
		return instanceBinding{}, err
	}
	bindGroup := c.resources.device.CreateBindGroup(&gpu.BindGroupDescriptor{
		Label:   label,
		Layout:  c.resources.bindGroupLayouts.instances,
		Entries: []gpu.BindGroupEntry{{Binding: 0, Resource: c.instanceBindingResource(&allocation)}},
	})
	return instanceBinding{bindGroup: bindGroup, firstInstance: allocation.firstInstance}, nil
}

func (c *wgpuRendererCore) writePathTiles(scene *gpui.Scene, instanceOffset *uint64) (pathTileBinding, error) {
	paintData := make([]gpui.PathPaint, 0, len(scene.Paths))
	var curveData []gpui.PathCurve
	var tileData []gpui.PathTile
	for _, path := range scene.Paths {
		path.AppendTo(&paintData, &curveData, &tileData)
	}
	paintBytes, curveBytes, tileBytes := instanceBytes(paintData), instanceBytes(curveData), instanceBytes(tileData)

	reserved := c.instanceAllocationBound(paintBytes) + c.instanceAllocationBound(curveBytes) + c.instanceAllocationBound(tileBytes)
	if err := c.reserveInstanceData(instanceOffset, reserved); err != nil {
		return pathTileBinding{}, err
	}

	paints, err := c.writeInstanceData(instanceOffset, paintBytes)
	if err != nil {
		return pathTileBinding{}, err
	}
	curves, err := c.writeInstanceData(instanceOffset, curveBytes)
	if err != nil {
		return pathTileBinding{}, err
	}
	tiles, err := c.writeInstanceData(instanceOffset, tileBytes)
	if err != nil {
		return pathTileBinding{}, err
	}

	var bindGroup gpu.BindGroup
	if c.usesTextureInstanceData {
		bindGroup = c.resources.device.CreateBindGroup(&gpu.BindGroupDescriptor{
			Label:   "path_tiles_bind_group",
			Layout:  c.resources.bindGroupLayouts.pathTiles,
			Entries: []gpu.BindGroupEntry{{Binding: 0, Resource: c.instanceBindingResource(&tiles)}},
		})
	} else {
		bindGroup = c.resources.device.CreateBindGroup(&gpu.BindGroupDescriptor{
			Label:  "path_tiles_bind_group",
			Layout: c.resources.bindGroupLayouts.pathTiles,
			Entries: []gpu.BindGroupEntry{
				{Binding: 0, Resource: c.instanceBindingResource(&tiles)},
				{Binding: 1, Resource: c.instanceBindingResource(&curves)},
				{Binding: 2, Resource: c.instanceBindingResource(&paints)},
			},
		})
	}
	return pathTileBinding{
		tiles:     instanceBinding{bindGroup: bindGroup, firstInstance: tiles.firstInstance},
		paintBase: paints.firstInstance,
		curveBase: curves.firstInstance,
	}, nil
}

func (c *wgpuRendererCore) writeInstanceTexture(offset uint64, data []byte) {
	target := &c.resources.instanceData
	if target.texture == nil {
		return
	}
	width, height := target.width, target.height
	byteOffset := 0
	texelOffset := offset / instanceTextureTexelSize
	for byteOffset < len(data) {
		x := uint32(texelOffset % uint64(width))
		y := uint32(texelOffset / uint64(width))
		if y >= height {
			if gpuiutil.DebugAssertions {
				panic(fmt.Sprintf("instance texture write out of bounds: row %d >= height %d", y, height))
			}
			slog.Error(fmt.Sprintf("instance texture write out of bounds; dropping %d bytes of instance data", len(data)-byteOffset))
			return
		}
		availableTexels := uint64(width - x)
		remainingBytes := len(data) - byteOffset
		completeTexels := uint64(remainingBytes) / instanceTextureTexelSize
		texels := min(completeTexels, availableTexels)
		if texels > 0 {
			byteCount := int(texels * instanceTextureTexelSize)
			c.resources.queue.WriteTexture(
				gpu.TexelCopyTextureInfo{Texture: target.texture, MipLevel: 0, Origin: gpu.Origin3d{X: x, Y: y}, Aspect: gpu.TextureAspectAll},
				data[byteOffset:byteOffset+byteCount],
				gpu.TexelCopyBufferLayout{Offset: 0, BytesPerRow: uint32(byteCount)},
				gpu.Extent3d{Width: uint32(texels), Height: 1, DepthOrArrayLayers: 1},
			)
			byteOffset += byteCount
			texelOffset += texels
			continue
		}

		var finalTexel [instanceTextureTexelSize]byte
		copy(finalTexel[:remainingBytes], data[byteOffset:])
		c.resources.queue.WriteTexture(
			gpu.TexelCopyTextureInfo{Texture: target.texture, MipLevel: 0, Origin: gpu.Origin3d{X: x, Y: y}, Aspect: gpu.TextureAspectAll},
			finalTexel[:],
			gpu.TexelCopyBufferLayout{Offset: 0, BytesPerRow: uint32(instanceTextureTexelSize)},
			gpu.Extent3d{Width: 1, Height: 1, DepthOrArrayLayers: 1},
		)
		break
	}
}

func (c *wgpuRendererCore) growInstanceData(required uint64) error {
	capacity := min(max(c.instanceDataCapacity*2, nextPowerOfTwo(required)), c.maxInstanceDataSize)
	if !(capacity >= required) {
		return fmt.Errorf("instance data needs %d bytes, above the maximum of %d", required, c.maxInstanceDataSize)
	}
	if !(capacity > c.instanceDataCapacity) {
		return fmt.Errorf("frame instance data exceeds the %d-byte maximum", c.maxInstanceDataSize)
	}
	slog.Debug(fmt.Sprintf("instance data grown from %d to %d", c.instanceDataCapacity, capacity))
	old := c.resources.instanceData
	if c.usesTextureInstanceData {
		maxTextureDimension := c.resources.device.Limits().MaxTextureDimension2D
		data, actualCapacity := wgpuRendererCoreCreateInstanceTexture(c.resources.device, capacity, maxTextureDimension)
		c.resources.instanceData = data
		c.instanceDataCapacity = actualCapacity
	} else {
		c.resources.instanceData = instanceData{storage: c.resources.device.CreateBuffer(&gpu.BufferDescriptor{
			Label: "instance_buffer",
			Size:  capacity,
			Usage: gpu.BufferUsagesStorage | gpu.BufferUsagesCopyDst,
		})}
		c.instanceDataCapacity = capacity
	}
	old.release()
	return nil
}

// UnconfigureSurface drops the surface so rendering is skipped until ReplaceSurface provides a
// new one. The device, queue, atlas and pipelines stay alive.
func (r *WgpuRenderer) UnconfigureSurface() {
	if r.state.kind == rendererStateReady {
		r.setState(rendererState{kind: rendererStateUnconfigured, core: r.state.core})
	}
}

// ReplaceSurface replaces the surface with one for window, keeping the device, queue, atlas and
// pipelines so cached atlas texture ids stay valid. instance must be the instance of the
// renderer's WgpuContext.
func (r *WgpuRenderer) ReplaceSurface(window gpui.RawWindowHandle, config WgpuSurfaceConfig, instance gpu.Instance) error {
	surface, err := createSurface(instance, window)
	if err != nil {
		return err
	}

	width := max(uint32(config.Size.Width), 1)
	height := max(uint32(config.Size.Height), 1)

	alphaMode := r.opaqueAlphaMode
	if config.Transparent {
		alphaMode = r.transparentAlphaMode
	}
	alphaModeChanged := alphaMode != r.surfaceConfig.AlphaMode

	r.surfaceConfig.Width = width
	r.surfaceConfig.Height = height
	r.surfaceConfig.AlphaMode = alphaMode
	if mode, ok := config.PreferredPresentMode.Get(); ok {
		r.surfaceConfig.PresentMode = mode
	}

	var core *wgpuRendererCore
	switch r.state.kind {
	case rendererStateReady:
		core = r.state.core
		r.setState(rendererState{kind: rendererStateUnconfigured, core: core})
	case rendererStateUnconfigured:
		core = r.state.core
	case rendererStateReleased:
		surface.Release()
		return errors.New("Cannot replace the surface: GPU resources have been released")
	}
	if alphaModeChanged {
		core.rebuildPipelines(r.surfaceConfig.Format, alphaMode)
	}
	surface.Configure(core.resources.device, &r.surfaceConfig)
	r.state = rendererState{kind: rendererStateReady, surface: surface, core: core}

	return nil
}

// Destroy releases the surface-bound GPU resources eagerly, so the native window and its
// display connection can be closed afterwards. It is idempotent.
func (r *WgpuRenderer) Destroy() {
	r.setState(rendererState{kind: rendererStateReleased})
}

// DeviceLost reports whether the GPU device was lost and recovery is needed.
func (r *WgpuRenderer) DeviceLost() bool {
	return r.deviceErrors.DeviceLost()
}

// NeedsRedraw reports whether a redraw is needed because GPU state was cleared, and clears the
// flag.
func (r *WgpuRenderer) NeedsRedraw() bool {
	needsRedraw := r.needsRedraw
	r.needsRedraw = false
	return needsRedraw
}

// Recover recreates the renderer's GPU state after DeviceLost reports true. The first window
// to recover recreates the shared context; later windows adopt it.
func (r *WgpuRenderer) Recover(window gpui.RawWindowHandle) error {
	if r.context == nil {
		panic("recover requires gpu_context")
	}
	gpuContext := r.context

	needsNewContext := gpuContext.Context == nil || gpuContext.Context.DeviceLost()

	config := WgpuSurfaceConfig{
		Size: gpui.Size[gpui.DevicePixels]{
			Width:  gpui.DevicePixels(int32(r.surfaceConfig.Width)),
			Height: gpui.DevicePixels(int32(r.surfaceConfig.Height)),
		},
		Transparent:          r.surfaceConfig.AlphaMode != gpu.CompositeAlphaModeOpaque,
		PreferredPresentMode: gpuiutil.Some(r.surfaceConfig.PresentMode),
	}

	var surface gpu.Surface
	if needsNewContext {
		slog.Warn("GPU device lost, recreating context...")

		r.setState(rendererState{kind: rendererStateReleased})
		gpuContext.Context = nil

		time.Sleep(350 * time.Millisecond)

		instance := WgpuContextInstance(gpuiutil.Some(window))
		created, err := createSurface(instance, window)
		if err != nil {
			instance.Release()
			return err
		}
		newContext, err := NewWgpuContextRejectingSoftware(instance, created, &config, r.compositorGPU)
		if err != nil {
			created.Release()
			return err
		}
		gpuContext.Context = newContext
		surface = created
	} else {
		created, err := createSurface(gpuContext.Context.Instance, window)
		if err != nil {
			return err
		}
		surface = created
	}

	context := gpuContext.Context

	r.setState(rendererState{kind: rendererStateReleased})
	r.atlas.HandleDeviceLost(context)

	isBGR := r.isBGR
	renderer, err := newWgpuRendererInternal(gpuContext, context, surface, config, needsNewContext, r.compositorGPU, r.atlas)
	if err != nil {
		return err
	}
	*r = *renderer
	r.SetSubpixelLayout(isBGR)

	slog.Info("GPU recovery complete")
	return nil
}

type headlessRenderTarget struct {
	texture gpu.Texture
	view    gpu.TextureView
}

func (t *headlessRenderTarget) size() gpui.Size[gpui.DevicePixels] {
	return gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(int32(t.texture.Width())),
		Height: gpui.DevicePixels(int32(t.texture.Height())),
	}
}

func (t *headlessRenderTarget) release() {
	t.view.Release()
	t.texture.Release()
}

// WgpuHeadlessRenderer renders scenes into an offscreen texture and reads them back, for
// screenshot tests and benchmarks.
type WgpuHeadlessRenderer struct {
	context                 *WgpuContext
	core                    *wgpuRendererCore
	renderTarget            *headlessRenderTarget
	observedErrorGeneration uint64
}

// NewWgpuHeadlessRenderer creates a headless renderer on the shared headless device.
func NewWgpuHeadlessRenderer() (*WgpuHeadlessRenderer, error) {
	context, targetFormat, err := wgpuContextNewHeadless()
	if err != nil {
		return nil, err
	}
	observedErrorGeneration := context.Errors().CurrentGeneration()
	atlas := WgpuAtlasFromContext(context)
	core := newWgpuRendererCore(context, atlas, targetFormat, gpu.CompositeAlphaModeOpaque)

	return &WgpuHeadlessRenderer{
		context:                 context,
		core:                    core,
		renderTarget:            nil,
		observedErrorGeneration: observedErrorGeneration,
	}, nil
}

// Destroy releases the renderer's GPU resources; the shared headless device stays alive. It is
// upstream's drop of the renderer.
func (h *WgpuHeadlessRenderer) Destroy() {
	if h.renderTarget != nil {
		h.renderTarget.release()
		h.renderTarget = nil
	}
	if h.core != nil {
		h.core.release()
		h.core = nil
	}
}

func (h *WgpuHeadlessRenderer) ensureRenderTarget(size gpui.Size[gpui.DevicePixels]) error {
	if !(size.Width > 0 && size.Height > 0) {
		return fmt.Errorf("invalid headless render target size: %v", size)
	}
	if !(uint32(size.Width) <= h.core.maxTextureSize && uint32(size.Height) <= h.core.maxTextureSize) {
		return fmt.Errorf("headless render target size %v exceeds maximum texture dimension %d", size, h.core.maxTextureSize)
	}
	if h.renderTarget != nil && h.renderTarget.size() == size {
		return nil
	}

	texture := h.core.resources.device.CreateTexture(&gpu.TextureDescriptor{
		Label:         "headless_render_target",
		Size:          gpu.Extent3d{Width: uint32(size.Width), Height: uint32(size.Height), DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     gpu.TextureDimensionD2,
		Format:        h.core.targetFormat,
		Usage:         gpu.TextureUsagesRenderAttachment | gpu.TextureUsagesCopySrc,
	})
	view := texture.CreateView(nil)
	if h.renderTarget != nil {
		h.renderTarget.release()
	}
	h.renderTarget = &headlessRenderTarget{texture: texture, view: view}
	return nil
}

func (h *WgpuHeadlessRenderer) checkGPUErrors() error {
	if h.context.DeviceLost() {
		return errors.New("GPU device was lost during headless rendering")
	}
	if errText, ok := h.context.Errors().ObserveError(&h.observedErrorGeneration); ok {
		return fmt.Errorf("GPU error during headless rendering: %s", errText)
	}
	return nil
}

func (h *WgpuHeadlessRenderer) render(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) error {
	if err := h.checkGPUErrors(); err != nil {
		return err
	}
	if err := h.ensureRenderTarget(size); err != nil {
		return err
	}
	if h.renderTarget == nil {
		return errors.New("Headless render target was not created")
	}
	view := h.renderTarget.view
	_, err := h.core.renderFrame(scene, view, size, gpu.ColorBlack)
	return err
}

func (h *WgpuHeadlessRenderer) readImage() (*image.RGBA, error) {
	target := h.renderTarget
	if target == nil {
		return nil, errors.New("Headless render target was not created")
	}
	width := target.texture.Width()
	height := target.texture.Height()
	if width > (^uint32(0))/4 {
		return nil, errors.New("Headless render target row size overflowed")
	}
	bytesPerRow := width * 4
	if bytesPerRow > ^uint32(0)-(gpu.CopyBytesPerRowAlignment-1) {
		return nil, errors.New("Headless padded row size overflowed")
	}
	paddedBytesPerRow := uint32(nextMultipleOf(uint64(bytesPerRow), gpu.CopyBytesPerRowAlignment))
	hi, bufferSize := bits.Mul64(uint64(paddedBytesPerRow), uint64(height))
	if hi != 0 {
		return nil, errors.New("Headless readback buffer size overflowed")
	}
	maxBufferSize := h.core.resources.device.Limits().MaxBufferSize
	if bufferSize > maxBufferSize {
		return nil, fmt.Errorf("Headless readback buffer size %d exceeds maximum buffer size %d", bufferSize, maxBufferSize)
	}
	readbackBuffer := h.core.resources.device.CreateBuffer(&gpu.BufferDescriptor{
		Label: "headless_readback_buffer",
		Size:  bufferSize,
		Usage: gpu.BufferUsagesCopyDst | gpu.BufferUsagesMapRead,
	})
	defer readbackBuffer.Release()
	encoder := h.core.resources.device.CreateCommandEncoder(&gpu.CommandEncoderDescriptor{Label: "headless_readback_encoder"})
	defer encoder.Release()
	encoder.CopyTextureToBuffer(
		gpu.TexelCopyTextureInfo{Texture: target.texture, MipLevel: 0, Aspect: gpu.TextureAspectAll},
		gpu.TexelCopyBufferInfo{
			Buffer: readbackBuffer,
			Layout: gpu.TexelCopyBufferLayout{Offset: 0, BytesPerRow: paddedBytesPerRow, RowsPerImage: height},
		},
		gpu.Extent3d{Width: width, Height: height, DepthOrArrayLayers: 1},
	)
	commands := encoder.Finish()
	defer commands.Release()
	h.core.resources.queue.Submit(commands)
	var mapped bool
	var mapErr error
	readbackBuffer.MapAsync(gpu.MapModeRead, 0, bufferSize, func(err error) {
		mapped = true
		mapErr = err
	})
	if err := h.core.resources.device.Poll(true); err != nil {
		return nil, fmt.Errorf("Failed to wait for headless rendering: %w", err)
	}
	if !mapped {
		return nil, errors.New("Failed to receive headless mapping result: timed out waiting on channel")
	}
	if mapErr != nil {
		return nil, fmt.Errorf("Failed to map headless readback buffer: %w", mapErr)
	}
	if err := h.checkGPUErrors(); err != nil {
		return nil, err
	}

	mappedData := readbackBuffer.GetMappedRange(0, bufferSize)
	if uint64(len(mappedData)) < bufferSize {
		readbackBuffer.Unmap()
		return nil, errors.New("Failed to get the mapped range of the headless readback buffer")
	}
	pixels := make([]byte, 0, int(bytesPerRow)*int(height))
	for row := range int(height) {
		start := row * int(paddedBytesPerRow)
		pixels = append(pixels, mappedData[start:start+int(bytesPerRow)]...)
	}
	readbackBuffer.Unmap()

	if h.core.targetFormat == gpu.TextureFormatBgra8Unorm {
		for i := 0; i+3 < len(pixels); i += 4 {
			pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
		}
	}

	return &image.RGBA{
		Pix:    pixels,
		Stride: int(bytesPerRow),
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}, nil
}

// RenderSceneToImage renders the scene at size and reads it back as an RGBA image.
func (h *WgpuHeadlessRenderer) RenderSceneToImage(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) (*image.RGBA, error) {
	if err := h.render(scene, size); err != nil {
		return nil, err
	}
	return h.readImage()
}

// RenderScene renders the scene at size without reading it back.
func (h *WgpuHeadlessRenderer) RenderScene(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) error {
	return h.render(scene, size)
}

// SpriteAtlas returns the atlas the renderer samples sprites from.
func (h *WgpuHeadlessRenderer) SpriteAtlas() gpui.PlatformAtlas {
	return h.core.atlas
}

func instanceRange(r gpui.Range) gpui.Range {
	return gpui.Range{Start: int(uint32(r.Start)), End: int(uint32(r.End))}
}

func createSurface(instance gpu.Instance, window gpui.RawWindowHandle) (gpu.Surface, error) {
	var target gpu.SurfaceTarget
	switch window.Kind {
	case gpui.RawWindowHandleWayland:
		target = gpu.SurfaceTarget{WaylandDisplay: window.WaylandDisplay, WaylandSurface: window.WaylandSurface}
	case gpui.RawWindowHandleXcb:
		target = gpu.SurfaceTarget{XcbConnection: window.XcbConnection, XcbWindow: window.XcbWindow}
	case gpui.RawWindowHandleWeb:
		target = gpu.SurfaceTarget{CanvasID: window.WebCanvasID}
	default:
		return nil, fmt.Errorf("unsupported window handle kind %d", window.Kind)
	}
	return instance.CreateSurface(target)
}

type renderingParameters struct {
	gammaRatios               [4]float32
	grayscaleEnhancedContrast float32
	subpixelEnhancedContrast  float32
}

func envFloat32(name string, fallback float32) float32 {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 32)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return fallback
	}
	return float32(parsed)
}

func maxF32(value, other float32) float32 {
	if value != value {
		return other
	}
	return max(value, other)
}

func newRenderingParameters() renderingParameters {
	gamma := max(min(envFloat32("ZED_FONTS_GAMMA", 1.8), 2.2), 1.0)
	gammaRatios := gpui.GetGammaCorrectionRatios(gamma)

	grayscaleEnhancedContrast := maxF32(envFloat32("ZED_FONTS_GRAYSCALE_ENHANCED_CONTRAST", 1.0), 0.0)

	subpixelEnhancedContrast := maxF32(envFloat32("ZED_FONTS_SUBPIXEL_ENHANCED_CONTRAST", 0.5), 0.0)

	return renderingParameters{
		gammaRatios:               gammaRatios,
		grayscaleEnhancedContrast: grayscaleEnhancedContrast,
		subpixelEnhancedContrast:  subpixelEnhancedContrast,
	}
}
