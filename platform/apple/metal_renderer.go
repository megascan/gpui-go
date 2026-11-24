//go:build darwin

package apple

import (
	"errors"
	"fmt"
	"image"
	"log/slog"
	"math"
	"math/bits"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/mtl"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/render"
)

const instanceBufferAlignment = 256
const maxInstanceBufferSize = 256 * 1024 * 1024

const isIOS = runtime.GOOS == "ios"

// Context is the instance buffer pool shared by every renderer of an application.
type Context = *InstanceBufferPool

// Renderer is the renderer the Apple platforms use.
type Renderer = MetalRenderer

var _ render.Renderer = (*MetalRenderer)(nil)

// NewRenderer returns a window renderer backed by a new CAMetalLayer.
func NewRenderer(context Context, transparent bool, useDisplayP3 bool) (*Renderer, error) {
	return NewMetalRenderer(context, transparent, useDisplayP3)
}

// InstanceBufferPool recycles the per-frame instance buffers of every renderer sharing it.
type InstanceBufferPool struct {
	mu         sync.Mutex
	bufferSize int
	buffers    []instanceBuffer
}

// NewInstanceBufferPool returns an empty pool of 2 MiB buffers.
func NewInstanceBufferPool() *InstanceBufferPool {
	return &InstanceBufferPool{bufferSize: 2 * 1024 * 1024}
}

type instanceBuffer struct {
	metalBuffer mtl.Buffer
	size        int
}

func (p *InstanceBufferPool) reset(bufferSize int) {
	p.bufferSize = bufferSize
	for _, buffer := range p.buffers {
		mtl.Release(buffer.metalBuffer)
	}
	p.buffers = nil
}

func (p *InstanceBufferPool) acquire(device mtl.Device, unifiedMemory bool) (instanceBuffer, error) {
	if n := len(p.buffers); n > 0 {
		buffer := p.buffers[n-1]
		p.buffers = p.buffers[:n-1]
		return buffer, nil
	}
	var options mtl.ResourceOptions
	if unifiedMemory {
		options = mtl.ResourceStorageModeShared | mtl.ResourceCPUCacheModeWriteCombined
	} else {
		options = mtl.ResourceStorageModeManaged
	}
	buffer := device.NewBufferWithLength(uint(p.bufferSize), options)
	if buffer == 0 {
		return instanceBuffer{}, fmt.Errorf("allocating %d-byte instance buffer", p.bufferSize)
	}
	return instanceBuffer{metalBuffer: buffer, size: p.bufferSize}, nil
}

func (p *InstanceBufferPool) release(buffer instanceBuffer) {
	if buffer.size == p.bufferSize {
		p.buffers = append(p.buffers, buffer)
	} else {
		mtl.Release(buffer.metalBuffer)
	}
}

type displayLinkMode struct {
	attached bool
	drawable mtl.Drawable
}

// MetalRenderer draws scenes with Metal into a CAMetalLayer or, headless, into offscreen textures.
type MetalRenderer struct {
	device                         mtl.Device
	layer                          mtl.MetalLayer
	isAppleGPU                     bool
	isUnifiedMemory                bool
	presentsWithTransaction        bool
	displayLinkMode                displayLinkMode
	opaque                         bool
	showsNativeViews               bool
	commandQueue                   mtl.CommandQueue
	pathTilesPipelineState         mtl.RenderPipelineState
	shadowsPipelineState           mtl.RenderPipelineState
	quadsPipelineState             mtl.RenderPipelineState
	opaqueQuadsPipelineState       mtl.RenderPipelineState
	underlinesPipelineState        mtl.RenderPipelineState
	monochromeSpritesPipelineState mtl.RenderPipelineState
	polychromeSpritesPipelineState mtl.RenderPipelineState
	surfacesPipelineState          mtl.RenderPipelineState
	nativeViewsPipelineState       mtl.RenderPipelineState
	depthWriteState                mtl.DepthStencilState
	depthTestState                 mtl.DepthStencilState
	depthTexture                   mtl.Texture
	instanceBufferPool             *InstanceBufferPool
	spriteAtlas                    *MetalAtlas
	coreVideoTextureCache          uintptr
	headlessRenderTarget           mtl.Texture
	destroyed                      bool
}

// NewMetalRenderer creates a renderer with a new CAMetalLayer for window-based rendering.
func NewMetalRenderer(instanceBufferPool *InstanceBufferPool, transparent bool, useDisplayP3 bool) (*MetalRenderer, error) {
	device, err := metalRendererCreateDevice()
	if err != nil {
		return nil, err
	}
	layer := mtl.NewMetalLayer()
	metalRendererConfigureLayer(layer, device, transparent, useDisplayP3)
	return metalRendererNewInternal(device, layer, !transparent, instanceBufferPool)
}

// MetalRendererFromLayer creates a renderer for a CAMetalLayer owned by a platform view, such as
// the backing layer UIKit creates for a view whose layerClass is CAMetalLayer. The renderer
// retains the layer.
func MetalRendererFromLayer(instanceBufferPool *InstanceBufferPool, layer mtl.MetalLayer, transparent bool) (*MetalRenderer, error) {
	device, err := metalRendererCreateDevice()
	if err != nil {
		return nil, err
	}
	layer = mtl.Retain(layer)
	metalRendererConfigureLayer(layer, device, transparent, false)
	return metalRendererNewInternal(device, layer, !transparent, instanceBufferPool)
}

func metalRendererConfigureLayer(layer mtl.MetalLayer, device mtl.Device, transparent bool, useDisplayP3 bool) {
	layer.SetDevice(device)
	layer.SetPixelFormat(mtl.PixelFormatBGRA8Unorm)
	if useDisplayP3 {
		if colorSpace, ok := displayP3ColorSpace(); ok {
			layer.SetColorspace(colorSpace)
			cf.Release(cf.TypeRef(colorSpace))
		} else {
			slog.Error("unable to create the Display P3 color space")
		}
	}
	layer.SetOpaque(!transparent)
	layer.SetMaximumDrawableCount(3)
	if gpuiutil.DebugAssertions {
		layer.SetFramebufferOnly(false)
	}
	layer.SetAllowsNextDrawableTimeout(false)
	layer.SetNeedsDisplayOnBoundsChange(true)
	if !isIOS {
		const kCALayerWidthSizable, kCALayerHeightSizable = 1 << 1, 1 << 4
		layer.SetAutoresizingMask(kCALayerWidthSizable | kCALayerHeightSizable)
	}
}

// NewMetalRendererHeadless creates a renderer for offscreen rendering without a window: it renders
// scenes to images without a CAMetalLayer, window or AppKit. Use RenderSceneToImage to render.
func NewMetalRendererHeadless(instanceBufferPool *InstanceBufferPool) (*MetalRenderer, error) {
	device, err := metalRendererCreateDevice()
	if err != nil {
		return nil, err
	}
	return metalRendererNewInternal(device, 0, true, instanceBufferPool)
}

func metalRendererCreateDevice() (mtl.Device, error) {
	if err := mtl.Load(); err != nil {
		return 0, fmt.Errorf("accessing a compatible Metal device: %w", err)
	}
	if isIOS {
		return metalRendererSystemDefaultDevice()
	}
	devices := mtl.CopyAllDevices()
	if len(devices) == 0 {
		slog.Error("Unable to enumerate Metal devices; attempting to use system default device")
		return metalRendererSystemDefaultDevice()
	}
	best := 0
	key := func(d mtl.Device) (bool, bool) { return d.IsRemovable(), !d.IsLowPower() }
	bestRemovable, bestHighPower := key(devices[0])
	for i := 1; i < len(devices); i++ {
		removable, highPower := key(devices[i])
		if boolLess(removable, bestRemovable) || (removable == bestRemovable && boolLess(highPower, bestHighPower)) {
			best, bestRemovable, bestHighPower = i, removable, highPower
		}
	}
	for i, d := range devices {
		if i != best {
			mtl.Release(d)
		}
	}
	return devices[best], nil
}

func boolLess(a, b bool) bool { return !a && b }

func metalRendererSystemDefaultDevice() (mtl.Device, error) {
	device, ok := mtl.CreateSystemDefaultDevice()
	if !ok {
		return 0, errors.New("accessing a compatible Metal device")
	}
	return device, nil
}

func metalRendererNewInternal(device mtl.Device, layer mtl.MetalLayer, opaque bool, instanceBufferPool *InstanceBufferPool) (renderer *MetalRenderer, err error) {
	r := &MetalRenderer{
		device:             device,
		layer:              layer,
		opaque:             opaque,
		instanceBufferPool: instanceBufferPool,
	}
	defer func() {
		if err != nil {
			r.Destroy()
		}
	}()
	if err := loadCoreVideo(); err != nil {
		return nil, err
	}
	var library mtl.Library
	objc.AutoreleasePool(func() {
		options := mtl.NewCompileOptions()
		defer mtl.Release(options)
		library, err = device.NewLibraryWithSource(stitchedShaderSource(), options)
	})
	if err != nil {
		return nil, fmt.Errorf("building Metal library: %w", err)
	}
	defer mtl.Release(library)

	r.isUnifiedMemory = isIOS || device.HasUnifiedMemory()
	r.isAppleGPU = device.SupportsFamily(mtl.GPUFamilyApple1)

	pipelines := []struct {
		state    *mtl.RenderPipelineState
		label    string
		blending bool
	}{
		{&r.pathTilesPipelineState, "path_tiles", true},
		{&r.shadowsPipelineState, "shadows", true},
		{&r.quadsPipelineState, "quads", true},
		{&r.opaqueQuadsPipelineState, "opaque_quads", false},
		{&r.underlinesPipelineState, "underlines", true},
		{&r.monochromeSpritesPipelineState, "monochrome_sprites", true},
		{&r.polychromeSpritesPipelineState, "polychrome_sprites", true},
		{&r.surfacesPipelineState, "surfaces", true},
		{&r.nativeViewsPipelineState, "native_views", false},
	}
	for i, p := range pipelines {
		state, err := buildPipelineState(device, library, p.label, shaderFunctionNames[i][0], shaderFunctionNames[i][1], mtl.PixelFormatBGRA8Unorm, p.blending)
		if err != nil {
			return nil, err
		}
		*p.state = state
	}

	if r.depthWriteState, err = buildDepthStencilState(device, true); err != nil {
		return nil, err
	}
	if r.depthTestState, err = buildDepthStencilState(device, false); err != nil {
		return nil, err
	}

	r.commandQueue = device.NewCommandQueue()
	if r.commandQueue == 0 {
		return nil, errors.New("creating command queue")
	}
	supportsSharedStorage := isIOS || r.isAppleGPU
	r.spriteAtlas = NewMetalAtlas(device, supportsSharedStorage)
	var cache uintptr
	if status := cvMetalTextureCacheCreate(0, 0, device.ID(), 0, &cache); status != kCVReturnSuccess {
		return nil, fmt.Errorf("creating CoreVideo Metal texture cache: status %d", status)
	}
	if cache == 0 {
		return nil, errors.New("retrieving CoreVideo Metal texture cache")
	}
	r.coreVideoTextureCache = cache
	return r, nil
}

// Layer returns the renderer's CAMetalLayer, or false for a headless renderer.
func (r *MetalRenderer) Layer() (mtl.MetalLayer, bool) {
	return r.layer, r.layer != 0
}

// LayerPtr returns the renderer's CAMetalLayer pointer, or 0 for a headless renderer.
func (r *MetalRenderer) LayerPtr() uintptr {
	return uintptr(r.layer)
}

// SpriteAtlas returns the atlas the renderer samples sprites from.
func (r *MetalRenderer) SpriteAtlas() gpui.PlatformAtlas {
	return r.spriteAtlas
}

// SetFrameDrawable makes the next frame draw into drawable, a +1 reference the renderer takes
// over, instead of asking the layer for one. A CAMetalDisplayLink hands out each frame's drawable
// early enough for the frame to reach the display on time. Pass 0 once the frame is done, so a
// drawable the frame did not use goes back to the layer.
//
// It returns the +1 drawable this replaces, which the caller releases: after a frame, a nonzero
// result means the frame neither drew nor presented.
func (r *MetalRenderer) SetFrameDrawable(drawable mtl.Drawable) mtl.Drawable {
	if r.displayLinkMode.attached {
		current := r.displayLinkMode.drawable
		r.displayLinkMode.drawable = drawable
		return current
	}
	if gpuiutil.DebugAssertions && drawable != 0 {
		panic("received a display link drawable while detached")
	}
	mtl.Release(drawable)
	return 0
}

// SetDisplayLinkAttached records whether a CAMetalDisplayLink drives the layer. While one does,
// frames are drawn only into the drawables it hands out.
func (r *MetalRenderer) SetDisplayLinkAttached(attached bool) {
	mtl.Release(r.displayLinkMode.drawable)
	r.displayLinkMode = displayLinkMode{attached: attached}
}

// IsDisplayLinkAttached reports whether a CAMetalDisplayLink drives the layer.
func (r *MetalRenderer) IsDisplayLinkAttached() bool {
	return r.displayLinkMode.attached
}

// PresentsWithTransaction reports whether frames are presented inside a Core Animation transaction.
func (r *MetalRenderer) PresentsWithTransaction() bool {
	return r.presentsWithTransaction
}

// SetPresentsWithTransaction sets whether frames are presented inside a Core Animation transaction.
func (r *MetalRenderer) SetPresentsWithTransaction(presentsWithTransaction bool) {
	r.presentsWithTransaction = presentsWithTransaction
	if r.layer != 0 {
		r.layer.SetPresentsWithTransaction(presentsWithTransaction)
	}
}

// UpdateDrawableSize resizes the layer's drawables.
func (r *MetalRenderer) UpdateDrawableSize(size gpui.Size[gpui.DevicePixels]) {
	if r.layer != 0 {
		r.layer.SetDrawableSize(objc.CGSize{Width: objc.CGFloat(size.Width), Height: objc.CGFloat(size.Height)})
	}
}

// UpdateTransparency switches between an opaque and a transparent layer.
func (r *MetalRenderer) UpdateTransparency(transparent bool) {
	r.opaque = !transparent
	if r.layer != 0 {
		r.layer.SetOpaque(r.opaque && !r.showsNativeViews)
	}
}

// SetSubpixelLayout does nothing: the Metal renderer draws no subpixel sprites.
func (r *MetalRenderer) SetSubpixelLayout(isBGR bool) {}

// SupportsDualSourceBlending reports false: the Metal renderer draws no subpixel sprites.
func (r *MetalRenderer) SupportsDualSourceBlending() bool { return false }

// GPUSpecs reports false: the Apple platforms do not describe their GPU.
func (r *MetalRenderer) GPUSpecs() (gpui.GPUSpecs, bool) { return gpui.GPUSpecs{}, false }

// DeviceLost reports false: Metal devices are not lost the way wgpu devices are.
func (r *MetalRenderer) DeviceLost() bool { return false }

// Recover does nothing, since DeviceLost never reports true.
func (r *MetalRenderer) Recover(handle gpui.RawWindowHandle) error { return nil }

// Destroy releases the renderer's Metal and CoreVideo objects, as dropping the Rust renderer
// does. It is idempotent.
func (r *MetalRenderer) Destroy() {
	if r.destroyed {
		return
	}
	r.destroyed = true
	mtl.Release(r.displayLinkMode.drawable)
	r.displayLinkMode = displayLinkMode{}
	for _, state := range []*mtl.RenderPipelineState{
		&r.pathTilesPipelineState, &r.shadowsPipelineState, &r.quadsPipelineState,
		&r.opaqueQuadsPipelineState, &r.underlinesPipelineState, &r.monochromeSpritesPipelineState,
		&r.polychromeSpritesPipelineState, &r.surfacesPipelineState, &r.nativeViewsPipelineState,
	} {
		mtl.Release(*state)
		*state = 0
	}
	mtl.Release(r.depthWriteState)
	mtl.Release(r.depthTestState)
	mtl.Release(r.depthTexture)
	mtl.Release(r.headlessRenderTarget)
	mtl.Release(r.commandQueue)
	r.depthWriteState, r.depthTestState, r.depthTexture, r.headlessRenderTarget, r.commandQueue = 0, 0, 0, 0, 0
	if r.spriteAtlas != nil {
		r.spriteAtlas.Release()
	}
	if r.coreVideoTextureCache != 0 {
		cf.Release(cf.TypeRef(r.coreVideoTextureCache))
		r.coreVideoTextureCache = 0
	}
	mtl.Release(r.layer)
	r.layer = 0
	mtl.Release(r.device)
	r.device = 0
}

// Draw renders the scene into the layer's next drawable and presents it, reporting whether a
// frame was submitted.
func (r *MetalRenderer) Draw(scene *gpui.Scene) bool {
	if r.layer == 0 {
		slog.Error("draw() called on headless renderer - use render_scene_to_image() instead")
		return false
	}
	layer := r.layer
	showsNativeViews := len(scene.NativeViews) != 0
	if showsNativeViews != r.showsNativeViews {
		r.showsNativeViews = showsNativeViews
		layer.SetOpaque(r.opaque && !showsNativeViews)
	}
	var drawable mtl.Drawable
	if r.displayLinkMode.attached {
		drawable = r.displayLinkMode.drawable
		r.displayLinkMode.drawable = 0
		if drawable == 0 {
			slog.Error("skipped a frame drawn outside its display link's update")
			return false
		}
	} else {
		next, ok := layer.NextDrawable()
		if !ok {
			size := layer.DrawableSize()
			slog.Error(fmt.Sprintf("failed to retrieve next drawable, drawable size: CGSize { width: %v, height: %v }", size.Width, size.Height))
			return false
		}
		drawable = mtl.Retain(next)
	}
	defer mtl.Release(drawable)
	texture := drawable.Texture()
	viewportSize := gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(int32(texture.Width())),
		Height: gpui.DevicePixels(int32(texture.Height())),
	}

	commandBuffer, err := r.renderFrame(scene, texture, viewportSize)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to render: %v", err))
		return false
	}
	defer mtl.Release(commandBuffer)

	if r.presentsWithTransaction {
		commandBuffer.Commit()
		commandBuffer.WaitUntilScheduled()
		drawable.Present()
	} else {
		commandBuffer.PresentDrawable(drawable)
		commandBuffer.Commit()
	}
	return true
}

func (r *MetalRenderer) renderFrame(scene *gpui.Scene, texture mtl.Texture, viewportSize gpui.Size[gpui.DevicePixels]) (mtl.CommandBuffer, error) {
	writer, err := newInstanceBufferWriter(r.device, r.instanceBufferPool, r.isUnifiedMemory)
	if err != nil {
		return 0, err
	}
	instanceBindings, err := writeInstances(scene, writer)
	if err != nil {
		writer.discard()
		return 0, fmt.Errorf("writing instance data: %d paths, %d shadows, %d quads, %d underlines, %d mono, %d poly, %d surfaces: %w",
			len(scene.Paths), len(scene.Shadows), len(scene.Quads), len(scene.Underlines),
			len(scene.MonochromeSprites), len(scene.PolychromeSprites), len(scene.Surfaces), err)
	}
	var surfaceTextures []uintptr
	commandBuffer, err := r.drawPrimitivesToTexture(scene, &instanceBindings, texture, viewportSize, &surfaceTextures)
	if err != nil {
		writer.discard()
		for _, t := range surfaceTextures {
			cf.Release(cf.TypeRef(t))
		}
		return 0, err
	}

	pool := r.instanceBufferPool
	instanceBuffer := writer.finish()
	var once sync.Once
	commandBuffer.AddCompletedHandler(func(mtl.CommandBuffer) {
		once.Do(func() {
			pool.mu.Lock()
			pool.release(instanceBuffer)
			pool.mu.Unlock()
			for _, t := range surfaceTextures {
				cf.Release(cf.TypeRef(t))
			}
		})
	})
	return commandBuffer, nil
}

// RenderToImage renders the scene into a texture of the layer's drawable size and returns its
// pixels, without presenting the frame. It requires a layer-backed renderer; headless renderers
// use RenderSceneToImage.
func (r *MetalRenderer) RenderToImage(scene *gpui.Scene) (*image.RGBA, error) {
	if r.layer == 0 {
		return nil, errors.New("render_to_image requires a layer-backed renderer")
	}
	drawableSize := r.layer.DrawableSize()
	viewportSize := gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(int32(math.Ceil(float64(drawableSize.Width)))),
		Height: gpui.DevicePixels(int32(math.Ceil(float64(drawableSize.Height)))),
	}
	return r.renderOffscreen(scene, viewportSize)
}

// RenderSceneToImage renders a scene to an image without a window or CAMetalLayer.
func (r *MetalRenderer) RenderSceneToImage(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) (img *image.RGBA, err error) {
	if size.Width <= 0 || size.Height <= 0 {
		return nil, fmt.Errorf("Invalid size for render_scene_to_image: %v", size)
	}
	objc.AutoreleasePool(func() { img, err = r.renderOffscreen(scene, size) })
	return img, err
}

func (r *MetalRenderer) renderOffscreen(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) (*image.RGBA, error) {
	if size.Width <= 0 || size.Height <= 0 {
		return nil, fmt.Errorf("Invalid size for render_offscreen: %v", size)
	}
	textureDescriptor := mtl.NewTextureDescriptor()
	defer mtl.Release(textureDescriptor)
	textureDescriptor.SetWidth(uint(size.Width))
	textureDescriptor.SetHeight(uint(size.Height))
	textureDescriptor.SetPixelFormat(mtl.PixelFormatBGRA8Unorm)
	textureDescriptor.SetUsage(mtl.TextureUsageRenderTarget | mtl.TextureUsageShaderRead)
	usesSharedStorage := isIOS || r.isAppleGPU
	if usesSharedStorage {
		textureDescriptor.SetStorageMode(mtl.StorageModeShared)
	} else {
		textureDescriptor.SetStorageMode(mtl.StorageModeManaged)
	}
	targetTexture := r.device.NewTextureWithDescriptor(textureDescriptor)
	if targetTexture == 0 {
		return nil, errors.New("allocating offscreen render target")
	}
	defer mtl.Release(targetTexture)

	commandBuffer, err := r.renderFrame(scene, targetTexture, size)
	if err != nil {
		return nil, err
	}
	defer mtl.Release(commandBuffer)

	if !usesSharedStorage {
		blit := commandBuffer.BlitCommandEncoder()
		if blit == 0 {
			return nil, errors.New("creating blit command encoder")
		}
		blit.SynchronizeResource(targetTexture.ID())
		blit.EndEncoding()
	}

	commandBuffer.Commit()
	commandBuffer.WaitUntilCompleted()

	return readTextureToImage(targetTexture)
}

// RenderScene renders a scene to a reused offscreen texture without reading pixels back or
// waiting for the GPU, mirroring the CPU cost of presenting a frame; headless benchmarks use it.
func (r *MetalRenderer) RenderScene(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) (err error) {
	if size.Width <= 0 || size.Height <= 0 {
		return fmt.Errorf("Invalid size for render_scene: %v", size)
	}
	objc.AutoreleasePool(func() { err = r.renderScene(scene, size) })
	return err
}

func (r *MetalRenderer) renderScene(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) error {
	needsNewTarget := r.headlessRenderTarget == 0 ||
		r.headlessRenderTarget.Width() != uint(size.Width) ||
		r.headlessRenderTarget.Height() != uint(size.Height)
	if needsNewTarget {
		textureDescriptor := mtl.NewTextureDescriptor()
		defer mtl.Release(textureDescriptor)
		textureDescriptor.SetWidth(uint(size.Width))
		textureDescriptor.SetHeight(uint(size.Height))
		textureDescriptor.SetPixelFormat(mtl.PixelFormatBGRA8Unorm)
		textureDescriptor.SetUsage(mtl.TextureUsageRenderTarget | mtl.TextureUsageShaderRead)
		textureDescriptor.SetStorageMode(mtl.StorageModePrivate)
		texture := r.device.NewTextureWithDescriptor(textureDescriptor)
		if texture == 0 {
			return errors.New("allocating headless render target")
		}
		mtl.Release(r.headlessRenderTarget)
		r.headlessRenderTarget = texture
	}
	commandBuffer, err := r.renderFrame(scene, r.headlessRenderTarget, size)
	if err != nil {
		return err
	}
	commandBuffer.Commit()
	mtl.Release(commandBuffer)
	return nil
}

func (r *MetalRenderer) drawPrimitivesToTexture(scene *gpui.Scene, instanceBindings *instanceBindings, texture mtl.Texture, viewportSize gpui.Size[gpui.DevicePixels], surfaceTextures *[]uintptr) (mtl.CommandBuffer, error) {
	commandBuffer := r.commandQueue.CommandBuffer()
	if commandBuffer == 0 {
		return 0, errors.New("creating command buffer")
	}
	commandBuffer = mtl.Retain(commandBuffer)
	alpha := 0.0
	if r.opaque {
		alpha = 1.0
	}
	depthTexture, err := r.depthTextureMatching(texture)
	if err != nil {
		mtl.Release(commandBuffer)
		return 0, err
	}

	commandEncoder, err := newCommandEncoderForTexture(commandBuffer, texture, depthTexture, viewportSize, mtl.ClearColor{Red: 0, Green: 0, Blue: 0, Alpha: alpha})
	if err != nil {
		mtl.Release(commandBuffer)
		return 0, err
	}
	defer mtl.Release(commandEncoder)
	depthScale := gpui.QuadDepthScale(len(scene.Quads))
	commandEncoder.SetVertexBytes(unsafe.Pointer(&viewportSize), uint(unsafe.Sizeof(viewportSize)), globalInputIndexViewportSize)
	commandEncoder.SetVertexBytes(unsafe.Pointer(&depthScale), uint(unsafe.Sizeof(depthScale)), globalInputIndexDepthScale)

	r.drawOpaqueQuads(len(scene.Quads), instanceBindings, commandEncoder)
	commandEncoder.SetDepthStencilState(r.depthTestState)

	var quadCursor uint32
	pathTileCursor := 0
	for batch := range scene.Batches() {
		znear, zfar := 0.0, 1.0
		if _, isQuads := batch.(gpui.PrimitiveBatchQuads); !isQuads {
			depth := float64(gpui.QuadDepth(quadCursor, depthScale))
			znear, zfar = depth, depth
		}
		commandEncoder.SetViewport(mtl.Viewport{
			OriginX: 0,
			OriginY: 0,
			Width:   float64(viewportSize.Width),
			Height:  float64(viewportSize.Height),
			ZNear:   znear,
			ZFar:    zfar,
		})

		switch batch := batch.(type) {
		case gpui.PrimitiveBatchShadows:
			r.drawShadows(batch.Range, instanceBindings, commandEncoder)
		case gpui.PrimitiveBatchQuads:
			quadCursor += uint32(batch.End - batch.Start)
			r.drawQuads(batch.Range, instanceBindings, commandEncoder)
		case gpui.PrimitiveBatchPaths:
			tilesStart := pathTileCursor
			for _, path := range scene.Paths[batch.Start:batch.End] {
				pathTileCursor += path.TileCount()
			}
			r.drawPathTiles(gpui.Range{Start: tilesStart, End: pathTileCursor}, instanceBindings, commandEncoder)
		case gpui.PrimitiveBatchUnderlines:
			r.drawUnderlines(batch.Range, instanceBindings, commandEncoder)
		case gpui.PrimitiveBatchMonochromeSprites:
			r.drawMonochromeSprites(batch.TextureID, batch.Range, instanceBindings, commandEncoder)
		case gpui.PrimitiveBatchPolychromeSprites:
			r.drawPolychromeSprites(batch.TextureID, batch.Range, instanceBindings, commandEncoder)
		case gpui.PrimitiveBatchSurfaces:
			if err := r.drawSurfaces(scene.Surfaces[batch.Start:batch.End], batch.Start, instanceBindings, commandEncoder, surfaceTextures); err != nil {
				commandEncoder.EndEncoding()
				mtl.Release(commandBuffer)
				return 0, err
			}
		case gpui.PrimitiveBatchNativeViews:
			r.drawNativeViews(batch.Range, instanceBindings, commandEncoder)
		case gpui.PrimitiveBatchSubpixelSprites:
			panic("internal error: entered unreachable code")
		}
	}

	commandEncoder.EndEncoding()

	return commandBuffer, nil
}

func (r *MetalRenderer) depthTextureMatching(target mtl.Texture) (mtl.Texture, error) {
	if r.depthTexture != 0 && r.depthTexture.Width() == target.Width() && r.depthTexture.Height() == target.Height() {
		return r.depthTexture, nil
	}
	descriptor := mtl.NewTextureDescriptor()
	defer mtl.Release(descriptor)
	descriptor.SetWidth(target.Width())
	descriptor.SetHeight(target.Height())
	descriptor.SetPixelFormat(mtl.PixelFormatDepth32Float)
	descriptor.SetUsage(mtl.TextureUsageRenderTarget)
	if r.isAppleGPU {
		descriptor.SetStorageMode(mtl.StorageModeMemoryless)
	} else {
		descriptor.SetStorageMode(mtl.StorageModePrivate)
	}
	depthTexture := r.device.NewTextureWithDescriptor(descriptor)
	if depthTexture == 0 {
		return 0, errors.New("allocating depth texture")
	}
	mtl.Release(r.depthTexture)
	r.depthTexture = depthTexture
	return depthTexture, nil
}

func drawInstances(commandEncoder mtl.RenderCommandEncoder, r gpui.Range) {
	commandEncoder.DrawPrimitives(mtl.PrimitiveTypeTriangleStrip, 0, 4, uint(r.End-r.Start), uint(r.Start))
}

func rangeIsEmpty(r gpui.Range) bool { return r.Start >= r.End }

func (r *MetalRenderer) drawOpaqueQuads(quadCount int, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder) {
	if quadCount == 0 {
		return
	}
	quadCountValue := uint32(quadCount)

	commandEncoder.SetRenderPipelineState(r.opaqueQuadsPipelineState)
	commandEncoder.SetDepthStencilState(r.depthWriteState)
	commandEncoder.SetVertexBuffer(b.quads.buffer, b.quads.offset, quadInputIndexQuads)
	commandEncoder.SetVertexBytes(unsafe.Pointer(&quadCountValue), uint(unsafe.Sizeof(quadCountValue)), quadInputIndexQuadCount)
	commandEncoder.DrawPrimitives(mtl.PrimitiveTypeTriangleStrip, 0, 4, uint(quadCount), 0)
}

func (r *MetalRenderer) drawNativeViews(nativeViews gpui.Range, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder) {
	if rangeIsEmpty(nativeViews) {
		return
	}
	commandEncoder.SetRenderPipelineState(r.nativeViewsPipelineState)
	commandEncoder.SetVertexBuffer(b.nativeViews.buffer, b.nativeViews.offset, surfaceInputIndexSurfaces)
	drawInstances(commandEncoder, nativeViews)
}

func (r *MetalRenderer) drawPathTiles(tiles gpui.Range, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder) {
	if rangeIsEmpty(tiles) {
		return
	}
	commandEncoder.SetRenderPipelineState(r.pathTilesPipelineState)
	commandEncoder.SetVertexBuffer(b.pathTiles.buffer, b.pathTiles.offset, pathTileInputIndexTiles)
	commandEncoder.SetVertexBuffer(b.pathPaints.buffer, b.pathPaints.offset, pathTileInputIndexPaints)
	commandEncoder.SetFragmentBuffer(b.pathTiles.buffer, b.pathTiles.offset, pathTileInputIndexTiles)
	commandEncoder.SetFragmentBuffer(b.pathCurves.buffer, b.pathCurves.offset, pathTileInputIndexCurves)
	commandEncoder.SetFragmentBuffer(b.pathPaints.buffer, b.pathPaints.offset, pathTileInputIndexPaints)
	drawInstances(commandEncoder, tiles)
}

func (r *MetalRenderer) drawShadows(shadows gpui.Range, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder) {
	if rangeIsEmpty(shadows) {
		return
	}
	commandEncoder.SetRenderPipelineState(r.shadowsPipelineState)
	commandEncoder.SetVertexBuffer(b.shadows.buffer, b.shadows.offset, shadowInputIndexShadows)
	commandEncoder.SetFragmentBuffer(b.shadows.buffer, b.shadows.offset, shadowInputIndexShadows)
	drawInstances(commandEncoder, shadows)
}

func (r *MetalRenderer) drawQuads(quads gpui.Range, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder) {
	if rangeIsEmpty(quads) {
		return
	}
	commandEncoder.SetRenderPipelineState(r.quadsPipelineState)
	commandEncoder.SetVertexBuffer(b.quads.buffer, b.quads.offset, quadInputIndexQuads)
	commandEncoder.SetFragmentBuffer(b.quads.buffer, b.quads.offset, quadInputIndexQuads)
	drawInstances(commandEncoder, quads)
}

func (r *MetalRenderer) drawUnderlines(underlines gpui.Range, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder) {
	if rangeIsEmpty(underlines) {
		return
	}
	commandEncoder.SetRenderPipelineState(r.underlinesPipelineState)
	commandEncoder.SetVertexBuffer(b.underlines.buffer, b.underlines.offset, underlineInputIndexUnderlines)
	commandEncoder.SetFragmentBuffer(b.underlines.buffer, b.underlines.offset, underlineInputIndexUnderlines)
	drawInstances(commandEncoder, underlines)
}

func (r *MetalRenderer) drawMonochromeSprites(textureID gpui.AtlasTextureID, sprites gpui.Range, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder) {
	r.drawSprites(r.monochromeSpritesPipelineState, b.monochromeSprites, textureID, sprites, commandEncoder)
}

func (r *MetalRenderer) drawPolychromeSprites(textureID gpui.AtlasTextureID, sprites gpui.Range, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder) {
	r.drawSprites(r.polychromeSpritesPipelineState, b.polychromeSprites, textureID, sprites, commandEncoder)
}

func (r *MetalRenderer) drawSprites(pipelineState mtl.RenderPipelineState, binding instanceBinding, textureID gpui.AtlasTextureID, sprites gpui.Range, commandEncoder mtl.RenderCommandEncoder) {
	if rangeIsEmpty(sprites) {
		return
	}
	texture, ok := r.spriteAtlas.MetalTexture(textureID)
	if !ok {
		return
	}
	defer mtl.Release(texture)
	textureSize := gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(int32(texture.Width())),
		Height: gpui.DevicePixels(int32(texture.Height())),
	}
	commandEncoder.SetRenderPipelineState(pipelineState)
	commandEncoder.SetVertexBuffer(binding.buffer, binding.offset, spriteInputIndexSprites)
	commandEncoder.SetVertexBytes(unsafe.Pointer(&textureSize), uint(unsafe.Sizeof(textureSize)), spriteInputIndexAtlasTextureSize)
	commandEncoder.SetFragmentBuffer(binding.buffer, binding.offset, spriteInputIndexSprites)
	commandEncoder.SetFragmentTexture(texture, spriteInputIndexAtlasTexture)
	drawInstances(commandEncoder, sprites)
}

func (r *MetalRenderer) drawSurfaces(surfaces []gpui.PaintSurface, firstSurface int, b *instanceBindings, commandEncoder mtl.RenderCommandEncoder, surfaceTextures *[]uintptr) error {
	if len(surfaces) == 0 {
		return nil
	}

	commandEncoder.SetRenderPipelineState(r.surfacesPipelineState)
	commandEncoder.SetVertexBuffer(b.surfaces.buffer, b.surfaces.offset, surfaceInputIndexSurfaces)

	for index, surface := range surfaces {
		imageBuffer := surface.ImageBuffer
		textureSize := gpui.Size[gpui.DevicePixels]{
			Width:  gpui.DevicePixels(int32(cvPixelBufferGetWidth(imageBuffer))),
			Height: gpui.DevicePixels(int32(cvPixelBufferGetHeight(imageBuffer))),
		}

		pixelFormat := cvPixelBufferGetPixelFormatType(imageBuffer)
		if pixelFormat != kCVPixelFormatType_420YpCbCr8BiPlanarFullRange {
			return fmt.Errorf("unsupported surface pixel format: %#x", pixelFormat)
		}

		planeTexture := func(plane uint, format mtl.PixelFormat) (uintptr, error) {
			width := cvPixelBufferGetWidthOfPlane(imageBuffer, plane)
			height := cvPixelBufferGetHeightOfPlane(imageBuffer, plane)
			var texture uintptr
			status := cvMetalTextureCacheCreateTextureFromImage(0, r.coreVideoTextureCache, imageBuffer, 0, uint(format), width, height, plane, &texture)
			if status != kCVReturnSuccess {
				return 0, fmt.Errorf("creating CoreVideo Metal texture for plane %d: status %d", plane, status)
			}
			if texture == 0 {
				return 0, errors.New("retrieving CoreVideo Metal texture")
			}
			return texture, nil
		}
		yTexture, err := planeTexture(0, mtl.PixelFormatR8Unorm)
		if err != nil {
			return err
		}
		cbCrTexture, err := planeTexture(1, mtl.PixelFormatRG8Unorm)
		if err != nil {
			cf.Release(cf.TypeRef(yTexture))
			return err
		}
		*surfaceTextures = append(*surfaceTextures, yTexture, cbCrTexture)
		yMetalTexture := mtl.Texture(cvMetalTextureGetTexture(yTexture))
		if yMetalTexture == 0 {
			return errors.New("retrieving luma Metal texture")
		}
		cbCrMetalTexture := mtl.Texture(cvMetalTextureGetTexture(cbCrTexture))
		if cbCrMetalTexture == 0 {
			return errors.New("retrieving chroma Metal texture")
		}

		commandEncoder.SetVertexBytes(unsafe.Pointer(&textureSize), uint(unsafe.Sizeof(textureSize)), surfaceInputIndexTextureSize)
		commandEncoder.SetFragmentTexture(yMetalTexture, surfaceInputIndexYTexture)
		commandEncoder.SetFragmentTexture(cbCrMetalTexture, surfaceInputIndexCbCrTexture)

		commandEncoder.DrawPrimitives(mtl.PrimitiveTypeTriangleStrip, 0, 4, 1, uint(firstSurface+index))
	}
	return nil
}

func newCommandEncoderForTexture(commandBuffer mtl.CommandBuffer, texture mtl.Texture, depthTexture mtl.Texture, viewportSize gpui.Size[gpui.DevicePixels], clearColor mtl.ClearColor) (mtl.RenderCommandEncoder, error) {
	renderPassDescriptor := mtl.NewRenderPassDescriptor()
	defer mtl.Release(renderPassDescriptor)
	colorAttachment := renderPassDescriptor.ColorAttachment(0)
	colorAttachment.SetTexture(texture)
	colorAttachment.SetStoreAction(mtl.StoreActionStore)
	colorAttachment.SetLoadAction(mtl.LoadActionClear)
	colorAttachment.SetClearColor(clearColor)

	depthAttachment := renderPassDescriptor.DepthAttachment()
	depthAttachment.SetTexture(depthTexture)
	depthAttachment.SetLoadAction(mtl.LoadActionClear)
	depthAttachment.SetClearDepth(0.0)
	depthAttachment.SetStoreAction(mtl.StoreActionDontCare)

	commandEncoder := commandBuffer.RenderCommandEncoderWithDescriptor(renderPassDescriptor)
	if commandEncoder == 0 {
		return 0, errors.New("creating render command encoder")
	}
	commandEncoder = mtl.Retain(commandEncoder)
	commandEncoder.SetViewport(mtl.Viewport{
		OriginX: 0,
		OriginY: 0,
		Width:   float64(viewportSize.Width),
		Height:  float64(viewportSize.Height),
		ZNear:   0,
		ZFar:    1,
	})
	return commandEncoder, nil
}

func readTextureToImage(texture mtl.Texture) (*image.RGBA, error) {
	width := int(texture.Width())
	height := int(texture.Height())
	bytesPerRow := width * 4
	pixels := make([]byte, height*bytesPerRow)

	region := mtl.Region{
		Origin: mtl.Origin{X: 0, Y: 0, Z: 0},
		Size:   mtl.Size{Width: uint(width), Height: uint(height), Depth: 1},
	}
	if len(pixels) > 0 {
		texture.GetBytes(unsafe.Pointer(&pixels[0]), uint(bytesPerRow), region, 0)
	}

	for i := 0; i+4 <= len(pixels); i += 4 {
		pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
	}

	if width <= 0 || height <= 0 {
		return nil, errors.New("creating RGBA image from pixel data")
	}
	return &image.RGBA{Pix: pixels, Stride: bytesPerRow, Rect: image.Rect(0, 0, width, height)}, nil
}

func buildPipelineState(device mtl.Device, library mtl.Library, label, vertexFnName, fragmentFnName string, pixelFormat mtl.PixelFormat, blending bool) (mtl.RenderPipelineState, error) {
	vertexFn, ok := library.NewFunctionWithName(vertexFnName)
	if !ok {
		return 0, fmt.Errorf("locating vertex function %s", vertexFnName)
	}
	defer mtl.Release(vertexFn)
	fragmentFn, ok := library.NewFunctionWithName(fragmentFnName)
	if !ok {
		return 0, fmt.Errorf("locating fragment function %s", fragmentFnName)
	}
	defer mtl.Release(fragmentFn)

	descriptor := mtl.NewRenderPipelineDescriptor()
	defer mtl.Release(descriptor)
	descriptor.SetLabel(label)
	descriptor.SetVertexFunction(vertexFn)
	descriptor.SetFragmentFunction(fragmentFn)
	colorAttachment := descriptor.ColorAttachment(0)
	colorAttachment.SetPixelFormat(pixelFormat)
	colorAttachment.SetBlendingEnabled(blending)
	colorAttachment.SetRgbBlendOperation(mtl.BlendOperationAdd)
	colorAttachment.SetAlphaBlendOperation(mtl.BlendOperationAdd)
	colorAttachment.SetSourceRGBBlendFactor(mtl.BlendFactorSourceAlpha)
	colorAttachment.SetSourceAlphaBlendFactor(mtl.BlendFactorOne)
	colorAttachment.SetDestinationRGBBlendFactor(mtl.BlendFactorOneMinusSourceAlpha)
	colorAttachment.SetDestinationAlphaBlendFactor(mtl.BlendFactorOneMinusSourceAlpha)
	descriptor.SetDepthAttachmentPixelFormat(mtl.PixelFormatDepth32Float)

	state, err := device.NewRenderPipelineStateWithDescriptor(descriptor)
	if err != nil {
		return 0, fmt.Errorf("creating %s render pipeline: %w", label, err)
	}
	return state, nil
}

func buildDepthStencilState(device mtl.Device, writeDepth bool) (mtl.DepthStencilState, error) {
	descriptor := mtl.NewDepthStencilDescriptor()
	defer mtl.Release(descriptor)
	descriptor.SetDepthCompareFunction(mtl.CompareFunctionGreater)
	descriptor.SetDepthWriteEnabled(writeDepth)
	state := device.NewDepthStencilStateWithDescriptor(descriptor)
	if state == 0 {
		return 0, errors.New("creating depth stencil state")
	}
	return state, nil
}

type instanceBinding struct {
	buffer mtl.Buffer
	offset uint
}

type instanceBindings struct {
	quads             instanceBinding
	pathTiles         instanceBinding
	pathCurves        instanceBinding
	pathPaints        instanceBinding
	shadows           instanceBinding
	underlines        instanceBinding
	monochromeSprites instanceBinding
	polychromeSprites instanceBinding
	surfaces          instanceBinding
	nativeViews       instanceBinding
}

func writeInstances(scene *gpui.Scene, writer *instanceBufferWriter) (instanceBindings, error) {
	pathPaints := make([]gpui.PathPaint, 0, len(scene.Paths))
	var pathCurves []gpui.PathCurve
	var pathTiles []gpui.PathTile
	for _, path := range scene.Paths {
		path.AppendTo(&pathPaints, &pathCurves, &pathTiles)
	}
	surfaces := make([]SurfaceBounds, len(scene.Surfaces))
	for i, surface := range scene.Surfaces {
		surfaces[i] = SurfaceBounds{Bounds: surface.Bounds, ContentMask: surface.ContentMask}
	}
	nativeViews := make([]SurfaceBounds, len(scene.NativeViews))
	for i, nativeView := range scene.NativeViews {
		nativeViews[i] = SurfaceBounds{Bounds: nativeView.Bounds, ContentMask: nativeView.ContentMask}
	}
	var b instanceBindings
	var err error
	if b.quads, err = writeInstanceSlice(writer, scene.Quads); err != nil {
		return b, err
	}
	if b.pathTiles, err = writeInstanceSlice(writer, pathTiles); err != nil {
		return b, err
	}
	if b.pathCurves, err = writeInstanceSlice(writer, pathCurves); err != nil {
		return b, err
	}
	if b.pathPaints, err = writeInstanceSlice(writer, pathPaints); err != nil {
		return b, err
	}
	if b.shadows, err = writeInstanceSlice(writer, scene.Shadows); err != nil {
		return b, err
	}
	if b.underlines, err = writeInstanceSlice(writer, scene.Underlines); err != nil {
		return b, err
	}
	if b.monochromeSprites, err = writeInstanceSlice(writer, scene.MonochromeSprites); err != nil {
		return b, err
	}
	if b.polychromeSprites, err = writeInstanceSlice(writer, scene.PolychromeSprites); err != nil {
		return b, err
	}
	if b.surfaces, err = writeInstanceSlice(writer, surfaces); err != nil {
		return b, err
	}
	if b.nativeViews, err = writeInstanceSlice(writer, nativeViews); err != nil {
		return b, err
	}
	return b, nil
}

type filledInstanceBuffer struct {
	buffer  instanceBuffer
	written int
}

type instanceBufferWriter struct {
	device        mtl.Device
	pool          *InstanceBufferPool
	unifiedMemory bool
	filled        []filledInstanceBuffer
	current       instanceBuffer
	offset        int
}

func newInstanceBufferWriter(device mtl.Device, pool *InstanceBufferPool, unifiedMemory bool) (*instanceBufferWriter, error) {
	pool.mu.Lock()
	current, err := pool.acquire(device, unifiedMemory)
	pool.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &instanceBufferWriter{
		device:        device,
		pool:          pool,
		unifiedMemory: unifiedMemory,
		current:       current,
	}, nil
}

func nextMultipleOf(value, multiple int) int {
	if r := value % multiple; r != 0 {
		return value + multiple - r
	}
	return value
}

func nextPowerOfTwo(value int) int {
	if value <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(value-1))
}

func (w *instanceBufferWriter) allocate(size int) (instanceBinding, unsafe.Pointer, error) {
	offset := nextMultipleOf(w.offset, instanceBufferAlignment)
	if offset+size > w.current.size {
		if err := w.grow(size); err != nil {
			return instanceBinding{}, nil, err
		}
		offset = 0
	}
	w.offset = offset + size

	binding := instanceBinding{buffer: w.current.metalBuffer, offset: uint(offset)}
	return binding, unsafe.Add(w.current.metalBuffer.Contents(), offset), nil
}

func writeInstanceSlice[T any](w *instanceBufferWriter, values []T) (instanceBinding, error) {
	var zero T
	binding, destination, err := w.allocate(int(unsafe.Sizeof(zero)) * len(values))
	if err != nil {
		return instanceBinding{}, err
	}
	if len(values) > 0 {
		copy(unsafe.Slice((*T)(destination), len(values)), values)
	}
	return binding, nil
}

func (w *instanceBufferWriter) grow(required int) error {
	w.pool.mu.Lock()
	defer w.pool.mu.Unlock()
	bufferSize := min(max(w.pool.bufferSize*2, nextPowerOfTwo(required)), maxInstanceBufferSize)
	if bufferSize < required {
		return fmt.Errorf("instance buffer needs %d bytes, above the maximum of %d", required, maxInstanceBufferSize)
	}
	if bufferSize <= w.current.size {
		return fmt.Errorf("frame instance data exceeds the %d-byte maximum", maxInstanceBufferSize)
	}
	if bufferSize != w.pool.bufferSize {
		slog.Info(fmt.Sprintf("increased instance buffer size to %d", bufferSize))
		w.pool.reset(bufferSize)
	}
	buffer, err := w.pool.acquire(w.device, w.unifiedMemory)
	if err != nil {
		return err
	}
	w.filled = append(w.filled, filledInstanceBuffer{buffer: w.current, written: w.offset})
	w.current = buffer
	w.offset = 0
	return nil
}

func (w *instanceBufferWriter) finish() instanceBuffer {
	if !w.unifiedMemory {
		for _, f := range w.filled {
			if f.written == 0 {
				continue
			}
			f.buffer.metalBuffer.DidModifyRange(objc.NSRange{Location: 0, Length: objc.NSUInteger(f.written)})
		}
		if w.offset > 0 {
			w.current.metalBuffer.DidModifyRange(objc.NSRange{Location: 0, Length: objc.NSUInteger(w.offset)})
		}
	}
	for _, f := range w.filled {
		mtl.Release(f.buffer.metalBuffer)
	}
	w.filled = nil
	return w.current
}

func (w *instanceBufferWriter) discard() {
	for _, f := range w.filled {
		mtl.Release(f.buffer.metalBuffer)
	}
	w.filled = nil
	mtl.Release(w.current.metalBuffer)
	w.current = instanceBuffer{}
}

const (
	globalInputIndexViewportSize = 0
	globalInputIndexDepthScale   = 1
)

const shadowInputIndexShadows = 2

const (
	quadInputIndexQuads     = 2
	quadInputIndexQuadCount = 3
)

const underlineInputIndexUnderlines = 2

const (
	spriteInputIndexSprites          = 2
	spriteInputIndexAtlasTextureSize = 3
	spriteInputIndexAtlasTexture     = 4
)

const (
	surfaceInputIndexSurfaces    = 2
	surfaceInputIndexTextureSize = 3
	surfaceInputIndexYTexture    = 4
	surfaceInputIndexCbCrTexture = 5
)

const (
	pathTileInputIndexTiles  = 2
	pathTileInputIndexCurves = 3
	pathTileInputIndexPaints = 4
)

// SurfaceBounds is the per-instance data of the surface and native view shaders.
type SurfaceBounds struct {
	// Bounds are the surface's bounds.
	Bounds gpui.Bounds[gpui.ScaledPixels]
	// ContentMask clips the surface.
	ContentMask gpui.ContentMask[gpui.ScaledPixels]
}

// MetalHeadlessRenderer renders scenes offscreen for tests and benchmarks.
type MetalHeadlessRenderer struct {
	renderer *MetalRenderer
}

var _ gpui.PlatformHeadlessRenderer = (*MetalHeadlessRenderer)(nil)

// NewMetalHeadlessRenderer returns a headless renderer with its own instance buffer pool.
func NewMetalHeadlessRenderer() (*MetalHeadlessRenderer, error) {
	renderer, err := NewMetalRendererHeadless(NewInstanceBufferPool())
	if err != nil {
		return nil, err
	}
	return &MetalHeadlessRenderer{renderer: renderer}, nil
}

// RenderSceneToImage renders the scene into an image of size.
func (h *MetalHeadlessRenderer) RenderSceneToImage(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) (*image.RGBA, error) {
	return h.renderer.RenderSceneToImage(scene, size)
}

// RenderScene renders the scene without reading it back.
func (h *MetalHeadlessRenderer) RenderScene(scene *gpui.Scene, size gpui.Size[gpui.DevicePixels]) error {
	return h.renderer.RenderScene(scene, size)
}

// SpriteAtlas returns the renderer's sprite atlas.
func (h *MetalHeadlessRenderer) SpriteAtlas() gpui.PlatformAtlas {
	return h.renderer.SpriteAtlas()
}

// Destroy releases the renderer, as dropping the Rust headless renderer does.
func (h *MetalHeadlessRenderer) Destroy() {
	h.renderer.Destroy()
}

const coreVideoPath = "/System/Library/Frameworks/CoreVideo.framework/CoreVideo"

const coreGraphicsPath = "/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics"

const kCVReturnSuccess int32 = 0

const kCVPixelFormatType_420YpCbCr8BiPlanarFullRange uint32 = 0x34323066

var (
	coreVideoOnce sync.Once
	coreVideoErr  error
	coreVideoLib  uintptr

	cvMetalTextureCacheCreate                 func(allocator, cacheAttributes uintptr, metalDevice objc.ID, textureAttributes uintptr, cacheOut *uintptr) int32
	cvMetalTextureCacheCreateTextureFromImage func(allocator, textureCache, sourceImage, textureAttributes uintptr, pixelFormat uint, width, height, planeIndex uint, textureOut *uintptr) int32
	cvMetalTextureGetTexture                  func(image uintptr) objc.ID
	cvPixelBufferGetWidth                     func(pixelBuffer uintptr) uint
	cvPixelBufferGetHeight                    func(pixelBuffer uintptr) uint
	cvPixelBufferGetPixelFormatType           func(pixelBuffer uintptr) uint32
	cvPixelBufferGetWidthOfPlane              func(pixelBuffer uintptr, planeIndex uint) uint
	cvPixelBufferGetHeightOfPlane             func(pixelBuffer uintptr, planeIndex uint) uint
	cgColorSpaceCreateWithName                func(name uintptr) uintptr
	kCGColorSpaceDisplayP3                    uintptr
)

func loadCoreVideo() error {
	coreVideoOnce.Do(func() {
		if err := cf.Load(); err != nil {
			coreVideoErr = err
			return
		}
		lib, err := purego.Dlopen(coreVideoPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			coreVideoErr = fmt.Errorf("open %s: %w", coreVideoPath, err)
			return
		}
		coreVideoLib = lib
		cg, err := purego.Dlopen(coreGraphicsPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			coreVideoErr = fmt.Errorf("open %s: %w", coreGraphicsPath, err)
			return
		}
		purego.RegisterLibFunc(&cvMetalTextureCacheCreate, lib, "CVMetalTextureCacheCreate")
		purego.RegisterLibFunc(&cvMetalTextureCacheCreateTextureFromImage, lib, "CVMetalTextureCacheCreateTextureFromImage")
		purego.RegisterLibFunc(&cvMetalTextureGetTexture, lib, "CVMetalTextureGetTexture")
		purego.RegisterLibFunc(&cvPixelBufferGetWidth, lib, "CVPixelBufferGetWidth")
		purego.RegisterLibFunc(&cvPixelBufferGetHeight, lib, "CVPixelBufferGetHeight")
		purego.RegisterLibFunc(&cvPixelBufferGetPixelFormatType, lib, "CVPixelBufferGetPixelFormatType")
		purego.RegisterLibFunc(&cvPixelBufferGetWidthOfPlane, lib, "CVPixelBufferGetWidthOfPlane")
		purego.RegisterLibFunc(&cvPixelBufferGetHeightOfPlane, lib, "CVPixelBufferGetHeightOfPlane")
		purego.RegisterLibFunc(&cgColorSpaceCreateWithName, cg, "CGColorSpaceCreateWithName")
		if name, err := cf.Constant(cg, "kCGColorSpaceDisplayP3"); err == nil {
			kCGColorSpaceDisplayP3 = uintptr(name)
		}
	})
	return coreVideoErr
}

func displayP3ColorSpace() (uintptr, bool) {
	if loadCoreVideo() != nil || kCGColorSpaceDisplayP3 == 0 {
		return 0, false
	}
	colorSpace := cgColorSpaceCreateWithName(kCGColorSpaceDisplayP3)
	return colorSpace, colorSpace != 0
}
